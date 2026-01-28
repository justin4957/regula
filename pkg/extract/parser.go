// Package extract provides document parsing and structure extraction for regulatory texts.
package extract

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/coolbeans/regula/pkg/pattern"
)

// DocumentType represents the type of regulatory document.
type DocumentType string

const (
	DocumentTypeRegulation DocumentType = "regulation"
	DocumentTypeDirective  DocumentType = "directive"
	DocumentTypeDecision   DocumentType = "decision"
	DocumentTypeStatute    DocumentType = "statute"
	DocumentTypeAct        DocumentType = "act"
	DocumentTypeUnknown    DocumentType = "unknown"
)

// DocumentFormat represents the structural format of a regulatory document.
type DocumentFormat string

const (
	FormatEU      DocumentFormat = "eu"      // EU-style: CHAPTER I, Article 1
	FormatUS      DocumentFormat = "us"      // US-style: CHAPTER 1, Section 1798.100
	FormatUK      DocumentFormat = "uk"      // UK-style: PART 1, numbered sections
	FormatGeneric DocumentFormat = "generic" // Inferred from whitespace/numbering patterns
	FormatUnknown DocumentFormat = "unknown"
)

// Document represents a parsed regulatory document.
type Document struct {
	Title       string        `json:"title"`
	Type        DocumentType  `json:"type"`
	Identifier  string        `json:"identifier"`
	Preamble    *Preamble     `json:"preamble,omitempty"`
	Chapters    []*Chapter    `json:"chapters"`
	Definitions []*Definition `json:"definitions,omitempty"`
}

// Preamble represents the preamble section of a regulation.
type Preamble struct {
	Citations []string   `json:"citations,omitempty"`
	Recitals  []*Recital `json:"recitals"`
}

// Recital represents a numbered recital in the preamble.
type Recital struct {
	Number int    `json:"number"`
	Text   string `json:"text"`
}

// Chapter represents a chapter in a regulatory document.
type Chapter struct {
	Number   string     `json:"number"`
	Title    string     `json:"title"`
	Sections []*Section `json:"sections,omitempty"`
	Articles []*Article `json:"articles,omitempty"`
}

// Section represents a section within a chapter.
type Section struct {
	Number   int        `json:"number"`
	Title    string     `json:"title"`
	Articles []*Article `json:"articles"`
}

// Article represents an article in a regulatory document.
type Article struct {
	Number     int          `json:"number"`
	Title      string       `json:"title"`
	Paragraphs []*Paragraph `json:"paragraphs,omitempty"`
	Text       string       `json:"text,omitempty"`
}

// Paragraph represents a numbered paragraph within an article.
type Paragraph struct {
	Number int      `json:"number"`
	Text   string   `json:"text"`
	Points []*Point `json:"points,omitempty"`
}

// Point represents a lettered point within a paragraph (a), (b), etc.
type Point struct {
	Letter    string      `json:"letter"`
	Text      string      `json:"text"`
	SubPoints []*SubPoint `json:"sub_points,omitempty"`
}

// Definition represents a defined term from Article 4 or similar.
type Definition struct {
	Number int    `json:"number"`
	Term   string `json:"term"`
	Text   string `json:"text,omitempty"`
}

// Parser parses regulatory documents into structured form.
type Parser struct {
	// EU-style patterns
	euChapterPattern *regexp.Regexp
	euSectionPattern *regexp.Regexp
	euArticlePattern *regexp.Regexp

	// US-style patterns (California Civil Code style)
	usChapterPattern    *regexp.Regexp
	usArticlePattern    *regexp.Regexp
	usSectionPattern    *regexp.Regexp
	usSectionNumPattern *regexp.Regexp

	// Virginia Code style patterns (Section 59.1-575)
	vaSectionPattern *regexp.Regexp

	// Colorado/Utah hyphenated style patterns (Section 6-1-1301, Section 13-61-101)
	coHyphenatedSectionPattern *regexp.Regexp

	// Iowa alphanumeric style patterns (Section 715D.1)
	ioAlphanumericSectionPattern *regexp.Regexp

	// UK-style patterns (Acts and Statutory Instruments)
	ukPartPattern       *regexp.Regexp
	ukSectionPattern    *regexp.Regexp
	ukSchedulePattern   *regexp.Regexp
	ukDefinitionPattern *regexp.Regexp

	// Common patterns
	paragraphPattern    *regexp.Regexp
	pointPattern        *regexp.Regexp
	recitalPattern      *regexp.Regexp
	definitionPattern   *regexp.Regexp
	usDefinitionPattern *regexp.Regexp

	// Detected format
	format DocumentFormat

	// Pattern library integration (optional)
	patternRegistry pattern.Registry
	patternBridge   *pattern.PatternBridge
}

// NewParser creates a new Parser with patterns for multiple regulation formats.
func NewParser() *Parser {
	return &Parser{
		// EU-style patterns (GDPR, etc.)
		euChapterPattern: regexp.MustCompile(`^CHAPTER\s+([IVX]+)$`),
		euSectionPattern: regexp.MustCompile(`^Section\s+(\d+)$`),
		euArticlePattern: regexp.MustCompile(`^Article\s+(\d+)$`),

		// US-style patterns (CCPA, California Civil Code, etc.)
		usChapterPattern:    regexp.MustCompile(`^CHAPTER\s+(\d+)$`),
		usArticlePattern:    regexp.MustCompile(`^Article\s+(\d+)$`),
		usSectionPattern:    regexp.MustCompile(`^Section\s+(\d+(?:\.\d+)*)$`),
		usSectionNumPattern: regexp.MustCompile(`^Section\s+(\d+)\.(\d+)$`),

		// Virginia Code style: Section 59.1-575 or § 59.1-575
		vaSectionPattern: regexp.MustCompile(`^(?:Section|§)\s*(\d+\.\d+)-(\d+)\.?$`),

		// Colorado/Utah hyphenated: Section 6-1-1301 or Section 13-61-101
		coHyphenatedSectionPattern: regexp.MustCompile(`^(?:Section|§)\s*(\d+)-(\d+)-(\d+)\.?$`),

		// Iowa alphanumeric: Section 715D.1
		ioAlphanumericSectionPattern: regexp.MustCompile(`^(?:Section|§)\s*(\d+[A-Z])\.(\d+)$`),

		// UK-style patterns (Acts and Statutory Instruments)
		ukPartPattern:       regexp.MustCompile(`^PART\s+(\d+)\s*$`),
		ukSectionPattern:    regexp.MustCompile(`^(\d+)\.\s*[-—]?\s*(.+)$`),
		ukSchedulePattern:   regexp.MustCompile(`^SCHEDULE\s+(\d+)\s*$`),
		ukDefinitionPattern: regexp.MustCompile(`(?m)^(?:\(\d+\)\s+)?[\x{201c}\x{201d}""]([^\x{201c}\x{201d}""]+)[\x{201c}\x{201d}""]\s+(?:means?|has\s+the\s+(?:same\s+)?meaning)`),

		// Common patterns
		paragraphPattern:    regexp.MustCompile(`^(\d+)\.\s+(.*)$`),
		pointPattern:        regexp.MustCompile(`^\(([a-z])\)\s+(.*)$`),
		recitalPattern:      regexp.MustCompile(`^\((\d+)\)\s+(.*)$`),
		definitionPattern:   regexp.MustCompile(`^\((\d+)\)\s+['''"\x{2018}\x{2019}]([^'''"\x{2018}\x{2019}]+)['''"\x{2018}\x{2019}].*means`),
		usDefinitionPattern: regexp.MustCompile(`^\(([a-z])\)\s+['''"\x{2018}\x{2019}]([^'''"\x{2018}\x{2019}]+)['''"\x{2018}\x{2019}]\s+means`),

		format: FormatUnknown,
	}
}

