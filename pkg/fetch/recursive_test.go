package fetch

import (
	"fmt"
	"testing"
	"time"

	"github.com/coolbeans/regula/pkg/eurlex"
	"github.com/coolbeans/regula/pkg/store"
)

// mockValidator implements URIValidator for testing.
type mockValidator struct {
	responses map[string]*eurlex.ValidationResult
	callCount int
}

func newMockValidator() *mockValidator {
	return &mockValidator{
		responses: make(map[string]*eurlex.ValidationResult),
	}
}

func (mockVal *mockValidator) addResponse(uri string, valid bool, statusCode int) {
	mockVal.responses[uri] = &eurlex.ValidationResult{
		URI:        uri,
		Valid:      valid,
		StatusCode: statusCode,
		CheckedAt:  time.Now(),
	}
}

func (mockVal *mockValidator) ValidateURI(uri string) (*eurlex.ValidationResult, error) {
	mockVal.callCount++

	if result, found := mockVal.responses[uri]; found {
		return result, nil
	}

	return &eurlex.ValidationResult{
		URI:        uri,
		Valid:      false,
		StatusCode: 404,
		CheckedAt:  time.Now(),
	}, nil
}

// buildTestStore creates a triple store with external references for testing.
func buildTestStore() *store.TripleStore {
	tripleStore := store.NewTripleStore()

	// Add some external references (as the reference resolver would).
	_ = tripleStore.Add("https://regula.dev/regulations/GDPR#Art1", store.PropExternalRef, "urn:eu:directive:1995/46")
	_ = tripleStore.Add("https://regula.dev/regulations/GDPR#Art1", store.PropExternalRef, "urn:eu:regulation:2001/45")
	_ = tripleStore.Add("https://regula.dev/regulations/GDPR#Art94", store.PropExternalRef, "urn:eu:directive:1995/46")
	_ = tripleStore.Add("https://regula.dev/regulations/GDPR#Art3", store.PropExternalRef, "urn:eu:treaty:TFEU")
	_ = tripleStore.Add("https://regula.dev/regulations/GDPR#Art50", store.PropExternalRef, "urn:eu:decision:2010/87")

	return tripleStore
}

func TestRecursiveFetcher_CollectExternalRefs(t *testing.T) {
	tripleStore := buildTestStore()
	validator := newMockValidator()

	fetchConfig := DefaultFetchConfig()
	fetcher, err := NewRecursiveFetcher(fetchConfig, validator)
	if err != nil {
		t.Fatalf("NewRecursiveFetcher failed: %v", err)
	}

	externalURNs := fetcher.collectExternalRefs(tripleStore)

	// Should find 4 unique URNs (directive:1995/46 appears twice but is deduped).
	if len(externalURNs) != 4 {
		t.Errorf("Collected URNs: got %d, want 4", len(externalURNs))
	}

	// Verify all expected URNs are present.
	expectedURNs := map[string]bool{
		"urn:eu:directive:1995/46": false,
		"urn:eu:regulation:2001/45": false,
		"urn:eu:treaty:TFEU":       false,
		"urn:eu:decision:2010/87":  false,
	}

	for _, urn := range externalURNs {
		if _, ok := expectedURNs[urn]; ok {
			expectedURNs[urn] = true
		}
	}

	for urn, found := range expectedURNs {
		if !found {
			t.Errorf("Missing expected URN: %s", urn)
		}
	}
}

