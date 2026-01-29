package citation

import (
	"fmt"
	"regexp"
	"strings"
)

// BluebookParser parses US-style legal citations following Bluebook conventions:
//   - U.S. Code: "42 U.S.C. § 1983", "15 U.S.C. Section 1681"
//   - C.F.R.: "45 C.F.R. Part 164", "21 C.F.R. § 50"
//   - Public Laws: "Pub. L. 104-191", "Public Law 106-102"
//   - Case citations: "Brown v. Board of Education, 347 U.S. 483 (1954)"
//
// This parser handles statutory citations common in US privacy laws (CCPA, HIPAA, VCDPA).
type BluebookParser struct {
	// Statutory citation patterns.
	uscPattern       *regexp.Regexp // 42 U.S.C. § 1983
	uscSecPattern    *regexp.Regexp // 42 U.S.C. Section 1681
	uscEtSeqPattern  *regexp.Regexp // 15 U.S.C. Section 1681 et seq.
	cfrPattern       *regexp.Regexp // 45 C.F.R. Part 164
	cfrPartsPattern  *regexp.Regexp // 45 C.F.R. Parts 160 and 164
	publicLawPattern *regexp.Regexp // Public Law 104-191, Pub. L. 111-5

	// Case citation patterns.
	casePattern *regexp.Regexp // Brown v. Board, 347 U.S. 483 (1954)
}

// NewBluebookParser creates a new Bluebook citation parser with compiled patterns.
func NewBluebookParser() *BluebookParser {
	return &BluebookParser{
		// U.S. Code citations with section symbol: "42 U.S.C. § 1983" or "42 U.S.C. §§ 1983-1988"
		uscPattern: regexp.MustCompile(`(\d+)\s+U\.?S\.?C\.?\s+§§?\s*(\d+[a-z]?)(?:[-–](\d+[a-z]?))?`),

		// U.S. Code citations with "Section" or "Sec.": "15 U.S.C. Section 1681"
		uscSecPattern: regexp.MustCompile(`(\d+)\s+U\.?S\.?C\.?\s+(?:Section|Sec\.?)\s+(\d+[a-z]?)`),

		// U.S. Code citations with "et seq.": "15 U.S.C. Section 1681 et seq."
		uscEtSeqPattern: regexp.MustCompile(`(\d+)\s+U\.?S\.?C\.?\s+(?:Section|Sec\.?|§)\s*(\d+[a-z]?)\s+et\s+seq\.?`),

		// C.F.R. citations: "45 C.F.R. Part 164" or "45 C.F.R. § 164.502"
		cfrPattern: regexp.MustCompile(`(\d+)\s+C\.?F\.?R\.?\s+(?:Part\s+|§\s*)?(\d+)(?:\.(\d+))?`),

		// C.F.R. multiple parts: "45 C.F.R. Parts 160 and 164"
		cfrPartsPattern: regexp.MustCompile(`(\d+)\s+C\.?F\.?R\.?\s+Parts\s+(\d+)\s+and\s+(\d+)`),

		// Public Law citations: "Public Law 104-191" or "Pub. L. 104-191" or "P.L. 111-5"
		publicLawPattern: regexp.MustCompile(`(?:Public\s+Law|Pub\.?\s*L\.?|P\.?L\.?)\s+(\d+)[-–](\d+)`),

		// US Supreme Court case citations: "Brown v. Board of Education, 347 U.S. 483 (1954)"
		// Pattern captures: party1, party2, volume, reporter, page, year
		// More permissive pattern to handle variations in case names
		casePattern: regexp.MustCompile(`([A-Z][a-zA-Z'\-]+(?:\s+[a-zA-Z'\-]+)*)\s+v\.?\s+([A-Z][a-zA-Z'\-]+(?:\s+[a-zA-Z'\-]+)*),?\s+(\d+)\s+(U\.S\.|S\.\s*Ct\.|F\.\d+[a-z]*|F\.\s*Supp\.\s*\d*[a-z]*)\s+(\d+)\s+\((\d{4})\)`),
	}
}

// Name returns the parser name.
func (parser *BluebookParser) Name() string {
	return "Bluebook Citation Parser"
}

