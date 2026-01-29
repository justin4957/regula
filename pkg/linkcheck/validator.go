package linkcheck

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// BatchValidator validates multiple URIs with per-domain rate limiting.
type BatchValidator struct {
	config          *BatchConfig
	cache           *LinkCache
	domainLimiters  *DomainRateLimiter
	httpClient      HTTPClient
	progressCb      ProgressCallback
	mu              sync.Mutex
}

// NewBatchValidator creates a new batch validator with the given configuration.
func NewBatchValidator(config *BatchConfig) *BatchValidator {
	if config == nil {
		config = DefaultBatchConfig()
	}

	// Create base HTTP client
	baseClient := &http.Client{
		Timeout: config.DefaultTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !config.FollowRedirects {
				return http.ErrUseLastResponse
			}
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	return &BatchValidator{
		config:         config,
		cache:          NewLinkCache(config.CacheTTL),
		domainLimiters: NewDomainRateLimiter(baseClient, config),
		httpClient:     baseClient,
	}
}

// SetProgressCallback sets a callback function to receive progress updates.
func (batchValidator *BatchValidator) SetProgressCallback(callback ProgressCallback) {
	batchValidator.mu.Lock()
	batchValidator.progressCb = callback
	batchValidator.mu.Unlock()
}

// ValidateLinks validates multiple links and returns a comprehensive report.
func (batchValidator *BatchValidator) ValidateLinks(links []LinkInput) *ValidationReport {
	return batchValidator.ValidateLinksWithContext(context.Background(), links)
}

// ValidateLinksWithContext validates multiple links with cancellation support.
func (batchValidator *BatchValidator) ValidateLinksWithContext(ctx context.Context, links []LinkInput) *ValidationReport {
	report := NewValidationReport()
	report.StartedAt = time.Now()

	if len(links) == 0 {
		report.Finalize()
		return report
	}

	// Group links by domain for efficient rate-limited processing
	domainGroups := batchValidator.groupByDomain(links)

	// Create channels for concurrent processing
	resultChan := make(chan *LinkResult, len(links))
	var wg sync.WaitGroup

	// Semaphore for limiting concurrent domain processing
	semaphore := make(chan struct{}, batchValidator.config.Concurrency)

	// Track progress
	completedCount := 0
	totalCount := len(links)
	progressMu := sync.Mutex{}

	// Process each domain group concurrently
	for domain, domainLinks := range domainGroups {
		wg.Add(1)
		go func(domain string, domainLinks []LinkInput) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check if domain should be skipped
			domainConfig := batchValidator.config.GetDomainConfig(domain)
			if domainConfig.SkipValidate {
				for _, link := range domainLinks {
					select {
					case <-ctx.Done():
						return
					default:
						result := &LinkResult{
							URI:       link.URI,
							Status:    StatusSkipped,
							CheckedAt: time.Now(),
							Domain:    domain,
							SourceContext: link.SourceContext,
						}
						resultChan <- result

						progressMu.Lock()
						completedCount++
						batchValidator.reportProgress(totalCount, completedCount, link.URI, domain)
						progressMu.Unlock()
					}
				}
				return
			}

			// Process links for this domain
			for _, link := range domainLinks {
				select {
				case <-ctx.Done():
					return
				default:
					result := batchValidator.validateSingleLink(ctx, link, domain, domainConfig)
					resultChan <- result

					progressMu.Lock()
					completedCount++
					batchValidator.reportProgress(totalCount, completedCount, link.URI, domain)
					progressMu.Unlock()
				}
			}
		}(domain, domainLinks)
	}

	// Close result channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for result := range resultChan {
		report.AddResult(result)
	}

	report.Finalize()
	return report
}

// ValidateSingleURI validates a single URI and returns the result.
func (batchValidator *BatchValidator) ValidateSingleURI(uri string) *LinkResult {
	return batchValidator.ValidateSingleURIWithContext(context.Background(), uri)
}

// ValidateSingleURIWithContext validates a single URI with cancellation support.
func (batchValidator *BatchValidator) ValidateSingleURIWithContext(ctx context.Context, uri string) *LinkResult {
	domain := ExtractDomain(uri)
	domainConfig := batchValidator.config.GetDomainConfig(domain)
	link := LinkInput{URI: uri}
	return batchValidator.validateSingleLink(ctx, link, domain, domainConfig)
}

// validateSingleLink validates a single link with retry logic.
func (batchValidator *BatchValidator) validateSingleLink(ctx context.Context, link LinkInput, domain string, domainConfig *DomainConfig) *LinkResult {
	// Check cache first
	if cached, found := batchValidator.cache.Get(link.URI); found {
		// Copy cached result to preserve source context
		result := *cached
		result.SourceContext = link.SourceContext
		return &result
	}

	var lastResult *LinkResult
	maxRetries := domainConfig.MaxRetries
	if maxRetries == 0 {
		maxRetries = batchValidator.config.DefaultMaxRetries
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff between retries
			backoff := time.Duration(attempt*attempt) * 500 * time.Millisecond
			select {
			case <-ctx.Done():
				return &LinkResult{
					URI:           link.URI,
					Status:        StatusError,
					Error:         "cancelled",
					CheckedAt:     time.Now(),
					Domain:        domain,
					SourceContext: link.SourceContext,
				}
			case <-time.After(backoff):
			}
		}

		lastResult = batchValidator.doValidation(ctx, link, domain, domainConfig)

		// Don't retry on success or definitive failures
		if lastResult.Status == StatusValid ||
			lastResult.Status == StatusRedirect ||
			lastResult.Status == StatusInvalid {
			break
		}
	}

	// Cache the result
	batchValidator.cache.Set(link.URI, lastResult)

	return lastResult
}

