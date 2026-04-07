package server

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/ruffini/prism/api"
	"github.com/ruffini/prism/graph"
	"github.com/ruffini/prism/mcp"
	"github.com/ruffini/prism/vector"
)

// UnifiedServer combines MCP, HTTP API, and static frontend serving
type UnifiedServer struct {
	Database    *sql.DB
	Graph       *graph.CodeGraph
	VectorStore *vector.VectorStore
	Embedder    *vector.Embedder
	Port        int
	MCPEnabled  bool
}

// NewUnifiedServer creates a new unified server instance
func NewUnifiedServer(db *sql.DB, port int, ollamaURL, embedModel string, mcpEnabled bool) *UnifiedServer {
	codeGraph := graph.NewGraph(db)
	vectorStore := vector.NewVectorStore(db)
	embedder := vector.NewEmbedderWithConfig(ollamaURL, embedModel)

	return &UnifiedServer{
		Database:    db,
		Graph:       codeGraph,
		VectorStore: vectorStore,
		Embedder:    embedder,
		Port:        port,
		MCPEnabled:  mcpEnabled,
	}
}

// Start starts both HTTP server and optionally MCP server
func (s *UnifiedServer) Start() error {
	// Setup HTTP server with API and frontend
	mux := http.NewServeMux()

	// Register API routes
	apiServer := api.NewAPIServer(s.Graph, s.VectorStore, s.Database)
	apiServer.RegisterRoutes(mux)

	// Serve embedded frontend
	frontendFS, err := GetFrontendFS()
	if err != nil {
		return fmt.Errorf("failed to get frontend filesystem: %w", err)
	}

	// Serve static files from embedded FS
	fileServer := http.FileServer(http.FS(frontendFS))
	mux.Handle("/", fileServer)

	// Start HTTP server in goroutine
	go func() {
		addr := fmt.Sprintf(":%d", s.Port)
		fmt.Fprintf(os.Stderr, "🌐 PRISM Server running at http://localhost:%d\n", s.Port)
		fmt.Fprintf(os.Stderr, "   📊 Web UI: http://localhost:%d\n", s.Port)
		fmt.Fprintf(os.Stderr, "   🔌 API: http://localhost:%d/api\n", s.Port)
		
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("❌ HTTP server error: %v", err)
		}
	}()

	// Start MCP server if enabled (stdio)
	if s.MCPEnabled {
		mcpServer := mcp.NewMCPServer(s.Graph, s.VectorStore, s.Embedder)
		fmt.Fprintln(os.Stderr, "📡 MCP server ready (stdio transport)")
		
		// This blocks on stdio
		if err := mcpServer.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  MCP connection closed: %v\n", err)
		}
	} else {
		// If MCP is disabled, block forever
		select {}
	}

	return nil
}
