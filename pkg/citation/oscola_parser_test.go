package citation

import (
	"os"
	"testing"
)

func TestOSCOLAParserName(t *testing.T) {
	parser := NewOSCOLAParser()
	if parser.Name() != "OSCOLA Citation Parser" {
		t.Errorf("Expected 'OSCOLA Citation Parser', got %q", parser.Name())
	}
}

func TestOSCOLAParserJurisdictions(t *testing.T) {
	parser := NewOSCOLAParser()
	jurisdictions := parser.Jurisdictions()
	if len(jurisdictions) != 1 || jurisdictions[0] != "UK" {
		t.Errorf("Expected [UK], got %v", jurisdictions)
	}
}

func TestOSCOLAParserImplementsInterface(t *testing.T) {
	// Compile-time check that OSCOLAParser implements CitationParser.
	var _ CitationParser = (*OSCOLAParser)(nil)
}

func TestOSCOLAParserParseActs(t *testing.T) {
	parser := NewOSCOLAParser()

	cases := []struct {
		name         string
		text         string
		expectedCount int
		expectedDoc  string
		expectedYear string
	}{
		{
			name:         "data_protection_act",
			text:         "Data Protection Act 2018",
			expectedCount: 1,
			expectedDoc:  "Data Protection Act 2018",
			expectedYear: "2018",
		},
		{
			name:         "human_rights_act",
			text:         "the Human Rights Act 1998",
			expectedCount: 1,
			expectedDoc:  "Human Rights Act 1998",
			expectedYear: "1998",
		},
		{
			name:         "eu_withdrawal_act",
			text:         "the European Union (Withdrawal) Act 2018",
			expectedCount: 1,
			expectedYear: "2018",
		},
		{
			name:         "with_the_prefix",
			text:         "pursuant to the Data Protection Act 2018",
			expectedCount: 1,
			expectedDoc:  "Data Protection Act 2018",
			expectedYear: "2018",
		},
		{
			name:         "multiple_acts",
			text:         "Data Protection Act 2018 and the Human Rights Act 1998",
			expectedCount: 2,
		},
		{
			name:         "act_with_chapter",
			text:         "Data Protection Act 2018 [2018 c. 12]",
			expectedCount: 2, // One for act name, one for chapter ref
		},
		{
			name:         "no_act_citation",
			text:         "This is plain text without Act citations",
			expectedCount: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			citations, err := parser.Parse(tc.text)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			actCitations := filterByCodeNameOSCOLA(citations, "ukact")
			if len(actCitations) != tc.expectedCount {
				t.Errorf("Expected %d act citations, got %d", tc.expectedCount, len(actCitations))
				for _, c := range citations {
					t.Logf("  Citation: type=%s code=%s doc=%q raw=%q",
						c.Type, c.Components.CodeName, c.Document, c.RawText)
				}
			}

			if tc.expectedCount == 1 && len(actCitations) == 1 {
				citation := actCitations[0]
				if tc.expectedDoc != "" && citation.Document != tc.expectedDoc {
					t.Errorf("Document: got %q, want %q", citation.Document, tc.expectedDoc)
				}
				if tc.expectedYear != "" && citation.Components.DocYear != tc.expectedYear {
					t.Errorf("Year: got %q, want %q", citation.Components.DocYear, tc.expectedYear)
				}
				if citation.Jurisdiction != "UK" {
					t.Errorf("Jurisdiction: got %q, want 'UK'", citation.Jurisdiction)
				}
				if citation.Type != CitationTypeStatute {
					t.Errorf("Type: got %q, want %q", citation.Type, CitationTypeStatute)
				}
			}
		})
	}
}

