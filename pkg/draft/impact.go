package draft

import (
	"fmt"
	"sort"

	"github.com/coolbeans/regula/pkg/analysis"
	"github.com/coolbeans/regula/pkg/library"
	"github.com/coolbeans/regula/pkg/store"
)

// DraftImpactResult aggregates the impact analysis across all amendments in a
// draft bill. It combines per-amendment impact results into a deduplicated view
// of which provisions, obligations, and rights are affected by the proposed
// legislation.
type DraftImpactResult struct {
	Bill                    *DraftBill          `json:"bill"`
	Diff                    *DraftDiff          `json:"diff"`
	DirectlyAffected        []AffectedProvision `json:"directly_affected"`
	TransitivelyAffected    []AffectedProvision `json:"transitively_affected"`
	BrokenCrossRefs         []BrokenReference   `json:"broken_cross_refs"`
	ObligationChanges       ObligationDelta     `json:"obligation_changes"`
	RightsChanges           RightsDelta         `json:"rights_changes"`
	TotalProvisionsAffected int                 `json:"total_provisions_affected"`
	MaxDepthReached         int                 `json:"max_depth_reached"`
}

// AffectedProvision represents a provision in the knowledge graph that is
// affected by one or more amendments in the draft bill. Depth 1 indicates a
// direct reference; depth 2+ indicates transitive impact.
type AffectedProvision struct {
	URI        string `json:"uri"`
	Label      string `json:"label"`
	DocumentID string `json:"document_id"`
	Depth      int    `json:"depth"`
	Reason     string `json:"reason"`
}

// ObligationDelta tracks how obligations change as a result of the draft bill's
// amendments. Added obligations come from new sections, removed from repeals,
// and modified from strike-insert amendments.
type ObligationDelta struct {
	Added    []string `json:"added"`
	Removed  []string `json:"removed"`
	Modified []string `json:"modified"`
}

// RightsDelta tracks how rights change as a result of the draft bill's
// amendments. The structure parallels ObligationDelta.
type RightsDelta struct {
	Added    []string `json:"added"`
	Removed  []string `json:"removed"`
	Modified []string `json:"modified"`
}

// AnalyzeDraftImpact runs transitive impact analysis for every amendment in a
// computed diff. For each modified or removed entry, it creates an ImpactAnalyzer
// against the target document's triple store and runs analysis at the specified
// depth. Results are aggregated, deduplicated, and enriched with obligation/rights
// deltas and broken cross-reference detection.
func AnalyzeDraftImpact(diff *DraftDiff, libraryPath string, depth int) (*DraftImpactResult, error) {
	if diff == nil {
		return nil, fmt.Errorf("diff is nil")
	}
	if depth < 1 {
		depth = 1
	}

	lib, err := library.Open(libraryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open library: %w", err)
	}

	result := &DraftImpactResult{
		Bill:                 diff.Bill,
		Diff:                 diff,
		DirectlyAffected:     []AffectedProvision{},
		TransitivelyAffected: []AffectedProvision{},
		BrokenCrossRefs:      []BrokenReference{},
		ObligationChanges: ObligationDelta{
			Added:    []string{},
			Removed:  []string{},
			Modified: []string{},
		},
		RightsChanges: RightsDelta{
			Added:    []string{},
			Removed:  []string{},
			Modified: []string{},
		},
	}

	// Cache triple stores by document ID
	tripleStoreCache := make(map[string]*store.TripleStore)

	// Track seen URIs for deduplication across amendments
	seenDirectURIs := make(map[string]bool)
	seenTransitiveURIs := make(map[string]bool)

	// Analyze modified entries
	for _, entry := range diff.Modified {
		tripleStore, loadErr := loadOrCacheTripleStore(lib, entry.TargetDocumentID, tripleStoreCache)
		if loadErr != nil {
			continue
		}
		analyzeEntry(entry, tripleStore, lib.BaseURI(), depth, "modified", result, seenDirectURIs, seenTransitiveURIs)
		collectObligationsAndRights(entry.TargetURI, tripleStore, &result.ObligationChanges.Modified, &result.RightsChanges.Modified)
	}

	// Analyze removed entries
	for _, entry := range diff.Removed {
		tripleStore, loadErr := loadOrCacheTripleStore(lib, entry.TargetDocumentID, tripleStoreCache)
		if loadErr != nil {
			continue
		}
		analyzeEntry(entry, tripleStore, lib.BaseURI(), depth, "repealed", result, seenDirectURIs, seenTransitiveURIs)
		collectObligationsAndRights(entry.TargetURI, tripleStore, &result.ObligationChanges.Removed, &result.RightsChanges.Removed)
	}

	// Analyze added entries for new obligations/rights
	for _, entry := range diff.Added {
		tripleStore, loadErr := loadOrCacheTripleStore(lib, entry.TargetDocumentID, tripleStoreCache)
		if loadErr != nil {
			continue
		}
		collectObligationsAndRights(entry.TargetURI, tripleStore, &result.ObligationChanges.Added, &result.RightsChanges.Added)
	}

	// Detect broken cross-references across all amendment categories
	brokenRefs, brokenErr := DetectBrokenCrossRefs(diff, libraryPath)
	if brokenErr == nil {
		result.BrokenCrossRefs = brokenRefs
	}

	result.TotalProvisionsAffected = len(result.DirectlyAffected) + len(result.TransitivelyAffected)
	result.MaxDepthReached = computeMaxDepthReached(result)

	return result, nil
}

