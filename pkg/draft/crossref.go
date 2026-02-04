package draft

import (
	"fmt"
	"sort"

	"github.com/coolbeans/regula/pkg/library"
	"github.com/coolbeans/regula/pkg/store"
)

// BrokenRefSeverity classifies the severity of a broken cross-reference based
// on the type of amendment that causes it. Error-level breaks indicate the
// target is fully repealed; warnings indicate substantial modification; info
// indicates minor text changes that may not invalidate the reference.
type BrokenRefSeverity int

const (
	// SeverityError indicates the reference target is fully repealed or
	// redesignated to a different identifier — the reference is invalid.
	SeverityError BrokenRefSeverity = iota
	// SeverityWarning indicates the reference target is substantially modified
	// via strike-and-insert — the reference may need review.
	SeverityWarning
	// SeverityInfo indicates the reference target has minor text changes that
	// are unlikely to invalidate the reference (e.g., add-at-end, table of contents).
	SeverityInfo
)

// severityLabels maps severity levels to human-readable strings for display.
var severityLabels = [...]string{
	SeverityError:   "error",
	SeverityWarning: "warning",
	SeverityInfo:    "info",
}

// String returns a human-readable label for the severity level.
func (s BrokenRefSeverity) String() string {
	if int(s) < len(severityLabels) {
		return severityLabels[s]
	}
	return "unknown"
}

// BrokenReference represents a cross-reference that will be invalidated or
// affected by a proposed amendment. It identifies both ends of the reference
// (source and target), the predicate connecting them, the severity of the
// break, and a human-readable reason.
type BrokenReference struct {
	SourceURI        string            `json:"source_uri"`
	SourceLabel      string            `json:"source_label"`
	SourceDocumentID string            `json:"source_document_id"`
	TargetURI        string            `json:"target_uri"`
	TargetLabel      string            `json:"target_label"`
	Severity         BrokenRefSeverity `json:"severity"`
	Predicate        string            `json:"predicate"`
	Reason           string            `json:"reason"`
}

// DetectBrokenCrossRefs analyzes a computed diff against the knowledge graph to
// find cross-references that will be broken by the proposed amendments. It
// examines:
//   - Removed entries: provisions referencing repealed sections (SeverityError)
//   - Modified entries: provisions referencing substantially changed sections (SeverityWarning/Info)
//   - Redesignated entries: provisions referencing old section identifiers (SeverityError)
//
// Results are sorted by severity (errors first, then warnings, then info).
func DetectBrokenCrossRefs(diff *DraftDiff, libraryPath string) ([]BrokenReference, error) {
	if diff == nil {
		return nil, fmt.Errorf("diff is nil")
	}

	lib, err := library.Open(libraryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open library: %w", err)
	}

	var brokenRefs []BrokenReference
	tripleStoreCache := make(map[string]*store.TripleStore)

	// Removed entries: target fully repealed — SeverityError
	for _, entry := range diff.Removed {
		tripleStore, loadErr := loadOrCacheTripleStore(lib, entry.TargetDocumentID, tripleStoreCache)
		if loadErr != nil {
			continue
		}
		refs := findIncomingReferences(entry, tripleStore, SeverityError, "target repealed")
		brokenRefs = append(brokenRefs, refs...)
	}

	// Modified entries: target substantially changed — severity based on amendment type
	for _, entry := range diff.Modified {
		tripleStore, loadErr := loadOrCacheTripleStore(lib, entry.TargetDocumentID, tripleStoreCache)
		if loadErr != nil {
			continue
		}
		severity := ClassifyBreakSeverity(entry.Amendment)
		reasonPrefix := classifyReasonPrefix(entry.Amendment)
		refs := findIncomingReferences(entry, tripleStore, severity, reasonPrefix)
		brokenRefs = append(brokenRefs, refs...)
	}

	// Redesignated entries: old identifier no longer valid — SeverityError
	for _, entry := range diff.Redesignated {
		tripleStore, loadErr := loadOrCacheTripleStore(lib, entry.TargetDocumentID, tripleStoreCache)
		if loadErr != nil {
			continue
		}
		refs := findIncomingReferences(entry, tripleStore, SeverityError, "target redesignated")
		brokenRefs = append(brokenRefs, refs...)
	}

	sortBrokenRefs(brokenRefs)
	return brokenRefs, nil
}

// ClassifyBreakSeverity maps an amendment type to the severity of cross-reference
// breakage it would cause:
//   - Repeal → SeverityError (target completely removed)
//   - Redesignate → SeverityError (target identifier changed)
//   - StrikeInsert → SeverityWarning (target text substantially changed)
//   - AddAtEnd, AddNewSection, TableOfContents → SeverityInfo (minor or additive change)
func ClassifyBreakSeverity(amendment Amendment) BrokenRefSeverity {
	switch amendment.Type {
	case AmendRepeal:
		return SeverityError
	case AmendRedesignate:
		return SeverityError
	case AmendStrikeInsert:
		return SeverityWarning
	case AmendAddAtEnd, AmendAddNewSection, AmendTableOfContents:
		return SeverityInfo
	default:
		return SeverityWarning
	}
}

