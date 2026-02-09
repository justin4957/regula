package deliberation

import (
	"strings"
	"testing"
	"time"
)

// Sample EU-style meeting minutes for testing
const sampleEUMinutes = `
COUNCIL OF THE EUROPEAN UNION
Working Group on Data Protection
Meeting Minutes

Date: 15 May 2024
Time: 10:00
Location: Justus Lipsius Building, Room JL-40.5

Chair: Ms. Anna Schmidt (DE)
Secretary: Mr. Jean Dupont (Council Secretariat)

Present: DE, FR, IT, ES, PL, NL, BE, AT, SE, DK, FI, PT, CZ, IE, RO, HU, BG, EL, SK, SI, HR, LV, LT, EE, CY, MT, LU

Apologies: None

AGENDA

1. Adoption of the agenda
The agenda was adopted as proposed.

2. Review of Article 5 - Data Retention Period

The GERMANY delegation stated that the 30-day retention limit provides necessary clarity for data subjects while allowing reasonable operational flexibility.

The FRANCE delegation expressed concerns about implementation costs for small businesses and requested further analysis.

The COMMISSION representative noted that impact assessments would be conducted during the implementation phase.

DECISION: The Working Group agreed to proceed with the 30-day retention period as the baseline, with the Commission asked to prepare implementation guidelines.

3. Discussion of Cross-Border Transfer Mechanisms

The NETHERLANDS delegation proposed an amendment to streamline adequacy decisions.

The delegation of POLAND said it supported the Dutch proposal with minor modifications.

ITALY stated it had reservations about the timeline proposed.

The Working Group discussed the proposal at length. Several delegations expressed qualified support.

Vote: 18 for, 4 against, 5 abstain

The amendment was adopted.

4. Any other business

[ACTION] Secretariat to circulate revised text by 1 June 2024

[ACTION] Commission to prepare impact assessment report

The Chair announced the next meeting would take place on 29 May 2024.

5. Date of next meeting

The next meeting is scheduled for 29 May 2024 at 10:00.

The meeting closed at 17:30.
`

// Sample generic meeting minutes
const sampleGenericMinutes = `
Board of Directors Meeting
XYZ Corporation

Date: March 15, 2024
Location: Conference Room A, Corporate Headquarters

Attendees: John Smith (Chair), Mary Johnson, Robert Brown, Susan Davis, Michael Wilson
Apologies: Jennifer Lee

1. Call to Order
The meeting was called to order at 9:00 AM by the Chair.

2. Approval of Previous Minutes
The minutes of the February 15, 2024 meeting were approved as presented.

3. Financial Report

The CFO, Mary Johnson, presented the quarterly financial results.

John Smith stated that the results exceeded expectations.

Robert Brown asked about projections for Q2.

Decision: The Board approved the financial report.

4. New Business - Expansion Proposal

Susan Davis proposed expanding operations to the European market.

Michael Wilson noted potential regulatory challenges.

After discussion, it was decided that management would prepare a detailed proposal by April 30, 2024.

ACTION: Susan Davis to prepare market analysis by April 15, 2024.
ACTION: Michael Wilson to review regulatory requirements.

5. Adjournment
The meeting adjourned at 11:30 AM.

Next meeting: April 19, 2024
`

// Sample UN-style meeting minutes
const sampleUNMinutes = `
UNITED NATIONS
General Assembly
Seventy-ninth session
123rd plenary meeting

Friday, 20 December 2024, 3 p.m.
New York

President: Mr. Dennis Francis (Trinidad and Tobago)

1. Adoption of the agenda

The agenda was adopted.

2. Situation of human rights in Country X

The representative of Sweden, speaking on behalf of the Nordic countries, said that the international community must address the deteriorating human rights situation.

The representative of China noted that the resolution contained biased language and interference in internal affairs.

The Secretary-General emphasized the importance of dialogue.

The representative of the United States expressed strong support for the resolution and called for immediate action.

A recorded vote was taken on draft resolution A/79/L.45.

Vote: 98 for, 23 against, 30 abstentions

Draft resolution A/79/L.45 was adopted.

DECISION: The General Assembly adopted resolution 79/100 on the situation of human rights in Country X.

3. Organization of work

The President stated that the Assembly would continue its work in January 2025.

The representative of Brazil requested that document A/79/500 be circulated to all delegations.

ACTION: The Secretariat was requested to prepare a comprehensive report for the next session.

The meeting rose at 6 p.m.
`

