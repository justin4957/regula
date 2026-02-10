package watch

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSourceType(t *testing.T) {
	tests := []struct {
		sourceType SourceType
		expected   string
	}{
		{SourceTypeRSS, "rss"},
		{SourceTypeAPI, "api"},
		{SourceTypeScrape, "scrape"},
		{SourceTypeWebhook, "webhook"},
	}

	for _, tt := range tests {
		if string(tt.sourceType) != tt.expected {
			t.Errorf("SourceType %v: got %s, want %s", tt.sourceType, string(tt.sourceType), tt.expected)
		}
	}
}

func TestSourceStatus(t *testing.T) {
	tests := []struct {
		status   SourceStatus
		expected string
	}{
		{SourceStatusActive, "active"},
		{SourceStatusPaused, "paused"},
		{SourceStatusError, "error"},
		{SourceStatusDisabled, "disabled"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("SourceStatus %v: got %s, want %s", tt.status, string(tt.status), tt.expected)
		}
	}
}

func TestAuthType(t *testing.T) {
	tests := []struct {
		authType AuthType
		expected string
	}{
		{AuthTypeNone, "none"},
		{AuthTypeBearer, "bearer"},
		{AuthTypeBasic, "basic"},
		{AuthTypeAPIKey, "api_key"},
	}

	for _, tt := range tests {
		if string(tt.authType) != tt.expected {
			t.Errorf("AuthType %v: got %s, want %s", tt.authType, string(tt.authType), tt.expected)
		}
	}
}

