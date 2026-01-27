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
