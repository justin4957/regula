// Package deliberation provides types and functions for modeling deliberation
// documents including meetings, agendas, decisions, and their evolution over time.
package deliberation

import (
	"encoding/json"
	"fmt"
	"html/template"
	"sort"
	"strings"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// ReportType identifies the type of process report.
type ReportType int

const (
	// ReportTypeProgress shows what happened since last report period.
	ReportTypeProgress ReportType = iota
	// ReportTypeStatus shows current state of all topics.
	ReportTypeStatus
	// ReportTypeDecisionLog shows all decisions chronologically.
	ReportTypeDecisionLog
	// ReportTypeEvolution traces how a provision changed over time.
	ReportTypeEvolution
	// ReportTypeParticipation shows who contributed on what topics.
	ReportTypeParticipation
)

// String returns a human-readable label for the report type.
func (t ReportType) String() string {
	switch t {
	case ReportTypeProgress:
		return "progress"
	case ReportTypeStatus:
		return "status"
	case ReportTypeDecisionLog:
		return "decision_log"
	case ReportTypeEvolution:
		return "evolution"
	case ReportTypeParticipation:
		return "participation"
	default:
		return "unknown"
	}
}

// TopicPhase indicates the current phase of a topic in deliberations.
type TopicPhase int

const (
	// TopicProposed indicates the topic has been proposed.
	TopicProposed TopicPhase = iota
	// TopicUnderDiscussion indicates active discussion.
	TopicUnderDiscussion
	// TopicAgreed indicates the topic has been agreed.
	TopicAgreed
	// TopicBlocked indicates the topic is blocked.
	TopicBlocked
	// TopicClosed indicates the topic is closed.
	TopicClosed
)

// String returns a human-readable label for the topic phase.
func (p TopicPhase) String() string {
	switch p {
	case TopicProposed:
		return "proposed"
	case TopicUnderDiscussion:
		return "under_discussion"
	case TopicAgreed:
		return "agreed"
	case TopicBlocked:
		return "blocked"
	case TopicClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// ReportPeriod defines a time range for report generation.
type ReportPeriod struct {
	// Start is the beginning of the period.
	Start time.Time `json:"start"`

	// End is the end of the period.
	End time.Time `json:"end"`
}

// ReportScope defines what to include in the report.
type ReportScope struct {
	// ProvisionURIs limits report to specific provisions.
	ProvisionURIs []string `json:"provision_uris,omitempty"`

	// MeetingURIs limits report to specific meetings.
	MeetingURIs []string `json:"meeting_uris,omitempty"`

	// IncludeBottlenecks adds bottleneck analysis.
	IncludeBottlenecks bool `json:"include_bottlenecks"`

	// IncludeActions adds action item summary.
	IncludeActions bool `json:"include_actions"`
}

// ReportConfig configures report generation.
type ReportConfig struct {
	// Type is the report type to generate.
	Type ReportType `json:"type"`

	// Title is the report title.
	Title string `json:"title"`

	// Period defines the time range (for progress reports).
	Period *ReportPeriod `json:"period,omitempty"`

	// Scope defines what to include.
	Scope ReportScope `json:"scope"`

	// ProvisionURI for evolution reports.
	ProvisionURI string `json:"provision_uri,omitempty"`
}

// ProcessReport contains a generated deliberation report.
type ProcessReport struct {
	// Type is the report type.
	Type ReportType `json:"type"`

	// Title is the report title.
	Title string `json:"title"`

	// GeneratedAt is when the report was generated.
	GeneratedAt time.Time `json:"generated_at"`

	// Period is the reporting period (for progress reports).
	Period *ReportPeriod `json:"period,omitempty"`

	// Scope defines what was included.
	Scope ReportScope `json:"scope"`

	// ExecutiveSummary provides a high-level overview.
	ExecutiveSummary string `json:"executive_summary"`

	// KeyDecisions lists important decisions.
	KeyDecisions []DecisionSummary `json:"key_decisions,omitempty"`

	// TopicStatus lists the status of each topic.
	TopicStatus []TopicStatus `json:"topic_status,omitempty"`

	// ActionSummary summarizes action items.
	ActionSummary *ReportActionSummary `json:"action_summary,omitempty"`

	// ParticipationStats contains participation statistics.
	ParticipationStats *ParticipationStats `json:"participation_stats,omitempty"`

	// EvolutionHistory traces provision changes.
	EvolutionHistory []EvolutionEntry `json:"evolution_history,omitempty"`

	// NextSteps lists recommended next actions.
	NextSteps []string `json:"next_steps,omitempty"`

	// Bottlenecks lists detected process issues.
	Bottlenecks []Bottleneck `json:"bottlenecks,omitempty"`
}

// DecisionSummary provides a compact view of a decision.
type DecisionSummary struct {
	// Date is when the decision was made.
	Date time.Time `json:"date"`

	// Topic is the subject of the decision.
	Topic string `json:"topic"`

	// TopicURI is the URI of the affected provision.
	TopicURI string `json:"topic_uri,omitempty"`

	// Decision is a brief description.
	Decision string `json:"decision"`

	// Vote is the vote tally (e.g., "18-3-2").
	Vote string `json:"vote,omitempty"`

	// MeetingURI is where the decision was made.
	MeetingURI string `json:"meeting_uri"`
}

// TopicStatus shows the current state of a topic.
type TopicStatus struct {
	// TopicURI is the unique identifier.
	TopicURI string `json:"topic_uri"`

	// TopicLabel is the human-readable name.
	TopicLabel string `json:"topic_label"`

	// Status indicates the current phase.
	Status TopicPhase `json:"status"`

	// StatusEmoji provides a visual indicator.
	StatusEmoji string `json:"status_emoji"`

	// LastMeeting is the URI of the last meeting where discussed.
	LastMeeting string `json:"last_meeting"`

	// LastMeetingDate is when it was last discussed.
	LastMeetingDate time.Time `json:"last_meeting_date"`

	// KeyPoints are bullet points of current position.
	KeyPoints []string `json:"key_points,omitempty"`

	// OpenIssues are unresolved questions.
	OpenIssues []string `json:"open_issues,omitempty"`

	// BlockedBy lists what's blocking this topic.
	BlockedBy []string `json:"blocked_by,omitempty"`
}

// ReportActionSummary provides statistics on action items for reports.
type ReportActionSummary struct {
	// Total is the total number of actions.
	Total int `json:"total"`

	// Completed is the number completed.
	Completed int `json:"completed"`

	// Pending is the number still pending.
	Pending int `json:"pending"`

	// Overdue is the number past due date.
	Overdue int `json:"overdue"`

	// OverdueItems lists the overdue actions.
	OverdueItems []ActionItemBrief `json:"overdue_items,omitempty"`
}

// ActionItemBrief is a compact action item view.
type ActionItemBrief struct {
	// Description is what needs to be done.
	Description string `json:"description"`

	// Assignee is who is responsible.
	Assignee string `json:"assignee"`

	// DueDate is when it was due.
	DueDate time.Time `json:"due_date"`

	// DaysOverdue is how many days past due.
	DaysOverdue int `json:"days_overdue"`
}

// ParticipationStats contains participation statistics.
type ParticipationStats struct {
	// TotalMeetings is the number of meetings analyzed.
	TotalMeetings int `json:"total_meetings"`

	// TotalSpeakers is the number of unique speakers.
	TotalSpeakers int `json:"total_speakers"`

	// TotalVotes is the number of votes taken.
	TotalVotes int `json:"total_votes"`

	// TopContributors lists most active participants.
	TopContributors []ContributorStat `json:"top_contributors"`

	// VotingRecord summarizes voting by stakeholder.
	VotingRecord map[string]VotingSummary `json:"voting_record,omitempty"`
}

// ContributorStat tracks a contributor's activity.
type ContributorStat struct {
	// StakeholderURI is the unique identifier.
	StakeholderURI string `json:"stakeholder_uri"`

	// StakeholderName is the display name.
	StakeholderName string `json:"stakeholder_name"`

	// Interventions is the number of times they spoke.
	Interventions int `json:"interventions"`

	// MotionsProposed is the number of motions proposed.
	MotionsProposed int `json:"motions_proposed"`

	// VotesCast is the number of votes cast.
	VotesCast int `json:"votes_cast"`

	// TopTopics lists their most discussed topics.
	TopTopics []string `json:"top_topics,omitempty"`
}

// VotingSummary summarizes a stakeholder's voting.
type VotingSummary struct {
	// TotalVotes is the number of votes cast.
	TotalVotes int `json:"total_votes"`

	// VotesFor is the number of yes votes.
	VotesFor int `json:"votes_for"`

	// VotesAgainst is the number of no votes.
	VotesAgainst int `json:"votes_against"`

	// Abstentions is the number of abstentions.
	Abstentions int `json:"abstentions"`
}

// EvolutionEntry tracks a change in a provision.
type EvolutionEntry struct {
	// Date is when the change occurred.
	Date time.Time `json:"date"`

	// MeetingURI is where the change occurred.
	MeetingURI string `json:"meeting_uri"`

	// MeetingLabel is the meeting name.
	MeetingLabel string `json:"meeting_label"`

	// EventType describes what happened.
	EventType string `json:"event_type"`

	// Description explains the change.
	Description string `json:"description"`

	// PreviousText is the text before change.
	PreviousText string `json:"previous_text,omitempty"`

	// NewText is the text after change.
	NewText string `json:"new_text,omitempty"`

	// ProposedBy is who proposed the change.
	ProposedBy string `json:"proposed_by,omitempty"`
}

// ReportGenerator generates process reports.
type ReportGenerator struct {
	// store is the triple store containing deliberation data.
	store *store.TripleStore

	// baseURI for constructing URIs.
	baseURI string

	// bottleneckDetector for bottleneck analysis.
	bottleneckDetector *BottleneckDetector
}

// NewReportGenerator creates a new report generator.
func NewReportGenerator(tripleStore *store.TripleStore, baseURI string) *ReportGenerator {
	return &ReportGenerator{
		store:              tripleStore,
		baseURI:            baseURI,
		bottleneckDetector: NewBottleneckDetector(tripleStore, baseURI),
	}
}

// GenerateReport creates a report based on the configuration.
func (g *ReportGenerator) GenerateReport(config ReportConfig) (*ProcessReport, error) {
	if g.store == nil {
		return nil, fmt.Errorf("no triple store configured")
	}

	report := &ProcessReport{
		Type:        config.Type,
		Title:       config.Title,
		GeneratedAt: time.Now(),
		Period:      config.Period,
		Scope:       config.Scope,
	}

	switch config.Type {
	case ReportTypeProgress:
		if err := g.generateProgressReport(report, config); err != nil {
			return nil, err
		}
	case ReportTypeStatus:
		if err := g.generateStatusReport(report, config); err != nil {
			return nil, err
		}
	case ReportTypeDecisionLog:
		if err := g.generateDecisionLog(report, config); err != nil {
			return nil, err
		}
	case ReportTypeEvolution:
		if err := g.generateEvolutionReport(report, config); err != nil {
			return nil, err
		}
	case ReportTypeParticipation:
		if err := g.generateParticipationReport(report, config); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown report type: %d", config.Type)
	}

	// Add bottlenecks if requested
	if config.Scope.IncludeBottlenecks {
		bottleneckReport, err := g.bottleneckDetector.DetectBottlenecks()
		if err == nil {
			report.Bottlenecks = bottleneckReport.Bottlenecks
		}
	}

	// Generate executive summary
	report.ExecutiveSummary = g.generateExecutiveSummary(report)

	return report, nil
}

// generateProgressReport creates a progress report for a time period.
func (g *ReportGenerator) generateProgressReport(report *ProcessReport, config ReportConfig) error {
	if config.Period == nil {
		return fmt.Errorf("progress report requires a period")
	}

	// Get decisions in period
	report.KeyDecisions = g.findDecisionsInPeriod(config.Period)

	// Get topic status
	report.TopicStatus = g.computeTopicStatus(config.Period)

	// Get action summary
	if config.Scope.IncludeActions {
		report.ActionSummary = g.summarizeActions()
	}

	// Generate next steps
	report.NextSteps = g.generateNextSteps(report)

	return nil
}

// generateStatusReport creates a status report of all topics.
func (g *ReportGenerator) generateStatusReport(report *ProcessReport, config ReportConfig) error {
	// Get all topic statuses
	report.TopicStatus = g.computeTopicStatus(nil)

	// Get action summary
	if config.Scope.IncludeActions {
		report.ActionSummary = g.summarizeActions()
	}

	// Generate next steps
	report.NextSteps = g.generateNextSteps(report)

	return nil
}

// generateDecisionLog creates a chronological decision log.
func (g *ReportGenerator) generateDecisionLog(report *ProcessReport, config ReportConfig) error {
	report.KeyDecisions = g.collectAllDecisions(config.Period)
	return nil
}

// generateEvolutionReport traces a provision's changes over time.
func (g *ReportGenerator) generateEvolutionReport(report *ProcessReport, config ReportConfig) error {
	if config.ProvisionURI == "" {
		return fmt.Errorf("evolution report requires a provision URI")
	}

	report.EvolutionHistory = g.traceProvisionEvolution(config.ProvisionURI)

	// Get current topic status
	status := g.getTopicStatus(config.ProvisionURI)
	if status != nil {
		report.TopicStatus = []TopicStatus{*status}
	}

	return nil
}

// generateParticipationReport creates participation statistics.
func (g *ReportGenerator) generateParticipationReport(report *ProcessReport, config ReportConfig) error {
	report.ParticipationStats = g.computeParticipationStats(config.Period)
	return nil
}

// findDecisionsInPeriod finds decisions made during a time period.
func (g *ReportGenerator) findDecisionsInPeriod(period *ReportPeriod) []DecisionSummary {
	var decisions []DecisionSummary

	// Find all decisions
	decisionTriples := g.store.Find("", store.RDFType, store.ClassDeliberationDecision)
	for _, triple := range decisionTriples {
		decisionURI := triple.Subject

		// Get decision date
		var decisionDate time.Time
		if props := g.store.Find(decisionURI, store.PropMeetingDate, ""); len(props) > 0 {
			if t, err := time.Parse(time.RFC3339, props[0].Object); err == nil {
				decisionDate = t
			}
		}

		// Check if in period
		if period != nil {
			if decisionDate.Before(period.Start) || decisionDate.After(period.End) {
				continue
			}
		}

		summary := DecisionSummary{
			Date: decisionDate,
		}

		// Get topic/label
		if props := g.store.Find(decisionURI, store.RDFSLabel, ""); len(props) > 0 {
			summary.Topic = props[0].Object
		}

		// Get decision description
		if props := g.store.Find(decisionURI, store.PropDecisionType, ""); len(props) > 0 {
			summary.Decision = props[0].Object
		}

		// Get meeting
		if props := g.store.Find(decisionURI, store.PropPartOf, ""); len(props) > 0 {
			summary.MeetingURI = props[0].Object
		}

		// Get affected provision
		if props := g.store.Find(decisionURI, store.PropAffectsProvision, ""); len(props) > 0 {
			summary.TopicURI = props[0].Object
		}

		// Get vote if available
		if props := g.store.Find(decisionURI, "reg:voteURI", ""); len(props) > 0 {
			voteURI := props[0].Object
			summary.Vote = g.formatVoteTally(voteURI)
		}

		decisions = append(decisions, summary)
	}

	// Sort by date
	sort.Slice(decisions, func(i, j int) bool {
		return decisions[i].Date.Before(decisions[j].Date)
	})

	return decisions
}

// collectAllDecisions collects all decisions.
func (g *ReportGenerator) collectAllDecisions(period *ReportPeriod) []DecisionSummary {
	return g.findDecisionsInPeriod(period)
}

// computeTopicStatus computes the status of all topics.
func (g *ReportGenerator) computeTopicStatus(period *ReportPeriod) []TopicStatus {
	var statuses []TopicStatus
	topicMap := make(map[string]*TopicStatus)

	// Find all provisions discussed
	agendaTriples := g.store.Find("", store.PropProvisionDiscussed, "")
	for _, triple := range agendaTriples {
		provisionURI := triple.Object
		agendaURI := triple.Subject

		// Get meeting for this agenda item
		var meetingURI string
		var meetingDate time.Time
		meetingTriples := g.store.Find("", store.PropHasAgendaItem, agendaURI)
		for _, mt := range meetingTriples {
			meetingURI = mt.Subject
			if props := g.store.Find(meetingURI, store.PropMeetingDate, ""); len(props) > 0 {
				if t, err := time.Parse(time.RFC3339, props[0].Object); err == nil {
					meetingDate = t
				}
			}
			break
		}

		// Check period filter
		if period != nil {
			if meetingDate.Before(period.Start) || meetingDate.After(period.End) {
				continue
			}
		}

		// Get or create topic status
		status, exists := topicMap[provisionURI]
		if !exists {
			status = &TopicStatus{
				TopicURI: provisionURI,
				Status:   TopicUnderDiscussion,
			}

			// Get label
			if props := g.store.Find(provisionURI, store.RDFSLabel, ""); len(props) > 0 {
				status.TopicLabel = props[0].Object
			} else {
				status.TopicLabel = extractURILabel(provisionURI)
			}

			topicMap[provisionURI] = status
		}

		// Update last meeting if more recent
		if meetingDate.After(status.LastMeetingDate) {
			status.LastMeeting = meetingURI
			status.LastMeetingDate = meetingDate
		}
	}

	// Determine phase for each topic
	for _, status := range topicMap {
		status.Status = g.determineTopicPhase(status.TopicURI)
		status.StatusEmoji = g.getStatusEmoji(status.Status)

		// Check for blockers
		status.BlockedBy = g.findBlockers(status.TopicURI)
		if len(status.BlockedBy) > 0 {
			status.Status = TopicBlocked
			status.StatusEmoji = "🔴"
		}

		statuses = append(statuses, *status)
	}

	// Sort by label
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].TopicLabel < statuses[j].TopicLabel
	})

	return statuses
}

