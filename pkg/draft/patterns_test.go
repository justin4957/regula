package draft

import (
	"strings"
	"testing"
)

func TestNewRecognizer(t *testing.T) {
	recognizer := NewRecognizer()

	if recognizer.uscCitationPattern == nil {
		t.Error("uscCitationPattern is nil")
	}
	if recognizer.sectionOfTitlePattern == nil {
		t.Error("sectionOfTitlePattern is nil")
	}
	if recognizer.titleOfUSCPattern == nil {
		t.Error("titleOfUSCPattern is nil")
	}
	if recognizer.strikeInsertPattern == nil {
		t.Error("strikeInsertPattern is nil")
	}
	if recognizer.repealPattern == nil {
		t.Error("repealPattern is nil")
	}
	if recognizer.addNewSectionPattern == nil {
		t.Error("addNewSectionPattern is nil")
	}
	if recognizer.addAtEndPattern == nil {
		t.Error("addAtEndPattern is nil")
	}
	if recognizer.redesignatePattern == nil {
		t.Error("redesignatePattern is nil")
	}
	if recognizer.tableOfContentsPattern == nil {
		t.Error("tableOfContentsPattern is nil")
	}
	if recognizer.isAmendedPattern == nil {
		t.Error("isAmendedPattern is nil")
	}
	if recognizer.numberedClausePattern == nil {
		t.Error("numberedClausePattern is nil")
	}
	if recognizer.letteredClausePattern == nil {
		t.Error("letteredClausePattern is nil")
	}
	if recognizer.inSubsectionPattern == nil {
		t.Error("inSubsectionPattern is nil")
	}
	if recognizer.paragraphRefPattern == nil {
		t.Error("paragraphRefPattern is nil")
	}
	if recognizer.quotedTextPattern == nil {
		t.Error("quotedTextPattern is nil")
	}
}

func TestClassifyAmendmentType(t *testing.T) {
	recognizer := NewRecognizer()

	testCases := []struct {
		name     string
		text     string
		wantType AmendmentType
	}{
		{
			name:     "strike and insert",
			text:     `by striking "13" and inserting "16"`,
			wantType: AmendStrikeInsert,
		},
		{
			name:     "strike and insert with dollar amounts",
			text:     `by striking "$50,000" and inserting "$100,000"`,
			wantType: AmendStrikeInsert,
		},
		{
			name:     "repeal",
			text:     "Section 5 is repealed",
			wantType: AmendRepeal,
		},
		{
			name:     "hereby repealed",
			text:     "such subsection is hereby repealed",
			wantType: AmendRepeal,
		},
		{
			name:     "add new section",
			text:     "by inserting after section 3 the following new section",
			wantType: AmendAddNewSection,
		},
		{
			name:     "add at end",
			text:     "by adding at the end the following",
			wantType: AmendAddAtEnd,
		},
		{
			name:     "add at end new subsection",
			text:     "by adding at the end the following new subsection",
			wantType: AmendAddAtEnd,
		},
		{
			name:     "redesignate paragraph",
			text:     "by redesignating paragraph (3) as paragraph (4)",
			wantType: AmendRedesignate,
		},
		{
			name:     "redesignate subsection",
			text:     "by redesignating subsection (a) as subsection (b)",
			wantType: AmendRedesignate,
		},
		{
			name:     "table of contents",
			text:     "The table of contents of such Act is amended",
			wantType: AmendTableOfContents,
		},
		{
			name:     "no match",
			text:     "The Secretary shall conduct a study",
			wantType: "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gotType := recognizer.ClassifyAmendmentType(testCase.text)
			if gotType != testCase.wantType {
				t.Errorf("ClassifyAmendmentType(%q) = %q, want %q", testCase.text, gotType, testCase.wantType)
			}
		})
	}
}

