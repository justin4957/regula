package ukleg

import (
	"testing"

	"github.com/coolbeans/regula/pkg/citation"
)

func TestGenerateLegislationURI(t *testing.T) {
	cases := []struct {
		name        string
		citation    *citation.Citation
		expectedURI string
	}{
		{
			name: "Data Protection Act 2018 chapter reference",
			citation: &citation.Citation{
				Type:         citation.CitationTypeStatute,
				Jurisdiction: "UK",
				Components: citation.CitationComponents{
					DocYear:   "2018",
					DocNumber: "12",
					CodeName:  "ukact",
				},
			},
			expectedURI: "https://www.legislation.gov.uk/ukpga/2018/12",
		},
		{
			name: "Human Rights Act 1998 chapter reference",
			citation: &citation.Citation{
				Type:         citation.CitationTypeStatute,
				Jurisdiction: "UK",
				Components: citation.CitationComponents{
					DocYear:   "1998",
					DocNumber: "42",
					CodeName:  "ukact",
				},
			},
			expectedURI: "https://www.legislation.gov.uk/ukpga/1998/42",
		},
		{
			name: "Statutory Instrument 2019/419",
			citation: &citation.Citation{
				Type:         citation.CitationTypeRegulation,
				Jurisdiction: "UK",
				Components: citation.CitationComponents{
					DocYear:   "2019",
					DocNumber: "419",
				},
			},
			expectedURI: "https://www.legislation.gov.uk/uksi/2019/419",
		},
		{
			name: "Statutory Instrument 2018/1400",
			citation: &citation.Citation{
				Type:         citation.CitationTypeRegulation,
				Jurisdiction: "UK",
				Components: citation.CitationComponents{
					DocYear:   "2018",
					DocNumber: "1400",
				},
			},
			expectedURI: "https://www.legislation.gov.uk/uksi/2018/1400",
		},
		{
			name: "Act with section reference",
			citation: &citation.Citation{
				Type:         citation.CitationTypeStatute,
				Jurisdiction: "UK",
				Components: citation.CitationComponents{
					DocYear:   "2018",
					DocNumber: "12",
					CodeName:  "ukact",
					Section:   "6",
				},
			},
			expectedURI: "https://www.legislation.gov.uk/id/ukpga/2018/12/section/6",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			legislationURI, err := GenerateLegislationURI(tc.citation)
			if err != nil {
				t.Fatalf("GenerateLegislationURI failed: %v", err)
			}

			uriStr := legislationURI.String()
			if uriStr != tc.expectedURI {
				t.Errorf("URI: got %q, want %q", uriStr, tc.expectedURI)
			}
		})
	}
}

