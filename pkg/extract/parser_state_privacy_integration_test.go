package extract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coolbeans/regula/pkg/pattern"
)

// Test data loader helpers for state privacy law texts.

func loadCPAText(t *testing.T) *os.File {
	t.Helper()
	testdataPath := filepath.Join("..", "..", "testdata", "cpa.txt")
	f, err := os.Open(testdataPath)
	if err != nil {
		t.Fatalf("Failed to load CPA text: %v", err)
	}
	return f
}

func loadCTDPAText(t *testing.T) *os.File {
	t.Helper()
	testdataPath := filepath.Join("..", "..", "testdata", "ctdpa.txt")
	f, err := os.Open(testdataPath)
	if err != nil {
		t.Fatalf("Failed to load CTDPA text: %v", err)
	}
	return f
}

func loadTDPSAText(t *testing.T) *os.File {
	t.Helper()
	testdataPath := filepath.Join("..", "..", "testdata", "tdpsa.txt")
	f, err := os.Open(testdataPath)
	if err != nil {
		t.Fatalf("Failed to load TDPSA text: %v", err)
	}
	return f
}

func loadUCPAText(t *testing.T) *os.File {
	t.Helper()
	testdataPath := filepath.Join("..", "..", "testdata", "ucpa.txt")
	f, err := os.Open(testdataPath)
	if err != nil {
		t.Fatalf("Failed to load UCPA text: %v", err)
	}
	return f
}

func loadICDPAText(t *testing.T) *os.File {
	t.Helper()
	testdataPath := filepath.Join("..", "..", "testdata", "icdpa.txt")
	f, err := os.Open(testdataPath)
	if err != nil {
		t.Fatalf("Failed to load ICDPA text: %v", err)
	}
	return f
}

// TestStatePrivacyPatternLoading validates that all new state pattern YAML files
// load correctly and have the expected metadata.
func TestStatePrivacyPatternLoading(t *testing.T) {
	registry := loadPatternRegistry(t)

	statePatterns := []struct {
		formatID     string
		jurisdiction string
		version      string
	}{
		{"us-colorado", "US-CO", "2.0.0"},
		{"us-connecticut", "US-CT", "2.0.0"},
		{"us-texas", "US-TX", "2.0.0"},
		{"us-utah", "US-UT", "2.0.0"},
		{"us-iowa", "US-IA", "2.0.0"},
	}

	for _, sp := range statePatterns {
		t.Run(sp.formatID, func(t *testing.T) {
			patternDef, found := registry.Get(sp.formatID)
			if !found {
				t.Fatalf("%s pattern not loaded from patterns directory", sp.formatID)
			}

			if patternDef.Version != sp.version {
				t.Errorf("%s version = %q, want %q", sp.formatID, patternDef.Version, sp.version)
			}

			if patternDef.Jurisdiction != sp.jurisdiction {
				t.Errorf("%s jurisdiction = %q, want %q", sp.formatID, patternDef.Jurisdiction, sp.jurisdiction)
			}

			if len(patternDef.Detection.RequiredIndicators) < 2 {
				t.Errorf("%s: expected at least 2 required indicators, got %d",
					sp.formatID, len(patternDef.Detection.RequiredIndicators))
			}

			if len(patternDef.Detection.OptionalIndicators) < 5 {
				t.Errorf("%s: expected at least 5 optional indicators, got %d",
					sp.formatID, len(patternDef.Detection.OptionalIndicators))
			}

			if len(patternDef.Detection.NegativeIndicators) < 5 {
				t.Errorf("%s: expected at least 5 negative indicators, got %d",
					sp.formatID, len(patternDef.Detection.NegativeIndicators))
			}

			if len(patternDef.Structure.Hierarchy) < 2 {
				t.Errorf("%s: expected at least 2 hierarchy levels, got %d",
					sp.formatID, len(patternDef.Structure.Hierarchy))
			}
		})
	}
}

