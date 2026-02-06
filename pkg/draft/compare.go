package draft

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/coolbeans/regula/pkg/simulate"
	"github.com/coolbeans/regula/pkg/store"
)

// ScenarioComparison holds the results of comparing a scenario's compliance
// under baseline (current law) vs proposed (current law + draft amendments).
type ScenarioComparison struct {
	Scenario           string                 `json:"scenario"`
	Bill               *DraftBill             `json:"bill,omitempty"`
	Baseline           *simulate.MatchResult  `json:"baseline"`
	Proposed           *simulate.MatchResult  `json:"proposed"`
	NewlyApplicable    []ProvisionDiff        `json:"newly_applicable"`
	NoLongerApplicable []ProvisionDiff        `json:"no_longer_applicable"`
	ChangedRelevance   []ProvisionDiff        `json:"changed_relevance"`
	ObligationsDiff    DeltaSummary           `json:"obligations_diff"`
	RightsDiff         DeltaSummary           `json:"rights_diff"`
}

// ProvisionDiff represents a difference in how a provision applies between
// baseline and proposed scenarios.
type ProvisionDiff struct {
	URI               string `json:"uri"`
	Label             string `json:"label"`
	BaselineRelevance string `json:"baseline_relevance"`
	ProposedRelevance string `json:"proposed_relevance"`
	Reason            string `json:"reason"`
}

// DeltaSummary summarizes changes in a category (obligations or rights).
type DeltaSummary struct {
	Added   int `json:"added"`
	Removed int `json:"removed"`
	Changed int `json:"changed"`
}

// CompareScenarios runs the same compliance scenario against baseline and
// proposed triple stores, then diffs the results.
func CompareScenarios(
	scenarioName string,
	scenario *simulate.Scenario,
	baseStore *store.TripleStore,
	proposedStore *store.TripleStore,
	baseURI string,
	bill *DraftBill,
) (*ScenarioComparison, error) {
	if scenario == nil {
		return nil, fmt.Errorf("scenario is nil")
	}
	if baseStore == nil {
		return nil, fmt.Errorf("base store is nil")
	}
	if proposedStore == nil {
		return nil, fmt.Errorf("proposed store is nil")
	}

	// Create matchers for baseline and proposed stores
	// Note: We pass nil for annotations and doc since we're working from stored triples
	baseMatcher := simulate.NewProvisionMatcher(baseStore, baseURI, nil, nil)
	proposedMatcher := simulate.NewProvisionMatcher(proposedStore, baseURI, nil, nil)

	// Run the scenario against both stores
	baselineResult := baseMatcher.Match(scenario)
	proposedResult := proposedMatcher.Match(scenario)

	// Diff the results
	newlyApplicable, noLongerApplicable, changedRelevance := DiffMatchResults(baselineResult, proposedResult)

	// Calculate obligation and rights diffs
	obligationsDiff := diffObligations(baselineResult, proposedResult)
	rightsDiff := diffRights(baselineResult, proposedResult)

	return &ScenarioComparison{
		Scenario:           scenarioName,
		Bill:               bill,
		Baseline:           baselineResult,
		Proposed:           proposedResult,
		NewlyApplicable:    newlyApplicable,
		NoLongerApplicable: noLongerApplicable,
		ChangedRelevance:   changedRelevance,
		ObligationsDiff:    obligationsDiff,
		RightsDiff:         rightsDiff,
	}, nil
}