// Jurisdictions returns supported jurisdiction codes.
func (parser *BluebookParser) Jurisdictions() []string {
	return []string{"US"}
}

// Parse extracts all Bluebook-style citations from the text.
func (parser *BluebookParser) Parse(text string) ([]*Citation, error) {
	var citations []*Citation

	// Track matched positions to avoid duplicates from overlapping patterns.
	matchedPositions := make(map[int]bool)

	// Parse in order of specificity: more specific patterns first.
	citations = append(citations, parser.parseUSCEtSeq(text, matchedPositions)...)
	citations = append(citations, parser.parseUSC(text, matchedPositions)...)
	citations = append(citations, parser.parseCFRParts(text, matchedPositions)...)
	citations = append(citations, parser.parseCFR(text, matchedPositions)...)
	citations = append(citations, parser.parsePublicLaw(text, matchedPositions)...)
	citations = append(citations, parser.parseCases(text, matchedPositions)...)

	return citations, nil
}

// Normalize converts a citation to its canonical Bluebook form.
func (parser *BluebookParser) Normalize(citation *Citation) string {
	switch citation.Type {
	case CitationTypeCode:
		if citation.Components.CodeName == "USC" {
			if citation.Components.Title != "" && citation.Components.Section != "" {
				return fmt.Sprintf("%s U.S.C. § %s", citation.Components.Title, citation.Components.Section)
			}
		} else if citation.Components.CodeName == "CFR" {
			if citation.Components.Title != "" && citation.Components.Section != "" {
				return fmt.Sprintf("%s C.F.R. § %s", citation.Components.Title, citation.Components.Section)
			}
		}
		return citation.RawText

	case CitationTypeStatute:
		if citation.Components.PublicLaw != "" {
			return fmt.Sprintf("Pub. L. %s", citation.Components.PublicLaw)
		}
		return citation.RawText

	case CitationTypeCase:
		return citation.Document

	default:
		return citation.RawText
	}
}

// ToURI generates a URN-style URI for the citation.
func (parser *BluebookParser) ToURI(citation *Citation) (string, error) {
	switch citation.Type {
	case CitationTypeCode:
		if citation.Components.CodeName == "USC" {
			if citation.Components.Title == "" || citation.Components.Section == "" {
				return "", fmt.Errorf("USC citation missing title or section")
			}
			return fmt.Sprintf("urn:us:usc:%s/%s",
				citation.Components.Title, citation.Components.Section), nil
		}
		if citation.Components.CodeName == "CFR" {
			if citation.Components.Title == "" || citation.Components.Section == "" {
				return "", fmt.Errorf("CFR citation missing title or section")
			}
			return fmt.Sprintf("urn:us:cfr:%s/%s",
				citation.Components.Title, citation.Components.Section), nil
		}
		return "", fmt.Errorf("unsupported code type: %s", citation.Components.CodeName)

	case CitationTypeStatute:
		if citation.Components.PublicLaw != "" {
			return fmt.Sprintf("urn:us:pl:%s", citation.Components.PublicLaw), nil
		}
		return "", fmt.Errorf("statute citation missing public law number")

	case CitationTypeCase:
		// Case URIs use a simplified format: volume-reporter-page
		if citation.Components.DocNumber != "" && citation.Components.DocYear != "" {
			// DocNumber holds "volume/reporter/page", DocYear holds year
			return fmt.Sprintf("urn:us:case:%s", citation.Components.DocNumber), nil
		}
		return "", fmt.Errorf("case citation missing volume/page information")

	default:
		return "", fmt.Errorf("unsupported citation type for URI generation: %s", citation.Type)
	}
}

