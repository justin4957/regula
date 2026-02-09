// Package deliberation provides types and functions for modeling deliberation
// documents including source watching and auto-ingestion.
package deliberation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// WatcherDocumentType classifies the type of document for parser selection.
type WatcherDocumentType string

const (
	// DocTypeMeetingMinutes represents meeting minutes documents.
	DocTypeMeetingMinutes WatcherDocumentType = "meeting-minutes"
	// DocTypeResolution represents resolution documents.
	DocTypeResolution WatcherDocumentType = "resolution"
	// DocTypeWorkingPaper represents working paper documents.
	DocTypeWorkingPaper WatcherDocumentType = "working-paper"
	// DocTypeRegulation represents regulation documents.
	DocTypeRegulation WatcherDocumentType = "regulation"
	// DocTypeUnknown represents unknown document types.
	DocTypeUnknown WatcherDocumentType = "unknown"
)

// String returns the string representation of a WatcherDocumentType.
func (t WatcherDocumentType) String() string {
	return string(t)
}

// ParseWatcherDocumentType parses a string into a WatcherDocumentType.
func ParseWatcherDocumentType(s string) WatcherDocumentType {
	switch strings.ToLower(s) {
	case "meeting-minutes", "meeting_minutes", "meetingminutes":
		return DocTypeMeetingMinutes
	case "resolution", "resolutions":
		return DocTypeResolution
	case "working-paper", "working_paper", "workingpaper":
		return DocTypeWorkingPaper
	case "regulation", "regulations":
		return DocTypeRegulation
	default:
		return DocTypeUnknown
	}
}

// WatchedDocument represents a document discovered by a watcher.
type WatchedDocument struct {
	// ID is a unique identifier for the document.
	ID string `json:"id" yaml:"id"`

	// Title is the document title.
	Title string `json:"title" yaml:"title"`

	// URL is the location where the document can be fetched.
	URL string `json:"url" yaml:"url"`

	// Path is the local file path (for filesystem sources).
	Path string `json:"path,omitempty" yaml:"path,omitempty"`

	// PublishedAt is when the document was published/modified.
	PublishedAt time.Time `json:"published_at" yaml:"published_at"`

	// Type is the document type for parser selection.
	Type WatcherDocumentType `json:"type" yaml:"type"`

	// Source is the name of the source that found this document.
	Source string `json:"source" yaml:"source"`

	// Metadata contains additional key-value pairs.
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Size is the document size in bytes (if known).
	Size int64 `json:"size,omitempty" yaml:"size,omitempty"`

	// Checksum is a hash of the document content (for change detection).
	Checksum string `json:"checksum,omitempty" yaml:"checksum,omitempty"`
}

// DocumentSource is the interface for document sources.
type DocumentSource interface {
	// Name returns the source identifier.
	Name() string

	// Check returns new documents since the given time.
	Check(ctx context.Context, since time.Time) ([]WatchedDocument, error)

	// Fetch retrieves document content.
	Fetch(ctx context.Context, doc WatchedDocument) ([]byte, error)

	// Type returns the document type for a document.
	Type(doc WatchedDocument) WatcherDocumentType
}

// DocumentParser is the interface for parsing document content.
type DocumentParser interface {
	// Parse parses document content and adds triples to the store.
	Parse(content []byte, docURI string, store *store.TripleStore) error
}

// DocumentCallback is called when a new document is detected.
type DocumentCallback func(doc WatchedDocument)

// IngestCallback is called after a document is ingested.
type IngestCallback func(doc WatchedDocument, tripleCount int, err error)

// WatcherConfig holds configuration for the watcher manager.
type WatcherConfig struct {
	// DefaultInterval is the default check interval.
	DefaultInterval time.Duration `json:"default_interval" yaml:"default_interval"`

	// AutoIngest enables automatic ingestion of new documents.
	AutoIngest bool `json:"auto_ingest" yaml:"auto_ingest"`

	// MaxConcurrent limits concurrent document fetches.
	MaxConcurrent int `json:"max_concurrent" yaml:"max_concurrent"`

	// RetryAttempts is the number of retry attempts for failed fetches.
	RetryAttempts int `json:"retry_attempts" yaml:"retry_attempts"`

	// RetryDelay is the delay between retry attempts.
	RetryDelay time.Duration `json:"retry_delay" yaml:"retry_delay"`
}

// DefaultWatcherConfig returns a default watcher configuration.
func DefaultWatcherConfig() WatcherConfig {
	return WatcherConfig{
		DefaultInterval: 1 * time.Hour,
		AutoIngest:      true,
		MaxConcurrent:   5,
		RetryAttempts:   3,
		RetryDelay:      5 * time.Second,
	}
}

