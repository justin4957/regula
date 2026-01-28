package extract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coolbeans/regula/pkg/pattern"
)

func loadUKDPA2018Text(t *testing.T) *os.File {
	t.Helper()

	testdataPath := filepath.Join("..", "..", "testdata", "uk-dpa2018.txt")

	f, err := os.Open(testdataPath)
	if err != nil {
		t.Fatalf("Failed to load UK DPA 2018 text: %v", err)
	}

	return f
}

func loadUKSIExampleText(t *testing.T) *os.File {
	t.Helper()

	testdataPath := filepath.Join("..", "..", "testdata", "uk-si-example.txt")

	f, err := os.Open(testdataPath)
	if err != nil {
		t.Fatalf("Failed to load UK SI example text: %v", err)
	}

	return f
}

func TestUKPatternRegistryDPA2018Parse(t *testing.T) {
	registry := loadPatternRegistry(t)

	f := loadUKDPA2018Text(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	stats := doc.Statistics()

	t.Logf("UK DPA 2018 Parse Results:")
	t.Logf("  Format:      %s", parser.format)
	t.Logf("  Type:        %s", doc.Type)
	t.Logf("  Identifier:  %s", doc.Identifier)
	t.Logf("  Chapters:    %d", stats.Chapters)
	t.Logf("  Articles:    %d", stats.Articles)
	t.Logf("  Definitions: %d", stats.Definitions)

	if parser.format != FormatUK {
		t.Errorf("Expected FormatUK, got %v", parser.format)
	}

	if doc.Type != DocumentTypeAct {
		t.Errorf("Expected DocumentTypeAct, got %v", doc.Type)
	}

	if doc.Identifier != "2018 c. 12" {
		t.Errorf("Expected identifier '2018 c. 12', got %q", doc.Identifier)
	}

	// DPA 2018 test data has 7 Parts + 2 Schedules = 9 chapters
	if stats.Chapters < 7 {
		t.Errorf("Expected at least 7 chapters (Parts), got %d", stats.Chapters)
	}

	// Should have at least 16 numbered sections
	if stats.Articles < 10 {
		t.Errorf("Expected at least 10 articles (sections), got %d", stats.Articles)
	}

	// Should have at least 3 definitions from section 3
	if stats.Definitions < 3 {
		t.Errorf("Expected at least 3 definitions, got %d", stats.Definitions)
	}
}

func TestUKPatternRegistryDPA2018Definitions(t *testing.T) {
	registry := loadPatternRegistry(t)

	f := loadUKDPA2018Text(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	t.Logf("UK DPA 2018 Definitions (%d):", len(doc.Definitions))
	for _, def := range doc.Definitions {
		t.Logf("  %d: %q", def.Number, def.Term)
	}

	// Check for expected definition terms
	expectedTerms := []string{
		"Personal data",
		"Controller",
		"Processor",
		"Commissioner",
	}

	foundTerms := make(map[string]bool)
	for _, def := range doc.Definitions {
		foundTerms[def.Term] = true
	}

	for _, expected := range expectedTerms {
		if !foundTerms[expected] {
			t.Errorf("Expected definition term %q not found", expected)
		}
	}
}

func TestUKPatternRegistrySIParse(t *testing.T) {
	registry := loadPatternRegistry(t)

	f := loadUKSIExampleText(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	stats := doc.Statistics()

	t.Logf("UK SI Parse Results:")
	t.Logf("  Format:      %s", parser.format)
	t.Logf("  Type:        %s", doc.Type)
	t.Logf("  Identifier:  %s", doc.Identifier)
	t.Logf("  Chapters:    %d", stats.Chapters)
	t.Logf("  Articles:    %d", stats.Articles)
	t.Logf("  Definitions: %d", stats.Definitions)

	if parser.format != FormatUK {
		t.Errorf("Expected FormatUK, got %v", parser.format)
	}

	if doc.Identifier != "S.I. 2019/419" {
		t.Errorf("Expected identifier 'S.I. 2019/419', got %q", doc.Identifier)
	}

	// SI test data has 4 Parts
	if stats.Chapters < 3 {
		t.Errorf("Expected at least 3 chapters (Parts), got %d", stats.Chapters)
	}

	// Should have at least 5 numbered regulations
	if stats.Articles < 5 {
		t.Errorf("Expected at least 5 articles (regulations), got %d", stats.Articles)
	}
}

func TestUKPatternRegistrySIDefinitions(t *testing.T) {
	registry := loadPatternRegistry(t)

	f := loadUKSIExampleText(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	t.Logf("UK SI Definitions (%d):", len(doc.Definitions))
	for _, def := range doc.Definitions {
		t.Logf("  %d: %q", def.Number, def.Term)
	}

	// SI uses quoted definitions: "the 2018 Act" means...
	if len(doc.Definitions) < 3 {
		t.Errorf("Expected at least 3 definitions from SI, got %d", len(doc.Definitions))
	}
}

func TestUKPatternRegistryFormatDetection(t *testing.T) {
	registry := loadPatternRegistry(t)

	testCases := []struct {
		name           string
		content        string
		expectedFormat string
	}{
		{
			name: "UK Act should match uk-primary",
			content: `Data Protection Act 2018
[2018 c. 12]
An Act to make provision for the regulation of the processing of information
BE IT ENACTED by the Queen's most Excellent Majesty, by and with the advice and consent of the Lords Spiritual and Temporal
PART 1
Preliminary`,
			expectedFormat: "uk-primary",
		},
		{
			name: "UK SI should match uk-si",
			content: `The Data Protection Regulations 2019
Statutory Instruments 2019 No. 419
S.I. 2019/419
Made 4th February 2019
Laid before Parliament 7th February 2019
Coming into force in accordance with regulation 1
The Secretary of State makes these Regulations`,
			expectedFormat: "uk-si",
		},
		{
			name: "EU Regulation should not match UK patterns",
			content: `Regulation (EU) 2016/679 of the European Parliament and of the Council
HAVE ADOPTED THIS REGULATION:
CHAPTER I
Article 1`,
			expectedFormat: "eu-regulation",
		},
		{
			name: "California Act should not match UK patterns",
			content: `CALIFORNIA CONSUMER PRIVACY ACT OF 2018
TITLE 1.81.5. CALIFORNIA CONSUMER PRIVACY ACT OF 2018
CHAPTER 1
General Provisions
Section 1798.100`,
			expectedFormat: "us-california",
		},
	}

	detector := pattern.NewFormatDetector(registry)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			best := detector.DetectBest(tc.content)
			if best == nil {
				t.Fatal("No format detected")
			}
			if best.FormatID != tc.expectedFormat {
				t.Errorf("Expected format %q, got %q (confidence: %.1f%%)",
					tc.expectedFormat, best.FormatID, best.Confidence*100)

				// Show all matches for debugging
				allMatches := detector.DetectAll(tc.content)
				for _, m := range allMatches {
					t.Logf("  %s: %.1f%% confidence", m.FormatID, m.Confidence*100)
				}
			}
		})
	}
}

func TestUKPatternRegistryPatternLoading(t *testing.T) {
	registry := loadPatternRegistry(t)

	// Verify uk-primary pattern was loaded
	ukPrimaryPattern, found := registry.Get("uk-primary")
	if !found {
		t.Fatal("uk-primary pattern not loaded from patterns directory")
	}

	if ukPrimaryPattern.Version != "2.0.0" {
		t.Errorf("uk-primary version = %q, want %q", ukPrimaryPattern.Version, "2.0.0")
	}

	if ukPrimaryPattern.Jurisdiction != "GB" {
		t.Errorf("uk-primary jurisdiction = %q, want %q", ukPrimaryPattern.Jurisdiction, "GB")
	}

	if len(ukPrimaryPattern.Detection.RequiredIndicators) < 2 {
		t.Errorf("Expected at least 2 required indicators, got %d", len(ukPrimaryPattern.Detection.RequiredIndicators))
	}

	if len(ukPrimaryPattern.Detection.OptionalIndicators) < 5 {
		t.Errorf("Expected at least 5 optional indicators, got %d", len(ukPrimaryPattern.Detection.OptionalIndicators))
	}

	if len(ukPrimaryPattern.Structure.Hierarchy) < 4 {
		t.Errorf("Expected at least 4 hierarchy levels, got %d", len(ukPrimaryPattern.Structure.Hierarchy))
	}

	// Verify uk-si pattern was loaded
	ukSIPattern, found := registry.Get("uk-si")
	if !found {
		t.Fatal("uk-si pattern not loaded from patterns directory")
	}

	if ukSIPattern.Version != "2.0.0" {
		t.Errorf("uk-si version = %q, want %q", ukSIPattern.Version, "2.0.0")
	}

	if ukSIPattern.Jurisdiction != "GB" {
		t.Errorf("uk-si jurisdiction = %q, want %q", ukSIPattern.Jurisdiction, "GB")
	}

	if len(ukSIPattern.Detection.RequiredIndicators) < 2 {
		t.Errorf("Expected at least 2 required indicators, got %d", len(ukSIPattern.Detection.RequiredIndicators))
	}

	if len(ukSIPattern.Detection.NegativeIndicators) < 3 {
		t.Errorf("Expected at least 3 negative indicators, got %d", len(ukSIPattern.Detection.NegativeIndicators))
	}
}

func TestUKPatternRegistryNoGDPRRegression(t *testing.T) {
	registry := loadPatternRegistry(t)
	expected := loadExpectedGDPR(t)

	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	stats := doc.Statistics()

	if stats.Articles != expected.Statistics.Articles {
		t.Errorf("GDPR article count regression: got %d, want %d", stats.Articles, expected.Statistics.Articles)
	}
	if stats.Chapters != expected.Statistics.Chapters {
		t.Errorf("GDPR chapter count regression: got %d, want %d", stats.Chapters, expected.Statistics.Chapters)
	}
	if stats.Definitions != expected.Statistics.Definitions {
		t.Errorf("GDPR definition count regression: got %d, want %d", stats.Definitions, expected.Statistics.Definitions)
	}
}

func TestUKPatternRegistryNoCCPARegression(t *testing.T) {
	registry := loadPatternRegistry(t)

	f := loadCCPAText(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	stats := doc.Statistics()

	t.Logf("CCPA regression check: Articles=%d, Chapters=%d, Definitions=%d",
		stats.Articles, stats.Chapters, stats.Definitions)

	if stats.Articles < 10 {
		t.Errorf("CCPA article count regression: got %d, expected >= 10", stats.Articles)
	}
	if stats.Chapters < 6 {
		t.Errorf("CCPA chapter count regression: got %d, expected >= 6", stats.Chapters)
	}
	if stats.Definitions < 5 {
		t.Errorf("CCPA definition count regression: got %d, expected >= 5", stats.Definitions)
	}
}

func TestUKPatternRegistryNoVCDPARegression(t *testing.T) {
	registry := loadPatternRegistry(t)

	f := loadVCDPAText(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	stats := doc.Statistics()

	t.Logf("VCDPA regression check: Articles=%d, Chapters=%d, Definitions=%d",
		stats.Articles, stats.Chapters, stats.Definitions)

	if stats.Articles < 3 {
		t.Errorf("VCDPA article count regression: got %d, expected >= 3", stats.Articles)
	}
	if stats.Chapters < 1 {
		t.Errorf("VCDPA chapter count regression: got %d, expected >= 1", stats.Chapters)
	}
}

func TestUKPatternDetectorCrossBoundary(t *testing.T) {
	registry := loadPatternRegistry(t)
	detector := pattern.NewFormatDetector(registry)

	testCases := []struct {
		name           string
		content        string
		expectedFormat string
	}{
		{
			name: "UK Act content should match uk-primary",
			content: `Data Protection Act 2018
[2018 c. 12]
BE IT ENACTED by the Queen's most Excellent Majesty
PART 1
1. Overview`,
			expectedFormat: "uk-primary",
		},
		{
			name: "UK SI content should match uk-si",
			content: `Statutory Instruments 2019 No. 419
S.I. 2019/419
Made 4th February 2019
Laid before Parliament 7th February 2019
Coming into force
The Secretary of State
regulation 1`,
			expectedFormat: "uk-si",
		},
		{
			name: "CCPA content should not match UK patterns",
			content: `CALIFORNIA CONSUMER PRIVACY ACT OF 2018
TITLE 1.81.5. CALIFORNIA CONSUMER PRIVACY ACT OF 2018
Section 1798.100`,
			expectedFormat: "us-california",
		},
		{
			name: "VCDPA content should not match UK patterns",
			content: `VIRGINIA CONSUMER DATA PROTECTION ACT
Code of Virginia Title 59.1 Chapter 53
Section 59.1-575`,
			expectedFormat: "us-virginia",
		},
		{
			name: "EU Regulation content should not match UK patterns",
			content: `Regulation (EU) 2016/679 of the European Parliament and of the Council
HAVE ADOPTED THIS REGULATION:
CHAPTER I
Article 1`,
			expectedFormat: "eu-regulation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			best := detector.DetectBest(tc.content)
			if best == nil {
				t.Fatal("No format detected")
			}
			if best.FormatID != tc.expectedFormat {
				t.Errorf("Expected format %q, got %q (confidence: %.1f%%)",
					tc.expectedFormat, best.FormatID, best.Confidence*100)

				// Show all matches for debugging
				allMatches := detector.DetectAll(tc.content)
				for _, m := range allMatches {
					t.Logf("  %s: %.1f%% confidence", m.FormatID, m.Confidence*100)
				}
			}
		})
	}
}

func TestUKPatternRegistryDPA2018Structure(t *testing.T) {
	registry := loadPatternRegistry(t)

	f := loadUKDPA2018Text(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	t.Logf("DPA 2018 Structure:")
	for _, ch := range doc.Chapters {
		t.Logf("  Chapter %s: %s (%d articles)", ch.Number, ch.Title, len(ch.Articles))
		for _, art := range ch.Articles {
			t.Logf("    Section %d: %s", art.Number, art.Title)
		}
	}

	// Verify Part 1 exists with title "Preliminary"
	foundPart1 := false
	for _, ch := range doc.Chapters {
		if ch.Number == "1" && strings.Contains(ch.Title, "Preliminary") {
			foundPart1 = true
			break
		}
	}
	if !foundPart1 {
		t.Error("Expected Part 1 with title 'Preliminary'")
	}

	// Verify schedules were parsed
	scheduleCount := 0
	for _, ch := range doc.Chapters {
		if strings.HasPrefix(ch.Number, "S") {
			scheduleCount++
		}
	}
	if scheduleCount < 2 {
		t.Errorf("Expected at least 2 schedules, got %d", scheduleCount)
	}
}

func TestUKPatternRegistryLegacyDetection(t *testing.T) {
	// Test the legacy (non-registry) detection for UK content
	parser := NewParser()

	ukActContent := `Data Protection Act 2018
[2018 c. 12]
An Act to make provision for the regulation
BE IT ENACTED by the Queen's most Excellent Majesty, by and with the advice and consent of the Lords Spiritual and Temporal, and Commons
Royal Assent
House of Commons
PART 1
Preliminary`

	doc, err := parser.Parse(strings.NewReader(ukActContent))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parser.format != FormatUK {
		t.Errorf("Expected FormatUK from legacy detection, got %v", parser.format)
	}

	if doc.Type != DocumentTypeAct {
		t.Errorf("Expected DocumentTypeAct, got %v", doc.Type)
	}
}
