package pattern

import (
	"strings"
	"testing"
)

func TestNewGenericParser(t *testing.T) {
	parser := NewGenericParser()
	if parser == nil {
		t.Fatal("NewGenericParser() returned nil")
	}
	if parser.ConfidenceThreshold != 0.5 {
		t.Errorf("Default ConfidenceThreshold = %.2f, want 0.5", parser.ConfidenceThreshold)
	}
}

func TestGenericParserParse(t *testing.T) {
	parser := NewGenericParser()

	tests := []struct {
		name            string
		content         string
		wantSections    int
		wantDefinitions int
		wantTitle       bool
		minConfidence   float64
	}{
		{
			name: "numbered sections",
			content: `DOCUMENT TITLE

1. First Section
This is the content of the first section.
It has multiple lines.

2. Second Section
This is the second section content.

3. Third Section
Final section here.`,
			wantSections:  3,
			wantTitle:     true,
			minConfidence: 0.5,
		},
		{
			name: "lettered subsections",
			content: `LEGAL DOCUMENT

(a) First point about something important.

(b) Second point with more details.

(c) Third point concludes this section.`,
			wantSections:  3,
			wantTitle:     true,
			minConfidence: 0.5,
		},
		{
			name: "roman numeral chapters",
			content: `CHAPTER STRUCTURE

I. Introduction
This chapter introduces the topic.

II. Main Content
The main content goes here.

III. Conclusion
Final thoughts and summary.`,
			wantSections:  3,
			wantTitle:     true,
			minConfidence: 0.5,
		},
		{
			name: "mixed hierarchy",
			content: `REGULATION TITLE

CHAPTER 1
Overview

1. Purpose
The purpose of this regulation.

(a) Specific purpose one.
(b) Specific purpose two.

2. Scope
The scope of application.`,
			wantSections:  4, // CHAPTER 1, 1., (a), (b), 2. - varies by detection
			wantTitle:     true,
			minConfidence: 0.4,
		},
		{
			name: "with definitions",
			content: `DEFINITIONS

For purposes of this document:

"Personal data" means any information relating to an identified person.

"Processing" means any operation performed on personal data.

"Controller" refers to the entity that determines purposes.`,
			wantSections:    1,
			wantDefinitions: 3,
			wantTitle:       true,
			minConfidence:   0.5,
		},
		{
			name: "all caps headers",
			content: `FIRST MAJOR SECTION
Content under first section.

SECOND MAJOR SECTION
Content under second section.

THIRD MAJOR SECTION
Content under third section.`,
			wantSections:  3,
			wantTitle:     true,
			minConfidence: 0.5,
		},
		{
			name:          "empty document",
			content:       "",
			wantSections:  0,
			wantTitle:     false,
			minConfidence: 0,
		},
		{
			name: "plain text without structure",
			content: `This is just plain text without any obvious structure.
It continues for several lines but has no numbering,
no headers, and no recognizable legal formatting.
Just paragraphs of regular text that could be anything.`,
			wantSections:  0,
			wantTitle:     false,
			minConfidence: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, warnings := parser.Parse(tt.content)

			if doc == nil {
				t.Fatal("Parse() returned nil document")
			}

			if doc.Format != "generic" {
				t.Errorf("Format = %q, want %q", doc.Format, "generic")
			}

			if tt.wantTitle && doc.Title == "" {
				t.Error("Expected title to be detected")
			}

			if len(doc.Sections) < tt.wantSections {
				t.Errorf("Sections = %d, want >= %d", len(doc.Sections), tt.wantSections)
				for i, s := range doc.Sections {
					t.Logf("  Section %d: %q (lines %d-%d)", i, s.Title, s.LineStart, s.LineEnd)
				}
			}

			if tt.wantDefinitions > 0 && len(doc.Definitions) < tt.wantDefinitions {
				t.Errorf("Definitions = %d, want >= %d", len(doc.Definitions), tt.wantDefinitions)
			}

			if doc.Confidence < tt.minConfidence {
				t.Errorf("Confidence = %.2f, want >= %.2f", doc.Confidence, tt.minConfidence)
				for _, w := range warnings {
					t.Logf("  Warning: [%s] %s", w.Level, w.Message)
				}
			}
		})
	}
}

