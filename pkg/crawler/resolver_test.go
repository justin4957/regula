package crawler

import (
	"strings"
	"testing"
)

func TestResolveUSCCitation(t *testing.T) {
	resolver := NewSourceResolver()

	testCases := []struct {
		name           string
		citation       string
		expectDocID    string
		expectDomain   string
		expectContains string
	}{
		{
			name:           "standard USC citation",
			citation:       "42 U.S.C. § 1320d",
			expectDocID:    "us-usc-42-1320d",
			expectDomain:   "uscode.house.gov",
			expectContains: "title42",
		},
		{
			name:           "USC without section symbol",
			citation:       "15 U.S.C. 6501",
			expectDocID:    "us-usc-15-6501",
			expectDomain:   "uscode.house.gov",
			expectContains: "title15",
		},
		{
			name:           "USC abbreviated",
			citation:       "42 USC 1320d",
			expectDocID:    "us-usc-42-1320d",
			expectDomain:   "uscode.house.gov",
			expectContains: "title42",
		},
		{
			name:           "USC with hyphenated section",
			citation:       "15 U.S.C. § 6501-6506",
			expectDocID:    "us-usc-15-6501-6506",
			expectDomain:   "uscode.house.gov",
			expectContains: "title15",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resolved, err := resolver.Resolve(testCase.citation)
			if err != nil {
				t.Fatalf("unexpected error resolving %q: %v", testCase.citation, err)
			}
			if resolved.DocumentID != testCase.expectDocID {
				t.Errorf("document ID = %q, want %q", resolved.DocumentID, testCase.expectDocID)
			}
			if resolved.Domain != testCase.expectDomain {
				t.Errorf("domain = %q, want %q", resolved.Domain, testCase.expectDomain)
			}
			if !strings.Contains(resolved.URL, testCase.expectContains) {
				t.Errorf("URL %q does not contain %q", resolved.URL, testCase.expectContains)
			}
		})
	}
}

func TestResolveCFRCitation(t *testing.T) {
	resolver := NewSourceResolver()

	testCases := []struct {
		name           string
		citation       string
		expectDocID    string
		expectDomain   string
		expectContains string
	}{
		{
			name:           "CFR part only",
			citation:       "45 C.F.R. Part 164",
			expectDocID:    "us-cfr-45-164",
			expectDomain:   "www.ecfr.gov",
			expectContains: "title-45",
		},
		{
			name:           "CFR with section",
			citation:       "45 CFR § 164.502",
			expectDocID:    "us-cfr-45-164-502",
			expectDomain:   "www.ecfr.gov",
			expectContains: "part-164",
		},
		{
			name:           "CFR abbreviated",
			citation:       "45 CFR Part 164",
			expectDocID:    "us-cfr-45-164",
			expectDomain:   "www.ecfr.gov",
			expectContains: "ecfr.gov",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resolved, err := resolver.Resolve(testCase.citation)
			if err != nil {
				t.Fatalf("unexpected error resolving %q: %v", testCase.citation, err)
			}
			if resolved.DocumentID != testCase.expectDocID {
				t.Errorf("document ID = %q, want %q", resolved.DocumentID, testCase.expectDocID)
			}
			if resolved.Domain != testCase.expectDomain {
				t.Errorf("domain = %q, want %q", resolved.Domain, testCase.expectDomain)
			}
			if !strings.Contains(resolved.URL, testCase.expectContains) {
				t.Errorf("URL %q does not contain %q", resolved.URL, testCase.expectContains)
			}
		})
	}
}

func TestResolvePublicLaw(t *testing.T) {
	resolver := NewSourceResolver()

	resolved, err := resolver.Resolve("Pub. L. 104-191")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.DocumentID != "us-pl-104-191" {
		t.Errorf("document ID = %q, want %q", resolved.DocumentID, "us-pl-104-191")
	}
	if !strings.Contains(resolved.URL, "congress.gov") {
		t.Errorf("URL %q does not contain congress.gov", resolved.URL)
	}
}

