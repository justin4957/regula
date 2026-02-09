// Package deliberation provides types and functions for modeling deliberation
// documents including meetings, agendas, decisions, and their evolution over time.
package deliberation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MinutesParser extracts structured deliberation data from meeting minutes text.
// It supports multiple formats including EU Council working groups, UN bodies,
// and generic meeting minutes.
type MinutesParser struct {
	// BaseURI is the base URI for generated entity URIs.
	BaseURI string

	// Format hints at the expected document format (eu, un, generic).
	Format string

	// patterns holds compiled regex patterns for extraction.
	patterns *minutesPatterns
}

// minutesPatterns holds compiled regex patterns for meeting minutes parsing.
type minutesPatterns struct {
	// Meeting metadata patterns
	datePatterns       []*regexp.Regexp
	timePatterns       []*regexp.Regexp
	locationPattern    *regexp.Regexp
	meetingNumPattern  *regexp.Regexp
	sessionPattern     *regexp.Regexp
	seriesPattern      *regexp.Regexp

	// Participant patterns
	chairPattern       *regexp.Regexp
	attendeesPattern   *regexp.Regexp
	apologyPattern     *regexp.Regexp

	// Agenda patterns
	agendaItemPatterns []*regexp.Regexp
	subItemPattern     *regexp.Regexp

	// Deliberation patterns
	speakerPatterns    []*regexp.Regexp
	motionPattern      *regexp.Regexp
	amendmentPattern   *regexp.Regexp
	secondedPattern    *regexp.Regexp

	// Decision patterns
	decisionPatterns   []*regexp.Regexp
	votePattern        *regexp.Regexp
	adoptedPattern     *regexp.Regexp
	rejectedPattern    *regexp.Regexp

	// Action patterns
	actionPatterns     []*regexp.Regexp

	// Reference patterns
	documentRefPattern *regexp.Regexp
	articleRefPattern  *regexp.Regexp
	meetingRefPattern  *regexp.Regexp
}

// NewMinutesParser creates a new parser with default patterns.
func NewMinutesParser(baseURI string) *MinutesParser {
	return &MinutesParser{
		BaseURI:  baseURI,
		Format:   "generic",
		patterns: compileDefaultPatterns(),
	}
}

// NewMinutesParserWithFormat creates a parser optimized for a specific format.
func NewMinutesParserWithFormat(baseURI, format string) *MinutesParser {
	p := &MinutesParser{
		BaseURI: baseURI,
		Format:  format,
	}

	switch format {
	case "eu":
		p.patterns = compileEUPatterns()
	case "un":
		p.patterns = compileUNPatterns()
	default:
		p.patterns = compileDefaultPatterns()
	}

	return p
}

// Parse extracts a complete Meeting structure from meeting minutes text.
func (p *MinutesParser) Parse(source string) (*Meeting, error) {
	if source == "" {
		return nil, fmt.Errorf("empty source text")
	}

	meeting := &Meeting{
		Status: MeetingCompleted,
	}

	// Extract meeting metadata
	if err := p.extractMetadata(source, meeting); err != nil {
		// Non-fatal: continue with partial data
	}

	// Generate URI if not set
	if meeting.URI == "" {
		meeting.URI = p.generateMeetingURI(meeting)
	}

	// Extract participants
	p.extractParticipants(source, meeting)

	// Extract agenda items
	agendaItems, err := p.ExtractAgenda(source)
	if err == nil {
		meeting.AgendaItems = agendaItems
	}

	// Extract decisions and attach to agenda items
	decisions, _ := p.ExtractDecisions(source)
	p.attachDecisionsToAgenda(meeting, decisions)

	// Extract interventions and attach to agenda items
	interventions, _ := p.ExtractSpeakers(source)
	p.attachInterventionsToAgenda(meeting, interventions)

	// Extract action items and attach to agenda items
	actions, _ := p.ExtractActions(source)
	p.attachActionsToAgenda(meeting, actions)

	return meeting, nil
}

