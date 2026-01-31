package bulk

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DownloadManifest tracks which datasets have been downloaded for resumability.
type DownloadManifest struct {
	Version   string                     `json:"version"`
	UpdatedAt time.Time                  `json:"updated_at"`
	Downloads map[string]*DownloadRecord `json:"downloads"`
}

// DownloadRecord tracks a single completed download.
type DownloadRecord struct {
	Identifier   string    `json:"identifier"`
	SourceName   string    `json:"source_name"`
	URL          string    `json:"url"`
	LocalPath    string    `json:"local_path"`
	SizeBytes    int64     `json:"size_bytes"`
	DownloadedAt time.Time `json:"downloaded_at"`
}

const manifestVersion = "1.0.0"

// NewDownloadManifest creates an empty manifest.
func NewDownloadManifest() *DownloadManifest {
	return &DownloadManifest{
		Version:   manifestVersion,
		UpdatedAt: time.Now(),
		Downloads: make(map[string]*DownloadRecord),
	}
}

// LoadManifest reads a download manifest from disk.
func LoadManifest(manifestPath string) (*DownloadManifest, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewDownloadManifest(), nil
		}
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	manifest := &DownloadManifest{}
	if err := json.Unmarshal(data, manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	if manifest.Downloads == nil {
		manifest.Downloads = make(map[string]*DownloadRecord)
	}

	return manifest, nil
}

// SaveManifest writes the manifest to disk.
func (manifest *DownloadManifest) SaveManifest(manifestPath string) error {
	manifest.UpdatedAt = time.Now()

	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %w", err)
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// RecordDownload adds a completed download to the manifest.
func (manifest *DownloadManifest) RecordDownload(record *DownloadRecord) {
	manifest.Downloads[record.Identifier] = record
}

// IsDownloaded checks if a dataset has already been downloaded.
func (manifest *DownloadManifest) IsDownloaded(identifier string) bool {
	_, exists := manifest.Downloads[identifier]
	return exists
}

// GetRecord returns the download record for an identifier, or nil.
func (manifest *DownloadManifest) GetRecord(identifier string) *DownloadRecord {
	return manifest.Downloads[identifier]
}

// CountBySource returns the number of downloads for a given source name.
func (manifest *DownloadManifest) CountBySource(sourceName string) int {
	count := 0
	for _, record := range manifest.Downloads {
		if record.SourceName == sourceName {
			count++
		}
	}
	return count
}

// TotalSizeBySource returns total bytes downloaded for a given source.
func (manifest *DownloadManifest) TotalSizeBySource(sourceName string) int64 {
	var totalBytes int64
	for _, record := range manifest.Downloads {
		if record.SourceName == sourceName {
			totalBytes += record.SizeBytes
		}
	}
	return totalBytes
}
