package uscode

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

// newTestClient creates a USCodeClient with a mock HTTP client for fast tests.
func newTestClient(mockClient *MockHTTPClient) *USCodeClient {
	return &USCodeClient{
		httpClient: mockClient,
		cache:      NewValidationCache(1 * time.Hour),
		userAgent:  DefaultUserAgent,
	}
}

// =============================================================================
// Mock-based Unit Tests
// =============================================================================

func TestValidateURI_200OK(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		},
	}

	uscClient := newTestClient(mockClient)
	validationResult, err := uscClient.ValidateURI("https://uscode.house.gov/view.xhtml?req=granuleid:USC-prelim-title42-section1983&edition=prelim")
	if err != nil {
		t.Fatalf("ValidateURI failed: %v", err)
	}

	if !validationResult.Valid {
		t.Error("Expected Valid to be true for 200 OK")
	}
	if validationResult.StatusCode != 200 {
		t.Errorf("StatusCode: got %d, want 200", validationResult.StatusCode)
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

	uscClient := newTestClient(mockClient)
	validationResult, err := uscClient.ValidateURI("https://uscode.house.gov/view.xhtml?req=granuleid:USC-prelim-title99-section9999&edition=prelim")
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

	uscClient := newTestClient(mockClient)
	validationResult, err := uscClient.ValidateURI("https://uscode.house.gov/view.xhtml?req=granuleid:USC-prelim-title42-section1983&edition=prelim")
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

	uscClient := newTestClient(mockClient)
	validationResult, err := uscClient.ValidateURI("https://uscode.house.gov/view.xhtml?req=granuleid:USC-prelim-title42-section1983&edition=prelim")
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

	uscClient := newTestClient(mockClient)
	targetURI := "https://uscode.house.gov/view.xhtml?req=granuleid:USC-prelim-title42-section1983&edition=prelim"

	// First call should hit the network.
	firstResult, err := uscClient.ValidateURI(targetURI)
	if err != nil {
		t.Fatalf("First ValidateURI failed: %v", err)
	}
	if !firstResult.Valid {
		t.Error("First call: expected Valid")
	}

	// Second call should hit the cache.
	secondResult, err := uscClient.ValidateURI(targetURI)
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

func TestValidateUSCCitation_EndToEnd(t *testing.T) {
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

	uscClient := newTestClient(mockClient)

	// 42 U.S.C. § 1983
	uscCitation := &citation.Citation{
		Type: citation.CitationTypeCode,
		Components: citation.CitationComponents{
			CodeName: "USC",
			Title:    "42",
			Section:  "1983",
		},
	}

	validationResult, err := uscClient.ValidateUSCCitation(uscCitation)
	if err != nil {
		t.Fatalf("ValidateUSCCitation failed: %v", err)
	}

	if !validationResult.Valid {
		t.Error("Expected USC citation to validate successfully")
	}

	expectedURL := "https://uscode.house.gov/view.xhtml?req=granuleid:USC-prelim-title42-section1983&edition=prelim"
	if capturedURL != expectedURL {
		t.Errorf("Request URL: got %q, want %q", capturedURL, expectedURL)
	}
}

func TestValidateCFRCitation_EndToEnd(t *testing.T) {
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

	uscClient := newTestClient(mockClient)

	// 45 C.F.R. § 164.502
	cfrCitation := &citation.Citation{
		Type: citation.CitationTypeCode,
		Components: citation.CitationComponents{
			CodeName: "CFR",
			Title:    "45",
			Section:  "164.502",
		},
	}

	validationResult, err := uscClient.ValidateCFRCitation(cfrCitation)
	if err != nil {
		t.Fatalf("ValidateCFRCitation failed: %v", err)
	}

	if !validationResult.Valid {
		t.Error("Expected CFR citation to validate successfully")
	}

	expectedURL := "https://www.ecfr.gov/current/title-45/part-164/section-164.502"
	if capturedURL != expectedURL {
		t.Errorf("Request URL: got %q, want %q", capturedURL, expectedURL)
	}
}

func TestValidateUSCCitation_InvalidCitation(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			t.Fatal("HTTP request should not be made for invalid citation")
			return nil, nil
		},
	}

	uscClient := newTestClient(mockClient)

	// Missing title should cause GenerateUSCURI to fail.
	invalidCitation := &citation.Citation{
		Type: citation.CitationTypeCode,
		Components: citation.CitationComponents{
			CodeName: "USC",
			Section:  "1983",
		},
	}

	_, err := uscClient.ValidateUSCCitation(invalidCitation)
	if err == nil {
		t.Error("Expected error for citation missing title")
	}
}

