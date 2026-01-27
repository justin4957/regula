package extract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ExpectedGDPR represents the expected output format from gdpr-expected.json
type ExpectedGDPR struct {
	Regulation string `json:"regulation"`
	FullName   string `json:"full_name"`
	Statistics struct {
		Chapters    int `json:"chapters"`
		Articles    int `json:"articles"`
		Definitions int `json:"definitions"`
	} `json:"statistics"`
	Chapters []struct {
		Number string `json:"number"`
		Title  string `json:"title"`
	} `json:"chapters"`
	Articles []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	} `json:"articles"`
	Definitions []struct {
		Number int    `json:"number"`
		Term   string `json:"term"`
	} `json:"definitions"`
}

func loadExpectedGDPR(t *testing.T) *ExpectedGDPR {
	t.Helper()

	// Find testdata relative to this test file
	testdataPath := filepath.Join("..", "..", "testdata", "gdpr-expected.json")

	data, err := os.ReadFile(testdataPath)
	if err != nil {
		t.Fatalf("Failed to load expected GDPR data: %v", err)
	}

	var expected ExpectedGDPR
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatalf("Failed to parse expected GDPR data: %v", err)
	}

	return &expected
}

func loadGDPRText(t *testing.T) *os.File {
	t.Helper()

	testdataPath := filepath.Join("..", "..", "testdata", "gdpr.txt")

	f, err := os.Open(testdataPath)
	if err != nil {
		t.Fatalf("Failed to load GDPR text: %v", err)
	}

	return f
}

func TestParseGDPR(t *testing.T) {
	expected := loadExpectedGDPR(t)
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	stats := doc.Statistics()

	// Log parsing results
	t.Logf("Parsed %d articles", stats.Articles)
	t.Logf("Parsed %d chapters", stats.Chapters)
	t.Logf("Parsed %d sections", stats.Sections)
	t.Logf("Parsed %d definitions", stats.Definitions)
	t.Logf("Parsed %d recitals", stats.Recitals)

	// Verify article count
	if stats.Articles != expected.Statistics.Articles {
		t.Errorf("Article count mismatch: got %d, want %d", stats.Articles, expected.Statistics.Articles)
	}

	// Verify chapter count
	if stats.Chapters != expected.Statistics.Chapters {
		t.Errorf("Chapter count mismatch: got %d, want %d", stats.Chapters, expected.Statistics.Chapters)
	}

	// Verify definition count
	if stats.Definitions != expected.Statistics.Definitions {
		t.Errorf("Definition count mismatch: got %d, want %d", stats.Definitions, expected.Statistics.Definitions)
	}
}

func TestParseGDPR_ChapterTitles(t *testing.T) {
	expected := loadExpectedGDPR(t)
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(doc.Chapters) != len(expected.Chapters) {
		t.Fatalf("Chapter count mismatch: got %d, want %d", len(doc.Chapters), len(expected.Chapters))
	}

	for i, expChapter := range expected.Chapters {
		gotChapter := doc.Chapters[i]

		if gotChapter.Number != expChapter.Number {
			t.Errorf("Chapter %d number mismatch: got %q, want %q", i+1, gotChapter.Number, expChapter.Number)
		}

		if gotChapter.Title != expChapter.Title {
			t.Errorf("Chapter %s title mismatch: got %q, want %q", expChapter.Number, gotChapter.Title, expChapter.Title)
		}
	}
}

func TestParseGDPR_ArticleTitles(t *testing.T) {
	expected := loadExpectedGDPR(t)
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	allArticles := doc.AllArticles()

	// Build a map for easier lookup
	articleMap := make(map[int]*Article)
	for _, art := range allArticles {
		articleMap[art.Number] = art
	}

	for _, expArticle := range expected.Articles {
		gotArticle, exists := articleMap[expArticle.Number]
		if !exists {
			t.Errorf("Article %d not found", expArticle.Number)
			continue
		}

		if gotArticle.Title != expArticle.Title {
			t.Errorf("Article %d title mismatch: got %q, want %q", expArticle.Number, gotArticle.Title, expArticle.Title)
		}
	}
}

func TestParseGDPR_Definitions(t *testing.T) {
	expected := loadExpectedGDPR(t)
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(doc.Definitions) != len(expected.Definitions) {
		t.Logf("Got definitions:")
		for _, def := range doc.Definitions {
			t.Logf("  %d: %s", def.Number, def.Term)
		}
		t.Fatalf("Definition count mismatch: got %d, want %d", len(doc.Definitions), len(expected.Definitions))
	}

	for i, expDef := range expected.Definitions {
		if i >= len(doc.Definitions) {
			break
		}
		gotDef := doc.Definitions[i]

		if gotDef.Number != expDef.Number {
			t.Errorf("Definition %d number mismatch: got %d, want %d", i+1, gotDef.Number, expDef.Number)
		}

		if gotDef.Term != expDef.Term {
			t.Errorf("Definition %d term mismatch: got %q, want %q", expDef.Number, gotDef.Term, expDef.Term)
		}
	}
}

