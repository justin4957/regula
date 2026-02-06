package draft

import (
	"fmt"
	"strings"

	"github.com/coolbeans/regula/pkg/extract"
	"github.com/coolbeans/regula/pkg/library"
	"github.com/coolbeans/regula/pkg/store"
)

// ScenarioOverlay represents a modified view of a regulation library that
// reflects proposed legislation changes without modifying the base store.
type ScenarioOverlay struct {
	// BaseLibraryPath is the path to the original library
	BaseLibraryPath string

	// OverlayStore is the cloned/modified triple store
	OverlayStore *store.TripleStore

	// AppliedAmendments tracks which amendments were successfully applied
	AppliedAmendments []Amendment

	// SkippedAmendments tracks amendments that couldn't be applied
	SkippedAmendments []SkippedAmendment

	// Stats provides summary statistics
	Stats OverlayStats
}

// SkippedAmendment records an amendment that couldn't be applied and why.
type SkippedAmendment struct {
	Amendment Amendment
	Reason    string
}

// OverlayStats provides summary statistics about the overlay application.
type OverlayStats struct {
	TriplesRemoved int
	TriplesAdded   int
	BaseTriples    int
	OverlayTriples int
}

// ApplyDraftOverlay creates a non-destructive overlay of the base triple store
// with draft amendments applied. The base library remains unchanged.
//
// For repealed sections: removes all triples where subject matches the target URI
// For modified sections: removes old triples for the target, inserts new from draft text
// For added sections: inserts new triples from draft text
func ApplyDraftOverlay(diff *DraftDiff, libraryPath string) (*ScenarioOverlay, error) {
	if diff == nil {
		return nil, fmt.Errorf("diff is nil")
	}

	lib, err := library.Open(libraryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open library: %w", err)
	}

	overlay := &ScenarioOverlay{
		BaseLibraryPath:   libraryPath,
		AppliedAmendments: []Amendment{},
		SkippedAmendments: []SkippedAmendment{},
	}

	// Cache triple stores by document ID, cloned for modification
	clonedStores := make(map[string]*store.TripleStore)

	// Helper to get or clone a triple store for a document
	getOrCloneStore := func(documentID string) (*store.TripleStore, error) {
		if cloned, ok := clonedStores[documentID]; ok {
			return cloned, nil
		}

		baseStore, err := lib.LoadTripleStore(documentID)
		if err != nil {
			return nil, err
		}

		// Clone by getting all triples and bulk adding to new store
		cloned := store.NewTripleStore()
		allTriples := baseStore.All()
		if err := cloned.BulkAdd(allTriples); err != nil {
			return nil, fmt.Errorf("failed to clone store: %w", err)
		}

		overlay.Stats.BaseTriples += len(allTriples)
		clonedStores[documentID] = cloned
		return cloned, nil
	}

	// Process repealed sections
	for _, entry := range diff.Removed {
		cloned, err := getOrCloneStore(entry.TargetDocumentID)
		if err != nil {
			overlay.SkippedAmendments = append(overlay.SkippedAmendments, SkippedAmendment{
				Amendment: entry.Amendment,
				Reason:    fmt.Sprintf("failed to load store: %v", err),
			})
			continue
		}

		removed := applyRepeal(entry.TargetURI, cloned)
		overlay.Stats.TriplesRemoved += removed
		overlay.AppliedAmendments = append(overlay.AppliedAmendments, entry.Amendment)
	}

	// Process modified sections
	for _, entry := range diff.Modified {
		cloned, err := getOrCloneStore(entry.TargetDocumentID)
		if err != nil {
			overlay.SkippedAmendments = append(overlay.SkippedAmendments, SkippedAmendment{
				Amendment: entry.Amendment,
				Reason:    fmt.Sprintf("failed to load store: %v", err),
			})
			continue
		}

		removed, added, applyErr := applyModification(entry, cloned, lib.BaseURI())
		if applyErr != nil {
			overlay.SkippedAmendments = append(overlay.SkippedAmendments, SkippedAmendment{
				Amendment: entry.Amendment,
				Reason:    fmt.Sprintf("failed to apply modification: %v", applyErr),
			})
			continue
		}

		overlay.Stats.TriplesRemoved += removed
		overlay.Stats.TriplesAdded += added
		overlay.AppliedAmendments = append(overlay.AppliedAmendments, entry.Amendment)
	}

	// Process added sections
	for _, entry := range diff.Added {
		cloned, err := getOrCloneStore(entry.TargetDocumentID)
		if err != nil {
			overlay.SkippedAmendments = append(overlay.SkippedAmendments, SkippedAmendment{
				Amendment: entry.Amendment,
				Reason:    fmt.Sprintf("failed to load store: %v", err),
			})
			continue
		}

		added, applyErr := applyAddition(entry, cloned, lib.BaseURI())
		if applyErr != nil {
			overlay.SkippedAmendments = append(overlay.SkippedAmendments, SkippedAmendment{
				Amendment: entry.Amendment,
				Reason:    fmt.Sprintf("failed to apply addition: %v", applyErr),
			})
			continue
		}

		overlay.Stats.TriplesAdded += added
		overlay.AppliedAmendments = append(overlay.AppliedAmendments, entry.Amendment)
	}

	// Process redesignated sections (similar to modifications)
	for _, entry := range diff.Redesignated {
		cloned, err := getOrCloneStore(entry.TargetDocumentID)
		if err != nil {
			overlay.SkippedAmendments = append(overlay.SkippedAmendments, SkippedAmendment{
				Amendment: entry.Amendment,
				Reason:    fmt.Sprintf("failed to load store: %v", err),
			})
			continue
		}

		added, applyErr := applyRedesignation(entry, cloned)
		if applyErr != nil {
			overlay.SkippedAmendments = append(overlay.SkippedAmendments, SkippedAmendment{
				Amendment: entry.Amendment,
				Reason:    fmt.Sprintf("failed to apply redesignation: %v", applyErr),
			})
			continue
		}

		overlay.Stats.TriplesAdded += added
		overlay.AppliedAmendments = append(overlay.AppliedAmendments, entry.Amendment)
	}

	// Merge all cloned stores into a single overlay store
	overlay.OverlayStore = store.NewTripleStore()
	for _, cloned := range clonedStores {
		overlay.OverlayStore.MergeFrom(cloned)
	}
	overlay.Stats.OverlayTriples = overlay.OverlayStore.Count()

	return overlay, nil
}