// DiffMatchResults compares baseline and proposed match results, returning:
// - newly: provisions that apply under proposed but not baseline
// - removed: provisions that apply under baseline but not proposed
// - changed: provisions where relevance level changed
func DiffMatchResults(baseline, proposed *simulate.MatchResult) (newly, removed, changed []ProvisionDiff) {
	// Build maps of URI -> MatchedProvision for quick lookup
	baselineMap := buildProvisionMap(baseline)
	proposedMap := buildProvisionMap(proposed)

	// Find newly applicable (in proposed but not baseline)
	for uri, prop := range proposedMap {
		if _, exists := baselineMap[uri]; !exists {
			newly = append(newly, ProvisionDiff{
				URI:               uri,
				Label:             prop.Title,
				BaselineRelevance: "",
				ProposedRelevance: string(prop.Relevance),
				Reason:            "New provision applies under proposed legislation",
			})
		}
	}

	// Find no longer applicable (in baseline but not proposed)
	for uri, prop := range baselineMap {
		if _, exists := proposedMap[uri]; !exists {
			removed = append(removed, ProvisionDiff{
				URI:               uri,
				Label:             prop.Title,
				BaselineRelevance: string(prop.Relevance),
				ProposedRelevance: "",
				Reason:            "Provision no longer applies under proposed legislation",
			})
		}
	}

	// Find changed relevance (in both, but with different relevance)
	for uri, baseProp := range baselineMap {
		if propProp, exists := proposedMap[uri]; exists {
			if baseProp.Relevance != propProp.Relevance {
				changed = append(changed, ProvisionDiff{
					URI:               uri,
					Label:             baseProp.Title,
					BaselineRelevance: string(baseProp.Relevance),
					ProposedRelevance: string(propProp.Relevance),
					Reason:            formatRelevanceChange(baseProp.Relevance, propProp.Relevance),
				})
			}
		}
	}

	// Sort for consistent output
	sortProvisionDiffs(newly)
	sortProvisionDiffs(removed)
	sortProvisionDiffs(changed)

	return newly, removed, changed
}

// buildProvisionMap creates a map of URI -> MatchedProvision from a MatchResult.
func buildProvisionMap(result *simulate.MatchResult) map[string]*simulate.MatchedProvision {
	provMap := make(map[string]*simulate.MatchedProvision)
	if result == nil {
		return provMap
	}

	for _, match := range result.AllMatches {
		provMap[match.URI] = match
	}
	return provMap
}

// formatRelevanceChange creates a human-readable description of a relevance change.
func formatRelevanceChange(from, to simulate.RelevanceScore) string {
	return fmt.Sprintf("Relevance changed from %s to %s", from, to)
}

// sortProvisionDiffs sorts provision diffs by URI for consistent output.
func sortProvisionDiffs(diffs []ProvisionDiff) {
	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].URI < diffs[j].URI
	})
}

// diffObligations computes the delta in obligations between baseline and proposed.
func diffObligations(baseline, proposed *simulate.MatchResult) DeltaSummary {
	baselineObligs := collectObligationTypes(baseline)
	proposedObligs := collectObligationTypes(proposed)

	return computeDelta(baselineObligs, proposedObligs)
}

// diffRights computes the delta in rights between baseline and proposed.
func diffRights(baseline, proposed *simulate.MatchResult) DeltaSummary {
	baselineRights := collectRightTypes(baseline)
	proposedRights := collectRightTypes(proposed)

	return computeDelta(baselineRights, proposedRights)
}

// collectObligationTypes collects all unique obligation types from match results.
func collectObligationTypes(result *simulate.MatchResult) map[string]bool {
	types := make(map[string]bool)
	if result == nil || result.Summary == nil {
		return types
	}

	for _, obligType := range result.Summary.ObligationsInvolved {
		types[string(obligType)] = true
	}
	return types
}

// collectRightTypes collects all unique right types from match results.
func collectRightTypes(result *simulate.MatchResult) map[string]bool {
	types := make(map[string]bool)
	if result == nil || result.Summary == nil {
		return types
	}

	for _, rightType := range result.Summary.RightsInvolved {
		types[string(rightType)] = true
	}
	return types
}

// computeDelta computes the added/removed/changed counts between two sets.
func computeDelta(baseline, proposed map[string]bool) DeltaSummary {
	var added, removed int

	for key := range proposed {
		if !baseline[key] {
			added++
		}
	}

	for key := range baseline {
		if !proposed[key] {
			removed++
		}
	}

	return DeltaSummary{
		Added:   added,
		Removed: removed,
		Changed: 0, // For obligation/right types, we only track add/remove
	}
}

