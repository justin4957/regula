package ukleg

import (
	"fmt"
	"net/http"
	"time"

	"github.com/coolbeans/regula/pkg/citation"
)

// DefaultUserAgent is the default User-Agent header sent with legislation.gov.uk requests.
const DefaultUserAgent = "regula-ukleg-connector/1.0"

// UKLegClientConfig holds configuration for a UKLegClient.
type UKLegClientConfig struct {
	// RateLimit is the minimum interval between HTTP requests to legislation.gov.uk.
	// Default: 1 second.
	RateLimit time.Duration

	// CacheTTL is the time-to-live for cached validation results.
	// Default: 1 hour.
	CacheTTL time.Duration

	// HTTPClient is the underlying HTTP client used for requests.
	// If nil, http.DefaultClient is used (wrapped with rate limiting).
	HTTPClient HTTPClient

	// UserAgent is the User-Agent header sent with requests.
	// Default: "regula-ukleg-connector/1.0".
	UserAgent string
}

// DefaultConfig returns a UKLegClientConfig with sensible defaults.
func DefaultConfig() UKLegClientConfig {
	return UKLegClientConfig{
		RateLimit:  DefaultRequestInterval,
		CacheTTL:   DefaultCacheTTL,
		HTTPClient: nil,
		UserAgent:  DefaultUserAgent,
	}
}

// UKLegClient provides legislation.gov.uk connectivity: URI validation,
// citation validation, and document metadata fetching with rate limiting and caching.
type UKLegClient struct {
	httpClient HTTPClient
	cache      *ValidationCache
	userAgent  string
}

// NewUKLegClient creates a new UKLegClient with the given configuration.
// If config.HTTPClient is nil, http.DefaultClient is used and wrapped with rate limiting.
func NewUKLegClient(config UKLegClientConfig) *UKLegClient {
	underlyingClient := config.HTTPClient
	if underlyingClient == nil {
		underlyingClient = http.DefaultClient
	}

	rateLimitedClient := NewRateLimitedHTTPClient(underlyingClient, config.RateLimit)

	userAgent := config.UserAgent
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}

	return &UKLegClient{
		httpClient: rateLimitedClient,
		cache:      NewValidationCache(config.CacheTTL),
		userAgent:  userAgent,
	}
}

// ValidateURI performs an HTTP HEAD request to the given URI to check if the
// resource exists on legislation.gov.uk. Results are cached for the configured TTL.
//
// A status code < 400 is considered valid (includes 200, 301, 302 redirects).
// Network errors and status codes >= 400 are considered invalid.
func (ukLegClient *UKLegClient) ValidateURI(uri string) (*ValidationResult, error) {
	// Check cache first.
	if cachedResult, found := ukLegClient.cache.Get(uri); found {
		return &cachedResult, nil
	}

	request, err := http.NewRequest(http.MethodHead, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", uri, err)
	}
	request.Header.Set("User-Agent", ukLegClient.userAgent)

	response, err := ukLegClient.httpClient.Do(request)
	if err != nil {
		// Network error — return as a validation result (not a Go error),
		// since the failure is expected in normal operation.
		networkErrorResult := ValidationResult{
			URI:       uri,
			Valid:     false,
			CheckedAt: time.Now(),
			Error:     err.Error(),
		}
		ukLegClient.cache.Set(uri, networkErrorResult)
		return &networkErrorResult, nil
	}
	defer response.Body.Close()

	validationResult := ValidationResult{
		URI:        uri,
		Valid:      response.StatusCode < 400,
		StatusCode: response.StatusCode,
		CheckedAt:  time.Now(),
	}

	ukLegClient.cache.Set(uri, validationResult)
	return &validationResult, nil
}

// ValidateCitation generates a legislation.gov.uk URI from the citation and validates
// it. This is a convenience method combining GenerateLegislationURI and ValidateURI.
func (ukLegClient *UKLegClient) ValidateCitation(citationRef *citation.Citation) (*ValidationResult, error) {
	legislationURI, err := GenerateLegislationURI(citationRef)
	if err != nil {
		return nil, fmt.Errorf("failed to generate legislation URI for citation: %w", err)
	}

	return ukLegClient.ValidateURI(legislationURI.String())
}

// FetchMetadata retrieves basic document metadata from legislation.gov.uk for the
// given legislation URI. This performs a GET request to the legislation page.
func (ukLegClient *UKLegClient) FetchMetadata(legislationURI *LegislationURI) (*DocumentMetadata, error) {
	metadataURL := legislationURI.String()

	request, err := http.NewRequest(http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata request for %s: %w", metadataURL, err)
	}
	request.Header.Set("User-Agent", ukLegClient.userAgent)
	request.Header.Set("Accept", "application/xml")

	response, err := ukLegClient.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata for %s: %w", metadataURL, err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("document not found at %s (HTTP %d)", metadataURL, response.StatusCode)
	}

	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("legislation.gov.uk returned HTTP %d for %s", response.StatusCode, metadataURL)
	}

	// Return minimal metadata with the legislation reference confirmed.
	// Full XML parsing of legislation.gov.uk responses is deferred to a future enhancement.
	metadata := &DocumentMetadata{
		LegislationType: string(legislationURI.LegislationType),
		Year:            legislationURI.Year,
		Number:          legislationURI.Number,
		URI:             metadataURL,
	}

	return metadata, nil
}
