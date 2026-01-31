package library

import (
	"path/filepath"
	"testing"
)

func TestSeedFromCorpus(t *testing.T) {
	tempDir := t.TempDir()
	lib, err := Init(filepath.Join(tempDir, "lib"), "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	testdataDir := filepath.Join("..", "..", "testdata")

	// Seed just two entries for speed
	entries := []CorpusEntry{
		{ID: "us-va-vcdpa", Jurisdiction: "US-VA", ShortName: "VCDPA", Format: "us", SourcePath: "vcdpa.txt"},
		{ID: "us-tx-tdpsa", Jurisdiction: "US-TX", ShortName: "TDPSA", Format: "us", SourcePath: "tdpsa.txt"},
	}

	seedReport, err := SeedFromCorpus(lib, testdataDir, entries)
	if err != nil {
		t.Fatalf("SeedFromCorpus failed: %v", err)
	}

	if seedReport.Succeeded != 2 {
		t.Errorf("expected 2 succeeded, got %d (failed: %d)", seedReport.Succeeded, seedReport.Failed)
		for _, entryState := range seedReport.Entries {
			if entryState.Error != "" {
				t.Logf("  %s: %s", entryState.ID, entryState.Error)
			}
		}
	}

	docs := lib.ListDocuments()
	if len(docs) != 2 {
		t.Errorf("expected 2 documents, got %d", len(docs))
	}
}

func TestSeedFromCorpusIdempotent(t *testing.T) {
	tempDir := t.TempDir()
	lib, err := Init(filepath.Join(tempDir, "lib"), "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	testdataDir := filepath.Join("..", "..", "testdata")
	entries := []CorpusEntry{
		{ID: "us-va-vcdpa", Jurisdiction: "US-VA", ShortName: "VCDPA", Format: "us", SourcePath: "vcdpa.txt"},
	}

	// First seed
	_, err = SeedFromCorpus(lib, testdataDir, entries)
	if err != nil {
		t.Fatalf("first SeedFromCorpus failed: %v", err)
	}

	// Second seed should skip existing
	seedReport, err := SeedFromCorpus(lib, testdataDir, entries)
	if err != nil {
		t.Fatalf("second SeedFromCorpus failed: %v", err)
	}

	if seedReport.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", seedReport.Skipped)
	}
	if seedReport.Succeeded != 0 {
		t.Errorf("expected 0 succeeded, got %d", seedReport.Succeeded)
	}
}

func TestSeedFromCorpusMissingFile(t *testing.T) {
	tempDir := t.TempDir()
	lib, err := Init(filepath.Join(tempDir, "lib"), "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	entries := []CorpusEntry{
		{ID: "nonexistent", SourcePath: "nonexistent.txt"},
	}

	seedReport, err := SeedFromCorpus(lib, "/nonexistent/dir", entries)
	if err != nil {
		t.Fatalf("SeedFromCorpus failed: %v", err)
	}

	if seedReport.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", seedReport.Failed)
	}
}

func TestSeedFromDirectory(t *testing.T) {
	tempDir := t.TempDir()
	lib, err := Init(filepath.Join(tempDir, "lib"), "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	testdataDir := filepath.Join("..", "..", "testdata")

	seedReport, err := SeedFromDirectory(lib, testdataDir)
	if err != nil {
		t.Fatalf("SeedFromDirectory failed: %v", err)
	}

	// Should find all .txt files in testdata
	if seedReport.TotalAttempted == 0 {
		t.Error("expected some files to be attempted")
	}
	if seedReport.Succeeded == 0 {
		t.Error("expected some files to succeed")
	}
}

func TestDefaultCorpusEntries(t *testing.T) {
	entries := DefaultCorpusEntries()
	if len(entries) < 15 {
		t.Errorf("expected at least 15 corpus entries, got %d", len(entries))
	}

	// Check that each entry has required fields
	for _, corpusEntry := range entries {
		if corpusEntry.ID == "" {
			t.Error("corpus entry has empty ID")
		}
		if corpusEntry.SourcePath == "" {
			t.Errorf("corpus entry %s has empty SourcePath", corpusEntry.ID)
		}
	}
}
