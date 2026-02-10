package query

import (
	"testing"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// buildTemporalTestStore creates a temporal store with test data.
func buildTemporalTestStore() *store.TemporalStore {
	ts := store.NewTemporalStore()

	// Add Article 17 with multiple versions
	art17Canonical := "https://regula.dev/regulations/GDPR:Art17"

	// Version 1: Initial proposal
	v1 := store.VersionInfo{
		URI:        art17Canonical + ":v1",
		Version:    "1.0",
		ValidFrom:  time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC),
		ValidUntil: time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC),
		Status:     "superseded",
		MeetingURI: "https://regula.dev/meetings/wg-40",
	}
	ts.AddVersion(art17Canonical, v1)
	ts.AddVersioned(v1.URI, store.PropText, "Right to erasure (initial)", "v1")

	// Version 2: After amendment
	v2 := store.VersionInfo{
		URI:          art17Canonical + ":v2",
		Version:      "2.0",
		ValidFrom:    time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC),
		ValidUntil:   time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
		Status:       "superseded",
		MeetingURI:   "https://regula.dev/meetings/wg-45",
		SupersedesURI: v1.URI,
	}
	ts.AddVersion(art17Canonical, v2)
	ts.AddVersioned(v2.URI, store.PropText, "Right to erasure (amended)", "v2")

	// Version 3: Final adoption
	v3 := store.VersionInfo{
		URI:          art17Canonical + ":v3",
		Version:      "3.0",
		ValidFrom:    time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
		Status:       "active",
		MeetingURI:   "https://regula.dev/meetings/wg-50",
		SupersedesURI: v2.URI,
	}
	ts.AddVersion(art17Canonical, v3)
	ts.AddVersioned(v3.URI, store.PropText, "Right to erasure (final)", "v3")
	ts.SetCurrentVersion(art17Canonical, v3.URI)

	// Add meetings with dates
	meetings := []struct {
		uri  string
		date time.Time
		seq  string
	}{
		{"https://regula.dev/meetings/wg-40", time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC), "40"},
		{"https://regula.dev/meetings/wg-45", time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC), "45"},
		{"https://regula.dev/meetings/wg-50", time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC), "50"},
	}
	for _, m := range meetings {
		ts.Add(m.uri, store.RDFType, store.ClassMeeting)
		ts.Add(m.uri, store.PropMeetingDate, m.date.Format(time.RFC3339))
		ts.Add(m.uri, store.PropMeetingSequence, m.seq)
	}

	// Add Article 17 basic metadata
	ts.Add(art17Canonical, store.RDFType, store.ClassArticle)
	ts.Add(art17Canonical, store.PropTitle, "Right to erasure")
	ts.Add(art17Canonical, store.PropNumber, "17")

	// Add discussion events
	ts.Add(art17Canonical, store.PropDiscussedAt, "https://regula.dev/meetings/wg-40")
	ts.Add(art17Canonical, store.PropDiscussedAt, "https://regula.dev/meetings/wg-45")
	ts.Add(art17Canonical, store.PropDecidedAt, "https://regula.dev/meetings/wg-50")

	// Add an amendment motion
	amendmentURI := "https://regula.dev/amendments/art17-amendment-1"
	ts.Add(amendmentURI, store.RDFType, store.ClassMotion)
	ts.Add(amendmentURI, store.PropTargetProvision, art17Canonical)
	ts.Add(amendmentURI, store.PropMotionStatus, "adopted")
	ts.Add(amendmentURI, store.PropAdoptedDate, time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339))
	ts.Add(amendmentURI, store.PropProposedText, "Add clarification on scope")

	// Add another article for testing queries
	art18Canonical := "https://regula.dev/regulations/GDPR:Art18"
	ts.Add(art18Canonical, store.RDFType, store.ClassArticle)
	ts.Add(art18Canonical, store.PropTitle, "Right to restriction of processing")
	ts.Add(art18Canonical, store.PropNumber, "18")

	v18 := store.VersionInfo{
		URI:       art18Canonical + ":v1",
		Version:   "1.0",
		ValidFrom: time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC),
		Status:    "active",
	}
	ts.AddVersion(art18Canonical, v18)

	return ts
}

func TestNewTemporalQueryExecutor(t *testing.T) {
	ts := buildTemporalTestStore()
	executor := NewTemporalQueryExecutor(ts)

	if executor == nil {
		t.Fatal("Expected non-nil executor")
	}
	if executor.temporalStore != ts {
		t.Error("Expected same temporal store")
	}
	if executor.Executor == nil {
		t.Error("Expected non-nil base executor")
	}
}