func TestRecursiveFetcher_Fetch_Success(t *testing.T) {
	tripleStore := buildTestStore()
	validator := newMockValidator()

	// Set up mock responses for mappable URLs.
	validator.addResponse("http://data.europa.eu/eli/dir/1995/46/oj", true, 200)
	validator.addResponse("http://data.europa.eu/eli/reg/2001/45/oj", true, 200)
	validator.addResponse("http://data.europa.eu/eli/dec/2010/87/oj", true, 200)

	fetchConfig := DefaultFetchConfig()
	fetcher, err := NewRecursiveFetcher(fetchConfig, validator)
	if err != nil {
		t.Fatalf("NewRecursiveFetcher failed: %v", err)
	}

	report, err := fetcher.Fetch(tripleStore, "https://regula.dev/regulations/GDPR")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if report.TotalReferences != 4 {
		t.Errorf("TotalReferences: got %d, want 4", report.TotalReferences)
	}

	// 3 mappable (directive, regulation, decision), 1 skipped (treaty).
	if report.MappableCount != 3 {
		t.Errorf("MappableCount: got %d, want 3", report.MappableCount)
	}

	if report.FetchedCount != 3 {
		t.Errorf("FetchedCount: got %d, want 3", report.FetchedCount)
	}

	// Treaty should be skipped.
	if report.SkippedCount != 1 {
		t.Errorf("SkippedCount: got %d, want 1", report.SkippedCount)
	}

	// Cross-document triples should have been added.
	if report.TriplesAdded == 0 {
		t.Error("TriplesAdded should be > 0")
	}

	// Verify federation triples exist in the store.
	federationTriples := tripleStore.Find("", store.PropFederatedFrom, "")
	if len(federationTriples) == 0 {
		t.Error("No federation triples found in store")
	}

	externalDocTriples := tripleStore.Find("", "rdf:type", store.ClassExternalDocument)
	if len(externalDocTriples) == 0 {
		t.Error("No ExternalDocument type triples found in store")
	}
}

func TestRecursiveFetcher_Fetch_MaxDepth(t *testing.T) {
	tripleStore := buildTestStore()
	validator := newMockValidator()

	validator.addResponse("http://data.europa.eu/eli/dir/1995/46/oj", true, 200)
	validator.addResponse("http://data.europa.eu/eli/reg/2001/45/oj", true, 200)
	validator.addResponse("http://data.europa.eu/eli/dec/2010/87/oj", true, 200)

	// Set depth to 1 (should still fetch since our refs are at depth 1).
	fetchConfig := DefaultFetchConfig()
	fetchConfig.MaxDepth = 1
	fetcher, err := NewRecursiveFetcher(fetchConfig, validator)
	if err != nil {
		t.Fatalf("NewRecursiveFetcher failed: %v", err)
	}

	report, err := fetcher.Fetch(tripleStore, "https://regula.dev/regulations/GDPR")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	// All mappable refs are at depth 1, so they should all be fetched.
	if report.FetchedCount != 3 {
		t.Errorf("FetchedCount: got %d, want 3", report.FetchedCount)
	}
}

func TestRecursiveFetcher_Fetch_MaxDocuments(t *testing.T) {
	tripleStore := buildTestStore()
	validator := newMockValidator()

	validator.addResponse("http://data.europa.eu/eli/dir/1995/46/oj", true, 200)
	validator.addResponse("http://data.europa.eu/eli/reg/2001/45/oj", true, 200)
	validator.addResponse("http://data.europa.eu/eli/dec/2010/87/oj", true, 200)

	// Limit to 2 documents.
	fetchConfig := DefaultFetchConfig()
	fetchConfig.MaxDocuments = 2
	fetcher, err := NewRecursiveFetcher(fetchConfig, validator)
	if err != nil {
		t.Fatalf("NewRecursiveFetcher failed: %v", err)
	}

	report, err := fetcher.Fetch(tripleStore, "https://regula.dev/regulations/GDPR")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	// Only 2 should be fetched, remaining mappable ones skipped.
	totalProcessed := report.FetchedCount + report.FailedCount
	if totalProcessed > 2 {
		t.Errorf("Total processed: got %d, want <= 2", totalProcessed)
	}

	// At least 2 should be skipped (1 treaty + 1 over limit).
	if report.SkippedCount < 2 {
		t.Errorf("SkippedCount: got %d, want >= 2", report.SkippedCount)
	}
}

func TestRecursiveFetcher_Fetch_AllowedDomains(t *testing.T) {
	tripleStore := buildTestStore()
	validator := newMockValidator()

	validator.addResponse("http://data.europa.eu/eli/dir/1995/46/oj", true, 200)

	// Only allow data.europa.eu.
	fetchConfig := DefaultFetchConfig()
	fetchConfig.AllowedDomains = []string{"data.europa.eu"}
	fetcher, err := NewRecursiveFetcher(fetchConfig, validator)
	if err != nil {
		t.Fatalf("NewRecursiveFetcher failed: %v", err)
	}

	report, err := fetcher.Fetch(tripleStore, "https://regula.dev/regulations/GDPR")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	// All EUR-Lex URLs use data.europa.eu, so 3 mappable refs should pass the domain check.
	// The treaty is skipped at the URN mapping stage.
	if report.MappableCount != 3 {
		t.Errorf("MappableCount: got %d, want 3", report.MappableCount)
	}
}

