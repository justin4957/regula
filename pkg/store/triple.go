package store

import "fmt"

// Triple represents an RDF Subject-Predicate-Object triple.
// In the regulation domain:
//   - Subject: typically a provision URI (e.g., "GDPR:Art17")
//   - Predicate: a relationship (e.g., "reg:references", "rdf:type")
//   - Object: another URI or literal value
type Triple struct {
	Subject   string
	Predicate string
	Object    string
}

// NewTriple creates a new triple with the given components.
func NewTriple(subject, predicate, object string) Triple {
	return Triple{
		Subject:   subject,
		Predicate: predicate,
		Object:    object,
	}
}

// Equals checks if two triples have identical components.
func (t Triple) Equals(other Triple) bool {
	return t.Subject == other.Subject &&
		t.Predicate == other.Predicate &&
		t.Object == other.Object
}

// String returns a human-readable representation of the triple.
func (t Triple) String() string {
	return fmt.Sprintf("<%s> <%s> <%s>", t.Subject, t.Predicate, t.Object)
}

// NTriples returns the triple in N-Triples format.
func (t Triple) NTriples() string {
	return fmt.Sprintf("<%s> <%s> <%s> .", t.Subject, t.Predicate, t.Object)
}

// IsValid returns true if all components are non-empty.
func (t Triple) IsValid() bool {
	return t.Subject != "" && t.Predicate != "" && t.Object != ""
}

// TriplePattern represents a pattern for matching triples.
// Empty strings act as wildcards that match any value.
type TriplePattern struct {
	Subject   string
	Predicate string
	Object    string
}

// NewTriplePattern creates a new pattern for querying.
// Use empty string "" for wildcards.
func NewTriplePattern(subject, predicate, object string) TriplePattern {
	return TriplePattern{
		Subject:   subject,
		Predicate: predicate,
		Object:    object,
	}
}

// Matches checks if a triple matches this pattern.
func (p TriplePattern) Matches(t Triple) bool {
	if p.Subject != "" && p.Subject != t.Subject {
		return false
	}
	if p.Predicate != "" && p.Predicate != t.Predicate {
		return false
	}
	if p.Object != "" && p.Object != t.Object {
		return false
	}
	return true
}

// HasWildcards returns true if any component is a wildcard.
func (p TriplePattern) HasWildcards() bool {
	return p.Subject == "" || p.Predicate == "" || p.Object == ""
}

// WildcardCount returns the number of wildcard components.
func (p TriplePattern) WildcardCount() int {
	count := 0
	if p.Subject == "" {
		count++
	}
	if p.Predicate == "" {
		count++
	}
	if p.Object == "" {
		count++
	}
	return count
}
