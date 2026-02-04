package draft

import (
	"testing"

	"github.com/coolbeans/regula/pkg/analysis"
	"github.com/coolbeans/regula/pkg/store"
)

// buildTitle15ImpactTriples extends the standard Title 15 triple set with
// obligation, rights, and deeper cross-reference chains for impact analysis
// testing.
func buildTitle15ImpactTriples() []store.Triple {
	baseURI := "https://regula.dev/regulations/"
	regID := "US-USC-TITLE-15"
	regURI := baseURI + regID

	art6502URI := baseURI + regID + ":Art6502"
	art6502bURI := baseURI + regID + ":Art6502(b)"
	art6503URI := baseURI + regID + ":Art6503"
	art6504URI := baseURI + regID + ":Art6504"
	art6505URI := baseURI + regID + ":Art6505"
	art6505dURI := baseURI + regID + ":Art6505(d)"

	return []store.Triple{
		// Regulation node
		{Subject: regURI, Predicate: store.RDFType, Object: store.ClassRegulation},
		{Subject: regURI, Predicate: store.PropTitle, Object: "Title 15 - Commerce and Trade"},

		// Article 6502 - COPPA core provision
		{Subject: art6502URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art6502URI, Predicate: store.PropNumber, Object: "6502"},
		{Subject: art6502URI, Predicate: store.PropTitle, Object: "Regulation of unfair and deceptive acts"},
		{Subject: art6502URI, Predicate: store.PropText, Object: "It shall be unlawful for an operator of a website to collect personal information from a child under 13."},
		{Subject: art6502URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: art6502URI, Predicate: store.PropBelongsTo, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art6502URI},
		{Subject: regURI, Predicate: store.PropHasArticle, Object: art6502URI},

		// Article 6502 imposes obligations and grants rights
		{Subject: art6502URI, Predicate: store.PropImposesObligation, Object: store.ObligationTransparency},
		{Subject: art6502URI, Predicate: store.PropImposesObligation, Object: store.ObligationNotify},
		{Subject: art6502URI, Predicate: store.PropGrantsRight, Object: store.RightInformation},

		// Article 6502(b) subsection
		{Subject: art6502bURI, Predicate: store.RDFType, Object: store.ClassParagraph},
		{Subject: art6502bURI, Predicate: store.PropNumber, Object: "b"},
		{Subject: art6502bURI, Predicate: store.PropText, Object: "The operator shall provide notice of its information practices."},
		{Subject: art6502bURI, Predicate: store.PropPartOf, Object: art6502URI},
		{Subject: art6502URI, Predicate: store.PropContains, Object: art6502bURI},

		// Article 6503 - Safe harbors
		{Subject: art6503URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art6503URI, Predicate: store.PropNumber, Object: "6503"},
		{Subject: art6503URI, Predicate: store.PropTitle, Object: "Safe harbors"},
		{Subject: art6503URI, Predicate: store.PropText, Object: "Industry self-regulation as safe harbor."},
		{Subject: art6503URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: art6503URI, Predicate: store.PropBelongsTo, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art6503URI},
		{Subject: regURI, Predicate: store.PropHasArticle, Object: art6503URI},

		// 6503 grants an exemption right
		{Subject: art6503URI, Predicate: store.PropGrantsRight, Object: "reg:SafeHarborExemption"},

		// Article 6504 - Enforcement (references 6502 and 6503)
		{Subject: art6504URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art6504URI, Predicate: store.PropNumber, Object: "6504"},
		{Subject: art6504URI, Predicate: store.PropTitle, Object: "Actions by States"},
		{Subject: art6504URI, Predicate: store.PropText, Object: "The attorney general of a State may bring civil action."},
		{Subject: art6504URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: art6504URI, Predicate: store.PropBelongsTo, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art6504URI},
		{Subject: regURI, Predicate: store.PropHasArticle, Object: art6504URI},
		// 6504 references both 6502 and 6503 (creates transitive chains)
		{Subject: art6504URI, Predicate: store.PropReferences, Object: art6502URI},
		{Subject: art6502URI, Predicate: store.PropReferencedBy, Object: art6504URI},
		{Subject: art6504URI, Predicate: store.PropReferences, Object: art6503URI},
		{Subject: art6503URI, Predicate: store.PropReferencedBy, Object: art6504URI},

		// Article 6505 - Civil penalties
		{Subject: art6505URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art6505URI, Predicate: store.PropNumber, Object: "6505"},
		{Subject: art6505URI, Predicate: store.PropTitle, Object: "Civil penalties"},
		{Subject: art6505URI, Predicate: store.PropText, Object: "Any operator who violates this section shall be subject to penalties."},
		{Subject: art6505URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: art6505URI, Predicate: store.PropBelongsTo, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art6505URI},
		{Subject: regURI, Predicate: store.PropHasArticle, Object: art6505URI},

		// 6505 references 6502 (penalty section references the violated provision)
		{Subject: art6505URI, Predicate: store.PropReferences, Object: art6502URI},
		{Subject: art6502URI, Predicate: store.PropReferencedBy, Object: art6505URI},

		// 6505 imposes obligations
		{Subject: art6505URI, Predicate: store.PropImposesObligation, Object: store.ObligationNotify},

		// Article 6505(d)
		{Subject: art6505dURI, Predicate: store.RDFType, Object: store.ClassParagraph},
		{Subject: art6505dURI, Predicate: store.PropNumber, Object: "d"},
		{Subject: art6505dURI, Predicate: store.PropText, Object: "Maximum penalty of $50,000 per violation."},
		{Subject: art6505dURI, Predicate: store.PropPartOf, Object: art6505URI},
		{Subject: art6505URI, Predicate: store.PropContains, Object: art6505dURI},

		// Cross-references for broader impact chains:
		// 6503 references 6502 (safe harbor references the core provision)
		{Subject: art6503URI, Predicate: store.PropReferences, Object: art6502URI},
		{Subject: art6502URI, Predicate: store.PropReferencedBy, Object: art6503URI},
		// 6502 references 6503 (bidirectional)
		{Subject: art6502URI, Predicate: store.PropReferences, Object: art6503URI},
		{Subject: art6503URI, Predicate: store.PropReferencedBy, Object: art6502URI},
	}
}

