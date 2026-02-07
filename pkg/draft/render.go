package draft

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"
)

// RenderReportMarkdown converts a LegislativeImpactReport into a GitHub-flavored
// Markdown document suitable for rendering on GitHub, GitLab, or similar platforms.
func RenderReportMarkdown(report *LegislativeImpactReport) (string, error) {
	if report == nil {
		return "", fmt.Errorf("report is nil")
	}

	var sb strings.Builder

	// Title and risk level
	billTitle := report.ExecutiveSummary.BillTitle
	if billTitle == "" && report.Bill != nil {
		billTitle = report.Bill.BillNumber
	}
	if billTitle == "" {
		billTitle = "Draft Legislation"
	}

	sb.WriteString(fmt.Sprintf("# Legislative Impact Report: %s\n\n", billTitle))

	// Bill number if different from title
	billNumber := report.ExecutiveSummary.BillNumber
	if billNumber != "" && billNumber != billTitle {
		sb.WriteString(fmt.Sprintf("**Bill Number:** %s\n\n", billNumber))
	}

	// Risk level with emoji
	riskEmoji := ""
	riskLabel := ""
	switch report.RiskLevel {
	case RiskHigh:
		riskEmoji = "🔴"
		riskLabel = "HIGH"
	case RiskMedium:
		riskEmoji = "🟠"
		riskLabel = "MEDIUM"
	default:
		riskEmoji = "🟢"
		riskLabel = "LOW"
	}
	sb.WriteString(fmt.Sprintf("**Risk Level: %s** %s\n\n", riskLabel, riskEmoji))

	// Executive Summary
	sb.WriteString("## Executive Summary\n\n")
	summary := report.ExecutiveSummary

	titlesStr := formatTitlesForMarkdown(summary.TitlesAffected)
	sb.WriteString(fmt.Sprintf("- **Amendments:** %d", summary.AmendmentCount))
	if titlesStr != "" {
		sb.WriteString(fmt.Sprintf(" across titles %s", titlesStr))
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("- **Provisions modified:** %d | **Repealed:** %d | **Added:** %d\n",
		summary.ProvisionsModified, summary.ProvisionsRepealed, summary.ProvisionsAdded))

	directCount := 0
	transitiveCount := 0
	if report.Impact != nil {
		directCount = len(report.Impact.DirectlyAffected)
		transitiveCount = len(report.Impact.TransitivelyAffected)
	}
	sb.WriteString(fmt.Sprintf("- **Total provisions affected:** %d (%d direct, %d transitive)\n",
		summary.TotalProvisionsAffected, directCount, transitiveCount))

	sb.WriteString(fmt.Sprintf("- **Broken cross-references:** %d\n", summary.BrokenCrossRefs))

	conflictInfoCount := 0
	if report.Conflicts != nil {
		conflictInfoCount = report.Conflicts.Summary.Infos
	}
	sb.WriteString(fmt.Sprintf("- **Conflicts:** %d errors, %d warnings, %d info\n",
		summary.ConflictErrors, summary.ConflictWarnings, conflictInfoCount))

	sb.WriteString(fmt.Sprintf("- **New obligations:** %d | **Removed:** %d\n",
		summary.ObligationsAdded, summary.ObligationsRemoved))
	sb.WriteString(fmt.Sprintf("- **New rights:** %d | **Removed:** %d\n\n",
		summary.RightsAdded, summary.RightsRemoved))

	// Structural Changes (Diff)
	if report.Diff != nil && hasDiffEntries(report.Diff) {
		sb.WriteString("## Structural Changes\n\n")
		sb.WriteString("| Type | Target | Description |\n")
		sb.WriteString("|------|--------|-------------|\n")

		for _, entry := range report.Diff.Modified {
			desc := entry.Amendment.Description
			if desc == "" {
				desc = "Strike and insert"
			}
			sb.WriteString(fmt.Sprintf("| Modified | %s | %s |\n",
				formatTargetForMarkdown(entry.Amendment), truncateMarkdown(desc, 50)))
		}
		for _, entry := range report.Diff.Removed {
			desc := entry.Amendment.Description
			if desc == "" {
				desc = "Repeal"
			}
			sb.WriteString(fmt.Sprintf("| Repealed | %s | %s |\n",
				formatTargetForMarkdown(entry.Amendment), truncateMarkdown(desc, 50)))
		}
		for _, entry := range report.Diff.Added {
			desc := entry.Amendment.Description
			if desc == "" {
				desc = "Add new section"
			}
			sb.WriteString(fmt.Sprintf("| Added | %s | %s |\n",
				formatTargetForMarkdown(entry.Amendment), truncateMarkdown(desc, 50)))
		}
		for _, entry := range report.Diff.Redesignated {
			desc := entry.Amendment.Description
			if desc == "" {
				desc = "Redesignate"
			}
			sb.WriteString(fmt.Sprintf("| Redesignated | %s | %s |\n",
				formatTargetForMarkdown(entry.Amendment), truncateMarkdown(desc, 50)))
		}
		sb.WriteString("\n")
	}

	// Impact Analysis
	if report.Impact != nil && (len(report.Impact.DirectlyAffected) > 0 || len(report.Impact.TransitivelyAffected) > 0) {
		sb.WriteString("## Impact Analysis\n\n")

		if len(report.Impact.DirectlyAffected) > 0 {
			sb.WriteString("### Directly Affected Provisions\n\n")
			sb.WriteString("| Provision | Reason |\n")
			sb.WriteString("|-----------|--------|\n")
			for _, prov := range report.Impact.DirectlyAffected {
				label := prov.Label
				if label == "" {
					label = extractURILabel(prov.URI)
				}
				sb.WriteString(fmt.Sprintf("| %s | %s |\n", label, truncateMarkdown(prov.Reason, 60)))
			}
			sb.WriteString("\n")
		}

		if len(report.Impact.TransitivelyAffected) > 0 {
			sb.WriteString("### Transitively Affected Provisions\n\n")
			sb.WriteString("| Provision | Depth | Reason |\n")
			sb.WriteString("|-----------|-------|--------|\n")
			for _, prov := range report.Impact.TransitivelyAffected {
				label := prov.Label
				if label == "" {
					label = extractURILabel(prov.URI)
				}
				sb.WriteString(fmt.Sprintf("| %s | %d | %s |\n", label, prov.Depth, truncateMarkdown(prov.Reason, 50)))
			}
			sb.WriteString("\n")
		}
	}

	// Conflict Findings
	if report.Conflicts != nil && len(report.Conflicts.Conflicts) > 0 {
		sb.WriteString("## Conflict Findings\n\n")
		sb.WriteString("| Severity | Type | Description |\n")
		sb.WriteString("|----------|------|-------------|\n")

		for _, conflict := range report.Conflicts.Conflicts {
			severityLabel := "ℹ️ Info"
			switch conflict.Severity {
			case ConflictError:
				severityLabel = "🔴 Error"
			case ConflictWarning:
				severityLabel = "🟠 Warning"
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
				severityLabel, conflict.Type, truncateMarkdown(conflict.Description, 60)))
		}
		sb.WriteString("\n")
	}

	// Temporal Analysis
	if len(report.TemporalFindings) > 0 {
		sb.WriteString("## Temporal Analysis\n\n")
		sb.WriteString("| Severity | Type | Finding |\n")
		sb.WriteString("|----------|------|--------|\n")

		for _, finding := range report.TemporalFindings {
			severityLabel := "ℹ️ Info"
			switch finding.Severity {
			case ConflictError:
				severityLabel = "🔴 Error"
			case ConflictWarning:
				severityLabel = "🟠 Warning"
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
				severityLabel, finding.Type, truncateMarkdown(finding.Description, 60)))
		}
		sb.WriteString("\n")
	}

	// Broken Cross-References
	if report.Impact != nil && len(report.Impact.BrokenCrossRefs) > 0 {
		sb.WriteString("## Broken Cross-References\n\n")
		sb.WriteString("| Severity | Source | Target | Reason |\n")
		sb.WriteString("|----------|--------|--------|--------|\n")

		for _, ref := range report.Impact.BrokenCrossRefs {
			severityLabel := "ℹ️ Info"
			switch ref.Severity {
			case SeverityError:
				severityLabel = "🔴 Error"
			case SeverityWarning:
				severityLabel = "🟠 Warning"
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				severityLabel, ref.SourceLabel, ref.TargetLabel, truncateMarkdown(ref.Reason, 40)))
		}
		sb.WriteString("\n")
	}

	// Scenario Comparisons
	if len(report.ScenarioResults) > 0 {
		sb.WriteString("## Scenario Comparisons\n\n")

		for _, comparison := range report.ScenarioResults {
			sb.WriteString(fmt.Sprintf("### %s\n\n", comparison.Scenario))
			sum := comparison.GetSummary()

			if !sum.HasDifferences {
				sb.WriteString("No differences detected between baseline and proposed.\n\n")
				continue
			}

			sb.WriteString("| Metric | Count |\n")
			sb.WriteString("|--------|-------|\n")
			sb.WriteString(fmt.Sprintf("| Newly Applicable | %d |\n", sum.NewlyApplicable))
			sb.WriteString(fmt.Sprintf("| No Longer Applicable | %d |\n", sum.NoLongerApplicable))
			sb.WriteString(fmt.Sprintf("| Changed Relevance | %d |\n", sum.ChangedRelevance))
			sb.WriteString(fmt.Sprintf("| Obligations Added | %d |\n", sum.ObligationsAdded))
			sb.WriteString(fmt.Sprintf("| Obligations Removed | %d |\n", sum.ObligationsRemoved))
			sb.WriteString("\n")
		}
	}

	// Visualization (as code block)
	if report.Visualization != "" {
		sb.WriteString("## Impact Visualization\n\n")
		sb.WriteString("```dot\n")
		sb.WriteString(report.Visualization)
		if !strings.HasSuffix(report.Visualization, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("```\n\n")
	}

	// Footer
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("*Generated by regula on %s*\n", report.GeneratedAt.Format("2006-01-02")))

	return sb.String(), nil
}

