package draft

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// RiskLevel classifies the overall risk of proposed legislation based on
// aggregated analysis findings.
type RiskLevel int

const (
	// RiskLow indicates no errors, no broken cross-refs, and no temporal issues.
	RiskLow RiskLevel = iota
	// RiskMedium indicates warnings only, 1-5 broken refs, or temporal findings.
	RiskMedium
	// RiskHigh indicates conflict errors, >5 broken refs, or >50 affected provisions.
	RiskHigh
)

// riskLevelLabels maps risk levels to human-readable strings.
var riskLevelLabels = [...]string{
	RiskLow:    "low",
	RiskMedium: "medium",
	RiskHigh:   "high",
}

// String returns a human-readable label for the risk level.
func (r RiskLevel) String() string {
	if int(r) < len(riskLevelLabels) {
		return riskLevelLabels[r]
	}
	return "unknown"
}

// LegislativeImpactReport aggregates all draft analysis results into a single
// structured report suitable for rendering in multiple formats.
type LegislativeImpactReport struct {
	Bill             *DraftBill             `json:"bill"`
	GeneratedAt      time.Time              `json:"generated_at"`
	RiskLevel        RiskLevel              `json:"risk_level"`
	ExecutiveSummary ExecutiveSummary       `json:"executive_summary"`
	Diff             *DraftDiff             `json:"diff,omitempty"`
	Impact           *DraftImpactResult     `json:"impact,omitempty"`
	Conflicts        *ConflictReport        `json:"conflicts,omitempty"`
	TemporalFindings []TemporalFinding      `json:"temporal_findings,omitempty"`
	ScenarioResults  []*ScenarioComparison  `json:"scenario_results,omitempty"`
	Visualization    string                 `json:"visualization,omitempty"`
}

// ExecutiveSummary provides a condensed overview of the legislative impact
// analysis with key metrics and risk assessment.
type ExecutiveSummary struct {
	BillTitle               string    `json:"bill_title"`
	BillNumber              string    `json:"bill_number"`
	AmendmentCount          int       `json:"amendment_count"`
	TitlesAffected          []int     `json:"titles_affected"`
	ProvisionsModified      int       `json:"provisions_modified"`
	ProvisionsRepealed      int       `json:"provisions_repealed"`
	ProvisionsAdded         int       `json:"provisions_added"`
	TotalProvisionsAffected int       `json:"total_provisions_affected"`
	BrokenCrossRefs         int       `json:"broken_cross_refs"`
	ConflictErrors          int       `json:"conflict_errors"`
	ConflictWarnings        int       `json:"conflict_warnings"`
	ObligationsAdded        int       `json:"obligations_added"`
	ObligationsRemoved      int       `json:"obligations_removed"`
	RightsAdded             int       `json:"rights_added"`
	RightsRemoved           int       `json:"rights_removed"`
	RiskLevel               RiskLevel `json:"risk_level"`
	RiskJustification       string    `json:"risk_justification"`
}

// ReportOptions configures the report generation pipeline.
type ReportOptions struct {
	// IncludeDiff includes the full diff in the report
	IncludeDiff bool
	// IncludeImpact includes transitive impact analysis
	IncludeImpact bool
	// ImpactDepth controls how many levels deep to analyze transitive impact
	ImpactDepth int
	// IncludeConflicts includes obligation conflict detection
	IncludeConflicts bool
	// IncludeTemporal includes temporal consistency analysis
	IncludeTemporal bool
	// IncludeVisualization generates a DOT graph
	IncludeVisualization bool
	// Scenarios lists scenarios to compare (empty = skip scenario comparison)
	Scenarios []string
}

// DefaultReportOptions returns sensible defaults for report generation.
func DefaultReportOptions() ReportOptions {
	return ReportOptions{
		IncludeDiff:          true,
		IncludeImpact:        true,
		ImpactDepth:          3,
		IncludeConflicts:     true,
		IncludeTemporal:      true,
		IncludeVisualization: true,
		Scenarios:            []string{},
	}
}

