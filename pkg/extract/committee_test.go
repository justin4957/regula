package extract

import (
	"os"
	"strings"
	"testing"
)

func TestNewCommitteeJurisdictionExtractor(t *testing.T) {
	extractor := NewCommitteeJurisdictionExtractor()
	if extractor == nil {
		t.Fatal("Expected extractor to be non-nil")
	}
	if extractor.committeePattern == nil {
		t.Error("Expected committeePattern to be compiled")
	}
	if extractor.topicPattern == nil {
		t.Error("Expected topicPattern to be compiled")
	}
}

func TestExtractFromRuleX_Agriculture(t *testing.T) {
	extractor := NewCommitteeJurisdictionExtractor()

	// Sample Rule X text for Committee on Agriculture
	text := `
(a) Committee on Agriculture.
(1) Adulteration of seeds, insect pests, and protection of birds and animals in forest reserves.
(2) Agriculture generally.
(3) Agricultural and industrial chemistry.
(4) Agricultural colleges and experiment stations.
`

	committees := extractor.ExtractFromRuleX(text)
	if len(committees) != 1 {
		t.Fatalf("Expected 1 committee, got %d", len(committees))
	}

	committee := committees[0]
	if committee.Name != "Committee on Agriculture" {
		t.Errorf("Expected 'Committee on Agriculture', got %q", committee.Name)
	}
	if committee.ShortName != "Agriculture" {
		t.Errorf("Expected 'Agriculture', got %q", committee.ShortName)
	}
	if committee.Letter != "a" {
		t.Errorf("Expected letter 'a', got %q", committee.Letter)
	}
	if len(committee.Topics) < 4 {
		t.Errorf("Expected at least 4 topics, got %d", len(committee.Topics))
	}
	if committee.SourceClause != "Rule X, clause 1(a)" {
		t.Errorf("Expected 'Rule X, clause 1(a)', got %q", committee.SourceClause)
	}
}

func TestExtractFromRuleX_HomelandSecurity(t *testing.T) {
	extractor := NewCommitteeJurisdictionExtractor()

	// Sample Rule X text for Committee on Homeland Security with sub-topics
	text := `
(j) Committee on Homeland Security.
(1) Overall homeland security policy.
(2) Organization, administration, and general management of the Department of Homeland Security.
(3) Functions of the Department of Homeland Security relating to the following:
(A) Border and port security (except immigration policy and non-border enforcement).
(B) Customs (except customs revenue).
(C) Integration, analysis, and dissemination of homeland security information.
(D) Domestic preparedness for and collective response to terrorism.
(E) Research and development.
(F) Transportation security.
(G) Cybersecurity.
`

	committees := extractor.ExtractFromRuleX(text)
	if len(committees) != 1 {
		t.Fatalf("Expected 1 committee, got %d", len(committees))
	}

	committee := committees[0]
	if committee.Name != "Committee on Homeland Security" {
		t.Errorf("Expected 'Committee on Homeland Security', got %q", committee.Name)
	}
	if committee.Letter != "j" {
		t.Errorf("Expected letter 'j', got %q", committee.Letter)
	}

	// Check that we captured the topics
	if len(committee.Topics) < 3 {
		t.Fatalf("Expected at least 3 topics, got %d", len(committee.Topics))
	}

	// Check that topic 3 has Cybersecurity in its text
	found := false
	for _, topic := range committee.Topics {
		if strings.Contains(strings.ToLower(topic.Text), "cybersecurity") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find 'Cybersecurity' in topics")
	}
}

func TestExtractFromRuleX_MultipleCommittees(t *testing.T) {
	extractor := NewCommitteeJurisdictionExtractor()

	text := `
(a) Committee on Agriculture.
(1) Agriculture generally.
(2) Rural development.

(b) Committee on Appropriations.
(1) Appropriation of the revenue for the support of the Government.
(2) Rescissions of appropriations contained in appropriation Acts.

(c) Committee on Armed Services.
(1) Common defense generally.
(2) The Department of Defense generally.
`

	committees := extractor.ExtractFromRuleX(text)
	if len(committees) != 3 {
		t.Fatalf("Expected 3 committees, got %d", len(committees))
	}

	expectedNames := []string{"Committee on Agriculture", "Committee on Appropriations", "Committee on Armed Services"}
	for i, expected := range expectedNames {
		if committees[i].Name != expected {
			t.Errorf("Committee %d: expected %q, got %q", i, expected, committees[i].Name)
		}
	}
}

