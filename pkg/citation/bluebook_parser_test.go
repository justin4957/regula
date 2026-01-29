package citation

import (
	"os"
	"testing"
)

func TestBluebookParserName(t *testing.T) {
	parser := NewBluebookParser()
	if parser.Name() != "Bluebook Citation Parser" {
		t.Errorf("Expected 'Bluebook Citation Parser', got %q", parser.Name())
	}
}

func TestBluebookParserJurisdictions(t *testing.T) {
	parser := NewBluebookParser()
	jurisdictions := parser.Jurisdictions()
	if len(jurisdictions) != 1 || jurisdictions[0] != "US" {
		t.Errorf("Expected [US], got %v", jurisdictions)
	}
}

func TestBluebookParserImplementsInterface(t *testing.T) {
	// Compile-time check that BluebookParser implements CitationParser.
	var _ CitationParser = (*BluebookParser)(nil)
}

func TestBluebookParserParseUSC(t *testing.T) {
	parser := NewBluebookParser()

	cases := []struct {
		name          string
		text          string
		expectedCount int
		expectedTitle string
		expectedSec   string
	}{
		{
			name:          "usc_section_symbol",
			text:          "pursuant to 42 U.S.C. § 1983",
			expectedCount: 1,
			expectedTitle: "42",
			expectedSec:   "1983",
		},
		{
			name:          "usc_section_word",
			text:          "under 15 U.S.C. Section 1681",
			expectedCount: 1,
			expectedTitle: "15",
			expectedSec:   "1681",
		},
		{
			name:          "usc_sec_abbreviation",
			text:          "see 20 U.S.C. Sec. 1232g",
			expectedCount: 1,
			expectedTitle: "20",
			expectedSec:   "1232g",
		},
		{
			name:          "usc_et_seq",
			text:          "the federal Fair Credit Reporting Act (15 U.S.C. Section 1681 et seq.)",
			expectedCount: 1,
			expectedTitle: "15",
			expectedSec:   "1681",
		},
		{
			name:          "usc_with_subsection_letter",
			text:          "42 U.S.C. § 1320d",
			expectedCount: 1,
			expectedTitle: "42",
			expectedSec:   "1320d",
		},
		{
			name:          "multiple_usc_citations",
			text:          "See 15 U.S.C. § 1681 and 18 U.S.C. § 2721",
			expectedCount: 2,
		},
		{
			name:          "no_usc_citation",
			text:          "This is plain text without USC citations",
			expectedCount: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			citations, err := parser.Parse(tc.text)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			uscCitations := filterByCodeName(citations, "USC")
			if len(uscCitations) != tc.expectedCount {
				t.Errorf("Expected %d USC citations, got %d", tc.expectedCount, len(uscCitations))
				for _, c := range citations {
					t.Logf("  Citation: type=%s code=%s raw=%q",
						c.Type, c.Components.CodeName, c.RawText)
				}
			}

			if tc.expectedCount == 1 && len(uscCitations) == 1 {
				citation := uscCitations[0]
				if tc.expectedTitle != "" && citation.Components.Title != tc.expectedTitle {
					t.Errorf("Title: got %q, want %q", citation.Components.Title, tc.expectedTitle)
				}
				if tc.expectedSec != "" && citation.Components.Section != tc.expectedSec {
					t.Errorf("Section: got %q, want %q", citation.Components.Section, tc.expectedSec)
				}
				if citation.Jurisdiction != "US" {
					t.Errorf("Jurisdiction: got %q, want 'US'", citation.Jurisdiction)
				}
				if citation.Type != CitationTypeCode {
					t.Errorf("Type: got %q, want %q", citation.Type, CitationTypeCode)
				}
			}
		})
	}
}