// GenerateReport orchestrates the full analysis pipeline and aggregates results
// into a LegislativeImpactReport. It computes the diff, impact analysis,
// conflict detection, temporal analysis, and optionally runs scenario comparisons.
func GenerateReport(bill *DraftBill, libraryPath string, options ReportOptions) (*LegislativeImpactReport, error) {
	if bill == nil {
		return nil, fmt.Errorf("bill is nil")
	}

	report := &LegislativeImpactReport{
		Bill:        bill,
		GeneratedAt: time.Now().UTC(),
	}

	// Step 1: Compute diff
	diff, diffErr := ComputeDiff(bill, libraryPath)
	if diffErr != nil {
		// Even if diff fails, we can still generate a partial report
		report.ExecutiveSummary = buildPartialSummary(bill, nil, nil, nil, nil)
		report.RiskLevel, report.ExecutiveSummary.RiskJustification = ComputeRiskLevel(report)
		return report, fmt.Errorf("diff computation failed: %w", diffErr)
	}
	if options.IncludeDiff {
		report.Diff = diff
	}

	// Step 2: Analyze impact (transitive)
	var impact *DraftImpactResult
	if options.IncludeImpact {
		depth := options.ImpactDepth
		if depth < 1 {
			depth = 3
		}
		impact, _ = AnalyzeDraftImpact(diff, libraryPath, depth)
		report.Impact = impact
	}

	// Step 3: Detect conflicts
	var conflicts *ConflictReport
	if options.IncludeConflicts {
		conflicts, _ = DetectObligationConflicts(diff, impact, libraryPath)
		report.Conflicts = conflicts
	}

	// Step 4: Temporal analysis
	if options.IncludeTemporal {
		temporalFindings, _ := AnalyzeTemporalConsistency(diff, libraryPath)
		report.TemporalFindings = temporalFindings
	}

	// Step 5: Generate visualization
	if options.IncludeVisualization && impact != nil {
		visualization, _ := RenderImpactGraph(impact)
		report.Visualization = visualization
	}

	// Step 6: Build executive summary
	report.ExecutiveSummary = SummarizeReport(report)

	// Step 7: Compute risk level
	report.RiskLevel, report.ExecutiveSummary.RiskJustification = ComputeRiskLevel(report)
	report.ExecutiveSummary.RiskLevel = report.RiskLevel

	return report, nil
}

// ComputeRiskLevel analyzes the report findings and returns an overall risk
// classification with justification.
//
// Risk thresholds:
//   - High: any conflict errors, >5 broken cross-refs, or >50 transitively affected provisions
//   - Medium: conflict warnings, 1-5 broken cross-refs, or temporal findings
//   - Low: no errors, no broken refs, no temporal issues
func ComputeRiskLevel(report *LegislativeImpactReport) (RiskLevel, string) {
	if report == nil {
		return RiskLow, "no analysis data available"
	}

	var reasons []string

	// Check for conflict errors (immediate high risk)
	if report.Conflicts != nil && report.Conflicts.Summary.Errors > 0 {
		reasons = append(reasons, fmt.Sprintf("%d conflict error(s) detected", report.Conflicts.Summary.Errors))
		return RiskHigh, strings.Join(reasons, "; ")
	}

	// Check for broken cross-refs
	brokenRefCount := 0
	if report.Impact != nil {
		brokenRefCount = len(report.Impact.BrokenCrossRefs)
	}
	if brokenRefCount > 5 {
		reasons = append(reasons, fmt.Sprintf("%d broken cross-references (>5)", brokenRefCount))
		return RiskHigh, strings.Join(reasons, "; ")
	}

	// Check for large transitive impact
	totalAffected := 0
	if report.Impact != nil {
		totalAffected = report.Impact.TotalProvisionsAffected
	}
	if totalAffected > 50 {
		reasons = append(reasons, fmt.Sprintf("%d provisions affected (>50)", totalAffected))
		return RiskHigh, strings.Join(reasons, "; ")
	}

	// Medium risk indicators
	mediumRisk := false

	// Conflict warnings
	if report.Conflicts != nil && report.Conflicts.Summary.Warnings > 0 {
		reasons = append(reasons, fmt.Sprintf("%d conflict warning(s)", report.Conflicts.Summary.Warnings))
		mediumRisk = true
	}

	// 1-5 broken cross-refs
	if brokenRefCount > 0 && brokenRefCount <= 5 {
		reasons = append(reasons, fmt.Sprintf("%d broken cross-reference(s)", brokenRefCount))
		mediumRisk = true
	}

	// Temporal findings (excluding info-level sunsets)
	temporalIssueCount := 0
	for _, finding := range report.TemporalFindings {
		if finding.Severity != ConflictInfo {
			temporalIssueCount++
		}
	}
	if temporalIssueCount > 0 {
		reasons = append(reasons, fmt.Sprintf("%d temporal issue(s)", temporalIssueCount))
		mediumRisk = true
	}

	if mediumRisk {
		return RiskMedium, strings.Join(reasons, "; ")
	}

	// Low risk
	if len(reasons) == 0 {
		return RiskLow, "no significant issues detected"
	}
	return RiskLow, strings.Join(reasons, "; ")
}

