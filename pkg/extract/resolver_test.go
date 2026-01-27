package extract

import (
	"os"
	"strings"
	"testing"
)

func TestNewReferenceResolver(t *testing.T) {
	resolver := NewReferenceResolver("https://regula.dev/regulations/", "GDPR")
	if resolver == nil {
		t.Fatal("NewReferenceResolver returned nil")
	}
	if resolver.baseURI != "https://regula.dev/regulations/" {
		t.Errorf("baseURI = %q, want %q", resolver.baseURI, "https://regula.dev/regulations/")
	}
	if resolver.regID != "GDPR" {
		t.Errorf("regID = %q, want %q", resolver.regID, "GDPR")
	}
}

func TestNewReferenceResolver_AddsSuffix(t *testing.T) {
	resolver := NewReferenceResolver("https://example.com", "REG")
	if !strings.HasSuffix(resolver.baseURI, "#") && !strings.HasSuffix(resolver.baseURI, "/") {
		t.Errorf("baseURI should have suffix, got %q", resolver.baseURI)
	}
}

func TestReferenceResolver_IndexDocument(t *testing.T) {
	resolver := NewReferenceResolver("https://regula.dev/", "TEST")

	doc := &Document{
		Chapters: []*Chapter{
			{
				Number: "I",
				Articles: []*Article{
					{
						Number: 1,
						Paragraphs: []*Paragraph{
							{
								Number: 1,
								Points: []*Point{
									{Letter: "a"},
									{Letter: "b"},
								},
							},
							{Number: 2},
						},
					},
					{Number: 2},
				},
			},
			{
				Number: "II",
				Sections: []*Section{
					{
						Number: 1,
						Articles: []*Article{
							{Number: 3},
						},
					},
				},
			},
		},
	}

	resolver.IndexDocument(doc)

	// Check articles indexed
	if !resolver.articles[1] || !resolver.articles[2] || !resolver.articles[3] {
		t.Error("Articles not properly indexed")
	}

	// Check chapters indexed
	if !resolver.chapters["I"] || !resolver.chapters["II"] {
		t.Error("Chapters not properly indexed")
	}

	// Check sections indexed
	if !resolver.sections["II:1"] {
		t.Error("Section not properly indexed")
	}

	// Check paragraphs indexed
	if !resolver.paragraphs["1:1"] || !resolver.paragraphs["1:2"] {
		t.Error("Paragraphs not properly indexed")
	}

	// Check points indexed
	if !resolver.points["1:1:a"] || !resolver.points["1:1:b"] {
		t.Error("Points not properly indexed")
	}

	// Check article-chapter mapping
	if resolver.articleChapter[1] != "I" || resolver.articleChapter[2] != "I" || resolver.articleChapter[3] != "II" {
		t.Error("Article-chapter mapping incorrect")
	}
}

func TestReferenceResolver_ResolveArticle_Found(t *testing.T) {
	resolver := NewReferenceResolver("https://regula.dev/", "GDPR")
	resolver.articles[17] = true

	ref := &Reference{
		Type:          ReferenceTypeInternal,
		Target:        TargetArticle,
		RawText:       "Article 17",
		ArticleNum:    17,
		SourceArticle: 5,
	}

	result := resolver.Resolve(ref)

	if result.Status != ResolutionResolved {
		t.Errorf("Status = %v, want %v", result.Status, ResolutionResolved)
	}
	if result.Confidence != ConfidenceHigh {
		t.Errorf("Confidence = %v, want %v", result.Confidence, ConfidenceHigh)
	}
	expectedURI := "https://regula.dev/GDPR:Art17"
	if result.TargetURI != expectedURI {
		t.Errorf("TargetURI = %q, want %q", result.TargetURI, expectedURI)
	}
}

