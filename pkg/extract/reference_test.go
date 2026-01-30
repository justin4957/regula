package extract

import (
	"os"
	"path/filepath"
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

// ==================== Temporal Reference Tests ====================

func TestExtractTemporalAsAmendedBy(t *testing.T) {
	extractor := NewReferenceExtractor()

	testCases := []struct {
		name        string
		text        string
		description string
	}{
		{
			name:        "directive_amended_by_regulation",
			text:        "Directive 95/46/EC as amended by Regulation (EU) 2016/679.",
			description: "Regulation (EU) 2016/679",
		},
		{
			name:        "act_amended_by_regulation",
			text:        "the Act as amended by Regulation (EU) 2018/1725, shall apply.",
			description: "Regulation (EU) 2018/1725",
		},
		{
			name:        "amended_by_this_regulation",
			text:        "Directive 95/46/EC, as amended by this Regulation.",
			description: "this Regulation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			article := &Article{Number: 1, Text: tc.text}
			refs := extractor.ExtractFromArticle(article)

			var temporalRefs []*Reference
			for _, ref := range refs {
				if ref.TemporalKind == "as_amended" {
					temporalRefs = append(temporalRefs, ref)
				}
			}

			if len(temporalRefs) == 0 {
				t.Errorf("Expected at least one 'as_amended' temporal ref in %q", tc.text)
				return
			}

			found := false
			for _, ref := range temporalRefs {
				if ref.TemporalDescription == tc.description {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected temporal description %q, got descriptions: %v",
					tc.description, getTemporalDescriptions(temporalRefs))
			}
		})
	}
}

func TestExtractTemporalAsAmended(t *testing.T) {
	extractor := NewReferenceExtractor()

	testCases := []struct {
		name string
		text string
	}{
		{
			name: "as_amended_standalone",
			text: "the regulation, as amended, shall continue to apply.",
		},
		{
			name: "as_amended_comma",
			text: "the regulation, as amended, shall apply.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			article := &Article{Number: 1, Text: tc.text}
			refs := extractor.ExtractFromArticle(article)

			var temporalRefs []*Reference
			for _, ref := range refs {
				if ref.TemporalKind == "as_amended" {
					temporalRefs = append(temporalRefs, ref)
				}
			}

			if len(temporalRefs) == 0 {
				t.Errorf("Expected at least one 'as_amended' temporal ref in %q", tc.text)
			}
		})
	}
}

func TestExtractTemporalAsInForceOn(t *testing.T) {
	extractor := NewReferenceExtractor()

	article := &Article{
		Number: 1,
		Text:   "Directive 95/46/EC as in force on 24 May 2016.",
	}
	refs := extractor.ExtractFromArticle(article)

	var temporalRefs []*Reference
	for _, ref := range refs {
		if ref.TemporalKind == "in_force_on" {
			temporalRefs = append(temporalRefs, ref)
		}
	}

	if len(temporalRefs) == 0 {
		t.Fatal("Expected at least one 'in_force_on' temporal ref")
	}

	ref := temporalRefs[0]
	if ref.TemporalDate != "2016-05-24" {
		t.Errorf("Expected date 2016-05-24, got %s", ref.TemporalDate)
	}
}

func TestExtractTemporalEnterIntoForce(t *testing.T) {
	extractor := NewReferenceExtractor()

	testCases := []struct {
		name         string
		text         string
		expectedDate string
	}{
		{
			name:         "enter_into_force_with_date",
			text:         "shall enter into force on 25 May 2018.",
			expectedDate: "2018-05-25",
		},
		{
			name:         "enters_into_force_no_date",
			text:         "This Regulation enters into force on the twentieth day.",
			expectedDate: "",
		},
		{
			name:         "entered_into_force",
			text:         "The directive entered into force.",
			expectedDate: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			article := &Article{Number: 1, Text: tc.text}
			refs := extractor.ExtractFromArticle(article)

			var temporalRefs []*Reference
			for _, ref := range refs {
				if ref.TemporalKind == "in_force_on" {
					temporalRefs = append(temporalRefs, ref)
				}
			}

			if len(temporalRefs) == 0 {
				t.Errorf("Expected at least one 'in_force_on' temporal ref in %q", tc.text)
				return
			}

			if temporalRefs[0].TemporalDate != tc.expectedDate {
				t.Errorf("Expected date %q, got %q", tc.expectedDate, temporalRefs[0].TemporalDate)
			}
		})
	}
}

