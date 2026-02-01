package extract

import (
	"fmt"
	"regexp"
	"strings"
)

// DefinedTerm represents a fully extracted definition with its complete text and metadata.
type DefinedTerm struct {
	Number         int                   `json:"number"`
	Term           string                `json:"term"`
	NormalizedTerm string                `json:"normalized_term"`
	Definition     string                `json:"definition"`
	Scope          string                `json:"scope"`
	ArticleRef     int                   `json:"article_ref"`
	SubPoints      []*DefinitionSubPoint `json:"sub_points,omitempty"`
	References     []string              `json:"references,omitempty"`
}

// DefinitionSubPoint represents a sub-point within a definition (e.g., (a), (b)).
type DefinitionSubPoint struct {
	Letter string `json:"letter"`
	Text   string `json:"text"`
}

// allQuoteChars is the character class matching all supported quote characters:
// ASCII quotes (', "), typographic quotes (', ', ", "), and Unicode quotes (U+2018, U+2019, U+201C, U+201D).
const allQuoteChars = `'''"\x{2018}\x{2019}\x{201C}\x{201D}`

// DefinitionExtractor extracts defined terms from regulatory text.
type DefinitionExtractor struct {
	definitionStartPattern *regexp.Regexp
	subPointPattern        *regexp.Regexp
	referencePattern       *regexp.Regexp
	uscDefinitionPattern   *regexp.Regexp
}

// NewDefinitionExtractor creates a new DefinitionExtractor.
func NewDefinitionExtractor() *DefinitionExtractor {
	return &DefinitionExtractor{
		// Matches "(1) 'term' means" or "(1) 'term' of the data subject means"
		definitionStartPattern: regexp.MustCompile(`^\((\d+)\)\s+[` + allQuoteChars + `]([^` + allQuoteChars + `]+)[` + allQuoteChars + `](?:\s+of[^m]*?)?\s*means[:\s]`),
		// Matches "(a) " sub-points within definitions
		subPointPattern: regexp.MustCompile(`^\(([a-z])\)\s+(.*)$`),
		// Matches references to other defined terms in quotes (all quote styles)
		referencePattern: regexp.MustCompile(`[` + allQuoteChars + `]([^` + allQuoteChars + `]+)[` + allQuoteChars + `]`),
		// Matches USC-style definitions: "  a The term \u201c...\u201d means/includes ..."
		// Format: optional leading whitespace, letter, space, "The term" + quoted term + means/includes
		uscDefinitionPattern: regexp.MustCompile(`^\s+([a-zA-Z])\s+[Tt]he\s+term\s+[` + allQuoteChars + `]([^` + allQuoteChars + `]+)[` + allQuoteChars + `]\s+(?:means|includes)[:\s,]`),
	}
}

// ExtractDefinitions extracts all definitions from a document.
// Automatically detects definition sections by title matching and
// definition pattern density, supporting multiple definition sections.
// Tries EU-style, then US state-style, then USC-style extraction.
func (e *DefinitionExtractor) ExtractDefinitions(doc *Document) []*DefinedTerm {
	definitions := make([]*DefinedTerm, 0)

	defArticles := e.findDefinitionsArticles(doc)
	for _, defArticle := range defArticles {
		// Try EU-style extraction first (numbered definitions: (1) 'term' means)
		articleDefs := e.extractEUDefinitions(defArticle)

		// If no definitions found, try US state-style (lettered: (a) 'term' means)
		if len(articleDefs) == 0 {
			articleDefs = e.extractUSDefinitions(defArticle)
		}

		// If still none, try USC-style (indented letter: a The term "..." means)
		if len(articleDefs) == 0 {
			articleDefs = e.extractUSCDefinitions(defArticle)
		}

		definitions = append(definitions, articleDefs...)
	}

	return definitions
}

// findDefinitionsArticles locates all articles containing definitions.
// Uses title-based detection with fallback to definition pattern density scanning.
func (e *DefinitionExtractor) findDefinitionsArticles(doc *Document) []*Article {
	matchedArticles := make([]*Article, 0)
	matchedArticleNumbers := make(map[int]bool)

	definitionTitlePattern := regexp.MustCompile(`(?i)definitions?|interpretation|terms`)

	// Search by title
	allArticles := doc.AllArticles()
	for _, article := range allArticles {
		if definitionTitlePattern.MatchString(article.Title) && article.Text != "" && !matchedArticleNumbers[article.Number] {
			matchedArticles = append(matchedArticles, article)
			matchedArticleNumbers[article.Number] = true
		}
	}

	// Density detection: scan remaining articles for high definition pattern density.
	// Checks all definition patterns (EU, US state, and USC) to find definition sections.
	if len(matchedArticles) == 0 {
		const definitionDensityThreshold = 3
		for _, article := range allArticles {
			if article.Text == "" || matchedArticleNumbers[article.Number] {
				continue
			}
			lines := strings.Split(article.Text, "\n")
			definitionMatchCount := 0
			for _, line := range lines {
				if e.definitionStartPattern.MatchString(line) || e.uscDefinitionPattern.MatchString(line) {
					definitionMatchCount++
				}
			}
			if definitionMatchCount >= definitionDensityThreshold {
				matchedArticles = append(matchedArticles, article)
				matchedArticleNumbers[article.Number] = true
			}
		}
	}

	return matchedArticles
}

