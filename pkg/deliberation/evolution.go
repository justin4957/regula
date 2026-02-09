// Package deliberation provides types and functions for modeling deliberation
// documents including provision evolution tracking across document versions.
package deliberation

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// EventType classifies the type of evolution event.
type EventType int

const (
	// EventProposed indicates a new provision was proposed.
	EventProposed EventType = iota
	// EventAmended indicates a provision was amended.
	EventAmended
	// EventAdopted indicates a provision was adopted.
	EventAdopted
	// EventRejected indicates a provision or amendment was rejected.
	EventRejected
	// EventWithdrawn indicates a provision or amendment was withdrawn.
	EventWithdrawn
	// EventSuperseded indicates a provision was superseded by a new version.
	EventSuperseded
	// EventRepealed indicates a provision was repealed.
	EventRepealed
)

// String returns a human-readable label for the event type.
func (e EventType) String() string {
	switch e {
	case EventProposed:
		return "proposed"
	case EventAmended:
		return "amended"
	case EventAdopted:
		return "adopted"
	case EventRejected:
		return "rejected"
	case EventWithdrawn:
		return "withdrawn"
	case EventSuperseded:
		return "superseded"
	case EventRepealed:
		return "repealed"
	default:
		return "unknown"
	}
}

// ParseEventType converts a string to an EventType.
func ParseEventType(s string) EventType {
	switch strings.ToLower(s) {
	case "proposed":
		return EventProposed
	case "amended":
		return EventAmended
	case "adopted":
		return EventAdopted
	case "rejected":
		return EventRejected
	case "withdrawn":
		return EventWithdrawn
	case "superseded":
		return EventSuperseded
	case "repealed":
		return EventRepealed
	default:
		return EventProposed
	}
}

// VersionEvent represents a single event in a provision's evolution.
type VersionEvent struct {
	// VersionURI is the URI of this version.
	VersionURI string `json:"version_uri"`

	// Version is the version identifier (e.g., "v1", "v2").
	Version string `json:"version"`

	// Date is when the event occurred.
	Date time.Time `json:"date"`

	// MeetingURI is the meeting where this event occurred.
	MeetingURI string `json:"meeting_uri,omitempty"`

	// MeetingName is a human-readable name for the meeting.
	MeetingName string `json:"meeting_name,omitempty"`

	// EventType classifies the event.
	EventType EventType `json:"event_type"`

	// ProposedBy is the stakeholder who proposed this change.
	ProposedBy string `json:"proposed_by,omitempty"`

	// ProposerName is the human-readable name of the proposer.
	ProposerName string `json:"proposer_name,omitempty"`

	// Text is the provision text at this version.
	Text string `json:"text,omitempty"`

	// PreviousVersionURI links to the version this amends/supersedes.
	PreviousVersionURI string `json:"previous_version_uri,omitempty"`

	// Vote contains the vote record for adoption/rejection.
	Vote *VoteRecord `json:"vote,omitempty"`

	// Notes contains additional context about the event.
	Notes string `json:"notes,omitempty"`
}

// TimelineEntry represents a point in time in the evolution timeline.
type TimelineEntry struct {
	// Date is the date of the timeline entry.
	Date time.Time `json:"date"`

	// Label is a human-readable label for display.
	Label string `json:"label"`

	// Description provides more detail about what happened.
	Description string `json:"description"`

	// EventType classifies the type of event.
	EventType EventType `json:"event_type"`

	// VersionURI is the version associated with this entry.
	VersionURI string `json:"version_uri,omitempty"`

	// MeetingURI is the meeting where this occurred.
	MeetingURI string `json:"meeting_uri,omitempty"`

	// ProposedBy is who made this change.
	ProposedBy string `json:"proposed_by,omitempty"`
}

// TextDiff represents differences between two versions.
type TextDiff struct {
	// Version1URI is the URI of the first version.
	Version1URI string `json:"version1_uri"`

	// Version2URI is the URI of the second version.
	Version2URI string `json:"version2_uri"`

	// Text1 is the text of the first version.
	Text1 string `json:"text1"`

	// Text2 is the text of the second version.
	Text2 string `json:"text2"`

	// Changes describes the changes between versions.
	Changes []TextChange `json:"changes,omitempty"`
}

// TextChange represents a single change between versions.
type TextChange struct {
	// Type is "addition", "deletion", or "modification".
	Type string `json:"type"`

	// OldText is the text before the change.
	OldText string `json:"old_text,omitempty"`

	// NewText is the text after the change.
	NewText string `json:"new_text,omitempty"`

	// Position is the approximate character position.
	Position int `json:"position,omitempty"`
}

