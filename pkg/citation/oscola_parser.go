package citation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// OSCOLAParser parses UK/Commonwealth legal citations following OSCOLA conventions:
//   - UK Acts: "Data Protection Act 2018", "Human Rights Act 1998"
//   - Statutory Instruments: "SI 2019/419", "Statutory Instruments 2019 No. 419"
//   - Section references: "s 6(1)", "section 114", "ss 12-14"
//   - Neutral citations: "[2019] UKSC 5", "[2021] EWCA Civ 1234"
//   - Law report citations: "[1994] 1 AC 212"
//   - ECHR articles: "ECHR art 8", "ECHR article 6(1)"
//   - Structural references: "Schedule 7", "Part 3"
type OSCOLAParser struct {
	// Primary legislation patterns.
	actPattern        *regexp.Regexp // Data Protection Act 2018
	actChapterPattern *regexp.Regexp // [2018 c. 12]

	// Statutory Instruments patterns.
	siPattern     *regexp.Regexp // SI 2019/419
	siLongPattern *regexp.Regexp // Statutory Instruments 2019 No. 419

	// Section reference patterns.
	sectionAbbrPattern *regexp.Regexp // s 6(1), ss 12
	sectionWordPattern *regexp.Regexp // section 114, sections 12

	// Case citation patterns.
	neutralCitePattern *regexp.Regexp // [2019] UKSC 5
	lawReportPattern   *regexp.Regexp // [1994] 1 AC 212

	// Treaty reference patterns.
	echrPattern *regexp.Regexp // ECHR art 8

	// Structural reference patterns.
	schedulePattern *regexp.Regexp // Schedule 7
	partPattern     *regexp.Regexp // Part 3
}

// NewOSCOLAParser creates a new OSCOLA citation parser with compiled patterns.
func NewOSCOLAParser() *OSCOLAParser {
	return &OSCOLAParser{
		// UK Acts: "Data Protection Act 2018", "the Human Rights Act 1998"
		// Handles parenthesized words like "European Union (Withdrawal) Act 2018"
		// Captures: (1) act name, (2) year
		actPattern: regexp.MustCompile(`(?:the\s+)?([A-Z][a-zA-Z][\w\s()]*?)\s+Act\s+(\d{4})`),

		// Chapter number in brackets: "[2018 c. 12]"
		// Captures: (1) year, (2) chapter number
		actChapterPattern: regexp.MustCompile(`\[(\d{4})\s+c\.\s+(\d+)\]`),

		// Short SI: "SI 2019/419"
		// Captures: (1) year, (2) number
		siPattern: regexp.MustCompile(`SI\s+(\d{4})/(\d+)`),

		// Long SI: "Statutory Instruments 2019 No. 419" or "Statutory Instrument 2019 No. 419"
		// Captures: (1) year, (2) number
		siLongPattern: regexp.MustCompile(`Statutory\s+Instruments?\s+(\d{4})\s+No\.\s+(\d+)`),

		// Section abbreviation: "s 6", "s 6(1)", "s 6(1)(a)", "ss 12"
		// Captures: (1) section number, (2) optional subsection, (3) optional paragraph letter
		sectionAbbrPattern: regexp.MustCompile(`\bss?\s+(\d+)(?:\((\d+)\))?(?:\(([a-z])\))?`),

		// Section word: "section 114", "section 6(1)", "sections 12"
		// Captures: (1) section number, (2) optional subsection, (3) optional paragraph letter
		sectionWordPattern: regexp.MustCompile(`\b[Ss]ections?\s+(\d+)(?:\((\d+)\))?(?:\(([a-z])\))?`),

		// Neutral citations: "[2019] UKSC 5", "[2021] EWCA Civ 1234", "[2020] EWHC 999"
		// Captures: (1) year, (2) court, (3) case number
		neutralCitePattern: regexp.MustCompile(`\[(\d{4})\]\s+(UKSC|UKHL|EWCA\s+(?:Civ|Crim)|EWHC|UKPC|UKUT|UKFTT)\s+(\d+)`),

		// Law report citations: "[1994] 1 AC 212", "[2003] QB 195"
		// Captures: (1) year, (2) optional volume, (3) report abbreviation, (4) page
		lawReportPattern: regexp.MustCompile(`\[(\d{4})\]\s+(\d+\s+)?([A-Z][A-Z.]+(?:\s+[A-Z.]+)?)\s+(\d+)`),

		// ECHR article references: "ECHR art 8", "ECHR article 6(1)"
		// Captures: (1) article number, (2) optional paragraph
		echrPattern: regexp.MustCompile(`ECHR\s+art(?:icle)?\s+(\d+)(?:\((\d+)\))?`),

		// Schedule references: "Schedule 7", "Schedule 12"
		// Captures: (1) schedule number
		schedulePattern: regexp.MustCompile(`Schedule\s+(\d+)`),

		// Part references: "Part 3", "Part 7"
		// Captures: (1) part number
		partPattern: regexp.MustCompile(`Part\s+(\d+)`),
	}
}

