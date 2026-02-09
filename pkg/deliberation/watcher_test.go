package deliberation

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// MockParser implements DocumentParser for testing.
type MockParser struct {
	ParseFunc func(content []byte, docURI string, store *store.TripleStore) error
	CallCount int
	mu        sync.Mutex
}

func (p *MockParser) Parse(content []byte, docURI string, s *store.TripleStore) error {
	p.mu.Lock()
	p.CallCount++
	p.mu.Unlock()

	if p.ParseFunc != nil {
		return p.ParseFunc(content, docURI, s)
	}
	// Default: add a simple triple
	if s != nil {
		s.Add(docURI, store.RDFType, "reg:Document")
		s.Add(docURI, store.PropText, string(content))
	}
	return nil
}

func TestNewWatcherManager(t *testing.T) {
	ts := store.NewTripleStore()
	manager := NewWatcherManager(ts, "https://example.org/")

	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}
	if manager.store != ts {
		t.Error("Expected store to be set")
	}
	if len(manager.sources) != 0 {
		t.Error("Expected no sources initially")
	}
}

func TestWatcherManager_AddSource(t *testing.T) {
	manager := NewWatcherManager(nil, "https://example.org/")

	source := NewMemorySource("test", DocTypeMeetingMinutes)
	manager.AddSource(source)

	sources := manager.Sources()
	if len(sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(sources))
	}
	if sources[0].Name() != "test" {
		t.Errorf("Expected source name 'test', got %s", sources[0].Name())
	}
}

func TestWatcherManager_RemoveSource(t *testing.T) {
	manager := NewWatcherManager(nil, "https://example.org/")

	source1 := NewMemorySource("source1", DocTypeMeetingMinutes)
	source2 := NewMemorySource("source2", DocTypeResolution)
	manager.AddSource(source1)
	manager.AddSource(source2)

	removed := manager.RemoveSource("source1")
	if !removed {
		t.Error("Expected source to be removed")
	}

	sources := manager.Sources()
	if len(sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(sources))
	}
	if sources[0].Name() != "source2" {
		t.Errorf("Expected source2, got %s", sources[0].Name())
	}

	// Try to remove non-existent source
	removed = manager.RemoveSource("nonexistent")
	if removed {
		t.Error("Expected false for non-existent source")
	}
}

func TestWatcherManager_RegisterParser(t *testing.T) {
	manager := NewWatcherManager(nil, "https://example.org/")

	parser := &MockParser{}
	manager.RegisterParser(DocTypeMeetingMinutes, parser)

	// Parser should be registered (we can't directly check, but it should work in ingest)
}

func TestWatcherManager_OnNewDocument(t *testing.T) {
	manager := NewWatcherManager(nil, "https://example.org/")

	var receivedDoc WatchedDocument
	var callbackCalled bool

	manager.OnNewDocument(func(doc WatchedDocument) {
		callbackCalled = true
		receivedDoc = doc
	})

	// Simulate handling a new document
	source := NewMemorySource("test", DocTypeMeetingMinutes)
	doc := WatchedDocument{
		ID:          "doc1",
		Title:       "Test Document",
		PublishedAt: time.Now(),
	}
	source.AddDocument(doc, []byte("content"))
	manager.AddSource(source)

	ctx := context.Background()
	docs, err := manager.CheckNow(ctx)
	if err != nil {
		t.Fatalf("CheckNow failed: %v", err)
	}

	// The callback is only called during Start loop or handleNewDocument
	// Let's test the callback mechanism directly
	manager.handleNewDocument(ctx, source, doc)

	if !callbackCalled {
		t.Error("Expected callback to be called")
	}
	if receivedDoc.ID != "doc1" {
		t.Errorf("Expected doc1, got %s", receivedDoc.ID)
	}

	_ = docs // suppress unused warning
}

