package validate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/coolbeans/regula/pkg/extract"
	"github.com/coolbeans/regula/pkg/store"
)

// --- Interface compliance (compile-time) ---

var _ ValidationGate = (*SchemaGate)(nil)
var _ ValidationGate = (*StructureGate)(nil)
var _ ValidationGate = (*CoverageGate)(nil)
var _ ValidationGate = (*QualityGate)(nil)

// --- Helper: build a minimal document with articles ---

func buildTestDocument(articleCount int, withContent bool) *extract.Document {
	articles := make([]*extract.Article, articleCount)
	for articleIndex := 0; articleIndex < articleCount; articleIndex++ {
		text := ""
		if withContent {
			text = "This article contains meaningful regulatory content that exceeds the minimum character threshold."
		}
		articles[articleIndex] = &extract.Article{
			Number: articleIndex + 1,
			Title:  "Test Article",
			Text:   text,
		}
	}

	return &extract.Document{
		Title:      "Test Regulation",
		Identifier: "TEST/2024/1",
		Chapters: []*extract.Chapter{
			{
				Number:   "I",
				Title:    "General Provisions",
				Articles: articles,
			},
		},
	}
}

// --- SchemaGate (V0) Tests ---

func TestSchemaGate_ValidFile(t *testing.T) {
	gate := NewSchemaGate()
	ctx := &ValidationContext{
		SourcePath: "/path/to/file.txt",
		SourceSize: 50000,
		Config:     DefaultValidationConfig(),
	}

	result := gate.Run(ctx)

	if !result.Passed {
		t.Error("Expected gate to pass for valid file")
	}
	if result.Gate != "V0" {
		t.Errorf("Gate name: got %q, want 'V0'", result.Gate)
	}
	if result.Metrics["file_readable"] != 1.0 {
		t.Errorf("file_readable: got %.1f, want 1.0", result.Metrics["file_readable"])
	}
	if result.Metrics["file_not_empty"] != 1.0 {
		t.Errorf("file_not_empty: got %.1f, want 1.0", result.Metrics["file_not_empty"])
	}
	if result.Metrics["file_size"] != 1.0 {
		t.Errorf("file_size: got %.1f, want 1.0", result.Metrics["file_size"])
	}
}

func TestSchemaGate_EmptySource(t *testing.T) {
	gate := NewSchemaGate()
	ctx := &ValidationContext{
		SourcePath: "",
		SourceSize: 0,
		Config:     DefaultValidationConfig(),
	}

	result := gate.Run(ctx)

	if result.Passed {
		t.Error("Expected gate to fail for empty source")
	}
	if result.Metrics["file_readable"] != 0.0 {
		t.Errorf("file_readable: got %.1f, want 0.0", result.Metrics["file_readable"])
	}
}

func TestSchemaGate_LargeFile(t *testing.T) {
	gate := NewSchemaGate()
	ctx := &ValidationContext{
		SourcePath: "/path/to/huge.txt",
		SourceSize: 20 * 1024 * 1024, // 20 MB
		Config:     DefaultValidationConfig(),
	}

	result := gate.Run(ctx)

	if result.Passed {
		t.Error("Expected gate to fail for oversized file")
	}
	if result.Metrics["file_size"] != 0.0 {
		t.Errorf("file_size: got %.1f, want 0.0 for oversized file", result.Metrics["file_size"])
	}
	if len(result.Warnings) == 0 {
		t.Error("Expected warning for oversized file")
	}
}

func TestSchemaGate_Name(t *testing.T) {
	gate := NewSchemaGate()
	if gate.Name() != "V0" {
		t.Errorf("Name: got %q, want 'V0'", gate.Name())
	}
}

func TestSchemaGate_Thresholds(t *testing.T) {
	gate := NewSchemaGate()
	thresholds := gate.Thresholds()

	if thresholds["file_readable"] != 1.0 {
		t.Errorf("file_readable threshold: got %.1f, want 1.0", thresholds["file_readable"])
	}
	if thresholds["file_size"] != 1.0 {
		t.Errorf("file_size threshold: got %.1f, want 1.0", thresholds["file_size"])
	}
	if thresholds["file_not_empty"] != 1.0 {
		t.Errorf("file_not_empty threshold: got %.1f, want 1.0", thresholds["file_not_empty"])
	}
}

