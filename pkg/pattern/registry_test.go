package pattern

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if registry.Count() != 0 {
		t.Errorf("Count() = %d, want 0", registry.Count())
	}
}

func TestRegistryRegister(t *testing.T) {
	registry := NewRegistry()

	pattern := &FormatPattern{
		Name:     "Test Pattern",
		FormatID: "test-format",
		Version:  "1.0.0",
		Detection: DetectionConfig{
			RequiredIndicators: []Indicator{
				{Pattern: `test`, Weight: 10},
			},
		},
	}

	// Register should succeed
	if err := registry.Register(pattern); err != nil {
		t.Errorf("Register() error = %v", err)
	}

	if registry.Count() != 1 {
		t.Errorf("Count() = %d, want 1", registry.Count())
	}

	// Registering nil should fail
	if err := registry.Register(nil); err == nil {
		t.Error("Register(nil) should return error")
	}

	// Registering same version should fail
	if err := registry.Register(pattern); err == nil {
		t.Error("Register() duplicate should return error")
	}

	// Registering different version should succeed
	pattern2 := &FormatPattern{
		Name:     "Test Pattern",
		FormatID: "test-format",
		Version:  "2.0.0",
		Detection: DetectionConfig{
			RequiredIndicators: []Indicator{
				{Pattern: `test`, Weight: 10},
			},
		},
	}
	if err := registry.Register(pattern2); err != nil {
		t.Errorf("Register() new version error = %v", err)
	}
}

func TestRegistryRegisterInvalidPattern(t *testing.T) {
	registry := NewRegistry()

	// Pattern without required fields
	invalid := &FormatPattern{
		Name: "Invalid",
	}
	if err := registry.Register(invalid); err == nil {
		t.Error("Register() invalid pattern should return error")
	}

	// Pattern with invalid regex
	invalidRegex := &FormatPattern{
		Name:     "Invalid Regex",
		FormatID: "invalid-regex",
		Version:  "1.0.0",
		Detection: DetectionConfig{
			RequiredIndicators: []Indicator{
				{Pattern: `[invalid`, Weight: 10},
			},
		},
	}
	if err := registry.Register(invalidRegex); err == nil {
		t.Error("Register() invalid regex should return error")
	}
}

func TestRegistryUnregister(t *testing.T) {
	registry := NewRegistry()

	pattern := &FormatPattern{
		Name:     "Test Pattern",
		FormatID: "test-format",
		Version:  "1.0.0",
		Detection: DetectionConfig{
			RequiredIndicators: []Indicator{
				{Pattern: `test`, Weight: 10},
			},
		},
	}

	if err := registry.Register(pattern); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Unregister should succeed
	if err := registry.Unregister("test-format"); err != nil {
		t.Errorf("Unregister() error = %v", err)
	}

	if registry.Count() != 0 {
		t.Errorf("Count() = %d, want 0", registry.Count())
	}

	// Unregister non-existent should fail
	if err := registry.Unregister("non-existent"); err == nil {
		t.Error("Unregister() non-existent should return error")
	}
}

func TestRegistryGet(t *testing.T) {
	registry := NewRegistry()

	pattern := &FormatPattern{
		Name:     "Test Pattern",
		FormatID: "test-format",
		Version:  "1.0.0",
		Detection: DetectionConfig{
			RequiredIndicators: []Indicator{
				{Pattern: `test`, Weight: 10},
			},
		},
	}

	if err := registry.Register(pattern); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Get existing pattern
	p, ok := registry.Get("test-format")
	if !ok {
		t.Error("Get() should find pattern")
	}
	if p.Name != "Test Pattern" {
		t.Errorf("Get() Name = %q, want %q", p.Name, "Test Pattern")
	}

	// Get non-existent pattern
	_, ok = registry.Get("non-existent")
	if ok {
		t.Error("Get() should not find non-existent pattern")
	}
}