// extractMetadata extracts date, time, location, and meeting identifiers.
func (p *MinutesParser) extractMetadata(source string, meeting *Meeting) error {
	// Extract date
	for _, pattern := range p.patterns.datePatterns {
		if match := pattern.FindStringSubmatch(source); match != nil {
			if date, err := p.parseDate(match); err == nil {
				meeting.Date = date
				break
			}
		}
	}

	// Extract time
	for _, pattern := range p.patterns.timePatterns {
		if match := pattern.FindStringSubmatch(source); match != nil {
			if t, err := p.parseTime(match); err == nil {
				meeting.StartTime = &t
				break
			}
		}
	}

	// Extract location
	if match := p.patterns.locationPattern.FindStringSubmatch(source); match != nil {
		meeting.Location = strings.TrimSpace(match[1])
	}

	// Extract meeting number/sequence
	if match := p.patterns.meetingNumPattern.FindStringSubmatch(source); match != nil {
		if num, err := strconv.Atoi(match[1]); err == nil {
			meeting.Sequence = num
		}
	}

	// Extract session/series
	if match := p.patterns.sessionPattern.FindStringSubmatch(source); match != nil {
		meeting.Identifier = strings.TrimSpace(match[0])
	}

	if match := p.patterns.seriesPattern.FindStringSubmatch(source); match != nil {
		meeting.Series = strings.TrimSpace(match[1])
	}

	// Extract title from first substantial line if not set
	if meeting.Title == "" {
		lines := strings.Split(source, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if len(line) > 10 && len(line) < 200 {
				meeting.Title = line
				break
			}
		}
	}

	return nil
}

// extractParticipants extracts chair, attendees, and apologies.
func (p *MinutesParser) extractParticipants(source string, meeting *Meeting) {
	// Extract chair
	if match := p.patterns.chairPattern.FindStringSubmatch(source); match != nil {
		meeting.ChairName = strings.TrimSpace(match[1])
		meeting.Chair = p.generateStakeholderURI(meeting.ChairName)
	}

	// Extract attendees
	if match := p.patterns.attendeesPattern.FindStringSubmatch(source); match != nil {
		attendeeText := match[1]
		attendees := p.parseNameList(attendeeText)
		for _, name := range attendees {
			meeting.Participants = append(meeting.Participants, p.generateStakeholderURI(name))
		}
	}
}

// ExtractAgenda parses agenda items from the meeting minutes.
func (p *MinutesParser) ExtractAgenda(text string) ([]AgendaItem, error) {
	var items []AgendaItem
	seenNumbers := make(map[string]bool)

	for _, pattern := range p.patterns.agendaItemPatterns {
		matches := pattern.FindAllStringSubmatchIndex(text, -1)
		for i, match := range matches {
			if len(match) < 4 {
				continue
			}

			number := text[match[2]:match[3]]
			if seenNumbers[number] {
				continue
			}
			seenNumbers[number] = true

			// Determine title: either from capture group or following text
			var title string
			if len(match) >= 6 && match[4] != -1 {
				title = strings.TrimSpace(text[match[4]:match[5]])
			} else {
				// Extract title from text following the number
				endPos := match[1]
				titleEnd := endPos + 200
				if titleEnd > len(text) {
					titleEnd = len(text)
				}
				remaining := text[endPos:titleEnd]
				if nlPos := strings.Index(remaining, "\n"); nlPos > 0 {
					title = strings.TrimSpace(remaining[:nlPos])
				} else {
					title = strings.TrimSpace(remaining)
				}
			}

			// Determine content boundaries
			contentStart := match[1]
			var contentEnd int
			if i+1 < len(matches) {
				contentEnd = matches[i+1][0]
			} else {
				contentEnd = len(text)
			}
			content := text[contentStart:contentEnd]

			item := AgendaItem{
				URI:     p.generateAgendaItemURI(number),
				Number:  number,
				Title:   truncateString(title, 200),
				Outcome: p.detectOutcome(content),
			}

			items = append(items, item)
		}
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no agenda items found")
	}

	return items, nil
}

