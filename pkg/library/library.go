package library

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

const (
	manifestFileName = "library.json"
	documentsDir     = "documents"
	sourceFileName   = "source.txt"
	triplesFileName  = "triples.json"
	metadataFileName = "metadata.json"
	manifestVersion  = "1.0.0"
)

// Library manages a persistent collection of ingested legislation documents.
type Library struct {
	mu       sync.RWMutex
	path     string
	manifest *LibraryManifest
}

// Init creates a new library at the given path with default settings.
func Init(libraryPath string, baseURI string) (*Library, error) {
	if baseURI == "" {
		baseURI = defaultBaseURI
	}

	// Create directory structure
	documentsPath := filepath.Join(libraryPath, documentsDir)
	if err := os.MkdirAll(documentsPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create library directory: %w", err)
	}

	manifest := &LibraryManifest{
		Version:   manifestVersion,
		BaseURI:   baseURI,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Documents: []*DocumentEntry{},
	}

	lib := &Library{
		path:     libraryPath,
		manifest: manifest,
	}

	if err := lib.saveManifest(); err != nil {
		return nil, fmt.Errorf("failed to save manifest: %w", err)
	}

	return lib, nil
}

// Open loads an existing library from disk.
func Open(libraryPath string) (*Library, error) {
	manifestPath := filepath.Join(libraryPath, manifestFileName)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read library manifest: %w", err)
	}

	var manifest LibraryManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse library manifest: %w", err)
	}

	return &Library{
		path:     libraryPath,
		manifest: &manifest,
	}, nil
}

// AddDocument ingests source text and stores it in the library.
func (lib *Library) AddDocument(documentID string, sourceText []byte, opts AddOptions) (*DocumentEntry, error) {
	lib.mu.Lock()
	defer lib.mu.Unlock()

	if documentID == "" {
		return nil, fmt.Errorf("document ID is required")
	}

	// Check for existing document
	existing := lib.findDocumentUnsafe(documentID)
	if existing != nil && !opts.Force {
		return existing, nil // idempotent: return existing entry
	}

	baseURI := opts.BaseURI
	if baseURI == "" {
		baseURI = lib.manifest.BaseURI
	}

	// Run ingestion pipeline with format hint from options
	result, err := IngestFromText(sourceText, documentID, baseURI, opts.Format)
	if err != nil {
		// Record failure
		entry := &DocumentEntry{
			ID:          documentID,
			Name:        opts.Name,
			ShortName:   opts.ShortName,
			FullName:    opts.FullName,
			Status:      StatusFailed,
			IngestedAt:  time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
			StorageHash: hashDocumentID(documentID),
			Error:       err.Error(),
		}
		lib.upsertEntry(entry)
		if saveErr := lib.saveManifest(); saveErr != nil {
			return nil, fmt.Errorf("ingestion failed (%v) and failed to save manifest: %w", err, saveErr)
		}
		return nil, fmt.Errorf("ingestion failed for %s: %w", documentID, err)
	}

	storageHash := hashDocumentID(documentID)

	// Persist source text
	if err := lib.writeDocumentFile(storageHash, sourceFileName, sourceText); err != nil {
		return nil, fmt.Errorf("failed to save source: %w", err)
	}

	// Persist serialized triples
	triplesData, err := SerializeTripleStore(result.TripleStore)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize triples: %w", err)
	}
	if err := lib.writeDocumentFile(storageHash, triplesFileName, triplesData); err != nil {
		return nil, fmt.Errorf("failed to save triples: %w", err)
	}

	// Persist metadata
	metadataBytes, err := json.MarshalIndent(result.Stats, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if err := lib.writeDocumentFile(storageHash, metadataFileName, metadataBytes); err != nil {
		return nil, fmt.Errorf("failed to save metadata: %w", err)
	}

	entry := &DocumentEntry{
		ID:           documentID,
		Name:         opts.Name,
		ShortName:    opts.ShortName,
		FullName:     opts.FullName,
		Jurisdiction: opts.Jurisdiction,
		Format:       opts.Format,
		Tags:         opts.Tags,
		Status:       StatusReady,
		IngestedAt:   time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
		SourceInfo:   opts.SourceInfo,
		Stats:        result.Stats,
		StorageHash:  storageHash,
	}

	lib.upsertEntry(entry)

	if err := lib.saveManifest(); err != nil {
		return nil, fmt.Errorf("failed to save manifest: %w", err)
	}

	return entry, nil
}

// RemoveDocument deletes a document and its associated files from the library.
func (lib *Library) RemoveDocument(documentID string) error {
	lib.mu.Lock()
	defer lib.mu.Unlock()

	entry := lib.findDocumentUnsafe(documentID)
	if entry == nil {
		return fmt.Errorf("document not found: %s", documentID)
	}

	// Remove files
	documentPath := filepath.Join(lib.path, documentsDir, entry.StorageHash)
	if err := os.RemoveAll(documentPath); err != nil {
		return fmt.Errorf("failed to remove document files: %w", err)
	}

	// Remove from manifest
	lib.removeEntry(documentID)

	if err := lib.saveManifest(); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	return nil
}

// GetDocument returns the entry for a specific document.
func (lib *Library) GetDocument(documentID string) *DocumentEntry {
	lib.mu.RLock()
	defer lib.mu.RUnlock()
	return lib.findDocumentUnsafe(documentID)
}

// ListDocuments returns all document entries, sorted by ID.
func (lib *Library) ListDocuments() []*DocumentEntry {
	lib.mu.RLock()
	defer lib.mu.RUnlock()

	result := make([]*DocumentEntry, len(lib.manifest.Documents))
	copy(result, lib.manifest.Documents)

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result
}

