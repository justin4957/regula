// Package extract provides document extraction and parsing utilities.
package extract

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ChangeType represents the type of change between two versions.
type ChangeType int

const (
	// ChangeAdded indicates a clause was added in the target version.
	ChangeAdded ChangeType = iota
	// ChangeRemoved indicates a clause was removed in the target version.
	ChangeRemoved
	// ChangeModified indicates a clause was modified between versions.
	ChangeModified
	// ChangeUnchanged indicates no change.
	ChangeUnchanged
)

// String returns the string representation of a ChangeType.
func (c ChangeType) String() string {
	switch c {
	case ChangeAdded:
		return "ADDED"
	case ChangeRemoved:
		return "REMOVED"
	case ChangeModified:
		return "MODIFIED"
	case ChangeUnchanged:
		return "UNCHANGED"
	default:
		return "UNKNOWN"
	}
}

// MarshalJSON implements json.Marshaler for ChangeType.
func (c ChangeType) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

// ClauseChange represents a change to a specific clause.
type ClauseChange struct {
	// Type is the kind of change.
	Type ChangeType `json:"type"`

	// Rule is the rule number (Roman numeral).
	Rule string `json:"rule"`

	// Clause is the clause number.
	Clause string `json:"clause"`

	// ClauseTitle is the clause title (from base or target).
	ClauseTitle string `json:"clause_title,omitempty"`

	// BaseText is the text from the base version (empty for added clauses).
	BaseText string `json:"base_text,omitempty"`

	// TargetText is the text from the target version (empty for removed clauses).
	TargetText string `json:"target_text,omitempty"`

	// Summary describes what changed.
	Summary string `json:"summary,omitempty"`

	// SimilarityScore is the text similarity (0-100) for modified clauses.
	SimilarityScore int `json:"similarity_score,omitempty"`
}

// RuleChange represents changes to an entire rule.
type RuleChange struct {
	// Rule is the rule number (Roman numeral).
	Rule string `json:"rule"`

	// RuleTitle is the rule title.
	RuleTitle string `json:"rule_title,omitempty"`

	// Type indicates the overall change type for this rule.
	Type ChangeType `json:"type"`

	// ClauseChanges contains changes to individual clauses within this rule.
	ClauseChanges []ClauseChange `json:"clause_changes,omitempty"`

	// ClausesAdded is the count of added clauses.
	ClausesAdded int `json:"clauses_added"`

	// ClausesRemoved is the count of removed clauses.
	ClausesRemoved int `json:"clauses_removed"`

	// ClausesModified is the count of modified clauses.
	ClausesModified int `json:"clauses_modified"`

	// ClausesUnchanged is the count of unchanged clauses.
	ClausesUnchanged int `json:"clauses_unchanged"`
}

// RulesDiffReport represents the full diff between two House Rules versions.
type RulesDiffReport struct {
	// BaseVersion is the identifier for the base version (e.g., "118th Congress").
	BaseVersion string `json:"base_version"`

	// TargetVersion is the identifier for the target version (e.g., "119th Congress").
	TargetVersion string `json:"target_version"`

	// RuleChanges contains changes organized by rule.
	RuleChanges []RuleChange `json:"rule_changes"`

	// Summary statistics
	RulesModified    int `json:"rules_modified"`
	RulesAdded       int `json:"rules_added"`
	RulesRemoved     int `json:"rules_removed"`
	TotalClausesAdded    int `json:"total_clauses_added"`
	TotalClausesRemoved  int `json:"total_clauses_removed"`
	TotalClausesModified int `json:"total_clauses_modified"`
}

// RulesDiffer compares two versions of House Rules.
type RulesDiffer struct {
	// baseSearcher contains parsed clauses from the base version.
	baseSearcher *KeywordSearcher

	// targetSearcher contains parsed clauses from the target version.
	targetSearcher *KeywordSearcher
}

// NewRulesDiffer creates a new differ for comparing House Rules.
func NewRulesDiffer(baseText, targetText string) *RulesDiffer {
	baseSearcher := NewKeywordSearcher()
	baseSearcher.ParseHouseRules(baseText)

	targetSearcher := NewKeywordSearcher()
	targetSearcher.ParseHouseRules(targetText)

	return &RulesDiffer{
		baseSearcher:   baseSearcher,
		targetSearcher: targetSearcher,
	}
}

