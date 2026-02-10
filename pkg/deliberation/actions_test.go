package deliberation

import (
	"strings"
	"testing"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

func TestNewActionExtractor(t *testing.T) {
	ts := store.NewTripleStore()
	extractor := NewActionExtractor(ts, "https://example.org")

	if extractor == nil {
		t.Fatal("Expected non-nil extractor")
	}
	if extractor.patterns == nil {
		t.Error("Expected non-nil patterns")
	}
	if extractor.store != ts {
		t.Error("Expected same triple store")
	}
}

func TestActionExtractor_ExtractActions_ExplicitMarkers(t *testing.T) {
	ts := store.NewTripleStore()
	extractor := NewActionExtractor(ts, "https://example.org")

	tests := []struct {
		name     string
		text     string
		wantText string
	}{
		{
			name:     "bracket action marker",
			text:     "[ACTION] John to review Article 5 by Friday",
			wantText: "John to review Article 5 by Friday",
		},
		{
			name:     "ACTION: prefix",
			text:     "ACTION: Submit revised proposal to secretariat",
			wantText: "Submit revised proposal to secretariat",
		},
		{
			name:     "numbered action",
			text:     "Action 1: Delegation X to prepare position paper",
			wantText: "Delegation X to prepare position paper",
		},
		{
			name:     "AI prefix",
			text:     "AI1: Review and comment on draft text",
			wantText: "Review and comment on draft text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions, err := extractor.ExtractActions(tt.text, "meeting:1")
			if err != nil {
				t.Fatalf("ExtractActions failed: %v", err)
			}

			if len(actions) == 0 {
				t.Fatal("Expected at least one action")
			}

			found := false
			for _, action := range actions {
				if strings.Contains(action.Text, tt.wantText) || tt.wantText == action.Text {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected action text containing %q, got actions: %v", tt.wantText, actions)
			}
		})
	}
}

func TestActionExtractor_ExtractActions_RequestPatterns(t *testing.T) {
	ts := store.NewTripleStore()
	extractor := NewActionExtractor(ts, "https://example.org")

	tests := []struct {
		name         string
		text         string
		wantAssignee string
	}{
		{
			name:         "was asked to",
			text:         "The Secretariat was asked to prepare a summary document.",
			wantAssignee: "Secretariat",
		},
		{
			name:         "was requested to",
			text:         "Germany was requested to submit amendments by next week.",
			wantAssignee: "Germany",
		},
		{
			name:         "invited to",
			text:         "Member States are invited to provide comments.",
			wantAssignee: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions, err := extractor.ExtractActions(tt.text, "meeting:1")
			if err != nil {
				t.Fatalf("ExtractActions failed: %v", err)
			}

			if len(actions) == 0 {
				t.Fatal("Expected at least one action")
			}

			if tt.wantAssignee != "" {
				found := false
				for _, action := range actions {
					for _, assignee := range action.Assignees {
						if strings.Contains(assignee, tt.wantAssignee) {
							found = true
							break
						}
					}
				}
				if !found {
					t.Errorf("Expected assignee containing %q", tt.wantAssignee)
				}
			}
		})
	}
}

func TestActionExtractor_ExtractActions_AgreementPatterns(t *testing.T) {
	ts := store.NewTripleStore()
	extractor := NewActionExtractor(ts, "https://example.org")

	text := "The committee agreed that the Secretariat would circulate the revised draft."
	actions, err := extractor.ExtractActions(text, "meeting:1")
	if err != nil {
		t.Fatalf("ExtractActions failed: %v", err)
	}

	if len(actions) == 0 {
		t.Fatal("Expected at least one action")
	}

	found := false
	for _, action := range actions {
		if action.MatchType == "agreement" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected an agreement-type action")
	}
}

