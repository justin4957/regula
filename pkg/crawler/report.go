package crawler

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// CrawlReport contains the results and statistics of a completed crawl.
type CrawlReport struct {
	// DryRun indicates this was a planning-only run.
	DryRun bool `json:"dry_run"`

	// TotalDiscovered is the number of unique citations/documents discovered.
	TotalDiscovered int `json:"total_discovered"`

	// TotalIngested is the number of documents successfully ingested.
	TotalIngested int `json:"total_ingested"`

	// TotalFailed is the number of documents that failed to fetch or ingest.
	TotalFailed int `json:"total_failed"`

	// TotalSkipped is the number of documents skipped (already in library, etc.).
	TotalSkipped int `json:"total_skipped"`

	// MaxDepthReached is the deepest BFS depth reached.
	MaxDepthReached int `json:"max_depth_reached"`

	// DepthStats contains per-depth level statistics.
	DepthStats map[int]*CrawlDepthStats `json:"depth_stats"`

	// DomainStats contains per-domain statistics.
	DomainStats map[string]*CrawlDomainStats `json:"domain_stats"`

	// Items contains the individual crawl item results.
	Items []*CrawlItem `json:"items"`

	// Seeds contains the original seeds that started the crawl.
	Seeds []CrawlSeed `json:"seeds"`
}

// CrawlDepthStats holds statistics for a specific BFS depth level.
type CrawlDepthStats struct {
	Depth    int `json:"depth"`
	Ingested int `json:"ingested"`
	Failed   int `json:"failed"`
	Skipped  int `json:"skipped"`
	Pending  int `json:"pending"`
}

// CrawlDomainStats holds statistics for a specific source domain.
type CrawlDomainStats struct {
	Domain   string `json:"domain"`
	Ingested int    `json:"ingested"`
	Failed   int    `json:"failed"`
	Skipped  int    `json:"skipped"`
}

// NewCrawlReport creates a new empty crawl report.
func NewCrawlReport(dryRun bool, seeds []CrawlSeed) *CrawlReport {
	return &CrawlReport{
		DryRun:      dryRun,
		DepthStats:  make(map[int]*CrawlDepthStats),
		DomainStats: make(map[string]*CrawlDomainStats),
		Items:       make([]*CrawlItem, 0),
		Seeds:       seeds,
	}
}

// RecordItem adds a crawl item result to the report.
func (report *CrawlReport) RecordItem(item *CrawlItem) {
	report.Items = append(report.Items, item)
	report.TotalDiscovered++

	// Update depth stats
	if _, hasDepth := report.DepthStats[item.Depth]; !hasDepth {
		report.DepthStats[item.Depth] = &CrawlDepthStats{Depth: item.Depth}
	}
	depthStats := report.DepthStats[item.Depth]

	// Update domain stats
	domain := item.Domain
	if domain == "" {
		domain = "unknown"
	}
	if _, hasDomain := report.DomainStats[domain]; !hasDomain {
		report.DomainStats[domain] = &CrawlDomainStats{Domain: domain}
	}
	domainStats := report.DomainStats[domain]

	switch item.Status {
	case CrawlItemIngested:
		report.TotalIngested++
		depthStats.Ingested++
		domainStats.Ingested++
	case CrawlItemFailed:
		report.TotalFailed++
		depthStats.Failed++
		domainStats.Failed++
	case CrawlItemSkipped:
		report.TotalSkipped++
		depthStats.Skipped++
		domainStats.Skipped++
	case CrawlItemPending:
		depthStats.Pending++
	}

	if item.Depth > report.MaxDepthReached {
		report.MaxDepthReached = item.Depth
	}
}

// Format returns the report in the specified format (table or json).
func (report *CrawlReport) Format(outputFormat string) string {
	switch strings.ToLower(outputFormat) {
	case "json":
		return report.formatJSON()
	default:
		return report.formatTable()
	}
}

