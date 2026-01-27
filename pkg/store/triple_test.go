package store

import "testing"

func TestNewTriple(t *testing.T) {
	triple := NewTriple("GDPR:Art1", "rdf:type", "reg:Article")

	if triple.Subject != "GDPR:Art1" {
		t.Errorf("Subject mismatch: got %s", triple.Subject)
	}
	if triple.Predicate != "rdf:type" {
		t.Errorf("Predicate mismatch: got %s", triple.Predicate)
	}
	if triple.Object != "reg:Article" {
		t.Errorf("Object mismatch: got %s", triple.Object)
	}
}

func TestTriple_Equals(t *testing.T) {
	t1 := NewTriple("GDPR:Art1", "rdf:type", "reg:Article")
	t2 := NewTriple("GDPR:Art1", "rdf:type", "reg:Article")
	t3 := NewTriple("GDPR:Art2", "rdf:type", "reg:Article")

	if !t1.Equals(t2) {
		t.Error("Identical triples should be equal")
	}

	if t1.Equals(t3) {
		t.Error("Different triples should not be equal")
	}
}

func TestTriple_String(t *testing.T) {
	triple := NewTriple("GDPR:Art1", "rdf:type", "reg:Article")
	s := triple.String()

	expected := "<GDPR:Art1> <rdf:type> <reg:Article>"
	if s != expected {
		t.Errorf("String mismatch: got %s, want %s", s, expected)
	}
}

func TestTriple_NTriples(t *testing.T) {
	triple := NewTriple("GDPR:Art1", "rdf:type", "reg:Article")
	s := triple.NTriples()

	expected := "<GDPR:Art1> <rdf:type> <reg:Article> ."
	if s != expected {
		t.Errorf("NTriples mismatch: got %s, want %s", s, expected)
	}
}

func TestTriple_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		triple  Triple
		isValid bool
	}{
		{
			name:    "valid triple",
			triple:  NewTriple("s", "p", "o"),
			isValid: true,
		},
		{
			name:    "empty subject",
			triple:  Triple{Subject: "", Predicate: "p", Object: "o"},
			isValid: false,
		},
		{
			name:    "empty predicate",
			triple:  Triple{Subject: "s", Predicate: "", Object: "o"},
			isValid: false,
		},
		{
			name:    "empty object",
			triple:  Triple{Subject: "s", Predicate: "p", Object: ""},
			isValid: false,
		},
		{
			name:    "all empty",
			triple:  Triple{},
			isValid: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.triple.IsValid() != tc.isValid {
				t.Errorf("IsValid() = %v, want %v", tc.triple.IsValid(), tc.isValid)
			}
		})
	}
}

func TestNewTriplePattern(t *testing.T) {
	pattern := NewTriplePattern("s", "p", "")

	if pattern.Subject != "s" {
		t.Errorf("Subject mismatch: got %s", pattern.Subject)
	}
	if pattern.Predicate != "p" {
		t.Errorf("Predicate mismatch: got %s", pattern.Predicate)
	}
	if pattern.Object != "" {
		t.Errorf("Object should be empty wildcard, got %s", pattern.Object)
	}
}

func TestTriplePattern_Matches(t *testing.T) {
	triple := NewTriple("GDPR:Art1", "rdf:type", "reg:Article")

	tests := []struct {
		name    string
		pattern TriplePattern
		matches bool
	}{
		{
			name:    "all wildcards",
			pattern: NewTriplePattern("", "", ""),
			matches: true,
		},
		{
			name:    "exact match",
			pattern: NewTriplePattern("GDPR:Art1", "rdf:type", "reg:Article"),
			matches: true,
		},
		{
			name:    "subject only",
			pattern: NewTriplePattern("GDPR:Art1", "", ""),
			matches: true,
		},
		{
			name:    "predicate only",
			pattern: NewTriplePattern("", "rdf:type", ""),
			matches: true,
		},
		{
			name:    "object only",
			pattern: NewTriplePattern("", "", "reg:Article"),
			matches: true,
		},
		{
			name:    "subject and predicate",
			pattern: NewTriplePattern("GDPR:Art1", "rdf:type", ""),
			matches: true,
		},
		{
			name:    "wrong subject",
			pattern: NewTriplePattern("GDPR:Art2", "", ""),
			matches: false,
		},
		{
			name:    "wrong predicate",
			pattern: NewTriplePattern("", "reg:title", ""),
			matches: false,
		},
		{
			name:    "wrong object",
			pattern: NewTriplePattern("", "", "reg:Chapter"),
			matches: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.pattern.Matches(triple) != tc.matches {
				t.Errorf("Matches() = %v, want %v", tc.pattern.Matches(triple), tc.matches)
			}
		})
	}
}

func TestTriplePattern_HasWildcards(t *testing.T) {
	tests := []struct {
		name     string
		pattern  TriplePattern
		expected bool
	}{
		{
			name:     "no wildcards",
			pattern:  NewTriplePattern("s", "p", "o"),
			expected: false,
		},
		{
			name:     "subject wildcard",
			pattern:  NewTriplePattern("", "p", "o"),
			expected: true,
		},
		{
			name:     "predicate wildcard",
			pattern:  NewTriplePattern("s", "", "o"),
			expected: true,
		},
		{
			name:     "object wildcard",
			pattern:  NewTriplePattern("s", "p", ""),
			expected: true,
		},
		{
			name:     "all wildcards",
			pattern:  NewTriplePattern("", "", ""),
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.pattern.HasWildcards() != tc.expected {
				t.Errorf("HasWildcards() = %v, want %v", tc.pattern.HasWildcards(), tc.expected)
			}
		})
	}
}

func TestTriplePattern_WildcardCount(t *testing.T) {
	tests := []struct {
		name    string
		pattern TriplePattern
		count   int
	}{
		{
			name:    "no wildcards",
			pattern: NewTriplePattern("s", "p", "o"),
			count:   0,
		},
		{
			name:    "one wildcard",
			pattern: NewTriplePattern("", "p", "o"),
			count:   1,
		},
		{
			name:    "two wildcards",
			pattern: NewTriplePattern("", "p", ""),
			count:   2,
		},
		{
			name:    "all wildcards",
			pattern: NewTriplePattern("", "", ""),
			count:   3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.pattern.WildcardCount() != tc.count {
				t.Errorf("WildcardCount() = %d, want %d", tc.pattern.WildcardCount(), tc.count)
			}
		})
	}
}
