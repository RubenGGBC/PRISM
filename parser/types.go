package parser

// CodeElement representa una función, clase, método o variable extraída del código
type CodeElement struct {
	ID         string                 `json:"id"`          // "file.py:function_name"
	Name       string                 `json:"name"`        // Nombre del elemento
	Type       string                 `json:"type"`        // "function", "class", "method"
	File       string                 `json:"file"`        // Ruta del archivo
	Language   string                 `json:"language"`    // "python", "typescript"
	Line       int                    `json:"line"`        // Línea de inicio
	EndLine    int                    `json:"end_line"`    // Línea final
	Signature  string                 `json:"signature"`   // Firma completa
	Body       string                 `json:"body"`        // Cuerpo del código (truncado)
	DocString  string                 `json:"docstring"`   // Docstring/comentario
	Params     []string               `json:"params"`      // Lista de parámetros
	ReturnType string                 `json:"return_type"` // Tipo de retorno
	CallsTo    []string               `json:"calls_to"`    // Funciones que llama
	Metadata   map[string]interface{} `json:"metadata"`    // Metadata extraída de anotaciones
}

// ParsedFile contiene todos los elementos extraídos de un archivo
type ParsedFile struct {
	Path      string        `json:"path"`
	Language  string        `json:"language"`
	Elements  []CodeElement `json:"elements"`
	Imports   []Import      `json:"imports"`
	ParseTime int64         `json:"parse_time_ms"`
	Error     string        `json:"error,omitempty"`
}

// Import representa un import/require
type Import struct {
	Module string   `json:"module"`
	Alias  string   `json:"alias,omitempty"`
	Items  []string `json:"items,omitempty"`
	Line   int      `json:"line"`
}
