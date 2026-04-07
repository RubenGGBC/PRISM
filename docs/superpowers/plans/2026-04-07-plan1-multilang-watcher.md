# Plan 1: Multi-language Parser (Generic Interface) + File Watcher

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reemplazar los parsers específicos por idioma con una única interfaz genérica tree-sitter configurable. Añadir soporte para Go, Rust, Java (y mantener Python/TypeScript). Añadir reindexado automático con file watcher.

**Architecture:** Un único `parser/treesitter.go` define `TreeSitterParser` que toma un `LanguageConfig` — una struct de configuración con el lenguaje tree-sitter, extensiones, y los nombres de nodos AST que representan funciones/clases/imports. `parser/languages.go` registra todos los lenguajes. `parser/parser.go` expone `GetParser(filename)` que consulta el registro. El file watcher vive en `watcher/watcher.go` y se activa con `prism serve --watch`.

**Tech Stack:** Go, tree-sitter (go-tree-sitter), fsnotify

---

## Archivos que se tocan

| Acción | Archivo |
|--------|---------|
| Modify | `go.mod` |
| Create | `parser/treesitter.go` |
| Create | `parser/languages.go` |
| Modify | `parser/parser.go` |
| Delete | `parser/python.go` (migrar a config en languages.go) |
| Delete | `parser/typescript.go` (migrar a config en languages.go) |
| Create | `watcher/watcher.go` |
| Modify | `main.go` |
| Create | `tests/parser_test.go` |

---

### Task 1: Añadir dependencias

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Añadir grammars y fsnotify**

```bash
go get github.com/tree-sitter/tree-sitter-go/bindings/go@latest
go get github.com/tree-sitter/tree-sitter-rust/bindings/go@latest
go get github.com/tree-sitter/tree-sitter-java/bindings/go@latest
go get github.com/fsnotify/fsnotify@latest
go mod tidy
```

- [ ] **Step 2: Verificar que compila**

```bash
go build ./...
```

Expected: sin errores (aún sin usar los imports, puede que haya warning de unused — está bien por ahora).

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "feat: add tree-sitter grammars for Go, Rust, Java and fsnotify"
```

---

### Task 2: Parser genérico tree-sitter

**Files:**
- Create: `parser/treesitter.go`

Este parser implementa la interfaz `Parser` existente usando una `LanguageConfig` que describe los tipos de nodos AST relevantes por lenguaje. No hay código específico por idioma aquí — solo el mecanismo genérico.

- [ ] **Step 1: Crear `parser/treesitter.go`**

```go
package parser

