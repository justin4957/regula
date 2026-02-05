package draft

import (
	"testing"

	"github.com/coolbeans/regula/pkg/store"
)

// buildObligationTriples creates a triple set with obligations and rights for
// testing conflict detection. Includes provisions with matching obligation types
// (for duplicate detection), cross-references (for orphan detection), and
// contrasting directive text.
func buildObligationTriples() []store.Triple {
	baseURI := "https://regula.dev/regulations/"
	regID := "US-USC-TITLE-15"
	regURI := baseURI + regID

	art6502URI := baseURI + regID + ":Art6502"
	art6503URI := baseURI + regID + ":Art6503"
	art6504URI := baseURI + regID + ":Art6504"
	art6505URI := baseURI + regID + ":Art6505"

	oblig6502URI := baseURI + regID + ":Obligation:6502:InformationProvisionObligation"
	oblig6503URI := baseURI + regID + ":Obligation:6503:DataMinimizationObligation"
	oblig6504URI := baseURI + regID + ":Obligation:6504:EnforcementObligation"
	oblig6505URI := baseURI + regID + ":Obligation:6505:InformationProvisionObligation"

	return []store.Triple{
		// Regulation
		{Subject: regURI, Predicate: store.RDFType, Object: store.ClassRegulation},
		{Subject: regURI, Predicate: store.PropTitle, Object: "Title 15 - Commerce and Trade"},

		// Article 6502 — has InformationProvisionObligation ("shall provide notice")
		{Subject: art6502URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art6502URI, Predicate: store.PropNumber, Object: "6502"},
		{Subject: art6502URI, Predicate: store.PropTitle, Object: "Notice requirements"},
		{Subject: art6502URI, Predicate: store.PropText, Object: "The operator shall provide clear and prominent notice of information collection practices."},
		{Subject: art6502URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art6502URI},
		{Subject: regURI, Predicate: store.PropHasArticle, Object: art6502URI},
		// Obligation on 6502
		{Subject: oblig6502URI, Predicate: store.RDFType, Object: store.ClassObligation},
		{Subject: oblig6502URI, Predicate: "reg:obligationType", Object: "InformationProvisionObligation"},
		{Subject: oblig6502URI, Predicate: store.PropText, Object: "shall provide clear and prominent notice of information collection practices"},
		{Subject: oblig6502URI, Predicate: store.PropPartOf, Object: art6502URI},
		{Subject: oblig6502URI, Predicate: store.PropBelongsTo, Object: regURI},
		{Subject: art6502URI, Predicate: store.PropImposesObligation, Object: oblig6502URI},

		// Article 6503 — has DataMinimizationObligation ("shall not collect more than necessary")
		{Subject: art6503URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art6503URI, Predicate: store.PropNumber, Object: "6503"},
		{Subject: art6503URI, Predicate: store.PropTitle, Object: "Data minimization"},
		{Subject: art6503URI, Predicate: store.PropText, Object: "The operator shall not collect personal information beyond what is necessary for the stated purpose."},
		{Subject: art6503URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art6503URI},
		{Subject: regURI, Predicate: store.PropHasArticle, Object: art6503URI},
		// Obligation on 6503
		{Subject: oblig6503URI, Predicate: store.RDFType, Object: store.ClassObligation},
		{Subject: oblig6503URI, Predicate: "reg:obligationType", Object: "DataMinimizationObligation"},
		{Subject: oblig6503URI, Predicate: store.PropText, Object: "shall not collect personal information beyond what is necessary"},
		{Subject: oblig6503URI, Predicate: store.PropPartOf, Object: art6503URI},
		{Subject: oblig6503URI, Predicate: store.PropBelongsTo, Object: regURI},
		{Subject: art6503URI, Predicate: store.PropImposesObligation, Object: oblig6503URI},

		// Article 6504 — enforcement, references 6502 and 6503
		{Subject: art6504URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art6504URI, Predicate: store.PropNumber, Object: "6504"},
		{Subject: art6504URI, Predicate: store.PropTitle, Object: "Enforcement"},
		{Subject: art6504URI, Predicate: store.PropText, Object: "Violations of sections 6502 and 6503 shall be enforced by the Commission."},
		{Subject: art6504URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art6504URI},
		{Subject: regURI, Predicate: store.PropHasArticle, Object: art6504URI},
		{Subject: art6504URI, Predicate: store.PropReferences, Object: art6502URI},
		{Subject: art6502URI, Predicate: store.PropReferencedBy, Object: art6504URI},
		{Subject: art6504URI, Predicate: store.PropReferences, Object: art6503URI},
		{Subject: art6503URI, Predicate: store.PropReferencedBy, Object: art6504URI},
		// Obligation on 6504
		{Subject: oblig6504URI, Predicate: store.RDFType, Object: store.ClassObligation},
		{Subject: oblig6504URI, Predicate: "reg:obligationType", Object: "EnforcementObligation"},
		{Subject: oblig6504URI, Predicate: store.PropText, Object: "shall be enforced by the Commission"},
		{Subject: oblig6504URI, Predicate: store.PropPartOf, Object: art6504URI},
		{Subject: oblig6504URI, Predicate: store.PropBelongsTo, Object: regURI},
		{Subject: art6504URI, Predicate: store.PropImposesObligation, Object: oblig6504URI},

		// Article 6505 — same obligation type as 6502 (for duplicate detection)
		{Subject: art6505URI, Predicate: store.RDFType, Object: store.ClassArticle},
		{Subject: art6505URI, Predicate: store.PropNumber, Object: "6505"},
		{Subject: art6505URI, Predicate: store.PropTitle, Object: "Additional notice requirements"},
		{Subject: art6505URI, Predicate: store.PropText, Object: "The operator shall provide notice of data sharing practices to consumers."},
		{Subject: art6505URI, Predicate: store.PropPartOf, Object: regURI},
		{Subject: regURI, Predicate: store.PropContains, Object: art6505URI},
		{Subject: regURI, Predicate: store.PropHasArticle, Object: art6505URI},
		// Obligation on 6505 — same type as 6502
		{Subject: oblig6505URI, Predicate: store.RDFType, Object: store.ClassObligation},
		{Subject: oblig6505URI, Predicate: "reg:obligationType", Object: "InformationProvisionObligation"},
		{Subject: oblig6505URI, Predicate: store.PropText, Object: "shall provide notice of data sharing practices to consumers"},
		{Subject: oblig6505URI, Predicate: store.PropPartOf, Object: art6505URI},
		{Subject: oblig6505URI, Predicate: store.PropBelongsTo, Object: regURI},
		{Subject: art6505URI, Predicate: store.PropImposesObligation, Object: oblig6505URI},
	}
}

