package store

import (
	"strings"
	"testing"
)

// --- Constructor tests ---

func TestNewTurtleSerializer(t *testing.T) {
	serializer := NewTurtleSerializer()

	if serializer == nil {
		t.Fatal("NewTurtleSerializer returned nil")
	}

	if len(serializer.prefixMappings) != 7 {
		t.Errorf("Expected 7 default prefix mappings, got %d", len(serializer.prefixMappings))
	}

	if serializer.prefixIndex["rdf"] != NamespaceRDF {
		t.Errorf("Expected rdf prefix to map to %s, got %s", NamespaceRDF, serializer.prefixIndex["rdf"])
	}

	if serializer.namespaceIndex[NamespaceReg] != "reg" {
		t.Errorf("Expected reg namespace to reverse-map to 'reg', got %s", serializer.namespaceIndex[NamespaceReg])
	}
}

func TestNewTurtleSerializer_WithCustomPrefix(t *testing.T) {
	serializer := NewTurtleSerializer(
		WithPrefix("gdpr", "https://regula.dev/regulations/GDPR#"),
	)

	if len(serializer.prefixMappings) != 8 {
		t.Errorf("Expected 8 prefix mappings (7 default + 1 custom), got %d", len(serializer.prefixMappings))
	}

	if serializer.prefixIndex["gdpr"] != "https://regula.dev/regulations/GDPR#" {
		t.Error("Custom prefix 'gdpr' not found in prefix index")
	}
}

func TestNewTurtleSerializer_WithoutDefaults(t *testing.T) {
	serializer := NewTurtleSerializer(
		WithoutDefaultPrefixes(),
		WithPrefix("custom", "https://example.org/ns#"),
	)

	if len(serializer.prefixMappings) != 1 {
		t.Errorf("Expected 1 prefix mapping (defaults cleared), got %d", len(serializer.prefixMappings))
	}

	if serializer.prefixIndex["custom"] != "https://example.org/ns#" {
		t.Error("Custom prefix not found after clearing defaults")
	}
}

// --- Escaping tests ---

func TestEscapeLiteralString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain", "hello world", "hello world"},
		{"backslash", `path\to\file`, `path\\to\\file`},
		{"double_quote", `say "hello"`, `say \"hello\"`},
		{"newline", "line1\nline2", `line1\nline2`},
		{"carriage_return", "text\rmore", `text\rmore`},
		{"tab", "col1\tcol2", `col1\tcol2`},
		{"combined", "a\"b\\c\nd", `a\"b\\c\nd`},
		{"empty", "", ""},
		{"unicode", "Recht auf Löschung", "Recht auf Löschung"},
		{"single_quotes", "'term' means something", "'term' means something"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := escapeLiteralString(testCase.input)
			if result != testCase.expected {
				t.Errorf("escapeLiteralString(%q) = %q, want %q", testCase.input, result, testCase.expected)
			}
		})
	}
}

func TestEscapeIRI(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal_uri", "https://example.org/resource", "https://example.org/resource"},
		{"angle_brackets", "https://example.org/<test>", `https://example.org/\u003Ctest\u003E`},
		{"space", "https://example.org/my resource", `https://example.org/my\u0020resource`},
		{"curly_braces", "https://example.org/{id}", `https://example.org/\u007Bid\u007D`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := escapeIRI(testCase.input)
			if result != testCase.expected {
				t.Errorf("escapeIRI(%q) = %q, want %q", testCase.input, result, testCase.expected)
			}
		})
	}
}

func TestFormatLiteral(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "hello", `"hello"`},
		{"with_quotes", `say "hi"`, `"say \"hi\""`},
		{"multiline", "line1\nline2", `"""line1\nline2"""`},
		{"empty", "", `""`},
		{"number_string", "17", `"17"`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := formatLiteral(testCase.input)
			if result != testCase.expected {
				t.Errorf("formatLiteral(%q) = %q, want %q", testCase.input, result, testCase.expected)
			}
		})
	}
}

// --- Prefixed name detection tests ---

