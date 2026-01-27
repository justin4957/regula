package query

import (
	"strings"
	"testing"
)

func TestParseQuery_SimpleSelect(t *testing.T) {
	queryStr := `
		PREFIX reg: <https://regula.dev/ontology#>
		SELECT ?article WHERE {
			?article rdf:type reg:Article .
		}
	`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	if query.Type != SelectQueryType {
		t.Errorf("Type = %v, want %v", query.Type, SelectQueryType)
	}

	if len(query.Select.Variables) != 1 || query.Select.Variables[0] != "?article" {
		t.Errorf("Variables = %v, want [?article]", query.Select.Variables)
	}

	if len(query.Select.Where) != 1 {
		t.Fatalf("Where patterns = %d, want 1", len(query.Select.Where))
	}

	pattern := query.Select.Where[0]
	if pattern.Subject != "?article" {
		t.Errorf("Subject = %v, want ?article", pattern.Subject)
	}
	if pattern.Predicate != "rdf:type" {
		t.Errorf("Predicate = %v, want rdf:type", pattern.Predicate)
	}
	if pattern.Object != "reg:Article" {
		t.Errorf("Object = %v, want reg:Article", pattern.Object)
	}
}

func TestParseQuery_MultipleVariables(t *testing.T) {
	queryStr := `SELECT ?s ?p ?o WHERE { ?s ?p ?o . }`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	if len(query.Select.Variables) != 3 {
		t.Errorf("Variables count = %d, want 3", len(query.Select.Variables))
	}

	expected := []string{"?s", "?p", "?o"}
	for i, v := range expected {
		if query.Select.Variables[i] != v {
			t.Errorf("Variable[%d] = %s, want %s", i, query.Select.Variables[i], v)
		}
	}
}

func TestParseQuery_SelectAll(t *testing.T) {
	queryStr := `SELECT * WHERE { ?s ?p ?o . }`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	if len(query.Select.Variables) != 1 || query.Select.Variables[0] != "*" {
		t.Errorf("Variables = %v, want [*]", query.Select.Variables)
	}
}

func TestParseQuery_WithDistinct(t *testing.T) {
	queryStr := `SELECT DISTINCT ?s WHERE { ?s ?p ?o . }`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	if !query.Select.Distinct {
		t.Error("Distinct should be true")
	}
}

func TestParseQuery_WithLimit(t *testing.T) {
	queryStr := `SELECT ?s WHERE { ?s ?p ?o . } LIMIT 10`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	if query.Select.Limit != 10 {
		t.Errorf("Limit = %d, want 10", query.Select.Limit)
	}
}

func TestParseQuery_WithOffset(t *testing.T) {
	queryStr := `SELECT ?s WHERE { ?s ?p ?o . } OFFSET 5`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	if query.Select.Offset != 5 {
		t.Errorf("Offset = %d, want 5", query.Select.Offset)
	}
}

func TestParseQuery_WithLimitAndOffset(t *testing.T) {
	queryStr := `SELECT ?s WHERE { ?s ?p ?o . } LIMIT 100 OFFSET 10`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	if query.Select.Limit != 100 {
		t.Errorf("Limit = %d, want 100", query.Select.Limit)
	}
	if query.Select.Offset != 10 {
		t.Errorf("Offset = %d, want 10", query.Select.Offset)
	}
}

func TestParseQuery_WithOrderBy(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantVar  string
		wantDesc bool
	}{
		{
			name:     "simple order by",
			query:    `SELECT ?s WHERE { ?s ?p ?o . } ORDER BY ?s`,
			wantVar:  "?s",
			wantDesc: false,
		},
		{
			name:     "ascending",
			query:    `SELECT ?s WHERE { ?s ?p ?o . } ORDER BY ASC(?s)`,
			wantVar:  "?s",
			wantDesc: false,
		},
		{
			name:     "descending",
			query:    `SELECT ?s WHERE { ?s ?p ?o . } ORDER BY DESC(?s)`,
			wantVar:  "?s",
			wantDesc: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			query, err := ParseQuery(tc.query)
			if err != nil {
				t.Fatalf("ParseQuery() error = %v", err)
			}

			if len(query.Select.OrderBy) != 1 {
				t.Fatalf("OrderBy count = %d, want 1", len(query.Select.OrderBy))
			}

			if query.Select.OrderBy[0].Variable != tc.wantVar {
				t.Errorf("OrderBy variable = %s, want %s", query.Select.OrderBy[0].Variable, tc.wantVar)
			}
			if query.Select.OrderBy[0].Descending != tc.wantDesc {
				t.Errorf("OrderBy descending = %v, want %v", query.Select.OrderBy[0].Descending, tc.wantDesc)
			}
		})
	}
}

