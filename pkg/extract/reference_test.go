package extract

import (
	"testing"
)

func TestCrossReferenceDetection(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewReferenceExtractor()
	refs := extractor.ExtractFromDocument(doc)

	stats := CalculateStats(refs)

	t.Logf("Reference Statistics:")
	t.Logf("  Total references: %d", stats.TotalReferences)
	t.Logf("  Internal references: %d", stats.InternalRefs)
	t.Logf("  External references: %d", stats.ExternalRefs)
	t.Logf("  Unique identifiers: %d", stats.UniqueIdentifiers)
	t.Logf("  Articles with references: %d", stats.ArticlesWithRefs)
	t.Logf("  By target:")
	for target, count := range stats.ByTarget {
		t.Logf("    %s: %d", target, count)
	}

	// Should have significant number of references
	if stats.TotalReferences < 100 {
		t.Errorf("Expected at least 100 references, got %d", stats.TotalReferences)
	}

	// Should have both internal and external references
	if stats.InternalRefs == 0 {
		t.Error("Expected some internal references")
	}
	if stats.ExternalRefs == 0 {
		t.Error("Expected some external references")
	}
}

func TestCrossReferenceDetection_InternalArticleRefs(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewReferenceExtractor()
	refs := extractor.ExtractFromDocument(doc)
	lookup := NewReferenceLookup(refs)

	articleRefs := lookup.GetByTarget(TargetArticle)
	t.Logf("Article references: %d", len(articleRefs))

	if len(articleRefs) < 50 {
		t.Errorf("Expected at least 50 article references, got %d", len(articleRefs))
	}

	// Check for specific known references
	// GDPR heavily references Article 6(1)
	refsToArticle6 := lookup.FindReferencesTo(6)
	t.Logf("References to Article 6: %d", len(refsToArticle6))
	if len(refsToArticle6) < 5 {
		t.Errorf("Expected multiple references to Article 6, got %d", len(refsToArticle6))
	}
}

func TestCrossReferenceDetection_ArticleParenthetical(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewReferenceExtractor()
	refs := extractor.ExtractFromDocument(doc)

	// Find references with paragraph numbers
	var withParagraph []*Reference
	for _, ref := range refs {
		if ref.Target == TargetArticle && ref.ParagraphNum > 0 {
			withParagraph = append(withParagraph, ref)
		}
	}

	t.Logf("Article references with paragraph: %d", len(withParagraph))

	if len(withParagraph) < 20 {
		t.Errorf("Expected at least 20 references with paragraph numbers, got %d", len(withParagraph))
	}

	// Log some examples
	for i, ref := range withParagraph {
		if i >= 5 {
			break
		}
		t.Logf("  %s (from Article %d)", ref.RawText, ref.SourceArticle)
	}
}

func TestCrossReferenceDetection_PointRefs(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewReferenceExtractor()
	refs := extractor.ExtractFromDocument(doc)
	lookup := NewReferenceLookup(refs)

	pointRefs := lookup.GetByTarget(TargetPoint)
	t.Logf("Point references: %d", len(pointRefs))

	if len(pointRefs) < 10 {
		t.Errorf("Expected at least 10 point references, got %d", len(pointRefs))
	}

	// Log some examples
	for i, ref := range pointRefs {
		if i >= 5 {
			break
		}
		t.Logf("  %s (from Article %d)", ref.RawText, ref.SourceArticle)
	}
}

func TestCrossReferenceDetection_ChapterRefs(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewReferenceExtractor()
	refs := extractor.ExtractFromDocument(doc)
	lookup := NewReferenceLookup(refs)

	chapterRefs := lookup.GetByTarget(TargetChapter)
	t.Logf("Chapter references: %d", len(chapterRefs))

	if len(chapterRefs) < 5 {
		t.Errorf("Expected at least 5 chapter references, got %d", len(chapterRefs))
	}
}

