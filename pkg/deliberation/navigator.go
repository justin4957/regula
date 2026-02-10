package deliberation

import (
	"encoding/json"
	"fmt"
	"html"
	"sort"
	"strings"

	"github.com/coolbeans/regula/pkg/store"
)

// NodeType indicates the type of node in the deliberation graph.
type NodeType int

const (
	// NodeMeeting represents a meeting node.
	NodeMeeting NodeType = iota
	// NodeProvision represents a provision/article node.
	NodeProvision
	// NodeStakeholder represents a stakeholder/participant node.
	NodeStakeholder
	// NodeDecision represents a decision node.
	NodeDecision
	// NodeAmendment represents an amendment/motion node.
	NodeAmendment
	// NodeVote represents a vote record node.
	NodeVote
	// NodeAction represents an action item node.
	NodeAction
	// NodeDocument represents a document node.
	NodeDocument
	// NodeAgenda represents an agenda item node.
	NodeAgenda
	// NodeProcess represents a deliberation process node.
	NodeProcess
	// NodeUnknown represents an unknown node type.
	NodeUnknown
)

// String returns a human-readable label for the node type.
func (t NodeType) String() string {
	switch t {
	case NodeMeeting:
		return "Meeting"
	case NodeProvision:
		return "Provision"
	case NodeStakeholder:
		return "Stakeholder"
	case NodeDecision:
		return "Decision"
	case NodeAmendment:
		return "Amendment"
	case NodeVote:
		return "Vote"
	case NodeAction:
		return "Action"
	case NodeDocument:
		return "Document"
	case NodeAgenda:
		return "Agenda"
	case NodeProcess:
		return "Process"
	default:
		return "Unknown"
	}
}

// Symbol returns a short symbol for the node type.
func (t NodeType) Symbol() string {
	switch t {
	case NodeMeeting:
		return "M"
	case NodeProvision:
		return "P"
	case NodeStakeholder:
		return "S"
	case NodeDecision:
		return "D"
	case NodeAmendment:
		return "A"
	case NodeVote:
		return "V"
	case NodeAction:
		return "!"
	case NodeDocument:
		return "◊"
	case NodeAgenda:
		return "#"
	case NodeProcess:
		return "⚙"
	default:
		return "?"
	}
}

// ConnectionDirection indicates the direction of a graph connection.
type ConnectionDirection int

const (
	// DirectionOutgoing indicates the connection goes from focus to target.
	DirectionOutgoing ConnectionDirection = iota
	// DirectionIncoming indicates the connection comes from target to focus.
	DirectionIncoming
	// DirectionBidirectional indicates a two-way connection.
	DirectionBidirectional
)

// String returns a human-readable label for the direction.
func (d ConnectionDirection) String() string {
	switch d {
	case DirectionOutgoing:
		return "outgoing"
	case DirectionIncoming:
		return "incoming"
	case DirectionBidirectional:
		return "bidirectional"
	default:
		return "unknown"
	}
}

// Connection represents a link between nodes in the graph.
type Connection struct {
	// Predicate is the relationship type (e.g., "reg:discusses").
	Predicate string `json:"predicate"`

	// PredicateLabel is a human-readable label for the predicate.
	PredicateLabel string `json:"predicate_label"`

	// TargetURI is the URI of the connected node.
	TargetURI string `json:"target_uri"`

	// TargetLabel is the human-readable label of the target.
	TargetLabel string `json:"target_label"`

	// TargetType is the type of the target node.
	TargetType NodeType `json:"target_type"`

	// Direction indicates whether this is incoming or outgoing.
	Direction ConnectionDirection `json:"direction"`
}

// NavigationNode represents a node in the navigation graph.
type NavigationNode struct {
	// URI is the unique identifier of the node.
	URI string `json:"uri"`

	// Type is the node type.
	Type NodeType `json:"type"`

	// Label is the human-readable label.
	Label string `json:"label"`

	// Properties holds additional node properties.
	Properties map[string]string `json:"properties,omitempty"`

	// Connections lists all connections from this node.
	Connections []Connection `json:"connections,omitempty"`

	// Expandable indicates if this node has unexpanded connections.
	Expandable bool `json:"expandable"`

	// Depth is the distance from the initial focus node.
	Depth int `json:"depth"`
}

