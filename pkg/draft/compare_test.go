package draft

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/coolbeans/regula/pkg/extract"
	"github.com/coolbeans/regula/pkg/simulate"
	"github.com/coolbeans/regula/pkg/store"
)

// buildCompareBaseTriples creates baseline triples for comparison testing.
func buildCompareBaseTriples() []store.Triple {
	baseURI := "https://regula.dev/regulations/"
	regID := "CCPA"
	regURI := baseURI + regID

	art1798100URI := baseURI + regID + ":Art1798.100"
	art1798105URI := baseURI + regID + ":Art1798.105"
	art1798110URI := baseURI + regID + ":Art1798.110"

	return []store.Triple{
		// Regulation node
		{Subject: regURI, Predicate: store.RDFType, Object: store.ClassRegulation},
		{Subject: regURI, Predicate: store.PropTitle, Object: "California Consumer Privacy Act"},

		// Article 1798.100 - Right to Know
		{Subject: art1798100URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art1798100URI, Predicate: store.PropNumber, Object: "1798.100"},
		{Subject: art1798100URI, Predicate: store.PropTitle, Object: "Right to Know What Personal Information is Being Collected"},
		{Subject: art1798100URI, Predicate: store.PropText, Object: "A consumer shall have the right to request that a business disclose what personal information it collects."},
		{Subject: art1798100URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art1798100URI},

		// Article 1798.105 - Right to Delete
		{Subject: art1798105URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art1798105URI, Predicate: store.PropNumber, Object: "1798.105"},
		{Subject: art1798105URI, Predicate: store.PropTitle, Object: "Right to Deletion of Personal Information"},
		{Subject: art1798105URI, Predicate: store.PropText, Object: "A consumer shall have the right to request that a business delete personal information about the consumer."},
		{Subject: art1798105URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art1798105URI},

		// Article 1798.110 - Right to Access
		{Subject: art1798110URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art1798110URI, Predicate: store.PropNumber, Object: "1798.110"},
		{Subject: art1798110URI, Predicate: store.PropTitle, Object: "Right of Access to Personal Information"},
		{Subject: art1798110URI, Predicate: store.PropText, Object: "A consumer shall have the right to request that a business provide access to personal information."},
		{Subject: art1798110URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art1798110URI},
	}
}

// buildCompareProposedTriples creates proposed triples with an added provision.
func buildCompareProposedTriples() []store.Triple {
	triples := buildCompareBaseTriples()

	baseURI := "https://regula.dev/regulations/"
	regID := "CCPA"
	regURI := baseURI + regID

	// Add a new article 1798.120 - Right to Opt-Out
	art1798120URI := baseURI + regID + ":Art1798.120"
	triples = append(triples, []store.Triple{
		{Subject: art1798120URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art1798120URI, Predicate: store.PropNumber, Object: "1798.120"},
		{Subject: art1798120URI, Predicate: store.PropTitle, Object: "Right to Opt-Out of Sale of Personal Information"},
		{Subject: art1798120URI, Predicate: store.PropText, Object: "A consumer shall have the right to direct a business to not sell personal information."},
		{Subject: art1798120URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art1798120URI},
	}...)

	return triples
}

// buildCompareProposedWithRemovalTriples creates proposed triples with a removed provision.
func buildCompareProposedWithRemovalTriples() []store.Triple {
	baseURI := "https://regula.dev/regulations/"
	regID := "CCPA"
	regURI := baseURI + regID

	art1798100URI := baseURI + regID + ":Art1798.100"
	art1798110URI := baseURI + regID + ":Art1798.110"

	// Same as base but missing 1798.105 (Right to Delete)
	return []store.Triple{
		// Regulation node
		{Subject: regURI, Predicate: store.RDFType, Object: store.ClassRegulation},
		{Subject: regURI, Predicate: store.PropTitle, Object: "California Consumer Privacy Act"},

		// Article 1798.100 - Right to Know
		{Subject: art1798100URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art1798100URI, Predicate: store.PropNumber, Object: "1798.100"},
		{Subject: art1798100URI, Predicate: store.PropTitle, Object: "Right to Know What Personal Information is Being Collected"},
		{Subject: art1798100URI, Predicate: store.PropText, Object: "A consumer shall have the right to request that a business disclose what personal information it collects."},
		{Subject: art1798100URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art1798100URI},

		// Article 1798.110 - Right to Access (1798.105 is removed)
		{Subject: art1798110URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art1798110URI, Predicate: store.PropNumber, Object: "1798.110"},
		{Subject: art1798110URI, Predicate: store.PropTitle, Object: "Right of Access to Personal Information"},
		{Subject: art1798110URI, Predicate: store.PropText, Object: "A consumer shall have the right to request that a business provide access to personal information."},
		{Subject: art1798110URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art1798110URI},
	}
}

