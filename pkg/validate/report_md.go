package validate

import (
	"fmt"
	"strings"
)

// ToMarkdown generates a Markdown-formatted validation report suitable for
// GitHub/GitLab rendering, PR comments, and documentation.
func (validationResult *ValidationResult) ToMarkdown() string {
	var markdownBuilder strings.Builder

	// Header with status badge
	statusBadge := statusToMarkdownBadge(validationResult.Status)
	markdownBuilder.WriteString(fmt.Sprintf("# Validation Report %s\n\n", statusBadge))

	// Summary table
	markdownBuilder.WriteString("## Summary\n\n")
	markdownBuilder.WriteString("| Metric | Value |\n")
	markdownBuilder.WriteString("|--------|-------|\n")
	markdownBuilder.WriteString(fmt.Sprintf("| **Overall Score** | %.1f%% |\n", validationResult.OverallScore*100))
	markdownBuilder.WriteString(fmt.Sprintf("| **Threshold** | %.1f%% |\n", validationResult.Threshold*100))
	markdownBuilder.WriteString(fmt.Sprintf("| **Status** | %s %s |\n", statusBadge, validationResult.Status))

	if validationResult.ProfileName != "" {
		markdownBuilder.WriteString(fmt.Sprintf("| **Profile** | %s |\n", validationResult.ProfileName))
	}

	markdownBuilder.WriteString("\n")

	// Component Scores
	if validationResult.ComponentScores != nil {
		markdownBuilder.WriteString("## Component Scores\n\n")
		markdownBuilder.WriteString("| Component | Score | Weight |\n")
		markdownBuilder.WriteString("|-----------|-------|--------|\n")
		markdownBuilder.WriteString(fmt.Sprintf("| References | %.1f%% | %.0f%% |\n",
			validationResult.ComponentScores.ReferenceScore*100,
			validationResult.ComponentScores.ReferenceWeight*100))
		markdownBuilder.WriteString(fmt.Sprintf("| Connectivity | %.1f%% | %.0f%% |\n",
			validationResult.ComponentScores.ConnectivityScore*100,
			validationResult.ComponentScores.ConnectivityWeight*100))
		markdownBuilder.WriteString(fmt.Sprintf("| Definitions | %.1f%% | %.0f%% |\n",
			validationResult.ComponentScores.DefinitionScore*100,
			validationResult.ComponentScores.DefinitionWeight*100))
		markdownBuilder.WriteString(fmt.Sprintf("| Semantics | %.1f%% | %.0f%% |\n",
			validationResult.ComponentScores.SemanticScore*100,
			validationResult.ComponentScores.SemanticWeight*100))
		markdownBuilder.WriteString(fmt.Sprintf("| Structure | %.1f%% | %.0f%% |\n",
			validationResult.ComponentScores.StructureScore*100,
			validationResult.ComponentScores.StructureWeight*100))
		markdownBuilder.WriteString("\n")
	}

	// Reference Resolution
	if validationResult.References != nil {
		markdownBuilder.WriteString("## Reference Resolution\n\n")
		markdownBuilder.WriteString("| Metric | Value |\n")
		markdownBuilder.WriteString("|--------|-------|\n")
		markdownBuilder.WriteString(fmt.Sprintf("| Total References | %d |\n", validationResult.References.TotalReferences))
		markdownBuilder.WriteString(fmt.Sprintf("| Resolved | %d |\n", validationResult.References.Resolved))
		markdownBuilder.WriteString(fmt.Sprintf("| Partial | %d |\n", validationResult.References.Partial))
		markdownBuilder.WriteString(fmt.Sprintf("| Ambiguous | %d |\n", validationResult.References.Ambiguous))
		markdownBuilder.WriteString(fmt.Sprintf("| Not Found | %d |\n", validationResult.References.NotFound))
		markdownBuilder.WriteString(fmt.Sprintf("| External | %d |\n", validationResult.References.External))
		markdownBuilder.WriteString(fmt.Sprintf("| Range Refs | %d |\n", validationResult.References.RangeRefs))
		markdownBuilder.WriteString(fmt.Sprintf("| Resolution Rate | %.1f%% |\n", validationResult.References.ResolutionRate*100))
		markdownBuilder.WriteString("\n")

		// Confidence breakdown
		markdownBuilder.WriteString("**Confidence Distribution:**\n\n")
		markdownBuilder.WriteString(fmt.Sprintf("- High: %d\n", validationResult.References.HighConfidence))
		markdownBuilder.WriteString(fmt.Sprintf("- Medium: %d\n", validationResult.References.MediumConfidence))
		markdownBuilder.WriteString(fmt.Sprintf("- Low: %d\n", validationResult.References.LowConfidence))
		markdownBuilder.WriteString("\n")

		if len(validationResult.References.UnresolvedExamples) > 0 {
			markdownBuilder.WriteString("**Unresolved Examples:**\n\n")
			markdownBuilder.WriteString("| Article | Reference | Reason |\n")
			markdownBuilder.WriteString("|---------|-----------|--------|\n")
			for _, example := range validationResult.References.UnresolvedExamples {
				markdownBuilder.WriteString(fmt.Sprintf("| Art %d | %s | %s |\n",
					example.SourceArticle, escapeMarkdownTableCell(example.RawText), example.Reason))
			}
			markdownBuilder.WriteString("\n")
		}

		if len(validationResult.References.AmbiguousExamples) > 0 {
			markdownBuilder.WriteString("**Ambiguous Examples:**\n\n")
			markdownBuilder.WriteString("| Article | Reference | Reason |\n")
			markdownBuilder.WriteString("|---------|-----------|--------|\n")
			for _, example := range validationResult.References.AmbiguousExamples {
				markdownBuilder.WriteString(fmt.Sprintf("| Art %d | %s | %s |\n",
					example.SourceArticle, escapeMarkdownTableCell(example.RawText), example.Reason))
			}
			markdownBuilder.WriteString("\n")
		}
	}

	// Graph Connectivity
	if validationResult.Connectivity != nil {
		markdownBuilder.WriteString("## Graph Connectivity\n\n")
		markdownBuilder.WriteString("| Metric | Value |\n")
		markdownBuilder.WriteString("|--------|-------|\n")
		markdownBuilder.WriteString(fmt.Sprintf("| Total Provisions | %d |\n", validationResult.Connectivity.TotalProvisions))
		markdownBuilder.WriteString(fmt.Sprintf("| Connected | %d |\n", validationResult.Connectivity.ConnectedCount))
		markdownBuilder.WriteString(fmt.Sprintf("| Orphans | %d |\n", validationResult.Connectivity.OrphanCount))
		markdownBuilder.WriteString(fmt.Sprintf("| Connectivity Rate | %.1f%% |\n", validationResult.Connectivity.ConnectivityRate*100))
		markdownBuilder.WriteString(fmt.Sprintf("| Avg Incoming Refs | %.1f |\n", validationResult.Connectivity.AvgIncomingRefs))
		markdownBuilder.WriteString(fmt.Sprintf("| Avg Outgoing Refs | %.1f |\n", validationResult.Connectivity.AvgOutgoingRefs))
		markdownBuilder.WriteString("\n")

		if len(validationResult.Connectivity.OrphanArticles) > 0 {
			articleStrings := make([]string, len(validationResult.Connectivity.OrphanArticles))
			for i, articleNum := range validationResult.Connectivity.OrphanArticles {
				articleStrings[i] = fmt.Sprintf("%d", articleNum)
			}
			markdownBuilder.WriteString(fmt.Sprintf("**Orphan Articles:** %s\n\n", strings.Join(articleStrings, ", ")))
		}

		if len(validationResult.Connectivity.MostReferenced) > 0 {
			markdownBuilder.WriteString("**Most Referenced Articles:**\n\n")
			markdownBuilder.WriteString("| Article | References |\n")
			markdownBuilder.WriteString("|---------|------------|\n")
			for _, articleRefCount := range validationResult.Connectivity.MostReferenced {
				markdownBuilder.WriteString(fmt.Sprintf("| Art %d | %d |\n",
					articleRefCount.ArticleNum, articleRefCount.Count))
			}
			markdownBuilder.WriteString("\n")
		}
	}

	// Definition Coverage
	if validationResult.Definitions != nil {
		markdownBuilder.WriteString("## Definition Coverage\n\n")
		markdownBuilder.WriteString("| Metric | Value |\n")
		markdownBuilder.WriteString("|--------|-------|\n")
		markdownBuilder.WriteString(fmt.Sprintf("| Total Definitions | %d |\n", validationResult.Definitions.TotalDefinitions))
		markdownBuilder.WriteString(fmt.Sprintf("| Used Definitions | %d |\n", validationResult.Definitions.UsedDefinitions))
		markdownBuilder.WriteString(fmt.Sprintf("| Unused Definitions | %d |\n", validationResult.Definitions.UnusedDefinitions))
		markdownBuilder.WriteString(fmt.Sprintf("| Usage Rate | %.1f%% |\n", validationResult.Definitions.UsageRate*100))
		markdownBuilder.WriteString(fmt.Sprintf("| Total Usages | %d |\n", validationResult.Definitions.TotalUsages))
		markdownBuilder.WriteString(fmt.Sprintf("| Articles Using Terms | %d |\n", validationResult.Definitions.ArticlesWithTerms))
		markdownBuilder.WriteString("\n")

		if len(validationResult.Definitions.UnusedTerms) > 0 {
			markdownBuilder.WriteString("**Unused Terms:**\n\n")
			for _, term := range validationResult.Definitions.UnusedTerms {
				markdownBuilder.WriteString(fmt.Sprintf("- %s\n", term))
			}
			markdownBuilder.WriteString("\n")
		}

		if len(validationResult.Definitions.MostUsedTerms) > 0 {
			markdownBuilder.WriteString("**Most Used Terms:**\n\n")
			markdownBuilder.WriteString("| Term | Usages | Articles |\n")
			markdownBuilder.WriteString("|------|--------|----------|\n")
			for _, termUsageCount := range validationResult.Definitions.MostUsedTerms {
				markdownBuilder.WriteString(fmt.Sprintf("| %s | %d | %d |\n",
					termUsageCount.Term, termUsageCount.UsageCount, termUsageCount.ArticleCount))
			}
			markdownBuilder.WriteString("\n")
		}
	}

	// Semantic Extraction
	if validationResult.Semantics != nil {
		markdownBuilder.WriteString("## Semantic Extraction\n\n")
		markdownBuilder.WriteString("| Metric | Value |\n")
		markdownBuilder.WriteString("|--------|-------|\n")
		markdownBuilder.WriteString(fmt.Sprintf("| Rights Found | %d |\n", validationResult.Semantics.RightsCount))
		markdownBuilder.WriteString(fmt.Sprintf("| Obligations Found | %d |\n", validationResult.Semantics.ObligationsCount))
		markdownBuilder.WriteString(fmt.Sprintf("| Articles with Rights | %d |\n", validationResult.Semantics.ArticlesWithRights))
		markdownBuilder.WriteString(fmt.Sprintf("| Articles with Obligations | %d |\n", validationResult.Semantics.ArticlesWithOblig))

		regulationLabel := validationResult.Semantics.RegulationType
		if regulationLabel == "" {
			regulationLabel = "known"
		}
		markdownBuilder.WriteString(fmt.Sprintf("| Known %s Rights | %d/%d |\n",
			regulationLabel, validationResult.Semantics.KnownRightsFound, validationResult.Semantics.KnownRightsTotal))
		markdownBuilder.WriteString("\n")

		if len(validationResult.Semantics.MissingRights) > 0 {
			markdownBuilder.WriteString("**Missing Rights:**\n\n")
			for _, right := range validationResult.Semantics.MissingRights {
				markdownBuilder.WriteString(fmt.Sprintf("- %s\n", right))
			}
			markdownBuilder.WriteString("\n")
		}

		if len(validationResult.Semantics.RightTypes) > 0 {
			markdownBuilder.WriteString(fmt.Sprintf("**Right Types:** %s\n\n",
				strings.Join(validationResult.Semantics.RightTypes, ", ")))
		}

		if len(validationResult.Semantics.ObligationTypes) > 0 {
			markdownBuilder.WriteString(fmt.Sprintf("**Obligation Types:** %s\n\n",
				strings.Join(validationResult.Semantics.ObligationTypes, ", ")))
		}
	}

	// Structure Quality
	if validationResult.Structure != nil {
		markdownBuilder.WriteString("## Structure Quality\n\n")
		markdownBuilder.WriteString("| Metric | Value |\n")
		markdownBuilder.WriteString("|--------|-------|\n")
		markdownBuilder.WriteString(fmt.Sprintf("| Articles | %d |\n", validationResult.Structure.TotalArticles))
		markdownBuilder.WriteString(fmt.Sprintf("| Chapters | %d |\n", validationResult.Structure.TotalChapters))
		markdownBuilder.WriteString(fmt.Sprintf("| Sections | %d |\n", validationResult.Structure.TotalSections))
		markdownBuilder.WriteString(fmt.Sprintf("| Recitals | %d |\n", validationResult.Structure.TotalRecitals))
		markdownBuilder.WriteString(fmt.Sprintf("| Content Rate | %.1f%% |\n", validationResult.Structure.ContentRate*100))
		markdownBuilder.WriteString(fmt.Sprintf("| Structure Score | %.1f%% |\n", validationResult.Structure.StructureScore*100))

		if validationResult.Structure.ExpectedArticles > 0 {
			markdownBuilder.WriteString(fmt.Sprintf("| Expected Articles | %d (%.1f%% complete) |\n",
				validationResult.Structure.ExpectedArticles, validationResult.Structure.ArticleCompleteness*100))
		}
		if validationResult.Structure.ExpectedChapters > 0 {
			markdownBuilder.WriteString(fmt.Sprintf("| Expected Chapters | %d (%.1f%% complete) |\n",
				validationResult.Structure.ExpectedChapters, validationResult.Structure.ChapterCompleteness*100))
		}

		markdownBuilder.WriteString("\n")
	}

	// Issues
	if len(validationResult.Issues) > 0 {
		markdownBuilder.WriteString("## Issues\n\n")
		markdownBuilder.WriteString("| Severity | Category | Message | Count |\n")
		markdownBuilder.WriteString("|----------|----------|---------|-------|\n")
		for _, issue := range validationResult.Issues {
			countStr := ""
			if issue.Count > 0 {
				countStr = fmt.Sprintf("%d", issue.Count)
			}
			markdownBuilder.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				issue.Severity, issue.Category, escapeMarkdownTableCell(issue.Message), countStr))
		}
		markdownBuilder.WriteString("\n")
	}

	// Warnings
	if len(validationResult.Warnings) > 0 {
		markdownBuilder.WriteString("## Warnings\n\n")
		markdownBuilder.WriteString("| Category | Message |\n")
		markdownBuilder.WriteString("|----------|---------|\n")
		for _, warning := range validationResult.Warnings {
			markdownBuilder.WriteString(fmt.Sprintf("| %s | %s |\n",
				warning.Category, escapeMarkdownTableCell(warning.Message)))
		}
		markdownBuilder.WriteString("\n")
	}

	return markdownBuilder.String()
}

