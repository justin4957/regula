// Package deliberation provides types and functions for modeling deliberation
// documents including meetings, agendas, decisions, and their evolution over time.
package deliberation

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// ActionExtractor parses meeting minutes and extracts action items,
// tracking assignments, due dates, and completion status across meetings.
type ActionExtractor struct {
	// patterns for detecting action items
	patterns *actionPatterns

	// store for persisting action items
	store *store.TripleStore

	// baseURI for generating action URIs
	baseURI string

	// actionCounter for generating unique IDs
	actionCounter int
}

// actionPatterns contains compiled regex patterns for action item detection.
type actionPatterns struct {
	// Explicit action markers
	explicitAction []*regexp.Regexp

	// Assignment patterns (X to do Y)
	assignmentPatterns []*regexp.Regexp

	// Request patterns (X was asked/requested to...)
	requestPatterns []*regexp.Regexp

	// Agreement patterns (agreed that X would...)
	agreementPatterns []*regexp.Regexp

	// Due date patterns
	dueDatePatterns []*regexp.Regexp

	// Completion patterns (action X completed/done)
	completionPatterns []*regexp.Regexp

	// Deferral patterns (action X deferred/postponed)
	deferralPatterns []*regexp.Regexp

	// Still pending patterns
	pendingPatterns []*regexp.Regexp

	// Reference patterns (provision references in action text)
	provisionRefPatterns []*regexp.Regexp

	// Assignee extraction
	assigneePatterns []*regexp.Regexp
}

// NewActionExtractor creates a new action item extractor.
func NewActionExtractor(tripleStore *store.TripleStore, baseURI string) *ActionExtractor {
	return &ActionExtractor{
		patterns:      compileActionPatterns(),
		store:         tripleStore,
		baseURI:       baseURI,
		actionCounter: 0,
	}
}

