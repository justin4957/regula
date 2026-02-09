// Package deliberation provides types and functions for modeling deliberation
// documents including meetings, agendas, decisions, and their evolution over time.
// This package forms the foundation of the living knowledge graph that tracks
// multi-process work across meetings and documents.
package deliberation

import (
	"fmt"
	"time"
)

// MeetingStatus indicates the current state of a meeting in the deliberation process.
type MeetingStatus int

const (
	// MeetingScheduled indicates the meeting is planned but not yet held.
	MeetingScheduled MeetingStatus = iota
	// MeetingInProgress indicates the meeting is currently underway.
	MeetingInProgress
	// MeetingCompleted indicates the meeting has concluded.
	MeetingCompleted
	// MeetingCancelled indicates the meeting was cancelled.
	MeetingCancelled
	// MeetingPostponed indicates the meeting was postponed to a later date.
	MeetingPostponed
)

// String returns a human-readable label for the meeting status.
func (s MeetingStatus) String() string {
	switch s {
	case MeetingScheduled:
		return "scheduled"
	case MeetingInProgress:
		return "in_progress"
	case MeetingCompleted:
		return "completed"
	case MeetingCancelled:
		return "cancelled"
	case MeetingPostponed:
		return "postponed"
	default:
		return "unknown"
	}
}

// Meeting represents a deliberation meeting where decisions are discussed,
// debated, and potentially adopted. Meetings are the primary temporal anchor
// for tracking the evolution of provisions and decisions.
type Meeting struct {
	// URI is the unique identifier for this meeting in the knowledge graph.
	URI string `json:"uri"`

	// Identifier is the human-readable meeting identifier (e.g., "WG-2024-05").
	Identifier string `json:"identifier"`

	// Title is the descriptive title of the meeting.
	Title string `json:"title"`

	// Series identifies the meeting series (e.g., "Working Group A", "Plenary").
	Series string `json:"series,omitempty"`

	// Sequence is the meeting number within its series (e.g., 43 for "43rd session").
	Sequence int `json:"sequence,omitempty"`

	// Date is the date of the meeting.
	Date time.Time `json:"date"`

	// StartTime is the scheduled start time (optional).
	StartTime *time.Time `json:"start_time,omitempty"`

	// EndTime is the scheduled or actual end time (optional).
	EndTime *time.Time `json:"end_time,omitempty"`

	// Location is where the meeting is held (physical or virtual).
	Location string `json:"location,omitempty"`

	// Status indicates the current state of the meeting.
	Status MeetingStatus `json:"status"`

	// Chair is the URI of the presiding officer.
	Chair string `json:"chair,omitempty"`

	// ChairName is the human-readable name of the chair.
	ChairName string `json:"chair_name,omitempty"`

	// Secretary is the URI of the meeting secretary/rapporteur.
	Secretary string `json:"secretary,omitempty"`

	// Participants lists URIs of attendees.
	Participants []string `json:"participants,omitempty"`

	// AgendaItems contains the items discussed in this meeting.
	AgendaItems []AgendaItem `json:"agenda_items,omitempty"`

	// PreviousMeeting is the URI of the preceding meeting in the series.
	PreviousMeeting string `json:"previous_meeting,omitempty"`

	// NextMeeting is the URI of the following meeting in the series.
	NextMeeting string `json:"next_meeting,omitempty"`

	// SourceDocument is the URI of the meeting minutes document.
	SourceDocument string `json:"source_document,omitempty"`

	// ProcessURI links to the parent deliberation process.
	ProcessURI string `json:"process_uri,omitempty"`
}

// AgendaItemOutcome indicates what happened with an agenda item.
type AgendaItemOutcome int

const (
	// OutcomePending indicates the item has not yet been addressed.
	OutcomePending AgendaItemOutcome = iota
	// OutcomeDiscussed indicates the item was discussed but no decision made.
	OutcomeDiscussed
	// OutcomeDeferred indicates the item was deferred to a future meeting.
	OutcomeDeferred
	// OutcomeDecided indicates a decision was reached.
	OutcomeDecided
	// OutcomeWithdrawn indicates the item was withdrawn from the agenda.
	OutcomeWithdrawn
	// OutcomeNoQuorum indicates the item could not proceed due to quorum issues.
	OutcomeNoQuorum
)