// Compare performs the diff between base and target versions.
func (d *RulesDiffer) Compare(baseVersion, targetVersion string) *RulesDiffReport {
	report := &RulesDiffReport{
		BaseVersion:   baseVersion,
		TargetVersion: targetVersion,
		RuleChanges:   []RuleChange{},
	}

	// Build maps for efficient lookup
	baseClauses := d.buildClauseMap(d.baseSearcher.GetClauses())
	targetClauses := d.buildClauseMap(d.targetSearcher.GetClauses())

	// Collect all rules from both versions
	allRules := make(map[string]bool)
	for key := range baseClauses {
		rule := strings.Split(key, ":")[0]
		allRules[rule] = true
	}
	for key := range targetClauses {
		rule := strings.Split(key, ":")[0]
		allRules[rule] = true
	}

	// Sort rules by Roman numeral order
	rules := make([]string, 0, len(allRules))
	for rule := range allRules {
		rules = append(rules, rule)
	}
	sortRomanNumerals(rules)

	// Compare each rule
	for _, rule := range rules {
		ruleChange := d.compareRule(rule, baseClauses, targetClauses)
		if ruleChange != nil {
			report.RuleChanges = append(report.RuleChanges, *ruleChange)

			// Update summary statistics
			if ruleChange.Type == ChangeAdded {
				report.RulesAdded++
			} else if ruleChange.Type == ChangeRemoved {
				report.RulesRemoved++
			} else if ruleChange.ClausesAdded > 0 || ruleChange.ClausesRemoved > 0 || ruleChange.ClausesModified > 0 {
				report.RulesModified++
			}

			report.TotalClausesAdded += ruleChange.ClausesAdded
			report.TotalClausesRemoved += ruleChange.ClausesRemoved
			report.TotalClausesModified += ruleChange.ClausesModified
		}
	}

	return report
}

// buildClauseMap creates a map of "Rule:Clause" -> RuleClause for efficient lookup.
func (d *RulesDiffer) buildClauseMap(clauses []RuleClause) map[string]RuleClause {
	m := make(map[string]RuleClause)
	for _, clause := range clauses {
		key := clause.Rule + ":" + clause.Clause
		m[key] = clause
	}
	return m
}

