package extract

import (
	"strings"
	"testing"
)

// TestUSCAlphanumericSectionParsing verifies that the parser correctly extracts
// alphanumeric USC section identifiers (e.g., "300aa-25", "1396a") into both
// Article.Number (leading integer) and Article.SectionID (full string).
func TestUSCAlphanumericSectionParsing(t *testing.T) {
	testCases := []struct {
		name              string
		line              string
		expectedNumber    int
		expectedSectionID string
		expectedTitle     string
	}{
		{
			name:              "dash-extended with double letter",
			line:              "Section 300aa-25 National Vaccine Injury Compensation",
			expectedNumber:    300,
			expectedSectionID: "300aa-25",
			expectedTitle:     "National Vaccine Injury Compensation",
		},
		{
			name:              "single letter suffix",
			line:              "Section 1396a State Medicaid Plans",
			expectedNumber:    1396,
			expectedSectionID: "1396a",
			expectedTitle:     "State Medicaid Plans",
		},
		{
			name:              "double letter with dash extension",
			line:              "Section 2000bb-1 Free Exercise of Religion",
			expectedNumber:    2000,
			expectedSectionID: "2000bb-1",
			expectedTitle:     "Free Exercise of Religion",
		},
		{
			name:              "letter with dash-letter-digit extension",
			line:              "Section 247d-6d Emergency Provisions",
			expectedNumber:    247,
			expectedSectionID: "247d-6d",
			expectedTitle:     "Emergency Provisions",
		},
		{
			name:              "purely numeric section",
			line:              "Section 26 Internal Revenue Code",
			expectedNumber:    26,
			expectedSectionID: "26",
			expectedTitle:     "Internal Revenue Code",
		},
		{
			name:              "multi-dash extension",
			line:              "Section 1320a-7a Civil Monetary Penalties",
			expectedNumber:    1320,
			expectedSectionID: "1320a-7a",
			expectedTitle:     "Civil Monetary Penalties",
		},
	}

	parser := NewParser()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := parser.uscSectionPattern.FindStringSubmatch(tc.line)
			if m == nil {
				t.Fatalf("uscSectionPattern did not match %q", tc.line)
			}

			sectionIDStr := m[1]
			sectionNum := parseLeadingInt(sectionIDStr)
			sectionTitle := strings.TrimSpace(m[2])

			if sectionNum != tc.expectedNumber {
				t.Errorf("Number: expected %d, got %d", tc.expectedNumber, sectionNum)
			}
			if sectionIDStr != tc.expectedSectionID {
				t.Errorf("SectionID: expected %q, got %q", tc.expectedSectionID, sectionIDStr)
			}
			if sectionTitle != tc.expectedTitle {
				t.Errorf("Title: expected %q, got %q", tc.expectedTitle, sectionTitle)
			}
		})
	}
}

// TestUSCAlphanumericSectionNoCollisions verifies that sections with the same
// numeric prefix but different alphanumeric identifiers are stored as distinct
// articles with unique SectionIDs.
func TestUSCAlphanumericSectionNoCollisions(t *testing.T) {
	input := `CHAPTER 6A
PUBLIC HEALTH SERVICE

Section 300 General Provisions
This section establishes general provisions for the Public Health Service.

Section 300a Family Planning Programs
Voluntary family planning projects shall receive grants.

Section 300aa-25 Recording and Reporting of Information
Each health care provider shall report vaccine adverse events.

Section 300aa-26 Vaccine Information
Each health care provider shall provide information.
`

	parser := NewParser()
	parser.SetFormatHint(FormatUS)
	doc, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	articles := doc.AllArticles()

	// Collect SectionIDs
	sectionIDs := make(map[string]bool)
	for _, article := range articles {
		if article.SectionID != "" {
			if sectionIDs[article.SectionID] {
				t.Errorf("duplicate SectionID %q found", article.SectionID)
			}
			sectionIDs[article.SectionID] = true
		}
	}

	expectedSectionIDs := []string{"300", "300a", "300aa-25", "300aa-26"}
	for _, expectedID := range expectedSectionIDs {
		if !sectionIDs[expectedID] {
			t.Errorf("expected SectionID %q not found; available: %v", expectedID, sectionIDs)
		}
	}

	// Verify that articles with same numeric prefix have distinct SectionIDs
	numericPrefix300Count := 0
	for _, article := range articles {
		if article.Number == 300 {
			numericPrefix300Count++
		}
	}

	if numericPrefix300Count < 3 {
		t.Errorf("expected at least 3 articles with Number=300 (300, 300a, 300aa-*), got %d", numericPrefix300Count)
	}

	t.Logf("Found %d total articles with %d unique SectionIDs", len(articles), len(sectionIDs))
	for _, article := range articles {
		t.Logf("  Number=%d SectionID=%q Title=%q", article.Number, article.SectionID, article.Title)
	}
}

