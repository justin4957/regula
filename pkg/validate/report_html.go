package validate

import (
	"fmt"
	"html"
	"strings"
)

// ToHTML generates a self-contained HTML validation report with inline CSS.
func (validationResult *ValidationResult) ToHTML() string {
	var htmlBuilder strings.Builder

	statusColor := statusToHTMLColor(validationResult.Status)
	statusLabel := string(validationResult.Status)

	htmlBuilder.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	htmlBuilder.WriteString("<meta charset=\"UTF-8\">\n")
	htmlBuilder.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	htmlBuilder.WriteString("<title>Validation Report</title>\n")
	htmlBuilder.WriteString(validationHTMLStyles())
	htmlBuilder.WriteString("</head>\n<body>\n")

	// Header
	htmlBuilder.WriteString("<div class=\"container\">\n")
	htmlBuilder.WriteString("<div class=\"header\">\n")
	htmlBuilder.WriteString("<h1>Validation Report</h1>\n")

	if validationResult.ProfileName != "" {
		htmlBuilder.WriteString(fmt.Sprintf("<span class=\"badge\">%s</span>\n",
			html.EscapeString(validationResult.ProfileName)))
	}

	htmlBuilder.WriteString(fmt.Sprintf("<span class=\"status-badge\" style=\"background-color:%s\">%s</span>\n",
		statusColor, statusLabel))
	htmlBuilder.WriteString("</div>\n\n")

	// Overall Score Bar
	htmlBuilder.WriteString("<div class=\"score-section\">\n")
	htmlBuilder.WriteString("<h2>Overall Score</h2>\n")
	htmlBuilder.WriteString(fmt.Sprintf("<div class=\"score-value\">%.1f%%</div>\n", validationResult.OverallScore*100))
	htmlBuilder.WriteString("<div class=\"score-bar-container\">\n")
	htmlBuilder.WriteString(fmt.Sprintf("<div class=\"score-bar\" style=\"width:%.1f%%;background-color:%s\"></div>\n",
		validationResult.OverallScore*100, statusColor))
	htmlBuilder.WriteString("</div>\n")
	htmlBuilder.WriteString(fmt.Sprintf("<div class=\"threshold-label\">Threshold: %.1f%%</div>\n",
		validationResult.Threshold*100))
	htmlBuilder.WriteString("</div>\n\n")

	// Component Scores
	if validationResult.ComponentScores != nil {
		htmlBuilder.WriteString("<div class=\"section\">\n")
		htmlBuilder.WriteString("<h2>Component Scores</h2>\n")

		componentEntries := []struct {
			name   string
			score  float64
			weight float64
		}{
			{"References", validationResult.ComponentScores.ReferenceScore, validationResult.ComponentScores.ReferenceWeight},
			{"Connectivity", validationResult.ComponentScores.ConnectivityScore, validationResult.ComponentScores.ConnectivityWeight},
			{"Definitions", validationResult.ComponentScores.DefinitionScore, validationResult.ComponentScores.DefinitionWeight},
			{"Semantics", validationResult.ComponentScores.SemanticScore, validationResult.ComponentScores.SemanticWeight},
			{"Structure", validationResult.ComponentScores.StructureScore, validationResult.ComponentScores.StructureWeight},
		}

		for _, componentEntry := range componentEntries {
			barColor := scoreToHTMLColor(componentEntry.score)
			htmlBuilder.WriteString("<div class=\"component-row\">\n")
			htmlBuilder.WriteString(fmt.Sprintf("<span class=\"component-name\">%s</span>\n", componentEntry.name))
			htmlBuilder.WriteString(fmt.Sprintf("<span class=\"component-weight\">(%.0f%%)</span>\n", componentEntry.weight*100))
			htmlBuilder.WriteString("<div class=\"component-bar-container\">\n")
			htmlBuilder.WriteString(fmt.Sprintf("<div class=\"component-bar\" style=\"width:%.1f%%;background-color:%s\"></div>\n",
				componentEntry.score*100, barColor))
			htmlBuilder.WriteString("</div>\n")
			htmlBuilder.WriteString(fmt.Sprintf("<span class=\"component-score\">%.1f%%</span>\n", componentEntry.score*100))
			htmlBuilder.WriteString("</div>\n")
		}

		htmlBuilder.WriteString("</div>\n\n")
	}

	// Reference Resolution
	if validationResult.References != nil {
		htmlBuilder.WriteString("<details class=\"section\" open>\n")
		htmlBuilder.WriteString("<summary><h2>Reference Resolution</h2></summary>\n")
		htmlBuilder.WriteString("<table>\n")
		htmlBuilder.WriteString("<tr><th>Metric</th><th>Value</th></tr>\n")
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Total References</td><td>%d</td></tr>\n", validationResult.References.TotalReferences))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Resolved</td><td>%d</td></tr>\n", validationResult.References.Resolved))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Partial</td><td>%d</td></tr>\n", validationResult.References.Partial))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Ambiguous</td><td>%d</td></tr>\n", validationResult.References.Ambiguous))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Not Found</td><td>%d</td></tr>\n", validationResult.References.NotFound))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>External</td><td>%d</td></tr>\n", validationResult.References.External))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Resolution Rate</td><td>%.1f%%</td></tr>\n", validationResult.References.ResolutionRate*100))
		htmlBuilder.WriteString("</table>\n")

		if len(validationResult.References.UnresolvedExamples) > 0 {
			htmlBuilder.WriteString("<h3>Unresolved Examples</h3>\n")
			htmlBuilder.WriteString("<table>\n")
			htmlBuilder.WriteString("<tr><th>Article</th><th>Reference</th><th>Reason</th></tr>\n")
			for _, example := range validationResult.References.UnresolvedExamples {
				htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Art %d</td><td>%s</td><td>%s</td></tr>\n",
					example.SourceArticle,
					html.EscapeString(example.RawText),
					html.EscapeString(example.Reason)))
			}
			htmlBuilder.WriteString("</table>\n")
		}

		htmlBuilder.WriteString("</details>\n\n")
	}

	// Graph Connectivity
	if validationResult.Connectivity != nil {
		htmlBuilder.WriteString("<details class=\"section\" open>\n")
		htmlBuilder.WriteString("<summary><h2>Graph Connectivity</h2></summary>\n")
		htmlBuilder.WriteString("<table>\n")
		htmlBuilder.WriteString("<tr><th>Metric</th><th>Value</th></tr>\n")
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Total Provisions</td><td>%d</td></tr>\n", validationResult.Connectivity.TotalProvisions))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Connected</td><td>%d</td></tr>\n", validationResult.Connectivity.ConnectedCount))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Orphans</td><td>%d</td></tr>\n", validationResult.Connectivity.OrphanCount))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Connectivity Rate</td><td>%.1f%%</td></tr>\n", validationResult.Connectivity.ConnectivityRate*100))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Avg Incoming Refs</td><td>%.1f</td></tr>\n", validationResult.Connectivity.AvgIncomingRefs))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Avg Outgoing Refs</td><td>%.1f</td></tr>\n", validationResult.Connectivity.AvgOutgoingRefs))
		htmlBuilder.WriteString("</table>\n")

		if len(validationResult.Connectivity.MostReferenced) > 0 {
			htmlBuilder.WriteString("<h3>Most Referenced Articles</h3>\n")
			htmlBuilder.WriteString("<table>\n")
			htmlBuilder.WriteString("<tr><th>Article</th><th>References</th></tr>\n")
			for _, articleRefCount := range validationResult.Connectivity.MostReferenced {
				htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Art %d</td><td>%d</td></tr>\n",
					articleRefCount.ArticleNum, articleRefCount.Count))
			}
			htmlBuilder.WriteString("</table>\n")
		}

		htmlBuilder.WriteString("</details>\n\n")
	}

	// Definition Coverage
	if validationResult.Definitions != nil {
		htmlBuilder.WriteString("<details class=\"section\" open>\n")
		htmlBuilder.WriteString("<summary><h2>Definition Coverage</h2></summary>\n")
		htmlBuilder.WriteString("<table>\n")
		htmlBuilder.WriteString("<tr><th>Metric</th><th>Value</th></tr>\n")
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Total Definitions</td><td>%d</td></tr>\n", validationResult.Definitions.TotalDefinitions))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Used Definitions</td><td>%d</td></tr>\n", validationResult.Definitions.UsedDefinitions))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Unused Definitions</td><td>%d</td></tr>\n", validationResult.Definitions.UnusedDefinitions))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Usage Rate</td><td>%.1f%%</td></tr>\n", validationResult.Definitions.UsageRate*100))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Total Usages</td><td>%d</td></tr>\n", validationResult.Definitions.TotalUsages))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Articles Using Terms</td><td>%d</td></tr>\n", validationResult.Definitions.ArticlesWithTerms))
		htmlBuilder.WriteString("</table>\n")

		if len(validationResult.Definitions.UnusedTerms) > 0 {
			htmlBuilder.WriteString("<h3>Unused Terms</h3>\n<ul>\n")
			for _, term := range validationResult.Definitions.UnusedTerms {
				htmlBuilder.WriteString(fmt.Sprintf("<li>%s</li>\n", html.EscapeString(term)))
			}
			htmlBuilder.WriteString("</ul>\n")
		}

		htmlBuilder.WriteString("</details>\n\n")
	}

	// Semantic Extraction
	if validationResult.Semantics != nil {
		htmlBuilder.WriteString("<details class=\"section\" open>\n")
		htmlBuilder.WriteString("<summary><h2>Semantic Extraction</h2></summary>\n")
		htmlBuilder.WriteString("<table>\n")
		htmlBuilder.WriteString("<tr><th>Metric</th><th>Value</th></tr>\n")
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Rights Found</td><td>%d</td></tr>\n", validationResult.Semantics.RightsCount))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Obligations Found</td><td>%d</td></tr>\n", validationResult.Semantics.ObligationsCount))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Articles with Rights</td><td>%d</td></tr>\n", validationResult.Semantics.ArticlesWithRights))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Articles with Obligations</td><td>%d</td></tr>\n", validationResult.Semantics.ArticlesWithOblig))

		regulationLabel := validationResult.Semantics.RegulationType
		if regulationLabel == "" {
			regulationLabel = "known"
		}
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Known %s Rights</td><td>%d/%d</td></tr>\n",
			html.EscapeString(regulationLabel),
			validationResult.Semantics.KnownRightsFound,
			validationResult.Semantics.KnownRightsTotal))
		htmlBuilder.WriteString("</table>\n")

		if len(validationResult.Semantics.MissingRights) > 0 {
			htmlBuilder.WriteString("<h3>Missing Rights</h3>\n<ul>\n")
			for _, right := range validationResult.Semantics.MissingRights {
				htmlBuilder.WriteString(fmt.Sprintf("<li>%s</li>\n", html.EscapeString(right)))
			}
			htmlBuilder.WriteString("</ul>\n")
		}

		htmlBuilder.WriteString("</details>\n\n")
	}

	// Structure Quality
	if validationResult.Structure != nil {
		htmlBuilder.WriteString("<details class=\"section\" open>\n")
		htmlBuilder.WriteString("<summary><h2>Structure Quality</h2></summary>\n")
		htmlBuilder.WriteString("<table>\n")
		htmlBuilder.WriteString("<tr><th>Metric</th><th>Value</th></tr>\n")
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Articles</td><td>%d</td></tr>\n", validationResult.Structure.TotalArticles))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Chapters</td><td>%d</td></tr>\n", validationResult.Structure.TotalChapters))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Sections</td><td>%d</td></tr>\n", validationResult.Structure.TotalSections))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Recitals</td><td>%d</td></tr>\n", validationResult.Structure.TotalRecitals))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Content Rate</td><td>%.1f%%</td></tr>\n", validationResult.Structure.ContentRate*100))
		htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Structure Score</td><td>%.1f%%</td></tr>\n", validationResult.Structure.StructureScore*100))

		if validationResult.Structure.ExpectedArticles > 0 {
			htmlBuilder.WriteString(fmt.Sprintf("<tr><td>Expected Articles</td><td>%d (%.1f%% complete)</td></tr>\n",
				validationResult.Structure.ExpectedArticles, validationResult.Structure.ArticleCompleteness*100))
		}

		htmlBuilder.WriteString("</table>\n")
		htmlBuilder.WriteString("</details>\n\n")
	}

	// Issues
	if len(validationResult.Issues) > 0 {
		htmlBuilder.WriteString("<div class=\"section\">\n")
		htmlBuilder.WriteString("<h2>Issues</h2>\n")
		for _, issue := range validationResult.Issues {
			alertClass := "alert-error"
			if issue.Severity == "warning" {
				alertClass = "alert-warning"
			} else if issue.Severity == "info" {
				alertClass = "alert-info"
			}
			htmlBuilder.WriteString(fmt.Sprintf("<div class=\"alert %s\">\n", alertClass))
			htmlBuilder.WriteString(fmt.Sprintf("<strong>[%s] %s:</strong> %s",
				html.EscapeString(issue.Severity),
				html.EscapeString(issue.Category),
				html.EscapeString(issue.Message)))
			if issue.Count > 0 {
				htmlBuilder.WriteString(fmt.Sprintf(" (count: %d)", issue.Count))
			}
			htmlBuilder.WriteString("\n</div>\n")
		}
		htmlBuilder.WriteString("</div>\n\n")
	}

	// Warnings
	if len(validationResult.Warnings) > 0 {
		htmlBuilder.WriteString("<div class=\"section\">\n")
		htmlBuilder.WriteString("<h2>Warnings</h2>\n")
		for _, warning := range validationResult.Warnings {
			htmlBuilder.WriteString("<div class=\"alert alert-warning\">\n")
			htmlBuilder.WriteString(fmt.Sprintf("<strong>[%s]:</strong> %s\n",
				html.EscapeString(warning.Category),
				html.EscapeString(warning.Message)))
			htmlBuilder.WriteString("</div>\n")
		}
		htmlBuilder.WriteString("</div>\n\n")
	}

	htmlBuilder.WriteString("</div>\n") // container
	htmlBuilder.WriteString("</body>\n</html>\n")

	return htmlBuilder.String()
}

