package store

import (
	"strings"
	"testing"
)

func TestNewRDFXMLSerializer(t *testing.T) {
	serializer := NewRDFXMLSerializer()

	if len(serializer.prefixMappings) == 0 {
		t.Error("Expected default prefix mappings")
	}

	if serializer.prefixIndex["rdf"] != NamespaceRDF {
		t.Errorf("Expected rdf prefix mapped to %s", NamespaceRDF)
	}

	if serializer.prefixIndex["reg"] != NamespaceReg {
		t.Errorf("Expected reg prefix mapped to %s", NamespaceReg)
	}

	if serializer.namespaceIndex[NamespaceRDF] != "rdf" {
		t.Error("Expected reverse index for rdf namespace")
	}
}

func TestNewRDFXMLSerializer_WithOptions(t *testing.T) {
	serializer := NewRDFXMLSerializer(
		WithRDFXMLPrefix("custom", "http://example.org/custom#"),
	)

	if serializer.prefixIndex["custom"] != "http://example.org/custom#" {
		t.Error("Expected custom prefix to be registered")
	}

	// Default prefixes should still be present
	if serializer.prefixIndex["rdf"] != NamespaceRDF {
		t.Error("Expected default rdf prefix to remain")
	}
}

func TestNewRDFXMLSerializer_WithoutDefaults(t *testing.T) {
	serializer := NewRDFXMLSerializer(
		WithoutRDFXMLDefaultPrefixes(),
		WithRDFXMLPrefix("rdf", NamespaceRDF),
	)

	if len(serializer.prefixMappings) != 1 {
		t.Errorf("Expected 1 prefix mapping, got %d", len(serializer.prefixMappings))
	}
}

func TestRDFXMLSerialize_EmptyStore(t *testing.T) {
	serializer := NewRDFXMLSerializer()
	tripleStore := NewTripleStore()

	output := serializer.Serialize(tripleStore)

	if !strings.Contains(output, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>") {
		t.Error("Missing XML declaration")
	}

	if !strings.Contains(output, "<rdf:RDF") {
		t.Error("Missing rdf:RDF opening element")
	}

	if !strings.Contains(output, "</rdf:RDF>") {
		t.Error("Missing rdf:RDF closing element")
	}

	if strings.Contains(output, "<rdf:Description") {
		t.Error("Empty store should not contain Description elements")
	}
}

func TestRDFXMLSerialize_SingleTriple(t *testing.T) {
	serializer := NewRDFXMLSerializer()
	tripleStore := NewTripleStore()
	tripleStore.Add("http://example.org/article1", "reg:title", "Test Article")

	output := serializer.Serialize(tripleStore)

	if !strings.Contains(output, `rdf:about="http://example.org/article1"`) {
		t.Error("Missing subject URI in rdf:about")
	}

	if !strings.Contains(output, "<reg:title>Test Article</reg:title>") {
		t.Error("Missing literal property element")
	}
}

func TestRDFXMLSerialize_SubjectGrouping(t *testing.T) {
	serializer := NewRDFXMLSerializer()
	tripleStore := NewTripleStore()
	tripleStore.Add("http://example.org/article1", "rdf:type", "reg:Article")
	tripleStore.Add("http://example.org/article1", "reg:title", "Right to erasure")
	tripleStore.Add("http://example.org/article1", "reg:number", "17")

	output := serializer.Serialize(tripleStore)

	// Should have exactly one Description block
	descriptionCount := strings.Count(output, "<rdf:Description")
	if descriptionCount != 1 {
		t.Errorf("Expected 1 Description block, got %d", descriptionCount)
	}

	// All properties should be present
	if !strings.Contains(output, "<rdf:type") {
		t.Error("Missing rdf:type property")
	}
	if !strings.Contains(output, "<reg:title>Right to erasure</reg:title>") {
		t.Error("Missing reg:title property")
	}
	if !strings.Contains(output, "<reg:number>17</reg:number>") {
		t.Error("Missing reg:number property")
	}
}

func TestRDFXMLSerialize_MultipleSubjects(t *testing.T) {
	serializer := NewRDFXMLSerializer()
	tripleStore := NewTripleStore()
	tripleStore.Add("http://example.org/article1", "reg:title", "First")
	tripleStore.Add("http://example.org/article2", "reg:title", "Second")

	output := serializer.Serialize(tripleStore)

	descriptionCount := strings.Count(output, "<rdf:Description")
	if descriptionCount != 2 {
		t.Errorf("Expected 2 Description blocks, got %d", descriptionCount)
	}

	if !strings.Contains(output, `rdf:about="http://example.org/article1"`) {
		t.Error("Missing article1 subject")
	}
	if !strings.Contains(output, `rdf:about="http://example.org/article2"`) {
		t.Error("Missing article2 subject")
	}
}