func TestDetectObligationContradiction(t *testing.T) {
	triples := buildObligationTriples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	// Strike-insert on 6502 that changes "shall provide notice" to "shall not provide notice"
	bill := &DraftBill{
		BillNumber: "H.R. 7001",
		Title:      "Contradiction Test Act",
		Sections: []*DraftSection{
			{
				Number: "1",
				Title:  "Amend notice requirements",
				Amendments: []Amendment{
					{
						Type:          AmendStrikeInsert,
						TargetTitle:   "15",
						TargetSection: "6502",
						StrikeText:    "shall provide clear and prominent notice",
						InsertText:    "shall not provide notice of information collection practices",
					},
				},
			},
		},
	}

	diff, err := ComputeDiff(bill, libraryPath)
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	report, err := DetectObligationConflicts(diff, nil, libraryPath)
	if err != nil {
		t.Fatalf("DetectObligationConflicts failed: %v", err)
	}

	// Should detect contradiction: existing "shall provide" vs proposed "shall not provide"
	if len(report.Conflicts) == 0 {
		t.Fatal("expected obligation contradiction to be detected")
	}

	foundContradiction := false
	for _, conflict := range report.Conflicts {
		if conflict.Type == ConflictObligationContradiction {
			foundContradiction = true
			if conflict.Severity != ConflictError {
				t.Errorf("expected ConflictError severity for contradiction, got %s", conflict.Severity)
			}
			if conflict.ExistingProvision == "" {
				t.Error("expected non-empty existing provision URI")
			}
			if conflict.Description == "" {
				t.Error("expected non-empty description")
			}
			t.Logf("Detected contradiction: %s", conflict.Description)
		}
	}

	if !foundContradiction {
		t.Error("expected ConflictObligationContradiction type in conflicts")
		for _, conflict := range report.Conflicts {
			t.Logf("  conflict: type=%s severity=%s desc=%q", conflict.Type, conflict.Severity, conflict.Description)
		}
	}
}

