package extract

import (
	"os"
	"strings"
	"testing"
)

func TestNewKeywordSearcher(t *testing.T) {
	searcher := NewKeywordSearcher()
	if searcher == nil {
		t.Fatal("Expected searcher to be non-nil")
	}
	if searcher.rulePattern == nil {
		t.Error("Expected rulePattern to be compiled")
	}
	if searcher.clausePattern == nil {
		t.Error("Expected clausePattern to be compiled")
	}
}

func TestParseHouseRules_SingleRule(t *testing.T) {
	searcher := NewKeywordSearcher()

	text := `
RULE XX
VOTING AND QUORUM CALLS

1. The Speaker shall put a question.

Quorum requirements
2. A quorum shall consist of a majority.

Automatic roll calls
3. Automatic roll calls shall be conducted.
`

	searcher.ParseHouseRules(text)
	clauses := searcher.GetClauses()

	if len(clauses) < 2 {
		t.Fatalf("Expected at least 2 clauses, got %d", len(clauses))
	}

	// Check first clause
	foundClause1 := false
	for _, c := range clauses {
		if c.Rule == "XX" && c.Clause == "1" {
			foundClause1 = true
			if !strings.Contains(c.Text, "Speaker shall put a question") {
				t.Errorf("Expected clause 1 to contain 'Speaker shall put a question', got %q", c.Text)
			}
		}
	}
	if !foundClause1 {
		t.Error("Expected to find Rule XX, clause 1")
	}
}

func TestSearch_Quorum(t *testing.T) {
	searcher := NewKeywordSearcher()

	text := `
RULE XX
VOTING AND QUORUM CALLS

1. The Speaker shall put a question.

Quorum requirements
2. A quorum shall consist of a majority of the House.

3. In the absence of a quorum, the Speaker shall call for a quorum.
`

	searcher.ParseHouseRules(text)
	matches := searcher.Search("quorum")

	if len(matches) < 2 {
		t.Fatalf("Expected at least 2 matches for 'quorum', got %d", len(matches))
	}

	// Check that matches are sorted by score (highest first)
	for i := 1; i < len(matches); i++ {
		if matches[i].Score > matches[i-1].Score {
			t.Errorf("Expected matches to be sorted by score descending")
		}
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	searcher := NewKeywordSearcher()

	text := `
RULE I
THE SPEAKER

1. The SPEAKER shall take the Chair.
`

	searcher.ParseHouseRules(text)

	for _, query := range []string{"speaker", "SPEAKER", "Speaker"} {
		matches := searcher.Search(query)
		if len(matches) == 0 {
			t.Errorf("Expected matches for query %q", query)
		}
	}
}

func TestSearchWithTemplate_Voting(t *testing.T) {
	searcher := NewKeywordSearcher()

	text := `
RULE XX
VOTING AND QUORUM CALLS

1. The Speaker shall put a question: Those in favor say Aye.

2. After a voice vote, the Speaker may conduct a roll call.

3. Electronic voting devices may be used.

4. Recorded votes shall be tallied.
`

	searcher.ParseHouseRules(text)
	matches := searcher.SearchWithTemplate("voting")

	if len(matches) == 0 {
		t.Fatal("Expected matches for 'voting' template")
	}

	// Should find clauses with vote-related terms
	foundVote := false
	for _, m := range matches {
		if strings.Contains(strings.ToLower(m.Text), "vote") ||
			strings.Contains(strings.ToLower(m.Text), "roll call") ||
			strings.Contains(strings.ToLower(m.Text), "electronic") {
			foundVote = true
			break
		}
	}
	if !foundVote {
		t.Error("Expected to find voting-related matches")
	}
}

func TestExtractKeywordContext(t *testing.T) {
	text := "The House shall meet for regular legislative business and the Speaker shall preside over all proceedings."
	keyword := "Speaker"

	context := extractKeywordContext(text, keyword, 20)

	if !strings.Contains(context, "Speaker") {
		t.Errorf("Expected context to contain keyword, got %q", context)
	}

	// Should have ellipsis at start since keyword is not at beginning
	if !strings.HasPrefix(context, "...") {
		t.Errorf("Expected context to start with ellipsis, got %q", context)
	}
}

func TestGetTemplates(t *testing.T) {
	templates := GetTemplates()

	if len(templates) == 0 {
		t.Fatal("Expected at least one template")
	}

	// Check for known templates
	expectedTemplates := []string{"voting", "quorum", "amendments", "debate"}
	for _, expected := range expectedTemplates {
		found := false
		for _, template := range templates {
			if template == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected template %q to be in list", expected)
		}
	}
}

func TestProceduralKeywords(t *testing.T) {
	// Verify all procedural keyword groups are non-empty
	for name, terms := range ProceduralKeywords {
		if len(terms) == 0 {
			t.Errorf("Expected %q to have at least one term", name)
		}
	}
}

func TestParseHouseRules_Integration(t *testing.T) {
	// Load actual House Rules file
	data, err := os.ReadFile("../../testdata/house-rules-119th.txt")
	if err != nil {
		t.Skipf("Skipping integration test: %v", err)
	}

	searcher := NewKeywordSearcher()
	searcher.ParseHouseRules(string(data))

	clauses := searcher.GetClauses()
	if len(clauses) < 50 {
		t.Errorf("Expected at least 50 clauses, got %d", len(clauses))
	}

	// Test quorum search
	quorumMatches := searcher.Search("quorum")
	if len(quorumMatches) == 0 {
		t.Error("Expected to find matches for 'quorum'")
	}

	// Rule XX should be highly ranked for quorum
	foundRuleXX := false
	for _, m := range quorumMatches[:minInt(5, len(quorumMatches))] {
		if m.Rule == "XX" {
			foundRuleXX = true
			break
		}
	}
	if !foundRuleXX {
		t.Error("Expected Rule XX to be in top results for 'quorum'")
	}

	// Test voting template
	votingMatches := searcher.SearchWithTemplate("voting")
	if len(votingMatches) == 0 {
		t.Error("Expected to find matches for 'voting' template")
	}

	// Test amendments search
	amendmentMatches := searcher.Search("amendment")
	if len(amendmentMatches) == 0 {
		t.Error("Expected to find matches for 'amendment'")
	}
}

func TestSearch_MatchCount(t *testing.T) {
	searcher := NewKeywordSearcher()

	text := `
RULE XX
QUORUM

1. The quorum requirement is a quorum of the majority. Without a quorum, no business.
`

	searcher.ParseHouseRules(text)
	matches := searcher.Search("quorum")

	if len(matches) == 0 {
		t.Fatal("Expected at least one match")
	}

	// First clause should have multiple matches
	if matches[0].MatchCount < 3 {
		t.Errorf("Expected at least 3 matches in first result, got %d", matches[0].MatchCount)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