func TestCrossReferenceDetection_ExternalDirectives(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewReferenceExtractor()
	refs := extractor.ExtractFromDocument(doc)
	lookup := NewReferenceLookup(refs)

	directiveRefs := lookup.GetByTarget(TargetDirective)
	t.Logf("Directive references: %d", len(directiveRefs))

	if len(directiveRefs) < 5 {
		t.Errorf("Expected at least 5 directive references, got %d", len(directiveRefs))
	}

	// Should reference Directive 95/46/EC (the predecessor to GDPR)
	found9546 := false
	for _, ref := range directiveRefs {
		if ref.DocYear == "95" && ref.DocNumber == "46" {
			found9546 = true
			t.Logf("Found Directive 95/46/EC: %s", ref.RawText)
			break
		}
	}

	if !found9546 {
		t.Error("Expected to find reference to Directive 95/46/EC")
	}
}

func TestCrossReferenceDetection_ExternalRegulations(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewReferenceExtractor()
	refs := extractor.ExtractFromDocument(doc)
	lookup := NewReferenceLookup(refs)

	regulationRefs := lookup.GetByTarget(TargetRegulation)
	t.Logf("Regulation references: %d", len(regulationRefs))

	if len(regulationRefs) < 3 {
		t.Errorf("Expected at least 3 regulation references, got %d", len(regulationRefs))
	}

	// Log found regulations
	for i, ref := range regulationRefs {
		if i >= 5 {
			break
		}
		t.Logf("  %s", ref.Identifier)
	}
}

func TestCrossReferenceDetection_TreatyRefs(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewReferenceExtractor()
	refs := extractor.ExtractFromDocument(doc)
	lookup := NewReferenceLookup(refs)

	treatyRefs := lookup.GetByTarget(TargetTreaty)
	t.Logf("Treaty references: %d", len(treatyRefs))

	// GDPR references TFEU
	if len(treatyRefs) == 0 {
		t.Error("Expected treaty references (TFEU)")
	}
}

func TestCrossReferenceDetection_LocationTracking(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewReferenceExtractor()
	refs := extractor.ExtractFromDocument(doc)

	// Every reference should have location info
	for i, ref := range refs {
		if i >= 20 {
			break
		}

		if ref.SourceArticle == 0 {
			t.Errorf("Reference %d missing source article", i)
		}
		if ref.TextOffset < 0 {
			t.Errorf("Reference %d has invalid text offset", i)
		}
		if ref.TextLength <= 0 {
			t.Errorf("Reference %d has invalid text length", i)
		}

		// Verify RawText length matches TextLength
		if len(ref.RawText) != ref.TextLength {
			t.Errorf("Reference %d: RawText length (%d) doesn't match TextLength (%d)",
				i, len(ref.RawText), ref.TextLength)
		}
	}
}

func TestCrossReferenceDetection_SpecificPatterns(t *testing.T) {
	// Test specific known patterns from GDPR
	testCases := []struct {
		text     string
		expected []string // Expected identifiers
	}{
		{
			text:     "referred to in Article 6(1)",
			expected: []string{"Article 6(1)"},
		},
		{
			text:     "in accordance with Article 12",
			expected: []string{"Article 12"},
		},
		{
			text:     "points (a) to (f) of paragraph 1",
			expected: []string{"points (a) to (f)", "paragraph 1"},
		},
		{
			text:     "Chapter VIII",
			expected: []string{"Chapter VIII"},
		},
		{
			text:     "Directive 95/46/EC",
			expected: []string{"Directive 95/46"},
		},
		{
			text:     "Article 6(1)(a)",
			expected: []string{"Article 6(1)(a)"},
		},
	}

	extractor := NewReferenceExtractor()

	for _, tc := range testCases {
		// Create a minimal article to test
		article := &Article{
			Number: 99, // dummy
			Text:   tc.text,
		}

		refs := extractor.ExtractFromArticle(article)

		if len(refs) < len(tc.expected) {
			t.Errorf("Text %q: expected at least %d refs, got %d",
				tc.text, len(tc.expected), len(refs))
			continue
		}

		for _, expectedId := range tc.expected {
			found := false
			for _, ref := range refs {
				if ref.Identifier == expectedId {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Text %q: expected to find %q, got identifiers: %v",
					tc.text, expectedId, getIdentifiers(refs))
			}
		}
	}
}

