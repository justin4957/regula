package pattern

import (
	"sort"
	"strings"
)

// FormatMatch represents a detected format match with confidence score.
type FormatMatch struct {
	FormatID   string
	Pattern    *FormatPattern
	Confidence float64
	Score      float64
	MaxScore   float64
	Indicators []IndicatorMatch
}

// IndicatorMatch represents a matched indicator.
type IndicatorMatch struct {
	Pattern    string
	Weight     int
	MatchCount int
	Type       string // "required", "optional", "negative"
}

// FormatDetector detects document formats using registered patterns.
type FormatDetector struct {
	registry Registry
}

// NewFormatDetector creates a new format detector.
func NewFormatDetector(registry Registry) *FormatDetector {
	return &FormatDetector{
		registry: registry,
	}
}

// Detect analyzes content and returns format matches ranked by confidence.
func (d *FormatDetector) Detect(content string) []FormatMatch {
	patterns := d.registry.List()
	if len(patterns) == 0 {
		return nil
	}

	matches := make([]FormatMatch, 0)

	for _, pattern := range patterns {
		match := d.evaluatePattern(content, pattern)
		if match.Confidence > 0 {
			matches = append(matches, match)
		}
	}

	// Sort by confidence (descending)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Confidence > matches[j].Confidence
	})

	return matches
}

// DetectBest returns the best matching format, or nil if no match found.
func (d *FormatDetector) DetectBest(content string) *FormatMatch {
	matches := d.Detect(content)
	if len(matches) == 0 {
		return nil
	}
	return &matches[0]
}

// DetectWithThreshold returns matches above the given confidence threshold.
func (d *FormatDetector) DetectWithThreshold(content string, threshold float64) []FormatMatch {
	matches := d.Detect(content)

	filtered := make([]FormatMatch, 0)
	for _, match := range matches {
		if match.Confidence >= threshold {
			filtered = append(filtered, match)
		}
	}

	return filtered
}

// evaluatePattern evaluates a single pattern against the content.
func (d *FormatDetector) evaluatePattern(content string, pattern *FormatPattern) FormatMatch {
	match := FormatMatch{
		FormatID:   pattern.FormatID,
		Pattern:    pattern,
		Indicators: make([]IndicatorMatch, 0),
	}

	var score float64
	var maxScore float64

	// Evaluate required indicators
	requiredMatched := false
	for _, ind := range pattern.Detection.RequiredIndicators {
		maxScore += float64(ind.Weight)
		if ind.compiled != nil {
			matches := ind.compiled.FindAllString(content, -1)
			if len(matches) > 0 {
				score += float64(ind.Weight)
				requiredMatched = true
				match.Indicators = append(match.Indicators, IndicatorMatch{
					Pattern:    ind.Pattern,
					Weight:     ind.Weight,
					MatchCount: len(matches),
					Type:       "required",
				})
			}
		}
	}

	// If no required indicators matched, confidence is 0
	if !requiredMatched {
		return match
	}

	// Evaluate optional indicators
	for _, ind := range pattern.Detection.OptionalIndicators {
		if ind.compiled != nil {
			matches := ind.compiled.FindAllString(content, -1)
			if len(matches) > 0 {
				score += float64(ind.Weight)
				match.Indicators = append(match.Indicators, IndicatorMatch{
					Pattern:    ind.Pattern,
					Weight:     ind.Weight,
					MatchCount: len(matches),
					Type:       "optional",
				})
			}
		}
	}

	// Evaluate negative indicators
	for _, ind := range pattern.Detection.NegativeIndicators {
		if ind.compiled != nil {
			matches := ind.compiled.FindAllString(content, -1)
			if len(matches) > 0 {
				score += float64(ind.Weight) // Weight should be negative
				match.Indicators = append(match.Indicators, IndicatorMatch{
					Pattern:    ind.Pattern,
					Weight:     ind.Weight,
					MatchCount: len(matches),
					Type:       "negative",
				})
			}
		}
	}

	// Calculate confidence
	match.Score = score
	match.MaxScore = maxScore
	if maxScore > 0 {
		match.Confidence = score / maxScore
		// Clamp to [0, 1]
		if match.Confidence < 0 {
			match.Confidence = 0
		}
		if match.Confidence > 1 {
			match.Confidence = 1
		}
	}

	return match
}

// DetectFromLines is a convenience method that joins lines and detects format.
func (d *FormatDetector) DetectFromLines(lines []string) []FormatMatch {
	content := strings.Join(lines, "\n")
	return d.Detect(content)
}

// DetectBestFromLines is a convenience method for single best match from lines.
func (d *FormatDetector) DetectBestFromLines(lines []string) *FormatMatch {
	content := strings.Join(lines, "\n")
	return d.DetectBest(content)
}