// parseUSCEtSeq extracts U.S. Code citations with "et seq." suffix.
func (parser *BluebookParser) parseUSCEtSeq(text string, matched map[int]bool) []*Citation {
	var citations []*Citation

	for _, matchIndices := range parser.uscEtSeqPattern.FindAllStringSubmatchIndex(text, -1) {
		if matched[matchIndices[0]] {
			continue
		}

		rawText := text[matchIndices[0]:matchIndices[1]]
		title := text[matchIndices[2]:matchIndices[3]]
		section := text[matchIndices[4]:matchIndices[5]]

		matched[matchIndices[0]] = true

		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeCode,
			Jurisdiction: "US",
			Document:     fmt.Sprintf("%s U.S.C. § %s et seq.", title, section),
			Confidence:   1.0,
			Parser:       "Bluebook Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				Title:    title,
				Section:  section,
				CodeName: "USC",
			},
		})
	}

	return citations
}

// parseUSC extracts U.S. Code citations.
func (parser *BluebookParser) parseUSC(text string, matched map[int]bool) []*Citation {
	var citations []*Citation

	// First, try Section/Sec. pattern
	for _, matchIndices := range parser.uscSecPattern.FindAllStringSubmatchIndex(text, -1) {
		if matched[matchIndices[0]] {
			continue
		}

		rawText := text[matchIndices[0]:matchIndices[1]]
		title := text[matchIndices[2]:matchIndices[3]]
		section := text[matchIndices[4]:matchIndices[5]]

		matched[matchIndices[0]] = true

		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeCode,
			Jurisdiction: "US",
			Document:     fmt.Sprintf("%s U.S.C. § %s", title, section),
			Confidence:   1.0,
			Parser:       "Bluebook Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				Title:    title,
				Section:  section,
				CodeName: "USC",
			},
		})
	}

	// Then, try § symbol pattern
	for _, matchIndices := range parser.uscPattern.FindAllStringSubmatchIndex(text, -1) {
		if matched[matchIndices[0]] {
			continue
		}

		rawText := text[matchIndices[0]:matchIndices[1]]
		title := text[matchIndices[2]:matchIndices[3]]
		section := text[matchIndices[4]:matchIndices[5]]

		// Check for range end section
		var rangeEnd string
		if matchIndices[6] != -1 {
			rangeEnd = text[matchIndices[6]:matchIndices[7]]
		}

		matched[matchIndices[0]] = true

		document := fmt.Sprintf("%s U.S.C. § %s", title, section)
		if rangeEnd != "" {
			document = fmt.Sprintf("%s U.S.C. §§ %s–%s", title, section, rangeEnd)
		}

		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeCode,
			Jurisdiction: "US",
			Document:     document,
			Confidence:   1.0,
			Parser:       "Bluebook Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				Title:    title,
				Section:  section,
				CodeName: "USC",
			},
		})
	}

	return citations
}

// parseCFRParts extracts C.F.R. citations with multiple parts.
func (parser *BluebookParser) parseCFRParts(text string, matched map[int]bool) []*Citation {
	var citations []*Citation

	for _, matchIndices := range parser.cfrPartsPattern.FindAllStringSubmatchIndex(text, -1) {
		if matched[matchIndices[0]] {
			continue
		}

		rawText := text[matchIndices[0]:matchIndices[1]]
		title := text[matchIndices[2]:matchIndices[3]]
		part1 := text[matchIndices[4]:matchIndices[5]]
		part2 := text[matchIndices[6]:matchIndices[7]]

		matched[matchIndices[0]] = true

		// Emit citations for both parts
		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeCode,
			Jurisdiction: "US",
			Document:     fmt.Sprintf("%s C.F.R. Parts %s and %s", title, part1, part2),
			Confidence:   1.0,
			Parser:       "Bluebook Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				Title:    title,
				Section:  part1,
				CodeName: "CFR",
			},
		})

		// Also emit the second part as a separate citation
		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeCode,
			Jurisdiction: "US",
			Document:     fmt.Sprintf("%s C.F.R. Part %s", title, part2),
			Confidence:   0.9,
			Parser:       "Bluebook Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				Title:    title,
				Section:  part2,
				CodeName: "CFR",
			},
		})
	}

	return citations
}

