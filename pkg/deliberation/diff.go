// Package deliberation provides diff views for comparing changes between
// meetings, document versions, and deliberation states.
package deliberation

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// DiffAnchorType classifies the type of anchor for a diff.
type DiffAnchorType int

const (
	// AnchorMeeting indicates the anchor is a meeting.
	AnchorMeeting DiffAnchorType = iota
	// AnchorVersion indicates the anchor is a provision version.
	AnchorVersion
	// AnchorDate indicates the anchor is a date/time.
	AnchorDate
)

// String returns a human-readable label for the anchor type.
func (t DiffAnchorType) String() string {
	switch t {
	case AnchorMeeting:
		return "meeting"
	case AnchorVersion:
		return "version"
	case AnchorDate:
		return "date"
	default:
		return "unknown"
	}
}

// DiffAnchor represents a reference point for a diff (meeting or version).
type DiffAnchor struct {
	// Type classifies the anchor.
	Type DiffAnchorType `json:"type"`

	// URI is the unique identifier for the anchor.
	URI string `json:"uri"`

	// Label is a human-readable name.
	Label string `json:"label"`

	// Timestamp is when the anchor occurred.
	Timestamp time.Time `json:"timestamp"`

	// Sequence is the meeting number (if applicable).
	Sequence int `json:"sequence,omitempty"`
}

// TopicChangeType indicates what happened to a topic.
type TopicChangeType int

const (
	// TopicAdded indicates a new topic was added.
	TopicAdded TopicChangeType = iota
	// TopicRemoved indicates a topic was removed.
	TopicRemoved
	// TopicModified indicates a topic was modified.
	TopicModified
	// TopicUnchanged indicates a topic was not changed.
	TopicUnchanged
)

// String returns a human-readable label for the change type.
func (t TopicChangeType) String() string {
	switch t {
	case TopicAdded:
		return "added"
	case TopicRemoved:
		return "removed"
	case TopicModified:
		return "modified"
	case TopicUnchanged:
		return "unchanged"
	default:
		return "unknown"
	}
}

// TopicChange represents a change to a discussion topic between meetings.
type TopicChange struct {
	// ChangeType indicates what happened.
	ChangeType TopicChangeType `json:"change_type"`

	// TopicURI is the unique identifier for the topic.
	TopicURI string `json:"topic_uri"`

	// TopicLabel is a human-readable name.
	TopicLabel string `json:"topic_label"`

	// OldStatus is the status before the change.
	OldStatus string `json:"old_status,omitempty"`

	// NewStatus is the status after the change.
	NewStatus string `json:"new_status,omitempty"`

	// Changes describes what changed in bullet points.
	Changes []string `json:"changes,omitempty"`
}

// DiffLineType indicates the type of a diff line.
type DiffLineType int

const (
	// DiffLineUnchanged indicates the line is unchanged.
	DiffLineUnchanged DiffLineType = iota
	// DiffLineAdded indicates the line was added.
	DiffLineAdded
	// DiffLineRemoved indicates the line was removed.
	DiffLineRemoved
	// DiffLineModified indicates the line was modified.
	DiffLineModified
)

// String returns a human-readable label for the diff line type.
func (t DiffLineType) String() string {
	switch t {
	case DiffLineUnchanged:
		return "unchanged"
	case DiffLineAdded:
		return "added"
	case DiffLineRemoved:
		return "removed"
	case DiffLineModified:
		return "modified"
	default:
		return "unknown"
	}
}

// DiffLine represents a single line in a text diff.
type DiffLine struct {
	// Type indicates what happened to this line.
	Type DiffLineType `json:"type"`

	// OldLine is the line content before (if removed or modified).
	OldLine string `json:"old_line,omitempty"`

	// NewLine is the line content after (if added or modified).
	NewLine string `json:"new_line,omitempty"`

	// OldLineNum is the line number in the old text.
	OldLineNum int `json:"old_line_num,omitempty"`

	// NewLineNum is the line number in the new text.
	NewLineNum int `json:"new_line_num,omitempty"`
}

// ProvisionTextDiff represents differences between two versions of provision text.
type ProvisionTextDiff struct {
	// ProvisionURI is the provision being compared.
	ProvisionURI string `json:"provision_uri"`

	// ProvisionLabel is a human-readable name.
	ProvisionLabel string `json:"provision_label,omitempty"`

	// OldVersionURI is the earlier version.
	OldVersionURI string `json:"old_version_uri,omitempty"`

	// NewVersionURI is the later version.
	NewVersionURI string `json:"new_version_uri,omitempty"`

	// OldText is the text before changes.
	OldText string `json:"old_text"`

	// NewText is the text after changes.
	NewText string `json:"new_text"`

	// DiffLines contains the line-by-line diff.
	DiffLines []DiffLine `json:"diff_lines"`

	// ProposedBy is who proposed the change.
	ProposedBy string `json:"proposed_by,omitempty"`

	// Rationale explains why the change was made.
	Rationale string `json:"rationale,omitempty"`

	// Vote contains the vote record if applicable.
	Vote *VoteRecord `json:"vote,omitempty"`
}

// DecisionDiff represents a decision that changed between anchors.
type DecisionDiff struct {
	// Decision is the decision data.
	Decision Decision `json:"decision"`

	// ChangeType indicates whether the decision was added, closed, or modified.
	ChangeType string `json:"change_type"`
}

// ActionDiff represents an action item that changed between anchors.
type ActionDiff struct {
	// Action is the action item data.
	Action ActionItem `json:"action"`

	// ChangeType indicates whether the action was added, completed, or modified.
	ChangeType string `json:"change_type"`
}

// ParticipantDiff represents participation differences between meetings.
type ParticipantDiff struct {
	// URI is the participant's URI.
	URI string `json:"uri"`

	// Name is the participant's name.
	Name string `json:"name"`

	// InFrom indicates presence in the "from" meeting.
	InFrom bool `json:"in_from"`

	// InTo indicates presence in the "to" meeting.
	InTo bool `json:"in_to"`
}

// DeliberationDiff contains the complete diff between two anchors.
type DeliberationDiff struct {
	// From is the starting anchor.
	From DiffAnchor `json:"from"`

	// To is the ending anchor.
	To DiffAnchor `json:"to"`

	// TopicsAdded are topics newly introduced.
	TopicsAdded []TopicChange `json:"topics_added,omitempty"`

	// TopicsRemoved are topics no longer on the agenda.
	TopicsRemoved []TopicChange `json:"topics_removed,omitempty"`

	// TopicsChanged are topics with status changes.
	TopicsChanged []TopicChange `json:"topics_changed,omitempty"`

	// DecisionsNew are decisions made in the "to" meeting.
	DecisionsNew []DecisionDiff `json:"decisions_new,omitempty"`

	// DecisionsClosed are decisions closed or superseded.
	DecisionsClosed []DecisionDiff `json:"decisions_closed,omitempty"`

	// ActionsNew are action items assigned.
	ActionsNew []ActionDiff `json:"actions_new,omitempty"`

	// ActionsCompleted are action items completed.
	ActionsCompleted []ActionDiff `json:"actions_completed,omitempty"`

	// ActionsModified are action items with status changes.
	ActionsModified []ActionDiff `json:"actions_modified,omitempty"`

	// TextChanges contains provision text diffs.
	TextChanges []ProvisionTextDiff `json:"text_changes,omitempty"`

	// ParticipantChanges shows who participated in each meeting.
	ParticipantChanges []ParticipantDiff `json:"participant_changes,omitempty"`
}

