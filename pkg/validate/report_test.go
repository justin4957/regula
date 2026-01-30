package validate

import (
	"strings"
	"testing"
	"time"
)

// buildTestValidationResult creates a ValidationResult with all components populated for testing.
func buildTestValidationResult(status ValidationStatus, overallScore float64) *ValidationResult {
	return &ValidationResult{
		Status:       status,
		Threshold:    0.80,
		OverallScore: overallScore,
		ProfileName:  "GDPR",
		References: &ReferenceValidation{
			TotalReferences:  50,
			Resolved:         35,
			Partial:          5,
			Ambiguous:        3,
			NotFound:         2,
			External:         4,
			RangeRefs:        1,
			ResolutionRate:   0.891,
			HighConfidence:   30,
			MediumConfidence: 12,
			LowConfidence:    8,
			UnresolvedExamples: []ReferenceExample{
				{SourceArticle: 15, RawText: "Article 99(2)", Reason: "article not found"},
				{SourceArticle: 42, RawText: "Section 3.1", Reason: "ambiguous target"},
			},
			AmbiguousExamples: []ReferenceExample{
				{SourceArticle: 7, RawText: "the regulation", Reason: "multiple matches"},
			},
		},
		Connectivity: &ConnectivityValidation{
			TotalProvisions:  99,
			ConnectedCount:   85,
			OrphanCount:      14,
			ConnectivityRate: 0.859,
			AvgIncomingRefs:  2.3,
			AvgOutgoingRefs:  1.8,
			OrphanArticles:   []int{5, 12, 67, 88},
			MostReferenced: []ArticleRefCount{
				{ArticleNum: 6, Count: 15},
				{ArticleNum: 4, Count: 12},
			},
		},
		Definitions: &DefinitionValidation{
			TotalDefinitions:  26,
			UsedDefinitions:   22,
			UnusedDefinitions: 4,
			UsageRate:         0.846,
			TotalUsages:       180,
			ArticlesWithTerms: 45,
			UnusedTerms:       []string{"binding corporate rules", "genetic data"},
			MostUsedTerms: []TermUsageCount{
				{Term: "personal data", UsageCount: 42, ArticleCount: 30},
				{Term: "controller", UsageCount: 35, ArticleCount: 25},
			},
		},
		Semantics: &SemanticValidation{
			RightsCount:        18,
			ObligationsCount:   24,
			ArticlesWithRights: 12,
			ArticlesWithOblig:  15,
			RegulationType:     "GDPR",
			KnownRightsFound:   5,
			KnownRightsTotal:   6,
			MissingRights:      []string{"right_to_object"},
			RightTypes:         []string{"access", "rectification", "erasure"},
			ObligationTypes:    []string{"consent", "notify_breach", "record_keeping"},
		},
		Structure: &StructureValidation{
			TotalArticles:          99,
			TotalChapters:          11,
			TotalSections:          22,
			TotalRecitals:          173,
			ExpectedArticles:       99,
			ExpectedChapters:       11,
			ArticleCompleteness:    1.0,
			ChapterCompleteness:    1.0,
			ArticlesWithContent:    95,
			ArticlesEmpty:          4,
			ContentRate:            0.96,
			StructureScore:         0.95,
			ExpectedDefinitions:    26,
			DefinitionCompleteness: 1.0,
		},
		ComponentScores: &ComponentScores{
			ReferenceScore:     0.891,
			ReferenceWeight:    0.25,
			ConnectivityScore:  0.859,
			ConnectivityWeight: 0.20,
			DefinitionScore:    0.846,
			DefinitionWeight:   0.20,
			SemanticScore:      0.833,
			SemanticWeight:     0.20,
			StructureScore:     0.950,
			StructureWeight:    0.15,
		},
		Issues: []ValidationIssue{
			{Category: "references", Severity: "error", Message: "2 references could not be resolved", Count: 2},
		},
		Warnings: []ValidationIssue{
			{Category: "definitions", Severity: "warning", Message: "4 defined terms have no usage links"},
		},
	}
}

