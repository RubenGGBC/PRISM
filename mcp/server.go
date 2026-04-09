package mcp

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/pkoukk/tiktoken-go"
	"github.com/ruffini/prism/graph"
	"github.com/ruffini/prism/internal/models"
	"github.com/ruffini/prism/mdchunker"
	"github.com/ruffini/prism/parser"
	"github.com/ruffini/prism/vector"
)

// MCPServer wraps the MCP server with code intelligence capabilities
type MCPServer struct {
	graph     *graph.CodeGraph
	vector    *vector.VectorStore
	docStore  *vector.DocVectorStore
	embedder  *vector.Embedder
	server    *server.MCPServer
	logger    *log.Logger
	tokenizer *tiktoken.Tiktoken
}

// NewMCPServer creates a new MCP server with code intelligence tools
func NewMCPServer(g *graph.CodeGraph, v *vector.VectorStore, e *vector.Embedder) *MCPServer {
	// Create log file for MCP server
	logFile, err := os.OpenFile("mcp_server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logFile = os.Stderr
	}

	logger := log.New(logFile, "[MCP] ", log.LstdFlags)
	logger.Println("🚀 MCP Server starting...")

	// Initialize tokenizer (cl100k_base used by GPT-4 and Claude)
	tokenizer, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		logger.Printf("⚠️  Failed to initialize tokenizer: %v (using estimation)", err)
		tokenizer = nil
	} else {
		logger.Println("✅ Real token counter initialized (cl100k_base)")
	}

	// Initialize doc vector store and create schema if needed
	docStore := vector.NewDocVectorStore(v.DB)
	if err := docStore.InitSchema(); err != nil {
		logger.Printf("⚠️  Failed to initialize doc schema: %v", err)
	} else {
		logger.Println("✅ Doc schema initialized")
	}

	m := &MCPServer{
		graph:     g,
		vector:    v,
		docStore:  docStore,
		embedder:  e,
		logger:    logger,
		tokenizer: tokenizer,
	}

	// Create MCP server
	s := server.NewMCPServer(
		"Code Intelligence Platform",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithLogging(),
	)

	m.server = s

	// Register tools
	m.registerTools()
	logger.Println("✅ Tools registered: search_context, get_file_smart, trace_impact, list_functions, index_docs, search_docs")

	return m
}

// Start starts the MCP server using stdio transport
func (m *MCPServer) Start() error {
	m.logger.Println("📡 Starting stdio transport...")
	return server.ServeStdio(m.server)
}

// registerTools registers all code intelligence tools
func (m *MCPServer) registerTools() {
	// Tool 1: search_context
	m.server.AddTool(
		mcp.NewTool("search_context",
			mcp.WithDescription("Search for relevant functions/code based on a natural language query. Returns the most relevant code snippets using semantic search."),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description("The search query in natural language"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of results to return (default: 5)"),
			),
		),
		m.handleSearchContext,
	)

	// Tool 2: get_file_smart
	m.server.AddTool(
		mcp.NewTool("get_file_smart",
			mcp.WithDescription("Get a specific function or class from a file without loading unnecessary code. Returns the code along with its callers, callees, and metadata."),
			mcp.WithString("file",
				mcp.Required(),
				mcp.Description("The file path containing the symbol"),
			),
			mcp.WithString("symbol",
				mcp.Required(),
				mcp.Description("The function or class name to retrieve"),
			),
		),
		m.handleGetFileSmart,
	)

	// Tool 3: trace_impact
	m.server.AddTool(
		mcp.NewTool("trace_impact",
			mcp.WithDescription("Analyze the impact of changing a function. Shows callers (upstream) and/or callees (downstream) up to a configurable depth."),
			mcp.WithString("function_id",
				mcp.Required(),
				mcp.Description("The function ID (e.g. 'src/auth.go:Login')"),
			),
			mcp.WithString("direction",
				mcp.Description("'upstream' (who calls this), 'downstream' (what this calls), or 'both' (default: upstream)"),
			),
			mcp.WithNumber("depth",
				mcp.Description("How many levels deep to traverse (default: 3, max: 5)"),
			),
		),
		m.handleTraceImpact,
	)

	// Tool 4: list_functions
	m.server.AddTool(
		mcp.NewTool("list_functions",
			mcp.WithDescription("List functions, methods, and classes. Supports pagination, type filter, and file filter."),
			mcp.WithString("file", mcp.Description("Filter by file path")),
			mcp.WithString("pattern", mcp.Description("Filter by name pattern")),
			mcp.WithString("type", mcp.Description("Filter by type: function, method, class")),
			mcp.WithNumber("limit", mcp.Description("Max results per page (default: 50)")),
			mcp.WithNumber("offset", mcp.Description("Pagination offset (default: 0)")),
		),
		m.handleListFunctions,
	)

	// Tool 5: index_docs
	m.server.AddTool(
		mcp.NewTool("index_docs",
			mcp.WithDescription("Index project markdown files into the vector store for semantic search via search_docs."),
			mcp.WithString("path",
				mcp.Description("Directory to scan for .md files. Defaults to ./docs, ./.github, and root *.md files"),
			),
		),
		m.handleIndexDocs,
	)

	// Tool 6: search_docs
	m.server.AddTool(
		mcp.NewTool("search_docs",
			mcp.WithDescription("Semantic search over indexed markdown documentation. Run index_docs first."),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description("Natural language search query"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of results to return (default: 5)"),
			),
		),
		m.handleSearchDocs,
	)

	// Tool 7: use_profile
	m.server.AddTool(
		mcp.NewTool("use_profile",
			mcp.WithDescription("Load a named context profile — returns all nodes in the profile with their annotations. Use before starting work on a specific area."),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("Profile name (e.g. 'auth', 'checkout', 'frontend')"),
			),
		),
		m.handleUseProfile,
	)

	// Tool 8: find_similar
	m.server.AddTool(
		mcp.NewTool("find_similar",
			mcp.WithDescription("Find functions with similar implementation to the given function. Useful for detecting near-duplicates and refactoring candidates."),
			mcp.WithString("function_id",
				mcp.Required(),
				mcp.Description("The function ID to find similar functions for (e.g. 'src/auth.go:validateForm')"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Max similar functions to return (default: 5)"),
			),
			mcp.WithNumber("threshold",
				mcp.Description("Minimum similarity score 0.0-1.0 (default: 0.75)"),
			),
		),
		m.handleFindSimilar,
	)
}

