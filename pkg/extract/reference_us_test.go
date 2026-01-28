package extract

import (
	"testing"
)

func TestExtractUSSectionRefs(t *testing.T) {
	extractor := NewReferenceExtractor()

	testCases := []struct {
		name     string
		text     string
		expected []struct {
			rawText    string
			identifier string
			articleNum int
			subRef     string
		}
	}{
		{
			name: "simple section reference",
			text: "pursuant to Section 1798.100",
			expected: []struct {
				rawText    string
				identifier string
				articleNum int
				subRef     string
			}{
				{"Section 1798.100", "Section 1798.100", 100, ""},
			},
		},
		{
			name: "section with subdivision",
			text: "as defined in Section 1798.140(o)",
			expected: []struct {
				rawText    string
				identifier string
				articleNum int
				subRef     string
			}{
				{"Section 1798.140(o)", "Section 1798.140(o)", 140, "subdivision"},
			},
		},
		{
			name: "section with subdivision and paragraph",
			text: "Section 1798.185(a)(1) requires",
			expected: []struct {
				rawText    string
				identifier string
				articleNum int
				subRef     string
			}{
				{"Section 1798.185(a)(1)", "Section 1798.185(a)(1)", 185, "paragraph"},
			},
		},
		{
			name: "subdivision of section",
			text: "subdivision (a) of Section 1798.100",
			expected: []struct {
				rawText    string
				identifier string
				articleNum int
				subRef     string
			}{
				{"subdivision (a) of Section 1798.100", "Section 1798.100(a)", 100, "subdivision"},
			},
		},
		{
			name: "paragraph of subdivision of section",
			text: "paragraph (5) of subdivision (a) of Section 1798.185",
			expected: []struct {
				rawText    string
				identifier string
				articleNum int
				subRef     string
			}{
				{"paragraph (5) of subdivision (a) of Section 1798.185", "Section 1798.185(a)(5)", 185, "paragraph"},
			},
		},
		{
			name: "sections range",
			text: "Sections 1798.100 to 1798.199",
			expected: []struct {
				rawText    string
				identifier string
				articleNum int
				subRef     string
			}{
				{"Sections 1798.100 to 1798.199", "Sections 1798.100-1798.199", 100, "range"},
			},
		},
		{
			name: "multiple references",
			text: "Section 1798.100 and Section 1798.140(o) apply",
			expected: []struct {
				rawText    string
				identifier string
				articleNum int
				subRef     string
			}{
				{"Section 1798.140(o)", "Section 1798.140(o)", 140, "subdivision"},
				{"Section 1798.100", "Section 1798.100", 100, ""},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			refs := extractor.extractUSSectionRefs(tc.text, 1)

			if len(refs) != len(tc.expected) {
				t.Errorf("expected %d refs, got %d", len(tc.expected), len(refs))
				for _, ref := range refs {
					t.Logf("  got: %q -> %s", ref.RawText, ref.Identifier)
				}
				return
			}

			for i, exp := range tc.expected {
				if refs[i].RawText != exp.rawText {
					t.Errorf("ref %d: expected rawText %q, got %q", i, exp.rawText, refs[i].RawText)
				}
				if refs[i].Identifier != exp.identifier {
					t.Errorf("ref %d: expected identifier %q, got %q", i, exp.identifier, refs[i].Identifier)
				}
				if refs[i].ArticleNum != exp.articleNum {
					t.Errorf("ref %d: expected articleNum %d, got %d", i, exp.articleNum, refs[i].ArticleNum)
				}
				if refs[i].SubRef != exp.subRef {
					t.Errorf("ref %d: expected subRef %q, got %q", i, exp.subRef, refs[i].SubRef)
				}
			}
		})
	}
}

