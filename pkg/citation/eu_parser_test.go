package citation

import (
	"os"
	"testing"
)

func TestEUCitationParserName(t *testing.T) {
	parser := NewEUCitationParser()
	if parser.Name() != "EU Citation Parser" {
		t.Errorf("Expected 'EU Citation Parser', got %q", parser.Name())
	}
}

func TestEUCitationParserJurisdictions(t *testing.T) {
	parser := NewEUCitationParser()
	jurisdictions := parser.Jurisdictions()
	if len(jurisdictions) != 1 || jurisdictions[0] != "EU" {
		t.Errorf("Expected [EU], got %v", jurisdictions)
	}
}

func TestEUCitationParserImplementsInterface(t *testing.T) {
	// Compile-time check that EUCitationParser implements CitationParser.
	var _ CitationParser = (*EUCitationParser)(nil)
}

func TestEUCitationParserParseRegulations(t *testing.T) {
	parser := NewEUCitationParser()

	cases := []struct {
		name          string
		text          string
		expectedCount int
		expectedYear  string
		expectedNum   string
	}{
		{
			name:          "regulation_eu_year_number",
			text:          "pursuant to Regulation (EU) 2016/679",
			expectedCount: 1,
			expectedYear:  "2016",
			expectedNum:   "679",
		},
		{
			name:          "regulation_ec_no_number_year",
			text:          "under Regulation (EC) No 45/2001",
			expectedCount: 1,
			expectedYear:  "2001",
			expectedNum:   "45",
		},
		{
			name:          "multiple_regulations",
			text:          "Regulation (EU) 2016/679 and Regulation (EC) No 45/2001",
			expectedCount: 2,
		},
		{
			name:          "no_regulation",
			text:          "This is plain text without regulations",
			expectedCount: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			citations, err := parser.Parse(tc.text)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			regulationCitations := filterByType(citations, CitationTypeRegulation)
			if len(regulationCitations) != tc.expectedCount {
				t.Errorf("Expected %d regulation citations, got %d", tc.expectedCount, len(regulationCitations))
				for _, c := range citations {
					t.Logf("  Citation: type=%s raw=%q year=%s num=%s",
						c.Type, c.RawText, c.Components.DocYear, c.Components.DocNumber)
				}
			}

			if tc.expectedCount == 1 && len(regulationCitations) == 1 {
				citation := regulationCitations[0]
				if tc.expectedYear != "" && citation.Components.DocYear != tc.expectedYear {
					t.Errorf("DocYear: got %q, want %q", citation.Components.DocYear, tc.expectedYear)
				}
				if tc.expectedNum != "" && citation.Components.DocNumber != tc.expectedNum {
					t.Errorf("DocNumber: got %q, want %q", citation.Components.DocNumber, tc.expectedNum)
				}
				if citation.Jurisdiction != "EU" {
					t.Errorf("Jurisdiction: got %q, want 'EU'", citation.Jurisdiction)
				}
				if citation.Confidence != 1.0 {
					t.Errorf("Confidence: got %f, want 1.0", citation.Confidence)
				}
				if citation.Parser != "EU Citation Parser" {
					t.Errorf("Parser: got %q, want 'EU Citation Parser'", citation.Parser)
				}
			}
		})
	}
}

func TestEUCitationParserParseDirectives(t *testing.T) {
	parser := NewEUCitationParser()

	cases := []struct {
		name string
		text string
		year string
		num  string
	}{
		{
			name: "directive_year_slash_ec",
			text: "Directive 95/46/EC",
			year: "95",
			num:  "46",
		},
		{
			name: "directive_eu_year_number",
			text: "Directive (EU) 2016/680",
			year: "2016",
			num:  "680",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			citations, err := parser.Parse(tc.text)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			directiveCitations := filterByType(citations, CitationTypeDirective)
			if len(directiveCitations) != 1 {
				t.Fatalf("Expected 1 directive, got %d", len(directiveCitations))
			}

			citation := directiveCitations[0]
			if citation.Components.DocYear != tc.year {
				t.Errorf("DocYear: got %q, want %q", citation.Components.DocYear, tc.year)
			}
			if citation.Components.DocNumber != tc.num {
				t.Errorf("DocNumber: got %q, want %q", citation.Components.DocNumber, tc.num)
			}
		})
	}
}

func TestEUCitationParserParseDecisions(t *testing.T) {
	parser := NewEUCitationParser()

	citations, err := parser.Parse("Decision 2010/87/EU on standard contractual clauses")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	decisionCitations := filterByType(citations, CitationTypeDecision)
	if len(decisionCitations) != 1 {
		t.Fatalf("Expected 1 decision, got %d", len(decisionCitations))
	}

	citation := decisionCitations[0]
	if citation.Components.DocYear != "2010" {
		t.Errorf("DocYear: got %q, want '2010'", citation.Components.DocYear)
	}
	if citation.Components.DocNumber != "87" {
		t.Errorf("DocNumber: got %q, want '87'", citation.Components.DocNumber)
	}
}

