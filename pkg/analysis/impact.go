// Package analysis provides impact analysis for regulatory provisions.
package analysis

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/coolbeans/regula/pkg/store"
)

// ImpactDirection represents the direction of impact analysis.
type ImpactDirection string

const (
	// DirectionIncoming finds provisions that reference the target.
	DirectionIncoming ImpactDirection = "incoming"
	// DirectionOutgoing finds provisions that the target references.
	DirectionOutgoing ImpactDirection = "outgoing"
	// DirectionBoth finds both incoming and outgoing references.
	DirectionBoth ImpactDirection = "both"
)

// ImpactType categorizes the type of impact.
type ImpactType string

const (
	ImpactDirect     ImpactType = "direct"
	ImpactTransitive ImpactType = "transitive"
)

// ImpactNode represents a node in the impact graph.
type ImpactNode struct {
	URI       string     `json:"uri"`
	Label     string     `json:"label"`
	Type      string     `json:"type"`
	Depth     int        `json:"depth"`
	Impact    ImpactType `json:"impact"`
	Direction string     `json:"direction"`
}

// ImpactEdge represents an edge in the impact graph.
type ImpactEdge struct {
	Source    string `json:"source"`
	Target    string `json:"target"`
	Predicate string `json:"predicate"`
	Depth     int    `json:"depth"`
}

// ImpactResult contains the results of an impact analysis.
type ImpactResult struct {
	TargetURI       string            `json:"target_uri"`
	TargetLabel     string            `json:"target_label"`
	MaxDepth        int               `json:"max_depth"`
	DirectIncoming  []*ImpactNode     `json:"direct_incoming"`
	DirectOutgoing  []*ImpactNode     `json:"direct_outgoing"`
	TransitiveNodes []*ImpactNode     `json:"transitive_nodes"`
	Edges           []*ImpactEdge     `json:"edges"`
	Summary         *ImpactSummary    `json:"summary"`
	ByDepth         map[int][]string  `json:"by_depth"`
}

// ImpactSummary provides summary statistics for the impact analysis.
type ImpactSummary struct {
	TotalAffected       int            `json:"total_affected"`
	DirectIncomingCount int            `json:"direct_incoming_count"`
	DirectOutgoingCount int            `json:"direct_outgoing_count"`
	TransitiveCount     int            `json:"transitive_count"`
	MaxDepthReached     int            `json:"max_depth_reached"`
	AffectedByType      map[string]int `json:"affected_by_type"`
	AffectedByDepth     map[int]int    `json:"affected_by_depth"`
}

// ImpactAnalyzer performs impact analysis on the knowledge graph.
type ImpactAnalyzer struct {
	store   *store.TripleStore
	baseURI string
}

// NewImpactAnalyzer creates a new impact analyzer.
func NewImpactAnalyzer(ts *store.TripleStore, baseURI string) *ImpactAnalyzer {
	return &ImpactAnalyzer{
		store:   ts,
		baseURI: baseURI,
	}
}

// Analyze performs impact analysis for a given provision.
func (a *ImpactAnalyzer) Analyze(provisionURI string, maxDepth int, direction ImpactDirection) *ImpactResult {
	result := &ImpactResult{
		TargetURI:       provisionURI,
		TargetLabel:     a.getLabel(provisionURI),
		MaxDepth:        maxDepth,
		DirectIncoming:  make([]*ImpactNode, 0),
		DirectOutgoing:  make([]*ImpactNode, 0),
		TransitiveNodes: make([]*ImpactNode, 0),
		Edges:           make([]*ImpactEdge, 0),
		ByDepth:         make(map[int][]string),
		Summary: &ImpactSummary{
			AffectedByType:  make(map[string]int),
			AffectedByDepth: make(map[int]int),
		},
	}

	// Track visited nodes to avoid cycles
	visited := make(map[string]bool)
	visited[provisionURI] = true

	// Find direct incoming references (what references the target)
	if direction == DirectionIncoming || direction == DirectionBoth {
		a.findIncomingReferences(provisionURI, result, visited)
	}

	// Find direct outgoing references (what the target references)
	if direction == DirectionOutgoing || direction == DirectionBoth {
		a.findOutgoingReferences(provisionURI, result, visited)
	}

	// Find transitive impact
	if maxDepth > 1 {
		a.findTransitiveImpact(result, maxDepth, direction, visited)
	}

	// Calculate summary
	a.calculateSummary(result)

	return result
}

