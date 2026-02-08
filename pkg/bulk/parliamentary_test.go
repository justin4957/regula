package bulk

import (
	"testing"
)

func TestParliamentarySource_Name(t *testing.T) {
	source := NewParliamentarySource(DefaultDownloadConfig())
	if source.Name() != "parliamentary" {
		t.Errorf("Expected name 'parliamentary', got %q", source.Name())
	}
}

func TestParliamentarySource_Description(t *testing.T) {
	source := NewParliamentarySource(DefaultDownloadConfig())
	desc := source.Description()
	if desc == "" {
		t.Error("Expected non-empty description")
	}
}

func TestParliamentarySource_ListDatasets(t *testing.T) {
	source := NewParliamentarySource(DefaultDownloadConfig())
	datasets, err := source.ListDatasets()
	if err != nil {
		t.Fatalf("ListDatasets failed: %v", err)
	}

	if len(datasets) == 0 {
		t.Error("Expected at least one dataset")
	}

	// Check that we have the expected document types
	foundHouse := false
	foundSenate := false
	foundJoint := false

	for _, ds := range datasets {
		if ds.SourceName != "parliamentary" {
			t.Errorf("Expected source name 'parliamentary', got %q", ds.SourceName)
		}
		if ds.Jurisdiction != "US-Federal" {
			t.Errorf("Expected jurisdiction 'US-Federal', got %q", ds.Jurisdiction)
		}
		if ds.Identifier == "house-rules-119th" {
			foundHouse = true
		}
		if ds.Identifier == "senate-rules" {
			foundSenate = true
		}
		if ds.Identifier == "joint-rules" {
			foundJoint = true
		}
	}

	if !foundHouse {
		t.Error("Expected to find house-rules-119th dataset")
	}
	if !foundSenate {
		t.Error("Expected to find senate-rules dataset")
	}
	if !foundJoint {
		t.Error("Expected to find joint-rules dataset")
	}
}

func TestParliamentaryDocuments_HaveValidURLs(t *testing.T) {
	for _, doc := range parliamentaryDocuments {
		if doc.URL == "" {
			t.Errorf("Document %q has empty URL", doc.Identifier)
		}
		if doc.Identifier == "" {
			t.Error("Found document with empty identifier")
		}
		if doc.DisplayName == "" {
			t.Errorf("Document %q has empty display name", doc.Identifier)
		}
		if doc.Format == "" {
			t.Errorf("Document %q has empty format", doc.Identifier)
		}
		if doc.Chamber == "" {
			t.Errorf("Document %q has empty chamber", doc.Identifier)
		}
	}
}

func TestGetParliamentaryDocumentsByChamber(t *testing.T) {
	tests := []struct {
		chamber     string
		minExpected int
	}{
		{"house", 2},
		{"senate", 2},
		{"joint", 1},
	}

	for _, tc := range tests {
		t.Run(tc.chamber, func(t *testing.T) {
			docs := GetParliamentaryDocumentsByChamber(tc.chamber)
			if len(docs) < tc.minExpected {
				t.Errorf("Expected at least %d documents for chamber %q, got %d",
					tc.minExpected, tc.chamber, len(docs))
			}
			for _, doc := range docs {
				if doc.Chamber != tc.chamber {
					t.Errorf("Document %q has chamber %q, expected %q",
						doc.Identifier, doc.Chamber, tc.chamber)
				}
			}
		})
	}
}

func TestGetParliamentaryDocumentByID(t *testing.T) {
	tests := []struct {
		id       string
		expected bool
	}{
		{"house-rules-119th", true},
		{"senate-rules", true},
		{"joint-rules", true},
		{"nonexistent", false},
	}

	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			doc := GetParliamentaryDocumentByID(tc.id)
			if tc.expected && doc == nil {
				t.Errorf("Expected to find document %q", tc.id)
			}
			if !tc.expected && doc != nil {
				t.Errorf("Expected not to find document %q", tc.id)
			}
		})
	}
}

func TestFormatParliamentarySummary(t *testing.T) {
	summary := &ParliamentarySummary{
		Documents:         3,
		TotalRules:        76,
		TotalClauses:      415,
		TotalCrossRefs:    250,
		TotalExternalRefs: 50,
		TotalTriples:      5847,
		CrossDocumentRefs: 15,
		DocumentStats: []ParliamentaryStats{
			{
				Identifier:      "house-rules-119th",
				DisplayName:     "House Rules (119th Congress)",
				Chamber:         "house",
				Rules:           29,
				Clauses:         215,
				CrossReferences: 161,
			},
			{
				Identifier:      "senate-rules",
				DisplayName:     "Senate Standing Rules",
				Chamber:         "senate",
				Rules:           43,
				Clauses:         188,
				CrossReferences: 79,
			},
		},
	}

	output := FormatParliamentarySummary(summary)

	// Check that output contains expected elements
	if output == "" {
		t.Error("Expected non-empty output")
	}

	expectedStrings := []string{
		"Parliamentary Rules Summary",
		"Documents ingested: 3",
		"Total rules: 76",
		"Total clauses: 415",
		"Cross-document references: 15",
		"Combined graph: 5847 triples",
		"House Rules",
		"Senate Standing Rules",
	}

	for _, expected := range expectedStrings {
		if !containsString(output, expected) {
			t.Errorf("Expected output to contain %q", expected)
		}
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a very long string", 10, "this is..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "..."},
	}

	for _, tc := range tests {
		result := truncateString(tc.input, tc.maxLen)
		if result != tc.expected {
			t.Errorf("truncateString(%q, %d): expected %q, got %q",
				tc.input, tc.maxLen, tc.expected, result)
		}
	}
}

func TestResolveSource_Parliamentary(t *testing.T) {
	config := DefaultDownloadConfig()
	source, err := ResolveSource("parliamentary", config)
	if err != nil {
		t.Fatalf("ResolveSource failed: %v", err)
	}
	if source.Name() != "parliamentary" {
		t.Errorf("Expected source name 'parliamentary', got %q", source.Name())
	}
}

func TestAllSourceNames_IncludesParliamentary(t *testing.T) {
	names := AllSourceNames()
	found := false
	for _, name := range names {
		if name == "parliamentary" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected AllSourceNames to include 'parliamentary'")
	}
}

// containsString checks if s contains substr
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