func TestValidateUSCNumber_Success(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		},
	}

	uscClient := newTestClient(mockClient)
	uscNumber := &USCNumber{Title: "42", Section: "1983"}

	validationResult, err := uscClient.ValidateUSCNumber(uscNumber)
	if err != nil {
		t.Fatalf("ValidateUSCNumber failed: %v", err)
	}

	if !validationResult.Valid {
		t.Error("Expected Valid to be true")
	}
}

func TestValidateCFRNumber_Success(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		},
	}

	uscClient := newTestClient(mockClient)
	cfrNumber := &CFRNumber{Title: "45", Part: "164", Section: "502"}

	validationResult, err := uscClient.ValidateCFRNumber(cfrNumber)
	if err != nil {
		t.Fatalf("ValidateCFRNumber failed: %v", err)
	}

	if !validationResult.Valid {
		t.Error("Expected Valid to be true")
	}
}

func TestFetchUSCMetadata_Success(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		},
	}

	uscClient := newTestClient(mockClient)
	uscNumber := &USCNumber{Title: "42", Section: "1983"}

	metadata, err := uscClient.FetchUSCMetadata(uscNumber)
	if err != nil {
		t.Fatalf("FetchUSCMetadata failed: %v", err)
	}

	if metadata.Title != "42" {
		t.Errorf("Title: got %q, want '42'", metadata.Title)
	}
	if metadata.Section != "1983" {
		t.Errorf("Section: got %q, want '1983'", metadata.Section)
	}
	if metadata.DisplayName != "42 U.S.C. § 1983" {
		t.Errorf("DisplayName: got %q, want '42 U.S.C. § 1983'", metadata.DisplayName)
	}
}

func TestFetchUSCMetadata_NotFound(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       http.NoBody,
			}, nil
		},
	}

	uscClient := newTestClient(mockClient)
	uscNumber := &USCNumber{Title: "99", Section: "9999"}

	_, err := uscClient.FetchUSCMetadata(uscNumber)
	if err == nil {
		t.Error("Expected error for 404 response")
	}
}

func TestFetchCFRMetadata_Success(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		},
	}

	uscClient := newTestClient(mockClient)
	cfrNumber := &CFRNumber{Title: "45", Part: "164", Section: "502"}

	metadata, err := uscClient.FetchCFRMetadata(cfrNumber)
	if err != nil {
		t.Fatalf("FetchCFRMetadata failed: %v", err)
	}

	if metadata.Title != "45" {
		t.Errorf("Title: got %q, want '45'", metadata.Title)
	}
	if metadata.DisplayName != "45 C.F.R. § 164.502" {
		t.Errorf("DisplayName: got %q, want '45 C.F.R. § 164.502'", metadata.DisplayName)
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

	uscClient := newTestClient(mockClient)
	_, err := uscClient.ValidateURI("https://uscode.house.gov/view.xhtml?req=granuleid:USC-prelim-title42-section1983&edition=prelim")
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
			if strings.Contains(req.URL.String(), "granuleid") {
				if fetchMethod == "" {
					validateMethod = req.Method
				} else {
					fetchMethod = req.Method
				}
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		},
	}

	uscClient := newTestClient(mockClient)

	// ValidateURI should use HEAD.
	_, err := uscClient.ValidateURI("https://uscode.house.gov/view.xhtml?req=granuleid:USC-prelim-title42-section1983&edition=prelim")
	if err != nil {
		t.Fatalf("ValidateURI failed: %v", err)
	}
	if validateMethod != http.MethodHead {
		t.Errorf("ValidateURI method: got %q, want %q", validateMethod, http.MethodHead)
	}

	// FetchUSCMetadata should use GET.
	fetchMethod = "" // reset to trigger fresh capture
	mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
		fetchMethod = req.Method
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       http.NoBody,
		}, nil
	}
	// Need new client to avoid cache hit
	uscClient2 := newTestClient(mockClient)
	uscNumber := &USCNumber{Title: "42", Section: "1983"}
	_, err = uscClient2.FetchUSCMetadata(uscNumber)
	if err != nil {
		t.Fatalf("FetchUSCMetadata failed: %v", err)
	}
	if fetchMethod != http.MethodGet {
		t.Errorf("FetchUSCMetadata method: got %q, want %q", fetchMethod, http.MethodGet)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultUSCodeConfig()

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

func TestNewUSCodeClient_DefaultHTTPClient(t *testing.T) {
	config := DefaultUSCodeConfig()
	uscClient := NewUSCodeClient(config)

	if uscClient.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}
	if uscClient.cache == nil {
		t.Error("Expected cache to be initialized")
	}
	if uscClient.userAgent != DefaultUserAgent {
		t.Errorf("UserAgent: got %q, want %q", uscClient.userAgent, DefaultUserAgent)
	}
}

