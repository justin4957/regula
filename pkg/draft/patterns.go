package draft

import (
	"fmt"
	"regexp"
	"strings"
)

// Recognizer extracts structured amendment instructions from draft bill
// section text. It identifies what a bill proposes to change in existing
// law by matching patterns like "by striking X and inserting Y" or
// "is repealed". The Recognizer is safe for concurrent use.
type Recognizer struct {
	// Target reference patterns
	uscCitationPattern    *regexp.Regexp
	sectionOfTitlePattern *regexp.Regexp
	titleOfUSCPattern     *regexp.Regexp

	// Amendment action patterns
	strikeInsertPattern    *regexp.Regexp
	repealPattern          *regexp.Regexp
	addNewSectionPattern   *regexp.Regexp
	addAtEndPattern        *regexp.Regexp
	redesignatePattern     *regexp.Regexp
	tableOfContentsPattern *regexp.Regexp

	// Structural patterns
	isAmendedPattern      *regexp.Regexp
	numberedClausePattern *regexp.Regexp
	letteredClausePattern *regexp.Regexp
	inSubsectionPattern   *regexp.Regexp
	paragraphRefPattern   *regexp.Regexp

	// Text extraction
	quotedTextPattern *regexp.Regexp
}

// numberedClause represents a numbered sub-item within an amendment block,
// such as "(1) by striking..." or "(2) by adding at the end...".
type numberedClause struct {
	number string
	text   string
}

// NewRecognizer creates a Recognizer with all regex patterns compiled.
// The recognizer is safe for concurrent use across multiple goroutines.
func NewRecognizer() *Recognizer {
	return &Recognizer{
		// "(15 U.S.C. 6502)" or "(15 U.S.C. 6505(d))" or "(11 U.S.C. 101 et seq.)"
		uscCitationPattern: regexp.MustCompile(
			`\((\d+)\s+U\.S\.C\.\s+(\d+[a-z]?)(\([a-z]\)(?:\(\d+\))*)?(?:\s+et\s+seq\.)?\)`,
		),
		// "Section 1303" or "Section 1306(d)" — extracts section number and optional subsection
		sectionOfTitlePattern: regexp.MustCompile(
			`(?i)Section\s+(\d+[a-zA-Z]?)(\([a-z]\)(?:\(\d+\))*)?`,
		),
		// "of title NN" or "Title NN," — extracts title number
		titleOfUSCPattern: regexp.MustCompile(
			`(?i)(?:of\s+)?title\s+(\d+)`,
		),

		// "by striking "X" and inserting "Y"" or "striking "X" and inserting "Y""
		// "by" is optional because "is amended by" anchor may consume it
		strikeInsertPattern: regexp.MustCompile(
			`(?i)(?:by\s+)?striking\s+` +
				`["\x{201c}]([^"\x{201d}]+)["\x{201d}]` +
				`\s+and\s+inserting\s+` +
				`["\x{201c}]([^"\x{201d}]+)["\x{201d}]`,
		),
		// "is repealed" or "is hereby repealed"
		repealPattern: regexp.MustCompile(
			`(?i)is\s+(?:hereby\s+)?repealed`,
		),
		// "inserting after section X the following new section/subsection"
		addNewSectionPattern: regexp.MustCompile(
			`(?i)(?:by\s+)?inserting\s+after\s+(?:section|subsection)\s+` +
				`(\([a-zA-Z0-9]+\)|\d+[a-zA-Z]?)\s+the\s+following\s+new\s+(?:section|subsection)`,
		),
		// "adding at the end the following"
		addAtEndPattern: regexp.MustCompile(
			`(?i)(?:by\s+)?adding\s+at\s+the\s+end\s+the\s+following`,
		),
		// "redesignating paragraph (X) as paragraph (Y)"
		redesignatePattern: regexp.MustCompile(
			`(?i)(?:by\s+)?redesignating\s+` +
				`(?:paragraph|subsection|section|subparagraph|clause)\s+` +
				`\(([a-zA-Z0-9]+)\)\s+as\s+` +
				`(?:paragraph|subsection|section|subparagraph|clause)\s+` +
				`\(([a-zA-Z0-9]+)\)`,
		),
		// "table of contents ... is amended"
		tableOfContentsPattern: regexp.MustCompile(
			`(?i)table\s+of\s+contents\s+.{0,120}is\s+amended`,
		),

		// "is amended--" (em dash, double hyphen) or "is amended by"
		// Works on both multi-line and normalized text
		isAmendedPattern: regexp.MustCompile(
			`(?i)is\s+amended\s*[\x{2014}\-]{1,2}|is\s+amended\s+by\b`,
		),
		// "(1) ", "(2) " etc. at clause boundaries (with leading whitespace)
		numberedClausePattern: regexp.MustCompile(
			`(?m)^\s+\((\d+)\)\s`,
		),
		// "(A) ", "(B) " etc. at lettered sub-item boundaries (with leading whitespace)
		letteredClausePattern: regexp.MustCompile(
			`(?m)^\s+\(([A-Z])\)\s`,
		),
		// "in subsection (b)--" or "intes in subsection (b)--"
		inSubsectionPattern: regexp.MustCompile(
			`(?i)(?:in(?:tes\s+in)?)\s+subsection\s+\(([a-z])\)`,
		),
		// "paragraph (1)", "paragraph (2)"
		paragraphRefPattern: regexp.MustCompile(
			`(?i)(?:in\s+)?paragraph\s+\((\d+[A-Za-z]*)\)`,
		),

		// Quoted text between straight or curly quotes
		quotedTextPattern: regexp.MustCompile(
			`["\x{201c}]([^"\x{201d}]+)["\x{201d}]`,
		),
	}
}