// ProvisionEvolution contains the complete evolution history of a provision.
type ProvisionEvolution struct {
	// ProvisionURI is the canonical URI of the provision.
	ProvisionURI string `json:"provision_uri"`

	// ProvisionLabel is a human-readable label.
	ProvisionLabel string `json:"provision_label,omitempty"`

	// CurrentVersionURI is the current active version.
	CurrentVersionURI string `json:"current_version_uri,omitempty"`

	// Versions lists all version events in chronological order.
	Versions []VersionEvent `json:"versions"`

	// Timeline provides a simplified timeline view.
	Timeline []TimelineEntry `json:"timeline,omitempty"`

	// TotalVersions is the count of versions.
	TotalVersions int `json:"total_versions"`

	// AmendmentCount is the number of amendments.
	AmendmentCount int `json:"amendment_count"`

	// LastModified is the most recent event date.
	LastModified *time.Time `json:"last_modified,omitempty"`
}

// AmendmentQuery represents a query for amendments.
type AmendmentQuery struct {
	// ChapterURI limits to amendments affecting a chapter.
	ChapterURI string

	// ProposerURI limits to amendments by a specific proposer.
	ProposerURI string

	// FromDate limits to amendments after this date.
	FromDate *time.Time

	// ToDate limits to amendments before this date.
	ToDate *time.Time

	// EventTypes limits to specific event types.
	EventTypes []EventType

	// Limit is the maximum results to return.
	Limit int
}

// AmendmentResult represents an amendment query result.
type AmendmentResult struct {
	// ProvisionURI is the provision that was amended.
	ProvisionURI string `json:"provision_uri"`

	// ProvisionLabel is a human-readable label.
	ProvisionLabel string `json:"provision_label,omitempty"`

	// ProposerURI is who proposed the amendment.
	ProposerURI string `json:"proposer_uri"`

	// ProposerName is the human-readable name.
	ProposerName string `json:"proposer_name,omitempty"`

	// MeetingURI is where the amendment was proposed.
	MeetingURI string `json:"meeting_uri"`

	// MeetingName is a human-readable name for the meeting.
	MeetingName string `json:"meeting_name,omitempty"`

	// Date is when the amendment was proposed.
	Date time.Time `json:"date"`

	// EventType is what happened (amended, adopted, rejected).
	EventType EventType `json:"event_type"`

	// Text is the amendment text.
	Text string `json:"text,omitempty"`
}

// EvolutionTracker tracks provision evolution through deliberations.
type EvolutionTracker struct {
	// store is the triple store containing deliberation data.
	store *store.TripleStore

	// baseURI is the base URI for generating version URIs.
	baseURI string
}

// NewEvolutionTracker creates a new evolution tracker.
func NewEvolutionTracker(tripleStore *store.TripleStore, baseURI string) *EvolutionTracker {
	return &EvolutionTracker{
		store:   tripleStore,
		baseURI: baseURI,
	}
}

// GetEvolution retrieves the complete evolution history of a provision.
func (t *EvolutionTracker) GetEvolution(provisionURI string) (*ProvisionEvolution, error) {
	if t.store == nil {
		return nil, fmt.Errorf("triple store is nil")
	}
	if provisionURI == "" {
		return nil, fmt.Errorf("provision URI is required")
	}

	evolution := &ProvisionEvolution{
		ProvisionURI: provisionURI,
		Versions:     []VersionEvent{},
		Timeline:     []TimelineEntry{},
	}

	// Get provision label
	labelTriples := t.store.Find(provisionURI, store.RDFSLabel, "")
	if len(labelTriples) > 0 {
		evolution.ProvisionLabel = labelTriples[0].Object
	} else {
		// Try PropTitle
		titleTriples := t.store.Find(provisionURI, store.PropTitle, "")
		if len(titleTriples) > 0 {
			evolution.ProvisionLabel = titleTriples[0].Object
		}
	}

	// Get current version
	currentTriples := t.store.Find(provisionURI, store.PropCurrentVersion, "")
	if len(currentTriples) > 0 {
		evolution.CurrentVersionURI = currentTriples[0].Object
	}

	// Find all versions of this provision
	versionTriples := t.store.Find("", store.PropVersionOf, provisionURI)
	for _, vt := range versionTriples {
		event := t.buildVersionEvent(vt.Subject)
		evolution.Versions = append(evolution.Versions, event)
	}

	// Also check for proposed/amended/adopted triples directly on the provision
	t.addDirectEvents(provisionURI, evolution)

	// Sort versions by date
	sort.Slice(evolution.Versions, func(i, j int) bool {
		return evolution.Versions[i].Date.Before(evolution.Versions[j].Date)
	})

	// Deduplicate versions by URI
	evolution.Versions = t.deduplicateVersions(evolution.Versions)

	// Build timeline from versions
	for _, v := range evolution.Versions {
		entry := TimelineEntry{
			Date:       v.Date,
			Label:      v.Version,
			EventType:  v.EventType,
			VersionURI: v.VersionURI,
			MeetingURI: v.MeetingURI,
			ProposedBy: v.ProposedBy,
		}
		entry.Description = t.buildEventDescription(v)
		evolution.Timeline = append(evolution.Timeline, entry)
	}

	// Calculate statistics
	evolution.TotalVersions = len(evolution.Versions)
	for _, v := range evolution.Versions {
		if v.EventType == EventAmended {
			evolution.AmendmentCount++
		}
	}
	if len(evolution.Versions) > 0 {
		lastDate := evolution.Versions[len(evolution.Versions)-1].Date
		evolution.LastModified = &lastDate
	}

	return evolution, nil
}