func TestParseQuery_WithFilter(t *testing.T) {
	queryStr := `
		SELECT ?article WHERE {
			?article rdf:type reg:Article .
			FILTER(CONTAINS(?article, "Art17"))
		}
	`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	if len(query.Select.Filters) != 1 {
		t.Fatalf("Filters count = %d, want 1", len(query.Select.Filters))
	}

	if !strings.Contains(query.Select.Filters[0].Expression, "CONTAINS") {
		t.Errorf("Filter expression should contain CONTAINS, got %s", query.Select.Filters[0].Expression)
	}
}

func TestParseQuery_WithMultipleFilters(t *testing.T) {
	queryStr := `
		SELECT ?article ?num WHERE {
			?article rdf:type reg:Article .
			?article reg:number ?num .
			FILTER(?num > 10)
			FILTER(?num < 50)
		}
	`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	if len(query.Select.Filters) != 2 {
		t.Errorf("Filters count = %d, want 2", len(query.Select.Filters))
	}
}

func TestParseQuery_WithOptional(t *testing.T) {
	queryStr := `
		SELECT ?article ?title WHERE {
			?article rdf:type reg:Article .
			OPTIONAL { ?article reg:title ?title . }
		}
	`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	if len(query.Select.Optional) != 1 {
		t.Fatalf("Optional count = %d, want 1", len(query.Select.Optional))
	}

	if len(query.Select.Optional[0]) != 1 {
		t.Fatalf("Optional[0] patterns = %d, want 1", len(query.Select.Optional[0]))
	}

	optPattern := query.Select.Optional[0][0]
	if optPattern.Subject != "?article" {
		t.Errorf("Optional subject = %s, want ?article", optPattern.Subject)
	}
}

func TestParseQuery_Prefixes(t *testing.T) {
	queryStr := `
		PREFIX reg: <https://regula.dev/ontology#>
		PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
		SELECT ?article WHERE {
			?article rdf:type reg:Article .
		}
	`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	if len(query.Select.Prefixes) != 2 {
		t.Errorf("Prefixes count = %d, want 2", len(query.Select.Prefixes))
	}

	if query.Select.Prefixes["reg"] != "https://regula.dev/ontology#" {
		t.Errorf("reg prefix = %v", query.Select.Prefixes["reg"])
	}

	if query.Select.Prefixes["rdf"] != "http://www.w3.org/1999/02/22-rdf-syntax-ns#" {
		t.Errorf("rdf prefix = %v", query.Select.Prefixes["rdf"])
	}
}

func TestParseQuery_ExpandPrefixes(t *testing.T) {
	queryStr := `
		PREFIX reg: <https://regula.dev/ontology#>
		PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
		SELECT ?article WHERE {
			?article rdf:type reg:Article .
		}
	`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	// Expand prefixes
	query.Select.ExpandPrefixes()

	pattern := query.Select.Where[0]
	if pattern.Predicate != "<http://www.w3.org/1999/02/22-rdf-syntax-ns#type>" {
		t.Errorf("Expanded predicate = %s, want <http://www.w3.org/1999/02/22-rdf-syntax-ns#type>", pattern.Predicate)
	}
	if pattern.Object != "<https://regula.dev/ontology#Article>" {
		t.Errorf("Expanded object = %s, want <https://regula.dev/ontology#Article>", pattern.Object)
	}
}

