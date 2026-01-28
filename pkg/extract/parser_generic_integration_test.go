package extract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Generic test data helpers

func loadGenericNumberedText(t *testing.T) *os.File {
	t.Helper()
	testdataPath := filepath.Join("..", "..", "testdata", "generic-numbered.txt")
	f, err := os.Open(testdataPath)
	if err != nil {
		t.Fatalf("Failed to load generic-numbered.txt: %v", err)
	}
	return f
}

func loadGenericOutlineText(t *testing.T) *os.File {
	t.Helper()
	testdataPath := filepath.Join("..", "..", "testdata", "generic-outline.txt")
	f, err := os.Open(testdataPath)
	if err != nil {
		t.Fatalf("Failed to load generic-outline.txt: %v", err)
	}
	return f
}

func loadGenericMixedText(t *testing.T) *os.File {
	t.Helper()
	testdataPath := filepath.Join("..", "..", "testdata", "generic-mixed.txt")
	f, err := os.Open(testdataPath)
	if err != nil {
		t.Fatalf("Failed to load generic-mixed.txt: %v", err)
	}
	return f
}

func loadGenericIndentedText(t *testing.T) *os.File {
	t.Helper()
	testdataPath := filepath.Join("..", "..", "testdata", "generic-indented.txt")
	f, err := os.Open(testdataPath)
	if err != nil {
		t.Fatalf("Failed to load generic-indented.txt: %v", err)
	}
	return f
}

// Integration tests: generic parsing through Parse()

func TestGenericParserIntegrationNumberedHierarchy(t *testing.T) {
	f := loadGenericNumberedText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should detect generic format (no EU/US/UK indicators)
	if parser.format != FormatGeneric {
		t.Errorf("Expected FormatGeneric, got %s", parser.format)
	}

	stats := doc.Statistics()
	t.Logf("Generic Numbered Parse Results:")
	t.Logf("  Title:       %s", doc.Title)
	t.Logf("  Chapters:    %d", stats.Chapters)
	t.Logf("  Sections:    %d", stats.Sections)
	t.Logf("  Articles:    %d", stats.Articles)
	t.Logf("  Definitions: %d", stats.Definitions)

	// Should have detected sections as chapters or articles
	if stats.Chapters < 1 {
		t.Errorf("Expected at least 1 chapter, got %d", stats.Chapters)
	}

	// Should have detected definitions
	if stats.Definitions < 3 {
		t.Errorf("Expected at least 3 definitions, got %d", stats.Definitions)
	}

	// Title should be detected
	if doc.Title == "" {
		t.Error("Expected non-empty title")
	}
}

func TestGenericParserIntegrationOutlineFormat(t *testing.T) {
	f := loadGenericOutlineText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parser.format != FormatGeneric {
		t.Errorf("Expected FormatGeneric, got %s", parser.format)
	}

	stats := doc.Statistics()
	t.Logf("Generic Outline Parse Results:")
	t.Logf("  Title:       %s", doc.Title)
	t.Logf("  Chapters:    %d", stats.Chapters)
	t.Logf("  Sections:    %d", stats.Sections)
	t.Logf("  Articles:    %d", stats.Articles)
	t.Logf("  Definitions: %d", stats.Definitions)

	// Should have some structure
	totalStructure := stats.Chapters + stats.Articles
	if totalStructure < 3 {
		t.Errorf("Expected at least 3 structural elements, got %d", totalStructure)
	}

	// Title should be detected
	if !strings.Contains(doc.Title, "CORPORATE COMPLIANCE POLICY") {
		t.Errorf("Expected title containing 'CORPORATE COMPLIANCE POLICY', got %q", doc.Title)
	}
}

func TestGenericParserIntegrationMixedNumbering(t *testing.T) {
	f := loadGenericMixedText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parser.format != FormatGeneric {
		t.Errorf("Expected FormatGeneric, got %s", parser.format)
	}

	stats := doc.Statistics()
	t.Logf("Generic Mixed Parse Results:")
	t.Logf("  Title:       %s", doc.Title)
	t.Logf("  Chapters:    %d", stats.Chapters)
	t.Logf("  Sections:    %d", stats.Sections)
	t.Logf("  Articles:    %d", stats.Articles)
	t.Logf("  Definitions: %d", stats.Definitions)

	// Should have detected sections
	if stats.Chapters < 1 {
		t.Errorf("Expected at least 1 chapter, got %d", stats.Chapters)
	}

	// Should have detected definitions
	if stats.Definitions < 2 {
		t.Errorf("Expected at least 2 definitions, got %d", stats.Definitions)
	}
}

func TestGenericParserIntegrationIndentBased(t *testing.T) {
	f := loadGenericIndentedText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parser.format != FormatGeneric {
		t.Errorf("Expected FormatGeneric, got %s", parser.format)
	}

	stats := doc.Statistics()
	t.Logf("Generic Indented Parse Results:")
	t.Logf("  Title:       %s", doc.Title)
	t.Logf("  Chapters:    %d", stats.Chapters)
	t.Logf("  Sections:    %d", stats.Sections)
	t.Logf("  Articles:    %d", stats.Articles)

	// Should have detected at least the major headers
	if stats.Chapters < 1 {
		t.Errorf("Expected at least 1 chapter, got %d", stats.Chapters)
	}
}

// Integration tests with pattern registry

