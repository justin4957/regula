package pattern

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/fsnotify.v1"
	"gopkg.in/yaml.v3"
)

// Registry manages a collection of format patterns.
type Registry interface {
	// Register adds a pattern to the registry
	Register(pattern *FormatPattern) error

	// Unregister removes a pattern from the registry
	Unregister(formatID string) error

	// Get returns a pattern by its format ID
	Get(formatID string) (*FormatPattern, bool)

	// List returns all registered patterns
	List() []*FormatPattern

	// ListByJurisdiction returns patterns for a specific jurisdiction
	ListByJurisdiction(jurisdiction string) []*FormatPattern

	// Reload reloads all patterns from the configured directory
	Reload() error

	// Watch starts watching the pattern directory for changes
	Watch() error

	// StopWatch stops watching the pattern directory
	StopWatch()

	// LoadDirectory loads all patterns from a directory
	LoadDirectory(dir string) error

	// LoadFile loads a single pattern file
	LoadFile(path string) error
}

// DefaultRegistry is the default implementation of the pattern Registry.
type DefaultRegistry struct {
	mu       sync.RWMutex
	patterns map[string]*FormatPattern
	dir      string
	watcher  *fsnotify.Watcher
	stopChan chan struct{}
	onChange func(event string, pattern *FormatPattern)
}

// NewRegistry creates a new pattern registry.
func NewRegistry() *DefaultRegistry {
	return &DefaultRegistry{
		patterns: make(map[string]*FormatPattern),
	}
}

// NewRegistryWithDirectory creates a new registry and loads patterns from the directory.
func NewRegistryWithDirectory(dir string) (*DefaultRegistry, error) {
	r := NewRegistry()
	r.dir = dir

	if err := r.LoadDirectory(dir); err != nil {
		return nil, err
	}

	return r, nil
}

// Register adds a pattern to the registry.
func (r *DefaultRegistry) Register(pattern *FormatPattern) error {
	if pattern == nil {
		return fmt.Errorf("pattern cannot be nil")
	}

	// Validate the pattern
	if err := pattern.Validate(); err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	// Compile the pattern
	if !pattern.IsCompiled() {
		if err := pattern.Compile(); err != nil {
			return fmt.Errorf("compiling pattern %q: %w", pattern.FormatID, err)
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate
	if existing, ok := r.patterns[pattern.FormatID]; ok {
		// Allow update if version is different
		if existing.Version == pattern.Version {
			return fmt.Errorf("pattern %q version %s already registered", pattern.FormatID, pattern.Version)
		}
	}

	r.patterns[pattern.FormatID] = pattern
	return nil
}

// Unregister removes a pattern from the registry.
func (r *DefaultRegistry) Unregister(formatID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.patterns[formatID]; !ok {
		return fmt.Errorf("pattern %q not found", formatID)
	}

	delete(r.patterns, formatID)
	return nil
}

// Get returns a pattern by its format ID.
func (r *DefaultRegistry) Get(formatID string) (*FormatPattern, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pattern, ok := r.patterns[formatID]
	return pattern, ok
}

// List returns all registered patterns.
func (r *DefaultRegistry) List() []*FormatPattern {
	r.mu.RLock()
	defer r.mu.RUnlock()

	patterns := make([]*FormatPattern, 0, len(r.patterns))
	for _, p := range r.patterns {
		patterns = append(patterns, p)
	}
	return patterns
}

// ListByJurisdiction returns patterns for a specific jurisdiction.
func (r *DefaultRegistry) ListByJurisdiction(jurisdiction string) []*FormatPattern {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var patterns []*FormatPattern
	jurisdictionLower := strings.ToLower(jurisdiction)
	for _, p := range r.patterns {
		if strings.ToLower(p.Jurisdiction) == jurisdictionLower {
			patterns = append(patterns, p)
		}
	}
	return patterns
}

// Count returns the number of registered patterns.
func (r *DefaultRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.patterns)
}

// LoadDirectory loads all YAML pattern files from a directory.
func (r *DefaultRegistry) LoadDirectory(dir string) error {
	r.dir = dir

	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist, nothing to load
			return nil
		}
		return fmt.Errorf("checking directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	// Find all YAML files
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading directory %s: %w", dir, err)
	}

	var loadErrors []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		path := filepath.Join(dir, name)
		if err := r.LoadFile(path); err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("%s: %v", name, err))
		}
	}

	if len(loadErrors) > 0 {
		return fmt.Errorf("errors loading patterns: %s", strings.Join(loadErrors, "; "))
	}

	return nil
}