// NavigationFilters configures which nodes and edges to show.
type NavigationFilters struct {
	// NodeTypes filters to specific node types (empty = all).
	NodeTypes []NodeType `json:"node_types,omitempty"`

	// Predicates filters to specific predicates (empty = all).
	Predicates []string `json:"predicates,omitempty"`

	// ExcludePredicates lists predicates to exclude.
	ExcludePredicates []string `json:"exclude_predicates,omitempty"`

	// MaxConnections limits connections per node.
	MaxConnections int `json:"max_connections,omitempty"`

	// SearchQuery filters nodes by label.
	SearchQuery string `json:"search_query,omitempty"`
}

// LayoutType indicates the graph layout algorithm.
type LayoutType string

const (
	// LayoutForce uses force-directed layout.
	LayoutForce LayoutType = "force"
	// LayoutHierarchy uses hierarchical layout.
	LayoutHierarchy LayoutType = "hierarchy"
	// LayoutRadial uses radial layout from focus.
	LayoutRadial LayoutType = "radial"
	// LayoutGrid uses grid layout.
	LayoutGrid LayoutType = "grid"
)

// NavigationState tracks the current state of navigation.
type NavigationState struct {
	// FocusNode is the URI of the currently focused node.
	FocusNode string `json:"focus_node"`

	// VisitedNodes tracks navigation history.
	VisitedNodes []string `json:"visited_nodes"`

	// ExpandedNodes tracks which nodes have been expanded.
	ExpandedNodes map[string]bool `json:"expanded_nodes"`

	// Filters configures what to show.
	Filters NavigationFilters `json:"filters"`

	// Layout configures the visual layout.
	Layout LayoutType `json:"layout"`

	// Nodes contains all loaded nodes.
	Nodes map[string]*NavigationNode `json:"nodes"`
}

// GraphNavigator provides interactive exploration of the deliberation graph.
type GraphNavigator struct {
	store   *store.TripleStore
	baseURI string
	state   *NavigationState
}

// NewGraphNavigator creates a new graph navigator.
func NewGraphNavigator(tripleStore *store.TripleStore, baseURI string) *GraphNavigator {
	return &GraphNavigator{
		store:   tripleStore,
		baseURI: baseURI,
		state: &NavigationState{
			VisitedNodes:  make([]string, 0),
			ExpandedNodes: make(map[string]bool),
			Nodes:         make(map[string]*NavigationNode),
			Layout:        LayoutForce,
		},
	}
}

// Focus sets the current focus node and loads its connections.
func (n *GraphNavigator) Focus(uri string) (*NavigationNode, error) {
	node, err := n.LoadNode(uri)
	if err != nil {
		return nil, err
	}

	// Update state
	n.state.FocusNode = uri
	n.state.VisitedNodes = append(n.state.VisitedNodes, uri)
	n.state.Nodes[uri] = node

	return node, nil
}

// LoadNode loads a node and its connections from the triple store.
func (n *GraphNavigator) LoadNode(uri string) (*NavigationNode, error) {
	// Check if already loaded
	if node, ok := n.state.Nodes[uri]; ok {
		return node, nil
	}

	node := &NavigationNode{
		URI:         uri,
		Type:        n.detectNodeType(uri),
		Label:       n.resolveLabel(uri),
		Properties:  make(map[string]string),
		Connections: make([]Connection, 0),
		Expandable:  true,
	}

	// Load properties
	n.loadProperties(node)

	// Load connections
	n.loadConnections(node)

	n.state.Nodes[uri] = node
	return node, nil
}

// Expand loads connections for a node.
func (n *GraphNavigator) Expand(uri string) (*NavigationNode, error) {
	node, err := n.LoadNode(uri)
	if err != nil {
		return nil, err
	}

	n.state.ExpandedNodes[uri] = true

	// Load connected nodes
	for _, conn := range node.Connections {
		if _, ok := n.state.Nodes[conn.TargetURI]; !ok {
			targetNode, _ := n.LoadNode(conn.TargetURI)
			if targetNode != nil {
				targetNode.Depth = node.Depth + 1
			}
		}
	}

	return node, nil
}

// Collapse removes expanded connections for a node.
func (n *GraphNavigator) Collapse(uri string) {
	delete(n.state.ExpandedNodes, uri)
}

