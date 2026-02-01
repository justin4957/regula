package bulk

import (
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"
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

	output := FormatStatusReport(manifest, "", nil)

	if !strings.Contains(output, "uscode") {
		t.Error("expected uscode source in status")
	}
	if !strings.Contains(output, "california") {
		t.Error("expected california source in status")
	}
	if !strings.Contains(output, "Downloads: 2") {
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

	output := FormatStatusReport(manifest, "uscode", nil)

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

func TestFormatStatusReportWithIngestStats(t *testing.T) {
	manifest := NewDownloadManifest()
	manifest.RecordDownload(&DownloadRecord{
		Identifier: "usc-title-42",
		SourceName: "uscode",
		SizeBytes:  5242880,
	})
	manifest.RecordDownload(&DownloadRecord{
		Identifier: "usc-title-04",
		SourceName: "uscode",
		SizeBytes:  102400,
	})

	documentStats := map[string]*DocumentStatsSummary{
		"us-usc-title-42": {
			Triples:  25100,
			Articles: 4099,
			Chapters: 195,
			Status:   "ready",
		},
	}

	output := FormatStatusReport(manifest, "uscode", documentStats)

	if !strings.Contains(output, "ready") {
		t.Error("expected 'ready' status for ingested title")
	}
	if !strings.Contains(output, "25100 triples") {
		t.Error("expected triple count in status output")
	}
	if !strings.Contains(output, "pending") {
		t.Error("expected 'pending' status for non-ingested title")
	}
	if !strings.Contains(output, "Ingested: 1") {
		t.Error("expected ingested count in totals")
	}
}

func TestFormatIngestReportWithAggregates(t *testing.T) {
	report := &IngestReport{
		TotalAttempted:   2,
		Succeeded:        2,
		TotalTriples:     30000,
		TotalArticles:    5000,
		TotalChapters:    200,
		TotalDefinitions: 10,
		TotalReferences:  50,
		Entries: []IngestEntry{
			{DocumentID: "us-usc-title-42", Status: "ingested", Triples: 25000, Articles: 4000, Chapters: 195},
			{DocumentID: "us-usc-title-04", Status: "ingested", Triples: 5000, Articles: 1000, Chapters: 5},
		},
	}

	output := FormatIngestReport(report)

	if !strings.Contains(output, "Totals:") {
		t.Error("expected aggregate totals section")
	}
	if !strings.Contains(output, "30000 triples") {
		t.Error("expected total triple count")
	}
	if !strings.Contains(output, "5000 articles") {
		t.Error("expected total article count")
	}
	if !strings.Contains(output, "200 chapters") {
		t.Error("expected total chapter count")
	}
}

func TestFormatDuration(t *testing.T) {
	testCases := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"milliseconds", 500 * time.Millisecond, "500ms"},
		{"seconds", 5500 * time.Millisecond, "5.5s"},
		{"minutes", 125 * time.Second, "2m5s"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := formatDuration(testCase.duration)
			if result != testCase.expected {
				t.Errorf("formatDuration(%v) = %q, want %q", testCase.duration, result, testCase.expected)
			}
		})
	}
}