func TestParseTargetReference(t *testing.T) {
	recognizer := NewRecognizer()

	testCases := []struct {
		name           string
		text           string
		wantTitle      string
		wantSection    string
		wantSubsection string
		wantErr        bool
	}{
		{
			name:           "USC citation",
			text:           "Section 1303 of the Act (15 U.S.C. 6502) is amended",
			wantTitle:      "15",
			wantSection:    "6502",
			wantSubsection: "",
		},
		{
			name:           "USC citation with subsection",
			text:           "Section 1306(d) of the Act (15 U.S.C. 6505(d)) is amended",
			wantTitle:      "15",
			wantSection:    "6505",
			wantSubsection: "(d)",
		},
		{
			name:           "title comma United States Code",
			text:           "Section 5 of title 42, United States Code, is amended",
			wantTitle:      "42",
			wantSection:    "5",
			wantSubsection: "",
		},
		{
			name:           "title of the United States Code",
			text:           "Section 362 of title 11 of the United States Code is amended",
			wantTitle:      "11",
			wantSection:    "362",
			wantSubsection: "",
		},
		{
			name:    "no reference",
			text:    "The Secretary shall conduct a study",
			wantErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gotTitle, gotSection, gotSubsection, err := recognizer.ParseTargetReference(testCase.text)
			if testCase.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotTitle != testCase.wantTitle {
				t.Errorf("title = %q, want %q", gotTitle, testCase.wantTitle)
			}
			if gotSection != testCase.wantSection {
				t.Errorf("section = %q, want %q", gotSection, testCase.wantSection)
			}
			if gotSubsection != testCase.wantSubsection {
				t.Errorf("subsection = %q, want %q", gotSubsection, testCase.wantSubsection)
			}
		})
	}
}

func TestExtractAmendments_StrikeInsert(t *testing.T) {
	recognizer := NewRecognizer()

	sectionText := `Section 5(a) of title 42, United States Code (42 U.S.C. 1396a(a)) is amended by striking "30 days" and inserting "60 days".`

	amendments, extractErr := recognizer.ExtractAmendments(sectionText)
	if extractErr != nil {
		t.Fatalf("unexpected error: %v", extractErr)
	}
	if len(amendments) != 1 {
		t.Fatalf("got %d amendments, want 1", len(amendments))
	}

	amendment := amendments[0]
	if amendment.Type != AmendStrikeInsert {
		t.Errorf("Type = %q, want %q", amendment.Type, AmendStrikeInsert)
	}
	if amendment.TargetTitle != "42" {
		t.Errorf("TargetTitle = %q, want %q", amendment.TargetTitle, "42")
	}
	if amendment.TargetSection != "1396a" {
		t.Errorf("TargetSection = %q, want %q", amendment.TargetSection, "1396a")
	}
	if amendment.StrikeText != "30 days" {
		t.Errorf("StrikeText = %q, want %q", amendment.StrikeText, "30 days")
	}
	if amendment.InsertText != "60 days" {
		t.Errorf("InsertText = %q, want %q", amendment.InsertText, "60 days")
	}
}

func TestExtractAmendments_Repeal(t *testing.T) {
	recognizer := NewRecognizer()

	testCases := []struct {
		name           string
		sectionText    string
		wantTitle      string
		wantSection    string
		wantSubsection string
	}{
		{
			name:        "section repeal",
			sectionText: `Section 203 of title 50, United States Code (50 U.S.C. 1403) is repealed.`,
			wantTitle:   "50",
			wantSection: "1403",
		},
		{
			name:           "hereby repealed with subsection",
			sectionText:    `Section 8(c) of title 12, United States Code (12 U.S.C. 1828(c)) is hereby repealed.`,
			wantTitle:      "12",
			wantSection:    "1828",
			wantSubsection: "(c)",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			amendments, extractErr := recognizer.ExtractAmendments(testCase.sectionText)
			if extractErr != nil {
				t.Fatalf("unexpected error: %v", extractErr)
			}
			if len(amendments) != 1 {
				t.Fatalf("got %d amendments, want 1", len(amendments))
			}
			if amendments[0].Type != AmendRepeal {
				t.Errorf("Type = %q, want %q", amendments[0].Type, AmendRepeal)
			}
			if amendments[0].TargetTitle != testCase.wantTitle {
				t.Errorf("TargetTitle = %q, want %q", amendments[0].TargetTitle, testCase.wantTitle)
			}
			if amendments[0].TargetSection != testCase.wantSection {
				t.Errorf("TargetSection = %q, want %q", amendments[0].TargetSection, testCase.wantSection)
			}
			if amendments[0].TargetSubsection != testCase.wantSubsection {
				t.Errorf("TargetSubsection = %q, want %q", amendments[0].TargetSubsection, testCase.wantSubsection)
			}
		})
	}
}