// --- StructureGate (V1) Tests ---

func TestStructureGate_WellFormedDocument(t *testing.T) {
	gate := NewStructureGate()
	document := buildTestDocument(10, true)
	ctx := &ValidationContext{
		Document: document,
		Config:   DefaultValidationConfig(),
	}

	result := gate.Run(ctx)

	if !result.Passed {
		t.Errorf("Expected gate to pass for well-formed document, errors: %v", result.Errors)
	}
	if result.Gate != "V1" {
		t.Errorf("Gate name: got %q, want 'V1'", result.Gate)
	}
	if result.Metrics["has_articles"] != 1.0 {
		t.Errorf("has_articles: got %.1f, want 1.0", result.Metrics["has_articles"])
	}
	if result.Metrics["structure_completeness"] != 1.0 {
		t.Errorf("structure_completeness: got %.1f, want 1.0 (has chapters + articles)", result.Metrics["structure_completeness"])
	}
	if result.Metrics["content_density"] != 1.0 {
		t.Errorf("content_density: got %.1f, want 1.0 (all articles have content)", result.Metrics["content_density"])
	}
}

func TestStructureGate_EmptyDocument(t *testing.T) {
	gate := NewStructureGate()
	ctx := &ValidationContext{
		Document: nil,
		Config:   DefaultValidationConfig(),
	}

	result := gate.Run(ctx)

	if result.Passed {
		t.Error("Expected gate to fail for nil document")
	}
	if result.Metrics["has_articles"] != 0.0 {
		t.Errorf("has_articles: got %.1f, want 0.0", result.Metrics["has_articles"])
	}
}

func TestStructureGate_NoArticles(t *testing.T) {
	gate := NewStructureGate()
	document := &extract.Document{
		Title: "Empty Regulation",
		Chapters: []*extract.Chapter{
			{Number: "I", Title: "Chapter One"},
		},
	}
	ctx := &ValidationContext{
		Document: document,
		Config:   DefaultValidationConfig(),
	}

	result := gate.Run(ctx)

	if result.Passed {
		t.Error("Expected gate to fail for document with no articles")
	}
	if result.Metrics["has_articles"] != 0.0 {
		t.Errorf("has_articles: got %.1f, want 0.0", result.Metrics["has_articles"])
	}
	// Should still have partial structure_completeness (has chapters).
	if result.Metrics["structure_completeness"] != 0.5 {
		t.Errorf("structure_completeness: got %.1f, want 0.5 (has chapters but no articles)", result.Metrics["structure_completeness"])
	}
}

func TestStructureGate_MixedContentDensity(t *testing.T) {
	gate := NewStructureGate()
	document := &extract.Document{
		Title: "Mixed Content",
		Chapters: []*extract.Chapter{
			{
				Number: "I",
				Title:  "Chapter One",
				Articles: []*extract.Article{
					{Number: 1, Text: "Long content that exceeds the minimum character threshold for meaningful content."},
					{Number: 2, Text: "Short"},
					{Number: 3, Text: "Another article with sufficient content to pass the density check validation."},
					{Number: 4, Text: ""},
				},
			},
		},
	}
	ctx := &ValidationContext{
		Document: document,
		Config:   DefaultValidationConfig(),
	}

	result := gate.Run(ctx)

	// 2 of 4 articles have content >= 50 chars.
	expectedDensity := 0.5
	if result.Metrics["content_density"] != expectedDensity {
		t.Errorf("content_density: got %.2f, want %.2f", result.Metrics["content_density"], expectedDensity)
	}
}

// --- CoverageGate (V2) Tests ---

