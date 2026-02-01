package bulk

import (
	"fmt"
	"net/http"
	"time"
)

// Source represents a bulk legislation data source capable of listing
// available datasets and downloading them.
type Source interface {
	// Name returns the short identifier of this source (e.g., "uscode").
	Name() string

	// Description returns a brief human-readable description of the source.
	Description() string

	// ListDatasets returns all datasets available from this source.
	ListDatasets() ([]Dataset, error)

	// DownloadDataset downloads a single dataset to the target directory.
	// Returns the local file path of the downloaded file.
	// Supports resumability: skips if file already exists with matching size.
	DownloadDataset(dataset Dataset, downloader *Downloader) (*DownloadResult, error)
}

// Dataset represents a single downloadable unit from a bulk source.
type Dataset struct {
	// SourceName identifies which Source this dataset belongs to.
	SourceName string `json:"source_name"`

	// Identifier is a unique key within the source (e.g., "usc-title-42").
	Identifier string `json:"identifier"`

	// DisplayName is a human-readable name.
	DisplayName string `json:"display_name"`

	// URL is the download URL.
	URL string `json:"url"`

	// Format describes the archive format ("zip", "tar.gz", "xml", "txt").
	Format string `json:"format"`

	// Jurisdiction is the jurisdiction code for library integration.
	Jurisdiction string `json:"jurisdiction"`

	// ExpectedSizeBytes is the expected file size, if known (0 if unknown).
	ExpectedSizeBytes int64 `json:"expected_size_bytes,omitempty"`
}

// DownloadResult captures the outcome of downloading a single dataset.
type DownloadResult struct {
	Dataset      Dataset   `json:"dataset"`
	LocalPath    string    `json:"local_path"`
	BytesWritten int64     `json:"bytes_written"`
	Skipped      bool      `json:"skipped"`
	DownloadedAt time.Time `json:"downloaded_at"`
	Error        string    `json:"error,omitempty"`
}

// ProgressCallback is called during download with bytes transferred so far.
type ProgressCallback func(bytesDownloaded int64, totalBytes int64)

// DownloadConfig holds configuration for the bulk download engine.
type DownloadConfig struct {
	// DownloadDirectory is the root directory for storing downloaded files.
	DownloadDirectory string

	// RateLimit is the minimum interval between HTTP requests per domain.
	RateLimit time.Duration

	// Timeout is the per-request HTTP timeout.
	Timeout time.Duration

	// UserAgent is the User-Agent header sent with requests.
	UserAgent string

	// HTTPClient allows injection of a custom HTTP client (for testing).
	HTTPClient *http.Client

	// DryRun when true, lists what would be downloaded without fetching.
	DryRun bool

	// TitleFilter limits downloads to specific title identifiers (empty = all).
	TitleFilter []string

	// CFRYear specifies the CFR edition year (default "2024").
	CFRYear string

	// MaxRetries is the maximum number of retry attempts for transient errors.
	MaxRetries int

	// RetryBaseDelay is the initial delay between retries (doubles each attempt).
	RetryBaseDelay time.Duration
}

// DefaultDownloadConfig returns a DownloadConfig with sensible defaults.
func DefaultDownloadConfig() DownloadConfig {
	return DownloadConfig{
		DownloadDirectory: ".regula/downloads",
		RateLimit:         3 * time.Second,
		Timeout:           5 * time.Minute,
		UserAgent:         "regula-bulk/1.0 (+https://regula.dev)",
		CFRYear:           "2024",
		MaxRetries:        3,
		RetryBaseDelay:    5 * time.Second,
	}
}

// IngestConfig holds configuration for the bulk ingest phase.
type IngestConfig struct {
	// LibraryPath is the path to the .regula library directory.
	LibraryPath string

	// DownloadDirectory is the root directory where downloads are stored.
	DownloadDirectory string

	// SourceFilter limits ingestion to a specific source name.
	SourceFilter string

	// TitleFilter limits ingestion to specific titles.
	TitleFilter []string

	// Force overwrites existing library documents.
	Force bool

	// DryRun lists what would be ingested without performing ingestion.
	DryRun bool

	// BaseURI is the base URI for the library.
	BaseURI string
}

// IngestReport summarizes the results of a bulk ingest operation.
type IngestReport struct {
	TotalAttempted   int           `json:"total_attempted"`
	Succeeded        int           `json:"succeeded"`
	Skipped          int           `json:"skipped"`
	Failed           int           `json:"failed"`
	TotalTriples     int           `json:"total_triples"`
	TotalArticles    int           `json:"total_articles"`
	TotalChapters    int           `json:"total_chapters"`
	TotalDefinitions int           `json:"total_definitions"`
	TotalReferences  int           `json:"total_references"`
	TotalRights      int           `json:"total_rights"`
	TotalObligations int           `json:"total_obligations"`
	Entries          []IngestEntry `json:"entries"`
}

// IngestEntry records the outcome of ingesting a single document.
type IngestEntry struct {
	Identifier  string        `json:"identifier"`
	DocumentID  string        `json:"document_id"`
	Status      string        `json:"status"` // "ingested", "skipped", "failed"
	Error       string        `json:"error,omitempty"`
	Triples     int           `json:"triples,omitempty"`
	Articles    int           `json:"articles,omitempty"`
	Chapters    int           `json:"chapters,omitempty"`
	Sections    int           `json:"sections,omitempty"`
	Definitions int           `json:"definitions,omitempty"`
	References  int           `json:"references,omitempty"`
	Rights      int           `json:"rights,omitempty"`
	Obligations int           `json:"obligations,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
	SourceBytes int           `json:"source_bytes,omitempty"`
}

// ResolveSource creates a Source instance from a source name string.
func ResolveSource(sourceName string, config DownloadConfig) (Source, error) {
	switch sourceName {
	case "uscode":
		return NewUSCodeSource(config), nil
	case "cfr":
		return NewCFRSource(config), nil
	case "california":
		return NewCaliforniaSource(config), nil
	case "archive":
		return NewInternetArchiveSource(config), nil
	default:
		return nil, fmt.Errorf("unknown source: %s (available: uscode, cfr, california, archive)", sourceName)
	}
}

// AllSourceNames returns the list of registered source names.
func AllSourceNames() []string {
	return []string{"uscode", "cfr", "california", "archive"}
}
