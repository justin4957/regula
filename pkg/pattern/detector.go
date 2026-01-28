package pattern

import (
	"fmt"
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

	// Scoring details for debugging and tie-breaking
	RequiredMatched int     // Number of required indicators matched
	RequiredTotal   int     // Total required indicators
	OptionalMatched int     // Number of optional indicators matched
	OptionalTotal   int     // Total optional indicators
	NegativeMatched int     // Number of negative indicators matched
	TotalMatchCount int     // Total pattern matches across all indicators
	Specificity     float64 // Specificity score for tie-breaking
}

// IndicatorMatch represents a matched indicator.
type IndicatorMatch struct {
	Pattern    string
	Weight     int
	MatchCount int
	Type       string // "required", "optional", "negative"
	Positions  []int  // Starting positions of matches (for debugging)
}

// String returns a human-readable summary of the match.
func (m *FormatMatch) String() string {
	return fmt.Sprintf("%s: %.1f%% confidence (score: %.1f/%.1f, %d indicators matched)",
		m.FormatID, m.Confidence*100, m.Score, m.MaxScore, len(m.Indicators))
}

// DebugString returns detailed debug information about the match.
func (m *FormatMatch) DebugString() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Format: %s (%s)\n", m.Pattern.Name, m.FormatID))
	sb.WriteString(fmt.Sprintf("  Confidence: %.2f (%.1f%%)\n", m.Confidence, m.Confidence*100))
	sb.WriteString(fmt.Sprintf("  Score: %.1f / %.1f (max)\n", m.Score, m.MaxScore))
	sb.WriteString(fmt.Sprintf("  Specificity: %.3f\n", m.Specificity))
	sb.WriteString(fmt.Sprintf("  Required: %d/%d matched\n", m.RequiredMatched, m.RequiredTotal))
	sb.WriteString(fmt.Sprintf("  Optional: %d/%d matched\n", m.OptionalMatched, m.OptionalTotal))
	sb.WriteString(fmt.Sprintf("  Negative: %d triggered\n", m.NegativeMatched))
	sb.WriteString(fmt.Sprintf("  Total pattern matches: %d\n", m.TotalMatchCount))

	if len(m.Indicators) > 0 {
		sb.WriteString("  Matched indicators:\n")
		for _, ind := range m.Indicators {
			sb.WriteString(fmt.Sprintf("    [%s] weight=%d matches=%d pattern=%q\n",
				ind.Type, ind.Weight, ind.MatchCount, truncatePattern(ind.Pattern, 50)))
		}
	}

	return sb.String()
}