func TestActionExtractor_ExtractActions_DueDate(t *testing.T) {
	ts := store.NewTripleStore()
	extractor := NewActionExtractor(ts, "https://example.org")

	tests := []struct {
		name        string
		text        string
		wantDueDate bool
	}{
		{
			name:        "by date format",
			text:        "[ACTION] Submit report by 15 January 2024",
			wantDueDate: true,
		},
		{
			name:        "by next meeting",
			text:        "[ACTION] Prepare analysis by the next meeting",
			wantDueDate: true,
		},
		{
			name:        "within weeks",
			text:        "[ACTION] Complete review within 2 weeks",
			wantDueDate: true,
		},
		{
			name:        "no due date",
			text:        "[ACTION] Review the proposal",
			wantDueDate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions, err := extractor.ExtractActions(tt.text, "meeting:1")
			if err != nil {
				t.Fatalf("ExtractActions failed: %v", err)
			}

			if len(actions) == 0 {
				t.Fatal("Expected at least one action")
			}

			hasDueDate := actions[0].DueDate != nil
			if hasDueDate != tt.wantDueDate {
				t.Errorf("DueDate presence: got %v, want %v", hasDueDate, tt.wantDueDate)
			}
		})
	}
}

func TestActionExtractor_ExtractActions_ProvisionReferences(t *testing.T) {
	ts := store.NewTripleStore()
	extractor := NewActionExtractor(ts, "https://example.org")

	text := "[ACTION] Review Article 17(2) and Section 3 for consistency"
	actions, err := extractor.ExtractActions(text, "meeting:1")
	if err != nil {
		t.Fatalf("ExtractActions failed: %v", err)
	}

	if len(actions) == 0 {
		t.Fatal("Expected at least one action")
	}

	if len(actions[0].RelatedProvisions) < 2 {
		t.Errorf("Expected at least 2 provision references, got %d", len(actions[0].RelatedProvisions))
	}
}

func TestActionExtractor_Priority(t *testing.T) {
	ts := store.NewTripleStore()
	extractor := NewActionExtractor(ts, "https://example.org")

	tests := []struct {
		name         string
		text         string
		wantPriority string
	}{
		{
			name:         "urgent",
			text:         "[ACTION] Urgently submit the revised proposal",
			wantPriority: "high",
		},
		{
			name:         "asap",
			text:         "[ACTION] Complete review ASAP",
			wantPriority: "high",
		},
		{
			name:         "critical",
			text:         "[ACTION] Critical: Fix the formatting issues",
			wantPriority: "high",
		},
		{
			name:         "when possible",
			text:         "[ACTION] Review when possible",
			wantPriority: "low",
		},
		{
			name:         "normal",
			text:         "[ACTION] Review the document",
			wantPriority: "medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions, err := extractor.ExtractActions(tt.text, "meeting:1")
			if err != nil {
				t.Fatalf("ExtractActions failed: %v", err)
			}

			if len(actions) == 0 {
				t.Fatal("Expected at least one action")
			}

			if actions[0].Priority != tt.wantPriority {
				t.Errorf("Priority: got %q, want %q", actions[0].Priority, tt.wantPriority)
			}
		})
	}
}

func TestActionExtractor_CreateActionItem(t *testing.T) {
	ts := store.NewTripleStore()
	extractor := NewActionExtractor(ts, "https://example.org")

	dueDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	extracted := ExtractedAction{
		Text:              "Review Article 5",
		Assignees:         []string{"Secretariat"},
		DueDate:           &dueDate,
		RelatedProvisions: []string{"Article 5"},
		Priority:          "high",
	}

	action := extractor.CreateActionItem(extracted, "meeting:1")

	if action == nil {
		t.Fatal("Expected non-nil action")
	}
	if action.Description != "Review Article 5" {
		t.Errorf("Description: got %q, want %q", action.Description, "Review Article 5")
	}
	if len(action.AssignedToURIs) != 1 || action.AssignedToURIs[0] != "Secretariat" {
		t.Errorf("AssignedToURIs: got %v", action.AssignedToURIs)
	}
	if action.DueDate == nil || !action.DueDate.Equal(dueDate) {
		t.Error("DueDate not set correctly")
	}
	if action.Priority != "high" {
		t.Errorf("Priority: got %q, want %q", action.Priority, "high")
	}
	if action.Status != ActionPending {
		t.Errorf("Status: got %v, want %v", action.Status, ActionPending)
	}
}

func TestNewActionTracker(t *testing.T) {
	ts := store.NewTripleStore()
	tracker := NewActionTracker(ts, "https://example.org")

	if tracker == nil {
		t.Fatal("Expected non-nil tracker")
	}
	if tracker.extractor == nil {
		t.Error("Expected non-nil extractor")
	}
	if tracker.actionIndex == nil {
		t.Error("Expected non-nil action index")
	}
}

