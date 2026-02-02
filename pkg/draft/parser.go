package draft

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// Parser extracts structured bill data from plain-text Congressional
// legislation. It uses compiled regular expressions to identify header
// elements (Congress, session, bill number), the enacting clause, and
// section boundaries.
type Parser struct {
	congressPattern    *regexp.Regexp
	sessionPattern     *regexp.Regexp
	billNumberPattern  *regexp.Regexp
	titlePattern       *regexp.Regexp
	enactingPattern    *regexp.Regexp
	sectionFullPattern *regexp.Regexp
	sectionAbbrPattern *regexp.Regexp
	shortTitlePattern  *regexp.Regexp
}

// NewParser creates a Parser with all regex patterns compiled. The parser
// is safe for concurrent use across multiple goroutines since it only
// reads from its compiled patterns.
func NewParser() *Parser {
	return &Parser{
		congressPattern:    regexp.MustCompile(`(?i)^(\d{1,3}(?:st|nd|rd|th))\s+CONGRESS\s*$`),
		sessionPattern:     regexp.MustCompile(`(?i)^(\d(?:st|nd|rd|th))\s+SESSION\s*$`),
		billNumberPattern:  regexp.MustCompile(`^\s*(H\.\s*R\.\s*\d+|S\.\s*\d+)\s*$`),
		titlePattern:       regexp.MustCompile(`(?i)^(AN\s+ACT|A\s+BILL)\s*$`),
		enactingPattern:    regexp.MustCompile(`(?i)^Be\s+it\s+enacted`),
		sectionFullPattern: regexp.MustCompile(`^SECTION\s+(\d+)\.\s+(.+)$`),
		sectionAbbrPattern: regexp.MustCompile(`^SEC\.\s+(\d+)\.\s+(.+)$`),
		shortTitlePattern:  regexp.MustCompile(`(?i)may\s+be\s+cited\s+as\s+the\s+["\x{201c}]([^"\x{201d}]+)["\x{201d}]`),
	}
}

// Parse reads a Congressional bill from the given reader and returns a
// structured DraftBill. It extracts header metadata (Congress, session,
// bill number, title), splits the body into sections, and detects the
// short title from section 1 when present.
//
// The parser initializes all Amendments slices as empty (non-nil).
// Amendment detection is handled separately by the pattern recognizer
// (issue #149).
func (parser *Parser) Parse(reader io.Reader) (*DraftBill, error) {
	scanner := bufio.NewScanner(reader)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("reading input: %w", scanErr)
	}

	if len(lines) == 0 {
		return nil, fmt.Errorf("empty input: no lines to parse")
	}

	rawText := strings.Join(lines, "\n")

	congress, session, billNumber, title, headerEndIndex := parser.parseHeader(lines)

	if billNumber == "" {
		return nil, fmt.Errorf("no bill number found (expected H.R. or S. designation)")
	}

	sections := parser.parseSections(lines, headerEndIndex)

	shortTitle := ""
	if len(sections) > 0 {
		shortTitle = parser.extractShortTitle(sections[0].RawText)
	}

	bill := &DraftBill{
		Title:      title,
		ShortTitle: shortTitle,
		BillNumber: billNumber,
		Congress:   congress,
		Session:    session,
		Sections:   sections,
		RawText:    rawText,
	}

	return bill, nil
}

// ParseBill parses a Congressional bill from a string.
func ParseBill(text string) (*DraftBill, error) {
	parser := NewParser()
	return parser.Parse(strings.NewReader(text))
}

// ParseBillFromFile parses a Congressional bill from a file path.
func ParseBillFromFile(path string) (*DraftBill, error) {
	file, openErr := os.Open(path)
	if openErr != nil {
		return nil, fmt.Errorf("opening file %s: %w", path, openErr)
	}
	defer file.Close()

	parser := NewParser()
	bill, parseErr := parser.Parse(file)
	if parseErr != nil {
		return nil, parseErr
	}
	bill.Filename = path
	return bill, nil
}

