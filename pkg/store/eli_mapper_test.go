package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coolbeans/regula/pkg/extract"
)

func TestIsEUDocumentType(t *testing.T) {
	testCases := []struct {
		name         string
		documentType extract.DocumentType
		expectedEU   bool
	}{
		{"regulation is EU", extract.DocumentTypeRegulation, true},
		{"directive is EU", extract.DocumentTypeDirective, true},
		{"decision is EU", extract.DocumentTypeDecision, true},
		{"statute is not EU", extract.DocumentTypeStatute, false},
		{"act is not EU", extract.DocumentTypeAct, false},
		{"unknown is not EU", extract.DocumentTypeUnknown, false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := IsEUDocumentType(testCase.documentType)
			if result != testCase.expectedEU {
				t.Errorf("IsEUDocumentType(%q) = %v, want %v",
					testCase.documentType, result, testCase.expectedEU)
			}
		})
	}
}

func TestEnrichWithELI_ClassMapping(t *testing.T) {
	testCases := []struct {
		name         string
		regulaClass  string
		expectedELI  string
	}{
		{"regulation to legal resource", ClassRegulation, ELIClassLegalResource},
		{"directive to legal resource", ClassDirective, ELIClassLegalResource},
		{"decision to legal resource", ClassDecision, ELIClassLegalResource},
		{"chapter to subdivision", ClassChapter, ELIClassLegalResourceSubdivision},
		{"section to subdivision", ClassSection, ELIClassLegalResourceSubdivision},
		{"article to subdivision", ClassArticle, ELIClassLegalResourceSubdivision},
		{"paragraph to subdivision", ClassParagraph, ELIClassLegalResourceSubdivision},
		{"point to subdivision", ClassPoint, ELIClassLegalResourceSubdivision},
		{"preamble to subdivision", ClassPreamble, ELIClassLegalResourceSubdivision},
		{"recital to subdivision", ClassRecital, ELIClassLegalResourceSubdivision},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tripleStore := NewTripleStore()
			subjectURI := "https://example.org/test"

			// Add a reg: type triple
			tripleStore.Add(subjectURI, RDFType, testCase.regulaClass)

			// Enrich with ELI
			enrichmentStats := EnrichWithELI(tripleStore, extract.DocumentTypeRegulation)

			// Verify ELI class was added
			eliTypeTriples := tripleStore.Find(subjectURI, RDFType, testCase.expectedELI)
			if len(eliTypeTriples) == 0 {
				t.Errorf("Expected ELI type %s for %s, but not found",
					testCase.expectedELI, testCase.regulaClass)
			}

			// Verify original reg: type still exists
			regTypeTriples := tripleStore.Find(subjectURI, RDFType, testCase.regulaClass)
			if len(regTypeTriples) == 0 {
				t.Errorf("Original reg: type %s was removed", testCase.regulaClass)
			}

			if enrichmentStats.ClassTriples == 0 {
				t.Error("Expected ClassTriples > 0")
			}
		})
	}
}

func TestEnrichWithELI_PropertyMapping(t *testing.T) {
	testCases := []struct {
		name             string
		regulaPredicate  string
		eliPredicate     string
		objectValue      string
	}{
		{"title mapping", PropTitle, ELIPropTitle, "Right to erasure"},
		{"number to id_local", PropNumber, ELIPropIDLocal, "17"},
		{"identifier to id_local", PropIdentifier, ELIPropIDLocal, "(EU) 2016/679"},
		{"partOf to is_part_of", PropPartOf, ELIPropIsPartOf, "https://example.org/parent"},
		{"contains to has_part", PropContains, ELIPropHasPart, "https://example.org/child"},
		{"date to date_document", PropDate, ELIPropDateDocument, "2016-04-27"},
		{"version to version", PropVersion, ELIPropVersion, "1.0"},
		{"references to cites", PropReferences, ELIPropCites, "https://example.org/target"},
		{"referencedBy to cited_by", PropReferencedBy, ELIPropCitedBy, "https://example.org/source"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tripleStore := NewTripleStore()
			subjectURI := "https://example.org/test"

			// Add a reg: property triple
			tripleStore.Add(subjectURI, testCase.regulaPredicate, testCase.objectValue)

			// Enrich with ELI
			enrichmentStats := EnrichWithELI(tripleStore, extract.DocumentTypeRegulation)

			// Verify ELI property was added with same subject and object
			eliTriples := tripleStore.Find(subjectURI, testCase.eliPredicate, testCase.objectValue)
			if len(eliTriples) == 0 {
				t.Errorf("Expected ELI property %s with value %q, but not found",
					testCase.eliPredicate, testCase.objectValue)
			}

			// Verify original reg: property still exists
			regTriples := tripleStore.Find(subjectURI, testCase.regulaPredicate, testCase.objectValue)
			if len(regTriples) == 0 {
				t.Errorf("Original reg: property %s was removed", testCase.regulaPredicate)
			}

			if enrichmentStats.PropertyTriples == 0 {
				t.Error("Expected PropertyTriples > 0")
			}
		})
	}
}

