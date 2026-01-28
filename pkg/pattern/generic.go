package pattern

import (
	"regexp"
	"strings"
	"unicode"
)

// WarningLevel indicates the severity of a parsing warning.
type WarningLevel string

const (
	WarningLevelInfo    WarningLevel = "info"
	WarningLevelWarning WarningLevel = "warning"
	WarningLevelError   WarningLevel = "error"
)

// ParseWarning represents a warning generated during parsing.
type ParseWarning struct {
	Level   WarningLevel
	Message string
	Line    int    // Line number where issue occurred (0 if not applicable)
	Context string // Surrounding context for debugging
}

// GenericDocument represents a document parsed by the generic parser.
type GenericDocument struct {
	Format       string            // "generic"
	Confidence   float64           // How confident we are in the structure
	Title        string            // Detected title (if any)
	Sections     []*GenericSection // Detected sections/chapters
	Definitions  []*GenericDefinition
	Hierarchy    *DetectedHierarchy
	RawLineCount int
}

// GenericSection represents a detected section in the document.
type GenericSection struct {
	Level      int    // Nesting level (0 = top)
	Number     string // Detected number (e.g., "1", "I", "A")
	Title      string // Section title
	Content    string // Section content
	Children   []*GenericSection
	LineStart  int
	LineEnd    int
	NumberType HierarchyType // Type of numbering used
}

// GenericDefinition represents a detected definition.
type GenericDefinition struct {
	Term       string
	Definition string
	Line       int
	Confidence float64 // How confident we are this is a definition
}

// HierarchyType represents the type of numbering in a hierarchy.
type HierarchyType string

const (
	HierarchyTypeArabic      HierarchyType = "arabic"       // 1, 2, 3
	HierarchyTypeLowerLetter HierarchyType = "lower_letter" // a, b, c
	HierarchyTypeUpperLetter HierarchyType = "upper_letter" // A, B, C
	HierarchyTypeLowerRoman  HierarchyType = "lower_roman"  // i, ii, iii
	HierarchyTypeUpperRoman  HierarchyType = "upper_roman"  // I, II, III
	HierarchyTypeUnknown     HierarchyType = "unknown"
)

// DetectedHierarchy describes the detected hierarchy structure.
type DetectedHierarchy struct {
	Levels        []DetectedLevel
	IndentBased   bool // Whether indentation is used for nesting
	MaxDepth      int
	TotalSections int
}

// DetectedLevel represents a level in the detected hierarchy.
type DetectedLevel struct {
	Depth   int
	Type    HierarchyType
	Pattern string
	Count   int // Number of items at this level
}

// GenericParser attempts to extract structure from documents without specific patterns.
type GenericParser struct {
	// Configuration
	ConfidenceThreshold float64 // Minimum confidence to trigger (default 0.5)

	// Hierarchy detection patterns
	arabicNumbered      *regexp.Regexp
	arabicParenNumbered *regexp.Regexp
	arabicDotNumbered   *regexp.Regexp
	lowerLetterParen    *regexp.Regexp
	lowerLetterDot      *regexp.Regexp
	upperLetterParen    *regexp.Regexp
	upperLetterDot      *regexp.Regexp
	lowerRomanParen     *regexp.Regexp
	lowerRomanDot       *regexp.Regexp
	upperRomanParen     *regexp.Regexp
	upperRomanDot       *regexp.Regexp

	// Section detection patterns
	allCapsHeader    *regexp.Regexp
	numberedHeader   *regexp.Regexp
	underlinedHeader *regexp.Regexp

	// Definition detection patterns
	quotedMeans     *regexp.Regexp
	quotedRefersTo  *regexp.Regexp
	colonDefinition *regexp.Regexp
}

