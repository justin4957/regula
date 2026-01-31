package fetch

import (
	"testing"
)

func TestMapURN_EURegulation(t *testing.T) {
	urnMapper := NewURNMapper()

	cases := []struct {
		name        string
		urn         string
		expectedURL string
	}{
		{
			name:        "GDPR regulation",
			urn:         "urn:eu:regulation:2016/679",
			expectedURL: "http://data.europa.eu/eli/reg/2016/679/oj",
		},
		{
			name:        "Regulation 45/2001",
			urn:         "urn:eu:regulation:2001/45",
			expectedURL: "http://data.europa.eu/eli/reg/2001/45/oj",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resultURL, err := urnMapper.MapURN(tc.urn)
			if err != nil {
				t.Fatalf("MapURN(%q) returned error: %v", tc.urn, err)
			}
			if resultURL != tc.expectedURL {
				t.Errorf("MapURN(%q): got %q, want %q", tc.urn, resultURL, tc.expectedURL)
			}
		})
	}
}

func TestMapURN_EUDirective(t *testing.T) {
	urnMapper := NewURNMapper()

	cases := []struct {
		name        string
		urn         string
		expectedURL string
	}{
		{
			name:        "Data Protection Directive",
			urn:         "urn:eu:directive:1995/46",
			expectedURL: "http://data.europa.eu/eli/dir/1995/46/oj",
		},
		{
			name:        "ePrivacy Directive",
			urn:         "urn:eu:directive:2002/58",
			expectedURL: "http://data.europa.eu/eli/dir/2002/58/oj",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resultURL, err := urnMapper.MapURN(tc.urn)
			if err != nil {
				t.Fatalf("MapURN(%q) returned error: %v", tc.urn, err)
			}
			if resultURL != tc.expectedURL {
				t.Errorf("MapURN(%q): got %q, want %q", tc.urn, resultURL, tc.expectedURL)
			}
		})
	}
}

func TestMapURN_EUDecision(t *testing.T) {
	urnMapper := NewURNMapper()

	resultURL, err := urnMapper.MapURN("urn:eu:decision:2010/87")
	if err != nil {
		t.Fatalf("MapURN returned error: %v", err)
	}

	expectedURL := "http://data.europa.eu/eli/dec/2010/87/oj"
	if resultURL != expectedURL {
		t.Errorf("MapURN: got %q, want %q", resultURL, expectedURL)
	}
}

func TestMapURN_Treaty(t *testing.T) {
	urnMapper := NewURNMapper()

	_, err := urnMapper.MapURN("urn:eu:treaty:TFEU")
	if err == nil {
		t.Error("Expected error for treaty URN, got nil")
	}
}

func TestMapURN_USSource(t *testing.T) {
	urnMapper := NewURNMapper()

	// USC and CFR URNs should now resolve successfully
	successURNs := []struct {
		urn      string
		contains string
	}{
		{"urn:us:usc:18/17014", "uscode.house.gov"},
		{"urn:us:cfr:47/222", "ecfr.gov"},
	}

	for _, testCase := range successURNs {
		t.Run(testCase.urn, func(t *testing.T) {
			resolvedURL, err := urnMapper.MapURN(testCase.urn)
			if err != nil {
				t.Errorf("Expected success for URN %q, got error: %v", testCase.urn, err)
			}
			if resolvedURL == "" {
				t.Errorf("Expected non-empty URL for URN %q", testCase.urn)
			}
		})
	}

	// Other US URN subtypes still return errors
	errorURNs := []string{
		"urn:us:pl:116-283",
		"urn:us:ca:title1/sec100",
	}

	for _, urn := range errorURNs {
		t.Run(urn, func(t *testing.T) {
			_, err := urnMapper.MapURN(urn)
			if err == nil {
				t.Errorf("Expected error for US URN %q, got nil", urn)
			}
		})
	}
}

func TestMapURN_GenericExternal(t *testing.T) {
	urnMapper := NewURNMapper()

	_, err := urnMapper.MapURN("urn:external:some-document")
	if err == nil {
		t.Error("Expected error for generic external URN, got nil")
	}
}

func TestMapURN_EmptyURN(t *testing.T) {
	urnMapper := NewURNMapper()

	_, err := urnMapper.MapURN("")
	if err == nil {
		t.Error("Expected error for empty URN, got nil")
	}
}

func TestMapURN_UnrecognizedFormat(t *testing.T) {
	urnMapper := NewURNMapper()

	_, err := urnMapper.MapURN("not-a-urn")
	if err == nil {
		t.Error("Expected error for unrecognized URN format, got nil")
	}
}

func TestParseEUDocURN(t *testing.T) {
	cases := []struct {
		name           string
		urn            string
		prefix         string
		expectedYear   string
		expectedNumber string
		expectError    bool
	}{
		{
			name:           "valid regulation URN",
			urn:            "urn:eu:regulation:2016/679",
			prefix:         "urn:eu:regulation:",
			expectedYear:   "2016",
			expectedNumber: "679",
		},
		{
			name:           "valid directive URN",
			urn:            "urn:eu:directive:1995/46",
			prefix:         "urn:eu:directive:",
			expectedYear:   "1995",
			expectedNumber: "46",
		},
		{
			name:        "missing number",
			urn:         "urn:eu:regulation:2016/",
			prefix:      "urn:eu:regulation:",
			expectError: true,
		},
		{
			name:        "missing year",
			urn:         "urn:eu:regulation:/679",
			prefix:      "urn:eu:regulation:",
			expectError: true,
		},
		{
			name:        "no slash separator",
			urn:         "urn:eu:regulation:2016",
			prefix:      "urn:eu:regulation:",
			expectError: true,
		},
		{
			name:        "empty suffix",
			urn:         "urn:eu:regulation:",
			prefix:      "urn:eu:regulation:",
			expectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			year, number, err := parseEUDocURN(tc.urn, tc.prefix)
			if tc.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if year != tc.expectedYear {
				t.Errorf("Year: got %q, want %q", year, tc.expectedYear)
			}
			if number != tc.expectedNumber {
				t.Errorf("Number: got %q, want %q", number, tc.expectedNumber)
			}
		})
	}
}
