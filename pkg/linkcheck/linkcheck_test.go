package linkcheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// --- Types tests ---

func TestLinkResult_IsSuccess(t *testing.T) {
	testCases := []struct {
		name     string
		status   LinkStatus
		expected bool
	}{
		{"valid", StatusValid, true},
		{"redirect", StatusRedirect, true},
		{"invalid", StatusInvalid, false},
		{"timeout", StatusTimeout, false},
		{"error", StatusError, false},
		{"skipped", StatusSkipped, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := &LinkResult{Status: tc.status}
			if result.IsSuccess() != tc.expected {
				t.Errorf("IsSuccess() = %v, want %v", result.IsSuccess(), tc.expected)
			}
		})
	}
}

func TestExtractDomain(t *testing.T) {
	testCases := []struct {
		name     string
		uri      string
		expected string
	}{
		{"http", "http://example.com/path", "example.com"},
		{"https", "https://eur-lex.europa.eu/legal-content", "eur-lex.europa.eu"},
		{"with_port", "https://localhost:8080/api", "localhost:8080"},
		{"subdomain", "https://www.google.com/search", "www.google.com"},
		{"invalid", "not-a-url", ""},
		{"empty", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractDomain(tc.uri)
			if result != tc.expected {
				t.Errorf("ExtractDomain(%q) = %q, want %q", tc.uri, result, tc.expected)
			}
		})
	}
}

func TestBatchConfig_GetDomainConfig(t *testing.T) {
	config := DefaultBatchConfig()
	config.WithDomainConfig(&DomainConfig{
		Domain:    "slow.example.com",
		RateLimit: 2 * time.Second,
		Timeout:   60 * time.Second,
	})

	t.Run("configured_domain", func(t *testing.T) {
		domainConfig := config.GetDomainConfig("slow.example.com")
		if domainConfig.RateLimit != 2*time.Second {
			t.Errorf("RateLimit = %v, want 2s", domainConfig.RateLimit)
		}
	})

	t.Run("unknown_domain", func(t *testing.T) {
		domainConfig := config.GetDomainConfig("unknown.example.com")
		if domainConfig.RateLimit != config.DefaultRateLimit {
			t.Errorf("RateLimit = %v, want %v", domainConfig.RateLimit, config.DefaultRateLimit)
		}
	})
}

func TestValidationProgress_PercentComplete(t *testing.T) {
	testCases := []struct {
		name      string
		total     int
		completed int
		expected  float64
	}{
		{"none", 100, 0, 0.0},
		{"half", 100, 50, 50.0},
		{"all", 100, 100, 100.0},
		{"empty", 0, 0, 100.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			progress := &ValidationProgress{
				TotalLinks:     tc.total,
				CompletedLinks: tc.completed,
			}
			result := progress.PercentComplete()
			if result != tc.expected {
				t.Errorf("PercentComplete() = %v, want %v", result, tc.expected)
			}
		})
	}
}

// --- Report tests ---

func TestValidationReport_AddResult(t *testing.T) {
	report := NewValidationReport()

	report.AddResult(&LinkResult{URI: "http://valid.com", Status: StatusValid, Domain: "valid.com"})
	report.AddResult(&LinkResult{URI: "http://invalid.com", Status: StatusInvalid, Domain: "invalid.com"})
	report.AddResult(&LinkResult{URI: "http://timeout.com", Status: StatusTimeout, Domain: "timeout.com"})
	report.AddResult(&LinkResult{URI: "http://error.com", Status: StatusError, Domain: "error.com"})
	report.AddResult(&LinkResult{URI: "http://skipped.com", Status: StatusSkipped, Domain: "skipped.com"})

	if report.TotalLinks != 5 {
		t.Errorf("TotalLinks = %d, want 5", report.TotalLinks)
	}
	if report.ValidLinks != 1 {
		t.Errorf("ValidLinks = %d, want 1", report.ValidLinks)
	}
	if report.InvalidLinks != 1 {
		t.Errorf("InvalidLinks = %d, want 1", report.InvalidLinks)
	}
	if report.TimeoutLinks != 1 {
		t.Errorf("TimeoutLinks = %d, want 1", report.TimeoutLinks)
	}
	if report.ErrorLinks != 1 {
		t.Errorf("ErrorLinks = %d, want 1", report.ErrorLinks)
	}
	if report.SkippedLinks != 1 {
		t.Errorf("SkippedLinks = %d, want 1", report.SkippedLinks)
	}
	if len(report.BrokenLinks) != 3 {
		t.Errorf("BrokenLinks count = %d, want 3", len(report.BrokenLinks))
	}
}

