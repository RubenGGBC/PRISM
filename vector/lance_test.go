package vector

import (
	"os"
	"testing"
)

func TestLanceDBStore(t *testing.T) {
	// Cleanup
	os.Remove("test_lance.db")
	defer os.Remove("test_lance.db")

	store, err := NewLanceDBStore("test_lance.db")
	if err != nil {
		t.Fatalf("NewLanceDBStore failed: %v", err)
	}

	// Create test embeddings
	embeddings := map[string][]float32{
		"func1": {0.1, 0.2, 0.3},
		"func2": {0.11, 0.21, 0.31},
	}

	err = store.StoreBatch(embeddings)
	if err != nil {
		t.Fatalf("StoreBatch failed: %v", err)
	}

	// Search for nearest neighbor to func1
	query := []float32{0.1, 0.2, 0.3}
	results, err := store.SearchVector(query, 1)
	if err != nil {
		t.Fatalf("SearchVector failed: %v", err)
	}

	if len(results) == 0 {
		t.Errorf("Expected 1 result, got 0")
	}
}