func TestDetectObligationDuplicate(t *testing.T) {
	triples := buildObligationTriples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	// Add new section with an obligation that duplicates 6502's "shall provide notice"
	bill := &DraftBill{
		BillNumber: "H.R. 7002",
		Title:      "Duplicate Obligation Test Act",
		Sections: []*DraftSection{
			{
				Number: "1",
				Title:  "New notice section",
				Amendments: []Amendment{
					{
						Type:          AmendAddNewSection,
						TargetTitle:   "15",
						TargetSection: "6510",
						InsertText:    "The operator shall provide clear notice of information collection practices to all users.",
					},
				},
			},
		},
	}

	diff, err := ComputeDiff(bill, libraryPath)
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	report, err := DetectObligationConflicts(diff, nil, libraryPath)
	if err != nil {
		t.Fatalf("DetectObligationConflicts failed: %v", err)
	}

	// Should detect duplicate: new obligation matches existing "shall provide notice" type
	foundDuplicate := false
	for _, conflict := range report.Conflicts {
		if conflict.Type == ConflictObligationDuplicate {
			foundDuplicate = true
			if conflict.Severity != ConflictInfo {
				t.Errorf("expected ConflictInfo severity for duplicate, got %s", conflict.Severity)
			}
			t.Logf("Detected duplicate: %s", conflict.Description)
		}
	}

	if !foundDuplicate {
		t.Error("expected ConflictObligationDuplicate to be detected")
		for _, conflict := range report.Conflicts {
			t.Logf("  conflict: type=%s severity=%s desc=%q", conflict.Type, conflict.Severity, conflict.Description)
		}
	}
}

func TestDetectObligationOrphaned(t *testing.T) {
	triples := buildObligationTriples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	// Repeal section 6502 which has obligations referenced by 6504
	bill := &DraftBill{
		BillNumber: "H.R. 7003",
		Title:      "Orphaned Obligation Test Act",
		Sections: []*DraftSection{
			{
				Number: "1",
				Title:  "Repeal notice requirements",
				Amendments: []Amendment{
					{
						Type:          AmendRepeal,
						TargetTitle:   "15",
						TargetSection: "6502",
						Description:   "Section 6502 of title 15 is repealed",
					},
				},
			},
		},
	}

	diff, err := ComputeDiff(bill, libraryPath)
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	report, err := DetectObligationConflicts(diff, nil, libraryPath)
	if err != nil {
		t.Fatalf("DetectObligationConflicts failed: %v", err)
	}

	// Should detect orphan: 6504 references 6502 which is being repealed
	foundOrphaned := false
	for _, conflict := range report.Conflicts {
		if conflict.Type == ConflictObligationOrphaned {
			foundOrphaned = true
			if conflict.Severity != ConflictWarning {
				t.Errorf("expected ConflictWarning severity for orphaned, got %s", conflict.Severity)
			}
			if conflict.Description == "" {
				t.Error("expected non-empty description")
			}
			t.Logf("Detected orphaned obligation: %s", conflict.Description)
		}
	}

	if !foundOrphaned {
		t.Error("expected ConflictObligationOrphaned to be detected")
		for _, conflict := range report.Conflicts {
			t.Logf("  conflict: type=%s severity=%s desc=%q", conflict.Type, conflict.Severity, conflict.Description)
		}
	}
}