// AnalyzeByID analyzes impact using a short ID like "Art17" or "GDPR:Art17".
func (a *ImpactAnalyzer) AnalyzeByID(shortID string, maxDepth int, direction ImpactDirection) *ImpactResult {
	uri := a.resolveShortID(shortID)
	return a.Analyze(uri, maxDepth, direction)
}

// resolveShortID converts a short ID to a full URI.
func (a *ImpactAnalyzer) resolveShortID(shortID string) string {
	// If it's already a URI, return as-is
	if strings.HasPrefix(shortID, "http://") || strings.HasPrefix(shortID, "https://") {
		return shortID
	}

	// Handle "GDPR:Art17" format
	if strings.Contains(shortID, ":") {
		parts := strings.SplitN(shortID, ":", 2)
		return a.baseURI + parts[0] + ":" + parts[1]
	}

	// Handle "Art17" format - assume GDPR
	if strings.HasPrefix(shortID, "Art") {
		return a.baseURI + "GDPR:" + shortID
	}

	// Default: prepend base URI
	return a.baseURI + shortID
}

// findIncomingReferences finds provisions that reference the target.
func (a *ImpactAnalyzer) findIncomingReferences(targetURI string, result *ImpactResult, visited map[string]bool) {
	// Find all subjects that have a references or referencedBy predicate pointing to target
	triples := a.store.Find("", store.PropReferences, targetURI)

	for _, t := range triples {
		if visited[t.Subject] {
			continue
		}

		node := &ImpactNode{
			URI:       t.Subject,
			Label:     a.getLabel(t.Subject),
			Type:      a.getType(t.Subject),
			Depth:     1,
			Impact:    ImpactDirect,
			Direction: "incoming",
		}
		result.DirectIncoming = append(result.DirectIncoming, node)

		edge := &ImpactEdge{
			Source:    t.Subject,
			Target:    targetURI,
			Predicate: store.PropReferences,
			Depth:     1,
		}
		result.Edges = append(result.Edges, edge)

		visited[t.Subject] = true
		result.ByDepth[1] = append(result.ByDepth[1], t.Subject)
	}

	// Also check referencedBy in reverse
	triples = a.store.Find(targetURI, store.PropReferencedBy, "")
	for _, t := range triples {
		if visited[t.Object] {
			continue
		}

		node := &ImpactNode{
			URI:       t.Object,
			Label:     a.getLabel(t.Object),
			Type:      a.getType(t.Object),
			Depth:     1,
			Impact:    ImpactDirect,
			Direction: "incoming",
		}
		result.DirectIncoming = append(result.DirectIncoming, node)

		edge := &ImpactEdge{
			Source:    t.Object,
			Target:    targetURI,
			Predicate: store.PropReferences,
			Depth:     1,
		}
		result.Edges = append(result.Edges, edge)

		visited[t.Object] = true
		result.ByDepth[1] = append(result.ByDepth[1], t.Object)
	}
}

