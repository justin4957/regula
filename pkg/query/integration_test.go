package query

import (
	"os"
	"testing"
	"time"

	"github.com/coolbeans/regula/pkg/extract"
	"github.com/coolbeans/regula/pkg/store"
)

// loadGDPRStore builds a GDPR-sized triple store for integration testing.
func loadGDPRStore(t *testing.T) *store.TripleStore {
	t.Helper()

	// Try to load actual GDPR data
	gdprPath := "../../testdata/gdpr.txt"
	if _, err := os.Stat(gdprPath); os.IsNotExist(err) {
		gdprPath = "../extract/testdata/gdpr.txt"
	}

	file, err := os.Open(gdprPath)
	if err != nil {
		t.Skipf("GDPR test data not available: %v", err)
	}
	defer file.Close()

	parser := extract.NewParser()
	doc, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Failed to parse GDPR: %v", err)
	}

	tripleStore := store.NewTripleStore()
	builder := store.NewGraphBuilder(tripleStore, "https://regula.dev/regulations/")

	defExtractor := extract.NewDefinitionExtractor()
	refExtractor := extract.NewReferenceExtractor()

	_, err = builder.BuildWithExtractors(doc, defExtractor, refExtractor)
	if err != nil {
		t.Fatalf("Failed to build GDPR graph: %v", err)
	}

	return tripleStore
}

func TestGDPRQuery_AllArticles(t *testing.T) {
	ts := loadGDPRStore(t)
	executor := NewExecutor(ts)

	start := time.Now()
	result, err := executor.ExecuteString(`
		SELECT ?article ?title WHERE {
			?article rdf:type reg:Article .
			?article reg:title ?title .
		}
	`)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Query error: %v", err)
	}

	t.Logf("All articles query: %d results in %v", result.Count, elapsed)

	if result.Count != 99 {
		t.Errorf("Expected 99 articles, got %d", result.Count)
	}

	if elapsed > 100*time.Millisecond {
		t.Errorf("Query took %v, expected < 100ms", elapsed)
	}
}

func TestGDPRQuery_AllChapters(t *testing.T) {
	ts := loadGDPRStore(t)
	executor := NewExecutor(ts)

	start := time.Now()
	result, err := executor.ExecuteString(`
		SELECT ?chapter ?title WHERE {
			?chapter rdf:type reg:Chapter .
			?chapter reg:title ?title .
		}
	`)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Query error: %v", err)
	}

	t.Logf("All chapters query: %d results in %v", result.Count, elapsed)

	if result.Count != 11 {
		t.Errorf("Expected 11 chapters, got %d", result.Count)
	}

	if elapsed > 100*time.Millisecond {
		t.Errorf("Query took %v, expected < 100ms", elapsed)
	}
}

func TestGDPRQuery_AllDefinitions(t *testing.T) {
	ts := loadGDPRStore(t)
	executor := NewExecutor(ts)

	start := time.Now()
	result, err := executor.ExecuteString(`
		SELECT ?term ?termText WHERE {
			?term rdf:type reg:DefinedTerm .
			?term reg:term ?termText .
		}
	`)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Query error: %v", err)
	}

	t.Logf("All definitions query: %d results in %v", result.Count, elapsed)

	if result.Count != 26 {
		t.Errorf("Expected 26 definitions, got %d", result.Count)
	}

	if elapsed > 100*time.Millisecond {
		t.Errorf("Query took %v, expected < 100ms", elapsed)
	}
}

func TestGDPRQuery_CrossReferences(t *testing.T) {
	ts := loadGDPRStore(t)
	executor := NewExecutor(ts)

	// Query all Reference objects (includes all types of references)
	start := time.Now()
	result, err := executor.ExecuteString(`
		SELECT ?ref ?text WHERE {
			?ref rdf:type reg:Reference .
			?ref reg:text ?text .
		}
	`)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Query error: %v", err)
	}

	t.Logf("All references query: %d results in %v", result.Count, elapsed)

	// Should have 255 reference objects
	if result.Count < 200 {
		t.Errorf("Expected at least 200 references, got %d", result.Count)
	}

	if elapsed > 100*time.Millisecond {
		t.Errorf("Query took %v, expected < 100ms", elapsed)
	}

	// Also query direct article-to-article references
	result2, err := executor.ExecuteString(`
		SELECT ?from ?to WHERE {
			?from reg:references ?to .
		}
	`)
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}

	t.Logf("Article-to-article references: %d results", result2.Count)

	// This is a subset - only internal references that could be resolved to articles
	if result2.Count < 50 {
		t.Errorf("Expected at least 50 article-to-article references, got %d", result2.Count)
	}
}