// ToHTML generates a self-contained HTML gate validation report with inline CSS.
func (gateReport *GateReport) ToHTML() string {
	var htmlBuilder strings.Builder

	overallStatusLabel := "PASS"
	overallStatusColor := "#4caf50"
	if !gateReport.OverallPass {
		overallStatusLabel = "FAIL"
		overallStatusColor = "#f44336"
	}

	htmlBuilder.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	htmlBuilder.WriteString("<meta charset=\"UTF-8\">\n")
	htmlBuilder.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	htmlBuilder.WriteString("<title>Gate Validation Report</title>\n")
	htmlBuilder.WriteString(validationHTMLStyles())
	htmlBuilder.WriteString("</head>\n<body>\n")

	// Header
	htmlBuilder.WriteString("<div class=\"container\">\n")
	htmlBuilder.WriteString("<div class=\"header\">\n")
	htmlBuilder.WriteString("<h1>Gate Validation Report</h1>\n")
	htmlBuilder.WriteString(fmt.Sprintf("<span class=\"status-badge\" style=\"background-color:%s\">%s</span>\n",
		overallStatusColor, overallStatusLabel))
	htmlBuilder.WriteString("</div>\n\n")

	// Overall Score
	htmlBuilder.WriteString("<div class=\"score-section\">\n")
	htmlBuilder.WriteString("<h2>Overall Score</h2>\n")
	htmlBuilder.WriteString(fmt.Sprintf("<div class=\"score-value\">%.1f%%</div>\n", gateReport.TotalScore*100))
	htmlBuilder.WriteString("<div class=\"score-bar-container\">\n")
	htmlBuilder.WriteString(fmt.Sprintf("<div class=\"score-bar\" style=\"width:%.1f%%;background-color:%s\"></div>\n",
		gateReport.TotalScore*100, overallStatusColor))
	htmlBuilder.WriteString("</div>\n")
	htmlBuilder.WriteString(fmt.Sprintf("<div class=\"threshold-label\">%d passed, %d failed, %d skipped | Duration: %v</div>\n",
		gateReport.GatesPassed, gateReport.GatesFailed, gateReport.GatesSkipped, gateReport.Duration))

	if gateReport.HaltedAt != "" {
		htmlBuilder.WriteString(fmt.Sprintf("<div class=\"alert alert-warning\">Pipeline halted at gate: %s</div>\n",
			html.EscapeString(gateReport.HaltedAt)))
	}

	htmlBuilder.WriteString("</div>\n\n")

	// Gate Results
	htmlBuilder.WriteString("<div class=\"section\">\n")
	htmlBuilder.WriteString("<h2>Gate Results</h2>\n")

	for _, gateResult := range gateReport.Results {
		gateStatusLabel := "PASS"
		gateStatusColor := "#4caf50"
		if gateResult.Skipped {
			gateStatusLabel = "SKIP"
			gateStatusColor = "#9e9e9e"
		} else if !gateResult.Passed {
			gateStatusLabel = "FAIL"
			gateStatusColor = "#f44336"
		}

		htmlBuilder.WriteString("<div class=\"gate-card\">\n")
		htmlBuilder.WriteString("<div class=\"gate-header\">\n")
		htmlBuilder.WriteString(fmt.Sprintf("<h3>%s</h3>\n", html.EscapeString(gateResult.Gate)))
		htmlBuilder.WriteString(fmt.Sprintf("<span class=\"status-badge\" style=\"background-color:%s\">%s</span>\n",
			gateStatusColor, gateStatusLabel))

		if !gateResult.Skipped {
			htmlBuilder.WriteString(fmt.Sprintf("<span class=\"gate-score\">%.1f%%</span>\n", gateResult.Score*100))
		}

		htmlBuilder.WriteString("</div>\n")

		if gateResult.Skipped {
			htmlBuilder.WriteString(fmt.Sprintf("<p class=\"skip-reason\">%s</p>\n",
				html.EscapeString(gateResult.SkipReason)))
			htmlBuilder.WriteString("</div>\n")
			continue
		}

		// Metrics table
		if len(gateResult.Metrics) > 0 {
			htmlBuilder.WriteString("<table>\n")
			htmlBuilder.WriteString("<tr><th>Metric</th><th>Value</th></tr>\n")
			for metricName, metricValue := range gateResult.Metrics {
				htmlBuilder.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%.1f%%</td></tr>\n",
					html.EscapeString(metricName), metricValue*100))
			}
			htmlBuilder.WriteString("</table>\n")
		}

		// Warnings
		if len(gateResult.Warnings) > 0 {
			for _, gateWarning := range gateResult.Warnings {
				htmlBuilder.WriteString(fmt.Sprintf("<div class=\"alert alert-warning\">[%s] %s</div>\n",
					html.EscapeString(gateWarning.Metric),
					html.EscapeString(gateWarning.Message)))
			}
		}

		// Errors
		if len(gateResult.Errors) > 0 {
			for _, gateError := range gateResult.Errors {
				htmlBuilder.WriteString(fmt.Sprintf("<div class=\"alert alert-error\">[%s] %s</div>\n",
					html.EscapeString(gateError.Metric),
					html.EscapeString(gateError.Message)))
			}
		}

		htmlBuilder.WriteString("</div>\n")
	}

	htmlBuilder.WriteString("</div>\n\n")

	htmlBuilder.WriteString("</div>\n") // container
	htmlBuilder.WriteString("</body>\n</html>\n")

	return htmlBuilder.String()
}

