package deliberation

import (
	"testing"
	"time"
)

func TestMeetingStatus_String(t *testing.T) {
	tests := []struct {
		status   MeetingStatus
		expected string
	}{
		{MeetingScheduled, "scheduled"},
		{MeetingInProgress, "in_progress"},
		{MeetingCompleted, "completed"},
		{MeetingCancelled, "cancelled"},
		{MeetingPostponed, "postponed"},
		{MeetingStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("MeetingStatus.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestAgendaItemOutcome_String(t *testing.T) {
	tests := []struct {
		outcome  AgendaItemOutcome
		expected string
	}{
		{OutcomePending, "pending"},
		{OutcomeDiscussed, "discussed"},
		{OutcomeDeferred, "deferred"},
		{OutcomeDecided, "decided"},
		{OutcomeWithdrawn, "withdrawn"},
		{OutcomeNoQuorum, "no_quorum"},
		{AgendaItemOutcome(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.outcome.String(); got != tt.expected {
				t.Errorf("AgendaItemOutcome.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestMotionStatus_String(t *testing.T) {
	tests := []struct {
		status   MotionStatus
		expected string
	}{
		{MotionProposed, "proposed"},
		{MotionSeconded, "seconded"},
		{MotionDebated, "debated"},
		{MotionVoted, "voted"},
		{MotionAdopted, "adopted"},
		{MotionRejected, "rejected"},
		{MotionWithdrawn, "withdrawn"},
		{MotionTabled, "tabled"},
		{MotionStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("MotionStatus.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestVotePosition_String(t *testing.T) {
	tests := []struct {
		position VotePosition
		expected string
	}{
		{VoteFor, "for"},
		{VoteAgainst, "against"},
		{VoteAbstain, "abstain"},
		{VoteAbsent, "absent"},
		{VotePosition(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.position.String(); got != tt.expected {
				t.Errorf("VotePosition.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestInterventionPosition_String(t *testing.T) {
	tests := []struct {
		position InterventionPosition
		expected string
	}{
		{PositionNeutral, "neutral"},
		{PositionSupport, "support"},
		{PositionOppose, "oppose"},
		{PositionQualified, "qualified"},
		{PositionQuestion, "question"},
		{PositionProcedural, "procedural"},
		{InterventionPosition(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.position.String(); got != tt.expected {
				t.Errorf("InterventionPosition.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestActionItemStatus_String(t *testing.T) {
	tests := []struct {
		status   ActionItemStatus
		expected string
	}{
		{ActionPending, "pending"},
		{ActionInProgress, "in_progress"},
		{ActionCompleted, "completed"},
		{ActionDeferred, "deferred"},
		{ActionCancelled, "cancelled"},
		{ActionItemStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("ActionItemStatus.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewMeeting(t *testing.T) {
	date := time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC)
	meeting := NewMeeting(
		"https://example.org/meetings/2024-05",
		"WG-2024-05",
		"Working Group Meeting 5",
		date,
	)

	if meeting.URI != "https://example.org/meetings/2024-05" {
		t.Errorf("URI = %q, want %q", meeting.URI, "https://example.org/meetings/2024-05")
	}
	if meeting.Identifier != "WG-2024-05" {
		t.Errorf("Identifier = %q, want %q", meeting.Identifier, "WG-2024-05")
	}
	if meeting.Title != "Working Group Meeting 5" {
		t.Errorf("Title = %q, want %q", meeting.Title, "Working Group Meeting 5")
	}
	if !meeting.Date.Equal(date) {
		t.Errorf("Date = %v, want %v", meeting.Date, date)
	}
	if meeting.Status != MeetingScheduled {
		t.Errorf("Status = %v, want %v", meeting.Status, MeetingScheduled)
	}
}

func TestNewAgendaItem(t *testing.T) {
	item := NewAgendaItem(
		"https://example.org/agenda/item-1",
		"3",
		"Review of Article 5",
		"https://example.org/meetings/2024-05",
	)

	if item.URI != "https://example.org/agenda/item-1" {
		t.Errorf("URI = %q, want %q", item.URI, "https://example.org/agenda/item-1")
	}
	if item.Number != "3" {
		t.Errorf("Number = %q, want %q", item.Number, "3")
	}
	if item.Title != "Review of Article 5" {
		t.Errorf("Title = %q, want %q", item.Title, "Review of Article 5")
	}
	if item.MeetingURI != "https://example.org/meetings/2024-05" {
		t.Errorf("MeetingURI = %q, want %q", item.MeetingURI, "https://example.org/meetings/2024-05")
	}
	if item.Outcome != OutcomePending {
		t.Errorf("Outcome = %v, want %v", item.Outcome, OutcomePending)
	}
}

func TestNewMotion(t *testing.T) {
	motion := NewMotion(
		"https://example.org/motions/amend-1",
		"Amendment 1",
		"30-day retention limit",
		"Personal data shall be retained for no longer than 30 days.",
		"https://example.org/stakeholders/member-state-x",
		"https://example.org/meetings/2024-05",
	)

	if motion.URI != "https://example.org/motions/amend-1" {
		t.Errorf("URI = %q, want %q", motion.URI, "https://example.org/motions/amend-1")
	}
	if motion.Identifier != "Amendment 1" {
		t.Errorf("Identifier = %q, want %q", motion.Identifier, "Amendment 1")
	}
	if motion.Status != MotionProposed {
		t.Errorf("Status = %v, want %v", motion.Status, MotionProposed)
	}
}

func TestNewDecision(t *testing.T) {
	decidedAt := time.Date(2024, 5, 15, 14, 30, 0, 0, time.UTC)
	decision := NewDecision(
		"https://example.org/decisions/2024-05-01",
		"Decision 2024-05-01",
		"Adoption of Article 5",
		"Article 5 was adopted with amendments.",
		"https://example.org/meetings/2024-05",
		decidedAt,
	)

	if decision.URI != "https://example.org/decisions/2024-05-01" {
		t.Errorf("URI = %q, want %q", decision.URI, "https://example.org/decisions/2024-05-01")
	}
	if decision.Title != "Adoption of Article 5" {
		t.Errorf("Title = %q, want %q", decision.Title, "Adoption of Article 5")
	}
	if !decision.DecidedAt.Equal(decidedAt) {
		t.Errorf("DecidedAt = %v, want %v", decision.DecidedAt, decidedAt)
	}
}

func TestNewActionItem(t *testing.T) {
	action := NewActionItem(
		"https://example.org/actions/2024-05-01",
		"Action 2024-05-01",
		"Draft consolidated text for Article 5",
		[]string{"https://example.org/stakeholders/secretariat"},
		"https://example.org/meetings/2024-05",
	)

	if action.URI != "https://example.org/actions/2024-05-01" {
		t.Errorf("URI = %q, want %q", action.URI, "https://example.org/actions/2024-05-01")
	}
	if action.Description != "Draft consolidated text for Article 5" {
		t.Errorf("Description = %q, want %q", action.Description, "Draft consolidated text for Article 5")
	}
	if action.Status != ActionPending {
		t.Errorf("Status = %v, want %v", action.Status, ActionPending)
	}
	if len(action.AssignedToURIs) != 1 {
		t.Errorf("len(AssignedToURIs) = %d, want 1", len(action.AssignedToURIs))
	}
}

func TestMeeting_String(t *testing.T) {
	date := time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC)
	meeting := NewMeeting(
		"https://example.org/meetings/2024-05",
		"WG-2024-05",
		"Working Group Meeting 5",
		date,
	)

	expected := "Working Group Meeting 5 (WG-2024-05) - 2024-05-15"
	if got := meeting.String(); got != expected {
		t.Errorf("Meeting.String() = %q, want %q", got, expected)
	}
}

func TestActionItem_IsOverdue(t *testing.T) {
	yesterday := time.Now().Add(-24 * time.Hour)
	tomorrow := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name     string
		action   *ActionItem
		expected bool
	}{
		{
			name: "pending past due date",
			action: &ActionItem{
				Status:  ActionPending,
				DueDate: &yesterday,
			},
			expected: true,
		},
		{
			name: "pending future due date",
			action: &ActionItem{
				Status:  ActionPending,
				DueDate: &tomorrow,
			},
			expected: false,
		},
		{
			name: "completed past due date",
			action: &ActionItem{
				Status:  ActionCompleted,
				DueDate: &yesterday,
			},
			expected: false,
		},
		{
			name: "cancelled past due date",
			action: &ActionItem{
				Status:  ActionCancelled,
				DueDate: &yesterday,
			},
			expected: false,
		},
		{
			name: "pending no due date",
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
				t.Errorf("ActionItem.IsOverdue() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestActionItem_DaysSinceDue(t *testing.T) {
	threeDaysAgo := time.Now().Add(-72 * time.Hour)
	inThreeDays := time.Now().Add(72 * time.Hour)

	tests := []struct {
		name     string
		action   *ActionItem
		minDays  int
		maxDays  int
	}{
		{
			name: "three days overdue",
			action: &ActionItem{
				DueDate: &threeDaysAgo,
			},
			minDays: 2, // Allow for time zone edge cases
			maxDays: 4,
		},
		{
			name: "due in three days",
			action: &ActionItem{
				DueDate: &inThreeDays,
			},
			minDays: -4,
			maxDays: -2,
		},
		{
			name: "no due date",
			action: &ActionItem{
				DueDate: nil,
			},
			minDays: 0,
			maxDays: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			days := tt.action.DaysSinceDue()
			if days < tt.minDays || days > tt.maxDays {
				t.Errorf("ActionItem.DaysSinceDue() = %d, want between %d and %d",
					days, tt.minDays, tt.maxDays)
			}
		})
	}
}

func TestVoteRecord(t *testing.T) {
	voteDate := time.Date(2024, 5, 15, 14, 30, 0, 0, time.UTC)
	vote := &VoteRecord{
		URI:              "https://example.org/votes/2024-05-01",
		VoteDate:         voteDate,
		VoteType:         "roll_call",
		Question:         "Shall Amendment 1 be adopted?",
		Result:           "adopted",
		MajorityRequired: "simple",
		ForCount:         18,
		AgainstCount:     4,
		AbstainCount:     1,
		AbsentCount:      2,
		IndividualVotes: []IndividualVote{
			{
				VoterURI:  "https://example.org/stakeholders/member-x",
				VoterName: "Member State X",
				Position:  VoteFor,
			},
			{
				VoterURI:    "https://example.org/stakeholders/member-y",
				VoterName:   "Member State Y",
				Position:    VoteAgainst,
				Explanation: "Concerns about implementation timeline",
			},
		},
	}

	if vote.ForCount != 18 {
		t.Errorf("ForCount = %d, want 18", vote.ForCount)
	}
	if vote.AgainstCount != 4 {
		t.Errorf("AgainstCount = %d, want 4", vote.AgainstCount)
	}
	if len(vote.IndividualVotes) != 2 {
		t.Errorf("len(IndividualVotes) = %d, want 2", len(vote.IndividualVotes))
	}
	if vote.IndividualVotes[1].Explanation == "" {
		t.Error("Expected explanation for Member State Y vote")
	}
}

func TestIntervention(t *testing.T) {
	intervention := &Intervention{
		URI:             "https://example.org/interventions/2024-05-01",
		SpeakerURI:      "https://example.org/speakers/delegate-x",
		SpeakerName:     "Delegate X",
		AffiliationURI:  "https://example.org/stakeholders/member-state-x",
		AffiliationName: "Member State X",
		MeetingURI:      "https://example.org/meetings/2024-05",
		AgendaItemURI:   "https://example.org/agenda/item-1",
		Position:        PositionSupport,
		Summary:         "Expressed strong support for the 30-day retention limit",
		Sequence:        3,
	}

	if intervention.Position != PositionSupport {
		t.Errorf("Position = %v, want %v", intervention.Position, PositionSupport)
	}
	if intervention.Sequence != 3 {
		t.Errorf("Sequence = %d, want 3", intervention.Sequence)
	}
}

func TestStakeholder(t *testing.T) {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	stakeholder := &Stakeholder{
		URI:     "https://example.org/stakeholders/member-state-x",
		Name:    "Member State X",
		Type:    "member_state",
		Aliases: []string{"MSX", "the delegation from X"},
		Roles: []StakeholderRole{
			{
				Role:       "Chair",
				Scope:      "Working Group A",
				StartDate:  &startDate,
				ProcessURI: "https://example.org/processes/gdpr-review",
			},
		},
	}

	if stakeholder.Name != "Member State X" {
		t.Errorf("Name = %q, want %q", stakeholder.Name, "Member State X")
	}
	if len(stakeholder.Aliases) != 2 {
		t.Errorf("len(Aliases) = %d, want 2", len(stakeholder.Aliases))
	}
	if len(stakeholder.Roles) != 1 {
		t.Errorf("len(Roles) = %d, want 1", len(stakeholder.Roles))
	}
	if stakeholder.Roles[0].Role != "Chair" {
		t.Errorf("Roles[0].Role = %q, want %q", stakeholder.Roles[0].Role, "Chair")
	}
}

func TestDeliberationProcess(t *testing.T) {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	process := &DeliberationProcess{
		URI:         "https://example.org/processes/gdpr-review-2024",
		Identifier:  "GDPR-Review-2024",
		Title:       "GDPR Review Process 2024",
		Description: "Comprehensive review of GDPR provisions",
		Type:        "legislation",
		Status:      "active",
		StartDate:   &startDate,
		MeetingURIs: []string{
			"https://example.org/meetings/2024-01",
			"https://example.org/meetings/2024-02",
			"https://example.org/meetings/2024-03",
		},
		ProvisionURIs: []string{
			"https://regula.dev/regulations/GDPR:Art5",
			"https://regula.dev/regulations/GDPR:Art6",
		},
	}

	if process.Identifier != "GDPR-Review-2024" {
		t.Errorf("Identifier = %q, want %q", process.Identifier, "GDPR-Review-2024")
	}
	if process.Status != "active" {
		t.Errorf("Status = %q, want %q", process.Status, "active")
	}
	if len(process.MeetingURIs) != 3 {
		t.Errorf("len(MeetingURIs) = %d, want 3", len(process.MeetingURIs))
	}
	if len(process.ProvisionURIs) != 2 {
		t.Errorf("len(ProvisionURIs) = %d, want 2", len(process.ProvisionURIs))
	}
}

func TestActionNote(t *testing.T) {
	noteDate := time.Date(2024, 5, 22, 0, 0, 0, 0, time.UTC)
	newStatus := ActionInProgress
	note := ActionNote{
		MeetingURI:    "https://example.org/meetings/2024-06",
		Date:          noteDate,
		Note:          "Work has begun on drafting the consolidated text",
		UpdatedStatus: &newStatus,
	}

	if note.Note == "" {
		t.Error("Expected non-empty note")
	}
	if note.UpdatedStatus == nil {
		t.Error("Expected updated status")
	}
	if *note.UpdatedStatus != ActionInProgress {
		t.Errorf("UpdatedStatus = %v, want %v", *note.UpdatedStatus, ActionInProgress)
	}
}
