package uscode

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

// DefaultRequestInterval is the default minimum interval between HTTP requests
// to US Code servers, to avoid overwhelming the service.
const DefaultRequestInterval = 1 * time.Second

// RateLimitedHTTPClient wraps an HTTPClient with a token-bucket style rate limiter
// that enforces a minimum interval between requests.
type RateLimitedHTTPClient struct {
	underlying      HTTPClient
	ticker          *time.Ticker
	requestInterval time.Duration
	mu              sync.Mutex
	closed          bool
}

// NewRateLimitedHTTPClient creates a rate-limited HTTP client that enforces
// the given minimum interval between requests.
func NewRateLimitedHTTPClient(underlying HTTPClient, requestInterval time.Duration) *RateLimitedHTTPClient {
	return &RateLimitedHTTPClient{
		underlying:      underlying,
		ticker:          time.NewTicker(requestInterval),
		requestInterval: requestInterval,
	}
}

// Do executes an HTTP request, waiting for the rate limiter before sending.
func (rateLimitedClient *RateLimitedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	rateLimitedClient.mu.Lock()
	if !rateLimitedClient.closed {
		<-rateLimitedClient.ticker.C
	}
	rateLimitedClient.mu.Unlock()

	return rateLimitedClient.underlying.Do(req)
}

// Close stops the rate limiter's internal ticker and releases resources.
func (rateLimitedClient *RateLimitedHTTPClient) Close() {
	rateLimitedClient.mu.Lock()
	defer rateLimitedClient.mu.Unlock()

	if !rateLimitedClient.closed {
		rateLimitedClient.ticker.Stop()
		rateLimitedClient.closed = true
	}
}
