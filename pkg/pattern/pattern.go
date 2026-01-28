// Package pattern provides a pluggable pattern registry for legislative document format detection and parsing.
package pattern

import (
	"fmt"
	"regexp"
)

// FormatPattern defines patterns for detecting and parsing a specific legislative document format.
type FormatPattern struct {
	// Metadata
	Name         string `yaml:"name" json:"name"`
	Version      string `yaml:"version" json:"version"`
	Jurisdiction string `yaml:"jurisdiction" json:"jurisdiction"`
	FormatID     string `yaml:"format_id" json:"format_id"`

	// Detection configuration
	Detection DetectionConfig `yaml:"detection" json:"detection"`

	// Structure parsing configuration
	Structure StructureConfig `yaml:"structure" json:"structure"`

	// Definition extraction configuration
	Definitions DefinitionConfig `yaml:"definitions" json:"definitions"`

	// Reference extraction configuration
	References ReferenceConfig `yaml:"references" json:"references"`

	// Compiled patterns (populated after loading)
	compiled *CompiledPattern
}

// DetectionConfig defines how to detect if a document matches this format.
type DetectionConfig struct {
	// RequiredIndicators must have at least one match for format detection
	RequiredIndicators []Indicator `yaml:"required_indicators" json:"required_indicators"`

	// OptionalIndicators add to confidence but are not required
	OptionalIndicators []Indicator `yaml:"optional_indicators" json:"optional_indicators"`

	// NegativeIndicators reduce confidence (use negative weights)
	NegativeIndicators []Indicator `yaml:"negative_indicators" json:"negative_indicators"`
}

// Indicator represents a pattern that indicates a particular format.
type Indicator struct {
	Pattern string `yaml:"pattern" json:"pattern"`
	Weight  int    `yaml:"weight" json:"weight"`

	// Compiled regex (populated after loading)
	compiled *regexp.Regexp
}

// StructureConfig defines how to parse document structure.
type StructureConfig struct {
	// Hierarchy defines the structural elements from highest to lowest level
	Hierarchy []HierarchyLevel `yaml:"hierarchy" json:"hierarchy"`

	// Preamble configuration for documents with preambles
	Preamble *PreambleConfig `yaml:"preamble,omitempty" json:"preamble,omitempty"`
}

// HierarchyLevel defines a structural level (chapter, section, article, etc.).
type HierarchyLevel struct {
	Type         string `yaml:"type" json:"type"`                   // chapter, section, article, paragraph, point
	Pattern      string `yaml:"pattern" json:"pattern"`             // Regex pattern
	TitleFollows bool   `yaml:"title_follows" json:"title_follows"` // Title on next line
	TitleInline  bool   `yaml:"title_inline" json:"title_inline"`   // Title in same match
	NumberGroup  int    `yaml:"number_group" json:"number_group"`   // Capture group for number

	// Compiled regex (populated after loading)
	compiled *regexp.Regexp
}

// PreambleConfig defines how to parse preamble sections.
type PreambleConfig struct {
	StartPattern   string `yaml:"start_pattern" json:"start_pattern"`
	EndPattern     string `yaml:"end_pattern" json:"end_pattern"`
	RecitalPattern string `yaml:"recital_pattern" json:"recital_pattern"`

	// Compiled patterns
	startCompiled   *regexp.Regexp
	endCompiled     *regexp.Regexp
	recitalCompiled *regexp.Regexp
}

// DefinitionConfig defines how to extract definitions.
type DefinitionConfig struct {
	// Location specifies where to find definitions
	Location []DefinitionLocation `yaml:"location" json:"location"`

	// Pattern for extracting individual definitions
	Pattern string `yaml:"pattern" json:"pattern"`

	// Compiled pattern
	compiled *regexp.Regexp
}

// DefinitionLocation specifies a location to search for definitions.
type DefinitionLocation struct {
	SectionNumber int    `yaml:"section_number,omitempty" json:"section_number,omitempty"`
	SectionTitle  string `yaml:"section_title,omitempty" json:"section_title,omitempty"`

	// Compiled title pattern
	titleCompiled *regexp.Regexp
}

// ReferenceConfig defines how to extract references.
type ReferenceConfig struct {
	// Internal references within the same document
	Internal []ReferencePattern `yaml:"internal" json:"internal"`

	// External references to other documents
	External []ExternalReferencePattern `yaml:"external" json:"external"`
}

// ReferencePattern defines a pattern for internal references.
type ReferencePattern struct {
	Pattern string         `yaml:"pattern" json:"pattern"`
	Target  string         `yaml:"target" json:"target"` // article, section, paragraph, etc.
	Groups  map[string]int `yaml:"groups" json:"groups"` // Named capture group mappings

	// Compiled pattern
	compiled *regexp.Regexp
}

// ExternalReferencePattern defines a pattern for external references.
type ExternalReferencePattern struct {
	Pattern     string         `yaml:"pattern" json:"pattern"`
	Type        string         `yaml:"type" json:"type"`                 // directive, regulation, usc, etc.
	URITemplate string         `yaml:"uri_template" json:"uri_template"` // Template for generating URIs
	Groups      map[string]int `yaml:"groups" json:"groups"`

	// Compiled pattern
	compiled *regexp.Regexp
}

// CompiledPattern holds all compiled regex patterns for efficient matching.
type CompiledPattern struct {
	RequiredIndicators  []*regexp.Regexp
	OptionalIndicators  []*regexp.Regexp
	NegativeIndicators  []*regexp.Regexp
	HierarchyPatterns   []*regexp.Regexp
	DefinitionPattern   *regexp.Regexp
	InternalRefPatterns []*regexp.Regexp
	ExternalRefPatterns []*regexp.Regexp
}

