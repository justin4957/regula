package query

import (
	"strings"
	"testing"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// --- Parser tests ---

func TestParseQuery_AggregateCountGroupBy(t *testing.T) {
	queryStr := `
		SELECT ?chapter (COUNT(?article) AS ?count) WHERE {
			?article rdf:type reg:Article .
			?article reg:partOf ?chapter .
		} GROUP BY ?chapter ORDER BY DESC(?count)
	`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	if query.Type != SelectQueryType {
		t.Fatalf("Type = %v, want %v", query.Type, SelectQueryType)
	}

	sel := query.Select

	// Check plain variable
	if len(sel.Variables) != 1 || sel.Variables[0] != "?chapter" {
		t.Errorf("Variables = %v, want [?chapter]", sel.Variables)
	}

	// Check aggregate
	if len(sel.Aggregates) != 1 {
		t.Fatalf("Aggregates count = %d, want 1", len(sel.Aggregates))
	}
	agg := sel.Aggregates[0]
	if agg.Function != AggregateCOUNT {
		t.Errorf("Function = %v, want COUNT", agg.Function)
	}
	if agg.Variable != "?article" {
		t.Errorf("Variable = %v, want ?article", agg.Variable)
	}
	if agg.Alias != "?count" {
		t.Errorf("Alias = %v, want ?count", agg.Alias)
	}
	if agg.Distinct {
		t.Error("Distinct should be false")
	}

	// Check GROUP BY
	if len(sel.GroupBy) != 1 || sel.GroupBy[0] != "?chapter" {
		t.Errorf("GroupBy = %v, want [?chapter]", sel.GroupBy)
	}

	// Check ORDER BY
	if len(sel.OrderBy) != 1 {
		t.Fatalf("OrderBy count = %d, want 1", len(sel.OrderBy))
	}
	if sel.OrderBy[0].Variable != "?count" {
		t.Errorf("OrderBy variable = %v, want ?count", sel.OrderBy[0].Variable)
	}
	if !sel.OrderBy[0].Descending {
		t.Error("OrderBy should be descending")
	}
}

func TestParseQuery_MultipleAggregates(t *testing.T) {
	queryStr := `
		SELECT ?chapter (COUNT(?article) AS ?count) (SUM(?num) AS ?total) WHERE {
			?article rdf:type reg:Article .
			?article reg:partOf ?chapter .
			?article reg:number ?num .
		} GROUP BY ?chapter
	`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	sel := query.Select
	if len(sel.Aggregates) != 2 {
		t.Fatalf("Aggregates count = %d, want 2", len(sel.Aggregates))
	}

	if sel.Aggregates[0].Function != AggregateCOUNT {
		t.Errorf("Agg[0].Function = %v, want COUNT", sel.Aggregates[0].Function)
	}
	if sel.Aggregates[1].Function != AggregateSUM {
		t.Errorf("Agg[1].Function = %v, want SUM", sel.Aggregates[1].Function)
	}
}

func TestParseQuery_CountDistinct(t *testing.T) {
	queryStr := `
		SELECT (COUNT(DISTINCT ?chapter) AS ?uniqueChapters) WHERE {
			?article reg:partOf ?chapter .
		}
	`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	sel := query.Select
	if len(sel.Aggregates) != 1 {
		t.Fatalf("Aggregates count = %d, want 1", len(sel.Aggregates))
	}

	agg := sel.Aggregates[0]
	if !agg.Distinct {
		t.Error("Distinct should be true")
	}
	if agg.Variable != "?chapter" {
		t.Errorf("Variable = %v, want ?chapter", agg.Variable)
	}
	if agg.Alias != "?uniqueChapters" {
		t.Errorf("Alias = %v, want ?uniqueChapters", agg.Alias)
	}
}

func TestParseQuery_AggregateWithHaving(t *testing.T) {
	queryStr := `
		SELECT ?chapter (COUNT(?article) AS ?count) WHERE {
			?article rdf:type reg:Article .
			?article reg:partOf ?chapter .
		} GROUP BY ?chapter HAVING(COUNT(?article) > 5)
	`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	sel := query.Select
	if len(sel.Having) != 1 {
		t.Fatalf("Having count = %d, want 1", len(sel.Having))
	}
	if !strings.Contains(sel.Having[0].Expression, "COUNT") {
		t.Errorf("Having expression should contain COUNT, got %q", sel.Having[0].Expression)
	}
}

func TestParseQuery_AggregateNoGroupBy(t *testing.T) {
	queryStr := `
		SELECT (COUNT(?article) AS ?total) WHERE {
			?article rdf:type reg:Article .
		}
	`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	sel := query.Select
	if len(sel.Aggregates) != 1 {
		t.Fatalf("Aggregates count = %d, want 1", len(sel.Aggregates))
	}
	if len(sel.GroupBy) != 0 {
		t.Errorf("GroupBy should be empty, got %v", sel.GroupBy)
	}
	if len(sel.Variables) != 0 {
		t.Errorf("Variables should be empty for aggregate-only query, got %v", sel.Variables)
	}
}

func TestParseQuery_NonAggregateUnchanged(t *testing.T) {
	queryStr := `SELECT ?s ?p ?o WHERE { ?s ?p ?o . } LIMIT 10`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	sel := query.Select
	if sel.HasAggregates() {
		t.Error("HasAggregates() should be false for non-aggregate query")
	}
	if len(sel.Variables) != 3 {
		t.Errorf("Variables count = %d, want 3", len(sel.Variables))
	}
	if sel.Limit != 10 {
		t.Errorf("Limit = %d, want 10", sel.Limit)
	}
}

func TestParseQuery_AggregateString(t *testing.T) {
	queryStr := `
		SELECT ?chapter (COUNT(?article) AS ?count) WHERE {
			?article rdf:type reg:Article .
			?article reg:partOf ?chapter .
		} GROUP BY ?chapter ORDER BY DESC(?count)
	`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	str := query.String()
	if !strings.Contains(str, "COUNT(?article) AS ?count") {
		t.Errorf("String() should contain aggregate expression, got %s", str)
	}
	if !strings.Contains(str, "GROUP BY ?chapter") {
		t.Errorf("String() should contain GROUP BY, got %s", str)
	}
}

func TestParseQuery_AggregateValidation(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantErrors int
	}{
		{
			name: "valid aggregate query",
			query: `SELECT ?chapter (COUNT(?article) AS ?count) WHERE {
				?article rdf:type reg:Article .
				?article reg:partOf ?chapter .
			} GROUP BY ?chapter`,
			wantErrors: 0,
		},
		{
			name: "aggregate with unbound source variable",
			query: `SELECT ?chapter (COUNT(?missing) AS ?count) WHERE {
				?article reg:partOf ?chapter .
			} GROUP BY ?chapter`,
			wantErrors: 1,
		},
		{
			name: "plain var not in GROUP BY",
			query: `SELECT ?chapter ?title (COUNT(?article) AS ?count) WHERE {
				?article reg:partOf ?chapter .
				?article reg:title ?title .
			} GROUP BY ?chapter`,
			wantErrors: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			query, err := ParseQuery(tc.query)
			if err != nil {
				t.Fatalf("ParseQuery() error = %v", err)
			}
			validationErrors := query.Validate()
			if len(validationErrors) != tc.wantErrors {
				t.Errorf("Validate() returned %d errors, want %d: %v", len(validationErrors), tc.wantErrors, validationErrors)
			}
		})
	}
}

