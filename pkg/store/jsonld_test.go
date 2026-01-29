package store

import (
	"encoding/json"
	"strings"
	"testing"
)

// --- Constructor tests ---

func TestNewJSONLDSerializer(t *testing.T) {
	serializer := NewJSONLDSerializer()

	if serializer == nil {
		t.Fatal("NewJSONLDSerializer returned nil")
	}

	if len(serializer.prefixMappings) != 7 {
		t.Errorf("Expected 7 default prefix mappings, got %d", len(serializer.prefixMappings))
	}

	if !serializer.compactForm {
		t.Error("Expected compact form to be enabled by default")
	}
}

func TestNewJSONLDSerializer_WithCustomPrefix(t *testing.T) {
	serializer := NewJSONLDSerializer(
		WithJSONLDPrefix("gdpr", "https://regula.dev/regulations/GDPR#"),
	)

	if len(serializer.prefixMappings) != 8 {
		t.Errorf("Expected 8 prefix mappings (7 default + 1 custom), got %d", len(serializer.prefixMappings))
	}

	if serializer.prefixIndex["gdpr"] != "https://regula.dev/regulations/GDPR#" {
		t.Error("Custom prefix 'gdpr' not found in prefix index")
	}
}

func TestNewJSONLDSerializer_WithExpandedForm(t *testing.T) {
	serializer := NewJSONLDSerializer(WithExpandedForm())

	if serializer.compactForm {
		t.Error("Expected compact form to be disabled with WithExpandedForm()")
	}
}

// --- Context tests ---

func TestBuildContext(t *testing.T) {
	serializer := NewJSONLDSerializer()
	context := serializer.BuildContext()

	// Check namespaces are present
	if context["reg"] != NamespaceReg {
		t.Errorf("Expected reg namespace %s, got %v", NamespaceReg, context["reg"])
	}
	if context["eli"] != NamespaceELI {
		t.Errorf("Expected eli namespace %s, got %v", NamespaceELI, context["eli"])
	}
	if context["dc"] != NamespaceDC {
		t.Errorf("Expected dc namespace %s, got %v", NamespaceDC, context["dc"])
	}

	// Check @type and @id aliases
	if context["type"] != "@type" {
		t.Error("Expected 'type' to map to '@type'")
	}
	if context["id"] != "@id" {
		t.Error("Expected 'id' to map to '@id'")
	}

	// Check property mappings
	titleMapping, ok := context["title"].(map[string]string)
	if !ok || titleMapping["@id"] != "reg:title" {
		t.Error("Expected 'title' to map to reg:title")
	}

	// Check relationship property has @type: @id
	partOfMapping, ok := context["partOf"].(map[string]string)
	if !ok || partOfMapping["@type"] != "@id" {
		t.Error("Expected 'partOf' to have @type: @id for URI references")
	}
}

func TestGetContextOnly(t *testing.T) {
	serializer := NewJSONLDSerializer()
	contextJSON, err := serializer.GetContextOnly()
	if err != nil {
		t.Fatalf("GetContextOnly failed: %v", err)
	}

	// Should be valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(contextJSON, &parsed); err != nil {
		t.Fatalf("Context is not valid JSON: %v", err)
	}

	// Should have namespace prefixes
	if _, exists := parsed["reg"]; !exists {
		t.Error("Context should contain 'reg' namespace")
	}
}

// --- Serialization tests ---

func TestJSONLD_Serialize_EmptyStore(t *testing.T) {
	store := NewTripleStore()
	serializer := NewJSONLDSerializer()

	data, err := serializer.Serialize(store)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	var doc JSONLDDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if doc.Context == nil {
		t.Error("Expected @context in output")
	}

	if len(doc.Graph) != 0 {
		t.Errorf("Expected empty @graph, got %d items", len(doc.Graph))
	}
}