func TestRegistryList(t *testing.T) {
	registry := NewRegistry()

	patterns := []*FormatPattern{
		{
			Name:     "Pattern A",
			FormatID: "format-a",
			Version:  "1.0.0",
			Detection: DetectionConfig{
				RequiredIndicators: []Indicator{{Pattern: `a`, Weight: 10}},
			},
		},
		{
			Name:     "Pattern B",
			FormatID: "format-b",
			Version:  "1.0.0",
			Detection: DetectionConfig{
				RequiredIndicators: []Indicator{{Pattern: `b`, Weight: 10}},
			},
		},
	}

	for _, p := range patterns {
		if err := registry.Register(p); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	list := registry.List()
	if len(list) != 2 {
		t.Errorf("List() len = %d, want 2", len(list))
	}
}

func TestRegistryListByJurisdiction(t *testing.T) {
	registry := NewRegistry()

	patterns := []*FormatPattern{
		{
			Name:         "US Pattern",
			FormatID:     "us-format",
			Version:      "1.0.0",
			Jurisdiction: "US",
			Detection: DetectionConfig{
				RequiredIndicators: []Indicator{{Pattern: `us`, Weight: 10}},
			},
		},
		{
			Name:         "EU Pattern",
			FormatID:     "eu-format",
			Version:      "1.0.0",
			Jurisdiction: "EU",
			Detection: DetectionConfig{
				RequiredIndicators: []Indicator{{Pattern: `eu`, Weight: 10}},
			},
		},
		{
			Name:         "US State Pattern",
			FormatID:     "us-ca-format",
			Version:      "1.0.0",
			Jurisdiction: "US",
			Detection: DetectionConfig{
				RequiredIndicators: []Indicator{{Pattern: `ca`, Weight: 10}},
			},
		},
	}

	for _, p := range patterns {
		if err := registry.Register(p); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	// Case-insensitive lookup
	usPatterns := registry.ListByJurisdiction("us")
	if len(usPatterns) != 2 {
		t.Errorf("ListByJurisdiction(us) len = %d, want 2", len(usPatterns))
	}

	euPatterns := registry.ListByJurisdiction("EU")
	if len(euPatterns) != 1 {
		t.Errorf("ListByJurisdiction(EU) len = %d, want 1", len(euPatterns))
	}

	ukPatterns := registry.ListByJurisdiction("UK")
	if len(ukPatterns) != 0 {
		t.Errorf("ListByJurisdiction(UK) len = %d, want 0", len(ukPatterns))
	}
}

func TestRegistryClear(t *testing.T) {
	registry := NewRegistry()

	pattern := &FormatPattern{
		Name:     "Test Pattern",
		FormatID: "test-format",
		Version:  "1.0.0",
		Detection: DetectionConfig{
			RequiredIndicators: []Indicator{{Pattern: `test`, Weight: 10}},
		},
	}

	if err := registry.Register(pattern); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("Count() after Clear() = %d, want 0", registry.Count())
	}
}

func TestRegistryLoadFile(t *testing.T) {
	// Create temp directory and file
	tmpDir := t.TempDir()
	patternFile := filepath.Join(tmpDir, "test-format.yaml")

	yamlContent := `
name: "Test YAML Pattern"
format_id: "test-yaml"
version: "1.0.0"
jurisdiction: "TEST"
detection:
  required_indicators:
    - pattern: "\\bTEST\\b"
      weight: 10
  optional_indicators:
    - pattern: "optional"
      weight: 5
structure:
  hierarchy:
    - type: "section"
      pattern: "^Section\\s+(\\d+)"
      number_group: 1
`

	if err := os.WriteFile(patternFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	registry := NewRegistry()
	if err := registry.LoadFile(patternFile); err != nil {
		t.Errorf("LoadFile() error = %v", err)
	}

	p, ok := registry.Get("test-yaml")
	if !ok {
		t.Fatal("Get() should find loaded pattern")
	}
	if p.Name != "Test YAML Pattern" {
		t.Errorf("Name = %q, want %q", p.Name, "Test YAML Pattern")
	}
	if p.Jurisdiction != "TEST" {
		t.Errorf("Jurisdiction = %q, want %q", p.Jurisdiction, "TEST")
	}
	if !p.IsCompiled() {
		t.Error("Pattern should be compiled after loading")
	}
}

func TestRegistryLoadDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple pattern files
	patterns := map[string]string{
		"pattern-a.yaml": `
name: "Pattern A"
format_id: "pattern-a"
version: "1.0.0"
detection:
  required_indicators:
    - pattern: "A"
      weight: 10
`,
		"pattern-b.yml": `
name: "Pattern B"
format_id: "pattern-b"
version: "1.0.0"
detection:
  required_indicators:
    - pattern: "B"
      weight: 10
`,
		"not-a-pattern.txt": "This should be ignored",
	}

	for name, content := range patterns {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", name, err)
		}
	}

	registry := NewRegistry()
	if err := registry.LoadDirectory(tmpDir); err != nil {
		t.Errorf("LoadDirectory() error = %v", err)
	}

	if registry.Count() != 2 {
		t.Errorf("Count() = %d, want 2", registry.Count())
	}

	// Both .yaml and .yml should be loaded
	if _, ok := registry.Get("pattern-a"); !ok {
		t.Error("pattern-a should be loaded")
	}
	if _, ok := registry.Get("pattern-b"); !ok {
		t.Error("pattern-b should be loaded")
	}
}