// doValidation performs the actual HTTP validation.
func (batchValidator *BatchValidator) doValidation(ctx context.Context, link LinkInput, domain string, domainConfig *DomainConfig) *LinkResult {
	startTime := time.Now()

	// Get rate-limited client for this domain
	rateLimitedClient := batchValidator.domainLimiters.GetClient(domain)

	// Create request with context
	request, err := http.NewRequestWithContext(ctx, http.MethodHead, link.URI, nil)
	if err != nil {
		return &LinkResult{
			URI:           link.URI,
			Status:        StatusError,
			Error:         fmt.Sprintf("failed to create request: %v", err),
			CheckedAt:     time.Now(),
			Domain:        domain,
			SourceContext: link.SourceContext,
			ResponseTime:  time.Since(startTime).Milliseconds(),
		}
	}

	request.Header.Set("User-Agent", batchValidator.config.UserAgent)

	// Execute request with timeout
	timeout := domainConfig.Timeout
	if timeout == 0 {
		timeout = batchValidator.config.DefaultTimeout
	}

	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	request = request.WithContext(timeoutCtx)

	response, err := rateLimitedClient.Do(request)
	responseTime := time.Since(startTime).Milliseconds()

	if err != nil {
		// Check if it's a timeout
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return &LinkResult{
				URI:           link.URI,
				Status:        StatusTimeout,
				Error:         "request timed out",
				CheckedAt:     time.Now(),
				Domain:        domain,
				SourceContext: link.SourceContext,
				ResponseTime:  responseTime,
			}
		}

		return &LinkResult{
			URI:           link.URI,
			Status:        StatusError,
			Error:         err.Error(),
			CheckedAt:     time.Now(),
			Domain:        domain,
			SourceContext: link.SourceContext,
			ResponseTime:  responseTime,
		}
	}
	defer response.Body.Close()

	result := &LinkResult{
		URI:           link.URI,
		StatusCode:    response.StatusCode,
		CheckedAt:     time.Now(),
		Domain:        domain,
		SourceContext: link.SourceContext,
		ResponseTime:  responseTime,
	}

	// Determine status based on HTTP status code
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		result.Status = StatusValid
	} else if response.StatusCode >= 300 && response.StatusCode < 400 {
		result.Status = StatusRedirect
		if location := response.Header.Get("Location"); location != "" {
			result.RedirectURI = location
		}
	} else if response.StatusCode >= 400 {
		result.Status = StatusInvalid
		result.Error = fmt.Sprintf("HTTP %d", response.StatusCode)
	} else {
		result.Status = StatusError
		result.Error = fmt.Sprintf("unexpected status code: %d", response.StatusCode)
	}

	return result
}

// groupByDomain groups links by their domain for efficient processing.
func (batchValidator *BatchValidator) groupByDomain(links []LinkInput) map[string][]LinkInput {
	groups := make(map[string][]LinkInput)
	for _, link := range links {
		domain := ExtractDomain(link.URI)
		if domain == "" {
			domain = "unknown"
		}
		groups[domain] = append(groups[domain], link)
	}
	return groups
}

// reportProgress sends a progress update via the callback if set.
func (batchValidator *BatchValidator) reportProgress(total, completed int, currentURI, currentDomain string) {
	batchValidator.mu.Lock()
	callback := batchValidator.progressCb
	batchValidator.mu.Unlock()

	if callback == nil {
		return
	}

	progress := &ValidationProgress{
		TotalLinks:     total,
		CompletedLinks: completed,
		CurrentURI:     currentURI,
		CurrentDomain:  currentDomain,
		StartedAt:      time.Now(), // This should ideally be tracked from start
	}

	// Estimate remaining time
	if completed > 0 && completed < total {
		avgTimePerLink := time.Since(progress.StartedAt).Milliseconds() / int64(completed)
		progress.EstimatedLeft = avgTimePerLink * int64(total-completed)
	}

	callback(progress)
}

// ClearCache clears the validation cache.
func (batchValidator *BatchValidator) ClearCache() {
	batchValidator.cache.Clear()
}

// CacheSize returns the number of cached entries.
func (batchValidator *BatchValidator) CacheSize() int {
	return batchValidator.cache.Len()
}

// ValidateURIStrings is a convenience method that accepts simple URI strings.
func (batchValidator *BatchValidator) ValidateURIStrings(uris []string) *ValidationReport {
	links := make([]LinkInput, len(uris))
	for i, uri := range uris {
		links[i] = LinkInput{URI: uri}
	}
	return batchValidator.ValidateLinks(links)
}
