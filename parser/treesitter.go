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

func (p *TreeSitterParser) Language() string     { return p.config.Name }
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
		// Para funciones y métodos, no recursamos para evitar funciones anidadas duplicadas.
		// Para clases, structs e interfaces, sí recursamos para encontrar sus métodos hijos.
		if nc.ElementType == "function" || nc.ElementType == "method" {
			return
		}
		// Continuar recursando para clases/structs/interfaces
		break
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
	// La estructura AST del receiver en Go es: parameter_list → parameter_declaration → identifier + type
	// Buscamos el type_identifier (el tipo del receiver, no la variable)
	if nc.ReceiverField != "" {
		if recvNode := node.ChildByFieldName(nc.ReceiverField); recvNode != nil {
			typeName := extractTypeIdentifier(recvNode, source)
			if typeName != "" {
				implType = typeName
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

// extractTypeIdentifier recursively finds the first type_identifier in a node subtree.
// Used to extract the receiver type from Go method declarations.
func extractTypeIdentifier(node *sitter.Node, source []byte) string {
	if node.Kind() == "type_identifier" {
		return string(source[node.StartByte():node.EndByte()])
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		if child := node.Child(uint(i)); child != nil {
			if name := extractTypeIdentifier(child, source); name != "" {
				return name
			}
		}
	}
	return ""
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