func TestReferenceResolver_ResolveArticle_NotFound(t *testing.T) {
	resolver := NewReferenceResolver("https://regula.dev/", "GDPR")
	// Article 999 not indexed

	ref := &Reference{
		Type:          ReferenceTypeInternal,
		Target:        TargetArticle,
		RawText:       "Article 999",
		ArticleNum:    999,
		SourceArticle: 5,
	}

	result := resolver.Resolve(ref)

	if result.Status != ResolutionNotFound {
		t.Errorf("Status = %v, want %v", result.Status, ResolutionNotFound)
	}
	if result.Confidence != ConfidenceNone {
		t.Errorf("Confidence = %v, want %v", result.Confidence, ConfidenceNone)
	}
}

func TestReferenceResolver_ResolveArticleWithParagraph(t *testing.T) {
	resolver := NewReferenceResolver("https://regula.dev/", "GDPR")
	resolver.articles[6] = true
	resolver.paragraphs["6:1"] = true

	ref := &Reference{
		Type:          ReferenceTypeInternal,
		Target:        TargetArticle,
		RawText:       "Article 6(1)",
		ArticleNum:    6,
		ParagraphNum:  1,
		SourceArticle: 7,
	}

	result := resolver.Resolve(ref)

	if result.Status != ResolutionResolved {
		t.Errorf("Status = %v, want %v", result.Status, ResolutionResolved)
	}
	expectedURI := "https://regula.dev/GDPR:Art6(1)"
	if result.TargetURI != expectedURI {
		t.Errorf("TargetURI = %q, want %q", result.TargetURI, expectedURI)
	}
}

func TestReferenceResolver_ResolveArticleWithPoint(t *testing.T) {
	resolver := NewReferenceResolver("https://regula.dev/", "GDPR")
	resolver.articles[6] = true
	resolver.paragraphs["6:1"] = true
	resolver.points["6:1:a"] = true

	ref := &Reference{
		Type:          ReferenceTypeInternal,
		Target:        TargetArticle,
		RawText:       "Article 6(1)(a)",
		ArticleNum:    6,
		ParagraphNum:  1,
		PointLetter:   "a",
		SourceArticle: 7,
	}

	result := resolver.Resolve(ref)

	if result.Status != ResolutionResolved {
		t.Errorf("Status = %v, want %v", result.Status, ResolutionResolved)
	}
	expectedURI := "https://regula.dev/GDPR:Art6(1)(a)"
	if result.TargetURI != expectedURI {
		t.Errorf("TargetURI = %q, want %q", result.TargetURI, expectedURI)
	}
}

func TestReferenceResolver_ResolveArticle_PartialParagraph(t *testing.T) {
	resolver := NewReferenceResolver("https://regula.dev/", "GDPR")
	resolver.articles[6] = true
	// Paragraph 99 doesn't exist

	ref := &Reference{
		Type:          ReferenceTypeInternal,
		Target:        TargetArticle,
		RawText:       "Article 6(99)",
		ArticleNum:    6,
		ParagraphNum:  99,
		SourceArticle: 7,
	}

	result := resolver.Resolve(ref)

	if result.Status != ResolutionPartial {
		t.Errorf("Status = %v, want %v", result.Status, ResolutionPartial)
	}
	if result.Confidence != ConfidenceMedium {
		t.Errorf("Confidence = %v, want %v", result.Confidence, ConfidenceMedium)
	}
}

func TestReferenceResolver_ResolveArticleRange(t *testing.T) {
	resolver := NewReferenceResolver("https://regula.dev/", "GDPR")
	resolver.articles[13] = true
	resolver.articles[14] = true
	resolver.articles[15] = true

	ref := &Reference{
		Type:          ReferenceTypeInternal,
		Target:        TargetArticle,
		RawText:       "Articles 13 to 15",
		Identifier:    "Articles 13-15",
		ArticleNum:    13,
		SubRef:        "range",
		SourceArticle: 7,
	}

	result := resolver.Resolve(ref)

	if result.Status != ResolutionRangeRef {
		t.Errorf("Status = %v, want %v", result.Status, ResolutionRangeRef)
	}
	if len(result.TargetURIs) != 3 {
		t.Errorf("TargetURIs count = %d, want 3", len(result.TargetURIs))
	}
}