func TestGDPRQuery_ArticlesInChapter(t *testing.T) {
	ts := loadGDPRStore(t)
	executor := NewExecutor(ts)

	// Find articles that are part of a chapter (simpler query)
	start := time.Now()
	result, err := executor.ExecuteString(`
		SELECT ?article ?chapter WHERE {
			?article rdf:type reg:Article .
			?article reg:partOf ?chapter .
			?chapter rdf:type reg:Chapter .
		}
	`)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Query error: %v", err)
	}

	t.Logf("Articles with chapters: %d results in %v", result.Count, elapsed)

	// Should have articles linked to chapters
	if result.Count < 10 {
		t.Errorf("Expected at least 10 articles with chapters, got %d", result.Count)
	}

	if elapsed > 100*time.Millisecond {
		t.Errorf("Query took %v, expected < 100ms", elapsed)
	}
}

func TestGDPRQuery_ArticleWithFilter(t *testing.T) {
	ts := loadGDPRStore(t)
	executor := NewExecutor(ts)

	// Find articles with "erasure" in title
	start := time.Now()
	result, err := executor.ExecuteString(`
		SELECT ?article ?title WHERE {
			?article rdf:type reg:Article .
			?article reg:title ?title .
			FILTER(CONTAINS(?title, "erasure"))
		}
	`)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Query error: %v", err)
	}

	t.Logf("Articles with 'erasure': %d results in %v", result.Count, elapsed)

	if result.Count < 1 {
		t.Error("Expected at least 1 article with 'erasure' in title")
	}

	if elapsed > 100*time.Millisecond {
		t.Errorf("Query took %v, expected < 100ms", elapsed)
	}
}

func TestGDPRQuery_OrderByWithLimit(t *testing.T) {
	ts := loadGDPRStore(t)
	executor := NewExecutor(ts)

	start := time.Now()
	result, err := executor.ExecuteString(`
		SELECT ?article ?num WHERE {
			?article rdf:type reg:Article .
			?article reg:number ?num .
		} ORDER BY ?num LIMIT 10
	`)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Query error: %v", err)
	}

	t.Logf("Ordered articles (first 10): %d results in %v", result.Count, elapsed)

	if result.Count != 10 {
		t.Errorf("Expected 10 results with LIMIT, got %d", result.Count)
	}

	if elapsed > 100*time.Millisecond {
		t.Errorf("Query took %v, expected < 100ms", elapsed)
	}
}

func TestGDPRQuery_DistinctPredicates(t *testing.T) {
	ts := loadGDPRStore(t)
	executor := NewExecutor(ts)

	start := time.Now()
	result, err := executor.ExecuteString(`
		SELECT DISTINCT ?predicate WHERE {
			?s ?predicate ?o .
		}
	`)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Query error: %v", err)
	}

	t.Logf("Distinct predicates: %d results in %v", result.Count, elapsed)

	// Should have various predicates (rdf:type, reg:title, reg:text, etc.)
	if result.Count < 5 {
		t.Errorf("Expected at least 5 distinct predicates, got %d", result.Count)
	}
}

