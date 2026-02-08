// Package extract provides document extraction and parsing utilities.
package extract

import (
	"fmt"
	"sort"
	"strings"
)

// ProceduralStep represents a single step in a legislative procedure.
type ProceduralStep struct {
	// StepNumber is the order of this step (1, 2, 3...).
	StepNumber int

	// Title is a short description of the step.
	Title string

	// Rule is the rule number (Roman numeral).
	Rule string

	// Clause is the clause number.
	Clause string

	// ClauseTitle is the clause title if available.
	ClauseTitle string

	// Description is the explanatory text for this step.
	Description string

	// Excerpt is a relevant excerpt from the rule text.
	Excerpt string

	// References lists other rules/clauses referenced by this step.
	References []string
}

// ProceduralPath represents a complete procedure with multiple steps.
type ProceduralPath struct {
	// Action is the legislative action name (e.g., "introduce a bill").
	Action string

	// Title is the formal title of the procedure.
	Title string

	// Description is an overview of the procedure.
	Description string

	// Steps is the ordered list of procedural steps.
	Steps []ProceduralStep

	// RelatedActions lists other related procedures.
	RelatedActions []string
}

// ProceduralScenario defines a pre-built legislative action with its rule mappings.
type ProceduralScenario struct {
	// Action is the action identifier (e.g., "introduce-bill").
	Action string

	// Title is the display title.
	Title string

	// Description is a brief description of what this procedure covers.
	Description string

	// Keywords are terms used to find relevant clauses.
	Keywords []string

	// RuleSequence is the expected order of rules involved.
	RuleSequence []RuleClauseRef

	// RelatedActions lists related procedure names.
	RelatedActions []string
}

// RuleClauseRef references a specific rule and clause.
type RuleClauseRef struct {
	Rule        string
	Clause      string
	StepTitle   string
	Description string
}

// Pathfinder navigates procedural paths through House Rules.
type Pathfinder struct {
	// searcher is used to find relevant clauses.
	searcher *KeywordSearcher

	// scenarios contains the pre-defined procedural scenarios.
	scenarios map[string]ProceduralScenario
}

