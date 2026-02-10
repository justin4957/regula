package deliberation

import (
	"strings"
	"testing"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

func TestReportType_String(t *testing.T) {
	tests := []struct {
		typ      ReportType
		expected string
	}{
		{ReportTypeProgress, "progress"},
		{ReportTypeStatus, "status"},
		{ReportTypeDecisionLog, "decision_log"},
		{ReportTypeEvolution, "evolution"},
		{ReportTypeParticipation, "participation"},
		{ReportType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.typ.String(); got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestTopicPhase_String(t *testing.T) {
	tests := []struct {
		phase    TopicPhase
		expected string
	}{
		{TopicProposed, "proposed"},
		{TopicUnderDiscussion, "under_discussion"},
		{TopicAgreed, "agreed"},
		{TopicBlocked, "blocked"},
		{TopicClosed, "closed"},
		{TopicPhase(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.phase.String(); got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestNewReportGenerator(t *testing.T) {
	tripleStore := store.NewTripleStore()
	generator := NewReportGenerator(tripleStore, "https://example.org/")

	if generator == nil {
		t.Fatal("expected non-nil generator")
	}
	if generator.store != tripleStore {
		t.Error("expected store to be set")
	}
	if generator.baseURI != "https://example.org/" {
		t.Errorf("expected baseURI 'https://example.org/', got %s", generator.baseURI)
	}
	if generator.bottleneckDetector == nil {
		t.Error("expected bottleneck detector to be initialized")
	}
}

func buildReportTestStore() *store.TripleStore {
	ts := store.NewTripleStore()

	// Create meetings
	for i := 1; i <= 3; i++ {
		meetingURI := "meeting:" + string(rune('0'+i))
		ts.Add(meetingURI, store.RDFType, store.ClassMeeting)
		ts.Add(meetingURI, store.PropMeetingDate, time.Date(2024, time.Month(i), 15, 10, 0, 0, 0, time.UTC).Format(time.RFC3339))
		ts.Add(meetingURI, store.RDFSLabel, "Meeting "+string(rune('0'+i)))

		// Add agenda item
		aiURI := "agenda:" + string(rune('0'+i))
		ts.Add(meetingURI, store.PropHasAgendaItem, aiURI)
		ts.Add(aiURI, store.RDFType, store.ClassAgendaItem)
		ts.Add(aiURI, store.PropProvisionDiscussed, "provision:article5")
	}

	// Add provision
	ts.Add("provision:article5", store.RDFType, store.ClassArticle)
	ts.Add("provision:article5", store.RDFSLabel, "Article 5 - Data Minimization")

	// Add decision
	ts.Add("decision:1", store.RDFType, store.ClassDeliberationDecision)
	ts.Add("decision:1", store.RDFSLabel, "Article 5(1) adopted")
	ts.Add("decision:1", store.PropDecisionType, "adoption")
	ts.Add("decision:1", store.PropMeetingDate, time.Date(2024, 2, 15, 10, 0, 0, 0, time.UTC).Format(time.RFC3339))
	ts.Add("decision:1", store.PropAffectsProvision, "provision:article5")
	ts.Add("decision:1", store.PropPartOf, "meeting:2")

	// Add action item
	ts.Add("action:1", store.RDFType, store.ClassActionItem)
	ts.Add("action:1", store.RDFSLabel, "Review delegation report")
	ts.Add("action:1", store.PropActionStatus, "pending")
	ts.Add("action:1", store.PropActionDueDate, time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339))
	ts.Add("action:1", store.PropActionAssignedTo, "stakeholder:X")

	// Add intervention
	ts.Add("intervention:1", store.RDFType, store.ClassIntervention)
	ts.Add("intervention:1", store.PropSpeaker, "stakeholder:Germany")
	ts.Add("intervention:1", store.PropPartOf, "meeting:1")
	ts.Add("stakeholder:Germany", store.RDFSLabel, "Germany")

	// Add motion
	ts.Add("motion:1", store.RDFType, store.ClassMotion)
	ts.Add("motion:1", store.RDFSLabel, "Amendment to Article 5")
	ts.Add("motion:1", store.PropMotionStatus, "adopted")
	ts.Add("motion:1", store.PropTargetProvision, "provision:article5")
	ts.Add("motion:1", store.PropProposedBy, "stakeholder:Germany")
	ts.Add("motion:1", store.PropPartOf, "meeting:2")

	return ts
}

func TestGenerateReport_Progress(t *testing.T) {
	ts := buildReportTestStore()
	generator := NewReportGenerator(ts, "https://example.org/")

	config := ReportConfig{
		Type:  ReportTypeProgress,
		Title: "Progress Report",
		Period: &ReportPeriod{
			Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC),
		},
		Scope: ReportScope{
			IncludeActions: true,
		},
	}

	report, err := generator.GenerateReport(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report == nil {
		t.Fatal("expected non-nil report")
	}
	if report.Type != ReportTypeProgress {
		t.Errorf("expected progress report type, got %s", report.Type.String())
	}
	if report.Title != "Progress Report" {
		t.Errorf("expected title 'Progress Report', got '%s'", report.Title)
	}
	if report.Period == nil {
		t.Error("expected period to be set")
	}
}

func TestGenerateReport_Status(t *testing.T) {
	ts := buildReportTestStore()
	generator := NewReportGenerator(ts, "https://example.org/")

	config := ReportConfig{
		Type:  ReportTypeStatus,
		Title: "Status Report",
	}

	report, err := generator.GenerateReport(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Type != ReportTypeStatus {
		t.Errorf("expected status report type, got %s", report.Type.String())
	}
}

func TestGenerateReport_DecisionLog(t *testing.T) {
	ts := buildReportTestStore()
	generator := NewReportGenerator(ts, "https://example.org/")

	config := ReportConfig{
		Type:  ReportTypeDecisionLog,
		Title: "Decision Log",
	}

	report, err := generator.GenerateReport(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Type != ReportTypeDecisionLog {
		t.Errorf("expected decision_log report type, got %s", report.Type.String())
	}

	// Should have at least one decision
	if len(report.KeyDecisions) == 0 {
		t.Error("expected at least one decision")
	}
}

func TestGenerateReport_Evolution(t *testing.T) {
	ts := buildReportTestStore()
	generator := NewReportGenerator(ts, "https://example.org/")

	config := ReportConfig{
		Type:         ReportTypeEvolution,
		Title:        "Article 5 Evolution",
		ProvisionURI: "provision:article5",
	}

	report, err := generator.GenerateReport(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Type != ReportTypeEvolution {
		t.Errorf("expected evolution report type, got %s", report.Type.String())
	}

	// Should have evolution history
	if len(report.EvolutionHistory) == 0 {
		t.Error("expected evolution history entries")
	}
}

func TestGenerateReport_Participation(t *testing.T) {
	ts := buildReportTestStore()
	generator := NewReportGenerator(ts, "https://example.org/")

	config := ReportConfig{
		Type:  ReportTypeParticipation,
		Title: "Participation Report",
	}

	report, err := generator.GenerateReport(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Type != ReportTypeParticipation {
		t.Errorf("expected participation report type, got %s", report.Type.String())
	}

	if report.ParticipationStats == nil {
		t.Fatal("expected participation stats")
	}

	if report.ParticipationStats.TotalSpeakers == 0 {
		t.Error("expected at least one speaker")
	}
}

func TestGenerateReport_NoStore(t *testing.T) {
	generator := &ReportGenerator{
		baseURI: "https://example.org/",
	}

	config := ReportConfig{
		Type:  ReportTypeStatus,
		Title: "Test",
	}

	_, err := generator.GenerateReport(config)
	if err == nil {
		t.Error("expected error for nil store")
	}
}

func TestGenerateReport_ProgressRequiresPeriod(t *testing.T) {
	ts := store.NewTripleStore()
	generator := NewReportGenerator(ts, "https://example.org/")

	config := ReportConfig{
		Type:  ReportTypeProgress,
		Title: "Progress Report",
		// No period set
	}

	_, err := generator.GenerateReport(config)
	if err == nil {
		t.Error("expected error for progress report without period")
	}
}

func TestGenerateReport_EvolutionRequiresProvision(t *testing.T) {
	ts := store.NewTripleStore()
	generator := NewReportGenerator(ts, "https://example.org/")

	config := ReportConfig{
		Type:  ReportTypeEvolution,
		Title: "Evolution Report",
		// No provision URI set
	}

	_, err := generator.GenerateReport(config)
	if err == nil {
		t.Error("expected error for evolution report without provision URI")
	}
}

func TestGenerateReport_WithBottlenecks(t *testing.T) {
	ts := buildReportTestStore()
	generator := NewReportGenerator(ts, "https://example.org/")

	config := ReportConfig{
		Type:  ReportTypeStatus,
		Title: "Status Report with Bottlenecks",
		Scope: ReportScope{
			IncludeBottlenecks: true,
		},
	}

	report, err := generator.GenerateReport(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Bottlenecks should be included (may be empty if no issues detected)
	// Just verify the field is accessible
	_ = report.Bottlenecks
}

func TestSummarizeActions(t *testing.T) {
	ts := store.NewTripleStore()

	// Add completed action
	ts.Add("action:done", store.RDFType, store.ClassActionItem)
	ts.Add("action:done", store.PropActionStatus, "completed")

	// Add pending action
	ts.Add("action:pending", store.RDFType, store.ClassActionItem)
	ts.Add("action:pending", store.PropActionStatus, "pending")

	// Add overdue action
	ts.Add("action:overdue", store.RDFType, store.ClassActionItem)
	ts.Add("action:overdue", store.RDFSLabel, "Overdue task")
	ts.Add("action:overdue", store.PropActionStatus, "pending")
	ts.Add("action:overdue", store.PropActionDueDate, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339))

	generator := NewReportGenerator(ts, "https://example.org/")
	summary := generator.summarizeActions()

	if summary.Total != 3 {
		t.Errorf("expected 3 total, got %d", summary.Total)
	}
	if summary.Completed != 1 {
		t.Errorf("expected 1 completed, got %d", summary.Completed)
	}
	if summary.Pending != 2 {
		t.Errorf("expected 2 pending, got %d", summary.Pending)
	}
	if summary.Overdue != 1 {
		t.Errorf("expected 1 overdue, got %d", summary.Overdue)
	}
}

func TestComputeTopicStatus(t *testing.T) {
	ts := buildReportTestStore()
	generator := NewReportGenerator(ts, "https://example.org/")

	statuses := generator.computeTopicStatus(nil)

	if len(statuses) == 0 {
		t.Error("expected at least one topic status")
	}

	// Find article 5
	found := false
	for _, status := range statuses {
		if status.TopicURI == "provision:article5" {
			found = true
			if status.TopicLabel == "" {
				t.Error("expected topic label to be set")
			}
			break
		}
	}

	if !found {
		t.Error("expected to find article 5 status")
	}
}

func TestGetStatusEmoji(t *testing.T) {
	generator := NewReportGenerator(nil, "")

	tests := []struct {
		phase    TopicPhase
		expected string
	}{
		{TopicProposed, "📝"},
		{TopicUnderDiscussion, "💬"},
		{TopicAgreed, "✅"},
		{TopicBlocked, "🔴"},
		{TopicClosed, "⬛"},
	}

	for _, tt := range tests {
		result := generator.getStatusEmoji(tt.phase)
		if result != tt.expected {
			t.Errorf("phase %s: expected %s, got %s", tt.phase.String(), tt.expected, result)
		}
	}
}

func TestGenerateExecutiveSummary(t *testing.T) {
	generator := NewReportGenerator(nil, "")

	report := &ProcessReport{
		ParticipationStats: &ParticipationStats{
			TotalMeetings: 4,
		},
		KeyDecisions: []DecisionSummary{
			{Topic: "A"}, {Topic: "B"}, {Topic: "C"},
		},
		TopicStatus: []TopicStatus{
			{Status: TopicAgreed},
			{Status: TopicBlocked},
			{Status: TopicUnderDiscussion},
			{Status: TopicUnderDiscussion},
		},
		ActionSummary: &ReportActionSummary{
			Overdue: 2,
		},
	}

	summary := generator.generateExecutiveSummary(report)

	if !strings.Contains(summary, "4 meetings") {
		t.Error("expected meeting count in summary")
	}
	if !strings.Contains(summary, "3 decisions") {
		t.Error("expected decision count in summary")
	}
	if !strings.Contains(summary, "1 topic(s) agreed") {
		t.Error("expected agreed count in summary")
	}
	if !strings.Contains(summary, "1 topic(s) blocked") {
		t.Error("expected blocked count in summary")
	}
	if !strings.Contains(summary, "2 action item(s) overdue") {
		t.Error("expected overdue count in summary")
	}
}

func TestGenerateNextSteps(t *testing.T) {
	generator := NewReportGenerator(nil, "")

	report := &ProcessReport{
		TopicStatus: []TopicStatus{
			{TopicLabel: "Article 5", Status: TopicBlocked},
			{TopicLabel: "Article 6", Status: TopicUnderDiscussion},
		},
		ActionSummary: &ReportActionSummary{
			Overdue: 3,
		},
	}

	steps := generator.generateNextSteps(report)

	if len(steps) == 0 {
		t.Error("expected at least one next step")
	}

	hasBlockerStep := false
	hasOverdueStep := false
	for _, step := range steps {
		if strings.Contains(step, "Article 5") {
			hasBlockerStep = true
		}
		if strings.Contains(step, "overdue") {
			hasOverdueStep = true
		}
	}

	if !hasBlockerStep {
		t.Error("expected blocker resolution step")
	}
	if !hasOverdueStep {
		t.Error("expected overdue action step")
	}
}

func TestRenderMarkdown(t *testing.T) {
	ts := buildReportTestStore()
	generator := NewReportGenerator(ts, "https://example.org/")

	report := &ProcessReport{
		Type:        ReportTypeProgress,
		Title:       "Test Report",
		GeneratedAt: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
		Period: &ReportPeriod{
			Start: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
		},
		ExecutiveSummary: "This is the summary.",
		KeyDecisions: []DecisionSummary{
			{
				Date:     time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
				Topic:    "Article 5",
				Decision: "Adopted",
				Vote:     "18-3-2",
			},
		},
		TopicStatus: []TopicStatus{
			{
				TopicLabel:  "Article 5",
				Status:      TopicAgreed,
				StatusEmoji: "✅",
			},
		},
		ActionSummary: &ReportActionSummary{
			Total:     10,
			Completed: 7,
			Pending:   2,
			Overdue:   1,
		},
		NextSteps: []string{"Review pending items"},
	}

	markdown := generator.RenderMarkdown(report)

	if !strings.Contains(markdown, "# Test Report") {
		t.Error("expected title in markdown")
	}
	if !strings.Contains(markdown, "Executive Summary") {
		t.Error("expected executive summary section")
	}
	if !strings.Contains(markdown, "Key Decisions") {
		t.Error("expected key decisions section")
	}
	if !strings.Contains(markdown, "18-3-2") {
		t.Error("expected vote tally in markdown")
	}
	if !strings.Contains(markdown, "Topic Status") {
		t.Error("expected topic status section")
	}
	if !strings.Contains(markdown, "Action Items") {
		t.Error("expected action items section")
	}
	if !strings.Contains(markdown, "Next Steps") {
		t.Error("expected next steps section")
	}
}

func TestRenderHTML(t *testing.T) {
	ts := buildReportTestStore()
	generator := NewReportGenerator(ts, "https://example.org/")

	report := &ProcessReport{
		Type:             ReportTypeStatus,
		Title:            "HTML Test Report",
		GeneratedAt:      time.Now(),
		ExecutiveSummary: "Test summary",
		KeyDecisions: []DecisionSummary{
			{Topic: "Test", Decision: "Approved"},
		},
		TopicStatus: []TopicStatus{
			{TopicLabel: "Topic A", Status: TopicAgreed, StatusEmoji: "✅"},
		},
		NextSteps: []string{"Step 1", "Step 2"},
	}

	html, err := generator.RenderHTML(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(html, "<html>") {
		t.Error("expected HTML structure")
	}
	if !strings.Contains(html, "HTML Test Report") {
		t.Error("expected title in HTML")
	}
	if !strings.Contains(html, "Executive Summary") {
		t.Error("expected executive summary in HTML")
	}
	if !strings.Contains(html, "Key Decisions") {
		t.Error("expected key decisions in HTML")
	}
}

func TestRenderJSON(t *testing.T) {
	ts := buildReportTestStore()
	generator := NewReportGenerator(ts, "https://example.org/")

	report := &ProcessReport{
		Type:             ReportTypeStatus,
		Title:            "JSON Test Report",
		GeneratedAt:      time.Now(),
		ExecutiveSummary: "Test summary",
	}

	jsonStr, err := generator.RenderJSON(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(jsonStr, `"type"`) {
		t.Error("expected type field in JSON")
	}
	if !strings.Contains(jsonStr, `"title": "JSON Test Report"`) {
		t.Error("expected title in JSON")
	}
	if !strings.Contains(jsonStr, `"executive_summary": "Test summary"`) {
		t.Error("expected executive summary in JSON")
	}
}

func TestExtractURILabel(t *testing.T) {
	tests := []struct {
		uri      string
		expected string
	}{
		{"https://example.org/regulation#Article5", "Article5"},
		{"https://example.org/provision/Chapter3", "Chapter3"},
		{"SimpleLabel", "SimpleLabel"},
	}

	for _, tt := range tests {
		result := extractURILabel(tt.uri)
		if result != tt.expected {
			t.Errorf("uri '%s': expected '%s', got '%s'", tt.uri, tt.expected, result)
		}
	}
}

func TestTraceProvisionEvolution(t *testing.T) {
	ts := buildReportTestStore()
	generator := NewReportGenerator(ts, "https://example.org/")

	history := generator.traceProvisionEvolution("provision:article5")

	if len(history) == 0 {
		t.Error("expected evolution history entries")
	}

	// Check first entry
	entry := history[0]
	if entry.EventType == "" {
		t.Error("expected event type to be set")
	}
}

func TestComputeParticipationStats(t *testing.T) {
	ts := buildReportTestStore()
	generator := NewReportGenerator(ts, "https://example.org/")

	stats := generator.computeParticipationStats(nil)

	if stats == nil {
		t.Fatal("expected non-nil stats")
	}

	if stats.TotalSpeakers == 0 {
		t.Error("expected at least one speaker")
	}

	if len(stats.TopContributors) == 0 {
		t.Error("expected at least one contributor")
	}
}

func TestFormatVoteTally(t *testing.T) {
	ts := store.NewTripleStore()
	ts.Add("vote:1", store.PropVoteFor, "15")
	ts.Add("vote:1", store.PropVoteAgainst, "5")
	ts.Add("vote:1", store.PropVoteAbstain, "3")

	generator := NewReportGenerator(ts, "https://example.org/")
	tally := generator.formatVoteTally("vote:1")

	if tally != "15-5-3" {
		t.Errorf("expected '15-5-3', got '%s'", tally)
	}
}
