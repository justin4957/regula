package bulk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coolbeans/regula/pkg/library"
)

// BulkIngester reads downloaded files, parses XML/text content, and adds
// documents to the library via library.AddDocument.
type BulkIngester struct {
	config     IngestConfig
	lib        *library.Library
	downloader *Downloader
}

// NewBulkIngester creates a BulkIngester with the given config.
func NewBulkIngester(config IngestConfig, lib *library.Library) *BulkIngester {
	return &BulkIngester{
		config: config,
		lib:    lib,
	}
}

// IngestSource processes all downloaded datasets from a specific source.
func (ingester *BulkIngester) IngestSource(sourceName string, downloadDir string) (*IngestReport, error) {
	manifest, err := LoadManifest(filepath.Join(downloadDir, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to load download manifest: %w", err)
	}

	report := &IngestReport{}

	for _, record := range manifest.Downloads {
		if record.SourceName != sourceName {
			continue
		}

		// Apply title filter
		if len(ingester.config.TitleFilter) > 0 && !matchesTitleFilter(record.Identifier, ingester.config.TitleFilter) {
			continue
		}

		report.TotalAttempted++

		entry := ingester.ingestDownloadedFile(record)
		report.Entries = append(report.Entries, entry)

		switch entry.Status {
		case "ingested":
			report.Succeeded++
		case "skipped":
			report.Skipped++
		case "failed":
			report.Failed++
		}
	}

	return report, nil
}

// IngestAll processes all downloaded datasets from all sources.
func (ingester *BulkIngester) IngestAll(downloadDir string) (*IngestReport, error) {
	manifest, err := LoadManifest(filepath.Join(downloadDir, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to load download manifest: %w", err)
	}

	report := &IngestReport{}

	for _, record := range manifest.Downloads {
		report.TotalAttempted++

		entry := ingester.ingestDownloadedFile(record)
		report.Entries = append(report.Entries, entry)

		switch entry.Status {
		case "ingested":
			report.Succeeded++
		case "skipped":
			report.Skipped++
		case "failed":
			report.Failed++
		}
	}

	return report, nil
}

// ingestDownloadedFile processes a single downloaded file based on its source.
func (ingester *BulkIngester) ingestDownloadedFile(record *DownloadRecord) IngestEntry {
	documentID := deriveDocumentID(record)

	// Check if already ingested
	if !ingester.config.Force {
		existingDocs := ingester.lib.ListDocuments()
		for _, doc := range existingDocs {
			if doc.ID == documentID {
				return IngestEntry{
					Identifier: record.Identifier,
					DocumentID: documentID,
					Status:     "skipped",
				}
			}
		}
	}

	if ingester.config.DryRun {
		return IngestEntry{
			Identifier: record.Identifier,
			DocumentID: documentID,
			Status:     "skipped",
			Error:      "dry run",
		}
	}

	// Route to source-specific ingestion
	var plaintext string
	var ingestErr error

	switch record.SourceName {
	case "uscode":
		plaintext, ingestErr = ingester.ingestUSCode(record)
	case "cfr":
		plaintext, ingestErr = ingester.ingestCFR(record)
	case "california":
		plaintext, ingestErr = ingester.ingestCalifornia(record)
	case "archive":
		plaintext, ingestErr = ingester.ingestArchive(record)
	default:
		ingestErr = fmt.Errorf("unknown source: %s", record.SourceName)
	}

	if ingestErr != nil {
		return IngestEntry{
			Identifier: record.Identifier,
			DocumentID: documentID,
			Status:     "failed",
			Error:      ingestErr.Error(),
		}
	}

	if plaintext == "" {
		return IngestEntry{
			Identifier: record.Identifier,
			DocumentID: documentID,
			Status:     "failed",
			Error:      "no content extracted",
		}
	}

	// Add to library
	addOptions := deriveAddOptions(record, documentID)
	docEntry, err := ingester.lib.AddDocument(documentID, []byte(plaintext), addOptions)
	if err != nil {
		return IngestEntry{
			Identifier: record.Identifier,
			DocumentID: documentID,
			Status:     "failed",
			Error:      err.Error(),
		}
	}

	triples := 0
	if docEntry.Stats != nil {
		triples = docEntry.Stats.TotalTriples
	}

	return IngestEntry{
		Identifier: record.Identifier,
		DocumentID: documentID,
		Status:     "ingested",
		Triples:    triples,
	}
}

// ingestUSCode extracts plaintext from a downloaded USC ZIP.
func (ingester *BulkIngester) ingestUSCode(record *DownloadRecord) (string, error) {
	localPath := record.LocalPath
	if !strings.HasSuffix(localPath, ".zip") {
		return "", fmt.Errorf("expected ZIP file, got: %s", localPath)
	}

	// Extract ZIP to a temp directory alongside the ZIP
	extractDir := strings.TrimSuffix(localPath, ".zip")
	downloader, err := NewDownloader(DefaultDownloadConfig())
	if err != nil {
		return "", err
	}

	extractedFiles, err := downloader.ExtractZIP(localPath, extractDir)
	if err != nil {
		return "", fmt.Errorf("failed to extract ZIP: %w", err)
	}

	// Find the XML file
	var xmlPath string
	for _, extractedFile := range extractedFiles {
		if strings.HasSuffix(extractedFile, ".xml") {
			xmlPath = extractedFile
			break
		}
	}

	if xmlPath == "" {
		return "", fmt.Errorf("no XML file found in ZIP")
	}

	xmlFile, err := os.Open(xmlPath)
	if err != nil {
		return "", fmt.Errorf("failed to open XML: %w", err)
	}
	defer xmlFile.Close()

	document, err := ParseUSLMXML(xmlFile)
	if err != nil {
		return "", fmt.Errorf("failed to parse USLM XML: %w", err)
	}

	return USLMToPlaintext(document), nil
}

// ingestCFR extracts plaintext from a downloaded CFR ZIP.
func (ingester *BulkIngester) ingestCFR(record *DownloadRecord) (string, error) {
	localPath := record.LocalPath
	if !strings.HasSuffix(localPath, ".zip") {
		return "", fmt.Errorf("expected ZIP file, got: %s", localPath)
	}

	extractDir := strings.TrimSuffix(localPath, ".zip")
	downloader, err := NewDownloader(DefaultDownloadConfig())
	if err != nil {
		return "", err
	}

	extractedFiles, err := downloader.ExtractZIP(localPath, extractDir)
	if err != nil {
		return "", fmt.Errorf("failed to extract ZIP: %w", err)
	}

	// Concatenate all XML volume files
	var allText strings.Builder

	for _, extractedFile := range extractedFiles {
		if !strings.HasSuffix(extractedFile, ".xml") {
			continue
		}

		xmlFile, err := os.Open(extractedFile)
		if err != nil {
			continue
		}

		document, err := ParseCFRXML(xmlFile)
		xmlFile.Close()
		if err != nil {
			// Try reading as raw text fallback
			rawBytes, readErr := os.ReadFile(extractedFile)
			if readErr == nil {
				allText.WriteString(string(rawBytes))
			}
			continue
		}

		allText.WriteString(CFRToPlaintext(document))
	}

	return allText.String(), nil
}

// ingestCalifornia reads a downloaded California code text file.
func (ingester *BulkIngester) ingestCalifornia(record *DownloadRecord) (string, error) {
	data, err := os.ReadFile(record.LocalPath)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", record.LocalPath, err)
	}
	return string(data), nil
}

// ingestArchive extracts and reads an Internet Archive download.
func (ingester *BulkIngester) ingestArchive(record *DownloadRecord) (string, error) {
	localPath := record.LocalPath

	switch {
	case strings.HasSuffix(localPath, ".tar.gz") || strings.HasSuffix(localPath, ".tgz"):
		extractDir := strings.TrimSuffix(strings.TrimSuffix(localPath, ".gz"), ".tar")
		if strings.HasSuffix(localPath, ".tgz") {
			extractDir = strings.TrimSuffix(localPath, ".tgz")
		}

		downloader, err := NewDownloader(DefaultDownloadConfig())
		if err != nil {
			return "", err
		}

		extractedFiles, err := downloader.ExtractTarGZ(localPath, extractDir)
		if err != nil {
			return "", fmt.Errorf("failed to extract tar.gz: %w", err)
		}

		return concatenateTextFiles(extractedFiles), nil

	case strings.HasSuffix(localPath, ".zip"):
		extractDir := strings.TrimSuffix(localPath, ".zip")
		downloader, err := NewDownloader(DefaultDownloadConfig())
		if err != nil {
			return "", err
		}

		extractedFiles, err := downloader.ExtractZIP(localPath, extractDir)
		if err != nil {
			return "", fmt.Errorf("failed to extract ZIP: %w", err)
		}

		return concatenateTextFiles(extractedFiles), nil

	default:
		// Try reading as plain text
		data, err := os.ReadFile(localPath)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}

// concatenateTextFiles reads and concatenates all text/XML files from a list.
func concatenateTextFiles(filePaths []string) string {
	var builder strings.Builder

	for _, filePath := range filePaths {
		ext := strings.ToLower(filepath.Ext(filePath))
		if ext != ".txt" && ext != ".xml" && ext != ".html" && ext != ".htm" {
			continue
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		builder.WriteString(string(data))
		builder.WriteString("\n\n")
	}

	return builder.String()
}

// deriveDocumentID generates a library document ID from a download record.
func deriveDocumentID(record *DownloadRecord) string {
	switch record.SourceName {
	case "uscode":
		// "usc-title-42" → "us-uscode-title-42"
		return "us-" + record.Identifier
	case "cfr":
		// "cfr-2024-title-42" → "us-cfr-2024-title-42"
		return "us-" + record.Identifier
	case "california":
		// "ca-civ" → "us-ca-civ"
		return "us-" + record.Identifier
	case "archive":
		// "govlawca" → "archive-govlawca"
		return "archive-" + record.Identifier
	default:
		return record.Identifier
	}
}

// deriveAddOptions generates library.AddOptions from a download record.
func deriveAddOptions(record *DownloadRecord, documentID string) library.AddOptions {
	jurisdiction := "US"
	format := "us"

	switch record.SourceName {
	case "california":
		jurisdiction = "US-CA"
	case "archive":
		format = "generic"
	}

	return library.AddOptions{
		Name:         documentID,
		ShortName:    documentID,
		FullName:     record.Identifier,
		Jurisdiction: jurisdiction,
		Format:       format,
		SourceInfo:   fmt.Sprintf("bulk download from %s: %s", record.SourceName, record.URL),
		Force:        false,
	}
}

// matchesTitleFilter checks if an identifier matches any title in the filter.
func matchesTitleFilter(identifier string, titleFilter []string) bool {
	identifierLower := strings.ToLower(identifier)
	for _, filterTitle := range titleFilter {
		filterLower := strings.ToLower(strings.TrimSpace(filterTitle))
		if strings.Contains(identifierLower, filterLower) {
			return true
		}
	}
	return false
}