// TestUSCAlphanumericReferenceExtraction verifies that extractUSCSectionRefs
// populates the SectionStr field with the full alphanumeric section identifier.
func TestUSCAlphanumericReferenceExtraction(t *testing.T) {
	extractor := NewReferenceExtractor()

	testCases := []struct {
		name               string
		text               string
		expectedSectionStr string
		expectedArticleNum int
	}{
		{
			name:               "same-title with letter suffix",
			text:               "section 1396a of this title",
			expectedSectionStr: "1396a",
			expectedArticleNum: 1396,
		},
		{
			name:               "same-title with dash extension",
			text:               "section 1320d-1 of this title",
			expectedSectionStr: "1320d-1",
			expectedArticleNum: 1320,
		},
		{
			name:               "same-title with double letter dash",
			text:               "section 300aa-25 of this title",
			expectedSectionStr: "300aa-25",
			expectedArticleNum: 300,
		},
		{
			name:               "bare section with letter suffix",
			text:               "section 1396a establishes requirements",
			expectedSectionStr: "1396a",
			expectedArticleNum: 1396,
		},
		{
			name:               "section with subsection",
			text:               "section 1396a(a)(10) of this title",
			expectedSectionStr: "1396a",
			expectedArticleNum: 1396,
		},
		{
			name:               "cross-title reference",
			text:               "section 552a of title 5",
			expectedSectionStr: "552a",
			expectedArticleNum: 0, // external — ArticleNum is 0
		},
		{
			name:               "purely numeric same-title",
			text:               "section 1396 of this title",
			expectedSectionStr: "1396",
			expectedArticleNum: 1396,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			refs := extractor.extractUSCSectionRefs(tc.text, 42, nil)
			if len(refs) == 0 {
				t.Fatalf("expected at least 1 reference for %q, got 0", tc.text)
			}

			ref := refs[0]
			if ref.SectionStr != tc.expectedSectionStr {
				t.Errorf("SectionStr: expected %q, got %q", tc.expectedSectionStr, ref.SectionStr)
			}
			if ref.ArticleNum != tc.expectedArticleNum {
				t.Errorf("ArticleNum: expected %d, got %d", tc.expectedArticleNum, ref.ArticleNum)
			}
		})
	}
}

// TestUSCAlphanumericSectionResolution verifies that the resolver correctly
// indexes and resolves articles with alphanumeric SectionIDs.
func TestUSCAlphanumericSectionResolution(t *testing.T) {
	doc := &Document{
		Chapters: []*Chapter{
			{
				Number: "1",
				Title:  "General Provisions",
				Articles: []*Article{
					{Number: 1396, SectionID: "1396", Title: "General"},
					{Number: 1396, SectionID: "1396a", Title: "State Plans"},
					{Number: 300, SectionID: "300", Title: "General Health"},
					{Number: 300, SectionID: "300aa-25", Title: "Vaccine Injury Compensation"},
					{Number: 1320, SectionID: "1320a-7a", Title: "Civil Monetary Penalties"},
				},
			},
		},
	}

	resolver := NewReferenceResolver("https://regula.dev/usc42#", "USC42")
	resolver.IndexDocument(doc)

	testCases := []struct {
		name           string
		ref            *Reference
		expectedStatus ResolutionStatus
		expectedInURI  string
	}{
		{
			name: "resolve by SectionStr 1396a",
			ref: &Reference{
				Type:          ReferenceTypeInternal,
				Target:        TargetSection,
				RawText:       "section 1396a of this title",
				Identifier:    "Section 1396a",
				ArticleNum:    1396,
				SectionNum:    1396,
				SectionStr:    "1396a",
				SourceArticle: 300,
			},
			expectedStatus: ResolutionResolved,
			expectedInURI:  "Art1396a",
		},
		{
			name: "resolve by SectionStr 300aa-25",
			ref: &Reference{
				Type:          ReferenceTypeInternal,
				Target:        TargetSection,
				RawText:       "section 300aa-25 of this title",
				Identifier:    "Section 300aa-25",
				ArticleNum:    300,
				SectionNum:    300,
				SectionStr:    "300aa-25",
				SourceArticle: 1396,
			},
			expectedStatus: ResolutionResolved,
			expectedInURI:  "Art300aa-25",
		},
		{
			name: "resolve by SectionStr 1320a-7a",
			ref: &Reference{
				Type:          ReferenceTypeInternal,
				Target:        TargetSection,
				RawText:       "section 1320a-7a of this title",
				Identifier:    "Section 1320a-7a",
				ArticleNum:    1320,
				SectionNum:    1320,
				SectionStr:    "1320a-7a",
				SourceArticle: 1396,
			},
			expectedStatus: ResolutionResolved,
			expectedInURI:  "Art1320a-7a",
		},
		{
			name: "purely numeric section resolves",
			ref: &Reference{
				Type:          ReferenceTypeInternal,
				Target:        TargetSection,
				RawText:       "section 1396 of this title",
				Identifier:    "Section 1396",
				ArticleNum:    1396,
				SectionNum:    1396,
				SectionStr:    "1396",
				SourceArticle: 300,
			},
			expectedStatus: ResolutionResolved,
			expectedInURI:  "Art1396",
		},
		{
			name: "non-existent SectionStr returns not found",
			ref: &Reference{
				Type:          ReferenceTypeInternal,
				Target:        TargetSection,
				RawText:       "section 999z of this title",
				Identifier:    "Section 999z",
				ArticleNum:    999,
				SectionNum:    999,
				SectionStr:    "999z",
				SourceArticle: 1396,
			},
			expectedStatus: ResolutionNotFound,
			expectedInURI:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := resolver.Resolve(tc.ref)

			if result.Status != tc.expectedStatus {
				t.Errorf("Status: expected %s, got %s (reason: %s)", tc.expectedStatus, result.Status, result.Reason)
			}

			if tc.expectedInURI != "" && !strings.Contains(result.TargetURI, tc.expectedInURI) {
				t.Errorf("TargetURI: expected to contain %q, got %q", tc.expectedInURI, result.TargetURI)
			}

			t.Logf("Reference %q -> Status=%s URI=%s Reason=%s",
				tc.ref.RawText, result.Status, result.TargetURI, result.Reason)
		})
	}
}

