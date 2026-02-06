package draft

import (
	"testing"

	"github.com/coolbeans/regula/pkg/store"
)

// buildScenarioTriples creates a test triple set for scenario overlay testing.
// Includes articles 6502, 6503, 6505 with text, obligations, and cross-references.
func buildScenarioTriples() []store.Triple {
	baseURI := "https://regula.dev/regulations/"
	regID := "US-USC-TITLE-15"
	regURI := baseURI + regID

	art6502URI := baseURI + regID + ":Art6502"
	art6503URI := baseURI + regID + ":Art6503"
	art6505URI := baseURI + regID + ":Art6505"
	oblig6502URI := baseURI + regID + ":Obligation:6502:InformationProvision"
	oblig6503URI := baseURI + regID + ":Obligation:6503:DataMinimization"

	return []store.Triple{
		// Regulation node
		{Subject: regURI, Predicate: store.RDFType, Object: store.ClassRegulation},
		{Subject: regURI, Predicate: store.PropTitle, Object: "Title 15 - Commerce and Trade"},

		// Article 6502
		{Subject: art6502URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art6502URI, Predicate: store.PropNumber, Object: "6502"},
		{Subject: art6502URI, Predicate: store.PropTitle, Object: "Regulation of unfair and deceptive acts"},
		{Subject: art6502URI, Predicate: store.PropText, Object: "The operator shall provide notice of its information practices."},
		{Subject: art6502URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: art6502URI, Predicate: store.PropBelongsTo, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art6502URI},
		{Subject: regURI, Predicate: store.PropHasArticle, Object: art6502URI},

		// Article 6502 obligation
		{Subject: oblig6502URI, Predicate: store.RDFType, Object: store.ClassObligation},
		{Subject: oblig6502URI, Predicate: "reg:obligationType", Object: "InformationProvision"},
		{Subject: oblig6502URI, Predicate: store.PropText, Object: "shall provide notice"},
		{Subject: oblig6502URI, Predicate: store.PropPartOf, Object: art6502URI},
		{Subject: art6502URI, Predicate: store.PropImposesObligation, Object: oblig6502URI},

		// Article 6503
		{Subject: art6503URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art6503URI, Predicate: store.PropNumber, Object: "6503"},
		{Subject: art6503URI, Predicate: store.PropTitle, Object: "Data minimization requirements"},
		{Subject: art6503URI, Predicate: store.PropText, Object: "The operator shall not collect more personal information than necessary."},
		{Subject: art6503URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: art6503URI, Predicate: store.PropBelongsTo, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art6503URI},
		{Subject: regURI, Predicate: store.PropHasArticle, Object: art6503URI},

		// Article 6503 obligation
		{Subject: oblig6503URI, Predicate: store.RDFType, Object: store.ClassObligation},
		{Subject: oblig6503URI, Predicate: "reg:obligationType", Object: "DataMinimization"},
		{Subject: oblig6503URI, Predicate: store.PropText, Object: "shall not collect more than necessary"},
		{Subject: oblig6503URI, Predicate: store.PropPartOf, Object: art6503URI},
		{Subject: art6503URI, Predicate: store.PropImposesObligation, Object: oblig6503URI},

		// Article 6505 with cross-references
		{Subject: art6505URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art6505URI, Predicate: store.PropNumber, Object: "6505"},
		{Subject: art6505URI, Predicate: store.PropTitle, Object: "Enforcement provisions"},
		{Subject: art6505URI, Predicate: store.PropText, Object: "Violations of section 6502 or 6503 are subject to enforcement."},
		{Subject: art6505URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: art6505URI, Predicate: store.PropBelongsTo, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art6505URI},
		{Subject: regURI, Predicate: store.PropHasArticle, Object: art6505URI},

		// Cross-references from 6505 to 6502 and 6503
		{Subject: art6505URI, Predicate: store.PropReferences, Object: art6502URI},
		{Subject: art6502URI, Predicate: store.PropReferencedBy, Object: art6505URI},
		{Subject: art6505URI, Predicate: store.PropReferences, Object: art6503URI},
		{Subject: art6503URI, Predicate: store.PropReferencedBy, Object: art6505URI},
	}
}