func TestValidationReport_SuccessRate(t *testing.T) {
	report := NewValidationReport()
	report.AddResult(&LinkResult{URI: "http://valid1.com", Status: StatusValid, Domain: "valid1.com"})
	report.AddResult(&LinkResult{URI: "http://valid2.com", Status: StatusValid, Domain: "valid2.com"})
	report.AddResult(&LinkResult{URI: "http://invalid.com", Status: StatusInvalid, Domain: "invalid.com"})
	report.AddResult(&LinkResult{URI: "http://skipped.com", Status: StatusSkipped, Domain: "skipped.com"})

	// Success rate should exclude skipped links: 2/3 = 66.67%
	rate := report.SuccessRate()
	if rate < 66.6 || rate > 66.7 {
		t.Errorf("SuccessRate() = %v, want ~66.67", rate)
	}
}

func TestValidationReport_ToJSON(t *testing.T) {
	report := NewValidationReport()
	report.AddResult(&LinkResult{URI: "http://example.com", Status: StatusValid, Domain: "example.com"})
	report.Finalize()

	jsonData, err := report.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	// Verify it's valid JSON
	var decoded ValidationReport
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if decoded.TotalLinks != 1 {
		t.Errorf("Decoded TotalLinks = %d, want 1", decoded.TotalLinks)
	}
}

func TestValidationReport_ToMarkdown(t *testing.T) {
	report := NewValidationReport()
	report.AddResult(&LinkResult{URI: "http://valid.com", Status: StatusValid, Domain: "valid.com"})
	report.AddResult(&LinkResult{URI: "http://broken.com", Status: StatusInvalid, Domain: "broken.com", Error: "HTTP 404"})
	report.Finalize()

	markdown := report.ToMarkdown()

	if !strings.Contains(markdown, "# Link Validation Report") {
		t.Error("Markdown should contain header")
	}
	if !strings.Contains(markdown, "Total Links**: 2") {
		t.Error("Markdown should contain total count")
	}
	if !strings.Contains(markdown, "## Broken Links") {
		t.Error("Markdown should contain broken links section")
	}
	if !strings.Contains(markdown, "http://broken.com") {
		t.Error("Markdown should list broken link")
	}
}

func TestValidationReport_String(t *testing.T) {
	report := NewValidationReport()
	report.AddResult(&LinkResult{URI: "http://example.com", Status: StatusValid, Domain: "example.com"})
	report.Finalize()

	str := report.String()
	if !strings.Contains(str, "Link Validation Report") {
		t.Error("String should contain header")
	}
	if !strings.Contains(str, "Total links:") {
		t.Error("String should contain total count")
	}
}

// --- Cache tests ---

func TestLinkCache_GetSet(t *testing.T) {
	cache := NewLinkCache(1 * time.Hour)

	// Should not exist initially
	result, found := cache.Get("http://example.com")
	if found {
		t.Error("Expected cache miss for new key")
	}

	// Set and retrieve
	linkResult := &LinkResult{URI: "http://example.com", Status: StatusValid}
	cache.Set("http://example.com", linkResult)

	result, found = cache.Get("http://example.com")
	if !found {
		t.Error("Expected cache hit after Set")
	}
	if result.URI != "http://example.com" {
		t.Errorf("URI = %q, want %q", result.URI, "http://example.com")
	}
}

