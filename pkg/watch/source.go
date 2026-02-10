// Package watch provides web source monitoring for RSS feeds, REST APIs,
// and web pages to automatically discover and ingest new deliberation documents.
package watch

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// SourceType indicates the type of web source being monitored.
type SourceType string

const (
	// SourceTypeRSS monitors RSS/Atom feeds for new documents.
	SourceTypeRSS SourceType = "rss"

	// SourceTypeAPI polls REST APIs for document listings.
	SourceTypeAPI SourceType = "api"

	// SourceTypeScrape extracts document links from web pages.
	SourceTypeScrape SourceType = "scrape"

	// SourceTypeWebhook accepts push notifications from document systems.
	SourceTypeWebhook SourceType = "webhook"
)

// SourceStatus indicates the operational state of a source monitor.
type SourceStatus string

const (
	// SourceStatusActive indicates the source is being monitored.
	SourceStatusActive SourceStatus = "active"

	// SourceStatusPaused indicates monitoring is temporarily paused.
	SourceStatusPaused SourceStatus = "paused"

	// SourceStatusError indicates the source is in an error state.
	SourceStatusError SourceStatus = "error"

	// SourceStatusDisabled indicates the source has been disabled.
	SourceStatusDisabled SourceStatus = "disabled"
)

// AuthType indicates the authentication method for API sources.
type AuthType string

const (
	// AuthTypeNone indicates no authentication required.
	AuthTypeNone AuthType = "none"

	// AuthTypeBearer indicates Bearer token authentication.
	AuthTypeBearer AuthType = "bearer"

	// AuthTypeBasic indicates HTTP Basic authentication.
	AuthTypeBasic AuthType = "basic"

	// AuthTypeAPIKey indicates API key authentication.
	AuthTypeAPIKey AuthType = "api_key"
)

// AuthConfig holds authentication configuration for API sources.
type AuthConfig struct {
	// Type is the authentication method.
	Type AuthType `yaml:"type" json:"type"`

	// TokenEnv is the environment variable containing the token.
	TokenEnv string `yaml:"token_env,omitempty" json:"token_env,omitempty"`

	// UsernameEnv is the environment variable containing the username.
	UsernameEnv string `yaml:"username_env,omitempty" json:"username_env,omitempty"`

	// PasswordEnv is the environment variable containing the password.
	PasswordEnv string `yaml:"password_env,omitempty" json:"password_env,omitempty"`

	// HeaderName is the header name for API key authentication.
	HeaderName string `yaml:"header_name,omitempty" json:"header_name,omitempty"`
}

// FilterConfig defines filters for document discovery.
type FilterConfig struct {
	// TitleContains filters documents by title substring.
	TitleContains string `yaml:"title_contains,omitempty" json:"title_contains,omitempty"`

	// TitleRegex filters documents by title regex pattern.
	TitleRegex string `yaml:"title_regex,omitempty" json:"title_regex,omitempty"`

	// Category filters by document category.
	Category string `yaml:"category,omitempty" json:"category,omitempty"`

	// MinAge excludes documents older than this duration.
	MinAge string `yaml:"min_age,omitempty" json:"min_age,omitempty"`

	// MaxAge excludes documents newer than this duration.
	MaxAge string `yaml:"max_age,omitempty" json:"max_age,omitempty"`
}

// SourceConfig holds configuration for a single web source.
type SourceConfig struct {
	// Name is the unique identifier for this source.
	Name string `yaml:"name" json:"name"`

	// Type is the source type (rss, api, scrape, webhook).
	Type SourceType `yaml:"type" json:"type"`

	// URL is the source URL (for rss, scrape).
	URL string `yaml:"url,omitempty" json:"url,omitempty"`

	// Endpoint is the API endpoint (for api type).
	Endpoint string `yaml:"endpoint,omitempty" json:"endpoint,omitempty"`

	// Method is the HTTP method for API sources (default GET).
	Method string `yaml:"method,omitempty" json:"method,omitempty"`

	// Params holds query parameters for API sources.
	Params map[string]string `yaml:"params,omitempty" json:"params,omitempty"`

	// Selector is the CSS selector for scrape sources.
	Selector string `yaml:"selector,omitempty" json:"selector,omitempty"`

	// Interval is the polling interval (e.g., "1h", "24h").
	Interval string `yaml:"interval" json:"interval"`

	// Auth holds authentication configuration.
	Auth *AuthConfig `yaml:"auth,omitempty" json:"auth,omitempty"`

	// Filters holds document filtering configuration.
	Filters *FilterConfig `yaml:"filters,omitempty" json:"filters,omitempty"`

	// RateLimit limits requests (e.g., "2/minute").
	RateLimit string `yaml:"rate_limit,omitempty" json:"rate_limit,omitempty"`

	// TargetLibrary is the library path for ingested documents.
	TargetLibrary string `yaml:"target_library" json:"target_library"`

	// Backfill enables fetching historical documents on first run.
	Backfill bool `yaml:"backfill,omitempty" json:"backfill,omitempty"`

	// Enabled indicates if this source is active (default true).
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
}

