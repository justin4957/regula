// Package linkcheck provides batch validation of external reference URIs with
// rate limiting, progress reporting, and report generation.
package linkcheck

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

// LinkStatus represents the validation status of a link.
type LinkStatus string

const (
	StatusValid    LinkStatus = "valid"
	StatusInvalid  LinkStatus = "invalid"
	StatusTimeout  LinkStatus = "timeout"
	StatusError    LinkStatus = "error"
	StatusSkipped  LinkStatus = "skipped"
	StatusPending  LinkStatus = "pending"
	StatusRedirect LinkStatus = "redirect"
)

// LinkResult captures the outcome of validating a single link.
type LinkResult struct {
	URI           string     `json:"uri"`
	Status        LinkStatus `json:"status"`
	StatusCode    int        `json:"status_code,omitempty"`
	RedirectURI   string     `json:"redirect_uri,omitempty"`
	Error         string     `json:"error,omitempty"`
	ResponseTime  int64      `json:"response_time_ms"`
	CheckedAt     time.Time  `json:"checked_at"`
	Domain        string     `json:"domain"`
	SourceContext string     `json:"source_context,omitempty"` // Where this link was found
}

// IsSuccess returns true if the link validated successfully.
func (linkResult *LinkResult) IsSuccess() bool {
	return linkResult.Status == StatusValid || linkResult.Status == StatusRedirect
}

// LinkInput represents a link to be validated with optional context.
type LinkInput struct {
	URI           string `json:"uri"`
	SourceContext string `json:"source_context,omitempty"` // E.g., "Article 5, paragraph 2"
}

// DomainConfig holds rate limiting configuration for a specific domain.
type DomainConfig struct {
	Domain       string        `json:"domain"`
	RateLimit    time.Duration `json:"rate_limit"`    // Minimum interval between requests
	MaxRetries   int           `json:"max_retries"`   // Number of retries on failure
	Timeout      time.Duration `json:"timeout"`       // Request timeout
	SkipValidate bool          `json:"skip_validate"` // Skip validation for this domain
}

// BatchConfig holds configuration for batch link validation.
type BatchConfig struct {
	// DefaultRateLimit is the minimum interval between requests for unknown domains.
	DefaultRateLimit time.Duration `json:"default_rate_limit"`

	// DefaultTimeout is the request timeout for unknown domains.
	DefaultTimeout time.Duration `json:"default_timeout"`

	// DefaultMaxRetries is the number of retries on failure for unknown domains.
	DefaultMaxRetries int `json:"default_max_retries"`

	// DomainConfigs holds per-domain rate limiting configuration.
	DomainConfigs map[string]*DomainConfig `json:"domain_configs"`

	// Concurrency is the maximum number of domains to process concurrently.
	Concurrency int `json:"concurrency"`

	// UserAgent is the User-Agent header sent with requests.
	UserAgent string `json:"user_agent"`

	// CacheTTL is the time-to-live for cached validation results.
	CacheTTL time.Duration `json:"cache_ttl"`

	// FollowRedirects determines whether to follow HTTP redirects.
	FollowRedirects bool `json:"follow_redirects"`
}

// DefaultBatchConfig returns a BatchConfig with sensible defaults.
func DefaultBatchConfig() *BatchConfig {
	return &BatchConfig{
		DefaultRateLimit:  1 * time.Second,
		DefaultTimeout:    30 * time.Second,
		DefaultMaxRetries: 2,
		DomainConfigs:     make(map[string]*DomainConfig),
		Concurrency:       3,
		UserAgent:         "regula-linkcheck/1.0",
		CacheTTL:          1 * time.Hour,
		FollowRedirects:   true,
	}
}

// WithDomainConfig adds or updates a domain-specific configuration.
func (batchConfig *BatchConfig) WithDomainConfig(domainConfig *DomainConfig) *BatchConfig {
	if batchConfig.DomainConfigs == nil {
		batchConfig.DomainConfigs = make(map[string]*DomainConfig)
	}
	batchConfig.DomainConfigs[domainConfig.Domain] = domainConfig
	return batchConfig
}

// GetDomainConfig returns the configuration for a domain, or a default config if not found.
func (batchConfig *BatchConfig) GetDomainConfig(domain string) *DomainConfig {
	if config, exists := batchConfig.DomainConfigs[domain]; exists {
		return config
	}
	return &DomainConfig{
		Domain:     domain,
		RateLimit:  batchConfig.DefaultRateLimit,
		MaxRetries: batchConfig.DefaultMaxRetries,
		Timeout:    batchConfig.DefaultTimeout,
	}
}

// ProgressCallback is called to report validation progress.
type ProgressCallback func(progress *ValidationProgress)

// ValidationProgress reports the current state of batch validation.
type ValidationProgress struct {
	TotalLinks     int       `json:"total_links"`
	CompletedLinks int       `json:"completed_links"`
	CurrentURI     string    `json:"current_uri,omitempty"`
	CurrentDomain  string    `json:"current_domain,omitempty"`
	StartedAt      time.Time `json:"started_at"`
	ElapsedTime    int64     `json:"elapsed_time_ms"`
	EstimatedLeft  int64     `json:"estimated_left_ms,omitempty"`
}

