package draft

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/coolbeans/regula/pkg/library"
	"github.com/coolbeans/regula/pkg/store"
)

// ConflictType classifies the nature of an obligation or rights conflict
// detected between a draft bill's amendments and existing law.
type ConflictType int

const (
	// ConflictObligationContradiction indicates opposing obligations — e.g., one
	// provision says "shall provide" while another says "shall not provide".
	ConflictObligationContradiction ConflictType = iota
	// ConflictObligationDuplicate indicates a redundant obligation that already
	// exists in another provision with the same obligation type and subject.
	ConflictObligationDuplicate
	// ConflictObligationOrphaned indicates an obligation is being removed (via
	// repeal) while other provisions still reference or depend on it.
	ConflictObligationOrphaned
	// ConflictRightsNarrowing indicates the draft restricts an existing right.
	ConflictRightsNarrowing
	// ConflictRightsContradiction indicates a right conflicts with an obligation.
	ConflictRightsContradiction
	// ConflictRightsExpansion indicates a new or expanded right (informational).
	ConflictRightsExpansion
)

// conflictTypeLabels maps conflict types to human-readable strings.
var conflictTypeLabels = [...]string{
	ConflictObligationContradiction: "obligation_contradiction",
	ConflictObligationDuplicate:     "obligation_duplicate",
	ConflictObligationOrphaned:      "obligation_orphaned",
	ConflictRightsNarrowing:         "rights_narrowing",
	ConflictRightsContradiction:     "rights_contradiction",
	ConflictRightsExpansion:         "rights_expansion",
}

// String returns a human-readable label for the conflict type.
func (ct ConflictType) String() string {
	if int(ct) < len(conflictTypeLabels) {
		return conflictTypeLabels[ct]
	}
	return "unknown"
}

// ConflictSeverity classifies the urgency of a detected conflict.
type ConflictSeverity int

const (
	// ConflictError indicates a direct contradiction that must be resolved.
	ConflictError ConflictSeverity = iota
	// ConflictWarning indicates a potential conflict that should be reviewed.
	ConflictWarning
	// ConflictInfo indicates redundancy or expansion — informational only.
	ConflictInfo
)

// conflictSeverityLabels maps severity levels to human-readable strings.
var conflictSeverityLabels = [...]string{
	ConflictError:   "error",
	ConflictWarning: "warning",
	ConflictInfo:    "info",
}

// String returns a human-readable label for the conflict severity.
func (cs ConflictSeverity) String() string {
	if int(cs) < len(conflictSeverityLabels) {
		return conflictSeverityLabels[cs]
	}
	return "unknown"
}

// Conflict represents a single detected conflict between a draft amendment and
// existing legislation. It captures the conflict type, severity, the source
// amendment, the conflicting existing provision, and relevant text.
type Conflict struct {
	Type              ConflictType     `json:"type"`
	Severity          ConflictSeverity `json:"severity"`
	SourceAmendment   Amendment        `json:"source_amendment"`
	ExistingProvision string           `json:"existing_provision"`
	ExistingText      string           `json:"existing_text"`
	ProposedText      string           `json:"proposed_text"`
	Description       string           `json:"description"`
}

// ConflictSummary aggregates conflict counts by severity and type.
type ConflictSummary struct {
	TotalConflicts int                  `json:"total_conflicts"`
	Errors         int                  `json:"errors"`
	Warnings       int                  `json:"warnings"`
	Infos          int                  `json:"infos"`
	ByType         map[ConflictType]int `json:"by_type"`
}

// ConflictReport contains all detected conflicts for a draft bill along with
// a summary of counts by severity and type.
type ConflictReport struct {
	Bill      *DraftBill      `json:"bill"`
	Conflicts []Conflict      `json:"conflicts"`
	Summary   ConflictSummary `json:"summary"`
}