// RenderReportJSON converts a LegislativeImpactReport into an indented JSON string.
// All nested structs are included, timestamps are in RFC3339 format.
func RenderReportJSON(report *LegislativeImpactReport) (string, error) {
	if report == nil {
		return "", fmt.Errorf("report is nil")
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal report to JSON: %w", err)
	}

	return string(data), nil
}

// RenderReportHTML converts a LegislativeImpactReport into a self-contained HTML
// document with inline CSS styling. No external dependencies are required.
func RenderReportHTML(report *LegislativeImpactReport) (string, error) {
	if report == nil {
		return "", fmt.Errorf("report is nil")
	}

	var sb strings.Builder

	// HTML header with inline CSS
	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Legislative Impact Report</title>
<style>
:root {
  --risk-high: #dc3545;
  --risk-medium: #fd7e14;
  --risk-low: #28a745;
  --bg-light: #f8f9fa;
  --border-color: #dee2e6;
  --text-color: #212529;
  --text-muted: #6c757d;
}
* { box-sizing: border-box; }
body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  line-height: 1.6;
  color: var(--text-color);
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
  background: #fff;
}
h1, h2, h3 { margin-top: 1.5em; margin-bottom: 0.5em; }
h1 { border-bottom: 2px solid var(--border-color); padding-bottom: 0.3em; }
h2 { border-bottom: 1px solid var(--border-color); padding-bottom: 0.2em; }
.risk-header {
  display: inline-block;
  padding: 8px 16px;
  border-radius: 4px;
  color: white;
  font-weight: bold;
  font-size: 1.1em;
  margin-bottom: 1em;
}
.risk-high { background-color: var(--risk-high); }
.risk-medium { background-color: var(--risk-medium); }
.risk-low { background-color: var(--risk-low); }
.summary-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 15px;
  margin: 1em 0;
}
.summary-card {
  background: var(--bg-light);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 15px;
}
.summary-card h4 {
  margin: 0 0 10px 0;
  color: var(--text-muted);
  font-size: 0.9em;
  text-transform: uppercase;
}
.summary-card .value {
  font-size: 1.8em;
  font-weight: bold;
}
table {
  width: 100%;
  border-collapse: collapse;
  margin: 1em 0;
  font-size: 0.95em;
}
th, td {
  padding: 10px 12px;
  text-align: left;
  border-bottom: 1px solid var(--border-color);
}
th {
  background: var(--bg-light);
  font-weight: 600;
}
tr:hover { background: #f1f3f4; }
.severity-error { color: var(--risk-high); font-weight: bold; }
.severity-warning { color: var(--risk-medium); font-weight: bold; }
.severity-info { color: var(--text-muted); }
.collapsible {
  cursor: pointer;
  user-select: none;
}
.collapsible::before {
  content: "▶ ";
  display: inline-block;
  transition: transform 0.2s;
}
.collapsible.open::before {
  transform: rotate(90deg);
}
.content { display: none; }
.content.show { display: block; }
pre {
  background: var(--bg-light);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  padding: 15px;
  overflow-x: auto;
  font-size: 0.85em;
}
.footer {
  margin-top: 3em;
  padding-top: 1em;
  border-top: 1px solid var(--border-color);
  color: var(--text-muted);
  font-size: 0.9em;
}
</style>
</head>
<body>
`)

	// Title
	billTitle := report.ExecutiveSummary.BillTitle
	if billTitle == "" && report.Bill != nil {
		billTitle = report.Bill.BillNumber
	}
	if billTitle == "" {
		billTitle = "Draft Legislation"
	}
	sb.WriteString(fmt.Sprintf("<h1>Legislative Impact Report: %s</h1>\n", html.EscapeString(billTitle)))

	// Risk level header
	riskClass := "risk-low"
	riskLabel := "LOW RISK"
	switch report.RiskLevel {
	case RiskHigh:
		riskClass = "risk-high"
		riskLabel = "HIGH RISK"
	case RiskMedium:
		riskClass = "risk-medium"
		riskLabel = "MEDIUM RISK"
	}
	sb.WriteString(fmt.Sprintf("<div class=\"risk-header %s\">%s</div>\n", riskClass, riskLabel))

	if report.ExecutiveSummary.RiskJustification != "" {
		sb.WriteString(fmt.Sprintf("<p><em>%s</em></p>\n", html.EscapeString(report.ExecutiveSummary.RiskJustification)))
	}

	// Executive Summary as cards
	sb.WriteString("<h2>Executive Summary</h2>\n")
	sb.WriteString("<div class=\"summary-grid\">\n")

	summary := report.ExecutiveSummary
	writeHTMLSummaryCard(&sb, "Amendments", fmt.Sprintf("%d", summary.AmendmentCount))
	writeHTMLSummaryCard(&sb, "Modified", fmt.Sprintf("%d", summary.ProvisionsModified))
	writeHTMLSummaryCard(&sb, "Repealed", fmt.Sprintf("%d", summary.ProvisionsRepealed))
	writeHTMLSummaryCard(&sb, "Added", fmt.Sprintf("%d", summary.ProvisionsAdded))
	writeHTMLSummaryCard(&sb, "Total Affected", fmt.Sprintf("%d", summary.TotalProvisionsAffected))
	writeHTMLSummaryCard(&sb, "Broken Refs", fmt.Sprintf("%d", summary.BrokenCrossRefs))
	writeHTMLSummaryCard(&sb, "Conflict Errors", fmt.Sprintf("%d", summary.ConflictErrors))
	writeHTMLSummaryCard(&sb, "Conflict Warnings", fmt.Sprintf("%d", summary.ConflictWarnings))

	sb.WriteString("</div>\n")

	// Obligation and Rights changes
	if summary.ObligationsAdded > 0 || summary.ObligationsRemoved > 0 || summary.RightsAdded > 0 || summary.RightsRemoved > 0 {
		sb.WriteString("<h3>Obligation &amp; Rights Changes</h3>\n")
		sb.WriteString("<div class=\"summary-grid\">\n")
		writeHTMLSummaryCard(&sb, "Obligations Added", fmt.Sprintf("%d", summary.ObligationsAdded))
		writeHTMLSummaryCard(&sb, "Obligations Removed", fmt.Sprintf("%d", summary.ObligationsRemoved))
		writeHTMLSummaryCard(&sb, "Rights Added", fmt.Sprintf("%d", summary.RightsAdded))
		writeHTMLSummaryCard(&sb, "Rights Removed", fmt.Sprintf("%d", summary.RightsRemoved))
		sb.WriteString("</div>\n")
	}

	// Structural Changes
	if report.Diff != nil && hasDiffEntries(report.Diff) {
		sb.WriteString("<h2>Structural Changes</h2>\n")
		sb.WriteString("<table>\n")
		sb.WriteString("<tr><th>Type</th><th>Target</th><th>Description</th></tr>\n")

		writeDiffRowsHTML(&sb, report.Diff.Modified, "Modified")
		writeDiffRowsHTML(&sb, report.Diff.Removed, "Repealed")
		writeDiffRowsHTML(&sb, report.Diff.Added, "Added")
		writeDiffRowsHTML(&sb, report.Diff.Redesignated, "Redesignated")

		sb.WriteString("</table>\n")
	}

	// Conflict Findings
	if report.Conflicts != nil && len(report.Conflicts.Conflicts) > 0 {
		sb.WriteString("<h2>Conflict Findings</h2>\n")
		sb.WriteString("<table>\n")
		sb.WriteString("<tr><th>Severity</th><th>Type</th><th>Description</th></tr>\n")

		for _, conflict := range report.Conflicts.Conflicts {
			severityClass := "severity-info"
			severityLabel := "Info"
			switch conflict.Severity {
			case ConflictError:
				severityClass = "severity-error"
				severityLabel = "Error"
			case ConflictWarning:
				severityClass = "severity-warning"
				severityLabel = "Warning"
			}
			sb.WriteString(fmt.Sprintf("<tr><td class=\"%s\">%s</td><td>%s</td><td>%s</td></tr>\n",
				severityClass, severityLabel,
				html.EscapeString(conflict.Type.String()),
				html.EscapeString(conflict.Description)))
		}

		sb.WriteString("</table>\n")
	}

	// Temporal Findings
	if len(report.TemporalFindings) > 0 {
		sb.WriteString("<h2>Temporal Analysis</h2>\n")
		sb.WriteString("<table>\n")
		sb.WriteString("<tr><th>Severity</th><th>Type</th><th>Finding</th></tr>\n")

		for _, finding := range report.TemporalFindings {
			severityClass := "severity-info"
			severityLabel := "Info"
			switch finding.Severity {
			case ConflictError:
				severityClass = "severity-error"
				severityLabel = "Error"
			case ConflictWarning:
				severityClass = "severity-warning"
				severityLabel = "Warning"
			}
			sb.WriteString(fmt.Sprintf("<tr><td class=\"%s\">%s</td><td>%s</td><td>%s</td></tr>\n",
				severityClass, severityLabel,
				html.EscapeString(finding.Type.String()),
				html.EscapeString(finding.Description)))
		}

		sb.WriteString("</table>\n")
	}

	// Broken Cross-References
	if report.Impact != nil && len(report.Impact.BrokenCrossRefs) > 0 {
		sb.WriteString("<h2>Broken Cross-References</h2>\n")
		sb.WriteString("<table>\n")
		sb.WriteString("<tr><th>Severity</th><th>Source</th><th>Target</th><th>Reason</th></tr>\n")

		for _, ref := range report.Impact.BrokenCrossRefs {
			severityClass := "severity-info"
			severityLabel := "Info"
			switch ref.Severity {
			case SeverityError:
				severityClass = "severity-error"
				severityLabel = "Error"
			case SeverityWarning:
				severityClass = "severity-warning"
				severityLabel = "Warning"
			}
			sb.WriteString(fmt.Sprintf("<tr><td class=\"%s\">%s</td><td>%s</td><td>%s</td><td>%s</td></tr>\n",
				severityClass, severityLabel,
				html.EscapeString(ref.SourceLabel),
				html.EscapeString(ref.TargetLabel),
				html.EscapeString(ref.Reason)))
		}

		sb.WriteString("</table>\n")
	}

	// Scenario Comparisons
	if len(report.ScenarioResults) > 0 {
		sb.WriteString("<h2>Scenario Comparisons</h2>\n")

		for _, comparison := range report.ScenarioResults {
			sb.WriteString(fmt.Sprintf("<h3>%s</h3>\n", html.EscapeString(comparison.Scenario)))
			sum := comparison.GetSummary()

			if !sum.HasDifferences {
				sb.WriteString("<p>No differences detected between baseline and proposed.</p>\n")
				continue
			}

			sb.WriteString("<table>\n")
			sb.WriteString("<tr><th>Metric</th><th>Count</th></tr>\n")
			sb.WriteString(fmt.Sprintf("<tr><td>Newly Applicable</td><td>%d</td></tr>\n", sum.NewlyApplicable))
			sb.WriteString(fmt.Sprintf("<tr><td>No Longer Applicable</td><td>%d</td></tr>\n", sum.NoLongerApplicable))
			sb.WriteString(fmt.Sprintf("<tr><td>Changed Relevance</td><td>%d</td></tr>\n", sum.ChangedRelevance))
			sb.WriteString("</table>\n")
		}
	}

	// Visualization
	if report.Visualization != "" {
		sb.WriteString("<h2>Impact Visualization</h2>\n")
		sb.WriteString("<details>\n")
		sb.WriteString("<summary>View DOT Graph Source</summary>\n")
		sb.WriteString("<pre>\n")
		sb.WriteString(html.EscapeString(report.Visualization))
		sb.WriteString("</pre>\n")
		sb.WriteString("</details>\n")
	}

	// Footer
	sb.WriteString("<div class=\"footer\">\n")
	sb.WriteString(fmt.Sprintf("Generated by regula on %s\n", report.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString("</div>\n")

	// Collapsible sections script
	sb.WriteString(`<script>
document.querySelectorAll('.collapsible').forEach(function(el) {
  el.addEventListener('click', function() {
    this.classList.toggle('open');
    var content = this.nextElementSibling;
    if (content.classList.contains('show')) {
      content.classList.remove('show');
    } else {
      content.classList.add('show');
    }
  });
});
</script>
`)

	sb.WriteString("</body>\n</html>\n")

	return sb.String(), nil
}

// Helper functions

// formatTitlesForMarkdown formats a slice of title numbers for display.
func formatTitlesForMarkdown(titles []int) string {
	if len(titles) == 0 {
		return ""
	}
	strs := make([]string, len(titles))
	for i, t := range titles {
		strs[i] = fmt.Sprintf("%d", t)
	}
	return strings.Join(strs, ", ")
}

// formatTargetForMarkdown formats an amendment target reference.
func formatTargetForMarkdown(amendment Amendment) string {
	target := fmt.Sprintf("%s U.S.C. § %s", amendment.TargetTitle, amendment.TargetSection)
	if amendment.TargetSubsection != "" {
		target += fmt.Sprintf("(%s)", amendment.TargetSubsection)
	}
	return target
}

// truncateMarkdown truncates text to maxLen, adding "..." if needed.
func truncateMarkdown(text string, maxLen int) string {
	// Replace newlines with spaces for table cells
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", "")
	text = strings.TrimSpace(text)

	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}

// hasDiffEntries checks if a diff has any entries.
func hasDiffEntries(diff *DraftDiff) bool {
	return len(diff.Modified) > 0 || len(diff.Removed) > 0 ||
		len(diff.Added) > 0 || len(diff.Redesignated) > 0
}

// writeHTMLSummaryCard writes a summary card div.
func writeHTMLSummaryCard(sb *strings.Builder, title, value string) {
	sb.WriteString("<div class=\"summary-card\">\n")
	sb.WriteString(fmt.Sprintf("  <h4>%s</h4>\n", html.EscapeString(title)))
	sb.WriteString(fmt.Sprintf("  <div class=\"value\">%s</div>\n", html.EscapeString(value)))
	sb.WriteString("</div>\n")
}

// writeDiffRowsHTML writes table rows for diff entries.
func writeDiffRowsHTML(sb *strings.Builder, entries []DiffEntry, changeType string) {
	for _, entry := range entries {
		desc := entry.Amendment.Description
		if desc == "" {
			switch changeType {
			case "Modified":
				desc = "Strike and insert"
			case "Repealed":
				desc = "Repeal"
			case "Added":
				desc = "Add new section"
			case "Redesignated":
				desc = "Redesignate"
			}
		}
		target := formatTargetForMarkdown(entry.Amendment)
		sb.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td></tr>\n",
			html.EscapeString(changeType),
			html.EscapeString(target),
			html.EscapeString(truncateMarkdown(desc, 80))))
	}
}
