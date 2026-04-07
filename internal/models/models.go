package models

// CodeElement representa una función, clase, método o variable extraída del código
type CodeElement struct {
	ID         string                 `json:"id"`          // "file.py:function_name" o "file.ts:ClassName.method"
	Name       string                 `json:"name"`        // Nombre del elemento
	Type       string                 `json:"type"`        // "function", "class", "method", "variable"
	File       string                 `json:"file"`        // Ruta relativa del archivo
	Language   string                 `json:"language"`    // "python", "typescript", "javascript"
	Line       int                    `json:"line"`        // Línea de inicio
	EndLine    int                    `json:"end_line"`    // Línea final
	Signature  string                 `json:"signature"`   // Firma completa (ej: "def login(user: str, password: str) -> bool")
	Body       string                 `json:"body"`        // Cuerpo del código (primeros 1000 chars)
	DocString  string                 `json:"docstring"`   // Docstring/comentario asociado
	Params     []string               `json:"params"`      // Lista de parámetros
	ReturnType string                 `json:"return_type"` // Tipo de retorno si está disponible
	CallsTo    []string               `json:"calls_to"`    // Funciones que llama
	Imports    []string               `json:"imports"`     // Imports en el archivo
	Metadata   map[string]interface{} `json:"metadata,omitempty"` // Parsed annotations (@deprecated, @hook, @todo, @author)
}

// ParsedFile contiene todos los elementos extraídos de un archivo
type ParsedFile struct {
	Path      string        `json:"path"`       // Ruta del archivo
	Language  string        `json:"language"`   // Lenguaje detectado
	Elements  []CodeElement `json:"elements"`   // Elementos extraídos
	Imports   []Import      `json:"imports"`    // Todos los imports del archivo
	ParseTime int64         `json:"parse_time"` // Tiempo de parseo en ms
	Error     string        `json:"error"`      // Error si hubo problemas
}

// Import representa un import/require
type Import struct {
	Module string   `json:"module"` // Nombre del módulo
	Alias  string   `json:"alias"`  // Alias si existe (import X as Y)
	Items  []string `json:"items"`  // Items importados (from X import a, b, c)
	Line   int      `json:"line"`   // Línea del import
}

// GraphNode representa un nodo en el grafo de dependencias
type GraphNode struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	File           string            `json:"file"`
	Line           int               `json:"line"`
	EndLine        int               `json:"end_line"`
	Signature      string            `json:"signature"`
	Body           string            `json:"body"`
	Comments       string            `json:"comments,omitempty"`      // User comments
	Tags           []string          `json:"tags,omitempty"`          // User-added tags
	CustomMetadata map[string]string `json:"custom_metadata,omitempty"` // Key-value metadata
	UpdatedAt      string            `json:"updated_at,omitempty"`   // Last annotation update
	PageRank       float64           `json:"pagerank"`
	BlastRadius    int               `json:"blast_radius"`
	// Structured annotations
	Why        string `json:"why,omitempty"`
	Status     string `json:"status,omitempty"`
	EntryPoint bool   `json:"entry_point,omitempty"`
	KnownBug   string `json:"known_bug,omitempty"`
}

// AnnotationUpdate is sent by frontend to update node annotations
type AnnotationUpdate struct {
	Comments       string            `json:"comments"`
	Tags           []string          `json:"tags"`
	CustomMetadata map[string]string `json:"custom_metadata"`
	Why            string            `json:"why"`
	Status         string            `json:"status"`
	EntryPoint     bool              `json:"entry_point"`
	KnownBug       string            `json:"known_bug"`
}

// GraphEdge representa una conexión entre nodos
type GraphEdge struct {
	Source   string `json:"source"`    // ID del nodo origen
	Target   string `json:"target"`    // ID del nodo destino
	EdgeType string `json:"edge_type"` // "calls", "imports", "extends", "implements"
}

// SearchResult representa un resultado de búsqueda
type SearchResult struct {
	Node      GraphNode `json:"node"`
	Score     float64   `json:"score"`      // Similaridad (0-1)
	MatchType string    `json:"match_type"` // "semantic", "keyword", "exact"
	Context   string    `json:"context"`    // Contexto adicional
}
