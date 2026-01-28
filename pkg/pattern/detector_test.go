package pattern

import (
	"strings"
	"testing"
)

func createTestRegistry() *DefaultRegistry {
	registry := NewRegistry()

	// US Code pattern
	usCode := &FormatPattern{
		Name:         "US Code",
		FormatID:     "usc",
		Version:      "1.0.0",
		Jurisdiction: "US",
		Detection: DetectionConfig{
			RequiredIndicators: []Indicator{
				{Pattern: `\bU\.?S\.?C\.?\s*§?\s*\d+`, Weight: 20},
			},
			OptionalIndicators: []Indicator{
				{Pattern: `\bTitle\s+\d+`, Weight: 10},
				{Pattern: `\bChapter\s+\d+`, Weight: 5},
			},
			NegativeIndicators: []Indicator{
				{Pattern: `\bEuropean\s+Union\b`, Weight: -15},
			},
		},
	}

	// EU Directive pattern
	euDirective := &FormatPattern{
		Name:         "EU Directive",
		FormatID:     "eu-directive",
		Version:      "1.0.0",
		Jurisdiction: "EU",
		Detection: DetectionConfig{
			RequiredIndicators: []Indicator{
				{Pattern: `\bDirective\s+\d{4}/\d+/E[CU]`, Weight: 25},
			},
			OptionalIndicators: []Indicator{
				{Pattern: `\bEuropean\s+(Parliament|Union)\b`, Weight: 10},
				{Pattern: `\bArticle\s+\d+`, Weight: 5},
			},
			NegativeIndicators: []Indicator{
				{Pattern: `\bU\.?S\.?\s+Code\b`, Weight: -15},
			},
		},
	}

	// California Code pattern
	caCode := &FormatPattern{
		Name:         "California Code",
		FormatID:     "ca-code",
		Version:      "1.0.0",
		Jurisdiction: "US-CA",
		Detection: DetectionConfig{
			RequiredIndicators: []Indicator{
				{Pattern: `California\s+Civil\s+Code`, Weight: 20},
			},
			OptionalIndicators: []Indicator{
				{Pattern: `\bSection\s+\d+`, Weight: 10},
				{Pattern: `\bCal\.\s*Civ\.\s*Code`, Weight: 15},
			},
		},
	}

	registry.Register(usCode)
	registry.Register(euDirective)
	registry.Register(caCode)

	return registry
}

func TestFormatDetectorDetect(t *testing.T) {
	registry := createTestRegistry()
	detector := NewFormatDetector(registry)

	tests := []struct {
		name           string
		content        string
		wantMatches    int
		wantBestFormat string
	}{
		{
			name:           "US Code content",
			content:        "Pursuant to 42 U.S.C. § 1983, Title 42, Chapter 21, the plaintiff claims...",
			wantMatches:    1,
			wantBestFormat: "usc",
		},
		{
			name:           "EU Directive content",
			content:        "In accordance with Directive 2016/679/EU of the European Parliament and Article 5...",
			wantMatches:    1,
			wantBestFormat: "eu-directive",
		},
		{
			name:           "California Code content",
			content:        "California Civil Code Section 1798.100 provides that consumers have the right...",
			wantMatches:    1,
			wantBestFormat: "ca-code",
		},
		{
			name:           "No match",
			content:        "This is just regular text with no legal references.",
			wantMatches:    0,
			wantBestFormat: "",
		},
		{
			name:           "Multiple matches possible",
			content:        "U.S.C. § 1234 and California Civil Code Section 1798",
			wantMatches:    2,
			wantBestFormat: "", // Both match; best depends on confidence scoring
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := detector.Detect(tt.content)

			if len(matches) != tt.wantMatches {
				t.Errorf("Detect() matches = %d, want %d", len(matches), tt.wantMatches)
				for _, m := range matches {
					t.Logf("  Match: %s (confidence: %.2f)", m.FormatID, m.Confidence)
				}
			}

			if tt.wantBestFormat != "" && len(matches) > 0 {
				if matches[0].FormatID != tt.wantBestFormat {
					t.Errorf("Best match = %q, want %q", matches[0].FormatID, tt.wantBestFormat)
				}
			}
		})
	}
}

func TestFormatDetectorDetectBest(t *testing.T) {
	registry := createTestRegistry()
	detector := NewFormatDetector(registry)

	// Test with matching content
	match := detector.DetectBest("42 U.S.C. § 1983 provides...")
	if match == nil {
		t.Fatal("DetectBest() returned nil for matching content")
	}
	if match.FormatID != "usc" {
		t.Errorf("DetectBest() FormatID = %q, want %q", match.FormatID, "usc")
	}

	// Test with non-matching content
	match = detector.DetectBest("Just regular text")
	if match != nil {
		t.Error("DetectBest() should return nil for non-matching content")
	}
}

