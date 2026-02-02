package draft

import (
	"path/filepath"
	"strings"
	"testing"
)

func testdataPath(filename string) string {
	return filepath.Join("..", "..", "testdata", "drafts", filename)
}

func TestNewParser(t *testing.T) {
	parser := NewParser()

	if parser.congressPattern == nil {
		t.Error("congressPattern is nil")
	}
	if parser.sessionPattern == nil {
		t.Error("sessionPattern is nil")
	}
	if parser.billNumberPattern == nil {
		t.Error("billNumberPattern is nil")
	}
	if parser.titlePattern == nil {
		t.Error("titlePattern is nil")
	}
	if parser.enactingPattern == nil {
		t.Error("enactingPattern is nil")
	}
	if parser.sectionFullPattern == nil {
		t.Error("sectionFullPattern is nil")
	}
	if parser.sectionAbbrPattern == nil {
		t.Error("sectionAbbrPattern is nil")
	}
	if parser.shortTitlePattern == nil {
		t.Error("shortTitlePattern is nil")
	}
}

func TestParseFullBill(t *testing.T) {
	bill, parseErr := ParseBillFromFile(testdataPath("hr1234.txt"))
	if parseErr != nil {
		t.Fatalf("ParseBillFromFile failed: %v", parseErr)
	}

	if bill.Congress != "118th" {
		t.Errorf("Congress = %q, want %q", bill.Congress, "118th")
	}
	if bill.Session != "1st" {
		t.Errorf("Session = %q, want %q", bill.Session, "1st")
	}
	if bill.BillNumber != "H.R. 1234" {
		t.Errorf("BillNumber = %q, want %q", bill.BillNumber, "H.R. 1234")
	}
	if !strings.Contains(bill.Title, "amend title 42") {
		t.Errorf("Title should contain 'amend title 42', got %q", bill.Title)
	}
	if bill.ShortTitle != "Children's Online Privacy Protection Modernization Act" {
		t.Errorf("ShortTitle = %q, want %q", bill.ShortTitle, "Children's Online Privacy Protection Modernization Act")
	}
	if len(bill.Sections) != 5 {
		t.Fatalf("expected 5 sections, got %d", len(bill.Sections))
	}

	expectedSections := []struct {
		number string
		title  string
	}{
		{"1", "SHORT TITLE"},
		{"2", "DEFINITIONS"},
		{"3", "ENHANCED PROTECTIONS FOR CHILDREN'S ONLINE PRIVACY"},
		{"4", "ENFORCEMENT"},
		{"5", "EFFECTIVE DATE"},
	}

	for sectionIndex, expectedSection := range expectedSections {
		section := bill.Sections[sectionIndex]
		if section.Number != expectedSection.number {
			t.Errorf("section[%d].Number = %q, want %q", sectionIndex, section.Number, expectedSection.number)
		}
		if section.Title != expectedSection.title {
			t.Errorf("section[%d].Title = %q, want %q", sectionIndex, section.Title, expectedSection.title)
		}
	}
}

func TestParseMinimalBill(t *testing.T) {
	bill, parseErr := ParseBillFromFile(testdataPath("s456_minimal.txt"))
	if parseErr != nil {
		t.Fatalf("ParseBillFromFile failed: %v", parseErr)
	}

	if bill.BillNumber != "S. 456" {
		t.Errorf("BillNumber = %q, want %q", bill.BillNumber, "S. 456")
	}
	if bill.Congress != "" {
		t.Errorf("Congress should be empty for minimal bill, got %q", bill.Congress)
	}
	if bill.Session != "" {
		t.Errorf("Session should be empty for minimal bill, got %q", bill.Session)
	}
	if len(bill.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(bill.Sections))
	}
	if bill.Sections[0].Number != "1" {
		t.Errorf("section number = %q, want %q", bill.Sections[0].Number, "1")
	}
	if bill.Sections[0].Title != "STUDY ON ARTIFICIAL INTELLIGENCE IMPACT" {
		t.Errorf("section title = %q, want %q", bill.Sections[0].Title, "STUDY ON ARTIFICIAL INTELLIGENCE IMPACT")
	}
}

