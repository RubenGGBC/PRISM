package mcp

import (
	"context"
	"database/sql"
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
			mcp.WithDescription("Show the blast radius - what functions would be affected if you change this one. Returns direct callers and transitive callers."),
			mcp.WithString("function_id",
				mcp.Required(),
				mcp.Description("The function ID to analyze (format: file.py:function_name)"),
			),
		),
		m.handleTraceImpact,
	)

	// Tool 4: list_functions
	m.server.AddTool(
		mcp.NewTool("list_functions",
			mcp.WithDescription("List all functions in a file or matching a search pattern."),
			mcp.WithString("file",
				mcp.Description("Optional: filter by file path"),
			),
			mcp.WithString("pattern",
				mcp.Description("Optional: filter by name pattern (substring match)"),
			),
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

	// Get all nodes in the file
	nodes, err := m.graph.GetNodesByFile(file)
	if err != nil {
		m.logger.Printf("❌ get_file_smart: error retrieving file: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("error retrieving file: %v", err)), nil
	}

	// Find the matching symbol
	var node *models.GraphNode
	for i := range nodes {
		if nodes[i].Name == symbol {
			node = &nodes[i]
			break
		}
	}

	if node == nil {
		m.logger.Printf("❌ get_file_smart: symbol not found")
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
		m.logger.Println("❌ trace_impact: invalid arguments")
		return mcp.NewToolResultError("invalid arguments"), nil
	}

	functionID, ok := args["function_id"].(string)
	if !ok || functionID == "" {
		m.logger.Println("❌ trace_impact: missing function_id parameter")
		return mcp.NewToolResultError("function_id parameter is required"), nil
	}

	m.logger.Printf("🎯 trace_impact called: function_id='%s'", functionID)

	// Get the node
	node, err := m.graph.GetNode(functionID)
	if err != nil {
		m.logger.Printf("❌ trace_impact: error getting node: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("error: %v", err)), nil
	}

	if node == nil {
		m.logger.Printf("❌ trace_impact: node not found")
		return mcp.NewToolResultError(fmt.Sprintf("function '%s' not found", functionID)), nil
	}

	// Get direct callers
	directCallers, _ := m.graph.GetCallers(functionID)

	// Get transitive callers (up to 3 levels)
	allCallers := make(map[string]bool)
	for _, caller := range directCallers {
		allCallers[caller] = true
		level2, _ := m.graph.GetCallers(caller)
		for _, l2 := range level2 {
			allCallers[l2] = true
			level3, _ := m.graph.GetCallers(l2)
			for _, l3 := range level3 {
				allCallers[l3] = true
			}
		}
	}

	elapsed := time.Since(start)
	m.logger.Printf("✅ trace_impact: found %d direct callers, %d total affected in %v", len(directCallers), len(allCallers), elapsed)

	result := fmt.Sprintf("## Impact Analysis: %s\n\n", node.Name)
	result += fmt.Sprintf("**Direct callers:** %d\n", len(directCallers))
	result += fmt.Sprintf("**Total affected (up to 3 levels):** %d\n\n", len(allCallers))

	if len(directCallers) > 0 {
		result += "**Direct callers:**\n"
		for _, caller := range directCallers {
			result += fmt.Sprintf("- %s\n", caller)
		}
	}

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

	m.logger.Printf("📋 list_functions called: file='%s', pattern='%s'", file, pattern)

	var nodes []models.GraphNode
	var err error

	if file != "" {
		nodes, err = m.graph.GetNodesByFile(file)
	} else if pattern != "" {
		nodes, err = m.graph.SearchByName(pattern)
	} else {
		nodes, err = m.graph.GetAllNodes(100)
	}

	if err != nil {
		m.logger.Printf("❌ list_functions: error: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("error: %v", err)), nil
	}

	elapsed := time.Since(start)
	
	// Count tokens
	totalTokens := 0
	for _, node := range nodes {
		totalTokens += m.countTokens(node.Name + " " + node.Signature)
	}
	tokensWithoutMCP := len(nodes) * 500 // would need more context per file
	tokenSavings := tokensWithoutMCP - totalTokens
	savingsPercent := 0.0
	if tokensWithoutMCP > 0 {
		savingsPercent = (float64(tokenSavings) / float64(tokensWithoutMCP)) * 100
	}
	
	m.logger.Printf("✅ list_functions: found %d functions in %v | %d tokens (saved ~%d tokens, %.0f%%)", 
		len(nodes), elapsed, totalTokens, tokenSavings, savingsPercent)

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

		output += fmt.Sprintf("### %d. %s (similarity: %.2f%%)\n", i+1, node.Name, r.Similarity*100)
		output += fmt.Sprintf("**File:** %s:%d\n", node.File, node.Line)
		output += fmt.Sprintf("**Type:** %s\n\n", node.Type)

		if node.Signature != "" {
			output += fmt.Sprintf("```\n%s\n```\n\n", node.Signature)
		}

		// Add annotations if they exist
		if len(node.Comments) > 0 || len(node.Tags) > 0 || len(node.CustomMetadata) > 0 {
			output += "**User Annotations:**\n"
			if node.Comments != "" {
				output += fmt.Sprintf("- **Comment:** %s\n", node.Comments)
			}
			if len(node.Tags) > 0 {
				output += fmt.Sprintf("- **Tags:** %v\n", node.Tags)
			}
			if len(node.CustomMetadata) > 0 {
				output += "- **Metadata:**\n"
				for key, value := range node.CustomMetadata {
					output += fmt.Sprintf("  - %s: %s\n", key, value)
				}
			}
			output += "\n"
		}
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

	elapsed := time.Since(start)
	result := fmt.Sprintf("## index_docs complete (%v)\n\n", elapsed)
	result += fmt.Sprintf("- Files indexed: **%d**\n", totalFiles)
	result += fmt.Sprintf("- Chunks created: **%d**\n", totalChunks)
	if len(errors) > 0 {
		result += fmt.Sprintf("\n**Errors (%d):**\n", len(errors))
		for _, e := range errors {
			result += fmt.Sprintf("- %s\n", e)
		}
	}

	m.logger.Printf("✅ index_docs: %d files, %d chunks in %v", totalFiles, totalChunks, elapsed)
	return mcp.NewToolResultText(result), nil
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

// formatNodeWithAnnotations formats a single node with its annotations from the DB
func (m *MCPServer) formatNodeWithAnnotations(id, name, nType, file string, line int, sig, doc string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### %s (%s)\n", name, nType))
	sb.WriteString(fmt.Sprintf("**File:** %s:%d\n", file, line))
	if sig != "" {
		sb.WriteString(fmt.Sprintf("**Signature:** `%s`\n", sig))
	}
	if doc != "" {
		sb.WriteString(fmt.Sprintf("**Doc:** %s\n", doc))
	}

	// Load annotations from DB
	annotations, err := m.graph.GetNodeAnnotations(id)
	if err == nil && len(annotations) > 0 {
		if comment, ok := annotations["comments"].(string); ok && comment != "" {
			sb.WriteString(fmt.Sprintf("**Comment:** %s\n", comment))
		}
		if tags, ok := annotations["tags"].([]string); ok && len(tags) > 0 {
			sb.WriteString(fmt.Sprintf("**Tags:** %v\n", tags))
		}
	}
	return sb.String()
}

// handleUseProfile handles the use_profile tool
func (m *MCPServer) handleUseProfile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments"), nil
	}
	name, _ := args["name"].(string)
	if name == "" {
		return mcp.NewToolResultError("profile name is required"), nil
	}

	// Find profile by name
	var profileID string
	err := m.graph.DB.QueryRow(`SELECT id FROM profiles WHERE name = ?`, name).Scan(&profileID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("profile '%s' not found", name)), nil
	}

	// Get nodes in profile
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
		var id, nName, nType, file string
		var line int
		var sigNull, docNull sql.NullString
		rows.Scan(&id, &nName, &nType, &file, &line, &sigNull, &docNull)

		entry := m.formatNodeWithAnnotations(id, nName, nType, file, line, sigNull.String, docNull.String)
		result.WriteString(entry)
		result.WriteString("\n")
		count++
		totalTokens += m.countTokens(entry)
	}

	if count == 0 {
		return mcp.NewToolResultError(fmt.Sprintf("profile '%s' has no nodes. Add nodes via the UI.", name)), nil
	}

	m.logger.Printf("📦 Profile '%s': %d nodes, ~%d tokens", name, count, totalTokens)
	return mcp.NewToolResultText(result.String()), nil
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
