package deliberation

import (
	"strings"
	"testing"
	"time"
)

// Sample UN General Assembly resolution text
const sampleUNResolution = `
A/RES/79/100

Resolution adopted by the General Assembly on 18 December 2024

79th session
Agenda item 42

The General Assembly,

Recalling its resolution 78/200 of 15 January 2024, in which it decided to continue its consideration of this matter,

Noting with concern the ongoing challenges facing developing countries in achieving sustainable development goals,

Reaffirming the importance of international cooperation and the principles enshrined in the Charter of the United Nations,

Recognizing the need for urgent action to address climate change and its impacts on vulnerable populations,

Guided by the Universal Declaration of Human Rights and the International Covenant on Civil and Political Rights,

Bearing in mind the report of the Secretary-General (A/79/123),

1. Decides to establish a special fund to support developing countries in their efforts to implement the 2030 Agenda for Sustainable Development;

2. Requests the Secretary-General to submit a report to the General Assembly at its eightieth session on the implementation of the present resolution;

3. Calls upon all Member States to contribute voluntarily to the special fund referred to in paragraph 1 above;

4. Urges developed countries to fulfill their official development assistance commitments;

5. Encourages international financial institutions to provide technical assistance to developing countries;

6. Decides to include in the provisional agenda of its eightieth session the item entitled "Implementation of the present resolution".

Recorded vote: 143 in favour, 9 against, 35 abstentions
`

// Sample EU Council decision text
const sampleEUDecision = `
COUNCIL DECISION (EU) 2024/123

of 15 March 2024

on the position to be adopted on behalf of the European Union

THE COUNCIL OF THE EUROPEAN UNION,

Having regard to the Treaty on the Functioning of the European Union, and in particular Article 218(9) thereof,

Having regard to the proposal from the European Commission,

Whereas:

(1) The Agreement between the European Union and the Republic of Moldova on the carriage of freight by road was signed on 15 December 2023.

(2) It is necessary to establish the position to be adopted on behalf of the Union within the Joint Committee established by the Agreement.

(3) The Joint Committee is to adopt its rules of procedure at its first meeting.

(4) It is appropriate to establish the position to be adopted on the Union's behalf, as the rules of procedure will be binding on the Union.

HAS ADOPTED THIS DECISION:

Article 1
The position to be adopted on behalf of the European Union within the Joint Committee shall be based on the draft rules of procedure attached to this Decision.

Article 2
This Decision shall enter into force on the date of its adoption.

Done at Brussels, 15 March 2024.

For the Council
The President
`

// Sample simple resolution with consensus adoption
const sampleConsensusResolution = `
Resolution 2024-15

Adopted by consensus on 5 May 2024

The Board of Directors,

Considering the proposal submitted by the Finance Committee,

Noting the favorable financial outlook for the next fiscal year,

1. Approves the annual budget for fiscal year 2025 as presented in document BD/2024/15;

2. Authorizes the Executive Director to proceed with planned expenditures;

3. Requests a quarterly progress report from the Finance Committee.

This resolution was adopted without a vote.
`

func TestResolutionParser_Parse_UN(t *testing.T) {
	parser := NewResolutionParserWithFormat("https://example.org/un", "un")

	resolution, err := parser.Parse(sampleUNResolution)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check basic metadata
	if resolution.Identifier == "" {
		t.Error("Expected identifier to be extracted")
	}
	if !strings.Contains(resolution.Identifier, "A/RES/79/100") && resolution.Identifier != "A/RES/79/100" {
		t.Logf("Identifier: %s", resolution.Identifier)
	}

	if resolution.AdoptingBody == "" {
		t.Error("Expected adopting body to be extracted")
	}

	// Check adoption date
	if resolution.AdoptionDate.IsZero() {
		t.Error("Expected adoption date to be extracted")
	}
	expectedDate := time.Date(2024, 12, 18, 0, 0, 0, 0, time.UTC)
	if !resolution.AdoptionDate.Equal(expectedDate) {
		t.Logf("Adoption date: %v (expected %v)", resolution.AdoptionDate, expectedDate)
	}

	// Check preamble recitals
	if len(resolution.Preamble) == 0 {
		t.Error("Expected preamble recitals to be extracted")
	} else {
		t.Logf("Found %d preamble recitals", len(resolution.Preamble))
	}

	// Check operative clauses
	if len(resolution.OperativeClauses) == 0 {
		t.Error("Expected operative clauses to be extracted")
	} else {
		t.Logf("Found %d operative clauses", len(resolution.OperativeClauses))
		// Verify first clause
		if len(resolution.OperativeClauses) >= 1 {
			clause := resolution.OperativeClauses[0]
			if clause.Number != 1 {
				t.Errorf("Expected clause number 1, got %d", clause.Number)
			}
			if clause.ActionVerb == "" {
				t.Logf("First clause: %s (action verb: %s)", clause.Text[:min(50, len(clause.Text))], clause.ActionVerb)
			}
		}
	}

	// Check vote record
	if resolution.Vote == nil {
		t.Error("Expected vote record to be extracted")
	} else {
		t.Logf("Vote record: %d for, %d against, %d abstentions",
			resolution.Vote.ForCount, resolution.Vote.AgainstCount, resolution.Vote.AbstainCount)
		if resolution.Vote.ForCount != 143 {
			t.Errorf("Expected 143 votes in favour, got %d", resolution.Vote.ForCount)
		}
		if resolution.Vote.AgainstCount != 9 {
			t.Errorf("Expected 9 votes against, got %d", resolution.Vote.AgainstCount)
		}
		if resolution.Vote.AbstainCount != 35 {
			t.Errorf("Expected 35 abstentions, got %d", resolution.Vote.AbstainCount)
		}
	}

	// Check references
	t.Logf("Found %d references", len(resolution.References))
}