func TestExtractAmendments_AddNewSection(t *testing.T) {
	recognizer := NewRecognizer()

	sectionText := `Title 11, United States Code (11 U.S.C. 101 et seq.) is amended by inserting after section 523 the following new section: "Sec. 523A. Additional exceptions."`

	amendments, extractErr := recognizer.ExtractAmendments(sectionText)
	if extractErr != nil {
		t.Fatalf("unexpected error: %v", extractErr)
	}
	if len(amendments) != 1 {
		t.Fatalf("got %d amendments, want 1", len(amendments))
	}
	if amendments[0].Type != AmendAddNewSection {
		t.Errorf("Type = %q, want %q", amendments[0].Type, AmendAddNewSection)
	}
	if amendments[0].TargetTitle != "11" {
		t.Errorf("TargetTitle = %q, want %q", amendments[0].TargetTitle, "11")
	}
}

func TestExtractAmendments_AddAtEnd(t *testing.T) {
	recognizer := NewRecognizer()

	sectionText := `Section 10 of title 18, United States Code (18 U.S.C. 1030) is amended by adding at the end the following: "(g) ENHANCED PENALTIES.--Any person who violates this section shall be fined."`

	amendments, extractErr := recognizer.ExtractAmendments(sectionText)
	if extractErr != nil {
		t.Fatalf("unexpected error: %v", extractErr)
	}
	if len(amendments) != 1 {
		t.Fatalf("got %d amendments, want 1", len(amendments))
	}
	if amendments[0].Type != AmendAddAtEnd {
		t.Errorf("Type = %q, want %q", amendments[0].Type, AmendAddAtEnd)
	}
	if amendments[0].TargetTitle != "18" {
		t.Errorf("TargetTitle = %q, want %q", amendments[0].TargetTitle, "18")
	}
	if amendments[0].TargetSection != "1030" {
		t.Errorf("TargetSection = %q, want %q", amendments[0].TargetSection, "1030")
	}
}

func TestExtractAmendments_Redesignate(t *testing.T) {
	recognizer := NewRecognizer()

	sectionText := `Section 5(b) of title 42, United States Code (42 U.S.C. 1305(b)) is amended by redesignating paragraph (3) as paragraph (4).`

	amendments, extractErr := recognizer.ExtractAmendments(sectionText)
	if extractErr != nil {
		t.Fatalf("unexpected error: %v", extractErr)
	}
	if len(amendments) != 1 {
		t.Fatalf("got %d amendments, want 1", len(amendments))
	}
	if amendments[0].Type != AmendRedesignate {
		t.Errorf("Type = %q, want %q", amendments[0].Type, AmendRedesignate)
	}
	if amendments[0].TargetTitle != "42" {
		t.Errorf("TargetTitle = %q, want %q", amendments[0].TargetTitle, "42")
	}
	if amendments[0].StrikeText != "(3)" {
		t.Errorf("StrikeText = %q, want %q", amendments[0].StrikeText, "(3)")
	}
	if amendments[0].InsertText != "(4)" {
		t.Errorf("InsertText = %q, want %q", amendments[0].InsertText, "(4)")
	}
}

