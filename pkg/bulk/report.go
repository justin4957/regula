package bulk

import (
	"encoding/json"
	"fmt"
	"strings"
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
	builder.WriteString(strings.Repeat("═", 60) + "\n")
	builder.WriteString(fmt.Sprintf("Attempted: %d | Succeeded: %d | Skipped: %d | Failed: %d\n",
		report.TotalAttempted, report.Succeeded, report.Skipped, report.Failed))
	builder.WriteString(strings.Repeat("─", 60) + "\n")

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

		line := fmt.Sprintf("  %-8s %-30s", status, entry.DocumentID)
		if entry.Triples > 0 {
			line += fmt.Sprintf(" (%d triples)", entry.Triples)
		}
		if entry.Error != "" {
			line += fmt.Sprintf(" error: %s", entry.Error)
		}
		builder.WriteString(line + "\n")
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

// FormatStatusReport formats download and ingest status for terminal output.
func FormatStatusReport(manifest *DownloadManifest, sourceFilter string) string {
	var builder strings.Builder

	builder.WriteString("\nBulk Download Status\n")
	builder.WriteString(strings.Repeat("═", 70) + "\n")

	sources := AllSourceNames()
	if sourceFilter != "" {
		sources = []string{sourceFilter}
	}

	for _, sourceName := range sources {
		count := manifest.CountBySource(sourceName)
		totalSize := manifest.TotalSizeBySource(sourceName)

		builder.WriteString(fmt.Sprintf("\n  %-12s  %3d downloads  %s\n",
			sourceName, count, FormatBytes(totalSize)))

		// List individual downloads for this source
		for _, record := range manifest.Downloads {
			if record.SourceName == sourceName {
				builder.WriteString(fmt.Sprintf("    %-30s  %s  %s\n",
					record.Identifier,
					FormatBytes(record.SizeBytes),
					record.DownloadedAt.Format("2006-01-02 15:04")))
			}
		}
	}

	totalDownloads := len(manifest.Downloads)
	builder.WriteString(fmt.Sprintf("\nTotal: %d downloads\n", totalDownloads))

	return builder.String()
}