// NewGenericParser creates a new generic parser with default configuration.
func NewGenericParser() *GenericParser {
	return &GenericParser{
		ConfidenceThreshold: 0.5,

		// Arabic numbered: "1.", "1)", "(1)"
		arabicNumbered:      regexp.MustCompile(`^(\d+)\.\s+`),
		arabicParenNumbered: regexp.MustCompile(`^\((\d+)\)\s+`),
		arabicDotNumbered:   regexp.MustCompile(`^(\d+)\)\s+`),

		// Letter patterns
		lowerLetterParen: regexp.MustCompile(`^\(([a-z])\)\s+`),
		lowerLetterDot:   regexp.MustCompile(`^([a-z])\.\s+`),
		upperLetterParen: regexp.MustCompile(`^\(([A-Z])\)\s+`),
		upperLetterDot:   regexp.MustCompile(`^([A-Z])\.\s+`),

		// Roman numeral patterns
		lowerRomanParen: regexp.MustCompile(`^\(([ivxlcdm]+)\)\s+`),
		lowerRomanDot:   regexp.MustCompile(`^([ivxlcdm]+)\.\s+`),
		upperRomanParen: regexp.MustCompile(`^\(([IVXLCDM]+)\)\s+`),
		upperRomanDot:   regexp.MustCompile(`^([IVXLCDM]+)\.\s+`),

		// Section/header patterns
		allCapsHeader:    regexp.MustCompile(`^[A-Z][A-Z\s]{3,}[A-Z]$`),
		numberedHeader:   regexp.MustCompile(`^(?:CHAPTER|SECTION|PART|TITLE|ARTICLE)\s+(?:\d+|[IVXLCDM]+)`),
		underlinedHeader: regexp.MustCompile(`^[-=]{3,}$`),

		// Definition patterns
		quotedMeans:     regexp.MustCompile(`[""'']([^""'']+)[""'']\s+(?:means?|shall\s+mean)`),
		quotedRefersTo:  regexp.MustCompile(`[""'']([^""'']+)[""'']\s+(?:refers?\s+to|has\s+the\s+(?:same\s+)?meaning)`),
		colonDefinition: regexp.MustCompile(`^([A-Z][a-zA-Z\s]+):\s+`),
	}
}

// GenericParserOptions configures generic parser behavior.
type GenericParserOptions struct {
	ConfidenceThreshold float64
	ExtractDefinitions  bool
	MaxSections         int // Maximum sections to extract (0 = unlimited)
}

// DefaultGenericParserOptions returns sensible defaults.
func DefaultGenericParserOptions() GenericParserOptions {
	return GenericParserOptions{
		ConfidenceThreshold: 0.5,
		ExtractDefinitions:  true,
		MaxSections:         0,
	}
}

// Parse attempts to extract structure from content using heuristics.
// Returns the parsed document and any warnings generated.
func (p *GenericParser) Parse(content string) (*GenericDocument, []ParseWarning) {
	return p.ParseWithOptions(content, DefaultGenericParserOptions())
}

// ParseWithOptions parses with custom options.
func (p *GenericParser) ParseWithOptions(content string, options GenericParserOptions) (*GenericDocument, []ParseWarning) {
	var warnings []ParseWarning

	lines := strings.Split(content, "\n")
	doc := &GenericDocument{
		Format:       "generic",
		RawLineCount: len(lines),
		Sections:     make([]*GenericSection, 0),
		Definitions:  make([]*GenericDefinition, 0),
	}

	// Detect title (usually first non-empty line or first ALL CAPS line)
	doc.Title, warnings = p.detectTitle(lines, warnings)

	// Detect hierarchy structure
	doc.Hierarchy, warnings = p.detectHierarchy(lines, warnings)

	// Extract sections based on detected hierarchy
	doc.Sections, warnings = p.extractSections(lines, doc.Hierarchy, warnings, options)

	// Extract definitions if requested
	if options.ExtractDefinitions {
		doc.Definitions, warnings = p.extractDefinitions(lines, warnings)
	}

	// Calculate overall confidence
	doc.Confidence = p.calculateConfidence(doc, warnings)

	// Add summary warning if confidence is low
	if doc.Confidence < options.ConfidenceThreshold {
		warnings = append(warnings, ParseWarning{
			Level:   WarningLevelWarning,
			Message: "Low confidence parsing - results may be unreliable",
		})
	}

	return doc, warnings
}

// ShouldUseGenericParser determines if generic parsing should be used based on detection results.
func ShouldUseGenericParser(matches []FormatMatch, threshold float64) bool {
	if len(matches) == 0 {
		return true
	}
	return matches[0].Confidence < threshold
}

// detectTitle attempts to find the document title.
func (p *GenericParser) detectTitle(lines []string, warnings []ParseWarning) (string, []ParseWarning) {
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if it's an ALL CAPS line (likely title)
		if p.allCapsHeader.MatchString(line) && len(line) > 5 {
			return line, warnings
		}

		// First substantial line might be the title
		if len(line) > 10 && i < 5 {
			return line, warnings
		}
	}

	warnings = append(warnings, ParseWarning{
		Level:   WarningLevelInfo,
		Message: "Could not detect document title",
	})
	return "", warnings
}

