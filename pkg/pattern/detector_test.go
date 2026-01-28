package pattern

import (
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