func TestRDFXMLSerialize_URIObject(t *testing.T) {
	serializer := NewRDFXMLSerializer()
	tripleStore := NewTripleStore()
	tripleStore.Add("http://example.org/article17", "reg:references", "http://example.org/article6")

	output := serializer.Serialize(tripleStore)

	if !strings.Contains(output, `rdf:resource="http://example.org/article6"`) {
		t.Error("URI objects should use rdf:resource attribute")
	}

	// Should be a self-closing element
	if !strings.Contains(output, "/>") {
		t.Error("URI object elements should be self-closing")
	}
}

func TestRDFXMLSerialize_LiteralObject(t *testing.T) {
	serializer := NewRDFXMLSerializer()
	tripleStore := NewTripleStore()
	tripleStore.Add("http://example.org/article17", "reg:title", "Right to erasure")

	output := serializer.Serialize(tripleStore)

	if !strings.Contains(output, "<reg:title>Right to erasure</reg:title>") {
		t.Error("Literal objects should be text content")
	}
}

func TestRDFXMLSerialize_RDFType(t *testing.T) {
	serializer := NewRDFXMLSerializer()
	tripleStore := NewTripleStore()
	tripleStore.Add("http://example.org/article17", "rdf:type", "reg:Article")

	output := serializer.Serialize(tripleStore)

	// rdf:type should use rdf:resource with expanded URI
	if !strings.Contains(output, `<rdf:type rdf:resource="`) {
		t.Error("rdf:type should use rdf:resource attribute")
	}

	if !strings.Contains(output, NamespaceReg+"Article") {
		t.Error("rdf:type resource should contain expanded Article URI")
	}
}

func TestRDFXMLSerialize_XMLEscaping(t *testing.T) {
	serializer := NewRDFXMLSerializer()
	tripleStore := NewTripleStore()
	tripleStore.Add("http://example.org/article1", "reg:title", "Rights & Obligations")
	tripleStore.Add("http://example.org/article2", "reg:title", "Section <A>")
	tripleStore.Add("http://example.org/article3", "reg:title", `Use of "quotes"`)

	output := serializer.Serialize(tripleStore)

	if !strings.Contains(output, "Rights &amp; Obligations") {
		t.Error("Ampersand should be escaped in text content")
	}

	if !strings.Contains(output, "Section &lt;A&gt;") {
		t.Error("Angle brackets should be escaped in text content")
	}

	// Quotes in text content don't need escaping (only in attributes)
	if !strings.Contains(output, `Use of "quotes"`) {
		t.Error("Quotes in text content should not be escaped")
	}
}

func TestRDFXMLSerialize_NamespaceDeclarations(t *testing.T) {
	serializer := NewRDFXMLSerializer()
	tripleStore := NewTripleStore()
	tripleStore.Add("http://example.org/article1", "rdf:type", "reg:Article")

	output := serializer.Serialize(tripleStore)

	expectedNamespaces := []struct {
		prefix    string
		namespace string
	}{
		{"rdf", NamespaceRDF},
		{"reg", NamespaceReg},
		{"rdfs", NamespaceRDFS},
		{"dc", NamespaceDC},
		{"eli", NamespaceELI},
	}

	for _, expectedNamespace := range expectedNamespaces {
		declaration := `xmlns:` + expectedNamespace.prefix + `="` + expectedNamespace.namespace + `"`
		if !strings.Contains(output, declaration) {
			t.Errorf("Missing namespace declaration for %s", expectedNamespace.prefix)
		}
	}
}

func TestRDFXMLSerialize_MultipleObjects(t *testing.T) {
	serializer := NewRDFXMLSerializer()
	tripleStore := NewTripleStore()
	tripleStore.Add("http://example.org/article17", "reg:references", "http://example.org/article6")
	tripleStore.Add("http://example.org/article17", "reg:references", "http://example.org/article9")

	output := serializer.Serialize(tripleStore)

	// Each object should produce a separate element
	refCount := strings.Count(output, "<reg:references")
	if refCount != 2 {
		t.Errorf("Expected 2 reg:references elements, got %d", refCount)
	}
}