func truncatePattern(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// DetectorOptions configures the format detector behavior.
type DetectorOptions struct {
	// MinConfidence filters out matches below this threshold (0.0-1.0)
	MinConfidence float64

	// MaxResults limits the number of results returned (0 = unlimited)
	MaxResults int

	// IncludePositions includes match positions in IndicatorMatch (slower)
	IncludePositions bool
}

// DefaultDetectorOptions returns sensible defaults.
func DefaultDetectorOptions() DetectorOptions {
	return DetectorOptions{
		MinConfidence:    0.0,
		MaxResults:       0,
		IncludePositions: false,
	}
}

// FormatDetector detects document formats using registered patterns.
type FormatDetector struct {
	registry Registry
	options  DetectorOptions
}

// NewFormatDetector creates a new format detector with default options.
func NewFormatDetector(registry Registry) *FormatDetector {
	return &FormatDetector{
		registry: registry,
		options:  DefaultDetectorOptions(),
	}
}

// NewFormatDetectorWithOptions creates a new format detector with custom options.
func NewFormatDetectorWithOptions(registry Registry, options DetectorOptions) *FormatDetector {
	return &FormatDetector{
		registry: registry,
		options:  options,
	}
}

// SetOptions updates the detector options.
func (d *FormatDetector) SetOptions(options DetectorOptions) {
	d.options = options
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
		if match.Confidence > d.options.MinConfidence {
			matches = append(matches, match)
		}
	}

	// Sort by confidence (descending), with specificity as tie-breaker
	sort.Slice(matches, func(i, j int) bool {
		// Primary: confidence
		if matches[i].Confidence != matches[j].Confidence {
			return matches[i].Confidence > matches[j].Confidence
		}
		// Secondary: specificity (more specific patterns win)
		if matches[i].Specificity != matches[j].Specificity {
			return matches[i].Specificity > matches[j].Specificity
		}
		// Tertiary: total match count (more matches = more certain)
		if matches[i].TotalMatchCount != matches[j].TotalMatchCount {
			return matches[i].TotalMatchCount > matches[j].TotalMatchCount
		}
		// Final: alphabetical by format ID for determinism
		return matches[i].FormatID < matches[j].FormatID
	})

	// Apply max results limit
	if d.options.MaxResults > 0 && len(matches) > d.options.MaxResults {
		matches = matches[:d.options.MaxResults]
	}

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
		FormatID:      pattern.FormatID,
		Pattern:       pattern,
		Indicators:    make([]IndicatorMatch, 0),
		RequiredTotal: len(pattern.Detection.RequiredIndicators),
		OptionalTotal: len(pattern.Detection.OptionalIndicators),
	}

	var score float64
	var maxScore float64
	var totalMatchCount int

	// Evaluate required indicators
	for _, ind := range pattern.Detection.RequiredIndicators {
		maxScore += float64(ind.Weight)
		if ind.compiled != nil {
			var positions []int
			if d.options.IncludePositions {
				locs := ind.compiled.FindAllStringIndex(content, -1)
				for _, loc := range locs {
					positions = append(positions, loc[0])
				}
			}

			matches := ind.compiled.FindAllString(content, -1)
			if len(matches) > 0 {
				score += float64(ind.Weight)
				match.RequiredMatched++
				totalMatchCount += len(matches)
				match.Indicators = append(match.Indicators, IndicatorMatch{
					Pattern:    ind.Pattern,
					Weight:     ind.Weight,
					MatchCount: len(matches),
					Type:       "required",
					Positions:  positions,
				})
			}
		}
	}

	// If no required indicators matched, confidence is 0
	if match.RequiredMatched == 0 {
		return match
	}

	// Evaluate optional indicators
	for _, ind := range pattern.Detection.OptionalIndicators {
		if ind.compiled != nil {
			var positions []int
			if d.options.IncludePositions {
				locs := ind.compiled.FindAllStringIndex(content, -1)
				for _, loc := range locs {
					positions = append(positions, loc[0])
				}
			}

			matches := ind.compiled.FindAllString(content, -1)
			if len(matches) > 0 {
				score += float64(ind.Weight)
				match.OptionalMatched++
				totalMatchCount += len(matches)
				match.Indicators = append(match.Indicators, IndicatorMatch{
					Pattern:    ind.Pattern,
					Weight:     ind.Weight,
					MatchCount: len(matches),
					Type:       "optional",
					Positions:  positions,
				})
			}
		}
	}

	// Evaluate negative indicators
	for _, ind := range pattern.Detection.NegativeIndicators {
		if ind.compiled != nil {
			var positions []int
			if d.options.IncludePositions {
				locs := ind.compiled.FindAllStringIndex(content, -1)
				for _, loc := range locs {
					positions = append(positions, loc[0])
				}
			}

			matches := ind.compiled.FindAllString(content, -1)
			if len(matches) > 0 {
				score += float64(ind.Weight) // Weight should be negative
				match.NegativeMatched++
				totalMatchCount += len(matches)
				match.Indicators = append(match.Indicators, IndicatorMatch{
					Pattern:    ind.Pattern,
					Weight:     ind.Weight,
					MatchCount: len(matches),
					Type:       "negative",
					Positions:  positions,
				})
			}
		}
	}

	// Store total match count
	match.TotalMatchCount = totalMatchCount

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

	// Calculate specificity for tie-breaking
	// Specificity is based on:
	// 1. Ratio of required indicators matched (most important)
	// 2. Number of different indicator types matched
	// 3. Total indicators in the pattern (more specific patterns have more indicators)
	match.Specificity = calculateSpecificity(&match)

	return match
}

// calculateSpecificity computes a specificity score for tie-breaking.
// Higher specificity means a more specific/certain match.
func calculateSpecificity(m *FormatMatch) float64 {
	if m.RequiredTotal == 0 {
		return 0
	}

	// Component 1: Required coverage (0-0.5)
	requiredCoverage := float64(m.RequiredMatched) / float64(m.RequiredTotal) * 0.5

	// Component 2: Optional coverage bonus (0-0.3)
	var optionalCoverage float64
	if m.OptionalTotal > 0 {
		optionalCoverage = float64(m.OptionalMatched) / float64(m.OptionalTotal) * 0.3
	}

	// Component 3: Pattern complexity bonus (0-0.2)
	// More indicators in the pattern = more specific pattern
	totalIndicators := m.RequiredTotal + m.OptionalTotal
	complexityBonus := float64(min(totalIndicators, 10)) / 10.0 * 0.2

	return requiredCoverage + optionalCoverage + complexityBonus
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

// DetectAll returns all matches including those with zero confidence.
// This is useful for debugging why certain patterns didn't match.
func (d *FormatDetector) DetectAll(content string) []FormatMatch {
	patterns := d.registry.List()
	if len(patterns) == 0 {
		return nil
	}

	matches := make([]FormatMatch, 0, len(patterns))
	for _, pattern := range patterns {
		match := d.evaluatePattern(content, pattern)
		matches = append(matches, match)
	}

	// Sort by confidence (descending)
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Confidence != matches[j].Confidence {
			return matches[i].Confidence > matches[j].Confidence
		}
		return matches[i].FormatID < matches[j].FormatID
	})

	return matches
}

