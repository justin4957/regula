package validate

import (
	"os"
	"testing"

	"github.com/coolbeans/regula/pkg/extract"
	"github.com/coolbeans/regula/pkg/store"
)

func TestValidateReferences(t *testing.T) {
	validator := NewValidator(0.80)

	// Create sample resolved references
	resolved := []*extract.ResolvedReference{
		{Status: extract.ResolutionResolved, Confidence: extract.ConfidenceHigh},
		{Status: extract.ResolutionResolved, Confidence: extract.ConfidenceHigh},
		{Status: extract.ResolutionPartial, Confidence: extract.ConfidenceMedium},
		{Status: extract.ResolutionNotFound, Confidence: extract.ConfidenceNone, Reason: "Not found",
			Original: &extract.Reference{SourceArticle: 5, RawText: "Article 999"}},
		{Status: extract.ResolutionExternal, Confidence: extract.ConfidenceHigh},
	}

	result := validator.validateReferences(resolved)

	if result.TotalReferences != 5 {
		t.Errorf("Expected 5 total references, got %d", result.TotalReferences)
	}

	if result.Resolved != 2 {
		t.Errorf("Expected 2 resolved, got %d", result.Resolved)
	}

	if result.External != 1 {
		t.Errorf("Expected 1 external, got %d", result.External)
	}

	// Resolution rate should be 3/4 = 75% (excluding external)
	expectedRate := 0.75 // (2 resolved + 1 partial) / 4 internal
	if result.ResolutionRate != expectedRate {
		t.Errorf("Expected resolution rate %.2f, got %.2f", expectedRate, result.ResolutionRate)
	}
}

func TestValidateDefinitions(t *testing.T) {
	validator := NewValidator(0.80)

	definitions := []*extract.DefinedTerm{
		{Term: "personal data", NormalizedTerm: "personal data"},
		{Term: "controller", NormalizedTerm: "controller"},
		{Term: "processor", NormalizedTerm: "processor"},
		{Term: "unused term", NormalizedTerm: "unused term"},
	}

	usages := []*extract.TermUsage{
		{NormalizedTerm: "personal data", ArticleNum: 1, Count: 2},
		{NormalizedTerm: "personal data", ArticleNum: 5, Count: 1},
		{NormalizedTerm: "controller", ArticleNum: 1, Count: 1},
	}

	result := validator.validateDefinitions(definitions, usages)

	if result.TotalDefinitions != 4 {
		t.Errorf("Expected 4 definitions, got %d", result.TotalDefinitions)
	}

	if result.UsedDefinitions != 2 {
		t.Errorf("Expected 2 used definitions, got %d", result.UsedDefinitions)
	}

	if result.UnusedDefinitions != 2 {
		t.Errorf("Expected 2 unused definitions, got %d", result.UnusedDefinitions)
	}

	// Usage rate should be 2/4 = 50%
	expectedRate := 0.5
	if result.UsageRate != expectedRate {
		t.Errorf("Expected usage rate %.2f, got %.2f", expectedRate, result.UsageRate)
	}

	if result.ArticlesWithTerms != 2 {
		t.Errorf("Expected 2 articles with terms, got %d", result.ArticlesWithTerms)
	}
}

func TestValidateSemantics(t *testing.T) {
	validator := NewValidator(0.80)
	validator.SetRegulationType(RegulationGDPR) // Set GDPR for known rights validation

	annotations := []*extract.SemanticAnnotation{
		{Type: extract.SemanticRight, RightType: extract.RightAccess, ArticleNum: 15},
		{Type: extract.SemanticRight, RightType: extract.RightErasure, ArticleNum: 17},
		{Type: extract.SemanticRight, RightType: extract.RightPortability, ArticleNum: 20},
		{Type: extract.SemanticObligation, ObligationType: extract.ObligationSecure, ArticleNum: 32},
		{Type: extract.SemanticObligation, ObligationType: extract.ObligationNotifyBreach, ArticleNum: 33},
	}

	result := validator.validateSemantics(annotations)

	if result.RightsCount != 3 {
		t.Errorf("Expected 3 rights, got %d", result.RightsCount)
	}

	if result.ObligationsCount != 2 {
		t.Errorf("Expected 2 obligations, got %d", result.ObligationsCount)
	}

	if result.ArticlesWithRights != 3 {
		t.Errorf("Expected 3 articles with rights, got %d", result.ArticlesWithRights)
	}

	// Should find 3 out of 6 known rights
	if result.KnownRightsFound != 3 {
		t.Errorf("Expected 3 known rights found, got %d", result.KnownRightsFound)
	}

	if len(result.MissingRights) != 3 {
		t.Errorf("Expected 3 missing rights, got %d", len(result.MissingRights))
	}
}