// detectHierarchy analyzes the document to detect its hierarchical structure.
func (p *GenericParser) detectHierarchy(lines []string, warnings []ParseWarning) (*DetectedHierarchy, []ParseWarning) {
	hierarchy := &DetectedHierarchy{
		Levels: make([]DetectedLevel, 0),
	}

	// Count different numbering patterns
	counts := map[HierarchyType]int{
		HierarchyTypeArabic:      0,
		HierarchyTypeLowerLetter: 0,
		HierarchyTypeUpperLetter: 0,
		HierarchyTypeLowerRoman:  0,
		HierarchyTypeUpperRoman:  0,
	}

	indentLevels := make(map[int]int) // indent -> count

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Check for different numbering types
		if p.arabicNumbered.MatchString(trimmed) || p.arabicParenNumbered.MatchString(trimmed) {
			counts[HierarchyTypeArabic]++
		}
		if p.lowerLetterParen.MatchString(trimmed) || p.lowerLetterDot.MatchString(trimmed) {
			counts[HierarchyTypeLowerLetter]++
		}
		if p.upperLetterParen.MatchString(trimmed) || p.upperLetterDot.MatchString(trimmed) {
			counts[HierarchyTypeUpperLetter]++
		}
		if p.lowerRomanParen.MatchString(trimmed) || p.lowerRomanDot.MatchString(trimmed) {
			counts[HierarchyTypeLowerRoman]++
		}
		if p.upperRomanParen.MatchString(trimmed) || p.upperRomanDot.MatchString(trimmed) {
			counts[HierarchyTypeUpperRoman]++
		}

		// Check indentation
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if indent > 0 {
			indentLevels[indent]++
		}
	}

	// Build hierarchy levels based on what we found
	depth := 0
	for _, hType := range []HierarchyType{
		HierarchyTypeUpperRoman, // Usually top level
		HierarchyTypeArabic,
		HierarchyTypeUpperLetter,
		HierarchyTypeLowerLetter,
		HierarchyTypeLowerRoman,
	} {
		if counts[hType] > 0 {
			hierarchy.Levels = append(hierarchy.Levels, DetectedLevel{
				Depth: depth,
				Type:  hType,
				Count: counts[hType],
			})
			hierarchy.TotalSections += counts[hType]
			depth++
		}
	}

	hierarchy.MaxDepth = depth
	hierarchy.IndentBased = len(indentLevels) > 1

	if len(hierarchy.Levels) == 0 {
		warnings = append(warnings, ParseWarning{
			Level:   WarningLevelWarning,
			Message: "Could not detect document hierarchy",
		})
	}

	return hierarchy, warnings
}

// extractSections extracts sections based on detected hierarchy.
func (p *GenericParser) extractSections(lines []string, hierarchy *DetectedHierarchy, warnings []ParseWarning, options GenericParserOptions) ([]*GenericSection, []ParseWarning) {
	sections := make([]*GenericSection, 0)

	if hierarchy == nil || len(hierarchy.Levels) == 0 {
		// Fall back to detecting major divisions
		return p.extractSectionsByHeaders(lines, warnings, options)
	}

	var currentSection *GenericSection
	var contentBuilder strings.Builder
	sectionCount := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if currentSection != nil {
				contentBuilder.WriteString("\n")
			}
			continue
		}

		// Check if this line starts a new section
		numberType, number := p.detectNumbering(trimmed)
		if numberType != HierarchyTypeUnknown {
			// Save previous section
			if currentSection != nil {
				currentSection.Content = strings.TrimSpace(contentBuilder.String())
				currentSection.LineEnd = i - 1
				sections = append(sections, currentSection)
				sectionCount++

				if options.MaxSections > 0 && sectionCount >= options.MaxSections {
					warnings = append(warnings, ParseWarning{
						Level:   WarningLevelInfo,
						Message: "Maximum section limit reached",
					})
					return sections, warnings
				}
			}

			// Extract title (text after the number on the same line)
			title := p.extractSectionTitle(trimmed, numberType)

			currentSection = &GenericSection{
				Level:      p.getHierarchyDepth(numberType, hierarchy),
				Number:     number,
				Title:      title,
				LineStart:  i,
				NumberType: numberType,
				Children:   make([]*GenericSection, 0),
			}
			contentBuilder.Reset()
		} else if currentSection != nil {
			contentBuilder.WriteString(trimmed)
			contentBuilder.WriteString("\n")
		} else if p.allCapsHeader.MatchString(trimmed) || p.numberedHeader.MatchString(trimmed) {
			// This is a header without typical numbering
			currentSection = &GenericSection{
				Level:      0,
				Title:      trimmed,
				LineStart:  i,
				NumberType: HierarchyTypeUnknown,
				Children:   make([]*GenericSection, 0),
			}
			contentBuilder.Reset()
		}
	}

	// Save last section
	if currentSection != nil {
		currentSection.Content = strings.TrimSpace(contentBuilder.String())
		currentSection.LineEnd = len(lines) - 1
		sections = append(sections, currentSection)
	}

	return sections, warnings
}

