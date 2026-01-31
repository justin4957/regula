// Package analysis provides cross-reference analysis for regulatory provisions.
package analysis

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/coolbeans/regula/pkg/store"
)

// CrossRefAnalyzer performs cross-legislation analysis across multiple documents.
type CrossRefAnalyzer struct {
	stores map[string]*store.TripleStore
	labels map[string]string
}

// NewCrossRefAnalyzer creates a new cross-reference analyzer.
func NewCrossRefAnalyzer() *CrossRefAnalyzer {
	return &CrossRefAnalyzer{
		stores: make(map[string]*store.TripleStore),
		labels: make(map[string]string),
	}
}

// AddDocument registers a document's triple store for analysis.
func (a *CrossRefAnalyzer) AddDocument(documentID, label string, tripleStore *store.TripleStore) {
	a.stores[documentID] = tripleStore
	a.labels[documentID] = label
}

// DocumentSummary provides an overview of a single document.
type DocumentSummary struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Articles     int    `json:"articles"`
	Definitions  int    `json:"definitions"`
	Rights       int    `json:"rights"`
	Obligations  int    `json:"obligations"`
	ExternalRefs int    `json:"external_refs"`
	References   int    `json:"references"`
	Triples      int    `json:"triples"`
}

// SharedConcept represents a concept shared across multiple documents.
type SharedConcept struct {
	Concept   string   `json:"concept"`
	Documents []string `json:"documents"`
	Type      string   `json:"type"`
}

// ExternalRefCluster groups external references pointing to the same target.
type ExternalRefCluster struct {
	Target  string              `json:"target"`
	Sources []ExternalRefSource `json:"sources"`
	Count   int                 `json:"count"`
}

// ExternalRefSource identifies where an external reference originates.
type ExternalRefSource struct {
	Document  string `json:"document"`
	Provision string `json:"provision"`
	RefText   string `json:"ref_text"`
}

// ConceptOverlap represents a concept found in multiple documents.
type ConceptOverlap struct {
	Concept   string              `json:"concept"`
	Documents map[string][]string `json:"documents"`
}

// CrossRefStats provides aggregate statistics across all documents.
type CrossRefStats struct {
	TotalDocuments        int `json:"total_documents"`
	TotalExternalRefs     int `json:"total_external_refs"`
	SharedDefinitions     int `json:"shared_definitions"`
	SharedRights          int `json:"shared_rights"`
	SharedObligations     int `json:"shared_obligations"`
	UniqueExternalTargets int `json:"unique_external_targets"`
}

// CrossRefResult contains the full cross-reference analysis.
type CrossRefResult struct {
	Documents         []DocumentSummary  `json:"documents"`
	SharedConcepts    []SharedConcept    `json:"shared_concepts"`
	ExternalRefs      []ExternalRefCluster `json:"external_refs"`
	RightsOverlap     []ConceptOverlap   `json:"rights_overlap"`
	ObligationOverlap []ConceptOverlap   `json:"obligation_overlap"`
	DefinitionOverlap []ConceptOverlap   `json:"definition_overlap"`
	Statistics        CrossRefStats      `json:"statistics"`
}

// ExternalRefReport provides external reference analysis for a single document.
type ExternalRefReport struct {
	DocumentID       string              `json:"document_id"`
	DocumentLabel    string              `json:"document_label"`
	TotalExternalRefs int               `json:"total_external_refs"`
	UniqueTargets    int                 `json:"unique_targets"`
	Clusters         []ExternalRefCluster `json:"clusters"`
	ByProvision      map[string][]string `json:"by_provision"`
}

// ComparisonResult contains a pair-wise document comparison.
type ComparisonResult struct {
	DocumentA         DocumentSummary  `json:"document_a"`
	DocumentB         DocumentSummary  `json:"document_b"`
	SharedDefinitions []ConceptOverlap `json:"shared_definitions"`
	SharedRights      []ConceptOverlap `json:"shared_rights"`
	SharedObligations []ConceptOverlap `json:"shared_obligations"`
	SharedExternalRefs []string        `json:"shared_external_refs"`
	Statistics        ComparisonStats  `json:"statistics"`
}