// compileActionPatterns creates the pattern set for action item extraction.
func compileActionPatterns() *actionPatterns {
	return &actionPatterns{
		// Explicit action markers like [ACTION] or Action:
		explicitAction: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\[ACTION\][:\s]*(.+?)(?:\n|$)`),
			regexp.MustCompile(`(?i)^ACTION[:\s]+(.+?)(?:\n|$)`),
			regexp.MustCompile(`(?i)Action\s+(?:Item\s+)?(\d+)[:\s]+(.+?)(?:\n|$)`),
			regexp.MustCompile(`(?i)(?:A\d+|AI\d+)[:\s]+(.+?)(?:\n|$)`),
		},

		// Assignment patterns: "X to do Y" or "X will do Y"
		assignmentPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)([A-Z][a-zA-Z\s.'-]+?)\s+(?:to|will|shall)\s+(review|prepare|draft|submit|circulate|provide|propose|develop|update|revise|coordinate|organize|contact|report|analyse|analyze|assess|evaluate|consider|examine|investigate|clarify|confirm)(.+?)(?:\.|;|\n|$)`),
			regexp.MustCompile(`(?i)([A-Z][a-zA-Z\s.'-]+?)\s+is\s+(?:to|expected\s+to)\s+(.+?)(?:\.|;|\n|$)`),
		},

		// Request patterns
		requestPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:The\s+)?([A-Z][a-zA-Z\s.'-]+?)\s+(?:was|is|were)\s+(?:asked|requested|invited|instructed|tasked)\s+to\s+(.+?)(?:\.|;|\n|$)`),
			regexp.MustCompile(`(?i)(?:asked|requested|invited|instructed)\s+(?:the\s+)?([A-Z][a-zA-Z\s.'-]+?)\s+to\s+(.+?)(?:\.|;|\n|$)`),
			regexp.MustCompile(`(?i)(?:Member\s+States|Delegations|Participants)\s+(?:are|were)\s+(?:asked|requested|invited)\s+to\s+(.+?)(?:\.|;|\n|$)`),
		},

		// Agreement patterns: "agreed that X would..."
		agreementPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:agreed|decided)\s+that\s+(?:the\s+)?([A-Z][a-zA-Z\s.'-]+?)\s+(?:would|should|will)\s+(.+?)(?:\.|;|\n|$)`),
			regexp.MustCompile(`(?i)(?:The\s+)?(?:committee|group|council|board)\s+agreed\s+(?:that\s+)?(?:the\s+)?([A-Z][a-zA-Z\s.'-]+?)\s+(?:would|should)\s+(.+?)(?:\.|;|\n|$)`),
		},

		// Due date patterns
		dueDatePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)by\s+(\d{1,2})\s+(January|February|March|April|May|June|July|August|September|October|November|December)(?:\s+(\d{4}))?`),
			regexp.MustCompile(`(?i)by\s+(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{1,2})(?:,?\s+(\d{4}))?`),
			regexp.MustCompile(`(?i)by\s+(\d{4}-\d{2}-\d{2})`),
			regexp.MustCompile(`(?i)by\s+(Friday|Monday|Tuesday|Wednesday|Thursday|Saturday|Sunday)`),
			regexp.MustCompile(`(?i)by\s+(?:the\s+)?(?:next|following)\s+meeting`),
			regexp.MustCompile(`(?i)by\s+(?:end\s+of\s+)?(?:this\s+)?(week|month|quarter|year)`),
			regexp.MustCompile(`(?i)within\s+(\d+)\s+(days?|weeks?|months?)`),
			regexp.MustCompile(`(?i)before\s+(\d{1,2})\s+(January|February|March|April|May|June|July|August|September|October|November|December)`),
		},

		// Completion patterns
		completionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:action|item)\s+(?:(?:no\.?\s*)?(\d+|[A-Z]\d+))\s+(?:has\s+been\s+)?(?:completed|done|finished|closed|resolved)`),
			regexp.MustCompile(`(?i)(?:completed|done|finished|closed|resolved)[:\s]+(?:action|item)\s+(?:(?:no\.?\s*)?(\d+|[A-Z]\d+))`),
			regexp.MustCompile(`(?i)([A-Z][a-zA-Z\s.'-]+?)\s+(?:has\s+)?(?:completed|finished|submitted|provided|delivered)\s+(.+?)(?:\.|;|\n|$)`),
			regexp.MustCompile(`(?i)(?:The\s+)?(?:report|document|proposal|analysis)\s+(?:was|has\s+been)\s+(?:submitted|provided|circulated|delivered)`),
		},

		// Deferral patterns
		deferralPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:action|item)\s+(?:(?:no\.?\s*)?(\d+|[A-Z]\d+))\s+(?:was\s+)?(?:deferred|postponed|delayed|carried\s+over)`),
			regexp.MustCompile(`(?i)(?:deferred|postponed|delayed|carried\s+over)\s+(?:to\s+)?(?:the\s+)?(?:next|following)\s+meeting`),
			regexp.MustCompile(`(?i)(?:action|item)\s+(?:(?:no\.?\s*)?(\d+|[A-Z]\d+))\s+(?:to\s+be\s+)?(?:deferred|postponed)\s+(?:to|until)\s+(.+?)(?:\.|;|\n|$)`),
		},

		// Still pending patterns
		pendingPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:action|item)\s+(?:(?:no\.?\s*)?(\d+|[A-Z]\d+))\s+(?:is\s+)?(?:still\s+)?(?:pending|outstanding|open|ongoing)`),
			regexp.MustCompile(`(?i)(?:still\s+)?(?:pending|awaiting|waiting\s+for)[:\s]+(.+?)(?:\.|;|\n|$)`),
			regexp.MustCompile(`(?i)(?:action|item)\s+(?:(?:no\.?\s*)?(\d+|[A-Z]\d+))\s+(?:remains?\s+)?(?:open|outstanding)`),
		},

		// Provision reference patterns
		provisionRefPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)Article\s+(\d+(?:\(\d+\))?(?:\([a-z]\))?)`),
			regexp.MustCompile(`(?i)(?:Section|Chapter|Part)\s+(\d+|[IVXLCDM]+)`),
			regexp.MustCompile(`(?i)(?:paragraph|para\.?)\s+(\d+(?:\.\d+)?)`),
			regexp.MustCompile(`(?i)(?:Rule|Regulation)\s+(\d+(?:\.\d+)?)`),
		},

		// Assignee extraction patterns
		assigneePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^(?:The\s+)?([A-Z][a-zA-Z\s.'-]+?)(?:\s+(?:delegation|representative))?$`),
			regexp.MustCompile(`(?i)^(Secretariat|Secretary|Chair|President|Rapporteur|Commission|Council|Presidency)$`),
			regexp.MustCompile(`(?i)^(?:Mr|Ms|Mrs|Dr|Prof)\.?\s+([A-Z][a-zA-Z\s.'-]+)$`),
			regexp.MustCompile(`(?i)^(Member\s+States?|Delegations?|Participants?)$`),
		},
	}
}

