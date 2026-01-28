package extract

import (
	"os"
	"strings"
	"testing"
)

// Unit tests for findDefinitionArticles

func buildTestDocumentWithArticles(articles []*Article) *Document {
	chapter := &Chapter{
		Number: "1",
		Title:  "Test Chapter",
	}
	for _, article := range articles {
		chapter.Articles = append(chapter.Articles, article)
	}
	return &Document{
		Title:    "Test Document",
		Chapters: []*Chapter{chapter},
	}
}

func TestFindDefinitionArticles_ByTitle(t *testing.T) {
	doc := buildTestDocumentWithArticles([]*Article{
		{Number: 1, Title: "General Provisions", Text: "Some text here."},
		{Number: 2, Title: "Definitions", Text: "(1) 'term' means something."},
		{Number: 3, Title: "Scope", Text: "This applies to all."},
	})

	parser := NewParser()
	foundArticles := parser.findDefinitionArticles(doc, parser.definitionPattern)

	if len(foundArticles) != 1 {
		t.Fatalf("Expected 1 definition article, got %d", len(foundArticles))
	}
	if foundArticles[0].Number != 2 {
		t.Errorf("Expected article 2, got article %d", foundArticles[0].Number)
	}
}

func TestFindDefinitionArticles_ByTitleInterpretation(t *testing.T) {
	doc := buildTestDocumentWithArticles([]*Article{
		{Number: 1, Title: "Overview", Text: "Some overview."},
		{Number: 3, Title: "Interpretation", Text: `"controller" means a person who determines purposes.`},
	})

	parser := NewParser()
	foundArticles := parser.findDefinitionArticles(doc, parser.ukDefinitionPattern)

	if len(foundArticles) != 1 {
		t.Fatalf("Expected 1 definition article, got %d", len(foundArticles))
	}
	if foundArticles[0].Number != 3 {
		t.Errorf("Expected article 3, got article %d", foundArticles[0].Number)
	}
}

func TestFindDefinitionArticles_ByDensity(t *testing.T) {
	// Article with no "Definitions" title but high density of definition patterns
	definitionText := strings.Join([]string{
		"(1) 'personal data' means any information relating to a natural person.",
		"(2) 'processing' means any operation performed on personal data.",
		"(3) 'controller' means the entity that determines processing purposes.",
		"(4) 'processor' means the entity processing data on behalf of controller.",
	}, "\n")

	doc := buildTestDocumentWithArticles([]*Article{
		{Number: 1, Title: "General Provisions", Text: "Some text."},
		{Number: 4, Title: "Subject Matter and Objectives", Text: definitionText},
		{Number: 5, Title: "Scope", Text: "This applies to all."},
	})

	parser := NewParser()
	foundArticles := parser.findDefinitionArticles(doc, parser.definitionPattern)

	if len(foundArticles) != 1 {
		t.Fatalf("Expected 1 definition article found by density, got %d", len(foundArticles))
	}
	if foundArticles[0].Number != 4 {
		t.Errorf("Expected article 4, got article %d", foundArticles[0].Number)
	}
}

func TestFindDefinitionArticles_DensityBelowThreshold(t *testing.T) {
	// Only 2 definition patterns, below threshold of 3
	definitionText := strings.Join([]string{
		"(1) 'personal data' means any information relating to a natural person.",
		"(2) 'processing' means any operation performed on personal data.",
		"This section does not define further terms.",
	}, "\n")

	doc := buildTestDocumentWithArticles([]*Article{
		{Number: 1, Title: "Scope", Text: definitionText},
	})

	parser := NewParser()
	foundArticles := parser.findDefinitionArticles(doc, parser.definitionPattern)

	if len(foundArticles) != 0 {
		t.Errorf("Expected 0 articles (density below threshold), got %d", len(foundArticles))
	}
}

func TestFindDefinitionArticles_Multiple(t *testing.T) {
	// Two definition articles: one by title, one by density
	densityText := strings.Join([]string{
		"(1) 'term A' means definition A.",
		"(2) 'term B' means definition B.",
		"(3) 'term C' means definition C.",
	}, "\n")

	doc := buildTestDocumentWithArticles([]*Article{
		{Number: 1, Title: "General Provisions", Text: "Some text."},
		{Number: 2, Title: "Definitions", Text: "(1) 'base term' means something."},
		{Number: 5, Title: "Supplementary Provisions", Text: densityText},
	})

	parser := NewParser()
	foundArticles := parser.findDefinitionArticles(doc, parser.definitionPattern)

	if len(foundArticles) != 2 {
		t.Fatalf("Expected 2 definition articles, got %d", len(foundArticles))
	}

	foundNumbers := map[int]bool{}
	for _, article := range foundArticles {
		foundNumbers[article.Number] = true
	}
	if !foundNumbers[2] {
		t.Error("Expected article 2 (by title) to be found")
	}
	if !foundNumbers[5] {
		t.Error("Expected article 5 (by density) to be found")
	}
}

