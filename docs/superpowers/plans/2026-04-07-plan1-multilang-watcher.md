# Plan 1: Multi-language Parser + File Watcher

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Añadir soporte para Go, Rust y Java al indexador, y reindexado automático cuando cambian archivos.

**Architecture:** Se añaden tres nuevos parsers siguiendo el patrón existente (tree-sitter). Se actualiza `DetectLanguage`/`IsCodeFile` en `parser/parser.go` y el `indexRepository` en `main.go`. El file watcher vive en `watcher/watcher.go` y se activa con `prism serve --watch`.

**Tech Stack:** Go, tree-sitter (go-tree-sitter), tree-sitter-go/rust/java grammars, fsnotify

---

## Archivos que se tocan

| Acción | Archivo |
|--------|---------|
| Modify | `go.mod` |
| Create | `parser/go_parser.go` |
| Create | `parser/rust_parser.go` |
| Create | `parser/java_parser.go` |
| Modify | `parser/parser.go` |
| Create | `watcher/watcher.go` |
| Modify | `graph/builder.go` |
| Modify | `main.go` |
| Create | `tests/parser_test.go` |

---

### Task 1: Añadir dependencias de lenguajes

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Añadir grammars al go.mod**

Ejecutar en el directorio raíz:
```bash
go get github.com/tree-sitter/tree-sitter-go@latest
go get github.com/tree-sitter/tree-sitter-rust@latest
go get github.com/tree-sitter/tree-sitter-java@latest
go get github.com/fsnotify/fsnotify@latest
go mod tidy
```

- [ ] **Step 2: Verificar que compila**

```bash
go build ./...
```

Expected: sin errores de compilación.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "feat: add tree-sitter grammars for Go, Rust, Java and fsnotify"
```

---

### Task 2: Parser de Go

**Files:**
- Create: `parser/go_parser.go`
- Test: `tests/parser_test.go`

- [ ] **Step 1: Escribir el test que falla**

Crear `tests/parser_test.go`:
```go
package tests

import (
	"os"
	"testing"

	"github.com/ruffini/prism/parser"
)

func TestGoParser(t *testing.T) {
	// Write a temp Go file
	src := []byte(`package main

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
`)
	f, _ := os.CreateTemp("", "test_*.go")
	f.Write(src)
	f.Close()
	defer os.Remove(f.Name())

	p := parser.NewGoParser()
	result, err := p.ParseFile(f.Name())
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if result.Language != "go" {
		t.Errorf("expected language=go, got %s", result.Language)
	}

	names := make(map[string]bool)
	for _, e := range result.Elements {
		names[e.Name] = true
	}
	if !names["NewServer"] {
		t.Error("expected to find function NewServer")
	}
	if !names["Server.Start"] {
		t.Error("expected to find method Server.Start")
	}
	if !names["Server"] {
		t.Error("expected to find struct Server")
	}
	if len(result.Imports) == 0 {
		t.Error("expected at least one import")
	}
}
```

- [ ] **Step 2: Ejecutar test para confirmar que falla**

```bash
go test ./tests/... -run TestGoParser -v
```

Expected: FAIL con "undefined: parser.NewGoParser"

- [ ] **Step 3: Implementar el parser de Go**

Crear `parser/go_parser.go`:
```go
package parser