// SummarizeReport extracts key metrics from the full report into an
// ExecutiveSummary structure.
func SummarizeReport(report *LegislativeImpactReport) ExecutiveSummary {
	if report == nil {
		return ExecutiveSummary{}
	}

	summary := ExecutiveSummary{}

	// Bill metadata
	if report.Bill != nil {
		summary.BillTitle = report.Bill.ShortTitle
		if summary.BillTitle == "" {
			summary.BillTitle = report.Bill.Title
		}
		summary.BillNumber = report.Bill.BillNumber
		summary.AmendmentCount = countAmendments(report.Bill)
	}

	// Diff statistics
	if report.Diff != nil {
		summary.ProvisionsModified = len(report.Diff.Modified)
		summary.ProvisionsRepealed = len(report.Diff.Removed)
		summary.ProvisionsAdded = len(report.Diff.Added)
		summary.TitlesAffected = extractAffectedTitles(report.Diff)
	}

	// Impact statistics
	if report.Impact != nil {
		summary.TotalProvisionsAffected = report.Impact.TotalProvisionsAffected
		summary.BrokenCrossRefs = len(report.Impact.BrokenCrossRefs)
		summary.ObligationsAdded = len(report.Impact.ObligationChanges.Added)
		summary.ObligationsRemoved = len(report.Impact.ObligationChanges.Removed)
		summary.RightsAdded = len(report.Impact.RightsChanges.Added)
		summary.RightsRemoved = len(report.Impact.RightsChanges.Removed)
	}

	// Conflict statistics
	if report.Conflicts != nil {
		summary.ConflictErrors = report.Conflicts.Summary.Errors
		summary.ConflictWarnings = report.Conflicts.Summary.Warnings
	}

	return summary
}

// buildPartialSummary creates an ExecutiveSummary when full analysis is not
// available, using only the bill and any available partial results.
func buildPartialSummary(bill *DraftBill, diff *DraftDiff, impact *DraftImpactResult, conflicts *ConflictReport, temporal []TemporalFinding) ExecutiveSummary {
	summary := ExecutiveSummary{}

	if bill != nil {
		summary.BillTitle = bill.ShortTitle
		if summary.BillTitle == "" {
			summary.BillTitle = bill.Title
		}
		summary.BillNumber = bill.BillNumber
		summary.AmendmentCount = countAmendments(bill)
	}

	if diff != nil {
		summary.ProvisionsModified = len(diff.Modified)
		summary.ProvisionsRepealed = len(diff.Removed)
		summary.ProvisionsAdded = len(diff.Added)
		summary.TitlesAffected = extractAffectedTitles(diff)
	}

	if impact != nil {
		summary.TotalProvisionsAffected = impact.TotalProvisionsAffected
		summary.BrokenCrossRefs = len(impact.BrokenCrossRefs)
	}

	if conflicts != nil {
		summary.ConflictErrors = conflicts.Summary.Errors
		summary.ConflictWarnings = conflicts.Summary.Warnings
	}

	return summary
}

// countAmendments tallies all amendments across all sections of a bill.
func countAmendments(bill *DraftBill) int {
	if bill == nil {
		return 0
	}
	total := 0
	for _, section := range bill.Sections {
		total += len(section.Amendments)
	}
	return total
}

// extractAffectedTitles collects unique USC title numbers from a diff.
func extractAffectedTitles(diff *DraftDiff) []int {
	if diff == nil {
		return nil
	}

	titleSet := make(map[int]bool)

	collectFromEntries := func(entries []DiffEntry) {
		for _, entry := range entries {
			if entry.Amendment.TargetTitle != "" {
				var titleNum int
				if _, err := fmt.Sscanf(entry.Amendment.TargetTitle, "%d", &titleNum); err == nil {
					titleSet[titleNum] = true
				}
			}
		}
	}

	collectFromEntries(diff.Modified)
	collectFromEntries(diff.Removed)
	collectFromEntries(diff.Added)
	collectFromEntries(diff.Redesignated)

	var titles []int
	for title := range titleSet {
		titles = append(titles, title)
	}
	sort.Ints(titles)
	return titles
}

