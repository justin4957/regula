// Package query provides SPARQL query parsing and execution.
package query

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// TemporalQueryExecutor extends the query executor with temporal query capabilities.
// It enables point-in-time queries, range queries, and version timeline queries.
type TemporalQueryExecutor struct {
	*Executor
	temporalStore *store.TemporalStore
}

// NewTemporalQueryExecutor creates a new temporal query executor.
func NewTemporalQueryExecutor(ts *store.TemporalStore, opts ...ExecutorOption) *TemporalQueryExecutor {
	return &TemporalQueryExecutor{
		Executor:      NewExecutor(ts.TripleStore, opts...),
		temporalStore: ts,
	}
}

// Timeline represents a chronological sequence of events for a subject.
type Timeline struct {
	// Subject is the URI of the entity being tracked.
	Subject string `json:"subject"`

	// Events lists all timeline events in chronological order.
	Events []TimelineEvent `json:"events"`

	// FirstEvent is the timestamp of the earliest event.
	FirstEvent time.Time `json:"first_event,omitempty"`

	// LastEvent is the timestamp of the most recent event.
	LastEvent time.Time `json:"last_event,omitempty"`

	// Duration is the time span from first to last event.
	Duration time.Duration `json:"duration,omitempty"`
}

// TimelineEvent represents a single event in a subject's timeline.
type TimelineEvent struct {
	// Date is when the event occurred.
	Date time.Time `json:"date"`

	// EventType classifies the event (proposal, amendment, adoption, etc.).
	EventType string `json:"event_type"`

	// Description provides a human-readable description.
	Description string `json:"description"`

	// Meeting is the URI of the meeting where this event occurred.
	Meeting string `json:"meeting,omitempty"`

	// TriplesDelta indicates the number of triples added/removed.
	TriplesDelta int `json:"triples_delta,omitempty"`

	// Version is the version URI associated with this event.
	Version string `json:"version,omitempty"`
}

// TemporalResult extends QueryResult with temporal metadata.
type TemporalResult struct {
	*QueryResult

	// AsOf is the point in time the query was evaluated at.
	AsOf time.Time `json:"as_of,omitempty"`

	// FromTime is the start of the range for range queries.
	FromTime time.Time `json:"from_time,omitempty"`

	// ToTime is the end of the range for range queries.
	ToTime time.Time `json:"to_time,omitempty"`

	// Version is the version that was active at the query time.
	Version *store.VersionInfo `json:"version,omitempty"`
}

// RangeQueryResult contains results from a temporal range query.
type RangeQueryResult struct {
	// FromTime is the start of the range.
	FromTime time.Time `json:"from_time"`

	// ToTime is the end of the range.
	ToTime time.Time `json:"to_time"`

	// Changes lists all changes within the range.
	Changes []RangeChange `json:"changes"`

	// Summary provides aggregate statistics.
	Summary RangeSummary `json:"summary"`
}

// RangeChange represents a single change within a time range.
type RangeChange struct {
	// Subject is the entity that changed.
	Subject string `json:"subject"`

	// ChangeType classifies the change (added, modified, removed).
	ChangeType string `json:"change_type"`

	// Date is when the change occurred.
	Date time.Time `json:"date"`

	// Meeting is the meeting where this change occurred.
	Meeting string `json:"meeting,omitempty"`

	// Description describes the change.
	Description string `json:"description,omitempty"`

	// OldVersion is the previous version URI.
	OldVersion string `json:"old_version,omitempty"`

	// NewVersion is the new version URI.
	NewVersion string `json:"new_version,omitempty"`
}

// RangeSummary provides aggregate statistics for a range query.
type RangeSummary struct {
	// TotalChanges is the count of all changes.
	TotalChanges int `json:"total_changes"`

	// AddedCount is the number of additions.
	AddedCount int `json:"added_count"`

	// ModifiedCount is the number of modifications.
	ModifiedCount int `json:"modified_count"`

	// RemovedCount is the number of removals.
	RemovedCount int `json:"removed_count"`

	// AffectedSubjects is the count of unique subjects affected.
	AffectedSubjects int `json:"affected_subjects"`
}

