package extract

import (
	"os"
	"strings"
	"testing"
)

func TestNewPathfinder(t *testing.T) {
	pathfinder := NewPathfinder(nil)
	if pathfinder == nil {
		t.Fatal("Expected pathfinder to be non-nil")
	}
	if len(pathfinder.scenarios) == 0 {
		t.Error("Expected scenarios to be populated")
	}
}

func TestGetActions(t *testing.T) {
	pathfinder := NewPathfinder(nil)
	actions := pathfinder.GetActions()

	if len(actions) == 0 {
		t.Fatal("Expected at least one action")
	}

	// Check for expected actions
	expectedActions := []string{"introduce-bill", "amend-bill", "vote-on-bill", "quorum-call", "debate"}
	for _, expected := range expectedActions {
		found := false
		for _, action := range actions {
			if action == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected action %q to be in list", expected)
		}
	}
}

func TestGetScenario(t *testing.T) {
	pathfinder := NewPathfinder(nil)

	tests := []struct {
		action   string
		expected string
	}{
		{"introduce-bill", "Introduce a Bill"},
		{"amend-bill", "Propose an Amendment"},
		{"vote-on-bill", "Vote on a Bill"},
		{"quorum-call", "Establish a Quorum"},
		{"nonexistent", ""},
	}

	for _, tc := range tests {
		scenario, ok := pathfinder.GetScenario(tc.action)
		if tc.expected == "" {
			if ok {
				t.Errorf("GetScenario(%q): expected not found, got %q", tc.action, scenario.Title)
			}
		} else {
			if !ok {
				t.Errorf("GetScenario(%q): expected %q, got not found", tc.action, tc.expected)
			} else if scenario.Title != tc.expected {
				t.Errorf("GetScenario(%q): expected %q, got %q", tc.action, tc.expected, scenario.Title)
			}
		}
	}
}

func TestGetScenario_Aliases(t *testing.T) {
	pathfinder := NewPathfinder(nil)

	aliases := map[string]string{
		"introduce":  "Introduce a Bill",
		"bill":       "Introduce a Bill",
		"amendment":  "Propose an Amendment",
		"amend":      "Propose an Amendment",
		"vote":       "Vote on a Bill",
		"voting":     "Vote on a Bill",
		"quorum":     "Establish a Quorum",
		"mtr":        "Motion to Recommit",
		"veto":       "Override a Presidential Veto",
		"conference": "Request a Conference",
	}

	for alias, expectedTitle := range aliases {
		scenario, ok := pathfinder.GetScenario(alias)
		if !ok {
			t.Errorf("Alias %q: expected to find scenario", alias)
			continue
		}
		if scenario.Title != expectedTitle {
			t.Errorf("Alias %q: expected %q, got %q", alias, expectedTitle, scenario.Title)
		}
	}
}

func TestNavigate_IntroduceBill(t *testing.T) {
	pathfinder := NewPathfinder(nil)

	path, err := pathfinder.Navigate("introduce-bill")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	if path.Title != "Introduce a Bill" {
		t.Errorf("Expected title 'Introduce a Bill', got %q", path.Title)
	}

	if len(path.Steps) < 5 {
		t.Errorf("Expected at least 5 steps, got %d", len(path.Steps))
	}

	// Check first step
	if path.Steps[0].Rule != "XII" {
		t.Errorf("Expected first step to be Rule XII, got Rule %s", path.Steps[0].Rule)
	}
	if path.Steps[0].StepNumber != 1 {
		t.Errorf("Expected step number 1, got %d", path.Steps[0].StepNumber)
	}
}

func TestNavigate_UnknownAction(t *testing.T) {
	pathfinder := NewPathfinder(nil)

	_, err := pathfinder.Navigate("unknown-action")
	if err == nil {
		t.Error("Expected error for unknown action")
	}
	if !strings.Contains(err.Error(), "unknown action") {
		t.Errorf("Expected 'unknown action' error, got: %v", err)
	}
}

func TestProceduralPath_String(t *testing.T) {
	pathfinder := NewPathfinder(nil)

	path, err := pathfinder.Navigate("vote-on-bill")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	output := path.String()

	if !strings.Contains(output, "Vote on a Bill") {
		t.Error("Expected output to contain title")
	}
	if !strings.Contains(output, "Step 1:") {
		t.Error("Expected output to contain 'Step 1:'")
	}
	if !strings.Contains(output, "Rule XX") {
		t.Error("Expected output to contain 'Rule XX'")
	}
}

