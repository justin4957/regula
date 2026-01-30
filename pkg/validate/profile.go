package validate

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ProfileSuggestion holds an auto-generated validation profile with reasoning
// explaining why each threshold and weight was chosen.
type ProfileSuggestion struct {
	Profile       *ValidationProfile `json:"profile"`
	Reasoning     []ProfileReasoning `json:"reasoning"`
	DocumentStats *DocumentAnalysis  `json:"document_stats"`
	Confidence    float64            `json:"confidence"`
}

// ProfileReasoning explains why a specific threshold or weight was chosen.
type ProfileReasoning struct {
	Field  string `json:"field"`
	Value  string `json:"value"`
	Reason string `json:"reason"`
}

// DocumentAnalysis holds computed metrics from document structure and content analysis.
type DocumentAnalysis struct {
	ArticleCount      int      `json:"article_count"`
	ChapterCount      int      `json:"chapter_count"`
	SectionCount      int      `json:"section_count"`
	RecitalCount      int      `json:"recital_count"`
	DefinitionCount   int      `json:"definition_count"`
	ReferenceCount    int      `json:"reference_count"`
	ExternalRefCount  int      `json:"external_ref_count"`
	RightsCount       int      `json:"rights_count"`
	ObligationsCount  int      `json:"obligations_count"`
	AvgArticleLength  float64  `json:"avg_article_length"`
	NestingDepth      int      `json:"nesting_depth"`
	DefinitionDensity float64  `json:"definition_density"`
	ReferenceDensity  float64  `json:"reference_density"`
	RightsTypes       []string `json:"rights_types,omitempty"`
	ObligationTypes   []string `json:"obligation_types,omitempty"`
}

// YAMLProfile is the YAML-serializable profile format for reading and writing
// validation profile files.
type YAMLProfile struct {
	Name             string             `yaml:"name"`
	Description      string             `yaml:"description"`
	Expected         YAMLExpected       `yaml:"expected"`
	KnownRights      []string           `yaml:"known_rights,omitempty"`
	KnownObligations []string           `yaml:"known_obligations,omitempty"`
	Weights          YAMLWeights        `yaml:"weights"`
	Thresholds       map[string]float64 `yaml:"thresholds,omitempty"`
	Reasoning        []YAMLReasoning    `yaml:"reasoning,omitempty"`
}

// YAMLExpected holds the expected document structure counts.
type YAMLExpected struct {
	Articles    int `yaml:"articles"`
	Definitions int `yaml:"definitions"`
	Chapters    int `yaml:"chapters"`
}

// YAMLWeights holds the component scoring weights.
type YAMLWeights struct {
	ReferenceResolution float64 `yaml:"reference_resolution"`
	GraphConnectivity   float64 `yaml:"graph_connectivity"`
	DefinitionCoverage  float64 `yaml:"definition_coverage"`
	SemanticExtraction  float64 `yaml:"semantic_extraction"`
	StructureQuality    float64 `yaml:"structure_quality"`
}

// YAMLReasoning documents why a specific value was chosen.
type YAMLReasoning struct {
	Field  string `yaml:"field"`
	Value  string `yaml:"value"`
	Reason string `yaml:"reason"`
}

// ToYAML serializes a ProfileSuggestion to YAML bytes.
func (suggestion *ProfileSuggestion) ToYAML() ([]byte, error) {
	yamlProfile := suggestion.toYAMLProfile()
	return yaml.Marshal(yamlProfile)
}

// ToJSON serializes a ProfileSuggestion to indented JSON bytes.
func (suggestion *ProfileSuggestion) ToJSON() ([]byte, error) {
	return json.MarshalIndent(suggestion, "", "  ")
}

// String returns a human-readable summary of the profile suggestion.
func (suggestion *ProfileSuggestion) String() string {
	var output string

	output += fmt.Sprintf("Profile Suggestion (confidence: %.0f%%)\n", suggestion.Confidence*100)
	output += "=====================================\n\n"

	if suggestion.Profile != nil {
		output += fmt.Sprintf("Name: %s\n", suggestion.Profile.Name)
		output += fmt.Sprintf("Description: %s\n\n", suggestion.Profile.Description)

		output += "Expected Structure:\n"
		output += fmt.Sprintf("  Articles:    %d\n", suggestion.Profile.ExpectedArticles)
		output += fmt.Sprintf("  Definitions: %d\n", suggestion.Profile.ExpectedDefinitions)
		output += fmt.Sprintf("  Chapters:    %d\n\n", suggestion.Profile.ExpectedChapters)

		output += "Weights:\n"
		output += fmt.Sprintf("  Reference Resolution: %.0f%%\n", suggestion.Profile.Weights.ReferenceResolution*100)
		output += fmt.Sprintf("  Graph Connectivity:   %.0f%%\n", suggestion.Profile.Weights.GraphConnectivity*100)
		output += fmt.Sprintf("  Definition Coverage:  %.0f%%\n", suggestion.Profile.Weights.DefinitionCoverage*100)
		output += fmt.Sprintf("  Semantic Extraction:  %.0f%%\n", suggestion.Profile.Weights.SemanticExtraction*100)
		output += fmt.Sprintf("  Structure Quality:    %.0f%%\n\n", suggestion.Profile.Weights.StructureQuality*100)

		if len(suggestion.Profile.KnownRights) > 0 {
			output += fmt.Sprintf("Known Rights: %d\n", len(suggestion.Profile.KnownRights))
		}
		if len(suggestion.Profile.KnownObligations) > 0 {
			output += fmt.Sprintf("Known Obligations: %d\n", len(suggestion.Profile.KnownObligations))
		}
	}

	if len(suggestion.Reasoning) > 0 {
		output += "\nReasoning:\n"
		for _, reasoningEntry := range suggestion.Reasoning {
			output += fmt.Sprintf("  [%s = %s] %s\n", reasoningEntry.Field, reasoningEntry.Value, reasoningEntry.Reason)
		}
	}

	return output
}