func TestResolveCaliforniaCode(t *testing.T) {
	resolver := NewSourceResolver()

	resolved, err := resolver.Resolve("Cal. Civ. Code § 1798.100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.DocumentID != "us-ca-civ-1798.100" {
		t.Errorf("document ID = %q, want %q", resolved.DocumentID, "us-ca-civ-1798.100")
	}
	if !strings.Contains(resolved.URL, "leginfo.legislature.ca.gov") {
		t.Errorf("URL %q does not contain CA legislature domain", resolved.URL)
	}
}

func TestResolveVirginiaCode(t *testing.T) {
	resolver := NewSourceResolver()

	resolved, err := resolver.Resolve("Va. Code Ann. § 59.1-575")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(resolved.DocumentID, "us-va-code-") {
		t.Errorf("document ID = %q, want prefix us-va-code-", resolved.DocumentID)
	}
	if !strings.Contains(resolved.URL, "virginia.gov") {
		t.Errorf("URL %q does not contain virginia.gov", resolved.URL)
	}
}

func TestResolveUnrecognizedCitation(t *testing.T) {
	resolver := NewSourceResolver()

	_, err := resolver.Resolve("some random text that is not a citation")
	if err == nil {
		t.Fatal("expected error for unrecognized citation, got nil")
	}
	if !strings.Contains(err.Error(), "unrecognized") {
		t.Errorf("error = %q, want 'unrecognized' in message", err.Error())
	}
}

func TestResolveEmptyCitation(t *testing.T) {
	resolver := NewSourceResolver()

	_, err := resolver.Resolve("")
	if err == nil {
		t.Fatal("expected error for empty citation")
	}
}

func TestResolveUSCURN(t *testing.T) {
	resolver := NewSourceResolver()

	resolved, err := resolver.ResolveURN("urn:us:usc:42/1320d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.DocumentID != "us-usc-42-1320d" {
		t.Errorf("document ID = %q, want %q", resolved.DocumentID, "us-usc-42-1320d")
	}
	if !strings.Contains(resolved.URL, "uscode.house.gov") {
		t.Errorf("URL %q does not contain uscode.house.gov", resolved.URL)
	}
}

func TestResolveCFRURN(t *testing.T) {
	resolver := NewSourceResolver()

	testCases := []struct {
		name        string
		urn         string
		expectDocID string
	}{
		{
			name:        "CFR part only",
			urn:         "urn:us:cfr:45/164",
			expectDocID: "us-cfr-45-164",
		},
		{
			name:        "CFR with section",
			urn:         "urn:us:cfr:45/164/502",
			expectDocID: "us-cfr-45-164-502",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resolved, err := resolver.ResolveURN(testCase.urn)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resolved.DocumentID != testCase.expectDocID {
				t.Errorf("document ID = %q, want %q", resolved.DocumentID, testCase.expectDocID)
			}
		})
	}
}

func TestResolveStateURN(t *testing.T) {
	resolver := NewSourceResolver()

	resolved, err := resolver.ResolveURN("urn:us:state:ca:civ:1798.100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(resolved.URL, "leginfo.legislature.ca.gov") {
		t.Errorf("URL %q does not contain CA domain", resolved.URL)
	}
}

func TestResolveUnsupportedURN(t *testing.T) {
	resolver := NewSourceResolver()

	_, err := resolver.ResolveURN("urn:eu:regulation:2016/679")
	if err == nil {
		t.Fatal("expected error for EU URN, got nil")
	}
}

func TestExtractDomainFromURL(t *testing.T) {
	testCases := []struct {
		rawURL   string
		expected string
	}{
		{"https://uscode.house.gov/view.xhtml?q=123", "uscode.house.gov"},
		{"https://www.ecfr.gov/current/title-45", "www.ecfr.gov"},
		{"", ""},
	}

	for _, testCase := range testCases {
		result := ExtractDomainFromURL(testCase.rawURL)
		if result != testCase.expected {
			t.Errorf("ExtractDomainFromURL(%q) = %q, want %q", testCase.rawURL, result, testCase.expected)
		}
	}
}