// Pre-defined legislative action scenarios.
var defaultScenarios = []ProceduralScenario{
	{
		Action:      "introduce-bill",
		Title:       "Introduce a Bill",
		Description: "The procedure for introducing new legislation in the House.",
		Keywords:    []string{"introduce", "introduction", "bill", "resolution", "hopper"},
		RuleSequence: []RuleClauseRef{
			{Rule: "XII", Clause: "6", StepTitle: "Introduction", Description: "Drop the bill in the hopper or present it from the floor"},
			{Rule: "XII", Clause: "2", StepTitle: "Referral to Committee", Description: "Speaker refers the bill to appropriate committee(s)"},
			{Rule: "X", Clause: "1", StepTitle: "Committee Jurisdiction", Description: "Committee with jurisdiction considers the bill"},
			{Rule: "XI", Clause: "2", StepTitle: "Committee Consideration", Description: "Committee may hold hearings and markup sessions"},
			{Rule: "XIII", Clause: "2", StepTitle: "Committee Report", Description: "Committee reports the bill with recommendations"},
			{Rule: "XIII", Clause: "3", StepTitle: "Calendar Placement", Description: "Reported bill is placed on appropriate calendar"},
		},
		RelatedActions: []string{"amend-bill", "vote-on-bill"},
	},
	{
		Action:      "amend-bill",
		Title:       "Propose an Amendment",
		Description: "The procedure for offering amendments to pending legislation.",
		Keywords:    []string{"amendment", "amend", "substitute", "germane", "modify"},
		RuleSequence: []RuleClauseRef{
			{Rule: "XVI", Clause: "1", StepTitle: "Recognition", Description: "Seek recognition from the Chair to offer an amendment"},
			{Rule: "XVI", Clause: "4", StepTitle: "Reading of Amendment", Description: "Amendment must be in writing and read"},
			{Rule: "XVI", Clause: "7", StepTitle: "Germaneness", Description: "Amendment must be germane to the pending matter"},
			{Rule: "XVIII", Clause: "5", StepTitle: "Reading for Amendment", Description: "Bill is read for amendment under five-minute rule"},
			{Rule: "XIX", Clause: "1", StepTitle: "Motion to Recommit", Description: "Motion to recommit with instructions may include amendments"},
			{Rule: "XX", Clause: "1", StepTitle: "Vote on Amendment", Description: "Amendment is put to a vote"},
		},
		RelatedActions: []string{"introduce-bill", "vote-on-bill", "debate"},
	},
	{
		Action:      "vote-on-bill",
		Title:       "Vote on a Bill",
		Description: "The procedure for conducting a vote on pending legislation.",
		Keywords:    []string{"vote", "voting", "yea", "nay", "roll call", "passage"},
		RuleSequence: []RuleClauseRef{
			{Rule: "XX", Clause: "1", StepTitle: "Putting the Question", Description: "The Speaker puts the question to a vote"},
			{Rule: "I", Clause: "6", StepTitle: "Form of Question", Description: "Speaker states: 'Those in favor say Aye, those opposed say No'"},
			{Rule: "XX", Clause: "2", StepTitle: "Division", Description: "Member may demand a division if voice vote is unclear"},
			{Rule: "XX", Clause: "3", StepTitle: "Recorded Vote", Description: "One-fifth of quorum may demand a recorded vote"},
			{Rule: "XX", Clause: "5", StepTitle: "Electronic Voting", Description: "Recorded vote conducted by electronic device"},
			{Rule: "XX", Clause: "7", StepTitle: "Announcement", Description: "Speaker announces the result of the vote"},
		},
		RelatedActions: []string{"amend-bill", "quorum-call"},
	},
	{
		Action:      "quorum-call",
		Title:       "Establish a Quorum",
		Description: "The procedure for establishing the presence of a quorum.",
		Keywords:    []string{"quorum", "quorum call", "absence", "majority", "present"},
		RuleSequence: []RuleClauseRef{
			{Rule: "XX", Clause: "7", StepTitle: "Point of No Quorum", Description: "Member raises point that a quorum is not present"},
			{Rule: "XX", Clause: "5", StepTitle: "Automatic Roll Call", Description: "Call of the House or automatic roll call is ordered"},
			{Rule: "XX", Clause: "6", StepTitle: "Arrest of Absent Members", Description: "Sergeant at Arms may be directed to secure attendance"},
			{Rule: "XVIII", Clause: "6", StepTitle: "Committee of the Whole Quorum", Description: "100 members constitute a quorum in Committee of the Whole"},
		},
		RelatedActions: []string{"vote-on-bill"},
	},
	{
		Action:      "debate",
		Title:       "Debate a Measure",
		Description: "The procedure for debating pending legislation on the floor.",
		Keywords:    []string{"debate", "recognition", "yield", "time", "five-minute", "one-hour"},
		RuleSequence: []RuleClauseRef{
			{Rule: "XVII", Clause: "1", StepTitle: "Recognition", Description: "Member must be recognized by the Chair to speak"},
			{Rule: "XVII", Clause: "2", StepTitle: "Decorum in Debate", Description: "Rules of decorum must be observed during debate"},
			{Rule: "XVII", Clause: "4", StepTitle: "Call to Order", Description: "Member may be called to order for violation of rules"},
			{Rule: "XIV", Clause: "1", StepTitle: "Order of Business", Description: "Business is taken up in prescribed order"},
			{Rule: "XVI", Clause: "4", StepTitle: "Previous Question", Description: "Motion for previous question can close debate"},
			{Rule: "XVIII", Clause: "5", StepTitle: "Five-Minute Rule", Description: "Debate in Committee of the Whole under five-minute rule"},
		},
		RelatedActions: []string{"amend-bill", "vote-on-bill"},
	},
	{
		Action:      "special-rule",
		Title:       "Adopt a Special Rule",
		Description: "The procedure for adopting a special rule from the Rules Committee.",
		Keywords:    []string{"special rule", "Rules Committee", "special order", "structured rule", "open rule"},
		RuleSequence: []RuleClauseRef{
			{Rule: "X", Clause: "1", StepTitle: "Rules Committee Jurisdiction", Description: "Rules Committee has jurisdiction over special rules"},
			{Rule: "XIII", Clause: "5", StepTitle: "Privileged Reports", Description: "Reports from Rules Committee are privileged"},
			{Rule: "IX", Clause: "1", StepTitle: "Questions of Privilege", Description: "Resolutions from Rules Committee are privileged"},
			{Rule: "XIV", Clause: "1", StepTitle: "Order of Business", Description: "Special rules set the order of business"},
			{Rule: "XVI", Clause: "1", StepTitle: "Motions in Order", Description: "Special rule specifies which amendments are in order"},
		},
		RelatedActions: []string{"debate", "amend-bill"},
	},
	{
		Action:      "suspend-rules",
		Title:       "Suspend the Rules",
		Description: "The procedure for suspending House rules to pass legislation.",
		Keywords:    []string{"suspend", "suspension", "two-thirds", "Monday", "Tuesday"},
		RuleSequence: []RuleClauseRef{
			{Rule: "XV", Clause: "1", StepTitle: "Suspension Days", Description: "Motions to suspend the rules in order on Monday, Tuesday, Wednesday"},
			{Rule: "XV", Clause: "1", StepTitle: "Recognition", Description: "Speaker has discretion to recognize for suspension motions"},
			{Rule: "XV", Clause: "1", StepTitle: "Debate Time", Description: "40 minutes of debate, equally divided"},
			{Rule: "XV", Clause: "1", StepTitle: "Two-Thirds Vote", Description: "Requires two-thirds of those voting to pass"},
			{Rule: "XX", Clause: "1", StepTitle: "Vote", Description: "Vote on the motion to suspend and pass"},
		},
		RelatedActions: []string{"vote-on-bill"},
	},
	{
		Action:      "recommit",
		Title:       "Motion to Recommit",
		Description: "The procedure for the motion to recommit a bill to committee.",
		Keywords:    []string{"recommit", "motion to recommit", "MTR", "instructions", "minority"},
		RuleSequence: []RuleClauseRef{
			{Rule: "XIX", Clause: "2", StepTitle: "Privileged Motion", Description: "Motion to recommit is in order after previous question"},
			{Rule: "XIX", Clause: "2", StepTitle: "Minority Priority", Description: "Priority to member opposed to the bill"},
			{Rule: "XIX", Clause: "2", StepTitle: "With Instructions", Description: "May include instructions to report back forthwith"},
			{Rule: "XVI", Clause: "4", StepTitle: "Debate", Description: "10 minutes of debate on the motion"},
			{Rule: "XX", Clause: "1", StepTitle: "Vote", Description: "Vote on the motion to recommit"},
		},
		RelatedActions: []string{"amend-bill", "vote-on-bill"},
	},
	{
		Action:      "override-veto",
		Title:       "Override a Presidential Veto",
		Description: "The procedure for overriding a presidential veto.",
		Keywords:    []string{"veto", "override", "reconsider", "objections", "two-thirds"},
		RuleSequence: []RuleClauseRef{
			{Rule: "XXII", Clause: "1", StepTitle: "Receipt of Veto Message", Description: "Veto message received from the President"},
			{Rule: "XXII", Clause: "1", StepTitle: "Privileged Matter", Description: "Veto messages are privileged for immediate consideration"},
			{Rule: "XVII", Clause: "1", StepTitle: "Debate", Description: "Debate on the question of reconsideration"},
			{Rule: "XX", Clause: "1", StepTitle: "Two-Thirds Vote", Description: "Two-thirds vote required to override"},
			{Rule: "XXII", Clause: "1", StepTitle: "Transmission to Senate", Description: "If passed, transmitted to Senate for action"},
		},
		RelatedActions: []string{"vote-on-bill"},
	},
	{
		Action:      "conference",
		Title:       "Request a Conference",
		Description: "The procedure for going to conference with the Senate.",
		Keywords:    []string{"conference", "conferees", "managers", "Senate", "disagreement"},
		RuleSequence: []RuleClauseRef{
			{Rule: "XXII", Clause: "1", StepTitle: "Stage of Disagreement", Description: "Houses must be at stage of disagreement"},
			{Rule: "XXII", Clause: "2", StepTitle: "Motion to Request Conference", Description: "Motion to request or agree to a conference"},
			{Rule: "X", Clause: "11", StepTitle: "Appointment of Conferees", Description: "Speaker appoints conferees (managers)"},
			{Rule: "XXII", Clause: "8", StepTitle: "Scope of Conference", Description: "Conferees limited to matters in disagreement"},
			{Rule: "XXII", Clause: "7", StepTitle: "Conference Report", Description: "Conferees file conference report"},
			{Rule: "XXII", Clause: "9", StepTitle: "Vote on Report", Description: "Conference report considered as privileged"},
		},
		RelatedActions: []string{"vote-on-bill"},
	},
}