// --- Executor tests ---

func setupAggregateTestStore() *store.TripleStore {
	ts := store.NewTripleStore()

	// Articles in ChapterII
	ts.Add("GDPR:Art5", "rdf:type", "reg:Article")
	ts.Add("GDPR:Art5", "reg:title", "Principles relating to processing")
	ts.Add("GDPR:Art5", "reg:number", "5")
	ts.Add("GDPR:Art5", "reg:partOf", "GDPR:ChapterII")

	ts.Add("GDPR:Art6", "rdf:type", "reg:Article")
	ts.Add("GDPR:Art6", "reg:title", "Lawfulness of processing")
	ts.Add("GDPR:Art6", "reg:number", "6")
	ts.Add("GDPR:Art6", "reg:partOf", "GDPR:ChapterII")

	ts.Add("GDPR:Art7", "rdf:type", "reg:Article")
	ts.Add("GDPR:Art7", "reg:title", "Conditions for consent")
	ts.Add("GDPR:Art7", "reg:number", "7")
	ts.Add("GDPR:Art7", "reg:partOf", "GDPR:ChapterII")

	// Articles in ChapterIII
	ts.Add("GDPR:Art15", "rdf:type", "reg:Article")
	ts.Add("GDPR:Art15", "reg:title", "Right of access")
	ts.Add("GDPR:Art15", "reg:number", "15")
	ts.Add("GDPR:Art15", "reg:partOf", "GDPR:ChapterIII")

	ts.Add("GDPR:Art17", "rdf:type", "reg:Article")
	ts.Add("GDPR:Art17", "reg:title", "Right to erasure")
	ts.Add("GDPR:Art17", "reg:number", "17")
	ts.Add("GDPR:Art17", "reg:partOf", "GDPR:ChapterIII")

	// Articles in ChapterIV (single article)
	ts.Add("GDPR:Art25", "rdf:type", "reg:Article")
	ts.Add("GDPR:Art25", "reg:title", "Data protection by design")
	ts.Add("GDPR:Art25", "reg:number", "25")
	ts.Add("GDPR:Art25", "reg:partOf", "GDPR:ChapterIV")

	// Chapters
	ts.Add("GDPR:ChapterII", "rdf:type", "reg:Chapter")
	ts.Add("GDPR:ChapterII", "reg:title", "Principles")
	ts.Add("GDPR:ChapterIII", "rdf:type", "reg:Chapter")
	ts.Add("GDPR:ChapterIII", "reg:title", "Rights of the data subject")
	ts.Add("GDPR:ChapterIV", "rdf:type", "reg:Chapter")
	ts.Add("GDPR:ChapterIV", "reg:title", "Controller and processor")

	return ts
}

