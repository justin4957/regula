package uscode

import (
	"fmt"
	"net/http"
	"time"

	"github.com/coolbeans/regula/pkg/citation"
)

// DefaultUserAgent is the default User-Agent header sent with US Code requests.
const DefaultUserAgent = "regula-uscode-connector/1.0"

// USCodeClientConfig holds configuration for a USCodeClient.
type USCodeClientConfig struct {
	// RateLimit is the minimum interval between HTTP requests.
	// Default: 1 second.
	RateLimit time.Duration

	// CacheTTL is the time-to-live for cached validation results.
	// Default: 1 hour.
	CacheTTL time.Duration

	// HTTPClient is the underlying HTTP client used for requests.
	// If nil, http.DefaultClient is used (wrapped with rate limiting).
	HTTPClient HTTPClient

	// UserAgent is the User-Agent header sent with requests.
	// Default: "regula-uscode-connector/1.0".
	UserAgent string
}

// DefaultUSCodeConfig returns a USCodeClientConfig with sensible defaults.
func DefaultUSCodeConfig() USCodeClientConfig {
	return USCodeClientConfig{
		RateLimit:  DefaultRequestInterval,
		CacheTTL:   DefaultCacheTTL,
		HTTPClient: nil, // Will use http.DefaultClient.
		UserAgent:  DefaultUserAgent,
	}
}

// USCodeClient provides US Code connectivity: URI validation, citation validation,
// and document metadata fetching with rate limiting and caching.
type USCodeClient struct {
	httpClient HTTPClient
	cache      *ValidationCache
	userAgent  string
}

// NewUSCodeClient creates a new USCodeClient with the given configuration.
// If config.HTTPClient is nil, http.DefaultClient is used and wrapped with rate limiting.
func NewUSCodeClient(config USCodeClientConfig) *USCodeClient {
	underlyingClient := config.HTTPClient
	if underlyingClient == nil {
		underlyingClient = http.DefaultClient
	}

	// Wrap with rate limiting if an interval is specified.
	rateLimitedClient := NewRateLimitedHTTPClient(underlyingClient, config.RateLimit)

	userAgent := config.UserAgent
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}

	return &USCodeClient{
		httpClient: rateLimitedClient,
		cache:      NewValidationCache(config.CacheTTL),
		userAgent:  userAgent,
	}
}

// ValidateURI performs an HTTP HEAD request to the given URI to check if the
// resource exists. Results are cached for the configured TTL.
//
// A status code < 400 is considered valid (includes 200, 301, 302 redirects).
// Network errors and status codes >= 400 are considered invalid.
func (uscClient *USCodeClient) ValidateURI(uri string) (*ValidationResult, error) {
	// Check cache first.
	if cachedResult, found := uscClient.cache.Get(uri); found {
		return &cachedResult, nil
	}

	request, err := http.NewRequest(http.MethodHead, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", uri, err)
	}
	request.Header.Set("User-Agent", uscClient.userAgent)

	response, err := uscClient.httpClient.Do(request)
	if err != nil {
		// Network error — return as a validation result (not a Go error),
		// since the failure is expected in normal operation.
		networkErrorResult := ValidationResult{
			URI:       uri,
			Valid:     false,
			CheckedAt: time.Now(),
			Error:     err.Error(),
		}
		uscClient.cache.Set(uri, networkErrorResult)
		return &networkErrorResult, nil
	}
	defer response.Body.Close()

	validationResult := ValidationResult{
		URI:        uri,
		Valid:      response.StatusCode < 400,
		StatusCode: response.StatusCode,
		CheckedAt:  time.Now(),
	}

	uscClient.cache.Set(uri, validationResult)
	return &validationResult, nil
}

// ValidateUSCCitation generates a USC URI from the citation and validates it
// against uscode.house.gov. This is a convenience method combining GenerateUSCURI and ValidateURI.
func (uscClient *USCodeClient) ValidateUSCCitation(citationRef *citation.Citation) (*ValidationResult, error) {
	uscURI, err := GenerateUSCURI(citationRef)
	if err != nil {
		return nil, fmt.Errorf("failed to generate USC URI for citation: %w", err)
	}

	return uscClient.ValidateURI(uscURI.String())
}