// validationHTMLStyles returns the inline CSS <style> block used by HTML reports.
func validationHTMLStyles() string {
	return `<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f5f5f5; color: #333; line-height: 1.6; }
.container { max-width: 900px; margin: 20px auto; padding: 20px; }
.header { display: flex; align-items: center; gap: 12px; margin-bottom: 24px; flex-wrap: wrap; }
.header h1 { font-size: 24px; }
.badge { background: #e3f2fd; color: #1565c0; padding: 4px 12px; border-radius: 4px; font-size: 14px; font-weight: 600; }
.status-badge { color: white; padding: 4px 12px; border-radius: 4px; font-size: 14px; font-weight: 700; text-transform: uppercase; }
.score-section { background: white; border-radius: 8px; padding: 20px; margin-bottom: 20px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
.score-value { font-size: 36px; font-weight: 700; margin-bottom: 8px; }
.score-bar-container { background: #e0e0e0; border-radius: 4px; height: 12px; overflow: hidden; margin-bottom: 8px; }
.score-bar { height: 100%; border-radius: 4px; transition: width 0.3s; }
.threshold-label { color: #757575; font-size: 14px; }
.section { background: white; border-radius: 8px; padding: 20px; margin-bottom: 20px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
.section h2 { font-size: 18px; margin-bottom: 16px; display: inline; }
.section summary { cursor: pointer; padding: 4px 0; }
.component-row { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.component-name { width: 120px; font-weight: 600; font-size: 14px; }
.component-weight { width: 40px; color: #757575; font-size: 12px; }
.component-bar-container { flex: 1; background: #e0e0e0; border-radius: 4px; height: 8px; overflow: hidden; }
.component-bar { height: 100%; border-radius: 4px; }
.component-score { width: 50px; text-align: right; font-size: 14px; font-weight: 600; }
table { width: 100%; border-collapse: collapse; margin: 12px 0; }
th, td { padding: 8px 12px; text-align: left; border-bottom: 1px solid #e0e0e0; }
th { background: #fafafa; font-weight: 600; font-size: 13px; text-transform: uppercase; color: #757575; }
td { font-size: 14px; }
.alert { padding: 10px 14px; border-radius: 4px; margin: 8px 0; font-size: 14px; }
.alert-error { background: #ffebee; color: #c62828; border-left: 4px solid #f44336; }
.alert-warning { background: #fff8e1; color: #f57f17; border-left: 4px solid #ff9800; }
.alert-info { background: #e3f2fd; color: #1565c0; border-left: 4px solid #2196f3; }
.gate-card { background: #fafafa; border: 1px solid #e0e0e0; border-radius: 8px; padding: 16px; margin-bottom: 12px; }
.gate-header { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; }
.gate-header h3 { font-size: 16px; }
.gate-score { font-size: 18px; font-weight: 700; margin-left: auto; }
.skip-reason { color: #757575; font-style: italic; }
details { border: none; }
details summary { list-style: none; }
details summary::-webkit-details-marker { display: none; }
details[open] summary h2::after { content: " -"; }
details:not([open]) summary h2::after { content: " +"; }
</style>
`
}

// statusToHTMLColor maps a validation status to an HTML color.
func statusToHTMLColor(status ValidationStatus) string {
	switch status {
	case StatusPass:
		return "#4caf50"
	case StatusFail:
		return "#f44336"
	case StatusWarn:
		return "#ff9800"
	default:
		return "#9e9e9e"
	}
}

// scoreToHTMLColor returns a color based on the score value.
func scoreToHTMLColor(score float64) string {
	if score >= 0.8 {
		return "#4caf50"
	}
	if score >= 0.6 {
		return "#ff9800"
	}
	return "#f44336"
}