// ExtractedAction represents an action item extracted from text.
type ExtractedAction struct {
	// Text is the full action item text
	Text string

	// Assignees are the entities responsible for the action
	Assignees []string

	// DueDate is the deadline (if specified)
	DueDate *time.Time

	// DueDateText is the raw due date text
	DueDateText string

	// RelatedProvisions are provision references found in the text
	RelatedProvisions []string

	// Priority indicates urgency
	Priority string

	// SourceOffset is where in the text this action was found
	SourceOffset int

	// MatchType indicates how the action was detected
	MatchType string
}

// ActionTracker manages action items across meetings, tracking status changes.
type ActionTracker struct {
	extractor *ActionExtractor
	store     *store.TripleStore
	baseURI   string

	// actionIndex maps action URIs to their current state
	actionIndex map[string]*ActionItem

	// meetingActions maps meeting URIs to their action items
	meetingActions map[string][]string
}

// NewActionTracker creates a new action tracker.
func NewActionTracker(tripleStore *store.TripleStore, baseURI string) *ActionTracker {
	return &ActionTracker{
		extractor:      NewActionExtractor(tripleStore, baseURI),
		store:          tripleStore,
		baseURI:        baseURI,
		actionIndex:    make(map[string]*ActionItem),
		meetingActions: make(map[string][]string),
	}
}

// ExtractActions extracts all action items from text.
func (e *ActionExtractor) ExtractActions(text string, meetingURI string) ([]ExtractedAction, error) {
	var actions []ExtractedAction

	// Try explicit action markers first
	for _, pattern := range e.patterns.explicitAction {
		matches := pattern.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			if len(match) >= 4 {
				actionText := strings.TrimSpace(text[match[2]:match[3]])
				action := e.buildExtractedAction(actionText, match[0], "explicit")
				actions = append(actions, action)
			}
		}
	}

	// Try assignment patterns
	for _, pattern := range e.patterns.assignmentPatterns {
		matches := pattern.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			if len(match) >= 6 {
				assignee := strings.TrimSpace(text[match[2]:match[3]])
				verb := strings.TrimSpace(text[match[4]:match[5]])
				rest := ""
				if len(match) >= 8 && match[6] != -1 {
					rest = strings.TrimSpace(text[match[6]:match[7]])
				}
				actionText := fmt.Sprintf("%s to %s%s", assignee, verb, rest)
				action := e.buildExtractedAction(actionText, match[0], "assignment")
				action.Assignees = []string{assignee}
				actions = append(actions, action)
			}
		}
	}

	// Try request patterns
	for _, pattern := range e.patterns.requestPatterns {
		matches := pattern.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			if len(match) >= 6 {
				assignee := ""
				actionText := ""
				if match[2] != -1 && match[3] != -1 {
					assignee = strings.TrimSpace(text[match[2]:match[3]])
				}
				if match[4] != -1 && match[5] != -1 {
					actionText = strings.TrimSpace(text[match[4]:match[5]])
				}
				if actionText == "" && assignee != "" {
					actionText = assignee
					assignee = ""
				}
				fullText := actionText
				if assignee != "" {
					fullText = fmt.Sprintf("%s to %s", assignee, actionText)
				}
				action := e.buildExtractedAction(fullText, match[0], "request")
				if assignee != "" {
					action.Assignees = []string{assignee}
				}
				actions = append(actions, action)
			}
		}
	}

	// Try agreement patterns
	for _, pattern := range e.patterns.agreementPatterns {
		matches := pattern.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			if len(match) >= 6 {
				assignee := strings.TrimSpace(text[match[2]:match[3]])
				actionText := strings.TrimSpace(text[match[4]:match[5]])
				fullText := fmt.Sprintf("%s to %s", assignee, actionText)
				action := e.buildExtractedAction(fullText, match[0], "agreement")
				action.Assignees = []string{assignee}
				actions = append(actions, action)
			}
		}
	}

	// Deduplicate by source offset
	actions = e.deduplicateActions(actions)

	return actions, nil
}