// Back navigates to the previous node in history.
func (n *GraphNavigator) Back() (*NavigationNode, error) {
	if len(n.state.VisitedNodes) < 2 {
		return nil, fmt.Errorf("no previous node in history")
	}

	// Remove current from history
	n.state.VisitedNodes = n.state.VisitedNodes[:len(n.state.VisitedNodes)-1]

	// Focus on previous
	prevURI := n.state.VisitedNodes[len(n.state.VisitedNodes)-1]
	n.state.FocusNode = prevURI

	return n.state.Nodes[prevURI], nil
}

// Search finds nodes matching the query.
func (n *GraphNavigator) Search(query string) []NavigationNode {
	results := make([]NavigationNode, 0)
	query = strings.ToLower(query)

	// Search in loaded nodes
	for _, node := range n.state.Nodes {
		if strings.Contains(strings.ToLower(node.Label), query) ||
			strings.Contains(strings.ToLower(node.URI), query) {
			results = append(results, *node)
		}
	}

	// Search in triple store for labels
	labelTriples := n.store.Find("", store.PropTitle, "")
	for _, triple := range labelTriples {
		if strings.Contains(strings.ToLower(triple.Object), query) {
			if _, ok := n.state.Nodes[triple.Subject]; !ok {
				node, _ := n.LoadNode(triple.Subject)
				if node != nil {
					results = append(results, *node)
				}
			}
		}
	}

	// Sort by relevance (exact match first, then prefix, then contains)
	sort.Slice(results, func(i, j int) bool {
		iLabel := strings.ToLower(results[i].Label)
		jLabel := strings.ToLower(results[j].Label)

		iExact := iLabel == query
		jExact := jLabel == query
		if iExact != jExact {
			return iExact
		}

		iPrefix := strings.HasPrefix(iLabel, query)
		jPrefix := strings.HasPrefix(jLabel, query)
		if iPrefix != jPrefix {
			return iPrefix
		}

		return iLabel < jLabel
	})

	return results
}

// FindPath finds the shortest path between two nodes using BFS.
func (n *GraphNavigator) FindPath(fromURI, toURI string, maxDepth int) ([]NavigationNode, error) {
	if maxDepth <= 0 {
		maxDepth = 10
	}

	type pathState struct {
		node string
		path []string
	}

	queue := []pathState{{node: fromURI, path: []string{fromURI}}}
	visited := map[string]bool{fromURI: true}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if len(current.path) > maxDepth {
			break
		}

		if current.node == toURI {
			// Build path of navigation nodes
			result := make([]NavigationNode, 0, len(current.path))
			for i, uri := range current.path {
				node, err := n.LoadNode(uri)
				if err != nil {
					continue
				}
				node.Depth = i
				result = append(result, *node)
			}
			return result, nil
		}

		// Get neighbors
		neighbors := n.getNeighbors(current.node)
		for _, neighbor := range neighbors {
			if !visited[neighbor] {
				visited[neighbor] = true
				newPath := make([]string, len(current.path)+1)
				copy(newPath, current.path)
				newPath[len(current.path)] = neighbor
				queue = append(queue, pathState{node: neighbor, path: newPath})
			}
		}
	}

	return nil, fmt.Errorf("no path found within depth %d", maxDepth)
}

// GetSubgraph returns all nodes within a certain depth from the focus.
func (n *GraphNavigator) GetSubgraph(focusURI string, maxDepth int) ([]NavigationNode, []Connection) {
	nodes := make(map[string]*NavigationNode)
	connections := make([]Connection, 0)

	// BFS to collect nodes
	type queueItem struct {
		uri   string
		depth int
	}
	queue := []queueItem{{uri: focusURI, depth: 0}}
	visited := map[string]bool{focusURI: true}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		node, _ := n.LoadNode(current.uri)
		if node == nil {
			continue
		}
		node.Depth = current.depth
		nodes[current.uri] = node

		if current.depth < maxDepth {
			for _, conn := range node.Connections {
				connections = append(connections, conn)
				if !visited[conn.TargetURI] {
					visited[conn.TargetURI] = true
					queue = append(queue, queueItem{uri: conn.TargetURI, depth: current.depth + 1})
				}
			}
		}
	}

	// Convert map to slice
	nodeList := make([]NavigationNode, 0, len(nodes))
	for _, node := range nodes {
		nodeList = append(nodeList, *node)
	}

	return nodeList, connections
}

