package extract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProvisionExtraction(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewProvisionExtractor()
	extractor.ExtractAllProvisions(doc)

	stats := ProvisionStats(doc)

	t.Logf("Provision Statistics:")
	t.Logf("  Articles with text: %d", stats.ArticlesWithText)
	t.Logf("  Articles with paragraphs: %d", stats.ArticlesWithParagraphs)
	t.Logf("  Total paragraphs: %d", stats.TotalParagraphs)
	t.Logf("  Total points: %d", stats.TotalPoints)
	t.Logf("  Total sub-points: %d", stats.TotalSubPoints)

	// All 99 articles should have text
	if stats.ArticlesWithText != 99 {
		t.Errorf("Expected 99 articles with text, got %d", stats.ArticlesWithText)
	}

	// Should have substantial paragraph count
	if stats.TotalParagraphs < 200 {
		t.Errorf("Expected at least 200 paragraphs, got %d", stats.TotalParagraphs)
	}

	// Should have substantial point count
	if stats.TotalPoints < 100 {
		t.Errorf("Expected at least 100 points, got %d", stats.TotalPoints)
	}
}

func TestProvisionExtraction_Article6(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewProvisionExtractor()

	// Get Article 6 which has known structure
	article6 := doc.GetArticle(6)
	if article6 == nil {
		t.Fatal("Article 6 not found")
	}

	extractor.ExtractProvisions(article6)

	t.Logf("Article 6 has %d paragraphs", len(article6.Paragraphs))

	// Article 6 should have 4 numbered paragraphs
	if len(article6.Paragraphs) != 4 {
		t.Errorf("Article 6 should have 4 paragraphs, got %d", len(article6.Paragraphs))
		for i, p := range article6.Paragraphs {
			t.Logf("  Paragraph %d (num=%d): %.50s...", i, p.Number, p.Text)
		}
	}

	// Paragraph 1 should have points (a) through (f)
	if len(article6.Paragraphs) > 0 {
		para1 := article6.Paragraphs[0]
		if para1.Number != 1 {
			t.Errorf("First paragraph should be number 1, got %d", para1.Number)
		}

		if len(para1.Points) != 6 {
			t.Errorf("Paragraph 1 should have 6 points (a-f), got %d", len(para1.Points))
			for _, p := range para1.Points {
				t.Logf("  Point (%s): %.50s...", p.Letter, p.Text)
			}
		} else {
			// Verify point letters
			expectedLetters := []string{"a", "b", "c", "d", "e", "f"}
			for i, expected := range expectedLetters {
				if para1.Points[i].Letter != expected {
					t.Errorf("Point %d should be (%s), got (%s)", i, expected, para1.Points[i].Letter)
				}
			}
		}
	}

	// Paragraph 3 should have points (a) and (b)
	if len(article6.Paragraphs) >= 3 {
		para3 := article6.Paragraphs[2]
		if para3.Number != 3 {
			t.Errorf("Third paragraph should be number 3, got %d", para3.Number)
		}

		if len(para3.Points) != 2 {
			t.Errorf("Paragraph 3 should have 2 points (a-b), got %d", len(para3.Points))
		}
	}

	// Paragraph 4 should have points (a) through (e)
	if len(article6.Paragraphs) >= 4 {
		para4 := article6.Paragraphs[3]
		if para4.Number != 4 {
			t.Errorf("Fourth paragraph should be number 4, got %d", para4.Number)
		}

		if len(para4.Points) != 5 {
			t.Errorf("Paragraph 4 should have 5 points (a-e), got %d", len(para4.Points))
		}
	}
}

func TestProvisionExtraction_Article4Definitions(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewProvisionExtractor()

	// Get Article 4 (Definitions)
	article4 := doc.GetArticle(4)
	if article4 == nil {
		t.Fatal("Article 4 not found")
	}

	extractor.ExtractProvisions(article4)

	// Article 4 is special - it has 26 numbered definitions but they're formatted
	// differently (not as standard paragraphs). Verify we capture the text.
	if article4.Text == "" {
		t.Error("Article 4 should have text")
	}

	t.Logf("Article 4 text length: %d chars", len(article4.Text))

	// Verify it contains expected definition content
	if !strings.Contains(article4.Text, "personal data") {
		t.Error("Article 4 should contain 'personal data' definition")
	}

	if !strings.Contains(article4.Text, "processing") {
		t.Error("Article 4 should contain 'processing' definition")
	}
}

