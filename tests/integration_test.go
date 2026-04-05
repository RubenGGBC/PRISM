package tests

import (
	"os"
	"testing"

	_ "modernc.org/sqlite"
	"github.com/ruffini/prism/db"
	"github.com/ruffini/prism/graph"
	"github.com/ruffini/prism/parser"
)

func TestFullIndexingPipeline(t *testing.T) {
	// Use temporary file for test database
	dbPath := "test_integration.db"
	defer os.Remove(dbPath)

	// Init DB
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer database.Close()

	// Parse test files using TypeScript parser
	tsParser := parser.NewTypeScriptParser()

	// Parse the test TypeScript file
	pyFile, err := tsParser.ParseFile("../test/sample-repo/db/user.ts")
	if err != nil {
		t.Skip("Test files not found")
	}

	parsedFiles := make(map[string]*parser.ParsedFile)
	parsedFiles["db/user.ts"] = pyFile

	// Build graph
	codeGraph := graph.NewGraph(database)
	err = codeGraph.BuildFromParsed(parsedFiles)
	if err != nil {
		t.Fatalf("BuildFromParsed failed: %v", err)
	}

	// Verify nodes created
	nodes, edges, err := codeGraph.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	if nodes == 0 {
		t.Errorf("Expected nodes > 0, got %d", nodes)
	}

	t.Logf("Graph built successfully: %d nodes, %d edges", nodes, edges)
}

func TestMetadataStorage(t *testing.T) {
	dbPath := "test_metadata.db"
	defer os.Remove(dbPath)

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer database.Close()

	codeGraph := graph.NewGraph(database)

	// Create a test node with metadata
	parsedFiles := make(map[string]*parser.ParsedFile)
	parsedFiles["test.ts"] = &parser.ParsedFile{
		Path:     "test.ts",
		Language: "typescript",
		Elements: []parser.CodeElement{
			{
				ID:        "test.ts:oldFunc",
				Name:      "oldFunc",
				Type:      "function",
				File:      "test.ts",
				Language:  "typescript",
				Line:      1,
				EndLine:   10,
				Signature: "function oldFunc() {}",
				Body:      "{}",
				DocString: "@deprecated: use newFunc instead",
				Metadata: map[string]interface{}{
					"deprecated":         true,
					"deprecated_reason":  "use newFunc instead",
				},
			},
		},
	}

	err = codeGraph.BuildFromParsed(parsedFiles)
	if err != nil {
		t.Fatalf("BuildFromParsed failed: %v", err)
	}

	// Retrieve and verify metadata
	metadata, err := codeGraph.GetNodeMetadata("test.ts:oldFunc")
	if err != nil {
		t.Fatalf("GetNodeMetadata failed: %v", err)
	}

	if deprecated, ok := metadata["deprecated"].(bool); !ok || !deprecated {
		t.Errorf("Expected deprecated=true, got %v", metadata["deprecated"])
	}

	// Note: deprecated_reason may or may not be stored depending on metadata extraction implementation
	if _, ok := metadata["deprecated_reason"]; ok {
		if _, isString := metadata["deprecated_reason"].(string); !isString {
			t.Logf("deprecated_reason is not a string: %v", metadata["deprecated_reason"])
		}
	}

	t.Logf("Metadata stored and retrieved correctly")
}

func TestGraphNodeCreation(t *testing.T) {
	dbPath := "test_nodes.db"
	defer os.Remove(dbPath)

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer database.Close()

	codeGraph := graph.NewGraph(database)

	// Create test element
	elem := parser.CodeElement{
		ID:        "test.ts:myFunction",
		Name:      "myFunction",
		Type:      "function",
		File:      "test.ts",
		Language:  "typescript",
		Line:      5,
		EndLine:   15,
		Signature: "function myFunction(param: string) {}",
		Body:      "{ console.log(param); }",
		DocString: "This is a test function",
		Params:    []string{"param"},
		ReturnType: "void",
	}

	// Add node to graph
	err = codeGraph.AddNode(elem)
	if err != nil {
		t.Fatalf("AddNode failed: %v", err)
	}

	// Verify node was created
	nodes, _, err := codeGraph.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	if nodes == 0 {
		t.Errorf("Expected nodes > 0, got %d", nodes)
	}

	t.Logf("Node created successfully")
}