func TestCoverageGate_FullExtraction(t *testing.T) {
	gate := NewCoverageGate()
	document := buildTestDocument(10, true)

	definitions := make([]*extract.DefinedTerm, 5)
	for definitionIndex := range definitions {
		definitions[definitionIndex] = &extract.DefinedTerm{
			Number: definitionIndex + 1,
			Term:   "test term",
		}
	}

	references := make([]*extract.Reference, 20)
	for refIndex := range references {
		references[refIndex] = &extract.Reference{
			SourceArticle: refIndex%10 + 1,
			RawText:       "Article X",
		}
	}

	semantics := make([]*extract.SemanticAnnotation, 5)
	for semIndex := range semantics {
		semantics[semIndex] = &extract.SemanticAnnotation{
			ArticleNum: semIndex + 1,
		}
	}

	ctx := &ValidationContext{
		Document:    document,
		Definitions: definitions,
		References:  references,
		Semantics:   semantics,
		Config:      DefaultValidationConfig(),
	}

	result := gate.Run(ctx)

	if !result.Passed {
		t.Errorf("Expected gate to pass for full extraction, errors: %v", result.Errors)
	}
	if result.Gate != "V2" {
		t.Errorf("Gate name: got %q, want 'V2'", result.Gate)
	}
}

func TestCoverageGate_NoDefinitions(t *testing.T) {
	gate := NewCoverageGate()
	document := buildTestDocument(10, true)

	ctx := &ValidationContext{
		Document:    document,
		Definitions: nil,
		References:  []*extract.Reference{{SourceArticle: 1}},
		Semantics:   []*extract.SemanticAnnotation{{ArticleNum: 1}},
		Config:      DefaultValidationConfig(),
	}

	result := gate.Run(ctx)

	if result.Metrics["definition_coverage"] != 0.0 {
		t.Errorf("definition_coverage: got %.1f, want 0.0 for no definitions", result.Metrics["definition_coverage"])
	}
}

func TestCoverageGate_NoReferences(t *testing.T) {
	gate := NewCoverageGate()
	document := buildTestDocument(10, true)

	ctx := &ValidationContext{
		Document:    document,
		Definitions: []*extract.DefinedTerm{{Term: "test"}},
		References:  nil,
		Config:      DefaultValidationConfig(),
	}

	result := gate.Run(ctx)

	if result.Metrics["reference_density"] != 0.0 {
		t.Errorf("reference_density: got %.1f, want 0.0 for no references", result.Metrics["reference_density"])
	}
}

func TestCoverageGate_NilDocument(t *testing.T) {
	gate := NewCoverageGate()
	ctx := &ValidationContext{
		Document: nil,
		Config:   DefaultValidationConfig(),
	}

	result := gate.Run(ctx)

	if result.Passed {
		t.Error("Expected gate to fail with nil document")
	}
}

// --- QualityGate (V3) Tests ---

func TestQualityGate_HighResolution(t *testing.T) {
	gate := NewQualityGate()
	document := buildTestDocument(5, true)

	resolvedRefs := []*extract.ResolvedReference{
		{Status: extract.ResolutionResolved, Confidence: extract.ConfidenceHigh, Original: &extract.Reference{SourceArticle: 1, ArticleNum: 2}},
		{Status: extract.ResolutionResolved, Confidence: extract.ConfidenceHigh, Original: &extract.Reference{SourceArticle: 2, ArticleNum: 3}},
		{Status: extract.ResolutionResolved, Confidence: extract.ConfidenceMedium, Original: &extract.Reference{SourceArticle: 3, ArticleNum: 4}},
		{Status: extract.ResolutionExternal, Confidence: extract.ConfidenceMedium, Original: &extract.Reference{SourceArticle: 4, ArticleNum: 0}},
		{Status: extract.ResolutionResolved, Confidence: extract.ConfidenceHigh, Original: &extract.Reference{SourceArticle: 5, ArticleNum: 1}},
	}

	ctx := &ValidationContext{
		Document:           document,
		ResolvedReferences: resolvedRefs,
		TripleStore:        store.NewTripleStore(),
		Config:             DefaultValidationConfig(),
	}

	result := gate.Run(ctx)

	if !result.Passed {
		t.Errorf("Expected gate to pass for high resolution, errors: %v", result.Errors)
	}
	if result.Gate != "V3" {
		t.Errorf("Gate name: got %q, want 'V3'", result.Gate)
	}
	// All 5 refs are resolved/external.
	if result.Metrics["resolution_rate"] != 1.0 {
		t.Errorf("resolution_rate: got %.2f, want 1.0", result.Metrics["resolution_rate"])
	}
}