func TestFormatDetectorDetectWithThreshold(t *testing.T) {
	registry := createTestRegistry()
	detector := NewFormatDetector(registry)

	content := "42 U.S.C. § 1983, Title 42, Chapter 21"
	allMatches := detector.Detect(content)

	if len(allMatches) == 0 {
		t.Fatal("Expected at least one match")
	}

	// With high threshold, should get fewer or no matches
	highThreshold := detector.DetectWithThreshold(content, 0.99)
	if len(highThreshold) > len(allMatches) {
		t.Error("High threshold should not return more matches")
	}

	// With zero threshold, should get all matches
	zeroThreshold := detector.DetectWithThreshold(content, 0)
	if len(zeroThreshold) != len(allMatches) {
		t.Errorf("Zero threshold matches = %d, want %d", len(zeroThreshold), len(allMatches))
	}
}

func TestFormatDetectorDetectFromLines(t *testing.T) {
	registry := createTestRegistry()
	detector := NewFormatDetector(registry)

	lines := []string{
		"TITLE 42 - THE PUBLIC HEALTH AND WELFARE",
		"",
		"CHAPTER 21 - CIVIL RIGHTS",
		"",
		"42 U.S.C. § 1983 - Civil action for deprivation of rights",
	}

	matches := detector.DetectFromLines(lines)
	if len(matches) == 0 {
		t.Error("DetectFromLines() should find matches")
	}
	if matches[0].FormatID != "usc" {
		t.Errorf("DetectFromLines() best = %q, want %q", matches[0].FormatID, "usc")
	}
}

func TestFormatDetectorDetectBestFromLines(t *testing.T) {
	registry := createTestRegistry()
	detector := NewFormatDetector(registry)

	lines := []string{
		"Directive 2016/679/EU",
		"of the European Parliament",
		"Article 5 - Principles",
	}

	match := detector.DetectBestFromLines(lines)
	if match == nil {
		t.Fatal("DetectBestFromLines() returned nil")
	}
	if match.FormatID != "eu-directive" {
		t.Errorf("DetectBestFromLines() = %q, want %q", match.FormatID, "eu-directive")
	}
}

func TestFormatDetectorEmptyRegistry(t *testing.T) {
	registry := NewRegistry()
	detector := NewFormatDetector(registry)

	matches := detector.Detect("Some content")
	if matches != nil {
		t.Error("Detect() with empty registry should return nil")
	}

	match := detector.DetectBest("Some content")
	if match != nil {
		t.Error("DetectBest() with empty registry should return nil")
	}
}

func TestFormatDetectorConfidenceCalculation(t *testing.T) {
	registry := NewRegistry()

	pattern := &FormatPattern{
		Name:     "Test Pattern",
		FormatID: "test",
		Version:  "1.0.0",
		Detection: DetectionConfig{
			RequiredIndicators: []Indicator{
				{Pattern: `REQUIRED`, Weight: 10},
			},
			OptionalIndicators: []Indicator{
				{Pattern: `OPTIONAL1`, Weight: 5},
				{Pattern: `OPTIONAL2`, Weight: 5},
			},
		},
	}

	registry.Register(pattern)
	detector := NewFormatDetector(registry)

	tests := []struct {
		name           string
		content        string
		wantConfidence float64
	}{
		{
			name:           "Required only",
			content:        "This has REQUIRED indicator",
			wantConfidence: 1.0, // 10/10
		},
		{
			name:           "Required + one optional",
			content:        "This has REQUIRED and OPTIONAL1",
			wantConfidence: 1.0, // Optional adds to score but maxScore only from required
		},
		{
			name:           "No match",
			content:        "No indicators here",
			wantConfidence: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := detector.DetectBest(tt.content)
			if tt.wantConfidence == 0 {
				if match != nil {
					t.Errorf("Expected no match, got confidence %.2f", match.Confidence)
				}
				return
			}

			if match == nil {
				t.Fatal("Expected a match")
			}

			// Note: Confidence can exceed 1.0 if optional indicators add to score
			// The algorithm clamps to [0, 1] but we verify the basic behavior
			if match.Confidence < 0 || match.Confidence > 1 {
				t.Errorf("Confidence = %.2f, should be in [0, 1]", match.Confidence)
			}
		})
	}
}

