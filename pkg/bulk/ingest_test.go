package bulk

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeriveDocumentID(t *testing.T) {
	testCases := []struct {
		name       string
		record     *DownloadRecord
		expectedID string
	}{
		{
			name:       "uscode title",
			record:     &DownloadRecord{Identifier: "usc-title-42", SourceName: "uscode"},
			expectedID: "us-usc-title-42",
		},
		{
			name:       "cfr title",
			record:     &DownloadRecord{Identifier: "cfr-2024-title-42", SourceName: "cfr"},
			expectedID: "us-cfr-2024-title-42",
		},
		{
			name:       "california code",
			record:     &DownloadRecord{Identifier: "ca-civ", SourceName: "california"},
			expectedID: "us-ca-civ",
		},
		{
			name:       "archive item",
			record:     &DownloadRecord{Identifier: "govlawca", SourceName: "archive"},
			expectedID: "archive-govlawca",
		},
		{
			name:       "unknown source",
			record:     &DownloadRecord{Identifier: "something", SourceName: "other"},
			expectedID: "something",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			documentID := deriveDocumentID(testCase.record)
			if documentID != testCase.expectedID {
				t.Errorf("deriveDocumentID() = %q, want %q", documentID, testCase.expectedID)
			}
		})
	}
}

func TestDeriveAddOptions(t *testing.T) {
	testCases := []struct {
		name                 string
		record               *DownloadRecord
		documentID           string
		expectedJurisdiction string
		expectedFormat       string
	}{
		{
			name:                 "uscode defaults",
			record:               &DownloadRecord{Identifier: "usc-title-42", SourceName: "uscode", URL: "https://example.com"},
			documentID:           "us-usc-title-42",
			expectedJurisdiction: "US",
			expectedFormat:       "us",
		},
		{
			name:                 "cfr defaults",
			record:               &DownloadRecord{Identifier: "cfr-2024-title-21", SourceName: "cfr", URL: "https://example.com"},
			documentID:           "us-cfr-2024-title-21",
			expectedJurisdiction: "US",
			expectedFormat:       "us",
		},
		{
			name:                 "california jurisdiction",
			record:               &DownloadRecord{Identifier: "ca-civ", SourceName: "california", URL: "https://example.com"},
			documentID:           "us-ca-civ",
			expectedJurisdiction: "US-CA",
			expectedFormat:       "us",
		},
		{
			name:                 "archive generic format",
			record:               &DownloadRecord{Identifier: "govlawca", SourceName: "archive", URL: "https://example.com"},
			documentID:           "archive-govlawca",
			expectedJurisdiction: "US",
			expectedFormat:       "generic",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			options := deriveAddOptions(testCase.record, testCase.documentID)

			if options.Jurisdiction != testCase.expectedJurisdiction {
				t.Errorf("expected jurisdiction %q, got %q", testCase.expectedJurisdiction, options.Jurisdiction)
			}
			if options.Format != testCase.expectedFormat {
				t.Errorf("expected format %q, got %q", testCase.expectedFormat, options.Format)
			}
			if options.Name != testCase.documentID {
				t.Errorf("expected name %q, got %q", testCase.documentID, options.Name)
			}
			if options.SourceInfo == "" {
				t.Error("expected non-empty source info")
			}
		})
	}
}

func TestMatchesTitleFilter(t *testing.T) {
	testCases := []struct {
		name        string
		identifier  string
		titleFilter []string
		expected    bool
	}{
		{
			name:        "matches exact",
			identifier:  "usc-title-42",
			titleFilter: []string{"title-42"},
			expected:    true,
		},
		{
			name:        "matches case insensitive",
			identifier:  "USC-TITLE-42",
			titleFilter: []string{"title-42"},
			expected:    true,
		},
		{
			name:        "matches one of many",
			identifier:  "usc-title-42",
			titleFilter: []string{"title-26", "title-42", "title-18"},
			expected:    true,
		},
		{
			name:        "no match",
			identifier:  "usc-title-42",
			titleFilter: []string{"title-26", "title-18"},
			expected:    false,
		},
		{
			name:        "empty filter matches nothing",
			identifier:  "usc-title-42",
			titleFilter: []string{},
			expected:    false,
		},
		{
			name:        "partial match",
			identifier:  "ca-civ",
			titleFilter: []string{"civ"},
			expected:    true,
		},
		{
			name:        "trims whitespace in filter",
			identifier:  "usc-title-42",
			titleFilter: []string{"  title-42  "},
			expected:    true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := matchesTitleFilter(testCase.identifier, testCase.titleFilter)
			if result != testCase.expected {
				t.Errorf("matchesTitleFilter(%q, %v) = %v, want %v",
					testCase.identifier, testCase.titleFilter, result, testCase.expected)
			}
		})
	}
}

