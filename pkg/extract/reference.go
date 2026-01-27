package extract

import (
	"regexp"
	"sort"
	"strings"
)

// ReferenceType indicates whether a reference is internal or external.
type ReferenceType string

const (
	ReferenceTypeInternal ReferenceType = "internal"
	ReferenceTypeExternal ReferenceType = "external"
)

// ReferenceTarget indicates what kind of element is being referenced.
type ReferenceTarget string

const (
	TargetArticle    ReferenceTarget = "article"
	TargetParagraph  ReferenceTarget = "paragraph"
	TargetPoint      ReferenceTarget = "point"
	TargetChapter    ReferenceTarget = "chapter"
	TargetSection    ReferenceTarget = "section"
	TargetDirective  ReferenceTarget = "directive"
	TargetRegulation ReferenceTarget = "regulation"
	TargetTreaty     ReferenceTarget = "treaty"
	TargetDecision   ReferenceTarget = "decision"
)

// Reference represents a detected cross-reference in the text.
type Reference struct {
	Type       ReferenceType   `json:"type"`
	Target     ReferenceTarget `json:"target"`
	RawText    string          `json:"raw_text"`
	Identifier string          `json:"identifier"`
	SubRef     string          `json:"sub_ref,omitempty"`

	// Location information
	SourceArticle   int `json:"source_article"`
	SourceParagraph int `json:"source_paragraph,omitempty"`
	TextOffset      int `json:"text_offset"`
	TextLength      int `json:"text_length"`

	// Parsed components (for internal references)
	ArticleNum   int    `json:"article_num,omitempty"`
	ParagraphNum int    `json:"paragraph_num,omitempty"`
	PointLetter  string `json:"point_letter,omitempty"`
	ChapterNum   string `json:"chapter_num,omitempty"`
	SectionNum   int    `json:"section_num,omitempty"`

	// For external references
	ExternalDoc string `json:"external_doc,omitempty"`
	DocYear     string `json:"doc_year,omitempty"`
	DocNumber   string `json:"doc_number,omitempty"`
}

// ReferenceExtractor detects cross-references in regulatory text.
type ReferenceExtractor struct {
	// Internal reference patterns
	articlePattern      *regexp.Regexp
	articleParenPattern *regexp.Regexp
	articlesPattern     *regexp.Regexp
	paragraphPattern    *regexp.Regexp
	pointPattern        *regexp.Regexp
	pointsRangePattern  *regexp.Regexp
	chapterPattern      *regexp.Regexp
	sectionPattern      *regexp.Regexp

	// External reference patterns
	directivePattern    *regexp.Regexp
	regulationPattern   *regexp.Regexp
	regulationNoPattern *regexp.Regexp
	treatyPattern       *regexp.Regexp
	decisionPattern     *regexp.Regexp
}

// NewReferenceExtractor creates a new ReferenceExtractor with default patterns.
func NewReferenceExtractor() *ReferenceExtractor {
	return &ReferenceExtractor{
		// Internal references
		// Simple "Article 6" - overlap with parenthetical is handled in extractArticleRefs
		articlePattern:      regexp.MustCompile(`Article\s+(\d+)`),
		articleParenPattern: regexp.MustCompile(`Article\s+(\d+)\((\d+)\)(?:\(([a-z])\))?`),
		// "Articles 13 and 14" or "Articles 15 to 22"
		articlesPattern: regexp.MustCompile(`Articles\s+(\d+)\s+(?:and|to)\s+(\d+)`),
		// "paragraph 1" or "paragraph 2"
		paragraphPattern: regexp.MustCompile(`paragraph\s+(\d+)`),
		// "point (a)" or "point (f)"
		pointPattern: regexp.MustCompile(`point\s+\(([a-z])\)`),
		// "points (a) to (f)" or "points (a) and (b)"
		pointsRangePattern: regexp.MustCompile(`points\s+\(([a-z])\)\s+(?:to|and)\s+\(([a-z])\)`),
		// "Chapter III" or "Chapter VIII"
		chapterPattern: regexp.MustCompile(`Chapter\s+([IVX]+)`),
		// "Section 1" or "Section 2"
		sectionPattern: regexp.MustCompile(`Section\s+(\d+)`),

		// External references
		// "Directive 95/46/EC" or "Directive (EU) 2016/680"
		directivePattern: regexp.MustCompile(`Directive\s+(?:\(E[CU]\)\s+)?(\d+)/(\d+)(?:/EC|/EU)?`),
		// "Regulation (EU) 2016/679"
		regulationPattern: regexp.MustCompile(`Regulation\s+\(E[CU]\)\s+(\d+)/(\d+)`),
		// "Regulation (EC) No 45/2001"
		regulationNoPattern: regexp.MustCompile(`Regulation\s+\(E[CU]\)\s+No\s+(\d+)/(\d+)`),
		// "Treaty on the Functioning of the European Union" or "TFEU"
		treatyPattern: regexp.MustCompile(`(?:Treaty\s+on\s+the\s+Functioning\s+of\s+the\s+European\s+Union|TFEU|TEU)`),
		// "Decision 2010/87/EU"
		decisionPattern: regexp.MustCompile(`Decision\s+(\d+)/(\d+)/E[CU]`),
	}
}

