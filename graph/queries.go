package graph

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ruffini/prism/internal/models"
)

func scanNodes(rows *sql.Rows) ([]models.GraphNode, error) {
	defer rows.Close()
	var nodes []models.GraphNode
	for rows.Next() {
		var node models.GraphNode
		var signature, body sql.NullString
		if err := rows.Scan(
			&node.ID, &node.Name, &node.Type, &node.File,
			&node.Line, &node.EndLine, &signature, &body,
			&node.PageRank, &node.BlastRadius,
		); err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}
		node.Signature = signature.String
		node.Body = body.String
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}

// GetNode retrieves a node by ID
func (g *CodeGraph) GetNode(id string) (*models.GraphNode, error) {
	id = strings.ReplaceAll(id, "\\", "/")

	query := `SELECT id, name, type, file, line, end_line, signature, body, pagerank, blast_radius FROM nodes WHERE id = ?`
	var node models.GraphNode
	var signature, body sql.NullString

	err := g.DB.QueryRow(query, id).Scan(
		&node.ID, &node.Name, &node.Type, &node.File,
		&node.Line, &node.EndLine, &signature, &body,
		&node.PageRank, &node.BlastRadius,
	)
	if err == nil {
		node.Signature = signature.String
		node.Body = body.String
		return &node, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	// Fuzzy fallback: match by suffix (handles partial paths)
	fuzzyQuery := `SELECT id, name, type, file, line, end_line, signature, body, pagerank, blast_radius FROM nodes WHERE id LIKE ? ORDER BY length(id) ASC LIMIT 1`
	err = g.DB.QueryRow(fuzzyQuery, "%"+id).Scan(
		&node.ID, &node.Name, &node.Type, &node.File,
		&node.Line, &node.EndLine, &signature, &body,
		&node.PageRank, &node.BlastRadius,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get node (fuzzy): %w", err)
	}
	node.Signature = signature.String
	node.Body = body.String
	return &node, nil
}

// GetNodesByFile retrieves all nodes in a file
func (g *CodeGraph) GetNodesByFile(file string) ([]models.GraphNode, error) {
	file = strings.ReplaceAll(file, "\\", "/")

	rows, err := g.DB.Query(`SELECT id, name, type, file, line, end_line, signature, body, pagerank, blast_radius FROM nodes WHERE file = ? ORDER BY line`, file)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}
	nodes, err := scanNodes(rows)
	if err != nil {
		return nil, err
	}
	if len(nodes) > 0 {
		return nodes, nil
	}

	// Fuzzy fallback: suffix match
	rows2, err := g.DB.Query(`SELECT id, name, type, file, line, end_line, signature, body, pagerank, blast_radius FROM nodes WHERE file LIKE ? ORDER BY line`, "%"+file)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes (fuzzy): %w", err)
	}
	return scanNodes(rows2)
}