// getTopicStatus gets status for a specific topic.
func (g *ReportGenerator) getTopicStatus(topicURI string) *TopicStatus {
	status := &TopicStatus{
		TopicURI: topicURI,
		Status:   TopicUnderDiscussion,
	}

	// Get label
	if props := g.store.Find(topicURI, store.RDFSLabel, ""); len(props) > 0 {
		status.TopicLabel = props[0].Object
	} else {
		status.TopicLabel = extractURILabel(topicURI)
	}

	// Find last meeting where discussed
	agendaTriples := g.store.Find("", store.PropProvisionDiscussed, topicURI)
	for _, triple := range agendaTriples {
		agendaURI := triple.Subject
		meetingTriples := g.store.Find("", store.PropHasAgendaItem, agendaURI)
		for _, mt := range meetingTriples {
			meetingURI := mt.Subject
			if props := g.store.Find(meetingURI, store.PropMeetingDate, ""); len(props) > 0 {
				if t, err := time.Parse(time.RFC3339, props[0].Object); err == nil {
					if t.After(status.LastMeetingDate) {
						status.LastMeeting = meetingURI
						status.LastMeetingDate = t
					}
				}
			}
		}
	}

	status.Status = g.determineTopicPhase(topicURI)
	status.StatusEmoji = g.getStatusEmoji(status.Status)
	status.BlockedBy = g.findBlockers(topicURI)

	return status
}