func TestValidateSemantics_CCPA(t *testing.T) {
	validator := NewValidator(0.80)
	validator.SetRegulationType(RegulationCCPA) // Set CCPA for known rights validation

	annotations := []*extract.SemanticAnnotation{
		{Type: extract.SemanticRight, RightType: extract.RightToKnow, ArticleNum: 100},
		{Type: extract.SemanticRight, RightType: extract.RightToDelete, ArticleNum: 105},
		{Type: extract.SemanticRight, RightType: extract.RightToOptOut, ArticleNum: 120},
		{Type: extract.SemanticObligation, ObligationType: extract.ObligationNoticeAtCollection, ArticleNum: 100},
		{Type: extract.SemanticObligation, ObligationType: extract.ObligationVerifyRequest, ArticleNum: 130},
	}

	result := validator.validateSemantics(annotations)

	if result.RightsCount != 3 {
		t.Errorf("Expected 3 rights, got %d", result.RightsCount)
	}

	if result.ObligationsCount != 2 {
		t.Errorf("Expected 2 obligations, got %d", result.ObligationsCount)
	}

	if result.ArticlesWithRights != 3 {
		t.Errorf("Expected 3 articles with rights, got %d", result.ArticlesWithRights)
	}

	// Should find 3 out of 5 known CCPA rights
	if result.KnownRightsFound != 3 {
		t.Errorf("Expected 3 known CCPA rights found, got %d", result.KnownRightsFound)
	}

	// CCPA has 5 known rights: RightToKnow, RightToDelete, RightToOptOut, RightToNonDiscrimination, RightToKnowAboutSales
	if result.KnownRightsTotal != 5 {
		t.Errorf("Expected 5 total known CCPA rights, got %d", result.KnownRightsTotal)
	}

	if len(result.MissingRights) != 2 {
		t.Errorf("Expected 2 missing rights (RightToNonDiscrimination, RightToKnowAboutSales), got %d: %v",
			len(result.MissingRights), result.MissingRights)
	}

	if result.RegulationType != string(RegulationCCPA) {
		t.Errorf("Expected regulation type CCPA, got %s", result.RegulationType)
	}
}

func TestValidateConnectivity(t *testing.T) {
	validator := NewValidator(0.80)

	doc := &extract.Document{
		Chapters: []*extract.Chapter{
			{
				Number: "I",
				Articles: []*extract.Article{
					{Number: 1, Title: "Subject matter"},
					{Number: 2, Title: "Scope"},
					{Number: 3, Title: "Definitions"},
					{Number: 4, Title: "Some article"},   // Key article (definitions)
					{Number: 5, Title: "Another article"}, // Orphan - no refs
					{Number: 6, Title: "Connected article"},
				},
			},
		},
	}

	// Create triple store with some references
	// Art 1, 2, 3, 4 are key articles (subject, scope, definitions, etc.)
	// Art 6 has refs so it's connected
	// Art 5 has no refs and is not a key article - it should be orphan
	ts := store.NewTripleStore()
	ts.Add("https://example.com:Art1", store.PropReferences, "https://example.com:Art2")
	ts.Add("https://example.com:Art2", store.PropReferences, "https://example.com:Art3")
	ts.Add("https://example.com:Art6", store.PropReferences, "https://example.com:Art1")

	result := validator.validateConnectivity(doc, ts)

	if result.TotalProvisions != 6 {
		t.Errorf("Expected 6 provisions, got %d", result.TotalProvisions)
	}

	// Art 5 should be orphan (no refs and not a key article)
	if result.OrphanCount != 1 {
		t.Errorf("Expected 1 orphan, got %d", result.OrphanCount)
	}

	if len(result.OrphanArticles) != 1 || result.OrphanArticles[0] != 5 {
		t.Errorf("Expected orphan article 5, got %v", result.OrphanArticles)
	}
}