func TestParseBillFromFile(t *testing.T) {
	bill, parseErr := ParseBillFromFile(testdataPath("hr1234.txt"))
	if parseErr != nil {
		t.Fatalf("ParseBillFromFile failed: %v", parseErr)
	}

	if bill.Filename != testdataPath("hr1234.txt") {
		t.Errorf("Filename = %q, want %q", bill.Filename, testdataPath("hr1234.txt"))
	}
	if bill.BillNumber != "H.R. 1234" {
		t.Errorf("BillNumber = %q, want %q", bill.BillNumber, "H.R. 1234")
	}
}

func TestParseBillString(t *testing.T) {
	billText := `H. R. 9999

AN ACT
To do something.

Be it enacted by the Senate and House of Representatives of the
United States of America in Congress assembled,

SECTION 1. SHORT TITLE.

    This is the content of section 1.
`
	bill, parseErr := ParseBill(billText)
	if parseErr != nil {
		t.Fatalf("ParseBill failed: %v", parseErr)
	}

	if bill.BillNumber != "H.R. 9999" {
		t.Errorf("BillNumber = %q, want %q", bill.BillNumber, "H.R. 9999")
	}
	if len(bill.Sections) != 1 {
		t.Errorf("expected 1 section, got %d", len(bill.Sections))
	}
}

func TestParseEmptyInput(t *testing.T) {
	_, parseErr := ParseBill("")
	if parseErr == nil {
		t.Error("expected error for empty input")
	}
	if parseErr != nil && !strings.Contains(parseErr.Error(), "empty input") {
		t.Errorf("error should mention 'empty input', got: %v", parseErr)
	}
}

func TestParseNoSections(t *testing.T) {
	billText := `H. R. 5555

AN ACT
To establish a commission.

Be it enacted by the Senate and House of Representatives of the
United States of America in Congress assembled,

`
	bill, parseErr := ParseBill(billText)
	if parseErr != nil {
		t.Fatalf("ParseBill failed: %v", parseErr)
	}

	if len(bill.Sections) != 0 {
		t.Errorf("expected 0 sections, got %d", len(bill.Sections))
	}
	if bill.BillNumber != "H.R. 5555" {
		t.Errorf("BillNumber = %q, want %q", bill.BillNumber, "H.R. 5555")
	}
}

func TestSectionBoundaries(t *testing.T) {
	bill, parseErr := ParseBillFromFile(testdataPath("hr1234.txt"))
	if parseErr != nil {
		t.Fatalf("ParseBillFromFile failed: %v", parseErr)
	}

	// Section 1 should contain "may be cited as" but not "COMMISSION"
	section1Text := bill.Sections[0].RawText
	if !strings.Contains(section1Text, "may be cited as") {
		t.Error("section 1 should contain 'may be cited as'")
	}
	if strings.Contains(section1Text, "COMMISSION") {
		t.Error("section 1 should not contain content from section 2")
	}

	// Section 2 should contain "COMMISSION" but not "may be cited as"
	section2Text := bill.Sections[1].RawText
	if !strings.Contains(section2Text, "COMMISSION") {
		t.Error("section 2 should contain 'COMMISSION'")
	}
	if strings.Contains(section2Text, "may be cited as") {
		t.Error("section 2 should not contain content from section 1")
	}

	// Section 5 (last) should contain "180 days"
	section5Text := bill.Sections[4].RawText
	if !strings.Contains(section5Text, "180 days") {
		t.Error("section 5 should contain '180 days'")
	}
}

func TestShortTitleExtraction(t *testing.T) {
	bill, parseErr := ParseBillFromFile(testdataPath("hr1234.txt"))
	if parseErr != nil {
		t.Fatalf("ParseBillFromFile failed: %v", parseErr)
	}

	if bill.ShortTitle != "Children's Online Privacy Protection Modernization Act" {
		t.Errorf("ShortTitle = %q, want %q", bill.ShortTitle, "Children's Online Privacy Protection Modernization Act")
	}
}

func TestShortTitleMissing(t *testing.T) {
	bill, parseErr := ParseBillFromFile(testdataPath("s456_minimal.txt"))
	if parseErr != nil {
		t.Fatalf("ParseBillFromFile failed: %v", parseErr)
	}

	if bill.ShortTitle != "" {
		t.Errorf("ShortTitle should be empty when no citation clause, got %q", bill.ShortTitle)
	}
}

