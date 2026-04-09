package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/ruffini/prism/api"
	"github.com/ruffini/prism/db"
	"github.com/ruffini/prism/graph"
	"github.com/ruffini/prism/internal/server"
	"github.com/ruffini/prism/mcp"
	"github.com/ruffini/prism/parser"
	"github.com/ruffini/prism/vector"
	"github.com/ruffini/prism/watcher"
)

var manifestFiles = []string{"go.mod", "package.json", "Cargo.toml", "pyproject.toml", "pom.xml"}

func isProjectRoot(dir string) bool {
	for _, m := range manifestFiles {
		if _, err := os.Stat(filepath.Join(dir, m)); err == nil {
			return true
		}
	}
	return false
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "start":
		startCmd := flag.NewFlagSet("start", flag.ExitOnError)
		repoPath := startCmd.String("repo", ".", "Repository path to index")
		port := startCmd.Int("port", 8080, "HTTP server port")
		mcpOnly := startCmd.Bool("mcp-only", false, "Only start MCP server (no web UI)")
		autoIndex := startCmd.Bool("auto-index", true, "Automatically index if database doesn't exist")
		ollamaURL := startCmd.String("ollama", "http://localhost:11434", "Ollama API URL")
		embedModel := startCmd.String("model", "nomic-embed-text", "Embedding model name")
		startCmd.Parse(os.Args[2:])

		startUnified(*repoPath, *port, *mcpOnly, *autoIndex, *ollamaURL, *embedModel)

	case "parse":
		parseCmd := flag.NewFlagSet("parse", flag.ExitOnError)
		jsonOutput := parseCmd.Bool("json", false, "Output as JSON")
		parseCmd.Parse(os.Args[2:])

		if parseCmd.NArg() < 1 {
			fmt.Println("Usage: prism parse <file>")
			os.Exit(1)
		}

		parseFile(parseCmd.Arg(0), *jsonOutput)

	case "index":
		indexCmd := flag.NewFlagSet("index", flag.ExitOnError)
		repoPath := indexCmd.String("repo", ".", "Repository path")
		indexCmd.Parse(os.Args[2:])

		indexRepository(*repoPath)

	case "serve":
		serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
		dbPath := serveCmd.String("db", "code_graph.db", "Database path")
		vectorPath := serveCmd.String("vectors", "vectors.bin", "Vector store path")
		ollamaURL := serveCmd.String("ollama", "http://localhost:11434", "Ollama API URL")
		embedModel := serveCmd.String("model", "nomic-embed-text", "Embedding model name")
		repoFlag := serveCmd.String("repo", ".", "Repository path to watch (used with --watch)")
		watchFlag := serveCmd.Bool("watch", false, "Auto-reindex files on change")
		serveCmd.Parse(os.Args[2:])

		startMCPServer(*dbPath, *vectorPath, *ollamaURL, *embedModel, *repoFlag, *watchFlag)

	case "export":
		exportCmd := flag.NewFlagSet("export", flag.ExitOnError)
		dbPath := exportCmd.String("db", "code_graph.db", "Database path")
		output := exportCmd.String("output", "CLAUDE.md", "Output file path")
		exportCmd.Parse(os.Args[2:])
		exportClaudeMD(*dbPath, *output)

	case "help":
		printUsage()

	case "search":
		searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
		dbPath := searchCmd.String("db", "code_graph.db", "Database path")
		topK := searchCmd.Int("top", 5, "Number of results")
		searchCmd.Parse(os.Args[2:])

		if searchCmd.NArg() < 1 {
			fmt.Println("Usage: prism search <query>")
			os.Exit(1)
		}

		query := searchCmd.Arg(0)
		searchCode(query, *dbPath, *topK)

	case "embed":
		embedCmd := flag.NewFlagSet("embed", flag.ExitOnError)
		dbPath := embedCmd.String("db", "code_graph.db", "Database path")
		ollamaURL := embedCmd.String("ollama", "http://localhost:11434", "Ollama API URL")
		model := embedCmd.String("model", "nomic-embed-text", "Embedding model")
		embedCmd.Parse(os.Args[2:])

		generateEmbeddings(*dbPath, *ollamaURL, *model)

	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`PRISM - Program Representation & Intelligence Semantic Mapper

Usage:
  prism <command> [options]

Commands:
  start                Start unified server (auto-index, embed, serve all-in-one)
  parse <file>         Parse a single file and show extracted elements
  index                Index the current directory (or use -repo <path> for specific directory)
  embed                Generate embeddings for indexed code
  search <query>       Semantic search for code elements
  serve                Start MCP server for Claude Code integration
  help                 Show this help

Options for 'start' (RECOMMENDED):
  -repo <path>        Repository path to index (default: current directory)
  -port <number>      HTTP server port (default: 8080)
  -mcp-only           Only start MCP server via stdio (no web UI)
  -auto-index         Automatically index if needed (default: true)
  -ollama <url>       Ollama API URL (default: http://localhost:11434)
  -model <name>       Embedding model (default: nomic-embed-text)

Options for 'parse':
  -json               Output as JSON

Options for 'index':
  -repo <path>        Repository path (default: current directory)

Options for 'embed':
  -db <path>          Database path (default: code_graph.db)
  -ollama <url>       Ollama API URL (default: http://localhost:11434)
  -model <name>       Embedding model (default: nomic-embed-text)

Options for 'search':
  -db <path>          Database path (default: code_graph.db)
  -top <n>            Number of results (default: 5)

Options for 'serve':
  -db <path>          Path to code_graph.db (default: code_graph.db)
  -ollama <url>       Ollama API URL (default: http://localhost:11434)
  -model <name>       Embedding model (default: nomic-embed-text)

  export               Generate CLAUDE.md from graph annotations

Options for 'export':
  -db <path>          Database path (default: code_graph.db)
  -output <path>      Output file (default: CLAUDE.md)

Examples:
  # Quick start - everything automated!
  prism start

  # Start with custom repo and port
  prism start -repo ./my-project -port 9000

  # For Claude Code MCP integration (stdio only)
  prism start -mcp-only -auto-index

  # Manual workflow (advanced users)
  prism parse main.py
  prism index -repo ./my-project
  prism embed -db code_graph.db
  prism serve -db code_graph.db`)
}

