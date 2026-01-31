package bulk

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// --- USLM XML Structures (US Code) ---
// Minimal structs for the elements needed from the USLM schema.
// The USLM format uses: <uscDoc> → <main> → <title> → <chapter> → <section>
// (Older schema used <usc> as root; current uses <uscDoc> with namespace.)

// USLMDocument represents the top-level element (<uscDoc> or <usc>).
type USLMDocument struct {
	XMLName xml.Name `xml:"uscDoc"`
	Main    USLMMain `xml:"main"`
}

// USLMMain represents the <main> element containing the title.
type USLMMain struct {
	Title USLMTitle `xml:"title"`
}

// USLMNum represents a <num> element which may have both a value attribute
// and display text (e.g., <num value="26">§ 26.</num>).
type USLMNum struct {
	Value string `xml:"value,attr"`
	Text  string `xml:",chardata"`
}

// CleanValue returns the clean numeric value, preferring the value attribute
// over the display text.
func (n USLMNum) CleanValue() string {
	if n.Value != "" {
		return cleanXMLText(n.Value)
	}
	return cleanXMLText(n.Text)
}

// USLMTitle represents a <title> element in USLM XML.
type USLMTitle struct {
	Identifier string        `xml:"identifier,attr"`
	Num        USLMNum       `xml:"num"`
	Heading    string        `xml:"heading"`
	Chapters   []USLMChapter `xml:"chapter"`
	Sections   []USLMSection `xml:"section"`
}

// USLMChapter represents a <chapter> element.
type USLMChapter struct {
	Identifier  string           `xml:"identifier,attr"`
	Num         USLMNum          `xml:"num"`
	Heading     string           `xml:"heading"`
	Sections    []USLMSection    `xml:"section"`
	Subchapters []USLMSubchapter `xml:"subchapter"`
}

// USLMSubchapter represents a <subchapter> element.
type USLMSubchapter struct {
	Identifier string        `xml:"identifier,attr"`
	Num        USLMNum       `xml:"num"`
	Heading    string        `xml:"heading"`
	Sections   []USLMSection `xml:"section"`
}

// USLMSection represents a <section> element in USLM XML.
type USLMSection struct {
	Identifier  string           `xml:"identifier,attr"`
	Num         USLMNum          `xml:"num"`
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
	Num        USLMNum     `xml:"num"`
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

// namespaceStripper wraps an xml.Decoder to strip XML namespace prefixes,
// allowing struct tags to match local element names regardless of namespace.
type namespaceStripper struct {
	decoder *xml.Decoder
}

func (stripper *namespaceStripper) Token() (xml.Token, error) {
	token, err := stripper.decoder.Token()
	if err != nil {
		return token, err
	}

	switch element := token.(type) {
	case xml.StartElement:
		element.Name.Space = ""
		for i := range element.Attr {
			element.Attr[i].Name.Space = ""
		}
		return element, nil
	case xml.EndElement:
		element.Name.Space = ""
		return element, nil
	}

	return token, nil
}

// ParseUSLMXML parses USLM XML and returns the structured document.
// Handles both <uscDoc> (current USLM 1.0) and <usc> (older format)
// by stripping XML namespaces before decoding.
func ParseUSLMXML(reader io.Reader) (*USLMDocument, error) {
	rawDecoder := xml.NewDecoder(reader)
	rawDecoder.Strict = false

	decoder := xml.NewTokenDecoder(&namespaceStripper{decoder: rawDecoder})

	document := &USLMDocument{}
	if err := decoder.Decode(document); err != nil {
		// Try legacy <usc> root element
		return nil, fmt.Errorf("failed to parse USLM XML: %w", err)
	}

	return document, nil
}

// ParseCFRXML parses CFR XML and returns the structured document.
func ParseCFRXML(reader io.Reader) (*CFRDocument, error) {
	rawDecoder := xml.NewDecoder(reader)
	rawDecoder.Strict = false

	decoder := xml.NewTokenDecoder(&namespaceStripper{decoder: rawDecoder})

	document := &CFRDocument{}
	if err := decoder.Decode(document); err != nil {
		return nil, fmt.Errorf("failed to parse CFR XML: %w", err)
	}

	return document, nil
}

// --- Plaintext Conversion ---

// USLMToPlaintext converts a parsed USLM document to plaintext suitable
// for the regula ingestion pipeline. Output uses clean numeric identifiers
// from the XML value attributes to produce parser-compatible headers.
func USLMToPlaintext(document *USLMDocument) string {
	var builder strings.Builder
	title := document.Main.Title

	heading := cleanXMLText(title.Heading)
	num := title.Num.CleanValue()
	builder.WriteString(fmt.Sprintf("TITLE %s\n%s\n\n", num, heading))

	// Direct sections under title
	for _, section := range title.Sections {
		writeSectionPlaintext(&builder, section)
	}

	// Chapters
	for _, chapter := range title.Chapters {
		chapterHeading := cleanXMLText(chapter.Heading)
		chapterNum := chapter.Num.CleanValue()
		// Extract clean chapter number from identifier if Num still has decorations
		if chapterNum == "" {
			chapterNum = extractIdentifierSuffix(chapter.Identifier)
		}
		builder.WriteString(fmt.Sprintf("CHAPTER %s\n%s\n\n", chapterNum, chapterHeading))

		for _, section := range chapter.Sections {
			writeSectionPlaintext(&builder, section)
		}

		for _, subchapter := range chapter.Subchapters {
			subHeading := cleanXMLText(subchapter.Heading)
			subNum := subchapter.Num.CleanValue()
			if subNum == "" {
				subNum = extractIdentifierSuffix(subchapter.Identifier)
			}
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
// Uses clean numeric values from XML value attributes, falling back to
// identifier-derived section numbers for parser compatibility.
func writeSectionPlaintext(builder *strings.Builder, section USLMSection) {
	num := section.Num.CleanValue()
	// Fall back to identifier suffix (e.g., "/us/usc/t42/s26" → "26")
	if num == "" {
		num = extractIdentifierSuffix(section.Identifier)
	}
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
		subNum := subsection.Num.CleanValue()
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

// extractIdentifierSuffix extracts the trailing segment from a USLM identifier
// path. For example, "/us/usc/t42/s26" returns "26", "/us/usc/t42/ch1" returns "1".
// The prefix letter (s for section, ch for chapter, etc.) is stripped.
func extractIdentifierSuffix(identifier string) string {
	if identifier == "" {
		return ""
	}
	lastSlash := strings.LastIndex(identifier, "/")
	if lastSlash < 0 || lastSlash >= len(identifier)-1 {
		return ""
	}
	suffix := identifier[lastSlash+1:]
	// Strip known prefixes: s (section), ch (chapter), sch (subchapter)
	for _, prefix := range []string{"sch", "ch", "s"} {
		if strings.HasPrefix(suffix, prefix) {
			return suffix[len(prefix):]
		}
	}
	return suffix
}

// cleanXMLText cleans up text extracted from XML, normalizing whitespace.
func cleanXMLText(text string) string {
	text = strings.TrimSpace(text)
	// Collapse internal whitespace
	fields := strings.Fields(text)
	return strings.Join(fields, " ")
}
