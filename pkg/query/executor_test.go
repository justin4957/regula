package query

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

func setupTestStore() *store.TripleStore {
	ts := store.NewTripleStore()

	// Add regulation-like test data
	ts.Add("GDPR:Art17", "rdf:type", "reg:Article")
	ts.Add("GDPR:Art17", "reg:title", "Right to erasure")
	ts.Add("GDPR:Art17", "reg:number", "17")
	ts.Add("GDPR:Art17", "reg:partOf", "GDPR:ChapterIII")
	ts.Add("GDPR:Art17", "reg:references", "GDPR:Art6")

	ts.Add("GDPR:Art6", "rdf:type", "reg:Article")
	ts.Add("GDPR:Art6", "reg:title", "Lawfulness of processing")
	ts.Add("GDPR:Art6", "reg:number", "6")
	ts.Add("GDPR:Art6", "reg:partOf", "GDPR:ChapterII")

	ts.Add("GDPR:Art5", "rdf:type", "reg:Article")
	ts.Add("GDPR:Art5", "reg:title", "Principles relating to processing")
	ts.Add("GDPR:Art5", "reg:number", "5")
	ts.Add("GDPR:Art5", "reg:partOf", "GDPR:ChapterII")

	ts.Add("GDPR:ChapterII", "rdf:type", "reg:Chapter")
	ts.Add("GDPR:ChapterII", "reg:title", "Principles")
	ts.Add("GDPR:ChapterII", "reg:number", "II")

	ts.Add("GDPR:ChapterIII", "rdf:type", "reg:Chapter")
	ts.Add("GDPR:ChapterIII", "reg:title", "Rights of the data subject")
	ts.Add("GDPR:ChapterIII", "reg:number", "III")

	ts.Add("GDPR:Term:personal_data", "rdf:type", "reg:DefinedTerm")
	ts.Add("GDPR:Term:personal_data", "reg:term", "personal data")
	ts.Add("GDPR:Term:personal_data", "reg:definedIn", "GDPR:Art4")

	return ts
}

func TestNewExecutor(t *testing.T) {
	ts := store.NewTripleStore()
	executor := NewExecutor(ts)

	if executor == nil {
		t.Fatal("NewExecutor() returned nil")
	}
	if executor.store != ts {
		t.Error("Executor store not set correctly")
	}
	if executor.planner == nil {
		t.Error("Executor planner not initialized")
	}
}

func TestExecutor_SimpleSelect(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `SELECT ?article WHERE { ?article rdf:type reg:Article . }`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	if result.Count != 3 {
		t.Errorf("Count = %d, want 3", result.Count)
	}

	if len(result.Variables) != 1 || result.Variables[0] != "article" {
		t.Errorf("Variables = %v, want [article]", result.Variables)
	}
}

