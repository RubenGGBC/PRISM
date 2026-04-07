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