// NewParserWithRegistry creates a new Parser that uses the pattern registry
// for format detection and structure extraction. The registry patterns are used
// to drive both EU and US format parsing when a matching pattern is found.
// Falls back to hardcoded patterns when no registry match is found.
func NewParserWithRegistry(registry pattern.Registry) *Parser {
	parser := NewParser()
	parser.patternRegistry = registry
	return parser
}

// applyPatternBridge configures the parser to use compiled patterns from the
// pattern bridge for EU document parsing. This replaces the hardcoded EU regex
// patterns with the ones loaded from the YAML pattern library.
func (p *Parser) applyPatternBridge(bridge *pattern.PatternBridge) {
	if bridge == nil {
		return
	}
	p.patternBridge = bridge

	// Override EU hierarchy patterns from the pattern library
	if chapterPattern := bridge.HierarchyPattern("chapter"); chapterPattern != nil {
		p.euChapterPattern = chapterPattern
	}
	if sectionPattern := bridge.HierarchyPattern("section"); sectionPattern != nil {
		p.euSectionPattern = sectionPattern
	}
	if articlePattern := bridge.HierarchyPattern("article"); articlePattern != nil {
		p.euArticlePattern = articlePattern
	}

	// Override definition pattern from pattern library
	if defPattern := bridge.DefinitionPattern(); defPattern != nil {
		p.definitionPattern = defPattern
	}

	// Override recital pattern from pattern library
	if recitalPattern := bridge.RecitalPattern(); recitalPattern != nil {
		p.recitalPattern = recitalPattern
	}
}

// applyUSPatternBridge configures the parser to use compiled patterns from the
// pattern bridge for US document parsing. This replaces the hardcoded US regex
// patterns with the ones loaded from the YAML pattern library.
func (p *Parser) applyUSPatternBridge(bridge *pattern.PatternBridge) {
	if bridge == nil {
		return
	}
	p.patternBridge = bridge

	// Override US hierarchy patterns from the pattern library
	if chapterPattern := bridge.HierarchyPattern("chapter"); chapterPattern != nil {
		p.usChapterPattern = chapterPattern
	}
	// Some states (Colorado, Utah) use "part" instead of "chapter" as the
	// top-level structural division. Map it to usChapterPattern when present.
	if partPattern := bridge.HierarchyPattern("part"); partPattern != nil {
		p.usChapterPattern = partPattern
	}
	if articlePattern := bridge.HierarchyPattern("article"); articlePattern != nil {
		p.usArticlePattern = articlePattern
	}
	if sectionPattern := bridge.HierarchyPattern("section"); sectionPattern != nil {
		// The section pattern varies by jurisdiction:
		// California: Section 1798.100 (two capture groups: prefix.number)
		// Virginia: Section 59.1-575 (two capture groups: prefix-number)
		jurisdiction := bridge.Jurisdiction()
		switch jurisdiction {
		case "US-VA", "US-CT":
			// Two-segment hyphenated: Section 59.1-575, Section 42-515
			p.vaSectionPattern = sectionPattern
		case "US-CO", "US-UT":
			// Three-segment hyphenated: Section 6-1-1301, Section 13-61-101
			p.coHyphenatedSectionPattern = sectionPattern
		case "US-IA":
			// Alphanumeric dotted: Section 715D.1
			p.ioAlphanumericSectionPattern = sectionPattern
		default:
			// California/Texas dotted: Section 1798.100, Section 541.001
			p.usSectionNumPattern = sectionPattern
		}
	}

	// Override US definition pattern from pattern library
	if defPattern := bridge.DefinitionPattern(); defPattern != nil {
		p.usDefinitionPattern = defPattern
	}
}

// applyUKPatternBridge configures the parser to use compiled patterns from the
// pattern bridge for UK document parsing. This replaces the hardcoded UK regex
// patterns with the ones loaded from the YAML pattern library.
func (p *Parser) applyUKPatternBridge(bridge *pattern.PatternBridge) {
	if bridge == nil {
		return
	}
	p.patternBridge = bridge

	// Override UK hierarchy patterns from the pattern library
	if partPattern := bridge.HierarchyPattern("part"); partPattern != nil {
		p.ukPartPattern = partPattern
	}
	if sectionPattern := bridge.HierarchyPattern("section"); sectionPattern != nil {
		p.ukSectionPattern = sectionPattern
	}
	if schedulePattern := bridge.HierarchyPattern("schedule"); schedulePattern != nil {
		p.ukSchedulePattern = schedulePattern
	}

	// Override UK definition pattern from pattern library
	if defPattern := bridge.DefinitionPattern(); defPattern != nil {
		p.ukDefinitionPattern = defPattern
	}
}

// Parse parses the regulatory text from a reader and returns a structured Document.
func (p *Parser) Parse(r io.Reader) (*Document, error) {
	scanner := bufio.NewScanner(r)

	// Read all lines
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}

	doc := &Document{
		Type:     DocumentTypeUnknown,
		Chapters: make([]*Chapter, 0),
	}

	// Detect document format and type from content
	p.format = p.detectFormat(lines)

	// Parse title and type from first lines
	if len(lines) > 0 {
		doc.Title = lines[0]
		doc.Type = p.detectDocumentType(lines)
	}

	// Find identifier based on format
	doc.Identifier = p.extractIdentifier(lines)

	// Parse based on detected format
	switch p.format {
	case FormatUS:
		p.parseUSDocument(doc, lines)
	case FormatUK:
		p.parseUKDocument(doc, lines)
	case FormatGeneric:
		p.parseGenericDocument(doc, lines)
	default:
		// EU format (default)
		p.parseEUDocument(doc, lines)
	}

	return doc, nil
}

