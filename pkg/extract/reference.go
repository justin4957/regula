package extract

import (
	"regexp"
	"sort"
	"strconv"
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
	TargetChapter     ReferenceTarget = "chapter"
	TargetSection     ReferenceTarget = "section"
	TargetSubsection  ReferenceTarget = "subsection"
	TargetSubchapter  ReferenceTarget = "subchapter"
	TargetDirective   ReferenceTarget = "directive"
	TargetRegulation ReferenceTarget = "regulation"
	TargetTreaty     ReferenceTarget = "treaty"
	TargetDecision   ReferenceTarget = "decision"

	// Parliamentary authority targets
	TargetPrecedent ReferenceTarget = "precedent"
	TargetManual    ReferenceTarget = "manual"
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
	SectionStr   string `json:"section_str,omitempty"` // Full alphanumeric section ID (e.g., "1396a", "300aa-25")

	// For external references
	ExternalDoc string `json:"external_doc,omitempty"`
	DocYear     string `json:"doc_year,omitempty"`
	DocNumber   string `json:"doc_number,omitempty"`

	// Temporal qualifier (optional)
	TemporalKind        string `json:"temporal_kind,omitempty"`        // e.g. "as_amended", "in_force_on", "repealed"
	TemporalDescription string `json:"temporal_description,omitempty"` // full matched text of temporal qualifier
	TemporalDate        string `json:"temporal_date,omitempty"`        // ISO format YYYY-MM-DD when date is present
}

// ReferenceExtractor detects cross-references in regulatory text.
type ReferenceExtractor struct {
	// Internal reference patterns (EU-style)
	articlePattern      *regexp.Regexp
	articleParenPattern *regexp.Regexp
	articlesPattern     *regexp.Regexp
	paragraphPattern    *regexp.Regexp
	pointPattern        *regexp.Regexp
	pointsRangePattern  *regexp.Regexp
	chapterPattern      *regexp.Regexp
	sectionPattern      *regexp.Regexp

	// Internal reference patterns (US-style)
	usSectionPattern          *regexp.Regexp // Section 1798.100
	usSectionSubdivPattern    *regexp.Regexp // Section 1798.100(a)
	usSubdivOfSectionPattern  *regexp.Regexp // subdivision (a) of Section 1798.100
	usParagraphSubdivPattern  *regexp.Regexp // paragraph (1) of subdivision (a) of Section 1798.100
	usSectionsRangePattern    *regexp.Regexp // Sections 1798.100 to 1798.199

	// Internal reference patterns (USC-style)
	uscSectionOfTitlePattern      *regexp.Regexp // section 1396a of this title
	uscSectionOfOtherTitlePattern *regexp.Regexp // section 552a of title 5
	uscSectionSubsecPattern       *regexp.Regexp // section 1396a(a)(10)
	uscSectionBarePattern         *regexp.Regexp // section 1396a (bare, letter suffix required)
	uscSubsectionPattern          *regexp.Regexp // subsection (a) or subsection (b)(1)
	uscParagraphOfSubsecPattern   *regexp.Regexp // paragraph (2) of subsection (a)
	uscSubchapterPattern          *regexp.Regexp // subchapter II of chapter 7
	uscChapterArabicPattern       *regexp.Regexp // chapter 7 (Arabic numerals)
	uscSectionOfActPattern        *regexp.Regexp // section 306 of the Public Health Service Act

	// External reference patterns (EU-style)
	directivePattern    *regexp.Regexp
	regulationPattern   *regexp.Regexp
	regulationNoPattern *regexp.Regexp
	treatyPattern       *regexp.Regexp
	decisionPattern     *regexp.Regexp

	// External reference patterns (US-style)
	usCodePattern      *regexp.Regexp // 15 U.S.C. Section 1681
	cfrPattern         *regexp.Regexp // 45 C.F.R. Part 164
	caTitlePattern     *regexp.Regexp // Section 17014 of Title 18
	publicLawPattern   *regexp.Regexp // Public Law 104-191

	// Internal reference patterns (House Rules-style)
	houseClauseOfRulePattern *regexp.Regexp // clause 5 of rule XX
	houseRuleRefPattern      *regexp.Regexp // rule XX
	houseClauseRefPattern    *regexp.Regexp // clause 5

	// External reference patterns (Parliamentary authorities)
	jeffersonsManualPattern      *regexp.Regexp // Jefferson's Manual, section 53
	jeffersonsManualShortPattern *regexp.Regexp // Jefferson's Manual (standalone)
	cannonsPattern               *regexp.Regexp // Cannon's Precedents, vol. 8, sec. 3449
	cannonsCitePattern           *regexp.Regexp // 8 Cannon § 3449
	deschlerPattern              *regexp.Regexp // Deschler's Precedents, ch. 21, § 18
	deschlerBrownPattern         *regexp.Regexp // Deschler-Brown Precedent ch. 29
	precedentsOfHousePattern     *regexp.Regexp // Precedents of the House
	hindsPattern                 *regexp.Regexp // Hinds' Precedents (older)

	// Temporal reference patterns
	asAmendedByPattern         *regexp.Regexp // as amended by {document}
	asAmendedPattern           *regexp.Regexp // as amended / as amended accordingly
	asInForceOnPattern         *regexp.Regexp // as in force on {date}
	inForceOnPattern           *regexp.Regexp // in force on/from {date}
	enterIntoForcePattern      *regexp.Regexp // enter(s/ed) into force (on {date})?
	asOriginallyEnactedPattern *regexp.Regexp // as originally enacted
	asItStoodOnPattern         *regexp.Regexp // as it stood on {date}
	consolidatedVersionPattern *regexp.Regexp // consolidated version (of)?
	repealedByPattern          *regexp.Regexp // repealed by {document}
	repealedWithEffectPattern  *regexp.Regexp // repealed with effect from {date}
}