func TestActionTracker_DetectStatusUpdate_Completion(t *testing.T) {
	ts := store.NewTripleStore()
	tracker := NewActionTracker(ts, "https://example.org")

	tests := []struct {
		name       string
		text       string
		wantStatus ActionItemStatus
	}{
		{
			name:       "action completed",
			text:       "Action 1 has been completed.",
			wantStatus: ActionCompleted,
		},
		{
			name:       "item closed",
			text:       "Item A1 closed.",
			wantStatus: ActionCompleted,
		},
		{
			name:       "report submitted",
			text:       "The Secretariat has submitted the requested report.",
			wantStatus: ActionCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updates := tracker.DetectStatusUpdate(tt.text, "meeting:2")

			if len(updates) == 0 {
				t.Fatal("Expected at least one status update")
			}

			if updates[0].NewStatus != tt.wantStatus {
				t.Errorf("NewStatus: got %v, want %v", updates[0].NewStatus, tt.wantStatus)
			}
		})
	}
}

func TestActionTracker_DetectStatusUpdate_Deferral(t *testing.T) {
	ts := store.NewTripleStore()
	tracker := NewActionTracker(ts, "https://example.org")

	text := "Action 2 was deferred to the next meeting."
	updates := tracker.DetectStatusUpdate(text, "meeting:2")

	if len(updates) == 0 {
		t.Fatal("Expected at least one status update")
	}

	if updates[0].NewStatus != ActionDeferred {
		t.Errorf("NewStatus: got %v, want %v", updates[0].NewStatus, ActionDeferred)
	}
}

func TestActionTracker_DetectStatusUpdate_Pending(t *testing.T) {
	ts := store.NewTripleStore()
	tracker := NewActionTracker(ts, "https://example.org")

	text := "Action 3 is still pending."
	updates := tracker.DetectStatusUpdate(text, "meeting:2")

	if len(updates) == 0 {
		t.Fatal("Expected at least one status update")
	}

	if updates[0].NewStatus != ActionPending {
		t.Errorf("NewStatus: got %v, want %v", updates[0].NewStatus, ActionPending)
	}
}

func TestActionTracker_ProcessMeeting(t *testing.T) {
	ts := store.NewTripleStore()
	tracker := NewActionTracker(ts, "https://example.org")

	meeting := NewMeeting("meeting:1", "WG-2024-01", "First Meeting", time.Now())

	minutesText := `
Meeting Minutes

1. Opening
The Chair opened the meeting.

2. Action Items
[ACTION] Secretariat to prepare summary document by next meeting.
[ACTION] France to submit position paper on Article 5.

3. Conclusion
The meeting was adjourned.
`

	report, err := tracker.ProcessMeeting(meeting, minutesText)
	if err != nil {
		t.Fatalf("ProcessMeeting failed: %v", err)
	}

	if report == nil {
		t.Fatal("Expected non-nil report")
	}

	if len(report.NewActions) < 2 {
		t.Errorf("Expected at least 2 new actions, got %d", len(report.NewActions))
	}

	// Verify actions were added to store
	actionTriples := ts.Find("", store.RDFType, store.ClassActionItem)
	if len(actionTriples) < 2 {
		t.Errorf("Expected at least 2 action triples, got %d", len(actionTriples))
	}
}

func TestActionTracker_ProcessMeeting_WithStatusUpdates(t *testing.T) {
	ts := store.NewTripleStore()
	tracker := NewActionTracker(ts, "https://example.org")

	// First meeting: create actions
	meeting1 := NewMeeting("meeting:1", "WG-2024-01", "First Meeting", time.Now().AddDate(0, -1, 0))
	minutes1 := "[ACTION] Secretariat to prepare report."
	_, _ = tracker.ProcessMeeting(meeting1, minutes1)

	// Second meeting: update status
	meeting2 := NewMeeting("meeting:2", "WG-2024-02", "Second Meeting", time.Now())
	minutes2 := "Action 1 has been completed. The report was circulated."
	report, err := tracker.ProcessMeeting(meeting2, minutes2)
	if err != nil {
		t.Fatalf("ProcessMeeting failed: %v", err)
	}

	if len(report.StatusUpdates) == 0 {
		t.Error("Expected at least one status update")
	}
}

