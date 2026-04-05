package parser

import (
	"fmt"
	"os"
	"strings"
	"time"

	sitter "github.com/tree-sitter/go-tree-sitter"
	python "github.com/tree-sitter/tree-sitter-python/bindings/go"
)

// PythonParser implementa Parser para archivos Python
type PythonParser struct {
	language *sitter.Language
}

// NewPythonParser crea un nuevo parser de Python
func NewPythonParser() *PythonParser {
	return &PythonParser{
		language: sitter.NewLanguage(python.Language()),
	}
}

// Language devuelve el lenguaje soportado
func (p *PythonParser) Language() string {
	return "python"
}

// Extensions devuelve las extensiones soportadas
func (p *PythonParser) Extensions() []string {
	return []string{".py"}
}

// ParseFile parsea un archivo Python
func (p *PythonParser) ParseFile(filepath string) (*ParsedFile, error) {
	source, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return p.Parse(source, filepath)
}

// Parse parsea código Python y extrae elementos
func (p *PythonParser) Parse(source []byte, filename string) (*ParsedFile, error) {
	start := time.Now()

	parser := sitter.NewParser()
	defer parser.Close()

	if err := parser.SetLanguage(p.language); err != nil {
		return nil, fmt.Errorf("failed to set language: %w", err)
	}

	tree := parser.Parse(source, nil)
	defer tree.Close()

	root := tree.RootNode()

	parsed := &ParsedFile{
		Path:     filename,
		Language: "python",
		Elements: []CodeElement{},
		Imports:  []Import{},
	}

	// Extraer imports
	parsed.Imports = p.extractImports(root, source)

	// Extraer funciones y clases
	p.walkNode(root, source, filename, parsed, "")

	parsed.ParseTime = time.Since(start).Milliseconds()
	return parsed, nil
}

// walkNode recorre el AST extrayendo elementos
func (p *PythonParser) walkNode(node *sitter.Node, source []byte, filename string, parsed *ParsedFile, parentClass string) {
	nodeType := node.Kind()

	switch nodeType {
	case "function_definition":
		elem := p.extractFunction(node, source, filename, parentClass)
		parsed.Elements = append(parsed.Elements, elem)

	case "class_definition":
		className := p.extractClassName(node, source)
		elem := p.extractClass(node, source, filename)
		parsed.Elements = append(parsed.Elements, elem)

		// Extraer métodos de la clase
		body := p.findChildByType(node, "block")
		if body != nil {
			childCount := body.ChildCount()
			for i := uint(0); i < childCount; i++ {
				child := body.Child(i)
				if child != nil && child.Kind() == "function_definition" {
					method := p.extractFunction(child, source, filename, className)
					method.Type = "method"
					parsed.Elements = append(parsed.Elements, method)
				}
			}
		}
		return

	case "decorated_definition":
		childCount := node.ChildCount()
		for i := uint(0); i < childCount; i++ {
			child := node.Child(i)
			if child != nil {
				childType := child.Kind()
				if childType == "function_definition" || childType == "class_definition" {
					p.walkNode(child, source, filename, parsed, parentClass)
				}
			}
		}
		return
	}

	// Recursar en hijos
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if child != nil {
			p.walkNode(child, source, filename, parsed, parentClass)
		}
	}
}

// extractFunction extrae información de una función
func (p *PythonParser) extractFunction(node *sitter.Node, source []byte, filename, parentClass string) CodeElement {
	name := p.extractFunctionName(node, source)
	params := p.extractParameters(node, source)
	returnType := p.extractReturnType(node, source)
	docstring := p.extractDocstring(node, source)
	calls := p.extractCalls(node, source)
	signature := p.buildSignature(node, source)
	body := p.extractBody(node, source, 1000)

	id := filename + ":" + name
	if parentClass != "" {
		id = filename + ":" + parentClass + "." + name
		name = parentClass + "." + name
	}

	return CodeElement{
		ID:         id,
		Name:       name,
		Type:       "function",
		File:       filename,
		Language:   "python",
		Line:       int(node.StartPosition().Row) + 1,
		EndLine:    int(node.EndPosition().Row) + 1,
		Signature:  signature,
		Body:       body,
		DocString:  docstring,
		Params:     params,
		ReturnType: returnType,
		CallsTo:    calls,
	}
}