func TestParseGDPR_NestedStructure(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Chapter III should have sections (Rights of the data subject)
	chapter3 := doc.GetChapter("III")
	if chapter3 == nil {
		t.Fatal("Chapter III not found")
	}

	if len(chapter3.Sections) == 0 {
		t.Error("Chapter III should have sections")
	}

	t.Logf("Chapter III has %d sections", len(chapter3.Sections))
	for _, section := range chapter3.Sections {
		t.Logf("  Section %d: %s (%d articles)", section.Number, section.Title, len(section.Articles))
	}

	// Verify that articles in sections are correctly assigned
	totalArticlesInSections := 0
	for _, section := range chapter3.Sections {
		totalArticlesInSections += len(section.Articles)
	}

	if totalArticlesInSections == 0 {
		t.Error("No articles found in Chapter III sections")
	}
}

func TestParseGDPR_ArticleContent(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Article 1 should have content about "subject-matter and objectives"
	article1 := doc.GetArticle(1)
	if article1 == nil {
		t.Fatal("Article 1 not found")
	}

	if article1.Title != "Subject-matter and objectives" {
		t.Errorf("Article 1 title mismatch: got %q", article1.Title)
	}

	if article1.Text == "" {
		t.Error("Article 1 should have text content")
	}

	// Should contain numbered paragraphs
	if len(article1.Text) < 100 {
		t.Errorf("Article 1 text seems too short: %d chars", len(article1.Text))
	}

	t.Logf("Article 1 text length: %d chars", len(article1.Text))
}

func TestParseGDPR_Recitals(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if doc.Preamble == nil {
		t.Fatal("Preamble not found")
	}

	// GDPR has 173 recitals
	expectedRecitals := 173
	if len(doc.Preamble.Recitals) != expectedRecitals {
		t.Errorf("Recital count mismatch: got %d, want %d", len(doc.Preamble.Recitals), expectedRecitals)
	}

	// Check first recital
	if len(doc.Preamble.Recitals) > 0 {
		first := doc.Preamble.Recitals[0]
		if first.Number != 1 {
			t.Errorf("First recital number mismatch: got %d, want 1", first.Number)
		}
		if first.Text == "" {
			t.Error("First recital should have text")
		}
		t.Logf("Recital 1 starts with: %.100s...", first.Text)
	}

	// Check last recital
	if len(doc.Preamble.Recitals) > 0 {
		last := doc.Preamble.Recitals[len(doc.Preamble.Recitals)-1]
		if last.Number != expectedRecitals {
			t.Errorf("Last recital number mismatch: got %d, want %d", last.Number, expectedRecitals)
		}
	}
}

func TestParseGDPR_DocumentMetadata(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check document type
	if doc.Type != DocumentTypeRegulation {
		t.Errorf("Document type mismatch: got %q, want %q", doc.Type, DocumentTypeRegulation)
	}

	// Check identifier
	if doc.Identifier == "" {
		t.Error("Document identifier should not be empty")
	}
	t.Logf("Document identifier: %s", doc.Identifier)

	// Check title
	if doc.Title == "" {
		t.Error("Document title should not be empty")
	}
	t.Logf("Document title: %s", doc.Title)
}

func TestParser_GetArticle(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Test various article retrievals
	testCases := []struct {
		number int
		title  string
	}{
		{1, "Subject-matter and objectives"},
		{4, "Definitions"},
		{17, "Right to erasure (\u2018right to be forgotten\u2019)"},
		{99, "Entry into force and application"},
	}

	for _, tc := range testCases {
		article := doc.GetArticle(tc.number)
		if article == nil {
			t.Errorf("Article %d not found", tc.number)
			continue
		}
		if article.Title != tc.title {
			t.Errorf("Article %d title mismatch: got %q, want %q", tc.number, article.Title, tc.title)
		}
	}

	// Test non-existent article
	if doc.GetArticle(100) != nil {
		t.Error("Article 100 should not exist")
	}
}

func TestParser_GetChapter(t *testing.T) {
	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Test chapter retrieval
	testCases := []struct {
		number string
		title  string
	}{
		{"I", "General provisions"},
		{"III", "Rights of the data subject"},
		{"XI", "Final provisions"},
	}

	for _, tc := range testCases {
		chapter := doc.GetChapter(tc.number)
		if chapter == nil {
			t.Errorf("Chapter %s not found", tc.number)
			continue
		}
		if chapter.Title != tc.title {
			t.Errorf("Chapter %s title mismatch: got %q, want %q", tc.number, chapter.Title, tc.title)
		}
	}

	// Test non-existent chapter
	if doc.GetChapter("XII") != nil {
		t.Error("Chapter XII should not exist")
	}
}