// determineTopicPhase determines what phase a topic is in.
func (g *ReportGenerator) determineTopicPhase(topicURI string) TopicPhase {
	// Check if adopted
	if props := g.store.Find(topicURI, store.PropMotionStatus, ""); len(props) > 0 {
		switch props[0].Object {
		case "adopted":
			return TopicAgreed
		case "rejected", "withdrawn":
			return TopicClosed
		case "proposed":
			return TopicProposed
		}
	}

	// Check for decisions
	decisionTriples := g.store.Find("", store.PropAffectsProvision, topicURI)
	for _, dt := range decisionTriples {
		decisionURI := dt.Subject
		if props := g.store.Find(decisionURI, store.PropDecisionType, ""); len(props) > 0 {
			if props[0].Object == "adoption" {
				return TopicAgreed
			}
		}
	}

	return TopicUnderDiscussion
}

// getStatusEmoji returns an emoji for the topic phase.
func (g *ReportGenerator) getStatusEmoji(phase TopicPhase) string {
	switch phase {
	case TopicProposed:
		return "📝"
	case TopicUnderDiscussion:
		return "💬"
	case TopicAgreed:
		return "✅"
	case TopicBlocked:
		return "🔴"
	case TopicClosed:
		return "⬛"
	default:
		return "❓"
	}
}

