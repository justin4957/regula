package store

import (
	"strings"
	"testing"
)

func TestNamespaces(t *testing.T) {
	// Verify namespaces are valid URIs
	namespaces := []struct {
		name string
		uri  string
	}{
		{"Reg", NamespaceReg},
		{"RDF", NamespaceRDF},
		{"RDFS", NamespaceRDFS},
		{"XSD", NamespaceXSD},
		{"DC", NamespaceDC},
	}

	for _, ns := range namespaces {
		if !strings.HasPrefix(ns.uri, "http") {
			t.Errorf("Namespace %s should be a valid URI, got %s", ns.name, ns.uri)
		}
	}
}

func TestPrefixes(t *testing.T) {
	// Verify prefixes end with colon
	prefixes := []string{PrefixReg, PrefixRDF, PrefixRDFS, PrefixXSD, PrefixDC}

	for _, prefix := range prefixes {
		if !strings.HasSuffix(prefix, ":") {
			t.Errorf("Prefix %s should end with colon", prefix)
		}
	}
}

func TestClassConstants(t *testing.T) {
	// Verify all classes use reg: prefix
	classes := []string{
		ClassRegulation,
		ClassDirective,
		ClassDecision,
		ClassChapter,
		ClassSection,
		ClassArticle,
		ClassParagraph,
		ClassPoint,
		ClassSubPoint,
		ClassRecital,
		ClassPreamble,
		ClassDefinedTerm,
		ClassReference,
		ClassObligation,
		ClassRight,
	}

	for _, class := range classes {
		if !strings.HasPrefix(class, "reg:") {
			t.Errorf("Class %s should have reg: prefix", class)
		}
	}
}

func TestPropertyConstants(t *testing.T) {
	// Verify properties have correct prefixes
	regProps := []string{
		PropTitle,
		PropText,
		PropNumber,
		PropIdentifier,
		PropPartOf,
		PropContains,
		PropReferences,
		PropDefinedIn,
		PropAmends,
		PropGrantsRight,
		PropImposesObligation,
	}

	for _, prop := range regProps {
		if !strings.HasPrefix(prop, "reg:") {
			t.Errorf("Property %s should have reg: prefix", prop)
		}
	}

	// RDF standard properties
	if !strings.HasPrefix(RDFType, "rdf:") {
		t.Errorf("RDFType should have rdf: prefix")
	}
	if !strings.HasPrefix(RDFSLabel, "rdfs:") {
		t.Errorf("RDFSLabel should have rdfs: prefix")
	}
}

func TestURIBuilder_Regulation(t *testing.T) {
	builder := NewURIBuilder("https://example.org/")

	uri := builder.Regulation("GDPR")
	expected := "https://example.org/GDPR"

	if uri != expected {
		t.Errorf("Regulation URI: got %s, want %s", uri, expected)
	}
}

func TestURIBuilder_Chapter(t *testing.T) {
	builder := NewURIBuilder("https://example.org/")

	uri := builder.Chapter("GDPR", "III")
	expected := "https://example.org/GDPR:ChapterIII"

	if uri != expected {
		t.Errorf("Chapter URI: got %s, want %s", uri, expected)
	}
}

func TestURIBuilder_Section(t *testing.T) {
	builder := NewURIBuilder("https://example.org/")

	uri := builder.Section("GDPR", "III", 2)
	expected := "https://example.org/GDPR:ChapterIII:Section2"

	if uri != expected {
		t.Errorf("Section URI: got %s, want %s", uri, expected)
	}
}

func TestURIBuilder_Article(t *testing.T) {
	builder := NewURIBuilder("https://example.org/")

	uri := builder.Article("GDPR", 17)
	expected := "https://example.org/GDPR:Art17"

	if uri != expected {
		t.Errorf("Article URI: got %s, want %s", uri, expected)
	}
}

func TestURIBuilder_Paragraph(t *testing.T) {
	builder := NewURIBuilder("https://example.org/")

	uri := builder.Paragraph("GDPR", 17, 1)
	expected := "https://example.org/GDPR:Art17:1"

	if uri != expected {
		t.Errorf("Paragraph URI: got %s, want %s", uri, expected)
	}
}

func TestURIBuilder_Point(t *testing.T) {
	builder := NewURIBuilder("https://example.org/")

	uri := builder.Point("GDPR", 6, 1, "a")
	expected := "https://example.org/GDPR:Art6:1:a"

	if uri != expected {
		t.Errorf("Point URI: got %s, want %s", uri, expected)
	}
}

func TestURIBuilder_Recital(t *testing.T) {
	builder := NewURIBuilder("https://example.org/")

	uri := builder.Recital("GDPR", 173)
	expected := "https://example.org/GDPR:Recital173"

	if uri != expected {
		t.Errorf("Recital URI: got %s, want %s", uri, expected)
	}
}

