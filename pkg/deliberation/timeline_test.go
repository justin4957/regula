package deliberation

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

func TestTimelineEventType_String(t *testing.T) {
	tests := []struct {
		eventType TimelineEventType
		expected  string
	}{
		{EventMeeting, "Meeting"},
		{EventProposal, "Proposal"},
		{EventAmendment, "Amendment"},
		{EventVote, "Vote"},
		{EventDecision, "Decision"},
		{EventDeferral, "Deferral"},
		{EventAction, "Action"},
		{EventComment, "Comment"},
		{TimelineEventType(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.eventType.String(); got != tt.expected {
			t.Errorf("TimelineEventType(%d).String() = %q, want %q", tt.eventType, got, tt.expected)
		}
	}
}

func TestTimelineEventType_Symbol(t *testing.T) {
	tests := []struct {
		eventType TimelineEventType
		expected  string
	}{
		{EventMeeting, "■"},
		{EventProposal, "●"},
		{EventAmendment, "◆"},
		{EventVote, "◐"},
		{EventDecision, "★"},
		{EventDeferral, "→"},
		{EventAction, "▶"},
		{EventComment, "○"},
		{TimelineEventType(99), "·"},
	}

	for _, tt := range tests {
		if got := tt.eventType.Symbol(); got != tt.expected {
			t.Errorf("TimelineEventType(%d).Symbol() = %q, want %q", tt.eventType, got, tt.expected)
		}
	}
}

func buildTimelineTriples() *store.TripleStore {
	ts := store.NewTripleStore()
	baseURI := "http://example.org/deliberation/"

	// Create a process
	processURI := baseURI + "process/budget-2024"
	ts.Add(processURI, "rdf:type", "reg:DeliberationProcess")
	ts.Add(processURI, store.PropTitle, "Budget Discussion 2024")

	// Create meetings
	meeting1URI := baseURI + "meeting/wg-2024-01"
	ts.Add(meeting1URI, "rdf:type", "reg:Meeting")
	ts.Add(meeting1URI, store.PropTitle, "Working Group Meeting 1")
	ts.Add(meeting1URI, "reg:identifier", "WG-2024-01")
	ts.Add(meeting1URI, "reg:date", "2024-03-15T10:00:00Z")
	ts.Add(meeting1URI, "reg:partOfProcess", processURI)

	meeting2URI := baseURI + "meeting/wg-2024-02"
	ts.Add(meeting2URI, "rdf:type", "reg:Meeting")
	ts.Add(meeting2URI, store.PropTitle, "Working Group Meeting 2")
	ts.Add(meeting2URI, "reg:identifier", "WG-2024-02")
	ts.Add(meeting2URI, "reg:date", "2024-04-15T10:00:00Z")
	ts.Add(meeting2URI, "reg:partOfProcess", processURI)

	meeting3URI := baseURI + "meeting/wg-2024-03"
	ts.Add(meeting3URI, "rdf:type", "reg:Meeting")
	ts.Add(meeting3URI, store.PropTitle, "Working Group Meeting 3")
	ts.Add(meeting3URI, "reg:identifier", "WG-2024-03")
	ts.Add(meeting3URI, "reg:date", "2024-05-15T10:00:00Z")
	ts.Add(meeting3URI, "reg:partOfProcess", processURI)

	// Create a provision
	provisionURI := baseURI + "provision/article-5"
	ts.Add(provisionURI, "rdf:type", "reg:Article")
	ts.Add(provisionURI, store.PropTitle, "Article 5 - Notification Requirements")

	// Create motions targeting the provision
	motion1URI := baseURI + "motion/amendment-1"
	ts.Add(motion1URI, "rdf:type", "reg:Motion")
	ts.Add(motion1URI, store.PropTitle, "Amendment to Article 5")
	ts.Add(motion1URI, store.PropText, "Change notification period from 30 to 60 days")
	ts.Add(motion1URI, "reg:targetProvision", provisionURI)
	ts.Add(motion1URI, "reg:proposedBy", baseURI+"stakeholder/delegation-x")
	ts.Add(motion1URI, "reg:proposedAt", "2024-03-15T10:30:00Z")

	motion2URI := baseURI + "motion/amendment-2"
	ts.Add(motion2URI, "rdf:type", "reg:Motion")
	ts.Add(motion2URI, store.PropTitle, "Revised Amendment to Article 5")
	ts.Add(motion2URI, store.PropText, "Change notification period to 45 days")
	ts.Add(motion2URI, "reg:targetProvision", provisionURI)
	ts.Add(motion2URI, "reg:proposedBy", baseURI+"stakeholder/delegation-y")
	ts.Add(motion2URI, "reg:proposedAt", "2024-04-15T11:00:00Z")

	// Create a decision
	decision1URI := baseURI + "decision/dec-2024-01"
	ts.Add(decision1URI, "rdf:type", "reg:Decision")
	ts.Add(decision1URI, store.PropTitle, "Adopted 45-day notification period")
	ts.Add(decision1URI, "reg:description", "Article 5 finalized with 45-day notification requirement")
	ts.Add(decision1URI, "reg:affectsProvision", provisionURI)
	ts.Add(decision1URI, "reg:decidedAt", "2024-05-15T14:00:00Z")

	// Create stakeholders
	stakeholder1URI := baseURI + "stakeholder/delegation-x"
	ts.Add(stakeholder1URI, "rdf:type", "reg:Stakeholder")
	ts.Add(stakeholder1URI, "reg:name", "Delegation X")

	stakeholder2URI := baseURI + "stakeholder/delegation-y"
	ts.Add(stakeholder2URI, "rdf:type", "reg:Stakeholder")
	ts.Add(stakeholder2URI, "reg:name", "Delegation Y")

	// Create interventions
	intervention1URI := baseURI + "intervention/int-1"
	ts.Add(intervention1URI, "rdf:type", "reg:Intervention")
	ts.Add(intervention1URI, "reg:speaker", stakeholder1URI)
	ts.Add(intervention1URI, "reg:speakerName", "Delegation X")
	ts.Add(intervention1URI, "reg:summary", "Proposed 60-day period for adequate preparation time")
	ts.Add(intervention1URI, "reg:timestamp", "2024-03-15T10:35:00Z")

	intervention2URI := baseURI + "intervention/int-2"
	ts.Add(intervention2URI, "rdf:type", "reg:Intervention")
	ts.Add(intervention2URI, "reg:speaker", stakeholder2URI)
	ts.Add(intervention2URI, "reg:speakerName", "Delegation Y")
	ts.Add(intervention2URI, "reg:summary", "Suggested 45-day compromise")
	ts.Add(intervention2URI, "reg:timestamp", "2024-04-15T11:10:00Z")

	// Create votes
	vote1URI := baseURI + "vote/vote-1"
	ts.Add(vote1URI, "rdf:type", "reg:IndividualVote")
	ts.Add(vote1URI, "reg:voter", stakeholder1URI)
	ts.Add(vote1URI, "reg:voterName", "Delegation X")
	ts.Add(vote1URI, "reg:position", "for")
	ts.Add(vote1URI, "reg:voteDate", "2024-05-15T14:30:00Z")

	// Create agenda items
	agenda1URI := baseURI + "agenda/item-1"
	ts.Add(agenda1URI, "rdf:type", "reg:AgendaItem")
	ts.Add(agenda1URI, store.PropTitle, "Discussion of Article 5 amendments")
	ts.Add(agenda1URI, "reg:discussesProvision", provisionURI)
	ts.Add(agenda1URI, "reg:meeting", meeting1URI)

	return ts
}

func TestNewTimelineBuilder(t *testing.T) {
	ts := store.NewTripleStore()
	builder := NewTimelineBuilder(ts, "http://example.org/")

	if builder == nil {
		t.Fatal("NewTimelineBuilder returned nil")
	}
	if builder.store != ts {
		t.Error("store not set correctly")
	}
	if builder.baseURI != "http://example.org/" {
		t.Errorf("baseURI = %q, want %q", builder.baseURI, "http://example.org/")
	}
}

func TestBuildProcessTimeline(t *testing.T) {
	ts := buildTimelineTriples()
	builder := NewTimelineBuilder(ts, "http://example.org/deliberation/")

	timeline, err := builder.BuildProcessTimeline("budget-2024", TimelineConfig{})
	if err != nil {
		t.Fatalf("BuildProcessTimeline failed: %v", err)
	}

	if timeline == nil {
		t.Fatal("timeline is nil")
	}
	if !strings.Contains(timeline.Title, "budget-2024") {
		t.Errorf("title should contain process ID, got %q", timeline.Title)
	}
	if timeline.ScopeType != "process" {
		t.Errorf("scope type = %q, want %q", timeline.ScopeType, "process")
	}

	// Should have at least meeting events
	if len(timeline.Events) == 0 {
		t.Error("expected some events in timeline")
	}
}

func TestBuildProvisionTimeline(t *testing.T) {
	ts := buildTimelineTriples()
	builder := NewTimelineBuilder(ts, "http://example.org/deliberation/")

	provisionURI := "http://example.org/deliberation/provision/article-5"
	timeline, err := builder.BuildProvisionTimeline(provisionURI, TimelineConfig{})
	if err != nil {
		t.Fatalf("BuildProvisionTimeline failed: %v", err)
	}

	if timeline == nil {
		t.Fatal("timeline is nil")
	}
	if !strings.Contains(timeline.Title, "article-5") {
		t.Errorf("title should contain provision label, got %q", timeline.Title)
	}
	if timeline.ScopeType != "provision" {
		t.Errorf("scope type = %q, want %q", timeline.ScopeType, "provision")
	}

	// Should have amendment and decision events
	hasAmendment := false
	hasDecision := false
	for _, event := range timeline.Events {
		if event.EventType == EventAmendment {
			hasAmendment = true
		}
		if event.EventType == EventDecision {
			hasDecision = true
		}
	}

	if !hasAmendment {
		t.Error("expected at least one amendment event")
	}
	if !hasDecision {
		t.Error("expected at least one decision event")
	}
}

func TestBuildStakeholderTimeline(t *testing.T) {
	ts := buildTimelineTriples()
	builder := NewTimelineBuilder(ts, "http://example.org/deliberation/")

	stakeholderURI := "http://example.org/deliberation/stakeholder/delegation-x"
	timeline, err := builder.BuildStakeholderTimeline(stakeholderURI, TimelineConfig{})
	if err != nil {
		t.Fatalf("BuildStakeholderTimeline failed: %v", err)
	}

	if timeline == nil {
		t.Fatal("timeline is nil")
	}
	if !strings.Contains(timeline.Title, "delegation-x") {
		t.Errorf("title should contain stakeholder label, got %q", timeline.Title)
	}
	if timeline.ScopeType != "stakeholder" {
		t.Errorf("scope type = %q, want %q", timeline.ScopeType, "stakeholder")
	}

	// Should have intervention, proposal, and vote events
	hasComment := false
	hasVote := false
	for _, event := range timeline.Events {
		if event.EventType == EventComment {
			hasComment = true
		}
		if event.EventType == EventVote {
			hasVote = true
		}
	}

	if !hasComment {
		t.Error("expected at least one comment/intervention event")
	}
	if !hasVote {
		t.Error("expected at least one vote event")
	}
}

func TestBuildComparativeTimeline(t *testing.T) {
	ts := buildTimelineTriples()
	builder := NewTimelineBuilder(ts, "http://example.org/deliberation/")

	// Add a second provision
	ts.Add("http://example.org/deliberation/provision/article-6", "rdf:type", "reg:Article")
	ts.Add("http://example.org/deliberation/provision/article-6", store.PropTitle, "Article 6")

	provisionURIs := []string{
		"http://example.org/deliberation/provision/article-5",
		"http://example.org/deliberation/provision/article-6",
	}

	timeline, err := builder.BuildComparativeTimeline(provisionURIs, TimelineConfig{})
	if err != nil {
		t.Fatalf("BuildComparativeTimeline failed: %v", err)
	}

	if timeline == nil {
		t.Fatal("timeline is nil")
	}
	if timeline.ScopeType != "comparative" {
		t.Errorf("scope type = %q, want %q", timeline.ScopeType, "comparative")
	}
	if len(timeline.Swimlanes) == 0 {
		t.Error("expected swimlanes in comparative timeline")
	}
}

func TestTimelineFilterByDateRange(t *testing.T) {
	ts := buildTimelineTriples()
	builder := NewTimelineBuilder(ts, "http://example.org/deliberation/")

	// Filter to only April 2024
	startDate := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 4, 30, 23, 59, 59, 0, time.UTC)

	timeline, err := builder.BuildProcessTimeline("budget-2024", TimelineConfig{
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		t.Fatalf("BuildProcessTimeline failed: %v", err)
	}

	for _, event := range timeline.Events {
		if event.Timestamp.Before(startDate) || event.Timestamp.After(endDate) {
			t.Errorf("event %q at %v should be filtered out (range: %v - %v)",
				event.Label, event.Timestamp, startDate, endDate)
		}
	}
}

func TestTimelineFilterByEventType(t *testing.T) {
	ts := buildTimelineTriples()
	builder := NewTimelineBuilder(ts, "http://example.org/deliberation/")

	timeline, err := builder.BuildProcessTimeline("budget-2024", TimelineConfig{
		EventTypes: []TimelineEventType{EventMeeting},
	})
	if err != nil {
		t.Fatalf("BuildProcessTimeline failed: %v", err)
	}

	for _, event := range timeline.Events {
		if event.EventType != EventMeeting {
			t.Errorf("event %q has type %s, expected only Meeting events",
				event.Label, event.EventType)
		}
	}
}

func TestTimelineGroupBySwimlane(t *testing.T) {
	ts := buildTimelineTriples()
	builder := NewTimelineBuilder(ts, "http://example.org/deliberation/")

	timeline, err := builder.BuildProcessTimeline("budget-2024", TimelineConfig{
		GroupBy: "type",
	})
	if err != nil {
		t.Fatalf("BuildProcessTimeline failed: %v", err)
	}

	if len(timeline.Swimlanes) == 0 {
		t.Error("expected swimlanes when grouping by type")
	}

	// Verify swimlanes have events
	totalEvents := 0
	for _, lane := range timeline.Swimlanes {
		totalEvents += len(lane.Events)
	}
	if totalEvents == 0 {
		t.Error("swimlanes should contain events")
	}
}

func TestMilestoneIdentification(t *testing.T) {
	ts := buildTimelineTriples()
	builder := NewTimelineBuilder(ts, "http://example.org/deliberation/")

	provisionURI := "http://example.org/deliberation/provision/article-5"
	timeline, err := builder.BuildProvisionTimeline(provisionURI, TimelineConfig{
		IncludeMilestones: true,
	})
	if err != nil {
		t.Fatalf("BuildProvisionTimeline failed: %v", err)
	}

	// Decisions should be identified as milestones
	if len(timeline.Milestones) == 0 {
		t.Error("expected at least one milestone (decision)")
	}

	for _, milestone := range timeline.Milestones {
		if milestone.Event.EventType != EventDecision {
			t.Logf("milestone: %s (type: %s)", milestone.Name, milestone.Event.EventType)
		}
	}
}

func TestRenderASCII(t *testing.T) {
	now := time.Now()
	timeline := &Timeline{
		Title:     "Test Timeline",
		Scope:     "test",
		ScopeType: "test",
		StartDate: now.AddDate(0, -1, 0),
		EndDate:   now,
		Events: []TimelineEvent{
			{
				Timestamp:   now.AddDate(0, 0, -20),
				EventType:   EventProposal,
				Label:       "Initial draft submitted",
				Description: "First version of the proposal",
			},
			{
				Timestamp:   now.AddDate(0, 0, -10),
				EventType:   EventAmendment,
				Label:       "Amendment proposed",
				Description: "Changes to section 2",
			},
			{
				Timestamp:   now.AddDate(0, 0, -5),
				EventType:   EventVote,
				Label:       "Vote on amendment",
				Description: "Amendment adopted (18-4-1)",
			},
			{
				Timestamp: now,
				EventType: EventDecision,
				Label:     "Final decision",
			},
		},
	}

	ascii := timeline.RenderASCII()

	// Check title
	if !strings.Contains(ascii, "Test Timeline") {
		t.Error("ASCII output should contain title")
	}

	// Check events are present
	if !strings.Contains(ascii, "Initial draft submitted") {
		t.Error("ASCII output should contain first event")
	}
	if !strings.Contains(ascii, "Amendment proposed") {
		t.Error("ASCII output should contain amendment event")
	}

	// Check legend
	if !strings.Contains(ascii, "Legend:") {
		t.Error("ASCII output should contain legend")
	}

	// Check symbols are present
	if !strings.Contains(ascii, "●") {
		t.Error("ASCII output should contain proposal symbol")
	}
	if !strings.Contains(ascii, "★") {
		t.Error("ASCII output should contain decision symbol")
	}
}

func TestRenderASCII_Empty(t *testing.T) {
	timeline := &Timeline{
		Title:  "Empty Timeline",
		Events: []TimelineEvent{},
	}

	ascii := timeline.RenderASCII()

	if !strings.Contains(ascii, "No events found") {
		t.Error("empty timeline should show 'No events found'")
	}
}

func TestRenderMermaid(t *testing.T) {
	now := time.Now()
	timeline := &Timeline{
		Title:     "Mermaid Test",
		StartDate: now.AddDate(0, -1, 0),
		EndDate:   now,
		Events: []TimelineEvent{
			{
				Timestamp: now.AddDate(0, 0, -10),
				EventType: EventProposal,
				Label:     "Test Proposal",
			},
			{
				Timestamp: now,
				EventType: EventDecision,
				Label:     "Test Decision",
			},
		},
	}

	mermaid := timeline.RenderMermaid()

	// Check Mermaid format
	if !strings.Contains(mermaid, "```mermaid") {
		t.Error("Mermaid output should start with code fence")
	}
	if !strings.Contains(mermaid, "timeline") {
		t.Error("Mermaid output should contain 'timeline' keyword")
	}
	if !strings.Contains(mermaid, "title Mermaid Test") {
		t.Error("Mermaid output should contain title")
	}
	if !strings.Contains(mermaid, "section") {
		t.Error("Mermaid output should contain date sections")
	}
	if !strings.Contains(mermaid, "Proposal") {
		t.Error("Mermaid output should contain event types")
	}
}

func TestRenderSVG(t *testing.T) {
	now := time.Now()
	timeline := &Timeline{
		Title:     "SVG Test",
		StartDate: now.AddDate(0, -1, 0),
		EndDate:   now,
		Events: []TimelineEvent{
			{
				Timestamp:   now.AddDate(0, 0, -10),
				EventType:   EventProposal,
				Label:       "Test Proposal",
				Description: "A test proposal",
			},
			{
				Timestamp: now,
				EventType: EventDecision,
				Label:     "Test Decision",
			},
		},
	}

	svg := timeline.RenderSVG()

	// Check SVG format
	if !strings.HasPrefix(svg, "<svg") {
		t.Error("SVG output should start with <svg tag")
	}
	if !strings.Contains(svg, "</svg>") {
		t.Error("SVG output should end with </svg>")
	}
	if !strings.Contains(svg, "SVG Test") {
		t.Error("SVG output should contain title")
	}
	if !strings.Contains(svg, "circle") {
		t.Error("SVG output should contain event circles")
	}
	if !strings.Contains(svg, "line") {
		t.Error("SVG output should contain timeline line")
	}
}

func TestRenderSVG_Empty(t *testing.T) {
	timeline := &Timeline{
		Title:  "Empty",
		Events: []TimelineEvent{},
	}

	svg := timeline.RenderSVG()

	if svg != "<svg></svg>" {
		t.Errorf("empty timeline SVG should be minimal, got %q", svg)
	}
}

func TestTimelineRenderHTML(t *testing.T) {
	now := time.Now()
	timeline := &Timeline{
		Title:     "HTML Test",
		StartDate: now.AddDate(0, -1, 0),
		EndDate:   now,
		Events: []TimelineEvent{
			{
				Timestamp:   now.AddDate(0, 0, -10),
				EventType:   EventProposal,
				Label:       "Test Proposal",
				Description: "A test proposal",
				Actors:      []string{"Alice", "Bob"},
			},
			{
				Timestamp:   now,
				EventType:   EventDecision,
				Label:       "Test Decision",
				IsMilestone: true,
			},
		},
	}

	html := timeline.RenderHTML()

	// Check HTML structure
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("HTML output should have DOCTYPE")
	}
	if !strings.Contains(html, "<title>HTML Test</title>") {
		t.Error("HTML output should have title")
	}
	if !strings.Contains(html, "class=\"timeline\"") {
		t.Error("HTML output should have timeline class")
	}
	if !strings.Contains(html, "class=\"event proposal\"") {
		t.Error("HTML output should have event with type class")
	}
	if !strings.Contains(html, "milestone") {
		t.Error("HTML output should mark milestones")
	}
	if !strings.Contains(html, "Alice, Bob") {
		t.Error("HTML output should show actors")
	}

	// Check filter buttons
	if !strings.Contains(html, "filter-btn") {
		t.Error("HTML output should have filter buttons")
	}

	// Check script for interactivity
	if !strings.Contains(html, "<script>") {
		t.Error("HTML output should have interactive script")
	}
}

func TestTimelineRenderJSON(t *testing.T) {
	now := time.Now()
	timeline := &Timeline{
		Title:     "JSON Test",
		Scope:     "test-scope",
		ScopeType: "test",
		StartDate: now.AddDate(0, -1, 0),
		EndDate:   now,
		Events: []TimelineEvent{
			{
				Timestamp: now,
				EventType: EventDecision,
				Label:     "Test Decision",
				Metadata: map[string]string{
					"key": "value",
				},
			},
		},
	}

	jsonStr, err := timeline.RenderJSON()
	if err != nil {
		t.Fatalf("RenderJSON failed: %v", err)
	}

	// Verify it's valid JSON
	var parsed Timeline
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("RenderJSON produced invalid JSON: %v", err)
	}

	if parsed.Title != "JSON Test" {
		t.Errorf("parsed title = %q, want %q", parsed.Title, "JSON Test")
	}
	if len(parsed.Events) != 1 {
		t.Errorf("parsed events count = %d, want 1", len(parsed.Events))
	}
}