func TestSearchCommitteeByTopic_Cybersecurity(t *testing.T) {
	committees := []CommitteeJurisdiction{
		{
			Name:         "Committee on Homeland Security",
			ShortName:    "Homeland Security",
			Letter:       "j",
			SourceClause: "Rule X, clause 1(j)",
			Topics: []JurisdictionTopic{
				{Number: "1", Text: "Overall homeland security policy."},
				{Number: "3", Text: "Functions of the Department of Homeland Security relating to the following:", SubTopics: []JurisdictionTopic{
					{Number: "G", Text: "Cybersecurity."},
				}},
			},
		},
		{
			Name:         "Committee on Agriculture",
			ShortName:    "Agriculture",
			Letter:       "a",
			SourceClause: "Rule X, clause 1(a)",
			Topics: []JurisdictionTopic{
				{Number: "1", Text: "Agriculture generally."},
			},
		},
	}

	matches := SearchCommitteeByTopic(committees, "cybersecurity")
	if len(matches) != 1 {
		t.Fatalf("Expected 1 match, got %d", len(matches))
	}

	match := matches[0]
	if match.Committee.Name != "Committee on Homeland Security" {
		t.Errorf("Expected 'Committee on Homeland Security', got %q", match.Committee.Name)
	}
	if !strings.Contains(match.SourceRef, "(3)(G)") {
		t.Errorf("Expected source ref to contain '(3)(G)', got %q", match.SourceRef)
	}
}

func TestSearchCommitteeByTopic_Energy(t *testing.T) {
	committees := []CommitteeJurisdiction{
		{
			Name:         "Committee on Energy and Commerce",
			ShortName:    "Energy and Commerce",
			Letter:       "f",
			SourceClause: "Rule X, clause 1(f)",
			Topics: []JurisdictionTopic{
				{Number: "6", Text: "Exploration, production, storage, supply, marketing, pricing, and regulation of energy resources."},
				{Number: "11", Text: "National energy policy generally."},
			},
		},
		{
			Name:         "Committee on Science, Space, and Technology",
			ShortName:    "Science, Space, and Technology",
			Letter:       "p",
			SourceClause: "Rule X, clause 1(p)",
			Topics: []JurisdictionTopic{
				{Number: "1", Text: "All energy research, development, and demonstration."},
			},
		},
	}

	matches := SearchCommitteeByTopic(committees, "energy")
	if len(matches) != 3 {
		t.Errorf("Expected 3 matches for 'energy', got %d", len(matches))
	}

	// Both committees should have matches
	committeeNames := make(map[string]bool)
	for _, m := range matches {
		committeeNames[m.Committee.Name] = true
	}
	if !committeeNames["Committee on Energy and Commerce"] {
		t.Error("Expected Committee on Energy and Commerce in matches")
	}
	if !committeeNames["Committee on Science, Space, and Technology"] {
		t.Error("Expected Committee on Science, Space, and Technology in matches")
	}
}

func TestSearchCommitteeByTopic_CaseInsensitive(t *testing.T) {
	committees := []CommitteeJurisdiction{
		{
			Name:         "Committee on Homeland Security",
			ShortName:    "Homeland Security",
			Letter:       "j",
			SourceClause: "Rule X, clause 1(j)",
			Topics: []JurisdictionTopic{
				{Number: "G", Text: "Cybersecurity."},
			},
		},
	}

	// Test case insensitivity
	for _, query := range []string{"CYBERSECURITY", "CyberSecurity", "cybersecurity"} {
		matches := SearchCommitteeByTopic(committees, query)
		if len(matches) != 1 {
			t.Errorf("Query %q: expected 1 match, got %d", query, len(matches))
		}
	}
}

