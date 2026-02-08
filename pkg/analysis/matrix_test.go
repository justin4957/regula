package analysis

import (
	"strings"
	"testing"

	"github.com/coolbeans/regula/pkg/store"
)

func TestBuildRuleMatrix_Empty(t *testing.T) {
	tripleStore := store.NewTripleStore()
	matrix := BuildRuleMatrix(tripleStore)

	if len(matrix.Rules) != 0 {
		t.Errorf("Expected 0 rules, got %d", len(matrix.Rules))
	}
	if matrix.TotalRefs != 0 {
		t.Errorf("Expected 0 total refs, got %d", matrix.TotalRefs)
	}
}

func TestBuildRuleMatrix_SimpleRefs(t *testing.T) {
	tripleStore := store.NewTripleStore()

	// Add some rule references
	tripleStore.Add(
		"https://example.com/Rule_I_clause_1",
		store.PropReferences,
		"https://example.com/Rule_X_clause_5",
	)
	tripleStore.Add(
		"https://example.com/Rule_I_clause_2",
		store.PropReferences,
		"https://example.com/Rule_X_clause_3",
	)
	tripleStore.Add(
		"https://example.com/Rule_X_clause_5",
		store.PropReferences,
		"https://example.com/Rule_XX_clause_1",
	)

	matrix := BuildRuleMatrix(tripleStore)

	if len(matrix.Rules) < 2 {
		t.Fatalf("Expected at least 2 rules, got %d", len(matrix.Rules))
	}

	if matrix.TotalRefs != 3 {
		t.Errorf("Expected 3 total refs, got %d", matrix.TotalRefs)
	}
}

func TestRomanToInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"I", 1},
		{"II", 2},
		{"III", 3},
		{"IV", 4},
		{"V", 5},
		{"IX", 9},
		{"X", 10},
		{"XI", 11},
		{"XVIII", 18},
		{"XIX", 19},
		{"XX", 20},
		{"XXI", 21},
		{"XXIX", 29},
	}

	for _, tc := range tests {
		result := romanToInt(tc.input)
		if result != tc.expected {
			t.Errorf("romanToInt(%q): expected %d, got %d", tc.input, tc.expected, result)
		}
	}
}

func TestSortRomanNumerals(t *testing.T) {
	numerals := []string{"X", "I", "XX", "V", "III", "IX", "XVIII"}
	sortRomanNumerals(numerals)

	expected := []string{"I", "III", "V", "IX", "X", "XVIII", "XX"}
	for i, exp := range expected {
		if numerals[i] != exp {
			t.Errorf("Position %d: expected %q, got %q", i, exp, numerals[i])
		}
	}
}

func TestExtractRuleFromURI(t *testing.T) {
	tests := []struct {
		uri      string
		expected string
	}{
		{"https://example.com/Rule_XX_clause_5", "XX"},
		{"https://example.com/rule_I_clause_1", "I"},
		{"https://example.com/Rule_XVIII_clause_6", "XVIII"},
		{"https://example.com/other/path", ""},
		{"https://example.com/Rule_", ""},
	}

	for _, tc := range tests {
		result := extractRuleFromURI(tc.uri)
		if result != tc.expected {
			t.Errorf("extractRuleFromURI(%q): expected %q, got %q", tc.uri, tc.expected, result)
		}
	}
}

func TestExtractRuleFromIdentifier(t *testing.T) {
	tests := []struct {
		id       string
		expected string
	}{
		{"Rule XX clause 5", "XX"},
		{"Rule I clause 1", "I"},
		{"Rule XVIII clause 6", "XVIII"},
		{"Article 17", ""},
		{"", ""},
	}

	for _, tc := range tests {
		result := extractRuleFromIdentifier(tc.id)
		if result != tc.expected {
			t.Errorf("extractRuleFromIdentifier(%q): expected %q, got %q", tc.id, tc.expected, result)
		}
	}
}

func TestMostConnected(t *testing.T) {
	matrix := &RuleMatrix{
		Rules:    []string{"I", "X", "XX"},
		Matrix:   [][]int{{0, 2, 1}, {1, 0, 3}, {0, 1, 0}},
		Incoming: []int{1, 3, 4},
		Outgoing: []int{3, 4, 1},
	}

	connections := matrix.MostConnected(2)

	if len(connections) < 2 {
		t.Fatalf("Expected at least 2 connections, got %d", len(connections))
	}

	// X should be most connected (4 outgoing + 3 incoming = 7)
	if connections[0].Rule != "X" {
		t.Errorf("Expected X to be most connected, got %s", connections[0].Rule)
	}
	if connections[0].Total != 7 {
		t.Errorf("Expected total of 7 for X, got %d", connections[0].Total)
	}
}