// ComparisonStats holds aggregate stats for a pair-wise comparison.
type ComparisonStats struct {
	SharedDefinitionCount  int `json:"shared_definition_count"`
	SharedRightCount       int `json:"shared_right_count"`
	SharedObligationCount  int `json:"shared_obligation_count"`
	SharedExternalRefCount int `json:"shared_external_ref_count"`
}

// Analyze performs full multi-document cross-reference analysis.
func (a *CrossRefAnalyzer) Analyze() *CrossRefResult {
	result := &CrossRefResult{
		Documents:         make([]DocumentSummary, 0, len(a.stores)),
		SharedConcepts:    make([]SharedConcept, 0),
		ExternalRefs:      make([]ExternalRefCluster, 0),
		RightsOverlap:     make([]ConceptOverlap, 0),
		ObligationOverlap: make([]ConceptOverlap, 0),
		DefinitionOverlap: make([]ConceptOverlap, 0),
	}

	// Build document summaries
	for docID := range a.stores {
		summary := a.buildDocumentSummary(docID)
		result.Documents = append(result.Documents, summary)
	}
	sort.Slice(result.Documents, func(i, j int) bool {
		return result.Documents[i].ID < result.Documents[j].ID
	})

	// Collect definitions, rights, obligations across all documents
	definitionsByDoc := a.collectDefinitions()
	rightsByDoc := a.collectRights()
	obligationsByDoc := a.collectObligations()
	externalRefsByDoc := a.collectExternalRefs()

	// Find overlapping definitions
	result.DefinitionOverlap = findOverlaps(definitionsByDoc)
	result.RightsOverlap = findOverlaps(rightsByDoc)
	result.ObligationOverlap = findOverlaps(obligationsByDoc)

	// Build shared concepts list
	for _, overlap := range result.DefinitionOverlap {
		documents := make([]string, 0, len(overlap.Documents))
		for docID := range overlap.Documents {
			documents = append(documents, docID)
		}
		sort.Strings(documents)
		result.SharedConcepts = append(result.SharedConcepts, SharedConcept{
			Concept:   overlap.Concept,
			Documents: documents,
			Type:      "definition",
		})
	}
	for _, overlap := range result.RightsOverlap {
		documents := make([]string, 0, len(overlap.Documents))
		for docID := range overlap.Documents {
			documents = append(documents, docID)
		}
		sort.Strings(documents)
		result.SharedConcepts = append(result.SharedConcepts, SharedConcept{
			Concept:   overlap.Concept,
			Documents: documents,
			Type:      "right",
		})
	}
	for _, overlap := range result.ObligationOverlap {
		documents := make([]string, 0, len(overlap.Documents))
		for docID := range overlap.Documents {
			documents = append(documents, docID)
		}
		sort.Strings(documents)
		result.SharedConcepts = append(result.SharedConcepts, SharedConcept{
			Concept:   overlap.Concept,
			Documents: documents,
			Type:      "obligation",
		})
	}

	// Aggregate external reference clusters
	result.ExternalRefs = a.aggregateExternalRefs(externalRefsByDoc)

	// Calculate statistics
	result.Statistics = CrossRefStats{
		TotalDocuments:        len(a.stores),
		SharedDefinitions:     len(result.DefinitionOverlap),
		SharedRights:          len(result.RightsOverlap),
		SharedObligations:     len(result.ObligationOverlap),
		UniqueExternalTargets: len(result.ExternalRefs),
	}
	for _, cluster := range result.ExternalRefs {
		result.Statistics.TotalExternalRefs += cluster.Count
	}

	return result
}