// Name returns the parser name.
func (parser *OSCOLAParser) Name() string {
	return "OSCOLA Citation Parser"
}

// Jurisdictions returns supported jurisdiction codes.
func (parser *OSCOLAParser) Jurisdictions() []string {
	return []string{"UK"}
}

// Parse extracts all OSCOLA-style citations from the text.
func (parser *OSCOLAParser) Parse(text string) ([]*Citation, error) {
	var citations []*Citation
	matchedPositions := make(map[int]bool)

	// Parse in specificity order: most specific patterns first.
	citations = append(citations, parser.parseStatutoryInstruments(text, matchedPositions)...)
	citations = append(citations, parser.parseActChapterRefs(text, matchedPositions)...)
	citations = append(citations, parser.parseActs(text, matchedPositions)...)
	citations = append(citations, parser.parseNeutralCitations(text, matchedPositions)...)
	citations = append(citations, parser.parseLawReportCitations(text, matchedPositions)...)
	citations = append(citations, parser.parseECHR(text, matchedPositions)...)
	citations = append(citations, parser.parseSectionReferences(text, matchedPositions)...)
	citations = append(citations, parser.parseScheduleReferences(text, matchedPositions)...)
	citations = append(citations, parser.parsePartReferences(text, matchedPositions)...)

	return citations, nil
}

// Normalize converts a citation to its canonical OSCOLA form.
func (parser *OSCOLAParser) Normalize(citation *Citation) string {
	switch citation.Type {
	case CitationTypeStatute:
		if citation.Components.CodeName == "ukact" {
			if citation.Components.DocNumber != "" {
				return fmt.Sprintf("%s [%s c. %s]",
					citation.Document, citation.Components.DocYear, citation.Components.DocNumber)
			}
			return citation.Document
		}
		// Section reference
		if citation.Components.Section != "" {
			return normalizeSection(citation)
		}
		// Schedule or Part
		if citation.Subdivision != "" {
			return citation.Subdivision
		}
		return citation.RawText

	case CitationTypeRegulation:
		if citation.Components.DocYear != "" && citation.Components.DocNumber != "" {
			return fmt.Sprintf("SI %s/%s", citation.Components.DocYear, citation.Components.DocNumber)
		}
		return citation.RawText

	case CitationTypeCase:
		if citation.Components.CodeName == "neutral" {
			return fmt.Sprintf("[%s] %s",
				citation.Components.DocYear, citation.Components.DocNumber)
		}
		if citation.Subdivision != "" {
			return fmt.Sprintf("[%s] %s", citation.Components.DocYear, citation.Subdivision)
		}
		return citation.RawText

	case CitationTypeTreaty:
		return normalizeECHRArticle(citation)

	default:
		return citation.RawText
	}
}