func TestURIBuilder_DefinedTerm(t *testing.T) {
	builder := NewURIBuilder("https://example.org/")

	tests := []struct {
		term     string
		expected string
	}{
		{"personal data", "https://example.org/GDPR:Term:personal_data"},
		{"controller", "https://example.org/GDPR:Term:controller"},
		{"data subject", "https://example.org/GDPR:Term:data_subject"},
		{"cross-border processing", "https://example.org/GDPR:Term:cross-border_processing"},
	}

	for _, tc := range tests {
		uri := builder.DefinedTerm("GDPR", tc.term)
		if uri != tc.expected {
			t.Errorf("DefinedTerm(%q): got %s, want %s", tc.term, uri, tc.expected)
		}
	}
}

func TestDefaultURIBuilder(t *testing.T) {
	builder := DefaultURIBuilder()

	if builder.BaseURI != NamespaceReg {
		t.Errorf("DefaultURIBuilder should use NamespaceReg, got %s", builder.BaseURI)
	}
}

func TestGDPRURIBuilder(t *testing.T) {
	builder := GDPRURIBuilder()

	if !strings.Contains(builder.BaseURI, "GDPR") {
		t.Errorf("GDPRURIBuilder should have GDPR in base URI, got %s", builder.BaseURI)
	}

	// Test article URI
	uri := builder.Article("", 17)
	if !strings.Contains(uri, "Art17") {
		t.Errorf("GDPR Article URI should contain Art17, got %s", uri)
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{17, "17"},
		{99, "99"},
		{173, "173"},
	}

	for _, tc := range tests {
		result := itoa(tc.input)
		if result != tc.expected {
			t.Errorf("itoa(%d): got %s, want %s", tc.input, result, tc.expected)
		}
	}
}

func TestSchemaTripleStoreIntegration(t *testing.T) {
	// Test that schema constants work with TripleStore
	store := NewTripleStore()
	builder := GDPRURIBuilder()

	// Add GDPR structure
	gdpr := builder.Regulation("")
	art17 := builder.Article("", 17)
	chapterIII := builder.Chapter("", "III")

	store.Add(gdpr, RDFType, ClassRegulation)
	store.Add(gdpr, PropTitle, "General Data Protection Regulation")

	store.Add(art17, RDFType, ClassArticle)
	store.Add(art17, PropTitle, "Right to erasure ('right to be forgotten')")
	store.Add(art17, PropNumber, "17")
	store.Add(art17, PropPartOf, chapterIII)
	store.Add(art17, PropGrantsRight, RightErasure)

	store.Add(chapterIII, RDFType, ClassChapter)
	store.Add(chapterIII, PropTitle, "Rights of the data subject")
	store.Add(chapterIII, PropContains, art17)

	// Verify queries work
	if store.Count() != 10 {
		t.Errorf("Expected 10 triples, got %d", store.Count())
	}

	// Query all articles
	articles := store.Find("", RDFType, ClassArticle)
	if len(articles) != 1 {
		t.Errorf("Expected 1 article, got %d", len(articles))
	}

	// Query provisions granting rights
	rightsProvisions := store.Find("", PropGrantsRight, "")
	if len(rightsProvisions) != 1 {
		t.Errorf("Expected 1 right-granting provision, got %d", len(rightsProvisions))
	}

	// Query what Art 17 is part of
	partOf := store.Find(art17, PropPartOf, "")
	if len(partOf) != 1 {
		t.Errorf("Expected Art 17 to be part of 1 chapter, got %d", len(partOf))
	}
	if partOf[0].Object != chapterIII {
		t.Errorf("Art 17 should be part of Chapter III, got %s", partOf[0].Object)
	}
}

func TestRightAndObligationConstants(t *testing.T) {
	// Verify all right constants have proper prefix
	rights := []string{
		RightAccess,
		RightRectification,
		RightErasure,
		RightRestriction,
		RightPortability,
		RightObject,
		RightNotAutomated,
		RightWithdrawConsent,
		RightLodgeComplaint,
		RightEffectiveRemedy,
		RightCompensation,
		RightInformation,
	}

	for _, right := range rights {
		if !strings.HasPrefix(right, "reg:Right") {
			t.Errorf("Right %s should have reg:Right prefix", right)
		}
	}

	// Verify obligation constants
	obligations := []string{
		ObligationTransparency,
		ObligationNotify,
		ObligationSecure,
		ObligationRecord,
		ObligationImpactAssessment,
		ObligationCooperate,
		ObligationAppoint,
	}

	for _, obligation := range obligations {
		if !strings.HasPrefix(obligation, "reg:") {
			t.Errorf("Obligation %s should have reg: prefix", obligation)
		}
	}
}

func TestLegalBasisConstants(t *testing.T) {
	bases := []string{
		LegalBasisConsent,
		LegalBasisContract,
		LegalBasisLegalObligation,
		LegalBasisVitalInterest,
		LegalBasisPublicTask,
		LegalBasisLegitimateInterest,
	}

	for _, basis := range bases {
		if !strings.HasPrefix(basis, "reg:") {
			t.Errorf("Legal basis %s should have reg: prefix", basis)
		}
	}

	// There should be exactly 6 legal bases (GDPR Article 6)
	if len(bases) != 6 {
		t.Errorf("Expected 6 legal bases (GDPR Art 6), got %d", len(bases))
	}
}