// String returns a human-readable label for the agenda item outcome.
func (o AgendaItemOutcome) String() string {
	switch o {
	case OutcomePending:
		return "pending"
	case OutcomeDiscussed:
		return "discussed"
	case OutcomeDeferred:
		return "deferred"
	case OutcomeDecided:
		return "decided"
	case OutcomeWithdrawn:
		return "withdrawn"
	case OutcomeNoQuorum:
		return "no_quorum"
	default:
		return "unknown"
	}
}

// AgendaItem represents a single item on a meeting agenda. Each item may
// involve discussion of one or more provisions, motions, or decisions.
type AgendaItem struct {
	// URI is the unique identifier for this agenda item.
	URI string `json:"uri"`

	// Number is the item number on the agenda (e.g., "3", "4.1", "A").
	Number string `json:"number"`

	// Title is the agenda item title/description.
	Title string `json:"title"`

	// Description provides additional context about the item.
	Description string `json:"description,omitempty"`

	// MeetingURI links back to the parent meeting.
	MeetingURI string `json:"meeting_uri"`

	// Outcome indicates what happened with this item.
	Outcome AgendaItemOutcome `json:"outcome"`

	// DocumentsConsidered lists URIs of documents discussed under this item.
	DocumentsConsidered []string `json:"documents_considered,omitempty"`

	// ProvisionsDiscussed lists URIs of provisions addressed.
	ProvisionsDiscussed []string `json:"provisions_discussed,omitempty"`

	// Motions lists motions/amendments proposed under this item.
	Motions []Motion `json:"motions,omitempty"`

	// Decisions lists decisions made under this item.
	Decisions []Decision `json:"decisions,omitempty"`

	// Interventions lists speaker interventions on this item.
	Interventions []Intervention `json:"interventions,omitempty"`

	// ActionItems lists actions assigned during discussion of this item.
	ActionItems []ActionItem `json:"action_items,omitempty"`

	// DeferredTo is the URI of the meeting to which this item was deferred.
	DeferredTo string `json:"deferred_to,omitempty"`

	// Notes contains additional notes about the item discussion.
	Notes string `json:"notes,omitempty"`
}

// MotionStatus indicates the current state of a motion or amendment.
type MotionStatus int

const (
	// MotionProposed indicates the motion has been formally proposed.
	MotionProposed MotionStatus = iota
	// MotionSeconded indicates the motion has received a second.
	MotionSeconded
	// MotionDebated indicates the motion is under debate.
	MotionDebated
	// MotionVoted indicates a vote has been taken.
	MotionVoted
	// MotionAdopted indicates the motion was adopted/passed.
	MotionAdopted
	// MotionRejected indicates the motion was rejected/failed.
	MotionRejected
	// MotionWithdrawn indicates the proposer withdrew the motion.
	MotionWithdrawn
	// MotionTabled indicates the motion was tabled for later consideration.
	MotionTabled
)

// String returns a human-readable label for the motion status.
func (s MotionStatus) String() string {
	switch s {
	case MotionProposed:
		return "proposed"
	case MotionSeconded:
		return "seconded"
	case MotionDebated:
		return "debated"
	case MotionVoted:
		return "voted"
	case MotionAdopted:
		return "adopted"
	case MotionRejected:
		return "rejected"
	case MotionWithdrawn:
		return "withdrawn"
	case MotionTabled:
		return "tabled"
	default:
		return "unknown"
	}
}

