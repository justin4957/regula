package extract

import (
	"os"
	"testing"
)

func TestTermUsageExtractor(t *testing.T) {
	// Create some sample definitions
	definitions := []*DefinedTerm{
		{
			Number:         1,
			Term:           "personal data",
			NormalizedTerm: "personal data",
			ArticleRef:     4,
		},
		{
			Number:         2,
			Term:           "controller",
			NormalizedTerm: "controller",
			ArticleRef:     4,
		},
		{
			Number:         3,
			Term:           "processor",
			NormalizedTerm: "processor",
			ArticleRef:     4,
		},
		{
			Number:         7,
			Term:           "data subject",
			NormalizedTerm: "data subject",
			ArticleRef:     4,
		},
	}

	extractor := NewTermUsageExtractor(definitions)

	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "Single term usage",
			text:     "The controller shall process personal data in accordance with this Regulation.",
			expected: []string{"controller", "personal data"},
		},
		{
			name:     "Plural form",
			text:     "Data subjects have the right to access their personal data held by controllers.",
			expected: []string{"data subject", "personal data", "controller"},
		},
		{
			name:     "Case insensitive",
			text:     "A CONTROLLER must ensure the security of Personal Data.",
			expected: []string{"controller", "personal data"},
		},
		{
			name:     "No matches",
			text:     "This article describes general provisions for the regulation.",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usages := extractor.findTermsInText(tt.text, 17, 1, "")

			found := make(map[string]bool)
			for _, u := range usages {
				found[u.NormalizedTerm] = true
			}

			for _, exp := range tt.expected {
				if !found[exp] {
					t.Errorf("Expected to find term %q but didn't", exp)
				}
			}

			if len(usages) != len(tt.expected) {
				t.Errorf("Expected %d terms, got %d", len(tt.expected), len(usages))
			}
		})
	}
}

func TestTermUsageExtractorWithDocument(t *testing.T) {
	// Create a minimal document
	doc := &Document{
		Title:      "Test Regulation",
		Identifier: "TEST/2024",
		Chapters: []*Chapter{
			{
				Number: "I",
				Title:  "General Provisions",
				Articles: []*Article{
					{
						Number: 1,
						Title:  "Subject matter",
						Text:   "This Regulation applies to the processing of personal data by controllers and processors.",
					},
					{
						Number: 4,
						Title:  "Definitions",
						Text:   "For the purposes of this Regulation: (1) 'personal data' means any information...",
					},
					{
						Number: 5,
						Title:  "Principles",
						Text:   "Personal data shall be processed lawfully. The controller must ensure compliance.",
					},
				},
			},
		},
	}

	definitions := []*DefinedTerm{
		{Number: 1, Term: "personal data", NormalizedTerm: "personal data", ArticleRef: 4},
		{Number: 2, Term: "controller", NormalizedTerm: "controller", ArticleRef: 4},
		{Number: 3, Term: "processor", NormalizedTerm: "processor", ArticleRef: 4},
	}

	extractor := NewTermUsageExtractor(definitions)
	usages := extractor.ExtractFromDocument(doc)

	// Should find usages in Article 1 and 5, but not in Article 4 (definitions)
	articleUsages := make(map[int][]string)
	for _, u := range usages {
		articleUsages[u.ArticleNum] = append(articleUsages[u.ArticleNum], u.NormalizedTerm)
	}

	// Article 4 should have no usages (it's the definition article)
	if len(articleUsages[4]) > 0 {
		t.Error("Should not find term usages in Article 4 (definitions)")
	}

	// Article 1 should have usages
	if len(articleUsages[1]) == 0 {
		t.Error("Should find term usages in Article 1")
	}

	// Check that personal data was found in Article 1
	foundPersonalData := false
	for _, term := range articleUsages[1] {
		if term == "personal data" {
			foundPersonalData = true
			break
		}
	}
	if !foundPersonalData {
		t.Error("Expected to find 'personal data' in Article 1")
	}
}