// detectFormat analyzes the document to determine its structural format.
// When a pattern registry is available, it uses the registry's confidence-based
// detection. Otherwise, it falls back to the hardcoded indicator counting.
func (p *Parser) detectFormat(lines []string) DocumentFormat {
	// Try pattern-registry-based detection first
	if p.patternRegistry != nil {
		content := strings.Join(lines, "\n")

		// Check if any pattern matches above threshold. If not, use
		// generic parsing (hierarchy inferred from whitespace/numbering).
		detector := pattern.NewFormatDetector(p.patternRegistry)
		matches := detector.DetectWithThreshold(content, 0.3)
		if pattern.ShouldUseGenericParser(matches, 0.3) {
			return FormatGeneric
		}

		bridge := pattern.DetectAndBridge(p.patternRegistry, content, 0.3)
		if bridge != nil {
			// Map the detected format to our internal format type
			switch bridge.Jurisdiction() {
			case "EU":
				p.applyPatternBridge(bridge)
				return FormatEU
			case "US-CA", "US-VA", "US-CO", "US-CT", "US-UT", "US-IA", "US-TX":
				p.applyUSPatternBridge(bridge)
				return FormatUS
			case "US", "US-Federal":
				p.applyUSPatternBridge(bridge)
				return FormatUS
			case "GB", "GB-SCT":
				p.applyUKPatternBridge(bridge)
				return FormatUK
			}
		}
	}

	// Fall back to hardcoded indicator counting
	return p.detectFormatLegacy(lines)
}

// detectFormatLegacy uses the original hardcoded indicator counting to
// determine document format. This is the pre-pattern-library detection method.
func (p *Parser) detectFormatLegacy(lines []string) DocumentFormat {
	euIndicators := 0
	usIndicators := 0
	ukIndicators := 0

	for _, line := range lines {
		upper := strings.ToUpper(strings.TrimSpace(line))

		// EU indicators
		if p.euChapterPattern.MatchString(line) {
			euIndicators += 2
		}
		if p.euArticlePattern.MatchString(line) {
			euIndicators++
		}
		if strings.Contains(line, "HAVE ADOPTED THIS REGULATION") {
			euIndicators += 3
		}
		if strings.Contains(line, "(EU)") || strings.Contains(line, "(EC)") {
			euIndicators += 2
		}

		// US indicators
		if p.usChapterPattern.MatchString(line) {
			usIndicators += 2
		}
		if p.usSectionNumPattern.MatchString(line) {
			usIndicators += 2
		}
		if strings.Contains(upper, "CALIFORNIA") {
			usIndicators += 2
		}
		if strings.Contains(upper, "VIRGINIA") {
			usIndicators += 2
		}
		if strings.Contains(line, "TITLE 1.81") || strings.Contains(line, "Section 1798") {
			usIndicators += 3
		}
		// Virginia Code style: Section 59.1-XXX
		if strings.Contains(line, "Section 59.1-") || strings.Contains(line, "§ 59.1-") {
			usIndicators += 3
		}

		// UK indicators
		if strings.Contains(upper, "BE IT ENACTED") {
			ukIndicators += 3
		}
		if strings.Contains(upper, "STATUTORY INSTRUMENT") {
			ukIndicators += 3
		}
		if strings.Contains(upper, "ROYAL ASSENT") {
			ukIndicators += 2
		}
		if strings.Contains(upper, "LORDS SPIRITUAL AND TEMPORAL") {
			ukIndicators += 2
		}
		if strings.Contains(upper, "HOUSE OF COMMONS") {
			ukIndicators += 2
		}
		if p.ukPartPattern.MatchString(strings.TrimSpace(line)) {
			ukIndicators++
		}
		if p.ukSchedulePattern.MatchString(strings.TrimSpace(line)) {
			ukIndicators++
		}
		// Chapter citation: [2018 c. 12]
		if matched, _ := regexp.MatchString(`\[\d{4}\s+c\.\s*\d+\]`, line); matched {
			ukIndicators += 3
		}
		// SI number: S.I. 2018/1234
		if matched, _ := regexp.MatchString(`S\.?I\.?\s+\d{4}/\d+`, line); matched {
			ukIndicators += 3
		}
	}

	// If all indicators are below a minimum threshold, this document
	// doesn't match any known format — use generic inference instead.
	minimumIndicatorThreshold := 2
	maxIndicatorScore := max(euIndicators, usIndicators, ukIndicators)
	if maxIndicatorScore < minimumIndicatorThreshold {
		return FormatGeneric
	}

	if ukIndicators > euIndicators && ukIndicators > usIndicators {
		return FormatUK
	}
	if usIndicators > euIndicators {
		return FormatUS
	}
	return FormatEU
}

// detectDocumentType determines the type of document from its content.
func (p *Parser) detectDocumentType(lines []string) DocumentType {
	for i := 0; i < min(20, len(lines)); i++ {
		upper := strings.ToUpper(lines[i])
		if strings.Contains(upper, "REGULATION") {
			return DocumentTypeRegulation
		}
		if strings.Contains(upper, "DIRECTIVE") {
			return DocumentTypeDirective
		}
		if strings.Contains(upper, "DECISION") {
			return DocumentTypeDecision
		}
		if strings.Contains(upper, "ACT") {
			return DocumentTypeAct
		}
		if strings.Contains(upper, "CODE") || strings.Contains(upper, "STATUTE") {
			return DocumentTypeStatute
		}
	}
	return DocumentTypeUnknown
}