// DiffSummary provides a summary of the diff.
type DiffSummary struct {
	// TopicsAdded count.
	TopicsAdded int `json:"topics_added"`

	// TopicsRemoved count.
	TopicsRemoved int `json:"topics_removed"`

	// TopicsChanged count.
	TopicsChanged int `json:"topics_changed"`

	// DecisionsNew count.
	DecisionsNew int `json:"decisions_new"`

	// DecisionsClosed count.
	DecisionsClosed int `json:"decisions_closed"`

	// ActionsNew count.
	ActionsNew int `json:"actions_new"`

	// ActionsCompleted count.
	ActionsCompleted int `json:"actions_completed"`

	// TextChanges count.
	TextChanges int `json:"text_changes"`

	// ParticipantsJoined count.
	ParticipantsJoined int `json:"participants_joined"`

	// ParticipantsLeft count.
	ParticipantsLeft int `json:"participants_left"`
}

// Summary returns a summary of the diff.
func (d *DeliberationDiff) Summary() DiffSummary {
	joined := 0
	left := 0
	for _, p := range d.ParticipantChanges {
		if p.InTo && !p.InFrom {
			joined++
		}
		if p.InFrom && !p.InTo {
			left++
		}
	}

	return DiffSummary{
		TopicsAdded:        len(d.TopicsAdded),
		TopicsRemoved:      len(d.TopicsRemoved),
		TopicsChanged:      len(d.TopicsChanged),
		DecisionsNew:       len(d.DecisionsNew),
		DecisionsClosed:    len(d.DecisionsClosed),
		ActionsNew:         len(d.ActionsNew),
		ActionsCompleted:   len(d.ActionsCompleted),
		TextChanges:        len(d.TextChanges),
		ParticipantsJoined: joined,
		ParticipantsLeft:   left,
	}
}

// DiffBuilder constructs diffs between meetings and versions.
type DiffBuilder struct {
	// store is the triple store containing deliberation data.
	store *store.TripleStore

	// baseURI is the base URI for the deliberation data.
	baseURI string
}

// NewDiffBuilder creates a new diff builder.
func NewDiffBuilder(tripleStore *store.TripleStore, baseURI string) *DiffBuilder {
	return &DiffBuilder{
		store:   tripleStore,
		baseURI: baseURI,
	}
}

// DiffConfig configures what to include in a diff.
type DiffConfig struct {
	// IncludeTopics includes topic changes.
	IncludeTopics bool

	// IncludeDecisions includes decision changes.
	IncludeDecisions bool

	// IncludeActions includes action item changes.
	IncludeActions bool

	// IncludeText includes provision text diffs.
	IncludeText bool

	// IncludeParticipants includes participation changes.
	IncludeParticipants bool

	// ProvisionFilter limits text diffs to specific provisions.
	ProvisionFilter []string

	// Since limits changes to those after this time.
	Since *time.Time
}

// DefaultDiffConfig returns the default configuration including all diff types.
func DefaultDiffConfig() DiffConfig {
	return DiffConfig{
		IncludeTopics:       true,
		IncludeDecisions:    true,
		IncludeActions:      true,
		IncludeText:         true,
		IncludeParticipants: true,
	}
}

// DiffMeetings computes the diff between two meetings.
func (b *DiffBuilder) DiffMeetings(fromMeetingURI, toMeetingURI string, config DiffConfig) (*DeliberationDiff, error) {
	if b.store == nil {
		return nil, fmt.Errorf("triple store is nil")
	}
	if fromMeetingURI == "" || toMeetingURI == "" {
		return nil, fmt.Errorf("both meeting URIs are required")
	}

	diff := &DeliberationDiff{
		From: b.buildMeetingAnchor(fromMeetingURI),
		To:   b.buildMeetingAnchor(toMeetingURI),
	}

	if config.IncludeTopics {
		b.diffTopics(fromMeetingURI, toMeetingURI, diff)
	}

	if config.IncludeDecisions {
		b.diffDecisions(fromMeetingURI, toMeetingURI, diff)
	}

	if config.IncludeActions {
		b.diffActions(fromMeetingURI, toMeetingURI, diff)
	}

	if config.IncludeText {
		b.diffTextChanges(fromMeetingURI, toMeetingURI, config.ProvisionFilter, diff)
	}

	if config.IncludeParticipants {
		b.diffParticipants(fromMeetingURI, toMeetingURI, diff)
	}

	return diff, nil
}

// buildMeetingAnchor constructs a DiffAnchor from a meeting URI.
func (b *DiffBuilder) buildMeetingAnchor(meetingURI string) DiffAnchor {
	anchor := DiffAnchor{
		Type: AnchorMeeting,
		URI:  meetingURI,
	}

	// Get label
	labelTriples := b.store.Find(meetingURI, store.RDFSLabel, "")
	if len(labelTriples) > 0 {
		anchor.Label = labelTriples[0].Object
	} else {
		titleTriples := b.store.Find(meetingURI, store.PropTitle, "")
		if len(titleTriples) > 0 {
			anchor.Label = titleTriples[0].Object
		} else {
			anchor.Label = extractURILabel(meetingURI)
		}
	}

	// Get date
	dateTriples := b.store.Find(meetingURI, store.PropMeetingDate, "")
	if len(dateTriples) > 0 {
		if d, err := time.Parse(time.RFC3339, dateTriples[0].Object); err == nil {
			anchor.Timestamp = d
		} else if d, err := time.Parse("2006-01-02", dateTriples[0].Object); err == nil {
			anchor.Timestamp = d
		}
	}

	// Get sequence
	seqTriples := b.store.Find(meetingURI, store.PropMeetingSequence, "")
	if len(seqTriples) > 0 {
		anchor.Sequence = parseInt(seqTriples[0].Object)
	}

	return anchor
}