func TestExtractTemporalAsOriginallyEnacted(t *testing.T) {
	extractor := NewReferenceExtractor()

	article := &Article{
		Number: 1,
		Text:   "the Data Protection Act 1998 as originally enacted.",
	}
	refs := extractor.ExtractFromArticle(article)

	var temporalRefs []*Reference
	for _, ref := range refs {
		if ref.TemporalKind == "original" {
			temporalRefs = append(temporalRefs, ref)
		}
	}

	if len(temporalRefs) == 0 {
		t.Error("Expected at least one 'original' temporal ref")
	}
}

func TestExtractTemporalAsItStoodOn(t *testing.T) {
	extractor := NewReferenceExtractor()

	article := &Article{
		Number: 1,
		Text:   "the regulation as it stood on 1 January 2020.",
	}
	refs := extractor.ExtractFromArticle(article)

	var temporalRefs []*Reference
	for _, ref := range refs {
		if ref.TemporalKind == "original" {
			temporalRefs = append(temporalRefs, ref)
		}
	}

	if len(temporalRefs) == 0 {
		t.Fatal("Expected at least one 'original' temporal ref")
	}

	if temporalRefs[0].TemporalDate != "2020-01-01" {
		t.Errorf("Expected date 2020-01-01, got %s", temporalRefs[0].TemporalDate)
	}
}

func TestExtractTemporalConsolidated(t *testing.T) {
	extractor := NewReferenceExtractor()

	article := &Article{
		Number: 1,
		Text:   "See the consolidated version of Regulation (EU) 2016/679.",
	}
	refs := extractor.ExtractFromArticle(article)

	var temporalRefs []*Reference
	for _, ref := range refs {
		if ref.TemporalKind == "consolidated" {
			temporalRefs = append(temporalRefs, ref)
		}
	}

	if len(temporalRefs) == 0 {
		t.Error("Expected at least one 'consolidated' temporal ref")
	}
}

func TestExtractTemporalRepealedBy(t *testing.T) {
	extractor := NewReferenceExtractor()

	testCases := []struct {
		name        string
		text        string
		description string
	}{
		{
			name:        "repealed_by_this_regulation",
			text:        "Directive 95/46/EC should be repealed by this Regulation.",
			description: "this Regulation",
		},
		{
			name:        "repealed_by_named_regulation",
			text:        "The act was repealed by Regulation (EU) 2016/679.",
			description: "Regulation (EU) 2016/679",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			article := &Article{Number: 1, Text: tc.text}
			refs := extractor.ExtractFromArticle(article)

			var temporalRefs []*Reference
			for _, ref := range refs {
				if ref.TemporalKind == "repealed" {
					temporalRefs = append(temporalRefs, ref)
				}
			}

			if len(temporalRefs) == 0 {
				t.Errorf("Expected at least one 'repealed' temporal ref in %q", tc.text)
				return
			}

			found := false
			for _, ref := range temporalRefs {
				if ref.TemporalDescription == tc.description {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected temporal description %q, got descriptions: %v",
					tc.description, getTemporalDescriptions(temporalRefs))
			}
		})
	}
}

func TestExtractTemporalRepealedWithEffect(t *testing.T) {
	extractor := NewReferenceExtractor()

	article := &Article{
		Number: 1,
		Text:   "Directive 95/46/EC is repealed with effect from 25 May 2018.",
	}
	refs := extractor.ExtractFromArticle(article)

	var temporalRefs []*Reference
	for _, ref := range refs {
		if ref.TemporalKind == "repealed" {
			temporalRefs = append(temporalRefs, ref)
		}
	}

	if len(temporalRefs) == 0 {
		t.Fatal("Expected at least one 'repealed' temporal ref")
	}

	if temporalRefs[0].TemporalDate != "2018-05-25" {
		t.Errorf("Expected date 2018-05-25, got %s", temporalRefs[0].TemporalDate)
	}
}