// buildVersionEvent constructs a VersionEvent from a version URI.
func (t *EvolutionTracker) buildVersionEvent(versionURI string) VersionEvent {
	event := VersionEvent{
		VersionURI: versionURI,
	}

	// Get version number
	versionTriples := t.store.Find(versionURI, store.PropVersionNumber, "")
	if len(versionTriples) > 0 {
		event.Version = versionTriples[0].Object
	}

	// Get event type
	typeTriples := t.store.Find(versionURI, "reg:eventType", "")
	if len(typeTriples) > 0 {
		event.EventType = ParseEventType(typeTriples[0].Object)
	}

	// Get date from various properties
	event.Date = t.getEventDate(versionURI)

	// Get meeting
	meetingTriples := t.store.Find(versionURI, store.PropDecidedAt, "")
	if len(meetingTriples) == 0 {
		meetingTriples = t.store.Find(versionURI, store.PropDiscussedAt, "")
	}
	if len(meetingTriples) == 0 {
		meetingTriples = t.store.Find(versionURI, "reg:proposedAt", "")
	}
	if len(meetingTriples) == 0 {
		meetingTriples = t.store.Find(versionURI, "reg:amendedAt", "")
	}
	if len(meetingTriples) == 0 {
		meetingTriples = t.store.Find(versionURI, "reg:adoptedAt", "")
	}
	if len(meetingTriples) > 0 {
		event.MeetingURI = meetingTriples[0].Object
		event.MeetingName = t.getLabel(meetingTriples[0].Object)
	}

	// Get proposer
	proposerTriples := t.store.Find(versionURI, store.PropProposedBy, "")
	if len(proposerTriples) > 0 {
		event.ProposedBy = proposerTriples[0].Object
		event.ProposerName = t.getLabel(proposerTriples[0].Object)
	}

	// Get text
	textTriples := t.store.Find(versionURI, store.PropText, "")
	if len(textTriples) > 0 {
		event.Text = textTriples[0].Object
	}

	// Get previous version
	prevTriples := t.store.Find(versionURI, store.PropPreviousVersion, "")
	if len(prevTriples) > 0 {
		event.PreviousVersionURI = prevTriples[0].Object
	} else {
		// Check amends relationship
		amendsTriples := t.store.Find(versionURI, store.PropAmends, "")
		if len(amendsTriples) > 0 {
			event.PreviousVersionURI = amendsTriples[0].Object
		}
	}

	// Get vote record if exists
	voteTriples := t.store.Find(versionURI, "reg:hasVote", "")
	if len(voteTriples) > 0 {
		event.Vote = t.getVoteRecord(voteTriples[0].Object)
	}

	return event
}

