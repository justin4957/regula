package draft

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderImpactGraph_SingleAmendment(t *testing.T) {
	triples := buildTitle15ImpactTriples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	bill := &DraftBill{
		BillNumber: "H.R. 9999",
		Title:      "Single Amendment Act",
		ShortTitle: "SAA",
		Sections: []*DraftSection{
			{
				Number: "1",
				Title:  "Penalty increase",
				Amendments: []Amendment{
					{
						Type:             AmendStrikeInsert,
						TargetTitle:      "15",
						TargetSection:    "6505",
						TargetSubsection: "d",
						StrikeText:       "$50,000",
						InsertText:       "$100,000",
						Description:      `by striking "$50,000" and inserting "$100,000"`,
					},
				},
			},
		},
	}

	diff, err := ComputeDiff(bill, libraryPath)
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	impactResult, err := AnalyzeDraftImpact(diff, libraryPath, 1)
	if err != nil {
		t.Fatalf("AnalyzeDraftImpact failed: %v", err)
	}

	dotOutput, err := RenderImpactGraph(impactResult)
	if err != nil {
		t.Fatalf("RenderImpactGraph failed: %v", err)
	}

	// Validate DOT structure
	assertDOTContains(t, dotOutput, "digraph DraftImpactAnalysis")
	assertDOTContains(t, dotOutput, "rankdir=LR")
	assertDOTContains(t, dotOutput, "H.R. 9999")

	// Should have at least one node (the modified provision)
	assertDOTContains(t, dotOutput, "fillcolor=red")

	// Should have a legend
	assertDOTContains(t, dotOutput, "cluster_legend")
	assertDOTContains(t, dotOutput, "Modified/Repealed")
	assertDOTContains(t, dotOutput, "Directly Affected")
	assertDOTContains(t, dotOutput, "Transitively Affected")

	// Should have a title cluster
	assertDOTContains(t, dotOutput, "Title 15")

	// DOT should be well-formed (opens and closes braces)
	openBraces := strings.Count(dotOutput, "{")
	closeBraces := strings.Count(dotOutput, "}")
	if openBraces != closeBraces {
		t.Errorf("unbalanced braces: %d open, %d close", openBraces, closeBraces)
	}

	// Must end with closing brace
	trimmed := strings.TrimSpace(dotOutput)
	if !strings.HasSuffix(trimmed, "}") {
		t.Error("DOT output should end with closing brace")
	}
}

func TestRenderImpactGraph_MultiTitle(t *testing.T) {
	// Build impact result referencing multiple titles
	result := &DraftImpactResult{
		Bill: &DraftBill{
			BillNumber: "H.R. 1234",
			Title:      "Multi-Title Act",
		},
		Diff: &DraftDiff{
			Bill: &DraftBill{BillNumber: "H.R. 1234"},
			Modified: []DiffEntry{
				{
					Amendment:        Amendment{Type: AmendStrikeInsert, Description: "amend title 15"},
					TargetURI:        "https://regula.dev/regulations/US-USC-TITLE-15:Art6502",
					TargetDocumentID: "us-usc-title-15",
				},
			},
			Removed: []DiffEntry{
				{
					Amendment:        Amendment{Type: AmendRepeal, Description: "repeal title 42 section"},
					TargetURI:        "https://regula.dev/regulations/US-USC-TITLE-42:Art1396a",
					TargetDocumentID: "us-usc-title-42",
				},
			},
			Added:        []DiffEntry{},
			Redesignated: []DiffEntry{},
		},
		DirectlyAffected: []AffectedProvision{
			{
				URI:        "https://regula.dev/regulations/US-USC-TITLE-15:Art6505",
				Label:      "Civil penalties",
				DocumentID: "us-usc-title-15",
				Depth:      1,
				Reason:     "references modified Art6502",
			},
			{
				URI:        "https://regula.dev/regulations/US-USC-TITLE-42:Art1396b",
				Label:      "Payment to States",
				DocumentID: "us-usc-title-42",
				Depth:      1,
				Reason:     "references repealed Art1396a",
			},
		},
		TransitivelyAffected: []AffectedProvision{},
		BrokenCrossRefs:      []BrokenReference{},
		ObligationChanges:    ObligationDelta{Added: []string{}, Removed: []string{}, Modified: []string{}},
		RightsChanges:        RightsDelta{Added: []string{}, Removed: []string{}, Modified: []string{}},
	}

	dotOutput, err := RenderImpactGraph(result)
	if err != nil {
		t.Fatalf("RenderImpactGraph failed: %v", err)
	}

	// Should have clusters for both titles
	assertDOTContains(t, dotOutput, "Title 15")
	assertDOTContains(t, dotOutput, "Title 42")

	// Should have nodes for both titles
	assertDOTContains(t, dotOutput, "Art6502")
	assertDOTContains(t, dotOutput, "Art1396a")

	// Modified and repealed should be red
	if strings.Count(dotOutput, "fillcolor=red") < 2 {
		t.Error("expected at least 2 red nodes (modified + repealed)")
	}

	// Directly affected should be orange
	if strings.Count(dotOutput, "fillcolor=orange") < 2 {
		t.Error("expected at least 2 orange nodes for directly affected provisions")
	}
}

