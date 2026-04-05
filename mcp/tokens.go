package mcp

// countTokens returns the actual token count using tiktoken
// Falls back to character/4 estimation if tokenizer is not available
func (m *MCPServer) countTokens(text string) int {
	if m.tokenizer != nil {
		tokens := m.tokenizer.Encode(text, nil, nil)
		return len(tokens)
	}
	// Fallback estimation
	return len(text) / 4
}