func TestTemporalQueryExecutor_ExecuteAsOf(t *testing.T) {
	ts := buildTemporalTestStore()
	executor := NewTemporalQueryExecutor(ts)

	tests := []struct {
		name    string
		asOf    time.Time
		query   string
		wantErr bool
	}{
		{
			name:    "query at v1 time",
			asOf:    time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC),
			query:   `SELECT ?title WHERE { <https://regula.dev/regulations/GDPR:Art17> reg:title ?title }`,
			wantErr: false,
		},
		{
			name:    "query at v2 time",
			asOf:    time.Date(2023, 8, 1, 0, 0, 0, 0, time.UTC),
			query:   `SELECT ?title WHERE { <https://regula.dev/regulations/GDPR:Art17> reg:title ?title }`,
			wantErr: false,
		},
		{
			name:    "query at current time",
			asOf:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			query:   `SELECT ?title WHERE { <https://regula.dev/regulations/GDPR:Art17> reg:title ?title }`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.ExecuteAsOf(tt.query, tt.asOf)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteAsOf() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == nil {
					t.Fatal("Expected non-nil result")
				}
				if !result.AsOf.Equal(tt.asOf) {
					t.Errorf("Expected AsOf = %v, got %v", tt.asOf, result.AsOf)
				}
			}
		})
	}
}

func TestTemporalQueryExecutor_ExecuteAsOf_InvalidQuery(t *testing.T) {
	ts := buildTemporalTestStore()
	executor := NewTemporalQueryExecutor(ts)

	_, err := executor.ExecuteAsOf("INVALID QUERY", time.Now())
	if err == nil {
		t.Error("Expected error for invalid query")
	}
}

func TestTemporalQueryExecutor_ExecuteBetween(t *testing.T) {
	ts := buildTemporalTestStore()
	executor := NewTemporalQueryExecutor(ts)

	from := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2023, 7, 1, 0, 0, 0, 0, time.UTC)

	result, err := executor.ExecuteBetween(
		`SELECT ?article WHERE { ?article rdf:type reg:Article }`,
		from, to,
	)

	if err != nil {
		t.Fatalf("ExecuteBetween failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !result.FromTime.Equal(from) {
		t.Errorf("Expected FromTime = %v, got %v", from, result.FromTime)
	}
	if !result.ToTime.Equal(to) {
		t.Errorf("Expected ToTime = %v, got %v", to, result.ToTime)
	}
}