// AnalyzeExternalRefs produces a detailed external reference report for one document.
func (a *CrossRefAnalyzer) AnalyzeExternalRefs(documentID string) *ExternalRefReport {
	tripleStore, exists := a.stores[documentID]
	if !exists {
		return &ExternalRefReport{
			DocumentID:    documentID,
			DocumentLabel: documentID,
			Clusters:      make([]ExternalRefCluster, 0),
			ByProvision:   make(map[string][]string),
		}
	}

	report := &ExternalRefReport{
		DocumentID:    documentID,
		DocumentLabel: a.labels[documentID],
		Clusters:      make([]ExternalRefCluster, 0),
		ByProvision:   make(map[string][]string),
	}

	// Collect all external refs
	externalRefTriples := tripleStore.Find("", store.PropExternalRef, "")
	targetCounts := make(map[string][]ExternalRefSource)

	for _, triple := range externalRefTriples {
		report.TotalExternalRefs++
		report.ByProvision[triple.Subject] = append(report.ByProvision[triple.Subject], triple.Object)

		normalizedTarget := normalizeExternalRef(triple.Object)
		targetCounts[normalizedTarget] = append(targetCounts[normalizedTarget], ExternalRefSource{
			Document:  documentID,
			Provision: extractURILabel(triple.Subject),
			RefText:   triple.Object,
		})
	}

	// Build clusters sorted by frequency
	for target, sources := range targetCounts {
		report.Clusters = append(report.Clusters, ExternalRefCluster{
			Target:  target,
			Sources: sources,
			Count:   len(sources),
		})
	}
	sort.Slice(report.Clusters, func(i, j int) bool {
		if report.Clusters[i].Count != report.Clusters[j].Count {
			return report.Clusters[i].Count > report.Clusters[j].Count
		}
		return report.Clusters[i].Target < report.Clusters[j].Target
	})

	report.UniqueTargets = len(report.Clusters)
	return report
}

// CompareDocuments performs a pair-wise comparison of two documents.
func (a *CrossRefAnalyzer) CompareDocuments(documentAID, documentBID string) *ComparisonResult {
	result := &ComparisonResult{
		DocumentA:          a.buildDocumentSummary(documentAID),
		DocumentB:          a.buildDocumentSummary(documentBID),
		SharedDefinitions:  make([]ConceptOverlap, 0),
		SharedRights:       make([]ConceptOverlap, 0),
		SharedObligations:  make([]ConceptOverlap, 0),
		SharedExternalRefs: make([]string, 0),
	}

	// Collect per-document data
	definitionsA := a.collectDocDefinitions(documentAID)
	definitionsB := a.collectDocDefinitions(documentBID)
	rightsA := a.collectDocRights(documentAID)
	rightsB := a.collectDocRights(documentBID)
	obligationsA := a.collectDocObligations(documentAID)
	obligationsB := a.collectDocObligations(documentBID)
	externalRefsA := a.collectDocExternalRefTargets(documentAID)
	externalRefsB := a.collectDocExternalRefTargets(documentBID)

	// Find overlapping definitions
	for concept, provisionsA := range definitionsA {
		if provisionsB, exists := definitionsB[concept]; exists {
			result.SharedDefinitions = append(result.SharedDefinitions, ConceptOverlap{
				Concept: concept,
				Documents: map[string][]string{
					documentAID: provisionsA,
					documentBID: provisionsB,
				},
			})
		}
	}
	sort.Slice(result.SharedDefinitions, func(i, j int) bool {
		return result.SharedDefinitions[i].Concept < result.SharedDefinitions[j].Concept
	})

	// Find overlapping rights
	for concept, provisionsA := range rightsA {
		if provisionsB, exists := rightsB[concept]; exists {
			result.SharedRights = append(result.SharedRights, ConceptOverlap{
				Concept: concept,
				Documents: map[string][]string{
					documentAID: provisionsA,
					documentBID: provisionsB,
				},
			})
		}
	}
	sort.Slice(result.SharedRights, func(i, j int) bool {
		return result.SharedRights[i].Concept < result.SharedRights[j].Concept
	})

	// Find overlapping obligations
	for concept, provisionsA := range obligationsA {
		if provisionsB, exists := obligationsB[concept]; exists {
			result.SharedObligations = append(result.SharedObligations, ConceptOverlap{
				Concept: concept,
				Documents: map[string][]string{
					documentAID: provisionsA,
					documentBID: provisionsB,
				},
			})
		}
	}
	sort.Slice(result.SharedObligations, func(i, j int) bool {
		return result.SharedObligations[i].Concept < result.SharedObligations[j].Concept
	})

	// Find shared external reference targets
	for target := range externalRefsA {
		if _, exists := externalRefsB[target]; exists {
			result.SharedExternalRefs = append(result.SharedExternalRefs, target)
		}
	}
	sort.Strings(result.SharedExternalRefs)

	// Calculate statistics
	result.Statistics = ComparisonStats{
		SharedDefinitionCount:  len(result.SharedDefinitions),
		SharedRightCount:       len(result.SharedRights),
		SharedObligationCount:  len(result.SharedObligations),
		SharedExternalRefCount: len(result.SharedExternalRefs),
	}

	return result
}