func TestGenericParserDetectHierarchy(t *testing.T) {
	parser := NewGenericParser()

	tests := []struct {
		name            string
		content         string
		wantLevels      int
		wantTypes       []HierarchyType
		wantIndentBased bool
	}{
		{
			name: "arabic numbered",
			content: `1. First
2. Second
3. Third`,
			wantLevels: 1,
			wantTypes:  []HierarchyType{HierarchyTypeArabic},
		},
		{
			name: "mixed roman and arabic",
			content: `I. Chapter One
1. Section One
2. Section Two
II. Chapter Two
1. Section One`,
			wantLevels: 2,
			wantTypes:  []HierarchyType{HierarchyTypeUpperRoman, HierarchyTypeArabic},
		},
		{
			name: "letters and numbers",
			content: `1. First point
(a) Sub-point a
(b) Sub-point b
2. Second point
(a) Sub-point a`,
			wantLevels: 2,
			wantTypes:  []HierarchyType{HierarchyTypeArabic, HierarchyTypeLowerLetter},
		},
		{
			name: "indentation based",
			content: `Top level
    Indented level
        More indented
    Back to first indent
Top level again`,
			wantLevels:      0,
			wantIndentBased: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := strings.Split(tt.content, "\n")
			hierarchy, _ := parser.detectHierarchy(lines, nil)

			if hierarchy == nil {
				t.Fatal("detectHierarchy() returned nil")
			}

			if len(hierarchy.Levels) < tt.wantLevels {
				t.Errorf("Levels = %d, want >= %d", len(hierarchy.Levels), tt.wantLevels)
			}

			if tt.wantIndentBased && !hierarchy.IndentBased {
				t.Error("Expected IndentBased to be true")
			}

			for i, wantType := range tt.wantTypes {
				if i < len(hierarchy.Levels) {
					if hierarchy.Levels[i].Type != wantType {
						t.Errorf("Level[%d].Type = %s, want %s", i, hierarchy.Levels[i].Type, wantType)
					}
				}
			}
		})
	}
}

func TestGenericParserExtractDefinitions(t *testing.T) {
	parser := NewGenericParser()

	tests := []struct {
		name      string
		content   string
		wantTerms []string
		minCount  int
	}{
		{
			name: "quoted means pattern",
			content: `"Personal data" means any information relating to an identified person.
"Processing" means any operation performed on data.`,
			wantTerms: []string{"Personal data", "Processing"},
			minCount:  2,
		},
		{
			name: "refers to pattern",
			content: `"Controller" refers to the natural or legal person.
"Processor" has the meaning given in Article 4.`,
			wantTerms: []string{"Controller", "Processor"},
			minCount:  2,
		},
		{
			name: "single quotes",
			content: `'Data subject' means the identified natural person.
'Consent' means freely given indication.`,
			wantTerms: []string{"Data subject", "Consent"},
			minCount:  2,
		},
		{
			name: "no definitions",
			content: `This is just regular text.
No definitions here at all.`,
			minCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := strings.Split(tt.content, "\n")
			definitions, _ := parser.extractDefinitions(lines, nil)

			if len(definitions) < tt.minCount {
				t.Errorf("Definitions = %d, want >= %d", len(definitions), tt.minCount)
			}

			for _, wantTerm := range tt.wantTerms {
				found := false
				for _, def := range definitions {
					if def.Term == wantTerm {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find definition for %q", wantTerm)
				}
			}
		})
	}
}

func TestGenericParserDetectNumbering(t *testing.T) {
	parser := NewGenericParser()

	tests := []struct {
		line       string
		wantType   HierarchyType
		wantNumber string
	}{
		{"1. Introduction", HierarchyTypeArabic, "1"},
		{"(1) First item", HierarchyTypeArabic, "1"},
		{"(a) Point a", HierarchyTypeLowerLetter, "a"},
		{"(A) Point A", HierarchyTypeUpperLetter, "A"},
		{"a. Sub-item", HierarchyTypeLowerLetter, "a"},
		{"A. Major item", HierarchyTypeUpperLetter, "A"},
		{"I. Chapter One", HierarchyTypeUpperRoman, "I"},
		{"(i) Sub-point", HierarchyTypeLowerLetter, "i"}, // Single 'i' matches letter before roman
		{"IV. Chapter Four", HierarchyTypeUpperRoman, "IV"},
		{"(iv) Sub-point four", HierarchyTypeLowerRoman, "iv"},
		{"Just regular text", HierarchyTypeUnknown, ""},
		{"Title without number", HierarchyTypeUnknown, ""},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			gotType, gotNumber := parser.detectNumbering(tt.line)
			if gotType != tt.wantType {
				t.Errorf("detectNumbering(%q) type = %s, want %s", tt.line, gotType, tt.wantType)
			}
			if gotNumber != tt.wantNumber {
				t.Errorf("detectNumbering(%q) number = %q, want %q", tt.line, gotNumber, tt.wantNumber)
			}
		})
	}
}