// parseCFR extracts C.F.R. citations.
func (parser *BluebookParser) parseCFR(text string, matched map[int]bool) []*Citation {
	var citations []*Citation

	for _, matchIndices := range parser.cfrPattern.FindAllStringSubmatchIndex(text, -1) {
		if matched[matchIndices[0]] {
			continue
		}

		rawText := text[matchIndices[0]:matchIndices[1]]
		title := text[matchIndices[2]:matchIndices[3]]
		part := text[matchIndices[4]:matchIndices[5]]

		// Check for subpart (e.g., 164.502)
		var subpart string
		if matchIndices[6] != -1 {
			subpart = text[matchIndices[6]:matchIndices[7]]
		}

		matched[matchIndices[0]] = true

		section := part
		if subpart != "" {
			section = fmt.Sprintf("%s.%s", part, subpart)
		}

		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeCode,
			Jurisdiction: "US",
			Document:     fmt.Sprintf("%s C.F.R. § %s", title, section),
			Confidence:   1.0,
			Parser:       "Bluebook Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				Title:    title,
				Section:  section,
				CodeName: "CFR",
			},
		})
	}

	return citations
}

// parsePublicLaw extracts Public Law citations.
func (parser *BluebookParser) parsePublicLaw(text string, matched map[int]bool) []*Citation {
	var citations []*Citation

	for _, matchIndices := range parser.publicLawPattern.FindAllStringSubmatchIndex(text, -1) {
		if matched[matchIndices[0]] {
			continue
		}

		rawText := text[matchIndices[0]:matchIndices[1]]
		congress := text[matchIndices[2]:matchIndices[3]]
		number := text[matchIndices[4]:matchIndices[5]]

		matched[matchIndices[0]] = true

		publicLawNum := fmt.Sprintf("%s-%s", congress, number)

		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeStatute,
			Jurisdiction: "US",
			Document:     fmt.Sprintf("Pub. L. %s", publicLawNum),
			Confidence:   1.0,
			Parser:       "Bluebook Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				PublicLaw: publicLawNum,
			},
		})
	}

	return citations
}

// parseCases extracts US case citations.
func (parser *BluebookParser) parseCases(text string, matched map[int]bool) []*Citation {
	var citations []*Citation

	for _, matchIndices := range parser.casePattern.FindAllStringSubmatchIndex(text, -1) {
		if matched[matchIndices[0]] {
			continue
		}

		rawText := text[matchIndices[0]:matchIndices[1]]
		plaintiff := text[matchIndices[2]:matchIndices[3]]
		defendant := text[matchIndices[4]:matchIndices[5]]
		volume := text[matchIndices[6]:matchIndices[7]]
		reporter := text[matchIndices[8]:matchIndices[9]]
		page := text[matchIndices[10]:matchIndices[11]]
		year := text[matchIndices[12]:matchIndices[13]]

		matched[matchIndices[0]] = true

		caseName := fmt.Sprintf("%s v. %s", plaintiff, defendant)
		// Normalize reporter name
		normalizedReporter := normalizeReporter(reporter)

		citations = append(citations, &Citation{
			RawText:      rawText,
			Type:         CitationTypeCase,
			Jurisdiction: "US",
			Document:     caseName,
			Subdivision:  fmt.Sprintf("%s %s %s (%s)", volume, normalizedReporter, page, year),
			Confidence:   0.9, // Case citations can be complex; slightly lower confidence
			Parser:       "Bluebook Citation Parser",
			TextOffset:   matchIndices[0],
			TextLength:   matchIndices[1] - matchIndices[0],
			Components: CitationComponents{
				DocNumber: fmt.Sprintf("%s/%s/%s", volume, normalizedReporter, page),
				DocYear:   year,
			},
		})
	}

	return citations
}

// normalizeReporter normalizes reporter abbreviations to standard form.
func normalizeReporter(reporter string) string {
	reporter = strings.TrimSpace(reporter)

	// Normalize common reporters
	switch {
	case strings.Contains(reporter, "U.S"):
		return "U.S."
	case strings.Contains(reporter, "S.") && strings.Contains(reporter, "Ct"):
		return "S. Ct."
	case strings.HasPrefix(reporter, "F.") && strings.Contains(reporter, "Supp"):
		return "F. Supp."
	case strings.HasPrefix(reporter, "F."):
		// F.2d, F.3d, etc.
		return reporter
	default:
		return reporter
	}
}