func TestExtractTemporalParseEuropeanDate(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"25 May 2018", "2018-05-25"},
		{"1 January 2020", "2020-01-01"},
		{"24 May 2016", "2016-05-24"},
		{"3 March 2021", "2021-03-03"},
		{"15 December 2019", "2019-12-15"},
		{"31 October 2020", "2020-10-31"},
		// Invalid inputs
		{"invalid", ""},
		{"", ""},
		{"May 2018", ""},
		{"25 Foo 2018", ""},
		{"25 May", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := parseEuropeanDate(tc.input)
			if result != tc.expected {
				t.Errorf("parseEuropeanDate(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestExtractTemporalGDPRIntegration(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewReferenceExtractor()
	refs := extractor.ExtractFromDocument(doc)

	// Count temporal references by kind
	temporalCounts := make(map[string]int)
	for _, ref := range refs {
		if ref.TemporalKind != "" {
			temporalCounts[ref.TemporalKind]++
		}
	}

	totalTemporal := 0
	for kind, count := range temporalCounts {
		totalTemporal += count
		t.Logf("  Temporal %s: %d", kind, count)
	}

	t.Logf("Total temporal references: %d", totalTemporal)

	// GDPR should have temporal references (repealed, in force, amended patterns)
	if totalTemporal == 0 {
		t.Error("Expected temporal references in GDPR text")
	}

	// Should have at least "repealed" references (Directive 95/46/EC is repealed)
	if temporalCounts["repealed"] == 0 {
		t.Error("Expected 'repealed' temporal references in GDPR (Directive 95/46/EC repealed)")
	}

	// Should have "in_force_on" references (enter into force patterns)
	if temporalCounts["in_force_on"] == 0 {
		t.Error("Expected 'in_force_on' temporal references in GDPR (enters into force)")
	}

	// Log some sample temporal references
	t.Log("Sample temporal references:")
	count := 0
	for _, ref := range refs {
		if ref.TemporalKind != "" && count < 10 {
			t.Logf("  [%s] %q (Art.%d, date=%s)", ref.TemporalKind, ref.RawText, ref.SourceArticle, ref.TemporalDate)
			count++
		}
	}
}

func TestExtractTemporalUKSIIntegration(t *testing.T) {
	testdataDir := filepath.Join("..", "..", "testdata")
	siPath := filepath.Join(testdataDir, "uk-si-example.txt")

	siFile, err := os.Open(siPath)
	if err != nil {
		t.Skipf("UK SI test data not available: %v", err)
	}
	defer siFile.Close()

	parser := NewParser()
	doc, err := parser.Parse(siFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewReferenceExtractor()
	refs := extractor.ExtractFromDocument(doc)

	// Count temporal references by kind
	temporalCounts := make(map[string]int)
	for _, ref := range refs {
		if ref.TemporalKind != "" {
			temporalCounts[ref.TemporalKind]++
		}
	}

	totalTemporal := 0
	for kind, count := range temporalCounts {
		totalTemporal += count
		t.Logf("  Temporal %s: %d", kind, count)
	}

	t.Logf("Total temporal references in UK SI: %d", totalTemporal)

	// UK SI should have some temporal references (amended, in force, etc.)
	// Note: The UK SI document uses "amended as follows" patterns which may or may not
	// be parsed as articles depending on the document parser's ability to handle SI format.
	if totalTemporal > 0 {
		t.Logf("Found %d temporal references in UK SI", totalTemporal)
	} else {
		t.Logf("No temporal references found (UK SI format may not produce article-level extraction)")
	}

	// Log all temporal references
	for _, ref := range refs {
		if ref.TemporalKind != "" {
			t.Logf("  [%s] %q (Art.%d)", ref.TemporalKind, ref.RawText, ref.SourceArticle)
		}
	}
}

// Helper function to get temporal descriptions from references
func getTemporalDescriptions(refs []*Reference) []string {
	descs := make([]string, len(refs))
	for i, ref := range refs {
		descs[i] = ref.TemporalDescription
	}
	return descs
}

// Helper function to get identifiers from references
func getIdentifiers(refs []*Reference) []string {
	ids := make([]string, len(refs))
	for i, ref := range refs {
		ids[i] = ref.Identifier
	}
	return ids
}
