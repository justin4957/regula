package validate

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coolbeans/regula/pkg/extract"
)

// buildProfileTestDocument creates a Document with the specified article count and structure.
func buildProfileTestDocument(articleCount int, chapterCount int, definitionCount int) *extract.Document {
	doc := &extract.Document{
		Title:      "Test Regulation",
		Identifier: "TEST/2026/1",
		Type:       "regulation",
		Chapters:   make([]*extract.Chapter, 0, chapterCount),
		Preamble: &extract.Preamble{
			Recitals: []*extract.Recital{
				{Number: 1, Text: "Whereas this regulation..."},
				{Number: 2, Text: "Having regard to..."},
			},
		},
	}

	// Distribute articles across chapters
	articlesPerChapter := articleCount
	if chapterCount > 0 {
		articlesPerChapter = articleCount / chapterCount
	}

	articleNumber := 1
	for chapterIndex := 0; chapterIndex < chapterCount; chapterIndex++ {
		chapter := &extract.Chapter{
			Number:   string(rune('I' + chapterIndex)),
			Title:    "Test Chapter",
			Articles: make([]*extract.Article, 0),
		}

		// Last chapter gets remainder articles
		chapterArticleCount := articlesPerChapter
		if chapterIndex == chapterCount-1 {
			chapterArticleCount = articleCount - (articlesPerChapter * (chapterCount - 1))
		}

		// Add sections to first chapter to create nesting
		if chapterIndex == 0 && chapterCount > 2 {
			section := &extract.Section{
				Number:   1,
				Title:    "Test Section",
				Articles: make([]*extract.Article, 0),
			}
			for articleIdx := 0; articleIdx < chapterArticleCount && articleNumber <= articleCount; articleIdx++ {
				section.Articles = append(section.Articles, &extract.Article{
					Number: articleNumber,
					Title:  "Test Article",
					Paragraphs: []*extract.Paragraph{
						{Number: 1, Text: "Test paragraph content."},
						{Number: 2, Text: "Another paragraph."},
					},
				})
				articleNumber++
			}
			chapter.Sections = append(chapter.Sections, section)
		} else {
			for articleIdx := 0; articleIdx < chapterArticleCount && articleNumber <= articleCount; articleIdx++ {
				chapter.Articles = append(chapter.Articles, &extract.Article{
					Number: articleNumber,
					Title:  "Test Article",
					Paragraphs: []*extract.Paragraph{
						{Number: 1, Text: "Test paragraph content."},
					},
				})
				articleNumber++
			}
		}

		doc.Chapters = append(doc.Chapters, chapter)
	}

	// Add definitions to document
	definitionList := make([]*extract.Definition, 0, definitionCount)
	for defIndex := 0; defIndex < definitionCount; defIndex++ {
		definitionList = append(definitionList, &extract.Definition{
			Number: defIndex + 1,
			Term:   "test term",
		})
	}
	doc.Definitions = definitionList

	return doc
}

// buildTestDefinitions creates a slice of DefinedTerm for testing.
func buildTestDefinitions(count int) []*extract.DefinedTerm {
	definitions := make([]*extract.DefinedTerm, count)
	for defIndex := 0; defIndex < count; defIndex++ {
		definitions[defIndex] = &extract.DefinedTerm{
			Number: defIndex + 1,
			Term:   "test_term",
		}
	}
	return definitions
}

// buildTestReferences creates a slice of ResolvedReference for testing.
func buildTestReferences(resolvedCount int, externalCount int) []*extract.ResolvedReference {
	references := make([]*extract.ResolvedReference, 0, resolvedCount+externalCount)
	for refIndex := 0; refIndex < resolvedCount; refIndex++ {
		references = append(references, &extract.ResolvedReference{
			Status: extract.ResolutionResolved,
		})
	}
	for refIndex := 0; refIndex < externalCount; refIndex++ {
		references = append(references, &extract.ResolvedReference{
			Status: extract.ResolutionExternal,
		})
	}
	return references
}