// DetectObligationConflicts analyzes a computed diff and optional impact result
// against the knowledge graph to find obligation conflicts. It examines:
//   - Modified entries: checks if proposed text contradicts existing obligations
//   - Removed entries: checks if repealed obligations are depended on by other provisions
//   - Added entries: checks for duplicate obligations matching existing types
//
// Results are sorted by severity (errors first, then warnings, then info).
func DetectObligationConflicts(diff *DraftDiff, impact *DraftImpactResult, libraryPath string) (*ConflictReport, error) {
	if diff == nil {
		return nil, fmt.Errorf("diff is nil")
	}

	lib, err := library.Open(libraryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open library: %w", err)
	}

	report := &ConflictReport{
		Bill:      diff.Bill,
		Conflicts: []Conflict{},
	}

	tripleStoreCache := make(map[string]*store.TripleStore)

	// Modified entries: check for obligation contradictions
	for _, entry := range diff.Modified {
		tripleStore, loadErr := loadOrCacheTripleStore(lib, entry.TargetDocumentID, tripleStoreCache)
		if loadErr != nil {
			continue
		}
		conflicts := detectContradictions(entry, tripleStore)
		report.Conflicts = append(report.Conflicts, conflicts...)
	}

	// Removed entries: check for orphaned obligations
	for _, entry := range diff.Removed {
		tripleStore, loadErr := loadOrCacheTripleStore(lib, entry.TargetDocumentID, tripleStoreCache)
		if loadErr != nil {
			continue
		}
		conflicts := detectOrphanedObligations(entry, tripleStore)
		report.Conflicts = append(report.Conflicts, conflicts...)
	}

	// Added entries: check for duplicate obligations
	for _, entry := range diff.Added {
		tripleStore, loadErr := loadOrCacheTripleStore(lib, entry.TargetDocumentID, tripleStoreCache)
		if loadErr != nil {
			continue
		}
		conflicts := detectDuplicateObligations(entry, tripleStore)
		report.Conflicts = append(report.Conflicts, conflicts...)
	}

	sortConflicts(report.Conflicts)
	report.Summary = buildConflictSummary(report.Conflicts)

	return report, nil
}

// detectContradictions checks if a modified provision's proposed text contradicts
// any existing obligations on that provision. For each obligation linked to the
// target URI, it compares the existing obligation text against the proposed
// amendment text using keyword-based contradiction detection.
func detectContradictions(entry DiffEntry, tripleStore *store.TripleStore) []Conflict {
	var conflicts []Conflict

	obligationTriples := tripleStore.Find(entry.TargetURI, store.PropImposesObligation, "")
	for _, obligTriple := range obligationTriples {
		obligationURI := obligTriple.Object
		existingText := getObligationText(obligationURI, tripleStore)

		if existingText == "" {
			continue
		}

		proposedText := entry.ProposedText
		if proposedText == "" {
			proposedText = entry.Amendment.InsertText
		}

		if proposedText != "" && DetectObligationContradiction(proposedText, existingText) {
			conflicts = append(conflicts, Conflict{
				Type:              ConflictObligationContradiction,
				Severity:          classifyConflictSeverity(ConflictObligationContradiction),
				SourceAmendment:   entry.Amendment,
				ExistingProvision: obligationURI,
				ExistingText:      existingText,
				ProposedText:      proposedText,
				Description: fmt.Sprintf(
					"proposed amendment contradicts existing obligation in %s: existing directive conflicts with proposed text",
					extractURILabel(obligationURI),
				),
			})
		}
	}

	return conflicts
}

// detectOrphanedObligations checks if repealing a provision would orphan
// obligations that other provisions depend on. An obligation is orphaned when
// its parent provision is repealed but other provisions still reference it.
func detectOrphanedObligations(entry DiffEntry, tripleStore *store.TripleStore) []Conflict {
	var conflicts []Conflict

	obligationTriples := tripleStore.Find(entry.TargetURI, store.PropImposesObligation, "")
	for _, obligTriple := range obligationTriples {
		obligationURI := obligTriple.Object
		dependentURIs := FindDependentObligations(obligationURI, tripleStore)

		if len(dependentURIs) > 0 {
			existingText := getObligationText(obligationURI, tripleStore)
			dependentLabels := make([]string, 0, len(dependentURIs))
			for _, depURI := range dependentURIs {
				dependentLabels = append(dependentLabels, extractURILabel(depURI))
			}

			conflicts = append(conflicts, Conflict{
				Type:              ConflictObligationOrphaned,
				Severity:          classifyConflictSeverity(ConflictObligationOrphaned),
				SourceAmendment:   entry.Amendment,
				ExistingProvision: obligationURI,
				ExistingText:      existingText,
				Description: fmt.Sprintf(
					"repealing %s orphans obligation %s depended on by: %s",
					extractURILabel(entry.TargetURI),
					extractURILabel(obligationURI),
					strings.Join(dependentLabels, ", "),
				),
			})
		}
	}

	return conflicts
}