// SourcesConfig holds the complete sources configuration.
type SourcesConfig struct {
	// Sources is the list of configured web sources.
	Sources []SourceConfig `yaml:"sources" json:"sources"`
}

// DocumentRef represents a discovered document reference.
type DocumentRef struct {
	// URL is the document URL.
	URL string `json:"url"`

	// Title is the document title.
	Title string `json:"title"`

	// PublishedAt is when the document was published.
	PublishedAt time.Time `json:"published_at"`

	// SourceName is the source that discovered this document.
	SourceName string `json:"source_name"`

	// TargetLibrary is the destination library path.
	TargetLibrary string `json:"target_library"`

	// Metadata holds additional document metadata.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// SourceStatusInfo provides status information for a source.
type SourceStatusInfo struct {
	// Name is the source name.
	Name string `json:"name"`

	// Type is the source type.
	Type SourceType `json:"type"`

	// Status is the current operational status.
	Status SourceStatus `json:"status"`

	// LastCheck is when the source was last checked.
	LastCheck time.Time `json:"last_check"`

	// NextCheck is when the next check is scheduled.
	NextCheck time.Time `json:"next_check"`

	// DocumentsFound is the total documents found.
	DocumentsFound int `json:"documents_found"`

	// DocumentsNew is documents found in the last check.
	DocumentsNew int `json:"documents_new"`

	// Errors holds recent error messages.
	Errors []string `json:"errors,omitempty"`
}

// WebSourceMonitor monitors web sources for new documents.
type WebSourceMonitor struct {
	config      *SourcesConfig
	client      *http.Client
	seen        map[string]bool // URL -> seen
	seenMu      sync.RWMutex
	statuses    map[string]*SourceStatusInfo
	statusMu    sync.RWMutex
	callbacks   []func(DocumentRef) error
	callbackMu  sync.RWMutex
	stopChan    chan struct{}
	running     bool
	runningMu   sync.Mutex
	userAgent   string
	rateLimiter *rateLimiter
}

// rateLimiter implements per-domain rate limiting.
type rateLimiter struct {
	domains map[string]time.Time
	mu      sync.Mutex
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		domains: make(map[string]time.Time),
	}
}

func (r *rateLimiter) wait(domain string, interval time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if lastReq, ok := r.domains[domain]; ok {
		waitTime := interval - time.Since(lastReq)
		if waitTime > 0 {
			time.Sleep(waitTime)
		}
	}
	r.domains[domain] = time.Now()
}

// DefaultUserAgent is the default User-Agent for requests.
const DefaultUserAgent = "regula-watch/1.0 (+https://regula.dev)"

// DefaultTimeout is the default HTTP request timeout.
const DefaultTimeout = 30 * time.Second

// NewWebSourceMonitor creates a new web source monitor.
func NewWebSourceMonitor(config *SourcesConfig) *WebSourceMonitor {
	return &WebSourceMonitor{
		config:      config,
		client:      &http.Client{Timeout: DefaultTimeout},
		seen:        make(map[string]bool),
		statuses:    make(map[string]*SourceStatusInfo),
		callbacks:   make([]func(DocumentRef) error, 0),
		stopChan:    make(chan struct{}),
		userAgent:   DefaultUserAgent,
		rateLimiter: newRateLimiter(),
	}
}