// toYAMLProfile converts a ProfileSuggestion to a YAMLProfile for serialization.
func (suggestion *ProfileSuggestion) toYAMLProfile() *YAMLProfile {
	yamlProfile := &YAMLProfile{}

	if suggestion.Profile != nil {
		yamlProfile.Name = suggestion.Profile.Name
		yamlProfile.Description = suggestion.Profile.Description
		yamlProfile.Expected = YAMLExpected{
			Articles:    suggestion.Profile.ExpectedArticles,
			Definitions: suggestion.Profile.ExpectedDefinitions,
			Chapters:    suggestion.Profile.ExpectedChapters,
		}
		yamlProfile.KnownRights = suggestion.Profile.KnownRights
		yamlProfile.KnownObligations = suggestion.Profile.KnownObligations
		yamlProfile.Weights = YAMLWeights{
			ReferenceResolution: suggestion.Profile.Weights.ReferenceResolution,
			GraphConnectivity:   suggestion.Profile.Weights.GraphConnectivity,
			DefinitionCoverage:  suggestion.Profile.Weights.DefinitionCoverage,
			SemanticExtraction:  suggestion.Profile.Weights.SemanticExtraction,
			StructureQuality:    suggestion.Profile.Weights.StructureQuality,
		}
	}

	yamlReasoningEntries := make([]YAMLReasoning, len(suggestion.Reasoning))
	for i, reasoningEntry := range suggestion.Reasoning {
		yamlReasoningEntries[i] = YAMLReasoning{
			Field:  reasoningEntry.Field,
			Value:  reasoningEntry.Value,
			Reason: reasoningEntry.Reason,
		}
	}
	yamlProfile.Reasoning = yamlReasoningEntries

	return yamlProfile
}

// ProfileFromYAML deserializes YAML bytes into a ValidationProfile.
func ProfileFromYAML(yamlData []byte) (*ValidationProfile, error) {
	var yamlProfile YAMLProfile
	if err := yaml.Unmarshal(yamlData, &yamlProfile); err != nil {
		return nil, fmt.Errorf("failed to parse YAML profile: %w", err)
	}

	profile := &ValidationProfile{
		Name:                yamlProfile.Name,
		Description:         yamlProfile.Description,
		ExpectedArticles:    yamlProfile.Expected.Articles,
		ExpectedDefinitions: yamlProfile.Expected.Definitions,
		ExpectedChapters:    yamlProfile.Expected.Chapters,
		KnownRights:         yamlProfile.KnownRights,
		KnownObligations:    yamlProfile.KnownObligations,
		Weights: ValidationWeights{
			ReferenceResolution: yamlProfile.Weights.ReferenceResolution,
			GraphConnectivity:   yamlProfile.Weights.GraphConnectivity,
			DefinitionCoverage:  yamlProfile.Weights.DefinitionCoverage,
			SemanticExtraction:  yamlProfile.Weights.SemanticExtraction,
			StructureQuality:    yamlProfile.Weights.StructureQuality,
		},
	}

	// Ensure nil slices become empty slices
	if profile.KnownRights == nil {
		profile.KnownRights = []string{}
	}
	if profile.KnownObligations == nil {
		profile.KnownObligations = []string{}
	}

	return profile, nil
}

// LoadProfileFromFile reads a YAML validation profile from disk.
func LoadProfileFromFile(filePath string) (*ValidationProfile, error) {
	yamlData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile file %s: %w", filePath, err)
	}
	return ProfileFromYAML(yamlData)
}

// SaveProfileToFile writes a ProfileSuggestion to a YAML file on disk.
func SaveProfileToFile(suggestion *ProfileSuggestion, filePath string) error {
	yamlData, err := suggestion.ToYAML()
	if err != nil {
		return fmt.Errorf("failed to serialize profile to YAML: %w", err)
	}
	if err := os.WriteFile(filePath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write profile file %s: %w", filePath, err)
	}
	return nil
}
