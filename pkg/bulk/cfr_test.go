package bulk

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCFRSourceName(t *testing.T) {
	source := NewCFRSource(DefaultDownloadConfig())

	if source.Name() != "cfr" {
		t.Errorf("expected name 'cfr', got %q", source.Name())
	}
	if !strings.Contains(source.Description(), "2024") {
		t.Errorf("expected description to contain default year '2024', got %q", source.Description())
	}
}

func TestCFRSourceCustomYear(t *testing.T) {
	config := DefaultDownloadConfig()
	config.CFRYear = "2023"
	source := NewCFRSource(config)

	if !strings.Contains(source.Description(), "2023") {
		t.Errorf("expected description to contain '2023', got %q", source.Description())
	}

	datasets, err := source.ListDatasets()
	if err != nil {
		t.Fatalf("ListDatasets failed: %v", err)
	}

	for _, dataset := range datasets {
		if !strings.Contains(dataset.URL, "2023") {
			t.Errorf("expected URL to contain '2023', got %q", dataset.URL)
			break
		}
		if !strings.Contains(dataset.Identifier, "2023") {
			t.Errorf("expected identifier to contain '2023', got %q", dataset.Identifier)
			break
		}
	}
}

func TestCFRSourceListDatasets(t *testing.T) {
	source := NewCFRSource(DefaultDownloadConfig())

	datasets, err := source.ListDatasets()
	if err != nil {
		t.Fatalf("ListDatasets failed: %v", err)
	}

	if len(datasets) != 50 {
		t.Fatalf("expected 50 CFR datasets, got %d", len(datasets))
	}

	firstDataset := datasets[0]
	if firstDataset.SourceName != "cfr" {
		t.Errorf("expected source name 'cfr', got %q", firstDataset.SourceName)
	}
	if firstDataset.Jurisdiction != "US" {
		t.Errorf("expected jurisdiction 'US', got %q", firstDataset.Jurisdiction)
	}
	if firstDataset.Format != "zip" {
		t.Errorf("expected format 'zip', got %q", firstDataset.Format)
	}
	if !strings.Contains(firstDataset.URL, "govinfo.gov") {
		t.Errorf("expected URL to contain govinfo.gov, got %q", firstDataset.URL)
	}

	// Verify Title 42 (Public Health) exists
	foundTitle42 := false
	for _, dataset := range datasets {
		if strings.Contains(dataset.Identifier, "title-42") {
			foundTitle42 = true
			if !strings.Contains(dataset.DisplayName, "Public Health") {
				t.Errorf("expected Title 42 to be Public Health, got %q", dataset.DisplayName)
			}
		}
	}
	if !foundTitle42 {
		t.Error("expected to find CFR title 42")
	}
}

func TestCFRSourceDownloadDataset(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Write([]byte("fake cfr zip data"))
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

	source := NewCFRSource(config)
	dataset := Dataset{
		SourceName: "cfr",
		Identifier: "cfr-2024-title-1",
		URL:        testServer.URL + "/CFR-2024-title-1.zip",
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
}

func TestCFRSourceDefaultYear(t *testing.T) {
	config := DefaultDownloadConfig()
	config.CFRYear = ""
	source := NewCFRSource(config)

	datasets, _ := source.ListDatasets()
	if len(datasets) > 0 {
		if !strings.Contains(datasets[0].URL, "2024") {
			t.Errorf("expected default year 2024 in URL, got %q", datasets[0].URL)
		}
	}
}