func TestExtractUSExternalRefs(t *testing.T) {
	extractor := NewReferenceExtractor()

	testCases := []struct {
		name     string
		text     string
		expected []struct {
			rawText     string
			identifier  string
			externalDoc string
		}
	}{
		{
			name: "US Code reference with Section",
			text: "15 U.S.C. Section 1681",
			expected: []struct {
				rawText     string
				identifier  string
				externalDoc string
			}{
				{"15 U.S.C. Section 1681", "15 U.S.C. § 1681", "USC"},
			},
		},
		{
			name: "US Code reference with Sec.",
			text: "42 U.S.C. Sec. 1320d",
			expected: []struct {
				rawText     string
				identifier  string
				externalDoc string
			}{
				{"42 U.S.C. Sec. 1320d", "42 U.S.C. § 1320d", "USC"},
			},
		},
		{
			name: "CFR reference with Part",
			text: "45 C.F.R. Part 164",
			expected: []struct {
				rawText     string
				identifier  string
				externalDoc string
			}{
				{"45 C.F.R. Part 164", "45 C.F.R. Part 164", "CFR"},
			},
		},
		{
			name: "California Title reference",
			text: "Section 17014 of Title 18",
			expected: []struct {
				rawText     string
				identifier  string
				externalDoc string
			}{
				{"Section 17014 of Title 18", "Cal. Title 18 § 17014", "CalTitle"},
			},
		},
		{
			name: "Public Law reference",
			text: "Public Law 104-191",
			expected: []struct {
				rawText     string
				identifier  string
				externalDoc string
			}{
				{"Public Law 104-191", "Pub. L. 104-191", "PublicLaw"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			refs := extractor.extractUSExternalRefs(tc.text, 1)

			if len(refs) != len(tc.expected) {
				t.Errorf("expected %d refs, got %d", len(tc.expected), len(refs))
				for _, ref := range refs {
					t.Logf("  got: %q -> %s", ref.RawText, ref.Identifier)
				}
				return
			}

			for i, exp := range tc.expected {
				if refs[i].RawText != exp.rawText {
					t.Errorf("ref %d: expected rawText %q, got %q", i, exp.rawText, refs[i].RawText)
				}
				if refs[i].Identifier != exp.identifier {
					t.Errorf("ref %d: expected identifier %q, got %q", i, exp.identifier, refs[i].Identifier)
				}
				if refs[i].ExternalDoc != exp.externalDoc {
					t.Errorf("ref %d: expected externalDoc %q, got %q", i, exp.externalDoc, refs[i].ExternalDoc)
				}
				if refs[i].Type != ReferenceTypeExternal {
					t.Errorf("ref %d: expected type external, got %v", i, refs[i].Type)
				}
			}
		})
	}
}

func TestUSSectionPatternDoesNotMatchSimpleSections(t *testing.T) {
	extractor := NewReferenceExtractor()

	// EU-style "Section 1" should NOT match US-style pattern
	text := "Section 1 provides definitions"
	refs := extractor.extractUSSectionRefs(text, 1)

	if len(refs) != 0 {
		t.Errorf("US section pattern should not match simple 'Section 1', got %d refs", len(refs))
		for _, ref := range refs {
			t.Logf("  got: %q", ref.RawText)
		}
	}

	// But EU-style should still work
	refs = extractor.extractSectionRefs(text, 1)
	if len(refs) != 1 {
		t.Errorf("EU section pattern should match 'Section 1', got %d refs", len(refs))
	}
}

func TestCCPADocumentReferences(t *testing.T) {
	// Test with actual CCPA-style text
	text := `The consumer has the right to request that a business that collects
personal information about the consumer disclose to the consumer the following:
(1) The categories of personal information it has collected about that consumer.
(2) The categories of sources from which the personal information is collected.
See Section 1798.100 for details.
As used in subdivision (a) of Section 1798.110, "personal information" means
information as defined in Section 1798.140(o).
This section shall be read in conjunction with Sections 1798.100 to 1798.199.
For purposes of paragraph (5) of subdivision (a) of Section 1798.185, the
Attorney General shall adopt regulations.`

	extractor := NewReferenceExtractor()
	article := &Article{
		Number: 100,
		Text:   text,
	}

	refs := extractor.ExtractFromArticle(article)

	t.Logf("Found %d references:", len(refs))
	for _, ref := range refs {
		t.Logf("  %s: %q -> %s (Article %d)", ref.Type, ref.RawText, ref.Identifier, ref.ArticleNum)
	}

	// Should find at least 5 internal references
	internalCount := 0
	for _, ref := range refs {
		if ref.Type == ReferenceTypeInternal {
			internalCount++
		}
	}

	if internalCount < 5 {
		t.Errorf("expected at least 5 internal references, got %d", internalCount)
	}

	// Check specific references were found
	foundRefs := make(map[string]bool)
	for _, ref := range refs {
		foundRefs[ref.Identifier] = true
	}

	expectedRefs := []string{
		"Section 1798.100",
		"Section 1798.110(a)",
		"Section 1798.140(o)",
		"Sections 1798.100-1798.199",
		"Section 1798.185(a)(5)",
	}

	for _, exp := range expectedRefs {
		if !foundRefs[exp] {
			t.Errorf("expected to find reference %q", exp)
		}
	}
}

func TestUSStyleReferenceResolution(t *testing.T) {
	// Create a mock document with CCPA-style articles
	doc := &Document{
		Chapters: []*Chapter{
			{
				Number: "1",
				Title:  "General Provisions",
				Articles: []*Article{
					{Number: 100, Title: "Title", Text: "This title may be cited as CCPA."},
					{Number: 105, Title: "Intent", Text: "The Legislature finds and declares..."},
					{Number: 110, Title: "Definitions", Text: "For purposes of this title..."},
				},
			},
			{
				Number: "2",
				Title:  "Consumer Rights",
				Articles: []*Article{
					{Number: 115, Title: "Right to Know", Text: "A consumer shall have the right..."},
					{Number: 120, Title: "Right to Delete", Text: "A consumer shall have the right..."},
					{Number: 125, Title: "Right to Opt-Out", Text: "A consumer shall have the right..."},
					{Number: 130, Title: "Notice Requirements", Text: "A business shall provide notice..."},
					{Number: 140, Title: "Privacy Policy", Text: "A business shall make available..."},
				},
			},
			{
				Number: "3",
				Title:  "Business Obligations",
				Articles: []*Article{
					{Number: 145, Title: "Verification", Text: "A business shall establish..."},
					{Number: 150, Title: "Service Providers", Text: "A service provider shall..."},
					{Number: 185, Title: "Regulations", Text: "The Attorney General shall..."},
				},
			},
		},
	}

	// Create resolver and index document
	resolver := NewReferenceResolver("https://regula.dev/ccpa#", "CCPA")
	resolver.IndexDocument(doc)

	// Test cases for US-style reference resolution
	testCases := []struct {
		name           string
		ref            *Reference
		expectedStatus ResolutionStatus
		expectResolved bool
	}{
		{
			name: "simple section reference",
			ref: &Reference{
				Type:          ReferenceTypeInternal,
				Target:        TargetSection,
				RawText:       "Section 1798.100",
				Identifier:    "Section 1798.100",
				ArticleNum:    100,
				SectionNum:    1798100,
				SourceArticle: 115,
			},
			expectedStatus: ResolutionResolved,
			expectResolved: true,
		},
		{
			name: "section with subdivision",
			ref: &Reference{
				Type:          ReferenceTypeInternal,
				Target:        TargetSection,
				RawText:       "Section 1798.140(o)",
				Identifier:    "Section 1798.140(o)",
				ArticleNum:    140,
				PointLetter:   "o",
				SubRef:        "subdivision",
				SectionNum:    1798140,
				SourceArticle: 110,
			},
			expectedStatus: ResolutionPartial, // Subdivision may not be indexed
			expectResolved: true,
		},
		{
			name: "section range",
			ref: &Reference{
				Type:          ReferenceTypeInternal,
				Target:        TargetSection,
				RawText:       "Sections 1798.100 to 1798.130",
				Identifier:    "Sections 1798.100-1798.130",
				ArticleNum:    100,
				SubRef:        "range",
				SectionNum:    1798100,
				SourceArticle: 185,
			},
			expectedStatus: ResolutionRangeRef,
			expectResolved: true,
		},
		{
			name: "non-existent section",
			ref: &Reference{
				Type:          ReferenceTypeInternal,
				Target:        TargetSection,
				RawText:       "Section 1798.999",
				Identifier:    "Section 1798.999",
				ArticleNum:    999,
				SectionNum:    1798999,
				SourceArticle: 100,
			},
			expectedStatus: ResolutionNotFound,
			expectResolved: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := resolver.Resolve(tc.ref)

			t.Logf("Reference: %q -> Status: %s, Reason: %s",
				tc.ref.RawText, result.Status, result.Reason)

			if result.Status != tc.expectedStatus {
				t.Errorf("expected status %s, got %s", tc.expectedStatus, result.Status)
			}

			if tc.expectResolved && result.TargetURI == "" && len(result.TargetURIs) == 0 {
				t.Error("expected resolution to produce target URI")
			}

			if result.TargetURI != "" {
				t.Logf("  Target URI: %s", result.TargetURI)
			}
			if len(result.TargetURIs) > 0 {
				t.Logf("  Target URIs: %v", result.TargetURIs)
			}
		})
	}
}