func TestFindDefinitionArticles_NoDuplicates(t *testing.T) {
	// Article matches both by title AND would match by density
	definitionText := strings.Join([]string{
		"(1) 'personal data' means any information relating to a natural person.",
		"(2) 'processing' means any operation performed on personal data.",
		"(3) 'controller' means the entity that determines processing purposes.",
	}, "\n")

	doc := buildTestDocumentWithArticles([]*Article{
		{Number: 4, Title: "Definitions", Text: definitionText},
	})

	parser := NewParser()
	foundArticles := parser.findDefinitionArticles(doc, parser.definitionPattern)

	if len(foundArticles) != 1 {
		t.Fatalf("Expected 1 article (no duplicates), got %d", len(foundArticles))
	}
}

func TestFindDefinitionArticles_EmptyDocument(t *testing.T) {
	doc := &Document{Title: "Empty", Chapters: []*Chapter{}}

	parser := NewParser()
	foundArticles := parser.findDefinitionArticles(doc, parser.definitionPattern)

	if len(foundArticles) != 0 {
		t.Errorf("Expected 0 articles from empty document, got %d", len(foundArticles))
	}
}

func TestFindDefinitionArticles_SkipsEmptyText(t *testing.T) {
	doc := buildTestDocumentWithArticles([]*Article{
		{Number: 2, Title: "Definitions", Text: ""},
	})

	parser := NewParser()
	foundArticles := parser.findDefinitionArticles(doc, parser.definitionPattern)

	if len(foundArticles) != 0 {
		t.Errorf("Expected 0 articles (empty text skipped), got %d", len(foundArticles))
	}
}

func TestFindDefinitionArticles_FallbackTitlePatterns(t *testing.T) {
	testCases := []struct {
		name  string
		title string
	}{
		{"Definitions_lowercase", "definitions"},
		{"Definition_singular", "Definition"},
		{"Interpretation", "Interpretation"},
		{"INTERPRETATION_upper", "INTERPRETATION"},
		{"Terms", "Terms"},
		{"TERMS_upper", "TERMS"},
		{"Definitions_mixed", "Article Definitions"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc := buildTestDocumentWithArticles([]*Article{
				{Number: 1, Title: tc.title, Text: "Some definition content."},
			})

			parser := NewParser()
			foundArticles := parser.findDefinitionArticles(doc, parser.definitionPattern)

			if len(foundArticles) != 1 {
				t.Errorf("Expected title %q to match, got %d articles", tc.title, len(foundArticles))
			}
		})
	}
}