// ToMarkdown generates a Markdown-formatted gate validation report.
func (gateReport *GateReport) ToMarkdown() string {
	var markdownBuilder strings.Builder

	// Header with overall status badge
	overallBadge := "PASS"
	if !gateReport.OverallPass {
		overallBadge = "FAIL"
	}
	markdownBuilder.WriteString(fmt.Sprintf("# Gate Validation Report %s\n\n",
		statusToMarkdownBadge(ValidationStatus(overallBadge))))

	// Summary table
	markdownBuilder.WriteString("## Summary\n\n")
	markdownBuilder.WriteString("| Metric | Value |\n")
	markdownBuilder.WriteString("|--------|-------|\n")
	markdownBuilder.WriteString(fmt.Sprintf("| **Overall Score** | %.1f%% |\n", gateReport.TotalScore*100))
	markdownBuilder.WriteString(fmt.Sprintf("| **Gates Passed** | %d |\n", gateReport.GatesPassed))
	markdownBuilder.WriteString(fmt.Sprintf("| **Gates Failed** | %d |\n", gateReport.GatesFailed))
	markdownBuilder.WriteString(fmt.Sprintf("| **Gates Skipped** | %d |\n", gateReport.GatesSkipped))
	markdownBuilder.WriteString(fmt.Sprintf("| **Duration** | %v |\n", gateReport.Duration))

	if gateReport.HaltedAt != "" {
		markdownBuilder.WriteString(fmt.Sprintf("| **Halted At** | %s |\n", gateReport.HaltedAt))
	}

	markdownBuilder.WriteString("\n")

	// Gate Results
	markdownBuilder.WriteString("## Gate Results\n\n")

	for _, gateResult := range gateReport.Results {
		gateStatusLabel := "PASS"
		if gateResult.Skipped {
			gateStatusLabel = "SKIP"
		} else if !gateResult.Passed {
			gateStatusLabel = "FAIL"
		}

		markdownBuilder.WriteString(fmt.Sprintf("### %s %s (%.1f%%)\n\n",
			statusToMarkdownBadge(ValidationStatus(gateStatusLabel)),
			gateResult.Gate,
			gateResult.Score*100))

		if gateResult.Skipped {
			markdownBuilder.WriteString(fmt.Sprintf("*Skipped: %s*\n\n", gateResult.SkipReason))
			continue
		}

		// Metrics table
		if len(gateResult.Metrics) > 0 {
			markdownBuilder.WriteString("| Metric | Value |\n")
			markdownBuilder.WriteString("|--------|-------|\n")
			for metricName, metricValue := range gateResult.Metrics {
				markdownBuilder.WriteString(fmt.Sprintf("| %s | %.1f%% |\n", metricName, metricValue*100))
			}
			markdownBuilder.WriteString("\n")
		}

		// Warnings
		if len(gateResult.Warnings) > 0 {
			markdownBuilder.WriteString("**Warnings:**\n\n")
			for _, gateWarning := range gateResult.Warnings {
				markdownBuilder.WriteString(fmt.Sprintf("- [%s] %s\n", gateWarning.Metric, gateWarning.Message))
			}
			markdownBuilder.WriteString("\n")
		}

		// Errors
		if len(gateResult.Errors) > 0 {
			markdownBuilder.WriteString("**Errors:**\n\n")
			for _, gateError := range gateResult.Errors {
				markdownBuilder.WriteString(fmt.Sprintf("- [%s] %s\n", gateError.Metric, gateError.Message))
			}
			markdownBuilder.WriteString("\n")
		}
	}

	return markdownBuilder.String()
}

// statusToMarkdownBadge converts a ValidationStatus to a text badge for Markdown.
func statusToMarkdownBadge(status ValidationStatus) string {
	switch status {
	case StatusPass:
		return "`PASS`"
	case StatusFail:
		return "`FAIL`"
	case StatusWarn:
		return "`WARN`"
	case "SKIP":
		return "`SKIP`"
	default:
		return fmt.Sprintf("`%s`", status)
	}
}

// escapeMarkdownTableCell escapes pipe characters in table cell content.
func escapeMarkdownTableCell(content string) string {
	return strings.ReplaceAll(content, "|", "\\|")
}