// buildTestSemantics creates semantic annotations with rights and obligations.
func buildTestSemantics(rightsCount int, obligationsCount int) []*extract.SemanticAnnotation {
	semantics := make([]*extract.SemanticAnnotation, 0, rightsCount+obligationsCount)
	rightTypes := []extract.RightType{extract.RightAccess, extract.RightErasure, extract.RightPortability}
	for rightIndex := 0; rightIndex < rightsCount; rightIndex++ {
		semantics = append(semantics, &extract.SemanticAnnotation{
			Type:      extract.SemanticRight,
			RightType: rightTypes[rightIndex%len(rightTypes)],
		})
	}
	obligationTypes := []extract.ObligationType{extract.ObligationConsent, extract.ObligationSecure}
	for obligationIndex := 0; obligationIndex < obligationsCount; obligationIndex++ {
		semantics = append(semantics, &extract.SemanticAnnotation{
			Type:           extract.SemanticObligation,
			ObligationType: obligationTypes[obligationIndex%len(obligationTypes)],
		})
	}
	return semantics
}

// --- SuggestProfile tests ---

func TestSuggestProfile_LargeDocument(t *testing.T) {
	generator := NewProfileGenerator()
	doc := buildProfileTestDocument(99, 11, 26)
	definitions := buildTestDefinitions(26)
	references := buildTestReferences(200, 10)
	semantics := buildTestSemantics(18, 24)

	suggestion := generator.SuggestProfile(doc, definitions, references, semantics, nil)

	if suggestion == nil {
		t.Fatal("SuggestProfile returned nil")
	}

	if suggestion.Profile == nil {
		t.Fatal("Profile is nil")
	}

	if suggestion.Profile.ExpectedArticles != 99 {
		t.Errorf("ExpectedArticles = %d, want 99", suggestion.Profile.ExpectedArticles)
	}

	if suggestion.Profile.ExpectedDefinitions != 26 {
		t.Errorf("ExpectedDefinitions = %d, want 26", suggestion.Profile.ExpectedDefinitions)
	}

	if suggestion.Profile.ExpectedChapters != 11 {
		t.Errorf("ExpectedChapters = %d, want 11", suggestion.Profile.ExpectedChapters)
	}

	// Weights should sum to 1.0
	weightSum := suggestion.Profile.Weights.ReferenceResolution +
		suggestion.Profile.Weights.GraphConnectivity +
		suggestion.Profile.Weights.DefinitionCoverage +
		suggestion.Profile.Weights.SemanticExtraction +
		suggestion.Profile.Weights.StructureQuality

	if math.Abs(weightSum-1.0) > 0.02 {
		t.Errorf("Weights sum to %.4f, want 1.0", weightSum)
	}

	// Should have high confidence with all data present
	if suggestion.Confidence < 0.7 {
		t.Errorf("Confidence = %.2f, want >= 0.7 for complete document", suggestion.Confidence)
	}
}

func TestSuggestProfile_SmallDocument(t *testing.T) {
	generator := NewProfileGenerator()
	doc := buildProfileTestDocument(5, 2, 3)
	definitions := buildTestDefinitions(3)
	references := buildTestReferences(10, 2)
	semantics := buildTestSemantics(2, 3)

	suggestion := generator.SuggestProfile(doc, definitions, references, semantics, nil)

	if suggestion.Profile.ExpectedArticles != 5 {
		t.Errorf("ExpectedArticles = %d, want 5", suggestion.Profile.ExpectedArticles)
	}

	if suggestion.Profile.ExpectedDefinitions != 3 {
		t.Errorf("ExpectedDefinitions = %d, want 3", suggestion.Profile.ExpectedDefinitions)
	}

	// Small documents should have lower confidence
	if suggestion.Confidence >= 1.0 {
		t.Error("Confidence should be < 1.0 for small document")
	}
}

func TestSuggestProfile_NoDefinitions(t *testing.T) {
	generator := NewProfileGenerator()
	doc := buildProfileTestDocument(20, 3, 0)
	definitions := buildTestDefinitions(0)
	references := buildTestReferences(50, 5)
	semantics := buildTestSemantics(5, 8)

	suggestion := generator.SuggestProfile(doc, definitions, references, semantics, nil)

	// Definition weight should be lower than default (0.20)
	if suggestion.Profile.Weights.DefinitionCoverage >= 0.20 {
		t.Errorf("DefinitionCoverage weight = %.2f, want < 0.20 when no definitions found",
			suggestion.Profile.Weights.DefinitionCoverage)
	}
}

