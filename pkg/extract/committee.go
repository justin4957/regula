// Package extract provides document extraction and parsing utilities.
package extract

import (
	"regexp"
	"strings"
)

// CommitteeJurisdiction represents a committee and its assigned jurisdiction topics.
type CommitteeJurisdiction struct {
	// Name is the official committee name (e.g., "Committee on Agriculture").
	Name string

	// ShortName is the short form (e.g., "Agriculture").
	ShortName string

	// Letter is the rule letter (e.g., "a" for Committee on Agriculture).
	Letter string

	// Topics is the list of jurisdiction topics.
	Topics []JurisdictionTopic

	// SourceClause is the source reference (e.g., "Rule X, clause 1(a)").
	SourceClause string
}

// JurisdictionTopic represents a single jurisdiction topic.
type JurisdictionTopic struct {
	// Number is the topic number within the committee (e.g., "1", "2", etc.).
	Number string

	// Text is the jurisdiction description text.
	Text string

	// SubTopics contains nested sub-topics if any (for hierarchical topics like 3(A), 3(B)).
	SubTopics []JurisdictionTopic
}

// CommitteeJurisdictionExtractor extracts committee jurisdictions from House Rules.
type CommitteeJurisdictionExtractor struct {
	// committeePattern matches committee headers like "(a) Committee on Agriculture."
	committeePattern *regexp.Regexp

	// topicPattern matches numbered topics like "(1) Agriculture generally."
	topicPattern *regexp.Regexp

	// subTopicPattern matches lettered sub-topics like "(A) Border and port security"
	subTopicPattern *regexp.Regexp
}

// NewCommitteeJurisdictionExtractor creates a new extractor.
func NewCommitteeJurisdictionExtractor() *CommitteeJurisdictionExtractor {
	return &CommitteeJurisdictionExtractor{
		// Match: (a) Committee on Agriculture. or (a) Committee on the Judiciary.
		// Handle multi-word names that may span lines with hyphenation
		committeePattern: regexp.MustCompile(`(?i)\(([a-z])\)\s+Committee\s+on\s+(?:the\s+)?`),

		// Match: (1) Topic text here. - handles multi-line topics
		topicPattern: regexp.MustCompile(`\((\d+)\)\s+`),

		// Match: (A) Sub-topic text.
		subTopicPattern: regexp.MustCompile(`\(([A-G])\)\s+([^(]+?)(?:\.|$)`),
	}
}

// ExtractFromRuleX extracts committee jurisdictions from Rule X, clause 1 text.
func (e *CommitteeJurisdictionExtractor) ExtractFromRuleX(text string) []CommitteeJurisdiction {
	var committees []CommitteeJurisdiction

	// First, normalize the text by joining hyphenated line breaks
	normalizedText := normalizeHyphenatedLines(text)

	// Find all committee sections
	matches := e.committeePattern.FindAllStringSubmatchIndex(normalizedText, -1)
	if len(matches) == 0 {
		return committees
	}

	for i, match := range matches {
		if len(match) < 4 {
			continue
		}

		letter := normalizedText[match[2]:match[3]]
		headerEnd := match[1]

		// Find the committee name - it ends with a period or the next numbered topic
		nameStart := headerEnd
		nameEnd := nameStart

		// Look for the end of the committee name (period followed by newline or numbered topic)
		remaining := normalizedText[nameStart:]
		periodIdx := strings.Index(remaining, ".")
		topicIdx := strings.Index(remaining, "(1)")

		if periodIdx > 0 && (topicIdx < 0 || periodIdx < topicIdx) {
			nameEnd = nameStart + periodIdx
		} else if topicIdx > 0 {
			nameEnd = nameStart + topicIdx
		} else {
			nameEnd = nameStart + 50 // Fallback
			if nameEnd > len(normalizedText) {
				nameEnd = len(normalizedText)
			}
		}

		committeeName := cleanText(normalizedText[nameStart:nameEnd])

		// Determine section boundaries
		sectionStart := match[1]
		var sectionEnd int
		if i+1 < len(matches) {
			sectionEnd = matches[i+1][0]
		} else {
			// Look for end of clause 1 or next clause
			clauseEnd := strings.Index(normalizedText[sectionStart:], "clause 2")
			if clauseEnd > 0 {
				sectionEnd = sectionStart + clauseEnd
			} else {
				sectionEnd = len(normalizedText)
			}
		}

		sectionText := normalizedText[sectionStart:sectionEnd]

		// Extract topics from this section
		topics := e.extractTopics(sectionText)

		trimmedName := strings.TrimSpace(committeeName)
		var fullName string
		// Handle "the" prefix
		if strings.HasPrefix(strings.ToLower(trimmedName), "the ") {
			fullName = "Committee on " + trimmedName
		} else {
			fullName = "Committee on " + trimmedName
		}

		committees = append(committees, CommitteeJurisdiction{
			Name:         fullName,
			ShortName:    strings.TrimPrefix(strings.TrimPrefix(committeeName, "the "), "The "),
			Letter:       letter,
			Topics:       topics,
			SourceClause: "Rule X, clause 1(" + letter + ")",
		})
	}

	return committees
}