// LoadTripleStore loads and deserializes a single document's triple store.
func (lib *Library) LoadTripleStore(documentID string) (*store.TripleStore, error) {
	lib.mu.RLock()
	defer lib.mu.RUnlock()

	entry := lib.findDocumentUnsafe(documentID)
	if entry == nil {
		return nil, fmt.Errorf("document not found: %s", documentID)
	}
	if entry.Status != StatusReady {
		return nil, fmt.Errorf("document %s is not ready (status: %s)", documentID, entry.Status)
	}

	data, err := lib.readDocumentFile(entry.StorageHash, triplesFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read triples for %s: %w", documentID, err)
	}

	return DeserializeTripleStore(data)
}

// LoadMergedTripleStore loads and merges triple stores for the specified documents.
func (lib *Library) LoadMergedTripleStore(documentIDs ...string) (*store.TripleStore, error) {
	merged := store.NewTripleStore()

	for _, documentID := range documentIDs {
		tripleStore, err := lib.LoadTripleStore(documentID)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", documentID, err)
		}
		merged.MergeFrom(tripleStore)
	}

	return merged, nil
}

// LoadAllTripleStores loads and merges all ready documents into a single store.
func (lib *Library) LoadAllTripleStores() (*store.TripleStore, error) {
	lib.mu.RLock()
	readyIDs := make([]string, 0)
	for _, entry := range lib.manifest.Documents {
		if entry.Status == StatusReady {
			readyIDs = append(readyIDs, entry.ID)
		}
	}
	lib.mu.RUnlock()

	return lib.LoadMergedTripleStore(readyIDs...)
}

// LoadSourceText returns the original source text for a document.
func (lib *Library) LoadSourceText(documentID string) ([]byte, error) {
	lib.mu.RLock()
	defer lib.mu.RUnlock()

	entry := lib.findDocumentUnsafe(documentID)
	if entry == nil {
		return nil, fmt.Errorf("document not found: %s", documentID)
	}

	return lib.readDocumentFile(entry.StorageHash, sourceFileName)
}

// Stats returns aggregate statistics across all documents.
func (lib *Library) Stats() *LibraryStats {
	lib.mu.RLock()
	defer lib.mu.RUnlock()

	libraryStats := &LibraryStats{
		ByJurisdiction: make(map[string]int),
		ByStatus:       make(map[string]int),
	}

	for _, entry := range lib.manifest.Documents {
		libraryStats.TotalDocuments++
		libraryStats.ByStatus[string(entry.Status)]++

		if entry.Jurisdiction != "" {
			libraryStats.ByJurisdiction[entry.Jurisdiction]++
		}

		if entry.Stats != nil {
			libraryStats.TotalTriples += entry.Stats.TotalTriples
			libraryStats.TotalArticles += entry.Stats.Articles
			libraryStats.TotalDefinitions += entry.Stats.Definitions
			libraryStats.TotalReferences += entry.Stats.References
			libraryStats.TotalRights += entry.Stats.Rights
			libraryStats.TotalObligations += entry.Stats.Obligations
		}
	}

	return libraryStats
}

// Path returns the library's root directory.
func (lib *Library) Path() string {
	return lib.path
}

// BaseURI returns the library's base URI.
func (lib *Library) BaseURI() string {
	lib.mu.RLock()
	defer lib.mu.RUnlock()
	return lib.manifest.BaseURI
}

// Close is a no-op provided for interface consistency.
func (lib *Library) Close() error {
	return nil
}

// --- Internal helpers ---

func (lib *Library) findDocumentUnsafe(documentID string) *DocumentEntry {
	for _, entry := range lib.manifest.Documents {
		if entry.ID == documentID {
			return entry
		}
	}
	return nil
}

func (lib *Library) upsertEntry(entry *DocumentEntry) {
	for i, existing := range lib.manifest.Documents {
		if existing.ID == entry.ID {
			lib.manifest.Documents[i] = entry
			lib.manifest.UpdatedAt = time.Now().UTC()
			return
		}
	}
	lib.manifest.Documents = append(lib.manifest.Documents, entry)
	lib.manifest.UpdatedAt = time.Now().UTC()
}

func (lib *Library) removeEntry(documentID string) {
	filtered := make([]*DocumentEntry, 0, len(lib.manifest.Documents))
	for _, entry := range lib.manifest.Documents {
		if entry.ID != documentID {
			filtered = append(filtered, entry)
		}
	}
	lib.manifest.Documents = filtered
	lib.manifest.UpdatedAt = time.Now().UTC()
}

func (lib *Library) saveManifest() error {
	manifestPath := filepath.Join(lib.path, manifestFileName)
	data, err := json.MarshalIndent(lib.manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	return os.WriteFile(manifestPath, data, 0644)
}

func (lib *Library) documentDir(storageHash string) string {
	return filepath.Join(lib.path, documentsDir, storageHash)
}

func (lib *Library) writeDocumentFile(storageHash string, fileName string, data []byte) error {
	dirPath := lib.documentDir(storageHash)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dirPath, fileName), data, 0644)
}

func (lib *Library) readDocumentFile(storageHash string, fileName string) ([]byte, error) {
	return os.ReadFile(filepath.Join(lib.documentDir(storageHash), fileName))
}

func hashDocumentID(documentID string) string {
	hash := sha256.Sum256([]byte(documentID))
	return fmt.Sprintf("%x", hash)
}
