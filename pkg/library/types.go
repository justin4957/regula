package library

import (
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// DocumentStatus represents the state of a document in the library.
type DocumentStatus string

const (
	// StatusReady indicates the document has been ingested and is available for queries.
	StatusReady DocumentStatus = "ready"

	// StatusIngesting indicates the document is currently being ingested.
	StatusIngesting DocumentStatus = "ingesting"

	// StatusFailed indicates ingestion failed for this document.
	StatusFailed DocumentStatus = "failed"
)

// LibraryManifest is the top-level index of all documents in the library.
type LibraryManifest struct {
	Version   string           `json:"version"`
	BaseURI   string           `json:"base_uri"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	Documents []*DocumentEntry `json:"documents"`
}

// DocumentEntry represents a single legislation document stored in the library.
type DocumentEntry struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	ShortName    string           `json:"short_name"`
	FullName     string           `json:"full_name"`
	Jurisdiction string           `json:"jurisdiction"`
	Format       string           `json:"format"`
	Tags         []string         `json:"tags,omitempty"`
	Status       DocumentStatus   `json:"status"`
	IngestedAt   time.Time        `json:"ingested_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	SourceInfo   string           `json:"source_info,omitempty"`
	Stats        *DocumentStats   `json:"stats,omitempty"`
	StorageHash  string           `json:"storage_hash"`
	Error        string           `json:"error,omitempty"`
}

// DocumentStats holds extraction statistics for a single document.
type DocumentStats struct {
	TotalTriples int `json:"total_triples"`
	Articles     int `json:"articles"`
	Chapters     int `json:"chapters"`
	Sections     int `json:"sections"`
	Recitals     int `json:"recitals"`
	Definitions  int `json:"definitions"`
	References   int `json:"references"`
	Rights       int `json:"rights"`
	Obligations  int `json:"obligations"`
	TermUsages   int `json:"term_usages"`
	SourceBytes  int `json:"source_bytes"`
}

// SerializedTriple is a JSON-serializable representation of an RDF triple.
type SerializedTriple struct {
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
}

// ToStoreTriple converts a SerializedTriple to a store.Triple.
func (st SerializedTriple) ToStoreTriple() store.Triple {
	return store.NewTriple(st.Subject, st.Predicate, st.Object)
}

// FromStoreTriple creates a SerializedTriple from a store.Triple.
func FromStoreTriple(triple store.Triple) SerializedTriple {
	return SerializedTriple{
		Subject:   triple.Subject,
		Predicate: triple.Predicate,
		Object:    triple.Object,
	}
}

// AddOptions configures how a document is added to the library.
type AddOptions struct {
	Name         string
	ShortName    string
	FullName     string
	Jurisdiction string
	Format       string
	Tags         []string
	SourceInfo   string
	BaseURI      string
	Force        bool // overwrite existing document with same ID
}

// LibraryStats aggregates statistics across all documents in the library.
type LibraryStats struct {
	TotalDocuments   int            `json:"total_documents"`
	TotalTriples     int            `json:"total_triples"`
	TotalArticles    int            `json:"total_articles"`
	TotalDefinitions int            `json:"total_definitions"`
	TotalReferences  int            `json:"total_references"`
	TotalRights      int            `json:"total_rights"`
	TotalObligations int            `json:"total_obligations"`
	ByJurisdiction   map[string]int `json:"by_jurisdiction"`
	ByStatus         map[string]int `json:"by_status"`
}

// CorpusEntry describes a testdata document available for seeding.
type CorpusEntry struct {
	ID           string `json:"id"`
	Jurisdiction string `json:"jurisdiction"`
	ShortName    string `json:"short_name"`
	FullName     string `json:"full_name"`
	Format       string `json:"format"`
	SourcePath   string `json:"source_path"`
	SourceInfo   string `json:"source_info,omitempty"`
}

// SeedReport summarizes the results of a corpus seeding operation.
type SeedReport struct {
	TotalAttempted int              `json:"total_attempted"`
	Succeeded      int              `json:"succeeded"`
	Skipped        int              `json:"skipped"`
	Failed         int              `json:"failed"`
	Entries        []SeedEntryState `json:"entries"`
}

// SeedEntryState records the outcome of seeding a single document.
type SeedEntryState struct {
	ID     string `json:"id"`
	Status string `json:"status"` // "ingested", "skipped", "failed"
	Error  string `json:"error,omitempty"`
}

// IngestResult holds the output of a single document ingestion.
type IngestResult struct {
	TripleStore *store.TripleStore
	Stats       *DocumentStats
	DocumentID  string
	RegID       string
}