// normalizeHyphenatedLines joins hyphenated line breaks common in PDF text.
func normalizeHyphenatedLines(text string) string {
	// Replace hyphen followed by newline and whitespace with empty string
	re := regexp.MustCompile(`-\s*\n\s*`)
	text = re.ReplaceAllString(text, "")

	// Also replace remaining newlines with spaces for easier parsing
	text = strings.ReplaceAll(text, "\n", " ")

	return text
}

// extractTopics extracts numbered jurisdiction topics from a committee section.
func (e *CommitteeJurisdictionExtractor) extractTopics(sectionText string) []JurisdictionTopic {
	var topics []JurisdictionTopic

	// Find all topic boundaries
	matches := e.topicPattern.FindAllStringSubmatchIndex(sectionText, -1)

	for i, match := range matches {
		if len(match) < 4 {
			continue
		}

		number := sectionText[match[2]:match[3]]
		topicStart := match[1] // End of the "(N) " pattern

		// Determine the end of this topic
		var topicEnd int
		if i+1 < len(matches) {
			topicEnd = matches[i+1][0]
		} else {
			topicEnd = len(sectionText)
		}

		topicText := sectionText[topicStart:topicEnd]

		// Clean up the text (remove line breaks and excessive whitespace)
		fullText := cleanText(topicText)

		// Check for sub-topics
		subTopics := e.extractSubTopics(fullText)

		topics = append(topics, JurisdictionTopic{
			Number:    number,
			Text:      fullText,
			SubTopics: subTopics,
		})
	}

	return topics
}

// extractSubTopics extracts lettered sub-topics from topic text.
func (e *CommitteeJurisdictionExtractor) extractSubTopics(topicText string) []JurisdictionTopic {
	var subTopics []JurisdictionTopic

	matches := e.subTopicPattern.FindAllStringSubmatch(topicText, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			subTopics = append(subTopics, JurisdictionTopic{
				Number: match[1],
				Text:   cleanText(match[2]),
			})
		}
	}

	return subTopics
}

// cleanText removes excessive whitespace and line breaks.
func cleanText(s string) string {
	// Replace multiple whitespace with single space
	re := regexp.MustCompile(`\s+`)
	s = re.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// SearchCommitteeByTopic finds committees that have jurisdiction over a topic.
// Uses case-insensitive partial matching.
func SearchCommitteeByTopic(committees []CommitteeJurisdiction, query string) []CommitteeJurisdictionMatch {
	var matches []CommitteeJurisdictionMatch

	queryLower := strings.ToLower(query)

	for _, committee := range committees {
		for _, topic := range committee.Topics {
			if strings.Contains(strings.ToLower(topic.Text), queryLower) {
				matches = append(matches, CommitteeJurisdictionMatch{
					Committee:    committee,
					MatchedTopic: topic,
					SourceRef:    committee.SourceClause + "(" + topic.Number + ")",
				})
			}

			// Also check sub-topics
			for _, subTopic := range topic.SubTopics {
				if strings.Contains(strings.ToLower(subTopic.Text), queryLower) {
					matches = append(matches, CommitteeJurisdictionMatch{
						Committee:    committee,
						MatchedTopic: subTopic,
						SourceRef:    committee.SourceClause + "(" + topic.Number + ")(" + subTopic.Number + ")",
					})
				}
			}
		}
	}

	return matches
}

// SearchCommitteeByName finds a committee by name (partial match).
func SearchCommitteeByName(committees []CommitteeJurisdiction, query string) *CommitteeJurisdiction {
	queryLower := strings.ToLower(query)

	for _, committee := range committees {
		if strings.Contains(strings.ToLower(committee.Name), queryLower) ||
			strings.Contains(strings.ToLower(committee.ShortName), queryLower) {
			return &committee
		}
	}

	return nil
}

// CommitteeJurisdictionMatch represents a search match result.
type CommitteeJurisdictionMatch struct {
	// Committee is the matched committee.
	Committee CommitteeJurisdiction

	// MatchedTopic is the specific topic that matched.
	MatchedTopic JurisdictionTopic

	// SourceRef is the full source reference (e.g., "Rule X, clause 1(j)(4)").
	SourceRef string
}

// GetJurisdictions returns a committee's jurisdiction topics as a list of strings.
func (c *CommitteeJurisdiction) GetJurisdictions() []string {
	var jurisdictions []string
	for _, topic := range c.Topics {
		jurisdictions = append(jurisdictions, topic.Text)
		for _, subTopic := range topic.SubTopics {
			jurisdictions = append(jurisdictions, subTopic.Text)
		}
	}
	return jurisdictions
}