func TestTermUsageStats(t *testing.T) {
	definitions := []*DefinedTerm{
		{Number: 1, Term: "personal data", NormalizedTerm: "personal data", ArticleRef: 4},
		{Number: 2, Term: "controller", NormalizedTerm: "controller", ArticleRef: 4},
		{Number: 3, Term: "processor", NormalizedTerm: "processor", ArticleRef: 4},
		{Number: 4, Term: "unused term", NormalizedTerm: "unused term", ArticleRef: 4},
	}

	usages := []*TermUsage{
		{NormalizedTerm: "personal data", ArticleNum: 1, Count: 2},
		{NormalizedTerm: "personal data", ArticleNum: 5, Count: 1},
		{NormalizedTerm: "controller", ArticleNum: 1, Count: 1},
		{NormalizedTerm: "controller", ArticleNum: 5, Count: 3},
		{NormalizedTerm: "controller", ArticleNum: 6, Count: 1},
	}

	stats := CalculateUsageStats(usages, definitions)

	if stats.TotalUsages != 5 {
		t.Errorf("Expected 5 total usages, got %d", stats.TotalUsages)
	}

	if stats.UniqueTermsUsed != 2 {
		t.Errorf("Expected 2 unique terms used, got %d", stats.UniqueTermsUsed)
	}

	if stats.ArticlesWithTerms != 3 {
		t.Errorf("Expected 3 articles with terms, got %d", stats.ArticlesWithTerms)
	}

	// Check unused definitions
	if len(stats.UnusedDefinitions) != 2 {
		t.Errorf("Expected 2 unused definitions, got %d", len(stats.UnusedDefinitions))
	}

	foundUnused := false
	for _, term := range stats.UnusedDefinitions {
		if term == "unused term" {
			foundUnused = true
			break
		}
	}
	if !foundUnused {
		t.Error("Expected 'unused term' to be in unused definitions")
	}
}

func TestTermUsageIndex(t *testing.T) {
	usages := []*TermUsage{
		{NormalizedTerm: "personal data", ArticleNum: 1, Count: 2},
		{NormalizedTerm: "personal data", ArticleNum: 5, Count: 1},
		{NormalizedTerm: "controller", ArticleNum: 1, Count: 1},
		{NormalizedTerm: "controller", ArticleNum: 5, Count: 3},
	}

	idx := NewTermUsageIndex(usages)

	// Test GetByArticle
	art1Usages := idx.GetByArticle(1)
	if len(art1Usages) != 2 {
		t.Errorf("Expected 2 usages in Article 1, got %d", len(art1Usages))
	}

	// Test GetByTerm
	pdUsages := idx.GetByTerm("personal data")
	if len(pdUsages) != 2 {
		t.Errorf("Expected 2 usages of 'personal data', got %d", len(pdUsages))
	}

	// Test ArticlesUsingTerm
	articles := idx.ArticlesUsingTerm("controller")
	if len(articles) != 2 {
		t.Errorf("Expected controller used in 2 articles, got %d", len(articles))
	}

	// Test TermsUsedInArticle
	terms := idx.TermsUsedInArticle(1)
	if len(terms) != 2 {
		t.Errorf("Expected 2 terms in Article 1, got %d", len(terms))
	}
}

func TestGDPRTermUsage(t *testing.T) {
	// Parse actual GDPR test data
	doc, err := parseTestFile("../../testdata/gdpr.txt")
	if err != nil {
		t.Skipf("Skipping GDPR test: %v", err)
	}

	// Extract definitions
	defExtractor := NewDefinitionExtractor()
	definitions := defExtractor.ExtractDefinitions(doc)

	if len(definitions) == 0 {
		t.Skip("No definitions extracted from GDPR")
	}

	// Create term usage extractor
	extractor := NewTermUsageExtractor(definitions)
	usages := extractor.ExtractFromDocument(doc)

	t.Logf("Found %d term usages across GDPR", len(usages))

	// Calculate stats
	stats := CalculateUsageStats(usages, definitions)
	t.Logf("Unique terms used: %d", stats.UniqueTermsUsed)
	t.Logf("Articles with terms: %d", stats.ArticlesWithTerms)
	t.Logf("Unused definitions: %d", len(stats.UnusedDefinitions))

	// Log top used terms
	t.Log("Most used terms:")
	for i, ts := range stats.MostUsedTerms {
		if i >= 5 {
			break
		}
		t.Logf("  %s: %d uses across %d articles", ts.Term, ts.TotalUses, ts.ArticleCount)
	}

	// We should have significant term usage
	if len(usages) < 50 {
		t.Errorf("Expected at least 50 term usages in GDPR, got %d", len(usages))
	}

	// Personal data should be commonly used
	idx := NewTermUsageIndex(usages)
	pdUsages := idx.GetByTerm("personal data")
	if len(pdUsages) < 10 {
		t.Errorf("Expected 'personal data' to be used in at least 10 articles, got %d", len(pdUsages))
	}
}

func parseTestFile(path string) (*Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	parser := NewParser()
	return parser.Parse(file)
}