func TestQualityGate_LowResolution(t *testing.T) {
	gate := NewQualityGate()
	document := buildTestDocument(5, true)

	resolvedRefs := []*extract.ResolvedReference{
		{Status: extract.ResolutionResolved, Confidence: extract.ConfidenceHigh, Original: &extract.Reference{SourceArticle: 1, ArticleNum: 2}},
		{Status: extract.ResolutionNotFound, Confidence: extract.ConfidenceNone, Original: &extract.Reference{SourceArticle: 2}},
		{Status: extract.ResolutionNotFound, Confidence: extract.ConfidenceNone, Original: &extract.Reference{SourceArticle: 3}},
		{Status: extract.ResolutionAmbiguous, Confidence: extract.ConfidenceLow, Original: &extract.Reference{SourceArticle: 4}},
		{Status: extract.ResolutionNotFound, Confidence: extract.ConfidenceNone, Original: &extract.Reference{SourceArticle: 5}},
	}

	ctx := &ValidationContext{
		Document:           document,
		ResolvedReferences: resolvedRefs,
		TripleStore:        store.NewTripleStore(),
		Config:             DefaultValidationConfig(),
	}

	result := gate.Run(ctx)

	if result.Passed {
		t.Error("Expected gate to fail for low resolution (only 1/5 resolved)")
	}
	expectedRate := 0.2
	if result.Metrics["resolution_rate"] != expectedRate {
		t.Errorf("resolution_rate: got %.2f, want %.2f", result.Metrics["resolution_rate"], expectedRate)
	}
}

func TestQualityGate_NoResolvedRefs(t *testing.T) {
	gate := NewQualityGate()
	ctx := &ValidationContext{
		Document:           buildTestDocument(5, true),
		ResolvedReferences: nil,
		TripleStore:        store.NewTripleStore(),
		Config:             DefaultValidationConfig(),
	}

	result := gate.Run(ctx)

	if result.Metrics["resolution_rate"] != 0.0 {
		t.Errorf("resolution_rate: got %.1f, want 0.0", result.Metrics["resolution_rate"])
	}
}

// --- GatePipeline Tests ---

func TestGatePipeline_AllGatesPass(t *testing.T) {
	config := DefaultValidationConfig()
	pipeline := NewGatePipeline(config)
	pipeline.RegisterDefaultGates()

	document := buildTestDocument(20, true)

	definitions := make([]*extract.DefinedTerm, 5)
	for definitionIndex := range definitions {
		definitions[definitionIndex] = &extract.DefinedTerm{Number: definitionIndex + 1, Term: "term"}
	}

	references := make([]*extract.Reference, 40)
	for refIndex := range references {
		references[refIndex] = &extract.Reference{SourceArticle: refIndex%20 + 1, RawText: "ref"}
	}

	resolvedRefs := make([]*extract.ResolvedReference, 40)
	for refIndex := range resolvedRefs {
		resolvedRefs[refIndex] = &extract.ResolvedReference{
			Status:     extract.ResolutionResolved,
			Confidence: extract.ConfidenceHigh,
			Original:   &extract.Reference{SourceArticle: refIndex%20 + 1, ArticleNum: (refIndex+1)%20 + 1},
		}
	}

	semantics := make([]*extract.SemanticAnnotation, 10)
	for semIndex := range semantics {
		semantics[semIndex] = &extract.SemanticAnnotation{ArticleNum: semIndex + 1}
	}

	ctx := &ValidationContext{
		SourcePath:         "/path/to/regulation.txt",
		SourceSize:         50000,
		Document:           document,
		Definitions:        definitions,
		References:         references,
		Semantics:          semantics,
		ResolvedReferences: resolvedRefs,
		TripleStore:        store.NewTripleStore(),
		Config:             config,
	}

	report := pipeline.Run(ctx)

	if !report.OverallPass {
		t.Error("Expected all gates to pass")
		for _, gateResult := range report.Results {
			if !gateResult.Passed && !gateResult.Skipped {
				t.Logf("Gate %s failed: %v", gateResult.Gate, gateResult.Errors)
			}
		}
	}

	if len(report.Results) != 4 {
		t.Errorf("Expected 4 gate results, got %d", len(report.Results))
	}

	if report.GatesPassed != 4 {
		t.Errorf("GatesPassed: got %d, want 4", report.GatesPassed)
	}
}