// ClassifyAmendmentType determines the type of amendment action described
// in the given text. It checks patterns in priority order (most specific
// first) and returns the first match, or an empty AmendmentType if no
// pattern matches.
func (recognizer *Recognizer) ClassifyAmendmentType(text string) AmendmentType {
	normalizedText := normalizeAmendmentText(text)

	if recognizer.strikeInsertPattern.MatchString(normalizedText) {
		return AmendStrikeInsert
	}
	if recognizer.redesignatePattern.MatchString(normalizedText) {
		return AmendRedesignate
	}
	if recognizer.tableOfContentsPattern.MatchString(normalizedText) {
		return AmendTableOfContents
	}
	if recognizer.addNewSectionPattern.MatchString(normalizedText) {
		return AmendAddNewSection
	}
	if recognizer.addAtEndPattern.MatchString(normalizedText) {
		return AmendAddAtEnd
	}
	if recognizer.repealPattern.MatchString(normalizedText) {
		return AmendRepeal
	}

	return ""
}

// ParseTargetReference extracts the USC title, section, and optional
// subsection from amendment text. It first tries the parenthetical
// U.S.C. citation format "(NN U.S.C. NNNN)", then falls back to the
// "Section X of title Y" format.
func (recognizer *Recognizer) ParseTargetReference(text string) (title, section, subsection string, err error) {
	normalizedText := normalizeAmendmentText(text)

	// Try parenthetical USC citation first — most reliable
	if uscMatch := recognizer.uscCitationPattern.FindStringSubmatch(normalizedText); uscMatch != nil {
		title = uscMatch[1]
		section = uscMatch[2]
		if len(uscMatch) > 3 {
			subsection = uscMatch[3]
		}
		return title, section, subsection, nil
	}

	// Fall back to "Section X of title Y" format
	sectionMatch := recognizer.sectionOfTitlePattern.FindStringSubmatch(normalizedText)
	titleMatch := recognizer.titleOfUSCPattern.FindStringSubmatch(normalizedText)

	if sectionMatch != nil && titleMatch != nil {
		title = titleMatch[1]
		section = sectionMatch[1]
		if len(sectionMatch) > 2 {
			subsection = sectionMatch[2]
		}
		return title, section, subsection, nil
	}

	// Handle "Title N" without explicit "Section X" (used with "et seq.")
	if titleMatch != nil {
		title = titleMatch[1]
		return title, "", "", nil
	}

	return "", "", "", fmt.Errorf("no target reference found in text")
}