func TestTimelineEvent_Metadata(t *testing.T) {
	event := TimelineEvent{
		Timestamp: time.Now(),
		EventType: EventVote,
		Label:     "Vote Result",
		Metadata: map[string]string{
			"result":  "adopted",
			"for":     "18",
			"against": "4",
			"abstain": "1",
		},
	}

	if event.Metadata["result"] != "adopted" {
		t.Errorf("metadata result = %q, want %q", event.Metadata["result"], "adopted")
	}
	if event.Metadata["for"] != "18" {
		t.Errorf("metadata for = %q, want %q", event.Metadata["for"], "18")
	}
}

func TestTimelineEvent_Links(t *testing.T) {
	event := TimelineEvent{
		Timestamp: time.Now(),
		EventType: EventDecision,
		Label:     "Multi-provision Decision",
		Links: []string{
			"http://example.org/provision/article-5",
			"http://example.org/provision/article-6",
		},
	}

	if len(event.Links) != 2 {
		t.Errorf("links count = %d, want 2", len(event.Links))
	}
}

func TestSwimlane(t *testing.T) {
	now := time.Now()
	swimlane := Swimlane{
		Name: "Delegation X",
		URI:  "http://example.org/stakeholder/delegation-x",
		Events: []TimelineEvent{
			{Timestamp: now.AddDate(0, 0, -10), EventType: EventProposal, Label: "Proposal 1"},
			{Timestamp: now, EventType: EventVote, Label: "Vote 1"},
		},
	}

	if swimlane.Name != "Delegation X" {
		t.Errorf("swimlane name = %q, want %q", swimlane.Name, "Delegation X")
	}
	if len(swimlane.Events) != 2 {
		t.Errorf("swimlane events = %d, want 2", len(swimlane.Events))
	}
}

