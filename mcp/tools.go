package mcp

// This file contains the tool definitions for the MCP server.
// The actual tool handlers are implemented in server.go.
//
// Available Tools:
//
// 1. search_context
//    - Searches for relevant functions/code based on a natural language query
//    - Uses semantic search with embeddings when available, falls back to keyword search
//    - Parameters:
//      - query (string, required): The search query in natural language
//      - limit (number, optional): Maximum results to return (default: 5)
//
// 2. get_file_smart
//    - Gets a specific function or class from a file without loading the entire file
//    - Returns the code along with its callers, callees, and metadata
//    - Parameters:
//      - file (string, required): The file path containing the symbol
//      - symbol (string, required): The function or class name to retrieve
//
// 3. trace_impact
//    - Shows the blast radius - what functions would be affected by changing this one
//    - Returns direct callers and transitive callers (up to 3 levels deep)
//    - Parameters:
//      - function_id (string, required): The function ID (format: file.py:function_name)
//
// 4. list_functions
//    - Lists all functions in a file or matching a search pattern
//    - Parameters:
//      - file (string, optional): Filter by file path
//      - pattern (string, optional): Filter by name pattern (substring match)

// ToolInfo contains metadata about a tool
type ToolInfo struct {
	Name        string
	Description string
	Parameters  []ToolParameter
}

// ToolParameter describes a tool parameter
type ToolParameter struct {
	Name        string
	Type        string
	Required    bool
	Description string
}

// GetToolDefinitions returns metadata about all available tools
func GetToolDefinitions() []ToolInfo {
	return []ToolInfo{
		{
			Name:        "search_context",
			Description: "Search for relevant functions/code based on a natural language query. Returns the most relevant code snippets using semantic search.",
			Parameters: []ToolParameter{
				{Name: "query", Type: "string", Required: true, Description: "The search query in natural language"},
				{Name: "limit", Type: "number", Required: false, Description: "Maximum number of results to return (default: 5)"},
			},
		},
		{
			Name:        "get_file_smart",
			Description: "Get a specific function or class from a file without loading unnecessary code. Returns the code along with its callers, callees, and metadata.",
			Parameters: []ToolParameter{
				{Name: "file", Type: "string", Required: true, Description: "The file path containing the symbol"},
				{Name: "symbol", Type: "string", Required: true, Description: "The function or class name to retrieve"},
			},
		},
		{
			Name:        "trace_impact",
			Description: "Show the blast radius - what functions would be affected if you change this one. Returns direct callers and transitive callers.",
			Parameters: []ToolParameter{
				{Name: "function_id", Type: "string", Required: true, Description: "The function ID to analyze (format: file.py:function_name)"},
			},
		},
		{
			Name:        "list_functions",
			Description: "List all functions in a file or matching a search pattern.",
			Parameters: []ToolParameter{
				{Name: "file", Type: "string", Required: false, Description: "Optional: filter by file path"},
				{Name: "pattern", Type: "string", Required: false, Description: "Optional: filter by name pattern (substring match)"},
			},
		},
		{
			Name:        "index_docs",
			Description: "Index project markdown files into the vector store for semantic search via search_docs.",
			Parameters: []ToolParameter{
				{Name: "path", Type: "string", Required: false, Description: "Directory to scan for .md files. Defaults to ./docs, ./.github, and root *.md files"},
			},
		},
		{
			Name:        "search_docs",
			Description: "Semantic search over indexed markdown documentation. Run index_docs first.",
			Parameters: []ToolParameter{
				{Name: "query", Type: "string", Required: true, Description: "Natural language search query"},
				{Name: "limit", Type: "number", Required: false, Description: "Maximum number of results to return (default: 5)"},
			},
		},
	}
}
