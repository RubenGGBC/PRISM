package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	sitter "github.com/tree-sitter/go-tree-sitter"
	typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// Import using replace directive via go.mod since the package structure requires it

// TypeScriptParser implements Parser for TypeScript/JavaScript files
type TypeScriptParser struct {
	tsLanguage  *sitter.Language
	tsxLanguage *sitter.Language
}

// NewTypeScriptParser creates a new TypeScript/JavaScript parser
func NewTypeScriptParser() *TypeScriptParser {
	return &TypeScriptParser{
		tsLanguage:  sitter.NewLanguage(typescript.LanguageTypescript()),
		tsxLanguage: sitter.NewLanguage(typescript.LanguageTSX()),
	}
}

// Language returns the supported language
func (p *TypeScriptParser) Language() string {
	return "typescript"
}

// Extensions returns supported file extensions
func (p *TypeScriptParser) Extensions() []string {
	return []string{".ts", ".tsx", ".js", ".jsx"}
}

// ParseFile parses a TypeScript/JavaScript file
func (p *TypeScriptParser) ParseFile(filepath string) (*ParsedFile, error) {
	source, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return p.Parse(source, filepath)
}

// Parse parses TypeScript/JavaScript code and extracts elements
func (p *TypeScriptParser) Parse(source []byte, filename string) (*ParsedFile, error) {
	start := time.Now()

	parser := sitter.NewParser()
	defer parser.Close()

	// Select language based on file extension
	lang := p.getLanguageForFile(filename)
	if err := parser.SetLanguage(lang); err != nil {
		return nil, fmt.Errorf("failed to set language: %w", err)
	}

	tree := parser.Parse(source, nil)
	defer tree.Close()

	root := tree.RootNode()

	// Determine language name for output
	langName := "typescript"
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".js" || ext == ".jsx" {
		langName = "javascript"
	}

	parsed := &ParsedFile{
		Path:     filename,
		Language: langName,
		Elements: []CodeElement{},
		Imports:  []Import{},
	}

	// Extract imports
	parsed.Imports = p.extractImports(root, source)

	// Extract functions and classes
	p.walkNode(root, source, filename, parsed, "", langName)

	parsed.ParseTime = time.Since(start).Milliseconds()
	return parsed, nil
}

// getLanguageForFile returns the appropriate tree-sitter language
func (p *TypeScriptParser) getLanguageForFile(filename string) *sitter.Language {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".tsx" || ext == ".jsx" {
		return p.tsxLanguage
	}
	return p.tsLanguage
}

// walkNode traverses the AST extracting elements
func (p *TypeScriptParser) walkNode(node *sitter.Node, source []byte, filename string, parsed *ParsedFile, parentClass string, langName string) {
	nodeType := node.Kind()

	switch nodeType {
	case "function_declaration":
		elem := p.extractFunction(node, source, filename, parentClass, langName)
		parsed.Elements = append(parsed.Elements, elem)
		return

	case "arrow_function", "function_expression", "function":
		// Only extract if it's a named variable declaration
		// These are handled when we find variable_declarator
		return

	case "lexical_declaration", "variable_declaration":
		// Handle const/let/var declarations that may contain arrow functions
		p.extractVariableDeclarations(node, source, filename, parsed, langName)
		return

	case "class_declaration":
		className := p.extractClassName(node, source)
		elem := p.extractClass(node, source, filename, langName)
		parsed.Elements = append(parsed.Elements, elem)

		// Extract methods from class body
		body := p.findChildByType(node, "class_body")
		if body != nil {
			childCount := body.ChildCount()
			for i := uint(0); i < childCount; i++ {
				child := body.Child(i)
				if child == nil {
					continue
				}
				switch child.Kind() {
				case "method_definition":
					method := p.extractMethod(child, source, filename, className, langName)
					parsed.Elements = append(parsed.Elements, method)
				case "public_field_definition", "field_definition":
					// Check if field contains arrow function
					p.extractFieldArrowFunction(child, source, filename, className, parsed, langName)
				}
			}
		}
		return

	case "export_statement":
		// Handle exported declarations
		childCount := node.ChildCount()
		for i := uint(0); i < childCount; i++ {
			child := node.Child(i)
			if child != nil {
				p.walkNode(child, source, filename, parsed, parentClass, langName)
			}
		}
		return
	}

	// Recurse into children
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if child != nil {
			p.walkNode(child, source, filename, parsed, parentClass, langName)
		}
	}
}