// parseHeader scans lines from the top of the bill, extracting Congress,
// session, bill number, and title. It returns the index of the first line
// after the enacting clause (or after the last header line if no enacting
// clause is found).
func (parser *Parser) parseHeader(lines []string) (congress, session, billNumber, title string, headerEndIndex int) {
	titleLines := []string{}
	inTitle := false
	headerEndIndex = len(lines)

	for lineIndex, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == "" {
			if inTitle {
				// Blank line after title content ends the title block
				inTitle = false
			}
			continue
		}

		if congressMatch := parser.congressPattern.FindStringSubmatch(trimmedLine); congressMatch != nil {
			congress = congressMatch[1]
			continue
		}

		if sessionMatch := parser.sessionPattern.FindStringSubmatch(trimmedLine); sessionMatch != nil {
			session = sessionMatch[1]
			continue
		}

		if billNumberMatch := parser.billNumberPattern.FindStringSubmatch(trimmedLine); billNumberMatch != nil {
			billNumber = normalizeBillNumber(billNumberMatch[1])
			continue
		}

		if parser.titlePattern.MatchString(trimmedLine) {
			inTitle = true
			continue
		}

		if parser.enactingPattern.MatchString(trimmedLine) {
			// Skip the enacting clause line and any continuation lines
			headerEndIndex = lineIndex + 1
			for headerEndIndex < len(lines) {
				nextLine := strings.TrimSpace(lines[headerEndIndex])
				if nextLine == "" {
					headerEndIndex++
					break
				}
				headerEndIndex++
			}
			break
		}

		if inTitle {
			titleLines = append(titleLines, trimmedLine)
		}
	}

	title = strings.Join(titleLines, " ")
	return
}

// parseSections splits lines starting from headerEndIndex into sections
// based on "SECTION N." and "SEC. N." boundaries. Each section captures
// its number, title, and raw text content.
func (parser *Parser) parseSections(lines []string, headerEndIndex int) []*DraftSection {
	var sections []*DraftSection
	var currentSection *DraftSection
	var currentLines []string

	flushCurrentSection := func() {
		if currentSection != nil {
			currentSection.RawText = strings.TrimSpace(strings.Join(currentLines, "\n"))
			currentSection.Amendments = []Amendment{}
			sections = append(sections, currentSection)
		}
	}

	for lineIndex := headerEndIndex; lineIndex < len(lines); lineIndex++ {
		line := lines[lineIndex]
		trimmedLine := strings.TrimSpace(line)

		sectionNumber, sectionTitle := parser.matchSectionHeader(trimmedLine)
		if sectionNumber != "" {
			flushCurrentSection()
			currentSection = &DraftSection{
				Number: sectionNumber,
				Title:  sectionTitle,
			}
			currentLines = []string{}
			continue
		}

		if currentSection != nil {
			currentLines = append(currentLines, line)
		}
	}

	flushCurrentSection()
	return sections
}

// matchSectionHeader checks if a line matches either the full "SECTION N."
// or abbreviated "SEC. N." pattern and returns the section number and title.
func (parser *Parser) matchSectionHeader(line string) (number, title string) {
	if fullMatch := parser.sectionFullPattern.FindStringSubmatch(line); fullMatch != nil {
		return fullMatch[1], strings.TrimRight(fullMatch[2], ".")
	}
	if abbrMatch := parser.sectionAbbrPattern.FindStringSubmatch(line); abbrMatch != nil {
		return abbrMatch[1], strings.TrimRight(abbrMatch[2], ".")
	}
	return "", ""
}

// extractShortTitle looks for the "may be cited as" pattern in section text
// and returns the quoted title if found. Whitespace within the title is
// normalized since quoted titles may span multiple lines in the source.
func (parser *Parser) extractShortTitle(sectionText string) string {
	// Normalize whitespace so the regex can match across line breaks
	normalizedText := strings.Join(strings.Fields(sectionText), " ")
	shortTitleMatch := parser.shortTitlePattern.FindStringSubmatch(normalizedText)
	if shortTitleMatch != nil {
		return shortTitleMatch[1]
	}
	return ""
}

// normalizeBillNumber removes extra whitespace from bill number strings
// (e.g., "H. R. 1234" becomes "H.R. 1234").
func normalizeBillNumber(raw string) string {
	normalized := strings.ReplaceAll(raw, " ", "")
	// Re-add space after the final period before the number
	if strings.HasPrefix(normalized, "H.R.") {
		normalized = "H.R. " + strings.TrimPrefix(normalized, "H.R.")
	} else if strings.HasPrefix(normalized, "S.") {
		normalized = "S. " + strings.TrimPrefix(normalized, "S.")
	}
	return normalized
}