func TestExecutor_CountGroupBy(t *testing.T) {
	ts := setupAggregateTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		SELECT ?chapter (COUNT(?article) AS ?count) WHERE {
			?article rdf:type reg:Article .
			?article reg:partOf ?chapter .
		} GROUP BY ?chapter ORDER BY DESC(?count)
	`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	// Should have 3 groups: ChapterII (3), ChapterIII (2), ChapterIV (1)
	if result.Count != 3 {
		t.Fatalf("Count = %d, want 3", result.Count)
	}

	// Verify variables
	if len(result.Variables) != 2 {
		t.Fatalf("Variables = %v, want [chapter, count]", result.Variables)
	}

	// Check DESC ordering: ChapterII(3), ChapterIII(2), ChapterIV(1)
	if result.Bindings[0]["count"] != "3" {
		t.Errorf("First group count = %s, want 3", result.Bindings[0]["count"])
	}
	if result.Bindings[1]["count"] != "2" {
		t.Errorf("Second group count = %s, want 2", result.Bindings[1]["count"])
	}
	if result.Bindings[2]["count"] != "1" {
		t.Errorf("Third group count = %s, want 1", result.Bindings[2]["count"])
	}
}

func TestExecutor_CountNoGroupBy(t *testing.T) {
	ts := setupAggregateTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		SELECT (COUNT(?article) AS ?total) WHERE {
			?article rdf:type reg:Article .
		}
	`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	// Scalar aggregate: single result row
	if result.Count != 1 {
		t.Fatalf("Count = %d, want 1", result.Count)
	}

	if result.Bindings[0]["total"] != "6" {
		t.Errorf("total = %s, want 6", result.Bindings[0]["total"])
	}
}

func TestExecutor_AggregateOrderByDesc(t *testing.T) {
	ts := setupAggregateTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		SELECT ?chapter (COUNT(?article) AS ?articleCount) WHERE {
			?article rdf:type reg:Article .
			?article reg:partOf ?chapter .
		} GROUP BY ?chapter ORDER BY DESC(?articleCount)
	`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	// Verify DESC ordering by numeric values
	for i := 0; i < len(result.Bindings)-1; i++ {
		countI := result.Bindings[i]["articleCount"]
		countJ := result.Bindings[i+1]["articleCount"]
		if countI < countJ {
			t.Errorf("DESC order violated: %s before %s", countI, countJ)
		}
	}
}

func TestExecutor_SumAggregate(t *testing.T) {
	ts := setupAggregateTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		SELECT ?chapter (SUM(?num) AS ?totalNum) WHERE {
			?article rdf:type reg:Article .
			?article reg:partOf ?chapter .
			?article reg:number ?num .
		} GROUP BY ?chapter
	`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	if result.Count != 3 {
		t.Fatalf("Count = %d, want 3", result.Count)
	}

	// Find ChapterII sum: 5 + 6 + 7 = 18
	for _, binding := range result.Bindings {
		if binding["chapter"] == "GDPR:ChapterII" {
			if binding["totalNum"] != "18" {
				t.Errorf("ChapterII sum = %s, want 18", binding["totalNum"])
			}
		}
	}
}

