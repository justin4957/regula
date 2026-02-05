package extract

import (
	"regexp"
	"strings"
)

var (
	// verDatePattern matches GPO VerDate metadata lines.
	verDatePattern = regexp.MustCompile(`^VerDate\s`)

	// runningHeaderPattern matches running headers like "Rule XX, clause N Rule XX, clause N"
	// that appear at page boundaries in the PDF-extracted text.
	runningHeaderPattern = regexp.MustCompile(`^Rule\s+[IVXLCDM]+,\s+clause\s+\d+\s+Rule\s+[IVXLCDM]+,\s+clause\s+\d+`)

	// standalonePageNumberPattern matches lines containing only a page number.
	standalonePageNumberPattern = regexp.MustCompile(`^\d+\s*$`)

	// hyphenatedLineEndPattern matches lines ending with a hyphen (word break across lines).
	hyphenatedLineEndPattern = regexp.MustCompile(`[a-zA-Z]-$`)
)

// PreprocessHouseRules cleans up PDF-extracted House Rules text by removing
// GPO metadata lines, running headers, standalone page numbers, repeated
// page headers, and rejoining hyphenated words split across line breaks.
func PreprocessHouseRules(lines []string) []string {
	// Track whether we've passed the document title section to distinguish
	// the real title from repeated page headers.
	passedTitle := false
	var cleanedLines []string

	for lineIndex := 0; lineIndex < len(lines); lineIndex++ {
		line := lines[lineIndex]
		trimmedLine := strings.TrimSpace(line)

		// Skip VerDate metadata lines
		if verDatePattern.MatchString(trimmedLine) {
			continue
		}

		// Skip running headers (e.g., "Rule XX, clause N Rule XX, clause N")
		if runningHeaderPattern.MatchString(trimmedLine) {
			continue
		}

		// Skip standalone page numbers
		if standalonePageNumberPattern.MatchString(trimmedLine) {
			continue
		}

		// Mark when we've passed the initial title section (first RULE header)
		if !passedTitle && strings.HasPrefix(trimmedLine, "RULE ") {
			ruleHeaderPattern := regexp.MustCompile(`^RULE\s+[IVXLCDM]+\s*$`)
			if ruleHeaderPattern.MatchString(trimmedLine) {
				passedTitle = true
			}
		}

		// Skip repeated page headers after the title section
		if passedTitle {
			if trimmedLine == "RULES OF THE" || trimmedLine == "HOUSE OF REPRESENTATIVES" {
				continue
			}
		}

		cleanedLines = append(cleanedLines, line)
	}

	// Second pass: rejoin orphan uppercase fragments from PDF column breaks
	// (e.g., "O" + "THER OFFICERS AND OFFICIALS" → "OTHER OFFICERS AND OFFICIALS")
	cleanedLines = rejoinOrphanFragments(cleanedLines)

	// Third pass: rejoin hyphenated words split across line breaks
	cleanedLines = rejoinHyphenatedLines(cleanedLines)

	return cleanedLines
}

// rejoinOrphanFragments merges short orphan uppercase fragments that result
// from PDF column breaks. For example:
//
//	"O"
//	"THER OFFICERS AND OFFICIALS"
//
// becomes:
//
//	"OTHER OFFICERS AND OFFICIALS"
func rejoinOrphanFragments(lines []string) []string {
	if len(lines) < 2 {
		return lines
	}

	var result []string
	for i := 0; i < len(lines); i++ {
		trimmedCurrent := strings.TrimSpace(lines[i])

		// Check for short orphan fragments (1-2 uppercase characters on their own line)
		if i+1 < len(lines) && len(trimmedCurrent) <= 2 && len(trimmedCurrent) > 0 {
			allUpper := true
			for _, ch := range trimmedCurrent {
				if ch < 'A' || ch > 'Z' {
					allUpper = false
					break
				}
			}

			nextTrimmed := strings.TrimSpace(lines[i+1])
			// Join if the next line continues with uppercase text
			if allUpper && len(nextTrimmed) > 0 && nextTrimmed[0] >= 'A' && nextTrimmed[0] <= 'Z' {
				joined := trimmedCurrent + nextTrimmed
				result = append(result, joined)
				i++ // Skip the next line
				continue
			}
		}

		result = append(result, lines[i])
	}

	return result
}

// rejoinHyphenatedLines merges lines where a word is split across a line
// break with a hyphen. For example:
//
//	"the House last ad-"
//	"journed and immediately"
//
// becomes:
//
//	"the House last adjourned and immediately"
func rejoinHyphenatedLines(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}

	var result []string
	for i := 0; i < len(lines); i++ {
		currentLine := lines[i]
		trimmedCurrent := strings.TrimRight(currentLine, " \t")

		// Check if the line ends with a hyphenated word break
		if i+1 < len(lines) && hyphenatedLineEndPattern.MatchString(trimmedCurrent) {
			nextLine := lines[i+1]
			trimmedNext := strings.TrimSpace(nextLine)

			// Only rejoin if the next line starts with a lowercase letter
			// (indicating a word continuation, not a new sentence or heading)
			if len(trimmedNext) > 0 && trimmedNext[0] >= 'a' && trimmedNext[0] <= 'z' {
				// Remove the trailing hyphen and append the next line's content
				joined := trimmedCurrent[:len(trimmedCurrent)-1] + trimmedNext
				result = append(result, joined)
				i++ // Skip the next line since it's been merged
				continue
			}
		}

		result = append(result, currentLine)
	}

	return result
}
