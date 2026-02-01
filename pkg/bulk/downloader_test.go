package bulk

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupTestDownloader(t *testing.T) (*Downloader, string) {
	t.Helper()
	temporaryDir := t.TempDir()

	config := DownloadConfig{
		DownloadDirectory: temporaryDir,
		RateLimit:         1 * time.Millisecond,
		Timeout:           10 * time.Second,
		UserAgent:         "regula-test/1.0",
	}

	downloader, err := NewDownloader(config)
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	return downloader, temporaryDir
}

func TestNewDownloader(t *testing.T) {
	downloader, temporaryDir := setupTestDownloader(t)

	if downloader.Manifest() == nil {
		t.Fatal("expected non-nil manifest")
	}

	manifestPath := filepath.Join(temporaryDir, "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		// Manifest file may not exist yet (only created on save)
		// but download directory should exist
	}

	if _, err := os.Stat(temporaryDir); os.IsNotExist(err) {
		t.Fatal("expected download directory to be created")
	}
}

func TestDownloadFile(t *testing.T) {
	testContent := "Hello, this is test content for the download."

	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Write([]byte(testContent))
	}))
	defer testServer.Close()

	downloader, temporaryDir := setupTestDownloader(t)
	localPath := filepath.Join(temporaryDir, "test-download.txt")

	bytesWritten, skipped, err := downloader.DownloadFile(testServer.URL+"/test.txt", localPath, nil)
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	if skipped {
		t.Error("expected download to not be skipped")
	}
	if bytesWritten != int64(len(testContent)) {
		t.Errorf("expected %d bytes, got %d", len(testContent), bytesWritten)
	}

	savedContent, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(savedContent) != testContent {
		t.Errorf("content mismatch: got %q, want %q", string(savedContent), testContent)
	}
}

func TestDownloadFileSkipsExisting(t *testing.T) {
	downloader, temporaryDir := setupTestDownloader(t)
	localPath := filepath.Join(temporaryDir, "existing.txt")

	os.WriteFile(localPath, []byte("pre-existing content"), 0644)

	bytesWritten, skipped, err := downloader.DownloadFile("https://example.com/unused", localPath, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !skipped {
		t.Error("expected download to be skipped for existing file")
	}
	if bytesWritten == 0 {
		t.Error("expected non-zero bytes for existing file")
	}
}

func TestDownloadFileHTTPError(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(http.StatusNotFound)
	}))
	defer testServer.Close()

	downloader, temporaryDir := setupTestDownloader(t)
	localPath := filepath.Join(temporaryDir, "missing.txt")

	_, _, err := downloader.DownloadFile(testServer.URL+"/missing", localPath, nil)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to contain '404', got: %v", err)
	}
}

func TestDownloadFileProgress(t *testing.T) {
	testContent := strings.Repeat("x", 1000)

	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Write([]byte(testContent))
	}))
	defer testServer.Close()

	downloader, temporaryDir := setupTestDownloader(t)
	localPath := filepath.Join(temporaryDir, "progress.txt")

	progressCallCount := 0
	progressCallback := func(bytesDownloaded int64, totalBytes int64) {
		progressCallCount++
	}

	_, _, err := downloader.DownloadFile(testServer.URL+"/data", localPath, progressCallback)
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	if progressCallCount == 0 {
		t.Error("expected progress callback to be called at least once")
	}
}

func TestExtractZIP(t *testing.T) {
	temporaryDir := t.TempDir()
	zipPath := filepath.Join(temporaryDir, "test.zip")

	createTestZIP(t, zipPath, map[string]string{
		"readme.txt":    "This is a readme",
		"data/info.xml": "<root>hello</root>",
	})

	downloader, _ := setupTestDownloader(t)
	extractDir := filepath.Join(temporaryDir, "extracted")

	extractedFiles, err := downloader.ExtractZIP(zipPath, extractDir)
	if err != nil {
		t.Fatalf("extraction failed: %v", err)
	}

	if len(extractedFiles) != 2 {
		t.Fatalf("expected 2 extracted files, got %d", len(extractedFiles))
	}

	readmeContent, err := os.ReadFile(filepath.Join(extractDir, "readme.txt"))
	if err != nil {
		t.Fatalf("failed to read extracted readme: %v", err)
	}
	if string(readmeContent) != "This is a readme" {
		t.Errorf("readme content mismatch: got %q", string(readmeContent))
	}

	xmlContent, err := os.ReadFile(filepath.Join(extractDir, "data", "info.xml"))
	if err != nil {
		t.Fatalf("failed to read extracted XML: %v", err)
	}
	if string(xmlContent) != "<root>hello</root>" {
		t.Errorf("XML content mismatch: got %q", string(xmlContent))
	}
}