// extractClass extrae información de una clase
func (p *PythonParser) extractClass(node *sitter.Node, source []byte, filename string) CodeElement {
	name := p.extractClassName(node, source)
	docstring := p.extractDocstring(node, source)
	signature := p.buildClassSignature(node, source)
	body := p.extractBody(node, source, 500)

	return CodeElement{
		ID:        filename + ":" + name,
		Name:      name,
		Type:      "class",
		File:      filename,
		Language:  "python",
		Line:      int(node.StartPosition().Row) + 1,
		EndLine:   int(node.EndPosition().Row) + 1,
		Signature: signature,
		Body:      body,
		DocString: docstring,
	}
}

// extractFunctionName obtiene el nombre de una función
func (p *PythonParser) extractFunctionName(node *sitter.Node, source []byte) string {
	nameNode := p.findChildByType(node, "identifier")
	if nameNode != nil {
		return string(source[nameNode.StartByte():nameNode.EndByte()])
	}
	return "anonymous"
}

// extractClassName obtiene el nombre de una clase
func (p *PythonParser) extractClassName(node *sitter.Node, source []byte) string {
	nameNode := p.findChildByType(node, "identifier")
	if nameNode != nil {
		return string(source[nameNode.StartByte():nameNode.EndByte()])
	}
	return "AnonymousClass"
}

// extractParameters extrae los parámetros de una función
func (p *PythonParser) extractParameters(node *sitter.Node, source []byte) []string {
	params := []string{}
	paramsNode := p.findChildByType(node, "parameters")
	if paramsNode == nil {
		return params
	}

	childCount := paramsNode.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := paramsNode.Child(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "identifier":
			params = append(params, string(source[child.StartByte():child.EndByte()]))
		case "typed_parameter", "default_parameter", "typed_default_parameter":
			idNode := p.findChildByType(child, "identifier")
			if idNode != nil {
				params = append(params, string(source[idNode.StartByte():idNode.EndByte()]))
			}
		}
	}
	return params
}

// extractReturnType extrae el tipo de retorno si existe
func (p *PythonParser) extractReturnType(node *sitter.Node, source []byte) string {
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if child != nil && child.Kind() == "type" {
			return string(source[child.StartByte():child.EndByte()])
		}
	}
	return ""
}

// extractDocstring extrae el docstring de una función/clase
func (p *PythonParser) extractDocstring(node *sitter.Node, source []byte) string {
	body := p.findChildByType(node, "block")
	if body == nil || body.ChildCount() == 0 {
		return ""
	}

	firstChild := body.Child(0)
	if firstChild != nil && firstChild.Kind() == "expression_statement" && firstChild.ChildCount() > 0 {
		expr := firstChild.Child(0)
		if expr != nil && expr.Kind() == "string" {
			docstring := string(source[expr.StartByte():expr.EndByte()])
			docstring = strings.Trim(docstring, "\"'")
			docstring = strings.TrimPrefix(docstring, "\"\"")
			docstring = strings.TrimSuffix(docstring, "\"\"")
			return strings.TrimSpace(docstring)
		}
	}
	return ""
}

// extractCalls extrae las llamadas a funciones dentro del código
func (p *PythonParser) extractCalls(node *sitter.Node, source []byte) []string {
	calls := []string{}
	seen := make(map[string]bool)

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}

		if n.Kind() == "call" {
			funcNode := n.Child(0)
			if funcNode != nil {
				var funcName string
				switch funcNode.Kind() {
				case "identifier":
					funcName = string(source[funcNode.StartByte():funcNode.EndByte()])
				case "attribute":
					// obj.method() -> extraer method
					childCount := funcNode.ChildCount()
					for i := uint(0); i < childCount; i++ {
						child := funcNode.Child(i)
						if child != nil && child.Kind() == "identifier" {
							funcName = string(source[child.StartByte():child.EndByte()])
						}
					}
				}
				if funcName != "" && !seen[funcName] {
					seen[funcName] = true
					calls = append(calls, funcName)
				}
			}
		}

		childCount := n.ChildCount()
		for i := uint(0); i < childCount; i++ {
			walk(n.Child(i))
		}
	}

	walk(node)
	return calls
}