// LoadSourcesConfig loads source configuration from a YAML file.
func LoadSourcesConfig(path string) (*SourcesConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read sources config: %w", err)
	}

	var config SourcesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse sources config: %w", err)
	}

	// Validate sources
	for i, source := range config.Sources {
		if source.Name == "" {
			return nil, fmt.Errorf("source %d: name is required", i)
		}
		if source.Type == "" {
			return nil, fmt.Errorf("source %s: type is required", source.Name)
		}
		if source.Interval == "" {
			return nil, fmt.Errorf("source %s: interval is required", source.Name)
		}
		if source.TargetLibrary == "" {
			return nil, fmt.Errorf("source %s: target_library is required", source.Name)
		}

		// Validate type-specific requirements
		switch source.Type {
		case SourceTypeRSS:
			if source.URL == "" {
				return nil, fmt.Errorf("source %s: url is required for rss type", source.Name)
			}
		case SourceTypeAPI:
			if source.Endpoint == "" {
				return nil, fmt.Errorf("source %s: endpoint is required for api type", source.Name)
			}
		case SourceTypeScrape:
			if source.URL == "" {
				return nil, fmt.Errorf("source %s: url is required for scrape type", source.Name)
			}
			if source.Selector == "" {
				return nil, fmt.Errorf("source %s: selector is required for scrape type", source.Name)
			}
		}
	}

	return &config, nil
}

// Start begins monitoring all configured sources.
func (m *WebSourceMonitor) Start(ctx context.Context) error {
	m.runningMu.Lock()
	if m.running {
		m.runningMu.Unlock()
		return fmt.Errorf("monitor is already running")
	}
	m.running = true
	m.stopChan = make(chan struct{})
	m.runningMu.Unlock()

	// Initialize status for all sources
	for _, source := range m.config.Sources {
		enabled := source.Enabled == nil || *source.Enabled
		status := SourceStatusActive
		if !enabled {
			status = SourceStatusDisabled
		}
		m.setStatus(source.Name, &SourceStatusInfo{
			Name:   source.Name,
			Type:   source.Type,
			Status: status,
		})
	}

	// Start a goroutine for each source
	var wg sync.WaitGroup
	for _, source := range m.config.Sources {
		if source.Enabled != nil && !*source.Enabled {
			continue
		}

		wg.Add(1)
		go func(src SourceConfig) {
			defer wg.Done()
			m.monitorSource(ctx, src)
		}(source)
	}

	// Wait for all monitors to finish
	go func() {
		wg.Wait()
	}()

	return nil
}

// Stop stops all source monitors.
func (m *WebSourceMonitor) Stop() error {
	m.runningMu.Lock()
	defer m.runningMu.Unlock()

	if !m.running {
		return fmt.Errorf("monitor is not running")
	}

	close(m.stopChan)
	m.running = false
	return nil
}

// CheckNow immediately checks a source for new documents.
func (m *WebSourceMonitor) CheckNow(sourceName string) ([]DocumentRef, error) {
	var source *SourceConfig
	for i := range m.config.Sources {
		if m.config.Sources[i].Name == sourceName {
			source = &m.config.Sources[i]
			break
		}
	}
	if source == nil {
		return nil, fmt.Errorf("source not found: %s", sourceName)
	}

	return m.checkSource(context.Background(), *source)
}

// Status returns the status of all sources.
func (m *WebSourceMonitor) Status() []SourceStatusInfo {
	m.statusMu.RLock()
	defer m.statusMu.RUnlock()

	result := make([]SourceStatusInfo, 0, len(m.statuses))
	for _, status := range m.statuses {
		result = append(result, *status)
	}
	return result
}

// OnNewDocument registers a callback for new document discoveries.
func (m *WebSourceMonitor) OnNewDocument(callback func(DocumentRef) error) {
	m.callbackMu.Lock()
	defer m.callbackMu.Unlock()
	m.callbacks = append(m.callbacks, callback)
}

// Pause pauses monitoring of a specific source.
func (m *WebSourceMonitor) Pause(sourceName string) error {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()

	status, ok := m.statuses[sourceName]
	if !ok {
		return fmt.Errorf("source not found: %s", sourceName)
	}

	status.Status = SourceStatusPaused
	return nil
}