// buildExtractedAction creates an ExtractedAction from text.
func (e *ActionExtractor) buildExtractedAction(text string, offset int, matchType string) ExtractedAction {
	action := ExtractedAction{
		Text:         text,
		SourceOffset: offset,
		MatchType:    matchType,
	}

	// Extract due date from patterns
	for _, pattern := range e.patterns.dueDatePatterns {
		if match := pattern.FindStringSubmatch(text); match != nil {
			action.DueDateText = match[0]
			action.DueDate = e.parseDueDate(match)
			break
		}
	}

	// If no due date found from patterns, check for relative dates directly
	if action.DueDate == nil {
		action.DueDate = e.parseRelativeDate(text)
	}

	// Extract provision references
	for _, pattern := range e.patterns.provisionRefPatterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				action.RelatedProvisions = append(action.RelatedProvisions, match[0])
			}
		}
	}

	// Detect priority
	action.Priority = e.detectPriority(text)

	return action
}

// parseRelativeDate checks text for relative date references and returns a due date.
func (e *ActionExtractor) parseRelativeDate(text string) *time.Time {
	lower := strings.ToLower(text)
	now := time.Now()

	// Check for "next meeting" or "following meeting"
	if strings.Contains(lower, "next meeting") || strings.Contains(lower, "following meeting") {
		t := now.AddDate(0, 0, 14) // Default 2 weeks
		return &t
	}

	// Check for "within X days/weeks/months"
	withinPattern := regexp.MustCompile(`(?i)within\s+(\d+)\s+(days?|weeks?|months?)`)
	if m := withinPattern.FindStringSubmatch(text); m != nil {
		num := parseIntFromString(m[1])
		unit := strings.ToLower(m[2])
		var t time.Time
		switch {
		case strings.HasPrefix(unit, "day"):
			t = now.AddDate(0, 0, num)
		case strings.HasPrefix(unit, "week"):
			t = now.AddDate(0, 0, num*7)
		case strings.HasPrefix(unit, "month"):
			t = now.AddDate(0, num, 0)
		}
		return &t
	}

	return nil
}

// parseDueDate attempts to parse a due date from regex match groups.
func (e *ActionExtractor) parseDueDate(match []string) *time.Time {
	if len(match) < 2 {
		return nil
	}

	// Try different date formats based on match structure
	fullMatch := match[0]

	// ISO format: 2024-01-15
	if t, err := time.Parse("2006-01-02", fullMatch); err == nil {
		return &t
	}

	// Try parsing "by X Month Year" format
	monthNames := map[string]time.Month{
		"january": time.January, "february": time.February, "march": time.March,
		"april": time.April, "may": time.May, "june": time.June,
		"july": time.July, "august": time.August, "september": time.September,
		"october": time.October, "november": time.November, "december": time.December,
	}

	for i := 1; i < len(match); i++ {
		if month, ok := monthNames[strings.ToLower(match[i])]; ok {
			// Found month, look for day and year
			day := 1
			year := time.Now().Year()
			for j := 1; j < len(match); j++ {
				if j != i {
					if d := parseIntFromString(match[j]); d > 0 && d <= 31 {
						day = d
					} else if y := parseIntFromString(match[j]); y >= 2020 && y <= 2100 {
						year = y
					}
				}
			}
			t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
			return &t
		}
	}

	// Handle relative dates
	now := time.Now()
	lowerMatch := strings.ToLower(fullMatch)

	if strings.Contains(lowerMatch, "next meeting") || strings.Contains(lowerMatch, "following meeting") {
		// Default to 2 weeks
		t := now.AddDate(0, 0, 14)
		return &t
	}

	// Handle "within X days/weeks/months"
	withinPattern := regexp.MustCompile(`(?i)within\s+(\d+)\s+(days?|weeks?|months?)`)
	if m := withinPattern.FindStringSubmatch(fullMatch); m != nil {
		num := parseIntFromString(m[1])
		unit := strings.ToLower(m[2])
		var t time.Time
		switch {
		case strings.HasPrefix(unit, "day"):
			t = now.AddDate(0, 0, num)
		case strings.HasPrefix(unit, "week"):
			t = now.AddDate(0, 0, num*7)
		case strings.HasPrefix(unit, "month"):
			t = now.AddDate(0, num, 0)
		}
		return &t
	}

	return nil
}

