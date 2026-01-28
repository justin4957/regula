package citation

import (
	"testing"

	"github.com/coolbeans/regula/pkg/extract"
)

func TestCitationFromReference(t *testing.T) {
	cases := []struct {
		name                string
		reference           *extract.Reference
		expectedType        CitationType
		expectedJurisdiction string
	}{
		{
			name: "directive_reference",
			reference: &extract.Reference{
				Type:        extract.ReferenceTypeExternal,
				Target:      extract.TargetDirective,
				RawText:     "Directive 95/46/EC",
				Identifier:  "Directive 95/46",
				ExternalDoc: "Directive",
				DocYear:     "95",
				DocNumber:   "46",
				TextOffset:  10,
				TextLength:  18,
			},
			expectedType:        CitationTypeDirective,
			expectedJurisdiction: "EU",
		},
		{
			name: "regulation_reference",
			reference: &extract.Reference{
				Type:        extract.ReferenceTypeExternal,
				Target:      extract.TargetRegulation,
				RawText:     "Regulation (EU) 2016/679",
				Identifier:  "Regulation 2016/679",
				ExternalDoc: "Regulation",
				DocYear:     "2016",
				DocNumber:   "679",
			},
			expectedType:        CitationTypeRegulation,
			expectedJurisdiction: "EU",
		},
		{
			name: "decision_reference",
			reference: &extract.Reference{
				Type:    extract.ReferenceTypeExternal,
				Target:  extract.TargetDecision,
				RawText: "Decision 2010/87/EU",
				DocYear: "2010",
				DocNumber: "87",
			},
			expectedType:        CitationTypeDecision,
			expectedJurisdiction: "EU",
		},
		{
			name: "treaty_reference",
			reference: &extract.Reference{
				Type:       extract.ReferenceTypeExternal,
				Target:     extract.TargetTreaty,
				RawText:    "TFEU",
				Identifier: "TFEU",
			},
			expectedType:        CitationTypeTreaty,
			expectedJurisdiction: "EU",
		},
		{
			name: "article_reference",
			reference: &extract.Reference{
				Type:         extract.ReferenceTypeInternal,
				Target:       extract.TargetArticle,
				RawText:      "Article 6(1)(a)",
				Identifier:   "Art6(1)(a)",
				ArticleNum:   6,
				ParagraphNum: 1,
				PointLetter:  "a",
				TextOffset:   42,
				TextLength:   15,
			},
			expectedType:        CitationTypeStatute,
			expectedJurisdiction: "",
		},
		{
			name: "chapter_reference",
			reference: &extract.Reference{
				Type:       extract.ReferenceTypeInternal,
				Target:     extract.TargetChapter,
				RawText:    "Chapter III",
				Identifier: "Chapter III",
				ChapterNum: "III",
			},
			expectedType:        CitationTypeStatute,
			expectedJurisdiction: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			citation := CitationFromReference(tc.reference)

			if citation.Type != tc.expectedType {
				t.Errorf("Type: got %q, want %q", citation.Type, tc.expectedType)
			}
			if citation.Jurisdiction != tc.expectedJurisdiction {
				t.Errorf("Jurisdiction: got %q, want %q", citation.Jurisdiction, tc.expectedJurisdiction)
			}
			if citation.RawText != tc.reference.RawText {
				t.Errorf("RawText: got %q, want %q", citation.RawText, tc.reference.RawText)
			}
			if citation.TextOffset != tc.reference.TextOffset {
				t.Errorf("TextOffset: got %d, want %d", citation.TextOffset, tc.reference.TextOffset)
			}
			if citation.TextLength != tc.reference.TextLength {
				t.Errorf("TextLength: got %d, want %d", citation.TextLength, tc.reference.TextLength)
			}
			if citation.Parser != "legacy-reference-extractor" {
				t.Errorf("Parser: got %q, want 'legacy-reference-extractor'", citation.Parser)
			}
		})
	}
}

func TestReferenceFromCitation(t *testing.T) {
	citation := &Citation{
		RawText:      "Directive 95/46/EC",
		Type:         CitationTypeDirective,
		Jurisdiction: "EU",
		Document:     "Directive 95/46/EC",
		Subdivision:  "Directive 95/46",
		Confidence:   1.0,
		Parser:       "EU Citation Parser",
		TextOffset:   20,
		TextLength:   18,
		Components: CitationComponents{
			DocYear:   "95",
			DocNumber: "46",
		},
	}

	ref := ReferenceFromCitation(citation, 5)

	if ref.Type != extract.ReferenceTypeExternal {
		t.Errorf("Type: got %q, want %q", ref.Type, extract.ReferenceTypeExternal)
	}
	if ref.Target != extract.TargetDirective {
		t.Errorf("Target: got %q, want %q", ref.Target, extract.TargetDirective)
	}
	if ref.RawText != "Directive 95/46/EC" {
		t.Errorf("RawText: got %q, want 'Directive 95/46/EC'", ref.RawText)
	}
	if ref.SourceArticle != 5 {
		t.Errorf("SourceArticle: got %d, want 5", ref.SourceArticle)
	}
	if ref.DocYear != "95" {
		t.Errorf("DocYear: got %q, want '95'", ref.DocYear)
	}
	if ref.DocNumber != "46" {
		t.Errorf("DocNumber: got %q, want '46'", ref.DocNumber)
	}
	if ref.TextOffset != 20 {
		t.Errorf("TextOffset: got %d, want 20", ref.TextOffset)
	}
}