func TestDetectObligationConflicts_NoConflicts(t *testing.T) {
	triples := buildObligationTriples()
	_, libraryPath := testLibrary(t, "us-usc-title-15", triples)

	// Add a new section with a completely different obligation type
	bill := &DraftBill{
		BillNumber: "H.R. 7004",
		Title:      "Clean Bill Test Act",
		Sections: []*DraftSection{
			{
				Number: "1",
				Title:  "New reporting section",
				Amendments: []Amendment{
					{
						Type:          AmendAddNewSection,
						TargetTitle:   "15",
						TargetSection: "6520",
						InsertText:    "The Commission shall submit an annual report to Congress on enforcement activities.",
					},
				},
			},
		},
	}

	diff, err := ComputeDiff(bill, libraryPath)
	if err != nil {
		t.Fatalf("ComputeDiff failed: %v", err)
	}

	report, err := DetectObligationConflicts(diff, nil, libraryPath)
	if err != nil {
		t.Fatalf("DetectObligationConflicts failed: %v", err)
	}

	if len(report.Conflicts) != 0 {
		t.Errorf("expected 0 conflicts for clean bill, got %d", len(report.Conflicts))
		for _, conflict := range report.Conflicts {
			t.Logf("  unexpected conflict: type=%s severity=%s desc=%q",
				conflict.Type, conflict.Severity, conflict.Description)
		}
	}

	if report.Summary.TotalConflicts != 0 {
		t.Errorf("expected 0 total conflicts in summary, got %d", report.Summary.TotalConflicts)
	}
}

func TestConflictSeverity(t *testing.T) {
	tests := []struct {
		name         string
		conflictType ConflictType
		expected     ConflictSeverity
	}{
		{
			name:         "contradiction is error",
			conflictType: ConflictObligationContradiction,
			expected:     ConflictError,
		},
		{
			name:         "rights contradiction is error",
			conflictType: ConflictRightsContradiction,
			expected:     ConflictError,
		},
		{
			name:         "orphaned is warning",
			conflictType: ConflictObligationOrphaned,
			expected:     ConflictWarning,
		},
		{
			name:         "rights narrowing is warning",
			conflictType: ConflictRightsNarrowing,
			expected:     ConflictWarning,
		},
		{
			name:         "duplicate is info",
			conflictType: ConflictObligationDuplicate,
			expected:     ConflictInfo,
		},
		{
			name:         "rights expansion is info",
			conflictType: ConflictRightsExpansion,
			expected:     ConflictInfo,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			severity := classifyConflictSeverity(testCase.conflictType)
			if severity != testCase.expected {
				t.Errorf("expected %s, got %s", testCase.expected, severity)
			}
		})
	}
}

func TestConflictType_String(t *testing.T) {
	tests := []struct {
		conflictType ConflictType
		expected     string
	}{
		{ConflictObligationContradiction, "obligation_contradiction"},
		{ConflictObligationDuplicate, "obligation_duplicate"},
		{ConflictObligationOrphaned, "obligation_orphaned"},
		{ConflictRightsNarrowing, "rights_narrowing"},
		{ConflictRightsContradiction, "rights_contradiction"},
		{ConflictRightsExpansion, "rights_expansion"},
		{ConflictType(99), "unknown"},
	}

	for _, testCase := range tests {
		result := testCase.conflictType.String()
		if result != testCase.expected {
			t.Errorf("ConflictType(%d).String() = %q, expected %q",
				testCase.conflictType, result, testCase.expected)
		}
	}
}

func TestConflictSeverity_String(t *testing.T) {
	tests := []struct {
		severity ConflictSeverity
		expected string
	}{
		{ConflictError, "error"},
		{ConflictWarning, "warning"},
		{ConflictInfo, "info"},
		{ConflictSeverity(99), "unknown"},
	}

	for _, testCase := range tests {
		result := testCase.severity.String()
		if result != testCase.expected {
			t.Errorf("ConflictSeverity(%d).String() = %q, expected %q",
				testCase.severity, result, testCase.expected)
		}
	}
}

func TestDetectObligationContradiction_Unit(t *testing.T) {
	tests := []struct {
		name         string
		draftText    string
		existingText string
		expected     bool
	}{
		{
			name:         "shall vs shall not — same subject",
			draftText:    "shall not provide notice of information collection",
			existingText: "shall provide clear notice of information collection practices",
			expected:     true,
		},
		{
			name:         "must vs must not — same subject",
			draftText:    "must not disclose personal information to third parties",
			existingText: "must disclose personal information to the consumer",
			expected:     true,
		},
		{
			name:         "no contradiction — different subjects",
			draftText:    "shall provide annual reports to Congress",
			existingText: "shall not collect biometric data from minors",
			expected:     false,
		},
		{
			name:         "no contradiction — same polarity",
			draftText:    "shall provide notice of collection",
			existingText: "shall provide notice of sharing",
			expected:     false,
		},
		{
			name:         "empty text",
			draftText:    "",
			existingText: "shall provide notice",
			expected:     false,
		},
		{
			name:         "is required vs is prohibited — same subject",
			draftText:    "is prohibited from collecting personal data without consent",
			existingText: "is required to collect personal data for verification purposes",
			expected:     true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := DetectObligationContradiction(testCase.draftText, testCase.existingText)
			if result != testCase.expected {
				t.Errorf("DetectObligationContradiction() = %v, expected %v", result, testCase.expected)
			}
		})
	}
}

