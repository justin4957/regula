// Package fetch provides recursive document fetching for building federated
// knowledge graphs that span multiple related legal documents.
package fetch

import (
	"time"
)

// DefaultMaxDepth is the default maximum BFS recursion depth for fetching.
const DefaultMaxDepth = 2

// DefaultMaxDocuments is the default maximum number of documents to fetch.
const DefaultMaxDocuments = 10

// DefaultFetchRateLimit is the default minimum interval between HTTP requests.
const DefaultFetchRateLimit = 2 * time.Second

// DefaultFetchTimeout is the default per-request timeout.
const DefaultFetchTimeout = 30 * time.Second

// DefaultCacheTTL is the default time-to-live for cached fetch results.
const DefaultCacheTTL = 24 * time.Hour

// FetchConfig holds configuration for recursive document fetching.
type FetchConfig struct {
	// MaxDepth is the maximum BFS recursion depth for fetching referenced documents.
	MaxDepth int

	// MaxDocuments is the maximum number of external documents to fetch.
	MaxDocuments int

	// AllowedDomains restricts fetching to these domains. If empty, all domains are allowed.
	AllowedDomains []string

	// RateLimit is the minimum interval between HTTP requests per domain.
	RateLimit time.Duration

	// Timeout is the per-request timeout.
	Timeout time.Duration

	// CacheDir is the directory for persistent fetch result caching.
	// If empty, caching is disabled.
	CacheDir string

	// DryRun when true, plans what would be fetched without making network calls.
	DryRun bool
}

// DefaultFetchConfig returns a FetchConfig with sensible defaults.
func DefaultFetchConfig() FetchConfig {
	return FetchConfig{
		MaxDepth:     DefaultMaxDepth,
		MaxDocuments: DefaultMaxDocuments,
		RateLimit:    DefaultFetchRateLimit,
		Timeout:      DefaultFetchTimeout,
	}
}

// FetchableReference represents an external reference that can be fetched.
type FetchableReference struct {
	// URN is the original URN identifier from the triple store.
	URN string `json:"urn"`

	// URL is the resolved HTTP URL for fetching.
	URL string `json:"url"`

	// SourceURI is the triple store subject that references this document.
	SourceURI string `json:"source_uri"`

	// Depth is the BFS depth level at which this reference was discovered.
	Depth int `json:"depth"`
}

// FetchResult captures the outcome of fetching a single external reference.
type FetchResult struct {
	// Reference is the fetchable reference that was processed.
	Reference FetchableReference `json:"reference"`

	// Success indicates whether the fetch succeeded.
	Success bool `json:"success"`

	// StatusCode is the HTTP status code from the fetch request.
	StatusCode int `json:"status_code"`

	// Metadata holds key-value pairs of fetched document metadata.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Error describes any error that occurred during fetching.
	Error string `json:"error,omitempty"`

	// Cached indicates this result came from the disk cache.
	Cached bool `json:"cached"`

	// FetchedAt is the timestamp when this result was obtained.
	FetchedAt time.Time `json:"fetched_at"`
}