func TestRDFXMLSerialize_PrefixedSubject(t *testing.T) {
	serializer := NewRDFXMLSerializer()
	tripleStore := NewTripleStore()
	tripleStore.Add("GDPR:Art17", "rdf:type", "reg:Article")
	tripleStore.Add("GDPR:Art17", "reg:title", "Right to erasure")

	output := serializer.Serialize(tripleStore)

	// Prefixed subjects without known namespace should remain as-is
	if !strings.Contains(output, `rdf:about="GDPR:Art17"`) {
		t.Error("Prefixed subject without matching namespace should use original form")
	}
}

func TestRDFXMLSerialize_TypeFirst(t *testing.T) {
	serializer := NewRDFXMLSerializer()
	tripleStore := NewTripleStore()
	tripleStore.Add("http://example.org/article1", "reg:title", "Test")
	tripleStore.Add("http://example.org/article1", "rdf:type", "reg:Article")
	tripleStore.Add("http://example.org/article1", "reg:number", "1")

	output := serializer.Serialize(tripleStore)

	// rdf:type should appear before other predicates
	typeIndex := strings.Index(output, "<rdf:type")
	titleIndex := strings.Index(output, "<reg:title")
	numberIndex := strings.Index(output, "<reg:number")

	if typeIndex == -1 {
		t.Fatal("Missing rdf:type element")
	}
	if titleIndex == -1 {
		t.Fatal("Missing reg:title element")
	}

	if typeIndex > titleIndex {
		t.Error("rdf:type should appear before reg:title")
	}
	if typeIndex > numberIndex {
		t.Error("rdf:type should appear before reg:number")
	}
}

func TestRDFXMLSerialize_FullURIPredicate(t *testing.T) {
	serializer := NewRDFXMLSerializer()
	tripleStore := NewTripleStore()
	tripleStore.Add("http://example.org/article1", NamespaceReg+"title", "Test Title")

	output := serializer.Serialize(tripleStore)

	// Full URI predicates should be compacted to prefixed form
	if !strings.Contains(output, "<reg:title>Test Title</reg:title>") {
		t.Errorf("Full URI predicate should be compacted to reg:title, got:\n%s", output)
	}
}

// --- Escaping unit tests ---

func TestEscapeXMLText(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"no escaping", "Hello world", "Hello world"},
		{"ampersand", "A & B", "A &amp; B"},
		{"less than", "A < B", "A &lt; B"},
		{"greater than", "A > B", "A &gt; B"},
		{"multiple specials", "a & b < c > d", "a &amp; b &lt; c &gt; d"},
		{"empty", "", ""},
		{"quotes not escaped", `He said "hello"`, `He said "hello"`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := escapeXMLText(testCase.input)
			if result != testCase.expected {
				t.Errorf("escapeXMLText(%q) = %q, want %q", testCase.input, result, testCase.expected)
			}
		})
	}
}

func TestEscapeXMLAttribute(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"no escaping", "http://example.org", "http://example.org"},
		{"ampersand", "A & B", "A &amp; B"},
		{"less than", "A < B", "A &lt; B"},
		{"greater than", "A > B", "A &gt; B"},
		{"quotes", `key="value"`, `key=&quot;value&quot;`},
		{"empty", "", ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := escapeXMLAttribute(testCase.input)
			if result != testCase.expected {
				t.Errorf("escapeXMLAttribute(%q) = %q, want %q", testCase.input, result, testCase.expected)
			}
		})
	}
}

// --- splitPrefixedName tests ---

func TestSplitPrefixedName(t *testing.T) {
	serializer := NewRDFXMLSerializer()

	testCases := []struct {
		name          string
		fullURI       string
		expectedFound bool
		expectedPrefix string
		expectedLocal  string
	}{
		{"reg namespace", NamespaceReg + "Article", true, "reg", "Article"},
		{"rdf namespace", NamespaceRDF + "type", true, "rdf", "type"},
		{"eli namespace", NamespaceELI + "LegalResource", true, "eli", "LegalResource"},
		{"unknown namespace", "http://unknown.org/term", false, "", ""},
		{"empty local name", NamespaceReg, false, "", ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			prefix, localName, found := serializer.splitPrefixedName(testCase.fullURI)
			if found != testCase.expectedFound {
				t.Errorf("splitPrefixedName(%q) found = %v, want %v", testCase.fullURI, found, testCase.expectedFound)
			}
			if found {
				if prefix != testCase.expectedPrefix {
					t.Errorf("prefix = %q, want %q", prefix, testCase.expectedPrefix)
				}
				if localName != testCase.expectedLocal {
					t.Errorf("localName = %q, want %q", localName, testCase.expectedLocal)
				}
			}
		})
	}
}