// ExtractAmendments analyzes the raw text of a bill section and returns
// structured Amendment values for each amendment instruction found. For
// sections that contain no amendment language, it returns an empty slice
// with no error.
//
// The function operates on the original multi-line text to preserve
// structural indentation needed for clause splitting, and normalizes
// individual clause text for pattern matching.
func (recognizer *Recognizer) ExtractAmendments(sectionText string) ([]Amendment, error) {
	var amendments []Amendment

	// Check for direct repeal pattern (no "is amended" anchor)
	repealAmendments := recognizer.extractRepealAmendments(sectionText)
	amendments = append(amendments, repealAmendments...)

	// Find "is amended" anchors on the original text
	anchorLocations := recognizer.isAmendedPattern.FindAllStringIndex(sectionText, -1)
	if len(anchorLocations) == 0 {
		if amendments == nil {
			amendments = []Amendment{}
		}
		return amendments, nil
	}

	for anchorIndex, anchorLocation := range anchorLocations {
		preambleText := sectionText[:anchorLocation[0]]

		// Determine the end of this amendment block
		var blockEnd int
		if anchorIndex+1 < len(anchorLocations) {
			// Find the start of the next preamble by looking for the subsection
			// marker (e.g. "(b) ") before the next "is amended"
			nextPreambleStart := findPreambleStart(sectionText, anchorLocations[anchorIndex+1][0])
			blockEnd = nextPreambleStart
		} else {
			blockEnd = len(sectionText)
		}

		afterAnchorText := sectionText[anchorLocation[1]:blockEnd]

		// Parse target reference from preamble
		targetTitle, targetSection, targetSubsection, targetErr := recognizer.ParseTargetReference(preambleText)
		if targetErr != nil {
			continue
		}

		// Split into numbered clauses if present
		clauses := recognizer.splitNumberedClauses(afterAnchorText)

		if len(clauses) == 0 {
			// Single amendment (no numbered sub-items)
			amendment := recognizer.buildAmendment(afterAnchorText, targetTitle, targetSection, targetSubsection)
			if amendment.Type != "" {
				amendments = append(amendments, amendment)
			}
		} else {
			// Check if the text before the first numbered clause is itself
			// a complete amendment. This handles cases where a single inline
			// amendment is followed by unrelated numbered content (e.g.,
			// subsection (b) with numbered paragraphs).
			firstClauseStart := strings.Index(afterAnchorText, "("+clauses[0].number+")")
			if firstClauseStart > 0 {
				textBeforeClauses := afterAnchorText[:firstClauseStart]
				preAmendType := recognizer.ClassifyAmendmentType(textBeforeClauses)
				if preAmendType != "" {
					// The amendment is complete before the numbered clauses
					amendment := recognizer.buildAmendment(textBeforeClauses, targetTitle, targetSection, targetSubsection)
					if amendment.Type != "" {
						amendments = append(amendments, amendment)
					}
					continue
				}
			}

			// Multi-amendment block
			blockAmendments := recognizer.processNumberedClauses(clauses, targetTitle, targetSection, targetSubsection)
			amendments = append(amendments, blockAmendments...)
		}
	}

	if amendments == nil {
		amendments = []Amendment{}
	}
	return amendments, nil
}

// extractRepealAmendments finds "is repealed" or "is hereby repealed"
// patterns that don't use the standard "is amended" anchor.
func (recognizer *Recognizer) extractRepealAmendments(sectionText string) []Amendment {
	normalizedText := normalizeAmendmentText(sectionText)
	if !recognizer.repealPattern.MatchString(normalizedText) {
		return nil
	}
	// Don't double-count if this text also has "is amended"
	if recognizer.isAmendedPattern.MatchString(sectionText) {
		return nil
	}

	targetTitle, targetSection, targetSubsection, targetErr := recognizer.ParseTargetReference(normalizedText)
	if targetErr != nil {
		return nil
	}

	return []Amendment{{
		Type:             AmendRepeal,
		TargetTitle:      targetTitle,
		TargetSection:    targetSection,
		TargetSubsection: targetSubsection,
		Description:      truncateDescription(normalizedText),
	}}
}