// parseIntFromString parses a string to int, returning 0 on error.
func parseIntFromString(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

// detectPriority infers priority from action text.
func (e *ActionExtractor) detectPriority(text string) string {
	lower := strings.ToLower(text)

	// High priority indicators
	highPriority := []string{
		"urgent", "urgently", "immediately", "asap", "as soon as possible",
		"critical", "priority", "essential", "vital", "imperative",
	}
	for _, indicator := range highPriority {
		if strings.Contains(lower, indicator) {
			return "high"
		}
	}

	// Low priority indicators
	lowPriority := []string{
		"when possible", "if time permits", "as resources allow",
		"non-urgent", "low priority",
	}
	for _, indicator := range lowPriority {
		if strings.Contains(lower, indicator) {
			return "low"
		}
	}

	return "medium"
}

// deduplicateActions removes duplicate actions by source offset proximity.
func (e *ActionExtractor) deduplicateActions(actions []ExtractedAction) []ExtractedAction {
	if len(actions) <= 1 {
		return actions
	}

	// Sort by offset
	sort.Slice(actions, func(i, j int) bool {
		return actions[i].SourceOffset < actions[j].SourceOffset
	})

	var deduplicated []ExtractedAction
	for i, action := range actions {
		isDuplicate := false
		// Check if this action overlaps with a previous one
		for j := 0; j < i; j++ {
			prev := actions[j]
			// If offsets are within 50 chars and text is similar, it's a duplicate
			if abs(action.SourceOffset-prev.SourceOffset) < 50 {
				if stringSimilarity(action.Text, prev.Text) > 0.5 {
					isDuplicate = true
					break
				}
			}
		}
		if !isDuplicate {
			deduplicated = append(deduplicated, action)
		}
	}

	return deduplicated
}

// abs returns absolute value.
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// stringSimilarity returns a rough similarity score between two strings.
func stringSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}
	a = strings.ToLower(a)
	b = strings.ToLower(b)

	// Simple word overlap
	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)

	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0.0
	}

	overlap := 0
	for _, wa := range wordsA {
		for _, wb := range wordsB {
			if wa == wb {
				overlap++
				break
			}
		}
	}

	return float64(overlap) / float64(max(len(wordsA), len(wordsB)))
}

// max returns the larger of two ints.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// CreateActionItem creates a new ActionItem from an ExtractedAction.
func (e *ActionExtractor) CreateActionItem(extracted ExtractedAction, meetingURI string) *ActionItem {
	e.actionCounter++
	identifier := fmt.Sprintf("Action-%d", e.actionCounter)
	uri := fmt.Sprintf("%s/actions/%s", e.baseURI, identifier)

	action := NewActionItem(uri, identifier, extracted.Text, extracted.Assignees, meetingURI)
	action.DueDate = extracted.DueDate
	action.Priority = extracted.Priority

	// Convert provision references to URIs
	for _, ref := range extracted.RelatedProvisions {
		provURI := fmt.Sprintf("%s/provisions/%s", e.baseURI, strings.ReplaceAll(ref, " ", "_"))
		action.RelatedProvisionURIs = append(action.RelatedProvisionURIs, provURI)
	}

	return action
}

// DetectStatusUpdate checks text for status updates to existing actions.
func (t *ActionTracker) DetectStatusUpdate(text string, meetingURI string) []ActionStatusUpdate {
	var updates []ActionStatusUpdate

	// Check for completion patterns
	for _, pattern := range t.extractor.patterns.completionPatterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			update := ActionStatusUpdate{
				NewStatus:  ActionCompleted,
				MeetingURI: meetingURI,
				Note:       match[0],
			}
			if len(match) > 1 && match[1] != "" {
				update.ActionRef = match[1]
			}
			updates = append(updates, update)
		}
	}

	// Check for deferral patterns
	for _, pattern := range t.extractor.patterns.deferralPatterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			update := ActionStatusUpdate{
				NewStatus:  ActionDeferred,
				MeetingURI: meetingURI,
				Note:       match[0],
			}
			if len(match) > 1 && match[1] != "" {
				update.ActionRef = match[1]
			}
			updates = append(updates, update)
		}
	}

	// Check for pending patterns
	for _, pattern := range t.extractor.patterns.pendingPatterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			update := ActionStatusUpdate{
				NewStatus:  ActionPending,
				MeetingURI: meetingURI,
				Note:       match[0],
			}
			if len(match) > 1 && match[1] != "" {
				update.ActionRef = match[1]
			}
			updates = append(updates, update)
		}
	}

	return updates
}