func parseFile(filePath string, jsonOutput bool) {
	p := parser.GetParser(filePath)
	if p == nil {
		fmt.Printf("❌ Unsupported file type: %s\n", filePath)
		os.Exit(1)
	}

	lang := p.Language()
	fmt.Printf("🔍 Parsing: %s (%s)\n", filePath, lang)

	parsed, err := p.ParseFile(filePath)
	if err != nil {
		fmt.Printf("❌ Parse error: %v\n", err)
		os.Exit(1)
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(parsed, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Output human-readable
	printParseResult(parsed)
}

func printParseResult(result interface{}) {
	// Type assertion basada en el tipo
	switch r := result.(type) {
	case *parser.ParsedFile:
		// Este es el import del parser package
	default:
		fmt.Printf("Unknown type: %T\n", r)
	}

	// Por ahora, imprimimos como JSON indentado
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}

func indexRepository(repoPath string) {
	fmt.Printf("🔍 Indexing: %s\n", repoPath)

	// Initialize SQLite database
	dbPath := "code_graph.db"
	database, err := db.InitDB(dbPath)
	if err != nil {
		fmt.Printf("❌ Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// Create the code graph
	codeGraph := graph.NewGraph(database)

	parsedFiles := make(map[string]*parser.ParsedFile)
	totalFiles := 0

	err = filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-code files
		if info.IsDir() {
			if parser.ShouldSkipPath(path) {
				return filepath.SkipDir
			}
			return nil
		}

		if !parser.IsCodeFile(path) {
			return nil
		}

		p := parser.GetParser(path)
		if p == nil {
			return nil
		}

		relPath, _ := filepath.Rel(repoPath, path)
		relPath = filepath.ToSlash(relPath)

		parsed, err := p.ParseFile(path)
		if err != nil {
			fmt.Printf("  ⚠ Error parsing %s: %v\n", relPath, err)
			return nil
		}
		// Store relative path in the parsed file
		parsed.Path = relPath
		for i := range parsed.Elements {
			parsed.Elements[i].File = relPath
			parsed.Elements[i].ID = relPath + ":" + parsed.Elements[i].Name
		}
		parsedFiles[relPath] = parsed
		totalFiles++
		fmt.Printf("  ✓ %s (%d elements)\n", relPath, len(parsed.Elements))

		return nil
	})

	if err != nil {
		fmt.Printf("❌ Walk error: %v\n", err)
		os.Exit(1)
	}

	// Build the graph from parsed files
	if err := codeGraph.BuildFromParsed(parsedFiles); err != nil {
		fmt.Printf("❌ Failed to build graph: %v\n", err)
		os.Exit(1)
	}

	// Resolve call edges to actual function definitions
	if err := codeGraph.ResolveCallEdges(); err != nil {
		fmt.Printf("⚠ Warning: Failed to resolve call edges: %v\n", err)
	}

	// Get stats
	nodes, edges, err := codeGraph.Stats()
	if err != nil {
		fmt.Printf("❌ Failed to get stats: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ Indexed %d files\n", totalFiles)
	fmt.Printf("   Nodes: %d\n", nodes)
	fmt.Printf("   Edges: %d\n", edges)
	fmt.Printf("   Database: %s\n", dbPath)
}

func searchCode(query, dbPath string, topK int) {
	fmt.Printf("🔍 Searching: %s\n", query)

	// Open database
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Printf("❌ Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// Initialize vector store
	store := vector.NewVectorStore(database)

	// Check if we have any embeddings
	count, err := store.Count()
	if err != nil {
		fmt.Printf("❌ Failed to count embeddings: %v\n", err)
		os.Exit(1)
	}

	if count == 0 {
		fmt.Println("⚠ No embeddings found. Run 'prism embed' first to generate embeddings.")
		os.Exit(1)
	}

	fmt.Printf("   Found %d embeddings in database\n", count)

	// Create embedder
	embedder := vector.NewEmbedder()

	// Search
	results, err := store.SearchText(embedder, query, topK)
	if err != nil {
		fmt.Printf("❌ Search failed: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println("   No results found.")
		return
	}

	fmt.Printf("\n📊 Top %d results:\n", len(results))
	for i, r := range results {
		// Get node info from database
		var name, nodeType, file string
		var line int
		err := database.QueryRow(
			`SELECT name, type, file, line FROM nodes WHERE id = ?`,
			r.NodeID,
		).Scan(&name, &nodeType, &file, &line)
		if err != nil {
			fmt.Printf("  %d. %s (similarity: %.4f)\n", i+1, r.NodeID, r.Similarity)
		} else {
			fmt.Printf("  %d. [%s] %s (%s:%d) - similarity: %.4f\n",
				i+1, nodeType, name, file, line, r.Similarity)
		}
	}
}

func indexSingleFile(g *graph.CodeGraph, filePath string) error {
	p := parser.GetParser(filePath)
	if p == nil {
		return nil
	}
	parsed, err := p.ParseFile(filePath)
	if err != nil {
		return err
	}
	return g.BuildFromParsed(map[string]*parser.ParsedFile{filePath: parsed})
}

func startMCPServer(dbPath, vectorPath, ollamaURL, embedModel, repoPath string, watch bool) {
	// Open database
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// Create graph instance
	codeGraph := graph.NewGraph(database)

	// Create vector store
	vectorStore := vector.NewVectorStore(database)

	// Create embedder
	embedder := vector.NewEmbedderWithConfig(ollamaURL, embedModel)

	// Start HTTP API server in a goroutine
	apiServer := api.NewAPIServer(codeGraph, vectorStore, database)
	mux := http.NewServeMux()
	apiServer.RegisterRoutes(mux)

	go func() {
		fmt.Printf("🌐 HTTP API server listening on http://localhost:8080\n")
		if err := http.ListenAndServe(":8080", mux); err != nil {
			fmt.Fprintf(os.Stderr, "❌ HTTP server error: %v\n", err)
		}
	}()

	// Create MCP server
	mcpServer := mcp.NewMCPServer(codeGraph, vectorStore, embedder)

	// Try to start MCP server, but don't fail if stdin is closed
	go func() {
		fmt.Println("📡 MCP server ready (stdio transport)")
		if err := mcpServer.Start(); err != nil {
			// Ignore EOF errors from stdin
			fmt.Fprintf(os.Stderr, "⚠️  MCP connection closed\n")
		}
	}()

	// Start file watcher if requested
	if watch {
		done := make(chan struct{})
		go func() {
			indexFn := func(path string) error {
				return indexSingleFile(codeGraph, path)
			}
			removeFn := func(path string) error {
				return codeGraph.RemoveFileNodes(path)
			}
			if err := watcher.Watch(repoPath, indexFn, removeFn, done); err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  File watcher failed: %v\n", err)
			}
		}()
	}

	// Keep the server running forever
	select {}
}

func generateEmbeddings(dbPath, ollamaURL, model string) {
	fmt.Println("🧠 Generating embeddings...")
	fmt.Printf("   Database: %s\n", dbPath)
	fmt.Printf("   Ollama: %s\n", ollamaURL)
	fmt.Printf("   Model: %s\n", model)

	// Open database
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Printf("❌ Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// Initialize vector store (creates embeddings table if needed)
	store := vector.NewVectorStore(database)
	if err := store.InitSchema(); err != nil {
		fmt.Printf("❌ Failed to initialize vector store: %v\n", err)
		os.Exit(1)
	}

	// Create embedder
	embedder := vector.NewEmbedderWithConfig(ollamaURL, model)

	// Get all nodes from database
	rows, err := database.Query(`
		SELECT id, name, type, file, line, end_line, signature, docstring, body
		FROM nodes
	`)
	if err != nil {
		fmt.Printf("❌ Failed to query nodes: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	var nodes []struct {
		ID        string
		Name      string
		Type      string
		File      string
		Line      int
		EndLine   int
		Signature string
		Docstring string
		Body      string
	}

	for rows.Next() {
		var n struct {
			ID        string
			Name      string
			Type      string
			File      string
			Line      int
			EndLine   int
			Signature string
			Docstring string
			Body      string
		}
		var sig, doc, body sql.NullString
		if err := rows.Scan(&n.ID, &n.Name, &n.Type, &n.File, &n.Line, &n.EndLine, &sig, &doc, &body); err != nil {
			continue
		}
		n.Signature = sig.String
		n.Docstring = doc.String
		n.Body = body.String
		nodes = append(nodes, n)
	}

	fmt.Printf("   Found %d nodes to embed\n", len(nodes))

	// Check which nodes already have embeddings
	existing := make(map[string]bool)
	existRows, err := database.Query(`SELECT node_id FROM embeddings`)
	if err == nil {
		defer existRows.Close()
		for existRows.Next() {
			var nodeID string
			existRows.Scan(&nodeID)
			existing[nodeID] = true
		}
	}

	// Generate embeddings for each node
	embedded := 0
	skipped := 0
	failed := 0

	for i, n := range nodes {
		// Skip if already embedded
		if existing[n.ID] {
			skipped++
			continue
		}

		// Create text representation for embedding
		text := vector.BuildEmbedText(n.Name, n.Type, n.Signature, n.Docstring, n.Body)

		// Generate embedding
		embedding, err := embedder.Embed(text)
		if err != nil {
			fmt.Printf("  ⚠ Failed to embed %s: %v\n", n.Name, err)
			failed++
			continue
		}

		// Store embedding
		if err := store.Store(n.ID, embedding); err != nil {
			fmt.Printf("  ⚠ Failed to store embedding for %s: %v\n", n.Name, err)
			failed++
			continue
		}

		embedded++
		if (i+1)%50 == 0 || i == len(nodes)-1 {
			fmt.Printf("   Progress: %d/%d (embedded: %d, skipped: %d)\n", i+1, len(nodes), embedded, skipped)
		}
	}

	fmt.Printf("\n✅ Embedding complete!\n")
	fmt.Printf("   Embedded: %d\n", embedded)
	fmt.Printf("   Skipped (existing): %d\n", skipped)
	if failed > 0 {
		fmt.Printf("   Failed: %d\n", failed)
	}
}

func exportClaudeMD(dbPath, outputPath string) {
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Printf("❌ Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	var sb strings.Builder
	sb.WriteString("# CLAUDE.md — Generado por PRISM\n\n")
	sb.WriteString("> Este archivo fue generado automáticamente desde las anotaciones del grafo de código.\n\n")

	// Nodes with comments
	rows, err := database.Query(`
		SELECT n.name, n.file, n.line, nc.comment
		FROM nodes n
		JOIN node_comments nc ON nc.node_id = n.id
		WHERE nc.comment IS NOT NULL AND nc.comment != ''
		ORDER BY n.file, n.line
	`)
	if err == nil {
		defer rows.Close()
		sb.WriteString("## Nodos Anotados\n\n")
		for rows.Next() {
			var name, file, comment string
			var line int
			rows.Scan(&name, &file, &line, &comment)
			sb.WriteString(fmt.Sprintf("- **%s** (`%s:%d`) — %s\n", name, file, line, comment))
		}
		sb.WriteString("\n")
	}

	// Nodes with tags
	rows2, err2 := database.Query(`
		SELECT n.name, n.file, n.line, GROUP_CONCAT(nt.tag, ', ') as tags
		FROM nodes n
		JOIN node_tags nt ON nt.node_id = n.id
		GROUP BY n.id, n.name, n.file, n.line
		ORDER BY n.file, n.line
	`)
	if err2 == nil {
		defer rows2.Close()
		sb.WriteString("## Nodos con Etiquetas\n\n")
		for rows2.Next() {
			var name, file, tags string
			var line int
			rows2.Scan(&name, &file, &line, &tags)
			sb.WriteString(fmt.Sprintf("- **%s** (`%s:%d`) [%s]\n", name, file, line, tags))
		}
		sb.WriteString("\n")
	}

	// Nodes with metadata
	rows3, err3 := database.Query(`
		SELECT n.name, n.file, n.line, nm.key, nm.value
		FROM nodes n
		JOIN node_metadata nm ON nm.node_id = n.id
		ORDER BY n.file, n.line, nm.key
	`)
	if err3 == nil {
		defer rows3.Close()
		sb.WriteString("## Metadata de Nodos\n\n")
		for rows3.Next() {
			var name, file, key, value string
			var line int
			rows3.Scan(&name, &file, &line, &key, &value)
			sb.WriteString(fmt.Sprintf("- **%s** (`%s:%d`) %s: %s\n", name, file, line, key, value))
		}
		sb.WriteString("\n")
	}

	// Context profiles
	rows4, err4 := database.Query(`SELECT name, description FROM profiles ORDER BY name`)
	if err4 == nil {
		defer rows4.Close()
		sb.WriteString("## Perfiles de Contexto PRISM\n\n")
		sb.WriteString("Usa `use_profile` en Claude Code para cargar el contexto de un área:\n\n")
		for rows4.Next() {
			var name, desc string
			rows4.Scan(&name, &desc)
			sb.WriteString(fmt.Sprintf("- `%s`", name))
			if desc != "" {
				sb.WriteString(fmt.Sprintf(" — %s", desc))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if err := os.WriteFile(outputPath, []byte(sb.String()), 0644); err != nil {
		fmt.Printf("❌ Failed to write %s: %v\n", outputPath, err)
		os.Exit(1)
	}
	fmt.Printf("✅ Exported to %s\n", outputPath)
}

func startUnified(repoPath string, port int, mcpOnly, autoIndex bool, ollamaURL, embedModel string) {
	fmt.Fprintln(os.Stderr, "🔮 PRISM - Starting Unified Server")
	
	// Determine database path (use .prism folder in repo)
	prismDir := filepath.Join(repoPath, ".prism")
	if err := os.MkdirAll(prismDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create .prism directory: %v\n", err)
		os.Exit(1)
	}
	
	dbPath := filepath.Join(prismDir, "code_graph.db")
	dbExists := fileExists(dbPath)
	
	// Auto-index if database doesn't exist and auto-index is enabled
	if !dbExists && autoIndex {
		fmt.Fprintf(os.Stderr, "📦 First time setup - indexing repository...\n")
		fmt.Fprintf(os.Stderr, "   Repository: %s\n", repoPath)
		
		// Initialize database
		database, err := db.InitDB(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to initialize database: %v\n", err)
			os.Exit(1)
		}
		
		// Index the repository
		codeGraph := graph.NewGraph(database)
		if err := indexRepositoryToGraph(repoPath, codeGraph); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to index repository: %v\n", err)
			database.Close()
			os.Exit(1)
		}
		
		// Generate embeddings in background
		fmt.Fprintf(os.Stderr, "🧠 Generating embeddings in background...\n")
		go generateEmbeddingsForDB(database, ollamaURL, embedModel)
		
		database.Close()
		fmt.Fprintf(os.Stderr, "✅ Repository indexed successfully\n")
	} else if !dbExists {
		fmt.Fprintf(os.Stderr, "⚠️  No index found. Run with --auto-index or run 'prism index' first.\n")
		os.Exit(1)
	} else {
		fmt.Fprintf(os.Stderr, "✅ Using existing index: %s\n", dbPath)
	}
	
	// Open database
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()
	
	// Start unified server
	srv := server.NewUnifiedServer(database, port, ollamaURL, embedModel, mcpOnly || !isTerminal())
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Server failed: %v\n", err)
		os.Exit(1)
	}
}

func indexRepositoryToGraph(repoPath string, codeGraph *graph.CodeGraph) error {
	parsedFiles := make(map[string]*parser.ParsedFile)
	totalFiles := 0

	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if parser.ShouldSkipPath(path) {
				return filepath.SkipDir
			}
			return nil
		}

		if !parser.IsCodeFile(path) {
			return nil
		}

		p := parser.GetParser(path)
		if p == nil {
			return nil
		}

		// Find nearest project root at or below repoPath for this file
		base := repoPath
		dir := filepath.Dir(path)
		for dir != repoPath && dir != filepath.Dir(dir) {
			if isProjectRoot(dir) {
				base = dir
				break
			}
			dir = filepath.Dir(dir)
		}

		relPath, _ := filepath.Rel(base, path)
		relPath = filepath.ToSlash(relPath)

		parsed, err := p.ParseFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ⚠ Error parsing %s: %v\n", relPath, err)
			return nil
		}

		parsed.Path = relPath
		for i := range parsed.Elements {
			parsed.Elements[i].File = relPath
			parsed.Elements[i].ID = relPath + ":" + parsed.Elements[i].Name
		}
		parsedFiles[relPath] = parsed
		totalFiles++
		
		if totalFiles%10 == 0 {
			fmt.Fprintf(os.Stderr, "  📄 Indexed %d files...\n", totalFiles)
		}

		return nil
	})

	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "  ✓ Parsed %d files\n", totalFiles)

	// Build graph
	if err := codeGraph.BuildFromParsed(parsedFiles); err != nil {
		return err
	}

	// Resolve call edges
	if err := codeGraph.ResolveCallEdges(); err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠ Warning: Failed to resolve call edges: %v\n", err)
	}

	return nil
}

func generateEmbeddingsForDB(database *sql.DB, ollamaURL, model string) {
	// Initialize vector store
	store := vector.NewVectorStore(database)
	if err := store.InitSchema(); err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠ Failed to init vector store: %v\n", err)
		return
	}

	embedder := vector.NewEmbedderWithConfig(ollamaURL, model)

	// Get nodes to embed
	rows, err := database.Query(`SELECT id, name, type, file, signature, docstring, body FROM nodes`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠ Failed to query nodes: %v\n", err)
		return
	}
	defer rows.Close()

	embedded := 0
	for rows.Next() {
		var id, name, nodeType, file string
		var sig, doc, body sql.NullString
		if err := rows.Scan(&id, &name, &nodeType, &file, &sig, &doc, &body); err != nil {
			continue
		}

		text := vector.BuildEmbedText(name, nodeType, sig.String, doc.String, body.String)

		embedding, err := embedder.Embed(text)
		if err != nil {
			continue
		}

		if err := store.Store(id, embedding); err != nil {
			continue
		}

		embedded++
		if embedded%50 == 0 {
			fmt.Fprintf(os.Stderr, "  🧠 Embedded %d nodes...\n", embedded)
		}
	}

	fmt.Fprintf(os.Stderr, "  ✅ Embedding complete (%d nodes)\n", embedded)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isTerminal() bool {
	// Check if running in a terminal (simple heuristic)
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) != 0
}