func TestActionTracker_GetPendingActions(t *testing.T) {
	ts := store.NewTripleStore()
	tracker := NewActionTracker(ts, "https://example.org")

	// Create some actions
	meeting := NewMeeting("meeting:1", "WG-2024-01", "Meeting", time.Now())
	minutes := `
[ACTION] Action 1: pending task
[ACTION] Action 2: another pending task
`
	_, _ = tracker.ProcessMeeting(meeting, minutes)

	pending := tracker.GetPendingActions()
	if len(pending) < 2 {
		t.Errorf("Expected at least 2 pending actions, got %d", len(pending))
	}

	for _, action := range pending {
		if action.Status != ActionPending && action.Status != ActionInProgress {
			t.Errorf("Expected pending/in_progress status, got %v", action.Status)
		}
	}
}

func TestActionTracker_GetActionsByAssignee(t *testing.T) {
	ts := store.NewTripleStore()
	tracker := NewActionTracker(ts, "https://example.org")

	// Manually add action with specific assignee
	action := NewActionItem("action:1", "Action-1", "Test task", []string{"Secretariat"}, "meeting:1")
	tracker.actionIndex[action.URI] = action

	actions := tracker.GetActionsByAssignee("Secretariat")
	if len(actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(actions))
	}
}

func TestActionTracker_GetActionsByProvision(t *testing.T) {
	ts := store.NewTripleStore()
	tracker := NewActionTracker(ts, "https://example.org")

	// Manually add action with provision
	action := NewActionItem("action:1", "Action-1", "Review Article 5", []string{}, "meeting:1")
	action.RelatedProvisionURIs = []string{"provision:art5"}
	tracker.actionIndex[action.URI] = action

	actions := tracker.GetActionsByProvision("provision:art5")
	if len(actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(actions))
	}
}

func TestActionTracker_GetOverdueActions(t *testing.T) {
	ts := store.NewTripleStore()
	tracker := NewActionTracker(ts, "https://example.org")

	// Add overdue action
	pastDate := time.Now().AddDate(0, 0, -7)
	action := NewActionItem("action:1", "Action-1", "Overdue task", []string{}, "meeting:1")
	action.DueDate = &pastDate
	tracker.actionIndex[action.URI] = action

	// Add future action
	futureDate := time.Now().AddDate(0, 0, 7)
	action2 := NewActionItem("action:2", "Action-2", "Future task", []string{}, "meeting:1")
	action2.DueDate = &futureDate
	tracker.actionIndex[action2.URI] = action2

	overdue := tracker.GetOverdueActions()
	if len(overdue) != 1 {
		t.Errorf("Expected 1 overdue action, got %d", len(overdue))
	}
}

func TestActionTracker_GetSummary(t *testing.T) {
	ts := store.NewTripleStore()
	tracker := NewActionTracker(ts, "https://example.org")

	// Add various actions
	a1 := NewActionItem("action:1", "Action-1", "Pending", []string{"Assignee1"}, "meeting:1")
	a1.Status = ActionPending
	tracker.actionIndex[a1.URI] = a1

	a2 := NewActionItem("action:2", "Action-2", "Completed", []string{"Assignee2"}, "meeting:1")
	a2.Status = ActionCompleted
	tracker.actionIndex[a2.URI] = a2

	a3 := NewActionItem("action:3", "Action-3", "Deferred", []string{"Assignee1"}, "meeting:2")
	a3.Status = ActionDeferred
	tracker.actionIndex[a3.URI] = a3

	summary := tracker.GetSummary()

	if summary.Total != 3 {
		t.Errorf("Total: got %d, want 3", summary.Total)
	}
	if summary.Pending != 1 {
		t.Errorf("Pending: got %d, want 1", summary.Pending)
	}
	if summary.Completed != 1 {
		t.Errorf("Completed: got %d, want 1", summary.Completed)
	}
	if summary.Deferred != 1 {
		t.Errorf("Deferred: got %d, want 1", summary.Deferred)
	}
	if summary.ByAssignee["Assignee1"] != 2 {
		t.Errorf("ByAssignee[Assignee1]: got %d, want 2", summary.ByAssignee["Assignee1"])
	}
}