// ActionStatusUpdate represents a detected status change.
type ActionStatusUpdate struct {
	// ActionRef is the action identifier (e.g., "1", "A1")
	ActionRef string

	// ActionURI is the full action URI (if resolved)
	ActionURI string

	// NewStatus is the detected new status
	NewStatus ActionItemStatus

	// MeetingURI is where this update was detected
	MeetingURI string

	// Note is the text that triggered the update
	Note string
}

// ProcessMeeting extracts actions from a meeting and updates existing action statuses.
func (t *ActionTracker) ProcessMeeting(meeting *Meeting, minutesText string) (*MeetingActionReport, error) {
	report := &MeetingActionReport{
		MeetingURI:     meeting.URI,
		MeetingDate:    meeting.Date,
		NewActions:     []*ActionItem{},
		StatusUpdates:  []ActionStatusUpdate{},
		PendingActions: []*ActionItem{},
	}

	// Extract new actions
	extracted, err := t.extractor.ExtractActions(minutesText, meeting.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to extract actions: %w", err)
	}

	for _, ext := range extracted {
		action := t.extractor.CreateActionItem(ext, meeting.URI)
		t.actionIndex[action.URI] = action
		t.meetingActions[meeting.URI] = append(t.meetingActions[meeting.URI], action.URI)
		report.NewActions = append(report.NewActions, action)

		// Add to store
		t.addActionToStore(action)
	}

	// Detect status updates
	updates := t.DetectStatusUpdate(minutesText, meeting.URI)
	for _, update := range updates {
		// Try to resolve action reference to URI
		update.ActionURI = t.resolveActionReference(update.ActionRef, meeting.URI)
		if update.ActionURI != "" {
			t.applyStatusUpdate(update)
		}
		report.StatusUpdates = append(report.StatusUpdates, update)
	}

	// Collect still-pending actions
	for _, action := range t.actionIndex {
		if action.Status == ActionPending || action.Status == ActionInProgress {
			report.PendingActions = append(report.PendingActions, action)
		}
	}

	// Sort pending by due date
	sort.Slice(report.PendingActions, func(i, j int) bool {
		a, b := report.PendingActions[i], report.PendingActions[j]
		if a.DueDate == nil && b.DueDate == nil {
			return false
		}
		if a.DueDate == nil {
			return false
		}
		if b.DueDate == nil {
			return true
		}
		return a.DueDate.Before(*b.DueDate)
	})

	return report, nil
}

// MeetingActionReport summarizes action items for a meeting.
type MeetingActionReport struct {
	// MeetingURI is the meeting this report covers
	MeetingURI string

	// MeetingDate is when the meeting occurred
	MeetingDate time.Time

	// NewActions are actions created at this meeting
	NewActions []*ActionItem

	// StatusUpdates are changes to existing actions
	StatusUpdates []ActionStatusUpdate

	// PendingActions are all actions still open
	PendingActions []*ActionItem
}

// addActionToStore persists an action item to the triple store.
func (t *ActionTracker) addActionToStore(action *ActionItem) {
	t.store.Add(action.URI, store.RDFType, store.ClassActionItem)
	t.store.Add(action.URI, store.PropLabel, action.Identifier)
	t.store.Add(action.URI, store.PropText, action.Description)
	t.store.Add(action.URI, store.PropActionAssignedAt, action.AssignedAtMeetingURI)
	t.store.Add(action.URI, store.PropActionStatus, action.Status.String())

	if action.DueDate != nil {
		t.store.Add(action.URI, store.PropActionDueDate, action.DueDate.Format(time.RFC3339))
	}

	for _, assignee := range action.AssignedToURIs {
		t.store.Add(action.URI, store.PropActionAssignedTo, assignee)
	}

	for _, prov := range action.RelatedProvisionURIs {
		t.store.Add(action.URI, store.PropActionRelatesTo, prov)
	}

	if action.Priority != "" {
		t.store.Add(action.URI, store.PropActionPriority, action.Priority)
	}
}