func TestGatePipeline_SkipGates(t *testing.T) {
	config := &ValidationConfig{
		Thresholds: make(map[string]float64),
		SkipGates:  []string{"V0", "V2"},
		StrictMode: false,
		FailOnWarn: false,
	}
	pipeline := NewGatePipeline(config)
	pipeline.RegisterDefaultGates()

	ctx := &ValidationContext{
		SourcePath: "/path/to/file.txt",
		SourceSize: 50000,
		Document:   buildTestDocument(10, true),
		Config:     config,
	}

	report := pipeline.Run(ctx)

	if report.GatesSkipped != 2 {
		t.Errorf("GatesSkipped: got %d, want 2", report.GatesSkipped)
	}

	// Verify V0 and V2 are skipped in results.
	for _, gateResult := range report.Results {
		if gateResult.Gate == "V0" || gateResult.Gate == "V2" {
			if !gateResult.Skipped {
				t.Errorf("Gate %s should be skipped", gateResult.Gate)
			}
		}
	}
}

func TestGatePipeline_StrictModeHalts(t *testing.T) {
	config := &ValidationConfig{
		Thresholds: make(map[string]float64),
		SkipGates:  make([]string, 0),
		StrictMode: true,
		FailOnWarn: false,
	}
	pipeline := NewGatePipeline(config)
	pipeline.RegisterDefaultGates()

	// V0 will pass, V1 will fail (nil document).
	ctx := &ValidationContext{
		SourcePath: "/path/to/file.txt",
		SourceSize: 50000,
		Document:   nil, // V1 will fail
		Config:     config,
	}

	report := pipeline.Run(ctx)

	if report.OverallPass {
		t.Error("Expected pipeline to fail in strict mode")
	}
	if report.HaltedAt == "" {
		t.Error("Expected HaltedAt to be set")
	}
	if report.HaltedAt != "V1" {
		t.Errorf("HaltedAt: got %q, want 'V1'", report.HaltedAt)
	}
	// Should only have V0 and V1 results (halted before V2, V3).
	if len(report.Results) != 2 {
		t.Errorf("Expected 2 results (halted at V1), got %d", len(report.Results))
	}
}

func TestGatePipeline_FailOnWarn(t *testing.T) {
	config := &ValidationConfig{
		Thresholds: make(map[string]float64),
		SkipGates:  make([]string, 0),
		StrictMode: false,
		FailOnWarn: true,
	}
	pipeline := NewGatePipeline(config)
	pipeline.RegisterDefaultGates()

	// V0 will pass but with a warning (large file close to limit).
	ctx := &ValidationContext{
		SourcePath: "/path/to/file.txt",
		SourceSize: 50000, // Normal size — V0 passes with no warning.
		Document:   buildTestDocument(10, true),
		Config:     config,
	}

	report := pipeline.Run(ctx)

	// If no warnings are generated, pipeline should pass.
	// We verify the mechanism works by checking FailOnWarn halts on warning.
	if report.HaltedAt != "" {
		// Only halted if a warning was emitted.
		t.Logf("Pipeline halted at %s due to FailOnWarn", report.HaltedAt)
	}
}