func TestMilestone(t *testing.T) {
	event := TimelineEvent{
		Timestamp:   time.Now(),
		EventType:   EventDecision,
		Label:       "Final Decision",
		IsMilestone: true,
	}

	milestone := Milestone{
		Event: event,
		Name:  "Project Completion",
	}

	if milestone.Name != "Project Completion" {
		t.Errorf("milestone name = %q, want %q", milestone.Name, "Project Completion")
	}
	if !milestone.Event.IsMilestone {
		t.Error("milestone event should have IsMilestone=true")
	}
}

func TestTimelineExtractURILabel(t *testing.T) {
	tests := []struct {
		uri      string
		expected string
	}{
		{"http://example.org/provision/article-5", "article-5"},
		{"http://example.org/stakeholder/delegation-x", "delegation-x"},
		{"simple", "simple"},
		{"", ""},
	}

	for _, tt := range tests {
		if got := extractURILabel(tt.uri); got != tt.expected {
			t.Errorf("extractURILabel(%q) = %q, want %q", tt.uri, got, tt.expected)
		}
	}
}

func TestTimelineConfig(t *testing.T) {
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	config := TimelineConfig{
		StartDate:         startDate,
		EndDate:           endDate,
		EventTypes:        []TimelineEventType{EventMeeting, EventDecision},
		GroupBy:           "stakeholder",
		IncludeMilestones: true,
		Provision:         "article-5",
		Stakeholder:       "delegation-x",
		CompareProvisions: []string{"article-5", "article-6"},
	}

	if !config.StartDate.Equal(startDate) {
		t.Error("StartDate not set correctly")
	}
	if len(config.EventTypes) != 2 {
		t.Errorf("EventTypes count = %d, want 2", len(config.EventTypes))
	}
	if config.GroupBy != "stakeholder" {
		t.Errorf("GroupBy = %q, want %q", config.GroupBy, "stakeholder")
	}
	if len(config.CompareProvisions) != 2 {
		t.Errorf("CompareProvisions count = %d, want 2", len(config.CompareProvisions))
	}
}

