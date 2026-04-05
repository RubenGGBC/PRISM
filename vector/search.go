package vector

import (
	"math"
	"sort"
)

// SearchResult represents a search result with similarity score
type SearchResult struct {
	NodeID     string
	Similarity float32
}

// Search finds top-k most similar nodes to the query embedding
func (v *VectorStore) Search(queryEmbedding []float32, topK int) ([]SearchResult, error) {
	allEmbeddings, err := v.GetAll()
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(allEmbeddings))
	for nodeID, embedding := range allEmbeddings {
		sim := cosineSimilarity(queryEmbedding, embedding)
		results = append(results, SearchResult{
			NodeID:     nodeID,
			Similarity: sim,
		})
	}

	// Sort by similarity descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	// Return top-k
	if topK > len(results) {
		topK = len(results)
	}
	return results[:topK], nil
}

// SearchText embeds the query text and performs search
func (v *VectorStore) SearchText(embedder *Embedder, query string, topK int) ([]SearchResult, error) {
	queryEmbedding, err := embedder.Embed(query)
	if err != nil {
		return nil, err
	}
	return v.Search(queryEmbedding, topK)
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float32
	var normA float32
	var normB float32

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// SearchWithThreshold finds nodes above a similarity threshold
func (v *VectorStore) SearchWithThreshold(queryEmbedding []float32, threshold float32) ([]SearchResult, error) {
	allEmbeddings, err := v.GetAll()
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	for nodeID, embedding := range allEmbeddings {
		sim := cosineSimilarity(queryEmbedding, embedding)
		if sim >= threshold {
			results = append(results, SearchResult{
				NodeID:     nodeID,
				Similarity: sim,
			})
		}
	}

	// Sort by similarity descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	return results, nil
}