// NewReferenceExtractor creates a new ReferenceExtractor with default patterns.
func NewReferenceExtractor() *ReferenceExtractor {
	return &ReferenceExtractor{
		// Internal references (EU-style)
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
		// "Section 1" or "Section 2" (EU-style, simple section numbers)
		// Note: We handle overlap with US-style in extractSectionRefs
		sectionPattern: regexp.MustCompile(`Section\s+(\d+)`),

		// Internal references (US-style California Civil Code)
		// "Section 1798.100" or "Section 1798.185" (simple, no subdivision)
		// Note: We handle overlap with subdivision pattern in extractUSSectionRefs
		usSectionPattern: regexp.MustCompile(`Section\s+(\d+)\.(\d+)`),
		// "Section 1798.100(a)" or "Section 1798.185(a)(1)"
		usSectionSubdivPattern: regexp.MustCompile(`Section\s+(\d+)\.(\d+)\(([a-z])\)(?:\((\d+)\))?`),
		// "subdivision (a) of Section 1798.100"
		usSubdivOfSectionPattern: regexp.MustCompile(`subdivision\s+\(([a-z])\)\s+of\s+Section\s+(\d+)\.(\d+)`),
		// "paragraph (1) of subdivision (a) of Section 1798.185"
		usParagraphSubdivPattern: regexp.MustCompile(`paragraph\s+\((\d+)\)\s+of\s+subdivision\s+\(([a-z])\)\s+of\s+Section\s+(\d+)\.(\d+)`),
		// "Sections 1798.100 to 1798.199" or "Sections 1798.100 through 1798.199"
		usSectionsRangePattern: regexp.MustCompile(`Sections\s+(\d+)\.(\d+)\s+(?:to|through)\s+(\d+)\.(\d+)`),

		// Internal references (USC-style)
		// "section 1396a of this title" or "section 300aa-25(a)(10) of this title"
		uscSectionOfTitlePattern: regexp.MustCompile(`(?i)section\s+(\d+[a-z]*(?:-\d+[a-z]*)?)\s*(\([^)]*\)(?:\(\d+\))?)?\s+of\s+this\s+title`),
		// "section 552a of title 5"
		uscSectionOfOtherTitlePattern: regexp.MustCompile(`(?i)section\s+(\d+[a-z]*(?:-\d+[a-z]*)?)\s*(\([^)]*\)(?:\(\d+\))?)?\s+of\s+title\s+(\d+)`),
		// "section 1396a(a)" or "section 300aa-25(a)(10)" (with parentheticals, no "of" context)
		uscSectionSubsecPattern: regexp.MustCompile(`(?i)\bsection\s+(\d+[a-z]*(?:-\d+[a-z]*)?)\(([a-z])\)(?:\((\d+)\))?`),
		// "section 1396a" or "section 300aa-25" (bare section with letter suffix, avoids matching "Section 1")
		uscSectionBarePattern: regexp.MustCompile(`(?i)\bsection\s+(\d+[a-z]+(?:-\d+[a-z]*)?)\b`),
		// "subsection (a)" or "subsection (b)(1)"
		uscSubsectionPattern: regexp.MustCompile(`(?i)subsection\s+\(([a-z])\)(?:\((\d+)\))?`),
		// "paragraph (2) of subsection (a)"
		uscParagraphOfSubsecPattern: regexp.MustCompile(`(?i)paragraph\s+\((\d+)\)\s+of\s+subsection\s+\(([a-z])\)`),
		// "subchapter II of chapter 7"
		uscSubchapterPattern: regexp.MustCompile(`(?i)subchapter\s+([IVXivx]+)\s+of\s+chapter\s+(\d+)`),
		// "chapter 7" (Arabic numerals, not Roman — avoids overlap with EU chapterPattern)
		uscChapterArabicPattern: regexp.MustCompile(`(?i)\bchapter\s+(\d+)\b`),
		// "section 306 of the Public Health Service Act"
		uscSectionOfActPattern: regexp.MustCompile(`(?i)section\s+(\d+[a-z]*(?:-\d+[a-z]*)?)\s*(\([^)]*\)(?:\(\d+\))?)?\s+of\s+the\s+([A-Z][^,;.]+?)\s+Act`),

		// Internal references (House Rules-style)
		// "clause 5 of rule XX" or "clause 1(a)(1) of rule X"
		houseClauseOfRulePattern: regexp.MustCompile(`(?i)clause\s+(\d+)(?:\(([a-z])\))?(?:\((\d+)\))?\s+of\s+rule\s+([IVXLCDM]+)`),
		// "rule XX" (standalone rule reference)
		houseRuleRefPattern: regexp.MustCompile(`(?i)\brule\s+([IVXLCDM]+)\b`),
		// "clause 5" (standalone clause reference)
		houseClauseRefPattern: regexp.MustCompile(`(?i)\bclause\s+(\d+)\b`),

		// External references (Parliamentary authorities)
		// "Jefferson's Manual" with optional section: "Jefferson's Manual, sec. 53" or "section 53 of Jefferson's Manual"
		jeffersonsManualPattern: regexp.MustCompile(`(?i)(?:(?:sec(?:tion)?\.?\s*(\d+)\s+of\s+)?Jefferson'?s\s+Manual(?:,?\s+sec(?:tion)?\.?\s*(\d+))?)`),
		// "Jefferson's Manual" standalone (for general references)
		jeffersonsManualShortPattern: regexp.MustCompile(`(?i)Jefferson'?s\s+Manual`),
		// "Cannon's Precedents" with volume and section: "Cannon's Precedents, vol. 8, sec. 3449" or "8 Cannon's Precedents § 3449"
		cannonsPattern: regexp.MustCompile(`(?i)(?:(\d+)\s+)?Cannon'?s\s+Precedents(?:,?\s*(?:vol(?:ume)?\.?\s*(\d+)))?(?:,?\s*(?:sec(?:tion)?\.?|§)\s*(\d+))?`),
		// Short citation: "8 Cannon § 3449"
		cannonsCitePattern: regexp.MustCompile(`(?i)(\d+)\s+Cannon\s+(?:§|sec\.?)\s*(\d+)`),
		// "Deschler's Precedents" with chapter and section: "Deschler's Precedents, ch. 21, § 18"
		deschlerPattern: regexp.MustCompile(`(?i)Deschler'?s\s+Precedents(?:,?\s*ch(?:apter)?\.?\s*(\d+))?(?:,?\s*(?:§|sec(?:tion)?\.?)\s*(\d+))?`),
		// "Deschler-Brown Precedent(s)" with optional chapter
		deschlerBrownPattern: regexp.MustCompile(`(?i)Deschler-Brown\s+Precedents?(?:,?\s*ch(?:apter)?\.?\s*(\d+))?`),
		// "Precedents of the House" (generic reference to House precedents)
		precedentsOfHousePattern: regexp.MustCompile(`(?i)Precedents\s+of\s+the\s+House`),
		// "Hinds' Precedents" (older 5-volume set)
		hindsPattern: regexp.MustCompile(`(?i)(?:(\d+)\s+)?Hinds'?\s+Precedents(?:,?\s*(?:§|sec\.?)\s*(\d+))?`),

		// External references (EU-style)
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

		// External references (US-style)
		// "15 U.S.C. Section 1681" or "15 U.S.C. § 1681" or "15 U.S.C. Sec. 1681" or "42 U.S.C. Sec. 1320d"
		usCodePattern: regexp.MustCompile(`(\d+)\s+U\.?S\.?C\.?\s+(?:Section|Sec\.?|§)\s*(\d+[a-z]?)`),
		// "45 C.F.R. Part 164" or "45 CFR 164"
		cfrPattern: regexp.MustCompile(`(\d+)\s+C\.?F\.?R\.?\s+(?:Part\s+)?(\d+)`),
		// "Section 17014 of Title 18" (California codes)
		caTitlePattern: regexp.MustCompile(`Section\s+(\d+)\s+of\s+Title\s+(\d+)`),
		// "Public Law 104-191"
		publicLawPattern: regexp.MustCompile(`Public\s+Law\s+(\d+)-(\d+)`),

		// Temporal reference patterns
		// "as amended by Regulation (EU) 2018/1725" or "as amended by this Regulation"
		asAmendedByPattern: regexp.MustCompile(`(?i)as\s+amended\s+by\s+(.+?)(?:\.|,|;|$)`),
		// "as amended" or "as amended accordingly" (standalone, no "by")
		asAmendedPattern: regexp.MustCompile(`(?i)(?:,\s*)?as\s+amended(?:\s+accordingly)?(?:\s|,|\.|;|$)`),
		// "as in force on 24 May 2016"
		asInForceOnPattern: regexp.MustCompile(`(?i)as\s+in\s+force\s+on\s+(\d{1,2}\s+\w+\s+\d{4})`),
		// "in force on 25 May 2018" or "in force from 25 May 2018"
		inForceOnPattern: regexp.MustCompile(`(?i)in\s+force\s+(?:on|from)\s+(\d{1,2}\s+\w+\s+\d{4})`),
		// "enter into force" or "enters into force on 25 May 2018" or "entered into force"
		enterIntoForcePattern: regexp.MustCompile(`(?i)enter(?:s|ed)?\s+into\s+force(?:\s+on\s+(\d{1,2}\s+\w+\s+\d{4}))?`),
		// "as originally enacted"
		asOriginallyEnactedPattern: regexp.MustCompile(`(?i)as\s+originally\s+enacted`),
		// "as it stood on 1 January 2020"
		asItStoodOnPattern: regexp.MustCompile(`(?i)as\s+it\s+stood\s+on\s+(\d{1,2}\s+\w+\s+\d{4})`),
		// "consolidated version" or "consolidated version of"
		consolidatedVersionPattern: regexp.MustCompile(`(?i)consolidated\s+version(?:\s+of)?`),
		// "repealed by this Regulation" or "repealed by Regulation (EU) 2016/679"
		repealedByPattern: regexp.MustCompile(`(?i)repealed\s+by\s+(.+?)(?:\.|,|;|$)`),
		// "repealed with effect from 25 May 2018"
		repealedWithEffectPattern: regexp.MustCompile(`(?i)repealed\s+with\s+effect\s+from\s+(\d{1,2}\s+\w+\s+\d{4})`),
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

	// Extract internal references (EU-style)
	refs = append(refs, e.extractArticleRefs(text, article.Number)...)
	refs = append(refs, e.extractParagraphRefs(text, article.Number)...)
	refs = append(refs, e.extractPointRefs(text, article.Number)...)
	refs = append(refs, e.extractChapterRefs(text, article.Number)...)
	refs = append(refs, e.extractSectionRefs(text, article.Number)...)

	// Extract internal references (US-style California Civil Code)
	refs = append(refs, e.extractUSSectionRefs(text, article.Number)...)

	// Extract internal references (USC-style)
	refs = append(refs, e.extractUSCSectionRefs(text, article.Number, refs)...)

	// Extract external references (EU-style)
	refs = append(refs, e.extractDirectiveRefs(text, article.Number)...)
	refs = append(refs, e.extractRegulationRefs(text, article.Number)...)
	refs = append(refs, e.extractTreatyRefs(text, article.Number)...)
	refs = append(refs, e.extractDecisionRefs(text, article.Number)...)

	// Extract internal references (House Rules-style)
	refs = append(refs, e.extractHouseRuleRefs(text, article.Number)...)

	// Extract external references (US-style)
	refs = append(refs, e.extractUSExternalRefs(text, article.Number)...)

	// Extract external references (Parliamentary authorities)
	refs = append(refs, e.extractParliamentaryAuthorityRefs(text, article.Number)...)

	// Extract temporal references
	refs = append(refs, e.extractTemporalRefs(text, article.Number)...)

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

// extractSectionRefs extracts section references (EU-style simple numbers).
func (e *ReferenceExtractor) extractSectionRefs(text string, sourceArticle int) []*Reference {
	var refs []*Reference

	matches := e.sectionPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		// Skip if this is a US-style section (followed by a decimal point)
		endPos := match[1]
		if endPos < len(text) && text[endPos] == '.' {
			continue
		}

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

// extractUSSectionRefs extracts US-style California Civil Code section references.
func (e *ReferenceExtractor) extractUSSectionRefs(text string, sourceArticle int) []*Reference {
	var refs []*Reference

	// Paragraph of subdivision of section: "paragraph (1) of subdivision (a) of Section 1798.185"
	matches := e.usParagraphSubdivPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		rawText := text[match[0]:match[1]]
		paragraphNum := text[match[2]:match[3]]
		subdivLetter := text[match[4]:match[5]]
		codePrefix := mustAtoi(text[match[6]:match[7]])
		sectionNum := mustAtoi(text[match[8]:match[9]])

		// For California Civil Code 1798.xxx, map to Article xxx
		articleNum := sectionNum
		if codePrefix == 1798 {
			articleNum = sectionNum
		}

		refs = append(refs, &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetSection,
			RawText:       rawText,
			Identifier:    buildUSSectionIdentifier(codePrefix, sectionNum, subdivLetter, paragraphNum),
			SubRef:        "paragraph",
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ArticleNum:    articleNum,
			ParagraphNum:  mustAtoi(paragraphNum),
			PointLetter:   subdivLetter,
			SectionNum:    codePrefix*1000 + sectionNum,
		})
	}

	// Subdivision of section: "subdivision (a) of Section 1798.100"
	matches = e.usSubdivOfSectionPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]
		subdivLetter := text[match[2]:match[3]]
		codePrefix := mustAtoi(text[match[4]:match[5]])
		sectionNum := mustAtoi(text[match[6]:match[7]])

		articleNum := sectionNum
		if codePrefix == 1798 {
			articleNum = sectionNum
		}

		refs = append(refs, &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetSection,
			RawText:       rawText,
			Identifier:    buildUSSectionIdentifier(codePrefix, sectionNum, subdivLetter, ""),
			SubRef:        "subdivision",
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ArticleNum:    articleNum,
			PointLetter:   subdivLetter,
			SectionNum:    codePrefix*1000 + sectionNum,
		})
	}

	// Section with subdivision: "Section 1798.100(a)" or "Section 1798.185(a)(1)"
	matches = e.usSectionSubdivPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]
		codePrefix := mustAtoi(text[match[2]:match[3]])
		sectionNum := mustAtoi(text[match[4]:match[5]])
		subdivLetter := text[match[6]:match[7]]

		var paragraphNum string
		if match[8] != -1 {
			paragraphNum = text[match[8]:match[9]]
		}

		articleNum := sectionNum
		if codePrefix == 1798 {
			articleNum = sectionNum
		}

		ref := &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetSection,
			RawText:       rawText,
			Identifier:    buildUSSectionIdentifier(codePrefix, sectionNum, subdivLetter, paragraphNum),
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ArticleNum:    articleNum,
			PointLetter:   subdivLetter,
			SectionNum:    codePrefix*1000 + sectionNum,
		}

		if paragraphNum != "" {
			ref.ParagraphNum = mustAtoi(paragraphNum)
			ref.SubRef = "paragraph"
		} else {
			ref.SubRef = "subdivision"
		}

		refs = append(refs, ref)
	}

	// Sections range: "Sections 1798.100 to 1798.199"
	matches = e.usSectionsRangePattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		rawText := text[match[0]:match[1]]
		startPrefix := mustAtoi(text[match[2]:match[3]])
		startSection := mustAtoi(text[match[4]:match[5]])
		endPrefix := mustAtoi(text[match[6]:match[7]])
		endSection := mustAtoi(text[match[8]:match[9]])

		refs = append(refs, &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetSection,
			RawText:       rawText,
			Identifier:    buildUSSectionsRangeIdentifier(startPrefix, startSection, endPrefix, endSection),
			SubRef:        "range",
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ArticleNum:    startSection,
			SectionNum:    startPrefix*1000 + startSection,
		})
	}

	// Simple section: "Section 1798.100"
	matches = e.usSectionPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]
		codePrefix := mustAtoi(text[match[2]:match[3]])
		sectionNum := mustAtoi(text[match[4]:match[5]])

		articleNum := sectionNum
		if codePrefix == 1798 {
			articleNum = sectionNum
		}

		refs = append(refs, &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetSection,
			RawText:       rawText,
			Identifier:    buildUSSectionIdentifier(codePrefix, sectionNum, "", ""),
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ArticleNum:    articleNum,
			SectionNum:    codePrefix*1000 + sectionNum,
		})
	}

	return refs
}