func TestCalculateDateRange(t *testing.T) {
	builder := &TimelineBuilder{}
	now := time.Now()

	// Test with events
	events := []TimelineEvent{
		{Timestamp: now.AddDate(0, -2, 0)},
		{Timestamp: now.AddDate(0, -1, 0)},
		{Timestamp: now},
	}

	start, end := builder.calculateDateRange(events, TimelineConfig{})
	if !start.Equal(events[0].Timestamp) {
		t.Errorf("start date should be first event timestamp")
	}
	if !end.Equal(events[2].Timestamp) {
		t.Errorf("end date should be last event timestamp")
	}

	// Test with config override
	configStart := now.AddDate(-1, 0, 0)
	configEnd := now.AddDate(1, 0, 0)
	start, end = builder.calculateDateRange(events, TimelineConfig{
		StartDate: configStart,
		EndDate:   configEnd,
	})
	if !start.Equal(configStart) {
		t.Errorf("start date should use config override")
	}
	if !end.Equal(configEnd) {
		t.Errorf("end date should use config override")
	}

	// Test with empty events
	start, end = builder.calculateDateRange([]TimelineEvent{}, TimelineConfig{})
	if start.After(end) {
		t.Error("empty events should still return valid range")
	}
}

func TestFilterEventsWithStakeholder(t *testing.T) {
	builder := &TimelineBuilder{}
	now := time.Now()

	events := []TimelineEvent{
		{Timestamp: now, EventType: EventProposal, Label: "Event 1", Actors: []string{"Alice"}},
		{Timestamp: now, EventType: EventVote, Label: "Event 2", Actors: []string{"Bob"}},
		{Timestamp: now, EventType: EventDecision, Label: "Event 3", Actors: []string{"Alice", "Bob"}},
	}

	filtered := builder.filterEvents(events, TimelineConfig{
		Stakeholder: "alice",
	})

	if len(filtered) != 2 {
		t.Errorf("expected 2 events with Alice, got %d", len(filtered))
	}
}

