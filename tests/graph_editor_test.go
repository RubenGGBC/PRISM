package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
	"github.com/ruffini/prism/api"
	"github.com/ruffini/prism/db"
	"github.com/ruffini/prism/graph"
)

func TestGraphEditorIntegration(t *testing.T) {
	// Initialize database with schema
	database, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}
	defer database.Close()

	// Create graph
	codeGraph := graph.NewGraph(database)

	// Insert test node
	nodeID := "test_combat_service"
	_, err = database.Exec(`
		INSERT INTO nodes (id, name, type, file, language, line, end_line, signature)
		VALUES (?, 'CombatService', 'class', 'combat.py', 'python', 10, 50, 'class CombatService:')
	`, nodeID)
	if err != nil {
		t.Fatalf("Failed to insert test node: %v", err)
	}

	// Create API server
	apiServer := api.NewAPIServer(codeGraph, nil, database)

	t.Run("Update node annotations", func(t *testing.T) {
		updateData := api.UpdateNodeRequest{
			Comments: "Main combat handler",
			Tags:     []string{"critical", "core"},
			CustomMetadata: map[string]string{
				"owner":  "john",
				"status": "stable",
			},
		}

		body, _ := json.Marshal(updateData)
		req := httptest.NewRequest("PATCH", "/api/node/update?id="+nodeID, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Call handler directly
		apiServer.HandleUpdateNode(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify in database
		annotations, _ := codeGraph.GetNodeAnnotations(nodeID)
		if comment, ok := annotations["comments"].(string); !ok || comment != "Main combat handler" {
			t.Errorf("Comment not saved correctly: %v", annotations["comments"])
		}

		if tags, ok := annotations["tags"].([]string); !ok || len(tags) != 2 {
			t.Errorf("Tags not saved correctly: %v", annotations["tags"])
		}

		t.Logf("Annotation update test passed")
	})

	t.Run("Get node with annotations", func(t *testing.T) {
		// First update node with annotations
		codeGraph.UpdateNodeComment(nodeID, "Test comment")
		codeGraph.AddNodeTag(nodeID, "important")
		codeGraph.SetNodeMetadata(nodeID, "reviewer", "alice")

		// Get node
		req := httptest.NewRequest("GET", "/api/node/full?id="+nodeID, nil)
		w := httptest.NewRecorder()

		apiServer.HandleGetNodeFull(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
			return
		}

		var response api.NodeFullResponse
		json.NewDecoder(w.Body).Decode(&response)

		if response.Node.Comments != "Test comment" {
			t.Errorf("Expected comment 'Test comment', got '%s'", response.Node.Comments)
		}

		if len(response.Node.Tags) < 1 {
			t.Errorf("Expected at least 1 tag, got %v", response.Node.Tags)
		}

		if response.Node.CustomMetadata["reviewer"] != "alice" {
			t.Errorf("Expected metadata reviewer=alice, got %v", response.Node.CustomMetadata)
		}

		t.Logf("Get node with annotations test passed")
	})

	t.Run("Remove annotations", func(t *testing.T) {
		// Clear all existing tags first
		database.Exec(`DELETE FROM node_tags WHERE node_id = ?`, nodeID)

		// Add multiple tags
		codeGraph.AddNodeTag(nodeID, "remove_test_tag1")
		codeGraph.AddNodeTag(nodeID, "remove_test_tag2")
		codeGraph.AddNodeTag(nodeID, "remove_test_tag3")

		// Remove one tag
		codeGraph.RemoveNodeTag(nodeID, "remove_test_tag2")

		// Verify only 2 tags remain
		annotations, _ := codeGraph.GetNodeAnnotations(nodeID)
		if tags, ok := annotations["tags"].([]string); ok {
			if len(tags) != 2 {
				t.Errorf("Expected 2 tags after removal, got %d: %v", len(tags), tags)
			}
			// Verify tag2 is gone
			for _, tag := range tags {
				if tag == "remove_test_tag2" {
					t.Errorf("Tag 'remove_test_tag2' should have been removed")
				}
			}
		}

		t.Logf("Remove annotations test passed")
	})

	t.Run("Multiple nodes with different annotations", func(t *testing.T) {
		node2ID := "test_strategy"
		_, err := database.Exec(`
			INSERT INTO nodes (id, name, type, file, language, line, end_line)
			VALUES (?, 'StrategyEngine', 'class', 'strategy.py', 'python', 1, 100)
		`, node2ID)
		if err != nil {
			t.Fatalf("Failed to insert second node: %v", err)
		}

		// Annotate both nodes differently
		codeGraph.UpdateNodeComment(nodeID, "Combat system")
		codeGraph.AddNodeTag(nodeID, "battle")

		codeGraph.UpdateNodeComment(node2ID, "AI strategy")
		codeGraph.AddNodeTag(node2ID, "ai")

		// Verify each node has correct annotations
		ann1, _ := codeGraph.GetNodeAnnotations(nodeID)
		ann2, _ := codeGraph.GetNodeAnnotations(node2ID)

		if ann1["comments"].(string) != "Combat system" {
			t.Errorf("Node 1 comment mismatch")
		}
		if ann2["comments"].(string) != "AI strategy" {
			t.Errorf("Node 2 comment mismatch")
		}

		if tags1, ok := ann1["tags"].([]string); !ok || len(tags1) == 0 || tags1[0] != "battle" {
			t.Errorf("Node 1 tags mismatch: %v", ann1["tags"])
		}
		if tags2, ok := ann2["tags"].([]string); !ok || len(tags2) == 0 || tags2[0] != "ai" {
			t.Errorf("Node 2 tags mismatch: %v", ann2["tags"])
		}

		t.Logf("Multiple nodes test passed")
	})
}