// Compile compiles all regex patterns in the FormatPattern.
// Returns an error if any pattern fails to compile.
func (fp *FormatPattern) Compile() error {
	fp.compiled = &CompiledPattern{}

	// Compile detection indicators
	for i := range fp.Detection.RequiredIndicators {
		ind := &fp.Detection.RequiredIndicators[i]
		compiled, err := regexp.Compile(ind.Pattern)
		if err != nil {
			return fmt.Errorf("compiling required indicator %d pattern %q: %w", i, ind.Pattern, err)
		}
		ind.compiled = compiled
		fp.compiled.RequiredIndicators = append(fp.compiled.RequiredIndicators, compiled)
	}

	for i := range fp.Detection.OptionalIndicators {
		ind := &fp.Detection.OptionalIndicators[i]
		compiled, err := regexp.Compile(ind.Pattern)
		if err != nil {
			return fmt.Errorf("compiling optional indicator %d pattern %q: %w", i, ind.Pattern, err)
		}
		ind.compiled = compiled
		fp.compiled.OptionalIndicators = append(fp.compiled.OptionalIndicators, compiled)
	}

	for i := range fp.Detection.NegativeIndicators {
		ind := &fp.Detection.NegativeIndicators[i]
		compiled, err := regexp.Compile(ind.Pattern)
		if err != nil {
			return fmt.Errorf("compiling negative indicator %d pattern %q: %w", i, ind.Pattern, err)
		}
		ind.compiled = compiled
		fp.compiled.NegativeIndicators = append(fp.compiled.NegativeIndicators, compiled)
	}

	// Compile hierarchy patterns
	for i := range fp.Structure.Hierarchy {
		level := &fp.Structure.Hierarchy[i]
		if level.Pattern != "" {
			compiled, err := regexp.Compile(level.Pattern)
			if err != nil {
				return fmt.Errorf("compiling hierarchy %s pattern %q: %w", level.Type, level.Pattern, err)
			}
			level.compiled = compiled
			fp.compiled.HierarchyPatterns = append(fp.compiled.HierarchyPatterns, compiled)
		}
	}

	// Compile preamble patterns
	if fp.Structure.Preamble != nil {
		if fp.Structure.Preamble.StartPattern != "" {
			compiled, err := regexp.Compile(fp.Structure.Preamble.StartPattern)
			if err != nil {
				return fmt.Errorf("compiling preamble start pattern: %w", err)
			}
			fp.Structure.Preamble.startCompiled = compiled
		}
		if fp.Structure.Preamble.EndPattern != "" {
			compiled, err := regexp.Compile(fp.Structure.Preamble.EndPattern)
			if err != nil {
				return fmt.Errorf("compiling preamble end pattern: %w", err)
			}
			fp.Structure.Preamble.endCompiled = compiled
		}
		if fp.Structure.Preamble.RecitalPattern != "" {
			compiled, err := regexp.Compile(fp.Structure.Preamble.RecitalPattern)
			if err != nil {
				return fmt.Errorf("compiling recital pattern: %w", err)
			}
			fp.Structure.Preamble.recitalCompiled = compiled
		}
	}

	// Compile definition pattern
	if fp.Definitions.Pattern != "" {
		compiled, err := regexp.Compile(fp.Definitions.Pattern)
		if err != nil {
			return fmt.Errorf("compiling definition pattern: %w", err)
		}
		fp.Definitions.compiled = compiled
		fp.compiled.DefinitionPattern = compiled
	}

	// Compile definition location title patterns
	for i := range fp.Definitions.Location {
		loc := &fp.Definitions.Location[i]
		if loc.SectionTitle != "" {
			compiled, err := regexp.Compile(loc.SectionTitle)
			if err != nil {
				return fmt.Errorf("compiling definition location title pattern: %w", err)
			}
			loc.titleCompiled = compiled
		}
	}

	// Compile internal reference patterns
	for i := range fp.References.Internal {
		ref := &fp.References.Internal[i]
		compiled, err := regexp.Compile(ref.Pattern)
		if err != nil {
			return fmt.Errorf("compiling internal reference %d pattern: %w", i, err)
		}
		ref.compiled = compiled
		fp.compiled.InternalRefPatterns = append(fp.compiled.InternalRefPatterns, compiled)
	}

	// Compile external reference patterns
	for i := range fp.References.External {
		ref := &fp.References.External[i]
		compiled, err := regexp.Compile(ref.Pattern)
		if err != nil {
			return fmt.Errorf("compiling external reference %d pattern: %w", i, err)
		}
		ref.compiled = compiled
		fp.compiled.ExternalRefPatterns = append(fp.compiled.ExternalRefPatterns, compiled)
	}

	return nil
}

// IsCompiled returns true if the pattern has been compiled.
func (fp *FormatPattern) IsCompiled() bool {
	return fp.compiled != nil
}

// GetHierarchyLevel returns the hierarchy level configuration for the given type.
func (fp *FormatPattern) GetHierarchyLevel(levelType string) *HierarchyLevel {
	for i := range fp.Structure.Hierarchy {
		if fp.Structure.Hierarchy[i].Type == levelType {
			return &fp.Structure.Hierarchy[i]
		}
	}
	return nil
}

// Validate checks that the pattern has all required fields.
func (fp *FormatPattern) Validate() error {
	if fp.Name == "" {
		return fmt.Errorf("pattern name is required")
	}
	if fp.FormatID == "" {
		return fmt.Errorf("pattern format_id is required")
	}
	if fp.Version == "" {
		return fmt.Errorf("pattern version is required")
	}
	if len(fp.Detection.RequiredIndicators) == 0 {
		return fmt.Errorf("at least one required indicator is needed for format detection")
	}
	return nil
}
