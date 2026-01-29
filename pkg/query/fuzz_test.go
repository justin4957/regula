package query

import (
	"strings"
	"testing"

	"github.com/coolbeans/regula/pkg/store"
)

// FuzzParseQuery tests the SPARQL query parser with arbitrary input.
// Run with: go test -fuzz=FuzzParseQuery -fuzztime=30s ./pkg/query/...
func FuzzParseQuery(f *testing.F) {
	// Add seed corpus with SPARQL query patterns
	seeds := []string{
		// Basic SELECT queries
		"SELECT ?s WHERE { ?s ?p ?o }",
		"SELECT ?s ?p ?o WHERE { ?s ?p ?o }",
		"SELECT * WHERE { ?s ?p ?o }",

		// With DISTINCT
		"SELECT DISTINCT ?s WHERE { ?s ?p ?o }",

		// With LIMIT/OFFSET
		"SELECT ?s WHERE { ?s ?p ?o } LIMIT 10",
		"SELECT ?s WHERE { ?s ?p ?o } LIMIT 10 OFFSET 5",

		// With ORDER BY
		"SELECT ?s WHERE { ?s ?p ?o } ORDER BY ?s",
		"SELECT ?s WHERE { ?s ?p ?o } ORDER BY DESC(?s)",
		"SELECT ?s WHERE { ?s ?p ?o } ORDER BY ASC(?s)",

		// With PREFIX
		`PREFIX reg: <https://regula.dev/ontology#>
SELECT ?article WHERE { ?article rdf:type reg:Article }`,

		// With multiple patterns
		`SELECT ?article ?title WHERE {
			?article rdf:type reg:Article .
			?article reg:title ?title .
		}`,

		// With FILTER
		`SELECT ?article WHERE {
			?article reg:title ?title .
			FILTER(CONTAINS(?title, "test"))
		}`,

		// With OPTIONAL
		`SELECT ?article ?refs WHERE {
			?article rdf:type reg:Article .
			OPTIONAL { ?article reg:references ?refs }
		}`,

		// CONSTRUCT queries
		"CONSTRUCT { ?s ?p ?o } WHERE { ?s ?p ?o }",
		`CONSTRUCT {
			?article <http://example.org/title> ?title
		} WHERE {
			?article rdf:type reg:Article .
			?article reg:title ?title
		}`,

		// Edge cases
		"",
		"SELECT",
		"WHERE",
		"SELECT WHERE {}",
		"SELECT ?s WHERE {}",
		"SELECT ?s",

		// Malformed queries
		"SELECT ?s ?p ?o",
		"SELECT * { ?s ?p ?o }",
		"?s ?p ?o",
		"SELECT ?s WHERE ?s ?p ?o",
		"SELECT ?s WHERE { ?s ?p ?o",
		"SELECT ?s WHERE ?s ?p ?o }",

		// Long queries
		"SELECT ?s WHERE { " + strings.Repeat("?s ?p ?o . ", 100) + "}",

		// Unicode
		"SELECT ?article WHERE { ?article reg:title \"Droit à l'effacement\" }",

		// Special characters in URIs
		`SELECT ?s WHERE { ?s <http://example.org/path?query=value> ?o }`,

		// Literals with escapes
		`SELECT ?s WHERE { ?s ?p "test \"quoted\" text" }`,

		// Numbers
		"SELECT ?s WHERE { ?s ?p ?o } LIMIT 0",
		"SELECT ?s WHERE { ?s ?p ?o } LIMIT -1",
		"SELECT ?s WHERE { ?s ?p ?o } LIMIT 999999999999999",
		"SELECT ?s WHERE { ?s ?p ?o } OFFSET -1",

		// Mixed case
		"select ?s where { ?s ?p ?o }",
		"Select ?s Where { ?s ?p ?o }",
		"SELECT ?s where { ?s ?p ?o }",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data string) {
		// The parser should not panic on any input
		query, err := ParseQuery(data)

		// We don't care about errors for malformed input
		if err != nil {
			return
		}

		if query == nil {
			t.Error("ParseQuery returned nil without error")
			return
		}

		// Validate the query doesn't panic
		_ = query.Validate()

		// String representation doesn't panic
		_ = query.String()

		// If it's a SELECT query, validate the structure
		if query.Select != nil {
			// ExpandPrefixes doesn't panic
			query.Select.ExpandPrefixes()
		}

		// If it's a CONSTRUCT query, validate the structure
		if query.Construct != nil {
			query.Construct.ExpandPrefixes()
		}
	})
}