// ExtractFromDocument extracts all references from a parsed document.
func (e *ReferenceExtractor) ExtractFromDocument(doc *Document) []*Reference {
	var refs []*Reference

	for _, article := range doc.AllArticles() {
		articleRefs := e.ExtractFromArticle(article)
		refs = append(refs, articleRefs...)
	}

	// Sort by source article, then by offset
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].SourceArticle != refs[j].SourceArticle {
			return refs[i].SourceArticle < refs[j].SourceArticle
		}
		return refs[i].TextOffset < refs[j].TextOffset
	})

	return refs
}

// ExtractFromArticle extracts all references from a single article.
func (e *ReferenceExtractor) ExtractFromArticle(article *Article) []*Reference {
	if article == nil || article.Text == "" {
		return nil
	}

	var refs []*Reference
	text := article.Text

	// Extract internal references
	refs = append(refs, e.extractArticleRefs(text, article.Number)...)
	refs = append(refs, e.extractParagraphRefs(text, article.Number)...)
	refs = append(refs, e.extractPointRefs(text, article.Number)...)
	refs = append(refs, e.extractChapterRefs(text, article.Number)...)
	refs = append(refs, e.extractSectionRefs(text, article.Number)...)

	// Extract external references
	refs = append(refs, e.extractDirectiveRefs(text, article.Number)...)
	refs = append(refs, e.extractRegulationRefs(text, article.Number)...)
	refs = append(refs, e.extractTreatyRefs(text, article.Number)...)
	refs = append(refs, e.extractDecisionRefs(text, article.Number)...)

	return refs
}

// extractArticleRefs extracts Article references.
func (e *ReferenceExtractor) extractArticleRefs(text string, sourceArticle int) []*Reference {
	var refs []*Reference

	// Article with parenthetical reference: "Article 6(1)" or "Article 6(1)(a)"
	matches := e.articleParenPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		rawText := text[match[0]:match[1]]
		articleNum := mustAtoi(text[match[2]:match[3]])
		paragraphNum := mustAtoi(text[match[4]:match[5]])

		ref := &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetArticle,
			RawText:       rawText,
			Identifier:    buildArticleIdentifier(articleNum, paragraphNum, ""),
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ArticleNum:    articleNum,
			ParagraphNum:  paragraphNum,
		}

		// Check for point letter
		if match[6] != -1 {
			ref.PointLetter = text[match[6]:match[7]]
			ref.Identifier = buildArticleIdentifier(articleNum, paragraphNum, ref.PointLetter)
		}

		refs = append(refs, ref)
	}

	// Simple Article reference: "Article 6"
	matches = e.articlePattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		// Skip if this overlaps with an articleParenPattern match
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]
		articleNum := mustAtoi(text[match[2]:match[3]])

		refs = append(refs, &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetArticle,
			RawText:       rawText,
			Identifier:    buildArticleIdentifier(articleNum, 0, ""),
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ArticleNum:    articleNum,
		})
	}

	// Multiple articles: "Articles 13 and 14"
	matches = e.articlesPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		rawText := text[match[0]:match[1]]
		startArticle := mustAtoi(text[match[2]:match[3]])
		endArticle := mustAtoi(text[match[4]:match[5]])

		refs = append(refs, &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetArticle,
			RawText:       rawText,
			Identifier:    buildArticlesRangeIdentifier(startArticle, endArticle),
			SubRef:        "range",
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ArticleNum:    startArticle,
		})
	}

	return refs
}

// extractParagraphRefs extracts paragraph references.
func (e *ReferenceExtractor) extractParagraphRefs(text string, sourceArticle int) []*Reference {
	var refs []*Reference

	matches := e.paragraphPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		rawText := text[match[0]:match[1]]
		paragraphNum := mustAtoi(text[match[2]:match[3]])

		refs = append(refs, &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetParagraph,
			RawText:       rawText,
			Identifier:    "paragraph " + text[match[2]:match[3]],
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ParagraphNum:  paragraphNum,
		})
	}

	return refs
}

