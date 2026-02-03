package draft

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coolbeans/regula/pkg/library"
	"github.com/coolbeans/regula/pkg/store"
)

// testLibrary creates a temporary library seeded with a mock USC title document.
// The triple store contains articles with text, cross-references, and structural
// hierarchy triples for testing diff computation.
func testLibrary(t *testing.T, documentID string, triples []store.Triple) (*library.Library, string) {
	t.Helper()

	libraryPath := t.TempDir()
	baseURI := "https://regula.dev/regulations/"

	lib, err := library.Init(libraryPath, baseURI)
	if err != nil {
		t.Fatalf("failed to init library: %v", err)
	}

	// Build a triple store from the provided triples
	tripleStore := store.NewTripleStore()
	if err := tripleStore.BulkAdd(triples); err != nil {
		t.Fatalf("failed to bulk add triples: %v", err)
	}

	// Serialize and write directly to the library's document storage
	triplesData, err := library.SerializeTripleStore(tripleStore)
	if err != nil {
		t.Fatalf("failed to serialize triples: %v", err)
	}

	// Add document entry via the library's normal flow with minimal source text
	sourceText := []byte("placeholder source text for test")
	_, err = lib.AddDocument(documentID, sourceText, library.AddOptions{
		Name:         documentID,
		Jurisdiction: "US",
		Format:       "us",
		Force:        true,
	})
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	// Overwrite the triples file with our test triples
	entry := lib.GetDocument(documentID)
	if entry == nil {
		t.Fatalf("document entry not found after add")
	}

	triplesPath := filepath.Join(libraryPath, "documents", entry.StorageHash, "triples.json")
	if err := os.WriteFile(triplesPath, triplesData, 0644); err != nil {
		t.Fatalf("failed to write triples: %v", err)
	}

	return lib, libraryPath
}

// buildTitle15Triples creates a representative set of triples for USC Title 15
// with sections 6502 and 6505, including text, type, hierarchy, and cross-references.
func buildTitle15Triples() []store.Triple {
	baseURI := "https://regula.dev/regulations/"
	regID := "US-USC-TITLE-15"
	regURI := baseURI + regID

	art6502URI := baseURI + regID + ":Art6502"
	art6502bURI := baseURI + regID + ":Art6502(b)"
	art6502b1URI := baseURI + regID + ":Art6502(b)(1)"
	art6505URI := baseURI + regID + ":Art6505"
	art6505dURI := baseURI + regID + ":Art6505(d)"
	art6503URI := baseURI + regID + ":Art6503"

	return []store.Triple{
		// Regulation node
		{Subject: regURI, Predicate: store.RDFType, Object: store.ClassRegulation},
		{Subject: regURI, Predicate: store.PropTitle, Object: "Title 15 - Commerce and Trade"},

		// Article 6502 - Regulation of unfair and deceptive acts (COPPA)
		{Subject: art6502URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art6502URI, Predicate: store.PropNumber, Object: "6502"},
		{Subject: art6502URI, Predicate: store.PropTitle, Object: "Regulation of unfair and deceptive acts"},
		{Subject: art6502URI, Predicate: store.PropText, Object: "It shall be unlawful for an operator of a website to collect personal information from a child under 13."},
		{Subject: art6502URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: art6502URI, Predicate: store.PropBelongsTo, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art6502URI},
		{Subject: regURI, Predicate: store.PropHasArticle, Object: art6502URI},

		// Article 6502(b) subsection
		{Subject: art6502bURI, Predicate: store.RDFType, Object: store.ClassParagraph},
		{Subject: art6502bURI, Predicate: store.PropNumber, Object: "b"},
		{Subject: art6502bURI, Predicate: store.PropText, Object: "The operator shall provide notice of its information practices."},
		{Subject: art6502bURI, Predicate: store.PropPartOf, Object: art6502URI},
		{Subject: art6502URI, Predicate: store.PropContains, Object: art6502bURI},

		// Article 6502(b)(1) sub-item
		{Subject: art6502b1URI, Predicate: store.RDFType, Object: store.ClassPoint},
		{Subject: art6502b1URI, Predicate: store.PropNumber, Object: "1"},
		{Subject: art6502b1URI, Predicate: store.PropText, Object: "Children under age 13 are protected."},
		{Subject: art6502b1URI, Predicate: store.PropPartOf, Object: art6502bURI},
		{Subject: art6502bURI, Predicate: store.PropContains, Object: art6502b1URI},

		// Article 6505 - Penalties
		{Subject: art6505URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art6505URI, Predicate: store.PropNumber, Object: "6505"},
		{Subject: art6505URI, Predicate: store.PropTitle, Object: "Civil penalties"},
		{Subject: art6505URI, Predicate: store.PropText, Object: "Any operator who violates this section shall be subject to penalties."},
		{Subject: art6505URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: art6505URI, Predicate: store.PropBelongsTo, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art6505URI},
		{Subject: regURI, Predicate: store.PropHasArticle, Object: art6505URI},

		// Article 6505(d) subsection
		{Subject: art6505dURI, Predicate: store.RDFType, Object: store.ClassParagraph},
		{Subject: art6505dURI, Predicate: store.PropNumber, Object: "d"},
		{Subject: art6505dURI, Predicate: store.PropText, Object: "Maximum penalty of $50,000 per violation."},
		{Subject: art6505dURI, Predicate: store.PropPartOf, Object: art6505URI},
		{Subject: art6505URI, Predicate: store.PropContains, Object: art6505dURI},

		// Article 6503
		{Subject: art6503URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art6503URI, Predicate: store.PropNumber, Object: "6503"},
		{Subject: art6503URI, Predicate: store.PropTitle, Object: "Safe harbors"},
		{Subject: art6503URI, Predicate: store.PropText, Object: "Industry self-regulation as safe harbor."},
		{Subject: art6503URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: art6503URI, Predicate: store.PropBelongsTo, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art6503URI},
		{Subject: regURI, Predicate: store.PropHasArticle, Object: art6503URI},

		// Cross-references between articles
		{Subject: art6505URI, Predicate: store.PropReferences, Object: art6502URI},
		{Subject: art6502URI, Predicate: store.PropReferencedBy, Object: art6505URI},
		{Subject: art6503URI, Predicate: store.PropReferences, Object: art6502URI},
		{Subject: art6502URI, Predicate: store.PropReferencedBy, Object: art6503URI},
		{Subject: art6502URI, Predicate: store.PropReferences, Object: art6503URI},
		{Subject: art6503URI, Predicate: store.PropReferencedBy, Object: art6502URI},
	}
}