func TestCollectStats(t *testing.T) {
	manifest := NewDownloadManifest()
	manifest.RecordDownload(&DownloadRecord{
		Identifier: "usc-title-42",
		SourceName: "uscode",
		SizeBytes:  5242880,
	})
	manifest.RecordDownload(&DownloadRecord{
		Identifier: "usc-title-04",
		SourceName: "uscode",
		SizeBytes:  102400,
	})

	ingestedAt := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	documentStats := map[string]*DocumentStatsSummary{
		"us-usc-title-42": {
			Triples:     25100,
			Articles:    4099,
			Chapters:    195,
			Definitions: 312,
			References:  1500,
			Rights:      85,
			Obligations: 230,
			Status:      "ready",
			DisplayName: "Title 42 - Public Health",
			Source:      "uscode",
			IngestedAt:  ingestedAt,
		},
	}

	report := CollectStats(manifest, documentStats)

	if report.TitlesTotal != 2 {
		t.Errorf("TitlesTotal = %d, want 2", report.TitlesTotal)
	}
	if report.TitlesIngested != 1 {
		t.Errorf("TitlesIngested = %d, want 1", report.TitlesIngested)
	}
	if report.TotalTriples != 25100 {
		t.Errorf("TotalTriples = %d, want 25100", report.TotalTriples)
	}
	if report.TotalArticles != 4099 {
		t.Errorf("TotalArticles = %d, want 4099", report.TotalArticles)
	}
	if report.TotalChapters != 195 {
		t.Errorf("TotalChapters = %d, want 195", report.TotalChapters)
	}
	if report.TotalDefinitions != 312 {
		t.Errorf("TotalDefinitions = %d, want 312", report.TotalDefinitions)
	}
	if report.TotalReferences != 1500 {
		t.Errorf("TotalReferences = %d, want 1500", report.TotalReferences)
	}
	if report.TotalRights != 85 {
		t.Errorf("TotalRights = %d, want 85", report.TotalRights)
	}
	if report.TotalObligations != 230 {
		t.Errorf("TotalObligations = %d, want 230", report.TotalObligations)
	}

	if len(report.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(report.Entries))
	}

	// Entries are sorted by identifier; usc-title-04 comes first
	pendingEntry := report.Entries[0]
	if pendingEntry.Identifier != "usc-title-04" {
		t.Errorf("first entry identifier = %q, want %q", pendingEntry.Identifier, "usc-title-04")
	}
	if pendingEntry.Status != "pending" {
		t.Errorf("pending entry status = %q, want %q", pendingEntry.Status, "pending")
	}
	if pendingEntry.Triples != 0 {
		t.Errorf("pending entry triples = %d, want 0", pendingEntry.Triples)
	}

	readyEntry := report.Entries[1]
	if readyEntry.Identifier != "usc-title-42" {
		t.Errorf("second entry identifier = %q, want %q", readyEntry.Identifier, "usc-title-42")
	}
	if readyEntry.Status != "ready" {
		t.Errorf("ready entry status = %q, want %q", readyEntry.Status, "ready")
	}
	if readyEntry.Triples != 25100 {
		t.Errorf("ready entry triples = %d, want 25100", readyEntry.Triples)
	}
	if readyEntry.DisplayName != "Title 42 - Public Health" {
		t.Errorf("ready entry display name = %q, want %q", readyEntry.DisplayName, "Title 42 - Public Health")
	}
	if !readyEntry.IngestedAt.Equal(ingestedAt) {
		t.Errorf("ready entry ingested at = %v, want %v", readyEntry.IngestedAt, ingestedAt)
	}
}

func TestCollectStatsEmpty(t *testing.T) {
	manifest := NewDownloadManifest()
	report := CollectStats(manifest, nil)

	if report.TitlesTotal != 0 {
		t.Errorf("TitlesTotal = %d, want 0", report.TitlesTotal)
	}
	if report.TitlesIngested != 0 {
		t.Errorf("TitlesIngested = %d, want 0", report.TitlesIngested)
	}
	if report.TotalTriples != 0 {
		t.Errorf("TotalTriples = %d, want 0", report.TotalTriples)
	}
	if len(report.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(report.Entries))
	}
}

func TestFormatStatsTable(t *testing.T) {
	report := &StatsReport{
		TitlesIngested:   1,
		TitlesTotal:      2,
		TotalTriples:     25100,
		TotalArticles:    4099,
		TotalChapters:    195,
		TotalDefinitions: 312,
		TotalReferences:  1500,
		TotalRights:      85,
		TotalObligations: 230,
		Entries: []StatsEntry{
			{
				Identifier:  "usc-title-04",
				DocumentID:  "us-usc-title-04",
				Source:      "uscode",
				Status:      "pending",
			},
			{
				Identifier:  "usc-title-42",
				DocumentID:  "us-usc-title-42",
				DisplayName: "Title 42 - Public Health",
				Source:      "uscode",
				Triples:     25100,
				Articles:    4099,
				Chapters:    195,
				Definitions: 312,
				References:  1500,
				Rights:      85,
				Obligations: 230,
				Status:      "ready",
			},
		},
	}

	output := FormatStatsTable(report)

	if !strings.Contains(output, "Bulk Ingestion Statistics") {
		t.Error("expected header 'Bulk Ingestion Statistics'")
	}
	if !strings.Contains(output, "Titles Ingested: 1/2") {
		t.Error("expected 'Titles Ingested: 1/2'")
	}
	if !strings.Contains(output, "TITLE") {
		t.Error("expected column header TITLE")
	}
	if !strings.Contains(output, "TRIPLES") {
		t.Error("expected column header TRIPLES")
	}
	if !strings.Contains(output, "Title 42 - Public Health") {
		t.Error("expected display name for ingested title")
	}
	if !strings.Contains(output, "25,100") {
		t.Error("expected comma-formatted triples count '25,100'")
	}
	if !strings.Contains(output, "4,099") {
		t.Error("expected comma-formatted articles count '4,099'")
	}
	if !strings.Contains(output, "TOTALS") {
		t.Error("expected TOTALS row")
	}
	// Pending entry should show dashes
	if !strings.Contains(output, "pending") {
		t.Error("expected 'pending' status for non-ingested entry")
	}
}