import (
	"fmt"
	"os"
	"strings"
	"time"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// NodeKindConfig describe cómo extraer un tipo de elemento del AST
type NodeKindConfig struct {
	Kind          string // tipo de nodo AST (e.g. "function_declaration")
	NameField     string // campo del nodo que contiene el nombre (e.g. "name")
	ElementType   string // tipo semántico resultante: "function", "method", "class", "struct", "interface"
	ReceiverField string // campo del receptor para métodos con receiver (Go: "receiver"), vacío si no aplica
	ParentKind    string // tipo del nodo padre que indica que es un método (e.g. "impl_item" en Rust), vacío si no aplica
	BodyField     string // campo del body para extraer signature sin body (e.g. "body")
}

// ImportKindConfig describe cómo encontrar imports en el AST
type ImportKindConfig struct {
	Kind       string // tipo de nodo (e.g. "import_declaration")
	PathField  string // campo con el path del módulo
	AliasField string // campo con el alias, vacío si no existe
}

// LanguageConfig configura el parser genérico para un lenguaje
type LanguageConfig struct {
	Name        string
	Extensions  []string
	Language    *sitter.Language
	NodeKinds   []NodeKindConfig
	ImportKinds []ImportKindConfig
	DocComment  string // "prev_sibling" (Go/Rust) o "first_child_string" (Python/JS docstrings)
}

// TreeSitterParser implementa Parser para cualquier lenguaje configurado via LanguageConfig
type TreeSitterParser struct {
	config LanguageConfig
}

// NewTreeSitterParser crea un parser para el lenguaje dado
func NewTreeSitterParser(config LanguageConfig) *TreeSitterParser {
	return &TreeSitterParser{config: config}
}

func (p *TreeSitterParser) Language() string    { return p.config.Name }
func (p *TreeSitterParser) Extensions() []string { return p.config.Extensions }

func (p *TreeSitterParser) ParseFile(fp string) (*ParsedFile, error) {
	source, err := os.ReadFile(fp)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return p.Parse(source, fp)
}

func (p *TreeSitterParser) Parse(source []byte, filename string) (*ParsedFile, error) {
	start := time.Now()

	par := sitter.NewParser()
	defer par.Close()

	if err := par.SetLanguage(p.config.Language); err != nil {
		return nil, fmt.Errorf("failed to set language: %w", err)
	}

	tree := par.Parse(source, nil)
	defer tree.Close()

	parsed := &ParsedFile{
		Path:     filename,
		Language: p.config.Name,
		Elements: []CodeElement{},
		Imports:  []Import{},
	}

	root := tree.RootNode()
	parsed.Imports = p.extractImports(root, source)
	p.walkNode(root, source, filename, parsed, "")
	parsed.ParseTime = time.Since(start).Milliseconds()
	return parsed, nil
}

func (p *TreeSitterParser) walkNode(node *sitter.Node, source []byte, filename string, parsed *ParsedFile, implType string) {
	kind := node.Kind()

	// Detectar contexto de impl/class que da nombre a métodos hijos
	currentImpl := implType
	for _, nc := range p.config.NodeKinds {
		if nc.ParentKind != "" && kind == nc.ParentKind {
			// Extraer el nombre del tipo contenedor (e.g. el tipo en "impl Foo {")
			if nameNode := node.ChildByFieldName("name"); nameNode != nil {
				currentImpl = string(source[nameNode.StartByte():nameNode.EndByte()])
			} else if nameNode := node.ChildByFieldName("type"); nameNode != nil {
				currentImpl = strings.TrimPrefix(
					strings.TrimSpace(string(source[nameNode.StartByte():nameNode.EndByte()])),
					"*",
				)
			}
		}
	}

	// Intentar extraer elemento según los NodeKinds configurados
	for _, nc := range p.config.NodeKinds {
		if kind != nc.Kind {
			continue
		}
		elem := p.extractElement(node, source, filename, nc, currentImpl)
		if elem != nil {
			parsed.Elements = append(parsed.Elements, *elem)
		}
		// No recursar dentro del cuerpo de funciones (evita funciones anidadas duplicadas)
		return
	}

	// Recursar
	count := int(node.ChildCount())
	for i := 0; i < count; i++ {
		if child := node.Child(uint(i)); child != nil {
			p.walkNode(child, source, filename, parsed, currentImpl)
		}
	}
}

func (p *TreeSitterParser) extractElement(node *sitter.Node, source []byte, filename string, nc NodeKindConfig, implType string) *CodeElement {
	// Obtener nombre
	var name string
	if nc.NameField != "" {
		if nameNode := node.ChildByFieldName(nc.NameField); nameNode != nil {
			name = string(source[nameNode.StartByte():nameNode.EndByte()])
		}
	}
	if name == "" {
		return nil
	}

	// Obtener receiver para métodos con receiver explícito (Go)
	if nc.ReceiverField != "" {
		if recvNode := node.ChildByFieldName(nc.ReceiverField); recvNode != nil {
			recvText := string(source[recvNode.StartByte():recvNode.EndByte()])
			recvText = strings.Trim(recvText, "()")
			parts := strings.Fields(recvText)
			for _, part := range parts {
				clean := strings.Trim(part, "*[]")
				if clean != "" && !strings.ContainsAny(clean, "()") {
					implType = clean
					break
				}
			}
		}
	}

	fullName := name
	if implType != "" && nc.ElementType == "method" {
		fullName = implType + "." + name
	}

	id := fmt.Sprintf("%s:%s", filename, fullName)

	// Signature (hasta el body)
	sig := p.extractSignature(node, source, nc.BodyField)

	// Body
	body := p.extractBody(node, source, nc.BodyField)

	// Doc comment
	docstring := p.extractDoc(node, source)

	// Calls
	calls := p.extractCalls(node, source)

	return &CodeElement{
		ID:        id,
		Name:      fullName,
		Type:      nc.ElementType,
		File:      filename,
		Language:  p.config.Name,
		Line:      int(node.StartPosition().Row) + 1,
		EndLine:   int(node.EndPosition().Row) + 1,
		Signature: sig,
		Body:      body,
		DocString: docstring,
		CallsTo:   calls,
	}
}

func (p *TreeSitterParser) extractSignature(node *sitter.Node, source []byte, bodyField string) string {
	var end uint
	if bodyField != "" {
		if bodyNode := node.ChildByFieldName(bodyField); bodyNode != nil {
			end = bodyNode.StartByte()
		}
	}
	if end == 0 {
		end = node.EndByte()
	}
	sig := strings.TrimSpace(string(source[node.StartByte():end]))
	if len(sig) > 300 {
		sig = sig[:300]
	}
	return sig
}

func (p *TreeSitterParser) extractBody(node *sitter.Node, source []byte, bodyField string) string {
	if bodyField == "" {
		return ""
	}
	bodyNode := node.ChildByFieldName(bodyField)
	if bodyNode == nil {
		return ""
	}
	body := string(source[bodyNode.StartByte():bodyNode.EndByte()])
	if len(body) > 1000 {
		body = body[:1000] + "..."
	}
	return body
}

func (p *TreeSitterParser) extractDoc(node *sitter.Node, source []byte) string {
	switch p.config.DocComment {
	case "prev_sibling":
		prev := node.PrevNamedSibling()
		if prev != nil && strings.Contains(prev.Kind(), "comment") {
			return strings.TrimSpace(string(source[prev.StartByte():prev.EndByte()]))
		}
	case "first_child_string":
		// Python/JS docstrings: primer hijo del body es una string
		for _, nc := range p.config.NodeKinds {
			if bodyNode := node.ChildByFieldName(nc.BodyField); bodyNode != nil {
				if bodyNode.ChildCount() > 0 {
					first := bodyNode.Child(0)
					if first != nil {
						txt := string(source[first.StartByte():first.EndByte()])
						if strings.HasPrefix(txt, `"`) || strings.HasPrefix(txt, `'`) {
							return strings.Trim(strings.TrimSpace(txt), `"'`)
						}
					}
				}
				break
			}
		}
	}
	return ""
}

func (p *TreeSitterParser) extractCalls(node *sitter.Node, source []byte) []string {
	seen := make(map[string]bool)
	var calls []string
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		// call_expression es el tipo en Go/Rust/TS; call_expression o "call" en Python
		if n.Kind() == "call_expression" || n.Kind() == "call" {
			var funcNode *sitter.Node
			if f := n.ChildByFieldName("function"); f != nil {
				funcNode = f
			} else if n.ChildCount() > 0 {
				funcNode = n.Child(0)
			}
			if funcNode != nil {
				call := strings.TrimSpace(string(source[funcNode.StartByte():funcNode.EndByte()]))
				// Tomar solo el último segmento (obj.Method → Method)
				if idx := strings.LastIndexAny(call, ".::"); idx >= 0 {
					call = call[idx+1:]
				}
				if call != "" && !seen[call] {
					seen[call] = true
					calls = append(calls, call)
				}
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			if child := n.Child(uint(i)); child != nil {
				walk(child)
			}
		}
	}
	walk(node)
	return calls
}

func (p *TreeSitterParser) extractImports(root *sitter.Node, source []byte) []Import {
	var imports []Import
	kindSet := make(map[string]ImportKindConfig)
	for _, ik := range p.config.ImportKinds {
		kindSet[ik.Kind] = ik
	}

	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if ik, ok := kindSet[n.Kind()]; ok {
			imp := Import{Line: int(n.StartPosition().Row) + 1}
			if ik.PathField != "" {
				if pathNode := n.ChildByFieldName(ik.PathField); pathNode != nil {
					imp.Module = strings.Trim(
						string(source[pathNode.StartByte():pathNode.EndByte()]), `"`)
				}
			}
			if ik.AliasField != "" {
				if aliasNode := n.ChildByFieldName(ik.AliasField); aliasNode != nil {
					imp.Alias = string(source[aliasNode.StartByte():aliasNode.EndByte()])
				}
			}
			if imp.Module != "" {
				imports = append(imports, imp)
			}
			return
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			if child := n.Child(uint(i)); child != nil {
				walk(child)
			}
		}
	}
	walk(root)
	return imports
}
```

- [ ] **Step 2: Verificar que compila**

```bash
go build ./parser/...
```

Expected: sin errores.

- [ ] **Step 3: Commit**

```bash
git add parser/treesitter.go
git commit -m "feat: add generic TreeSitterParser with LanguageConfig"
```

---

### Task 3: Registro de lenguajes

**Files:**
- Create: `parser/languages.go`

Este archivo registra todos los lenguajes. Para añadir un nuevo lenguaje basta añadir una entrada aquí.

- [ ] **Step 1: Crear `parser/languages.go`**

```go
package parser

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	golang "github.com/tree-sitter/tree-sitter-go/bindings/go"
	java "github.com/tree-sitter/tree-sitter-java/bindings/go"
	python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	rust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// languageRegistry mapea extensión → LanguageConfig