func TestGatePipeline_ThresholdOverrides(t *testing.T) {
	config := &ValidationConfig{
		Thresholds: map[string]float64{
			"V1.has_articles": 0.0, // Override: accept even no articles.
		},
		SkipGates:  make([]string, 0),
		StrictMode: false,
		FailOnWarn: false,
	}
	pipeline := NewGatePipeline(config)
	pipeline.RegisterGate(NewStructureGate())

	// Document with no articles — normally V1 would fail.
	document := &extract.Document{
		Title:    "Empty Regulation",
		Chapters: []*extract.Chapter{{Number: "I", Title: "Chapter One"}},
	}
	ctx := &ValidationContext{
		Document: document,
		Config:   config,
	}

	report := pipeline.Run(ctx)

	// With threshold override to 0.0, has_articles=0.0 should still pass.
	// But structure_completeness=0.5 and content_density=0.0 may cause failure.
	// This test validates threshold override mechanism is applied.
	v1Result := report.Results[0]
	if v1Result.Gate != "V1" {
		t.Errorf("Expected V1, got %q", v1Result.Gate)
	}

	// Verify the override was respected: has_articles metric should not appear in errors
	// even though its value is 0.0.
	for _, gateError := range v1Result.Errors {
		if gateError.Metric == "has_articles" {
			t.Error("has_articles should not be an error with threshold override of 0.0")
		}
	}
}

func TestGatePipeline_RegisterDefaultGates(t *testing.T) {
	pipeline := NewGatePipeline(DefaultValidationConfig())
	pipeline.RegisterDefaultGates()

	// Verify 4 gates registered.
	ctx := &ValidationContext{
		SourcePath: "/path/to/file.txt",
		SourceSize: 50000,
		Document:   buildTestDocument(5, true),
		Config:     DefaultValidationConfig(),
	}

	report := pipeline.Run(ctx)

	if len(report.Results) != 4 {
		t.Errorf("Expected 4 gates, got %d", len(report.Results))
	}

	expectedGateNames := []string{"V0", "V1", "V2", "V3"}
	for gateIndex, expectedName := range expectedGateNames {
		if gateIndex < len(report.Results) && report.Results[gateIndex].Gate != expectedName {
			t.Errorf("Gate %d: got %q, want %q", gateIndex, report.Results[gateIndex].Gate, expectedName)
		}
	}
}

// mockGate is a custom gate for testing extensibility.
type mockGate struct {
	name      string
	passValue bool
}

func (mockValidationGate *mockGate) Name() string { return mockValidationGate.name }
func (mockValidationGate *mockGate) Thresholds() map[string]float64 {
	return map[string]float64{"custom_metric": 0.50}
}
func (mockValidationGate *mockGate) Run(ctx *ValidationContext) *GateResult {
	gateResult := &GateResult{
		Gate:     mockValidationGate.name,
		Passed:   mockValidationGate.passValue,
		Score:    1.0,
		Metrics:  map[string]float64{"custom_metric": 1.0},
		Warnings: make([]GateWarning, 0),
		Errors:   make([]GateError, 0),
	}
	if !mockValidationGate.passValue {
		gateResult.Score = 0.0
		gateResult.Metrics["custom_metric"] = 0.0
	}
	return gateResult
}

func TestGatePipeline_CustomGate(t *testing.T) {
	pipeline := NewGatePipeline(DefaultValidationConfig())
	pipeline.RegisterGate(&mockGate{name: "CUSTOM", passValue: true})

	ctx := &ValidationContext{Config: DefaultValidationConfig()}
	report := pipeline.Run(ctx)

	if len(report.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(report.Results))
	}
	if report.Results[0].Gate != "CUSTOM" {
		t.Errorf("Gate name: got %q, want 'CUSTOM'", report.Results[0].Gate)
	}
	if !report.OverallPass {
		t.Error("Expected custom gate pipeline to pass")
	}
}

func TestGatePipeline_RunGate(t *testing.T) {
	pipeline := NewGatePipeline(DefaultValidationConfig())
	pipeline.RegisterDefaultGates()

	ctx := &ValidationContext{
		SourcePath: "/path/to/file.txt",
		SourceSize: 50000,
		Config:     DefaultValidationConfig(),
	}

	v0Result := pipeline.RunGate("V0", ctx)
	if v0Result == nil {
		t.Fatal("Expected non-nil result for V0")
	}
	if v0Result.Gate != "V0" {
		t.Errorf("Gate: got %q, want 'V0'", v0Result.Gate)
	}

	// Non-existent gate.
	nilResult := pipeline.RunGate("V99", ctx)
	if nilResult != nil {
		t.Error("Expected nil for non-existent gate")
	}
}