func TestRecursiveFetcher_Fetch_DomainBlocked(t *testing.T) {
	tripleStore := buildTestStore()
	validator := newMockValidator()

	// Only allow a domain that doesn't match any mapped URLs.
	fetchConfig := DefaultFetchConfig()
	fetchConfig.AllowedDomains = []string{"www.legislation.gov.uk"}
	fetcher, err := NewRecursiveFetcher(fetchConfig, validator)
	if err != nil {
		t.Fatalf("NewRecursiveFetcher failed: %v", err)
	}

	report, err := fetcher.Fetch(tripleStore, "https://regula.dev/regulations/GDPR")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	// No EUR-Lex URLs should pass the domain check.
	if report.FetchedCount != 0 {
		t.Errorf("FetchedCount: got %d, want 0", report.FetchedCount)
	}

	// All should be skipped (1 treaty unmappable + 3 domain-blocked).
	if report.SkippedCount != 4 {
		t.Errorf("SkippedCount: got %d, want 4", report.SkippedCount)
	}
}

func TestRecursiveFetcher_Fetch_CacheHit(t *testing.T) {
	tripleStore := buildTestStore()
	validator := newMockValidator()
	validator.addResponse("http://data.europa.eu/eli/dir/1995/46/oj", true, 200)
	validator.addResponse("http://data.europa.eu/eli/reg/2001/45/oj", true, 200)
	validator.addResponse("http://data.europa.eu/eli/dec/2010/87/oj", true, 200)

	cacheDir := t.TempDir()
	fetchConfig := DefaultFetchConfig()
	fetchConfig.CacheDir = cacheDir

	fetcher, err := NewRecursiveFetcher(fetchConfig, validator)
	if err != nil {
		t.Fatalf("NewRecursiveFetcher failed: %v", err)
	}

	// First fetch — should hit the network.
	report1, err := fetcher.Fetch(tripleStore, "https://regula.dev/regulations/GDPR")
	if err != nil {
		t.Fatalf("First Fetch failed: %v", err)
	}

	firstFetchedCount := report1.FetchedCount
	firstValidatorCalls := validator.callCount

	if firstFetchedCount == 0 {
		t.Error("First fetch should have fetched documents from network")
	}

	// Reset store for second fetch (re-add external refs).
	tripleStore2 := buildTestStore()

	// Second fetch — should hit the cache.
	report2, err := fetcher.Fetch(tripleStore2, "https://regula.dev/regulations/GDPR")
	if err != nil {
		t.Fatalf("Second Fetch failed: %v", err)
	}

	if report2.CachedCount != firstFetchedCount {
		t.Errorf("CachedCount: got %d, want %d (same as first fetch)", report2.CachedCount, firstFetchedCount)
	}

	// Validator should not have been called again for cached entries.
	additionalCalls := validator.callCount - firstValidatorCalls
	if additionalCalls != 0 {
		t.Errorf("Validator called %d additional times for cached entries", additionalCalls)
	}
}

func TestRecursiveFetcher_Fetch_FailureGraceful(t *testing.T) {
	tripleStore := buildTestStore()
	validator := newMockValidator()

	// One succeeds, one fails (404).
	validator.addResponse("http://data.europa.eu/eli/dir/1995/46/oj", true, 200)
	validator.addResponse("http://data.europa.eu/eli/reg/2001/45/oj", false, 404)
	validator.addResponse("http://data.europa.eu/eli/dec/2010/87/oj", true, 200)

	fetchConfig := DefaultFetchConfig()
	fetcher, err := NewRecursiveFetcher(fetchConfig, validator)
	if err != nil {
		t.Fatalf("NewRecursiveFetcher failed: %v", err)
	}

	report, err := fetcher.Fetch(tripleStore, "https://regula.dev/regulations/GDPR")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	// Should not return an error even though one validation failed.
	if report.FetchedCount != 2 {
		t.Errorf("FetchedCount: got %d, want 2", report.FetchedCount)
	}
	if report.FailedCount != 1 {
		t.Errorf("FailedCount: got %d, want 1", report.FailedCount)
	}
}