// extractEUDefinitions extracts EU-style definitions (numbered: (1) 'term' means...).
func (e *DefinitionExtractor) extractEUDefinitions(defArticle *Article) []*DefinedTerm {
	definitions := make([]*DefinedTerm, 0)

	lines := strings.Split(defArticle.Text, "\n")

	var currentDef *DefinedTerm
	var textBuffer strings.Builder
	var currentSubPoint *DefinitionSubPoint
	var subPointBuffer strings.Builder

	flushSubPoint := func() {
		if currentSubPoint != nil && currentDef != nil {
			currentSubPoint.Text = strings.TrimSpace(subPointBuffer.String())
			if currentSubPoint.Text != "" {
				currentDef.SubPoints = append(currentDef.SubPoints, currentSubPoint)
			}
			currentSubPoint = nil
			subPointBuffer.Reset()
		}
	}

	flushDefinition := func() {
		flushSubPoint()
		if currentDef != nil {
			if len(currentDef.SubPoints) == 0 {
				currentDef.Definition = strings.TrimSpace(textBuffer.String())
			} else {
				// For definitions with sub-points, the main text is the intro
				currentDef.Definition = strings.TrimSpace(textBuffer.String())
			}
			// Extract references to other terms
			currentDef.References = e.extractReferences(currentDef.Definition, currentDef.SubPoints)
			definitions = append(definitions, currentDef)
			currentDef = nil
			textBuffer.Reset()
		}
	}

	for _, line := range lines {
		// Check for new definition
		if m := e.definitionStartPattern.FindStringSubmatch(line); m != nil {
			flushDefinition()

			num := mustAtoi(m[1])
			term := strings.TrimSpace(m[2])

			currentDef = &DefinedTerm{
				Number:         num,
				Term:           term,
				NormalizedTerm: normalizeTerm(term),
				Scope:          fmt.Sprintf("Article %d", defArticle.Number),
				ArticleRef:     defArticle.Number,
				SubPoints:      make([]*DefinitionSubPoint, 0),
			}

			// Extract the rest of the line after "means" or "means:"
			rest := e.extractAfterMeans(line)
			if rest != "" {
				textBuffer.WriteString(rest)
			}
			continue
		}

		// Check for sub-point within current definition
		if currentDef != nil {
			if m := e.subPointPattern.FindStringSubmatch(line); m != nil {
				flushSubPoint()
				currentSubPoint = &DefinitionSubPoint{
					Letter: m[1],
				}
				subPointBuffer.WriteString(m[2])
				continue
			}
		}

		// Continuation line
		if currentDef != nil && line != "" {
			if currentSubPoint != nil {
				// Continue sub-point text
				if subPointBuffer.Len() > 0 {
					subPointBuffer.WriteString(" ")
				}
				subPointBuffer.WriteString(line)
			} else {
				// Continue main definition text
				if textBuffer.Len() > 0 {
					textBuffer.WriteString(" ")
				}
				textBuffer.WriteString(line)
			}
		}
	}

	// Flush final definition
	flushDefinition()

	return definitions
}

