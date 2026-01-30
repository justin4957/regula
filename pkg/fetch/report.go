package fetch

import (
	"fmt"
	"strings"
)

// FetchReport summarizes the results of a recursive fetch operation.
type FetchReport struct {
	// TotalReferences is the number of external references found in the graph.
	TotalReferences int `json:"total_references"`

	// MappableCount is the number of references that could be mapped to fetchable URLs.
	MappableCount int `json:"mappable_count"`

	// FetchedCount is the number of references successfully fetched from the network.
	FetchedCount int `json:"fetched_count"`

	// CachedCount is the number of results served from the disk cache.
	CachedCount int `json:"cached_count"`

	// FailedCount is the number of fetch attempts that failed.
	FailedCount int `json:"failed_count"`

	// SkippedCount is the number of references skipped (unmappable, domain-blocked, limit-reached).
	SkippedCount int `json:"skipped_count"`

	// Results contains the individual fetch results.
	Results []FetchResult `json:"results"`

	// TriplesAdded is the number of cross-document triples added to the federated graph.
	TriplesAdded int `json:"triples_added"`

	// DryRun indicates this was a plan-only operation with no network calls.
	DryRun bool `json:"dry_run"`
}

// String returns a CLI-friendly summary of the fetch report.
func (report *FetchReport) String() string {
	var summaryBuilder strings.Builder

	if report.DryRun {
		summaryBuilder.WriteString("Fetch Plan (dry-run):\n")
	} else {
		summaryBuilder.WriteString("Fetch Report:\n")
	}

	summaryBuilder.WriteString(fmt.Sprintf("  External references found: %d\n", report.TotalReferences))
	summaryBuilder.WriteString(fmt.Sprintf("  Mappable to URLs:          %d\n", report.MappableCount))

	if !report.DryRun {
		summaryBuilder.WriteString(fmt.Sprintf("  Successfully fetched:      %d\n", report.FetchedCount))
		summaryBuilder.WriteString(fmt.Sprintf("  Served from cache:         %d\n", report.CachedCount))
		summaryBuilder.WriteString(fmt.Sprintf("  Failed:                    %d\n", report.FailedCount))
	}

	summaryBuilder.WriteString(fmt.Sprintf("  Skipped:                   %d\n", report.SkippedCount))

	if !report.DryRun && report.TriplesAdded > 0 {
		summaryBuilder.WriteString(fmt.Sprintf("  Triples added:             %d\n", report.TriplesAdded))
	}

	if len(report.Results) > 0 {
		summaryBuilder.WriteString("\n  References:\n")
		for _, result := range report.Results {
			statusIndicator := "+"
			if !result.Success {
				statusIndicator = "-"
			}
			if result.Cached {
				statusIndicator = "c"
			}

			sourceLabel := ""
			if result.Error != "" {
				sourceLabel = fmt.Sprintf(" (%s)", result.Error)
			}

			summaryBuilder.WriteString(fmt.Sprintf("    [%s] %s → %s%s\n",
				statusIndicator,
				result.Reference.URN,
				result.Reference.URL,
				sourceLabel,
			))
		}
	}

	return summaryBuilder.String()
}

// ToMarkdown returns a Markdown-formatted table of the fetch results.
func (report *FetchReport) ToMarkdown() string {
	var markdownBuilder strings.Builder

	if report.DryRun {
		markdownBuilder.WriteString("## Fetch Plan (dry-run)\n\n")
	} else {
		markdownBuilder.WriteString("## Fetch Report\n\n")
	}

	markdownBuilder.WriteString("| Metric | Count |\n")
	markdownBuilder.WriteString("|---|---|\n")
	markdownBuilder.WriteString(fmt.Sprintf("| External references | %d |\n", report.TotalReferences))
	markdownBuilder.WriteString(fmt.Sprintf("| Mappable to URLs | %d |\n", report.MappableCount))

	if !report.DryRun {
		markdownBuilder.WriteString(fmt.Sprintf("| Fetched | %d |\n", report.FetchedCount))
		markdownBuilder.WriteString(fmt.Sprintf("| Cached | %d |\n", report.CachedCount))
		markdownBuilder.WriteString(fmt.Sprintf("| Failed | %d |\n", report.FailedCount))
	}

	markdownBuilder.WriteString(fmt.Sprintf("| Skipped | %d |\n", report.SkippedCount))

	if !report.DryRun {
		markdownBuilder.WriteString(fmt.Sprintf("| Triples added | %d |\n", report.TriplesAdded))
	}

	if len(report.Results) > 0 {
		markdownBuilder.WriteString("\n### References\n\n")
		markdownBuilder.WriteString("| URN | URL | Status |\n")
		markdownBuilder.WriteString("|---|---|---|\n")

		for _, result := range report.Results {
			status := "fetched"
			if result.Cached {
				status = "cached"
			} else if !result.Success {
				status = "failed"
				if result.Error != "" {
					status = result.Error
				}
			}

			markdownBuilder.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n",
				result.Reference.URN,
				result.Reference.URL,
				status,
			))
		}
	}

	return markdownBuilder.String()
}
