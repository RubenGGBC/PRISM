package vector

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// LanceDBStore wraps LanceDB-compatible vector storage using SQLite
// LanceDB is a vector database designed to work alongside SQLite
// This implementation provides LanceDB-like functionality for embeddings
type LanceDBStore struct {
	db *sql.DB
}

// EmbeddingRecord represents a stored embedding
type EmbeddingRecord struct {
	ID        string
	Embedding []float32
}

// NewLanceDBStore creates a new LanceDB-compatible store
func NewLanceDBStore(dbPath string) (*LanceDBStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LanceDB: %w", err)
	}

	// Create embeddings table if it doesn't exist
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS embeddings (
		id TEXT PRIMARY KEY,
		embedding BLOB NOT NULL
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings table: %w", err)
	}

	return &LanceDBStore{
		db: db,
	}, nil
}

// StoreBatch stores multiple embeddings efficiently
func (l *LanceDBStore) StoreBatch(embeddings map[string][]float32) error {
	if len(embeddings) == 0 {
		return nil
	}

	ctx := context.Background()
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO embeddings (id, embedding) VALUES (?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for nodeID, embedding := range embeddings {
		// Serialize embedding as JSON
		embeddingJSON, err := json.Marshal(embedding)
		if err != nil {
			return fmt.Errorf("failed to marshal embedding: %w", err)
		}

		_, err = stmt.ExecContext(ctx, nodeID, embeddingJSON)
		if err != nil {
			return fmt.Errorf("failed to add data: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// SearchVector performs vector similarity search
func (l *LanceDBStore) SearchVector(query []float32, topK int) ([]string, error) {
	ctx := context.Background()

	rows, err := l.db.QueryContext(ctx, `
		SELECT id, embedding FROM embeddings
	`)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	defer rows.Close()

	type result struct {
		id        string
		similarity float32
	}

	var results []result
	for rows.Next() {
		var id string
		var embeddingJSON []byte

		if err := rows.Scan(&id, &embeddingJSON); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		var embedding []float32
		if err := json.Unmarshal(embeddingJSON, &embedding); err != nil {
			return nil, fmt.Errorf("failed to unmarshal embedding: %w", err)
		}

		similarity := cosineSimilarity(query, embedding)
		results = append(results, result{id, similarity})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	// Sort by similarity (descending)
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].similarity > results[i].similarity {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Return top K results
	var nodeIDs []string
	for i := 0; i < len(results) && i < topK; i++ {
		nodeIDs = append(nodeIDs, results[i].id)
	}

	return nodeIDs, nil
}

// Close closes the database connection
func (l *LanceDBStore) Close() error {
	return l.db.Close()
}
