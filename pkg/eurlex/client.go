package eurlex

import (
	"fmt"
	"net/http"
	"time"

	"github.com/coolbeans/regula/pkg/citation"
)

// DefaultUserAgent is the default User-Agent header sent with EUR-Lex requests.
const DefaultUserAgent = "regula-eurlex-connector/1.0"

// EURLexClientConfig holds configuration for an EURLexClient.
type EURLexClientConfig struct {
	// RateLimit is the minimum interval between HTTP requests to EUR-Lex.
	// Default: 1 second.
	RateLimit time.Duration

	// CacheTTL is the time-to-live for cached validation results.
	// Default: 1 hour.
	CacheTTL time.Duration

	// HTTPClient is the underlying HTTP client used for requests.
	// If nil, http.DefaultClient is used (wrapped with rate limiting).
	HTTPClient HTTPClient

	// UserAgent is the User-Agent header sent with requests.
	// Default: "regula-eurlex-connector/1.0".
	UserAgent string
}

// DefaultConfig returns an EURLexClientConfig with sensible defaults.
func DefaultConfig() EURLexClientConfig {
	return EURLexClientConfig{
		RateLimit:  DefaultRequestInterval,
		CacheTTL:   DefaultCacheTTL,
		HTTPClient: nil, // Will use http.DefaultClient.
		UserAgent:  DefaultUserAgent,
	}
}

// EURLexClient provides EUR-Lex connectivity: URI validation, citation validation,
// and document metadata fetching with rate limiting and caching.
type EURLexClient struct {
	httpClient HTTPClient
	cache      *ValidationCache
	userAgent  string
}

// NewEURLexClient creates a new EURLexClient with the given configuration.
// If config.HTTPClient is nil, http.DefaultClient is used and wrapped with rate limiting.
func NewEURLexClient(config EURLexClientConfig) *EURLexClient {
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

	return &EURLexClient{
		httpClient: rateLimitedClient,
		cache:      NewValidationCache(config.CacheTTL),
		userAgent:  userAgent,
	}
}

// ValidateURI performs an HTTP HEAD request to the given URI to check if the
// resource exists on EUR-Lex. Results are cached for the configured TTL.
//
// A status code < 400 is considered valid (includes 200, 301, 302 redirects).
// Network errors and status codes >= 400 are considered invalid.
func (eurlexClient *EURLexClient) ValidateURI(uri string) (*ValidationResult, error) {
	// Check cache first.
	if cachedResult, found := eurlexClient.cache.Get(uri); found {
		return &cachedResult, nil
	}

	request, err := http.NewRequest(http.MethodHead, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", uri, err)
	}
	request.Header.Set("User-Agent", eurlexClient.userAgent)

	response, err := eurlexClient.httpClient.Do(request)
	if err != nil {
		// Network error — return as a validation result (not a Go error),
		// since the failure is expected in normal operation.
		networkErrorResult := ValidationResult{
			URI:       uri,
			Valid:     false,
			CheckedAt: time.Now(),
			Error:     err.Error(),
		}
		eurlexClient.cache.Set(uri, networkErrorResult)
		return &networkErrorResult, nil
	}
	defer response.Body.Close()

	validationResult := ValidationResult{
		URI:        uri,
		Valid:      response.StatusCode < 400,
		StatusCode: response.StatusCode,
		CheckedAt:  time.Now(),
	}

	eurlexClient.cache.Set(uri, validationResult)
	return &validationResult, nil
}

// ValidateCitation generates an ELI URI from the citation and validates it
// against EUR-Lex. This is a convenience method combining GenerateELI and ValidateURI.
func (eurlexClient *EURLexClient) ValidateCitation(citationRef *citation.Citation) (*ValidationResult, error) {
	eliURI, err := GenerateELI(citationRef)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ELI for citation: %w", err)
	}

	return eurlexClient.ValidateURI(eliURI.String())
}

// FetchMetadata retrieves basic document metadata from EUR-Lex for the given CELEX number.
// This performs a GET request to the EUR-Lex search endpoint.
func (eurlexClient *EURLexClient) FetchMetadata(celexNumber string) (*DocumentMetadata, error) {
	metadataURL := "https://eur-lex.europa.eu/legal-content/EN/ALL/?uri=CELEX:" + celexNumber

	request, err := http.NewRequest(http.MethodGet, metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata request for CELEX %s: %w", celexNumber, err)
	}
	request.Header.Set("User-Agent", eurlexClient.userAgent)

	response, err := eurlexClient.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata for CELEX %s: %w", celexNumber, err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("document not found for CELEX %s (HTTP %d)", celexNumber, response.StatusCode)
	}

	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("EUR-Lex returned HTTP %d for CELEX %s", response.StatusCode, celexNumber)
	}

	// Return minimal metadata with the CELEX number confirmed.
	// Full HTML parsing of EUR-Lex pages is deferred to a future enhancement.
	metadata := &DocumentMetadata{
		CELEX: celexNumber,
	}

	return metadata, nil
}