// extractImports extrae todos los imports del archivo
func (p *PythonParser) extractImports(root *sitter.Node, source []byte) []Import {
	imports := []Import{}

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}

		switch n.Kind() {
		case "import_statement":
			imp := Import{Line: int(n.StartPosition().Row) + 1}
			childCount := n.ChildCount()
			for i := uint(0); i < childCount; i++ {
				child := n.Child(i)
				if child == nil {
					continue
				}
				if child.Kind() == "dotted_name" {
					imp.Module = string(source[child.StartByte():child.EndByte()])
				} else if child.Kind() == "aliased_import" {
					nameNode := p.findChildByType(child, "dotted_name")
					if nameNode != nil {
						imp.Module = string(source[nameNode.StartByte():nameNode.EndByte()])
					}
					aliasNode := p.findChildByType(child, "identifier")
					if aliasNode != nil {
						imp.Alias = string(source[aliasNode.StartByte():aliasNode.EndByte()])
					}
				}
			}
			if imp.Module != "" {
				imports = append(imports, imp)
			}

		case "import_from_statement":
			imp := Import{Line: int(n.StartPosition().Row) + 1}
			childCount := n.ChildCount()
			for i := uint(0); i < childCount; i++ {
				child := n.Child(i)
				if child == nil {
					continue
				}
				switch child.Kind() {
				case "dotted_name", "relative_import":
					imp.Module = string(source[child.StartByte():child.EndByte()])
				case "identifier":
					imp.Items = append(imp.Items, string(source[child.StartByte():child.EndByte()]))
				case "aliased_import":
					nameNode := p.findChildByType(child, "identifier")
					if nameNode != nil {
						imp.Items = append(imp.Items, string(source[nameNode.StartByte():nameNode.EndByte()]))
					}
				}
			}
			if imp.Module != "" {
				imports = append(imports, imp)
			}
		}

		childCount := n.ChildCount()
		for i := uint(0); i < childCount; i++ {
			walk(n.Child(i))
		}
	}

	walk(root)
	return imports
}

// buildSignature construye la firma de una función
func (p *PythonParser) buildSignature(node *sitter.Node, source []byte) string {
	start := node.StartByte()
	body := p.findChildByType(node, "block")
	if body != nil {
		end := body.StartByte()
		sig := string(source[start:end])
		sig = strings.TrimRight(sig, ": \n\t")
		return sig
	}
	end := minUint(node.EndByte(), start+200)
	return string(source[start:end])
}

// buildClassSignature construye la firma de una clase
func (p *PythonParser) buildClassSignature(node *sitter.Node, source []byte) string {
	start := node.StartByte()
	body := p.findChildByType(node, "block")
	if body != nil {
		end := body.StartByte()
		sig := string(source[start:end])
		sig = strings.TrimRight(sig, ": \n\t")
		return sig
	}
	end := minUint(node.EndByte(), start+200)
	return string(source[start:end])
}

// extractBody extrae el cuerpo de código hasta maxLen caracteres
func (p *PythonParser) extractBody(node *sitter.Node, source []byte, maxLen int) string {
	start := node.StartByte()
	end := node.EndByte()
	if int(end-start) > maxLen {
		end = start + uint(maxLen)
	}
	body := string(source[start:end])
	if int(node.EndByte()-node.StartByte()) > maxLen {
		body += "\n... (truncated)"
	}
	return body
}

// findChildByType busca un hijo por tipo
func (p *PythonParser) findChildByType(node *sitter.Node, nodeType string) *sitter.Node {
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if child != nil && child.Kind() == nodeType {
			return child
		}
	}
	return nil
}

func minUint(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}
