package simulate

import (
	"os"
	"testing"

	"github.com/coolbeans/regula/pkg/extract"
	"github.com/coolbeans/regula/pkg/store"
)

func TestNewProvisionMatcher(t *testing.T) {
	ts := store.NewTripleStore()
	annotations := []*extract.SemanticAnnotation{}

	matcher := NewProvisionMatcher(ts, "https://regula.dev/regulations/", annotations, nil)

	if matcher == nil {
		t.Fatal("Expected non-nil matcher")
	}
	if matcher.store != ts {
		t.Error("Store not set correctly")
	}
}

func TestScenarioCreation(t *testing.T) {
	scenario := NewScenario("Test Scenario")
	scenario.Description = "Test description"
	scenario.AddEntity(extract.EntityDataSubject, "User")
	scenario.AddEntity(extract.EntityController, "Company")
	scenario.AddAction(ActionWithdrawConsent, "user", "User withdraws consent")

	if scenario.Name != "Test Scenario" {
		t.Errorf("Expected name 'Test Scenario', got '%s'", scenario.Name)
	}
	if len(scenario.Entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(scenario.Entities))
	}
	if len(scenario.Actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(scenario.Actions))
	}
}

func TestConsentWithdrawalScenario(t *testing.T) {
	scenario := ConsentWithdrawalScenario()

	if scenario.Name != "Consent Withdrawal" {
		t.Errorf("Expected name 'Consent Withdrawal', got '%s'", scenario.Name)
	}

	keywords := scenario.GetAllKeywords()
	if len(keywords) == 0 {
		t.Error("Expected keywords in consent withdrawal scenario")
	}

	// Should contain consent-related keywords
	hasConsent := false
	for _, k := range keywords {
		if k == "consent" {
			hasConsent = true
			break
		}
	}
	if !hasConsent {
		t.Error("Expected 'consent' keyword in consent withdrawal scenario")
	}
}

func TestMatchDirectProvisions(t *testing.T) {
	ts := store.NewTripleStore()
	baseURI := "https://regula.dev/regulations/"

	// Set up minimal graph with Art 7 (consent conditions)
	ts.Add(baseURI+"GDPR:Art7", store.RDFType, store.ClassArticle)
	ts.Add(baseURI+"GDPR:Art7", store.PropTitle, "Conditions for consent")

	// Create annotation for consent
	annotations := []*extract.SemanticAnnotation{
		{
			Type:           extract.SemanticRight,
			ArticleNum:     7,
			RightType:      extract.RightWithdrawConsent,
			Beneficiary:    extract.EntityDataSubject,
			Confidence:     1.0,
		},
	}

	matcher := NewProvisionMatcher(ts, baseURI, annotations, nil)
	scenario := ConsentWithdrawalScenario()

	result := matcher.Match(scenario)

	if result.Summary.DirectCount == 0 {
		t.Error("Expected at least one direct match for consent withdrawal")
	}

	// Art 7 should be matched
	found := false
	for _, match := range result.DirectMatches {
		if match.ArticleNum == 7 {
			found = true
			if match.Relevance != RelevanceDirect {
				t.Errorf("Expected DIRECT relevance, got %s", match.Relevance)
			}
		}
	}
	if !found {
		t.Error("Expected Art 7 in direct matches")
	}
}