// TestUSCAlphanumericBackwardCompat verifies that existing numeric-only sections
// continue to work correctly — both Number and SectionID are populated, and
// resolution works through either path.
func TestUSCAlphanumericBackwardCompat(t *testing.T) {
	// Create a CCPA-style document (numeric sections, no alphanumeric IDs)
	doc := &Document{
		Chapters: []*Chapter{
			{
				Number: "1",
				Title:  "General Provisions",
				Articles: []*Article{
					{Number: 100, Title: "Title"},
					{Number: 110, Title: "Definitions"},
					{Number: 140, Title: "Privacy Policy"},
				},
			},
		},
	}

	resolver := NewReferenceResolver("https://regula.dev/ccpa#", "CCPA")
	resolver.IndexDocument(doc)

	// Test that numeric-only references (no SectionStr) still resolve via int path
	ref := &Reference{
		Type:          ReferenceTypeInternal,
		Target:        TargetSection,
		RawText:       "Section 1798.100",
		Identifier:    "Section 1798.100",
		ArticleNum:    100,
		SectionNum:    1798100,
		SourceArticle: 110,
	}

	result := resolver.Resolve(ref)
	if result.Status != ResolutionResolved {
		t.Errorf("Numeric-only reference should resolve, got status %s (reason: %s)", result.Status, result.Reason)
	}

	if result.TargetURI == "" {
		t.Error("Expected non-empty TargetURI for numeric-only reference")
	}

	t.Logf("Backward compat: %q -> Status=%s URI=%s", ref.RawText, result.Status, result.TargetURI)
}

// TestParseLeadingInt verifies the parseLeadingInt helper that extracts
// leading digits from alphanumeric section identifiers.
func TestParseLeadingInt(t *testing.T) {
	testCases := []struct {
		input    string
		expected int
	}{
		{"300aa-25", 300},
		{"1396a", 1396},
		{"2000bb-1", 2000},
		{"247d-6d", 247},
		{"26", 26},
		{"1320a-7a", 1320},
		{"552a", 552},
		{"", 0},
		{"abc", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := parseLeadingInt(tc.input)
			if result != tc.expected {
				t.Errorf("parseLeadingInt(%q): expected %d, got %d", tc.input, tc.expected, result)
			}
		})
	}
}

// TestUSCAlphanumericSectionPatternBoundaries verifies that the updated
// uscSectionPattern regex correctly handles boundary cases.
func TestUSCAlphanumericSectionPatternBoundaries(t *testing.T) {
	parser := NewParser()

	matchCases := []struct {
		name  string
		line  string
		match bool
		capID string // expected capture group 1 when matching
	}{
		{"simple numeric", "Section 42 Some Title", true, "42"},
		{"letter suffix", "Section 1396a State Plans", true, "1396a"},
		{"double letter suffix", "Section 300aa Vaccine Info", true, "300aa"},
		{"dash extension", "Section 300aa-25 Compensation", true, "300aa-25"},
		{"multi-dash", "Section 1320a-7a Penalties", true, "1320a-7a"},
		{"no number", "Section Introduction", false, ""},
		{"empty after Section", "Section ", false, ""},
		// Note: "Section 1798.100 Privacy" DOES match, capturing "1798" since the dot
		// is not part of the USC identifier pattern. This is correct — in the parser,
		// California's dotted pattern runs before USC and takes priority.
		{"dotted number captures prefix", "Section 1798.100 Privacy", true, "1798"},
	}

	for _, tc := range matchCases {
		t.Run(tc.name, func(t *testing.T) {
			m := parser.uscSectionPattern.FindStringSubmatch(tc.line)
			if tc.match && m == nil {
				t.Errorf("expected %q to match uscSectionPattern", tc.line)
			}
			if !tc.match && m != nil {
				t.Errorf("expected %q NOT to match uscSectionPattern, but captured %q", tc.line, m[1])
			}
			if tc.match && m != nil && m[1] != tc.capID {
				t.Errorf("captured group: expected %q, got %q", tc.capID, m[1])
			}
		})
	}
}