// buildTestGateReport creates a GateReport with multiple gate results for testing.
func buildTestGateReport(overallPass bool) *GateReport {
	return &GateReport{
		Results: []*GateResult{
			{
				Gate:    "V0",
				Passed:  true,
				Score:   0.95,
				Metrics: map[string]float64{"file_size": 0.99, "file_type": 1.0},
			},
			{
				Gate:   "V1",
				Passed: true,
				Score:  0.88,
				Metrics: map[string]float64{
					"parse_success":   1.0,
					"article_count":   0.85,
					"structure_depth": 0.80,
				},
				Warnings: []GateWarning{
					{Metric: "structure_depth", Message: "structure_depth (80.0%) close to threshold (72.0%)", Value: 0.80},
				},
			},
			{
				Gate:   "V2",
				Passed: !overallPass || true,
				Score:  0.72,
				Metrics: map[string]float64{
					"reference_resolution": 0.89,
					"definition_coverage":  0.85,
					"semantic_extraction":  0.42,
				},
				Errors: func() []GateError {
					if !overallPass {
						return []GateError{
							{Metric: "semantic_extraction", Message: "semantic_extraction (42.0%) below threshold (60.0%)", Value: 0.42},
						}
					}
					return nil
				}(),
			},
			{
				Gate:       "V3",
				Skipped:    true,
				SkipReason: "skipped by configuration",
				Metrics:    map[string]float64{},
			},
		},
		OverallPass:  overallPass,
		TotalScore:   0.85,
		GatesPassed:  2,
		GatesFailed:  func() int { if overallPass { return 0 } else { return 1 } }(),
		GatesSkipped: 1,
		Duration:     150 * time.Millisecond,
	}
}

// --- ValidationResult.ToMarkdown() tests ---

func TestValidationResult_ToMarkdown(t *testing.T) {
	validationResult := buildTestValidationResult(StatusPass, 0.876)
	markdownOutput := validationResult.ToMarkdown()

	expectedSubstrings := []string{
		"# Validation Report",
		"`PASS`",
		"## Summary",
		"87.6%",
		"80.0%",
		"GDPR",
		"## Component Scores",
		"References",
		"Connectivity",
		"Definitions",
		"Semantics",
		"Structure",
	}

	for _, expectedSubstring := range expectedSubstrings {
		if !strings.Contains(markdownOutput, expectedSubstring) {
			t.Errorf("ToMarkdown() missing expected content: %q", expectedSubstring)
		}
	}
}

func TestValidationResult_ToMarkdown_AllComponents(t *testing.T) {
	validationResult := buildTestValidationResult(StatusPass, 0.876)
	markdownOutput := validationResult.ToMarkdown()

	expectedSections := []string{
		"## Reference Resolution",
		"## Graph Connectivity",
		"## Definition Coverage",
		"## Semantic Extraction",
		"## Structure Quality",
		"## Issues",
		"## Warnings",
	}

	for _, expectedSection := range expectedSections {
		if !strings.Contains(markdownOutput, expectedSection) {
			t.Errorf("ToMarkdown() missing section: %q", expectedSection)
		}
	}

	// Verify specific data appears
	dataChecks := []string{
		"50",                        // TotalReferences
		"89.1%",                     // ResolutionRate
		"Art 15",                    // Unresolved example
		"Article 99(2)",             // Unresolved raw text
		"Art 6",                     // Most referenced article
		"personal data",             // Most used term
		"binding corporate rules",   // Unused term
		"right_to_object",           // Missing right
		"Orphan Articles",           // Orphan articles header
		"5, 12, 67, 88",            // Orphan article numbers
	}

	for _, dataCheck := range dataChecks {
		if !strings.Contains(markdownOutput, dataCheck) {
			t.Errorf("ToMarkdown() missing data: %q", dataCheck)
		}
	}
}

