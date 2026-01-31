package bulk

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUSCodeSourceName(t *testing.T) {
	source := NewUSCodeSource(DefaultDownloadConfig())

	if source.Name() != "uscode" {
		t.Errorf("expected name 'uscode', got %q", source.Name())
	}
	if source.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestUSCodeSourceListDatasets(t *testing.T) {
	source := NewUSCodeSource(DefaultDownloadConfig())

	datasets, err := source.ListDatasets()
	if err != nil {
		t.Fatalf("ListDatasets failed: %v", err)
	}

	if len(datasets) != 54 {
		t.Fatalf("expected 54 USC datasets, got %d", len(datasets))
	}

	// Verify first title
	firstDataset := datasets[0]
	if firstDataset.Identifier != "usc-title-01" {
		t.Errorf("expected first identifier 'usc-title-01', got %q", firstDataset.Identifier)
	}
	if firstDataset.SourceName != "uscode" {
		t.Errorf("expected source name 'uscode', got %q", firstDataset.SourceName)
	}
	if firstDataset.Jurisdiction != "US" {
		t.Errorf("expected jurisdiction 'US', got %q", firstDataset.Jurisdiction)
	}
	if firstDataset.Format != "zip" {
		t.Errorf("expected format 'zip', got %q", firstDataset.Format)
	}
	if !strings.Contains(firstDataset.URL, "uscode.house.gov") {
		t.Errorf("expected URL to contain uscode.house.gov, got %q", firstDataset.URL)
	}
	if !strings.Contains(firstDataset.URL, ".zip") {
		t.Errorf("expected URL to end with .zip, got %q", firstDataset.URL)
	}

	// Verify Title 42 exists
	foundTitle42 := false
	for _, dataset := range datasets {
		if dataset.Identifier == "usc-title-42" {
			foundTitle42 = true
			if !strings.Contains(dataset.DisplayName, "Public Health") {
				t.Errorf("expected Title 42 display name to contain 'Public Health', got %q", dataset.DisplayName)
			}
		}
	}
	if !foundTitle42 {
		t.Error("expected to find usc-title-42 in datasets")
	}
}

func TestUSCodeSourceDownloadDataset(t *testing.T) {
	testZIPContent := "PK\x03\x04fake zip content"

	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Write([]byte(testZIPContent))
	}))
	defer testServer.Close()

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

	source := NewUSCodeSource(config)
	dataset := Dataset{
		SourceName: "uscode",
		Identifier: "usc-title-04",
		URL:        testServer.URL + "/xml_usc04@119-73not60.zip",
		Format:     "zip",
	}

	result, err := source.DownloadDataset(dataset, downloader)
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}

	if result.Skipped {
		t.Error("expected download to not be skipped")
	}
	if result.BytesWritten == 0 {
		t.Error("expected non-zero bytes written")
	}

	expectedLocalPath := filepath.Join(temporaryDir, "uscode", "usc-title-04.zip")
	if result.LocalPath != expectedLocalPath {
		t.Errorf("expected local path %q, got %q", expectedLocalPath, result.LocalPath)
	}

	// Verify manifest was updated
	if !downloader.Manifest().IsDownloaded("usc-title-04") {
		t.Error("expected manifest to record download")
	}
}