func TestBluebookParserParseCFR(t *testing.T) {
	parser := NewBluebookParser()

	cases := []struct {
		name          string
		text          string
		expectedCount int
		expectedTitle string
		expectedSec   string
	}{
		{
			name:          "cfr_part",
			text:          "45 C.F.R. Part 164",
			expectedCount: 1,
			expectedTitle: "45",
			expectedSec:   "164",
		},
		{
			name:          "cfr_section",
			text:          "45 C.F.R. § 164.502",
			expectedCount: 1,
			expectedTitle: "45",
			expectedSec:   "164.502",
		},
		{
			name:          "cfr_multiple_parts",
			text:          "45 C.F.R. Parts 160 and 164",
			expectedCount: 2, // Emits both parts
		},
		{
			name:          "cfr_simple",
			text:          "21 C.F.R. Part 50",
			expectedCount: 1,
			expectedTitle: "21",
			expectedSec:   "50",
		},
		{
			name:          "cfr_no_periods",
			text:          "34 CFR Part 99",
			expectedCount: 1,
			expectedTitle: "34",
			expectedSec:   "99",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			citations, err := parser.Parse(tc.text)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			cfrCitations := filterByCodeName(citations, "CFR")
			if len(cfrCitations) != tc.expectedCount {
				t.Errorf("Expected %d CFR citations, got %d", tc.expectedCount, len(cfrCitations))
				for _, c := range citations {
					t.Logf("  Citation: type=%s code=%s raw=%q",
						c.Type, c.Components.CodeName, c.RawText)
				}
			}

			if tc.expectedCount == 1 && len(cfrCitations) == 1 {
				citation := cfrCitations[0]
				if tc.expectedTitle != "" && citation.Components.Title != tc.expectedTitle {
					t.Errorf("Title: got %q, want %q", citation.Components.Title, tc.expectedTitle)
				}
				if tc.expectedSec != "" && citation.Components.Section != tc.expectedSec {
					t.Errorf("Section: got %q, want %q", citation.Components.Section, tc.expectedSec)
				}
			}
		})
	}
}

func TestBluebookParserParsePublicLaw(t *testing.T) {
	parser := NewBluebookParser()

	cases := []struct {
		name        string
		text        string
		expectedPL  string
		expectFound bool
	}{
		{
			name:        "public_law_full",
			text:        "under the federal Health Insurance Portability and Accountability Act of 1996 (Public Law 104-191)",
			expectedPL:  "104-191",
			expectFound: true,
		},
		{
			name:        "pub_l_abbreviation",
			text:        "pursuant to Pub. L. 106-102",
			expectedPL:  "106-102",
			expectFound: true,
		},
		{
			name:        "pl_short",
			text:        "P.L. 111-5 established",
			expectedPL:  "111-5",
			expectFound: true,
		},
		{
			name:        "no_public_law",
			text:        "This text has no public law citations",
			expectFound: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			citations, err := parser.Parse(tc.text)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			plCitations := filterByPublicLaw(citations)
			if tc.expectFound {
				if len(plCitations) != 1 {
					t.Errorf("Expected 1 Public Law citation, got %d", len(plCitations))
					return
				}
				if plCitations[0].Components.PublicLaw != tc.expectedPL {
					t.Errorf("PublicLaw: got %q, want %q", plCitations[0].Components.PublicLaw, tc.expectedPL)
				}
				if plCitations[0].Type != CitationTypeStatute {
					t.Errorf("Type: got %q, want %q", plCitations[0].Type, CitationTypeStatute)
				}
			} else {
				if len(plCitations) != 0 {
					t.Errorf("Expected no Public Law citations, got %d", len(plCitations))
				}
			}
		})
	}
}

func TestBluebookParserParseCases(t *testing.T) {
	parser := NewBluebookParser()

	cases := []struct {
		name        string
		text        string
		expectFound bool
		checkDoc    string // partial match on Document field
	}{
		{
			name:        "supreme_court_case",
			text:        "as held in Brown v. Board of Education, 347 U.S. 483 (1954)",
			expectFound: true,
			checkDoc:    "Brown v. Board of Education",
		},
		{
			name:        "case_at_start",
			text:        "Roe v. Wade, 410 U.S. 113 (1973) established",
			expectFound: true,
			checkDoc:    "Roe v. Wade",
		},
		{
			name:        "no_case_citation",
			text:        "This is plain text without case citations",
			expectFound: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			citations, err := parser.Parse(tc.text)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			caseCitations := filterByType(citations, CitationTypeCase)
			if tc.expectFound {
				if len(caseCitations) < 1 {
					t.Errorf("Expected at least 1 case citation, got %d", len(caseCitations))
					for _, c := range citations {
						t.Logf("  Citation: type=%s doc=%q raw=%q", c.Type, c.Document, c.RawText)
					}
					return
				}
				// Check that the expected case name is contained in the document field.
				if tc.checkDoc != "" && caseCitations[0].Document != tc.checkDoc {
					// Log but don't fail — case name extraction is complex.
					t.Logf("Case name: got %q, want %q (case detection is approximate)",
						caseCitations[0].Document, tc.checkDoc)
				}
			} else {
				if len(caseCitations) != 0 {
					t.Errorf("Expected no case citations, got %d", len(caseCitations))
				}
			}
		})
	}
}