func TestValidationResult_ToMarkdown_EmptyComponents(t *testing.T) {
	validationResult := &ValidationResult{
		Status:       StatusPass,
		Threshold:    0.80,
		OverallScore: 0.90,
	}

	markdownOutput := validationResult.ToMarkdown()

	if !strings.Contains(markdownOutput, "# Validation Report") {
		t.Error("ToMarkdown() should contain header even with empty components")
	}

	if !strings.Contains(markdownOutput, "90.0%") {
		t.Error("ToMarkdown() should contain overall score")
	}

	// Should not contain component sections when nil
	absentSections := []string{
		"## Reference Resolution",
		"## Graph Connectivity",
		"## Definition Coverage",
		"## Semantic Extraction",
		"## Structure Quality",
	}

	for _, absentSection := range absentSections {
		if strings.Contains(markdownOutput, absentSection) {
			t.Errorf("ToMarkdown() should not contain %q when component is nil", absentSection)
		}
	}
}

func TestValidationResult_ToMarkdown_FailStatus(t *testing.T) {
	validationResult := buildTestValidationResult(StatusFail, 0.65)
	markdownOutput := validationResult.ToMarkdown()

	if !strings.Contains(markdownOutput, "`FAIL`") {
		t.Error("ToMarkdown() should show FAIL badge for failing result")
	}
}

func TestValidationResult_ToMarkdown_TableEscaping(t *testing.T) {
	validationResult := &ValidationResult{
		Status:       StatusPass,
		Threshold:    0.80,
		OverallScore: 0.90,
		Issues: []ValidationIssue{
			{Category: "test", Severity: "error", Message: "contains | pipe character"},
		},
	}

	markdownOutput := validationResult.ToMarkdown()

	if strings.Contains(markdownOutput, "contains | pipe") {
		t.Error("ToMarkdown() should escape pipe characters in table cells")
	}

	if !strings.Contains(markdownOutput, "contains \\| pipe") {
		t.Error("ToMarkdown() should contain escaped pipe character")
	}
}

// --- ValidationResult.ToHTML() tests ---

func TestValidationResult_ToHTML(t *testing.T) {
	validationResult := buildTestValidationResult(StatusPass, 0.876)
	htmlOutput := validationResult.ToHTML()

	expectedElements := []string{
		"<!DOCTYPE html>",
		"<html",
		"<style>",
		"Validation Report",
		"87.6%",
		"80.0%",
		"GDPR",
		"<table>",
		"</table>",
		"<details",
		"</details>",
		"</html>",
	}

	for _, expectedElement := range expectedElements {
		if !strings.Contains(htmlOutput, expectedElement) {
			t.Errorf("ToHTML() missing expected element: %q", expectedElement)
		}
	}
}

func TestValidationResult_ToHTML_PassStatus(t *testing.T) {
	validationResult := buildTestValidationResult(StatusPass, 0.90)
	htmlOutput := validationResult.ToHTML()

	if !strings.Contains(htmlOutput, "#4caf50") {
		t.Error("ToHTML() should use green color (#4caf50) for PASS status")
	}

	if !strings.Contains(htmlOutput, ">PASS<") {
		t.Error("ToHTML() should display PASS badge")
	}
}

func TestValidationResult_ToHTML_FailStatus(t *testing.T) {
	validationResult := buildTestValidationResult(StatusFail, 0.60)
	htmlOutput := validationResult.ToHTML()

	if !strings.Contains(htmlOutput, "#f44336") {
		t.Error("ToHTML() should use red color (#f44336) for FAIL status")
	}

	if !strings.Contains(htmlOutput, ">FAIL<") {
		t.Error("ToHTML() should display FAIL badge")
	}
}

func TestValidationResult_ToHTML_ComponentScoreBars(t *testing.T) {
	validationResult := buildTestValidationResult(StatusPass, 0.876)
	htmlOutput := validationResult.ToHTML()

	// Check that component bars are rendered
	componentNames := []string{"References", "Connectivity", "Definitions", "Semantics", "Structure"}
	for _, componentName := range componentNames {
		if !strings.Contains(htmlOutput, componentName) {
			t.Errorf("ToHTML() missing component score bar for: %s", componentName)
		}
	}

	// Check that bar widths use correct percentages
	if !strings.Contains(htmlOutput, "width:89.1%") {
		t.Error("ToHTML() should contain bar width for reference score (89.1%)")
	}
}

