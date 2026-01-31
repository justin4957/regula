package crawler

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ContentFetcher fetches web content with per-domain rate limiting,
// HTML-to-text conversion, and caching support.
type ContentFetcher struct {
	httpClient   *http.Client
	domainTimers map[string]time.Time
	timerMu      sync.Mutex
	config       CrawlConfig
	maxBodyBytes int64
}

// NewContentFetcher creates a ContentFetcher with the given configuration.
func NewContentFetcher(config CrawlConfig) *ContentFetcher {
	httpClient := &http.Client{
		Timeout: config.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	return &ContentFetcher{
		httpClient:   httpClient,
		domainTimers: make(map[string]time.Time),
		config:       config,
		maxBodyBytes: 10 * 1024 * 1024, // 10MB max
	}
}

// Fetch retrieves content from the given URL, respecting rate limits.
// Returns extracted plain text suitable for the ingestion pipeline.
func (fetcher *ContentFetcher) Fetch(targetURL string) (*FetchedContent, error) {
	if targetURL == "" {
		return nil, fmt.Errorf("empty URL")
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %s: %w", targetURL, err)
	}

	// Rate limit per domain
	fetcher.waitForDomain(parsedURL.Host)

	userAgent := fetcher.config.UserAgent
	if userAgent == "" {
		userAgent = DefaultCrawlUserAgent
	}

	// Check for domain-specific user agent
	if domainConfig, hasDomainConfig := fetcher.config.DomainConfigs[parsedURL.Host]; hasDomainConfig && domainConfig.UserAgent != "" {
		userAgent = domainConfig.UserAgent
	}

	request, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", targetURL, err)
	}
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Accept", "text/html, text/plain, application/xhtml+xml")

	response, err := fetcher.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", targetURL, err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		return &FetchedContent{
			URL:         targetURL,
			StatusCode:  response.StatusCode,
			ContentType: response.Header.Get("Content-Type"),
			FetchedAt:   time.Now(),
		}, fmt.Errorf("HTTP %d for %s", response.StatusCode, targetURL)
	}

	limitedReader := io.LimitReader(response.Body, fetcher.maxBodyBytes)
	rawBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read body from %s: %w", targetURL, err)
	}

	contentType := response.Header.Get("Content-Type")
	var plainText []byte

	if strings.Contains(contentType, "text/html") || strings.Contains(contentType, "application/xhtml") {
		plainText = ExtractTextFromHTML(rawBody)
	} else if strings.Contains(contentType, "text/plain") {
		plainText = rawBody
	} else {
		// Attempt HTML extraction as fallback
		plainText = ExtractTextFromHTML(rawBody)
	}

	return &FetchedContent{
		URL:         targetURL,
		RawHTML:     rawBody,
		PlainText:   plainText,
		ContentType: contentType,
		StatusCode:  response.StatusCode,
		FetchedAt:   time.Now(),
	}, nil
}

// waitForDomain enforces per-domain rate limiting.
func (fetcher *ContentFetcher) waitForDomain(domain string) {
	fetcher.timerMu.Lock()

	rateLimit := fetcher.config.RateLimit
	if domainConfig, hasDomainConfig := fetcher.config.DomainConfigs[domain]; hasDomainConfig && domainConfig.RateLimit > 0 {
		rateLimit = domainConfig.RateLimit
	}

	lastRequestTime, hasLastRequest := fetcher.domainTimers[domain]
	if hasLastRequest {
		elapsed := time.Since(lastRequestTime)
		if elapsed < rateLimit {
			waitDuration := rateLimit - elapsed
			fetcher.timerMu.Unlock()
			time.Sleep(waitDuration)
			fetcher.timerMu.Lock()
		}
	}

	fetcher.domainTimers[domain] = time.Now()
	fetcher.timerMu.Unlock()
}

// Pre-compiled regex patterns for HTML-to-text conversion.
var (
	reScript     = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle      = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reComment    = regexp.MustCompile(`(?s)<!--.*?-->`)
	reNav        = regexp.MustCompile(`(?is)<nav[^>]*>.*?</nav>`)
	reHeader     = regexp.MustCompile(`(?is)<header[^>]*>.*?</header>`)
	reFooter     = regexp.MustCompile(`(?is)<footer[^>]*>.*?</footer>`)
	reHeading    = regexp.MustCompile(`(?i)<h[1-6][^>]*>`)
	reHeadingEnd = regexp.MustCompile(`(?i)</h[1-6]>`)
	rePara       = regexp.MustCompile(`(?i)<(?:p|div|br|tr)[^>]*/?>`)
	reListItem   = regexp.MustCompile(`(?i)<li[^>]*>`)
	reTag        = regexp.MustCompile(`<[^>]+>`)
	reMultiNL    = regexp.MustCompile(`\n{3,}`)
	reMultiSpace = regexp.MustCompile(`[^\S\n]{2,}`)
)

// ExtractTextFromHTML converts raw HTML to plain text preserving document structure.
// Uses Go stdlib only (regexp + strings), no external HTML parser dependency.
func ExtractTextFromHTML(rawHTML []byte) []byte {
	content := string(rawHTML)

	// Extract body content if present
	bodyStart := strings.Index(strings.ToLower(content), "<body")
	if bodyStart >= 0 {
		bodyTagEnd := strings.Index(content[bodyStart:], ">")
		if bodyTagEnd >= 0 {
			content = content[bodyStart+bodyTagEnd+1:]
		}
	}
	bodyEnd := strings.Index(strings.ToLower(content), "</body>")
	if bodyEnd >= 0 {
		content = content[:bodyEnd]
	}

	// Remove non-content elements
	content = reScript.ReplaceAllString(content, "")
	content = reStyle.ReplaceAllString(content, "")
	content = reComment.ReplaceAllString(content, "")
	content = reNav.ReplaceAllString(content, "")
	content = reHeader.ReplaceAllString(content, "")
	content = reFooter.ReplaceAllString(content, "")

	// Convert structural elements to newlines
	content = reHeading.ReplaceAllString(content, "\n\n")
	content = reHeadingEnd.ReplaceAllString(content, "\n")
	content = reListItem.ReplaceAllString(content, "\n- ")
	content = rePara.ReplaceAllString(content, "\n")

	// Remove all remaining HTML tags
	content = reTag.ReplaceAllString(content, "")

	// Decode common HTML entities
	content = strings.ReplaceAll(content, "&amp;", "&")
	content = strings.ReplaceAll(content, "&lt;", "<")
	content = strings.ReplaceAll(content, "&gt;", ">")
	content = strings.ReplaceAll(content, "&quot;", "\"")
	content = strings.ReplaceAll(content, "&nbsp;", " ")
	content = strings.ReplaceAll(content, "&#39;", "'")

	// Clean up whitespace
	content = reMultiSpace.ReplaceAllString(content, " ")
	content = reMultiNL.ReplaceAllString(content, "\n\n")
	content = strings.TrimSpace(content)

	return []byte(content)
}
