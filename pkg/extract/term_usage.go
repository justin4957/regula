package extract

import (
	"regexp"
	"strings"
)

// TermUsage represents the usage of a defined term in an article.
type TermUsage struct {
	// Term information
	Term           string `json:"term"`
	NormalizedTerm string `json:"normalized_term"`
	DefinitionNum  int    `json:"definition_num"`

	// Usage location
	ArticleNum   int    `json:"article_num"`
	ParagraphNum int    `json:"paragraph_num,omitempty"`
	PointLetter  string `json:"point_letter,omitempty"`

	// Match details
	MatchedText string `json:"matched_text"`
	TextOffset  int    `json:"text_offset"`
	Count       int    `json:"count"` // Number of times term appears in this location
}

// TermUsageExtractor finds where defined terms are used throughout the document.
type TermUsageExtractor struct {
	definitions *DefinitionLookup
	patterns    map[string]*regexp.Regexp
}

// NewTermUsageExtractor creates a new extractor with the given definitions.
func NewTermUsageExtractor(definitions []*DefinedTerm) *TermUsageExtractor {
	lookup := NewDefinitionLookup(definitions)

	e := &TermUsageExtractor{
		definitions: lookup,
		patterns:    make(map[string]*regexp.Regexp),
	}

	// Build regex patterns for each term
	for _, def := range definitions {
		pattern := e.buildTermPattern(def.Term)
		if pattern != nil {
			e.patterns[def.NormalizedTerm] = pattern
		}
	}

	return e
}

// buildTermPattern creates a regex pattern that matches variations of a term.
func (e *TermUsageExtractor) buildTermPattern(term string) *regexp.Regexp {
	// Escape special regex characters
	escaped := regexp.QuoteMeta(term)

	// Build pattern that matches:
	// 1. Exact term
	// 2. Term with 's' suffix (plurals)
	// 3. Term in quotes
	// Use word boundaries to avoid partial matches
	pattern := `(?i)\b` + escaped + `(?:s|'s)?\b`

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}
	return re
}

// ExtractFromDocument finds all term usages in a document.
func (e *TermUsageExtractor) ExtractFromDocument(doc *Document) []*TermUsage {
	if doc == nil {
		return nil
	}

	usages := make([]*TermUsage, 0)

	// Track unique usages per article to avoid duplicates
	seen := make(map[string]bool)

	for _, chapter := range doc.Chapters {
		// Process sections
		for _, section := range chapter.Sections {
			for _, article := range section.Articles {
				articleUsages := e.extractFromArticle(article, seen)
				usages = append(usages, articleUsages...)
			}
		}

		// Process articles directly in chapter
		for _, article := range chapter.Articles {
			articleUsages := e.extractFromArticle(article, seen)
			usages = append(usages, articleUsages...)
		}
	}

	return usages
}

// extractFromArticle finds term usages within a single article.
func (e *TermUsageExtractor) extractFromArticle(article *Article, seen map[string]bool) []*TermUsage {
	usages := make([]*TermUsage, 0)

	// Skip Article 4 (definitions article) - terms are defined there, not "used"
	if article.Number == 4 {
		return usages
	}

	// Check article text
	if article.Text != "" {
		textUsages := e.findTermsInText(article.Text, article.Number, 0, "")
		for _, u := range textUsages {
			key := e.usageKey(u)
			if !seen[key] {
				seen[key] = true
				usages = append(usages, u)
			}
		}
	}

	// Check paragraphs
	for _, para := range article.Paragraphs {
		if para.Text != "" {
			textUsages := e.findTermsInText(para.Text, article.Number, para.Number, "")
			for _, u := range textUsages {
				key := e.usageKey(u)
				if !seen[key] {
					seen[key] = true
					usages = append(usages, u)
				}
			}
		}

		// Check points within paragraph
		for _, point := range para.Points {
			if point.Text != "" {
				textUsages := e.findTermsInText(point.Text, article.Number, para.Number, point.Letter)
				for _, u := range textUsages {
					key := e.usageKey(u)
					if !seen[key] {
						seen[key] = true
						usages = append(usages, u)
					}
				}
			}
		}
	}

	return usages
}

// findTermsInText finds all defined terms used in a piece of text.
func (e *TermUsageExtractor) findTermsInText(text string, articleNum, paraNum int, pointLetter string) []*TermUsage {
	usages := make([]*TermUsage, 0)

	for normalizedTerm, pattern := range e.patterns {
		matches := pattern.FindAllStringIndex(text, -1)
		if len(matches) > 0 {
			def := e.definitions.GetByNormalizedTerm(normalizedTerm)
			if def == nil {
				continue
			}

			// Take the first match for the matched text
			matchedText := text[matches[0][0]:matches[0][1]]

			usage := &TermUsage{
				Term:           def.Term,
				NormalizedTerm: normalizedTerm,
				DefinitionNum:  def.Number,
				ArticleNum:     articleNum,
				ParagraphNum:   paraNum,
				PointLetter:    pointLetter,
				MatchedText:    matchedText,
				TextOffset:     matches[0][0],
				Count:          len(matches),
			}
			usages = append(usages, usage)
		}
	}

	return usages
}

// usageKey creates a unique key for a term usage (term + article).
func (e *TermUsageExtractor) usageKey(u *TermUsage) string {
	return u.NormalizedTerm + ":" + itoa(u.ArticleNum)
}