import (
	"fmt"
	"os"
	"strings"
	"time"

	sitter "github.com/tree-sitter/go-tree-sitter"
	golang "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

type GoParser struct {
	language *sitter.Language
}

func NewGoParser() *GoParser {
	return &GoParser{
		language: sitter.NewLanguage(golang.Language()),
	}
}

func (p *GoParser) Language() string        { return "go" }
func (p *GoParser) Extensions() []string    { return []string{".go"} }

func (p *GoParser) ParseFile(fp string) (*ParsedFile, error) {
	source, err := os.ReadFile(fp)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return p.Parse(source, fp)
}

func (p *GoParser) Parse(source []byte, filename string) (*ParsedFile, error) {
	start := time.Now()

	par := sitter.NewParser()
	defer par.Close()

	if err := par.SetLanguage(p.language); err != nil {
		return nil, fmt.Errorf("failed to set language: %w", err)
	}

	tree := par.Parse(source, nil)
	defer tree.Close()

	parsed := &ParsedFile{
		Path:     filename,
		Language: "go",
		Elements: []CodeElement{},
		Imports:  []Import{},
	}

	root := tree.RootNode()
	parsed.Imports = p.extractImports(root, source)
	p.walkNode(root, source, filename, parsed)
	parsed.ParseTime = time.Since(start).Milliseconds()
	return parsed, nil
}

func (p *GoParser) walkNode(node *sitter.Node, source []byte, filename string, parsed *ParsedFile) {
	switch node.Kind() {
	case "function_declaration":
		if elem := p.extractFunction(node, source, filename); elem != nil {
			parsed.Elements = append(parsed.Elements, *elem)
		}
		return // don't recurse into function body for top-level walk
	case "method_declaration":
		if elem := p.extractMethod(node, source, filename); elem != nil {
			parsed.Elements = append(parsed.Elements, *elem)
		}
		return
	case "type_declaration":
		if elem := p.extractTypeDecl(node, source, filename); elem != nil {
			parsed.Elements = append(parsed.Elements, *elem)
		}
		return
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		if child := node.Child(uint(i)); child != nil {
			p.walkNode(child, source, filename, parsed)
		}
	}
}

func (p *GoParser) extractFunction(node *sitter.Node, source []byte, filename string) *CodeElement {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	name := string(source[nameNode.StartByte():nameNode.EndByte()])
	id := fmt.Sprintf("%s:%s", filename, name)

	sig := p.nodeSignature(node, source)
	body := p.nodeBody(node, source)

	return &CodeElement{
		ID:        id,
		Name:      name,
		Type:      "function",
		File:      filename,
		Language:  "go",
		Line:      int(node.StartPosition().Row) + 1,
		EndLine:   int(node.EndPosition().Row) + 1,
		Signature: sig,
		Body:      body,
		CallsTo:   p.extractCalls(node, source),
		DocString: p.extractDocComment(node, source),
	}
}

func (p *GoParser) extractMethod(node *sitter.Node, source []byte, filename string) *CodeElement {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	name := string(source[nameNode.StartByte():nameNode.EndByte()])

	receiverType := p.extractReceiverType(node, source)
	fullName := name
	if receiverType != "" {
		fullName = receiverType + "." + name
	}

	id := fmt.Sprintf("%s:%s", filename, fullName)
	sig := p.nodeSignature(node, source)
	body := p.nodeBody(node, source)

	return &CodeElement{
		ID:        id,
		Name:      fullName,
		Type:      "method",
		File:      filename,
		Language:  "go",
		Line:      int(node.StartPosition().Row) + 1,
		EndLine:   int(node.EndPosition().Row) + 1,
		Signature: sig,
		Body:      body,
		CallsTo:   p.extractCalls(node, source),
		DocString: p.extractDocComment(node, source),
	}
}

func (p *GoParser) extractTypeDecl(node *sitter.Node, source []byte, filename string) *CodeElement {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child == nil || child.Kind() != "type_spec" {
			continue
		}
		nameNode := child.ChildByFieldName("name")
		typeNode := child.ChildByFieldName("type")
		if nameNode == nil {
			continue
		}
		name := string(source[nameNode.StartByte():nameNode.EndByte()])
		elemType := "type"
		if typeNode != nil {
			switch typeNode.Kind() {
			case "struct_type":
				elemType = "struct"
			case "interface_type":
				elemType = "interface"
			}
		}
		sig := strings.TrimSpace(string(source[node.StartByte():node.EndByte()]))
		if len(sig) > 300 {
			sig = sig[:300]
		}
		return &CodeElement{
			ID:        fmt.Sprintf("%s:%s", filename, name),
			Name:      name,
			Type:      elemType,
			File:      filename,
			Language:  "go",
			Line:      int(node.StartPosition().Row) + 1,
			EndLine:   int(node.EndPosition().Row) + 1,
			Signature: sig,
		}
	}
	return nil
}

func (p *GoParser) extractReceiverType(node *sitter.Node, source []byte) string {
	recv := node.ChildByFieldName("receiver")
	if recv == nil {
		return ""
	}
	text := string(source[recv.StartByte():recv.EndByte()])
	// "(s *Server)" → "Server"
	text = strings.Trim(text, "()")
	parts := strings.Fields(text)
	for _, part := range parts {
		clean := strings.Trim(part, "*[]")
		if clean != "" && !strings.ContainsAny(clean, "()") {
			return clean
		}
	}
	return ""
}

func (p *GoParser) nodeSignature(node *sitter.Node, source []byte) string {
	bodyNode := node.ChildByFieldName("body")
	var end uint
	if bodyNode != nil {
		end = bodyNode.StartByte()
	} else {
		end = node.EndByte()
	}
	sig := strings.TrimSpace(string(source[node.StartByte():end]))
	if len(sig) > 300 {
		sig = sig[:300]
	}
	return sig
}

func (p *GoParser) nodeBody(node *sitter.Node, source []byte) string {
	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		return ""
	}
	body := string(source[bodyNode.StartByte():bodyNode.EndByte()])
	if len(body) > 1000 {
		body = body[:1000] + "..."
	}
	return body
}