// diffTopics computes topic changes between meetings.
func (b *DiffBuilder) diffTopics(fromMeetingURI, toMeetingURI string, diff *DeliberationDiff) {
	fromTopics := b.getMeetingTopics(fromMeetingURI)
	toTopics := b.getMeetingTopics(toMeetingURI)

	// Build maps for comparison
	fromMap := make(map[string]*topicInfo)
	for _, t := range fromTopics {
		fromMap[t.URI] = t
	}

	toMap := make(map[string]*topicInfo)
	for _, t := range toTopics {
		toMap[t.URI] = t
	}

	// Find added topics
	for uri, t := range toMap {
		if _, exists := fromMap[uri]; !exists {
			diff.TopicsAdded = append(diff.TopicsAdded, TopicChange{
				ChangeType: TopicAdded,
				TopicURI:   uri,
				TopicLabel: t.Label,
				NewStatus:  t.Status,
				Changes:    []string{"New topic added to agenda"},
			})
		}
	}

	// Find removed topics
	for uri, t := range fromMap {
		if _, exists := toMap[uri]; !exists {
			diff.TopicsRemoved = append(diff.TopicsRemoved, TopicChange{
				ChangeType: TopicRemoved,
				TopicURI:   uri,
				TopicLabel: t.Label,
				OldStatus:  t.Status,
				Changes:    []string{"Topic removed from agenda"},
			})
		}
	}

	// Find changed topics
	for uri, fromTopic := range fromMap {
		if toTopic, exists := toMap[uri]; exists {
			if fromTopic.Status != toTopic.Status {
				changes := []string{}
				if fromTopic.Status != "" && toTopic.Status != "" {
					changes = append(changes, fmt.Sprintf("Status: %s → %s", fromTopic.Status, toTopic.Status))
				}
				diff.TopicsChanged = append(diff.TopicsChanged, TopicChange{
					ChangeType: TopicModified,
					TopicURI:   uri,
					TopicLabel: toTopic.Label,
					OldStatus:  fromTopic.Status,
					NewStatus:  toTopic.Status,
					Changes:    changes,
				})
			}
		}
	}

	// Sort results
	sort.Slice(diff.TopicsAdded, func(i, j int) bool {
		return diff.TopicsAdded[i].TopicLabel < diff.TopicsAdded[j].TopicLabel
	})
	sort.Slice(diff.TopicsRemoved, func(i, j int) bool {
		return diff.TopicsRemoved[i].TopicLabel < diff.TopicsRemoved[j].TopicLabel
	})
	sort.Slice(diff.TopicsChanged, func(i, j int) bool {
		return diff.TopicsChanged[i].TopicLabel < diff.TopicsChanged[j].TopicLabel
	})
}

// topicInfo holds topic data for comparison.
type topicInfo struct {
	URI    string
	Label  string
	Status string
}

// getMeetingTopics retrieves topics discussed in a meeting.
func (b *DiffBuilder) getMeetingTopics(meetingURI string) []*topicInfo {
	var topics []*topicInfo

	// Get agenda items
	agendaTriples := b.store.Find(meetingURI, store.PropHasAgendaItem, "")
	for _, at := range agendaTriples {
		topic := &topicInfo{URI: at.Object}

		// Get label
		labelTriples := b.store.Find(at.Object, store.RDFSLabel, "")
		if len(labelTriples) > 0 {
			topic.Label = labelTriples[0].Object
		} else {
			titleTriples := b.store.Find(at.Object, store.PropTitle, "")
			if len(titleTriples) > 0 {
				topic.Label = titleTriples[0].Object
			}
		}

		// Get outcome/status
		outcomeTriples := b.store.Find(at.Object, store.PropAgendaItemOutcome, "")
		if len(outcomeTriples) > 0 {
			topic.Status = outcomeTriples[0].Object
		}

		topics = append(topics, topic)
	}

	// Also get provisions discussed
	provTriples := b.store.Find("", store.PropDiscussedAt, meetingURI)
	for _, pt := range provTriples {
		topic := &topicInfo{URI: pt.Subject}

		labelTriples := b.store.Find(pt.Subject, store.RDFSLabel, "")
		if len(labelTriples) > 0 {
			topic.Label = labelTriples[0].Object
		} else {
			topic.Label = extractURILabel(pt.Subject)
		}

		// Get status from decision if any
		decTriples := b.store.Find("", store.PropAffectsProvision, pt.Subject)
		for _, dt := range decTriples {
			meetingTriples := b.store.Find(dt.Subject, store.PropDecidedAt, meetingURI)
			if len(meetingTriples) > 0 {
				typeTriples := b.store.Find(dt.Subject, store.PropDecisionType, "")
				if len(typeTriples) > 0 {
					topic.Status = typeTriples[0].Object
				}
			}
		}

		topics = append(topics, topic)
	}

	return topics
}

// diffDecisions computes decision changes between meetings.
func (b *DiffBuilder) diffDecisions(fromMeetingURI, toMeetingURI string, diff *DeliberationDiff) {
	// Get decisions made in each meeting
	fromDecisions := b.getMeetingDecisions(fromMeetingURI)
	toDecisions := b.getMeetingDecisions(toMeetingURI)

	fromMap := make(map[string]*Decision)
	for _, d := range fromDecisions {
		fromMap[d.URI] = d
	}

	// New decisions in "to" meeting
	for _, d := range toDecisions {
		if _, exists := fromMap[d.URI]; !exists {
			diff.DecisionsNew = append(diff.DecisionsNew, DecisionDiff{
				Decision:   *d,
				ChangeType: "new",
			})
		}
	}

	// Check for superseded decisions
	for _, d := range fromDecisions {
		supersededTriples := b.store.Find(d.URI, store.PropSupersededBy, "")
		for _, st := range supersededTriples {
			// Check if superseding decision was made in the "to" meeting
			decMeetingTriples := b.store.Find(st.Object, store.PropDecidedAt, toMeetingURI)
			if len(decMeetingTriples) > 0 {
				diff.DecisionsClosed = append(diff.DecisionsClosed, DecisionDiff{
					Decision:   *d,
					ChangeType: "superseded",
				})
			}
		}
	}

	// Sort by identifier
	sort.Slice(diff.DecisionsNew, func(i, j int) bool {
		return diff.DecisionsNew[i].Decision.Identifier < diff.DecisionsNew[j].Decision.Identifier
	})
	sort.Slice(diff.DecisionsClosed, func(i, j int) bool {
		return diff.DecisionsClosed[i].Decision.Identifier < diff.DecisionsClosed[j].Decision.Identifier
	})
}

// getMeetingDecisions retrieves decisions made in a meeting.
func (b *DiffBuilder) getMeetingDecisions(meetingURI string) []*Decision {
	var decisions []*Decision

	decTriples := b.store.Find("", store.PropDecidedAt, meetingURI)
	for _, dt := range decTriples {
		dec := &Decision{URI: dt.Subject, MeetingURI: meetingURI}

		// Get identifier
		idTriples := b.store.Find(dt.Subject, store.PropIdentifier, "")
		if len(idTriples) > 0 {
			dec.Identifier = idTriples[0].Object
		}

		// Get title
		titleTriples := b.store.Find(dt.Subject, store.PropTitle, "")
		if len(titleTriples) > 0 {
			dec.Title = titleTriples[0].Object
		}

		// Get description
		descTriples := b.store.Find(dt.Subject, store.PropText, "")
		if len(descTriples) > 0 {
			dec.Description = descTriples[0].Object
		}

		// Get type
		typeTriples := b.store.Find(dt.Subject, store.PropDecisionType, "")
		if len(typeTriples) > 0 {
			dec.Type = typeTriples[0].Object
		}

		// Get affected provisions
		affectTriples := b.store.Find(dt.Subject, store.PropAffectsProvision, "")
		for _, at := range affectTriples {
			dec.AffectedProvisionURIs = append(dec.AffectedProvisionURIs, at.Object)
		}

		decisions = append(decisions, dec)
	}

	return decisions
}