// Motion represents a formal motion or amendment proposed during deliberations.
// Motions track the full lifecycle from proposal through voting to outcome.
type Motion struct {
	// URI is the unique identifier for this motion.
	URI string `json:"uri"`

	// Identifier is the motion number/reference (e.g., "Amendment 1", "Motion A").
	Identifier string `json:"identifier"`

	// Type classifies the motion (e.g., "amendment", "procedural", "substantive").
	Type string `json:"type"`

	// Title is a brief description of the motion.
	Title string `json:"title"`

	// Text is the full text of the proposed motion or amendment.
	Text string `json:"text"`

	// ProposedText is the text proposed to replace existing text (for amendments).
	ProposedText string `json:"proposed_text,omitempty"`

	// ExistingText is the current text being amended (for amendments).
	ExistingText string `json:"existing_text,omitempty"`

	// ProposerURI is the URI of the stakeholder who proposed the motion.
	ProposerURI string `json:"proposer_uri"`

	// ProposerName is the human-readable name of the proposer.
	ProposerName string `json:"proposer_name,omitempty"`

	// SeconderURI is the URI of the stakeholder who seconded the motion.
	SeconderURI string `json:"seconder_uri,omitempty"`

	// SeconderName is the human-readable name of the seconder.
	SeconderName string `json:"seconder_name,omitempty"`

	// Status indicates the current state of the motion.
	Status MotionStatus `json:"status"`

	// AgendaItemURI links to the agenda item under which this motion was made.
	AgendaItemURI string `json:"agenda_item_uri"`

	// MeetingURI links to the meeting where this motion was made.
	MeetingURI string `json:"meeting_uri"`

	// TargetProvisionURI is the URI of the provision being amended (if applicable).
	TargetProvisionURI string `json:"target_provision_uri,omitempty"`

	// Vote contains the vote record if a vote was taken.
	Vote *VoteRecord `json:"vote,omitempty"`

	// SupportersURIs lists stakeholders who spoke in favor.
	SupportersURIs []string `json:"supporters_uris,omitempty"`

	// OpponentsURIs lists stakeholders who spoke against.
	OpponentsURIs []string `json:"opponents_uris,omitempty"`

	// WithdrawalReason explains why the motion was withdrawn (if applicable).
	WithdrawalReason string `json:"withdrawal_reason,omitempty"`

	// ProposedAt is when the motion was formally proposed.
	ProposedAt *time.Time `json:"proposed_at,omitempty"`

	// DecidedAt is when the motion was decided.
	DecidedAt *time.Time `json:"decided_at,omitempty"`
}

// VoteRecord captures the results of a vote on a motion or decision.
type VoteRecord struct {
	// URI is the unique identifier for this vote record.
	URI string `json:"uri"`

	// VoteDate is when the vote was taken.
	VoteDate time.Time `json:"vote_date"`

	// VoteType classifies the vote (e.g., "roll_call", "voice", "show_of_hands").
	VoteType string `json:"vote_type"`

	// Question is the question put to the vote.
	Question string `json:"question"`

	// Result is the outcome (e.g., "adopted", "rejected", "tie").
	Result string `json:"result"`

	// MajorityRequired indicates the threshold (e.g., "simple", "two_thirds").
	MajorityRequired string `json:"majority_required,omitempty"`

	// ForCount is the number of votes in favor.
	ForCount int `json:"for_count"`

	// AgainstCount is the number of votes against.
	AgainstCount int `json:"against_count"`

	// AbstainCount is the number of abstentions.
	AbstainCount int `json:"abstain_count"`

	// AbsentCount is the number of absent/not voting.
	AbsentCount int `json:"absent_count,omitempty"`

	// IndividualVotes contains per-stakeholder votes (for roll calls).
	IndividualVotes []IndividualVote `json:"individual_votes,omitempty"`

	// MotionURI links to the motion being voted on.
	MotionURI string `json:"motion_uri,omitempty"`

	// MeetingURI links to the meeting where the vote occurred.
	MeetingURI string `json:"meeting_uri"`
}

// VotePosition indicates how a stakeholder voted.
type VotePosition int

const (
	// VoteFor indicates a vote in favor.
	VoteFor VotePosition = iota
	// VoteAgainst indicates a vote against.
	VoteAgainst
	// VoteAbstain indicates an abstention.
	VoteAbstain
	// VoteAbsent indicates the stakeholder was absent/not voting.
	VoteAbsent
)

// String returns a human-readable label for the vote position.
func (p VotePosition) String() string {
	switch p {
	case VoteFor:
		return "for"
	case VoteAgainst:
		return "against"
	case VoteAbstain:
		return "abstain"
	case VoteAbsent:
		return "absent"
	default:
		return "unknown"
	}
}

// IndividualVote records how a specific stakeholder voted.
type IndividualVote struct {
	// VoterURI is the URI of the stakeholder who voted.
	VoterURI string `json:"voter_uri"`

	// VoterName is the human-readable name of the voter.
	VoterName string `json:"voter_name"`

	// Position indicates how they voted.
	Position VotePosition `json:"position"`

	// Explanation is an optional explanation of vote.
	Explanation string `json:"explanation,omitempty"`
}