// AggregateImpactResults combines multiple per-entry ImpactResults into a
// single DraftImpactResult. This is useful when impact results are computed
// individually and need to be merged with deduplication.
func AggregateImpactResults(results []*analysis.ImpactResult) *DraftImpactResult {
	aggregated := &DraftImpactResult{
		DirectlyAffected:     []AffectedProvision{},
		TransitivelyAffected: []AffectedProvision{},
		BrokenCrossRefs:      []BrokenReference{},
		ObligationChanges: ObligationDelta{
			Added:    []string{},
			Removed:  []string{},
			Modified: []string{},
		},
		RightsChanges: RightsDelta{
			Added:    []string{},
			Removed:  []string{},
			Modified: []string{},
		},
	}

	seenDirectURIs := make(map[string]bool)
	seenTransitiveURIs := make(map[string]bool)

	for _, impactResult := range results {
		if impactResult == nil {
			continue
		}

		// Collect direct incoming as directly affected
		for _, node := range impactResult.DirectIncoming {
			if seenDirectURIs[node.URI] {
				continue
			}
			seenDirectURIs[node.URI] = true
			aggregated.DirectlyAffected = append(aggregated.DirectlyAffected, AffectedProvision{
				URI:   node.URI,
				Label: node.Label,
				Depth: node.Depth,
				Reason: fmt.Sprintf("references %s", extractURILabel(impactResult.TargetURI)),
			})
		}

		// Collect transitive nodes
		for _, node := range impactResult.TransitiveNodes {
			if seenTransitiveURIs[node.URI] {
				continue
			}
			seenTransitiveURIs[node.URI] = true
			aggregated.TransitivelyAffected = append(aggregated.TransitivelyAffected, AffectedProvision{
				URI:   node.URI,
				Label: node.Label,
				Depth: node.Depth,
				Reason: fmt.Sprintf("transitively linked via %s", extractURILabel(impactResult.TargetURI)),
			})
		}

		// Track max depth
		if impactResult.Summary != nil && impactResult.Summary.MaxDepthReached > aggregated.MaxDepthReached {
			aggregated.MaxDepthReached = impactResult.Summary.MaxDepthReached
		}
	}

	aggregated.TotalProvisionsAffected = len(aggregated.DirectlyAffected) + len(aggregated.TransitivelyAffected)
	return aggregated
}

// extractURILabel extracts the last segment from a URI for display.
func extractURILabel(uri string) string {
	for i := len(uri) - 1; i >= 0; i-- {
		if uri[i] == ':' || uri[i] == '/' || uri[i] == '#' {
			return uri[i+1:]
		}
	}
	return uri
}