// extractSectionsByHeaders extracts sections using header detection when no hierarchy found.
func (p *GenericParser) extractSectionsByHeaders(lines []string, warnings []ParseWarning, options GenericParserOptions) ([]*GenericSection, []ParseWarning) {
	sections := make([]*GenericSection, 0)

	var currentSection *GenericSection
	var contentBuilder strings.Builder
	sectionCount := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if currentSection != nil {
				contentBuilder.WriteString("\n")
			}
			continue
		}

		isHeader := p.allCapsHeader.MatchString(trimmed) ||
			p.numberedHeader.MatchString(trimmed) ||
			(i+1 < len(lines) && p.underlinedHeader.MatchString(strings.TrimSpace(lines[i+1])))

		if isHeader {
			// Save previous section
			if currentSection != nil {
				currentSection.Content = strings.TrimSpace(contentBuilder.String())
				currentSection.LineEnd = i - 1
				sections = append(sections, currentSection)
				sectionCount++

				if options.MaxSections > 0 && sectionCount >= options.MaxSections {
					return sections, warnings
				}
			}

			currentSection = &GenericSection{
				Level:     0,
				Title:     trimmed,
				LineStart: i,
				Children:  make([]*GenericSection, 0),
			}
			contentBuilder.Reset()
		} else if currentSection != nil && !p.underlinedHeader.MatchString(trimmed) {
			contentBuilder.WriteString(trimmed)
			contentBuilder.WriteString("\n")
		}
	}

	// Save last section
	if currentSection != nil {
		currentSection.Content = strings.TrimSpace(contentBuilder.String())
		currentSection.LineEnd = len(lines) - 1
		sections = append(sections, currentSection)
	}

	if len(sections) == 0 {
		warnings = append(warnings, ParseWarning{
			Level:   WarningLevelWarning,
			Message: "Could not detect any sections in document",
		})
	}

	return sections, warnings
}

// extractDefinitions attempts to find definitions in the text.
func (p *GenericParser) extractDefinitions(lines []string, warnings []ParseWarning) ([]*GenericDefinition, []ParseWarning) {
	definitions := make([]*GenericDefinition, 0)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Try different definition patterns
		if matches := p.quotedMeans.FindStringSubmatch(trimmed); len(matches) > 1 {
			definitions = append(definitions, &GenericDefinition{
				Term:       matches[1],
				Definition: extractDefinitionText(trimmed, matches[0]),
				Line:       i,
				Confidence: 0.9,
			})
		} else if matches := p.quotedRefersTo.FindStringSubmatch(trimmed); len(matches) > 1 {
			definitions = append(definitions, &GenericDefinition{
				Term:       matches[1],
				Definition: extractDefinitionText(trimmed, matches[0]),
				Line:       i,
				Confidence: 0.85,
			})
		} else if matches := p.colonDefinition.FindStringSubmatch(trimmed); len(matches) > 1 {
			// Lower confidence for colon definitions (might be false positives)
			definitions = append(definitions, &GenericDefinition{
				Term:       strings.TrimSpace(matches[1]),
				Definition: strings.TrimPrefix(trimmed, matches[0]),
				Line:       i,
				Confidence: 0.6,
			})
		}
	}

	if len(definitions) == 0 {
		warnings = append(warnings, ParseWarning{
			Level:   WarningLevelInfo,
			Message: "No definitions detected in document",
		})
	}

	return definitions, warnings
}