// addDirectEvents adds events from direct triples on the provision.
func (t *EvolutionTracker) addDirectEvents(provisionURI string, evolution *ProvisionEvolution) {
	// Check for proposedAt
	proposedTriples := t.store.Find(provisionURI, "reg:proposedAt", "")
	for _, pt := range proposedTriples {
		event := VersionEvent{
			VersionURI: provisionURI + ":proposed",
			Version:    "v0",
			EventType:  EventProposed,
			MeetingURI: pt.Object,
		}
		event.Date = t.getMeetingDate(pt.Object)
		event.MeetingName = t.getLabel(pt.Object)

		// Get proposer
		proposerTriples := t.store.Find(provisionURI, store.PropProposedBy, "")
		if len(proposerTriples) > 0 {
			event.ProposedBy = proposerTriples[0].Object
			event.ProposerName = t.getLabel(proposerTriples[0].Object)
		}

		// Get text
		textTriples := t.store.Find(provisionURI, store.PropText, "")
		if len(textTriples) > 0 {
			event.Text = textTriples[0].Object
		}

		evolution.Versions = append(evolution.Versions, event)
	}

	// Check for adoptedAt
	adoptedTriples := t.store.Find(provisionURI, "reg:adoptedAt", "")
	for _, at := range adoptedTriples {
		event := VersionEvent{
			VersionURI: provisionURI + ":adopted",
			Version:    "final",
			EventType:  EventAdopted,
			MeetingURI: at.Object,
		}
		event.Date = t.getMeetingDate(at.Object)
		event.MeetingName = t.getLabel(at.Object)

		// Get vote record
		voteTriples := t.store.Find(provisionURI, "reg:hasVote", "")
		if len(voteTriples) > 0 {
			event.Vote = t.getVoteRecord(voteTriples[0].Object)
		}

		evolution.Versions = append(evolution.Versions, event)
	}
}

// getEventDate extracts the date for an event from various properties.
func (t *EvolutionTracker) getEventDate(uri string) time.Time {
	// Try specific event date
	dateTriples := t.store.Find(uri, "reg:eventDate", "")
	if len(dateTriples) > 0 {
		if d, err := time.Parse(time.RFC3339, dateTriples[0].Object); err == nil {
			return d
		}
		if d, err := time.Parse("2006-01-02", dateTriples[0].Object); err == nil {
			return d
		}
	}

	// Try adopted date
	dateTriples = t.store.Find(uri, store.PropAdoptedDate, "")
	if len(dateTriples) > 0 {
		if d, err := time.Parse(time.RFC3339, dateTriples[0].Object); err == nil {
			return d
		}
		if d, err := time.Parse("2006-01-02", dateTriples[0].Object); err == nil {
			return d
		}
	}

	// Try valid from
	dateTriples = t.store.Find(uri, store.PropValidFrom, "")
	if len(dateTriples) > 0 {
		if d, err := time.Parse(time.RFC3339, dateTriples[0].Object); err == nil {
			return d
		}
		if d, err := time.Parse("2006-01-02", dateTriples[0].Object); err == nil {
			return d
		}
	}

	// Try general date
	dateTriples = t.store.Find(uri, store.PropDate, "")
	if len(dateTriples) > 0 {
		if d, err := time.Parse(time.RFC3339, dateTriples[0].Object); err == nil {
			return d
		}
		if d, err := time.Parse("2006-01-02", dateTriples[0].Object); err == nil {
			return d
		}
	}

	return time.Time{}
}

// getMeetingDate gets the date of a meeting.
func (t *EvolutionTracker) getMeetingDate(meetingURI string) time.Time {
	dateTriples := t.store.Find(meetingURI, store.PropMeetingDate, "")
	if len(dateTriples) > 0 {
		if d, err := time.Parse(time.RFC3339, dateTriples[0].Object); err == nil {
			return d
		}
		if d, err := time.Parse("2006-01-02", dateTriples[0].Object); err == nil {
			return d
		}
	}
	return t.getEventDate(meetingURI)
}

// getLabel gets a human-readable label for a URI.
func (t *EvolutionTracker) getLabel(uri string) string {
	labelTriples := t.store.Find(uri, store.RDFSLabel, "")
	if len(labelTriples) > 0 {
		return labelTriples[0].Object
	}
	titleTriples := t.store.Find(uri, store.PropTitle, "")
	if len(titleTriples) > 0 {
		return titleTriples[0].Object
	}
	// Extract last segment of URI as fallback
	if idx := strings.LastIndex(uri, "#"); idx >= 0 {
		return uri[idx+1:]
	}
	if idx := strings.LastIndex(uri, "/"); idx >= 0 {
		return uri[idx+1:]
	}
	return uri
}

