# PRISM Platform - Plan Semanal Detallado

**3-4 semanas, 5 días/semana = 15-20 días de trabajo**

---

## 📅 SEMANA 1: Backend Foundation (Parser + Graph)

### 🔴 LUNES: Setup + Tree-sitter Parser Básico

#### Mañana: Setup Inicial (1-2 horas)

```bash
# 1. Crear proyecto
mkdir prism-platform
cd prism-platform
git init
go mod init github.com/ruffini/prism-platform

# 2. Instalar dependencias
go get github.com/smacker/go-tree-sitter
go get github.com/smacker/go-tree-sitter/javascript
go get github.com/smacker/go-tree-sitter/python
go get github.com/smacker/go-tree-sitter/go
go get modernc.org/sqlite
go get github.com/lancedb/lancedb

# 3. Estructura inicial
mkdir -p parser graph vector metadata api mcp cmd tests docs
touch main.go Makefile .gitignore README.md
```

**Archivo: `.gitignore`**
```
code_graph.db
code_vectors.lance
.env
*.log
dist/
node_modules/
frontend/dist
frontend/node_modules
```

**Archivo: `Makefile`**
```makefile
.PHONY: build run test clean setup

setup:
	go mod download
	go get github.com/cosmtrek/air  # Hot reload

build:
	go build -o cip ./cmd/main.go

run:
	go run ./cmd/main.go serve

dev:
	air

test:
	go test ./...

clean:
	rm -f cip code_graph.db code_vectors.lance
	find . -name "*.out" -delete

parser-test:
	go test -v ./parser -run TestParseFile
```

#### Tarde: Primeros Tests (2-3 horas)

**Archivo: `parser/ast.go`** (básico, solo tipos)

```go
package parser

import (
	"os"
	"path/filepath"
	"strings"
)

// CodeElement representa una función, clase o método extraído
type CodeElement struct {
	ID          string   // "file.ts:functionName"
	Name        string   // "functionName"
	Type        string   // "function", "class", "method"
	File        string   // "auth/login.ts"
	Line        int      // número de línea
	EndLine     int
	Signature   string   // "function login(user, pass)"
	Body        string   // primeras 500 chars del cuerpo
	StartByte   uint32
	EndByte     uint32
	Params      []string // ["user", "pass"]
	ReturnType  string   // "Promise<Token>"
	DocString   string   // comentarios/docstring
	CallsTo     []string // ["validatePassword", "createToken"]
	CalledBy    []string
}

// ParsedFile contiene todos los elementos extraídos de un archivo
type ParsedFile struct {
	Path      string
	Language  string
	Elements  []CodeElement
	Raw       []byte
	ParseTime int64 // milisegundos
}

// ParseFile lee y parsea un archivo
func ParseFile(filepath string) (*ParsedFile, error) {
	// TODO: implementar en MARTES
	return nil, nil
}

// DetectLanguage detecta lenguaje por extensión
func DetectLanguage(filepath string) string {
	ext := filepath.Ext(filepath)
	switch ext {
	case ".js", ".jsx":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".py":
		return "python"
	case ".go":
		return "go"
	default:
		return "unknown"
	}
}

// IsCodeFile checks if it's a file we can parse
func IsCodeFile(path string) bool {
	ext := filepath.Ext(path)
	validExts := map[string]bool{
		".js":  true,
		".jsx": true,
		".ts":  true,
		".tsx": true,
		".py":  true,
		".go":  true,
	}
	return validExts[ext]
}
```

**Archivo: `parser/ast_test.go`**

```go
package parser

import (
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"auth/login.ts", "typescript"},
		{"db/user.js", "javascript"},
		{"utils.py", "python"},
		{"main.go", "go"},
		{"readme.md", "unknown"},
	}

	for _, test := range tests {
		result := DetectLanguage(test.input)
		if result != test.expected {
			t.Errorf("DetectLanguage(%s) = %s, want %s", test.input, result, test.expected)
		}
	}
}

func TestIsCodeFile(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"auth.ts", true},
		{"utils.py", true},
		{"readme.md", false},
		{"package.json", false},
	}

	for _, test := range tests {
		result := IsCodeFile(test.input)
		if result != test.expected {
			t.Errorf("IsCodeFile(%s) = %v, want %v", test.input, result, test.expected)
		}
	}
}
```