// SetFilters updates the navigation filters.
func (n *GraphNavigator) SetFilters(filters NavigationFilters) {
	n.state.Filters = filters
}

// GetState returns the current navigation state.
func (n *GraphNavigator) GetState() *NavigationState {
	return n.state
}

// detectNodeType determines the type of a node from its URI or triples.
func (n *GraphNavigator) detectNodeType(uri string) NodeType {
	// Check RDF type
	typeTriples := n.store.Find(uri, "rdf:type", "")
	for _, triple := range typeTriples {
		typeName := strings.ToLower(triple.Object)
		if strings.Contains(typeName, "meeting") {
			return NodeMeeting
		}
		if strings.Contains(typeName, "article") || strings.Contains(typeName, "provision") ||
			strings.Contains(typeName, "section") || strings.Contains(typeName, "chapter") {
			return NodeProvision
		}
		if strings.Contains(typeName, "stakeholder") || strings.Contains(typeName, "participant") ||
			strings.Contains(typeName, "delegation") {
			return NodeStakeholder
		}
		if strings.Contains(typeName, "decision") {
			return NodeDecision
		}
		if strings.Contains(typeName, "motion") || strings.Contains(typeName, "amendment") {
			return NodeAmendment
		}
		if strings.Contains(typeName, "vote") {
			return NodeVote
		}
		if strings.Contains(typeName, "action") {
			return NodeAction
		}
		if strings.Contains(typeName, "document") {
			return NodeDocument
		}
		if strings.Contains(typeName, "agenda") {
			return NodeAgenda
		}
		if strings.Contains(typeName, "process") {
			return NodeProcess
		}
	}

	// Infer from URI
	uriLower := strings.ToLower(uri)
	if strings.Contains(uriLower, "/meeting/") {
		return NodeMeeting
	}
	if strings.Contains(uriLower, "/article/") || strings.Contains(uriLower, "/provision/") {
		return NodeProvision
	}
	if strings.Contains(uriLower, "/stakeholder/") || strings.Contains(uriLower, "/delegation/") {
		return NodeStakeholder
	}
	if strings.Contains(uriLower, "/decision/") {
		return NodeDecision
	}
	if strings.Contains(uriLower, "/motion/") || strings.Contains(uriLower, "/amendment/") {
		return NodeAmendment
	}
	if strings.Contains(uriLower, "/vote/") {
		return NodeVote
	}
	if strings.Contains(uriLower, "/action/") {
		return NodeAction
	}
	if strings.Contains(uriLower, "/process/") {
		return NodeProcess
	}

	return NodeUnknown
}

// resolveLabel gets a human-readable label for a URI.
func (n *GraphNavigator) resolveLabel(uri string) string {
	// Try title property
	if triples := n.store.Find(uri, store.PropTitle, ""); len(triples) > 0 {
		return triples[0].Object
	}

	// Try rdfs:label
	if triples := n.store.Find(uri, "rdfs:label", ""); len(triples) > 0 {
		return triples[0].Object
	}

	// Try name property
	if triples := n.store.Find(uri, "reg:name", ""); len(triples) > 0 {
		return triples[0].Object
	}

	// Try identifier
	if triples := n.store.Find(uri, "reg:identifier", ""); len(triples) > 0 {
		return triples[0].Object
	}

	// Extract from URI
	return extractURILabel(uri)
}

// loadProperties loads properties for a node.
func (n *GraphNavigator) loadProperties(node *NavigationNode) {
	// Load common properties
	propertyPredicates := []string{
		store.PropTitle,
		store.PropText,
		"reg:identifier",
		"reg:date",
		"reg:status",
		"reg:description",
	}

	for _, pred := range propertyPredicates {
		if triples := n.store.Find(node.URI, pred, ""); len(triples) > 0 {
			// Use short predicate name
			shortName := pred
			if idx := strings.LastIndex(pred, ":"); idx >= 0 {
				shortName = pred[idx+1:]
			}
			node.Properties[shortName] = triples[0].Object
		}
	}
}

