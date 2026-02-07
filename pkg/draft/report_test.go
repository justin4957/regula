package draft

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateReport_FullPipeline(t *testing.T) {
	testDataPath := getTestDataPath(t)
	billPath := filepath.Join(testDataPath, "public-health-reporting.txt")

	// Parse the bill
	bill, err := ParseBillFromFile(billPath)
	if err != nil {
		t.Fatalf("Failed to parse bill: %v", err)
	}

	// Extract amendments from sections
	extractAllAmendments(bill)

	// Create a temporary library for testing
	tempDir, err := os.MkdirTemp("", "report_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Generate report with default options
	options := DefaultReportOptions()
	report, err := GenerateReport(bill, tempDir, options)
	// We expect an error due to missing library documents, but report should still be generated
	if report == nil {
		t.Fatal("Expected non-nil report even with library errors")
	}

	// Verify report structure
	if report.Bill == nil {
		t.Error("Expected non-nil bill in report")
	}
	if report.Bill.BillNumber != "H.R. 2847" {
		t.Errorf("Expected bill number 'H.R. 2847', got '%s'", report.Bill.BillNumber)
	}
	if report.GeneratedAt.IsZero() {
		t.Error("Expected non-zero GeneratedAt timestamp")
	}

	// Verify executive summary was built
	if report.ExecutiveSummary.BillNumber != "H.R. 2847" {
		t.Errorf("Expected summary bill number 'H.R. 2847', got '%s'", report.ExecutiveSummary.BillNumber)
	}

	t.Logf("Report generated: %s, Risk Level: %s", report.Bill.BillNumber, report.RiskLevel)
}

func TestGenerateReport_EmptyBill(t *testing.T) {
	// Test with nil bill
	report, err := GenerateReport(nil, "", DefaultReportOptions())
	if err == nil {
		t.Error("Expected error for nil bill")
	}
	if report != nil {
		t.Error("Expected nil report for nil bill")
	}

	// Test with empty bill
	emptyBill := &DraftBill{}
	tempDir, err := os.MkdirTemp("", "empty_bill_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	report, _ = GenerateReport(emptyBill, tempDir, DefaultReportOptions())
	if report == nil {
		t.Fatal("Expected non-nil report for empty bill")
	}
	if report.ExecutiveSummary.AmendmentCount != 0 {
		t.Errorf("Expected 0 amendments for empty bill, got %d", report.ExecutiveSummary.AmendmentCount)
	}
}

func TestComputeRiskLevel_High(t *testing.T) {
	tests := []struct {
		name     string
		report   *LegislativeImpactReport
		expected RiskLevel
		contains string
	}{
		{
			name: "conflict errors present",
			report: &LegislativeImpactReport{
				Conflicts: &ConflictReport{
					Summary: ConflictSummary{
						Errors: 2,
					},
				},
			},
			expected: RiskHigh,
			contains: "conflict error",
		},
		{
			name: "more than 5 broken refs",
			report: &LegislativeImpactReport{
				Impact: &DraftImpactResult{
					BrokenCrossRefs: make([]BrokenReference, 6),
				},
			},
			expected: RiskHigh,
			contains: "broken cross-references",
		},
		{
			name: "more than 50 affected provisions",
			report: &LegislativeImpactReport{
				Impact: &DraftImpactResult{
					TotalProvisionsAffected: 55,
				},
			},
			expected: RiskHigh,
			contains: "provisions affected",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			level, justification := ComputeRiskLevel(tc.report)
			if level != tc.expected {
				t.Errorf("Expected risk level %s, got %s", tc.expected, level)
			}
			if !strings.Contains(justification, tc.contains) {
				t.Errorf("Expected justification to contain '%s', got '%s'", tc.contains, justification)
			}
		})
	}
}

func TestComputeRiskLevel_Medium(t *testing.T) {
	tests := []struct {
		name     string
		report   *LegislativeImpactReport
		expected RiskLevel
		contains string
	}{
		{
			name: "conflict warnings present",
			report: &LegislativeImpactReport{
				Conflicts: &ConflictReport{
					Summary: ConflictSummary{
						Warnings: 3,
					},
				},
			},
			expected: RiskMedium,
			contains: "warning",
		},
		{
			name: "1-5 broken refs",
			report: &LegislativeImpactReport{
				Impact: &DraftImpactResult{
					BrokenCrossRefs: make([]BrokenReference, 3),
				},
			},
			expected: RiskMedium,
			contains: "broken cross-reference",
		},
		{
			name: "temporal issues",
			report: &LegislativeImpactReport{
				TemporalFindings: []TemporalFinding{
					{Severity: ConflictWarning, Type: TemporalGap},
				},
			},
			expected: RiskMedium,
			contains: "temporal issue",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			level, justification := ComputeRiskLevel(tc.report)
			if level != tc.expected {
				t.Errorf("Expected risk level %s, got %s", tc.expected, level)
			}
			if !strings.Contains(justification, tc.contains) {
				t.Errorf("Expected justification to contain '%s', got '%s'", tc.contains, justification)
			}
		})
	}
}