// GetNodesByFileFlexible retrieves nodes using normalized path matching.
// It supports slash/backslash differences and absolute-vs-relative paths.
func (g *CodeGraph) GetNodesByFileFlexible(file string) ([]models.GraphNode, error) {
	file = strings.TrimSpace(file)
	if file == "" {
		return []models.GraphNode{}, nil
	}

	// Fast path: exact matches with common normalized variants.
	candidates := []string{
		file,
		filepath.Clean(file),
		strings.ReplaceAll(file, "/", `\`),
		strings.ReplaceAll(file, `\`, "/"),
	}
	seen := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true

		nodes, err := g.GetNodesByFile(candidate)
		if err != nil {
			return nil, err
		}
		if len(nodes) > 0 {
			return nodes, nil
		}
	}

	normalized := normalizeLookupPath(file)
	if normalized == "" {
		return []models.GraphNode{}, nil
	}

	// Slow path: suffix match on normalized paths.
	rows, err := g.DB.Query(`
		SELECT id, name, type, file, line, end_line, signature, body, pagerank, blast_radius
		FROM nodes
		WHERE LOWER(REPLACE(file, '\', '/')) = LOWER(?)
		   OR LOWER(REPLACE(file, '\', '/')) LIKE LOWER(?)
		ORDER BY
			CASE WHEN LOWER(REPLACE(file, '\', '/')) = LOWER(?) THEN 0 ELSE 1 END,
			LENGTH(file),
			file,
			line
	`, normalized, "%"+normalized, normalized)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes by normalized file: %w", err)
	}
	defer rows.Close()

	var (
		nodes    []models.GraphNode
		bestFile string
	)
	for rows.Next() {
		var node models.GraphNode
		var signature, body sql.NullString

		if err := rows.Scan(
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
		); err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}

		// Keep only the best-matching file (first in ordered result set).
		if bestFile == "" {
			bestFile = node.File
		}
		if node.File != bestFile {
			break
		}

		node.Signature = signature.String
		node.Body = body.String
		nodes = append(nodes, node)
	}

	return nodes, rows.Err()
}

func normalizeLookupPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	// On Windows, callers may provide absolute paths. Convert to cwd-relative
	// when possible so it can match indexed relative file paths.
	if filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err == nil {
			if cwd, cwdErr := os.Getwd(); cwdErr == nil {
				if rel, relErr := filepath.Rel(cwd, abs); relErr == nil && rel != "." && !strings.HasPrefix(rel, "..") {
					path = rel
				} else {
					path = abs
				}
			}
		}
	}

	path = filepath.ToSlash(filepath.Clean(path))
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimPrefix(path, "/")

	// Strip Windows drive prefix for suffix matching (e.g. C:/repo/file.ts -> repo/file.ts).
	if len(path) >= 3 && path[1] == ':' && path[2] == '/' {
		path = path[3:]
	}

	return path
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

// GetAllNodesPaginated retrieves nodes with pagination and optional type filter
func (g *CodeGraph) GetAllNodesPaginated(limit, offset int, nodeType string) ([]models.GraphNode, error) {
	var rows *sql.Rows
	var err error
	if nodeType != "" {
		rows, err = g.DB.Query(`SELECT id, name, type, file, line, end_line, signature, body, pagerank, blast_radius FROM nodes WHERE type = ? ORDER BY file, line LIMIT ? OFFSET ?`, nodeType, limit, offset)
	} else {
		rows, err = g.DB.Query(`SELECT id, name, type, file, line, end_line, signature, body, pagerank, blast_radius FROM nodes ORDER BY file, line LIMIT ? OFFSET ?`, limit, offset)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}
	return scanNodes(rows)
}

// GetCalleesTransitive returns all transitive callees up to maxDepth levels deep
func (g *CodeGraph) GetCalleesTransitive(nodeID string, maxDepth int) (map[string]bool, error) {
	nodeID = strings.ReplaceAll(nodeID, "\\", "/")
	visited := make(map[string]bool)
	queue := []string{nodeID}

	for depth := 0; depth < maxDepth && len(queue) > 0; depth++ {
		next := []string{}
		for _, id := range queue {
			callees, err := g.GetCallees(id)
			if err != nil {
				return nil, err
			}
			for _, c := range callees {
				if !visited[c] {
					visited[c] = true
					next = append(next, c)
				}
			}
		}
		queue = next
	}
	return visited, nil
}

// GetDistinctDirectories returns top-level directory prefixes found in node file paths
func (g *CodeGraph) GetDistinctDirectories() ([]string, error) {
	rows, err := g.DB.Query(`SELECT DISTINCT file FROM nodes`)
	if err != nil {
		return nil, fmt.Errorf("failed to query files: %w", err)
	}
	defer rows.Close()

	dirs := make(map[string]bool)
	for rows.Next() {
		var file string
		if err := rows.Scan(&file); err != nil {
			continue
		}
		parts := strings.SplitN(file, "/", 2)
		if len(parts) > 1 {
			dirs[parts[0]] = true
		}
	}

	result := make([]string, 0, len(dirs))
	for d := range dirs {
		result = append(result, d)
	}
	sort.Strings(result)
	return result, rows.Err()
}

// GetNodesByDirectoryPrefix retrieves nodes whose file path starts with the given prefix
func (g *CodeGraph) GetNodesByDirectoryPrefix(prefix string, limit int) ([]models.GraphNode, error) {
	rows, err := g.DB.Query(`SELECT id, name, type, file, line, end_line, signature, body, pagerank, blast_radius FROM nodes WHERE file LIKE ? ORDER BY pagerank DESC LIMIT ?`, prefix+"/%", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes by dir: %w", err)
	}
	return scanNodes(rows)
}

// ListProfileNames returns all profile names sorted alphabetically
func (g *CodeGraph) ListProfileNames() ([]string, error) {
	rows, err := g.DB.Query(`SELECT name FROM profiles ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to list profiles: %w", err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		names = append(names, name)
	}
	return names, rows.Err()
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
