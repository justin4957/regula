package validate

import (
	"fmt"
	"math"
	"strings"

	"github.com/coolbeans/regula/pkg/extract"
)

// ProfileGenerator analyzes documents and extraction results to suggest
// optimized validation profiles with adaptive weights and thresholds.
type ProfileGenerator struct{}

// NewProfileGenerator creates a new ProfileGenerator.
func NewProfileGenerator() *ProfileGenerator {
	return &ProfileGenerator{}
}

// SuggestProfile analyzes a parsed document and extraction results to generate
// a tailored validation profile with reasoning for each suggested value.
func (generator *ProfileGenerator) SuggestProfile(
	doc *extract.Document,
	definitions []*extract.DefinedTerm,
	references []*extract.ResolvedReference,
	semantics []*extract.SemanticAnnotation,
	termUsages []*extract.TermUsage,
) *ProfileSuggestion {
	documentAnalysis := generator.analyzeDocument(doc, definitions, references, semantics, termUsages)

	var allReasoning []ProfileReasoning

	// Suggest expected counts
	expectedArticles, expectedDefinitions, expectedChapters, countReasoning := generator.suggestExpectedCounts(documentAnalysis)
	allReasoning = append(allReasoning, countReasoning...)

	// Suggest weights
	suggestedWeights, weightReasoning := generator.suggestWeights(documentAnalysis)
	allReasoning = append(allReasoning, weightReasoning...)

	// Classify rights and obligations
	rightsTypes, obligationTypes := generator.classifyRightsAndObligations(semantics)

	// Build profile name and description
	profileName := generator.suggestProfileName(doc)
	profileDescription := generator.suggestProfileDescription(doc, documentAnalysis)

	profile := &ValidationProfile{
		Name:                profileName,
		Description:         profileDescription,
		ExpectedArticles:    expectedArticles,
		ExpectedDefinitions: expectedDefinitions,
		ExpectedChapters:    expectedChapters,
		KnownRights:         rightsTypes,
		KnownObligations:    obligationTypes,
		Weights:             suggestedWeights,
	}

	confidence := generator.computeConfidence(documentAnalysis)

	return &ProfileSuggestion{
		Profile:       profile,
		Reasoning:     allReasoning,
		DocumentStats: documentAnalysis,
		Confidence:    confidence,
	}
}