func TestFilterEventsWithProvision(t *testing.T) {
	builder := &TimelineBuilder{}
	now := time.Now()

	events := []TimelineEvent{
		{Timestamp: now, EventType: EventAmendment, Label: "Event 1", Links: []string{"article-5"}},
		{Timestamp: now, EventType: EventDecision, Label: "Event 2", Links: []string{"article-6"}},
		{Timestamp: now, EventType: EventVote, Label: "Event 3", URI: "motion-article-5-1"},
	}

	filtered := builder.filterEvents(events, TimelineConfig{
		Provision: "article-5",
	})

	if len(filtered) != 2 {
		t.Errorf("expected 2 events related to article-5, got %d", len(filtered))
	}
}

func TestCreateSwimlanesGroupByType(t *testing.T) {
	builder := &TimelineBuilder{}
	now := time.Now()

	events := []TimelineEvent{
		{Timestamp: now, EventType: EventMeeting, Label: "Meeting 1"},
		{Timestamp: now, EventType: EventMeeting, Label: "Meeting 2"},
		{Timestamp: now, EventType: EventDecision, Label: "Decision 1"},
		{Timestamp: now, EventType: EventVote, Label: "Vote 1"},
	}

	swimlanes := builder.createSwimlanes(events, "type")

	// Should have 3 swimlanes: Meeting, Decision, Vote
	if len(swimlanes) != 3 {
		t.Errorf("expected 3 swimlanes, got %d", len(swimlanes))
	}

	// Find Meeting swimlane and check count
	for _, lane := range swimlanes {
		if lane.Name == "Meeting" && len(lane.Events) != 2 {
			t.Errorf("Meeting swimlane should have 2 events, got %d", len(lane.Events))
		}
	}
}