// NewPathfinder creates a new pathfinder with the given keyword searcher.
func NewPathfinder(searcher *KeywordSearcher) *Pathfinder {
	scenarios := make(map[string]ProceduralScenario)
	for _, s := range defaultScenarios {
		scenarios[s.Action] = s
	}

	return &Pathfinder{
		searcher:  searcher,
		scenarios: scenarios,
	}
}

// GetActions returns a list of available procedural actions.
func (p *Pathfinder) GetActions() []string {
	actions := make([]string, 0, len(p.scenarios))
	for action := range p.scenarios {
		actions = append(actions, action)
	}
	sort.Strings(actions)
	return actions
}

// GetScenario returns the scenario for a given action.
func (p *Pathfinder) GetScenario(action string) (*ProceduralScenario, bool) {
	// Normalize action name
	action = normalizeAction(action)

	scenario, ok := p.scenarios[action]
	if ok {
		return &scenario, true
	}

	// Try partial match
	for name, s := range p.scenarios {
		if strings.Contains(name, action) || strings.Contains(strings.ToLower(s.Title), action) {
			return &s, true
		}
	}

	return nil, false
}

// Navigate generates a procedural path for a given action.
func (p *Pathfinder) Navigate(action string) (*ProceduralPath, error) {
	scenario, ok := p.GetScenario(action)
	if !ok {
		return nil, fmt.Errorf("unknown action: %q (use --list-actions to see available actions)", action)
	}

	path := &ProceduralPath{
		Action:         scenario.Action,
		Title:          scenario.Title,
		Description:    scenario.Description,
		RelatedActions: scenario.RelatedActions,
		Steps:          make([]ProceduralStep, 0, len(scenario.RuleSequence)),
	}

	// Build steps from the rule sequence
	for i, ref := range scenario.RuleSequence {
		step := ProceduralStep{
			StepNumber:  i + 1,
			Title:       ref.StepTitle,
			Rule:        ref.Rule,
			Clause:      ref.Clause,
			Description: ref.Description,
		}

		// Try to find the actual clause text and extract an excerpt
		if p.searcher != nil {
			clause := p.findClause(ref.Rule, ref.Clause)
			if clause != nil {
				step.ClauseTitle = clause.ClauseTitle
				step.Excerpt = extractExcerpt(clause.Text, 200)
				step.References = extractReferences(clause.Text)
			}
		}

		path.Steps = append(path.Steps, step)
	}

	return path, nil
}