func TestExecutor_MultiplePatterns(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		SELECT ?article ?title WHERE {
			?article rdf:type reg:Article .
			?article reg:title ?title .
		}
	`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	if result.Count != 3 {
		t.Errorf("Count = %d, want 3", result.Count)
	}

	// Check that we got article titles
	foundErasure := false
	for _, binding := range result.Bindings {
		if binding["title"] == "Right to erasure" {
			foundErasure = true
			if binding["article"] != "GDPR:Art17" {
				t.Errorf("Wrong article for 'Right to erasure': %s", binding["article"])
			}
		}
	}

	if !foundErasure {
		t.Error("Expected to find 'Right to erasure'")
	}
}

func TestExecutor_SelectAll(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `SELECT * WHERE { ?s ?p ?o . } LIMIT 5`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	if result.Count != 5 {
		t.Errorf("Count = %d, want 5", result.Count)
	}

	// Should have s, p, o variables
	if len(result.Variables) != 3 {
		t.Errorf("Variables count = %d, want 3", len(result.Variables))
	}
}

func TestExecutor_WithLimit(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `SELECT ?s WHERE { ?s rdf:type reg:Article . } LIMIT 2`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	if result.Count != 2 {
		t.Errorf("Count = %d, want 2 (LIMIT applied)", result.Count)
	}
}

func TestExecutor_WithOffset(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `SELECT ?s WHERE { ?s rdf:type reg:Article . } OFFSET 1`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	if result.Count != 2 {
		t.Errorf("Count = %d, want 2 (OFFSET 1 from 3)", result.Count)
	}
}

func TestExecutor_WithLimitAndOffset(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `SELECT ?s WHERE { ?s rdf:type reg:Article . } LIMIT 1 OFFSET 1`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	if result.Count != 1 {
		t.Errorf("Count = %d, want 1 (LIMIT 1 OFFSET 1)", result.Count)
	}
}

func TestExecutor_WithDistinct(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	// Multiple articles are partOf chapters, so without distinct we'd get duplicates
	queryStr := `SELECT DISTINCT ?chapter WHERE { ?article reg:partOf ?chapter . }`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	// Should have 2 unique chapters (ChapterII and ChapterIII)
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2 (DISTINCT chapters)", result.Count)
	}
}

func TestExecutor_WithOrderBy(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		SELECT ?article ?num WHERE {
			?article rdf:type reg:Article .
			?article reg:number ?num .
		} ORDER BY ?num
	`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	if result.Count != 3 {
		t.Fatalf("Count = %d, want 3", result.Count)
	}

	// Check ordering (5, 6, 17 alphabetically)
	expected := []string{"17", "5", "6"} // Alphabetic order, not numeric
	for i, exp := range expected {
		if result.Bindings[i]["num"] != exp {
			t.Errorf("Binding[%d].num = %s, want %s", i, result.Bindings[i]["num"], exp)
		}
	}
}

func TestExecutor_WithOrderByDesc(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		SELECT ?article ?num WHERE {
			?article rdf:type reg:Article .
			?article reg:number ?num .
		} ORDER BY DESC(?num)
	`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	// Check descending order
	expected := []string{"6", "5", "17"} // Alphabetic descending
	for i, exp := range expected {
		if result.Bindings[i]["num"] != exp {
			t.Errorf("Binding[%d].num = %s, want %s", i, result.Bindings[i]["num"], exp)
		}
	}
}

func TestExecutor_WithFilterCONTAINS(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		SELECT ?article ?title WHERE {
			?article rdf:type reg:Article .
			?article reg:title ?title .
			FILTER(CONTAINS(?title, "erasure"))
		}
	`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	if result.Count != 1 {
		t.Errorf("Count = %d, want 1 (filtered by CONTAINS)", result.Count)
	}

	if result.Bindings[0]["title"] != "Right to erasure" {
		t.Errorf("title = %v, want 'Right to erasure'", result.Bindings[0]["title"])
	}
}