// loadConnections loads all connections for a node.
func (n *GraphNavigator) loadConnections(node *NavigationNode) {
	connections := make([]Connection, 0)

	// Outgoing connections
	outgoing := n.store.Find(node.URI, "", "")
	for _, triple := range outgoing {
		// Skip literal values
		if !strings.HasPrefix(triple.Object, "http") && !strings.Contains(triple.Object, ":") {
			continue
		}

		// Skip type triples and properties
		if triple.Predicate == "rdf:type" {
			continue
		}

		conn := Connection{
			Predicate:      triple.Predicate,
			PredicateLabel: n.formatPredicateLabel(triple.Predicate),
			TargetURI:      triple.Object,
			TargetLabel:    n.resolveLabel(triple.Object),
			TargetType:     n.detectNodeType(triple.Object),
			Direction:      DirectionOutgoing,
		}

		if n.matchesFilters(conn) {
			connections = append(connections, conn)
		}
	}

	// Incoming connections
	incoming := n.store.Find("", "", node.URI)
	for _, triple := range incoming {
		conn := Connection{
			Predicate:      triple.Predicate,
			PredicateLabel: n.formatPredicateLabel(triple.Predicate),
			TargetURI:      triple.Subject,
			TargetLabel:    n.resolveLabel(triple.Subject),
			TargetType:     n.detectNodeType(triple.Subject),
			Direction:      DirectionIncoming,
		}

		if n.matchesFilters(conn) {
			connections = append(connections, conn)
		}
	}

	// Apply max connections limit
	if n.state.Filters.MaxConnections > 0 && len(connections) > n.state.Filters.MaxConnections {
		connections = connections[:n.state.Filters.MaxConnections]
		node.Expandable = true
	} else {
		node.Expandable = len(connections) > 0
	}

	node.Connections = connections
}