// NavigateWithDiscovery generates a procedural path and discovers additional
// relevant clauses based on keywords.
func (p *Pathfinder) NavigateWithDiscovery(action string) (*ProceduralPath, error) {
	// Start with the base navigation
	path, err := p.Navigate(action)
	if err != nil {
		return nil, err
	}

	// Get the scenario for keywords
	scenario, _ := p.GetScenario(action)
	if scenario == nil || p.searcher == nil {
		return path, nil
	}

	// Search for additional relevant clauses
	discoveredRules := make(map[string]bool)
	for _, step := range path.Steps {
		key := step.Rule + ":" + step.Clause
		discoveredRules[key] = true
	}

	// Search using scenario keywords and add discovered steps
	for _, keyword := range scenario.Keywords {
		matches := p.searcher.Search(keyword)
		for _, match := range matches {
			key := match.Rule + ":" + match.Clause
			if !discoveredRules[key] && match.Score > 20 {
				// Add as a discovered step
				step := ProceduralStep{
					StepNumber:  len(path.Steps) + 1,
					Title:       fmt.Sprintf("Related: %s", match.ClauseTitle),
					Rule:        match.Rule,
					Clause:      match.Clause,
					ClauseTitle: match.ClauseTitle,
					Description: fmt.Sprintf("Discovered via keyword '%s'", keyword),
					Excerpt:     match.Context,
					References:  extractReferences(match.Text),
				}
				path.Steps = append(path.Steps, step)
				discoveredRules[key] = true

				// Limit discovered steps
				if len(path.Steps) > 15 {
					break
				}
			}
		}
	}

	return path, nil
}

// findClause finds a specific rule/clause in the searcher's clauses.
func (p *Pathfinder) findClause(rule, clause string) *RuleClause {
	if p.searcher == nil {
		return nil
	}

	for _, c := range p.searcher.GetClauses() {
		if c.Rule == rule && c.Clause == clause {
			return &c
		}
	}
	return nil
}

// String returns a formatted string representation of the procedural path.
func (path *ProceduralPath) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Procedure: %s\n", path.Title))
	sb.WriteString(strings.Repeat("=", len(path.Title)+12) + "\n\n")

	if path.Description != "" {
		sb.WriteString(path.Description + "\n\n")
	}

	for _, step := range path.Steps {
		ruleRef := fmt.Sprintf("Rule %s, clause %s", step.Rule, step.Clause)
		if step.ClauseTitle != "" {
			sb.WriteString(fmt.Sprintf("Step %d: %s (%s: %s)\n", step.StepNumber, step.Title, ruleRef, step.ClauseTitle))
		} else {
			sb.WriteString(fmt.Sprintf("Step %d: %s (%s)\n", step.StepNumber, step.Title, ruleRef))
		}

		if step.Description != "" {
			sb.WriteString(fmt.Sprintf("  %s\n", step.Description))
		}

		if step.Excerpt != "" {
			sb.WriteString(fmt.Sprintf("  \"%s\"\n", step.Excerpt))
		}

		if len(step.References) > 0 {
			sb.WriteString(fmt.Sprintf("  → References: %s\n", strings.Join(step.References, ", ")))
		}

		sb.WriteString("\n")
	}

	if len(path.RelatedActions) > 0 {
		sb.WriteString("Related procedures: " + strings.Join(path.RelatedActions, ", ") + "\n")
	}

	return sb.String()
}