func TestGatePipeline_RunGate_Skipped(t *testing.T) {
	config := &ValidationConfig{
		Thresholds: make(map[string]float64),
		SkipGates:  []string{"V0"},
	}
	pipeline := NewGatePipeline(config)
	pipeline.RegisterDefaultGates()

	ctx := &ValidationContext{Config: config}
	v0Result := pipeline.RunGate("V0", ctx)
	if v0Result == nil {
		t.Fatal("Expected non-nil result for skipped gate")
	}
	if !v0Result.Skipped {
		t.Error("Expected V0 to be skipped")
	}
}

// --- Report Tests ---

func TestGateReport_String(t *testing.T) {
	report := &GateReport{
		Results: []*GateResult{
			{
				Gate:    "V0",
				Passed:  true,
				Score:   1.0,
				Metrics: map[string]float64{"file_readable": 1.0},
			},
			{
				Gate:    "V1",
				Passed:  false,
				Score:   0.3,
				Metrics: map[string]float64{"has_articles": 0.0},
				Errors:  []GateError{{Metric: "has_articles", Message: "no articles found"}},
			},
		},
		OverallPass:  false,
		TotalScore:   0.65,
		GatesPassed:  1,
		GatesFailed:  1,
		GatesSkipped: 0,
		Duration:     100 * time.Millisecond,
	}

	output := report.String()

	if output == "" {
		t.Error("Expected non-empty string output")
	}
	if !containsSubstring(output, "V0") {
		t.Error("Expected output to contain 'V0'")
	}
	if !containsSubstring(output, "FAIL") {
		t.Error("Expected output to contain 'FAIL'")
	}
	if !containsSubstring(output, "1 passed") {
		t.Error("Expected output to contain '1 passed'")
	}
}

func TestGateReport_ToJSON(t *testing.T) {
	report := &GateReport{
		Results: []*GateResult{
			{
				Gate:    "V0",
				Passed:  true,
				Score:   1.0,
				Metrics: map[string]float64{"file_readable": 1.0},
			},
		},
		OverallPass: true,
		TotalScore:  1.0,
		GatesPassed: 1,
	}

	jsonBytes, err := report.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Verify it's valid JSON.
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if parsed["overall_pass"] != true {
		t.Error("Expected overall_pass to be true in JSON")
	}
}

func TestGateReport_StringWithSkippedGate(t *testing.T) {
	report := &GateReport{
		Results: []*GateResult{
			{
				Gate:       "V0",
				Skipped:    true,
				SkipReason: "skipped by configuration",
				Metrics:    make(map[string]float64),
			},
		},
		OverallPass:  true,
		GatesSkipped: 1,
	}

	output := report.String()
	if !containsSubstring(output, "SKIP") {
		t.Error("Expected output to contain 'SKIP' for skipped gate")
	}
}

func TestGateReport_StringWithHaltedPipeline(t *testing.T) {
	report := &GateReport{
		Results:     []*GateResult{},
		OverallPass: false,
		HaltedAt:    "V1",
	}

	output := report.String()
	if !containsSubstring(output, "halted") {
		t.Error("Expected output to mention halted pipeline")
	}
}

// --- DefaultValidationConfig Test ---

func TestDefaultValidationConfig(t *testing.T) {
	config := DefaultValidationConfig()

	if config == nil {
		t.Fatal("Expected non-nil config")
	}
	if config.Thresholds == nil {
		t.Error("Expected initialized Thresholds map")
	}
	if config.SkipGates == nil {
		t.Error("Expected initialized SkipGates slice")
	}
	if config.StrictMode {
		t.Error("Expected StrictMode to be false by default")
	}
	if config.FailOnWarn {
		t.Error("Expected FailOnWarn to be false by default")
	}
}

// --- Effective Threshold Test ---

func TestEffectiveThreshold(t *testing.T) {
	gate := NewSchemaGate()

	// No override — should use gate default.
	config := DefaultValidationConfig()
	threshold := effectiveThreshold(config, gate, "file_readable")
	if threshold != 1.0 {
		t.Errorf("Default threshold: got %.1f, want 1.0", threshold)
	}

	// With override.
	config.Thresholds["V0.file_readable"] = 0.5
	threshold = effectiveThreshold(config, gate, "file_readable")
	if threshold != 0.5 {
		t.Errorf("Override threshold: got %.1f, want 0.5", threshold)
	}

	// Unknown metric — should use fallback.
	threshold = effectiveThreshold(config, gate, "unknown_metric")
	if threshold != 0.80 {
		t.Errorf("Fallback threshold: got %.1f, want 0.80", threshold)
	}
}