func TestReferenceResolver_ResolveChapter(t *testing.T) {
	resolver := NewReferenceResolver("https://regula.dev/", "GDPR")
	resolver.chapters["III"] = true

	ref := &Reference{
		Type:          ReferenceTypeInternal,
		Target:        TargetChapter,
		RawText:       "Chapter III",
		ChapterNum:    "III",
		SourceArticle: 5,
	}

	result := resolver.Resolve(ref)

	if result.Status != ResolutionResolved {
		t.Errorf("Status = %v, want %v", result.Status, ResolutionResolved)
	}
	expectedURI := "https://regula.dev/GDPR:ChapterIII"
	if result.TargetURI != expectedURI {
		t.Errorf("TargetURI = %q, want %q", result.TargetURI, expectedURI)
	}
}

func TestReferenceResolver_ResolveExternalReference(t *testing.T) {
	resolver := NewReferenceResolver("https://regula.dev/", "GDPR")

	ref := &Reference{
		Type:          ReferenceTypeExternal,
		Target:        TargetDirective,
		RawText:       "Directive 95/46/EC",
		Identifier:    "Directive 95/46",
		ExternalDoc:   "Directive",
		DocYear:       "95",
		DocNumber:     "46",
		SourceArticle: 5,
	}

	result := resolver.Resolve(ref)

	if result.Status != ResolutionExternal {
		t.Errorf("Status = %v, want %v", result.Status, ResolutionExternal)
	}
	if result.Confidence != ConfidenceHigh {
		t.Errorf("Confidence = %v, want %v", result.Confidence, ConfidenceHigh)
	}
	if !strings.Contains(result.TargetURI, "directive") {
		t.Errorf("TargetURI should contain 'directive', got %q", result.TargetURI)
	}
}

func TestReferenceResolver_RelativeParagraphRef(t *testing.T) {
	resolver := NewReferenceResolver("https://regula.dev/", "GDPR")
	resolver.articles[5] = true
	resolver.paragraphs["5:2"] = true

	ref := &Reference{
		Type:          ReferenceTypeInternal,
		Target:        TargetParagraph,
		RawText:       "paragraph 2",
		ArticleNum:    0, // Relative - no article specified
		ParagraphNum:  2,
		SourceArticle: 5, // Source article provides context
	}

	result := resolver.Resolve(ref)

	if result.Status != ResolutionResolved {
		t.Errorf("Status = %v, want %v", result.Status, ResolutionResolved)
	}
	expectedURI := "https://regula.dev/GDPR:Art5(2)"
	if result.TargetURI != expectedURI {
		t.Errorf("TargetURI = %q, want %q", result.TargetURI, expectedURI)
	}
}

func TestGenerateReport(t *testing.T) {
	resolved := []*ResolvedReference{
		{Status: ResolutionResolved, Confidence: ConfidenceHigh, Original: &Reference{Type: ReferenceTypeInternal}},
		{Status: ResolutionResolved, Confidence: ConfidenceHigh, Original: &Reference{Type: ReferenceTypeInternal}},
		{Status: ResolutionPartial, Confidence: ConfidenceMedium, Original: &Reference{Type: ReferenceTypeInternal}},
		{Status: ResolutionNotFound, Confidence: ConfidenceNone, Original: &Reference{Type: ReferenceTypeInternal, SourceArticle: 1, RawText: "test"}},
		{Status: ResolutionExternal, Confidence: ConfidenceHigh, Original: &Reference{Type: ReferenceTypeExternal}},
	}

	report := GenerateReport(resolved)

	if report.TotalReferences != 5 {
		t.Errorf("TotalReferences = %d, want 5", report.TotalReferences)
	}
	if report.Resolved != 2 {
		t.Errorf("Resolved = %d, want 2", report.Resolved)
	}
	if report.Partial != 1 {
		t.Errorf("Partial = %d, want 1", report.Partial)
	}
	if report.NotFound != 1 {
		t.Errorf("NotFound = %d, want 1", report.NotFound)
	}
	if report.External != 1 {
		t.Errorf("External = %d, want 1", report.External)
	}
	if report.HighConfidence != 3 {
		t.Errorf("HighConfidence = %d, want 3", report.HighConfidence)
	}

	// Resolution rate: (2 resolved + 1 partial) / 4 internal = 75%
	expectedRate := 0.75
	if report.ResolutionRate != expectedRate {
		t.Errorf("ResolutionRate = %.2f, want %.2f", report.ResolutionRate, expectedRate)
	}
}