// extractUSDefinitions extracts US-style definitions (lettered: (a) 'term' means...).
func (e *DefinitionExtractor) extractUSDefinitions(defArticle *Article) []*DefinedTerm {
	definitions := make([]*DefinedTerm, 0)

	// US-style definition pattern: (a) 'term' means or (a) "term" means
	usDefPattern := regexp.MustCompile(`^\(([a-z])\)\s+['''"\x{2018}\x{2019}]([^'''"\x{2018}\x{2019}]+)['''"\x{2018}\x{2019}]\s+means[:\s]`)

	lines := strings.Split(defArticle.Text, "\n")

	var currentDef *DefinedTerm
	var textBuffer strings.Builder
	defNum := 0

	flushDefinition := func() {
		if currentDef != nil {
			currentDef.Definition = strings.TrimSpace(textBuffer.String())
			currentDef.References = e.extractReferences(currentDef.Definition, nil)
			definitions = append(definitions, currentDef)
			currentDef = nil
			textBuffer.Reset()
		}
	}

	for _, line := range lines {
		// Check for new definition
		if m := usDefPattern.FindStringSubmatch(line); m != nil {
			flushDefinition()

			defNum++
			term := strings.TrimSpace(m[2])

			currentDef = &DefinedTerm{
				Number:         defNum,
				Term:           term,
				NormalizedTerm: normalizeTerm(term),
				Scope:          "Section " + defArticle.Title,
				ArticleRef:     defArticle.Number,
				SubPoints:      make([]*DefinitionSubPoint, 0),
			}

			// Extract the rest of the line after "means"
			rest := e.extractAfterMeans(line)
			if rest != "" {
				textBuffer.WriteString(rest)
			}
			continue
		}

		// Continuation line for current definition
		if currentDef != nil && line != "" {
			// Stop if we hit the next lettered point that's not a definition
			if matched, _ := regexp.MatchString(`^\([a-z]\)\s+[^'''"]+`, line); matched {
				// Check if this is a new definition or just a sub-point
				if usDefPattern.MatchString(line) {
					// This is a new definition, will be handled next iteration
					continue
				}
				// Not a definition, might be end of definitions section
			}

			if textBuffer.Len() > 0 {
				textBuffer.WriteString(" ")
			}
			textBuffer.WriteString(strings.TrimSpace(line))
		}
	}

	// Flush final definition
	flushDefinition()

	return definitions
}

// extractUSCDefinitions extracts USC-style definitions (indented letter: a The term "..." means/includes...).
// USC format uses leading whitespace + letter + "The term" + curly-quoted term + means/includes.
func (e *DefinitionExtractor) extractUSCDefinitions(defArticle *Article) []*DefinedTerm {
	definitions := make([]*DefinedTerm, 0)

	lines := strings.Split(defArticle.Text, "\n")

	var currentDef *DefinedTerm
	var textBuffer strings.Builder
	defNum := 0

	flushDefinition := func() {
		if currentDef != nil {
			currentDef.Definition = strings.TrimSpace(textBuffer.String())
			currentDef.References = e.extractReferences(currentDef.Definition, nil)
			definitions = append(definitions, currentDef)
			currentDef = nil
			textBuffer.Reset()
		}
	}

	for _, line := range lines {
		// Check for new USC-style definition
		if m := e.uscDefinitionPattern.FindStringSubmatch(line); m != nil {
			flushDefinition()

			defNum++
			term := strings.TrimSpace(m[2])

			currentDef = &DefinedTerm{
				Number:         defNum,
				Term:           term,
				NormalizedTerm: normalizeTerm(term),
				Scope:          fmt.Sprintf("Section %s", defArticle.Title),
				ArticleRef:     defArticle.Number,
				SubPoints:      make([]*DefinitionSubPoint, 0),
			}

			// Extract the rest of the line after "means" or "includes"
			rest := e.extractAfterDefinitionVerb(line)
			if rest != "" {
				textBuffer.WriteString(rest)
			}
			continue
		}

		// Continuation line for current definition
		if currentDef != nil && line != "" {
			trimmedLine := strings.TrimSpace(line)
			if trimmedLine == "" {
				continue
			}

			// Stop if we encounter another lettered definition start (without matching the full pattern)
			// USC continuation lines are typically indented; a new unindented section header ends the definition
			if textBuffer.Len() > 0 {
				textBuffer.WriteString(" ")
			}
			textBuffer.WriteString(trimmedLine)
		}
	}

	// Flush final definition
	flushDefinition()

	return definitions
}