func TestRenderImpactGraph_BrokenRefs(t *testing.T) {
	result := &DraftImpactResult{
		Bill: &DraftBill{
			BillNumber: "H.R. 8888",
			Title:      "Repeal Act",
		},
		Diff: &DraftDiff{
			Bill: &DraftBill{BillNumber: "H.R. 8888"},
			Removed: []DiffEntry{
				{
					Amendment:        Amendment{Type: AmendRepeal, Description: "repeal section 6503"},
					TargetURI:        "https://regula.dev/regulations/US-USC-TITLE-15:Art6503",
					TargetDocumentID: "us-usc-title-15",
				},
			},
			Modified:     []DiffEntry{},
			Added:        []DiffEntry{},
			Redesignated: []DiffEntry{},
		},
		DirectlyAffected: []AffectedProvision{
			{
				URI:        "https://regula.dev/regulations/US-USC-TITLE-15:Art6504",
				Label:      "Actions by States",
				DocumentID: "us-usc-title-15",
				Depth:      1,
				Reason:     "references repealed Art6503",
			},
		},
		TransitivelyAffected: []AffectedProvision{},
		BrokenCrossRefs: []BrokenReference{
			{
				SourceURI:        "https://regula.dev/regulations/US-USC-TITLE-15:Art6504",
				SourceLabel:      "Actions by States",
				SourceDocumentID: "us-usc-title-15",
				TargetURI:        "https://regula.dev/regulations/US-USC-TITLE-15:Art6503",
				TargetLabel:      "Art6503",
				Severity:         SeverityError,
				Predicate:        "reg:references",
				Reason:           "target repealed §Art6503",
			},
			{
				SourceURI:        "https://regula.dev/regulations/US-USC-TITLE-15:Art6502",
				SourceLabel:      "Regulation of unfair acts",
				SourceDocumentID: "us-usc-title-15",
				TargetURI:        "https://regula.dev/regulations/US-USC-TITLE-15:Art6503",
				TargetLabel:      "Art6503",
				Severity:         SeverityError,
				Predicate:        "reg:references",
				Reason:           "target repealed §Art6503",
			},
		},
		ObligationChanges: ObligationDelta{Added: []string{}, Removed: []string{}, Modified: []string{}},
		RightsChanges:     RightsDelta{Added: []string{}, Removed: []string{}, Modified: []string{}},
	}

	dotOutput, err := RenderImpactGraph(result)
	if err != nil {
		t.Fatalf("RenderImpactGraph failed: %v", err)
	}

	// Broken refs should produce dashed red edges
	assertDOTContains(t, dotOutput, "style=dashed")
	assertDOTContains(t, dotOutput, "color=red")

	// Error severity should produce "BROKEN" label
	assertDOTContains(t, dotOutput, "BROKEN")

	// Should have both broken ref edges
	dashedCount := strings.Count(dotOutput, "style=dashed")
	if dashedCount < 2 {
		t.Errorf("expected at least 2 dashed edges for broken refs, got %d", dashedCount)
	}
}