func TestComputeRiskLevel_Low(t *testing.T) {
	tests := []struct {
		name   string
		report *LegislativeImpactReport
	}{
		{
			name:   "nil report",
			report: nil,
		},
		{
			name:   "empty report",
			report: &LegislativeImpactReport{},
		},
		{
			name: "only info-level findings",
			report: &LegislativeImpactReport{
				TemporalFindings: []TemporalFinding{
					{Severity: ConflictInfo, Type: TemporalSunset},
				},
				Conflicts: &ConflictReport{
					Summary: ConflictSummary{
						Infos: 2,
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			level, _ := ComputeRiskLevel(tc.report)
			if level != RiskLow {
				t.Errorf("Expected risk level %s, got %s", RiskLow, level)
			}
		})
	}
}

func TestSummarizeReport(t *testing.T) {
	bill := &DraftBill{
		BillNumber: "H.R. 1234",
		ShortTitle: "Test Act of 2026",
		Sections: []*DraftSection{
			{Amendments: []Amendment{{}, {}}},
			{Amendments: []Amendment{{}}},
		},
	}

	diff := &DraftDiff{
		Modified: []DiffEntry{
			{Amendment: Amendment{TargetTitle: "15"}},
			{Amendment: Amendment{TargetTitle: "42"}},
		},
		Removed: []DiffEntry{
			{Amendment: Amendment{TargetTitle: "15"}},
		},
		Added: []DiffEntry{
			{Amendment: Amendment{TargetTitle: "26"}},
		},
	}

	impact := &DraftImpactResult{
		TotalProvisionsAffected: 25,
		BrokenCrossRefs:         make([]BrokenReference, 2),
		ObligationChanges: ObligationDelta{
			Added:   []string{"obl1", "obl2"},
			Removed: []string{"obl3"},
		},
		RightsChanges: RightsDelta{
			Added:   []string{"right1"},
			Removed: []string{},
		},
	}

	conflicts := &ConflictReport{
		Summary: ConflictSummary{
			Errors:   1,
			Warnings: 2,
		},
	}

	report := &LegislativeImpactReport{
		Bill:      bill,
		Diff:      diff,
		Impact:    impact,
		Conflicts: conflicts,
	}

	summary := SummarizeReport(report)

	// Verify counts
	if summary.BillNumber != "H.R. 1234" {
		t.Errorf("Expected bill number 'H.R. 1234', got '%s'", summary.BillNumber)
	}
	if summary.BillTitle != "Test Act of 2026" {
		t.Errorf("Expected bill title 'Test Act of 2026', got '%s'", summary.BillTitle)
	}
	if summary.AmendmentCount != 3 {
		t.Errorf("Expected 3 amendments, got %d", summary.AmendmentCount)
	}
	if summary.ProvisionsModified != 2 {
		t.Errorf("Expected 2 modified provisions, got %d", summary.ProvisionsModified)
	}
	if summary.ProvisionsRepealed != 1 {
		t.Errorf("Expected 1 repealed provision, got %d", summary.ProvisionsRepealed)
	}
	if summary.ProvisionsAdded != 1 {
		t.Errorf("Expected 1 added provision, got %d", summary.ProvisionsAdded)
	}
	if summary.TotalProvisionsAffected != 25 {
		t.Errorf("Expected 25 total affected, got %d", summary.TotalProvisionsAffected)
	}
	if summary.BrokenCrossRefs != 2 {
		t.Errorf("Expected 2 broken refs, got %d", summary.BrokenCrossRefs)
	}
	if summary.ConflictErrors != 1 {
		t.Errorf("Expected 1 conflict error, got %d", summary.ConflictErrors)
	}
	if summary.ConflictWarnings != 2 {
		t.Errorf("Expected 2 conflict warnings, got %d", summary.ConflictWarnings)
	}
	if summary.ObligationsAdded != 2 {
		t.Errorf("Expected 2 obligations added, got %d", summary.ObligationsAdded)
	}
	if summary.ObligationsRemoved != 1 {
		t.Errorf("Expected 1 obligation removed, got %d", summary.ObligationsRemoved)
	}
	if summary.RightsAdded != 1 {
		t.Errorf("Expected 1 right added, got %d", summary.RightsAdded)
	}

	// Verify titles affected
	expectedTitles := []int{15, 26, 42}
	if len(summary.TitlesAffected) != len(expectedTitles) {
		t.Errorf("Expected %d titles affected, got %d", len(expectedTitles), len(summary.TitlesAffected))
	}
}

func TestFormatReport_JSON(t *testing.T) {
	report := &LegislativeImpactReport{
		Bill: &DraftBill{
			BillNumber: "H.R. 5678",
			ShortTitle: "JSON Test Act",
		},
		GeneratedAt: time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
		RiskLevel:   RiskMedium,
		ExecutiveSummary: ExecutiveSummary{
			BillNumber:     "H.R. 5678",
			AmendmentCount: 5,
			RiskLevel:      RiskMedium,
		},
	}

	jsonOutput := FormatReport(report, "json")

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOutput), &parsed); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Verify key fields are present
	if _, ok := parsed["bill"]; !ok {
		t.Error("Expected 'bill' field in JSON output")
	}
	if _, ok := parsed["executive_summary"]; !ok {
		t.Error("Expected 'executive_summary' field in JSON output")
	}
	if _, ok := parsed["risk_level"]; !ok {
		t.Error("Expected 'risk_level' field in JSON output")
	}
}