// findPreambleStart walks backward from the next "is amended" anchor to
// find where the preamble paragraph for that anchor begins (typically a
// subsection marker like "(b) IN GENERAL.--").
func findPreambleStart(text string, nextAnchorStart int) int {
	// Walk backward to find the line that starts this paragraph
	pos := nextAnchorStart
	for pos > 0 && text[pos-1] != '\n' {
		pos--
	}
	// Continue backward through continuation lines (lines that start with
	// non-whitespace or are part of the same paragraph)
	for pos > 0 {
		// Find the previous line
		prevLineEnd := pos - 1 // skip the \n
		prevLineStart := prevLineEnd
		for prevLineStart > 0 && text[prevLineStart-1] != '\n' {
			prevLineStart--
		}
		prevLine := strings.TrimSpace(text[prevLineStart:prevLineEnd])
		// If blank line, stop — this is a paragraph boundary
		if prevLine == "" {
			break
		}
		// If this line looks like a subsection start "(b) " or "(a) IN GENERAL"
		// it's the start of the preamble
		if len(prevLine) > 0 && prevLine[0] == '(' {
			pos = prevLineStart
			break
		}
		// Otherwise it's a continuation line — keep going
		pos = prevLineStart
	}
	return pos
}

// splitNumberedClauses splits text at "(1) ", "(2) " etc. boundaries.
func (recognizer *Recognizer) splitNumberedClauses(text string) []numberedClause {
	var clauses []numberedClause
	allMatches := recognizer.numberedClausePattern.FindAllStringSubmatchIndex(text, -1)

	if len(allMatches) == 0 {
		return nil
	}

	for matchIndex, matchLocation := range allMatches {
		clauseNumber := text[matchLocation[2]:matchLocation[3]]
		clauseStart := matchLocation[0]
		var clauseEnd int
		if matchIndex+1 < len(allMatches) {
			clauseEnd = allMatches[matchIndex+1][0]
		} else {
			clauseEnd = len(text)
		}

		clauseText := strings.TrimSpace(text[clauseStart:clauseEnd])
		clauses = append(clauses, numberedClause{
			number: clauseNumber,
			text:   clauseText,
		})
	}

	return clauses
}

// splitLetteredSubItems splits text at "(A) ", "(B) " etc. boundaries.
func (recognizer *Recognizer) splitLetteredSubItems(text string) []string {
	allMatches := recognizer.letteredClausePattern.FindAllStringIndex(text, -1)

	if len(allMatches) == 0 {
		return nil
	}

	var subItems []string
	for matchIndex, matchLocation := range allMatches {
		subItemStart := matchLocation[0]
		var subItemEnd int
		if matchIndex+1 < len(allMatches) {
			subItemEnd = allMatches[matchIndex+1][0]
		} else {
			subItemEnd = len(text)
		}
		subItems = append(subItems, strings.TrimSpace(text[subItemStart:subItemEnd]))
	}

	return subItems
}

// processNumberedClauses handles a multi-amendment block with numbered
// sub-items. Each clause may itself contain lettered sub-items that
// further refine the scope (e.g., (1) in subsection (b)-- (A)... (B)...).
func (recognizer *Recognizer) processNumberedClauses(clauses []numberedClause, targetTitle, targetSection, targetSubsection string) []Amendment {
	var amendments []Amendment

	for _, clause := range clauses {
		// Check if clause introduces a subsection scope
		subsectionScope := recognizer.inSubsectionPattern.FindStringSubmatch(clause.text)

		if subsectionScope != nil {
			scopedSubsection := "(" + subsectionScope[1] + ")"

			// Split into lettered sub-items
			subItems := recognizer.splitLetteredSubItems(clause.text)
			if len(subItems) > 0 {
				for _, subItem := range subItems {
					paragraphRef := recognizer.extractParagraphRef(subItem)
					fullSubsection := scopedSubsection
					if paragraphRef != "" {
						fullSubsection += paragraphRef
					}
					amendment := recognizer.buildAmendment(subItem, targetTitle, targetSection, fullSubsection)
					if amendment.Type != "" {
						amendments = append(amendments, amendment)
					}
				}
			} else {
				// No lettered sub-items, treat clause as single amendment
				amendment := recognizer.buildAmendment(clause.text, targetTitle, targetSection, scopedSubsection)
				if amendment.Type != "" {
					amendments = append(amendments, amendment)
				}
			}
		} else {
			// Direct amendment clause without subsection scope
			amendment := recognizer.buildAmendment(clause.text, targetTitle, targetSection, targetSubsection)
			if amendment.Type != "" {
				amendments = append(amendments, amendment)
			}
		}
	}

	return amendments
}