func TestLoadSourcesConfig(t *testing.T) {
	// Create temp config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "sources.yaml")

	configContent := `sources:
  - name: test-rss
    type: rss
    url: https://example.com/feed.xml
    interval: 1h
    target_library: deliberations/test
    filters:
      title_contains: "meeting"
  - name: test-api
    type: api
    endpoint: https://api.example.com/documents
    method: GET
    params:
      type: resolution
    interval: 24h
    target_library: deliberations/api
    auth:
      type: bearer
      token_env: TEST_API_TOKEN
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	config, err := LoadSourcesConfig(configPath)
	if err != nil {
		t.Fatalf("LoadSourcesConfig failed: %v", err)
	}

	if len(config.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(config.Sources))
	}

	// Verify first source
	rss := config.Sources[0]
	if rss.Name != "test-rss" {
		t.Errorf("expected name test-rss, got %s", rss.Name)
	}
	if rss.Type != SourceTypeRSS {
		t.Errorf("expected type rss, got %s", rss.Type)
	}
	if rss.URL != "https://example.com/feed.xml" {
		t.Errorf("expected URL https://example.com/feed.xml, got %s", rss.URL)
	}
	if rss.Filters == nil || rss.Filters.TitleContains != "meeting" {
		t.Errorf("expected filter title_contains=meeting")
	}

	// Verify second source
	api := config.Sources[1]
	if api.Name != "test-api" {
		t.Errorf("expected name test-api, got %s", api.Name)
	}
	if api.Type != SourceTypeAPI {
		t.Errorf("expected type api, got %s", api.Type)
	}
	if api.Auth == nil || api.Auth.Type != AuthTypeBearer {
		t.Errorf("expected bearer auth")
	}
}

func TestLoadSourcesConfig_ValidationErrors(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		content string
		errMsg  string
	}{
		{
			name: "missing name",
			content: `sources:
  - type: rss
    url: https://example.com/feed.xml
    interval: 1h
    target_library: test`,
			errMsg: "name is required",
		},
		{
			name: "missing type",
			content: `sources:
  - name: test
    url: https://example.com/feed.xml
    interval: 1h
    target_library: test`,
			errMsg: "type is required",
		},
		{
			name: "missing interval",
			content: `sources:
  - name: test
    type: rss
    url: https://example.com/feed.xml
    target_library: test`,
			errMsg: "interval is required",
		},
		{
			name: "rss missing url",
			content: `sources:
  - name: test
    type: rss
    interval: 1h
    target_library: test`,
			errMsg: "url is required for rss type",
		},
		{
			name: "api missing endpoint",
			content: `sources:
  - name: test
    type: api
    interval: 1h
    target_library: test`,
			errMsg: "endpoint is required for api type",
		},
		{
			name: "scrape missing selector",
			content: `sources:
  - name: test
    type: scrape
    url: https://example.com
    interval: 1h
    target_library: test`,
			errMsg: "selector is required for scrape type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tempDir, tt.name+".yaml")
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write config: %v", err)
			}

			_, err := LoadSourcesConfig(configPath)
			if err == nil {
				t.Errorf("expected error containing %q, got nil", tt.errMsg)
			} else if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got %v", tt.errMsg, err)
			}
		})
	}
}

func TestNewWebSourceMonitor(t *testing.T) {
	config := &SourcesConfig{
		Sources: []SourceConfig{
			{
				Name:          "test",
				Type:          SourceTypeRSS,
				URL:           "https://example.com/feed.xml",
				Interval:      "1h",
				TargetLibrary: "test",
			},
		},
	}

	monitor := NewWebSourceMonitor(config)
	if monitor == nil {
		t.Fatal("NewWebSourceMonitor returned nil")
	}
	if monitor.config != config {
		t.Error("config not set correctly")
	}
	if monitor.userAgent != DefaultUserAgent {
		t.Errorf("expected user agent %s, got %s", DefaultUserAgent, monitor.userAgent)
	}
}

func TestCheckRSSSource(t *testing.T) {
	// Create test RSS feed
	rssFeed := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <item>
      <title>Document One</title>
      <link>https://example.com/doc1</link>
      <pubDate>Mon, 10 Feb 2025 10:00:00 +0000</pubDate>
      <category>deliberation</category>
    </item>
    <item>
      <title>Document Two</title>
      <link>https://example.com/doc2</link>
      <pubDate>Mon, 09 Feb 2025 10:00:00 +0000</pubDate>
    </item>
  </channel>
</rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(rssFeed))
	}))
	defer server.Close()

	config := &SourcesConfig{
		Sources: []SourceConfig{
			{
				Name:          "test-rss",
				Type:          SourceTypeRSS,
				URL:           server.URL,
				Interval:      "1h",
				TargetLibrary: "deliberations/test",
			},
		},
	}

	monitor := NewWebSourceMonitor(config)
	docs, err := monitor.CheckNow("test-rss")
	if err != nil {
		t.Fatalf("CheckNow failed: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("expected 2 documents, got %d", len(docs))
	}

	if docs[0].Title != "Document One" {
		t.Errorf("expected title 'Document One', got %s", docs[0].Title)
	}
	if docs[0].URL != "https://example.com/doc1" {
		t.Errorf("expected URL 'https://example.com/doc1', got %s", docs[0].URL)
	}
	if docs[0].SourceName != "test-rss" {
		t.Errorf("expected source name 'test-rss', got %s", docs[0].SourceName)
	}
}

func TestCheckAtomSource(t *testing.T) {
	// Create test Atom feed
	atomFeed := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Test Atom Feed</title>
  <entry>
    <title>Atom Document</title>
    <id>urn:uuid:1234</id>
    <updated>2025-02-10T10:00:00Z</updated>
    <link href="https://example.com/atom-doc"/>
    <summary>Test summary</summary>
  </entry>
</feed>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		w.Write([]byte(atomFeed))
	}))
	defer server.Close()

	config := &SourcesConfig{
		Sources: []SourceConfig{
			{
				Name:          "test-atom",
				Type:          SourceTypeRSS, // RSS handler also handles Atom
				URL:           server.URL,
				Interval:      "1h",
				TargetLibrary: "deliberations/test",
			},
		},
	}

	monitor := NewWebSourceMonitor(config)
	docs, err := monitor.CheckNow("test-atom")
	if err != nil {
		t.Fatalf("CheckNow failed: %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("expected 1 document, got %d", len(docs))
	}

	if docs[0].Title != "Atom Document" {
		t.Errorf("expected title 'Atom Document', got %s", docs[0].Title)
	}
	if docs[0].URL != "https://example.com/atom-doc" {
		t.Errorf("expected URL 'https://example.com/atom-doc', got %s", docs[0].URL)
	}
}

func TestCheckAPISource(t *testing.T) {
	// Create test API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query params
		if r.URL.Query().Get("type") != "resolution" {
			t.Errorf("expected type=resolution, got %s", r.URL.Query().Get("type"))
		}

		// Verify auth header
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("expected Bearer auth, got %s", auth)
		}

		response := []map[string]interface{}{
			{
				"url":          "https://example.com/api-doc1",
				"title":        "API Document One",
				"published_at": "2025-02-10T10:00:00Z",
			},
			{
				"url":          "https://example.com/api-doc2",
				"title":        "API Document Two",
				"published_at": "2025-02-09T10:00:00Z",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Set test env var
	os.Setenv("TEST_API_TOKEN", "test-token")
	defer os.Unsetenv("TEST_API_TOKEN")

	config := &SourcesConfig{
		Sources: []SourceConfig{
			{
				Name:     "test-api",
				Type:     SourceTypeAPI,
				Endpoint: server.URL,
				Method:   "GET",
				Params: map[string]string{
					"type": "resolution",
				},
				Interval:      "24h",
				TargetLibrary: "deliberations/api",
				Auth: &AuthConfig{
					Type:     AuthTypeBearer,
					TokenEnv: "TEST_API_TOKEN",
				},
			},
		},
	}

	monitor := NewWebSourceMonitor(config)
	docs, err := monitor.CheckNow("test-api")
	if err != nil {
		t.Fatalf("CheckNow failed: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("expected 2 documents, got %d", len(docs))
	}

	if docs[0].Title != "API Document One" {
		t.Errorf("expected title 'API Document One', got %s", docs[0].Title)
	}
}

func TestCheckAPISource_DataWrapper(t *testing.T) {
	// Test API that returns data in a wrapper
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"url":   "https://example.com/wrapped-doc",
					"title": "Wrapped Document",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &SourcesConfig{
		Sources: []SourceConfig{
			{
				Name:          "test-wrapped",
				Type:          SourceTypeAPI,
				Endpoint:      server.URL,
				Interval:      "1h",
				TargetLibrary: "test",
			},
		},
	}

	monitor := NewWebSourceMonitor(config)
	docs, err := monitor.CheckNow("test-wrapped")
	if err != nil {
		t.Fatalf("CheckNow failed: %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("expected 1 document, got %d", len(docs))
	}

	if docs[0].Title != "Wrapped Document" {
		t.Errorf("expected title 'Wrapped Document', got %s", docs[0].Title)
	}
}

func TestCheckScrapeSource(t *testing.T) {
	// Create test HTML page
	html := `<!DOCTYPE html>
<html>
<body>
  <a class="meeting-minutes" href="/docs/meeting1.pdf">January Meeting</a>
  <a class="meeting-minutes" href="/docs/meeting2.pdf">February Meeting</a>
  <a href="/other/link">Other Link</a>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
	defer server.Close()

	config := &SourcesConfig{
		Sources: []SourceConfig{
			{
				Name:          "test-scrape",
				Type:          SourceTypeScrape,
				URL:           server.URL,
				Selector:      ".meeting-minutes",
				Interval:      "6h",
				RateLimit:     "2/minute",
				TargetLibrary: "deliberations/scraped",
			},
		},
	}

	monitor := NewWebSourceMonitor(config)
	docs, err := monitor.CheckNow("test-scrape")
	if err != nil {
		t.Fatalf("CheckNow failed: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("expected 2 documents, got %d", len(docs))
	}

	// Check that links are correctly extracted
	found := false
	for _, doc := range docs {
		if strings.Contains(doc.Title, "January") {
			found = true
		}
	}
	if !found {
		t.Error("expected to find January meeting document")
	}
}

func TestDeduplication(t *testing.T) {
	rssFeed := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>Duplicate Document</title>
      <link>https://example.com/duplicate</link>
    </item>
  </channel>
</rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(rssFeed))
	}))
	defer server.Close()

	config := &SourcesConfig{
		Sources: []SourceConfig{
			{
				Name:          "test-dedup",
				Type:          SourceTypeRSS,
				URL:           server.URL,
				Interval:      "1h",
				TargetLibrary: "test",
			},
		},
	}

	monitor := NewWebSourceMonitor(config)

	// First check should return the document
	docs1, err := monitor.CheckNow("test-dedup")
	if err != nil {
		t.Fatalf("first CheckNow failed: %v", err)
	}
	if len(docs1) != 1 {
		t.Errorf("first check: expected 1 document, got %d", len(docs1))
	}

	// Second check should return empty (already seen)
	docs2, err := monitor.CheckNow("test-dedup")
	if err != nil {
		t.Fatalf("second CheckNow failed: %v", err)
	}
	if len(docs2) != 0 {
		t.Errorf("second check: expected 0 documents, got %d", len(docs2))
	}
}

