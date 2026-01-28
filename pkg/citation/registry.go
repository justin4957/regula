package citation

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// CitationRegistry manages a collection of citation parsers and dispatches
// parse requests to the appropriate parser(s) based on jurisdiction.
// Thread-safe for concurrent use.
type CitationRegistry struct {
	mu                sync.RWMutex
	parsers           map[string]CitationParser
	jurisdictionIndex map[string][]string // uppercase jurisdiction -> parser names
}

// NewCitationRegistry creates an empty citation registry.
func NewCitationRegistry() *CitationRegistry {
	return &CitationRegistry{
		parsers:           make(map[string]CitationParser),
		jurisdictionIndex: make(map[string][]string),
	}
}

// Register adds a parser to the registry.
// Returns an error if the parser is nil, has an empty name, or a parser
// with the same name is already registered.
func (r *CitationRegistry) Register(parser CitationParser) error {
	if parser == nil {
		return fmt.Errorf("citation parser cannot be nil")
	}
	parserName := parser.Name()
	if parserName == "" {
		return fmt.Errorf("citation parser name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.parsers[parserName]; exists {
		return fmt.Errorf("citation parser %q already registered", parserName)
	}

	r.parsers[parserName] = parser

	for _, jurisdiction := range parser.Jurisdictions() {
		jurisdictionKey := strings.ToUpper(jurisdiction)
		r.jurisdictionIndex[jurisdictionKey] = append(
			r.jurisdictionIndex[jurisdictionKey], parserName,
		)
	}

	return nil
}

// Unregister removes a parser by name.
// Returns an error if the parser is not found.
func (r *CitationRegistry) Unregister(parserName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	parser, exists := r.parsers[parserName]
	if !exists {
		return fmt.Errorf("citation parser %q not found", parserName)
	}

	// Remove from jurisdiction index.
	for _, jurisdiction := range parser.Jurisdictions() {
		jurisdictionKey := strings.ToUpper(jurisdiction)
		existingNames := r.jurisdictionIndex[jurisdictionKey]
		filteredNames := make([]string, 0, len(existingNames))
		for _, existingName := range existingNames {
			if existingName != parserName {
				filteredNames = append(filteredNames, existingName)
			}
		}
		if len(filteredNames) == 0 {
			delete(r.jurisdictionIndex, jurisdictionKey)
		} else {
			r.jurisdictionIndex[jurisdictionKey] = filteredNames
		}
	}

	delete(r.parsers, parserName)
	return nil
}

// Get returns a parser by name.
func (r *CitationRegistry) Get(parserName string) (CitationParser, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	parser, ok := r.parsers[parserName]
	return parser, ok
}

// List returns all registered parser names in sorted order.
func (r *CitationRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	parserNames := make([]string, 0, len(r.parsers))
	for parserName := range r.parsers {
		parserNames = append(parserNames, parserName)
	}
	sort.Strings(parserNames)
	return parserNames
}

// ListByJurisdiction returns parser names registered for a given jurisdiction.
// Jurisdiction matching is case-insensitive.
func (r *CitationRegistry) ListByJurisdiction(jurisdiction string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	jurisdictionKey := strings.ToUpper(jurisdiction)
	parserNames := r.jurisdictionIndex[jurisdictionKey]
	if parserNames == nil {
		return []string{}
	}
	result := make([]string, len(parserNames))
	copy(result, parserNames)
	return result
}

// ParseAll runs parsers (optionally filtered by jurisdiction) and returns
// merged, deduplicated results sorted by text position then confidence.
// If jurisdiction is empty, all registered parsers are run.
func (r *CitationRegistry) ParseAll(text string, jurisdiction string) []*Citation {
	r.mu.RLock()
	selectedParsers := r.selectParsers(jurisdiction)
	r.mu.RUnlock()

	var allCitations []*Citation
	for _, parser := range selectedParsers {
		citations, err := parser.Parse(text)
		if err != nil {
			continue // skip failing parsers
		}
		allCitations = append(allCitations, citations...)
	}

	// Sort by text offset, then by confidence descending.
	sort.Slice(allCitations, func(i, j int) bool {
		if allCitations[i].TextOffset != allCitations[j].TextOffset {
			return allCitations[i].TextOffset < allCitations[j].TextOffset
		}
		return allCitations[i].Confidence > allCitations[j].Confidence
	})

	return deduplicateCitations(allCitations)
}

// Count returns the number of registered parsers.
func (r *CitationRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.parsers)
}

// selectParsers returns the parsers to run. Must be called under read lock.
func (r *CitationRegistry) selectParsers(jurisdiction string) []CitationParser {
	if jurisdiction == "" {
		selectedParsers := make([]CitationParser, 0, len(r.parsers))
		for _, parser := range r.parsers {
			selectedParsers = append(selectedParsers, parser)
		}
		return selectedParsers
	}

	jurisdictionKey := strings.ToUpper(jurisdiction)
	parserNames := r.jurisdictionIndex[jurisdictionKey]
	selectedParsers := make([]CitationParser, 0, len(parserNames))
	for _, parserName := range parserNames {
		if parser, ok := r.parsers[parserName]; ok {
			selectedParsers = append(selectedParsers, parser)
		}
	}
	return selectedParsers
}

// deduplicateCitations removes duplicate citations based on overlapping text
// positions, keeping the one with higher confidence.
func deduplicateCitations(citations []*Citation) []*Citation {
	if len(citations) == 0 {
		return citations
	}

	deduplicated := make([]*Citation, 0, len(citations))
	for _, candidate := range citations {
		overlapping := false
		for existingIndex, existing := range deduplicated {
			if citationsOverlap(candidate, existing) {
				if candidate.Confidence > existing.Confidence {
					deduplicated[existingIndex] = candidate
				}
				overlapping = true
				break
			}
		}
		if !overlapping {
			deduplicated = append(deduplicated, candidate)
		}
	}
	return deduplicated
}

// citationsOverlap checks if two citations overlap in text position.
func citationsOverlap(a, b *Citation) bool {
	if a.TextOffset == 0 && a.TextLength == 0 {
		return a.RawText == b.RawText
	}
	aEnd := a.TextOffset + a.TextLength
	bEnd := b.TextOffset + b.TextLength
	return a.TextOffset < bEnd && b.TextOffset < aEnd
}
