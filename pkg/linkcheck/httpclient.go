package linkcheck

import (
	"net/http"
	"sync"
	"time"
)

// HTTPClient is an interface matching the Do method of *http.Client.
// This allows injection of mock clients for testing and custom transports.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// RateLimitedHTTPClient wraps an HTTPClient with a token-bucket style rate limiter
// that enforces a minimum interval between requests.
type RateLimitedHTTPClient struct {
	underlying      HTTPClient
	requestInterval time.Duration
	lastRequest     time.Time
	mu              sync.Mutex
}

// NewRateLimitedHTTPClient creates a rate-limited HTTP client that enforces
// the given minimum interval between requests.
func NewRateLimitedHTTPClient(underlying HTTPClient, requestInterval time.Duration) *RateLimitedHTTPClient {
	return &RateLimitedHTTPClient{
		underlying:      underlying,
		requestInterval: requestInterval,
		lastRequest:     time.Time{}, // Zero time means no requests yet
	}
}

// Do executes an HTTP request, waiting for the rate limiter before sending.
func (rateLimitedClient *RateLimitedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	rateLimitedClient.mu.Lock()

	// Calculate how long to wait
	if !rateLimitedClient.lastRequest.IsZero() {
		elapsed := time.Since(rateLimitedClient.lastRequest)
		if elapsed < rateLimitedClient.requestInterval {
			waitTime := rateLimitedClient.requestInterval - elapsed
			rateLimitedClient.mu.Unlock()
			time.Sleep(waitTime)
			rateLimitedClient.mu.Lock()
		}
	}

	rateLimitedClient.lastRequest = time.Now()
	rateLimitedClient.mu.Unlock()

	return rateLimitedClient.underlying.Do(req)
}

// DomainRateLimiter manages rate limiting across multiple domains.
type DomainRateLimiter struct {
	clients       map[string]*RateLimitedHTTPClient
	underlying    HTTPClient
	defaultConfig *BatchConfig
	mu            sync.Mutex
}

// NewDomainRateLimiter creates a rate limiter that manages per-domain rate limits.
func NewDomainRateLimiter(underlying HTTPClient, config *BatchConfig) *DomainRateLimiter {
	return &DomainRateLimiter{
		clients:       make(map[string]*RateLimitedHTTPClient),
		underlying:    underlying,
		defaultConfig: config,
	}
}

// GetClient returns a rate-limited client for the given domain.
func (domainRateLimiter *DomainRateLimiter) GetClient(domain string) *RateLimitedHTTPClient {
	domainRateLimiter.mu.Lock()
	defer domainRateLimiter.mu.Unlock()

	if client, exists := domainRateLimiter.clients[domain]; exists {
		return client
	}

	// Create new client for this domain
	domainConfig := domainRateLimiter.defaultConfig.GetDomainConfig(domain)
	client := NewRateLimitedHTTPClient(domainRateLimiter.underlying, domainConfig.RateLimit)
	domainRateLimiter.clients[domain] = client
	return client
}

// TimeoutHTTPClient wraps an HTTPClient with a configurable timeout.
type TimeoutHTTPClient struct {
	underlying HTTPClient
	timeout    time.Duration
}

// NewTimeoutHTTPClient creates an HTTP client with the specified timeout.
func NewTimeoutHTTPClient(underlying HTTPClient, timeout time.Duration) *TimeoutHTTPClient {
	return &TimeoutHTTPClient{
		underlying: underlying,
		timeout:    timeout,
	}
}

// Do executes an HTTP request with the configured timeout.
func (timeoutClient *TimeoutHTTPClient) Do(req *http.Request) (*http.Response, error) {
	// Create HTTP client with timeout for this specific request
	httpClient := &http.Client{
		Timeout: timeoutClient.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	return httpClient.Do(req)
}