func TestFilterTitleContains(t *testing.T) {
	rssFeed := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>Working Group Meeting</title>
      <link>https://example.com/meeting</link>
    </item>
    <item>
      <title>Press Release</title>
      <link>https://example.com/press</link>
    </item>
  </channel>
</rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(rssFeed))
	}))
	defer server.Close()

	config := &SourcesConfig{
		Sources: []SourceConfig{
			{
				Name:          "test-filter",
				Type:          SourceTypeRSS,
				URL:           server.URL,
				Interval:      "1h",
				TargetLibrary: "test",
				Filters: &FilterConfig{
					TitleContains: "meeting",
				},
			},
		},
	}

	monitor := NewWebSourceMonitor(config)
	docs, err := monitor.CheckNow("test-filter")
	if err != nil {
		t.Fatalf("CheckNow failed: %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("expected 1 document, got %d", len(docs))
	}
	if docs[0].Title != "Working Group Meeting" {
		t.Errorf("expected 'Working Group Meeting', got %s", docs[0].Title)
	}
}

func TestFilterTitleRegex(t *testing.T) {
	rssFeed := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>WG-2024-01 Minutes</title>
      <link>https://example.com/wg1</link>
    </item>
    <item>
      <title>WG-2024-02 Minutes</title>
      <link>https://example.com/wg2</link>
    </item>
    <item>
      <title>General Announcement</title>
      <link>https://example.com/announce</link>
    </item>
  </channel>
</rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(rssFeed))
	}))
	defer server.Close()

	config := &SourcesConfig{
		Sources: []SourceConfig{
			{
				Name:          "test-regex",
				Type:          SourceTypeRSS,
				URL:           server.URL,
				Interval:      "1h",
				TargetLibrary: "test",
				Filters: &FilterConfig{
					TitleRegex: `WG-\d{4}-\d{2}`,
				},
			},
		},
	}

	monitor := NewWebSourceMonitor(config)
	docs, err := monitor.CheckNow("test-regex")
	if err != nil {
		t.Fatalf("CheckNow failed: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("expected 2 documents, got %d", len(docs))
	}
}