// DetectWithDebug returns matches along with a debug string explaining the scoring.
func (d *FormatDetector) DetectWithDebug(content string) ([]FormatMatch, string) {
	matches := d.Detect(content)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Format Detection Results (%d matches)\n", len(matches)))
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")

	if len(matches) == 0 {
		sb.WriteString("No matching formats found.\n")
	} else {
		for i, match := range matches {
			sb.WriteString(fmt.Sprintf("#%d ", i+1))
			sb.WriteString(match.DebugString())
			sb.WriteString("\n")
		}
	}

	return matches, sb.String()
}

// ExplainMatch returns a detailed explanation of why a specific format matched or didn't match.
func (d *FormatDetector) ExplainMatch(content string, formatID string) string {
	pattern, ok := d.registry.Get(formatID)
	if !ok {
		return fmt.Sprintf("Format %q not found in registry", formatID)
	}

	match := d.evaluatePattern(content, pattern)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Explanation for format: %s (%s)\n", pattern.Name, formatID))
	sb.WriteString(strings.Repeat("-", 50) + "\n\n")

	// Required indicators
	sb.WriteString("Required Indicators:\n")
	for i, ind := range pattern.Detection.RequiredIndicators {
		matched := false
		matchCount := 0
		if ind.compiled != nil {
			matches := ind.compiled.FindAllString(content, -1)
			matchCount = len(matches)
			matched = matchCount > 0
		}
		status := "✗"
		if matched {
			status = "✓"
		}
		sb.WriteString(fmt.Sprintf("  %s [%d] weight=%d matches=%d pattern=%q\n",
			status, i, ind.Weight, matchCount, truncatePattern(ind.Pattern, 40)))
	}

	// Optional indicators
	if len(pattern.Detection.OptionalIndicators) > 0 {
		sb.WriteString("\nOptional Indicators:\n")
		for i, ind := range pattern.Detection.OptionalIndicators {
			matched := false
			matchCount := 0
			if ind.compiled != nil {
				matches := ind.compiled.FindAllString(content, -1)
				matchCount = len(matches)
				matched = matchCount > 0
			}
			status := "✗"
			if matched {
				status = "✓"
			}
			sb.WriteString(fmt.Sprintf("  %s [%d] weight=%d matches=%d pattern=%q\n",
				status, i, ind.Weight, matchCount, truncatePattern(ind.Pattern, 40)))
		}
	}

	// Negative indicators
	if len(pattern.Detection.NegativeIndicators) > 0 {
		sb.WriteString("\nNegative Indicators:\n")
		for i, ind := range pattern.Detection.NegativeIndicators {
			triggered := false
			matchCount := 0
			if ind.compiled != nil {
				matches := ind.compiled.FindAllString(content, -1)
				matchCount = len(matches)
				triggered = matchCount > 0
			}
			status := "✓" // Not triggered is good
			if triggered {
				status = "⚠" // Triggered (penalty applied)
			}
			sb.WriteString(fmt.Sprintf("  %s [%d] weight=%d matches=%d pattern=%q\n",
				status, i, ind.Weight, matchCount, truncatePattern(ind.Pattern, 40)))
		}
	}

	// Summary
	sb.WriteString(fmt.Sprintf("\nSummary:\n"))
	sb.WriteString(fmt.Sprintf("  Score: %.1f / %.1f\n", match.Score, match.MaxScore))
	sb.WriteString(fmt.Sprintf("  Confidence: %.2f (%.1f%%)\n", match.Confidence, match.Confidence*100))
	sb.WriteString(fmt.Sprintf("  Required: %d/%d matched\n", match.RequiredMatched, match.RequiredTotal))
	sb.WriteString(fmt.Sprintf("  Optional: %d/%d matched\n", match.OptionalMatched, match.OptionalTotal))
	sb.WriteString(fmt.Sprintf("  Specificity: %.3f\n", match.Specificity))

	if match.Confidence > 0 {
		sb.WriteString("\n  → This format MATCHES the content\n")
	} else if match.RequiredMatched == 0 {
		sb.WriteString("\n  → No required indicators matched - format does NOT match\n")
	} else {
		sb.WriteString("\n  → Confidence too low - format does NOT match\n")
	}

	return sb.String()
}