// buildDocumentSummary constructs a summary for a single document.
func (a *CrossRefAnalyzer) buildDocumentSummary(documentID string) DocumentSummary {
	tripleStore, exists := a.stores[documentID]
	if !exists {
		return DocumentSummary{ID: documentID, Label: documentID}
	}

	summary := DocumentSummary{
		ID:    documentID,
		Label: a.labels[documentID],
	}

	summary.Triples = tripleStore.Count()
	summary.Articles = len(tripleStore.Find("", store.RDFType, store.ClassArticle))
	summary.Definitions = len(tripleStore.Find("", store.RDFType, store.ClassDefinedTerm))
	summary.Rights = len(tripleStore.Find("", store.PropGrantsRight, ""))
	summary.Obligations = len(tripleStore.Find("", store.PropImposesObligation, ""))
	summary.ExternalRefs = len(tripleStore.Find("", store.PropExternalRef, ""))
	summary.References = len(tripleStore.Find("", store.PropReferences, ""))

	return summary
}

// collectDefinitions gathers normalized defined terms per document.
// Returns: normalizedTerm -> docID -> list of provision URIs.
func (a *CrossRefAnalyzer) collectDefinitions() map[string]map[string][]string {
	collected := make(map[string]map[string][]string)
	for docID, tripleStore := range a.stores {
		termTriples := tripleStore.Find("", store.PropNormalizedTerm, "")
		for _, triple := range termTriples {
			normalizedTerm := strings.ToLower(triple.Object)
			if collected[normalizedTerm] == nil {
				collected[normalizedTerm] = make(map[string][]string)
			}
			collected[normalizedTerm][docID] = append(collected[normalizedTerm][docID], triple.Subject)
		}
	}
	return collected
}

// collectRights gathers right types per document.
func (a *CrossRefAnalyzer) collectRights() map[string]map[string][]string {
	collected := make(map[string]map[string][]string)
	for docID, tripleStore := range a.stores {
		rightTriples := tripleStore.Find("", store.PropGrantsRight, "")
		for _, triple := range rightTriples {
			rightType := normalizeConceptName(triple.Object)
			if collected[rightType] == nil {
				collected[rightType] = make(map[string][]string)
			}
			collected[rightType][docID] = append(collected[rightType][docID], triple.Subject)
		}
	}
	return collected
}

// collectObligations gathers obligation types per document.
func (a *CrossRefAnalyzer) collectObligations() map[string]map[string][]string {
	collected := make(map[string]map[string][]string)
	for docID, tripleStore := range a.stores {
		obligationTriples := tripleStore.Find("", store.PropImposesObligation, "")
		for _, triple := range obligationTriples {
			obligationType := normalizeConceptName(triple.Object)
			if collected[obligationType] == nil {
				collected[obligationType] = make(map[string][]string)
			}
			collected[obligationType][docID] = append(collected[obligationType][docID], triple.Subject)
		}
	}
	return collected
}

// collectExternalRefs gathers external references per document.
func (a *CrossRefAnalyzer) collectExternalRefs() map[string]map[string][]ExternalRefSource {
	collected := make(map[string]map[string][]ExternalRefSource)
	for docID, tripleStore := range a.stores {
		extRefTriples := tripleStore.Find("", store.PropExternalRef, "")
		for _, triple := range extRefTriples {
			normalizedTarget := normalizeExternalRef(triple.Object)
			if collected[normalizedTarget] == nil {
				collected[normalizedTarget] = make(map[string][]ExternalRefSource)
			}
			collected[normalizedTarget][docID] = append(collected[normalizedTarget][docID], ExternalRefSource{
				Document:  docID,
				Provision: extractURILabel(triple.Subject),
				RefText:   triple.Object,
			})
		}
	}
	return collected
}