// extractIdentifier extracts the document identifier based on format.
func (p *Parser) extractIdentifier(lines []string) string {
	switch p.format {
	case FormatUK:
		return p.extractUKIdentifier(lines)
	case FormatUS:
		// Check pattern bridge jurisdiction for targeted extraction
		if p.patternBridge != nil {
			switch p.patternBridge.Jurisdiction() {
			case "US-CO":
				for i := 0; i < min(20, len(lines)); i++ {
					coSectionPattern := regexp.MustCompile(`(?:Section|§)\s*(\d+-\d+-\d+)`)
					if m := coSectionPattern.FindStringSubmatch(lines[i]); m != nil {
						return fmt.Sprintf("C.R.S. § %s", m[1])
					}
					if strings.Contains(lines[i], "C.R.S.") {
						return "C.R.S. § 6-1-1301 et seq."
					}
				}
			case "US-CT":
				for i := 0; i < min(20, len(lines)); i++ {
					ctSectionPattern := regexp.MustCompile(`(?i)(?:Section|Sec\.|§)\s*(\d+-\d+)`)
					if m := ctSectionPattern.FindStringSubmatch(lines[i]); m != nil {
						return fmt.Sprintf("Conn. Gen. Stat. § %s", m[1])
					}
					if strings.Contains(lines[i], "Conn. Gen. Stat.") || strings.Contains(lines[i], "CGS") {
						return "Conn. Gen. Stat. § 42-515 et seq."
					}
				}
			case "US-TX":
				for i := 0; i < min(20, len(lines)); i++ {
					txSectionPattern := regexp.MustCompile(`(?i)(?:Section|Sec\.|§)\s*(\d+\.\d+)`)
					if m := txSectionPattern.FindStringSubmatch(lines[i]); m != nil {
						return fmt.Sprintf("Tex. Bus. & Com. Code § %s", m[1])
					}
					if strings.Contains(lines[i], "Tex. Bus.") || strings.Contains(lines[i], "Texas Business") {
						return "Tex. Bus. & Com. Code § 541.001 et seq."
					}
				}
			case "US-UT":
				for i := 0; i < min(20, len(lines)); i++ {
					utSectionPattern := regexp.MustCompile(`(?:Section|§)\s*(\d+-\d+-\d+)`)
					if m := utSectionPattern.FindStringSubmatch(lines[i]); m != nil {
						return fmt.Sprintf("U.C.A. § %s", m[1])
					}
					if strings.Contains(lines[i], "U.C.A.") || strings.Contains(lines[i], "Utah Code") {
						return "U.C.A. § 13-61-101 et seq."
					}
				}
			case "US-IA":
				for i := 0; i < min(20, len(lines)); i++ {
					iaSectionPattern := regexp.MustCompile(`(?:Section|§)\s*(\d+[A-Z]\.\d+)`)
					if m := iaSectionPattern.FindStringSubmatch(lines[i]); m != nil {
						return fmt.Sprintf("Iowa Code § %s", m[1])
					}
					if strings.Contains(lines[i], "Iowa Code") {
						return "Iowa Code § 715D.1 et seq."
					}
				}
			}
		}

		// Look for Virginia Code style identifiers first
		for i := 0; i < min(20, len(lines)); i++ {
			// Virginia Code: Title 59.1 Chapter 53
			if strings.Contains(lines[i], "Title 59.1") || strings.Contains(lines[i], "TITLE 59.1") {
				if strings.Contains(lines[i], "Chapter 53") || strings.Contains(lines[i], "CHAPTER 53") {
					return "Va. Code Ann. § 59.1-575 et seq."
				}
				// Generic Title 59.1
				titlePattern := regexp.MustCompile(`(?i)Title\s+59\.1\s+Chapter\s+(\d+)`)
				if m := titlePattern.FindStringSubmatch(lines[i]); m != nil {
					return fmt.Sprintf("Va. Code Ann. Title 59.1 Chapter %s", m[1])
				}
			}
			if strings.Contains(lines[i], "Section 59.1-") || strings.Contains(lines[i], "§ 59.1-") {
				return "Va. Code Ann. § 59.1"
			}
		}
		// Look for California Civil Code style identifiers
		for i := 0; i < min(20, len(lines)); i++ {
			if strings.Contains(lines[i], "TITLE") {
				// Extract title number (e.g., "TITLE 1.81.5")
				titlePattern := regexp.MustCompile(`TITLE\s+([\d.]+)`)
				if m := titlePattern.FindStringSubmatch(lines[i]); m != nil {
					return fmt.Sprintf("Cal. Civ. Code Title %s", m[1])
				}
			}
			if strings.Contains(lines[i], "Section 1798") {
				return "Cal. Civ. Code § 1798"
			}
		}
	default:
		// EU format
		for i := 0; i < min(10, len(lines)); i++ {
			if strings.Contains(lines[i], "(EU)") || strings.Contains(lines[i], "(EC)") {
				idPattern := regexp.MustCompile(`\(E[UC]\)\s*(?:No\s*)?(\d+/\d+)`)
				if m := idPattern.FindStringSubmatch(lines[i]); m != nil {
					return fmt.Sprintf("(EU) %s", m[1])
				}
			}
		}
	}
	return ""
}

// parseEUDocument parses an EU-style document (GDPR, etc.).
func (p *Parser) parseEUDocument(doc *Document, lines []string) {
	// Find where main body starts (after "HAVE ADOPTED THIS REGULATION:")
	mainBodyStart := 0
	endPattern := p.preambleEndPattern()
	for i, line := range lines {
		if endPattern != nil && endPattern.MatchString(line) {
			mainBodyStart = i + 1
			break
		}
		// Fallback string check for backwards compatibility
		if endPattern == nil && strings.Contains(line, "HAVE ADOPTED THIS") {
			mainBodyStart = i + 1
			break
		}
	}

	// Parse preamble (recitals) - everything from "Whereas:" to main body
	doc.Preamble = p.parsePreamble(lines[:mainBodyStart])

	// Parse main body structure
	p.parseMainBody(doc, lines[mainBodyStart:])

	// Extract definitions from Article 4
	doc.Definitions = p.extractDefinitions(doc)
}

