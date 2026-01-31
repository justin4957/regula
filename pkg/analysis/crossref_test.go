package analysis

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/coolbeans/regula/pkg/store"
)

// buildTestStore creates a minimal TripleStore with known data for testing.
func buildTestStore(articles int, definitions []string, rights []string, obligations []string, externalRefs []string) *store.TripleStore {
	tripleStore := store.NewTripleStore()
	baseURI := "https://regula.dev/test#"

	// Add articles
	for i := 1; i <= articles; i++ {
		articleURI := baseURI + "Art" + itoa(i)
		tripleStore.Add(articleURI, store.RDFType, store.ClassArticle)
		tripleStore.Add(articleURI, store.PropTitle, "Article "+itoa(i))
	}

	// Add definitions
	for _, defTerm := range definitions {
		termURI := baseURI + "Term:" + strings.ReplaceAll(defTerm, " ", "_")
		tripleStore.Add(termURI, store.RDFType, store.ClassDefinedTerm)
		tripleStore.Add(termURI, store.PropNormalizedTerm, strings.ToLower(defTerm))
		tripleStore.Add(termURI, store.PropTerm, defTerm)
	}

	// Add rights
	for _, right := range rights {
		rightURI := baseURI + "Right:" + strings.ReplaceAll(right, " ", "_")
		tripleStore.Add(baseURI+"Art1", store.PropGrantsRight, rightURI)
	}

	// Add obligations
	for _, obligation := range obligations {
		obligationURI := baseURI + "Obligation:" + strings.ReplaceAll(obligation, " ", "_")
		tripleStore.Add(baseURI+"Art1", store.PropImposesObligation, obligationURI)
	}

	// Add external refs
	for i, ref := range externalRefs {
		articleURI := baseURI + "Art" + itoa((i%articles)+1)
		tripleStore.Add(articleURI, store.PropExternalRef, ref)
	}

	// Add some internal references
	if articles > 1 {
		tripleStore.Add(baseURI+"Art1", store.PropReferences, baseURI+"Art2")
	}

	return tripleStore
}

// itoa is a simple int to string helper for test data.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	digits := ""
	for i > 0 {
		digits = string(rune('0'+i%10)) + digits
		i /= 10
	}
	return digits
}

func TestNewCrossRefAnalyzer(t *testing.T) {
	analyzer := NewCrossRefAnalyzer()
	if analyzer == nil {
		t.Fatal("NewCrossRefAnalyzer returned nil")
	}
	if len(analyzer.stores) != 0 {
		t.Errorf("expected empty stores, got %d", len(analyzer.stores))
	}
}

func TestCrossRefAnalyzer_AddDocument(t *testing.T) {
	analyzer := NewCrossRefAnalyzer()
	tripleStore := store.NewTripleStore()
	tripleStore.Add("s", "p", "o")

	analyzer.AddDocument("gdpr", "GDPR", tripleStore)

	if len(analyzer.stores) != 1 {
		t.Errorf("expected 1 store, got %d", len(analyzer.stores))
	}
	if analyzer.labels["gdpr"] != "GDPR" {
		t.Errorf("expected label GDPR, got %s", analyzer.labels["gdpr"])
	}
}

func TestAnalyzeExternalRefs_SingleDoc(t *testing.T) {
	analyzer := NewCrossRefAnalyzer()
	tripleStore := buildTestStore(3, nil, nil, nil, []string{
		"Directive 95/46/EC",
		"Directive 95/46/EC",
		"Regulation (EC) No 45/2001",
		"Directive 2002/58/EC",
	})
	analyzer.AddDocument("gdpr", "GDPR", tripleStore)

	report := analyzer.AnalyzeExternalRefs("gdpr")

	if report.TotalExternalRefs != 4 {
		t.Errorf("expected 4 total external refs, got %d", report.TotalExternalRefs)
	}
	if report.UniqueTargets != 3 {
		t.Errorf("expected 3 unique targets, got %d", report.UniqueTargets)
	}
	// First cluster should be the most frequent
	if len(report.Clusters) > 0 && report.Clusters[0].Count != 2 {
		t.Errorf("expected most frequent cluster to have 2 refs, got %d", report.Clusters[0].Count)
	}
}

func TestAnalyzeExternalRefs_Empty(t *testing.T) {
	analyzer := NewCrossRefAnalyzer()
	tripleStore := buildTestStore(2, nil, nil, nil, nil)
	analyzer.AddDocument("doc", "Document", tripleStore)

	report := analyzer.AnalyzeExternalRefs("doc")

	if report.TotalExternalRefs != 0 {
		t.Errorf("expected 0 external refs, got %d", report.TotalExternalRefs)
	}
	if report.UniqueTargets != 0 {
		t.Errorf("expected 0 unique targets, got %d", report.UniqueTargets)
	}
}