// Resume resumes monitoring of a paused source.
func (m *WebSourceMonitor) Resume(sourceName string) error {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()

	status, ok := m.statuses[sourceName]
	if !ok {
		return fmt.Errorf("source not found: %s", sourceName)
	}

	if status.Status == SourceStatusPaused {
		status.Status = SourceStatusActive
	}
	return nil
}

// monitorSource continuously monitors a single source.
func (m *WebSourceMonitor) monitorSource(ctx context.Context, source SourceConfig) {
	interval, err := time.ParseDuration(source.Interval)
	if err != nil {
		m.recordError(source.Name, fmt.Sprintf("invalid interval: %s", source.Interval))
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial check
	m.doCheck(ctx, source)

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.statusMu.RLock()
			status := m.statuses[source.Name]
			isPaused := status != nil && status.Status == SourceStatusPaused
			m.statusMu.RUnlock()

			if !isPaused {
				m.doCheck(ctx, source)
			}
		}
	}
}

// doCheck performs a single check of a source.
func (m *WebSourceMonitor) doCheck(ctx context.Context, source SourceConfig) {
	docs, err := m.checkSource(ctx, source)
	if err != nil {
		m.recordError(source.Name, err.Error())
		return
	}

	// Update status
	m.statusMu.Lock()
	if status, ok := m.statuses[source.Name]; ok {
		status.LastCheck = time.Now()
		interval, _ := time.ParseDuration(source.Interval)
		status.NextCheck = time.Now().Add(interval)
		status.DocumentsNew = len(docs)
		status.DocumentsFound += len(docs)
		status.Status = SourceStatusActive
	}
	m.statusMu.Unlock()

	// Notify callbacks for new documents
	for _, doc := range docs {
		m.notifyCallbacks(doc)
	}
}

// checkSource checks a source and returns new documents.
func (m *WebSourceMonitor) checkSource(ctx context.Context, source SourceConfig) ([]DocumentRef, error) {
	switch source.Type {
	case SourceTypeRSS:
		return m.checkRSSSource(ctx, source)
	case SourceTypeAPI:
		return m.checkAPISource(ctx, source)
	case SourceTypeScrape:
		return m.checkScrapeSource(ctx, source)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", source.Type)
	}
}

// checkRSSSource fetches and parses an RSS/Atom feed.
func (m *WebSourceMonitor) checkRSSSource(ctx context.Context, source SourceConfig) ([]DocumentRef, error) {
	// Rate limit
	if u, err := url.Parse(source.URL); err == nil {
		m.rateLimiter.wait(u.Host, 2*time.Second)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", source.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", m.userAgent)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feed returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read feed body: %w", err)
	}

	// Try RSS first, then Atom
	docs, err := m.parseRSSFeed(body, source)
	if err != nil {
		docs, err = m.parseAtomFeed(body, source)
		if err != nil {
			return nil, fmt.Errorf("failed to parse feed: %w", err)
		}
	}

	return m.filterAndDedup(docs, source)
}

// RSSFeed represents an RSS feed.
type RSSFeed struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Items []RSSItem `xml:"item"`
	} `xml:"channel"`
}

// RSSItem represents an RSS item.
type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Category    string `xml:"category"`
	GUID        string `xml:"guid"`
}

// parseRSSFeed parses RSS XML into document references.
func (m *WebSourceMonitor) parseRSSFeed(data []byte, source SourceConfig) ([]DocumentRef, error) {
	var feed RSSFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, err
	}

	docs := make([]DocumentRef, 0, len(feed.Channel.Items))
	for _, item := range feed.Channel.Items {
		pubTime := parseRSSDate(item.PubDate)
		docs = append(docs, DocumentRef{
			URL:           item.Link,
			Title:         item.Title,
			PublishedAt:   pubTime,
			SourceName:    source.Name,
			TargetLibrary: source.TargetLibrary,
			Metadata: map[string]string{
				"description": item.Description,
				"category":    item.Category,
				"guid":        item.GUID,
			},
		})
	}

	return docs, nil
}

// AtomFeed represents an Atom feed.
type AtomFeed struct {
	XMLName xml.Name   `xml:"feed"`
	Entries []AtomEntry `xml:"entry"`
}

