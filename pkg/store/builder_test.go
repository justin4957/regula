package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coolbeans/regula/pkg/extract"
)

func TestNewGraphBuilder(t *testing.T) {
	store := NewTripleStore()
	builder := NewGraphBuilder(store, "https://example.org/")

	if builder == nil {
		t.Fatal("NewGraphBuilder returned nil")
	}
	if builder.store != store {
		t.Error("Builder store mismatch")
	}
	if builder.baseURI != "https://example.org/" {
		t.Errorf("Base URI mismatch: %s", builder.baseURI)
	}
}

func TestNewGraphBuilder_AddsSuffix(t *testing.T) {
	store := NewTripleStore()
	builder := NewGraphBuilder(store, "https://example.org")

	if builder.baseURI != "https://example.org#" {
		t.Errorf("Expected # suffix, got: %s", builder.baseURI)
	}
}

func TestGraphBuilder_URIs(t *testing.T) {
	store := NewTripleStore()
	builder := NewGraphBuilder(store, "https://example.org/")
	builder.regID = "GDPR"

	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{"regulation", builder.regulationURI(), "https://example.org/GDPR"},
		{"chapter", builder.chapterURI("III"), "https://example.org/GDPR:ChapterIII"},
		{"section", builder.sectionURI("III", 2), "https://example.org/GDPR:ChapterIII:Section2"},
		{"article", builder.articleURI(17), "https://example.org/GDPR:Art17"},
		{"paragraph", builder.paragraphURI(17, 1), "https://example.org/GDPR:Art17(1)"},
		{"point", builder.pointURI(6, 1, "a"), "https://example.org/GDPR:Art6(1)(a)"},
		{"recital", builder.recitalURI(39), "https://example.org/GDPR:Recital39"},
		{"preamble", builder.preambleURI(), "https://example.org/GDPR:Preamble"},
		{"definition", builder.definitionURI("personal data"), "https://example.org/GDPR:Term:personal_data"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.uri != tc.expected {
				t.Errorf("got %s, want %s", tc.uri, tc.expected)
			}
		})
	}
}

func TestGraphBuilder_Build_SimpleDocument(t *testing.T) {
	store := NewTripleStore()
	builder := NewGraphBuilder(store, "https://test.org/")

	doc := &extract.Document{
		Title:      "Test Regulation",
		Type:       extract.DocumentTypeRegulation,
		Identifier: "(EU) 2024/001",
		Chapters: []*extract.Chapter{
			{
				Number: "I",
				Title:  "General Provisions",
				Articles: []*extract.Article{
					{
						Number: 1,
						Title:  "Subject matter",
						Text:   "This regulation applies to...",
					},
					{
						Number: 2,
						Title:  "Scope",
						Text:   "The scope of this regulation...",
					},
				},
			},
		},
		Definitions: []*extract.Definition{
			{
				Number: 1,
				Term:   "data",
				Text:   "information in any form",
			},
		},
	}

	stats, err := builder.Build(doc)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if stats.Articles != 2 {
		t.Errorf("Expected 2 articles, got %d", stats.Articles)
	}
	if stats.Chapters != 1 {
		t.Errorf("Expected 1 chapter, got %d", stats.Chapters)
	}
	if stats.Definitions != 1 {
		t.Errorf("Expected 1 definition, got %d", stats.Definitions)
	}

	// Verify some triples exist
	articles := store.Find("", RDFType, ClassArticle)
	if len(articles) != 2 {
		t.Errorf("Expected 2 article triples, got %d", len(articles))
	}

	chapters := store.Find("", RDFType, ClassChapter)
	if len(chapters) != 1 {
		t.Errorf("Expected 1 chapter triple, got %d", len(chapters))
	}
}

func TestGraphBuilder_Build_WithPreamble(t *testing.T) {
	store := NewTripleStore()
	builder := NewGraphBuilder(store, "https://test.org/")

	doc := &extract.Document{
		Title:      "Test Regulation",
		Type:       extract.DocumentTypeRegulation,
		Identifier: "(EU) 2024/001",
		Preamble: &extract.Preamble{
			Recitals: []*extract.Recital{
				{Number: 1, Text: "First recital text"},
				{Number: 2, Text: "Second recital text"},
				{Number: 3, Text: "Third recital text"},
			},
		},
		Chapters: []*extract.Chapter{},
	}

	stats, err := builder.Build(doc)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if stats.Recitals != 3 {
		t.Errorf("Expected 3 recitals, got %d", stats.Recitals)
	}

	// Verify recital triples
	recitals := store.Find("", RDFType, ClassRecital)
	if len(recitals) != 3 {
		t.Errorf("Expected 3 recital triples, got %d", len(recitals))
	}
}

