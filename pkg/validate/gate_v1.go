package validate

import (
	"time"
)

// contentDensityMinChars is the minimum character count for an article to be
// considered as having meaningful content.
const contentDensityMinChars = 50

// StructureGate (V1) validates parsed document structure quality.
// Runs after document parsing.
type StructureGate struct{}

// NewStructureGate creates a new V1 structure validation gate.
func NewStructureGate() *StructureGate {
	return &StructureGate{}
}

// Name returns "V1".
func (structureGate *StructureGate) Name() string { return "V1" }

// Thresholds returns the default thresholds for structure validation metrics.
func (structureGate *StructureGate) Thresholds() map[string]float64 {
	return map[string]float64{
		"has_articles":           0.70,
		"structure_completeness": 0.60,
		"content_density":        0.50,
	}
}

// Run validates the parsed document has sufficient structural elements.
func (structureGate *StructureGate) Run(ctx *ValidationContext) *GateResult {
	startTime := time.Now()

	gateResult := &GateResult{
		Gate:     structureGate.Name(),
		Metrics:  make(map[string]float64),
		Warnings: make([]GateWarning, 0),
		Errors:   make([]GateError, 0),
	}

	if ctx.Document == nil {
		gateResult.Metrics["has_articles"] = 0.0
		gateResult.Metrics["structure_completeness"] = 0.0
		gateResult.Metrics["content_density"] = 0.0
		evaluateMetrics(gateResult, ctx.Config, structureGate)
		gateResult.Duration = time.Since(startTime)
		return gateResult
	}

	articles := CollectArticles(ctx.Document)
	articleCount := len(articles)
	chapterCount := len(ctx.Document.Chapters)

	// has_articles: 1.0 if any articles found, 0.0 otherwise.
	if articleCount > 0 {
		gateResult.Metrics["has_articles"] = 1.0
	} else {
		gateResult.Metrics["has_articles"] = 0.0
	}

	// structure_completeness: ratio of structural elements found.
	// Score based on having both chapters and articles.
	structureScore := 0.0
	if chapterCount > 0 {
		structureScore += 0.5
	}
	if articleCount > 0 {
		structureScore += 0.5
	}
	gateResult.Metrics["structure_completeness"] = structureScore

	// content_density: fraction of articles with meaningful content.
	if articleCount > 0 {
		articlesWithContent := 0
		for _, article := range articles {
			articleTextLength := len(article.Text)
			for _, paragraph := range article.Paragraphs {
				articleTextLength += len(paragraph.Text)
			}
			if articleTextLength >= contentDensityMinChars {
				articlesWithContent++
			}
		}
		gateResult.Metrics["content_density"] = float64(articlesWithContent) / float64(articleCount)
	} else {
		gateResult.Metrics["content_density"] = 0.0
	}

	evaluateMetrics(gateResult, ctx.Config, structureGate)
	gateResult.Duration = time.Since(startTime)
	return gateResult
}