func TestFormatReport_Summary(t *testing.T) {
	report := &LegislativeImpactReport{
		Bill: &DraftBill{
			BillNumber: "H.R. 9999",
			ShortTitle: "Summary Test Act",
		},
		GeneratedAt: time.Now(),
		RiskLevel:   RiskHigh,
		ExecutiveSummary: ExecutiveSummary{
			BillTitle:          "Summary Test Act",
			BillNumber:         "H.R. 9999",
			AmendmentCount:     10,
			ProvisionsModified: 5,
			BrokenCrossRefs:    8,
			ConflictErrors:     2,
			RiskLevel:          RiskHigh,
			RiskJustification:  "conflict errors detected",
		},
	}

	output := FormatReport(report, "summary")

	// Verify key sections are present
	if !strings.Contains(output, "H.R. 9999") {
		t.Error("Expected bill number in summary output")
	}
	if !strings.Contains(output, "Summary Test Act") {
		t.Error("Expected bill title in summary output")
	}
	if !strings.Contains(output, "HIGH") {
		t.Error("Expected HIGH risk level in summary output")
	}
	if !strings.Contains(output, "Key Metrics") {
		t.Error("Expected 'Key Metrics' section in summary output")
	}
	if !strings.Contains(output, "Issues Detected") {
		t.Error("Expected 'Issues Detected' section in summary output")
	}
}

func TestFormatReport_Table(t *testing.T) {
	report := &LegislativeImpactReport{
		Bill: &DraftBill{
			BillNumber: "H.R. 1111",
			ShortTitle: "Table Test Act",
		},
		GeneratedAt: time.Now(),
		RiskLevel:   RiskLow,
		ExecutiveSummary: ExecutiveSummary{
			BillTitle:  "Table Test Act",
			BillNumber: "H.R. 1111",
		},
		TemporalFindings: []TemporalFinding{
			{
				Type:        TemporalGap,
				Severity:    ConflictWarning,
				Description: "potential temporal gap detected",
			},
		},
		Conflicts: &ConflictReport{
			Conflicts: []Conflict{
				{
					Type:        ConflictObligationDuplicate,
					Severity:    ConflictInfo,
					Description: "duplicate obligation detected",
				},
			},
		},
	}

	output := FormatReport(report, "table")

	// Verify sections are present
	if !strings.Contains(output, "Temporal Findings") {
		t.Error("Expected 'Temporal Findings' section in table output")
	}
	if !strings.Contains(output, "temporal gap") {
		t.Error("Expected temporal gap finding in table output")
	}
	if !strings.Contains(output, "Conflicts") {
		t.Error("Expected 'Conflicts' section in table output")
	}
	if !strings.Contains(output, "duplicate obligation") {
		t.Error("Expected conflict description in table output")
	}
}