// Decision represents a formal decision reached during deliberations.
// Decisions are the outcomes that affect provisions in the knowledge graph.
type Decision struct {
	// URI is the unique identifier for this decision.
	URI string `json:"uri"`

	// Identifier is the decision reference (e.g., "Decision 2024-05-01").
	Identifier string `json:"identifier"`

	// Title is a brief description of the decision.
	Title string `json:"title"`

	// Description is the full text of what was decided.
	Description string `json:"description"`

	// Type classifies the decision (e.g., "adoption", "amendment", "rejection", "deferral").
	Type string `json:"type"`

	// MeetingURI links to the meeting where this decision was made.
	MeetingURI string `json:"meeting_uri"`

	// AgendaItemURI links to the agenda item under which this decision was made.
	AgendaItemURI string `json:"agenda_item_uri,omitempty"`

	// MotionURI links to the motion that led to this decision (if applicable).
	MotionURI string `json:"motion_uri,omitempty"`

	// VoteURI links to the vote record (if applicable).
	VoteURI string `json:"vote_uri,omitempty"`

	// AffectedProvisionURIs lists provisions affected by this decision.
	AffectedProvisionURIs []string `json:"affected_provision_uris,omitempty"`

	// EffectiveDate is when the decision takes effect.
	EffectiveDate *time.Time `json:"effective_date,omitempty"`

	// DecidedAt is when the decision was formally made.
	DecidedAt time.Time `json:"decided_at"`

	// SupersedesURI links to a previous decision this one supersedes.
	SupersedesURI string `json:"supersedes_uri,omitempty"`

	// SupersededByURI links to a later decision that supersedes this one.
	SupersededByURI string `json:"superseded_by_uri,omitempty"`
}

// InterventionPosition indicates the speaker's stance on an issue.
type InterventionPosition int

const (
	// PositionNeutral indicates no clear position taken.
	PositionNeutral InterventionPosition = iota
	// PositionSupport indicates support for the matter under discussion.
	PositionSupport
	// PositionOppose indicates opposition to the matter under discussion.
	PositionOppose
	// PositionQualified indicates conditional or qualified support/opposition.
	PositionQualified
	// PositionQuestion indicates a question or request for clarification.
	PositionQuestion
	// PositionProcedural indicates a procedural intervention.
	PositionProcedural
)

// String returns a human-readable label for the intervention position.
func (p InterventionPosition) String() string {
	switch p {
	case PositionNeutral:
		return "neutral"
	case PositionSupport:
		return "support"
	case PositionOppose:
		return "oppose"
	case PositionQualified:
		return "qualified"
	case PositionQuestion:
		return "question"
	case PositionProcedural:
		return "procedural"
	default:
		return "unknown"
	}
}

// Intervention represents a speaker's contribution during deliberations.
// Tracking interventions enables analysis of stakeholder positions and participation.
type Intervention struct {
	// URI is the unique identifier for this intervention.
	URI string `json:"uri"`

	// SpeakerURI is the URI of the person who spoke.
	SpeakerURI string `json:"speaker_uri"`

	// SpeakerName is the human-readable name of the speaker.
	SpeakerName string `json:"speaker_name"`

	// AffiliationURI is the URI of the speaker's organization/delegation.
	AffiliationURI string `json:"affiliation_uri,omitempty"`

	// AffiliationName is the human-readable name of the affiliation.
	AffiliationName string `json:"affiliation_name,omitempty"`

	// MeetingURI links to the meeting where this intervention occurred.
	MeetingURI string `json:"meeting_uri"`

	// AgendaItemURI links to the agenda item being discussed.
	AgendaItemURI string `json:"agenda_item_uri,omitempty"`

	// MotionURI links to the motion being discussed (if applicable).
	MotionURI string `json:"motion_uri,omitempty"`

	// Position indicates the speaker's stance.
	Position InterventionPosition `json:"position"`

	// Summary is a brief summary of what was said.
	Summary string `json:"summary,omitempty"`

	// FullText is the full text of the intervention (if available).
	FullText string `json:"full_text,omitempty"`

	// Timestamp is when the intervention occurred.
	Timestamp *time.Time `json:"timestamp,omitempty"`

	// Sequence is the order of this intervention within the agenda item.
	Sequence int `json:"sequence,omitempty"`
}