func TestAnalyzeExternalRefs_NonexistentDoc(t *testing.T) {
	analyzer := NewCrossRefAnalyzer()

	report := analyzer.AnalyzeExternalRefs("nonexistent")

	if report.TotalExternalRefs != 0 {
		t.Errorf("expected 0 external refs for nonexistent doc, got %d", report.TotalExternalRefs)
	}
}

func TestCompareDocuments_DefinitionOverlap(t *testing.T) {
	analyzer := NewCrossRefAnalyzer()
	storeA := buildTestStore(3, []string{"personal data", "consent", "controller"}, nil, nil, nil)
	storeB := buildTestStore(2, []string{"personal data", "consumer", "consent"}, nil, nil, nil)

	analyzer.AddDocument("gdpr", "GDPR", storeA)
	analyzer.AddDocument("ccpa", "CCPA", storeB)

	result := analyzer.CompareDocuments("gdpr", "ccpa")

	if result.Statistics.SharedDefinitionCount != 2 {
		t.Errorf("expected 2 shared definitions, got %d", result.Statistics.SharedDefinitionCount)
	}
	// Verify the shared definitions include expected terms
	sharedTerms := make(map[string]bool)
	for _, overlap := range result.SharedDefinitions {
		sharedTerms[overlap.Concept] = true
	}
	if !sharedTerms["personal data"] {
		t.Error("expected 'personal data' in shared definitions")
	}
	if !sharedTerms["consent"] {
		t.Error("expected 'consent' in shared definitions")
	}
}

func TestCompareDocuments_RightsOverlap(t *testing.T) {
	analyzer := NewCrossRefAnalyzer()
	storeA := buildTestStore(2, nil, []string{"reg:RightOfAccess", "reg:RightToErasure"}, nil, nil)
	storeB := buildTestStore(2, nil, []string{"reg:RightOfAccess", "reg:RightToObject"}, nil, nil)

	analyzer.AddDocument("gdpr", "GDPR", storeA)
	analyzer.AddDocument("ccpa", "CCPA", storeB)

	result := analyzer.CompareDocuments("gdpr", "ccpa")

	if result.Statistics.SharedRightCount != 1 {
		t.Errorf("expected 1 shared right, got %d", result.Statistics.SharedRightCount)
	}
}

func TestCompareDocuments_ObligationOverlap(t *testing.T) {
	analyzer := NewCrossRefAnalyzer()
	storeA := buildTestStore(2, nil, nil, []string{"reg:TransparencyObligation", "reg:NotificationObligation"}, nil)
	storeB := buildTestStore(2, nil, nil, []string{"reg:TransparencyObligation", "reg:SecurityObligation"}, nil)

	analyzer.AddDocument("gdpr", "GDPR", storeA)
	analyzer.AddDocument("ccpa", "CCPA", storeB)

	result := analyzer.CompareDocuments("gdpr", "ccpa")

	if result.Statistics.SharedObligationCount != 1 {
		t.Errorf("expected 1 shared obligation, got %d", result.Statistics.SharedObligationCount)
	}
}

func TestCompareDocuments_ExternalRefOverlap(t *testing.T) {
	analyzer := NewCrossRefAnalyzer()
	storeA := buildTestStore(2, nil, nil, nil, []string{"Directive 95/46/EC", "Regulation (EC) No 45/2001"})
	storeB := buildTestStore(2, nil, nil, nil, []string{"Directive 95/46/EC", "Directive 2002/58/EC"})

	analyzer.AddDocument("gdpr", "GDPR", storeA)
	analyzer.AddDocument("ccpa", "CCPA", storeB)

	result := analyzer.CompareDocuments("gdpr", "ccpa")

	if result.Statistics.SharedExternalRefCount != 1 {
		t.Errorf("expected 1 shared external ref, got %d", result.Statistics.SharedExternalRefCount)
	}
}