func TestExtractAmendments_MultipleInOneSection(t *testing.T) {
	recognizer := NewRecognizer()

	// Modeled on section 3 of hr1234.txt with compound amendments
	sectionText := `    (a) IN GENERAL.--Section 1303 of the Children's Online Privacy
Protection Act of 1998 (15 U.S.C. 6502) is amended--
        (1) intes in subsection (b)--
            (A) in paragraph (1), by striking "13" and inserting "16";
            (B) in paragraph (2), by adding at the end the following:
                "(C) PROHIBITION ON TARGETED ADVERTISING.--An operator
            shall not engage in targeted advertising directed at a
            child whose age the operator has actual knowledge of."; and
        (2) by adding at the end the following new subsection:
    "(e) DATA MINIMIZATION.--An operator shall--
        "(1) limit the collection of personal information from
    children to what is strictly necessary for the activity; and
        "(2) delete personal information collected from a child
    after a reasonable period not to exceed 2 years.".`

	amendments, extractErr := recognizer.ExtractAmendments(sectionText)
	if extractErr != nil {
		t.Fatalf("unexpected error: %v", extractErr)
	}

	if len(amendments) < 3 {
		t.Fatalf("got %d amendments, want at least 3", len(amendments))
	}

	// First amendment: strike "13" insert "16" targeting (b)(1)
	if amendments[0].Type != AmendStrikeInsert {
		t.Errorf("amendment[0].Type = %q, want %q", amendments[0].Type, AmendStrikeInsert)
	}
	if amendments[0].TargetTitle != "15" {
		t.Errorf("amendment[0].TargetTitle = %q, want %q", amendments[0].TargetTitle, "15")
	}
	if amendments[0].TargetSection != "6502" {
		t.Errorf("amendment[0].TargetSection = %q, want %q", amendments[0].TargetSection, "6502")
	}
	if amendments[0].StrikeText != "13" {
		t.Errorf("amendment[0].StrikeText = %q, want %q", amendments[0].StrikeText, "13")
	}
	if amendments[0].InsertText != "16" {
		t.Errorf("amendment[0].InsertText = %q, want %q", amendments[0].InsertText, "16")
	}
	if !strings.Contains(amendments[0].TargetSubsection, "(b)") {
		t.Errorf("amendment[0].TargetSubsection should contain (b), got %q", amendments[0].TargetSubsection)
	}

	// Second amendment: add at end targeting (b)(2)
	if amendments[1].Type != AmendAddAtEnd {
		t.Errorf("amendment[1].Type = %q, want %q", amendments[1].Type, AmendAddAtEnd)
	}
	if !strings.Contains(amendments[1].TargetSubsection, "(b)") {
		t.Errorf("amendment[1].TargetSubsection should contain (b), got %q", amendments[1].TargetSubsection)
	}

	// Third amendment: add at end (new subsection)
	if amendments[2].Type != AmendAddAtEnd {
		t.Errorf("amendment[2].Type = %q, want %q", amendments[2].Type, AmendAddAtEnd)
	}
	if amendments[2].TargetTitle != "15" {
		t.Errorf("amendment[2].TargetTitle = %q, want %q", amendments[2].TargetTitle, "15")
	}
}

func TestExtractAmendments_NonAmendmentSection(t *testing.T) {
	recognizer := NewRecognizer()

	testCases := []struct {
		name        string
		sectionText string
	}{
		{
			name: "definitions section",
			sectionText: `    In this Act:
        (1) COMMISSION.--The term "Commission" means the Federal
    Trade Commission.
        (2) OPERATOR.--The term "operator" means any person who
    operates a website or online service directed to children.`,
		},
		{
			name: "effective date section",
			sectionText: `    This Act shall take effect 180 days after the date of enactment
of this Act.`,
		},
		{
			name:        "short title section",
			sectionText: `    This Act may be cited as the "Example Act of 2024".`,
		},
		{
			name:        "empty text",
			sectionText: "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			amendments, extractErr := recognizer.ExtractAmendments(testCase.sectionText)
			if extractErr != nil {
				t.Fatalf("unexpected error: %v", extractErr)
			}
			if len(amendments) != 0 {
				t.Errorf("got %d amendments, want 0", len(amendments))
			}
		})
	}
}

