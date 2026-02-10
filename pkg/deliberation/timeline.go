package deliberation

import (
	"encoding/json"
	"fmt"
	"html"
	"sort"
	"strings"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// TimelineEventType indicates the type of event on a timeline.
type TimelineEventType int

const (
	// EventMeeting represents a meeting event.
	EventMeeting TimelineEventType = iota
	// EventProposal represents a new proposal.
	EventProposal
	// EventAmendment represents an amendment to a provision.
	EventAmendment
	// EventVote represents a vote on a motion.
	EventVote
	// EventDecision represents a formal decision.
	EventDecision
	// EventDeferral represents deferral to a future meeting.
	EventDeferral
	// EventAction represents an action item.
	EventAction
	// EventComment represents a comment or intervention.
	EventComment
)

// String returns a human-readable label for the event type.
func (t TimelineEventType) String() string {
	switch t {
	case EventMeeting:
		return "Meeting"
	case EventProposal:
		return "Proposal"
	case EventAmendment:
		return "Amendment"
	case EventVote:
		return "Vote"
	case EventDecision:
		return "Decision"
	case EventDeferral:
		return "Deferral"
	case EventAction:
		return "Action"
	case EventComment:
		return "Comment"
	default:
		return "Unknown"
	}
}

// Symbol returns a symbol for ASCII rendering.
func (t TimelineEventType) Symbol() string {
	switch t {
	case EventMeeting:
		return "■"
	case EventProposal:
		return "●"
	case EventAmendment:
		return "◆"
	case EventVote:
		return "◐"
	case EventDecision:
		return "★"
	case EventDeferral:
		return "→"
	case EventAction:
		return "▶"
	case EventComment:
		return "○"
	default:
		return "·"
	}
}

// TimelineEvent represents a single event on a timeline.
type TimelineEvent struct {
	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`

	// EventType classifies the event.
	EventType TimelineEventType `json:"event_type"`

	// Label is a short description of the event.
	Label string `json:"label"`

	// Description provides more detail about the event.
	Description string `json:"description,omitempty"`

	// URI is the identifier of the event in the knowledge graph.
	URI string `json:"uri"`

	// Actors lists stakeholders involved in the event.
	Actors []string `json:"actors,omitempty"`

	// Links lists related provision or document URIs.
	Links []string `json:"links,omitempty"`

	// Metadata holds additional key-value pairs.
	Metadata map[string]string `json:"metadata,omitempty"`

	// IsMilestone indicates if this is a significant event.
	IsMilestone bool `json:"is_milestone,omitempty"`
}

// Milestone represents a significant point in the timeline.
type Milestone struct {
	// Event is the milestone event.
	Event TimelineEvent `json:"event"`

	// Name is the milestone label.
	Name string `json:"name"`
}

// Swimlane groups events in a multi-track timeline.
type Swimlane struct {
	// Name identifies the swimlane (e.g., stakeholder name, provision).
	Name string `json:"name"`

	// URI is the identifier for this swimlane.
	URI string `json:"uri,omitempty"`

	// Events contains events in this swimlane.
	Events []TimelineEvent `json:"events"`
}

// Timeline represents a chronological view of deliberation events.
type Timeline struct {
	// Title is the timeline heading.
	Title string `json:"title"`

	// Scope identifies what this timeline covers (process URI, provision URI).
	Scope string `json:"scope"`

	// ScopeType indicates what kind of scope (process, provision, stakeholder).
	ScopeType string `json:"scope_type"`

	// StartDate is the beginning of the timeline.
	StartDate time.Time `json:"start_date"`

	// EndDate is the end of the timeline.
	EndDate time.Time `json:"end_date"`

	// Events contains all events in chronological order.
	Events []TimelineEvent `json:"events"`

	// Milestones lists significant events.
	Milestones []Milestone `json:"milestones"`

	// Swimlanes groups events by category (for multi-track timelines).
	Swimlanes []Swimlane `json:"swimlanes,omitempty"`
}

// TimelineConfig configures timeline generation.
type TimelineConfig struct {
	// StartDate filters events after this date.
	StartDate time.Time

	// EndDate filters events before this date.
	EndDate time.Time

	// EventTypes filters to specific event types (empty = all).
	EventTypes []TimelineEventType

	// GroupBy creates swimlanes grouped by this field (stakeholder, provision).
	GroupBy string

	// IncludeMilestones highlights significant events.
	IncludeMilestones bool

	// Provision filters to events affecting this provision.
	Provision string

	// Stakeholder filters to events involving this stakeholder.
	Stakeholder string

	// CompareProvisions lists provisions for comparative timeline.
	CompareProvisions []string
}

// TimelineBuilder constructs timelines from a triple store.
type TimelineBuilder struct {
	store   *store.TripleStore
	baseURI string
}

// NewTimelineBuilder creates a new timeline builder.
func NewTimelineBuilder(tripleStore *store.TripleStore, baseURI string) *TimelineBuilder {
	return &TimelineBuilder{
		store:   tripleStore,
		baseURI: baseURI,
	}
}

// BuildProcessTimeline creates a timeline for an entire deliberation process.
func (b *TimelineBuilder) BuildProcessTimeline(processID string, config TimelineConfig) (*Timeline, error) {
	processURI := b.baseURI + "process/" + processID

	// Query all meetings in the process
	meetings := b.queryMeetings(processURI)

	// Build events from meetings
	events := make([]TimelineEvent, 0)
	for _, meeting := range meetings {
		events = append(events, b.meetingToEvents(meeting)...)
	}

	// Sort events by timestamp
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	// Apply filters
	events = b.filterEvents(events, config)

	// Identify milestones
	milestones := b.identifyMilestones(events)

	// Calculate date range
	startDate, endDate := b.calculateDateRange(events, config)

	timeline := &Timeline{
		Title:      "Process Timeline: " + processID,
		Scope:      processURI,
		ScopeType:  "process",
		StartDate:  startDate,
		EndDate:    endDate,
		Events:     events,
		Milestones: milestones,
	}

	// Create swimlanes if grouping requested
	if config.GroupBy != "" {
		timeline.Swimlanes = b.createSwimlanes(events, config.GroupBy)
	}

	return timeline, nil
}

// BuildProvisionTimeline creates a timeline for a specific provision's evolution.
func (b *TimelineBuilder) BuildProvisionTimeline(provisionURI string, config TimelineConfig) (*Timeline, error) {
	events := make([]TimelineEvent, 0)

	// Query events affecting this provision
	// Find motions targeting this provision
	motionTriples := b.store.Find("", "reg:targetProvision", provisionURI)
	for _, triple := range motionTriples {
		motionURI := triple.Subject
		motionEvents := b.queryMotionEvents(motionURI)
		events = append(events, motionEvents...)
	}

	// Find decisions affecting this provision
	decisionTriples := b.store.Find("", "reg:affectsProvision", provisionURI)
	for _, triple := range decisionTriples {
		decisionURI := triple.Subject
		decisionEvents := b.queryDecisionEvents(decisionURI)
		events = append(events, decisionEvents...)
	}

	// Find agenda items discussing this provision
	agendaTriples := b.store.Find("", "reg:discussesProvision", provisionURI)
	for _, triple := range agendaTriples {
		agendaURI := triple.Subject
		agendaEvents := b.queryAgendaEvents(agendaURI)
		events = append(events, agendaEvents...)
	}

	// Sort and filter
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})
	events = b.filterEvents(events, config)

	// Extract label from provision URI
	provisionLabel := extractURILabel(provisionURI)

	milestones := b.identifyMilestones(events)
	startDate, endDate := b.calculateDateRange(events, config)

	return &Timeline{
		Title:      "Provision Timeline: " + provisionLabel,
		Scope:      provisionURI,
		ScopeType:  "provision",
		StartDate:  startDate,
		EndDate:    endDate,
		Events:     events,
		Milestones: milestones,
	}, nil
}

// BuildStakeholderTimeline creates a timeline of a stakeholder's participation.
func (b *TimelineBuilder) BuildStakeholderTimeline(stakeholderURI string, config TimelineConfig) (*Timeline, error) {
	events := make([]TimelineEvent, 0)

	// Find interventions by this stakeholder
	interventionTriples := b.store.Find("", "reg:speaker", stakeholderURI)
	for _, triple := range interventionTriples {
		interventionURI := triple.Subject
		event := b.queryInterventionEvent(interventionURI)
		if event != nil {
			events = append(events, *event)
		}
	}

	// Find motions proposed by this stakeholder
	proposerTriples := b.store.Find("", "reg:proposedBy", stakeholderURI)
	for _, triple := range proposerTriples {
		motionURI := triple.Subject
		motionEvents := b.queryMotionEvents(motionURI)
		events = append(events, motionEvents...)
	}

	// Find votes cast by this stakeholder
	voteTriples := b.store.Find("", "reg:voter", stakeholderURI)
	for _, triple := range voteTriples {
		voteURI := triple.Subject
		event := b.queryVoteEvent(voteURI)
		if event != nil {
			events = append(events, *event)
		}
	}

	// Sort and filter
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})
	events = b.filterEvents(events, config)

	stakeholderLabel := extractURILabel(stakeholderURI)
	milestones := b.identifyMilestones(events)
	startDate, endDate := b.calculateDateRange(events, config)

	return &Timeline{
		Title:      "Stakeholder Timeline: " + stakeholderLabel,
		Scope:      stakeholderURI,
		ScopeType:  "stakeholder",
		StartDate:  startDate,
		EndDate:    endDate,
		Events:     events,
		Milestones: milestones,
	}, nil
}

// BuildComparativeTimeline creates a side-by-side timeline for multiple provisions.
func (b *TimelineBuilder) BuildComparativeTimeline(provisionURIs []string, config TimelineConfig) (*Timeline, error) {
	swimlanes := make([]Swimlane, 0, len(provisionURIs))

	var allEvents []TimelineEvent
	for _, uri := range provisionURIs {
		provisionTimeline, err := b.BuildProvisionTimeline(uri, config)
		if err != nil {
			continue
		}

		label := extractURILabel(uri)
		swimlanes = append(swimlanes, Swimlane{
			Name:   label,
			URI:    uri,
			Events: provisionTimeline.Events,
		})
		allEvents = append(allEvents, provisionTimeline.Events...)
	}

	// Sort all events for date range calculation
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].Timestamp.Before(allEvents[j].Timestamp)
	})

	startDate, endDate := b.calculateDateRange(allEvents, config)
	milestones := b.identifyMilestones(allEvents)

	return &Timeline{
		Title:      "Comparative Timeline",
		Scope:      strings.Join(provisionURIs, ","),
		ScopeType:  "comparative",
		StartDate:  startDate,
		EndDate:    endDate,
		Events:     allEvents,
		Milestones: milestones,
		Swimlanes:  swimlanes,
	}, nil
}

// queryMeetings returns meetings for a process.
func (b *TimelineBuilder) queryMeetings(processURI string) []Meeting {
	meetings := make([]Meeting, 0)

	// Find meetings in this process
	meetingTriples := b.store.Find("", "reg:partOfProcess", processURI)
	for _, triple := range meetingTriples {
		meetingURI := triple.Subject
		meeting := b.loadMeeting(meetingURI)
		if meeting != nil {
			meetings = append(meetings, *meeting)
		}
	}

	// Also find meetings via type
	typeTriples := b.store.Find("", "rdf:type", "reg:Meeting")
	for _, triple := range typeTriples {
		meetingURI := triple.Subject
		meeting := b.loadMeeting(meetingURI)
		if meeting != nil {
			meetings = append(meetings, *meeting)
		}
	}

	return meetings
}

// loadMeeting loads a meeting from the triple store.
func (b *TimelineBuilder) loadMeeting(meetingURI string) *Meeting {
	// Check if it's a meeting
	typeTriples := b.store.Find(meetingURI, "rdf:type", "")
	isMeeting := false
	for _, t := range typeTriples {
		if strings.Contains(t.Object, "Meeting") {
			isMeeting = true
			break
		}
	}
	if !isMeeting && len(typeTriples) > 0 {
		return nil
	}

	meeting := &Meeting{URI: meetingURI}

	// Load meeting properties
	if triples := b.store.Find(meetingURI, store.PropTitle, ""); len(triples) > 0 {
		meeting.Title = triples[0].Object
	}
	if triples := b.store.Find(meetingURI, "reg:identifier", ""); len(triples) > 0 {
		meeting.Identifier = triples[0].Object
	}
	if triples := b.store.Find(meetingURI, "reg:date", ""); len(triples) > 0 {
		if t, err := time.Parse(time.RFC3339, triples[0].Object); err == nil {
			meeting.Date = t
		} else if t, err := time.Parse("2006-01-02", triples[0].Object); err == nil {
			meeting.Date = t
		}
	}

	return meeting
}

// meetingToEvents converts a meeting to timeline events.
func (b *TimelineBuilder) meetingToEvents(meeting Meeting) []TimelineEvent {
	events := make([]TimelineEvent, 0)

	// Add meeting event
	events = append(events, TimelineEvent{
		Timestamp:   meeting.Date,
		EventType:   EventMeeting,
		Label:       meeting.Title,
		Description: fmt.Sprintf("Meeting %s", meeting.Identifier),
		URI:         meeting.URI,
		IsMilestone: false,
	})

	// Add events from agenda items
	for _, item := range meeting.AgendaItems {
		// Add motions
		for _, motion := range item.Motions {
			proposedAt := meeting.Date
			if motion.ProposedAt != nil {
				proposedAt = *motion.ProposedAt
			}

			eventType := EventProposal
			if motion.Type == "amendment" {
				eventType = EventAmendment
			}

			events = append(events, TimelineEvent{
				Timestamp:   proposedAt,
				EventType:   eventType,
				Label:       motion.Title,
				Description: motion.Text,
				URI:         motion.URI,
				Actors:      []string{motion.ProposerName},
			})

			// Add vote event if voted
			if motion.Vote != nil {
				events = append(events, TimelineEvent{
					Timestamp:   motion.Vote.VoteDate,
					EventType:   EventVote,
					Label:       fmt.Sprintf("Vote: %s", motion.Title),
					Description: fmt.Sprintf("%s (%d-%d-%d)", motion.Vote.Result, motion.Vote.ForCount, motion.Vote.AgainstCount, motion.Vote.AbstainCount),
					URI:         motion.Vote.URI,
					Metadata: map[string]string{
						"result":   motion.Vote.Result,
						"for":      fmt.Sprintf("%d", motion.Vote.ForCount),
						"against":  fmt.Sprintf("%d", motion.Vote.AgainstCount),
						"abstain":  fmt.Sprintf("%d", motion.Vote.AbstainCount),
					},
				})
			}
		}

		// Add decisions
		for _, decision := range item.Decisions {
			events = append(events, TimelineEvent{
				Timestamp:   decision.DecidedAt,
				EventType:   EventDecision,
				Label:       decision.Title,
				Description: decision.Description,
				URI:         decision.URI,
				IsMilestone: true,
				Links:       decision.AffectedProvisionURIs,
			})
		}

		// Add deferrals
		if item.Outcome == OutcomeDeferred {
			events = append(events, TimelineEvent{
				Timestamp:   meeting.Date,
				EventType:   EventDeferral,
				Label:       fmt.Sprintf("Deferred: %s", item.Title),
				Description: item.Notes,
				URI:         item.URI,
				Metadata: map[string]string{
					"deferred_to": item.DeferredTo,
				},
			})
		}

		// Add action items
		for _, action := range item.ActionItems {
			events = append(events, TimelineEvent{
				Timestamp:   meeting.Date,
				EventType:   EventAction,
				Label:       action.Description,
				URI:         action.URI,
				Actors:      action.AssignedToNames,
			})
		}
	}

	return events
}

// queryMotionEvents returns events for a motion.
func (b *TimelineBuilder) queryMotionEvents(motionURI string) []TimelineEvent {
	events := make([]TimelineEvent, 0)

	// Get motion details
	var title, text, proposer string
	var proposedAt time.Time

	if triples := b.store.Find(motionURI, store.PropTitle, ""); len(triples) > 0 {
		title = triples[0].Object
	}
	if triples := b.store.Find(motionURI, store.PropText, ""); len(triples) > 0 {
		text = triples[0].Object
	}
	if triples := b.store.Find(motionURI, "reg:proposedBy", ""); len(triples) > 0 {
		proposer = extractURILabel(triples[0].Object)
	}
	if triples := b.store.Find(motionURI, "reg:proposedAt", ""); len(triples) > 0 {
		proposedAt, _ = time.Parse(time.RFC3339, triples[0].Object)
	}

	if title != "" {
		events = append(events, TimelineEvent{
			Timestamp:   proposedAt,
			EventType:   EventAmendment,
			Label:       title,
			Description: text,
			URI:         motionURI,
			Actors:      []string{proposer},
		})
	}

	return events
}

// queryDecisionEvents returns events for a decision.
func (b *TimelineBuilder) queryDecisionEvents(decisionURI string) []TimelineEvent {
	events := make([]TimelineEvent, 0)

	var title, description string
	var decidedAt time.Time

	if triples := b.store.Find(decisionURI, store.PropTitle, ""); len(triples) > 0 {
		title = triples[0].Object
	}
	if triples := b.store.Find(decisionURI, "reg:description", ""); len(triples) > 0 {
		description = triples[0].Object
	}
	if triples := b.store.Find(decisionURI, "reg:decidedAt", ""); len(triples) > 0 {
		decidedAt, _ = time.Parse(time.RFC3339, triples[0].Object)
	}

	if title != "" {
		events = append(events, TimelineEvent{
			Timestamp:   decidedAt,
			EventType:   EventDecision,
			Label:       title,
			Description: description,
			URI:         decisionURI,
			IsMilestone: true,
		})
	}

	return events
}

// queryAgendaEvents returns events for an agenda item.
func (b *TimelineBuilder) queryAgendaEvents(agendaURI string) []TimelineEvent {
	events := make([]TimelineEvent, 0)

	var title string
	var meetingDate time.Time

	if triples := b.store.Find(agendaURI, store.PropTitle, ""); len(triples) > 0 {
		title = triples[0].Object
	}

	// Get meeting date
	if triples := b.store.Find(agendaURI, "reg:meeting", ""); len(triples) > 0 {
		meetingURI := triples[0].Object
		if dateTriples := b.store.Find(meetingURI, "reg:date", ""); len(dateTriples) > 0 {
			meetingDate, _ = time.Parse(time.RFC3339, dateTriples[0].Object)
		}
	}

	if title != "" {
		events = append(events, TimelineEvent{
			Timestamp:   meetingDate,
			EventType:   EventComment,
			Label:       fmt.Sprintf("Discussed: %s", title),
			URI:         agendaURI,
		})
	}

	return events
}

// queryInterventionEvent returns an event for an intervention.
func (b *TimelineBuilder) queryInterventionEvent(interventionURI string) *TimelineEvent {
	var summary, speaker string
	var timestamp time.Time

	if triples := b.store.Find(interventionURI, "reg:summary", ""); len(triples) > 0 {
		summary = triples[0].Object
	}
	if triples := b.store.Find(interventionURI, "reg:speakerName", ""); len(triples) > 0 {
		speaker = triples[0].Object
	}
	if triples := b.store.Find(interventionURI, "reg:timestamp", ""); len(triples) > 0 {
		timestamp, _ = time.Parse(time.RFC3339, triples[0].Object)
	}

	if summary != "" || speaker != "" {
		return &TimelineEvent{
			Timestamp:   timestamp,
			EventType:   EventComment,
			Label:       fmt.Sprintf("%s spoke", speaker),
			Description: summary,
			URI:         interventionURI,
			Actors:      []string{speaker},
		}
	}

	return nil
}

// queryVoteEvent returns an event for a vote.
func (b *TimelineBuilder) queryVoteEvent(voteURI string) *TimelineEvent {
	var position, voter string
	var voteDate time.Time

	if triples := b.store.Find(voteURI, "reg:position", ""); len(triples) > 0 {
		position = triples[0].Object
	}
	if triples := b.store.Find(voteURI, "reg:voterName", ""); len(triples) > 0 {
		voter = triples[0].Object
	}
	if triples := b.store.Find(voteURI, "reg:voteDate", ""); len(triples) > 0 {
		voteDate, _ = time.Parse(time.RFC3339, triples[0].Object)
	}

	if voter != "" {
		return &TimelineEvent{
			Timestamp:   voteDate,
			EventType:   EventVote,
			Label:       fmt.Sprintf("%s voted %s", voter, position),
			URI:         voteURI,
			Actors:      []string{voter},
			Metadata: map[string]string{
				"position": position,
			},
		}
	}

	return nil
}

// filterEvents applies config filters to events.
func (b *TimelineBuilder) filterEvents(events []TimelineEvent, config TimelineConfig) []TimelineEvent {
	result := make([]TimelineEvent, 0)

	for _, event := range events {
		// Date range filter
		if !config.StartDate.IsZero() && event.Timestamp.Before(config.StartDate) {
			continue
		}
		if !config.EndDate.IsZero() && event.Timestamp.After(config.EndDate) {
			continue
		}

		// Event type filter
		if len(config.EventTypes) > 0 {
			found := false
			for _, t := range config.EventTypes {
				if event.EventType == t {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Stakeholder filter
		if config.Stakeholder != "" {
			found := false
			for _, actor := range event.Actors {
				if strings.Contains(strings.ToLower(actor), strings.ToLower(config.Stakeholder)) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Provision filter
		if config.Provision != "" {
			found := false
			for _, link := range event.Links {
				if strings.Contains(link, config.Provision) {
					found = true
					break
				}
			}
			if !found && !strings.Contains(event.URI, config.Provision) {
				continue
			}
		}

		result = append(result, event)
	}

	return result
}

// identifyMilestones finds significant events.
func (b *TimelineBuilder) identifyMilestones(events []TimelineEvent) []Milestone {
	milestones := make([]Milestone, 0)

	for _, event := range events {
		if event.IsMilestone || event.EventType == EventDecision {
			milestones = append(milestones, Milestone{
				Event: event,
				Name:  event.Label,
			})
		}
	}

	return milestones
}

// calculateDateRange determines the timeline's date range.
func (b *TimelineBuilder) calculateDateRange(events []TimelineEvent, config TimelineConfig) (time.Time, time.Time) {
	if len(events) == 0 {
		now := time.Now()
		return now.AddDate(0, -1, 0), now
	}

	startDate := events[0].Timestamp
	endDate := events[len(events)-1].Timestamp

	// Apply config overrides
	if !config.StartDate.IsZero() {
		startDate = config.StartDate
	}
	if !config.EndDate.IsZero() {
		endDate = config.EndDate
	}

	// Ensure end is after start
	if endDate.Before(startDate) {
		endDate = startDate.AddDate(0, 1, 0)
	}

	return startDate, endDate
}

// createSwimlanes groups events into swimlanes.
func (b *TimelineBuilder) createSwimlanes(events []TimelineEvent, groupBy string) []Swimlane {
	lanes := make(map[string]*Swimlane)

	for _, event := range events {
		var key string
		switch groupBy {
		case "stakeholder":
			if len(event.Actors) > 0 {
				key = event.Actors[0]
			} else {
				key = "Unknown"
			}
		case "type":
			key = event.EventType.String()
		case "provision":
			if len(event.Links) > 0 {
				key = extractURILabel(event.Links[0])
			} else {
				key = "General"
			}
		default:
			key = "Default"
		}

		if _, ok := lanes[key]; !ok {
			lanes[key] = &Swimlane{Name: key, Events: make([]TimelineEvent, 0)}
		}
		lanes[key].Events = append(lanes[key].Events, event)
	}

	// Convert map to slice
	result := make([]Swimlane, 0, len(lanes))
	for _, lane := range lanes {
		result = append(result, *lane)
	}

	// Sort by name
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// RenderASCII renders the timeline as ASCII art for terminal display.
func (t *Timeline) RenderASCII() string {
	var sb strings.Builder

	// Title
	sb.WriteString(t.Title + "\n")
	sb.WriteString(strings.Repeat("=", len(t.Title)) + "\n\n")

	if len(t.Events) == 0 {
		sb.WriteString("No events found.\n")
		return sb.String()
	}

	// Date range header
	sb.WriteString(fmt.Sprintf("%s  ○─────────────────────────────────────────○  %s\n",
		t.StartDate.Format("2006-01-02"),
		t.EndDate.Format("2006-01-02")))
	sb.WriteString("            │                                         │\n")

	// Events
	for _, event := range t.Events {
		symbol := event.EventType.Symbol()
		dateStr := event.Timestamp.Format("Jan 02")

		label := event.Label
		if len(label) > 40 {
			label = label[:37] + "..."
		}

		sb.WriteString(fmt.Sprintf("%-11s %s  [%s] %s\n",
			dateStr, symbol, event.EventType.String(), label))

		if event.Description != "" {
			desc := event.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			sb.WriteString(fmt.Sprintf("            │   %s\n", desc))
		}
		sb.WriteString("            │\n")
	}

	// Legend
	sb.WriteString("\nLegend:\n")
	sb.WriteString("  ■ Meeting  ● Proposal  ◆ Amendment  ◐ Vote  ★ Decision  → Deferral  ▶ Action  ○ Comment\n")

	return sb.String()
}

// RenderMermaid renders the timeline as a Mermaid diagram.
func (t *Timeline) RenderMermaid() string {
	var sb strings.Builder

	sb.WriteString("```mermaid\n")
	sb.WriteString("timeline\n")
	sb.WriteString(fmt.Sprintf("    title %s\n", t.Title))

	// Group events by date
	eventsByDate := make(map[string][]TimelineEvent)
	for _, event := range t.Events {
		dateKey := event.Timestamp.Format("2006-01-02")
		eventsByDate[dateKey] = append(eventsByDate[dateKey], event)
	}

	// Sort dates
	dates := make([]string, 0, len(eventsByDate))
	for date := range eventsByDate {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	// Render each date section
	for _, date := range dates {
		events := eventsByDate[date]
		sb.WriteString(fmt.Sprintf("    section %s\n", date))
		for _, event := range events {
			label := strings.ReplaceAll(event.Label, ":", "-")
			sb.WriteString(fmt.Sprintf("        %s : %s\n", event.EventType.String(), label))
		}
	}

	sb.WriteString("```\n")
	return sb.String()
}

// RenderSVG renders the timeline as an SVG image.
func (t *Timeline) RenderSVG() string {
	var sb strings.Builder

	if len(t.Events) == 0 {
		return "<svg></svg>"
	}

	// Calculate dimensions
	eventHeight := 60
	marginTop := 80
	marginLeft := 150
	width := 800
	height := marginTop + len(t.Events)*eventHeight + 50

	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d">`, width, height))
	sb.WriteString("\n<style>")
	sb.WriteString(`
    .title { font: bold 18px sans-serif; }
    .date-label { font: 12px sans-serif; fill: #666; }
    .event-label { font: 14px sans-serif; }
    .event-desc { font: 11px sans-serif; fill: #666; }
    .timeline-line { stroke: #ccc; stroke-width: 2; }
    .event-dot { stroke: #333; stroke-width: 2; }
    .meeting { fill: #2196F3; }
    .proposal { fill: #4CAF50; }
    .amendment { fill: #FF9800; }
    .vote { fill: #9C27B0; }
    .decision { fill: #F44336; }
    .deferral { fill: #607D8B; }
    .action { fill: #00BCD4; }
    .comment { fill: #E0E0E0; }
  `)
	sb.WriteString("</style>\n")

	// Title
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="30" class="title">%s</text>`, width/2, html.EscapeString(t.Title)))
	sb.WriteString("\n")

	// Date range
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="55" class="date-label">%s to %s</text>`,
		width/2, t.StartDate.Format("Jan 2, 2006"), t.EndDate.Format("Jan 2, 2006")))
	sb.WriteString("\n")

	// Timeline line
	lineX := marginLeft - 30
	sb.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" class="timeline-line"/>`,
		lineX, marginTop, lineX, marginTop+len(t.Events)*eventHeight))
	sb.WriteString("\n")

	// Events
	for i, event := range t.Events {
		y := marginTop + i*eventHeight + 20

		// Event dot
		eventClass := strings.ToLower(event.EventType.String())
		sb.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="8" class="event-dot %s"/>`,
			lineX, y, eventClass))
		sb.WriteString("\n")

		// Date label
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="date-label" text-anchor="end">%s</text>`,
			lineX-20, y+5, event.Timestamp.Format("Jan 02")))
		sb.WriteString("\n")

		// Event label
		label := event.Label
		if len(label) > 50 {
			label = label[:47] + "..."
		}
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="event-label">[%s] %s</text>`,
			lineX+20, y+5, event.EventType.String(), html.EscapeString(label)))
		sb.WriteString("\n")

		// Description
		if event.Description != "" {
			desc := event.Description
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="event-desc">%s</text>`,
				lineX+20, y+22, html.EscapeString(desc)))
			sb.WriteString("\n")
		}
	}

	sb.WriteString("</svg>")
	return sb.String()
}