// extractAfterDefinitionVerb extracts text after "means", "means:", "includes", or "includes:" in a line.
func (e *DefinitionExtractor) extractAfterDefinitionVerb(line string) string {
	lineLower := strings.ToLower(line)
	verbPatterns := []string{"means: ", "means ", "means:", "includes: ", "includes ", "includes,", "includes:"}

	for _, verbPattern := range verbPatterns {
		idx := strings.Index(lineLower, verbPattern)
		if idx != -1 {
			rest := line[idx+len(verbPattern):]
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

// extractAfterMeans extracts the text after "means", "means:", "includes", or "includes:" in a line.
func (e *DefinitionExtractor) extractAfterMeans(line string) string {
	// Find "means" or "includes" followed by optional ":" and space
	patterns := []string{"means: ", "means ", "means:", "includes: ", "includes ", "includes,", "includes:"}
	lineLower := strings.ToLower(line)

	for _, pattern := range patterns {
		idx := strings.Index(lineLower, pattern)
		if idx != -1 {
			rest := line[idx+len(pattern):]
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

// extractReferences finds references to other defined terms within definition text.
func (e *DefinitionExtractor) extractReferences(mainText string, subPoints []*DefinitionSubPoint) []string {
	refs := make(map[string]bool)

	// Check main text
	matches := e.referencePattern.FindAllStringSubmatch(mainText, -1)
	for _, m := range matches {
		term := strings.TrimSpace(m[1])
		if term != "" {
			refs[normalizeTerm(term)] = true
		}
	}

	// Check sub-points
	for _, sp := range subPoints {
		matches := e.referencePattern.FindAllStringSubmatch(sp.Text, -1)
		for _, m := range matches {
			term := strings.TrimSpace(m[1])
			if term != "" {
				refs[normalizeTerm(term)] = true
			}
		}
	}

	// Convert to slice
	result := make([]string, 0, len(refs))
	for ref := range refs {
		result = append(result, ref)
	}
	return result
}

// normalizeTerm normalizes a term for consistent lookup.
func normalizeTerm(term string) string {
	// Lowercase and trim whitespace
	normalized := strings.ToLower(strings.TrimSpace(term))
	// Replace multiple spaces with single space
	normalized = strings.Join(strings.Fields(normalized), " ")
	return normalized
}

// DefinitionLookup provides fast lookup of definitions by term.
type DefinitionLookup struct {
	byNumber     map[int]*DefinedTerm
	byTerm       map[string]*DefinedTerm
	byNormalized map[string]*DefinedTerm
	all          []*DefinedTerm
}

// NewDefinitionLookup creates a lookup from a slice of definitions.
func NewDefinitionLookup(definitions []*DefinedTerm) *DefinitionLookup {
	lookup := &DefinitionLookup{
		byNumber:     make(map[int]*DefinedTerm),
		byTerm:       make(map[string]*DefinedTerm),
		byNormalized: make(map[string]*DefinedTerm),
		all:          definitions,
	}

	for _, def := range definitions {
		lookup.byNumber[def.Number] = def
		lookup.byTerm[def.Term] = def
		lookup.byNormalized[def.NormalizedTerm] = def
	}

	return lookup
}

// GetByNumber returns a definition by its number.
func (l *DefinitionLookup) GetByNumber(num int) *DefinedTerm {
	return l.byNumber[num]
}

// GetByTerm returns a definition by its exact term.
func (l *DefinitionLookup) GetByTerm(term string) *DefinedTerm {
	return l.byTerm[term]
}

// GetByNormalizedTerm returns a definition by normalized term (case-insensitive).
func (l *DefinitionLookup) GetByNormalizedTerm(term string) *DefinedTerm {
	return l.byNormalized[normalizeTerm(term)]
}

// All returns all definitions.
func (l *DefinitionLookup) All() []*DefinedTerm {
	return l.all
}

// Count returns the number of definitions.
func (l *DefinitionLookup) Count() int {
	return len(l.all)
}

// FindReferencedBy returns all definitions that reference the given term.
func (l *DefinitionLookup) FindReferencedBy(term string) []*DefinedTerm {
	normalized := normalizeTerm(term)
	var result []*DefinedTerm

	for _, def := range l.all {
		for _, ref := range def.References {
			if ref == normalized {
				result = append(result, def)
				break
			}
		}
	}

	return result
}

// DefinitionStats holds statistics about extracted definitions.
type DefinitionStats struct {
	TotalDefinitions     int `json:"total_definitions"`
	WithSubPoints        int `json:"with_sub_points"`
	TotalSubPoints       int `json:"total_sub_points"`
	WithReferences       int `json:"with_references"`
	TotalReferences      int `json:"total_references"`
	AverageDefinitionLen int `json:"average_definition_len"`
}

// Stats calculates statistics about the definitions.
func (l *DefinitionLookup) Stats() DefinitionStats {
	stats := DefinitionStats{
		TotalDefinitions: len(l.all),
	}

	totalLen := 0
	for _, def := range l.all {
		if len(def.SubPoints) > 0 {
			stats.WithSubPoints++
			stats.TotalSubPoints += len(def.SubPoints)
		}
		if len(def.References) > 0 {
			stats.WithReferences++
			stats.TotalReferences += len(def.References)
		}
		totalLen += len(def.Definition)
		for _, sp := range def.SubPoints {
			totalLen += len(sp.Text)
		}
	}

	if len(l.all) > 0 {
		stats.AverageDefinitionLen = totalLen / len(l.all)
	}

	return stats
}
