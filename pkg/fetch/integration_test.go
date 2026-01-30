package fetch

import (
	"testing"

	"github.com/coolbeans/regula/pkg/eurlex"
)

// TestIntegration_FetchRealEURLex validates real EUR-Lex ELI URIs via HEAD requests.
// This test is skipped in short mode since it makes real network calls.
func TestIntegration_FetchRealEURLex(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	eurlexClient := eurlex.NewEURLexClient(eurlex.DefaultConfig())

	cases := []struct {
		name string
		urn  string
	}{
		{
			name: "GDPR Regulation 2016/679",
			urn:  "urn:eu:regulation:2016/679",
		},
		{
			name: "Data Protection Directive 1995/46",
			urn:  "urn:eu:directive:1995/46",
		},
	}

	urnMapper := NewURNMapper()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fetchableURL, err := urnMapper.MapURN(tc.urn)
			if err != nil {
				t.Fatalf("MapURN(%q) failed: %v", tc.urn, err)
			}

			t.Logf("URN %s → URL %s", tc.urn, fetchableURL)

			validationResult, err := eurlexClient.ValidateURI(fetchableURL)
			if err != nil {
				t.Fatalf("ValidateURI failed: %v", err)
			}

			t.Logf("Status: %d, Valid: %v", validationResult.StatusCode, validationResult.Valid)

			if !validationResult.Valid {
				t.Errorf("Expected URI to be valid: %s (status %d)", fetchableURL, validationResult.StatusCode)
			}
		})
	}
}

// TestIntegration_EndToEnd validates the full pipeline: build store with refs,
// map URNs, fetch, and verify federated graph.
func TestIntegration_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	eurlexClient := eurlex.NewEURLexClient(eurlex.DefaultConfig())

	tripleStore := buildTestStore()

	fetchConfig := DefaultFetchConfig()
	fetchConfig.MaxDocuments = 3

	fetcher, err := NewRecursiveFetcher(fetchConfig, eurlexClient)
	if err != nil {
		t.Fatalf("NewRecursiveFetcher failed: %v", err)
	}

	report, err := fetcher.Fetch(tripleStore, "https://regula.dev/regulations/GDPR")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	t.Logf("Report:\n%s", report.String())

	if report.TotalReferences == 0 {
		t.Error("TotalReferences should be > 0")
	}

	if report.MappableCount == 0 {
		t.Error("MappableCount should be > 0")
	}
}