// compareRule compares a single rule between base and target versions.
func (d *RulesDiffer) compareRule(rule string, baseClauses, targetClauses map[string]RuleClause) *RuleChange {
	// Collect clauses for this rule from both versions
	baseCls := make(map[string]RuleClause)
	targetCls := make(map[string]RuleClause)

	for key, clause := range baseClauses {
		if strings.HasPrefix(key, rule+":") {
			baseCls[key] = clause
		}
	}
	for key, clause := range targetClauses {
		if strings.HasPrefix(key, rule+":") {
			targetCls[key] = clause
		}
	}

	// If no clauses in either version, skip
	if len(baseCls) == 0 && len(targetCls) == 0 {
		return nil
	}

	ruleChange := &RuleChange{
		Rule:          rule,
		ClauseChanges: []ClauseChange{},
	}

	// Get rule title from either version
	for _, clause := range baseCls {
		if clause.RuleTitle != "" {
			ruleChange.RuleTitle = clause.RuleTitle
			break
		}
	}
	if ruleChange.RuleTitle == "" {
		for _, clause := range targetCls {
			if clause.RuleTitle != "" {
				ruleChange.RuleTitle = clause.RuleTitle
				break
			}
		}
	}

	// Determine overall rule change type
	if len(baseCls) == 0 {
		ruleChange.Type = ChangeAdded
	} else if len(targetCls) == 0 {
		ruleChange.Type = ChangeRemoved
	} else {
		ruleChange.Type = ChangeModified // Will be refined based on clauses
	}

	// Collect all clause numbers
	allClauseNums := make(map[string]bool)
	for key := range baseCls {
		clauseNum := strings.TrimPrefix(key, rule+":")
		allClauseNums[clauseNum] = true
	}
	for key := range targetCls {
		clauseNum := strings.TrimPrefix(key, rule+":")
		allClauseNums[clauseNum] = true
	}

	// Sort clause numbers numerically
	clauseNums := make([]string, 0, len(allClauseNums))
	for num := range allClauseNums {
		clauseNums = append(clauseNums, num)
	}
	sort.Slice(clauseNums, func(i, j int) bool {
		return compareClauseNumbers(clauseNums[i], clauseNums[j])
	})

	// Compare each clause
	for _, clauseNum := range clauseNums {
		key := rule + ":" + clauseNum
		baseClause, inBase := baseCls[key]
		targetClause, inTarget := targetCls[key]

		var change ClauseChange
		change.Rule = rule
		change.Clause = clauseNum

		if !inBase && inTarget {
			// Added
			change.Type = ChangeAdded
			change.TargetText = targetClause.Text
			change.ClauseTitle = targetClause.ClauseTitle
			change.Summary = "New clause added"
			ruleChange.ClausesAdded++
		} else if inBase && !inTarget {
			// Removed
			change.Type = ChangeRemoved
			change.BaseText = baseClause.Text
			change.ClauseTitle = baseClause.ClauseTitle
			change.Summary = "Clause removed"
			ruleChange.ClausesRemoved++
		} else {
			// Both exist - compare text
			similarity := calculateSimilarity(baseClause.Text, targetClause.Text)
			change.SimilarityScore = similarity
			change.ClauseTitle = targetClause.ClauseTitle
			if change.ClauseTitle == "" {
				change.ClauseTitle = baseClause.ClauseTitle
			}

			if similarity >= 95 {
				change.Type = ChangeUnchanged
				ruleChange.ClausesUnchanged++
				continue // Skip unchanged clauses in output
			} else {
				change.Type = ChangeModified
				change.BaseText = baseClause.Text
				change.TargetText = targetClause.Text
				change.Summary = generateChangeSummary(baseClause.Text, targetClause.Text, similarity)
				ruleChange.ClausesModified++
			}
		}

		ruleChange.ClauseChanges = append(ruleChange.ClauseChanges, change)
	}

	// If all clauses unchanged, mark rule as unchanged
	if ruleChange.ClausesAdded == 0 && ruleChange.ClausesRemoved == 0 && ruleChange.ClausesModified == 0 {
		ruleChange.Type = ChangeUnchanged
		return nil // Don't include unchanged rules
	}

	return ruleChange
}

// calculateSimilarity calculates text similarity as a percentage (0-100).
func calculateSimilarity(text1, text2 string) int {
	// Normalize texts
	t1 := normalizeForComparison(text1)
	t2 := normalizeForComparison(text2)

	if t1 == t2 {
		return 100
	}

	// Use word-based Jaccard similarity
	words1 := strings.Fields(t1)
	words2 := strings.Fields(t2)

	set1 := make(map[string]bool)
	for _, w := range words1 {
		set1[w] = true
	}

	set2 := make(map[string]bool)
	for _, w := range words2 {
		set2[w] = true
	}

	// Calculate intersection and union
	intersection := 0
	for w := range set1 {
		if set2[w] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 100
	}

	return (intersection * 100) / union
}

// normalizeForComparison normalizes text for comparison.
func normalizeForComparison(text string) string {
	text = strings.ToLower(text)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\t", " ")
	// Collapse multiple spaces
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}
	return strings.TrimSpace(text)
}

// generateChangeSummary creates a human-readable summary of changes.
func generateChangeSummary(baseText, targetText string, similarity int) string {
	baseLen := len(strings.Fields(baseText))
	targetLen := len(strings.Fields(targetText))

	if similarity >= 80 {
		return "Minor text updates"
	} else if similarity >= 60 {
		return "Moderate revisions"
	} else if similarity >= 40 {
		return "Substantial changes"
	} else if targetLen > baseLen*2 {
		return "Significantly expanded"
	} else if baseLen > targetLen*2 {
		return "Significantly condensed"
	} else {
		return "Major rewrite"
	}
}