func TestParseQuery_MultiplePatterns(t *testing.T) {
	queryStr := `
		SELECT ?article ?title WHERE {
			?article rdf:type reg:Article .
			?article reg:title ?title .
		}
	`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	if len(query.Select.Where) != 2 {
		t.Fatalf("Where patterns = %d, want 2", len(query.Select.Where))
	}

	// First pattern
	if query.Select.Where[0].Subject != "?article" {
		t.Errorf("Pattern 1 subject = %s, want ?article", query.Select.Where[0].Subject)
	}

	// Second pattern
	if query.Select.Where[1].Predicate != "reg:title" {
		t.Errorf("Pattern 2 predicate = %s, want reg:title", query.Select.Where[1].Predicate)
	}
}

func TestParseQuery_ShorthandA(t *testing.T) {
	queryStr := `SELECT ?article WHERE { ?article a reg:Article . }`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	if query.Select.Where[0].Predicate != "rdf:type" {
		t.Errorf("Predicate 'a' should be expanded to rdf:type, got %s", query.Select.Where[0].Predicate)
	}
}

func TestParseQuery_URISubject(t *testing.T) {
	queryStr := `SELECT ?p ?o WHERE { <https://regula.dev/GDPR:Art17> ?p ?o . }`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	if query.Select.Where[0].Subject != "<https://regula.dev/GDPR:Art17>" {
		t.Errorf("Subject = %s, want <https://regula.dev/GDPR:Art17>", query.Select.Where[0].Subject)
	}
}

func TestParseQuery_LiteralObject(t *testing.T) {
	queryStr := `SELECT ?article WHERE { ?article reg:title "Right to erasure" . }`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	if query.Select.Where[0].Object != `"Right to erasure"` {
		t.Errorf("Object = %s, want \"Right to erasure\"", query.Select.Where[0].Object)
	}
}

func TestParseQuery_RegulaExamples(t *testing.T) {
	// Test queries from the issue requirements
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name: "all articles with titles",
			query: `SELECT ?article ?title WHERE {
				?article rdf:type reg:Article .
				?article reg:title ?title
			}`,
			wantErr: false,
		},
		{
			name: "defined terms with limit",
			query: `SELECT ?term WHERE {
				?term rdf:type reg:DefinedTerm
			} LIMIT 10`,
			wantErr: false,
		},
		{
			name: "cross-references ordered",
			query: `SELECT ?from ?to WHERE {
				?from reg:references ?to
			} ORDER BY ?from`,
			wantErr: false,
		},
		{
			name: "articles in chapter",
			query: `
				PREFIX reg: <https://regula.dev/ontology#>
				SELECT ?article ?title WHERE {
					?article rdf:type reg:Article .
					?article reg:partOf <GDPR:ChapterIII> .
					?article reg:title ?title .
				}
			`,
			wantErr: false,
		},
		{
			name: "filter with contains",
			query: `SELECT ?article WHERE {
				?article rdf:type reg:Article .
				?article reg:title ?title .
				FILTER(CONTAINS(?title, "erasure"))
			}`,
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			query, err := ParseQuery(tc.query)
			if (err != nil) != tc.wantErr {
				t.Errorf("ParseQuery() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !tc.wantErr && query == nil {
				t.Error("ParseQuery() returned nil query without error")
			}
		})
	}
}