// ExtractDecisions parses decisions from the meeting minutes.
func (p *MinutesParser) ExtractDecisions(text string) ([]Decision, error) {
	var decisions []Decision
	now := time.Now()

	for _, pattern := range p.patterns.decisionPatterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for i, match := range matches {
			if len(match) < 2 {
				continue
			}

			description := strings.TrimSpace(match[1])
			if len(description) < 10 {
				continue
			}

			// Detect decision type
			decisionType := "decision"
			lowerDesc := strings.ToLower(description)
			if strings.Contains(lowerDesc, "adopt") {
				decisionType = "adoption"
			} else if strings.Contains(lowerDesc, "reject") {
				decisionType = "rejection"
			} else if strings.Contains(lowerDesc, "defer") {
				decisionType = "deferral"
			} else if strings.Contains(lowerDesc, "amend") {
				decisionType = "amendment"
			}

			decision := Decision{
				URI:         p.generateDecisionURI(i + 1),
				Identifier:  fmt.Sprintf("Decision %d", i+1),
				Title:       truncateString(description, 100),
				Description: description,
				Type:        decisionType,
				DecidedAt:   now,
			}

			// Try to extract vote information
			if vote := p.extractVoteFromContext(text, match[0]); vote != nil {
				decision.VoteURI = vote.URI
			}

			decisions = append(decisions, decision)
		}
	}

	// Also check for adopted/rejected patterns
	if match := p.patterns.adoptedPattern.FindStringSubmatch(text); match != nil {
		// Check if this is already captured
		found := false
		for _, d := range decisions {
			if strings.Contains(d.Description, match[1]) {
				found = true
				break
			}
		}
		if !found && len(match) > 1 {
			decisions = append(decisions, Decision{
				URI:         p.generateDecisionURI(len(decisions) + 1),
				Identifier:  fmt.Sprintf("Decision %d", len(decisions)+1),
				Title:       truncateString(match[1], 100),
				Description: match[1],
				Type:        "adoption",
				DecidedAt:   now,
			})
		}
	}

	return decisions, nil
}

// ExtractSpeakers parses speaker interventions from the meeting minutes.
func (p *MinutesParser) ExtractSpeakers(text string) ([]Intervention, error) {
	var interventions []Intervention
	sequence := 0

	for _, pattern := range p.patterns.speakerPatterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}

			speakerName := strings.TrimSpace(match[1])
			if speakerName == "" {
				continue
			}

			// Get the context after the speaker mention
			idx := strings.Index(text, match[0])
			var summary string
			if idx >= 0 {
				contextEnd := idx + len(match[0]) + 500
				if contextEnd > len(text) {
					contextEnd = len(text)
				}
				context := text[idx+len(match[0]) : contextEnd]
				if nlPos := strings.Index(context, "\n\n"); nlPos > 0 {
					summary = strings.TrimSpace(context[:nlPos])
				} else if nlPos := strings.Index(context, "\n"); nlPos > 0 {
					summary = strings.TrimSpace(context[:nlPos])
				}
			}

			sequence++
			intervention := Intervention{
				URI:         p.generateInterventionURI(sequence),
				SpeakerName: speakerName,
				SpeakerURI:  p.generateStakeholderURI(speakerName),
				Summary:     truncateString(summary, 500),
				Position:    p.detectPosition(summary),
				Sequence:    sequence,
			}

			interventions = append(interventions, intervention)
		}
	}

	return interventions, nil
}

// ExtractActions parses action items from the meeting minutes.
func (p *MinutesParser) ExtractActions(text string) ([]ActionItem, error) {
	var actions []ActionItem
	count := 0

	for _, pattern := range p.patterns.actionPatterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}

			description := strings.TrimSpace(match[1])
			if len(description) < 10 {
				continue
			}

			count++
			action := ActionItem{
				URI:         p.generateActionURI(count),
				Identifier:  fmt.Sprintf("Action %d", count),
				Description: description,
				Status:      ActionPending,
			}

			// Try to extract assignee
			if len(match) > 2 && match[2] != "" {
				assignee := strings.TrimSpace(match[2])
				action.AssignedToNames = []string{assignee}
				action.AssignedToURIs = []string{p.generateStakeholderURI(assignee)}
			}

			// Try to extract due date
			if dueDate := p.extractDueDate(description); dueDate != nil {
				action.DueDate = dueDate
			}

			actions = append(actions, action)
		}
	}

	return actions, nil
}