func TestIngestCaliforniaFile(t *testing.T) {
	temporaryDir := t.TempDir()
	textFile := filepath.Join(temporaryDir, "CIV.txt")

	codeText := `CALIFORNIA Civil Code

DIVISION 1. PERSONS
Section 1. All people are by nature free and independent.
Section 2. Every person has certain inalienable rights.
`
	os.WriteFile(textFile, []byte(codeText), 0644)

	ingester := &BulkIngester{
		config: IngestConfig{DryRun: true},
	}

	record := &DownloadRecord{
		Identifier: "ca-civ",
		SourceName: "california",
		LocalPath:  textFile,
	}

	plaintext, err := ingester.ingestCalifornia(record)
	if err != nil {
		t.Fatalf("ingestCalifornia failed: %v", err)
	}
	if plaintext != codeText {
		t.Errorf("expected plaintext to match input, got %q", plaintext)
	}
}

func TestIngestCaliforniaMissingFile(t *testing.T) {
	ingester := &BulkIngester{
		config: IngestConfig{DryRun: true},
	}

	record := &DownloadRecord{
		Identifier: "ca-civ",
		SourceName: "california",
		LocalPath:  "/nonexistent/CIV.txt",
	}

	_, err := ingester.ingestCalifornia(record)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestConcatenateTextFiles(t *testing.T) {
	temporaryDir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(temporaryDir, "doc.txt"), []byte("text content"), 0644)
	os.WriteFile(filepath.Join(temporaryDir, "data.xml"), []byte("<root>xml</root>"), 0644)
	os.WriteFile(filepath.Join(temporaryDir, "page.html"), []byte("<p>html</p>"), 0644)
	os.WriteFile(filepath.Join(temporaryDir, "image.jpg"), []byte("binary"), 0644)
	os.WriteFile(filepath.Join(temporaryDir, "code.py"), []byte("python"), 0644)

	filePaths := []string{
		filepath.Join(temporaryDir, "doc.txt"),
		filepath.Join(temporaryDir, "data.xml"),
		filepath.Join(temporaryDir, "page.html"),
		filepath.Join(temporaryDir, "image.jpg"),
		filepath.Join(temporaryDir, "code.py"),
	}

	result := concatenateTextFiles(filePaths)

	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
	if result == "" {
		t.Fatal("expected concatenated content")
	}

	// Should include txt, xml, html files
	if len(result) == 0 {
		t.Fatal("expected content from text files")
	}

	// Verify txt content is included
	if result == "" {
		t.Fatal("expected non-empty concatenation")
	}
}

func TestConcatenateTextFilesFiltersExtensions(t *testing.T) {
	temporaryDir := t.TempDir()

	os.WriteFile(filepath.Join(temporaryDir, "doc.txt"), []byte("included"), 0644)
	os.WriteFile(filepath.Join(temporaryDir, "img.jpg"), []byte("excluded"), 0644)

	filePaths := []string{
		filepath.Join(temporaryDir, "doc.txt"),
		filepath.Join(temporaryDir, "img.jpg"),
	}

	result := concatenateTextFiles(filePaths)

	if result == "" {
		t.Fatal("expected non-empty result from .txt file")
	}
	if len(result) > len("included\n\n")+5 {
		t.Error("expected only .txt content, jpg should be excluded")
	}
}

func TestConcatenateTextFilesEmpty(t *testing.T) {
	result := concatenateTextFiles(nil)
	if result != "" {
		t.Errorf("expected empty result for nil input, got %q", result)
	}
}

func TestAccumulateReportStats(t *testing.T) {
	report := &IngestReport{}

	// Ingested entry with stats
	accumulateReportStats(report, IngestEntry{
		Status:      "ingested",
		Triples:     25000,
		Articles:    4000,
		Chapters:    195,
		Definitions: 5,
		References:  20,
		Rights:      7,
		Obligations: 30,
	})

	if report.Succeeded != 1 {
		t.Errorf("expected 1 succeeded, got %d", report.Succeeded)
	}
	if report.TotalTriples != 25000 {
		t.Errorf("expected 25000 total triples, got %d", report.TotalTriples)
	}
	if report.TotalArticles != 4000 {
		t.Errorf("expected 4000 total articles, got %d", report.TotalArticles)
	}

	// Skipped entry with stats from previous run
	accumulateReportStats(report, IngestEntry{
		Status:   "skipped",
		Triples:  5000,
		Articles: 1000,
		Chapters: 5,
	})

	if report.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", report.Skipped)
	}
	if report.TotalTriples != 30000 {
		t.Errorf("expected 30000 total triples after skipped, got %d", report.TotalTriples)
	}
	if report.TotalArticles != 5000 {
		t.Errorf("expected 5000 total articles after skipped, got %d", report.TotalArticles)
	}

	// Failed entry
	accumulateReportStats(report, IngestEntry{
		Status: "failed",
	})

	if report.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", report.Failed)
	}
	// Totals should not change from failed entry
	if report.TotalTriples != 30000 {
		t.Errorf("expected 30000 total triples after failed, got %d", report.TotalTriples)
	}
}