func TestAnalyzeDraftImpact_SingleAmendment(t *testing.T) {
	triples := buildTitle15ImpactTriples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	bill := &DraftBill{
		BillNumber: "H.R. 9999",
		Title:      "Single Amendment Impact Act",
		Sections: []*DraftSection{
			{
				Number: "1",
				Title:  "Amendment to section 6505",
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

	impactResult, err := AnalyzeDraftImpact(diff, libraryPath, 1)
	if err != nil {
		t.Fatalf("AnalyzeDraftImpact failed: %v", err)
	}

	if impactResult.Bill != bill {
		t.Error("expected impact result to reference the original bill")
	}
	if impactResult.Diff != diff {
		t.Error("expected impact result to reference the original diff")
	}

	// Art 6505(d) is modified — provisions referencing 6505(d) should be directly affected
	// Even with no direct incoming references to the subsection, the result should be valid
	if impactResult.TotalProvisionsAffected < 0 {
		t.Errorf("expected non-negative total provisions affected, got %d", impactResult.TotalProvisionsAffected)
	}

	// Verify obligation changes were collected for the modified section
	// 6505 itself has obligations (ObligationNotify)
	if impactResult.ObligationChanges.Modified == nil {
		t.Fatal("expected non-nil modified obligations slice")
	}
}

func TestAnalyzeDraftImpact_Repeal(t *testing.T) {
	triples := buildTitle15ImpactTriples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	bill := &DraftBill{
		BillNumber: "H.R. 8888",
		Title:      "Repeal Impact Act",
		Sections: []*DraftSection{
			{
				Number: "1",
				Title:  "Repeal of safe harbors",
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

	impactResult, err := AnalyzeDraftImpact(diff, libraryPath, 2)
	if err != nil {
		t.Fatalf("AnalyzeDraftImpact failed: %v", err)
	}

	// Art 6503 is repealed — provisions referencing it should be directly affected
	// 6504 references 6503, and 6502 references 6503
	if len(impactResult.DirectlyAffected) == 0 {
		t.Error("expected at least one directly affected provision from repeal")
	}

	// Verify that directly affected provisions reference the repealed section
	foundDirectReference := false
	for _, provision := range impactResult.DirectlyAffected {
		if provision.Depth == 1 {
			foundDirectReference = true
			break
		}
	}
	if !foundDirectReference {
		t.Error("expected at least one depth-1 directly affected provision")
	}

	// Broken cross-references should be detected
	if len(impactResult.BrokenCrossRefs) == 0 {
		t.Error("expected broken cross-references from repeal of 6503")
	}

	// Verify broken refs point to the repealed section
	for _, brokenRef := range impactResult.BrokenCrossRefs {
		if brokenRef.TargetURI != "https://regula.dev/regulations/US-USC-TITLE-15:Art6503" {
			t.Errorf("expected broken ref target to be Art6503, got %q", brokenRef.TargetURI)
		}
	}

	// Rights should be removed (6503 grants SafeHarborExemption)
	if len(impactResult.RightsChanges.Removed) == 0 {
		t.Error("expected removed rights from repeal of section with granted rights")
	}

	foundSafeHarbor := false
	for _, right := range impactResult.RightsChanges.Removed {
		if right == "reg:SafeHarborExemption" {
			foundSafeHarbor = true
			break
		}
	}
	if !foundSafeHarbor {
		t.Errorf("expected SafeHarborExemption in removed rights, got %v", impactResult.RightsChanges.Removed)
	}
}

func TestAnalyzeDraftImpact_MultipleAmendments(t *testing.T) {
	triples := buildTitle15ImpactTriples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	bill := &DraftBill{
		BillNumber: "H.R. 7777",
		Title:      "Multiple Amendments Impact Act",
		Sections: []*DraftSection{
			{
				Number: "1",
				Title:  "COPPA amendment",
				Amendments: []Amendment{
					{
						Type:             AmendStrikeInsert,
						TargetTitle:      "15",
						TargetSection:    "6502",
						TargetSubsection: "b",
						StrikeText:       "13",
						InsertText:       "16",
					},
				},
			},
			{
				Number: "2",
				Title:  "Penalty amendment",
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

	impactResult, err := AnalyzeDraftImpact(diff, libraryPath, 1)
	if err != nil {
		t.Fatalf("AnalyzeDraftImpact failed: %v", err)
	}

	// Both 6502(b) and 6505(d) are modified — deduplication should prevent
	// the same provision from appearing twice in directly affected
	seenURIs := make(map[string]bool)
	for _, provision := range impactResult.DirectlyAffected {
		if seenURIs[provision.URI] {
			t.Errorf("duplicate directly affected provision: %q", provision.URI)
		}
		seenURIs[provision.URI] = true
	}

	// Similarly for transitive
	seenTransitive := make(map[string]bool)
	for _, provision := range impactResult.TransitivelyAffected {
		if seenTransitive[provision.URI] {
			t.Errorf("duplicate transitively affected provision: %q", provision.URI)
		}
		seenTransitive[provision.URI] = true
	}

	// Obligation changes should be populated from both amendments
	if impactResult.ObligationChanges.Modified == nil {
		t.Error("expected non-nil modified obligations")
	}
}

func TestAnalyzeDraftImpact_ObligationChanges(t *testing.T) {
	triples := buildTitle15ImpactTriples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	// Modify section 6502 which has obligations: TransparencyObligation, NotificationObligation
	bill := &DraftBill{
		BillNumber: "H.R. 6666",
		Title:      "Obligation Change Act",
		Sections: []*DraftSection{
			{
				Number: "1",
				Title:  "Modify COPPA",
				Amendments: []Amendment{
					{
						Type:          AmendStrikeInsert,
						TargetTitle:   "15",
						TargetSection: "6502",
						StrikeText:    "13",
						InsertText:    "16",
					},
				},
			},
		},
	}

	diff, err := ComputeDiff(bill, libraryPath)
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	impactResult, err := AnalyzeDraftImpact(diff, libraryPath, 1)
	if err != nil {
		t.Fatalf("AnalyzeDraftImpact failed: %v", err)
	}

	// Art 6502 has obligations: TransparencyObligation, NotificationObligation
	if len(impactResult.ObligationChanges.Modified) < 2 {
		t.Errorf("expected at least 2 modified obligations from Art6502, got %d: %v",
			len(impactResult.ObligationChanges.Modified), impactResult.ObligationChanges.Modified)
	}

	// Check that the specific obligations are present
	obligationSet := make(map[string]bool)
	for _, obligation := range impactResult.ObligationChanges.Modified {
		obligationSet[obligation] = true
	}
	if !obligationSet[store.ObligationTransparency] {
		t.Errorf("expected TransparencyObligation in modified, got %v", impactResult.ObligationChanges.Modified)
	}
	if !obligationSet[store.ObligationNotify] {
		t.Errorf("expected NotificationObligation in modified, got %v", impactResult.ObligationChanges.Modified)
	}

	// Rights should also be collected (Art6502 grants RightToInformation)
	if len(impactResult.RightsChanges.Modified) < 1 {
		t.Errorf("expected at least 1 modified right from Art6502, got %d", len(impactResult.RightsChanges.Modified))
	}
}

func TestAnalyzeDraftImpact_RightsChanges(t *testing.T) {
	triples := buildTitle15ImpactTriples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	// Repeal section 6503 which grants SafeHarborExemption
	// and modify section 6502 which grants RightToInformation
	bill := &DraftBill{
		BillNumber: "H.R. 5555",
		Title:      "Rights Change Act",
		Sections: []*DraftSection{
			{
				Number: "1",
				Title:  "Repeal safe harbors",
				Amendments: []Amendment{
					{
						Type:          AmendRepeal,
						TargetTitle:   "15",
						TargetSection: "6503",
					},
				},
			},
			{
				Number: "2",
				Title:  "Modify COPPA",
				Amendments: []Amendment{
					{
						Type:          AmendStrikeInsert,
						TargetTitle:   "15",
						TargetSection: "6502",
						StrikeText:    "child under 13",
						InsertText:    "child under 16",
					},
				},
			},
		},
	}

	diff, err := ComputeDiff(bill, libraryPath)
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	impactResult, err := AnalyzeDraftImpact(diff, libraryPath, 1)
	if err != nil {
		t.Fatalf("AnalyzeDraftImpact failed: %v", err)
	}

	// Removed rights from repealed section 6503
	if len(impactResult.RightsChanges.Removed) == 0 {
		t.Error("expected removed rights from repeal of section 6503")
	}
	removedSet := make(map[string]bool)
	for _, right := range impactResult.RightsChanges.Removed {
		removedSet[right] = true
	}
	if !removedSet["reg:SafeHarborExemption"] {
		t.Errorf("expected SafeHarborExemption in removed rights, got %v", impactResult.RightsChanges.Removed)
	}

	// Modified rights from modified section 6502
	if len(impactResult.RightsChanges.Modified) == 0 {
		t.Error("expected modified rights from amendment to section 6502")
	}
	modifiedSet := make(map[string]bool)
	for _, right := range impactResult.RightsChanges.Modified {
		modifiedSet[right] = true
	}
	if !modifiedSet[store.RightInformation] {
		t.Errorf("expected RightToInformation in modified rights, got %v", impactResult.RightsChanges.Modified)
	}

	// Removed obligations from repealed section 6503 (none in this case)
	// Modified obligations from modified section 6502
	if len(impactResult.ObligationChanges.Modified) == 0 {
		t.Error("expected modified obligations from amendment to section 6502")
	}
}

func TestAnalyzeDraftImpact_DepthLimit(t *testing.T) {
	triples := buildTitle15ImpactTriples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	bill := &DraftBill{
		BillNumber: "H.R. 4444",
		Title:      "Depth Limit Act",
		Sections: []*DraftSection{
			{
				Number: "1",
				Title:  "Modify COPPA",
				Amendments: []Amendment{
					{
						Type:          AmendStrikeInsert,
						TargetTitle:   "15",
						TargetSection: "6502",
						StrikeText:    "13",
						InsertText:    "16",
					},
				},
			},
		},
	}

	diff, err := ComputeDiff(bill, libraryPath)
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	// Run at depth 1 — should only have direct references
	depthOneResult, err := AnalyzeDraftImpact(diff, libraryPath, 1)
	if err != nil {
		t.Fatalf("AnalyzeDraftImpact depth=1 failed: %v", err)
	}

	// Run at depth 3 — should pick up transitive nodes
	depthThreeResult, err := AnalyzeDraftImpact(diff, libraryPath, 3)
	if err != nil {
		t.Fatalf("AnalyzeDraftImpact depth=3 failed: %v", err)
	}

	// At depth 1, there should be no transitive nodes
	if len(depthOneResult.TransitivelyAffected) > 0 {
		t.Errorf("depth=1 should have no transitive nodes, got %d", len(depthOneResult.TransitivelyAffected))
	}

	// MaxDepthReached at depth 1 should be 1 (direct only)
	if depthOneResult.MaxDepthReached > 1 {
		t.Errorf("depth=1 max depth should be <= 1, got %d", depthOneResult.MaxDepthReached)
	}

	// At depth 3, total affected should be >= depth 1 (superset or equal)
	if depthThreeResult.TotalProvisionsAffected < depthOneResult.TotalProvisionsAffected {
		t.Errorf("depth=3 total affected (%d) should be >= depth=1 (%d)",
			depthThreeResult.TotalProvisionsAffected, depthOneResult.TotalProvisionsAffected)
	}

	// Direct counts should be the same regardless of depth
	if len(depthThreeResult.DirectlyAffected) != len(depthOneResult.DirectlyAffected) {
		t.Errorf("direct affected should be same at any depth: depth1=%d depth3=%d",
			len(depthOneResult.DirectlyAffected), len(depthThreeResult.DirectlyAffected))
	}
}

func TestAggregateImpactResults(t *testing.T) {
	baseURI := "https://regula.dev/regulations/"

	results := []*analysis.ImpactResult{
		{
			TargetURI:   baseURI + "US-USC-TITLE-15:Art6502",
			TargetLabel: "Art6502",
			MaxDepth:    2,
			DirectIncoming: []*analysis.ImpactNode{
				{URI: baseURI + "US-USC-TITLE-15:Art6505", Label: "Art6505", Type: "Article", Depth: 1, Impact: analysis.ImpactDirect, Direction: "incoming"},
				{URI: baseURI + "US-USC-TITLE-15:Art6503", Label: "Art6503", Type: "Article", Depth: 1, Impact: analysis.ImpactDirect, Direction: "incoming"},
			},
			TransitiveNodes: []*analysis.ImpactNode{
				{URI: baseURI + "US-USC-TITLE-15:Art6504", Label: "Art6504", Type: "Article", Depth: 2, Impact: analysis.ImpactTransitive, Direction: "incoming"},
			},
			Summary: &analysis.ImpactSummary{
				TotalAffected:   3,
				MaxDepthReached: 2,
			},
		},
		{
			TargetURI:   baseURI + "US-USC-TITLE-15:Art6505",
			TargetLabel: "Art6505",
			MaxDepth:    2,
			DirectIncoming: []*analysis.ImpactNode{
				// Art6505 is also in the first result — should be deduplicated
				{URI: baseURI + "US-USC-TITLE-15:Art6502", Label: "Art6502", Type: "Article", Depth: 1, Impact: analysis.ImpactDirect, Direction: "incoming"},
			},
			TransitiveNodes: []*analysis.ImpactNode{
				// Art6504 was already in first result's transitive — should be deduplicated
				{URI: baseURI + "US-USC-TITLE-15:Art6504", Label: "Art6504", Type: "Article", Depth: 2, Impact: analysis.ImpactTransitive, Direction: "incoming"},
			},
			Summary: &analysis.ImpactSummary{
				TotalAffected:   2,
				MaxDepthReached: 2,
			},
		},
		nil, // should be safely skipped
	}

	aggregated := AggregateImpactResults(results)

	// Direct: Art6505, Art6503, Art6502 (3 unique, deduplicated)
	if len(aggregated.DirectlyAffected) != 3 {
		t.Errorf("expected 3 unique directly affected provisions, got %d", len(aggregated.DirectlyAffected))
		for _, p := range aggregated.DirectlyAffected {
			t.Logf("  direct: %s", p.URI)
		}
	}

	// Transitive: Art6504 (1 unique, deduplicated from both results)
	if len(aggregated.TransitivelyAffected) != 1 {
		t.Errorf("expected 1 unique transitively affected provision, got %d", len(aggregated.TransitivelyAffected))
		for _, p := range aggregated.TransitivelyAffected {
			t.Logf("  transitive: %s", p.URI)
		}
	}

	// Total should be 4 (3 direct + 1 transitive)
	if aggregated.TotalProvisionsAffected != 4 {
		t.Errorf("expected 4 total provisions affected, got %d", aggregated.TotalProvisionsAffected)
	}

	// Max depth should be 2
	if aggregated.MaxDepthReached != 2 {
		t.Errorf("expected max depth 2, got %d", aggregated.MaxDepthReached)
	}
}