// extractFunction extracts function information from function_declaration
func (p *TypeScriptParser) extractFunction(node *sitter.Node, source []byte, filename, parentClass, langName string) CodeElement {
	name := p.extractFunctionName(node, source)
	params := p.extractParameters(node, source)
	returnType := p.extractReturnType(node, source)
	calls := p.extractCalls(node, source)
	signature := p.buildFunctionSignature(node, source)
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
		Language:   langName,
		Line:       int(node.StartPosition().Row) + 1,
		EndLine:    int(node.EndPosition().Row) + 1,
		Signature:  signature,
		Body:       body,
		Params:     params,
		ReturnType: returnType,
		CallsTo:    calls,
	}
}

// extractMethod extracts method information from method_definition
func (p *TypeScriptParser) extractMethod(node *sitter.Node, source []byte, filename, className, langName string) CodeElement {
	name := p.extractMethodName(node, source)
	params := p.extractParameters(node, source)
	returnType := p.extractReturnType(node, source)
	calls := p.extractCalls(node, source)
	signature := p.buildMethodSignature(node, source)
	body := p.extractBody(node, source, 1000)

	fullName := className + "." + name
	id := filename + ":" + fullName

	return CodeElement{
		ID:         id,
		Name:       fullName,
		Type:       "method",
		File:       filename,
		Language:   langName,
		Line:       int(node.StartPosition().Row) + 1,
		EndLine:    int(node.EndPosition().Row) + 1,
		Signature:  signature,
		Body:       body,
		Params:     params,
		ReturnType: returnType,
		CallsTo:    calls,
	}
}

// extractClass extracts class information
func (p *TypeScriptParser) extractClass(node *sitter.Node, source []byte, filename, langName string) CodeElement {
	name := p.extractClassName(node, source)
	signature := p.buildClassSignature(node, source)
	body := p.extractBody(node, source, 500)

	return CodeElement{
		ID:        filename + ":" + name,
		Name:      name,
		Type:      "class",
		File:      filename,
		Language:  langName,
		Line:      int(node.StartPosition().Row) + 1,
		EndLine:   int(node.EndPosition().Row) + 1,
		Signature: signature,
		Body:      body,
	}
}

// extractVariableDeclarations handles const/let/var with arrow functions
func (p *TypeScriptParser) extractVariableDeclarations(node *sitter.Node, source []byte, filename string, parsed *ParsedFile, langName string) {
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		if child.Kind() == "variable_declarator" {
			p.extractVariableDeclarator(child, source, filename, parsed, langName)
		}
	}
}

// extractVariableDeclarator extracts arrow function from variable_declarator
func (p *TypeScriptParser) extractVariableDeclarator(node *sitter.Node, source []byte, filename string, parsed *ParsedFile, langName string) {
	var name string
	var valueNode *sitter.Node

	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "identifier":
			name = string(source[child.StartByte():child.EndByte()])
		case "arrow_function", "function_expression", "function":
			valueNode = child
		}
	}

	if name != "" && valueNode != nil {
		params := p.extractArrowParams(valueNode, source)
		returnType := p.extractArrowReturnType(valueNode, source)
		calls := p.extractCalls(valueNode, source)
		signature := p.buildArrowSignature(name, valueNode, source)
		body := p.extractBody(node, source, 1000)

		elem := CodeElement{
			ID:         filename + ":" + name,
			Name:       name,
			Type:       "function",
			File:       filename,
			Language:   langName,
			Line:       int(node.StartPosition().Row) + 1,
			EndLine:    int(node.EndPosition().Row) + 1,
			Signature:  signature,
			Body:       body,
			Params:     params,
			ReturnType: returnType,
			CallsTo:    calls,
		}
		parsed.Elements = append(parsed.Elements, elem)
	}
}