func TestIsPrefixedName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{"reg_article", "reg:Article", true},
		{"rdf_type", "rdf:type", true},
		{"xsd_string", "xsd:string", true},
		{"gdpr_art17", "GDPR:Art17", true},
		{"plain_text", "General Data Protection Regulation", false},
		{"no_prefix", ":localName", false},
		{"empty", "", false},
		{"colon_only", ":", false},
		{"space_in_local", "reg:has space", false},
		{"number_prefix", "ns1:term", true},
		{"status_word", "resolved", false},
		{"number", "17", false},
		{"path_like", "/some/path", false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := isPrefixedName(testCase.input)
			if result != testCase.expected {
				t.Errorf("isPrefixedName(%q) = %v, want %v", testCase.input, result, testCase.expected)
			}
		})
	}
}

// --- URI compaction tests ---

func TestCompactURI(t *testing.T) {
	serializer := NewTurtleSerializer()

	testCases := []struct {
		name           string
		inputURI       string
		expectedResult string
		expectedOK     bool
	}{
		{"reg_namespace", "https://regula.dev/ontology#Article", "reg:Article", true},
		{"rdf_namespace", "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "rdf:type", true},
		{"dc_namespace", "http://purl.org/dc/terms/title", "dc:title", true},
		{"eli_namespace", "http://data.europa.eu/eli/ontology#LegalResource", "eli:LegalResource", true},
		{"no_match", "https://unknown.example.org/something", "", false},
		{"partial_match", "https://regula.dev/other#thing", "", false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, ok := serializer.compactURI(testCase.inputURI)
			if ok != testCase.expectedOK {
				t.Errorf("compactURI(%q) ok = %v, want %v", testCase.inputURI, ok, testCase.expectedOK)
			}
			if result != testCase.expectedResult {
				t.Errorf("compactURI(%q) = %q, want %q", testCase.inputURI, result, testCase.expectedResult)
			}
		})
	}
}

// --- Predicate formatting tests ---

func TestFormatPredicate(t *testing.T) {
	serializer := NewTurtleSerializer()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"rdf_type_prefixed", "rdf:type", "a"},
		{"rdf_type_full", NamespaceRDF + "type", "a"},
		{"regular_predicate", "reg:title", "reg:title"},
		{"full_uri_predicate", "https://regula.dev/ontology#title", "reg:title"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := serializer.formatPredicate(testCase.input)
			if result != testCase.expected {
				t.Errorf("formatPredicate(%q) = %q, want %q", testCase.input, result, testCase.expected)
			}
		})
	}
}

// --- Sort predicates tests ---

func TestSortPredicatesTypeFirst(t *testing.T) {
	serializer := NewTurtleSerializer()

	predicateObjectMap := map[string][]string{
		"reg:title":  {"Test"},
		"rdf:type":   {"reg:Article"},
		"reg:number": {"17"},
	}

	result := serializer.sortPredicatesTypeFirst(predicateObjectMap)

	if len(result) != 3 {
		t.Fatalf("Expected 3 predicates, got %d", len(result))
	}

	if result[0] != "rdf:type" {
		t.Errorf("Expected rdf:type first, got %q", result[0])
	}

	// Remaining should be alphabetical
	if result[1] != "reg:number" {
		t.Errorf("Expected reg:number second, got %q", result[1])
	}
	if result[2] != "reg:title" {
		t.Errorf("Expected reg:title third, got %q", result[2])
	}
}

func TestSortPredicatesTypeFirst_NoType(t *testing.T) {
	serializer := NewTurtleSerializer()

	predicateObjectMap := map[string][]string{
		"reg:title":  {"Test"},
		"reg:number": {"17"},
	}

	result := serializer.sortPredicatesTypeFirst(predicateObjectMap)

	if len(result) != 2 {
		t.Fatalf("Expected 2 predicates, got %d", len(result))
	}

	if result[0] != "reg:number" {
		t.Errorf("Expected reg:number first (alphabetical), got %q", result[0])
	}
}

// --- Serialization integration tests ---