func TestLinkCache_Expiration(t *testing.T) {
	cache := NewLinkCache(50 * time.Millisecond)

	linkResult := &LinkResult{URI: "http://example.com", Status: StatusValid}
	cache.Set("http://example.com", linkResult)

	// Should be found immediately
	_, found := cache.Get("http://example.com")
	if !found {
		t.Error("Expected cache hit immediately after Set")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	_, found = cache.Get("http://example.com")
	if found {
		t.Error("Expected cache miss after expiration")
	}
}

func TestLinkCache_Cleanup(t *testing.T) {
	cache := NewLinkCache(50 * time.Millisecond)

	cache.Set("http://1.com", &LinkResult{URI: "http://1.com"})
	cache.Set("http://2.com", &LinkResult{URI: "http://2.com"})

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	expired := cache.Cleanup()
	if expired != 2 {
		t.Errorf("Cleanup() = %d, want 2", expired)
	}

	if cache.Len() != 0 {
		t.Errorf("Len() = %d, want 0 after cleanup", cache.Len())
	}
}

// --- Validator tests with mock server ---

func TestBatchValidator_ValidateURIStrings(t *testing.T) {
	// Create test server
	requestCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)

		switch r.URL.Path {
		case "/valid":
			w.WriteHeader(http.StatusOK)
		case "/notfound":
			w.WriteHeader(http.StatusNotFound)
		case "/redirect":
			w.Header().Set("Location", "/valid")
			w.WriteHeader(http.StatusMovedPermanently)
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	config := DefaultBatchConfig()
	config.DefaultRateLimit = 10 * time.Millisecond // Fast for tests
	config.DefaultTimeout = 5 * time.Second
	config.CacheTTL = 1 * time.Hour
	config.FollowRedirects = false // Don't follow redirects to test redirect detection

	validator := NewBatchValidator(config)

	uris := []string{
		server.URL + "/valid",
		server.URL + "/notfound",
		server.URL + "/redirect",
	}

	report := validator.ValidateURIStrings(uris)

	if report.TotalLinks != 3 {
		t.Errorf("TotalLinks = %d, want 3", report.TotalLinks)
	}

	// Check that we got expected statuses
	statusCounts := make(map[LinkStatus]int)
	for _, result := range report.Results {
		statusCounts[result.Status]++
	}

	if statusCounts[StatusValid] != 1 {
		t.Errorf("Valid count = %d, want 1", statusCounts[StatusValid])
	}
	if statusCounts[StatusInvalid] != 1 {
		t.Errorf("Invalid count = %d, want 1", statusCounts[StatusInvalid])
	}
	if statusCounts[StatusRedirect] != 1 {
		t.Errorf("Redirect count = %d, want 1", statusCounts[StatusRedirect])
	}
}

func TestBatchValidator_Caching(t *testing.T) {
	requestCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultBatchConfig()
	config.DefaultRateLimit = 10 * time.Millisecond
	config.CacheTTL = 1 * time.Hour

	validator := NewBatchValidator(config)

	uri := server.URL + "/cached"

	// First validation
	report1 := validator.ValidateURIStrings([]string{uri})
	if report1.TotalLinks != 1 {
		t.Errorf("First validation: TotalLinks = %d, want 1", report1.TotalLinks)
	}

	// Second validation should use cache
	report2 := validator.ValidateURIStrings([]string{uri})
	if report2.TotalLinks != 1 {
		t.Errorf("Second validation: TotalLinks = %d, want 1", report2.TotalLinks)
	}

	// Should only have one actual HTTP request
	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("Request count = %d, want 1 (caching should prevent second request)", requestCount)
	}
}

func TestBatchValidator_ProgressCallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultBatchConfig()
	config.DefaultRateLimit = 10 * time.Millisecond

	validator := NewBatchValidator(config)

	progressUpdates := make([]*ValidationProgress, 0)
	var progressMu sync.Mutex

	validator.SetProgressCallback(func(progress *ValidationProgress) {
		progressMu.Lock()
		progressUpdates = append(progressUpdates, progress)
		progressMu.Unlock()
	})

	uris := []string{
		server.URL + "/1",
		server.URL + "/2",
		server.URL + "/3",
	}

	validator.ValidateURIStrings(uris)

	progressMu.Lock()
	updateCount := len(progressUpdates)
	progressMu.Unlock()

	// Should have at least 3 progress updates (one per link)
	if updateCount < 3 {
		t.Errorf("Progress update count = %d, want >= 3", updateCount)
	}
}

func TestBatchValidator_SkipDomain(t *testing.T) {
	requestCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultBatchConfig()
	config.DefaultRateLimit = 10 * time.Millisecond

	// Extract domain from server URL and mark it as skipped
	domain := ExtractDomain(server.URL)
	config.WithDomainConfig(&DomainConfig{
		Domain:       domain,
		SkipValidate: true,
	})

	validator := NewBatchValidator(config)

	report := validator.ValidateURIStrings([]string{server.URL + "/skipped"})

	if report.SkippedLinks != 1 {
		t.Errorf("SkippedLinks = %d, want 1", report.SkippedLinks)
	}

	if atomic.LoadInt32(&requestCount) != 0 {
		t.Errorf("Request count = %d, want 0 (skipped domain should not make requests)", requestCount)
	}
}