// ToURI generates a URN-style URI for the citation.
func (parser *OSCOLAParser) ToURI(citation *Citation) (string, error) {
	switch citation.Type {
	case CitationTypeStatute:
		if citation.Components.CodeName == "ukact" {
			if citation.Components.DocYear == "" {
				return "", fmt.Errorf("UK act citation missing year")
			}
			if citation.Components.DocNumber != "" {
				return fmt.Sprintf("urn:uk:act:%s/%s",
					citation.Components.DocYear, citation.Components.DocNumber), nil
			}
			actSlug := slugifyActName(citation.Document)
			return fmt.Sprintf("urn:uk:act:%s/%s",
				citation.Components.DocYear, actSlug), nil
		}
		return "", fmt.Errorf("cannot generate URI for section/structural reference")

	case CitationTypeRegulation:
		if citation.Components.DocYear == "" || citation.Components.DocNumber == "" {
			return "", fmt.Errorf("SI citation missing year or number")
		}
		return fmt.Sprintf("urn:uk:si:%s/%s",
			citation.Components.DocYear, citation.Components.DocNumber), nil

	case CitationTypeCase:
		if citation.Components.DocYear == "" || citation.Components.DocNumber == "" {
			return "", fmt.Errorf("case citation missing year or court/number")
		}
		return fmt.Sprintf("urn:uk:case:%s/%s",
			citation.Components.DocYear, citation.Components.DocNumber), nil

	case CitationTypeTreaty:
		articleNum := citation.Components.ArticleNumber
		if articleNum == 0 {
			return "", fmt.Errorf("ECHR citation missing article number")
		}
		return fmt.Sprintf("urn:echr:article:%d", articleNum), nil

	default:
		return "", fmt.Errorf("unsupported citation type for URI generation: %s", citation.Type)
	}
}

// parseStatutoryInstruments extracts SI citations using both short and long forms.
func (parser *OSCOLAParser) parseStatutoryInstruments(text string, matchedPositions map[int]bool) []*Citation {
	var citations []*Citation

	// Long form first (more specific): "Statutory Instruments 2019 No. 419"
	for _, matchIndices := range parser.siLongPattern.FindAllStringSubmatchIndex(text, -1) {
		rawText := text[matchIndices[0]:matchIndices[1]]
		siYear := text[matchIndices[2]:matchIndices[3]]
		siNumber := text[matchIndices[4]:matchIndices[5]]
		markRange(matchedPositions, matchIndices[0], matchIndices[1])
		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeRegulation,
			Jurisdiction: "UK",
			Document:     fmt.Sprintf("SI %s/%s", siYear, siNumber),
			Confidence:   1.0,
			Parser:       "OSCOLA Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				DocYear:   siYear,
				DocNumber: siNumber,
			},
		})
	}

	// Short form: "SI 2019/419"
	for _, matchIndices := range parser.siPattern.FindAllStringSubmatchIndex(text, -1) {
		if isPositionMatched(matchedPositions, matchIndices[0], matchIndices[1]) {
			continue
		}
		rawText := text[matchIndices[0]:matchIndices[1]]
		siYear := text[matchIndices[2]:matchIndices[3]]
		siNumber := text[matchIndices[4]:matchIndices[5]]
		markRange(matchedPositions, matchIndices[0], matchIndices[1])
		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeRegulation,
			Jurisdiction: "UK",
			Document:     fmt.Sprintf("SI %s/%s", siYear, siNumber),
			Confidence:   1.0,
			Parser:       "OSCOLA Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				DocYear:   siYear,
				DocNumber: siNumber,
			},
		})
	}

	return citations
}

// parseActChapterRefs extracts act chapter references like "[2018 c. 12]".
func (parser *OSCOLAParser) parseActChapterRefs(text string, matchedPositions map[int]bool) []*Citation {
	var citations []*Citation

	for _, matchIndices := range parser.actChapterPattern.FindAllStringSubmatchIndex(text, -1) {
		if isPositionMatched(matchedPositions, matchIndices[0], matchIndices[1]) {
			continue
		}
		rawText := text[matchIndices[0]:matchIndices[1]]
		actYear := text[matchIndices[2]:matchIndices[3]]
		actChapter := text[matchIndices[4]:matchIndices[5]]
		markRange(matchedPositions, matchIndices[0], matchIndices[1])
		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeStatute,
			Jurisdiction: "UK",
			Document:     fmt.Sprintf("[%s c. %s]", actYear, actChapter),
			Confidence:   1.0,
			Parser:       "OSCOLA Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				DocYear:   actYear,
				DocNumber: actChapter,
				CodeName:  "ukact",
			},
		})
	}

	return citations
}

