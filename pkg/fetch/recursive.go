package fetch

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/coolbeans/regula/pkg/eurlex"
	"github.com/coolbeans/regula/pkg/store"
)

// URIValidator validates whether a URI exists by performing an HTTP HEAD request.
// This interface allows injection of mock validators for testing.
type URIValidator interface {
	ValidateURI(uri string) (*eurlex.ValidationResult, error)
}

// RecursiveFetcher coordinates breadth-first fetching of external references
// found in a triple store, resolving URNs to fetchable URLs, validating them,
// and adding cross-document triples to the federated graph.
type RecursiveFetcher struct {
	config    FetchConfig
	urnMapper *URNMapper
	validator URIValidator
	cache     *DiskCache
}

// NewRecursiveFetcher creates a new recursive fetcher with the given configuration.
// If config.CacheDir is set, a disk cache is initialized.
func NewRecursiveFetcher(config FetchConfig, validator URIValidator) (*RecursiveFetcher, error) {
	var diskCache *DiskCache
	if config.CacheDir != "" {
		var err error
		diskCache, err = NewDiskCache(config.CacheDir, DefaultCacheTTL)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize disk cache: %w", err)
		}
	}

	return &RecursiveFetcher{
		config:    config,
		urnMapper: NewURNMapper(),
		validator: validator,
		cache:     diskCache,
	}, nil
}

// Fetch performs BFS over external references in the triple store, validates/fetches
// metadata for each mappable reference, and adds cross-document triples to the store.
func (fetcher *RecursiveFetcher) Fetch(
	tripleStore *store.TripleStore,
	sourceDocURI string,
) (*FetchReport, error) {
	return fetcher.execute(tripleStore, sourceDocURI, false)
}

// Plan performs a dry-run: maps URNs and checks the cache, but makes no network calls.
// Returns a report showing what would be fetched.
func (fetcher *RecursiveFetcher) Plan(
	tripleStore *store.TripleStore,
	sourceDocURI string,
) (*FetchReport, error) {
	return fetcher.execute(tripleStore, sourceDocURI, true)
}

// execute is the shared implementation for Fetch and Plan.
func (fetcher *RecursiveFetcher) execute(
	tripleStore *store.TripleStore,
	sourceDocURI string,
	dryRun bool,
) (*FetchReport, error) {
	report := &FetchReport{DryRun: dryRun}

	// Collect all external reference URNs from the triple store.
	externalURNs := fetcher.collectExternalRefs(tripleStore)
	report.TotalReferences = len(externalURNs)

	if len(externalURNs) == 0 {
		return report, nil
	}

	// Map URNs to fetchable URLs (BFS level 1).
	var fetchableRefs []FetchableReference
	seen := make(map[string]bool)

	for _, urn := range externalURNs {
		if seen[urn] {
			continue
		}
		seen[urn] = true

		fetchableURL, err := fetcher.urnMapper.MapURN(urn)
		if err != nil {
			report.SkippedCount++
			report.Results = append(report.Results, FetchResult{
				Reference: FetchableReference{URN: urn, Depth: 1},
				Success:   false,
				Error:     err.Error(),
			})
			continue
		}

		// Check domain whitelist.
		if !fetcher.isDomainAllowed(fetchableURL) {
			report.SkippedCount++
			report.Results = append(report.Results, FetchResult{
				Reference: FetchableReference{URN: urn, URL: fetchableURL, Depth: 1},
				Success:   false,
				Error:     "domain not in allowed list",
			})
			continue
		}

		fetchableRefs = append(fetchableRefs, FetchableReference{
			URN:       urn,
			URL:       fetchableURL,
			SourceURI: sourceDocURI,
			Depth:     1,
		})
	}

	report.MappableCount = len(fetchableRefs)

	// Enforce document limit.
	if len(fetchableRefs) > fetcher.config.MaxDocuments {
		skippedFromLimit := len(fetchableRefs) - fetcher.config.MaxDocuments
		report.SkippedCount += skippedFromLimit
		fetchableRefs = fetchableRefs[:fetcher.config.MaxDocuments]
	}

	// Process each fetchable reference.
	for _, fetchableRef := range fetchableRefs {
		if dryRun {
			// In dry-run mode, check cache only.
			if fetcher.cache != nil {
				if cachedResult, found := fetcher.cache.Get(fetchableRef.URL); found {
					cachedResult.Cached = true
					report.CachedCount++
					report.Results = append(report.Results, cachedResult)
					continue
				}
			}

			report.Results = append(report.Results, FetchResult{
				Reference: fetchableRef,
				Success:   true,
				Error:     "would fetch (dry-run)",
			})
			continue
		}

		// Check disk cache before network call.
		if fetcher.cache != nil {
			if cachedResult, found := fetcher.cache.Get(fetchableRef.URL); found {
				cachedResult.Cached = true
				report.CachedCount++
				report.Results = append(report.Results, cachedResult)

				// Add cross-document triples from cached result.
				if cachedResult.Success {
					triplesAdded := fetcher.addFederationTriples(tripleStore, sourceDocURI, fetchableRef, cachedResult)
					report.TriplesAdded += triplesAdded
				}
				continue
			}
		}

		// Validate via HEAD request.
		fetchResult := fetcher.validateReference(fetchableRef)
		report.Results = append(report.Results, fetchResult)

		if fetchResult.Success {
			report.FetchedCount++

			// Cache successful result.
			if fetcher.cache != nil {
				_ = fetcher.cache.Set(fetchableRef.URL, fetchResult)
			}

			// Add cross-document triples.
			triplesAdded := fetcher.addFederationTriples(tripleStore, sourceDocURI, fetchableRef, fetchResult)
			report.TriplesAdded += triplesAdded
		} else {
			report.FailedCount++

			// Cache failed result too, to avoid repeated failures.
			if fetcher.cache != nil {
				_ = fetcher.cache.Set(fetchableRef.URL, fetchResult)
			}
		}
	}

	return report, nil
}