// compareClauseNumbers compares clause numbers for sorting.
func compareClauseNumbers(a, b string) bool {
	// Try numeric comparison first
	var aNum, bNum int
	if _, err := fmt.Sscanf(a, "%d", &aNum); err == nil {
		if _, err := fmt.Sscanf(b, "%d", &bNum); err == nil {
			return aNum < bNum
		}
	}
	// Fall back to string comparison
	return a < b
}

// sortRomanNumerals sorts Roman numerals in numeric order.
func sortRomanNumerals(numerals []string) {
	sort.Slice(numerals, func(i, j int) bool {
		return romanToInt(numerals[i]) < romanToInt(numerals[j])
	})
}

// romanToInt converts a Roman numeral to an integer.
func romanToInt(s string) int {
	values := map[rune]int{
		'I': 1, 'V': 5, 'X': 10, 'L': 50,
		'C': 100, 'D': 500, 'M': 1000,
	}

	result := 0
	prev := 0
	for i := len(s) - 1; i >= 0; i-- {
		val := values[rune(s[i])]
		if val < prev {
			result -= val
		} else {
			result += val
		}
		prev = val
	}
	return result
}

// String returns a formatted string representation of the diff report.
func (r *RulesDiffReport) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("House Rules Diff: %s → %s\n", r.BaseVersion, r.TargetVersion))
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")

	// Summary statistics
	sb.WriteString("Summary:\n")
	sb.WriteString(fmt.Sprintf("  Rules modified: %d\n", r.RulesModified))
	if r.RulesAdded > 0 {
		sb.WriteString(fmt.Sprintf("  Rules added: %d\n", r.RulesAdded))
	}
	if r.RulesRemoved > 0 {
		sb.WriteString(fmt.Sprintf("  Rules removed: %d\n", r.RulesRemoved))
	}
	sb.WriteString(fmt.Sprintf("  Clauses added: %d\n", r.TotalClausesAdded))
	sb.WriteString(fmt.Sprintf("  Clauses removed: %d\n", r.TotalClausesRemoved))
	sb.WriteString(fmt.Sprintf("  Clauses modified: %d\n", r.TotalClausesModified))
	sb.WriteString("\n")

	// Details by rule
	if len(r.RuleChanges) > 0 {
		sb.WriteString("Changes:\n")
		for _, ruleChange := range r.RuleChanges {
			if len(ruleChange.ClauseChanges) == 0 {
				continue
			}

			ruleName := "Rule " + ruleChange.Rule
			if ruleChange.RuleTitle != "" {
				ruleName += " (" + ruleChange.RuleTitle + ")"
			}
			sb.WriteString(fmt.Sprintf("\n%s:\n", ruleName))

			for _, change := range ruleChange.ClauseChanges {
				clauseRef := fmt.Sprintf("  clause %s", change.Clause)
				if change.ClauseTitle != "" {
					clauseRef += fmt.Sprintf(" (%s)", truncate(change.ClauseTitle, 40))
				}
				sb.WriteString(fmt.Sprintf("%s: %s", clauseRef, change.Type))
				if change.Summary != "" && change.Type == ChangeModified {
					sb.WriteString(fmt.Sprintf(" - %s", change.Summary))
					if change.SimilarityScore > 0 {
						sb.WriteString(fmt.Sprintf(" (%d%% similar)", change.SimilarityScore))
					}
				}
				sb.WriteString("\n")
			}
		}
	}

	return sb.String()
}

// truncate shortens text to the specified length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// ToJSON returns the report as JSON.
func (r *RulesDiffReport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// GetSignificantChanges returns only changes with similarity below threshold.
func (r *RulesDiffReport) GetSignificantChanges(maxSimilarity int) []ClauseChange {
	var significant []ClauseChange
	for _, ruleChange := range r.RuleChanges {
		for _, change := range ruleChange.ClauseChanges {
			if change.Type == ChangeAdded || change.Type == ChangeRemoved ||
				(change.Type == ChangeModified && change.SimilarityScore <= maxSimilarity) {
				significant = append(significant, change)
			}
		}
	}
	return significant
}