func TestGDPRQuery_ComplexJoin(t *testing.T) {
	ts := loadGDPRStore(t)
	executor := NewExecutor(ts)

	// Complex query: articles referencing other articles, with titles
	start := time.Now()
	result, err := executor.ExecuteString(`
		SELECT ?from ?fromTitle ?to ?toTitle WHERE {
			?from rdf:type reg:Article .
			?from reg:title ?fromTitle .
			?from reg:references ?to .
			?to rdf:type reg:Article .
			?to reg:title ?toTitle .
		} LIMIT 20
	`)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Query error: %v", err)
	}

	t.Logf("Article-to-article references: %d results in %v", result.Count, elapsed)

	if elapsed > 100*time.Millisecond {
		t.Errorf("Complex query took %v, expected < 100ms", elapsed)
	}
}

func TestGDPRQuery_PerformanceSummary(t *testing.T) {
	ts := loadGDPRStore(t)
	executor := NewExecutor(ts)

	stats := ts.Stats()
	t.Logf("GDPR Graph Statistics:")
	t.Logf("  Total triples: %d", stats.TotalTriples)
	t.Logf("  Unique subjects: %d", stats.UniqueSubjects)
	t.Logf("  Unique predicates: %d", stats.UniquePredicates)
	t.Logf("  Unique objects: %d", stats.UniqueObjects)

	// Run a variety of queries and measure performance
	queries := []struct {
		name  string
		query string
	}{
		{"Simple type query", `SELECT ?a WHERE { ?a rdf:type reg:Article . }`},
		{"Two patterns", `SELECT ?a ?t WHERE { ?a rdf:type reg:Article . ?a reg:title ?t . }`},
		{"Three patterns", `SELECT ?a ?t ?c WHERE { ?a rdf:type reg:Article . ?a reg:title ?t . ?a reg:partOf ?c . }`},
		{"With filter", `SELECT ?a ?t WHERE { ?a rdf:type reg:Article . ?a reg:title ?t . FILTER(CONTAINS(?t, "Right")) }`},
		{"Ordered with limit", `SELECT ?a WHERE { ?a rdf:type reg:Article . } ORDER BY ?a LIMIT 10`},
	}

	for _, q := range queries {
		start := time.Now()
		result, err := executor.ExecuteString(q.query)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("%s: error %v", q.name, err)
			continue
		}

		t.Logf("  %s: %d results in %v", q.name, result.Count, elapsed)

		if elapsed > 100*time.Millisecond {
			t.Errorf("%s: took %v, expected < 100ms", q.name, elapsed)
		}
	}
}

// Benchmark GDPR queries
func BenchmarkGDPRQuery_AllArticles(b *testing.B) {
	ts := loadGDPRStoreForBenchmark(b)
	executor := NewExecutor(ts)
	queryStr := `SELECT ?article ?title WHERE { ?article rdf:type reg:Article . ?article reg:title ?title . }`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.ExecuteString(queryStr)
	}
}

func BenchmarkGDPRQuery_ComplexJoin(b *testing.B) {
	ts := loadGDPRStoreForBenchmark(b)
	executor := NewExecutor(ts)
	queryStr := `
		SELECT ?from ?to WHERE {
			?from rdf:type reg:Article .
			?from reg:references ?to .
			?to rdf:type reg:Article .
		} LIMIT 20
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.ExecuteString(queryStr)
	}
}

func loadGDPRStoreForBenchmark(b *testing.B) *store.TripleStore {
	b.Helper()

	gdprPath := "../../testdata/gdpr.txt"
	if _, err := os.Stat(gdprPath); os.IsNotExist(err) {
		gdprPath = "../extract/testdata/gdpr.txt"
	}

	file, err := os.Open(gdprPath)
	if err != nil {
		b.Skipf("GDPR test data not available: %v", err)
	}
	defer file.Close()

	parser := extract.NewParser()
	doc, err := parser.Parse(file)
	if err != nil {
		b.Fatalf("Failed to parse GDPR: %v", err)
	}

	tripleStore := store.NewTripleStore()
	builder := store.NewGraphBuilder(tripleStore, "https://regula.dev/regulations/")

	defExtractor := extract.NewDefinitionExtractor()
	refExtractor := extract.NewReferenceExtractor()

	_, err = builder.BuildWithExtractors(doc, defExtractor, refExtractor)
	if err != nil {
		b.Fatalf("Failed to build GDPR graph: %v", err)
	}

	return tripleStore
}