// findOutgoingReferences finds provisions that the target references.
func (a *ImpactAnalyzer) findOutgoingReferences(targetURI string, result *ImpactResult, visited map[string]bool) {
	// Find all objects that the target references
	triples := a.store.Find(targetURI, store.PropReferences, "")

	for _, t := range triples {
		if visited[t.Object] {
			continue
		}

		node := &ImpactNode{
			URI:       t.Object,
			Label:     a.getLabel(t.Object),
			Type:      a.getType(t.Object),
			Depth:     1,
			Impact:    ImpactDirect,
			Direction: "outgoing",
		}
		result.DirectOutgoing = append(result.DirectOutgoing, node)

		edge := &ImpactEdge{
			Source:    targetURI,
			Target:    t.Object,
			Predicate: store.PropReferences,
			Depth:     1,
		}
		result.Edges = append(result.Edges, edge)

		visited[t.Object] = true
		result.ByDepth[1] = append(result.ByDepth[1], t.Object)
	}

	// Also check resolvedTarget for resolved references
	triples = a.store.Find(targetURI, store.PropResolvedTarget, "")
	for _, t := range triples {
		if visited[t.Object] {
			continue
		}

		node := &ImpactNode{
			URI:       t.Object,
			Label:     a.getLabel(t.Object),
			Type:      a.getType(t.Object),
			Depth:     1,
			Impact:    ImpactDirect,
			Direction: "outgoing",
		}
		result.DirectOutgoing = append(result.DirectOutgoing, node)

		edge := &ImpactEdge{
			Source:    targetURI,
			Target:    t.Object,
			Predicate: store.PropResolvedTarget,
			Depth:     1,
		}
		result.Edges = append(result.Edges, edge)

		visited[t.Object] = true
		result.ByDepth[1] = append(result.ByDepth[1], t.Object)
	}
}

// findTransitiveImpact finds transitive references up to maxDepth.
func (a *ImpactAnalyzer) findTransitiveImpact(result *ImpactResult, maxDepth int, direction ImpactDirection, visited map[string]bool) {
	// Start from depth 1 nodes and expand
	currentDepthNodes := make([]string, 0)

	// Collect all depth 1 nodes
	for _, node := range result.DirectIncoming {
		currentDepthNodes = append(currentDepthNodes, node.URI)
	}
	for _, node := range result.DirectOutgoing {
		currentDepthNodes = append(currentDepthNodes, node.URI)
	}

	// Expand for each depth level
	for depth := 2; depth <= maxDepth; depth++ {
		nextDepthNodes := make([]string, 0)

		for _, nodeURI := range currentDepthNodes {
			// Find incoming references for this node
			if direction == DirectionIncoming || direction == DirectionBoth {
				triples := a.store.Find("", store.PropReferences, nodeURI)
				for _, t := range triples {
					if visited[t.Subject] {
						continue
					}

					node := &ImpactNode{
						URI:       t.Subject,
						Label:     a.getLabel(t.Subject),
						Type:      a.getType(t.Subject),
						Depth:     depth,
						Impact:    ImpactTransitive,
						Direction: "incoming",
					}
					result.TransitiveNodes = append(result.TransitiveNodes, node)

					edge := &ImpactEdge{
						Source:    t.Subject,
						Target:    nodeURI,
						Predicate: store.PropReferences,
						Depth:     depth,
					}
					result.Edges = append(result.Edges, edge)

					visited[t.Subject] = true
					nextDepthNodes = append(nextDepthNodes, t.Subject)
					result.ByDepth[depth] = append(result.ByDepth[depth], t.Subject)
				}
			}

			// Find outgoing references from this node
			if direction == DirectionOutgoing || direction == DirectionBoth {
				triples := a.store.Find(nodeURI, store.PropReferences, "")
				for _, t := range triples {
					if visited[t.Object] {
						continue
					}

					node := &ImpactNode{
						URI:       t.Object,
						Label:     a.getLabel(t.Object),
						Type:      a.getType(t.Object),
						Depth:     depth,
						Impact:    ImpactTransitive,
						Direction: "outgoing",
					}
					result.TransitiveNodes = append(result.TransitiveNodes, node)

					edge := &ImpactEdge{
						Source:    nodeURI,
						Target:    t.Object,
						Predicate: store.PropReferences,
						Depth:     depth,
					}
					result.Edges = append(result.Edges, edge)

					visited[t.Object] = true
					nextDepthNodes = append(nextDepthNodes, t.Object)
					result.ByDepth[depth] = append(result.ByDepth[depth], t.Object)
				}
			}
		}

		currentDepthNodes = nextDepthNodes
		if len(currentDepthNodes) == 0 {
			break
		}
	}
}