// diffActions computes action item changes between meetings.
func (b *DiffBuilder) diffActions(fromMeetingURI, toMeetingURI string, diff *DeliberationDiff) {
	// Get all action items
	allActions := b.getAllActions()

	for _, action := range allActions {
		assignedAt := action.AssignedAtMeetingURI
		completedAt := action.CompletedAtMeetingURI

		// New actions assigned in "to" meeting
		if assignedAt == toMeetingURI {
			diff.ActionsNew = append(diff.ActionsNew, ActionDiff{
				Action:     *action,
				ChangeType: "assigned",
			})
		}

		// Actions completed in "to" meeting
		if completedAt == toMeetingURI {
			diff.ActionsCompleted = append(diff.ActionsCompleted, ActionDiff{
				Action:     *action,
				ChangeType: "completed",
			})
		}

		// Check for status changes between meetings
		for _, note := range action.Notes {
			if note.MeetingURI == toMeetingURI && note.UpdatedStatus != nil {
				// Find if there was a previous note in "from" meeting
				for _, prevNote := range action.Notes {
					if prevNote.MeetingURI == fromMeetingURI {
						diff.ActionsModified = append(diff.ActionsModified, ActionDiff{
							Action:     *action,
							ChangeType: "modified",
						})
						break
					}
				}
			}
		}
	}

	// Sort by identifier
	sort.Slice(diff.ActionsNew, func(i, j int) bool {
		return diff.ActionsNew[i].Action.Identifier < diff.ActionsNew[j].Action.Identifier
	})
	sort.Slice(diff.ActionsCompleted, func(i, j int) bool {
		return diff.ActionsCompleted[i].Action.Identifier < diff.ActionsCompleted[j].Action.Identifier
	})
	sort.Slice(diff.ActionsModified, func(i, j int) bool {
		return diff.ActionsModified[i].Action.Identifier < diff.ActionsModified[j].Action.Identifier
	})
}

// getAllActions retrieves all action items from the store.
func (b *DiffBuilder) getAllActions() []*ActionItem {
	var actions []*ActionItem

	actionTriples := b.store.Find("", "rdf:type", store.ClassActionItem)
	for _, at := range actionTriples {
		action := &ActionItem{URI: at.Subject}

		// Get identifier
		idTriples := b.store.Find(at.Subject, store.PropIdentifier, "")
		if len(idTriples) > 0 {
			action.Identifier = idTriples[0].Object
		}

		// Get description
		descTriples := b.store.Find(at.Subject, store.PropText, "")
		if len(descTriples) > 0 {
			action.Description = descTriples[0].Object
		}

		// Get assigned meeting
		assignedTriples := b.store.Find(at.Subject, store.PropActionAssignedAt, "")
		if len(assignedTriples) > 0 {
			action.AssignedAtMeetingURI = assignedTriples[0].Object
		}

		// Get completed meeting
		completedTriples := b.store.Find(at.Subject, store.PropActionCompletedAt, "")
		if len(completedTriples) > 0 {
			action.CompletedAtMeetingURI = completedTriples[0].Object
		}

		// Get status
		statusTriples := b.store.Find(at.Subject, store.PropActionStatus, "")
		if len(statusTriples) > 0 {
			action.Status = parseActionStatusString(statusTriples[0].Object)
		}

		// Get assignees
		assigneeTriples := b.store.Find(at.Subject, store.PropActionAssignedTo, "")
		for _, ast := range assigneeTriples {
			action.AssignedToURIs = append(action.AssignedToURIs, ast.Object)
			action.AssignedToNames = append(action.AssignedToNames, b.getLabel(ast.Object))
		}

		actions = append(actions, action)
	}

	return actions
}

// parseActionStatusString converts a string to ActionItemStatus.
// Note: uses different name to avoid conflict with parseActionStatus in actions.go
func parseActionStatusString(s string) ActionItemStatus {
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

// diffTextChanges computes provision text changes between meetings.
func (b *DiffBuilder) diffTextChanges(fromMeetingURI, toMeetingURI string, provisionFilter []string, diff *DeliberationDiff) {
	// Find provisions that were amended between meetings
	amendedTriples := b.store.Find("", "reg:amendedAt", toMeetingURI)

	filterMap := make(map[string]bool)
	for _, p := range provisionFilter {
		filterMap[p] = true
	}

	for _, at := range amendedTriples {
		versionURI := at.Subject

		// Get the provision this version belongs to
		provTriples := b.store.Find(versionURI, store.PropVersionOf, "")
		if len(provTriples) == 0 {
			continue
		}
		provisionURI := provTriples[0].Object

		// Apply filter if specified
		if len(filterMap) > 0 && !filterMap[provisionURI] {
			continue
		}

		textDiff := ProvisionTextDiff{
			ProvisionURI:  provisionURI,
			NewVersionURI: versionURI,
		}

		// Get provision label
		labelTriples := b.store.Find(provisionURI, store.RDFSLabel, "")
		if len(labelTriples) > 0 {
			textDiff.ProvisionLabel = labelTriples[0].Object
		} else {
			textDiff.ProvisionLabel = extractURILabel(provisionURI)
		}

		// Get new text
		newTextTriples := b.store.Find(versionURI, store.PropText, "")
		if len(newTextTriples) > 0 {
			textDiff.NewText = newTextTriples[0].Object
		}

		// Get previous version
		prevTriples := b.store.Find(versionURI, store.PropPreviousVersion, "")
		if len(prevTriples) > 0 {
			textDiff.OldVersionURI = prevTriples[0].Object

			// Get old text
			oldTextTriples := b.store.Find(prevTriples[0].Object, store.PropText, "")
			if len(oldTextTriples) > 0 {
				textDiff.OldText = oldTextTriples[0].Object
			}
		}

		// Get proposer
		proposerTriples := b.store.Find(versionURI, store.PropProposedBy, "")
		if len(proposerTriples) > 0 {
			textDiff.ProposedBy = b.getLabel(proposerTriples[0].Object)
		}

		// Compute line diff
		textDiff.DiffLines = computeLineDiff(textDiff.OldText, textDiff.NewText)

		diff.TextChanges = append(diff.TextChanges, textDiff)
	}

	// Sort by provision label
	sort.Slice(diff.TextChanges, func(i, j int) bool {
		return diff.TextChanges[i].ProvisionLabel < diff.TextChanges[j].ProvisionLabel
	})
}

// computeLineDiff performs a line-by-line diff between two texts.
func computeLineDiff(oldText, newText string) []DiffLine {
	var diffLines []DiffLine

	oldLines := strings.Split(oldText, "\n")
	newLines := strings.Split(newText, "\n")

	// Simple LCS-based diff
	lcs := computeLCS(oldLines, newLines)

	oldIdx := 0
	newIdx := 0
	lcsIdx := 0

	for oldIdx < len(oldLines) || newIdx < len(newLines) {
		if lcsIdx < len(lcs) {
			// Skip old lines not in LCS (removed)
			for oldIdx < len(oldLines) && oldLines[oldIdx] != lcs[lcsIdx] {
				diffLines = append(diffLines, DiffLine{
					Type:       DiffLineRemoved,
					OldLine:    oldLines[oldIdx],
					OldLineNum: oldIdx + 1,
				})
				oldIdx++
			}

			// Skip new lines not in LCS (added)
			for newIdx < len(newLines) && newLines[newIdx] != lcs[lcsIdx] {
				diffLines = append(diffLines, DiffLine{
					Type:       DiffLineAdded,
					NewLine:    newLines[newIdx],
					NewLineNum: newIdx + 1,
				})
				newIdx++
			}

			// Matching line
			if oldIdx < len(oldLines) && newIdx < len(newLines) {
				diffLines = append(diffLines, DiffLine{
					Type:       DiffLineUnchanged,
					OldLine:    oldLines[oldIdx],
					NewLine:    newLines[newIdx],
					OldLineNum: oldIdx + 1,
					NewLineNum: newIdx + 1,
				})
				oldIdx++
				newIdx++
				lcsIdx++
			}
		} else {
			// Remaining old lines are removed
			for oldIdx < len(oldLines) {
				diffLines = append(diffLines, DiffLine{
					Type:       DiffLineRemoved,
					OldLine:    oldLines[oldIdx],
					OldLineNum: oldIdx + 1,
				})
				oldIdx++
			}

			// Remaining new lines are added
			for newIdx < len(newLines) {
				diffLines = append(diffLines, DiffLine{
					Type:       DiffLineAdded,
					NewLine:    newLines[newIdx],
					NewLineNum: newIdx + 1,
				})
				newIdx++
			}
		}
	}

	return diffLines
}

// computeLCS computes the longest common subsequence of two string slices.
func computeLCS(a, b []string) []string {
	m := len(a)
	n := len(b)

	// Build LCS length table
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				if dp[i-1][j] > dp[i][j-1] {
					dp[i][j] = dp[i-1][j]
				} else {
					dp[i][j] = dp[i][j-1]
				}
			}
		}
	}

	// Backtrack to find LCS
	var lcs []string
	i, j := m, n
	for i > 0 && j > 0 {
		if a[i-1] == b[j-1] {
			lcs = append([]string{a[i-1]}, lcs...)
			i--
			j--
		} else if dp[i-1][j] > dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	return lcs
}

