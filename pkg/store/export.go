package store

import (
	"encoding/json"
	"fmt"
	"strings"
)

// GraphNode represents a node in the graph visualization.
type GraphNode struct {
	ID       string            `json:"id"`
	Label    string            `json:"label"`
	Type     string            `json:"type"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// GraphEdge represents an edge in the graph visualization.
type GraphEdge struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Label    string `json:"label"`
	Type     string `json:"type"`
}

// GraphExport represents the complete graph for visualization.
type GraphExport struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
	Stats GraphStats  `json:"stats"`
}

// GraphStats contains summary statistics for the graph.
type GraphStats struct {
	TotalNodes   int            `json:"total_nodes"`
	TotalEdges   int            `json:"total_edges"`
	NodesByType  map[string]int `json:"nodes_by_type"`
	EdgesByType  map[string]int `json:"edges_by_type"`
}

// ExportGraph exports the triple store as a graph structure for visualization.
func ExportGraph(store *TripleStore) *GraphExport {
	export := &GraphExport{
		Nodes: make([]GraphNode, 0),
		Edges: make([]GraphEdge, 0),
		Stats: GraphStats{
			NodesByType: make(map[string]int),
			EdgesByType: make(map[string]int),
		},
	}

	// Track unique nodes
	nodeMap := make(map[string]*GraphNode)

	// Build from triples
	for _, t := range store.All() {
		// Create source node if not exists
		if _, exists := nodeMap[t.Subject]; !exists {
			node := createNode(t.Subject, store)
			nodeMap[t.Subject] = &node
			export.Nodes = append(export.Nodes, node)
			export.Stats.NodesByType[node.Type]++
		}

		// Create target node if it looks like a URI (not a literal value)
		if isURI(t.Object) {
			if _, exists := nodeMap[t.Object]; !exists {
				node := createNode(t.Object, store)
				nodeMap[t.Object] = &node
				export.Nodes = append(export.Nodes, node)
				export.Stats.NodesByType[node.Type]++
			}

			// Create edge for relationship predicates
			if isRelationshipPredicate(t.Predicate) {
				edge := GraphEdge{
					Source: t.Subject,
					Target: t.Object,
					Label:  extractLabel(t.Predicate),
					Type:   t.Predicate,
				}
				export.Edges = append(export.Edges, edge)
				export.Stats.EdgesByType[t.Predicate]++
			}
		}
	}

	export.Stats.TotalNodes = len(export.Nodes)
	export.Stats.TotalEdges = len(export.Edges)

	return export
}

// createNode creates a GraphNode from a URI.
func createNode(uri string, store *TripleStore) GraphNode {
	node := GraphNode{
		ID:       uri,
		Label:    extractNodeLabel(uri, store),
		Type:     getNodeType(uri, store),
		Metadata: make(map[string]string),
	}

	// Add metadata from properties
	for _, t := range store.Find(uri, "", "") {
		if !isURI(t.Object) && len(t.Object) < 100 {
			propName := extractLabel(t.Predicate)
			node.Metadata[propName] = t.Object
		}
	}

	return node
}

// extractNodeLabel extracts a readable label for a node.
func extractNodeLabel(uri string, store *TripleStore) string {
	// Try to get rdfs:label
	triples := store.Find(uri, RDFSLabel, "")
	if len(triples) > 0 {
		return triples[0].Object
	}

	// Try to get reg:title
	triples = store.Find(uri, PropTitle, "")
	if len(triples) > 0 && len(triples[0].Object) < 50 {
		return triples[0].Object
	}

	// Try to get reg:term
	triples = store.Find(uri, PropTerm, "")
	if len(triples) > 0 {
		return triples[0].Object
	}

	// Extract from URI
	return extractURILabel(uri)
}

// extractURILabel extracts a label from a URI.
func extractURILabel(uri string) string {
	// Find the last segment
	if idx := strings.LastIndex(uri, "#"); idx != -1 {
		return uri[idx+1:]
	}
	if idx := strings.LastIndex(uri, "/"); idx != -1 {
		return uri[idx+1:]
	}
	if idx := strings.LastIndex(uri, ":"); idx != -1 {
		return uri[idx+1:]
	}
	return uri
}

// getNodeType determines the type of a node.
func getNodeType(uri string, store *TripleStore) string {
	// Check rdf:type
	triples := store.Find(uri, RDFType, "")
	if len(triples) > 0 {
		return extractLabel(triples[0].Object)
	}

	// Infer from URI pattern
	if strings.Contains(uri, ":Art") && !strings.Contains(uri, ":Right:") {
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
	if strings.Contains(uri, ":Ref:") {
		return "Reference"
	}

	return "Node"
}

// extractLabel extracts a readable label from a predicate.
func extractLabel(predicate string) string {
	// Remove namespace prefix
	if idx := strings.LastIndex(predicate, ":"); idx != -1 {
		return predicate[idx+1:]
	}
	if idx := strings.LastIndex(predicate, "#"); idx != -1 {
		return predicate[idx+1:]
	}
	return predicate
}

// isURI checks if a value looks like a URI.
func isURI(value string) bool {
	return strings.HasPrefix(value, "http://") ||
		strings.HasPrefix(value, "https://") ||
		strings.HasPrefix(value, "urn:") ||
		(strings.Contains(value, ":") && !strings.Contains(value, " ") && len(value) < 200)
}

// isRelationshipPredicate checks if a predicate represents a relationship.
func isRelationshipPredicate(predicate string) bool {
	relationshipPredicates := []string{
		PropPartOf,
		PropContains,
		PropBelongsTo,
		PropHasChapter,
		PropHasSection,
		PropHasArticle,
		PropHasParagraph,
		PropHasPoint,
		PropHasRecital,
		PropReferences,
		PropReferencedBy,
		PropRefersToArticle,
		PropRefersToChapter,
		PropRefersToPoint,
		PropDefines,
		PropDefinedIn,
		PropUsesTerm,
		PropGrantsRight,
		PropImposesObligation,
		PropAmends,
		PropAmendedBy,
		PropSupersedes,
		PropSupersededBy,
		PropRepeals,
		PropRepealedBy,
		PropDelegatesTo,
		PropResolvedTarget,
		PropAlternativeTarget,
		PropExternalRef,
		ELIPropIsPartOf,
		ELIPropHasPart,
		ELIPropCites,
		ELIPropCitedBy,
	}

	for _, rp := range relationshipPredicates {
		if predicate == rp {
			return true
		}
	}
	return false
}

// ExportRelationshipSubgraph exports only relationship edges (no metadata).
func ExportRelationshipSubgraph(store *TripleStore) *GraphExport {
	export := &GraphExport{
		Nodes: make([]GraphNode, 0),
		Edges: make([]GraphEdge, 0),
		Stats: GraphStats{
			NodesByType: make(map[string]int),
			EdgesByType: make(map[string]int),
		},
	}

	nodeMap := make(map[string]bool)

	// Only collect relationship edges
	for _, t := range store.All() {
		if !isRelationshipPredicate(t.Predicate) {
			continue
		}
		if !isURI(t.Object) {
			continue
		}

		// Track nodes involved in relationships
		nodeMap[t.Subject] = true
		nodeMap[t.Object] = true

		edge := GraphEdge{
			Source: t.Subject,
			Target: t.Object,
			Label:  extractLabel(t.Predicate),
			Type:   t.Predicate,
		}
		export.Edges = append(export.Edges, edge)
		export.Stats.EdgesByType[t.Predicate]++
	}

	// Build node list
	for uri := range nodeMap {
		node := createNode(uri, store)
		export.Nodes = append(export.Nodes, node)
		export.Stats.NodesByType[node.Type]++
	}

	export.Stats.TotalNodes = len(export.Nodes)
	export.Stats.TotalEdges = len(export.Edges)

	return export
}

// ToJSON serializes the graph export to JSON.
func (g *GraphExport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(g, "", "  ")
}

// ToDOT exports the graph in DOT format for Graphviz.
func (g *GraphExport) ToDOT() string {
	var sb strings.Builder

	sb.WriteString("digraph RegulationGraph {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [shape=box];\n\n")

	// Define node styles by type
	typeColors := map[string]string{
		"Article":     "lightblue",
		"Chapter":     "lightgreen",
		"Section":     "lightyellow",
		"DefinedTerm": "lightpink",
		"Right":       "lightcoral",
		"Obligation":  "lightsalmon",
		"Reference":   "lightgray",
		"Regulation":  "gold",
		"Preamble":    "lavender",
		"Recital":     "lavender",
	}

	// Write nodes
	for _, node := range g.Nodes {
		color := typeColors[node.Type]
		if color == "" {
			color = "white"
		}
		// Escape label for DOT
		label := strings.ReplaceAll(node.Label, "\"", "\\\"")
		if len(label) > 30 {
			label = label[:30] + "..."
		}
		sb.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\" style=filled fillcolor=%s];\n",
			node.ID, label, color))
	}

	sb.WriteString("\n")

	// Write edges
	edgeColors := map[string]string{
		"partOf":           "blue",
		"contains":         "blue",
		"references":       "red",
		"referencedBy":     "red",
		"defines":          "green",
		"definedIn":        "green",
		"usesTerm":         "purple",
		"grantsRight":      "orange",
		"imposesObligation": "brown",
	}

	for _, edge := range g.Edges {
		color := edgeColors[edge.Label]
		if color == "" {
			color = "black"
		}
		sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [label=\"%s\" color=%s];\n",
			edge.Source, edge.Target, edge.Label, color))
	}

	sb.WriteString("}\n")
	return sb.String()
}

// RelationshipSummary provides a summary of relationships in the graph.
type RelationshipSummary struct {
	TotalRelationships   int                    `json:"total_relationships"`
	RelationshipCounts   map[string]int         `json:"relationship_counts"`
	ArticlesWithIncoming map[int]int            `json:"articles_with_incoming"`
	ArticlesWithOutgoing map[int]int            `json:"articles_with_outgoing"`
	MostReferencedArticles []ArticleRefCount    `json:"most_referenced_articles"`
	MostReferencingArticles []ArticleRefCount   `json:"most_referencing_articles"`
}

// ArticleRefCount holds reference count for an article.
type ArticleRefCount struct {
	ArticleNum int `json:"article_num"`
	Count      int `json:"count"`
}

// CalculateRelationshipSummary calculates a summary of relationships.
func CalculateRelationshipSummary(store *TripleStore) *RelationshipSummary {
	summary := &RelationshipSummary{
		RelationshipCounts:     make(map[string]int),
		ArticlesWithIncoming:   make(map[int]int),
		ArticlesWithOutgoing:   make(map[int]int),
	}

	for _, t := range store.All() {
		if !isRelationshipPredicate(t.Predicate) {
			continue
		}

		summary.TotalRelationships++
		summary.RelationshipCounts[t.Predicate]++

		// Track article references
		if t.Predicate == PropReferences {
			sourceNum := extractArticleNum(t.Subject)
			targetNum := extractArticleNum(t.Object)
			if sourceNum > 0 {
				summary.ArticlesWithOutgoing[sourceNum]++
			}
			if targetNum > 0 {
				summary.ArticlesWithIncoming[targetNum]++
			}
		}
	}

	// Calculate most referenced
	for artNum, count := range summary.ArticlesWithIncoming {
		summary.MostReferencedArticles = append(summary.MostReferencedArticles, ArticleRefCount{
			ArticleNum: artNum,
			Count:      count,
		})
	}
	// Sort descending
	for i := 0; i < len(summary.MostReferencedArticles); i++ {
		for j := i + 1; j < len(summary.MostReferencedArticles); j++ {
			if summary.MostReferencedArticles[j].Count > summary.MostReferencedArticles[i].Count {
				summary.MostReferencedArticles[i], summary.MostReferencedArticles[j] =
					summary.MostReferencedArticles[j], summary.MostReferencedArticles[i]
			}
		}
	}
	if len(summary.MostReferencedArticles) > 10 {
		summary.MostReferencedArticles = summary.MostReferencedArticles[:10]
	}

	// Calculate most referencing
	for artNum, count := range summary.ArticlesWithOutgoing {
		summary.MostReferencingArticles = append(summary.MostReferencingArticles, ArticleRefCount{
			ArticleNum: artNum,
			Count:      count,
		})
	}
	for i := 0; i < len(summary.MostReferencingArticles); i++ {
		for j := i + 1; j < len(summary.MostReferencingArticles); j++ {
			if summary.MostReferencingArticles[j].Count > summary.MostReferencingArticles[i].Count {
				summary.MostReferencingArticles[i], summary.MostReferencingArticles[j] =
					summary.MostReferencingArticles[j], summary.MostReferencingArticles[i]
			}
		}
	}
	if len(summary.MostReferencingArticles) > 10 {
		summary.MostReferencingArticles = summary.MostReferencingArticles[:10]
	}

	return summary
}

// extractArticleNum extracts article number from a URI like "...GDPR:Art17"
func extractArticleNum(uri string) int {
	if idx := strings.Index(uri, ":Art"); idx != -1 {
		// Extract number after ":Art"
		rest := uri[idx+4:]
		// Take only digits
		var numStr strings.Builder
		for _, c := range rest {
			if c >= '0' && c <= '9' {
				numStr.WriteRune(c)
			} else {
				break
			}
		}
		if numStr.Len() > 0 {
			return mustAtoi(numStr.String())
		}
	}
	return 0
}

// mustAtoi converts string to int, panicking on error.
func mustAtoi(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
