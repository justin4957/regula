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
		{
			name: "USC cross-title section reference",
			ref: &Reference{
				Type:        ReferenceTypeExternal,
				Target:      TargetSection,
				RawText:     "section 552a of title 5",
				Identifier:  "5 U.S.C. § 552a",
				ExternalDoc: "USC",
				DocNumber:   "5",
				SectionNum:  552,
			},
			expectedURI: "urn:us:usc:5/552",
		},
		{
			name: "USC act reference",
			ref: &Reference{
				Type:        ReferenceTypeExternal,
				Target:      TargetSection,
				RawText:     "section 306 of the Public Health Service Act",
				Identifier:  "306 of Public Health Service Act",
				ExternalDoc: "USAct",
				DocNumber:   "Public Health Service Act",
				SectionNum:  306,
			},
			expectedURI: "urn:us:act:public-health-service-act/sec306",
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

// --- USC-specific reference extraction tests ---

func TestExtractUSCSectionRefs(t *testing.T) {
	extractor := NewReferenceExtractor()

	testCases := []struct {
		name     string
		text     string
		expected []struct {
			rawText     string
			identifier  string
			refType     ReferenceType
			target      ReferenceTarget
			articleNum  int
			sectionNum  int
			externalDoc string
		}
	}{
		{
			name: "section of this title",
			text: "as described in section 1396 of this title",
			expected: []struct {
				rawText     string
				identifier  string
				refType     ReferenceType
				target      ReferenceTarget
				articleNum  int
				sectionNum  int
				externalDoc string
			}{
				{"section 1396 of this title", "Section 1396", ReferenceTypeInternal, TargetSection, 1396, 1396, ""},
			},
		},
		{
			name: "section with letter suffix of this title",
			text: "under section 1396a of this title",
			expected: []struct {
				rawText     string
				identifier  string
				refType     ReferenceType
				target      ReferenceTarget
				articleNum  int
				sectionNum  int
				externalDoc string
			}{
				{"section 1396a of this title", "Section 1396a", ReferenceTypeInternal, TargetSection, 1396, 1396, ""},
			},
		},
		{
			name: "section with subsection of this title",
			text: "section 1396a(a)(10) of this title",
			expected: []struct {
				rawText     string
				identifier  string
				refType     ReferenceType
				target      ReferenceTarget
				articleNum  int
				sectionNum  int
				externalDoc string
			}{
				{"section 1396a(a)(10) of this title", "Section 1396a(a)(10)", ReferenceTypeInternal, TargetSection, 1396, 1396, ""},
			},
		},
		{
			name: "section with dash extension",
			text: "pursuant to section 1320d-1 of this title",
			expected: []struct {
				rawText     string
				identifier  string
				refType     ReferenceType
				target      ReferenceTarget
				articleNum  int
				sectionNum  int
				externalDoc string
			}{
				{"section 1320d-1 of this title", "Section 1320d-1", ReferenceTypeInternal, TargetSection, 1320, 1320, ""},
			},
		},
		{
			name: "cross-title reference",
			text: "section 552a of title 5",
			expected: []struct {
				rawText     string
				identifier  string
				refType     ReferenceType
				target      ReferenceTarget
				articleNum  int
				sectionNum  int
				externalDoc string
			}{
				{"section 552a of title 5", "5 U.S.C. § 552a", ReferenceTypeExternal, TargetSection, 0, 552, "USC"},
			},
		},
		{
			name: "section of an Act",
			text: "section 306 of the Public Health Service Act",
			expected: []struct {
				rawText     string
				identifier  string
				refType     ReferenceType
				target      ReferenceTarget
				articleNum  int
				sectionNum  int
				externalDoc string
			}{
				{"section 306 of the Public Health Service Act", "306 of Public Health Service Act", ReferenceTypeExternal, TargetSection, 0, 306, "USAct"},
			},
		},
		{
			name: "section with subsection notation",
			text: "the requirements of section 1396a(a) apply",
			expected: []struct {
				rawText     string
				identifier  string
				refType     ReferenceType
				target      ReferenceTarget
				articleNum  int
				sectionNum  int
				externalDoc string
			}{
				{"section 1396a(a)", "Section 1396a(a)", ReferenceTypeInternal, TargetSection, 1396, 1396, ""},
			},
		},
		{
			name: "section with subsection and paragraph",
			text: "section 1396a(a)(10) provides",
			expected: []struct {
				rawText     string
				identifier  string
				refType     ReferenceType
				target      ReferenceTarget
				articleNum  int
				sectionNum  int
				externalDoc string
			}{
				{"section 1396a(a)(10)", "Section 1396a(a)(10)", ReferenceTypeInternal, TargetSection, 1396, 1396, ""},
			},
		},
		{
			name: "subsection relative reference",
			text: "as provided in subsection (b)",
			expected: []struct {
				rawText     string
				identifier  string
				refType     ReferenceType
				target      ReferenceTarget
				articleNum  int
				sectionNum  int
				externalDoc string
			}{
				{"subsection (b)", "subsection (b)", ReferenceTypeInternal, TargetSubsection, 42, 0, ""},
			},
		},
		{
			name: "subsection with paragraph",
			text: "under subsection (b)(1)",
			expected: []struct {
				rawText     string
				identifier  string
				refType     ReferenceType
				target      ReferenceTarget
				articleNum  int
				sectionNum  int
				externalDoc string
			}{
				{"subsection (b)(1)", "subsection (b)(1)", ReferenceTypeInternal, TargetSubsection, 42, 0, ""},
			},
		},
		{
			name: "paragraph of subsection",
			text: "paragraph (2) of subsection (a)",
			expected: []struct {
				rawText     string
				identifier  string
				refType     ReferenceType
				target      ReferenceTarget
				articleNum  int
				sectionNum  int
				externalDoc string
			}{
				{"paragraph (2) of subsection (a)", "subsection (a)(2)", ReferenceTypeInternal, TargetSubsection, 42, 0, ""},
			},
		},
		{
			name: "subchapter of chapter",
			text: "subchapter II of chapter 7",
			expected: []struct {
				rawText     string
				identifier  string
				refType     ReferenceType
				target      ReferenceTarget
				articleNum  int
				sectionNum  int
				externalDoc string
			}{
				{"subchapter II of chapter 7", "subchapter II of chapter 7", ReferenceTypeInternal, TargetSubchapter, 0, 0, ""},
			},
		},
		{
			name: "bare section with letter suffix",
			text: "section 1396a establishes requirements",
			expected: []struct {
				rawText     string
				identifier  string
				refType     ReferenceType
				target      ReferenceTarget
				articleNum  int
				sectionNum  int
				externalDoc string
			}{
				{"section 1396a", "Section 1396a", ReferenceTypeInternal, TargetSection, 1396, 1396, ""},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Source article 42 for relative references
			refs := extractor.extractUSCSectionRefs(tc.text, 42, nil)

			if len(refs) != len(tc.expected) {
				t.Errorf("expected %d refs, got %d", len(tc.expected), len(refs))
				for _, ref := range refs {
					t.Logf("  got: type=%s target=%s rawText=%q identifier=%q articleNum=%d sectionNum=%d externalDoc=%q",
						ref.Type, ref.Target, ref.RawText, ref.Identifier, ref.ArticleNum, ref.SectionNum, ref.ExternalDoc)
				}
				return
			}

			for i, exp := range tc.expected {
				ref := refs[i]
				if ref.RawText != exp.rawText {
					t.Errorf("ref %d: expected rawText %q, got %q", i, exp.rawText, ref.RawText)
				}
				if ref.Identifier != exp.identifier {
					t.Errorf("ref %d: expected identifier %q, got %q", i, exp.identifier, ref.Identifier)
				}
				if ref.Type != exp.refType {
					t.Errorf("ref %d: expected type %s, got %s", i, exp.refType, ref.Type)
				}
				if ref.Target != exp.target {
					t.Errorf("ref %d: expected target %s, got %s", i, exp.target, ref.Target)
				}
				if ref.ArticleNum != exp.articleNum {
					t.Errorf("ref %d: expected articleNum %d, got %d", i, exp.articleNum, ref.ArticleNum)
				}
				if ref.SectionNum != exp.sectionNum {
					t.Errorf("ref %d: expected sectionNum %d, got %d", i, exp.sectionNum, ref.SectionNum)
				}
				if ref.ExternalDoc != exp.externalDoc {
					t.Errorf("ref %d: expected externalDoc %q, got %q", i, exp.externalDoc, ref.ExternalDoc)
				}
			}
		})
	}
}

func TestUSCPatternDoesNotMatchCaliforniaStyle(t *testing.T) {
	extractor := NewReferenceExtractor()

	// California-style "Section 1798.100" should NOT be matched by USC patterns
	text := "Section 1798.100 and Section 1798.140(o) apply"
	uscRefs := extractor.extractUSCSectionRefs(text, 1, nil)

	// USC bare pattern requires letter suffix — dotted numbers should not match
	for _, ref := range uscRefs {
		if ref.RawText == "Section 1798" || ref.RawText == "section 1798" {
			t.Errorf("USC pattern should not match California-style dotted section, got %q", ref.RawText)
		}
	}

	// California pattern should still work
	calRefs := extractor.extractUSSectionRefs(text, 1)
	if len(calRefs) < 2 {
		t.Errorf("California pattern should match 'Section 1798.100' and 'Section 1798.140(o)', got %d refs", len(calRefs))
	}
}

func TestUSCDocumentReferences(t *testing.T) {
	// Realistic USC-style article text from Title 42
	text := `(a) For purposes of this section, the term "medical assistance" means payment of
part or all of the cost of the following care and services provided under a State
plan approved under this subchapter. Each State must provide medical assistance
for all individuals who meet the requirements of section 1396a of this title with
respect to eligibility. The amount, duration, and scope of medical assistance made
available shall be determined in accordance with section 1396a(a)(10) of this title,
subject to the limitations specified in subsection (b).
(b) The Secretary shall establish regulations consistent with section 1320a-7a of
this title and section 552a of title 5 to ensure compliance. The requirements set
forth in paragraph (2) of subsection (a) shall apply to all providers.
(c) Nothing in this section shall be construed to modify the requirements under
subchapter II of chapter 7 or chapter 8 of this title.
(d) Any provider of services described in section 1395x(s) of this title or
section 306 of the Public Health Service Act shall comply with the standards
established under subsection (a)(1).`

	extractor := NewReferenceExtractor()
	article := &Article{
		Number: 1396,
		Text:   text,
	}

	refs := extractor.ExtractFromArticle(article)

	t.Logf("Found %d total references:", len(refs))
	for _, ref := range refs {
		t.Logf("  %s/%s: %q -> %s (article=%d, section=%d, doc=%q)",
			ref.Type, ref.Target, ref.RawText, ref.Identifier, ref.ArticleNum, ref.SectionNum, ref.ExternalDoc)
	}

	// Count by type
	internalCount := 0
	externalCount := 0
	for _, ref := range refs {
		switch ref.Type {
		case ReferenceTypeInternal:
			internalCount++
		case ReferenceTypeExternal:
			externalCount++
		}
	}

	// Should find many internal references
	if internalCount < 8 {
		t.Errorf("expected at least 8 internal references, got %d", internalCount)
	}

	// Should find external references (cross-title + act)
	if externalCount < 2 {
		t.Errorf("expected at least 2 external references, got %d", externalCount)
	}

	// Check specific references
	foundIdentifiers := make(map[string]bool)
	for _, ref := range refs {
		foundIdentifiers[ref.Identifier] = true
	}

	expectedIdentifiers := []string{
		"Section 1396a",                 // "section 1396a of this title"
		"Section 1396a(a)(10)",          // "section 1396a(a)(10) of this title"
		"Section 1320a-7a",              // "section 1320a-7a of this title" (bare with dash)
		"5 U.S.C. § 552a",              // "section 552a of title 5" (cross-title)
		"subsection (b)",                // "subsection (b)"
		"subsection (a)(2)",             // "paragraph (2) of subsection (a)"
		"subchapter II of chapter 7",    // "subchapter II of chapter 7"
		"subsection (a)(1)",             // "subsection (a)(1)"
	}

	for _, exp := range expectedIdentifiers {
		if !foundIdentifiers[exp] {
			t.Errorf("expected to find reference with identifier %q", exp)
		}
	}

	// Verify the Act reference
	foundAct := false
	for _, ref := range refs {
		if ref.ExternalDoc == "USAct" && ref.SectionNum == 306 {
			foundAct = true
			break
		}
	}
	if !foundAct {
		t.Error("expected to find Public Health Service Act reference")
	}
}

func TestUSCPatternCaseInsensitive(t *testing.T) {
	extractor := NewReferenceExtractor()

	testCases := []struct {
		name string
		text string
	}{
		{"lowercase section", "section 1396a of this title"},
		{"uppercase Section", "Section 1396a of this title"},
		{"uppercase SECTION", "SECTION 1396a of this title"},
		{"lowercase subsection", "subsection (a)"},
		{"uppercase Subsection", "Subsection (a)"},
		{"lowercase subchapter", "subchapter II of chapter 7"},
		{"mixed case Chapter", "Chapter 7"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			refs := extractor.extractUSCSectionRefs(tc.text, 1, nil)
			if len(refs) == 0 {
				t.Errorf("expected at least 1 reference for %q, got 0", tc.text)
			}
		})
	}
}