func TestSuggestProfile_HeavyReferences(t *testing.T) {
	generator := NewProfileGenerator()
	doc := buildProfileTestDocument(20, 3, 10)
	definitions := buildTestDefinitions(10)
	// 100 references for 20 articles = 5.0 refs/article > 3.0 threshold
	references := buildTestReferences(100, 10)
	semantics := buildTestSemantics(5, 8)

	suggestion := generator.SuggestProfile(doc, definitions, references, semantics, nil)

	// Reference weight should be boosted above default (0.25)
	defaultWeights := DefaultWeights()
	if suggestion.Profile.Weights.ReferenceResolution <= defaultWeights.ReferenceResolution-0.05 {
		t.Errorf("ReferenceResolution weight = %.2f, expected higher than default for reference-heavy document",
			suggestion.Profile.Weights.ReferenceResolution)
	}
}

func TestSuggestProfile_NoSemantics(t *testing.T) {
	generator := NewProfileGenerator()
	doc := buildProfileTestDocument(30, 4, 15)
	definitions := buildTestDefinitions(15)
	references := buildTestReferences(80, 5)
	semantics := buildTestSemantics(0, 0)

	suggestion := generator.SuggestProfile(doc, definitions, references, semantics, nil)

	// Semantic weight should be reduced
	if suggestion.Profile.Weights.SemanticExtraction >= 0.20 {
		t.Errorf("SemanticExtraction weight = %.2f, want < 0.20 when no semantics found",
			suggestion.Profile.Weights.SemanticExtraction)
	}
}

func TestSuggestProfile_Reasoning(t *testing.T) {
	generator := NewProfileGenerator()
	doc := buildProfileTestDocument(50, 8, 20)
	definitions := buildTestDefinitions(20)
	references := buildTestReferences(150, 10)
	semantics := buildTestSemantics(12, 15)

	suggestion := generator.SuggestProfile(doc, definitions, references, semantics, nil)

	if len(suggestion.Reasoning) == 0 {
		t.Error("Reasoning should not be empty")
	}

	// Should have reasoning for expected counts
	hasArticleReasoning := false
	hasDefinitionReasoning := false
	hasChapterReasoning := false

	for _, reasoningEntry := range suggestion.Reasoning {
		if reasoningEntry.Field == "expected.articles" {
			hasArticleReasoning = true
		}
		if reasoningEntry.Field == "expected.definitions" {
			hasDefinitionReasoning = true
		}
		if reasoningEntry.Field == "expected.chapters" {
			hasChapterReasoning = true
		}
	}

	if !hasArticleReasoning {
		t.Error("Missing reasoning for expected.articles")
	}
	if !hasDefinitionReasoning {
		t.Error("Missing reasoning for expected.definitions")
	}
	if !hasChapterReasoning {
		t.Error("Missing reasoning for expected.chapters")
	}
}

func TestSuggestProfile_KnownRightsAndObligations(t *testing.T) {
	generator := NewProfileGenerator()
	doc := buildProfileTestDocument(30, 5, 10)
	definitions := buildTestDefinitions(10)
	references := buildTestReferences(50, 5)
	semantics := buildTestSemantics(15, 20)

	suggestion := generator.SuggestProfile(doc, definitions, references, semantics, nil)

	if len(suggestion.Profile.KnownRights) == 0 {
		t.Error("KnownRights should not be empty when semantic rights are present")
	}

	if len(suggestion.Profile.KnownObligations) == 0 {
		t.Error("KnownObligations should not be empty when semantic obligations are present")
	}
}

// --- YAML serialization tests ---