// diffParticipants computes participation changes between meetings.
func (b *DiffBuilder) diffParticipants(fromMeetingURI, toMeetingURI string, diff *DeliberationDiff) {
	fromParticipants := b.getMeetingParticipants(fromMeetingURI)
	toParticipants := b.getMeetingParticipants(toMeetingURI)

	// Build maps
	fromMap := make(map[string]string)
	for _, p := range fromParticipants {
		fromMap[p.URI] = p.Name
	}

	toMap := make(map[string]string)
	for _, p := range toParticipants {
		toMap[p.URI] = p.Name
	}

	// Find all unique participants
	allURIs := make(map[string]bool)
	for uri := range fromMap {
		allURIs[uri] = true
	}
	for uri := range toMap {
		allURIs[uri] = true
	}

	for uri := range allURIs {
		inFrom := fromMap[uri] != ""
		inTo := toMap[uri] != ""

		name := fromMap[uri]
		if name == "" {
			name = toMap[uri]
		}

		diff.ParticipantChanges = append(diff.ParticipantChanges, ParticipantDiff{
			URI:    uri,
			Name:   name,
			InFrom: inFrom,
			InTo:   inTo,
		})
	}

	// Sort by name
	sort.Slice(diff.ParticipantChanges, func(i, j int) bool {
		return diff.ParticipantChanges[i].Name < diff.ParticipantChanges[j].Name
	})
}

// participantInfo holds participant data.
type participantInfo struct {
	URI  string
	Name string
}

// getMeetingParticipants retrieves participants from a meeting.
func (b *DiffBuilder) getMeetingParticipants(meetingURI string) []participantInfo {
	var participants []participantInfo

	partTriples := b.store.Find(meetingURI, store.PropParticipant, "")
	for _, pt := range partTriples {
		p := participantInfo{URI: pt.Object}
		p.Name = b.getLabel(pt.Object)
		participants = append(participants, p)
	}

	return participants
}

// getLabel retrieves a human-readable label for a URI.
func (b *DiffBuilder) getLabel(uri string) string {
	labelTriples := b.store.Find(uri, store.RDFSLabel, "")
	if len(labelTriples) > 0 {
		return labelTriples[0].Object
	}
	titleTriples := b.store.Find(uri, store.PropTitle, "")
	if len(titleTriples) > 0 {
		return titleTriples[0].Object
	}
	return extractURILabel(uri)
}

// DiffProvisionVersions computes the diff between two versions of a provision.
func (b *DiffBuilder) DiffProvisionVersions(fromVersionURI, toVersionURI string) (*ProvisionTextDiff, error) {
	if b.store == nil {
		return nil, fmt.Errorf("triple store is nil")
	}
	if fromVersionURI == "" || toVersionURI == "" {
		return nil, fmt.Errorf("both version URIs are required")
	}

	diff := &ProvisionTextDiff{
		OldVersionURI: fromVersionURI,
		NewVersionURI: toVersionURI,
	}

	// Get provision URI
	provTriples := b.store.Find(fromVersionURI, store.PropVersionOf, "")
	if len(provTriples) > 0 {
		diff.ProvisionURI = provTriples[0].Object
		diff.ProvisionLabel = b.getLabel(provTriples[0].Object)
	}

	// Get old text
	oldTextTriples := b.store.Find(fromVersionURI, store.PropText, "")
	if len(oldTextTriples) > 0 {
		diff.OldText = oldTextTriples[0].Object
	}

	// Get new text
	newTextTriples := b.store.Find(toVersionURI, store.PropText, "")
	if len(newTextTriples) > 0 {
		diff.NewText = newTextTriples[0].Object
	}

	// Get proposer
	proposerTriples := b.store.Find(toVersionURI, store.PropProposedBy, "")
	if len(proposerTriples) > 0 {
		diff.ProposedBy = b.getLabel(proposerTriples[0].Object)
	}

	// Compute line diff
	diff.DiffLines = computeLineDiff(diff.OldText, diff.NewText)

	return diff, nil
}