func TestEnrichWithELI_NonEUDocument(t *testing.T) {
	nonEUDocumentTypes := []extract.DocumentType{
		extract.DocumentTypeStatute,
		extract.DocumentTypeAct,
		extract.DocumentTypeUnknown,
	}

	for _, documentType := range nonEUDocumentTypes {
		t.Run(string(documentType), func(t *testing.T) {
			tripleStore := NewTripleStore()
			subjectURI := "https://example.org/test"

			tripleStore.Add(subjectURI, RDFType, ClassArticle)
			tripleStore.Add(subjectURI, PropTitle, "Test Article")
			tripleStore.Add(subjectURI, PropNumber, "1")

			initialCount := tripleStore.Count()

			enrichmentStats := EnrichWithELI(tripleStore, documentType)

			finalCount := tripleStore.Count()
			if finalCount != initialCount {
				t.Errorf("Non-EU document should not be enriched: initial=%d, final=%d",
					initialCount, finalCount)
			}

			if enrichmentStats.TotalTriples != 0 {
				t.Errorf("Expected zero ELI triples for non-EU document, got %d",
					enrichmentStats.TotalTriples)
			}
		})
	}
}

func TestEnrichWithELI_Idempotent(t *testing.T) {
	tripleStore := NewTripleStore()
	subjectURI := "https://example.org/test"

	tripleStore.Add(subjectURI, RDFType, ClassArticle)
	tripleStore.Add(subjectURI, PropTitle, "Test Article")
	tripleStore.Add(subjectURI, PropNumber, "1")
	tripleStore.Add(subjectURI, PropPartOf, "https://example.org/parent")

	// First enrichment
	EnrichWithELI(tripleStore, extract.DocumentTypeRegulation)
	countAfterFirstEnrichment := tripleStore.Count()

	// Second enrichment
	EnrichWithELI(tripleStore, extract.DocumentTypeRegulation)
	countAfterSecondEnrichment := tripleStore.Count()

	if countAfterFirstEnrichment != countAfterSecondEnrichment {
		t.Errorf("Enrichment is not idempotent: first=%d, second=%d",
			countAfterFirstEnrichment, countAfterSecondEnrichment)
	}
}

func TestEnrichWithELI_PreservesExistingTriples(t *testing.T) {
	tripleStore := NewTripleStore()
	subjectURI := "https://example.org/article17"
	parentURI := "https://example.org/chapterIII"
	regulationURI := "https://example.org/GDPR"

	// Build a realistic article with multiple predicates
	tripleStore.Add(subjectURI, RDFType, ClassArticle)
	tripleStore.Add(subjectURI, PropNumber, "17")
	tripleStore.Add(subjectURI, PropTitle, "Right to erasure")
	tripleStore.Add(subjectURI, PropText, "The data subject shall have the right...")
	tripleStore.Add(subjectURI, PropPartOf, parentURI)
	tripleStore.Add(subjectURI, PropBelongsTo, regulationURI)

	originalTriples := tripleStore.All()
	originalCount := len(originalTriples)

	// Enrich
	EnrichWithELI(tripleStore, extract.DocumentTypeRegulation)

	// Verify all original triples still exist
	for _, originalTriple := range originalTriples {
		found := tripleStore.Find(originalTriple.Subject, originalTriple.Predicate, originalTriple.Object)
		if len(found) == 0 {
			t.Errorf("Original triple lost after enrichment: <%s, %s, %s>",
				originalTriple.Subject, originalTriple.Predicate, originalTriple.Object)
		}
	}

	// Verify new triples were added (should have more than original)
	finalCount := tripleStore.Count()
	if finalCount <= originalCount {
		t.Errorf("Expected more triples after enrichment: original=%d, final=%d",
			originalCount, finalCount)
	}
}

