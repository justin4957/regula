package eurlex

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coolbeans/regula/pkg/citation"
)

// MockHTTPClient implements HTTPClient for testing.
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (mockClient *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return mockClient.DoFunc(req)
}

// newTestClient creates an EURLexClient with a mock HTTP client and short rate limit
// for fast tests.
func newTestClient(mockClient *MockHTTPClient) *EURLexClient {
	return &EURLexClient{
		httpClient: mockClient,
		cache:      NewValidationCache(1 * time.Hour),
		userAgent:  DefaultUserAgent,
	}
}

func TestValidateURI_200OK(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		},
	}

	eurlexClient := newTestClient(mockClient)
	validationResult, err := eurlexClient.ValidateURI("http://data.europa.eu/eli/reg/2016/679/oj")
	if err != nil {
		t.Fatalf("ValidateURI failed: %v", err)
	}

	if !validationResult.Valid {
		t.Error("Expected Valid to be true for 200 OK")
	}
	if validationResult.StatusCode != 200 {
		t.Errorf("StatusCode: got %d, want 200", validationResult.StatusCode)
	}
	if validationResult.URI != "http://data.europa.eu/eli/reg/2016/679/oj" {
		t.Errorf("URI: got %q, want ELI URI", validationResult.URI)
	}
	if validationResult.Error != "" {
		t.Errorf("Error: expected empty, got %q", validationResult.Error)
	}
}

func TestValidateURI_404NotFound(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       http.NoBody,
			}, nil
		},
	}

	eurlexClient := newTestClient(mockClient)
	validationResult, err := eurlexClient.ValidateURI("http://data.europa.eu/eli/reg/2016/999/oj")
	if err != nil {
		t.Fatalf("ValidateURI failed: %v", err)
	}

	if validationResult.Valid {
		t.Error("Expected Valid to be false for 404")
	}
	if validationResult.StatusCode != 404 {
		t.Errorf("StatusCode: got %d, want 404", validationResult.StatusCode)
	}
}

func TestValidateURI_Redirect(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusMovedPermanently,
				Body:       http.NoBody,
			}, nil
		},
	}

	eurlexClient := newTestClient(mockClient)
	validationResult, err := eurlexClient.ValidateURI("http://data.europa.eu/eli/dir/1995/46/oj")
	if err != nil {
		t.Fatalf("ValidateURI failed: %v", err)
	}

	if !validationResult.Valid {
		t.Error("Expected Valid to be true for 301 redirect (status < 400)")
	}
	if validationResult.StatusCode != 301 {
		t.Errorf("StatusCode: got %d, want 301", validationResult.StatusCode)
	}
}

func TestValidateURI_NetworkError(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	eurlexClient := newTestClient(mockClient)
	validationResult, err := eurlexClient.ValidateURI("http://data.europa.eu/eli/reg/2016/679/oj")
	if err != nil {
		t.Fatalf("ValidateURI should not return Go error for network failures: %v", err)
	}

	if validationResult.Valid {
		t.Error("Expected Valid to be false for network error")
	}
	if validationResult.Error == "" {
		t.Error("Expected Error field to be populated for network error")
	}
	if !strings.Contains(validationResult.Error, "connection refused") {
		t.Errorf("Error should contain underlying error message, got %q", validationResult.Error)
	}
}

func TestValidateURI_Caching(t *testing.T) {
	var requestCount atomic.Int32

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			requestCount.Add(1)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		},
	}

	eurlexClient := newTestClient(mockClient)
	targetURI := "http://data.europa.eu/eli/reg/2016/679/oj"

	// First call should hit the network.
	firstResult, err := eurlexClient.ValidateURI(targetURI)
	if err != nil {
		t.Fatalf("First ValidateURI failed: %v", err)
	}
	if !firstResult.Valid {
		t.Error("First call: expected Valid")
	}

	// Second call should hit the cache.
	secondResult, err := eurlexClient.ValidateURI(targetURI)
	if err != nil {
		t.Fatalf("Second ValidateURI failed: %v", err)
	}
	if !secondResult.Valid {
		t.Error("Second call: expected Valid")
	}

	if requestCount.Load() != 1 {
		t.Errorf("Expected 1 HTTP request (second should be cached), got %d", requestCount.Load())
	}
}

func TestValidateCitation_EndToEnd(t *testing.T) {
	var capturedURL string

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			capturedURL = req.URL.String()
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		},
	}

	eurlexClient := newTestClient(mockClient)

	gdprCitation := &citation.Citation{
		Type:       citation.CitationTypeRegulation,
		Components: citation.CitationComponents{DocYear: "2016", DocNumber: "679"},
	}

	validationResult, err := eurlexClient.ValidateCitation(gdprCitation)
	if err != nil {
		t.Fatalf("ValidateCitation failed: %v", err)
	}

	if !validationResult.Valid {
		t.Error("Expected GDPR citation to validate successfully")
	}

	expectedURL := "http://data.europa.eu/eli/reg/2016/679/oj"
	if capturedURL != expectedURL {
		t.Errorf("Request URL: got %q, want %q", capturedURL, expectedURL)
	}
}