func TestExtractZIPPathTraversal(t *testing.T) {
	temporaryDir := t.TempDir()
	zipPath := filepath.Join(temporaryDir, "malicious.zip")

	createTestZIP(t, zipPath, map[string]string{
		"../escape.txt": "should not be extracted outside target",
		"safe.txt":      "safe content",
	})

	downloader, _ := setupTestDownloader(t)
	extractDir := filepath.Join(temporaryDir, "extracted")

	extractedFiles, err := downloader.ExtractZIP(zipPath, extractDir)
	if err != nil {
		t.Fatalf("extraction failed: %v", err)
	}

	// The traversal entry should be skipped
	escapePath := filepath.Join(temporaryDir, "escape.txt")
	if _, err := os.Stat(escapePath); err == nil {
		t.Error("path traversal file should not have been extracted outside target")
	}

	hasSafeFile := false
	for _, extractedFile := range extractedFiles {
		if strings.HasSuffix(extractedFile, "safe.txt") {
			hasSafeFile = true
		}
	}
	if !hasSafeFile {
		t.Error("expected safe.txt to be extracted")
	}
}

func TestExtractTarGZ(t *testing.T) {
	temporaryDir := t.TempDir()
	tarGzPath := filepath.Join(temporaryDir, "test.tar.gz")

	createTestTarGZ(t, tarGzPath, map[string]string{
		"doc.txt": "Document content",
		"code.xml": "<law>section 1</law>",
	})

	downloader, _ := setupTestDownloader(t)
	extractDir := filepath.Join(temporaryDir, "extracted")

	extractedFiles, err := downloader.ExtractTarGZ(tarGzPath, extractDir)
	if err != nil {
		t.Fatalf("extraction failed: %v", err)
	}

	if len(extractedFiles) != 2 {
		t.Fatalf("expected 2 extracted files, got %d", len(extractedFiles))
	}

	docContent, err := os.ReadFile(filepath.Join(extractDir, "doc.txt"))
	if err != nil {
		t.Fatalf("failed to read extracted doc: %v", err)
	}
	if string(docContent) != "Document content" {
		t.Errorf("doc content mismatch: got %q", string(docContent))
	}
}

func TestCheckRemoteSize(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodHead {
			responseWriter.Header().Set("Content-Length", "12345")
			responseWriter.WriteHeader(http.StatusOK)
			return
		}
		responseWriter.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer testServer.Close()

	downloader, _ := setupTestDownloader(t)
	size, err := downloader.CheckRemoteSize(testServer.URL + "/file.zip")
	if err != nil {
		t.Fatalf("check remote size failed: %v", err)
	}
	if size != 12345 {
		t.Errorf("expected size 12345, got %d", size)
	}
}

func TestSourceDirectory(t *testing.T) {
	downloader, temporaryDir := setupTestDownloader(t)

	sourceDir := downloader.SourceDirectory("uscode")
	expectedPath := filepath.Join(temporaryDir, "uscode")

	if sourceDir != expectedPath {
		t.Errorf("expected %q, got %q", expectedPath, sourceDir)
	}
}