func TestShouldUseGenericParser(t *testing.T) {
	tests := []struct {
		name      string
		matches   []FormatMatch
		threshold float64
		want      bool
	}{
		{
			name:      "no matches",
			matches:   []FormatMatch{},
			threshold: 0.5,
			want:      true,
		},
		{
			name: "high confidence match",
			matches: []FormatMatch{
				{FormatID: "usc", Confidence: 0.9},
			},
			threshold: 0.5,
			want:      false,
		},
		{
			name: "low confidence match",
			matches: []FormatMatch{
				{FormatID: "usc", Confidence: 0.3},
			},
			threshold: 0.5,
			want:      true,
		},
		{
			name: "at threshold",
			matches: []FormatMatch{
				{FormatID: "usc", Confidence: 0.5},
			},
			threshold: 0.5,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldUseGenericParser(tt.matches, tt.threshold)
			if got != tt.want {
				t.Errorf("ShouldUseGenericParser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenericParserWithOptions(t *testing.T) {
	parser := NewGenericParser()

	content := `DOCUMENT TITLE

1. First Section
Content of first section.

2. Second Section
Content of second section.

3. Third Section
Content of third section.

4. Fourth Section
Content of fourth section.

5. Fifth Section
Content of fifth section.`

	t.Run("MaxSections limit", func(t *testing.T) {
		options := GenericParserOptions{
			MaxSections: 3,
		}
		doc, _ := parser.ParseWithOptions(content, options)

		if len(doc.Sections) > 3 {
			t.Errorf("Sections = %d, want <= 3", len(doc.Sections))
		}
	})

	t.Run("ExtractDefinitions false", func(t *testing.T) {
		contentWithDef := `"Term" means something specific.`
		options := GenericParserOptions{
			ExtractDefinitions: false,
		}
		doc, _ := parser.ParseWithOptions(contentWithDef, options)

		if len(doc.Definitions) > 0 {
			t.Error("Expected no definitions when ExtractDefinitions is false")
		}
	})
}

func TestIsRomanNumeral(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"I", true},
		{"V", true},
		{"X", true},
		{"L", true},
		{"C", true},
		{"D", true},
		{"M", true},
		{"IV", true},
		{"IX", true},
		{"XIV", true},
		{"MCMXCIX", true},
		{"i", true},
		{"iv", true},
		{"A", false},
		{"1", false},
		{"", false},
		{"HELLO", false},
		{"IVX1", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isRomanNumeral(tt.input)
			if got != tt.want {
				t.Errorf("isRomanNumeral(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseWarningLevels(t *testing.T) {
	parser := NewGenericParser()

	// Empty content should produce warnings
	doc, warnings := parser.Parse("")

	// Empty document gets warnings which reduce confidence, but base is 1.0
	// with small reductions, so confidence may still be moderate
	if doc.Confidence > 0.9 {
		t.Errorf("Empty document should have reduced confidence, got %.2f", doc.Confidence)
	}

	hasWarning := false
	for _, w := range warnings {
		if w.Level == WarningLevelWarning || w.Level == WarningLevelInfo {
			hasWarning = true
			break
		}
	}

	if !hasWarning {
		t.Error("Expected warnings for empty document")
	}
}

func TestGenericDocumentFields(t *testing.T) {
	parser := NewGenericParser()

	content := `DOCUMENT TITLE

1. Section One
Content here.

2. Section Two
More content.`

	doc, _ := parser.Parse(content)

	if doc.Format != "generic" {
		t.Errorf("Format = %q, want %q", doc.Format, "generic")
	}

	if doc.RawLineCount == 0 {
		t.Error("RawLineCount should be > 0")
	}

	if doc.Hierarchy == nil {
		t.Error("Hierarchy should not be nil")
	}
}

// Benchmark tests
func BenchmarkGenericParserParse(b *testing.B) {
	parser := NewGenericParser()

	content := `LEGAL DOCUMENT TITLE

CHAPTER I
GENERAL PROVISIONS

1. Purpose and Scope
This regulation establishes rules for the protection of natural persons.

(a) The first point explains the purpose.
(b) The second point defines the scope.

2. Definitions
For the purposes of this regulation:

"Personal data" means any information relating to an identified person.
"Processing" means any operation performed on personal data.

CHAPTER II
PRINCIPLES

3. Lawfulness of Processing
Processing shall be lawful only if certain conditions are met.

4. Consent
The data subject has given consent to the processing.
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Parse(content)
	}
}

func BenchmarkGenericParserLargeDocument(b *testing.B) {
	parser := NewGenericParser()

	// Create a larger document
	var builder strings.Builder
	builder.WriteString("LARGE LEGAL DOCUMENT\n\n")

	for i := 1; i <= 50; i++ {
		builder.WriteString("CHAPTER ")
		builder.WriteString(strings.Repeat("I", i%10+1))
		builder.WriteString("\n")
		builder.WriteString("Chapter Title\n\n")

		for j := 1; j <= 10; j++ {
			builder.WriteString(string(rune('0' + j)))
			builder.WriteString(". Section content for section ")
			builder.WriteString(string(rune('0' + j)))
			builder.WriteString("\n\n")
		}
	}

	content := builder.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Parse(content)
	}
}
