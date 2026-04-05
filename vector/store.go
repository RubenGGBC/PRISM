package vector

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
)

// VectorStore stores embeddings in SQLite
type VectorStore struct {
	DB *sql.DB
}

// NewVectorStore creates a new vector store with the given database
func NewVectorStore(db *sql.DB) *VectorStore {
	return &VectorStore{DB: db}
}

// InitSchema creates the embeddings table if it doesn't exist
func (v *VectorStore) InitSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS embeddings (
		node_id TEXT PRIMARY KEY,
		embedding BLOB NOT NULL,
		FOREIGN KEY (node_id) REFERENCES nodes(id)
	)`
	_, err := v.DB.Exec(query)
	return err
}

// Store saves an embedding for a node
func (v *VectorStore) Store(nodeID string, embedding []float32) error {
	blob := serializeEmbedding(embedding)
	query := `INSERT OR REPLACE INTO embeddings (node_id, embedding) VALUES (?, ?)`
	_, err := v.DB.Exec(query, nodeID, blob)
	return err
}

// StoreBatch stores multiple embeddings efficiently
func (v *VectorStore) StoreBatch(nodeIDs []string, embeddings [][]float32) error {
	if len(nodeIDs) != len(embeddings) {
		return fmt.Errorf("nodeIDs and embeddings must have same length")
	}

	tx, err := v.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO embeddings (node_id, embedding) VALUES (?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, nodeID := range nodeIDs {
		blob := serializeEmbedding(embeddings[i])
		if _, err := stmt.Exec(nodeID, blob); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Get retrieves the embedding for a node
func (v *VectorStore) Get(nodeID string) ([]float32, error) {
	var blob []byte
	query := `SELECT embedding FROM embeddings WHERE node_id = ?`
	err := v.DB.QueryRow(query, nodeID).Scan(&blob)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("embedding not found for node: %s", nodeID)
		}
		return nil, err
	}
	return deserializeEmbedding(blob), nil
}

// GetAll retrieves all embeddings (for brute-force search)
func (v *VectorStore) GetAll() (map[string][]float32, error) {
	query := `SELECT node_id, embedding FROM embeddings`
	rows, err := v.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]float32)
	for rows.Next() {
		var nodeID string
		var blob []byte
		if err := rows.Scan(&nodeID, &blob); err != nil {
			return nil, err
		}
		result[nodeID] = deserializeEmbedding(blob)
	}
	return result, rows.Err()
}

// Count returns the number of stored embeddings
func (v *VectorStore) Count() (int, error) {
	var count int
	err := v.DB.QueryRow(`SELECT COUNT(*) FROM embeddings`).Scan(&count)
	return count, err
}

// Delete removes an embedding
func (v *VectorStore) Delete(nodeID string) error {
	_, err := v.DB.Exec(`DELETE FROM embeddings WHERE node_id = ?`, nodeID)
	return err
}

// serializeEmbedding converts []float32 to []byte
func serializeEmbedding(embedding []float32) []byte {
	buf := make([]byte, len(embedding)*4)
	for i, v := range embedding {
		bits := math.Float32bits(v)
		binary.LittleEndian.PutUint32(buf[i*4:], bits)
	}
	return buf
}

// deserializeEmbedding converts []byte back to []float32
func deserializeEmbedding(blob []byte) []float32 {
	embedding := make([]float32, len(blob)/4)
	for i := range embedding {
		bits := binary.LittleEndian.Uint32(blob[i*4:])
		embedding[i] = math.Float32frombits(bits)
	}
	return embedding
}