// analyzeDocument computes metrics from the parsed document and extraction results.
func (generator *ProfileGenerator) analyzeDocument(
	doc *extract.Document,
	definitions []*extract.DefinedTerm,
	references []*extract.ResolvedReference,
	semantics []*extract.SemanticAnnotation,
	termUsages []*extract.TermUsage,
) *DocumentAnalysis {
	documentAnalysis := &DocumentAnalysis{
		DefinitionCount:  len(definitions),
		ReferenceCount:   len(references),
		RightsTypes:      []string{},
		ObligationTypes:  []string{},
	}

	if doc != nil {
		articles := CollectArticles(doc)
		documentAnalysis.ArticleCount = len(articles)
		documentAnalysis.ChapterCount = len(doc.Chapters)

		// Count sections and compute nesting depth
		maxNestingDepth := 0
		sectionCount := 0
		for _, chapter := range doc.Chapters {
			if len(chapter.Sections) > 0 {
				sectionCount += len(chapter.Sections)
				// Chapter > Section > Article = depth 3
				if 3 > maxNestingDepth {
					maxNestingDepth = 3
				}
			} else if len(chapter.Articles) > 0 {
				// Chapter > Article = depth 2
				if 2 > maxNestingDepth {
					maxNestingDepth = 2
				}
			}
		}
		documentAnalysis.SectionCount = sectionCount
		documentAnalysis.NestingDepth = maxNestingDepth

		// Count recitals
		if doc.Preamble != nil {
			documentAnalysis.RecitalCount = len(doc.Preamble.Recitals)
		}

		// Compute average article length (paragraph count)
		if len(articles) > 0 {
			totalParagraphs := 0
			for _, article := range articles {
				totalParagraphs += len(article.Paragraphs)
			}
			documentAnalysis.AvgArticleLength = float64(totalParagraphs) / float64(len(articles))
		}
	}

	// Count external references
	externalRefCount := 0
	for _, resolvedRef := range references {
		if resolvedRef.Status == extract.ResolutionExternal {
			externalRefCount++
		}
	}
	documentAnalysis.ExternalRefCount = externalRefCount

	// Count semantic annotations
	rightsCount := 0
	obligationsCount := 0
	rightsTypeSet := make(map[string]bool)
	obligationTypeSet := make(map[string]bool)

	for _, semanticAnnotation := range semantics {
		if semanticAnnotation.Type == extract.SemanticRight {
			rightsCount++
			if semanticAnnotation.RightType != "" {
				rightsTypeSet[string(semanticAnnotation.RightType)] = true
			}
		} else if semanticAnnotation.Type == extract.SemanticObligation {
			obligationsCount++
			if semanticAnnotation.ObligationType != "" {
				obligationTypeSet[string(semanticAnnotation.ObligationType)] = true
			}
		}
	}
	documentAnalysis.RightsCount = rightsCount
	documentAnalysis.ObligationsCount = obligationsCount

	for rightType := range rightsTypeSet {
		documentAnalysis.RightsTypes = append(documentAnalysis.RightsTypes, rightType)
	}
	for obligationType := range obligationTypeSet {
		documentAnalysis.ObligationTypes = append(documentAnalysis.ObligationTypes, obligationType)
	}

	// Compute densities
	if documentAnalysis.ArticleCount > 0 {
		documentAnalysis.DefinitionDensity = float64(documentAnalysis.DefinitionCount) / float64(documentAnalysis.ArticleCount)
		documentAnalysis.ReferenceDensity = float64(documentAnalysis.ReferenceCount) / float64(documentAnalysis.ArticleCount)
	}

	return documentAnalysis
}

