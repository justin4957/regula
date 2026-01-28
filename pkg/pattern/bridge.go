package pattern

import (
	"regexp"
	"strings"
)

// PatternBridge provides compiled regex patterns from a loaded FormatPattern
// for use by the document parser. It bridges the pattern library (YAML-based)
// with the extraction pipeline.
type PatternBridge struct {
	pattern *FormatPattern
}

// NewPatternBridge creates a bridge from a compiled FormatPattern.
// Returns nil if the pattern is nil or not compiled.
func NewPatternBridge(formatPattern *FormatPattern) *PatternBridge {
	if formatPattern == nil || !formatPattern.IsCompiled() {
		return nil
	}
	return &PatternBridge{pattern: formatPattern}
}

// FormatID returns the format identifier of the underlying pattern.
func (b *PatternBridge) FormatID() string {
	return b.pattern.FormatID
}

// Jurisdiction returns the jurisdiction of the underlying pattern.
func (b *PatternBridge) Jurisdiction() string {
	return b.pattern.Jurisdiction
}

// HierarchyPattern returns the compiled regex for a given hierarchy level type
// (e.g., "chapter", "section", "article", "paragraph", "point", "subpoint").
func (b *PatternBridge) HierarchyPattern(levelType string) *regexp.Regexp {
	level := b.pattern.GetHierarchyLevel(levelType)
	if level == nil {
		return nil
	}
	return level.compiled
}

// HierarchyLevelInfo returns the full HierarchyLevel config for a given type.
func (b *PatternBridge) HierarchyLevelInfo(levelType string) *HierarchyLevel {
	return b.pattern.GetHierarchyLevel(levelType)
}

// AllHierarchyLevels returns all hierarchy level configurations.
func (b *PatternBridge) AllHierarchyLevels() []HierarchyLevel {
	return b.pattern.Structure.Hierarchy
}

// PreambleStartPattern returns the compiled regex for detecting preamble start.
func (b *PatternBridge) PreambleStartPattern() *regexp.Regexp {
	if b.pattern.Structure.Preamble == nil {
		return nil
	}
	return b.pattern.Structure.Preamble.startCompiled
}

// PreambleEndPattern returns the compiled regex for detecting preamble end.
func (b *PatternBridge) PreambleEndPattern() *regexp.Regexp {
	if b.pattern.Structure.Preamble == nil {
		return nil
	}
	return b.pattern.Structure.Preamble.endCompiled
}

// RecitalPattern returns the compiled regex for detecting recitals.
func (b *PatternBridge) RecitalPattern() *regexp.Regexp {
	if b.pattern.Structure.Preamble == nil {
		return nil
	}
	return b.pattern.Structure.Preamble.recitalCompiled
}

// DefinitionPattern returns the compiled regex for extracting definitions.
func (b *PatternBridge) DefinitionPattern() *regexp.Regexp {
	return b.pattern.Definitions.compiled
}

// DefinitionLocations returns the locations where definitions can be found.
func (b *PatternBridge) DefinitionLocations() []DefinitionLocation {
	return b.pattern.Definitions.Location
}

// InternalRefPatterns returns all compiled internal reference patterns
// along with their target and group metadata.
func (b *PatternBridge) InternalRefPatterns() []ReferencePattern {
	return b.pattern.References.Internal
}

// ExternalRefPatterns returns all compiled external reference patterns
// along with their type, URI template, and group metadata.
func (b *PatternBridge) ExternalRefPatterns() []ExternalReferencePattern {
	return b.pattern.References.External
}

// DetectFormat uses the pattern registry's detector to determine if content
// matches this pattern's format. Returns the confidence score (0.0 to 1.0).
func (b *PatternBridge) DetectFormat(content string) float64 {
	registry := NewRegistry()
	// Register a copy to avoid issues
	_ = registry.Register(b.pattern)

	detector := NewFormatDetector(registry)
	bestMatch := detector.DetectBest(content)
	if bestMatch == nil {
		return 0.0
	}
	return bestMatch.Confidence
}

// BridgeFromRegistry looks up a format pattern in the registry by ID,
// and returns a PatternBridge for it.
func BridgeFromRegistry(registry Registry, formatID string) *PatternBridge {
	formatPattern, found := registry.Get(formatID)
	if !found {
		return nil
	}
	return NewPatternBridge(formatPattern)
}

// DetectAndBridge uses the registry's detector to find the best-matching
// format for the given content, then returns a PatternBridge for that format.
// Returns nil if no format matches above the confidence threshold.
func DetectAndBridge(registry Registry, content string, confidenceThreshold float64) *PatternBridge {
	detector := NewFormatDetector(registry)
	matches := detector.DetectWithThreshold(content, confidenceThreshold)
	if len(matches) == 0 {
		return nil
	}
	return NewPatternBridge(matches[0].Pattern)
}

// DetectFormatFromLines is a convenience function that detects format from
// a slice of lines, returning the best matching PatternBridge.
func DetectFormatFromLines(registry Registry, lines []string, confidenceThreshold float64) *PatternBridge {
	content := strings.Join(lines, "\n")
	return DetectAndBridge(registry, content, confidenceThreshold)
}