// extractHouseRuleRefs extracts House Rules-style internal references:
// "clause N of rule X", "rule X", "clause N".
func (e *ReferenceExtractor) extractHouseRuleRefs(text string, sourceArticle int) []*Reference {
	var refs []*Reference

	// "clause 5 of rule XX" or "clause 1(a)(1) of rule X"
	clauseOfRuleMatches := e.houseClauseOfRulePattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range clauseOfRuleMatches {
		rawText := text[match[0]:match[1]]
		clauseNum := text[match[2]:match[3]]
		ruleNum := text[match[8]:match[9]]
		clauseNumInt, _ := strconv.Atoi(clauseNum)

		ref := &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetArticle,
			RawText:       rawText,
			Identifier:    "Rule " + ruleNum + " clause " + clauseNum,
			SourceArticle: sourceArticle,
			ArticleNum:    clauseNumInt,
			ChapterNum:    ruleNum,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
		}
		refs = append(refs, ref)
	}

	// "rule XX" (standalone, but skip if already part of "clause N of rule X")
	ruleMatches := e.houseRuleRefPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range ruleMatches {
		// Skip if this match is part of a "clause of rule" match
		alreadyCovered := false
		for _, corMatch := range clauseOfRuleMatches {
			if match[0] >= corMatch[0] && match[1] <= corMatch[1] {
				alreadyCovered = true
				break
			}
		}
		if alreadyCovered {
			continue
		}

		rawText := text[match[0]:match[1]]
		ruleNum := text[match[2]:match[3]]
		ref := &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetChapter,
			RawText:       rawText,
			Identifier:    "Rule " + ruleNum,
			SourceArticle: sourceArticle,
			ChapterNum:    ruleNum,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
		}
		refs = append(refs, ref)
	}

	return refs
}