func TestExecutor_AvgAggregate(t *testing.T) {
	ts := setupAggregateTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		SELECT ?chapter (AVG(?num) AS ?avgNum) WHERE {
			?article rdf:type reg:Article .
			?article reg:partOf ?chapter .
			?article reg:number ?num .
		} GROUP BY ?chapter
	`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	// Find ChapterII avg: (5 + 6 + 7) / 3 = 6
	for _, binding := range result.Bindings {
		if binding["chapter"] == "GDPR:ChapterII" {
			if binding["avgNum"] != "6" {
				t.Errorf("ChapterII avg = %s, want 6", binding["avgNum"])
			}
		}
	}
}

func TestExecutor_MinMaxAggregate(t *testing.T) {
	ts := setupAggregateTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		SELECT ?chapter (MIN(?num) AS ?minNum) (MAX(?num) AS ?maxNum) WHERE {
			?article rdf:type reg:Article .
			?article reg:partOf ?chapter .
			?article reg:number ?num .
		} GROUP BY ?chapter
	`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	for _, binding := range result.Bindings {
		if binding["chapter"] == "GDPR:ChapterII" {
			if binding["minNum"] != "5" {
				t.Errorf("ChapterII min = %s, want 5", binding["minNum"])
			}
			if binding["maxNum"] != "7" {
				t.Errorf("ChapterII max = %s, want 7", binding["maxNum"])
			}
		}
		if binding["chapter"] == "GDPR:ChapterIII" {
			if binding["minNum"] != "15" {
				t.Errorf("ChapterIII min = %s, want 15", binding["minNum"])
			}
			if binding["maxNum"] != "17" {
				t.Errorf("ChapterIII max = %s, want 17", binding["maxNum"])
			}
		}
	}
}

func TestExecutor_CountDistinct(t *testing.T) {
	ts := setupAggregateTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		SELECT (COUNT(DISTINCT ?chapter) AS ?uniqueChapters) WHERE {
			?article rdf:type reg:Article .
			?article reg:partOf ?chapter .
		}
	`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	if result.Count != 1 {
		t.Fatalf("Count = %d, want 1", result.Count)
	}

	if result.Bindings[0]["uniqueChapters"] != "3" {
		t.Errorf("uniqueChapters = %s, want 3", result.Bindings[0]["uniqueChapters"])
	}
}

func TestExecutor_Having(t *testing.T) {
	ts := setupAggregateTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		SELECT ?chapter (COUNT(?article) AS ?count) WHERE {
			?article rdf:type reg:Article .
			?article reg:partOf ?chapter .
		} GROUP BY ?chapter HAVING(COUNT(?article) > 1)
	`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	// Only ChapterII (3) and ChapterIII (2) have > 1 article
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2 (HAVING count > 1 filters out ChapterIV)", result.Count)
	}

	for _, binding := range result.Bindings {
		if binding["chapter"] == "GDPR:ChapterIV" {
			t.Error("ChapterIV should be filtered out by HAVING")
		}
	}
}

func TestExecutor_AggregateBackwardCompat(t *testing.T) {
	ts := setupAggregateTestStore()
	executor := NewExecutor(ts)

	// Non-aggregate query should still work exactly as before
	queryStr := `
		SELECT ?article ?title WHERE {
			?article rdf:type reg:Article .
			?article reg:title ?title .
		} ORDER BY ?title LIMIT 3
	`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	if result.Count != 3 {
		t.Errorf("Count = %d, want 3", result.Count)
	}

	// Verify lexicographic ordering by title
	for i := 0; i < len(result.Bindings)-1; i++ {
		if result.Bindings[i]["title"] > result.Bindings[i+1]["title"] {
			t.Errorf("Non-aggregate ORDER BY broken: %q > %q",
				result.Bindings[i]["title"], result.Bindings[i+1]["title"])
		}
	}
}