// parseUSDocument parses a US-style document (CCPA, California Civil Code, etc.).
func (p *Parser) parseUSDocument(doc *Document, lines []string) {
	var currentChapter *Chapter
	var currentSection *Article // In US format, Sections are treated as Articles
	var sectionText strings.Builder

	// Track section title for next line
	pendingSectionTitle := false
	var pendingSection *Article

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmedLine := strings.TrimSpace(line)

		// Check for CHAPTER
		if m := p.usChapterPattern.FindStringSubmatch(trimmedLine); m != nil {
			// Save previous section
			if currentSection != nil {
				currentSection.Text = strings.TrimSpace(sectionText.String())
				p.addArticle(currentChapter, nil, currentSection)
				currentSection = nil
				sectionText.Reset()
			}

			// Get chapter title (next non-empty line)
			title := ""
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				if strings.TrimSpace(lines[j]) != "" {
					title = strings.TrimSpace(lines[j])
					break
				}
			}

			currentChapter = &Chapter{
				Number:   m[1],
				Title:    title,
				Sections: make([]*Section, 0),
				Articles: make([]*Article, 0),
			}
			doc.Chapters = append(doc.Chapters, currentChapter)
			continue
		}

		// Check for Article header (in US format, these are grouping labels, skip them)
		if p.usArticlePattern.MatchString(trimmedLine) {
			// Save previous section
			if currentSection != nil {
				currentSection.Text = strings.TrimSpace(sectionText.String())
				p.addArticle(currentChapter, nil, currentSection)
				currentSection = nil
				sectionText.Reset()
			}
			// Skip article headers - they just group sections
			continue
		}

		// Check for Section (e.g., "Section 1798.100" or "Section 59.1-575")
		// Try Virginia Code style first: Section 59.1-575
		if m := p.vaSectionPattern.FindStringSubmatch(trimmedLine); m != nil {
			// Save previous section
			if currentSection != nil {
				currentSection.Text = strings.TrimSpace(sectionText.String())
				p.addArticle(currentChapter, nil, currentSection)
			}

			// Parse section number - use the section number after the hyphen
			// e.g., "Section 59.1-575" -> Article 575
			subNum, _ := strconv.Atoi(m[2])

			currentSection = &Article{
				Number: subNum,
				Title:  "", // Will be set from next line
			}
			pendingSectionTitle = true
			pendingSection = currentSection
			sectionText.Reset()
			continue
		}

		// Try Colorado/Utah three-segment hyphenated: Section 6-1-1301 or Section 13-61-101
		if m := p.coHyphenatedSectionPattern.FindStringSubmatch(trimmedLine); m != nil {
			// Save previous section
			if currentSection != nil {
				currentSection.Text = strings.TrimSpace(sectionText.String())
				p.addArticle(currentChapter, nil, currentSection)
			}

			// Use third segment as section number: 6-1-1301 -> 1301
			subNum, _ := strconv.Atoi(m[3])

			currentSection = &Article{
				Number: subNum,
				Title:  "", // Will be set from next line
			}
			pendingSectionTitle = true
			pendingSection = currentSection
			sectionText.Reset()

			// Ensure there's at least a default chapter container
			if currentChapter == nil {
				currentChapter = &Chapter{
					Number:   "1",
					Title:    "",
					Sections: make([]*Section, 0),
					Articles: make([]*Article, 0),
				}
				doc.Chapters = append(doc.Chapters, currentChapter)
			}
			continue
		}

		// Try Iowa alphanumeric dotted: Section 715D.1
		if m := p.ioAlphanumericSectionPattern.FindStringSubmatch(trimmedLine); m != nil {
			// Save previous section
			if currentSection != nil {
				currentSection.Text = strings.TrimSpace(sectionText.String())
				p.addArticle(currentChapter, nil, currentSection)
			}

			// Use numeric suffix as section number: 715D.1 -> 1
			subNum, _ := strconv.Atoi(m[2])

			currentSection = &Article{
				Number: subNum,
				Title:  "", // Will be set from next line
			}
			pendingSectionTitle = true
			pendingSection = currentSection
			sectionText.Reset()

			// Ensure there's at least a default chapter container
			if currentChapter == nil {
				currentChapter = &Chapter{
					Number:   "1",
					Title:    "",
					Sections: make([]*Section, 0),
					Articles: make([]*Article, 0),
				}
				doc.Chapters = append(doc.Chapters, currentChapter)
			}
			continue
		}

		// Try California/Texas style: Section 1798.100 or Section 541.001
		if m := p.usSectionNumPattern.FindStringSubmatch(trimmedLine); m != nil {
			// Save previous section
			if currentSection != nil {
				currentSection.Text = strings.TrimSpace(sectionText.String())
				p.addArticle(currentChapter, nil, currentSection)
			}

			// Parse section number - use the subsection part as the article number
			// e.g., "Section 1798.100" -> Article 100
			subNum, _ := strconv.Atoi(m[2])

			currentSection = &Article{
				Number: subNum,
				Title:  "", // Will be set from next line
			}
			pendingSectionTitle = true
			pendingSection = currentSection
			sectionText.Reset()
			continue
		}

		// Set section title from line after "Section X"
		if pendingSectionTitle && pendingSection != nil && trimmedLine != "" {
			pendingSection.Title = trimmedLine
			pendingSectionTitle = false
			pendingSection = nil
			continue
		}

		// Accumulate section text
		if currentSection != nil && trimmedLine != "" {
			// Skip if this line is the section title
			if currentSection.Title != "" && trimmedLine == currentSection.Title {
				continue
			}
			if sectionText.Len() > 0 {
				sectionText.WriteString("\n")
			}
			sectionText.WriteString(trimmedLine)
		}
	}

	// Save last section
	if currentSection != nil {
		currentSection.Text = strings.TrimSpace(sectionText.String())
		p.addArticle(currentChapter, nil, currentSection)
	}

	// Extract definitions
	doc.Definitions = p.extractUSDefinitions(doc)
}

// extractUKIdentifier extracts an identifier from UK legislation.
// For Acts: chapter citation like [2018 c. 12]
// For SIs: SI number like S.I. 2019/419 or Statutory Instruments 2019 No. 419
func (p *Parser) extractUKIdentifier(lines []string) string {
	chapterCitationPattern := regexp.MustCompile(`\[(\d{4})\s+c\.\s*(\d+)\]`)
	siShortPattern := regexp.MustCompile(`S\.?I\.?\s+(\d{4})/(\d+)`)
	siLongPattern := regexp.MustCompile(`(?i)Statutory\s+Instruments?\s+(\d{4})\s+No\.\s*(\d+)`)

	for i := 0; i < min(30, len(lines)); i++ {
		// Check for chapter citation: [2018 c. 12]
		if m := chapterCitationPattern.FindStringSubmatch(lines[i]); m != nil {
			return fmt.Sprintf("%s c. %s", m[1], m[2])
		}
		// Check for short SI number: S.I. 2019/419
		if m := siShortPattern.FindStringSubmatch(lines[i]); m != nil {
			return fmt.Sprintf("S.I. %s/%s", m[1], m[2])
		}
		// Check for long SI number: Statutory Instruments 2019 No. 419
		if m := siLongPattern.FindStringSubmatch(lines[i]); m != nil {
			return fmt.Sprintf("S.I. %s/%s", m[1], m[2])
		}
	}
	return ""
}