func TestTemporalQueryExecutor_ExecuteBetweenMeetings(t *testing.T) {
	ts := buildTemporalTestStore()
	executor := NewTemporalQueryExecutor(ts)

	result, err := executor.ExecuteBetweenMeetings(
		`SELECT ?article WHERE { ?article rdf:type reg:Article }`,
		"https://regula.dev/meetings/wg-40",
		"https://regula.dev/meetings/wg-50",
	)

	if err != nil {
		t.Fatalf("ExecuteBetweenMeetings failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestTemporalQueryExecutor_ExecuteBetweenMeetings_InvalidMeeting(t *testing.T) {
	ts := buildTemporalTestStore()
	executor := NewTemporalQueryExecutor(ts)

	_, err := executor.ExecuteBetweenMeetings(
		`SELECT ?x WHERE { ?x ?y ?z }`,
		"https://regula.dev/meetings/nonexistent",
		"https://regula.dev/meetings/wg-50",
	)

	if err == nil {
		t.Error("Expected error for non-existent meeting")
	}
}

func TestTemporalQueryExecutor_GetVersions(t *testing.T) {
	ts := buildTemporalTestStore()
	executor := NewTemporalQueryExecutor(ts)

	result, err := executor.GetVersions("https://regula.dev/regulations/GDPR:Art17")
	if err != nil {
		t.Fatalf("GetVersions failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Subject != "https://regula.dev/regulations/GDPR:Art17" {
		t.Errorf("Unexpected subject: %s", result.Subject)
	}

	if len(result.Versions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(result.Versions))
	}

	// Versions should be sorted by date
	for i := 1; i < len(result.Versions); i++ {
		if result.Versions[i].Date.Before(result.Versions[i-1].Date) {
			t.Error("Versions not sorted by date")
		}
	}

	// Current version should be set
	if result.CurrentVersion == nil {
		t.Error("Expected current version to be set")
	} else if result.CurrentVersion.Version != "3.0" {
		t.Errorf("Expected current version 3.0, got %s", result.CurrentVersion.Version)
	}
}

func TestTemporalQueryExecutor_GetVersions_NotFound(t *testing.T) {
	ts := buildTemporalTestStore()
	executor := NewTemporalQueryExecutor(ts)

	_, err := executor.GetVersions("https://regula.dev/regulations/nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent subject")
	}
}

func TestTemporalQueryExecutor_GetTimeline(t *testing.T) {
	ts := buildTemporalTestStore()
	executor := NewTemporalQueryExecutor(ts)

	timeline, err := executor.GetTimeline("https://regula.dev/regulations/GDPR:Art17")
	if err != nil {
		t.Fatalf("GetTimeline failed: %v", err)
	}

	if timeline == nil {
		t.Fatal("Expected non-nil timeline")
	}

	if timeline.Subject != "https://regula.dev/regulations/GDPR:Art17" {
		t.Errorf("Unexpected subject: %s", timeline.Subject)
	}

	// Should have events (versions + discussions + decisions)
	if len(timeline.Events) == 0 {
		t.Error("Expected at least one event")
	}

	// Events should be sorted by date
	for i := 1; i < len(timeline.Events); i++ {
		if timeline.Events[i].Date.Before(timeline.Events[i-1].Date) {
			t.Error("Events not sorted by date")
		}
	}

	// Duration should be calculated
	if timeline.FirstEvent.IsZero() {
		t.Error("Expected first event to be set")
	}
	if timeline.LastEvent.IsZero() {
		t.Error("Expected last event to be set")
	}
}

func TestTemporalQueryExecutor_GetDuration(t *testing.T) {
	ts := buildTemporalTestStore()
	executor := NewTemporalQueryExecutor(ts)

	result, err := executor.GetDuration("https://regula.dev/regulations/GDPR:Art17")
	if err != nil {
		t.Fatalf("GetDuration failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Subject != "https://regula.dev/regulations/GDPR:Art17" {
		t.Errorf("Unexpected subject: %s", result.Subject)
	}

	// Should have proposal date
	if result.ProposalDate.IsZero() {
		t.Error("Expected proposal date to be set")
	}

	// Should have adoption date
	if result.AdoptionDate.IsZero() {
		t.Error("Expected adoption date to be set")
	}

	// Duration should be calculated
	if result.TotalDuration == 0 {
		t.Error("Expected non-zero total duration")
	}

	// Should have discussion count
	if result.DiscussionCount < 1 {
		t.Errorf("Expected at least 1 discussion, got %d", result.DiscussionCount)
	}
}

func TestTemporalQueryExecutor_ProvisionsInForceAt(t *testing.T) {
	ts := buildTemporalTestStore()
	executor := NewTemporalQueryExecutor(ts)

	tests := []struct {
		name     string
		asOf     time.Time
		minCount int
	}{
		{
			name:     "mid 2023",
			asOf:     time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC),
			minCount: 1,
		},
		{
			name:     "current",
			asOf:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			minCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inForce, err := executor.ProvisionsInForceAt(tt.asOf)
			if err != nil {
				t.Fatalf("ProvisionsInForceAt failed: %v", err)
			}

			if len(inForce) < tt.minCount {
				t.Errorf("Expected at least %d provisions, got %d", tt.minCount, len(inForce))
			}
		})
	}
}

func TestTemporalQueryExecutor_AmendmentsAdoptedInRange(t *testing.T) {
	ts := buildTemporalTestStore()
	executor := NewTemporalQueryExecutor(ts)

	// Range that includes the amendment
	from := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)

	amendments, err := executor.AmendmentsAdoptedInRange(from, to)
	if err != nil {
		t.Fatalf("AmendmentsAdoptedInRange failed: %v", err)
	}

	if len(amendments) < 1 {
		t.Error("Expected at least 1 amendment")
	}

	// Verify the amendment is included
	found := false
	for _, a := range amendments {
		if a.Subject == "https://regula.dev/amendments/art17-amendment-1" {
			found = true
			if a.ChangeType != "amendment_adopted" {
				t.Errorf("Expected change type 'amendment_adopted', got %s", a.ChangeType)
			}
			break
		}
	}
	if !found {
		t.Error("Expected to find art17-amendment-1")
	}
}