func TestRecursiveFetcher_Plan_DryRun(t *testing.T) {
	tripleStore := buildTestStore()
	validator := newMockValidator()

	fetchConfig := DefaultFetchConfig()
	fetcher, err := NewRecursiveFetcher(fetchConfig, validator)
	if err != nil {
		t.Fatalf("NewRecursiveFetcher failed: %v", err)
	}

	report, err := fetcher.Plan(tripleStore, "https://regula.dev/regulations/GDPR")
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if !report.DryRun {
		t.Error("DryRun: got false, want true")
	}

	// Validator should not have been called in dry-run mode.
	if validator.callCount != 0 {
		t.Errorf("Validator called %d times in dry-run mode, want 0", validator.callCount)
	}

	// Should still report total references and mappable count.
	if report.TotalReferences != 4 {
		t.Errorf("TotalReferences: got %d, want 4", report.TotalReferences)
	}

	if report.MappableCount != 3 {
		t.Errorf("MappableCount: got %d, want 3", report.MappableCount)
	}
}

func TestRecursiveFetcher_FederatedTriples(t *testing.T) {
	tripleStore := store.NewTripleStore()
	_ = tripleStore.Add("https://regula.dev/regulations/GDPR#Art1", store.PropExternalRef, "urn:eu:directive:1995/46")

	validator := newMockValidator()
	validator.addResponse("http://data.europa.eu/eli/dir/1995/46/oj", true, 200)

	fetchConfig := DefaultFetchConfig()
	fetcher, err := NewRecursiveFetcher(fetchConfig, validator)
	if err != nil {
		t.Fatalf("NewRecursiveFetcher failed: %v", err)
	}

	report, err := fetcher.Fetch(tripleStore, "https://regula.dev/regulations/GDPR")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if report.TriplesAdded == 0 {
		t.Error("TriplesAdded should be > 0")
	}

	// Verify specific federation triples.
	externalDocURI := "urn:eu:directive:1995/46"

	// Check rdf:type reg:ExternalDocument
	typeTriples := tripleStore.Find(externalDocURI, "rdf:type", store.ClassExternalDocument)
	if len(typeTriples) != 1 {
		t.Errorf("ExternalDocument type triples: got %d, want 1", len(typeTriples))
	}

	// Check reg:federatedFrom
	fedTriples := tripleStore.Find("https://regula.dev/regulations/GDPR", store.PropFederatedFrom, externalDocURI)
	if len(fedTriples) != 1 {
		t.Errorf("FederatedFrom triples: got %d, want 1", len(fedTriples))
	}

	// Check reg:externalDocURI
	uriTriples := tripleStore.Find(externalDocURI, store.PropExternalDocURI, "http://data.europa.eu/eli/dir/1995/46/oj")
	if len(uriTriples) != 1 {
		t.Errorf("ExternalDocURI triples: got %d, want 1", len(uriTriples))
	}

	// Check reg:fetchDepth
	depthTriples := tripleStore.Find(externalDocURI, store.PropFetchDepth, "1")
	if len(depthTriples) != 1 {
		t.Errorf("FetchDepth triples: got %d, want 1", len(depthTriples))
	}

	// Check reg:fetchedAt exists
	fetchedAtTriples := tripleStore.Find(externalDocURI, store.PropFetchedAt, "")
	if len(fetchedAtTriples) != 1 {
		t.Errorf("FetchedAt triples: got %d, want 1", len(fetchedAtTriples))
	}
}

