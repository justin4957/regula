// Package deliberation provides enhanced file system watching capabilities
// for local document directories with debouncing, state persistence, and
// real-time change detection.
package deliberation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileState represents the state of a processed file.
type FileState struct {
	// Path is the absolute path to the file.
	Path string `json:"path"`

	// ModTime is the last modification time.
	ModTime time.Time `json:"mod_time"`

	// Hash is the SHA256 hash of the file content.
	Hash string `json:"hash"`

	// Size is the file size in bytes.
	Size int64 `json:"size"`

	// ProcessedAt is when the file was processed.
	ProcessedAt time.Time `json:"processed_at"`

	// DocumentType is the detected document type.
	DocumentType WatcherDocumentType `json:"document_type"`
}

// WatcherState tracks the state of processed files for persistence.
type WatcherState struct {
	// ProcessedFiles maps file paths to their state.
	ProcessedFiles map[string]FileState `json:"processed_files"`

	// LastCheck is the last check timestamp.
	LastCheck time.Time `json:"last_check"`

	// Version is the state format version.
	Version int `json:"version"`
}

// NewWatcherState creates a new watcher state.
func NewWatcherState() *WatcherState {
	return &WatcherState{
		ProcessedFiles: make(map[string]FileState),
		Version:        1,
	}
}

// EnhancedFileSystemSource extends FileSystemSource with additional features.
type EnhancedFileSystemSource struct {
	// Embedded base source
	name      string
	paths     []string
	patterns  []string
	docType   WatcherDocumentType
	recursive bool

	// State management
	state     *WatcherState
	statePath string

	// Debouncing
	debounce       time.Duration
	pendingChanges map[string]time.Time
	pendingMu      sync.Mutex

	// Change detection
	onChange   func(WatchedDocument)
	onDelete   func(string)
	watching   bool
	watchStopCh chan struct{}

	// Configuration
	computeHash bool
	autoSave    bool

	mu sync.RWMutex
}

// EnhancedFileSystemConfig holds configuration for an enhanced filesystem source.
type EnhancedFileSystemConfig struct {
	// Name is the source identifier.
	Name string `json:"name" yaml:"name"`

	// Paths are the directories to watch (supports multiple).
	Paths []string `json:"paths" yaml:"paths"`

	// Path is a single directory to watch (for backwards compatibility).
	Path string `json:"path" yaml:"path"`

	// Patterns are glob patterns for file matching.
	Patterns []string `json:"patterns" yaml:"patterns"`

	// DocumentType is the type of documents in these directories.
	DocumentType WatcherDocumentType `json:"document_type" yaml:"document_type"`

	// Recursive enables recursive directory scanning.
	Recursive bool `json:"recursive" yaml:"recursive"`

	// StatePath is the path to save/load state.
	StatePath string `json:"state_path" yaml:"state_path"`

	// Debounce is the debounce duration for file changes.
	Debounce time.Duration `json:"debounce" yaml:"debounce"`

	// ComputeHash enables content hashing for change detection.
	ComputeHash bool `json:"compute_hash" yaml:"compute_hash"`

	// AutoSave enables automatic state saving after each check.
	AutoSave bool `json:"auto_save" yaml:"auto_save"`
}

// NewEnhancedFileSystemSource creates a new enhanced filesystem source.
func NewEnhancedFileSystemSource(config EnhancedFileSystemConfig) *EnhancedFileSystemSource {
	paths := config.Paths
	if len(paths) == 0 && config.Path != "" {
		paths = []string{config.Path}
	}

	if len(config.Patterns) == 0 {
		config.Patterns = []string{"*"}
	}

	if config.Debounce == 0 {
		config.Debounce = 500 * time.Millisecond
	}

	return &EnhancedFileSystemSource{
		name:           config.Name,
		paths:          paths,
		patterns:       config.Patterns,
		docType:        config.DocumentType,
		recursive:      config.Recursive,
		state:          NewWatcherState(),
		statePath:      config.StatePath,
		debounce:       config.Debounce,
		pendingChanges: make(map[string]time.Time),
		computeHash:    config.ComputeHash,
		autoSave:       config.AutoSave,
	}
}