// FormatScenarioComparison formats a scenario comparison for display.
// Supported formats: "table", "json"
func FormatScenarioComparison(comparison *ScenarioComparison, format string) string {
	if comparison == nil {
		return ""
	}

	switch strings.ToLower(format) {
	case "json":
		return formatComparisonJSON(comparison)
	case "table":
		fallthrough
	default:
		return formatComparisonTable(comparison)
	}
}

// formatComparisonTable formats the comparison as a readable table.
func formatComparisonTable(comparison *ScenarioComparison) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("Scenario Comparison: %s\n", comparison.Scenario))
	sb.WriteString(strings.Repeat("=", 60) + "\n\n")

	if comparison.Bill != nil {
		sb.WriteString(fmt.Sprintf("Bill: %s\n", comparison.Bill.Title))
		if comparison.Bill.BillNumber != "" {
			sb.WriteString(fmt.Sprintf("Number: %s\n", comparison.Bill.BillNumber))
		}
		sb.WriteString("\n")
	}

	// Summary counts
	sb.WriteString("Summary\n")
	sb.WriteString(strings.Repeat("-", 40) + "\n")
	baselineCount := 0
	proposedCount := 0
	if comparison.Baseline != nil && comparison.Baseline.Summary != nil {
		baselineCount = comparison.Baseline.Summary.TotalMatches
	}
	if comparison.Proposed != nil && comparison.Proposed.Summary != nil {
		proposedCount = comparison.Proposed.Summary.TotalMatches
	}
	sb.WriteString(fmt.Sprintf("%-25s %10s %10s\n", "", "Baseline", "Proposed"))
	sb.WriteString(fmt.Sprintf("%-25s %10d %10d\n", "Total Matches", baselineCount, proposedCount))
	sb.WriteString("\n")

	// Newly applicable provisions
	if len(comparison.NewlyApplicable) > 0 {
		sb.WriteString("Newly Applicable Provisions\n")
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		for _, diff := range comparison.NewlyApplicable {
			label := diff.Label
			if label == "" {
				label = extractCompareURILabel(diff.URI)
			}
			sb.WriteString(fmt.Sprintf("  + %s (%s)\n", label, diff.ProposedRelevance))
		}
		sb.WriteString("\n")
	}

	// No longer applicable provisions
	if len(comparison.NoLongerApplicable) > 0 {
		sb.WriteString("No Longer Applicable Provisions\n")
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		for _, diff := range comparison.NoLongerApplicable {
			label := diff.Label
			if label == "" {
				label = extractCompareURILabel(diff.URI)
			}
			sb.WriteString(fmt.Sprintf("  - %s (was %s)\n", label, diff.BaselineRelevance))
		}
		sb.WriteString("\n")
	}

	// Changed relevance
	if len(comparison.ChangedRelevance) > 0 {
		sb.WriteString("Changed Relevance\n")
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		for _, diff := range comparison.ChangedRelevance {
			label := diff.Label
			if label == "" {
				label = extractCompareURILabel(diff.URI)
			}
			sb.WriteString(fmt.Sprintf("  ~ %s: %s -> %s\n", label, diff.BaselineRelevance, diff.ProposedRelevance))
		}
		sb.WriteString("\n")
	}

	// Obligations diff
	if comparison.ObligationsDiff.Added > 0 || comparison.ObligationsDiff.Removed > 0 {
		sb.WriteString("Obligations Changes\n")
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		if comparison.ObligationsDiff.Added > 0 {
			sb.WriteString(fmt.Sprintf("  + %d new obligation type(s)\n", comparison.ObligationsDiff.Added))
		}
		if comparison.ObligationsDiff.Removed > 0 {
			sb.WriteString(fmt.Sprintf("  - %d obligation type(s) no longer apply\n", comparison.ObligationsDiff.Removed))
		}
		sb.WriteString("\n")
	}

	// Rights diff
	if comparison.RightsDiff.Added > 0 || comparison.RightsDiff.Removed > 0 {
		sb.WriteString("Rights Changes\n")
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		if comparison.RightsDiff.Added > 0 {
			sb.WriteString(fmt.Sprintf("  + %d new right type(s)\n", comparison.RightsDiff.Added))
		}
		if comparison.RightsDiff.Removed > 0 {
			sb.WriteString(fmt.Sprintf("  - %d right type(s) no longer apply\n", comparison.RightsDiff.Removed))
		}
		sb.WriteString("\n")
	}

	// No differences
	if len(comparison.NewlyApplicable) == 0 &&
		len(comparison.NoLongerApplicable) == 0 &&
		len(comparison.ChangedRelevance) == 0 &&
		comparison.ObligationsDiff.Added == 0 &&
		comparison.ObligationsDiff.Removed == 0 &&
		comparison.RightsDiff.Added == 0 &&
		comparison.RightsDiff.Removed == 0 {
		sb.WriteString("No differences detected between baseline and proposed.\n")
	}

	return sb.String()
}