// RenderHTML renders the timeline as an interactive HTML page.
func (t *Timeline) RenderHTML() string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>`)
	sb.WriteString(html.EscapeString(t.Title))
	sb.WriteString(`</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 20px; }
    h1 { color: #333; }
    .timeline { position: relative; padding-left: 40px; }
    .timeline::before { content: ''; position: absolute; left: 15px; top: 0; bottom: 0; width: 2px; background: #ddd; }
    .event { position: relative; margin-bottom: 30px; padding-left: 30px; }
    .event::before { content: ''; position: absolute; left: -25px; top: 5px; width: 12px; height: 12px; border-radius: 50%; border: 2px solid #333; }
    .event.meeting::before { background: #2196F3; }
    .event.proposal::before { background: #4CAF50; }
    .event.amendment::before { background: #FF9800; }
    .event.vote::before { background: #9C27B0; }
    .event.decision::before { background: #F44336; }
    .event.deferral::before { background: #607D8B; }
    .event.action::before { background: #00BCD4; }
    .event.comment::before { background: #E0E0E0; }
    .event-date { font-size: 12px; color: #666; }
    .event-type { font-size: 11px; color: #fff; padding: 2px 6px; border-radius: 3px; margin-left: 10px; }
    .event-type.meeting { background: #2196F3; }
    .event-type.proposal { background: #4CAF50; }
    .event-type.amendment { background: #FF9800; }
    .event-type.vote { background: #9C27B0; }
    .event-type.decision { background: #F44336; }
    .event-type.deferral { background: #607D8B; }
    .event-type.action { background: #00BCD4; }
    .event-type.comment { background: #999; }
    .event-label { font-weight: bold; margin-top: 5px; }
    .event-desc { color: #666; font-size: 14px; margin-top: 5px; }
    .event-actors { font-size: 12px; color: #888; margin-top: 5px; }
    .filters { margin-bottom: 20px; }
    .filter-btn { padding: 5px 10px; margin-right: 5px; border: 1px solid #ddd; border-radius: 3px; cursor: pointer; }
    .filter-btn.active { background: #2196F3; color: white; border-color: #2196F3; }
    .milestone { border-left: 3px solid #F44336; }
  </style>
</head>
<body>
  <h1>`)
	sb.WriteString(html.EscapeString(t.Title))
	sb.WriteString(`</h1>
  <p>`)
	sb.WriteString(fmt.Sprintf("%s to %s", t.StartDate.Format("January 2, 2006"), t.EndDate.Format("January 2, 2006")))
	sb.WriteString(`</p>

  <div class="filters">
    <button class="filter-btn active" data-filter="all">All</button>
    <button class="filter-btn" data-filter="meeting">Meetings</button>
    <button class="filter-btn" data-filter="proposal">Proposals</button>
    <button class="filter-btn" data-filter="amendment">Amendments</button>
    <button class="filter-btn" data-filter="vote">Votes</button>
    <button class="filter-btn" data-filter="decision">Decisions</button>
  </div>

  <div class="timeline">
`)

	for _, event := range t.Events {
		eventClass := strings.ToLower(event.EventType.String())
		milestoneClass := ""
		if event.IsMilestone {
			milestoneClass = " milestone"
		}

		sb.WriteString(fmt.Sprintf(`    <div class="event %s%s" data-type="%s">
      <div class="event-date">%s <span class="event-type %s">%s</span></div>
      <div class="event-label">%s</div>
`,
			eventClass, milestoneClass, eventClass,
			event.Timestamp.Format("January 2, 2006"),
			eventClass, event.EventType.String(),
			html.EscapeString(event.Label)))

		if event.Description != "" {
			sb.WriteString(fmt.Sprintf(`      <div class="event-desc">%s</div>
`, html.EscapeString(event.Description)))
		}

		if len(event.Actors) > 0 {
			sb.WriteString(fmt.Sprintf(`      <div class="event-actors">Participants: %s</div>
`, html.EscapeString(strings.Join(event.Actors, ", "))))
		}

		sb.WriteString(`    </div>
`)
	}

	sb.WriteString(`  </div>

  <script>
    document.querySelectorAll('.filter-btn').forEach(btn => {
      btn.addEventListener('click', () => {
        document.querySelectorAll('.filter-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        const filter = btn.dataset.filter;
        document.querySelectorAll('.event').forEach(event => {
          if (filter === 'all' || event.dataset.type === filter) {
            event.style.display = 'block';
          } else {
            event.style.display = 'none';
          }
        });
      });
    });
  </script>
</body>
</html>`)

	return sb.String()
}

// RenderJSON renders the timeline as JSON.
func (t *Timeline) RenderJSON() (string, error) {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
