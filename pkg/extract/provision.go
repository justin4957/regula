package extract

import (
	"regexp"
	"strings"
)

// SubPoint represents a sub-point within a point, using roman numerals (i), (ii), etc.
type SubPoint struct {
	Numeral string `json:"numeral"`
	Text    string `json:"text"`
}

// ProvisionExtractor extracts structured content from article text.
type ProvisionExtractor struct {
	paragraphPattern *regexp.Regexp
	pointPattern     *regexp.Regexp
	subPointPattern  *regexp.Regexp
}

// NewProvisionExtractor creates a new ProvisionExtractor with default patterns.
func NewProvisionExtractor() *ProvisionExtractor {
	return &ProvisionExtractor{
		// Matches "1.   " at start of line (paragraph numbers with multiple spaces or non-breaking spaces)
		paragraphPattern: regexp.MustCompile(`^(\d+)\.[\s\x{00A0}]{2,}(.*)$`),
		// Matches "(a) " at start of line followed by text
		pointPattern: regexp.MustCompile(`^\(([a-z])\)\s+(.*)$`),
		// Matches "(i) ", "(ii) ", "(iii) ", etc. at start of line (roman numeral sub-points)
		subPointPattern: regexp.MustCompile(`^\((i{1,3}|iv|vi{0,3}|ix|x{0,3})\)\s+(.*)$`),
	}
}

// ExtractProvisions parses article text into structured paragraphs with points.
// It modifies the article in place, populating the Paragraphs field.
func (e *ProvisionExtractor) ExtractProvisions(article *Article) {
	if article == nil || article.Text == "" {
		return
	}

	lines := strings.Split(article.Text, "\n")
	article.Paragraphs = make([]*Paragraph, 0)

	var currentParagraph *Paragraph
	var currentPoint *Point
	var textBuffer strings.Builder

	flushText := func() string {
		result := strings.TrimSpace(textBuffer.String())
		textBuffer.Reset()
		return result
	}

	saveCurrentPoint := func() {
		if currentPoint != nil && currentParagraph != nil {
			currentPoint.Text = flushText()
			if currentPoint.Text != "" {
				currentParagraph.Points = append(currentParagraph.Points, currentPoint)
			}
			currentPoint = nil
		}
	}

	saveCurrentParagraph := func() {
		saveCurrentPoint()
		if currentParagraph != nil {
			if textBuffer.Len() > 0 {
				currentParagraph.Text = flushText()
			}
			article.Paragraphs = append(article.Paragraphs, currentParagraph)
			currentParagraph = nil
		}
	}

	for _, line := range lines {
		// Check for new paragraph
		if m := e.paragraphPattern.FindStringSubmatch(line); m != nil {
			saveCurrentParagraph()

			num := mustAtoi(m[1])
			currentParagraph = &Paragraph{
				Number: num,
				Points: make([]*Point, 0),
			}
			textBuffer.WriteString(m[2])
			continue
		}

		// Check for new point
		if m := e.pointPattern.FindStringSubmatch(line); m != nil {
			// Skip if this looks like a reference (e.g., "(a) of Article 9(2)")
			if isPointReference(m[2]) {
				// Treat as continuation text
				if textBuffer.Len() > 0 {
					textBuffer.WriteString(" ")
				}
				textBuffer.WriteString(line)
				continue
			}

			saveCurrentPoint()

			currentPoint = &Point{
				Letter:    m[1],
				SubPoints: make([]*SubPoint, 0),
			}
			textBuffer.WriteString(m[2])
			continue
		}

		// Check for sub-point (roman numeral)
		if m := e.subPointPattern.FindStringSubmatch(line); m != nil {
			if currentPoint != nil {
				// Save accumulated text to current point before adding sub-point
				if textBuffer.Len() > 0 && currentPoint.Text == "" {
					currentPoint.Text = flushText()
				}

				subPoint := &SubPoint{
					Numeral: m[1],
					Text:    m[2],
				}
				currentPoint.SubPoints = append(currentPoint.SubPoints, subPoint)
			}
			continue
		}

		// Continuation line - append to current buffer
		if line != "" {
			if textBuffer.Len() > 0 {
				textBuffer.WriteString(" ")
			}
			textBuffer.WriteString(line)
		}
	}

	// Save final paragraph/point
	saveCurrentParagraph()

	// If no paragraphs were found but there's text, create a single implicit paragraph
	if len(article.Paragraphs) == 0 && article.Text != "" {
		article.Paragraphs = append(article.Paragraphs, &Paragraph{
			Number: 0, // Implicit/unnumbered paragraph
			Text:   article.Text,
		})
	}
}