// extractUSExternalRefs extracts US-style external references (USC, CFR, etc.).
func (e *ReferenceExtractor) extractUSExternalRefs(text string, sourceArticle int) []*Reference {
	var refs []*Reference

	// US Code: "15 U.S.C. Section 1681"
	matches := e.usCodePattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		rawText := text[match[0]:match[1]]
		title := text[match[2]:match[3]]
		section := text[match[4]:match[5]]

		refs = append(refs, &Reference{
			Type:          ReferenceTypeExternal,
			Target:        TargetRegulation,
			RawText:       rawText,
			Identifier:    title + " U.S.C. § " + section,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ExternalDoc:   "USC",
			DocNumber:     title,
			SectionNum:    mustAtoi(section),
		})
	}

	// CFR: "45 C.F.R. Part 164"
	matches = e.cfrPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		rawText := text[match[0]:match[1]]
		title := text[match[2]:match[3]]
		part := text[match[4]:match[5]]

		refs = append(refs, &Reference{
			Type:          ReferenceTypeExternal,
			Target:        TargetRegulation,
			RawText:       rawText,
			Identifier:    title + " C.F.R. Part " + part,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ExternalDoc:   "CFR",
			DocNumber:     title,
			SectionNum:    mustAtoi(part),
		})
	}

	// California Title references: "Section 17014 of Title 18"
	matches = e.caTitlePattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		rawText := text[match[0]:match[1]]
		section := text[match[2]:match[3]]
		title := text[match[4]:match[5]]

		refs = append(refs, &Reference{
			Type:          ReferenceTypeExternal,
			Target:        TargetSection,
			RawText:       rawText,
			Identifier:    "Cal. Title " + title + " § " + section,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ExternalDoc:   "CalTitle",
			DocNumber:     title,
			SectionNum:    mustAtoi(section),
		})
	}

	// Public Law: "Public Law 104-191"
	matches = e.publicLawPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		rawText := text[match[0]:match[1]]
		congress := text[match[2]:match[3]]
		lawNum := text[match[4]:match[5]]

		refs = append(refs, &Reference{
			Type:          ReferenceTypeExternal,
			Target:        TargetRegulation,
			RawText:       rawText,
			Identifier:    "Pub. L. " + congress + "-" + lawNum,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ExternalDoc:   "PublicLaw",
			DocYear:       congress,
			DocNumber:     lawNum,
		})
	}

	return refs
}

