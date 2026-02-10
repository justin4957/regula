package deliberation

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/coolbeans/regula/pkg/store"
)

func TestNodeType_String(t *testing.T) {
	tests := []struct {
		nodeType NodeType
		expected string
	}{
		{NodeMeeting, "Meeting"},
		{NodeProvision, "Provision"},
		{NodeStakeholder, "Stakeholder"},
		{NodeDecision, "Decision"},
		{NodeAmendment, "Amendment"},
		{NodeVote, "Vote"},
		{NodeAction, "Action"},
		{NodeDocument, "Document"},
		{NodeAgenda, "Agenda"},
		{NodeProcess, "Process"},
		{NodeUnknown, "Unknown"},
		{NodeType(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.nodeType.String(); got != tt.expected {
			t.Errorf("NodeType(%d).String() = %q, want %q", tt.nodeType, got, tt.expected)
		}
	}
}

func TestNodeType_Symbol(t *testing.T) {
	tests := []struct {
		nodeType NodeType
		expected string
	}{
		{NodeMeeting, "M"},
		{NodeProvision, "P"},
		{NodeStakeholder, "S"},
		{NodeDecision, "D"},
		{NodeAmendment, "A"},
		{NodeVote, "V"},
		{NodeAction, "!"},
		{NodeDocument, "◊"},
		{NodeAgenda, "#"},
		{NodeProcess, "⚙"},
		{NodeUnknown, "?"},
	}

	for _, tt := range tests {
		if got := tt.nodeType.Symbol(); got != tt.expected {
			t.Errorf("NodeType(%d).Symbol() = %q, want %q", tt.nodeType, got, tt.expected)
		}
	}
}

func TestConnectionDirection_String(t *testing.T) {
	tests := []struct {
		direction ConnectionDirection
		expected  string
	}{
		{DirectionOutgoing, "outgoing"},
		{DirectionIncoming, "incoming"},
		{DirectionBidirectional, "bidirectional"},
		{ConnectionDirection(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.direction.String(); got != tt.expected {
			t.Errorf("ConnectionDirection(%d).String() = %q, want %q", tt.direction, got, tt.expected)
		}
	}
}

func buildNavigatorTriples() *store.TripleStore {
	ts := store.NewTripleStore()
	baseURI := "http://example.org/deliberation/"

	// Create meetings
	meeting1URI := baseURI + "meeting/wg-2024-01"
	ts.Add(meeting1URI, "rdf:type", "reg:Meeting")
	ts.Add(meeting1URI, store.PropTitle, "Working Group Meeting 1")
	ts.Add(meeting1URI, "reg:identifier", "WG-2024-01")
	ts.Add(meeting1URI, "reg:date", "2024-03-15")

	meeting2URI := baseURI + "meeting/wg-2024-02"
	ts.Add(meeting2URI, "rdf:type", "reg:Meeting")
	ts.Add(meeting2URI, store.PropTitle, "Working Group Meeting 2")
	ts.Add(meeting2URI, "reg:identifier", "WG-2024-02")
	ts.Add(meeting2URI, "reg:date", "2024-04-15")
	ts.Add(meeting2URI, "reg:follows", meeting1URI)
	ts.Add(meeting1URI, "reg:precedes", meeting2URI)

	// Create provisions
	provision1URI := baseURI + "provision/article-5"
	ts.Add(provision1URI, "rdf:type", "reg:Article")
	ts.Add(provision1URI, store.PropTitle, "Article 5 - Notification Requirements")

	provision2URI := baseURI + "provision/article-6"
	ts.Add(provision2URI, "rdf:type", "reg:Article")
	ts.Add(provision2URI, store.PropTitle, "Article 6 - Processing Conditions")

	// Create stakeholders
	stakeholder1URI := baseURI + "stakeholder/delegation-x"
	ts.Add(stakeholder1URI, "rdf:type", "reg:Stakeholder")
	ts.Add(stakeholder1URI, "reg:name", "Delegation X")

	stakeholder2URI := baseURI + "stakeholder/delegation-y"
	ts.Add(stakeholder2URI, "rdf:type", "reg:Stakeholder")
	ts.Add(stakeholder2URI, "reg:name", "Delegation Y")

	// Create decisions
	decision1URI := baseURI + "decision/dec-2024-01"
	ts.Add(decision1URI, "rdf:type", "reg:Decision")
	ts.Add(decision1URI, store.PropTitle, "Adopted 45-day notification period")
	ts.Add(decision1URI, "reg:affectsProvision", provision1URI)
	ts.Add(decision1URI, "reg:madeAt", meeting2URI)

	// Create motions/amendments
	motion1URI := baseURI + "motion/amendment-1"
	ts.Add(motion1URI, "rdf:type", "reg:Amendment")
	ts.Add(motion1URI, store.PropTitle, "Amendment to Article 5")
	ts.Add(motion1URI, "reg:targetProvision", provision1URI)
	ts.Add(motion1URI, "reg:proposedBy", stakeholder1URI)
	ts.Add(motion1URI, "reg:discussedAt", meeting1URI)

	// Link meetings to provisions discussed
	ts.Add(meeting1URI, "reg:discusses", provision1URI)
	ts.Add(meeting2URI, "reg:discusses", provision1URI)
	ts.Add(meeting2URI, "reg:discusses", provision2URI)

	// Link stakeholders to meetings
	ts.Add(meeting1URI, "reg:attendedBy", stakeholder1URI)
	ts.Add(meeting1URI, "reg:attendedBy", stakeholder2URI)
	ts.Add(meeting2URI, "reg:attendedBy", stakeholder1URI)

	return ts
}

func TestNewGraphNavigator(t *testing.T) {
	ts := store.NewTripleStore()
	nav := NewGraphNavigator(ts, "http://example.org/")

	if nav == nil {
		t.Fatal("NewGraphNavigator returned nil")
	}
	if nav.store != ts {
		t.Error("store not set correctly")
	}
	if nav.baseURI != "http://example.org/" {
		t.Errorf("baseURI = %q, want %q", nav.baseURI, "http://example.org/")
	}
	if nav.state == nil {
		t.Error("state should be initialized")
	}
}

func TestFocus(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	meetingURI := "http://example.org/deliberation/meeting/wg-2024-01"
	node, err := nav.Focus(meetingURI)
	if err != nil {
		t.Fatalf("Focus failed: %v", err)
	}

	if node == nil {
		t.Fatal("node is nil")
	}
	if node.URI != meetingURI {
		t.Errorf("node URI = %q, want %q", node.URI, meetingURI)
	}
	if node.Type != NodeMeeting {
		t.Errorf("node type = %v, want %v", node.Type, NodeMeeting)
	}
	if node.Label != "Working Group Meeting 1" {
		t.Errorf("node label = %q, want %q", node.Label, "Working Group Meeting 1")
	}

	// Check state updated
	if nav.state.FocusNode != meetingURI {
		t.Errorf("state focus = %q, want %q", nav.state.FocusNode, meetingURI)
	}
	if len(nav.state.VisitedNodes) != 1 {
		t.Errorf("visited nodes count = %d, want 1", len(nav.state.VisitedNodes))
	}
}

func TestLoadNode(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	provisionURI := "http://example.org/deliberation/provision/article-5"
	node, err := nav.LoadNode(provisionURI)
	if err != nil {
		t.Fatalf("LoadNode failed: %v", err)
	}

	if node.Type != NodeProvision {
		t.Errorf("node type = %v, want %v", node.Type, NodeProvision)
	}
	if node.Label != "Article 5 - Notification Requirements" {
		t.Errorf("node label = %q, want %q", node.Label, "Article 5 - Notification Requirements")
	}

	// Should have connections
	if len(node.Connections) == 0 {
		t.Error("expected connections for provision node")
	}

	// Check cached
	node2, _ := nav.LoadNode(provisionURI)
	if node != node2 {
		t.Error("LoadNode should return cached node")
	}
}

func TestExpand(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	meetingURI := "http://example.org/deliberation/meeting/wg-2024-01"
	node, err := nav.Expand(meetingURI)
	if err != nil {
		t.Fatalf("Expand failed: %v", err)
	}

	if !nav.state.ExpandedNodes[meetingURI] {
		t.Error("node should be marked as expanded")
	}

	// Connected nodes should be loaded
	for _, conn := range node.Connections {
		if _, ok := nav.state.Nodes[conn.TargetURI]; !ok {
			t.Errorf("connected node %q should be loaded", conn.TargetURI)
		}
	}
}

func TestCollapse(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	meetingURI := "http://example.org/deliberation/meeting/wg-2024-01"
	nav.Expand(meetingURI)

	if !nav.state.ExpandedNodes[meetingURI] {
		t.Error("node should be expanded before collapse")
	}

	nav.Collapse(meetingURI)

	if nav.state.ExpandedNodes[meetingURI] {
		t.Error("node should not be expanded after collapse")
	}
}

func TestBack(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	meeting1URI := "http://example.org/deliberation/meeting/wg-2024-01"
	meeting2URI := "http://example.org/deliberation/meeting/wg-2024-02"

	nav.Focus(meeting1URI)
	nav.Focus(meeting2URI)

	if nav.state.FocusNode != meeting2URI {
		t.Errorf("focus should be meeting2, got %q", nav.state.FocusNode)
	}

	node, err := nav.Back()
	if err != nil {
		t.Fatalf("Back failed: %v", err)
	}

	if nav.state.FocusNode != meeting1URI {
		t.Errorf("focus should be meeting1 after back, got %q", nav.state.FocusNode)
	}
	if node.URI != meeting1URI {
		t.Errorf("returned node should be meeting1")
	}
}

func TestBack_NoHistory(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	_, err := nav.Back()
	if err == nil {
		t.Error("Back should fail with no history")
	}

	nav.Focus("http://example.org/deliberation/meeting/wg-2024-01")
	_, err = nav.Back()
	if err == nil {
		t.Error("Back should fail with only one node in history")
	}
}

func TestSearch(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	// Load some nodes first
	nav.Focus("http://example.org/deliberation/meeting/wg-2024-01")
	nav.Expand("http://example.org/deliberation/meeting/wg-2024-01")

	results := nav.Search("Article")
	if len(results) == 0 {
		t.Error("expected search results for 'Article'")
	}

	found := false
	for _, node := range results {
		if strings.Contains(node.Label, "Article 5") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find Article 5 in search results")
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	nav.Focus("http://example.org/deliberation/meeting/wg-2024-01")
	nav.Expand("http://example.org/deliberation/meeting/wg-2024-01")

	results1 := nav.Search("article")
	results2 := nav.Search("ARTICLE")

	if len(results1) != len(results2) {
		t.Error("search should be case insensitive")
	}
}

func TestFindPath(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	meeting1URI := "http://example.org/deliberation/meeting/wg-2024-01"
	meeting2URI := "http://example.org/deliberation/meeting/wg-2024-02"

	path, err := nav.FindPath(meeting1URI, meeting2URI, 10)
	if err != nil {
		t.Fatalf("FindPath failed: %v", err)
	}

	if len(path) < 2 {
		t.Errorf("path should have at least 2 nodes, got %d", len(path))
	}

	// First node should be meeting1
	if path[0].URI != meeting1URI {
		t.Errorf("path should start at meeting1, got %q", path[0].URI)
	}

	// Last node should be meeting2
	if path[len(path)-1].URI != meeting2URI {
		t.Errorf("path should end at meeting2, got %q", path[len(path)-1].URI)
	}
}

func TestFindPath_NoPath(t *testing.T) {
	ts := store.NewTripleStore()
	// Create two completely disconnected subgraphs
	ts.Add("http://example.org/a", "rdf:type", "reg:NodeA")
	ts.Add("http://example.org/a", "reg:prop", "value-a")
	ts.Add("http://example.org/b", "rdf:type", "reg:NodeB")
	ts.Add("http://example.org/b", "reg:prop", "value-b")
	// No connection between a and b

	nav := NewGraphNavigator(ts, "http://example.org/")

	_, err := nav.FindPath("http://example.org/a", "http://example.org/b", 3)
	if err == nil {
		t.Error("FindPath should fail when no path exists")
	}
}

func TestGetSubgraph(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	meetingURI := "http://example.org/deliberation/meeting/wg-2024-01"
	nodes, connections := nav.GetSubgraph(meetingURI, 2)

	if len(nodes) == 0 {
		t.Error("subgraph should have nodes")
	}

	// Focus node should be included
	foundFocus := false
	for _, node := range nodes {
		if node.URI == meetingURI {
			foundFocus = true
			if node.Depth != 0 {
				t.Errorf("focus node depth should be 0, got %d", node.Depth)
			}
			break
		}
	}
	if !foundFocus {
		t.Error("subgraph should include focus node")
	}

	// Should have some connections
	if len(connections) == 0 {
		t.Error("subgraph should have connections")
	}
}

func TestSetFilters(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	filters := NavigationFilters{
		NodeTypes:      []NodeType{NodeMeeting, NodeDecision},
		MaxConnections: 5,
	}
	nav.SetFilters(filters)

	if len(nav.state.Filters.NodeTypes) != 2 {
		t.Errorf("filters.NodeTypes should have 2 types, got %d", len(nav.state.Filters.NodeTypes))
	}
	if nav.state.Filters.MaxConnections != 5 {
		t.Errorf("filters.MaxConnections = %d, want 5", nav.state.Filters.MaxConnections)
	}
}

func TestFilters_NodeType(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	// Filter to only show Meeting nodes
	nav.SetFilters(NavigationFilters{
		NodeTypes: []NodeType{NodeMeeting},
	})

	meetingURI := "http://example.org/deliberation/meeting/wg-2024-01"
	node, _ := nav.LoadNode(meetingURI)

	for _, conn := range node.Connections {
		if conn.TargetType != NodeMeeting {
			t.Errorf("connection target should be Meeting, got %v", conn.TargetType)
		}
	}
}

func TestFilters_ExcludePredicates(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	nav.SetFilters(NavigationFilters{
		ExcludePredicates: []string{"reg:attendedBy"},
	})

	meetingURI := "http://example.org/deliberation/meeting/wg-2024-01"
	node, _ := nav.LoadNode(meetingURI)

	for _, conn := range node.Connections {
		if conn.Predicate == "reg:attendedBy" {
			t.Error("should not include excluded predicate")
		}
	}
}

func TestDetectNodeType(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	tests := []struct {
		uri      string
		expected NodeType
	}{
		{"http://example.org/deliberation/meeting/wg-2024-01", NodeMeeting},
		{"http://example.org/deliberation/provision/article-5", NodeProvision},
		{"http://example.org/deliberation/stakeholder/delegation-x", NodeStakeholder},
		{"http://example.org/deliberation/decision/dec-2024-01", NodeDecision},
		{"http://example.org/deliberation/motion/amendment-1", NodeAmendment},
	}

	for _, tt := range tests {
		got := nav.detectNodeType(tt.uri)
		if got != tt.expected {
			t.Errorf("detectNodeType(%q) = %v, want %v", tt.uri, got, tt.expected)
		}
	}
}

func TestResolveLabel(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	meetingURI := "http://example.org/deliberation/meeting/wg-2024-01"
	label := nav.resolveLabel(meetingURI)

	if label != "Working Group Meeting 1" {
		t.Errorf("resolveLabel = %q, want %q", label, "Working Group Meeting 1")
	}

	// For URI without title, should extract from URI
	unknownURI := "http://example.org/something/unknown-item"
	label = nav.resolveLabel(unknownURI)
	if label != "unknown-item" {
		t.Errorf("resolveLabel for unknown = %q, want %q", label, "unknown-item")
	}
}

func TestFormatPredicateLabel(t *testing.T) {
	nav := &GraphNavigator{}

	tests := []struct {
		predicate string
		expected  string
	}{
		{"reg:discusses", "discusses"},
		{"rdf:type", "type"},
		{"http://example.org/ontology#camelCase", "camel case"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		got := nav.formatPredicateLabel(tt.predicate)
		if got != tt.expected {
			t.Errorf("formatPredicateLabel(%q) = %q, want %q", tt.predicate, got, tt.expected)
		}
	}
}

func TestRenderTUI(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	// Without focus
	output := nav.RenderTUI()
	if !strings.Contains(output, "No node focused") {
		t.Error("TUI should show no focus message")
	}

	// With focus
	nav.Focus("http://example.org/deliberation/meeting/wg-2024-01")
	output = nav.RenderTUI()

	if !strings.Contains(output, "Graph Navigator") {
		t.Error("TUI should contain title")
	}
	if !strings.Contains(output, "Working Group Meeting 1") {
		t.Error("TUI should show focus node label")
	}
	if !strings.Contains(output, "Incoming") {
		t.Error("TUI should show incoming connections")
	}
	if !strings.Contains(output, "Outgoing") {
		t.Error("TUI should show outgoing connections")
	}
}

func TestRenderDOT(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	meetingURI := "http://example.org/deliberation/meeting/wg-2024-01"
	nav.Focus(meetingURI)

	dot := nav.RenderDOT(meetingURI, 1)

	if !strings.Contains(dot, "digraph") {
		t.Error("DOT should contain 'digraph'")
	}
	if !strings.Contains(dot, "rankdir=LR") {
		t.Error("DOT should contain layout direction")
	}
	if !strings.Contains(dot, "Working Group Meeting 1") {
		t.Error("DOT should contain node label")
	}
	if !strings.Contains(dot, "->") {
		t.Error("DOT should contain edges")
	}
}

func TestNavigatorRenderHTML(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	meetingURI := "http://example.org/deliberation/meeting/wg-2024-01"
	nav.Focus(meetingURI)

	html := nav.RenderHTML(meetingURI, 2)

	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("HTML should have doctype")
	}
	if !strings.Contains(html, "Deliberation Graph Navigator") {
		t.Error("HTML should have title")
	}
	if !strings.Contains(html, "d3js.org") {
		t.Error("HTML should include D3.js")
	}
	if !strings.Contains(html, "forceSimulation") {
		t.Error("HTML should have force simulation code")
	}
}

func TestNavigatorRenderJSON(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	meetingURI := "http://example.org/deliberation/meeting/wg-2024-01"
	nav.Focus(meetingURI)

	jsonStr, err := nav.RenderJSON(meetingURI, 1)
	if err != nil {
		t.Fatalf("RenderJSON failed: %v", err)
	}

	// Verify it's valid JSON
	var data struct {
		FocusURI    string           `json:"focus_uri"`
		Nodes       []NavigationNode `json:"nodes"`
		Connections []Connection     `json:"connections"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		t.Fatalf("RenderJSON produced invalid JSON: %v", err)
	}

	if data.FocusURI != meetingURI {
		t.Errorf("focus_uri = %q, want %q", data.FocusURI, meetingURI)
	}
	if len(data.Nodes) == 0 {
		t.Error("should have nodes in JSON output")
	}
}

func TestConnection(t *testing.T) {
	conn := Connection{
		Predicate:      "reg:discusses",
		PredicateLabel: "discusses",
		TargetURI:      "http://example.org/provision/article-5",
		TargetLabel:    "Article 5",
		TargetType:     NodeProvision,
		Direction:      DirectionOutgoing,
	}

	if conn.Predicate != "reg:discusses" {
		t.Errorf("predicate = %q, want %q", conn.Predicate, "reg:discusses")
	}
	if conn.Direction != DirectionOutgoing {
		t.Errorf("direction = %v, want %v", conn.Direction, DirectionOutgoing)
	}
}

func TestNavigationNode(t *testing.T) {
	node := NavigationNode{
		URI:   "http://example.org/meeting/1",
		Type:  NodeMeeting,
		Label: "Meeting 1",
		Properties: map[string]string{
			"date":   "2024-01-15",
			"status": "completed",
		},
		Connections: []Connection{
			{TargetURI: "http://example.org/provision/1", TargetType: NodeProvision},
		},
		Expandable: true,
		Depth:      0,
	}

	if node.Type != NodeMeeting {
		t.Errorf("type = %v, want %v", node.Type, NodeMeeting)
	}
	if len(node.Properties) != 2 {
		t.Errorf("properties count = %d, want 2", len(node.Properties))
	}
	if len(node.Connections) != 1 {
		t.Errorf("connections count = %d, want 1", len(node.Connections))
	}
}

func TestNavigationState(t *testing.T) {
	state := &NavigationState{
		FocusNode:     "http://example.org/node/1",
		VisitedNodes:  []string{"http://example.org/node/1"},
		ExpandedNodes: map[string]bool{"http://example.org/node/1": true},
		Filters:       NavigationFilters{MaxConnections: 10},
		Layout:        LayoutForce,
		Nodes:         make(map[string]*NavigationNode),
	}

	if state.FocusNode != "http://example.org/node/1" {
		t.Errorf("focus = %q, want %q", state.FocusNode, "http://example.org/node/1")
	}
	if state.Layout != LayoutForce {
		t.Errorf("layout = %v, want %v", state.Layout, LayoutForce)
	}
}

func TestLayoutType(t *testing.T) {
	tests := []struct {
		layout   LayoutType
		expected string
	}{
		{LayoutForce, "force"},
		{LayoutHierarchy, "hierarchy"},
		{LayoutRadial, "radial"},
		{LayoutGrid, "grid"},
	}

	for _, tt := range tests {
		if string(tt.layout) != tt.expected {
			t.Errorf("LayoutType = %q, want %q", string(tt.layout), tt.expected)
		}
	}
}

func TestGetState(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	state := nav.GetState()
	if state == nil {
		t.Fatal("GetState returned nil")
	}
	if state != nav.state {
		t.Error("GetState should return the internal state")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a longer string", 10, "this is..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
		}
	}
}

func TestNodeColor(t *testing.T) {
	nav := &GraphNavigator{}

	// Just verify colors are returned for each type
	types := []NodeType{
		NodeMeeting, NodeProvision, NodeStakeholder, NodeDecision,
		NodeAmendment, NodeVote, NodeAction, NodeDocument, NodeUnknown,
	}

	for _, nodeType := range types {
		color := nav.nodeColor(nodeType)
		if !strings.HasPrefix(color, "#") {
			t.Errorf("nodeColor(%v) = %q, expected hex color", nodeType, color)
		}
	}
}

func TestNavigationFilters(t *testing.T) {
	filters := NavigationFilters{
		NodeTypes:         []NodeType{NodeMeeting, NodeDecision},
		Predicates:        []string{"reg:discusses", "reg:decides"},
		ExcludePredicates: []string{"rdf:type"},
		MaxConnections:    20,
		SearchQuery:       "article",
	}

	if len(filters.NodeTypes) != 2 {
		t.Errorf("NodeTypes count = %d, want 2", len(filters.NodeTypes))
	}
	if len(filters.Predicates) != 2 {
		t.Errorf("Predicates count = %d, want 2", len(filters.Predicates))
	}
	if filters.SearchQuery != "article" {
		t.Errorf("SearchQuery = %q, want %q", filters.SearchQuery, "article")
	}
}

func TestGetNeighbors(t *testing.T) {
	ts := buildNavigatorTriples()
	nav := NewGraphNavigator(ts, "http://example.org/deliberation/")

	meetingURI := "http://example.org/deliberation/meeting/wg-2024-01"
	neighbors := nav.getNeighbors(meetingURI)

	if len(neighbors) == 0 {
		t.Error("meeting should have neighbors")
	}

	// Should include connected provisions, stakeholders, etc.
	hasProvision := false
	hasStakeholder := false
	for _, n := range neighbors {
		if strings.Contains(n, "provision") {
			hasProvision = true
		}
		if strings.Contains(n, "stakeholder") {
			hasStakeholder = true
		}
	}

	if !hasProvision {
		t.Error("neighbors should include provisions")
	}
	if !hasStakeholder {
		t.Error("neighbors should include stakeholders")
	}
}