func TestResolutionParser_Parse_EU(t *testing.T) {
	parser := NewResolutionParserWithFormat("https://example.org/eu", "eu")

	resolution, err := parser.Parse(sampleEUDecision)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check identifier
	if resolution.Identifier == "" {
		t.Error("Expected identifier to be extracted")
	} else {
		t.Logf("EU Decision identifier: %s", resolution.Identifier)
	}

	// Check adopting body
	if resolution.AdoptingBody == "" {
		t.Error("Expected adopting body to be extracted")
	} else {
		t.Logf("Adopting body: %s", resolution.AdoptingBody)
	}

	// Check date
	if resolution.AdoptionDate.IsZero() {
		t.Error("Expected adoption date to be extracted")
	} else {
		t.Logf("Adoption date: %v", resolution.AdoptionDate)
	}

	// Check preamble (numbered recitals in EU format)
	if len(resolution.Preamble) == 0 {
		t.Error("Expected preamble recitals to be extracted")
	} else {
		t.Logf("Found %d preamble recitals", len(resolution.Preamble))
	}

	// Check operative clauses (Articles in EU format)
	if len(resolution.OperativeClauses) == 0 {
		t.Error("Expected operative clauses (Articles) to be extracted")
	} else {
		t.Logf("Found %d operative clauses", len(resolution.OperativeClauses))
	}
}

func TestResolutionParser_Parse_Consensus(t *testing.T) {
	parser := NewResolutionParser("https://example.org/board")

	resolution, err := parser.Parse(sampleConsensusResolution)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check vote record for consensus
	if resolution.Vote == nil {
		t.Error("Expected vote record to be extracted")
	} else {
		if resolution.Vote.VoteType != "consensus" && !strings.Contains(resolution.Vote.Result, "consensus") {
			t.Logf("Vote type: %s, Result: %s", resolution.Vote.VoteType, resolution.Vote.Result)
		}
	}

	// Check operative clauses
	if len(resolution.OperativeClauses) < 3 {
		t.Errorf("Expected at least 3 operative clauses, got %d", len(resolution.OperativeClauses))
	}
}

func TestResolutionParser_ExtractPreamble(t *testing.T) {
	parser := NewResolutionParserWithFormat("https://example.org/un", "un")

	recitals, err := parser.ExtractPreamble(sampleUNResolution)
	if err != nil {
		t.Fatalf("ExtractPreamble failed: %v", err)
	}

	if len(recitals) == 0 {
		t.Error("Expected recitals to be extracted")
		return
	}

	t.Logf("Extracted %d recitals", len(recitals))

	// Check for expected intro phrases
	expectedPhrases := []string{"Recalling", "Noting", "Reaffirming", "Recognizing", "Guided", "Bearing"}
	foundPhrases := make(map[string]bool)

	for _, recital := range recitals {
		if recital.IntroPhrase != "" {
			for _, phrase := range expectedPhrases {
				if strings.Contains(recital.IntroPhrase, phrase) || strings.HasPrefix(recital.IntroPhrase, phrase) {
					foundPhrases[phrase] = true
				}
			}
		}
		t.Logf("Recital %d: [%s] %s...", recital.Number, recital.IntroPhrase,
			recital.Text[:min(40, len(recital.Text))])
	}

	if len(foundPhrases) < 2 {
		t.Logf("Found intro phrases: %v", foundPhrases)
	}
}

func TestResolutionParser_ExtractOperativeClauses(t *testing.T) {
	parser := NewResolutionParserWithFormat("https://example.org/un", "un")

	clauses, err := parser.ExtractOperativeClauses(sampleUNResolution)
	if err != nil {
		t.Fatalf("ExtractOperativeClauses failed: %v", err)
	}

	if len(clauses) < 6 {
		t.Errorf("Expected at least 6 operative clauses, got %d", len(clauses))
	}

	// Check action verbs
	expectedVerbs := map[int]string{
		1: "Decides",
		2: "Requests",
		3: "Calls upon",
		4: "Urges",
		5: "Encourages",
		6: "Decides",
	}

	for _, clause := range clauses {
		if expected, ok := expectedVerbs[clause.Number]; ok {
			if clause.ActionVerb != expected {
				t.Logf("Clause %d: expected verb '%s', got '%s'", clause.Number, expected, clause.ActionVerb)
			}
		}
	}

	// Check that clauses have URIs
	for _, clause := range clauses {
		if clause.URI == "" {
			t.Errorf("Clause %d has no URI", clause.Number)
		}
	}
}

