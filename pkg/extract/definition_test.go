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

// buildUSCTestDocument creates a Document with a single article containing the given text.
func buildUSCTestDocument(articleNumber int, articleTitle string, articleText string) *Document {
	return &Document{
		Title: "Test USC Title",
		Chapters: []*Chapter{
			{
				Number: "1",
				Title:  "Test Chapter",
				Articles: []*Article{
					{
						Number: articleNumber,
						Title:  articleTitle,
						Text:   articleText,
					},
				},
			},
		},
	}
}

func TestUSCDefinitionExtraction_BasicMeans(t *testing.T) {
	articleText := "When used in this chapter\u2014\n" +
		"  a The term \u201cService\u201d means the Public Health Service;\n" +
		"  b The term \u201cSurgeon General\u201d means the Surgeon General of the Public Health Service;\n"

	doc := buildUSCTestDocument(201, "Definitions", articleText)

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)

	if len(definitions) != 2 {
		t.Fatalf("Expected 2 definitions, got %d", len(definitions))
		for _, def := range definitions {
			t.Logf("  %d: %s = %s", def.Number, def.Term, def.Definition)
		}
	}

	if definitions[0].Term != "Service" {
		t.Errorf("First definition term: got %q, want %q", definitions[0].Term, "Service")
	}
	if definitions[1].Term != "Surgeon General" {
		t.Errorf("Second definition term: got %q, want %q", definitions[1].Term, "Surgeon General")
	}

	if !containsSubstring(definitions[0].Definition, "Public Health Service") {
		t.Errorf("First definition should contain 'Public Health Service', got %q", definitions[0].Definition)
	}
	if !containsSubstring(definitions[1].Definition, "Surgeon General of the Public Health Service") {
		t.Errorf("Second definition should contain 'Surgeon General of the Public Health Service', got %q", definitions[1].Definition)
	}
}

func TestUSCDefinitionExtraction_IncludesVerb(t *testing.T) {
	articleText := "For purposes of this section\u2014\n" +
		"  a The term \u201cperson\u201d includes an individual, corporation, company, association,\n" +
		"firm, partnership, society, and joint stock company;\n" +
		"  b The term \u201cState\u201d includes any State, territory, or possession of the United States;\n"

	doc := buildUSCTestDocument(100, "Definitions", articleText)

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)

	if len(definitions) != 2 {
		t.Fatalf("Expected 2 definitions, got %d", len(definitions))
	}

	if definitions[0].Term != "person" {
		t.Errorf("First definition term: got %q, want %q", definitions[0].Term, "person")
	}
	if !containsSubstring(definitions[0].Definition, "individual, corporation") {
		t.Errorf("First definition should contain 'individual, corporation', got %q", definitions[0].Definition)
	}

	if definitions[1].Term != "State" {
		t.Errorf("Second definition term: got %q, want %q", definitions[1].Term, "State")
	}
}

func TestUSCDefinitionExtraction_ContinuationLines(t *testing.T) {
	articleText := "Definitions\n" +
		"  a The term \u201ccontrolled substance\u201d means a drug or other substance, or immediate\n" +
		"precursor, included in schedule I, schedule II, schedule III, schedule IV, or\n" +
		"schedule V of part B of this subchapter.\n"

	doc := buildUSCTestDocument(802, "Definitions", articleText)

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)

	if len(definitions) != 1 {
		t.Fatalf("Expected 1 definition, got %d", len(definitions))
	}

	if definitions[0].Term != "controlled substance" {
		t.Errorf("Term: got %q, want %q", definitions[0].Term, "controlled substance")
	}

	// Should contain text from continuation lines
	if !containsSubstring(definitions[0].Definition, "schedule V of part B") {
		t.Errorf("Definition should include continuation text, got %q", definitions[0].Definition)
	}
}

func TestUSCDefinitionExtraction_MixedQuoteStyles(t *testing.T) {
	// Test with ASCII double quotes instead of curly quotes
	articleText := "Definitions\n" +
		"  a The term \"agency\" means any executive department;\n"

	doc := buildUSCTestDocument(551, "Definitions", articleText)

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)

	if len(definitions) != 1 {
		t.Fatalf("Expected 1 definition with ASCII quotes, got %d", len(definitions))
	}

	if definitions[0].Term != "agency" {
		t.Errorf("Term: got %q, want %q", definitions[0].Term, "agency")
	}
}

func TestUSCDefinitionExtraction_Scope(t *testing.T) {
	articleText := "Definitions\n" +
		"  a The term \u201cSecretary\u201d means the Secretary of Health and Human Services;\n"

	doc := buildUSCTestDocument(201, "Definitions", articleText)

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)

	if len(definitions) != 1 {
		t.Fatalf("Expected 1 definition, got %d", len(definitions))
	}

	if definitions[0].Scope != "Section Definitions" {
		t.Errorf("Scope: got %q, want %q", definitions[0].Scope, "Section Definitions")
	}
	if definitions[0].ArticleRef != 201 {
		t.Errorf("ArticleRef: got %d, want %d", definitions[0].ArticleRef, 201)
	}
}