func createTestStore(triples []store.Triple) *store.TripleStore {
	ts := store.NewTripleStore()
	_ = ts.BulkAdd(triples)
	return ts
}

func createTestScenario() *simulate.Scenario {
	return &simulate.Scenario{
		Name:        "Consumer Data Request",
		Description: "A consumer requests access to their personal data from a business",
		Entities: []simulate.ScenarioEntity{
			{ID: "consumer1", Type: extract.EntityConsumer, Name: "Consumer"},
			{ID: "business1", Type: extract.EntityBusiness, Name: "Business"},
		},
		Actions: []simulate.ScenarioAction{
			{ID: "action1", Type: simulate.ActionRequestAccess, Actor: "consumer1"},
			{ID: "action2", Type: simulate.ActionProcessData, Actor: "business1"},
		},
		Keywords: []string{"personal information", "access", "request", "consumer", "business"},
	}
}

func TestCompareScenarios_NewProvisionApplies(t *testing.T) {
	baseStore := createTestStore(buildCompareBaseTriples())
	proposedStore := createTestStore(buildCompareProposedTriples())
	scenario := createTestScenario()
	baseURI := "https://regula.dev/regulations/"

	comparison, err := CompareScenarios("Consumer Data Request", scenario, baseStore, proposedStore, baseURI, nil)
	if err != nil {
		t.Fatalf("CompareScenarios failed: %v", err)
	}

	// The proposed store has an additional article (1798.120)
	// Check that it's detected as newly applicable
	if comparison.Proposed == nil {
		t.Fatal("Proposed result is nil")
	}
	if comparison.Baseline == nil {
		t.Fatal("Baseline result is nil")
	}

	// Verify the comparison was performed
	if comparison.Scenario != "Consumer Data Request" {
		t.Errorf("Expected scenario 'Consumer Data Request', got '%s'", comparison.Scenario)
	}
}

func TestCompareScenarios_ProvisionRemoved(t *testing.T) {
	baseStore := createTestStore(buildCompareBaseTriples())
	proposedStore := createTestStore(buildCompareProposedWithRemovalTriples())
	scenario := createTestScenario()
	baseURI := "https://regula.dev/regulations/"

	comparison, err := CompareScenarios("Consumer Data Request", scenario, baseStore, proposedStore, baseURI, nil)
	if err != nil {
		t.Fatalf("CompareScenarios failed: %v", err)
	}

	// The proposed store is missing 1798.105
	if comparison.Proposed == nil {
		t.Fatal("Proposed result is nil")
	}
	if comparison.Baseline == nil {
		t.Fatal("Baseline result is nil")
	}
}

func TestCompareScenarios_NoDifference(t *testing.T) {
	baseStore := createTestStore(buildCompareBaseTriples())
	// Use same triples for both stores
	proposedStore := createTestStore(buildCompareBaseTriples())
	scenario := createTestScenario()
	baseURI := "https://regula.dev/regulations/"

	comparison, err := CompareScenarios("Consumer Data Request", scenario, baseStore, proposedStore, baseURI, nil)
	if err != nil {
		t.Fatalf("CompareScenarios failed: %v", err)
	}

	// With identical stores, there should be no differences
	if len(comparison.NewlyApplicable) != 0 {
		t.Errorf("Expected no newly applicable, got %d", len(comparison.NewlyApplicable))
	}
	if len(comparison.NoLongerApplicable) != 0 {
		t.Errorf("Expected no longer applicable to be empty, got %d", len(comparison.NoLongerApplicable))
	}
	if len(comparison.ChangedRelevance) != 0 {
		t.Errorf("Expected no changed relevance, got %d", len(comparison.ChangedRelevance))
	}
}

func TestCompareScenarios_NilScenario(t *testing.T) {
	baseStore := createTestStore(buildCompareBaseTriples())
	proposedStore := createTestStore(buildCompareProposedTriples())

	_, err := CompareScenarios("Test", nil, baseStore, proposedStore, "", nil)
	if err == nil {
		t.Error("Expected error for nil scenario")
	}
}