// resolveActionReference tries to find an action URI from a reference.
func (t *ActionTracker) resolveActionReference(ref string, currentMeetingURI string) string {
	if ref == "" {
		return ""
	}

	// Try direct match on identifier
	for uri, action := range t.actionIndex {
		if strings.EqualFold(action.Identifier, ref) ||
			strings.EqualFold(action.Identifier, "Action-"+ref) ||
			strings.EqualFold(action.Identifier, "Action "+ref) {
			return uri
		}
	}

	// Try matching by number suffix
	refNum := extractNumber(ref)
	if refNum > 0 {
		for uri, action := range t.actionIndex {
			actionNum := extractNumber(action.Identifier)
			if actionNum == refNum {
				return uri
			}
		}
	}

	return ""
}

// extractNumber extracts the first number from a string.
func extractNumber(s string) int {
	numPattern := regexp.MustCompile(`\d+`)
	if match := numPattern.FindString(s); match != "" {
		return parseIntFromString(match)
	}
	return 0
}

// applyStatusUpdate updates an action's status in the index and store.
func (t *ActionTracker) applyStatusUpdate(update ActionStatusUpdate) {
	action, ok := t.actionIndex[update.ActionURI]
	if !ok {
		return
	}

	action.Status = update.NewStatus

	// Add note
	note := ActionNote{
		MeetingURI:    update.MeetingURI,
		Date:          time.Now(),
		Note:          update.Note,
		UpdatedStatus: &update.NewStatus,
	}
	action.Notes = append(action.Notes, note)

	if update.NewStatus == ActionCompleted {
		action.CompletedAtMeetingURI = update.MeetingURI
		now := time.Now()
		action.CompletedAt = &now
	}

	// Update store
	t.store.Delete(update.ActionURI, store.PropActionStatus, "")
	t.store.Add(update.ActionURI, store.PropActionStatus, update.NewStatus.String())

	if update.NewStatus == ActionCompleted {
		t.store.Add(update.ActionURI, store.PropActionCompletedAt, update.MeetingURI)
	}
}

// GetPendingActions returns all pending action items.
func (t *ActionTracker) GetPendingActions() []*ActionItem {
	var pending []*ActionItem
	for _, action := range t.actionIndex {
		if action.Status == ActionPending || action.Status == ActionInProgress {
			pending = append(pending, action)
		}
	}
	return pending
}

// GetActionsByAssignee returns actions assigned to a specific entity.
func (t *ActionTracker) GetActionsByAssignee(assigneeURI string) []*ActionItem {
	var actions []*ActionItem
	for _, action := range t.actionIndex {
		for _, uri := range action.AssignedToURIs {
			if uri == assigneeURI {
				actions = append(actions, action)
				break
			}
		}
	}
	return actions
}

// GetActionsByProvision returns actions related to a specific provision.
func (t *ActionTracker) GetActionsByProvision(provisionURI string) []*ActionItem {
	var actions []*ActionItem
	for _, action := range t.actionIndex {
		for _, uri := range action.RelatedProvisionURIs {
			if uri == provisionURI {
				actions = append(actions, action)
				break
			}
		}
	}
	return actions
}

// GetActionsForMeeting returns all actions from a specific meeting.
func (t *ActionTracker) GetActionsForMeeting(meetingURI string) []*ActionItem {
	actionURIs := t.meetingActions[meetingURI]
	var actions []*ActionItem
	for _, uri := range actionURIs {
		if action, ok := t.actionIndex[uri]; ok {
			actions = append(actions, action)
		}
	}
	return actions
}

// GetOverdueActions returns all overdue pending actions.
func (t *ActionTracker) GetOverdueActions() []*ActionItem {
	var overdue []*ActionItem
	for _, action := range t.actionIndex {
		if action.IsOverdue() {
			overdue = append(overdue, action)
		}
	}
	return overdue
}