// detectNumbering determines the numbering type of a line.
func (p *GenericParser) detectNumbering(line string) (HierarchyType, string) {
	if matches := p.upperRomanDot.FindStringSubmatch(line); len(matches) > 1 {
		if isRomanNumeral(matches[1]) {
			return HierarchyTypeUpperRoman, matches[1]
		}
	}
	if matches := p.upperRomanParen.FindStringSubmatch(line); len(matches) > 1 {
		if isRomanNumeral(matches[1]) {
			return HierarchyTypeUpperRoman, matches[1]
		}
	}
	if matches := p.arabicNumbered.FindStringSubmatch(line); len(matches) > 1 {
		return HierarchyTypeArabic, matches[1]
	}
	if matches := p.arabicParenNumbered.FindStringSubmatch(line); len(matches) > 1 {
		return HierarchyTypeArabic, matches[1]
	}
	if matches := p.upperLetterDot.FindStringSubmatch(line); len(matches) > 1 {
		return HierarchyTypeUpperLetter, matches[1]
	}
	if matches := p.upperLetterParen.FindStringSubmatch(line); len(matches) > 1 {
		return HierarchyTypeUpperLetter, matches[1]
	}
	if matches := p.lowerLetterDot.FindStringSubmatch(line); len(matches) > 1 {
		return HierarchyTypeLowerLetter, matches[1]
	}
	if matches := p.lowerLetterParen.FindStringSubmatch(line); len(matches) > 1 {
		return HierarchyTypeLowerLetter, matches[1]
	}
	if matches := p.lowerRomanDot.FindStringSubmatch(line); len(matches) > 1 {
		if isRomanNumeral(strings.ToUpper(matches[1])) {
			return HierarchyTypeLowerRoman, matches[1]
		}
	}
	if matches := p.lowerRomanParen.FindStringSubmatch(line); len(matches) > 1 {
		if isRomanNumeral(strings.ToUpper(matches[1])) {
			return HierarchyTypeLowerRoman, matches[1]
		}
	}

	return HierarchyTypeUnknown, ""
}

// extractSectionTitle extracts the title text after a section number.
func (p *GenericParser) extractSectionTitle(line string, numberType HierarchyType) string {
	// Remove the numbering prefix
	patterns := []*regexp.Regexp{
		p.upperRomanDot, p.upperRomanParen,
		p.arabicNumbered, p.arabicParenNumbered, p.arabicDotNumbered,
		p.upperLetterDot, p.upperLetterParen,
		p.lowerLetterDot, p.lowerLetterParen,
		p.lowerRomanDot, p.lowerRomanParen,
	}

	for _, pat := range patterns {
		if loc := pat.FindStringIndex(line); loc != nil {
			title := strings.TrimSpace(line[loc[1]:])
			// Clean up common title formatting
			title = strings.TrimPrefix(title, "- ")
			title = strings.TrimPrefix(title, "– ")
			title = strings.TrimPrefix(title, "— ")
			return title
		}
	}

	return ""
}

// getHierarchyDepth returns the depth for a given numbering type.
func (p *GenericParser) getHierarchyDepth(numberType HierarchyType, hierarchy *DetectedHierarchy) int {
	for _, level := range hierarchy.Levels {
		if level.Type == numberType {
			return level.Depth
		}
	}
	return 0
}

// calculateConfidence calculates overall parsing confidence.
func (p *GenericParser) calculateConfidence(doc *GenericDocument, warnings []ParseWarning) float64 {
	confidence := 1.0

	// Reduce confidence for each warning
	for _, w := range warnings {
		switch w.Level {
		case WarningLevelError:
			confidence -= 0.3
		case WarningLevelWarning:
			confidence -= 0.15
		case WarningLevelInfo:
			confidence -= 0.05
		}
	}

	// Boost confidence if we found structure
	if doc.Hierarchy != nil && len(doc.Hierarchy.Levels) > 0 {
		confidence += 0.1
	}
	if len(doc.Sections) > 0 {
		confidence += 0.1
	}
	if len(doc.Definitions) > 0 {
		confidence += 0.05
	}
	if doc.Title != "" {
		confidence += 0.05
	}

	// Clamp to [0, 1]
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}

	return confidence
}

// Helper functions

func extractDefinitionText(line, match string) string {
	idx := strings.Index(line, match)
	if idx >= 0 {
		return strings.TrimSpace(line[idx+len(match):])
	}
	return ""
}

func isRomanNumeral(s string) bool {
	if s == "" {
		return false
	}
	s = strings.ToUpper(s)
	for _, c := range s {
		switch c {
		case 'I', 'V', 'X', 'L', 'C', 'D', 'M':
			continue
		default:
			return false
		}
	}
	return true
}

// isAllCaps checks if a string is all uppercase letters.
func isAllCaps(s string) bool {
	hasLetter := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			hasLetter = true
			if !unicode.IsUpper(r) {
				return false
			}
		}
	}
	return hasLetter
}