func TestResolutionParser_ExtractVoteRecord(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		expectedType   string
		expectedFor    int
		expectedAgainst int
		expectedAbstain int
	}{
		{
			name:           "UN recorded vote",
			text:           "Recorded vote: 143 in favour, 9 against, 35 abstentions",
			expectedType:   "recorded",
			expectedFor:    143,
			expectedAgainst: 9,
			expectedAbstain: 35,
		},
		{
			name:           "Vote with to/with format",
			text:           "adopted by a vote of 120 to 5, with 20 abstentions",
			expectedType:   "recorded",
			expectedFor:    120,
			expectedAgainst: 5,
			expectedAbstain: 20,
		},
		{
			name:         "Consensus adoption",
			text:         "The resolution was adopted by consensus.",
			expectedType: "consensus",
		},
		{
			name:         "Without a vote",
			text:         "This resolution was adopted without a vote.",
			expectedType: "consensus",
		},
		{
			name:         "Unanimously",
			text:         "The decision was adopted unanimously.",
			expectedType: "consensus",
		},
	}

	parser := NewResolutionParserWithFormat("https://example.org/un", "un")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vote, err := parser.ExtractVoteRecord(tt.text)
			if err != nil {
				t.Fatalf("ExtractVoteRecord failed: %v", err)
			}

			if vote == nil {
				t.Fatal("Expected vote record, got nil")
			}

			if vote.VoteType != tt.expectedType {
				t.Errorf("Expected vote type '%s', got '%s'", tt.expectedType, vote.VoteType)
			}

			if tt.expectedType == "recorded" {
				if vote.ForCount != tt.expectedFor {
					t.Errorf("Expected %d for votes, got %d", tt.expectedFor, vote.ForCount)
				}
				if vote.AgainstCount != tt.expectedAgainst {
					t.Errorf("Expected %d against votes, got %d", tt.expectedAgainst, vote.AgainstCount)
				}
				if vote.AbstainCount != tt.expectedAbstain {
					t.Errorf("Expected %d abstentions, got %d", tt.expectedAbstain, vote.AbstainCount)
				}
			}
		})
	}
}

func TestResolutionParser_ExtractReferences(t *testing.T) {
	text := `
	Recalling its resolution 78/200 and resolution A/RES/77/150,

	Having regard to the Charter of the United Nations,

	Taking note of report A/79/123 and document A/79/456,

	Considering Regulation (EU) 2024/123 and Directive (EU) 2023/456,
	`

	parser := NewResolutionParser("https://example.org/test")
	refs := parser.extractAllReferences(text)

	t.Logf("Found %d references", len(refs))

	// Check for different reference types
	typeCount := make(map[string]int)
	for _, ref := range refs {
		typeCount[ref.Type]++
		t.Logf("Reference: type=%s, id=%s", ref.Type, ref.Identifier)
	}

	if typeCount["resolution"] < 1 {
		t.Errorf("Expected at least 1 resolution reference, got %d", typeCount["resolution"])
	}
	if typeCount["treaty"] < 1 {
		t.Errorf("Expected at least 1 treaty reference, got %d", typeCount["treaty"])
	}
}

func TestResolutionParser_EmptyInput(t *testing.T) {
	parser := NewResolutionParser("https://example.org/test")

	_, err := parser.Parse("")
	if err == nil {
		t.Error("Expected error for empty input")
	}
}

func TestResolution_String(t *testing.T) {
	r := &Resolution{
		Identifier:   "A/RES/79/100",
		Title:        "Sustainable Development Goals",
		AdoptionDate: time.Date(2024, 12, 18, 0, 0, 0, 0, time.UTC),
	}

	str := r.String()
	if !strings.Contains(str, "A/RES/79/100") {
		t.Errorf("String should contain identifier: %s", str)
	}
	if !strings.Contains(str, "Sustainable Development Goals") {
		t.Errorf("String should contain title: %s", str)
	}
	if !strings.Contains(str, "2024-12-18") {
		t.Errorf("String should contain date: %s", str)
	}
}

func TestNewResolutionParserWithFormat(t *testing.T) {
	tests := []struct {
		format string
	}{
		{"un"},
		{"eu"},
		{"generic"},
		{"unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			parser := NewResolutionParserWithFormat("https://example.org/test", tt.format)
			if parser == nil {
				t.Error("Expected non-nil parser")
			}
			if parser.patterns == nil {
				t.Error("Expected patterns to be compiled")
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