// parseUKDocument parses a UK-style document (Acts of Parliament, Statutory Instruments).
// UK Acts use numbered sections (1, 2, 3...) with inline titles, grouped into Parts.
// UK SIs use numbered regulations with similar structure.
func (p *Parser) parseUKDocument(doc *Document, lines []string) {
	var currentChapter *Chapter // Parts map to Chapters in our model
	var currentArticle *Article // Sections/Regulations map to Articles
	var articleText strings.Builder

	// Track section title for next line (when title_follows is true)
	pendingSectionTitle := false
	var pendingArticle *Article

	// Find where preamble ends and main body starts
	mainBodyStart := 0
	enactingPattern := regexp.MustCompile(`(?i)BE\s+IT\s+ENACTED`)
	madePattern := regexp.MustCompile(`(?i)^Made\s+\d`)

	for i, line := range lines {
		if enactingPattern.MatchString(line) || madePattern.MatchString(line) {
			mainBodyStart = i + 1
			break
		}
	}

	for i := mainBodyStart; i < len(lines); i++ {
		line := lines[i]
		trimmedLine := strings.TrimSpace(line)

		// Check for PART (maps to Chapter in our model)
		if m := p.ukPartPattern.FindStringSubmatch(trimmedLine); m != nil {
			// Save previous article
			if currentArticle != nil {
				currentArticle.Text = strings.TrimSpace(articleText.String())
				p.addArticle(currentChapter, nil, currentArticle)
				currentArticle = nil
				articleText.Reset()
			}

			// Get part title (next non-empty line)
			title := ""
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				if strings.TrimSpace(lines[j]) != "" {
					title = strings.TrimSpace(lines[j])
					break
				}
			}

			currentChapter = &Chapter{
				Number:   m[1],
				Title:    title,
				Sections: make([]*Section, 0),
				Articles: make([]*Article, 0),
			}
			doc.Chapters = append(doc.Chapters, currentChapter)
			continue
		}

		// Check for SCHEDULE (also maps to Chapter)
		if m := p.ukSchedulePattern.FindStringSubmatch(trimmedLine); m != nil {
			// Save previous article
			if currentArticle != nil {
				currentArticle.Text = strings.TrimSpace(articleText.String())
				p.addArticle(currentChapter, nil, currentArticle)
				currentArticle = nil
				articleText.Reset()
			}

			// Get schedule title (next non-empty line)
			title := ""
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				if strings.TrimSpace(lines[j]) != "" {
					title = strings.TrimSpace(lines[j])
					break
				}
			}

			currentChapter = &Chapter{
				Number:   "S" + m[1], // Prefix with S to distinguish from Parts
				Title:    title,
				Sections: make([]*Section, 0),
				Articles: make([]*Article, 0),
			}
			doc.Chapters = append(doc.Chapters, currentChapter)
			continue
		}

		// Check for numbered section (UK Acts: "1 Overview" or "1.—(1) Citation")
		if m := p.ukSectionPattern.FindStringSubmatch(trimmedLine); m != nil {
			// Save previous article
			if currentArticle != nil {
				currentArticle.Text = strings.TrimSpace(articleText.String())
				p.addArticle(currentChapter, nil, currentArticle)
			}

			sectionNum, _ := strconv.Atoi(m[1])

			// Title is inline in group 2 for UK Acts
			sectionTitle := ""
			if len(m) > 2 {
				sectionTitle = strings.TrimSpace(m[2])
				// Remove any leading dash/em-dash followed by content
				sectionTitle = strings.TrimLeft(sectionTitle, "—-")
				sectionTitle = strings.TrimSpace(sectionTitle)
			}

			// If title is empty, look at the next non-empty line
			if sectionTitle == "" {
				pendingSectionTitle = true
			}

			currentArticle = &Article{
				Number: sectionNum,
				Title:  sectionTitle,
			}
			if pendingSectionTitle {
				pendingArticle = currentArticle
			}
			articleText.Reset()

			// Ensure there's at least a default chapter
			if currentChapter == nil {
				currentChapter = &Chapter{
					Number:   "1",
					Title:    "",
					Sections: make([]*Section, 0),
					Articles: make([]*Article, 0),
				}
				doc.Chapters = append(doc.Chapters, currentChapter)
			}
			continue
		}

		// Set section title from line after section header (when title_follows)
		if pendingSectionTitle && pendingArticle != nil && trimmedLine != "" {
			pendingArticle.Title = trimmedLine
			pendingSectionTitle = false
			pendingArticle = nil
			continue
		}

		// Accumulate article text
		if currentArticle != nil && trimmedLine != "" {
			// Skip lines that are the section title
			if currentArticle.Title != "" && trimmedLine == currentArticle.Title {
				continue
			}
			if articleText.Len() > 0 {
				articleText.WriteString("\n")
			}
			articleText.WriteString(trimmedLine)
		}
	}

	// Save last article
	if currentArticle != nil {
		currentArticle.Text = strings.TrimSpace(articleText.String())
		p.addArticle(currentChapter, nil, currentArticle)
	}

	// Extract definitions
	doc.Definitions = p.extractUKDefinitions(doc)
}

// extractUKDefinitions extracts defined terms from UK legislation.
// UK Acts typically define terms in an "Interpretation" section using the pattern:
//
//	"term" means ...
func (p *Parser) extractUKDefinitions(doc *Document) []*Definition {
	definitions := make([]*Definition, 0)

	// Determine definition locations from pattern bridge or search by title
	defArticleNumbers := []int{}
	var defTitleRegexps []*regexp.Regexp

	if p.patternBridge != nil {
		bridgeLocations := p.patternBridge.DefinitionLocations()
		for _, loc := range bridgeLocations {
			if loc.SectionNumber > 0 {
				defArticleNumbers = append(defArticleNumbers, loc.SectionNumber)
			}
			if loc.SectionTitle != "" {
				compiled, err := regexp.Compile(loc.SectionTitle)
				if err == nil {
					defTitleRegexps = append(defTitleRegexps, compiled)
				}
			}
		}
	}

	// Fallback title patterns if bridge doesn't provide any
	if len(defTitleRegexps) == 0 {
		defTitleRegexps = []*regexp.Regexp{
			regexp.MustCompile(`(?i)interpretation|definitions?|terms`),
		}
	}

	// Find the definitions article
	var defArticle *Article
	for _, chapter := range doc.Chapters {
		for _, article := range chapter.Articles {
			// Check by section number
			for _, defNum := range defArticleNumbers {
				if article.Number == defNum {
					defArticle = article
					break
				}
			}
			if defArticle != nil {
				break
			}
			// Check by title pattern
			for _, titleRegexp := range defTitleRegexps {
				if titleRegexp.MatchString(article.Title) {
					defArticle = article
					break
				}
			}
			if defArticle != nil {
				break
			}
		}
		if defArticle != nil {
			break
		}
	}

	if defArticle == nil || defArticle.Text == "" {
		return definitions
	}

	// Parse UK-style definitions: "term" means ...
	allLines := strings.Split(defArticle.Text, "\n")
	defNum := 0
	for _, line := range allLines {
		if m := p.ukDefinitionPattern.FindStringSubmatch(line); m != nil {
			defNum++
			definitions = append(definitions, &Definition{
				Number: defNum,
				Term:   strings.TrimSpace(m[1]),
			})
		}
	}

	return definitions
}