func TestParseQuery_Errors(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr string
	}{
		{
			name:    "empty query",
			query:   "",
			wantErr: "empty query",
		},
		{
			name:    "no WHERE clause",
			query:   "SELECT ?s",
			wantErr: "missing WHERE clause",
		},
		{
			name:    "missing braces",
			query:   "SELECT ?s WHERE ?s ?p ?o",
			wantErr: "missing braces",
		},
		{
			name:    "unsupported query type",
			query:   "INSERT DATA { <s> <p> <o> }",
			wantErr: "unsupported query type",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseQuery(tc.query)
			if err == nil {
				t.Error("ParseQuery() should return error")
				return
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("ParseQuery() error = %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestIsVariable(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"?var", true},
		{"?article", true},
		{"?s", true},
		{"article", false},
		{"<uri>", false},
		{"", false},
		{"reg:Article", false},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := IsVariable(tc.input)
			if got != tc.want {
				t.Errorf("IsVariable(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestIsURI(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"<http://example.org>", true},
		{"<https://regula.dev/GDPR#Art17>", true},
		{"<#local>", true},
		{"?var", false},
		{"reg:Article", false},
		{"", false},
		{"<>", false},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := IsURI(tc.input)
			if got != tc.want {
				t.Errorf("IsURI(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestIsLiteral(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{`"hello"`, true},
		{`"Right to erasure"`, true},
		{`""`, false},
		{"?var", false},
		{"<uri>", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := IsLiteral(tc.input)
			if got != tc.want {
				t.Errorf("IsLiteral(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestIsPrefixed(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"reg:Article", true},
		{"rdf:type", true},
		{"prefix:local", true},
		{"?var", false},
		{"<uri>", false},
		{`"literal"`, false},
		{"nocolon", false},
		{":startcolon", false},
		{"endcolon:", false},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := IsPrefixed(tc.input)
			if got != tc.want {
				t.Errorf("IsPrefixed(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestStripVariable(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"?var", "var"},
		{"?article", "article"},
		{"article", "article"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := StripVariable(tc.input)
			if got != tc.want {
				t.Errorf("StripVariable(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestStripURI(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<http://example.org>", "http://example.org"},
		{"<#local>", "#local"},
		{"plain", "plain"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := StripURI(tc.input)
			if got != tc.want {
				t.Errorf("StripURI(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestStripLiteral(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`"hello"`, "hello"},
		{`"Right to erasure"`, "Right to erasure"},
		{"plain", "plain"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := StripLiteral(tc.input)
			if got != tc.want {
				t.Errorf("StripLiteral(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestQuery_Validate(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantErrors int
	}{
		{
			name:       "valid query",
			query:      `SELECT ?s WHERE { ?s ?p ?o . }`,
			wantErrors: 0,
		},
		{
			name:       "unbound variable in select",
			query:      `SELECT ?s ?unbound WHERE { ?s ?p ?o . }`,
			wantErrors: 1,
		},
		{
			name:       "order by variable not in select",
			query:      `SELECT ?s WHERE { ?s ?p ?o . } ORDER BY ?other`,
			wantErrors: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			query, err := ParseQuery(tc.query)
			if err != nil {
				t.Fatalf("ParseQuery() error = %v", err)
			}

			errors := query.Validate()
			if len(errors) != tc.wantErrors {
				t.Errorf("Validate() returned %d errors, want %d: %v", len(errors), tc.wantErrors, errors)
			}
		})
	}
}

func TestQuery_String(t *testing.T) {
	queryStr := `
		PREFIX reg: <https://regula.dev/ontology#>
		SELECT ?article ?title WHERE {
			?article rdf:type reg:Article .
			?article reg:title ?title .
		} ORDER BY ?article LIMIT 10
	`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	str := query.String()

	// Check that key elements are present
	if !strings.Contains(str, "SELECT") {
		t.Error("String() should contain SELECT")
	}
	if !strings.Contains(str, "?article") {
		t.Error("String() should contain ?article")
	}
	if !strings.Contains(str, "WHERE") {
		t.Error("String() should contain WHERE")
	}
	if !strings.Contains(str, "LIMIT 10") {
		t.Error("String() should contain LIMIT 10")
	}
}

func TestParseQuery_ComplexFilter(t *testing.T) {
	queryStr := `
		SELECT ?article WHERE {
			?article rdf:type reg:Article .
			?article reg:number ?num .
			FILTER(REGEX(STR(?article), "Art[0-9]+"))
		}
	`

	query, err := ParseQuery(queryStr)
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	if len(query.Select.Filters) != 1 {
		t.Fatalf("Filters count = %d, want 1", len(query.Select.Filters))
	}

	// The expression should contain the nested function calls
	if !strings.Contains(query.Select.Filters[0].Expression, "REGEX") {
		t.Errorf("Filter should contain REGEX, got %s", query.Select.Filters[0].Expression)
	}
}
