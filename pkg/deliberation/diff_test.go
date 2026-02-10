package deliberation

import (
	"strings"
	"testing"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// buildDiffTriples creates test data for diff testing.
func buildDiffTriples() *store.TripleStore {
	ts := store.NewTripleStore()

	// Meeting 5
	meeting5 := "reg:meeting5"
	ts.Add(meeting5, "rdf:type", store.ClassMeeting)
	ts.Add(meeting5, store.RDFSLabel, "Meeting 5")
	ts.Add(meeting5, store.PropMeetingDate, "2024-01-15")
	ts.Add(meeting5, store.PropMeetingSequence, "5")

	// Meeting 6
	meeting6 := "reg:meeting6"
	ts.Add(meeting6, "rdf:type", store.ClassMeeting)
	ts.Add(meeting6, store.RDFSLabel, "Meeting 6")
	ts.Add(meeting6, store.PropMeetingDate, "2024-01-22")
	ts.Add(meeting6, store.PropMeetingSequence, "6")

	// Agenda items for meeting 5
	// Using same URI for recurring agenda items to test status change detection
	agenda5a := "reg:agenda-article5"
	ts.Add(meeting5, store.PropHasAgendaItem, agenda5a)
	ts.Add(agenda5a, store.RDFSLabel, "Article 5 Discussion")
	ts.Add(agenda5a, store.PropAgendaItemOutcome, "discussed")

	agenda5b := "reg:agenda5b"
	ts.Add(meeting5, store.PropHasAgendaItem, agenda5b)
	ts.Add(agenda5b, store.RDFSLabel, "Article 7 Review")
	ts.Add(agenda5b, store.PropAgendaItemOutcome, "deferred")

	// Agenda items for meeting 6 - note agenda5a (Article 5) is carried over
	ts.Add(meeting6, store.PropHasAgendaItem, agenda5a)
	// Update outcome for this meeting (simulate change)

	agenda6b := "reg:agenda6b"
	ts.Add(meeting6, store.PropHasAgendaItem, agenda6b)
	ts.Add(agenda6b, store.RDFSLabel, "Article 14 Proposal")
	ts.Add(agenda6b, store.PropAgendaItemOutcome, "proposed")

	// Decision in meeting 6
	decision1 := "reg:decision1"
	ts.Add(decision1, "rdf:type", store.ClassDeliberationDecision)
	ts.Add(decision1, store.PropDecidedAt, meeting6)
	ts.Add(decision1, store.PropIdentifier, "Decision-2024-01")
	ts.Add(decision1, store.PropTitle, "Article 5(2) amendments adopted")
	ts.Add(decision1, store.PropText, "The committee adopts the proposed amendments to Article 5(2).")
	ts.Add(decision1, store.PropDecisionType, "adoption")
	ts.Add(decision1, store.PropAffectsProvision, "reg:article5")

	// Action items
	action1 := "reg:action1"
	ts.Add(action1, "rdf:type", store.ClassActionItem)
	ts.Add(action1, store.PropIdentifier, "Action-01")
	ts.Add(action1, store.PropText, "Secretariat to draft consolidated text")
	ts.Add(action1, store.PropActionAssignedAt, meeting6)
	ts.Add(action1, store.PropActionAssignedTo, "reg:secretariat")
	ts.Add(action1, store.PropActionStatus, "pending")

	action2 := "reg:action2"
	ts.Add(action2, "rdf:type", store.ClassActionItem)
	ts.Add(action2, store.PropIdentifier, "Action-00")
	ts.Add(action2, store.PropText, "Prepare amendment proposal")
	ts.Add(action2, store.PropActionAssignedAt, meeting5)
	ts.Add(action2, store.PropActionCompletedAt, meeting6)
	ts.Add(action2, store.PropActionAssignedTo, "reg:delegation_a")
	ts.Add(action2, store.PropActionStatus, "completed")

	// Provision with versions
	article5 := "reg:article5"
	ts.Add(article5, store.RDFSLabel, "Article 5")

	version1 := "reg:article5:v1"
	ts.Add(version1, store.PropVersionOf, article5)
	ts.Add(version1, store.PropText, "Personal data shall be:\nprocessed fairly\ncollected for specified purposes")
	ts.Add(version1, store.PropVersionNumber, "v1")

	version2 := "reg:article5:v2"
	ts.Add(version2, store.PropVersionOf, article5)
	ts.Add(version2, store.PropText, "Personal data shall be:\nprocessed lawfully, fairly and transparently\ncollected for specified purposes\nretained for no longer than necessary")
	ts.Add(version2, store.PropVersionNumber, "v2")
	ts.Add(version2, store.PropPreviousVersion, version1)
	ts.Add(version2, "reg:amendedAt", meeting6)
	ts.Add(version2, store.PropProposedBy, "reg:member_state_x")

	// Participants
	ts.Add(meeting5, store.PropParticipant, "reg:delegation_a")
	ts.Add(meeting5, store.PropParticipant, "reg:delegation_b")
	ts.Add(meeting5, store.PropParticipant, "reg:delegation_c")

	ts.Add(meeting6, store.PropParticipant, "reg:delegation_a")
	ts.Add(meeting6, store.PropParticipant, "reg:delegation_b")
	ts.Add(meeting6, store.PropParticipant, "reg:delegation_d")

	// Stakeholder labels
	ts.Add("reg:secretariat", store.RDFSLabel, "Secretariat")
	ts.Add("reg:delegation_a", store.RDFSLabel, "Delegation A")
	ts.Add("reg:delegation_b", store.RDFSLabel, "Delegation B")
	ts.Add("reg:delegation_c", store.RDFSLabel, "Delegation C")
	ts.Add("reg:delegation_d", store.RDFSLabel, "Delegation D")
	ts.Add("reg:member_state_x", store.RDFSLabel, "Member State X")

	return ts
}

func TestNewDiffBuilder(t *testing.T) {
	ts := store.NewTripleStore()
	builder := NewDiffBuilder(ts, "https://example.org/")

	if builder == nil {
		t.Fatal("expected non-nil DiffBuilder")
	}
	if builder.store != ts {
		t.Error("expected store to be set")
	}
	if builder.baseURI != "https://example.org/" {
		t.Errorf("expected baseURI 'https://example.org/', got %q", builder.baseURI)
	}
}

func TestDiffMeetings_NilStore(t *testing.T) {
	builder := &DiffBuilder{store: nil}
	_, err := builder.DiffMeetings("reg:meeting5", "reg:meeting6", DefaultDiffConfig())
	if err == nil {
		t.Error("expected error for nil store")
	}
}

func TestDiffMeetings_EmptyURIs(t *testing.T) {
	ts := store.NewTripleStore()
	builder := NewDiffBuilder(ts, "https://example.org/")

	_, err := builder.DiffMeetings("", "reg:meeting6", DefaultDiffConfig())
	if err == nil {
		t.Error("expected error for empty from URI")
	}

	_, err = builder.DiffMeetings("reg:meeting5", "", DefaultDiffConfig())
	if err == nil {
		t.Error("expected error for empty to URI")
	}
}

func TestDiffMeetings_TopicChanges(t *testing.T) {
	ts := buildDiffTriples()
	builder := NewDiffBuilder(ts, "https://example.org/")

	diff, err := builder.DiffMeetings("reg:meeting5", "reg:meeting6", DefaultDiffConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check anchors
	if diff.From.Label != "Meeting 5" {
		t.Errorf("expected From label 'Meeting 5', got %q", diff.From.Label)
	}
	if diff.To.Label != "Meeting 6" {
		t.Errorf("expected To label 'Meeting 6', got %q", diff.To.Label)
	}

	// Should have topic added (Article 14 Proposal)
	found := false
	for _, topic := range diff.TopicsAdded {
		if topic.TopicLabel == "Article 14 Proposal" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'Article 14 Proposal' in TopicsAdded")
	}

	// Should have topic removed (Article 7 Review)
	found = false
	for _, topic := range diff.TopicsRemoved {
		if topic.TopicLabel == "Article 7 Review" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'Article 7 Review' in TopicsRemoved")
	}

	// Note: Topic status changes require the same topic URI to appear in both meetings
	// with different status values. Since agenda items typically store a single status,
	// status change detection works when the RDF data includes per-meeting status triples.
	// The shared agenda item (Article 5 Discussion) appears in both meetings
	// but won't show as "changed" because it has the same status value in the store.
}

func TestDiffMeetings_DecisionChanges(t *testing.T) {
	ts := buildDiffTriples()
	builder := NewDiffBuilder(ts, "https://example.org/")

	diff, err := builder.DiffMeetings("reg:meeting5", "reg:meeting6", DefaultDiffConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have new decision
	if len(diff.DecisionsNew) == 0 {
		t.Fatal("expected at least one new decision")
	}

	found := false
	for _, dec := range diff.DecisionsNew {
		if dec.Decision.Title == "Article 5(2) amendments adopted" {
			found = true
			if dec.ChangeType != "new" {
				t.Errorf("expected change type 'new', got %q", dec.ChangeType)
			}
			break
		}
	}
	if !found {
		t.Error("expected 'Article 5(2) amendments adopted' decision")
	}
}

func TestDiffMeetings_ActionChanges(t *testing.T) {
	ts := buildDiffTriples()
	builder := NewDiffBuilder(ts, "https://example.org/")

	diff, err := builder.DiffMeetings("reg:meeting5", "reg:meeting6", DefaultDiffConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have new action
	foundNew := false
	for _, a := range diff.ActionsNew {
		if strings.Contains(a.Action.Description, "Secretariat to draft consolidated text") {
			foundNew = true
			break
		}
	}
	if !foundNew {
		t.Error("expected 'Secretariat to draft consolidated text' in ActionsNew")
	}

	// Should have completed action
	foundCompleted := false
	for _, a := range diff.ActionsCompleted {
		if strings.Contains(a.Action.Description, "Prepare amendment proposal") {
			foundCompleted = true
			break
		}
	}
	if !foundCompleted {
		t.Error("expected 'Prepare amendment proposal' in ActionsCompleted")
	}
}

func TestDiffMeetings_TextChanges(t *testing.T) {
	ts := buildDiffTriples()
	builder := NewDiffBuilder(ts, "https://example.org/")

	diff, err := builder.DiffMeetings("reg:meeting5", "reg:meeting6", DefaultDiffConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have text change for Article 5
	if len(diff.TextChanges) == 0 {
		t.Fatal("expected at least one text change")
	}

	found := false
	for _, tc := range diff.TextChanges {
		if tc.ProvisionLabel == "Article 5" {
			found = true

			// Check old and new text
			if !strings.Contains(tc.OldText, "processed fairly") {
				t.Error("expected old text to contain 'processed fairly'")
			}
			if !strings.Contains(tc.NewText, "processed lawfully, fairly and transparently") {
				t.Error("expected new text to contain 'processed lawfully, fairly and transparently'")
			}

			// Check diff lines
			hasRemoved := false
			hasAdded := false
			for _, line := range tc.DiffLines {
				if line.Type == DiffLineRemoved && strings.Contains(line.OldLine, "processed fairly") {
					hasRemoved = true
				}
				if line.Type == DiffLineAdded && strings.Contains(line.NewLine, "processed lawfully") {
					hasAdded = true
				}
			}
			if !hasRemoved {
				t.Error("expected removed line with 'processed fairly'")
			}
			if !hasAdded {
				t.Error("expected added line with 'processed lawfully'")
			}
			break
		}
	}
	if !found {
		t.Error("expected text change for 'Article 5'")
	}
}

func TestDiffMeetings_ParticipantChanges(t *testing.T) {
	ts := buildDiffTriples()
	builder := NewDiffBuilder(ts, "https://example.org/")

	diff, err := builder.DiffMeetings("reg:meeting5", "reg:meeting6", DefaultDiffConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check participant changes
	if len(diff.ParticipantChanges) == 0 {
		t.Fatal("expected participant changes")
	}

	// Delegation C should be marked as left (in meeting 5, not in meeting 6)
	// Delegation D should be marked as joined (not in meeting 5, in meeting 6)
	foundLeft := false
	foundJoined := false
	for _, p := range diff.ParticipantChanges {
		if p.Name == "Delegation C" && p.InFrom && !p.InTo {
			foundLeft = true
		}
		if p.Name == "Delegation D" && !p.InFrom && p.InTo {
			foundJoined = true
		}
	}
	if !foundLeft {
		t.Error("expected Delegation C to have left")
	}
	if !foundJoined {
		t.Error("expected Delegation D to have joined")
	}
}

func TestDiffMeetings_WithFilter(t *testing.T) {
	ts := buildDiffTriples()
	builder := NewDiffBuilder(ts, "https://example.org/")

	config := DiffConfig{
		IncludeTopics:       false,
		IncludeDecisions:    false,
		IncludeActions:      false,
		IncludeText:         true,
		IncludeParticipants: false,
		ProvisionFilter:     []string{"reg:nonexistent"}, // Filter out all provisions
	}

	diff, err := builder.DiffMeetings("reg:meeting5", "reg:meeting6", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have no text changes due to filter
	if len(diff.TextChanges) != 0 {
		t.Errorf("expected 0 text changes with filter, got %d", len(diff.TextChanges))
	}

	// Should have no topics/decisions/actions/participants
	if len(diff.TopicsAdded) != 0 || len(diff.TopicsRemoved) != 0 || len(diff.TopicsChanged) != 0 {
		t.Error("expected no topic changes when IncludeTopics is false")
	}
	if len(diff.DecisionsNew) != 0 || len(diff.DecisionsClosed) != 0 {
		t.Error("expected no decision changes when IncludeDecisions is false")
	}
}

func TestDiffProvisionVersions(t *testing.T) {
	ts := buildDiffTriples()
	builder := NewDiffBuilder(ts, "https://example.org/")

	diff, err := builder.DiffProvisionVersions("reg:article5:v1", "reg:article5:v2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if diff.ProvisionURI != "reg:article5" {
		t.Errorf("expected provision URI 'reg:article5', got %q", diff.ProvisionURI)
	}

	if diff.ProposedBy != "Member State X" {
		t.Errorf("expected proposed by 'Member State X', got %q", diff.ProposedBy)
	}

	// Check diff lines
	hasRemovedFairly := false
	hasAddedLawfully := false
	hasAddedRetained := false

	for _, line := range diff.DiffLines {
		if line.Type == DiffLineRemoved && strings.Contains(line.OldLine, "processed fairly") {
			hasRemovedFairly = true
		}
		if line.Type == DiffLineAdded && strings.Contains(line.NewLine, "lawfully") {
			hasAddedLawfully = true
		}
		if line.Type == DiffLineAdded && strings.Contains(line.NewLine, "retained") {
			hasAddedRetained = true
		}
	}

	if !hasRemovedFairly {
		t.Error("expected removed line with 'processed fairly'")
	}
	if !hasAddedLawfully {
		t.Error("expected added line with 'lawfully'")
	}
	if !hasAddedRetained {
		t.Error("expected added line with 'retained'")
	}
}

func TestDiffProvisionVersions_Errors(t *testing.T) {
	ts := store.NewTripleStore()
	builder := NewDiffBuilder(ts, "https://example.org/")

	// Nil store
	builder2 := &DiffBuilder{store: nil}
	_, err := builder2.DiffProvisionVersions("v1", "v2")
	if err == nil {
		t.Error("expected error for nil store")
	}

	// Empty URIs
	_, err = builder.DiffProvisionVersions("", "v2")
	if err == nil {
		t.Error("expected error for empty from URI")
	}

	_, err = builder.DiffProvisionVersions("v1", "")
	if err == nil {
		t.Error("expected error for empty to URI")
	}
}

func TestDiffSince(t *testing.T) {
	ts := buildDiffTriples()
	builder := NewDiffBuilder(ts, "https://example.org/")

	// Diff since before meeting 6
	since := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)
	diff, err := builder.DiffSince(since, DefaultDiffConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have anchor set
	if diff.From.Type != AnchorDate {
		t.Errorf("expected from anchor type AnchorDate, got %v", diff.From.Type)
	}

	// Should have found meeting 6's changes (after Jan 20)
	// The exact content depends on what's captured
	if diff.To.Label != "now" {
		t.Errorf("expected to anchor label 'now', got %q", diff.To.Label)
	}
}

func TestDiffSummary(t *testing.T) {
	ts := buildDiffTriples()
	builder := NewDiffBuilder(ts, "https://example.org/")

	diff, err := builder.DiffMeetings("reg:meeting5", "reg:meeting6", DefaultDiffConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	summary := diff.Summary()

	// Verify summary counts match actual data
	if summary.TopicsAdded != len(diff.TopicsAdded) {
		t.Errorf("summary TopicsAdded %d != actual %d", summary.TopicsAdded, len(diff.TopicsAdded))
	}
	if summary.TopicsRemoved != len(diff.TopicsRemoved) {
		t.Errorf("summary TopicsRemoved %d != actual %d", summary.TopicsRemoved, len(diff.TopicsRemoved))
	}
	if summary.DecisionsNew != len(diff.DecisionsNew) {
		t.Errorf("summary DecisionsNew %d != actual %d", summary.DecisionsNew, len(diff.DecisionsNew))
	}
	if summary.ActionsNew != len(diff.ActionsNew) {
		t.Errorf("summary ActionsNew %d != actual %d", summary.ActionsNew, len(diff.ActionsNew))
	}
	if summary.TextChanges != len(diff.TextChanges) {
		t.Errorf("summary TextChanges %d != actual %d", summary.TextChanges, len(diff.TextChanges))
	}

	// Check participant joined/left counts
	expectedJoined := 0
	expectedLeft := 0
	for _, p := range diff.ParticipantChanges {
		if p.InTo && !p.InFrom {
			expectedJoined++
		}
		if p.InFrom && !p.InTo {
			expectedLeft++
		}
	}
	if summary.ParticipantsJoined != expectedJoined {
		t.Errorf("summary ParticipantsJoined %d != expected %d", summary.ParticipantsJoined, expectedJoined)
	}
	if summary.ParticipantsLeft != expectedLeft {
		t.Errorf("summary ParticipantsLeft %d != expected %d", summary.ParticipantsLeft, expectedLeft)
	}
}

func TestDiffRenderUnified(t *testing.T) {
	ts := buildDiffTriples()
	builder := NewDiffBuilder(ts, "https://example.org/")

	diff, err := builder.DiffMeetings("reg:meeting5", "reg:meeting6", DefaultDiffConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := diff.RenderUnified()

	// Check header
	if !strings.Contains(output, "Diff: Meeting 5 → Meeting 6") {
		t.Error("expected header in unified output")
	}

	// Check topics section
	if !strings.Contains(output, "Topics:") {
		t.Error("expected Topics section")
	}
	if !strings.Contains(output, "+ Article 14 Proposal") {
		t.Error("expected added topic marker")
	}
	if !strings.Contains(output, "- Article 7 Review") {
		t.Error("expected removed topic marker")
	}

	// Check decisions section
	if !strings.Contains(output, "Decisions:") {
		t.Error("expected Decisions section")
	}

	// Check text changes
	if !strings.Contains(output, "Article 5 Text Changes:") {
		t.Error("expected Article 5 text changes")
	}
	if !strings.Contains(output, "+ processed lawfully") {
		t.Error("expected added line in text diff")
	}
	if !strings.Contains(output, "- processed fairly") {
		t.Error("expected removed line in text diff")
	}
}

func TestDiffRenderHTML(t *testing.T) {
	ts := buildDiffTriples()
	builder := NewDiffBuilder(ts, "https://example.org/")

	diff, err := builder.DiffMeetings("reg:meeting5", "reg:meeting6", DefaultDiffConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := diff.RenderHTML()

	// Check HTML structure
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("expected HTML doctype")
	}
	if !strings.Contains(output, "<title>Deliberation Diff</title>") {
		t.Error("expected title")
	}

	// Check CSS classes in style section
	if !strings.Contains(output, ".added {") {
		t.Error("expected .added CSS class in style")
	}
	if !strings.Contains(output, ".removed {") {
		t.Error("expected .removed CSS class in style")
	}

	// Check content
	if !strings.Contains(output, "Meeting 5") {
		t.Error("expected Meeting 5 in output")
	}
	if !strings.Contains(output, "Meeting 6") {
		t.Error("expected Meeting 6 in output")
	}

	// Check participant table
	if !strings.Contains(output, "participant-table") {
		t.Error("expected participant table")
	}
}

func TestDiffRenderJSON(t *testing.T) {
	ts := buildDiffTriples()
	builder := NewDiffBuilder(ts, "https://example.org/")

	diff, err := builder.DiffMeetings("reg:meeting5", "reg:meeting6", DefaultDiffConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jsonOutput, err := diff.RenderJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check JSON structure
	if !strings.Contains(jsonOutput, "\"from\"") {
		t.Error("expected 'from' field in JSON")
	}
	if !strings.Contains(jsonOutput, "\"to\"") {
		t.Error("expected 'to' field in JSON")
	}
	if !strings.Contains(jsonOutput, "\"topics_added\"") {
		t.Error("expected 'topics_added' field in JSON")
	}
	if !strings.Contains(jsonOutput, "\"decisions_new\"") {
		t.Error("expected 'decisions_new' field in JSON")
	}
}

func TestDiffRenderMarkdown(t *testing.T) {
	ts := buildDiffTriples()
	builder := NewDiffBuilder(ts, "https://example.org/")

	diff, err := builder.DiffMeetings("reg:meeting5", "reg:meeting6", DefaultDiffConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := diff.RenderMarkdown()

	// Check markdown structure
	if !strings.Contains(output, "# Diff: Meeting 5 → Meeting 6") {
		t.Error("expected H1 header")
	}
	if !strings.Contains(output, "## Summary") {
		t.Error("expected Summary section")
	}
	if !strings.Contains(output, "## Topics") {
		t.Error("expected Topics section")
	}

	// Check markdown diff block
	if !strings.Contains(output, "```diff") {
		t.Error("expected diff code block")
	}

	// Check table
	if !strings.Contains(output, "| Participant |") {
		t.Error("expected participant table header")
	}
}

func TestComputeLineDiff(t *testing.T) {
	tests := []struct {
		name        string
		oldText     string
		newText     string
		wantAdded   int
		wantRemoved int
	}{
		{
			name:        "empty to non-empty",
			oldText:     "",
			newText:     "line 1\nline 2",
			wantAdded:   2,
			wantRemoved: 1, // empty string splits to [""], which gets removed
		},
		{
			name:        "non-empty to empty",
			oldText:     "line 1\nline 2",
			newText:     "",
			wantAdded:   1, // empty string splits to [""], which gets added
			wantRemoved: 2,
		},
		{
			name:        "modification",
			oldText:     "line 1\nold line\nline 3",
			newText:     "line 1\nnew line\nline 3",
			wantAdded:   1,
			wantRemoved: 1,
		},
		{
			name:        "identical",
			oldText:     "line 1\nline 2",
			newText:     "line 1\nline 2",
			wantAdded:   0,
			wantRemoved: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffLines := computeLineDiff(tt.oldText, tt.newText)

			added := 0
			removed := 0
			for _, line := range diffLines {
				if line.Type == DiffLineAdded {
					added++
				}
				if line.Type == DiffLineRemoved {
					removed++
				}
			}

			if added != tt.wantAdded {
				t.Errorf("added lines: got %d, want %d", added, tt.wantAdded)
			}
			if removed != tt.wantRemoved {
				t.Errorf("removed lines: got %d, want %d", removed, tt.wantRemoved)
			}
		})
	}
}

func TestComputeLCS(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		wantLen  int
		wantLast string
	}{
		{
			name:     "identical",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "c"},
			wantLen:  3,
			wantLast: "c",
		},
		{
			name:     "partial match",
			a:        []string{"a", "b", "c", "d"},
			b:        []string{"a", "x", "c", "d"},
			wantLen:  3,
			wantLast: "d",
		},
		{
			name:    "no match",
			a:       []string{"a", "b", "c"},
			b:       []string{"x", "y", "z"},
			wantLen: 0,
		},
		{
			name:    "empty",
			a:       []string{},
			b:       []string{"a", "b"},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lcs := computeLCS(tt.a, tt.b)

			if len(lcs) != tt.wantLen {
				t.Errorf("LCS length: got %d, want %d", len(lcs), tt.wantLen)
			}

			if tt.wantLen > 0 && lcs[len(lcs)-1] != tt.wantLast {
				t.Errorf("LCS last element: got %q, want %q", lcs[len(lcs)-1], tt.wantLast)
			}
		})
	}
}

func TestDiffAnchorTypeString(t *testing.T) {
	tests := []struct {
		anchorType DiffAnchorType
		want       string
	}{
		{AnchorMeeting, "meeting"},
		{AnchorVersion, "version"},
		{AnchorDate, "date"},
		{DiffAnchorType(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.anchorType.String()
		if got != tt.want {
			t.Errorf("DiffAnchorType(%d).String() = %q, want %q", tt.anchorType, got, tt.want)
		}
	}
}

func TestTopicChangeTypeString(t *testing.T) {
	tests := []struct {
		changeType TopicChangeType
		want       string
	}{
		{TopicAdded, "added"},
		{TopicRemoved, "removed"},
		{TopicModified, "modified"},
		{TopicUnchanged, "unchanged"},
		{TopicChangeType(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.changeType.String()
		if got != tt.want {
			t.Errorf("TopicChangeType(%d).String() = %q, want %q", tt.changeType, got, tt.want)
		}
	}
}

func TestDiffLineTypeString(t *testing.T) {
	tests := []struct {
		lineType DiffLineType
		want     string
	}{
		{DiffLineUnchanged, "unchanged"},
		{DiffLineAdded, "added"},
		{DiffLineRemoved, "removed"},
		{DiffLineModified, "modified"},
		{DiffLineType(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.lineType.String()
		if got != tt.want {
			t.Errorf("DiffLineType(%d).String() = %q, want %q", tt.lineType, got, tt.want)
		}
	}
}

func TestDefaultDiffConfig(t *testing.T) {
	config := DefaultDiffConfig()

	if !config.IncludeTopics {
		t.Error("expected IncludeTopics to be true")
	}
	if !config.IncludeDecisions {
		t.Error("expected IncludeDecisions to be true")
	}
	if !config.IncludeActions {
		t.Error("expected IncludeActions to be true")
	}
	if !config.IncludeText {
		t.Error("expected IncludeText to be true")
	}
	if !config.IncludeParticipants {
		t.Error("expected IncludeParticipants to be true")
	}
}

func TestParseActionStatusString(t *testing.T) {
	tests := []struct {
		input string
		want  ActionItemStatus
	}{
		{"pending", ActionPending},
		{"PENDING", ActionPending},
		{"in_progress", ActionInProgress},
		{"IN_PROGRESS", ActionInProgress},
		{"completed", ActionCompleted},
		{"deferred", ActionDeferred},
		{"cancelled", ActionCancelled},
		{"unknown", ActionPending},
	}

	for _, tt := range tests {
		got := parseActionStatusString(tt.input)
		if got != tt.want {
			t.Errorf("parseActionStatusString(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