func TestBillStatistics(t *testing.T) {
	bill, parseErr := ParseBillFromFile(testdataPath("hr1234.txt"))
	if parseErr != nil {
		t.Fatalf("ParseBillFromFile failed: %v", parseErr)
	}

	statistics := bill.Statistics()
	if statistics.SectionCount != 5 {
		t.Errorf("SectionCount = %d, want 5", statistics.SectionCount)
	}
	if statistics.AmendmentCount != 0 {
		t.Errorf("AmendmentCount = %d, want 0 (parser does not extract amendments)", statistics.AmendmentCount)
	}
	if statistics.TotalCharacters == 0 {
		t.Error("TotalCharacters should be > 0")
	}
}

func TestBillString(t *testing.T) {
	bill, parseErr := ParseBillFromFile(testdataPath("hr1234.txt"))
	if parseErr != nil {
		t.Fatalf("ParseBillFromFile failed: %v", parseErr)
	}

	displayString := bill.String()
	expectedSubstring := "H.R. 1234"
	if !strings.Contains(displayString, expectedSubstring) {
		t.Errorf("String() should contain %q, got %q", expectedSubstring, displayString)
	}
	if !strings.Contains(displayString, "Children's Online Privacy Protection Modernization Act") {
		t.Errorf("String() should contain short title, got %q", displayString)
	}
	if !strings.Contains(displayString, "118th Congress") {
		t.Errorf("String() should contain Congress, got %q", displayString)
	}
}

func TestBillStringWithoutCongress(t *testing.T) {
	bill, parseErr := ParseBillFromFile(testdataPath("s456_minimal.txt"))
	if parseErr != nil {
		t.Fatalf("ParseBillFromFile failed: %v", parseErr)
	}

	displayString := bill.String()
	if strings.Contains(displayString, "Congress") {
		t.Errorf("String() should not contain 'Congress' for minimal bill, got %q", displayString)
	}
	if !strings.Contains(displayString, "S. 456") {
		t.Errorf("String() should contain bill number, got %q", displayString)
	}
}

func TestAmendmentsInitializedEmpty(t *testing.T) {
	bill, parseErr := ParseBillFromFile(testdataPath("hr1234.txt"))
	if parseErr != nil {
		t.Fatalf("ParseBillFromFile failed: %v", parseErr)
	}

	for sectionIndex, section := range bill.Sections {
		if section.Amendments == nil {
			t.Errorf("section[%d].Amendments is nil, should be empty non-nil slice", sectionIndex)
		}
		if len(section.Amendments) != 0 {
			t.Errorf("section[%d].Amendments has %d entries, want 0", sectionIndex, len(section.Amendments))
		}
	}
}

func TestSectionNumberParsing(t *testing.T) {
	billText := `H. R. 7777

AN ACT
To test section parsing.

Be it enacted by the Senate and House of Representatives of the
United States of America in Congress assembled,

SECTION 1. FIRST SECTION TITLE.

    Content of section one.

SEC. 2. SECOND SECTION TITLE.

    Content of section two.

SEC. 10. TENTH SECTION TITLE.

    Content of section ten.
`
	bill, parseErr := ParseBill(billText)
	if parseErr != nil {
		t.Fatalf("ParseBill failed: %v", parseErr)
	}

	if len(bill.Sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(bill.Sections))
	}

	testCases := []struct {
		index          int
		expectedNumber string
		expectedTitle  string
	}{
		{0, "1", "FIRST SECTION TITLE"},
		{1, "2", "SECOND SECTION TITLE"},
		{2, "10", "TENTH SECTION TITLE"},
	}

	for _, testCase := range testCases {
		section := bill.Sections[testCase.index]
		if section.Number != testCase.expectedNumber {
			t.Errorf("section[%d].Number = %q, want %q", testCase.index, section.Number, testCase.expectedNumber)
		}
		if section.Title != testCase.expectedTitle {
			t.Errorf("section[%d].Title = %q, want %q", testCase.index, section.Title, testCase.expectedTitle)
		}
	}
}