// findBlockers finds what's blocking a topic.
func (g *ReportGenerator) findBlockers(topicURI string) []string {
	var blockers []string

	// Check for pending references
	refs := g.store.Find(topicURI, store.PropReferences, "")
	for _, ref := range refs {
		targetURI := ref.Object
		if props := g.store.Find(targetURI, store.PropMotionStatus, ""); len(props) > 0 {
			if props[0].Object != "adopted" && props[0].Object != "rejected" {
				blockers = append(blockers, targetURI)
			}
		}
	}

	return blockers
}

// summarizeActions summarizes action items.
func (g *ReportGenerator) summarizeActions() *ReportActionSummary {
	summary := &ReportActionSummary{}
	now := time.Now()

	actionTriples := g.store.Find("", store.RDFType, store.ClassActionItem)
	for _, triple := range actionTriples {
		actionURI := triple.Subject
		summary.Total++

		// Get status
		status := "pending"
		if props := g.store.Find(actionURI, store.PropActionStatus, ""); len(props) > 0 {
			status = props[0].Object
		}

		switch status {
		case "completed":
			summary.Completed++
		case "pending", "in_progress":
			summary.Pending++

			// Check if overdue
			if props := g.store.Find(actionURI, store.PropActionDueDate, ""); len(props) > 0 {
				if dueDate, err := time.Parse(time.RFC3339, props[0].Object); err == nil {
					if now.After(dueDate) {
						summary.Overdue++

						// Get action details
						brief := ActionItemBrief{
							DueDate:     dueDate,
							DaysOverdue: int(now.Sub(dueDate).Hours() / 24),
						}

						if labels := g.store.Find(actionURI, store.RDFSLabel, ""); len(labels) > 0 {
							brief.Description = labels[0].Object
						}
						if assignees := g.store.Find(actionURI, store.PropActionAssignedTo, ""); len(assignees) > 0 {
							brief.Assignee = assignees[0].Object
						}

						summary.OverdueItems = append(summary.OverdueItems, brief)
					}
				}
			}
		}
	}

	return summary
}

