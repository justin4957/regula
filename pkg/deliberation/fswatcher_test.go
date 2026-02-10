package deliberation

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewEnhancedFileSystemSource(t *testing.T) {
	source := NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name:         "test-source",
		Path:         "/tmp/test",
		Patterns:     []string{"*.pdf", "*.docx"},
		DocumentType: DocTypeWorkingPaper,
		Recursive:    true,
	})

	if source == nil {
		t.Fatal("Expected non-nil source")
	}
	if source.Name() != "test-source" {
		t.Errorf("Expected name 'test-source', got %s", source.Name())
	}
	if len(source.paths) != 1 {
		t.Errorf("Expected 1 path, got %d", len(source.paths))
	}
}

func TestEnhancedFileSystemSource_MultiplePaths(t *testing.T) {
	source := NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name:  "multi-path",
		Paths: []string{"/tmp/path1", "/tmp/path2"},
	})

	if len(source.paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(source.paths))
	}
}

func TestEnhancedFileSystemSource_Check(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fswatcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	file1 := filepath.Join(tmpDir, "doc1.pdf")
	file2 := filepath.Join(tmpDir, "doc2.docx")
	file3 := filepath.Join(tmpDir, "other.log")

	os.WriteFile(file1, []byte("pdf content"), 0644)
	os.WriteFile(file2, []byte("docx content"), 0644)
	os.WriteFile(file3, []byte("log content"), 0644)

	source := NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name:         "test",
		Path:         tmpDir,
		Patterns:     []string{"*.pdf", "*.docx"},
		DocumentType: DocTypeWorkingPaper,
	})

	ctx := context.Background()
	docs, err := source.Check(ctx, time.Time{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(docs))
	}

	// Check that files are marked as processed
	if !source.IsProcessed(file1) {
		t.Error("Expected file1 to be processed")
	}
	if !source.IsProcessed(file2) {
		t.Error("Expected file2 to be processed")
	}
}

func TestEnhancedFileSystemSource_Recursive(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fswatcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	subDir := filepath.Join(tmpDir, "subdir")
	os.MkdirAll(subDir, 0755)

	// Create files
	file1 := filepath.Join(tmpDir, "doc1.txt")
	file2 := filepath.Join(subDir, "doc2.txt")

	os.WriteFile(file1, []byte("content1"), 0644)
	os.WriteFile(file2, []byte("content2"), 0644)

	// Non-recursive
	source := NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name:      "test",
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
	source = NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name:      "test",
		Path:      tmpDir,
		Patterns:  []string{"*.txt"},
		Recursive: true,
	})

	docs, _ = source.Check(ctx, time.Time{})
	if len(docs) != 2 {
		t.Errorf("Expected 2 documents (recursive), got %d", len(docs))
	}
}

func TestEnhancedFileSystemSource_StateOperations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fswatcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	statePath := filepath.Join(tmpDir, "state.json")

	// Create source and mark a file as processed
	file1 := filepath.Join(tmpDir, "doc1.txt")
	os.WriteFile(file1, []byte("content"), 0644)

	source := NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name:     "test",
		Path:     tmpDir,
		Patterns: []string{"*.txt"},
	})

	err = source.MarkProcessed(file1)
	if err != nil {
		t.Fatalf("MarkProcessed failed: %v", err)
	}

	if !source.IsProcessed(file1) {
		t.Error("Expected file to be marked as processed")
	}

	// Save state
	err = source.SaveState(statePath)
	if err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Create new source and load state
	source2 := NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name:     "test",
		Path:     tmpDir,
		Patterns: []string{"*.txt"},
	})

	err = source2.LoadState(statePath)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if !source2.IsProcessed(file1) {
		t.Error("Expected file to be processed after loading state")
	}
}

