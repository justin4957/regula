package extract

import (
	"os"
	"testing"
)

func TestNewSemanticExtractor(t *testing.T) {
	extractor := NewSemanticExtractor()
	if extractor == nil {
		t.Fatal("NewSemanticExtractor returned nil")
	}
	if len(extractor.rightPatterns) == 0 {
		t.Error("No right patterns initialized")
	}
	if len(extractor.obligationPatterns) == 0 {
		t.Error("No obligation patterns initialized")
	}
	if len(extractor.entityPatterns) == 0 {
		t.Error("No entity patterns initialized")
	}
}

func TestSemanticExtractor_RightPatterns(t *testing.T) {
	extractor := NewSemanticExtractor()

	tests := []struct {
		text         string
		expectedType RightType
		description  string
	}{
		{
			text:         "The data subject shall have the right of access to personal data",
			expectedType: RightAccess,
			description:  "Right of access",
		},
		{
			text:         "The data subject shall have the right to obtain rectification",
			expectedType: RightRectification,
			description:  "Right to rectification",
		},
		{
			text:         "The data subject shall have the right to erasure of personal data",
			expectedType: RightErasure,
			description:  "Right to erasure",
		},
		{
			text:         "The right to be forgotten shall apply",
			expectedType: RightErasure,
			description:  "Right to be forgotten",
		},
		{
			text:         "The data subject shall have the right to restriction of processing",
			expectedType: RightRestriction,
			description:  "Right to restriction",
		},
		{
			text:         "The data subject shall have the right to data portability",
			expectedType: RightPortability,
			description:  "Right to portability",
		},
		{
			text:         "The data subject shall have the right to object",
			expectedType: RightObject,
			description:  "Right to object",
		},
		{
			text:         "The data subject has the right to withdraw his or her consent at any time",
			expectedType: RightWithdrawConsent,
			description:  "Right to withdraw consent",
		},
		{
			text:         "Every data subject shall have the right to lodge a complaint with a supervisory authority",
			expectedType: RightLodgeComplaint,
			description:  "Right to lodge complaint",
		},
		{
			text:         "Every data subject shall have the right to an effective judicial remedy",
			expectedType: RightEffectiveRemedy,
			description:  "Right to effective remedy",
		},
		{
			text:         "Any person who has suffered damage shall have the right to receive compensation",
			expectedType: RightCompensation,
			description:  "Right to compensation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			annotations := extractor.extractFromText(tt.text, 1, 0, "")

			found := false
			for _, ann := range annotations {
				if ann.Type == SemanticRight && ann.RightType == tt.expectedType {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected to find right type %v in %q", tt.expectedType, tt.text)
			}
		})
	}
}

func TestSemanticExtractor_ObligationPatterns(t *testing.T) {
	extractor := NewSemanticExtractor()

	tests := []struct {
		text         string
		expectedType ObligationType
		description  string
	}{
		{
			text:         "Processing shall be lawful only if consent is given",
			expectedType: ObligationLawfulProcessing,
			description:  "Lawful processing",
		},
		{
			text:         "The controller shall be able to demonstrate that consent was given",
			expectedType: ObligationConsent,
			description:  "Consent demonstration",
		},
		{
			text:         "The controller shall notify the personal data breach to the supervisory authority",
			expectedType: ObligationNotifyBreach,
			description:  "Breach notification",
		},
		{
			text:         "The controller shall implement appropriate technical and organisational security measures",
			expectedType: ObligationSecure,
			description:  "Security measures",
		},
		{
			text:         "Each controller shall maintain a record of processing activities",
			expectedType: ObligationRecord,
			description:  "Record keeping",
		},
		{
			text:         "The controller shall carry out a data protection impact assessment",
			expectedType: ObligationImpactAssessment,
			description:  "Impact assessment",
		},
		{
			text:         "The controller and processor shall cooperate with the supervisory authority",
			expectedType: ObligationCooperate,
			description:  "Cooperation",
		},
		{
			text:         "The controller shall designate a data protection officer",
			expectedType: ObligationAppoint,
			description:  "DPO appointment",
		},
		{
			text:         "The controller shall provide the following information",
			expectedType: ObligationProvideInformation,
			description:  "Information provision",
		},
		{
			text:         "The controller shall ensure that appropriate measures are in place",
			expectedType: ObligationEnsure,
			description:  "Ensure obligation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			annotations := extractor.extractFromText(tt.text, 1, 0, "")

			found := false
			for _, ann := range annotations {
				if (ann.Type == SemanticObligation || ann.Type == SemanticProhibition) && ann.ObligationType == tt.expectedType {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected to find obligation type %v in %q", tt.expectedType, tt.text)
			}
		})
	}
}