// loadOrCacheTripleStore loads a triple store for a document ID, caching the
// result for reuse across multiple amendments targeting the same document.
func loadOrCacheTripleStore(lib *library.Library, documentID string, cache map[string]*store.TripleStore) (*store.TripleStore, error) {
	if tripleStore, ok := cache[documentID]; ok {
		return tripleStore, nil
	}
	tripleStore, err := lib.LoadTripleStore(documentID)
	if err != nil {
		return nil, err
	}
	cache[documentID] = tripleStore
	return tripleStore, nil
}

// analyzeEntry runs the impact analyzer for a single diff entry and appends
// results to the aggregated DraftImpactResult, respecting deduplication maps.
func analyzeEntry(
	entry DiffEntry,
	tripleStore *store.TripleStore,
	baseURI string,
	depth int,
	changeKind string,
	result *DraftImpactResult,
	seenDirectURIs map[string]bool,
	seenTransitiveURIs map[string]bool,
) {
	analyzer := analysis.NewImpactAnalyzer(tripleStore, baseURI)
	impactResult := analyzer.Analyze(entry.TargetURI, depth, analysis.DirectionBoth)

	targetLabel := extractURILabel(entry.TargetURI)

	// Collect directly affected provisions (incoming references to the target)
	for _, node := range impactResult.DirectIncoming {
		if seenDirectURIs[node.URI] {
			continue
		}
		seenDirectURIs[node.URI] = true
		result.DirectlyAffected = append(result.DirectlyAffected, AffectedProvision{
			URI:        node.URI,
			Label:      node.Label,
			DocumentID: entry.TargetDocumentID,
			Depth:      1,
			Reason:     fmt.Sprintf("references %s %s", changeKind, targetLabel),
		})
	}

	// Collect transitively affected provisions
	for _, node := range impactResult.TransitiveNodes {
		if seenTransitiveURIs[node.URI] {
			continue
		}
		seenTransitiveURIs[node.URI] = true
		result.TransitivelyAffected = append(result.TransitivelyAffected, AffectedProvision{
			URI:        node.URI,
			Label:      node.Label,
			DocumentID: entry.TargetDocumentID,
			Depth:      node.Depth,
			Reason:     fmt.Sprintf("transitively linked via %s %s", changeKind, targetLabel),
		})
	}
}

// collectObligationsAndRights scans the triple store for obligations and rights
// linked to the target URI and appends them to the appropriate delta slices.
func collectObligationsAndRights(targetURI string, tripleStore *store.TripleStore, obligations *[]string, rights *[]string) {
	// Find obligations imposed by this provision
	obligationTriples := tripleStore.Find(targetURI, store.PropImposesObligation, "")
	for _, triple := range obligationTriples {
		*obligations = append(*obligations, triple.Object)
	}

	// Find rights granted by this provision
	rightsTriples := tripleStore.Find(targetURI, store.PropGrantsRight, "")
	for _, triple := range rightsTriples {
		*rights = append(*rights, triple.Object)
	}
}

// computeMaxDepthReached finds the maximum depth across all affected provisions.
func computeMaxDepthReached(result *DraftImpactResult) int {
	maxDepth := 0
	for _, provision := range result.DirectlyAffected {
		if provision.Depth > maxDepth {
			maxDepth = provision.Depth
		}
	}
	for _, provision := range result.TransitivelyAffected {
		if provision.Depth > maxDepth {
			maxDepth = provision.Depth
		}
	}
	return maxDepth
}

// SortByDepth sorts the affected provisions by depth for deterministic output.
func (r *DraftImpactResult) SortByDepth() {
	sortProvisions := func(provisions []AffectedProvision) {
		sort.Slice(provisions, func(i, j int) bool {
			if provisions[i].Depth != provisions[j].Depth {
				return provisions[i].Depth < provisions[j].Depth
			}
			return provisions[i].URI < provisions[j].URI
		})
	}
	sortProvisions(r.DirectlyAffected)
	sortProvisions(r.TransitivelyAffected)
}
