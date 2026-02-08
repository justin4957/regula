// Package extract provides document extraction and parsing utilities.
package extract

import (
	"regexp"
	"sort"
	"strings"
)

// KeywordMatch represents a search result for a keyword query.
type KeywordMatch struct {
	// Rule is the rule number (e.g., "XX", "XVIII").
	Rule string

	// RuleTitle is the rule title (e.g., "Voting and Quorum Calls").
	RuleTitle string

	// Clause is the clause number (e.g., "5", "6").
	Clause string

	// ClauseTitle is the clause title if available.
	ClauseTitle string

	// Text is the matching text content.
	Text string

	// Context is a snippet showing the keyword in context.
	Context string

	// Score is the relevance score (higher is better).
	Score int

	// MatchCount is the number of keyword occurrences.
	MatchCount int
}

// RuleClause represents a parsed rule and clause from House Rules.
type RuleClause struct {
	// Rule is the rule number (Roman numeral).
	Rule string

	// RuleTitle is the rule title.
	RuleTitle string

	// Clause is the clause number.
	Clause string

	// ClauseTitle is the clause title.
	ClauseTitle string

	// Text is the full clause text.
	Text string
}

// KeywordSearcher provides procedural keyword search over House Rules.
type KeywordSearcher struct {
	// clauses contains all parsed rule clauses.
	clauses []RuleClause

	// rulePattern matches rule headers like "RULE XX".
	rulePattern *regexp.Regexp

	// clausePattern matches clause headers like "clause 5" or numbered clauses.
	clausePattern *regexp.Regexp

	// titlePattern matches clause titles (capitalized phrases before numbered content).
	titlePattern *regexp.Regexp
}

// ProceduralKeywords contains pre-built query templates for common concepts.
var ProceduralKeywords = map[string][]string{
	"voting": {
		"vote", "voting", "yea", "nay", "ballot", "roll call", "voice vote",
		"recorded vote", "automatic roll call", "electronic device",
	},
	"quorum": {
		"quorum", "quorum call", "absence of a quorum", "Committee of the Whole",
	},
	"amendments": {
		"amendment", "amend", "substitute", "motion to strike", "germane",
		"first degree", "second degree", "en bloc",
	},
	"debate": {
		"debate", "recognition", "yield", "time", "five-minute rule",
		"one-minute", "special order", "morning hour",
	},
	"appropriations": {
		"appropriation", "appropriations", "authorization", "budget",
		"revenue", "rescission", "earmark",
	},
	"motions": {
		"motion", "previous question", "recommit", "reconsider", "table",
		"postpone", "refer", "discharge",
	},
	"committees": {
		"committee", "subcommittee", "jurisdiction", "referral", "markup",
		"hearing", "report", "discharge petition",
	},
	"speaker": {
		"Speaker", "pro tempore", "Chair", "presiding officer", "recognition",
	},
	"calendar": {
		"calendar", "Union Calendar", "House Calendar", "Private Calendar",
		"Consent Calendar", "special rule",
	},
	"privilege": {
		"privilege", "privileged", "question of privilege", "personal privilege",
		"constitutional question",
	},
}

// NewKeywordSearcher creates a new keyword searcher.
func NewKeywordSearcher() *KeywordSearcher {
	return &KeywordSearcher{
		clauses: []RuleClause{},
		// Match: RULE XX or RULE XVIII etc.
		rulePattern: regexp.MustCompile(`(?m)^RULE\s+([IVXLCDM]+)\s*$`),
		// Match: clause header with number, e.g., "1. The Speaker shall"
		clausePattern: regexp.MustCompile(`(?m)^\s*(\d+)\.\s*`),
		// Match: clause titles (Capitalized words before the clause text)
		titlePattern: regexp.MustCompile(`(?m)^([A-Z][a-z]+(?:\s+[a-z]+)*(?:\s+[A-Z][a-z]+)*)\s*$`),
	}
}

