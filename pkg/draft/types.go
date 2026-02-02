// Package draft provides types and parsing for draft Congressional legislation
// in plain-text bill format (H.R. and Senate bills). It extracts structural
// elements — sections, titles, metadata — without interpreting amendment
// semantics (see issue #149 for amendment pattern recognition).
package draft

import "fmt"

// AmendmentType classifies the kind of legislative amendment. These types
// are defined here for structural completeness but are populated by the
// amendment pattern recognizer (issue #149), not by the parser.
type AmendmentType string

const (
	// AmendStrikeInsert replaces existing text with new text.
	AmendStrikeInsert AmendmentType = "strike_insert"
	// AmendRepeal removes an existing provision entirely.
	AmendRepeal AmendmentType = "repeal"
	// AmendAddNewSection inserts a new section into existing law.
	AmendAddNewSection AmendmentType = "add_new_section"
	// AmendAddAtEnd appends content to the end of an existing section.
	AmendAddAtEnd AmendmentType = "add_at_end"
	// AmendRedesignate renumbers or reletters existing provisions.
	AmendRedesignate AmendmentType = "redesignate"
	// AmendTableOfContents updates a table of contents to reflect changes.
	AmendTableOfContents AmendmentType = "table_of_contents"
)

// Amendment represents a single amendment directive within a draft bill section.
// The parser initializes Amendments slices as empty; the amendment pattern
// recognizer (issue #149) populates them.
type Amendment struct {
	Type             AmendmentType `json:"type"`
	TargetTitle      string        `json:"target_title"`
	TargetSection    string        `json:"target_section"`
	TargetSubsection string        `json:"target_subsection,omitempty"`
	StrikeText       string        `json:"strike_text,omitempty"`
	InsertText       string        `json:"insert_text,omitempty"`
	Description      string        `json:"description,omitempty"`
}

// DraftSection represents a numbered section within a draft bill. Sections
// are the primary structural unit of Congressional legislation, typically
// starting with "SECTION N." or "SEC. N." followed by a title.
type DraftSection struct {
	Number     string      `json:"number"`
	Title      string      `json:"title"`
	Amendments []Amendment `json:"amendments"`
	RawText    string      `json:"raw_text"`
}

// DraftBill represents a parsed Congressional bill with structural metadata.
// The parser extracts header information (Congress, session, bill number),
// the bill title, and individual sections with their raw text content.
type DraftBill struct {
	Filename   string          `json:"filename,omitempty"`
	Title      string          `json:"title"`
	ShortTitle string          `json:"short_title,omitempty"`
	BillNumber string          `json:"bill_number"`
	Congress   string          `json:"congress,omitempty"`
	Session    string          `json:"session,omitempty"`
	Sections   []*DraftSection `json:"sections"`
	RawText    string          `json:"raw_text"`
}

// BillStatistics provides aggregate counts for a parsed bill.
type BillStatistics struct {
	SectionCount    int `json:"section_count"`
	AmendmentCount  int `json:"amendment_count"`
	TotalCharacters int `json:"total_characters"`
}

// Statistics computes aggregate counts from the bill's sections.
func (bill *DraftBill) Statistics() BillStatistics {
	amendmentCount := 0
	for _, section := range bill.Sections {
		amendmentCount += len(section.Amendments)
	}
	return BillStatistics{
		SectionCount:    len(bill.Sections),
		AmendmentCount:  amendmentCount,
		TotalCharacters: len(bill.RawText),
	}
}

// String returns a canonical display representation of the bill.
// Format: "H.R. 1234 — Short Title (118th Congress)" or
// "S. 456 — Title" when Congress is not specified.
func (bill *DraftBill) String() string {
	displayTitle := bill.ShortTitle
	if displayTitle == "" {
		displayTitle = bill.Title
	}

	result := fmt.Sprintf("%s — %s", bill.BillNumber, displayTitle)
	if bill.Congress != "" {
		result += fmt.Sprintf(" (%s Congress)", bill.Congress)
	}
	return result
}