func TestProfileSuggestion_ToYAML(t *testing.T) {
	generator := NewProfileGenerator()
	doc := buildProfileTestDocument(30, 5, 10)
	definitions := buildTestDefinitions(10)
	references := buildTestReferences(50, 5)
	semantics := buildTestSemantics(5, 8)

	suggestion := generator.SuggestProfile(doc, definitions, references, semantics, nil)

	yamlData, err := suggestion.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}

	yamlStr := string(yamlData)

	// Verify YAML contains expected fields
	expectedFields := []string{
		"name:",
		"description:",
		"expected:",
		"articles:",
		"definitions:",
		"chapters:",
		"weights:",
		"reference_resolution:",
		"graph_connectivity:",
		"definition_coverage:",
		"semantic_extraction:",
		"structure_quality:",
		"reasoning:",
	}

	for _, expectedField := range expectedFields {
		if !strings.Contains(yamlStr, expectedField) {
			t.Errorf("YAML output missing expected field: %q", expectedField)
		}
	}
}

func TestProfileSuggestion_ToYAML_Roundtrip(t *testing.T) {
	generator := NewProfileGenerator()
	doc := buildProfileTestDocument(50, 8, 20)
	definitions := buildTestDefinitions(20)
	references := buildTestReferences(100, 10)
	semantics := buildTestSemantics(10, 15)

	suggestion := generator.SuggestProfile(doc, definitions, references, semantics, nil)

	yamlData, err := suggestion.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}

	// Parse back
	loadedProfile, err := ProfileFromYAML(yamlData)
	if err != nil {
		t.Fatalf("ProfileFromYAML failed: %v", err)
	}

	// Verify round-trip
	if loadedProfile.ExpectedArticles != suggestion.Profile.ExpectedArticles {
		t.Errorf("ExpectedArticles = %d, want %d", loadedProfile.ExpectedArticles, suggestion.Profile.ExpectedArticles)
	}

	if loadedProfile.ExpectedDefinitions != suggestion.Profile.ExpectedDefinitions {
		t.Errorf("ExpectedDefinitions = %d, want %d", loadedProfile.ExpectedDefinitions, suggestion.Profile.ExpectedDefinitions)
	}

	if loadedProfile.ExpectedChapters != suggestion.Profile.ExpectedChapters {
		t.Errorf("ExpectedChapters = %d, want %d", loadedProfile.ExpectedChapters, suggestion.Profile.ExpectedChapters)
	}

	if math.Abs(loadedProfile.Weights.ReferenceResolution-suggestion.Profile.Weights.ReferenceResolution) > 0.01 {
		t.Errorf("ReferenceResolution weight = %.4f, want %.4f",
			loadedProfile.Weights.ReferenceResolution, suggestion.Profile.Weights.ReferenceResolution)
	}
}

func TestProfileFromYAML(t *testing.T) {
	yamlInput := `
name: "Test Profile"
description: "A test profile"
expected:
  articles: 50
  definitions: 15
  chapters: 8
known_rights:
  - access
  - erasure
known_obligations:
  - consent
weights:
  reference_resolution: 0.30
  graph_connectivity: 0.20
  definition_coverage: 0.15
  semantic_extraction: 0.20
  structure_quality: 0.15
`

	profile, err := ProfileFromYAML([]byte(yamlInput))
	if err != nil {
		t.Fatalf("ProfileFromYAML failed: %v", err)
	}

	if profile.Name != "Test Profile" {
		t.Errorf("Name = %q, want %q", profile.Name, "Test Profile")
	}

	if profile.ExpectedArticles != 50 {
		t.Errorf("ExpectedArticles = %d, want 50", profile.ExpectedArticles)
	}

	if profile.ExpectedDefinitions != 15 {
		t.Errorf("ExpectedDefinitions = %d, want 15", profile.ExpectedDefinitions)
	}

	if len(profile.KnownRights) != 2 {
		t.Errorf("KnownRights count = %d, want 2", len(profile.KnownRights))
	}

	if profile.Weights.ReferenceResolution != 0.30 {
		t.Errorf("ReferenceResolution = %.2f, want 0.30", profile.Weights.ReferenceResolution)
	}
}

func TestProfileFromYAML_NoRights(t *testing.T) {
	yamlInput := `
name: "Minimal Profile"
description: "No rights or obligations"
expected:
  articles: 10
  definitions: 5
  chapters: 3
weights:
  reference_resolution: 0.25
  graph_connectivity: 0.20
  definition_coverage: 0.20
  semantic_extraction: 0.20
  structure_quality: 0.15
`

	profile, err := ProfileFromYAML([]byte(yamlInput))
	if err != nil {
		t.Fatalf("ProfileFromYAML failed: %v", err)
	}

	// Should have empty slices, not nil
	if profile.KnownRights == nil {
		t.Error("KnownRights should be empty slice, not nil")
	}

	if profile.KnownObligations == nil {
		t.Error("KnownObligations should be empty slice, not nil")
	}
}

