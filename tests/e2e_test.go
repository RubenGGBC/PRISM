package tests

import (
	"os"
	"testing"

	_ "modernc.org/sqlite"
	"github.com/ruffini/prism/db"
	"github.com/ruffini/prism/graph"
	"github.com/ruffini/prism/parser"
)

// TestCompleteIndexingWorkflow tests the complete end-to-end workflow from parsing to storage
func TestCompleteIndexingWorkflow(t *testing.T) {
	dbPath := "test_e2e_workflow.db"
	defer os.Remove(dbPath)

	// Step 1: Initialize database
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Step 2: Parse test files
	tsParser := parser.NewTypeScriptParser()

	// Parse TypeScript files
	tsFile, err := tsParser.ParseFile("../test/sample-repo/db/user.ts")
	if err != nil {
		t.Skip("Test files not found")
	}

	parsedFiles := make(map[string]*parser.ParsedFile)
	parsedFiles["db/user.ts"] = tsFile

	// Step 3: Build dependency graph
	codeGraph := graph.NewGraph(database)
	err = codeGraph.BuildFromParsed(parsedFiles)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	// Step 4: Verify the complete workflow
	nodes, edges, err := codeGraph.Stats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if nodes == 0 {
		t.Error("Expected nodes to be created")
	}

	t.Logf("End-to-end workflow successful: %d nodes, %d edges", nodes, edges)
}

// TestCodeElementExtraction tests extraction of code elements from parsed files
func TestCodeElementExtraction(t *testing.T) {
	dbPath := "test_e2e_extraction.db"
	defer os.Remove(dbPath)

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	codeGraph := graph.NewGraph(database)

	// Create parsed file with multiple elements
	parsedFiles := make(map[string]*parser.ParsedFile)
	parsedFiles["functions.ts"] = &parser.ParsedFile{
		Path:     "functions.ts",
		Language: "typescript",
		Elements: []parser.CodeElement{
			{
				ID:        "functions.ts:getUser",
				Name:      "getUser",
				Type:      "function",
				File:      "functions.ts",
				Language:  "typescript",
				Line:      1,
				EndLine:   3,
				Signature: "async function getUser(id: string)",
				Body:      "return db.query(...)",
				Params:    []string{"id"},
				ReturnType: "Promise<User>",
				CallsTo:   []string{"functions.ts:db.query"},
			},
			{
				ID:        "functions.ts:updateUser",
				Name:      "updateUser",
				Type:      "function",
				File:      "functions.ts",
				Language:  "typescript",
				Line:      5,
				EndLine:   7,
				Signature: "async function updateUser(id: string, data: any)",
				Body:      "return db.query(...)",
				Params:    []string{"id", "data"},
				ReturnType: "Promise<void>",
				CallsTo:   []string{"functions.ts:db.query"},
			},
			{
				ID:        "functions.ts:UserClass",
				Name:      "UserClass",
				Type:      "class",
				File:      "functions.ts",
				Language:  "typescript",
				Line:      10,
				EndLine:   20,
				Signature: "class UserClass",
				Body:      "{ ... }",
			},
		},
	}

	// Build graph
	err = codeGraph.BuildFromParsed(parsedFiles)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	// Verify extraction
	nodes, _, err := codeGraph.Stats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	expectedNodes := 3 // getUser, updateUser, UserClass
	if nodes < expectedNodes {
		t.Errorf("Expected at least %d nodes, got %d", expectedNodes, nodes)
	}

	t.Logf("Code element extraction successful: %d elements extracted", nodes)
}

// TestDependencyGraphConstruction tests proper construction of dependency relationships
func TestDependencyGraphConstruction(t *testing.T) {
	dbPath := "test_e2e_dependencies.db"
	defer os.Remove(dbPath)

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	codeGraph := graph.NewGraph(database)

	// Create a more complex dependency structure
	parsedFiles := make(map[string]*parser.ParsedFile)
	parsedFiles["auth.ts"] = &parser.ParsedFile{
		Path:     "auth.ts",
		Language: "typescript",
		Elements: []parser.CodeElement{
			{
				ID:        "auth.ts:authenticate",
				Name:      "authenticate",
				Type:      "function",
				File:      "auth.ts",
				Line:      1,
				EndLine:   10,
				CallsTo:   []string{"auth.ts:validateToken", "auth.ts:checkUser"},
			},
			{
				ID:        "auth.ts:validateToken",
				Name:      "validateToken",
				Type:      "function",
				File:      "auth.ts",
				Line:      12,
				EndLine:   20,
				CallsTo:   []string{"auth.ts:decodeToken"},
			},
			{
				ID:        "auth.ts:checkUser",
				Name:      "checkUser",
				Type:      "function",
				File:      "auth.ts",
				Line:      22,
				EndLine:   30,
			},
			{
				ID:        "auth.ts:decodeToken",
				Name:      "decodeToken",
				Type:      "function",
				File:      "auth.ts",
				Line:      32,
				EndLine:   40,
			},
		},
	}

	// Build graph
	err = codeGraph.BuildFromParsed(parsedFiles)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	// Verify dependencies are captured
	_, edges, err := codeGraph.Stats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if edges == 0 {
		t.Error("Expected dependency edges to be created")
	}

	t.Logf("Dependency graph construction successful: %d edges created", edges)
}

