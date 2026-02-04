package draft

import (
	"testing"

	"github.com/coolbeans/regula/pkg/store"
)

// buildCrossRefTriples creates a triple set with varied cross-reference
// patterns for testing broken reference detection. Includes articles with
// typed references (refersToArticle), bidirectional references, and provisions
// with no incoming references.
func buildCrossRefTriples() []store.Triple {
	baseURI := "https://regula.dev/regulations/"
	regID := "US-USC-TITLE-15"
	regURI := baseURI + regID

	art6502URI := baseURI + regID + ":Art6502"
	art6503URI := baseURI + regID + ":Art6503"
	art6504URI := baseURI + regID + ":Art6504"
	art6505URI := baseURI + regID + ":Art6505"
	art6506URI := baseURI + regID + ":Art6506"

	return []store.Triple{
		// Regulation
		{Subject: regURI, Predicate: store.RDFType, Object: store.ClassRegulation},
		{Subject: regURI, Predicate: store.PropTitle, Object: "Title 15 - Commerce and Trade"},

		// Article 6502 — core provision, referenced by many
		{Subject: art6502URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art6502URI, Predicate: store.PropNumber, Object: "6502"},
		{Subject: art6502URI, Predicate: store.PropTitle, Object: "Regulation of unfair acts"},
		{Subject: art6502URI, Predicate: store.PropText, Object: "It shall be unlawful to collect personal information."},
		{Subject: art6502URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art6502URI},
		{Subject: regURI, Predicate: store.PropHasArticle, Object: art6502URI},

		// Article 6503 — safe harbors, referenced by 6504
		{Subject: art6503URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art6503URI, Predicate: store.PropNumber, Object: "6503"},
		{Subject: art6503URI, Predicate: store.PropTitle, Object: "Safe harbors"},
		{Subject: art6503URI, Predicate: store.PropText, Object: "Industry self-regulation as safe harbor."},
		{Subject: art6503URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art6503URI},
		{Subject: regURI, Predicate: store.PropHasArticle, Object: art6503URI},

		// Article 6504 — enforcement, references 6502 and 6503
		{Subject: art6504URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art6504URI, Predicate: store.PropNumber, Object: "6504"},
		{Subject: art6504URI, Predicate: store.PropTitle, Object: "Actions by States"},
		{Subject: art6504URI, Predicate: store.PropText, Object: "State attorney general may bring civil action."},
		{Subject: art6504URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art6504URI},
		{Subject: regURI, Predicate: store.PropHasArticle, Object: art6504URI},
		{Subject: art6504URI, Predicate: store.PropReferences, Object: art6502URI},
		{Subject: art6502URI, Predicate: store.PropReferencedBy, Object: art6504URI},
		{Subject: art6504URI, Predicate: store.PropReferences, Object: art6503URI},
		{Subject: art6503URI, Predicate: store.PropReferencedBy, Object: art6504URI},
		// Also typed reference
		{Subject: art6504URI, Predicate: store.PropRefersToArticle, Object: art6503URI},

		// Article 6505 — penalties, references 6502
		{Subject: art6505URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art6505URI, Predicate: store.PropNumber, Object: "6505"},
		{Subject: art6505URI, Predicate: store.PropTitle, Object: "Civil penalties"},
		{Subject: art6505URI, Predicate: store.PropText, Object: "Operator who violates shall be subject to penalties."},
		{Subject: art6505URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art6505URI},
		{Subject: regURI, Predicate: store.PropHasArticle, Object: art6505URI},
		{Subject: art6505URI, Predicate: store.PropReferences, Object: art6502URI},
		{Subject: art6502URI, Predicate: store.PropReferencedBy, Object: art6505URI},

		// Article 6506 — standalone provision, no incoming references
		{Subject: art6506URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art6506URI, Predicate: store.PropNumber, Object: "6506"},
		{Subject: art6506URI, Predicate: store.PropTitle, Object: "Definitions"},
		{Subject: art6506URI, Predicate: store.PropText, Object: "For purposes of this chapter, the following definitions apply."},
		{Subject: art6506URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art6506URI},
		{Subject: regURI, Predicate: store.PropHasArticle, Object: art6506URI},
	}
}

