package pattern

import (
	"path/filepath"
	"runtime"
	"testing"
)

func getProjectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine caller information")
	}
	// From pkg/pattern/ go up two levels to project root
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

func loadEURegulationPattern(t *testing.T) *FormatPattern {
	t.Helper()
	projectRoot := getProjectRoot(t)
	patternsDir := filepath.Join(projectRoot, "patterns")

	registry, err := NewRegistryWithDirectory(patternsDir)
	if err != nil {
		t.Fatalf("Failed to load patterns directory: %v", err)
	}

	euPattern, found := registry.Get("eu-regulation")
	if !found {
		t.Fatal("eu-regulation pattern not found in registry")
	}
	return euPattern
}

func TestNewPatternBridge(t *testing.T) {
	euPattern := loadEURegulationPattern(t)
	bridge := NewPatternBridge(euPattern)

	if bridge == nil {
		t.Fatal("NewPatternBridge returned nil for valid pattern")
	}

	if bridge.FormatID() != "eu-regulation" {
		t.Errorf("FormatID() = %q, want %q", bridge.FormatID(), "eu-regulation")
	}

	if bridge.Jurisdiction() != "EU" {
		t.Errorf("Jurisdiction() = %q, want %q", bridge.Jurisdiction(), "EU")
	}
}

func TestNewPatternBridgeNil(t *testing.T) {
	bridge := NewPatternBridge(nil)
	if bridge != nil {
		t.Error("NewPatternBridge(nil) should return nil")
	}
}

func TestPatternBridgeHierarchyPatterns(t *testing.T) {
	euPattern := loadEURegulationPattern(t)
	bridge := NewPatternBridge(euPattern)

	tests := []struct {
		levelType string
		testLine  string
		wantMatch bool
	}{
		{"chapter", "CHAPTER IV", true},
		{"chapter", "Section 3", false},
		{"section", "Section 3", true},
		{"section", "CHAPTER IV", false},
		{"article", "Article 17", true},
		{"article", "CHAPTER IV", false},
		{"paragraph", "1. The data subject shall have the right", true},
		{"point", "(a) the identity of the controller", true},
		{"subpoint", "(iv) a sub-point reference", true},
	}

	for _, tt := range tests {
		t.Run(tt.levelType+"_"+tt.testLine, func(t *testing.T) {
			compiledPattern := bridge.HierarchyPattern(tt.levelType)
			if compiledPattern == nil {
				t.Fatalf("HierarchyPattern(%q) returned nil", tt.levelType)
			}

			gotMatch := compiledPattern.MatchString(tt.testLine)
			if gotMatch != tt.wantMatch {
				t.Errorf("HierarchyPattern(%q).MatchString(%q) = %v, want %v",
					tt.levelType, tt.testLine, gotMatch, tt.wantMatch)
			}
		})
	}
}

func TestPatternBridgeNonExistentLevel(t *testing.T) {
	euPattern := loadEURegulationPattern(t)
	bridge := NewPatternBridge(euPattern)

	compiledPattern := bridge.HierarchyPattern("nonexistent")
	if compiledPattern != nil {
		t.Error("HierarchyPattern(\"nonexistent\") should return nil")
	}
}

func TestPatternBridgePreamblePatterns(t *testing.T) {
	euPattern := loadEURegulationPattern(t)
	bridge := NewPatternBridge(euPattern)

	startPattern := bridge.PreambleStartPattern()
	if startPattern == nil {
		t.Fatal("PreambleStartPattern() returned nil")
	}

	endPattern := bridge.PreambleEndPattern()
	if endPattern == nil {
		t.Fatal("PreambleEndPattern() returned nil")
	}

	recitalPattern := bridge.RecitalPattern()
	if recitalPattern == nil {
		t.Fatal("RecitalPattern() returned nil")
	}

	// Test preamble start
	if !startPattern.MatchString("THE EUROPEAN PARLIAMENT AND THE COUNCIL OF THE EUROPEAN UNION") {
		t.Error("PreambleStartPattern should match EU Parliament text")
	}

	// Test preamble end
	if !endPattern.MatchString("HAVE ADOPTED THIS REGULATION:") {
		t.Error("PreambleEndPattern should match HAVE ADOPTED text")
	}

	// Test recital
	if m := recitalPattern.FindStringSubmatch("(1) The protection of natural persons"); m == nil {
		t.Error("RecitalPattern should match numbered recitals")
	} else if m[1] != "1" {
		t.Errorf("RecitalPattern group 1 = %q, want %q", m[1], "1")
	}
}

func TestPatternBridgeDefinitionPattern(t *testing.T) {
	euPattern := loadEURegulationPattern(t)
	bridge := NewPatternBridge(euPattern)

	defPattern := bridge.DefinitionPattern()
	if defPattern == nil {
		t.Fatal("DefinitionPattern() returned nil")
	}

	tests := []struct {
		name     string
		line     string
		wantTerm string
	}{
		{
			name:     "single quotes",
			line:     "(1) 'personal data' means any information relating to an identified person",
			wantTerm: "personal data",
		},
		{
			name:     "double quotes",
			line:     `(2) "processing" means any operation performed on personal data`,
			wantTerm: "processing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := defPattern.FindStringSubmatch(tt.line)
			if m == nil {
				t.Errorf("DefinitionPattern should match: %s", tt.line)
				return
			}
			if len(m) > 2 && m[2] != tt.wantTerm {
				t.Errorf("DefinitionPattern term = %q, want %q", m[2], tt.wantTerm)
			}
		})
	}
}