func TestUSExternalReferenceResolution(t *testing.T) {
	resolver := NewReferenceResolver("https://regula.dev/ccpa#", "CCPA")

	testCases := []struct {
		name        string
		ref         *Reference
		expectedURI string
	}{
		{
			name: "US Code reference",
			ref: &Reference{
				Type:        ReferenceTypeExternal,
				Target:      TargetRegulation,
				RawText:     "15 U.S.C. Section 1681",
				Identifier:  "15 U.S.C. § 1681",
				ExternalDoc: "USC",
				DocNumber:   "15",
				SectionNum:  1681,
			},
			expectedURI: "urn:us:usc:15/1681",
		},
		{
			name: "CFR reference",
			ref: &Reference{
				Type:        ReferenceTypeExternal,
				Target:      TargetRegulation,
				RawText:     "45 C.F.R. Part 164",
				Identifier:  "45 C.F.R. Part 164",
				ExternalDoc: "CFR",
				DocNumber:   "45",
				SectionNum:  164,
			},
			expectedURI: "urn:us:cfr:45/164",
		},
		{
			name: "Public Law reference",
			ref: &Reference{
				Type:        ReferenceTypeExternal,
				Target:      TargetRegulation,
				RawText:     "Public Law 104-191",
				Identifier:  "Pub. L. 104-191",
				ExternalDoc: "PublicLaw",
				DocYear:     "104",
				DocNumber:   "191",
			},
			expectedURI: "urn:us:pl:104-191",
		},
		{
			name: "California Title reference",
			ref: &Reference{
				Type:        ReferenceTypeExternal,
				Target:      TargetSection,
				RawText:     "Section 17014 of Title 18",
				Identifier:  "Cal. Title 18 § 17014",
				ExternalDoc: "CalTitle",
				DocNumber:   "18",
				SectionNum:  17014,
			},
			expectedURI: "urn:us:ca:title18/sec17014",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := resolver.Resolve(tc.ref)

			if result.Status != ResolutionExternal {
				t.Errorf("expected status external, got %s", result.Status)
			}

			if result.TargetURI != tc.expectedURI {
				t.Errorf("expected URI %q, got %q", tc.expectedURI, result.TargetURI)
			}

			t.Logf("External reference %q -> %s", tc.ref.RawText, result.TargetURI)
		})
	}
}