func TestMatchTriggeredProvisions(t *testing.T) {
	ts := store.NewTripleStore()
	baseURI := "https://regula.dev/regulations/"

	// Set up graph: Art7 referenced by Art17
	ts.Add(baseURI+"GDPR:Art7", store.RDFType, store.ClassArticle)
	ts.Add(baseURI+"GDPR:Art7", store.PropTitle, "Conditions for consent")
	ts.Add(baseURI+"GDPR:Art17", store.RDFType, store.ClassArticle)
	ts.Add(baseURI+"GDPR:Art17", store.PropTitle, "Right to erasure")
	ts.Add(baseURI+"GDPR:Art17", store.PropReferences, baseURI+"GDPR:Art7")

	annotations := []*extract.SemanticAnnotation{
		{
			Type:           extract.SemanticRight,
			ArticleNum:     7,
			RightType:      extract.RightWithdrawConsent,
			Beneficiary:    extract.EntityDataSubject,
			Confidence:     1.0,
		},
	}

	matcher := NewProvisionMatcher(ts, baseURI, annotations, nil)
	scenario := ConsentWithdrawalScenario()

	result := matcher.Match(scenario)

	// Art 17 should be triggered by Art 7
	found := false
	for _, match := range result.AllMatches {
		if match.ArticleNum == 17 {
			found = true
			if match.Relevance != RelevanceTriggered {
				t.Errorf("Expected TRIGGERED relevance for Art 17, got %s", match.Relevance)
			}
		}
	}
	if !found {
		t.Error("Expected Art 17 as triggered match")
	}
}

func TestMatchRelatedProvisions(t *testing.T) {
	ts := store.NewTripleStore()
	baseURI := "https://regula.dev/regulations/"

	// Create a simple document with consent keyword
	doc := &extract.Document{
		Chapters: []*extract.Chapter{
			{
				Number: "I",
				Articles: []*extract.Article{
					{
						Number: 99,
						Title:  "Article about consent management",
						Text:   "This article discusses consent and withdrawal procedures.",
					},
				},
			},
		},
	}

	ts.Add(baseURI+"GDPR:Art99", store.RDFType, store.ClassArticle)
	ts.Add(baseURI+"GDPR:Art99", store.PropTitle, "Article about consent management")

	matcher := NewProvisionMatcher(ts, baseURI, []*extract.SemanticAnnotation{}, doc)
	scenario := ConsentWithdrawalScenario()

	result := matcher.Match(scenario)

	// Art 99 should be related due to keyword match
	found := false
	for _, match := range result.RelatedMatches {
		if match.ArticleNum == 99 {
			found = true
		}
	}
	if !found {
		t.Error("Expected Art 99 in related matches due to keyword")
	}
}

func TestMatchResultString(t *testing.T) {
	result := &MatchResult{
		Scenario: &Scenario{Name: "Test"},
		DirectMatches: []*MatchedProvision{
			{ArticleNum: 7, Title: "Test Article", Score: 0.9, Relevance: RelevanceDirect},
		},
		TriggeredMatches: []*MatchedProvision{},
		RelatedMatches:   []*MatchedProvision{},
		AllMatches: []*MatchedProvision{
			{ArticleNum: 7, Title: "Test Article", Score: 0.9, Relevance: RelevanceDirect},
		},
		Summary: &MatchSummary{
			TotalMatches:   1,
			DirectCount:    1,
			TriggeredCount: 0,
			RelatedCount:   0,
		},
	}

	str := result.String()
	if str == "" {
		t.Error("String() returned empty")
	}
	if !containsString(str, "Test") {
		t.Error("String() missing scenario name")
	}
}

func TestMatchResultJSON(t *testing.T) {
	result := &MatchResult{
		Scenario: &Scenario{Name: "Test"},
		DirectMatches:    []*MatchedProvision{},
		TriggeredMatches: []*MatchedProvision{},
		RelatedMatches:   []*MatchedProvision{},
		AllMatches:       []*MatchedProvision{},
		Summary: &MatchSummary{
			TotalMatches: 0,
		},
	}

	data, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}
	if len(data) == 0 {
		t.Error("ToJSON() returned empty")
	}
}

