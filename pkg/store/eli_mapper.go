package store

import "github.com/coolbeans/regula/pkg/extract"

// ELIEnrichmentStats tracks statistics about ELI vocabulary enrichment.
type ELIEnrichmentStats struct {
	ClassTriples    int `json:"eli_class_triples"`
	PropertyTriples int `json:"eli_property_triples"`
	TotalTriples    int `json:"eli_total_triples"`
}

// eliClassMapping maps reg: class constants to their ELI equivalents.
// Top-level document types map to eli:LegalResource;
// structural subdivisions map to eli:LegalResourceSubdivision.
var eliClassMapping = map[string]string{
	ClassRegulation: ELIClassLegalResource,
	ClassDirective:  ELIClassLegalResource,
	ClassDecision:   ELIClassLegalResource,
	ClassChapter:    ELIClassLegalResourceSubdivision,
	ClassSection:    ELIClassLegalResourceSubdivision,
	ClassArticle:    ELIClassLegalResourceSubdivision,
	ClassParagraph:  ELIClassLegalResourceSubdivision,
	ClassPoint:      ELIClassLegalResourceSubdivision,
	ClassPreamble:   ELIClassLegalResourceSubdivision,
	ClassRecital:    ELIClassLegalResourceSubdivision,
}

// eliPropertyMapping maps reg: predicates to their ELI equivalents.
// Only predicates with a clear semantic match are included.
// Notably, reg:text is NOT mapped to eli:description (ELI description
// is a summary, while reg:text is the full provision text).
var eliPropertyMapping = map[string]string{
	PropTitle:        ELIPropTitle,
	PropNumber:       ELIPropIDLocal,
	PropIdentifier:   ELIPropIDLocal,
	PropPartOf:       ELIPropIsPartOf,
	PropContains:     ELIPropHasPart,
	PropDate:         ELIPropDateDocument,
	PropVersion:      ELIPropVersion,
	PropReferences:   ELIPropCites,
	PropReferencedBy: ELIPropCitedBy,
}

// IsEUDocumentType returns true if the document type is an EU legislative type
// for which ELI vocabulary is appropriate. Non-EU formats (statutes, acts)
// return false.
func IsEUDocumentType(documentType extract.DocumentType) bool {
	switch documentType {
	case extract.DocumentTypeRegulation,
		extract.DocumentTypeDirective,
		extract.DocumentTypeDecision:
		return true
	default:
		return false
	}
}

// EnrichWithELI adds ELI vocabulary triples alongside existing reg: triples
// in the given store. Only EU document types (regulation, directive, decision)
// receive ELI enrichment; non-EU documents are left unchanged.
//
// This is an additive, idempotent operation: existing reg: triples are preserved
// and ELI triples are added alongside them. Calling this function multiple times
// produces the same result due to the TripleStore's deduplication.
func EnrichWithELI(tripleStore *TripleStore, documentType extract.DocumentType) *ELIEnrichmentStats {
	enrichmentStats := &ELIEnrichmentStats{}

	if !IsEUDocumentType(documentType) {
		return enrichmentStats
	}

	enrichClassTypes(tripleStore, enrichmentStats)
	enrichPropertyMappings(tripleStore, enrichmentStats)

	enrichmentStats.TotalTriples = enrichmentStats.ClassTriples + enrichmentStats.PropertyTriples
	return enrichmentStats
}

// enrichClassTypes adds ELI class type assertions (eli:LegalResource or
// eli:LegalResourceSubdivision) alongside existing reg: type assertions.
func enrichClassTypes(tripleStore *TripleStore, enrichmentStats *ELIEnrichmentStats) {
	for regulaClass, eliClass := range eliClassMapping {
		matchingTriples := tripleStore.Find("", RDFType, regulaClass)
		for _, matchingTriple := range matchingTriples {
			tripleStore.Add(matchingTriple.Subject, RDFType, eliClass)
			enrichmentStats.ClassTriples++
		}
	}
}

// enrichPropertyMappings adds ELI property triples alongside existing
// reg: property triples using the defined property mapping.
func enrichPropertyMappings(tripleStore *TripleStore, enrichmentStats *ELIEnrichmentStats) {
	for regulaProperty, eliProperty := range eliPropertyMapping {
		matchingTriples := tripleStore.Find("", regulaProperty, "")
		for _, matchingTriple := range matchingTriples {
			tripleStore.Add(matchingTriple.Subject, eliProperty, matchingTriple.Object)
			enrichmentStats.PropertyTriples++
		}
	}
}

// ELIClassMappingEntries returns a copy of the class mapping for documentation
// and testing purposes.
func ELIClassMappingEntries() map[string]string {
	mappingCopy := make(map[string]string, len(eliClassMapping))
	for regulaClass, eliClass := range eliClassMapping {
		mappingCopy[regulaClass] = eliClass
	}
	return mappingCopy
}

// ELIPropertyMappingEntries returns a copy of the property mapping for documentation
// and testing purposes.
func ELIPropertyMappingEntries() map[string]string {
	mappingCopy := make(map[string]string, len(eliPropertyMapping))
	for regulaProperty, eliProperty := range eliPropertyMapping {
		mappingCopy[regulaProperty] = eliProperty
	}
	return mappingCopy
}