// ActionSummary provides aggregate statistics on action items.
type ActionSummary struct {
	// Total is the count of all action items
	Total int

	// Pending is the count of pending actions
	Pending int

	// InProgress is the count of in-progress actions
	InProgress int

	// Completed is the count of completed actions
	Completed int

	// Deferred is the count of deferred actions
	Deferred int

	// Cancelled is the count of cancelled actions
	Cancelled int

	// Overdue is the count of overdue actions
	Overdue int

	// AverageCompletionDays is the average days to complete
	AverageCompletionDays float64

	// ByAssignee maps assignee to their action count
	ByAssignee map[string]int

	// ByMeeting maps meeting to action count
	ByMeeting map[string]int
}

// GetSummary returns aggregate statistics on all tracked actions.
func (t *ActionTracker) GetSummary() ActionSummary {
	summary := ActionSummary{
		ByAssignee: make(map[string]int),
		ByMeeting:  make(map[string]int),
	}

	var totalCompletionDays int
	var completedWithDueDate int

	for _, action := range t.actionIndex {
		summary.Total++

		switch action.Status {
		case ActionPending:
			summary.Pending++
		case ActionInProgress:
			summary.InProgress++
		case ActionCompleted:
			summary.Completed++
			// Calculate completion time
			if action.CompletedAt != nil && action.DueDate != nil {
				days := int(action.CompletedAt.Sub(*action.DueDate).Hours() / 24)
				totalCompletionDays += days
				completedWithDueDate++
			}
		case ActionDeferred:
			summary.Deferred++
		case ActionCancelled:
			summary.Cancelled++
		}

		if action.IsOverdue() {
			summary.Overdue++
		}

		for _, assignee := range action.AssignedToURIs {
			summary.ByAssignee[assignee]++
		}

		summary.ByMeeting[action.AssignedAtMeetingURI]++
	}

	if completedWithDueDate > 0 {
		summary.AverageCompletionDays = float64(totalCompletionDays) / float64(completedWithDueDate)
	}

	return summary
}

// LoadActionsFromStore loads action items from the triple store into the tracker.
func (t *ActionTracker) LoadActionsFromStore() error {
	// Find all action items
	triples := t.store.Find("", store.RDFType, store.ClassActionItem)

	for _, triple := range triples {
		actionURI := triple.Subject

		// Build action item from triples
		action := &ActionItem{
			URI: actionURI,
		}

		// Get basic properties
		if labels := t.store.Find(actionURI, store.PropLabel, ""); len(labels) > 0 {
			action.Identifier = labels[0].Object
		}
		if texts := t.store.Find(actionURI, store.PropText, ""); len(texts) > 0 {
			action.Description = texts[0].Object
		}
		if statuses := t.store.Find(actionURI, store.PropActionStatus, ""); len(statuses) > 0 {
			action.Status = parseActionStatus(statuses[0].Object)
		}
		if meetings := t.store.Find(actionURI, store.PropActionAssignedAt, ""); len(meetings) > 0 {
			action.AssignedAtMeetingURI = meetings[0].Object
		}
		if dates := t.store.Find(actionURI, store.PropActionDueDate, ""); len(dates) > 0 {
			if t, err := time.Parse(time.RFC3339, dates[0].Object); err == nil {
				action.DueDate = &t
			}
		}
		if priorities := t.store.Find(actionURI, store.PropActionPriority, ""); len(priorities) > 0 {
			action.Priority = priorities[0].Object
		}

		// Get assignees
		assignees := t.store.Find(actionURI, store.PropActionAssignedTo, "")
		for _, a := range assignees {
			action.AssignedToURIs = append(action.AssignedToURIs, a.Object)
		}

		// Get related provisions
		provisions := t.store.Find(actionURI, store.PropActionRelatesTo, "")
		for _, p := range provisions {
			action.RelatedProvisionURIs = append(action.RelatedProvisionURIs, p.Object)
		}

		t.actionIndex[actionURI] = action
		t.meetingActions[action.AssignedAtMeetingURI] = append(
			t.meetingActions[action.AssignedAtMeetingURI],
			actionURI,
		)
	}

	return nil
}

// parseActionStatus converts a status string to ActionItemStatus.
func parseActionStatus(s string) ActionItemStatus {
	switch strings.ToLower(s) {
	case "pending":
		return ActionPending
	case "in_progress":
		return ActionInProgress
	case "completed":
		return ActionCompleted
	case "deferred":
		return ActionDeferred
	case "cancelled":
		return ActionCancelled
	default:
		return ActionPending
	}
}