func TestSemanticExtractor_EntityIdentification(t *testing.T) {
	extractor := NewSemanticExtractor()

	tests := []struct {
		text           string
		expectedEntity EntityType
		description    string
	}{
		{
			text:           "The data subject has the right to access personal data",
			expectedEntity: EntityDataSubject,
			description:    "Data subject identification",
		},
		{
			text:           "The controller shall implement appropriate measures",
			expectedEntity: EntityController,
			description:    "Controller identification",
		},
		{
			text:           "The processor shall process data only on instructions",
			expectedEntity: EntityProcessor,
			description:    "Processor identification",
		},
		{
			text:           "The supervisory authority shall have the power to investigate",
			expectedEntity: EntitySupervisoryAuth,
			description:    "Supervisory authority identification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			entity := extractor.identifyEntity(tt.text)
			if entity != tt.expectedEntity {
				t.Errorf("Expected entity %v, got %v", tt.expectedEntity, entity)
			}
		})
	}
}

func TestSemanticExtractor_ExtractFromArticle(t *testing.T) {
	extractor := NewSemanticExtractor()

	article := &Article{
		Number: 17,
		Title:  "Right to erasure",
		Text:   "The data subject shall have the right to obtain from the controller the erasure of personal data concerning him or her without undue delay.",
		Paragraphs: []*Paragraph{
			{
				Number: 1,
				Text:   "The data subject shall have the right to obtain erasure",
			},
			{
				Number: 2,
				Text:   "The controller shall be obliged to erase personal data",
			},
		},
	}

	annotations := extractor.ExtractFromArticle(article)

	// Should find right to erasure
	foundErasure := false
	for _, ann := range annotations {
		if ann.RightType == RightErasure {
			foundErasure = true
			if ann.ArticleNum != 17 {
				t.Errorf("Article number = %d, want 17", ann.ArticleNum)
			}
			break
		}
	}

	if !foundErasure {
		t.Error("Expected to find right to erasure in Article 17")
	}
}

func TestCalculateSemanticStats(t *testing.T) {
	annotations := []*SemanticAnnotation{
		{Type: SemanticRight, RightType: RightAccess, Beneficiary: EntityDataSubject, ArticleNum: 15, Confidence: 1.0},
		{Type: SemanticRight, RightType: RightErasure, Beneficiary: EntityDataSubject, ArticleNum: 17, Confidence: 1.0},
		{Type: SemanticRight, RightType: RightPortability, Beneficiary: EntityDataSubject, ArticleNum: 20, Confidence: 0.9},
		{Type: SemanticObligation, ObligationType: ObligationSecure, DutyBearer: EntityController, ArticleNum: 32, Confidence: 1.0},
		{Type: SemanticObligation, ObligationType: ObligationNotifyBreach, DutyBearer: EntityController, ArticleNum: 33, Confidence: 0.8},
		{Type: SemanticProhibition, ObligationType: ObligationGeneric, DutyBearer: EntityController, ArticleNum: 9, Confidence: 0.7},
	}

	stats := CalculateSemanticStats(annotations)

	if stats.TotalAnnotations != 6 {
		t.Errorf("TotalAnnotations = %d, want 6", stats.TotalAnnotations)
	}
	if stats.Rights != 3 {
		t.Errorf("Rights = %d, want 3", stats.Rights)
	}
	if stats.Obligations != 2 {
		t.Errorf("Obligations = %d, want 2", stats.Obligations)
	}
	if stats.Prohibitions != 1 {
		t.Errorf("Prohibitions = %d, want 1", stats.Prohibitions)
	}
	if stats.ArticlesWithRights != 3 {
		t.Errorf("ArticlesWithRights = %d, want 3", stats.ArticlesWithRights)
	}
	if stats.ArticlesWithObligations != 3 {
		t.Errorf("ArticlesWithObligations = %d, want 3", stats.ArticlesWithObligations)
	}
	if stats.HighConfidence != 4 {
		t.Errorf("HighConfidence = %d, want 4", stats.HighConfidence)
	}
}