// extractParliamentaryAuthorityRefs extracts references to parliamentary authorities:
// Jefferson's Manual, Cannon's Precedents, Deschler's Precedents, Deschler-Brown Precedents,
// Hinds' Precedents, and general House precedents.
func (e *ReferenceExtractor) extractParliamentaryAuthorityRefs(text string, sourceArticle int) []*Reference {
	var refs []*Reference

	// Jefferson's Manual with section: "Jefferson's Manual, sec. 53" or "section 53 of Jefferson's Manual"
	matches := e.jeffersonsManualPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		rawText := text[match[0]:match[1]]

		// Extract section number from either capture group
		var sectionNum string
		if match[2] != -1 && match[3] != -1 {
			sectionNum = text[match[2]:match[3]] // "section N of Jefferson's Manual"
		} else if match[4] != -1 && match[5] != -1 {
			sectionNum = text[match[4]:match[5]] // "Jefferson's Manual, sec. N"
		}

		identifier := "Jefferson's Manual"
		if sectionNum != "" {
			identifier += " § " + sectionNum
		}

		refs = append(refs, &Reference{
			Type:          ReferenceTypeExternal,
			Target:        TargetManual,
			RawText:       rawText,
			Identifier:    identifier,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ExternalDoc:   "JeffersonsManual",
			SectionNum:    mustAtoi(sectionNum),
		})
	}

	// Cannon's Precedents: "Cannon's Precedents, vol. 8, sec. 3449" or "8 Cannon's Precedents § 3449"
	matches = e.cannonsPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]
		var volumeNum, sectionNum string

		// Volume from prefix or vol. X
		if match[2] != -1 && match[3] != -1 {
			volumeNum = text[match[2]:match[3]]
		} else if match[4] != -1 && match[5] != -1 {
			volumeNum = text[match[4]:match[5]]
		}

		// Section number
		if match[6] != -1 && match[7] != -1 {
			sectionNum = text[match[6]:match[7]]
		}

		identifier := "Cannon's Precedents"
		if volumeNum != "" {
			identifier += " vol. " + volumeNum
		}
		if sectionNum != "" {
			identifier += " § " + sectionNum
		}

		refs = append(refs, &Reference{
			Type:          ReferenceTypeExternal,
			Target:        TargetPrecedent,
			RawText:       rawText,
			Identifier:    identifier,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ExternalDoc:   "CannonsPrecedents",
			DocNumber:     volumeNum,
			SectionNum:    mustAtoi(sectionNum),
		})
	}

	// Short Cannon citation: "8 Cannon § 3449"
	matches = e.cannonsCitePattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]
		volumeNum := text[match[2]:match[3]]
		sectionNum := text[match[4]:match[5]]

		identifier := "Cannon's Precedents vol. " + volumeNum + " § " + sectionNum

		refs = append(refs, &Reference{
			Type:          ReferenceTypeExternal,
			Target:        TargetPrecedent,
			RawText:       rawText,
			Identifier:    identifier,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ExternalDoc:   "CannonsPrecedents",
			DocNumber:     volumeNum,
			SectionNum:    mustAtoi(sectionNum),
		})
	}

	// Deschler's Precedents: "Deschler's Precedents, ch. 21, § 18"
	matches = e.deschlerPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]
		var chapterNum, sectionNum string

		if match[2] != -1 && match[3] != -1 {
			chapterNum = text[match[2]:match[3]]
		}
		if match[4] != -1 && match[5] != -1 {
			sectionNum = text[match[4]:match[5]]
		}

		identifier := "Deschler's Precedents"
		if chapterNum != "" {
			identifier += " ch. " + chapterNum
		}
		if sectionNum != "" {
			identifier += " § " + sectionNum
		}

		refs = append(refs, &Reference{
			Type:          ReferenceTypeExternal,
			Target:        TargetPrecedent,
			RawText:       rawText,
			Identifier:    identifier,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ExternalDoc:   "DeschlersPrecedents",
			ChapterNum:    chapterNum,
			SectionNum:    mustAtoi(sectionNum),
		})
	}

	// Deschler-Brown Precedents: "Deschler-Brown Precedent ch. 29"
	matches = e.deschlerBrownPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]
		var chapterNum string

		if match[2] != -1 && match[3] != -1 {
			chapterNum = text[match[2]:match[3]]
		}

		identifier := "Deschler-Brown Precedents"
		if chapterNum != "" {
			identifier += " ch. " + chapterNum
		}

		refs = append(refs, &Reference{
			Type:          ReferenceTypeExternal,
			Target:        TargetPrecedent,
			RawText:       rawText,
			Identifier:    identifier,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ExternalDoc:   "DeschlerBrownPrecedents",
			ChapterNum:    chapterNum,
		})
	}

	// Hinds' Precedents: "5 Hinds' Precedents § 5445" or "Hinds' Precedents"
	matches = e.hindsPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]
		var volumeNum, sectionNum string

		if match[2] != -1 && match[3] != -1 {
			volumeNum = text[match[2]:match[3]]
		}
		if match[4] != -1 && match[5] != -1 {
			sectionNum = text[match[4]:match[5]]
		}

		identifier := "Hinds' Precedents"
		if volumeNum != "" {
			identifier += " vol. " + volumeNum
		}
		if sectionNum != "" {
			identifier += " § " + sectionNum
		}

		refs = append(refs, &Reference{
			Type:          ReferenceTypeExternal,
			Target:        TargetPrecedent,
			RawText:       rawText,
			Identifier:    identifier,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ExternalDoc:   "HindsPrecedents",
			DocNumber:     volumeNum,
			SectionNum:    mustAtoi(sectionNum),
		})
	}

	// Generic "Precedents of the House"
	matches = e.precedentsOfHousePattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]

		refs = append(refs, &Reference{
			Type:          ReferenceTypeExternal,
			Target:        TargetPrecedent,
			RawText:       rawText,
			Identifier:    "Precedents of the House",
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ExternalDoc:   "HousePrecedents",
		})
	}

	return refs
}