// findIncomingReferences queries the triple store for all provisions that
// reference the target URI of a diff entry, constructing a BrokenReference for
// each one. It checks both reg:references and reg:referencedBy predicates and
// deduplicates the results.
func findIncomingReferences(entry DiffEntry, tripleStore *store.TripleStore, severity BrokenRefSeverity, reasonPrefix string) []BrokenReference {
	targetLabel := extractURILabel(entry.TargetURI)
	var brokenRefs []BrokenReference
	seenSources := make(map[string]bool)

	// Check direct reg:references triples pointing to target
	incomingTriples := tripleStore.Find("", store.PropReferences, entry.TargetURI)
	for _, triple := range incomingTriples {
		if seenSources[triple.Subject] {
			continue
		}
		seenSources[triple.Subject] = true
		brokenRefs = append(brokenRefs, BrokenReference{
			SourceURI:        triple.Subject,
			SourceLabel:      resolveLabel(triple.Subject, tripleStore),
			SourceDocumentID: entry.TargetDocumentID,
			TargetURI:        entry.TargetURI,
			TargetLabel:      targetLabel,
			Severity:         severity,
			Predicate:        store.PropReferences,
			Reason:           fmt.Sprintf("%s §%s", reasonPrefix, targetLabel),
		})
	}

	// Check inverse reg:referencedBy triples
	referencedByTriples := tripleStore.Find(entry.TargetURI, store.PropReferencedBy, "")
	for _, triple := range referencedByTriples {
		if seenSources[triple.Object] {
			continue
		}
		seenSources[triple.Object] = true
		brokenRefs = append(brokenRefs, BrokenReference{
			SourceURI:        triple.Object,
			SourceLabel:      resolveLabel(triple.Object, tripleStore),
			SourceDocumentID: entry.TargetDocumentID,
			TargetURI:        entry.TargetURI,
			TargetLabel:      targetLabel,
			Severity:         severity,
			Predicate:        store.PropReferences,
			Reason:           fmt.Sprintf("%s §%s", reasonPrefix, targetLabel),
		})
	}

	// Also check typed reference predicates (refersToArticle, etc.)
	typedPredicates := []string{
		store.PropRefersToArticle,
		store.PropRefersToChapter,
		store.PropRefersToParagraph,
		store.PropRefersToPoint,
	}
	for _, predicate := range typedPredicates {
		typedTriples := tripleStore.Find("", predicate, entry.TargetURI)
		for _, triple := range typedTriples {
			if seenSources[triple.Subject] {
				continue
			}
			seenSources[triple.Subject] = true
			brokenRefs = append(brokenRefs, BrokenReference{
				SourceURI:        triple.Subject,
				SourceLabel:      resolveLabel(triple.Subject, tripleStore),
				SourceDocumentID: entry.TargetDocumentID,
				TargetURI:        entry.TargetURI,
				TargetLabel:      targetLabel,
				Severity:         severity,
				Predicate:        predicate,
				Reason:           fmt.Sprintf("%s §%s", reasonPrefix, targetLabel),
			})
		}
	}

	return brokenRefs
}

// resolveLabel retrieves a human-readable label for a URI from the triple store,
// falling back to the URI's last segment if no label is found.
func resolveLabel(uri string, tripleStore *store.TripleStore) string {
	// Try reg:title first
	titleTriples := tripleStore.Find(uri, store.PropTitle, "")
	if len(titleTriples) > 0 {
		return titleTriples[0].Object
	}

	// Try rdfs:label
	labelTriples := tripleStore.Find(uri, store.RDFSLabel, "")
	if len(labelTriples) > 0 {
		return labelTriples[0].Object
	}

	// Fall back to URI label extraction
	return extractURILabel(uri)
}

// classifyReasonPrefix returns a human-readable reason prefix based on the
// amendment type.
func classifyReasonPrefix(amendment Amendment) string {
	switch amendment.Type {
	case AmendRepeal:
		return "target repealed"
	case AmendRedesignate:
		return "target redesignated"
	case AmendStrikeInsert:
		return "target substantially modified"
	case AmendAddAtEnd:
		return "target extended"
	case AmendAddNewSection:
		return "target section added"
	case AmendTableOfContents:
		return "target table of contents updated"
	default:
		return "target modified"
	}
}

// sortBrokenRefs sorts broken references by severity (errors first), then by
// source URI for deterministic output.
func sortBrokenRefs(refs []BrokenReference) {
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Severity != refs[j].Severity {
			return refs[i].Severity < refs[j].Severity
		}
		return refs[i].SourceURI < refs[j].SourceURI
	})
}