// FormatReport renders a LegislativeImpactReport in the specified format.
// Supported formats: "table", "json", "summary"
func FormatReport(report *LegislativeImpactReport, format string) string {
	if report == nil {
		return ""
	}

	switch strings.ToLower(format) {
	case "json":
		return formatReportJSON(report)
	case "summary":
		return formatReportSummary(report)
	case "table":
		fallthrough
	default:
		return formatReportTable(report)
	}
}

// formatReportJSON renders the full report as indented JSON.
func formatReportJSON(report *LegislativeImpactReport) string {
	// Create a copy without the visualization to avoid bloating JSON output
	reportCopy := *report
	reportCopy.Visualization = "" // Omit DOT graph from JSON

	data, err := json.MarshalIndent(reportCopy, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error())
	}
	return string(data)
}

// formatReportSummary renders only the executive summary.
func formatReportSummary(report *LegislativeImpactReport) string {
	var sb strings.Builder

	summary := report.ExecutiveSummary

	sb.WriteString("Legislative Impact Report Summary\n")
	sb.WriteString(strings.Repeat("=", 60) + "\n\n")

	// Bill info
	if summary.BillTitle != "" {
		sb.WriteString(fmt.Sprintf("Bill: %s\n", summary.BillTitle))
	}
	if summary.BillNumber != "" {
		sb.WriteString(fmt.Sprintf("Number: %s\n", summary.BillNumber))
	}
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", report.GeneratedAt.Format(time.RFC3339)))

	// Risk assessment
	riskColor := ""
	switch report.RiskLevel {
	case RiskHigh:
		riskColor = "HIGH"
	case RiskMedium:
		riskColor = "MEDIUM"
	default:
		riskColor = "LOW"
	}
	sb.WriteString(fmt.Sprintf("Risk Level: %s\n", riskColor))
	sb.WriteString(fmt.Sprintf("Justification: %s\n\n", summary.RiskJustification))

	// Key metrics
	sb.WriteString("Key Metrics\n")
	sb.WriteString(strings.Repeat("-", 40) + "\n")
	sb.WriteString(fmt.Sprintf("  Amendments: %d\n", summary.AmendmentCount))
	sb.WriteString(fmt.Sprintf("  Titles Affected: %v\n", summary.TitlesAffected))
	sb.WriteString(fmt.Sprintf("  Provisions Modified: %d\n", summary.ProvisionsModified))
	sb.WriteString(fmt.Sprintf("  Provisions Repealed: %d\n", summary.ProvisionsRepealed))
	sb.WriteString(fmt.Sprintf("  Provisions Added: %d\n", summary.ProvisionsAdded))
	sb.WriteString(fmt.Sprintf("  Total Affected (direct + transitive): %d\n", summary.TotalProvisionsAffected))
	sb.WriteString("\n")

	// Issues
	sb.WriteString("Issues Detected\n")
	sb.WriteString(strings.Repeat("-", 40) + "\n")
	sb.WriteString(fmt.Sprintf("  Broken Cross-References: %d\n", summary.BrokenCrossRefs))
	sb.WriteString(fmt.Sprintf("  Conflict Errors: %d\n", summary.ConflictErrors))
	sb.WriteString(fmt.Sprintf("  Conflict Warnings: %d\n", summary.ConflictWarnings))
	sb.WriteString("\n")

	// Obligation/Rights changes
	sb.WriteString("Obligation & Rights Changes\n")
	sb.WriteString(strings.Repeat("-", 40) + "\n")
	sb.WriteString(fmt.Sprintf("  Obligations Added: %d\n", summary.ObligationsAdded))
	sb.WriteString(fmt.Sprintf("  Obligations Removed: %d\n", summary.ObligationsRemoved))
	sb.WriteString(fmt.Sprintf("  Rights Added: %d\n", summary.RightsAdded))
	sb.WriteString(fmt.Sprintf("  Rights Removed: %d\n", summary.RightsRemoved))

	return sb.String()
}