// buildUSSectionIdentifier creates a standardized US section identifier.
func buildUSSectionIdentifier(codePrefix, sectionNum int, subdivLetter, paragraphNum string) string {
	id := "Section " + itoa(codePrefix) + "." + itoa(sectionNum)
	if subdivLetter != "" {
		id += "(" + subdivLetter + ")"
	}
	if paragraphNum != "" {
		id += "(" + paragraphNum + ")"
	}
	return id
}

// buildUSSectionsRangeIdentifier creates an identifier for US section ranges.
func buildUSSectionsRangeIdentifier(startPrefix, startSection, endPrefix, endSection int) string {
	return "Sections " + itoa(startPrefix) + "." + itoa(startSection) + "-" + itoa(endPrefix) + "." + itoa(endSection)
}

// extractUSCSectionRefs extracts USC-style internal cross-references.
// USC uses lowercase "section", no dot separators, letter suffixes (1396a), dash extensions (1320d-1),
// and context phrases like "of this title" (internal) or "of title 5" (cross-title external).
func (e *ReferenceExtractor) extractUSCSectionRefs(text string, sourceArticle int, existingRefs []*Reference) []*Reference {
	var refs []*Reference

	// 1. Cross-title: "section 552a of title 5" (external)
	matches := e.uscSectionOfOtherTitlePattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if isOverlappingWithSlice(match[0], match[1], existingRefs) || e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]
		sectionStr := text[match[2]:match[3]]
		var subsecStr string
		if match[4] != -1 {
			subsecStr = text[match[4]:match[5]]
		}
		titleNum := text[match[6]:match[7]]

		refs = append(refs, &Reference{
			Type:          ReferenceTypeExternal,
			Target:        TargetSection,
			RawText:       rawText,
			Identifier:    titleNum + " U.S.C. § " + sectionStr + subsecStr,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ExternalDoc:   "USC",
			DocNumber:     titleNum,
			SectionNum:    parseUSCSectionNum(sectionStr),
			SectionStr:    sectionStr,
		})
	}

	// 2. Same-title: "section 1396a of this title" (internal)
	matches = e.uscSectionOfTitlePattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if isOverlappingWithSlice(match[0], match[1], existingRefs) || e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]
		sectionStr := text[match[2]:match[3]]
		var subsecStr string
		if match[4] != -1 {
			subsecStr = text[match[4]:match[5]]
		}

		sectionNum := parseUSCSectionNum(sectionStr)
		identifier := buildUSCSectionIdentifier(sectionStr, subsecStr, "")

		refs = append(refs, &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetSection,
			RawText:       rawText,
			Identifier:    identifier,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ArticleNum:    sectionNum,
			SectionNum:    sectionNum,
			SectionStr:    sectionStr,
		})
	}

	// 3. Section of Act: "section 306 of the Public Health Service Act" (external)
	matches = e.uscSectionOfActPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if isOverlappingWithSlice(match[0], match[1], existingRefs) || e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]
		sectionStr := text[match[2]:match[3]]
		var subsecStr string
		if match[4] != -1 {
			subsecStr = text[match[4]:match[5]]
		}
		actName := strings.TrimSpace(text[match[6]:match[7]])

		refs = append(refs, &Reference{
			Type:          ReferenceTypeExternal,
			Target:        TargetSection,
			RawText:       rawText,
			Identifier:    sectionStr + subsecStr + " of " + actName + " Act",
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ExternalDoc:   "USAct",
			DocNumber:     actName + " Act",
			SectionNum:    parseUSCSectionNum(sectionStr),
			SectionStr:    sectionStr,
		})
	}

	// 4. Paragraph of subsection: "paragraph (2) of subsection (a)"
	matches = e.uscParagraphOfSubsecPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if isOverlappingWithSlice(match[0], match[1], existingRefs) || e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]
		paragraphNum := text[match[2]:match[3]]
		subsecLetter := text[match[4]:match[5]]

		refs = append(refs, &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetSubsection,
			RawText:       rawText,
			Identifier:    "subsection (" + subsecLetter + ")(" + paragraphNum + ")",
			SubRef:        "paragraph",
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ArticleNum:    sourceArticle,
			ParagraphNum:  mustAtoi(paragraphNum),
			PointLetter:   subsecLetter,
		})
	}

	// 5. Section with subsection: "section 1396a(a)" or "section 1396a(a)(10)"
	matches = e.uscSectionSubsecPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if isOverlappingWithSlice(match[0], match[1], existingRefs) || e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]
		sectionStr := text[match[2]:match[3]]
		subsecLetter := text[match[4]:match[5]]
		var paragraphNum string
		if match[6] != -1 {
			paragraphNum = text[match[6]:match[7]]
		}

		sectionNum := parseUSCSectionNum(sectionStr)
		identifier := buildUSCSectionIdentifier(sectionStr, "("+subsecLetter+")", paragraphNum)

		ref := &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetSection,
			RawText:       rawText,
			Identifier:    identifier,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ArticleNum:    sectionNum,
			SectionNum:    sectionNum,
			SectionStr:    sectionStr,
			PointLetter:   subsecLetter,
		}
		if paragraphNum != "" {
			ref.ParagraphNum = mustAtoi(paragraphNum)
			ref.SubRef = "paragraph"
		} else {
			ref.SubRef = "subsection"
		}
		refs = append(refs, ref)
	}

	// 6. Subsection: "subsection (a)" or "subsection (b)(1)"
	matches = e.uscSubsectionPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if isOverlappingWithSlice(match[0], match[1], existingRefs) || e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]
		subsecLetter := text[match[2]:match[3]]
		var paragraphNum string
		if match[4] != -1 {
			paragraphNum = text[match[4]:match[5]]
		}

		identifier := "subsection (" + subsecLetter + ")"
		if paragraphNum != "" {
			identifier += "(" + paragraphNum + ")"
		}

		refs = append(refs, &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetSubsection,
			RawText:       rawText,
			Identifier:    identifier,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ArticleNum:    sourceArticle,
			PointLetter:   subsecLetter,
			ParagraphNum:  mustAtoi(paragraphNum),
		})
	}

	// 7. Subchapter: "subchapter II of chapter 7"
	matches = e.uscSubchapterPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if isOverlappingWithSlice(match[0], match[1], existingRefs) || e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]
		subchapterNum := strings.ToUpper(text[match[2]:match[3]])
		chapterNum := text[match[4]:match[5]]

		refs = append(refs, &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetSubchapter,
			RawText:       rawText,
			Identifier:    "subchapter " + subchapterNum + " of chapter " + chapterNum,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ChapterNum:    chapterNum,
		})
	}

	// 8. Chapter (Arabic numerals): "chapter 7"
	matches = e.uscChapterArabicPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if isOverlappingWithSlice(match[0], match[1], existingRefs) || e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]
		chapterNum := text[match[2]:match[3]]

		refs = append(refs, &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetChapter,
			RawText:       rawText,
			Identifier:    "chapter " + chapterNum,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ChapterNum:    chapterNum,
		})
	}

	// 9. Bare section with letter suffix: "section 1396a" or "section 1320d-1"
	matches = e.uscSectionBarePattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if isOverlappingWithSlice(match[0], match[1], existingRefs) || e.isOverlapping(match[0], match[1], refs) {
			continue
		}

		rawText := text[match[0]:match[1]]
		// Trim leading whitespace from rawText (word boundary may match start of line)
		rawText = strings.TrimLeft(rawText, " \t")
		sectionStr := text[match[2]:match[3]]
		sectionNum := parseUSCSectionNum(sectionStr)

		refs = append(refs, &Reference{
			Type:          ReferenceTypeInternal,
			Target:        TargetSection,
			RawText:       rawText,
			Identifier:    "Section " + sectionStr,
			SourceArticle: sourceArticle,
			TextOffset:    match[0],
			TextLength:    match[1] - match[0],
			ArticleNum:    sectionNum,
			SectionNum:    sectionNum,
			SectionStr:    sectionStr,
		})
	}

	return refs
}