// DiffSince computes all changes since a given time.
func (b *DiffBuilder) DiffSince(since time.Time, config DiffConfig) (*DeliberationDiff, error) {
	if b.store == nil {
		return nil, fmt.Errorf("triple store is nil")
	}

	diff := &DeliberationDiff{
		From: DiffAnchor{
			Type:      AnchorDate,
			Label:     since.Format("2006-01-02"),
			Timestamp: since,
		},
		To: DiffAnchor{
			Type:      AnchorDate,
			Label:     "now",
			Timestamp: time.Now(),
		},
	}

	// Find meetings since the given time
	meetingTriples := b.store.Find("", "rdf:type", store.ClassMeeting)
	var recentMeetings []string

	for _, mt := range meetingTriples {
		dateTriples := b.store.Find(mt.Subject, store.PropMeetingDate, "")
		if len(dateTriples) > 0 {
			if d, err := time.Parse(time.RFC3339, dateTriples[0].Object); err == nil && d.After(since) {
				recentMeetings = append(recentMeetings, mt.Subject)
			} else if d, err := time.Parse("2006-01-02", dateTriples[0].Object); err == nil && d.After(since) {
				recentMeetings = append(recentMeetings, mt.Subject)
			}
		}
	}

	// Aggregate changes from all recent meetings
	for _, meetingURI := range recentMeetings {
		if config.IncludeDecisions {
			decisions := b.getMeetingDecisions(meetingURI)
			for _, d := range decisions {
				diff.DecisionsNew = append(diff.DecisionsNew, DecisionDiff{
					Decision:   *d,
					ChangeType: "new",
				})
			}
		}

		if config.IncludeActions {
			actions := b.getAllActions()
			for _, a := range actions {
				if a.AssignedAtMeetingURI == meetingURI {
					diff.ActionsNew = append(diff.ActionsNew, ActionDiff{
						Action:     *a,
						ChangeType: "assigned",
					})
				}
				if a.CompletedAtMeetingURI == meetingURI {
					diff.ActionsCompleted = append(diff.ActionsCompleted, ActionDiff{
						Action:     *a,
						ChangeType: "completed",
					})
				}
			}
		}

		if config.IncludeText {
			b.diffTextChanges("", meetingURI, config.ProvisionFilter, diff)
		}
	}

	return diff, nil
}

// RenderUnified renders the diff in unified diff format for terminal output.
func (d *DeliberationDiff) RenderUnified() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Diff: %s → %s\n", d.From.Label, d.To.Label))
	sb.WriteString("===========================\n\n")

	// Topics section
	if len(d.TopicsAdded) > 0 || len(d.TopicsRemoved) > 0 || len(d.TopicsChanged) > 0 {
		sb.WriteString("Topics:\n")
		for _, t := range d.TopicsAdded {
			sb.WriteString(fmt.Sprintf("  + %s (NEW", t.TopicLabel))
			if t.NewStatus != "" {
				sb.WriteString(fmt.Sprintf(" - %s", t.NewStatus))
			}
			sb.WriteString(")\n")
		}
		for _, t := range d.TopicsChanged {
			sb.WriteString(fmt.Sprintf("  ~ %s (%s → %s)\n", t.TopicLabel, t.OldStatus, t.NewStatus))
		}
		for _, t := range d.TopicsRemoved {
			sb.WriteString(fmt.Sprintf("  - %s (%s - not on agenda)\n", t.TopicLabel, t.OldStatus))
		}
		sb.WriteString("\n")
	}

	// Decisions section
	if len(d.DecisionsNew) > 0 || len(d.DecisionsClosed) > 0 {
		sb.WriteString("Decisions:\n")
		for _, dec := range d.DecisionsNew {
			sb.WriteString(fmt.Sprintf("  + %s\n", dec.Decision.Title))
		}
		for _, dec := range d.DecisionsClosed {
			sb.WriteString(fmt.Sprintf("  - %s (%s)\n", dec.Decision.Title, dec.ChangeType))
		}
		sb.WriteString("\n")
	}

	// Actions section
	if len(d.ActionsNew) > 0 || len(d.ActionsCompleted) > 0 {
		sb.WriteString("Actions:\n")
		for _, a := range d.ActionsNew {
			sb.WriteString(fmt.Sprintf("  + %s\n", a.Action.Description))
		}
		for _, a := range d.ActionsCompleted {
			sb.WriteString(fmt.Sprintf("  ✓ %s (completed)\n", a.Action.Description))
		}
		sb.WriteString("\n")
	}

	// Text changes section
	for _, tc := range d.TextChanges {
		sb.WriteString(fmt.Sprintf("%s Text Changes:\n", tc.ProvisionLabel))
		sb.WriteString("─────────────────────────\n")

		if tc.ProposedBy != "" {
			sb.WriteString(fmt.Sprintf("  Proposed by: %s\n", tc.ProposedBy))
		}
		if tc.Rationale != "" {
			sb.WriteString(fmt.Sprintf("  Rationale: %s\n", tc.Rationale))
		}
		if tc.Vote != nil {
			sb.WriteString(fmt.Sprintf("  Vote: %s (%d-%d-%d)\n",
				tc.Vote.Result, tc.Vote.ForCount, tc.Vote.AgainstCount, tc.Vote.AbstainCount))
		}
		sb.WriteString("\n")

		// Render diff lines
		for _, line := range tc.DiffLines {
			switch line.Type {
			case DiffLineAdded:
				sb.WriteString(fmt.Sprintf("  + %s\n", line.NewLine))
			case DiffLineRemoved:
				sb.WriteString(fmt.Sprintf("  - %s\n", line.OldLine))
			case DiffLineUnchanged:
				sb.WriteString(fmt.Sprintf("    %s\n", line.OldLine))
			}
		}
		sb.WriteString("\n")
	}

	// Participation changes
	joined := []string{}
	left := []string{}
	for _, p := range d.ParticipantChanges {
		if p.InTo && !p.InFrom {
			joined = append(joined, p.Name)
		}
		if p.InFrom && !p.InTo {
			left = append(left, p.Name)
		}
	}
	if len(joined) > 0 || len(left) > 0 {
		sb.WriteString("Participation:\n")
		for _, name := range joined {
			sb.WriteString(fmt.Sprintf("  + %s (joined)\n", name))
		}
		for _, name := range left {
			sb.WriteString(fmt.Sprintf("  - %s (absent)\n", name))
		}
	}

	return sb.String()
}