func TestEnhancedFileSystemSource_DetectModifiedFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fswatcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	file1 := filepath.Join(tmpDir, "doc1.txt")
	os.WriteFile(file1, []byte("original content"), 0644)

	source := NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name:     "test",
		Path:     tmpDir,
		Patterns: []string{"*.txt"},
	})

	ctx := context.Background()

	// First check
	docs, _ := source.Check(ctx, time.Time{})
	if len(docs) != 1 {
		t.Errorf("Expected 1 document on first check, got %d", len(docs))
	}

	// Second check - no changes
	docs, _ = source.Check(ctx, time.Time{})
	if len(docs) != 0 {
		t.Errorf("Expected 0 documents (no changes), got %d", len(docs))
	}

	// Modify file
	time.Sleep(10 * time.Millisecond) // Ensure mod time changes
	os.WriteFile(file1, []byte("modified content"), 0644)

	// Third check - should detect modification
	docs, _ = source.Check(ctx, time.Time{})
	if len(docs) != 1 {
		t.Errorf("Expected 1 document (modified), got %d", len(docs))
	}
}

func TestEnhancedFileSystemSource_ComputeHash(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fswatcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	file1 := filepath.Join(tmpDir, "doc1.txt")
	os.WriteFile(file1, []byte("test content"), 0644)

	source := NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name:        "test",
		Path:        tmpDir,
		Patterns:    []string{"*.txt"},
		ComputeHash: true,
	})

	ctx := context.Background()
	docs, _ := source.Check(ctx, time.Time{})

	if len(docs) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(docs))
	}

	if docs[0].Checksum == "" {
		t.Error("Expected checksum to be computed")
	}

	// Verify hash format (should be hex string)
	if len(docs[0].Checksum) != 64 { // SHA256 hex length
		t.Errorf("Expected 64 character hash, got %d", len(docs[0].Checksum))
	}
}

func TestEnhancedFileSystemSource_DetectDeletedFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fswatcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	file1 := filepath.Join(tmpDir, "doc1.txt")
	file2 := filepath.Join(tmpDir, "doc2.txt")
	os.WriteFile(file1, []byte("content1"), 0644)
	os.WriteFile(file2, []byte("content2"), 0644)

	source := NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name:     "test",
		Path:     tmpDir,
		Patterns: []string{"*.txt"},
	})

	ctx := context.Background()
	source.Check(ctx, time.Time{})

	// Delete one file
	os.Remove(file1)

	// Detect deleted
	deleted := source.DetectDeletedFiles()
	if len(deleted) != 1 {
		t.Errorf("Expected 1 deleted file, got %d", len(deleted))
	}

	// Cleanup deleted
	cleaned := source.CleanupDeletedFiles()
	if len(cleaned) != 1 {
		t.Errorf("Expected 1 cleaned file, got %d", len(cleaned))
	}

	if source.IsProcessed(file1) {
		t.Error("Expected file1 to be removed from state")
	}
}

func TestEnhancedFileSystemSource_Debouncing(t *testing.T) {
	source := NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name:     "test",
		Debounce: 100 * time.Millisecond,
	})

	// Notify changes
	source.NotifyChange("/path/to/file1.txt")
	source.NotifyChange("/path/to/file2.txt")

	pending := source.GetPendingChanges()
	if len(pending) != 2 {
		t.Errorf("Expected 2 pending changes, got %d", len(pending))
	}

	// Changes should still be pending (within debounce window)
	ctx := context.Background()
	docs, _ := source.DebouncedCheck(ctx)
	if len(docs) != 0 {
		t.Errorf("Expected 0 documents (still debouncing), got %d", len(docs))
	}
}

func TestEnhancedFileSystemSource_Reset(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fswatcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	file1 := filepath.Join(tmpDir, "doc1.txt")
	os.WriteFile(file1, []byte("content"), 0644)

	source := NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name:     "test",
		Path:     tmpDir,
		Patterns: []string{"*.txt"},
	})

	ctx := context.Background()
	source.Check(ctx, time.Time{})

	if source.ProcessedCount() != 1 {
		t.Errorf("Expected 1 processed file, got %d", source.ProcessedCount())
	}

	source.Reset()

	if source.ProcessedCount() != 0 {
		t.Errorf("Expected 0 processed files after reset, got %d", source.ProcessedCount())
	}
}

