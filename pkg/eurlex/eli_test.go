package eurlex

import (
	"testing"

	"github.com/coolbeans/regula/pkg/citation"
)

func TestGenerateELI(t *testing.T) {
	cases := []struct {
		name     string
		citation *citation.Citation
		expected string
	}{
		{
			name: "gdpr_regulation",
			citation: &citation.Citation{
				Type:       citation.CitationTypeRegulation,
				Components: citation.CitationComponents{DocYear: "2016", DocNumber: "679"},
			},
			expected: "http://data.europa.eu/eli/reg/2016/679/oj",
		},
		{
			name: "directive_95_46",
			citation: &citation.Citation{
				Type:       citation.CitationTypeDirective,
				Components: citation.CitationComponents{DocYear: "95", DocNumber: "46"},
			},
			expected: "http://data.europa.eu/eli/dir/1995/46/oj",
		},
		{
			name: "decision_2010_87",
			citation: &citation.Citation{
				Type:       citation.CitationTypeDecision,
				Components: citation.CitationComponents{DocYear: "2010", DocNumber: "87"},
			},
			expected: "http://data.europa.eu/eli/dec/2010/87/oj",
		},
		{
			name: "directive_eu_2016_680",
			citation: &citation.Citation{
				Type:       citation.CitationTypeDirective,
				Components: citation.CitationComponents{DocYear: "2016", DocNumber: "680"},
			},
			expected: "http://data.europa.eu/eli/dir/2016/680/oj",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eliURI, err := GenerateELI(tc.citation)
			if err != nil {
				t.Fatalf("GenerateELI failed: %v", err)
			}
			result := eliURI.String()
			if result != tc.expected {
				t.Errorf("ELI URI: got %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestGenerateELI_Errors(t *testing.T) {
	cases := []struct {
		name     string
		citation *citation.Citation
	}{
		{
			name:     "nil_citation",
			citation: nil,
		},
		{
			name: "missing_year",
			citation: &citation.Citation{
				Type:       citation.CitationTypeRegulation,
				Components: citation.CitationComponents{DocNumber: "679"},
			},
		},
		{
			name: "missing_number",
			citation: &citation.Citation{
				Type:       citation.CitationTypeRegulation,
				Components: citation.CitationComponents{DocYear: "2016"},
			},
		},
		{
			name: "unsupported_type_treaty",
			citation: &citation.Citation{
				Type:       citation.CitationTypeTreaty,
				Components: citation.CitationComponents{DocYear: "2016", DocNumber: "679"},
			},
		},
		{
			name: "unsupported_type_unknown",
			citation: &citation.Citation{
				Type:       citation.CitationTypeUnknown,
				Components: citation.CitationComponents{DocYear: "2016", DocNumber: "679"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := GenerateELI(tc.citation)
			if err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func TestELIDoesNotPadNumbers(t *testing.T) {
	// ELI uses unpadded numbers (contrast with CELEX which pads to 4 digits).
	citationRef := &citation.Citation{
		Type:       citation.CitationTypeRegulation,
		Components: citation.CitationComponents{DocYear: "2016", DocNumber: "679"},
	}

	eliURI, err := GenerateELI(citationRef)
	if err != nil {
		t.Fatalf("GenerateELI failed: %v", err)
	}

	// ELI should use "679", not "0679".
	if eliURI.Number != "679" {
		t.Errorf("ELI number should not be padded: got %q, want '679'", eliURI.Number)
	}

	// Contrast with CELEX.
	celexNumber, err := GenerateCELEX(citationRef)
	if err != nil {
		t.Fatalf("GenerateCELEX failed: %v", err)
	}
	if celexNumber.Number != "0679" {
		t.Errorf("CELEX number should be padded: got %q, want '0679'", celexNumber.Number)
	}
}

func TestELIURIString(t *testing.T) {
	eliURI := ELIURI{
		TypeSlug: "reg",
		Year:     "2016",
		Number:   "679",
	}

	result := eliURI.String()
	expected := "http://data.europa.eu/eli/reg/2016/679/oj"
	if result != expected {
		t.Errorf("String(): got %q, want %q", result, expected)
	}
}

func TestGenerateELI_StructFields(t *testing.T) {
	citationRef := &citation.Citation{
		Type:       citation.CitationTypeDirective,
		Components: citation.CitationComponents{DocYear: "95", DocNumber: "46"},
	}

	eliURI, err := GenerateELI(citationRef)
	if err != nil {
		t.Fatalf("GenerateELI failed: %v", err)
	}

	if eliURI.TypeSlug != "dir" {
		t.Errorf("TypeSlug: got %q, want 'dir'", eliURI.TypeSlug)
	}
	if eliURI.Year != "1995" {
		t.Errorf("Year: got %q, want '1995' (normalized from '95')", eliURI.Year)
	}
	if eliURI.Number != "46" {
		t.Errorf("Number: got %q, want '46' (not padded)", eliURI.Number)
	}
}