// extractFieldArrowFunction extracts arrow functions from class fields
func (p *TypeScriptParser) extractFieldArrowFunction(node *sitter.Node, source []byte, filename, className string, parsed *ParsedFile, langName string) {
	var name string
	var valueNode *sitter.Node

	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "property_identifier":
			name = string(source[child.StartByte():child.EndByte()])
		case "arrow_function", "function_expression":
			valueNode = child
		}
	}

	if name != "" && valueNode != nil {
		params := p.extractArrowParams(valueNode, source)
		returnType := p.extractArrowReturnType(valueNode, source)
		calls := p.extractCalls(valueNode, source)
		signature := p.buildArrowSignature(name, valueNode, source)
		body := p.extractBody(node, source, 1000)

		fullName := className + "." + name
		elem := CodeElement{
			ID:         filename + ":" + fullName,
			Name:       fullName,
			Type:       "method",
			File:       filename,
			Language:   langName,
			Line:       int(node.StartPosition().Row) + 1,
			EndLine:    int(node.EndPosition().Row) + 1,
			Signature:  signature,
			Body:       body,
			Params:     params,
			ReturnType: returnType,
			CallsTo:    calls,
		}
		parsed.Elements = append(parsed.Elements, elem)
	}
}

// extractFunctionName gets the name of a function declaration
func (p *TypeScriptParser) extractFunctionName(node *sitter.Node, source []byte) string {
	nameNode := p.findChildByType(node, "identifier")
	if nameNode != nil {
		return string(source[nameNode.StartByte():nameNode.EndByte()])
	}
	return "anonymous"
}

// extractMethodName gets the name of a method
func (p *TypeScriptParser) extractMethodName(node *sitter.Node, source []byte) string {
	nameNode := p.findChildByType(node, "property_identifier")
	if nameNode != nil {
		return string(source[nameNode.StartByte():nameNode.EndByte()])
	}
	// Try identifier for computed properties
	nameNode = p.findChildByType(node, "identifier")
	if nameNode != nil {
		return string(source[nameNode.StartByte():nameNode.EndByte()])
	}
	return "anonymous"
}

// extractClassName gets the name of a class
func (p *TypeScriptParser) extractClassName(node *sitter.Node, source []byte) string {
	// Try type_identifier first (TypeScript)
	nameNode := p.findChildByType(node, "type_identifier")
	if nameNode != nil {
		return string(source[nameNode.StartByte():nameNode.EndByte()])
	}
	// Fall back to identifier
	nameNode = p.findChildByType(node, "identifier")
	if nameNode != nil {
		return string(source[nameNode.StartByte():nameNode.EndByte()])
	}
	return "AnonymousClass"
}

// extractParameters extracts parameters from function/method
func (p *TypeScriptParser) extractParameters(node *sitter.Node, source []byte) []string {
	params := []string{}
	paramsNode := p.findChildByType(node, "formal_parameters")
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
		case "required_parameter", "optional_parameter":
			idNode := p.findFirstIdentifier(child, source)
			if idNode != "" {
				params = append(params, idNode)
			}
		case "rest_parameter":
			idNode := p.findFirstIdentifier(child, source)
			if idNode != "" {
				params = append(params, "..."+idNode)
			}
		}
	}
	return params
}

// extractArrowParams extracts parameters from arrow function
func (p *TypeScriptParser) extractArrowParams(node *sitter.Node, source []byte) []string {
	params := []string{}

	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "identifier":
			// Single parameter without parentheses
			params = append(params, string(source[child.StartByte():child.EndByte()]))
		case "formal_parameters":
			// Multiple parameters with parentheses
			pCount := child.ChildCount()
			for j := uint(0); j < pCount; j++ {
				param := child.Child(j)
				if param == nil {
					continue
				}
				switch param.Kind() {
				case "identifier":
					params = append(params, string(source[param.StartByte():param.EndByte()]))
				case "required_parameter", "optional_parameter":
					idNode := p.findFirstIdentifier(param, source)
					if idNode != "" {
						params = append(params, idNode)
					}
				case "rest_parameter":
					idNode := p.findFirstIdentifier(param, source)
					if idNode != "" {
						params = append(params, "..."+idNode)
					}
				}
			}
		}
	}
	return params
}