// collectDocDefinitions gathers normalized terms for a single document.
func (a *CrossRefAnalyzer) collectDocDefinitions(documentID string) map[string][]string {
	tripleStore, exists := a.stores[documentID]
	if !exists {
		return nil
	}
	result := make(map[string][]string)
	for _, triple := range tripleStore.Find("", store.PropNormalizedTerm, "") {
		normalizedTerm := strings.ToLower(triple.Object)
		result[normalizedTerm] = append(result[normalizedTerm], triple.Subject)
	}
	return result
}

// collectDocRights gathers right types for a single document.
func (a *CrossRefAnalyzer) collectDocRights(documentID string) map[string][]string {
	tripleStore, exists := a.stores[documentID]
	if !exists {
		return nil
	}
	result := make(map[string][]string)
	for _, triple := range tripleStore.Find("", store.PropGrantsRight, "") {
		rightType := normalizeConceptName(triple.Object)
		result[rightType] = append(result[rightType], triple.Subject)
	}
	return result
}

// collectDocObligations gathers obligation types for a single document.
func (a *CrossRefAnalyzer) collectDocObligations(documentID string) map[string][]string {
	tripleStore, exists := a.stores[documentID]
	if !exists {
		return nil
	}
	result := make(map[string][]string)
	for _, triple := range tripleStore.Find("", store.PropImposesObligation, "") {
		obligationType := normalizeConceptName(triple.Object)
		result[obligationType] = append(result[obligationType], triple.Subject)
	}
	return result
}

// collectDocExternalRefTargets gathers unique external ref targets for a document.
func (a *CrossRefAnalyzer) collectDocExternalRefTargets(documentID string) map[string]bool {
	tripleStore, exists := a.stores[documentID]
	if !exists {
		return nil
	}
	result := make(map[string]bool)
	for _, triple := range tripleStore.Find("", store.PropExternalRef, "") {
		result[normalizeExternalRef(triple.Object)] = true
	}
	return result
}

// aggregateExternalRefs builds clusters across all documents.
func (a *CrossRefAnalyzer) aggregateExternalRefs(refsByDoc map[string]map[string][]ExternalRefSource) []ExternalRefCluster {
	clusters := make([]ExternalRefCluster, 0, len(refsByDoc))
	for target, docSources := range refsByDoc {
		cluster := ExternalRefCluster{
			Target:  target,
			Sources: make([]ExternalRefSource, 0),
		}
		for _, sources := range docSources {
			cluster.Sources = append(cluster.Sources, sources...)
			cluster.Count += len(sources)
		}
		sort.Slice(cluster.Sources, func(i, j int) bool {
			if cluster.Sources[i].Document != cluster.Sources[j].Document {
				return cluster.Sources[i].Document < cluster.Sources[j].Document
			}
			return cluster.Sources[i].Provision < cluster.Sources[j].Provision
		})
		clusters = append(clusters, cluster)
	}
	sort.Slice(clusters, func(i, j int) bool {
		if clusters[i].Count != clusters[j].Count {
			return clusters[i].Count > clusters[j].Count
		}
		return clusters[i].Target < clusters[j].Target
	})
	return clusters
}

// findOverlaps identifies concepts present in more than one document.
func findOverlaps(conceptsByDoc map[string]map[string][]string) []ConceptOverlap {
	overlaps := make([]ConceptOverlap, 0)
	for concept, docProvisions := range conceptsByDoc {
		if len(docProvisions) > 1 {
			overlaps = append(overlaps, ConceptOverlap{
				Concept:   concept,
				Documents: docProvisions,
			})
		}
	}
	sort.Slice(overlaps, func(i, j int) bool {
		return overlaps[i].Concept < overlaps[j].Concept
	})
	return overlaps
}

// normalizeExternalRef normalizes an external reference literal for matching.
func normalizeExternalRef(refText string) string {
	normalized := strings.ToLower(strings.TrimSpace(refText))
	// Remove common prefixes
	normalized = strings.TrimPrefix(normalized, "urn:external:")
	normalized = strings.TrimPrefix(normalized, "directive:")
	normalized = strings.TrimPrefix(normalized, "regulation:")
	// Collapse whitespace
	parts := strings.Fields(normalized)
	return strings.Join(parts, " ")
}