func TestProfileFromYAML_Invalid(t *testing.T) {
	invalidYAML := []byte("{{invalid yaml content")

	_, err := ProfileFromYAML(invalidYAML)
	if err == nil {
		t.Error("ProfileFromYAML should fail on invalid YAML")
	}
}

// --- File I/O tests ---

func TestLoadProfileFromFile(t *testing.T) {
	tempDir := t.TempDir()
	profilePath := filepath.Join(tempDir, "test-profile.yaml")

	yamlContent := `
name: "File Test Profile"
description: "Loaded from file"
expected:
  articles: 25
  definitions: 10
  chapters: 5
weights:
  reference_resolution: 0.25
  graph_connectivity: 0.20
  definition_coverage: 0.20
  semantic_extraction: 0.20
  structure_quality: 0.15
`

	if err := os.WriteFile(profilePath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	profile, err := LoadProfileFromFile(profilePath)
	if err != nil {
		t.Fatalf("LoadProfileFromFile failed: %v", err)
	}

	if profile.Name != "File Test Profile" {
		t.Errorf("Name = %q, want %q", profile.Name, "File Test Profile")
	}

	if profile.ExpectedArticles != 25 {
		t.Errorf("ExpectedArticles = %d, want 25", profile.ExpectedArticles)
	}
}

func TestLoadProfileFromFile_NotFound(t *testing.T) {
	_, err := LoadProfileFromFile("/nonexistent/path/profile.yaml")
	if err == nil {
		t.Error("LoadProfileFromFile should fail for nonexistent file")
	}
}

func TestSaveProfileToFile(t *testing.T) {
	tempDir := t.TempDir()
	profilePath := filepath.Join(tempDir, "saved-profile.yaml")

	generator := NewProfileGenerator()
	doc := buildProfileTestDocument(30, 5, 10)
	definitions := buildTestDefinitions(10)
	references := buildTestReferences(50, 5)
	semantics := buildTestSemantics(5, 8)

	suggestion := generator.SuggestProfile(doc, definitions, references, semantics, nil)

	err := SaveProfileToFile(suggestion, profilePath)
	if err != nil {
		t.Fatalf("SaveProfileToFile failed: %v", err)
	}

	// Verify file exists and can be loaded back
	loadedProfile, err := LoadProfileFromFile(profilePath)
	if err != nil {
		t.Fatalf("LoadProfileFromFile failed after save: %v", err)
	}

	if loadedProfile.ExpectedArticles != suggestion.Profile.ExpectedArticles {
		t.Errorf("Round-trip ExpectedArticles = %d, want %d",
			loadedProfile.ExpectedArticles, suggestion.Profile.ExpectedArticles)
	}
}

// --- Document analysis tests ---

func TestAnalyzeDocument(t *testing.T) {
	generator := NewProfileGenerator()
	doc := buildProfileTestDocument(40, 6, 15)
	definitions := buildTestDefinitions(15)
	references := buildTestReferences(80, 10)
	semantics := buildTestSemantics(8, 12)

	documentAnalysis := generator.analyzeDocument(doc, definitions, references, semantics, nil)

	if documentAnalysis.ArticleCount != 40 {
		t.Errorf("ArticleCount = %d, want 40", documentAnalysis.ArticleCount)
	}

	if documentAnalysis.ChapterCount != 6 {
		t.Errorf("ChapterCount = %d, want 6", documentAnalysis.ChapterCount)
	}

	if documentAnalysis.DefinitionCount != 15 {
		t.Errorf("DefinitionCount = %d, want 15", documentAnalysis.DefinitionCount)
	}

	if documentAnalysis.ReferenceCount != 90 { // 80 resolved + 10 external
		t.Errorf("ReferenceCount = %d, want 90", documentAnalysis.ReferenceCount)
	}

	if documentAnalysis.ExternalRefCount != 10 {
		t.Errorf("ExternalRefCount = %d, want 10", documentAnalysis.ExternalRefCount)
	}

	if documentAnalysis.RightsCount != 8 {
		t.Errorf("RightsCount = %d, want 8", documentAnalysis.RightsCount)
	}

	if documentAnalysis.ObligationsCount != 12 {
		t.Errorf("ObligationsCount = %d, want 12", documentAnalysis.ObligationsCount)
	}

	if documentAnalysis.RecitalCount != 2 {
		t.Errorf("RecitalCount = %d, want 2", documentAnalysis.RecitalCount)
	}

	expectedDefinitionDensity := 15.0 / 40.0
	if math.Abs(documentAnalysis.DefinitionDensity-expectedDefinitionDensity) > 0.01 {
		t.Errorf("DefinitionDensity = %.4f, want %.4f", documentAnalysis.DefinitionDensity, expectedDefinitionDensity)
	}

	expectedReferenceDensity := 90.0 / 40.0
	if math.Abs(documentAnalysis.ReferenceDensity-expectedReferenceDensity) > 0.01 {
		t.Errorf("ReferenceDensity = %.4f, want %.4f", documentAnalysis.ReferenceDensity, expectedReferenceDensity)
	}
}

func TestAnalyzeDocument_Empty(t *testing.T) {
	generator := NewProfileGenerator()
	doc := buildProfileTestDocument(0, 0, 0)

	documentAnalysis := generator.analyzeDocument(doc, nil, nil, nil, nil)

	if documentAnalysis.ArticleCount != 0 {
		t.Errorf("ArticleCount = %d, want 0", documentAnalysis.ArticleCount)
	}

	if documentAnalysis.DefinitionDensity != 0 {
		t.Errorf("DefinitionDensity = %.4f, want 0", documentAnalysis.DefinitionDensity)
	}

	if documentAnalysis.ReferenceDensity != 0 {
		t.Errorf("ReferenceDensity = %.4f, want 0", documentAnalysis.ReferenceDensity)
	}
}

func TestSuggestWeights_Normalization(t *testing.T) {
	generator := NewProfileGenerator()

	testCases := []struct {
		name            string
		documentAnalysis *DocumentAnalysis
	}{
		{
			name: "balanced document",
			documentAnalysis: &DocumentAnalysis{
				ArticleCount:      50,
				DefinitionCount:   20,
				ReferenceCount:    100,
				DefinitionDensity: 0.4,
				ReferenceDensity:  2.0,
				RightsCount:       10,
				ObligationsCount:  15,
				NestingDepth:      2,
			},
		},
		{
			name: "no definitions or semantics",
			documentAnalysis: &DocumentAnalysis{
				ArticleCount:      30,
				DefinitionCount:   0,
				ReferenceCount:    50,
				DefinitionDensity: 0,
				ReferenceDensity:  1.67,
				RightsCount:       0,
				ObligationsCount:  0,
				NestingDepth:      2,
			},
		},
		{
			name: "everything zero",
			documentAnalysis: &DocumentAnalysis{},
		},
		{
			name: "heavy everything",
			documentAnalysis: &DocumentAnalysis{
				ArticleCount:      100,
				DefinitionCount:   40,
				ReferenceCount:    500,
				DefinitionDensity: 0.4,
				ReferenceDensity:  5.0,
				RightsCount:       30,
				ObligationsCount:  40,
				NestingDepth:      3,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			suggestedWeights, _ := generator.suggestWeights(testCase.documentAnalysis)

			weightSum := suggestedWeights.ReferenceResolution +
				suggestedWeights.GraphConnectivity +
				suggestedWeights.DefinitionCoverage +
				suggestedWeights.SemanticExtraction +
				suggestedWeights.StructureQuality

			if math.Abs(weightSum-1.0) > 0.02 {
				t.Errorf("Weights sum to %.4f, want 1.0 (ref=%.2f, conn=%.2f, def=%.2f, sem=%.2f, struct=%.2f)",
					weightSum,
					suggestedWeights.ReferenceResolution,
					suggestedWeights.GraphConnectivity,
					suggestedWeights.DefinitionCoverage,
					suggestedWeights.SemanticExtraction,
					suggestedWeights.StructureQuality)
			}

			// All weights should be positive
			if suggestedWeights.ReferenceResolution < 0 ||
				suggestedWeights.GraphConnectivity < 0 ||
				suggestedWeights.DefinitionCoverage < 0 ||
				suggestedWeights.SemanticExtraction < 0 ||
				suggestedWeights.StructureQuality < 0 {
				t.Error("All weights should be non-negative")
			}
		})
	}
}

func TestComputeConfidence(t *testing.T) {
	generator := NewProfileGenerator()

	testCases := []struct {
		name             string
		documentAnalysis *DocumentAnalysis
		minConfidence    float64
		maxConfidence    float64
	}{
		{
			name:             "empty document",
			documentAnalysis: &DocumentAnalysis{},
			minConfidence:    0.0,
			maxConfidence:    0.1,
		},
		{
			name: "articles only",
			documentAnalysis: &DocumentAnalysis{
				ArticleCount: 10,
			},
			minConfidence: 0.20,
			maxConfidence: 0.40,
		},
		{
			name: "complete document",
			documentAnalysis: &DocumentAnalysis{
				ArticleCount:     50,
				ChapterCount:     8,
				DefinitionCount:  20,
				ReferenceCount:   100,
				RightsCount:      10,
				ObligationsCount: 15,
			},
			minConfidence: 0.80,
			maxConfidence: 1.00,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			confidence := generator.computeConfidence(testCase.documentAnalysis)

			if confidence < testCase.minConfidence || confidence > testCase.maxConfidence {
				t.Errorf("Confidence = %.2f, want between %.2f and %.2f",
					confidence, testCase.minConfidence, testCase.maxConfidence)
			}
		})
	}
}

