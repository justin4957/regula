package extract

import (
	"strings"
	"testing"
)

func TestPreprocessHouseRules_RemovesVerDateLines(t *testing.T) {
	lines := []string{
		"RULES",
		"of the",
		"HOUSE OF REPRESENTATIVES",
		"VerDate dec 05 2003 15:17 Jan 06, 2025 Jkt 000000 PO 00000 Frm 00001 Fmt 0204 Sfmt 0204",
		"RULE I",
		"THE SPEAKER",
	}
	result := PreprocessHouseRules(lines)

	for _, line := range result {
		if strings.HasPrefix(strings.TrimSpace(line), "VerDate") {
			t.Errorf("VerDate line was not removed: %q", line)
		}
	}
}

func TestPreprocessHouseRules_RemovesRunningHeaders(t *testing.T) {
	lines := []string{
		"RULE I",
		"THE SPEAKER",
		"1. The Speaker shall take the Chair",
		"Rule II, clause 2 Rule I, clause 12 ",
		"on every legislative day",
	}
	result := PreprocessHouseRules(lines)

	for _, line := range result {
		if strings.HasPrefix(strings.TrimSpace(line), "Rule II, clause") {
			t.Errorf("Running header was not removed: %q", line)
		}
	}
}

func TestPreprocessHouseRules_RemovesStandalonePageNumbers(t *testing.T) {
	lines := []string{
		"RULE I",
		"THE SPEAKER",
		"1. The Speaker shall take the Chair",
		"42",
		"on every legislative day",
	}
	result := PreprocessHouseRules(lines)

	for _, line := range result {
		if strings.TrimSpace(line) == "42" {
			t.Errorf("Standalone page number was not removed: %q", line)
		}
	}
}

func TestPreprocessHouseRules_RemovesRepeatedPageHeaders(t *testing.T) {
	lines := []string{
		"RULES",
		"of the",
		"HOUSE OF REPRESENTATIVES",
		"ONE HUNDRED NINETEENTH CONGRESS",
		"RULE I",
		"THE SPEAKER",
		"some clause text here",
		"RULES OF THE",
		"more clause text",
		"HOUSE OF REPRESENTATIVES",
		"continued text",
	}
	result := PreprocessHouseRules(lines)

	// Count occurrences of page headers after the initial title
	pageHeaderCount := 0
	for _, line := range result {
		trimmed := strings.TrimSpace(line)
		if trimmed == "RULES OF THE" || (trimmed == "HOUSE OF REPRESENTATIVES" && pageHeaderCount > 0) {
			pageHeaderCount++
		}
	}

	// The initial "HOUSE OF REPRESENTATIVES" in the title should remain,
	// but repeated page headers should be removed
	if pageHeaderCount > 0 {
		t.Errorf("Repeated page headers were not removed, found %d instances", pageHeaderCount)
	}
}

func TestPreprocessHouseRules_PreservesInitialTitle(t *testing.T) {
	lines := []string{
		"RULES",
		"of the",
		"HOUSE OF REPRESENTATIVES",
		"ONE HUNDRED NINETEENTH CONGRESS",
		"RULE I",
		"THE SPEAKER",
	}
	result := PreprocessHouseRules(lines)

	found := false
	for _, line := range result {
		if strings.TrimSpace(line) == "HOUSE OF REPRESENTATIVES" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Initial title 'HOUSE OF REPRESENTATIVES' should be preserved")
	}
}

func TestPreprocessHouseRules_RejoinsHyphenatedWords(t *testing.T) {
	lines := []string{
		"RULE I",
		"THE SPEAKER",
		"1. The Speaker shall take the Chair",
		"on every legislative day precisely at",
		"the hour to which the House last ad-",
		"journed and immediately call the",
		"House to order.",
	}
	result := PreprocessHouseRules(lines)

	found := false
	for _, line := range result {
		if strings.Contains(line, "adjourned") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Hyphenated word 'ad-journed' should be rejoined as 'adjourned'")
	}

	// Verify the hyphen line was consumed
	for _, line := range result {
		if strings.HasSuffix(strings.TrimSpace(line), "ad-") {
			t.Error("Hyphenated line ending should have been merged")
		}
	}
}

func TestPreprocessHouseRules_DoesNotRejoinNonWordHyphens(t *testing.T) {
	lines := []string{
		"RULE I",
		"THE SPEAKER",
		"1. The Speaker shall-",
		"(a) do something",
	}
	result := PreprocessHouseRules(lines)

	// The hyphen followed by a parenthesized subsection should NOT be joined
	foundOriginal := false
	for _, line := range result {
		if strings.Contains(line, "shall-") {
			foundOriginal = true
			break
		}
	}
	if !foundOriginal {
		t.Error("Hyphen before a capitalized/parenthesized line should not be rejoined")
	}
}

func TestRejoinHyphenatedLines(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "simple rejoin",
			input:    []string{"the ad-", "journed session"},
			expected: []string{"the adjourned session"},
		},
		{
			name:     "no rejoin with uppercase next line",
			input:    []string{"some text-", "CHAPTER II"},
			expected: []string{"some text-", "CHAPTER II"},
		},
		{
			name:     "multiple rejoins",
			input:    []string{"legis-", "lative", "pro-", "ceedings"},
			expected: []string{"legislative", "proceedings"},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "no hyphens",
			input:    []string{"no hyphens here", "at all"},
			expected: []string{"no hyphens here", "at all"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := rejoinHyphenatedLines(testCase.input)
			if len(result) == 0 && len(testCase.expected) == 0 {
				return
			}
			if len(result) != len(testCase.expected) {
				t.Errorf("expected %d lines, got %d: %v", len(testCase.expected), len(result), result)
				return
			}
			for i, line := range result {
				if line != testCase.expected[i] {
					t.Errorf("line %d: expected %q, got %q", i, testCase.expected[i], line)
				}
			}
		})
	}
}
