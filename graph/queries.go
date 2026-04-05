package graph

import (
	"database/sql"
	"fmt"

	"github.com/ruffini/prism/internal/models"
)

// GetNode retrieves a node by ID
func (g *CodeGraph) GetNode(id string) (*models.GraphNode, error) {
	query := `
		SELECT id, name, type, file, line, end_line, signature, body, pagerank, blast_radius
		FROM nodes WHERE id = ?
	`

	var node models.GraphNode
	var signature, body sql.NullString

	err := g.DB.QueryRow(query, id).Scan(
		&node.ID,
		&node.Name,
		&node.Type,
		&node.File,
		&node.Line,
		&node.EndLine,
		&signature,
		&body,
		&node.PageRank,
		&node.BlastRadius,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	node.Signature = signature.String
	node.Body = body.String

	return &node, nil
}

// GetNodesByFile retrieves all nodes in a file
func (g *CodeGraph) GetNodesByFile(file string) ([]models.GraphNode, error) {
	query := `
		SELECT id, name, type, file, line, end_line, signature, body, pagerank, blast_radius
		FROM nodes WHERE file = ?
		ORDER BY line
	`

	rows, err := g.DB.Query(query, file)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}
	defer rows.Close()

	var nodes []models.GraphNode
	for rows.Next() {
		var node models.GraphNode
		var signature, body sql.NullString

		err := rows.Scan(
			&node.ID,
			&node.Name,
			&node.Type,
			&node.File,
			&node.Line,
			&node.EndLine,
			&signature,
			&body,
			&node.PageRank,
			&node.BlastRadius,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}

		node.Signature = signature.String
		node.Body = body.String
		nodes = append(nodes, node)
	}

	return nodes, rows.Err()
}

// GetCallers retrieves functions that call the given node
func (g *CodeGraph) GetCallers(nodeID string) ([]string, error) {
	query := `
		SELECT source FROM edges 
		WHERE target = ? AND edge_type = 'calls'
	`

	rows, err := g.DB.Query(query, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query callers: %w", err)
	}
	defer rows.Close()

	var callers []string
	for rows.Next() {
		var caller string
		if err := rows.Scan(&caller); err != nil {
			return nil, fmt.Errorf("failed to scan caller: %w", err)
		}
		callers = append(callers, caller)
	}

	return callers, rows.Err()
}

// GetCallees retrieves functions that the given node calls
func (g *CodeGraph) GetCallees(nodeID string) ([]string, error) {
	query := `
		SELECT target FROM edges 
		WHERE source = ? AND edge_type = 'calls'
	`

	rows, err := g.DB.Query(query, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query callees: %w", err)
	}
	defer rows.Close()

	var callees []string
	for rows.Next() {
		var callee string
		if err := rows.Scan(&callee); err != nil {
			return nil, fmt.Errorf("failed to scan callee: %w", err)
		}
		callees = append(callees, callee)
	}

	return callees, rows.Err()
}

// SearchByName searches nodes by name using LIKE query
func (g *CodeGraph) SearchByName(query string) ([]models.GraphNode, error) {
	sqlQuery := `
		SELECT id, name, type, file, line, end_line, signature, body, pagerank, blast_radius
		FROM nodes WHERE name LIKE ?
		ORDER BY pagerank DESC, name
		LIMIT 100
	`

	searchPattern := "%" + query + "%"
	rows, err := g.DB.Query(sqlQuery, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search nodes: %w", err)
	}
	defer rows.Close()

	var nodes []models.GraphNode
	for rows.Next() {
		var node models.GraphNode
		var signature, body sql.NullString

		err := rows.Scan(
			&node.ID,
			&node.Name,
			&node.Type,
			&node.File,
			&node.Line,
			&node.EndLine,
			&signature,
			&body,
			&node.PageRank,
			&node.BlastRadius,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}

		node.Signature = signature.String
		node.Body = body.String
		nodes = append(nodes, node)
	}

	return nodes, rows.Err()
}

// GetImports retrieves all modules imported by a node
func (g *CodeGraph) GetImports(nodeID string) ([]string, error) {
	query := `
		SELECT target FROM edges 
		WHERE source = ? AND edge_type = 'imports'
	`

	rows, err := g.DB.Query(query, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query imports: %w", err)
	}
	defer rows.Close()

	var imports []string
	for rows.Next() {
		var imp string
		if err := rows.Scan(&imp); err != nil {
			return nil, fmt.Errorf("failed to scan import: %w", err)
		}
		imports = append(imports, imp)
	}

	return imports, rows.Err()
}

// GetAllNodes retrieves all nodes (with optional limit)
func (g *CodeGraph) GetAllNodes(limit int) ([]models.GraphNode, error) {
	query := `
		SELECT id, name, type, file, line, end_line, signature, body, pagerank, blast_radius
		FROM nodes
		ORDER BY file, line
		LIMIT ?
	`

	rows, err := g.DB.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query all nodes: %w", err)
	}
	defer rows.Close()

	var nodes []models.GraphNode
	for rows.Next() {
		var node models.GraphNode
		var signature, body sql.NullString

		err := rows.Scan(
			&node.ID,
			&node.Name,
			&node.Type,
			&node.File,
			&node.Line,
			&node.EndLine,
			&signature,
			&body,
			&node.PageRank,
			&node.BlastRadius,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}

		node.Signature = signature.String
		node.Body = body.String
		nodes = append(nodes, node)
	}

	return nodes, rows.Err()
}

// GetDistinctFiles returns all unique files in the graph
func (g *CodeGraph) GetDistinctFiles() ([]string, error) {
	query := `SELECT DISTINCT file FROM nodes ORDER BY file`

	rows, err := g.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query distinct files: %w", err)
	}
	defer rows.Close()

	var files []string
	for rows.Next() {
		var file string
		if err := rows.Scan(&file); err != nil {
			continue
		}
		files = append(files, file)
	}

	return files, rows.Err()
}

// GetNodeMetadata retrieves metadata for a node
func (g *CodeGraph) GetNodeMetadata(nodeID string) (map[string]interface{}, error) {
	var (
		deprecated string
		hooks      sql.NullString
		todos      sql.NullString
		authors    sql.NullString
	)

	err := g.DB.QueryRow(`
		SELECT COALESCE(deprecated, 'false'), COALESCE(hooks, ''), COALESCE(todos, ''), COALESCE(authors, '')
		FROM metadata
		WHERE function_id = ?
	`, nodeID).Scan(&deprecated, &hooks, &todos, &authors)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	result := make(map[string]interface{})
	if deprecated == "true" || deprecated == "1" {
		result["deprecated"] = true
	}
	if hooks.Valid && hooks.String != "" {
		result["hooks"] = hooks.String
	}
	if todos.Valid && todos.String != "" {
		result["todos"] = todos.String
	}
	if authors.Valid && authors.String != "" {
		result["authors"] = authors.String
	}

	return result, nil
}