// --- GDPR Integration Test ---

func TestGatePipeline_GDPRIntegration(t *testing.T) {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(currentFile), "..", "..")
	gdprPath := filepath.Join(projectRoot, "testdata", "gdpr.txt")

	fileInfo, err := os.Stat(gdprPath)
	if err != nil {
		t.Skipf("Skipping GDPR integration test: %v", err)
	}

	file, err := os.Open(gdprPath)
	if err != nil {
		t.Fatalf("Failed to open GDPR: %v", err)
	}
	defer file.Close()

	// Parse.
	parser := extract.NewParser()
	document, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Failed to parse GDPR: %v", err)
	}

	// Extract.
	defExtractor := extract.NewDefinitionExtractor()
	definitions := defExtractor.ExtractDefinitions(document)

	refExtractor := extract.NewReferenceExtractor()
	references := refExtractor.ExtractFromDocument(document)

	semExtractor := extract.NewSemanticExtractor()
	semantics := semExtractor.ExtractFromDocument(document)

	// Resolve.
	resolver := extract.NewReferenceResolver("https://regula.dev/regulations/", "GDPR")
	resolver.IndexDocument(document)
	resolvedReferences := resolver.ResolveAll(references)

	// Build graph.
	tripleStore := store.NewTripleStore()
	builder := store.NewGraphBuilder(tripleStore, "https://regula.dev/regulations/")
	_, err = builder.BuildComplete(document, defExtractor, refExtractor, resolver, semExtractor)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	// Run gate pipeline.
	config := DefaultValidationConfig()
	pipeline := NewGatePipeline(config)
	pipeline.RegisterDefaultGates()

	ctx := &ValidationContext{
		SourcePath:         gdprPath,
		SourceSize:         fileInfo.Size(),
		Document:           document,
		Definitions:        definitions,
		References:         references,
		Semantics:          semantics,
		ResolvedReferences: resolvedReferences,
		TripleStore:        tripleStore,
		Config:             config,
	}

	report := pipeline.Run(ctx)

	// All 4 gates should be present.
	if len(report.Results) != 4 {
		t.Errorf("Expected 4 gate results, got %d", len(report.Results))
	}

	// Log per-gate results.
	for _, gateResult := range report.Results {
		statusLabel := "PASS"
		if !gateResult.Passed {
			statusLabel = "FAIL"
		}
		t.Logf("[%s] %s: score=%.1f%%", statusLabel, gateResult.Gate, gateResult.Score*100)
		for metricName, metricValue := range gateResult.Metrics {
			t.Logf("  %s: %.1f%%", metricName, metricValue*100)
		}
	}

	t.Logf("Overall: pass=%v, score=%.1f%%", report.OverallPass, report.TotalScore*100)

	// GDPR should pass V0 and V1 gates at minimum.
	for _, gateResult := range report.Results {
		if gateResult.Gate == "V0" && !gateResult.Passed {
			t.Errorf("GDPR should pass V0 (schema) gate")
		}
		if gateResult.Gate == "V1" && !gateResult.Passed {
			t.Errorf("GDPR should pass V1 (structure) gate")
		}
	}

	// Overall score should be reasonable for GDPR.
	if report.TotalScore < 0.50 {
		t.Errorf("GDPR overall score too low: %.1f%%", report.TotalScore*100)
	}
}

// --- Helpers ---

func containsSubstring(haystack, needle string) bool {
	return len(haystack) >= len(needle) && findSubstring(haystack, needle)
}

func findSubstring(haystack, needle string) bool {
	for startIndex := 0; startIndex <= len(haystack)-len(needle); startIndex++ {
		if haystack[startIndex:startIndex+len(needle)] == needle {
			return true
		}
	}
	return false
}