// normalizeConceptName extracts a readable name from a URI-like concept.
func normalizeConceptName(concept string) string {
	// Extract local name from prefixed URI
	if idx := strings.LastIndex(concept, "#"); idx != -1 {
		concept = concept[idx+1:]
	} else if idx := strings.LastIndex(concept, ":"); idx != -1 {
		concept = concept[idx+1:]
	}
	return strings.ToLower(concept)
}

// ToJSON serializes the cross-reference result to JSON.
func (r *CrossRefResult) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// ToJSON serializes the external ref report to JSON.
func (r *ExternalRefReport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// ToJSON serializes the comparison result to JSON.
func (r *ComparisonResult) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// String returns a human-readable string representation of the cross-ref result.
func (r *CrossRefResult) String() string {
	var sb strings.Builder

	sb.WriteString("Cross-Legislation Analysis\n")
	sb.WriteString("==========================\n\n")

	// Document summaries
	sb.WriteString(fmt.Sprintf("Documents analyzed: %d\n\n", r.Statistics.TotalDocuments))
	sb.WriteString("+----------------------+----------+------+-------+------+------+----------+\n")
	sb.WriteString("| Document             | Articles | Defs | Refs  | Rts  | Obls | ExtRefs  |\n")
	sb.WriteString("+----------------------+----------+------+-------+------+------+----------+\n")
	for _, doc := range r.Documents {
		label := doc.Label
		if len(label) > 20 {
			label = label[:17] + "..."
		}
		sb.WriteString(fmt.Sprintf("| %-20s | %8d | %4d | %5d | %4d | %4d | %8d |\n",
			label, doc.Articles, doc.Definitions, doc.References, doc.Rights, doc.Obligations, doc.ExternalRefs))
	}
	sb.WriteString("+----------------------+----------+------+-------+------+------+----------+\n\n")

	// Shared concepts
	if len(r.SharedConcepts) > 0 {
		sb.WriteString(fmt.Sprintf("Shared Concepts: %d\n", len(r.SharedConcepts)))
		sb.WriteString("---\n")
		for _, concept := range r.SharedConcepts {
			sb.WriteString(fmt.Sprintf("  [%s] %s — found in: %s\n",
				concept.Type, concept.Concept, strings.Join(concept.Documents, ", ")))
		}
		sb.WriteString("\n")
	}

	// Definition overlaps
	if len(r.DefinitionOverlap) > 0 {
		sb.WriteString(fmt.Sprintf("Shared Definitions: %d\n", len(r.DefinitionOverlap)))
		sb.WriteString("---\n")
		for _, overlap := range r.DefinitionOverlap {
			documents := make([]string, 0, len(overlap.Documents))
			for docID := range overlap.Documents {
				documents = append(documents, docID)
			}
			sort.Strings(documents)
			sb.WriteString(fmt.Sprintf("  %s — %s\n", overlap.Concept, strings.Join(documents, ", ")))
		}
		sb.WriteString("\n")
	}

	// Rights overlaps
	if len(r.RightsOverlap) > 0 {
		sb.WriteString(fmt.Sprintf("Shared Rights: %d\n", len(r.RightsOverlap)))
		sb.WriteString("---\n")
		for _, overlap := range r.RightsOverlap {
			documents := make([]string, 0, len(overlap.Documents))
			for docID := range overlap.Documents {
				documents = append(documents, docID)
			}
			sort.Strings(documents)
			sb.WriteString(fmt.Sprintf("  %s — %s\n", overlap.Concept, strings.Join(documents, ", ")))
		}
		sb.WriteString("\n")
	}

	// Obligation overlaps
	if len(r.ObligationOverlap) > 0 {
		sb.WriteString(fmt.Sprintf("Shared Obligations: %d\n", len(r.ObligationOverlap)))
		sb.WriteString("---\n")
		for _, overlap := range r.ObligationOverlap {
			documents := make([]string, 0, len(overlap.Documents))
			for docID := range overlap.Documents {
				documents = append(documents, docID)
			}
			sort.Strings(documents)
			sb.WriteString(fmt.Sprintf("  %s — %s\n", overlap.Concept, strings.Join(documents, ", ")))
		}
		sb.WriteString("\n")
	}

	// External reference clusters
	if len(r.ExternalRefs) > 0 {
		sb.WriteString(fmt.Sprintf("External Reference Targets: %d unique targets, %d total refs\n",
			r.Statistics.UniqueExternalTargets, r.Statistics.TotalExternalRefs))
		sb.WriteString("---\n")
		limit := len(r.ExternalRefs)
		if limit > 15 {
			limit = 15
		}
		for _, cluster := range r.ExternalRefs[:limit] {
			docSet := make(map[string]bool)
			for _, src := range cluster.Sources {
				docSet[src.Document] = true
			}
			documents := make([]string, 0, len(docSet))
			for docID := range docSet {
				documents = append(documents, docID)
			}
			sort.Strings(documents)
			sb.WriteString(fmt.Sprintf("  %s (%d refs from %s)\n",
				cluster.Target, cluster.Count, strings.Join(documents, ", ")))
		}
		if len(r.ExternalRefs) > 15 {
			sb.WriteString(fmt.Sprintf("  ... and %d more targets\n", len(r.ExternalRefs)-15))
		}
	}

	return sb.String()
}

// FormatTable formats the cross-ref result as a comparison table.
func (r *CrossRefResult) FormatTable() string {
	var sb strings.Builder

	sb.WriteString("Cross-Legislation Comparison Table\n")
	sb.WriteString("==================================\n\n")

	// Side-by-side metrics
	sb.WriteString("Structural Metrics:\n")
	sb.WriteString("+------------------+")
	for range r.Documents {
		sb.WriteString("----------+")
	}
	sb.WriteString("\n| Metric           |")
	for _, doc := range r.Documents {
		label := doc.ID
		if len(label) > 8 {
			label = label[:8]
		}
		sb.WriteString(fmt.Sprintf(" %8s |", label))
	}
	sb.WriteString("\n+------------------+")
	for range r.Documents {
		sb.WriteString("----------+")
	}

	metrics := []struct {
		name string
		fn   func(DocumentSummary) int
	}{
		{"Articles", func(d DocumentSummary) int { return d.Articles }},
		{"Definitions", func(d DocumentSummary) int { return d.Definitions }},
		{"References", func(d DocumentSummary) int { return d.References }},
		{"Rights", func(d DocumentSummary) int { return d.Rights }},
		{"Obligations", func(d DocumentSummary) int { return d.Obligations }},
		{"External Refs", func(d DocumentSummary) int { return d.ExternalRefs }},
		{"Total Triples", func(d DocumentSummary) int { return d.Triples }},
	}

	for _, metric := range metrics {
		sb.WriteString(fmt.Sprintf("\n| %-16s |", metric.name))
		for _, doc := range r.Documents {
			sb.WriteString(fmt.Sprintf(" %8d |", metric.fn(doc)))
		}
	}

	sb.WriteString("\n+------------------+")
	for range r.Documents {
		sb.WriteString("----------+")
	}
	sb.WriteString("\n\n")

	// Summary stats
	sb.WriteString(fmt.Sprintf("Shared definitions:  %d\n", r.Statistics.SharedDefinitions))
	sb.WriteString(fmt.Sprintf("Shared rights:       %d\n", r.Statistics.SharedRights))
	sb.WriteString(fmt.Sprintf("Shared obligations:  %d\n", r.Statistics.SharedObligations))
	sb.WriteString(fmt.Sprintf("External ref targets: %d\n", r.Statistics.UniqueExternalTargets))

	return sb.String()
}

// String returns a human-readable representation of an external ref report.
func (r *ExternalRefReport) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("External Reference Report: %s\n", r.DocumentLabel))
	sb.WriteString("=" + strings.Repeat("=", 50) + "\n\n")
	sb.WriteString(fmt.Sprintf("Total external references: %d\n", r.TotalExternalRefs))
	sb.WriteString(fmt.Sprintf("Unique external targets:   %d\n\n", r.UniqueTargets))

	if len(r.Clusters) > 0 {
		sb.WriteString("External Documents Referenced:\n")
		sb.WriteString("+------+---------------------------------------------------+\n")
		sb.WriteString("| Refs | Target Document                                   |\n")
		sb.WriteString("+------+---------------------------------------------------+\n")
		for _, cluster := range r.Clusters {
			target := cluster.Target
			if len(target) > 49 {
				target = target[:46] + "..."
			}
			sb.WriteString(fmt.Sprintf("| %4d | %-49s |\n", cluster.Count, target))
		}
		sb.WriteString("+------+---------------------------------------------------+\n\n")

		// Detail by cluster
		sb.WriteString("Detail by Target:\n")
		for _, cluster := range r.Clusters {
			sb.WriteString(fmt.Sprintf("  %s (%d references):\n", cluster.Target, cluster.Count))
			for _, src := range cluster.Sources {
				sb.WriteString(fmt.Sprintf("    - %s: %s\n", src.Provision, src.RefText))
			}
		}
	}

	return sb.String()
}