func TestUSCNoFalsePositivesOnGDPR(t *testing.T) {
	// GDPR text should not trigger USC-specific patterns
	gdprText := `Article 5(1)(a) requires that personal data shall be processed lawfully,
fairly and in a transparent manner in relation to the data subject. The controller
shall implement appropriate technical and organisational measures in accordance with
Article 25 and Article 32 of this Regulation. References to Chapter III and Chapter IV
shall be interpreted consistently with the principles set forth in Section 1.`

	extractor := NewReferenceExtractor()
	uscRefs := extractor.extractUSCSectionRefs(gdprText, 5, nil)

	// "Section 1" should not match bare USC pattern (no letter suffix)
	for _, ref := range uscRefs {
		t.Logf("  USC pattern matched in GDPR text: %q -> %s", ref.RawText, ref.Identifier)
	}

	// The only possible match is "Chapter III" and "Chapter IV" via uscChapterArabicPattern
	// but those use Roman numerals, so they should NOT match the Arabic pattern
	for _, ref := range uscRefs {
		if ref.Target == TargetChapter && (ref.Identifier == "chapter III" || ref.Identifier == "chapter IV") {
			t.Errorf("USC chapter pattern should not match Roman numeral chapters, got %q", ref.RawText)
		}
	}
}

func TestUSCActReferenceResolution(t *testing.T) {
	resolver := NewReferenceResolver("https://regula.dev/usc42#", "USC42")

	ref := &Reference{
		Type:        ReferenceTypeExternal,
		Target:      TargetSection,
		RawText:     "section 306 of the Public Health Service Act",
		Identifier:  "306 of Public Health Service Act",
		ExternalDoc: "USAct",
		DocNumber:   "Public Health Service Act",
		SectionNum:  306,
	}

	result := resolver.Resolve(ref)

	if result.Status != ResolutionExternal {
		t.Errorf("expected status external, got %s", result.Status)
	}

	expectedURI := "urn:us:act:public-health-service-act/sec306"
	if result.TargetURI != expectedURI {
		t.Errorf("expected URI %q, got %q", expectedURI, result.TargetURI)
	}

	t.Logf("Act reference %q -> %s", ref.RawText, result.TargetURI)
}