// ExtractReferences finds cross-references to documents and regulations.
func (p *MinutesParser) ExtractReferences(text string) []DocumentReference {
	var refs []DocumentReference

	// Document references (e.g., "document WG/2024/05")
	if matches := p.patterns.documentRefPattern.FindAllStringSubmatch(text, -1); matches != nil {
		for _, match := range matches {
			if len(match) >= 2 {
				refs = append(refs, DocumentReference{
					Type:       "document",
					Identifier: match[1],
				})
			}
		}
	}

	// Article references (e.g., "Article 5", "Articles 12-15")
	if matches := p.patterns.articleRefPattern.FindAllStringSubmatch(text, -1); matches != nil {
		for _, match := range matches {
			if len(match) >= 2 {
				refs = append(refs, DocumentReference{
					Type:       "article",
					Identifier: match[1],
				})
			}
		}
	}

	// Meeting references (e.g., "previous meeting", "meeting of 15 May")
	if matches := p.patterns.meetingRefPattern.FindAllStringSubmatch(text, -1); matches != nil {
		for _, match := range matches {
			if len(match) >= 1 {
				refs = append(refs, DocumentReference{
					Type:       "meeting",
					Identifier: match[0],
				})
			}
		}
	}

	return refs
}

// DocumentReference represents a reference to an external document.
type DocumentReference struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
	URI        string `json:"uri,omitempty"`
}

// Helper methods

