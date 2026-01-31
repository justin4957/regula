package bulk

import (
	"strings"
	"testing"
)

const testUSLMXML = `<?xml version="1.0" encoding="UTF-8"?>
<usc>
  <main>
    <title identifier="/us/usc/t4">
      <num>4</num>
      <heading>Flag and Seal, Seat of Government, and the States</heading>
      <chapter identifier="/us/usc/t4/ch1">
        <num>1</num>
        <heading>The Flag</heading>
        <section identifier="/us/usc/t4/s1">
          <num>1</num>
          <heading>Flag; stripes and stars on</heading>
          <content>The flag of the United States shall be thirteen horizontal stripes.</content>
          <subsection identifier="/us/usc/t4/s1/a">
            <num>(a)</num>
            <content>Additional stars shall be added on admission of new States.</content>
          </subsection>
        </section>
        <section identifier="/us/usc/t4/s2">
          <num>2</num>
          <heading>Same; additional stars</heading>
          <content>On the admission of a new State, one star shall be added.</content>
        </section>
      </chapter>
      <chapter identifier="/us/usc/t4/ch2">
        <num>2</num>
        <heading>The Seal</heading>
        <subchapter identifier="/us/usc/t4/ch2/schI">
          <num>I</num>
          <heading>General Provisions</heading>
          <section identifier="/us/usc/t4/s41">
            <num>41</num>
            <heading>Seal of the United States</heading>
            <content>The seal heretofore used by the United States shall be the seal.</content>
          </section>
        </subchapter>
      </chapter>
    </title>
  </main>
</usc>`

const testCFRXML = `<?xml version="1.0" encoding="UTF-8"?>
<CFRDOC>
  <TITLE>
    <CHAPTER>
      <PART>
        <HD>PART 1 - GENERAL PROVISIONS</HD>
        <SECTION>
          <SECTNO>1.1</SECTNO>
          <SUBJECT>Purpose</SUBJECT>
          <P>This part establishes general provisions.</P>
          <P>It applies to all regulated entities.</P>
        </SECTION>
        <SECTION>
          <SECTNO>1.2</SECTNO>
          <SUBJECT>Definitions</SUBJECT>
          <P>As used in this part, the term "agency" means any department.</P>
        </SECTION>
        <SUBPART>
          <HD>Subpart A - Administrative Matters</HD>
          <SECTION>
            <SECTNO>1.10</SECTNO>
            <SUBJECT>Administrative procedures</SUBJECT>
            <P>Procedures for administrative actions.</P>
          </SECTION>
        </SUBPART>
      </PART>
    </CHAPTER>
  </TITLE>
</CFRDOC>`

func TestParseUSLMXML(t *testing.T) {
	reader := strings.NewReader(testUSLMXML)
	document, err := ParseUSLMXML(reader)
	if err != nil {
		t.Fatalf("failed to parse USLM XML: %v", err)
	}

	title := document.Main.Title
	if title.Num != "4" {
		t.Errorf("expected title num '4', got %q", title.Num)
	}
	if !strings.Contains(title.Heading, "Flag and Seal") {
		t.Errorf("expected heading to contain 'Flag and Seal', got %q", title.Heading)
	}
	if len(title.Chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(title.Chapters))
	}

	chapter1 := title.Chapters[0]
	if chapter1.Num != "1" {
		t.Errorf("expected chapter 1, got %q", chapter1.Num)
	}
	if len(chapter1.Sections) != 2 {
		t.Fatalf("expected 2 sections in chapter 1, got %d", len(chapter1.Sections))
	}

	section1 := chapter1.Sections[0]
	if section1.Num != "1" {
		t.Errorf("expected section num '1', got %q", section1.Num)
	}
	if !strings.Contains(section1.Content.Text, "thirteen horizontal stripes") {
		t.Errorf("expected section content about stripes, got %q", section1.Content.Text)
	}
	if len(section1.Subsections) != 1 {
		t.Fatalf("expected 1 subsection, got %d", len(section1.Subsections))
	}
	if section1.Subsections[0].Num != "(a)" {
		t.Errorf("expected subsection num '(a)', got %q", section1.Subsections[0].Num)
	}

	chapter2 := title.Chapters[1]
	if len(chapter2.Subchapters) != 1 {
		t.Fatalf("expected 1 subchapter, got %d", len(chapter2.Subchapters))
	}
	if chapter2.Subchapters[0].Num != "I" {
		t.Errorf("expected subchapter num 'I', got %q", chapter2.Subchapters[0].Num)
	}
}