func TestGenerateLegislationURI_Errors(t *testing.T) {
	cases := []struct {
		name     string
		citation *citation.Citation
	}{
		{
			name:     "nil citation",
			citation: nil,
		},
		{
			name: "missing year",
			citation: &citation.Citation{
				Type:       citation.CitationTypeStatute,
				Components: citation.CitationComponents{DocNumber: "12", CodeName: "ukact"},
			},
		},
		{
			name: "missing number",
			citation: &citation.Citation{
				Type:       citation.CitationTypeStatute,
				Components: citation.CitationComponents{DocYear: "2018", CodeName: "ukact"},
			},
		},
		{
			name: "unsupported type - case",
			citation: &citation.Citation{
				Type:       citation.CitationTypeCase,
				Components: citation.CitationComponents{DocYear: "2019", DocNumber: "5"},
			},
		},
		{
			name: "unsupported type - treaty",
			citation: &citation.Citation{
				Type:       citation.CitationTypeTreaty,
				Components: citation.CitationComponents{DocYear: "1950", DocNumber: "1"},
			},
		},
		{
			name: "statute without ukact code name",
			citation: &citation.Citation{
				Type:       citation.CitationTypeStatute,
				Components: citation.CitationComponents{DocYear: "2018", DocNumber: "12"},
			},
		},
		{
			name: "SI missing year",
			citation: &citation.Citation{
				Type:       citation.CitationTypeRegulation,
				Components: citation.CitationComponents{DocNumber: "419"},
			},
		},
		{
			name: "SI missing number",
			citation: &citation.Citation{
				Type:       citation.CitationTypeRegulation,
				Components: citation.CitationComponents{DocYear: "2019"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := GenerateLegislationURI(tc.citation)
			if err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func TestGenerateSectionURI(t *testing.T) {
	legislationURI := GenerateSectionURI("2018", "12", "6")

	expectedStr := "https://www.legislation.gov.uk/id/ukpga/2018/12/section/6"
	if legislationURI.String() != expectedStr {
		t.Errorf("String(): got %q, want %q", legislationURI.String(), expectedStr)
	}

	expectedID := "https://www.legislation.gov.uk/id/ukpga/2018/12/section/6"
	if legislationURI.IDString() != expectedID {
		t.Errorf("IDString(): got %q, want %q", legislationURI.IDString(), expectedID)
	}

	if legislationURI.LegislationType != LegislationTypeUKPGA {
		t.Errorf("LegislationType: got %q, want %q", legislationURI.LegislationType, LegislationTypeUKPGA)
	}
	if legislationURI.Year != "2018" {
		t.Errorf("Year: got %q, want %q", legislationURI.Year, "2018")
	}
	if legislationURI.Number != "12" {
		t.Errorf("Number: got %q, want %q", legislationURI.Number, "12")
	}
	if legislationURI.Section != "6" {
		t.Errorf("Section: got %q, want %q", legislationURI.Section, "6")
	}
}

func TestLegislationURI_String(t *testing.T) {
	cases := []struct {
		name        string
		uri         LegislationURI
		expectedStr string
	}{
		{
			name: "UKPGA without section",
			uri: LegislationURI{
				LegislationType: LegislationTypeUKPGA,
				Year:            "2018",
				Number:          "12",
			},
			expectedStr: "https://www.legislation.gov.uk/ukpga/2018/12",
		},
		{
			name: "UKSI without section",
			uri: LegislationURI{
				LegislationType: LegislationTypeUKSI,
				Year:            "2019",
				Number:          "419",
			},
			expectedStr: "https://www.legislation.gov.uk/uksi/2019/419",
		},
		{
			name: "UKPGA with section",
			uri: LegislationURI{
				LegislationType: LegislationTypeUKPGA,
				Year:            "2018",
				Number:          "12",
				Section:         "6",
			},
			expectedStr: "https://www.legislation.gov.uk/id/ukpga/2018/12/section/6",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.uri.String()
			if result != tc.expectedStr {
				t.Errorf("String(): got %q, want %q", result, tc.expectedStr)
			}
		})
	}
}

func TestLegislationURI_IDString(t *testing.T) {
	cases := []struct {
		name        string
		uri         LegislationURI
		expectedStr string
	}{
		{
			name: "UKPGA ID without section",
			uri: LegislationURI{
				LegislationType: LegislationTypeUKPGA,
				Year:            "2018",
				Number:          "12",
			},
			expectedStr: "https://www.legislation.gov.uk/id/ukpga/2018/12",
		},
		{
			name: "UKSI ID without section",
			uri: LegislationURI{
				LegislationType: LegislationTypeUKSI,
				Year:            "2019",
				Number:          "419",
			},
			expectedStr: "https://www.legislation.gov.uk/id/uksi/2019/419",
		},
		{
			name: "UKPGA ID with section",
			uri: LegislationURI{
				LegislationType: LegislationTypeUKPGA,
				Year:            "2018",
				Number:          "12",
				Section:         "114",
			},
			expectedStr: "https://www.legislation.gov.uk/id/ukpga/2018/12/section/114",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.uri.IDString()
			if result != tc.expectedStr {
				t.Errorf("IDString(): got %q, want %q", result, tc.expectedStr)
			}
		})
	}
}

func TestCitationTypeToLegislationSlug(t *testing.T) {
	cases := []struct {
		name         string
		citation     *citation.Citation
		expectedSlug LegislationType
		expectError  bool
	}{
		{
			name: "statute with ukact",
			citation: &citation.Citation{
				Type:       citation.CitationTypeStatute,
				Components: citation.CitationComponents{CodeName: "ukact"},
			},
			expectedSlug: LegislationTypeUKPGA,
		},
		{
			name: "regulation",
			citation: &citation.Citation{
				Type: citation.CitationTypeRegulation,
			},
			expectedSlug: LegislationTypeUKSI,
		},
		{
			name: "case law",
			citation: &citation.Citation{
				Type: citation.CitationTypeCase,
			},
			expectError: true,
		},
		{
			name: "treaty",
			citation: &citation.Citation{
				Type: citation.CitationTypeTreaty,
			},
			expectError: true,
		},
		{
			name: "statute without ukact code",
			citation: &citation.Citation{
				Type:       citation.CitationTypeStatute,
				Components: citation.CitationComponents{CodeName: "neutral"},
			},
			expectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			slug, err := citationTypeToLegislationSlug(tc.citation)
			if tc.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if slug != tc.expectedSlug {
				t.Errorf("Slug: got %q, want %q", slug, tc.expectedSlug)
			}
		})
	}
}
