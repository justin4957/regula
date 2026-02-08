// Package analysis provides regulation analysis tools.
package analysis

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/coolbeans/regula/pkg/store"
)

// RuleMatrix represents a cross-reference adjacency matrix between rules.
type RuleMatrix struct {
	// Rules is the ordered list of rule identifiers (e.g., ["I", "II", "III", ...]).
	Rules []string

	// Matrix[i][j] is the count of references from Rules[i] to Rules[j].
	Matrix [][]int

	// Incoming[i] is the total incoming references to Rules[i].
	Incoming []int

	// Outgoing[i] is the total outgoing references from Rules[i].
	Outgoing []int

	// TotalRefs is the total number of cross-references.
	TotalRefs int
}

// RuleConnection represents a connected rule with reference counts.
type RuleConnection struct {
	Rule     string `json:"rule"`
	Incoming int    `json:"incoming"`
	Outgoing int    `json:"outgoing"`
	Total    int    `json:"total"`
}

// RuleCluster represents a cluster of mutually referencing rules.
type RuleCluster struct {
	Rules []string `json:"rules"`
	Size  int      `json:"size"`
}

// MatrixReport contains the full cross-reference matrix analysis.
type MatrixReport struct {
	Matrix        *RuleMatrix      `json:"matrix"`
	MostConnected []RuleConnection `json:"most_connected"`
	Clusters      []RuleCluster    `json:"clusters"`
}

// BuildRuleMatrix builds a cross-reference matrix from a triple store.
// It extracts rule-to-rule references based on the reg:references predicate.
func BuildRuleMatrix(tripleStore *store.TripleStore) *RuleMatrix {
	// Build article-to-chapter mapping first
	articleToChapter := make(map[string]string)
	partOfTriples := tripleStore.Find("", store.PropPartOf, "")
	for _, triple := range partOfTriples {
		chapterRule := extractRuleFromURI(triple.Object)
		if chapterRule != "" {
			articleToChapter[triple.Subject] = chapterRule
		}
	}

	// Find all rules in the document
	rulesMap := make(map[string]bool)
	refsBySource := make(map[string]map[string]int) // source rule -> target rule -> count

	// Query for all references
	refs := tripleStore.Find("", store.PropReferences, "")
	for _, triple := range refs {
		// Try to get rule from URI directly
		sourceRule := extractRuleFromURI(triple.Subject)
		targetRule := extractRuleFromURI(triple.Object)

		// If source doesn't have a rule, look it up via partOf
		if sourceRule == "" {
			if chapter, ok := articleToChapter[triple.Subject]; ok {
				sourceRule = chapter
			}
		}

		// If target doesn't have a rule, look it up via partOf
		if targetRule == "" {
			if chapter, ok := articleToChapter[triple.Object]; ok {
				targetRule = chapter
			}
		}

		if sourceRule != "" {
			rulesMap[sourceRule] = true
		}
		if targetRule != "" {
			rulesMap[targetRule] = true
		}

		if sourceRule != "" && targetRule != "" && sourceRule != targetRule {
			if refsBySource[sourceRule] == nil {
				refsBySource[sourceRule] = make(map[string]int)
			}
			refsBySource[sourceRule][targetRule]++
		}
	}

	// Also look for rule/clause references in the identifiers
	allTriples := tripleStore.All()
	for _, triple := range allTriples {
		if triple.Predicate == store.PropReferences {
			sourceRule := extractRuleFromIdentifier(triple.Subject)
			targetRule := extractRuleFromIdentifier(triple.Object)

			// If source doesn't have a rule, look it up via partOf
			if sourceRule == "" {
				if chapter, ok := articleToChapter[triple.Subject]; ok {
					sourceRule = chapter
				}
			}

			// If target doesn't have a rule, look it up via partOf
			if targetRule == "" {
				if chapter, ok := articleToChapter[triple.Object]; ok {
					targetRule = chapter
				}
			}

			if sourceRule != "" {
				rulesMap[sourceRule] = true
			}
			if targetRule != "" {
				rulesMap[targetRule] = true
			}

			if sourceRule != "" && targetRule != "" && sourceRule != targetRule {
				if refsBySource[sourceRule] == nil {
					refsBySource[sourceRule] = make(map[string]int)
				}
				refsBySource[sourceRule][targetRule]++
			}
		}
	}

	// Sort rules by Roman numeral order
	rules := make([]string, 0, len(rulesMap))
	for rule := range rulesMap {
		rules = append(rules, rule)
	}
	sortRomanNumerals(rules)

	// Create index map for quick lookup
	ruleIndex := make(map[string]int)
	for i, rule := range rules {
		ruleIndex[rule] = i
	}

	// Build matrix
	n := len(rules)
	matrix := make([][]int, n)
	for i := range matrix {
		matrix[i] = make([]int, n)
	}

	incoming := make([]int, n)
	outgoing := make([]int, n)
	totalRefs := 0

	for sourceRule, targets := range refsBySource {
		sourceIdx, ok := ruleIndex[sourceRule]
		if !ok {
			continue
		}
		for targetRule, count := range targets {
			targetIdx, ok := ruleIndex[targetRule]
			if !ok {
				continue
			}
			matrix[sourceIdx][targetIdx] = count
			outgoing[sourceIdx] += count
			incoming[targetIdx] += count
			totalRefs += count
		}
	}

	return &RuleMatrix{
		Rules:     rules,
		Matrix:    matrix,
		Incoming:  incoming,
		Outgoing:  outgoing,
		TotalRefs: totalRefs,
	}
}