func TestComputeDiff_SingleModification(t *testing.T) {
	triples := buildTitle15Triples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	bill := &DraftBill{
		BillNumber: "H.R. 9999",
		Title:      "Test Amendment Act",
		Sections: []*DraftSection{
			{
				Number: "1",
				Title:  "Amendment to section 6505",
				Amendments: []Amendment{
					{
						Type:          AmendStrikeInsert,
						TargetTitle:   "15",
						TargetSection: "6505",
						TargetSubsection: "d",
						StrikeText:    "$50,000",
						InsertText:    "$100,000",
						Description:   `by striking "$50,000" and inserting "$100,000"`,
					},
				},
			},
		},
	}

	diff, err := ComputeDiff(bill, libraryPath)
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	if len(diff.Modified) != 1 {
		t.Fatalf("expected 1 modified entry, got %d", len(diff.Modified))
	}
	if len(diff.Added) != 0 {
		t.Errorf("expected 0 added entries, got %d", len(diff.Added))
	}
	if len(diff.Removed) != 0 {
		t.Errorf("expected 0 removed entries, got %d", len(diff.Removed))
	}
	if len(diff.UnresolvedTargets) != 0 {
		t.Errorf("expected 0 unresolved targets, got %d: %v", len(diff.UnresolvedTargets), diff.UnresolvedTargets)
	}

	entry := diff.Modified[0]
	expectedURI := "https://regula.dev/regulations/US-USC-TITLE-15:Art6505(d)"
	if entry.TargetURI != expectedURI {
		t.Errorf("expected target URI %q, got %q", expectedURI, entry.TargetURI)
	}
	if entry.TargetDocumentID != "us-usc-title-15" {
		t.Errorf("expected document ID %q, got %q", "us-usc-title-15", entry.TargetDocumentID)
	}
	if entry.ExistingText != "Maximum penalty of $50,000 per violation." {
		t.Errorf("expected existing text, got %q", entry.ExistingText)
	}
	if entry.ProposedText != "$100,000" {
		t.Errorf("expected proposed text %q, got %q", "$100,000", entry.ProposedText)
	}
	if entry.AffectedTriples == 0 {
		t.Error("expected non-zero affected triples")
	}
	if diff.TriplesInvalidated == 0 {
		t.Error("expected non-zero triples invalidated")
	}
}