// handleSearchContext handles the search_context tool
func (m *MCPServer) handleSearchContext(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		m.logger.Println("❌ search_context: invalid arguments")
		return mcp.NewToolResultError("invalid arguments"), nil
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		m.logger.Println("❌ search_context: missing query parameter")
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	limit := 5
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	m.logger.Printf("🔍 search_context called: query='%s', limit=%d", query, limit)

	// Try semantic search if embedder and vector store are available
	if m.embedder != nil && m.vector != nil {
		count, err := m.vector.Count()
		if err == nil && count > 0 {
			results, err := m.vector.SearchText(m.embedder, query, limit)
			if err == nil && len(results) > 0 {
				elapsed := time.Since(start)

				// For semantic search results, include annotations
				var nodesWithAnnotations []*models.GraphNode
				for _, result := range results {
					node, _, _ := m.graph.GetNodeWithAnnotations(result.NodeID)
					if node != nil {
						nodesWithAnnotations = append(nodesWithAnnotations, node)
					}
				}

				// Count tokens in semantic results
				totalTokens := 0
				for _, result := range results {
					totalTokens += m.countTokens(result.NodeID)
				}
				tokensWithoutMCP := len(results) * 2000 // would need full files
				tokenSavings := tokensWithoutMCP - totalTokens
				savingsPercent := 0.0
				if tokensWithoutMCP > 0 {
					savingsPercent = (float64(tokenSavings) / float64(tokensWithoutMCP)) * 100
				}

				m.logger.Printf("✅ search_context: found %d results (semantic) in %v | %d tokens (saved ~%d tokens, %.0f%%)",
					len(results), elapsed, totalTokens, tokenSavings, savingsPercent)
				return m.formatVectorResults(results, nodesWithAnnotations), nil
			}
		}
	}

	// Fallback to keyword search
	nodes, err := m.graph.SearchByName(query)
	if err != nil {
		m.logger.Printf("❌ search_context: search failed: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	if len(nodes) == 0 {
		m.logger.Println("⚠️  search_context: no results found")
		return mcp.NewToolResultText("No results found for query: " + query), nil
	}

	// Limit results
	if len(nodes) > limit {
		nodes = nodes[:limit]
	}

	elapsed := time.Since(start)

	// Count tokens in results
	totalTokens := 0
	for _, node := range nodes {
		totalTokens += m.countTokens(node.Name + " " + node.Signature)
	}
	tokensWithoutMCP := len(nodes) * 2000 // would need full files
	tokenSavings := tokensWithoutMCP - totalTokens
	savingsPercent := 0.0
	if tokensWithoutMCP > 0 {
		savingsPercent = (float64(tokenSavings) / float64(tokensWithoutMCP)) * 100
	}

	m.logger.Printf("✅ search_context: found %d results (keyword) in %v | %d tokens (saved ~%d tokens, %.0f%%)",
		len(nodes), elapsed, totalTokens, tokenSavings, savingsPercent)

	// Track tokens served vs tokens that would have been needed reading full files
	m.graph.DB.Exec(`
		INSERT INTO mcp_sessions (id, tokens_served, tokens_saved)
		VALUES (?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			tokens_served = tokens_served + excluded.tokens_served,
			tokens_saved = tokens_saved + excluded.tokens_saved`,
		"current_session", totalTokens, tokenSavings)

	return formatNodesResult(nodes), nil
}

// handleGetFileSmart handles the get_file_smart tool
func (m *MCPServer) handleGetFileSmart(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		m.logger.Println("❌ get_file_smart: invalid arguments")
		return mcp.NewToolResultError("invalid arguments"), nil
	}

	file, ok := args["file"].(string)
	if !ok || file == "" {
		m.logger.Println("❌ get_file_smart: missing file parameter")
		return mcp.NewToolResultError("file parameter is required"), nil
	}

	symbol, ok := args["symbol"].(string)
	if !ok || symbol == "" {
		m.logger.Println("❌ get_file_smart: missing symbol parameter")
		return mcp.NewToolResultError("symbol parameter is required"), nil
	}

	m.logger.Printf("📄 get_file_smart called: file='%s', symbol='%s'", file, symbol)

	// Get all nodes in the file (supports slash/backslash and absolute path variants)
	nodes, err := m.graph.GetNodesByFileFlexible(file)
	if err != nil {
		m.logger.Printf("❌ get_file_smart: error retrieving file: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("error retrieving file: %v", err)), nil
	}
	if len(nodes) == 0 {
		m.logger.Printf("❌ get_file_smart: file not found in index")
		return mcp.NewToolResultError(fmt.Sprintf("file '%s' not found in index", file)), nil
	}

	// Find the matching symbol
	node := findNodeBySymbol(nodes, symbol)

	if node == nil {
		m.logger.Printf("❌ get_file_smart: symbol not found")
		suggestions := make([]string, 0, min(6, len(nodes)))
		for i := 0; i < len(nodes) && i < 6; i++ {
			suggestions = append(suggestions, nodes[i].Name)
		}
		if len(suggestions) > 0 {
			return mcp.NewToolResultError(
				fmt.Sprintf("symbol '%s' not found in file '%s'. Try one of: %s",
					symbol, nodes[0].File, strings.Join(suggestions, ", ")),
			), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("symbol '%s' not found in file '%s'", symbol, file)), nil
	}

	// Get annotations for the node
	annotations, _ := m.graph.GetNodeAnnotations(node.ID)
	if len(annotations) > 0 {
		if comments, ok := annotations["comments"].(string); ok && comments != "" {
			node.Comments = comments
		}
		if tags, ok := annotations["tags"].([]string); ok {
			node.Tags = tags
		}
		if metadata, ok := annotations["custom_metadata"].(map[string]string); ok {
			node.CustomMetadata = metadata
		}
	}

	// Get callers and callees
	callerIDs, _ := m.graph.GetCallers(node.ID)
	calleeIDs, _ := m.graph.GetCallees(node.ID)

	elapsed := time.Since(start)

	// Count actual tokens
	tokensWithMCP := m.countTokens(node.Signature + "\n" + node.Body)
	tokensWithoutMCP := 2000 // typical full file read (estimated)
	tokenSavings := tokensWithoutMCP - tokensWithMCP
	savingsPercent := (float64(tokenSavings) / float64(tokensWithoutMCP)) * 100

	m.logger.Printf("✅ get_file_smart: found symbol with %d callers, %d callees in %v | %d tokens (saved ~%d tokens, %.0f%%)",
		len(callerIDs), len(calleeIDs), elapsed, tokensWithMCP, tokenSavings, savingsPercent)

	// Get metadata
	metadata, _ := m.graph.GetNodeMetadata(node.ID)

	result := fmt.Sprintf("## %s (%s)\n\n", node.Name, node.Type)
	result += fmt.Sprintf("**File:** %s (lines %d-%d)\n\n", node.File, node.Line, node.EndLine)

	// Add metadata section if exists
	if len(metadata) > 0 {
		result += "**Metadata:**\n"
		if deprecated, ok := metadata["deprecated"].(bool); ok && deprecated {
			result += "- DEPRECATED"
			if reason, ok := metadata["deprecated_reason"].(string); ok {
				result += fmt.Sprintf(": %s", reason)
			}
			result += "\n"
		}
		if hooks, ok := metadata["hooks"].(string); ok && hooks != "" {
			result += fmt.Sprintf("- **Hooks:** %s\n", hooks)
		}
		if todos, ok := metadata["todos"].(string); ok && todos != "" {
			result += fmt.Sprintf("- **TODOs:** %s\n", todos)
		}
		if authors, ok := metadata["authors"].(string); ok && authors != "" {
			result += fmt.Sprintf("- **Authors:** %s\n", authors)
		}
		result += "\n"
	}

	if node.Signature != "" {
		result += fmt.Sprintf("**Signature:**\n```\n%s\n```\n\n", node.Signature)
	}

	if node.Body != "" {
		result += fmt.Sprintf("**Code:**\n```\n%s\n```\n\n", node.Body)
	}

	if len(callerIDs) > 0 {
		result += fmt.Sprintf("**Called by:** %v\n\n", callerIDs)
	}

	if len(calleeIDs) > 0 {
		result += fmt.Sprintf("**Calls:** %v\n\n", calleeIDs)
	}

	// Add user annotations section if any exist
	if len(node.Comments) > 0 || len(node.Tags) > 0 || len(node.CustomMetadata) > 0 {
		result += "**User Annotations:**\n"
		if node.Comments != "" {
			result += fmt.Sprintf("- **Comment:** %s\n", node.Comments)
		}
		if len(node.Tags) > 0 {
			result += fmt.Sprintf("- **Tags:** %v\n", node.Tags)
		}
		if len(node.CustomMetadata) > 0 {
			result += "- **Metadata:**\n"
			for key, value := range node.CustomMetadata {
				result += fmt.Sprintf("  - %s: %s\n", key, value)
			}
		}
		result += "\n"
	}

	return mcp.NewToolResultText(result), nil
}

// handleTraceImpact handles the trace_impact tool
func (m *MCPServer) handleTraceImpact(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments"), nil
	}

	functionID, ok := args["function_id"].(string)
	if !ok || functionID == "" {
		return mcp.NewToolResultError("function_id parameter is required"), nil
	}

	direction := "upstream"
	if d, ok := args["direction"].(string); ok && d != "" {
		direction = d
	}

	depth := 3
	if d, ok := args["depth"].(float64); ok && d > 0 {
		depth = int(d)
		if depth > 5 {
			depth = 5
		}
	}

	m.logger.Printf("🎯 trace_impact: function_id='%s' direction='%s' depth=%d", functionID, direction, depth)

	node, err := m.graph.GetNode(functionID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("error: %v", err)), nil
	}
	if node == nil {
		return mcp.NewToolResultError(fmt.Sprintf("function '%s' not found", functionID)), nil
	}

	result := fmt.Sprintf("## Impact Analysis: %s\n\n", node.Name)

	if direction == "upstream" || direction == "both" {
		directCallers, _ := m.graph.GetCallers(functionID)
		allCallers := make(map[string]bool)
		queue := make([]string, len(directCallers))
		copy(queue, directCallers)
		for d := 0; d < depth && len(queue) > 0; d++ {
			next := []string{}
			for _, caller := range queue {
				if !allCallers[caller] {
					allCallers[caller] = true
					lvl, _ := m.graph.GetCallers(caller)
					next = append(next, lvl...)
				}
			}
			queue = next
		}
		result += fmt.Sprintf("### Upstream (callers) — %d direct, %d total\n", len(directCallers), len(allCallers))
		for _, c := range directCallers {
			result += fmt.Sprintf("- %s\n", c)
		}
		result += "\n"
	}

	if direction == "downstream" || direction == "both" {
		directCallees, _ := m.graph.GetCallees(functionID)
		allCallees, err := m.graph.GetCalleesTransitive(functionID, depth)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("error getting callees: %v", err)), nil
		}
		result += fmt.Sprintf("### Downstream (callees) — %d direct, %d total\n", len(directCallees), len(allCallees))
		for _, c := range directCallees {
			result += fmt.Sprintf("- %s\n", c)
		}
		result += "\n"
	}

	elapsed := time.Since(start)
	m.logger.Printf("✅ trace_impact: done in %v", elapsed)
	return mcp.NewToolResultText(result), nil
}