func TestNormalizeAction(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"introduce-bill", "introduce-bill"},
		{"INTRODUCE-BILL", "introduce-bill"},
		{"introduce bill", "introduce-bill"},
		{"introduce_bill", "introduce-bill"},
		{"introduce", "introduce-bill"},
		{"bill", "introduce-bill"},
		{"vote", "vote-on-bill"},
		{"mtr", "recommit"},
	}

	for _, tc := range tests {
		result := normalizeAction(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeAction(%q): expected %q, got %q", tc.input, tc.expected, result)
		}
	}
}

func TestExtractExcerpt(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"Short text.", 200, "Short text."},
		{"First sentence. Second sentence.", 200, "First sentence."},
		{"Very long text that exceeds the maximum length and should be truncated appropriately at a word boundary", 50, "Very long text that exceeds the maximum length..."},
	}

	for _, tc := range tests {
		result := extractExcerpt(tc.input, tc.maxLen)
		if result != tc.expected {
			t.Errorf("extractExcerpt(%q, %d): expected %q, got %q", tc.input, tc.maxLen, tc.expected, result)
		}
	}
}

func TestExtractReferences(t *testing.T) {
	text := "pursuant to clause 5 of rule XX and also see rule XVIII, clause 6"
	refs := extractReferences(text)

	if len(refs) < 1 {
		t.Fatal("Expected at least 1 reference")
	}

	// Should find Rule XX
	foundRuleXX := false
	for _, ref := range refs {
		if strings.Contains(ref, "Rule XX") {
			foundRuleXX = true
		}
	}
	if !foundRuleXX {
		t.Error("Expected to find reference to Rule XX")
	}
}

func TestIsRomanNumeral(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"I", true},
		{"XX", true},
		{"XVIII", true},
		{"XXIX", true},
		{"ABC", false},
		{"12", false},
		{"", false},
	}

	for _, tc := range tests {
		result := isRomanNumeral(tc.input)
		if result != tc.expected {
			t.Errorf("isRomanNumeral(%q): expected %v, got %v", tc.input, tc.expected, result)
		}
	}
}

func TestListScenarios(t *testing.T) {
	pathfinder := NewPathfinder(nil)
	scenarios := pathfinder.ListScenarios()

	if len(scenarios) < 5 {
		t.Errorf("Expected at least 5 scenarios, got %d", len(scenarios))
	}

	// Check that they are sorted by title
	for i := 1; i < len(scenarios); i++ {
		if scenarios[i].Title < scenarios[i-1].Title {
			t.Error("Expected scenarios to be sorted by title")
		}
	}
}

func TestNavigate_Integration(t *testing.T) {
	// Load actual House Rules file
	data, err := os.ReadFile("../../testdata/house-rules-119th.txt")
	if err != nil {
		t.Skipf("Skipping integration test: %v", err)
	}

	searcher := NewKeywordSearcher()
	searcher.ParseHouseRules(string(data))

	pathfinder := NewPathfinder(searcher)

	// Test navigate with real clauses
	path, err := pathfinder.Navigate("vote-on-bill")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	// Check that excerpts were populated from real text
	hasExcerpt := false
	for _, step := range path.Steps {
		if step.Excerpt != "" {
			hasExcerpt = true
			break
		}
	}
	if !hasExcerpt {
		t.Error("Expected at least one step to have an excerpt from real text")
	}
}

func TestNavigateWithDiscovery(t *testing.T) {
	// Load actual House Rules file
	data, err := os.ReadFile("../../testdata/house-rules-119th.txt")
	if err != nil {
		t.Skipf("Skipping integration test: %v", err)
	}

	searcher := NewKeywordSearcher()
	searcher.ParseHouseRules(string(data))

	pathfinder := NewPathfinder(searcher)

	// Test navigate with discovery
	path, err := pathfinder.NavigateWithDiscovery("quorum-call")
	if err != nil {
		t.Fatalf("NavigateWithDiscovery failed: %v", err)
	}

	// With discovery, we should have more steps than just the pre-defined ones
	scenario, _ := pathfinder.GetScenario("quorum-call")
	if len(path.Steps) < len(scenario.RuleSequence) {
		t.Errorf("Expected at least %d steps (pre-defined), got %d", len(scenario.RuleSequence), len(path.Steps))
	}
}

func TestRelatedActions(t *testing.T) {
	pathfinder := NewPathfinder(nil)

	path, err := pathfinder.Navigate("introduce-bill")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}

	if len(path.RelatedActions) == 0 {
		t.Error("Expected related actions for introduce-bill")
	}

	// Check that amend-bill is related
	found := false
	for _, related := range path.RelatedActions {
		if related == "amend-bill" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected amend-bill to be a related action")
	}
}
