package crawler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coolbeans/regula/pkg/library"
)

// setupTestLibrary creates a temporary library for testing.
func setupTestLibrary(t *testing.T) *library.Library {
	t.Helper()
	tempDir := t.TempDir()
	lib, err := library.Init(tempDir, "https://regula.dev/regulations/")
	if err != nil {
		t.Fatalf("failed to init library: %v", err)
	}
	return lib
}

// setupTestServer creates a mock HTTP server that serves legislation content.
func setupTestServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/doc1", func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Content-Type", "text/html")
		responseWriter.WriteHeader(http.StatusOK)
		responseWriter.Write([]byte(`<html><body>
<h1>Title 42 - Section 1320d</h1>
<p>Health Insurance Portability and Accountability Act</p>
<p>As referenced in 15 U.S.C. § 6501 and 45 C.F.R. Part 164.</p>
</body></html>`))
	})

	mux.HandleFunc("/doc2", func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Content-Type", "text/html")
		responseWriter.WriteHeader(http.StatusOK)
		responseWriter.Write([]byte(`<html><body>
<h1>Title 15 - Section 6501</h1>
<p>Children's Online Privacy Protection Act</p>
</body></html>`))
	})

	mux.HandleFunc("/doc3", func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Content-Type", "text/html")
		responseWriter.WriteHeader(http.StatusOK)
		responseWriter.Write([]byte(`<html><body>
<h1>45 CFR Part 164</h1>
<p>Security and Privacy Standards</p>
</body></html>`))
	})

	mux.HandleFunc("/", func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(http.StatusNotFound)
	})

	return httptest.NewServer(mux)
}

func TestCrawlerFromURL(t *testing.T) {
	testServer := setupTestServer()
	defer testServer.Close()

	testLib := setupTestLibrary(t)

	config := CrawlConfig{
		MaxDepth:      1,
		MaxDocuments:  5,
		RateLimit:     10 * time.Millisecond,
		Timeout:       5 * time.Second,
		LibraryPath:   t.TempDir(),
		BaseURI:       "https://regula.dev/regulations/",
		UserAgent:     "test-crawler/1.0",
		DomainConfigs: make(map[string]*DomainConfig),
	}

	crawlerInstance := NewCrawlerWithLibrary(config, testLib)

	seeds := []CrawlSeed{
		{Type: SeedTypeURL, Value: testServer.URL + "/doc1"},
	}

	report, err := crawlerInstance.Crawl(seeds)
	if err != nil {
		t.Fatalf("unexpected crawl error: %v", err)
	}

	if report.TotalIngested < 1 {
		t.Errorf("total ingested = %d, want >= 1", report.TotalIngested)
	}
}

func TestCrawlerDryRun(t *testing.T) {
	testLib := setupTestLibrary(t)

	// Add a seed document with some external references
	sampleSource := []byte("Section 1. This act references 42 U.S.C. § 1320d for health data privacy.")
	addOptions := library.AddOptions{
		Name:      "test-doc",
		ShortName: "test-doc",
		Force:     true,
	}
	_, err := testLib.AddDocument("test-doc", sampleSource, addOptions)
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	config := CrawlConfig{
		MaxDepth:      2,
		MaxDocuments:  10,
		RateLimit:     10 * time.Millisecond,
		Timeout:       5 * time.Second,
		DryRun:        true,
		BaseURI:       "https://regula.dev/regulations/",
		DomainConfigs: make(map[string]*DomainConfig),
	}

	crawlerInstance := NewCrawlerWithLibrary(config, testLib)
	report, err := crawlerInstance.Plan([]CrawlSeed{
		{Type: SeedTypeDocumentID, Value: "test-doc"},
	})
	if err != nil {
		t.Fatalf("unexpected plan error: %v", err)
	}

	if !report.DryRun {
		t.Error("report should indicate dry run")
	}
}

func TestCrawlerDepthLimit(t *testing.T) {
	requestCount := 0
	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		requestCount++
		responseWriter.Header().Set("Content-Type", "text/plain")
		responseWriter.WriteHeader(http.StatusOK)
		responseWriter.Write([]byte(fmt.Sprintf("Document %d content. References 42 U.S.C. § %d.", requestCount, 1000+requestCount)))
	}))
	defer testServer.Close()

	testLib := setupTestLibrary(t)

	config := CrawlConfig{
		MaxDepth:      1,
		MaxDocuments:  20,
		RateLimit:     10 * time.Millisecond,
		Timeout:       5 * time.Second,
		BaseURI:       "https://regula.dev/regulations/",
		DomainConfigs: make(map[string]*DomainConfig),
	}

	crawlerInstance := NewCrawlerWithLibrary(config, testLib)
	seeds := []CrawlSeed{
		{Type: SeedTypeURL, Value: testServer.URL + "/doc"},
	}

	report, err := crawlerInstance.Crawl(seeds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With depth limit 1, should not go deeper than initial + 1 level
	if report.MaxDepthReached > 1 {
		t.Errorf("max depth reached = %d, want <= 1", report.MaxDepthReached)
	}
}