// handleListFunctions handles the list_functions tool
func (m *MCPServer) handleListFunctions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}

	file, _ := args["file"].(string)
	pattern, _ := args["pattern"].(string)
	nodeType, _ := args["type"].(string)

	limit := 50
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}
	offset := 0
	if o, ok := args["offset"].(float64); ok && o >= 0 {
		offset = int(o)
	}

	m.logger.Printf("📋 list_functions: file='%s' pattern='%s' type='%s' limit=%d offset=%d", file, pattern, nodeType, limit, offset)

	var nodes []models.GraphNode
	var err error

	if file != "" {
		nodes, err = m.graph.GetNodesByFile(file)
	} else if pattern != "" {
		nodes, err = m.graph.SearchByName(pattern)
	} else {
		nodes, err = m.graph.GetAllNodesPaginated(limit, offset, nodeType)
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("error: %v", err)), nil
	}

	// Apply type filter post-query when file or pattern was used
	if nodeType != "" && (file != "" || pattern != "") {
		filtered := nodes[:0]
		for _, n := range nodes {
			if n.Type == nodeType {
				filtered = append(filtered, n)
			}
		}
		nodes = filtered
	}

	elapsed := time.Since(start)
	m.logger.Printf("✅ list_functions: %d results in %v", len(nodes), elapsed)
	return formatNodesResult(nodes), nil
}

