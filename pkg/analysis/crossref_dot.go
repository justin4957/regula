package analysis

import (
	"fmt"
	"strings"
)

// ToDOT generates a Graphviz DOT representation of the cross-reference result.
// Documents are rendered as cluster subgraphs with shared concepts as cross-cluster edges.
func (r *CrossRefResult) ToDOT() string {
	var sb strings.Builder

	sb.WriteString("digraph CrossLegislationAnalysis {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  compound=true;\n")
	sb.WriteString("  fontname=\"Helvetica\";\n")
	sb.WriteString("  node [fontname=\"Helvetica\" fontsize=10];\n")
	sb.WriteString("  edge [fontname=\"Helvetica\" fontsize=8];\n\n")

	// Document cluster subgraphs
	clusterIndex := 0
	documentNodeIDs := make(map[string]string) // docID -> DOT node ID for anchoring

	for _, doc := range r.Documents {
		nodeID := sanitizeDOTID(doc.ID)
		documentNodeIDs[doc.ID] = nodeID

		sb.WriteString(fmt.Sprintf("  subgraph cluster_%d {\n", clusterIndex))
		sb.WriteString(fmt.Sprintf("    label=\"%s\";\n", escapeDOTLabel(doc.Label)))
		sb.WriteString("    style=filled;\n")
		sb.WriteString("    color=lightgrey;\n")
		sb.WriteString("    node [style=filled];\n\n")

		// Document summary node
		summaryLabel := fmt.Sprintf("%s\\nArticles: %d\\nDefs: %d\\nRights: %d\\nObls: %d",
			escapeDOTLabel(doc.Label), doc.Articles, doc.Definitions, doc.Rights, doc.Obligations)
		sb.WriteString(fmt.Sprintf("    \"%s\" [label=\"%s\" shape=box fillcolor=lightyellow];\n",
			nodeID, summaryLabel))

		// Definitions node
		if doc.Definitions > 0 {
			defNodeID := nodeID + "_defs"
			sb.WriteString(fmt.Sprintf("    \"%s\" [label=\"%d definitions\" shape=ellipse fillcolor=lightblue];\n",
				defNodeID, doc.Definitions))
			sb.WriteString(fmt.Sprintf("    \"%s\" -> \"%s\" [label=\"defines\" color=blue];\n",
				nodeID, defNodeID))
		}

		// External refs node
		if doc.ExternalRefs > 0 {
			extNodeID := nodeID + "_extrefs"
			sb.WriteString(fmt.Sprintf("    \"%s\" [label=\"%d ext refs\" shape=ellipse fillcolor=lightsalmon];\n",
				extNodeID, doc.ExternalRefs))
			sb.WriteString(fmt.Sprintf("    \"%s\" -> \"%s\" [label=\"references\" color=red style=dashed];\n",
				nodeID, extNodeID))
		}

		sb.WriteString("  }\n\n")
		clusterIndex++
	}

	// Shared definition edges (green, solid)
	for _, overlap := range r.DefinitionOverlap {
		docIDs := make([]string, 0, len(overlap.Documents))
		for docID := range overlap.Documents {
			docIDs = append(docIDs, docID)
		}
		if len(docIDs) >= 2 {
			for i := 0; i < len(docIDs)-1; i++ {
				for j := i + 1; j < len(docIDs); j++ {
					sourceNode := documentNodeIDs[docIDs[i]] + "_defs"
					targetNode := documentNodeIDs[docIDs[j]] + "_defs"
					sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [label=\"%s\" color=green style=bold dir=both];\n",
						sourceNode, targetNode, escapeDOTLabel(overlap.Concept)))
				}
			}
		}
	}

	// Shared right edges (orange, solid)
	for _, overlap := range r.RightsOverlap {
		docIDs := make([]string, 0, len(overlap.Documents))
		for docID := range overlap.Documents {
			docIDs = append(docIDs, docID)
		}
		if len(docIDs) >= 2 {
			for i := 0; i < len(docIDs)-1; i++ {
				for j := i + 1; j < len(docIDs); j++ {
					sourceNode := documentNodeIDs[docIDs[i]]
					targetNode := documentNodeIDs[docIDs[j]]
					sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [label=\"right: %s\" color=orange style=bold dir=both];\n",
						sourceNode, targetNode, escapeDOTLabel(overlap.Concept)))
				}
			}
		}
	}

	// Shared obligation edges (brown, solid)
	for _, overlap := range r.ObligationOverlap {
		docIDs := make([]string, 0, len(overlap.Documents))
		for docID := range overlap.Documents {
			docIDs = append(docIDs, docID)
		}
		if len(docIDs) >= 2 {
			for i := 0; i < len(docIDs)-1; i++ {
				for j := i + 1; j < len(docIDs); j++ {
					sourceNode := documentNodeIDs[docIDs[i]]
					targetNode := documentNodeIDs[docIDs[j]]
					sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [label=\"obligation: %s\" color=brown style=bold dir=both];\n",
						sourceNode, targetNode, escapeDOTLabel(overlap.Concept)))
				}
			}
		}
	}

	// External reference target nodes (red, dashed edges)
	externalTargetNodes := make(map[string]bool)
	for _, cluster := range r.ExternalRefs {
		targetNodeID := "ext_" + sanitizeDOTID(cluster.Target)
		if !externalTargetNodes[targetNodeID] {
			externalTargetNodes[targetNodeID] = true
			targetLabel := cluster.Target
			if len(targetLabel) > 30 {
				targetLabel = targetLabel[:27] + "..."
			}
			sb.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\" shape=hexagon style=filled fillcolor=mistyrose];\n",
				targetNodeID, escapeDOTLabel(targetLabel)))
		}

		// Connect source documents to external target
		sourceDocSet := make(map[string]bool)
		for _, src := range cluster.Sources {
			sourceDocSet[src.Document] = true
		}
		for docID := range sourceDocSet {
			extRefNode := documentNodeIDs[docID] + "_extrefs"
			sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [color=red style=dashed];\n",
				extRefNode, targetNodeID))
		}
	}

	sb.WriteString("}\n")
	return sb.String()
}

