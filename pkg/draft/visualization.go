package draft

import (
	"fmt"
	"os"
	"strings"
)

// RenderImpactGraph generates a Graphviz DOT string representing the impact
// radius of proposed legislation. Nodes are color-coded by impact category:
//   - Red: directly modified or repealed provisions (from the diff)
//   - Orange: directly affected provisions (depth 1 references)
//   - Yellow: transitively affected provisions (depth 2+)
//   - Green: context nodes (referenced but not affected)
//
// Edges are styled to distinguish reference types:
//   - Solid: reg:references relationships
//   - Dashed red: broken cross-references (pointing to repealed sections)
//   - Bold: reference chains that propagate impact
//
// Nodes are clustered by USC title, with proposed amendments in a separate
// cluster. A legend subgraph explains the color coding.
func RenderImpactGraph(result *DraftImpactResult) (string, error) {
	if result == nil {
		return "", fmt.Errorf("impact result is nil")
	}

	var sb strings.Builder

	billLabel := ""
	if result.Bill != nil {
		billLabel = result.Bill.BillNumber
		if result.Bill.ShortTitle != "" {
			billLabel += " — " + result.Bill.ShortTitle
		} else if result.Bill.Title != "" {
			billLabel += " — " + result.Bill.Title
		}
	}

	sb.WriteString("digraph DraftImpactAnalysis {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  compound=true;\n")
	sb.WriteString("  fontname=\"Helvetica\";\n")
	sb.WriteString("  node [fontname=\"Helvetica\" fontsize=10 shape=box];\n")
	sb.WriteString("  edge [fontname=\"Helvetica\" fontsize=8];\n")
	sb.WriteString(fmt.Sprintf("  label=\"Impact Analysis: %s\";\n", escapeDOTString(billLabel)))
	sb.WriteString("  labelloc=t;\n\n")

	// Collect all nodes and organize by document/title cluster
	nodesByCluster := make(map[string][]dotNode)
	emittedNodes := make(map[string]bool)

	// Modified/repealed provisions (red) — from the diff
	if result.Diff != nil {
		addDiffEntriesToClusters(result.Diff.Modified, "modified", nodesByCluster, emittedNodes)
		addDiffEntriesToClusters(result.Diff.Removed, "repealed", nodesByCluster, emittedNodes)
		addDiffEntriesToClusters(result.Diff.Added, "added", nodesByCluster, emittedNodes)
		addDiffEntriesToClusters(result.Diff.Redesignated, "redesignated", nodesByCluster, emittedNodes)
	}

	// Directly affected provisions (orange)
	for _, provision := range result.DirectlyAffected {
		if emittedNodes[provision.URI] {
			continue
		}
		emittedNodes[provision.URI] = true
		clusterKey := clusterKeyFromDocumentID(provision.DocumentID)
		nodesByCluster[clusterKey] = append(nodesByCluster[clusterKey], dotNode{
			id:       provision.URI,
			label:    nodeLabel(provision.Label, provision.URI),
			fillColor: "orange",
			category: "directly affected",
		})
	}

	// Transitively affected provisions (yellow)
	for _, provision := range result.TransitivelyAffected {
		if emittedNodes[provision.URI] {
			continue
		}
		emittedNodes[provision.URI] = true
		clusterKey := clusterKeyFromDocumentID(provision.DocumentID)
		nodesByCluster[clusterKey] = append(nodesByCluster[clusterKey], dotNode{
			id:        provision.URI,
			label:     nodeLabel(provision.Label, provision.URI),
			fillColor: "yellow",
			category:  "transitively affected",
		})
	}

	// Ensure broken cross-ref endpoints are declared as nodes
	for _, brokenRef := range result.BrokenCrossRefs {
		if !emittedNodes[brokenRef.SourceURI] {
			emittedNodes[brokenRef.SourceURI] = true
			clusterKey := clusterKeyFromDocumentID(brokenRef.SourceDocumentID)
			nodesByCluster[clusterKey] = append(nodesByCluster[clusterKey], dotNode{
				id:        brokenRef.SourceURI,
				label:     nodeLabel(brokenRef.SourceLabel, brokenRef.SourceURI),
				fillColor: "lightsalmon",
				category:  "broken ref source",
			})
		}
	}

	// Write title clusters
	clusterIndex := 0
	for clusterKey, nodes := range nodesByCluster {
		if clusterKey == "_proposed" {
			continue // write proposed changes cluster separately
		}
		sb.WriteString(fmt.Sprintf("  subgraph cluster_%d {\n", clusterIndex))
		sb.WriteString(fmt.Sprintf("    label=\"%s\";\n", escapeDOTString(clusterKey)))
		sb.WriteString("    style=filled;\n")
		sb.WriteString("    color=lightgrey;\n")
		sb.WriteString("    node [style=filled];\n\n")

		for _, node := range nodes {
			writeNode(&sb, node, "    ")
		}

		sb.WriteString("  }\n\n")
		clusterIndex++
	}

	// Write proposed changes cluster
	if proposedNodes, ok := nodesByCluster["_proposed"]; ok && len(proposedNodes) > 0 {
		sb.WriteString(fmt.Sprintf("  subgraph cluster_%d {\n", clusterIndex))
		sb.WriteString("    label=\"Proposed Changes\";\n")
		sb.WriteString("    style=filled;\n")
		sb.WriteString("    color=lightyellow;\n")
		sb.WriteString("    node [style=filled];\n\n")

		for _, node := range proposedNodes {
			writeNode(&sb, node, "    ")
		}

		sb.WriteString("  }\n\n")
	}

	// Write edges: impact reference chains (bold)
	writtenEdges := make(map[string]bool)
	for _, provision := range result.DirectlyAffected {
		writeImpactEdge(&sb, provision, result, writtenEdges)
	}
	for _, provision := range result.TransitivelyAffected {
		writeImpactEdge(&sb, provision, result, writtenEdges)
	}

	// Write edges: broken cross-references (dashed red)
	// Use separate tracking so broken ref edges are never suppressed by impact edges.
	brokenEdgesSeen := make(map[string]bool)
	for _, brokenRef := range result.BrokenCrossRefs {
		edgeKey := brokenRef.SourceURI + "->" + brokenRef.TargetURI
		if brokenEdgesSeen[edgeKey] {
			continue
		}
		brokenEdgesSeen[edgeKey] = true

		edgeLabel := "broken ref"
		if brokenRef.Severity == SeverityError {
			edgeLabel = "BROKEN"
		} else if brokenRef.Severity == SeverityWarning {
			edgeLabel = "at risk"
		}

		sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [label=\"%s\" color=red style=dashed penwidth=2.0];\n",
			sanitizeDOTNodeID(brokenRef.SourceURI),
			sanitizeDOTNodeID(brokenRef.TargetURI),
			edgeLabel))
	}

	sb.WriteString("\n")

	// Legend subgraph
	writeLegend(&sb)

	sb.WriteString("}\n")
	return sb.String(), nil
}