func TestComputeDiff_Repeal(t *testing.T) {
	triples := buildTitle15Triples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	bill := &DraftBill{
		BillNumber: "H.R. 8888",
		Title:      "Repeal Test Act",
		Sections: []*DraftSection{
			{
				Number: "1",
				Title:  "Repeal of section 6503",
				Amendments: []Amendment{
					{
						Type:          AmendRepeal,
						TargetTitle:   "15",
						TargetSection: "6503",
						Description:   "Section 6503 of title 15 is repealed",
					},
				},
			},
		},
	}

	diff, err := ComputeDiff(bill, libraryPath)
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	if len(diff.Removed) != 1 {
		t.Fatalf("expected 1 removed entry, got %d", len(diff.Removed))
	}
	if len(diff.Modified) != 0 {
		t.Errorf("expected 0 modified entries, got %d", len(diff.Modified))
	}

	entry := diff.Removed[0]
	expectedURI := "https://regula.dev/regulations/US-USC-TITLE-15:Art6503"
	if entry.TargetURI != expectedURI {
		t.Errorf("expected target URI %q, got %q", expectedURI, entry.TargetURI)
	}
	if entry.ExistingText != "Industry self-regulation as safe harbor." {
		t.Errorf("expected existing text, got %q", entry.ExistingText)
	}

	// Section 6503 is referenced by 6502 and references 6502
	if len(entry.CrossRefsTo) == 0 {
		t.Error("expected incoming cross-references to repealed section")
	}
}

func TestComputeDiff_AddNewSection(t *testing.T) {
	triples := buildTitle15Triples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	bill := &DraftBill{
		BillNumber: "H.R. 7777",
		Title:      "New Section Act",
		Sections: []*DraftSection{
			{
				Number: "1",
				Title:  "New section 6510",
				Amendments: []Amendment{
					{
						Type:          AmendAddNewSection,
						TargetTitle:   "15",
						TargetSection: "6510",
						InsertText:    "The Commission shall establish a youth privacy program.",
						Description:   "inserting after section 6509 the following new section",
					},
				},
			},
		},
	}

	diff, err := ComputeDiff(bill, libraryPath)
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	if len(diff.Added) != 1 {
		t.Fatalf("expected 1 added entry, got %d", len(diff.Added))
	}

	entry := diff.Added[0]
	expectedURI := "https://regula.dev/regulations/US-USC-TITLE-15:Art6510"
	if entry.TargetURI != expectedURI {
		t.Errorf("expected target URI %q, got %q", expectedURI, entry.TargetURI)
	}
	// New section won't have existing text
	if entry.ExistingText != "" {
		t.Errorf("expected empty existing text for new section, got %q", entry.ExistingText)
	}
	if entry.ProposedText == "" {
		t.Error("expected non-empty proposed text for new section")
	}
	// New section won't have existing triples (it doesn't exist yet)
	if entry.AffectedTriples != 0 {
		t.Errorf("expected 0 affected triples for new section, got %d", entry.AffectedTriples)
	}
}