func TestAnalyze_MultiDocument(t *testing.T) {
	analyzer := NewCrossRefAnalyzer()

	storeA := buildTestStore(5, []string{"personal data", "consent", "controller"}, []string{"reg:RightOfAccess"}, []string{"reg:TransparencyObligation"}, []string{"Directive 95/46/EC"})
	storeB := buildTestStore(3, []string{"personal data", "consumer"}, []string{"reg:RightOfAccess"}, nil, []string{"Directive 95/46/EC"})
	storeC := buildTestStore(4, []string{"AI system", "personal data"}, nil, []string{"reg:TransparencyObligation"}, []string{"Regulation (EU) 2016/679"})

	analyzer.AddDocument("gdpr", "GDPR", storeA)
	analyzer.AddDocument("ccpa", "CCPA", storeB)
	analyzer.AddDocument("ai-act", "AI Act", storeC)

	result := analyzer.Analyze()

	if result.Statistics.TotalDocuments != 3 {
		t.Errorf("expected 3 documents, got %d", result.Statistics.TotalDocuments)
	}
	if len(result.Documents) != 3 {
		t.Errorf("expected 3 document summaries, got %d", len(result.Documents))
	}

	// "personal data" should be shared across all 3 documents
	foundPersonalData := false
	for _, overlap := range result.DefinitionOverlap {
		if overlap.Concept == "personal data" {
			foundPersonalData = true
			if len(overlap.Documents) != 3 {
				t.Errorf("expected 'personal data' in 3 docs, got %d", len(overlap.Documents))
			}
		}
	}
	if !foundPersonalData {
		t.Error("expected 'personal data' in definition overlaps")
	}

	// Rights overlap: RightOfAccess in gdpr and ccpa
	if result.Statistics.SharedRights < 1 {
		t.Errorf("expected at least 1 shared right, got %d", result.Statistics.SharedRights)
	}

	// Obligation overlap: TransparencyObligation in gdpr and ai-act
	if result.Statistics.SharedObligations < 1 {
		t.Errorf("expected at least 1 shared obligation, got %d", result.Statistics.SharedObligations)
	}
}

func TestCrossRefResult_String(t *testing.T) {
	analyzer := NewCrossRefAnalyzer()
	storeA := buildTestStore(3, []string{"personal data"}, nil, nil, []string{"Directive 95/46/EC"})
	storeB := buildTestStore(2, []string{"personal data"}, nil, nil, nil)

	analyzer.AddDocument("gdpr", "GDPR", storeA)
	analyzer.AddDocument("ccpa", "CCPA", storeB)

	result := analyzer.Analyze()
	output := result.String()

	if !strings.Contains(output, "Cross-Legislation Analysis") {
		t.Error("expected output to contain header")
	}
	if !strings.Contains(output, "GDPR") {
		t.Error("expected output to contain 'GDPR'")
	}
	if !strings.Contains(output, "CCPA") {
		t.Error("expected output to contain 'CCPA'")
	}
	if !strings.Contains(output, "Documents analyzed: 2") {
		t.Error("expected output to contain document count")
	}
}

func TestCrossRefResult_FormatTable(t *testing.T) {
	analyzer := NewCrossRefAnalyzer()
	storeA := buildTestStore(5, []string{"consent"}, nil, nil, nil)
	storeB := buildTestStore(3, nil, nil, nil, nil)

	analyzer.AddDocument("gdpr", "GDPR", storeA)
	analyzer.AddDocument("ccpa", "CCPA", storeB)

	result := analyzer.Analyze()
	table := result.FormatTable()

	if !strings.Contains(table, "Structural Metrics") {
		t.Error("expected table to contain 'Structural Metrics'")
	}
	if !strings.Contains(table, "Articles") {
		t.Error("expected table to contain 'Articles' metric")
	}
}

func TestCrossRefResult_ToJSON(t *testing.T) {
	analyzer := NewCrossRefAnalyzer()
	storeA := buildTestStore(2, []string{"consent"}, nil, nil, nil)
	analyzer.AddDocument("gdpr", "GDPR", storeA)

	result := analyzer.Analyze()
	jsonData, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var parsed CrossRefResult
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if parsed.Statistics.TotalDocuments != 1 {
		t.Errorf("expected 1 document in JSON, got %d", parsed.Statistics.TotalDocuments)
	}
}

func TestComparisonResult_String(t *testing.T) {
	analyzer := NewCrossRefAnalyzer()
	storeA := buildTestStore(3, []string{"personal data"}, nil, nil, nil)
	storeB := buildTestStore(2, []string{"personal data"}, nil, nil, nil)

	analyzer.AddDocument("gdpr", "GDPR", storeA)
	analyzer.AddDocument("ccpa", "CCPA", storeB)

	result := analyzer.CompareDocuments("gdpr", "ccpa")
	output := result.String()

	if !strings.Contains(output, "Comparison: GDPR vs CCPA") {
		t.Error("expected comparison header")
	}
	if !strings.Contains(output, "Shared definitions") {
		t.Error("expected shared definitions count")
	}
}