// ParseHouseRules parses House Rules text into searchable clauses.
func (k *KeywordSearcher) ParseHouseRules(text string) {
	// Only fix hyphenated line breaks (keep newlines for structure detection)
	text = normalizeHyphensOnly(text)

	// Find all rule boundaries
	ruleMatches := k.rulePattern.FindAllStringSubmatchIndex(text, -1)
	if len(ruleMatches) == 0 {
		return
	}

	// Build rule title map from table of contents
	ruleTitles := k.extractRuleTitles(text)

	// Process each rule section
	for i, match := range ruleMatches {
		if len(match) < 4 {
			continue
		}

		ruleNum := text[match[2]:match[3]]
		ruleStart := match[1]

		// Find end of this rule (start of next rule or end of document)
		var ruleEnd int
		if i+1 < len(ruleMatches) {
			ruleEnd = ruleMatches[i+1][0]
		} else {
			ruleEnd = len(text)
		}

		ruleText := text[ruleStart:ruleEnd]
		ruleTitle := ruleTitles[ruleNum]

		// Extract clauses from this rule section
		k.extractClauses(ruleNum, ruleTitle, ruleText)
	}
}

// extractRuleTitles extracts rule titles from the table of contents.
func (k *KeywordSearcher) extractRuleTitles(text string) map[string]string {
	titles := make(map[string]string)

	// Find "C O N T E N T S" section
	contentStart := strings.Index(text, "C O N T E N T S")
	if contentStart == -1 {
		return titles
	}

	// Parse title lines like "RULE I.— The Speaker"
	titlePattern := regexp.MustCompile(`(?i)(?:RULE\s+)?([IVXLCDM]+)\.?[—–-]+\s*([^\d\n]+)`)
	contentEnd := strings.Index(text[contentStart:], "RULE I")
	if contentEnd == -1 {
		contentEnd = 2000 // Limit search
	} else {
		contentEnd += contentStart
	}

	contentSection := text[contentStart:contentEnd]
	matches := titlePattern.FindAllStringSubmatch(contentSection, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			ruleNum := strings.TrimSpace(match[1])
			title := strings.TrimSpace(match[2])
			titles[ruleNum] = title
		}
	}

	return titles
}

// extractClauses extracts individual clauses from a rule section.
func (k *KeywordSearcher) extractClauses(ruleNum, ruleTitle, ruleText string) {
	// Find clause titles (lines that appear to be headers)
	// Look for patterns like "Approval of the Journal" followed by a numbered clause
	clauseTitlePattern := regexp.MustCompile(`(?m)^([A-Z][a-z]+(?:[\s,]+(?:of|the|and|in|on|to|a|an|or|for|by|with|from|at|as|is|it)?\s*)?(?:[a-zA-Z]+\s*)*)\s*$`)

	// Find clause boundaries using numbered patterns like "1. ", "2. "
	clauseNumPattern := regexp.MustCompile(`(?m)(?:^|\s)(\d+)\.\s+(?:\([a-z]\)|[A-Z])`)

	clauseMatches := clauseNumPattern.FindAllStringSubmatchIndex(ruleText, -1)
	if len(clauseMatches) == 0 {
		// If no numbered clauses found, treat whole rule as one clause
		k.clauses = append(k.clauses, RuleClause{
			Rule:      ruleNum,
			RuleTitle: ruleTitle,
			Clause:    "1",
			Text:      cleanText(ruleText),
		})
		return
	}

	// Build a map of clause titles from text before each clause
	clauseTitles := make(map[string]string)
	for i, match := range clauseMatches {
		clauseNum := ruleText[match[2]:match[3]]

		// Look for title in text before this clause
		var searchStart int
		if i > 0 {
			searchStart = clauseMatches[i-1][1]
		} else {
			searchStart = 0
		}
		searchEnd := match[0]

		if searchEnd > searchStart {
			precedingText := ruleText[searchStart:searchEnd]
			titleMatches := clauseTitlePattern.FindAllStringSubmatch(precedingText, -1)
			if len(titleMatches) > 0 {
				// Take the last title match as the clause title
				lastMatch := titleMatches[len(titleMatches)-1]
				if len(lastMatch) >= 2 {
					clauseTitles[clauseNum] = strings.TrimSpace(lastMatch[1])
				}
			}
		}
	}

	// Process each clause
	for i, match := range clauseMatches {
		if len(match) < 4 {
			continue
		}

		clauseNum := ruleText[match[2]:match[3]]
		clauseStart := match[1]

		// Find end of this clause
		var clauseEnd int
		if i+1 < len(clauseMatches) {
			// Look for the title before the next clause
			clauseEnd = clauseMatches[i+1][0]
		} else {
			clauseEnd = len(ruleText)
		}

		clauseText := cleanText(ruleText[clauseStart:clauseEnd])

		k.clauses = append(k.clauses, RuleClause{
			Rule:        ruleNum,
			RuleTitle:   ruleTitle,
			Clause:      clauseNum,
			ClauseTitle: clauseTitles[clauseNum],
			Text:        clauseText,
		})
	}
}