func TestExecutor_AggregateWithLimit(t *testing.T) {
	ts := setupAggregateTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		SELECT ?chapter (COUNT(?article) AS ?count) WHERE {
			?article rdf:type reg:Article .
			?article reg:partOf ?chapter .
		} GROUP BY ?chapter ORDER BY DESC(?count) LIMIT 2
	`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	if result.Count != 2 {
		t.Errorf("Count = %d, want 2 (LIMIT 2)", result.Count)
	}
}

func TestExecutor_AggregateWithOffset(t *testing.T) {
	ts := setupAggregateTestStore()
	executor := NewExecutor(ts)

	queryStr := `
		SELECT ?chapter (COUNT(?article) AS ?count) WHERE {
			?article rdf:type reg:Article .
			?article reg:partOf ?chapter .
		} GROUP BY ?chapter ORDER BY DESC(?count) OFFSET 1
	`

	result, err := executor.ExecuteString(queryStr)
	if err != nil {
		t.Fatalf("ExecuteString() error = %v", err)
	}

	// 3 groups total, OFFSET 1 → 2 results
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2 (OFFSET 1 from 3 groups)", result.Count)
	}
}

// --- Integration test with GDPR data ---

func TestGDPRQuery_ArticlesPerChapter(t *testing.T) {
	ts := loadGDPRStore(t)
	executor := NewExecutor(ts)

	start := time.Now()
	result, err := executor.ExecuteString(`
		SELECT ?chapter (COUNT(?article) AS ?articleCount) WHERE {
			?article rdf:type reg:Article .
			?article reg:partOf ?chapter .
			?chapter rdf:type reg:Chapter .
		} GROUP BY ?chapter ORDER BY DESC(?articleCount)
	`)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Query error: %v", err)
	}

	t.Logf("Articles per chapter: %d groups in %v", result.Count, elapsed)

	if result.Count < 5 {
		t.Errorf("Expected at least 5 chapter groups, got %d", result.Count)
	}

	// Verify DESC ordering
	for i := 0; i < len(result.Bindings)-1; i++ {
		countI := result.Bindings[i]["articleCount"]
		countJ := result.Bindings[i+1]["articleCount"]
		// Numeric comparison for DESC
		if compareValues(countI, countJ) < 0 {
			t.Errorf("DESC order violated at position %d: %s < %s", i, countI, countJ)
		}
	}

	// Log results for manual inspection
	for _, binding := range result.Bindings {
		t.Logf("  %s: %s articles", binding["chapter"], binding["articleCount"])
	}

	if elapsed > 200*time.Millisecond {
		t.Errorf("Aggregate query took %v, expected < 200ms", elapsed)
	}
}

func TestGDPRQuery_TotalArticleCount(t *testing.T) {
	ts := loadGDPRStore(t)
	executor := NewExecutor(ts)

	result, err := executor.ExecuteString(`
		SELECT (COUNT(?article) AS ?total) WHERE {
			?article rdf:type reg:Article .
		}
	`)
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}

	if result.Count != 1 {
		t.Fatalf("Expected 1 result row, got %d", result.Count)
	}

	total := result.Bindings[0]["total"]
	t.Logf("Total GDPR articles: %s", total)

	if total != "99" {
		t.Errorf("Expected 99 total articles, got %s", total)
	}
}

// --- Helper function tests ---

func TestCompareValues(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"3", "10", -1},  // numeric: 3 < 10
		{"10", "3", 1},   // numeric: 10 > 3
		{"5", "5", 0},    // equal
		{"abc", "def", -1}, // string fallback
		{"def", "abc", 1},
	}

	for _, tc := range tests {
		got := compareValues(tc.a, tc.b)
		if (tc.want < 0 && got >= 0) || (tc.want > 0 && got <= 0) || (tc.want == 0 && got != 0) {
			t.Errorf("compareValues(%q, %q) = %d, want sign %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestEvaluateHavingExpression(t *testing.T) {
	tests := []struct {
		expr string
		want bool
	}{
		{"3 > 1", true},
		{"1 > 3", false},
		{"5 >= 5", true},
		{"5 >= 6", false},
		{"10 < 20", true},
		{"20 < 10", false},
		{"3 = 3", true},
		{"3 != 4", true},
		{"3 != 3", false},
	}

	for _, tc := range tests {
		got := evaluateHavingExpression(tc.expr)
		if got != tc.want {
			t.Errorf("evaluateHavingExpression(%q) = %v, want %v", tc.expr, got, tc.want)
		}
	}
}

// --- Benchmark ---

func BenchmarkExecutor_AggregateCountGroupBy(b *testing.B) {
	ts := setupAggregateTestStore()
	executor := NewExecutor(ts)
	queryStr := `
		SELECT ?chapter (COUNT(?article) AS ?count) WHERE {
			?article rdf:type reg:Article .
			?article reg:partOf ?chapter .
		} GROUP BY ?chapter ORDER BY DESC(?count)
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.ExecuteString(queryStr)
	}
}

func BenchmarkExecutor_ScalarCount(b *testing.B) {
	ts := setupAggregateTestStore()
	executor := NewExecutor(ts)
	queryStr := `
		SELECT (COUNT(?article) AS ?total) WHERE {
			?article rdf:type reg:Article .
		}
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.ExecuteString(queryStr)
	}
}