func TestExecutor_WithFilterREGEX(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		SELECT ?article ?title WHERE {
			?article rdf:type reg:Article .
			?article reg:title ?title .
			FILTER(REGEX(?title, "^Right"))
		}
	`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	if result.Count != 1 {
		t.Errorf("Count = %d, want 1 (filtered by REGEX)", result.Count)
	}
}

func TestExecutor_SpecificSubject(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `SELECT ?title WHERE { GDPR:Art17 reg:title ?title . }`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	if result.Count != 1 {
		t.Errorf("Count = %d, want 1", result.Count)
	}

	if result.Count > 0 && result.Bindings[0]["title"] != "Right to erasure" {
		t.Errorf("title = %v, want 'Right to erasure'", result.Bindings[0]["title"])
	}
}

func TestExecutor_NoResults(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `SELECT ?article WHERE { ?article reg:nonexistent ?o . }`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	if result.Count != 0 {
		t.Errorf("Count = %d, want 0", result.Count)
	}
}

func TestExecutor_ParseError(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	_, err := executor.ExecuteString("INVALID QUERY")
	if err == nil {
		t.Error("ExecuteString() should return error for invalid query")
	}
	if !strings.Contains(err.Error(), "parse error") {
		t.Errorf("Error should mention 'parse error': %v", err)
	}
}

func TestExecutor_WithTimeout(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts, WithTimeout(1*time.Millisecond))

	// This query should complete fast, but we're testing the timeout mechanism
	queryStr := `SELECT ?s WHERE { ?s ?p ?o . }`

	_, err := executor.ExecuteString(queryStr)
	// May or may not timeout depending on speed, just ensure no panic
	if err != nil && !strings.Contains(err.Error(), "context") {
		t.Logf("Query completed or timed out: %v", err)
	}
}

func TestExecutor_WithContext(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	query, _ := ParseQuery(`SELECT ?s WHERE { ?s ?p ?o . }`)
	_, err := executor.ExecuteWithContext(ctx, query)

	if err == nil {
		t.Error("Expected context cancellation error")
	}
}

func TestExecutor_Metrics(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `SELECT ?article WHERE { ?article rdf:type reg:Article . }`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	if result.Metrics.TotalTime == 0 {
		t.Error("TotalTime should be > 0")
	}
	if result.Metrics.ExecuteTime == 0 {
		t.Error("ExecuteTime should be > 0")
	}
	if result.Metrics.PatternsCount != 1 {
		t.Errorf("PatternsCount = %d, want 1", result.Metrics.PatternsCount)
	}
	if result.Metrics.ResultCount != 3 {
		t.Errorf("ResultCount = %d, want 3", result.Metrics.ResultCount)
	}
}

func TestExecutor_WithOptional(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		SELECT ?article ?refs WHERE {
			?article rdf:type reg:Article .
			OPTIONAL { ?article reg:references ?refs . }
		}
	`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	// Should have 3 articles, one with references
	if result.Count != 3 {
		t.Errorf("Count = %d, want 3", result.Count)
	}

	// Find the one with references
	foundRef := false
	for _, binding := range result.Bindings {
		if binding["refs"] == "GDPR:Art6" {
			foundRef = true
			if binding["article"] != "GDPR:Art17" {
				t.Errorf("Wrong article has ref: %s", binding["article"])
			}
		}
	}

	if !foundRef {
		t.Error("Expected to find binding with references")
	}
}

func TestQueryResult_FormatTable(t *testing.T) {
	result := &QueryResult{
		Variables: []string{"article", "title"},
		Bindings: []map[string]string{
			{"article": "Art17", "title": "Right to erasure"},
			{"article": "Art6", "title": "Lawfulness"},
		},
		Count: 2,
	}

	table := result.FormatTable()

	if !strings.Contains(table, "article") {
		t.Error("Table should contain 'article' header")
	}
	if !strings.Contains(table, "Right to erasure") {
		t.Error("Table should contain 'Right to erasure' value")
	}
	if !strings.Contains(table, "2 rows") {
		t.Error("Table should show row count")
	}
}

func TestQueryResult_FormatJSON(t *testing.T) {
	result := &QueryResult{
		Variables: []string{"article"},
		Bindings: []map[string]string{
			{"article": "Art17"},
		},
		Count: 1,
	}

	json, err := result.FormatJSON()
	if err != nil {
		t.Fatalf("FormatJSON() error = %v", err)
	}

	if !strings.Contains(json, `"variables"`) {
		t.Error("JSON should contain 'variables'")
	}
	if !strings.Contains(json, `"Art17"`) {
		t.Error("JSON should contain 'Art17'")
	}
}