// buildUSCSectionIdentifier creates a standardized USC section identifier.
// sectionStr is the raw section number like "1396a" or "1320d-1".
// subsecStr is the parenthetical like "(a)" or "(a)(10)" (may be empty).
// paragraphNum is a standalone paragraph number (may be empty).
func buildUSCSectionIdentifier(sectionStr, subsecStr, paragraphNum string) string {
	identifier := "Section " + sectionStr
	if subsecStr != "" {
		identifier += subsecStr
	}
	if paragraphNum != "" {
		identifier += "(" + paragraphNum + ")"
	}
	return identifier
}

// parseUSCSectionNum extracts the numeric portion of a USC section number.
// "1396" → 1396, "1396a" → 1396, "1320d-1" → 1320.
func parseUSCSectionNum(sectionStr string) int {
	var digits strings.Builder
	for _, ch := range sectionStr {
		if ch >= '0' && ch <= '9' {
			digits.WriteRune(ch)
		} else {
			break
		}
	}
	return mustAtoi(digits.String())
}

// isOverlappingWithSlice checks if a match region overlaps with any reference in a slice.
func isOverlappingWithSlice(start, end int, refs []*Reference) bool {
	for _, ref := range refs {
		refEnd := ref.TextOffset + ref.TextLength
		if start < refEnd && end > ref.TextOffset {
			return true
		}
	}
	return false
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

// extractTemporalRefs extracts temporal qualifiers from the text.
// These capture patterns like "as amended by", "as in force on", "repealed by", etc.
func (e *ReferenceExtractor) extractTemporalRefs(text string, sourceArticle int) []*Reference {
	var refs []*Reference

	// "repealed with effect from 25 May 2018" (most specific repeal pattern first)
	matches := e.repealedWithEffectPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		rawText := text[match[0]:match[1]]
		dateStr := text[match[2]:match[3]]
		isoDate := parseEuropeanDate(dateStr)

		refs = append(refs, &Reference{
			Type:                ReferenceTypeInternal,
			Target:              TargetArticle,
			RawText:             rawText,
			Identifier:          "temporal:repealed",
			SourceArticle:       sourceArticle,
			TextOffset:          match[0],
			TextLength:          match[1] - match[0],
			TemporalKind:        "repealed",
			TemporalDescription: rawText,
			TemporalDate:        isoDate,
		})
	}

	// "repealed by {document}" (skip if overlapping with repealedWithEffect)
	matches = e.repealedByPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}
		rawText := text[match[0]:match[1]]
		amendingDoc := strings.TrimSpace(text[match[2]:match[3]])

		refs = append(refs, &Reference{
			Type:                ReferenceTypeInternal,
			Target:              TargetArticle,
			RawText:             rawText,
			Identifier:          "temporal:repealed",
			SourceArticle:       sourceArticle,
			TextOffset:          match[0],
			TextLength:          match[1] - match[0],
			TemporalKind:        "repealed",
			TemporalDescription: amendingDoc,
		})
	}

	// "as amended by {document}" (most specific amendment pattern)
	matches = e.asAmendedByPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}
		rawText := text[match[0]:match[1]]
		amendingDoc := strings.TrimSpace(text[match[2]:match[3]])

		refs = append(refs, &Reference{
			Type:                ReferenceTypeInternal,
			Target:              TargetArticle,
			RawText:             rawText,
			Identifier:          "temporal:as_amended",
			SourceArticle:       sourceArticle,
			TextOffset:          match[0],
			TextLength:          match[1] - match[0],
			TemporalKind:        "as_amended",
			TemporalDescription: amendingDoc,
		})
	}

	// "as amended" / "as amended accordingly" (standalone, no "by")
	matches = e.asAmendedPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}
		rawText := strings.TrimSpace(text[match[0]:match[1]])

		refs = append(refs, &Reference{
			Type:                ReferenceTypeInternal,
			Target:              TargetArticle,
			RawText:             rawText,
			Identifier:          "temporal:as_amended",
			SourceArticle:       sourceArticle,
			TextOffset:          match[0],
			TextLength:          match[1] - match[0],
			TemporalKind:        "as_amended",
			TemporalDescription: rawText,
		})
	}

	// "as in force on {date}"
	matches = e.asInForceOnPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}
		rawText := text[match[0]:match[1]]
		dateStr := text[match[2]:match[3]]
		isoDate := parseEuropeanDate(dateStr)

		refs = append(refs, &Reference{
			Type:                ReferenceTypeInternal,
			Target:              TargetArticle,
			RawText:             rawText,
			Identifier:          "temporal:in_force_on",
			SourceArticle:       sourceArticle,
			TextOffset:          match[0],
			TextLength:          match[1] - match[0],
			TemporalKind:        "in_force_on",
			TemporalDescription: rawText,
			TemporalDate:        isoDate,
		})
	}

	// "in force on/from {date}" (skip if overlapping with asInForceOn)
	matches = e.inForceOnPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}
		rawText := text[match[0]:match[1]]
		dateStr := text[match[2]:match[3]]
		isoDate := parseEuropeanDate(dateStr)

		refs = append(refs, &Reference{
			Type:                ReferenceTypeInternal,
			Target:              TargetArticle,
			RawText:             rawText,
			Identifier:          "temporal:in_force_on",
			SourceArticle:       sourceArticle,
			TextOffset:          match[0],
			TextLength:          match[1] - match[0],
			TemporalKind:        "in_force_on",
			TemporalDescription: rawText,
			TemporalDate:        isoDate,
		})
	}

	// "enter(s/ed) into force (on {date})?"
	matches = e.enterIntoForcePattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}
		rawText := text[match[0]:match[1]]
		ref := &Reference{
			Type:                ReferenceTypeInternal,
			Target:              TargetArticle,
			RawText:             rawText,
			Identifier:          "temporal:in_force_on",
			SourceArticle:       sourceArticle,
			TextOffset:          match[0],
			TextLength:          match[1] - match[0],
			TemporalKind:        "in_force_on",
			TemporalDescription: rawText,
		}
		// Optional date group
		if match[2] != -1 && match[3] != -1 {
			dateStr := text[match[2]:match[3]]
			ref.TemporalDate = parseEuropeanDate(dateStr)
		}
		refs = append(refs, ref)
	}

	// "as originally enacted"
	matches = e.asOriginallyEnactedPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}
		rawText := text[match[0]:match[1]]

		refs = append(refs, &Reference{
			Type:                ReferenceTypeInternal,
			Target:              TargetArticle,
			RawText:             rawText,
			Identifier:          "temporal:original",
			SourceArticle:       sourceArticle,
			TextOffset:          match[0],
			TextLength:          match[1] - match[0],
			TemporalKind:        "original",
			TemporalDescription: rawText,
		})
	}

	// "as it stood on {date}"
	matches = e.asItStoodOnPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}
		rawText := text[match[0]:match[1]]
		dateStr := text[match[2]:match[3]]
		isoDate := parseEuropeanDate(dateStr)

		refs = append(refs, &Reference{
			Type:                ReferenceTypeInternal,
			Target:              TargetArticle,
			RawText:             rawText,
			Identifier:          "temporal:original",
			SourceArticle:       sourceArticle,
			TextOffset:          match[0],
			TextLength:          match[1] - match[0],
			TemporalKind:        "original",
			TemporalDescription: rawText,
			TemporalDate:        isoDate,
		})
	}

	// "consolidated version (of)?"
	matches = e.consolidatedVersionPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if e.isOverlapping(match[0], match[1], refs) {
			continue
		}
		rawText := text[match[0]:match[1]]

		refs = append(refs, &Reference{
			Type:                ReferenceTypeInternal,
			Target:              TargetArticle,
			RawText:             rawText,
			Identifier:          "temporal:consolidated",
			SourceArticle:       sourceArticle,
			TextOffset:          match[0],
			TextLength:          match[1] - match[0],
			TemporalKind:        "consolidated",
			TemporalDescription: rawText,
		})
	}

	return refs
}

// parseEuropeanDate parses a European-style date string like "25 May 2018" to ISO format "2018-05-25".
// Returns empty string if parsing fails.
func parseEuropeanDate(dateStr string) string {
	monthNames := map[string]string{
		"january": "01", "february": "02", "march": "03", "april": "04",
		"may": "05", "june": "06", "july": "07", "august": "08",
		"september": "09", "october": "10", "november": "11", "december": "12",
	}

	parts := strings.Fields(strings.TrimSpace(dateStr))
	if len(parts) != 3 {
		return ""
	}

	day := parts[0]
	monthStr := strings.ToLower(parts[1])
	year := parts[2]

	month, ok := monthNames[monthStr]
	if !ok {
		return ""
	}

	// Pad day to 2 digits
	if len(day) == 1 {
		day = "0" + day
	}

	// Validate year is 4 digits
	if len(year) != 4 {
		return ""
	}

	return year + "-" + month + "-" + day
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
