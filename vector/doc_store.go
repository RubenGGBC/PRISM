package vector

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// DocVectorStore stores and searches embeddings for markdown document chunks
type DocVectorStore struct {
	DB *sql.DB
}

// DocSearchResult is a semantic search result over doc chunks
type DocSearchResult struct {
	ChunkID    string
	File       string
	LineStart  int
	Content    string
	Similarity float32
}

// NewDocVectorStore creates a new DocVectorStore
func NewDocVectorStore(db *sql.DB) *DocVectorStore {
	return &DocVectorStore{DB: db}
}

// InitSchema creates doc_chunks and doc_embeddings tables if they don't exist
func (d *DocVectorStore) InitSchema() error {
	_, err := d.DB.Exec(`
	CREATE TABLE IF NOT EXISTS doc_chunks (
		id          TEXT PRIMARY KEY,
		file        TEXT NOT NULL,
		chunk_index INTEGER NOT NULL,
		line_start  INTEGER NOT NULL,
		content     TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS doc_embeddings (
		chunk_id  TEXT PRIMARY KEY,
		embedding BLOB NOT NULL,
		FOREIGN KEY (chunk_id) REFERENCES doc_chunks(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_doc_chunks_file ON doc_chunks(file);
	`)
	return err
}

// StoreChunk inserts or replaces a doc chunk
func (d *DocVectorStore) StoreChunk(id, file string, chunkIndex, lineStart int, content string) error {
	_, err := d.DB.Exec(
		`INSERT OR REPLACE INTO doc_chunks (id, file, chunk_index, line_start, content) VALUES (?, ?, ?, ?, ?)`,
		id, file, chunkIndex, lineStart, content,
	)
	return err
}

// StoreChunksBatch inserts multiple doc chunks in a transaction
func (d *DocVectorStore) StoreChunksBatch(chunks []chunkRow) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO doc_chunks (id, file, chunk_index, line_start, content) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range chunks {
		if _, err := stmt.Exec(c.id, c.file, c.chunkIndex, c.lineStart, c.content); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// DeleteChunksForFile removes all chunks (and their embeddings via FK cascade) for a file
func (d *DocVectorStore) DeleteChunksForFile(file string) error {
	_, err := d.DB.Exec(`DELETE FROM doc_chunks WHERE file = ?`, file)
	return err
}

// StoreEmbedding saves an embedding for a chunk
func (d *DocVectorStore) StoreEmbedding(chunkID string, embedding []float32) error {
	blob := serializeEmbedding(embedding)
	_, err := d.DB.Exec(`INSERT OR REPLACE INTO doc_embeddings (chunk_id, embedding) VALUES (?, ?)`, chunkID, blob)
	return err
}

// StoreEmbeddingsBatch saves multiple embeddings in a transaction
func (d *DocVectorStore) StoreEmbeddingsBatch(chunkIDs []string, embeddings [][]float32) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO doc_embeddings (chunk_id, embedding) VALUES (?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, id := range chunkIDs {
		if _, err := stmt.Exec(id, serializeEmbedding(embeddings[i])); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Count returns the number of stored doc embeddings
func (d *DocVectorStore) Count() (int, error) {
	var count int
	err := d.DB.QueryRow(`SELECT COUNT(*) FROM doc_embeddings`).Scan(&count)
	return count, err
}

// SearchText embeds the query and returns the top-k most similar doc chunks
func (d *DocVectorStore) SearchText(embedder *Embedder, query string, topK int) ([]DocSearchResult, error) {
	queryEmbed, err := embedder.Embed(query)
	if err != nil {
		return nil, err
	}

	rows, err := d.DB.Query(`
		SELECT de.chunk_id, de.embedding, dc.file, dc.line_start, dc.content
		FROM doc_embeddings de
		JOIN doc_chunks dc ON de.chunk_id = dc.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DocSearchResult
	for rows.Next() {
		var chunkID, file, content string
		var lineStart int
		var blob []byte
		if err := rows.Scan(&chunkID, &blob, &file, &lineStart, &content); err != nil {
			continue
		}
		emb := deserializeEmbedding(blob)
		sim := cosineSimilarity(queryEmbed, emb)
		results = append(results, DocSearchResult{
			ChunkID:    chunkID,
			File:       file,
			LineStart:  lineStart,
			Content:    content,
			Similarity: sim,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if topK > len(results) {
		topK = len(results)
	}
	return results[:topK], nil
}

// SearchKeyword does a basic keyword search over doc_chunks content.
// Used as fallback when Ollama is unavailable or no embeddings exist.
func (d *DocVectorStore) SearchKeyword(query string, topK int) ([]DocSearchResult, error) {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return nil, nil
	}

	rows, err := d.DB.Query(`SELECT id, file, line_start, content FROM doc_chunks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type scored struct {
		DocSearchResult
		score int
	}

	var candidates []scored
	for rows.Next() {
		var r DocSearchResult
		if err := rows.Scan(&r.ChunkID, &r.File, &r.LineStart, &r.Content); err != nil {
			continue
		}
		lower := strings.ToLower(r.Content)
		score := 0
		for _, w := range words {
			score += strings.Count(lower, w)
		}
		if score > 0 {
			candidates = append(candidates, scored{r, score})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	if topK > len(candidates) {
		topK = len(candidates)
	}
	results := make([]DocSearchResult, topK)
	for i := range results {
		// Use score as a rough similarity proxy (capped at 1.0)
		maxScore := float32(len(words) * 5)
		sim := float32(candidates[i].score) / maxScore
		if sim > 1.0 {
			sim = 1.0
		}
		results[i] = candidates[i].DocSearchResult
		results[i].Similarity = sim
	}
	return results, nil
}

// CountChunks returns the total number of stored doc chunks
func (d *DocVectorStore) CountChunks() (int, error) {
	var count int
	err := d.DB.QueryRow(`SELECT COUNT(*) FROM doc_chunks`).Scan(&count)
	return count, err
}

// HasChunks returns true if there are any doc chunks stored
func (d *DocVectorStore) HasChunks() bool {
	n, err := d.CountChunks()
	return err == nil && n > 0
}

// Ensure fmt is used
var _ = fmt.Sprintf

// chunkRow is used internally for batch insertion
type chunkRow struct {
	id         string
	file       string
	chunkIndex int
	lineStart  int
	content    string
}
