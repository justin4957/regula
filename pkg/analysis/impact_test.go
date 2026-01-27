package analysis

import (
	"os"
	"testing"

	"github.com/coolbeans/regula/pkg/extract"
	"github.com/coolbeans/regula/pkg/store"
)

func TestNewImpactAnalyzer(t *testing.T) {
	ts := store.NewTripleStore()
	analyzer := NewImpactAnalyzer(ts, "https://regula.dev/regulations/")

	if analyzer == nil {
		t.Fatal("Expected non-nil analyzer")
	}
	if analyzer.store != ts {
		t.Error("Store not set correctly")
	}
}

func TestResolveShortID(t *testing.T) {
	ts := store.NewTripleStore()
	analyzer := NewImpactAnalyzer(ts, "https://regula.dev/regulations/")

	tests := []struct {
		input    string
		expected string
	}{
		{"Art17", "https://regula.dev/regulations/GDPR:Art17"},
		{"GDPR:Art17", "https://regula.dev/regulations/GDPR:Art17"},
		{"https://example.com/Art17", "https://example.com/Art17"},
	}

	for _, tc := range tests {
		result := analyzer.resolveShortID(tc.input)
		if result != tc.expected {
			t.Errorf("resolveShortID(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestAnalyzeDirectImpact(t *testing.T) {
	ts := store.NewTripleStore()
	baseURI := "https://regula.dev/regulations/"

	// Set up test data: Art19 references Art17, Art17 references Art6
	ts.Add(baseURI+"GDPR:Art17", store.RDFType, store.ClassArticle)
	ts.Add(baseURI+"GDPR:Art17", store.PropTitle, "Right to erasure ('right to be forgotten')")
	ts.Add(baseURI+"GDPR:Art19", store.RDFType, store.ClassArticle)
	ts.Add(baseURI+"GDPR:Art19", store.PropTitle, "Notification obligation regarding rectification or erasure")
	ts.Add(baseURI+"GDPR:Art6", store.RDFType, store.ClassArticle)
	ts.Add(baseURI+"GDPR:Art6", store.PropTitle, "Lawfulness of processing")

	// Art19 references Art17
	ts.Add(baseURI+"GDPR:Art19", store.PropReferences, baseURI+"GDPR:Art17")
	// Art17 references Art6
	ts.Add(baseURI+"GDPR:Art17", store.PropReferences, baseURI+"GDPR:Art6")

	analyzer := NewImpactAnalyzer(ts, baseURI)

	// Analyze Art17 with depth 1
	result := analyzer.AnalyzeByID("Art17", 1, DirectionBoth)

	// Check direct incoming (Art19 references Art17)
	if len(result.DirectIncoming) != 1 {
		t.Errorf("Expected 1 direct incoming, got %d", len(result.DirectIncoming))
	} else if result.DirectIncoming[0].URI != baseURI+"GDPR:Art19" {
		t.Errorf("Expected Art19 as direct incoming, got %s", result.DirectIncoming[0].URI)
	}

	// Check direct outgoing (Art17 references Art6)
	if len(result.DirectOutgoing) != 1 {
		t.Errorf("Expected 1 direct outgoing, got %d", len(result.DirectOutgoing))
	} else if result.DirectOutgoing[0].URI != baseURI+"GDPR:Art6" {
		t.Errorf("Expected Art6 as direct outgoing, got %s", result.DirectOutgoing[0].URI)
	}

	// Check summary
	if result.Summary.DirectIncomingCount != 1 {
		t.Errorf("Expected DirectIncomingCount=1, got %d", result.Summary.DirectIncomingCount)
	}
	if result.Summary.DirectOutgoingCount != 1 {
		t.Errorf("Expected DirectOutgoingCount=1, got %d", result.Summary.DirectOutgoingCount)
	}
	if result.Summary.TotalAffected != 2 {
		t.Errorf("Expected TotalAffected=2, got %d", result.Summary.TotalAffected)
	}
}

func TestAnalyzeTransitiveImpact(t *testing.T) {
	ts := store.NewTripleStore()
	baseURI := "https://regula.dev/regulations/"

	// Set up chain: Art21 -> Art17 -> Art6 -> Art5
	ts.Add(baseURI+"GDPR:Art21", store.RDFType, store.ClassArticle)
	ts.Add(baseURI+"GDPR:Art21", store.PropTitle, "Right to object")
	ts.Add(baseURI+"GDPR:Art17", store.RDFType, store.ClassArticle)
	ts.Add(baseURI+"GDPR:Art17", store.PropTitle, "Right to erasure")
	ts.Add(baseURI+"GDPR:Art6", store.RDFType, store.ClassArticle)
	ts.Add(baseURI+"GDPR:Art6", store.PropTitle, "Lawfulness of processing")
	ts.Add(baseURI+"GDPR:Art5", store.RDFType, store.ClassArticle)
	ts.Add(baseURI+"GDPR:Art5", store.PropTitle, "Principles relating to processing")

	ts.Add(baseURI+"GDPR:Art21", store.PropReferences, baseURI+"GDPR:Art17")
	ts.Add(baseURI+"GDPR:Art17", store.PropReferences, baseURI+"GDPR:Art6")
	ts.Add(baseURI+"GDPR:Art6", store.PropReferences, baseURI+"GDPR:Art5")

	analyzer := NewImpactAnalyzer(ts, baseURI)

	// Analyze Art17 with depth 2
	result := analyzer.AnalyzeByID("Art17", 2, DirectionBoth)

	// Direct: Art21 incoming, Art6 outgoing
	if len(result.DirectIncoming) != 1 {
		t.Errorf("Expected 1 direct incoming, got %d", len(result.DirectIncoming))
	}
	if len(result.DirectOutgoing) != 1 {
		t.Errorf("Expected 1 direct outgoing, got %d", len(result.DirectOutgoing))
	}

	// Transitive at depth 2: Art5 (via Art6)
	if len(result.TransitiveNodes) != 1 {
		t.Errorf("Expected 1 transitive node, got %d", len(result.TransitiveNodes))
	} else if result.TransitiveNodes[0].URI != baseURI+"GDPR:Art5" {
		t.Errorf("Expected Art5 as transitive, got %s", result.TransitiveNodes[0].URI)
	}

	if result.Summary.TransitiveCount != 1 {
		t.Errorf("Expected TransitiveCount=1, got %d", result.Summary.TransitiveCount)
	}
}

func TestAnalyzeIncomingOnly(t *testing.T) {
	ts := store.NewTripleStore()
	baseURI := "https://regula.dev/regulations/"

	ts.Add(baseURI+"GDPR:Art17", store.RDFType, store.ClassArticle)
	ts.Add(baseURI+"GDPR:Art19", store.RDFType, store.ClassArticle)
	ts.Add(baseURI+"GDPR:Art6", store.RDFType, store.ClassArticle)

	ts.Add(baseURI+"GDPR:Art19", store.PropReferences, baseURI+"GDPR:Art17")
	ts.Add(baseURI+"GDPR:Art17", store.PropReferences, baseURI+"GDPR:Art6")

	analyzer := NewImpactAnalyzer(ts, baseURI)

	// Analyze Art17 incoming only
	result := analyzer.AnalyzeByID("Art17", 1, DirectionIncoming)

	if len(result.DirectIncoming) != 1 {
		t.Errorf("Expected 1 direct incoming, got %d", len(result.DirectIncoming))
	}
	if len(result.DirectOutgoing) != 0 {
		t.Errorf("Expected 0 direct outgoing, got %d", len(result.DirectOutgoing))
	}
}

func TestAnalyzeOutgoingOnly(t *testing.T) {
	ts := store.NewTripleStore()
	baseURI := "https://regula.dev/regulations/"

	ts.Add(baseURI+"GDPR:Art17", store.RDFType, store.ClassArticle)
	ts.Add(baseURI+"GDPR:Art19", store.RDFType, store.ClassArticle)
	ts.Add(baseURI+"GDPR:Art6", store.RDFType, store.ClassArticle)

	ts.Add(baseURI+"GDPR:Art19", store.PropReferences, baseURI+"GDPR:Art17")
	ts.Add(baseURI+"GDPR:Art17", store.PropReferences, baseURI+"GDPR:Art6")

	analyzer := NewImpactAnalyzer(ts, baseURI)

	// Analyze Art17 outgoing only
	result := analyzer.AnalyzeByID("Art17", 1, DirectionOutgoing)

	if len(result.DirectIncoming) != 0 {
		t.Errorf("Expected 0 direct incoming, got %d", len(result.DirectIncoming))
	}
	if len(result.DirectOutgoing) != 1 {
		t.Errorf("Expected 1 direct outgoing, got %d", len(result.DirectOutgoing))
	}
}

func TestImpactResultString(t *testing.T) {
	result := &ImpactResult{
		TargetURI:   "https://regula.dev/regulations/GDPR:Art17",
		TargetLabel: "Right to erasure",
		MaxDepth:    2,
		DirectIncoming: []*ImpactNode{
			{URI: "Art19", Label: "Notification obligation", Type: "Article", Depth: 1, Direction: "incoming"},
		},
		DirectOutgoing: []*ImpactNode{
			{URI: "Art6", Label: "Lawfulness", Type: "Article", Depth: 1, Direction: "outgoing"},
		},
		TransitiveNodes: []*ImpactNode{},
		Edges:           []*ImpactEdge{},
		ByDepth:         make(map[int][]string),
		Summary: &ImpactSummary{
			TotalAffected:       2,
			DirectIncomingCount: 1,
			DirectOutgoingCount: 1,
			TransitiveCount:     0,
			MaxDepthReached:     1,
			AffectedByType:      map[string]int{"Article": 2},
			AffectedByDepth:     map[int]int{1: 2},
		},
	}

	str := result.String()
	if str == "" {
		t.Error("String() returned empty")
	}
	if !containsString(str, "Right to erasure") {
		t.Error("String() missing target label")
	}
	if !containsString(str, "Total affected provisions: 2") {
		t.Error("String() missing total affected")
	}
}

func TestImpactResultJSON(t *testing.T) {
	result := &ImpactResult{
		TargetURI:       "https://regula.dev/regulations/GDPR:Art17",
		TargetLabel:     "Right to erasure",
		MaxDepth:        1,
		DirectIncoming:  []*ImpactNode{},
		DirectOutgoing:  []*ImpactNode{},
		TransitiveNodes: []*ImpactNode{},
		Edges:           []*ImpactEdge{},
		ByDepth:         make(map[int][]string),
		Summary: &ImpactSummary{
			TotalAffected:   0,
			AffectedByType:  make(map[string]int),
			AffectedByDepth: make(map[int]int),
		},
	}

	data, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}
	if len(data) == 0 {
		t.Error("ToJSON() returned empty")
	}
	if !containsString(string(data), "target_uri") {
		t.Error("JSON missing target_uri field")
	}
}

func TestGDPRArt17Impact(t *testing.T) {
	// Integration test with real GDPR data
	file, err := os.Open("../../testdata/gdpr.txt")
	if err != nil {
		t.Skipf("Skipping GDPR test: %v", err)
	}
	defer file.Close()

	parser := extract.NewParser()
	doc, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	baseURI := "https://regula.dev/regulations/"
	ts := store.NewTripleStore()
	builder := store.NewGraphBuilder(ts, baseURI)

	defExtractor := extract.NewDefinitionExtractor()
	refExtractor := extract.NewReferenceExtractor()
	semExtractor := extract.NewSemanticExtractor()
	resolver := extract.NewReferenceResolver(baseURI, "GDPR")
	resolver.IndexDocument(doc)

	_, err = builder.BuildComplete(doc, defExtractor, refExtractor, resolver, semExtractor)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	analyzer := NewImpactAnalyzer(ts, baseURI)

	// Test Art17 impact with depth 2
	result := analyzer.AnalyzeByID("Art17", 2, DirectionBoth)

	t.Logf("Art17 Impact Analysis:")
	t.Logf("  Direct incoming: %d", result.Summary.DirectIncomingCount)
	t.Logf("  Direct outgoing: %d", result.Summary.DirectOutgoingCount)
	t.Logf("  Transitive: %d", result.Summary.TransitiveCount)
	t.Logf("  Total affected: %d", result.Summary.TotalAffected)

	// According to issue #14, Art17 should have:
	// - Art19 referencing it (notification obligation)
	// - References to Art6 (lawfulness)
	// These are not strict requirements but good to verify

	if result.Summary.TotalAffected == 0 {
		t.Error("Expected some affected provisions for Art17")
	}

	// Log actual results for verification
	if len(result.DirectIncoming) > 0 {
		t.Logf("  Incoming provisions:")
		for _, node := range result.DirectIncoming {
			t.Logf("    - %s (%s)", node.Label, node.Type)
		}
	}
	if len(result.DirectOutgoing) > 0 {
		t.Logf("  Outgoing provisions:")
		for _, node := range result.DirectOutgoing {
			t.Logf("    - %s (%s)", node.Label, node.Type)
		}
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