func TestQueryResult_FormatCSV(t *testing.T) {
	result := &QueryResult{
		Variables: []string{"article", "title"},
		Bindings: []map[string]string{
			{"article": "Art17", "title": "Right to erasure"},
		},
		Count: 1,
	}

	csvOut, err := result.FormatCSV()
	if err != nil {
		t.Fatalf("FormatCSV() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(csvOut), "\n")
	if len(lines) != 2 {
		t.Errorf("CSV should have 2 lines (header + 1 row), got %d", len(lines))
	}
	if !strings.Contains(lines[0], "article") {
		t.Error("CSV header should contain 'article'")
	}
}

func TestQueryPlanner_OptimizeQuery(t *testing.T) {
	// Create stats that make certain patterns more selective
	stats := store.IndexStats{
		TotalTriples: 1000,
		PredicateCounts: map[string]int{
			"rdf:type":   500,
			"reg:number": 100,
		},
		SubjectCounts: map[string]int{},
		ObjectCounts:  map[string]int{},
	}

	planner := NewQueryPlanner(stats)

	query := &SelectQuery{
		Variables: []string{"?article"},
		Where: []TriplePattern{
			{Subject: "?article", Predicate: "rdf:type", Object: "reg:Article"},
			{Subject: "?article", Predicate: "reg:number", Object: "17"},
		},
	}

	optimized := planner.OptimizeQuery(query)

	// The pattern with reg:number (100 count) should come before rdf:type (500 count)
	if optimized.Where[0].Predicate != "reg:number" {
		t.Errorf("Expected reg:number first (more selective), got %s", optimized.Where[0].Predicate)
	}
}

// Benchmark tests
func BenchmarkExecutor_SimpleQuery(b *testing.B) {
	ts := setupTestStore()
	executor := NewExecutor(ts)
	queryStr := `SELECT ?article WHERE { ?article rdf:type reg:Article . }`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.ExecuteString(queryStr)
	}
}

func BenchmarkExecutor_TwoPatternQuery(b *testing.B) {
	ts := setupTestStore()
	executor := NewExecutor(ts)
	queryStr := `
		SELECT ?article ?title WHERE {
			?article rdf:type reg:Article .
			?article reg:title ?title .
		}
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.ExecuteString(queryStr)
	}
}

func BenchmarkExecutor_ThreePatternQuery(b *testing.B) {
	ts := setupTestStore()
	executor := NewExecutor(ts)
	queryStr := `
		SELECT ?article ?title ?chapter WHERE {
			?article rdf:type reg:Article .
			?article reg:title ?title .
			?article reg:partOf ?chapter .
		}
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.ExecuteString(queryStr)
	}
}

func BenchmarkExecutor_WithFilter(b *testing.B) {
	ts := setupTestStore()
	executor := NewExecutor(ts)
	queryStr := `
		SELECT ?article ?title WHERE {
			?article rdf:type reg:Article .
			?article reg:title ?title .
			FILTER(CONTAINS(?title, "erasure"))
		}
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.ExecuteString(queryStr)
	}
}

// CONSTRUCT query execution tests

func TestExecutor_SimpleConstruct(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		CONSTRUCT {
			?article <http://example.org/hasTitle> ?title .
		}
		WHERE {
			?article rdf:type reg:Article .
			?article reg:title ?title .
		}
	`

	result, err := executor.ExecuteConstructString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteConstructString() error = %v", err)
	}

	// Should have 3 triples (one for each article)
	if result.Count != 3 {
		t.Errorf("Count = %d, want 3", result.Count)
	}

	// Check that triples have the correct predicate
	for _, triple := range result.Triples {
		if triple.Predicate != "http://example.org/hasTitle" {
			t.Errorf("Predicate = %s, want http://example.org/hasTitle", triple.Predicate)
		}
	}
}

func TestExecutor_ConstructWithMultipleTemplates(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		CONSTRUCT {
			?article <http://example.org/id> ?article .
			?article <http://example.org/name> ?title .
		}
		WHERE {
			?article rdf:type reg:Article .
			?article reg:title ?title .
		}
	`

	result, err := executor.ExecuteConstructString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteConstructString() error = %v", err)
	}

	// Should have 6 triples (2 patterns * 3 articles)
	if result.Count != 6 {
		t.Errorf("Count = %d, want 6", result.Count)
	}
}

func TestExecutor_ConstructWithFilter(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		CONSTRUCT {
			?article <http://example.org/erasure> ?title .
		}
		WHERE {
			?article rdf:type reg:Article .
			?article reg:title ?title .
			FILTER(CONTAINS(?title, "erasure"))
		}
	`

	result, err := executor.ExecuteConstructString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteConstructString() error = %v", err)
	}

	// Should have 1 triple (only Art17 has "erasure" in title)
	if result.Count != 1 {
		t.Errorf("Count = %d, want 1", result.Count)
	}

	if result.Triples[0].Subject != "GDPR:Art17" {
		t.Errorf("Subject = %s, want GDPR:Art17", result.Triples[0].Subject)
	}
}