// extractPointRefs extracts point references.
func (e *ReferenceExtractor) extractPointRefs(text string, sourceArticle int) []*Reference {
	var refs []*Reference

	// Points range: "points (a) to (f)"
	matches := e.pointsRangePattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		rawText := text[match[0]:match[1]]
		startLetter := text[match[2]:match[3]]
		endLetter := text[match[4]:match[5]]

		refs = append(refs, &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetPoint,
			RawText:       rawText,
			Identifier:    "points (" + startLetter + ") to (" + endLetter + ")",
			SubRef:        "range",
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			PointLetter:   startLetter,
		})
	}

	// Single point: "point (a)"
	matches = e.pointPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		// Skip if overlapping with range
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]
		letter := text[match[2]:match[3]]

		refs = append(refs, &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetPoint,
			RawText:       rawText,
			Identifier:    "point (" + letter + ")",
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			PointLetter:   letter,
		})
	}

	return refs
}

// extractChapterRefs extracts chapter references.
func (e *ReferenceExtractor) extractChapterRefs(text string, sourceArticle int) []*Reference {
	var refs []*Reference

	matches := e.chapterPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		rawText := text[match[0]:match[1]]
		chapterNum := text[match[2]:match[3]]

		refs = append(refs, &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetChapter,
			RawText:       rawText,
			Identifier:    "Chapter " + chapterNum,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ChapterNum:    chapterNum,
		})
	}

	return refs
}

// extractSectionRefs extracts section references.
func (e *ReferenceExtractor) extractSectionRefs(text string, sourceArticle int) []*Reference {
	var refs []*Reference

	matches := e.sectionPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		rawText := text[match[0]:match[1]]
		sectionNum := mustAtoi(text[match[2]:match[3]])

		refs = append(refs, &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetSection,
			RawText:       rawText,
			Identifier:    "Section " + text[match[2]:match[3]],
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			SectionNum:    sectionNum,
		})
	}

	return refs
}

// extractDirectiveRefs extracts Directive references.
func (e *ReferenceExtractor) extractDirectiveRefs(text string, sourceArticle int) []*Reference {
	var refs []*Reference

	matches := e.directivePattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		rawText := text[match[0]:match[1]]
		year := text[match[2]:match[3]]
		number := text[match[4]:match[5]]

		refs = append(refs, &Reference{
			Type:          ReferenceTypeExternal,
			Target:        TargetDirective,
			RawText:       rawText,
			Identifier:    "Directive " + year + "/" + number,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ExternalDoc:   "Directive",
			DocYear:       year,
			DocNumber:     number,
		})
	}

	return refs
}

// extractRegulationRefs extracts Regulation references.
func (e *ReferenceExtractor) extractRegulationRefs(text string, sourceArticle int) []*Reference {
	var refs []*Reference

	// "Regulation (EU) No 45/2001"
	matches := e.regulationNoPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		rawText := text[match[0]:match[1]]
		number := text[match[2]:match[3]]
		year := text[match[4]:match[5]]

		refs = append(refs, &Reference{
			Type:          ReferenceTypeExternal,
			Target:        TargetRegulation,
			RawText:       rawText,
			Identifier:    "Regulation (EU) No " + number + "/" + year,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ExternalDoc:   "Regulation",
			DocYear:       year,
			DocNumber:     number,
		})
	}

	// "Regulation (EU) 2016/679"
	matches = e.regulationPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		// Skip if overlapping with No pattern
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]
		year := text[match[2]:match[3]]
		number := text[match[4]:match[5]]

		refs = append(refs, &Reference{
			Type:          ReferenceTypeExternal,
			Target:        TargetRegulation,
			RawText:       rawText,
			Identifier:    "Regulation (EU) " + year + "/" + number,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ExternalDoc:   "Regulation",
			DocYear:       year,
			DocNumber:     number,
		})
	}

	return refs
}

// extractTreatyRefs extracts Treaty references.
func (e *ReferenceExtractor) extractTreatyRefs(text string, sourceArticle int) []*Reference {
	var refs []*Reference

	matches := e.treatyPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		rawText := text[match[0]:match[1]]

		var identifier string
		if strings.Contains(rawText, "TFEU") {
			identifier = "TFEU"
		} else if strings.Contains(rawText, "TEU") {
			identifier = "TEU"
		} else {
			identifier = "TFEU"
		}

		refs = append(refs, &Reference{
			Type:          ReferenceTypeExternal,
			Target:        TargetTreaty,
			RawText:       rawText,
			Identifier:    identifier,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ExternalDoc:   "Treaty",
		})
	}

	return refs
}