func TestOverallValidation(t *testing.T) {
	validator := NewValidator(0.80)

	doc := &extract.Document{
		Chapters: []*extract.Chapter{
			{
				Number: "I",
				Articles: []*extract.Article{
					{Number: 1, Title: "Test"},
				},
			},
		},
	}

	resolved := []*extract.ResolvedReference{
		{Status: extract.ResolutionResolved, Confidence: extract.ConfidenceHigh},
		{Status: extract.ResolutionResolved, Confidence: extract.ConfidenceHigh},
	}

	definitions := []*extract.DefinedTerm{
		{Term: "term1", NormalizedTerm: "term1"},
	}

	usages := []*extract.TermUsage{
		{NormalizedTerm: "term1", ArticleNum: 1, Count: 1},
	}

	annotations := []*extract.SemanticAnnotation{
		{Type: extract.SemanticRight, RightType: extract.RightAccess, ArticleNum: 1},
		{Type: extract.SemanticRight, RightType: extract.RightErasure, ArticleNum: 1},
		{Type: extract.SemanticRight, RightType: extract.RightRectification, ArticleNum: 1},
		{Type: extract.SemanticRight, RightType: extract.RightRestriction, ArticleNum: 1},
		{Type: extract.SemanticRight, RightType: extract.RightPortability, ArticleNum: 1},
		{Type: extract.SemanticRight, RightType: extract.RightObject, ArticleNum: 1},
	}

	ts := store.NewTripleStore()

	result := validator.Validate(doc, resolved, definitions, usages, annotations, ts)

	if result.Status != StatusPass {
		t.Errorf("Expected PASS status, got %s", result.Status)
	}

	if result.OverallScore < 0.80 {
		t.Errorf("Expected overall score >= 80%%, got %.1f%%", result.OverallScore*100)
	}
}

func TestGDPRValidation(t *testing.T) {
	// Parse actual GDPR test data
	file, err := os.Open("../../testdata/gdpr.txt")
	if err != nil {
		t.Skipf("Skipping GDPR test: %v", err)
	}
	defer file.Close()

	parser := extract.NewParser()
	doc, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Extract all data
	defExtractor := extract.NewDefinitionExtractor()
	definitions := defExtractor.ExtractDefinitions(doc)

	refExtractor := extract.NewReferenceExtractor()
	refs := refExtractor.ExtractFromDocument(doc)

	resolver := extract.NewReferenceResolver("https://regula.dev/regulations/", "GDPR")
	resolver.IndexDocument(doc)
	resolved := resolver.ResolveAll(refs)

	usageExtractor := extract.NewTermUsageExtractor(definitions)
	usages := usageExtractor.ExtractFromDocument(doc)

	semExtractor := extract.NewSemanticExtractor()
	annotations := semExtractor.ExtractFromDocument(doc)

	// Build graph
	ts := store.NewTripleStore()
	builder := store.NewGraphBuilder(ts, "https://regula.dev/regulations/")
	_, err = builder.BuildComplete(doc, defExtractor, refExtractor, resolver, semExtractor)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	// Validate
	validator := NewValidator(0.80)
	result := validator.Validate(doc, resolved, definitions, usages, annotations, ts)

	t.Logf("Validation Result:\n%s", result.String())

	// Check acceptance criteria
	if result.References.ResolutionRate < 0.85 {
		t.Errorf("Resolution rate %.1f%% is below 85%% target", result.References.ResolutionRate*100)
	}

	if result.Status == StatusFail {
		t.Errorf("Validation failed: %v", result.Issues)
	}

	// Log key metrics
	t.Logf("Reference resolution: %.1f%%", result.References.ResolutionRate*100)
	t.Logf("Graph connectivity: %.1f%%", result.Connectivity.ConnectivityRate*100)
	t.Logf("Definition usage: %.1f%%", result.Definitions.UsageRate*100)
	t.Logf("Known rights found: %d/%d", result.Semantics.KnownRightsFound, result.Semantics.KnownRightsTotal)
	t.Logf("Overall score: %.1f%%", result.OverallScore*100)
}