**Test:**
```bash
go test -v ./parser
# Output: TestDetectLanguage ... PASS
#         TestIsCodeFile ... PASS
```

#### Commit
```bash
git add .
git commit -m "day1: project setup + basic types and tests"
```

---

### 🟠 MARTES: Tree-sitter Parser Implementation

#### Mañana: JavaScript/TypeScript Parser (3 horas)

**Archivo: `parser/ast.go`** (actualizar con implementación)

```go
package parser

import (
	"fmt"
	"os"
	"strings"
	"time"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/go"
)

// ParseFile parsea un archivo y extrae elementos
func ParseFile(filepath string) (*ParsedFile, error) {
	start := time.Now()

	// 1. Leer archivo
	source, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// 2. Detectar lenguaje
	lang := DetectLanguage(filepath)

	// 3. Crear parser
	parser := sitter.NewParser()

	switch lang {
	case "javascript", "typescript":
		parser.SetLanguage(javascript.GetLanguage())
	case "python":
		parser.SetLanguage(python.GetLanguage())
	case "go":
		parser.SetLanguage(go.GetLanguage())
	default:
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}

	// 4. Parse
	tree := parser.Parse(source, nil)
	root := tree.RootNode()

	// 5. Extract elements
	elements := extractElements(root, source, filepath, lang)

	return &ParsedFile{
		Path:      filepath,
		Language:  lang,
		Elements:  elements,
		Raw:       source,
		ParseTime: time.Since(start).Milliseconds(),
	}, nil
}

func extractElements(node *sitter.Node, source []byte, filepath string, lang string) []CodeElement {
	var elements []CodeElement

	cursor := sitter.NewTreeCursor(node)

	for {
		child := cursor.CurrentNode()

		// Función
		if isFunctionNode(child, lang) {
			elem := extractFunction(child, source, filepath)
			elements = append(elements, elem)
		}

		// Clase
		if isClassNode(child, lang) {
			elem := extractClass(child, source, filepath)
			elements = append(elements, elem)
		}

		if !cursor.GoToNextSibling() {
			break
		}
	}

	return elements
}

func isFunctionNode(node *sitter.Node, lang string) bool {
	nodeType := node.Type()

	switch lang {
	case "javascript", "typescript":
		return nodeType == "function_declaration" ||
			nodeType == "arrow_function" ||
			nodeType == "method_definition"
	case "python":
		return nodeType == "function_definition"
	case "go":
		return nodeType == "function_declaration" ||
			nodeType == "method_declaration"
	default:
		return false
	}
}

func isClassNode(node *sitter.Node, lang string) bool {
	nodeType := node.Type()

	switch lang {
	case "javascript", "typescript":
		return nodeType == "class_declaration"
	case "python":
		return nodeType == "class_definition"
	case "go":
		return false // no classes in Go
	default:
		return false
	}
}

func extractFunction(node *sitter.Node, source []byte, filepath string) CodeElement {
	name := extractName(node, source)
	params := extractParams(node, source)
	calls := extractCalls(node, source)
	body := extractBody(node, source)
	signature := extractSignature(node, source)

	return CodeElement{
		ID:        filepath + ":" + name,
		Name:      name,
		Type:      "function",
		File:      filepath,
		Line:      int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		Signature: signature,
		Body:      body,
		StartByte: node.StartByte(),
		EndByte:   node.EndByte(),
		Params:    params,
		CallsTo:   calls,
	}
}

func extractClass(node *sitter.Node, source []byte, filepath string) CodeElement {
	name := extractName(node, source)

	return CodeElement{
		ID:        filepath + ":" + name,
		Name:      name,
		Type:      "class",
		File:      filepath,
		Line:      int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		Signature: string(source[node.StartByte():min(node.EndByte(), node.StartByte()+200)]),
		Body:      string(source[node.StartByte():min(node.EndByte(), node.StartByte()+500)]),
		StartByte: node.StartByte(),
		EndByte:   node.EndByte(),
	}
}

// Helper functions

func extractName(node *sitter.Node, source []byte) string {
	// Buscar nodo "identifier"
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" {
			return string(source[child.StartByte():child.EndByte()])
		}
	}
	return "anonymous"
}

func extractParams(node *sitter.Node, source []byte) []string {
	var params []string

	// Buscar parámetros (simplificado)
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "formal_parameters" || child.Type() == "parameters" {
			// Iterar parámetros
			for j := 0; j < int(child.ChildCount()); j++ {
				param := child.Child(j)
				if param.Type() == "identifier" {
					params = append(params, string(source[param.StartByte():param.EndByte()]))
				}
			}
		}
	}

	return params
}

func extractBody(node *sitter.Node, source []byte) string {
	endByte := node.EndByte()
	if endByte-node.StartByte() > 500 {
		endByte = node.StartByte() + 500
	}
	return string(source[node.StartByte():endByte])
}

func extractSignature(node *sitter.Node, source []byte) string {
	// Primeros 200 chars del nodo
	endByte := node.EndByte()
	if endByte-node.StartByte() > 200 {
		endByte = node.StartByte() + 200
	}
	return string(source[node.StartByte():endByte])
}

func extractCalls(node *sitter.Node, source []byte) []string {
	var calls []string

	// Buscar call_expression nodes (simplificado)
	// En producción, habría que recorrer recursivamente
	// Por ahora retornar vacío

	return calls
}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}
```