// detectDuplicateObligations checks if a new provision introduces obligations
// that duplicate existing obligations of the same type in the knowledge graph.
func detectDuplicateObligations(entry DiffEntry, tripleStore *store.TripleStore) []Conflict {
	var conflicts []Conflict

	proposedText := entry.ProposedText
	if proposedText == "" {
		proposedText = entry.Amendment.InsertText
	}
	if proposedText == "" {
		return conflicts
	}

	// Extract obligation keywords from the proposed text to determine its type
	proposedDirectives := extractDirectives(proposedText)
	if len(proposedDirectives) == 0 {
		return conflicts
	}

	// Find all existing obligations in the store
	allObligationTriples := tripleStore.Find("", store.RDFType, store.ClassObligation)
	for _, obligTriple := range allObligationTriples {
		existingObligURI := obligTriple.Subject
		existingText := getObligationText(existingObligURI, tripleStore)
		if existingText == "" {
			continue
		}

		existingDirectives := extractDirectives(existingText)
		if len(existingDirectives) == 0 {
			continue
		}

		// Check for duplicate: same directive polarity and overlapping subject matter
		if directivesDuplicate(proposedDirectives, existingDirectives) {
			parentURI := getObligationParent(existingObligURI, tripleStore)

			conflicts = append(conflicts, Conflict{
				Type:              ConflictObligationDuplicate,
				Severity:          classifyConflictSeverity(ConflictObligationDuplicate),
				SourceAmendment:   entry.Amendment,
				ExistingProvision: existingObligURI,
				ExistingText:      existingText,
				ProposedText:      proposedText,
				Description: fmt.Sprintf(
					"proposed obligation duplicates existing obligation in %s",
					extractURILabel(parentURI),
				),
			})
		}
	}

	return conflicts
}

// DetectObligationContradiction performs keyword-based detection of contradictory
// obligations between two texts. It extracts directive phrases (e.g., "shall",
// "shall not", "must provide", "must not disclose") and checks for opposing
// directives on overlapping subject matter.
func DetectObligationContradiction(draftText, existingText string) bool {
	draftDirectives := extractDirectives(draftText)
	existingDirectives := extractDirectives(existingText)

	for _, draftDirective := range draftDirectives {
		for _, existingDirective := range existingDirectives {
			if directivesContradict(draftDirective, existingDirective) {
				return true
			}
		}
	}

	return false
}

// FindDependentObligations finds provisions that reference or depend on the
// provision containing the given obligation. It returns URIs of provisions that
// would be affected if the obligation is removed.
func FindDependentObligations(obligationURI string, tripleStore *store.TripleStore) []string {
	// Find the parent provision of this obligation
	parentTriples := tripleStore.Find(obligationURI, store.PropPartOf, "")
	if len(parentTriples) == 0 {
		return nil
	}
	parentURI := parentTriples[0].Object

	var dependentURIs []string
	seen := make(map[string]bool)

	// Find provisions that reference the parent provision
	incomingRefTriples := tripleStore.Find("", store.PropReferences, parentURI)
	for _, triple := range incomingRefTriples {
		if !seen[triple.Subject] && triple.Subject != parentURI {
			seen[triple.Subject] = true
			dependentURIs = append(dependentURIs, triple.Subject)
		}
	}

	// Check inverse references
	referencedByTriples := tripleStore.Find(parentURI, store.PropReferencedBy, "")
	for _, triple := range referencedByTriples {
		if !seen[triple.Object] && triple.Object != parentURI {
			seen[triple.Object] = true
			dependentURIs = append(dependentURIs, triple.Object)
		}
	}

	return dependentURIs
}

// directive represents an extracted legislative directive phrase with its
// polarity (positive/negative) and associated subject matter keywords.
type directive struct {
	verb     string // "shall", "must", "may", "is required to", "is prohibited from"
	negated  bool   // true for "shall not", "must not", "may not", etc.
	keywords []string // subject matter keywords following the directive
}