// parseActs extracts UK Act citations like "Data Protection Act 2018".
func (parser *OSCOLAParser) parseActs(text string, matchedPositions map[int]bool) []*Citation {
	var citations []*Citation

	for _, matchIndices := range parser.actPattern.FindAllStringSubmatchIndex(text, -1) {
		if isPositionMatched(matchedPositions, matchIndices[0], matchIndices[1]) {
			continue
		}
		rawText := text[matchIndices[0]:matchIndices[1]]
		actName := strings.TrimSpace(text[matchIndices[2]:matchIndices[3]])
		actYear := text[matchIndices[4]:matchIndices[5]]
		markRange(matchedPositions, matchIndices[0], matchIndices[1])

		documentName := fmt.Sprintf("%s Act %s", actName, actYear)
		// Strip leading "the " from the raw text for normalization.
		if strings.HasPrefix(strings.ToLower(rawText), "the ") {
			documentName = fmt.Sprintf("%s Act %s", actName, actYear)
		}

		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeStatute,
			Jurisdiction: "UK",
			Document:     documentName,
			Confidence:   0.95,
			Parser:       "OSCOLA Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				DocYear:  actYear,
				CodeName: "ukact",
			},
		})
	}

	return citations
}

// parseNeutralCitations extracts neutral case citations like "[2019] UKSC 5".
func (parser *OSCOLAParser) parseNeutralCitations(text string, matchedPositions map[int]bool) []*Citation {
	var citations []*Citation

	for _, matchIndices := range parser.neutralCitePattern.FindAllStringSubmatchIndex(text, -1) {
		if isPositionMatched(matchedPositions, matchIndices[0], matchIndices[1]) {
			continue
		}
		rawText := text[matchIndices[0]:matchIndices[1]]
		caseYear := text[matchIndices[2]:matchIndices[3]]
		courtCode := text[matchIndices[4]:matchIndices[5]]
		caseNumber := text[matchIndices[6]:matchIndices[7]]
		markRange(matchedPositions, matchIndices[0], matchIndices[1])

		normalizedCourt := strings.ReplaceAll(courtCode, " ", "-")
		courtCaseIdentifier := fmt.Sprintf("%s %s", courtCode, caseNumber)

		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeCase,
			Jurisdiction: "UK",
			Document:     fmt.Sprintf("[%s] %s", caseYear, courtCaseIdentifier),
			Confidence:   1.0,
			Parser:       "OSCOLA Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				DocYear:   caseYear,
				DocNumber: fmt.Sprintf("%s/%s", normalizedCourt, caseNumber),
				CodeName:  "neutral",
			},
		})
	}

	return citations
}

// parseLawReportCitations extracts traditional law report citations like "[1994] 1 AC 212".
func (parser *OSCOLAParser) parseLawReportCitations(text string, matchedPositions map[int]bool) []*Citation {
	var citations []*Citation

	// Known UK law report abbreviations to filter false positives.
	knownReporters := map[string]bool{
		"AC":      true, // Appeal Cases
		"QB":      true, // Queen's Bench
		"KB":      true, // King's Bench
		"Ch":      true, // Chancery
		"WLR":     true, // Weekly Law Reports
		"All ER":  true, // All England Reports
		"Cr App R": true, // Criminal Appeal Reports
		"BCLC":    true, // Butterworths Company Law Cases
		"FLR":     true, // Family Law Reports
		"ICR":     true, // Industrial Cases Reports
		"IRLR":    true, // Industrial Relations Law Reports
		"Lloyd's Rep": true, // Lloyd's Law Reports
	}

	for _, matchIndices := range parser.lawReportPattern.FindAllStringSubmatchIndex(text, -1) {
		if isPositionMatched(matchedPositions, matchIndices[0], matchIndices[1]) {
			continue
		}

		rawText := text[matchIndices[0]:matchIndices[1]]
		reportYear := text[matchIndices[2]:matchIndices[3]]

		var volumeStr string
		if matchIndices[4] != -1 {
			volumeStr = strings.TrimSpace(text[matchIndices[4]:matchIndices[5]])
		}

		reportAbbrev := text[matchIndices[6]:matchIndices[7]]
		pageNumber := text[matchIndices[8]:matchIndices[9]]

		// Only accept known law report abbreviations to reduce false positives.
		if !knownReporters[reportAbbrev] {
			continue
		}

		markRange(matchedPositions, matchIndices[0], matchIndices[1])

		var subdivisionParts string
		if volumeStr != "" {
			subdivisionParts = fmt.Sprintf("%s %s %s", volumeStr, reportAbbrev, pageNumber)
		} else {
			subdivisionParts = fmt.Sprintf("%s %s", reportAbbrev, pageNumber)
		}

		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeCase,
			Jurisdiction: "UK",
			Document:     fmt.Sprintf("[%s] %s", reportYear, subdivisionParts),
			Subdivision:  subdivisionParts,
			Confidence:   0.9,
			Parser:       "OSCOLA Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				DocYear:   reportYear,
				DocNumber: fmt.Sprintf("%s/%s/%s", reportAbbrev, volumeStr, pageNumber),
				CodeName:  "lawreport",
			},
		})
	}

	return citations
}