// VersionsResult contains all versions of a subject.
type VersionsResult struct {
	// Subject is the canonical URI of the entity.
	Subject string `json:"subject"`

	// Versions lists all versions in chronological order.
	Versions []VersionEntry `json:"versions"`

	// CurrentVersion is the currently active version.
	CurrentVersion *VersionEntry `json:"current_version,omitempty"`
}

// VersionEntry represents a single version in the versions result.
type VersionEntry struct {
	// URI is the version-specific URI.
	URI string `json:"uri"`

	// Version is the version identifier.
	Version string `json:"version"`

	// Date is when this version became valid.
	Date time.Time `json:"date"`

	// Status is the version status (draft, active, superseded).
	Status string `json:"status"`

	// Meeting is the meeting where this version was adopted.
	Meeting string `json:"meeting,omitempty"`

	// Text is the text content at this version (if available).
	Text string `json:"text,omitempty"`
}

// DurationResult contains duration analysis for a subject.
type DurationResult struct {
	// Subject is the entity being analyzed.
	Subject string `json:"subject"`

	// ProposalDate is when the subject was first proposed.
	ProposalDate time.Time `json:"proposal_date,omitempty"`

	// AdoptionDate is when the subject was adopted.
	AdoptionDate time.Time `json:"adoption_date,omitempty"`

	// TotalDuration is the time from proposal to adoption.
	TotalDuration time.Duration `json:"total_duration,omitempty"`

	// DiscussionCount is the number of meetings where this was discussed.
	DiscussionCount int `json:"discussion_count"`

	// AmendmentCount is the number of amendments made.
	AmendmentCount int `json:"amendment_count"`
}

// ExecuteAsOf executes a SPARQL query at a specific point in time.
// It filters results to only include data valid at the specified time.
func (e *TemporalQueryExecutor) ExecuteAsOf(queryStr string, asOf time.Time) (*TemporalResult, error) {
	return e.ExecuteAsOfWithContext(context.Background(), queryStr, asOf)
}

// ExecuteAsOfWithContext executes a point-in-time query with context.
func (e *TemporalQueryExecutor) ExecuteAsOfWithContext(ctx context.Context, queryStr string, asOf time.Time) (*TemporalResult, error) {
	// Parse the query
	query, err := ParseQuery(queryStr)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	if query.Type != SelectQueryType {
		return nil, fmt.Errorf("AS_OF queries only support SELECT, got: %s", query.Type)
	}

	// Execute base query
	result, err := e.ExecuteWithContext(ctx, query)
	if err != nil {
		return nil, err
	}

	// Filter bindings to only include subjects valid at asOf time
	filteredBindings := e.filterByValidity(result.Bindings, asOf)

	temporalResult := &TemporalResult{
		QueryResult: &QueryResult{
			Variables: result.Variables,
			Bindings:  filteredBindings,
			Count:     len(filteredBindings),
			Metrics:   result.Metrics,
		},
		AsOf: asOf,
	}

	return temporalResult, nil
}

// ExecuteBetween executes a SPARQL query and returns changes within a time range.
func (e *TemporalQueryExecutor) ExecuteBetween(queryStr string, from, to time.Time) (*RangeQueryResult, error) {
	return e.ExecuteBetweenWithContext(context.Background(), queryStr, from, to)
}