// TestStatePrivacyCPAParse validates that the Colorado Privacy Act parses correctly.
func TestStatePrivacyCPAParse(t *testing.T) {
	registry := loadPatternRegistry(t)

	f := loadCPAText(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	stats := doc.Statistics()

	t.Logf("Colorado CPA Parse Results:")
	t.Logf("  Format:      %s", parser.format)
	t.Logf("  Type:        %s", doc.Type)
	t.Logf("  Identifier:  %s", doc.Identifier)
	t.Logf("  Chapters:    %d", stats.Chapters)
	t.Logf("  Articles:    %d", stats.Articles)
	t.Logf("  Definitions: %d", stats.Definitions)

	if parser.format != FormatUS {
		t.Errorf("Expected FormatUS, got %v", parser.format)
	}

	// CPA has PART 13 as its main structural division
	if stats.Chapters < 1 {
		t.Errorf("Expected at least 1 chapter (part), got %d", stats.Chapters)
	}

	// Should have multiple sections (6-1-1301 through 6-1-1310)
	if stats.Articles < 5 {
		t.Errorf("Expected at least 5 articles (sections), got %d", stats.Articles)
	}

	// Should extract definitions from Section 6-1-1303
	if stats.Definitions < 5 {
		t.Errorf("Expected at least 5 definitions, got %d", stats.Definitions)
	}

	// Verify identifier extraction
	if doc.Identifier == "" {
		t.Error("Expected non-empty identifier")
	} else {
		t.Logf("  Identifier: %s", doc.Identifier)
		if !strings.Contains(doc.Identifier, "C.R.S.") {
			t.Errorf("Expected identifier containing 'C.R.S.', got %q", doc.Identifier)
		}
	}
}

// TestStatePrivacyCTDPAParse validates that the Connecticut CTDPA parses correctly.
func TestStatePrivacyCTDPAParse(t *testing.T) {
	registry := loadPatternRegistry(t)

	f := loadCTDPAText(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	stats := doc.Statistics()

	t.Logf("Connecticut CTDPA Parse Results:")
	t.Logf("  Format:      %s", parser.format)
	t.Logf("  Type:        %s", doc.Type)
	t.Logf("  Identifier:  %s", doc.Identifier)
	t.Logf("  Chapters:    %d", stats.Chapters)
	t.Logf("  Articles:    %d", stats.Articles)
	t.Logf("  Definitions: %d", stats.Definitions)

	if parser.format != FormatUS {
		t.Errorf("Expected FormatUS, got %v", parser.format)
	}

	if stats.Chapters < 1 {
		t.Errorf("Expected at least 1 chapter, got %d", stats.Chapters)
	}

	// Should have multiple sections (42-515 through 42-523)
	if stats.Articles < 5 {
		t.Errorf("Expected at least 5 articles (sections), got %d", stats.Articles)
	}

	// Should extract definitions from Section 42-515
	if stats.Definitions < 5 {
		t.Errorf("Expected at least 5 definitions, got %d", stats.Definitions)
	}

	// Verify identifier extraction
	if doc.Identifier == "" {
		t.Error("Expected non-empty identifier")
	} else {
		t.Logf("  Identifier: %s", doc.Identifier)
		if !strings.Contains(doc.Identifier, "Conn.") {
			t.Errorf("Expected identifier containing 'Conn.', got %q", doc.Identifier)
		}
	}
}

// TestStatePrivacyTDPSAParse validates that the Texas TDPSA parses correctly.
func TestStatePrivacyTDPSAParse(t *testing.T) {
	registry := loadPatternRegistry(t)

	f := loadTDPSAText(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	stats := doc.Statistics()

	t.Logf("Texas TDPSA Parse Results:")
	t.Logf("  Format:      %s", parser.format)
	t.Logf("  Type:        %s", doc.Type)
	t.Logf("  Identifier:  %s", doc.Identifier)
	t.Logf("  Chapters:    %d", stats.Chapters)
	t.Logf("  Articles:    %d", stats.Articles)
	t.Logf("  Definitions: %d", stats.Definitions)

	if parser.format != FormatUS {
		t.Errorf("Expected FormatUS, got %v", parser.format)
	}

	// Texas has CHAPTER 541 and multiple SUBCHAPTER entries
	if stats.Chapters < 1 {
		t.Errorf("Expected at least 1 chapter, got %d", stats.Chapters)
	}

	// Should have multiple sections (541.001 through 541.151)
	if stats.Articles < 5 {
		t.Errorf("Expected at least 5 articles (sections), got %d", stats.Articles)
	}

	// Should extract definitions from Section 541.001
	if stats.Definitions < 5 {
		t.Errorf("Expected at least 5 definitions, got %d", stats.Definitions)
	}

	// Verify identifier extraction
	if doc.Identifier == "" {
		t.Error("Expected non-empty identifier")
	} else {
		t.Logf("  Identifier: %s", doc.Identifier)
		if !strings.Contains(doc.Identifier, "Tex.") {
			t.Errorf("Expected identifier containing 'Tex.', got %q", doc.Identifier)
		}
	}
}

// TestStatePrivacyUCPAParse validates that the Utah UCPA parses correctly.
func TestStatePrivacyUCPAParse(t *testing.T) {
	registry := loadPatternRegistry(t)

	f := loadUCPAText(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	stats := doc.Statistics()

	t.Logf("Utah UCPA Parse Results:")
	t.Logf("  Format:      %s", parser.format)
	t.Logf("  Type:        %s", doc.Type)
	t.Logf("  Identifier:  %s", doc.Identifier)
	t.Logf("  Chapters:    %d", stats.Chapters)
	t.Logf("  Articles:    %d", stats.Articles)
	t.Logf("  Definitions: %d", stats.Definitions)

	if parser.format != FormatUS {
		t.Errorf("Expected FormatUS, got %v", parser.format)
	}

	// Utah has multiple PART entries
	if stats.Chapters < 1 {
		t.Errorf("Expected at least 1 chapter (part), got %d", stats.Chapters)
	}

	// Should have multiple sections (13-61-101 through 13-61-401)
	if stats.Articles < 5 {
		t.Errorf("Expected at least 5 articles (sections), got %d", stats.Articles)
	}

	// Should extract definitions from Section 13-61-101
	if stats.Definitions < 5 {
		t.Errorf("Expected at least 5 definitions, got %d", stats.Definitions)
	}

	// Verify identifier extraction
	if doc.Identifier == "" {
		t.Error("Expected non-empty identifier")
	} else {
		t.Logf("  Identifier: %s", doc.Identifier)
		if !strings.Contains(doc.Identifier, "U.C.A.") {
			t.Errorf("Expected identifier containing 'U.C.A.', got %q", doc.Identifier)
		}
	}
}

// TestStatePrivacyICDPAParse validates that the Iowa ICDPA parses correctly.
func TestStatePrivacyICDPAParse(t *testing.T) {
	registry := loadPatternRegistry(t)

	f := loadICDPAText(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	stats := doc.Statistics()

	t.Logf("Iowa ICDPA Parse Results:")
	t.Logf("  Format:      %s", parser.format)
	t.Logf("  Type:        %s", doc.Type)
	t.Logf("  Identifier:  %s", doc.Identifier)
	t.Logf("  Chapters:    %d", stats.Chapters)
	t.Logf("  Articles:    %d", stats.Articles)
	t.Logf("  Definitions: %d", stats.Definitions)

	if parser.format != FormatUS {
		t.Errorf("Expected FormatUS, got %v", parser.format)
	}

	// Should have sections (715D.1 through 715D.7)
	if stats.Articles < 3 {
		t.Errorf("Expected at least 3 articles (sections), got %d", stats.Articles)
	}

	// Should extract definitions from Section 715D.1
	if stats.Definitions < 5 {
		t.Errorf("Expected at least 5 definitions, got %d", stats.Definitions)
	}

	// Verify identifier extraction
	if doc.Identifier == "" {
		t.Error("Expected non-empty identifier")
	} else {
		t.Logf("  Identifier: %s", doc.Identifier)
		if !strings.Contains(doc.Identifier, "Iowa Code") {
			t.Errorf("Expected identifier containing 'Iowa Code', got %q", doc.Identifier)
		}
	}
}

// TestStatePrivacyFormatDetection verifies each state's test content is detected
// as FormatUS with the correct jurisdiction-specific pattern winning.
func TestStatePrivacyFormatDetection(t *testing.T) {
	registry := loadPatternRegistry(t)
	detector := pattern.NewFormatDetector(registry)

	testCases := []struct {
		name           string
		content        string
		expectedFormat string
	}{
		{
			name: "Colorado CPA content",
			content: `COLORADO PRIVACY ACT
Colorado Revised Statutes Title 6 Article 1 Part 13
C.R.S. 6-1-1301 et seq.
Section 6-1-1301
Short title`,
			expectedFormat: "us-colorado",
		},
		{
			name: "Connecticut CTDPA content",
			content: `CONNECTICUT DATA PRIVACY ACT
Connecticut General Statutes Title 42 Chapter 815e
Conn. Gen. Stat. Section 42-515 et seq.
Section 42-515
Definitions`,
			expectedFormat: "us-connecticut",
		},
		{
			name: "Texas TDPSA content",
			content: `TEXAS DATA PRIVACY AND SECURITY ACT
Texas Business and Commerce Code Title 11 Subtitle C Chapter 541
Tex. Bus. & Com. Code Section 541.001 et seq.
Section 541.001
Definitions`,
			expectedFormat: "us-texas",
		},
		{
			name: "Utah UCPA content",
			content: `UTAH CONSUMER PRIVACY ACT
Utah Code Title 13 Chapter 61
U.C.A. Section 13-61-101 et seq.
Section 13-61-101
Definitions`,
			expectedFormat: "us-utah",
		},
		{
			name: "Iowa ICDPA content",
			content: `IOWA CONSUMER DATA PROTECTION ACT
Iowa Code Chapter 715D
Iowa Code Section 715D.1 et seq.
Section 715D.1
Definitions`,
			expectedFormat: "us-iowa",
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
			} else {
				t.Logf("  %s detected with %.1f%% confidence", best.FormatID, best.Confidence*100)
			}
		})
	}
}

// TestStatePrivacyCrossBoundaryDetection verifies that each state's content is
// correctly distinguished from all other states and non-US formats.
func TestStatePrivacyCrossBoundaryDetection(t *testing.T) {
	registry := loadPatternRegistry(t)
	detector := pattern.NewFormatDetector(registry)

	testCases := []struct {
		name           string
		content        string
		expectedFormat string
	}{
		{
			name: "Colorado content not detected as California",
			content: `COLORADO PRIVACY ACT
Colorado Revised Statutes
Section 6-1-1303
Definitions`,
			expectedFormat: "us-colorado",
		},
		{
			name: "Connecticut content not detected as Virginia",
			content: `CONNECTICUT DATA PRIVACY ACT
Connecticut General Statutes
Section 42-515
Definitions`,
			expectedFormat: "us-connecticut",
		},
		{
			name: "Texas content not detected as California",
			content: `TEXAS DATA PRIVACY AND SECURITY ACT
Texas Business and Commerce Code
Section 541.001
Definitions`,
			expectedFormat: "us-texas",
		},
		{
			name: "Utah content not detected as Colorado",
			content: `UTAH CONSUMER PRIVACY ACT
Utah Code Title 13 Chapter 61
Section 13-61-101
Definitions`,
			expectedFormat: "us-utah",
		},
		{
			name: "Iowa content not detected as any other state",
			content: `IOWA CONSUMER DATA PROTECTION ACT
Iowa Code Chapter 715D
Section 715D.1
Definitions`,
			expectedFormat: "us-iowa",
		},
		{
			name: "California content unchanged by new patterns",
			content: `CALIFORNIA CONSUMER PRIVACY ACT OF 2018
TITLE 1.81.5. CALIFORNIA CONSUMER PRIVACY ACT OF 2018
Section 1798.100`,
			expectedFormat: "us-california",
		},
		{
			name: "Virginia content unchanged by new patterns",
			content: `VIRGINIA CONSUMER DATA PROTECTION ACT
Code of Virginia Title 59.1 Chapter 53
Section 59.1-575`,
			expectedFormat: "us-virginia",
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

				allMatches := detector.DetectAll(tc.content)
				for _, m := range allMatches {
					t.Logf("  %s: %.1f%% confidence", m.FormatID, m.Confidence*100)
				}
			}
		})
	}
}

// TestStatePrivacyDefinitions validates that definitions are correctly extracted
// from each state's privacy law.
func TestStatePrivacyDefinitions(t *testing.T) {
	registry := loadPatternRegistry(t)

	testCases := []struct {
		name           string
		loadFunc       func(t *testing.T) *os.File
		minDefinitions int
		expectedTerms  []string
	}{
		{
			name:           "Colorado CPA definitions",
			loadFunc:       loadCPAText,
			minDefinitions: 5,
			expectedTerms:  []string{"Consumer", "Controller", "Personal data"},
		},
		{
			name:           "Connecticut CTDPA definitions",
			loadFunc:       loadCTDPAText,
			minDefinitions: 5,
			expectedTerms:  []string{"Consumer", "Controller", "Personal data"},
		},
		{
			name:           "Texas TDPSA definitions",
			loadFunc:       loadTDPSAText,
			minDefinitions: 5,
			expectedTerms:  []string{"Consumer", "Controller", "Personal data"},
		},
		{
			name:           "Utah UCPA definitions",
			loadFunc:       loadUCPAText,
			minDefinitions: 5,
			expectedTerms:  []string{"Consumer", "Controller", "Personal data"},
		},
		{
			name:           "Iowa ICDPA definitions",
			loadFunc:       loadICDPAText,
			minDefinitions: 5,
			expectedTerms:  []string{"Consumer", "Controller", "Personal data"},
		},
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

			if len(doc.Definitions) < tc.minDefinitions {
				t.Errorf("Expected at least %d definitions, got %d", tc.minDefinitions, len(doc.Definitions))
			}

			t.Logf("  Definitions found (%d):", len(doc.Definitions))
			for _, def := range doc.Definitions {
				t.Logf("    %d. %s", def.Number, def.Term)
			}

			// Check that key terms are found
			foundTerms := make(map[string]bool)
			for _, def := range doc.Definitions {
				foundTerms[def.Term] = true
			}
			for _, expectedTerm := range tc.expectedTerms {
				if !foundTerms[expectedTerm] {
					t.Errorf("Expected definition for %q not found", expectedTerm)
				}
			}
		})
	}
}