var languageRegistry = map[string]LanguageConfig{}

func init() {
	configs := []LanguageConfig{
		goConfig(),
		rustConfig(),
		javaConfig(),
		pythonConfig(),
		typescriptConfig(".ts"),
		typescriptConfig(".tsx"),
		typescriptConfig(".js"),
		typescriptConfig(".jsx"),
	}
	for _, c := range configs {
		for _, ext := range c.Extensions {
			languageRegistry[ext] = c
		}
	}
}

func goConfig() LanguageConfig {
	return LanguageConfig{
		Name:       "go",
		Extensions: []string{".go"},
		Language:   sitter.NewLanguage(golang.Language()),
		DocComment: "prev_sibling",
		NodeKinds: []NodeKindConfig{
			{Kind: "function_declaration", NameField: "name", ElementType: "function", BodyField: "body"},
			{Kind: "method_declaration", NameField: "name", ElementType: "method", ReceiverField: "receiver", BodyField: "body"},
			{Kind: "type_spec", NameField: "name", ElementType: "struct", BodyField: ""},
		},
		ImportKinds: []ImportKindConfig{
			{Kind: "import_spec", PathField: "path", AliasField: "name"},
		},
	}
}

func rustConfig() LanguageConfig {
	return LanguageConfig{
		Name:       "rust",
		Extensions: []string{".rs"},
		Language:   sitter.NewLanguage(rust.Language()),
		DocComment: "prev_sibling",
		NodeKinds: []NodeKindConfig{
			{Kind: "function_item", NameField: "name", ElementType: "function", BodyField: "body"},
			// Métodos dentro de impl_item: el ParentKind "impl_item" actualiza el implType en el walker
			{Kind: "function_item", NameField: "name", ElementType: "method", ParentKind: "impl_item", BodyField: "body"},
			{Kind: "struct_item", NameField: "name", ElementType: "struct", BodyField: "body"},
			{Kind: "enum_item", NameField: "name", ElementType: "class", BodyField: "body"},
			{Kind: "trait_item", NameField: "name", ElementType: "interface", BodyField: "body"},
		},
		ImportKinds: []ImportKindConfig{
			{Kind: "use_declaration", PathField: "argument"},
		},
	}
}