// ExecuteBetweenWithContext executes a range query with context.
func (e *TemporalQueryExecutor) ExecuteBetweenWithContext(ctx context.Context, queryStr string, from, to time.Time) (*RangeQueryResult, error) {
	// Parse the query to extract the subject variable/pattern
	query, err := ParseQuery(queryStr)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	if query.Type != SelectQueryType {
		return nil, fmt.Errorf("BETWEEN queries only support SELECT, got: %s", query.Type)
	}

	// Execute base query to get matching subjects
	result, err := e.ExecuteWithContext(ctx, query)
	if err != nil {
		return nil, err
	}

	// Collect unique subjects from results
	subjects := make(map[string]bool)
	for _, binding := range result.Bindings {
		for _, value := range binding {
			if strings.Contains(value, "://") || strings.HasPrefix(value, "reg:") {
				subjects[value] = true
			}
		}
	}

	// Find changes for each subject in the range
	var changes []RangeChange
	affectedSubjects := make(map[string]bool)

	for subject := range subjects {
		subjectChanges := e.findChangesInRange(subject, from, to)
		for _, change := range subjectChanges {
			changes = append(changes, change)
			affectedSubjects[change.Subject] = true
		}
	}

	// Sort changes by date
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Date.Before(changes[j].Date)
	})

	// Build summary
	summary := RangeSummary{
		TotalChanges:     len(changes),
		AffectedSubjects: len(affectedSubjects),
	}
	for _, change := range changes {
		switch change.ChangeType {
		case "added":
			summary.AddedCount++
		case "modified":
			summary.ModifiedCount++
		case "removed":
			summary.RemovedCount++
		}
	}

	return &RangeQueryResult{
		FromTime: from,
		ToTime:   to,
		Changes:  changes,
		Summary:  summary,
	}, nil
}

// ExecuteBetweenMeetings executes a range query between two meeting URIs.
func (e *TemporalQueryExecutor) ExecuteBetweenMeetings(queryStr string, fromMeeting, toMeeting string) (*RangeQueryResult, error) {
	// Look up meeting dates
	fromDate, err := e.getMeetingDate(fromMeeting)
	if err != nil {
		return nil, fmt.Errorf("cannot find date for meeting %s: %w", fromMeeting, err)
	}

	toDate, err := e.getMeetingDate(toMeeting)
	if err != nil {
		return nil, fmt.Errorf("cannot find date for meeting %s: %w", toMeeting, err)
	}

	return e.ExecuteBetween(queryStr, fromDate, toDate)
}

// GetVersions returns all versions of a subject entity.
func (e *TemporalQueryExecutor) GetVersions(subject string) (*VersionsResult, error) {
	history, err := e.temporalStore.GetVersionHistory(subject)
	if err != nil {
		// Try to build version list from triples if no indexed history
		return e.buildVersionsFromTriples(subject)
	}

	result := &VersionsResult{
		Subject:  subject,
		Versions: make([]VersionEntry, len(history.Versions)),
	}

	for i, v := range history.Versions {
		entry := VersionEntry{
			URI:     v.URI,
			Version: v.Version,
			Date:    v.ValidFrom,
			Status:  v.Status,
			Meeting: v.MeetingURI,
		}

		// Try to get text for this version
		textTriples := e.store.Find(v.URI, store.PropText, "")
		if len(textTriples) > 0 {
			entry.Text = textTriples[0].Object
		}

		result.Versions[i] = entry

		if v.Status == "active" {
			result.CurrentVersion = &entry
		}
	}

	return result, nil
}

