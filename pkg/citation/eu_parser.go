package citation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// EUCitationParser parses EU-style legal citations including:
//   - Regulations: "Regulation (EU) 2016/679", "Regulation (EC) No 45/2001"
//   - Directives: "Directive 95/46/EC", "Directive (EU) 2016/680"
//   - Decisions: "Decision 2010/87/EU"
//   - Treaties: "TFEU", "TEU", "Treaty on the Functioning of the European Union"
//   - Internal references: "Article 6(1)(a)", "Chapter III", "Section 2"
type EUCitationParser struct {
	// External citation patterns (mirrored from pkg/extract/reference.go).
	regulationPattern   *regexp.Regexp
	regulationNoPattern *regexp.Regexp
	directivePattern    *regexp.Regexp
	decisionPattern     *regexp.Regexp
	treatyPattern       *regexp.Regexp

	// Internal reference patterns.
	articleParenPattern  *regexp.Regexp
	articlePattern       *regexp.Regexp
	articlesRangePattern *regexp.Regexp
	chapterPattern       *regexp.Regexp
	sectionPattern       *regexp.Regexp
}

// NewEUCitationParser creates a new EU citation parser with compiled patterns.
func NewEUCitationParser() *EUCitationParser {
	return &EUCitationParser{
		// External: mirrored from reference.go lines 128-136.
		regulationPattern:   regexp.MustCompile(`Regulation\s+\(E[CU]\)\s+(\d{4})/(\d+)`),
		regulationNoPattern: regexp.MustCompile(`Regulation\s+\(E[CU]\)\s+No\s+(\d+)/(\d+)`),
		directivePattern:    regexp.MustCompile(`Directive\s+(?:\(E[CU]\)\s+)?(\d+)/(\d+)(?:/EC|/EU)?`),
		decisionPattern:     regexp.MustCompile(`Decision\s+(\d+)/(\d+)/E[CU]`),
		treatyPattern:       regexp.MustCompile(`(?:Treaty\s+on\s+the\s+Functioning\s+of\s+the\s+European\s+Union|TFEU|TEU)`),

		// Internal: mirrored from reference.go lines 97-111.
		articleParenPattern:  regexp.MustCompile(`Article\s+(\d+)\((\d+)\)(?:\(([a-z])\))?`),
		articlePattern:       regexp.MustCompile(`Article\s+(\d+)`),
		articlesRangePattern: regexp.MustCompile(`Articles\s+(\d+)\s+(?:and|to)\s+(\d+)`),
		chapterPattern:       regexp.MustCompile(`Chapter\s+([IVX]+)`),
		sectionPattern:       regexp.MustCompile(`Section\s+(\d+)`),
	}
}

// Name returns the parser name.
func (p *EUCitationParser) Name() string {
	return "EU Citation Parser"
}

// Jurisdictions returns supported jurisdiction codes.
func (p *EUCitationParser) Jurisdictions() []string {
	return []string{"EU"}
}

// Parse extracts all EU-style citations from the text.
func (p *EUCitationParser) Parse(text string) ([]*Citation, error) {
	var citations []*Citation

	citations = append(citations, p.parseRegulations(text)...)
	citations = append(citations, p.parseDirectives(text)...)
	citations = append(citations, p.parseDecisions(text)...)
	citations = append(citations, p.parseTreaties(text)...)
	citations = append(citations, p.parseArticleReferences(text)...)
	citations = append(citations, p.parseChapterReferences(text)...)
	citations = append(citations, p.parseSectionReferences(text)...)

	return citations, nil
}

// Normalize converts a citation to its canonical EU form.
func (p *EUCitationParser) Normalize(citation *Citation) string {
	switch citation.Type {
	case CitationTypeRegulation:
		if citation.Components.DocYear != "" && citation.Components.DocNumber != "" {
			return fmt.Sprintf("Regulation (EU) %s/%s",
				citation.Components.DocYear, citation.Components.DocNumber)
		}
		return citation.RawText
	case CitationTypeDirective:
		if citation.Components.DocYear != "" && citation.Components.DocNumber != "" {
			return fmt.Sprintf("Directive %s/%s/EC",
				citation.Components.DocYear, citation.Components.DocNumber)
		}
		return citation.RawText
	case CitationTypeDecision:
		if citation.Components.DocYear != "" && citation.Components.DocNumber != "" {
			return fmt.Sprintf("Decision %s/%s/EU",
				citation.Components.DocYear, citation.Components.DocNumber)
		}
		return citation.RawText
	case CitationTypeTreaty:
		return citation.Document
	default:
		if citation.Subdivision != "" {
			return citation.Subdivision
		}
		return citation.RawText
	}
}

