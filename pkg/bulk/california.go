package bulk

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// CaliforniaSource downloads California code text from leginfo.legislature.ca.gov.
// Since Go stdlib has no FTP client, this adapter scrapes the HTTP site to
// download code text division-by-division.
type CaliforniaSource struct {
	config     DownloadConfig
	httpClient *http.Client
}

// NewCaliforniaSource creates a CaliforniaSource with the given config.
func NewCaliforniaSource(config DownloadConfig) *CaliforniaSource {
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: config.Timeout}
	}
	return &CaliforniaSource{config: config, httpClient: httpClient}
}

func (source *CaliforniaSource) Name() string { return "california" }

func (source *CaliforniaSource) Description() string {
	return "California codes from leginfo.legislature.ca.gov (29 codes + Constitution)"
}

// ListDatasets returns all 30 California codes as downloadable datasets.
func (source *CaliforniaSource) ListDatasets() ([]Dataset, error) {
	var datasets []Dataset

	for _, codeEntry := range californiaCodeEntries {
		datasets = append(datasets, Dataset{
			SourceName:   "california",
			Identifier:   fmt.Sprintf("ca-%s", strings.ToLower(codeEntry.Abbreviation)),
			DisplayName:  fmt.Sprintf("%s (%s)", codeEntry.FullName, codeEntry.Abbreviation),
			URL:          fmt.Sprintf("%s?tocCode=%s", californiaBaseURL, codeEntry.Abbreviation),
			Format:       "html",
			Jurisdiction: "US-CA",
		})
	}

	return datasets, nil
}

// DownloadDataset downloads a California code by scraping the TOC and
// fetching expanded branch text for each top-level division.
func (source *CaliforniaSource) DownloadDataset(dataset Dataset, downloader *Downloader) (*DownloadResult, error) {
	sourceDir := downloader.SourceDirectory("california")
	codeAbbrev := strings.TrimPrefix(dataset.Identifier, "ca-")
	codeAbbrev = strings.ToUpper(codeAbbrev)
	localPath := filepath.Join(sourceDir, codeAbbrev+".txt")

	// Check if already downloaded
	existingInfo, err := os.Stat(localPath)
	if err == nil && existingInfo.Size() > 0 {
		return &DownloadResult{
			Dataset:      dataset,
			LocalPath:    localPath,
			BytesWritten: existingInfo.Size(),
			Skipped:      true,
			DownloadedAt: time.Now(),
		}, nil
	}

	os.MkdirAll(sourceDir, 0755)

	// Fetch the TOC page to discover divisions
	tocURL := fmt.Sprintf("%s?tocCode=%s&tocTitle=+%s",
		californiaBaseURL, codeAbbrev,
		url.QueryEscape(source.codeFullName(codeAbbrev)))

	downloader.waitForDomain("leginfo.legislature.ca.gov")

	tocRequest, err := http.NewRequest(http.MethodGet, tocURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create TOC request: %w", err)
	}
	tocRequest.Header.Set("User-Agent", downloader.config.UserAgent)

	tocResponse, err := source.httpClient.Do(tocRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch TOC for %s: %w", codeAbbrev, err)
	}
	defer tocResponse.Body.Close()

	if tocResponse.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d fetching TOC for %s", tocResponse.StatusCode, codeAbbrev)
	}

	tocBody, err := io.ReadAll(tocResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read TOC body: %w", err)
	}

	// Extract division/heading links from the TOC page
	branchURLs := extractBranchURLs(string(tocBody), codeAbbrev)

	if len(branchURLs) == 0 {
		// Fallback: try fetching the entire code as a single expanded branch
		branchURLs = []string{fmt.Sprintf(
			"https://leginfo.legislature.ca.gov/faces/codes_displayexpandedbranch.xhtml?tocCode=%s",
			codeAbbrev)}
	}

	// Fetch each branch and concatenate text
	var allText strings.Builder
	allText.WriteString(fmt.Sprintf("CALIFORNIA %s\n\n", source.codeFullName(codeAbbrev)))

	for _, branchURL := range branchURLs {
		downloader.waitForDomain("leginfo.legislature.ca.gov")

		branchRequest, err := http.NewRequest(http.MethodGet, branchURL, nil)
		if err != nil {
			continue
		}
		branchRequest.Header.Set("User-Agent", downloader.config.UserAgent)

		branchResponse, err := source.httpClient.Do(branchRequest)
		if err != nil {
			continue
		}

		branchBody, err := io.ReadAll(branchResponse.Body)
		branchResponse.Body.Close()
		if err != nil {
			continue
		}

		// Extract text content from HTML
		plainText := extractCaliforniaText(branchBody)
		if plainText != "" {
			allText.WriteString(plainText)
			allText.WriteString("\n\n")
		}
	}

	// Write combined text to file
	codeText := allText.String()
	if err := os.WriteFile(localPath, []byte(codeText), 0644); err != nil {
		return nil, fmt.Errorf("failed to write %s: %w", localPath, err)
	}

	bytesWritten := int64(len(codeText))

	downloader.Manifest().RecordDownload(&DownloadRecord{
		Identifier:   dataset.Identifier,
		SourceName:   "california",
		URL:          tocURL,
		LocalPath:    localPath,
		SizeBytes:    bytesWritten,
		DownloadedAt: time.Now(),
	})
	downloader.SaveManifest()

	return &DownloadResult{
		Dataset:      dataset,
		LocalPath:    localPath,
		BytesWritten: bytesWritten,
		Skipped:      false,
		DownloadedAt: time.Now(),
	}, nil
}