// GetTimeline returns a chronological timeline for a subject.
func (e *TemporalQueryExecutor) GetTimeline(subject string) (*Timeline, error) {
	timeline := &Timeline{
		Subject: subject,
		Events:  []TimelineEvent{},
	}

	// Collect events from different sources

	// 1. Version events
	if history, err := e.temporalStore.GetVersionHistory(subject); err == nil {
		for _, v := range history.Versions {
			event := TimelineEvent{
				Date:        v.ValidFrom,
				EventType:   e.versionStatusToEventType(v.Status),
				Description: fmt.Sprintf("Version %s", v.Version),
				Meeting:     v.MeetingURI,
				Version:     v.URI,
			}
			timeline.Events = append(timeline.Events, event)
		}
	}

	// 2. Decision events
	decisionTriples := e.store.Find(subject, store.PropDecidedAt, "")
	for _, t := range decisionTriples {
		meetingDate, _ := e.getMeetingDate(t.Object)
		event := TimelineEvent{
			Date:        meetingDate,
			EventType:   "decision",
			Description: "Decision made",
			Meeting:     t.Object,
		}
		timeline.Events = append(timeline.Events, event)
	}

	// 3. Discussion events
	discussionTriples := e.store.Find(subject, store.PropDiscussedAt, "")
	for _, t := range discussionTriples {
		meetingDate, _ := e.getMeetingDate(t.Object)
		event := TimelineEvent{
			Date:        meetingDate,
			EventType:   "discussion",
			Description: "Discussed",
			Meeting:     t.Object,
		}
		timeline.Events = append(timeline.Events, event)
	}

	// 4. Amendment events (find amendments targeting this provision)
	amendmentTriples := e.store.Find("", store.PropTargetProvision, subject)
	for _, t := range amendmentTriples {
		// Get motion status and date
		statusTriples := e.store.Find(t.Subject, store.PropMotionStatus, "")
		dateTriples := e.store.Find(t.Subject, store.PropAdoptedDate, "")

		eventType := "amendment_proposed"
		if len(statusTriples) > 0 && statusTriples[0].Object == "adopted" {
			eventType = "amendment_adopted"
		}

		var eventDate time.Time
		if len(dateTriples) > 0 {
			eventDate, _ = time.Parse(time.RFC3339, dateTriples[0].Object)
		}

		event := TimelineEvent{
			Date:        eventDate,
			EventType:   eventType,
			Description: fmt.Sprintf("Amendment: %s", t.Subject),
		}
		timeline.Events = append(timeline.Events, event)
	}

	// Sort events by date
	sort.Slice(timeline.Events, func(i, j int) bool {
		return timeline.Events[i].Date.Before(timeline.Events[j].Date)
	})

	// Calculate duration stats
	if len(timeline.Events) > 0 {
		timeline.FirstEvent = timeline.Events[0].Date
		timeline.LastEvent = timeline.Events[len(timeline.Events)-1].Date
		if !timeline.FirstEvent.IsZero() && !timeline.LastEvent.IsZero() {
			timeline.Duration = timeline.LastEvent.Sub(timeline.FirstEvent)
		}
	}

	return timeline, nil
}

// GetDuration calculates the duration from proposal to adoption for a subject.
func (e *TemporalQueryExecutor) GetDuration(subject string) (*DurationResult, error) {
	result := &DurationResult{
		Subject: subject,
	}

	// Find proposal date (earliest valid from date)
	if history, err := e.temporalStore.GetVersionHistory(subject); err == nil && len(history.Versions) > 0 {
		result.ProposalDate = history.Versions[0].ValidFrom
	}

	// Find adoption date (when status became active)
	if history, err := e.temporalStore.GetVersionHistory(subject); err == nil {
		for _, v := range history.Versions {
			if v.Status == "active" {
				result.AdoptionDate = v.ValidFrom
				break
			}
		}
	}

	// Calculate total duration
	if !result.ProposalDate.IsZero() && !result.AdoptionDate.IsZero() {
		result.TotalDuration = result.AdoptionDate.Sub(result.ProposalDate)
	}

	// Count discussions
	discussionTriples := e.store.Find(subject, store.PropDiscussedAt, "")
	result.DiscussionCount = len(discussionTriples)

	// Count amendments
	amendmentTriples := e.store.Find("", store.PropTargetProvision, subject)
	result.AmendmentCount = len(amendmentTriples)

	return result, nil
}

// AverageTimeToAdoption calculates the average duration from proposal to adoption
// for a set of subjects matching a query pattern.
func (e *TemporalQueryExecutor) AverageTimeToAdoption(queryStr string) (time.Duration, int, error) {
	// Parse and execute query to get matching subjects
	query, err := ParseQuery(queryStr)
	if err != nil {
		return 0, 0, fmt.Errorf("parse error: %w", err)
	}

	result, err := e.Execute(query)
	if err != nil {
		return 0, 0, err
	}

	// Collect unique subjects
	subjects := make(map[string]bool)
	for _, binding := range result.Bindings {
		for _, value := range binding {
			if strings.Contains(value, "://") || strings.HasPrefix(value, "reg:") {
				subjects[value] = true
			}
		}
	}

	// Calculate durations
	var totalDuration time.Duration
	count := 0

	for subject := range subjects {
		duration, err := e.GetDuration(subject)
		if err == nil && duration.TotalDuration > 0 {
			totalDuration += duration.TotalDuration
			count++
		}
	}

	if count == 0 {
		return 0, 0, nil
	}

	averageDuration := totalDuration / time.Duration(count)
	return averageDuration, count, nil
}

