package graph

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ruffini/prism/internal/models"
)

// GetNodeAnnotations retrieves all annotations for a node
func (g *CodeGraph) GetNodeAnnotations(nodeID string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Get comment
	var comment sql.NullString
	err := g.DB.QueryRow(`SELECT comment FROM node_comments WHERE node_id = ?`, nodeID).Scan(&comment)
	if err == nil && comment.Valid {
		result["comments"] = comment.String
	}

	// Get tags
	rows, err := g.DB.Query(`SELECT tag FROM node_tags WHERE node_id = ? ORDER BY tag`, nodeID)
	if err == nil {
		defer rows.Close()
		var tags []string
		for rows.Next() {
			var tag string
			rows.Scan(&tag)
			tags = append(tags, tag)
		}
		if len(tags) > 0 {
			result["tags"] = tags
		}
	}

	// Get custom metadata
	rows, err = g.DB.Query(`SELECT key, value FROM node_metadata WHERE node_id = ?`, nodeID)
	if err == nil {
		defer rows.Close()
		metadata := make(map[string]string)
		for rows.Next() {
			var key, value string
			rows.Scan(&key, &value)
			metadata[key] = value
		}
		if len(metadata) > 0 {
			result["custom_metadata"] = metadata
		}
	}

	return result, nil
}

// UpdateNodeComment updates or creates comment for a node
func (g *CodeGraph) UpdateNodeComment(nodeID, comment string) error {
	query := `
	INSERT INTO node_comments (node_id, comment, updated_at)
	VALUES (?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(node_id) DO UPDATE SET
		comment = excluded.comment,
		updated_at = CURRENT_TIMESTAMP
	`
	_, err := g.DB.Exec(query, nodeID, comment)
	return err
}

// AddNodeTag adds a tag to a node
func (g *CodeGraph) AddNodeTag(nodeID, tag string) error {
	query := `
	INSERT OR IGNORE INTO node_tags (node_id, tag)
	VALUES (?, ?)
	`
	_, err := g.DB.Exec(query, nodeID, strings.TrimSpace(tag))
	return err
}

// RemoveNodeTag removes a tag from a node
func (g *CodeGraph) RemoveNodeTag(nodeID, tag string) error {
	query := `DELETE FROM node_tags WHERE node_id = ? AND tag = ?`
	_, err := g.DB.Exec(query, nodeID, strings.TrimSpace(tag))
	return err
}

// SetNodeMetadata sets custom key-value metadata on a node
func (g *CodeGraph) SetNodeMetadata(nodeID, key, value string) error {
	query := `
	INSERT INTO node_metadata (node_id, key, value, updated_at)
	VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(node_id, key) DO UPDATE SET
		value = excluded.value,
		updated_at = CURRENT_TIMESTAMP
	`
	_, err := g.DB.Exec(query, nodeID, strings.TrimSpace(key), value)
	return err
}

// RemoveNodeMetadata removes a metadata key from a node
func (g *CodeGraph) RemoveNodeMetadata(nodeID, key string) error {
	query := `DELETE FROM node_metadata WHERE node_id = ? AND key = ?`
	_, err := g.DB.Exec(query, nodeID, strings.TrimSpace(key))
	return err
}

// GetNodeWithAnnotations returns a node with all its annotations
func (g *CodeGraph) GetNodeWithAnnotations(nodeID string) (*models.GraphNode, map[string]interface{}, error) {
	node, err := g.GetNode(nodeID)
	if err != nil {
		return nil, nil, err
	}
	if node == nil {
		return nil, nil, fmt.Errorf("node not found")
	}

	annotations, err := g.GetNodeAnnotations(nodeID)
	if err != nil {
		return node, make(map[string]interface{}), nil // Return node even if annotations fail
	}

	// Populate node fields from annotations
	if comments, ok := annotations["comments"].(string); ok {
		node.Comments = comments
	}
	if tags, ok := annotations["tags"].([]string); ok {
		node.Tags = tags
	}
	if metadata, ok := annotations["custom_metadata"].(map[string]string); ok {
		node.CustomMetadata = metadata
	}
	node.UpdatedAt = time.Now().Format(time.RFC3339)

	return node, annotations, nil
}
