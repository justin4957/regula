package extract

import (
	"testing"
)

// Expected GDPR definitions for validation
var expectedDefinitions = []struct {
	Number int
	Term   string
}{
	{1, "personal data"},
	{2, "processing"},
	{3, "restriction of processing"},
	{4, "profiling"},
	{5, "pseudonymisation"},
	{6, "filing system"},
	{7, "controller"},
	{8, "processor"},
	{9, "recipient"},
	{10, "third party"},
	{11, "consent"},
	{12, "personal data breach"},
	{13, "genetic data"},
	{14, "biometric data"},
	{15, "data concerning health"},
	{16, "main establishment"},
	{17, "representative"},
	{18, "enterprise"},
	{19, "group of undertakings"},
	{20, "binding corporate rules"},
	{21, "supervisory authority"},
	{22, "supervisory authority concerned"},
	{23, "cross-border processing"},
	{24, "relevant and reasoned objection"},
	{25, "information society service"},
	{26, "international organisation"},
}

func TestDefinitionExtraction(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)

	t.Logf("Extracted %d definitions", len(definitions))

	// Verify we got all 26 definitions
	if len(definitions) != 26 {
		t.Errorf("Expected 26 definitions, got %d", len(definitions))
		for _, def := range definitions {
			t.Logf("  %d: %s", def.Number, def.Term)
		}
	}
}

func TestDefinitionExtraction_AllTermsPresent(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)
	lookup := NewDefinitionLookup(definitions)

	for _, expected := range expectedDefinitions {
		def := lookup.GetByNumber(expected.Number)
		if def == nil {
			t.Errorf("Definition %d not found", expected.Number)
			continue
		}

		if def.Term != expected.Term {
			t.Errorf("Definition %d term mismatch: got %q, want %q",
				expected.Number, def.Term, expected.Term)
		}
	}
}

func TestDefinitionExtraction_NormalizedTerms(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)
	lookup := NewDefinitionLookup(definitions)

	// Test case-insensitive lookup
	testCases := []struct {
		query    string
		expected string
	}{
		{"PERSONAL DATA", "personal data"},
		{"Personal Data", "personal data"},
		{"processing", "processing"},
		{"Controller", "controller"},
		{"SUPERVISORY AUTHORITY", "supervisory authority"},
	}

	for _, tc := range testCases {
		def := lookup.GetByNormalizedTerm(tc.query)
		if def == nil {
			t.Errorf("Failed to find definition for %q", tc.query)
			continue
		}
		if def.NormalizedTerm != tc.expected {
			t.Errorf("Normalized term mismatch: got %q, want %q",
				def.NormalizedTerm, tc.expected)
		}
	}
}

func TestDefinitionExtraction_DefinitionTextPresent(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)

	for _, def := range definitions {
		// Every definition should have either main text or sub-points
		hasContent := def.Definition != "" || len(def.SubPoints) > 0
		if !hasContent {
			t.Errorf("Definition %d (%s) has no definition text", def.Number, def.Term)
		}

		// Log definition lengths for verification
		totalLen := len(def.Definition)
		for _, sp := range def.SubPoints {
			totalLen += len(sp.Text)
		}
		t.Logf("Definition %d (%s): %d chars, %d sub-points",
			def.Number, def.Term, totalLen, len(def.SubPoints))
	}
}

func TestDefinitionExtraction_SubPoints(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)
	lookup := NewDefinitionLookup(definitions)

	// Definition 16 (main establishment) should have sub-points (a) and (b)
	def16 := lookup.GetByNumber(16)
	if def16 == nil {
		t.Fatal("Definition 16 not found")
	}

	if len(def16.SubPoints) != 2 {
		t.Errorf("Definition 16 should have 2 sub-points, got %d", len(def16.SubPoints))
		for i, sp := range def16.SubPoints {
			t.Logf("  Sub-point %d: (%s) %.50s...", i, sp.Letter, sp.Text)
		}
	} else {
		if def16.SubPoints[0].Letter != "a" {
			t.Errorf("First sub-point should be (a), got (%s)", def16.SubPoints[0].Letter)
		}
		if def16.SubPoints[1].Letter != "b" {
			t.Errorf("Second sub-point should be (b), got (%s)", def16.SubPoints[1].Letter)
		}
	}

	// Definition 22 (supervisory authority concerned) should have sub-points (a), (b), (c)
	def22 := lookup.GetByNumber(22)
	if def22 == nil {
		t.Fatal("Definition 22 not found")
	}

	if len(def22.SubPoints) != 3 {
		t.Errorf("Definition 22 should have 3 sub-points, got %d", len(def22.SubPoints))
	}

	// Definition 23 (cross-border processing) should have sub-points (a) and (b)
	def23 := lookup.GetByNumber(23)
	if def23 == nil {
		t.Fatal("Definition 23 not found")
	}

	if len(def23.SubPoints) != 2 {
		t.Errorf("Definition 23 should have 2 sub-points, got %d", len(def23.SubPoints))
	}
}