func javaConfig() LanguageConfig {
	return LanguageConfig{
		Name:       "java",
		Extensions: []string{".java"},
		Language:   sitter.NewLanguage(java.Language()),
		DocComment: "prev_sibling",
		NodeKinds: []NodeKindConfig{
			{Kind: "method_declaration", NameField: "name", ElementType: "method", BodyField: "body"},
			{Kind: "constructor_declaration", NameField: "name", ElementType: "function", BodyField: "body"},
			{Kind: "class_declaration", NameField: "name", ElementType: "class", BodyField: "body"},
			{Kind: "interface_declaration", NameField: "name", ElementType: "interface", BodyField: "body"},
		},
		ImportKinds: []ImportKindConfig{
			{Kind: "import_declaration", PathField: "name"},
		},
	}
}

func pythonConfig() LanguageConfig {
	return LanguageConfig{
		Name:       "python",
		Extensions: []string{".py"},
		Language:   sitter.NewLanguage(python.Language()),
		DocComment: "first_child_string",
		NodeKinds: []NodeKindConfig{
			{Kind: "function_definition", NameField: "name", ElementType: "function", BodyField: "body"},
			{Kind: "class_definition", NameField: "name", ElementType: "class", BodyField: "body"},
		},
		ImportKinds: []ImportKindConfig{
			{Kind: "import_statement", PathField: "name"},
			{Kind: "import_from_statement", PathField: "module_name"},
		},
	}
}