// getVoteRecord retrieves a vote record from the store.
func (t *EvolutionTracker) getVoteRecord(voteURI string) *VoteRecord {
	vote := &VoteRecord{URI: voteURI}

	// Get vote date
	dateTriples := t.store.Find(voteURI, store.PropVoteDate, "")
	if len(dateTriples) > 0 {
		if d, err := time.Parse(time.RFC3339, dateTriples[0].Object); err == nil {
			vote.VoteDate = d
		} else if d, err := time.Parse("2006-01-02", dateTriples[0].Object); err == nil {
			vote.VoteDate = d
		}
	}

	// Get vote counts
	forTriples := t.store.Find(voteURI, store.PropVoteFor, "")
	if len(forTriples) > 0 {
		vote.ForCount = parseInt(forTriples[0].Object)
	}

	againstTriples := t.store.Find(voteURI, store.PropVoteAgainst, "")
	if len(againstTriples) > 0 {
		vote.AgainstCount = parseInt(againstTriples[0].Object)
	}

	abstainTriples := t.store.Find(voteURI, store.PropVoteAbstain, "")
	if len(abstainTriples) > 0 {
		vote.AbstainCount = parseInt(abstainTriples[0].Object)
	}

	// Get result
	resultTriples := t.store.Find(voteURI, store.PropVoteResult, "")
	if len(resultTriples) > 0 {
		vote.Result = resultTriples[0].Object
	}

	return vote
}

// parseInt parses an integer from a string.
func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

// buildEventDescription creates a human-readable description of an event.
func (t *EvolutionTracker) buildEventDescription(event VersionEvent) string {
	var desc string
	switch event.EventType {
	case EventProposed:
		desc = "Proposed"
		if event.ProposerName != "" {
			desc += " by " + event.ProposerName
		}
	case EventAmended:
		desc = "Amended"
		if event.ProposerName != "" {
			desc += " by " + event.ProposerName
		}
	case EventAdopted:
		desc = "Adopted"
		if event.Vote != nil {
			desc += fmt.Sprintf(" (%d for, %d against, %d abstain)",
				event.Vote.ForCount, event.Vote.AgainstCount, event.Vote.AbstainCount)
		}
	case EventRejected:
		desc = "Rejected"
		if event.Vote != nil {
			desc += fmt.Sprintf(" (%d for, %d against, %d abstain)",
				event.Vote.ForCount, event.Vote.AgainstCount, event.Vote.AbstainCount)
		}
	case EventWithdrawn:
		desc = "Withdrawn"
		if event.ProposerName != "" {
			desc += " by " + event.ProposerName
		}
	case EventSuperseded:
		desc = "Superseded by newer version"
	case EventRepealed:
		desc = "Repealed"
	default:
		desc = event.EventType.String()
	}

	if event.MeetingName != "" {
		desc += " at " + event.MeetingName
	}

	return desc
}

// deduplicateVersions removes duplicate versions by URI.
func (t *EvolutionTracker) deduplicateVersions(versions []VersionEvent) []VersionEvent {
	seen := make(map[string]bool)
	result := make([]VersionEvent, 0, len(versions))
	for _, v := range versions {
		if !seen[v.VersionURI] {
			seen[v.VersionURI] = true
			result = append(result, v)
		}
	}
	return result
}

// RecordProposal records a new provision proposal.
func (t *EvolutionTracker) RecordProposal(provisionURI, meetingURI, proposerURI, text string) error {
	if t.store == nil {
		return fmt.Errorf("triple store is nil")
	}
	if provisionURI == "" {
		return fmt.Errorf("provision URI is required")
	}
	if meetingURI == "" {
		return fmt.Errorf("meeting URI is required")
	}

	versionURI := provisionURI + ":v1"

	// Create version triples
	t.store.Add(versionURI, store.RDFType, "reg:ProposedText")
	t.store.Add(versionURI, store.PropVersionOf, provisionURI)
	t.store.Add(versionURI, store.PropVersionNumber, "v1")
	t.store.Add(versionURI, "reg:eventType", "proposed")
	t.store.Add(versionURI, "reg:proposedAt", meetingURI)

	if proposerURI != "" {
		t.store.Add(versionURI, store.PropProposedBy, proposerURI)
	}

	if text != "" {
		t.store.Add(versionURI, store.PropText, text)
	}

	// Record current date
	now := time.Now().Format(time.RFC3339)
	t.store.Add(versionURI, "reg:eventDate", now)

	// Set as current version
	t.store.Add(provisionURI, store.PropCurrentVersion, versionURI)

	return nil
}