func TestApplyDraftOverlay_Repeal(t *testing.T) {
	// Setup: Create library with test triples
	_, libraryPath := testLibrary(t, "us-usc-title-15", buildScenarioTriples())

	// Create a diff with a repeal of section 6502
	diff := &DraftDiff{
		Bill: &DraftBill{
			BillNumber: "HR-TEST-1",
			Congress:   "119th",
			Title:      "Test Repeal Act",
		},
		Removed: []DiffEntry{
			{
				Amendment: Amendment{
					Type:          AmendRepeal,
					TargetTitle:   "15",
					TargetSection: "6502",
				},
				TargetURI:        "https://regula.dev/regulations/US-USC-TITLE-15:Art6502",
				TargetDocumentID: "us-usc-title-15",
				ExistingText:     "The operator shall provide notice of its information practices.",
				AffectedTriples:  10,
			},
		},
	}

	// Apply overlay
	overlay, err := ApplyDraftOverlay(diff, libraryPath)
	if err != nil {
		t.Fatalf("ApplyDraftOverlay failed: %v", err)
	}

	// Verify section 6502 is removed
	art6502URI := "https://regula.dev/regulations/US-USC-TITLE-15:Art6502"
	art6502Triples := overlay.OverlayStore.Find(art6502URI, "", "")
	if len(art6502Triples) > 0 {
		t.Errorf("Expected section 6502 to be removed, but found %d triples", len(art6502Triples))
	}

	// Verify the obligation is also removed
	oblig6502URI := "https://regula.dev/regulations/US-USC-TITLE-15:Obligation:6502:InformationProvision"
	obligTriples := overlay.OverlayStore.Find(oblig6502URI, "", "")
	if len(obligTriples) > 0 {
		t.Errorf("Expected obligation to be removed, but found %d triples", len(obligTriples))
	}

	// Verify section 6503 still exists
	art6503URI := "https://regula.dev/regulations/US-USC-TITLE-15:Art6503"
	art6503Triples := overlay.OverlayStore.Find(art6503URI, "", "")
	if len(art6503Triples) == 0 {
		t.Error("Expected section 6503 to still exist")
	}

	// Verify stats
	if overlay.Stats.TriplesRemoved == 0 {
		t.Error("Expected some triples to be removed")
	}
	if len(overlay.AppliedAmendments) != 1 {
		t.Errorf("Expected 1 applied amendment, got %d", len(overlay.AppliedAmendments))
	}
}

func TestApplyDraftOverlay_Modification(t *testing.T) {
	// Setup
	_, libraryPath := testLibrary(t, "us-usc-title-15", buildScenarioTriples())

	// Create a diff with a modification to section 6502
	diff := &DraftDiff{
		Bill: &DraftBill{
			BillNumber:   "HR-TEST-2",
			Congress: "119th",
			Title:    "Test Modification Act",
		},
		Modified: []DiffEntry{
			{
				Amendment: Amendment{
					Type:          AmendStrikeInsert,
					TargetTitle:   "15",
					TargetSection: "6502",
					StrikeText:    "shall provide notice",
					InsertText:    "The operator must provide clear and conspicuous notice of all data collection practices.",
				},
				TargetURI:        "https://regula.dev/regulations/US-USC-TITLE-15:Art6502",
				TargetDocumentID: "us-usc-title-15",
				ExistingText:     "The operator shall provide notice of its information practices.",
				ProposedText:     "The operator must provide clear and conspicuous notice of all data collection practices.",
				AffectedTriples:  10,
			},
		},
	}

	// Apply overlay
	overlay, err := ApplyDraftOverlay(diff, libraryPath)
	if err != nil {
		t.Fatalf("ApplyDraftOverlay failed: %v", err)
	}

	// Verify section 6502 has new text
	art6502URI := "https://regula.dev/regulations/US-USC-TITLE-15:Art6502"
	newText := overlay.OverlayStore.GetOne(art6502URI, store.PropText)
	if newText != "The operator must provide clear and conspicuous notice of all data collection practices." {
		t.Errorf("Expected new text, got: %s", newText)
	}

	// Verify stats
	if overlay.Stats.TriplesRemoved == 0 {
		t.Error("Expected some triples to be removed")
	}
	if overlay.Stats.TriplesAdded == 0 {
		t.Error("Expected some triples to be added")
	}
}

