// Package extract provides document parsing and structure extraction for regulatory texts.
package extract

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// DocumentType represents the type of regulatory document.
type DocumentType string

const (
	DocumentTypeRegulation DocumentType = "regulation"
	DocumentTypeDirective  DocumentType = "directive"
	DocumentTypeDecision   DocumentType = "decision"
	DocumentTypeUnknown    DocumentType = "unknown"
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
	chapterPattern    *regexp.Regexp
	sectionPattern    *regexp.Regexp
	articlePattern    *regexp.Regexp
	paragraphPattern  *regexp.Regexp
	pointPattern      *regexp.Regexp
	recitalPattern    *regexp.Regexp
	definitionPattern *regexp.Regexp
}

// NewParser creates a new Parser with default patterns for EU regulations.
func NewParser() *Parser {
	return &Parser{
		chapterPattern:    regexp.MustCompile(`^CHAPTER\s+([IVX]+)$`),
		sectionPattern:    regexp.MustCompile(`^Section\s+(\d+)$`),
		articlePattern:    regexp.MustCompile(`^Article\s+(\d+)$`),
		paragraphPattern:  regexp.MustCompile(`^(\d+)\.\s+(.*)$`),
		pointPattern:      regexp.MustCompile(`^\(([a-z])\)\s+(.*)$`),
		recitalPattern:    regexp.MustCompile(`^\((\d+)\)\s+(.*)$`),
		definitionPattern: regexp.MustCompile(`^\((\d+)\)\s+['''"\x{2018}\x{2019}]([^'''"\x{2018}\x{2019}]+)['''"\x{2018}\x{2019}].*means`),
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

	// Parse title and type from first lines
	if len(lines) > 0 {
		doc.Title = lines[0]
		if strings.Contains(strings.ToUpper(lines[0]), "REGULATION") {
			doc.Type = DocumentTypeRegulation
		} else if strings.Contains(strings.ToUpper(lines[0]), "DIRECTIVE") {
			doc.Type = DocumentTypeDirective
		} else if strings.Contains(strings.ToUpper(lines[0]), "DECISION") {
			doc.Type = DocumentTypeDecision
		}
	}

	// Find identifier (e.g., "(EU) 2016/679")
	for i := 0; i < min(10, len(lines)); i++ {
		if strings.Contains(lines[i], "(EU)") || strings.Contains(lines[i], "(EC)") {
			// Extract the identifier
			idPattern := regexp.MustCompile(`\(E[UC]\)\s*(?:No\s*)?(\d+/\d+)`)
			if m := idPattern.FindStringSubmatch(lines[i]); m != nil {
				doc.Identifier = fmt.Sprintf("(EU) %s", m[1])
			}
		}
	}

	// Find where main body starts (after "HAVE ADOPTED THIS REGULATION:")
	mainBodyStart := 0
	for i, line := range lines {
		if strings.Contains(line, "HAVE ADOPTED THIS") {
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

	return doc, nil
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

// parseMainBody parses chapters, sections, and articles from the main body.
func (p *Parser) parseMainBody(doc *Document, lines []string) {
	var currentChapter *Chapter
	var currentSection *Section
	var currentArticle *Article
	var articleText strings.Builder

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check for chapter
		if m := p.chapterPattern.FindStringSubmatch(line); m != nil {
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
		if m := p.sectionPattern.FindStringSubmatch(line); m != nil {
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
		if m := p.articlePattern.FindStringSubmatch(line); m != nil {
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
				if p.articlePattern.MatchString(lines[j]) ||
					p.sectionPattern.MatchString(lines[j]) ||
					p.chapterPattern.MatchString(lines[j]) {
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
