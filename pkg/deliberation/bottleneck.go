// Package deliberation provides types and functions for modeling deliberation
// documents including meetings, agendas, decisions, and their evolution over time.
package deliberation

import (
	"fmt"
	"sort"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// BottleneckType classifies the type of process bottleneck.
type BottleneckType int

const (
	// BottleneckInactiveTopic indicates a topic not discussed in recent meetings.
	BottleneckInactiveTopic BottleneckType = iota
	// BottleneckRepeatedDeferral indicates an item deferred multiple times.
	BottleneckRepeatedDeferral
	// BottleneckOverdueAction indicates an action item past its due date.
	BottleneckOverdueAction
	// BottleneckBlockedDecision indicates a decision waiting on other decisions.
	BottleneckBlockedDecision
	// BottleneckMissingQuorum indicates votes that couldn't proceed due to quorum.
	BottleneckMissingQuorum
	// BottleneckCircularDependency indicates amendments referencing each other.
	BottleneckCircularDependency
)

// String returns a human-readable label for the bottleneck type.
func (t BottleneckType) String() string {
	switch t {
	case BottleneckInactiveTopic:
		return "inactive_topic"
	case BottleneckRepeatedDeferral:
		return "repeated_deferral"
	case BottleneckOverdueAction:
		return "overdue_action"
	case BottleneckBlockedDecision:
		return "blocked_decision"
	case BottleneckMissingQuorum:
		return "missing_quorum"
	case BottleneckCircularDependency:
		return "circular_dependency"
	default:
		return "unknown"
	}
}

// BottleneckSeverity indicates how critical a bottleneck is.
type BottleneckSeverity int

const (
	// BottleneckLow indicates 1-2 meetings stalled.
	BottleneckLow BottleneckSeverity = iota
	// BottleneckMedium indicates 3-4 meetings stalled.
	BottleneckMedium
	// BottleneckHigh indicates 5+ meetings stalled.
	BottleneckHigh
	// BottleneckCritical indicates blocking critical path.
	BottleneckCritical
)

// String returns a human-readable label for the bottleneck severity.
func (s BottleneckSeverity) String() string {
	switch s {
	case BottleneckLow:
		return "low"
	case BottleneckMedium:
		return "medium"
	case BottleneckHigh:
		return "high"
	case BottleneckCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Bottleneck represents a process stall or blockage in deliberations.
type Bottleneck struct {
	// Type classifies what kind of bottleneck this is.
	Type BottleneckType `json:"type"`

	// Severity indicates how critical the bottleneck is.
	Severity BottleneckSeverity `json:"severity"`

	// Description provides a human-readable explanation.
	Description string `json:"description"`

	// AffectedItems lists URIs of blocked provisions/actions.
	AffectedItems []string `json:"affected_items"`

	// BlockedBy lists URIs of items causing the blockage.
	BlockedBy []string `json:"blocked_by,omitempty"`

	// StalledSince is when the bottleneck was first detected.
	StalledSince time.Time `json:"stalled_since"`

	// MeetingCount is the number of meetings the item has been stalled.
	MeetingCount int `json:"meeting_count"`

	// Suggestions provides potential remediation steps.
	Suggestions []string `json:"suggestions,omitempty"`

	// SourceMeeting is the URI of the meeting where this was detected.
	SourceMeeting string `json:"source_meeting,omitempty"`

	// RelatedTopics lists related topic areas.
	RelatedTopics []string `json:"related_topics,omitempty"`
}

// BottleneckConfig contains thresholds for bottleneck detection.
type BottleneckConfig struct {
	// InactiveMeetings is the number of meetings without discussion
	// before a topic is considered inactive.
	InactiveMeetings int

	// MaxDeferrals is the number of consecutive deferrals before
	// flagging as a bottleneck.
	MaxDeferrals int

	// OverdueDays is the number of days past due before flagging
	// an action as a bottleneck.
	OverdueDays int

	// Now is the reference time for calculations.
	Now time.Time
}

// DefaultBottleneckConfig returns sensible default thresholds.
func DefaultBottleneckConfig() BottleneckConfig {
	return BottleneckConfig{
		InactiveMeetings: 3,
		MaxDeferrals:     3,
		OverdueDays:      7,
		Now:              time.Now(),
	}
}

// BottleneckReport contains the results of bottleneck analysis.
type BottleneckReport struct {
	// ProcessURI is the deliberation process analyzed.
	ProcessURI string `json:"process_uri"`

	// AnalyzedAt is when the analysis was performed.
	AnalyzedAt time.Time `json:"analyzed_at"`

	// Bottlenecks lists all detected bottlenecks.
	Bottlenecks []Bottleneck `json:"bottlenecks"`

	// Summary provides high-level statistics.
	Summary BottleneckSummary `json:"summary"`
}

// BottleneckSummary provides aggregate statistics.
type BottleneckSummary struct {
	// TotalBottlenecks is the total count.
	TotalBottlenecks int `json:"total_bottlenecks"`

	// CriticalCount is the number of critical bottlenecks.
	CriticalCount int `json:"critical_count"`

	// HighCount is the number of high severity bottlenecks.
	HighCount int `json:"high_count"`

	// MediumCount is the number of medium severity bottlenecks.
	MediumCount int `json:"medium_count"`

	// LowCount is the number of low severity bottlenecks.
	LowCount int `json:"low_count"`

	// ByType counts bottlenecks by type.
	ByType map[BottleneckType]int `json:"by_type"`

	// OldestStall is the date of the oldest bottleneck.
	OldestStall *time.Time `json:"oldest_stall,omitempty"`

	// MostAffected lists the most impacted items.
	MostAffected []string `json:"most_affected,omitempty"`
}

// BottleneckDetector analyzes deliberation graphs to identify stalls.
type BottleneckDetector struct {
	// store is the triple store containing deliberation data.
	store *store.TripleStore

	// config contains detection thresholds.
	config BottleneckConfig

	// baseURI for constructing URIs.
	baseURI string
}

// NewBottleneckDetector creates a new bottleneck detector.
func NewBottleneckDetector(tripleStore *store.TripleStore, baseURI string) *BottleneckDetector {
	return &BottleneckDetector{
		store:   tripleStore,
		config:  DefaultBottleneckConfig(),
		baseURI: baseURI,
	}
}

// WithConfig sets custom configuration thresholds.
func (d *BottleneckDetector) WithConfig(config BottleneckConfig) *BottleneckDetector {
	d.config = config
	return d
}

// DetectBottlenecks performs comprehensive bottleneck analysis.
func (d *BottleneckDetector) DetectBottlenecks() (*BottleneckReport, error) {
	if d.store == nil {
		return nil, fmt.Errorf("no triple store configured")
	}

	report := &BottleneckReport{
		AnalyzedAt:  time.Now(),
		Bottlenecks: make([]Bottleneck, 0),
		Summary: BottleneckSummary{
			ByType: make(map[BottleneckType]int),
		},
	}

	// Load meetings in chronological order
	meetings, err := d.loadMeetings()
	if err != nil {
		return nil, fmt.Errorf("failed to load meetings: %w", err)
	}

	// 1. Find inactive topics
	inactiveBottlenecks := d.findInactiveTopics(meetings)
	report.Bottlenecks = append(report.Bottlenecks, inactiveBottlenecks...)

	// 2. Find repeated deferrals
	deferralBottlenecks := d.findRepeatedDeferrals(meetings)
	report.Bottlenecks = append(report.Bottlenecks, deferralBottlenecks...)

	// 3. Find overdue actions
	overdueBottlenecks := d.findOverdueActions()
	report.Bottlenecks = append(report.Bottlenecks, overdueBottlenecks...)

	// 4. Find blocked decision chains
	blockedBottlenecks := d.findBlockedDecisions()
	report.Bottlenecks = append(report.Bottlenecks, blockedBottlenecks...)

	// 5. Find missing quorum issues
	quorumBottlenecks := d.findMissingQuorum(meetings)
	report.Bottlenecks = append(report.Bottlenecks, quorumBottlenecks...)

	// 6. Find circular dependencies
	circularBottlenecks := d.findCircularDependencies()
	report.Bottlenecks = append(report.Bottlenecks, circularBottlenecks...)

	// Sort by severity
	d.sortBySeverity(report.Bottlenecks)

	// Generate summary
	report.Summary = d.generateSummary(report.Bottlenecks)

	return report, nil
}

// loadMeetings retrieves all meetings sorted by date.
func (d *BottleneckDetector) loadMeetings() ([]*Meeting, error) {
	var meetings []*Meeting

	meetingTriples := d.store.Find("", store.RDFType, store.ClassMeeting)
	for _, triple := range meetingTriples {
		meetingURI := triple.Subject
		meeting := &Meeting{URI: meetingURI}

		// Load meeting date
		if props := d.store.Find(meetingURI, store.PropMeetingDate, ""); len(props) > 0 {
			if t, err := time.Parse(time.RFC3339, props[0].Object); err == nil {
				meeting.Date = t
			} else if t, err := time.Parse("2006-01-02", props[0].Object); err == nil {
				meeting.Date = t
			}
		}

		// Load meeting sequence
		if props := d.store.Find(meetingURI, store.PropMeetingSequence, ""); len(props) > 0 {
			fmt.Sscanf(props[0].Object, "%d", &meeting.Sequence)
		}

		// Load title
		if props := d.store.Find(meetingURI, store.RDFSLabel, ""); len(props) > 0 {
			meeting.Title = props[0].Object
		}

		// Load status
		if props := d.store.Find(meetingURI, store.PropMeetingStatus, ""); len(props) > 0 {
			meeting.Status = parseMeetingStatus(props[0].Object)
		}

		meetings = append(meetings, meeting)
	}

	// Sort by date
	sort.Slice(meetings, func(i, j int) bool {
		return meetings[i].Date.Before(meetings[j].Date)
	})

	return meetings, nil
}

// findInactiveTopics identifies topics not discussed in recent meetings.
func (d *BottleneckDetector) findInactiveTopics(meetings []*Meeting) []Bottleneck {
	var bottlenecks []Bottleneck

	if len(meetings) < d.config.InactiveMeetings+1 {
		return bottlenecks
	}

	// Track when each topic was last discussed
	topicLastDiscussed := make(map[string]int) // topic URI -> meeting index
	topicFirstDiscussed := make(map[string]time.Time)

	for i, meeting := range meetings {
		// Find provisions discussed in this meeting
		agendaItems := d.store.Find(meeting.URI, store.PropHasAgendaItem, "")
		for _, ai := range agendaItems {
			// Get provisions discussed under this agenda item
			provisions := d.store.Find(ai.Object, store.PropProvisionDiscussed, "")
			for _, prov := range provisions {
				topicLastDiscussed[prov.Object] = i
				if _, exists := topicFirstDiscussed[prov.Object]; !exists {
					topicFirstDiscussed[prov.Object] = meeting.Date
				}
			}
		}
	}

	// Find topics not discussed in recent N meetings
	recentThreshold := len(meetings) - d.config.InactiveMeetings
	for topic, lastIdx := range topicLastDiscussed {
		if lastIdx < recentThreshold {
			meetingsSince := len(meetings) - lastIdx - 1
			severity := d.calculateSeverity(meetingsSince)

			bottleneck := Bottleneck{
				Type:          BottleneckInactiveTopic,
				Severity:      severity,
				Description:   fmt.Sprintf("Topic inactive for %d meetings", meetingsSince),
				AffectedItems: []string{topic},
				StalledSince:  meetings[lastIdx].Date,
				MeetingCount:  meetingsSince,
				Suggestions: []string{
					"Add to upcoming agenda",
					"Close topic if no longer relevant",
					"Assign action item to revive discussion",
				},
			}

			// Get topic label
			if labels := d.store.Find(topic, store.RDFSLabel, ""); len(labels) > 0 {
				bottleneck.Description = fmt.Sprintf("%s inactive for %d meetings", labels[0].Object, meetingsSince)
			}

			bottlenecks = append(bottlenecks, bottleneck)
		}
	}

	return bottlenecks
}

// findRepeatedDeferrals identifies items deferred multiple times.
func (d *BottleneckDetector) findRepeatedDeferrals(meetings []*Meeting) []Bottleneck {
	var bottlenecks []Bottleneck

	// Track deferral counts by agenda item topic
	deferralCounts := make(map[string]int)
	deferralFirstDate := make(map[string]time.Time)
	deferralMeetings := make(map[string][]string)

	for _, meeting := range meetings {
		agendaItems := d.store.Find(meeting.URI, store.PropHasAgendaItem, "")
		for _, ai := range agendaItems {
			aiURI := ai.Object

			// Check if this item was deferred
			if props := d.store.Find(aiURI, store.PropAgendaItemOutcome, ""); len(props) > 0 {
				if props[0].Object == "deferred" {
					// Get the topic/title for grouping
					title := aiURI
					if labels := d.store.Find(aiURI, store.RDFSLabel, ""); len(labels) > 0 {
						title = labels[0].Object
					}

					deferralCounts[title]++
					deferralMeetings[title] = append(deferralMeetings[title], meeting.URI)
					if _, exists := deferralFirstDate[title]; !exists {
						deferralFirstDate[title] = meeting.Date
					}
				}
			}
		}
	}

	// Flag items deferred more than threshold
	for topic, count := range deferralCounts {
		if count >= d.config.MaxDeferrals {
			severity := d.calculateSeverity(count)

			bottleneck := Bottleneck{
				Type:          BottleneckRepeatedDeferral,
				Severity:      severity,
				Description:   fmt.Sprintf("Item deferred %d times: %s", count, topic),
				AffectedItems: deferralMeetings[topic],
				StalledSince:  deferralFirstDate[topic],
				MeetingCount:  count,
				Suggestions: []string{
					"Schedule dedicated session to resolve",
					"Break item into smaller sub-items",
					"Identify and address blocking issues",
					"Consider withdrawing if no longer viable",
				},
			}

			bottlenecks = append(bottlenecks, bottleneck)
		}
	}

	return bottlenecks
}

// findOverdueActions identifies action items past their due date.
func (d *BottleneckDetector) findOverdueActions() []Bottleneck {
	var bottlenecks []Bottleneck

	// Find all action items
	actionTriples := d.store.Find("", store.RDFType, store.ClassActionItem)
	for _, triple := range actionTriples {
		actionURI := triple.Subject

		// Check status - skip completed/cancelled
		if props := d.store.Find(actionURI, store.PropActionStatus, ""); len(props) > 0 {
			status := props[0].Object
			if status == "completed" || status == "cancelled" {
				continue
			}
		}

		// Check due date
		var dueDate time.Time
		if props := d.store.Find(actionURI, store.PropActionDueDate, ""); len(props) > 0 {
			if t, err := time.Parse(time.RFC3339, props[0].Object); err == nil {
				dueDate = t
			} else if t, err := time.Parse("2006-01-02", props[0].Object); err == nil {
				dueDate = t
			}
		}

		if dueDate.IsZero() {
			continue
		}

		// Check if overdue
		daysOverdue := int(d.config.Now.Sub(dueDate).Hours() / 24)
		if daysOverdue >= d.config.OverdueDays {
			severity := d.calculateOverdueSeverity(daysOverdue)

			// Get action description
			description := actionURI
			if props := d.store.Find(actionURI, store.RDFSLabel, ""); len(props) > 0 {
				description = props[0].Object
			}

			// Get assignee
			var assignee string
			if props := d.store.Find(actionURI, store.PropActionAssignedTo, ""); len(props) > 0 {
				assignee = props[0].Object
			}

			bottleneck := Bottleneck{
				Type:          BottleneckOverdueAction,
				Severity:      severity,
				Description:   fmt.Sprintf("Action overdue by %d days: %s", daysOverdue, description),
				AffectedItems: []string{actionURI},
				StalledSince:  dueDate,
				MeetingCount:  0, // Not meeting-based
				Suggestions: []string{
					"Follow up with assigned party",
					"Escalate to leadership",
					"Reassign action if needed",
					"Extend deadline with justification",
				},
				RelatedTopics: []string{assignee},
			}

			bottlenecks = append(bottlenecks, bottleneck)
		}
	}

	return bottlenecks
}

// findBlockedDecisions identifies decisions waiting on other decisions.
func (d *BottleneckDetector) findBlockedDecisions() []Bottleneck {
	var bottlenecks []Bottleneck

	// Find all pending decisions/motions
	motionTriples := d.store.Find("", store.RDFType, store.ClassMotion)
	for _, triple := range motionTriples {
		motionURI := triple.Subject

		// Check if pending (not adopted, rejected, or withdrawn)
		isPending := true
		if props := d.store.Find(motionURI, store.PropMotionStatus, ""); len(props) > 0 {
			status := props[0].Object
			if status == "adopted" || status == "rejected" || status == "withdrawn" {
				isPending = false
			}
		}

		if !isPending {
			continue
		}

		// Check for dependencies
		dependencies := d.store.Find(motionURI, store.PropReferences, "")
		var blockedBy []string

		for _, dep := range dependencies {
			targetURI := dep.Object

			// Check if the target is also pending
			targetPending := false
			if props := d.store.Find(targetURI, store.PropMotionStatus, ""); len(props) > 0 {
				status := props[0].Object
				if status != "adopted" && status != "rejected" && status != "withdrawn" {
					targetPending = true
				}
			}

			if targetPending {
				blockedBy = append(blockedBy, targetURI)
			}
		}

		if len(blockedBy) > 0 {
			// Get motion description
			description := motionURI
			if props := d.store.Find(motionURI, store.RDFSLabel, ""); len(props) > 0 {
				description = props[0].Object
			}

			// Find when motion was proposed
			var proposedDate time.Time
			if props := d.store.Find(motionURI, store.PropMeetingDate, ""); len(props) > 0 {
				if t, err := time.Parse(time.RFC3339, props[0].Object); err == nil {
					proposedDate = t
				}
			}

			severity := BottleneckMedium
			if len(blockedBy) > 2 {
				severity = BottleneckHigh
			}

			bottleneck := Bottleneck{
				Type:          BottleneckBlockedDecision,
				Severity:      severity,
				Description:   fmt.Sprintf("Decision blocked by %d pending items: %s", len(blockedBy), description),
				AffectedItems: []string{motionURI},
				BlockedBy:     blockedBy,
				StalledSince:  proposedDate,
				Suggestions: []string{
					"Resolve blocking items first",
					"Decouple amendments if possible",
					"Schedule joint discussion of related items",
				},
			}

			bottlenecks = append(bottlenecks, bottleneck)
		}
	}

	return bottlenecks
}

// findMissingQuorum identifies votes that couldn't proceed due to quorum.
func (d *BottleneckDetector) findMissingQuorum(meetings []*Meeting) []Bottleneck {
	var bottlenecks []Bottleneck

	for _, meeting := range meetings {
		// Find agenda items with no_quorum outcome
		agendaItems := d.store.Find(meeting.URI, store.PropHasAgendaItem, "")
		for _, ai := range agendaItems {
			aiURI := ai.Object

			if props := d.store.Find(aiURI, store.PropAgendaItemOutcome, ""); len(props) > 0 {
				if props[0].Object == "no_quorum" {
					// Get item description
					description := aiURI
					if labels := d.store.Find(aiURI, store.RDFSLabel, ""); len(labels) > 0 {
						description = labels[0].Object
					}

					bottleneck := Bottleneck{
						Type:          BottleneckMissingQuorum,
						Severity:      BottleneckMedium,
						Description:   fmt.Sprintf("Vote failed due to missing quorum: %s", description),
						AffectedItems: []string{aiURI},
						StalledSince:  meeting.Date,
						SourceMeeting: meeting.URI,
						MeetingCount:  1,
						Suggestions: []string{
							"Reschedule vote with confirmed attendance",
							"Consider proxy voting if allowed",
							"Address attendance issues with participants",
						},
					}

					bottlenecks = append(bottlenecks, bottleneck)
				}
			}
		}
	}

	return bottlenecks
}

// findCircularDependencies identifies amendments referencing each other.
func (d *BottleneckDetector) findCircularDependencies() []Bottleneck {
	var bottlenecks []Bottleneck

	// Build dependency graph
	dependencies := make(map[string][]string)

	// Find all references between motions/amendments
	motionTriples := d.store.Find("", store.RDFType, store.ClassMotion)
	for _, triple := range motionTriples {
		motionURI := triple.Subject

		refs := d.store.Find(motionURI, store.PropReferences, "")
		for _, ref := range refs {
			dependencies[motionURI] = append(dependencies[motionURI], ref.Object)
		}
	}

	// Detect cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var cycles [][]string

	var detectCycle func(node string, path []string) bool
	detectCycle = func(node string, path []string) bool {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, neighbor := range dependencies[node] {
			if !visited[neighbor] {
				if detectCycle(neighbor, path) {
					return true
				}
			} else if recStack[neighbor] {
				// Found cycle - extract it
				cycleStart := -1
				for i, n := range path {
					if n == neighbor {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cycle := make([]string, len(path)-cycleStart)
					copy(cycle, path[cycleStart:])
					cycles = append(cycles, cycle)
				}
				return true
			}
		}

		recStack[node] = false
		return false
	}

	for node := range dependencies {
		if !visited[node] {
			detectCycle(node, nil)
		}
	}

	// Create bottlenecks for each cycle
	for _, cycle := range cycles {
		bottleneck := Bottleneck{
			Type:          BottleneckCircularDependency,
			Severity:      BottleneckCritical,
			Description:   fmt.Sprintf("Circular dependency detected between %d items", len(cycle)),
			AffectedItems: cycle,
			BlockedBy:     cycle,
			Suggestions: []string{
				"Break circular reference by decoupling amendments",
				"Merge related amendments into single proposal",
				"Prioritize one amendment and defer others",
			},
		}

		bottlenecks = append(bottlenecks, bottleneck)
	}

	return bottlenecks
}

// calculateSeverity determines severity based on meeting count.
func (d *BottleneckDetector) calculateSeverity(meetingCount int) BottleneckSeverity {
	switch {
	case meetingCount >= 5:
		return BottleneckHigh
	case meetingCount >= 3:
		return BottleneckMedium
	default:
		return BottleneckLow
	}
}

// calculateOverdueSeverity determines severity based on days overdue.
func (d *BottleneckDetector) calculateOverdueSeverity(daysOverdue int) BottleneckSeverity {
	switch {
	case daysOverdue >= 60:
		return BottleneckCritical
	case daysOverdue >= 30:
		return BottleneckHigh
	case daysOverdue >= 14:
		return BottleneckMedium
	default:
		return BottleneckLow
	}
}

// sortBySeverity sorts bottlenecks with most severe first.
func (d *BottleneckDetector) sortBySeverity(bottlenecks []Bottleneck) {
	sort.Slice(bottlenecks, func(i, j int) bool {
		if bottlenecks[i].Severity != bottlenecks[j].Severity {
			return bottlenecks[i].Severity > bottlenecks[j].Severity
		}
		return bottlenecks[i].StalledSince.Before(bottlenecks[j].StalledSince)
	})
}

// generateSummary creates aggregate statistics.
func (d *BottleneckDetector) generateSummary(bottlenecks []Bottleneck) BottleneckSummary {
	summary := BottleneckSummary{
		TotalBottlenecks: len(bottlenecks),
		ByType:           make(map[BottleneckType]int),
	}

	var oldestStall *time.Time
	affectedCounts := make(map[string]int)

	for _, b := range bottlenecks {
		// Count by severity
		switch b.Severity {
		case BottleneckCritical:
			summary.CriticalCount++
		case BottleneckHigh:
			summary.HighCount++
		case BottleneckMedium:
			summary.MediumCount++
		case BottleneckLow:
			summary.LowCount++
		}

		// Count by type
		summary.ByType[b.Type]++

		// Track oldest stall
		if !b.StalledSince.IsZero() {
			if oldestStall == nil || b.StalledSince.Before(*oldestStall) {
				oldestStall = &b.StalledSince
			}
		}

		// Count affected items
		for _, item := range b.AffectedItems {
			affectedCounts[item]++
		}
	}

	summary.OldestStall = oldestStall

	// Find most affected items
	type itemCount struct {
		item  string
		count int
	}
	var items []itemCount
	for item, count := range affectedCounts {
		items = append(items, itemCount{item, count})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].count > items[j].count
	})

	// Take top 5 most affected
	for i := 0; i < len(items) && i < 5; i++ {
		summary.MostAffected = append(summary.MostAffected, items[i].item)
	}

	return summary
}

// GetBottlenecksForProvision returns bottlenecks affecting a specific provision.
func (d *BottleneckDetector) GetBottlenecksForProvision(provisionURI string) ([]Bottleneck, error) {
	report, err := d.DetectBottlenecks()
	if err != nil {
		return nil, err
	}

	var relevant []Bottleneck
	for _, b := range report.Bottlenecks {
		for _, item := range b.AffectedItems {
			if item == provisionURI {
				relevant = append(relevant, b)
				break
			}
		}
		for _, blocker := range b.BlockedBy {
			if blocker == provisionURI {
				relevant = append(relevant, b)
				break
			}
		}
	}

	return relevant, nil
}

// GenerateReport formats the bottleneck report for display.
func (d *BottleneckDetector) GenerateReport(report *BottleneckReport) string {
	var result string

	result += fmt.Sprintf("Deliberation Bottleneck Analysis\n")
	result += fmt.Sprintf("================================\n")
	result += fmt.Sprintf("Analyzed: %s\n\n", report.AnalyzedAt.Format("2006-01-02 15:04"))

	// Group by severity
	bySeverity := make(map[BottleneckSeverity][]Bottleneck)
	for _, b := range report.Bottlenecks {
		bySeverity[b.Severity] = append(bySeverity[b.Severity], b)
	}

	// Print critical first
	for _, severity := range []BottleneckSeverity{BottleneckCritical, BottleneckHigh, BottleneckMedium, BottleneckLow} {
		bottlenecks := bySeverity[severity]
		if len(bottlenecks) == 0 {
			continue
		}

		result += fmt.Sprintf("%s (%d):\n", severityLabel(severity), len(bottlenecks))

		for _, b := range bottlenecks {
			result += fmt.Sprintf("  %s\n", b.Description)

			if !b.StalledSince.IsZero() {
				result += fmt.Sprintf("    Stalled since: %s", b.StalledSince.Format("2006-01-02"))
				if b.MeetingCount > 0 {
					result += fmt.Sprintf(" (%d meetings)", b.MeetingCount)
				}
				result += "\n"
			}

			if len(b.BlockedBy) > 0 {
				result += fmt.Sprintf("    Blocked by: %v\n", b.BlockedBy)
			}

			if len(b.Suggestions) > 0 {
				result += fmt.Sprintf("    Suggestion: %s\n", b.Suggestions[0])
			}

			result += "\n"
		}
	}

	// Print summary
	result += fmt.Sprintf("Summary:\n")
	result += fmt.Sprintf("  Total: %d bottlenecks\n", report.Summary.TotalBottlenecks)
	result += fmt.Sprintf("  Critical: %d, High: %d, Medium: %d, Low: %d\n",
		report.Summary.CriticalCount, report.Summary.HighCount,
		report.Summary.MediumCount, report.Summary.LowCount)

	return result
}

// severityLabel returns a display label for severity.
func severityLabel(s BottleneckSeverity) string {
	switch s {
	case BottleneckCritical:
		return "CRITICAL"
	case BottleneckHigh:
		return "HIGH"
	case BottleneckMedium:
		return "MEDIUM"
	case BottleneckLow:
		return "LOW"
	default:
		return "UNKNOWN"
	}
}

// parseMeetingStatus parses a meeting status string.
func parseMeetingStatus(s string) MeetingStatus {
	switch s {
	case "scheduled":
		return MeetingScheduled
	case "in_progress":
		return MeetingInProgress
	case "completed":
		return MeetingCompleted
	case "cancelled":
		return MeetingCancelled
	case "postponed":
		return MeetingPostponed
	default:
		return MeetingScheduled
	}
}