// AtomEntry represents an Atom entry.
type AtomEntry struct {
	Title   string     `xml:"title"`
	ID      string     `xml:"id"`
	Updated string     `xml:"updated"`
	Links   []AtomLink `xml:"link"`
	Summary string     `xml:"summary"`
}

// AtomLink represents an Atom link.
type AtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

// parseAtomFeed parses Atom XML into document references.
func (m *WebSourceMonitor) parseAtomFeed(data []byte, source SourceConfig) ([]DocumentRef, error) {
	var feed AtomFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, err
	}

	docs := make([]DocumentRef, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		// Find the alternate link
		var link string
		for _, l := range entry.Links {
			if l.Rel == "" || l.Rel == "alternate" {
				link = l.Href
				break
			}
		}
		if link == "" && len(entry.Links) > 0 {
			link = entry.Links[0].Href
		}

		pubTime := parseAtomDate(entry.Updated)
		docs = append(docs, DocumentRef{
			URL:           link,
			Title:         entry.Title,
			PublishedAt:   pubTime,
			SourceName:    source.Name,
			TargetLibrary: source.TargetLibrary,
			Metadata: map[string]string{
				"id":      entry.ID,
				"summary": entry.Summary,
			},
		})
	}

	return docs, nil
}

// checkAPISource polls a REST API for documents.
func (m *WebSourceMonitor) checkAPISource(ctx context.Context, source SourceConfig) ([]DocumentRef, error) {
	// Build URL with parameters
	reqURL, err := url.Parse(source.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}

	// Add query parameters
	q := reqURL.Query()
	for k, v := range source.Params {
		// Handle template variables
		v = m.expandTemplateVars(v, source)
		q.Set(k, v)
	}
	reqURL.RawQuery = q.Encode()

	// Rate limit
	m.rateLimiter.wait(reqURL.Host, 2*time.Second)

	method := source.Method
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", m.userAgent)

	// Add authentication
	if source.Auth != nil {
		if err := m.addAuth(req, source.Auth); err != nil {
			return nil, fmt.Errorf("failed to add auth: %w", err)
		}
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON response - expecting array of documents
	var items []map[string]interface{}
	if err := json.Unmarshal(body, &items); err != nil {
		// Try wrapping in data field
		var wrapper struct {
			Data []map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal(body, &wrapper); err != nil {
			return nil, fmt.Errorf("failed to parse API response: %w", err)
		}
		items = wrapper.Data
	}

	docs := make([]DocumentRef, 0, len(items))
	for _, item := range items {
		doc := DocumentRef{
			SourceName:    source.Name,
			TargetLibrary: source.TargetLibrary,
			Metadata:      make(map[string]string),
		}

		// Extract common fields
		if url, ok := item["url"].(string); ok {
			doc.URL = url
		} else if link, ok := item["link"].(string); ok {
			doc.URL = link
		}
		if title, ok := item["title"].(string); ok {
			doc.Title = title
		} else if name, ok := item["name"].(string); ok {
			doc.Title = name
		}
		if pubDate, ok := item["published_at"].(string); ok {
			doc.PublishedAt, _ = time.Parse(time.RFC3339, pubDate)
		} else if pubDate, ok := item["date"].(string); ok {
			doc.PublishedAt, _ = time.Parse(time.RFC3339, pubDate)
		}

		// Copy all fields to metadata
		for k, v := range item {
			if s, ok := v.(string); ok {
				doc.Metadata[k] = s
			}
		}

		if doc.URL != "" {
			docs = append(docs, doc)
		}
	}

	return m.filterAndDedup(docs, source)
}

// checkScrapeSource extracts document links from a web page.
func (m *WebSourceMonitor) checkScrapeSource(ctx context.Context, source SourceConfig) ([]DocumentRef, error) {
	// Rate limit
	if u, err := url.Parse(source.URL); err == nil {
		interval := 30 * time.Second // Default for scraping
		if source.RateLimit != "" {
			interval = parseRateLimit(source.RateLimit)
		}
		m.rateLimiter.wait(u.Host, interval)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", source.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", m.userAgent)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read page: %w", err)
	}

	// Simple link extraction based on selector
	// For production, would use a proper HTML parser like goquery
	docs := extractLinksSimple(string(body), source)

	return m.filterAndDedup(docs, source)
}

// extractLinksSimple performs basic link extraction from HTML.
// This is a simplified implementation; production would use goquery or similar.
func extractLinksSimple(html string, source SourceConfig) []DocumentRef {
	docs := make([]DocumentRef, 0)

	// Extract all href links
	linkRegex := regexp.MustCompile(`<a[^>]*href="([^"]*)"[^>]*>([^<]*)</a>`)
	matches := linkRegex.FindAllStringSubmatch(html, -1)

	// Check if selector looks like a class selector
	isClassSelector := strings.HasPrefix(source.Selector, ".")
	className := ""
	if isClassSelector {
		className = strings.TrimPrefix(source.Selector, ".")
	}

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		href := match[0]
		link := match[1]
		title := match[2]

		// Filter by class if specified
		if isClassSelector && !strings.Contains(href, className) {
			continue
		}

		// Make absolute URL if relative
		if !strings.HasPrefix(link, "http") {
			if u, err := url.Parse(source.URL); err == nil {
				if strings.HasPrefix(link, "/") {
					link = u.Scheme + "://" + u.Host + link
				} else {
					link = u.Scheme + "://" + u.Host + "/" + link
				}
			}
		}

		docs = append(docs, DocumentRef{
			URL:           link,
			Title:         strings.TrimSpace(title),
			PublishedAt:   time.Now(), // Unknown publish date
			SourceName:    source.Name,
			TargetLibrary: source.TargetLibrary,
		})
	}

	return docs
}