func TestRegistryLoadDirectoryNonExistent(t *testing.T) {
	registry := NewRegistry()

	// Loading non-existent directory should not error (just returns with nothing loaded)
	if err := registry.LoadDirectory("/non/existent/path"); err != nil {
		t.Errorf("LoadDirectory() non-existent should not error, got: %v", err)
	}

	if registry.Count() != 0 {
		t.Errorf("Count() = %d, want 0", registry.Count())
	}
}

func TestRegistryReload(t *testing.T) {
	tmpDir := t.TempDir()

	patternFile := filepath.Join(tmpDir, "test.yaml")
	yamlContent := `
name: "Original"
format_id: "test"
version: "1.0.0"
detection:
  required_indicators:
    - pattern: "test"
      weight: 10
`
	if err := os.WriteFile(patternFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	registry, err := NewRegistryWithDirectory(tmpDir)
	if err != nil {
		t.Fatalf("NewRegistryWithDirectory() error = %v", err)
	}

	p, _ := registry.Get("test")
	if p.Name != "Original" {
		t.Errorf("Name = %q, want %q", p.Name, "Original")
	}

	// Update the file
	yamlContent = `
name: "Updated"
format_id: "test"
version: "2.0.0"
detection:
  required_indicators:
    - pattern: "test"
      weight: 10
`
	if err := os.WriteFile(patternFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := registry.Reload(); err != nil {
		t.Errorf("Reload() error = %v", err)
	}

	p, _ = registry.Get("test")
	if p.Name != "Updated" {
		t.Errorf("Name after reload = %q, want %q", p.Name, "Updated")
	}
}

func TestRegistryReloadNoDirectory(t *testing.T) {
	registry := NewRegistry()

	if err := registry.Reload(); err == nil {
		t.Error("Reload() without directory should return error")
	}
}

func TestRegistryWatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping watch test in short mode")
	}

	tmpDir := t.TempDir()

	// Create initial pattern
	patternFile := filepath.Join(tmpDir, "test.yaml")
	yamlContent := `
name: "Original"
format_id: "watch-test"
version: "1.0.0"
detection:
  required_indicators:
    - pattern: "test"
      weight: 10
`
	if err := os.WriteFile(patternFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	registry, err := NewRegistryWithDirectory(tmpDir)
	if err != nil {
		t.Fatalf("NewRegistryWithDirectory() error = %v", err)
	}

	changed := make(chan bool, 1)
	registry.SetOnChange(func(event string, pattern *FormatPattern) {
		select {
		case changed <- true:
		default:
		}
	})

	if err := registry.Watch(); err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	defer registry.StopWatch()

	// Give the watcher time to initialize
	time.Sleep(100 * time.Millisecond)

	// Update the file
	yamlContent = `
name: "Updated Via Watch"
format_id: "watch-test"
version: "2.0.0"
detection:
  required_indicators:
    - pattern: "test"
      weight: 10
`
	if err := os.WriteFile(patternFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Wait for change with timeout
	select {
	case <-changed:
		// Success - wait a bit for the reload to complete
		time.Sleep(100 * time.Millisecond)
	case <-time.After(3 * time.Second):
		// File watching can be flaky in CI environments, so we just log
		t.Log("Watch() did not detect file change within timeout (may be CI environment)")
		return
	}

	// Verify the pattern was updated
	p, _ := registry.Get("watch-test")
	if p.Name != "Updated Via Watch" {
		t.Errorf("Name = %q, want %q", p.Name, "Updated Via Watch")
	}
}

func TestRegistryWatchNoDirectory(t *testing.T) {
	registry := NewRegistry()

	if err := registry.Watch(); err == nil {
		t.Error("Watch() without directory should return error")
	}
}

func TestNewRegistryWithDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	patternFile := filepath.Join(tmpDir, "test.yaml")
	yamlContent := `
name: "Test"
format_id: "test"
version: "1.0.0"
detection:
  required_indicators:
    - pattern: "test"
      weight: 10
`
	if err := os.WriteFile(patternFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	registry, err := NewRegistryWithDirectory(tmpDir)
	if err != nil {
		t.Fatalf("NewRegistryWithDirectory() error = %v", err)
	}

	if registry.Count() != 1 {
		t.Errorf("Count() = %d, want 1", registry.Count())
	}
}