// Name returns the source identifier.
func (s *EnhancedFileSystemSource) Name() string {
	return s.name
}

// Check returns new or modified files since the given time.
func (s *EnhancedFileSystemSource) Check(ctx context.Context, since time.Time) ([]WatchedDocument, error) {
	var docs []WatchedDocument

	for _, path := range s.paths {
		pathDocs, err := s.checkPath(ctx, path, since)
		if err != nil {
			continue // Log but continue with other paths
		}
		docs = append(docs, pathDocs...)
	}

	// Update last check time
	s.mu.Lock()
	s.state.LastCheck = time.Now()
	s.mu.Unlock()

	// Auto-save state if enabled
	if s.autoSave && s.statePath != "" {
		_ = s.SaveState(s.statePath)
	}

	return docs, nil
}

// checkPath checks a single path for new/modified files.
func (s *EnhancedFileSystemSource) checkPath(ctx context.Context, basePath string, since time.Time) ([]WatchedDocument, error) {
	var docs []WatchedDocument

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if info.IsDir() {
			if !s.recursive && path != basePath {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file matches patterns
		if !s.matchesPatterns(filepath.Base(path)) {
			return nil
		}

		// Check if file is new or modified
		doc, isNew := s.checkFile(path, info, since)
		if isNew {
			docs = append(docs, doc)
		}

		return nil
	}

	err := filepath.Walk(basePath, walkFn)
	return docs, err
}

// checkFile checks if a file is new or modified.
func (s *EnhancedFileSystemSource) checkFile(path string, info os.FileInfo, since time.Time) (WatchedDocument, bool) {
	modTime := info.ModTime()

	s.mu.RLock()
	prevState, known := s.state.ProcessedFiles[path]
	s.mu.RUnlock()

	// Determine if file should be returned
	isNew := false

	if !known {
		// New file
		isNew = since.IsZero() || modTime.After(since)
	} else {
		// Known file - check if modified
		if modTime.After(prevState.ModTime) {
			isNew = true
		} else if s.computeHash && prevState.Hash != "" {
			// Check hash if enabled
			hash, _ := s.computeFileHash(path)
			if hash != prevState.Hash {
				isNew = true
			}
		}
	}

	if !isNew {
		return WatchedDocument{}, false
	}

	// Create document
	doc := WatchedDocument{
		ID:          path,
		Title:       filepath.Base(path),
		Path:        path,
		URL:         "file://" + path,
		PublishedAt: modTime,
		Type:        s.detectDocumentType(path),
		Source:      s.name,
		Size:        info.Size(),
		Metadata:    make(map[string]string),
	}
	doc.Metadata["extension"] = filepath.Ext(path)

	// Compute hash if enabled
	if s.computeHash {
		if hash, err := s.computeFileHash(path); err == nil {
			doc.Checksum = hash
		}
	}

	// Update state
	s.mu.Lock()
	s.state.ProcessedFiles[path] = FileState{
		Path:         path,
		ModTime:      modTime,
		Hash:         doc.Checksum,
		Size:         info.Size(),
		ProcessedAt:  time.Now(),
		DocumentType: doc.Type,
	}
	s.mu.Unlock()

	return doc, true
}

// Fetch reads the file content.
func (s *EnhancedFileSystemSource) Fetch(ctx context.Context, doc WatchedDocument) ([]byte, error) {
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
func (s *EnhancedFileSystemSource) Type(doc WatchedDocument) WatcherDocumentType {
	if doc.Type != "" && doc.Type != DocTypeUnknown {
		return doc.Type
	}
	return s.detectDocumentType(doc.Path)
}

// detectDocumentType determines document type from file extension.
func (s *EnhancedFileSystemSource) detectDocumentType(path string) WatcherDocumentType {
	if s.docType != "" && s.docType != DocTypeUnknown {
		return s.docType
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".pdf", ".docx", ".doc":
		return DocTypeWorkingPaper
	case ".txt", ".md":
		return DocTypeMeetingMinutes
	case ".html", ".htm":
		return DocTypeResolution
	default:
		return DocTypeUnknown
	}
}

// matchesPatterns checks if a filename matches any pattern.
func (s *EnhancedFileSystemSource) matchesPatterns(name string) bool {
	for _, pattern := range s.patterns {
		matched, _ := filepath.Match(pattern, name)
		if matched {
			return true
		}
	}
	return false
}

// computeFileHash computes the SHA256 hash of a file.
func (s *EnhancedFileSystemSource) computeFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// SaveState saves the watcher state to a file.
func (s *EnhancedFileSystemSource) SaveState(path string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	// Write to temp file first, then rename (atomic)
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename file: %w", err)
	}

	return nil
}

// LoadState loads the watcher state from a file.
func (s *EnhancedFileSystemSource) LoadState(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No state file, start fresh
		}
		return fmt.Errorf("read state file: %w", err)
	}

	var state WatcherState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("unmarshal state: %w", err)
	}

	s.mu.Lock()
	s.state = &state
	s.mu.Unlock()

	return nil
}