func (source *CaliforniaSource) codeFullName(abbreviation string) string {
	for _, entry := range californiaCodeEntries {
		if entry.Abbreviation == abbreviation {
			return entry.FullName
		}
	}
	return abbreviation
}

// Pre-compiled regex patterns for California HTML parsing.
var (
	reBranchHref = regexp.MustCompile(`href="[^"]*codes_displayexpandedbranch\.xhtml\?tocCode=([A-Z]+)[^"]*"`)
	reCAScript   = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reCAStyle    = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reCATag      = regexp.MustCompile(`<[^>]+>`)
	reCAMultiNL  = regexp.MustCompile(`\n{3,}`)
	reCAMultiSp  = regexp.MustCompile(`[^\S\n]{2,}`)
)

// extractBranchURLs finds expanded branch URLs from a TOC page.
func extractBranchURLs(tocHTML string, codeAbbrev string) []string {
	matches := reBranchHref.FindAllStringSubmatch(tocHTML, -1)
	var urls []string
	seen := make(map[string]bool)

	for _, match := range matches {
		if match[1] == codeAbbrev {
			fullURL := "https://leginfo.legislature.ca.gov/faces/" + strings.TrimPrefix(match[0], `href="`)
			// Clean up the URL
			if idx := strings.Index(fullURL, `"`); idx >= 0 {
				fullURL = fullURL[:idx]
			}
			if !seen[fullURL] {
				seen[fullURL] = true
				urls = append(urls, fullURL)
			}
		}
	}

	return urls
}

// extractCaliforniaText strips HTML tags from California legislature HTML
// and returns plain text preserving section structure.
func extractCaliforniaText(rawHTML []byte) string {
	content := string(rawHTML)
	content = reCAScript.ReplaceAllString(content, "")
	content = reCAStyle.ReplaceAllString(content, "")
	content = reCATag.ReplaceAllString(content, "\n")
	content = strings.ReplaceAll(content, "&amp;", "&")
	content = strings.ReplaceAll(content, "&lt;", "<")
	content = strings.ReplaceAll(content, "&gt;", ">")
	content = strings.ReplaceAll(content, "&quot;", `"`)
	content = strings.ReplaceAll(content, "&nbsp;", " ")
	content = strings.ReplaceAll(content, "&#39;", "'")
	content = reCAMultiSp.ReplaceAllString(content, " ")
	content = reCAMultiNL.ReplaceAllString(content, "\n\n")
	return strings.TrimSpace(content)
}

const californiaBaseURL = "https://leginfo.legislature.ca.gov/faces/codesTOCSelected.xhtml"

// californiaCodeEntries contains all 30 California codes (29 codes + Constitution).
var californiaCodeEntries = []struct {
	Abbreviation string
	FullName     string
}{
	{"CONS", "California Constitution"},
	{"BPC", "Business and Professions Code"},
	{"CIV", "Civil Code"},
	{"CCP", "Code of Civil Procedure"},
	{"COM", "Commercial Code"},
	{"CORP", "Corporations Code"},
	{"EDC", "Education Code"},
	{"ELEC", "Elections Code"},
	{"EVID", "Evidence Code"},
	{"FAM", "Family Code"},
	{"FIN", "Financial Code"},
	{"FGC", "Fish and Game Code"},
	{"FAC", "Food and Agricultural Code"},
	{"GOV", "Government Code"},
	{"HNC", "Harbors and Navigation Code"},
	{"HSC", "Health and Safety Code"},
	{"INS", "Insurance Code"},
	{"LAB", "Labor Code"},
	{"MVC", "Military and Veterans Code"},
	{"PEN", "Penal Code"},
	{"PROB", "Probate Code"},
	{"PCC", "Public Contract Code"},
	{"PRC", "Public Resources Code"},
	{"PUC", "Public Utilities Code"},
	{"RTC", "Revenue and Taxation Code"},
	{"SHC", "Streets and Highways Code"},
	{"UIC", "Unemployment Insurance Code"},
	{"VEH", "Vehicle Code"},
	{"WAT", "Water Code"},
	{"WIC", "Welfare and Institutions Code"},
}