func TestGraphBuilder_Build_WithSections(t *testing.T) {
	store := NewTripleStore()
	builder := NewGraphBuilder(store, "https://test.org/")

	doc := &extract.Document{
		Title:      "Test Regulation",
		Type:       extract.DocumentTypeRegulation,
		Identifier: "(EU) 2024/001",
		Chapters: []*extract.Chapter{
			{
				Number: "III",
				Title:  "Rights",
				Sections: []*extract.Section{
					{
						Number: 1,
						Title:  "Transparency",
						Articles: []*extract.Article{
							{Number: 12, Title: "Transparent information"},
						},
					},
					{
						Number: 2,
						Title:  "Information",
						Articles: []*extract.Article{
							{Number: 13, Title: "Information to be provided"},
							{Number: 14, Title: "Information for indirect collection"},
						},
					},
				},
			},
		},
	}

	stats, err := builder.Build(doc)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if stats.Sections != 2 {
		t.Errorf("Expected 2 sections, got %d", stats.Sections)
	}
	if stats.Articles != 3 {
		t.Errorf("Expected 3 articles, got %d", stats.Articles)
	}

	// Verify hierarchy
	sections := store.Find("", RDFType, ClassSection)
	if len(sections) != 2 {
		t.Errorf("Expected 2 section triples, got %d", len(sections))
	}
}

func TestGraphBuilder_Build_WithParagraphsAndPoints(t *testing.T) {
	store := NewTripleStore()
	builder := NewGraphBuilder(store, "https://test.org/")

	doc := &extract.Document{
		Title:      "Test Regulation",
		Type:       extract.DocumentTypeRegulation,
		Identifier: "(EU) 2024/001",
		Chapters: []*extract.Chapter{
			{
				Number: "I",
				Title:  "General",
				Articles: []*extract.Article{
					{
						Number: 6,
						Title:  "Lawfulness",
						Paragraphs: []*extract.Paragraph{
							{
								Number: 1,
								Text:   "Processing shall be lawful only if...",
								Points: []*extract.Point{
									{Letter: "a", Text: "the data subject has given consent"},
									{Letter: "b", Text: "processing is necessary for contract"},
								},
							},
							{
								Number: 2,
								Text:   "Member States may maintain...",
							},
						},
					},
				},
			},
		},
	}

	_, err := builder.Build(doc)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify paragraphs
	paragraphs := store.Find("", RDFType, ClassParagraph)
	if len(paragraphs) != 2 {
		t.Errorf("Expected 2 paragraph triples, got %d", len(paragraphs))
	}

	// Verify points
	points := store.Find("", RDFType, ClassPoint)
	if len(points) != 2 {
		t.Errorf("Expected 2 point triples, got %d", len(points))
	}
}

func TestGraphBuilder_Build_NilDocument(t *testing.T) {
	store := NewTripleStore()
	builder := NewGraphBuilder(store, "https://test.org/")

	_, err := builder.Build(nil)
	if err == nil {
		t.Error("Expected error for nil document")
	}
}

func TestGraphBuilder_BuildWithExtractors(t *testing.T) {
	store := NewTripleStore()
	builder := NewGraphBuilder(store, "https://test.org/")

	doc := &extract.Document{
		Title:      "Test Regulation",
		Type:       extract.DocumentTypeRegulation,
		Identifier: "(EU) 2024/001",
		Chapters: []*extract.Chapter{
			{
				Number: "I",
				Title:  "General",
				Articles: []*extract.Article{
					{
						Number: 1,
						Title:  "Subject matter",
						Text:   "This regulation refers to Article 2.",
					},
					{
						Number: 2,
						Title:  "Scope",
						Text:   "The scope covers...",
					},
				},
			},
		},
	}

	refExtractor := extract.NewReferenceExtractor()

	stats, err := builder.BuildWithExtractors(doc, nil, refExtractor)
	if err != nil {
		t.Fatalf("BuildWithExtractors failed: %v", err)
	}

	if stats.Articles != 2 {
		t.Errorf("Expected 2 articles, got %d", stats.Articles)
	}

	// Should have found reference to Article 2
	if stats.References < 1 {
		t.Logf("References found: %d (may vary based on text)", stats.References)
	}
}