func TestOSCOLAParserParseActChapterRefs(t *testing.T) {
	parser := NewOSCOLAParser()

	cases := []struct {
		name           string
		text           string
		expectedCount  int
		expectedYear   string
		expectedNumber string
	}{
		{
			name:           "standard_chapter",
			text:           "[2018 c. 12]",
			expectedCount:  1,
			expectedYear:   "2018",
			expectedNumber: "12",
		},
		{
			name:           "different_chapter",
			text:           "[2018 c. 16]",
			expectedCount:  1,
			expectedYear:   "2018",
			expectedNumber: "16",
		},
		{
			name:          "no_chapter_ref",
			text:          "This text has no chapter references",
			expectedCount: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			citations, err := parser.Parse(tc.text)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Filter for chapter-specific citations (ukact with DocNumber set).
			var chapterCitations []*Citation
			for _, c := range citations {
				if c.Components.CodeName == "ukact" && c.Components.DocNumber != "" {
					chapterCitations = append(chapterCitations, c)
				}
			}

			if len(chapterCitations) != tc.expectedCount {
				t.Errorf("Expected %d chapter citations, got %d", tc.expectedCount, len(chapterCitations))
			}

			if tc.expectedCount == 1 && len(chapterCitations) == 1 {
				citation := chapterCitations[0]
				if citation.Components.DocYear != tc.expectedYear {
					t.Errorf("Year: got %q, want %q", citation.Components.DocYear, tc.expectedYear)
				}
				if citation.Components.DocNumber != tc.expectedNumber {
					t.Errorf("Number: got %q, want %q", citation.Components.DocNumber, tc.expectedNumber)
				}
			}
		})
	}
}

func TestOSCOLAParserParseSI(t *testing.T) {
	parser := NewOSCOLAParser()

	cases := []struct {
		name           string
		text           string
		expectedCount  int
		expectedYear   string
		expectedNumber string
	}{
		{
			name:           "si_short",
			text:           "SI 2019/419",
			expectedCount:  1,
			expectedYear:   "2019",
			expectedNumber: "419",
		},
		{
			name:           "si_long",
			text:           "Statutory Instruments 2019 No. 419",
			expectedCount:  1,
			expectedYear:   "2019",
			expectedNumber: "419",
		},
		{
			name:           "si_singular",
			text:           "Statutory Instrument 2018 No. 1400",
			expectedCount:  1,
			expectedYear:   "2018",
			expectedNumber: "1400",
		},
		{
			name:         "multiple_si",
			text:         "SI 2019/419 and SI 2018/1400",
			expectedCount: 2,
		},
		{
			name:          "no_si",
			text:          "This text has no statutory instruments",
			expectedCount: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			citations, err := parser.Parse(tc.text)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			siCitations := filterByType(citations, CitationTypeRegulation)
			if len(siCitations) != tc.expectedCount {
				t.Errorf("Expected %d SI citations, got %d", tc.expectedCount, len(siCitations))
				for _, c := range citations {
					t.Logf("  Citation: type=%s doc=%q raw=%q", c.Type, c.Document, c.RawText)
				}
			}

			if tc.expectedCount == 1 && len(siCitations) == 1 {
				citation := siCitations[0]
				if tc.expectedYear != "" && citation.Components.DocYear != tc.expectedYear {
					t.Errorf("Year: got %q, want %q", citation.Components.DocYear, tc.expectedYear)
				}
				if tc.expectedNumber != "" && citation.Components.DocNumber != tc.expectedNumber {
					t.Errorf("Number: got %q, want %q", citation.Components.DocNumber, tc.expectedNumber)
				}
				if citation.Jurisdiction != "UK" {
					t.Errorf("Jurisdiction: got %q, want 'UK'", citation.Jurisdiction)
				}
			}
		})
	}
}