// suggestWeights computes adaptive scoring weights based on document characteristics.
// Weights always sum to 1.0.
func (generator *ProfileGenerator) suggestWeights(
	documentAnalysis *DocumentAnalysis,
) (ValidationWeights, []ProfileReasoning) {
	var reasoning []ProfileReasoning

	// Start from default weights
	referenceWeight := 0.25
	connectivityWeight := 0.20
	definitionWeight := 0.20
	semanticWeight := 0.20
	structureWeight := 0.15

	// Adjust based on document characteristics

	// High definition density → boost definition weight
	if documentAnalysis.DefinitionDensity > 0.3 {
		definitionWeight += 0.05
		reasoning = append(reasoning, ProfileReasoning{
			Field:  "weights.definition_coverage",
			Value:  fmt.Sprintf("%.2f", definitionWeight),
			Reason: fmt.Sprintf("Definition density is high (%.2f defs/article), increasing weight", documentAnalysis.DefinitionDensity),
		})
	} else if documentAnalysis.DefinitionCount == 0 {
		// No definitions — reduce weight significantly
		definitionWeight = 0.05
		reasoning = append(reasoning, ProfileReasoning{
			Field:  "weights.definition_coverage",
			Value:  fmt.Sprintf("%.2f", definitionWeight),
			Reason: "No definitions found, reducing weight to minimal",
		})
	}

	// High reference density → boost reference weight
	if documentAnalysis.ReferenceDensity > 3.0 {
		referenceWeight += 0.05
		reasoning = append(reasoning, ProfileReasoning{
			Field:  "weights.reference_resolution",
			Value:  fmt.Sprintf("%.2f", referenceWeight),
			Reason: fmt.Sprintf("Reference density is high (%.1f refs/article), increasing weight", documentAnalysis.ReferenceDensity),
		})
	} else if documentAnalysis.ReferenceCount == 0 {
		referenceWeight = 0.05
		reasoning = append(reasoning, ProfileReasoning{
			Field:  "weights.reference_resolution",
			Value:  fmt.Sprintf("%.2f", referenceWeight),
			Reason: "No references found, reducing weight to minimal",
		})
	}

	// Deep nesting → boost structure weight
	if documentAnalysis.NestingDepth >= 3 {
		structureWeight += 0.05
		reasoning = append(reasoning, ProfileReasoning{
			Field:  "weights.structure_quality",
			Value:  fmt.Sprintf("%.2f", structureWeight),
			Reason: fmt.Sprintf("Deep document nesting (depth %d), increasing structure weight", documentAnalysis.NestingDepth),
		})
	}

	// Many rights/obligations → boost semantic weight
	if documentAnalysis.RightsCount > 10 || documentAnalysis.ObligationsCount > 10 {
		semanticWeight += 0.05
		reasoning = append(reasoning, ProfileReasoning{
			Field:  "weights.semantic_extraction",
			Value:  fmt.Sprintf("%.2f", semanticWeight),
			Reason: fmt.Sprintf("Rich semantic content (%d rights, %d obligations), increasing weight", documentAnalysis.RightsCount, documentAnalysis.ObligationsCount),
		})
	} else if documentAnalysis.RightsCount == 0 && documentAnalysis.ObligationsCount == 0 {
		semanticWeight = 0.05
		reasoning = append(reasoning, ProfileReasoning{
			Field:  "weights.semantic_extraction",
			Value:  fmt.Sprintf("%.2f", semanticWeight),
			Reason: "No rights or obligations found, reducing weight to minimal",
		})
	}

	// Normalize weights to sum to 1.0
	totalWeight := referenceWeight + connectivityWeight + definitionWeight + semanticWeight + structureWeight
	if totalWeight > 0 {
		referenceWeight /= totalWeight
		connectivityWeight /= totalWeight
		definitionWeight /= totalWeight
		semanticWeight /= totalWeight
		structureWeight /= totalWeight
	}

	// Round to 2 decimal places
	referenceWeight = roundToDecimals(referenceWeight, 2)
	connectivityWeight = roundToDecimals(connectivityWeight, 2)
	definitionWeight = roundToDecimals(definitionWeight, 2)
	semanticWeight = roundToDecimals(semanticWeight, 2)
	// Structure weight absorbs rounding remainder
	structureWeight = roundToDecimals(1.0-referenceWeight-connectivityWeight-definitionWeight-semanticWeight, 2)

	suggestedWeights := ValidationWeights{
		ReferenceResolution: referenceWeight,
		GraphConnectivity:   connectivityWeight,
		DefinitionCoverage:  definitionWeight,
		SemanticExtraction:  semanticWeight,
		StructureQuality:    structureWeight,
	}

	return suggestedWeights, reasoning
}

// suggestExpectedCounts determines expected articles, definitions, and chapters
// based on the actual document analysis.
func (generator *ProfileGenerator) suggestExpectedCounts(
	documentAnalysis *DocumentAnalysis,
) (int, int, int, []ProfileReasoning) {
	var reasoning []ProfileReasoning

	expectedArticles := documentAnalysis.ArticleCount
	reasoning = append(reasoning, ProfileReasoning{
		Field:  "expected.articles",
		Value:  fmt.Sprintf("%d", expectedArticles),
		Reason: fmt.Sprintf("Document contains %d articles", documentAnalysis.ArticleCount),
	})

	expectedDefinitions := documentAnalysis.DefinitionCount
	reasoning = append(reasoning, ProfileReasoning{
		Field:  "expected.definitions",
		Value:  fmt.Sprintf("%d", expectedDefinitions),
		Reason: fmt.Sprintf("Document contains %d definitions", documentAnalysis.DefinitionCount),
	})

	expectedChapters := documentAnalysis.ChapterCount
	reasoning = append(reasoning, ProfileReasoning{
		Field:  "expected.chapters",
		Value:  fmt.Sprintf("%d", expectedChapters),
		Reason: fmt.Sprintf("Document contains %d chapters", documentAnalysis.ChapterCount),
	})

	return expectedArticles, expectedDefinitions, expectedChapters, reasoning
}