func TestRenderImpactGraph_EmptyImpact(t *testing.T) {
	result := &DraftImpactResult{
		Bill: &DraftBill{
			BillNumber: "H.R. 0001",
			Title:      "No Impact Act",
		},
		Diff: &DraftDiff{
			Bill:         &DraftBill{BillNumber: "H.R. 0001"},
			Modified:     []DiffEntry{},
			Removed:      []DiffEntry{},
			Added:        []DiffEntry{},
			Redesignated: []DiffEntry{},
		},
		DirectlyAffected:     []AffectedProvision{},
		TransitivelyAffected: []AffectedProvision{},
		BrokenCrossRefs:      []BrokenReference{},
		ObligationChanges:    ObligationDelta{Added: []string{}, Removed: []string{}, Modified: []string{}},
		RightsChanges:        RightsDelta{Added: []string{}, Removed: []string{}, Modified: []string{}},
	}

	dotOutput, err := RenderImpactGraph(result)
	if err != nil {
		t.Fatalf("RenderImpactGraph failed: %v", err)
	}

	// Should still produce valid DOT
	assertDOTContains(t, dotOutput, "digraph DraftImpactAnalysis")
	assertDOTContains(t, dotOutput, "cluster_legend")

	// Should have no red or orange nodes outside the legend
	redCount := strings.Count(dotOutput, "fillcolor=red")
	legendRedCount := strings.Count(dotOutput, "legend_modified")
	if redCount > legendRedCount {
		t.Errorf("empty impact should have no red nodes outside legend, found %d extra", redCount-legendRedCount)
	}
	orangeCount := strings.Count(dotOutput, "fillcolor=orange")
	legendOrangeCount := strings.Count(dotOutput, "legend_direct")
	if orangeCount > legendOrangeCount {
		t.Errorf("empty impact should have no orange nodes outside legend, found %d extra", orangeCount-legendOrangeCount)
	}

	// Should be well-formed
	openBraces := strings.Count(dotOutput, "{")
	closeBraces := strings.Count(dotOutput, "}")
	if openBraces != closeBraces {
		t.Errorf("unbalanced braces: %d open, %d close", openBraces, closeBraces)
	}
}

func TestRenderImpactGraph_NilResult(t *testing.T) {
	_, err := RenderImpactGraph(nil)
	if err == nil {
		t.Error("expected error for nil impact result")
	}
}

func TestRenderImpactGraphToFile(t *testing.T) {
	result := &DraftImpactResult{
		Bill: &DraftBill{
			BillNumber: "H.R. 0002",
			Title:      "File Output Act",
		},
		Diff: &DraftDiff{
			Bill:         &DraftBill{BillNumber: "H.R. 0002"},
			Modified:     []DiffEntry{},
			Removed:      []DiffEntry{},
			Added:        []DiffEntry{},
			Redesignated: []DiffEntry{},
		},
		DirectlyAffected:     []AffectedProvision{},
		TransitivelyAffected: []AffectedProvision{},
		BrokenCrossRefs:      []BrokenReference{},
		ObligationChanges:    ObligationDelta{Added: []string{}, Removed: []string{}, Modified: []string{}},
		RightsChanges:        RightsDelta{Added: []string{}, Removed: []string{}, Modified: []string{}},
	}

	outputPath := filepath.Join(t.TempDir(), "impact.dot")
	err := RenderImpactGraphToFile(result, outputPath)
	if err != nil {
		t.Fatalf("RenderImpactGraphToFile failed: %v", err)
	}

	// Verify file was created and contains valid DOT
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "digraph DraftImpactAnalysis") {
		t.Error("output file should contain valid DOT graph")
	}
}