func TestOSCOLAParserParseSections(t *testing.T) {
	parser := NewOSCOLAParser()

	cases := []struct {
		name            string
		text            string
		expectedCount   int
		expectedSection string
		expectedSubsec  int
		expectedPoint   string
	}{
		{
			name:            "section_abbreviation",
			text:            "see s 6 of the Act",
			expectedCount:   1,
			expectedSection: "6",
		},
		{
			name:            "section_with_subsection",
			text:            "under s 6(1)",
			expectedCount:   1,
			expectedSection: "6",
			expectedSubsec:  1,
		},
		{
			name:            "section_with_paragraph",
			text:            "pursuant to s 8(2)(a)",
			expectedCount:   1,
			expectedSection: "8",
			expectedSubsec:  2,
			expectedPoint:   "a",
		},
		{
			name:            "section_word_form",
			text:            "section 114 provides",
			expectedCount:   1,
			expectedSection: "114",
		},
		{
			name:            "section_word_with_subsection",
			text:            "section 21(1) defines",
			expectedCount:   1,
			expectedSection: "21",
			expectedSubsec:  1,
		},
		{
			name:          "multiple_sections",
			text:          "see s 6(1) and section 17(2)",
			expectedCount: 2,
		},
		{
			name:          "no_section",
			text:          "This text has no section references",
			expectedCount: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			citations, err := parser.Parse(tc.text)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Filter for section citations (have Section component set).
			sectionCitations := filterBySectionComponent(citations)
			if len(sectionCitations) != tc.expectedCount {
				t.Errorf("Expected %d section citations, got %d", tc.expectedCount, len(sectionCitations))
				for _, c := range citations {
					t.Logf("  Citation: type=%s section=%s raw=%q",
						c.Type, c.Components.Section, c.RawText)
				}
			}

			if tc.expectedCount == 1 && len(sectionCitations) == 1 {
				citation := sectionCitations[0]
				if tc.expectedSection != "" && citation.Components.Section != tc.expectedSection {
					t.Errorf("Section: got %q, want %q", citation.Components.Section, tc.expectedSection)
				}
				if tc.expectedSubsec > 0 && citation.Components.ParagraphNumber != tc.expectedSubsec {
					t.Errorf("Subsection: got %d, want %d", citation.Components.ParagraphNumber, tc.expectedSubsec)
				}
				if tc.expectedPoint != "" && citation.Components.PointLetter != tc.expectedPoint {
					t.Errorf("Point: got %q, want %q", citation.Components.PointLetter, tc.expectedPoint)
				}
			}
		})
	}
}

func TestOSCOLAParserParseCases(t *testing.T) {
	parser := NewOSCOLAParser()

	cases := []struct {
		name        string
		text        string
		expectFound bool
		checkDoc    string
	}{
		{
			name:        "neutral_supreme_court",
			text:        "[2019] UKSC 5",
			expectFound: true,
			checkDoc:    "[2019] UKSC 5",
		},
		{
			name:        "neutral_court_of_appeal_civil",
			text:        "[2021] EWCA Civ 1234",
			expectFound: true,
			checkDoc:    "[2021] EWCA Civ 1234",
		},
		{
			name:        "neutral_court_of_appeal_criminal",
			text:        "[2020] EWCA Crim 456",
			expectFound: true,
			checkDoc:    "[2020] EWCA Crim 456",
		},
		{
			name:        "neutral_high_court",
			text:        "[2022] EWHC 789",
			expectFound: true,
			checkDoc:    "[2022] EWHC 789",
		},
		{
			name:        "neutral_house_of_lords",
			text:        "[2004] UKHL 56",
			expectFound: true,
			checkDoc:    "[2004] UKHL 56",
		},
		{
			name:        "law_report_ac",
			text:        "[1994] 1 AC 212",
			expectFound: true,
		},
		{
			name:        "law_report_qb",
			text:        "[2003] QB 195",
			expectFound: true,
		},
		{
			name:        "no_case",
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
				if tc.checkDoc != "" && caseCitations[0].Document != tc.checkDoc {
					t.Errorf("Document: got %q, want %q", caseCitations[0].Document, tc.checkDoc)
				}
			} else {
				if len(caseCitations) != 0 {
					t.Errorf("Expected no case citations, got %d", len(caseCitations))
				}
			}
		})
	}
}

func TestOSCOLAParserParseECHR(t *testing.T) {
	parser := NewOSCOLAParser()

	cases := []struct {
		name            string
		text            string
		expectedCount   int
		expectedArticle int
		expectedPara    int
	}{
		{
			name:            "echr_art",
			text:            "ECHR art 8",
			expectedCount:   1,
			expectedArticle: 8,
		},
		{
			name:            "echr_article",
			text:            "ECHR article 6",
			expectedCount:   1,
			expectedArticle: 6,
		},
		{
			name:            "echr_with_paragraph",
			text:            "ECHR art 6(1)",
			expectedCount:   1,
			expectedArticle: 6,
			expectedPara:    1,
		},
		{
			name:          "multiple_echr",
			text:          "ECHR art 8 and ECHR art 10",
			expectedCount: 2,
		},
		{
			name:          "no_echr",
			text:          "This text has no ECHR references",
			expectedCount: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			citations, err := parser.Parse(tc.text)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			echrCitations := filterByType(citations, CitationTypeTreaty)
			if len(echrCitations) != tc.expectedCount {
				t.Errorf("Expected %d ECHR citations, got %d", tc.expectedCount, len(echrCitations))
			}

			if tc.expectedCount == 1 && len(echrCitations) == 1 {
				citation := echrCitations[0]
				if citation.Components.ArticleNumber != tc.expectedArticle {
					t.Errorf("Article: got %d, want %d", citation.Components.ArticleNumber, tc.expectedArticle)
				}
				if tc.expectedPara > 0 && citation.Components.ParagraphNumber != tc.expectedPara {
					t.Errorf("Paragraph: got %d, want %d", citation.Components.ParagraphNumber, tc.expectedPara)
				}
				if citation.Document != "ECHR" {
					t.Errorf("Document: got %q, want 'ECHR'", citation.Document)
				}
			}
		})
	}
}