func TestValidationResult_ToHTML_HTMLEscaping(t *testing.T) {
	validationResult := &ValidationResult{
		Status:       StatusPass,
		Threshold:    0.80,
		OverallScore: 0.90,
		ProfileName:  "<script>alert('xss')</script>",
	}

	htmlOutput := validationResult.ToHTML()

	if strings.Contains(htmlOutput, "<script>alert") {
		t.Error("ToHTML() must escape HTML in profile name")
	}

	if !strings.Contains(htmlOutput, "&lt;script&gt;") {
		t.Error("ToHTML() should contain escaped profile name")
	}
}

func TestValidationResult_ToHTML_EmptyComponents(t *testing.T) {
	validationResult := &ValidationResult{
		Status:       StatusWarn,
		Threshold:    0.80,
		OverallScore: 0.78,
	}

	htmlOutput := validationResult.ToHTML()

	if !strings.Contains(htmlOutput, "<!DOCTYPE html>") {
		t.Error("ToHTML() should produce valid HTML even with no components")
	}

	if !strings.Contains(htmlOutput, "#ff9800") {
		t.Error("ToHTML() should use orange color (#ff9800) for WARN status")
	}
}

// --- GateReport.ToMarkdown() tests ---

func TestGateReport_ToMarkdown(t *testing.T) {
	gateReport := buildTestGateReport(true)
	markdownOutput := gateReport.ToMarkdown()

	expectedSubstrings := []string{
		"# Gate Validation Report",
		"`PASS`",
		"## Summary",
		"85.0%",
		"## Gate Results",
		"V0",
		"V1",
		"V2",
		"V3",
	}

	for _, expectedSubstring := range expectedSubstrings {
		if !strings.Contains(markdownOutput, expectedSubstring) {
			t.Errorf("GateReport.ToMarkdown() missing expected content: %q", expectedSubstring)
		}
	}
}

func TestGateReport_ToMarkdown_WithWarnings(t *testing.T) {
	gateReport := buildTestGateReport(true)
	markdownOutput := gateReport.ToMarkdown()

	if !strings.Contains(markdownOutput, "**Warnings:**") {
		t.Error("GateReport.ToMarkdown() should contain warnings section")
	}

	if !strings.Contains(markdownOutput, "structure_depth") {
		t.Error("GateReport.ToMarkdown() should contain warning metric name")
	}

	if !strings.Contains(markdownOutput, "close to threshold") {
		t.Error("GateReport.ToMarkdown() should contain warning message")
	}
}

func TestGateReport_ToMarkdown_SkippedGates(t *testing.T) {
	gateReport := buildTestGateReport(true)
	markdownOutput := gateReport.ToMarkdown()

	if !strings.Contains(markdownOutput, "`SKIP`") {
		t.Error("GateReport.ToMarkdown() should show SKIP badge for skipped gates")
	}

	if !strings.Contains(markdownOutput, "skipped by configuration") {
		t.Error("GateReport.ToMarkdown() should show skip reason")
	}
}

func TestGateReport_ToMarkdown_FailedGates(t *testing.T) {
	gateReport := buildTestGateReport(false)
	markdownOutput := gateReport.ToMarkdown()

	if !strings.Contains(markdownOutput, "`FAIL`") {
		t.Error("GateReport.ToMarkdown() should show FAIL badge for overall failure")
	}

	if !strings.Contains(markdownOutput, "**Errors:**") {
		t.Error("GateReport.ToMarkdown() should contain errors section for failed gates")
	}

	if !strings.Contains(markdownOutput, "below threshold") {
		t.Error("GateReport.ToMarkdown() should contain error message for threshold failure")
	}
}

// --- GateReport.ToHTML() tests ---

func TestGateReport_ToHTML(t *testing.T) {
	gateReport := buildTestGateReport(true)
	htmlOutput := gateReport.ToHTML()

	expectedElements := []string{
		"<!DOCTYPE html>",
		"Gate Validation Report",
		"<style>",
		"<table>",
		"gate-card",
		"V0",
		"V1",
		"V2",
		"V3",
		"</html>",
	}

	for _, expectedElement := range expectedElements {
		if !strings.Contains(htmlOutput, expectedElement) {
			t.Errorf("GateReport.ToHTML() missing expected element: %q", expectedElement)
		}
	}
}