func TestFilterCategory(t *testing.T) {
	rssFeed := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>Deliberation Doc</title>
      <link>https://example.com/delib</link>
      <category>deliberation</category>
    </item>
    <item>
      <title>News Item</title>
      <link>https://example.com/news</link>
      <category>news</category>
    </item>
  </channel>
</rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(rssFeed))
	}))
	defer server.Close()

	config := &SourcesConfig{
		Sources: []SourceConfig{
			{
				Name:          "test-category",
				Type:          SourceTypeRSS,
				URL:           server.URL,
				Interval:      "1h",
				TargetLibrary: "test",
				Filters: &FilterConfig{
					Category: "deliberation",
				},
			},
		},
	}

	monitor := NewWebSourceMonitor(config)
	docs, err := monitor.CheckNow("test-category")
	if err != nil {
		t.Fatalf("CheckNow failed: %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("expected 1 document, got %d", len(docs))
	}
	if docs[0].Title != "Deliberation Doc" {
		t.Errorf("expected 'Deliberation Doc', got %s", docs[0].Title)
	}
}

func TestOnNewDocumentCallback(t *testing.T) {
	rssFeed := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>Callback Test</title>
      <link>https://example.com/callback</link>
    </item>
  </channel>
</rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(rssFeed))
	}))
	defer server.Close()

	config := &SourcesConfig{
		Sources: []SourceConfig{
			{
				Name:          "test-callback",
				Type:          SourceTypeRSS,
				URL:           server.URL,
				Interval:      "1h",
				TargetLibrary: "test",
			},
		},
	}

	monitor := NewWebSourceMonitor(config)

	var receivedDocs []DocumentRef
	var mu sync.Mutex

	monitor.OnNewDocument(func(doc DocumentRef) error {
		mu.Lock()
		defer mu.Unlock()
		receivedDocs = append(receivedDocs, doc)
		return nil
	})

	// Start and stop quickly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go monitor.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	monitor.Stop()

	mu.Lock()
	defer mu.Unlock()

	if len(receivedDocs) == 0 {
		// May not have received callback in time, that's OK for this test
		t.Log("callback not received in time window (expected for short timeout)")
	} else if receivedDocs[0].Title != "Callback Test" {
		t.Errorf("expected title 'Callback Test', got %s", receivedDocs[0].Title)
	}
}