func TestEnrichWithELI_Stats(t *testing.T) {
	tripleStore := NewTripleStore()

	// Add triples that will match both class and property mappings
	tripleStore.Add("https://example.org/reg", RDFType, ClassRegulation)
	tripleStore.Add("https://example.org/reg", PropTitle, "Test Regulation")
	tripleStore.Add("https://example.org/art1", RDFType, ClassArticle)
	tripleStore.Add("https://example.org/art1", PropTitle, "Article One")
	tripleStore.Add("https://example.org/art1", PropNumber, "1")
	tripleStore.Add("https://example.org/art1", PropPartOf, "https://example.org/ch1")
	tripleStore.Add("https://example.org/ch1", RDFType, ClassChapter)
	tripleStore.Add("https://example.org/ch1", PropContains, "https://example.org/art1")
	tripleStore.Add("https://example.org/art1", PropReferences, "https://example.org/art2")
	tripleStore.Add("https://example.org/art2", PropReferencedBy, "https://example.org/art1")

	beforeCount := tripleStore.Count()

	enrichmentStats := EnrichWithELI(tripleStore, extract.DocumentTypeRegulation)

	afterCount := tripleStore.Count()
	actualNewTriples := afterCount - beforeCount

	// ClassTriples: 3 class mappings (regulation->LegalResource, article->Subdivision, chapter->Subdivision)
	if enrichmentStats.ClassTriples != 3 {
		t.Errorf("Expected 3 class triples, got %d", enrichmentStats.ClassTriples)
	}

	// PropertyTriples: title*2 + number*1 + partOf*1 + contains*1 + references*1 + referencedBy*1 = 7
	if enrichmentStats.PropertyTriples != 7 {
		t.Errorf("Expected 7 property triples, got %d", enrichmentStats.PropertyTriples)
	}

	if enrichmentStats.TotalTriples != enrichmentStats.ClassTriples+enrichmentStats.PropertyTriples {
		t.Errorf("TotalTriples (%d) != ClassTriples (%d) + PropertyTriples (%d)",
			enrichmentStats.TotalTriples, enrichmentStats.ClassTriples, enrichmentStats.PropertyTriples)
	}

	// Verify actual new triples matches stats total
	if actualNewTriples != enrichmentStats.TotalTriples {
		t.Errorf("Stats total (%d) does not match actual new triples (%d)",
			enrichmentStats.TotalTriples, actualNewTriples)
	}
}

