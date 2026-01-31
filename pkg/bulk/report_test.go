package bulk

import (
	"strings"
	"testing"
)

func TestFormatBytes(t *testing.T) {
	testCases := []struct {
		name      string
		input     int64
		expected  string
	}{
		{"zero bytes", 0, "0 B"},
		{"small bytes", 512, "512 B"},
		{"kilobytes", 1536, "1.5 KB"},
		{"megabytes", 5242880, "5.0 MB"},
		{"gigabytes", 1610612736, "1.5 GB"},
		{"exact KB", 1024, "1.0 KB"},
		{"exact MB", 1048576, "1.0 MB"},
		{"exact GB", 1073741824, "1.0 GB"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := FormatBytes(testCase.input)
			if result != testCase.expected {
				t.Errorf("FormatBytes(%d) = %q, want %q", testCase.input, result, testCase.expected)
			}
		})
	}
}

func TestFormatDatasetTable(t *testing.T) {
	datasets := []Dataset{
		{
			Identifier:   "usc-title-42",
			Jurisdiction: "US",
			Format:       "zip",
			DisplayName:  "Title 42 - The Public Health and Welfare",
		},
		{
			Identifier:   "ca-civ",
			Jurisdiction: "US-CA",
			Format:       "html",
			DisplayName:  "Civil Code (CIV)",
		},
	}

	output := FormatDatasetTable(datasets)

	if !strings.Contains(output, "IDENTIFIER") {
		t.Error("expected header row with IDENTIFIER")
	}
	if !strings.Contains(output, "usc-title-42") {
		t.Error("expected usc-title-42 in output")
	}
	if !strings.Contains(output, "ca-civ") {
		t.Error("expected ca-civ in output")
	}
	if !strings.Contains(output, "US-CA") {
		t.Error("expected US-CA jurisdiction")
	}
	if !strings.Contains(output, "Total: 2 datasets") {
		t.Error("expected total count of 2")
	}
}

func TestFormatDatasetTableLongName(t *testing.T) {
	datasets := []Dataset{
		{
			Identifier:  "test",
			DisplayName: "This is a very long display name that should be truncated to fit the table",
		},
	}

	output := FormatDatasetTable(datasets)
	if !strings.Contains(output, "...") {
		t.Error("expected long display name to be truncated with '...'")
	}
}

func TestFormatDatasetTableEmpty(t *testing.T) {
	output := FormatDatasetTable(nil)
	if !strings.Contains(output, "Total: 0 datasets") {
		t.Error("expected total count of 0 for empty dataset list")
	}
}

func TestFormatIngestReport(t *testing.T) {
	report := &IngestReport{
		TotalAttempted: 3,
		Succeeded:      1,
		Skipped:        1,
		Failed:         1,
		Entries: []IngestEntry{
			{Identifier: "usc-title-42", DocumentID: "us-usc-title-42", Status: "ingested", Triples: 500},
			{Identifier: "ca-civ", DocumentID: "us-ca-civ", Status: "skipped"},
			{Identifier: "cfr-t21", DocumentID: "us-cfr-t21", Status: "failed", Error: "parse error"},
		},
	}

	output := FormatIngestReport(report)

	if !strings.Contains(output, "Attempted: 3") {
		t.Error("expected attempted count in report")
	}
	if !strings.Contains(output, "Succeeded: 1") {
		t.Error("expected succeeded count in report")
	}
	if !strings.Contains(output, "[OK]") {
		t.Error("expected [OK] status marker")
	}
	if !strings.Contains(output, "[SKIP]") {
		t.Error("expected [SKIP] status marker")
	}
	if !strings.Contains(output, "[FAIL]") {
		t.Error("expected [FAIL] status marker")
	}
	if !strings.Contains(output, "500 triples") {
		t.Error("expected triple count in output")
	}
	if !strings.Contains(output, "parse error") {
		t.Error("expected error message in output")
	}
}

func TestFormatIngestReportJSON(t *testing.T) {
	report := &IngestReport{
		TotalAttempted: 1,
		Succeeded:      1,
		Entries: []IngestEntry{
			{Identifier: "test", DocumentID: "test-id", Status: "ingested"},
		},
	}

	jsonOutput := FormatIngestReportJSON(report)

	if !strings.Contains(jsonOutput, `"total_attempted": 1`) {
		t.Error("expected total_attempted in JSON output")
	}
	if !strings.Contains(jsonOutput, `"succeeded": 1`) {
		t.Error("expected succeeded in JSON output")
	}
	if !strings.Contains(jsonOutput, `"test-id"`) {
		t.Error("expected document ID in JSON output")
	}
}

func TestFormatStatusReport(t *testing.T) {
	manifest := NewDownloadManifest()
	manifest.RecordDownload(&DownloadRecord{
		Identifier: "usc-title-42",
		SourceName: "uscode",
		SizeBytes:  5242880,
	})
	manifest.RecordDownload(&DownloadRecord{
		Identifier: "ca-civ",
		SourceName: "california",
		SizeBytes:  1048576,
	})

	output := FormatStatusReport(manifest, "")

	if !strings.Contains(output, "uscode") {
		t.Error("expected uscode source in status")
	}
	if !strings.Contains(output, "california") {
		t.Error("expected california source in status")
	}
	if !strings.Contains(output, "Total: 2 downloads") {
		t.Error("expected total download count")
	}
}

func TestFormatStatusReportFiltered(t *testing.T) {
	manifest := NewDownloadManifest()
	manifest.RecordDownload(&DownloadRecord{
		Identifier: "usc-title-42",
		SourceName: "uscode",
		SizeBytes:  5242880,
	})
	manifest.RecordDownload(&DownloadRecord{
		Identifier: "ca-civ",
		SourceName: "california",
		SizeBytes:  1048576,
	})

	output := FormatStatusReport(manifest, "uscode")

	if !strings.Contains(output, "uscode") {
		t.Error("expected uscode in filtered status")
	}
	// The filtered report should only list the uscode source section header
	lines := strings.Split(output, "\n")
	hasCaliforniaHeader := false
	for _, line := range lines {
		if strings.Contains(line, "california") && strings.Contains(line, "downloads") {
			hasCaliforniaHeader = true
		}
	}
	if hasCaliforniaHeader {
		t.Error("expected california source to be excluded from filtered status")
	}
}