func TestSemanticLookup(t *testing.T) {
	annotations := []*SemanticAnnotation{
		{Type: SemanticRight, RightType: RightAccess, ArticleNum: 15},
		{Type: SemanticRight, RightType: RightErasure, ArticleNum: 17},
		{Type: SemanticObligation, ObligationType: ObligationSecure, ArticleNum: 32},
	}

	lookup := NewSemanticLookup(annotations)

	if lookup.RightsCount() != 2 {
		t.Errorf("RightsCount = %d, want 2", lookup.RightsCount())
	}
	if lookup.ObligationsCount() != 1 {
		t.Errorf("ObligationsCount = %d, want 1", lookup.ObligationsCount())
	}

	// Test GetByArticle
	art17 := lookup.GetByArticle(17)
	if len(art17) != 1 {
		t.Errorf("GetByArticle(17) returned %d annotations, want 1", len(art17))
	}

	// Test GetByRightType
	erasure := lookup.GetByRightType(RightErasure)
	if len(erasure) != 1 {
		t.Errorf("GetByRightType(RightErasure) returned %d annotations, want 1", len(erasure))
	}
}

// Integration test with GDPR data
func TestGDPRSemanticExtraction(t *testing.T) {
	gdprPath := "../../testdata/gdpr.txt"
	if _, err := os.Stat(gdprPath); os.IsNotExist(err) {
		gdprPath = "testdata/gdpr.txt"
		if _, err := os.Stat(gdprPath); os.IsNotExist(err) {
			t.Skip("GDPR test data not available")
		}
	}

	file, err := os.Open(gdprPath)
	if err != nil {
		t.Fatalf("Failed to open GDPR: %v", err)
	}
	defer file.Close()

	parser := NewParser()
	doc, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Failed to parse GDPR: %v", err)
	}

	extractor := NewSemanticExtractor()
	annotations := extractor.ExtractFromDocument(doc)
	stats := CalculateSemanticStats(annotations)

	t.Logf("Semantic Extraction Statistics:")
	t.Logf("  Total annotations: %d", stats.TotalAnnotations)
	t.Logf("  Rights: %d", stats.Rights)
	t.Logf("  Obligations: %d", stats.Obligations)
	t.Logf("  Prohibitions: %d", stats.Prohibitions)
	t.Logf("  Articles with rights: %d", stats.ArticlesWithRights)
	t.Logf("  Articles with obligations: %d", stats.ArticlesWithObligations)
	t.Logf("  High confidence: %d", stats.HighConfidence)
	t.Logf("  Medium confidence: %d", stats.MediumConfidence)
	t.Logf("  Low confidence: %d", stats.LowConfidence)

	// Create lookup for validation
	lookup := NewSemanticLookup(annotations)

	// Validate known GDPR rights
	knownRights := map[int]RightType{
		15: RightAccess,       // Art 15 - Right of access
		16: RightRectification, // Art 16 - Right to rectification
		17: RightErasure,      // Art 17 - Right to erasure
		18: RightRestriction,  // Art 18 - Right to restriction
		20: RightPortability,  // Art 20 - Right to data portability
		21: RightObject,       // Art 21 - Right to object
	}

	t.Logf("\nKnown GDPR Rights Validation:")
	foundRights := 0
	for artNum, expectedRight := range knownRights {
		found := false
		for _, ann := range lookup.GetByArticle(artNum) {
			if ann.Type == SemanticRight && ann.RightType == expectedRight {
				found = true
				break
			}
		}
		if found {
			foundRights++
			t.Logf("  [PASS] Article %d: %s", artNum, expectedRight)
		} else {
			t.Logf("  [FAIL] Article %d: %s not found", artNum, expectedRight)
		}
	}

	t.Logf("\nKnown Rights Found: %d/%d (%.0f%%)", foundRights, len(knownRights), float64(foundRights)/float64(len(knownRights))*100)

	// Should find the major rights
	if foundRights < 4 {
		t.Errorf("Expected to find at least 4 of 6 known rights, found %d", foundRights)
	}

	// Log rights by type
	t.Logf("\nRights by Type:")
	for rightType, count := range stats.ByRightType {
		t.Logf("  %s: %d", rightType, count)
	}

	// Log obligations by type
	t.Logf("\nObligations by Type:")
	for obligType, count := range stats.ByObligationType {
		t.Logf("  %s: %d", obligType, count)
	}

	// Sample output for manual validation
	t.Logf("\nSample Rights (first 10):")
	rights := lookup.GetRights()
	for i, right := range rights {
		if i >= 10 {
			break
		}
		t.Logf("  Article %d: %s - %q", right.ArticleNum, right.RightType, right.MatchedText)
	}

	t.Logf("\nSample Obligations (first 10):")
	obligations := lookup.GetObligations()
	for i, obl := range obligations {
		if i >= 10 {
			break
		}
		t.Logf("  Article %d: %s - %q", obl.ArticleNum, obl.ObligationType, obl.MatchedText)
	}
}