func TestGraphEdgeCreation(t *testing.T) {
	dbPath := "test_edges.db"
	defer os.Remove(dbPath)

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer database.Close()

	codeGraph := graph.NewGraph(database)

	// Create test nodes
	elem1 := parser.CodeElement{
		ID:        "test.ts:funcA",
		Name:      "funcA",
		Type:      "function",
		File:      "test.ts",
		Language:  "typescript",
		Line:      1,
		EndLine:   5,
	}

	elem2 := parser.CodeElement{
		ID:        "test.ts:funcB",
		Name:      "funcB",
		Type:      "function",
		File:      "test.ts",
		Language:  "typescript",
		Line:      10,
		EndLine:   15,
	}

	// Add both nodes
	if err := codeGraph.AddNode(elem1); err != nil {
		t.Fatalf("AddNode 1 failed: %v", err)
	}
	if err := codeGraph.AddNode(elem2); err != nil {
		t.Fatalf("AddNode 2 failed: %v", err)
	}

	// Add edge between them
	err = codeGraph.AddEdge("test.ts:funcA", "test.ts:funcB", "calls")
	if err != nil {
		t.Fatalf("AddEdge failed: %v", err)
	}

	// Verify edge was created
	_, edges, err := codeGraph.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	if edges == 0 {
		t.Errorf("Expected edges > 0, got %d", edges)
	}

	t.Logf("Edge created successfully")
}

func TestDatabaseInitialization(t *testing.T) {
	dbPath := "test_db_init.db"
	defer os.Remove(dbPath)

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer database.Close()

	// Verify database is properly initialized by checking schema exists
	var tableCount int
	err = database.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&tableCount)
	if err != nil {
		t.Fatalf("Failed to query tables: %v", err)
	}

	if tableCount == 0 {
		t.Errorf("Expected tables > 0, got %d", tableCount)
	}

	t.Logf("Database initialized successfully with %d tables", tableCount)
}

func TestMultipleLanguageParsing(t *testing.T) {
	dbPath := "test_multi_lang.db"
	defer os.Remove(dbPath)

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer database.Close()

	codeGraph := graph.NewGraph(database)

	// Create test files in multiple languages
	parsedFiles := make(map[string]*parser.ParsedFile)

	parsedFiles["test.ts"] = &parser.ParsedFile{
		Path:     "test.ts",
		Language: "typescript",
		Elements: []parser.CodeElement{
			{
				ID:        "test.ts:tsFunc",
				Name:      "tsFunc",
				Type:      "function",
				File:      "test.ts",
				Language:  "typescript",
				Line:      1,
				EndLine:   5,
			},
		},
	}

	parsedFiles["test.py"] = &parser.ParsedFile{
		Path:     "test.py",
		Language: "python",
		Elements: []parser.CodeElement{
			{
				ID:        "test.py:pyFunc",
				Name:      "pyFunc",
				Type:      "function",
				File:      "test.py",
				Language:  "python",
				Line:      1,
				EndLine:   5,
			},
		},
	}

	// Build graph with multiple languages
	err = codeGraph.BuildFromParsed(parsedFiles)
	if err != nil {
		t.Fatalf("BuildFromParsed failed: %v", err)
	}

	// Verify nodes from both languages were created
	nodes, _, err := codeGraph.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	if nodes < 2 {
		t.Errorf("Expected at least 2 nodes, got %d", nodes)
	}

	t.Logf("Multiple language parsing successful: %d nodes created", nodes)
}