func TestActionTracker_LoadActionsFromStore(t *testing.T) {
	ts := store.NewTripleStore()

	// Pre-populate store
	actionURI := "https://example.org/actions/Action-1"
	ts.Add(actionURI, store.RDFType, store.ClassActionItem)
	ts.Add(actionURI, store.PropLabel, "Action-1")
	ts.Add(actionURI, store.PropText, "Test action description")
	ts.Add(actionURI, store.PropActionStatus, "pending")
	ts.Add(actionURI, store.PropActionAssignedAt, "meeting:1")
	ts.Add(actionURI, store.PropActionAssignedTo, "Secretariat")
	ts.Add(actionURI, store.PropActionPriority, "high")

	tracker := NewActionTracker(ts, "https://example.org")
	err := tracker.LoadActionsFromStore()
	if err != nil {
		t.Fatalf("LoadActionsFromStore failed: %v", err)
	}

	if len(tracker.actionIndex) != 1 {
		t.Errorf("Expected 1 action in index, got %d", len(tracker.actionIndex))
	}

	action, ok := tracker.actionIndex[actionURI]
	if !ok {
		t.Fatal("Action not found in index")
	}

	if action.Identifier != "Action-1" {
		t.Errorf("Identifier: got %q, want %q", action.Identifier, "Action-1")
	}
	if action.Description != "Test action description" {
		t.Errorf("Description: got %q", action.Description)
	}
	if action.Status != ActionPending {
		t.Errorf("Status: got %v, want %v", action.Status, ActionPending)
	}
	if action.Priority != "high" {
		t.Errorf("Priority: got %q, want %q", action.Priority, "high")
	}
}

