package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

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
	// Graph edges and impact
	mux.HandleFunc("/api/graph/edges", a.corsMiddleware(a.handleGetCallEdges))
	mux.HandleFunc("/api/node/impact", a.corsMiddleware(a.handleGetNodeImpact))
	// Profiles
	mux.HandleFunc("/api/profiles", a.corsMiddleware(a.handleProfiles))
	mux.HandleFunc("/api/profile/nodes", a.corsMiddleware(a.handleProfileNodes))
	mux.HandleFunc("/api/stats", a.corsMiddleware(a.handleStats))
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

// handleGetCallEdges returns all resolved call edges (GET /api/graph/edges)
func (a *APIServer) handleGetCallEdges(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodOptions {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	rows, err := a.db.Query(`
		SELECT source, target, edge_type
		FROM edges
		WHERE edge_type = 'calls'
		  AND target IN (SELECT id FROM nodes)
		LIMIT 2000
	`)
	if err != nil {
		http.Error(w, fmt.Sprintf("Query error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Edge struct {
		Source string `json:"source"`
		Target string `json:"target"`
		Type   string `json:"type"`
	}

	var edges []Edge
	for rows.Next() {
		var e Edge
		if err := rows.Scan(&e.Source, &e.Target, &e.Type); err != nil {
			continue
		}
		edges = append(edges, e)
	}
	if edges == nil {
		edges = []Edge{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"edges": edges})
}

// handleGetNodeImpact returns nodes affected if this node changes (GET /api/node/impact?id=...)
func (a *APIServer) handleGetNodeImpact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodOptions {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	nodeID := r.URL.Query().Get("id")
	if nodeID == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	// Direct callers (level 1)
	level1, err := a.getDirectCallers(nodeID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Query error: %v", err), http.StatusInternalServerError)
		return
	}

	// Level 2 callers
	level2 := []string{}
	seen := map[string]bool{nodeID: true}
	for _, id := range level1 {
		seen[id] = true
	}
	for _, callerID := range level1 {
		callers, err := a.getDirectCallers(callerID)
		if err != nil {
			continue
		}
		for _, id := range callers {
			if !seen[id] {
				seen[id] = true
				level2 = append(level2, id)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"node_id": nodeID,
		"level1":  level1,
		"level2":  level2,
	})
}

func (a *APIServer) getDirectCallers(nodeID string) ([]string, error) {
	rows, err := a.db.Query(
		`SELECT source FROM edges WHERE target = ? AND edge_type = 'calls'`,
		nodeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var callers []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			callers = append(callers, id)
		}
	}
	return callers, nil
}

// handleProfiles — GET lista perfiles, POST crea uno, DELETE elimina
func (a *APIServer) handleProfiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		rows, err := a.db.Query(`SELECT id, name, description, created_at FROM profiles ORDER BY name`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		type Profile struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			CreatedAt   string `json:"created_at"`
		}
		var profiles []Profile
		for rows.Next() {
			var p Profile
			rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt)
			profiles = append(profiles, p)
		}
		if profiles == nil {
			profiles = []Profile{}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"profiles": profiles})

	case http.MethodPost:
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		id := fmt.Sprintf("profile_%d", time.Now().UnixNano())
		_, err := a.db.Exec(`INSERT INTO profiles (id, name, description) VALUES (?, ?, ?)`,
			id, req.Name, req.Description)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"id": id, "status": "ok"})

	case http.MethodDelete:
		profileID := r.URL.Query().Get("id")
		if profileID == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		a.db.Exec(`DELETE FROM profiles WHERE id = ?`, profileID)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// handleProfileNodes — GET nodos de un perfil, POST añade nodo, DELETE quita nodo
func (a *APIServer) handleProfileNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	profileID := r.URL.Query().Get("profile_id")
	if profileID == "" {
		http.Error(w, "missing profile_id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		rows, err := a.db.Query(`
			SELECT n.id, n.name, n.type, n.file, n.line
			FROM profile_nodes pn
			JOIN nodes n ON n.id = pn.node_id
			WHERE pn.profile_id = ?`, profileID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		type Node struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
			File string `json:"file"`
			Line int    `json:"line"`
		}
		var nodes []Node
		for rows.Next() {
			var n Node
			rows.Scan(&n.ID, &n.Name, &n.Type, &n.File, &n.Line)
			nodes = append(nodes, n)
		}
		if nodes == nil {
			nodes = []Node{}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"nodes": nodes})

	case http.MethodPost:
		var req struct {
			NodeID string `json:"node_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NodeID == "" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		a.db.Exec(`INSERT OR IGNORE INTO profile_nodes (profile_id, node_id) VALUES (?, ?)`,
			profileID, req.NodeID)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	case http.MethodDelete:
		nodeID := r.URL.Query().Get("node_id")
		if nodeID == "" {
			http.Error(w, "missing node_id", http.StatusBadRequest)
			return
		}
		a.db.Exec(`DELETE FROM profile_nodes WHERE profile_id = ? AND node_id = ?`, profileID, nodeID)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// handleStats returns token savings stats for the current MCP session
func (a *APIServer) handleStats(w http.ResponseWriter, r *http.Request) {
	var served, saved int
	row := a.db.QueryRow(`
		SELECT COALESCE(SUM(tokens_served),0), COALESCE(SUM(tokens_saved),0)
		FROM mcp_sessions
		WHERE started_at > datetime('now', '-24 hours')
	`)
	row.Scan(&served, &saved)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tokens_served": served,
		"tokens_saved":  saved,
	})
}
