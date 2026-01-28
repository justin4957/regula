package validate

import (
	"time"
)

// CoverageGate (V2) validates extraction coverage and completeness.
// Runs after definition, reference, and semantic extraction.
type CoverageGate struct{}

// NewCoverageGate creates a new V2 coverage validation gate.
func NewCoverageGate() *CoverageGate {
	return &CoverageGate{}
}

// Name returns "V2".
func (coverageGate *CoverageGate) Name() string { return "V2" }

// Thresholds returns the default thresholds for coverage validation metrics.
func (coverageGate *CoverageGate) Thresholds() map[string]float64 {
	return map[string]float64{
		"definition_coverage": 0.50,
		"reference_density":   0.50,
		"semantic_coverage":   0.30,
	}
}

// Run validates that extraction produced adequate definitions, references, and semantics.
func (coverageGate *CoverageGate) Run(ctx *ValidationContext) *GateResult {
	startTime := time.Now()

	gateResult := &GateResult{
		Gate:     coverageGate.Name(),
		Metrics:  make(map[string]float64),
		Warnings: make([]GateWarning, 0),
		Errors:   make([]GateError, 0),
	}

	// Count articles for normalization.
	articleCount := 0
	if ctx.Document != nil {
		articleCount = len(CollectArticles(ctx.Document))
	}

	// definition_coverage: normalized definition count relative to articles.
	// A ratio of definitions/articles >= 0.2 maps to 1.0 (well-defined regulation).
	definitionCount := len(ctx.Definitions)
	if articleCount > 0 && definitionCount > 0 {
		normalizedDefinitionCoverage := float64(definitionCount) / float64(articleCount)
		// Cap at 1.0: having 20%+ definitions per article is full coverage.
		if normalizedDefinitionCoverage > 1.0 {
			normalizedDefinitionCoverage = 1.0
		}
		// Scale: 0.2 ratio → 1.0 score (multiply by 5, cap at 1.0).
		scaledCoverage := normalizedDefinitionCoverage * 5.0
		if scaledCoverage > 1.0 {
			scaledCoverage = 1.0
		}
		gateResult.Metrics["definition_coverage"] = scaledCoverage
	} else {
		gateResult.Metrics["definition_coverage"] = 0.0
	}

	// reference_density: normalized reference count relative to articles.
	// A ratio of references/articles >= 2.0 maps to 1.0 (well-referenced regulation).
	referenceCount := len(ctx.References)
	if articleCount > 0 && referenceCount > 0 {
		normalizedReferenceDensity := float64(referenceCount) / float64(articleCount)
		// Scale: 2.0 ratio → 1.0 score (divide by 2, cap at 1.0).
		scaledDensity := normalizedReferenceDensity / 2.0
		if scaledDensity > 1.0 {
			scaledDensity = 1.0
		}
		gateResult.Metrics["reference_density"] = scaledDensity
	} else {
		gateResult.Metrics["reference_density"] = 0.0
	}

	// semantic_coverage: fraction of articles with rights or obligations.
	if articleCount > 0 && len(ctx.Semantics) > 0 {
		articlesWithSemantics := make(map[int]bool)
		for _, annotation := range ctx.Semantics {
			articlesWithSemantics[annotation.ArticleNum] = true
		}
		gateResult.Metrics["semantic_coverage"] = float64(len(articlesWithSemantics)) / float64(articleCount)
	} else {
		gateResult.Metrics["semantic_coverage"] = 0.0
	}

	evaluateMetrics(gateResult, ctx.Config, coverageGate)
	gateResult.Duration = time.Since(startTime)
	return gateResult
}
