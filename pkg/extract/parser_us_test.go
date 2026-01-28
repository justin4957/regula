package extract

import (
	"os"
	"path/filepath"
	"testing"
)

func loadCCPAText(t *testing.T) *os.File {
	t.Helper()

	testdataPath := filepath.Join("..", "..", "testdata", "ccpa.txt")

	f, err := os.Open(testdataPath)
	if err != nil {
		t.Fatalf("Failed to load CCPA text: %v", err)
	}

	return f
}

func TestParseCCPA_FormatDetection(t *testing.T) {
	f := loadCCPAText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check format was detected as US
	if parser.format != FormatUS {
		t.Errorf("Expected FormatUS, got %v", parser.format)
	}

	// Check document type
	if doc.Type != DocumentTypeAct {
		t.Logf("Document type: %v (expected Act)", doc.Type)
	}

	// Check identifier
	t.Logf("Document identifier: %s", doc.Identifier)
	t.Logf("Document title: %s", doc.Title)
}

func TestParseCCPA_Structure(t *testing.T) {
	f := loadCCPAText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	stats := doc.Statistics()

	t.Logf("CCPA Statistics:")
	t.Logf("  Chapters: %d", stats.Chapters)
	t.Logf("  Sections: %d", stats.Sections)
	t.Logf("  Articles: %d", stats.Articles)
	t.Logf("  Definitions: %d", stats.Definitions)
	t.Logf("  Recitals: %d", stats.Recitals)

	// CCPA should have at least 6 chapters
	if stats.Chapters < 6 {
		t.Errorf("Expected at least 6 chapters, got %d", stats.Chapters)
	}

	// Should have articles (sections in CCPA are mapped to articles)
	if stats.Articles < 10 {
		t.Errorf("Expected at least 10 articles (sections), got %d", stats.Articles)
	}

	// List chapters
	t.Logf("\nChapters:")
	for _, ch := range doc.Chapters {
		t.Logf("  Chapter %s: %s (%d articles)", ch.Number, ch.Title, len(ch.Articles))
	}
}

func TestParseCCPA_Articles(t *testing.T) {
	f := loadCCPAText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	articles := doc.AllArticles()
	t.Logf("Found %d articles (sections)", len(articles))

	// Show first few articles
	for i, art := range articles {
		if i >= 10 {
			break
		}
		textPreview := art.Text
		if len(textPreview) > 100 {
			textPreview = textPreview[:100] + "..."
		}
		t.Logf("  Section 1798.%d: %s", art.Number, art.Title)
		t.Logf("    Text: %s", textPreview)
	}
}

func TestParseCCPA_Definitions(t *testing.T) {
	f := loadCCPAText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	t.Logf("Found %d definitions", len(doc.Definitions))

	for _, def := range doc.Definitions {
		t.Logf("  %d: %s", def.Number, def.Term)
	}

	// CCPA Section 1798.110 defines many terms
	if len(doc.Definitions) < 5 {
		t.Errorf("Expected at least 5 definitions, got %d", len(doc.Definitions))
	}
}

func TestParseCCPA_DefinitionsContent(t *testing.T) {
	f := loadCCPAText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Find the definitions article (Section 1798.110 -> Article 110)
	defArticle := doc.GetArticle(110)
	if defArticle == nil {
		t.Log("Article 110 (Definitions) not found")

		// List what articles we do have
		for _, art := range doc.AllArticles() {
			t.Logf("  Found Article %d: %s", art.Number, art.Title)
		}
		return
	}

	t.Logf("Article 110 title: %s", defArticle.Title)
	t.Logf("Article 110 text length: %d chars", len(defArticle.Text))

	// Show first part of text
	preview := defArticle.Text
	if len(preview) > 500 {
		preview = preview[:500] + "..."
	}
	t.Logf("Article 110 text preview:\n%s", preview)
}