// RenderHTML renders the diff as HTML with syntax highlighting.
func (d *DeliberationDiff) RenderHTML() string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>Deliberation Diff</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 20px; }
h1 { color: #333; }
h2 { color: #555; border-bottom: 1px solid #ddd; padding-bottom: 5px; }
.diff-header { background: #f5f5f5; padding: 15px; border-radius: 5px; margin-bottom: 20px; }
.added { background: #e6ffec; color: #24292f; }
.removed { background: #ffebe9; color: #24292f; }
.modified { background: #fff8c5; color: #24292f; }
.unchanged { color: #666; }
.topic, .decision, .action { padding: 8px; margin: 5px 0; border-radius: 3px; }
.topic.added, .decision.added, .action.added { border-left: 3px solid #28a745; }
.topic.removed, .decision.removed, .action.removed { border-left: 3px solid #d73a49; }
.topic.modified { border-left: 3px solid #f9c513; }
.diff-line { font-family: 'Monaco', 'Menlo', monospace; font-size: 13px; padding: 2px 10px; }
.line-num { color: #999; min-width: 40px; display: inline-block; }
.side-by-side { display: flex; gap: 20px; }
.side { flex: 1; }
.summary { display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 10px; }
.summary-item { background: #f5f5f5; padding: 10px; border-radius: 5px; text-align: center; }
.summary-count { font-size: 24px; font-weight: bold; }
.participant-table { border-collapse: collapse; width: 100%; }
.participant-table th, .participant-table td { border: 1px solid #ddd; padding: 8px; text-align: left; }
.participant-table th { background: #f5f5f5; }
.check { color: #28a745; }
.cross { color: #d73a49; }
</style>
</head>
<body>
`)

	// Header
	sb.WriteString(fmt.Sprintf(`<div class="diff-header">
<h1>Diff: %s → %s</h1>
`, d.From.Label, d.To.Label))

	if !d.From.Timestamp.IsZero() && !d.To.Timestamp.IsZero() {
		sb.WriteString(fmt.Sprintf("<p>%s to %s</p>",
			d.From.Timestamp.Format("2006-01-02"),
			d.To.Timestamp.Format("2006-01-02")))
	}
	sb.WriteString("</div>\n")

	// Summary
	summary := d.Summary()
	sb.WriteString(`<h2>Summary</h2>
<div class="summary">
`)
	if summary.TopicsAdded > 0 {
		sb.WriteString(fmt.Sprintf(`<div class="summary-item added"><div class="summary-count">+%d</div>Topics Added</div>`, summary.TopicsAdded))
	}
	if summary.TopicsRemoved > 0 {
		sb.WriteString(fmt.Sprintf(`<div class="summary-item removed"><div class="summary-count">-%d</div>Topics Removed</div>`, summary.TopicsRemoved))
	}
	if summary.DecisionsNew > 0 {
		sb.WriteString(fmt.Sprintf(`<div class="summary-item added"><div class="summary-count">+%d</div>New Decisions</div>`, summary.DecisionsNew))
	}
	if summary.ActionsNew > 0 {
		sb.WriteString(fmt.Sprintf(`<div class="summary-item added"><div class="summary-count">+%d</div>Actions Assigned</div>`, summary.ActionsNew))
	}
	if summary.ActionsCompleted > 0 {
		sb.WriteString(fmt.Sprintf(`<div class="summary-item"><div class="summary-count">✓%d</div>Actions Completed</div>`, summary.ActionsCompleted))
	}
	if summary.TextChanges > 0 {
		sb.WriteString(fmt.Sprintf(`<div class="summary-item modified"><div class="summary-count">~%d</div>Text Changes</div>`, summary.TextChanges))
	}
	sb.WriteString("</div>\n")

	// Topics
	if len(d.TopicsAdded) > 0 || len(d.TopicsRemoved) > 0 || len(d.TopicsChanged) > 0 {
		sb.WriteString("<h2>Topics</h2>\n")
		for _, t := range d.TopicsAdded {
			sb.WriteString(fmt.Sprintf(`<div class="topic added"><strong>+ %s</strong>`, t.TopicLabel))
			if t.NewStatus != "" {
				sb.WriteString(fmt.Sprintf(" <em>(%s)</em>", t.NewStatus))
			}
			sb.WriteString("</div>\n")
		}
		for _, t := range d.TopicsChanged {
			sb.WriteString(fmt.Sprintf(`<div class="topic modified"><strong>~ %s</strong> <em>%s → %s</em></div>`, t.TopicLabel, t.OldStatus, t.NewStatus))
		}
		for _, t := range d.TopicsRemoved {
			sb.WriteString(fmt.Sprintf(`<div class="topic removed"><strong>- %s</strong> <em>(%s)</em></div>`, t.TopicLabel, t.OldStatus))
		}
	}

	// Decisions
	if len(d.DecisionsNew) > 0 || len(d.DecisionsClosed) > 0 {
		sb.WriteString("<h2>Decisions</h2>\n")
		for _, dec := range d.DecisionsNew {
			sb.WriteString(fmt.Sprintf(`<div class="decision added"><strong>+ %s</strong>`, dec.Decision.Title))
			if dec.Decision.Description != "" {
				sb.WriteString(fmt.Sprintf("<p>%s</p>", dec.Decision.Description))
			}
			sb.WriteString("</div>\n")
		}
		for _, dec := range d.DecisionsClosed {
			sb.WriteString(fmt.Sprintf(`<div class="decision removed"><strong>- %s</strong> <em>(%s)</em></div>`, dec.Decision.Title, dec.ChangeType))
		}
	}

	// Actions
	if len(d.ActionsNew) > 0 || len(d.ActionsCompleted) > 0 {
		sb.WriteString("<h2>Actions</h2>\n")
		for _, a := range d.ActionsNew {
			sb.WriteString(fmt.Sprintf(`<div class="action added"><strong>+ %s</strong>`, a.Action.Description))
			if len(a.Action.AssignedToNames) > 0 {
				sb.WriteString(fmt.Sprintf(" <em>(Assigned to: %s)</em>", strings.Join(a.Action.AssignedToNames, ", ")))
			}
			sb.WriteString("</div>\n")
		}
		for _, a := range d.ActionsCompleted {
			sb.WriteString(fmt.Sprintf(`<div class="action"><strong>✓ %s</strong> <em>(completed)</em></div>`, a.Action.Description))
		}
	}

	// Text changes
	for _, tc := range d.TextChanges {
		sb.WriteString(fmt.Sprintf("<h2>%s Text Changes</h2>\n", tc.ProvisionLabel))

		if tc.ProposedBy != "" {
			sb.WriteString(fmt.Sprintf("<p><strong>Proposed by:</strong> %s</p>\n", tc.ProposedBy))
		}
		if tc.Rationale != "" {
			sb.WriteString(fmt.Sprintf("<p><strong>Rationale:</strong> %s</p>\n", tc.Rationale))
		}
		if tc.Vote != nil {
			sb.WriteString(fmt.Sprintf("<p><strong>Vote:</strong> %s (%d for, %d against, %d abstain)</p>\n",
				tc.Vote.Result, tc.Vote.ForCount, tc.Vote.AgainstCount, tc.Vote.AbstainCount))
		}

		// Side-by-side view
		sb.WriteString(`<div class="side-by-side">
<div class="side"><h3>Before</h3>`)
		for _, line := range tc.DiffLines {
			switch line.Type {
			case DiffLineRemoved:
				sb.WriteString(fmt.Sprintf(`<div class="diff-line removed"><span class="line-num">%d</span>%s</div>`, line.OldLineNum, line.OldLine))
			case DiffLineUnchanged:
				sb.WriteString(fmt.Sprintf(`<div class="diff-line unchanged"><span class="line-num">%d</span>%s</div>`, line.OldLineNum, line.OldLine))
			}
		}
		sb.WriteString("</div>\n")

		sb.WriteString(`<div class="side"><h3>After</h3>`)
		for _, line := range tc.DiffLines {
			switch line.Type {
			case DiffLineAdded:
				sb.WriteString(fmt.Sprintf(`<div class="diff-line added"><span class="line-num">%d</span>%s</div>`, line.NewLineNum, line.NewLine))
			case DiffLineUnchanged:
				sb.WriteString(fmt.Sprintf(`<div class="diff-line unchanged"><span class="line-num">%d</span>%s</div>`, line.NewLineNum, line.NewLine))
			}
		}
		sb.WriteString("</div></div>\n")
	}

	// Participation
	if len(d.ParticipantChanges) > 0 {
		sb.WriteString("<h2>Participation</h2>\n")
		sb.WriteString(`<table class="participant-table">
<tr><th>Participant</th><th>`)
		sb.WriteString(d.From.Label)
		sb.WriteString("</th><th>")
		sb.WriteString(d.To.Label)
		sb.WriteString("</th></tr>\n")

		for _, p := range d.ParticipantChanges {
			sb.WriteString("<tr><td>")
			sb.WriteString(p.Name)
			sb.WriteString("</td><td>")
			if p.InFrom {
				sb.WriteString(`<span class="check">✓</span>`)
			} else {
				sb.WriteString(`<span class="cross">✗</span>`)
			}
			sb.WriteString("</td><td>")
			if p.InTo {
				sb.WriteString(`<span class="check">✓</span>`)
			} else {
				sb.WriteString(`<span class="cross">✗</span>`)
			}
			sb.WriteString("</td></tr>\n")
		}
		sb.WriteString("</table>\n")
	}

	sb.WriteString("</body>\n</html>")
	return sb.String()
}

// RenderJSON renders the diff as JSON.
func (d *DeliberationDiff) RenderJSON() (string, error) {
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RenderMarkdown renders the diff as Markdown.
func (d *DeliberationDiff) RenderMarkdown() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Diff: %s → %s\n\n", d.From.Label, d.To.Label))

	if !d.From.Timestamp.IsZero() && !d.To.Timestamp.IsZero() {
		sb.WriteString(fmt.Sprintf("**Period:** %s to %s\n\n",
			d.From.Timestamp.Format("2006-01-02"),
			d.To.Timestamp.Format("2006-01-02")))
	}

	// Summary
	summary := d.Summary()
	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Category | Count |\n")
	sb.WriteString("|----------|-------|\n")
	if summary.TopicsAdded > 0 {
		sb.WriteString(fmt.Sprintf("| Topics Added | +%d |\n", summary.TopicsAdded))
	}
	if summary.TopicsRemoved > 0 {
		sb.WriteString(fmt.Sprintf("| Topics Removed | -%d |\n", summary.TopicsRemoved))
	}
	if summary.TopicsChanged > 0 {
		sb.WriteString(fmt.Sprintf("| Topics Changed | ~%d |\n", summary.TopicsChanged))
	}
	if summary.DecisionsNew > 0 {
		sb.WriteString(fmt.Sprintf("| New Decisions | +%d |\n", summary.DecisionsNew))
	}
	if summary.ActionsNew > 0 {
		sb.WriteString(fmt.Sprintf("| Actions Assigned | +%d |\n", summary.ActionsNew))
	}
	if summary.ActionsCompleted > 0 {
		sb.WriteString(fmt.Sprintf("| Actions Completed | ✓%d |\n", summary.ActionsCompleted))
	}
	if summary.TextChanges > 0 {
		sb.WriteString(fmt.Sprintf("| Text Changes | ~%d |\n", summary.TextChanges))
	}
	sb.WriteString("\n")

	// Topics
	if len(d.TopicsAdded) > 0 || len(d.TopicsRemoved) > 0 || len(d.TopicsChanged) > 0 {
		sb.WriteString("## Topics\n\n")
		for _, t := range d.TopicsAdded {
			sb.WriteString(fmt.Sprintf("- **+** %s", t.TopicLabel))
			if t.NewStatus != "" {
				sb.WriteString(fmt.Sprintf(" _(NEW - %s)_", t.NewStatus))
			}
			sb.WriteString("\n")
		}
		for _, t := range d.TopicsChanged {
			sb.WriteString(fmt.Sprintf("- **~** %s _%s → %s_\n", t.TopicLabel, t.OldStatus, t.NewStatus))
		}
		for _, t := range d.TopicsRemoved {
			sb.WriteString(fmt.Sprintf("- **-** %s _%s_\n", t.TopicLabel, t.OldStatus))
		}
		sb.WriteString("\n")
	}

	// Decisions
	if len(d.DecisionsNew) > 0 || len(d.DecisionsClosed) > 0 {
		sb.WriteString("## Decisions\n\n")
		for _, dec := range d.DecisionsNew {
			sb.WriteString(fmt.Sprintf("- **+** %s\n", dec.Decision.Title))
			if dec.Decision.Description != "" {
				sb.WriteString(fmt.Sprintf("  - %s\n", dec.Decision.Description))
			}
		}
		for _, dec := range d.DecisionsClosed {
			sb.WriteString(fmt.Sprintf("- **-** %s (%s)\n", dec.Decision.Title, dec.ChangeType))
		}
		sb.WriteString("\n")
	}

	// Actions
	if len(d.ActionsNew) > 0 || len(d.ActionsCompleted) > 0 {
		sb.WriteString("## Actions\n\n")
		for _, a := range d.ActionsNew {
			sb.WriteString(fmt.Sprintf("- **+** %s", a.Action.Description))
			if len(a.Action.AssignedToNames) > 0 {
				sb.WriteString(fmt.Sprintf(" _(Assigned to: %s)_", strings.Join(a.Action.AssignedToNames, ", ")))
			}
			sb.WriteString("\n")
		}
		for _, a := range d.ActionsCompleted {
			sb.WriteString(fmt.Sprintf("- **✓** %s _(completed)_\n", a.Action.Description))
		}
		sb.WriteString("\n")
	}

	// Text changes
	for _, tc := range d.TextChanges {
		sb.WriteString(fmt.Sprintf("## %s Text Changes\n\n", tc.ProvisionLabel))

		if tc.ProposedBy != "" {
			sb.WriteString(fmt.Sprintf("**Proposed by:** %s\n\n", tc.ProposedBy))
		}
		if tc.Rationale != "" {
			sb.WriteString(fmt.Sprintf("**Rationale:** %s\n\n", tc.Rationale))
		}
		if tc.Vote != nil {
			sb.WriteString(fmt.Sprintf("**Vote:** %s (%d for, %d against, %d abstain)\n\n",
				tc.Vote.Result, tc.Vote.ForCount, tc.Vote.AgainstCount, tc.Vote.AbstainCount))
		}

		sb.WriteString("```diff\n")
		for _, line := range tc.DiffLines {
			switch line.Type {
			case DiffLineAdded:
				sb.WriteString(fmt.Sprintf("+ %s\n", line.NewLine))
			case DiffLineRemoved:
				sb.WriteString(fmt.Sprintf("- %s\n", line.OldLine))
			case DiffLineUnchanged:
				sb.WriteString(fmt.Sprintf("  %s\n", line.OldLine))
			}
		}
		sb.WriteString("```\n\n")
	}

	// Participation
	if len(d.ParticipantChanges) > 0 {
		sb.WriteString("## Participation\n\n")
		sb.WriteString(fmt.Sprintf("| Participant | %s | %s |\n", d.From.Label, d.To.Label))
		sb.WriteString("|-------------|------|------|\n")

		for _, p := range d.ParticipantChanges {
			fromMark := "✗"
			toMark := "✗"
			if p.InFrom {
				fromMark = "✓"
			}
			if p.InTo {
				toMark = "✓"
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", p.Name, fromMark, toMark))
		}
	}

	return sb.String()
}
