package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/ruffini/prism/graph"
	"github.com/ruffini/prism/vector"
)

// APIServer wraps API handlers
type APIServer struct {
	graph  *graph.CodeGraph
	vector *vector.VectorStore
	ws     *WSServer
	db     *sql.DB
}

// NewAPIServer creates a new API server
func NewAPIServer(g *graph.CodeGraph, v *vector.VectorStore, db *sql.DB) *APIServer {
	return &APIServer{
		graph:  g,
		vector: v,
		ws:     NewWSServer(g),
		db:     db,
	}
}

// RegisterRoutes registers all HTTP routes
func (a *APIServer) RegisterRoutes(mux *http.ServeMux) {
	// Wrap with CORS middleware
	mux.HandleFunc("/api/files", a.corsMiddleware(a.handleGetFiles))
	mux.HandleFunc("/api/file/nodes", a.corsMiddleware(a.handleGetFileNodes))
	mux.HandleFunc("/api/node", a.corsMiddleware(a.handleGetNode))
	mux.HandleFunc("/api/search", a.corsMiddleware(a.handleSearch))
	mux.HandleFunc("/api/node/update", a.corsMiddleware(a.HandleUpdateNode))
	mux.HandleFunc("/api/node/full", a.corsMiddleware(a.HandleGetNodeFull))
	mux.HandleFunc("/ws", a.corsMiddleware(a.ws.HandleWSConnection))
	// Health check
	mux.HandleFunc("/api/health", a.corsMiddleware(a.handleHealth))
}

// corsMiddleware adds CORS headers
func (a *APIServer) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// handleHealth returns server status
func (a *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// handleGetFiles returns all files in the graph
func (a *APIServer) handleGetFiles(w http.ResponseWriter, r *http.Request) {
	log.Println("📄 GET /api/files")

	if a.graph == nil {
		log.Println("❌ Graph is nil")
		http.Error(w, "graph not initialized", http.StatusInternalServerError)
		return
	}

	// Get distinct files from nodes
	files, err := a.graph.GetDistinctFiles()
	if err != nil {
		log.Printf("❌ Error getting files: %v", err)
		http.Error(w, fmt.Sprintf("error: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Found %d files", len(files))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"files": files,
	})
}

// handleGetNode returns a specific node
func (a *APIServer) handleGetNode(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("id")
	log.Printf("📄 GET /api/node?id=%s", nodeID)

	node, err := a.graph.GetNode(nodeID)
	if err != nil {
		log.Printf("❌ Error getting node: %v", err)
		http.Error(w, fmt.Sprintf("error: %v", err), http.StatusInternalServerError)
		return
	}
	if node == nil {
		log.Println("⚠️  Node not found")
		http.Error(w, "node not found", http.StatusNotFound)
		return
	}

	log.Printf("✅ Found node: %s", node.Name)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(node)
}

// handleSearch performs semantic search
func (a *APIServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	log.Printf("🔍 GET /api/search?q=%s", query)

	nodes, err := a.graph.SearchByName(query)
	if err != nil {
		log.Printf("❌ Error searching: %v", err)
		http.Error(w, fmt.Sprintf("error: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Found %d results", len(nodes))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": nodes,
	})
}

// handleGetFileNodes returns all nodes (functions, classes) in a file
func (a *APIServer) handleGetFileNodes(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("file")
	if filePath == "" {
		http.Error(w, "Missing file parameter", http.StatusBadRequest)
		return
	}

	log.Printf("📂 GET /api/file/nodes?file=%s", filePath)

	nodes, err := a.graph.GetNodesByFile(filePath)
	if err != nil {
		log.Printf("❌ Error getting file nodes: %v", err)
		http.Error(w, fmt.Sprintf("error: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Found %d nodes in file %s", len(nodes), filePath)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes": nodes,
	})
}