func TestResolutionReport_String(t *testing.T) {
	report := &ResolutionReport{
		TotalReferences:  10,
		Resolved:         6,
		Partial:          1,
		NotFound:         1,
		External:         2,
		HighConfidence:   7,
		MediumConfidence: 2,
		LowConfidence:    1,
		ResolutionRate:   0.875,
		ConfidenceRate:   0.7,
	}

	str := report.String()

	if !strings.Contains(str, "Reference Resolution Report") {
		t.Error("Report should contain title")
	}
	if !strings.Contains(str, "Total references: 10") {
		t.Error("Report should contain total count")
	}
	if !strings.Contains(str, "Resolved:   6") {
		t.Error("Report should contain resolved count")
	}
	if !strings.Contains(str, "87.5%") {
		t.Error("Report should contain resolution rate")
	}
}

// Integration test with GDPR data
func TestGDPRReferenceResolution(t *testing.T) {
	gdprPath := "../../testdata/gdpr.txt"
	if _, err := os.Stat(gdprPath); os.IsNotExist(err) {
		gdprPath = "testdata/gdpr.txt"
		if _, err := os.Stat(gdprPath); os.IsNotExist(err) {
			t.Skip("GDPR test data not available")
		}
	}

	file, err := os.Open(gdprPath)
	if err != nil {
		t.Fatalf("Failed to open GDPR: %v", err)
	}
	defer file.Close()

	parser := NewParser()
	doc, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Failed to parse GDPR: %v", err)
	}

	// Extract references
	extractor := NewReferenceExtractor()
	refs := extractor.ExtractFromDocument(doc)

	t.Logf("Extracted %d references", len(refs))

	// Create and index resolver
	resolver := NewReferenceResolver("https://regula.dev/regulations/", "GDPR")
	resolver.IndexDocument(doc)

	// Resolve all references
	resolved := resolver.ResolveAll(refs)

	// Generate report
	report := GenerateReport(resolved)

	t.Logf("Resolution Report:")
	t.Logf("  Total: %d", report.TotalReferences)
	t.Logf("  Resolved: %d", report.Resolved)
	t.Logf("  Partial: %d", report.Partial)
	t.Logf("  Range refs: %d", report.RangeRef)
	t.Logf("  Ambiguous: %d", report.Ambiguous)
	t.Logf("  Not found: %d", report.NotFound)
	t.Logf("  External: %d", report.External)
	t.Logf("  Resolution rate: %.1f%%", report.ResolutionRate*100)
	t.Logf("  High confidence: %.1f%%", report.ConfidenceRate*100)

	// Acceptance criteria: ≥85% resolution rate
	if report.ResolutionRate < 0.85 {
		t.Errorf("Resolution rate %.1f%% is below 85%% target", report.ResolutionRate*100)
	}

	// Log some unresolved refs for debugging
	if len(report.UnresolvedRefs) > 0 {
		t.Logf("\nSample unresolved references (max 10):")
		for i, ref := range report.UnresolvedRefs {
			if i >= 10 {
				break
			}
			t.Logf("  - Article %d: %q - %s",
				ref.Original.SourceArticle, ref.Original.RawText, ref.Reason)
		}
	}
}