**Archivo: `parser/ast_test.go`** (actualizar)

```go
package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFile(t *testing.T) {
	// Crear test file
	testFile := "test_sample.ts"
	content := `
function login(user: string, pass: string) {
  const valid = validatePassword(pass);
  if (valid) {
    return createToken(user);
  }
}

class Auth {
  constructor() {}
}
`
	os.WriteFile(testFile, []byte(content), 0644)
	defer os.Remove(testFile)

	// Parse
	parsed, err := ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Verify
	if parsed.Language != "typescript" {
		t.Errorf("Language = %s, want typescript", parsed.Language)
	}

	if len(parsed.Elements) < 2 {
		t.Errorf("Found %d elements, want at least 2", len(parsed.Elements))
	}

	// Find login function
	var loginFunc *CodeElement
	for _, elem := range parsed.Elements {
		if elem.Name == "login" {
			loginFunc = &elem
			break
		}
	}

	if loginFunc == nil {
		t.Fatal("login function not found")
	}

	if loginFunc.Type != "function" {
		t.Errorf("Type = %s, want function", loginFunc.Type)
	}

	if len(loginFunc.Params) != 2 {
		t.Errorf("Params = %v, want 2 params", loginFunc.Params)
	}
}
```

**Test:**
```bash
go test -v ./parser
# Output: TestParseFile ... PASS
```

#### Tarde: Test Real File (2 horas)

Crear directorio test:
```bash
mkdir -p test/sample-repo/auth
mkdir -p test/sample-repo/db
```

**Archivo: `test/sample-repo/auth/login.ts`**
```typescript
// @deprecated: migrate to OAuth2
// @hook: beforeAuth, afterSession
// @author: Ruffini

export async function loginHandler(req: Request) {
  const { email, password } = req.body;
  
  // Validar
  const user = await validatePassword(email, password);
  if (!user) {
    throw new Error('Invalid credentials');
  }
  
  // Crear token
  const token = createToken(user);
  const session = createSession(user);
  
  return { token, session };
}

// @todo: use bcrypt instead
function validatePassword(email: string, pass: string) {
  return checkHash(email, pass);
}

function createToken(user) {
  return signJWT(user);
}

function createSession(user) {
  return db.users.update({ session: true });
}

function checkHash(email, pass) {
  // Implementation
  return true;
}

function signJWT(user) {
  // Implementation
  return 'token';
}
```

**Archivo: `cmd/main.go`** (básico)

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ruffini/prism-platform/parser"
)

func main() {
	indexCmd := flag.NewFlagSet("index", flag.ExitOnError)
	repoPath := indexCmd.String("repo", ".", "Repository path")

	if len(os.Args) < 2 {
		fmt.Println("Usage: cip <command>")
		fmt.Println("Commands: index")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "index":
		indexCmd.Parse(os.Args[2:])
		indexRepository(*repoPath)
	}
}