func (p *GoParser) extractDocComment(node *sitter.Node, source []byte) string {
	// Go doc comments are the comment_group immediately before the declaration
	prev := node.PrevNamedSibling()
	if prev != nil && prev.Kind() == "comment" {
		return strings.TrimSpace(string(source[prev.StartByte():prev.EndByte()]))
	}
	return ""
}

func (p *GoParser) extractCalls(node *sitter.Node, source []byte) []string {
	seen := make(map[string]bool)
	var calls []string
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Kind() == "call_expression" {
			fn := n.ChildByFieldName("function")
			if fn != nil {
				call := strings.TrimSpace(string(source[fn.StartByte():fn.EndByte()]))
				if !seen[call] {
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

func (p *GoParser) extractImports(root *sitter.Node, source []byte) []Import {
	var imports []Import
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Kind() == "import_spec" {
			pathNode := n.ChildByFieldName("path")
			if pathNode != nil {
				path := strings.Trim(string(source[pathNode.StartByte():pathNode.EndByte()]), `"`)
				imp := Import{Module: path, Line: int(n.StartPosition().Row) + 1}
				if alias := n.ChildByFieldName("name"); alias != nil {
					imp.Alias = string(source[alias.StartByte():alias.EndByte()])
				}
				imports = append(imports, imp)
			}
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

- [ ] **Step 4: Ejecutar test**

```bash
go test ./tests/... -run TestGoParser -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add parser/go_parser.go tests/parser_test.go
git commit -m "feat: add Go language parser via tree-sitter"
```

---

### Task 3: Parser de Rust

**Files:**
- Create: `parser/rust_parser.go`
- Modify: `tests/parser_test.go`

- [ ] **Step 1: Añadir test para Rust**

Añadir al final de `tests/parser_test.go`:
```go
func TestRustParser(t *testing.T) {
	src := []byte(`use std::io;

pub struct Config {
    pub port: u16,
}

pub fn new_config(port: u16) -> Config {
    Config { port }
}

impl Config {
    pub fn start(&self) {
        println!("started on {}", self.port);
    }
}
`)
	f, _ := os.CreateTemp("", "test_*.rs")
	f.Write(src)
	f.Close()
	defer os.Remove(f.Name())

	p := parser.NewRustParser()
	result, err := p.ParseFile(f.Name())
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if result.Language != "rust" {
		t.Errorf("expected language=rust, got %s", result.Language)
	}
	names := make(map[string]bool)
	for _, e := range result.Elements {
		names[e.Name] = true
	}
	if !names["new_config"] {
		t.Error("expected to find function new_config")
	}
	if !names["Config"] {
		t.Error("expected to find struct Config")
	}
}
```

- [ ] **Step 2: Verificar que falla**

```bash
go test ./tests/... -run TestRustParser -v
```

Expected: FAIL

- [ ] **Step 3: Implementar parser de Rust**

Crear `parser/rust_parser.go`:
```go
package parser

import (
	"fmt"
	"os"
	"strings"
	"time"

	sitter "github.com/tree-sitter/go-tree-sitter"
	rust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
)

type RustParser struct {
	language *sitter.Language
}

func NewRustParser() *RustParser {
	return &RustParser{language: sitter.NewLanguage(rust.Language())}
}

func (p *RustParser) Language() string     { return "rust" }
func (p *RustParser) Extensions() []string { return []string{".rs"} }

func (p *RustParser) ParseFile(fp string) (*ParsedFile, error) {
	source, err := os.ReadFile(fp)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return p.Parse(source, fp)
}

func (p *RustParser) Parse(source []byte, filename string) (*ParsedFile, error) {
	start := time.Now()

	par := sitter.NewParser()
	defer par.Close()
	if err := par.SetLanguage(p.language); err != nil {
		return nil, fmt.Errorf("failed to set language: %w", err)
	}

	tree := par.Parse(source, nil)
	defer tree.Close()

	parsed := &ParsedFile{
		Path: filename, Language: "rust",
		Elements: []CodeElement{}, Imports: []Import{},
	}

	root := tree.RootNode()
	parsed.Imports = p.extractUses(root, source)
	p.walkNode(root, source, filename, parsed, "")
	parsed.ParseTime = time.Since(start).Milliseconds()
	return parsed, nil
}

func (p *RustParser) walkNode(node *sitter.Node, source []byte, filename string, parsed *ParsedFile, implType string) {
	switch node.Kind() {
	case "function_item":
		elemType := "function"
		name := ""
		if implType != "" {
			elemType = "method"
		}
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			name = string(source[nameNode.StartByte():nameNode.EndByte()])
		}
		if name == "" {
			break
		}
		fullName := name
		if implType != "" {
			fullName = implType + "::" + name
		}
		bodyNode := node.ChildByFieldName("body")
		var sig string
		if bodyNode != nil {
			sig = strings.TrimSpace(string(source[node.StartByte():bodyNode.StartByte()]))
		} else {
			sig = strings.TrimSpace(string(source[node.StartByte():node.EndByte()]))
		}
		if len(sig) > 300 {
			sig = sig[:300]
		}
		body := ""
		if bodyNode != nil {
			body = string(source[bodyNode.StartByte():bodyNode.EndByte()])
			if len(body) > 1000 {
				body = body[:1000] + "..."
			}
		}
		parsed.Elements = append(parsed.Elements, CodeElement{
			ID:        fmt.Sprintf("%s:%s", filename, fullName),
			Name:      fullName,
			Type:      elemType,
			File:      filename,
			Language:  "rust",
			Line:      int(node.StartPosition().Row) + 1,
			EndLine:   int(node.EndPosition().Row) + 1,
			Signature: sig,
			Body:      body,
			CallsTo:   p.extractCalls(node, source),
		})
		return
	case "struct_item":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			name := string(source[nameNode.StartByte():nameNode.EndByte()])
			sig := strings.TrimSpace(string(source[node.StartByte():node.EndByte()]))
			if len(sig) > 300 {
				sig = sig[:300]
			}
			parsed.Elements = append(parsed.Elements, CodeElement{
				ID: fmt.Sprintf("%s:%s", filename, name), Name: name,
				Type: "struct", File: filename, Language: "rust",
				Line: int(node.StartPosition().Row) + 1, EndLine: int(node.EndPosition().Row) + 1,
				Signature: sig,
			})
		}
		return
	case "impl_item":
		// Extract the type this impl is for
		typeNode := node.ChildByFieldName("type")
		implT := ""
		if typeNode != nil {
			implT = string(source[typeNode.StartByte():typeNode.EndByte()])
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			if child := node.Child(uint(i)); child != nil {
				p.walkNode(child, source, filename, parsed, implT)
			}
		}
		return
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		if child := node.Child(uint(i)); child != nil {
			p.walkNode(child, source, filename, parsed, implType)
		}
	}
}

func (p *RustParser) extractCalls(node *sitter.Node, source []byte) []string {
	seen := make(map[string]bool)
	var calls []string
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Kind() == "call_expression" {
			if fn := n.ChildByFieldName("function"); fn != nil {
				call := strings.TrimSpace(string(source[fn.StartByte():fn.EndByte()]))
				if !seen[call] {
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

func (p *RustParser) extractUses(root *sitter.Node, source []byte) []Import {
	var imports []Import
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Kind() == "use_declaration" {
			text := strings.TrimSpace(string(source[n.StartByte():n.EndByte()]))
			text = strings.TrimPrefix(text, "use ")
			text = strings.TrimSuffix(text, ";")
			imports = append(imports, Import{Module: text, Line: int(n.StartPosition().Row) + 1})
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

- [ ] **Step 4: Ejecutar test**

```bash
go test ./tests/... -run TestRustParser -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add parser/rust_parser.go tests/parser_test.go
git commit -m "feat: add Rust language parser via tree-sitter"
```

---

### Task 4: Parser de Java

**Files:**
- Create: `parser/java_parser.go`
- Modify: `tests/parser_test.go`

- [ ] **Step 1: Añadir test para Java**

Añadir al final de `tests/parser_test.go`:
```go
func TestJavaParser(t *testing.T) {
	src := []byte(`import java.util.List;

public class UserService {
    private String name;

    public UserService(String name) {
        this.name = name;
    }

    public String getName() {
        return this.name;
    }
}
`)
	f, _ := os.CreateTemp("", "test_*.java")
	f.Write(src)
	f.Close()
	defer os.Remove(f.Name())

	p := parser.NewJavaParser()
	result, err := p.ParseFile(f.Name())
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if result.Language != "java" {
		t.Errorf("expected language=java, got %s", result.Language)
	}
	names := make(map[string]bool)
	for _, e := range result.Elements {
		names[e.Name] = true
	}
	if !names["UserService"] {
		t.Error("expected to find class UserService")
	}
	if !names["UserService.getName"] {
		t.Error("expected to find method UserService.getName")
	}
}
```

- [ ] **Step 2: Verificar que falla**

```bash
go test ./tests/... -run TestJavaParser -v
```

Expected: FAIL

- [ ] **Step 3: Implementar parser de Java**

Crear `parser/java_parser.go`:
```go
package parser

import (
	"fmt"
	"os"
	"strings"
	"time"

	sitter "github.com/tree-sitter/go-tree-sitter"
	java "github.com/tree-sitter/tree-sitter-java/bindings/go"
)

type JavaParser struct {
	language *sitter.Language
}

func NewJavaParser() *JavaParser {
	return &JavaParser{language: sitter.NewLanguage(java.Language())}
}

func (p *JavaParser) Language() string     { return "java" }
func (p *JavaParser) Extensions() []string { return []string{".java"} }

func (p *JavaParser) ParseFile(fp string) (*ParsedFile, error) {
	source, err := os.ReadFile(fp)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return p.Parse(source, fp)
}

func (p *JavaParser) Parse(source []byte, filename string) (*ParsedFile, error) {
	start := time.Now()

	par := sitter.NewParser()
	defer par.Close()
	if err := par.SetLanguage(p.language); err != nil {
		return nil, fmt.Errorf("failed to set language: %w", err)
	}

	tree := par.Parse(source, nil)
	defer tree.Close()

	parsed := &ParsedFile{
		Path: filename, Language: "java",
		Elements: []CodeElement{}, Imports: []Import{},
	}

	root := tree.RootNode()
	parsed.Imports = p.extractImports(root, source)
	p.walkNode(root, source, filename, parsed, "")
	parsed.ParseTime = time.Since(start).Milliseconds()
	return parsed, nil
}

func (p *JavaParser) walkNode(node *sitter.Node, source []byte, filename string, parsed *ParsedFile, className string) {
	switch node.Kind() {
	case "class_declaration":
		nameNode := node.ChildByFieldName("name")
		if nameNode == nil {
			break
		}
		cName := string(source[nameNode.StartByte():nameNode.EndByte()])
		sig := strings.TrimSpace(string(source[node.StartByte():node.StartByte()+uint(min(300, int(node.EndByte()-node.StartByte())))]))
		parsed.Elements = append(parsed.Elements, CodeElement{
			ID: fmt.Sprintf("%s:%s", filename, cName), Name: cName,
			Type: "class", File: filename, Language: "java",
			Line: int(node.StartPosition().Row) + 1, EndLine: int(node.EndPosition().Row) + 1,
			Signature: sig,
		})
		// Walk class body with class name context
		for i := 0; i < int(node.ChildCount()); i++ {
			if child := node.Child(uint(i)); child != nil {
				p.walkNode(child, source, filename, parsed, cName)
			}
		}
		return
	case "method_declaration", "constructor_declaration":
		nameNode := node.ChildByFieldName("name")
		if nameNode == nil {
			break
		}
		name := string(source[nameNode.StartByte():nameNode.EndByte()])
		fullName := name
		if className != "" {
			fullName = className + "." + name
		}
		bodyNode := node.ChildByFieldName("body")
		var sig string
		if bodyNode != nil {
			sig = strings.TrimSpace(string(source[node.StartByte():bodyNode.StartByte()]))
		} else {
			sig = strings.TrimSpace(string(source[node.StartByte():node.EndByte()]))
		}
		if len(sig) > 300 {
			sig = sig[:300]
		}
		body := ""
		if bodyNode != nil {
			body = string(source[bodyNode.StartByte():bodyNode.EndByte()])
			if len(body) > 1000 {
				body = body[:1000] + "..."
			}
		}
		parsed.Elements = append(parsed.Elements, CodeElement{
			ID: fmt.Sprintf("%s:%s", filename, fullName), Name: fullName,
			Type: "method", File: filename, Language: "java",
			Line: int(node.StartPosition().Row) + 1, EndLine: int(node.EndPosition().Row) + 1,
			Signature: sig, Body: body,
			CallsTo: p.extractCalls(node, source),
		})
		return
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		if child := node.Child(uint(i)); child != nil {
			p.walkNode(child, source, filename, parsed, className)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (p *JavaParser) extractCalls(node *sitter.Node, source []byte) []string {
	seen := make(map[string]bool)
	var calls []string
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Kind() == "method_invocation" {
			if name := n.ChildByFieldName("name"); name != nil {
				call := string(source[name.StartByte():name.EndByte()])
				if !seen[call] {
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

func (p *JavaParser) extractImports(root *sitter.Node, source []byte) []Import {
	var imports []Import
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Kind() == "import_declaration" {
			text := strings.TrimSpace(string(source[n.StartByte():n.EndByte()]))
			text = strings.TrimPrefix(text, "import ")
			text = strings.TrimSuffix(text, ";")
			imports = append(imports, Import{Module: strings.TrimSpace(text), Line: int(n.StartPosition().Row) + 1})
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

- [ ] **Step 4: Ejecutar tests**

```bash
go test ./tests/... -v
```

Expected: todos los tests PASS

- [ ] **Step 5: Commit**

```bash
git add parser/java_parser.go tests/parser_test.go
git commit -m "feat: add Java language parser via tree-sitter"
```

---

### Task 5: Registrar los nuevos parsers en el sistema

**Files:**
- Modify: `parser/parser.go`
- Modify: `main.go`

- [ ] **Step 1: Actualizar DetectLanguage e IsCodeFile**

En `parser/parser.go`, reemplazar las funciones `DetectLanguage` e `IsCodeFile`:
```go
func DetectLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".py":
		return "python"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".go":
		return "go"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	default:
		return "unknown"
	}
}

func IsCodeFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := map[string]bool{
		".py":   true,
		".ts":   true,
		".tsx":  true,
		".js":   true,
		".jsx":  true,
		".go":   true,
		".rs":   true,
		".java": true,
	}
	return validExts[ext]
}
```

- [ ] **Step 2: Actualizar indexRepository en main.go**

En `main.go`, función `indexRepository`, reemplazar la inicialización de parsers y el switch:
```go
func indexRepository(repoPath string) {
	fmt.Printf("🔍 Indexing: %s\n", repoPath)

	dbPath := "code_graph.db"
	database, err := db.InitDB(dbPath)
	if err != nil {
		fmt.Printf("❌ Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	codeGraph := graph.NewGraph(database)

	parsers := map[string]interface {
		ParseFile(string) (*parser.ParsedFile, error)
	}{
		"python":     parser.NewPythonParser(),
		"typescript": parser.NewTypeScriptParser(),
		"javascript": parser.NewTypeScriptParser(),
		"go":         parser.NewGoParser(),
		"rust":       parser.NewRustParser(),
		"java":       parser.NewJavaParser(),
	}

	parsedFiles := make(map[string]*parser.ParsedFile)
	totalFiles := 0

	err = filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if parser.ShouldSkipPath(path) {
				return filepath.SkipDir
			}
			return nil
		}
		if !parser.IsCodeFile(path) {
			return nil
		}

		lang := parser.DetectLanguage(path)
		p, ok := parsers[lang]
		if !ok {
			return nil
		}

		relPath, _ := filepath.Rel(repoPath, path)
		parsed, err := p.ParseFile(path)
		if err != nil {
			fmt.Printf("  ⚠ Error parsing %s: %v\n", relPath, err)
			return nil
		}
		parsed.Path = relPath
		for i := range parsed.Elements {
			parsed.Elements[i].File = relPath
		}
		parsedFiles[relPath] = parsed
		totalFiles++
		fmt.Printf("  ✓ %s (%d elements)\n", relPath, len(parsed.Elements))
		return nil
	})

	if err != nil {
		fmt.Printf("❌ Walk error: %v\n", err)
		os.Exit(1)
	}

	if err := codeGraph.BuildFromParsed(parsedFiles); err != nil {
		fmt.Printf("❌ Failed to build graph: %v\n", err)
		os.Exit(1)
	}

	if err := codeGraph.ResolveCallEdges(); err != nil {
		fmt.Printf("⚠ Warning: Failed to resolve call edges: %v\n", err)
	}

	nodes, edges, err := codeGraph.Stats()
	if err != nil {
		fmt.Printf("❌ Failed to get stats: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ Indexed %d files\n", totalFiles)
	fmt.Printf("   Nodes: %d\n", nodes)
	fmt.Printf("   Edges: %d\n", edges)
	fmt.Printf("   Database: %s\n", dbPath)
}
```

- [ ] **Step 3: Compilar y verificar**

```bash
go build ./...
```

Expected: sin errores.

- [ ] **Step 4: Test de integración — indexar el propio repo**

```bash
go run . index -repo .
```

Expected: aparecen archivos `.go` en la lista con sus elementos.

- [ ] **Step 5: Commit**

```bash
git add parser/parser.go main.go
git commit -m "feat: register Go, Rust, Java parsers in indexer"
```

---

### Task 6: File Watcher

**Files:**
- Create: `watcher/watcher.go`
- Modify: `graph/builder.go`
- Modify: `main.go`

- [ ] **Step 1: Añadir RemoveFileNodes al graph**

En `graph/builder.go`, añadir al final:
```go
// RemoveFileNodes elimina todos los nodos de un archivo del grafo
func (g *CodeGraph) RemoveFileNodes(file string) error {
	_, err := g.DB.Exec(`DELETE FROM nodes WHERE file = ?`, file)
	return err
}
```

- [ ] **Step 2: Crear watcher/watcher.go**

```go
package watcher

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ruffini/prism/graph"
	"github.com/ruffini/prism/parser"
)

// Watcher observa cambios en el filesystem y reindexea archivos
type Watcher struct {
	fsw    *fsnotify.Watcher
	graph  *graph.CodeGraph
	repo   string
	logger *log.Logger
}

// New crea un nuevo Watcher
func New(g *graph.CodeGraph, repoPath string, logger *log.Logger) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{fsw: fsw, graph: g, repo: repoPath, logger: logger}, nil
}

// Start inicia el watcher en background
func (w *Watcher) Start() error {
	// Registrar todos los directorios
	err := filepath.Walk(w.repo, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return err
		}
		if parser.ShouldSkipPath(path) {
			return filepath.SkipDir
		}
		return w.fsw.Add(path)
	})
	if err != nil {
		return err
	}

	pending := make(map[string]time.Time)
	ticker := time.NewTicker(300 * time.Millisecond)

	go func() {
		for {
			select {
			case event, ok := <-w.fsw.Events:
				if !ok {
					return
				}
				if !parser.IsCodeFile(event.Name) {
					continue
				}
				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					pending[event.Name] = time.Now()
				} else if event.Op&fsnotify.Remove != 0 {
					w.handleRemove(event.Name)
				}
			case <-ticker.C:
				now := time.Now()
				for path, t := range pending {
					if now.Sub(t) >= 300*time.Millisecond {
						w.handleChange(path)
						delete(pending, path)
					}
				}
			case err, ok := <-w.fsw.Errors:
				if !ok {
					return
				}
				w.logger.Printf("Watcher error: %v", err)
			}
		}
	}()

	w.logger.Printf("👁️  Watching: %s", w.repo)
	return nil
}

// Stop detiene el watcher
func (w *Watcher) Stop() {
	w.fsw.Close()
}

func (w *Watcher) handleChange(path string) {
	lang := parser.DetectLanguage(path)
	if lang == "unknown" {
		return
	}

	relPath, _ := filepath.Rel(w.repo, path)

	var parsed *parser.ParsedFile
	var err error

	switch lang {
	case "python":
		parsed, err = parser.NewPythonParser().ParseFile(path)
	case "typescript", "javascript":
		parsed, err = parser.NewTypeScriptParser().ParseFile(path)
	case "go":
		parsed, err = parser.NewGoParser().ParseFile(path)
	case "rust":
		parsed, err = parser.NewRustParser().ParseFile(path)
	case "java":
		parsed, err = parser.NewJavaParser().ParseFile(path)
	default:
		return
	}

	if err != nil {
		w.logger.Printf("❌ Error parsing %s: %v", relPath, err)
		return
	}

	parsed.Path = relPath
	for i := range parsed.Elements {
		parsed.Elements[i].File = relPath
	}

	if err := w.graph.BuildFromParsed(map[string]*parser.ParsedFile{relPath: parsed}); err != nil {
		w.logger.Printf("❌ Error indexing %s: %v", relPath, err)
		return
	}

	w.logger.Printf("🔄 Re-indexed: %s (%d elements)", relPath, len(parsed.Elements))
}

func (w *Watcher) handleRemove(path string) {
	relPath, _ := filepath.Rel(w.repo, path)
	if err := w.graph.RemoveFileNodes(relPath); err != nil {
		w.logger.Printf("❌ Error removing nodes for %s: %v", relPath, err)
		return
	}
	w.logger.Printf("🗑️  Removed nodes for: %s", relPath)
}
```

- [ ] **Step 3: Añadir flag --watch al comando serve en main.go**

En `main.go`, case `"serve"`, añadir el flag y activar el watcher:
```go
case "serve":
    serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
    dbPath := serveCmd.String("db", "code_graph.db", "Database path")
    vectorPath := serveCmd.String("vectors", "vectors.bin", "Vector store path")
    ollamaURL := serveCmd.String("ollama", "http://localhost:11434", "Ollama API URL")
    embedModel := serveCmd.String("model", "nomic-embed-text", "Embedding model name")
    watchRepo := serveCmd.String("watch", "", "Watch directory for file changes (e.g. --watch .)")
    serveCmd.Parse(os.Args[2:])

    startMCPServer(*dbPath, *vectorPath, *ollamaURL, *embedModel, *watchRepo)
```

Actualizar la firma de `startMCPServer` y añadir el watcher:
```go
func startMCPServer(dbPath, vectorPath, ollamaURL, embedModel, watchRepo string) {
    database, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "❌ Failed to open database: %v\n", err)
        os.Exit(1)
    }
    defer database.Close()

    codeGraph := graph.NewGraph(database)
    vectorStore := vector.NewVectorStore(database)
    embedder := vector.NewEmbedderWithConfig(ollamaURL, embedModel)

    // Start file watcher if requested
    if watchRepo != "" {
        logger := log.New(os.Stdout, "[WATCH] ", log.LstdFlags)
        w, err := watcher.New(codeGraph, watchRepo, logger)
        if err != nil {
            fmt.Fprintf(os.Stderr, "⚠️  Failed to create watcher: %v\n", err)
        } else {
            if err := w.Start(); err != nil {
                fmt.Fprintf(os.Stderr, "⚠️  Failed to start watcher: %v\n", err)
            } else {
                defer w.Stop()
            }
        }
    }

    apiServer := api.NewAPIServer(codeGraph, vectorStore, database)
    mux := http.NewServeMux()
    apiServer.RegisterRoutes(mux)

    go func() {
        fmt.Printf("🌐 HTTP API server listening on http://localhost:8080\n")
        if err := http.ListenAndServe(":8080", mux); err != nil {
            fmt.Fprintf(os.Stderr, "❌ HTTP server error: %v\n", err)
        }
    }()

    mcpServer := mcp.NewMCPServer(codeGraph, vectorStore, embedder)
    go func() {
        fmt.Println("📡 MCP server ready (stdio transport)")
        if err := mcpServer.Start(); err != nil {
            fmt.Fprintf(os.Stderr, "⚠️  MCP connection closed\n")
        }
    }()

    select {}
}
```

Añadir el import de watcher en main.go:
```go
import (
    // ... imports existentes ...
    "github.com/ruffini/prism/watcher"
)
```

- [ ] **Step 4: Compilar**

```bash
go build ./...
```

Expected: sin errores.

- [ ] **Step 5: Test manual del watcher**

En una terminal:
```bash
go run . serve --watch .
```

En otra terminal, crear un archivo de prueba:
```bash
echo "def hello(): pass" > /tmp/test_watch.py
cp /tmp/test_watch.py ./test_watch.py
sleep 1
rm ./test_watch.py
```

Expected en la primera terminal: logs mostrando "Re-indexed: test_watch.py" y "Removed nodes for: test_watch.py"

- [ ] **Step 6: Actualizar help en main.go**

En `printUsage()`, añadir a la sección de opciones de `serve`:
```
Options for 'serve':
  -db <path>          Path to code_graph.db (default: code_graph.db)
  -ollama <url>       Ollama API URL (default: http://localhost:11434)
  -model <name>       Embedding model (default: nomic-embed-text)
  -watch <path>       Watch directory for live re-indexing (e.g. --watch .)
```

- [ ] **Step 7: Commit final**

```bash
git add watcher/ graph/builder.go main.go parser/parser.go
git commit -m "feat: add file watcher for live re-indexing (--watch flag)"
```

---

### Task 7: Test de integración completo

**Files:**
- Modify: `tests/parser_test.go`

- [ ] **Step 1: Test que verifica todos los lenguajes**

Añadir a `tests/parser_test.go`:
```go
func TestAllParsers(t *testing.T) {
	tests := []struct {
		name    string
		ext     string
		content string
		wantFn  string
	}{
		{"Go", ".go", "package main\nfunc Foo() {}", "Foo"},
		{"Rust", ".rs", "pub fn bar() {}", "bar"},
		{"Java", ".java", "public class A { public void baz() {} }", "A.baz"},
		{"Python", ".py", "def qux(): pass", "qux"},
		{"TypeScript", ".ts", "export function quux() {}", "quux"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _ := os.CreateTemp("", "test_*"+tt.ext)
			f.Write([]byte(tt.content))
			f.Close()
			defer os.Remove(f.Name())

			lang := parser.DetectLanguage(f.Name())
			if lang == "unknown" {
				t.Fatalf("DetectLanguage returned unknown for %s", tt.ext)
			}
			if !parser.IsCodeFile(f.Name()) {
				t.Fatalf("IsCodeFile returned false for %s", tt.ext)
			}

			var elements []parser.CodeElement
			switch lang {
			case "go":
				p := parser.NewGoParser()
				r, err := p.ParseFile(f.Name())
				if err != nil {
					t.Fatalf("parse error: %v", err)
				}
				elements = r.Elements
			case "rust":
				p := parser.NewRustParser()
				r, err := p.ParseFile(f.Name())
				if err != nil {
					t.Fatalf("parse error: %v", err)
				}
				elements = r.Elements
			case "java":
				p := parser.NewJavaParser()
				r, err := p.ParseFile(f.Name())
				if err != nil {
					t.Fatalf("parse error: %v", err)
				}
				elements = r.Elements
			case "python":
				p := parser.NewPythonParser()
				r, err := p.ParseFile(f.Name())
				if err != nil {
					t.Fatalf("parse error: %v", err)
				}
				elements = r.Elements
			case "typescript", "javascript":
				p := parser.NewTypeScriptParser()
				r, err := p.ParseFile(f.Name())
				if err != nil {
					t.Fatalf("parse error: %v", err)
				}
				elements = r.Elements
			}

			found := false
			for _, e := range elements {
				if e.Name == tt.wantFn {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected to find %q in %s elements: %v", tt.wantFn, tt.name, elements)
			}
		})
	}
}
```

- [ ] **Step 2: Ejecutar todos los tests**

```bash
go test ./tests/... -v
```

Expected: todos PASS

- [ ] **Step 3: Commit**

```bash
git add tests/parser_test.go
git commit -m "test: add integration tests for all language parsers"
```