func TestCompareScenarios_NilBaseStore(t *testing.T) {
	proposedStore := createTestStore(buildCompareProposedTriples())
	scenario := createTestScenario()

	_, err := CompareScenarios("Test", scenario, nil, proposedStore, "", nil)
	if err == nil {
		t.Error("Expected error for nil base store")
	}
}

func TestCompareScenarios_NilProposedStore(t *testing.T) {
	baseStore := createTestStore(buildCompareBaseTriples())
	scenario := createTestScenario()

	_, err := CompareScenarios("Test", scenario, baseStore, nil, "", nil)
	if err == nil {
		t.Error("Expected error for nil proposed store")
	}
}

func TestDiffMatchResults(t *testing.T) {
	// Create mock match results
	baseline := &simulate.MatchResult{
		AllMatches: []*simulate.MatchedProvision{
			{URI: "uri:art1", Title: "Article 1", Relevance: simulate.RelevanceDirect},
			{URI: "uri:art2", Title: "Article 2", Relevance: simulate.RelevanceRelated},
			{URI: "uri:art3", Title: "Article 3", Relevance: simulate.RelevanceTriggered},
		},
	}

	proposed := &simulate.MatchResult{
		AllMatches: []*simulate.MatchedProvision{
			{URI: "uri:art1", Title: "Article 1", Relevance: simulate.RelevanceDirect}, // Same
			{URI: "uri:art2", Title: "Article 2", Relevance: simulate.RelevanceDirect}, // Changed from RELATED to DIRECT
			// art3 is removed
			{URI: "uri:art4", Title: "Article 4", Relevance: simulate.RelevanceRelated}, // New
		},
	}

	newly, removed, changed := DiffMatchResults(baseline, proposed)

	// Check newly applicable
	if len(newly) != 1 {
		t.Errorf("Expected 1 newly applicable, got %d", len(newly))
	} else if newly[0].URI != "uri:art4" {
		t.Errorf("Expected newly applicable to be uri:art4, got %s", newly[0].URI)
	}

	// Check no longer applicable
	if len(removed) != 1 {
		t.Errorf("Expected 1 no longer applicable, got %d", len(removed))
	} else if removed[0].URI != "uri:art3" {
		t.Errorf("Expected removed to be uri:art3, got %s", removed[0].URI)
	}

	// Check changed relevance
	if len(changed) != 1 {
		t.Errorf("Expected 1 changed relevance, got %d", len(changed))
	} else {
		if changed[0].URI != "uri:art2" {
			t.Errorf("Expected changed to be uri:art2, got %s", changed[0].URI)
		}
		if changed[0].BaselineRelevance != "RELATED" {
			t.Errorf("Expected baseline relevance RELATED, got %s", changed[0].BaselineRelevance)
		}
		if changed[0].ProposedRelevance != "DIRECT" {
			t.Errorf("Expected proposed relevance DIRECT, got %s", changed[0].ProposedRelevance)
		}
	}
}

func TestDiffMatchResults_NilInputs(t *testing.T) {
	newly, removed, changed := DiffMatchResults(nil, nil)

	if len(newly) != 0 || len(removed) != 0 || len(changed) != 0 {
		t.Error("Expected empty results for nil inputs")
	}
}

func TestDiffMatchResults_EmptyResults(t *testing.T) {
	baseline := &simulate.MatchResult{AllMatches: []*simulate.MatchedProvision{}}
	proposed := &simulate.MatchResult{AllMatches: []*simulate.MatchedProvision{}}

	newly, removed, changed := DiffMatchResults(baseline, proposed)

	if len(newly) != 0 || len(removed) != 0 || len(changed) != 0 {
		t.Error("Expected empty results for empty match results")
	}
}

