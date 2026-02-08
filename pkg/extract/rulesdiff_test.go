package extract

import (
	"strings"
	"testing"
)

func TestChangeType_String(t *testing.T) {
	tests := []struct {
		input    ChangeType
		expected string
	}{
		{ChangeAdded, "ADDED"},
		{ChangeRemoved, "REMOVED"},
		{ChangeModified, "MODIFIED"},
		{ChangeUnchanged, "UNCHANGED"},
	}

	for _, tc := range tests {
		result := tc.input.String()
		if result != tc.expected {
			t.Errorf("ChangeType(%d).String(): expected %q, got %q", tc.input, tc.expected, result)
		}
	}
}

func TestNewRulesDiffer(t *testing.T) {
	baseText := `
RULE I
THE SPEAKER

1. The Speaker shall take the Chair.
`
	targetText := `
RULE I
THE SPEAKER

1. The Speaker shall take the Chair on every legislative day.
`

	differ := NewRulesDiffer(baseText, targetText)
	if differ == nil {
		t.Fatal("Expected differ to be non-nil")
	}
	if differ.baseSearcher == nil {
		t.Error("Expected baseSearcher to be non-nil")
	}
	if differ.targetSearcher == nil {
		t.Error("Expected targetSearcher to be non-nil")
	}
}

func TestCompare_AddedClause(t *testing.T) {
	baseText := `
RULE I
THE SPEAKER

1. The Speaker shall take the Chair.
`
	targetText := `
RULE I
THE SPEAKER

1. The Speaker shall take the Chair.

2. The Speaker shall preserve order.
`

	differ := NewRulesDiffer(baseText, targetText)
	report := differ.Compare("118th", "119th")

	if report.TotalClausesAdded < 1 {
		t.Errorf("Expected at least 1 added clause, got %d", report.TotalClausesAdded)
	}

	// Find the added clause
	found := false
	for _, rc := range report.RuleChanges {
		for _, cc := range rc.ClauseChanges {
			if cc.Type == ChangeAdded && cc.Clause == "2" {
				found = true
			}
		}
	}
	if !found {
		t.Error("Expected to find added clause 2")
	}
}

func TestCompare_RemovedClause(t *testing.T) {
	baseText := `
RULE I
THE SPEAKER

1. The Speaker shall take the Chair.

2. The Speaker shall preserve order.
`
	targetText := `
RULE I
THE SPEAKER

1. The Speaker shall take the Chair.
`

	differ := NewRulesDiffer(baseText, targetText)
	report := differ.Compare("118th", "119th")

	if report.TotalClausesRemoved < 1 {
		t.Errorf("Expected at least 1 removed clause, got %d", report.TotalClausesRemoved)
	}
}

func TestCompare_ModifiedClause(t *testing.T) {
	baseText := `
RULE I
THE SPEAKER

1. The Speaker shall take the Chair.
`
	targetText := `
RULE I
THE SPEAKER

1. The Speaker shall take the Chair on every legislative day precisely at the hour.
`

	differ := NewRulesDiffer(baseText, targetText)
	report := differ.Compare("118th", "119th")

	if report.TotalClausesModified < 1 {
		t.Errorf("Expected at least 1 modified clause, got %d", report.TotalClausesModified)
	}

	// Check that similarity is calculated
	for _, rc := range report.RuleChanges {
		for _, cc := range rc.ClauseChanges {
			if cc.Type == ChangeModified {
				if cc.SimilarityScore <= 0 || cc.SimilarityScore >= 100 {
					t.Errorf("Expected similarity score between 1-99, got %d", cc.SimilarityScore)
				}
			}
		}
	}
}

func TestCompare_UnchangedRule(t *testing.T) {
	text := `
RULE I
THE SPEAKER

1. The Speaker shall take the Chair.
`

	differ := NewRulesDiffer(text, text)
	report := differ.Compare("118th", "119th")

	// Unchanged rules should not appear in report
	if len(report.RuleChanges) > 0 {
		t.Errorf("Expected no rule changes for identical text, got %d", len(report.RuleChanges))
	}
}

func TestCompare_MultipleRules(t *testing.T) {
	baseText := `
RULE I
THE SPEAKER

1. The Speaker shall take the Chair.

RULE II
OTHER OFFICERS

1. There shall be elected a Clerk.
`
	targetText := `
RULE I
THE SPEAKER

1. The Speaker shall take the Chair on every legislative day.

RULE II
OTHER OFFICERS

1. There shall be elected a Clerk.

2. There shall be elected a Sergeant at Arms.
`

	differ := NewRulesDiffer(baseText, targetText)
	report := differ.Compare("118th", "119th")

	if report.RulesModified < 1 {
		t.Errorf("Expected at least 1 modified rule, got %d", report.RulesModified)
	}

	if report.TotalClausesModified < 1 {
		t.Errorf("Expected at least 1 modified clause, got %d", report.TotalClausesModified)
	}

	if report.TotalClausesAdded < 1 {
		t.Errorf("Expected at least 1 added clause, got %d", report.TotalClausesAdded)
	}
}