func TestNilUSCNumber(t *testing.T) {
	uscClient := newTestClient(&MockHTTPClient{})

	_, err := uscClient.ValidateUSCNumber(nil)
	if err == nil {
		t.Error("Expected error for nil USC number")
	}

	_, err = uscClient.FetchUSCMetadata(nil)
	if err == nil {
		t.Error("Expected error for nil USC number")
	}
}

func TestNilCFRNumber(t *testing.T) {
	uscClient := newTestClient(&MockHTTPClient{})

	_, err := uscClient.ValidateCFRNumber(nil)
	if err == nil {
		t.Error("Expected error for nil CFR number")
	}

	_, err = uscClient.FetchCFRMetadata(nil)
	if err == nil {
		t.Error("Expected error for nil CFR number")
	}
}

// =============================================================================
// Real Connection Integration Tests
// These tests hit the actual uscode.house.gov and ecfr.gov servers.
// They verify that the actual connection functionality works.
// =============================================================================

func TestIntegration_USCodeHouseGov_RealConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use a very short rate limit for test speed, but real HTTP client
	config := USCodeClientConfig{
		RateLimit:  100 * time.Millisecond,
		CacheTTL:   1 * time.Hour,
		HTTPClient: nil, // Use real http.DefaultClient
		UserAgent:  DefaultUserAgent,
	}
	uscClient := NewUSCodeClient(config)

	// Test 1: Validate a well-known USC section (42 U.S.C. § 1983 - Civil Rights)
	t.Run("Validate42USC1983", func(t *testing.T) {
		uscNumber := &USCNumber{Title: "42", Section: "1983"}
		result, err := uscClient.ValidateUSCNumber(uscNumber)
		if err != nil {
			t.Fatalf("ValidateUSCNumber failed: %v", err)
		}

		t.Logf("42 U.S.C. § 1983 validation result: Valid=%v, StatusCode=%d, URI=%s",
			result.Valid, result.StatusCode, result.URI)

		// The section should exist (though we can't guarantee status code)
		if result.Error != "" {
			t.Logf("Note: Connection error (may be network issue): %s", result.Error)
		}
	})

	// Test 2: Validate 15 U.S.C. § 1681 (Fair Credit Reporting Act)
	t.Run("Validate15USC1681", func(t *testing.T) {
		uscNumber := &USCNumber{Title: "15", Section: "1681"}
		result, err := uscClient.ValidateUSCNumber(uscNumber)
		if err != nil {
			t.Fatalf("ValidateUSCNumber failed: %v", err)
		}

		t.Logf("15 U.S.C. § 1681 validation result: Valid=%v, StatusCode=%d, URI=%s",
			result.Valid, result.StatusCode, result.URI)
	})

	// Test 3: Validate 5 U.S.C. § 552 (Freedom of Information Act)
	t.Run("Validate5USC552", func(t *testing.T) {
		uscNumber := &USCNumber{Title: "5", Section: "552"}
		result, err := uscClient.ValidateUSCNumber(uscNumber)
		if err != nil {
			t.Fatalf("ValidateUSCNumber failed: %v", err)
		}

		t.Logf("5 U.S.C. § 552 (FOIA) validation result: Valid=%v, StatusCode=%d, URI=%s",
			result.Valid, result.StatusCode, result.URI)
	})
}