func TestFindClusters(t *testing.T) {
	// Create a matrix where I-X-XX form a connected cluster, but V is isolated
	matrix := &RuleMatrix{
		Rules: []string{"I", "V", "X", "XX"},
		Matrix: [][]int{
			{0, 0, 1, 0}, // I -> X
			{0, 0, 0, 0}, // V (isolated)
			{1, 0, 0, 1}, // X -> I, XX
			{0, 0, 1, 0}, // XX -> X
		},
		Incoming: []int{1, 0, 2, 1},
		Outgoing: []int{1, 0, 2, 1},
	}

	clusters := matrix.FindClusters()

	if len(clusters) != 1 {
		t.Fatalf("Expected 1 cluster, got %d", len(clusters))
	}

	cluster := clusters[0]
	if cluster.Size != 3 {
		t.Errorf("Expected cluster size 3, got %d", cluster.Size)
	}

	// V should not be in the cluster
	for _, rule := range cluster.Rules {
		if rule == "V" {
			t.Error("V should not be in the cluster (isolated)")
		}
	}
}

func TestToASCII(t *testing.T) {
	matrix := &RuleMatrix{
		Rules:     []string{"I", "X"},
		Matrix:    [][]int{{0, 2}, {1, 0}},
		Incoming:  []int{1, 2},
		Outgoing:  []int{2, 1},
		TotalRefs: 3,
	}

	ascii := matrix.ToASCII()

	if !strings.Contains(ascii, "I") {
		t.Error("ASCII output should contain 'I'")
	}
	if !strings.Contains(ascii, "X") {
		t.Error("ASCII output should contain 'X'")
	}
	if !strings.Contains(ascii, "2") {
		t.Error("ASCII output should contain the count '2'")
	}
}

func TestToCSV(t *testing.T) {
	matrix := &RuleMatrix{
		Rules:     []string{"I", "X"},
		Matrix:    [][]int{{0, 2}, {1, 0}},
		Incoming:  []int{1, 2},
		Outgoing:  []int{2, 1},
		TotalRefs: 3,
	}

	csvOutput := matrix.ToCSV()

	// Check header
	if !strings.Contains(csvOutput, "Source/Target") {
		t.Error("CSV should contain header 'Source/Target'")
	}

	// Check data
	lines := strings.Split(csvOutput, "\n")
	if len(lines) < 3 {
		t.Error("CSV should have at least 3 lines (header + 2 rules + incoming)")
	}
}

func TestToSVGHeatmap(t *testing.T) {
	matrix := &RuleMatrix{
		Rules:     []string{"I", "X", "XX"},
		Matrix:    [][]int{{0, 2, 1}, {1, 0, 3}, {0, 1, 0}},
		Incoming:  []int{1, 3, 4},
		Outgoing:  []int{3, 4, 1},
		TotalRefs: 8,
	}

	svg := matrix.ToSVGHeatmap()

	if !strings.Contains(svg, "<svg") {
		t.Error("SVG output should contain <svg tag")
	}
	if !strings.Contains(svg, "</svg>") {
		t.Error("SVG output should contain </svg> tag")
	}
	if !strings.Contains(svg, "rect") {
		t.Error("SVG output should contain rect elements")
	}
}

func TestGenerateMatrixReport(t *testing.T) {
	tripleStore := store.NewTripleStore()

	// Add some rule references
	tripleStore.Add(
		"https://example.com/Rule_I_clause_1",
		store.PropReferences,
		"https://example.com/Rule_X_clause_5",
	)

	report := GenerateMatrixReport(tripleStore)

	if report.Matrix == nil {
		t.Fatal("Report matrix should not be nil")
	}
}

func TestMatrixReport_String(t *testing.T) {
	matrix := &RuleMatrix{
		Rules:     []string{"I", "X"},
		Matrix:    [][]int{{0, 2}, {1, 0}},
		Incoming:  []int{1, 2},
		Outgoing:  []int{2, 1},
		TotalRefs: 3,
	}

	report := &MatrixReport{
		Matrix:        matrix,
		MostConnected: matrix.MostConnected(5),
		Clusters:      matrix.FindClusters(),
	}

	output := report.String()

	if !strings.Contains(output, "Cross-Reference Matrix") {
		t.Error("Report should contain title")
	}
	if !strings.Contains(output, "Most Connected Rules") {
		t.Error("Report should contain 'Most Connected Rules'")
	}
}

func TestMatrixReport_ToJSON(t *testing.T) {
	matrix := &RuleMatrix{
		Rules:     []string{"I", "X"},
		Matrix:    [][]int{{0, 2}, {1, 0}},
		Incoming:  []int{1, 2},
		Outgoing:  []int{2, 1},
		TotalRefs: 3,
	}

	report := &MatrixReport{
		Matrix:        matrix,
		MostConnected: matrix.MostConnected(5),
		Clusters:      matrix.FindClusters(),
	}

	jsonData, err := report.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if !strings.Contains(string(jsonData), `"rules"`) {
		t.Error("JSON should contain 'rules' field")
	}
}