func TestJSONLD_Serialize_SingleTriple(t *testing.T) {
	store := NewTripleStore()
	store.Add("GDPR:Art1", "rdf:type", "reg:Article")

	serializer := NewJSONLDSerializer()
	data, err := serializer.Serialize(store)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	var doc JSONLDDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if len(doc.Graph) != 1 {
		t.Fatalf("Expected 1 node in @graph, got %d", len(doc.Graph))
	}

	node := doc.Graph[0]
	if node["@id"] != "GDPR:Art1" {
		t.Errorf("Expected @id = GDPR:Art1, got %v", node["@id"])
	}

	// rdf:type should be rendered as @type
	if node["@type"] != "reg:Article" {
		t.Errorf("Expected @type = reg:Article, got %v", node["@type"])
	}
}

func TestJSONLD_Serialize_MultiplePredicates(t *testing.T) {
	store := NewTripleStore()
	store.Add("GDPR:Art1", "rdf:type", "reg:Article")
	store.Add("GDPR:Art1", "reg:number", "1")
	store.Add("GDPR:Art1", "reg:title", "Subject-matter and objectives")

	serializer := NewJSONLDSerializer()
	data, err := serializer.Serialize(store)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	var doc JSONLDDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if len(doc.Graph) != 1 {
		t.Fatalf("Expected 1 node in @graph, got %d", len(doc.Graph))
	}

	node := doc.Graph[0]

	// Check all properties present
	if node["@type"] != "reg:Article" {
		t.Error("Missing @type")
	}
	if node["number"] != "1" {
		t.Errorf("Expected number = '1', got %v", node["number"])
	}
	if node["title"] != "Subject-matter and objectives" {
		t.Errorf("Expected title, got %v", node["title"])
	}
}

func TestJSONLD_Serialize_MultipleObjects(t *testing.T) {
	store := NewTripleStore()
	store.Add("GDPR:Art17", "reg:references", "GDPR:Art6")
	store.Add("GDPR:Art17", "reg:references", "GDPR:Art9")

	serializer := NewJSONLDSerializer()
	data, err := serializer.Serialize(store)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	var doc JSONLDDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	node := doc.Graph[0]
	refs, ok := node["references"].([]interface{})
	if !ok {
		t.Fatalf("Expected references to be an array, got %T", node["references"])
	}

	if len(refs) != 2 {
		t.Errorf("Expected 2 references, got %d", len(refs))
	}
}

func TestJSONLD_Serialize_MultipleSubjects(t *testing.T) {
	store := NewTripleStore()
	store.Add("GDPR:Art1", "rdf:type", "reg:Article")
	store.Add("GDPR:Art2", "rdf:type", "reg:Article")

	serializer := NewJSONLDSerializer()
	data, err := serializer.Serialize(store)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	var doc JSONLDDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if len(doc.Graph) != 2 {
		t.Errorf("Expected 2 nodes in @graph, got %d", len(doc.Graph))
	}
}

func TestJSONLD_Serialize_RelationshipAsURIReference(t *testing.T) {
	store := NewTripleStore()
	store.Add("GDPR:Art1", "reg:partOf", "GDPR:ChapterI")

	serializer := NewJSONLDSerializer()
	data, err := serializer.Serialize(store)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	var doc JSONLDDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	node := doc.Graph[0]
	partOf, ok := node["partOf"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected partOf to be an object with @id, got %T: %v", node["partOf"], node["partOf"])
	}

	if partOf["@id"] != "GDPR:ChapterI" {
		t.Errorf("Expected partOf to reference GDPR:ChapterI, got %v", partOf["@id"])
	}
}

func TestJSONLD_Serialize_DeterministicOutput(t *testing.T) {
	store := NewTripleStore()
	store.Add("GDPR:Art1", "rdf:type", "reg:Article")
	store.Add("GDPR:Art1", "reg:number", "1")
	store.Add("GDPR:Art1", "reg:title", "Subject-matter and objectives")
	store.Add("GDPR:Art2", "rdf:type", "reg:Article")
	store.Add("GDPR:Art2", "reg:number", "2")

	serializer := NewJSONLDSerializer()

	firstOutput, _ := serializer.Serialize(store)
	secondOutput, _ := serializer.Serialize(store)

	if string(firstOutput) != string(secondOutput) {
		t.Error("Expected deterministic output, but two serializations differ")
	}
}

// --- Expanded form tests ---

