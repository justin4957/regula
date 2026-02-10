package deliberation

import (
	"strings"
	"testing"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// buildProvenanceTriples creates test data for provenance chain testing.
func buildProvenanceTriples() *store.TripleStore {
	ts := store.NewTripleStore()

	// Create meetings
	meeting1 := "reg:meeting1"
	ts.Add(meeting1, "rdf:type", store.ClassMeeting)
	ts.Add(meeting1, store.RDFSLabel, "Meeting 1")
	ts.Add(meeting1, store.PropMeetingDate, "2024-03-15")

	meeting2 := "reg:meeting2"
	ts.Add(meeting2, "rdf:type", store.ClassMeeting)
	ts.Add(meeting2, store.RDFSLabel, "Meeting 2")
	ts.Add(meeting2, store.PropMeetingDate, "2024-03-29")

	meeting3 := "reg:meeting3"
	ts.Add(meeting3, "rdf:type", store.ClassMeeting)
	ts.Add(meeting3, store.RDFSLabel, "Meeting 3")
	ts.Add(meeting3, store.PropMeetingDate, "2024-04-12")

	meeting4 := "reg:meeting4"
	ts.Add(meeting4, "rdf:type", store.ClassMeeting)
	ts.Add(meeting4, store.RDFSLabel, "Meeting 4")
	ts.Add(meeting4, store.PropMeetingDate, "2024-04-26")

	meeting5 := "reg:meeting5"
	ts.Add(meeting5, "rdf:type", store.ClassMeeting)
	ts.Add(meeting5, store.RDFSLabel, "Meeting 5")
	ts.Add(meeting5, store.PropMeetingDate, "2024-05-10")

	// Create provision (Article 5)
	article5 := "reg:article5"
	ts.Add(article5, "rdf:type", store.ClassArticle)
	ts.Add(article5, store.RDFSLabel, "Article 5")

	// Proposal at Meeting 1
	ts.Add(article5, "reg:proposedAt", meeting1)
	ts.Add(article5, store.PropProposedBy, "reg:secretariat")
	ts.Add("reg:secretariat", store.RDFSLabel, "Secretariat")

	// Discussion at Meeting 2
	ts.Add(article5, store.PropDiscussedAt, meeting2)

	// Version 1 (original)
	version1 := "reg:article5:v1"
	ts.Add(version1, store.PropVersionOf, article5)
	ts.Add(version1, store.PropVersionNumber, "v1")
	ts.Add(version1, store.PropText, "Personal data shall be processed fairly.")

	// Amendment at Meeting 3 (rejected)
	version2 := "reg:article5:v2"
	ts.Add(version2, store.PropVersionOf, article5)
	ts.Add(version2, store.PropVersionNumber, "v2")
	ts.Add(version2, store.PropText, "Personal data shall be processed within 30 days.")
	ts.Add(version2, store.PropPreviousVersion, version1)
	ts.Add(version2, "reg:amendedAt", meeting3)
	ts.Add(version2, store.PropProposedBy, "reg:member_state_x")
	ts.Add("reg:member_state_x", store.RDFSLabel, "Member State X")

	// Vote on rejected amendment
	vote1 := "reg:vote1"
	ts.Add(version2, "reg:hasVote", vote1)
	ts.Add(version2, "reg:rejectedAt", meeting3)
	ts.Add(vote1, "rdf:type", store.ClassVoteRecord)
	ts.Add(vote1, store.PropVoteFor, "8")
	ts.Add(vote1, store.PropVoteAgainst, "12")
	ts.Add(vote1, store.PropVoteAbstain, "3")
	ts.Add(vote1, store.PropVoteResult, "rejected")

	// Amendment at Meeting 4 (revised proposal)
	version3 := "reg:article5:v3"
	ts.Add(version3, store.PropVersionOf, article5)
	ts.Add(version3, store.PropVersionNumber, "v3")
	ts.Add(version3, store.PropText, "Personal data shall be processed within 60 days.")
	ts.Add(version3, store.PropPreviousVersion, version1)
	ts.Add(version3, "reg:amendedAt", meeting4)
	ts.Add(version3, store.PropProposedBy, "reg:member_state_x")

	// Adoption at Meeting 5
	ts.Add(version3, "reg:adoptedAt", meeting5)
	vote2 := "reg:vote2"
	ts.Add(version3, "reg:hasVote", vote2)
	ts.Add(vote2, "rdf:type", store.ClassVoteRecord)
	ts.Add(vote2, store.PropVoteFor, "18")
	ts.Add(vote2, store.PropVoteAgainst, "4")
	ts.Add(vote2, store.PropVoteAbstain, "1")
	ts.Add(vote2, store.PropVoteResult, "adopted")

	// Create final decision
	decision1 := "reg:decision1"
	ts.Add(decision1, "rdf:type", store.ClassDeliberationDecision)
	ts.Add(decision1, store.PropDecidedAt, meeting5)
	ts.Add(decision1, store.PropTitle, "Article 5 adopted with 60-day processing limit")
	ts.Add(decision1, store.PropDecisionType, "adoption")
	ts.Add(decision1, store.PropAffectsProvision, article5)

	return ts
}

func TestNewProvenanceBuilder(t *testing.T) {
	ts := store.NewTripleStore()
	builder := NewProvenanceBuilder(ts, "https://example.org/")

	if builder == nil {
		t.Fatal("expected non-nil ProvenanceBuilder")
	}
	if builder.store != ts {
		t.Error("expected store to be set")
	}
	if builder.baseURI != "https://example.org/" {
		t.Errorf("expected baseURI 'https://example.org/', got %q", builder.baseURI)
	}
}

func TestBuildChainForDecision_NilStore(t *testing.T) {
	builder := &ProvenanceBuilder{store: nil}
	_, err := builder.BuildChainForDecision("reg:decision1")
	if err == nil {
		t.Error("expected error for nil store")
	}
}

func TestBuildChainForDecision_EmptyURI(t *testing.T) {
	ts := store.NewTripleStore()
	builder := NewProvenanceBuilder(ts, "https://example.org/")

	_, err := builder.BuildChainForDecision("")
	if err == nil {
		t.Error("expected error for empty URI")
	}
}

func TestBuildChainForDecision(t *testing.T) {
	ts := buildProvenanceTriples()
	builder := NewProvenanceBuilder(ts, "https://example.org/")

	chain, err := builder.BuildChainForDecision("reg:decision1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chain.FinalDecision != "reg:decision1" {
		t.Errorf("expected FinalDecision 'reg:decision1', got %q", chain.FinalDecision)
	}

	if chain.ProvisionURI != "reg:article5" {
		t.Errorf("expected ProvisionURI 'reg:article5', got %q", chain.ProvisionURI)
	}

	if chain.ProvisionLabel != "Article 5" {
		t.Errorf("expected ProvisionLabel 'Article 5', got %q", chain.ProvisionLabel)
	}

	// Should have multiple steps
	if len(chain.Steps) == 0 {
		t.Error("expected provenance steps")
	}

	// Check statistics
	if chain.MeetingCount == 0 {
		t.Error("expected non-zero meeting count")
	}
}

func TestBuildChainForProvision(t *testing.T) {
	ts := buildProvenanceTriples()
	builder := NewProvenanceBuilder(ts, "https://example.org/")

	chain, err := builder.BuildChainForProvision("reg:article5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chain.ProvisionURI != "reg:article5" {
		t.Errorf("expected ProvisionURI 'reg:article5', got %q", chain.ProvisionURI)
	}

	if chain.ProvisionLabel != "Article 5" {
		t.Errorf("expected ProvisionLabel 'Article 5', got %q", chain.ProvisionLabel)
	}

	// Should have steps from proposal through adoption
	if len(chain.Steps) < 3 {
		t.Errorf("expected at least 3 steps, got %d", len(chain.Steps))
	}

	// First step should be a proposal
	if len(chain.Steps) > 0 && chain.Steps[0].EventType != ProvenanceProposal {
		t.Errorf("expected first step to be PROPOSAL, got %s", chain.Steps[0].EventType.String())
	}
}

func TestBuildChainForProvision_NilStore(t *testing.T) {
	builder := &ProvenanceBuilder{store: nil}
	_, err := builder.BuildChainForProvision("reg:article5")
	if err == nil {
		t.Error("expected error for nil store")
	}
}

func TestBuildChainForProvision_EmptyURI(t *testing.T) {
	ts := store.NewTripleStore()
	builder := NewProvenanceBuilder(ts, "https://example.org/")

	_, err := builder.BuildChainForProvision("")
	if err == nil {
		t.Error("expected error for empty URI")
	}
}

func TestProvenanceChainStatistics(t *testing.T) {
	ts := buildProvenanceTriples()
	builder := NewProvenanceBuilder(ts, "https://example.org/")

	chain, err := builder.BuildChainForProvision("reg:article5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check origin meeting
	if chain.OriginMeeting == "" {
		t.Error("expected origin meeting to be set")
	}

	// Check meeting count
	if chain.MeetingCount < 2 {
		t.Errorf("expected at least 2 meetings, got %d", chain.MeetingCount)
	}

	// Check amendment count
	if chain.AmendmentCount < 1 {
		t.Errorf("expected at least 1 amendment, got %d", chain.AmendmentCount)
	}

	// Check current state - should end in adopted
	if chain.CurrentState != "adopted" {
		t.Errorf("expected current state 'adopted', got %q", chain.CurrentState)
	}
}

func TestProvenanceStateTransitions(t *testing.T) {
	ts := buildProvenanceTriples()
	builder := NewProvenanceBuilder(ts, "https://example.org/")

	chain, err := builder.BuildChainForProvision("reg:article5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify state transitions are continuous
	for i, step := range chain.Steps {
		if i > 0 {
			prevStep := chain.Steps[i-1]
			if step.FromState != prevStep.ToState {
				t.Errorf("step %d: FromState %q doesn't match previous ToState %q",
					i, step.FromState, prevStep.ToState)
			}
		}

		// FromState should never equal ToState (except for unchanged states)
		if step.FromState == step.ToState && step.EventType != ProvenanceDiscussion {
			// Discussion can keep same state, others should change
			switch step.EventType {
			case ProvenanceProposal, ProvenanceAmendment, ProvenanceAdoption, ProvenanceRejection:
				t.Errorf("step %d: state didn't change (%s -> %s) for event type %s",
					i, step.FromState, step.ToState, step.EventType.String())
			}
		}
	}
}

func TestQueryLongestChains(t *testing.T) {
	ts := buildProvenanceTriples()
	builder := NewProvenanceBuilder(ts, "https://example.org/")

	summaries, err := builder.QueryLongestChains(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find Article 5
	found := false
	for _, s := range summaries {
		if s.ProvisionURI == "reg:article5" {
			found = true
			if s.StepCount == 0 {
				t.Error("expected non-zero step count for Article 5")
			}
			break
		}
	}
	if !found {
		t.Error("expected to find Article 5 in longest chains")
	}

	// Should be sorted by step count (descending)
	for i := 1; i < len(summaries); i++ {
		if summaries[i].StepCount > summaries[i-1].StepCount {
			t.Error("summaries not sorted by step count descending")
		}
	}
}

func TestQueryLongestChains_NilStore(t *testing.T) {
	builder := &ProvenanceBuilder{store: nil}
	_, err := builder.QueryLongestChains(10)
	if err == nil {
		t.Error("expected error for nil store")
	}
}

func TestProvenanceEventTypeString(t *testing.T) {
	tests := []struct {
		eventType ProvenanceEventType
		want      string
	}{
		{ProvenanceProposal, "PROPOSAL"},
		{ProvenanceDiscussion, "DISCUSSION"},
		{ProvenanceAmendment, "AMENDMENT"},
		{ProvenanceVote, "VOTE"},
		{ProvenanceDeferral, "DEFERRAL"},
		{ProvenanceAdoption, "ADOPTION"},
		{ProvenanceRejection, "REJECTION"},
		{ProvenanceWithdrawal, "WITHDRAWAL"},
		{ProvenanceEventType(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		got := tt.eventType.String()
		if got != tt.want {
			t.Errorf("ProvenanceEventType(%d).String() = %q, want %q", tt.eventType, got, tt.want)
		}
	}
}

func TestParseProvenanceEventType(t *testing.T) {
	tests := []struct {
		input string
		want  ProvenanceEventType
	}{
		{"proposal", ProvenanceProposal},
		{"PROPOSAL", ProvenanceProposal},
		{"proposed", ProvenanceProposal},
		{"discussion", ProvenanceDiscussion},
		{"discussed", ProvenanceDiscussion},
		{"amendment", ProvenanceAmendment},
		{"amended", ProvenanceAmendment},
		{"vote", ProvenanceVote},
		{"voted", ProvenanceVote},
		{"deferral", ProvenanceDeferral},
		{"deferred", ProvenanceDeferral},
		{"adoption", ProvenanceAdoption},
		{"adopted", ProvenanceAdoption},
		{"rejection", ProvenanceRejection},
		{"rejected", ProvenanceRejection},
		{"withdrawal", ProvenanceWithdrawal},
		{"withdrawn", ProvenanceWithdrawal},
		{"unknown", ProvenanceDiscussion}, // default
	}

	for _, tt := range tests {
		got := parseProvenanceEventType(tt.input)
		if got != tt.want {
			t.Errorf("parseProvenanceEventType(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestComputeNextState(t *testing.T) {
	tests := []struct {
		currentState string
		eventType    ProvenanceEventType
		want         string
	}{
		{"none", ProvenanceProposal, "proposed"},
		{"proposed", ProvenanceDiscussion, "under_discussion"},
		{"under_discussion", ProvenanceAmendment, "amended"},
		{"amended", ProvenanceVote, "voted"},
		{"voted", ProvenanceDeferral, "deferred"},
		{"proposed", ProvenanceAdoption, "adopted"},
		{"amended", ProvenanceRejection, "rejected"},
		{"proposed", ProvenanceWithdrawal, "withdrawn"},
	}

	for _, tt := range tests {
		got := computeNextState(tt.currentState, tt.eventType)
		if got != tt.want {
			t.Errorf("computeNextState(%q, %s) = %q, want %q",
				tt.currentState, tt.eventType.String(), got, tt.want)
		}
	}
}

func TestProvenanceChainRenderASCII(t *testing.T) {
	ts := buildProvenanceTriples()
	builder := NewProvenanceBuilder(ts, "https://example.org/")

	chain, err := builder.BuildChainForProvision("reg:article5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := chain.RenderASCII()

	// Check header
	if !strings.Contains(output, "Provenance Chain: Article 5") {
		t.Error("expected header in ASCII output")
	}

	// Check for meeting labels
	if !strings.Contains(output, "Meeting 1") {
		t.Error("expected Meeting 1 in ASCII output")
	}

	// Check for event types
	if !strings.Contains(output, "[PROPOSAL]") {
		t.Error("expected PROPOSAL event in ASCII output")
	}

	// Check for state transitions
	if !strings.Contains(output, "State:") {
		t.Error("expected state transitions in ASCII output")
	}

	// Check for summary line
	if !strings.Contains(output, "Duration:") {
		t.Error("expected Duration in summary")
	}
	if !strings.Contains(output, "Meetings:") {
		t.Error("expected Meetings in summary")
	}
}

func TestProvenanceChainRenderASCII_Empty(t *testing.T) {
	chain := &ProvenanceChain{
		ProvisionLabel: "Empty Provision",
		Steps:          []ProvenanceStep{},
	}

	output := chain.RenderASCII()

	if !strings.Contains(output, "No provenance steps found") {
		t.Error("expected empty message for chain with no steps")
	}
}

func TestProvenanceChainRenderBrief(t *testing.T) {
	ts := buildProvenanceTriples()
	builder := NewProvenanceBuilder(ts, "https://example.org/")

	chain, err := builder.BuildChainForProvision("reg:article5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := chain.RenderBrief()

	// Should be a single line
	if strings.Contains(output, "\n") {
		t.Error("brief output should be single line")
	}

	// Should contain provision label
	if !strings.Contains(output, "Article 5") {
		t.Error("expected provision label in brief output")
	}

	// Should contain event types
	if !strings.Contains(output, "PROPOSAL") {
		t.Error("expected PROPOSAL in brief output")
	}

	// Should contain current state
	if !strings.Contains(output, "adopted") {
		t.Error("expected current state in brief output")
	}

	// Should contain meeting count
	if !strings.Contains(output, "meetings") {
		t.Error("expected meeting count in brief output")
	}
}

func TestProvenanceChainRenderJSON(t *testing.T) {
	ts := buildProvenanceTriples()
	builder := NewProvenanceBuilder(ts, "https://example.org/")

	chain, err := builder.BuildChainForProvision("reg:article5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jsonOutput, err := chain.RenderJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check JSON structure
	if !strings.Contains(jsonOutput, "\"provision_uri\"") {
		t.Error("expected 'provision_uri' field in JSON")
	}
	if !strings.Contains(jsonOutput, "\"steps\"") {
		t.Error("expected 'steps' field in JSON")
	}
	if !strings.Contains(jsonOutput, "\"meeting_count\"") {
		t.Error("expected 'meeting_count' field in JSON")
	}
	if !strings.Contains(jsonOutput, "\"current_state\"") {
		t.Error("expected 'current_state' field in JSON")
	}
}

func TestProvenanceChainRenderHTML(t *testing.T) {
	ts := buildProvenanceTriples()
	builder := NewProvenanceBuilder(ts, "https://example.org/")

	chain, err := builder.BuildChainForProvision("reg:article5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := chain.RenderHTML()

	// Check HTML structure
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("expected HTML doctype")
	}
	if !strings.Contains(output, "<title>Provenance Chain</title>") {
		t.Error("expected title")
	}

	// Check CSS classes
	if !strings.Contains(output, "class=\"step") {
		t.Error("expected step CSS class")
	}

	// Check content
	if !strings.Contains(output, "Article 5") {
		t.Error("expected Article 5 in output")
	}

	// Check summary
	if !strings.Contains(output, "summary-value") {
		t.Error("expected summary values")
	}
}

func TestProvenanceChainRenderHTML_Empty(t *testing.T) {
	chain := &ProvenanceChain{
		ProvisionLabel: "Empty Provision",
		Steps:          []ProvenanceStep{},
	}

	output := chain.RenderHTML()

	if !strings.Contains(output, "No provenance steps found") {
		t.Error("expected empty message for chain with no steps")
	}
}

func TestProvenanceChainRenderMermaid(t *testing.T) {
	ts := buildProvenanceTriples()
	builder := NewProvenanceBuilder(ts, "https://example.org/")

	chain, err := builder.BuildChainForProvision("reg:article5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := chain.RenderMermaid()

	// Check Mermaid structure
	if !strings.Contains(output, "graph TD") {
		t.Error("expected 'graph TD' in Mermaid output")
	}

	// Check for node definitions
	if !strings.Contains(output, "step0[") {
		t.Error("expected step0 node in Mermaid output")
	}

	// Check for edges
	if !strings.Contains(output, "-->") {
		t.Error("expected edges in Mermaid output")
	}

	// Check for class definitions
	if !strings.Contains(output, "classDef proposal") {
		t.Error("expected classDef for proposal")
	}
	if !strings.Contains(output, "classDef adoption") {
		t.Error("expected classDef for adoption")
	}
}

func TestProvenanceChainRenderMermaid_Empty(t *testing.T) {
	chain := &ProvenanceChain{
		Steps: []ProvenanceStep{},
	}

	output := chain.RenderMermaid()

	if !strings.Contains(output, "empty[No provenance steps]") {
		t.Error("expected empty message node")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{0, "< 1 day"},
		{12 * time.Hour, "< 1 day"},
		{24 * time.Hour, "1 day"},
		{48 * time.Hour, "2 days"},
		{7 * 24 * time.Hour, "7 days"},
		{56 * 24 * time.Hour, "56 days"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.duration)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
		}
	}
}

func TestProvenanceChainWithVoteRecord(t *testing.T) {
	ts := buildProvenanceTriples()
	builder := NewProvenanceBuilder(ts, "https://example.org/")

	chain, err := builder.BuildChainForProvision("reg:article5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find steps with vote records
	hasVoteRecord := false
	for _, step := range chain.Steps {
		if step.VoteRecord != nil {
			hasVoteRecord = true

			// Verify vote record has data
			if step.VoteRecord.ForCount == 0 && step.VoteRecord.AgainstCount == 0 {
				t.Error("expected vote counts to be populated")
			}
			break
		}
	}

	if !hasVoteRecord {
		t.Error("expected at least one step with a vote record")
	}
}

func TestProvenanceChainActorTracking(t *testing.T) {
	ts := buildProvenanceTriples()
	builder := NewProvenanceBuilder(ts, "https://example.org/")

	chain, err := builder.BuildChainForProvision("reg:article5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find steps with actors
	hasActor := false
	for _, step := range chain.Steps {
		if step.ActorName != "" {
			hasActor = true
			break
		}
	}

	if !hasActor {
		t.Error("expected at least one step with an actor")
	}
}

func TestProvenanceChainRelatedURIs(t *testing.T) {
	ts := buildProvenanceTriples()
	builder := NewProvenanceBuilder(ts, "https://example.org/")

	chain, err := builder.BuildChainForProvision("reg:article5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find steps with related URIs
	hasRelated := false
	for _, step := range chain.Steps {
		if len(step.RelatedURIs) > 0 {
			hasRelated = true
			break
		}
	}

	if !hasRelated {
		t.Error("expected at least one step with related URIs")
	}
}

func TestProvenanceChainDuration(t *testing.T) {
	ts := buildProvenanceTriples()
	builder := NewProvenanceBuilder(ts, "https://example.org/")

	chain, err := builder.BuildChainForProvision("reg:article5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Duration should be positive (from March 15 to May 10)
	if chain.TotalDuration <= 0 {
		t.Errorf("expected positive duration, got %v", chain.TotalDuration)
	}

	// Duration should be approximately 56 days
	expectedDays := 56
	actualDays := int(chain.TotalDuration.Hours() / 24)
	if actualDays < expectedDays-1 || actualDays > expectedDays+1 {
		t.Errorf("expected ~%d days, got %d days", expectedDays, actualDays)
	}
}