// GetState returns a copy of the current state.
func (s *EnhancedFileSystemSource) GetState() WatcherState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state := WatcherState{
		ProcessedFiles: make(map[string]FileState),
		LastCheck:      s.state.LastCheck,
		Version:        s.state.Version,
	}
	for k, v := range s.state.ProcessedFiles {
		state.ProcessedFiles[k] = v
	}
	return state
}

// Reset clears the state and starts fresh.
func (s *EnhancedFileSystemSource) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = NewWatcherState()
}

// IsProcessed checks if a file has been processed.
func (s *EnhancedFileSystemSource) IsProcessed(path string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.state.ProcessedFiles[path]
	return exists
}

// MarkProcessed marks a file as processed.
func (s *EnhancedFileSystemSource) MarkProcessed(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var hash string
	if s.computeHash {
		hash, _ = s.computeFileHash(path)
	}

	s.state.ProcessedFiles[path] = FileState{
		Path:         path,
		ModTime:      info.ModTime(),
		Hash:         hash,
		Size:         info.Size(),
		ProcessedAt:  time.Now(),
		DocumentType: s.detectDocumentType(path),
	}

	return nil
}

// UnmarkProcessed removes a file from the processed state.
func (s *EnhancedFileSystemSource) UnmarkProcessed(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.state.ProcessedFiles, path)
}

// SetOnChange sets the callback for file changes.
func (s *EnhancedFileSystemSource) SetOnChange(callback func(WatchedDocument)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onChange = callback
}

// SetOnDelete sets the callback for file deletions.
func (s *EnhancedFileSystemSource) SetOnDelete(callback func(string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onDelete = callback
}

// ProcessedCount returns the number of processed files.
func (s *EnhancedFileSystemSource) ProcessedCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.state.ProcessedFiles)
}

// DetectDeletedFiles detects files that have been deleted since last check.
func (s *EnhancedFileSystemSource) DetectDeletedFiles() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var deleted []string
	for path := range s.state.ProcessedFiles {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			deleted = append(deleted, path)
		}
	}
	return deleted
}

// CleanupDeletedFiles removes deleted files from the state.
func (s *EnhancedFileSystemSource) CleanupDeletedFiles() []string {
	deleted := s.DetectDeletedFiles()

	s.mu.Lock()
	for _, path := range deleted {
		delete(s.state.ProcessedFiles, path)
	}
	s.mu.Unlock()

	return deleted
}

// DebouncedCheck performs a check with debouncing for pending changes.
func (s *EnhancedFileSystemSource) DebouncedCheck(ctx context.Context) ([]WatchedDocument, error) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()

	var docs []WatchedDocument
	now := time.Now()
	threshold := now.Add(-s.debounce)

	for path, changeTime := range s.pendingChanges {
		// Skip if still within debounce window
		if changeTime.After(threshold) {
			continue
		}

		// Check if file exists and get info
		info, err := os.Stat(path)
		if err != nil {
			// File was deleted
			delete(s.pendingChanges, path)
			continue
		}

		// Create document
		doc, isNew := s.checkFile(path, info, time.Time{})
		if isNew {
			docs = append(docs, doc)
		}

		delete(s.pendingChanges, path)
	}

	return docs, nil
}

