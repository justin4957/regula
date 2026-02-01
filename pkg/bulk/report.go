package bulk

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
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
	Triples     int
	Articles    int
	Chapters    int
	Definitions int
	References  int
	Rights      int
	Obligations int
	Status      string
	DisplayName string
	Source      string
	IngestedAt  time.Time
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

// CollectStats builds a StatsReport from a download manifest and per-document statistics.
// It cross-references each download record with its corresponding library stats entry.
func CollectStats(manifest *DownloadManifest, documentStats map[string]*DocumentStatsSummary) *StatsReport {
	report := &StatsReport{}

	// Collect entries sorted by identifier for deterministic output
	var identifiers []string
	for identifier := range manifest.Downloads {
		identifiers = append(identifiers, identifier)
	}
	sort.Strings(identifiers)

	report.TitlesTotal = len(identifiers)

	for _, identifier := range identifiers {
		record := manifest.Downloads[identifier]
		documentID := deriveDocumentID(record)

		entry := StatsEntry{
			Identifier: record.Identifier,
			DocumentID: documentID,
			Source:     record.SourceName,
			Status:     "pending",
		}

		if documentStats != nil {
			if stats, exists := documentStats[documentID]; exists {
				entry.Triples = stats.Triples
				entry.Articles = stats.Articles
				entry.Chapters = stats.Chapters
				entry.Definitions = stats.Definitions
				entry.References = stats.References
				entry.Rights = stats.Rights
				entry.Obligations = stats.Obligations
				entry.Status = stats.Status
				entry.DisplayName = stats.DisplayName
				entry.Source = stats.Source
				entry.IngestedAt = stats.IngestedAt

				if stats.Triples > 0 {
					report.TitlesIngested++
					report.TotalTriples += stats.Triples
					report.TotalArticles += stats.Articles
					report.TotalChapters += stats.Chapters
					report.TotalDefinitions += stats.Definitions
					report.TotalReferences += stats.References
					report.TotalRights += stats.Rights
					report.TotalObligations += stats.Obligations
				}

				// Use source from record if not set via documentStats
				if entry.Source == "" {
					entry.Source = record.SourceName
				}
			}
		}

		// Fallback source from download record
		if entry.Source == "" {
			entry.Source = record.SourceName
		}

		report.Entries = append(report.Entries, entry)
	}

	return report
}

// FormatStatsTable formats a StatsReport as an ASCII table for terminal display.
func FormatStatsTable(report *StatsReport) string {
	var builder strings.Builder

	builder.WriteString("\nBulk Ingestion Statistics\n")
	builder.WriteString(strings.Repeat("═", 110) + "\n")
	builder.WriteString(fmt.Sprintf("Titles Ingested: %d/%d\n", report.TitlesIngested, report.TitlesTotal))
	builder.WriteString(strings.Repeat("─", 110) + "\n")

	// Column headers
	builder.WriteString(fmt.Sprintf("  %-28s %10s %10s %8s %8s %8s %8s %8s  %-8s\n",
		"TITLE", "TRIPLES", "ARTICLES", "CHPTRS", "DEFS", "XREFS", "RIGHTS", "OBLIGS", "STATUS"))
	builder.WriteString("  " + strings.Repeat("─", 106) + "\n")

	for _, entry := range report.Entries {
		titleLabel := entry.Identifier
		if entry.DisplayName != "" {
			titleLabel = entry.DisplayName
		}
		if len(titleLabel) > 28 {
			titleLabel = titleLabel[:25] + "..."
		}

		if entry.Status == "pending" || entry.Triples == 0 {
			builder.WriteString(fmt.Sprintf("  %-28s %10s %10s %8s %8s %8s %8s %8s  %-8s\n",
				titleLabel, "-", "-", "-", "-", "-", "-", "-", entry.Status))
		} else {
			builder.WriteString(fmt.Sprintf("  %-28s %10s %10s %8s %8s %8s %8s %8s  %-8s\n",
				titleLabel,
				formatNumber(entry.Triples),
				formatNumber(entry.Articles),
				formatNumber(entry.Chapters),
				formatNumber(entry.Definitions),
				formatNumber(entry.References),
				formatNumber(entry.Rights),
				formatNumber(entry.Obligations),
				entry.Status))
		}
	}

	// Totals row
	builder.WriteString("  " + strings.Repeat("─", 106) + "\n")
	builder.WriteString(fmt.Sprintf("  %-28s %10s %10s %8s %8s %8s %8s %8s\n",
		"TOTALS",
		formatNumber(report.TotalTriples),
		formatNumber(report.TotalArticles),
		formatNumber(report.TotalChapters),
		formatNumber(report.TotalDefinitions),
		formatNumber(report.TotalReferences),
		formatNumber(report.TotalRights),
		formatNumber(report.TotalObligations)))

	return builder.String()
}

// FormatStatsJSON formats a StatsReport as indented JSON.
func FormatStatsJSON(report *StatsReport) string {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	return string(data)
}

// FormatStatsCSV formats a StatsReport as CSV with a header row and per-entry data rows.
func FormatStatsCSV(report *StatsReport) string {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	// Header row
	header := []string{
		"identifier", "document_id", "display_name", "source",
		"triples", "articles", "chapters", "definitions",
		"references", "rights", "obligations", "status", "ingested_at",
	}
	_ = writer.Write(header)

	// Data rows
	for _, entry := range report.Entries {
		ingestedAt := ""
		if !entry.IngestedAt.IsZero() {
			ingestedAt = entry.IngestedAt.Format(time.RFC3339)
		}

		row := []string{
			entry.Identifier,
			entry.DocumentID,
			entry.DisplayName,
			entry.Source,
			fmt.Sprintf("%d", entry.Triples),
			fmt.Sprintf("%d", entry.Articles),
			fmt.Sprintf("%d", entry.Chapters),
			fmt.Sprintf("%d", entry.Definitions),
			fmt.Sprintf("%d", entry.References),
			fmt.Sprintf("%d", entry.Rights),
			fmt.Sprintf("%d", entry.Obligations),
			entry.Status,
			ingestedAt,
		}
		_ = writer.Write(row)
	}

	writer.Flush()
	return buffer.String()
}

// formatNumber returns a comma-separated number string (e.g., 25100 → "25,100").
func formatNumber(value int) string {
	if value == 0 {
		return "0"
	}

	negative := value < 0
	if negative {
		value = -value
	}

	raw := fmt.Sprintf("%d", value)
	length := len(raw)

	var result strings.Builder
	if negative {
		result.WriteByte('-')
	}

	for index, digit := range raw {
		if index > 0 && (length-index)%3 == 0 {
			result.WriteByte(',')
		}
		result.WriteRune(digit)
	}

	return result.String()
}