// ActionItemStatus indicates the current state of an action item.
type ActionItemStatus int

const (
	// ActionPending indicates the action has not yet been started.
	ActionPending ActionItemStatus = iota
	// ActionInProgress indicates work on the action is underway.
	ActionInProgress
	// ActionCompleted indicates the action has been completed.
	ActionCompleted
	// ActionDeferred indicates the action has been deferred.
	ActionDeferred
	// ActionCancelled indicates the action has been cancelled.
	ActionCancelled
)

// String returns a human-readable label for the action item status.
func (s ActionItemStatus) String() string {
	switch s {
	case ActionPending:
		return "pending"
	case ActionInProgress:
		return "in_progress"
	case ActionCompleted:
		return "completed"
	case ActionDeferred:
		return "deferred"
	case ActionCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// ActionItem represents a task assigned during a meeting that needs to be
// completed and tracked across subsequent meetings.
type ActionItem struct {
	// URI is the unique identifier for this action item.
	URI string `json:"uri"`

	// Identifier is the action item reference (e.g., "Action 2024-05-01").
	Identifier string `json:"identifier"`

	// Description is what needs to be done.
	Description string `json:"description"`

	// AssignedToURIs lists URIs of stakeholders responsible for the action.
	AssignedToURIs []string `json:"assigned_to_uris"`

	// AssignedToNames lists human-readable names of assignees.
	AssignedToNames []string `json:"assigned_to_names,omitempty"`

	// AssignedAtMeetingURI is the meeting where this action was assigned.
	AssignedAtMeetingURI string `json:"assigned_at_meeting_uri"`

	// AgendaItemURI links to the agenda item that generated this action.
	AgendaItemURI string `json:"agenda_item_uri,omitempty"`

	// DueDate is when the action should be completed.
	DueDate *time.Time `json:"due_date,omitempty"`

	// Status indicates the current state of the action.
	Status ActionItemStatus `json:"status"`

	// CompletedAtMeetingURI is the meeting where this action was marked complete.
	CompletedAtMeetingURI string `json:"completed_at_meeting_uri,omitempty"`

	// CompletedAt is when the action was actually completed.
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// RelatedProvisionURIs lists provisions this action relates to.
	RelatedProvisionURIs []string `json:"related_provision_uris,omitempty"`

	// Notes contains follow-up notes from subsequent meetings.
	Notes []ActionNote `json:"notes,omitempty"`

	// Priority indicates urgency (e.g., "high", "medium", "low").
	Priority string `json:"priority,omitempty"`
}

// ActionNote records a follow-up note on an action item from a subsequent meeting.
type ActionNote struct {
	// MeetingURI is where this note was recorded.
	MeetingURI string `json:"meeting_uri"`

	// Date is when the note was recorded.
	Date time.Time `json:"date"`

	// Note is the content of the follow-up.
	Note string `json:"note"`

	// UpdatedStatus is the new status if it changed.
	UpdatedStatus *ActionItemStatus `json:"updated_status,omitempty"`
}

// Stakeholder represents a participant in the deliberation process.
// Stakeholders can be individuals, organizations, member states, or delegations.
type Stakeholder struct {
	// URI is the unique identifier for this stakeholder.
	URI string `json:"uri"`

	// Name is the human-readable name.
	Name string `json:"name"`

	// Type classifies the stakeholder (e.g., "individual", "delegation", "organization").
	Type string `json:"type"`

	// Aliases lists alternative names or references used in documents.
	Aliases []string `json:"aliases,omitempty"`

	// ParentOrganizationURI is the URI of the parent organization (for individuals).
	ParentOrganizationURI string `json:"parent_organization_uri,omitempty"`

	// MemberURIs lists member URIs (for organizations/groups).
	MemberURIs []string `json:"member_uris,omitempty"`

	// Roles lists roles held by this stakeholder.
	Roles []StakeholderRole `json:"roles,omitempty"`
}

// StakeholderRole represents a role held by a stakeholder in a deliberation process.
type StakeholderRole struct {
	// Role is the role title (e.g., "Chair", "Rapporteur", "Secretary").
	Role string `json:"role"`

	// Scope is where the role applies (e.g., "Working Group A", "Plenary").
	Scope string `json:"scope,omitempty"`

	// ProcessURI links to the deliberation process.
	ProcessURI string `json:"process_uri,omitempty"`

	// StartDate is when the role began.
	StartDate *time.Time `json:"start_date,omitempty"`

	// EndDate is when the role ended.
	EndDate *time.Time `json:"end_date,omitempty"`
}

// DeliberationProcess represents an ongoing deliberation spanning multiple meetings.
// It groups related meetings, decisions, and provisions being developed.
type DeliberationProcess struct {
	// URI is the unique identifier for this process.
	URI string `json:"uri"`

	// Identifier is the process reference (e.g., "GDPR-Review-2024").
	Identifier string `json:"identifier"`

	// Title is the descriptive title of the process.
	Title string `json:"title"`

	// Description provides context about the process.
	Description string `json:"description,omitempty"`

	// Type classifies the process (e.g., "legislation", "treaty", "policy").
	Type string `json:"type,omitempty"`

	// Status indicates the current state (e.g., "active", "concluded", "suspended").
	Status string `json:"status"`

	// StartDate is when the process began.
	StartDate *time.Time `json:"start_date,omitempty"`

	// EndDate is when the process concluded (if applicable).
	EndDate *time.Time `json:"end_date,omitempty"`

	// MeetingURIs lists all meetings in this process.
	MeetingURIs []string `json:"meeting_uris,omitempty"`

	// ProvisionURIs lists provisions being developed or amended.
	ProvisionURIs []string `json:"provision_uris,omitempty"`

	// ParticipantURIs lists stakeholders involved in the process.
	ParticipantURIs []string `json:"participant_uris,omitempty"`
}

// NewMeeting creates a new Meeting with required fields.
func NewMeeting(uri, identifier, title string, date time.Time) *Meeting {
	return &Meeting{
		URI:        uri,
		Identifier: identifier,
		Title:      title,
		Date:       date,
		Status:     MeetingScheduled,
	}
}

// NewAgendaItem creates a new AgendaItem with required fields.
func NewAgendaItem(uri, number, title, meetingURI string) *AgendaItem {
	return &AgendaItem{
		URI:        uri,
		Number:     number,
		Title:      title,
		MeetingURI: meetingURI,
		Outcome:    OutcomePending,
	}
}

// NewMotion creates a new Motion with required fields.
func NewMotion(uri, identifier, title, text, proposerURI, meetingURI string) *Motion {
	return &Motion{
		URI:         uri,
		Identifier:  identifier,
		Title:       title,
		Text:        text,
		ProposerURI: proposerURI,
		MeetingURI:  meetingURI,
		Status:      MotionProposed,
	}
}

// NewDecision creates a new Decision with required fields.
func NewDecision(uri, identifier, title, description, meetingURI string, decidedAt time.Time) *Decision {
	return &Decision{
		URI:         uri,
		Identifier:  identifier,
		Title:       title,
		Description: description,
		MeetingURI:  meetingURI,
		DecidedAt:   decidedAt,
	}
}

// NewActionItem creates a new ActionItem with required fields.
func NewActionItem(uri, identifier, description string, assignedToURIs []string, meetingURI string) *ActionItem {
	return &ActionItem{
		URI:                  uri,
		Identifier:           identifier,
		Description:          description,
		AssignedToURIs:       assignedToURIs,
		AssignedAtMeetingURI: meetingURI,
		Status:               ActionPending,
	}
}

// String returns a human-readable representation of the meeting.
func (m *Meeting) String() string {
	return fmt.Sprintf("%s (%s) - %s", m.Title, m.Identifier, m.Date.Format("2006-01-02"))
}

// IsOverdue returns true if the action item is past its due date and not completed.
func (a *ActionItem) IsOverdue() bool {
	if a.Status == ActionCompleted || a.Status == ActionCancelled {
		return false
	}
	if a.DueDate == nil {
		return false
	}
	return time.Now().After(*a.DueDate)
}

// DaysSinceDue returns the number of days since the due date (negative if not yet due).
func (a *ActionItem) DaysSinceDue() int {
	if a.DueDate == nil {
		return 0
	}
	duration := time.Since(*a.DueDate)
	return int(duration.Hours() / 24)
}
