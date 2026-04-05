package parser

import (
	"path/filepath"
	"strings"

	"github.com/ruffini/prism/internal/models"
)

// Parser interface para diferentes lenguajes
type Parser interface {
	Parse(source []byte, filename string) (*models.ParsedFile, error)
	Language() string
	Extensions() []string
}

// DetectLanguage detecta el lenguaje basándose en la extensión del archivo
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
	default:
		return "unknown"
	}
}

// IsCodeFile verifica si el archivo es código parseable
func IsCodeFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := map[string]bool{
		".py":  true,
		".ts":  true,
		".tsx": true,
		".js":  true,
		".jsx": true,
	}
	return validExts[ext]
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