func TestOSCOLAParserParseSchedules(t *testing.T) {
	parser := NewOSCOLAParser()

	cases := []struct {
		name             string
		text             string
		expectedCount    int
		expectedSchedule string
	}{
		{
			name:             "schedule_7",
			text:             "Schedule 7 to the Act",
			expectedCount:    1,
			expectedSchedule: "7",
		},
		{
			name:             "schedule_12",
			text:             "Schedule 12 makes provision",
			expectedCount:    1,
			expectedSchedule: "12",
		},
		{
			name:          "multiple_schedules",
			text:          "Schedule 1 and Schedule 2",
			expectedCount: 2,
		},
		{
			name:          "no_schedule",
			text:          "This text has no schedule references",
			expectedCount: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			citations, err := parser.Parse(tc.text)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			scheduleCitations := filterBySubdivisionPrefix(citations, "Schedule")
			if len(scheduleCitations) != tc.expectedCount {
				t.Errorf("Expected %d schedule citations, got %d", tc.expectedCount, len(scheduleCitations))
			}

			if tc.expectedCount == 1 && len(scheduleCitations) == 1 {
				if tc.expectedSchedule != "" && scheduleCitations[0].Components.ChapterNumber != tc.expectedSchedule {
					t.Errorf("Schedule: got %q, want %q",
						scheduleCitations[0].Components.ChapterNumber, tc.expectedSchedule)
				}
			}
		})
	}
}

func TestOSCOLAParserParseParts(t *testing.T) {
	parser := NewOSCOLAParser()

	cases := []struct {
		name         string
		text         string
		expectedCount int
		expectedPart string
	}{
		{
			name:         "part_3",
			text:         "Part 3 makes provision",
			expectedCount: 1,
			expectedPart: "3",
		},
		{
			name:         "part_7",
			text:         "Part 7 supplementary",
			expectedCount: 1,
			expectedPart: "7",
		},
		{
			name:          "multiple_parts",
			text:          "Part 1 and Part 2",
			expectedCount: 2,
		},
		{
			name:          "no_part",
			text:          "This text has no part references",
			expectedCount: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			citations, err := parser.Parse(tc.text)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			partCitations := filterBySubdivisionPrefix(citations, "Part")
			if len(partCitations) != tc.expectedCount {
				t.Errorf("Expected %d part citations, got %d", tc.expectedCount, len(partCitations))
			}

			if tc.expectedCount == 1 && len(partCitations) == 1 {
				if tc.expectedPart != "" && partCitations[0].Components.ChapterNumber != tc.expectedPart {
					t.Errorf("Part: got %q, want %q",
						partCitations[0].Components.ChapterNumber, tc.expectedPart)
				}
			}
		})
	}
}