func TestSerialize_EmptyStore(t *testing.T) {
	store := NewTripleStore()
	serializer := NewTurtleSerializer()

	output := serializer.Serialize(store)

	if !strings.Contains(output, "@prefix rdf:") {
		t.Error("Empty store output should contain prefix declarations")
	}

	// Should only contain prefix lines and a blank line
	lines := strings.Split(strings.TrimSpace(output), "\n")
	nonEmptyLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines++
		}
	}

	if nonEmptyLines != 7 {
		t.Errorf("Expected 7 prefix lines for empty store, got %d non-empty lines", nonEmptyLines)
	}
}

func TestSerialize_SingleTriple(t *testing.T) {
	store := NewTripleStore()
	store.Add("GDPR:Art1", "rdf:type", "reg:Article")

	serializer := NewTurtleSerializer()
	output := serializer.Serialize(store)

	if !strings.Contains(output, "GDPR:Art1 a reg:Article .") {
		t.Errorf("Expected single triple with 'a' shorthand, got:\n%s", output)
	}
}

func TestSerialize_SubjectGrouping(t *testing.T) {
	store := NewTripleStore()
	store.Add("GDPR:Art1", "rdf:type", "reg:Article")
	store.Add("GDPR:Art1", "reg:number", "1")
	store.Add("GDPR:Art1", "reg:title", "Subject-matter and objectives")

	serializer := NewTurtleSerializer()
	output := serializer.Serialize(store)

	// Should have semicolons for grouping
	if !strings.Contains(output, ";") {
		t.Error("Expected semicolons for subject grouping")
	}

	// Should have exactly one period for the subject group
	subjectBlock := extractSubjectBlock(output, "GDPR:Art1")
	if subjectBlock == "" {
		t.Fatal("Could not find GDPR:Art1 subject block")
	}

	if !strings.HasSuffix(strings.TrimSpace(subjectBlock), ".") {
		t.Error("Subject block should end with period")
	}
}

func TestSerialize_MultipleObjects(t *testing.T) {
	store := NewTripleStore()
	store.Add("GDPR:Art17", "reg:references", "GDPR:Art6")
	store.Add("GDPR:Art17", "reg:references", "GDPR:Art9")

	serializer := NewTurtleSerializer()
	output := serializer.Serialize(store)

	// Should have comma for multiple objects
	if !strings.Contains(output, ",") {
		t.Error("Expected comma for multiple objects of same predicate")
	}

	if !strings.Contains(output, "GDPR:Art6") || !strings.Contains(output, "GDPR:Art9") {
		t.Error("Expected both referenced articles in output")
	}
}

func TestSerialize_MultipleSubjects(t *testing.T) {
	store := NewTripleStore()
	store.Add("GDPR:Art1", "rdf:type", "reg:Article")
	store.Add("GDPR:Art2", "rdf:type", "reg:Article")

	serializer := NewTurtleSerializer()
	output := serializer.Serialize(store)

	// Both subjects should appear
	if !strings.Contains(output, "GDPR:Art1") || !strings.Contains(output, "GDPR:Art2") {
		t.Error("Expected both subjects in output")
	}

	// Should have a blank line between subject groups
	if !strings.Contains(output, ".\n\n") {
		t.Error("Expected blank line between subject groups")
	}
}

func TestSerialize_RDFTypeShorthand(t *testing.T) {
	store := NewTripleStore()
	store.Add("GDPR:Art1", "rdf:type", "reg:Article")

	serializer := NewTurtleSerializer()
	output := serializer.Serialize(store)

	if !strings.Contains(output, " a ") {
		t.Errorf("Expected 'a' shorthand for rdf:type, got:\n%s", output)
	}

	if strings.Contains(output, "rdf:type") && !strings.Contains(output, "@prefix rdf:") {
		t.Error("rdf:type should be rendered as 'a', not as 'rdf:type' in triple")
	}
}