// extractRuleFromURI extracts a rule number from a URI like "...Rule_XX_clause_5" or "...ChapterXX".
func extractRuleFromURI(uri string) string {
	uri = strings.ToUpper(uri)

	// Try multiple patterns: RULE_XX, RULE-XX, /RULE, CHAPTER, CHAPTERXX
	patterns := []struct {
		prefix string
		skip   int
	}{
		{"RULE_", 5},
		{"RULE-", 5},
		{":CHAPTER", 8},
		{"/CHAPTER", 8},
		{"CHAPTER", 7},
		{"/RULE", 5},
	}

	for _, p := range patterns {
		idx := strings.LastIndex(uri, p.prefix)
		if idx != -1 {
			rest := uri[idx+p.skip:]
			ruleNum := ""
			for _, ch := range rest {
				if ch == 'I' || ch == 'V' || ch == 'X' || ch == 'L' || ch == 'C' || ch == 'D' || ch == 'M' {
					ruleNum += string(ch)
				} else {
					break
				}
			}
			if ruleNum != "" {
				return ruleNum
			}
		}
	}

	return ""
}

// extractRuleFromIdentifier extracts a rule number from an identifier like "Rule XX clause 5".
func extractRuleFromIdentifier(identifier string) string {
	identifier = strings.ToUpper(identifier)

	// Find "RULE " followed by Roman numerals
	idx := strings.Index(identifier, "RULE ")
	if idx == -1 {
		return ""
	}

	rest := identifier[idx+5:]
	ruleNum := ""
	for _, ch := range rest {
		if ch == 'I' || ch == 'V' || ch == 'X' || ch == 'L' || ch == 'C' || ch == 'D' || ch == 'M' {
			ruleNum += string(ch)
		} else if ch == ' ' && ruleNum != "" {
			break
		} else if ch != ' ' && ruleNum == "" {
			// Not a Roman numeral at start
			return ""
		}
	}

	return ruleNum
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

// MostConnected returns the rules with the most connections (incoming + outgoing).
func (m *RuleMatrix) MostConnected(limit int) []RuleConnection {
	connections := make([]RuleConnection, len(m.Rules))
	for i, rule := range m.Rules {
		connections[i] = RuleConnection{
			Rule:     rule,
			Incoming: m.Incoming[i],
			Outgoing: m.Outgoing[i],
			Total:    m.Incoming[i] + m.Outgoing[i],
		}
	}

	sort.Slice(connections, func(i, j int) bool {
		return connections[i].Total > connections[j].Total
	})

	if limit > 0 && limit < len(connections) {
		connections = connections[:limit]
	}

	// Filter out rules with no connections
	result := make([]RuleConnection, 0)
	for _, c := range connections {
		if c.Total > 0 {
			result = append(result, c)
		}
	}

	return result
}

// FindClusters identifies clusters of mutually referencing rules.
// A cluster is a set of rules where each rule references or is referenced by at least one other rule in the cluster.
func (m *RuleMatrix) FindClusters() []RuleCluster {
	n := len(m.Rules)
	if n == 0 {
		return nil
	}

	// Build adjacency list (undirected - both incoming and outgoing count)
	adj := make([][]int, n)
	for i := range adj {
		adj[i] = make([]int, 0)
	}

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if m.Matrix[i][j] > 0 || m.Matrix[j][i] > 0 {
				// Check if j is already in adj[i]
				found := false
				for _, k := range adj[i] {
					if k == j {
						found = true
						break
					}
				}
				if !found && i != j {
					adj[i] = append(adj[i], j)
				}
			}
		}
	}

	// Find connected components using BFS
	visited := make([]bool, n)
	var clusters []RuleCluster

	for start := 0; start < n; start++ {
		if visited[start] {
			continue
		}

		// Check if this node has any connections
		hasConnections := false
		for _, neighbor := range adj[start] {
			if neighbor != start {
				hasConnections = true
				break
			}
		}
		if !hasConnections {
			visited[start] = true
			continue
		}

		// BFS to find all connected nodes
		queue := []int{start}
		visited[start] = true
		var component []string

		for len(queue) > 0 {
			node := queue[0]
			queue = queue[1:]
			component = append(component, m.Rules[node])

			for _, neighbor := range adj[node] {
				if !visited[neighbor] {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}

		if len(component) > 1 {
			sortRomanNumerals(component)
			clusters = append(clusters, RuleCluster{
				Rules: component,
				Size:  len(component),
			})
		}
	}

	// Sort clusters by size (largest first)
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Size > clusters[j].Size
	})

	return clusters
}