func TestDownloadFileRetriesOn500(t *testing.T) {
	requestCount := 0
	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		requestCount++
		if requestCount <= 2 {
			responseWriter.WriteHeader(http.StatusInternalServerError)
			return
		}
		responseWriter.Write([]byte("success after retries"))
	}))
	defer testServer.Close()

	temporaryDir := t.TempDir()
	config := DownloadConfig{
		DownloadDirectory: temporaryDir,
		RateLimit:         1 * time.Millisecond,
		Timeout:           10 * time.Second,
		UserAgent:         "regula-test/1.0",
		MaxRetries:        3,
		RetryBaseDelay:    1 * time.Millisecond,
	}

	downloader, err := NewDownloader(config)
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	localPath := filepath.Join(temporaryDir, "retry-test.txt")
	bytesWritten, skipped, err := downloader.DownloadFile(testServer.URL+"/test", localPath, nil)
	if err != nil {
		t.Fatalf("expected download to succeed after retries, got: %v", err)
	}
	if skipped {
		t.Error("expected download not to be skipped")
	}
	if bytesWritten != int64(len("success after retries")) {
		t.Errorf("expected %d bytes, got %d", len("success after retries"), bytesWritten)
	}
	if requestCount != 3 {
		t.Errorf("expected 3 requests (2 failures + 1 success), got %d", requestCount)
	}
}

func TestDownloadFileDoesNotRetryOn404(t *testing.T) {
	requestCount := 0
	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		requestCount++
		responseWriter.WriteHeader(http.StatusNotFound)
	}))
	defer testServer.Close()

	temporaryDir := t.TempDir()
	config := DownloadConfig{
		DownloadDirectory: temporaryDir,
		RateLimit:         1 * time.Millisecond,
		Timeout:           10 * time.Second,
		UserAgent:         "regula-test/1.0",
		MaxRetries:        3,
		RetryBaseDelay:    1 * time.Millisecond,
	}

	downloader, err := NewDownloader(config)
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	localPath := filepath.Join(temporaryDir, "no-retry.txt")
	_, _, err = downloader.DownloadFile(testServer.URL+"/missing", localPath, nil)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if requestCount != 1 {
		t.Errorf("expected exactly 1 request (no retry for 404), got %d", requestCount)
	}
}

func TestDownloadFileExhaustsRetries(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer testServer.Close()

	temporaryDir := t.TempDir()
	config := DownloadConfig{
		DownloadDirectory: temporaryDir,
		RateLimit:         1 * time.Millisecond,
		Timeout:           10 * time.Second,
		UserAgent:         "regula-test/1.0",
		MaxRetries:        2,
		RetryBaseDelay:    1 * time.Millisecond,
	}

	downloader, err := NewDownloader(config)
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	localPath := filepath.Join(temporaryDir, "exhaust.txt")
	_, _, err = downloader.DownloadFile(testServer.URL+"/failing", localPath, nil)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if !strings.Contains(err.Error(), "failed after 2 attempts") {
		t.Errorf("expected 'failed after 2 attempts' in error, got: %v", err)
	}
}

func TestIsRetryableError(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"retryable HTTP 500", &retryableHTTPError{StatusCode: 500, URL: "test"}, true},
		{"retryable HTTP 503", &retryableHTTPError{StatusCode: 503, URL: "test"}, true},
		{"non-retryable error", fmt.Errorf("HTTP 404 for test"), false},
		{"timeout error", fmt.Errorf("connection timeout"), true},
		{"connection reset", fmt.Errorf("connection reset by peer"), true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := isRetryableError(testCase.err)
			if result != testCase.expected {
				t.Errorf("isRetryableError(%v) = %v, want %v", testCase.err, result, testCase.expected)
			}
		})
	}
}

// createTestZIP creates a ZIP file with the given name→content entries.
func createTestZIP(t *testing.T, zipPath string, entries map[string]string) {
	t.Helper()

	outputFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create ZIP file: %v", err)
	}
	defer outputFile.Close()

	zipWriter := zip.NewWriter(outputFile)
	defer zipWriter.Close()

	for name, content := range entries {
		entryWriter, err := zipWriter.Create(name)
		if err != nil {
			t.Fatalf("failed to create ZIP entry %s: %v", name, err)
		}
		entryWriter.Write([]byte(content))
	}
}

// createTestTarGZ creates a .tar.gz file with the given name→content entries.
func createTestTarGZ(t *testing.T, tarGzPath string, entries map[string]string) {
	t.Helper()

	outputFile, err := os.Create(tarGzPath)
	if err != nil {
		t.Fatalf("failed to create tar.gz file: %v", err)
	}
	defer outputFile.Close()

	gzipWriter := gzip.NewWriter(outputFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for name, content := range entries {
		header := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("failed to write tar header for %s: %v", name, err)
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write tar content for %s: %v", name, err)
		}
	}
}
