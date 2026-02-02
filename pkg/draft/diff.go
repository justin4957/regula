package draft

import (
	"fmt"
	"strings"

	"github.com/coolbeans/regula/pkg/library"
	"github.com/coolbeans/regula/pkg/store"
)

// DraftDiff holds the structured diff between a draft bill's amendments and
// existing provisions in the USC knowledge graph. Downstream analysis phases
// consume this to compute impact scores, generate reports, etc.
type DraftDiff struct {
	Bill               *DraftBill  `json:"bill"`
	Added              []DiffEntry `json:"added"`
	Removed            []DiffEntry `json:"removed"`
	Modified           []DiffEntry `json:"modified"`
	Redesignated       []DiffEntry `json:"redesignated"`
	UnresolvedTargets  []string    `json:"unresolved_targets"`
	TriplesInvalidated int         `json:"triples_invalidated"`
}

// DiffEntry represents a single amendment's impact on a provision in the
// knowledge graph. It captures the amendment, the resolved target, existing
// and proposed text, and cross-reference information.
type DiffEntry struct {
	Amendment       Amendment `json:"amendment"`
	TargetURI       string    `json:"target_uri"`
	TargetDocumentID string   `json:"target_document_id"`
	ExistingText    string    `json:"existing_text,omitempty"`
	ProposedText    string    `json:"proposed_text,omitempty"`
	AffectedTriples int       `json:"affected_triples"`
	CrossRefsTo     []string  `json:"cross_refs_to"`
	CrossRefsFrom   []string  `json:"cross_refs_from"`
}

// defaultBaseURI matches the library's default base URI for constructing
// knowledge graph URIs from amendment target references.
const defaultBaseURI = "https://regula.dev/regulations/"

// ComputeDiff analyzes a parsed draft bill against the knowledge graph stored
// in the library at libraryPath. For each amendment in the bill, it resolves
// the target provision, loads the relevant triple store, and classifies the
// change as an addition, removal, modification, or redesignation.
//
// Amendments targeting provisions not found in the knowledge graph are
// collected in UnresolvedTargets rather than causing an error.
func ComputeDiff(bill *DraftBill, libraryPath string) (*DraftDiff, error) {
	if bill == nil {
		return nil, fmt.Errorf("bill is nil")
	}

	lib, err := library.Open(libraryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open library: %w", err)
	}

	diff := &DraftDiff{
		Bill:         bill,
		Added:        []DiffEntry{},
		Removed:      []DiffEntry{},
		Modified:     []DiffEntry{},
		Redesignated: []DiffEntry{},
	}

	// Cache loaded triple stores by document ID to avoid reloading
	tripleStoreCache := make(map[string]*store.TripleStore)

	for _, section := range bill.Sections {
		for _, amendment := range section.Amendments {
			targetURI, documentID, resolveErr := ResolveAmendmentTarget(amendment, lib)
			if resolveErr != nil {
				targetDescription := formatUnresolvedTarget(amendment)
				diff.UnresolvedTargets = append(diff.UnresolvedTargets, targetDescription)
				continue
			}

			// Load or retrieve cached triple store for the target document
			tripleStore, ok := tripleStoreCache[documentID]
			if !ok {
				tripleStore, err = lib.LoadTripleStore(documentID)
				if err != nil {
					targetDescription := formatUnresolvedTarget(amendment)
					diff.UnresolvedTargets = append(diff.UnresolvedTargets, targetDescription)
					continue
				}
				tripleStoreCache[documentID] = tripleStore
			}

			existingText := tripleStore.GetOne(targetURI, store.PropText)
			affectedTripleCount := CountAffectedTriples(targetURI, tripleStore)
			incomingRefs, outgoingRefs := FindCrossReferences(targetURI, tripleStore)

			entry := DiffEntry{
				Amendment:        amendment,
				TargetURI:        targetURI,
				TargetDocumentID: documentID,
				ExistingText:     existingText,
				AffectedTriples:  affectedTripleCount,
				CrossRefsTo:      incomingRefs,
				CrossRefsFrom:    outgoingRefs,
			}

			classifyAndAppendEntry(diff, entry, amendment)
			diff.TriplesInvalidated += affectedTripleCount
		}
	}

	return diff, nil
}

// classifyAndAppendEntry routes a DiffEntry to the correct slice in the
// DraftDiff based on the amendment type, setting ProposedText as appropriate.
func classifyAndAppendEntry(diff *DraftDiff, entry DiffEntry, amendment Amendment) {
	switch amendment.Type {
	case AmendStrikeInsert:
		entry.ProposedText = amendment.InsertText
		diff.Modified = append(diff.Modified, entry)
	case AmendRepeal:
		diff.Removed = append(diff.Removed, entry)
	case AmendAddNewSection, AmendAddAtEnd:
		entry.ProposedText = amendment.InsertText
		diff.Added = append(diff.Added, entry)
	case AmendRedesignate:
		entry.ProposedText = amendment.InsertText
		diff.Redesignated = append(diff.Redesignated, entry)
	case AmendTableOfContents:
		entry.ProposedText = amendment.InsertText
		diff.Modified = append(diff.Modified, entry)
	default:
		entry.ProposedText = amendment.InsertText
		diff.Modified = append(diff.Modified, entry)
	}
}