func typescriptConfig(ext string) LanguageConfig {
	lang := sitter.NewLanguage(typescript.LanguageTypescript())
	name := "typescript"
	if ext == ".tsx" {
		lang = sitter.NewLanguage(typescript.LanguageTSX())
		name = "tsx"
	} else if ext == ".js" || ext == ".jsx" {
		name = "javascript"
	}
	return LanguageConfig{
		Name:       name,
		Extensions: []string{ext},
		Language:   lang,
		DocComment: "first_child_string",
		NodeKinds: []NodeKindConfig{
			{Kind: "function_declaration", NameField: "name", ElementType: "function", BodyField: "body"},
			{Kind: "arrow_function", NameField: "name", ElementType: "function", BodyField: "body"},
			{Kind: "method_definition", NameField: "name", ElementType: "method", BodyField: "body"},
			{Kind: "class_declaration", NameField: "name", ElementType: "class", BodyField: "body"},
			{Kind: "interface_declaration", NameField: "name", ElementType: "interface", BodyField: "body"},
		},
		ImportKinds: []ImportKindConfig{
			{Kind: "import_statement", PathField: "source"},
		},
	}
}
```

- [ ] **Step 2: Compilar**

```bash
go build ./parser/...
```

Expected: sin errores.

- [ ] **Step 3: Commit**

```bash
git add parser/languages.go
git commit -m "feat: add language registry with Go, Rust, Java, Python, TypeScript configs"
```

---

### Task 4: Actualizar parser.go y eliminar parsers específicos

**Files:**
- Modify: `parser/parser.go`
- Delete: `parser/python.go`
- Delete: `parser/typescript.go`

- [ ] **Step 1: Reemplazar el contenido de `parser/parser.go`**

```go
package parser

import (
	"path/filepath"
	"strings"
)

// Parser interface para diferentes lenguajes
type Parser interface {
	Parse(source []byte, filename string) (*ParsedFile, error)
	Language() string
	Extensions() []string
}

// GetParser devuelve el parser adecuado para un archivo, o nil si no se soporta
func GetParser(filename string) Parser {
	ext := strings.ToLower(filepath.Ext(filename))
	if config, ok := languageRegistry[ext]; ok {
		return NewTreeSitterParser(config)
	}
	return nil
}

// DetectLanguage detecta el lenguaje basándose en la extensión del archivo
func DetectLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if config, ok := languageRegistry[ext]; ok {
		return config.Name
	}
	return "unknown"
}

// IsCodeFile verifica si el archivo es código parseable
func IsCodeFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	_, ok := languageRegistry[ext]
	return ok
}

