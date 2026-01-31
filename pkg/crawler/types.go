// Package crawler provides a BFS tree-walking legislation crawler that
// automatically discovers and ingests US legislation by following cross-references.
package crawler

import (
	"time"
)

// Default configuration values for the crawler.
const (
	// DefaultCrawlMaxDepth is the default maximum BFS depth for crawling.
	DefaultCrawlMaxDepth = 3

	// DefaultCrawlMaxDocuments is the default maximum number of documents to ingest.
	DefaultCrawlMaxDocuments = 50

	// DefaultCrawlRateLimit is the default minimum interval between HTTP requests per domain.
	DefaultCrawlRateLimit = 3 * time.Second

	// DefaultCrawlTimeout is the default per-request HTTP timeout.
	DefaultCrawlTimeout = 30 * time.Second

	// DefaultCrawlUserAgent is the default User-Agent header.
	DefaultCrawlUserAgent = "regula-crawler/1.0 (+https://regula.dev)"

	// DefaultCrawlCacheTTL is the default cache TTL for fetched content.
	DefaultCrawlCacheTTL = 7 * 24 * time.Hour
)

// CrawlConfig holds configuration for the legislation crawler.
type CrawlConfig struct {
	// MaxDepth is the maximum BFS depth for following cross-references.
	MaxDepth int

	// MaxDocuments is the maximum total number of documents to crawl and ingest.
	MaxDocuments int

	// AllowedDomains restricts fetching to specific domains. Empty means all allowed.
	AllowedDomains []string

	// RateLimit is the minimum interval between HTTP requests per domain.
	RateLimit time.Duration

	// Timeout is the per-request HTTP timeout.
	Timeout time.Duration

	// LibraryPath is the path to the .regula library directory.
	LibraryPath string

	// BaseURI is the base URI for RDF triples.
	BaseURI string

	// CacheDir is the directory for persistent content caching.
	CacheDir string

	// DryRun when true, plans the crawl without making network requests.
	DryRun bool

	// Resume when true, attempts to resume from a saved crawl state.
	Resume bool

	// StatePath is the path for saving/loading crawl state.
	StatePath string

	// UserAgent is the User-Agent header sent with requests.
	UserAgent string

	// DomainConfigs holds per-domain configuration overrides.
	DomainConfigs map[string]*DomainConfig

	// OutputFormat is the format for the crawl report (table, json).
	OutputFormat string
}

// DefaultCrawlConfig returns a CrawlConfig with sensible defaults.
func DefaultCrawlConfig() CrawlConfig {
	return CrawlConfig{
		MaxDepth:     DefaultCrawlMaxDepth,
		MaxDocuments: DefaultCrawlMaxDocuments,
		RateLimit:    DefaultCrawlRateLimit,
		Timeout:      DefaultCrawlTimeout,
		LibraryPath:  ".regula",
		BaseURI:      "https://regula.dev/regulations/",
		UserAgent:    DefaultCrawlUserAgent,
		OutputFormat: "table",
	}
}

// DomainConfig holds per-domain configuration for rate limiting and content extraction.
type DomainConfig struct {
	// RateLimit overrides the global rate limit for this domain.
	RateLimit time.Duration

	// UserAgent overrides the global user agent for this domain.
	UserAgent string

	// ContentSelector is an optional CSS-like selector hint for extracting main content.
	ContentSelector string
}

// DefaultDomainConfigs returns pre-configured domain settings for known US law sources.
func DefaultDomainConfigs() map[string]*DomainConfig {
	return map[string]*DomainConfig{
		"uscode.house.gov": {
			RateLimit: 3 * time.Second,
		},
		"www.ecfr.gov": {
			RateLimit: 2 * time.Second,
		},
		"www.law.cornell.edu": {
			RateLimit: 3 * time.Second,
		},
		"leginfo.legislature.ca.gov": {
			RateLimit: 3 * time.Second,
		},
	}
}

// SeedType indicates the type of crawl seed.
type SeedType string

const (
	// SeedTypeDocumentID seeds from an existing library document ID.
	SeedTypeDocumentID SeedType = "document_id"

	// SeedTypeCitation seeds from a raw citation string.
	SeedTypeCitation SeedType = "citation"

	// SeedTypeURL seeds from a direct URL.
	SeedTypeURL SeedType = "url"
)

// CrawlSeed represents a starting point for the crawler.
type CrawlSeed struct {
	// Type indicates the seed type.
	Type SeedType `json:"type"`

	// Value is the seed value (document ID, citation string, or URL).
	Value string `json:"value"`
}

// CrawlItemStatus tracks the processing state of a crawl item.
type CrawlItemStatus string

const (
	// CrawlItemPending indicates the item is queued for processing.
	CrawlItemPending CrawlItemStatus = "pending"

	// CrawlItemFetching indicates the item is currently being fetched.
	CrawlItemFetching CrawlItemStatus = "fetching"

	// CrawlItemIngested indicates the item has been successfully ingested.
	CrawlItemIngested CrawlItemStatus = "ingested"

	// CrawlItemFailed indicates the item failed to process.
	CrawlItemFailed CrawlItemStatus = "failed"

	// CrawlItemSkipped indicates the item was skipped (already in library, duplicate, etc.).
	CrawlItemSkipped CrawlItemStatus = "skipped"
)

// CrawlItem represents a single item in the BFS frontier queue.
type CrawlItem struct {
	// Citation is the citation or identifier that triggered this item.
	Citation string `json:"citation"`

	// URL is the resolved fetchable URL.
	URL string `json:"url,omitempty"`

	// DocumentID is the derived document ID for the library.
	DocumentID string `json:"document_id"`

	// Depth is the BFS depth at which this item was discovered.
	Depth int `json:"depth"`

	// DiscoveredBy is the document ID of the document that led to this discovery.
	DiscoveredBy string `json:"discovered_by,omitempty"`

	// Status is the processing state of this item.
	Status CrawlItemStatus `json:"status"`

	// Error is the error message if the item failed.
	Error string `json:"error,omitempty"`

	// Domain is the source domain of the URL.
	Domain string `json:"domain,omitempty"`

	// FetchedAt is the timestamp when the item was fetched.
	FetchedAt time.Time `json:"fetched_at,omitempty"`
}

// FetchedContent holds the result of fetching a URL.
type FetchedContent struct {
	// URL is the URL that was fetched.
	URL string `json:"url"`

	// RawHTML is the raw HTML content (nil for non-HTML responses).
	RawHTML []byte `json:"-"`

	// PlainText is the extracted plain text content.
	PlainText []byte `json:"plain_text_length"`

	// ContentType is the HTTP Content-Type header.
	ContentType string `json:"content_type"`

	// StatusCode is the HTTP response status code.
	StatusCode int `json:"status_code"`

	// FetchedAt is the timestamp when the content was fetched.
	FetchedAt time.Time `json:"fetched_at"`

	// Cached indicates whether this content came from the cache.
	Cached bool `json:"cached"`
}

// SourceMapping maps a citation pattern to a URL template and metadata.
type SourceMapping struct {
	// Name is the human-readable name of this source.
	Name string

	// Domain is the domain this mapping targets.
	Domain string

	// Description describes what this source covers.
	Description string
}