func TestExternalRefReport_String(t *testing.T) {
	analyzer := NewCrossRefAnalyzer()
	tripleStore := buildTestStore(2, nil, nil, nil, []string{"Directive 95/46/EC", "Regulation (EC) No 45/2001"})
	analyzer.AddDocument("gdpr", "GDPR", tripleStore)

	report := analyzer.AnalyzeExternalRefs("gdpr")
	output := report.String()

	if !strings.Contains(output, "External Reference Report: GDPR") {
		t.Error("expected report header")
	}
	if !strings.Contains(output, "Total external references:") {
		t.Error("expected total count")
	}
}

func TestCrossRefResult_ToDOT(t *testing.T) {
	analyzer := NewCrossRefAnalyzer()
	storeA := buildTestStore(3, []string{"personal data"}, nil, nil, []string{"Directive 95/46/EC"})
	storeB := buildTestStore(2, []string{"personal data"}, nil, nil, []string{"Directive 95/46/EC"})

	analyzer.AddDocument("gdpr", "GDPR", storeA)
	analyzer.AddDocument("ccpa", "CCPA", storeB)

	result := analyzer.Analyze()
	dotOutput := result.ToDOT()

	if !strings.Contains(dotOutput, "digraph CrossLegislationAnalysis") {
		t.Error("expected DOT graph header")
	}
	if !strings.Contains(dotOutput, "subgraph cluster_") {
		t.Error("expected cluster subgraphs")
	}
	if !strings.Contains(dotOutput, "GDPR") {
		t.Error("expected GDPR in DOT output")
	}
}

func TestComparisonResult_ToDOT(t *testing.T) {
	analyzer := NewCrossRefAnalyzer()
	storeA := buildTestStore(3, []string{"personal data"}, nil, nil, nil)
	storeB := buildTestStore(2, []string{"personal data"}, nil, nil, nil)

	analyzer.AddDocument("gdpr", "GDPR", storeA)
	analyzer.AddDocument("ccpa", "CCPA", storeB)

	result := analyzer.CompareDocuments("gdpr", "ccpa")
	dotOutput := result.ToDOT()

	if !strings.Contains(dotOutput, "digraph DocumentComparison") {
		t.Error("expected DOT comparison header")
	}
}

func TestNormalizeExternalRef(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Directive 95/46/EC", "directive 95/46/ec"},
		{"urn:external:Directive 95/46/EC", "directive 95/46/ec"},
		{"  Regulation (EU) 2016/679  ", "regulation (eu) 2016/679"},
		{"Directive:2002/58/EC", "2002/58/ec"},
	}

	for _, tc := range tests {
		result := normalizeExternalRef(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeExternalRef(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestNormalizeConceptName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"reg:RightOfAccess", "rightofaccess"},
		{"https://regula.dev/ontology#TransparencyObligation", "transparencyobligation"},
		{"SimpleValue", "simplevalue"},
	}

	for _, tc := range tests {
		result := normalizeConceptName(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeConceptName(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestSanitizeDOTID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"gdpr", "gdpr"},
		{"eu-ai-act", "eu_ai_act"},
		{"test/doc.txt", "test_doc_txt"},
	}

	for _, tc := range tests {
		result := sanitizeDOTID(tc.input)
		if result != tc.expected {
			t.Errorf("sanitizeDOTID(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestBuildDocumentSummary(t *testing.T) {
	analyzer := NewCrossRefAnalyzer()
	tripleStore := buildTestStore(5,
		[]string{"personal data", "consent"},
		[]string{"reg:RightOfAccess"},
		[]string{"reg:TransparencyObligation"},
		[]string{"Directive 95/46/EC"},
	)
	analyzer.AddDocument("gdpr", "GDPR", tripleStore)

	summary := analyzer.buildDocumentSummary("gdpr")

	if summary.Articles != 5 {
		t.Errorf("expected 5 articles, got %d", summary.Articles)
	}
	if summary.Definitions != 2 {
		t.Errorf("expected 2 definitions, got %d", summary.Definitions)
	}
	if summary.Rights != 1 {
		t.Errorf("expected 1 right, got %d", summary.Rights)
	}
	if summary.Obligations != 1 {
		t.Errorf("expected 1 obligation, got %d", summary.Obligations)
	}
	if summary.ExternalRefs != 1 {
		t.Errorf("expected 1 external ref, got %d", summary.ExternalRefs)
	}
}

func TestBuildDocumentSummary_Nonexistent(t *testing.T) {
	analyzer := NewCrossRefAnalyzer()
	summary := analyzer.buildDocumentSummary("nonexistent")

	if summary.ID != "nonexistent" {
		t.Errorf("expected ID 'nonexistent', got %s", summary.ID)
	}
	if summary.Triples != 0 {
		t.Errorf("expected 0 triples for nonexistent doc, got %d", summary.Triples)
	}
}