func TestValidateCitation_InvalidCitation(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			t.Fatal("HTTP request should not be made for invalid citation")
			return nil, nil
		},
	}

	eurlexClient := newTestClient(mockClient)

	// Missing year should cause GenerateELI to fail.
	invalidCitation := &citation.Citation{
		Type:       citation.CitationTypeRegulation,
		Components: citation.CitationComponents{DocNumber: "679"},
	}

	_, err := eurlexClient.ValidateCitation(invalidCitation)
	if err == nil {
		t.Error("Expected error for citation missing year")
	}
}

func TestFetchMetadata_Success(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		},
	}

	eurlexClient := newTestClient(mockClient)
	metadata, err := eurlexClient.FetchMetadata("32016R0679")
	if err != nil {
		t.Fatalf("FetchMetadata failed: %v", err)
	}

	if metadata.CELEX != "32016R0679" {
		t.Errorf("CELEX: got %q, want '32016R0679'", metadata.CELEX)
	}
}

func TestFetchMetadata_NotFound(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       http.NoBody,
			}, nil
		},
	}

	eurlexClient := newTestClient(mockClient)
	_, err := eurlexClient.FetchMetadata("3XXXX0000")
	if err == nil {
		t.Error("Expected error for 404 response")
	}
}

func TestFetchMetadata_NetworkError(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("DNS resolution failed")
		},
	}

	eurlexClient := newTestClient(mockClient)
	_, err := eurlexClient.FetchMetadata("32016R0679")
	if err == nil {
		t.Error("Expected error for network failure")
	}
}

func TestUserAgentHeader(t *testing.T) {
	var capturedUserAgent string

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			capturedUserAgent = req.Header.Get("User-Agent")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		},
	}

	eurlexClient := newTestClient(mockClient)
	_, err := eurlexClient.ValidateURI("http://data.europa.eu/eli/reg/2016/679/oj")
	if err != nil {
		t.Fatalf("ValidateURI failed: %v", err)
	}

	if capturedUserAgent != DefaultUserAgent {
		t.Errorf("User-Agent: got %q, want %q", capturedUserAgent, DefaultUserAgent)
	}
}

func TestHTTPMethodUsed(t *testing.T) {
	var validateMethod string
	var fetchMethod string

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.String(), "eli/") {
				validateMethod = req.Method
			} else {
				fetchMethod = req.Method
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		},
	}

	eurlexClient := newTestClient(mockClient)

	// ValidateURI should use HEAD.
	_, err := eurlexClient.ValidateURI("http://data.europa.eu/eli/reg/2016/679/oj")
	if err != nil {
		t.Fatalf("ValidateURI failed: %v", err)
	}
	if validateMethod != http.MethodHead {
		t.Errorf("ValidateURI method: got %q, want %q", validateMethod, http.MethodHead)
	}

	// FetchMetadata should use GET.
	_, err = eurlexClient.FetchMetadata("32016R0679")
	if err != nil {
		t.Fatalf("FetchMetadata failed: %v", err)
	}
	if fetchMethod != http.MethodGet {
		t.Errorf("FetchMetadata method: got %q, want %q", fetchMethod, http.MethodGet)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.RateLimit != DefaultRequestInterval {
		t.Errorf("RateLimit: got %v, want %v", config.RateLimit, DefaultRequestInterval)
	}
	if config.CacheTTL != DefaultCacheTTL {
		t.Errorf("CacheTTL: got %v, want %v", config.CacheTTL, DefaultCacheTTL)
	}
	if config.HTTPClient != nil {
		t.Error("HTTPClient: expected nil (uses http.DefaultClient)")
	}
	if config.UserAgent != DefaultUserAgent {
		t.Errorf("UserAgent: got %q, want %q", config.UserAgent, DefaultUserAgent)
	}
}

func TestNewEURLexClient_DefaultHTTPClient(t *testing.T) {
	config := DefaultConfig()
	eurlexClient := NewEURLexClient(config)

	if eurlexClient.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}
	if eurlexClient.cache == nil {
		t.Error("Expected cache to be initialized")
	}
	if eurlexClient.userAgent != DefaultUserAgent {
		t.Errorf("UserAgent: got %q, want %q", eurlexClient.userAgent, DefaultUserAgent)
	}
}

func TestFetchMetadata_ServerError(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       http.NoBody,
			}, nil
		},
	}

	eurlexClient := newTestClient(mockClient)
	_, err := eurlexClient.FetchMetadata("32016R0679")
	if err == nil {
		t.Error("Expected error for 500 server error")
	}
}

func TestFetchMetadata_RequestURL(t *testing.T) {
	var capturedURL string

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			capturedURL = req.URL.String()
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		},
	}

	eurlexClient := newTestClient(mockClient)
	_, err := eurlexClient.FetchMetadata("32016R0679")
	if err != nil {
		t.Fatalf("FetchMetadata failed: %v", err)
	}

	expectedURL := "https://eur-lex.europa.eu/legal-content/EN/ALL/?uri=CELEX:32016R0679"
	if capturedURL != expectedURL {
		t.Errorf("Request URL: got %q, want %q", capturedURL, expectedURL)
	}
}