// LoadFile loads a single pattern file.
func (r *DefaultRegistry) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	var pattern FormatPattern
	if err := yaml.Unmarshal(data, &pattern); err != nil {
		return fmt.Errorf("parsing YAML: %w", err)
	}

	if err := r.Register(&pattern); err != nil {
		return fmt.Errorf("registering pattern: %w", err)
	}

	return nil
}

// Reload reloads all patterns from the configured directory.
func (r *DefaultRegistry) Reload() error {
	if r.dir == "" {
		return fmt.Errorf("no directory configured for reload")
	}

	// Clear existing patterns
	r.mu.Lock()
	r.patterns = make(map[string]*FormatPattern)
	r.mu.Unlock()

	// Reload from directory
	return r.LoadDirectory(r.dir)
}

// SetOnChange sets a callback function that is called when patterns change.
func (r *DefaultRegistry) SetOnChange(fn func(event string, pattern *FormatPattern)) {
	r.onChange = fn
}

// Watch starts watching the pattern directory for changes.
func (r *DefaultRegistry) Watch() error {
	if r.dir == "" {
		return fmt.Errorf("no directory configured for watching")
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating watcher: %w", err)
	}

	r.watcher = watcher
	r.stopChan = make(chan struct{})

	// Start watching goroutine
	go r.watchLoop()

	// Add directory to watch
	if err := watcher.Add(r.dir); err != nil {
		r.watcher.Close()
		return fmt.Errorf("watching directory %s: %w", r.dir, err)
	}

	return nil
}

// watchLoop handles file system events.
func (r *DefaultRegistry) watchLoop() {
	for {
		select {
		case <-r.stopChan:
			return

		case event, ok := <-r.watcher.Events:
			if !ok {
				return
			}

			// Only process YAML files
			if !strings.HasSuffix(event.Name, ".yaml") && !strings.HasSuffix(event.Name, ".yml") {
				continue
			}

			switch {
			case event.Op&fsnotify.Create == fsnotify.Create:
				r.handleFileChange(event.Name, "create")

			case event.Op&fsnotify.Write == fsnotify.Write:
				r.handleFileChange(event.Name, "modify")

			case event.Op&fsnotify.Remove == fsnotify.Remove:
				r.handleFileRemove(event.Name)

			case event.Op&fsnotify.Rename == fsnotify.Rename:
				r.handleFileRemove(event.Name)
			}

		case err, ok := <-r.watcher.Errors:
			if !ok {
				return
			}
			// Log error but continue watching
			_ = err // In production, this should be logged
		}
	}
}

// handleFileChange handles file creation or modification.
func (r *DefaultRegistry) handleFileChange(path string, eventType string) {
	if err := r.LoadFile(path); err != nil {
		// Log error but continue
		_ = err // In production, this should be logged
		return
	}

	if r.onChange != nil {
		pattern, ok := r.getPatternByFile(path)
		if ok {
			r.onChange(eventType, pattern)
		}
	}
}

// handleFileRemove handles file removal.
func (r *DefaultRegistry) handleFileRemove(path string) {
	// We need to find which pattern was in this file
	// This is tricky since we don't track file->pattern mapping
	// For now, just reload the entire directory
	if err := r.Reload(); err != nil {
		// Log error but continue
		_ = err // In production, this should be logged
	}

	if r.onChange != nil {
		r.onChange("remove", nil)
	}
}

// getPatternByFile attempts to find a pattern that was loaded from the given file.
func (r *DefaultRegistry) getPatternByFile(path string) (*FormatPattern, bool) {
	// Extract the format ID from the filename as a heuristic
	base := filepath.Base(path)
	formatID := strings.TrimSuffix(strings.TrimSuffix(base, ".yaml"), ".yml")

	return r.Get(formatID)
}

// StopWatch stops watching the pattern directory.
func (r *DefaultRegistry) StopWatch() {
	if r.stopChan != nil {
		close(r.stopChan)
	}
	if r.watcher != nil {
		r.watcher.Close()
	}
}

// Clear removes all patterns from the registry.
func (r *DefaultRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.patterns = make(map[string]*FormatPattern)
}