// ToURI generates a URN-style URI for the citation.
// URI conventions follow pkg/extract/resolver.go buildExternalURI().
func (p *EUCitationParser) ToURI(citation *Citation) (string, error) {
	switch citation.Type {
	case CitationTypeRegulation:
		if citation.Components.DocYear == "" || citation.Components.DocNumber == "" {
			return "", fmt.Errorf("regulation citation missing year or number")
		}
		return fmt.Sprintf("urn:eu:regulation:%s/%s",
			citation.Components.DocYear, citation.Components.DocNumber), nil
	case CitationTypeDirective:
		if citation.Components.DocYear == "" || citation.Components.DocNumber == "" {
			return "", fmt.Errorf("directive citation missing year or number")
		}
		return fmt.Sprintf("urn:eu:directive:%s/%s",
			citation.Components.DocYear, citation.Components.DocNumber), nil
	case CitationTypeDecision:
		if citation.Components.DocYear == "" || citation.Components.DocNumber == "" {
			return "", fmt.Errorf("decision citation missing year or number")
		}
		return fmt.Sprintf("urn:eu:decision:%s/%s",
			citation.Components.DocYear, citation.Components.DocNumber), nil
	case CitationTypeTreaty:
		if citation.Document == "" {
			return "", fmt.Errorf("treaty citation missing document identifier")
		}
		return fmt.Sprintf("urn:eu:treaty:%s", citation.Document), nil
	default:
		return "", fmt.Errorf("unsupported citation type for URI generation: %s", citation.Type)
	}
}

// parseRegulations extracts Regulation citations.
func (p *EUCitationParser) parseRegulations(text string) []*Citation {
	var citations []*Citation

	// Track matched positions to avoid duplicates between the two patterns.
	matchedPositions := make(map[int]bool)

	// "Regulation (EC) No 45/2001" — more specific pattern first.
	for _, matchIndices := range p.regulationNoPattern.FindAllStringSubmatchIndex(text, -1) {
		rawText := text[matchIndices[0]:matchIndices[1]]
		docNumber := text[matchIndices[2]:matchIndices[3]]
		docYear := text[matchIndices[4]:matchIndices[5]]
		matchedPositions[matchIndices[0]] = true
		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeRegulation,
			Jurisdiction: "EU",
			Document:     fmt.Sprintf("Regulation (EU) No %s/%s", docNumber, docYear),
			Confidence:   1.0,
			Parser:       "EU Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				DocYear:   docYear,
				DocNumber: docNumber,
			},
		})
	}

	// "Regulation (EU) 2016/679" — skip positions already matched by No pattern.
	for _, matchIndices := range p.regulationPattern.FindAllStringSubmatchIndex(text, -1) {
		if matchedPositions[matchIndices[0]] {
			continue
		}
		rawText := text[matchIndices[0]:matchIndices[1]]
		docYear := text[matchIndices[2]:matchIndices[3]]
		docNumber := text[matchIndices[4]:matchIndices[5]]
		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeRegulation,
			Jurisdiction: "EU",
			Document:     fmt.Sprintf("Regulation (EU) %s/%s", docYear, docNumber),
			Confidence:   1.0,
			Parser:       "EU Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				DocYear:   docYear,
				DocNumber: docNumber,
			},
		})
	}

	return citations
}

// parseDirectives extracts Directive citations.
func (p *EUCitationParser) parseDirectives(text string) []*Citation {
	var citations []*Citation

	for _, matchIndices := range p.directivePattern.FindAllStringSubmatchIndex(text, -1) {
		rawText := text[matchIndices[0]:matchIndices[1]]
		docYear := text[matchIndices[2]:matchIndices[3]]
		docNumber := text[matchIndices[4]:matchIndices[5]]
		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeDirective,
			Jurisdiction: "EU",
			Document:     fmt.Sprintf("Directive %s/%s/EC", docYear, docNumber),
			Confidence:   1.0,
			Parser:       "EU Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				DocYear:   docYear,
				DocNumber: docNumber,
			},
		})
	}

	return citations
}

// parseDecisions extracts Decision citations.
func (p *EUCitationParser) parseDecisions(text string) []*Citation {
	var citations []*Citation

	for _, matchIndices := range p.decisionPattern.FindAllStringSubmatchIndex(text, -1) {
		rawText := text[matchIndices[0]:matchIndices[1]]
		docYear := text[matchIndices[2]:matchIndices[3]]
		docNumber := text[matchIndices[4]:matchIndices[5]]
		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeDecision,
			Jurisdiction: "EU",
			Document:     fmt.Sprintf("Decision %s/%s/EU", docYear, docNumber),
			Confidence:   1.0,
			Parser:       "EU Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				DocYear:   docYear,
				DocNumber: docNumber,
			},
		})
	}

	return citations
}

// parseTreaties extracts Treaty citations.
func (p *EUCitationParser) parseTreaties(text string) []*Citation {
	var citations []*Citation

	for _, matchIndices := range p.treatyPattern.FindAllStringIndex(text, -1) {
		rawText := text[matchIndices[0]:matchIndices[1]]
		// Normalize to abbreviation for the Document field.
		documentName := normalizeTreatyName(rawText)
		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeTreaty,
			Jurisdiction: "EU",
			Document:     documentName,
			Confidence:   1.0,
			Parser:       "EU Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
		})
	}

	return citations
}

// normalizeTreatyName normalizes a treaty name to its standard abbreviation.
func normalizeTreatyName(rawText string) string {
	trimmed := strings.TrimSpace(rawText)
	if strings.Contains(trimmed, "Functioning") || trimmed == "TFEU" {
		return "TFEU"
	}
	if trimmed == "TEU" {
		return "TEU"
	}
	return trimmed
}

