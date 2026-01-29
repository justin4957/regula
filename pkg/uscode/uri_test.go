package uscode

import (
	"testing"

	"github.com/coolbeans/regula/pkg/citation"
)

func TestGenerateUSCURI(t *testing.T) {
	testCases := []struct {
		name        string
		citation    *citation.Citation
		wantTitle   string
		wantSection string
		wantURI     string
		wantErr     bool
	}{
		{
			name: "42 USC 1983",
			citation: &citation.Citation{
				Type: citation.CitationTypeCode,
				Components: citation.CitationComponents{
					CodeName: "USC",
					Title:    "42",
					Section:  "1983",
				},
			},
			wantTitle:   "42",
			wantSection: "1983",
			wantURI:     "https://uscode.house.gov/view.xhtml?req=granuleid:USC-prelim-title42-section1983&edition=prelim",
			wantErr:     false,
		},
		{
			name: "15 USC 1681",
			citation: &citation.Citation{
				Type: citation.CitationTypeCode,
				Components: citation.CitationComponents{
					CodeName: "USC",
					Title:    "15",
					Section:  "1681",
				},
			},
			wantTitle:   "15",
			wantSection: "1681",
			wantURI:     "https://uscode.house.gov/view.xhtml?req=granuleid:USC-prelim-title15-section1681&edition=prelim",
			wantErr:     false,
		},
		{
			name:     "nil citation",
			citation: nil,
			wantErr:  true,
		},
		{
			name: "non-USC citation",
			citation: &citation.Citation{
				Type: citation.CitationTypeCode,
				Components: citation.CitationComponents{
					CodeName: "CFR",
					Title:    "45",
					Section:  "164",
				},
			},
			wantErr: true,
		},
		{
			name: "missing title",
			citation: &citation.Citation{
				Type: citation.CitationTypeCode,
				Components: citation.CitationComponents{
					CodeName: "USC",
					Section:  "1983",
				},
			},
			wantErr: true,
		},
		{
			name: "missing section",
			citation: &citation.Citation{
				Type: citation.CitationTypeCode,
				Components: citation.CitationComponents{
					CodeName: "USC",
					Title:    "42",
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := GenerateUSCURI(tc.citation)
			if tc.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Title != tc.wantTitle {
				t.Errorf("Title: got %q, want %q", result.Title, tc.wantTitle)
			}
			if result.Section != tc.wantSection {
				t.Errorf("Section: got %q, want %q", result.Section, tc.wantSection)
			}
			if result.String() != tc.wantURI {
				t.Errorf("URI: got %q, want %q", result.String(), tc.wantURI)
			}
		})
	}
}

func TestGenerateCFRURI(t *testing.T) {
	testCases := []struct {
		name        string
		citation    *citation.Citation
		wantTitle   string
		wantPart    string
		wantSection string
		wantURI     string
		wantErr     bool
	}{
		{
			name: "45 CFR 164 (part only)",
			citation: &citation.Citation{
				Type: citation.CitationTypeCode,
				Components: citation.CitationComponents{
					CodeName: "CFR",
					Title:    "45",
					Section:  "164",
				},
			},
			wantTitle:   "45",
			wantPart:    "164",
			wantSection: "",
			wantURI:     "https://www.ecfr.gov/current/title-45/part-164",
			wantErr:     false,
		},
		{
			name: "45 CFR 164.502 (with section)",
			citation: &citation.Citation{
				Type: citation.CitationTypeCode,
				Components: citation.CitationComponents{
					CodeName: "CFR",
					Title:    "45",
					Section:  "164.502",
				},
			},
			wantTitle:   "45",
			wantPart:    "164",
			wantSection: "502",
			wantURI:     "https://www.ecfr.gov/current/title-45/part-164/section-164.502",
			wantErr:     false,
		},
		{
			name: "16 CFR 312 (COPPA)",
			citation: &citation.Citation{
				Type: citation.CitationTypeCode,
				Components: citation.CitationComponents{
					CodeName: "CFR",
					Title:    "16",
					Section:  "312",
				},
			},
			wantTitle:   "16",
			wantPart:    "312",
			wantSection: "",
			wantURI:     "https://www.ecfr.gov/current/title-16/part-312",
			wantErr:     false,
		},
		{
			name:     "nil citation",
			citation: nil,
			wantErr:  true,
		},
		{
			name: "non-CFR citation",
			citation: &citation.Citation{
				Type: citation.CitationTypeCode,
				Components: citation.CitationComponents{
					CodeName: "USC",
					Title:    "42",
					Section:  "1983",
				},
			},
			wantErr: true,
		},
		{
			name: "missing title",
			citation: &citation.Citation{
				Type: citation.CitationTypeCode,
				Components: citation.CitationComponents{
					CodeName: "CFR",
					Section:  "164",
				},
			},
			wantErr: true,
		},
		{
			name: "missing section",
			citation: &citation.Citation{
				Type: citation.CitationTypeCode,
				Components: citation.CitationComponents{
					CodeName: "CFR",
					Title:    "45",
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := GenerateCFRURI(tc.citation)
			if tc.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Title != tc.wantTitle {
				t.Errorf("Title: got %q, want %q", result.Title, tc.wantTitle)
			}
			if result.Part != tc.wantPart {
				t.Errorf("Part: got %q, want %q", result.Part, tc.wantPart)
			}
			if result.Section != tc.wantSection {
				t.Errorf("Section: got %q, want %q", result.Section, tc.wantSection)
			}
			if result.String() != tc.wantURI {
				t.Errorf("URI: got %q, want %q", result.String(), tc.wantURI)
			}
		})
	}
}

func TestParseUSCNumber(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		wantTitle   string
		wantSection string
		wantErr     bool
	}{
		{
			name:        "standard format with section symbol",
			input:       "42 U.S.C. § 1983",
			wantTitle:   "42",
			wantSection: "1983",
		},
		{
			name:        "with 'Section' word",
			input:       "15 U.S.C. Section 1681",
			wantTitle:   "15",
			wantSection: "1681",
		},
		{
			name:        "simple format",
			input:       "42 USC 1983",
			wantTitle:   "42",
			wantSection: "1983",
		},
		{
			name:        "with Sec. abbreviation",
			input:       "5 U.S.C. Sec. 552",
			wantTitle:   "5",
			wantSection: "552",
		},
		{
			name:        "with extra whitespace",
			input:       "  42  U.S.C.  §  1983  ",
			wantTitle:   "42",
			wantSection: "1983",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "too few parts",
			input:   "42 USC",
			wantErr: true,
		},
		{
			name:    "no USC marker",
			input:   "42 CFR 164",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseUSCNumber(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Title != tc.wantTitle {
				t.Errorf("Title: got %q, want %q", result.Title, tc.wantTitle)
			}
			if result.Section != tc.wantSection {
				t.Errorf("Section: got %q, want %q", result.Section, tc.wantSection)
			}
		})
	}
}

func TestParseCFRNumber(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		wantTitle   string
		wantPart    string
		wantSection string
		wantErr     bool
	}{
		{
			name:        "part only",
			input:       "45 C.F.R. Part 164",
			wantTitle:   "45",
			wantPart:    "164",
			wantSection: "",
		},
		{
			name:        "with section symbol and subsection",
			input:       "45 C.F.R. § 164.502",
			wantTitle:   "45",
			wantPart:    "164",
			wantSection: "502",
		},
		{
			name:        "simple format",
			input:       "45 CFR 164",
			wantTitle:   "45",
			wantPart:    "164",
			wantSection: "",
		},
		{
			name:        "simple format with section",
			input:       "16 CFR 312.3",
			wantTitle:   "16",
			wantPart:    "312",
			wantSection: "3",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "too few parts",
			input:   "45 CFR",
			wantErr: true,
		},
		{
			name:    "no CFR marker",
			input:   "42 USC 1983",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseCFRNumber(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Title != tc.wantTitle {
				t.Errorf("Title: got %q, want %q", result.Title, tc.wantTitle)
			}
			if result.Part != tc.wantPart {
				t.Errorf("Part: got %q, want %q", result.Part, tc.wantPart)
			}
			if result.Section != tc.wantSection {
				t.Errorf("Section: got %q, want %q", result.Section, tc.wantSection)
			}
		})
	}
}