// RecordAmendment records an amendment to a provision.
func (t *EvolutionTracker) RecordAmendment(provisionURI, meetingURI, proposerURI, text string) error {
	if t.store == nil {
		return fmt.Errorf("triple store is nil")
	}
	if provisionURI == "" {
		return fmt.Errorf("provision URI is required")
	}
	if meetingURI == "" {
		return fmt.Errorf("meeting URI is required")
	}

	// Get current version to link to
	currentTriples := t.store.Find(provisionURI, store.PropCurrentVersion, "")
	var previousVersionURI string
	if len(currentTriples) > 0 {
		previousVersionURI = currentTriples[0].Object
	}

	// Determine next version number
	nextVersion := t.getNextVersionNumber(provisionURI)
	versionURI := fmt.Sprintf("%s:v%d", provisionURI, nextVersion)

	// Create version triples
	t.store.Add(versionURI, store.RDFType, "reg:AmendedText")
	t.store.Add(versionURI, store.PropVersionOf, provisionURI)
	t.store.Add(versionURI, store.PropVersionNumber, fmt.Sprintf("v%d", nextVersion))
	t.store.Add(versionURI, "reg:eventType", "amended")
	t.store.Add(versionURI, "reg:amendedAt", meetingURI)

	if previousVersionURI != "" {
		t.store.Add(versionURI, store.PropPreviousVersion, previousVersionURI)
		t.store.Add(versionURI, store.PropAmends, previousVersionURI)
		t.store.Add(previousVersionURI, store.PropNextVersion, versionURI)
	}

	if proposerURI != "" {
		t.store.Add(versionURI, store.PropProposedBy, proposerURI)
	}

	if text != "" {
		t.store.Add(versionURI, store.PropText, text)
	}

	// Record current date
	now := time.Now().Format(time.RFC3339)
	t.store.Add(versionURI, "reg:eventDate", now)

	// Update current version
	if len(currentTriples) > 0 {
		t.store.Delete(provisionURI, store.PropCurrentVersion, currentTriples[0].Object)
	}
	t.store.Add(provisionURI, store.PropCurrentVersion, versionURI)

	return nil
}

// RecordAdoption records the adoption of a provision with vote record.
func (t *EvolutionTracker) RecordAdoption(provisionURI, meetingURI string, vote *VoteRecord) error {
	if t.store == nil {
		return fmt.Errorf("triple store is nil")
	}
	if provisionURI == "" {
		return fmt.Errorf("provision URI is required")
	}
	if meetingURI == "" {
		return fmt.Errorf("meeting URI is required")
	}

	// Get current version
	currentTriples := t.store.Find(provisionURI, store.PropCurrentVersion, "")
	var versionURI string
	if len(currentTriples) > 0 {
		versionURI = currentTriples[0].Object
	} else {
		// Create final version
		versionURI = provisionURI + ":final"
		t.store.Add(versionURI, store.PropVersionOf, provisionURI)
		t.store.Add(versionURI, store.PropVersionNumber, "final")
	}

	// Update version type
	t.store.Add(versionURI, store.RDFType, "reg:AdoptedText")
	t.store.Add(versionURI, "reg:eventType", "adopted")
	t.store.Add(versionURI, "reg:adoptedAt", meetingURI)

	// Record current date
	now := time.Now().Format(time.RFC3339)
	t.store.Add(versionURI, store.PropAdoptedDate, now)
	t.store.Add(versionURI, "reg:eventDate", now)

	// Add vote record if provided
	if vote != nil {
		voteURI := versionURI + ":vote"
		t.store.Add(versionURI, "reg:hasVote", voteURI)
		t.store.Add(voteURI, store.RDFType, store.ClassVoteRecord)
		t.store.Add(voteURI, store.PropVoteFor, fmt.Sprintf("%d", vote.ForCount))
		t.store.Add(voteURI, store.PropVoteAgainst, fmt.Sprintf("%d", vote.AgainstCount))
		t.store.Add(voteURI, store.PropVoteAbstain, fmt.Sprintf("%d", vote.AbstainCount))
		if vote.Result != "" {
			t.store.Add(voteURI, store.PropVoteResult, vote.Result)
		}
	}

	// Set as current version
	t.store.Add(provisionURI, store.PropCurrentVersion, versionURI)

	return nil
}