// formatVectorResults formats vector search results with annotations
func (m *MCPServer) formatVectorResults(results []vector.SearchResult, nodesWithAnnotations []*models.GraphNode) *mcp.CallToolResult {
	output := "## Semantic Search Results\n\n"

	for i, r := range results {
		var node *models.GraphNode
		if i < len(nodesWithAnnotations) && nodesWithAnnotations[i] != nil {
			node = nodesWithAnnotations[i]
		} else {
			n, _ := m.graph.GetNode(r.NodeID)
			node = n
		}

		if node == nil {
			continue
		}

		output += fmt.Sprintf("### %d. (similarity: %.2f%%)\n", i+1, r.Similarity*100)
		output += m.formatNodeWithAnnotations(node.ID, node.Name, node.Type, node.File, node.Line, node.Signature, "")
		output += "\n"
	}

	return mcp.NewToolResultText(output)
}

// handleIndexDocs indexes .md files into the doc vector store
func (m *MCPServer) handleIndexDocs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	args, _ := request.Params.Arguments.(map[string]interface{})
	customPath, _ := args["path"].(string)

	// Discover .md files
	var mdFiles []string
	var err error
	if customPath != "" {
		mdFiles, err = findMdFiles(customPath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to scan path: %v", err)), nil
		}
	} else {
		mdFiles = discoverDefaultMdFiles()
	}

	if len(mdFiles) == 0 {
		return mcp.NewToolResultText("No markdown files found to index."), nil
	}

	m.logger.Printf("📚 index_docs: found %d .md files", len(mdFiles))

	totalChunks := 0
	totalFiles := 0
	var errors []string

	for _, filePath := range mdFiles {
		content, err := os.ReadFile(filePath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", filePath, err))
			continue
		}

		// Use relative path as stable identifier
		relPath := filePath
		if abs, err := filepath.Abs(filePath); err == nil {
			if cwd, err := os.Getwd(); err == nil {
				if rel, err := filepath.Rel(cwd, abs); err == nil {
					relPath = filepath.ToSlash(rel)
				}
			}
		}

		chunks := mdchunker.ChunkText(string(content), relPath)
		if len(chunks) == 0 {
			continue
		}

		// Delete existing chunks for this file (cascade deletes embeddings)
		if err := m.docStore.DeleteChunksForFile(relPath); err != nil {
			errors = append(errors, fmt.Sprintf("%s: delete failed: %v", relPath, err))
			continue
		}

		// Insert chunks
		for _, c := range chunks {
			if err := m.docStore.StoreChunk(c.ID, c.File, c.ChunkIndex, c.LineStart, c.Content); err != nil {
				errors = append(errors, fmt.Sprintf("%s chunk %d: %v", relPath, c.ChunkIndex, err))
				continue
			}
		}

		// Generate and store embeddings if Ollama is available
		if m.embedder != nil {
			texts := make([]string, len(chunks))
			ids := make([]string, len(chunks))
			for i, c := range chunks {
				texts[i] = c.Content
				ids[i] = c.ID
			}
			embeddings, err := m.embedder.EmbedBatch(texts)
			if err != nil {
				m.logger.Printf("  ⚠ %s: embedding skipped (Ollama unavailable): %v", relPath, err)
				errors = append(errors, fmt.Sprintf("%s: embedding skipped (Ollama unavailable)", relPath))
			} else if err := m.docStore.StoreEmbeddingsBatch(ids, embeddings); err != nil {
				errors = append(errors, fmt.Sprintf("%s: store embeddings failed: %v", relPath, err))
			}
		}

		totalChunks += len(chunks)
		totalFiles++
		m.logger.Printf("  ✓ %s (%d chunks)", relPath, len(chunks))
	}

	// Also index inline comments and JSDoc from source files
	codeRoot := "."
	if customPath != "" {
		codeRoot = customPath
	}
	codeFilesIndexed := 0
	filepath.Walk(codeRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if parser.ShouldSkipPath(path) || !parser.IsCodeFile(path) {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		comments := extractCodeComments(source)
		if len(strings.TrimSpace(comments)) < 20 {
			return nil
		}
		relPath := path
		if abs, err2 := filepath.Abs(path); err2 == nil {
			if cwd, err3 := os.Getwd(); err3 == nil {
				if rel, err4 := filepath.Rel(cwd, abs); err4 == nil {
					relPath = filepath.ToSlash(rel)
				}
			}
		}
		docID := "comments:" + relPath
		chunks := mdchunker.ChunkText(comments, docID)
		if len(chunks) == 0 {
			return nil
		}
		embedTexts := make([]string, len(chunks))
		for i, c := range chunks {
			embedTexts[i] = c.Content
		}
		embeddings, err := m.embedder.EmbedBatch(embedTexts)
		if err != nil {
			return nil
		}
		m.docStore.DeleteChunksForFile(docID)
		for i, chunk := range chunks {
			m.docStore.StoreChunk(chunk.ID, chunk.File, chunk.ChunkIndex, chunk.LineStart, chunk.Content)
			if i < len(embeddings) {
				m.vector.Store(fmt.Sprintf("%s#%d", docID, i), embeddings[i])
			}
		}
		codeFilesIndexed++
		return nil
	})

	elapsed := time.Since(start)
	result := fmt.Sprintf("## index_docs complete (%v)\n\n", elapsed)
	result += fmt.Sprintf("- Files indexed: **%d**\n", totalFiles)
	result += fmt.Sprintf("- Chunks created: **%d**\n", totalChunks)
	result += fmt.Sprintf("- Source files (comments): **%d**\n", codeFilesIndexed)
	if len(errors) > 0 {
		result += fmt.Sprintf("\n**Errors (%d):**\n", len(errors))
		for _, e := range errors {
			result += fmt.Sprintf("- %s\n", e)
		}
	}

	m.logger.Printf("✅ index_docs: %d files, %d chunks in %v, %d source files (comments)", totalFiles, totalChunks, elapsed, codeFilesIndexed)
	return mcp.NewToolResultText(result), nil
}