// Search searches for a keyword across all parsed clauses.
func (k *KeywordSearcher) Search(keyword string) []KeywordMatch {
	var matches []KeywordMatch
	keywordLower := strings.ToLower(keyword)

	// Check if it's a predefined procedural keyword
	relatedTerms := []string{keywordLower}
	if terms, ok := ProceduralKeywords[keywordLower]; ok {
		relatedTerms = append(relatedTerms, terms...)
	}

	for _, clause := range k.clauses {
		textLower := strings.ToLower(clause.Text)

		// Count matches for all related terms
		matchCount := 0
		score := 0
		var contextSnippet string

		for _, term := range relatedTerms {
			termLower := strings.ToLower(term)
			count := strings.Count(textLower, termLower)
			if count > 0 {
				matchCount += count
				// Primary keyword gets higher weight
				if termLower == keywordLower {
					score += count * 10
				} else {
					score += count * 2
				}

				// Extract context for first match if we don't have one yet
				if contextSnippet == "" {
					contextSnippet = extractKeywordContext(clause.Text, term, 50)
				}
			}
		}

		if matchCount > 0 {
			// Boost score for title matches
			titleLower := strings.ToLower(clause.ClauseTitle)
			ruleTitleLower := strings.ToLower(clause.RuleTitle)
			for _, term := range relatedTerms {
				termLower := strings.ToLower(term)
				if strings.Contains(titleLower, termLower) {
					score += 20
				}
				if strings.Contains(ruleTitleLower, termLower) {
					score += 15
				}
			}

			matches = append(matches, KeywordMatch{
				Rule:        clause.Rule,
				RuleTitle:   clause.RuleTitle,
				Clause:      clause.Clause,
				ClauseTitle: clause.ClauseTitle,
				Text:        clause.Text,
				Context:     contextSnippet,
				Score:       score,
				MatchCount:  matchCount,
			})
		}
	}

	// Sort by score (highest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	return matches
}

// SearchWithTemplate searches using a predefined procedural keyword template.
func (k *KeywordSearcher) SearchWithTemplate(templateName string) []KeywordMatch {
	var allMatches []KeywordMatch
	seenClauses := make(map[string]bool)

	terms, ok := ProceduralKeywords[templateName]
	if !ok {
		// Fall back to direct search
		return k.Search(templateName)
	}

	for _, term := range terms {
		matches := k.Search(term)
		for _, match := range matches {
			clauseKey := match.Rule + ":" + match.Clause
			if !seenClauses[clauseKey] {
				seenClauses[clauseKey] = true
				allMatches = append(allMatches, match)
			}
		}
	}

	// Sort by score
	sort.Slice(allMatches, func(i, j int) bool {
		return allMatches[i].Score > allMatches[j].Score
	})

	return allMatches
}

// GetClauses returns all parsed clauses.
func (k *KeywordSearcher) GetClauses() []RuleClause {
	return k.clauses
}

// GetTemplates returns the available procedural keyword templates.
func GetTemplates() []string {
	templates := make([]string, 0, len(ProceduralKeywords))
	for name := range ProceduralKeywords {
		templates = append(templates, name)
	}
	sort.Strings(templates)
	return templates
}

// normalizeHyphensOnly fixes hyphenated line breaks without removing newlines.
func normalizeHyphensOnly(text string) string {
	// Replace hyphen followed by newline and whitespace with empty string
	re := regexp.MustCompile(`-\s*\n\s*`)
	return re.ReplaceAllString(text, "")
}

// extractKeywordContext extracts a snippet of text surrounding a keyword match.
func extractKeywordContext(text, keyword string, contextChars int) string {
	textLower := strings.ToLower(text)
	keywordLower := strings.ToLower(keyword)

	idx := strings.Index(textLower, keywordLower)
	if idx == -1 {
		return ""
	}

	start := idx - contextChars
	if start < 0 {
		start = 0
	}

	end := idx + len(keyword) + contextChars
	if end > len(text) {
		end = len(text)
	}

	snippet := text[start:end]

	// Add ellipses if truncated
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(text) {
		snippet = snippet + "..."
	}

	// Clean up whitespace
	snippet = cleanText(snippet)

	return snippet
}
