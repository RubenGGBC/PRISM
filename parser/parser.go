package parser

import (
	"path/filepath"
	"strings"
)

// Parser interface para diferentes lenguajes
type Parser interface {
	Parse(source []byte, filename string) (*ParsedFile, error)
	ParseFile(filename string) (*ParsedFile, error)
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