func TestGateReport_ToHTML_PassStatus(t *testing.T) {
	gateReport := buildTestGateReport(true)
	htmlOutput := gateReport.ToHTML()

	if !strings.Contains(htmlOutput, "#4caf50") {
		t.Error("GateReport.ToHTML() should use green color for PASS status")
	}

	if !strings.Contains(htmlOutput, ">PASS<") {
		t.Error("GateReport.ToHTML() should display PASS badge")
	}
}

func TestGateReport_ToHTML_FailStatus(t *testing.T) {
	gateReport := buildTestGateReport(false)
	htmlOutput := gateReport.ToHTML()

	if !strings.Contains(htmlOutput, "#f44336") {
		t.Error("GateReport.ToHTML() should use red color for FAIL status")
	}
}

func TestGateReport_ToHTML_HaltedPipeline(t *testing.T) {
	gateReport := buildTestGateReport(false)
	gateReport.HaltedAt = "V2"
	htmlOutput := gateReport.ToHTML()

	if !strings.Contains(htmlOutput, "Pipeline halted at gate: V2") {
		t.Error("GateReport.ToHTML() should indicate halted pipeline")
	}

	if !strings.Contains(htmlOutput, "alert-warning") {
		t.Error("GateReport.ToHTML() should show halted message as warning alert")
	}
}

func TestGateReport_ToHTML_SkippedGateCard(t *testing.T) {
	gateReport := buildTestGateReport(true)
	htmlOutput := gateReport.ToHTML()

	if !strings.Contains(htmlOutput, "#9e9e9e") {
		t.Error("GateReport.ToHTML() should use grey color for skipped gates")
	}

	if !strings.Contains(htmlOutput, "skip-reason") {
		t.Error("GateReport.ToHTML() should display skip reason for skipped gates")
	}
}

// --- Helper function tests ---

func TestStatusToMarkdownBadge(t *testing.T) {
	testCases := []struct {
		status   ValidationStatus
		expected string
	}{
		{StatusPass, "`PASS`"},
		{StatusFail, "`FAIL`"},
		{StatusWarn, "`WARN`"},
		{"SKIP", "`SKIP`"},
		{"UNKNOWN", "`UNKNOWN`"},
	}

	for _, testCase := range testCases {
		result := statusToMarkdownBadge(testCase.status)
		if result != testCase.expected {
			t.Errorf("statusToMarkdownBadge(%q) = %q, want %q", testCase.status, result, testCase.expected)
		}
	}
}

func TestEscapeMarkdownTableCell(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"no pipes here", "no pipes here"},
		{"has | pipe", "has \\| pipe"},
		{"multiple | pipes | here", "multiple \\| pipes \\| here"},
		{"", ""},
	}

	for _, testCase := range testCases {
		result := escapeMarkdownTableCell(testCase.input)
		if result != testCase.expected {
			t.Errorf("escapeMarkdownTableCell(%q) = %q, want %q", testCase.input, result, testCase.expected)
		}
	}
}

func TestStatusToHTMLColor(t *testing.T) {
	testCases := []struct {
		status   ValidationStatus
		expected string
	}{
		{StatusPass, "#4caf50"},
		{StatusFail, "#f44336"},
		{StatusWarn, "#ff9800"},
		{"UNKNOWN", "#9e9e9e"},
	}

	for _, testCase := range testCases {
		result := statusToHTMLColor(testCase.status)
		if result != testCase.expected {
			t.Errorf("statusToHTMLColor(%q) = %q, want %q", testCase.status, result, testCase.expected)
		}
	}
}

func TestScoreToHTMLColor(t *testing.T) {
	testCases := []struct {
		score    float64
		expected string
	}{
		{0.90, "#4caf50"},
		{0.80, "#4caf50"},
		{0.70, "#ff9800"},
		{0.60, "#ff9800"},
		{0.50, "#f44336"},
		{0.0, "#f44336"},
	}

	for _, testCase := range testCases {
		result := scoreToHTMLColor(testCase.score)
		if result != testCase.expected {
			t.Errorf("scoreToHTMLColor(%.2f) = %q, want %q", testCase.score, result, testCase.expected)
		}
	}
}