// traceProvisionEvolution traces how a provision changed over time.
func (g *ReportGenerator) traceProvisionEvolution(provisionURI string) []EvolutionEntry {
	var entries []EvolutionEntry

	// Find all motions/amendments affecting this provision
	motionTriples := g.store.Find("", store.PropTargetProvision, provisionURI)
	for _, triple := range motionTriples {
		motionURI := triple.Subject

		entry := EvolutionEntry{}

		// Get meeting and date
		if props := g.store.Find(motionURI, store.PropPartOf, ""); len(props) > 0 {
			entry.MeetingURI = props[0].Object
			if labels := g.store.Find(entry.MeetingURI, store.RDFSLabel, ""); len(labels) > 0 {
				entry.MeetingLabel = labels[0].Object
			}
			if dates := g.store.Find(entry.MeetingURI, store.PropMeetingDate, ""); len(dates) > 0 {
				if t, err := time.Parse(time.RFC3339, dates[0].Object); err == nil {
					entry.Date = t
				}
			}
		}

		// Get motion type/status
		if props := g.store.Find(motionURI, store.PropMotionStatus, ""); len(props) > 0 {
			entry.EventType = props[0].Object
		}

		// Get description
		if props := g.store.Find(motionURI, store.RDFSLabel, ""); len(props) > 0 {
			entry.Description = props[0].Object
		}

		// Get proposer
		if props := g.store.Find(motionURI, store.PropProposedBy, ""); len(props) > 0 {
			entry.ProposedBy = props[0].Object
		}

		// Get text changes
		if props := g.store.Find(motionURI, store.PropExistingText, ""); len(props) > 0 {
			entry.PreviousText = props[0].Object
		}
		if props := g.store.Find(motionURI, store.PropProposedText, ""); len(props) > 0 {
			entry.NewText = props[0].Object
		}

		entries = append(entries, entry)
	}

	// Sort by date
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Date.Before(entries[j].Date)
	})

	return entries
}