// Helper methods

// filterByValidity filters bindings to include only subjects valid at the given time.
func (e *TemporalQueryExecutor) filterByValidity(bindings []map[string]string, asOf time.Time) []map[string]string {
	var filtered []map[string]string

	for _, binding := range bindings {
		valid := true

		// Check each value that could be a subject URI
		for _, value := range binding {
			if strings.Contains(value, "://") || strings.HasPrefix(value, "reg:") {
				// Try to find version info
				version, err := e.temporalStore.GetVersionAtTime(value, asOf)
				if err == nil && version != nil {
					// Subject has version info - check if it was valid
					if version.ValidFrom.After(asOf) {
						valid = false
						break
					}
					if !version.ValidUntil.IsZero() && version.ValidUntil.Before(asOf) {
						valid = false
						break
					}
				}
				// If no version info, assume always valid
			}
		}

		if valid {
			filtered = append(filtered, binding)
		}
	}

	return filtered
}

// findChangesInRange finds all changes to a subject within a time range.
func (e *TemporalQueryExecutor) findChangesInRange(subject string, from, to time.Time) []RangeChange {
	var changes []RangeChange

	// Get version history and find versions that changed in range
	history, err := e.temporalStore.GetVersionHistory(subject)
	if err != nil {
		return changes
	}

	for i, v := range history.Versions {
		if v.ValidFrom.After(from) && (v.ValidFrom.Before(to) || v.ValidFrom.Equal(to)) {
			changeType := "added"
			if i > 0 {
				changeType = "modified"
			}

			var oldVersion string
			if i > 0 {
				oldVersion = history.Versions[i-1].URI
			}

			changes = append(changes, RangeChange{
				Subject:     subject,
				ChangeType:  changeType,
				Date:        v.ValidFrom,
				Meeting:     v.MeetingURI,
				Description: fmt.Sprintf("Version %s", v.Version),
				OldVersion:  oldVersion,
				NewVersion:  v.URI,
			})
		}

		// Check if version was superseded within range
		if !v.ValidUntil.IsZero() && v.ValidUntil.After(from) && (v.ValidUntil.Before(to) || v.ValidUntil.Equal(to)) {
			if v.Status == "superseded" && i+1 < len(history.Versions) {
				// Already captured by the next version's addition
				continue
			}
			if v.Status == "withdrawn" {
				changes = append(changes, RangeChange{
					Subject:     subject,
					ChangeType:  "removed",
					Date:        v.ValidUntil,
					Description: fmt.Sprintf("Version %s withdrawn", v.Version),
					OldVersion:  v.URI,
				})
			}
		}
	}

	return changes
}

// getMeetingDate retrieves the date of a meeting from its URI.
func (e *TemporalQueryExecutor) getMeetingDate(meetingURI string) (time.Time, error) {
	dateTriples := e.store.Find(meetingURI, store.PropMeetingDate, "")
	if len(dateTriples) > 0 {
		return time.Parse(time.RFC3339, dateTriples[0].Object)
	}

	// Try alternative date properties
	dateTriples = e.store.Find(meetingURI, store.PropDate, "")
	if len(dateTriples) > 0 {
		return time.Parse(time.RFC3339, dateTriples[0].Object)
	}

	return time.Time{}, fmt.Errorf("no date found for meeting %s", meetingURI)
}