func TestEUCitationParserParseTreaties(t *testing.T) {
	parser := NewEUCitationParser()

	cases := []struct {
		name         string
		text         string
		expectedDoc  string
	}{
		{
			name:        "TFEU_abbreviation",
			text:        "pursuant to Article 16 TFEU",
			expectedDoc: "TFEU",
		},
		{
			name:        "TEU_abbreviation",
			text:        "in accordance with TEU",
			expectedDoc: "TEU",
		},
		{
			name:        "full_treaty_name",
			text:        "Treaty on the Functioning of the European Union",
			expectedDoc: "TFEU",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			citations, err := parser.Parse(tc.text)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			treatyCitations := filterByType(citations, CitationTypeTreaty)
			if len(treatyCitations) < 1 {
				t.Fatalf("Expected at least 1 treaty citation, got %d", len(treatyCitations))
			}

			if treatyCitations[0].Document != tc.expectedDoc {
				t.Errorf("Document: got %q, want %q", treatyCitations[0].Document, tc.expectedDoc)
			}
		})
	}
}

func TestEUCitationParserParseArticleRefs(t *testing.T) {
	parser := NewEUCitationParser()

	cases := []struct {
		name            string
		text            string
		expectedCount   int
		checkSubdiv     string
		checkArticleNum int
		checkParaNum    int
		checkPointLtr   string
	}{
		{
			name:            "article_paren_with_point",
			text:            "Article 6(1)(a) provides",
			expectedCount:   1,
			checkSubdiv:     "Article 6(1)(a)",
			checkArticleNum: 6,
			checkParaNum:    1,
			checkPointLtr:   "a",
		},
		{
			name:            "article_paren_without_point",
			text:            "Article 6(1) requires",
			expectedCount:   1,
			checkSubdiv:     "Article 6(1)",
			checkArticleNum: 6,
			checkParaNum:    1,
		},
		{
			name:            "plain_article",
			text:            "Article 17 grants the right",
			expectedCount:   1,
			checkSubdiv:     "Article 17",
			checkArticleNum: 17,
		},
		{
			name:          "articles_range",
			text:          "Articles 13 to 22 apply",
			expectedCount: 2, // Start and end articles.
		},
		{
			name:          "articles_and",
			text:          "Articles 13 and 14 require",
			expectedCount: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			citations, err := parser.Parse(tc.text)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Filter to statute-type citations (article/chapter refs).
			articleCitations := filterStatutesWithArticle(citations)
			if len(articleCitations) != tc.expectedCount {
				t.Errorf("Expected %d article citations, got %d", tc.expectedCount, len(articleCitations))
				for _, c := range citations {
					t.Logf("  Citation: type=%s subdiv=%q art=%d para=%d point=%s",
						c.Type, c.Subdivision, c.Components.ArticleNumber,
						c.Components.ParagraphNumber, c.Components.PointLetter)
				}
				return
			}

			if tc.expectedCount == 1 && len(articleCitations) == 1 {
				citation := articleCitations[0]
				if tc.checkSubdiv != "" && citation.Subdivision != tc.checkSubdiv {
					t.Errorf("Subdivision: got %q, want %q", citation.Subdivision, tc.checkSubdiv)
				}
				if tc.checkArticleNum != 0 && citation.Components.ArticleNumber != tc.checkArticleNum {
					t.Errorf("ArticleNumber: got %d, want %d",
						citation.Components.ArticleNumber, tc.checkArticleNum)
				}
				if tc.checkParaNum != 0 && citation.Components.ParagraphNumber != tc.checkParaNum {
					t.Errorf("ParagraphNumber: got %d, want %d",
						citation.Components.ParagraphNumber, tc.checkParaNum)
				}
				if tc.checkPointLtr != "" && citation.Components.PointLetter != tc.checkPointLtr {
					t.Errorf("PointLetter: got %q, want %q",
						citation.Components.PointLetter, tc.checkPointLtr)
				}
			}
		})
	}
}

func TestEUCitationParserParseChapterRefs(t *testing.T) {
	parser := NewEUCitationParser()

	citations, err := parser.Parse("Chapter III sets out the rules")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	chapterCitations := filterByChapter(citations)
	if len(chapterCitations) != 1 {
		t.Fatalf("Expected 1 chapter citation, got %d", len(chapterCitations))
	}

	if chapterCitations[0].Components.ChapterNumber != "III" {
		t.Errorf("ChapterNumber: got %q, want 'III'", chapterCitations[0].Components.ChapterNumber)
	}
	if chapterCitations[0].Subdivision != "Chapter III" {
		t.Errorf("Subdivision: got %q, want 'Chapter III'", chapterCitations[0].Subdivision)
	}
}