// filterAndDedup filters documents and removes duplicates.
func (m *WebSourceMonitor) filterAndDedup(docs []DocumentRef, source SourceConfig) ([]DocumentRef, error) {
	result := make([]DocumentRef, 0)

	for _, doc := range docs {
		// Check if already seen
		m.seenMu.RLock()
		seen := m.seen[doc.URL]
		m.seenMu.RUnlock()
		if seen {
			continue
		}

		// Apply filters
		if source.Filters != nil {
			if !m.matchesFilters(doc, source.Filters) {
				continue
			}
		}

		// Mark as seen
		m.seenMu.Lock()
		m.seen[doc.URL] = true
		m.seenMu.Unlock()

		result = append(result, doc)
	}

	return result, nil
}

// matchesFilters checks if a document matches the configured filters.
func (m *WebSourceMonitor) matchesFilters(doc DocumentRef, filters *FilterConfig) bool {
	// Title contains filter
	if filters.TitleContains != "" {
		if !strings.Contains(strings.ToLower(doc.Title), strings.ToLower(filters.TitleContains)) {
			return false
		}
	}

	// Title regex filter
	if filters.TitleRegex != "" {
		re, err := regexp.Compile(filters.TitleRegex)
		if err == nil && !re.MatchString(doc.Title) {
			return false
		}
	}

	// Category filter
	if filters.Category != "" {
		if cat, ok := doc.Metadata["category"]; ok {
			if strings.ToLower(cat) != strings.ToLower(filters.Category) {
				return false
			}
		}
	}

	// Age filters
	if filters.MinAge != "" {
		if dur, err := time.ParseDuration(filters.MinAge); err == nil {
			if time.Since(doc.PublishedAt) < dur {
				return false
			}
		}
	}
	if filters.MaxAge != "" {
		if dur, err := time.ParseDuration(filters.MaxAge); err == nil {
			if time.Since(doc.PublishedAt) > dur {
				return false
			}
		}
	}

	return true
}

// addAuth adds authentication to a request.
func (m *WebSourceMonitor) addAuth(req *http.Request, auth *AuthConfig) error {
	switch auth.Type {
	case AuthTypeBearer:
		token := os.Getenv(auth.TokenEnv)
		if token == "" {
			return fmt.Errorf("token env %s not set", auth.TokenEnv)
		}
		req.Header.Set("Authorization", "Bearer "+token)

	case AuthTypeBasic:
		username := os.Getenv(auth.UsernameEnv)
		password := os.Getenv(auth.PasswordEnv)
		if username == "" || password == "" {
			return fmt.Errorf("basic auth credentials not set")
		}
		req.SetBasicAuth(username, password)

	case AuthTypeAPIKey:
		token := os.Getenv(auth.TokenEnv)
		if token == "" {
			return fmt.Errorf("API key env %s not set", auth.TokenEnv)
		}
		headerName := auth.HeaderName
		if headerName == "" {
			headerName = "X-API-Key"
		}
		req.Header.Set(headerName, token)
	}

	return nil
}