// PercentComplete returns the completion percentage.
func (validationProgress *ValidationProgress) PercentComplete() float64 {
	if validationProgress.TotalLinks == 0 {
		return 100.0
	}
	return float64(validationProgress.CompletedLinks) / float64(validationProgress.TotalLinks) * 100.0
}

// ValidationReport is the complete report of batch link validation.
type ValidationReport struct {
	// Summary statistics
	TotalLinks   int `json:"total_links"`
	ValidLinks   int `json:"valid_links"`
	InvalidLinks int `json:"invalid_links"`
	TimeoutLinks int `json:"timeout_links"`
	ErrorLinks   int `json:"error_links"`
	SkippedLinks int `json:"skipped_links"`

	// Timing
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	DurationMs  int64     `json:"duration_ms"`

	// Domain breakdown
	DomainStats map[string]*DomainStats `json:"domain_stats"`

	// Individual results
	Results []*LinkResult `json:"results"`

	// Broken links (convenience accessor)
	BrokenLinks []*LinkResult `json:"broken_links"`
}

// DomainStats holds statistics for a specific domain.
type DomainStats struct {
	Domain       string `json:"domain"`
	TotalLinks   int    `json:"total_links"`
	ValidLinks   int    `json:"valid_links"`
	InvalidLinks int    `json:"invalid_links"`
	TimeoutLinks int    `json:"timeout_links"`
	ErrorLinks   int    `json:"error_links"`
	AvgResponse  int64  `json:"avg_response_ms"`
}

// NewValidationReport creates an empty validation report.
func NewValidationReport() *ValidationReport {
	return &ValidationReport{
		DomainStats: make(map[string]*DomainStats),
		Results:     make([]*LinkResult, 0),
		BrokenLinks: make([]*LinkResult, 0),
	}
}

// AddResult adds a link result to the report and updates statistics.
func (validationReport *ValidationReport) AddResult(linkResult *LinkResult) {
	validationReport.Results = append(validationReport.Results, linkResult)
	validationReport.TotalLinks++

	switch linkResult.Status {
	case StatusValid, StatusRedirect:
		validationReport.ValidLinks++
	case StatusInvalid:
		validationReport.InvalidLinks++
		validationReport.BrokenLinks = append(validationReport.BrokenLinks, linkResult)
	case StatusTimeout:
		validationReport.TimeoutLinks++
		validationReport.BrokenLinks = append(validationReport.BrokenLinks, linkResult)
	case StatusError:
		validationReport.ErrorLinks++
		validationReport.BrokenLinks = append(validationReport.BrokenLinks, linkResult)
	case StatusSkipped:
		validationReport.SkippedLinks++
	}

	// Update domain stats
	domainStats, exists := validationReport.DomainStats[linkResult.Domain]
	if !exists {
		domainStats = &DomainStats{Domain: linkResult.Domain}
		validationReport.DomainStats[linkResult.Domain] = domainStats
	}
	domainStats.TotalLinks++

	switch linkResult.Status {
	case StatusValid, StatusRedirect:
		domainStats.ValidLinks++
	case StatusInvalid:
		domainStats.InvalidLinks++
	case StatusTimeout:
		domainStats.TimeoutLinks++
	case StatusError:
		domainStats.ErrorLinks++
	}

	// Update average response time
	if linkResult.ResponseTime > 0 && domainStats.TotalLinks > 0 {
		currentTotal := domainStats.AvgResponse * int64(domainStats.TotalLinks-1)
		domainStats.AvgResponse = (currentTotal + linkResult.ResponseTime) / int64(domainStats.TotalLinks)
	}
}

// Finalize completes the report with timing information.
func (validationReport *ValidationReport) Finalize() {
	validationReport.CompletedAt = time.Now()
	validationReport.DurationMs = validationReport.CompletedAt.Sub(validationReport.StartedAt).Milliseconds()

	// Sort results by URI for consistent output
	sort.Slice(validationReport.Results, func(i, j int) bool {
		return validationReport.Results[i].URI < validationReport.Results[j].URI
	})

	// Sort broken links by URI
	sort.Slice(validationReport.BrokenLinks, func(i, j int) bool {
		return validationReport.BrokenLinks[i].URI < validationReport.BrokenLinks[j].URI
	})
}

// SuccessRate returns the percentage of valid links.
func (validationReport *ValidationReport) SuccessRate() float64 {
	checkable := validationReport.TotalLinks - validationReport.SkippedLinks
	if checkable == 0 {
		return 100.0
	}
	return float64(validationReport.ValidLinks) / float64(checkable) * 100.0
}

// ToJSON serializes the report to JSON.
func (validationReport *ValidationReport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(validationReport, "", "  ")
}

