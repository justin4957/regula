package deliberation

import (
	"strings"
	"testing"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

func TestBottleneckType_String(t *testing.T) {
	tests := []struct {
		typ      BottleneckType
		expected string
	}{
		{BottleneckInactiveTopic, "inactive_topic"},
		{BottleneckRepeatedDeferral, "repeated_deferral"},
		{BottleneckOverdueAction, "overdue_action"},
		{BottleneckBlockedDecision, "blocked_decision"},
		{BottleneckMissingQuorum, "missing_quorum"},
		{BottleneckCircularDependency, "circular_dependency"},
		{BottleneckType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.typ.String(); got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestBottleneckSeverity_String(t *testing.T) {
	tests := []struct {
		sev      BottleneckSeverity
		expected string
	}{
		{BottleneckLow, "low"},
		{BottleneckMedium, "medium"},
		{BottleneckHigh, "high"},
		{BottleneckCritical, "critical"},
		{BottleneckSeverity(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.sev.String(); got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestNewBottleneckDetector(t *testing.T) {
	tripleStore := store.NewTripleStore()
	detector := NewBottleneckDetector(tripleStore, "https://example.org/")

	if detector == nil {
		t.Fatal("expected non-nil detector")
	}
	if detector.store != tripleStore {
		t.Error("expected store to be set")
	}
	if detector.baseURI != "https://example.org/" {
		t.Errorf("expected baseURI 'https://example.org/', got %s", detector.baseURI)
	}
	if detector.config.InactiveMeetings != 3 {
		t.Errorf("expected default InactiveMeetings 3, got %d", detector.config.InactiveMeetings)
	}
}

func TestBottleneckDetector_WithConfig(t *testing.T) {
	tripleStore := store.NewTripleStore()
	detector := NewBottleneckDetector(tripleStore, "https://example.org/")

	config := BottleneckConfig{
		InactiveMeetings: 5,
		MaxDeferrals:     4,
		OverdueDays:      14,
		Now:              time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
	}

	detector.WithConfig(config)

	if detector.config.InactiveMeetings != 5 {
		t.Errorf("expected InactiveMeetings 5, got %d", detector.config.InactiveMeetings)
	}
	if detector.config.MaxDeferrals != 4 {
		t.Errorf("expected MaxDeferrals 4, got %d", detector.config.MaxDeferrals)
	}
}

func TestDefaultBottleneckConfig(t *testing.T) {
	config := DefaultBottleneckConfig()

	if config.InactiveMeetings != 3 {
		t.Errorf("expected InactiveMeetings 3, got %d", config.InactiveMeetings)
	}
	if config.MaxDeferrals != 3 {
		t.Errorf("expected MaxDeferrals 3, got %d", config.MaxDeferrals)
	}
	if config.OverdueDays != 7 {
		t.Errorf("expected OverdueDays 7, got %d", config.OverdueDays)
	}
	if config.Now.IsZero() {
		t.Error("expected Now to be set")
	}
}

func buildBottleneckTestStore() *store.TripleStore {
	ts := store.NewTripleStore()

	// Create 5 meetings
	for i := 1; i <= 5; i++ {
		meetingURI := "meeting:" + string(rune('0'+i))
		ts.Add(meetingURI, store.RDFType, store.ClassMeeting)
		ts.Add(meetingURI, store.PropMeetingDate, time.Date(2024, time.Month(i), 15, 10, 0, 0, 0, time.UTC).Format(time.RFC3339))
		ts.Add(meetingURI, store.PropMeetingSequence, string(rune('0'+i)))
		ts.Add(meetingURI, store.RDFSLabel, "Meeting "+string(rune('0'+i)))
		ts.Add(meetingURI, store.PropMeetingStatus, "completed")

		// Add an agenda item for each meeting
		aiURI := "agenda:" + string(rune('0'+i))
		ts.Add(meetingURI, store.PropHasAgendaItem, aiURI)
		ts.Add(aiURI, store.RDFType, store.ClassAgendaItem)
		ts.Add(aiURI, store.RDFSLabel, "Agenda Item "+string(rune('0'+i)))
	}

	return ts
}

func TestDetectBottlenecks_EmptyStore(t *testing.T) {
	tripleStore := store.NewTripleStore()
	detector := NewBottleneckDetector(tripleStore, "https://example.org/")

	report, err := detector.DetectBottlenecks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report == nil {
		t.Fatal("expected non-nil report")
	}
	if len(report.Bottlenecks) != 0 {
		t.Errorf("expected 0 bottlenecks, got %d", len(report.Bottlenecks))
	}
}

func TestDetectBottlenecks_NoStore(t *testing.T) {
	detector := &BottleneckDetector{
		config:  DefaultBottleneckConfig(),
		baseURI: "https://example.org/",
	}

	_, err := detector.DetectBottlenecks()
	if err == nil {
		t.Fatal("expected error for nil store")
	}
}

func TestFindInactiveTopics(t *testing.T) {
	ts := buildBottleneckTestStore()

	// Add a topic discussed only in meeting 1
	ts.Add("agenda:1", store.PropProvisionDiscussed, "provision:old-topic")
	ts.Add("provision:old-topic", store.RDFSLabel, "Old Topic")

	// Add a topic discussed in meeting 5 (recent)
	ts.Add("agenda:5", store.PropProvisionDiscussed, "provision:recent-topic")
	ts.Add("provision:recent-topic", store.RDFSLabel, "Recent Topic")

	detector := NewBottleneckDetector(ts, "https://example.org/")
	report, err := detector.DetectBottlenecks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find old topic as inactive
	foundInactive := false
	for _, b := range report.Bottlenecks {
		if b.Type == BottleneckInactiveTopic {
			for _, item := range b.AffectedItems {
				if item == "provision:old-topic" {
					foundInactive = true
					break
				}
			}
		}
	}

	if !foundInactive {
		t.Error("expected to find old topic as inactive bottleneck")
	}
}

func TestFindRepeatedDeferrals(t *testing.T) {
	ts := buildBottleneckTestStore()

	// Add same item deferred in 3 meetings
	for i := 1; i <= 3; i++ {
		aiURI := "agenda:deferred:" + string(rune('0'+i))
		ts.Add("meeting:"+string(rune('0'+i)), store.PropHasAgendaItem, aiURI)
		ts.Add(aiURI, store.RDFType, store.ClassAgendaItem)
		ts.Add(aiURI, store.RDFSLabel, "Deferred Item")
		ts.Add(aiURI, store.PropAgendaItemOutcome, "deferred")
	}

	detector := NewBottleneckDetector(ts, "https://example.org/")
	report, err := detector.DetectBottlenecks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find repeated deferral
	foundDeferral := false
	for _, b := range report.Bottlenecks {
		if b.Type == BottleneckRepeatedDeferral {
			if strings.Contains(b.Description, "Deferred Item") {
				foundDeferral = true
				if b.MeetingCount != 3 {
					t.Errorf("expected 3 deferrals, got %d", b.MeetingCount)
				}
				break
			}
		}
	}

	if !foundDeferral {
		t.Error("expected to find repeated deferral bottleneck")
	}
}

func TestFindOverdueActions(t *testing.T) {
	ts := store.NewTripleStore()

	// Add overdue action (30 days overdue)
	ts.Add("action:overdue", store.RDFType, store.ClassActionItem)
	ts.Add("action:overdue", store.RDFSLabel, "Overdue Action")
	ts.Add("action:overdue", store.PropActionStatus, "pending")
	ts.Add("action:overdue", store.PropActionDueDate, time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339))
	ts.Add("action:overdue", store.PropActionAssignedTo, "stakeholder:X")

	// Add completed action (should not be flagged)
	ts.Add("action:done", store.RDFType, store.ClassActionItem)
	ts.Add("action:done", store.PropActionStatus, "completed")
	ts.Add("action:done", store.PropActionDueDate, time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339))

	detector := NewBottleneckDetector(ts, "https://example.org/")
	detector.config.Now = time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC) // 30 days after June 1

	report, err := detector.DetectBottlenecks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find overdue action
	foundOverdue := false
	for _, b := range report.Bottlenecks {
		if b.Type == BottleneckOverdueAction {
			for _, item := range b.AffectedItems {
				if item == "action:overdue" {
					foundOverdue = true
					// Check severity (30 days = high)
					if b.Severity != BottleneckHigh {
						t.Errorf("expected HIGH severity for 30 days overdue, got %s", b.Severity.String())
					}
					break
				}
			}
		}
	}

	if !foundOverdue {
		t.Error("expected to find overdue action bottleneck")
	}

	// Completed action should not be flagged
	for _, b := range report.Bottlenecks {
		for _, item := range b.AffectedItems {
			if item == "action:done" {
				t.Error("completed action should not be flagged")
			}
		}
	}
}

func TestFindBlockedDecisions(t *testing.T) {
	ts := store.NewTripleStore()

	// Motion A is pending
	ts.Add("motion:A", store.RDFType, store.ClassMotion)
	ts.Add("motion:A", store.RDFSLabel, "Motion A")
	ts.Add("motion:A", store.PropMotionStatus, "pending")

	// Motion B is pending and references Motion A
	ts.Add("motion:B", store.RDFType, store.ClassMotion)
	ts.Add("motion:B", store.RDFSLabel, "Motion B")
	ts.Add("motion:B", store.PropMotionStatus, "pending")
	ts.Add("motion:B", store.PropReferences, "motion:A")

	// Motion C is adopted (should not block)
	ts.Add("motion:C", store.RDFType, store.ClassMotion)
	ts.Add("motion:C", store.PropMotionStatus, "adopted")

	detector := NewBottleneckDetector(ts, "https://example.org/")
	report, err := detector.DetectBottlenecks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find Motion B blocked by Motion A
	foundBlocked := false
	for _, b := range report.Bottlenecks {
		if b.Type == BottleneckBlockedDecision {
			for _, item := range b.AffectedItems {
				if item == "motion:B" {
					foundBlocked = true
					if len(b.BlockedBy) != 1 || b.BlockedBy[0] != "motion:A" {
						t.Errorf("expected blocked by motion:A, got %v", b.BlockedBy)
					}
					break
				}
			}
		}
	}

	if !foundBlocked {
		t.Error("expected to find blocked decision bottleneck")
	}
}

func TestFindMissingQuorum(t *testing.T) {
	ts := buildBottleneckTestStore()

	// Add agenda item with no_quorum outcome
	ts.Add("agenda:quorum", store.RDFType, store.ClassAgendaItem)
	ts.Add("agenda:quorum", store.RDFSLabel, "Article 5 Vote")
	ts.Add("agenda:quorum", store.PropAgendaItemOutcome, "no_quorum")
	ts.Add("meeting:3", store.PropHasAgendaItem, "agenda:quorum")

	detector := NewBottleneckDetector(ts, "https://example.org/")
	report, err := detector.DetectBottlenecks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find quorum issue
	foundQuorum := false
	for _, b := range report.Bottlenecks {
		if b.Type == BottleneckMissingQuorum {
			for _, item := range b.AffectedItems {
				if item == "agenda:quorum" {
					foundQuorum = true
					break
				}
			}
		}
	}

	if !foundQuorum {
		t.Error("expected to find missing quorum bottleneck")
	}
}

func TestFindCircularDependencies(t *testing.T) {
	ts := store.NewTripleStore()

	// Create circular reference: A -> B -> C -> A
	ts.Add("motion:A", store.RDFType, store.ClassMotion)
	ts.Add("motion:A", store.PropReferences, "motion:B")

	ts.Add("motion:B", store.RDFType, store.ClassMotion)
	ts.Add("motion:B", store.PropReferences, "motion:C")

	ts.Add("motion:C", store.RDFType, store.ClassMotion)
	ts.Add("motion:C", store.PropReferences, "motion:A")

	detector := NewBottleneckDetector(ts, "https://example.org/")
	report, err := detector.DetectBottlenecks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find circular dependency
	foundCircular := false
	for _, b := range report.Bottlenecks {
		if b.Type == BottleneckCircularDependency {
			foundCircular = true
			if b.Severity != BottleneckCritical {
				t.Errorf("expected CRITICAL severity, got %s", b.Severity.String())
			}
			break
		}
	}

	if !foundCircular {
		t.Error("expected to find circular dependency bottleneck")
	}
}

func TestCalculateSeverity(t *testing.T) {
	detector := NewBottleneckDetector(nil, "")

	tests := []struct {
		meetingCount int
		expected     BottleneckSeverity
	}{
		{1, BottleneckLow},
		{2, BottleneckLow},
		{3, BottleneckMedium},
		{4, BottleneckMedium},
		{5, BottleneckHigh},
		{10, BottleneckHigh},
	}

	for _, tt := range tests {
		result := detector.calculateSeverity(tt.meetingCount)
		if result != tt.expected {
			t.Errorf("meetingCount %d: expected %s, got %s", tt.meetingCount, tt.expected.String(), result.String())
		}
	}
}

func TestCalculateOverdueSeverity(t *testing.T) {
	detector := NewBottleneckDetector(nil, "")

	tests := []struct {
		daysOverdue int
		expected    BottleneckSeverity
	}{
		{7, BottleneckLow},
		{14, BottleneckMedium},
		{30, BottleneckHigh},
		{60, BottleneckCritical},
		{100, BottleneckCritical},
	}

	for _, tt := range tests {
		result := detector.calculateOverdueSeverity(tt.daysOverdue)
		if result != tt.expected {
			t.Errorf("daysOverdue %d: expected %s, got %s", tt.daysOverdue, tt.expected.String(), result.String())
		}
	}
}

func TestSortBySeverity(t *testing.T) {
	detector := NewBottleneckDetector(nil, "")

	bottlenecks := []Bottleneck{
		{Severity: BottleneckLow, Description: "Low"},
		{Severity: BottleneckCritical, Description: "Critical"},
		{Severity: BottleneckMedium, Description: "Medium"},
		{Severity: BottleneckHigh, Description: "High"},
	}

	detector.sortBySeverity(bottlenecks)

	expected := []BottleneckSeverity{BottleneckCritical, BottleneckHigh, BottleneckMedium, BottleneckLow}
	for i, b := range bottlenecks {
		if b.Severity != expected[i] {
			t.Errorf("position %d: expected %s, got %s", i, expected[i].String(), b.Severity.String())
		}
	}
}

func TestGenerateSummary(t *testing.T) {
	detector := NewBottleneckDetector(nil, "")

	stalledTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	bottlenecks := []Bottleneck{
		{Severity: BottleneckCritical, Type: BottleneckCircularDependency, AffectedItems: []string{"A"}, StalledSince: stalledTime},
		{Severity: BottleneckHigh, Type: BottleneckOverdueAction, AffectedItems: []string{"B"}, StalledSince: stalledTime.Add(24 * time.Hour)},
		{Severity: BottleneckHigh, Type: BottleneckBlockedDecision, AffectedItems: []string{"A", "C"}},
		{Severity: BottleneckMedium, Type: BottleneckRepeatedDeferral, AffectedItems: []string{"D"}},
		{Severity: BottleneckLow, Type: BottleneckInactiveTopic, AffectedItems: []string{"E"}},
	}

	summary := detector.generateSummary(bottlenecks)

	if summary.TotalBottlenecks != 5 {
		t.Errorf("expected 5 total, got %d", summary.TotalBottlenecks)
	}
	if summary.CriticalCount != 1 {
		t.Errorf("expected 1 critical, got %d", summary.CriticalCount)
	}
	if summary.HighCount != 2 {
		t.Errorf("expected 2 high, got %d", summary.HighCount)
	}
	if summary.MediumCount != 1 {
		t.Errorf("expected 1 medium, got %d", summary.MediumCount)
	}
	if summary.LowCount != 1 {
		t.Errorf("expected 1 low, got %d", summary.LowCount)
	}
	if summary.ByType[BottleneckCircularDependency] != 1 {
		t.Error("expected 1 circular dependency in ByType")
	}
	if summary.OldestStall == nil || !summary.OldestStall.Equal(stalledTime) {
		t.Error("expected oldest stall to be set correctly")
	}
	// "A" appears in 2 bottlenecks, should be first in MostAffected
	if len(summary.MostAffected) == 0 || summary.MostAffected[0] != "A" {
		t.Errorf("expected 'A' as most affected, got %v", summary.MostAffected)
	}
}

func TestGetBottlenecksForProvision(t *testing.T) {
	ts := store.NewTripleStore()

	// Create some bottleneck-triggering data
	ts.Add("action:1", store.RDFType, store.ClassActionItem)
	ts.Add("action:1", store.RDFSLabel, "Action for Provision X")
	ts.Add("action:1", store.PropActionStatus, "pending")
	ts.Add("action:1", store.PropActionDueDate, time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339))

	detector := NewBottleneckDetector(ts, "https://example.org/")
	detector.config.Now = time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)

	bottlenecks, err := detector.GetBottlenecksForProvision("action:1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(bottlenecks) == 0 {
		t.Error("expected to find bottleneck for provision")
	}
}

func TestGenerateReport(t *testing.T) {
	detector := NewBottleneckDetector(nil, "")

	report := &BottleneckReport{
		AnalyzedAt: time.Date(2024, 7, 1, 10, 30, 0, 0, time.UTC),
		Bottlenecks: []Bottleneck{
			{
				Type:          BottleneckCircularDependency,
				Severity:      BottleneckCritical,
				Description:   "Critical issue",
				StalledSince:  time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
				MeetingCount:  5,
				BlockedBy:     []string{"blocker:1"},
				Suggestions:   []string{"Fix it immediately"},
			},
			{
				Type:        BottleneckInactiveTopic,
				Severity:    BottleneckLow,
				Description: "Minor issue",
			},
		},
		Summary: BottleneckSummary{
			TotalBottlenecks: 2,
			CriticalCount:    1,
			LowCount:         1,
		},
	}

	output := detector.GenerateReport(report)

	if !strings.Contains(output, "Deliberation Bottleneck Analysis") {
		t.Error("expected header in report")
	}
	if !strings.Contains(output, "CRITICAL (1)") {
		t.Error("expected CRITICAL section in report")
	}
	if !strings.Contains(output, "Critical issue") {
		t.Error("expected critical issue description in report")
	}
	if !strings.Contains(output, "Stalled since") {
		t.Error("expected stalled since in report")
	}
	if !strings.Contains(output, "Fix it immediately") {
		t.Error("expected suggestion in report")
	}
	if !strings.Contains(output, "Total: 2 bottlenecks") {
		t.Error("expected summary in report")
	}
}

func TestParseMeetingStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected MeetingStatus
	}{
		{"scheduled", MeetingScheduled},
		{"in_progress", MeetingInProgress},
		{"completed", MeetingCompleted},
		{"cancelled", MeetingCancelled},
		{"postponed", MeetingPostponed},
		{"unknown", MeetingScheduled},
	}

	for _, tt := range tests {
		result := parseMeetingStatus(tt.input)
		if result != tt.expected {
			t.Errorf("input %s: expected %d, got %d", tt.input, tt.expected, result)
		}
	}
}

func TestBottleneck_Suggestions(t *testing.T) {
	ts := store.NewTripleStore()

	// Add overdue action
	ts.Add("action:test", store.RDFType, store.ClassActionItem)
	ts.Add("action:test", store.PropActionStatus, "pending")
	ts.Add("action:test", store.PropActionDueDate, time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339))

	detector := NewBottleneckDetector(ts, "https://example.org/")
	detector.config.Now = time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)

	report, err := detector.DetectBottlenecks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that suggestions are provided
	for _, b := range report.Bottlenecks {
		if b.Type == BottleneckOverdueAction {
			if len(b.Suggestions) == 0 {
				t.Error("expected suggestions for overdue action")
			}
			break
		}
	}
}
