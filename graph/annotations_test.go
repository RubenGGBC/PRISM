package graph

import (
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/ruffini/prism/db"
)

func TestAnnotations(t *testing.T) {
	// Initialize schema
	database, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatal("Failed to init DB:", err)
	}
	defer database.Close()

	// Create a test graph
	graph := NewGraph(database)

	// Insert a test node
	nodeID := "test_func"
	graph.DB.Exec(`
		INSERT INTO nodes (id, name, type, file, language, line, end_line)
		VALUES (?, 'test_func', 'function', 'test.py', 'python', 1, 10)
	`, nodeID)

	// Test UpdateNodeComment
	comment := "This is a test comment"
	if err := graph.UpdateNodeComment(nodeID, comment); err != nil {
		t.Errorf("UpdateNodeComment failed: %v", err)
	}

	// Test AddNodeTag
	if err := graph.AddNodeTag(nodeID, "important"); err != nil {
		t.Errorf("AddNodeTag failed: %v", err)
	}
	if err := graph.AddNodeTag(nodeID, "reviewed"); err != nil {
		t.Errorf("AddNodeTag failed: %v", err)
	}

	// Test SetNodeMetadata
	if err := graph.SetNodeMetadata(nodeID, "owner", "john"); err != nil {
		t.Errorf("SetNodeMetadata failed: %v", err)
	}

	// Test GetNodeAnnotations
	annotations, err := graph.GetNodeAnnotations(nodeID)
	if err != nil {
		t.Errorf("GetNodeAnnotations failed: %v", err)
	}

	if annotations["comments"] != comment {
		t.Errorf("Expected comment %q, got %q", comment, annotations["comments"])
	}

	if tags, ok := annotations["tags"].([]string); !ok || len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %v", annotations["tags"])
	}

	if metadata, ok := annotations["custom_metadata"].(map[string]string); !ok || metadata["owner"] != "john" {
		t.Errorf("Expected metadata owner=john, got %v", annotations["custom_metadata"])
	}

	// Test RemoveNodeTag
	if err := graph.RemoveNodeTag(nodeID, "important"); err != nil {
		t.Errorf("RemoveNodeTag failed: %v", err)
	}

	annotations, _ = graph.GetNodeAnnotations(nodeID)
	tags := annotations["tags"].([]string)
	if len(tags) != 1 || tags[0] != "reviewed" {
		t.Errorf("Expected only 'reviewed' tag, got %v", tags)
	}

	t.Logf("All annotation tests passed")
}
