package validate

import (
	"time"

	"github.com/coolbeans/regula/pkg/extract"
)

// QualityGate (V3) validates resolution quality and overall graph confidence.
// Runs after reference resolution and graph building.
type QualityGate struct{}

// NewQualityGate creates a new V3 quality validation gate.
func NewQualityGate() *QualityGate {
	return &QualityGate{}
}

// Name returns "V3".
func (qualityGate *QualityGate) Name() string { return "V3" }

// Thresholds returns the default thresholds for quality validation metrics.
func (qualityGate *QualityGate) Thresholds() map[string]float64 {
	return map[string]float64{
		"resolution_rate":    0.80,
		"confidence_average": 0.70,
		"graph_connectivity": 0.50,
	}
}

// Run validates reference resolution rate, confidence levels, and graph connectivity.
func (qualityGate *QualityGate) Run(ctx *ValidationContext) *GateResult {
	startTime := time.Now()

	gateResult := &GateResult{
		Gate:     qualityGate.Name(),
		Metrics:  make(map[string]float64),
		Warnings: make([]GateWarning, 0),
		Errors:   make([]GateError, 0),
	}

	// resolution_rate: fraction of internal references successfully resolved.
	// Aligns with existing Validator logic: counts Resolved + Partial + RangeRef as
	// successful, excludes External from the denominator.
	totalReferences := len(ctx.ResolvedReferences)
	if totalReferences > 0 {
		successfullyResolved := 0
		externalCount := 0
		totalConfidence := 0.0

		for _, resolvedRef := range ctx.ResolvedReferences {
			switch resolvedRef.Status {
			case extract.ResolutionResolved, extract.ResolutionPartial, extract.ResolutionRangeRef, extract.ResolutionSelfRef:
				successfullyResolved++
			case extract.ResolutionExternal:
				externalCount++
			}
			totalConfidence += float64(resolvedRef.Confidence)
		}

		internalReferences := totalReferences - externalCount
		if internalReferences > 0 {
			gateResult.Metrics["resolution_rate"] = float64(successfullyResolved) / float64(internalReferences)
		} else {
			gateResult.Metrics["resolution_rate"] = 1.0 // All external — consider resolved.
		}
		gateResult.Metrics["confidence_average"] = totalConfidence / float64(totalReferences)
	} else {
		gateResult.Metrics["resolution_rate"] = 0.0
		gateResult.Metrics["confidence_average"] = 0.0
	}

	// graph_connectivity: fraction of articles that are connected (non-orphan).
	if ctx.Document != nil && ctx.TripleStore != nil {
		articles := CollectArticles(ctx.Document)
		totalArticles := len(articles)

		if totalArticles > 0 {
			connectedCount := 0
			for _, article := range articles {
				articleNum := article.Number
				// Check if this article has any incoming or outgoing references.
				hasConnection := false
				for _, resolvedRef := range ctx.ResolvedReferences {
					if resolvedRef.Original != nil {
						if resolvedRef.Original.SourceArticle == articleNum {
							hasConnection = true
							break
						}
						if resolvedRef.Original.ArticleNum == articleNum &&
							(resolvedRef.Status == extract.ResolutionResolved || resolvedRef.Status == extract.ResolutionRangeRef) {
							hasConnection = true
							break
						}
					}
				}
				if hasConnection {
					connectedCount++
				}
			}
			gateResult.Metrics["graph_connectivity"] = float64(connectedCount) / float64(totalArticles)
		} else {
			gateResult.Metrics["graph_connectivity"] = 0.0
		}
	} else {
		gateResult.Metrics["graph_connectivity"] = 0.0
	}

	evaluateMetrics(gateResult, ctx.Config, qualityGate)
	gateResult.Duration = time.Since(startTime)
	return gateResult
}