func TestTemporalQueryExecutor_AmendmentsAdoptedInRange_Empty(t *testing.T) {
	ts := buildTemporalTestStore()
	executor := NewTemporalQueryExecutor(ts)

	// Range that excludes the amendment
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC)

	amendments, err := executor.AmendmentsAdoptedInRange(from, to)
	if err != nil {
		t.Fatalf("AmendmentsAdoptedInRange failed: %v", err)
	}

	if len(amendments) != 0 {
		t.Errorf("Expected 0 amendments, got %d", len(amendments))
	}
}

func TestTemporalQueryExecutor_AverageTimeToAdoption(t *testing.T) {
	ts := buildTemporalTestStore()
	executor := NewTemporalQueryExecutor(ts)

	avgDuration, count, err := executor.AverageTimeToAdoption(
		`SELECT ?article WHERE { ?article rdf:type reg:Article }`,
	)

	if err != nil {
		t.Fatalf("AverageTimeToAdoption failed: %v", err)
	}

	if count == 0 {
		t.Skip("No articles with adoption timeline found")
	}

	if avgDuration == 0 {
		t.Error("Expected non-zero average duration")
	}
}

func TestTimeline_Empty(t *testing.T) {
	timeline := &Timeline{
		Subject: "test",
		Events:  []TimelineEvent{},
	}

	if len(timeline.Events) != 0 {
		t.Error("Expected empty events")
	}
}

func TestRangeSummary(t *testing.T) {
	summary := RangeSummary{
		TotalChanges:     5,
		AddedCount:       2,
		ModifiedCount:    2,
		RemovedCount:     1,
		AffectedSubjects: 3,
	}

	if summary.TotalChanges != 5 {
		t.Errorf("Expected TotalChanges = 5, got %d", summary.TotalChanges)
	}
	if summary.AddedCount+summary.ModifiedCount+summary.RemovedCount != summary.TotalChanges {
		t.Error("Change counts don't add up to total")
	}
}

func TestVersionEntry(t *testing.T) {
	entry := VersionEntry{
		URI:     "https://example.org/doc:v1",
		Version: "1.0",
		Date:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:  "active",
		Text:    "Test content",
	}

	if entry.URI == "" {
		t.Error("Expected non-empty URI")
	}
	if entry.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", entry.Version)
	}
}

func TestTemporalResult(t *testing.T) {
	result := &TemporalResult{
		QueryResult: &QueryResult{
			Variables: []string{"x"},
			Bindings:  []map[string]string{{"x": "value"}},
			Count:     1,
		},
		AsOf: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	if result.Count != 1 {
		t.Errorf("Expected Count = 1, got %d", result.Count)
	}
	if result.AsOf.IsZero() {
		t.Error("Expected non-zero AsOf")
	}
}

func TestTemporalQueryExecutor_filterByValidity(t *testing.T) {
	ts := buildTemporalTestStore()
	executor := NewTemporalQueryExecutor(ts)

	bindings := []map[string]string{
		{"article": "https://regula.dev/regulations/GDPR:Art17:v1"},
		{"article": "https://regula.dev/regulations/GDPR:Art17:v2"},
		{"article": "https://regula.dev/regulations/GDPR:Art17:v3"},
	}

	// Filter at a time when v1 was valid
	filtered := executor.filterByValidity(bindings, time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC))

	// Should still have entries (no version info means assumed valid)
	if len(filtered) == 0 {
		t.Error("Expected at least some filtered results")
	}
}