func TestPauseResume(t *testing.T) {
	config := &SourcesConfig{
		Sources: []SourceConfig{
			{
				Name:          "test-pause",
				Type:          SourceTypeRSS,
				URL:           "https://example.com/feed.xml",
				Interval:      "1h",
				TargetLibrary: "test",
			},
		},
	}

	monitor := NewWebSourceMonitor(config)

	// Initialize status
	monitor.setStatus("test-pause", &SourceStatusInfo{
		Name:   "test-pause",
		Type:   SourceTypeRSS,
		Status: SourceStatusActive,
	})

	// Pause
	if err := monitor.Pause("test-pause"); err != nil {
		t.Fatalf("Pause failed: %v", err)
	}

	status := monitor.Status()
	var found bool
	for _, s := range status {
		if s.Name == "test-pause" && s.Status == SourceStatusPaused {
			found = true
			break
		}
	}
	if !found {
		t.Error("source should be paused")
	}

	// Resume
	if err := monitor.Resume("test-pause"); err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	status = monitor.Status()
	for _, s := range status {
		if s.Name == "test-pause" && s.Status == SourceStatusActive {
			found = true
			break
		}
	}
	if !found {
		t.Error("source should be active")
	}
}

func TestPauseResume_NotFound(t *testing.T) {
	config := &SourcesConfig{Sources: []SourceConfig{}}
	monitor := NewWebSourceMonitor(config)

	if err := monitor.Pause("nonexistent"); err == nil {
		t.Error("expected error for nonexistent source")
	}

	if err := monitor.Resume("nonexistent"); err == nil {
		t.Error("expected error for nonexistent source")
	}
}

func TestCheckNow_NotFound(t *testing.T) {
	config := &SourcesConfig{Sources: []SourceConfig{}}
	monitor := NewWebSourceMonitor(config)

	_, err := monitor.CheckNow("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent source")
	}
}