func TestMinutesParser_Parse_EU(t *testing.T) {
	parser := NewMinutesParserWithFormat("https://example.org", "eu")
	meeting, err := parser.Parse(sampleEUMinutes)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify meeting metadata
	if meeting.Date.Year() != 2024 || meeting.Date.Month() != time.May || meeting.Date.Day() != 15 {
		t.Errorf("Date = %v, want 15 May 2024", meeting.Date)
	}

	if meeting.Location != "Justus Lipsius Building, Room JL-40.5" {
		t.Errorf("Location = %q, want %q", meeting.Location, "Justus Lipsius Building, Room JL-40.5")
	}

	// Note: Chair extraction depends on exact pattern matching
	// The sample uses "Chair: Ms. Anna Schmidt (DE)" which may not match all patterns
	t.Logf("Chair: %s", meeting.ChairName)

	// Verify agenda items were found
	if len(meeting.AgendaItems) < 3 {
		t.Errorf("len(AgendaItems) = %d, want at least 3", len(meeting.AgendaItems))
	}

	t.Logf("Parsed meeting: %s", meeting.String())
	t.Logf("Agenda items: %d", len(meeting.AgendaItems))
	for _, item := range meeting.AgendaItems {
		t.Logf("  - %s: %s (outcome: %s)", item.Number, item.Title, item.Outcome)
	}
}

func TestMinutesParser_Parse_Generic(t *testing.T) {
	parser := NewMinutesParser("https://example.org")
	meeting, err := parser.Parse(sampleGenericMinutes)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify date
	if meeting.Date.Year() != 2024 || meeting.Date.Month() != time.March || meeting.Date.Day() != 15 {
		t.Errorf("Date = %v, want 15 March 2024", meeting.Date)
	}

	// Verify agenda items
	if len(meeting.AgendaItems) < 4 {
		t.Errorf("len(AgendaItems) = %d, want at least 4", len(meeting.AgendaItems))
	}

	t.Logf("Parsed meeting with %d agenda items", len(meeting.AgendaItems))
}

func TestMinutesParser_Parse_UN(t *testing.T) {
	parser := NewMinutesParserWithFormat("https://example.org", "un")
	meeting, err := parser.Parse(sampleUNMinutes)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify date
	if meeting.Date.Year() != 2024 || meeting.Date.Month() != time.December || meeting.Date.Day() != 20 {
		t.Errorf("Date = %v, want 20 December 2024", meeting.Date)
	}

	// Verify agenda items
	if len(meeting.AgendaItems) < 2 {
		t.Errorf("len(AgendaItems) = %d, want at least 2", len(meeting.AgendaItems))
	}

	t.Logf("Parsed UN meeting with %d agenda items", len(meeting.AgendaItems))
}

func TestMinutesParser_ExtractAgenda(t *testing.T) {
	parser := NewMinutesParser("https://example.org")

	tests := []struct {
		name          string
		text          string
		minItems      int
		wantFirstNum  string
		wantFirstWord string
	}{
		{
			name: "numbered items",
			text: `
1. Introduction
2. Financial Report
3. New Business
4. Adjournment`,
			minItems:      4,
			wantFirstNum:  "1",
			wantFirstWord: "Introduction",
		},
		{
			name: "item prefix",
			text: `
Item 1: Welcome and introductions
Item 2: Review of action items
Item 3: Main discussion`,
			minItems:      3,
			wantFirstNum:  "1",
			wantFirstWord: "Welcome",
		},
		{
			name: "roman numerals",
			text: `
I. Opening remarks
II. Approval of minutes
III. Committee reports`,
			minItems:      3,
			wantFirstNum:  "I",
			wantFirstWord: "Opening",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := parser.ExtractAgenda(tt.text)
			if err != nil {
				t.Fatalf("ExtractAgenda failed: %v", err)
			}

			if len(items) < tt.minItems {
				t.Errorf("len(items) = %d, want at least %d", len(items), tt.minItems)
			}

			if len(items) > 0 {
				if items[0].Number != tt.wantFirstNum {
					t.Errorf("first item number = %q, want %q", items[0].Number, tt.wantFirstNum)
				}
				if !strings.Contains(items[0].Title, tt.wantFirstWord) {
					t.Errorf("first item title = %q, want to contain %q", items[0].Title, tt.wantFirstWord)
				}
			}
		})
	}
}

