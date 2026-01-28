package extract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coolbeans/regula/pkg/pattern"
)

func loadVCDPAText(t *testing.T) *os.File {
	t.Helper()

	testdataPath := filepath.Join("..", "..", "testdata", "vcdpa.txt")

	f, err := os.Open(testdataPath)
	if err != nil {
		t.Fatalf("Failed to load VCDPA text: %v", err)
	}

	return f
}

func TestUSPatternRegistryCCPAParse(t *testing.T) {
	registry := loadPatternRegistry(t)

	f := loadCCPAText(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	stats := doc.Statistics()

	t.Logf("Pattern Registry CCPA Parse Results:")
	t.Logf("  Articles:    %d", stats.Articles)
	t.Logf("  Chapters:    %d", stats.Chapters)
	t.Logf("  Sections:    %d", stats.Sections)
	t.Logf("  Definitions: %d", stats.Definitions)

	if stats.Chapters < 6 {
		t.Errorf("Expected at least 6 chapters, got %d", stats.Chapters)
	}
	if stats.Articles < 10 {
		t.Errorf("Expected at least 10 articles, got %d", stats.Articles)
	}
	if stats.Definitions < 5 {
		t.Errorf("Expected at least 5 definitions, got %d", stats.Definitions)
	}
}

func TestUSPatternRegistryCCPAIdenticalOutput(t *testing.T) {
	registry := loadPatternRegistry(t)

	// Parse with legacy parser
	f1 := loadCCPAText(t)
	defer f1.Close()

	legacyParser := NewParser()
	legacyDoc, err := legacyParser.Parse(f1)
	if err != nil {
		t.Fatalf("Legacy parse failed: %v", err)
	}

	// Parse with registry parser
	f2 := loadCCPAText(t)
	defer f2.Close()

	registryParser := NewParserWithRegistry(registry)
	registryDoc, err := registryParser.Parse(f2)
	if err != nil {
		t.Fatalf("Registry parse failed: %v", err)
	}

	legacyStats := legacyDoc.Statistics()
	registryStats := registryDoc.Statistics()

	t.Logf("CCPA Comparison (Legacy vs Registry):")
	t.Logf("  Articles:    %d vs %d", legacyStats.Articles, registryStats.Articles)
	t.Logf("  Chapters:    %d vs %d", legacyStats.Chapters, registryStats.Chapters)
	t.Logf("  Sections:    %d vs %d", legacyStats.Sections, registryStats.Sections)
	t.Logf("  Definitions: %d vs %d", legacyStats.Definitions, registryStats.Definitions)

	if legacyStats.Articles != registryStats.Articles {
		t.Errorf("Article count diverged: legacy=%d, registry=%d", legacyStats.Articles, registryStats.Articles)
	}
	if legacyStats.Chapters != registryStats.Chapters {
		t.Errorf("Chapter count diverged: legacy=%d, registry=%d", legacyStats.Chapters, registryStats.Chapters)
	}
	if legacyStats.Definitions != registryStats.Definitions {
		t.Errorf("Definition count diverged: legacy=%d, registry=%d", legacyStats.Definitions, registryStats.Definitions)
	}

	// Compare article titles
	legacyArticles := legacyDoc.AllArticles()
	registryArticles := registryDoc.AllArticles()

	if len(legacyArticles) != len(registryArticles) {
		t.Fatalf("CCPA article count differs: legacy=%d, registry=%d", len(legacyArticles), len(registryArticles))
	}

	for i := range legacyArticles {
		if legacyArticles[i].Number != registryArticles[i].Number {
			t.Errorf("Article %d number differs: legacy=%d, registry=%d",
				i, legacyArticles[i].Number, registryArticles[i].Number)
		}
		if legacyArticles[i].Title != registryArticles[i].Title {
			t.Errorf("Article %d title differs: legacy=%q, registry=%q",
				legacyArticles[i].Number, legacyArticles[i].Title, registryArticles[i].Title)
		}
	}

	// Compare definitions
	if len(legacyDoc.Definitions) != len(registryDoc.Definitions) {
		t.Errorf("Definition count differs: legacy=%d, registry=%d",
			len(legacyDoc.Definitions), len(registryDoc.Definitions))
	} else {
		for i := range legacyDoc.Definitions {
			if legacyDoc.Definitions[i].Term != registryDoc.Definitions[i].Term {
				t.Errorf("Definition %d term differs: legacy=%q, registry=%q",
					i, legacyDoc.Definitions[i].Term, registryDoc.Definitions[i].Term)
			}
		}
	}
}

func TestUSPatternRegistryVCDPAParse(t *testing.T) {
	registry := loadPatternRegistry(t)

	f := loadVCDPAText(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	stats := doc.Statistics()

	t.Logf("Pattern Registry VCDPA Parse Results:")
	t.Logf("  Articles:    %d", stats.Articles)
	t.Logf("  Chapters:    %d", stats.Chapters)
	t.Logf("  Sections:    %d", stats.Sections)
	t.Logf("  Definitions: %d", stats.Definitions)

	if stats.Chapters < 1 {
		t.Errorf("Expected at least 1 chapter, got %d", stats.Chapters)
	}
	if stats.Articles < 3 {
		t.Errorf("Expected at least 3 articles, got %d", stats.Articles)
	}
}

func TestUSPatternRegistryVCDPAIdenticalOutput(t *testing.T) {
	registry := loadPatternRegistry(t)

	// Parse with legacy parser
	f1 := loadVCDPAText(t)
	defer f1.Close()

	legacyParser := NewParser()
	legacyDoc, err := legacyParser.Parse(f1)
	if err != nil {
		t.Fatalf("Legacy parse failed: %v", err)
	}

	// Parse with registry parser
	f2 := loadVCDPAText(t)
	defer f2.Close()

	registryParser := NewParserWithRegistry(registry)
	registryDoc, err := registryParser.Parse(f2)
	if err != nil {
		t.Fatalf("Registry parse failed: %v", err)
	}

	legacyStats := legacyDoc.Statistics()
	registryStats := registryDoc.Statistics()

	t.Logf("VCDPA Comparison (Legacy vs Registry):")
	t.Logf("  Articles:    %d vs %d", legacyStats.Articles, registryStats.Articles)
	t.Logf("  Chapters:    %d vs %d", legacyStats.Chapters, registryStats.Chapters)
	t.Logf("  Sections:    %d vs %d", legacyStats.Sections, registryStats.Sections)
	t.Logf("  Definitions: %d vs %d", legacyStats.Definitions, registryStats.Definitions)

	if legacyStats.Articles != registryStats.Articles {
		t.Errorf("Article count diverged: legacy=%d, registry=%d", legacyStats.Articles, registryStats.Articles)
	}
	if legacyStats.Chapters != registryStats.Chapters {
		t.Errorf("Chapter count diverged: legacy=%d, registry=%d", legacyStats.Chapters, registryStats.Chapters)
	}
	if legacyStats.Definitions != registryStats.Definitions {
		t.Errorf("Definition count diverged: legacy=%d, registry=%d", legacyStats.Definitions, registryStats.Definitions)
	}

	// Compare article titles
	legacyArticles := legacyDoc.AllArticles()
	registryArticles := registryDoc.AllArticles()

	if len(legacyArticles) != len(registryArticles) {
		t.Fatalf("VCDPA article count differs: legacy=%d, registry=%d", len(legacyArticles), len(registryArticles))
	}

	for i := range legacyArticles {
		if legacyArticles[i].Number != registryArticles[i].Number {
			t.Errorf("Article %d number differs: legacy=%d, registry=%d",
				i, legacyArticles[i].Number, registryArticles[i].Number)
		}
		if legacyArticles[i].Title != registryArticles[i].Title {
			t.Errorf("Article %d title differs: legacy=%q, registry=%q",
				legacyArticles[i].Number, legacyArticles[i].Title, registryArticles[i].Title)
		}
	}
}

func TestUSPatternRegistryCAFormatDetection(t *testing.T) {
	registry := loadPatternRegistry(t)

	caContent := `CALIFORNIA CONSUMER PRIVACY ACT OF 2018

TITLE 1.81.5. CALIFORNIA CONSUMER PRIVACY ACT OF 2018

CHAPTER 1
General Provisions

Section 1798.100
Title

This title may be cited as the California Consumer Privacy Act of 2018.`

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(strings.NewReader(caContent))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should detect US format
	if parser.format != FormatUS {
		t.Errorf("Expected FormatUS, got %v", parser.format)
	}

	// Should have at least one chapter
	if len(doc.Chapters) == 0 {
		t.Error("Expected at least 1 chapter")
	}
}

func TestUSPatternRegistryVAFormatDetection(t *testing.T) {
	registry := loadPatternRegistry(t)

	vaContent := `VIRGINIA CONSUMER DATA PROTECTION ACT

TITLE 59.1. TRADE AND COMMERCE
CHAPTER 53. CONSUMER DATA PROTECTION ACT

Code of Virginia Title 59.1 Chapter 53

CHAPTER 1
General Provisions

Section 59.1-575
Definitions`

	parser := NewParserWithRegistry(registry)
	_, err := parser.Parse(strings.NewReader(vaContent))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should detect US format
	if parser.format != FormatUS {
		t.Errorf("Expected FormatUS, got %v", parser.format)
	}
}

func TestUSPatternRegistryPatternLoading(t *testing.T) {
	registry := loadPatternRegistry(t)

	// Verify us-california pattern was loaded
	caPattern, found := registry.Get("us-california")
	if !found {
		t.Fatal("us-california pattern not loaded from patterns directory")
	}

	if caPattern.Version != "2.0.0" {
		t.Errorf("us-california version = %q, want %q", caPattern.Version, "2.0.0")
	}

	if caPattern.Jurisdiction != "US-CA" {
		t.Errorf("us-california jurisdiction = %q, want %q", caPattern.Jurisdiction, "US-CA")
	}

	if len(caPattern.Detection.RequiredIndicators) < 2 {
		t.Errorf("Expected at least 2 required indicators, got %d", len(caPattern.Detection.RequiredIndicators))
	}

	if len(caPattern.Detection.OptionalIndicators) < 5 {
		t.Errorf("Expected at least 5 optional indicators, got %d", len(caPattern.Detection.OptionalIndicators))
	}

	if len(caPattern.Structure.Hierarchy) < 4 {
		t.Errorf("Expected at least 4 hierarchy levels, got %d", len(caPattern.Structure.Hierarchy))
	}

	// Verify us-virginia pattern was loaded
	vaPattern, found := registry.Get("us-virginia")
	if !found {
		t.Fatal("us-virginia pattern not loaded from patterns directory")
	}

	if vaPattern.Version != "2.0.0" {
		t.Errorf("us-virginia version = %q, want %q", vaPattern.Version, "2.0.0")
	}

	if vaPattern.Jurisdiction != "US-VA" {
		t.Errorf("us-virginia jurisdiction = %q, want %q", vaPattern.Jurisdiction, "US-VA")
	}

	if len(vaPattern.Detection.RequiredIndicators) < 2 {
		t.Errorf("Expected at least 2 required indicators, got %d", len(vaPattern.Detection.RequiredIndicators))
	}
}

func TestUSPatternRegistryNoGDPRRegression(t *testing.T) {
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

func TestUSPatternDetectorCrossBoundary(t *testing.T) {
	registry := loadPatternRegistry(t)
	detector := pattern.NewFormatDetector(registry)

	testCases := []struct {
		name           string
		content        string
		expectedFormat string
	}{
		{
			name: "CCPA content should match us-california",
			content: `CALIFORNIA CONSUMER PRIVACY ACT OF 2018
TITLE 1.81.5. CALIFORNIA CONSUMER PRIVACY ACT OF 2018
Section 1798.100`,
			expectedFormat: "us-california",
		},
		{
			name: "VCDPA content should match us-virginia",
			content: `VIRGINIA CONSUMER DATA PROTECTION ACT
Code of Virginia Title 59.1 Chapter 53
Section 59.1-575`,
			expectedFormat: "us-virginia",
		},
		{
			name: "CPA content should match us-colorado",
			content: `COLORADO PRIVACY ACT
Colorado Revised Statutes Title 6 Article 1 Part 13
C.R.S. 6-1-1301 et seq.
Section 6-1-1301`,
			expectedFormat: "us-colorado",
		},
		{
			name: "CTDPA content should match us-connecticut",
			content: `CONNECTICUT DATA PRIVACY ACT
Connecticut General Statutes Title 42 Chapter 815e
Conn. Gen. Stat. Section 42-515 et seq.
Section 42-515`,
			expectedFormat: "us-connecticut",
		},
		{
			name: "TDPSA content should match us-texas",
			content: `TEXAS DATA PRIVACY AND SECURITY ACT
Texas Business and Commerce Code Title 11 Subtitle C Chapter 541
Tex. Bus. & Com. Code Section 541.001 et seq.
Section 541.001`,
			expectedFormat: "us-texas",
		},
		{
			name: "UCPA content should match us-utah",
			content: `UTAH CONSUMER PRIVACY ACT
Utah Code Title 13 Chapter 61
U.C.A. Section 13-61-101 et seq.
Section 13-61-101`,
			expectedFormat: "us-utah",
		},
		{
			name: "ICDPA content should match us-iowa",
			content: `IOWA CONSUMER DATA PROTECTION ACT
Iowa Code Chapter 715D
Iowa Code Section 715D.1 et seq.
Section 715D.1`,
			expectedFormat: "us-iowa",
		},
		{
			name: "EU Regulation should not match US patterns",
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
