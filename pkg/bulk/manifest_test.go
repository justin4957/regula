package bulk

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewDownloadManifest(t *testing.T) {
	manifest := NewDownloadManifest()

	if manifest.Version != manifestVersion {
		t.Errorf("expected version %q, got %q", manifestVersion, manifest.Version)
	}
	if manifest.Downloads == nil {
		t.Fatal("expected Downloads map to be initialized")
	}
	if len(manifest.Downloads) != 0 {
		t.Errorf("expected 0 downloads, got %d", len(manifest.Downloads))
	}
}

func TestManifestSaveAndLoad(t *testing.T) {
	temporaryDir := t.TempDir()
	manifestPath := filepath.Join(temporaryDir, "manifest.json")

	original := NewDownloadManifest()
	original.RecordDownload(&DownloadRecord{
		Identifier:   "usc-title-42",
		SourceName:   "uscode",
		URL:          "https://example.com/usc42.zip",
		LocalPath:    "/downloads/usc42.zip",
		SizeBytes:    1024000,
		DownloadedAt: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
	})

	if err := original.SaveManifest(manifestPath); err != nil {
		t.Fatalf("failed to save manifest: %v", err)
	}

	loaded, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	if loaded.Version != original.Version {
		t.Errorf("version mismatch: got %q, want %q", loaded.Version, original.Version)
	}
	if len(loaded.Downloads) != 1 {
		t.Fatalf("expected 1 download, got %d", len(loaded.Downloads))
	}

	record := loaded.GetRecord("usc-title-42")
	if record == nil {
		t.Fatal("expected record for usc-title-42")
	}
	if record.SourceName != "uscode" {
		t.Errorf("expected source 'uscode', got %q", record.SourceName)
	}
	if record.SizeBytes != 1024000 {
		t.Errorf("expected 1024000 bytes, got %d", record.SizeBytes)
	}
}

func TestLoadManifestNonExistent(t *testing.T) {
	manifest, err := LoadManifest("/nonexistent/path/manifest.json")
	if err != nil {
		t.Fatalf("expected nil error for non-existent file, got: %v", err)
	}
	if manifest == nil {
		t.Fatal("expected non-nil manifest for non-existent file")
	}
	if len(manifest.Downloads) != 0 {
		t.Errorf("expected empty downloads, got %d", len(manifest.Downloads))
	}
}

func TestLoadManifestCorruptJSON(t *testing.T) {
	temporaryDir := t.TempDir()
	manifestPath := filepath.Join(temporaryDir, "manifest.json")

	os.WriteFile(manifestPath, []byte("not valid json"), 0644)

	_, err := LoadManifest(manifestPath)
	if err == nil {
		t.Fatal("expected error for corrupt JSON")
	}
}

func TestManifestRecordAndLookup(t *testing.T) {
	manifest := NewDownloadManifest()

	manifest.RecordDownload(&DownloadRecord{
		Identifier: "cfr-2024-title-21",
		SourceName: "cfr",
		SizeBytes:  5000000,
	})
	manifest.RecordDownload(&DownloadRecord{
		Identifier: "cfr-2024-title-42",
		SourceName: "cfr",
		SizeBytes:  8000000,
	})
	manifest.RecordDownload(&DownloadRecord{
		Identifier: "ca-civ",
		SourceName: "california",
		SizeBytes:  2000000,
	})

	if !manifest.IsDownloaded("cfr-2024-title-21") {
		t.Error("expected cfr-2024-title-21 to be downloaded")
	}
	if manifest.IsDownloaded("cfr-2024-title-99") {
		t.Error("expected cfr-2024-title-99 to not be downloaded")
	}
	if record := manifest.GetRecord("ca-civ"); record == nil {
		t.Error("expected record for ca-civ")
	}
	if record := manifest.GetRecord("nonexistent"); record != nil {
		t.Error("expected nil for nonexistent record")
	}
}

func TestManifestCountBySource(t *testing.T) {
	manifest := NewDownloadManifest()

	manifest.RecordDownload(&DownloadRecord{Identifier: "cfr-t21", SourceName: "cfr", SizeBytes: 5000000})
	manifest.RecordDownload(&DownloadRecord{Identifier: "cfr-t42", SourceName: "cfr", SizeBytes: 8000000})
	manifest.RecordDownload(&DownloadRecord{Identifier: "ca-civ", SourceName: "california", SizeBytes: 2000000})

	if count := manifest.CountBySource("cfr"); count != 2 {
		t.Errorf("expected 2 CFR downloads, got %d", count)
	}
	if count := manifest.CountBySource("california"); count != 1 {
		t.Errorf("expected 1 California download, got %d", count)
	}
	if count := manifest.CountBySource("archive"); count != 0 {
		t.Errorf("expected 0 archive downloads, got %d", count)
	}
}

func TestManifestTotalSizeBySource(t *testing.T) {
	manifest := NewDownloadManifest()

	manifest.RecordDownload(&DownloadRecord{Identifier: "cfr-t21", SourceName: "cfr", SizeBytes: 5000000})
	manifest.RecordDownload(&DownloadRecord{Identifier: "cfr-t42", SourceName: "cfr", SizeBytes: 8000000})
	manifest.RecordDownload(&DownloadRecord{Identifier: "ca-civ", SourceName: "california", SizeBytes: 2000000})

	if totalBytes := manifest.TotalSizeBySource("cfr"); totalBytes != 13000000 {
		t.Errorf("expected 13000000 bytes for CFR, got %d", totalBytes)
	}
	if totalBytes := manifest.TotalSizeBySource("california"); totalBytes != 2000000 {
		t.Errorf("expected 2000000 bytes for California, got %d", totalBytes)
	}
	if totalBytes := manifest.TotalSizeBySource("archive"); totalBytes != 0 {
		t.Errorf("expected 0 bytes for archive, got %d", totalBytes)
	}
}

func TestManifestOverwriteRecord(t *testing.T) {
	manifest := NewDownloadManifest()

	manifest.RecordDownload(&DownloadRecord{
		Identifier: "usc-title-42",
		SourceName: "uscode",
		SizeBytes:  1000,
	})
	manifest.RecordDownload(&DownloadRecord{
		Identifier: "usc-title-42",
		SourceName: "uscode",
		SizeBytes:  2000,
	})

	if len(manifest.Downloads) != 1 {
		t.Errorf("expected 1 record after overwrite, got %d", len(manifest.Downloads))
	}
	if record := manifest.GetRecord("usc-title-42"); record.SizeBytes != 2000 {
		t.Errorf("expected updated size 2000, got %d", record.SizeBytes)
	}
}
