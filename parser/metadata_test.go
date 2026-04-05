package parser

import (
	"testing"
)

func TestExtractMetadata(t *testing.T) {
	tests := []struct {
		name     string
		docstring string
		expected map[string]interface{}
	}{
		{
			name: "deprecated",
			docstring: "// @deprecated: use newFunction instead",
			expected: map[string]interface{}{
				"deprecated": true,
				"deprecated_reason": "use newFunction instead",
			},
		},
		{
			name: "hooks",
			docstring: "// @hook: beforeAuth, afterSession",
			expected: map[string]interface{}{
				"hooks": []string{"beforeAuth, afterSession"},
			},
		},
		{
			name: "todo",
			docstring: "// @todo: optimize algorithm\n// @todo: add error handling",
			expected: map[string]interface{}{
				"todos": []string{"optimize algorithm", "add error handling"},
			},
		},
		{
			name: "author",
			docstring: "// @author: John Doe",
			expected: map[string]interface{}{
				"authors": []string{"John Doe"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractMetadata(tt.docstring)
			if result == nil && tt.expected == nil {
				return
			}
			if result == nil || tt.expected == nil {
				t.Errorf("metadata = %v, want %v", result, tt.expected)
				return
			}
		})
	}
}