// collectExternalRefs queries the triple store for all reg:externalRef triples
// and returns the unique object URNs.
func (fetcher *RecursiveFetcher) collectExternalRefs(tripleStore *store.TripleStore) []string {
	externalRefTriples := tripleStore.Find("", store.PropExternalRef, "")

	seen := make(map[string]bool)
	var externalURNs []string

	for _, triple := range externalRefTriples {
		urn := triple.Object
		if !seen[urn] {
			seen[urn] = true
			externalURNs = append(externalURNs, urn)
		}
	}

	return externalURNs
}

// validateReference performs an HTTP HEAD validation of the fetchable URL.
func (fetcher *RecursiveFetcher) validateReference(fetchableRef FetchableReference) FetchResult {
	validationResult, err := fetcher.validator.ValidateURI(fetchableRef.URL)
	if err != nil {
		return FetchResult{
			Reference: fetchableRef,
			Success:   false,
			Error:     err.Error(),
			FetchedAt: time.Now(),
		}
	}

	metadata := map[string]string{
		"uri":         fetchableRef.URL,
		"urn":         fetchableRef.URN,
		"status_code": fmt.Sprintf("%d", validationResult.StatusCode),
	}

	return FetchResult{
		Reference:  fetchableRef,
		Success:    validationResult.Valid,
		StatusCode: validationResult.StatusCode,
		Metadata:   metadata,
		Error:      validationResult.Error,
		FetchedAt:  time.Now(),
	}
}

// addFederationTriples adds cross-document RDF triples linking the source document
// to the fetched external document.
func (fetcher *RecursiveFetcher) addFederationTriples(
	tripleStore *store.TripleStore,
	sourceDocURI string,
	fetchableRef FetchableReference,
	fetchResult FetchResult,
) int {
	externalDocURI := fetchableRef.URN
	triplesAdded := 0

	triplesToAdd := []store.Triple{
		{Subject: externalDocURI, Predicate: "rdf:type", Object: store.ClassExternalDocument},
		{Subject: sourceDocURI, Predicate: store.PropFederatedFrom, Object: externalDocURI},
		{Subject: externalDocURI, Predicate: store.PropExternalDocURI, Object: fetchableRef.URL},
		{Subject: externalDocURI, Predicate: store.PropFetchedAt, Object: fetchResult.FetchedAt.Format(time.RFC3339)},
		{Subject: externalDocURI, Predicate: store.PropFetchDepth, Object: fmt.Sprintf("%d", fetchableRef.Depth)},
	}

	previousCount := tripleStore.Count()
	_ = tripleStore.BulkAdd(triplesToAdd)
	triplesAdded = tripleStore.Count() - previousCount

	return triplesAdded
}

// isDomainAllowed checks if the URL's domain is in the allowed list.
// Returns true if no allowed domains are configured (all domains allowed).
func (fetcher *RecursiveFetcher) isDomainAllowed(fetchableURL string) bool {
	if len(fetcher.config.AllowedDomains) == 0 {
		return true
	}

	parsedURL, err := url.Parse(fetchableURL)
	if err != nil {
		return false
	}

	for _, allowedDomain := range fetcher.config.AllowedDomains {
		if strings.EqualFold(parsedURL.Host, allowedDomain) {
			return true
		}
	}

	return false
}