func TestExtractAmendments_Integration(t *testing.T) {
	bill, parseErr := ParseBillFromFile(testdataPath("hr1234.txt"))
	if parseErr != nil {
		t.Fatalf("ParseBillFromFile failed: %v", parseErr)
	}

	recognizer := NewRecognizer()

	// Section 3 (index 2) should yield 3 amendments
	t.Run("section_3_amendments", func(t *testing.T) {
		section3Amendments, extractErr := recognizer.ExtractAmendments(bill.Sections[2].RawText)
		if extractErr != nil {
			t.Fatalf("ExtractAmendments failed: %v", extractErr)
		}
		if len(section3Amendments) != 3 {
			t.Fatalf("section 3: got %d amendments, want 3", len(section3Amendments))
		}

		// First: strike_insert targeting 15 USC 6502, strike "13" insert "16"
		if section3Amendments[0].Type != AmendStrikeInsert {
			t.Errorf("amendment[0].Type = %q, want %q", section3Amendments[0].Type, AmendStrikeInsert)
		}
		if section3Amendments[0].TargetTitle != "15" {
			t.Errorf("amendment[0].TargetTitle = %q, want %q", section3Amendments[0].TargetTitle, "15")
		}
		if section3Amendments[0].TargetSection != "6502" {
			t.Errorf("amendment[0].TargetSection = %q, want %q", section3Amendments[0].TargetSection, "6502")
		}
		if section3Amendments[0].StrikeText != "13" {
			t.Errorf("amendment[0].StrikeText = %q, want %q", section3Amendments[0].StrikeText, "13")
		}
		if section3Amendments[0].InsertText != "16" {
			t.Errorf("amendment[0].InsertText = %q, want %q", section3Amendments[0].InsertText, "16")
		}

		// Second: add_at_end
		if section3Amendments[1].Type != AmendAddAtEnd {
			t.Errorf("amendment[1].Type = %q, want %q", section3Amendments[1].Type, AmendAddAtEnd)
		}

		// Third: add_at_end (new subsection)
		if section3Amendments[2].Type != AmendAddAtEnd {
			t.Errorf("amendment[2].Type = %q, want %q", section3Amendments[2].Type, AmendAddAtEnd)
		}
	})

	// Section 4 (index 3) should yield 1 amendment
	t.Run("section_4_amendments", func(t *testing.T) {
		section4Amendments, extractErr := recognizer.ExtractAmendments(bill.Sections[3].RawText)
		if extractErr != nil {
			t.Fatalf("ExtractAmendments failed: %v", extractErr)
		}
		if len(section4Amendments) != 1 {
			t.Fatalf("section 4: got %d amendments, want 1", len(section4Amendments))
		}

		amendment := section4Amendments[0]
		if amendment.Type != AmendStrikeInsert {
			t.Errorf("Type = %q, want %q", amendment.Type, AmendStrikeInsert)
		}
		if amendment.TargetTitle != "15" {
			t.Errorf("TargetTitle = %q, want %q", amendment.TargetTitle, "15")
		}
		if amendment.TargetSection != "6505" {
			t.Errorf("TargetSection = %q, want %q", amendment.TargetSection, "6505")
		}
		if amendment.TargetSubsection != "(d)" {
			t.Errorf("TargetSubsection = %q, want %q", amendment.TargetSubsection, "(d)")
		}
		if amendment.StrikeText != "$50,000" {
			t.Errorf("StrikeText = %q, want %q", amendment.StrikeText, "$50,000")
		}
		if amendment.InsertText != "$100,000" {
			t.Errorf("InsertText = %q, want %q", amendment.InsertText, "$100,000")
		}
	})

	// Sections 1, 2, 5 should yield 0 amendments each
	noAmendmentSections := []struct {
		name         string
		sectionIndex int
	}{
		{"section_1_no_amendments", 0},
		{"section_2_no_amendments", 1},
		{"section_5_no_amendments", 4},
	}

	for _, testCase := range noAmendmentSections {
		t.Run(testCase.name, func(t *testing.T) {
			sectionAmendments, extractErr := recognizer.ExtractAmendments(bill.Sections[testCase.sectionIndex].RawText)
			if extractErr != nil {
				t.Fatalf("ExtractAmendments failed: %v", extractErr)
			}
			if len(sectionAmendments) != 0 {
				t.Errorf("got %d amendments, want 0", len(sectionAmendments))
			}
		})
	}
}
