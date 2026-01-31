package bulk

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// --- USLM XML Structures (US Code) ---
// Minimal structs for the elements needed from the USLM schema.
// The USLM format uses: <usc> → <main> → <title> → <chapter> → <section>

// USLMDocument represents the top-level <usc> element.
type USLMDocument struct {
	XMLName xml.Name   `xml:"usc"`
	Main    USLMMain   `xml:"main"`
}

// USLMMain represents the <main> element containing the title.
type USLMMain struct {
	Title USLMTitle `xml:"title"`
}

// USLMTitle represents a <title> element in USLM XML.
type USLMTitle struct {
	Identifier string        `xml:"identifier,attr"`
	Num        string        `xml:"num"`
	Heading    string        `xml:"heading"`
	Chapters   []USLMChapter `xml:"chapter"`
	Sections   []USLMSection `xml:"section"`
}

// USLMChapter represents a <chapter> element.
type USLMChapter struct {
	Identifier string        `xml:"identifier,attr"`
	Num        string        `xml:"num"`
	Heading    string        `xml:"heading"`
	Sections   []USLMSection `xml:"section"`
	Subchapters []USLMSubchapter `xml:"subchapter"`
}

// USLMSubchapter represents a <subchapter> element.
type USLMSubchapter struct {
	Identifier string        `xml:"identifier,attr"`
	Num        string        `xml:"num"`
	Heading    string        `xml:"heading"`
	Sections   []USLMSection `xml:"section"`
}

// USLMSection represents a <section> element in USLM XML.
type USLMSection struct {
	Identifier  string           `xml:"identifier,attr"`
	Num         string           `xml:"num"`
	Heading     string           `xml:"heading"`
	Chapeau     string           `xml:"chapeau"`
	Content     USLMContent      `xml:"content"`
	Subsections []USLMSubsection `xml:"subsection"`
}

// USLMContent represents mixed content within a section.
type USLMContent struct {
	Text string `xml:",chardata"`
}

// USLMSubsection represents a <subsection> within a <section>.
type USLMSubsection struct {
	Identifier string      `xml:"identifier,attr"`
	Num        string      `xml:"num"`
	Heading    string      `xml:"heading"`
	Content    USLMContent `xml:"content"`
	Chapeau    string      `xml:"chapeau"`
}

// --- CFR XML Structures ---
// CFR XML from govinfo uses uppercase tags.

// CFRDocument represents the top-level CFR XML element.
type CFRDocument struct {
	XMLName xml.Name   `xml:"CFRDOC"`
	Titles  []CFRTitle `xml:"TITLE"`
}

// CFRTitle represents a <TITLE> element.
type CFRTitle struct {
	Parts []CFRPart `xml:"CHAPTER>PART"`
}

// CFRPart represents a <PART> element.
type CFRPart struct {
	HD       string       `xml:"HD"`
	Contents string       `xml:"CONTENTS"`
	Auth     string       `xml:"AUTH>HD"`
	Source   string       `xml:"SOURCE>HD"`
	Sections []CFRSection `xml:"SECTION"`
	Subparts []CFRSubpart `xml:"SUBPART"`
}

// CFRSubpart represents a <SUBPART> element.
type CFRSubpart struct {
	HD       string       `xml:"HD"`
	Sections []CFRSection `xml:"SECTION"`
}

// CFRSection represents a <SECTION> element.
type CFRSection struct {
	SectNo  string   `xml:"SECTNO"`
	Subject string   `xml:"SUBJECT"`
	Paras   []string `xml:"P"`
}

// --- Parsing Functions ---

// ParseUSLMXML parses USLM XML and returns the structured document.
func ParseUSLMXML(reader io.Reader) (*USLMDocument, error) {
	decoder := xml.NewDecoder(reader)
	decoder.Strict = false

	document := &USLMDocument{}
	if err := decoder.Decode(document); err != nil {
		return nil, fmt.Errorf("failed to parse USLM XML: %w", err)
	}

	return document, nil
}

// ParseCFRXML parses CFR XML and returns the structured document.
func ParseCFRXML(reader io.Reader) (*CFRDocument, error) {
	decoder := xml.NewDecoder(reader)
	decoder.Strict = false

	document := &CFRDocument{}
	if err := decoder.Decode(document); err != nil {
		return nil, fmt.Errorf("failed to parse CFR XML: %w", err)
	}

	return document, nil
}