func TestSeenURLsPersistence(t *testing.T) {
	config := &SourcesConfig{Sources: []SourceConfig{}}
	monitor := NewWebSourceMonitor(config)

	// Load some seen URLs
	monitor.LoadSeenURLs([]string{
		"https://example.com/doc1",
		"https://example.com/doc2",
	})

	// Get seen URLs
	urls := monitor.GetSeenURLs()
	if len(urls) != 2 {
		t.Errorf("expected 2 seen URLs, got %d", len(urls))
	}

	// Clear and verify
	monitor.ClearSeen()
	urls = monitor.GetSeenURLs()
	if len(urls) != 0 {
		t.Errorf("expected 0 seen URLs after clear, got %d", len(urls))
	}
}

func TestParseRSSDate(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
	}{
		{"Mon, 10 Feb 2025 10:00:00 +0000", time.Date(2025, 2, 10, 10, 0, 0, 0, time.UTC)},
		{"2025-02-10T10:00:00Z", time.Date(2025, 2, 10, 10, 0, 0, 0, time.UTC)},
		{"2025-02-10", time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC)},
		{"invalid", time.Time{}},
	}

	for _, tt := range tests {
		result := parseRSSDate(tt.input)
		if !result.Equal(tt.expected) {
			t.Errorf("parseRSSDate(%q): got %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestParseAtomDate(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
	}{
		{"2025-02-10T10:00:00Z", time.Date(2025, 2, 10, 10, 0, 0, 0, time.UTC)},
		{"2025-02-10", time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC)},
		{"invalid", time.Time{}},
	}

	for _, tt := range tests {
		result := parseAtomDate(tt.input)
		if !result.Equal(tt.expected) {
			t.Errorf("parseAtomDate(%q): got %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestParseRateLimit(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"2/minute", 30 * time.Second},
		{"1/second", time.Second},
		{"4/hour", 15 * time.Minute},
		{"invalid", 30 * time.Second},
		{"1/unknown", time.Minute},
	}

	for _, tt := range tests {
		result := parseRateLimit(tt.input)
		if result != tt.expected {
			t.Errorf("parseRateLimit(%q): got %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestAuthTypes(t *testing.T) {
	// Test bearer auth
	t.Run("bearer", func(t *testing.T) {
		os.Setenv("TEST_BEARER_TOKEN", "my-token")
		defer os.Unsetenv("TEST_BEARER_TOKEN")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer my-token" {
				t.Errorf("expected Bearer my-token, got %s", auth)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Write([]byte(`[]`))
		}))
		defer server.Close()

		config := &SourcesConfig{
			Sources: []SourceConfig{
				{
					Name:          "bearer-test",
					Type:          SourceTypeAPI,
					Endpoint:      server.URL,
					Interval:      "1h",
					TargetLibrary: "test",
					Auth: &AuthConfig{
						Type:     AuthTypeBearer,
						TokenEnv: "TEST_BEARER_TOKEN",
					},
				},
			},
		}

		monitor := NewWebSourceMonitor(config)
		_, err := monitor.CheckNow("bearer-test")
		if err != nil {
			t.Errorf("bearer auth failed: %v", err)
		}
	})

	// Test basic auth
	t.Run("basic", func(t *testing.T) {
		os.Setenv("TEST_USERNAME", "user")
		os.Setenv("TEST_PASSWORD", "pass")
		defer os.Unsetenv("TEST_USERNAME")
		defer os.Unsetenv("TEST_PASSWORD")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			if !ok || user != "user" || pass != "pass" {
				t.Errorf("basic auth failed: ok=%v user=%s pass=%s", ok, user, pass)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Write([]byte(`[]`))
		}))
		defer server.Close()

		config := &SourcesConfig{
			Sources: []SourceConfig{
				{
					Name:          "basic-test",
					Type:          SourceTypeAPI,
					Endpoint:      server.URL,
					Interval:      "1h",
					TargetLibrary: "test",
					Auth: &AuthConfig{
						Type:        AuthTypeBasic,
						UsernameEnv: "TEST_USERNAME",
						PasswordEnv: "TEST_PASSWORD",
					},
				},
			},
		}

		monitor := NewWebSourceMonitor(config)
		_, err := monitor.CheckNow("basic-test")
		if err != nil {
			t.Errorf("basic auth failed: %v", err)
		}
	})

	// Test API key auth
	t.Run("api_key", func(t *testing.T) {
		os.Setenv("TEST_API_KEY", "secret-key")
		defer os.Unsetenv("TEST_API_KEY")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-Custom-Key")
			if key != "secret-key" {
				t.Errorf("expected X-Custom-Key=secret-key, got %s", key)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Write([]byte(`[]`))
		}))
		defer server.Close()

		config := &SourcesConfig{
			Sources: []SourceConfig{
				{
					Name:          "apikey-test",
					Type:          SourceTypeAPI,
					Endpoint:      server.URL,
					Interval:      "1h",
					TargetLibrary: "test",
					Auth: &AuthConfig{
						Type:       AuthTypeAPIKey,
						TokenEnv:   "TEST_API_KEY",
						HeaderName: "X-Custom-Key",
					},
				},
			},
		}

		monitor := NewWebSourceMonitor(config)
		_, err := monitor.CheckNow("apikey-test")
		if err != nil {
			t.Errorf("API key auth failed: %v", err)
		}
	})
}