// ValidateCFRCitation generates a CFR URI from the citation and validates it
// against ecfr.gov. This is a convenience method combining GenerateCFRURI and ValidateURI.
func (uscClient *USCodeClient) ValidateCFRCitation(citationRef *citation.Citation) (*ValidationResult, error) {
	cfrURI, err := GenerateCFRURI(citationRef)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CFR URI for citation: %w", err)
	}

	return uscClient.ValidateURI(cfrURI.String())
}

// ValidateUSCNumber validates a USC citation using the parsed USCNumber struct.
func (uscClient *USCodeClient) ValidateUSCNumber(uscNumber *USCNumber) (*ValidationResult, error) {
	if uscNumber == nil {
		return nil, fmt.Errorf("USC number is nil")
	}

	uscURI := USCURI{
		Title:   uscNumber.Title,
		Section: uscNumber.Section,
	}

	return uscClient.ValidateURI(uscURI.String())
}

// ValidateCFRNumber validates a CFR citation using the parsed CFRNumber struct.
func (uscClient *USCodeClient) ValidateCFRNumber(cfrNumber *CFRNumber) (*ValidationResult, error) {
	if cfrNumber == nil {
		return nil, fmt.Errorf("CFR number is nil")
	}

	cfrURI := CFRURI{
		Title:   cfrNumber.Title,
		Part:    cfrNumber.Part,
		Section: cfrNumber.Section,
	}

	return uscClient.ValidateURI(cfrURI.String())
}

// FetchUSCMetadata retrieves basic document metadata from uscode.house.gov for the given USC citation.
// This performs a GET request to verify the document exists.
func (uscClient *USCodeClient) FetchUSCMetadata(uscNumber *USCNumber) (*DocumentMetadata, error) {
	if uscNumber == nil {
		return nil, fmt.Errorf("USC number is nil")
	}

	uscURI := USCURI{
		Title:   uscNumber.Title,
		Section: uscNumber.Section,
	}
	metadataURL := uscURI.String()

	request, err := http.NewRequest(http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata request for %s: %w", uscNumber.String(), err)
	}
	request.Header.Set("User-Agent", uscClient.userAgent)

	response, err := uscClient.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata for %s: %w", uscNumber.String(), err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("document not found for %s (HTTP %d)", uscNumber.String(), response.StatusCode)
	}

	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("uscode.house.gov returned HTTP %d for %s", response.StatusCode, uscNumber.String())
	}

	// Return minimal metadata with the citation confirmed.
	// Full HTML parsing of US Code pages is deferred to a future enhancement.
	metadata := &DocumentMetadata{
		Title:       uscNumber.Title,
		Section:     uscNumber.Section,
		DisplayName: uscNumber.String(),
	}

	return metadata, nil
}

// FetchCFRMetadata retrieves basic document metadata from ecfr.gov for the given CFR citation.
// This performs a GET request to verify the document exists.
func (uscClient *USCodeClient) FetchCFRMetadata(cfrNumber *CFRNumber) (*DocumentMetadata, error) {
	if cfrNumber == nil {
		return nil, fmt.Errorf("CFR number is nil")
	}

	cfrURI := CFRURI{
		Title:   cfrNumber.Title,
		Part:    cfrNumber.Part,
		Section: cfrNumber.Section,
	}
	metadataURL := cfrURI.String()

	request, err := http.NewRequest(http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata request for %s: %w", cfrNumber.String(), err)
	}
	request.Header.Set("User-Agent", uscClient.userAgent)

	response, err := uscClient.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata for %s: %w", cfrNumber.String(), err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("document not found for %s (HTTP %d)", cfrNumber.String(), response.StatusCode)
	}

	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("ecfr.gov returned HTTP %d for %s", response.StatusCode, cfrNumber.String())
	}

	// Return minimal metadata with the citation confirmed.
	metadata := &DocumentMetadata{
		Title:       cfrNumber.Title,
		Section:     cfrNumber.Part,
		DisplayName: cfrNumber.String(),
	}

	return metadata, nil
}