// WatcherManager manages multiple document sources and auto-ingestion.
type WatcherManager struct {
	// sources is the list of registered sources.
	sources []DocumentSource

	// store is the target triple store for ingestion.
	store *store.TripleStore

	// parsers maps document types to parsers.
	parsers map[WatcherDocumentType]DocumentParser

	// config holds the manager configuration.
	config WatcherConfig

	// lastCheck tracks the last check time for each source.
	lastCheck map[string]time.Time

	// callbacks are called when new documents are detected.
	callbacks []DocumentCallback

	// ingestCallbacks are called after ingestion.
	ingestCallbacks []IngestCallback

	// running indicates if the manager is running.
	running bool

	// stopCh is used to signal stop.
	stopCh chan struct{}

	// mu protects concurrent access.
	mu sync.RWMutex

	// wg tracks running goroutines.
	wg sync.WaitGroup

	// baseURI is the base URI for generating document URIs.
	baseURI string
}

// NewWatcherManager creates a new watcher manager.
func NewWatcherManager(tripleStore *store.TripleStore, baseURI string) *WatcherManager {
	return &WatcherManager{
		sources:         []DocumentSource{},
		store:           tripleStore,
		parsers:         make(map[WatcherDocumentType]DocumentParser),
		config:          DefaultWatcherConfig(),
		lastCheck:       make(map[string]time.Time),
		callbacks:       []DocumentCallback{},
		ingestCallbacks: []IngestCallback{},
		baseURI:         baseURI,
	}
}

// SetConfig sets the watcher configuration.
func (m *WatcherManager) SetConfig(config WatcherConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
}

// GetConfig returns the current configuration.
func (m *WatcherManager) GetConfig() WatcherConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// AddSource adds a document source to watch.
func (m *WatcherManager) AddSource(source DocumentSource) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sources = append(m.sources, source)
}

// RemoveSource removes a document source by name.
func (m *WatcherManager) RemoveSource(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, s := range m.sources {
		if s.Name() == name {
			m.sources = append(m.sources[:i], m.sources[i+1:]...)
			delete(m.lastCheck, name)
			return true
		}
	}
	return false
}

// Sources returns the list of registered sources.
func (m *WatcherManager) Sources() []DocumentSource {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]DocumentSource, len(m.sources))
	copy(result, m.sources)
	return result
}

// RegisterParser registers a parser for a document type.
func (m *WatcherManager) RegisterParser(docType WatcherDocumentType, parser DocumentParser) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.parsers[docType] = parser
}

// OnNewDocument registers a callback for new document detection.
func (m *WatcherManager) OnNewDocument(callback DocumentCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks = append(m.callbacks, callback)
}

// OnIngest registers a callback for after ingestion.
func (m *WatcherManager) OnIngest(callback IngestCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ingestCallbacks = append(m.ingestCallbacks, callback)
}

// Start begins watching sources at the configured interval.
func (m *WatcherManager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("watcher manager already running")
	}
	m.running = true
	m.stopCh = make(chan struct{})
	m.mu.Unlock()

	m.wg.Add(1)
	go m.watchLoop(ctx)

	return nil
}

// Stop stops the watcher manager.
func (m *WatcherManager) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	close(m.stopCh)
	m.mu.Unlock()

	m.wg.Wait()
}

// IsRunning returns whether the manager is running.
func (m *WatcherManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// watchLoop runs the main watching loop.
func (m *WatcherManager) watchLoop(ctx context.Context) {
	defer m.wg.Done()

	// Initial check
	m.checkAllSources(ctx)

	ticker := time.NewTicker(m.config.DefaultInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkAllSources(ctx)
		}
	}
}

// checkAllSources checks all sources for new documents.
func (m *WatcherManager) checkAllSources(ctx context.Context) {
	m.mu.RLock()
	sources := make([]DocumentSource, len(m.sources))
	copy(sources, m.sources)
	m.mu.RUnlock()

	for _, source := range sources {
		select {
		case <-ctx.Done():
			return
		default:
			m.checkSource(ctx, source)
		}
	}
}

// checkSource checks a single source for new documents.
func (m *WatcherManager) checkSource(ctx context.Context, source DocumentSource) {
	m.mu.RLock()
	since := m.lastCheck[source.Name()]
	m.mu.RUnlock()

	docs, err := source.Check(ctx, since)
	if err != nil {
		// Log error but continue
		return
	}

	m.mu.Lock()
	m.lastCheck[source.Name()] = time.Now()
	m.mu.Unlock()

	for _, doc := range docs {
		m.handleNewDocument(ctx, source, doc)
	}
}

