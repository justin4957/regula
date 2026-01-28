package citation

// CitationParser parses legal citations from text.
// Each parser handles one or more jurisdictions and citation styles
// (e.g., EU citations, Bluebook, OSCOLA).
// Implementations must be safe for concurrent use.
type CitationParser interface {
	// Name returns the human-readable parser name (e.g., "EU Citation Parser", "Bluebook").
	Name() string

	// Jurisdictions returns the jurisdiction codes this parser supports.
	// Codes follow conventions in pkg/types/jurisdiction.go (e.g., "EU", "US", "US-CA", "UK").
	Jurisdictions() []string

	// Parse extracts all citations from the given text.
	// Returns an empty slice (not nil) if no citations are found.
	Parse(text string) ([]*Citation, error)

	// Normalize converts a citation to its canonical string form.
	// For example, "Dir. 95/46" becomes "Directive 95/46/EC".
	Normalize(citation *Citation) string

	// ToURI generates a canonical URI for the citation.
	// URI schemes follow existing conventions (e.g., "urn:eu:regulation:2016/679").
	ToURI(citation *Citation) (string, error)
}