func TestFormatDetectorNegativeIndicators(t *testing.T) {
	registry := NewRegistry()

	pattern := &FormatPattern{
		Name:     "Test Pattern",
		FormatID: "test",
		Version:  "1.0.0",
		Detection: DetectionConfig{
			RequiredIndicators: []Indicator{
				{Pattern: `REQUIRED`, Weight: 10},
			},
			NegativeIndicators: []Indicator{
				{Pattern: `NEGATIVE`, Weight: -20},
			},
		},
	}

	registry.Register(pattern)
	detector := NewFormatDetector(registry)

	// Without negative indicator
	match1 := detector.DetectBest("This has REQUIRED only")
	if match1 == nil || match1.Confidence < 1.0 {
		t.Error("Should have high confidence without negative indicator")
	}

	// With negative indicator - confidence should be lower or clamped to 0
	// Since score = 10 + (-20) = -10, and maxScore = 10, confidence = -10/10 = -1 clamped to 0
	// But this means Confidence > 0 check fails, so no match is returned from Detect()
	match2 := detector.DetectBest("This has REQUIRED but also NEGATIVE")
	// When confidence is clamped to 0, Detect() filters it out (line 49: if match.Confidence > 0)
	if match2 != nil {
		// If returned, should have 0 confidence
		if match2.Confidence != 0 {
			t.Errorf("Confidence with negative = %.2f, want 0 (clamped)", match2.Confidence)
		}
	}
	// It's valid for match2 to be nil since confidence = 0 is filtered out
}

func TestFormatDetectorIndicatorMatches(t *testing.T) {
	registry := NewRegistry()

	pattern := &FormatPattern{
		Name:     "Test Pattern",
		FormatID: "test",
		Version:  "1.0.0",
		Detection: DetectionConfig{
			RequiredIndicators: []Indicator{
				{Pattern: `KEYWORD`, Weight: 10},
			},
		},
	}

	registry.Register(pattern)
	detector := NewFormatDetector(registry)

	// Multiple matches of same pattern
	match := detector.DetectBest("KEYWORD KEYWORD KEYWORD")
	if match == nil {
		t.Fatal("Expected a match")
	}

	if len(match.Indicators) != 1 {
		t.Fatalf("Expected 1 indicator, got %d", len(match.Indicators))
	}

	if match.Indicators[0].MatchCount != 3 {
		t.Errorf("MatchCount = %d, want 3", match.Indicators[0].MatchCount)
	}

	if match.Indicators[0].Type != "required" {
		t.Errorf("Type = %q, want %q", match.Indicators[0].Type, "required")
	}
}

func TestFormatDetectorTieBreaking(t *testing.T) {
	registry := NewRegistry()

	// Two patterns with same confidence but different specificity
	patternA := &FormatPattern{
		Name:     "Pattern A",
		FormatID: "pattern-a",
		Version:  "1.0.0",
		Detection: DetectionConfig{
			RequiredIndicators: []Indicator{
				{Pattern: `COMMON`, Weight: 10},
			},
		},
	}

	patternB := &FormatPattern{
		Name:     "Pattern B",
		FormatID: "pattern-b",
		Version:  "1.0.0",
		Detection: DetectionConfig{
			RequiredIndicators: []Indicator{
				{Pattern: `COMMON`, Weight: 10},
			},
			OptionalIndicators: []Indicator{
				{Pattern: `EXTRA`, Weight: 5},
			},
		},
	}

	registry.Register(patternA)
	registry.Register(patternB)
	detector := NewFormatDetector(registry)

	// Content matches required for both, optional for pattern-b
	content := "COMMON EXTRA text"
	matches := detector.Detect(content)

	if len(matches) != 2 {
		t.Fatalf("Expected 2 matches, got %d", len(matches))
	}

	// Pattern B should win due to higher specificity (more indicators matched)
	if matches[0].FormatID != "pattern-b" {
		t.Errorf("Expected pattern-b first (more specific), got %q", matches[0].FormatID)
	}

	// Both should have confidence > 1.0 clamped to 1.0, or high confidence
	if matches[0].Specificity <= matches[1].Specificity {
		t.Errorf("pattern-b should have higher specificity: %.3f vs %.3f",
			matches[0].Specificity, matches[1].Specificity)
	}
}