// --- Plaintext Conversion ---

// USLMToPlaintext converts a parsed USLM document to plaintext suitable
// for the regula ingestion pipeline.
func USLMToPlaintext(document *USLMDocument) string {
	var builder strings.Builder
	title := document.Main.Title

	heading := cleanXMLText(title.Heading)
	num := cleanXMLText(title.Num)
	builder.WriteString(fmt.Sprintf("TITLE %s\n%s\n\n", num, heading))

	// Direct sections under title
	for _, section := range title.Sections {
		writeSectionPlaintext(&builder, section)
	}

	// Chapters
	for _, chapter := range title.Chapters {
		chapterHeading := cleanXMLText(chapter.Heading)
		chapterNum := cleanXMLText(chapter.Num)
		builder.WriteString(fmt.Sprintf("CHAPTER %s\n%s\n\n", chapterNum, chapterHeading))

		for _, section := range chapter.Sections {
			writeSectionPlaintext(&builder, section)
		}

		for _, subchapter := range chapter.Subchapters {
			subHeading := cleanXMLText(subchapter.Heading)
			subNum := cleanXMLText(subchapter.Num)
			builder.WriteString(fmt.Sprintf("SUBCHAPTER %s\n%s\n\n", subNum, subHeading))

			for _, section := range subchapter.Sections {
				writeSectionPlaintext(&builder, section)
			}
		}
	}

	return builder.String()
}

// CFRToPlaintext converts a parsed CFR document to plaintext.
func CFRToPlaintext(document *CFRDocument) string {
	var builder strings.Builder

	for _, title := range document.Titles {
		for _, part := range title.Parts {
			partHeading := cleanXMLText(part.HD)
			if partHeading != "" {
				builder.WriteString(fmt.Sprintf("PART %s\n\n", partHeading))
			}

			for _, section := range part.Sections {
				writeCFRSectionPlaintext(&builder, section)
			}

			for _, subpart := range part.Subparts {
				subpartHeading := cleanXMLText(subpart.HD)
				if subpartHeading != "" {
					builder.WriteString(fmt.Sprintf("Subpart %s\n\n", subpartHeading))
				}
				for _, section := range subpart.Sections {
					writeCFRSectionPlaintext(&builder, section)
				}
			}
		}
	}

	return builder.String()
}

// writeSectionPlaintext writes a USLM section as plaintext.
func writeSectionPlaintext(builder *strings.Builder, section USLMSection) {
	num := cleanXMLText(section.Num)
	heading := cleanXMLText(section.Heading)

	if num != "" || heading != "" {
		builder.WriteString(fmt.Sprintf("Section %s %s\n", num, heading))
	}

	if chapeau := cleanXMLText(section.Chapeau); chapeau != "" {
		builder.WriteString(chapeau + "\n")
	}

	if content := cleanXMLText(section.Content.Text); content != "" {
		builder.WriteString(content + "\n")
	}

	for _, subsection := range section.Subsections {
		subNum := cleanXMLText(subsection.Num)
		subContent := cleanXMLText(subsection.Content.Text)
		subChapeau := cleanXMLText(subsection.Chapeau)

		if subNum != "" {
			builder.WriteString(fmt.Sprintf("  %s ", subNum))
		}
		if subChapeau != "" {
			builder.WriteString(subChapeau)
		}
		if subContent != "" {
			builder.WriteString(subContent)
		}
		builder.WriteString("\n")
	}

	builder.WriteString("\n")
}

// writeCFRSectionPlaintext writes a CFR section as plaintext.
func writeCFRSectionPlaintext(builder *strings.Builder, section CFRSection) {
	sectNo := cleanXMLText(section.SectNo)
	subject := cleanXMLText(section.Subject)

	if sectNo != "" || subject != "" {
		builder.WriteString(fmt.Sprintf("Section %s %s\n", sectNo, subject))
	}

	for _, paragraph := range section.Paras {
		text := cleanXMLText(paragraph)
		if text != "" {
			builder.WriteString(text + "\n")
		}
	}

	builder.WriteString("\n")
}

// cleanXMLText cleans up text extracted from XML, normalizing whitespace.
func cleanXMLText(text string) string {
	text = strings.TrimSpace(text)
	// Collapse internal whitespace
	fields := strings.Fields(text)
	return strings.Join(fields, " ")
}
