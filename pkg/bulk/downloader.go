package bulk

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Downloader provides shared download infrastructure: HTTP fetching with
// progress reporting, per-domain rate limiting, archive extraction, and
// resume support via a persistent manifest.
type Downloader struct {
	config       DownloadConfig
	httpClient   *http.Client
	domainTimers map[string]time.Time
	timerMu      sync.Mutex
	manifest     *DownloadManifest
	manifestPath string
}

// NewDownloader creates a Downloader with the given config.
// Initializes the download directory and loads any existing manifest.
func NewDownloader(config DownloadConfig) (*Downloader, error) {
	if err := os.MkdirAll(config.DownloadDirectory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create download directory: %w", err)
	}

	manifestPath := filepath.Join(config.DownloadDirectory, "manifest.json")
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: config.Timeout,
			CheckRedirect: func(request *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		}
	}

	return &Downloader{
		config:       config,
		httpClient:   httpClient,
		domainTimers: make(map[string]time.Time),
		manifest:     manifest,
		manifestPath: manifestPath,
	}, nil
}

// DownloadFile fetches a URL to a local file path with progress reporting.
// Skips the download if the file already exists with non-zero size.
// Retries transient errors (5xx, timeouts) with exponential backoff.
func (downloader *Downloader) DownloadFile(downloadURL string, localPath string, progressCallback ProgressCallback) (int64, bool, error) {
	// Check if file already exists
	existingInfo, err := os.Stat(localPath)
	if err == nil && existingInfo.Size() > 0 {
		return existingInfo.Size(), true, nil
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return 0, false, fmt.Errorf("failed to create directory for %s: %w", localPath, err)
	}

	maxRetries := downloader.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}
	retryDelay := downloader.config.RetryBaseDelay
	if retryDelay <= 0 {
		retryDelay = 5 * time.Second
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			currentDelay := retryDelay * time.Duration(1<<uint(attempt-1))
			time.Sleep(currentDelay)
		}

		bytesWritten, err := downloader.downloadFileAttempt(downloadURL, localPath, progressCallback)
		if err == nil {
			return bytesWritten, false, nil
		}

		lastErr = err

		// Only retry on transient errors (5xx, network errors)
		if !isRetryableError(err) {
			return 0, false, err
		}

		// Clean up partial file before retry
		os.Remove(localPath)
	}

	return 0, false, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// downloadFileAttempt performs a single download attempt.
func (downloader *Downloader) downloadFileAttempt(downloadURL string, localPath string, progressCallback ProgressCallback) (int64, error) {
	// Rate limit per domain
	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return 0, fmt.Errorf("invalid URL %s: %w", downloadURL, err)
	}
	downloader.waitForDomain(parsedURL.Host)

	// Create HTTP request
	request, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	request.Header.Set("User-Agent", downloader.config.UserAgent)

	response, err := downloader.httpClient.Do(request)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch %s: %w", downloadURL, err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 500 {
		return 0, &retryableHTTPError{StatusCode: response.StatusCode, URL: downloadURL}
	}
	if response.StatusCode >= 400 {
		return 0, fmt.Errorf("HTTP %d for %s", response.StatusCode, downloadURL)
	}

	// Create output file
	outputFile, err := os.Create(localPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create file %s: %w", localPath, err)
	}
	defer outputFile.Close()

	// Stream with progress reporting
	totalBytes := response.ContentLength
	var bytesWritten int64

	buffer := make([]byte, 32*1024) // 32KB buffer
	for {
		bytesRead, readErr := response.Body.Read(buffer)
		if bytesRead > 0 {
			written, writeErr := outputFile.Write(buffer[:bytesRead])
			if writeErr != nil {
				return bytesWritten, fmt.Errorf("write error: %w", writeErr)
			}
			bytesWritten += int64(written)

			if progressCallback != nil {
				progressCallback(bytesWritten, totalBytes)
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return bytesWritten, fmt.Errorf("read error: %w", readErr)
		}
	}

	return bytesWritten, nil
}

// retryableHTTPError represents an HTTP error that should trigger a retry.
type retryableHTTPError struct {
	StatusCode int
	URL        string
}

func (e *retryableHTTPError) Error() string {
	return fmt.Sprintf("HTTP %d for %s", e.StatusCode, e.URL)
}

// isRetryableError returns true if the error warrants a retry attempt.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	// Retry on 5xx HTTP errors
	if _, ok := err.(*retryableHTTPError); ok {
		return true
	}
	// Retry on network errors (connection reset, timeout, etc.)
	errMsg := err.Error()
	retryablePatterns := []string{
		"connection reset",
		"connection refused",
		"timeout",
		"EOF",
		"broken pipe",
		"temporary failure",
	}
	for _, pattern := range retryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}
	return false
}

