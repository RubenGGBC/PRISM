package mdchunker

import (
	"fmt"
	"strings"
)

const chunkSize = 200
const overlapSize = 30

// DocChunk represents a chunk of a markdown document
type DocChunk struct {
	ID         string
	File       string
	ChunkIndex int
	LineStart  int
	Content    string
}

// ChunkText splits content into overlapping word-based chunks.
// Each chunk is chunkSize words with overlapSize words from the previous chunk prepended.
func ChunkText(content, file string) []DocChunk {
	words := strings.Fields(content)
	if len(words) == 0 {
		return nil
	}

	wordLines := buildWordLineMap(content)

	var chunks []DocChunk
	chunkIndex := 0

	for start := 0; start < len(words); {
		end := start + chunkSize
		if end > len(words) {
			end = len(words)
		}

		lineStart := 1
		if start < len(wordLines) {
			lineStart = wordLines[start]
		}

		chunks = append(chunks, DocChunk{
			ID:         fmt.Sprintf("%s#chunk_%d", file, chunkIndex),
			File:       file,
			ChunkIndex: chunkIndex,
			LineStart:  lineStart,
			Content:    strings.Join(words[start:end], " "),
		})
		chunkIndex++

		if end == len(words) {
			break
		}
		start += chunkSize - overlapSize
	}

	return chunks
}

// buildWordLineMap returns a slice where index i is the line number (1-based) of the i-th word in content.
func buildWordLineMap(content string) []int {
	var wordLines []int
	lineNum := 1
	inWord := false

	for _, ch := range content {
		switch ch {
		case '\n':
			lineNum++
			inWord = false
		case ' ', '\t', '\r':
			inWord = false
		default:
			if !inWord {
				wordLines = append(wordLines, lineNum)
				inWord = true
			}
		}
	}

	return wordLines
}