// ExtractAllProvisions extracts provisions from all articles in a document.
func (e *ProvisionExtractor) ExtractAllProvisions(doc *Document) {
	for _, article := range doc.AllArticles() {
		e.ExtractProvisions(article)
	}
}

// mustAtoi converts string to int, returning 0 on error.
func mustAtoi(s string) int {
	var result int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		}
	}
	return result
}

// ProvisionStatistics holds statistics about extracted provisions.
type ProvisionStatistics struct {
	ArticlesWithText       int `json:"articles_with_text"`
	TotalParagraphs        int `json:"total_paragraphs"`
	TotalPoints            int `json:"total_points"`
	TotalSubPoints         int `json:"total_sub_points"`
	ArticlesWithParagraphs int `json:"articles_with_paragraphs"`
}

// ProvisionStats calculates statistics about extracted provisions in a document.
func ProvisionStats(doc *Document) ProvisionStatistics {
	stats := ProvisionStatistics{}

	for _, article := range doc.AllArticles() {
		if article.Text != "" {
			stats.ArticlesWithText++
		}

		if len(article.Paragraphs) > 0 {
			stats.ArticlesWithParagraphs++
		}

		for _, para := range article.Paragraphs {
			if para.Number > 0 { // Only count numbered paragraphs
				stats.TotalParagraphs++
			}

			for _, point := range para.Points {
				stats.TotalPoints++
				stats.TotalSubPoints += len(point.SubPoints)
			}
		}
	}

	return stats
}

// SerializeArticle converts an article back to text form for round-trip testing.
func SerializeArticle(article *Article) string {
	if article == nil {
		return ""
	}

	var sb strings.Builder

	for _, para := range article.Paragraphs {
		if para.Number > 0 {
			sb.WriteString(formatParagraphNumber(para.Number))
			sb.WriteString(para.Text)
			sb.WriteString("\n")
		} else if para.Text != "" {
			sb.WriteString(para.Text)
			sb.WriteString("\n")
		}

		for _, point := range para.Points {
			sb.WriteString("(")
			sb.WriteString(point.Letter)
			sb.WriteString(") ")
			sb.WriteString(point.Text)
			sb.WriteString("\n")

			for _, subPoint := range point.SubPoints {
				sb.WriteString("(")
				sb.WriteString(subPoint.Numeral)
				sb.WriteString(") ")
				sb.WriteString(subPoint.Text)
				sb.WriteString("\n")
			}
		}
	}

	return strings.TrimSpace(sb.String())
}

// formatParagraphNumber formats a paragraph number with appropriate spacing.
func formatParagraphNumber(num int) string {
	return strings.Repeat(" ", 0) + itoa(num) + ".   "
}

// itoa converts int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// isPointReference checks if text after a point letter looks like a reference
// rather than the start of a new point (e.g., "of Article 9(2)" is a reference).
func isPointReference(text string) bool {
	text = strings.TrimSpace(text)
	// References typically start with "of Article" or "of paragraph"
	if strings.HasPrefix(text, "of Article") || strings.HasPrefix(text, "of paragraph") {
		return true
	}
	// Also check for "of the first" which is a reference to earlier text
	if strings.HasPrefix(text, "of the first") || strings.HasPrefix(text, "of the second") {
		return true
	}
	return false
}
