package eurlex

import (
	"testing"

	"github.com/coolbeans/regula/pkg/citation"
)

func TestGenerateCELEX(t *testing.T) {
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
			expected: "32016R0679",
		},
		{
			name: "directive_95_46",
			citation: &citation.Citation{
				Type:       citation.CitationTypeDirective,
				Components: citation.CitationComponents{DocYear: "95", DocNumber: "46"},
			},
			expected: "31995L0046",
		},
		{
			name: "decision_2010_87",
			citation: &citation.Citation{
				Type:       citation.CitationTypeDecision,
				Components: citation.CitationComponents{DocYear: "2010", DocNumber: "87"},
			},
			expected: "32010D0087",
		},
		{
			name: "regulation_ec_no_45_2001",
			citation: &citation.Citation{
				Type:       citation.CitationTypeRegulation,
				Components: citation.CitationComponents{DocYear: "2001", DocNumber: "45"},
			},
			expected: "32001R0045",
		},
		{
			name: "directive_eu_2016_680",
			citation: &citation.Citation{
				Type:       citation.CitationTypeDirective,
				Components: citation.CitationComponents{DocYear: "2016", DocNumber: "680"},
			},
			expected: "32016L0680",
		},
		{
			name: "large_number_no_padding",
			citation: &citation.Citation{
				Type:       citation.CitationTypeRegulation,
				Components: citation.CitationComponents{DocYear: "2022", DocNumber: "1234"},
			},
			expected: "32022R1234",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			celexNumber, err := GenerateCELEX(tc.citation)
			if err != nil {
				t.Fatalf("GenerateCELEX failed: %v", err)
			}
			result := celexNumber.String()
			if result != tc.expected {
				t.Errorf("CELEX: got %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestGenerateCELEX_Errors(t *testing.T) {
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
			name: "unsupported_type_case",
			citation: &citation.Citation{
				Type:       citation.CitationTypeCase,
				Components: citation.CitationComponents{DocYear: "2016", DocNumber: "679"},
			},
		},
		{
			name: "unsupported_type_statute",
			citation: &citation.Citation{
				Type:       citation.CitationTypeStatute,
				Components: citation.CitationComponents{DocYear: "2016", DocNumber: "679"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := GenerateCELEX(tc.citation)
			if err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func TestGenerateCELEX_StructFields(t *testing.T) {
	citationRef := &citation.Citation{
		Type:       citation.CitationTypeRegulation,
		Components: citation.CitationComponents{DocYear: "2016", DocNumber: "679"},
	}

	celexNumber, err := GenerateCELEX(citationRef)
	if err != nil {
		t.Fatalf("GenerateCELEX failed: %v", err)
	}

	if celexNumber.Sector != SectorLegislation {
		t.Errorf("Sector: got %q, want %q", celexNumber.Sector, SectorLegislation)
	}
	if celexNumber.Year != "2016" {
		t.Errorf("Year: got %q, want '2016'", celexNumber.Year)
	}
	if celexNumber.TypeCode != TypeRegulation {
		t.Errorf("TypeCode: got %q, want %q", celexNumber.TypeCode, TypeRegulation)
	}
	if celexNumber.Number != "0679" {
		t.Errorf("Number: got %q, want '0679'", celexNumber.Number)
	}
}

func TestNormalizeYear(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"95", "1995"},
		{"58", "1958"},
		{"99", "1999"},
		{"57", "2057"},
		{"16", "2016"},
		{"02", "2002"},
		{"00", "2000"},
		{"2016", "2016"},
		{"1995", "1995"},
		{"2022", "2022"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			result := normalizeYear(tc.input)
			if result != tc.expected {
				t.Errorf("normalizeYear(%q): got %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestPadCELEXNumber(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"1", "0001"},
		{"46", "0046"},
		{"679", "0679"},
		{"1234", "1234"},
		{"12345", "12345"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			result := padCELEXNumber(tc.input)
			if result != tc.expected {
				t.Errorf("padCELEXNumber(%q): got %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestCELEXNumberString(t *testing.T) {
	celexNumber := CELEXNumber{
		Sector:   SectorLegislation,
		Year:     "2016",
		TypeCode: TypeRegulation,
		Number:   "0679",
	}

	result := celexNumber.String()
	if result != "32016R0679" {
		t.Errorf("String(): got %q, want '32016R0679'", result)
	}
}