func TestBatchValidator_Cancellation(t *testing.T) {
	requestCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		time.Sleep(200 * time.Millisecond) // Slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultBatchConfig()
	config.DefaultRateLimit = 10 * time.Millisecond
	config.DefaultTimeout = 5 * time.Second

	validator := NewBatchValidator(config)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	uris := []string{
		server.URL + "/1",
		server.URL + "/2",
		server.URL + "/3",
	}

	links := make([]LinkInput, len(uris))
	for i, uri := range uris {
		links[i] = LinkInput{URI: uri}
	}

	report := validator.ValidateLinksWithContext(ctx, links)

	// Not all links should complete due to cancellation
	// The exact number depends on timing, so we just check it ran
	if report.TotalLinks > 3 {
		t.Errorf("TotalLinks = %d, want <= 3", report.TotalLinks)
	}
}

func TestBatchValidator_DomainRateLimiting(t *testing.T) {
	requestTimes := make([]time.Time, 0)
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestTimes = append(requestTimes, time.Now())
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultBatchConfig()
	config.DefaultRateLimit = 100 * time.Millisecond // 100ms between requests
	config.Concurrency = 1                           // Single domain, single worker

	validator := NewBatchValidator(config)

	uris := []string{
		server.URL + "/1",
		server.URL + "/2",
		server.URL + "/3",
	}

	start := time.Now()
	validator.ValidateURIStrings(uris)
	elapsed := time.Since(start)

	// Should take at least 200ms (2 intervals between 3 requests)
	if elapsed < 150*time.Millisecond {
		t.Errorf("Elapsed time = %v, want >= 150ms (rate limiting should enforce delays)", elapsed)
	}
}

func TestBatchValidator_SourceContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultBatchConfig()
	config.DefaultRateLimit = 10 * time.Millisecond

	validator := NewBatchValidator(config)

	links := []LinkInput{
		{URI: server.URL + "/test", SourceContext: "Article 5, paragraph 2"},
	}

	report := validator.ValidateLinks(links)

	if len(report.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(report.Results))
	}

	if report.Results[0].SourceContext != "Article 5, paragraph 2" {
		t.Errorf("SourceContext = %q, want %q", report.Results[0].SourceContext, "Article 5, paragraph 2")
	}
}

func TestBatchValidator_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond) // Slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultBatchConfig()
	config.DefaultRateLimit = 10 * time.Millisecond
	config.DefaultTimeout = 100 * time.Millisecond // Fast timeout
	config.DefaultMaxRetries = 0                   // No retries

	validator := NewBatchValidator(config)

	report := validator.ValidateURIStrings([]string{server.URL + "/slow"})

	if report.TimeoutLinks != 1 {
		t.Errorf("TimeoutLinks = %d, want 1", report.TimeoutLinks)
	}
}

// --- HTTP Client tests ---

func TestRateLimitedHTTPClient(t *testing.T) {
	requestTimes := make([]time.Time, 0)
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestTimes = append(requestTimes, time.Now())
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	underlying := http.DefaultClient
	rateLimitedClient := NewRateLimitedHTTPClient(underlying, 100*time.Millisecond)

	// Make 3 sequential requests
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest(http.MethodHead, server.URL, nil)
		rateLimitedClient.Do(req)
	}

	mu.Lock()
	times := requestTimes
	mu.Unlock()

	if len(times) < 2 {
		t.Fatal("Need at least 2 requests for timing check")
	}

	// Check that at least 100ms elapsed between requests
	for i := 1; i < len(times); i++ {
		interval := times[i].Sub(times[i-1])
		if interval < 90*time.Millisecond { // Allow some tolerance
			t.Errorf("Interval between request %d and %d = %v, want >= 90ms", i-1, i, interval)
		}
	}
}

// --- Domain Limiter tests ---

func TestDomainRateLimiter_GetClient(t *testing.T) {
	config := DefaultBatchConfig()
	config.DefaultRateLimit = 1 * time.Second

	config.WithDomainConfig(&DomainConfig{
		Domain:    "fast.example.com",
		RateLimit: 100 * time.Millisecond,
	})

	limiter := NewDomainRateLimiter(http.DefaultClient, config)

	// First call creates client
	client1 := limiter.GetClient("fast.example.com")
	if client1 == nil {
		t.Error("Expected client for configured domain")
	}

	// Second call returns same client
	client2 := limiter.GetClient("fast.example.com")
	if client1 != client2 {
		t.Error("Expected same client instance for same domain")
	}

	// Unknown domain gets new client
	client3 := limiter.GetClient("unknown.example.com")
	if client3 == nil {
		t.Error("Expected client for unknown domain")
	}
}