// ResolveAmendmentTarget maps an amendment's target reference (title and section)
// to a knowledge graph URI and library document ID. It constructs the document
// ID as "us-usc-title-{N}" and the article URI using the library's base URI
// and the uppercased document ID as the regulation identifier.
//
// The lib parameter must be an opened *library.Library. Returns an error if the
// amendment has no target title, no target section, or the document is not found
// in the library.
func ResolveAmendmentTarget(amendment Amendment, lib *library.Library) (targetURI string, documentID string, err error) {
	if amendment.TargetTitle == "" {
		return "", "", fmt.Errorf("amendment has no target title")
	}
	if amendment.TargetSection == "" {
		return "", "", fmt.Errorf("amendment has no target section")
	}

	documentID = buildDocumentID(amendment.TargetTitle)

	// Verify document exists in the library
	documentEntry := lib.GetDocument(documentID)
	if documentEntry == nil {
		return "", "", fmt.Errorf("document %s not found in library", documentID)
	}

	baseURI := lib.BaseURI()
	if baseURI == "" {
		baseURI = defaultBaseURI
	}

	targetURI = buildTargetURI(baseURI, documentID, amendment.TargetSection, amendment.TargetSubsection)
	return targetURI, documentID, nil
}

// CountAffectedTriples counts all triples in the store where the targetURI
// appears as either the subject or the object. This represents the total
// number of facts that would be invalidated or need updating if the target
// provision is amended.
func CountAffectedTriples(targetURI string, tripleStore *store.TripleStore) int {
	// Triples where target is the subject (facts about this provision)
	subjectTriples := tripleStore.Find(targetURI, "", "")
	// Triples where target is the object (facts pointing to this provision)
	objectTriples := tripleStore.Find("", "", targetURI)

	return len(subjectTriples) + len(objectTriples)
}

// FindCrossReferences looks up bidirectional cross-references for a target URI.
// It returns:
//   - incoming: URIs of provisions that reference the target (via reg:references)
//   - outgoing: URIs of provisions that the target references (via reg:references)
func FindCrossReferences(targetURI string, tripleStore *store.TripleStore) (incoming []string, outgoing []string) {
	// Incoming: other provisions that reference this target
	// Find triples where object=targetURI and predicate=reg:references
	incomingTriples := tripleStore.Find("", store.PropReferences, targetURI)
	incoming = make([]string, 0, len(incomingTriples))
	for _, triple := range incomingTriples {
		incoming = append(incoming, triple.Subject)
	}

	// Also check reg:referencedBy where this target is the subject
	referencedByTriples := tripleStore.Find(targetURI, store.PropReferencedBy, "")
	for _, triple := range referencedByTriples {
		incoming = append(incoming, triple.Object)
	}
	incoming = deduplicateStrings(incoming)

	// Outgoing: provisions that this target references
	outgoingTriples := tripleStore.Find(targetURI, store.PropReferences, "")
	outgoing = make([]string, 0, len(outgoingTriples))
	for _, triple := range outgoingTriples {
		outgoing = append(outgoing, triple.Object)
	}

	// Also check reg:referencedBy where this target is the object
	referencedByInverse := tripleStore.Find("", store.PropReferencedBy, targetURI)
	for _, triple := range referencedByInverse {
		outgoing = append(outgoing, triple.Subject)
	}
	outgoing = deduplicateStrings(outgoing)

	return incoming, outgoing
}

// buildDocumentID constructs a library document ID from a USC title number.
// For example, title "15" becomes "us-usc-title-15".
func buildDocumentID(titleNumber string) string {
	return "us-usc-title-" + strings.TrimSpace(titleNumber)
}

// buildTargetURI constructs a knowledge graph URI for a specific provision.
// The regID is the uppercased document ID (matching the ingestion convention).
// The URI format is: {baseURI}{RegID}:Art{section} or
// {baseURI}{RegID}:Art{section}({subsection}) for subsection targets.
func buildTargetURI(baseURI, documentID, section, subsection string) string {
	regID := strings.ToUpper(documentID)

	if !strings.HasSuffix(baseURI, "/") && !strings.HasSuffix(baseURI, "#") {
		baseURI += "/"
	}

	articleURI := baseURI + regID + ":Art" + section
	if subsection != "" {
		articleURI += "(" + subsection + ")"
	}
	return articleURI
}

// formatUnresolvedTarget creates a human-readable description of an amendment
// target that could not be resolved in the knowledge graph.
func formatUnresolvedTarget(amendment Amendment) string {
	description := amendment.TargetTitle + " U.S.C. " + amendment.TargetSection
	if amendment.TargetSubsection != "" {
		description += "(" + amendment.TargetSubsection + ")"
	}
	return description
}

// deduplicateStrings removes duplicate strings from a slice while preserving order.
func deduplicateStrings(input []string) []string {
	seen := make(map[string]bool, len(input))
	deduplicated := make([]string, 0, len(input))
	for _, value := range input {
		if !seen[value] {
			seen[value] = true
			deduplicated = append(deduplicated, value)
		}
	}
	return deduplicated
}