// calculateSummary calculates summary statistics.
func (a *ImpactAnalyzer) calculateSummary(result *ImpactResult) {
	result.Summary.DirectIncomingCount = len(result.DirectIncoming)
	result.Summary.DirectOutgoingCount = len(result.DirectOutgoing)
	result.Summary.TransitiveCount = len(result.TransitiveNodes)
	result.Summary.TotalAffected = result.Summary.DirectIncomingCount +
		result.Summary.DirectOutgoingCount + result.Summary.TransitiveCount

	// Calculate by type
	allNodes := make([]*ImpactNode, 0)
	allNodes = append(allNodes, result.DirectIncoming...)
	allNodes = append(allNodes, result.DirectOutgoing...)
	allNodes = append(allNodes, result.TransitiveNodes...)

	for _, node := range allNodes {
		result.Summary.AffectedByType[node.Type]++
	}

	// Calculate by depth
	maxDepth := 0
	for depth, nodes := range result.ByDepth {
		result.Summary.AffectedByDepth[depth] = len(nodes)
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	result.Summary.MaxDepthReached = maxDepth
}

// getLabel retrieves the label for a URI from the store.
func (a *ImpactAnalyzer) getLabel(uri string) string {
	// Try reg:title first
	triples := a.store.Find(uri, store.PropTitle, "")
	if len(triples) > 0 {
		return triples[0].Object
	}

	// Try rdfs:label
	triples = a.store.Find(uri, store.RDFSLabel, "")
	if len(triples) > 0 {
		return triples[0].Object
	}

	// Try reg:term for definitions
	triples = a.store.Find(uri, store.PropTerm, "")
	if len(triples) > 0 {
		return triples[0].Object
	}

	// Extract from URI
	return extractURILabel(uri)
}

// getType retrieves the type for a URI from the store.
func (a *ImpactAnalyzer) getType(uri string) string {
	triples := a.store.Find(uri, store.RDFType, "")
	if len(triples) > 0 {
		return extractURILabel(triples[0].Object)
	}

	// Infer from URI pattern
	if strings.Contains(uri, ":Art") && !strings.Contains(uri, ":Right:") && !strings.Contains(uri, ":Obligation:") {
		return "Article"
	}
	if strings.Contains(uri, ":Chapter") {
		return "Chapter"
	}
	if strings.Contains(uri, ":Section") {
		return "Section"
	}
	if strings.Contains(uri, ":Recital") {
		return "Recital"
	}
	if strings.Contains(uri, ":Term:") {
		return "DefinedTerm"
	}
	if strings.Contains(uri, ":Right:") {
		return "Right"
	}
	if strings.Contains(uri, ":Obligation:") {
		return "Obligation"
	}

	return "Unknown"
}

// extractURILabel extracts a label from a URI.
func extractURILabel(uri string) string {
	if idx := strings.LastIndex(uri, "#"); idx != -1 {
		return uri[idx+1:]
	}
	if idx := strings.LastIndex(uri, ":"); idx != -1 {
		return uri[idx+1:]
	}
	if idx := strings.LastIndex(uri, "/"); idx != -1 {
		return uri[idx+1:]
	}
	return uri
}

// ToJSON serializes the impact result to JSON.
func (r *ImpactResult) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// String returns a human-readable string representation.
func (r *ImpactResult) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Impact Analysis for: %s\n", r.TargetLabel))
	sb.WriteString(fmt.Sprintf("URI: %s\n", r.TargetURI))
	sb.WriteString(fmt.Sprintf("Analysis Depth: %d\n", r.MaxDepth))
	sb.WriteString("=" + strings.Repeat("=", 50) + "\n\n")

	// Summary
	sb.WriteString("Summary:\n")
	sb.WriteString(fmt.Sprintf("  Total affected provisions: %d\n", r.Summary.TotalAffected))
	sb.WriteString(fmt.Sprintf("  Direct incoming (references this): %d\n", r.Summary.DirectIncomingCount))
	sb.WriteString(fmt.Sprintf("  Direct outgoing (this references): %d\n", r.Summary.DirectOutgoingCount))
	sb.WriteString(fmt.Sprintf("  Transitive: %d\n", r.Summary.TransitiveCount))
	sb.WriteString(fmt.Sprintf("  Max depth reached: %d\n\n", r.Summary.MaxDepthReached))

	// Direct incoming
	if len(r.DirectIncoming) > 0 {
		sb.WriteString("Direct Incoming (provisions referencing this):\n")
		for _, node := range r.DirectIncoming {
			sb.WriteString(fmt.Sprintf("  - %s (%s)\n", node.Label, node.Type))
		}
		sb.WriteString("\n")
	}

	// Direct outgoing
	if len(r.DirectOutgoing) > 0 {
		sb.WriteString("Direct Outgoing (provisions this references):\n")
		for _, node := range r.DirectOutgoing {
			sb.WriteString(fmt.Sprintf("  - %s (%s)\n", node.Label, node.Type))
		}
		sb.WriteString("\n")
	}

	// Transitive by depth
	if len(r.TransitiveNodes) > 0 {
		sb.WriteString("Transitive Impact:\n")

		// Group by depth
		byDepth := make(map[int][]*ImpactNode)
		for _, node := range r.TransitiveNodes {
			byDepth[node.Depth] = append(byDepth[node.Depth], node)
		}

		// Sort depths
		depths := make([]int, 0, len(byDepth))
		for d := range byDepth {
			depths = append(depths, d)
		}
		sort.Ints(depths)

		for _, depth := range depths {
			nodes := byDepth[depth]
			sb.WriteString(fmt.Sprintf("  Depth %d:\n", depth))
			for _, node := range nodes {
				sb.WriteString(fmt.Sprintf("    - %s (%s, %s)\n", node.Label, node.Type, node.Direction))
			}
		}
		sb.WriteString("\n")
	}

	// Affected by type
	if len(r.Summary.AffectedByType) > 0 {
		sb.WriteString("Affected by Type:\n")
		for nodeType, count := range r.Summary.AffectedByType {
			sb.WriteString(fmt.Sprintf("  %s: %d\n", nodeType, count))
		}
	}

	return sb.String()
}