// extractDecisionRefs extracts Decision references.
func (e *ReferenceExtractor) extractDecisionRefs(text string, sourceArticle int) []*Reference {
	var refs []*Reference

	matches := e.decisionPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		rawText := text[match[0]:match[1]]
		year := text[match[2]:match[3]]
		number := text[match[4]:match[5]]

		refs = append(refs, &Reference{
			Type:          ReferenceTypeExternal,
			Target:        TargetDecision,
			RawText:       rawText,
			Identifier:    "Decision " + year + "/" + number,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ExternalDoc:   "Decision",
			DocYear:       year,
			DocNumber:     number,
		})
	}

	return refs
}

// isOverlapping checks if a match overlaps with any existing reference.
func (e *ReferenceExtractor) isOverlapping(start, end int, refs []*Reference) bool {
	for _, ref := range refs {
		refEnd := ref.TextOffset + ref.TextLength
		if start < refEnd && end > ref.TextOffset {
			return true
		}
	}
	return false
}

// buildArticleIdentifier creates a standardized article identifier.
func buildArticleIdentifier(article, paragraph int, point string) string {
	id := "Article " + itoa(article)
	if paragraph > 0 {
		id += "(" + itoa(paragraph) + ")"
	}
	if point != "" {
		id += "(" + point + ")"
	}
	return id
}

// buildArticlesRangeIdentifier creates an identifier for article ranges.
func buildArticlesRangeIdentifier(start, end int) string {
	return "Articles " + itoa(start) + "-" + itoa(end)
}

// ReferenceStats holds statistics about extracted references.
type ReferenceStats struct {
	TotalReferences   int            `json:"total_references"`
	InternalRefs      int            `json:"internal_refs"`
	ExternalRefs      int            `json:"external_refs"`
	ByTarget          map[string]int `json:"by_target"`
	ArticlesWithRefs  int            `json:"articles_with_refs"`
	UniqueIdentifiers int            `json:"unique_identifiers"`
}

// CalculateStats calculates statistics for a set of references.
func CalculateStats(refs []*Reference) ReferenceStats {
	stats := ReferenceStats{
		TotalReferences: len(refs),
		ByTarget:        make(map[string]int),
	}

	identifiers := make(map[string]bool)
	articlesSeen := make(map[int]bool)

	for _, ref := range refs {
		if ref.Type == ReferenceTypeInternal {
			stats.InternalRefs++
		} else {
			stats.ExternalRefs++
		}

		stats.ByTarget[string(ref.Target)]++
		identifiers[ref.Identifier] = true
		articlesSeen[ref.SourceArticle] = true
	}

	stats.UniqueIdentifiers = len(identifiers)
	stats.ArticlesWithRefs = len(articlesSeen)

	return stats
}

// ReferenceLookup provides indexed access to references.
type ReferenceLookup struct {
	all             []*Reference
	bySourceArticle map[int][]*Reference
	byTarget        map[ReferenceTarget][]*Reference
	byIdentifier    map[string][]*Reference
}

// NewReferenceLookup creates a lookup from a slice of references.
func NewReferenceLookup(refs []*Reference) *ReferenceLookup {
	lookup := &ReferenceLookup{
		all:             refs,
		bySourceArticle: make(map[int][]*Reference),
		byTarget:        make(map[ReferenceTarget][]*Reference),
		byIdentifier:    make(map[string][]*Reference),
	}

	for _, ref := range refs {
		lookup.bySourceArticle[ref.SourceArticle] = append(lookup.bySourceArticle[ref.SourceArticle], ref)
		lookup.byTarget[ref.Target] = append(lookup.byTarget[ref.Target], ref)
		lookup.byIdentifier[ref.Identifier] = append(lookup.byIdentifier[ref.Identifier], ref)
	}

	return lookup
}

// GetBySourceArticle returns all references from a specific article.
func (l *ReferenceLookup) GetBySourceArticle(articleNum int) []*Reference {
	return l.bySourceArticle[articleNum]
}

// GetByTarget returns all references of a specific target type.
func (l *ReferenceLookup) GetByTarget(target ReferenceTarget) []*Reference {
	return l.byTarget[target]
}

// GetByIdentifier returns all references with a specific identifier.
func (l *ReferenceLookup) GetByIdentifier(identifier string) []*Reference {
	return l.byIdentifier[identifier]
}

// FindReferencesTo finds all references to a specific article.
func (l *ReferenceLookup) FindReferencesTo(articleNum int) []*Reference {
	var result []*Reference
	for _, ref := range l.all {
		if ref.Target == TargetArticle && ref.ArticleNum == articleNum {
			result = append(result, ref)
		}
	}
	return result
}

// All returns all references.
func (l *ReferenceLookup) All() []*Reference {
	return l.all
}

// Count returns the total number of references.
func (l *ReferenceLookup) Count() int {
	return len(l.all)
}