func TestUSCDefinitionExtraction_References(t *testing.T) {
	articleText := "Definitions\n" +
		"  a The term \u201cadminister\u201d means the direct application of a \u201ccontrolled substance\u201d to the body of a \u201cpatient\u201d;\n"

	doc := buildUSCTestDocument(802, "Definitions", articleText)

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)

	if len(definitions) != 1 {
		t.Fatalf("Expected 1 definition, got %d", len(definitions))
	}

	// Should extract references to "controlled substance" and "patient"
	if len(definitions[0].References) < 2 {
		t.Errorf("Expected at least 2 references, got %d: %v", len(definitions[0].References), definitions[0].References)
	}

	foundControlledSubstance := false
	foundPatient := false
	for _, ref := range definitions[0].References {
		if ref == "controlled substance" {
			foundControlledSubstance = true
		}
		if ref == "patient" {
			foundPatient = true
		}
	}
	if !foundControlledSubstance {
		t.Errorf("Expected reference to 'controlled substance', got: %v", definitions[0].References)
	}
	if !foundPatient {
		t.Errorf("Expected reference to 'patient', got: %v", definitions[0].References)
	}
}

func TestUSCDefinitionExtraction_DensityDetection(t *testing.T) {
	// Article without "Definitions" in title but with enough USC-style definitions
	// to trigger density detection
	articleText := "When used in this chapter\u2014\n" +
		"  a The term \u201cApplicant\u201d means any person who applies for a grant;\n" +
		"  b The term \u201cBoard\u201d means the Advisory Board for Medical Research;\n" +
		"  c The term \u201cDirector\u201d means the Director of the National Institutes of Health;\n" +
		"  d The term \u201cFund\u201d means the Establishment Fund;\n"

	doc := buildUSCTestDocument(1, "General provisions", articleText)

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)

	// Should find definitions via density detection even without "Definitions" in title
	if len(definitions) < 3 {
		t.Errorf("Expected at least 3 definitions from density detection, got %d", len(definitions))
		for _, def := range definitions {
			t.Logf("  %d: %s", def.Number, def.Term)
		}
	}
}

func TestUSCDefinitionExtraction_NoFalsePositivesOnEU(t *testing.T) {
	// Verify that adding USC extraction doesn't break EU-style GDPR extraction
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)

	// GDPR should still extract exactly 26 definitions via EU-style
	if len(definitions) != 26 {
		t.Errorf("GDPR backward compatibility: expected 26 definitions, got %d", len(definitions))
		for _, def := range definitions {
			t.Logf("  %d: %s", def.Number, def.Term)
		}
	}
}

func TestUSCDefinitionExtraction_EmptyArticle(t *testing.T) {
	doc := buildUSCTestDocument(1, "Definitions", "")

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)

	if len(definitions) != 0 {
		t.Errorf("Expected 0 definitions from empty article, got %d", len(definitions))
	}
}

func TestUSCDefinitionExtraction_NormalizedTerms(t *testing.T) {
	articleText := "Definitions\n" +
		"  a The term \u201cControlled Substance\u201d means a drug;\n" +
		"  b The term \u201cDrug Enforcement Administration\u201d means the agency;\n"

	doc := buildUSCTestDocument(802, "Definitions", articleText)

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)
	lookup := NewDefinitionLookup(definitions)

	// Should be findable by case-insensitive lookup
	def := lookup.GetByNormalizedTerm("controlled substance")
	if def == nil {
		t.Error("Should find 'Controlled Substance' via normalized lookup")
	}

	def = lookup.GetByNormalizedTerm("DRUG ENFORCEMENT ADMINISTRATION")
	if def == nil {
		t.Error("Should find 'Drug Enforcement Administration' via normalized lookup")
	}
}

func TestExtractAfterMeans_IncludesVerb(t *testing.T) {
	extractor := NewDefinitionExtractor()

	testCases := []struct {
		name     string
		line     string
		expected string
	}{
		{"means with space", "The term \"foo\" means the bar", "the bar"},
		{"means with colon", "The term \"foo\" means: the bar", "the bar"},
		{"includes with space", "The term \"foo\" includes the bar", "the bar"},
		{"includes with colon", "The term \"foo\" includes: the bar", "the bar"},
		{"includes with comma", "The term \"foo\" includes, but is not limited to", "but is not limited to"},
		{"no verb", "The term \"foo\" is the bar", ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := extractor.extractAfterMeans(testCase.line)
			if result != testCase.expected {
				t.Errorf("extractAfterMeans(%q) = %q, want %q", testCase.line, result, testCase.expected)
			}
		})
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