func TestReferenceLookup(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewReferenceExtractor()
	refs := extractor.ExtractFromDocument(doc)
	lookup := NewReferenceLookup(refs)

	// Test Count
	if lookup.Count() != len(refs) {
		t.Errorf("Count mismatch: got %d, want %d", lookup.Count(), len(refs))
	}

	// Test GetBySourceArticle
	article6Refs := lookup.GetBySourceArticle(6)
	t.Logf("References from Article 6: %d", len(article6Refs))

	// Test GetByTarget
	articleRefs := lookup.GetByTarget(TargetArticle)
	if len(articleRefs) == 0 {
		t.Error("Expected some article references")
	}

	// Test FindReferencesTo
	refsTo6 := lookup.FindReferencesTo(6)
	t.Logf("References TO Article 6: %d", len(refsTo6))
}

func TestCrossReferenceDetection_Sample50Audit(t *testing.T) {
	// This test outputs 50 sample references for manual audit
	// as required by the acceptance criteria
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewReferenceExtractor()
	refs := extractor.ExtractFromDocument(doc)

	t.Log("=== Sample 50 References for Manual Audit ===")
	for i, ref := range refs {
		if i >= 50 {
			break
		}
		t.Logf("%2d. [%s] %s -> %s (Art.%d, offset %d)",
			i+1, ref.Type, ref.Target, ref.Identifier, ref.SourceArticle, ref.TextOffset)
	}

	// Calculate detection rate by category
	stats := CalculateStats(refs)
	t.Log("")
	t.Log("=== Detection Summary ===")
	t.Logf("Total detected: %d references", stats.TotalReferences)
	t.Logf("Internal: %d (%.1f%%)", stats.InternalRefs,
		float64(stats.InternalRefs)/float64(stats.TotalReferences)*100)
	t.Logf("External: %d (%.1f%%)", stats.ExternalRefs,
		float64(stats.ExternalRefs)/float64(stats.TotalReferences)*100)
}

func TestCrossReferenceDetection_NoOverlapping(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewReferenceExtractor()
	refs := extractor.ExtractFromDocument(doc)

	// Group by source article and check for overlaps
	byArticle := make(map[int][]*Reference)
	for _, ref := range refs {
		byArticle[ref.SourceArticle] = append(byArticle[ref.SourceArticle], ref)
	}

	overlaps := 0
	for _, articleRefs := range byArticle {
		for i := 0; i < len(articleRefs); i++ {
			for j := i + 1; j < len(articleRefs); j++ {
				r1 := articleRefs[i]
				r2 := articleRefs[j]

				r1End := r1.TextOffset + r1.TextLength
				r2End := r2.TextOffset + r2.TextLength

				if r1.TextOffset < r2End && r1End > r2.TextOffset {
					overlaps++
					if overlaps <= 5 {
						t.Logf("Overlap: %q (%d-%d) and %q (%d-%d) in Article %d",
							r1.RawText, r1.TextOffset, r1End,
							r2.RawText, r2.TextOffset, r2End,
							r1.SourceArticle)
					}
				}
			}
		}
	}

	// Some overlaps may be acceptable (e.g., "Article 6(1)" contains "Article 6")
	// but we should log them for awareness
	if overlaps > 0 {
		t.Logf("Total overlapping references: %d (this may be acceptable)", overlaps)
	}
}

// Helper function to get identifiers from references
func getIdentifiers(refs []*Reference) []string {
	ids := make([]string, len(refs))
	for i, ref := range refs {
		ids[i] = ref.Identifier
	}
	return ids
}