// FuzzExecuteQuery tests the SPARQL query executor with arbitrary queries.
// Run with: go test -fuzz=FuzzExecuteQuery -fuzztime=30s ./pkg/query/...
func FuzzExecuteQuery(f *testing.F) {
	// Add seed corpus with executable SPARQL queries
	seeds := []string{
		// Basic queries that should execute
		"SELECT ?s WHERE { ?s ?p ?o }",
		"SELECT ?s ?p ?o WHERE { ?s ?p ?o }",
		"SELECT * WHERE { ?s ?p ?o }",
		"SELECT DISTINCT ?s WHERE { ?s ?p ?o }",
		"SELECT ?s WHERE { ?s ?p ?o } LIMIT 10",
		"SELECT ?s WHERE { ?s ?p ?o } ORDER BY ?s",

		// Queries with specific predicates
		"SELECT ?s WHERE { ?s rdf:type reg:Article }",
		"SELECT ?s ?title WHERE { ?s rdf:type reg:Article . ?s reg:title ?title }",

		// CONSTRUCT queries
		"CONSTRUCT { ?s <http://example.org/type> ?o } WHERE { ?s rdf:type ?o }",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	// Create a test store with sample data
	ts := store.NewTripleStore()
	ts.Add("GDPR:Art17", "rdf:type", "reg:Article")
	ts.Add("GDPR:Art17", "reg:title", "Right to erasure")
	ts.Add("GDPR:Art6", "rdf:type", "reg:Article")
	ts.Add("GDPR:Art6", "reg:title", "Lawfulness")
	ts.Add("GDPR:ChapterIII", "rdf:type", "reg:Chapter")

	executor := NewExecutor(ts)

	f.Fuzz(func(t *testing.T, data string) {
		// Parse first
		query, err := ParseQuery(data)
		if err != nil {
			return
		}

		// The executor should not panic on any parsed query
		if query.Type == SelectQueryType {
			result, err := executor.Execute(query)
			if err != nil {
				return
			}
			if result == nil {
				t.Error("Execute returned nil result without error")
				return
			}

			// Format methods shouldn't panic
			_, _ = result.Format(FormatTable)
			_, _ = result.Format(FormatJSON)
			_, _ = result.Format(FormatCSV)
		}

		if query.Type == ConstructQueryType {
			result, err := executor.ExecuteConstruct(query)
			if err != nil {
				return
			}
			if result == nil {
				t.Error("ExecuteConstruct returned nil result without error")
				return
			}

			// Format methods shouldn't panic
			_, _ = result.Format(FormatTurtle)
			_, _ = result.Format(FormatNTriples)
			_, _ = result.Format(FormatJSON)
		}
	})
}

// FuzzFilterEvaluation tests the filter expression evaluator with arbitrary expressions.
// Run with: go test -fuzz=FuzzFilterEvaluation -fuzztime=30s ./pkg/query/...
func FuzzFilterEvaluation(f *testing.F) {
	// Add seed corpus with filter expressions
	seeds := []string{
		// CONTAINS
		`CONTAINS(?title, "test")`,
		`CONTAINS(?title, "")`,

		// REGEX
		`REGEX(?title, "^Art")`,
		`REGEX(?title, "[0-9]+")`,
		`REGEX(?title, ".*")`,

		// STRSTARTS/STRENDS
		`STRSTARTS(?title, "Article")`,
		`STRENDS(?title, "ion")`,

		// Numeric comparisons
		`?num > 10`,
		`?num < 5`,
		`?num >= 0`,
		`?num <= 100`,
		`?num = 17`,
		`?num != 0`,

		// BOUND
		`BOUND(?x)`,
		`!BOUND(?x)`,

		// STR function
		`STR(?uri)`,

		// Edge cases
		"",
		"(",
		")",
		"()",
		"FILTER",
		"CONTAINS",
		"REGEX",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	// Create a minimal executor for testing
	ts := store.NewTripleStore()
	executor := NewExecutor(ts)

	f.Fuzz(func(t *testing.T, data string) {
		// Create a test binding
		binding := map[string]string{
			"title": "Test Article Title",
			"num":   "42",
			"uri":   "http://example.org/test",
			"x":     "bound_value",
		}

		// The evaluator should not panic on any input
		_ = executor.evaluateFilter(data, binding)
	})
}