func TestComputeDiff_MultipleAmendments(t *testing.T) {
	triples := buildTitle15Triples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	bill := &DraftBill{
		BillNumber: "H.R. 1234",
		Title:      "Mixed Amendments Act",
		Sections: []*DraftSection{
			{
				Number: "1",
				Title:  "Short title",
			},
			{
				Number: "2",
				Title:  "Amendments to COPPA",
				Amendments: []Amendment{
					{
						Type:             AmendStrikeInsert,
						TargetTitle:      "15",
						TargetSection:    "6502",
						TargetSubsection: "b(1)",
						StrikeText:       "13",
						InsertText:       "16",
						Description:      `by striking "13" and inserting "16"`,
					},
					{
						Type:          AmendAddAtEnd,
						TargetTitle:   "15",
						TargetSection: "6502",
						InsertText:    "(e) Additional protections for teenagers.",
						Description:   "adding at the end the following new subsection",
					},
				},
			},
			{
				Number: "3",
				Title:  "Penalty increase",
				Amendments: []Amendment{
					{
						Type:             AmendStrikeInsert,
						TargetTitle:      "15",
						TargetSection:    "6505",
						TargetSubsection: "d",
						StrikeText:       "$50,000",
						InsertText:       "$100,000",
						Description:      `by striking "$50,000" and inserting "$100,000"`,
					},
				},
			},
			{
				Number: "4",
				Title:  "Repeal",
				Amendments: []Amendment{
					{
						Type:          AmendRepeal,
						TargetTitle:   "15",
						TargetSection: "6503",
						Description:   "Section 6503 of title 15 is repealed",
					},
				},
			},
		},
	}

	diff, err := ComputeDiff(bill, libraryPath)
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	if len(diff.Modified) != 2 {
		t.Errorf("expected 2 modified entries, got %d", len(diff.Modified))
	}
	if len(diff.Added) != 1 {
		t.Errorf("expected 1 added entry, got %d", len(diff.Added))
	}
	if len(diff.Removed) != 1 {
		t.Errorf("expected 1 removed entry, got %d", len(diff.Removed))
	}
	if len(diff.UnresolvedTargets) != 0 {
		t.Errorf("expected 0 unresolved targets, got %d: %v", len(diff.UnresolvedTargets), diff.UnresolvedTargets)
	}
	if diff.TriplesInvalidated == 0 {
		t.Error("expected non-zero total triples invalidated")
	}
}

func TestComputeDiff_UnresolvedTarget(t *testing.T) {
	triples := buildTitle15Triples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	bill := &DraftBill{
		BillNumber: "H.R. 5555",
		Title:      "Unresolved Target Act",
		Sections: []*DraftSection{
			{
				Number: "1",
				Title:  "Amendment to nonexistent title",
				Amendments: []Amendment{
					{
						Type:          AmendStrikeInsert,
						TargetTitle:   "99",
						TargetSection: "1234",
						StrikeText:    "old",
						InsertText:    "new",
						Description:   "targeting a title not in the library",
					},
				},
			},
			{
				Number: "2",
				Title:  "Valid amendment",
				Amendments: []Amendment{
					{
						Type:             AmendStrikeInsert,
						TargetTitle:      "15",
						TargetSection:    "6505",
						TargetSubsection: "d",
						StrikeText:       "$50,000",
						InsertText:       "$100,000",
					},
				},
			},
		},
	}

	diff, err := ComputeDiff(bill, libraryPath)
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	if len(diff.UnresolvedTargets) != 1 {
		t.Fatalf("expected 1 unresolved target, got %d: %v", len(diff.UnresolvedTargets), diff.UnresolvedTargets)
	}
	if len(diff.Modified) != 1 {
		t.Errorf("expected 1 modified entry, got %d", len(diff.Modified))
	}

	// Unresolved target should contain enough info to identify it
	unresolved := diff.UnresolvedTargets[0]
	if unresolved != "99 U.S.C. 1234" {
		t.Errorf("expected unresolved target %q, got %q", "99 U.S.C. 1234", unresolved)
	}
}