// findFirstIdentifier finds the first identifier in a node
func (p *TypeScriptParser) findFirstIdentifier(node *sitter.Node, source []byte) string {
	if node.Kind() == "identifier" {
		return string(source[node.StartByte():node.EndByte()])
	}
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if child != nil && child.Kind() == "identifier" {
			return string(source[child.StartByte():child.EndByte()])
		}
	}
	return ""
}

// extractReturnType extracts return type annotation
func (p *TypeScriptParser) extractReturnType(node *sitter.Node, source []byte) string {
	// Look for type_annotation after parameters
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if child != nil && child.Kind() == "type_annotation" {
			return strings.TrimPrefix(string(source[child.StartByte():child.EndByte()]), ": ")
		}
	}
	return ""
}

// extractArrowReturnType extracts return type from arrow function
func (p *TypeScriptParser) extractArrowReturnType(node *sitter.Node, source []byte) string {
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if child != nil && child.Kind() == "type_annotation" {
			return strings.TrimPrefix(string(source[child.StartByte():child.EndByte()]), ": ")
		}
	}
	return ""
}

// extractCalls extracts function calls within the code
func (p *TypeScriptParser) extractCalls(node *sitter.Node, source []byte) []string {
	calls := []string{}
	seen := make(map[string]bool)

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}

		if n.Kind() == "call_expression" {
			funcNode := n.Child(0)
			if funcNode != nil {
				var funcName string
				switch funcNode.Kind() {
				case "identifier":
					funcName = string(source[funcNode.StartByte():funcNode.EndByte()])
				case "member_expression":
					// obj.method() -> extract method name
					propNode := p.findChildByType(funcNode, "property_identifier")
					if propNode != nil {
						funcName = string(source[propNode.StartByte():propNode.EndByte()])
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

// extractImports extracts all imports from the file
func (p *TypeScriptParser) extractImports(root *sitter.Node, source []byte) []Import {
	imports := []Import{}

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}

		switch n.Kind() {
		case "import_statement":
			imp := p.parseImportStatement(n, source)
			if imp.Module != "" {
				imports = append(imports, imp)
			}

		case "call_expression":
			// Handle require() calls
			funcNode := n.Child(0)
			if funcNode != nil && funcNode.Kind() == "identifier" {
				funcName := string(source[funcNode.StartByte():funcNode.EndByte()])
				if funcName == "require" {
					imp := p.parseRequireCall(n, source)
					if imp.Module != "" {
						imports = append(imports, imp)
					}
				}
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

// parseImportStatement parses an import statement
func (p *TypeScriptParser) parseImportStatement(node *sitter.Node, source []byte) Import {
	imp := Import{Line: int(node.StartPosition().Row) + 1}

	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Kind() {
		case "string", "string_fragment":
			// Module path
			moduleStr := string(source[child.StartByte():child.EndByte()])
			imp.Module = strings.Trim(moduleStr, "\"'`")

		case "import_clause":
			// Parse import clause for named imports, default import, etc.
			p.parseImportClause(child, source, &imp)
		}
	}

	return imp
}

// parseImportClause parses the import clause
func (p *TypeScriptParser) parseImportClause(node *sitter.Node, source []byte, imp *Import) {
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Kind() {
		case "identifier":
			// Default import
			imp.Alias = string(source[child.StartByte():child.EndByte()])

		case "named_imports":
			// { name1, name2 as alias }
			p.parseNamedImports(child, source, imp)

		case "namespace_import":
			// * as name
			idNode := p.findChildByType(child, "identifier")
			if idNode != nil {
				imp.Alias = string(source[idNode.StartByte():idNode.EndByte()])
			}
		}
	}
}

// parseNamedImports parses named imports { a, b, c }
func (p *TypeScriptParser) parseNamedImports(node *sitter.Node, source []byte, imp *Import) {
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Kind() {
		case "import_specifier":
			// Could be "name" or "name as alias"
			nameNode := child.Child(0)
			if nameNode != nil && nameNode.Kind() == "identifier" {
				imp.Items = append(imp.Items, string(source[nameNode.StartByte():nameNode.EndByte()]))
			}
		}
	}
}

// parseRequireCall parses require('module') call
func (p *TypeScriptParser) parseRequireCall(node *sitter.Node, source []byte) Import {
	imp := Import{Line: int(node.StartPosition().Row) + 1}

	argsNode := p.findChildByType(node, "arguments")
	if argsNode != nil && argsNode.ChildCount() > 0 {
		// Get first argument (the module path)
		childCount := argsNode.ChildCount()
		for i := uint(0); i < childCount; i++ {
			child := argsNode.Child(i)
			if child != nil && child.Kind() == "string" {
				moduleStr := string(source[child.StartByte():child.EndByte()])
				imp.Module = strings.Trim(moduleStr, "\"'`")
				break
			}
		}
	}

	return imp
}

// buildFunctionSignature builds the signature of a function
func (p *TypeScriptParser) buildFunctionSignature(node *sitter.Node, source []byte) string {
	start := node.StartByte()
	// Find statement_block (function body)
	body := p.findChildByType(node, "statement_block")
	if body != nil {
		end := body.StartByte()
		sig := string(source[start:end])
		sig = strings.TrimRight(sig, " \n\t{")
		return sig
	}
	end := tsMinUint(node.EndByte(), start+200)
	return string(source[start:end])
}

// buildMethodSignature builds the signature of a method
func (p *TypeScriptParser) buildMethodSignature(node *sitter.Node, source []byte) string {
	start := node.StartByte()
	body := p.findChildByType(node, "statement_block")
	if body != nil {
		end := body.StartByte()
		sig := string(source[start:end])
		sig = strings.TrimRight(sig, " \n\t{")
		return sig
	}
	end := tsMinUint(node.EndByte(), start+200)
	return string(source[start:end])
}

// buildArrowSignature builds signature for arrow function
func (p *TypeScriptParser) buildArrowSignature(name string, node *sitter.Node, source []byte) string {
	// Build: const name = (params) => or const name = (params): ReturnType =>
	params := []string{}
	var returnType string

	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "identifier":
			params = append(params, string(source[child.StartByte():child.EndByte()]))
		case "formal_parameters":
			paramStr := string(source[child.StartByte():child.EndByte()])
			return fmt.Sprintf("const %s = %s =>", name, paramStr)
		case "type_annotation":
			returnType = strings.TrimPrefix(string(source[child.StartByte():child.EndByte()]), ": ")
		}
	}

	if len(params) == 1 {
		if returnType != "" {
			return fmt.Sprintf("const %s = (%s): %s =>", name, params[0], returnType)
		}
		return fmt.Sprintf("const %s = %s =>", name, params[0])
	}
	return fmt.Sprintf("const %s = () =>", name)
}

// buildClassSignature builds the signature of a class
func (p *TypeScriptParser) buildClassSignature(node *sitter.Node, source []byte) string {
	start := node.StartByte()
	body := p.findChildByType(node, "class_body")
	if body != nil {
		end := body.StartByte()
		sig := string(source[start:end])
		sig = strings.TrimRight(sig, " \n\t{")
		return sig
	}
	end := tsMinUint(node.EndByte(), start+200)
	return string(source[start:end])
}

// extractBody extracts the body of code up to maxLen characters
func (p *TypeScriptParser) extractBody(node *sitter.Node, source []byte, maxLen int) string {
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

// findChildByType finds a child node by type
func (p *TypeScriptParser) findChildByType(node *sitter.Node, nodeType string) *sitter.Node {
	childCount := node.ChildCount()
	for i := uint(0); i < childCount; i++ {
		child := node.Child(i)
		if child != nil && child.Kind() == nodeType {
			return child
		}
	}
	return nil
}

// tsMinUint returns the minimum of two uints (prefixed to avoid collision)
func tsMinUint(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}