// formatReportTable renders the full report as a detailed table format.
func formatReportTable(report *LegislativeImpactReport) string {
	var sb strings.Builder

	// Start with summary
	sb.WriteString(formatReportSummary(report))
	sb.WriteString("\n")

	// Temporal findings
	if len(report.TemporalFindings) > 0 {
		sb.WriteString("Temporal Findings\n")
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		for _, finding := range report.TemporalFindings {
			severityLabel := "INFO"
			switch finding.Severity {
			case ConflictError:
				severityLabel = "ERROR"
			case ConflictWarning:
				severityLabel = "WARNING"
			}
			sb.WriteString(fmt.Sprintf("  [%s] %s: %s\n", severityLabel, finding.Type, finding.Description))
		}
		sb.WriteString("\n")
	}

	// Conflict details
	if report.Conflicts != nil && len(report.Conflicts.Conflicts) > 0 {
		sb.WriteString("Conflicts\n")
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		for _, conflict := range report.Conflicts.Conflicts {
			severityLabel := "INFO"
			switch conflict.Severity {
			case ConflictError:
				severityLabel = "ERROR"
			case ConflictWarning:
				severityLabel = "WARNING"
			}
			sb.WriteString(fmt.Sprintf("  [%s] %s\n", severityLabel, conflict.Description))
		}
		sb.WriteString("\n")
	}

	// Broken cross-references
	if report.Impact != nil && len(report.Impact.BrokenCrossRefs) > 0 {
		sb.WriteString("Broken Cross-References\n")
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		for _, ref := range report.Impact.BrokenCrossRefs {
			severityLabel := "INFO"
			switch ref.Severity {
			case SeverityError:
				severityLabel = "ERROR"
			case SeverityWarning:
				severityLabel = "WARNING"
			}
			sb.WriteString(fmt.Sprintf("  [%s] %s -> %s: %s\n",
				severityLabel,
				ref.SourceLabel,
				ref.TargetLabel,
				ref.Reason))
		}
		sb.WriteString("\n")
	}

	// Scenario comparisons
	if len(report.ScenarioResults) > 0 {
		sb.WriteString("Scenario Comparisons\n")
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		for _, comparison := range report.ScenarioResults {
			sum := comparison.GetSummary()
			sb.WriteString(fmt.Sprintf("  %s:\n", comparison.Scenario))
			if sum.HasDifferences {
				sb.WriteString(fmt.Sprintf("    Newly Applicable: %d\n", sum.NewlyApplicable))
				sb.WriteString(fmt.Sprintf("    No Longer Applicable: %d\n", sum.NoLongerApplicable))
				sb.WriteString(fmt.Sprintf("    Changed Relevance: %d\n", sum.ChangedRelevance))
			} else {
				sb.WriteString("    No differences detected\n")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// GetVisualization returns the DOT graph string from the report.
func (r *LegislativeImpactReport) GetVisualization() string {
	if r == nil {
		return ""
	}
	return r.Visualization
}

// HasErrors returns true if the report contains any error-level findings.
func (r *LegislativeImpactReport) HasErrors() bool {
	if r == nil {
		return false
	}

	// Check conflict errors
	if r.Conflicts != nil && r.Conflicts.Summary.Errors > 0 {
		return true
	}

	// Check for error-level broken refs
	if r.Impact != nil {
		for _, ref := range r.Impact.BrokenCrossRefs {
			if ref.Severity == SeverityError {
				return true
			}
		}
	}

	// Check for error-level temporal findings
	for _, finding := range r.TemporalFindings {
		if finding.Severity == ConflictError {
			return true
		}
	}

	return false
}

// HasWarnings returns true if the report contains any warning-level findings.
func (r *LegislativeImpactReport) HasWarnings() bool {
	if r == nil {
		return false
	}

	// Check conflict warnings
	if r.Conflicts != nil && r.Conflicts.Summary.Warnings > 0 {
		return true
	}

	// Check for warning-level broken refs
	if r.Impact != nil {
		for _, ref := range r.Impact.BrokenCrossRefs {
			if ref.Severity == SeverityWarning {
				return true
			}
		}
	}

	// Check for warning-level temporal findings
	for _, finding := range r.TemporalFindings {
		if finding.Severity == ConflictWarning {
			return true
		}
	}

	return false
}