func TestFormatStatsJSON(t *testing.T) {
	report := &StatsReport{
		TitlesIngested: 1,
		TitlesTotal:    2,
		TotalTriples:   25100,
		Entries: []StatsEntry{
			{
				Identifier: "usc-title-42",
				DocumentID: "us-usc-title-42",
				Triples:    25100,
				Status:     "ready",
			},
		},
	}

	jsonOutput := FormatStatsJSON(report)

	// Verify it's valid JSON
	var parsed StatsReport
	if err := json.Unmarshal([]byte(jsonOutput), &parsed); err != nil {
		t.Fatalf("FormatStatsJSON produced invalid JSON: %v", err)
	}

	if parsed.TitlesIngested != 1 {
		t.Errorf("parsed TitlesIngested = %d, want 1", parsed.TitlesIngested)
	}
	if parsed.TotalTriples != 25100 {
		t.Errorf("parsed TotalTriples = %d, want 25100", parsed.TotalTriples)
	}
	if len(parsed.Entries) != 1 {
		t.Fatalf("parsed entries count = %d, want 1", len(parsed.Entries))
	}
	if parsed.Entries[0].DocumentID != "us-usc-title-42" {
		t.Errorf("parsed entry document ID = %q, want %q", parsed.Entries[0].DocumentID, "us-usc-title-42")
	}
}

func TestFormatStatsCSV(t *testing.T) {
	ingestedAt := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	report := &StatsReport{
		TitlesIngested: 1,
		TitlesTotal:    2,
		Entries: []StatsEntry{
			{
				Identifier:  "usc-title-42",
				DocumentID:  "us-usc-title-42",
				DisplayName: "Title 42",
				Source:      "uscode",
				Triples:     25100,
				Articles:    4099,
				Chapters:    195,
				Definitions: 312,
				References:  1500,
				Rights:      85,
				Obligations: 230,
				Status:      "ready",
				IngestedAt:  ingestedAt,
			},
			{
				Identifier: "usc-title-04",
				DocumentID: "us-usc-title-04",
				Source:     "uscode",
				Status:     "pending",
			},
		},
	}

	csvOutput := FormatStatsCSV(report)

	// Verify parseable by csv.Reader
	reader := csv.NewReader(strings.NewReader(csvOutput))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("FormatStatsCSV produced invalid CSV: %v", err)
	}

	// Header + 2 data rows
	if len(records) != 3 {
		t.Fatalf("expected 3 CSV rows (header + 2 data), got %d", len(records))
	}

	// Verify header
	header := records[0]
	expectedHeaders := []string{
		"identifier", "document_id", "display_name", "source",
		"triples", "articles", "chapters", "definitions",
		"references", "rights", "obligations", "status", "ingested_at",
	}
	if len(header) != len(expectedHeaders) {
		t.Fatalf("header column count = %d, want %d", len(header), len(expectedHeaders))
	}
	for columnIndex, expectedColumn := range expectedHeaders {
		if header[columnIndex] != expectedColumn {
			t.Errorf("header[%d] = %q, want %q", columnIndex, header[columnIndex], expectedColumn)
		}
	}

	// Verify first data row
	firstRow := records[1]
	if firstRow[0] != "usc-title-42" {
		t.Errorf("first row identifier = %q, want %q", firstRow[0], "usc-title-42")
	}
	if firstRow[4] != "25100" {
		t.Errorf("first row triples = %q, want %q", firstRow[4], "25100")
	}
	if firstRow[11] != "ready" {
		t.Errorf("first row status = %q, want %q", firstRow[11], "ready")
	}
	if firstRow[12] == "" {
		t.Error("first row ingested_at should not be empty")
	}

	// Verify second data row (pending, no timestamp)
	secondRow := records[2]
	if secondRow[0] != "usc-title-04" {
		t.Errorf("second row identifier = %q, want %q", secondRow[0], "usc-title-04")
	}
	if secondRow[4] != "0" {
		t.Errorf("second row triples = %q, want %q", secondRow[4], "0")
	}
	if secondRow[12] != "" {
		t.Errorf("second row ingested_at = %q, want empty", secondRow[12])
	}
}

func TestFormatNumber(t *testing.T) {
	testCases := []struct {
		name     string
		input    int
		expected string
	}{
		{"zero", 0, "0"},
		{"small number", 999, "999"},
		{"one thousand", 1000, "1,000"},
		{"typical count", 1234, "1,234"},
		{"large number", 1234567, "1,234,567"},
		{"exact thousands", 25100, "25,100"},
		{"very large", 999999999, "999,999,999"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := formatNumber(testCase.input)
			if result != testCase.expected {
				t.Errorf("formatNumber(%d) = %q, want %q", testCase.input, result, testCase.expected)
			}
		})
	}
}