// computeParticipationStats computes participation statistics.
func (g *ReportGenerator) computeParticipationStats(period *ReportPeriod) *ParticipationStats {
	stats := &ParticipationStats{
		VotingRecord: make(map[string]VotingSummary),
	}

	// Count meetings
	meetingSet := make(map[string]bool)
	speakerSet := make(map[string]bool)
	contributorMap := make(map[string]*ContributorStat)

	// Find all interventions
	interventionTriples := g.store.Find("", store.RDFType, store.ClassIntervention)
	for _, triple := range interventionTriples {
		interventionURI := triple.Subject

		// Get meeting
		var meetingURI string
		if props := g.store.Find(interventionURI, store.PropPartOf, ""); len(props) > 0 {
			meetingURI = props[0].Object
		}

		// Check period
		if period != nil && meetingURI != "" {
			if props := g.store.Find(meetingURI, store.PropMeetingDate, ""); len(props) > 0 {
				if t, err := time.Parse(time.RFC3339, props[0].Object); err == nil {
					if t.Before(period.Start) || t.After(period.End) {
						continue
					}
				}
			}
		}

		meetingSet[meetingURI] = true

		// Get speaker
		var speakerURI, speakerName string
		if props := g.store.Find(interventionURI, store.PropSpeaker, ""); len(props) > 0 {
			speakerURI = props[0].Object
			if labels := g.store.Find(speakerURI, store.RDFSLabel, ""); len(labels) > 0 {
				speakerName = labels[0].Object
			}
		}

		speakerSet[speakerURI] = true

		// Update contributor stats
		contributor, exists := contributorMap[speakerURI]
		if !exists {
			contributor = &ContributorStat{
				StakeholderURI:  speakerURI,
				StakeholderName: speakerName,
			}
			contributorMap[speakerURI] = contributor
		}
		contributor.Interventions++
	}

	// Count motions proposed
	motionTriples := g.store.Find("", store.RDFType, store.ClassMotion)
	for _, triple := range motionTriples {
		motionURI := triple.Subject
		if props := g.store.Find(motionURI, store.PropProposedBy, ""); len(props) > 0 {
			proposerURI := props[0].Object
			if contributor, exists := contributorMap[proposerURI]; exists {
				contributor.MotionsProposed++
			}
		}
	}

	// Count votes
	voteTriples := g.store.Find("", store.RDFType, store.ClassVoteRecord)
	for range voteTriples {
		stats.TotalVotes++
	}

	// Process individual votes for voting record
	individualVoteTriples := g.store.Find("", store.RDFType, store.ClassIndividualVote)
	for _, triple := range individualVoteTriples {
		ivURI := triple.Subject

		var voterName string
		if props := g.store.Find(ivURI, store.RDFSLabel, ""); len(props) > 0 {
			voterName = props[0].Object
		}

		var position string
		if props := g.store.Find(ivURI, store.PropVotePosition, ""); len(props) > 0 {
			position = props[0].Object
		}

		summary := stats.VotingRecord[voterName]
		summary.TotalVotes++
		switch position {
		case "for":
			summary.VotesFor++
		case "against":
			summary.VotesAgainst++
		case "abstain":
			summary.Abstentions++
		}
		stats.VotingRecord[voterName] = summary

		// Update contributor
		for _, contributor := range contributorMap {
			if contributor.StakeholderName == voterName {
				contributor.VotesCast++
				break
			}
		}
	}

	stats.TotalMeetings = len(meetingSet)
	stats.TotalSpeakers = len(speakerSet)

	// Get top contributors
	var contributors []*ContributorStat
	for _, c := range contributorMap {
		contributors = append(contributors, c)
	}
	sort.Slice(contributors, func(i, j int) bool {
		return contributors[i].Interventions > contributors[j].Interventions
	})

	// Take top 10
	for i := 0; i < len(contributors) && i < 10; i++ {
		stats.TopContributors = append(stats.TopContributors, *contributors[i])
	}

	return stats
}

// formatVoteTally formats a vote as "for-against-abstain".
func (g *ReportGenerator) formatVoteTally(voteURI string) string {
	var forCount, againstCount, abstainCount int

	if props := g.store.Find(voteURI, store.PropVoteFor, ""); len(props) > 0 {
		fmt.Sscanf(props[0].Object, "%d", &forCount)
	}
	if props := g.store.Find(voteURI, store.PropVoteAgainst, ""); len(props) > 0 {
		fmt.Sscanf(props[0].Object, "%d", &againstCount)
	}
	if props := g.store.Find(voteURI, store.PropVoteAbstain, ""); len(props) > 0 {
		fmt.Sscanf(props[0].Object, "%d", &abstainCount)
	}

	return fmt.Sprintf("%d-%d-%d", forCount, againstCount, abstainCount)
}

// generateNextSteps generates recommended next steps.
func (g *ReportGenerator) generateNextSteps(report *ProcessReport) []string {
	var steps []string

	// Check for blocked topics
	for _, topic := range report.TopicStatus {
		if topic.Status == TopicBlocked {
			steps = append(steps, fmt.Sprintf("Resolve blockers for %s", topic.TopicLabel))
		}
	}

	// Check for overdue actions
	if report.ActionSummary != nil && report.ActionSummary.Overdue > 0 {
		steps = append(steps, fmt.Sprintf("Address %d overdue action item(s)", report.ActionSummary.Overdue))
	}

	// Check for topics under discussion
	discussionCount := 0
	for _, topic := range report.TopicStatus {
		if topic.Status == TopicUnderDiscussion {
			discussionCount++
		}
	}
	if discussionCount > 0 {
		steps = append(steps, fmt.Sprintf("Continue discussion on %d pending topic(s)", discussionCount))
	}

	return steps
}