// RecordRejection records the rejection of a provision or amendment.
func (t *EvolutionTracker) RecordRejection(provisionURI, meetingURI string, vote *VoteRecord) error {
	if t.store == nil {
		return fmt.Errorf("triple store is nil")
	}
	if provisionURI == "" {
		return fmt.Errorf("provision URI is required")
	}
	if meetingURI == "" {
		return fmt.Errorf("meeting URI is required")
	}

	// Get current version
	currentTriples := t.store.Find(provisionURI, store.PropCurrentVersion, "")
	var versionURI string
	if len(currentTriples) > 0 {
		versionURI = currentTriples[0].Object
	} else {
		versionURI = provisionURI + ":rejected"
		t.store.Add(versionURI, store.PropVersionOf, provisionURI)
	}

	// Update version with rejection
	t.store.Add(versionURI, "reg:eventType", "rejected")
	t.store.Add(versionURI, "reg:rejectedAt", meetingURI)

	// Record current date
	now := time.Now().Format(time.RFC3339)
	t.store.Add(versionURI, "reg:eventDate", now)

	// Add vote record if provided
	if vote != nil {
		voteURI := versionURI + ":vote"
		t.store.Add(versionURI, "reg:hasVote", voteURI)
		t.store.Add(voteURI, store.RDFType, store.ClassVoteRecord)
		t.store.Add(voteURI, store.PropVoteFor, fmt.Sprintf("%d", vote.ForCount))
		t.store.Add(voteURI, store.PropVoteAgainst, fmt.Sprintf("%d", vote.AgainstCount))
		t.store.Add(voteURI, store.PropVoteAbstain, fmt.Sprintf("%d", vote.AbstainCount))
		if vote.Result != "" {
			t.store.Add(voteURI, store.PropVoteResult, vote.Result)
		}
	}

	return nil
}

// getNextVersionNumber determines the next version number for a provision.
func (t *EvolutionTracker) getNextVersionNumber(provisionURI string) int {
	versionTriples := t.store.Find("", store.PropVersionOf, provisionURI)
	maxVersion := 0

	for _, vt := range versionTriples {
		numTriples := t.store.Find(vt.Subject, store.PropVersionNumber, "")
		if len(numTriples) > 0 {
			v := numTriples[0].Object
			// Parse "v1", "v2", etc.
			if len(v) > 1 && v[0] == 'v' {
				n := parseInt(v[1:])
				if n > maxVersion {
					maxVersion = n
				}
			}
		}
	}

	return maxVersion + 1
}

// CompareVersions compares two versions and returns their differences.
func (t *EvolutionTracker) CompareVersions(version1URI, version2URI string) (*TextDiff, error) {
	if t.store == nil {
		return nil, fmt.Errorf("triple store is nil")
	}
	if version1URI == "" || version2URI == "" {
		return nil, fmt.Errorf("both version URIs are required")
	}

	diff := &TextDiff{
		Version1URI: version1URI,
		Version2URI: version2URI,
		Changes:     []TextChange{},
	}

	// Get text for version 1
	text1Triples := t.store.Find(version1URI, store.PropText, "")
	if len(text1Triples) > 0 {
		diff.Text1 = text1Triples[0].Object
	}

	// Get text for version 2
	text2Triples := t.store.Find(version2URI, store.PropText, "")
	if len(text2Triples) > 0 {
		diff.Text2 = text2Triples[0].Object
	}

	// Simple word-level diff
	if diff.Text1 != diff.Text2 {
		diff.Changes = t.computeSimpleDiff(diff.Text1, diff.Text2)
	}

	return diff, nil
}

// computeSimpleDiff performs a simple word-level diff between two texts.
func (t *EvolutionTracker) computeSimpleDiff(text1, text2 string) []TextChange {
	var changes []TextChange

	words1 := strings.Fields(text1)
	words2 := strings.Fields(text2)

	// Simple LCS-based approach for demonstration
	// In production, use a proper diff library
	if len(words1) == 0 && len(words2) > 0 {
		changes = append(changes, TextChange{
			Type:    "addition",
			NewText: text2,
		})
	} else if len(words2) == 0 && len(words1) > 0 {
		changes = append(changes, TextChange{
			Type:    "deletion",
			OldText: text1,
		})
	} else if text1 != text2 {
		changes = append(changes, TextChange{
			Type:    "modification",
			OldText: text1,
			NewText: text2,
		})
	}

	return changes
}