// parseGenericDocument parses a document using the GenericParser for documents
// that don't match any specific format (EU, US, UK). It infers hierarchy from
// whitespace and numbering patterns, then converts the result into the standard
// Document model.
func (p *Parser) parseGenericDocument(doc *Document, lines []string) {
	genericParser := pattern.NewGenericParser()
	content := strings.Join(lines, "\n")

	genericDocument, _ := genericParser.Parse(content)

	// Convert GenericDocument into our Document model
	convertedDocument := convertGenericDocument(genericDocument)

	doc.Chapters = convertedDocument.Chapters
	doc.Definitions = convertedDocument.Definitions

	// Use the generic parser's title detection if it found a better title
	if genericDocument.Title != "" {
		doc.Title = genericDocument.Title
	}
}

// preambleEndPattern returns the compiled preamble end pattern from the
// pattern bridge, or nil if no bridge is available.
func (p *Parser) preambleEndPattern() *regexp.Regexp {
	if p.patternBridge != nil {
		return p.patternBridge.PreambleEndPattern()
	}
	return nil
}

// parsePreamble extracts recitals from the preamble section.
func (p *Parser) parsePreamble(lines []string) *Preamble {
	preamble := &Preamble{
		Recitals: make([]*Recital, 0),
	}

	inRecitals := false
	var currentRecital *Recital
	var recitalText strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "Whereas:") {
			inRecitals = true
			continue
		}

		if !inRecitals {
			continue
		}

		// Check for new recital
		if m := p.recitalPattern.FindStringSubmatch(line); m != nil {
			// Save previous recital
			if currentRecital != nil {
				currentRecital.Text = strings.TrimSpace(recitalText.String())
				preamble.Recitals = append(preamble.Recitals, currentRecital)
			}

			num, _ := strconv.Atoi(m[1])
			currentRecital = &Recital{Number: num}
			recitalText.Reset()
			recitalText.WriteString(m[2])
		} else if currentRecital != nil && line != "" {
			// Continue current recital
			recitalText.WriteString(" ")
			recitalText.WriteString(line)
		}
	}

	// Save last recital
	if currentRecital != nil {
		currentRecital.Text = strings.TrimSpace(recitalText.String())
		preamble.Recitals = append(preamble.Recitals, currentRecital)
	}

	return preamble
}

// parseMainBody parses chapters, sections, and articles from the main body (EU format).
func (p *Parser) parseMainBody(doc *Document, lines []string) {
	var currentChapter *Chapter
	var currentSection *Section
	var currentArticle *Article
	var articleText strings.Builder

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check for chapter
		if m := p.euChapterPattern.FindStringSubmatch(line); m != nil {
			// Save previous article
			if currentArticle != nil {
				currentArticle.Text = strings.TrimSpace(articleText.String())
				p.addArticle(currentChapter, currentSection, currentArticle)
				currentArticle = nil
				articleText.Reset()
			}

			// Get chapter title (next non-empty line)
			title := ""
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				if lines[j] != "" {
					title = lines[j]
					break
				}
			}

			currentChapter = &Chapter{
				Number:   m[1],
				Title:    title,
				Sections: make([]*Section, 0),
				Articles: make([]*Article, 0),
			}
			doc.Chapters = append(doc.Chapters, currentChapter)
			currentSection = nil
			continue
		}

		// Check for section
		if m := p.euSectionPattern.FindStringSubmatch(line); m != nil {
			// Save previous article
			if currentArticle != nil {
				currentArticle.Text = strings.TrimSpace(articleText.String())
				p.addArticle(currentChapter, currentSection, currentArticle)
				currentArticle = nil
				articleText.Reset()
			}

			num, _ := strconv.Atoi(m[1])

			// Get section title (next non-empty line)
			title := ""
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				if lines[j] != "" {
					title = lines[j]
					break
				}
			}

			currentSection = &Section{
				Number:   num,
				Title:    title,
				Articles: make([]*Article, 0),
			}
			if currentChapter != nil {
				currentChapter.Sections = append(currentChapter.Sections, currentSection)
			}
			continue
		}

		// Check for article
		if m := p.euArticlePattern.FindStringSubmatch(line); m != nil {
			// Save previous article
			if currentArticle != nil {
				currentArticle.Text = strings.TrimSpace(articleText.String())
				p.addArticle(currentChapter, currentSection, currentArticle)
			}

			num, _ := strconv.Atoi(m[1])

			// Get article title - collect lines until we hit content or a blank after title
			var titleLines []string
			sawBlankAfterTitle := false
			for j := i + 1; j < len(lines); j++ {
				if lines[j] == "" {
					if len(titleLines) > 0 {
						sawBlankAfterTitle = true
					}
					continue // Skip blank lines
				}
				// Stop when we hit a paragraph number (e.g., "1.   text" or "1.\u00a0\u00a0\u00a0text")
				if startsWithParagraphNumber(lines[j]) {
					break
				}
				// Stop when we hit a definition/point number (e.g., "(1) " or "(a) ")
				if startsWithPointOrDefinition(lines[j]) {
					break
				}
				// Stop when we hit another structural element
				if p.euArticlePattern.MatchString(lines[j]) ||
					p.euSectionPattern.MatchString(lines[j]) ||
					p.euChapterPattern.MatchString(lines[j]) {
					break
				}
				// If we saw a blank after collecting title, this is body text, not title
				if sawBlankAfterTitle {
					break
				}
				titleLines = append(titleLines, lines[j])
			}
			title := strings.Join(titleLines, " ")

			currentArticle = &Article{
				Number: num,
				Title:  title,
			}
			articleText.Reset()
			continue
		}

		// Accumulate article text
		if currentArticle != nil && line != "" {
			// Skip lines that are part of the title
			if !strings.Contains(currentArticle.Title, line) {
				if articleText.Len() > 0 {
					articleText.WriteString("\n")
				}
				articleText.WriteString(line)
			}
		}
	}

	// Save last article
	if currentArticle != nil {
		currentArticle.Text = strings.TrimSpace(articleText.String())
		p.addArticle(currentChapter, currentSection, currentArticle)
	}
}