func TestExecutor_ConstructWithOptional(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		CONSTRUCT {
			?article <http://example.org/references> ?ref .
		}
		WHERE {
			?article rdf:type reg:Article .
			OPTIONAL { ?article reg:references ?ref . }
		}
	`

	result, err := executor.ExecuteConstructString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteConstructString() error = %v", err)
	}

	// Only Art17 has references, so only 1 triple should be constructed
	// (triples with unbound variables are skipped)
	if result.Count != 1 {
		t.Errorf("Count = %d, want 1", result.Count)
	}
}

func TestExecutor_ConstructDeduplication(t *testing.T) {
	ts := store.NewTripleStore()
	// Add duplicate data that would produce duplicate triples
	ts.Add("A", "p1", "X")
	ts.Add("A", "p2", "Y")
	ts.Add("A", "p3", "Z")

	executor := NewExecutor(ts)

	queryStr := `
		CONSTRUCT {
			<http://example.org/A> <http://example.org/type> <http://example.org/Entity> .
		}
		WHERE {
			?s ?p ?o .
		}
	`

	result, err := executor.ExecuteConstructString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteConstructString() error = %v", err)
	}

	// Should be deduplicated to 1 triple
	if result.Count != 1 {
		t.Errorf("Count = %d, want 1 (deduplicated)", result.Count)
	}
}

func TestExecutor_ConstructNoResults(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		CONSTRUCT {
			?article <http://example.org/nonexistent> ?value .
		}
		WHERE {
			?article rdf:type reg:NonExistentType .
		}
	`

	result, err := executor.ExecuteConstructString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteConstructString() error = %v", err)
	}

	if result.Count != 0 {
		t.Errorf("Count = %d, want 0", result.Count)
	}
}

func TestConstructResult_FormatTurtle(t *testing.T) {
	result := &ConstructResult{
		Triples: []ConstructedTriple{
			{Subject: "http://example.org/Art17", Predicate: "http://example.org/title", Object: "Right to erasure"},
			{Subject: "http://example.org/Art17", Predicate: "http://example.org/number", Object: "17"},
			{Subject: "http://example.org/Art6", Predicate: "http://example.org/title", Object: "Lawfulness"},
		},
		Count: 3,
	}

	turtle := result.FormatTurtle()

	if !strings.Contains(turtle, "# CONSTRUCT query result") {
		t.Error("Turtle should contain header comment")
	}
	if !strings.Contains(turtle, "3 triple(s)") {
		t.Error("Turtle should show triple count")
	}
	if !strings.Contains(turtle, "<http://example.org/Art17>") {
		t.Error("Turtle should contain Art17 subject")
	}
}

func TestConstructResult_FormatNTriples(t *testing.T) {
	result := &ConstructResult{
		Triples: []ConstructedTriple{
			{Subject: "http://example.org/Art17", Predicate: "http://example.org/title", Object: "Right to erasure"},
		},
		Count: 1,
	}

	ntriples := result.FormatNTriples()

	if !strings.Contains(ntriples, "<http://example.org/Art17>") {
		t.Error("N-Triples should contain Art17 subject as URI")
	}
	if !strings.Contains(ntriples, "<http://example.org/title>") {
		t.Error("N-Triples should contain predicate as URI")
	}
	if !strings.Contains(ntriples, `"Right to erasure"`) {
		t.Error("N-Triples should contain literal in quotes")
	}
	if !strings.Contains(ntriples, " .") {
		t.Error("N-Triples should end with period")
	}
}