func TestSemanticExtractor_Confidence(t *testing.T) {
	extractor := NewSemanticExtractor()

	// High confidence pattern
	highConfText := "The data subject shall have the right of access"
	highAnnotations := extractor.extractFromText(highConfText, 1, 0, "")
	if len(highAnnotations) == 0 {
		t.Fatal("Expected annotation for high confidence text")
	}
	if highAnnotations[0].Confidence < 0.9 {
		t.Errorf("Expected high confidence (>=0.9), got %f", highAnnotations[0].Confidence)
	}

	// Medium confidence pattern
	medConfText := "The entity is entitled to compensation"
	medAnnotations := extractor.extractFromText(medConfText, 1, 0, "")
	if len(medAnnotations) == 0 {
		t.Fatal("Expected annotation for medium confidence text")
	}
	if medAnnotations[0].Confidence >= 0.9 || medAnnotations[0].Confidence < 0.6 {
		t.Errorf("Expected medium confidence (0.6-0.9), got %f", medAnnotations[0].Confidence)
	}
}

func TestExtractContext(t *testing.T) {
	text := "This is a long text that contains a right to erasure in the middle of the sentence."
	start := 39 // "right"
	end := 55   // "erasure"

	context := extractContext(text, start, end, 10)

	// Should contain the matched text
	if !contains(context, "right") {
		t.Error("Context should contain 'right'")
	}

	// Should be truncated with ellipsis
	if !contains(context, "...") {
		t.Error("Context should contain ellipsis for truncation")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func BenchmarkSemanticExtractor_ExtractFromDocument(b *testing.B) {
	gdprPath := "../../testdata/gdpr.txt"
	if _, err := os.Stat(gdprPath); os.IsNotExist(err) {
		gdprPath = "testdata/gdpr.txt"
		if _, err := os.Stat(gdprPath); os.IsNotExist(err) {
			b.Skip("GDPR test data not available")
		}
	}

	file, err := os.Open(gdprPath)
	if err != nil {
		b.Fatalf("Failed to open GDPR: %v", err)
	}
	defer file.Close()

	parser := NewParser()
	doc, err := parser.Parse(file)
	if err != nil {
		b.Fatalf("Failed to parse GDPR: %v", err)
	}

	extractor := NewSemanticExtractor()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractor.ExtractFromDocument(doc)
	}
}