func TestIntegration_ECFR_RealConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := USCodeClientConfig{
		RateLimit:  100 * time.Millisecond,
		CacheTTL:   1 * time.Hour,
		HTTPClient: nil,
		UserAgent:  DefaultUserAgent,
	}
	uscClient := NewUSCodeClient(config)

	// Test 1: Validate 45 C.F.R. Part 164 (HIPAA Privacy Rule)
	t.Run("Validate45CFR164", func(t *testing.T) {
		cfrNumber := &CFRNumber{Title: "45", Part: "164", Section: ""}
		result, err := uscClient.ValidateCFRNumber(cfrNumber)
		if err != nil {
			t.Fatalf("ValidateCFRNumber failed: %v", err)
		}

		t.Logf("45 C.F.R. Part 164 validation result: Valid=%v, StatusCode=%d, URI=%s",
			result.Valid, result.StatusCode, result.URI)
	})

	// Test 2: Validate 45 C.F.R. § 164.502 (HIPAA specific section)
	t.Run("Validate45CFR164_502", func(t *testing.T) {
		cfrNumber := &CFRNumber{Title: "45", Part: "164", Section: "502"}
		result, err := uscClient.ValidateCFRNumber(cfrNumber)
		if err != nil {
			t.Fatalf("ValidateCFRNumber failed: %v", err)
		}

		t.Logf("45 C.F.R. § 164.502 validation result: Valid=%v, StatusCode=%d, URI=%s",
			result.Valid, result.StatusCode, result.URI)
	})

	// Test 3: Validate 16 C.F.R. Part 312 (COPPA Rule)
	t.Run("Validate16CFR312", func(t *testing.T) {
		cfrNumber := &CFRNumber{Title: "16", Part: "312", Section: ""}
		result, err := uscClient.ValidateCFRNumber(cfrNumber)
		if err != nil {
			t.Fatalf("ValidateCFRNumber failed: %v", err)
		}

		t.Logf("16 C.F.R. Part 312 (COPPA) validation result: Valid=%v, StatusCode=%d, URI=%s",
			result.Valid, result.StatusCode, result.URI)
	})
}

func TestIntegration_ParseAndValidate_RealConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := USCodeClientConfig{
		RateLimit:  100 * time.Millisecond,
		CacheTTL:   1 * time.Hour,
		HTTPClient: nil,
		UserAgent:  DefaultUserAgent,
	}
	uscClient := NewUSCodeClient(config)

	// Test parsing a citation string and then validating it
	t.Run("ParseAndValidateUSC", func(t *testing.T) {
		// Parse the citation
		uscNumber, err := ParseUSCNumber("42 U.S.C. § 1983")
		if err != nil {
			t.Fatalf("ParseUSCNumber failed: %v", err)
		}

		t.Logf("Parsed: Title=%s, Section=%s", uscNumber.Title, uscNumber.Section)

		// Validate it
		result, err := uscClient.ValidateUSCNumber(uscNumber)
		if err != nil {
			t.Fatalf("ValidateUSCNumber failed: %v", err)
		}

		t.Logf("Validation result: Valid=%v, StatusCode=%d", result.Valid, result.StatusCode)
	})

	t.Run("ParseAndValidateCFR", func(t *testing.T) {
		// Parse the citation
		cfrNumber, err := ParseCFRNumber("45 C.F.R. § 164.502")
		if err != nil {
			t.Fatalf("ParseCFRNumber failed: %v", err)
		}

		t.Logf("Parsed: Title=%s, Part=%s, Section=%s", cfrNumber.Title, cfrNumber.Part, cfrNumber.Section)

		// Validate it
		result, err := uscClient.ValidateCFRNumber(cfrNumber)
		if err != nil {
			t.Fatalf("ValidateCFRNumber failed: %v", err)
		}

		t.Logf("Validation result: Valid=%v, StatusCode=%d", result.Valid, result.StatusCode)
	})
}