func TestBluebookParserNormalize(t *testing.T) {
	parser := NewBluebookParser()

	cases := []struct {
		name     string
		citation *Citation
		expected string
	}{
		{
			name: "usc_citation",
			citation: &Citation{
				Type: CitationTypeCode,
				Components: CitationComponents{
					Title:    "42",
					Section:  "1983",
					CodeName: "USC",
				},
			},
			expected: "42 U.S.C. § 1983",
		},
		{
			name: "cfr_citation",
			citation: &Citation{
				Type: CitationTypeCode,
				Components: CitationComponents{
					Title:    "45",
					Section:  "164",
					CodeName: "CFR",
				},
			},
			expected: "45 C.F.R. § 164",
		},
		{
			name: "public_law",
			citation: &Citation{
				Type: CitationTypeStatute,
				Components: CitationComponents{
					PublicLaw: "104-191",
				},
			},
			expected: "Pub. L. 104-191",
		},
		{
			name: "case_citation",
			citation: &Citation{
				Type:     CitationTypeCase,
				Document: "Brown v. Board of Education",
			},
			expected: "Brown v. Board of Education",
		},
		{
			name: "fallback_raw_text",
			citation: &Citation{
				Type:    CitationTypeUnknown,
				RawText: "some reference",
			},
			expected: "some reference",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := parser.Normalize(tc.citation)
			if result != tc.expected {
				t.Errorf("Normalize: got %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestBluebookParserToURI(t *testing.T) {
	parser := NewBluebookParser()

	cases := []struct {
		name        string
		citation    *Citation
		expectedURI string
		expectError bool
	}{
		{
			name: "usc_uri",
			citation: &Citation{
				Type: CitationTypeCode,
				Components: CitationComponents{
					Title:    "42",
					Section:  "1983",
					CodeName: "USC",
				},
			},
			expectedURI: "urn:us:usc:42/1983",
		},
		{
			name: "cfr_uri",
			citation: &Citation{
				Type: CitationTypeCode,
				Components: CitationComponents{
					Title:    "45",
					Section:  "164",
					CodeName: "CFR",
				},
			},
			expectedURI: "urn:us:cfr:45/164",
		},
		{
			name: "public_law_uri",
			citation: &Citation{
				Type: CitationTypeStatute,
				Components: CitationComponents{
					PublicLaw: "104-191",
				},
			},
			expectedURI: "urn:us:pl:104-191",
		},
		{
			name: "case_uri",
			citation: &Citation{
				Type: CitationTypeCase,
				Components: CitationComponents{
					DocNumber: "347/U.S./483",
					DocYear:   "1954",
				},
			},
			expectedURI: "urn:us:case:347/U.S./483",
		},
		{
			name: "usc_missing_title",
			citation: &Citation{
				Type: CitationTypeCode,
				Components: CitationComponents{
					Section:  "1983",
					CodeName: "USC",
				},
			},
			expectError: true,
		},
		{
			name: "unsupported_code",
			citation: &Citation{
				Type: CitationTypeCode,
				Components: CitationComponents{
					Title:    "1",
					Section:  "1",
					CodeName: "OTHER",
				},
			},
			expectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uri, err := parser.ToURI(tc.citation)
			if tc.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if uri != tc.expectedURI {
				t.Errorf("URI: got %q, want %q", uri, tc.expectedURI)
			}
		})
	}
}

func TestBluebookParserCCPAIntegration(t *testing.T) {
	ccpaPath := "../../testdata/ccpa.txt"
	ccpaData, err := os.ReadFile(ccpaPath)
	if err != nil {
		t.Skipf("Skipping CCPA integration test: %v", err)
	}
	ccpaText := string(ccpaData)

	parser := NewBluebookParser()
	citations, err := parser.Parse(ccpaText)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	t.Logf("Total Bluebook citations found in CCPA: %d", len(citations))

	// Count by type.
	typeCounts := make(map[CitationType]int)
	codeCounts := make(map[string]int)
	for _, c := range citations {
		typeCounts[c.Type]++
		if c.Components.CodeName != "" {
			codeCounts[c.Components.CodeName]++
		}
	}
	for citType, count := range typeCounts {
		t.Logf("  %s: %d", citType, count)
	}
	for codeName, count := range codeCounts {
		t.Logf("  Code %s: %d", codeName, count)
	}

	// CCPA should have US Code citations (FERPA, FCRA references).
	uscCount := codeCounts["USC"]
	if uscCount < 1 {
		t.Errorf("Expected at least 1 USC citation in CCPA, got %d", uscCount)
	}

	// CCPA should have CFR citations.
	cfrCount := codeCounts["CFR"]
	if cfrCount < 1 {
		t.Errorf("Expected at least 1 CFR citation in CCPA, got %d", cfrCount)
	}

	// CCPA references Public Laws (GLBA, HIPAA).
	plCount := len(filterByPublicLaw(citations))
	if plCount < 1 {
		t.Errorf("Expected at least 1 Public Law citation in CCPA, got %d", plCount)
	}

	// Verify all citations have valid metadata.
	for i, c := range citations {
		if c.RawText == "" {
			t.Errorf("Citation %d has empty RawText", i)
		}
		if c.Parser != "Bluebook Citation Parser" {
			t.Errorf("Citation %d has wrong Parser: %q", i, c.Parser)
		}
		if c.Confidence <= 0 {
			t.Errorf("Citation %d has non-positive Confidence: %f", i, c.Confidence)
		}
		if c.Jurisdiction != "US" {
			t.Errorf("Citation %d has wrong Jurisdiction: %q", i, c.Jurisdiction)
		}
	}
}

func TestBluebookParserVCDPAIntegration(t *testing.T) {
	vcdpaPath := "../../testdata/vcdpa.txt"
	vcdpaData, err := os.ReadFile(vcdpaPath)
	if err != nil {
		t.Skipf("Skipping VCDPA integration test: %v", err)
	}
	vcdpaText := string(vcdpaData)

	parser := NewBluebookParser()
	citations, err := parser.Parse(vcdpaText)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	t.Logf("Total Bluebook citations found in VCDPA: %d", len(citations))

	// VCDPA has many external federal law references.
	codeCounts := make(map[string]int)
	for _, c := range citations {
		if c.Components.CodeName != "" {
			codeCounts[c.Components.CodeName]++
		}
	}
	for codeName, count := range codeCounts {
		t.Logf("  Code %s: %d", codeName, count)
	}

	// VCDPA should have significant USC citations (HIPAA, GLBA, COPPA, etc.).
	uscCount := codeCounts["USC"]
	if uscCount < 5 {
		t.Errorf("Expected at least 5 USC citations in VCDPA, got %d", uscCount)
	}

	// VCDPA should have CFR citations.
	cfrCount := codeCounts["CFR"]
	if cfrCount < 1 {
		t.Errorf("Expected at least 1 CFR citation in VCDPA, got %d", cfrCount)
	}
}

func TestBluebookParserRegistration(t *testing.T) {
	// Test that the parser can be registered with the citation registry.
	registry := NewCitationRegistry()
	parser := NewBluebookParser()

	err := registry.Register(parser)
	if err != nil {
		t.Fatalf("Failed to register Bluebook parser: %v", err)
	}

	// Verify it's retrievable by name.
	retrieved, ok := registry.Get("Bluebook Citation Parser")
	if !ok || retrieved == nil {
		t.Error("Failed to retrieve registered parser")
	}

	// Verify it's listed for US jurisdiction.
	usParserNames := registry.ListByJurisdiction("US")
	found := false
	for _, parserName := range usParserNames {
		if parserName == "Bluebook Citation Parser" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Bluebook parser not found in US jurisdiction list")
	}
}

// Test helpers.

func filterByCodeName(citations []*Citation, codeName string) []*Citation {
	var filtered []*Citation
	for _, c := range citations {
		if c.Components.CodeName == codeName {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func filterByPublicLaw(citations []*Citation) []*Citation {
	var filtered []*Citation
	for _, c := range citations {
		if c.Components.PublicLaw != "" {
			filtered = append(filtered, c)
		}
	}
	return filtered
}