// ToASCII generates an ASCII representation of the matrix.
func (m *RuleMatrix) ToASCII() string {
	if len(m.Rules) == 0 {
		return "No cross-references found.\n"
	}

	var sb strings.Builder

	// Calculate column widths
	maxRuleLen := 4 // minimum width for "Rule"
	for _, rule := range m.Rules {
		if len(rule) > maxRuleLen {
			maxRuleLen = len(rule)
		}
	}
	colWidth := maxRuleLen
	if colWidth < 3 {
		colWidth = 3
	}

	// Header row
	sb.WriteString(strings.Repeat(" ", colWidth+1))
	for _, rule := range m.Rules {
		sb.WriteString(fmt.Sprintf("%*s ", colWidth, rule))
	}
	sb.WriteString("\n")

	// Separator
	sb.WriteString(strings.Repeat(" ", colWidth+1))
	for range m.Rules {
		sb.WriteString(strings.Repeat("─", colWidth) + " ")
	}
	sb.WriteString("\n")

	// Data rows
	for i, sourceRule := range m.Rules {
		sb.WriteString(fmt.Sprintf("%*s│", colWidth, sourceRule))
		for j := range m.Rules {
			count := m.Matrix[i][j]
			if i == j {
				sb.WriteString(fmt.Sprintf("%*s ", colWidth, "-"))
			} else if count == 0 {
				sb.WriteString(fmt.Sprintf("%*s ", colWidth, "·"))
			} else {
				sb.WriteString(fmt.Sprintf("%*d ", colWidth, count))
			}
		}
		sb.WriteString(fmt.Sprintf("│ out:%d\n", m.Outgoing[i]))
	}

	// Footer separator
	sb.WriteString(strings.Repeat(" ", colWidth+1))
	for range m.Rules {
		sb.WriteString(strings.Repeat("─", colWidth) + " ")
	}
	sb.WriteString("\n")

	// Incoming totals
	sb.WriteString(fmt.Sprintf("%*s│", colWidth, "in"))
	for j := range m.Rules {
		sb.WriteString(fmt.Sprintf("%*d ", colWidth, m.Incoming[j]))
	}
	sb.WriteString("\n")

	return sb.String()
}

// ToCSV generates a CSV representation of the matrix.
func (m *RuleMatrix) ToCSV() string {
	if len(m.Rules) == 0 {
		return ""
	}

	var sb strings.Builder
	w := csv.NewWriter(&sb)

	// Header row
	header := append([]string{"Source/Target"}, m.Rules...)
	header = append(header, "Outgoing")
	w.Write(header)

	// Data rows
	for i, sourceRule := range m.Rules {
		row := []string{sourceRule}
		for j := range m.Rules {
			if i == j {
				row = append(row, "-")
			} else {
				row = append(row, fmt.Sprintf("%d", m.Matrix[i][j]))
			}
		}
		row = append(row, fmt.Sprintf("%d", m.Outgoing[i]))
		w.Write(row)
	}

	// Incoming totals row
	incoming := []string{"Incoming"}
	for j := range m.Rules {
		incoming = append(incoming, fmt.Sprintf("%d", m.Incoming[j]))
	}
	incoming = append(incoming, fmt.Sprintf("%d", m.TotalRefs))
	w.Write(incoming)

	w.Flush()
	return sb.String()
}