// handleNewDocument processes a newly discovered document.
func (m *WatcherManager) handleNewDocument(ctx context.Context, source DocumentSource, doc WatchedDocument) {
	// Notify callbacks
	m.mu.RLock()
	callbacks := make([]DocumentCallback, len(m.callbacks))
	copy(callbacks, m.callbacks)
	autoIngest := m.config.AutoIngest
	m.mu.RUnlock()

	for _, cb := range callbacks {
		cb(doc)
	}

	// Auto-ingest if enabled
	if autoIngest {
		m.ingestDocument(ctx, source, doc)
	}
}

// ingestDocument fetches and ingests a document.
func (m *WatcherManager) ingestDocument(ctx context.Context, source DocumentSource, doc WatchedDocument) {
	var tripleCount int
	var ingestErr error

	defer func() {
		// Notify ingest callbacks
		m.mu.RLock()
		callbacks := make([]IngestCallback, len(m.ingestCallbacks))
		copy(callbacks, m.ingestCallbacks)
		m.mu.RUnlock()

		for _, cb := range callbacks {
			cb(doc, tripleCount, ingestErr)
		}
	}()

	// Fetch document content
	content, err := m.fetchWithRetry(ctx, source, doc)
	if err != nil {
		ingestErr = fmt.Errorf("fetch failed: %w", err)
		return
	}

	// Get parser for document type
	docType := source.Type(doc)
	m.mu.RLock()
	parser, hasParser := m.parsers[docType]
	m.mu.RUnlock()

	if !hasParser {
		ingestErr = fmt.Errorf("no parser registered for document type: %s", docType)
		return
	}

	// Parse and ingest
	docURI := m.generateDocumentURI(doc)
	countBefore := 0
	if m.store != nil {
		countBefore = m.store.Count()
	}

	err = parser.Parse(content, docURI, m.store)
	if err != nil {
		ingestErr = fmt.Errorf("parse failed: %w", err)
		return
	}

	if m.store != nil {
		tripleCount = m.store.Count() - countBefore
	}
}

// fetchWithRetry fetches document content with retry logic.
func (m *WatcherManager) fetchWithRetry(ctx context.Context, source DocumentSource, doc WatchedDocument) ([]byte, error) {
	m.mu.RLock()
	retryAttempts := m.config.RetryAttempts
	retryDelay := m.config.RetryDelay
	m.mu.RUnlock()

	var lastErr error
	for attempt := 0; attempt <= retryAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
			}
		}

		content, err := source.Fetch(ctx, doc)
		if err == nil {
			return content, nil
		}
		lastErr = err
	}

	return nil, lastErr
}

// generateDocumentURI generates a URI for a document.
func (m *WatcherManager) generateDocumentURI(doc WatchedDocument) string {
	if doc.URL != "" {
		return doc.URL
	}
	return m.baseURI + sanitizeForURI(doc.ID)
}

// CheckNow performs an immediate check of all sources.
func (m *WatcherManager) CheckNow(ctx context.Context) ([]WatchedDocument, error) {
	m.mu.RLock()
	sources := make([]DocumentSource, len(m.sources))
	copy(sources, m.sources)
	m.mu.RUnlock()

	var allDocs []WatchedDocument

	for _, source := range sources {
		select {
		case <-ctx.Done():
			return allDocs, ctx.Err()
		default:
		}

		m.mu.RLock()
		since := m.lastCheck[source.Name()]
		m.mu.RUnlock()

		docs, err := source.Check(ctx, since)
		if err != nil {
			continue
		}

		m.mu.Lock()
		m.lastCheck[source.Name()] = time.Now()
		m.mu.Unlock()

		allDocs = append(allDocs, docs...)
	}

	return allDocs, nil
}

// IngestDocument manually ingests a document.
func (m *WatcherManager) IngestDocument(ctx context.Context, source DocumentSource, doc WatchedDocument) error {
	content, err := source.Fetch(ctx, doc)
	if err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}

	docType := source.Type(doc)
	m.mu.RLock()
	parser, hasParser := m.parsers[docType]
	m.mu.RUnlock()

	if !hasParser {
		return fmt.Errorf("no parser registered for document type: %s", docType)
	}

	docURI := m.generateDocumentURI(doc)
	return parser.Parse(content, docURI, m.store)
}

// GetLastCheck returns the last check time for a source.
func (m *WatcherManager) GetLastCheck(sourceName string) (time.Time, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.lastCheck[sourceName]
	return t, ok
}

// SetLastCheck sets the last check time for a source.
func (m *WatcherManager) SetLastCheck(sourceName string, t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastCheck[sourceName] = t
}