// matchesFilters checks if a connection matches the current filters.
func (n *GraphNavigator) matchesFilters(conn Connection) bool {
	filters := n.state.Filters

	// Check node type filter
	if len(filters.NodeTypes) > 0 {
		found := false
		for _, t := range filters.NodeTypes {
			if conn.TargetType == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check predicate filter
	if len(filters.Predicates) > 0 {
		found := false
		for _, p := range filters.Predicates {
			if conn.Predicate == p {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check excluded predicates
	for _, p := range filters.ExcludePredicates {
		if conn.Predicate == p {
			return false
		}
	}

	// Check search query
	if filters.SearchQuery != "" {
		query := strings.ToLower(filters.SearchQuery)
		if !strings.Contains(strings.ToLower(conn.TargetLabel), query) &&
			!strings.Contains(strings.ToLower(conn.TargetURI), query) {
			return false
		}
	}

	return true
}

// formatPredicateLabel creates a human-readable label for a predicate.
func (n *GraphNavigator) formatPredicateLabel(predicate string) string {
	// Remove namespace prefix
	label := predicate
	if idx := strings.LastIndex(predicate, ":"); idx >= 0 {
		label = predicate[idx+1:]
	}
	if idx := strings.LastIndex(predicate, "/"); idx >= 0 {
		label = predicate[idx+1:]
	}
	if idx := strings.LastIndex(predicate, "#"); idx >= 0 {
		label = predicate[idx+1:]
	}

	// Convert camelCase to words
	var result strings.Builder
	for i, r := range label {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune(' ')
		}
		result.WriteRune(r)
	}

	return strings.ToLower(result.String())
}

// getNeighbors returns all URIs connected to a node.
func (n *GraphNavigator) getNeighbors(uri string) []string {
	neighbors := make(map[string]bool)

	// Outgoing
	outgoing := n.store.Find(uri, "", "")
	for _, triple := range outgoing {
		if strings.HasPrefix(triple.Object, "http") || strings.Contains(triple.Object, ":") {
			neighbors[triple.Object] = true
		}
	}

	// Incoming
	incoming := n.store.Find("", "", uri)
	for _, triple := range incoming {
		neighbors[triple.Subject] = true
	}

	result := make([]string, 0, len(neighbors))
	for neighbor := range neighbors {
		result = append(result, neighbor)
	}
	return result
}

// RenderTUI renders a terminal UI representation of the current state.
func (n *GraphNavigator) RenderTUI() string {
	var sb strings.Builder

	focusNode := n.state.Nodes[n.state.FocusNode]
	if focusNode == nil {
		sb.WriteString("No node focused. Use Focus() to select a node.\n")
		return sb.String()
	}

	// Header
	sb.WriteString("┌─ Deliberation Graph Navigator ─────────────────────────────┐\n")
	sb.WriteString("│                                                             │\n")

	// Focus node info
	focusLine := fmt.Sprintf("  Focus: [%s] %s", focusNode.Type.Symbol(), focusNode.Label)
	if len(focusLine) > 57 {
		focusLine = focusLine[:54] + "..."
	}
	sb.WriteString(fmt.Sprintf("│%-61s│\n", focusLine))
	sb.WriteString(fmt.Sprintf("│  %s│\n", strings.Repeat("═", 59)))
	sb.WriteString("│                                                             │\n")

	// Split connections by direction
	var incoming, outgoing []Connection
	for _, conn := range focusNode.Connections {
		if conn.Direction == DirectionIncoming {
			incoming = append(incoming, conn)
		} else {
			outgoing = append(outgoing, conn)
		}
	}

	// Headers
	inHeader := fmt.Sprintf("← Incoming (%d)", len(incoming))
	outHeader := fmt.Sprintf("Outgoing → (%d)", len(outgoing))
	sb.WriteString(fmt.Sprintf("│  %-28s %28s  │\n", inHeader, outHeader))
	sb.WriteString(fmt.Sprintf("│  %s %28s  │\n", strings.Repeat("─", 28), strings.Repeat("─", 28)))

	// Connections (side by side)
	maxRows := len(incoming)
	if len(outgoing) > maxRows {
		maxRows = len(outgoing)
	}
	if maxRows > 10 {
		maxRows = 10 // Limit display
	}

	for i := 0; i < maxRows; i++ {
		var inStr, outStr string

		if i < len(incoming) {
			conn := incoming[i]
			inStr = fmt.Sprintf("[%s] %s", conn.TargetType.Symbol(), truncate(conn.TargetLabel, 20))
		}
		if i < len(outgoing) {
			conn := outgoing[i]
			outStr = fmt.Sprintf("[%s] %s", conn.TargetType.Symbol(), truncate(conn.TargetLabel, 20))
		}

		sb.WriteString(fmt.Sprintf("│  %-28s %28s  │\n", inStr, outStr))
	}

	// Show more indicator
	if len(incoming) > 10 || len(outgoing) > 10 {
		moreIn := ""
		moreOut := ""
		if len(incoming) > 10 {
			moreIn = fmt.Sprintf("... +%d more", len(incoming)-10)
		}
		if len(outgoing) > 10 {
			moreOut = fmt.Sprintf("... +%d more", len(outgoing)-10)
		}
		sb.WriteString(fmt.Sprintf("│  %-28s %28s  │\n", moreIn, moreOut))
	}

	sb.WriteString("│                                                             │\n")

	// Help line
	sb.WriteString("│  [↑↓] Navigate  [Enter] Expand  [b] Back  [/] Search       │\n")
	sb.WriteString("│  [f] Filter     [p] Path        [q] Quit                   │\n")
	sb.WriteString("└─────────────────────────────────────────────────────────────┘\n")

	return sb.String()
}

// RenderDOT renders the graph in DOT/GraphViz format.
func (n *GraphNavigator) RenderDOT(focusURI string, maxDepth int) string {
	nodes, connections := n.GetSubgraph(focusURI, maxDepth)

	var sb strings.Builder
	sb.WriteString("digraph DeliberationGraph {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [shape=box, style=rounded];\n\n")

	// Node definitions
	for _, node := range nodes {
		color := n.nodeColor(node.Type)
		label := strings.ReplaceAll(node.Label, "\"", "\\\"")
		sb.WriteString(fmt.Sprintf("  \"%s\" [label=\"[%s] %s\", fillcolor=\"%s\", style=\"filled,rounded\"];\n",
			node.URI, node.Type.Symbol(), label, color))
	}

	sb.WriteString("\n")

	// Edge definitions
	seen := make(map[string]bool)
	for _, conn := range connections {
		edgeKey := conn.TargetURI + "->" + conn.Predicate
		if conn.Direction == DirectionIncoming {
			edgeKey = conn.TargetURI + "<-" + conn.Predicate
		}
		if seen[edgeKey] {
			continue
		}
		seen[edgeKey] = true

		predLabel := conn.PredicateLabel
		if conn.Direction == DirectionIncoming {
			sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [label=\"%s\"];\n",
				conn.TargetURI, n.state.FocusNode, predLabel))
		} else {
			sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [label=\"%s\"];\n",
				n.state.FocusNode, conn.TargetURI, predLabel))
		}
	}

	sb.WriteString("}\n")
	return sb.String()
}

// RenderHTML renders the graph as an interactive HTML page.
func (n *GraphNavigator) RenderHTML(focusURI string, maxDepth int) string {
	nodes, connections := n.GetSubgraph(focusURI, maxDepth)

	// Build JSON data for JavaScript
	nodesJSON, _ := json.Marshal(nodes)
	connectionsJSON, _ := json.Marshal(connections)

	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Deliberation Graph Navigator</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 0; }
    #container { display: flex; height: 100vh; }
    #sidebar { width: 300px; padding: 20px; background: #f5f5f5; overflow-y: auto; }
    #graph { flex: 1; }
    .node-info { margin-bottom: 20px; }
    .node-info h3 { margin: 0 0 10px 0; }
    .node-info .type { display: inline-block; padding: 2px 8px; border-radius: 4px; font-size: 12px; }
    .node-info .properties { margin-top: 10px; }
    .node-info .property { font-size: 13px; margin: 5px 0; }
    .connections { margin-top: 20px; }
    .connection { padding: 8px; margin: 5px 0; background: white; border-radius: 4px; cursor: pointer; }
    .connection:hover { background: #e0e0e0; }
    .connection .symbol { font-weight: bold; margin-right: 5px; }
    svg { width: 100%; height: 100%; }
    .node { cursor: pointer; }
    .node circle { stroke: #333; stroke-width: 2px; }
    .node text { font-size: 12px; }
    .link { stroke: #999; stroke-opacity: 0.6; }
    .link text { font-size: 10px; fill: #666; }
    .Meeting circle { fill: #2196F3; }
    .Provision circle { fill: #4CAF50; }
    .Stakeholder circle { fill: #9C27B0; }
    .Decision circle { fill: #F44336; }
    .Amendment circle { fill: #FF9800; }
    .Vote circle { fill: #00BCD4; }
    .Action circle { fill: #795548; }
    .Document circle { fill: #607D8B; }
  </style>
</head>
<body>
  <div id="container">
    <div id="sidebar">
      <h2>Graph Navigator</h2>
      <div id="node-info" class="node-info">
        <p>Click a node to see details</p>
      </div>
      <div id="connections" class="connections">
        <h4>Connections</h4>
        <div id="connection-list"></div>
      </div>
    </div>
    <div id="graph">
      <svg></svg>
    </div>
  </div>

  <script src="https://d3js.org/d3.v7.min.js"></script>
  <script>
    const nodes = `)
	sb.Write(nodesJSON)
	sb.WriteString(`;
    const connections = `)
	sb.Write(connectionsJSON)
	sb.WriteString(`;

    // Build links from connections
    const nodeMap = new Map(nodes.map(n => [n.uri, n]));
    const links = [];
    const focusURI = "`)
	sb.WriteString(html.EscapeString(focusURI))
	sb.WriteString(`";

    connections.forEach(conn => {
      if (conn.direction === 0) { // outgoing
        links.push({
          source: focusURI,
          target: conn.target_uri,
          predicate: conn.predicate_label
        });
      } else { // incoming
        links.push({
          source: conn.target_uri,
          target: focusURI,
          predicate: conn.predicate_label
        });
      }
    });

    // D3 force simulation
    const svg = d3.select("svg");
    const width = document.getElementById("graph").clientWidth;
    const height = document.getElementById("graph").clientHeight;

    const simulation = d3.forceSimulation(nodes)
      .force("link", d3.forceLink(links).id(d => d.uri).distance(150))
      .force("charge", d3.forceManyBody().strength(-300))
      .force("center", d3.forceCenter(width / 2, height / 2));

    const link = svg.append("g")
      .selectAll("line")
      .data(links)
      .enter().append("line")
      .attr("class", "link");

    const node = svg.append("g")
      .selectAll("g")
      .data(nodes)
      .enter().append("g")
      .attr("class", d => "node " + d.type)
      .call(d3.drag()
        .on("start", dragstarted)
        .on("drag", dragged)
        .on("end", dragended))
      .on("click", showNodeInfo);

    node.append("circle")
      .attr("r", d => d.uri === focusURI ? 15 : 10);

    node.append("text")
      .attr("dx", 15)
      .attr("dy", 4)
      .text(d => d.label.substring(0, 20));

    simulation.on("tick", () => {
      link
        .attr("x1", d => d.source.x)
        .attr("y1", d => d.source.y)
        .attr("x2", d => d.target.x)
        .attr("y2", d => d.target.y);

      node.attr("transform", d => "translate(" + d.x + "," + d.y + ")");
    });

    function dragstarted(event, d) {
      if (!event.active) simulation.alphaTarget(0.3).restart();
      d.fx = d.x;
      d.fy = d.y;
    }

    function dragged(event, d) {
      d.fx = event.x;
      d.fy = event.y;
    }

    function dragended(event, d) {
      if (!event.active) simulation.alphaTarget(0);
      d.fx = null;
      d.fy = null;
    }

    function showNodeInfo(event, d) {
      const typeNames = ["Meeting", "Provision", "Stakeholder", "Decision", "Amendment", "Vote", "Action", "Document", "Agenda", "Process", "Unknown"];
      const typeName = typeNames[d.type] || "Unknown";

      let html = '<h3>' + d.label + '</h3>';
      html += '<span class="type" style="background: ' + getColor(d.type) + '; color: white;">' + typeName + '</span>';
      html += '<div class="properties">';
      if (d.properties) {
        for (const [key, value] of Object.entries(d.properties)) {
          html += '<div class="property"><strong>' + key + ':</strong> ' + value + '</div>';
        }
      }
      html += '</div>';
      document.getElementById('node-info').innerHTML = html;

      // Show connections
      let connHtml = '';
      if (d.connections) {
        d.connections.forEach(conn => {
          const dir = conn.direction === 0 ? '→' : '←';
          connHtml += '<div class="connection" onclick="focusNode(\'' + conn.target_uri + '\')">';
          connHtml += '<span class="symbol">[' + getSymbol(conn.target_type) + ']</span>';
          connHtml += dir + ' ' + conn.target_label + ' (' + conn.predicate_label + ')';
          connHtml += '</div>';
        });
      }
      document.getElementById('connection-list').innerHTML = connHtml;
    }

    function getColor(type) {
      const colors = ["#2196F3", "#4CAF50", "#9C27B0", "#F44336", "#FF9800", "#00BCD4", "#795548", "#607D8B", "#9E9E9E", "#3F51B5", "#9E9E9E"];
      return colors[type] || "#9E9E9E";
    }

    function getSymbol(type) {
      const symbols = ["M", "P", "S", "D", "A", "V", "!", "◊", "#", "⚙", "?"];
      return symbols[type] || "?";
    }

    function focusNode(uri) {
      const node = nodes.find(n => n.uri === uri);
      if (node) {
        showNodeInfo(null, node);
      }
    }
  </script>
</body>
</html>`)

	return sb.String()
}

// RenderJSON renders the graph as JSON.
func (n *GraphNavigator) RenderJSON(focusURI string, maxDepth int) (string, error) {
	nodes, connections := n.GetSubgraph(focusURI, maxDepth)

	data := struct {
		FocusURI    string           `json:"focus_uri"`
		Nodes       []NavigationNode `json:"nodes"`
		Connections []Connection     `json:"connections"`
	}{
		FocusURI:    focusURI,
		Nodes:       nodes,
		Connections: connections,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

// nodeColor returns a color for a node type.
func (n *GraphNavigator) nodeColor(nodeType NodeType) string {
	switch nodeType {
	case NodeMeeting:
		return "#bbdefb"
	case NodeProvision:
		return "#c8e6c9"
	case NodeStakeholder:
		return "#e1bee7"
	case NodeDecision:
		return "#ffcdd2"
	case NodeAmendment:
		return "#ffe0b2"
	case NodeVote:
		return "#b2ebf2"
	case NodeAction:
		return "#d7ccc8"
	case NodeDocument:
		return "#cfd8dc"
	default:
		return "#eeeeee"
	}
}

// truncate truncates a string to a maximum length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