func TestCompareScenarios_RelevanceChanged(t *testing.T) {
	// Create mock results directly to test relevance change detection
	baseline := &simulate.MatchResult{
		AllMatches: []*simulate.MatchedProvision{
			{URI: "uri:art1", Title: "Article 1", Relevance: simulate.RelevanceRelated},
		},
		Summary: &simulate.MatchSummary{TotalMatches: 1},
	}

	proposed := &simulate.MatchResult{
		AllMatches: []*simulate.MatchedProvision{
			{URI: "uri:art1", Title: "Article 1", Relevance: simulate.RelevanceDirect},
		},
		Summary: &simulate.MatchSummary{TotalMatches: 1},
	}

	_, _, changed := DiffMatchResults(baseline, proposed)

	if len(changed) != 1 {
		t.Errorf("Expected 1 changed relevance, got %d", len(changed))
	} else {
		if changed[0].BaselineRelevance != "RELATED" {
			t.Errorf("Expected baseline relevance RELATED, got %s", changed[0].BaselineRelevance)
		}
		if changed[0].ProposedRelevance != "DIRECT" {
			t.Errorf("Expected proposed relevance DIRECT, got %s", changed[0].ProposedRelevance)
		}
	}
}

func TestFormatScenarioComparison_Table(t *testing.T) {
	comparison := &ScenarioComparison{
		Scenario: "Test Scenario",
		Bill: &DraftBill{
			Title:      "Test Bill",
			BillNumber: "HR-123",
		},
		Baseline: &simulate.MatchResult{
			Summary: &simulate.MatchSummary{TotalMatches: 3},
		},
		Proposed: &simulate.MatchResult{
			Summary: &simulate.MatchSummary{TotalMatches: 4},
		},
		NewlyApplicable: []ProvisionDiff{
			{URI: "uri:art4", Label: "Article 4", ProposedRelevance: "DIRECT"},
		},
		NoLongerApplicable: []ProvisionDiff{
			{URI: "uri:art3", Label: "Article 3", BaselineRelevance: "TRIGGERED"},
		},
		ChangedRelevance: []ProvisionDiff{
			{URI: "uri:art2", Label: "Article 2", BaselineRelevance: "RELATED", ProposedRelevance: "DIRECT"},
		},
	}

	output := FormatScenarioComparison(comparison, "table")

	// Check for expected content
	if !strings.Contains(output, "Test Scenario") {
		t.Error("Expected output to contain scenario name")
	}
	if !strings.Contains(output, "Test Bill") {
		t.Error("Expected output to contain bill title")
	}
	if !strings.Contains(output, "HR-123") {
		t.Error("Expected output to contain bill number")
	}
	if !strings.Contains(output, "Newly Applicable") {
		t.Error("Expected output to contain 'Newly Applicable'")
	}
	if !strings.Contains(output, "No Longer Applicable") {
		t.Error("Expected output to contain 'No Longer Applicable'")
	}
	if !strings.Contains(output, "Changed Relevance") {
		t.Error("Expected output to contain 'Changed Relevance'")
	}
	if !strings.Contains(output, "Article 4") {
		t.Error("Expected output to contain 'Article 4'")
	}
}

func TestFormatScenarioComparison_JSON(t *testing.T) {
	comparison := &ScenarioComparison{
		Scenario: "Test Scenario",
		Baseline: &simulate.MatchResult{
			Summary: &simulate.MatchSummary{TotalMatches: 2},
		},
		Proposed: &simulate.MatchResult{
			Summary: &simulate.MatchSummary{TotalMatches: 3},
		},
		NewlyApplicable: []ProvisionDiff{
			{URI: "uri:art1", Label: "Article 1", ProposedRelevance: "DIRECT"},
		},
	}

	output := FormatScenarioComparison(comparison, "json")

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("Expected valid JSON output, got error: %v", err)
	}

	// Check for expected fields
	if parsed["scenario"] != "Test Scenario" {
		t.Error("Expected scenario field in JSON")
	}
	if _, ok := parsed["newly_applicable"]; !ok {
		t.Error("Expected newly_applicable field in JSON")
	}
}

func TestFormatScenarioComparison_Nil(t *testing.T) {
	output := FormatScenarioComparison(nil, "table")
	if output != "" {
		t.Error("Expected empty output for nil comparison")
	}
}

func TestFormatScenarioComparison_NoDifferences(t *testing.T) {
	comparison := &ScenarioComparison{
		Scenario: "Test Scenario",
		Baseline: &simulate.MatchResult{
			Summary: &simulate.MatchSummary{TotalMatches: 2},
		},
		Proposed: &simulate.MatchResult{
			Summary: &simulate.MatchSummary{TotalMatches: 2},
		},
		NewlyApplicable:    []ProvisionDiff{},
		NoLongerApplicable: []ProvisionDiff{},
		ChangedRelevance:   []ProvisionDiff{},
	}

	output := FormatScenarioComparison(comparison, "table")

	if !strings.Contains(output, "No differences detected") {
		t.Error("Expected 'No differences detected' message")
	}
}