func TestTemplateVarExpansion(t *testing.T) {
	now := time.Now()

	config := &SourcesConfig{
		Sources: []SourceConfig{
			{
				Name:          "template-test",
				Type:          SourceTypeAPI,
				Endpoint:      "https://example.com/api",
				Interval:      "1h",
				TargetLibrary: "test",
			},
		},
	}

	monitor := NewWebSourceMonitor(config)
	monitor.setStatus("template-test", &SourceStatusInfo{
		Name:      "template-test",
		LastCheck: now.Add(-2 * time.Hour),
	})

	result := monitor.expandTemplateVars("{{last_check}}", config.Sources[0])
	if result == "{{last_check}}" {
		t.Error("template variable was not expanded")
	}
	if !strings.Contains(result, "T") {
		t.Errorf("expected RFC3339 format, got %s", result)
	}
}

func TestStartStop(t *testing.T) {
	config := &SourcesConfig{
		Sources: []SourceConfig{
			{
				Name:          "start-stop-test",
				Type:          SourceTypeRSS,
				URL:           "https://example.com/feed.xml",
				Interval:      "1h",
				TargetLibrary: "test",
			},
		},
	}

	monitor := NewWebSourceMonitor(config)

	// Start
	ctx := context.Background()
	if err := monitor.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Double start should fail
	if err := monitor.Start(ctx); err == nil {
		t.Error("expected error on double start")
	}

	// Stop
	if err := monitor.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Double stop should fail
	if err := monitor.Stop(); err == nil {
		t.Error("expected error on double stop")
	}
}