func TestRoundtripConversion(t *testing.T) {
	originalRef := &extract.Reference{
		Type:          extract.ReferenceTypeInternal,
		Target:        extract.TargetArticle,
		RawText:       "Article 6(1)(a)",
		Identifier:    "Art6(1)(a)",
		SourceArticle: 3,
		TextOffset:    100,
		TextLength:    15,
		ArticleNum:    6,
		ParagraphNum:  1,
		PointLetter:   "a",
		ChapterNum:    "",
		DocYear:       "",
		DocNumber:     "",
	}

	// Reference -> Citation -> Reference
	citation := CitationFromReference(originalRef)
	restoredRef := ReferenceFromCitation(citation, originalRef.SourceArticle)

	// Verify key fields are preserved.
	if restoredRef.RawText != originalRef.RawText {
		t.Errorf("RawText roundtrip: got %q, want %q", restoredRef.RawText, originalRef.RawText)
	}
	if restoredRef.ArticleNum != originalRef.ArticleNum {
		t.Errorf("ArticleNum roundtrip: got %d, want %d", restoredRef.ArticleNum, originalRef.ArticleNum)
	}
	if restoredRef.ParagraphNum != originalRef.ParagraphNum {
		t.Errorf("ParagraphNum roundtrip: got %d, want %d", restoredRef.ParagraphNum, originalRef.ParagraphNum)
	}
	if restoredRef.PointLetter != originalRef.PointLetter {
		t.Errorf("PointLetter roundtrip: got %q, want %q", restoredRef.PointLetter, originalRef.PointLetter)
	}
	if restoredRef.TextOffset != originalRef.TextOffset {
		t.Errorf("TextOffset roundtrip: got %d, want %d", restoredRef.TextOffset, originalRef.TextOffset)
	}
	if restoredRef.TextLength != originalRef.TextLength {
		t.Errorf("TextLength roundtrip: got %d, want %d", restoredRef.TextLength, originalRef.TextLength)
	}
	if restoredRef.SourceArticle != originalRef.SourceArticle {
		t.Errorf("SourceArticle roundtrip: got %d, want %d", restoredRef.SourceArticle, originalRef.SourceArticle)
	}
}

func TestBatchCitationsFromReferences(t *testing.T) {
	refs := []*extract.Reference{
		{
			Type:       extract.ReferenceTypeExternal,
			Target:     extract.TargetDirective,
			RawText:    "Directive 95/46/EC",
			DocYear:    "95",
			DocNumber:  "46",
			TextOffset: 0,
			TextLength: 18,
		},
		{
			Type:       extract.ReferenceTypeInternal,
			Target:     extract.TargetArticle,
			RawText:    "Article 5",
			ArticleNum: 5,
			TextOffset: 30,
			TextLength: 9,
		},
	}

	citations := BatchCitationsFromReferences(refs)

	if len(citations) != 2 {
		t.Fatalf("Expected 2 citations, got %d", len(citations))
	}
	if citations[0].Type != CitationTypeDirective {
		t.Errorf("First citation type: got %q, want 'directive'", citations[0].Type)
	}
	if citations[1].Type != CitationTypeStatute {
		t.Errorf("Second citation type: got %q, want 'statute'", citations[1].Type)
	}
}

func TestBatchReferencesFromCitations(t *testing.T) {
	citations := []*Citation{
		{
			RawText:    "Directive 95/46/EC",
			Type:       CitationTypeDirective,
			TextOffset: 0,
			TextLength: 18,
			Components: CitationComponents{DocYear: "95", DocNumber: "46"},
		},
		{
			RawText:    "Article 5",
			Type:       CitationTypeStatute,
			TextOffset: 30,
			TextLength: 9,
			Components: CitationComponents{ArticleNumber: 5},
		},
	}

	refs := BatchReferencesFromCitations(citations, 1)

	if len(refs) != 2 {
		t.Fatalf("Expected 2 references, got %d", len(refs))
	}
	if refs[0].Target != extract.TargetDirective {
		t.Errorf("First ref target: got %q, want 'directive'", refs[0].Target)
	}
	if refs[0].SourceArticle != 1 {
		t.Errorf("First ref SourceArticle: got %d, want 1", refs[0].SourceArticle)
	}
	if refs[1].ArticleNum != 5 {
		t.Errorf("Second ref ArticleNum: got %d, want 5", refs[1].ArticleNum)
	}
}