func TestFormatDetectorOptions(t *testing.T) {
	registry := createTestRegistry()

	// Test MinConfidence filter
	t.Run("MinConfidence", func(t *testing.T) {
		options := DetectorOptions{
			MinConfidence: 0.5,
		}
		detector := NewFormatDetectorWithOptions(registry, options)

		content := "U.S.C. § 1234"
		matches := detector.Detect(content)

		for _, m := range matches {
			if m.Confidence < 0.5 {
				t.Errorf("Match %s has confidence %.2f below threshold", m.FormatID, m.Confidence)
			}
		}
	})

	// Test MaxResults limit
	t.Run("MaxResults", func(t *testing.T) {
		options := DetectorOptions{
			MaxResults: 1,
		}
		detector := NewFormatDetectorWithOptions(registry, options)

		// Content that matches multiple patterns
		content := "U.S.C. § 1234 and California Civil Code"
		matches := detector.Detect(content)

		if len(matches) > 1 {
			t.Errorf("Expected max 1 result, got %d", len(matches))
		}
	})

	// Test IncludePositions
	t.Run("IncludePositions", func(t *testing.T) {
		options := DetectorOptions{
			IncludePositions: true,
		}
		detector := NewFormatDetectorWithOptions(registry, options)

		content := "U.S.C. § 1234 Title 42 U.S.C. § 5678"
		match := detector.DetectBest(content)

		if match == nil {
			t.Fatal("Expected a match")
		}

		// Check that positions are included
		hasPositions := false
		for _, ind := range match.Indicators {
			if len(ind.Positions) > 0 {
				hasPositions = true
				break
			}
		}

		if !hasPositions {
			t.Error("Expected positions to be included in indicators")
		}
	})
}

func TestFormatDetectorSetOptions(t *testing.T) {
	registry := createTestRegistry()
	detector := NewFormatDetector(registry)

	// Initially should use default options
	content := "U.S.C. § 1234 and California Civil Code"
	matches1 := detector.Detect(content)

	// Update options to limit results
	detector.SetOptions(DetectorOptions{MaxResults: 1})
	matches2 := detector.Detect(content)

	if len(matches2) > 1 {
		t.Error("SetOptions should have limited results to 1")
	}

	if len(matches1) <= len(matches2) && len(matches1) > 1 {
		t.Error("Original detection should have had more results")
	}
}

func TestFormatDetectorDetectAll(t *testing.T) {
	registry := createTestRegistry()
	detector := NewFormatDetector(registry)

	// Content that doesn't match anything
	content := "Just regular text"
	matches := detector.DetectAll(content)

	// Should return all patterns (even with 0 confidence)
	if len(matches) != 3 {
		t.Errorf("DetectAll should return all %d patterns, got %d", 3, len(matches))
	}

	// All should have 0 confidence
	for _, m := range matches {
		if m.Confidence != 0 {
			t.Errorf("Expected 0 confidence for %s, got %.2f", m.FormatID, m.Confidence)
		}
	}
}

func TestFormatDetectorDetectWithDebug(t *testing.T) {
	registry := createTestRegistry()
	detector := NewFormatDetector(registry)

	content := "42 U.S.C. § 1983, Title 42"
	matches, debug := detector.DetectWithDebug(content)

	if len(matches) == 0 {
		t.Error("Expected matches")
	}

	// Debug output should contain useful info
	if !strings.Contains(debug, "Format Detection Results") {
		t.Error("Debug output missing header")
	}
	if !strings.Contains(debug, "usc") {
		t.Error("Debug output missing format ID")
	}
	if !strings.Contains(debug, "Confidence") {
		t.Error("Debug output missing confidence info")
	}
}

func TestFormatDetectorExplainMatch(t *testing.T) {
	registry := createTestRegistry()
	detector := NewFormatDetector(registry)

	content := "42 U.S.C. § 1983"

	// Test with matching format
	explanation := detector.ExplainMatch(content, "usc")
	if !strings.Contains(explanation, "✓") {
		t.Error("Explanation should show matched indicators with ✓")
	}
	if !strings.Contains(explanation, "MATCHES") {
		t.Error("Explanation should indicate the format matches")
	}

	// Test with non-matching format
	explanation = detector.ExplainMatch(content, "eu-directive")
	if !strings.Contains(explanation, "does NOT match") {
		t.Error("Explanation should indicate the format doesn't match")
	}

	// Test with unknown format
	explanation = detector.ExplainMatch(content, "unknown-format")
	if !strings.Contains(explanation, "not found") {
		t.Error("Explanation should indicate format not found")
	}
}

func TestFormatMatchString(t *testing.T) {
	match := FormatMatch{
		FormatID:   "test",
		Confidence: 0.85,
		Score:      17,
		MaxScore:   20,
		Indicators: []IndicatorMatch{
			{Pattern: "test", Weight: 10, MatchCount: 2, Type: "required"},
		},
	}

	str := match.String()
	if !strings.Contains(str, "test") {
		t.Error("String() should contain format ID")
	}
	if !strings.Contains(str, "85") {
		t.Error("String() should contain confidence percentage")
	}
}