// extractCodeComments extracts line and block comments from source code
func extractCodeComments(source []byte) string {
	content := string(source)
	var comments []string
	lines := strings.Split(content, "\n")
	inBlock := false
	var blockLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if inBlock {
			blockLines = append(blockLines, trimmed)
			if strings.Contains(trimmed, "*/") {
				inBlock = false
				comments = append(comments, strings.Join(blockLines, " "))
				blockLines = nil
			}
			continue
		}
		if strings.HasPrefix(trimmed, "/*") {
			inBlock = true
			blockLines = []string{trimmed}
			if strings.Contains(trimmed, "*/") {
				inBlock = false
				comments = append(comments, trimmed)
				blockLines = nil
			}
			continue
		}
		if strings.HasPrefix(trimmed, "//") {
			comments = append(comments, strings.TrimPrefix(trimmed, "//"))
		}
	}
	return strings.Join(comments, "\n")
}

// handleSearchDocs handles the search_docs tool
func (m *MCPServer) handleSearchDocs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments"), nil
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	limit := 5
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	m.logger.Printf("📖 search_docs called: query='%s', limit=%d", query, limit)

	if !m.docStore.HasChunks() {
		return mcp.NewToolResultText("No docs indexed yet. Run index_docs first."), nil
	}

	searchMode := "semantic"
	var results []vector.DocSearchResult
	var err error

	embeddingCount, _ := m.docStore.Count()
	if m.embedder != nil && embeddingCount > 0 {
		results, err = m.docStore.SearchText(m.embedder, query, limit)
		if err != nil {
			m.logger.Printf("⚠️  search_docs: semantic search failed, falling back to keyword: %v", err)
			results, err = m.docStore.SearchKeyword(query, limit)
			searchMode = "keyword (Ollama unavailable)"
		}
	} else {
		results, err = m.docStore.SearchKeyword(query, limit)
		searchMode = "keyword (no embeddings — run index_docs with Ollama running)"
	}

	if err != nil {
		m.logger.Printf("❌ search_docs: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText("No results found for: " + query), nil
	}

	elapsed := time.Since(start)
	output := fmt.Sprintf("## Doc Search Results — %s (%v)\n\n", searchMode, elapsed)
	for i, r := range results {
		output += fmt.Sprintf("### %d. %s (line ~%d, similarity: %.0f%%)\n", i+1, r.File, r.LineStart, r.Similarity*100)
		// Trim long chunks for display
		snippet := r.Content
		if len(snippet) > 500 {
			snippet = snippet[:500] + "..."
		}
		output += fmt.Sprintf("\n> %s\n\n", strings.ReplaceAll(snippet, "\n", "\n> "))
	}

	m.logger.Printf("✅ search_docs: %d results in %v", len(results), elapsed)
	return mcp.NewToolResultText(output), nil
}