func TestHTMLEscaping(t *testing.T) {
	timeline := &Timeline{
		Title: "Test <script>alert('xss')</script>",
		Events: []TimelineEvent{
			{
				Timestamp:   time.Now(),
				EventType:   EventProposal,
				Label:       "Event with <b>HTML</b>",
				Description: "Description with \"quotes\" and 'apostrophes'",
			},
		},
		StartDate: time.Now().AddDate(0, -1, 0),
		EndDate:   time.Now(),
	}

	html := timeline.RenderHTML()

	// Check that HTML is escaped
	if strings.Contains(html, "<script>alert") {
		t.Error("HTML should escape script tags in title")
	}
	if strings.Contains(html, "<b>HTML</b>") {
		t.Error("HTML should escape bold tags in label")
	}
}

func TestLongLabelTruncation(t *testing.T) {
	longLabel := strings.Repeat("A", 100)
	timeline := &Timeline{
		Title: "Test",
		Events: []TimelineEvent{
			{
				Timestamp: time.Now(),
				EventType: EventProposal,
				Label:     longLabel,
			},
		},
		StartDate: time.Now().AddDate(0, -1, 0),
		EndDate:   time.Now(),
	}

	ascii := timeline.RenderASCII()
	svg := timeline.RenderSVG()

	// Both should truncate long labels
	if strings.Contains(ascii, longLabel) {
		t.Error("ASCII should truncate long labels")
	}
	if !strings.Contains(ascii, "...") {
		t.Error("ASCII truncated label should have ellipsis")
	}
	if strings.Contains(svg, longLabel) {
		t.Error("SVG should truncate long labels")
	}
}