func TestApplyDraftOverlay_Addition(t *testing.T) {
	// Setup
	_, libraryPath := testLibrary(t, "us-usc-title-15", buildScenarioTriples())

	// Create a diff with an addition of new section 6510
	diff := &DraftDiff{
		Bill: &DraftBill{
			BillNumber:   "HR-TEST-3",
			Congress: "119th",
			Title:    "Test Addition Act",
		},
		Added: []DiffEntry{
			{
				Amendment: Amendment{
					Type:          AmendAddNewSection,
					TargetTitle:   "15",
					TargetSection: "6510",
					InsertText:    "The data subject shall have the right to access personal data.",
				},
				TargetURI:        "https://regula.dev/regulations/US-USC-TITLE-15:Art6510",
				TargetDocumentID: "us-usc-title-15",
				ProposedText:     "The data subject shall have the right to access personal data.",
			},
		},
	}

	// Apply overlay
	overlay, err := ApplyDraftOverlay(diff, libraryPath)
	if err != nil {
		t.Fatalf("ApplyDraftOverlay failed: %v", err)
	}

	// Verify new section 6510 exists
	art6510URI := "https://regula.dev/regulations/US-USC-TITLE-15:Art6510"
	art6510Triples := overlay.OverlayStore.Find(art6510URI, "", "")
	if len(art6510Triples) == 0 {
		t.Error("Expected new section 6510 to be added")
	}

	// Verify text
	newText := overlay.OverlayStore.GetOne(art6510URI, store.PropText)
	if newText != "The data subject shall have the right to access personal data." {
		t.Errorf("Expected new section text, got: %s", newText)
	}

	// Verify stats
	if overlay.Stats.TriplesAdded == 0 {
		t.Error("Expected some triples to be added")
	}
}

func TestApplyDraftOverlay_BaseUnchanged(t *testing.T) {
	// Setup
	lib, libraryPath := testLibrary(t, "us-usc-title-15", buildScenarioTriples())

	// Count initial triples
	baseStore, err := lib.LoadTripleStore("us-usc-title-15")
	if err != nil {
		t.Fatalf("Failed to load base store: %v", err)
	}
	initialCount := baseStore.Count()

	// Create a diff with a repeal
	diff := &DraftDiff{
		Bill: &DraftBill{
			BillNumber: "HR-TEST-4",
			Title:  "Test Base Unchanged Act",
		},
		Removed: []DiffEntry{
			{
				Amendment: Amendment{
					Type:          AmendRepeal,
					TargetTitle:   "15",
					TargetSection: "6502",
				},
				TargetURI:        "https://regula.dev/regulations/US-USC-TITLE-15:Art6502",
				TargetDocumentID: "us-usc-title-15",
			},
		},
	}

	// Apply overlay
	_, err = ApplyDraftOverlay(diff, libraryPath)
	if err != nil {
		t.Fatalf("ApplyDraftOverlay failed: %v", err)
	}

	// Reload base store and verify it's unchanged
	reloadedStore, err := lib.LoadTripleStore("us-usc-title-15")
	if err != nil {
		t.Fatalf("Failed to reload base store: %v", err)
	}

	if reloadedStore.Count() != initialCount {
		t.Errorf("Base store was modified: expected %d triples, got %d", initialCount, reloadedStore.Count())
	}

	// Verify the original article still exists in base
	art6502URI := "https://regula.dev/regulations/US-USC-TITLE-15:Art6502"
	art6502Triples := reloadedStore.Find(art6502URI, "", "")
	if len(art6502Triples) == 0 {
		t.Error("Base store was modified: section 6502 should still exist")
	}
}