// FormatTable formats the result as a simple table.
func (r *ImpactResult) FormatTable() string {
	var sb strings.Builder

	sb.WriteString("+-------+--------------------------------------------------+------------+-----------+\n")
	sb.WriteString("| Depth | Provision                                        | Type       | Direction |\n")
	sb.WriteString("+-------+--------------------------------------------------+------------+-----------+\n")

	allNodes := make([]*ImpactNode, 0)
	allNodes = append(allNodes, r.DirectIncoming...)
	allNodes = append(allNodes, r.DirectOutgoing...)
	allNodes = append(allNodes, r.TransitiveNodes...)

	// Sort by depth then by label
	sort.Slice(allNodes, func(i, j int) bool {
		if allNodes[i].Depth != allNodes[j].Depth {
			return allNodes[i].Depth < allNodes[j].Depth
		}
		return allNodes[i].Label < allNodes[j].Label
	})

	for _, node := range allNodes {
		label := node.Label
		if len(label) > 48 {
			label = label[:45] + "..."
		}
		sb.WriteString(fmt.Sprintf("| %5d | %-48s | %-10s | %-9s |\n",
			node.Depth, label, node.Type, node.Direction))
	}

	sb.WriteString("+-------+--------------------------------------------------+------------+-----------+\n")

	return sb.String()
}