func TestSearchCommitteeByName(t *testing.T) {
	committees := []CommitteeJurisdiction{
		{Name: "Committee on Agriculture", ShortName: "Agriculture"},
		{Name: "Committee on Homeland Security", ShortName: "Homeland Security"},
		{Name: "Committee on the Judiciary", ShortName: "Judiciary"},
	}

	tests := []struct {
		query    string
		expected string
	}{
		{"Agriculture", "Committee on Agriculture"},
		{"homeland", "Committee on Homeland Security"},
		{"judiciary", "Committee on the Judiciary"},
		{"Nonexistent", ""},
	}

	for _, tc := range tests {
		result := SearchCommitteeByName(committees, tc.query)
		if tc.expected == "" {
			if result != nil {
				t.Errorf("Query %q: expected nil, got %q", tc.query, result.Name)
			}
		} else {
			if result == nil {
				t.Errorf("Query %q: expected %q, got nil", tc.query, tc.expected)
			} else if result.Name != tc.expected {
				t.Errorf("Query %q: expected %q, got %q", tc.query, tc.expected, result.Name)
			}
		}
	}
}

func TestGetJurisdictions(t *testing.T) {
	committee := CommitteeJurisdiction{
		Name: "Committee on Homeland Security",
		Topics: []JurisdictionTopic{
			{Number: "1", Text: "Overall homeland security policy."},
			{Number: "3", Text: "Functions relating to:", SubTopics: []JurisdictionTopic{
				{Number: "A", Text: "Border security."},
				{Number: "B", Text: "Cybersecurity."},
			}},
		},
	}

	jurisdictions := committee.GetJurisdictions()
	if len(jurisdictions) != 4 {
		t.Errorf("Expected 4 jurisdictions, got %d", len(jurisdictions))
	}
}

func TestCleanText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  multiple   spaces  ", "multiple spaces"},
		{"line\nbreak", "line break"},
		{"tab\there", "tab here"},
		{"  leading and trailing  ", "leading and trailing"},
	}

	for _, tc := range tests {
		result := cleanText(tc.input)
		if result != tc.expected {
			t.Errorf("cleanText(%q): expected %q, got %q", tc.input, tc.expected, result)
		}
	}
}

func TestExtractFromRuleX_Integration(t *testing.T) {
	// Load actual House Rules file
	data, err := os.ReadFile("../../testdata/house-rules-119th.txt")
	if err != nil {
		t.Skipf("Skipping integration test: %v", err)
	}

	text := string(data)
	extractor := NewCommitteeJurisdictionExtractor()

	// Find Rule X section
	ruleXStart := strings.Index(text, "RULE X")
	if ruleXStart == -1 {
		t.Skip("Could not find RULE X in document")
	}

	// Find next RULE to delimit
	ruleXEnd := strings.Index(text[ruleXStart+6:], "RULE XI")
	if ruleXEnd == -1 {
		ruleXEnd = len(text) - ruleXStart
	} else {
		ruleXEnd += ruleXStart + 6
	}

	ruleXText := text[ruleXStart:ruleXEnd]

	committees := extractor.ExtractFromRuleX(ruleXText)

	// Should extract at least 10 committees
	if len(committees) < 10 {
		t.Errorf("Expected at least 10 committees, got %d", len(committees))
	}

	// Check for specific committees (note: "the" may or may not be present depending on source)
	expectedCommittees := []string{
		"Committee on Agriculture",
		"Committee on Appropriations",
		"Committee on Armed Services",
		"Committee on Energy and Commerce",
		"Committee on Homeland Security",
		"Committee on Judiciary", // or "Committee on the Judiciary" - both valid
	}

	for _, expected := range expectedCommittees {
		found := false
		for _, c := range committees {
			if c.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find %q", expected)
		}
	}

	// Search for cybersecurity
	matches := SearchCommitteeByTopic(committees, "cybersecurity")
	if len(matches) == 0 {
		t.Error("Expected to find matches for 'cybersecurity'")
	} else {
		// Should be Committee on Homeland Security
		found := false
		for _, m := range matches {
			if strings.Contains(m.Committee.Name, "Homeland Security") {
				found = true
			}
		}
		if !found {
			t.Error("Expected 'cybersecurity' to match Committee on Homeland Security")
		}
	}
}