func indexRepository(repoPath string) {
	fmt.Printf("Indexing: %s\n", repoPath)

	filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || !parser.IsCodeFile(path) {
			return nil
		}

		fmt.Printf("  Parsing: %s\n", path)
		parsed, err := parser.ParseFile(path)
		if err != nil {
			fmt.Printf("    Error: %v\n", err)
			return nil
		}

		fmt.Printf("    Found %d elements\n", len(parsed.Elements))
		for _, elem := range parsed.Elements {
			fmt.Printf("      - %s (%s)\n", elem.Name, elem.Type)
		}

		return nil
	})

	fmt.Println("✅ Done!")
}
```

**Test manual:**
```bash
go build -o cip ./cmd/main.go
./cip index -repo ./test/sample-repo
```

**Expected output:**
```
Indexing: ./test/sample-repo
  Parsing: test/sample-repo/auth/login.ts
    Found 6 elements
      - loginHandler (function)
      - validatePassword (function)
      - createToken (function)
      - createSession (function)
      - checkHash (function)
      - signJWT (function)
✅ Done!
```

#### Commit
```bash
git add .
git commit -m "day2: implement tree-sitter parser for JS/TS"
```

---

### 🟡 MIÉRCOLES: Graph Builder + SQLite

#### Mañana: SQLite Schema + Graph Structure (2-3 horas)

**Archivo: `db/schema.go`**

```go
package db

import (
	"database/sql"
	"fmt"
)

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Create schema
	if err := createSchema(db); err != nil {
		return nil, err
	}

	return db, nil
}

func createSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS nodes (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT,
		file TEXT NOT NULL,
		line INTEGER,
		end_line INTEGER,
		signature TEXT,
		body TEXT,
		start_byte INTEGER,
		end_byte INTEGER,
		pagerank REAL DEFAULT 0,
		blast_radius INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS edges (
		source TEXT NOT NULL,
		target TEXT NOT NULL,
		type TEXT NOT NULL DEFAULT 'calls',
		PRIMARY KEY (source, target, type),
		FOREIGN KEY (source) REFERENCES nodes(id),
		FOREIGN KEY (target) REFERENCES nodes(id)
	);

	CREATE TABLE IF NOT EXISTS metadata (
		function_id TEXT PRIMARY KEY,
		deprecated BOOLEAN DEFAULT 0,
		deprecated_reason TEXT,
		hooks TEXT,
		todos TEXT,
		authors TEXT,
		tags TEXT,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (function_id) REFERENCES nodes(id)
	);

	CREATE INDEX IF NOT EXISTS idx_nodes_file ON nodes(file);
	CREATE INDEX IF NOT EXISTS idx_nodes_type ON nodes(type);
	CREATE INDEX IF NOT EXISTS idx_edges_source ON edges(source);
	CREATE INDEX IF NOT EXISTS idx_edges_target ON edges(target);
	`

	_, err := db.Exec(schema)
	return err
}

func ClearDB(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM edges; DELETE FROM metadata; DELETE FROM nodes;")
	return err
}
```

#### Tarde: Graph Builder Implementation (3 horas)

**Archivo: `graph/builder.go`**

```go
package graph

import (
	"database/sql"
	"fmt"

	"github.com/ruffini/prism-platform/parser"
)

type CodeGraph struct {
	DB *sql.DB
}

type Node struct {
	ID        string
	Name      string
	Type      string
	File      string
	Line      int
	EndLine   int
	Signature string
	Body      string
	PageRank  float64
	BlastRad  int
}

type Edge struct {
	Source string
	Target string
	Type   string
}

// NewGraph creates a new graph
func NewGraph(db *sql.DB) *CodeGraph {
	return &CodeGraph{DB: db}
}

// AddNode inserts a node
func (g *CodeGraph) AddNode(node Node) error {
	query := `
	INSERT INTO nodes (id, name, type, file, line, end_line, signature, body)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		signature = excluded.signature,
		body = excluded.body,
		updated_at = CURRENT_TIMESTAMP
	`

	_, err := g.DB.Exec(query,
		node.ID, node.Name, node.Type, node.File, node.Line, node.EndLine,
		node.Signature, node.Body)

	return err
}

// AddEdge inserts an edge
func (g *CodeGraph) AddEdge(source, target, edgeType string) error {
	query := `
	INSERT INTO edges (source, target, type)
	VALUES (?, ?, ?)
	ON CONFLICT DO NOTHING
	`

	_, err := g.DB.Exec(query, source, target, edgeType)
	return err
}

// BuildFromParsed constructs graph from parsed files
func (g *CodeGraph) BuildFromParsed(parsedFiles map[string]*parser.ParsedFile) error {
	// 1. Add all nodes
	for filepath, parsed := range parsedFiles {
		for _, elem := range parsed.Elements {
			node := Node{
				ID:        elem.ID,
				Name:      elem.Name,
				Type:      elem.Type,
				File:      filepath,
				Line:      elem.Line,
				EndLine:   elem.EndLine,
				Signature: elem.Signature,
				Body:      elem.Body,
			}

			if err := g.AddNode(node); err != nil {
				return fmt.Errorf("failed to add node %s: %w", elem.ID, err)
			}
		}
	}

	// 2. Add all edges
	for _, parsed := range parsedFiles {
		for _, elem := range parsed.Elements {
			for _, callee := range elem.CallsTo {
				// Note: callee might not be in graph (external call)
				// That's OK, we skip it
				if err := g.AddEdge(elem.ID, callee, "calls"); err != nil {
					// Ignore if target doesn't exist
					// return fmt.Errorf("failed to add edge %s -> %s: %w", elem.ID, callee, err)
				}
			}
		}
	}

	// 3. Calculate PageRank
	if err := g.CalculatePageRank(); err != nil {
		return fmt.Errorf("failed to calculate pagerank: %w", err)
	}

	// 4. Calculate blast radius
	if err := g.CalculateBlastRadius(); err != nil {
		return fmt.Errorf("failed to calculate blast radius: %w", err)
	}

	return nil
}

// GetNode retrieves a node
func (g *CodeGraph) GetNode(id string) (*Node, error) {
	query := `SELECT id, name, type, file, line, end_line, signature, body, pagerank, blast_radius
	          FROM nodes WHERE id = ?`

	var node Node
	err := g.DB.QueryRow(query, id).Scan(
		&node.ID, &node.Name, &node.Type, &node.File, &node.Line, &node.EndLine,
		&node.Signature, &node.Body, &node.PageRank, &node.BlastRad)

	if err != nil {
		return nil, err
	}

	return &node, nil
}

// GetNodesByFile retrieves all nodes in a file
func (g *CodeGraph) GetNodesByFile(file string) ([]Node, error) {
	query := `SELECT id, name, type, file, line, end_line, signature, body, pagerank, blast_radius
	          FROM nodes WHERE file = ? ORDER BY line`

	rows, err := g.DB.Query(query, file)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var node Node
		if err := rows.Scan(&node.ID, &node.Name, &node.Type, &node.File,
			&node.Line, &node.EndLine, &node.Signature, &node.Body, &node.PageRank, &node.BlastRad); err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// GetDependencies retrieves what a node calls
func (g *CodeGraph) GetDependencies(nodeID string) ([]string, error) {
	query := `SELECT target FROM edges WHERE source = ? AND type = 'calls'`

	rows, err := g.DB.Query(query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []string
	for rows.Next() {
		var target string
		if err := rows.Scan(&target); err != nil {
			return nil, err
		}
		deps = append(deps, target)
	}

	return deps, nil
}

// GetCallers retrieves what calls a node
func (g *CodeGraph) GetCallers(nodeID string) ([]string, error) {
	query := `SELECT source FROM edges WHERE target = ? AND type = 'calls'`

	rows, err := g.DB.Query(query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var callers []string
	for rows.Next() {
		var source string
		if err := rows.Scan(&source); err != nil {
			return nil, err
		}
		callers = append(callers, source)
	}

	return callers, nil
}

// CalculatePageRank computes PageRank for all nodes
func (g *CodeGraph) CalculatePageRank() error {
	// Simplified PageRank: count in-degree
	query := `
	UPDATE nodes SET pagerank = (
		SELECT COALESCE(COUNT(*), 0) FROM edges WHERE target = nodes.id
	) / (SELECT COUNT(*) FROM nodes)
	`

	_, err := g.DB.Exec(query)
	return err
}

// CalculateBlastRadius computes blast radius for all nodes
func (g *CodeGraph) CalculateBlastRadius() error {
	// Simplified: how many nodes are reachable (BFS)
	// For now, just count direct dependencies
	query := `
	UPDATE nodes SET blast_radius = (
		SELECT COUNT(*) FROM edges WHERE source = nodes.id
	)
	`

	_, err := g.DB.Exec(query)
	return err
}

// Stats returns graph statistics
func (g *CodeGraph) Stats() (map[string]interface{}, error) {
	var nodeCount, edgeCount int

	g.DB.QueryRow("SELECT COUNT(*) FROM nodes").Scan(&nodeCount)
	g.DB.QueryRow("SELECT COUNT(*) FROM edges").Scan(&edgeCount)

	return map[string]interface{}{
		"nodes": nodeCount,
		"edges": edgeCount,
	}, nil
}
```

#### Test

**Archivo: `graph/builder_test.go`**

```go
package graph

import (
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
	"github.com/ruffini/prism-platform/db"
	"github.com/ruffini/prism-platform/parser"
)

func setupTestDB() (*sql.DB, func()) {
	os.Remove("test.db")
	d, _ := db.InitDB("test.db")
	return d, func() {
		d.Close()
		os.Remove("test.db")
	}
}

func TestAddNode(t *testing.T) {
	sqlDB, cleanup := setupTestDB()
	defer cleanup()

	g := NewGraph(sqlDB)

	node := Node{
		ID:        "auth.ts:login",
		Name:      "login",
		Type:      "function",
		File:      "auth.ts",
		Line:      1,
		Signature: "function login(user)",
		Body:      "{ ... }",
	}

	if err := g.AddNode(node); err != nil {
		t.Fatalf("AddNode failed: %v", err)
	}

	retrieved, _ := g.GetNode("auth.ts:login")
	if retrieved.Name != "login" {
		t.Errorf("Name = %s, want login", retrieved.Name)
	}
}

func TestAddEdge(t *testing.T) {
	sqlDB, cleanup := setupTestDB()
	defer cleanup()

	g := NewGraph(sqlDB)

	// Add nodes first
	g.AddNode(Node{ID: "a", Name: "a", Type: "function", File: "test.ts", Line: 1})
	g.AddNode(Node{ID: "b", Name: "b", Type: "function", File: "test.ts", Line: 2})

	// Add edge
	if err := g.AddEdge("a", "b", "calls"); err != nil {
		t.Fatalf("AddEdge failed: %v", err)
	}

	deps, _ := g.GetDependencies("a")
	if len(deps) != 1 || deps[0] != "b" {
		t.Errorf("Dependencies = %v, want [b]", deps)
	}
}
```

**Test:**
```bash
go test -v ./graph
# Output: TestAddNode ... PASS
#         TestAddEdge ... PASS
```

#### Commit
```bash
git add .
git commit -m "day3: implement SQLite graph builder with PageRank"
```

---

### 🟢 JUEVES: Call Extraction + Full Integration

#### Mañana: Extract Function Calls (2-3 horas)

**Archivo: `parser/calls.go`** (NEW)

```go
package parser

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// ExtractCalls finds all function calls in an AST node
func ExtractCalls(node *sitter.Node, source []byte, lang string) []string {
	var calls []string
	visited := make(map[uint32]bool)

	walkForCalls(node, source, lang, &calls, visited)

	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, call := range calls {
		if !seen[call] {
			unique = append(unique, call)
			seen[call] = true
		}
	}

	return unique
}

func walkForCalls(node *sitter.Node, source []byte, lang string, calls *[]string, visited map[uint32]bool) {
	if visited[node.StartByte()] {
		return
	}
	visited[node.StartByte()] = true

	nodeType := node.Type()

	// Detectar call_expression
	if nodeType == "call_expression" {
		// Get the function name (first child usually)
		if node.ChildCount() > 0 {
			child := node.Child(0)
			if child.Type() == "identifier" {
				name := string(source[child.StartByte():child.EndByte()])
				*calls = append(*calls, name)
			}
		}
	}

	// Recurse
	for i := 0; i < int(node.ChildCount()); i++ {
		walkForCalls(node.Child(i), source, lang, calls, visited)
	}
}
```

**Actualizar: `parser/ast.go`**

```go
// En extractFunction:
func extractFunction(node *sitter.Node, source []byte, filepath string) CodeElement {
	name := extractName(node, source)
	params := extractParams(node, source)
	calls := ExtractCalls(node, source, "") // NEW
	body := extractBody(node, source)
	signature := extractSignature(node, source)

	return CodeElement{
		// ... rest same
		CallsTo: calls, // NOW populated
	}
}
```

#### Tarde: CLI + Full Integration (2-3 horas)

**Actualizar: `cmd/main.go`**

```go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/ruffini/prism-platform/db"
	"github.com/ruffini/prism-platform/graph"
	"github.com/ruffini/prism-platform/parser"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: cip <command> [options]")
		fmt.Println("Commands:")
		fmt.Println("  index -repo <path>  - Index a repository")
		fmt.Println("  stats                - Show graph statistics")
		fmt.Println("  export -out <file>  - Export graph as JSON")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "index":
		indexCmd := flag.NewFlagSet("index", flag.ExitOnError)
		repoPath := indexCmd.String("repo", ".", "Repository path")
		dbPath := indexCmd.String("db", "code_graph.db", "Database path")
		indexCmd.Parse(os.Args[2:])
		indexRepository(*repoPath, *dbPath)

	case "stats":
		statsCmd := flag.NewFlagSet("stats", flag.ExitOnError)
		dbPath := statsCmd.String("db", "code_graph.db", "Database path")
		statsCmd.Parse(os.Args[2:])
		showStats(*dbPath)

	case "export":
		exportCmd := flag.NewFlagSet("export", flag.ExitOnError)
		dbPath := exportCmd.String("db", "code_graph.db", "Database path")
		outPath := exportCmd.String("out", "graph.json", "Output path")
		exportCmd.Parse(os.Args[2:])
		exportGraph(*dbPath, *outPath)

	default:
		fmt.Println("Unknown command:", os.Args[1])
		os.Exit(1)
	}
}

func indexRepository(repoPath, dbPath string) {
	fmt.Printf("🔍 Indexing: %s\n", repoPath)

	// Initialize DB
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to init DB: %v\n", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	// Clear previous data
	db.ClearDB(sqlDB)

	// Parse all files
	fmt.Println("📄 Parsing files...")
	parsedFiles := make(map[string]*parser.ParsedFile)
	fileCount := 0

	filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || !parser.IsCodeFile(path) {
			return nil
		}

		rel, _ := filepath.Rel(repoPath, path)
		fmt.Printf("  ✓ %s\n", rel)

		parsed, err := parser.ParseFile(path)
		if err != nil {
			fmt.Printf("    ⚠ Error: %v\n", err)
			return nil
		}

		parsedFiles[rel] = parsed
		fileCount++

		return nil
	})

	fmt.Printf("✅ Parsed %d files\n", fileCount)

	// Build graph
	fmt.Println("🔗 Building graph...")
	g := graph.NewGraph(sqlDB)

	if err := g.BuildFromParsed(parsedFiles); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to build graph: %v\n", err)
		os.Exit(1)
	}

	// Show stats
	stats, _ := g.Stats()
	fmt.Printf("✅ Graph complete:\n")
	fmt.Printf("   Nodes: %v\n", stats["nodes"])
	fmt.Printf("   Edges: %v\n", stats["edges"])
	fmt.Printf("   Database: %s\n", dbPath)
}

func showStats(dbPath string) {
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to open DB: %v\n", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	g := graph.NewGraph(sqlDB)
	stats, _ := g.Stats()

	fmt.Printf("📊 Graph Statistics\n")
	fmt.Printf("   Nodes: %v\n", stats["nodes"])
	fmt.Printf("   Edges: %v\n", stats["edges"])
}

func exportGraph(dbPath, outPath string) {
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to open DB: %v\n", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	// TODO: implement proper export
	fmt.Printf("📤 Exporting to %s...\n", outPath)
}
```

#### Test

```bash
go build -o cip ./cmd/main.go

# Full test
./cip index -repo ./test/sample-repo

# Check stats
./cip stats
```

**Expected:**
```
🔍 Indexing: ./test/sample-repo
📄 Parsing files...
  ✓ auth/login.ts
✅ Parsed 1 files
🔗 Building graph...
✅ Graph complete:
   Nodes: 6
   Edges: 4
   Database: code_graph.db

📊 Graph Statistics
   Nodes: 6
   Edges: 4
```

#### Commit
```bash
git add .
git commit -m "day4: implement call extraction + full CLI integration"
```

---

### 🔵 VIERNES: Testing + Polish Week 1

#### Mañana: Comprehensive Tests (2-3 horas)

Crear más test files para validar todo funciona:

```bash
mkdir -p test/sample-repo/db
mkdir -p test/sample-repo/utils
```

**Archivo: `test/sample-repo/db/user.ts`**
```typescript
export async function getUser(id: string) {
  return db.query('SELECT * FROM users WHERE id = ?', [id]);
}

export async function updateUser(id: string, data) {
  return db.query('UPDATE users SET ...', [id, data]);
}
```

**Archivo: `test/sample-repo/utils/helpers.ts`**
```typescript
export function hashPassword(pass: string) {
  return crypto.hash(pass);
}

export function validateEmail(email: string) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}
```

**Test suite completo:**
```bash
go test -v ./...
# Should pass all tests
```

#### Tarde: Documentation + Cleanup (1-2 horas)

**Archivo: `README.md`**

```markdown
# PRISM Platform

Graph-powered code understanding + token optimization for Claude Code.

## Quick Start

```bash
# Index a repository
./cip index -repo /path/to/your/code

# View statistics
./cip stats

# Serve API (coming soon)
./cip serve
```

## Architecture

- **Parser**: Tree-sitter based extraction (JS, TS, Python, Go)
- **Graph**: SQLite dependency graph with PageRank
- **Vector**: Ollama embeddings + LanceDB (coming soon)
- **API**: REST + WebSocket (coming soon)
- **Frontend**: React + Monaco editor (coming soon)

## Development

```bash
make dev      # Run with hot reload
make test     # Run all tests
make build    # Build binary
```
```

**Archivo: `docs/architecture.md`**

```markdown
# Architecture

## Parser (Lunes-Martes)
- Tree-sitter extracts AST
- Supports JS, TS, Python, Go
- Outputs: CodeElement with name, type, line, calls

## Graph (Miércoles-Jueves)
- SQLite stores nodes + edges
- PageRank for importance ranking
- Blast radius for impact analysis

## Coming Soon
- Vector embeddings (LanceDB)
- MCP Server integration
- React frontend
```

#### Final Commit Week 1
```bash
git add .
git commit -m "week1-complete: parser + graph builder working end-to-end"
git tag week1-complete
```

---

## 📅 SEMANA 2: Vector Search + Frontend Setup

### 🔴 LUNES: Ollama Embeddings

[Similar structure: mañana + tarde, commits diarios]

**Tareas:**
- Instalar Ollama localmente
- Crear `vector/embedder.go` con Ollama integration
- Test embeddings generadas

### 🟠 MARTES: LanceDB Indexing

**Tareas:**
- Implementar `vector/indexer.go`
- Index todos los nodos con embeddings
- Test search functionality

### 🟡 MIÉRCOLES-VIERNES: Frontend Setup

**Tareas:**
- Setup React + Vite
- Crear FileTree component
- Crear CodeEditor (Monaco)
- Crear Graph component (D3)
- WebSocket sync backend ↔ frontend

---

## 📅 SEMANA 3: Annotations + MCP Server

### Lunes-Martes: Metadata Parser

**Tareas:**
- Parse `@deprecated`, `@hook`, `@todo`, `@author`
- Store in SQLite metadata table
- Expose via API

### Miércoles-Viernes: MCP Server

**Tareas:**
- Implement MCP Server
- Tools: search_context, get_file_smart, trace_impact
- Test with Claude Code

---

## 🎯 Checkpoints / Go/No-Go

| Week | Checkpoint | Status |
|------|-----------|--------|
| 1 | Parser + SQLite graph complete | GO ✅ |
| 2 | Frontend + embeddings complete | GO ✅ |
| 3 | MCP Server + annotations complete | GO ✅ |

---

## 📊 Time Allocation

```
Total: 3 weeks × 5 days = 15 days × 8 hours = 120 hours

Week 1: Backend         = 40 hours
  - Parser             = 12 hours
  - Graph              = 16 hours
  - Integration        = 12 hours

Week 2: Frontend + Vector = 40 hours
  - Vector DB          = 12 hours
  - React setup        = 16 hours
  - Components         = 12 hours

Week 3: MCP + Polish    = 40 hours
  - MCP Server         = 16 hours
  - Annotations        = 12 hours
  - Testing + docs     = 12 hours
```

---

## 💾 Git Workflow

```bash
# Daily commits
git commit -m "dayN: what you did today"

# Weekly tags
git tag week1-parser-graph
git tag week2-frontend-vector
git tag week3-mcp-complete

# Push to GitHub
git push origin main --tags
```

---

## 🚀 Ready to Start?

**Next step:** Create GitHub repo + make first commit

```bash
git remote add origin https://github.com/ruffini/prism-platform.git
git branch -M main
git push -u origin main
```

Then follow Week 1 schedule day by day.