func TestRiskLevel_String(t *testing.T) {
	tests := []struct {
		level    RiskLevel
		expected string
	}{
		{RiskLow, "low"},
		{RiskMedium, "medium"},
		{RiskHigh, "high"},
		{RiskLevel(99), "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			if tc.level.String() != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, tc.level.String())
			}
		})
	}
}

func TestReport_HasErrors(t *testing.T) {
	// Report with no errors
	report := &LegislativeImpactReport{
		Conflicts: &ConflictReport{
			Summary: ConflictSummary{
				Warnings: 2,
				Infos:    1,
			},
		},
	}
	if report.HasErrors() {
		t.Error("Expected HasErrors to return false")
	}

	// Report with conflict errors
	report.Conflicts.Summary.Errors = 1
	if !report.HasErrors() {
		t.Error("Expected HasErrors to return true with conflict errors")
	}

	// Report with error-level broken ref
	report2 := &LegislativeImpactReport{
		Impact: &DraftImpactResult{
			BrokenCrossRefs: []BrokenReference{
				{Severity: SeverityError},
			},
		},
	}
	if !report2.HasErrors() {
		t.Error("Expected HasErrors to return true with error-level broken ref")
	}

	// Report with error-level temporal finding
	report3 := &LegislativeImpactReport{
		TemporalFindings: []TemporalFinding{
			{Severity: ConflictError},
		},
	}
	if !report3.HasErrors() {
		t.Error("Expected HasErrors to return true with error-level temporal finding")
	}
}

func TestReport_HasWarnings(t *testing.T) {
	// Report with no warnings
	report := &LegislativeImpactReport{
		Conflicts: &ConflictReport{
			Summary: ConflictSummary{
				Infos: 5,
			},
		},
	}
	if report.HasWarnings() {
		t.Error("Expected HasWarnings to return false")
	}

	// Report with conflict warnings
	report.Conflicts.Summary.Warnings = 1
	if !report.HasWarnings() {
		t.Error("Expected HasWarnings to return true with conflict warnings")
	}

	// Report with warning-level broken ref
	report2 := &LegislativeImpactReport{
		Impact: &DraftImpactResult{
			BrokenCrossRefs: []BrokenReference{
				{Severity: SeverityWarning},
			},
		},
	}
	if !report2.HasWarnings() {
		t.Error("Expected HasWarnings to return true with warning-level broken ref")
	}
}

func TestDefaultReportOptions(t *testing.T) {
	options := DefaultReportOptions()

	if !options.IncludeDiff {
		t.Error("Expected IncludeDiff to be true by default")
	}
	if !options.IncludeImpact {
		t.Error("Expected IncludeImpact to be true by default")
	}
	if options.ImpactDepth != 3 {
		t.Errorf("Expected ImpactDepth to be 3, got %d", options.ImpactDepth)
	}
	if !options.IncludeConflicts {
		t.Error("Expected IncludeConflicts to be true by default")
	}
	if !options.IncludeTemporal {
		t.Error("Expected IncludeTemporal to be true by default")
	}
	if !options.IncludeVisualization {
		t.Error("Expected IncludeVisualization to be true by default")
	}
	if len(options.Scenarios) != 0 {
		t.Error("Expected Scenarios to be empty by default")
	}
}

func TestExtractAffectedTitles(t *testing.T) {
	diff := &DraftDiff{
		Modified: []DiffEntry{
			{Amendment: Amendment{TargetTitle: "15"}},
			{Amendment: Amendment{TargetTitle: "42"}},
		},
		Removed: []DiffEntry{
			{Amendment: Amendment{TargetTitle: "15"}}, // duplicate
		},
		Added: []DiffEntry{
			{Amendment: Amendment{TargetTitle: "26"}},
			{Amendment: Amendment{TargetTitle: "invalid"}}, // non-numeric
		},
	}

	titles := extractAffectedTitles(diff)

	// Should have 3 unique titles: 15, 26, 42 (sorted)
	expected := []int{15, 26, 42}
	if len(titles) != len(expected) {
		t.Errorf("Expected %d titles, got %d", len(expected), len(titles))
	}
	for i, title := range titles {
		if title != expected[i] {
			t.Errorf("Expected title %d at index %d, got %d", expected[i], i, title)
		}
	}
}