func TestRenderImpactGraph_TransitiveNodes(t *testing.T) {
	result := &DraftImpactResult{
		Bill: &DraftBill{
			BillNumber: "H.R. 3333",
			Title:      "Transitive Impact Act",
		},
		Diff: &DraftDiff{
			Bill: &DraftBill{BillNumber: "H.R. 3333"},
			Modified: []DiffEntry{
				{
					Amendment:        Amendment{Type: AmendStrikeInsert, Description: "amend 6502"},
					TargetURI:        "https://regula.dev/regulations/US-USC-TITLE-15:Art6502",
					TargetDocumentID: "us-usc-title-15",
				},
			},
			Removed:      []DiffEntry{},
			Added:        []DiffEntry{},
			Redesignated: []DiffEntry{},
		},
		DirectlyAffected: []AffectedProvision{
			{
				URI:        "https://regula.dev/regulations/US-USC-TITLE-15:Art6505",
				Label:      "Civil penalties",
				DocumentID: "us-usc-title-15",
				Depth:      1,
				Reason:     "references modified Art6502",
			},
		},
		TransitivelyAffected: []AffectedProvision{
			{
				URI:        "https://regula.dev/regulations/US-USC-TITLE-15:Art6504",
				Label:      "Actions by States",
				DocumentID: "us-usc-title-15",
				Depth:      2,
				Reason:     "transitively linked via modified Art6502",
			},
		},
		BrokenCrossRefs:   []BrokenReference{},
		ObligationChanges: ObligationDelta{Added: []string{}, Removed: []string{}, Modified: []string{}},
		RightsChanges:     RightsDelta{Added: []string{}, Removed: []string{}, Modified: []string{}},
	}

	dotOutput, err := RenderImpactGraph(result)
	if err != nil {
		t.Fatalf("RenderImpactGraph failed: %v", err)
	}

	// Should have red (modified), orange (direct), and yellow (transitive) nodes
	assertDOTContains(t, dotOutput, "fillcolor=red")
	assertDOTContains(t, dotOutput, "fillcolor=orange")
	assertDOTContains(t, dotOutput, "fillcolor=yellow")

	// Impact chain edges should be bold
	assertDOTContains(t, dotOutput, "style=bold")
}

func TestClusterKeyFromDocumentID(t *testing.T) {
	tests := []struct {
		documentID string
		expected   string
	}{
		{"us-usc-title-15", "Title 15"},
		{"us-usc-title-42", "Title 42"},
		{"", "_proposed"},
		{"some-other-id", "some-other-id"},
	}

	for _, testCase := range tests {
		result := clusterKeyFromDocumentID(testCase.documentID)
		if result != testCase.expected {
			t.Errorf("clusterKeyFromDocumentID(%q) = %q, expected %q",
				testCase.documentID, result, testCase.expected)
		}
	}
}

func TestTruncateLabel(t *testing.T) {
	tests := []struct {
		label    string
		maxLen   int
		expected string
	}{
		{"short", 40, "short"},
		{"this is a very long label that exceeds the maximum length", 30, "this is a very long label t..."},
		{"exact", 5, "exact"},
	}

	for _, testCase := range tests {
		result := truncateLabel(testCase.label, testCase.maxLen)
		if result != testCase.expected {
			t.Errorf("truncateLabel(%q, %d) = %q, expected %q",
				testCase.label, testCase.maxLen, result, testCase.expected)
		}
	}
}

// assertDOTContains is a test helper that checks the DOT output contains the
// expected substring.
func assertDOTContains(t *testing.T, dotOutput, expected string) {
	t.Helper()
	if !strings.Contains(dotOutput, expected) {
		t.Errorf("DOT output missing expected content: %q\nFirst 500 chars:\n%s",
			expected, truncateForLog(dotOutput, 500))
	}
}

// truncateForLog shortens a string for test log output.
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