func TestBuildConflictSummary(t *testing.T) {
	conflicts := []Conflict{
		{Type: ConflictObligationContradiction, Severity: ConflictError},
		{Type: ConflictObligationContradiction, Severity: ConflictError},
		{Type: ConflictObligationOrphaned, Severity: ConflictWarning},
		{Type: ConflictObligationDuplicate, Severity: ConflictInfo},
		{Type: ConflictObligationDuplicate, Severity: ConflictInfo},
		{Type: ConflictObligationDuplicate, Severity: ConflictInfo},
	}

	summary := buildConflictSummary(conflicts)

	if summary.TotalConflicts != 6 {
		t.Errorf("expected 6 total conflicts, got %d", summary.TotalConflicts)
	}
	if summary.Errors != 2 {
		t.Errorf("expected 2 errors, got %d", summary.Errors)
	}
	if summary.Warnings != 1 {
		t.Errorf("expected 1 warning, got %d", summary.Warnings)
	}
	if summary.Infos != 3 {
		t.Errorf("expected 3 infos, got %d", summary.Infos)
	}
	if summary.ByType[ConflictObligationContradiction] != 2 {
		t.Errorf("expected 2 contradictions, got %d", summary.ByType[ConflictObligationContradiction])
	}
	if summary.ByType[ConflictObligationOrphaned] != 1 {
		t.Errorf("expected 1 orphaned, got %d", summary.ByType[ConflictObligationOrphaned])
	}
	if summary.ByType[ConflictObligationDuplicate] != 3 {
		t.Errorf("expected 3 duplicates, got %d", summary.ByType[ConflictObligationDuplicate])
	}
}

func TestSortConflicts(t *testing.T) {
	conflicts := []Conflict{
		{Type: ConflictObligationDuplicate, Severity: ConflictInfo},
		{Type: ConflictObligationContradiction, Severity: ConflictError},
		{Type: ConflictObligationOrphaned, Severity: ConflictWarning},
		{Type: ConflictObligationContradiction, Severity: ConflictError},
	}

	sortConflicts(conflicts)

	// Errors should come first
	if conflicts[0].Severity != ConflictError {
		t.Errorf("expected first conflict to be error, got %s", conflicts[0].Severity)
	}
	if conflicts[1].Severity != ConflictError {
		t.Errorf("expected second conflict to be error, got %s", conflicts[1].Severity)
	}
	// Warning next
	if conflicts[2].Severity != ConflictWarning {
		t.Errorf("expected third conflict to be warning, got %s", conflicts[2].Severity)
	}
	// Info last
	if conflicts[3].Severity != ConflictInfo {
		t.Errorf("expected fourth conflict to be info, got %s", conflicts[3].Severity)
	}
}

func TestFindDependentObligations(t *testing.T) {
	triples := buildObligationTriples()

	tripleStore := store.NewTripleStore()
	if err := tripleStore.BulkAdd(triples); err != nil {
		t.Fatalf("failed to bulk add triples: %v", err)
	}

	baseURI := "https://regula.dev/regulations/"
	regID := "US-USC-TITLE-15"

	// Obligation on 6502 should have dependents (6504 references 6502)
	oblig6502URI := baseURI + regID + ":Obligation:6502:InformationProvisionObligation"
	dependents := FindDependentObligations(oblig6502URI, tripleStore)

	if len(dependents) == 0 {
		t.Fatal("expected dependent provisions for obligation on 6502")
	}

	// 6504 references 6502, so it should be a dependent
	art6504URI := baseURI + regID + ":Art6504"
	foundDependent := false
	for _, depURI := range dependents {
		if depURI == art6504URI {
			foundDependent = true
			break
		}
	}

	if !foundDependent {
		t.Errorf("expected Art6504 as dependent of obligation on 6502, got: %v", dependents)
	}
}