// directivePatterns matches common legislative directive phrases.
var directivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(shall\s+not)\b\s+(.+?)(?:[.;]|$)`),
	regexp.MustCompile(`(?i)\b(must\s+not)\b\s+(.+?)(?:[.;]|$)`),
	regexp.MustCompile(`(?i)\b(may\s+not)\b\s+(.+?)(?:[.;]|$)`),
	regexp.MustCompile(`(?i)\b(is\s+prohibited\s+from)\b\s+(.+?)(?:[.;]|$)`),
	regexp.MustCompile(`(?i)\b(shall)\b\s+(.+?)(?:[.;]|$)`),
	regexp.MustCompile(`(?i)\b(must)\b\s+(.+?)(?:[.;]|$)`),
	regexp.MustCompile(`(?i)\b(is\s+required\s+to)\b\s+(.+?)(?:[.;]|$)`),
}

// negatedVerbs identifies which directive verbs are inherently negative.
var negatedVerbs = map[string]bool{
	"shall not":           true,
	"must not":            true,
	"may not":             true,
	"is prohibited from":  true,
}

// extractDirectives parses legislative text to extract directive phrases with
// their polarity and subject keywords.
func extractDirectives(text string) []directive {
	var directives []directive
	normalizedText := strings.Join(strings.Fields(text), " ")

	for _, pattern := range directivePatterns {
		matches := pattern.FindAllStringSubmatch(normalizedText, -1)
		for _, match := range matches {
			if len(match) < 3 {
				continue
			}
			verb := strings.ToLower(strings.TrimSpace(match[1]))
			subjectText := strings.ToLower(strings.TrimSpace(match[2]))
			keywords := extractSubjectKeywords(subjectText)

			directives = append(directives, directive{
				verb:     verb,
				negated:  negatedVerbs[verb],
				keywords: keywords,
			})
		}
	}

	return directives
}

// extractSubjectKeywords splits subject matter text into meaningful keywords,
// filtering out common stop words.
func extractSubjectKeywords(text string) []string {
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "of": true, "to": true,
		"in": true, "for": true, "and": true, "or": true, "with": true,
		"be": true, "by": true, "on": true, "at": true, "from": true,
		"as": true, "is": true, "it": true, "that": true, "this": true,
		"any": true, "all": true, "each": true, "such": true,
	}

	words := strings.Fields(text)
	var keywords []string
	for _, word := range words {
		cleaned := strings.Trim(word, ".,;:()\"'")
		if len(cleaned) > 2 && !stopWords[cleaned] {
			keywords = append(keywords, cleaned)
		}
	}
	return keywords
}

// directivesContradict checks if two directives have opposing polarity with
// overlapping subject matter. A contradiction occurs when one directive is
// positive and the other negative, and they share at least one subject keyword.
func directivesContradict(dirA, dirB directive) bool {
	// Both must have some subject keywords to compare
	if len(dirA.keywords) == 0 || len(dirB.keywords) == 0 {
		return false
	}

	// Opposing polarity required
	if dirA.negated == dirB.negated {
		return false
	}

	// Check for overlapping subject keywords
	keywordsA := make(map[string]bool, len(dirA.keywords))
	for _, keyword := range dirA.keywords {
		keywordsA[keyword] = true
	}

	for _, keyword := range dirB.keywords {
		if keywordsA[keyword] {
			return true
		}
	}

	return false
}

// directivesDuplicate checks if two sets of directives have the same polarity
// and overlapping subject matter, indicating a redundant obligation.
func directivesDuplicate(directivesA, directivesB []directive) bool {
	for _, dirA := range directivesA {
		for _, dirB := range directivesB {
			if dirA.negated == dirB.negated && len(dirA.keywords) > 0 && len(dirB.keywords) > 0 {
				// Check for significant keyword overlap (at least 2 shared keywords)
				keywordsA := make(map[string]bool, len(dirA.keywords))
				for _, keyword := range dirA.keywords {
					keywordsA[keyword] = true
				}

				sharedCount := 0
				for _, keyword := range dirB.keywords {
					if keywordsA[keyword] {
						sharedCount++
					}
				}

				if sharedCount >= 2 {
					return true
				}
			}
		}
	}
	return false
}

// classifyConflictSeverity maps a conflict type to its default severity level.
// Contradictions are errors, orphans are warnings, duplicates and expansions
// are informational.
func classifyConflictSeverity(conflictType ConflictType) ConflictSeverity {
	switch conflictType {
	case ConflictObligationContradiction, ConflictRightsContradiction:
		return ConflictError
	case ConflictObligationOrphaned, ConflictRightsNarrowing:
		return ConflictWarning
	case ConflictObligationDuplicate, ConflictRightsExpansion:
		return ConflictInfo
	default:
		return ConflictWarning
	}
}

// buildConflictSummary computes aggregate counts from a slice of conflicts.
func buildConflictSummary(conflicts []Conflict) ConflictSummary {
	summary := ConflictSummary{
		TotalConflicts: len(conflicts),
		ByType:         make(map[ConflictType]int),
	}

	for _, conflict := range conflicts {
		switch conflict.Severity {
		case ConflictError:
			summary.Errors++
		case ConflictWarning:
			summary.Warnings++
		case ConflictInfo:
			summary.Infos++
		}
		summary.ByType[conflict.Type]++
	}

	return summary
}

// sortConflicts sorts conflicts by severity (errors first), then by conflict
// type, then by existing provision URI for deterministic output.
func sortConflicts(conflicts []Conflict) {
	sort.Slice(conflicts, func(i, j int) bool {
		if conflicts[i].Severity != conflicts[j].Severity {
			return conflicts[i].Severity < conflicts[j].Severity
		}
		if conflicts[i].Type != conflicts[j].Type {
			return conflicts[i].Type < conflicts[j].Type
		}
		return conflicts[i].ExistingProvision < conflicts[j].ExistingProvision
	})
}

// getObligationText retrieves the text content of an obligation URI from the
// triple store.
func getObligationText(obligationURI string, tripleStore *store.TripleStore) string {
	textTriples := tripleStore.Find(obligationURI, store.PropText, "")
	if len(textTriples) > 0 {
		return textTriples[0].Object
	}
	return ""
}

// getObligationParent retrieves the parent provision URI of an obligation.
func getObligationParent(obligationURI string, tripleStore *store.TripleStore) string {
	parentTriples := tripleStore.Find(obligationURI, store.PropPartOf, "")
	if len(parentTriples) > 0 {
		return parentTriples[0].Object
	}
	return obligationURI
}
