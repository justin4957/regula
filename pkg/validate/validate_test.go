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