func (p *MinutesParser) parseDate(match []string) (time.Time, error) {
	// Try various date formats
	formats := []string{
		"2 January 2006",
		"January 2, 2006",
		"02/01/2006",
		"2006-01-02",
		"02-01-2006",
		"2.1.2006",
		"02 Jan 2006",
	}

	dateStr := strings.TrimSpace(match[0])
	// Clean up the date string
	dateStr = regexp.MustCompile(`\s+`).ReplaceAllString(dateStr, " ")

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	// Try to extract components
	if len(match) >= 4 {
		day, _ := strconv.Atoi(match[1])
		month := p.parseMonth(match[2])
		year, _ := strconv.Atoi(match[3])
		if day > 0 && month > 0 && year > 0 {
			return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse date: %s", dateStr)
}

func (p *MinutesParser) parseMonth(s string) int {
	months := map[string]int{
		"january": 1, "jan": 1,
		"february": 2, "feb": 2,
		"march": 3, "mar": 3,
		"april": 4, "apr": 4,
		"may": 5,
		"june": 6, "jun": 6,
		"july": 7, "jul": 7,
		"august": 8, "aug": 8,
		"september": 9, "sep": 9, "sept": 9,
		"october": 10, "oct": 10,
		"november": 11, "nov": 11,
		"december": 12, "dec": 12,
	}
	return months[strings.ToLower(s)]
}

func (p *MinutesParser) parseTime(match []string) (time.Time, error) {
	if len(match) < 2 {
		return time.Time{}, fmt.Errorf("invalid time match")
	}

	timeStr := strings.TrimSpace(match[1])
	formats := []string{
		"15:04",
		"3:04 PM",
		"3:04PM",
		"15.04",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse time: %s", timeStr)
}

func (p *MinutesParser) parseNameList(text string) []string {
	var names []string

	// Split by common delimiters
	delimiters := regexp.MustCompile(`[,;]\s*|\s+and\s+`)
	parts := delimiters.Split(text, -1)

	for _, part := range parts {
		name := strings.TrimSpace(part)
		// Filter out common non-name items
		if name != "" && len(name) > 2 && !strings.HasPrefix(strings.ToLower(name), "mr") &&
			!strings.HasPrefix(strings.ToLower(name), "ms") &&
			!strings.HasPrefix(strings.ToLower(name), "dr") {
			names = append(names, name)
		} else if name != "" && len(name) > 2 {
			// Remove title prefix
			name = regexp.MustCompile(`^(?i)(mr|ms|mrs|dr|prof)\.?\s+`).ReplaceAllString(name, "")
			if name != "" {
				names = append(names, name)
			}
		}
	}

	return names
}

func (p *MinutesParser) detectOutcome(content string) AgendaItemOutcome {
	lower := strings.ToLower(content)

	if strings.Contains(lower, "decision") || strings.Contains(lower, "adopted") ||
		strings.Contains(lower, "agreed") || strings.Contains(lower, "approved") {
		return OutcomeDecided
	}
	if strings.Contains(lower, "deferred") || strings.Contains(lower, "postponed") {
		return OutcomeDeferred
	}
	if strings.Contains(lower, "withdrawn") {
		return OutcomeWithdrawn
	}
	if strings.Contains(lower, "no quorum") {
		return OutcomeNoQuorum
	}
	if strings.Contains(lower, "discussed") || strings.Contains(lower, "debate") {
		return OutcomeDiscussed
	}

	return OutcomePending
}

func (p *MinutesParser) detectPosition(text string) InterventionPosition {
	lower := strings.ToLower(text)

	// Check qualified first as it may contain "support" with qualifications
	if strings.Contains(lower, "reserv") || strings.Contains(lower, "conditional") ||
		strings.Contains(lower, "subject to") ||
		(strings.Contains(lower, "support") && strings.Contains(lower, "with")) {
		return PositionQualified
	}
	if strings.Contains(lower, "support") || strings.Contains(lower, "favour") ||
		strings.Contains(lower, "favor") || strings.Contains(lower, "agree") ||
		strings.Contains(lower, "welcome") {
		return PositionSupport
	}
	if strings.Contains(lower, "oppose") || strings.Contains(lower, "against") ||
		strings.Contains(lower, "reject") || strings.Contains(lower, "concern") {
		return PositionOppose
	}
	if strings.Contains(lower, "question") || strings.Contains(lower, "clarif") ||
		strings.Contains(lower, "ask") {
		return PositionQuestion
	}
	if strings.Contains(lower, "procedure") || strings.Contains(lower, "order") {
		return PositionProcedural
	}

	return PositionNeutral
}

func (p *MinutesParser) extractVoteFromContext(text, decisionText string) *VoteRecord {
	// Find the position of the decision
	idx := strings.Index(text, decisionText)
	if idx < 0 {
		return nil
	}

	// Look for vote pattern nearby
	contextStart := idx - 200
	if contextStart < 0 {
		contextStart = 0
	}
	contextEnd := idx + len(decisionText) + 200
	if contextEnd > len(text) {
		contextEnd = len(text)
	}
	context := text[contextStart:contextEnd]

	match := p.patterns.votePattern.FindStringSubmatch(context)
	if match == nil {
		return nil
	}

	vote := &VoteRecord{
		URI:      p.generateVoteURI(1),
		VoteDate: time.Now(),
		VoteType: "recorded",
	}

	// Parse vote counts
	if len(match) >= 4 {
		vote.ForCount, _ = strconv.Atoi(match[1])
		vote.AgainstCount, _ = strconv.Atoi(match[2])
		vote.AbstainCount, _ = strconv.Atoi(match[3])
	}

	if vote.ForCount > vote.AgainstCount {
		vote.Result = "adopted"
	} else if vote.ForCount < vote.AgainstCount {
		vote.Result = "rejected"
	} else {
		vote.Result = "tie"
	}

	return vote
}

func (p *MinutesParser) extractDueDate(text string) *time.Time {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)by\s+(\d{1,2})\s+(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{4})`),
		regexp.MustCompile(`(?i)deadline[:\s]+(\d{1,2})[/\-](\d{1,2})[/\-](\d{4})`),
		regexp.MustCompile(`(?i)due[:\s]+(\d{1,2})\s+(January|February|March|April|May|June|July|August|September|October|November|December)`),
	}

	for _, pattern := range patterns {
		if match := pattern.FindStringSubmatch(text); match != nil {
			if date, err := p.parseDate(match); err == nil {
				return &date
			}
		}
	}

	return nil
}

// URI generation helpers

func (p *MinutesParser) generateMeetingURI(m *Meeting) string {
	if m.Identifier != "" {
		return fmt.Sprintf("%s/meetings/%s", p.BaseURI, sanitizeForURI(m.Identifier))
	}
	if !m.Date.IsZero() {
		return fmt.Sprintf("%s/meetings/%s", p.BaseURI, m.Date.Format("2006-01-02"))
	}
	return fmt.Sprintf("%s/meetings/unknown", p.BaseURI)
}

func (p *MinutesParser) generateAgendaItemURI(number string) string {
	return fmt.Sprintf("%s/agenda/%s", p.BaseURI, sanitizeForURI(number))
}

func (p *MinutesParser) generateDecisionURI(seq int) string {
	return fmt.Sprintf("%s/decisions/%d", p.BaseURI, seq)
}

func (p *MinutesParser) generateInterventionURI(seq int) string {
	return fmt.Sprintf("%s/interventions/%d", p.BaseURI, seq)
}

func (p *MinutesParser) generateActionURI(seq int) string {
	return fmt.Sprintf("%s/actions/%d", p.BaseURI, seq)
}

func (p *MinutesParser) generateVoteURI(seq int) string {
	return fmt.Sprintf("%s/votes/%d", p.BaseURI, seq)
}

func (p *MinutesParser) generateStakeholderURI(name string) string {
	return fmt.Sprintf("%s/stakeholders/%s", p.BaseURI, sanitizeForURI(name))
}

// Attachment helpers

func (p *MinutesParser) attachDecisionsToAgenda(meeting *Meeting, decisions []Decision) {
	if len(meeting.AgendaItems) == 0 || len(decisions) == 0 {
		return
	}

	// Simple heuristic: distribute decisions across agenda items
	// In a real implementation, this would use text proximity
	for i := range decisions {
		agendaIdx := i % len(meeting.AgendaItems)
		decisions[i].AgendaItemURI = meeting.AgendaItems[agendaIdx].URI
		decisions[i].MeetingURI = meeting.URI
		meeting.AgendaItems[agendaIdx].Decisions = append(meeting.AgendaItems[agendaIdx].Decisions, decisions[i])
	}
}

func (p *MinutesParser) attachInterventionsToAgenda(meeting *Meeting, interventions []Intervention) {
	if len(meeting.AgendaItems) == 0 || len(interventions) == 0 {
		return
	}

	for i := range interventions {
		interventions[i].MeetingURI = meeting.URI
		agendaIdx := i % len(meeting.AgendaItems)
		interventions[i].AgendaItemURI = meeting.AgendaItems[agendaIdx].URI
		meeting.AgendaItems[agendaIdx].Interventions = append(meeting.AgendaItems[agendaIdx].Interventions, interventions[i])
	}
}

func (p *MinutesParser) attachActionsToAgenda(meeting *Meeting, actions []ActionItem) {
	if len(meeting.AgendaItems) == 0 || len(actions) == 0 {
		return
	}

	for i := range actions {
		actions[i].AssignedAtMeetingURI = meeting.URI
		agendaIdx := i % len(meeting.AgendaItems)
		actions[i].AgendaItemURI = meeting.AgendaItems[agendaIdx].URI
		meeting.AgendaItems[agendaIdx].ActionItems = append(meeting.AgendaItems[agendaIdx].ActionItems, actions[i])
	}
}

// Utility functions

func sanitizeForURI(s string) string {
	// Replace spaces and special chars with hyphens
	result := strings.ToLower(s)
	result = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(result, "-")
	result = strings.Trim(result, "-")
	return result
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