// formatTable renders the report as a human-readable table.
func (report *CrawlReport) formatTable() string {
	var builder strings.Builder

	if report.DryRun {
		builder.WriteString("=== Crawl Plan (Dry Run) ===\n\n")
	} else {
		builder.WriteString("=== Crawl Report ===\n\n")
	}

	// Summary
	builder.WriteString("Summary:\n")
	builder.WriteString(fmt.Sprintf("  Discovered: %d\n", report.TotalDiscovered))
	builder.WriteString(fmt.Sprintf("  Ingested:   %d\n", report.TotalIngested))
	builder.WriteString(fmt.Sprintf("  Failed:     %d\n", report.TotalFailed))
	builder.WriteString(fmt.Sprintf("  Skipped:    %d\n", report.TotalSkipped))
	builder.WriteString(fmt.Sprintf("  Max Depth:  %d\n", report.MaxDepthReached))
	builder.WriteString("\n")

	// Depth breakdown
	if len(report.DepthStats) > 0 {
		builder.WriteString("By Depth:\n")
		builder.WriteString(fmt.Sprintf("  %-7s %-10s %-8s %-8s %-8s\n", "Depth", "Ingested", "Failed", "Skipped", "Pending"))
		builder.WriteString(fmt.Sprintf("  %-7s %-10s %-8s %-8s %-8s\n", "-----", "--------", "------", "-------", "-------"))

		depthKeys := make([]int, 0, len(report.DepthStats))
		for depth := range report.DepthStats {
			depthKeys = append(depthKeys, depth)
		}
		sort.Ints(depthKeys)

		for _, depth := range depthKeys {
			depthStat := report.DepthStats[depth]
			builder.WriteString(fmt.Sprintf("  %-7d %-10d %-8d %-8d %-8d\n",
				depthStat.Depth, depthStat.Ingested, depthStat.Failed, depthStat.Skipped, depthStat.Pending))
		}
		builder.WriteString("\n")
	}

	// Domain breakdown
	if len(report.DomainStats) > 0 {
		builder.WriteString("By Domain:\n")
		builder.WriteString(fmt.Sprintf("  %-35s %-10s %-8s %-8s\n", "Domain", "Ingested", "Failed", "Skipped"))
		builder.WriteString(fmt.Sprintf("  %-35s %-10s %-8s %-8s\n", "------", "--------", "------", "-------"))

		domainKeys := make([]string, 0, len(report.DomainStats))
		for domain := range report.DomainStats {
			domainKeys = append(domainKeys, domain)
		}
		sort.Strings(domainKeys)

		for _, domain := range domainKeys {
			domainStat := report.DomainStats[domain]
			builder.WriteString(fmt.Sprintf("  %-35s %-10d %-8d %-8d\n",
				domainStat.Domain, domainStat.Ingested, domainStat.Failed, domainStat.Skipped))
		}
		builder.WriteString("\n")
	}

	// Individual items
	if len(report.Items) > 0 {
		builder.WriteString("Documents:\n")
		builder.WriteString(fmt.Sprintf("  %-30s %-10s %-7s %-25s %s\n", "Document ID", "Status", "Depth", "Domain", "Citation"))
		builder.WriteString(fmt.Sprintf("  %-30s %-10s %-7s %-25s %s\n", "-----------", "------", "-----", "------", "--------"))

		for _, item := range report.Items {
			citation := item.Citation
			if len(citation) > 40 {
				citation = citation[:37] + "..."
			}
			builder.WriteString(fmt.Sprintf("  %-30s %-10s %-7d %-25s %s\n",
				truncateReportString(item.DocumentID, 30),
				item.Status,
				item.Depth,
				truncateReportString(item.Domain, 25),
				citation))
		}
	}

	return builder.String()
}

// formatJSON renders the report as JSON.
func (report *CrawlReport) formatJSON() string {
	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	return string(reportJSON)
}

// truncateReportString truncates a string to maxLen characters.
func truncateReportString(inputStr string, maxLen int) string {
	if len(inputStr) <= maxLen {
		return inputStr
	}
	if maxLen <= 3 {
		return inputStr[:maxLen]
	}
	return inputStr[:maxLen-3] + "..."
}