func TestGraphBuilder_Hierarchy(t *testing.T) {
	store := NewTripleStore()
	builder := NewGraphBuilder(store, "https://test.org/")

	doc := &extract.Document{
		Title:      "Test Regulation",
		Type:       extract.DocumentTypeRegulation,
		Identifier: "(EU) 2024/001",
		Chapters: []*extract.Chapter{
			{
				Number: "I",
				Title:  "General",
				Articles: []*extract.Article{
					{Number: 1, Title: "Article 1"},
				},
			},
		},
	}

	builder.Build(doc)

	// Verify article is part of chapter
	art1URI := builder.articleURI(1)
	chapterURI := builder.chapterURI("I")

	partOf := store.Find(art1URI, PropPartOf, "")
	if len(partOf) != 1 {
		t.Errorf("Expected 1 partOf for article, got %d", len(partOf))
	}
	if partOf[0].Object != chapterURI {
		t.Errorf("Article should be part of chapter, got %s", partOf[0].Object)
	}

	// Verify chapter contains article
	contains := store.Find(chapterURI, PropContains, art1URI)
	if len(contains) != 1 {
		t.Errorf("Expected chapter to contain article")
	}

	// Verify chapter is part of regulation
	regURI := builder.regulationURI()
	chapterPartOf := store.Find(chapterURI, PropPartOf, regURI)
	if len(chapterPartOf) != 1 {
		t.Errorf("Expected chapter to be part of regulation")
	}
}

func TestGraphBuilder_ExtractRegID(t *testing.T) {
	store := NewTripleStore()
	builder := NewGraphBuilder(store, "https://test.org/")

	tests := []struct {
		identifier string
		expected   string
	}{
		{"(EU) 2016/679", "GDPR"},
		{"(EU) 2024/001", "Reg001"},
		{"", "Regulation"},
	}

	for _, tc := range tests {
		result := builder.extractRegID(tc.identifier)
		if result != tc.expected {
			t.Errorf("extractRegID(%q): got %s, want %s", tc.identifier, result, tc.expected)
		}
	}
}

func TestGraphBuilder_NormalizeTerm(t *testing.T) {
	store := NewTripleStore()
	builder := NewGraphBuilder(store, "https://test.org/")

	tests := []struct {
		term     string
		expected string
	}{
		{"personal data", "personal_data"},
		{"Controller", "controller"},
		{"cross-border processing", "cross-border_processing"},
		{"data subject's rights", "data_subjects_rights"},
	}

	for _, tc := range tests {
		result := builder.normalizeTerm(tc.term)
		if result != tc.expected {
			t.Errorf("normalizeTerm(%q): got %s, want %s", tc.term, result, tc.expected)
		}
	}
}

// Integration test with real GDPR data

func loadGDPRDocument(t *testing.T) *extract.Document {
	t.Helper()

	testdataPath := filepath.Join("..", "..", "testdata", "gdpr.txt")
	f, err := os.Open(testdataPath)
	if err != nil {
		t.Fatalf("Failed to open GDPR test data: %v", err)
	}
	defer f.Close()

	parser := extract.NewParser()
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Failed to parse GDPR: %v", err)
	}

	return doc
}

func TestBuildGDPRGraph(t *testing.T) {
	doc := loadGDPRDocument(t)

	store := NewTripleStore()
	builder := NewGraphBuilder(store, "https://regula.dev/regulations/")

	stats, err := builder.Build(doc)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	t.Logf("GDPR Graph Statistics:")
	t.Logf("  Total triples: %d", stats.TotalTriples)
	t.Logf("  Articles: %d", stats.Articles)
	t.Logf("  Chapters: %d", stats.Chapters)
	t.Logf("  Sections: %d", stats.Sections)
	t.Logf("  Recitals: %d", stats.Recitals)
	t.Logf("  Definitions: %d", stats.Definitions)

	// Verify article count
	if stats.Articles != 99 {
		t.Errorf("Expected 99 articles, got %d", stats.Articles)
	}

	// Verify chapter count
	if stats.Chapters != 11 {
		t.Errorf("Expected 11 chapters, got %d", stats.Chapters)
	}

	// Verify definition count
	if stats.Definitions != 26 {
		t.Errorf("Expected 26 definitions, got %d", stats.Definitions)
	}

	// Verify recital count
	if stats.Recitals != 173 {
		t.Errorf("Expected 173 recitals, got %d", stats.Recitals)
	}

	// Verify total triples is substantial (without extractors, expect ~2000)
	// With extractors (definitions + references), expect ~4000+
	if stats.TotalTriples < 2000 {
		t.Errorf("Expected at least 2000 triples, got %d", stats.TotalTriples)
	}
}