func TestResolveAmendmentTarget(t *testing.T) {
	triples := buildTitle15Triples()
	lib, _ := testLibrary(t, "us-usc-title-15", triples)

	tests := []struct {
		name             string
		amendment        Amendment
		expectedURI      string
		expectedDocID    string
		expectError      bool
	}{
		{
			name: "basic section reference",
			amendment: Amendment{
				TargetTitle:   "15",
				TargetSection: "6502",
			},
			expectedURI:   "https://regula.dev/regulations/US-USC-TITLE-15:Art6502",
			expectedDocID: "us-usc-title-15",
		},
		{
			name: "section with subsection",
			amendment: Amendment{
				TargetTitle:      "15",
				TargetSection:    "6505",
				TargetSubsection: "d",
			},
			expectedURI:   "https://regula.dev/regulations/US-USC-TITLE-15:Art6505(d)",
			expectedDocID: "us-usc-title-15",
		},
		{
			name: "missing title",
			amendment: Amendment{
				TargetSection: "6502",
			},
			expectError: true,
		},
		{
			name: "missing section",
			amendment: Amendment{
				TargetTitle: "15",
			},
			expectError: true,
		},
		{
			name: "document not in library",
			amendment: Amendment{
				TargetTitle:   "99",
				TargetSection: "1234",
			},
			expectError: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			targetURI, documentID, err := ResolveAmendmentTarget(testCase.amendment, lib)

			if testCase.expectError {
				if err == nil {
					t.Fatalf("expected error, got URI=%q docID=%q", targetURI, documentID)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if targetURI != testCase.expectedURI {
				t.Errorf("expected URI %q, got %q", testCase.expectedURI, targetURI)
			}
			if documentID != testCase.expectedDocID {
				t.Errorf("expected document ID %q, got %q", testCase.expectedDocID, documentID)
			}
		})
	}
}

func TestCountAffectedTriples(t *testing.T) {
	tripleStore := store.NewTripleStore()
	targetURI := "https://regula.dev/regulations/US-USC-TITLE-15:Art6502"
	otherURI := "https://regula.dev/regulations/US-USC-TITLE-15:Art6505"
	regURI := "https://regula.dev/regulations/US-USC-TITLE-15"

	triples := []store.Triple{
		// Triples where target is the subject
		{Subject: targetURI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: targetURI, Predicate: store.PropNumber, Object: "6502"},
		{Subject: targetURI, Predicate: store.PropText, Object: "Some provision text."},
		{Subject: targetURI, Predicate: store.PropPartOf, Object: regURI},
		// Triples where target is the object
		{Subject: otherURI, Predicate: store.PropReferences, Object: targetURI},
		{Subject: regURI, Predicate: store.PropContains, Object: targetURI},
		// Unrelated triples
		{Subject: otherURI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: otherURI, Predicate: store.PropText, Object: "Unrelated text."},
	}
	if err := tripleStore.BulkAdd(triples); err != nil {
		t.Fatalf("failed to bulk add triples: %v", err)
	}

	affectedCount := CountAffectedTriples(targetURI, tripleStore)

	// 4 as subject + 2 as object = 6
	expectedCount := 6
	if affectedCount != expectedCount {
		t.Errorf("expected %d affected triples, got %d", expectedCount, affectedCount)
	}

	// Non-existent URI should return 0
	noMatchCount := CountAffectedTriples("https://regula.dev/regulations/NONEXISTENT:Art1", tripleStore)
	if noMatchCount != 0 {
		t.Errorf("expected 0 affected triples for non-existent URI, got %d", noMatchCount)
	}
}

func TestFindCrossReferences(t *testing.T) {
	tripleStore := store.NewTripleStore()
	baseURI := "https://regula.dev/regulations/US-USC-TITLE-15:"
	targetURI := baseURI + "Art6502"
	refererURI1 := baseURI + "Art6505"
	refererURI2 := baseURI + "Art6503"
	referencedURI := baseURI + "Art6501"

	triples := []store.Triple{
		// Incoming: other articles reference the target
		{Subject: refererURI1, Predicate: store.PropReferences, Object: targetURI},
		{Subject: refererURI2, Predicate: store.PropReferences, Object: targetURI},
		// Also add inverse predicate for one of them (tests deduplication)
		{Subject: targetURI, Predicate: store.PropReferencedBy, Object: refererURI1},
		// Outgoing: target references another article
		{Subject: targetURI, Predicate: store.PropReferences, Object: referencedURI},
	}
	if err := tripleStore.BulkAdd(triples); err != nil {
		t.Fatalf("failed to bulk add triples: %v", err)
	}

	incoming, outgoing := FindCrossReferences(targetURI, tripleStore)

	// Incoming should have both referers (deduplicated)
	if len(incoming) != 2 {
		t.Errorf("expected 2 incoming cross-refs, got %d: %v", len(incoming), incoming)
	}

	// Check that both referers are present
	incomingMap := make(map[string]bool)
	for _, uri := range incoming {
		incomingMap[uri] = true
	}
	if !incomingMap[refererURI1] {
		t.Errorf("expected incoming ref from %q", refererURI1)
	}
	if !incomingMap[refererURI2] {
		t.Errorf("expected incoming ref from %q", refererURI2)
	}

	// Outgoing should have the referenced article
	if len(outgoing) != 1 {
		t.Errorf("expected 1 outgoing cross-ref, got %d: %v", len(outgoing), outgoing)
	}
	if len(outgoing) > 0 && outgoing[0] != referencedURI {
		t.Errorf("expected outgoing ref to %q, got %q", referencedURI, outgoing[0])
	}
}