// FileSystemSource watches a local directory for documents.
type FileSystemSource struct {
	name        string
	path        string
	patterns    []string
	docType     WatcherDocumentType
	recursive   bool
	knownFiles  map[string]time.Time
	mu          sync.RWMutex
}

// FileSystemSourceConfig holds configuration for a filesystem source.
type FileSystemSourceConfig struct {
	// Name is the source identifier.
	Name string `json:"name" yaml:"name"`

	// Path is the directory to watch.
	Path string `json:"path" yaml:"path"`

	// Patterns are glob patterns for file matching (e.g., "*.pdf", "*.docx").
	Patterns []string `json:"patterns" yaml:"patterns"`

	// DocumentType is the type of documents in this directory.
	DocumentType WatcherDocumentType `json:"document_type" yaml:"document_type"`

	// Recursive enables recursive directory scanning.
	Recursive bool `json:"recursive" yaml:"recursive"`
}

// NewFileSystemSource creates a new filesystem source.
func NewFileSystemSource(config FileSystemSourceConfig) *FileSystemSource {
	if len(config.Patterns) == 0 {
		config.Patterns = []string{"*"}
	}
	return &FileSystemSource{
		name:       config.Name,
		path:       config.Path,
		patterns:   config.Patterns,
		docType:    config.DocumentType,
		recursive:  config.Recursive,
		knownFiles: make(map[string]time.Time),
	}
}

// Name returns the source identifier.
func (s *FileSystemSource) Name() string {
	return s.name
}

// Check returns new or modified files since the given time.
func (s *FileSystemSource) Check(ctx context.Context, since time.Time) ([]WatchedDocument, error) {
	var docs []WatchedDocument

	err := s.walkFiles(ctx, func(path string, info os.FileInfo) error {
		// Check if file matches patterns
		if !s.matchesPatterns(filepath.Base(path)) {
			return nil
		}

		modTime := info.ModTime()

		// Check if file is new or modified
		s.mu.RLock()
		lastMod, known := s.knownFiles[path]
		s.mu.RUnlock()

		isNew := !known || modTime.After(lastMod)
		isAfterSince := since.IsZero() || modTime.After(since)

		if isNew && isAfterSince {
			doc := WatchedDocument{
				ID:          path,
				Title:       filepath.Base(path),
				Path:        path,
				URL:         "file://" + path,
				PublishedAt: modTime,
				Type:        s.docType,
				Source:      s.name,
				Size:        info.Size(),
				Metadata:    make(map[string]string),
			}
			doc.Metadata["extension"] = filepath.Ext(path)
			docs = append(docs, doc)

			s.mu.Lock()
			s.knownFiles[path] = modTime
			s.mu.Unlock()
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by modification time
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].PublishedAt.Before(docs[j].PublishedAt)
	})

	return docs, nil
}

// walkFiles walks the directory tree.
func (s *FileSystemSource) walkFiles(ctx context.Context, fn func(string, os.FileInfo) error) error {
	if s.recursive {
		return filepath.Walk(s.path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip files with errors
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if info.IsDir() {
				return nil
			}
			return fn(path, info)
		})
	}

	entries, err := os.ReadDir(s.path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if entry.IsDir() {
			continue
		}

		path := filepath.Join(s.path, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if err := fn(path, info); err != nil {
			return err
		}
	}

	return nil
}

// matchesPatterns checks if a filename matches any of the configured patterns.
func (s *FileSystemSource) matchesPatterns(name string) bool {
	for _, pattern := range s.patterns {
		matched, _ := filepath.Match(pattern, name)
		if matched {
			return true
		}
	}
	return false
}

// Fetch reads the file content.
func (s *FileSystemSource) Fetch(ctx context.Context, doc WatchedDocument) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	path := doc.Path
	if path == "" {
		path = doc.ID
	}

	return os.ReadFile(path)
}

// Type returns the document type.
func (s *FileSystemSource) Type(doc WatchedDocument) WatcherDocumentType {
	if doc.Type != "" && doc.Type != DocTypeUnknown {
		return doc.Type
	}
	return s.docType
}

// Reset clears the known files cache.
func (s *FileSystemSource) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.knownFiles = make(map[string]time.Time)
}

// MemorySource is a document source backed by in-memory documents for testing.
type MemorySource struct {
	name      string
	documents []WatchedDocument
	content   map[string][]byte
	docType   WatcherDocumentType
	mu        sync.RWMutex
}

// NewMemorySource creates a new memory source for testing.
func NewMemorySource(name string, docType WatcherDocumentType) *MemorySource {
	return &MemorySource{
		name:      name,
		documents: []WatchedDocument{},
		content:   make(map[string][]byte),
		docType:   docType,
	}
}

