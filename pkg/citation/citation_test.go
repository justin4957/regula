package citation

import (
	"encoding/json"
	"testing"

	"github.com/coolbeans/regula/pkg/types"
)

func TestCitationTypeStringValues(t *testing.T) {
	cases := []struct {
		citationType CitationType
		expected     string
	}{
		{CitationTypeStatute, "statute"},
		{CitationTypeRegulation, "regulation"},
		{CitationTypeDirective, "directive"},
		{CitationTypeDecision, "decision"},
		{CitationTypeTreaty, "treaty"},
		{CitationTypeCase, "case"},
		{CitationTypeCode, "code"},
		{CitationTypeUnknown, "unknown"},
	}

	for _, tc := range cases {
		t.Run(tc.expected, func(t *testing.T) {
			if string(tc.citationType) != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, string(tc.citationType))
			}
		})
	}
}

func TestTemporalRefKindStringValues(t *testing.T) {
	cases := []struct {
		kind     TemporalRefKind
		expected string
	}{
		{TemporalAsAmended, "as_amended"},
		{TemporalInForceOn, "in_force_on"},
		{TemporalRepealed, "repealed"},
		{TemporalOriginal, "original"},
		{TemporalConsolidated, "consolidated"},
	}

	for _, tc := range cases {
		t.Run(tc.expected, func(t *testing.T) {
			if string(tc.kind) != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, string(tc.kind))
			}
		})
	}
}

func TestCitationJSONRoundtrip(t *testing.T) {
	date := types.Date{Year: 2025, Month: 5, Day: 25}
	original := &Citation{
		RawText:      "Regulation (EU) 2016/679",
		Type:         CitationTypeRegulation,
		Jurisdiction: "EU",
		Document:     "Regulation (EU) 2016/679",
		Subdivision:  "Article 6(1)(a)",
		Temporal: &TemporalRef{
			Kind:        TemporalAsAmended,
			Description: "as amended by Regulation (EU) 2018/1725",
			Date:        &date,
		},
		Confidence: 0.95,
		Parser:     "EU Citation Parser",
		TextOffset: 42,
		TextLength: 24,
		Components: CitationComponents{
			DocYear:         "2016",
			DocNumber:       "679",
			ArticleNumber:   6,
			ParagraphNumber: 1,
			PointLetter:     "a",
		},
	}

	// Marshal to JSON.
	jsonBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal Citation to JSON: %v", err)
	}

	// Unmarshal back.
	var restored Citation
	if err := json.Unmarshal(jsonBytes, &restored); err != nil {
		t.Fatalf("Failed to unmarshal Citation from JSON: %v", err)
	}

	// Verify key fields.
	if restored.RawText != original.RawText {
		t.Errorf("RawText mismatch: got %q, want %q", restored.RawText, original.RawText)
	}
	if restored.Type != original.Type {
		t.Errorf("Type mismatch: got %q, want %q", restored.Type, original.Type)
	}
	if restored.Jurisdiction != original.Jurisdiction {
		t.Errorf("Jurisdiction mismatch: got %q, want %q", restored.Jurisdiction, original.Jurisdiction)
	}
	if restored.Document != original.Document {
		t.Errorf("Document mismatch: got %q, want %q", restored.Document, original.Document)
	}
	if restored.Confidence != original.Confidence {
		t.Errorf("Confidence mismatch: got %f, want %f", restored.Confidence, original.Confidence)
	}
	if restored.TextOffset != original.TextOffset {
		t.Errorf("TextOffset mismatch: got %d, want %d", restored.TextOffset, original.TextOffset)
	}
	if restored.Components.DocYear != original.Components.DocYear {
		t.Errorf("DocYear mismatch: got %q, want %q", restored.Components.DocYear, original.Components.DocYear)
	}
	if restored.Components.ArticleNumber != original.Components.ArticleNumber {
		t.Errorf("ArticleNumber mismatch: got %d, want %d", restored.Components.ArticleNumber, original.Components.ArticleNumber)
	}
	if restored.Temporal == nil {
		t.Fatal("Temporal should not be nil")
	}
	if restored.Temporal.Kind != original.Temporal.Kind {
		t.Errorf("Temporal.Kind mismatch: got %q, want %q", restored.Temporal.Kind, original.Temporal.Kind)
	}
}

func TestCitationJSONWithoutOptionalFields(t *testing.T) {
	citation := &Citation{
		RawText:      "Article 5",
		Type:         CitationTypeStatute,
		Jurisdiction: "EU",
		Confidence:   0.9,
		Parser:       "EU Citation Parser",
	}

	jsonBytes, err := json.Marshal(citation)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Ensure omitempty fields are absent.
	jsonStr := string(jsonBytes)
	if containsField(jsonStr, "subdivision") {
		t.Error("Empty Subdivision should be omitted from JSON")
	}
	if containsField(jsonStr, "temporal") {
		t.Error("Nil Temporal should be omitted from JSON")
	}
}

// containsField checks if a JSON string contains a given field key.
func containsField(jsonStr, fieldName string) bool {
	return len(jsonStr) > 0 && json.Valid([]byte(jsonStr)) &&
		contains(jsonStr, `"`+fieldName+`"`)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