func TestApplyDraftOverlay_MultipleAmendments(t *testing.T) {
	// Setup
	_, libraryPath := testLibrary(t, "us-usc-title-15", buildScenarioTriples())

	// Create a diff with multiple amendments
	diff := &DraftDiff{
		Bill: &DraftBill{
			BillNumber:   "HR-TEST-5",
			Congress: "119th",
			Title:    "Test Multiple Amendments Act",
		},
		Removed: []DiffEntry{
			{
				Amendment: Amendment{
					Type:          AmendRepeal,
					TargetTitle:   "15",
					TargetSection: "6502",
				},
				TargetURI:        "https://regula.dev/regulations/US-USC-TITLE-15:Art6502",
				TargetDocumentID: "us-usc-title-15",
			},
		},
		Modified: []DiffEntry{
			{
				Amendment: Amendment{
					Type:          AmendStrikeInsert,
					TargetTitle:   "15",
					TargetSection: "6503",
					StrikeText:    "shall not collect",
					InsertText:    "The operator is prohibited from collecting more personal information than strictly necessary.",
				},
				TargetURI:        "https://regula.dev/regulations/US-USC-TITLE-15:Art6503",
				TargetDocumentID: "us-usc-title-15",
			},
		},
		Added: []DiffEntry{
			{
				Amendment: Amendment{
					Type:          AmendAddNewSection,
					TargetTitle:   "15",
					TargetSection: "6510",
					InsertText:    "New section added by the act.",
				},
				TargetURI:        "https://regula.dev/regulations/US-USC-TITLE-15:Art6510",
				TargetDocumentID: "us-usc-title-15",
			},
		},
	}

	// Apply overlay
	overlay, err := ApplyDraftOverlay(diff, libraryPath)
	if err != nil {
		t.Fatalf("ApplyDraftOverlay failed: %v", err)
	}

	// Verify all amendments applied
	if len(overlay.AppliedAmendments) != 3 {
		t.Errorf("Expected 3 applied amendments, got %d", len(overlay.AppliedAmendments))
	}

	// Verify 6502 is removed
	art6502URI := "https://regula.dev/regulations/US-USC-TITLE-15:Art6502"
	if len(overlay.OverlayStore.Find(art6502URI, "", "")) > 0 {
		t.Error("Expected section 6502 to be removed")
	}

	// Verify 6503 is modified
	art6503URI := "https://regula.dev/regulations/US-USC-TITLE-15:Art6503"
	text6503 := overlay.OverlayStore.GetOne(art6503URI, store.PropText)
	if text6503 != "The operator is prohibited from collecting more personal information than strictly necessary." {
		t.Errorf("Expected modified text for 6503, got: %s", text6503)
	}

	// Verify 6510 is added
	art6510URI := "https://regula.dev/regulations/US-USC-TITLE-15:Art6510"
	if len(overlay.OverlayStore.Find(art6510URI, "", "")) == 0 {
		t.Error("Expected section 6510 to be added")
	}
}

func TestIngestDraftSection(t *testing.T) {
	// Create a simple triple store
	tripleStore := store.NewTripleStore()
	baseURI := "https://regula.dev/regulations/"
	documentID := "us-usc-title-15"

	amendment := Amendment{
		Type:          AmendAddNewSection,
		TargetTitle:   "15",
		TargetSection: "6510",
		InsertText:    "The data subject shall have the right to access personal data collected by the operator.",
	}

	// Ingest the section
	added, err := IngestDraftSection(amendment, tripleStore, baseURI, documentID)
	if err != nil {
		t.Fatalf("IngestDraftSection failed: %v", err)
	}

	// Verify basic triples were added
	if added == 0 {
		t.Error("Expected some triples to be added")
	}

	// Verify article exists
	articleURI := "https://regula.dev/regulations/US-USC-TITLE-15:Art6510"
	articleTriples := tripleStore.Find(articleURI, "", "")
	if len(articleTriples) == 0 {
		t.Error("Expected article triples to be added")
	}

	// Verify text
	text := tripleStore.GetOne(articleURI, store.PropText)
	if text != "The data subject shall have the right to access personal data collected by the operator." {
		t.Errorf("Expected article text, got: %s", text)
	}

	// Verify type
	typeVal := tripleStore.GetOne(articleURI, store.RDFType)
	if typeVal != store.ClassArticle {
		t.Errorf("Expected article type, got: %s", typeVal)
	}

	// Verify number
	number := tripleStore.GetOne(articleURI, store.PropNumber)
	if number != "6510" {
		t.Errorf("Expected number 6510, got: %s", number)
	}
}