func TestEnhancedFileSystemSource_ScanOnce(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fswatcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	file1 := filepath.Join(tmpDir, "doc1.txt")
	file2 := filepath.Join(tmpDir, "doc2.txt")
	os.WriteFile(file1, []byte("content1"), 0644)
	os.WriteFile(file2, []byte("content2"), 0644)

	source := NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name:     "test",
		Path:     tmpDir,
		Patterns: []string{"*.txt"},
	})

	ctx := context.Background()

	// First scan
	docs, _ := source.ScanOnce(ctx)
	if len(docs) != 2 {
		t.Errorf("Expected 2 documents on first scan, got %d", len(docs))
	}

	// Scan again - should return all files again (ScanOnce resets)
	docs, _ = source.ScanOnce(ctx)
	if len(docs) != 2 {
		t.Errorf("Expected 2 documents on second scan (reset), got %d", len(docs))
	}
}

func TestEnhancedFileSystemSource_Stats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fswatcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	file1 := filepath.Join(tmpDir, "doc1.txt")
	os.WriteFile(file1, []byte("content"), 0644)

	source := NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name:     "test",
		Path:     tmpDir,
		Patterns: []string{"*.txt"},
	})

	ctx := context.Background()
	source.Check(ctx, time.Time{})
	source.NotifyChange("/pending/file.txt")

	stats := source.Stats()

	if stats.ProcessedFiles != 1 {
		t.Errorf("Expected 1 processed file, got %d", stats.ProcessedFiles)
	}
	if stats.PendingChanges != 1 {
		t.Errorf("Expected 1 pending change, got %d", stats.PendingChanges)
	}
	if stats.WatchedPaths != 1 {
		t.Errorf("Expected 1 watched path, got %d", stats.WatchedPaths)
	}
}

func TestEnhancedFileSystemSource_FilterByType(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fswatcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files with different extensions
	file1 := filepath.Join(tmpDir, "doc1.pdf")
	file2 := filepath.Join(tmpDir, "doc2.txt")
	os.WriteFile(file1, []byte("pdf"), 0644)
	os.WriteFile(file2, []byte("txt"), 0644)

	source := NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name:     "test",
		Path:     tmpDir,
		Patterns: []string{"*"},
	})

	ctx := context.Background()
	source.Check(ctx, time.Time{})

	// Filter by type
	workingPapers := source.FilterByType(DocTypeWorkingPaper)
	if len(workingPapers) != 1 {
		t.Errorf("Expected 1 working paper, got %d", len(workingPapers))
	}

	meetingMinutes := source.FilterByType(DocTypeMeetingMinutes)
	if len(meetingMinutes) != 1 {
		t.Errorf("Expected 1 meeting minutes, got %d", len(meetingMinutes))
	}
}

func TestEnhancedFileSystemSource_GetProcessedAfter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fswatcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	file1 := filepath.Join(tmpDir, "doc1.txt")
	os.WriteFile(file1, []byte("content"), 0644)

	source := NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name:     "test",
		Path:     tmpDir,
		Patterns: []string{"*.txt"},
	})

	beforeCheck := time.Now().Add(-1 * time.Second)

	ctx := context.Background()
	source.Check(ctx, time.Time{})

	afterCheck := time.Now().Add(1 * time.Second)

	// Should find files processed after beforeCheck
	files := source.GetProcessedAfter(beforeCheck)
	if len(files) != 1 {
		t.Errorf("Expected 1 file after beforeCheck, got %d", len(files))
	}

	// Should find no files processed after afterCheck
	files = source.GetProcessedAfter(afterCheck)
	if len(files) != 0 {
		t.Errorf("Expected 0 files after afterCheck, got %d", len(files))
	}
}

func TestEnhancedFileSystemSource_UnmarkProcessed(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fswatcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	file1 := filepath.Join(tmpDir, "doc1.txt")
	os.WriteFile(file1, []byte("content"), 0644)

	source := NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name:     "test",
		Path:     tmpDir,
		Patterns: []string{"*.txt"},
	})

	source.MarkProcessed(file1)
	if !source.IsProcessed(file1) {
		t.Error("Expected file to be processed")
	}

	source.UnmarkProcessed(file1)
	if source.IsProcessed(file1) {
		t.Error("Expected file to not be processed after unmark")
	}
}

func TestEnhancedFileSystemSource_AutoSave(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fswatcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	file1 := filepath.Join(tmpDir, "doc1.txt")
	statePath := filepath.Join(tmpDir, "state.json")
	os.WriteFile(file1, []byte("content"), 0644)

	source := NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name:      "test",
		Path:      tmpDir,
		Patterns:  []string{"*.txt"},
		StatePath: statePath,
		AutoSave:  true,
	})

	ctx := context.Background()
	source.Check(ctx, time.Time{})

	// State file should exist
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("Expected state file to be created")
	}
}

func TestDetectDocumentTypeFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected WatcherDocumentType
	}{
		{"/path/to/doc.pdf", DocTypeWorkingPaper},
		{"/path/to/doc.docx", DocTypeWorkingPaper},
		{"/path/to/doc.doc", DocTypeWorkingPaper},
		{"/path/to/minutes.txt", DocTypeMeetingMinutes},
		{"/path/to/notes.md", DocTypeMeetingMinutes},
		{"/path/to/resolution.html", DocTypeResolution},
		{"/path/to/page.htm", DocTypeResolution},
		{"/path/to/regulation.xml", DocTypeRegulation},
		{"/path/to/file.unknown", DocTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := DetectDocumentTypeFromPath(tt.path)
			if got != tt.expected {
				t.Errorf("DetectDocumentTypeFromPath(%s) = %v, expected %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestWatchEventType_String(t *testing.T) {
	tests := []struct {
		eventType WatchEventType
		expected  string
	}{
		{EventCreated, "created"},
		{EventModified, "modified"},
		{EventDeleted, "deleted"},
		{EventRenamed, "renamed"},
		{WatchEventType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.eventType.String(); got != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestNewWatcherState(t *testing.T) {
	state := NewWatcherState()

	if state == nil {
		t.Fatal("Expected non-nil state")
	}
	if state.ProcessedFiles == nil {
		t.Error("Expected ProcessedFiles map to be initialized")
	}
	if state.Version != 1 {
		t.Errorf("Expected version 1, got %d", state.Version)
	}
}

func TestFileState_Fields(t *testing.T) {
	now := time.Now()
	state := FileState{
		Path:         "/path/to/file.txt",
		ModTime:      now,
		Hash:         "abc123",
		Size:         1024,
		ProcessedAt:  now,
		DocumentType: DocTypeMeetingMinutes,
	}

	if state.Path != "/path/to/file.txt" {
		t.Error("Path mismatch")
	}
	if !state.ModTime.Equal(now) {
		t.Error("ModTime mismatch")
	}
	if state.Hash != "abc123" {
		t.Error("Hash mismatch")
	}
	if state.Size != 1024 {
		t.Error("Size mismatch")
	}
	if !state.ProcessedAt.Equal(now) {
		t.Error("ProcessedAt mismatch")
	}
	if state.DocumentType != DocTypeMeetingMinutes {
		t.Error("DocumentType mismatch")
	}
}

func TestEnhancedFileSystemSource_SetCallbacks(t *testing.T) {
	source := NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name: "test",
	})

	var onChangeCalled bool
	var onDeleteCalled bool

	source.SetOnChange(func(doc WatchedDocument) {
		onChangeCalled = true
	})

	source.SetOnDelete(func(path string) {
		onDeleteCalled = true
	})

	// Callbacks are set but we can't easily trigger them without fsnotify
	// Just verify they can be set without error
	_ = onChangeCalled
	_ = onDeleteCalled
}

func TestEnhancedFileSystemSource_LoadNonexistentState(t *testing.T) {
	source := NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name: "test",
	})

	err := source.LoadState("/nonexistent/path/state.json")
	if err != nil {
		t.Errorf("Expected nil error for nonexistent state file, got: %v", err)
	}
}

func TestEnhancedFileSystemSource_GetState(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fswatcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	file1 := filepath.Join(tmpDir, "doc1.txt")
	os.WriteFile(file1, []byte("content"), 0644)

	source := NewEnhancedFileSystemSource(EnhancedFileSystemConfig{
		Name:     "test",
		Path:     tmpDir,
		Patterns: []string{"*.txt"},
	})

	ctx := context.Background()
	source.Check(ctx, time.Time{})

	state := source.GetState()

	if len(state.ProcessedFiles) != 1 {
		t.Errorf("Expected 1 processed file in state, got %d", len(state.ProcessedFiles))
	}

	// Verify it's a copy (modifying returned state shouldn't affect source)
	state.ProcessedFiles["new_file"] = FileState{}
	if source.ProcessedCount() != 1 {
		t.Error("State should be a copy, not affect source")
	}
}