// findMdFiles recursively finds all .md files under root
func findMdFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".md" {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// discoverDefaultMdFiles finds .md files in ./docs, ./.github, and root *.md
func discoverDefaultMdFiles() []string {
	seen := make(map[string]bool)
	var files []string

	addFiles := func(paths []string) {
		for _, p := range paths {
			abs, err := filepath.Abs(p)
			if err != nil {
				continue
			}
			if !seen[abs] {
				seen[abs] = true
				files = append(files, p)
			}
		}
	}

	// ./docs/**/*.md
	if docsFiles, err := findMdFiles("docs"); err == nil {
		addFiles(docsFiles)
	}

	// ./.github/**/*.md
	if ghFiles, err := findMdFiles(".github"); err == nil {
		addFiles(ghFiles)
	}

	// root *.md
	if rootFiles, err := filepath.Glob("*.md"); err == nil {
		addFiles(rootFiles)
	}

	return files
}

// formatNodeWithAnnotations formats a node including its human annotations
func (m *MCPServer) formatNodeWithAnnotations(nodeID, name, nodeType, file string, line int, signature, docstring string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %s (%s:%d)\n", name, file, line))
	sb.WriteString(fmt.Sprintf("Type: %s\n", nodeType))

	// Load structured annotations
	why, status, knownBug, entryPoint, err := m.graph.GetAnnotations(nodeID)
	if err == nil {
		if status != "" && status != "stable" {
			sb.WriteString(fmt.Sprintf("Status: %s\n", status))
		}
		if entryPoint {
			sb.WriteString("Entry point: YES\n")
		}
		if why != "" {
			sb.WriteString(fmt.Sprintf("Why: %s\n", why))
		}
		if knownBug != "" {
			sb.WriteString(fmt.Sprintf("Known bug: %s\n", knownBug))
		}
	}

	// Load user comments
	var comment string
	row := m.graph.DB.QueryRow(`SELECT comment FROM node_comments WHERE node_id = ?`, nodeID)
	row.Scan(&comment)
	if comment != "" {
		sb.WriteString(fmt.Sprintf("Notes: %s\n", comment))
	}

	if signature != "" {
		sb.WriteString(fmt.Sprintf("Signature: %s\n", signature))
	}
	if docstring != "" {
		sb.WriteString(fmt.Sprintf("Docs: %s\n", docstring))
	}
	return sb.String()
}

// loadStoredProfile loads a profile by ID and returns its formatted content
func (m *MCPServer) loadStoredProfile(profileID, name string) (*mcp.CallToolResult, error) {
	rows, err := m.graph.DB.Query(`
		SELECT n.id, n.name, n.type, n.file, n.line, n.signature, n.docstring
		FROM profile_nodes pn
		JOIN nodes n ON n.id = pn.node_id
		WHERE pn.profile_id = ?`, profileID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("query error: %v", err)), nil
	}
	defer rows.Close()

	var result strings.Builder
	result.WriteString(fmt.Sprintf("# Context Profile: %s\n\n", name))

	count := 0
	totalTokens := 0
	for rows.Next() {
		var id, nName, nType, file, signature, docstring string
		var line int
		if err := rows.Scan(&id, &nName, &nType, &file, &line, &signature, &docstring); err != nil {
			continue
		}
		result.WriteString(m.formatNodeWithAnnotations(id, nName, nType, file, line, signature, docstring))
		totalTokens += m.countTokens(signature + " " + docstring)
		count++
	}

	result.WriteString(fmt.Sprintf("\n---\n*%d functions, ~%d tokens*\n", count, totalTokens))
	return mcp.NewToolResultText(result.String()), nil
}