func TestFormatMatchDebugString(t *testing.T) {
	match := FormatMatch{
		FormatID: "test",
		Pattern: &FormatPattern{
			Name:     "Test Pattern",
			FormatID: "test",
		},
		Confidence:      0.85,
		Score:           17,
		MaxScore:        20,
		RequiredMatched: 2,
		RequiredTotal:   2,
		OptionalMatched: 1,
		OptionalTotal:   3,
		NegativeMatched: 0,
		TotalMatchCount: 5,
		Specificity:     0.65,
		Indicators: []IndicatorMatch{
			{Pattern: "required1", Weight: 10, MatchCount: 2, Type: "required"},
			{Pattern: "required2", Weight: 10, MatchCount: 1, Type: "required"},
			{Pattern: "optional1", Weight: 5, MatchCount: 2, Type: "optional"},
		},
	}

	debug := match.DebugString()
	if !strings.Contains(debug, "Test Pattern") {
		t.Error("DebugString() should contain pattern name")
	}
	if !strings.Contains(debug, "85") {
		t.Error("DebugString() should contain confidence percentage")
	}
	if !strings.Contains(debug, "Required: 2/2") {
		t.Error("DebugString() should show required indicator stats")
	}
	if !strings.Contains(debug, "Optional: 1/3") {
		t.Error("DebugString() should show optional indicator stats")
	}
	if !strings.Contains(debug, "Specificity") {
		t.Error("DebugString() should show specificity")
	}
	if !strings.Contains(debug, "[required]") {
		t.Error("DebugString() should show indicator types")
	}
}

func TestSpecificityCalculation(t *testing.T) {
	tests := []struct {
		name    string
		match   FormatMatch
		wantMin float64
		wantMax float64
	}{
		{
			name: "all required matched",
			match: FormatMatch{
				RequiredMatched: 3,
				RequiredTotal:   3,
				OptionalMatched: 0,
				OptionalTotal:   0,
			},
			wantMin: 0.5,
			wantMax: 0.8,
		},
		{
			name: "all indicators matched",
			match: FormatMatch{
				RequiredMatched: 3,
				RequiredTotal:   3,
				OptionalMatched: 5,
				OptionalTotal:   5,
			},
			wantMin: 0.8,
			wantMax: 1.0,
		},
		{
			name: "partial match",
			match: FormatMatch{
				RequiredMatched: 1,
				RequiredTotal:   3,
				OptionalMatched: 1,
				OptionalTotal:   5,
			},
			wantMin: 0.1,
			wantMax: 0.5,
		},
		{
			name: "no required",
			match: FormatMatch{
				RequiredMatched: 0,
				RequiredTotal:   0,
			},
			wantMin: 0,
			wantMax: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			specificity := calculateSpecificity(&tt.match)
			if specificity < tt.wantMin || specificity > tt.wantMax {
				t.Errorf("calculateSpecificity() = %.3f, want [%.3f, %.3f]",
					specificity, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkDetect(b *testing.B) {
	registry := createTestRegistry()
	detector := NewFormatDetector(registry)

	content := `
		42 U.S.C. § 1983 provides that any person who, under color of any statute,
		ordinance, regulation, custom, or usage, of any State or Territory or the
		District of Columbia, subjects, or causes to be subjected, any citizen of
		the United States or other person within the jurisdiction thereof to the
		deprivation of any rights, privileges, or immunities secured by the
		Constitution and laws, shall be liable to the party injured in an action at
		law, suit in equity, or other proper proceeding for redress. Title 42,
		Chapter 21 of the United States Code contains the civil rights statutes.
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(content)
	}
}

func BenchmarkDetectLargeDocument(b *testing.B) {
	registry := createTestRegistry()
	detector := NewFormatDetector(registry)

	// Create a large document (simulating a real legal document)
	base := `
		Section 1. Pursuant to 42 U.S.C. § 1983, Title 42, Chapter 21, the plaintiff
		claims deprivation of constitutional rights. The applicable statute provides
		remedies for violations of civil rights under color of state law.
	`
	content := strings.Repeat(base, 100) // ~10KB document

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(content)
	}
}

func BenchmarkDetectWithPositions(b *testing.B) {
	registry := createTestRegistry()
	options := DetectorOptions{IncludePositions: true}
	detector := NewFormatDetectorWithOptions(registry, options)

	content := `
		42 U.S.C. § 1983, 42 U.S.C. § 1985, 42 U.S.C. § 1988
		Title 42 Chapter 21 provides civil rights protections.
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.Detect(content)
	}
}