// RenderImpactGraphToFile generates a DOT graph and writes it to the specified
// output path. The file can be rendered with Graphviz: `dot -Tpng output.dot -o output.png`
func RenderImpactGraphToFile(result *DraftImpactResult, outputPath string) error {
	dotContent, err := RenderImpactGraph(result)
	if err != nil {
		return fmt.Errorf("failed to render impact graph: %w", err)
	}
	if err := os.WriteFile(outputPath, []byte(dotContent), 0644); err != nil {
		return fmt.Errorf("failed to write DOT file: %w", err)
	}
	return nil
}

// dotNode represents a node in the DOT graph with its visual properties.
type dotNode struct {
	id        string
	label     string
	fillColor string
	category  string
	dashed    bool
}

// addDiffEntriesToClusters categorizes diff entries into cluster groups and
// assigns appropriate colors based on the change type.
func addDiffEntriesToClusters(entries []DiffEntry, changeType string, clusters map[string][]dotNode, emitted map[string]bool) {
	for _, entry := range entries {
		if emitted[entry.TargetURI] {
			continue
		}
		emitted[entry.TargetURI] = true

		fillColor := "red"
		switch changeType {
		case "added":
			fillColor = "palegreen"
		case "redesignated":
			fillColor = "plum"
		}

		amendmentLabel := extractURILabel(entry.TargetURI)
		if entry.Amendment.Description != "" {
			amendmentLabel = truncateLabel(entry.Amendment.Description, 40)
		}

		clusterKey := clusterKeyFromDocumentID(entry.TargetDocumentID)
		clusters[clusterKey] = append(clusters[clusterKey], dotNode{
			id:        entry.TargetURI,
			label:     amendmentLabel,
			fillColor: fillColor,
			category:  changeType,
		})
	}
}

// writeNode writes a single DOT node declaration to the builder.
func writeNode(sb *strings.Builder, node dotNode, indent string) {
	nodeID := sanitizeDOTNodeID(node.URI())
	escapedLabel := escapeDOTString(node.label)

	style := "filled"
	if node.dashed {
		style = "filled,dashed"
	}

	sb.WriteString(fmt.Sprintf("%s\"%s\" [label=\"%s\" style=\"%s\" fillcolor=%s];\n",
		indent, nodeID, escapedLabel, style, node.fillColor))
}