// buildVersionsFromTriples constructs version info from triples when no indexed history exists.
func (e *TemporalQueryExecutor) buildVersionsFromTriples(subject string) (*VersionsResult, error) {
	result := &VersionsResult{
		Subject:  subject,
		Versions: []VersionEntry{},
	}

	// Find versions via versionOf predicate
	versionTriples := e.store.Find("", store.PropVersionOf, subject)
	for _, t := range versionTriples {
		entry := VersionEntry{
			URI: t.Subject,
		}

		// Get version number
		numTriples := e.store.Find(t.Subject, store.PropVersionNumber, "")
		if len(numTriples) > 0 {
			entry.Version = numTriples[0].Object
		}

		// Get status
		statusTriples := e.store.Find(t.Subject, store.PropVersionStatus, "")
		if len(statusTriples) > 0 {
			entry.Status = statusTriples[0].Object
		}

		// Get valid from date
		dateTriples := e.store.Find(t.Subject, store.PropValidFrom, "")
		if len(dateTriples) > 0 {
			entry.Date, _ = time.Parse(time.RFC3339, dateTriples[0].Object)
		}

		// Get text
		textTriples := e.store.Find(t.Subject, store.PropText, "")
		if len(textTriples) > 0 {
			entry.Text = textTriples[0].Object
		}

		// Get meeting
		meetingTriples := e.store.Find(t.Subject, store.PropDecidedAt, "")
		if len(meetingTriples) > 0 {
			entry.Meeting = meetingTriples[0].Object
		}

		result.Versions = append(result.Versions, entry)

		if entry.Status == "active" {
			result.CurrentVersion = &entry
		}
	}

	// Sort by date
	sort.Slice(result.Versions, func(i, j int) bool {
		return result.Versions[i].Date.Before(result.Versions[j].Date)
	})

	if len(result.Versions) == 0 {
		return nil, fmt.Errorf("no versions found for %s", subject)
	}

	return result, nil
}

// versionStatusToEventType converts a version status to a timeline event type.
func (e *TemporalQueryExecutor) versionStatusToEventType(status string) string {
	switch status {
	case "draft":
		return "proposal"
	case "active":
		return "adoption"
	case "superseded":
		return "supersession"
	case "withdrawn":
		return "withdrawal"
	default:
		return "version_change"
	}
}

// ProvisionsInForceAt returns all provisions that were in force at a specific date.
func (e *TemporalQueryExecutor) ProvisionsInForceAt(asOf time.Time) ([]string, error) {
	activeVersions := e.temporalStore.GetActiveVersions()

	var inForce []string
	for _, v := range activeVersions {
		if v.ValidFrom.Before(asOf) || v.ValidFrom.Equal(asOf) {
			if v.ValidUntil.IsZero() || v.ValidUntil.After(asOf) {
				inForce = append(inForce, v.URI)
			}
		}
	}

	return inForce, nil
}

// AmendmentsAdoptedInRange returns all amendments adopted within a date range.
func (e *TemporalQueryExecutor) AmendmentsAdoptedInRange(from, to time.Time) ([]RangeChange, error) {
	var amendments []RangeChange

	// Query for all motions with adopted status
	statusTriples := e.store.Find("", store.PropMotionStatus, "adopted")
	for _, t := range statusTriples {
		// Get adoption date
		dateTriples := e.store.Find(t.Subject, store.PropAdoptedDate, "")
		if len(dateTriples) == 0 {
			continue
		}

		adoptionDate, err := time.Parse(time.RFC3339, dateTriples[0].Object)
		if err != nil {
			continue
		}

		// Check if within range
		if adoptionDate.After(from) && (adoptionDate.Before(to) || adoptionDate.Equal(to)) {
			// Get target provision
			targetTriples := e.store.Find(t.Subject, store.PropTargetProvision, "")
			target := ""
			if len(targetTriples) > 0 {
				target = targetTriples[0].Object
			}

			// Get description
			descTriples := e.store.Find(t.Subject, store.PropProposedText, "")
			desc := ""
			if len(descTriples) > 0 {
				desc = descTriples[0].Object
			}

			amendments = append(amendments, RangeChange{
				Subject:     t.Subject,
				ChangeType:  "amendment_adopted",
				Date:        adoptionDate,
				Description: desc,
				NewVersion:  target,
			})
		}
	}

	// Sort by date
	sort.Slice(amendments, func(i, j int) bool {
		return amendments[i].Date.Before(amendments[j].Date)
	})

	return amendments, nil
}