func TestCrawlerDocumentLimit(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Content-Type", "text/plain")
		responseWriter.WriteHeader(http.StatusOK)
		responseWriter.Write([]byte("Some legislation content."))
	}))
	defer testServer.Close()

	testLib := setupTestLibrary(t)

	config := CrawlConfig{
		MaxDepth:      5,
		MaxDocuments:  2,
		RateLimit:     10 * time.Millisecond,
		Timeout:       5 * time.Second,
		BaseURI:       "https://regula.dev/regulations/",
		DomainConfigs: make(map[string]*DomainConfig),
	}

	crawlerInstance := NewCrawlerWithLibrary(config, testLib)
	seeds := []CrawlSeed{
		{Type: SeedTypeURL, Value: testServer.URL + "/doc1"},
		{Type: SeedTypeURL, Value: testServer.URL + "/doc2"},
		{Type: SeedTypeURL, Value: testServer.URL + "/doc3"},
	}

	report, err := crawlerInstance.Crawl(seeds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.TotalIngested > 2 {
		t.Errorf("total ingested = %d, should not exceed max documents (2)", report.TotalIngested)
	}
}

func TestCrawlerDeduplication(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Content-Type", "text/plain")
		responseWriter.WriteHeader(http.StatusOK)
		responseWriter.Write([]byte("Some legislation content."))
	}))
	defer testServer.Close()

	testLib := setupTestLibrary(t)

	config := CrawlConfig{
		MaxDepth:      2,
		MaxDocuments:  10,
		RateLimit:     10 * time.Millisecond,
		Timeout:       5 * time.Second,
		BaseURI:       "https://regula.dev/regulations/",
		DomainConfigs: make(map[string]*DomainConfig),
	}

	crawlerInstance := NewCrawlerWithLibrary(config, testLib)

	// Same URL twice should only ingest once
	sameURL := testServer.URL + "/doc1"
	seeds := []CrawlSeed{
		{Type: SeedTypeURL, Value: sameURL},
		{Type: SeedTypeURL, Value: sameURL},
	}

	report, err := crawlerInstance.Crawl(seeds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.TotalIngested > 1 {
		t.Errorf("total ingested = %d, want 1 (dedup should prevent duplicate)", report.TotalIngested)
	}
}

func TestCrawlerReportFormat(t *testing.T) {
	report := NewCrawlReport(false, []CrawlSeed{{Type: SeedTypeCitation, Value: "42 USC 1320d"}})
	report.RecordItem(&CrawlItem{
		DocumentID: "us-usc-42-1320d",
		Citation:   "42 USC 1320d",
		Depth:      0,
		Status:     CrawlItemIngested,
		Domain:     "uscode.house.gov",
	})
	report.RecordItem(&CrawlItem{
		DocumentID: "us-cfr-45-164",
		Citation:   "45 CFR Part 164",
		Depth:      1,
		Status:     CrawlItemFailed,
		Domain:     "www.ecfr.gov",
		Error:      "fetch failed",
	})

	// Table format
	tableOutput := report.Format("table")
	if !strings.Contains(tableOutput, "Crawl Report") {
		t.Error("table output missing header")
	}
	if !strings.Contains(tableOutput, "us-usc-42-1320d") {
		t.Error("table output missing document ID")
	}

	// JSON format
	jsonOutput := report.Format("json")
	if !strings.Contains(jsonOutput, "total_ingested") {
		t.Error("JSON output missing total_ingested field")
	}
}

func TestCrawlerProvenance(t *testing.T) {
	tracker := NewProvenanceTracker("https://regula.dev/regulations/")

	tracker.RecordDiscovery("us-hipaa", "us-usc-42-1320d", "42 U.S.C. § 1320d", 1)
	tracker.RecordFetch("us-usc-42-1320d", "https://uscode.house.gov/view.xhtml", time.Now())

	if tracker.DiscoveryCount() != 1 {
		t.Errorf("discovery count = %d, want 1", tracker.DiscoveryCount())
	}

	chain := tracker.GetDiscoveryChain("us-usc-42-1320d")
	if len(chain) != 1 || chain[0] != "us-hipaa" {
		t.Errorf("discovery chain = %v, want [us-hipaa]", chain)
	}
}

func TestCrawlerProvenanceFailure(t *testing.T) {
	tracker := NewProvenanceTracker("https://regula.dev/regulations/")

	tracker.RecordFailure("bad citation", "https://example.com/bad", "404 not found")

	tripleStore := tracker.TripleStore()
	if tripleStore.Count() == 0 {
		t.Error("expected failure triples to be recorded")
	}
}

func TestCrawlerFromCitationNotFound(t *testing.T) {
	testLib := setupTestLibrary(t)

	config := CrawlConfig{
		MaxDepth:      1,
		MaxDocuments:  5,
		RateLimit:     10 * time.Millisecond,
		Timeout:       5 * time.Second,
		BaseURI:       "https://regula.dev/regulations/",
		DomainConfigs: make(map[string]*DomainConfig),
	}

	crawlerInstance := NewCrawlerWithLibrary(config, testLib)

	_, err := crawlerInstance.CrawlFromCitation("some random nonsense")
	if err != nil {
		// CrawlFromCitation resolves before crawling, so unrecognized citation is an error
		if !strings.Contains(err.Error(), "unrecognized") {
			t.Errorf("unexpected error: %v", err)
		}
	}
}

func TestCrawlerFromDocumentNotFound(t *testing.T) {
	testLib := setupTestLibrary(t)

	config := CrawlConfig{
		MaxDepth:      1,
		MaxDocuments:  5,
		RateLimit:     10 * time.Millisecond,
		Timeout:       5 * time.Second,
		BaseURI:       "https://regula.dev/regulations/",
		DomainConfigs: make(map[string]*DomainConfig),
	}

	crawlerInstance := NewCrawlerWithLibrary(config, testLib)

	report, err := crawlerInstance.CrawlFromDocument("nonexistent-doc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should report failure for nonexistent document
	if report.TotalFailed < 1 && report.TotalSkipped < 1 {
		t.Error("expected at least one failed/skipped item for nonexistent document")
	}
}