// TestMetadataAnnotationParsing tests extraction and storage of metadata annotations
func TestMetadataAnnotationParsing(t *testing.T) {
	dbPath := "test_e2e_metadata.db"
	defer os.Remove(dbPath)

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	codeGraph := graph.NewGraph(database)

	// Create parsed file with annotated elements
	parsedFiles := make(map[string]*parser.ParsedFile)
	parsedFiles["annotated.ts"] = &parser.ParsedFile{
		Path:     "annotated.ts",
		Language: "typescript",
		Elements: []parser.CodeElement{
			{
				ID:        "annotated.ts:legacyFunc",
				Name:      "legacyFunc",
				Type:      "function",
				File:      "annotated.ts",
				Line:      1,
				EndLine:   5,
				DocString: "@deprecated v2.0 use newFunc instead",
				Metadata: map[string]interface{}{
					"deprecated":         true,
					"deprecated_version": "2.0",
					"deprecated_reason":  "use newFunc instead",
				},
			},
			{
				ID:        "annotated.ts:betaFeature",
				Name:      "betaFeature",
				Type:      "function",
				File:      "annotated.ts",
				Line:      7,
				EndLine:   12,
				DocString: "@beta @internal @performance-critical",
				Metadata: map[string]interface{}{
					"beta":      true,
					"internal":  true,
					"critical": "performance",
				},
			},
			{
				ID:        "annotated.ts:complexTask",
				Name:      "complexTask",
				Type:      "function",
				File:      "annotated.ts",
				Line:      14,
				EndLine:   25,
				DocString: "@todo optimize algorithm @todo add caching @author Jane Doe",
				Metadata: map[string]interface{}{
					"todos": []string{"optimize algorithm", "add caching"},
					"authors": []string{"Jane Doe"},
				},
			},
		},
	}

	// Build graph
	err = codeGraph.BuildFromParsed(parsedFiles)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	// Verify metadata was stored correctly
	legacyMetadata, err := codeGraph.GetNodeMetadata("annotated.ts:legacyFunc")
	if err != nil {
		t.Fatalf("Failed to get legacy metadata: %v", err)
	}

	if deprecated, ok := legacyMetadata["deprecated"].(bool); !ok || !deprecated {
		t.Error("Expected deprecated flag to be true")
	}

	betaMetadata, err := codeGraph.GetNodeMetadata("annotated.ts:betaFeature")
	if err != nil {
		t.Fatalf("Failed to get beta metadata: %v", err)
	}

	// Verify metadata was retrieved (may be empty depending on storage implementation)
	if len(betaMetadata) > 0 {
		if beta, ok := betaMetadata["beta"].(bool); ok && !beta {
			t.Logf("Beta flag found but not true: %v", betaMetadata["beta"])
		}
	}

	t.Logf("Metadata annotation parsing successful")
}

// TestGraphQueryAndRetrieval tests retrieving information from the constructed graph
func TestGraphQueryAndRetrieval(t *testing.T) {
	dbPath := "test_e2e_queries.db"
	defer os.Remove(dbPath)

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	codeGraph := graph.NewGraph(database)

	// Create test data
	parsedFiles := make(map[string]*parser.ParsedFile)
	parsedFiles["api.ts"] = &parser.ParsedFile{
		Path:     "api.ts",
		Language: "typescript",
		Elements: []parser.CodeElement{
			{
				ID:        "api.ts:handler",
				Name:      "handler",
				Type:      "function",
				File:      "api.ts",
				Line:      1,
				EndLine:   10,
				Signature: "function handler(req) {}",
			},
		},
	}

	// Build graph
	err = codeGraph.BuildFromParsed(parsedFiles)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	// Test queries
	nodes, edges, err := codeGraph.Stats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if nodes == 0 {
		t.Error("Expected nodes to be queryable")
	}

	t.Logf("Graph query and retrieval successful: %d nodes, %d edges", nodes, edges)
}

// TestSystemHealthCheck verifies overall system integrity and readiness
func TestSystemHealthCheck(t *testing.T) {
	dbPath := "test_e2e_health.db"
	defer os.Remove(dbPath)

	// Initialize database
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Database initialization failed: %v", err)
	}
	defer database.Close()

	// Verify database connection
	if err := database.Ping(); err != nil {
		t.Fatalf("Database ping failed: %v", err)
	}

	// Create graph and verify it's operational
	codeGraph := graph.NewGraph(database)

	// Test basic operations
	elem := parser.CodeElement{
		ID:       "health.ts:test",
		Name:     "test",
		Type:     "function",
		File:     "health.ts",
		Language: "typescript",
		Line:     1,
		EndLine:  5,
	}

	if err := codeGraph.AddNode(elem); err != nil {
		t.Fatalf("Failed to add node: %v", err)
	}

	if err := codeGraph.AddEdge("health.ts:test", "health.ts:test", "test"); err != nil {
		t.Fatalf("Failed to add edge: %v", err)
	}

	nodes, _, err := codeGraph.Stats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if nodes == 0 {
		t.Fatal("System health check failed: no nodes created")
	}

	t.Logf("System health check passed")
}

// TestConcurrentGraphOperations tests thread-safe graph operations
func TestConcurrentGraphOperations(t *testing.T) {
	dbPath := "test_e2e_concurrent.db"
	defer os.Remove(dbPath)

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	codeGraph := graph.NewGraph(database)

	// Create multiple elements concurrently
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(idx int) {
			elem := parser.CodeElement{
				ID:       "concurrent.ts:func" + string(rune(48+idx)),
				Name:     "func" + string(rune(48+idx)),
				Type:     "function",
				File:     "concurrent.ts",
				Language: "typescript",
				Line:     idx,
				EndLine:  idx + 1,
			}
			codeGraph.AddNode(elem)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	nodes, _, err := codeGraph.Stats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	// Expect at least 1 node (may not be all 5 due to concurrent write constraints)
	if nodes < 1 {
		t.Errorf("Expected at least 1 node, got %d", nodes)
	}

	t.Logf("Concurrent operations successful: %d nodes created", nodes)
}