// parseArticleReferences extracts article-level internal references.
func (p *EUCitationParser) parseArticleReferences(text string) []*Citation {
	var citations []*Citation

	// Track positions matched by more specific patterns to avoid duplicates.
	matchedPositions := make(map[int]bool)

	// "Articles 13 and 14" / "Articles 15 to 22" — range pattern first.
	for _, matchIndices := range p.articlesRangePattern.FindAllStringSubmatchIndex(text, -1) {
		rawText := text[matchIndices[0]:matchIndices[1]]
		startArticle := text[matchIndices[2]:matchIndices[3]]
		endArticle := text[matchIndices[4]:matchIndices[5]]
		startNum, _ := strconv.Atoi(startArticle)
		endNum, _ := strconv.Atoi(endArticle)
		matchedPositions[matchIndices[0]] = true
		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeStatute,
			Jurisdiction: "EU",
			Subdivision:  fmt.Sprintf("Articles %s to %s", startArticle, endArticle),
			Confidence:   0.9,
			Parser:       "EU Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				ArticleNumber: startNum,
			},
		})
		// Also emit the end article if different.
		if endNum != startNum {
			citations = append(citations, &Citation{
				RawText:      rawText,
				Type:         CitationTypeStatute,
				Jurisdiction: "EU",
				Subdivision:  fmt.Sprintf("Article %s", endArticle),
				Confidence:   0.9,
				Parser:       "EU Citation Parser",
				TextOffset:   matchIndices[0],
				TextLength:   matchIndices[1] - matchIndices[0],
				Components: CitationComponents{
					ArticleNumber: endNum,
				},
			})
		}
	}

	// "Article 6(1)(a)" — parenthetical pattern, more specific than plain article.
	for _, matchIndices := range p.articleParenPattern.FindAllStringSubmatchIndex(text, -1) {
		if matchedPositions[matchIndices[0]] {
			continue
		}
		rawText := text[matchIndices[0]:matchIndices[1]]
		articleStr := text[matchIndices[2]:matchIndices[3]]
		paragraphStr := text[matchIndices[4]:matchIndices[5]]
		articleNum, _ := strconv.Atoi(articleStr)
		paragraphNum, _ := strconv.Atoi(paragraphStr)

		var pointLetter string
		if matchIndices[6] != -1 {
			pointLetter = text[matchIndices[6]:matchIndices[7]]
		}

		matchedPositions[matchIndices[0]] = true

		subdivision := fmt.Sprintf("Article %s(%s)", articleStr, paragraphStr)
		if pointLetter != "" {
			subdivision += fmt.Sprintf("(%s)", pointLetter)
		}

		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeStatute,
			Jurisdiction: "EU",
			Subdivision:  subdivision,
			Confidence:   1.0,
			Parser:       "EU Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				ArticleNumber:   articleNum,
				ParagraphNumber: paragraphNum,
				PointLetter:     pointLetter,
			},
		})
	}

	// "Article 6" — plain article, skip positions already matched.
	for _, matchIndices := range p.articlePattern.FindAllStringSubmatchIndex(text, -1) {
		if matchedPositions[matchIndices[0]] {
			continue
		}
		rawText := text[matchIndices[0]:matchIndices[1]]
		articleStr := text[matchIndices[2]:matchIndices[3]]
		articleNum, _ := strconv.Atoi(articleStr)
		matchedPositions[matchIndices[0]] = true

		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeStatute,
			Jurisdiction: "EU",
			Subdivision:  fmt.Sprintf("Article %s", articleStr),
			Confidence:   0.9,
			Parser:       "EU Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				ArticleNumber: articleNum,
			},
		})
	}

	return citations
}

// parseChapterReferences extracts chapter-level internal references.
func (p *EUCitationParser) parseChapterReferences(text string) []*Citation {
	var citations []*Citation

	for _, matchIndices := range p.chapterPattern.FindAllStringSubmatchIndex(text, -1) {
		rawText := text[matchIndices[0]:matchIndices[1]]
		chapterNumeral := text[matchIndices[2]:matchIndices[3]]
		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeStatute,
			Jurisdiction: "EU",
			Subdivision:  fmt.Sprintf("Chapter %s", chapterNumeral),
			Confidence:   0.9,
			Parser:       "EU Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				ChapterNumber: chapterNumeral,
			},
		})
	}

	return citations
}

// parseSectionReferences extracts section-level internal references.
func (p *EUCitationParser) parseSectionReferences(text string) []*Citation {
	var citations []*Citation

	for _, matchIndices := range p.sectionPattern.FindAllStringSubmatchIndex(text, -1) {
		rawText := text[matchIndices[0]:matchIndices[1]]
		sectionNum := text[matchIndices[2]:matchIndices[3]]
		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeStatute,
			Jurisdiction: "EU",
			Subdivision:  fmt.Sprintf("Section %s", sectionNum),
			Confidence:   0.85,
			Parser:       "EU Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
		})
	}

	return citations
}
