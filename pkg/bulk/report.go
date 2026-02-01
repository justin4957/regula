package bulk

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// PrintDownloadProgress is a ProgressCallback that prints a progress bar.
func PrintDownloadProgress(bytesDownloaded int64, totalBytes int64) {
	if totalBytes > 0 {
		percentage := float64(bytesDownloaded) / float64(totalBytes) * 100
		barLength := int(percentage / 2)
		if barLength > 50 {
			barLength = 50
		}
		fmt.Printf("\r  [%-50s] %.1f%% (%s / %s)",
			strings.Repeat("=", barLength)+strings.Repeat(" ", 50-barLength),
			percentage,
			FormatBytes(bytesDownloaded),
			FormatBytes(totalBytes))
	} else {
		fmt.Printf("\r  Downloaded: %s", FormatBytes(bytesDownloaded))
	}
}

// FormatBytes converts byte count to human-readable format.
func FormatBytes(byteCount int64) string {
	switch {
	case byteCount >= 1024*1024*1024:
		return fmt.Sprintf("%.1f GB", float64(byteCount)/(1024*1024*1024))
	case byteCount >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(byteCount)/(1024*1024))
	case byteCount >= 1024:
		return fmt.Sprintf("%.1f KB", float64(byteCount)/1024)
	default:
		return fmt.Sprintf("%d B", byteCount)
	}
}

// FormatDatasetTable formats a list of datasets as a table.
func FormatDatasetTable(datasets []Dataset) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("%-30s %-12s %-8s %s\n",
		"IDENTIFIER", "JURISDICTION", "FORMAT", "DISPLAY NAME"))
	builder.WriteString(strings.Repeat("─", 90) + "\n")

	for _, dataset := range datasets {
		displayName := dataset.DisplayName
		if len(displayName) > 40 {
			displayName = displayName[:37] + "..."
		}
		builder.WriteString(fmt.Sprintf("%-30s %-12s %-8s %s\n",
			dataset.Identifier, dataset.Jurisdiction, dataset.Format, displayName))
	}

	builder.WriteString(fmt.Sprintf("\nTotal: %d datasets\n", len(datasets)))

	return builder.String()
}

// FormatIngestReport formats an IngestReport for terminal output.
func FormatIngestReport(report *IngestReport) string {
	var builder strings.Builder

	builder.WriteString("\nBulk Ingest Report\n")
	builder.WriteString(strings.Repeat("═", 80) + "\n")
	builder.WriteString(fmt.Sprintf("Attempted: %d | Succeeded: %d | Skipped: %d | Failed: %d\n",
		report.TotalAttempted, report.Succeeded, report.Skipped, report.Failed))
	builder.WriteString(strings.Repeat("─", 80) + "\n")

	for _, entry := range report.Entries {
		status := entry.Status
		switch status {
		case "ingested":
			status = "[OK]"
		case "skipped":
			status = "[SKIP]"
		case "failed":
			status = "[FAIL]"
		}

		line := fmt.Sprintf("  %-8s %-35s", status, entry.DocumentID)
		if entry.Triples > 0 {
			line += fmt.Sprintf(" %6d triples", entry.Triples)
			if entry.Articles > 0 {
				line += fmt.Sprintf("  %5d articles", entry.Articles)
			}
			if entry.Chapters > 0 {
				line += fmt.Sprintf("  %3d chapters", entry.Chapters)
			}
		}
		if entry.Duration > 0 {
			line += fmt.Sprintf("  [%s]", formatDuration(entry.Duration))
		}
		if entry.Error != "" && entry.Error != "dry run" {
			line += fmt.Sprintf("  error: %s", entry.Error)
		}
		builder.WriteString(line + "\n")
	}

	// Aggregate totals
	if report.TotalTriples > 0 {
		builder.WriteString(strings.Repeat("─", 80) + "\n")
		builder.WriteString(fmt.Sprintf("  Totals:  %d triples | %d articles | %d chapters\n",
			report.TotalTriples, report.TotalArticles, report.TotalChapters))
		if report.TotalDefinitions > 0 || report.TotalReferences > 0 {
			builder.WriteString(fmt.Sprintf("           %d definitions | %d references | %d rights | %d obligations\n",
				report.TotalDefinitions, report.TotalReferences, report.TotalRights, report.TotalObligations))
		}
	}

	return builder.String()
}

// FormatIngestReportJSON formats an IngestReport as JSON.
func FormatIngestReportJSON(report *IngestReport) string {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	return string(data)
}

// DocumentStatsSummary holds per-document statistics for status reporting.
type DocumentStatsSummary struct {
	Triples  int
	Articles int
	Chapters int
	Status   string
}

// FormatStatusReport formats download and ingest status for terminal output.
// If documentStats is non-nil, ingest status is shown alongside each download.
func FormatStatusReport(manifest *DownloadManifest, sourceFilter string, documentStats map[string]*DocumentStatsSummary) string {
	var builder strings.Builder

	builder.WriteString("\nBulk Download & Ingest Status\n")
	builder.WriteString(strings.Repeat("═", 90) + "\n")

	sources := AllSourceNames()
	if sourceFilter != "" {
		sources = []string{sourceFilter}
	}

	var grandTotalTriples, grandTotalArticles, grandTotalChapters int
	var totalIngested, totalDownloads int

	for _, sourceName := range sources {
		count := manifest.CountBySource(sourceName)
		totalSize := manifest.TotalSizeBySource(sourceName)

		builder.WriteString(fmt.Sprintf("\n  %-12s  %3d downloads  %s\n",
			sourceName, count, FormatBytes(totalSize)))

		if count == 0 {
			continue
		}

		builder.WriteString(fmt.Sprintf("    %-30s  %-10s  %-8s  %s\n",
			"IDENTIFIER", "SIZE", "STATUS", "TRIPLES"))
		builder.WriteString("    " + strings.Repeat("─", 70) + "\n")

		// List individual downloads for this source
		for _, record := range manifest.Downloads {
			if record.SourceName != sourceName {
				continue
			}

			totalDownloads++
			documentID := deriveDocumentID(record)
			ingestStatus := "pending"
			triplesInfo := ""

			if documentStats != nil {
				if stats, exists := documentStats[documentID]; exists {
					ingestStatus = stats.Status
					if stats.Triples > 0 {
						triplesInfo = fmt.Sprintf("%d triples, %d articles, %d chapters",
							stats.Triples, stats.Articles, stats.Chapters)
						grandTotalTriples += stats.Triples
						grandTotalArticles += stats.Articles
						grandTotalChapters += stats.Chapters
						totalIngested++
					}
				}
			}

			builder.WriteString(fmt.Sprintf("    %-30s  %-10s  %-8s  %s\n",
				record.Identifier,
				FormatBytes(record.SizeBytes),
				ingestStatus,
				triplesInfo))
		}
	}

	builder.WriteString(strings.Repeat("─", 90) + "\n")
	builder.WriteString(fmt.Sprintf("  Downloads: %d | Ingested: %d\n", totalDownloads, totalIngested))
	if grandTotalTriples > 0 {
		builder.WriteString(fmt.Sprintf("  Totals: %d triples | %d articles | %d chapters\n",
			grandTotalTriples, grandTotalArticles, grandTotalChapters))
	}

	return builder.String()
}

// formatDuration formats a duration in a human-readable compact form.
func formatDuration(duration time.Duration) string {
	if duration < time.Second {
		return fmt.Sprintf("%dms", duration.Milliseconds())
	}
	if duration < time.Minute {
		return fmt.Sprintf("%.1fs", duration.Seconds())
	}
	minutes := int(duration.Minutes())
	seconds := int(duration.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}