// ShouldSkipPath verifica si debemos saltar este path
func ShouldSkipPath(path string) bool {
	skipDirs := []string{
		"node_modules",
		"__pycache__",
		".git",
		".venv",
		"venv",
		"env",
		"dist",
		"build",
		".next",
		".nuxt",
		"coverage",
		".pytest_cache",
		".mypy_cache",
	}

	pathLower := strings.ToLower(path)
	for _, skip := range skipDirs {
		if strings.Contains(pathLower, skip) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Eliminar los parsers específicos**

```bash
rm parser/python.go parser/typescript.go
```

- [ ] **Step 3: Verificar que nada usa NewPythonParser/NewTypeScriptParser directamente**

```bash
grep -r "NewPythonParser\|NewTypeScriptParser\|NewGoParser\|NewRustParser\|NewJavaParser" --include="*.go" .
```

Expected: 0 resultados. Si hay resultados, reemplazarlos por `parser.GetParser(filename)`.

- [ ] **Step 4: Compilar todo el proyecto**

```bash
go build ./...
```

Expected: sin errores.

- [ ] **Step 5: Commit**

```bash
git add parser/parser.go
git rm parser/python.go parser/typescript.go
git commit -m "feat: replace per-language parsers with generic TreeSitterParser registry"
```

---

### Task 5: Actualizar main.go para usar GetParser

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Verificar cómo se usan los parsers en main.go**

```bash
grep -n "PythonParser\|TypeScriptParser\|parser\." main.go | head -20
```

- [ ] **Step 2: Reemplazar la selección de parser por GetParser**

Buscar el bloque que selecciona el parser según el lenguaje (probablemente algo como):
```go
switch lang {
case "python":
    p = parser.NewPythonParser()
case "typescript", "javascript":
    p = parser.NewTypeScriptParser()
}
```

Reemplazarlo por:
```go
p := parser.GetParser(filePath)
if p == nil {
    continue // lenguaje no soportado
}
```

- [ ] **Step 3: Compilar**

```bash
go build ./...
```

Expected: sin errores.

- [ ] **Step 4: Probar indexado**

```bash
go run . index -repo test/sample-repo
```

Expected: indexa archivos .ts sin errores.

- [ ] **Step 5: Commit**

```bash
git add main.go
git commit -m "feat: use parser.GetParser in indexer — supports all registered languages"
```

---

### Task 6: Tests del parser genérico

**Files:**
- Create: `tests/parser_test.go`

- [ ] **Step 1: Crear tests para Go, Rust, Python (los 3 nuevos o migrados)**

Crear `tests/parser_test.go`:
```go
package tests

import (
	"os"
	"testing"

	"github.com/ruffini/prism/parser"
)

func TestGoParser(t *testing.T) {
	f := writeTempFile(t, "*.go", []byte(`package main

import "fmt"

type Server struct {
	port int
}

func NewServer(port int) *Server {
	return &Server{port: port}
}

func (s *Server) Start() {
	fmt.Println("started")
}
`))
	result, err := parser.GetParser(f).ParseFile(f)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if result.Language != "go" {
		t.Errorf("expected language=go, got %s", result.Language)
	}
	assertElementFound(t, result, "NewServer")
	assertElementFound(t, result, "Server.Start")
}

func TestRustParser(t *testing.T) {
	f := writeTempFile(t, "*.rs", []byte(`use std::io;

pub struct Config {
    pub port: u16,
}

pub fn new_config(port: u16) -> Config {
    Config { port }
}

impl Config {
    pub fn start(&self) {
        println!("started");
    }
}
`))
	result, err := parser.GetParser(f).ParseFile(f)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if result.Language != "rust" {
		t.Errorf("expected language=rust, got %s", result.Language)
	}
	assertElementFound(t, result, "new_config")
	assertElementFound(t, result, "Config")
}

func TestPythonParser(t *testing.T) {
	f := writeTempFile(t, "*.py", []byte(`class Greeter:
    def greet(self, name: str) -> str:
        return f"hello {name}"

def standalone():
    pass
`))
	result, err := parser.GetParser(f).ParseFile(f)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if result.Language != "python" {
		t.Errorf("expected language=python, got %s", result.Language)
	}
	assertElementFound(t, result, "Greeter")
	assertElementFound(t, result, "standalone")
}

func TestJavaParser(t *testing.T) {
	f := writeTempFile(t, "*.java", []byte(`public class Calculator {
    public int add(int a, int b) {
        return a + b;
    }
}
`))
	result, err := parser.GetParser(f).ParseFile(f)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if result.Language != "java" {
		t.Errorf("expected language=java, got %s", result.Language)
	}
	assertElementFound(t, result, "Calculator")
	assertElementFound(t, result, "add")
}

func TestGetParserReturnsNilForUnknown(t *testing.T) {
	p := parser.GetParser("somefile.xyz")
	if p != nil {
		t.Error("expected nil parser for unknown extension")
	}
}

// helpers

func writeTempFile(t *testing.T, pattern string, content []byte) string {
	t.Helper()
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		t.Fatal(err)
	}
	f.Write(content)
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func assertElementFound(t *testing.T, result *parser.ParsedFile, name string) {
	t.Helper()
	for _, e := range result.Elements {
		if e.Name == name {
			return
		}
	}
	t.Errorf("expected to find element %q; got: %v", name, elementNames(result))
}

func elementNames(result *parser.ParsedFile) []string {
	names := make([]string, len(result.Elements))
	for i, e := range result.Elements {
		names[i] = e.Name
	}
	return names
}
```

- [ ] **Step 2: Ejecutar los tests**

```bash
go test ./tests/... -run "TestGoParser|TestRustParser|TestPythonParser|TestJavaParser|TestGetParser" -v
```

Expected: todos PASS. Si algún test falla por nombre de campo AST incorrecto (diferente versión de gramática), ajustar el `NameField`/`BodyField` en `languages.go` según el mensaje de error.

- [ ] **Step 3: Commit**

```bash
git add tests/parser_test.go
git commit -m "test: add generic parser tests for Go, Rust, Python, Java"
```

---

### Task 7: File Watcher

**Files:**
- Create: `watcher/watcher.go`
- Modify: `main.go`

- [ ] **Step 1: Crear `watcher/watcher.go`**

```go
package watcher

import (
	"log"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ruffini/prism/parser"
)

// IndexFn es la función que reindexea un único archivo
type IndexFn func(path string) error

// RemoveFn elimina los nodos de un archivo del grafo
type RemoveFn func(path string) error

// Watch observa el directorio root y llama a indexFn/removeFn ante cambios.
// Bloquea hasta que se cierre done.
func Watch(root string, indexFn IndexFn, removeFn RemoveFn, done <-chan struct{}) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()

	if err := w.Add(root); err != nil {
		return err
	}

	log.Printf("👁️  Watching %s for changes...", root)

	debounce := make(map[string]*time.Timer)

	for {
		select {
		case <-done:
			return nil

		case event, ok := <-w.Events:
			if !ok {
				return nil
			}
			path := event.Name

			// Solo procesar archivos de código
			if !parser.IsCodeFile(path) {
				continue
			}

			// Debounce: esperar 500ms antes de procesar
			if t, exists := debounce[path]; exists {
				t.Stop()
			}

			op := event.Op
			debounce[path] = time.AfterFunc(500*time.Millisecond, func() {
				delete(debounce, path)
				if op&fsnotify.Remove != 0 || op&fsnotify.Rename != 0 {
					if err := removeFn(path); err != nil {
						log.Printf("⚠️  Failed to remove nodes for %s: %v", path, err)
					} else {
						log.Printf("🗑️  Removed nodes for %s", path)
					}
					return
				}
				if op&(fsnotify.Create|fsnotify.Write) != 0 {
					if err := indexFn(path); err != nil {
						log.Printf("⚠️  Failed to reindex %s: %v", path, err)
					} else {
						log.Printf("🔄 Reindexed %s", path)
					}
				}
			})

		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			log.Printf("⚠️  Watcher error: %v", err)
		}
	}
}
```

- [ ] **Step 2: Integrar en main.go — añadir flag `--watch` a `prism serve`**

Buscar el case `"serve"` en `main.go`:
```bash
grep -n '"serve"' main.go
```

Añadir el flag `--watch` y arrancar el watcher si está activo. El código a añadir después de arrancar el servidor HTTP:
```go
// Añadir junto a los flags del subcomando serve:
watchFlag := serveCmd.Bool("watch", false, "Auto-reindex files on change")

// Añadir después de iniciar el servidor, antes del select/block:
if *watchFlag {
    done := make(chan struct{})
    go func() {
        indexFn := func(path string) error {
            return indexSingleFile(graph, path)
        }
        removeFn := func(path string) error {
            return graph.RemoveFileNodes(path)
        }
        if err := watcher.Watch(*repoFlag, indexFn, removeFn, done); err != nil {
            log.Printf("⚠️  File watcher failed: %v", err)
        }
    }()
}
```

Añadir el import `"github.com/ruffini/prism/watcher"` en main.go.

- [ ] **Step 3: Verificar que `indexSingleFile` y `graph.RemoveFileNodes` existen**

```bash
grep -n "indexSingleFile\|RemoveFileNodes" main.go graph/builder.go
```

Si `indexSingleFile` no existe, añadirla en `main.go`:
```go
func indexSingleFile(g *graph.CodeGraph, filePath string) error {
	p := parser.GetParser(filePath)
	if p == nil {
		return nil
	}
	parsed, err := p.ParseFile(filePath)
	if err != nil {
		return err
	}
	return g.AddParsedFile(parsed)
}
```

Si `RemoveFileNodes` no existe en `graph/builder.go`, añadirla:
```go
func (g *CodeGraph) RemoveFileNodes(filePath string) error {
	_, err := g.DB.Exec(`DELETE FROM nodes WHERE file = ?`, filePath)
	return err
}
```

- [ ] **Step 4: Compilar**

```bash
go build ./...
```

Expected: sin errores.

- [ ] **Step 5: Commit**

```bash
git add watcher/watcher.go main.go graph/builder.go
git commit -m "feat: add file watcher with 500ms debounce for auto-reindex"
```

---

### Task 8: Verificación final

- [ ] **Step 1: Ejecutar todos los tests**

```bash
go test ./... -v 2>&1 | tail -30
```

Expected: PASS en los tests de parser. Los tests de integración existentes no deben romperse.

- [ ] **Step 2: Probar indexado con repo de prueba**

```bash
go run . index -repo test/sample-repo
go run . index -repo .
```

Expected: indexa .ts y .go sin errores.

- [ ] **Step 3: Verificar `go vet`**

```bash
go vet ./...
```

Expected: sin warnings.

- [ ] **Step 4: Commit final si es necesario**

```bash
git add -A
git status
# Solo commitear si hay cambios pendientes
```