func TestOSCOLAParserNormalize(t *testing.T) {
	parser := NewOSCOLAParser()

	cases := []struct {
		name     string
		citation *Citation
		expected string
	}{
		{
			name: "act_with_chapter",
			citation: &Citation{
				Type:     CitationTypeStatute,
				Document: "Data Protection Act 2018",
				Components: CitationComponents{
					DocYear:   "2018",
					DocNumber: "12",
					CodeName:  "ukact",
				},
			},
			expected: "Data Protection Act 2018 [2018 c. 12]",
		},
		{
			name: "act_without_chapter",
			citation: &Citation{
				Type:     CitationTypeStatute,
				Document: "Human Rights Act 1998",
				Components: CitationComponents{
					DocYear:  "1998",
					CodeName: "ukact",
				},
			},
			expected: "Human Rights Act 1998",
		},
		{
			name: "statutory_instrument",
			citation: &Citation{
				Type: CitationTypeRegulation,
				Components: CitationComponents{
					DocYear:   "2019",
					DocNumber: "419",
				},
			},
			expected: "SI 2019/419",
		},
		{
			name: "neutral_citation",
			citation: &Citation{
				Type: CitationTypeCase,
				Components: CitationComponents{
					DocYear:   "2019",
					DocNumber: "UKSC/5",
					CodeName:  "neutral",
				},
			},
			expected: "[2019] UKSC/5",
		},
		{
			name: "law_report",
			citation: &Citation{
				Type:        CitationTypeCase,
				Subdivision: "1 AC 212",
				Components: CitationComponents{
					DocYear:   "1994",
					DocNumber: "AC/1/212",
					CodeName:  "lawreport",
				},
			},
			expected: "[1994] 1 AC 212",
		},
		{
			name: "section_simple",
			citation: &Citation{
				Type: CitationTypeStatute,
				Components: CitationComponents{
					Section: "114",
				},
			},
			expected: "s 114",
		},
		{
			name: "section_with_subsection",
			citation: &Citation{
				Type: CitationTypeStatute,
				Components: CitationComponents{
					Section:         "6",
					ParagraphNumber: 1,
				},
			},
			expected: "s 6(1)",
		},
		{
			name: "echr_article",
			citation: &Citation{
				Type: CitationTypeTreaty,
				Components: CitationComponents{
					ArticleNumber: 8,
				},
			},
			expected: "ECHR art 8",
		},
		{
			name: "echr_with_paragraph",
			citation: &Citation{
				Type: CitationTypeTreaty,
				Components: CitationComponents{
					ArticleNumber:   6,
					ParagraphNumber: 1,
				},
			},
			expected: "ECHR art 6(1)",
		},
		{
			name: "schedule_reference",
			citation: &Citation{
				Type:        CitationTypeStatute,
				Subdivision: "Schedule 7",
			},
			expected: "Schedule 7",
		},
		{
			name: "part_reference",
			citation: &Citation{
				Type:        CitationTypeStatute,
				Subdivision: "Part 3",
			},
			expected: "Part 3",
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

func TestOSCOLAParserToURI(t *testing.T) {
	parser := NewOSCOLAParser()

	cases := []struct {
		name        string
		citation    *Citation
		expectedURI string
		expectError bool
	}{
		{
			name: "act_with_chapter",
			citation: &Citation{
				Type:     CitationTypeStatute,
				Document: "Data Protection Act 2018",
				Components: CitationComponents{
					DocYear:   "2018",
					DocNumber: "12",
					CodeName:  "ukact",
				},
			},
			expectedURI: "urn:uk:act:2018/12",
		},
		{
			name: "act_without_chapter",
			citation: &Citation{
				Type:     CitationTypeStatute,
				Document: "Human Rights Act 1998",
				Components: CitationComponents{
					DocYear:  "1998",
					CodeName: "ukact",
				},
			},
			expectedURI: "urn:uk:act:1998/human-rights-act-1998",
		},
		{
			name: "statutory_instrument",
			citation: &Citation{
				Type: CitationTypeRegulation,
				Components: CitationComponents{
					DocYear:   "2019",
					DocNumber: "419",
				},
			},
			expectedURI: "urn:uk:si:2019/419",
		},
		{
			name: "neutral_case",
			citation: &Citation{
				Type: CitationTypeCase,
				Components: CitationComponents{
					DocYear:   "2019",
					DocNumber: "UKSC/5",
				},
			},
			expectedURI: "urn:uk:case:2019/UKSC/5",
		},
		{
			name: "echr_article",
			citation: &Citation{
				Type: CitationTypeTreaty,
				Components: CitationComponents{
					ArticleNumber: 8,
				},
			},
			expectedURI: "urn:echr:article:8",
		},
		{
			name: "act_missing_year",
			citation: &Citation{
				Type: CitationTypeStatute,
				Components: CitationComponents{
					CodeName: "ukact",
				},
			},
			expectError: true,
		},
		{
			name: "si_missing_year",
			citation: &Citation{
				Type: CitationTypeRegulation,
				Components: CitationComponents{
					DocNumber: "419",
				},
			},
			expectError: true,
		},
		{
			name: "case_missing_year",
			citation: &Citation{
				Type: CitationTypeCase,
				Components: CitationComponents{
					DocNumber: "UKSC/5",
				},
			},
			expectError: true,
		},
		{
			name: "echr_missing_article",
			citation: &Citation{
				Type: CitationTypeTreaty,
			},
			expectError: true,
		},
		{
			name: "section_ref_no_uri",
			citation: &Citation{
				Type: CitationTypeStatute,
				Components: CitationComponents{
					Section: "6",
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

func TestOSCOLAParserDPA2018Integration(t *testing.T) {
	dpaPath := "../../testdata/uk-dpa2018.txt"
	dpaData, err := os.ReadFile(dpaPath)
	if err != nil {
		t.Skipf("Skipping DPA 2018 integration test: %v", err)
	}
	dpaText := string(dpaData)

	parser := NewOSCOLAParser()
	citations, err := parser.Parse(dpaText)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	t.Logf("Total OSCOLA citations found in DPA 2018: %d", len(citations))

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

	// DPA 2018 should reference itself and other Acts.
	actCount := codeCounts["ukact"]
	if actCount < 1 {
		t.Errorf("Expected at least 1 UK Act citation in DPA 2018, got %d", actCount)
	}

	// DPA 2018 has Part references (Part 1 through Part 7).
	partCitations := filterBySubdivisionPrefix(citations, "Part")
	if len(partCitations) < 3 {
		t.Errorf("Expected at least 3 Part citations in DPA 2018, got %d", len(partCitations))
	}

	// DPA 2018 has section references.
	sectionCitations := filterBySectionComponent(citations)
	if len(sectionCitations) < 1 {
		t.Errorf("Expected at least 1 section citation in DPA 2018, got %d", len(sectionCitations))
	}

	// DPA 2018 has Schedule references.
	scheduleCitations := filterBySubdivisionPrefix(citations, "Schedule")
	if len(scheduleCitations) >= 1 {
		t.Logf("Found %d Schedule citations", len(scheduleCitations))
	}

	// Verify all citations have valid metadata.
	for i, c := range citations {
		if c.RawText == "" {
			t.Errorf("Citation %d has empty RawText", i)
		}
		if c.Parser != "OSCOLA Citation Parser" {
			t.Errorf("Citation %d has wrong Parser: %q", i, c.Parser)
		}
		if c.Confidence <= 0 {
			t.Errorf("Citation %d has non-positive Confidence: %f", i, c.Confidence)
		}
		if c.Jurisdiction != "UK" {
			t.Errorf("Citation %d has wrong Jurisdiction: %q", i, c.Jurisdiction)
		}
	}
}

func TestOSCOLAParserSIIntegration(t *testing.T) {
	siPath := "../../testdata/uk-si-example.txt"
	siData, err := os.ReadFile(siPath)
	if err != nil {
		t.Skipf("Skipping SI integration test: %v", err)
	}
	siText := string(siData)

	parser := NewOSCOLAParser()
	citations, err := parser.Parse(siText)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	t.Logf("Total OSCOLA citations found in SI example: %d", len(citations))

	// Count by type.
	typeCounts := make(map[CitationType]int)
	for _, c := range citations {
		typeCounts[c.Type]++
	}
	for citType, count := range typeCounts {
		t.Logf("  %s: %d", citType, count)
	}

	// The SI example should reference Acts (e.g., "Data Protection Act 2018").
	actCitations := filterByCodeNameOSCOLA(citations, "ukact")
	if len(actCitations) < 1 {
		t.Errorf("Expected at least 1 UK Act citation in SI example, got %d", len(actCitations))
	}

	// The SI example has section references.
	sectionCitations := filterBySectionComponent(citations)
	if len(sectionCitations) < 1 {
		t.Errorf("Expected at least 1 section citation in SI example, got %d", len(sectionCitations))
	}

	// Verify all citations have valid metadata.
	for i, c := range citations {
		if c.RawText == "" {
			t.Errorf("Citation %d has empty RawText", i)
		}
		if c.Parser != "OSCOLA Citation Parser" {
			t.Errorf("Citation %d has wrong Parser: %q", i, c.Parser)
		}
		if c.Jurisdiction != "UK" {
			t.Errorf("Citation %d has wrong Jurisdiction: %q", i, c.Jurisdiction)
		}
	}
}

func TestOSCOLAParserRegistration(t *testing.T) {
	registry := NewCitationRegistry()
	parser := NewOSCOLAParser()

	err := registry.Register(parser)
	if err != nil {
		t.Fatalf("Failed to register OSCOLA parser: %v", err)
	}

	// Verify it's retrievable by name.
	retrieved, ok := registry.Get("OSCOLA Citation Parser")
	if !ok || retrieved == nil {
		t.Error("Failed to retrieve registered parser")
	}

	// Verify it's listed for UK jurisdiction.
	ukParserNames := registry.ListByJurisdiction("UK")
	found := false
	for _, parserName := range ukParserNames {
		if parserName == "OSCOLA Citation Parser" {
			found = true
			break
		}
	}
	if !found {
		t.Error("OSCOLA parser not found in UK jurisdiction list")
	}
}

func TestOSCOLAParserMixedCitations(t *testing.T) {
	parser := NewOSCOLAParser()

	text := `The Data Protection Act 2018 [2018 c. 12] implements Regulation (EU) 2016/679.
Section 3 defines key terms. See s 6(1) for lawfulness requirements.
The European Union (Withdrawal) Act 2018 (2018 c. 16) provides the legal framework.
Under ECHR art 8, the right to private life is protected.
Schedule 7 lists competent authorities. Part 3 covers law enforcement processing.
SI 2019/419 amends several provisions.`

	citations, err := parser.Parse(text)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	t.Logf("Total citations in mixed text: %d", len(citations))
	for _, c := range citations {
		t.Logf("  [%s] %s: %q (conf=%.2f)", c.Type, c.Jurisdiction, c.RawText, c.Confidence)
	}

	// Should find: 2+ Acts, 1 SI, 1 ECHR, sections, schedule, part.
	actCitations := filterByCodeNameOSCOLA(citations, "ukact")
	if len(actCitations) < 2 {
		t.Errorf("Expected at least 2 UK Act citations, got %d", len(actCitations))
	}

	siCitations := filterByType(citations, CitationTypeRegulation)
	if len(siCitations) < 1 {
		t.Errorf("Expected at least 1 SI citation, got %d", len(siCitations))
	}

	echrCitations := filterByType(citations, CitationTypeTreaty)
	if len(echrCitations) < 1 {
		t.Errorf("Expected at least 1 ECHR citation, got %d", len(echrCitations))
	}
}

func TestOSCOLAParserNoOverlap(t *testing.T) {
	parser := NewOSCOLAParser()

	// The "[2018 c. 12]" chapter reference and "Act 2018" should not double-count
	// as the chapter ref is consumed first and its positions are marked.
	text := "[2018 c. 12]"
	citations, err := parser.Parse(text)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should be exactly 1 citation (chapter ref), not 2.
	if len(citations) != 1 {
		t.Errorf("Expected 1 citation for chapter ref, got %d", len(citations))
		for _, c := range citations {
			t.Logf("  Citation: type=%s doc=%q raw=%q", c.Type, c.Document, c.RawText)
		}
	}
}

func TestOSCOLAParserEmptyInput(t *testing.T) {
	parser := NewOSCOLAParser()

	citations, err := parser.Parse("")
	if err != nil {
		t.Fatalf("Parse failed on empty input: %v", err)
	}

	if len(citations) != 0 {
		t.Errorf("Expected 0 citations for empty input, got %d", len(citations))
	}
}

// Test helpers for OSCOLA parser tests.

func filterByCodeNameOSCOLA(citations []*Citation, codeName string) []*Citation {
	var filtered []*Citation
	for _, c := range citations {
		if c.Components.CodeName == codeName {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func filterBySectionComponent(citations []*Citation) []*Citation {
	var filtered []*Citation
	for _, c := range citations {
		if c.Components.Section != "" {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func filterBySubdivisionPrefix(citations []*Citation, prefix string) []*Citation {
	var filtered []*Citation
	for _, c := range citations {
		if len(c.Subdivision) >= len(prefix) && c.Subdivision[:len(prefix)] == prefix {
			filtered = append(filtered, c)
		}
	}
	return filtered
}