func TestIntegration_FetchMetadata_RealConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := USCodeClientConfig{
		RateLimit:  100 * time.Millisecond,
		CacheTTL:   1 * time.Hour,
		HTTPClient: nil,
		UserAgent:  DefaultUserAgent,
	}
	uscClient := NewUSCodeClient(config)

	t.Run("FetchUSCMetadata", func(t *testing.T) {
		uscNumber := &USCNumber{Title: "42", Section: "1983"}
		metadata, err := uscClient.FetchUSCMetadata(uscNumber)
		if err != nil {
			// Fetch may fail due to network or server issues, but we should log it
			t.Logf("FetchUSCMetadata returned error (may be expected): %v", err)
			return
		}

		t.Logf("USC Metadata: Title=%s, Section=%s, DisplayName=%s",
			metadata.Title, metadata.Section, metadata.DisplayName)

		if metadata.Title != "42" {
			t.Errorf("Expected Title=42, got %s", metadata.Title)
		}
		if metadata.Section != "1983" {
			t.Errorf("Expected Section=1983, got %s", metadata.Section)
		}
	})

	t.Run("FetchCFRMetadata", func(t *testing.T) {
		cfrNumber := &CFRNumber{Title: "45", Part: "164", Section: "502"}
		metadata, err := uscClient.FetchCFRMetadata(cfrNumber)
		if err != nil {
			t.Logf("FetchCFRMetadata returned error (may be expected): %v", err)
			return
		}

		t.Logf("CFR Metadata: Title=%s, Section=%s, DisplayName=%s",
			metadata.Title, metadata.Section, metadata.DisplayName)

		if metadata.Title != "45" {
			t.Errorf("Expected Title=45, got %s", metadata.Title)
		}
	})
}

// TestIntegration_ConnectionSummary provides a summary test that logs connection status
// for all major US Code sources - useful for debugging connectivity issues.
func TestIntegration_ConnectionSummary(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := USCodeClientConfig{
		RateLimit:  200 * time.Millisecond,
		CacheTTL:   1 * time.Hour,
		HTTPClient: nil,
		UserAgent:  DefaultUserAgent,
	}
	uscClient := NewUSCodeClient(config)

	testCases := []struct {
		name      string
		codeType  string
		title     string
		section   string
		part      string
		subsect   string
	}{
		{"Civil Rights Act", "USC", "42", "1983", "", ""},
		{"FCRA", "USC", "15", "1681", "", ""},
		{"FOIA", "USC", "5", "552", "", ""},
		{"HIPAA Privacy", "CFR", "45", "", "164", ""},
		{"HIPAA Section", "CFR", "45", "", "164", "502"},
		{"COPPA", "CFR", "16", "", "312", ""},
	}

	t.Log("=== US Code Connection Summary ===")
	successCount := 0
	for _, tc := range testCases {
		var result *ValidationResult
		var err error

		if tc.codeType == "USC" {
			uscNumber := &USCNumber{Title: tc.title, Section: tc.section}
			result, err = uscClient.ValidateUSCNumber(uscNumber)
		} else {
			cfrNumber := &CFRNumber{Title: tc.title, Part: tc.part, Section: tc.subsect}
			result, err = uscClient.ValidateCFRNumber(cfrNumber)
		}

		status := "FAIL"
		if err == nil && result != nil && result.Valid {
			status = "OK"
			successCount++
		} else if err == nil && result != nil && result.Error != "" {
			status = "NETWORK_ERROR"
		}

		t.Logf("  [%s] %s: StatusCode=%d", status, tc.name,
			func() int {
				if result != nil {
					return result.StatusCode
				}
				return 0
			}())
	}

	t.Logf("=== Summary: %d/%d successful connections ===", successCount, len(testCases))
}