// ExtractZIP extracts a ZIP archive to the target directory.
// Returns the list of extracted file paths.
func (downloader *Downloader) ExtractZIP(zipPath string, targetDirectory string) ([]string, error) {
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP %s: %w", zipPath, err)
	}
	defer zipReader.Close()

	if err := os.MkdirAll(targetDirectory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create extraction directory: %w", err)
	}

	var extractedPaths []string

	for _, zipEntry := range zipReader.File {
		extractedPath := filepath.Join(targetDirectory, zipEntry.Name)

		// Prevent path traversal
		if !strings.HasPrefix(filepath.Clean(extractedPath), filepath.Clean(targetDirectory)) {
			continue
		}

		if zipEntry.FileInfo().IsDir() {
			os.MkdirAll(extractedPath, 0755)
			continue
		}

		// Ensure parent directory exists
		os.MkdirAll(filepath.Dir(extractedPath), 0755)

		entryReader, err := zipEntry.Open()
		if err != nil {
			return extractedPaths, fmt.Errorf("failed to open ZIP entry %s: %w", zipEntry.Name, err)
		}

		outputFile, err := os.Create(extractedPath)
		if err != nil {
			entryReader.Close()
			return extractedPaths, fmt.Errorf("failed to create %s: %w", extractedPath, err)
		}

		_, err = io.Copy(outputFile, entryReader)
		outputFile.Close()
		entryReader.Close()

		if err != nil {
			return extractedPaths, fmt.Errorf("failed to extract %s: %w", zipEntry.Name, err)
		}

		extractedPaths = append(extractedPaths, extractedPath)
	}

	return extractedPaths, nil
}

// ExtractTarGZ extracts a .tar.gz archive to the target directory.
// Returns the list of extracted file paths.
func (downloader *Downloader) ExtractTarGZ(tarGzPath string, targetDirectory string) ([]string, error) {
	archiveFile, err := os.Open(tarGzPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", tarGzPath, err)
	}
	defer archiveFile.Close()

	gzipReader, err := gzip.NewReader(archiveFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	if err := os.MkdirAll(targetDirectory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create extraction directory: %w", err)
	}

	var extractedPaths []string

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return extractedPaths, fmt.Errorf("tar read error: %w", err)
		}

		extractedPath := filepath.Join(targetDirectory, header.Name)

		// Prevent path traversal
		if !strings.HasPrefix(filepath.Clean(extractedPath), filepath.Clean(targetDirectory)) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(extractedPath, 0755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(extractedPath), 0755)

			outputFile, err := os.Create(extractedPath)
			if err != nil {
				return extractedPaths, fmt.Errorf("failed to create %s: %w", extractedPath, err)
			}

			_, err = io.Copy(outputFile, tarReader)
			outputFile.Close()

			if err != nil {
				return extractedPaths, fmt.Errorf("failed to extract %s: %w", header.Name, err)
			}

			extractedPaths = append(extractedPaths, extractedPath)
		}
	}

	return extractedPaths, nil
}

// CheckRemoteSize performs an HTTP HEAD request to get Content-Length.
func (downloader *Downloader) CheckRemoteSize(downloadURL string) (int64, error) {
	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return 0, fmt.Errorf("invalid URL: %w", err)
	}

	downloader.waitForDomain(parsedURL.Host)

	request, err := http.NewRequest(http.MethodHead, downloadURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create HEAD request: %w", err)
	}
	request.Header.Set("User-Agent", downloader.config.UserAgent)

	response, err := downloader.httpClient.Do(request)
	if err != nil {
		return 0, err
	}
	response.Body.Close()

	if response.StatusCode >= 400 {
		return 0, fmt.Errorf("HTTP %d", response.StatusCode)
	}

	return response.ContentLength, nil
}

// Manifest returns the underlying download manifest.
func (downloader *Downloader) Manifest() *DownloadManifest {
	return downloader.manifest
}

// SaveManifest persists the download manifest to disk.
func (downloader *Downloader) SaveManifest() error {
	return downloader.manifest.SaveManifest(downloader.manifestPath)
}

// SourceDirectory returns the download subdirectory for a source.
func (downloader *Downloader) SourceDirectory(sourceName string) string {
	return filepath.Join(downloader.config.DownloadDirectory, sourceName)
}

// waitForDomain enforces per-domain rate limiting.
func (downloader *Downloader) waitForDomain(domain string) {
	downloader.timerMu.Lock()

	lastRequestTime, hasLastRequest := downloader.domainTimers[domain]
	if hasLastRequest {
		elapsed := time.Since(lastRequestTime)
		if elapsed < downloader.config.RateLimit {
			waitDuration := downloader.config.RateLimit - elapsed
			downloader.timerMu.Unlock()
			time.Sleep(waitDuration)
			downloader.timerMu.Lock()
		}
	}

	downloader.domainTimers[domain] = time.Now()
	downloader.timerMu.Unlock()
}