// parseECHR extracts ECHR article references like "ECHR art 8".
func (parser *OSCOLAParser) parseECHR(text string, matchedPositions map[int]bool) []*Citation {
	var citations []*Citation

	for _, matchIndices := range parser.echrPattern.FindAllStringSubmatchIndex(text, -1) {
		if isPositionMatched(matchedPositions, matchIndices[0], matchIndices[1]) {
			continue
		}
		rawText := text[matchIndices[0]:matchIndices[1]]
		articleNumStr := text[matchIndices[2]:matchIndices[3]]
		articleNum, _ := strconv.Atoi(articleNumStr)

		var paragraphNum int
		if matchIndices[4] != -1 {
			paragraphStr := text[matchIndices[4]:matchIndices[5]]
			paragraphNum, _ = strconv.Atoi(paragraphStr)
		}

		markRange(matchedPositions, matchIndices[0], matchIndices[1])

		subdivision := fmt.Sprintf("art %s", articleNumStr)
		if paragraphNum > 0 {
			subdivision = fmt.Sprintf("art %s(%d)", articleNumStr, paragraphNum)
		}

		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeTreaty,
			Jurisdiction: "UK",
			Document:     "ECHR",
			Subdivision:  subdivision,
			Confidence:   1.0,
			Parser:       "OSCOLA Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				ArticleNumber:   articleNum,
				ParagraphNumber: paragraphNum,
			},
		})
	}

	return citations
}

// parseSectionReferences extracts section references using both "s" abbreviation and "section" word forms.
func (parser *OSCOLAParser) parseSectionReferences(text string, matchedPositions map[int]bool) []*Citation {
	var citations []*Citation

	// "section"/"sections" word form first (more specific due to word length).
	for _, matchIndices := range parser.sectionWordPattern.FindAllStringSubmatchIndex(text, -1) {
		if isPositionMatched(matchedPositions, matchIndices[0], matchIndices[1]) {
			continue
		}
		citation := parseSectionMatch(text, matchIndices, 0.85)
		if citation != nil {
			markRange(matchedPositions, matchIndices[0], matchIndices[1])
			citations = append(citations, citation)
		}
	}

	// "s"/"ss" abbreviation form.
	for _, matchIndices := range parser.sectionAbbrPattern.FindAllStringSubmatchIndex(text, -1) {
		if isPositionMatched(matchedPositions, matchIndices[0], matchIndices[1]) {
			continue
		}
		citation := parseSectionMatch(text, matchIndices, 0.9)
		if citation != nil {
			markRange(matchedPositions, matchIndices[0], matchIndices[1])
			citations = append(citations, citation)
		}
	}

	return citations
}

// parseScheduleReferences extracts "Schedule N" references.
func (parser *OSCOLAParser) parseScheduleReferences(text string, matchedPositions map[int]bool) []*Citation {
	var citations []*Citation

	for _, matchIndices := range parser.schedulePattern.FindAllStringSubmatchIndex(text, -1) {
		if isPositionMatched(matchedPositions, matchIndices[0], matchIndices[1]) {
			continue
		}
		rawText := text[matchIndices[0]:matchIndices[1]]
		scheduleNum := text[matchIndices[2]:matchIndices[3]]
		markRange(matchedPositions, matchIndices[0], matchIndices[1])

		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeStatute,
			Jurisdiction: "UK",
			Subdivision:  fmt.Sprintf("Schedule %s", scheduleNum),
			Confidence:   0.85,
			Parser:       "OSCOLA Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				ChapterNumber: scheduleNum,
			},
		})
	}

	return citations
}