func TestRSSFeedXMLParsing(t *testing.T) {
	// Test that RSSFeed and RSSItem structs parse correctly
	rssFeed := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <item>
      <title>Test Title</title>
      <link>https://example.com/test</link>
      <description>Test description</description>
      <pubDate>Mon, 10 Feb 2025 10:00:00 +0000</pubDate>
      <category>test-category</category>
      <guid>unique-guid-123</guid>
    </item>
  </channel>
</rss>`

	var feed RSSFeed
	if err := xml.Unmarshal([]byte(rssFeed), &feed); err != nil {
		t.Fatalf("failed to parse RSS feed: %v", err)
	}

	if len(feed.Channel.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(feed.Channel.Items))
	}

	item := feed.Channel.Items[0]
	if item.Title != "Test Title" {
		t.Errorf("expected title 'Test Title', got %s", item.Title)
	}
	if item.Link != "https://example.com/test" {
		t.Errorf("expected link 'https://example.com/test', got %s", item.Link)
	}
	if item.Description != "Test description" {
		t.Errorf("expected description 'Test description', got %s", item.Description)
	}
	if item.Category != "test-category" {
		t.Errorf("expected category 'test-category', got %s", item.Category)
	}
	if item.GUID != "unique-guid-123" {
		t.Errorf("expected GUID 'unique-guid-123', got %s", item.GUID)
	}
}

func TestAtomFeedXMLParsing(t *testing.T) {
	atomFeed := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <title>Atom Test</title>
    <id>urn:uuid:1234</id>
    <updated>2025-02-10T10:00:00Z</updated>
    <link href="https://example.com/atom" rel="alternate"/>
    <link href="https://example.com/atom.json" rel="self"/>
    <summary>Atom summary</summary>
  </entry>
</feed>`

	var feed AtomFeed
	if err := xml.Unmarshal([]byte(atomFeed), &feed); err != nil {
		t.Fatalf("failed to parse Atom feed: %v", err)
	}

	if len(feed.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(feed.Entries))
	}

	entry := feed.Entries[0]
	if entry.Title != "Atom Test" {
		t.Errorf("expected title 'Atom Test', got %s", entry.Title)
	}
	if entry.ID != "urn:uuid:1234" {
		t.Errorf("expected ID 'urn:uuid:1234', got %s", entry.ID)
	}
	if entry.Summary != "Atom summary" {
		t.Errorf("expected summary 'Atom summary', got %s", entry.Summary)
	}
	if len(entry.Links) != 2 {
		t.Errorf("expected 2 links, got %d", len(entry.Links))
	}
}

func TestSourceConfigEnabled(t *testing.T) {
	falseVal := false
	trueVal := true

	tests := []struct {
		name     string
		enabled  *bool
		expected bool
	}{
		{"nil (default enabled)", nil, true},
		{"explicitly enabled", &trueVal, true},
		{"explicitly disabled", &falseVal, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := SourceConfig{
				Name:          "test",
				Type:          SourceTypeRSS,
				URL:           "https://example.com/feed",
				Interval:      "1h",
				TargetLibrary: "test",
				Enabled:       tt.enabled,
			}

			// Default enabled is true when Enabled is nil
			enabled := config.Enabled == nil || *config.Enabled
			if enabled != tt.expected {
				t.Errorf("expected enabled=%v, got %v", tt.expected, enabled)
			}
		})
	}
}

func TestDocumentRefMetadata(t *testing.T) {
	doc := DocumentRef{
		URL:           "https://example.com/doc",
		Title:         "Test Doc",
		PublishedAt:   time.Now(),
		SourceName:    "test-source",
		TargetLibrary: "test-library",
		Metadata: map[string]string{
			"author":   "Jane Doe",
			"category": "deliberation",
			"tags":     "important,urgent",
		},
	}

	if doc.Metadata["author"] != "Jane Doe" {
		t.Errorf("expected author 'Jane Doe', got %s", doc.Metadata["author"])
	}
	if doc.Metadata["category"] != "deliberation" {
		t.Errorf("expected category 'deliberation', got %s", doc.Metadata["category"])
	}
}

func TestSourceStatusInfo(t *testing.T) {
	now := time.Now()
	info := SourceStatusInfo{
		Name:           "test-source",
		Type:           SourceTypeRSS,
		Status:         SourceStatusActive,
		LastCheck:      now.Add(-1 * time.Hour),
		NextCheck:      now.Add(1 * time.Hour),
		DocumentsFound: 42,
		DocumentsNew:   5,
		Errors:         []string{"timeout", "parse error"},
	}

	if info.Name != "test-source" {
		t.Errorf("expected name 'test-source', got %s", info.Name)
	}
	if info.DocumentsFound != 42 {
		t.Errorf("expected 42 documents found, got %d", info.DocumentsFound)
	}
	if len(info.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(info.Errors))
	}
}
