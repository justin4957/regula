package extract

import (
	"os"
	"strings"
	"testing"

	"github.com/coolbeans/regula/pkg/pattern"
)

func TestHouseRulesPatternDetection(t *testing.T) {
	// Load pattern registry
	registry := pattern.NewRegistry()
	patternsDir := "../../patterns"
	if _, err := os.Stat(patternsDir); err != nil {
		t.Skipf("patterns directory not found at %s: %v", patternsDir, err)
	}
	if err := registry.LoadDirectory(patternsDir); err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}

	// Sample House Rules text
	sampleText := `RULES
of the
HOUSE OF REPRESENTATIVES
ONE HUNDRED NINETEENTH CONGRESS

RULE I
THE SPEAKER
Approval of the Journal
1. The Speaker shall take the Chair on every legislative day.

RULE II
OTHER OFFICERS AND OFFICIALS
1. There shall be elected a Clerk, a Sergeant-at-Arms, a Chief Administrative Officer, and a Chaplain.

RULE X
ORGANIZATION OF COMMITTEES
Committees and their legislative jurisdictions
1. (a) There shall be in the House the following standing committees, each of which shall have the jurisdiction indicated.
(1) Committee on Agriculture.
(A) Adulteration of seeds, insect pests, and protection of birds and animals.`

	detector := pattern.NewFormatDetector(registry)
	matches := detector.DetectWithThreshold(sampleText, 0.3)

	if len(matches) == 0 {
		t.Fatal("No patterns matched the House Rules sample text")
	}

	// The us-house-rules pattern should be the top match
	topMatch := matches[0]
	if topMatch.FormatID != "us-house-rules" {
		t.Errorf("Expected top match to be 'us-house-rules', got %q (confidence: %.2f)",
			topMatch.FormatID, topMatch.Confidence)
		for i, m := range matches {
			t.Logf("  match %d: %s (confidence: %.2f, score: %.0f/%.0f)",
				i, m.FormatID, m.Confidence, m.Score, m.MaxScore)
		}
	}
}

func TestHouseRulesParsing(t *testing.T) {
	// Load pattern registry
	registry := pattern.NewRegistry()
	patternsDir := "../../patterns"
	if _, err := os.Stat(patternsDir); err != nil {
		t.Skipf("patterns directory not found at %s: %v", patternsDir, err)
	}
	if err := registry.LoadDirectory(patternsDir); err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}

	// Load the actual House Rules file
	filePath := "../../testdata/house-rules-119th.txt"
	if _, err := os.Stat(filePath); err != nil {
		t.Skipf("House Rules file not found at %s: %v", filePath, err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("Failed to open House Rules file: %v", err)
	}
	defer file.Close()

	parser := NewParserWithRegistry(registry)

	doc, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Failed to parse House Rules: %v", err)
	}

	// Verify we got chapters (rules)
	if len(doc.Chapters) == 0 {
		t.Fatal("No chapters (rules) extracted from House Rules document")
	}

	// The House Rules have 29 rules (I through XXIX)
	t.Logf("Extracted %d rules (chapters)", len(doc.Chapters))

	// Check for expected rule numbers
	expectedFirstRules := []string{"I", "II", "III", "IV", "V"}
	for i, expectedNum := range expectedFirstRules {
		if i >= len(doc.Chapters) {
			t.Errorf("Expected rule %s but only got %d rules", expectedNum, len(doc.Chapters))
			continue
		}
		if doc.Chapters[i].Number != expectedNum {
			t.Errorf("Rule %d: expected number %q, got %q", i+1, expectedNum, doc.Chapters[i].Number)
		}
	}

	// Verify Rule I has title "THE SPEAKER"
	if len(doc.Chapters) > 0 {
		rule1Title := doc.Chapters[0].Title
		if !strings.Contains(rule1Title, "SPEAKER") {
			t.Errorf("Rule I title should contain 'SPEAKER', got %q", rule1Title)
		}
	}

	// Count total articles (clauses) across all rules
	totalArticles := 0
	for _, chapter := range doc.Chapters {
		totalArticles += len(chapter.Articles)
	}
	t.Logf("Total clauses (articles) extracted: %d", totalArticles)

	// We should have a significant number of clauses
	if totalArticles < 20 {
		t.Errorf("Expected at least 20 clauses, got %d", totalArticles)
	}

	// Rule I should have clauses about the Speaker
	if len(doc.Chapters) > 0 && len(doc.Chapters[0].Articles) > 0 {
		firstClause := doc.Chapters[0].Articles[0]
		if firstClause.Number != 1 {
			t.Errorf("First clause of Rule I should be number 1, got %d", firstClause.Number)
		}
		t.Logf("Rule I, clause 1 title: %q", firstClause.Title)
		t.Logf("Rule I, clause 1 text (first 100 chars): %q", truncateText(firstClause.Text, 100))
	}

	// Log rule overview
	for _, chapter := range doc.Chapters {
		t.Logf("  Rule %s: %q (%d clauses)", chapter.Number, chapter.Title, len(chapter.Articles))
	}
}

func TestHouseRulesCrossReferences(t *testing.T) {
	// Load pattern registry
	registry := pattern.NewRegistry()
	patternsDir := "../../patterns"
	if _, err := os.Stat(patternsDir); err != nil {
		t.Skipf("patterns directory not found at %s: %v", patternsDir, err)
	}
	if err := registry.LoadDirectory(patternsDir); err != nil {
		t.Fatalf("Failed to load patterns: %v", err)
	}

	// Load the actual House Rules file
	filePath := "../../testdata/house-rules-119th.txt"
	if _, err := os.Stat(filePath); err != nil {
		t.Skipf("House Rules file not found at %s: %v", filePath, err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("Failed to open House Rules file: %v", err)
	}
	defer file.Close()

	parser := NewParserWithRegistry(registry)

	doc, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Failed to parse House Rules: %v", err)
	}

	// Extract cross-references
	refExtractor := NewReferenceExtractor()
	allRefs := refExtractor.ExtractFromDocument(doc)

	// Count reference types
	ruleRefs := 0
	clauseOfRuleRefs := 0
	otherRefs := 0
	for _, ref := range allRefs {
		if ref.Target == TargetChapter && strings.HasPrefix(ref.Identifier, "Rule ") {
			ruleRefs++
		} else if ref.Target == TargetArticle && strings.Contains(ref.Identifier, "Rule ") {
			clauseOfRuleRefs++
		} else {
			otherRefs++
		}
	}

	t.Logf("Total cross-references: %d", len(allRefs))
	t.Logf("  Rule references (e.g., 'rule X'): %d", ruleRefs)
	t.Logf("  Clause-of-rule references (e.g., 'clause 5 of rule X'): %d", clauseOfRuleRefs)
	t.Logf("  Other references: %d", otherRefs)

	// We should detect some cross-references
	if len(allRefs) == 0 {
		t.Error("Expected at least some cross-references in the House Rules")
	}

	// Log first few references for verification
	for i, ref := range allRefs {
		if i >= 10 {
			break
		}
		t.Logf("  ref %d: %q (target=%s, identifier=%s, source=article %d)",
			i, ref.RawText, ref.Target, ref.Identifier, ref.SourceArticle)
	}
}

func truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength] + "..."
}