func TestJSONLD_Serialize_ExpandedForm(t *testing.T) {
	store := NewTripleStore()
	store.Add("GDPR:Art1", "rdf:type", "reg:Article")
	store.Add("GDPR:Art1", "reg:title", "Test")

	serializer := NewJSONLDSerializer(WithExpandedForm())
	data, err := serializer.Serialize(store)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Expanded form should be an array, not an object with @context
	var expanded []map[string]interface{}
	if err := json.Unmarshal(data, &expanded); err != nil {
		t.Fatalf("Expanded form should be a JSON array: %v", err)
	}

	if len(expanded) != 1 {
		t.Fatalf("Expected 1 node, got %d", len(expanded))
	}

	node := expanded[0]

	// Should have full URIs, not prefixed names
	rdfTypeKey := NamespaceRDF + "type"
	if _, exists := node[rdfTypeKey]; !exists {
		t.Errorf("Expected full URI key %s in expanded form", rdfTypeKey)
	}
}

// --- URI compaction/expansion tests ---

func TestJSONLD_CompactURI(t *testing.T) {
	serializer := NewJSONLDSerializer()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"reg_namespace", "https://regula.dev/ontology#Article", "reg:Article"},
		{"eli_namespace", "http://data.europa.eu/eli/ontology#LegalResource", "eli:LegalResource"},
		{"already_prefixed", "reg:Article", "reg:Article"},
		{"unknown_namespace", "https://unknown.example.org/thing", "https://unknown.example.org/thing"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := serializer.compactURI(tc.input)
			if result != tc.expected {
				t.Errorf("compactURI(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestJSONLD_ExpandURI(t *testing.T) {
	serializer := NewJSONLDSerializer()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"reg_prefix", "reg:Article", "https://regula.dev/ontology#Article"},
		{"eli_prefix", "eli:LegalResource", "http://data.europa.eu/eli/ontology#LegalResource"},
		{"full_uri", "https://example.org/thing", "https://example.org/thing"},
		{"unknown_prefix", "unknown:term", "unknown:term"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := serializer.expandURI(tc.input)
			if result != tc.expected {
				t.Errorf("expandURI(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// --- Predicate to key mapping tests ---

func TestJSONLD_PredicateToKey(t *testing.T) {
	serializer := NewJSONLDSerializer()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"rdf_type_prefixed", "rdf:type", "@type"},
		{"rdf_type_full", NamespaceRDF + "type", "@type"},
		{"reg_title", "reg:title", "title"},
		{"reg_number", "reg:number", "number"},
		{"eli_title", "eli:title", "eli:title"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := serializer.predicateToKey(tc.input)
			if result != tc.expected {
				t.Errorf("predicateToKey(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// --- Relationship predicate detection tests ---

func TestJSONLD_IsRelationshipPredicate(t *testing.T) {
	serializer := NewJSONLDSerializer()

	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{"reg_partOf", "reg:partOf", true},
		{"reg_references", "reg:references", true},
		{"reg_contains", "reg:contains", true},
		{"eli_cites", "eli:cites", true},
		{"reg_title", "reg:title", false},
		{"reg_number", "reg:number", false},
		{"reg_text", "reg:text", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := serializer.isRelationshipPredicate(tc.input)
			if result != tc.expected {
				t.Errorf("isRelationshipPredicate(%q) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

// --- JSON validity tests ---

func TestJSONLD_Serialize_ValidJSON(t *testing.T) {
	store := NewTripleStore()
	store.Add("GDPR:Art1", "rdf:type", "reg:Article")
	store.Add("GDPR:Art1", "reg:title", `Title with "quotes" and special chars: <>&`)
	store.Add("GDPR:Art1", "reg:text", "Line1\nLine2\tTabbed")

	serializer := NewJSONLDSerializer()
	data, err := serializer.Serialize(store)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Should be valid JSON
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nOutput was:\n%s", err, string(data))
	}
}

func TestJSONLD_Serialize_JSONLDStructure(t *testing.T) {
	store := NewTripleStore()
	store.Add("GDPR:Art1", "rdf:type", "reg:Article")

	serializer := NewJSONLDSerializer()
	data, err := serializer.Serialize(store)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Parse as raw JSON to check structure
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Failed to parse as JSON object: %v", err)
	}

	// Must have @context
	if _, exists := raw["@context"]; !exists {
		t.Error("Missing @context in compact JSON-LD")
	}

	// Must have @graph
	if _, exists := raw["@graph"]; !exists {
		t.Error("Missing @graph in JSON-LD document")
	}
}

// --- GDPR Integration test ---

func TestJSONLD_Serialize_GDPRIntegration(t *testing.T) {
	gdprDocument := loadGDPRDocument(t)

	tripleStore := NewTripleStore()
	graphBuilder := NewGraphBuilder(tripleStore, "https://regula.dev/regulations/")

	buildStats, err := graphBuilder.Build(gdprDocument)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	serializer := NewJSONLDSerializer()
	data, err := serializer.Serialize(tripleStore)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Should be valid JSON
	var doc JSONLDDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Output is not valid JSON-LD: %v", err)
	}

	// Should have @context
	if doc.Context == nil {
		t.Error("Missing @context in output")
	}

	// Should have substantial content
	if len(doc.Graph) < 100 {
		t.Errorf("Expected substantial output, got %d nodes", len(doc.Graph))
	}

	// Verify some content
	hasArticle := false
	hasChapter := false
	for _, node := range doc.Graph {
		nodeType, _ := node["@type"].(string)
		if nodeType == "reg:Article" {
			hasArticle = true
		}
		if nodeType == "reg:Chapter" {
			hasChapter = true
		}
	}

	if !hasArticle {
		t.Error("Expected reg:Article nodes in output")
	}
	if !hasChapter {
		t.Error("Expected reg:Chapter nodes in output")
	}

	t.Logf("GDPR JSON-LD output: %d nodes, %d bytes, %d triples",
		len(doc.Graph), len(data), buildStats.TotalTriples)
}

// --- SerializeToString test ---

func TestJSONLD_SerializeToString(t *testing.T) {
	store := NewTripleStore()
	store.Add("GDPR:Art1", "rdf:type", "reg:Article")

	serializer := NewJSONLDSerializer()
	output, err := serializer.SerializeToString(store)
	if err != nil {
		t.Fatalf("SerializeToString failed: %v", err)
	}

	if !strings.Contains(output, "@context") {
		t.Error("String output should contain @context")
	}

	if !strings.Contains(output, "@graph") {
		t.Error("String output should contain @graph")
	}
}

// --- DefaultJSONLDContext test ---

func TestDefaultJSONLDContext(t *testing.T) {
	context := DefaultJSONLDContext()

	if context == nil {
		t.Fatal("DefaultJSONLDContext returned nil")
	}

	if context["reg"] != NamespaceReg {
		t.Error("Default context should include reg namespace")
	}

	if context["eli"] != NamespaceELI {
		t.Error("Default context should include eli namespace")
	}
}

// --- Concurrent access test ---

func TestJSONLD_Serialize_ConcurrentAccess(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	serializer := NewJSONLDSerializer()

	done := make(chan []byte, 10)

	for i := 0; i < 10; i++ {
		go func() {
			result, _ := serializer.Serialize(store)
			done <- result
		}()
	}

	var firstResult []byte
	for i := 0; i < 10; i++ {
		result := <-done
		if i == 0 {
			firstResult = result
		} else if string(result) != string(firstResult) {
			t.Error("Concurrent serializations produced different output")
		}
	}
}

// --- Edge case tests ---

func TestJSONLD_Serialize_EmptySubject(t *testing.T) {
	store := NewTripleStore()
	// Store should reject empty components
	err := store.Add("", "rdf:type", "reg:Article")
	if err == nil {
		t.Error("Expected error when adding triple with empty subject")
	}
}

func TestJSONLD_Serialize_SpecialCharactersInLiterals(t *testing.T) {
	store := NewTripleStore()
	store.Add("test:Subject", "reg:text", "Text with \"quotes\" and \\ backslashes")
	store.Add("test:Subject", "reg:title", "Unicode: 日本語")

	serializer := NewJSONLDSerializer()
	data, err := serializer.Serialize(store)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Should still be valid JSON
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}
}