func TestReferenceResolver_ResolveSection(t *testing.T) {
	resolver := NewReferenceResolver("https://regula.dev/", "GDPR")
	resolver.chapters["III"] = true
	resolver.sections["III:1"] = true
	resolver.articleChapter[20] = "III"

	ref := &Reference{
		Type:          ReferenceTypeInternal,
		Target:        TargetSection,
		RawText:       "Section 1",
		SectionNum:    1,
		SourceArticle: 20, // Article in Chapter III
	}

	result := resolver.Resolve(ref)

	if result.Status != ResolutionResolved {
		t.Errorf("Status = %v, want %v", result.Status, ResolutionResolved)
	}
	expectedURI := "https://regula.dev/GDPR:ChapterIII:Section1"
	if result.TargetURI != expectedURI {
		t.Errorf("TargetURI = %q, want %q", result.TargetURI, expectedURI)
	}
}

func TestReferenceResolver_ResolveSectionAmbiguous(t *testing.T) {
	resolver := NewReferenceResolver("https://regula.dev/", "GDPR")
	resolver.chapters["III"] = true
	resolver.chapters["IV"] = true
	resolver.sections["III:1"] = true
	resolver.sections["IV:1"] = true // Same section number in different chapters

	ref := &Reference{
		Type:          ReferenceTypeInternal,
		Target:        TargetSection,
		RawText:       "Section 1",
		SectionNum:    1,
		SourceArticle: 99, // No chapter context
	}

	result := resolver.Resolve(ref)

	// Should find it but mark as ambiguous
	if result.Status != ResolutionAmbiguous {
		t.Errorf("Status = %v, want %v", result.Status, ResolutionAmbiguous)
	}
	if len(result.AlternativeURIs) == 0 {
		t.Error("Should have alternative URIs for ambiguous section")
	}
}

func TestReferenceResolver_PointRange(t *testing.T) {
	resolver := NewReferenceResolver("https://regula.dev/", "GDPR")
	resolver.articles[6] = true
	resolver.paragraphs["6:1"] = true
	resolver.points["6:1:a"] = true
	resolver.points["6:1:b"] = true
	resolver.points["6:1:c"] = true

	ref := &Reference{
		Type:          ReferenceTypeInternal,
		Target:        TargetPoint,
		RawText:       "points (a) to (c)",
		ArticleNum:    6,
		ParagraphNum:  1,
		PointLetter:   "a",
		SubRef:        "range",
		SourceArticle: 7,
	}

	result := resolver.Resolve(ref)

	if result.Status != ResolutionRangeRef {
		t.Errorf("Status = %v, want %v", result.Status, ResolutionRangeRef)
	}
	if len(result.TargetURIs) != 3 {
		t.Errorf("TargetURIs count = %d, want 3", len(result.TargetURIs))
	}
}

func BenchmarkReferenceResolver_ResolveAll(b *testing.B) {
	gdprPath := "../../testdata/gdpr.txt"
	if _, err := os.Stat(gdprPath); os.IsNotExist(err) {
		gdprPath = "testdata/gdpr.txt"
		if _, err := os.Stat(gdprPath); os.IsNotExist(err) {
			b.Skip("GDPR test data not available")
		}
	}

	file, err := os.Open(gdprPath)
	if err != nil {
		b.Fatalf("Failed to open GDPR: %v", err)
	}
	defer file.Close()

	parser := NewParser()
	doc, err := parser.Parse(file)
	if err != nil {
		b.Fatalf("Failed to parse GDPR: %v", err)
	}

	extractor := NewReferenceExtractor()
	refs := extractor.ExtractFromDocument(doc)

	resolver := NewReferenceResolver("https://regula.dev/regulations/", "GDPR")
	resolver.IndexDocument(doc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolver.ResolveAll(refs)
	}
}