// TermUsageStats holds statistics about term usage in a document.
type TermUsageStats struct {
	TotalUsages        int            `json:"total_usages"`
	UniqueTermsUsed    int            `json:"unique_terms_used"`
	ArticlesWithTerms  int            `json:"articles_with_terms"`
	MostUsedTerms      []TermUseStat  `json:"most_used_terms"`
	TermsByArticle     map[int]int    `json:"terms_by_article"`
	UnusedDefinitions  []string       `json:"unused_definitions"`
}

// TermUseStat holds usage statistics for a single term.
type TermUseStat struct {
	Term       string `json:"term"`
	TotalUses  int    `json:"total_uses"`
	ArticleCount int  `json:"article_count"`
}

// CalculateUsageStats computes statistics about term usage.
func CalculateUsageStats(usages []*TermUsage, definitions []*DefinedTerm) *TermUsageStats {
	stats := &TermUsageStats{
		TotalUsages:    len(usages),
		TermsByArticle: make(map[int]int),
	}

	// Track unique terms and articles
	uniqueTerms := make(map[string]bool)
	articlesWithTerms := make(map[int]bool)
	termCounts := make(map[string]int)
	termArticles := make(map[string]map[int]bool)
	usedTerms := make(map[string]bool)

	for _, u := range usages {
		uniqueTerms[u.NormalizedTerm] = true
		articlesWithTerms[u.ArticleNum] = true
		stats.TermsByArticle[u.ArticleNum]++
		termCounts[u.NormalizedTerm] += u.Count
		usedTerms[u.NormalizedTerm] = true

		if termArticles[u.NormalizedTerm] == nil {
			termArticles[u.NormalizedTerm] = make(map[int]bool)
		}
		termArticles[u.NormalizedTerm][u.ArticleNum] = true
	}

	stats.UniqueTermsUsed = len(uniqueTerms)
	stats.ArticlesWithTerms = len(articlesWithTerms)

	// Build most used terms list
	for term, count := range termCounts {
		def := NewDefinitionLookup(definitions).GetByNormalizedTerm(term)
		if def != nil {
			stats.MostUsedTerms = append(stats.MostUsedTerms, TermUseStat{
				Term:         def.Term,
				TotalUses:    count,
				ArticleCount: len(termArticles[term]),
			})
		}
	}

	// Sort by total uses (descending)
	for i := 0; i < len(stats.MostUsedTerms); i++ {
		for j := i + 1; j < len(stats.MostUsedTerms); j++ {
			if stats.MostUsedTerms[j].TotalUses > stats.MostUsedTerms[i].TotalUses {
				stats.MostUsedTerms[i], stats.MostUsedTerms[j] = stats.MostUsedTerms[j], stats.MostUsedTerms[i]
			}
		}
	}

	// Limit to top 10
	if len(stats.MostUsedTerms) > 10 {
		stats.MostUsedTerms = stats.MostUsedTerms[:10]
	}

	// Find unused definitions
	for _, def := range definitions {
		if !usedTerms[def.NormalizedTerm] {
			stats.UnusedDefinitions = append(stats.UnusedDefinitions, def.Term)
		}
	}

	return stats
}

// GroupByArticle groups usages by article number.
func GroupByArticle(usages []*TermUsage) map[int][]*TermUsage {
	grouped := make(map[int][]*TermUsage)
	for _, u := range usages {
		grouped[u.ArticleNum] = append(grouped[u.ArticleNum], u)
	}
	return grouped
}

// GroupByTerm groups usages by normalized term.
func GroupByTerm(usages []*TermUsage) map[string][]*TermUsage {
	grouped := make(map[string][]*TermUsage)
	for _, u := range usages {
		grouped[u.NormalizedTerm] = append(grouped[u.NormalizedTerm], u)
	}
	return grouped
}

// TermUsageIndex provides fast lookup of term usages.
type TermUsageIndex struct {
	byArticle map[int][]*TermUsage
	byTerm    map[string][]*TermUsage
	all       []*TermUsage
}

// NewTermUsageIndex creates a new index from usages.
func NewTermUsageIndex(usages []*TermUsage) *TermUsageIndex {
	return &TermUsageIndex{
		byArticle: GroupByArticle(usages),
		byTerm:    GroupByTerm(usages),
		all:       usages,
	}
}

// GetByArticle returns all term usages in an article.
func (idx *TermUsageIndex) GetByArticle(articleNum int) []*TermUsage {
	return idx.byArticle[articleNum]
}

// GetByTerm returns all usages of a specific term.
func (idx *TermUsageIndex) GetByTerm(normalizedTerm string) []*TermUsage {
	return idx.byTerm[strings.ToLower(normalizedTerm)]
}

// All returns all usages.
func (idx *TermUsageIndex) All() []*TermUsage {
	return idx.all
}

// ArticlesUsingTerm returns article numbers that use a specific term.
func (idx *TermUsageIndex) ArticlesUsingTerm(normalizedTerm string) []int {
	usages := idx.byTerm[strings.ToLower(normalizedTerm)]
	articles := make(map[int]bool)
	for _, u := range usages {
		articles[u.ArticleNum] = true
	}

	result := make([]int, 0, len(articles))
	for artNum := range articles {
		result = append(result, artNum)
	}
	return result
}

// TermsUsedInArticle returns normalized terms used in a specific article.
func (idx *TermUsageIndex) TermsUsedInArticle(articleNum int) []string {
	usages := idx.byArticle[articleNum]
	terms := make(map[string]bool)
	for _, u := range usages {
		terms[u.NormalizedTerm] = true
	}

	result := make([]string, 0, len(terms))
	for term := range terms {
		result = append(result, term)
	}
	return result
}