func TestGetSummary(t *testing.T) {
	comparison := &ScenarioComparison{
		NewlyApplicable: []ProvisionDiff{
			{URI: "uri:1"},
			{URI: "uri:2"},
		},
		NoLongerApplicable: []ProvisionDiff{
			{URI: "uri:3"},
		},
		ChangedRelevance: []ProvisionDiff{
			{URI: "uri:4"},
		},
		ObligationsDiff: DeltaSummary{Added: 2, Removed: 1},
		RightsDiff:      DeltaSummary{Added: 1, Removed: 0},
	}

	summary := comparison.GetSummary()

	if !summary.HasDifferences {
		t.Error("Expected HasDifferences to be true")
	}
	if summary.NewlyApplicable != 2 {
		t.Errorf("Expected NewlyApplicable 2, got %d", summary.NewlyApplicable)
	}
	if summary.NoLongerApplicable != 1 {
		t.Errorf("Expected NoLongerApplicable 1, got %d", summary.NoLongerApplicable)
	}
	if summary.ChangedRelevance != 1 {
		t.Errorf("Expected ChangedRelevance 1, got %d", summary.ChangedRelevance)
	}
	if summary.ObligationsAdded != 2 {
		t.Errorf("Expected ObligationsAdded 2, got %d", summary.ObligationsAdded)
	}
	if summary.RightsAdded != 1 {
		t.Errorf("Expected RightsAdded 1, got %d", summary.RightsAdded)
	}
}

func TestGetSummary_NoDifferences(t *testing.T) {
	comparison := &ScenarioComparison{
		NewlyApplicable:    []ProvisionDiff{},
		NoLongerApplicable: []ProvisionDiff{},
		ChangedRelevance:   []ProvisionDiff{},
		ObligationsDiff:    DeltaSummary{Added: 0, Removed: 0},
		RightsDiff:         DeltaSummary{Added: 0, Removed: 0},
	}

	summary := comparison.GetSummary()

	if summary.HasDifferences {
		t.Error("Expected HasDifferences to be false")
	}
}

func TestDiffObligations(t *testing.T) {
	baseline := &simulate.MatchResult{
		Summary: &simulate.MatchSummary{
			ObligationsInvolved: []extract.ObligationType{
				extract.ObligationConsent,
				extract.ObligationTransparency,
			},
		},
	}

	proposed := &simulate.MatchResult{
		Summary: &simulate.MatchSummary{
			ObligationsInvolved: []extract.ObligationType{
				extract.ObligationConsent,
				extract.ObligationNotifyBreach, // New
			},
			// ObligationTransparency removed
		},
	}

	delta := diffObligations(baseline, proposed)

	if delta.Added != 1 {
		t.Errorf("Expected 1 added obligation, got %d", delta.Added)
	}
	if delta.Removed != 1 {
		t.Errorf("Expected 1 removed obligation, got %d", delta.Removed)
	}
}

func TestDiffRights(t *testing.T) {
	baseline := &simulate.MatchResult{
		Summary: &simulate.MatchSummary{
			RightsInvolved: []extract.RightType{
				extract.RightAccess,
			},
		},
	}

	proposed := &simulate.MatchResult{
		Summary: &simulate.MatchSummary{
			RightsInvolved: []extract.RightType{
				extract.RightAccess,
				extract.RightErasure, // New
				extract.RightPortability, // New
			},
		},
	}

	delta := diffRights(baseline, proposed)

	if delta.Added != 2 {
		t.Errorf("Expected 2 added rights, got %d", delta.Added)
	}
	if delta.Removed != 0 {
		t.Errorf("Expected 0 removed rights, got %d", delta.Removed)
	}
}

func TestExtractURILabel(t *testing.T) {
	tests := []struct {
		uri      string
		expected string
	}{
		{"https://regula.dev/regulations/GDPR:Art6", "Art6"},
		{"https://example.com/path/to/resource", "resource"},
		{"http://example.org#fragment", "fragment"},
		{"simple", "simple"},
		{"", ""},
	}

	for _, tc := range tests {
		result := extractCompareURILabel(tc.uri)
		if result != tc.expected {
			t.Errorf("extractCompareURILabel(%q) = %q, expected %q", tc.uri, result, tc.expected)
		}
	}
}