func TestGDPRConsentWithdrawal(t *testing.T) {
	// Integration test with real GDPR data
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

	baseURI := "https://regula.dev/regulations/"
	ts := store.NewTripleStore()
	builder := store.NewGraphBuilder(ts, baseURI)

	defExtractor := extract.NewDefinitionExtractor()
	refExtractor := extract.NewReferenceExtractor()
	semExtractor := extract.NewSemanticExtractor()
	resolver := extract.NewReferenceResolver(baseURI, "GDPR")
	resolver.IndexDocument(doc)

	_, err = builder.BuildComplete(doc, defExtractor, refExtractor, resolver, semExtractor)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	annotations := semExtractor.ExtractFromDocument(doc)
	matcher := NewProvisionMatcher(ts, baseURI, annotations, doc)

	// Test consent withdrawal scenario
	scenario := ConsentWithdrawalScenario()
	result := matcher.Match(scenario)

	t.Logf("Consent Withdrawal Match Results:")
	t.Logf("  Total matches: %d", result.Summary.TotalMatches)
	t.Logf("  Direct: %d", result.Summary.DirectCount)
	t.Logf("  Triggered: %d", result.Summary.TriggeredCount)
	t.Logf("  Related: %d", result.Summary.RelatedCount)

	// According to issue #15, consent withdrawal should find:
	// - Art 7(3): DIRECT (withdrawal right)
	// - Art 17: TRIGGERED (erasure follows withdrawal)
	// - Art 12: RELATED (response timeline)

	if result.Summary.TotalMatches == 0 {
		t.Error("Expected matches for consent withdrawal scenario")
	}

	// Check for Art 7
	hasArt7 := false
	for _, match := range result.AllMatches {
		if match.ArticleNum == 7 {
			hasArt7 = true
			t.Logf("  Art 7: %s (relevance: %s, score: %.2f)", match.Title, match.Relevance, match.Score)
		}
	}
	if !hasArt7 {
		t.Log("Warning: Art 7 (consent) not found in matches")
	}

	// Log direct matches
	if len(result.DirectMatches) > 0 {
		t.Logf("\nDirect matches:")
		for _, match := range result.DirectMatches {
			t.Logf("  Art %d: %s (score: %.2f)", match.ArticleNum, match.Title, match.Score)
		}
	}

	// Log triggered matches
	if len(result.TriggeredMatches) > 0 {
		t.Logf("\nTriggered matches:")
		for _, match := range result.TriggeredMatches {
			t.Logf("  Art %d: %s (score: %.2f)", match.ArticleNum, match.Title, match.Score)
		}
	}
}

func TestAccessRequestScenario(t *testing.T) {
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

	baseURI := "https://regula.dev/regulations/"
	ts := store.NewTripleStore()
	builder := store.NewGraphBuilder(ts, baseURI)

	defExtractor := extract.NewDefinitionExtractor()
	refExtractor := extract.NewReferenceExtractor()
	semExtractor := extract.NewSemanticExtractor()
	resolver := extract.NewReferenceResolver(baseURI, "GDPR")
	resolver.IndexDocument(doc)

	builder.BuildComplete(doc, defExtractor, refExtractor, resolver, semExtractor)

	annotations := semExtractor.ExtractFromDocument(doc)
	matcher := NewProvisionMatcher(ts, baseURI, annotations, doc)

	scenario := AccessRequestScenario()
	result := matcher.Match(scenario)

	t.Logf("Access Request Match Results:")
	t.Logf("  Total matches: %d", result.Summary.TotalMatches)
	t.Logf("  Direct: %d", result.Summary.DirectCount)

	// Should match Art 15 (right of access)
	hasArt15 := false
	for _, match := range result.DirectMatches {
		if match.ArticleNum == 15 {
			hasArt15 = true
			t.Logf("  Art 15 found: %s (relevance: %s)", match.Title, match.Relevance)
		}
	}
	if !hasArt15 {
		t.Log("Warning: Art 15 (right of access) not found as direct match")
	}
}

func TestPredefinedScenarios(t *testing.T) {
	scenarios := PredefinedScenarios

	if len(scenarios) == 0 {
		t.Error("Expected predefined scenarios")
	}

	for name, scenario := range scenarios {
		if scenario.Name == "" {
			t.Errorf("Scenario %s has empty name", name)
		}
		if len(scenario.Actions) == 0 {
			t.Errorf("Scenario %s has no actions", name)
		}
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