func TestParseCFRXML(t *testing.T) {
	reader := strings.NewReader(testCFRXML)
	document, err := ParseCFRXML(reader)
	if err != nil {
		t.Fatalf("failed to parse CFR XML: %v", err)
	}

	if len(document.Titles) != 1 {
		t.Fatalf("expected 1 title, got %d", len(document.Titles))
	}

	title := document.Titles[0]
	if len(title.Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(title.Parts))
	}

	part := title.Parts[0]
	if !strings.Contains(part.HD, "GENERAL PROVISIONS") {
		t.Errorf("expected part heading to contain 'GENERAL PROVISIONS', got %q", part.HD)
	}
	if len(part.Sections) != 2 {
		t.Errorf("expected 2 direct sections, got %d", len(part.Sections))
	}
	if len(part.Subparts) != 1 {
		t.Errorf("expected 1 subpart, got %d", len(part.Subparts))
	}

	section := part.Sections[0]
	if section.SectNo != "1.1" {
		t.Errorf("expected section number '1.1', got %q", section.SectNo)
	}
	if section.Subject != "Purpose" {
		t.Errorf("expected subject 'Purpose', got %q", section.Subject)
	}
	if len(section.Paras) != 2 {
		t.Errorf("expected 2 paragraphs, got %d", len(section.Paras))
	}
}

func TestUSLMToPlaintext(t *testing.T) {
	reader := strings.NewReader(testUSLMXML)
	document, err := ParseUSLMXML(reader)
	if err != nil {
		t.Fatalf("failed to parse USLM XML: %v", err)
	}

	plaintext := USLMToPlaintext(document)

	expectedSubstrings := []string{
		"TITLE 4",
		"Flag and Seal",
		"CHAPTER 1",
		"The Flag",
		"Section 1 Flag; stripes and stars on",
		"thirteen horizontal stripes",
		"(a)",
		"Additional stars",
		"Section 2 Same; additional stars",
		"CHAPTER 2",
		"The Seal",
		"SUBCHAPTER I",
		"General Provisions",
		"Section 41 Seal of the United States",
	}

	for _, expected := range expectedSubstrings {
		if !strings.Contains(plaintext, expected) {
			t.Errorf("expected plaintext to contain %q\nGot:\n%s", expected, plaintext)
		}
	}
}

func TestCFRToPlaintext(t *testing.T) {
	reader := strings.NewReader(testCFRXML)
	document, err := ParseCFRXML(reader)
	if err != nil {
		t.Fatalf("failed to parse CFR XML: %v", err)
	}

	plaintext := CFRToPlaintext(document)

	expectedSubstrings := []string{
		"PART",
		"GENERAL PROVISIONS",
		"Section 1.1 Purpose",
		"general provisions",
		"regulated entities",
		"Section 1.2 Definitions",
		"Subpart",
		"Administrative Matters",
		"Section 1.10 Administrative procedures",
	}

	for _, expected := range expectedSubstrings {
		if !strings.Contains(plaintext, expected) {
			t.Errorf("expected plaintext to contain %q\nGot:\n%s", expected, plaintext)
		}
	}
}

func TestCleanXMLText(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trims whitespace",
			input:    "  hello world  ",
			expected: "hello world",
		},
		{
			name:     "collapses internal whitespace",
			input:    "hello    world   test",
			expected: "hello world test",
		},
		{
			name:     "handles newlines and tabs",
			input:    "hello\n\t\tworld\ntest",
			expected: "hello world test",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \t\n   ",
			expected: "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := cleanXMLText(testCase.input)
			if result != testCase.expected {
				t.Errorf("cleanXMLText(%q) = %q, want %q", testCase.input, result, testCase.expected)
			}
		})
	}
}

func TestParseUSLMXMLInvalid(t *testing.T) {
	reader := strings.NewReader("not valid xml at all")
	_, err := ParseUSLMXML(reader)
	if err == nil {
		t.Fatal("expected error for invalid XML")
	}
}

func TestParseCFRXMLInvalid(t *testing.T) {
	reader := strings.NewReader("<broken><unclosed>")
	_, err := ParseCFRXML(reader)
	if err == nil {
		t.Fatal("expected error for invalid XML")
	}
}

func TestUSLMToPlaintextEmpty(t *testing.T) {
	document := &USLMDocument{}
	plaintext := USLMToPlaintext(document)

	if !strings.Contains(plaintext, "TITLE") {
		t.Error("expected plaintext to contain 'TITLE' header even for empty document")
	}
}