func TestEnrichWithELI_MappingTableCompleteness(t *testing.T) {
	classMapping := ELIClassMappingEntries()
	propertyMapping := ELIPropertyMappingEntries()

	// Verify all document-level classes map to LegalResource
	documentClasses := []string{ClassRegulation, ClassDirective, ClassDecision}
	for _, documentClass := range documentClasses {
		eliClass, exists := classMapping[documentClass]
		if !exists {
			t.Errorf("Missing class mapping for %s", documentClass)
		}
		if eliClass != ELIClassLegalResource {
			t.Errorf("Expected %s -> %s, got %s", documentClass, ELIClassLegalResource, eliClass)
		}
	}

	// Verify all structural classes map to LegalResourceSubdivision
	structuralClasses := []string{
		ClassChapter, ClassSection, ClassArticle,
		ClassParagraph, ClassPoint, ClassPreamble, ClassRecital,
	}
	for _, structuralClass := range structuralClasses {
		eliClass, exists := classMapping[structuralClass]
		if !exists {
			t.Errorf("Missing class mapping for %s", structuralClass)
		}
		if eliClass != ELIClassLegalResourceSubdivision {
			t.Errorf("Expected %s -> %s, got %s",
				structuralClass, ELIClassLegalResourceSubdivision, eliClass)
		}
	}

	// Verify expected property mappings exist
	expectedPropertyMappings := map[string]string{
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

	for regulaProperty, expectedELI := range expectedPropertyMappings {
		actualELI, exists := propertyMapping[regulaProperty]
		if !exists {
			t.Errorf("Missing property mapping for %s", regulaProperty)
		}
		if actualELI != expectedELI {
			t.Errorf("Expected %s -> %s, got %s", regulaProperty, expectedELI, actualELI)
		}
	}

	// Verify reg:text is NOT mapped (intentional design decision)
	if _, exists := propertyMapping[PropText]; exists {
		t.Error("reg:text should NOT be mapped to ELI (ELI description is a summary, not full text)")
	}
}

func TestEnrichWithELI_EmptyStore(t *testing.T) {
	tripleStore := NewTripleStore()

	enrichmentStats := EnrichWithELI(tripleStore, extract.DocumentTypeRegulation)

	if enrichmentStats.TotalTriples != 0 {
		t.Errorf("Expected zero triples for empty store, got %d", enrichmentStats.TotalTriples)
	}

	if tripleStore.Count() != 0 {
		t.Errorf("Expected empty store after enrichment of empty store, got %d triples",
			tripleStore.Count())
	}
}

func TestEnrichWithELI_MultipleSubjectsWithSameClass(t *testing.T) {
	tripleStore := NewTripleStore()

	// Add multiple articles
	tripleStore.Add("https://example.org/art1", RDFType, ClassArticle)
	tripleStore.Add("https://example.org/art2", RDFType, ClassArticle)
	tripleStore.Add("https://example.org/art3", RDFType, ClassArticle)

	EnrichWithELI(tripleStore, extract.DocumentTypeRegulation)

	// All three should have ELI type
	for _, articleURI := range []string{
		"https://example.org/art1",
		"https://example.org/art2",
		"https://example.org/art3",
	} {
		eliTriples := tripleStore.Find(articleURI, RDFType, ELIClassLegalResourceSubdivision)
		if len(eliTriples) == 0 {
			t.Errorf("Article %s missing ELI type", articleURI)
		}
	}
}

func TestEnrichWithELI_GDPRIntegration(t *testing.T) {
	testdataPath := filepath.Join("..", "..", "testdata", "gdpr.txt")
	gdprFile, err := os.Open(testdataPath)
	if err != nil {
		t.Skipf("Skipping GDPR integration test: %v", err)
	}
	defer gdprFile.Close()

	parser := extract.NewParser()
	document, err := parser.Parse(gdprFile)
	if err != nil {
		t.Fatalf("Failed to parse GDPR: %v", err)
	}

	// Build the full graph
	tripleStore := NewTripleStore()
	graphBuilder := NewGraphBuilder(tripleStore, "https://regula.dev/regulations/")
	buildStats, err := graphBuilder.Build(document)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	tripleCountBeforeELI := tripleStore.Count()

	// Enrich with ELI
	enrichmentStats := EnrichWithELI(tripleStore, document.Type)

	tripleCountAfterELI := tripleStore.Count()

	t.Logf("GDPR ELI Enrichment:")
	t.Logf("  Build stats - articles: %d, chapters: %d, recitals: %d",
		buildStats.Articles, buildStats.Chapters, buildStats.Recitals)
	t.Logf("  Triples before ELI: %d", tripleCountBeforeELI)
	t.Logf("  Triples after ELI:  %d", tripleCountAfterELI)
	t.Logf("  ELI class triples:    %d", enrichmentStats.ClassTriples)
	t.Logf("  ELI property triples: %d", enrichmentStats.PropertyTriples)
	t.Logf("  ELI total triples:    %d", enrichmentStats.TotalTriples)

	// GDPR is an EU Regulation, so ELI should be applied
	if enrichmentStats.TotalTriples == 0 {
		t.Fatal("Expected ELI triples for GDPR (EU regulation)")
	}

	// Should have class triples for all structural elements
	// GDPR has 99 articles + 11 chapters + recitals + preamble + regulation = lots
	if enrichmentStats.ClassTriples < 100 {
		t.Errorf("Expected at least 100 class triples for GDPR, got %d",
			enrichmentStats.ClassTriples)
	}

	// Should have property triples for titles, numbers, partOf, contains etc
	if enrichmentStats.PropertyTriples < 200 {
		t.Errorf("Expected at least 200 property triples for GDPR, got %d",
			enrichmentStats.PropertyTriples)
	}

	// Verify specific GDPR entities got ELI types
	regulationURI := graphBuilder.GetRegulationURI()
	eliRegTriples := tripleStore.Find(regulationURI, RDFType, ELIClassLegalResource)
	if len(eliRegTriples) == 0 {
		t.Error("GDPR regulation should be typed as eli:LegalResource")
	}

	// Verify Article 17 got ELI subdivision type
	article17URI := graphBuilder.GetBaseURI() + graphBuilder.GetRegID() + ":Art17"
	eliArt17Triples := tripleStore.Find(article17URI, RDFType, ELIClassLegalResourceSubdivision)
	if len(eliArt17Triples) == 0 {
		t.Error("Article 17 should be typed as eli:LegalResourceSubdivision")
	}

	// Verify Article 17 has eli:title
	eliTitleTriples := tripleStore.Find(article17URI, ELIPropTitle, "")
	if len(eliTitleTriples) == 0 {
		t.Error("Article 17 should have eli:title")
	}

	// Verify Article 17 has eli:id_local
	eliIDLocalTriples := tripleStore.Find(article17URI, ELIPropIDLocal, "")
	if len(eliIDLocalTriples) == 0 {
		t.Error("Article 17 should have eli:id_local")
	}

	// Verify eli:is_part_of exists for articles
	eliIsPartOfTriples := tripleStore.Find(article17URI, ELIPropIsPartOf, "")
	if len(eliIsPartOfTriples) == 0 {
		t.Error("Article 17 should have eli:is_part_of")
	}
}