func TestCalculateSimilarity(t *testing.T) {
	tests := []struct {
		text1    string
		text2    string
		minSim   int
		maxSim   int
	}{
		{"hello world", "hello world", 100, 100},
		{"hello world", "hello there", 30, 60},
		{"the quick brown fox", "the slow brown fox", 60, 90},
		{"completely different text", "nothing in common here", 0, 30},
	}

	for _, tc := range tests {
		sim := calculateSimilarity(tc.text1, tc.text2)
		if sim < tc.minSim || sim > tc.maxSim {
			t.Errorf("calculateSimilarity(%q, %q): expected %d-%d, got %d",
				tc.text1, tc.text2, tc.minSim, tc.maxSim, sim)
		}
	}
}

func TestGenerateChangeSummary(t *testing.T) {
	tests := []struct {
		similarity int
		contains   string
	}{
		{85, "Minor"},
		{70, "Moderate"},
		{50, "Substantial"},
		{20, "Major"},
	}

	for _, tc := range tests {
		summary := generateChangeSummary("base text", "target text", tc.similarity)
		if !strings.Contains(summary, tc.contains) {
			t.Errorf("generateChangeSummary with similarity %d: expected to contain %q, got %q",
				tc.similarity, tc.contains, summary)
		}
	}
}

func TestRulesDiffReport_String(t *testing.T) {
	baseText := `
RULE I
THE SPEAKER

1. The Speaker shall take the Chair.
`
	targetText := `
RULE I
THE SPEAKER

1. The Speaker shall take the Chair on every legislative day.

2. The Speaker shall preserve order.
`

	differ := NewRulesDiffer(baseText, targetText)
	report := differ.Compare("118th Congress", "119th Congress")

	output := report.String()

	if !strings.Contains(output, "118th Congress") {
		t.Error("Expected output to contain base version")
	}
	if !strings.Contains(output, "119th Congress") {
		t.Error("Expected output to contain target version")
	}
	if !strings.Contains(output, "Rule I") {
		t.Error("Expected output to contain 'Rule I'")
	}
}

func TestRulesDiffReport_ToJSON(t *testing.T) {
	baseText := `
RULE I
THE SPEAKER

1. The Speaker shall take the Chair.
`
	targetText := `
RULE I
THE SPEAKER

1. The Speaker shall take the Chair.

2. New clause.
`

	differ := NewRulesDiffer(baseText, targetText)
	report := differ.Compare("118th", "119th")

	jsonData, err := report.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if !strings.Contains(string(jsonData), `"base_version"`) {
		t.Error("JSON should contain base_version field")
	}
	if !strings.Contains(string(jsonData), `"ADDED"`) {
		t.Error("JSON should contain ADDED change type")
	}
}

func TestGetSignificantChanges(t *testing.T) {
	baseText := `
RULE I
THE SPEAKER

1. The Speaker shall take the Chair.

2. Minor text here.
`
	targetText := `
RULE I
THE SPEAKER

1. Completely different text for clause one that has nothing in common.

2. Minor text here updated slightly.

3. New clause added.
`

	differ := NewRulesDiffer(baseText, targetText)
	report := differ.Compare("118th", "119th")

	// Get only significant changes (< 80% similar)
	significant := report.GetSignificantChanges(80)

	if len(significant) < 1 {
		t.Error("Expected at least 1 significant change")
	}

	// Added clauses should always be included
	hasAdded := false
	for _, c := range significant {
		if c.Type == ChangeAdded {
			hasAdded = true
		}
	}
	if !hasAdded {
		t.Error("Expected added clauses to be in significant changes")
	}
}

func TestCompareClauseNumbers(t *testing.T) {
	tests := []struct {
		a, b     string
		expected bool
	}{
		{"1", "2", true},
		{"2", "1", false},
		{"10", "2", false},
		{"1", "10", true},
		{"a", "b", true},
	}

	for _, tc := range tests {
		result := compareClauseNumbers(tc.a, tc.b)
		if result != tc.expected {
			t.Errorf("compareClauseNumbers(%q, %q): expected %v, got %v",
				tc.a, tc.b, tc.expected, result)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"very long text here", 10, "very lo..."},
		{"exactly10!", 10, "exactly10!"},
	}

	for _, tc := range tests {
		result := truncate(tc.input, tc.maxLen)
		if result != tc.expected {
			t.Errorf("truncate(%q, %d): expected %q, got %q",
				tc.input, tc.maxLen, tc.expected, result)
		}
	}
}