// generateExecutiveSummary generates a summary paragraph.
func (g *ReportGenerator) generateExecutiveSummary(report *ProcessReport) string {
	var parts []string

	// Meeting count
	if report.ParticipationStats != nil && report.ParticipationStats.TotalMeetings > 0 {
		parts = append(parts, fmt.Sprintf("Analyzed %d meetings.", report.ParticipationStats.TotalMeetings))
	}

	// Decision count
	if len(report.KeyDecisions) > 0 {
		parts = append(parts, fmt.Sprintf("%d decisions were made.", len(report.KeyDecisions)))
	}

	// Topic status summary
	agreedCount := 0
	blockedCount := 0
	discussionCount := 0
	for _, topic := range report.TopicStatus {
		switch topic.Status {
		case TopicAgreed:
			agreedCount++
		case TopicBlocked:
			blockedCount++
		case TopicUnderDiscussion:
			discussionCount++
		}
	}

	if agreedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d topic(s) agreed.", agreedCount))
	}
	if blockedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d topic(s) blocked.", blockedCount))
	}
	if discussionCount > 0 {
		parts = append(parts, fmt.Sprintf("%d topic(s) under discussion.", discussionCount))
	}

	// Action item status
	if report.ActionSummary != nil {
		if report.ActionSummary.Overdue > 0 {
			parts = append(parts, fmt.Sprintf("%d action item(s) overdue.", report.ActionSummary.Overdue))
		}
	}

	// Bottleneck count
	if len(report.Bottlenecks) > 0 {
		criticalCount := 0
		for _, b := range report.Bottlenecks {
			if b.Severity == BottleneckCritical {
				criticalCount++
			}
		}
		if criticalCount > 0 {
			parts = append(parts, fmt.Sprintf("%d critical bottleneck(s) detected.", criticalCount))
		}
	}

	return strings.Join(parts, " ")
}

// RenderMarkdown renders the report as Markdown.
func (g *ReportGenerator) RenderMarkdown(report *ProcessReport) string {
	var sb strings.Builder

	// Title
	sb.WriteString(fmt.Sprintf("# %s\n", report.Title))
	if report.Period != nil {
		sb.WriteString(fmt.Sprintf("Period: %s - %s | Generated: %s\n\n",
			report.Period.Start.Format("January 2, 2006"),
			report.Period.End.Format("January 2, 2006"),
			report.GeneratedAt.Format("January 2, 2006")))
	} else {
		sb.WriteString(fmt.Sprintf("Generated: %s\n\n", report.GeneratedAt.Format("January 2, 2006")))
	}

	// Executive Summary
	if report.ExecutiveSummary != "" {
		sb.WriteString("## Executive Summary\n\n")
		sb.WriteString(report.ExecutiveSummary + "\n\n")
	}

	// Key Decisions
	if len(report.KeyDecisions) > 0 {
		sb.WriteString("## Key Decisions\n\n")
		sb.WriteString("| Date | Topic | Decision | Vote |\n")
		sb.WriteString("|------|-------|----------|------|\n")
		for _, d := range report.KeyDecisions {
			vote := d.Vote
			if vote == "" {
				vote = "N/A"
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				d.Date.Format("Jan 2"),
				d.Topic,
				d.Decision,
				vote))
		}
		sb.WriteString("\n")
	}

	// Topic Status
	if len(report.TopicStatus) > 0 {
		sb.WriteString("## Topic Status\n\n")
		for _, topic := range report.TopicStatus {
			sb.WriteString(fmt.Sprintf("### %s %s %s\n",
				topic.TopicLabel,
				topic.StatusEmoji,
				strings.ToUpper(topic.Status.String())))

			if !topic.LastMeetingDate.IsZero() {
				sb.WriteString(fmt.Sprintf("Last discussed: %s\n",
					topic.LastMeetingDate.Format("January 2, 2006")))
			}

			if len(topic.KeyPoints) > 0 {
				sb.WriteString("Key points:\n")
				for _, point := range topic.KeyPoints {
					sb.WriteString(fmt.Sprintf("- %s\n", point))
				}
			}

			if len(topic.OpenIssues) > 0 {
				sb.WriteString("Open issues:\n")
				for _, issue := range topic.OpenIssues {
					sb.WriteString(fmt.Sprintf("- %s\n", issue))
				}
			}

			if len(topic.BlockedBy) > 0 {
				sb.WriteString(fmt.Sprintf("Blocked by: %s\n", strings.Join(topic.BlockedBy, ", ")))
			}

			sb.WriteString("\n")
		}
	}

	// Action Items
	if report.ActionSummary != nil {
		sb.WriteString("## Action Items\n\n")
		sb.WriteString("| Total | Completed | Pending | Overdue |\n")
		sb.WriteString("|-------|-----------|---------|--------|\n")
		sb.WriteString(fmt.Sprintf("| %d | %d | %d | %d |\n\n",
			report.ActionSummary.Total,
			report.ActionSummary.Completed,
			report.ActionSummary.Pending,
			report.ActionSummary.Overdue))

		if len(report.ActionSummary.OverdueItems) > 0 {
			sb.WriteString("**Overdue:**\n")
			for _, item := range report.ActionSummary.OverdueItems {
				sb.WriteString(fmt.Sprintf("- %s (due %s, %d days overdue)\n",
					item.Description,
					item.DueDate.Format("Jan 2"),
					item.DaysOverdue))
			}
			sb.WriteString("\n")
		}
	}

	// Evolution History
	if len(report.EvolutionHistory) > 0 {
		sb.WriteString("## Evolution History\n\n")
		for _, entry := range report.EvolutionHistory {
			sb.WriteString(fmt.Sprintf("### %s - %s\n",
				entry.Date.Format("January 2, 2006"),
				entry.EventType))
			sb.WriteString(fmt.Sprintf("%s\n", entry.Description))
			if entry.ProposedBy != "" {
				sb.WriteString(fmt.Sprintf("Proposed by: %s\n", entry.ProposedBy))
			}
			sb.WriteString("\n")
		}
	}

	// Participation Stats
	if report.ParticipationStats != nil {
		sb.WriteString("## Participation Statistics\n\n")
		sb.WriteString(fmt.Sprintf("- Meetings: %d\n", report.ParticipationStats.TotalMeetings))
		sb.WriteString(fmt.Sprintf("- Unique speakers: %d\n", report.ParticipationStats.TotalSpeakers))
		sb.WriteString(fmt.Sprintf("- Votes taken: %d\n\n", report.ParticipationStats.TotalVotes))

		if len(report.ParticipationStats.TopContributors) > 0 {
			sb.WriteString("**Top Contributors:**\n")
			for i, c := range report.ParticipationStats.TopContributors {
				if i >= 5 {
					break
				}
				sb.WriteString(fmt.Sprintf("- %s: %d interventions, %d motions\n",
					c.StakeholderName,
					c.Interventions,
					c.MotionsProposed))
			}
			sb.WriteString("\n")
		}
	}

	// Bottlenecks
	if len(report.Bottlenecks) > 0 {
		sb.WriteString("## Process Issues\n\n")
		for _, b := range report.Bottlenecks {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n",
				strings.ToUpper(b.Severity.String()),
				b.Description))
		}
		sb.WriteString("\n")
	}

	// Next Steps
	if len(report.NextSteps) > 0 {
		sb.WriteString("## Next Steps\n\n")
		for i, step := range report.NextSteps {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
		}
	}

	return sb.String()
}

