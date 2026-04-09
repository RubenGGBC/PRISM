package vector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Embedder generates vector embeddings using Ollama API
type Embedder struct {
	BaseURL string
	Model   string
}

// embedRequest is the request body for Ollama embed API
type embedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// embedResponse is the response from Ollama embed API
type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// NewEmbedder creates a new embedder with default settings
// Default: localhost:11434, nomic-embed-text (768 dimensions)
func NewEmbedder() *Embedder {
	return &Embedder{
		BaseURL: "http://localhost:11434",
		Model:   "nomic-embed-text",
	}
}

// NewEmbedderWithConfig creates an embedder with custom settings
func NewEmbedderWithConfig(baseURL, model string) *Embedder {
	return &Embedder{
		BaseURL: baseURL,
		Model:   model,
	}
}

// Embed generates embedding for a single text
func (e *Embedder) Embed(text string) ([]float32, error) {
	embeddings, err := e.EmbedBatch([]string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	return embeddings[0], nil
}

// EmbedBatch generates embeddings for multiple texts (more efficient)
func (e *Embedder) EmbedBatch(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	req := embedRequest{
		Model: e.Model,
		Input: texts,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/embed", e.BaseURL)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Embeddings, nil
}

// BuildEmbedText builds a richer text representation of a code element for embedding.
// Includes type, name, signature, docstring excerpt, and body context for better semantic search.
func BuildEmbedText(name, nodeType, signature, docstring, body string) string {
	parts := []string{}
	if nodeType != "" {
		parts = append(parts, nodeType)
	}
	if name != "" {
		parts = append(parts, name)
	}
	if signature != "" {
		parts = append(parts, signature)
	}
	if docstring != "" {
		doc := docstring
		if len(doc) > 200 {
			doc = doc[:200]
		}
		parts = append(parts, doc)
	}
	if body != "" {
		lines := strings.SplitN(body, "\n", 4)
		if len(lines) > 3 {
			lines = lines[:3]
		}
		parts = append(parts, strings.Join(lines, " "))
	}
	return strings.Join(parts, " | ")
}

// Dimension returns the expected embedding dimension for the model
func (e *Embedder) Dimension() int {
	// nomic-embed-text produces 768-dimensional embeddings
	if e.Model == "nomic-embed-text" {
		return 768
	}
	return 768 // default
}