func TestActionItem_IsOverdueMethod(t *testing.T) {
	tests := []struct {
		name     string
		action   *ActionItem
		expected bool
	}{
		{
			name: "overdue",
			action: &ActionItem{
				Status:  ActionPending,
				DueDate: func() *time.Time { t := time.Now().AddDate(0, 0, -7); return &t }(),
			},
			expected: true,
		},
		{
			name: "not yet due",
			action: &ActionItem{
				Status:  ActionPending,
				DueDate: func() *time.Time { t := time.Now().AddDate(0, 0, 7); return &t }(),
			},
			expected: false,
		},
		{
			name: "completed not overdue",
			action: &ActionItem{
				Status:  ActionCompleted,
				DueDate: func() *time.Time { t := time.Now().AddDate(0, 0, -7); return &t }(),
			},
			expected: false,
		},
		{
			name: "no due date",
			action: &ActionItem{
				Status:  ActionPending,
				DueDate: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.action.IsOverdue(); got != tt.expected {
				t.Errorf("IsOverdue: got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestActionItemStatus_StringMethod(t *testing.T) {
	tests := []struct {
		status   ActionItemStatus
		expected string
	}{
		{ActionPending, "pending"},
		{ActionInProgress, "in_progress"},
		{ActionCompleted, "completed"},
		{ActionDeferred, "deferred"},
		{ActionCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("String(): got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParseActionStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected ActionItemStatus
	}{
		{"pending", ActionPending},
		{"Pending", ActionPending},
		{"in_progress", ActionInProgress},
		{"completed", ActionCompleted},
		{"deferred", ActionDeferred},
		{"cancelled", ActionCancelled},
		{"unknown", ActionPending}, // defaults to pending
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := parseActionStatus(tt.input); got != tt.expected {
				t.Errorf("parseActionStatus(%q): got %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDeduplicateActions(t *testing.T) {
	extractor := &ActionExtractor{patterns: compileActionPatterns()}

	actions := []ExtractedAction{
		{Text: "Review Article 5", SourceOffset: 0},
		{Text: "Review Article 5 by Friday", SourceOffset: 10}, // Similar, close offset
		{Text: "Submit report", SourceOffset: 200},             // Different
	}

	deduplicated := extractor.deduplicateActions(actions)

	if len(deduplicated) != 2 {
		t.Errorf("Expected 2 deduplicated actions, got %d", len(deduplicated))
	}
}

func TestStringSimilarity(t *testing.T) {
	tests := []struct {
		a, b     string
		minScore float64
	}{
		{"hello world", "hello world", 1.0},
		{"hello world", "hello there", 0.4},
		{"completely different", "nothing similar", 0.0},
		{"review article 5", "REVIEW ARTICLE 5", 1.0}, // case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.a+" vs "+tt.b, func(t *testing.T) {
			score := stringSimilarity(tt.a, tt.b)
			if score < tt.minScore {
				t.Errorf("Similarity: got %v, want >= %v", score, tt.minScore)
			}
		})
	}
}

func TestMeetingActionReport(t *testing.T) {
	report := &MeetingActionReport{
		MeetingURI:  "meeting:1",
		MeetingDate: time.Now(),
		NewActions: []*ActionItem{
			NewActionItem("action:1", "Action-1", "Task 1", []string{}, "meeting:1"),
		},
		StatusUpdates: []ActionStatusUpdate{
			{ActionRef: "A1", NewStatus: ActionCompleted},
		},
		PendingActions: []*ActionItem{
			NewActionItem("action:2", "Action-2", "Task 2", []string{}, "meeting:1"),
		},
	}

	if len(report.NewActions) != 1 {
		t.Errorf("NewActions: got %d, want 1", len(report.NewActions))
	}
	if len(report.StatusUpdates) != 1 {
		t.Errorf("StatusUpdates: got %d, want 1", len(report.StatusUpdates))
	}
	if len(report.PendingActions) != 1 {
		t.Errorf("PendingActions: got %d, want 1", len(report.PendingActions))
	}
}

func TestExtractedAction(t *testing.T) {
	dueDate := time.Now().AddDate(0, 0, 7)
	action := ExtractedAction{
		Text:              "Review document",
		Assignees:         []string{"Secretariat", "Chair"},
		DueDate:           &dueDate,
		DueDateText:       "by next Friday",
		RelatedProvisions: []string{"Article 5", "Article 6"},
		Priority:          "high",
		SourceOffset:      100,
		MatchType:         "explicit",
	}

	if action.Text != "Review document" {
		t.Errorf("Text: got %q", action.Text)
	}
	if len(action.Assignees) != 2 {
		t.Errorf("Assignees: got %d, want 2", len(action.Assignees))
	}
	if action.Priority != "high" {
		t.Errorf("Priority: got %q, want %q", action.Priority, "high")
	}
}

func TestActionTracker_GetActionsForMeeting(t *testing.T) {
	ts := store.NewTripleStore()
	tracker := NewActionTracker(ts, "https://example.org")

	// Add actions for different meetings
	a1 := NewActionItem("action:1", "Action-1", "Task 1", []string{}, "meeting:1")
	a2 := NewActionItem("action:2", "Action-2", "Task 2", []string{}, "meeting:1")
	a3 := NewActionItem("action:3", "Action-3", "Task 3", []string{}, "meeting:2")

	tracker.actionIndex[a1.URI] = a1
	tracker.actionIndex[a2.URI] = a2
	tracker.actionIndex[a3.URI] = a3
	tracker.meetingActions["meeting:1"] = []string{a1.URI, a2.URI}
	tracker.meetingActions["meeting:2"] = []string{a3.URI}

	actions := tracker.GetActionsForMeeting("meeting:1")
	if len(actions) != 2 {
		t.Errorf("Expected 2 actions for meeting:1, got %d", len(actions))
	}

	actions = tracker.GetActionsForMeeting("meeting:2")
	if len(actions) != 1 {
		t.Errorf("Expected 1 action for meeting:2, got %d", len(actions))
	}
}

func TestActionNoteStruct(t *testing.T) {
	status := ActionInProgress
	note := ActionNote{
		MeetingURI:    "meeting:2",
		Date:          time.Now(),
		Note:          "Progress update: 50% complete",
		UpdatedStatus: &status,
	}

	if note.MeetingURI != "meeting:2" {
		t.Errorf("MeetingURI: got %q", note.MeetingURI)
	}
	if note.UpdatedStatus == nil || *note.UpdatedStatus != ActionInProgress {
		t.Error("UpdatedStatus not set correctly")
	}
}