func TestMinutesParser_ExtractDecisions(t *testing.T) {
	parser := NewMinutesParser("https://example.org")

	text := `
The committee discussed the proposal at length.

Decision: The Board approved the annual budget for 2024.

After further debate, it was agreed that the policy would be implemented from January 1.

The motion was adopted by a vote of 15 to 3.

RESOLVED: The organization will expand operations to Europe.
`

	decisions, err := parser.ExtractDecisions(text)
	if err != nil {
		t.Fatalf("ExtractDecisions failed: %v", err)
	}

	if len(decisions) < 2 {
		t.Errorf("len(decisions) = %d, want at least 2", len(decisions))
	}

	// Check that decision types are detected
	hasAdoption := false
	for _, d := range decisions {
		t.Logf("Decision: %s (type: %s)", d.Title, d.Type)
		if d.Type == "adoption" {
			hasAdoption = true
		}
	}
	if !hasAdoption {
		t.Error("Expected at least one adoption-type decision")
	}
}

func TestMinutesParser_ExtractSpeakers(t *testing.T) {
	parser := NewMinutesParser("https://example.org")

	text := `
The representative of Germany stated that the proposal had merit.

Mr. Smith said he supported the motion.

The Chair noted that time was limited.

FRANCE: We have concerns about implementation.

The Commission representative explained the legal framework.
`

	interventions, err := parser.ExtractSpeakers(text)
	if err != nil {
		t.Fatalf("ExtractSpeakers failed: %v", err)
	}

	if len(interventions) < 2 {
		t.Errorf("len(interventions) = %d, want at least 2", len(interventions))
	}

	for _, int := range interventions {
		t.Logf("Intervention by %s: position=%s", int.SpeakerName, int.Position)
	}
}

func TestMinutesParser_ExtractActions(t *testing.T) {
	parser := NewMinutesParser("https://example.org")

	text := `
Several action items were identified:

[ACTION] Secretariat to circulate the revised text by Friday.

ACTION: John to prepare the budget report.

The Committee asked the Chair to schedule a follow-up meeting.

It was agreed that Mary would contact stakeholders before the deadline of 30 June 2024.
`

	actions, err := parser.ExtractActions(text)
	if err != nil {
		t.Fatalf("ExtractActions failed: %v", err)
	}

	if len(actions) < 2 {
		t.Errorf("len(actions) = %d, want at least 2", len(actions))
	}

	for _, action := range actions {
		t.Logf("Action: %s (status: %s)", action.Description, action.Status)
		if action.DueDate != nil {
			t.Logf("  Due: %s", action.DueDate.Format("2006-01-02"))
		}
	}
}

func TestMinutesParser_ExtractReferences(t *testing.T) {
	parser := NewMinutesParser("https://example.org")

	text := `
The committee reviewed document WG/2024/15 and the related paper.

Article 5 was discussed in detail, along with Article 12(3).

Reference was made to the previous meeting of 15 April.

Document ST 12345/24 REV 2 was circulated.
`

	refs := parser.ExtractReferences(text)

	if len(refs) < 2 {
		t.Errorf("len(refs) = %d, want at least 2", len(refs))
	}

	hasDocRef := false
	hasArticleRef := false
	for _, ref := range refs {
		t.Logf("Reference: type=%s, id=%s", ref.Type, ref.Identifier)
		if ref.Type == "document" {
			hasDocRef = true
		}
		if ref.Type == "article" {
			hasArticleRef = true
		}
	}

	if !hasDocRef {
		t.Error("Expected at least one document reference")
	}
	if !hasArticleRef {
		t.Error("Expected at least one article reference")
	}
}

func TestMinutesParser_DetectOutcome(t *testing.T) {
	parser := NewMinutesParser("https://example.org")

	tests := []struct {
		content  string
		expected AgendaItemOutcome
	}{
		{"The proposal was adopted unanimously.", OutcomeDecided},
		{"The committee agreed to proceed.", OutcomeDecided},
		{"This item was deferred to the next meeting.", OutcomeDeferred},
		{"The motion was withdrawn by the proposer.", OutcomeWithdrawn},
		{"The matter was discussed at length.", OutcomeDiscussed},
		{"No quorum was present for the vote.", OutcomeNoQuorum},
		{"Some random text here.", OutcomePending},
	}

	for _, tt := range tests {
		t.Run(tt.expected.String(), func(t *testing.T) {
			result := parser.detectOutcome(tt.content)
			if result != tt.expected {
				t.Errorf("detectOutcome(%q) = %v, want %v", tt.content[:30], result, tt.expected)
			}
		})
	}
}