func TestValidationJSON(t *testing.T) {
	result := &ValidationResult{
		Status:       StatusPass,
		Threshold:    0.80,
		OverallScore: 0.92,
		References: &ReferenceValidation{
			TotalReferences: 100,
			Resolved:        90,
			ResolutionRate:  0.90,
		},
	}

	data, err := result.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize to JSON: %v", err)
	}

	if len(data) == 0 {
		t.Error("JSON output is empty")
	}

	// Verify it contains expected fields
	jsonStr := string(data)
	if !contains(jsonStr, `"status": "PASS"`) {
		t.Error("JSON missing status field")
	}
	if !contains(jsonStr, `"overall_score": 0.92`) {
		t.Error("JSON missing overall_score field")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestValidationProfiles(t *testing.T) {
	// Test that profiles exist for known regulation types
	profiles := []RegulationType{RegulationGDPR, RegulationCCPA, RegulationGeneric}

	for _, regType := range profiles {
		profile, ok := ValidationProfiles[regType]
		if !ok {
			t.Errorf("No profile found for %s", regType)
			continue
		}

		if profile.Name == "" {
			t.Errorf("Profile %s has empty name", regType)
		}

		// Validate weights sum to ~1.0
		weights := profile.Weights
		sum := weights.ReferenceResolution + weights.GraphConnectivity +
			weights.DefinitionCoverage + weights.SemanticExtraction + weights.StructureQuality
		if sum < 0.99 || sum > 1.01 {
			t.Errorf("Profile %s weights sum to %.2f, expected ~1.0", regType, sum)
		}
	}
}

func TestValidatorSetProfile(t *testing.T) {
	validator := NewValidator(0.80)

	// Test default profile is nil
	if validator.GetProfile() != nil {
		t.Error("Expected nil profile initially")
	}

	// Test setting regulation type
	validator.SetRegulationType(RegulationGDPR)
	profile := validator.GetProfile()
	if profile == nil {
		t.Error("Expected profile after setting regulation type")
	}
	if profile.Name != "GDPR" {
		t.Errorf("Expected GDPR profile, got %s", profile.Name)
	}

	// Test that known rights are set correctly
	knownRights := validator.getKnownRights()
	if len(knownRights) != 6 {
		t.Errorf("Expected 6 GDPR rights, got %d", len(knownRights))
	}

	// Test CCPA profile
	validator.SetRegulationType(RegulationCCPA)
	knownRights = validator.getKnownRights()
	if len(knownRights) != 5 {
		t.Errorf("Expected 5 CCPA rights, got %d", len(knownRights))
	}
}

func TestValidateStructure(t *testing.T) {
	validator := NewValidator(0.80)
	validator.SetRegulationType(RegulationGDPR)

	doc := &extract.Document{
		Chapters: []*extract.Chapter{
			{
				Number: "I",
				Articles: []*extract.Article{
					{Number: 1, Title: "Subject matter", Text: "This regulation lays down rules relating to the protection of natural persons with regard to the processing of personal data."},
					{Number: 2, Title: "Scope", Text: "This regulation applies to the processing of personal data wholly or partly by automated means."},
					{Number: 3, Title: "Definitions", Text: "For the purposes of this regulation, the following definitions shall apply throughout this document."},
				},
				Sections: []*extract.Section{
					{
						Title: "Section 1",
						Articles: []*extract.Article{
							{Number: 4, Title: "Principles", Text: "Personal data shall be processed lawfully, fairly and in a transparent manner in relation to the data subject."},
						},
					},
				},
			},
		},
	}

	definitions := []*extract.DefinedTerm{
		{Term: "personal data", NormalizedTerm: "personal data"},
		{Term: "processing", NormalizedTerm: "processing"},
	}

	result := validator.validateStructure(doc, definitions)

	if result.TotalArticles != 4 {
		t.Errorf("Expected 4 articles, got %d", result.TotalArticles)
	}

	if result.TotalChapters != 1 {
		t.Errorf("Expected 1 chapter, got %d", result.TotalChapters)
	}

	if result.TotalSections != 1 {
		t.Errorf("Expected 1 section, got %d", result.TotalSections)
	}

	// With GDPR profile (99 expected articles), completeness should be low
	if result.ArticleCompleteness > 0.1 {
		t.Errorf("Expected low article completeness (4/99), got %.2f", result.ArticleCompleteness)
	}

	// Content rate should be 100% since all articles have text > 50 chars
	if result.ContentRate != 1.0 {
		t.Errorf("Expected 100%% content rate, got %.2f", result.ContentRate)
	}
}

func TestWeightedScoring(t *testing.T) {
	validator := NewValidator(0.80)
	validator.SetRegulationType(RegulationGDPR)

	doc := &extract.Document{
		Identifier: "(EU) 2016/679",
		Chapters: []*extract.Chapter{
			{
				Number: "I",
				Articles: []*extract.Article{
					{Number: 1, Title: "Test", Text: "Test article content"},
				},
			},
		},
	}

	resolved := []*extract.ResolvedReference{
		{Status: extract.ResolutionResolved, Confidence: extract.ConfidenceHigh},
	}

	definitions := []*extract.DefinedTerm{
		{Term: "term1", NormalizedTerm: "term1"},
	}

	usages := []*extract.TermUsage{
		{NormalizedTerm: "term1", ArticleNum: 1, Count: 1},
	}

	annotations := []*extract.SemanticAnnotation{
		{Type: extract.SemanticRight, RightType: extract.RightAccess, ArticleNum: 1},
	}

	ts := store.NewTripleStore()

	result := validator.Validate(doc, resolved, definitions, usages, annotations, ts)

	// Check component scores are populated
	if result.ComponentScores == nil {
		t.Fatal("Expected component scores to be populated")
	}

	// Check weights are from profile
	weights := DefaultWeights()
	if result.ComponentScores.ReferenceWeight != weights.ReferenceResolution {
		t.Errorf("Expected reference weight %.2f, got %.2f",
			weights.ReferenceResolution, result.ComponentScores.ReferenceWeight)
	}

	// Check profile is reported
	if result.ProfileName != "GDPR" {
		t.Errorf("Expected profile name GDPR, got %s", result.ProfileName)
	}

	// Check structure validation is present
	if result.Structure == nil {
		t.Error("Expected structure validation to be present")
	}
}

func TestCCPAValidationWithProfile(t *testing.T) {
	// Parse actual CCPA test data
	file, err := os.Open("../../testdata/ccpa.txt")
	if err != nil {
		t.Skipf("Skipping CCPA test: %v", err)
	}
	defer file.Close()

	parser := extract.NewParser()
	doc, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Extract all data
	defExtractor := extract.NewDefinitionExtractor()
	definitions := defExtractor.ExtractDefinitions(doc)

	refExtractor := extract.NewReferenceExtractor()
	refs := refExtractor.ExtractFromDocument(doc)

	resolver := extract.NewReferenceResolver("https://regula.dev/regulations/", "CCPA")
	resolver.IndexDocument(doc)
	resolved := resolver.ResolveAll(refs)

	usageExtractor := extract.NewTermUsageExtractor(definitions)
	usages := usageExtractor.ExtractFromDocument(doc)

	semExtractor := extract.NewSemanticExtractor()
	annotations := semExtractor.ExtractFromDocument(doc)

	ts := store.NewTripleStore()
	builder := store.NewGraphBuilder(ts, "https://regula.dev/regulations/")
	_, err = builder.BuildComplete(doc, defExtractor, refExtractor, resolver, semExtractor)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	// Validate with CCPA profile
	validator := NewValidator(0.80)
	result := validator.Validate(doc, resolved, definitions, usages, annotations, ts)

	// Check profile was auto-detected
	if result.ProfileName != "CCPA" {
		t.Errorf("Expected CCPA profile, got %s", result.ProfileName)
	}

	// Check structure completeness (should be 100% for CCPA since we have expected counts)
	if result.Structure == nil {
		t.Fatal("Expected structure validation")
	}

	t.Logf("CCPA Validation Results:")
	t.Logf("  Profile: %s", result.ProfileName)
	t.Logf("  Overall Score: %.1f%%", result.OverallScore*100)
	t.Logf("  Status: %s", result.Status)
	t.Logf("  Structure Score: %.1f%%", result.Structure.StructureScore*100)
	t.Logf("  Articles: %d (expected: %d)", result.Structure.TotalArticles, result.Structure.ExpectedArticles)

	// CCPA should pass with the profile-based scoring
	if result.Status != StatusPass {
		t.Errorf("Expected PASS status for CCPA, got %s (score: %.1f%%)",
			result.Status, result.OverallScore*100)
	}
}