func TestSerialize_FullURICompaction(t *testing.T) {
	store := NewTripleStore()
	store.Add("https://regula.dev/ontology#Art1", "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "https://regula.dev/ontology#Article")

	serializer := NewTurtleSerializer()
	output := serializer.Serialize(store)

	if !strings.Contains(output, "reg:Art1") {
		t.Error("Expected subject URI to be compacted to reg:Art1")
	}

	if !strings.Contains(output, "reg:Article") {
		t.Error("Expected object URI to be compacted to reg:Article")
	}

	// rdf:type should become "a"
	if strings.Contains(output, "rdf:type") && !strings.Contains(output, "@prefix") {
		t.Error("Expected full RDF type URI to be compacted to 'a'")
	}
}

func TestSerialize_FullURIWithBrackets(t *testing.T) {
	store := NewTripleStore()
	store.Add("https://unknown.example.org/thing", "rdf:type", "reg:Article")

	serializer := NewTurtleSerializer()
	output := serializer.Serialize(store)

	if !strings.Contains(output, "<https://unknown.example.org/thing>") {
		t.Errorf("Expected uncompactable URI in angle brackets, got:\n%s", output)
	}
}

func TestSerialize_LiteralEscaping(t *testing.T) {
	store := NewTripleStore()
	store.Add("GDPR:Art17", "reg:title", `Right to erasure ('right to be forgotten')`)

	serializer := NewTurtleSerializer()
	output := serializer.Serialize(store)

	if !strings.Contains(output, `"Right to erasure ('right to be forgotten')"`) {
		t.Errorf("Expected properly quoted literal, got:\n%s", output)
	}
}

func TestSerialize_LiteralWithSpecialChars(t *testing.T) {
	store := NewTripleStore()
	store.Add("test:Subject", "reg:text", "line1\nline2")

	serializer := NewTurtleSerializer()
	output := serializer.Serialize(store)

	if !strings.Contains(output, `"""`) {
		t.Errorf("Expected triple-quoted literal for multiline text, got:\n%s", output)
	}
}

func TestSerialize_DeterministicOutput(t *testing.T) {
	store := NewTripleStore()
	store.Add("GDPR:Art1", "rdf:type", "reg:Article")
	store.Add("GDPR:Art1", "reg:number", "1")
	store.Add("GDPR:Art1", "reg:title", "Subject-matter and objectives")
	store.Add("GDPR:Art2", "rdf:type", "reg:Article")
	store.Add("GDPR:Art2", "reg:number", "2")

	serializer := NewTurtleSerializer()

	firstOutput := serializer.Serialize(store)
	secondOutput := serializer.Serialize(store)

	if firstOutput != secondOutput {
		t.Error("Expected deterministic output, but two serializations differ")
	}
}

func TestSerialize_CustomPrefixes(t *testing.T) {
	store := NewTripleStore()
	store.Add("https://regula.dev/regulations/GDPR#Art1", "rdf:type", "reg:Article")

	serializer := NewTurtleSerializer(
		WithPrefix("gdpr", "https://regula.dev/regulations/GDPR#"),
	)
	output := serializer.Serialize(store)

	if !strings.Contains(output, "@prefix gdpr: <https://regula.dev/regulations/GDPR#> .") {
		t.Error("Expected custom prefix declaration in output")
	}

	if !strings.Contains(output, "gdpr:Art1") {
		t.Errorf("Expected URI to be compacted with custom prefix, got:\n%s", output)
	}
}

func TestSerialize_PrefixDeclarations(t *testing.T) {
	serializer := NewTurtleSerializer()
	store := NewTripleStore()

	output := serializer.Serialize(store)

	expectedPrefixes := []string{
		"@prefix dc:",
		"@prefix eli:",
		"@prefix frbr:",
		"@prefix rdf:",
		"@prefix rdfs:",
		"@prefix reg:",
		"@prefix xsd:",
	}

	for _, expectedPrefix := range expectedPrefixes {
		if !strings.Contains(output, expectedPrefix) {
			t.Errorf("Missing prefix declaration: %s", expectedPrefix)
		}
	}
}

func TestSerialize_NoPrefixes(t *testing.T) {
	store := NewTripleStore()
	store.Add("GDPR:Art1", "rdf:type", "reg:Article")

	serializer := NewTurtleSerializer(WithoutDefaultPrefixes())
	output := serializer.Serialize(store)

	if strings.Contains(output, "@prefix") {
		t.Error("Expected no prefix declarations when defaults are cleared")
	}

	if !strings.Contains(output, "GDPR:Art1") {
		t.Error("Expected triple content even without prefixes")
	}
}

func TestSerialize_TypeFirstInSubjectGroup(t *testing.T) {
	store := NewTripleStore()
	store.Add("GDPR:Art1", "reg:title", "Test Title")
	store.Add("GDPR:Art1", "rdf:type", "reg:Article")
	store.Add("GDPR:Art1", "reg:number", "1")

	serializer := NewTurtleSerializer()
	output := serializer.Serialize(store)

	subjectBlock := extractSubjectBlock(output, "GDPR:Art1")

	// 'a' (rdf:type) should come before other predicates
	typeIndex := strings.Index(subjectBlock, " a ")
	titleIndex := strings.Index(subjectBlock, "reg:title")
	numberIndex := strings.Index(subjectBlock, "reg:number")

	if typeIndex < 0 {
		t.Fatal("Expected 'a' shorthand in subject block")
	}
	if titleIndex < 0 || numberIndex < 0 {
		t.Fatal("Expected reg:title and reg:number in subject block")
	}
	if typeIndex > titleIndex || typeIndex > numberIndex {
		t.Error("Expected rdf:type (as 'a') to appear before other predicates")
	}
}

// --- GDPR Integration test ---

func TestSerialize_GDPRIntegration(t *testing.T) {
	gdprDocument := loadGDPRDocument(t)

	tripleStore := NewTripleStore()
	graphBuilder := NewGraphBuilder(tripleStore, "https://regula.dev/regulations/")

	buildStats, err := graphBuilder.Build(gdprDocument)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	serializer := NewTurtleSerializer()
	output := serializer.Serialize(tripleStore)

	// Verify prefix declarations present
	if !strings.Contains(output, "@prefix rdf:") {
		t.Error("Missing rdf prefix declaration")
	}
	if !strings.Contains(output, "@prefix reg:") {
		t.Error("Missing reg prefix declaration")
	}

	// Verify rdf:type shorthand used
	if !strings.Contains(output, " a ") {
		t.Error("Expected 'a' shorthand for rdf:type")
	}

	// Verify some content structure
	if !strings.Contains(output, "reg:Article") {
		t.Error("Expected reg:Article type in output")
	}
	if !strings.Contains(output, "reg:Chapter") {
		t.Error("Expected reg:Chapter type in output")
	}

	// Verify substantial output
	lines := strings.Split(output, "\n")
	if len(lines) < 100 {
		t.Errorf("Expected substantial output, got %d lines", len(lines))
	}

	// Verify no stray angle brackets for known prefixed names
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "@prefix") {
			continue
		}
		// Lines should not contain <reg: or <rdf: (these should be compacted)
		if strings.Contains(trimmed, "<reg:") || strings.Contains(trimmed, "<rdf:") {
			t.Errorf("Found un-compacted prefixed name in angle brackets: %s", trimmed)
			break
		}
	}

	t.Logf("GDPR Turtle output: %d lines, %d bytes, %d triples",
		len(lines), len(output), buildStats.TotalTriples)
}

// --- Concurrent access test ---

func TestSerialize_ConcurrentAccess(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	serializer := NewTurtleSerializer()

	done := make(chan string, 10)

	for i := 0; i < 10; i++ {
		go func() {
			result := serializer.Serialize(store)
			done <- result
		}()
	}

	var firstResult string
	for i := 0; i < 10; i++ {
		result := <-done
		if i == 0 {
			firstResult = result
		} else if result != firstResult {
			t.Error("Concurrent serializations produced different output")
		}
	}
}

// --- Helper ---

func extractSubjectBlock(turtleOutput, subject string) string {
	lines := strings.Split(turtleOutput, "\n")
	var blockLines []string
	inBlock := false

	for _, line := range lines {
		if strings.HasPrefix(line, subject+" ") {
			inBlock = true
		}
		if inBlock {
			blockLines = append(blockLines, line)
			if strings.HasSuffix(strings.TrimSpace(line), ".") {
				break
			}
		}
	}

	return strings.Join(blockLines, "\n")
}