func TestFindDefinitionArticles_WithBridge(t *testing.T) {
	registry := loadPatternRegistry(t)

	// Parse GDPR with registry (bridge should provide section_number: 4 and section_title)
	gdprFile := loadGDPRText(t)
	defer gdprFile.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(gdprFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// The bridge-aware findDefinitionArticles should find the definition article
	foundArticles := parser.findDefinitionArticles(doc, parser.definitionPattern)
	if len(foundArticles) < 1 {
		t.Error("Expected at least 1 definition article from GDPR with bridge")
	}

	// Verify article 4 (Definitions) is among them
	foundArticle4 := false
	for _, article := range foundArticles {
		if article.Number == 4 {
			foundArticle4 = true
			break
		}
	}
	if !foundArticle4 {
		t.Error("Expected Article 4 to be found via bridge")
	}
}

// Integration regression tests

func TestDefinitionAutoDetect_GDPR(t *testing.T) {
	expected := loadExpectedGDPR(t)

	gdprFile := loadGDPRText(t)
	defer gdprFile.Close()

	parser := NewParser()
	doc, err := parser.Parse(gdprFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	stats := doc.Statistics()
	if stats.Definitions != expected.Statistics.Definitions {
		t.Errorf("GDPR definitions: got %d, want %d", stats.Definitions, expected.Statistics.Definitions)
	}
}

func TestDefinitionAutoDetect_GDPRWithRegistry(t *testing.T) {
	expected := loadExpectedGDPR(t)
	registry := loadPatternRegistry(t)

	gdprFile := loadGDPRText(t)
	defer gdprFile.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(gdprFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	stats := doc.Statistics()
	if stats.Definitions != expected.Statistics.Definitions {
		t.Errorf("GDPR definitions (registry): got %d, want %d", stats.Definitions, expected.Statistics.Definitions)
	}
}

func TestDefinitionAutoDetect_CCPA(t *testing.T) {
	ccpaFile := loadCCPAText(t)
	defer ccpaFile.Close()

	parser := NewParser()
	doc, err := parser.Parse(ccpaFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	stats := doc.Statistics()
	if stats.Definitions < 5 {
		t.Errorf("CCPA definitions: got %d, want at least 5", stats.Definitions)
	}
	t.Logf("CCPA auto-detect: %d definitions found", stats.Definitions)
}

func TestDefinitionAutoDetect_CCPAWithRegistry(t *testing.T) {
	registry := loadPatternRegistry(t)

	ccpaFile := loadCCPAText(t)
	defer ccpaFile.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(ccpaFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	stats := doc.Statistics()
	if stats.Definitions < 5 {
		t.Errorf("CCPA definitions (registry): got %d, want at least 5", stats.Definitions)
	}
	t.Logf("CCPA auto-detect (registry): %d definitions found", stats.Definitions)
}

func TestDefinitionAutoDetect_AllUSStatesWithRegistry(t *testing.T) {
	// Verify definition extraction across all US state privacy laws with registry
	registry := loadPatternRegistry(t)

	testCases := []struct {
		name     string
		loadFunc func(t *testing.T) *os.File
		minDefs  int
	}{
		{"CCPA", loadCCPAText, 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := tc.loadFunc(t)
			defer f.Close()

			parser := NewParserWithRegistry(registry)
			doc, err := parser.Parse(f)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			stats := doc.Statistics()
			t.Logf("%s (registry): format=%s, definitions=%d", tc.name, parser.format, stats.Definitions)

			if stats.Definitions < tc.minDefs {
				t.Errorf("Expected at least %d definitions, got %d", tc.minDefs, stats.Definitions)
			}
		})
	}
}

// DefinitionExtractor tests

func TestDefinitionExtractor_DynamicScope(t *testing.T) {
	// Build a document where definitions are in Article 7, not Article 4
	doc := buildTestDocumentWithArticles([]*Article{
		{Number: 1, Title: "General", Text: "General provisions."},
		{Number: 7, Title: "Definitions", Text: strings.Join([]string{
			"(1) 'personal data' means any information relating to a natural person.",
			"(2) 'processing' means any operation performed on personal data.",
			"(3) 'controller' means the entity that determines processing purposes.",
		}, "\n")},
	})

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)

	if len(definitions) < 3 {
		t.Fatalf("Expected at least 3 definitions, got %d", len(definitions))
	}

	for _, def := range definitions {
		expectedScope := "Article 7"
		if def.Scope != expectedScope {
			t.Errorf("Definition %q: got scope %q, want %q", def.Term, def.Scope, expectedScope)
		}
		if def.ArticleRef != 7 {
			t.Errorf("Definition %q: got ArticleRef %d, want 7", def.Term, def.ArticleRef)
		}
	}
}

func TestDefinitionExtractor_MultipleSections(t *testing.T) {
	// Build a document with definitions spread across two articles
	chapter := &Chapter{
		Number: "1",
		Title:  "Main",
		Articles: []*Article{
			{Number: 4, Title: "Definitions", Text: strings.Join([]string{
				"(1) 'personal data' means any information relating to a natural person.",
				"(2) 'processing' means any operation performed on personal data.",
				"(3) 'controller' means the entity that determines processing purposes.",
			}, "\n")},
			{Number: 12, Title: "Additional Definitions", Text: strings.Join([]string{
				"(1) 'profiling' means automated processing of personal data.",
				"(2) 'consent' means freely given, specific, informed agreement.",
				"(3) 'recipient' means an entity to which data is disclosed.",
			}, "\n")},
		},
	}

	doc := &Document{
		Title:    "Test Regulation",
		Chapters: []*Chapter{chapter},
	}

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)

	if len(definitions) < 6 {
		t.Errorf("Expected at least 6 definitions from two articles, got %d", len(definitions))
	}

	// Check that both scopes are present
	scopeArticle4 := false
	scopeArticle12 := false
	for _, def := range definitions {
		if def.Scope == "Article 4" {
			scopeArticle4 = true
		}
		if def.Scope == "Article 12" {
			scopeArticle12 = true
		}
	}
	if !scopeArticle4 {
		t.Error("Expected definitions with scope 'Article 4'")
	}
	if !scopeArticle12 {
		t.Error("Expected definitions with scope 'Article 12'")
	}

	t.Logf("Multi-section extraction: %d definitions found", len(definitions))
	for _, def := range definitions {
		t.Logf("  %d. %q (scope: %s)", def.Number, def.Term, def.Scope)
	}
}

func TestDefinitionExtractor_DensityFallback(t *testing.T) {
	// Article with no "Definitions" title but high density of definition patterns
	doc := buildTestDocumentWithArticles([]*Article{
		{Number: 99, Title: "Preliminary Matters", Text: strings.Join([]string{
			"(1) 'personal data' means any information relating to a natural person.",
			"(2) 'processing' means any operation performed on personal data.",
			"(3) 'controller' means the entity that determines processing purposes.",
			"(4) 'processor' means the entity processing data on behalf.",
		}, "\n")},
	})

	extractor := NewDefinitionExtractor()
	definitions := extractor.ExtractDefinitions(doc)

	if len(definitions) < 3 {
		t.Errorf("Expected at least 3 definitions via density detection, got %d", len(definitions))
	}

	for _, def := range definitions {
		if def.Scope != "Article 99" {
			t.Errorf("Expected scope 'Article 99', got %q", def.Scope)
		}
	}
}