// normalizeAction normalizes an action name for lookup.
func normalizeAction(action string) string {
	action = strings.ToLower(action)
	action = strings.ReplaceAll(action, " ", "-")
	action = strings.ReplaceAll(action, "_", "-")

	// Common aliases
	aliases := map[string]string{
		"introduce":        "introduce-bill",
		"introduce-a-bill": "introduce-bill",
		"bill":             "introduce-bill",
		"amendment":        "amend-bill",
		"amend":            "amend-bill",
		"propose-an-amendment": "amend-bill",
		"propose-amendment": "amend-bill",
		"vote":                 "vote-on-bill",
		"voting":               "vote-on-bill",
		"vote-on-a-bill":       "vote-on-bill",
		"quorum":               "quorum-call",
		"establish-a-quorum":   "quorum-call",
		"establish-quorum":     "quorum-call",
		"suspend":              "suspend-rules",
		"suspension":           "suspend-rules",
		"suspend-the-rules":    "suspend-rules",
		"mtr":                  "recommit",
		"motion-recommit":      "recommit",
		"motion-to-recommit":   "recommit",
		"veto":                 "override-veto",
		"override":             "override-veto",
		"override-a-veto":      "override-veto",
		"conf":                 "conference",
		"request-a-conference": "conference",
		"special":              "special-rule",
		"adopt-a-special-rule": "special-rule",
		"debate-a-measure":     "debate",
	}

	if normalized, ok := aliases[action]; ok {
		return normalized
	}

	return action
}

// extractExcerpt extracts a short excerpt from text.
func extractExcerpt(text string, maxLen int) string {
	text = strings.TrimSpace(text)

	// Find first sentence or meaningful phrase
	endMarkers := []string{". ", ".\n", "; "}
	for _, marker := range endMarkers {
		if idx := strings.Index(text, marker); idx > 0 && idx < maxLen {
			return text[:idx+1]
		}
	}

	if len(text) > maxLen {
		// Find a good break point
		text = text[:maxLen]
		if idx := strings.LastIndex(text, " "); idx > maxLen/2 {
			text = text[:idx]
		}
		return text + "..."
	}

	return text
}

// extractReferences extracts rule/clause references from text.
func extractReferences(text string) []string {
	var refs []string
	seen := make(map[string]bool)

	// Pattern for "rule X" or "clause N of rule X"
	text = strings.ToLower(text)

	// Find "clause N of rule X" patterns
	words := strings.Fields(text)
	for i := 0; i < len(words)-3; i++ {
		if words[i] == "clause" && i+3 < len(words) && words[i+2] == "of" && words[i+3] == "rule" && i+4 < len(words) {
			ref := fmt.Sprintf("Rule %s, clause %s", strings.ToUpper(words[i+4]), words[i+1])
			if !seen[ref] {
				refs = append(refs, ref)
				seen[ref] = true
			}
		}
		// Also catch "rule X, clause N"
		if words[i] == "rule" && i+3 < len(words) {
			ruleNum := strings.TrimSuffix(strings.TrimSuffix(words[i+1], ","), ".")
			if isRomanNumeral(ruleNum) {
				ref := fmt.Sprintf("Rule %s", strings.ToUpper(ruleNum))
				if !seen[ref] {
					refs = append(refs, ref)
					seen[ref] = true
				}
			}
		}
	}

	return refs
}

// isRomanNumeral checks if a string is a valid Roman numeral.
func isRomanNumeral(s string) bool {
	s = strings.ToUpper(s)
	for _, ch := range s {
		if ch != 'I' && ch != 'V' && ch != 'X' && ch != 'L' && ch != 'C' && ch != 'D' && ch != 'M' {
			return false
		}
	}
	return len(s) > 0
}

// ListScenarios returns all available scenarios with descriptions.
func (p *Pathfinder) ListScenarios() []ProceduralScenario {
	scenarios := make([]ProceduralScenario, 0, len(p.scenarios))
	for _, s := range p.scenarios {
		scenarios = append(scenarios, s)
	}
	sort.Slice(scenarios, func(i, j int) bool {
		return scenarios[i].Title < scenarios[j].Title
	})
	return scenarios
}