// ToDOT generates a DOT representation of a comparison result.
func (r *ComparisonResult) ToDOT() string {
	var sb strings.Builder

	sb.WriteString("digraph DocumentComparison {\n")
	sb.WriteString("  rankdir=TB;\n")
	sb.WriteString("  fontname=\"Helvetica\";\n")
	sb.WriteString("  node [fontname=\"Helvetica\" fontsize=10];\n")
	sb.WriteString("  edge [fontname=\"Helvetica\" fontsize=8];\n\n")

	nodeA := sanitizeDOTID(r.DocumentA.ID)
	nodeB := sanitizeDOTID(r.DocumentB.ID)

	// Document A
	labelA := fmt.Sprintf("%s\\nArt: %d | Def: %d | Rts: %d | Obl: %d",
		escapeDOTLabel(r.DocumentA.Label), r.DocumentA.Articles, r.DocumentA.Definitions,
		r.DocumentA.Rights, r.DocumentA.Obligations)
	sb.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\" shape=box style=filled fillcolor=lightblue];\n",
		nodeA, labelA))

	// Document B
	labelB := fmt.Sprintf("%s\\nArt: %d | Def: %d | Rts: %d | Obl: %d",
		escapeDOTLabel(r.DocumentB.Label), r.DocumentB.Articles, r.DocumentB.Definitions,
		r.DocumentB.Rights, r.DocumentB.Obligations)
	sb.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\" shape=box style=filled fillcolor=lightgreen];\n",
		nodeB, labelB))

	// Shared definitions
	for _, overlap := range r.SharedDefinitions {
		sharedNode := "shared_def_" + sanitizeDOTID(overlap.Concept)
		sb.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\" shape=ellipse style=filled fillcolor=lightyellow];\n",
			sharedNode, escapeDOTLabel(overlap.Concept)))
		sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [label=\"defines\" color=green];\n", nodeA, sharedNode))
		sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [label=\"defines\" color=green];\n", nodeB, sharedNode))
	}

	// Shared rights
	for _, overlap := range r.SharedRights {
		sharedNode := "shared_right_" + sanitizeDOTID(overlap.Concept)
		sb.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\" shape=diamond style=filled fillcolor=lightsalmon];\n",
			sharedNode, escapeDOTLabel(overlap.Concept)))
		sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [label=\"grants\" color=orange];\n", nodeA, sharedNode))
		sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [label=\"grants\" color=orange];\n", nodeB, sharedNode))
	}

	// Shared external refs
	for _, ref := range r.SharedExternalRefs {
		extNode := "shared_ext_" + sanitizeDOTID(ref)
		extLabel := ref
		if len(extLabel) > 25 {
			extLabel = extLabel[:22] + "..."
		}
		sb.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\" shape=hexagon style=filled fillcolor=mistyrose];\n",
			extNode, escapeDOTLabel(extLabel)))
		sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [color=red style=dashed];\n", nodeA, extNode))
		sb.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [color=red style=dashed];\n", nodeB, extNode))
	}

	sb.WriteString("}\n")
	return sb.String()
}

// sanitizeDOTID converts a string into a valid DOT node identifier.
func sanitizeDOTID(s string) string {
	var sb strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			sb.WriteRune(c)
		} else {
			sb.WriteRune('_')
		}
	}
	return sb.String()
}

// escapeDOTLabel escapes special characters for DOT label strings.
func escapeDOTLabel(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