func TestConstructResult_FormatJSON(t *testing.T) {
	result := &ConstructResult{
		Triples: []ConstructedTriple{
			{Subject: "Art17", Predicate: "title", Object: "Right to erasure"},
		},
		Count: 1,
	}

	jsonOut, err := result.FormatJSON()
	if err != nil {
		t.Fatalf("FormatJSON() error = %v", err)
	}

	if !strings.Contains(jsonOut, `"triples"`) {
		t.Error("JSON should contain 'triples'")
	}
	if !strings.Contains(jsonOut, `"subject"`) {
		t.Error("JSON should contain 'subject'")
	}
	if !strings.Contains(jsonOut, `"Art17"`) {
		t.Error("JSON should contain 'Art17'")
	}
	if !strings.Contains(jsonOut, `"count": 1`) {
		t.Error("JSON should contain count")
	}
}

func TestConstructResult_FormatEmpty(t *testing.T) {
	result := &ConstructResult{
		Triples: []ConstructedTriple{},
		Count:   0,
	}

	turtle := result.FormatTurtle()
	if !strings.Contains(turtle, "No triples constructed") {
		t.Error("Empty result should indicate no triples")
	}

	ntriples := result.FormatNTriples()
	if ntriples != "" {
		t.Error("Empty N-Triples should be empty string")
	}
}

func TestExecutor_ConstructWithContext(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		CONSTRUCT { ?s <http://example.org/p> ?o . }
		WHERE { ?s ?p ?o . }
	`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	result, err := executor.ExecuteConstructWithContext(ctx, query)
	if err != nil {
		t.Fatalf("ExecuteConstructWithContext() error = %v", err)
	}

	if result.Count == 0 {
		t.Error("Expected some results")
	}
}

func TestExecutor_ConstructMetrics(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		CONSTRUCT { ?article <http://example.org/type> reg:Article . }
		WHERE { ?article rdf:type reg:Article . }
	`

	result, err := executor.ExecuteConstructString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteConstructString() error = %v", err)
	}

	if result.Metrics.ExecuteTime == 0 {
		t.Error("ExecuteTime should be > 0")
	}
	if result.Metrics.PatternsCount != 1 {
		t.Errorf("PatternsCount = %d, want 1", result.Metrics.PatternsCount)
	}
	if result.Metrics.ResultCount != 3 {
		t.Errorf("ResultCount = %d, want 3", result.Metrics.ResultCount)
	}
}