// URI returns the node's URI, used as the DOT node identifier.
func (n dotNode) URI() string {
	return n.id
}

// writeImpactEdge writes a bold edge from an affected provision to the target
// it references, deriving the target from the provision's Reason field.
func writeImpactEdge(sb *strings.Builder, provision AffectedProvision, result *DraftImpactResult, writtenEdges map[string]bool) {
	// Find what this provision references by scanning diff entries
	if result.Diff == nil {
		return
	}

	allEntries := collectAllDiffEntries(result.Diff)
	for _, entry := range allEntries {
		edgeKey := provision.URI + "->" + entry.TargetURI
		if writtenEdges[edgeKey] {
			continue
		}

		// Check if this provision's reason references this entry's target
		targetLabel := extractURILabel(entry.TargetURI)
		if strings.Contains(provision.Reason, targetLabel) {
			writtenEdges[edgeKey] = true
			sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [label=\"references\" color=blue style=bold penwidth=1.5];\n",
				sanitizeDOTNodeID(provision.URI),
				sanitizeDOTNodeID(entry.TargetURI)))
			break
		}
	}
}

// collectAllDiffEntries gathers all diff entries across categories into a
// single slice for iteration.
func collectAllDiffEntries(diff *DraftDiff) []DiffEntry {
	var allEntries []DiffEntry
	allEntries = append(allEntries, diff.Modified...)
	allEntries = append(allEntries, diff.Removed...)
	allEntries = append(allEntries, diff.Added...)
	allEntries = append(allEntries, diff.Redesignated...)
	return allEntries
}

// writeLegend writes a legend subgraph explaining the color coding.
func writeLegend(sb *strings.Builder) {
	sb.WriteString("  subgraph cluster_legend {\n")
	sb.WriteString("    label=\"Legend\";\n")
	sb.WriteString("    style=filled;\n")
	sb.WriteString("    color=white;\n")
	sb.WriteString("    node [shape=box fontsize=9];\n\n")

	sb.WriteString("    legend_modified [label=\"Modified/Repealed\" style=filled fillcolor=red fontcolor=white];\n")
	sb.WriteString("    legend_direct [label=\"Directly Affected\" style=filled fillcolor=orange];\n")
	sb.WriteString("    legend_transitive [label=\"Transitively Affected\" style=filled fillcolor=yellow];\n")
	sb.WriteString("    legend_added [label=\"Added\" style=filled fillcolor=palegreen];\n")
	sb.WriteString("    legend_redesignated [label=\"Redesignated\" style=filled fillcolor=plum];\n\n")

	sb.WriteString("    legend_modified -> legend_direct [style=invis];\n")
	sb.WriteString("    legend_direct -> legend_transitive [style=invis];\n")
	sb.WriteString("    legend_transitive -> legend_added [style=invis];\n")
	sb.WriteString("    legend_added -> legend_redesignated [style=invis];\n")

	sb.WriteString("  }\n\n")
}

// clusterKeyFromDocumentID derives a human-readable cluster label from a
// library document ID (e.g., "us-usc-title-15" becomes "Title 15").
func clusterKeyFromDocumentID(documentID string) string {
	if documentID == "" {
		return "_proposed"
	}
	if strings.HasPrefix(documentID, "us-usc-title-") {
		titleNum := strings.TrimPrefix(documentID, "us-usc-title-")
		return "Title " + titleNum
	}
	return documentID
}

// nodeLabel returns a display label for a node, preferring the resolved label
// over the URI-extracted label.
func nodeLabel(label, uri string) string {
	if label != "" {
		return truncateLabel(label, 40)
	}
	return truncateLabel(extractURILabel(uri), 40)
}

// truncateLabel shortens a label to maxLen characters, appending "..." if truncated.
func truncateLabel(label string, maxLen int) string {
	if len(label) <= maxLen {
		return label
	}
	return label[:maxLen-3] + "..."
}

// sanitizeDOTNodeID converts a URI into a valid DOT node identifier by
// replacing non-alphanumeric characters with underscores.
func sanitizeDOTNodeID(uri string) string {
	var sb strings.Builder
	for _, c := range uri {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			sb.WriteRune(c)
		} else {
			sb.WriteRune('_')
		}
	}
	return sb.String()
}

// escapeDOTString escapes special characters for DOT label and attribute strings.
func escapeDOTString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