// handleUseProfile handles the use_profile MCP tool
func (m *MCPServer) handleUseProfile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments"), nil
	}
	name, _ := args["name"].(string)
	if name == "" {
		return mcp.NewToolResultError("profile name is required"), nil
	}

	// Try stored profile
	var profileID string
	err := m.graph.DB.QueryRow(`SELECT id FROM profiles WHERE name = ?`, name).Scan(&profileID)
	if err == nil {
		return m.loadStoredProfile(profileID, name)
	}

	// Try auto-generating from directory clustering
	dirs, _ := m.graph.GetDistinctDirectories()
	for _, dir := range dirs {
		if strings.EqualFold(dir, name) {
			nodes, err := m.graph.GetNodesByDirectoryPrefix(dir, 30)
			if err != nil || len(nodes) == 0 {
				break
			}
			var result strings.Builder
			result.WriteString(fmt.Sprintf("# Auto-generated Profile: %s\n\n", name))
			result.WriteString(fmt.Sprintf("*Inferred from directory `%s/` — top %d functions by PageRank*\n\n", dir, len(nodes)))
			for _, n := range nodes {
				result.WriteString(m.formatNodeWithAnnotations(n.ID, n.Name, n.Type, n.File, n.Line, n.Signature, ""))
			}
			return mcp.NewToolResultText(result.String()), nil
		}
	}

	// Not found — suggest alternatives
	storedNames, _ := m.graph.ListProfileNames()
	suggestion := fmt.Sprintf("Profile '%s' not found.\n\n", name)
	if len(storedNames) > 0 {
		suggestion += "**Stored profiles:** " + strings.Join(storedNames, ", ") + "\n\n"
	}
	if len(dirs) > 0 {
		suggestion += "**Auto-generatable from directories:** " + strings.Join(dirs, ", ") + "\n\n"
	}
	suggestion += "Use one of the names above, or create a profile via the web UI."
	return mcp.NewToolResultError(suggestion), nil
}