// extractParagraphRef extracts a paragraph reference like "(1)" from
// "in paragraph (1), by striking...".
func (recognizer *Recognizer) extractParagraphRef(text string) string {
	paragraphMatch := recognizer.paragraphRefPattern.FindStringSubmatch(text)
	if paragraphMatch != nil {
		return "(" + paragraphMatch[1] + ")"
	}
	return ""
}

// buildAmendment creates an Amendment struct from clause text and target
// information. It classifies the amendment type and extracts strike/insert
// text where applicable.
func (recognizer *Recognizer) buildAmendment(clauseText, targetTitle, targetSection, targetSubsection string) Amendment {
	amendmentType := recognizer.ClassifyAmendmentType(clauseText)

	amendment := Amendment{
		Type:             amendmentType,
		TargetTitle:      targetTitle,
		TargetSection:    targetSection,
		TargetSubsection: targetSubsection,
		Description:      truncateDescription(clauseText),
	}

	// Extract strike/insert text for strike-insert amendments
	if amendmentType == AmendStrikeInsert {
		normalizedClause := normalizeAmendmentText(clauseText)
		if strikeMatch := recognizer.strikeInsertPattern.FindStringSubmatch(normalizedClause); strikeMatch != nil {
			amendment.StrikeText = strikeMatch[1]
			amendment.InsertText = strikeMatch[2]
		}
	}

	// Extract inserted text for add-at-end and add-new-section amendments
	if amendmentType == AmendAddAtEnd || amendmentType == AmendAddNewSection {
		amendment.InsertText = extractQuotedInsertText(clauseText)
	}

	// Extract redesignation details
	if amendmentType == AmendRedesignate {
		normalizedClause := normalizeAmendmentText(clauseText)
		if redesignateMatch := recognizer.redesignatePattern.FindStringSubmatch(normalizedClause); redesignateMatch != nil {
			amendment.StrikeText = "(" + redesignateMatch[1] + ")"
			amendment.InsertText = "(" + redesignateMatch[2] + ")"
		}
	}

	return amendment
}

// normalizeAmendmentText collapses runs of whitespace (including newlines)
// into single spaces for pattern matching.
func normalizeAmendmentText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// extractQuotedInsertText extracts the text content that follows a colon
// and is within quotes, handling multi-line quoted legislative text.
func extractQuotedInsertText(text string) string {
	// Find the colon that precedes the quoted text
	colonIndex := strings.LastIndex(text, ":")
	if colonIndex < 0 {
		colonIndex = 0
	}
	afterColon := text[colonIndex:]

	// Normalize and extract the first quoted block
	normalizedText := normalizeAmendmentText(afterColon)
	quotePattern := regexp.MustCompile(`["\x{201c}](.+)["\x{201d}]`)
	quotedMatch := quotePattern.FindStringSubmatch(normalizedText)
	if quotedMatch != nil {
		return quotedMatch[1]
	}
	return ""
}

// truncateDescription creates a short description from the clause text,
// trimming to a reasonable length.
func truncateDescription(text string) string {
	normalized := normalizeAmendmentText(text)
	if len(normalized) > 200 {
		return normalized[:200] + "..."
	}
	return normalized
}
