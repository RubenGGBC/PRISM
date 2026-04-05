package parser

import (
	"regexp"
	"strings"
)

// ExtractMetadata extracts annotations from docstrings/comments
func ExtractMetadata(docstring string) map[string]interface{} {
	if docstring == "" {
		return nil
	}

	metadata := make(map[string]interface{})

	// Parse @deprecated
	if deprecatedRe := regexp.MustCompile(`@deprecated(?:\s*:\s*(.+))?`); deprecatedRe.MatchString(docstring) {
		match := deprecatedRe.FindStringSubmatch(docstring)
		metadata["deprecated"] = true
		if len(match) > 1 {
			metadata["deprecated_reason"] = strings.TrimSpace(match[1])
		}
	}

	// Parse @hook
	if hooksRe := regexp.MustCompile(`@hook[:\s]+(.+)`); hooksRe.MatchString(docstring) {
		matches := hooksRe.FindAllStringSubmatch(docstring, -1)
		var hooks []string
		for _, match := range matches {
			if len(match) > 1 {
				hooks = append(hooks, strings.TrimSpace(match[1]))
			}
		}
		if len(hooks) > 0 {
			metadata["hooks"] = hooks
		}
	}

	// Parse @todo
	if todoRe := regexp.MustCompile(`@todo[:\s]+(.+)`); todoRe.MatchString(docstring) {
		matches := todoRe.FindAllStringSubmatch(docstring, -1)
		var todos []string
		for _, match := range matches {
			if len(match) > 1 {
				todos = append(todos, strings.TrimSpace(match[1]))
			}
		}
		if len(todos) > 0 {
			metadata["todos"] = todos
		}
	}

	// Parse @author
	if authorRe := regexp.MustCompile(`@author[:\s]+(.+)`); authorRe.MatchString(docstring) {
		matches := authorRe.FindAllStringSubmatch(docstring, -1)
		var authors []string
		for _, match := range matches {
			if len(match) > 1 {
				authors = append(authors, strings.TrimSpace(match[1]))
			}
		}
		if len(authors) > 0 {
			metadata["authors"] = authors
		}
	}

	if len(metadata) == 0 {
		return nil
	}

	return metadata
}