func TestUSCNumberString(t *testing.T) {
	testCases := []struct {
		name     string
		usc      USCNumber
		expected string
	}{
		{
			name:     "basic",
			usc:      USCNumber{Title: "42", Section: "1983"},
			expected: "42 U.S.C. § 1983",
		},
		{
			name:     "with subsection",
			usc:      USCNumber{Title: "42", Section: "1983", Subsection: "(a)"},
			expected: "42 U.S.C. § 1983(a)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.usc.String()
			if result != tc.expected {
				t.Errorf("String(): got %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestCFRNumberString(t *testing.T) {
	testCases := []struct {
		name     string
		cfr      CFRNumber
		expected string
	}{
		{
			name:     "part only",
			cfr:      CFRNumber{Title: "45", Part: "164"},
			expected: "45 C.F.R. § 164",
		},
		{
			name:     "with section",
			cfr:      CFRNumber{Title: "45", Part: "164", Section: "502"},
			expected: "45 C.F.R. § 164.502",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.cfr.String()
			if result != tc.expected {
				t.Errorf("String(): got %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestUSCURIString(t *testing.T) {
	testCases := []struct {
		name     string
		uri      USCURI
		expected string
	}{
		{
			name:     "42 USC 1983",
			uri:      USCURI{Title: "42", Section: "1983"},
			expected: "https://uscode.house.gov/view.xhtml?req=granuleid:USC-prelim-title42-section1983&edition=prelim",
		},
		{
			name:     "15 USC 1681",
			uri:      USCURI{Title: "15", Section: "1681"},
			expected: "https://uscode.house.gov/view.xhtml?req=granuleid:USC-prelim-title15-section1681&edition=prelim",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.uri.String()
			if result != tc.expected {
				t.Errorf("String(): got %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestCFRURIString(t *testing.T) {
	testCases := []struct {
		name     string
		uri      CFRURI
		expected string
	}{
		{
			name:     "part only",
			uri:      CFRURI{Title: "45", Part: "164"},
			expected: "https://www.ecfr.gov/current/title-45/part-164",
		},
		{
			name:     "with section",
			uri:      CFRURI{Title: "45", Part: "164", Section: "502"},
			expected: "https://www.ecfr.gov/current/title-45/part-164/section-164.502",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.uri.String()
			if result != tc.expected {
				t.Errorf("String(): got %q, want %q", result, tc.expected)
			}
		})
	}
}