// String returns a human-readable comparison result.
func (r *ComparisonResult) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Comparison: %s vs %s\n", r.DocumentA.Label, r.DocumentB.Label))
	sb.WriteString("=" + strings.Repeat("=", 50) + "\n\n")

	// Side-by-side
	sb.WriteString("+------------------+----------+----------+\n")
	sb.WriteString(fmt.Sprintf("| Metric           | %-8s | %-8s |\n", truncate(r.DocumentA.ID, 8), truncate(r.DocumentB.ID, 8)))
	sb.WriteString("+------------------+----------+----------+\n")
	sb.WriteString(fmt.Sprintf("| Articles         | %8d | %8d |\n", r.DocumentA.Articles, r.DocumentB.Articles))
	sb.WriteString(fmt.Sprintf("| Definitions      | %8d | %8d |\n", r.DocumentA.Definitions, r.DocumentB.Definitions))
	sb.WriteString(fmt.Sprintf("| References       | %8d | %8d |\n", r.DocumentA.References, r.DocumentB.References))
	sb.WriteString(fmt.Sprintf("| Rights           | %8d | %8d |\n", r.DocumentA.Rights, r.DocumentB.Rights))
	sb.WriteString(fmt.Sprintf("| Obligations      | %8d | %8d |\n", r.DocumentA.Obligations, r.DocumentB.Obligations))
	sb.WriteString(fmt.Sprintf("| External Refs    | %8d | %8d |\n", r.DocumentA.ExternalRefs, r.DocumentB.ExternalRefs))
	sb.WriteString(fmt.Sprintf("| Total Triples    | %8d | %8d |\n", r.DocumentA.Triples, r.DocumentB.Triples))
	sb.WriteString("+------------------+----------+----------+\n\n")

	// Overlaps
	sb.WriteString(fmt.Sprintf("Shared definitions:     %d\n", r.Statistics.SharedDefinitionCount))
	sb.WriteString(fmt.Sprintf("Shared rights:          %d\n", r.Statistics.SharedRightCount))
	sb.WriteString(fmt.Sprintf("Shared obligations:     %d\n", r.Statistics.SharedObligationCount))
	sb.WriteString(fmt.Sprintf("Shared external refs:   %d\n\n", r.Statistics.SharedExternalRefCount))

	if len(r.SharedDefinitions) > 0 {
		sb.WriteString("Shared Definitions:\n")
		for _, overlap := range r.SharedDefinitions {
			sb.WriteString(fmt.Sprintf("  - %s\n", overlap.Concept))
		}
		sb.WriteString("\n")
	}

	if len(r.SharedRights) > 0 {
		sb.WriteString("Shared Rights:\n")
		for _, overlap := range r.SharedRights {
			sb.WriteString(fmt.Sprintf("  - %s\n", overlap.Concept))
		}
		sb.WriteString("\n")
	}

	if len(r.SharedObligations) > 0 {
		sb.WriteString("Shared Obligations:\n")
		for _, overlap := range r.SharedObligations {
			sb.WriteString(fmt.Sprintf("  - %s\n", overlap.Concept))
		}
		sb.WriteString("\n")
	}

	if len(r.SharedExternalRefs) > 0 {
		sb.WriteString("Shared External Reference Targets:\n")
		for _, ref := range r.SharedExternalRefs {
			sb.WriteString(fmt.Sprintf("  - %s\n", ref))
		}
	}

	return sb.String()
}

// truncate shortens a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