// parsePartReferences extracts "Part N" references.
func (parser *OSCOLAParser) parsePartReferences(text string, matchedPositions map[int]bool) []*Citation {
	var citations []*Citation

	for _, matchIndices := range parser.partPattern.FindAllStringSubmatchIndex(text, -1) {
		if isPositionMatched(matchedPositions, matchIndices[0], matchIndices[1]) {
			continue
		}
		rawText := text[matchIndices[0]:matchIndices[1]]
		partNum := text[matchIndices[2]:matchIndices[3]]
		markRange(matchedPositions, matchIndices[0], matchIndices[1])

		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeStatute,
			Jurisdiction: "UK",
			Subdivision:  fmt.Sprintf("Part %s", partNum),
			Confidence:   0.8,
			Parser:       "OSCOLA Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				ChapterNumber: partNum,
			},
		})
	}

	return citations
}

// parseSectionMatch creates a Citation from a section regex match.
func parseSectionMatch(text string, matchIndices []int, confidence float64) *Citation {
	rawText := text[matchIndices[0]:matchIndices[1]]
	sectionNumStr := text[matchIndices[2]:matchIndices[3]]
	sectionNum, _ := strconv.Atoi(sectionNumStr)

	var subsectionNum int
	if matchIndices[4] != -1 {
		subsectionStr := text[matchIndices[4]:matchIndices[5]]
		subsectionNum, _ = strconv.Atoi(subsectionStr)
	}

	var pointLetter string
	if matchIndices[6] != -1 {
		pointLetter = text[matchIndices[6]:matchIndices[7]]
	}

	subdivision := fmt.Sprintf("s %s", sectionNumStr)
	if subsectionNum > 0 {
		subdivision = fmt.Sprintf("s %s(%d)", sectionNumStr, subsectionNum)
	}
	if pointLetter != "" {
		subdivision = fmt.Sprintf("s %s(%d)(%s)", sectionNumStr, subsectionNum, pointLetter)
	}

	return &Citation{
		RawText:      rawText,
		Type:         CitationTypeStatute,
		Jurisdiction: "UK",
		Subdivision:  subdivision,
		Confidence:   confidence,
		Parser:       "OSCOLA Citation Parser",
		TextOffset:   matchIndices[0],
		TextLength:   matchIndices[1] - matchIndices[0],
		Components: CitationComponents{
			Section:         sectionNumStr,
			ArticleNumber:   sectionNum,
			ParagraphNumber: subsectionNum,
			PointLetter:     pointLetter,
		},
	}
}

// normalizeSection produces the canonical OSCOLA section reference.
func normalizeSection(citation *Citation) string {
	sectionStr := citation.Components.Section
	result := fmt.Sprintf("s %s", sectionStr)
	if citation.Components.ParagraphNumber > 0 {
		result = fmt.Sprintf("s %s(%d)", sectionStr, citation.Components.ParagraphNumber)
	}
	if citation.Components.PointLetter != "" {
		result = fmt.Sprintf("s %s(%d)(%s)", sectionStr, citation.Components.ParagraphNumber, citation.Components.PointLetter)
	}
	return result
}

// normalizeECHRArticle produces the canonical ECHR article reference.
func normalizeECHRArticle(citation *Citation) string {
	result := fmt.Sprintf("ECHR art %d", citation.Components.ArticleNumber)
	if citation.Components.ParagraphNumber > 0 {
		result = fmt.Sprintf("ECHR art %d(%d)", citation.Components.ArticleNumber, citation.Components.ParagraphNumber)
	}
	return result
}

// slugifyActName converts an act name to a URL-safe slug.
func slugifyActName(documentName string) string {
	slug := strings.ToLower(documentName)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove trailing year portion like "-act-2018"
	return slug
}

// markRange marks all positions in the range as matched.
func markRange(positions map[int]bool, start, end int) {
	for i := start; i < end; i++ {
		positions[i] = true
	}
}

// isPositionMatched checks if any position in the given range is already matched.
func isPositionMatched(positions map[int]bool, start, end int) bool {
	for i := start; i < end; i++ {
		if positions[i] {
			return true
		}
	}
	return false
}