// --- String and JSON output tests ---

func TestProfileSuggestion_String(t *testing.T) {
	generator := NewProfileGenerator()
	doc := buildProfileTestDocument(30, 5, 10)
	definitions := buildTestDefinitions(10)
	references := buildTestReferences(50, 5)
	semantics := buildTestSemantics(5, 8)

	suggestion := generator.SuggestProfile(doc, definitions, references, semantics, nil)
	output := suggestion.String()

	expectedSubstrings := []string{
		"Profile Suggestion",
		"confidence",
		"Expected Structure:",
		"Weights:",
		"Reference Resolution:",
		"Reasoning:",
	}

	for _, expectedSubstring := range expectedSubstrings {
		if !strings.Contains(output, expectedSubstring) {
			t.Errorf("String() missing expected content: %q", expectedSubstring)
		}
	}
}

func TestProfileSuggestion_ToJSON(t *testing.T) {
	generator := NewProfileGenerator()
	doc := buildProfileTestDocument(30, 5, 10)
	definitions := buildTestDefinitions(10)
	references := buildTestReferences(50, 5)
	semantics := buildTestSemantics(5, 8)

	suggestion := generator.SuggestProfile(doc, definitions, references, semantics, nil)

	jsonData, err := suggestion.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	jsonStr := string(jsonData)

	if !strings.Contains(jsonStr, "\"profile\"") {
		t.Error("JSON output missing 'profile' field")
	}

	if !strings.Contains(jsonStr, "\"reasoning\"") {
		t.Error("JSON output missing 'reasoning' field")
	}

	if !strings.Contains(jsonStr, "\"confidence\"") {
		t.Error("JSON output missing 'confidence' field")
	}
}

// --- Helper function tests ---

func TestRoundToDecimals(t *testing.T) {
	testCases := []struct {
		value    float64
		decimals int
		expected float64
	}{
		{0.12345, 2, 0.12},
		{0.12567, 2, 0.13},
		{0.5, 0, 1.0},
		{1.0, 2, 1.0},
		{0.0, 2, 0.0},
	}

	for _, testCase := range testCases {
		result := roundToDecimals(testCase.value, testCase.decimals)
		if math.Abs(result-testCase.expected) > 0.001 {
			t.Errorf("roundToDecimals(%.5f, %d) = %.5f, want %.5f",
				testCase.value, testCase.decimals, result, testCase.expected)
		}
	}
}