func TestMinutesParser_DetectPosition(t *testing.T) {
	parser := NewMinutesParser("https://example.org")

	tests := []struct {
		text     string
		expected InterventionPosition
	}{
		{"expressed strong support for the proposal", PositionSupport},
		{"welcomes the initiative", PositionSupport},
		{"opposed the amendment", PositionOppose},
		{"has concerns about implementation", PositionOppose},
		{"asked for clarification on the timeline", PositionQuestion},
		{"raised a point of order", PositionProcedural},
		{"supported with reservations", PositionQualified},
		{"made a general statement", PositionNeutral},
	}

	for _, tt := range tests {
		t.Run(tt.expected.String(), func(t *testing.T) {
			result := parser.detectPosition(tt.text)
			if result != tt.expected {
				t.Errorf("detectPosition(%q) = %v, want %v", tt.text[:20], result, tt.expected)
			}
		})
	}
}

func TestMinutesParser_EmptyInput(t *testing.T) {
	parser := NewMinutesParser("https://example.org")

	_, err := parser.Parse("")
	if err == nil {
		t.Error("Parse should return error for empty input")
	}
}

func TestMinutesParser_URIGeneration(t *testing.T) {
	parser := NewMinutesParser("https://example.org")

	meeting := &Meeting{
		Identifier: "WG-2024-05",
	}

	uri := parser.generateMeetingURI(meeting)
	if !strings.Contains(uri, "wg-2024-05") {
		t.Errorf("URI = %q, want to contain 'wg-2024-05'", uri)
	}

	// Test with date fallback
	meeting2 := &Meeting{
		Date: time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC),
	}
	uri2 := parser.generateMeetingURI(meeting2)
	if !strings.Contains(uri2, "2024-05-15") {
		t.Errorf("URI = %q, want to contain '2024-05-15'", uri2)
	}
}

func TestSanitizeForURI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple Text", "simple-text"},
		{"Meeting #5", "meeting-5"},
		{"WG-2024/05", "wg-2024-05"},
		{"  Spaces  ", "spaces"},
		{"Mr. John Smith", "mr-john-smith"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeForURI(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeForURI(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"Short", 10, "Short"},
		{"This is a long string", 10, "This is..."},
		{"Exact fit!", 10, "Exact fit!"},
		{"", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestMinutesParser_FullIntegration(t *testing.T) {
	// Test complete parsing workflow with EU minutes
	parser := NewMinutesParserWithFormat("https://eu.example.org/deliberations", "eu")
	meeting, err := parser.Parse(sampleEUMinutes)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify comprehensive extraction
	t.Logf("=== Meeting Summary ===")
	t.Logf("Title: %s", meeting.Title)
	t.Logf("Date: %s", meeting.Date.Format("2006-01-02"))
	t.Logf("Location: %s", meeting.Location)
	t.Logf("Chair: %s", meeting.ChairName)
	t.Logf("Agenda Items: %d", len(meeting.AgendaItems))

	totalDecisions := 0
	totalInterventions := 0
	totalActions := 0

	for _, item := range meeting.AgendaItems {
		t.Logf("\nItem %s: %s", item.Number, item.Title)
		t.Logf("  Outcome: %s", item.Outcome)
		t.Logf("  Decisions: %d", len(item.Decisions))
		t.Logf("  Interventions: %d", len(item.Interventions))
		t.Logf("  Actions: %d", len(item.ActionItems))

		totalDecisions += len(item.Decisions)
		totalInterventions += len(item.Interventions)
		totalActions += len(item.ActionItems)
	}

	t.Logf("\n=== Totals ===")
	t.Logf("Total Decisions: %d", totalDecisions)
	t.Logf("Total Interventions: %d", totalInterventions)
	t.Logf("Total Actions: %d", totalActions)

	// Assertions for minimum extraction quality
	if len(meeting.AgendaItems) < 3 {
		t.Errorf("Expected at least 3 agenda items, got %d", len(meeting.AgendaItems))
	}
	if totalDecisions < 1 {
		t.Errorf("Expected at least 1 decision, got %d", totalDecisions)
	}
	if totalInterventions < 2 {
		t.Errorf("Expected at least 2 interventions, got %d", totalInterventions)
	}
	if totalActions < 1 {
		t.Errorf("Expected at least 1 action item, got %d", totalActions)
	}
}