func TestRecursiveFetcher_EmptyStore(t *testing.T) {
	tripleStore := store.NewTripleStore()
	validator := newMockValidator()

	fetchConfig := DefaultFetchConfig()
	fetcher, err := NewRecursiveFetcher(fetchConfig, validator)
	if err != nil {
		t.Fatalf("NewRecursiveFetcher failed: %v", err)
	}

	report, err := fetcher.Fetch(tripleStore, "https://regula.dev/regulations/GDPR")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if report.TotalReferences != 0 {
		t.Errorf("TotalReferences: got %d, want 0", report.TotalReferences)
	}
	if report.FetchedCount != 0 {
		t.Errorf("FetchedCount: got %d, want 0", report.FetchedCount)
	}
}

func TestFetchReport_String(t *testing.T) {
	report := &FetchReport{
		TotalReferences: 4,
		MappableCount:   3,
		FetchedCount:    2,
		CachedCount:     0,
		FailedCount:     1,
		SkippedCount:    1,
		TriplesAdded:    10,
		Results: []FetchResult{
			{
				Reference: FetchableReference{URN: "urn:eu:directive:1995/46", URL: "http://data.europa.eu/eli/dir/1995/46/oj"},
				Success:   true,
			},
			{
				Reference: FetchableReference{URN: "urn:eu:regulation:2001/45", URL: "http://data.europa.eu/eli/reg/2001/45/oj"},
				Success:   false,
				Error:     "HTTP 404",
			},
		},
	}

	output := report.String()
	if output == "" {
		t.Error("String() returned empty string")
	}

	// Check key elements are present.
	expectedPhrases := []string{
		"External references found: 4",
		"Mappable to URLs:",
		"Successfully fetched:",
		"Failed:",
		"Triples added:",
	}

	for _, phrase := range expectedPhrases {
		if !contains(output, phrase) {
			t.Errorf("String() missing phrase: %q", phrase)
		}
	}
}

func TestFetchReport_String_DryRun(t *testing.T) {
	report := &FetchReport{
		TotalReferences: 3,
		MappableCount:   2,
		SkippedCount:    1,
		DryRun:          true,
	}

	output := report.String()
	if !contains(output, "dry-run") {
		t.Error("String() should contain 'dry-run' for dry-run reports")
	}
}

func TestFetchReport_ToMarkdown(t *testing.T) {
	report := &FetchReport{
		TotalReferences: 2,
		MappableCount:   2,
		FetchedCount:    2,
		TriplesAdded:    10,
		Results: []FetchResult{
			{
				Reference: FetchableReference{URN: "urn:eu:directive:1995/46", URL: "http://data.europa.eu/eli/dir/1995/46/oj"},
				Success:   true,
			},
		},
	}

	markdown := report.ToMarkdown()
	if markdown == "" {
		t.Error("ToMarkdown() returned empty string")
	}

	expectedPhrases := []string{
		"## Fetch Report",
		"| Metric | Count |",
		"| External references |",
		"| URN | URL | Status |",
		"urn:eu:directive:1995/46",
	}

	for _, phrase := range expectedPhrases {
		if !contains(markdown, phrase) {
			t.Errorf("ToMarkdown() missing phrase: %q", phrase)
		}
	}
}

func TestDefaultFetchConfig(t *testing.T) {
	fetchConfig := DefaultFetchConfig()

	if fetchConfig.MaxDepth != DefaultMaxDepth {
		t.Errorf("MaxDepth: got %d, want %d", fetchConfig.MaxDepth, DefaultMaxDepth)
	}
	if fetchConfig.MaxDocuments != DefaultMaxDocuments {
		t.Errorf("MaxDocuments: got %d, want %d", fetchConfig.MaxDocuments, DefaultMaxDocuments)
	}
	if fetchConfig.RateLimit != DefaultFetchRateLimit {
		t.Errorf("RateLimit: got %v, want %v", fetchConfig.RateLimit, DefaultFetchRateLimit)
	}
	if fetchConfig.Timeout != DefaultFetchTimeout {
		t.Errorf("Timeout: got %v, want %v", fetchConfig.Timeout, DefaultFetchTimeout)
	}
	if fetchConfig.DryRun {
		t.Error("DryRun should be false by default")
	}
}

func contains(haystack, needle string) bool {
	return fmt.Sprintf("%s", haystack) != "" && len(haystack) > 0 && len(needle) > 0 && containsSubstring(haystack, needle)
}

func containsSubstring(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
