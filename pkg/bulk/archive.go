package bulk

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// InternetArchiveSource downloads state code data from the Internet Archive
// govlaw collection (Public.Resource.Org / Fastcase data).
type InternetArchiveSource struct {
	config     DownloadConfig
	httpClient *http.Client
}

// NewInternetArchiveSource creates an InternetArchiveSource.
func NewInternetArchiveSource(config DownloadConfig) *InternetArchiveSource {
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: config.Timeout}
	}
	return &InternetArchiveSource{config: config, httpClient: httpClient}
}

func (source *InternetArchiveSource) Name() string { return "archive" }

func (source *InternetArchiveSource) Description() string {
	return "State code archives from Internet Archive govlaw collection (Public.Resource.Org)"
}

// ListDatasets queries the Internet Archive search API to discover available
// items in the govlaw collection.
func (source *InternetArchiveSource) ListDatasets() ([]Dataset, error) {
	searchURL := "https://archive.org/advancedsearch.php?q=collection:govlaw&output=json&rows=100&fl[]=identifier,title,description"

	request, err := http.NewRequest(http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}
	request.Header.Set("User-Agent", source.config.UserAgent)

	response, err := source.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to search Internet Archive: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d from Internet Archive search", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read search response: %w", err)
	}

	var searchResult iaSearchResult
	if err := json.Unmarshal(body, &searchResult); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	var datasets []Dataset
	for _, doc := range searchResult.Response.Docs {
		jurisdiction := extractJurisdictionFromID(doc.Identifier)

		datasets = append(datasets, Dataset{
			SourceName:   "archive",
			Identifier:   doc.Identifier,
			DisplayName:  doc.Title,
			URL:          fmt.Sprintf("https://archive.org/download/%s", doc.Identifier),
			Format:       "tar.gz",
			Jurisdiction: jurisdiction,
		})
	}

	return datasets, nil
}

// DownloadDataset downloads an Internet Archive item.
// First fetches the item metadata to find downloadable files, then
// downloads the first suitable archive file (tar.gz, zip, or xml).
func (source *InternetArchiveSource) DownloadDataset(dataset Dataset, downloader *Downloader) (*DownloadResult, error) {
	sourceDir := downloader.SourceDirectory("archive")

	// Fetch item metadata to find downloadable files
	metadataURL := fmt.Sprintf("https://archive.org/metadata/%s", dataset.Identifier)

	downloader.waitForDomain("archive.org")

	metaRequest, err := http.NewRequest(http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata request: %w", err)
	}
	metaRequest.Header.Set("User-Agent", downloader.config.UserAgent)

	metaResponse, err := source.httpClient.Do(metaRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata for %s: %w", dataset.Identifier, err)
	}
	defer metaResponse.Body.Close()

	body, err := io.ReadAll(metaResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var itemMetadata iaItemMetadata
	if err := json.Unmarshal(body, &itemMetadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata for %s: %w", dataset.Identifier, err)
	}

	// Find the best downloadable file
	downloadFilename := findBestArchiveFile(itemMetadata.Files)
	if downloadFilename == "" {
		return nil, fmt.Errorf("no suitable archive file found for %s", dataset.Identifier)
	}

	downloadURL := fmt.Sprintf("https://archive.org/download/%s/%s",
		dataset.Identifier, downloadFilename)
	localPath := filepath.Join(sourceDir, dataset.Identifier, downloadFilename)

	bytesWritten, skipped, err := downloader.DownloadFile(
		downloadURL, localPath, PrintDownloadProgress)
	if err != nil {
		return &DownloadResult{
			Dataset: dataset,
			Error:   err.Error(),
		}, err
	}

	if !skipped {
		downloader.Manifest().RecordDownload(&DownloadRecord{
			Identifier:   dataset.Identifier,
			SourceName:   "archive",
			URL:          downloadURL,
			LocalPath:    localPath,
			SizeBytes:    bytesWritten,
			DownloadedAt: time.Now(),
		})
		downloader.SaveManifest()
		fmt.Println()
	}

	return &DownloadResult{
		Dataset:      dataset,
		LocalPath:    localPath,
		BytesWritten: bytesWritten,
		Skipped:      skipped,
		DownloadedAt: time.Now(),
	}, nil
}

// iaSearchResult represents the Internet Archive search API response.
type iaSearchResult struct {
	Response struct {
		NumFound int     `json:"numFound"`
		Docs     []iaDoc `json:"docs"`
	} `json:"response"`
}

// iaDoc represents a single document in search results.
type iaDoc struct {
	Identifier  string `json:"identifier"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// iaItemMetadata represents the Internet Archive item metadata response.
type iaItemMetadata struct {
	Files []iaFile `json:"files"`
}

// iaFile represents a file within an Internet Archive item.
type iaFile struct {
	Name   string `json:"name"`
	Format string `json:"format"`
	Size   string `json:"size"`
}

// findBestArchiveFile selects the best file to download from an IA item.
// Prefers tar.gz > zip > xml > txt files.
func findBestArchiveFile(files []iaFile) string {
	var tarGzFile, zipFile, xmlFile, txtFile string

	for _, file := range files {
		nameLower := strings.ToLower(file.Name)
		switch {
		case strings.HasSuffix(nameLower, ".tar.gz") || strings.HasSuffix(nameLower, ".tgz"):
			if tarGzFile == "" {
				tarGzFile = file.Name
			}
		case strings.HasSuffix(nameLower, ".zip"):
			if zipFile == "" {
				zipFile = file.Name
			}
		case strings.HasSuffix(nameLower, ".xml"):
			if xmlFile == "" {
				xmlFile = file.Name
			}
		case strings.HasSuffix(nameLower, ".txt"):
			if txtFile == "" {
				txtFile = file.Name
			}
		}
	}

	switch {
	case tarGzFile != "":
		return tarGzFile
	case zipFile != "":
		return zipFile
	case xmlFile != "":
		return xmlFile
	case txtFile != "":
		return txtFile
	default:
		return ""
	}
}

// extractJurisdictionFromID guesses a jurisdiction from an IA identifier.
// Examples: "govlawca" → "US-CA", "govlawga" → "US-GA"
func extractJurisdictionFromID(identifier string) string {
	identifier = strings.ToLower(identifier)

	// Try to extract state abbreviation from "govlaw{state}" pattern
	if strings.HasPrefix(identifier, "govlaw") {
		remainder := strings.TrimPrefix(identifier, "govlaw")
		// Take first 2 alpha characters as state code
		var stateCode []byte
		for _, ch := range []byte(remainder) {
			if ch >= 'a' && ch <= 'z' && len(stateCode) < 2 {
				stateCode = append(stateCode, ch)
			} else {
				break
			}
		}
		if len(stateCode) == 2 {
			return "US-" + strings.ToUpper(string(stateCode))
		}
	}

	// Check for "gov.{state}" pattern
	if strings.HasPrefix(identifier, "gov.") {
		parts := strings.SplitN(identifier, ".", 3)
		if len(parts) >= 2 && len(parts[1]) == 2 {
			return "US-" + strings.ToUpper(parts[1])
		}
	}

	return "US"
}