// ToMarkdown generates a Markdown formatted report.
func (validationReport *ValidationReport) ToMarkdown() string {
	var markdownBuilder strings.Builder

	markdownBuilder.WriteString("# Link Validation Report\n\n")

	// Summary
	markdownBuilder.WriteString("## Summary\n\n")
	markdownBuilder.WriteString(fmt.Sprintf("- **Total Links**: %d\n", validationReport.TotalLinks))
	markdownBuilder.WriteString(fmt.Sprintf("- **Valid Links**: %d\n", validationReport.ValidLinks))
	markdownBuilder.WriteString(fmt.Sprintf("- **Invalid Links**: %d\n", validationReport.InvalidLinks))
	markdownBuilder.WriteString(fmt.Sprintf("- **Timeout Links**: %d\n", validationReport.TimeoutLinks))
	markdownBuilder.WriteString(fmt.Sprintf("- **Error Links**: %d\n", validationReport.ErrorLinks))
	markdownBuilder.WriteString(fmt.Sprintf("- **Skipped Links**: %d\n", validationReport.SkippedLinks))
	markdownBuilder.WriteString(fmt.Sprintf("- **Success Rate**: %.1f%%\n", validationReport.SuccessRate()))
	markdownBuilder.WriteString(fmt.Sprintf("- **Duration**: %dms\n\n", validationReport.DurationMs))

	// Domain breakdown
	if len(validationReport.DomainStats) > 0 {
		markdownBuilder.WriteString("## Domain Statistics\n\n")
		markdownBuilder.WriteString("| Domain | Total | Valid | Invalid | Timeout | Error | Avg Response |\n")
		markdownBuilder.WriteString("|--------|-------|-------|---------|---------|-------|-------------|\n")

		// Sort domains for consistent output
		domains := make([]string, 0, len(validationReport.DomainStats))
		for domain := range validationReport.DomainStats {
			domains = append(domains, domain)
		}
		sort.Strings(domains)

		for _, domain := range domains {
			domainStats := validationReport.DomainStats[domain]
			markdownBuilder.WriteString(fmt.Sprintf("| %s | %d | %d | %d | %d | %d | %dms |\n",
				domain, domainStats.TotalLinks, domainStats.ValidLinks, domainStats.InvalidLinks,
				domainStats.TimeoutLinks, domainStats.ErrorLinks, domainStats.AvgResponse))
		}
		markdownBuilder.WriteString("\n")
	}

	// Broken links
	if len(validationReport.BrokenLinks) > 0 {
		markdownBuilder.WriteString("## Broken Links\n\n")
		markdownBuilder.WriteString("| URI | Status | Error | Source |\n")
		markdownBuilder.WriteString("|-----|--------|-------|--------|\n")

		for _, linkResult := range validationReport.BrokenLinks {
			errorStr := linkResult.Error
			if errorStr == "" && linkResult.StatusCode > 0 {
				errorStr = fmt.Sprintf("HTTP %d", linkResult.StatusCode)
			}
			sourceCtx := linkResult.SourceContext
			if sourceCtx == "" {
				sourceCtx = "-"
			}
			markdownBuilder.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				linkResult.URI, linkResult.Status, errorStr, sourceCtx))
		}
		markdownBuilder.WriteString("\n")
	}

	return markdownBuilder.String()
}

// String returns a human-readable summary of the report.
func (validationReport *ValidationReport) String() string {
	var summaryBuilder strings.Builder

	summaryBuilder.WriteString("Link Validation Report\n")
	summaryBuilder.WriteString("======================\n\n")
	summaryBuilder.WriteString(fmt.Sprintf("Total links:   %d\n", validationReport.TotalLinks))
	summaryBuilder.WriteString(fmt.Sprintf("Valid:         %d\n", validationReport.ValidLinks))
	summaryBuilder.WriteString(fmt.Sprintf("Invalid:       %d\n", validationReport.InvalidLinks))
	summaryBuilder.WriteString(fmt.Sprintf("Timeout:       %d\n", validationReport.TimeoutLinks))
	summaryBuilder.WriteString(fmt.Sprintf("Error:         %d\n", validationReport.ErrorLinks))
	summaryBuilder.WriteString(fmt.Sprintf("Skipped:       %d\n", validationReport.SkippedLinks))
	summaryBuilder.WriteString(fmt.Sprintf("Success rate:  %.1f%%\n", validationReport.SuccessRate()))
	summaryBuilder.WriteString(fmt.Sprintf("Duration:      %dms\n", validationReport.DurationMs))

	if len(validationReport.BrokenLinks) > 0 {
		summaryBuilder.WriteString(fmt.Sprintf("\nBroken links (%d):\n", len(validationReport.BrokenLinks)))
		for _, linkResult := range validationReport.BrokenLinks {
			errorStr := linkResult.Error
			if errorStr == "" && linkResult.StatusCode > 0 {
				errorStr = fmt.Sprintf("HTTP %d", linkResult.StatusCode)
			}
			summaryBuilder.WriteString(fmt.Sprintf("  - %s: %s (%s)\n", linkResult.URI, linkResult.Status, errorStr))
		}
	}

	return summaryBuilder.String()
}

// ExtractDomain extracts the domain from a URI.
func ExtractDomain(uri string) string {
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return ""
	}
	return parsedURL.Host
}