func TestEUCitationParserNormalize(t *testing.T) {
	parser := NewEUCitationParser()

	cases := []struct {
		name     string
		citation *Citation
		expected string
	}{
		{
			name: "regulation",
			citation: &Citation{
				Type:       CitationTypeRegulation,
				Components: CitationComponents{DocYear: "2016", DocNumber: "679"},
			},
			expected: "Regulation (EU) 2016/679",
		},
		{
			name: "directive",
			citation: &Citation{
				Type:       CitationTypeDirective,
				Components: CitationComponents{DocYear: "95", DocNumber: "46"},
			},
			expected: "Directive 95/46/EC",
		},
		{
			name: "decision",
			citation: &Citation{
				Type:       CitationTypeDecision,
				Components: CitationComponents{DocYear: "2010", DocNumber: "87"},
			},
			expected: "Decision 2010/87/EU",
		},
		{
			name: "treaty",
			citation: &Citation{
				Type:     CitationTypeTreaty,
				Document: "TFEU",
			},
			expected: "TFEU",
		},
		{
			name: "article_subdivision",
			citation: &Citation{
				Type:        CitationTypeStatute,
				Subdivision: "Article 6(1)(a)",
			},
			expected: "Article 6(1)(a)",
		},
		{
			name: "fallback_raw_text",
			citation: &Citation{
				Type:    CitationTypeUnknown,
				RawText: "some ref",
			},
			expected: "some ref",
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

func TestEUCitationParserToURI(t *testing.T) {
	parser := NewEUCitationParser()

	cases := []struct {
		name        string
		citation    *Citation
		expectedURI string
		expectError bool
	}{
		{
			name: "regulation_uri",
			citation: &Citation{
				Type:       CitationTypeRegulation,
				Components: CitationComponents{DocYear: "2016", DocNumber: "679"},
			},
			expectedURI: "urn:eu:regulation:2016/679",
		},
		{
			name: "directive_uri",
			citation: &Citation{
				Type:       CitationTypeDirective,
				Components: CitationComponents{DocYear: "95", DocNumber: "46"},
			},
			expectedURI: "urn:eu:directive:95/46",
		},
		{
			name: "decision_uri",
			citation: &Citation{
				Type:       CitationTypeDecision,
				Components: CitationComponents{DocYear: "2010", DocNumber: "87"},
			},
			expectedURI: "urn:eu:decision:2010/87",
		},
		{
			name: "treaty_uri",
			citation: &Citation{
				Type:     CitationTypeTreaty,
				Document: "TFEU",
			},
			expectedURI: "urn:eu:treaty:TFEU",
		},
		{
			name: "regulation_missing_year",
			citation: &Citation{
				Type:       CitationTypeRegulation,
				Components: CitationComponents{DocNumber: "679"},
			},
			expectError: true,
		},
		{
			name: "directive_missing_number",
			citation: &Citation{
				Type:       CitationTypeDirective,
				Components: CitationComponents{DocYear: "95"},
			},
			expectError: true,
		},
		{
			name: "unsupported_type",
			citation: &Citation{
				Type: CitationTypeCase,
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

func TestEUCitationParserGDPRIntegration(t *testing.T) {
	gdprPath := "../../testdata/gdpr.txt"
	gdprData, err := os.ReadFile(gdprPath)
	if err != nil {
		t.Skipf("Skipping GDPR integration test: %v", err)
	}
	gdprText := string(gdprData)

	parser := NewEUCitationParser()
	citations, err := parser.Parse(gdprText)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	t.Logf("Total citations found: %d", len(citations))

	// Count by type.
	typeCounts := make(map[CitationType]int)
	for _, c := range citations {
		typeCounts[c.Type]++
	}
	for citType, count := range typeCounts {
		t.Logf("  %s: %d", citType, count)
	}

	// GDPR should have substantial citation counts.
	if len(citations) < 50 {
		t.Errorf("Expected at least 50 total citations from GDPR, got %d", len(citations))
	}

	// Should find regulations (at minimum, self-references to GDPR).
	regulationCount := typeCounts[CitationTypeRegulation]
	if regulationCount < 1 {
		t.Errorf("Expected at least 1 regulation citation, got %d", regulationCount)
	}

	// Should find directives (e.g., Directive 95/46/EC).
	directiveCount := typeCounts[CitationTypeDirective]
	if directiveCount < 1 {
		t.Errorf("Expected at least 1 directive citation, got %d", directiveCount)
	}

	// Should find article references.
	statuteCount := typeCounts[CitationTypeStatute]
	if statuteCount < 30 {
		t.Errorf("Expected at least 30 article/statute citations, got %d", statuteCount)
	}

	// Verify all citations have valid metadata.
	for i, c := range citations {
		if c.RawText == "" {
			t.Errorf("Citation %d has empty RawText", i)
		}
		if c.Parser != "EU Citation Parser" {
			t.Errorf("Citation %d has wrong Parser: %q", i, c.Parser)
		}
		if c.Confidence <= 0 {
			t.Errorf("Citation %d has non-positive Confidence: %f", i, c.Confidence)
		}
	}
}

// Test helpers.

func filterByType(citations []*Citation, citationType CitationType) []*Citation {
	var filtered []*Citation
	for _, c := range citations {
		if c.Type == citationType {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func filterStatutesWithArticle(citations []*Citation) []*Citation {
	var filtered []*Citation
	for _, c := range citations {
		if c.Type == CitationTypeStatute && c.Components.ArticleNumber > 0 {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func filterByChapter(citations []*Citation) []*Citation {
	var filtered []*Citation
	for _, c := range citations {
		if c.Components.ChapterNumber != "" {
			filtered = append(filtered, c)
		}
	}
	return filtered
}