// handleFindSimilar handles the find_similar tool
func (m *MCPServer) handleFindSimilar(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments"), nil
	}

	functionID, ok := args["function_id"].(string)
	if !ok || functionID == "" {
		return mcp.NewToolResultError("function_id is required"), nil
	}

	limit := 5
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	threshold := float32(0.75)
	if t, ok := args["threshold"].(float64); ok && t > 0 {
		threshold = float32(t)
	}

	m.logger.Printf("🔍 find_similar: function_id='%s' limit=%d threshold=%.2f", functionID, limit, threshold)

	node, err := m.graph.GetNode(functionID)
	if err != nil || node == nil {
		return mcp.NewToolResultError(fmt.Sprintf("function '%s' not found", functionID)), nil
	}

	// Get embedding for the source node
	sourceEmbedding, err := m.vector.Get(functionID)
	if err != nil || sourceEmbedding == nil {
		// Compute on the fly if not stored
		text := vector.BuildEmbedText(node.Name, node.Type, node.Signature, "", node.Body)
		sourceEmbedding, err = m.embedder.Embed(text)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("could not embed function: %v", err)), nil
		}
	}

	results, err := m.vector.SearchWithThreshold(sourceEmbedding, threshold)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search error: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Similar functions to `%s`\n\n", node.Name))
	sb.WriteString(fmt.Sprintf("*Threshold: %.0f%%, top %d results*\n\n", threshold*100, limit))

	count := 0
	for _, r := range results {
		if r.NodeID == functionID {
			continue
		}
		similar, err := m.graph.GetNode(r.NodeID)
		if err != nil || similar == nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("### %.0f%% — `%s` (%s:%d)\n", r.Similarity*100, similar.Name, similar.File, similar.Line))
		if similar.Signature != "" {
			sb.WriteString(fmt.Sprintf("```\n%s\n```\n", similar.Signature))
		}
		sb.WriteString("\n")
		count++
		if count >= limit {
			break
		}
	}

	if count == 0 {
		sb.WriteString("No similar functions found above the threshold.\n")
	}

	elapsed := time.Since(start)
	m.logger.Printf("✅ find_similar: %d results in %v", count, elapsed)
	return mcp.NewToolResultText(sb.String()), nil
}

func findNodeBySymbol(nodes []models.GraphNode, symbol string) *models.GraphNode {
	candidates := symbolCandidates(symbol)
	if len(candidates) == 0 {
		return nil
	}

	// 1) Exact/normalized ID match
	for i := range nodes {
		nodeIDSlash := strings.ReplaceAll(nodes[i].ID, `\`, "/")
		for _, candidate := range candidates {
			if nodes[i].ID == candidate || strings.EqualFold(nodes[i].ID, candidate) {
				return &nodes[i]
			}
			candidateSlash := strings.ReplaceAll(candidate, `\`, "/")
			if nodeIDSlash == candidateSlash || strings.EqualFold(nodeIDSlash, candidateSlash) {
				return &nodes[i]
			}
		}
	}

	// 2) Name match (exact first)
	for i := range nodes {
		for _, candidate := range candidates {
			if nodes[i].Name == candidate {
				return &nodes[i]
			}
		}
	}

	// 3) Name suffix match for qualified names (e.g., Class.method -> method)
	for i := range nodes {
		for _, candidate := range candidates {
			if strings.EqualFold(nodes[i].Name, candidate) {
				return &nodes[i]
			}
			if idx := strings.LastIndex(nodes[i].Name, "."); idx >= 0 && idx+1 < len(nodes[i].Name) {
				if nodes[i].Name[idx+1:] == candidate || strings.EqualFold(nodes[i].Name[idx+1:], candidate) {
					return &nodes[i]
				}
			}
		}
	}

	return nil
}

func symbolCandidates(symbol string) []string {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return nil
	}

	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}

	add(symbol)
	if idx := strings.LastIndex(symbol, ":"); idx >= 0 && idx+1 < len(symbol) {
		add(symbol[idx+1:])
	}
	if idx := strings.LastIndex(symbol, "::"); idx >= 0 && idx+2 < len(symbol) {
		add(symbol[idx+2:])
	}
	if idx := strings.LastIndex(symbol, "."); idx >= 0 && idx+1 < len(symbol) {
		add(symbol[idx+1:])
	}

	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// formatNodesResult formats a list of nodes
func formatNodesResult(nodes []models.GraphNode) *mcp.CallToolResult {
	if len(nodes) == 0 {
		return mcp.NewToolResultText("No results found.")
	}

	output := fmt.Sprintf("## Found %d results\n\n", len(nodes))

	for i, node := range nodes {
		output += fmt.Sprintf("### %d. %s\n", i+1, node.Name)
		output += fmt.Sprintf("**Type:** %s | **File:** %s:%d\n\n", node.Type, node.File, node.Line)

		if node.Signature != "" {
			output += fmt.Sprintf("```\n%s\n```\n\n", node.Signature)
		}
	}

	return mcp.NewToolResultText(output)
}