// expandTemplateVars expands template variables in parameter values.
func (m *WebSourceMonitor) expandTemplateVars(value string, source SourceConfig) string {
	// Expand {{last_check}}
	if strings.Contains(value, "{{last_check}}") {
		m.statusMu.RLock()
		status := m.statuses[source.Name]
		m.statusMu.RUnlock()

		lastCheck := time.Now().Add(-24 * time.Hour) // Default to 24h ago
		if status != nil && !status.LastCheck.IsZero() {
			lastCheck = status.LastCheck
		}
		value = strings.ReplaceAll(value, "{{last_check}}", lastCheck.Format(time.RFC3339))
	}

	return value
}

// notifyCallbacks notifies all registered callbacks of a new document.
func (m *WebSourceMonitor) notifyCallbacks(doc DocumentRef) {
	m.callbackMu.RLock()
	defer m.callbackMu.RUnlock()

	for _, callback := range m.callbacks {
		// Errors from callbacks are logged but don't stop processing
		if err := callback(doc); err != nil {
			m.recordError(doc.SourceName, fmt.Sprintf("callback error: %v", err))
		}
	}
}

// recordError records an error for a source.
func (m *WebSourceMonitor) recordError(sourceName, errMsg string) {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()

	status, ok := m.statuses[sourceName]
	if !ok {
		status = &SourceStatusInfo{Name: sourceName}
		m.statuses[sourceName] = status
	}

	status.Status = SourceStatusError
	status.Errors = append(status.Errors, errMsg)

	// Keep only last 10 errors
	if len(status.Errors) > 10 {
		status.Errors = status.Errors[len(status.Errors)-10:]
	}
}

// setStatus sets the status for a source.
func (m *WebSourceMonitor) setStatus(sourceName string, info *SourceStatusInfo) {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	m.statuses[sourceName] = info
}

// parseRSSDate parses common RSS date formats.
func parseRSSDate(dateStr string) time.Time {
	formats := []string{
		time.RFC1123,
		time.RFC1123Z,
		time.RFC822,
		time.RFC822Z,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 MST",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	return time.Time{}
}

// parseAtomDate parses Atom/ISO date formats.
func parseAtomDate(dateStr string) time.Time {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	return time.Time{}
}

// parseRateLimit parses rate limit strings like "2/minute".
func parseRateLimit(rateLimit string) time.Duration {
	parts := strings.Split(rateLimit, "/")
	if len(parts) != 2 {
		return 30 * time.Second // Default
	}

	// Parse the unit
	unit := strings.TrimSpace(parts[1])
	var duration time.Duration
	switch strings.ToLower(unit) {
	case "second", "s":
		duration = time.Second
	case "minute", "m", "min":
		duration = time.Minute
	case "hour", "h":
		duration = time.Hour
	default:
		duration = time.Minute
	}

	// Parse the count
	count := 1
	if _, err := fmt.Sscanf(parts[0], "%d", &count); err != nil || count < 1 {
		count = 1
	}

	return duration / time.Duration(count)
}

// GetSeenURLs returns the set of seen URLs for persistence.
func (m *WebSourceMonitor) GetSeenURLs() []string {
	m.seenMu.RLock()
	defer m.seenMu.RUnlock()

	urls := make([]string, 0, len(m.seen))
	for url := range m.seen {
		urls = append(urls, url)
	}
	return urls
}

// LoadSeenURLs loads previously seen URLs.
func (m *WebSourceMonitor) LoadSeenURLs(urls []string) {
	m.seenMu.Lock()
	defer m.seenMu.Unlock()

	for _, url := range urls {
		m.seen[url] = true
	}
}

// ClearSeen clears the seen URL cache.
func (m *WebSourceMonitor) ClearSeen() {
	m.seenMu.Lock()
	defer m.seenMu.Unlock()
	m.seen = make(map[string]bool)
}