func TestIngestDraftSection_WithSubsection(t *testing.T) {
	tripleStore := store.NewTripleStore()
	baseURI := "https://regula.dev/regulations/"
	documentID := "us-usc-title-15"

	amendment := Amendment{
		Type:             AmendAddNewSection,
		TargetTitle:      "15",
		TargetSection:    "6502",
		TargetSubsection: "b",
		InsertText:       "The operator must provide a privacy policy.",
	}

	_, err := IngestDraftSection(amendment, tripleStore, baseURI, documentID)
	if err != nil {
		t.Fatalf("IngestDraftSection failed: %v", err)
	}

	// Verify subsection URI format
	subsectionURI := "https://regula.dev/regulations/US-USC-TITLE-15:Art6502(b)"
	subsectionTriples := tripleStore.Find(subsectionURI, "", "")
	if len(subsectionTriples) == 0 {
		t.Error("Expected subsection triples to be added")
	}
}

func TestCloneTripleStore(t *testing.T) {
	// Create source store
	source := store.NewTripleStore()
	source.Add("subject1", "predicate1", "object1")
	source.Add("subject2", "predicate2", "object2")

	// Clone
	cloned := CloneTripleStore(source)

	// Verify clone has same content
	if cloned.Count() != source.Count() {
		t.Errorf("Expected clone to have %d triples, got %d", source.Count(), cloned.Count())
	}

	// Modify clone
	cloned.Add("subject3", "predicate3", "object3")

	// Verify source is unchanged
	if source.Count() != 2 {
		t.Errorf("Source was modified: expected 2 triples, got %d", source.Count())
	}

	// Verify clone has new triple
	if cloned.Count() != 3 {
		t.Errorf("Clone should have 3 triples, got %d", cloned.Count())
	}
}

func TestCloneTripleStore_Nil(t *testing.T) {
	cloned := CloneTripleStore(nil)
	if cloned != nil {
		t.Error("Expected nil when cloning nil store")
	}
}

func TestApplyDraftOverlay_NilDiff(t *testing.T) {
	_, err := ApplyDraftOverlay(nil, "/tmp/test")
	if err == nil {
		t.Error("Expected error for nil diff")
	}
}

func TestOverlayStats(t *testing.T) {
	// Setup
	_, libraryPath := testLibrary(t, "us-usc-title-15", buildScenarioTriples())

	diff := &DraftDiff{
		Bill: &DraftBill{
			BillNumber: "HR-TEST-6",
			Title:  "Test Stats Act",
		},
		Removed: []DiffEntry{
			{
				Amendment: Amendment{
					Type:          AmendRepeal,
					TargetTitle:   "15",
					TargetSection: "6502",
				},
				TargetURI:        "https://regula.dev/regulations/US-USC-TITLE-15:Art6502",
				TargetDocumentID: "us-usc-title-15",
			},
		},
		Added: []DiffEntry{
			{
				Amendment: Amendment{
					Type:          AmendAddNewSection,
					TargetTitle:   "15",
					TargetSection: "6510",
					InsertText:    "New section.",
				},
				TargetURI:        "https://regula.dev/regulations/US-USC-TITLE-15:Art6510",
				TargetDocumentID: "us-usc-title-15",
			},
		},
	}

	overlay, err := ApplyDraftOverlay(diff, libraryPath)
	if err != nil {
		t.Fatalf("ApplyDraftOverlay failed: %v", err)
	}

	// Verify stats fields are populated
	if overlay.Stats.BaseTriples == 0 {
		t.Error("Expected BaseTriples to be populated")
	}
	if overlay.Stats.OverlayTriples == 0 {
		t.Error("Expected OverlayTriples to be populated")
	}
	if overlay.Stats.TriplesRemoved == 0 {
		t.Error("Expected TriplesRemoved to be populated")
	}
	if overlay.Stats.TriplesAdded == 0 {
		t.Error("Expected TriplesAdded to be populated")
	}
}