// Name returns the source identifier.
func (s *MemorySource) Name() string {
	return s.name
}

// AddDocument adds a document to the source.
func (s *MemorySource) AddDocument(doc WatchedDocument, content []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc.Source = s.name
	if doc.Type == "" {
		doc.Type = s.docType
	}
	s.documents = append(s.documents, doc)
	s.content[doc.ID] = content
}

// Check returns documents published after the given time.
func (s *MemorySource) Check(ctx context.Context, since time.Time) ([]WatchedDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var docs []WatchedDocument
	for _, doc := range s.documents {
		if since.IsZero() || doc.PublishedAt.After(since) {
			docs = append(docs, doc)
		}
	}

	return docs, nil
}

// Fetch retrieves document content.
func (s *MemorySource) Fetch(ctx context.Context, doc WatchedDocument) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	content, ok := s.content[doc.ID]
	if !ok {
		return nil, fmt.Errorf("document not found: %s", doc.ID)
	}

	return content, nil
}

// Type returns the document type.
func (s *MemorySource) Type(doc WatchedDocument) WatcherDocumentType {
	if doc.Type != "" && doc.Type != DocTypeUnknown {
		return doc.Type
	}
	return s.docType
}

// Clear removes all documents.
func (s *MemorySource) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.documents = []WatchedDocument{}
	s.content = make(map[string][]byte)
}

// WatcherSourceConfig represents a watcher source configuration from YAML.
type WatcherSourceConfig struct {
	// Name is the source identifier.
	Name string `json:"name" yaml:"name"`

	// Type is the source type (filesystem, rss, api, scraper).
	Type string `json:"type" yaml:"type"`

	// URL is the source URL (for rss, api, scraper types).
	URL string `json:"url,omitempty" yaml:"url,omitempty"`

	// Path is the directory path (for filesystem type).
	Path string `json:"path,omitempty" yaml:"path,omitempty"`

	// Patterns are file patterns (for filesystem type).
	Patterns []string `json:"patterns,omitempty" yaml:"patterns,omitempty"`

	// Interval is the check interval.
	Interval time.Duration `json:"interval,omitempty" yaml:"interval,omitempty"`

	// DocumentType is the type of documents from this source.
	DocumentType string `json:"document_type" yaml:"document_type"`

	// Recursive enables recursive scanning (for filesystem type).
	Recursive bool `json:"recursive,omitempty" yaml:"recursive,omitempty"`
}

// WatchersConfig holds the complete watcher configuration.
type WatchersConfig struct {
	// Watchers is the list of watcher configurations.
	Watchers []WatcherSourceConfig `json:"watchers" yaml:"watchers"`
}

// CreateSourceFromConfig creates a DocumentSource from configuration.
func CreateSourceFromConfig(config WatcherSourceConfig) (DocumentSource, error) {
	switch strings.ToLower(config.Type) {
	case "filesystem", "fs", "file":
		return NewFileSystemSource(FileSystemSourceConfig{
			Name:         config.Name,
			Path:         config.Path,
			Patterns:     config.Patterns,
			DocumentType: ParseWatcherDocumentType(config.DocumentType),
			Recursive:    config.Recursive,
		}), nil

	case "memory", "test":
		return NewMemorySource(config.Name, ParseWatcherDocumentType(config.DocumentType)), nil

	default:
		return nil, fmt.Errorf("unsupported source type: %s", config.Type)
	}
}

// WatcherStats holds statistics about watcher activity.
type WatcherStats struct {
	// SourceCount is the number of registered sources.
	SourceCount int `json:"source_count"`

	// DocumentsFound is the total documents found.
	DocumentsFound int `json:"documents_found"`

	// DocumentsIngested is the total documents ingested.
	DocumentsIngested int `json:"documents_ingested"`

	// TriplesAdded is the total triples added.
	TriplesAdded int `json:"triples_added"`

	// Errors is the count of errors encountered.
	Errors int `json:"errors"`

	// LastCheck is the most recent check time.
	LastCheck time.Time `json:"last_check"`

	// IsRunning indicates if the manager is running.
	IsRunning bool `json:"is_running"`
}

// Stats returns statistics about the watcher manager.
func (m *WatcherManager) Stats() WatcherStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := WatcherStats{
		SourceCount: len(m.sources),
		IsRunning:   m.running,
	}

	// Find most recent check time
	for _, t := range m.lastCheck {
		if t.After(stats.LastCheck) {
			stats.LastCheck = t
		}
	}

	return stats
}