func TestWatcherManager_CheckNow(t *testing.T) {
	manager := NewWatcherManager(nil, "https://example.org/")

	source := NewMemorySource("test", DocTypeMeetingMinutes)
	now := time.Now()
	doc1 := WatchedDocument{
		ID:          "doc1",
		Title:       "Document 1",
		PublishedAt: now.Add(-1 * time.Hour),
	}
	doc2 := WatchedDocument{
		ID:          "doc2",
		Title:       "Document 2",
		PublishedAt: now,
	}
	source.AddDocument(doc1, []byte("content1"))
	source.AddDocument(doc2, []byte("content2"))
	manager.AddSource(source)

	ctx := context.Background()
	docs, err := manager.CheckNow(ctx)
	if err != nil {
		t.Fatalf("CheckNow failed: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(docs))
	}
}

func TestWatcherManager_CheckNow_WithSince(t *testing.T) {
	manager := NewWatcherManager(nil, "https://example.org/")

	source := NewMemorySource("test", DocTypeMeetingMinutes)
	now := time.Now()
	doc1 := WatchedDocument{
		ID:          "doc1",
		Title:       "Old Document",
		PublishedAt: now.Add(-2 * time.Hour),
	}
	doc2 := WatchedDocument{
		ID:          "doc2",
		Title:       "New Document",
		PublishedAt: now,
	}
	source.AddDocument(doc1, []byte("content1"))
	source.AddDocument(doc2, []byte("content2"))
	manager.AddSource(source)

	// Set last check to 1 hour ago
	manager.SetLastCheck("test", now.Add(-1*time.Hour))

	ctx := context.Background()
	docs, err := manager.CheckNow(ctx)
	if err != nil {
		t.Fatalf("CheckNow failed: %v", err)
	}

	// Only doc2 should be returned (after since time)
	if len(docs) != 1 {
		t.Errorf("Expected 1 document, got %d", len(docs))
	}
	if len(docs) > 0 && docs[0].ID != "doc2" {
		t.Errorf("Expected doc2, got %s", docs[0].ID)
	}
}

func TestWatcherManager_IngestDocument(t *testing.T) {
	ts := store.NewTripleStore()
	manager := NewWatcherManager(ts, "https://example.org/")

	parser := &MockParser{}
	manager.RegisterParser(DocTypeMeetingMinutes, parser)

	source := NewMemorySource("test", DocTypeMeetingMinutes)
	doc := WatchedDocument{
		ID:          "doc1",
		Title:       "Test Document",
		PublishedAt: time.Now(),
		Type:        DocTypeMeetingMinutes,
	}
	source.AddDocument(doc, []byte("test content"))

	ctx := context.Background()
	err := manager.IngestDocument(ctx, source, doc)
	if err != nil {
		t.Fatalf("IngestDocument failed: %v", err)
	}

	if parser.CallCount != 1 {
		t.Errorf("Expected parser to be called once, got %d", parser.CallCount)
	}

	// Check that triples were added
	if ts.Count() == 0 {
		t.Error("Expected triples to be added")
	}
}

func TestWatcherManager_IngestDocument_NoParser(t *testing.T) {
	ts := store.NewTripleStore()
	manager := NewWatcherManager(ts, "https://example.org/")

	source := NewMemorySource("test", DocTypeMeetingMinutes)
	doc := WatchedDocument{
		ID:          "doc1",
		Title:       "Test Document",
		PublishedAt: time.Now(),
		Type:        DocTypeMeetingMinutes,
	}
	source.AddDocument(doc, []byte("test content"))

	ctx := context.Background()
	err := manager.IngestDocument(ctx, source, doc)
	if err == nil {
		t.Error("Expected error for missing parser")
	}
}

func TestWatcherManager_StartStop(t *testing.T) {
	manager := NewWatcherManager(nil, "https://example.org/")

	// Set short interval for testing
	config := manager.GetConfig()
	config.DefaultInterval = 100 * time.Millisecond
	manager.SetConfig(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !manager.IsRunning() {
		t.Error("Expected manager to be running")
	}

	// Try to start again (should fail)
	err = manager.Start(ctx)
	if err == nil {
		t.Error("Expected error for double start")
	}

	manager.Stop()

	if manager.IsRunning() {
		t.Error("Expected manager to be stopped")
	}
}

func TestWatcherManager_GetLastCheck(t *testing.T) {
	manager := NewWatcherManager(nil, "https://example.org/")

	// Initially no last check
	_, ok := manager.GetLastCheck("test")
	if ok {
		t.Error("Expected no last check initially")
	}

	// Set last check
	now := time.Now()
	manager.SetLastCheck("test", now)

	lastCheck, ok := manager.GetLastCheck("test")
	if !ok {
		t.Error("Expected last check to be set")
	}
	if !lastCheck.Equal(now) {
		t.Error("Expected last check time to match")
	}
}

func TestWatcherManager_Stats(t *testing.T) {
	manager := NewWatcherManager(nil, "https://example.org/")

	source1 := NewMemorySource("source1", DocTypeMeetingMinutes)
	source2 := NewMemorySource("source2", DocTypeResolution)
	manager.AddSource(source1)
	manager.AddSource(source2)

	stats := manager.Stats()

	if stats.SourceCount != 2 {
		t.Errorf("Expected 2 sources, got %d", stats.SourceCount)
	}
	if stats.IsRunning {
		t.Error("Expected not running")
	}
}

func TestMemorySource_Name(t *testing.T) {
	source := NewMemorySource("test-source", DocTypeMeetingMinutes)

	if source.Name() != "test-source" {
		t.Errorf("Expected 'test-source', got %s", source.Name())
	}
}

func TestMemorySource_AddDocument(t *testing.T) {
	source := NewMemorySource("test", DocTypeMeetingMinutes)

	doc := WatchedDocument{
		ID:          "doc1",
		Title:       "Test Doc",
		PublishedAt: time.Now(),
	}
	source.AddDocument(doc, []byte("content"))

	ctx := context.Background()
	docs, err := source.Check(ctx, time.Time{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("Expected 1 document, got %d", len(docs))
	}
	if docs[0].Source != "test" {
		t.Errorf("Expected source 'test', got %s", docs[0].Source)
	}
}

func TestMemorySource_Check(t *testing.T) {
	source := NewMemorySource("test", DocTypeMeetingMinutes)

	now := time.Now()
	doc1 := WatchedDocument{
		ID:          "doc1",
		PublishedAt: now.Add(-2 * time.Hour),
	}
	doc2 := WatchedDocument{
		ID:          "doc2",
		PublishedAt: now,
	}
	source.AddDocument(doc1, []byte("content1"))
	source.AddDocument(doc2, []byte("content2"))

	ctx := context.Background()

	// Check all
	docs, err := source.Check(ctx, time.Time{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(docs))
	}

	// Check since 1 hour ago
	docs, err = source.Check(ctx, now.Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("Expected 1 document, got %d", len(docs))
	}
}

func TestMemorySource_Fetch(t *testing.T) {
	source := NewMemorySource("test", DocTypeMeetingMinutes)

	doc := WatchedDocument{
		ID:          "doc1",
		PublishedAt: time.Now(),
	}
	source.AddDocument(doc, []byte("test content"))

	ctx := context.Background()
	content, err := source.Fetch(ctx, doc)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if string(content) != "test content" {
		t.Errorf("Expected 'test content', got %s", string(content))
	}
}

func TestMemorySource_Fetch_NotFound(t *testing.T) {
	source := NewMemorySource("test", DocTypeMeetingMinutes)

	doc := WatchedDocument{
		ID: "nonexistent",
	}

	ctx := context.Background()
	_, err := source.Fetch(ctx, doc)
	if err == nil {
		t.Error("Expected error for non-existent document")
	}
}

func TestMemorySource_Type(t *testing.T) {
	source := NewMemorySource("test", DocTypeMeetingMinutes)

	doc := WatchedDocument{
		ID: "doc1",
	}

	docType := source.Type(doc)
	if docType != DocTypeMeetingMinutes {
		t.Errorf("Expected DocTypeMeetingMinutes, got %s", docType)
	}

	// Document with explicit type
	doc.Type = DocTypeResolution
	docType = source.Type(doc)
	if docType != DocTypeResolution {
		t.Errorf("Expected DocTypeResolution, got %s", docType)
	}
}

func TestMemorySource_Clear(t *testing.T) {
	source := NewMemorySource("test", DocTypeMeetingMinutes)

	doc := WatchedDocument{
		ID:          "doc1",
		PublishedAt: time.Now(),
	}
	source.AddDocument(doc, []byte("content"))

	source.Clear()

	ctx := context.Background()
	docs, _ := source.Check(ctx, time.Time{})
	if len(docs) != 0 {
		t.Errorf("Expected 0 documents after clear, got %d", len(docs))
	}
}

func TestFileSystemSource_Name(t *testing.T) {
	source := NewFileSystemSource(FileSystemSourceConfig{
		Name: "fs-source",
		Path: "/tmp",
	})

	if source.Name() != "fs-source" {
		t.Errorf("Expected 'fs-source', got %s", source.Name())
	}
}

func TestFileSystemSource_Check(t *testing.T) {
	// Create temp directory with test files
	tmpDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	file1 := filepath.Join(tmpDir, "doc1.txt")
	file2 := filepath.Join(tmpDir, "doc2.pdf")
	file3 := filepath.Join(tmpDir, "other.log")

	os.WriteFile(file1, []byte("content1"), 0644)
	os.WriteFile(file2, []byte("content2"), 0644)
	os.WriteFile(file3, []byte("content3"), 0644)

	source := NewFileSystemSource(FileSystemSourceConfig{
		Name:         "test-fs",
		Path:         tmpDir,
		Patterns:     []string{"*.txt", "*.pdf"},
		DocumentType: DocTypeWorkingPaper,
	})

	ctx := context.Background()
	docs, err := source.Check(ctx, time.Time{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Should find 2 files (txt and pdf, not log)
	if len(docs) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(docs))
	}

	// Check document properties
	for _, doc := range docs {
		if doc.Source != "test-fs" {
			t.Errorf("Expected source 'test-fs', got %s", doc.Source)
		}
		if doc.Type != DocTypeWorkingPaper {
			t.Errorf("Expected DocTypeWorkingPaper, got %s", doc.Type)
		}
	}
}

func TestFileSystemSource_Check_Recursive(t *testing.T) {
	// Create temp directory with subdirectories
	tmpDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	subDir := filepath.Join(tmpDir, "subdir")
	os.MkdirAll(subDir, 0755)

	// Create test files
	file1 := filepath.Join(tmpDir, "doc1.txt")
	file2 := filepath.Join(subDir, "doc2.txt")

	os.WriteFile(file1, []byte("content1"), 0644)
	os.WriteFile(file2, []byte("content2"), 0644)

	// Non-recursive
	source := NewFileSystemSource(FileSystemSourceConfig{
		Name:      "test-fs",
		Path:      tmpDir,
		Patterns:  []string{"*.txt"},
		Recursive: false,
	})

	ctx := context.Background()
	docs, _ := source.Check(ctx, time.Time{})
	if len(docs) != 1 {
		t.Errorf("Expected 1 document (non-recursive), got %d", len(docs))
	}

	// Recursive
	source.Reset()
	source = NewFileSystemSource(FileSystemSourceConfig{
		Name:      "test-fs",
		Path:      tmpDir,
		Patterns:  []string{"*.txt"},
		Recursive: true,
	})

	docs, _ = source.Check(ctx, time.Time{})
	if len(docs) != 2 {
		t.Errorf("Expected 2 documents (recursive), got %d", len(docs))
	}
}

func TestFileSystemSource_Fetch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(filePath, []byte("test content"), 0644)

	source := NewFileSystemSource(FileSystemSourceConfig{
		Name: "test-fs",
		Path: tmpDir,
	})

	doc := WatchedDocument{
		ID:   filePath,
		Path: filePath,
	}

	ctx := context.Background()
	content, err := source.Fetch(ctx, doc)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if string(content) != "test content" {
		t.Errorf("Expected 'test content', got %s", string(content))
	}
}

func TestFileSystemSource_Check_OnlyNewFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	source := NewFileSystemSource(FileSystemSourceConfig{
		Name:     "test-fs",
		Path:     tmpDir,
		Patterns: []string{"*.txt"},
	})

	ctx := context.Background()

	// First check - no files
	docs, _ := source.Check(ctx, time.Time{})
	if len(docs) != 0 {
		t.Errorf("Expected 0 documents initially, got %d", len(docs))
	}

	// Create a file
	file1 := filepath.Join(tmpDir, "doc1.txt")
	os.WriteFile(file1, []byte("content"), 0644)

	// Second check - should find the new file
	docs, _ = source.Check(ctx, time.Time{})
	if len(docs) != 1 {
		t.Errorf("Expected 1 new document, got %d", len(docs))
	}

	// Third check - file is now known, should not return it
	docs, _ = source.Check(ctx, time.Time{})
	if len(docs) != 0 {
		t.Errorf("Expected 0 documents (already known), got %d", len(docs))
	}
}

func TestFileSystemSource_Reset(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	file1 := filepath.Join(tmpDir, "doc1.txt")
	os.WriteFile(file1, []byte("content"), 0644)

	source := NewFileSystemSource(FileSystemSourceConfig{
		Name:     "test-fs",
		Path:     tmpDir,
		Patterns: []string{"*.txt"},
	})

	ctx := context.Background()

	// First check
	docs, _ := source.Check(ctx, time.Time{})
	if len(docs) != 1 {
		t.Errorf("Expected 1 document, got %d", len(docs))
	}

	// Second check - file is known
	docs, _ = source.Check(ctx, time.Time{})
	if len(docs) != 0 {
		t.Errorf("Expected 0 documents, got %d", len(docs))
	}

	// Reset and check again
	source.Reset()
	docs, _ = source.Check(ctx, time.Time{})
	if len(docs) != 1 {
		t.Errorf("Expected 1 document after reset, got %d", len(docs))
	}
}

func TestWatcherDocumentType_String(t *testing.T) {
	tests := []struct {
		docType  WatcherDocumentType
		expected string
	}{
		{DocTypeMeetingMinutes, "meeting-minutes"},
		{DocTypeResolution, "resolution"},
		{DocTypeWorkingPaper, "working-paper"},
		{DocTypeRegulation, "regulation"},
		{DocTypeUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.docType.String(); got != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestParseWatcherDocumentType(t *testing.T) {
	tests := []struct {
		input    string
		expected WatcherDocumentType
	}{
		{"meeting-minutes", DocTypeMeetingMinutes},
		{"meeting_minutes", DocTypeMeetingMinutes},
		{"meetingminutes", DocTypeMeetingMinutes},
		{"resolution", DocTypeResolution},
		{"resolutions", DocTypeResolution},
		{"working-paper", DocTypeWorkingPaper},
		{"regulation", DocTypeRegulation},
		{"unknown", DocTypeUnknown},
		{"invalid", DocTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ParseWatcherDocumentType(tt.input); got != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestDefaultWatcherConfig(t *testing.T) {
	config := DefaultWatcherConfig()

	if config.DefaultInterval != 1*time.Hour {
		t.Errorf("Expected 1 hour interval, got %v", config.DefaultInterval)
	}
	if !config.AutoIngest {
		t.Error("Expected AutoIngest to be true")
	}
	if config.MaxConcurrent != 5 {
		t.Errorf("Expected MaxConcurrent 5, got %d", config.MaxConcurrent)
	}
	if config.RetryAttempts != 3 {
		t.Errorf("Expected RetryAttempts 3, got %d", config.RetryAttempts)
	}
}

func TestWatcherManager_SetGetConfig(t *testing.T) {
	manager := NewWatcherManager(nil, "https://example.org/")

	config := WatcherConfig{
		DefaultInterval: 30 * time.Minute,
		AutoIngest:      false,
		MaxConcurrent:   10,
		RetryAttempts:   5,
		RetryDelay:      10 * time.Second,
	}

	manager.SetConfig(config)
	got := manager.GetConfig()

	if got.DefaultInterval != config.DefaultInterval {
		t.Errorf("Expected interval %v, got %v", config.DefaultInterval, got.DefaultInterval)
	}
	if got.AutoIngest != config.AutoIngest {
		t.Errorf("Expected AutoIngest %v, got %v", config.AutoIngest, got.AutoIngest)
	}
	if got.MaxConcurrent != config.MaxConcurrent {
		t.Errorf("Expected MaxConcurrent %d, got %d", config.MaxConcurrent, got.MaxConcurrent)
	}
}

func TestCreateSourceFromConfig(t *testing.T) {
	tests := []struct {
		name       string
		config     WatcherSourceConfig
		expectErr  bool
		expectType string
	}{
		{
			name: "filesystem source",
			config: WatcherSourceConfig{
				Name:         "fs-source",
				Type:         "filesystem",
				Path:         "/tmp",
				Patterns:     []string{"*.txt"},
				DocumentType: "working-paper",
			},
			expectErr:  false,
			expectType: "*deliberation.FileSystemSource",
		},
		{
			name: "memory source",
			config: WatcherSourceConfig{
				Name:         "mem-source",
				Type:         "memory",
				DocumentType: "resolution",
			},
			expectErr:  false,
			expectType: "*deliberation.MemorySource",
		},
		{
			name: "unknown source type",
			config: WatcherSourceConfig{
				Name: "unknown",
				Type: "rss", // Not implemented yet
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := CreateSourceFromConfig(tt.config)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if source.Name() != tt.config.Name {
				t.Errorf("Expected name %s, got %s", tt.config.Name, source.Name())
			}
		})
	}
}

func TestWatcherManager_OnIngest(t *testing.T) {
	ts := store.NewTripleStore()
	manager := NewWatcherManager(ts, "https://example.org/")

	parser := &MockParser{}
	manager.RegisterParser(DocTypeMeetingMinutes, parser)

	var ingestDoc WatchedDocument
	var ingestTripleCount int
	var ingestErr error

	manager.OnIngest(func(doc WatchedDocument, tripleCount int, err error) {
		ingestDoc = doc
		ingestTripleCount = tripleCount
		ingestErr = err
	})

	source := NewMemorySource("test", DocTypeMeetingMinutes)
	doc := WatchedDocument{
		ID:          "doc1",
		Title:       "Test Document",
		PublishedAt: time.Now(),
		Type:        DocTypeMeetingMinutes,
	}
	source.AddDocument(doc, []byte("test content"))
	manager.AddSource(source)

	// Enable auto-ingest
	config := manager.GetConfig()
	config.AutoIngest = true
	manager.SetConfig(config)

	ctx := context.Background()
	manager.handleNewDocument(ctx, source, doc)

	// Wait a bit for async processing
	time.Sleep(100 * time.Millisecond)

	if ingestDoc.ID != "doc1" {
		t.Errorf("Expected doc1, got %s", ingestDoc.ID)
	}
	if ingestErr != nil {
		t.Errorf("Unexpected ingest error: %v", ingestErr)
	}
	if ingestTripleCount == 0 {
		t.Error("Expected triples to be added")
	}
}

func TestWatchedDocument_Fields(t *testing.T) {
	now := time.Now()
	doc := WatchedDocument{
		ID:          "doc-123",
		Title:       "Test Document",
		URL:         "https://example.org/doc.pdf",
		Path:        "/path/to/doc.pdf",
		PublishedAt: now,
		Type:        DocTypeMeetingMinutes,
		Source:      "test-source",
		Size:        1024,
		Checksum:    "abc123",
		Metadata: map[string]string{
			"author": "Test Author",
		},
	}

	if doc.ID != "doc-123" {
		t.Error("ID mismatch")
	}
	if doc.Title != "Test Document" {
		t.Error("Title mismatch")
	}
	if doc.URL != "https://example.org/doc.pdf" {
		t.Error("URL mismatch")
	}
	if doc.Path != "/path/to/doc.pdf" {
		t.Error("Path mismatch")
	}
	if !doc.PublishedAt.Equal(now) {
		t.Error("PublishedAt mismatch")
	}
	if doc.Type != DocTypeMeetingMinutes {
		t.Error("Type mismatch")
	}
	if doc.Source != "test-source" {
		t.Error("Source mismatch")
	}
	if doc.Size != 1024 {
		t.Error("Size mismatch")
	}
	if doc.Checksum != "abc123" {
		t.Error("Checksum mismatch")
	}
	if doc.Metadata["author"] != "Test Author" {
		t.Error("Metadata mismatch")
	}
}

func TestFileSystemSource_MatchesPatterns(t *testing.T) {
	source := NewFileSystemSource(FileSystemSourceConfig{
		Name:     "test",
		Path:     "/tmp",
		Patterns: []string{"*.txt", "*.pdf", "report-*.docx"},
	})

	tests := []struct {
		name     string
		expected bool
	}{
		{"document.txt", true},
		{"file.pdf", true},
		{"report-2024.docx", true},
		{"other.log", false},
		{"report.docx", false}, // doesn't match "report-*.docx"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := source.matchesPatterns(tt.name); got != tt.expected {
				t.Errorf("matchesPatterns(%s) = %v, expected %v", tt.name, got, tt.expected)
			}
		})
	}
}