// QueryAmendments queries for amendments matching the given criteria.
func (t *EvolutionTracker) QueryAmendments(query *AmendmentQuery) ([]AmendmentResult, error) {
	if t.store == nil {
		return nil, fmt.Errorf("triple store is nil")
	}

	var results []AmendmentResult

	// Find all amended versions
	amendedTriples := t.store.Find("", store.RDFType, "reg:AmendedText")
	for _, at := range amendedTriples {
		versionURI := at.Subject

		// Get the provision this version belongs to
		provisionTriples := t.store.Find(versionURI, store.PropVersionOf, "")
		if len(provisionTriples) == 0 {
			continue
		}
		provisionURI := provisionTriples[0].Object

		// Filter by chapter if specified
		if query != nil && query.ChapterURI != "" {
			partOfTriples := t.store.Find(provisionURI, store.PropPartOf, "")
			matchesChapter := false
			for _, pot := range partOfTriples {
				if pot.Object == query.ChapterURI {
					matchesChapter = true
					break
				}
				// Check parent's parent (for articles in sections in chapters)
				parentParts := t.store.Find(pot.Object, store.PropPartOf, "")
				for _, pp := range parentParts {
					if pp.Object == query.ChapterURI {
						matchesChapter = true
						break
					}
				}
			}
			if !matchesChapter {
				continue
			}
		}

		// Get proposer
		proposerTriples := t.store.Find(versionURI, store.PropProposedBy, "")
		var proposerURI, proposerName string
		if len(proposerTriples) > 0 {
			proposerURI = proposerTriples[0].Object
			proposerName = t.getLabel(proposerURI)
		}

		// Filter by proposer if specified
		if query != nil && query.ProposerURI != "" && proposerURI != query.ProposerURI {
			continue
		}

		// Get meeting
		meetingTriples := t.store.Find(versionURI, "reg:amendedAt", "")
		var meetingURI, meetingName string
		if len(meetingTriples) > 0 {
			meetingURI = meetingTriples[0].Object
			meetingName = t.getLabel(meetingURI)
		}

		// Get date
		eventDate := t.getEventDate(versionURI)

		// Filter by date range if specified
		if query != nil && query.FromDate != nil && eventDate.Before(*query.FromDate) {
			continue
		}
		if query != nil && query.ToDate != nil && eventDate.After(*query.ToDate) {
			continue
		}

		// Get text
		textTriples := t.store.Find(versionURI, store.PropText, "")
		var text string
		if len(textTriples) > 0 {
			text = textTriples[0].Object
		}

		result := AmendmentResult{
			ProvisionURI:   provisionURI,
			ProvisionLabel: t.getLabel(provisionURI),
			ProposerURI:    proposerURI,
			ProposerName:   proposerName,
			MeetingURI:     meetingURI,
			MeetingName:    meetingName,
			Date:           eventDate,
			EventType:      EventAmended,
			Text:           text,
		}

		results = append(results, result)
	}

	// Sort by date
	sort.Slice(results, func(i, j int) bool {
		return results[i].Date.Before(results[j].Date)
	})

	// Apply limit
	if query != nil && query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}

	return results, nil
}

// GetChapterAmendments returns all amendments to provisions in a chapter.
func (t *EvolutionTracker) GetChapterAmendments(chapterURI string) ([]AmendmentResult, error) {
	query := &AmendmentQuery{
		ChapterURI: chapterURI,
	}
	return t.QueryAmendments(query)
}

// GetProposerAmendments returns all amendments proposed by a stakeholder.
func (t *EvolutionTracker) GetProposerAmendments(proposerURI string) ([]AmendmentResult, error) {
	query := &AmendmentQuery{
		ProposerURI: proposerURI,
	}
	return t.QueryAmendments(query)
}

// GenerateTimelineData generates visualization data for a provision timeline.
func (t *EvolutionTracker) GenerateTimelineData(provisionURI string) ([]TimelineEntry, error) {
	evolution, err := t.GetEvolution(provisionURI)
	if err != nil {
		return nil, err
	}
	return evolution.Timeline, nil
}

// GetCurrentText returns the current text of a provision.
func (t *EvolutionTracker) GetCurrentText(provisionURI string) (string, error) {
	if t.store == nil {
		return "", fmt.Errorf("triple store is nil")
	}

	// Get current version
	currentTriples := t.store.Find(provisionURI, store.PropCurrentVersion, "")
	if len(currentTriples) > 0 {
		textTriples := t.store.Find(currentTriples[0].Object, store.PropText, "")
		if len(textTriples) > 0 {
			return textTriples[0].Object, nil
		}
	}

	// Fall back to direct text on provision
	textTriples := t.store.Find(provisionURI, store.PropText, "")
	if len(textTriples) > 0 {
		return textTriples[0].Object, nil
	}

	return "", nil
}

// GetVersionText returns the text of a specific version.
func (t *EvolutionTracker) GetVersionText(versionURI string) (string, error) {
	if t.store == nil {
		return "", fmt.Errorf("triple store is nil")
	}

	textTriples := t.store.Find(versionURI, store.PropText, "")
	if len(textTriples) > 0 {
		return textTriples[0].Object, nil
	}

	return "", nil
}