func TestGenericParserIntegrationWithRegistry(t *testing.T) {
	registry := loadPatternRegistry(t)

	testCases := []struct {
		name     string
		loadFunc func(t *testing.T) *os.File
	}{
		{"numbered", loadGenericNumberedText},
		{"outline", loadGenericOutlineText},
		{"mixed", loadGenericMixedText},
		{"indented", loadGenericIndentedText},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := tc.loadFunc(t)
			defer f.Close()

			parser := NewParserWithRegistry(registry)
			doc, err := parser.Parse(f)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// With registry active, these should still detect as generic
			if parser.format != FormatGeneric {
				t.Errorf("Expected FormatGeneric, got %s", parser.format)
			}

			stats := doc.Statistics()
			t.Logf("Registry %s: chapters=%d, sections=%d, articles=%d, definitions=%d",
				tc.name, stats.Chapters, stats.Sections, stats.Articles, stats.Definitions)

			// Should produce some structure
			if stats.Chapters == 0 && stats.Articles == 0 {
				t.Error("Expected at least some structure from generic parsing")
			}
		})
	}
}

// Regression tests: ensure existing formats are not affected

func TestGenericFormatNoRegressionGDPR(t *testing.T) {
	expected := loadExpectedGDPR(t)

	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// GDPR should NOT be detected as generic
	if parser.format == FormatGeneric {
		t.Error("GDPR should not be detected as FormatGeneric")
	}

	stats := doc.Statistics()
	if stats.Articles != expected.Statistics.Articles {
		t.Errorf("GDPR articles regression: got %d, want %d", stats.Articles, expected.Statistics.Articles)
	}
	if stats.Chapters != expected.Statistics.Chapters {
		t.Errorf("GDPR chapters regression: got %d, want %d", stats.Chapters, expected.Statistics.Chapters)
	}
}

func TestGenericFormatNoRegressionCCPA(t *testing.T) {
	f := loadCCPAText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// CCPA should NOT be detected as generic
	if parser.format == FormatGeneric {
		t.Error("CCPA should not be detected as FormatGeneric")
	}

	stats := doc.Statistics()
	if stats.Chapters < 6 {
		t.Errorf("CCPA chapters regression: got %d, want at least 6", stats.Chapters)
	}
}

func TestGenericFormatNoRegressionGDPRWithRegistry(t *testing.T) {
	registry := loadPatternRegistry(t)
	expected := loadExpectedGDPR(t)

	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// GDPR should NOT be detected as generic with registry active
	if parser.format == FormatGeneric {
		t.Error("GDPR should not be detected as FormatGeneric with registry")
	}

	stats := doc.Statistics()
	if stats.Articles != expected.Statistics.Articles {
		t.Errorf("GDPR articles regression (registry): got %d, want %d", stats.Articles, expected.Statistics.Articles)
	}
}

func TestGenericFormatNoRegressionCCPAWithRegistry(t *testing.T) {
	registry := loadPatternRegistry(t)

	f := loadCCPAText(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// CCPA should NOT be detected as generic with registry active
	if parser.format == FormatGeneric {
		t.Error("CCPA should not be detected as FormatGeneric with registry")
	}

	stats := doc.Statistics()
	if stats.Chapters < 6 {
		t.Errorf("CCPA chapters regression (registry): got %d, want at least 6", stats.Chapters)
	}
}

// Format detection test: verify FormatGeneric is a string format

func TestFormatGenericConstant(t *testing.T) {
	if FormatGeneric != "generic" {
		t.Errorf("FormatGeneric = %q, want %q", FormatGeneric, "generic")
	}
}

// Test generic parsing via string content (no file)

func TestGenericParseFromContent(t *testing.T) {
	content := `Policy on Information Security

1. Purpose
This policy establishes the information security requirements.

2. Scope
This policy applies to all employees and contractors.

3. Definitions
"Information asset" means any data owned by the organization.
"Security incident" means any event compromising information security.

4. Requirements
All information assets shall be classified and protected.
`

	parser := NewParser()
	doc, err := parser.Parse(strings.NewReader(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parser.format != FormatGeneric {
		t.Errorf("Expected FormatGeneric, got %s", parser.format)
	}

	stats := doc.Statistics()
	t.Logf("Content parse: chapters=%d, articles=%d, definitions=%d",
		stats.Chapters, stats.Articles, stats.Definitions)

	// Should produce some structure
	totalStructure := stats.Chapters + stats.Articles
	if totalStructure < 1 {
		t.Error("Expected at least 1 structural element")
	}
}

// Benchmark

func BenchmarkGenericParseLargeDocument(b *testing.B) {
	// Build a large generic document
	var builder strings.Builder
	builder.WriteString("LARGE GENERIC DOCUMENT\n\n")

	for i := 1; i <= 20; i++ {
		builder.WriteString(strings.ToUpper("PART "))
		builder.WriteString(strings.Repeat("I", i%10+1))
		builder.WriteString("\n\n")

		for j := 1; j <= 10; j++ {
			builder.WriteString(string(rune('0'+j)) + ". Section content for item " + string(rune('0'+j)) + "\n")
			builder.WriteString("This is the body text of the section.\n\n")

			for k := 'a'; k <= 'c'; k++ {
				builder.WriteString("(" + string(k) + ") Sub-item text\n")
			}
			builder.WriteString("\n")
		}
	}

	content := builder.String()
	reader := strings.NewReader(content)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader.Reset(content)
		parser := NewParser()
		_, _ = parser.Parse(reader)
	}
}