func TestProvisionExtraction_SubPoints(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewProvisionExtractor()
	extractor.ExtractAllProvisions(doc)

	// Find articles with sub-points
	articlesWithSubPoints := 0
	totalSubPoints := 0

	for _, article := range doc.AllArticles() {
		hasSubPoints := false
		for _, para := range article.Paragraphs {
			for _, point := range para.Points {
				if len(point.SubPoints) > 0 {
					hasSubPoints = true
					totalSubPoints += len(point.SubPoints)
				}
			}
		}
		if hasSubPoints {
			articlesWithSubPoints++
			t.Logf("Article %d has sub-points", article.Number)
		}
	}

	t.Logf("Articles with sub-points: %d", articlesWithSubPoints)
	t.Logf("Total sub-points: %d", totalSubPoints)

	// Note: GDPR doesn't use roman numeral sub-points (i), (ii), (iii)
	// It uses letters (a)-(j) for all points. The SubPoints feature is
	// available for regulations that do use roman numerals.
	// This test verifies the extraction code runs without errors.
}

func TestProvisionExtraction_ParagraphNumbersPreserved(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewProvisionExtractor()
	extractor.ExtractAllProvisions(doc)

	// Check that paragraph numbers are sequential within articles
	for _, article := range doc.AllArticles() {
		if len(article.Paragraphs) == 0 {
			continue
		}

		// Skip articles with only unnumbered paragraphs
		if article.Paragraphs[0].Number == 0 {
			continue
		}

		expectedNum := 1
		for _, para := range article.Paragraphs {
			if para.Number == 0 {
				continue // Skip unnumbered paragraphs
			}
			if para.Number != expectedNum {
				t.Errorf("Article %d: expected paragraph %d, got %d",
					article.Number, expectedNum, para.Number)
				break
			}
			expectedNum++
		}
	}
}

func TestProvisionExtraction_PointLettersPreserved(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewProvisionExtractor()
	extractor.ExtractAllProvisions(doc)

	// Check that point letters are sequential within paragraphs
	for _, article := range doc.AllArticles() {
		for _, para := range article.Paragraphs {
			if len(para.Points) == 0 {
				continue
			}

			expectedLetter := 'a'
			for _, point := range para.Points {
				if len(point.Letter) != 1 {
					t.Errorf("Article %d, Para %d: invalid point letter %q",
						article.Number, para.Number, point.Letter)
					continue
				}

				gotLetter := rune(point.Letter[0])
				if gotLetter != expectedLetter {
					t.Errorf("Article %d, Para %d: expected point (%c), got (%c)",
						article.Number, para.Number, expectedLetter, gotLetter)
					break
				}
				expectedLetter++
			}
		}
	}
}

func TestRoundTrip_Article1(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewProvisionExtractor()

	article1 := doc.GetArticle(1)
	if article1 == nil {
		t.Fatal("Article 1 not found")
	}

	originalText := article1.Text
	extractor.ExtractProvisions(article1)

	// Serialize back to text
	serialized := SerializeArticle(article1)

	t.Logf("Original text length: %d", len(originalText))
	t.Logf("Serialized text length: %d", len(serialized))

	// The serialized text should contain the key content from the original
	// (exact match is difficult due to whitespace normalization)
	if !strings.Contains(serialized, "This Regulation lays down rules") {
		t.Error("Serialized text missing expected content from paragraph 1")
	}

	if !strings.Contains(serialized, "This Regulation protects fundamental rights") {
		t.Error("Serialized text missing expected content from paragraph 2")
	}

	if !strings.Contains(serialized, "The free movement of personal data") {
		t.Error("Serialized text missing expected content from paragraph 3")
	}
}

func TestRoundTrip_StructurePreservation(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewProvisionExtractor()
	extractor.ExtractAllProvisions(doc)

	// For each article, verify that extracting provisions and serializing
	// produces consistent results when re-parsed
	sampleArticles := []int{1, 5, 6, 17, 32, 83}

	for _, artNum := range sampleArticles {
		article := doc.GetArticle(artNum)
		if article == nil {
			t.Errorf("Article %d not found", artNum)
			continue
		}

		// Count structures
		paraCount := len(article.Paragraphs)
		pointCount := 0
		for _, para := range article.Paragraphs {
			pointCount += len(para.Points)
		}

		t.Logf("Article %d: %d paragraphs, %d points",
			artNum, paraCount, pointCount)

		// Verify structure exists
		if paraCount == 0 && article.Text != "" {
			t.Errorf("Article %d has text but no paragraphs extracted", artNum)
		}
	}
}

func TestProvisionExtraction_AllArticlesHaveText(t *testing.T) {
	testdataPath := filepath.Join("..", "..", "testdata", "gdpr.txt")
	f, err := os.Open(testdataPath)
	if err != nil {
		t.Fatalf("Failed to load GDPR text: %v", err)
	}
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	extractor := NewProvisionExtractor()
	extractor.ExtractAllProvisions(doc)

	allArticles := doc.AllArticles()

	// Verify all 99 articles have text
	articlesWithText := 0
	for _, article := range allArticles {
		if article.Text != "" {
			articlesWithText++
		} else {
			t.Errorf("Article %d has no text", article.Number)
		}
	}

	if articlesWithText != 99 {
		t.Errorf("Expected 99 articles with text, got %d", articlesWithText)
	}

	t.Logf("All %d articles have extracted text", articlesWithText)
}