// applyRepeal removes all triples where the targetURI is the subject.
// Returns the number of triples removed.
func applyRepeal(targetURI string, tripleStore *store.TripleStore) int {
	// Find all triples where target is the subject
	subjectTriples := tripleStore.Find(targetURI, "", "")

	// Also remove nested content (paragraphs, points, etc.)
	nestedURIs := findNestedURIs(targetURI, tripleStore)

	totalRemoved := 0

	// Remove the target's triples
	totalRemoved += tripleStore.Delete(targetURI, "", "")

	// Remove references to the target (where it's the object)
	totalRemoved += tripleStore.Delete("", "", targetURI)

	// Remove nested content
	for _, nestedURI := range nestedURIs {
		totalRemoved += tripleStore.Delete(nestedURI, "", "")
		totalRemoved += tripleStore.Delete("", "", nestedURI)
	}

	return len(subjectTriples) + totalRemoved
}

// findNestedURIs finds all URIs that are contained within the target.
func findNestedURIs(targetURI string, tripleStore *store.TripleStore) []string {
	var nested []string

	// Find direct children via reg:contains
	containsTriples := tripleStore.Find(targetURI, store.PropContains, "")
	for _, triple := range containsTriples {
		nested = append(nested, triple.Object)
		// Recursively find nested content
		nested = append(nested, findNestedURIs(triple.Object, tripleStore)...)
	}

	// Also find entities where reg:partOf points to this target
	// (e.g., obligations, rights that are part of this article)
	partOfTriples := tripleStore.Find("", store.PropPartOf, targetURI)
	for _, triple := range partOfTriples {
		nested = append(nested, triple.Subject)
		// Recursively find nested content
		nested = append(nested, findNestedURIs(triple.Subject, tripleStore)...)
	}

	return nested
}

// applyModification removes old triples for the target and inserts new ones
// from the draft text. Returns (removed, added, error).
func applyModification(entry DiffEntry, tripleStore *store.TripleStore, baseURI string) (int, int, error) {
	// Remove existing triples for the target
	removed := applyRepeal(entry.TargetURI, tripleStore)

	// Ingest new content from the amendment
	added, err := IngestDraftSection(entry.Amendment, tripleStore, baseURI, entry.TargetDocumentID)
	if err != nil {
		return removed, 0, err
	}

	return removed, added, nil
}

// applyAddition inserts new triples from the draft text.
// Returns (added, error).
func applyAddition(entry DiffEntry, tripleStore *store.TripleStore, baseURI string) (int, error) {
	return IngestDraftSection(entry.Amendment, tripleStore, baseURI, entry.TargetDocumentID)
}

// applyRedesignation updates the section number/designation without changing content.
// Returns (added, error).
func applyRedesignation(entry DiffEntry, tripleStore *store.TripleStore) (int, error) {
	// For redesignation, we update the number/label triples
	if entry.Amendment.InsertText == "" {
		return 0, nil
	}

	// Update the number property
	tripleStore.Delete(entry.TargetURI, store.PropNumber, "")
	if err := tripleStore.Add(entry.TargetURI, store.PropNumber, entry.Amendment.InsertText); err != nil {
		return 0, err
	}

	return 1, nil
}