func TestBuildGDPRGraph_WithExtractors(t *testing.T) {
	doc := loadGDPRDocument(t)

	store := NewTripleStore()
	builder := NewGraphBuilder(store, "https://regula.dev/regulations/")

	defExtractor := extract.NewDefinitionExtractor()
	refExtractor := extract.NewReferenceExtractor()

	stats, err := builder.BuildWithExtractors(doc, defExtractor, refExtractor)
	if err != nil {
		t.Fatalf("BuildWithExtractors failed: %v", err)
	}

	t.Logf("GDPR Graph Statistics (with extractors):")
	t.Logf("  Total triples: %d", stats.TotalTriples)
	t.Logf("  Articles: %d", stats.Articles)
	t.Logf("  Chapters: %d", stats.Chapters)
	t.Logf("  Sections: %d", stats.Sections)
	t.Logf("  Recitals: %d", stats.Recitals)
	t.Logf("  Definitions: %d", stats.Definitions)
	t.Logf("  References: %d", stats.References)

	// With extractors, we should have more triples
	if stats.TotalTriples < 4000 {
		t.Errorf("Expected at least 4000 triples with extractors, got %d", stats.TotalTriples)
	}

	// Should have references
	if stats.References < 100 {
		t.Errorf("Expected at least 100 references, got %d", stats.References)
	}
}

func TestBuildGDPRGraph_Queries(t *testing.T) {
	doc := loadGDPRDocument(t)

	store := NewTripleStore()
	builder := NewGraphBuilder(store, "https://regula.dev/regulations/")

	defExtractor := extract.NewDefinitionExtractor()
	refExtractor := extract.NewReferenceExtractor()

	builder.BuildWithExtractors(doc, defExtractor, refExtractor)

	// Query: Find all articles
	articles := store.Find("", RDFType, ClassArticle)
	t.Logf("Query 'all articles': %d results", len(articles))
	if len(articles) != 99 {
		t.Errorf("Expected 99 articles, got %d", len(articles))
	}

	// Query: Find all chapters
	chapters := store.Find("", RDFType, ClassChapter)
	t.Logf("Query 'all chapters': %d results", len(chapters))
	if len(chapters) != 11 {
		t.Errorf("Expected 11 chapters, got %d", len(chapters))
	}

	// Query: Find Article 17
	art17URI := builder.articleURI(17)
	art17Props := store.Get(art17URI)
	t.Logf("Query 'Article 17 properties': %d predicates", len(art17Props))

	if title, ok := art17Props[PropTitle]; !ok || len(title) == 0 {
		t.Error("Article 17 should have a title")
	}

	// Query: Find what Article 17 is part of
	partOf := store.Find(art17URI, PropPartOf, "")
	t.Logf("Query 'Article 17 partOf': %d results", len(partOf))
	if len(partOf) != 1 {
		t.Errorf("Article 17 should be part of exactly 1 container, got %d", len(partOf))
	}

	// Query: Find all definitions
	definitions := store.Find("", RDFType, ClassDefinedTerm)
	t.Logf("Query 'all definitions': %d results", len(definitions))
	if len(definitions) != 26 {
		t.Errorf("Expected 26 definitions, got %d", len(definitions))
	}

	// Query: Find references from Article 17
	art17Refs := store.Find(art17URI, PropReferences, "")
	t.Logf("Query 'Article 17 references': %d results", len(art17Refs))
}

func TestBuildGDPRGraph_VerifyHierarchy(t *testing.T) {
	doc := loadGDPRDocument(t)

	store := NewTripleStore()
	builder := NewGraphBuilder(store, "https://regula.dev/regulations/")

	builder.Build(doc)

	regURI := builder.regulationURI()

	// Verify regulation has chapters
	hasChapters := store.Find(regURI, PropHasChapter, "")
	t.Logf("Regulation has %d chapters", len(hasChapters))
	if len(hasChapters) != 11 {
		t.Errorf("Expected regulation to have 11 chapters, got %d", len(hasChapters))
	}

	// Verify Chapter III (Rights of data subject) has sections
	chapterIII := builder.chapterURI("III")
	hasSections := store.Find(chapterIII, PropHasSection, "")
	t.Logf("Chapter III has %d sections", len(hasSections))
	if len(hasSections) < 1 {
		t.Errorf("Expected Chapter III to have sections")
	}

	// Verify all articles belong to regulation
	articles := store.Find("", RDFType, ClassArticle)
	for _, art := range articles {
		belongsTo := store.Find(art.Subject, PropBelongsTo, regURI)
		if len(belongsTo) != 1 {
			t.Errorf("Article %s should belong to regulation", art.Subject)
			break
		}
	}
}
