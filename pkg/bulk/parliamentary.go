package bulk

import (
	"fmt"
	"path/filepath"
	"time"
)

// ParliamentarySource downloads Congressional rules documents from official sources.
// Includes House Rules, Senate Rules, Joint Rules, and committee-specific rules.
type ParliamentarySource struct {
	config DownloadConfig
}

// NewParliamentarySource creates a ParliamentarySource with the given config.
func NewParliamentarySource(config DownloadConfig) *ParliamentarySource {
	return &ParliamentarySource{config: config}
}

func (source *ParliamentarySource) Name() string { return "parliamentary" }

func (source *ParliamentarySource) Description() string {
	return "Congressional rules: House Rules, Senate Rules, Joint Rules"
}

// ListDatasets returns all available parliamentary rule documents.
func (source *ParliamentarySource) ListDatasets() ([]Dataset, error) {
	var datasets []Dataset

	for _, doc := range parliamentaryDocuments {
		datasets = append(datasets, Dataset{
			SourceName:   "parliamentary",
			Identifier:   doc.Identifier,
			DisplayName:  doc.DisplayName,
			URL:          doc.URL,
			Format:       doc.Format,
			Jurisdiction: "US-Federal",
		})
	}

	return datasets, nil
}

// DownloadDataset downloads a parliamentary rules document.
func (source *ParliamentarySource) DownloadDataset(dataset Dataset, downloader *Downloader) (*DownloadResult, error) {
	sourceDir := downloader.SourceDirectory("parliamentary")

	// Determine file extension from format
	ext := ".txt"
	if dataset.Format == "pdf" {
		ext = ".pdf"
	} else if dataset.Format == "htm" || dataset.Format == "html" {
		ext = ".html"
	}

	localPath := filepath.Join(sourceDir, dataset.Identifier+ext)

	bytesWritten, skipped, err := downloader.DownloadFile(
		dataset.URL, localPath, PrintDownloadProgress)
	if err != nil {
		return &DownloadResult{
			Dataset: dataset,
			Error:   err.Error(),
		}, err
	}

	// Record in manifest
	if !skipped {
		downloader.Manifest().RecordDownload(&DownloadRecord{
			Identifier:   dataset.Identifier,
			SourceName:   "parliamentary",
			URL:          dataset.URL,
			LocalPath:    localPath,
			SizeBytes:    bytesWritten,
			DownloadedAt: time.Now(),
		})
		downloader.SaveManifest()
		fmt.Println() // newline after progress bar
	}

	return &DownloadResult{
		Dataset:      dataset,
		LocalPath:    localPath,
		BytesWritten: bytesWritten,
		Skipped:      skipped,
		DownloadedAt: time.Now(),
	}, nil
}

// ParliamentaryDocument describes a congressional rules document.
type ParliamentaryDocument struct {
	Identifier  string
	DisplayName string
	URL         string
	Format      string
	Congress    string
	Chamber     string // "house", "senate", "joint"
}

// parliamentaryDocuments contains the available parliamentary rules.
// URLs are from official congressional sources (GPO, rules.house.gov, rules.senate.gov).
var parliamentaryDocuments = []ParliamentaryDocument{
	// House Rules - 119th Congress
	{
		Identifier:  "house-rules-119th",
		DisplayName: "House Rules (119th Congress)",
		URL:         "https://rules.house.gov/sites/republicans.rules118.house.gov/files/documents/Rules-and-Manual-119th.pdf",
		Format:      "pdf",
		Congress:    "119",
		Chamber:     "house",
	},
	// Senate Standing Rules
	{
		Identifier:  "senate-rules",
		DisplayName: "Senate Standing Rules",
		URL:         "https://www.rules.senate.gov/rules-of-the-senate",
		Format:      "htm",
		Congress:    "current",
		Chamber:     "senate",
	},
	// Joint Rules of Congress
	{
		Identifier:  "joint-rules",
		DisplayName: "Joint Rules of Congress",
		URL:         "https://www.govinfo.gov/content/pkg/HMAN-118/pdf/HMAN-118-pg1095.pdf",
		Format:      "pdf",
		Congress:    "current",
		Chamber:     "joint",
	},
	// House Rules Committee - Procedures
	{
		Identifier:  "house-rules-committee-procedures",
		DisplayName: "House Rules Committee Procedures",
		URL:         "https://rules.house.gov/about",
		Format:      "htm",
		Congress:    "current",
		Chamber:     "house",
	},
	// Senate Precedents (Riddick's)
	{
		Identifier:  "senate-precedents-riddick",
		DisplayName: "Riddick's Senate Procedure",
		URL:         "https://www.govinfo.gov/content/pkg/GPO-RIDDICK-1992/pdf/GPO-RIDDICK-1992.pdf",
		Format:      "pdf",
		Congress:    "1992",
		Chamber:     "senate",
	},
}

// ParliamentaryStats holds statistics for a parsed parliamentary document.
type ParliamentaryStats struct {
	Identifier         string
	DisplayName        string
	Chamber            string
	Rules              int
	Clauses            int
	CrossReferences    int
	ExternalReferences int
	Triples            int
}

// ParliamentarySummary aggregates stats across multiple parliamentary documents.
type ParliamentarySummary struct {
	Documents          int
	TotalRules         int
	TotalClauses       int
	TotalCrossRefs     int
	TotalExternalRefs  int
	TotalTriples       int
	CrossDocumentRefs  int
	DocumentStats      []ParliamentaryStats
}

// FormatParliamentarySummary formats the summary for display.
func FormatParliamentarySummary(summary *ParliamentarySummary) string {
	var result string

	result += fmt.Sprintf("Parliamentary Rules Summary\n")
	result += fmt.Sprintf("===========================\n\n")

	result += fmt.Sprintf("Documents ingested: %d\n", summary.Documents)
	result += fmt.Sprintf("Total rules: %d\n", summary.TotalRules)
	result += fmt.Sprintf("Total clauses: %d\n", summary.TotalClauses)
	result += fmt.Sprintf("Total cross-references: %d\n", summary.TotalCrossRefs)
	result += fmt.Sprintf("Total external references: %d\n", summary.TotalExternalRefs)
	result += fmt.Sprintf("Cross-document references: %d\n", summary.CrossDocumentRefs)
	result += fmt.Sprintf("Combined graph: %d triples\n\n", summary.TotalTriples)

	if len(summary.DocumentStats) > 0 {
		result += fmt.Sprintf("%-40s %8s %8s %8s\n", "Document", "Rules", "Clauses", "Refs")
		result += fmt.Sprintf("%-40s %8s %8s %8s\n", "--------", "-----", "-------", "----")
		for _, doc := range summary.DocumentStats {
			result += fmt.Sprintf("%-40s %8d %8d %8d\n",
				truncateString(doc.DisplayName, 40),
				doc.Rules,
				doc.Clauses,
				doc.CrossReferences)
		}
	}

	return result
}

// truncateString truncates a string to the specified length with ellipsis.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// GetParliamentaryDocumentsByChamber returns documents filtered by chamber.
func GetParliamentaryDocumentsByChamber(chamber string) []ParliamentaryDocument {
	var result []ParliamentaryDocument
	for _, doc := range parliamentaryDocuments {
		if doc.Chamber == chamber {
			result = append(result, doc)
		}
	}
	return result
}

// GetParliamentaryDocumentByID returns a document by its identifier.
func GetParliamentaryDocumentByID(identifier string) *ParliamentaryDocument {
	for _, doc := range parliamentaryDocuments {
		if doc.Identifier == identifier {
			return &doc
		}
	}
	return nil
}