// IngestDraftSection parses the amendment's InsertText as regulation text,
// extracts structure and semantics, and adds resulting triples to the store.
func IngestDraftSection(amendment Amendment, tripleStore *store.TripleStore, baseURI, documentID string) (int, error) {
	if amendment.InsertText == "" {
		return 0, nil
	}

	initialCount := tripleStore.Count()

	// Create a minimal document structure for the amendment text
	text := amendment.InsertText

	// Build the regulation ID from document ID (e.g., "us-usc-title-15" -> "US-USC-TITLE-15")
	regID := strings.ToUpper(documentID)

	// Construct the article URI
	articleURI := buildAmendmentURI(baseURI, regID, amendment)

	// Add basic article triples
	if err := tripleStore.Add(articleURI, store.RDFType, store.ClassArticle); err != nil {
		return 0, err
	}
	if err := tripleStore.Add(articleURI, store.PropNumber, amendment.TargetSection); err != nil {
		return 0, err
	}
	if err := tripleStore.Add(articleURI, store.PropText, text); err != nil {
		return 0, err
	}

	// Add hierarchy triples
	regURI := baseURI + regID
	if err := tripleStore.Add(articleURI, store.PropPartOf, regURI); err != nil {
		return 0, err
	}
	if err := tripleStore.Add(articleURI, store.PropBelongsTo, regURI); err != nil {
		return 0, err
	}
	if err := tripleStore.Add(regURI, store.PropContains, articleURI); err != nil {
		return 0, err
	}
	if err := tripleStore.Add(regURI, store.PropHasArticle, articleURI); err != nil {
		return 0, err
	}

	// Create a minimal Article structure to use with extractors
	article := &extract.Article{
		Number: 0,
		Text:   text,
	}

	// Extract cross-references from the text
	refExtractor := extract.NewReferenceExtractor()
	refs := refExtractor.ExtractFromArticle(article)

	for _, ref := range refs {
		if ref.Type == extract.ReferenceTypeInternal && ref.ArticleNum > 0 {
			targetURI := fmt.Sprintf("%s%s:Art%d", baseURI, regID, ref.ArticleNum)
			if err := tripleStore.Add(articleURI, store.PropReferences, targetURI); err != nil {
				return 0, err
			}
			if err := tripleStore.Add(targetURI, store.PropReferencedBy, articleURI); err != nil {
				return 0, err
			}
		}
	}

	// Extract semantic annotations (obligations, rights)
	semExtractor := extract.NewSemanticExtractor()
	annotations := semExtractor.ExtractFromArticle(article)

	for _, ann := range annotations {
		switch ann.Type {
		case extract.SemanticRight:
			rightURI := fmt.Sprintf("%s%s:Right:%s:%s", baseURI, regID, amendment.TargetSection, ann.RightType)
			if err := tripleStore.Add(rightURI, store.RDFType, store.ClassRight); err != nil {
				return 0, err
			}
			if err := tripleStore.Add(rightURI, "reg:rightType", string(ann.RightType)); err != nil {
				return 0, err
			}
			if err := tripleStore.Add(rightURI, store.PropText, ann.MatchedText); err != nil {
				return 0, err
			}
			if err := tripleStore.Add(rightURI, store.PropPartOf, articleURI); err != nil {
				return 0, err
			}
			if err := tripleStore.Add(articleURI, store.PropGrantsRight, rightURI); err != nil {
				return 0, err
			}

		case extract.SemanticObligation, extract.SemanticProhibition:
			obligURI := fmt.Sprintf("%s%s:Obligation:%s:%s", baseURI, regID, amendment.TargetSection, ann.ObligationType)
			if err := tripleStore.Add(obligURI, store.RDFType, store.ClassObligation); err != nil {
				return 0, err
			}
			if err := tripleStore.Add(obligURI, "reg:obligationType", string(ann.ObligationType)); err != nil {
				return 0, err
			}
			if err := tripleStore.Add(obligURI, store.PropText, ann.MatchedText); err != nil {
				return 0, err
			}
			if err := tripleStore.Add(obligURI, store.PropPartOf, articleURI); err != nil {
				return 0, err
			}
			if err := tripleStore.Add(articleURI, store.PropImposesObligation, obligURI); err != nil {
				return 0, err
			}
		}
	}

	return tripleStore.Count() - initialCount, nil
}

// buildAmendmentURI constructs the URI for an amendment target.
func buildAmendmentURI(baseURI, regID string, amendment Amendment) string {
	if !strings.HasSuffix(baseURI, "/") && !strings.HasSuffix(baseURI, "#") {
		baseURI += "/"
	}

	articleURI := baseURI + regID + ":Art" + amendment.TargetSection
	if amendment.TargetSubsection != "" {
		articleURI += "(" + amendment.TargetSubsection + ")"
	}
	return articleURI
}

// CloneTripleStore creates a deep copy of a triple store.
func CloneTripleStore(source *store.TripleStore) *store.TripleStore {
	if source == nil {
		return nil
	}

	cloned := store.NewTripleStore()
	allTriples := source.All()
	_ = cloned.BulkAdd(allTriples)
	return cloned
}