// ToJSON generates a JSON representation of the matrix report.
func (report *MatrixReport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

// GenerateMatrixReport creates a complete matrix analysis report.
func GenerateMatrixReport(tripleStore *store.TripleStore) *MatrixReport {
	matrix := BuildRuleMatrix(tripleStore)

	return &MatrixReport{
		Matrix:        matrix,
		MostConnected: matrix.MostConnected(10),
		Clusters:      matrix.FindClusters(),
	}
}

// ToSVGHeatmap generates an SVG heatmap visualization of the matrix.
func (m *RuleMatrix) ToSVGHeatmap() string {
	if len(m.Rules) == 0 {
		return ""
	}

	n := len(m.Rules)
	cellSize := 30
	labelWidth := 50
	margin := 20
	width := labelWidth + n*cellSize + margin*2
	height := labelWidth + n*cellSize + margin*2

	// Find max count for color scaling
	maxCount := 1
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if m.Matrix[i][j] > maxCount {
				maxCount = m.Matrix[i][j]
			}
		}
	}

	var sb strings.Builder

	// SVG header
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">
`, width, height, width, height))

	// Style
	sb.WriteString(`<style>
  .label { font-family: monospace; font-size: 12px; }
  .cell-text { font-family: monospace; font-size: 10px; fill: white; text-anchor: middle; }
  .title { font-family: sans-serif; font-size: 14px; font-weight: bold; }
</style>
`)

	// Background
	sb.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" fill="white"/>
`, width, height))

	// Column labels (target rules)
	for j, rule := range m.Rules {
		x := labelWidth + j*cellSize + cellSize/2 + margin
		y := margin + 10
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="label" text-anchor="middle">%s</text>
`, x, y, rule))
	}

	// Row labels (source rules) and cells
	for i, sourceRule := range m.Rules {
		// Row label
		x := margin + labelWidth - 5
		y := labelWidth + i*cellSize + cellSize/2 + 5 + margin
		sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="label" text-anchor="end">%s</text>
`, x, y, sourceRule))

		// Cells
		for j := range m.Rules {
			cellX := labelWidth + j*cellSize + margin
			cellY := labelWidth + i*cellSize + margin

			var color string
			count := m.Matrix[i][j]
			if i == j {
				color = "#e0e0e0" // Diagonal
			} else if count == 0 {
				color = "#f8f8f8" // No reference
			} else {
				// Heat color from light blue to dark blue
				intensity := float64(count) / float64(maxCount)
				r := int(240 - intensity*200)
				g := int(240 - intensity*150)
				b := int(255 - intensity*55)
				color = fmt.Sprintf("rgb(%d,%d,%d)", r, g, b)
			}

			sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="%s" stroke="#ccc"/>
`, cellX, cellY, cellSize, cellSize, color))

			// Cell text (count) for non-zero, non-diagonal cells
			if count > 0 && i != j {
				textX := cellX + cellSize/2
				textY := cellY + cellSize/2 + 4
				textColor := "white"
				if intensity := float64(count) / float64(maxCount); intensity < 0.5 {
					textColor = "black"
				}
				sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="cell-text" fill="%s">%d</text>
`, textX, textY, textColor, count))
			}
		}
	}

	// Legend
	legendX := width - margin - 100
	legendY := margin + 10
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" class="title">References</text>
`, legendX, legendY))

	sb.WriteString("</svg>\n")

	return sb.String()
}

// String returns a formatted string representation of the report.
func (report *MatrixReport) String() string {
	var sb strings.Builder

	sb.WriteString("Cross-Reference Matrix\n")
	sb.WriteString(strings.Repeat("═", 60) + "\n\n")

	sb.WriteString(report.Matrix.ToASCII())
	sb.WriteString("\n")

	if len(report.MostConnected) > 0 {
		sb.WriteString("Most Connected Rules:\n")
		for i, conn := range report.MostConnected {
			if i >= 5 {
				break
			}
			sb.WriteString(fmt.Sprintf("  Rule %s: %d outgoing, %d incoming (total: %d)\n",
				conn.Rule, conn.Outgoing, conn.Incoming, conn.Total))
		}
		sb.WriteString("\n")
	}

	if len(report.Clusters) > 0 {
		sb.WriteString("Rule Clusters (mutually referencing):\n")
		for _, cluster := range report.Clusters {
			sb.WriteString(fmt.Sprintf("  {%s}\n", strings.Join(cluster.Rules, ", ")))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("Total cross-references: %d\n", report.Matrix.TotalRefs))

	return sb.String()
}