// RenderHTML renders the report as HTML.
func (g *ReportGenerator) RenderHTML(report *ProcessReport) (string, error) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; max-width: 900px; margin: 0 auto; padding: 20px; }
        h1 { color: #333; border-bottom: 2px solid #007bff; padding-bottom: 10px; }
        h2 { color: #555; margin-top: 30px; }
        h3 { color: #666; }
        table { border-collapse: collapse; width: 100%; margin: 15px 0; }
        th, td { border: 1px solid #ddd; padding: 10px; text-align: left; }
        th { background-color: #f5f5f5; }
        .summary { background-color: #f8f9fa; padding: 15px; border-radius: 5px; margin: 15px 0; }
        .status-agreed { color: #28a745; }
        .status-blocked { color: #dc3545; }
        .status-discussion { color: #ffc107; }
        .bottleneck-critical { color: #dc3545; font-weight: bold; }
        .bottleneck-high { color: #fd7e14; }
        ul { margin: 10px 0; }
    </style>
</head>
<body>
    <h1>{{.Title}}</h1>
    <p>Generated: {{.GeneratedAt.Format "January 2, 2006 15:04"}}</p>

    {{if .ExecutiveSummary}}
    <div class="summary">
        <h2>Executive Summary</h2>
        <p>{{.ExecutiveSummary}}</p>
    </div>
    {{end}}

    {{if .KeyDecisions}}
    <h2>Key Decisions</h2>
    <table>
        <tr><th>Date</th><th>Topic</th><th>Decision</th><th>Vote</th></tr>
        {{range .KeyDecisions}}
        <tr>
            <td>{{.Date.Format "Jan 2"}}</td>
            <td>{{.Topic}}</td>
            <td>{{.Decision}}</td>
            <td>{{if .Vote}}{{.Vote}}{{else}}N/A{{end}}</td>
        </tr>
        {{end}}
    </table>
    {{end}}

    {{if .TopicStatus}}
    <h2>Topic Status</h2>
    {{range .TopicStatus}}
    <h3>{{.StatusEmoji}} {{.TopicLabel}}</h3>
    <p><strong>Status:</strong> {{.Status}}</p>
    {{if not .LastMeetingDate.IsZero}}
    <p><strong>Last discussed:</strong> {{.LastMeetingDate.Format "January 2, 2006"}}</p>
    {{end}}
    {{if .BlockedBy}}
    <p><strong>Blocked by:</strong> {{range .BlockedBy}}{{.}} {{end}}</p>
    {{end}}
    {{end}}
    {{end}}

    {{if .ActionSummary}}
    <h2>Action Items</h2>
    <table>
        <tr><th>Total</th><th>Completed</th><th>Pending</th><th>Overdue</th></tr>
        <tr>
            <td>{{.ActionSummary.Total}}</td>
            <td>{{.ActionSummary.Completed}}</td>
            <td>{{.ActionSummary.Pending}}</td>
            <td>{{.ActionSummary.Overdue}}</td>
        </tr>
    </table>
    {{end}}

    {{if .NextSteps}}
    <h2>Next Steps</h2>
    <ol>
    {{range .NextSteps}}
        <li>{{.}}</li>
    {{end}}
    </ol>
    {{end}}
</body>
</html>`

	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	if err := t.Execute(&sb, report); err != nil {
		return "", err
	}

	return sb.String(), nil
}

// RenderJSON renders the report as JSON.
func (g *ReportGenerator) RenderJSON(report *ProcessReport) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// extractURILabel extracts the last segment from a URI as a label.
func extractURILabel(uri string) string {
	if idx := strings.LastIndex(uri, "#"); idx >= 0 {
		return uri[idx+1:]
	}
	if idx := strings.LastIndex(uri, "/"); idx >= 0 {
		return uri[idx+1:]
	}
	return uri
}