func TestTemporalQueryExecutor_versionStatusToEventType(t *testing.T) {
	executor := &TemporalQueryExecutor{}

	tests := []struct {
		status   string
		expected string
	}{
		{"draft", "proposal"},
		{"active", "adoption"},
		{"superseded", "supersession"},
		{"withdrawn", "withdrawal"},
		{"unknown", "version_change"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := executor.versionStatusToEventType(tt.status)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRangeChange(t *testing.T) {
	change := RangeChange{
		Subject:     "https://example.org/article",
		ChangeType:  "modified",
		Date:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Description: "Updated text",
		OldVersion:  "v1",
		NewVersion:  "v2",
	}

	if change.Subject == "" {
		t.Error("Expected non-empty subject")
	}
	if change.ChangeType != "modified" {
		t.Errorf("Expected change type 'modified', got %s", change.ChangeType)
	}
}

func TestDurationResult(t *testing.T) {
	result := &DurationResult{
		Subject:         "https://example.org/article",
		ProposalDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		AdoptionDate:    time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
		TotalDuration:   time.Hour * 24 * 334, // approximately 11 months
		DiscussionCount: 5,
		AmendmentCount:  2,
	}

	if result.Subject == "" {
		t.Error("Expected non-empty subject")
	}
	if result.TotalDuration == 0 {
		t.Error("Expected non-zero duration")
	}
	if result.DiscussionCount != 5 {
		t.Errorf("Expected 5 discussions, got %d", result.DiscussionCount)
	}
}

func TestRangeQueryResult(t *testing.T) {
	result := &RangeQueryResult{
		FromTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		ToTime:   time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
		Changes: []RangeChange{
			{Subject: "a", ChangeType: "added"},
			{Subject: "b", ChangeType: "modified"},
		},
		Summary: RangeSummary{
			TotalChanges: 2,
			AddedCount:   1,
			ModifiedCount: 1,
		},
	}

	if len(result.Changes) != 2 {
		t.Errorf("Expected 2 changes, got %d", len(result.Changes))
	}
	if result.Summary.TotalChanges != 2 {
		t.Errorf("Expected 2 total changes, got %d", result.Summary.TotalChanges)
	}
}

func TestVersionsResult(t *testing.T) {
	v1 := VersionEntry{
		URI:     "doc:v1",
		Version: "1.0",
		Status:  "superseded",
	}
	v2 := VersionEntry{
		URI:     "doc:v2",
		Version: "2.0",
		Status:  "active",
	}

	result := &VersionsResult{
		Subject:        "doc",
		Versions:       []VersionEntry{v1, v2},
		CurrentVersion: &v2,
	}

	if len(result.Versions) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(result.Versions))
	}
	if result.CurrentVersion == nil {
		t.Error("Expected current version to be set")
	}
	if result.CurrentVersion.Version != "2.0" {
		t.Errorf("Expected current version 2.0, got %s", result.CurrentVersion.Version)
	}
}

func TestTemporalQueryExecutor_buildVersionsFromTriples(t *testing.T) {
	ts := store.NewTemporalStore()

	// Add version info via triples only (not using AddVersion)
	subject := "https://example.org/doc"
	v1URI := subject + ":v1"

	ts.Add(v1URI, store.PropVersionOf, subject)
	ts.Add(v1URI, store.PropVersionNumber, "1.0")
	ts.Add(v1URI, store.PropVersionStatus, "active")
	ts.Add(v1URI, store.PropValidFrom, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339))
	ts.Add(v1URI, store.PropText, "Version 1 content")

	executor := NewTemporalQueryExecutor(ts)
	result, err := executor.buildVersionsFromTriples(subject)

	if err != nil {
		t.Fatalf("buildVersionsFromTriples failed: %v", err)
	}

	if len(result.Versions) != 1 {
		t.Errorf("Expected 1 version, got %d", len(result.Versions))
	}

	if result.Versions[0].Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", result.Versions[0].Version)
	}

	if result.Versions[0].Text != "Version 1 content" {
		t.Errorf("Expected text 'Version 1 content', got %s", result.Versions[0].Text)
	}
}

func TestTemporalQueryExecutor_getMeetingDate(t *testing.T) {
	ts := buildTemporalTestStore()
	executor := NewTemporalQueryExecutor(ts)

	date, err := executor.getMeetingDate("https://regula.dev/meetings/wg-40")
	if err != nil {
		t.Fatalf("getMeetingDate failed: %v", err)
	}

	expected := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	if !date.Equal(expected) {
		t.Errorf("Expected date %v, got %v", expected, date)
	}
}

func TestTemporalQueryExecutor_getMeetingDate_NotFound(t *testing.T) {
	ts := buildTemporalTestStore()
	executor := NewTemporalQueryExecutor(ts)

	_, err := executor.getMeetingDate("https://regula.dev/meetings/nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent meeting")
	}
}

func TestTemporalQueryExecutor_findChangesInRange(t *testing.T) {
	ts := buildTemporalTestStore()
	executor := NewTemporalQueryExecutor(ts)

	subject := "https://regula.dev/regulations/GDPR:Art17"
	from := time.Date(2023, 5, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)

	changes := executor.findChangesInRange(subject, from, to)

	// Should find v2 and v3 within this range
	if len(changes) < 2 {
		t.Errorf("Expected at least 2 changes, got %d", len(changes))
	}

	// First change should be v2 (modified)
	if len(changes) > 0 && changes[0].ChangeType != "modified" {
		t.Errorf("Expected first change type 'modified', got %s", changes[0].ChangeType)
	}
}
