package ukleg

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

// newTestClient creates a UKLegClient with a mock HTTP client for fast tests.
func newTestClient(mockClient *MockHTTPClient) *UKLegClient {
	return &UKLegClient{
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

	ukLegClient := newTestClient(mockClient)
	validationResult, err := ukLegClient.ValidateURI("https://www.legislation.gov.uk/ukpga/2018/12")
	if err != nil {
		t.Fatalf("ValidateURI failed: %v", err)
	}

	if !validationResult.Valid {
		t.Error("Expected Valid to be true for 200 OK")
	}
	if validationResult.StatusCode != 200 {
		t.Errorf("StatusCode: got %d, want 200", validationResult.StatusCode)
	}
	if validationResult.URI != "https://www.legislation.gov.uk/ukpga/2018/12" {
		t.Errorf("URI: got %q, want legislation.gov.uk URI", validationResult.URI)
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

	ukLegClient := newTestClient(mockClient)
	validationResult, err := ukLegClient.ValidateURI("https://www.legislation.gov.uk/ukpga/2099/999")
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

	ukLegClient := newTestClient(mockClient)
	validationResult, err := ukLegClient.ValidateURI("https://www.legislation.gov.uk/ukpga/2018/12")
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

	ukLegClient := newTestClient(mockClient)
	validationResult, err := ukLegClient.ValidateURI("https://www.legislation.gov.uk/ukpga/2018/12")
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

	ukLegClient := newTestClient(mockClient)
	targetURI := "https://www.legislation.gov.uk/ukpga/2018/12"

	// First call should hit the network.
	firstResult, err := ukLegClient.ValidateURI(targetURI)
	if err != nil {
		t.Fatalf("First ValidateURI failed: %v", err)
	}
	if !firstResult.Valid {
		t.Error("First call: expected Valid")
	}

	// Second call should hit the cache.
	secondResult, err := ukLegClient.ValidateURI(targetURI)
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

	ukLegClient := newTestClient(mockClient)

	// Data Protection Act 2018 [2018 c. 12]
	dataProtectionAct := &citation.Citation{
		Type:         citation.CitationTypeStatute,
		Jurisdiction: "UK",
		Components: citation.CitationComponents{
			DocYear:   "2018",
			DocNumber: "12",
			CodeName:  "ukact",
		},
	}

	validationResult, err := ukLegClient.ValidateCitation(dataProtectionAct)
	if err != nil {
		t.Fatalf("ValidateCitation failed: %v", err)
	}

	if !validationResult.Valid {
		t.Error("Expected Data Protection Act citation to validate successfully")
	}

	expectedURL := "https://www.legislation.gov.uk/ukpga/2018/12"
	if capturedURL != expectedURL {
		t.Errorf("Request URL: got %q, want %q", capturedURL, expectedURL)
	}
}

func TestValidateCitation_SI(t *testing.T) {
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

	ukLegClient := newTestClient(mockClient)

	// SI 2019/419
	statutoryInstrument := &citation.Citation{
		Type:         citation.CitationTypeRegulation,
		Jurisdiction: "UK",
		Components: citation.CitationComponents{
			DocYear:   "2019",
			DocNumber: "419",
		},
	}

	validationResult, err := ukLegClient.ValidateCitation(statutoryInstrument)
	if err != nil {
		t.Fatalf("ValidateCitation failed: %v", err)
	}

	if !validationResult.Valid {
		t.Error("Expected SI citation to validate successfully")
	}

	expectedURL := "https://www.legislation.gov.uk/uksi/2019/419"
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

	ukLegClient := newTestClient(mockClient)

	// Missing year should cause GenerateLegislationURI to fail.
	invalidCitation := &citation.Citation{
		Type:       citation.CitationTypeStatute,
		Components: citation.CitationComponents{DocNumber: "12", CodeName: "ukact"},
	}

	_, err := ukLegClient.ValidateCitation(invalidCitation)
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

	ukLegClient := newTestClient(mockClient)
	legislationURI := &LegislationURI{
		LegislationType: LegislationTypeUKPGA,
		Year:            "2018",
		Number:          "12",
	}

	metadata, err := ukLegClient.FetchMetadata(legislationURI)
	if err != nil {
		t.Fatalf("FetchMetadata failed: %v", err)
	}

	if metadata.LegislationType != "ukpga" {
		t.Errorf("LegislationType: got %q, want 'ukpga'", metadata.LegislationType)
	}
	if metadata.Year != "2018" {
		t.Errorf("Year: got %q, want '2018'", metadata.Year)
	}
	if metadata.Number != "12" {
		t.Errorf("Number: got %q, want '12'", metadata.Number)
	}
	if metadata.URI != "https://www.legislation.gov.uk/ukpga/2018/12" {
		t.Errorf("URI: got %q, want legislation.gov.uk URI", metadata.URI)
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

	ukLegClient := newTestClient(mockClient)
	legislationURI := &LegislationURI{
		LegislationType: LegislationTypeUKPGA,
		Year:            "2099",
		Number:          "999",
	}

	_, err := ukLegClient.FetchMetadata(legislationURI)
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

	ukLegClient := newTestClient(mockClient)
	legislationURI := &LegislationURI{
		LegislationType: LegislationTypeUKPGA,
		Year:            "2018",
		Number:          "12",
	}

	_, err := ukLegClient.FetchMetadata(legislationURI)
	if err == nil {
		t.Error("Expected error for network failure")
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

	ukLegClient := newTestClient(mockClient)
	legislationURI := &LegislationURI{
		LegislationType: LegislationTypeUKPGA,
		Year:            "2018",
		Number:          "12",
	}

	_, err := ukLegClient.FetchMetadata(legislationURI)
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

	ukLegClient := newTestClient(mockClient)
	legislationURI := &LegislationURI{
		LegislationType: LegislationTypeUKSI,
		Year:            "2019",
		Number:          "419",
	}

	_, err := ukLegClient.FetchMetadata(legislationURI)
	if err != nil {
		t.Fatalf("FetchMetadata failed: %v", err)
	}

	expectedURL := "https://www.legislation.gov.uk/uksi/2019/419"
	if capturedURL != expectedURL {
		t.Errorf("Request URL: got %q, want %q", capturedURL, expectedURL)
	}
}

func TestFetchMetadata_AcceptHeader(t *testing.T) {
	var capturedAccept string

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			capturedAccept = req.Header.Get("Accept")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		},
	}

	ukLegClient := newTestClient(mockClient)
	legislationURI := &LegislationURI{
		LegislationType: LegislationTypeUKPGA,
		Year:            "2018",
		Number:          "12",
	}

	_, err := ukLegClient.FetchMetadata(legislationURI)
	if err != nil {
		t.Fatalf("FetchMetadata failed: %v", err)
	}

	if capturedAccept != "application/xml" {
		t.Errorf("Accept header: got %q, want 'application/xml'", capturedAccept)
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

	ukLegClient := newTestClient(mockClient)
	_, err := ukLegClient.ValidateURI("https://www.legislation.gov.uk/ukpga/2018/12")
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
			if req.Method == http.MethodHead {
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

	ukLegClient := newTestClient(mockClient)

	// ValidateURI should use HEAD.
	_, err := ukLegClient.ValidateURI("https://www.legislation.gov.uk/ukpga/2018/12")
	if err != nil {
		t.Fatalf("ValidateURI failed: %v", err)
	}
	if validateMethod != http.MethodHead {
		t.Errorf("ValidateURI method: got %q, want %q", validateMethod, http.MethodHead)
	}

	// FetchMetadata should use GET.
	legislationURI := &LegislationURI{
		LegislationType: LegislationTypeUKPGA,
		Year:            "2018",
		Number:          "12",
	}
	_, err = ukLegClient.FetchMetadata(legislationURI)
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

func TestNewUKLegClient_DefaultHTTPClient(t *testing.T) {
	config := DefaultConfig()
	ukLegClient := NewUKLegClient(config)

	if ukLegClient.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}
	if ukLegClient.cache == nil {
		t.Error("Expected cache to be initialized")
	}
	if ukLegClient.userAgent != DefaultUserAgent {
		t.Errorf("UserAgent: got %q, want %q", ukLegClient.userAgent, DefaultUserAgent)
	}
}

// Integration tests — skipped during short runs.
// These make real HTTP requests to legislation.gov.uk.

func TestIntegration_ValidateRealURI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := DefaultConfig()
	ukLegClient := NewUKLegClient(config)

	cases := []struct {
		name        string
		uri         string
		expectValid bool
	}{
		{
			name:        "Data Protection Act 2018",
			uri:         "https://www.legislation.gov.uk/ukpga/2018/12",
			expectValid: true,
		},
		{
			name:        "Human Rights Act 1998",
			uri:         "https://www.legislation.gov.uk/ukpga/1998/42",
			expectValid: true,
		},
		{
			name:        "GDPR UK SI 2019/419",
			uri:         "https://www.legislation.gov.uk/uksi/2019/419",
			expectValid: true,
		},
		{
			name:        "Nonexistent legislation",
			uri:         "https://www.legislation.gov.uk/ukpga/2099/9999",
			expectValid: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			validationResult, err := ukLegClient.ValidateURI(tc.uri)
			if err != nil {
				t.Fatalf("ValidateURI failed: %v", err)
			}

			if validationResult.Valid != tc.expectValid {
				t.Errorf("Valid: got %v, want %v (status: %d, error: %s)",
					validationResult.Valid, tc.expectValid,
					validationResult.StatusCode, validationResult.Error)
			}
		})
	}
}

func TestIntegration_ParseAndValidate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	oscolaParser := citation.NewOSCOLAParser()
	config := DefaultConfig()
	ukLegClient := NewUKLegClient(config)

	// Parse a text containing UK legislation citations.
	text := "The Data Protection Act 2018 [2018 c. 12] implements the GDPR in UK law."
	citations, err := oscolaParser.Parse(text)
	if err != nil {
		t.Fatalf("OSCOLA parser failed: %v", err)
	}

	// Find the act chapter citation (which has DocNumber set).
	var actCitation *citation.Citation
	for _, citationRef := range citations {
		if citationRef.Components.CodeName == "ukact" && citationRef.Components.DocNumber != "" {
			actCitation = citationRef
			break
		}
	}

	if actCitation == nil {
		t.Fatal("Expected to find a UK act citation with chapter number")
	}

	// Validate the citation against legislation.gov.uk.
	validationResult, err := ukLegClient.ValidateCitation(actCitation)
	if err != nil {
		t.Fatalf("ValidateCitation failed: %v", err)
	}

	if !validationResult.Valid {
		t.Errorf("Expected Data Protection Act 2018 to validate (status: %d, error: %s)",
			validationResult.StatusCode, validationResult.Error)
	}
}