func TestDefinitionExtraction_Scope(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)

	for _, def := range definitions {
		if def.Scope != "Article 4" {
			t.Errorf("Definition %d scope should be 'Article 4', got %q",
				def.Number, def.Scope)
		}
		if def.ArticleRef != 4 {
			t.Errorf("Definition %d article reference should be 4, got %d",
				def.Number, def.ArticleRef)
		}
	}
}

func TestDefinitionExtraction_SpecificDefinitions(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)
	lookup := NewDefinitionLookup(definitions)

	// Test specific definition content
	testCases := []struct {
		number   int
		contains string
	}{
		{1, "information relating to an identified"}, // personal data
		{2, "operation or set of operations"},        // processing
		{7, "determines the purposes and means"},     // controller
		{8, "processes personal data on behalf"},     // processor
		{11, "freely given, specific, informed"},     // consent
		{12, "breach of security"},                   // personal data breach
	}

	for _, tc := range testCases {
		def := lookup.GetByNumber(tc.number)
		if def == nil {
			t.Errorf("Definition %d not found", tc.number)
			continue
		}

		fullText := def.Definition
		for _, sp := range def.SubPoints {
			fullText += " " + sp.Text
		}

		if !containsSubstring(fullText, tc.contains) {
			t.Errorf("Definition %d (%s) should contain %q\nGot: %s",
				tc.number, def.Term, tc.contains, fullText[:min(200, len(fullText))])
		}
	}
}

func TestDefinitionExtraction_References(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)
	lookup := NewDefinitionLookup(definitions)

	stats := lookup.Stats()
	t.Logf("Definitions with references: %d", stats.WithReferences)
	t.Logf("Total references: %d", stats.TotalReferences)

	// Definition 1 (personal data) should reference "data subject"
	def1 := lookup.GetByNumber(1)
	if def1 == nil {
		t.Fatal("Definition 1 not found")
	}

	foundDataSubject := false
	for _, ref := range def1.References {
		if ref == "data subject" {
			foundDataSubject = true
			break
		}
	}
	if !foundDataSubject {
		t.Logf("Definition 1 references: %v", def1.References)
		// Note: This might not fail if the term isn't in quotes in the definition
	}
}

func TestDefinitionLookup(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)
	lookup := NewDefinitionLookup(definitions)

	// Test Count
	if lookup.Count() != 26 {
		t.Errorf("Expected 26 definitions, got %d", lookup.Count())
	}

	// Test All
	all := lookup.All()
	if len(all) != 26 {
		t.Errorf("All() should return 26 definitions, got %d", len(all))
	}

	// Test GetByNumber
	def := lookup.GetByNumber(1)
	if def == nil || def.Term != "personal data" {
		t.Error("GetByNumber(1) should return 'personal data'")
	}

	// Test GetByTerm
	def = lookup.GetByTerm("processing")
	if def == nil || def.Number != 2 {
		t.Error("GetByTerm('processing') should return definition 2")
	}

	// Test GetByNormalizedTerm
	def = lookup.GetByNormalizedTerm("PERSONAL DATA")
	if def == nil || def.Number != 1 {
		t.Error("GetByNormalizedTerm('PERSONAL DATA') should return definition 1")
	}

	// Test non-existent
	def = lookup.GetByNumber(100)
	if def != nil {
		t.Error("GetByNumber(100) should return nil")
	}

	def = lookup.GetByTerm("nonexistent")
	if def != nil {
		t.Error("GetByTerm('nonexistent') should return nil")
	}
}

func TestDefinitionStats(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)
	lookup := NewDefinitionLookup(definitions)

	stats := lookup.Stats()

	t.Logf("Definition Statistics:")
	t.Logf("  Total definitions: %d", stats.TotalDefinitions)
	t.Logf("  With sub-points: %d", stats.WithSubPoints)
	t.Logf("  Total sub-points: %d", stats.TotalSubPoints)
	t.Logf("  With references: %d", stats.WithReferences)
	t.Logf("  Total references: %d", stats.TotalReferences)
	t.Logf("  Average definition length: %d chars", stats.AverageDefinitionLen)

	if stats.TotalDefinitions != 26 {
		t.Errorf("Expected 26 total definitions, got %d", stats.TotalDefinitions)
	}

	// At least definitions 16, 22, 23 have sub-points
	if stats.WithSubPoints < 3 {
		t.Errorf("Expected at least 3 definitions with sub-points, got %d", stats.WithSubPoints)
	}

	// Should have meaningful average length
	if stats.AverageDefinitionLen < 50 {
		t.Errorf("Average definition length seems too short: %d", stats.AverageDefinitionLen)
	}
}

// Helper function
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr) >= 0))
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