func TestDetectBrokenCrossRefs_Repeal(t *testing.T) {
	triples := buildCrossRefTriples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	// Repeal section 6503 — referenced by 6504 (both generic and typed refs)
	bill := &DraftBill{
		BillNumber: "H.R. 1111",
		Title:      "Repeal Broken Refs Act",
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

	brokenRefs, err := DetectBrokenCrossRefs(diff, libraryPath)
	if err != nil {
		t.Fatalf("DetectBrokenCrossRefs failed: %v", err)
	}

	// 6504 references 6503 — should produce broken references
	if len(brokenRefs) == 0 {
		t.Fatal("expected broken cross-references from repeal of 6503")
	}

	// All should be SeverityError for repeal
	for _, ref := range brokenRefs {
		if ref.Severity != SeverityError {
			t.Errorf("expected SeverityError for repeal, got %s for source %q", ref.Severity, ref.SourceURI)
		}
		if ref.TargetURI != "https://regula.dev/regulations/US-USC-TITLE-15:Art6503" {
			t.Errorf("expected target Art6503, got %q", ref.TargetURI)
		}
	}

	// Verify 6504 is among the broken ref sources
	sourceURIs := make(map[string]bool)
	for _, ref := range brokenRefs {
		sourceURIs[ref.SourceURI] = true
	}
	if !sourceURIs["https://regula.dev/regulations/US-USC-TITLE-15:Art6504"] {
		t.Error("expected Art6504 as a broken ref source (it references Art6503)")
	}

	// Verify labels are resolved
	for _, ref := range brokenRefs {
		if ref.SourceLabel == "" {
			t.Errorf("expected non-empty source label for %q", ref.SourceURI)
		}
		if ref.TargetLabel == "" {
			t.Errorf("expected non-empty target label for %q", ref.TargetURI)
		}
	}

	// Verify results are sorted by severity (errors first)
	for i := 1; i < len(brokenRefs); i++ {
		if brokenRefs[i].Severity < brokenRefs[i-1].Severity {
			t.Error("expected broken refs sorted by severity (errors first)")
			break
		}
	}
}

func TestDetectBrokenCrossRefs_Modification(t *testing.T) {
	triples := buildCrossRefTriples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	// Modify section 6502 via strike-insert — referenced by 6504 and 6505
	bill := &DraftBill{
		BillNumber: "H.R. 2222",
		Title:      "Modification Broken Refs Act",
		Sections: []*DraftSection{
			{
				Number: "1",
				Title:  "Amend COPPA",
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

	brokenRefs, err := DetectBrokenCrossRefs(diff, libraryPath)
	if err != nil {
		t.Fatalf("DetectBrokenCrossRefs failed: %v", err)
	}

	// 6504 and 6505 both reference 6502 — should produce warnings
	if len(brokenRefs) == 0 {
		t.Fatal("expected broken cross-references from modification of 6502")
	}

	// All should be SeverityWarning for strike-insert
	for _, ref := range brokenRefs {
		if ref.Severity != SeverityWarning {
			t.Errorf("expected SeverityWarning for strike-insert, got %s for source %q", ref.Severity, ref.SourceURI)
		}
	}

	// Verify both 6504 and 6505 are among the broken ref sources
	sourceURIs := make(map[string]bool)
	for _, ref := range brokenRefs {
		sourceURIs[ref.SourceURI] = true
	}
	if !sourceURIs["https://regula.dev/regulations/US-USC-TITLE-15:Art6504"] {
		t.Error("expected Art6504 as a broken ref source")
	}
	if !sourceURIs["https://regula.dev/regulations/US-USC-TITLE-15:Art6505"] {
		t.Error("expected Art6505 as a broken ref source")
	}
}

func TestDetectBrokenCrossRefs_Redesignation(t *testing.T) {
	triples := buildCrossRefTriples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	// Redesignate section 6503 — old identifier becomes invalid
	bill := &DraftBill{
		BillNumber: "H.R. 3333",
		Title:      "Redesignation Broken Refs Act",
		Sections: []*DraftSection{
			{
				Number: "1",
				Title:  "Redesignate section 6503",
				Amendments: []Amendment{
					{
						Type:          AmendRedesignate,
						TargetTitle:   "15",
						TargetSection: "6503",
						InsertText:    "6503A",
						Description:   "redesignate section 6503 as section 6503A",
					},
				},
			},
		},
	}

	diff, err := ComputeDiff(bill, libraryPath)
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	brokenRefs, err := DetectBrokenCrossRefs(diff, libraryPath)
	if err != nil {
		t.Fatalf("DetectBrokenCrossRefs failed: %v", err)
	}

	// 6504 references 6503 — redesignation makes old ID invalid
	if len(brokenRefs) == 0 {
		t.Fatal("expected broken cross-references from redesignation of 6503")
	}

	// All should be SeverityError for redesignation
	for _, ref := range brokenRefs {
		if ref.Severity != SeverityError {
			t.Errorf("expected SeverityError for redesignation, got %s for source %q", ref.Severity, ref.SourceURI)
		}
	}

	// Verify reason mentions redesignation
	for _, ref := range brokenRefs {
		if ref.Reason == "" {
			t.Error("expected non-empty reason for broken reference")
		}
	}
}

func TestDetectBrokenCrossRefs_NoBreaks(t *testing.T) {
	triples := buildCrossRefTriples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	// Add a new section — no existing provisions reference it, so no broken refs
	bill := &DraftBill{
		BillNumber: "H.R. 4444",
		Title:      "No Broken Refs Act",
		Sections: []*DraftSection{
			{
				Number: "1",
				Title:  "Add new section",
				Amendments: []Amendment{
					{
						Type:          AmendAddNewSection,
						TargetTitle:   "15",
						TargetSection: "6510",
						InsertText:    "The Commission shall establish a youth program.",
					},
				},
			},
		},
	}

	diff, err := ComputeDiff(bill, libraryPath)
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	brokenRefs, err := DetectBrokenCrossRefs(diff, libraryPath)
	if err != nil {
		t.Fatalf("DetectBrokenCrossRefs failed: %v", err)
	}

	// New section has no existing references — no broken refs
	if len(brokenRefs) != 0 {
		t.Errorf("expected 0 broken references for new section addition, got %d", len(brokenRefs))
		for _, ref := range brokenRefs {
			t.Logf("  unexpected broken ref: %s -> %s (%s)", ref.SourceURI, ref.TargetURI, ref.Severity)
		}
	}
}

func TestDetectBrokenCrossRefs_NoBreaks_StandaloneRepeal(t *testing.T) {
	triples := buildCrossRefTriples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	// Repeal section 6506 which has no incoming references
	bill := &DraftBill{
		BillNumber: "H.R. 4445",
		Title:      "No Breaks Repeal Act",
		Sections: []*DraftSection{
			{
				Number: "1",
				Title:  "Repeal definitions",
				Amendments: []Amendment{
					{
						Type:          AmendRepeal,
						TargetTitle:   "15",
						TargetSection: "6506",
					},
				},
			},
		},
	}

	diff, err := ComputeDiff(bill, libraryPath)
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	brokenRefs, err := DetectBrokenCrossRefs(diff, libraryPath)
	if err != nil {
		t.Fatalf("DetectBrokenCrossRefs failed: %v", err)
	}

	// 6506 has no incoming references — repeal should produce no broken refs
	if len(brokenRefs) != 0 {
		t.Errorf("expected 0 broken references for standalone section repeal, got %d", len(brokenRefs))
	}
}

func TestClassifyBreakSeverity(t *testing.T) {
	tests := []struct {
		name     string
		amendment Amendment
		expected BrokenRefSeverity
	}{
		{
			name:     "repeal is error",
			amendment: Amendment{Type: AmendRepeal},
			expected: SeverityError,
		},
		{
			name:     "redesignate is error",
			amendment: Amendment{Type: AmendRedesignate},
			expected: SeverityError,
		},
		{
			name:     "strike-insert is warning",
			amendment: Amendment{Type: AmendStrikeInsert},
			expected: SeverityWarning,
		},
		{
			name:     "add-at-end is info",
			amendment: Amendment{Type: AmendAddAtEnd},
			expected: SeverityInfo,
		},
		{
			name:     "add-new-section is info",
			amendment: Amendment{Type: AmendAddNewSection},
			expected: SeverityInfo,
		},
		{
			name:     "table-of-contents is info",
			amendment: Amendment{Type: AmendTableOfContents},
			expected: SeverityInfo,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			severity := ClassifyBreakSeverity(testCase.amendment)
			if severity != testCase.expected {
				t.Errorf("expected %s, got %s", testCase.expected, severity)
			}
		})
	}
}

func TestBrokenRefSeverity_String(t *testing.T) {
	tests := []struct {
		severity BrokenRefSeverity
		expected string
	}{
		{SeverityError, "error"},
		{SeverityWarning, "warning"},
		{SeverityInfo, "info"},
		{BrokenRefSeverity(99), "unknown"},
	}

	for _, testCase := range tests {
		result := testCase.severity.String()
		if result != testCase.expected {
			t.Errorf("BrokenRefSeverity(%d).String() = %q, expected %q",
				testCase.severity, result, testCase.expected)
		}
	}
}

func TestDetectBrokenCrossRefs_MixedSeverities(t *testing.T) {
	triples := buildCrossRefTriples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	// Repeal 6503 (error) and modify 6502 (warning) in the same bill
	bill := &DraftBill{
		BillNumber: "H.R. 5555",
		Title:      "Mixed Severity Act",
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

	brokenRefs, err := DetectBrokenCrossRefs(diff, libraryPath)
	if err != nil {
		t.Fatalf("DetectBrokenCrossRefs failed: %v", err)
	}

	// Should have both error and warning severity refs
	hasError := false
	hasWarning := false
	for _, ref := range brokenRefs {
		if ref.Severity == SeverityError {
			hasError = true
		}
		if ref.Severity == SeverityWarning {
			hasWarning = true
		}
	}

	if !hasError {
		t.Error("expected at least one SeverityError from repeal")
	}
	if !hasWarning {
		t.Error("expected at least one SeverityWarning from modification")
	}

	// Verify sort order: errors before warnings
	for i := 1; i < len(brokenRefs); i++ {
		if brokenRefs[i].Severity < brokenRefs[i-1].Severity {
			t.Errorf("broken refs not sorted by severity: index %d is %s, index %d is %s",
				i-1, brokenRefs[i-1].Severity, i, brokenRefs[i].Severity)
			break
		}
	}
}