// TestStatePrivacyNoRegressionCCPA ensures CCPA still parses correctly with new patterns loaded.
func TestStatePrivacyNoRegressionCCPA(t *testing.T) {
	registry := loadPatternRegistry(t)

	f := loadCCPAText(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	stats := doc.Statistics()

	if stats.Chapters < 6 {
		t.Errorf("CCPA regression: expected at least 6 chapters, got %d", stats.Chapters)
	}
	if stats.Articles < 10 {
		t.Errorf("CCPA regression: expected at least 10 articles, got %d", stats.Articles)
	}
	if stats.Definitions < 5 {
		t.Errorf("CCPA regression: expected at least 5 definitions, got %d", stats.Definitions)
	}
}

// TestStatePrivacyNoRegressionVCDPA ensures VCDPA still parses correctly with new patterns loaded.
func TestStatePrivacyNoRegressionVCDPA(t *testing.T) {
	registry := loadPatternRegistry(t)

	f := loadVCDPAText(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	stats := doc.Statistics()

	if stats.Chapters < 1 {
		t.Errorf("VCDPA regression: expected at least 1 chapter, got %d", stats.Chapters)
	}
	if stats.Articles < 3 {
		t.Errorf("VCDPA regression: expected at least 3 articles, got %d", stats.Articles)
	}
}