// NotifyChange notifies the source of a file change (for debouncing).
func (s *EnhancedFileSystemSource) NotifyChange(path string) {
	s.pendingMu.Lock()
	s.pendingChanges[path] = time.Now()
	s.pendingMu.Unlock()
}

// GetPendingChanges returns paths with pending changes.
func (s *EnhancedFileSystemSource) GetPendingChanges() []string {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()

	paths := make([]string, 0, len(s.pendingChanges))
	for path := range s.pendingChanges {
		paths = append(paths, path)
	}
	return paths
}

// ScanOnce performs a one-time scan of all paths.
func (s *EnhancedFileSystemSource) ScanOnce(ctx context.Context) ([]WatchedDocument, error) {
	// Reset state for fresh scan
	s.Reset()
	return s.Check(ctx, time.Time{})
}

// FileTypeMap maps file extensions to document types.
var FileTypeMap = map[string]WatcherDocumentType{
	".pdf":  DocTypeWorkingPaper,
	".docx": DocTypeWorkingPaper,
	".doc":  DocTypeWorkingPaper,
	".txt":  DocTypeMeetingMinutes,
	".md":   DocTypeMeetingMinutes,
	".html": DocTypeResolution,
	".htm":  DocTypeResolution,
	".xml":  DocTypeRegulation,
	".json": DocTypeUnknown,
}

// DetectDocumentTypeFromPath determines document type from a file path.
func DetectDocumentTypeFromPath(path string) WatcherDocumentType {
	ext := strings.ToLower(filepath.Ext(path))
	if docType, ok := FileTypeMap[ext]; ok {
		return docType
	}
	return DocTypeUnknown
}

// WatchEvent represents a file system event.
type WatchEvent struct {
	// Path is the file path.
	Path string `json:"path"`

	// Type is the event type.
	Type WatchEventType `json:"type"`

	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`
}

// WatchEventType classifies file system events.
type WatchEventType int

const (
	// EventCreated indicates a new file.
	EventCreated WatchEventType = iota
	// EventModified indicates a modified file.
	EventModified
	// EventDeleted indicates a deleted file.
	EventDeleted
	// EventRenamed indicates a renamed file.
	EventRenamed
)

// String returns a string representation of the event type.
func (e WatchEventType) String() string {
	switch e {
	case EventCreated:
		return "created"
	case EventModified:
		return "modified"
	case EventDeleted:
		return "deleted"
	case EventRenamed:
		return "renamed"
	default:
		return "unknown"
	}
}

// WatcherStats holds statistics about the watcher.
type FileWatcherStats struct {
	// TotalFiles is the number of files being tracked.
	TotalFiles int `json:"total_files"`

	// ProcessedFiles is the number of files processed.
	ProcessedFiles int `json:"processed_files"`

	// PendingChanges is the number of pending debounced changes.
	PendingChanges int `json:"pending_changes"`

	// LastCheck is the last check timestamp.
	LastCheck time.Time `json:"last_check"`

	// WatchedPaths is the number of watched paths.
	WatchedPaths int `json:"watched_paths"`
}

// Stats returns statistics about the source.
func (s *EnhancedFileSystemSource) Stats() FileWatcherStats {
	s.mu.RLock()
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	defer s.mu.RUnlock()

	return FileWatcherStats{
		ProcessedFiles: len(s.state.ProcessedFiles),
		PendingChanges: len(s.pendingChanges),
		LastCheck:      s.state.LastCheck,
		WatchedPaths:   len(s.paths),
	}
}

// FilterByType returns only files of a specific document type.
func (s *EnhancedFileSystemSource) FilterByType(docType WatcherDocumentType) []FileState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []FileState
	for _, state := range s.state.ProcessedFiles {
		if state.DocumentType == docType {
			result = append(result, state)
		}
	}
	return result
}

// GetProcessedAfter returns files processed after a given time.
func (s *EnhancedFileSystemSource) GetProcessedAfter(t time.Time) []FileState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []FileState
	for _, state := range s.state.ProcessedFiles {
		if state.ProcessedAt.After(t) {
			result = append(result, state)
		}
	}
	return result
}