// formatComparisonJSON formats the comparison as JSON.
func formatComparisonJSON(comparison *ScenarioComparison) string {
	data, err := json.MarshalIndent(comparison, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error())
	}
	return string(data)
}

// extractCompareURILabel extracts a human-readable label from a URI.
func extractCompareURILabel(uri string) string {
	if uri == "" {
		return ""
	}

	// Try to extract from common URI patterns
	// Check for fragment first (#)
	if idx := strings.LastIndex(uri, "#"); idx != -1 {
		return uri[idx+1:]
	}
	// Then check for colon (e.g., "GDPR:Art6" -> "Art6")
	// But skip colons that are part of URL scheme (://)
	if idx := strings.LastIndex(uri, ":"); idx != -1 {
		// Make sure it's not a URL scheme colon (not followed by //)
		if idx+2 < len(uri) && uri[idx+1:idx+3] != "//" {
			return uri[idx+1:]
		}
	}
	// Finally check for path separator
	if idx := strings.LastIndex(uri, "/"); idx != -1 {
		return uri[idx+1:]
	}
	return uri
}

// ComparisonSummary provides a brief summary of comparison results.
type ComparisonSummary struct {
	HasDifferences     bool `json:"has_differences"`
	NewlyApplicable    int  `json:"newly_applicable"`
	NoLongerApplicable int  `json:"no_longer_applicable"`
	ChangedRelevance   int  `json:"changed_relevance"`
	ObligationsAdded   int  `json:"obligations_added"`
	ObligationsRemoved int  `json:"obligations_removed"`
	RightsAdded        int  `json:"rights_added"`
	RightsRemoved      int  `json:"rights_removed"`
}

// GetSummary returns a brief summary of the comparison.
func (c *ScenarioComparison) GetSummary() ComparisonSummary {
	hasDiffs := len(c.NewlyApplicable) > 0 ||
		len(c.NoLongerApplicable) > 0 ||
		len(c.ChangedRelevance) > 0 ||
		c.ObligationsDiff.Added > 0 ||
		c.ObligationsDiff.Removed > 0 ||
		c.RightsDiff.Added > 0 ||
		c.RightsDiff.Removed > 0

	return ComparisonSummary{
		HasDifferences:     hasDiffs,
		NewlyApplicable:    len(c.NewlyApplicable),
		NoLongerApplicable: len(c.NoLongerApplicable),
		ChangedRelevance:   len(c.ChangedRelevance),
		ObligationsAdded:   c.ObligationsDiff.Added,
		ObligationsRemoved: c.ObligationsDiff.Removed,
		RightsAdded:        c.RightsDiff.Added,
		RightsRemoved:      c.RightsDiff.Removed,
	}
}
