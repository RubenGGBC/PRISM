package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/ruffini/prism/internal/models"
)

// UpdateNodeRequest is the request body for updating node annotations
type UpdateNodeRequest struct {
	Comments       string            `json:"comments"`
	Tags           []string          `json:"tags"`
	CustomMetadata map[string]string `json:"custom_metadata"`
}

// NodeFullResponse includes node data with annotations
type NodeFullResponse struct {
	Node        *models.GraphNode      `json:"node"`
	Annotations map[string]interface{} `json:"annotations"`
}

// HandleUpdateNode updates node annotations (PATCH /api/node/update)
func (a *APIServer) HandleUpdateNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nodeID := r.URL.Query().Get("id")
	if nodeID == "" {
		http.Error(w, "Missing node id parameter", http.StatusBadRequest)
		return
	}

	var req UpdateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("📝 Updating node annotations: %s", nodeID)

	// Update comment
	if req.Comments != "" {
		if err := a.graph.UpdateNodeComment(nodeID, req.Comments); err != nil {
			log.Printf("❌ Failed to update comment: %v", err)
			http.Error(w, fmt.Sprintf("Failed to update comment: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Update tags - first clear existing
	a.db.Exec(`DELETE FROM node_tags WHERE node_id = ?`, nodeID)
	for _, tag := range req.Tags {
		if tag != "" {
			if err := a.graph.AddNodeTag(nodeID, tag); err != nil {
				log.Printf("❌ Failed to add tag: %v", err)
				http.Error(w, fmt.Sprintf("Failed to add tag: %v", err), http.StatusInternalServerError)
				return
			}
		}
	}

	// Update metadata - first clear existing
	a.db.Exec(`DELETE FROM node_metadata WHERE node_id = ?`, nodeID)
	for key, value := range req.CustomMetadata {
		if key != "" {
			if err := a.graph.SetNodeMetadata(nodeID, key, value); err != nil {
				log.Printf("❌ Failed to set metadata: %v", err)
				http.Error(w, fmt.Sprintf("Failed to set metadata: %v", err), http.StatusInternalServerError)
				return
			}
		}
	}

	log.Printf("✅ Updated node: %s", nodeID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"nodeId": nodeID,
	})
}

// HandleGetNodeFull returns node with annotations (GET /api/node/full?id=...)
func (a *APIServer) HandleGetNodeFull(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("id")
	if nodeID == "" {
		http.Error(w, "Missing node id parameter", http.StatusBadRequest)
		return
	}

	log.Printf("📄 Getting node with annotations: %s", nodeID)

	node, annotations, err := a.graph.GetNodeWithAnnotations(nodeID)
	if err != nil {
		log.Printf("❌ Error getting node: %v", err)
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	if node == nil {
		http.Error(w, "Node not found", http.StatusNotFound)
		return
	}

	log.Printf("✅ Found node with %d annotations", len(annotations))

	response := NodeFullResponse{
		Node:        node,
		Annotations: annotations,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
