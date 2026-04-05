# 🔮 PRISM - Program Representation & Intelligence Semantic Mapper

**Una herramienta integral que combina:**
- ✅ Auto-parse de código (dependencias, funciones, imports)
- ✅ Editor visual interactivo con grafo en vivo
- ✅ Anotaciones (deprecated, hooks, TODO, links)
- ✅ Smart token optimization para Claude Code
- ✅ Documentación visual que nunca envejece

> Like Soulforge + Visual Code Editor + Token Optimizer + Documentation all in one

---

## 📋 Tabla de Contenidos

1. [El Problema](#el-problema)
2. [La Solución](#la-solución)
3. [Arquitectura](#arquitectura)
4. [Plan de Implementación (3-4 semanas)](#plan-de-implementación-3-4-semanas)
5. [Componentes Detallados](#componentes-detallados)
6. [Workflow del Usuario](#workflow-del-usuario)
7. [MCP Tools para Claude Code](#mcp-tools-para-claude-code)

---

## El Problema

### Los 3 Problemas que Resuelve

#### 1️⃣ **Documentación Estática = Mentirosa**

```
README.md escrito hace 6 meses:
"loginHandler() → validatePassword() → createToken()"

Realidad hoy:
loginHandler() → validatePassword() → getUser() → createToken() 
                                                   ↓
                                              [DEPRECATED - usa OAuth2]
                                              [TODO - agregar rate limit]

README nunca se actualiza
         ↓
Dev nuevo se pierde
         ↓
Claude Code lee código desactualizado
         ↓
Respuestas equivocadas
```

#### 2️⃣ **Token Wastage en Claude Code**

```
Pregunta: "¿Cómo funciona login?"

Claude Code lee:
- auth/login.ts (200 líneas)
- auth/validate.ts (150 líneas)
- auth/session.ts (300 líneas)
- crypto/jwt.ts (250 líneas)
- crypto/hash.ts (200 líneas)
- db/users.ts (500 líneas)
- ... más 20 archivos

Total: 50,000 tokens 💀

Necesitaba: 5,000 tokens (solo el core logic)
Desperdicio: 45,000 tokens (90%)
```

#### 3️⃣ **Refactoring es Aterrador**

```
Quiero cambiar validatePassword()

Preguntas:
- ¿Quién lo llama?
- ¿Hay otras funciones que dependen?
- ¿Está marcado como deprecated?
- ¿Afecta a hooks?

Sin herramienta:
- Grep → confusing
- Manual code review → tedious
- Claude Code → lee TODO el repo

Con herramienta:
- Click en función → "Show blast radius"
- Ve visual: qué se afecta
- Refactoriza con confianza
```

---

## La Solución

### Arquitectura de 3 Capas

```
┌───────────────────────────────────────────────────────────┐
│        CAPA 1: Auto-Parse (Backend)                       │
├───────────────────────────────────────────────────────────┤
│                                                           │
│  Tu código → tree-sitter → AST                            │
│       ↓                                                    │
│  Extrae: funciones, imports, calls, clases, etc.        │
│       ↓                                                    │
│  Construye SQLite graph:                                  │
│  ├─ Nodos: {file, function, line, signature, body}       │
│  ├─ Edges: {source → target, type: "calls", "imports"}   │
│  └─ Metadata: {PageRank, blast radius, cochange}         │
│       ↓                                                    │
│  Genera embeddings (Ollama local):                        │
│  ├─ Cada función → vector (384 dims)                      │
│  └─ Indexa en LanceDB para search                         │
│                                                           │
└───────────────────────────────────────────────────────────┘
                            ↓
┌───────────────────────────────────────────────────────────┐
│     CAPA 2: Editor Visual + Anotaciones (React UI)        │
├───────────────────────────────────────────────────────────┤
│                                                           │
│  ┌─────────────┬──────────────────┬──────────────────┐   │
│  │ FILE TREE   │  CODE EDITOR     │  GRAPH VISUAL    │   │
│  │             │  (Monaco)        │  (D3/Konva)      │   │
│  │ auth/       │                  │                  │   │
│  │ ├─ login.ts │ function login() │  login ─────┐   │   │
│  │ │ ├─ [!]    │ {                │      │      │   │   │
│  │ │ │ login() │   // @deprecated │      ▼      │   │   │
│  │ │ ├─ [X]    │   validate()     │   validate  │   │   │
│  │ │ │ hash()  │   createToken()  │      │      │   │   │
│  │ │ └─ [○]    │ }                │      ▼      │   │   │
│  │ │   token() │                  │  createToken    │   │
│  │ └─ validate │                  │                  │   │
│  │   └─ pass() │ ANNOTATIONS      │                  │   │
│  │             │ ────────────────  │                  │   │
│  │ [LEGEND]    │ Deprecated: true  │                  │   │
│  │ [!] = hook  │ Reason: OAuth2    │                  │   │
│  │ [X] = deprecated
│  │ [○] = todo  │ Hooks:            │                  │   │
│  │             │ - beforeAuth      │                  │   │
│  │             │ - afterSession    │                  │   │
│  │             │                   │                  │   │
│  │             │ TODO:             │                  │   │
│  │             │ - add rate limit  │                  │   │
│  │             │                   │                  │   │
│  │             │ Authors: Ruffini  │                  │   │
│  └─────────────┴──────────────────┴──────────────────┘   │
│                                                           │
│  Editar código → Auto-actualiza grafo ✅                 │
│  Dragging grafo → Crea relación + metadata ✅            │
│  Anotar función → Se guarda en DB ✅                     │
│                                                           │
└───────────────────────────────────────────────────────────┘
                            ↓
┌───────────────────────────────────────────────────────────┐
│  CAPA 3: MCP Server (Claude Code Integration)            │
├───────────────────────────────────────────────────────────┤
│                                                           │
│  Tools expuestos:                                         │
│  • search_context(query)                                  │
│  • get_file_smart(filename, symbol)                       │
│  • trace_impact(function)                                 │
│  • list_deprecated()                                      │
│  • get_annotations(function)                              │
│                                                           │
│  Resultado: 50K tokens → 5K tokens ✅                    │
│            1 min → 10 seg ✅                              │
│                                                           │
└───────────────────────────────────────────────────────────┘
```

---

## Arquitectura

### Estructura de Carpetas

```
code-intelligence-platform/
│
├── backend/ (Go)
│   ├── main.go
│   ├── parser/
│   │   ├── ast.go           ← tree-sitter parsing
│   │   ├── extractor.go      ← extract functions/calls/imports
│   │   └── walker.go         ← traverse AST
│   │
│   ├── graph/
│   │   ├── builder.go        ← construct SQLite graph
│   │   ├── enricher.go       ← PageRank, blast radius, cochange
│   │   ├── traversal.go      ← efficient queries
│   │   └── sync.go           ← keep graph in sync with changes
│   │
│   ├── vector/
│   │   ├── embedder.go       ← Ollama embeddings
│   │   ├── indexer.go        ← LanceDB indexing
│   │   └── search.go         ← semantic + keyword search
│   │
│   ├── metadata/
│   │   ├── parser.go         ← parse @deprecated, @hook, @todo
│   │   ├── validator.go      ← validate metadata schema
│   │   └── store.go          ← persist to SQLite
│   │
│   ├── api/
│   │   ├── http.go           ← REST API endpoints
│   │   ├── websocket.go      ← real-time sync
│   │   └── routes.go         ← route definitions
│   │
│   ├── mcp/
│   │   ├── server.go         ← MCP Server
│   │   └── tools.go          ← tool definitions
│   │
│   ├── db/
│   │   ├── schema.go         ← SQLite schema
│   │   └── migrations.go
│   │
│   └── go.mod, go.sum
│
├── frontend/ (React + TypeScript)
│   ├── src/
│   │   ├── components/
│   │   │   ├── FileTree.tsx           ← left sidebar tree
│   │   │   ├── CodeEditor.tsx         ← Monaco editor
│   │   │   ├── Graph.tsx              ← D3/Konva graph
│   │   │   ├── AnnotationsSidebar.tsx ← right sidebar metadata
│   │   │   └── Toolbar.tsx            ← top actions
│   │   │
│   │   ├── state/
│   │   │   ├── codeStore.ts           ← code content
│   │   │   ├── graphStore.ts          ← graph state
│   │   │   ├── metadataStore.ts       ← annotations
│   │   │   ├── uiStore.ts             ← UI state
│   │   │   └── sync.ts                ← bidirectional sync
│   │   │
│   │   ├── api/
│   │   │   ├── client.ts              ← API calls
│   │   │   └── websocket.ts           ← WS client
│   │   │
│   │   ├── hooks/
│   │   │   ├── useCode.ts
│   │   │   ├── useGraph.ts
│   │   │   └── useMetadata.ts
│   │   │
│   │   ├── utils/
│   │   │   ├── parser.ts              ← client-side parsing
│   │   │   ├── graph.ts               ← graph utilities
│   │   │   └── formatting.ts
│   │   │
│   │   └── App.tsx, main.tsx
│   │
│   ├── package.json
│   ├── vite.config.ts
│   └── tsconfig.json
│
├── tests/
│   ├── parser_test.go
│   ├── graph_test.go
│   └── integration_test.go
│
├── docs/
│   ├── architecture.md
│   ├── api-reference.md
│   └── examples.md
│
├── Makefile
├── docker-compose.yml
├── .gitignore
└── README.md
```

---

## Plan de Implementación (3-4 semanas)

### 📅 SEMANA 1: Parser + Graph (Backend Foundation)

#### Lunes-Martes: Setup + Tree-sitter Parser

**Objetivo:** Extraer funciones, imports, calls de código

```bash
go mod init code-intelligence-platform

go get github.com/smacker/go-tree-sitter
go get github.com/smacker/go-tree-sitter/javascript
go get github.com/smacker/go-tree-sitter/python
go get github.com/smacker/go-tree-sitter/go
go get modernc.org/sqlite
go get github.com/lancedb/lancedb
```

**Archivo: `parser/ast.go`**

```go
package parser

import (
    sitter "github.com/smacker/go-tree-sitter"
    "github.com/smacker/go-tree-sitter/javascript"
    "os"
)

type CodeElement struct {
    ID          string
    Name        string
    Type        string        // "function", "class", "method"
    File        string
    Line        int
    EndLine     int
    Signature   string
    Body        string
    StartByte   uint32
    EndByte     uint32
    Params      []string
    ReturnType  string
    DocString   string
    CallsTo     []string      // quién llama a quién
    CalledBy    []string
}

type ParsedFile struct {
    Path     string
    Language string
    Elements []CodeElement
    Raw      []byte
}

func ParseFile(filepath string) (*ParsedFile, error) {
    // 1. Leer archivo
    source, err := os.ReadFile(filepath)
    if err != nil {
        return nil, err
    }

    // 2. Detectar lenguaje
    lang := detectLanguage(filepath)

    // 3. Parse con tree-sitter
    parser := sitter.NewParser()
    switch lang {
    case "javascript":
        parser.SetLanguage(javascript.GetLanguage())
    case "typescript":
        parser.SetLanguage(javascript.GetLanguage()) // Same as JS
    case "python":
        parser.SetLanguage(python.GetLanguage())
    case "go":
        parser.SetLanguage(go.GetLanguage())
    }

    tree := parser.Parse(source, nil)

    // 4. Extract elements
    elements := extractElements(tree.RootNode(), filepath, source, lang)

    return &ParsedFile{
        Path:     filepath,
        Language: lang,
        Elements: elements,
        Raw:      source,
    }, nil
}

func extractElements(node *sitter.Node, filepath string, source []byte, lang string) []CodeElement {
    var elements []CodeElement
    
    cursor := sitter.NewTreeCursor(node)
    
    for {
        child := cursor.CurrentNode()
        
        // Detectar funciones
        if isFunctionNode(child, lang) {
            elem := extractFunction(child, filepath, source)
            elements = append(elements, elem)
        }
        
        // Detectar clases
        if isClassNode(child, lang) {
            elem := extractClass(child, filepath, source)
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
        return nodeType == "function_declaration" || nodeType == "method_declaration"
    }
    return false
}

func extractFunction(node *sitter.Node, filepath string, source []byte) CodeElement {
    name := extractName(node, source)
    params := extractParams(node, source)
    calls := extractCalls(node, source)
    
    return CodeElement{
        ID:        filepath + ":" + name,
        Name:      name,
        Type:      "function",
        File:      filepath,
        Line:      int(node.StartPoint().Row) + 1,
        EndLine:   int(node.EndPoint().Row) + 1,
        Signature: extractSignature(node, source),
        Body:      string(source[node.StartByte():min(node.EndByte(), node.StartByte()+500)]),
        StartByte: node.StartByte(),
        EndByte:   node.EndByte(),
        Params:    params,
        CallsTo:   calls,
    }
}

// Helper functions (simplified)

func extractName(node *sitter.Node, source []byte) string {
    for i := 0; i < int(node.ChildCount()); i++ {
        child := node.Child(i)
        if child.Type() == "identifier" {
            return string(source[child.StartByte():child.EndByte()])
        }
    }
    return "anonymous"
}

func extractParams(node *sitter.Node, source []byte) []string {
    // Similar logic
    return []string{}
}

func extractCalls(node *sitter.Node, source []byte) []string {
    // Find call_expression nodes
    return []string{}
}

func extractSignature(node *sitter.Node, source []byte) string {
    return string(source[node.StartByte():min(node.EndByte(), node.StartByte()+200)])
}

func detectLanguage(filepath string) string {
    if strings.HasSuffix(filepath, ".js") {
        return "javascript"
    }
    if strings.HasSuffix(filepath, ".ts") || strings.HasSuffix(filepath, ".tsx") {
        return "typescript"
    }
    if strings.HasSuffix(filepath, ".py") {
        return "python"
    }
    if strings.HasSuffix(filepath, ".go") {
        return "go"
    }
    return "unknown"
}
```

**Checklist:**
- [ ] `ParseFile()` funciona para JS/TS/Python/Go
- [ ] Extrae funciones correctamente
- [ ] Obtiene parámetros y llamadas
- [ ] Test en pequeño repo: `cip parse ./test`

#### Miércoles-Jueves: Dependency Graph + SQLite

**Objetivo:** Construir grafo de dependencias en SQLite

```go
// graph/builder.go

type CodeGraph struct {
    DB *sql.DB
}

type Node struct {
    ID        string
    Name      string
    Type      string
    File      string
    Line      int
    Body      string
    Signature string
}

type Edge struct {
    Source string
    Target string
    Type   string // "calls", "imports", "depends_on"
}

func (g *CodeGraph) AddNode(node Node) error {
    query := `
    INSERT INTO nodes (id, name, type, file, line, body, signature)
    VALUES (?, ?, ?, ?, ?, ?, ?)
    `
    _, err := g.DB.Exec(query, 
        node.ID, node.Name, node.Type, node.File, node.Line, node.Body, node.Signature)
    return err
}

func (g *CodeGraph) AddEdge(source, target, edgeType string) error {
    query := `
    INSERT INTO edges (source, target, type)
    VALUES (?, ?, ?)
    `
    _, err := g.DB.Exec(query, source, target, edgeType)
    return err
}

func BuildGraph(parsedFiles map[string]*ParsedFile) (*CodeGraph, error) {
    // 1. Create SQLite DB
    db, err := sql.Open("sqlite3", "code_graph.db")
    if err != nil {
        return nil, err
    }
    
    graph := &CodeGraph{DB: db}
    
    // 2. Create schema
    graph.createSchema()
    
    // 3. Add all nodes
    for filepath, parsed := range parsedFiles {
        for _, elem := range parsed.Elements {
            graph.AddNode(Node{
                ID:        elem.ID,
                Name:      elem.Name,
                Type:      elem.Type,
                File:      filepath,
                Line:      elem.Line,
                Body:      elem.Body,
                Signature: elem.Signature,
            })
        }
    }
    
    // 4. Connect edges
    for filepath, parsed := range parsedFiles {
        for _, elem := range parsed.Elements {
            for _, callee := range elem.CallsTo {
                graph.AddEdge(elem.ID, callee, "calls")
            }
        }
    }
    
    // 5. Calculate PageRank, blast radius
    graph.enrichMetrics()
    
    return graph, nil
}

func (g *CodeGraph) enrichMetrics() {
    // Calculate PageRank for importance ranking
    // Calculate blast radius: cuántos nodos se afectan si cambio uno
    // etc.
}

func (g *CodeGraph) createSchema() {
    schema := `
    CREATE TABLE IF NOT EXISTS nodes (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL,
        type TEXT,
        file TEXT,
        line INTEGER,
        body TEXT,
        signature TEXT,
        pagerank REAL DEFAULT 0,
        blast_radius INTEGER DEFAULT 0,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    
    CREATE TABLE IF NOT EXISTS edges (
        source TEXT NOT NULL,
        target TEXT NOT NULL,
        type TEXT,
        PRIMARY KEY (source, target, type),
        FOREIGN KEY (source) REFERENCES nodes(id),
        FOREIGN KEY (target) REFERENCES nodes(id)
    );
    `
    g.DB.Exec(schema)
}
```

**Checklist:**
- [ ] SQLite schema creado
- [ ] `AddNode()` y `AddEdge()` funcionan
- [ ] `enrichMetrics()` calcula PageRank
- [ ] Test: query el grafo

#### Viernes: CLI para indexar

```go
// cmd/cli.go

func main() {
    indexCmd := flag.NewFlagSet("index", flag.ExitOnError)
    repoPath := indexCmd.String("repo", ".", "Repository path")
    
    if len(os.Args) < 2 {
        fmt.Println("Usage: cip <command>")
        fmt.Println("Commands: index, serve, export")
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
    
    var parsedFiles map[string]*ParsedFile = make(map[string]*ParsedFile)
    
    filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
        if info.IsDir() || !isCodeFile(path) {
            return nil
        }
        
        fmt.Printf("  Parsing: %s\n", path)
        parsed, _ := parser.ParseFile(path)
        parsedFiles[path] = parsed
        return nil
    })
    
    fmt.Println("Building graph...")
    graphDb, _ := graph.BuildGraph(parsedFiles)
    
    fmt.Println("✅ Done!")
}

func isCodeFile(path string) bool {
    ext := filepath.Ext(path)
    validExts := map[string]bool{
        ".js": true, ".jsx": true, ".ts": true, ".tsx": true,
        ".py": true, ".go": true,
    }
    return validExts[ext]
}
```

**Test:**
```bash
./cip index -repo ./test-backend
# Resultado: code_graph.db creado con todos los nodos/edges
```

---

### 📅 SEMANA 2: Embeddings + Frontend Básico

#### Lunes-Martes: Ollama Embeddings + LanceDB

```go
// vector/embedder.go

func GenerateEmbedding(text string) ([]float32, error) {
    payload := map[string]interface{}{
        "model": "nomic-embed-text",
        "input": text,
    }
    
    jsonData, _ := json.Marshal(payload)
    resp, _ := http.Post(
        "http://localhost:11434/api/embed",
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    
    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    return result["embedding"].([]float32), nil
}

// vector/indexer.go

func IndexVectors(graphDB *sql.DB) error {
    db := lancedb.Connect("code_vectors.lance")
    
    rows, _ := graphDB.Query("SELECT id, name, body, signature FROM nodes")
    defer rows.Close()
    
    var embeddings []map[string]interface{}
    
    for rows.Next() {
        var id, name, body, signature string
        rows.Scan(&id, &name, &body, &signature)
        
        summary := fmt.Sprintf("%s: %s", name, signature)
        vector, _ := GenerateEmbedding(summary)
        
        embeddings = append(embeddings, map[string]interface{}{
            "id":   id,
            "name": name,
            "body": body,
            "vector": vector,
        })
    }
    
    db.CreateTable("functions", embeddings)
    return nil
}
```

#### Miércoles-Jueves: Frontend Setup + File Tree

**Objetivo:** React app que muestre árbol de archivos y editor

```bash
npm create vite@latest frontend -- --template react-ts
cd frontend
npm install monaco-editor d3 konva react-konva zustand
```

**`frontend/src/App.tsx`**

```tsx
import React, { useState, useEffect } from 'react'
import FileTree from './components/FileTree'
import CodeEditor from './components/CodeEditor'
import Graph from './components/Graph'
import './App.css'

function App() {
  const [selectedFile, setSelectedFile] = useState<string | null>(null)
  const [files, setFiles] = useState<Record<string, string>>({})
  const [graph, setGraph] = useState<any>(null)

  useEffect(() => {
    // Cargar archivos del backend
    fetch('/api/files')
      .then(r => r.json())
      .then(data => setFiles(data))
  }, [])

  useEffect(() => {
    // Cargar grafo del backend
    fetch('/api/graph')
      .then(r => r.json())
      .then(data => setGraph(data))
  }, [])

  return (
    <div className="app">
      <div className="left">
        <FileTree files={files} onSelect={setSelectedFile} />
      </div>
      <div className="center">
        {selectedFile && (
          <CodeEditor file={selectedFile} initialContent={files[selectedFile]} />
        )}
      </div>
      <div className="right">
        {graph && <Graph data={graph} />}
      </div>
    </div>
  )
}

export default App
```

#### Viernes: Real-time Sync (WebSocket)

```go
// api/websocket.go

func HandleWSConnection(w http.ResponseWriter, r *http.Request) {
    conn, _ := websocket.Upgrade(w, r, nil, 1024, 1024)
    
    go func() {
        for {
            var msg map[string]interface{}
            conn.ReadJSON(&msg)
            
            // Si es edición de código
            if msg["type"] == "code_change" {
                file := msg["file"].(string)
                content := msg["content"].(string)
                
                // Guardar archivo
                os.WriteFile(file, []byte(content), 0644)
                
                // Re-parsear + actualizar grafo
                parsed, _ := parser.ParseFile(file)
                updateGraph(parsed)
                
                // Broadcast a todos los clientes
                broadcastGraphUpdate()
            }
        }
    }()
}
```

---

### 📅 SEMANA 3: Anotaciones + MCP Server

#### Lunes-Martes: Metadata Parser + Store

```go
// metadata/parser.go

type FunctionMetadata struct {
    FunctionID  string
    Deprecated  bool
    DeprecatedReason string
    Hooks       []string // ["beforeAuth", "afterSession"]
    TODO        []string // ["add rate limit"]
    Authors     []string
    Tags        []string
}

func ParseMetadata(code string) FunctionMetadata {
    // Parsear comentarios con @ markers
    // @deprecated: razón
    // @hook: hookName
    // @todo: tarea
    // @author: nombre
    
    metadata := FunctionMetadata{}
    
    lines := strings.Split(code, "\n")
    for _, line := range lines {
        if strings.Contains(line, "@deprecated") {
            metadata.Deprecated = true
            metadata.DeprecatedReason = extractValue(line)
        }
        if strings.Contains(line, "@hook") {
            metadata.Hooks = append(metadata.Hooks, extractValue(line))
        }
        if strings.Contains(line, "@todo") {
            metadata.TODO = append(metadata.TODO, extractValue(line))
        }
    }
    
    return metadata
}

// metadata/store.go

func (g *CodeGraph) SaveMetadata(funcID string, meta FunctionMetadata) error {
    query := `
    INSERT OR REPLACE INTO metadata (function_id, deprecated, reason, hooks, todos, authors)
    VALUES (?, ?, ?, ?, ?, ?)
    `
    hooks, _ := json.Marshal(meta.Hooks)
    todos, _ := json.Marshal(meta.TODO)
    authors, _ := json.Marshal(meta.Authors)
    
    _, err := g.DB.Exec(query, funcID, meta.Deprecated, meta.DeprecatedReason, 
        string(hooks), string(todos), string(authors))
    return err
}
```

#### Miércoles: MCP Server with Tools

```go
// mcp/tools.go

func SearchContext(query string, graph *CodeGraph) map[string]interface{} {
    // 1. Buscar en embeddings
    results := vectorSearch(query, 5)
    
    // 2. Obtener contexto de cada resultado
    var files []map[string]interface{}
    
    for _, result := range results {
        node := graph.GetNode(result.ID)
        metadata := graph.GetMetadata(result.ID)
        
        files = append(files, map[string]interface{}{
            "path":       node.File,
            "function":   node.Name,
            "relevance":  result.Similarity,
            "snippet":    node.Body,
            "deprecated": metadata.Deprecated,
            "hooks":      metadata.Hooks,
            "todo":       metadata.TODO,
        })
    }
    
    return map[string]interface{}{
        "files": files,
    }
}

func GetFileSmart(filename string, symbol string, graph *CodeGraph) map[string]interface{} {
    // 1. Encontrar nodo exacto
    node := graph.GetNode(filename + ":" + symbol)
    
    // 2. Extraer solo esa función
    return map[string]interface{}{
        "code":     node.Body,
        "calls":    graph.GetEdges(node.ID, "calls"),
        "calledBy": graph.GetEdges(node.ID, "calledBy"),
        "metadata": graph.GetMetadata(node.ID),
    }
}

func TraceImpact(functionID string, graph *CodeGraph) map[string]interface{} {
    // BFS desde functionID
    impacted := graph.BFS(functionID)
    
    return map[string]interface{}{
        "impactedFunctions": impacted,
        "cascadeSize":       len(impacted),
        "critical":          hasDeprecatedInPath(impacted),
    }
}

// mcp/server.go

func StartMCPServer(graph *CodeGraph) {
    srv := server.NewServer("code-intelligence")
    
    srv.AddTool(&types.Tool{
        Name:        "search_context",
        Description: "Busca archivos relevantes para una query",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "query": map[string]interface{}{
                    "type":        "string",
                    "description": "Tu pregunta sobre el código",
                },
            },
        },
        Handler: func(args map[string]interface{}) string {
            query := args["query"].(string)
            result := SearchContext(query, graph)
            data, _ := json.Marshal(result)
            return string(data)
        },
    })
    
    srv.AddTool(&types.Tool{
        Name:        "get_file_smart",
        Description: "Obtiene una función específica sin código innecesario",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "filename": map[string]interface{}{"type": "string"},
                "symbol":   map[string]interface{}{"type": "string"},
            },
        },
        Handler: func(args map[string]interface{}) string {
            filename := args["filename"].(string)
            symbol := args["symbol"].(string)
            result := GetFileSmart(filename, symbol, graph)
            data, _ := json.Marshal(result)
            return string(data)
        },
    })
    
    srv.AddTool(&types.Tool{
        Name:        "trace_impact",
        Description: "Muestra qué se afecta si cambio una función",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "function": map[string]interface{}{"type": "string"},
            },
        },
        Handler: func(args map[string]interface{}) string {
            function := args["function"].(string)
            result := TraceImpact(function, graph)
            data, _ := json.Marshal(result)
            return string(data)
        },
    })
    
    srv.Start(":8000")
}
```

#### Jueves-Viernes: Annotations UI + Polish

Frontend components para visualizar y editar metadata:

```tsx
// frontend/src/components/AnnotationsSidebar.tsx

export function AnnotationsSidebar({ selectedFunction }: Props) {
  const [metadata, setMetadata] = useState<FunctionMetadata>()

  const toggleDeprecated = () => {
    const updated = { ...metadata, deprecated: !metadata.deprecated }
    setMetadata(updated)
    saveMetadata(selectedFunction, updated)
  }

  const addHook = (hook: string) => {
    const updated = {
      ...metadata,
      hooks: [...metadata.hooks, hook]
    }
    setMetadata(updated)
    saveMetadata(selectedFunction, updated)
  }

  return (
    <div className="sidebar">
      <h2>{selectedFunction}</h2>
      
      <section>
        <h3>Deprecated</h3>
        <input
          type="checkbox"
          checked={metadata?.deprecated}
          onChange={toggleDeprecated}
        />
        {metadata?.deprecated && (
          <input
            type="text"
            placeholder="Reason"
            value={metadata.deprecatedReason}
            onChange={e => /* update */}
          />
        )}
      </section>

      <section>
        <h3>Hooks</h3>
        {metadata?.hooks.map(hook => (
          <div key={hook} className="tag">
            {hook}
          </div>
        ))}
        <input
          type="text"
          placeholder="Add hook"
          onKeyDown={e => e.key === 'Enter' && addHook(e.currentTarget.value)}
        />
      </section>

      <section>
        <h3>TODO</h3>
        {metadata?.todo.map(t => (
          <div key={t} className="todo">{t}</div>
        ))}
      </section>
    </div>
  )
}
```

---

## Componentes Detallados

### Backend: Parser

**Entrada:**
```
/my-backend/
├── auth/
│  ├── login.ts (200 líneas)
│  └── validate.ts (150 líneas)
├── db/
│  └── user.ts (300 líneas)
└── crypto/
   └── jwt.ts (250 líneas)
```

**Salida:**
```json
{
  "nodes": [
    {
      "id": "auth/login.ts:loginHandler",
      "name": "loginHandler",
      "type": "function",
      "file": "auth/login.ts",
      "line": 12,
      "signature": "function loginHandler(req: Request)",
      "body": "{ ... }",
      "pagerank": 0.35
    },
    {
      "id": "auth/validate.ts:validatePassword",
      "name": "validatePassword",
      "type": "function",
      "file": "auth/validate.ts",
      "line": 5,
      "pagerank": 0.28
    }
  ],
  "edges": [
    {
      "source": "auth/login.ts:loginHandler",
      "target": "auth/validate.ts:validatePassword",
      "type": "calls"
    }
  ]
}
```

### Frontend: Editor

**Características:**
1. **Árbol de archivos** (izquierda)
   - Click en archivo → abre en editor
   - Click en función → scrollea a esa línea
   - Indicadores: [!] hook, [X] deprecated, [○] todo

2. **Editor de código** (centro)
   - Monaco editor
   - Syntax highlighting
   - Auto-save cada 2 segundos
   - Cambios se sincronizan al backend

3. **Grafo visual** (derecha)
   - Nodos = funciones
   - Edges = dependencias
   - Drag-drop para crear relaciones
   - Colores según estado (deprecated, hooks, etc.)
   - Click en nodo → abre en editor

4. **Sidebar de anotaciones** (oculto, click para abrir)
   - Metadata de función seleccionada
   - Checkboxes para deprecated, hooks, todos
   - Input para agregar tags/authors

### MCP Server: Tools para Claude Code

**Tool 1: search_context**
```
Input:  { query: "¿Cómo funciona authentication?" }
Output: {
  files: [
    {
      path: "auth/login.ts",
      function: "loginHandler",
      relevance: 0.98,
      snippet: "function loginHandler(req) { ... }",
      deprecated: true,
      hooks: ["beforeAuth"],
      todo: ["add rate limiting"]
    }
  ]
}
```

**Tool 2: get_file_smart**
```
Input:  { filename: "auth/login.ts", symbol: "loginHandler" }
Output: {
  code: "function loginHandler(req) { ... }",
  calls: ["validatePassword", "createToken"],
  calledBy: ["api/routes.ts::POST /auth/login"],
  metadata: {
    deprecated: true,
    hooks: ["beforeAuth"],
    todo: ["add rate limiting"]
  }
}
```

**Tool 3: trace_impact**
```
Input:  { function: "validatePassword" }
Output: {
  impactedFunctions: [
    "loginHandler",
    "resetPassword",
    "api/routes.ts::POST /auth/login"
  ],
  cascadeSize: 3,
  critical: false
}
```

---

## Workflow del Usuario

### Día 0: Setup

```bash
# 1. Clonar
git clone https://github.com/ruffini/code-intelligence
cd code-intelligence

# 2. Backend
go mod download
go run cmd/main.go serve

# 3. Frontend
cd frontend && npm install && npm run dev

# 4. Ollama (otra terminal)
ollama pull nomic-embed-text
ollama serve

# 5. Abrir http://localhost:5173
```

### Día 1: Indexar Backend

```bash
# Terminal 1
./cip index -repo /path/to/onekin-backend

# Resultado:
# - code_graph.db creado
# - Vectores generados
# - Frontend se actualiza automáticamente
```

### Día 2-5: Refinar Anotaciones

En editor visual:

```
1. Abre auth/login.ts
2. Ve función loginHandler
3. Sidebar derecha → marca @deprecated
4. Input razón: "migrar a OAuth2"
5. Agrega @hook: "beforeAuth"
6. Click en nodo del grafo → arrastra a validatePassword
   → Crea relación visual
7. Cambios se guardan en DB automáticamente
```

### Semana 2+: Usar en Claude Code

```
En Claude Code:

Pregunta: "¿Cuál es el flujo de autenticación completo?"

Claude Code:
1. Llama tool: search_context("authentication flow")
2. Backend busca en embeddings
3. Devuelve: login.ts, validate.ts, createToken.ts
4. Plus metadata: deprecated notices, hooks, todos
5. Lee solo 3 archivos = 5K tokens (vs 50K sin herramienta)
6. Responde: "Actualmente login() → validate() → createToken()
              pero está deprecado porque X. Nueva versión 
              debe usar OAuth2. TODO: agregar rate limiting."
```

---

## MCP Tools para Claude Code

### Installation

Crear `.claude/settings.json`:

```json
{
  "mcpServers": {
    "code-intel": {
      "command": "/path/to/cip",
      "args": ["serve"]
    }
  }
}
```

Luego en Claude Code, tienes access a:

- `search_context(query)` → busca en grafo + embeddings
- `get_file_smart(filename, symbol)` → extrae solo símbolo necesario
- `trace_impact(function)` → muestra cascada de cambios
- `list_deprecated()` → todas las funciones deprecated
- `get_annotations(function)` → metadata completa

---

## Diferenciales vs Soulforge

| Feature | Soulforge | Code Intelligence ✅ |
|---------|-----------|---------------------|
| **Auto-parse** | ✅ CLI | ✅ CLI + UI |
| **Visual editor** | ❌ | ✅ |
| **Annotations** | ❌ | ✅ |
| **Deprecated tracking** | ❌ | ✅ |
| **Hook management** | ❌ | ✅ |
| **TODO tracking** | ❌ | ✅ |
| **Drag-drop graph** | ❌ | ✅ |
| **Manual relations** | ❌ | ✅ |
| **Token optimization** | ✅ | ✅ |
| **License** | BSL 1.1 | Apache 2.0 |
| **Multi-repo** | ❌ | 🟡 (roadmap) |

---

## Roadmap (después de MVP)

**Mes 2:**
- [ ] Multi-repo support (grafo unificado)
- [ ] Export a Markdown/Diagrams
- [ ] Collaboration (multi-user editing)
- [ ] Git integration (track changes)

**Mes 3:**
- [ ] Cloud deployment
- [ ] API rate limiting
- [ ] Advanced analytics

---

## Stack Final

**Backend:**
- Go (parser, graph, vector, mcp)
- SQLite (graph storage)
- LanceDB (vector DB)
- Ollama (local embeddings)

**Frontend:**
- React + TypeScript
- Monaco (editor)
- D3 (graph visualization)
- Zustand (state management)
- Vite (bundler)

**Total:** ~5000 lines of code

---

## ¿Por qué es PERFECTO?

✅ **Único:** Nadie combina auto-parse + visual editor + anotaciones + embeddings
✅ **Práctico:** ONEKIN lo usaría día 1 para documentar repos
✅ **Visual:** Mejor que Soulforge (que es CLI-only)
✅ **Flexible:** Funciona con JS/TS/Python/Go (tree-sitter)
✅ **Colaborativo:** Potencial para multi-user
✅ **Académico:** "Interactive code graph composition with semantic search"
✅ **Open source:** Apache 2.0 desde day 1
✅ **Diferencial:** No es "copiar Soulforge", es "mejorar la experiencia"

---

**¿Empezamos?**

Semana 1 es puramente backend. Semana 2-3 es UI + integration.

Next step: crear repo en GitHub + primer commit de scaffolding.