func TestPatternBridgeDefinitionLocations(t *testing.T) {
	euPattern := loadEURegulationPattern(t)
	bridge := NewPatternBridge(euPattern)

	locations := bridge.DefinitionLocations()
	if len(locations) == 0 {
		t.Fatal("DefinitionLocations() returned empty")
	}

	foundSectionNumber := false
	foundSectionTitle := false
	for _, loc := range locations {
		if loc.SectionNumber == 4 {
			foundSectionNumber = true
		}
		if loc.SectionTitle != "" {
			foundSectionTitle = true
		}
	}

	if !foundSectionNumber {
		t.Error("Expected definition location with section_number 4")
	}
	if !foundSectionTitle {
		t.Error("Expected definition location with section_title pattern")
	}
}

func TestPatternBridgeInternalRefPatterns(t *testing.T) {
	euPattern := loadEURegulationPattern(t)
	bridge := NewPatternBridge(euPattern)

	internalPatterns := bridge.InternalRefPatterns()
	if len(internalPatterns) == 0 {
		t.Fatal("InternalRefPatterns() returned empty")
	}

	// Verify we have patterns for key reference types
	targetsSeen := make(map[string]bool)
	for _, refPattern := range internalPatterns {
		targetsSeen[refPattern.Target] = true
		if refPattern.compiled == nil {
			t.Errorf("Internal reference pattern %q is not compiled", refPattern.Pattern)
		}
	}

	expectedTargets := []string{"article", "chapter", "paragraph", "point", "section"}
	for _, target := range expectedTargets {
		if !targetsSeen[target] {
			t.Errorf("Expected internal reference pattern with target %q", target)
		}
	}
}

func TestPatternBridgeExternalRefPatterns(t *testing.T) {
	euPattern := loadEURegulationPattern(t)
	bridge := NewPatternBridge(euPattern)

	externalPatterns := bridge.ExternalRefPatterns()
	if len(externalPatterns) == 0 {
		t.Fatal("ExternalRefPatterns() returned empty")
	}

	typesSeen := make(map[string]bool)
	for _, refPattern := range externalPatterns {
		typesSeen[refPattern.Type] = true
		if refPattern.compiled == nil {
			t.Errorf("External reference pattern %q is not compiled", refPattern.Pattern)
		}
	}

	expectedTypes := []string{"directive", "regulation", "treaty", "decision"}
	for _, refType := range expectedTypes {
		if !typesSeen[refType] {
			t.Errorf("Expected external reference pattern with type %q", refType)
		}
	}
}

func TestBridgeFromRegistry(t *testing.T) {
	projectRoot := getProjectRoot(t)
	patternsDir := filepath.Join(projectRoot, "patterns")

	registry, err := NewRegistryWithDirectory(patternsDir)
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}

	bridge := BridgeFromRegistry(registry, "eu-regulation")
	if bridge == nil {
		t.Fatal("BridgeFromRegistry returned nil for eu-regulation")
	}

	if bridge.FormatID() != "eu-regulation" {
		t.Errorf("FormatID() = %q, want %q", bridge.FormatID(), "eu-regulation")
	}
}

func TestBridgeFromRegistryNotFound(t *testing.T) {
	registry := NewRegistry()

	bridge := BridgeFromRegistry(registry, "nonexistent")
	if bridge != nil {
		t.Error("BridgeFromRegistry should return nil for nonexistent format")
	}
}

func TestDetectAndBridge(t *testing.T) {
	projectRoot := getProjectRoot(t)
	patternsDir := filepath.Join(projectRoot, "patterns")

	registry, err := NewRegistryWithDirectory(patternsDir)
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}

	euContent := `Regulation (EU) 2016/679 of the European Parliament and of the Council
of 27 April 2016
on the protection of natural persons with regard to the processing of personal data
(General Data Protection Regulation)

THE EUROPEAN PARLIAMENT AND THE COUNCIL OF THE EUROPEAN UNION,

HAVE ADOPTED THIS REGULATION:

CHAPTER I
General provisions

Article 1
Subject-matter and objectives`

	bridge := DetectAndBridge(registry, euContent, 0.3)
	if bridge == nil {
		t.Fatal("DetectAndBridge returned nil for EU regulation content")
	}

	if bridge.FormatID() != "eu-regulation" {
		t.Errorf("Detected format = %q, want %q", bridge.FormatID(), "eu-regulation")
	}
}

func TestDetectAndBridgeLowConfidence(t *testing.T) {
	projectRoot := getProjectRoot(t)
	patternsDir := filepath.Join(projectRoot, "patterns")

	registry, err := NewRegistryWithDirectory(patternsDir)
	if err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}

	plainContent := "This is plain text with no regulatory formatting whatsoever."

	bridge := DetectAndBridge(registry, plainContent, 0.5)
	if bridge != nil {
		t.Error("DetectAndBridge should return nil for non-regulatory content")
	}
}

func TestAllHierarchyLevels(t *testing.T) {
	euPattern := loadEURegulationPattern(t)
	bridge := NewPatternBridge(euPattern)

	levels := bridge.AllHierarchyLevels()
	if len(levels) < 4 {
		t.Errorf("AllHierarchyLevels() returned %d levels, want >= 4", len(levels))
	}

	// Verify order: chapter, section, article, paragraph, point, subpoint
	expectedOrder := []string{"chapter", "section", "article", "paragraph", "point", "subpoint"}
	for i, expectedType := range expectedOrder {
		if i >= len(levels) {
			break
		}
		if levels[i].Type != expectedType {
			t.Errorf("Level %d type = %q, want %q", i, levels[i].Type, expectedType)
		}
	}
}