func BenchmarkExecutor_SimpleConstruct(b *testing.B) {
	ts := setupTestStore()
	executor := NewExecutor(ts)
	queryStr := `
		CONSTRUCT { ?article <http://example.org/type> reg:Article . }
		WHERE { ?article rdf:type reg:Article . }
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.ExecuteConstructString(queryStr)
	}
}

// DESCRIBE query execution tests

func TestExecutor_DescribeDirectURI(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `DESCRIBE GDPR:Art17`

	result, err := executor.ExecuteDescribeString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteDescribeString() error = %v", err)
	}

	// Art17 has: rdf:type, reg:title, reg:number, reg:partOf, reg:references as subject
	if result.Count < 5 {
		t.Errorf("Count = %d, want >= 5 (Art17 has type, title, number, partOf, references)", result.Count)
	}

	// Verify expected triples where Art17 is subject
	foundType := false
	foundTitle := false
	foundNumber := false
	foundPartOf := false
	foundReferences := false

	for _, triple := range result.Triples {
		if triple.Subject == "GDPR:Art17" {
			switch triple.Predicate {
			case "rdf:type":
				foundType = true
				if triple.Object != "reg:Article" {
					t.Errorf("type object = %s, want reg:Article", triple.Object)
				}
			case "reg:title":
				foundTitle = true
				if triple.Object != "Right to erasure" {
					t.Errorf("title = %s, want Right to erasure", triple.Object)
				}
			case "reg:number":
				foundNumber = true
				if triple.Object != "17" {
					t.Errorf("number = %s, want 17", triple.Object)
				}
			case "reg:partOf":
				foundPartOf = true
			case "reg:references":
				foundReferences = true
				if triple.Object != "GDPR:Art6" {
					t.Errorf("references = %s, want GDPR:Art6", triple.Object)
				}
			}
		}
	}

	if !foundType {
		t.Error("Missing rdf:type triple for Art17")
	}
	if !foundTitle {
		t.Error("Missing reg:title triple for Art17")
	}
	if !foundNumber {
		t.Error("Missing reg:number triple for Art17")
	}
	if !foundPartOf {
		t.Error("Missing reg:partOf triple for Art17")
	}
	if !foundReferences {
		t.Error("Missing reg:references triple for Art17")
	}
}

func TestExecutor_DescribeBidirectional(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	// Art6 is referenced by Art17 (Art17 reg:references Art6)
	queryStr := `DESCRIBE GDPR:Art6`

	result, err := executor.ExecuteDescribeString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteDescribeString() error = %v", err)
	}

	// Art6 should have its own triples as subject AND the incoming reference as object
	foundAsSubject := false
	foundAsObject := false

	for _, triple := range result.Triples {
		if triple.Subject == "GDPR:Art6" && triple.Predicate == "rdf:type" {
			foundAsSubject = true
		}
		if triple.Object == "GDPR:Art6" && triple.Predicate == "reg:references" {
			foundAsObject = true
			if triple.Subject != "GDPR:Art17" {
				t.Errorf("Incoming reference from %s, want GDPR:Art17", triple.Subject)
			}
		}
	}

	if !foundAsSubject {
		t.Error("DESCRIBE should include triples where resource is subject")
	}
	if !foundAsObject {
		t.Error("DESCRIBE should include triples where resource is object (bidirectional)")
	}
}

func TestExecutor_DescribeVariable(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `DESCRIBE ?article WHERE { ?article reg:title "Right to erasure" . }`

	result, err := executor.ExecuteDescribeString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteDescribeString() error = %v", err)
	}

	// Should describe Art17 (the one with title "Right to erasure")
	if result.Count < 5 {
		t.Errorf("Count = %d, want >= 5", result.Count)
	}

	// Verify we got Art17's triples
	foundArt17Title := false
	for _, triple := range result.Triples {
		if triple.Subject == "GDPR:Art17" && triple.Predicate == "reg:title" && triple.Object == "Right to erasure" {
			foundArt17Title = true
		}
	}
	if !foundArt17Title {
		t.Error("Expected Art17 title triple from variable DESCRIBE")
	}
}

func TestExecutor_DescribeMultipleResources(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `DESCRIBE GDPR:Art17 GDPR:Art5`

	result, err := executor.ExecuteDescribeString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteDescribeString() error = %v", err)
	}

	// Should have triples from both articles
	foundArt17 := false
	foundArt5 := false

	for _, triple := range result.Triples {
		if triple.Subject == "GDPR:Art17" {
			foundArt17 = true
		}
		if triple.Subject == "GDPR:Art5" {
			foundArt5 = true
		}
	}

	if !foundArt17 {
		t.Error("Expected triples for GDPR:Art17")
	}
	if !foundArt5 {
		t.Error("Expected triples for GDPR:Art5")
	}
}

func TestExecutor_DescribeURIWithBrackets(t *testing.T) {
	ts := setupTestStore()
	// Add a triple with a full URI
	ts.Add("urn:regula:gdpr:article:17", "rdf:type", "reg:Article")
	executor := NewExecutor(ts)

	queryStr := `DESCRIBE <urn:regula:gdpr:article:17>`

	result, err := executor.ExecuteDescribeString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteDescribeString() error = %v", err)
	}

	if result.Count < 1 {
		t.Errorf("Count = %d, want >= 1", result.Count)
	}

	foundType := false
	for _, triple := range result.Triples {
		if triple.Subject == "urn:regula:gdpr:article:17" && triple.Predicate == "rdf:type" {
			foundType = true
		}
	}
	if !foundType {
		t.Error("Expected rdf:type triple for URI with brackets")
	}
}

func TestExecutor_DescribeNoResults(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `DESCRIBE GDPR:NonExistent`

	result, err := executor.ExecuteDescribeString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteDescribeString() error = %v", err)
	}

	if result.Count != 0 {
		t.Errorf("Count = %d, want 0 for non-existent resource", result.Count)
	}
}

func TestExecutor_DescribeDeduplication(t *testing.T) {
	ts := store.NewTripleStore()
	// Create a cycle: A references B, B references A
	ts.Add("A", "reg:references", "B")
	ts.Add("B", "reg:references", "A")

	executor := NewExecutor(ts)

	// Describe both A and B - triples should be deduplicated
	queryStr := `DESCRIBE A B`

	result, err := executor.ExecuteDescribeString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteDescribeString() error = %v", err)
	}

	// Should have exactly 2 triples (not 4 from double counting)
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2 (deduplicated)", result.Count)
	}
}

func TestExecutor_DescribeWithContext(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `DESCRIBE GDPR:Art17`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	result, err := executor.ExecuteDescribeWithContext(ctx, query)
	if err != nil {
		t.Fatalf("ExecuteDescribeWithContext() error = %v", err)
	}

	if result.Count == 0 {
		t.Error("Expected some results")
	}
}

func TestExecutor_DescribeContextCancelled(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `DESCRIBE ?article WHERE { ?article ?p ?o . }`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = executor.ExecuteDescribeWithContext(ctx, query)
	if err == nil {
		t.Error("Expected context cancellation error")
	}
}

func TestExecutor_DescribeMetrics(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	queryStr := `DESCRIBE GDPR:Art17`

	result, err := executor.ExecuteDescribeString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteDescribeString() error = %v", err)
	}

	if result.Metrics.TotalTime == 0 {
		t.Error("TotalTime should be > 0")
	}
	if result.Metrics.ExecuteTime == 0 {
		t.Error("ExecuteTime should be > 0")
	}
	if result.Metrics.ResultCount < 5 {
		t.Errorf("ResultCount = %d, want >= 5", result.Metrics.ResultCount)
	}
}

func TestExecutor_DescribeFormatTurtle(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	result, err := executor.ExecuteDescribeString(`DESCRIBE GDPR:Art17`)
	if err != nil {
		t.Fatalf("ExecuteDescribeString() error = %v", err)
	}

	turtle := result.FormatTurtle()

	if !strings.Contains(turtle, "GDPR:Art17") {
		t.Error("Turtle output should contain GDPR:Art17")
	}
	if !strings.Contains(turtle, "Right to erasure") {
		t.Error("Turtle output should contain title")
	}
}

func TestExecutor_DescribeFormatJSON(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	result, err := executor.ExecuteDescribeString(`DESCRIBE GDPR:Art17`)
	if err != nil {
		t.Fatalf("ExecuteDescribeString() error = %v", err)
	}

	jsonOut, err := result.FormatJSON()
	if err != nil {
		t.Fatalf("FormatJSON() error = %v", err)
	}

	if !strings.Contains(jsonOut, `"triples"`) {
		t.Error("JSON should contain 'triples'")
	}
	if !strings.Contains(jsonOut, `"GDPR:Art17"`) {
		t.Error("JSON should contain GDPR:Art17")
	}
}

func TestExecutor_DescribeWrongQueryType(t *testing.T) {
	ts := setupTestStore()
	executor := NewExecutor(ts)

	// Try to execute a SELECT query as DESCRIBE
	query := &Query{
		Type:   SelectQueryType,
		Select: &SelectQuery{Variables: []string{"?s"}, Where: []TriplePattern{{Subject: "?s", Predicate: "?p", Object: "?o"}}},
	}

	_, err := executor.ExecuteDescribe(query)
	if err == nil {
		t.Error("Expected error for wrong query type")
	}
	if !strings.Contains(err.Error(), "expected DESCRIBE") {
		t.Errorf("Error should mention DESCRIBE, got: %v", err)
	}
}

func BenchmarkExecutor_DescribeQuery(b *testing.B) {
	ts := setupTestStore()
	executor := NewExecutor(ts)
	queryStr := `DESCRIBE GDPR:Art17`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.ExecuteDescribeString(queryStr)
	}
}

func BenchmarkExecutor_DescribeVariableQuery(b *testing.B) {
	ts := setupTestStore()
	executor := NewExecutor(ts)
	queryStr := `DESCRIBE ?article WHERE { ?article reg:title "Right to erasure" . }`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.ExecuteDescribeString(queryStr)
	}
}