// --- Integration tests ---

func TestRDFXMLSerialize_GDPRIntegration(t *testing.T) {
	gdprDocument := loadGDPRDocument(t)

	tripleStore := NewTripleStore()
	graphBuilder := NewGraphBuilder(tripleStore, "https://regula.dev/regulations/")

	_, err := graphBuilder.Build(gdprDocument)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	serializer := NewRDFXMLSerializer()
	output := serializer.Serialize(tripleStore)

	// Verify XML declaration
	if !strings.HasPrefix(output, "<?xml version=") {
		t.Error("Missing XML declaration at start")
	}

	// Verify namespace declarations
	if !strings.Contains(output, `xmlns:rdf="`) {
		t.Error("Missing rdf namespace declaration")
	}
	if !strings.Contains(output, `xmlns:reg="`) {
		t.Error("Missing reg namespace declaration")
	}

	// Verify rdf:type elements present
	if !strings.Contains(output, "<rdf:type") {
		t.Error("Expected rdf:type elements in output")
	}

	// Verify Article class
	if !strings.Contains(output, NamespaceReg+"Article") {
		t.Error("Expected reg:Article class URI in output")
	}

	// Verify Chapter class
	if !strings.Contains(output, NamespaceReg+"Chapter") {
		t.Error("Expected reg:Chapter class URI in output")
	}

	// Verify closing element
	if !strings.Contains(output, "</rdf:RDF>") {
		t.Error("Missing closing rdf:RDF element")
	}

	// Verify substantial output
	lines := strings.Split(output, "\n")
	if len(lines) < 100 {
		t.Errorf("Expected substantial output, got %d lines", len(lines))
	}

	// Verify well-formed XML structure (basic check: Description blocks closed)
	openCount := strings.Count(output, "<rdf:Description")
	closeCount := strings.Count(output, "</rdf:Description>")
	if openCount != closeCount {
		t.Errorf("Unbalanced Description elements: %d opened, %d closed", openCount, closeCount)
	}

	t.Logf("RDF/XML output: %d lines, %d Description blocks, %d triples",
		len(lines), openCount, tripleStore.Count())
}

// --- Concurrent access test ---

func TestRDFXMLSerialize_ConcurrentAccess(t *testing.T) {
	tripleStore := NewTripleStore()
	populateTestStore(tripleStore)

	serializer := NewRDFXMLSerializer()

	done := make(chan string, 10)

	for goroutineIndex := 0; goroutineIndex < 10; goroutineIndex++ {
		go func() {
			result := serializer.Serialize(tripleStore)
			done <- result
		}()
	}

	var firstResult string
	for resultIndex := 0; resultIndex < 10; resultIndex++ {
		result := <-done
		if resultIndex == 0 {
			firstResult = result
		} else if result != firstResult {
			t.Error("Concurrent serializations produced different output")
		}
	}
}

// --- expandToFullURI tests ---

func TestExpandToFullURI(t *testing.T) {
	serializer := NewRDFXMLSerializer()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"full URI unchanged", "http://example.org/article1", "http://example.org/article1"},
		{"reg prefix expanded", "reg:Article", NamespaceReg + "Article"},
		{"rdf prefix expanded", "rdf:type", NamespaceRDF + "type"},
		{"unknown prefix unchanged", "GDPR:Art17", "GDPR:Art17"},
		{"no colon unchanged", "plain_value", "plain_value"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := serializer.expandToFullURI(testCase.input)
			if result != testCase.expected {
				t.Errorf("expandToFullURI(%q) = %q, want %q", testCase.input, result, testCase.expected)
			}
		})
	}
}

// --- isURIObject tests ---

func TestIsURIObject(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{"http URI", "http://example.org/article1", true},
		{"https URI", "https://example.org/article1", true},
		{"urn URI", "urn:regula:gdpr:article:17", true},
		{"prefixed name", "reg:Article", true},
		{"literal string", "Right to erasure", false},
		{"number string", "17", false},
		{"empty string", "", false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := isURIObject(testCase.input)
			if result != testCase.expected {
				t.Errorf("isURIObject(%q) = %v, want %v", testCase.input, result, testCase.expected)
			}
		})
	}
}