// classifyRightsAndObligations extracts unique right and obligation type strings
// from semantic annotations.
func (generator *ProfileGenerator) classifyRightsAndObligations(
	semantics []*extract.SemanticAnnotation,
) ([]string, []string) {
	rightsTypeSet := make(map[string]bool)
	obligationTypeSet := make(map[string]bool)

	for _, semanticAnnotation := range semantics {
		if semanticAnnotation.Type == extract.SemanticRight && semanticAnnotation.RightType != "" {
			rightsTypeSet[string(semanticAnnotation.RightType)] = true
		}
		if semanticAnnotation.Type == extract.SemanticObligation && semanticAnnotation.ObligationType != "" {
			obligationTypeSet[string(semanticAnnotation.ObligationType)] = true
		}
	}

	rightsTypes := make([]string, 0, len(rightsTypeSet))
	for rightType := range rightsTypeSet {
		rightsTypes = append(rightsTypes, rightType)
	}

	obligationTypes := make([]string, 0, len(obligationTypeSet))
	for obligationType := range obligationTypeSet {
		obligationTypes = append(obligationTypes, obligationType)
	}

	return rightsTypes, obligationTypes
}

// computeConfidence returns a confidence score (0.0-1.0) based on how much
// data was available for the suggestion. More data points yield higher confidence.
func (generator *ProfileGenerator) computeConfidence(
	documentAnalysis *DocumentAnalysis,
) float64 {
	confidence := 0.0
	signalCount := 0

	// Articles are the primary data source
	if documentAnalysis.ArticleCount > 0 {
		confidence += 0.25
		signalCount++
		// Larger documents give more confident suggestions
		if documentAnalysis.ArticleCount >= 20 {
			confidence += 0.10
		}
	}

	// Chapters provide structural context
	if documentAnalysis.ChapterCount > 0 {
		confidence += 0.10
		signalCount++
	}

	// Definitions indicate a well-structured document
	if documentAnalysis.DefinitionCount > 0 {
		confidence += 0.15
		signalCount++
	}

	// References provide connectivity data
	if documentAnalysis.ReferenceCount > 0 {
		confidence += 0.15
		signalCount++
	}

	// Semantic annotations provide content understanding
	if documentAnalysis.RightsCount > 0 || documentAnalysis.ObligationsCount > 0 {
		confidence += 0.15
		signalCount++
	}

	// Bonus for having multiple signal types
	if signalCount >= 4 {
		confidence += 0.10
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return roundToDecimals(confidence, 2)
}

// suggestProfileName generates a descriptive name based on the document.
func (generator *ProfileGenerator) suggestProfileName(doc *extract.Document) string {
	if doc == nil {
		return "Custom"
	}

	if doc.Title != "" {
		// Truncate long titles
		title := doc.Title
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		return title
	}

	return "Custom"
}

// suggestProfileDescription generates a description based on document analysis.
func (generator *ProfileGenerator) suggestProfileDescription(
	doc *extract.Document,
	documentAnalysis *DocumentAnalysis,
) string {
	parts := []string{"Auto-generated profile"}

	if doc != nil && doc.Title != "" {
		parts = append(parts, fmt.Sprintf("for %q", doc.Title))
	}

	parts = append(parts, fmt.Sprintf("(%d articles, %d definitions, %d chapters)",
		documentAnalysis.ArticleCount,
		documentAnalysis.DefinitionCount,
		documentAnalysis.ChapterCount))

	return strings.Join(parts, " ")
}

// roundToDecimals rounds a float64 to the specified number of decimal places.
func roundToDecimals(value float64, decimals int) float64 {
	multiplier := math.Pow(10, float64(decimals))
	return math.Round(value*multiplier) / multiplier
}