// addArticle adds an article to the appropriate container (section or chapter).
func (p *Parser) addArticle(chapter *Chapter, section *Section, article *Article) {
	if chapter == nil {
		return
	}
	if section != nil {
		section.Articles = append(section.Articles, article)
	} else {
		chapter.Articles = append(chapter.Articles, article)
	}
}

// extractDefinitions extracts defined terms from Article 4 (Definitions).
func (p *Parser) extractDefinitions(doc *Document) []*Definition {
	definitions := make([]*Definition, 0)

	// Find Article 4
	var article4 *Article
	for _, chapter := range doc.Chapters {
		for _, article := range chapter.Articles {
			if article.Number == 4 {
				article4 = article
				break
			}
		}
		for _, section := range chapter.Sections {
			for _, article := range section.Articles {
				if article.Number == 4 {
					article4 = article
					break
				}
			}
		}
		if article4 != nil {
			break
		}
	}

	if article4 == nil || article4.Text == "" {
		return definitions
	}

	// Parse definitions from article text
	lines := strings.Split(article4.Text, "\n")
	for _, line := range lines {
		if m := p.definitionPattern.FindStringSubmatch(line); m != nil {
			num, _ := strconv.Atoi(m[1])
			definitions = append(definitions, &Definition{
				Number: num,
				Term:   strings.TrimSpace(m[2]),
			})
		}
	}

	return definitions
}

// extractUSDefinitions extracts defined terms from US-style documents (e.g., CCPA Section 1798.110).
func (p *Parser) extractUSDefinitions(doc *Document) []*Definition {
	definitions := make([]*Definition, 0)

	// Determine definition locations from pattern bridge or hardcoded defaults
	defArticleNumbers := []int{110, 575, 1303, 515, 1, 101} // CCPA: 110, VCDPA: 575, CPA: 1303, CTDPA: 515, TDPSA/ICDPA: 1, UCPA: 101
	if p.patternBridge != nil {
		bridgeLocations := p.patternBridge.DefinitionLocations()
		if len(bridgeLocations) > 0 {
			defArticleNumbers = make([]int, 0, len(bridgeLocations))
			for _, loc := range bridgeLocations {
				if loc.SectionNumber > 0 {
					defArticleNumbers = append(defArticleNumbers, loc.SectionNumber)
				}
			}
		}
	}

	var defArticle *Article
	for _, chapter := range doc.Chapters {
		for _, article := range chapter.Articles {
			// Check for definitions section by number or title
			for _, defNum := range defArticleNumbers {
				if article.Number == defNum {
					defArticle = article
					break
				}
			}
			if defArticle != nil {
				break
			}
			if strings.Contains(strings.ToLower(article.Title), "definition") {
				defArticle = article
				break
			}
		}
		if defArticle != nil {
			break
		}
	}

	if defArticle == nil || defArticle.Text == "" {
		return definitions
	}

	// Parse US-style definitions: (a) 'term' means ...
	lines := strings.Split(defArticle.Text, "\n")
	defNum := 0
	for _, line := range lines {
		// Match pattern: (a) 'term' means or (a) "term" means
		if m := p.usDefinitionPattern.FindStringSubmatch(line); m != nil {
			defNum++
			definitions = append(definitions, &Definition{
				Number: defNum,
				Term:   strings.TrimSpace(m[2]),
			})
		}
	}

	return definitions
}

// Statistics returns parsing statistics for validation.
type Statistics struct {
	Chapters    int `json:"chapters"`
	Sections    int `json:"sections"`
	Articles    int `json:"articles"`
	Definitions int `json:"definitions"`
	Recitals    int `json:"recitals"`
}

// Statistics returns statistics about the parsed document.
func (d *Document) Statistics() Statistics {
	stats := Statistics{}

	if d.Preamble != nil {
		stats.Recitals = len(d.Preamble.Recitals)
	}

	stats.Chapters = len(d.Chapters)

	for _, chapter := range d.Chapters {
		stats.Sections += len(chapter.Sections)
		stats.Articles += len(chapter.Articles)

		for _, section := range chapter.Sections {
			stats.Articles += len(section.Articles)
		}
	}

	stats.Definitions = len(d.Definitions)

	return stats
}

// GetArticle returns an article by number, or nil if not found.
func (d *Document) GetArticle(number int) *Article {
	for _, chapter := range d.Chapters {
		for _, article := range chapter.Articles {
			if article.Number == number {
				return article
			}
		}
		for _, section := range chapter.Sections {
			for _, article := range section.Articles {
				if article.Number == number {
					return article
				}
			}
		}
	}
	return nil
}

// GetChapter returns a chapter by roman numeral, or nil if not found.
func (d *Document) GetChapter(number string) *Chapter {
	for _, chapter := range d.Chapters {
		if chapter.Number == number {
			return chapter
		}
	}
	return nil
}

// AllArticles returns all articles in document order.
func (d *Document) AllArticles() []*Article {
	var articles []*Article
	for _, chapter := range d.Chapters {
		articles = append(articles, chapter.Articles...)
		for _, section := range chapter.Sections {
			articles = append(articles, section.Articles...)
		}
	}
	return articles
}

// startsWithPointOrDefinition checks if a line starts with a point or definition number
// like "(1) " or "(a) ".
func startsWithPointOrDefinition(line string) bool {
	if len(line) < 4 {
		return false
	}
	if line[0] != '(' {
		return false
	}
	// Check for (digit) or (letter) followed by space
	closeIdx := strings.Index(line, ")")
	if closeIdx < 2 || closeIdx > 4 { // e.g., "(1)" or "(26)" or "(a)"
		return false
	}
	if closeIdx+1 >= len(line) {
		return false
	}
	// Should have space after closing paren
	return line[closeIdx+1] == ' '
}

// startsWithParagraphNumber checks if a line starts with a paragraph number
// like "1.   " or "1.\u00a0\u00a0\u00a0" (with non-breaking spaces).
func startsWithParagraphNumber(line string) bool {
	if len(line) < 3 {
		return false
	}
	// Check for digit followed by period followed by whitespace
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i == 0 || i >= len(line) {
		return false
	}
	if line[i] != '.' {
		return false
	}
	i++
	if i >= len(line) {
		return false
	}
	// Check for at least 2 whitespace chars (regular space or non-breaking space \u00a0)
	whitespaceCount := 0
	for i < len(line) {
		if line[i] == ' ' || line[i] == '\u00a0' {
			whitespaceCount++
			i++
		} else if len(line) >= i+2 && line[i:i+2] == "\u00a0" {
			// UTF-8 encoding of non-breaking space
			whitespaceCount++
			i += 2
		} else {
			break
		}
	}
	return whitespaceCount >= 2
}
