package deliberation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DocumentStatus represents the current status of a working paper.
type DocumentStatus int

const (
	// StatusDraft indicates the document is a working draft.
	StatusDraft DocumentStatus = iota
	// StatusUnderReview indicates the document is being reviewed.
	StatusUnderReview
	// StatusRevised indicates the document has been revised.
	StatusRevised
	// StatusFinal indicates the document is finalized.
	StatusFinal
	// StatusSuperseded indicates the document has been replaced by a newer version.
	StatusSuperseded
	// StatusWithdrawn indicates the document has been withdrawn.
	StatusWithdrawn
)

// String returns a human-readable label for the document status.
func (s DocumentStatus) String() string {
	switch s {
	case StatusDraft:
		return "draft"
	case StatusUnderReview:
		return "under_review"
	case StatusRevised:
		return "revised"
	case StatusFinal:
		return "final"
	case StatusSuperseded:
		return "superseded"
	case StatusWithdrawn:
		return "withdrawn"
	default:
		return "unknown"
	}
}

// ParseDocumentStatus converts a string to a DocumentStatus.
func ParseDocumentStatus(s string) DocumentStatus {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "draft":
		return StatusDraft
	case "under_review", "under review", "review":
		return StatusUnderReview
	case "revised", "rev":
		return StatusRevised
	case "final", "adopted", "approved":
		return StatusFinal
	case "superseded", "replaced":
		return StatusSuperseded
	case "withdrawn", "cancelled":
		return StatusWithdrawn
	default:
		return StatusDraft
	}
}

// DocumentType classifies the type of working paper.
type DocumentType int

const (
	// TypeWorkingPaper is a general working paper.
	TypeWorkingPaper DocumentType = iota
	// TypeDraftRegulation is a draft regulation or directive.
	TypeDraftRegulation
	// TypePositionPaper is a position paper.
	TypePositionPaper
	// TypeDiscussionPaper is a discussion document.
	TypeDiscussionPaper
	// TypeAnnotatedAgenda is an annotated agenda.
	TypeAnnotatedAgenda
	// TypeWorkPlan is a work/resourcing plan.
	TypeWorkPlan
	// TypeReport is a report or assessment.
	TypeReport
)

// String returns a human-readable label for the document type.
func (t DocumentType) String() string {
	switch t {
	case TypeWorkingPaper:
		return "working_paper"
	case TypeDraftRegulation:
		return "draft_regulation"
	case TypePositionPaper:
		return "position_paper"
	case TypeDiscussionPaper:
		return "discussion_paper"
	case TypeAnnotatedAgenda:
		return "annotated_agenda"
	case TypeWorkPlan:
		return "work_plan"
	case TypeReport:
		return "report"
	default:
		return "unknown"
	}
}

// WorkingPaper represents a draft document that evolves through deliberation.
// Working papers may go through multiple versions as they are discussed and amended.
type WorkingPaper struct {
	// URI is the unique identifier for this document version.
	URI string `json:"uri"`

	// Identifier is the document number (e.g., "210201_work_plan_draft_en").
	Identifier string `json:"identifier"`

	// Title is the document title.
	Title string `json:"title"`

	// Type classifies the document.
	Type DocumentType `json:"type"`

	// Version holds version/revision information.
	Version Version `json:"version"`

	// Status indicates the current document status.
	Status DocumentStatus `json:"status"`

	// Author is the document author or originating body.
	Author string `json:"author,omitempty"`

	// Date is the document date.
	Date time.Time `json:"date"`

	// Language is the document language code.
	Language string `json:"language,omitempty"`

	// Sections contains the document structure.
	Sections []Section `json:"sections,omitempty"`

	// ActionPoints lists action items extracted from the document.
	ActionPoints []ActionPoint `json:"action_points,omitempty"`

	// Annotations contains comments and track changes.
	Annotations []Annotation `json:"annotations,omitempty"`

	// References lists cross-references to other documents.
	References []DocumentReference `json:"references,omitempty"`

	// Tables contains extracted table data.
	Tables []Table `json:"tables,omitempty"`

	// Annexes lists attached annexes.
	Annexes []Annex `json:"annexes,omitempty"`

	// SupersedesURI links to the previous version this one replaces.
	SupersedesURI string `json:"supersedes_uri,omitempty"`

	// SupersededByURI links to the newer version that replaces this one.
	SupersededByURI string `json:"superseded_by_uri,omitempty"`

	// DiscussedAt lists URIs of meetings where this document was discussed.
	DiscussedAt []string `json:"discussed_at,omitempty"`

	// AdoptedAt is the URI of the meeting where this version was adopted.
	AdoptedAt string `json:"adopted_at,omitempty"`

	// SourceDocument is the URI or path of the source file.
	SourceDocument string `json:"source_document,omitempty"`

	// ProcessURI links to the parent deliberation process.
	ProcessURI string `json:"process_uri,omitempty"`
}

// Version tracks document version information and lineage.
type Version struct {
	// Number is the version identifier (e.g., "1.0", "2", "REV1").
	Number string `json:"number"`

	// Revision is the revision number if applicable.
	Revision int `json:"revision,omitempty"`

	// Date is when this version was created.
	Date time.Time `json:"date"`

	// SupersedesID is the identifier of the version this one replaces.
	SupersedesID string `json:"supersedes_id,omitempty"`

	// Status indicates the version status.
	Status string `json:"status,omitempty"`

	// ChangeSummary describes changes from the previous version.
	ChangeSummary string `json:"change_summary,omitempty"`

	// MeetingURI links to the meeting where changes were adopted.
	MeetingURI string `json:"meeting_uri,omitempty"`
}

// Section represents a structural section within the document.
type Section struct {
	// URI is the unique identifier for this section.
	URI string `json:"uri"`

	// Number is the section number (e.g., "1", "2.1", "A").
	Number string `json:"number"`

	// Title is the section heading.
	Title string `json:"title"`

	// Level indicates the nesting level (1 = top level).
	Level int `json:"level"`

	// Text is the section content.
	Text string `json:"text"`

	// Paragraphs contains numbered paragraphs within the section.
	Paragraphs []Paragraph `json:"paragraphs,omitempty"`

	// SubSections contains nested sections.
	SubSections []Section `json:"sub_sections,omitempty"`

	// ParentURI links to the parent section if nested.
	ParentURI string `json:"parent_uri,omitempty"`

	// DocumentURI links to the parent document.
	DocumentURI string `json:"document_uri,omitempty"`
}

// Paragraph represents a numbered paragraph within a section.
type Paragraph struct {
	// Number is the paragraph number.
	Number string `json:"number"`

	// Text is the paragraph content.
	Text string `json:"text"`

	// SectionURI links to the parent section.
	SectionURI string `json:"section_uri,omitempty"`
}

// ActionPoint represents an action item extracted from the document.
type ActionPoint struct {
	// Number is the action point number.
	Number string `json:"number"`

	// Description describes the required action.
	Description string `json:"description"`

	// Assignee is the responsible party.
	Assignee string `json:"assignee,omitempty"`

	// Deadline is the target completion date.
	Deadline *time.Time `json:"deadline,omitempty"`

	// Status indicates completion status.
	Status string `json:"status,omitempty"`

	// SectionURI links to the source section.
	SectionURI string `json:"section_uri,omitempty"`
}

// Annotation represents a comment, note, or track change markup.
type Annotation struct {
	// Type classifies the annotation (comment, insertion, deletion, note).
	Type string `json:"type"`

	// Author is who made the annotation.
	Author string `json:"author,omitempty"`

	// Date is when the annotation was made.
	Date *time.Time `json:"date,omitempty"`

	// Text is the annotation content.
	Text string `json:"text"`

	// TargetText is the text being annotated.
	TargetText string `json:"target_text,omitempty"`

	// SectionNumber indicates which section contains this annotation.
	SectionNumber string `json:"section_number,omitempty"`

	// IsReservation indicates if this is a delegation reservation.
	IsReservation bool `json:"is_reservation,omitempty"`

	// Delegation identifies the delegation making a reservation.
	Delegation string `json:"delegation,omitempty"`
}

// Note: DocumentReference type is defined in parser.go

// Table represents a table extracted from the document.
type Table struct {
	// Number is the table number.
	Number string `json:"number"`

	// Title is the table caption.
	Title string `json:"title,omitempty"`

	// Headers are column headers.
	Headers []string `json:"headers,omitempty"`

	// Rows contains the table data.
	Rows [][]string `json:"rows,omitempty"`

	// SectionNumber indicates which section contains this table.
	SectionNumber string `json:"section_number,omitempty"`
}

// Annex represents an annexed document or appendix.
type Annex struct {
	// Number is the annex identifier (e.g., "I", "A", "1").
	Number string `json:"number"`

	// Title is the annex title.
	Title string `json:"title"`

	// Text is the annex content.
	Text string `json:"text,omitempty"`

	// DocumentURI links to an external annexed document.
	DocumentURI string `json:"document_uri,omitempty"`
}

// VersionDiff represents the differences between two document versions.
type VersionDiff struct {
	// OldVersion is the earlier version.
	OldVersion Version `json:"old_version"`

	// NewVersion is the later version.
	NewVersion Version `json:"new_version"`

	// Added lists sections that were added.
	Added []Section `json:"added,omitempty"`

	// Removed lists sections that were removed.
	Removed []Section `json:"removed,omitempty"`

	// Modified lists sections that were changed.
	Modified []SectionChange `json:"modified,omitempty"`

	// Summary is a text summary of the changes.
	Summary string `json:"summary,omitempty"`
}

// SectionChange represents a modification to a section.
type SectionChange struct {
	// SectionNumber identifies the changed section.
	SectionNumber string `json:"section_number"`

	// OldText is the previous text.
	OldText string `json:"old_text"`

	// NewText is the updated text.
	NewText string `json:"new_text"`

	// ChangeType describes the nature of the change.
	ChangeType string `json:"change_type"`
}

// WorkingPaperParser extracts structured data from working paper documents.
type WorkingPaperParser struct {
	// BaseURI is the base URI for generated entity URIs.
	BaseURI string

	// patterns holds compiled regex patterns.
	patterns *workingPaperPatterns
}

// workingPaperPatterns holds compiled regex patterns for parsing.
type workingPaperPatterns struct {
	// Document metadata patterns
	identifierPatterns []*regexp.Regexp
	versionPatterns    []*regexp.Regexp
	datePatterns       []*regexp.Regexp
	authorPatterns     []*regexp.Regexp
	statusPatterns     []*regexp.Regexp
	titlePatterns      []*regexp.Regexp

	// Structure patterns
	sectionPatterns   []*regexp.Regexp
	paragraphPattern  *regexp.Regexp
	annexPattern      *regexp.Regexp
	tableStartPattern *regexp.Regexp

	// Content patterns
	actionPointPatterns []*regexp.Regexp
	annotationPatterns  []*regexp.Regexp
	reservationPattern  *regexp.Regexp
	referencePatterns   []*regexp.Regexp

	// Version relationship patterns
	supersedesPattern  *regexp.Regexp
	revisionOfPattern  *regexp.Regexp
	changeSummaryStart *regexp.Regexp
}

// NewWorkingPaperParser creates a new parser with default patterns.
func NewWorkingPaperParser(baseURI string) *WorkingPaperParser {
	return &WorkingPaperParser{
		BaseURI:  baseURI,
		patterns: compileWorkingPaperPatterns(),
	}
}

// compileWorkingPaperPatterns creates the default pattern set.
func compileWorkingPaperPatterns() *workingPaperPatterns {
	return &workingPaperPatterns{
		// Document identifier patterns
		identifierPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:Document|Doc\.?|Paper)\s*(?:No\.?|Number)?[:\s]+([A-Z0-9/_\-]+)`),
			regexp.MustCompile(`(?i)Reference[:\s]+([A-Z0-9/_\-]+)`),
			regexp.MustCompile(`(\d{6}_[a-z_]+(?:_draft)?(?:_[a-z]{2})?)`), // 210201_work_plan_draft_en
			regexp.MustCompile(`(?i)([A-Z]+/\d+/\d+(?:/REV\.?\d*)?)`),      // WG/2024/15/REV1
		},

		// Version patterns
		versionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)Version[:\s]+(\d+(?:\.\d+)*)`),
			regexp.MustCompile(`(?i)Rev(?:ision)?\.?[:\s]*(\d+)`),
			regexp.MustCompile(`(?i)v(\d+(?:\.\d+)*)`),
			regexp.MustCompile(`(?i)/REV\.?(\d+)`),
			regexp.MustCompile(`(?i)Draft\s+(\d+)`),
		},

		// Date patterns
		datePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)Date[:\s]+(\d{1,2})\s+(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{4})`),
			regexp.MustCompile(`(?i)(\d{1,2})\s+(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{4})`),
			regexp.MustCompile(`(?i)(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{1,2}),?\s+(\d{4})`),
			regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`),
			regexp.MustCompile(`(\d{1,2})/(\d{1,2})/(\d{4})`),
			regexp.MustCompile(`(\d{2})(\d{2})(\d{2})_`), // YYMMDD prefix
		},

		// Author patterns
		authorPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:Author|Prepared\s+by|Submitted\s+by|From)[:\s]+(.+?)(?:\n|$)`),
			regexp.MustCompile(`(?i)(?:Secretariat|Commission|Presidency)\s+(?:paper|document|note)`),
		},

		// Status patterns
		statusPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)Status[:\s]+(\w+)`),
			regexp.MustCompile(`(?i)\b(DRAFT|FINAL|REVISED|SUPERSEDED|WITHDRAWN)\b`),
			regexp.MustCompile(`(?i)(?:for\s+)?(adoption|discussion|information|decision|approval)`),
		},

		// Title patterns
		titlePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)Title[:\s]+(.+?)(?:\n|$)`),
			regexp.MustCompile(`(?i)Subject[:\s]+(.+?)(?:\n|$)`),
			regexp.MustCompile(`(?im)^(?:DRAFT\s+)?([A-Z][A-Z\s]+[A-Z])$`), // ALL CAPS title line
		},

		// Section patterns
		sectionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?m)^(\d+)\.\s+([A-Z][^\n]+)$`),                       // 1. Section Title
			regexp.MustCompile(`(?m)^(\d+\.\d+)\s+([A-Z][^\n]+)$`),                    // 1.1 Subsection
			regexp.MustCompile(`(?m)^(\d+\.\d+\.\d+)\s+([^\n]+)$`),                    // 1.1.1 Sub-subsection
			regexp.MustCompile(`(?im)^Section\s+(\d+)[:\s]+(.+?)$`),                   // Section 1: Title
			regexp.MustCompile(`(?m)^([IVXLCDM]+)\.\s+([A-Z][^\n]+)$`),                // I. Roman numeral
			regexp.MustCompile(`(?m)^([A-Z])\.\s+([A-Z][^\n]+)$`),                     // A. Letter heading
			regexp.MustCompile(`(?im)^(?:Chapter|Part)\s+(\d+|[IVXLCDM]+)[:\s]+(.+)$`), // Chapter/Part
		},

		// Paragraph pattern
		paragraphPattern: regexp.MustCompile(`(?m)^\s*(\d+)\.\s+([^\n]+)`),

		// Annex pattern
		annexPattern: regexp.MustCompile(`(?im)^(?:Annex|Appendix)\s+([IVXLCDM]+|\d+|[A-Z])[:\s]*(.+)?$`),

		// Table start pattern
		tableStartPattern: regexp.MustCompile(`(?im)^Table\s+(\d+)[:\s]*(.+)?$`),

		// Action point patterns
		actionPointPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?im)^\[?(?:ACTION|AP)\]?[:\s]+(.+?)(?:\n|$)`),
			regexp.MustCompile(`(?im)Action\s+(\d+)[:\s]+(.+?)(?:\n|$)`),
			regexp.MustCompile(`(?i)(?:is\s+)?(?:requested|invited|asked)\s+to\s+(.+?)(?:\.|$)`),
			regexp.MustCompile(`(?i)(?:shall|will|should)\s+(?:be\s+)?(?:required\s+to\s+)?(.+?)(?:by\s+|before\s+|$)`),
		},

		// Annotation patterns
		annotationPatterns: []*regexp.Regexp{
			regexp.MustCompile(`\[([^\]]+)\]`),                           // [bracketed text]
			regexp.MustCompile(`(?i)(?:Note|Comment)[:\s]+(.+?)(?:\n|$)`), // Note: comment
			regexp.MustCompile(`(?i)\{([^}]+)\}`),                         // {curly braces}
		},

		// Reservation pattern (delegation reservations)
		reservationPattern: regexp.MustCompile(`(?i)(?:reservation|scrutiny\s+reservation)(?:\s+by)?\s+([A-Z]{2,}(?:\s+[A-Z]+)?)`),

		// Reference patterns
		referencePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:see|cf\.?|refer\s+to)\s+(?:document\s+)?([A-Z0-9/\-]+)`),
			regexp.MustCompile(`(?i)(?:document|doc\.?)\s+(?:No\.?\s*)?([A-Z0-9/\-]+)`),
			regexp.MustCompile(`(?i)(?:Regulation|Directive)\s+\(?(?:EU|EC)\)?\s*(?:No\.?\s*)?(\d+/\d+)`),
		},

		// Version relationship patterns
		supersedesPattern:  regexp.MustCompile(`(?i)(?:supersedes|replaces|revises)\s+(?:document\s+)?([A-Z0-9/\-]+)`),
		revisionOfPattern:  regexp.MustCompile(`(?i)(?:revision\s+of|revised\s+version\s+of)\s+([A-Z0-9/\-]+)`),
		changeSummaryStart: regexp.MustCompile(`(?i)(?:Changes?|Modifications?|Revisions?)\s*(?:from|since|summary)[:\s]`),
	}
}

// Parse extracts a complete WorkingPaper structure from document text.
func (p *WorkingPaperParser) Parse(source string) (*WorkingPaper, error) {
	if source == "" {
		return nil, fmt.Errorf("empty source text")
	}

	doc := &WorkingPaper{
		Status: StatusDraft,
		Type:   TypeWorkingPaper,
	}

	// Extract identifier
	for _, pattern := range p.patterns.identifierPatterns {
		if match := pattern.FindStringSubmatch(source); match != nil {
			doc.Identifier = strings.TrimSpace(match[1])
			break
		}
	}

	// Extract version
	version, _ := p.ExtractVersion(source)
	if version != nil {
		doc.Version = *version
	}

	// Extract date
	doc.Date = p.extractDate(source)
	if doc.Version.Date.IsZero() && !doc.Date.IsZero() {
		doc.Version.Date = doc.Date
	}

	// Extract author
	for _, pattern := range p.patterns.authorPatterns {
		if match := pattern.FindStringSubmatch(source); match != nil {
			if len(match) > 1 {
				doc.Author = strings.TrimSpace(match[1])
			} else {
				doc.Author = strings.TrimSpace(match[0])
			}
			break
		}
	}

	// Extract status
	for _, pattern := range p.patterns.statusPatterns {
		if match := pattern.FindStringSubmatch(source); match != nil {
			doc.Status = ParseDocumentStatus(match[1])
			break
		}
	}

	// Extract title
	for _, pattern := range p.patterns.titlePatterns {
		if match := pattern.FindStringSubmatch(source); match != nil {
			doc.Title = strings.TrimSpace(match[1])
			break
		}
	}

	// Detect document type from content
	doc.Type = p.detectDocumentType(source)

	// Generate URI
	doc.URI = p.generateDocumentURI(doc)

	// Extract sections
	sections, _ := p.ExtractSections(source)
	doc.Sections = sections

	// Link sections to document
	for i := range doc.Sections {
		doc.Sections[i].DocumentURI = doc.URI
	}

	// Extract action points
	doc.ActionPoints = p.extractActionPoints(source)

	// Extract annotations
	doc.Annotations = p.extractAnnotations(source)

	// Extract references
	doc.References = p.extractReferences(source)

	// Extract annexes
	doc.Annexes = p.extractAnnexes(source)

	// Extract version relationships
	if match := p.patterns.supersedesPattern.FindStringSubmatch(source); match != nil {
		doc.SupersedesURI = fmt.Sprintf("%s/documents/%s", p.BaseURI, sanitizeForURI(match[1]))
		doc.Version.SupersedesID = match[1]
	}

	return doc, nil
}

// ExtractVersion parses version information from document text.
func (p *WorkingPaperParser) ExtractVersion(text string) (*Version, error) {
	version := &Version{
		Number: "1",
		Status: "draft",
	}

	// Extract version number
	for _, pattern := range p.patterns.versionPatterns {
		if match := pattern.FindStringSubmatch(text); match != nil {
			version.Number = strings.TrimSpace(match[1])
			// Try to extract revision number
			if rev, err := strconv.Atoi(version.Number); err == nil {
				version.Revision = rev
			}
			break
		}
	}

	// Extract version date
	version.Date = p.extractDate(text)

	// Extract supersedes reference
	if match := p.patterns.supersedesPattern.FindStringSubmatch(text); match != nil {
		version.SupersedesID = strings.TrimSpace(match[1])
	} else if match := p.patterns.revisionOfPattern.FindStringSubmatch(text); match != nil {
		version.SupersedesID = strings.TrimSpace(match[1])
	}

	// Extract change summary
	if match := p.patterns.changeSummaryStart.FindStringIndex(text); match != nil {
		endIdx := match[1] + 500
		if endIdx > len(text) {
			endIdx = len(text)
		}
		summary := text[match[1]:endIdx]
		if nlIdx := strings.Index(summary, "\n\n"); nlIdx > 0 {
			summary = summary[:nlIdx]
		}
		version.ChangeSummary = strings.TrimSpace(summary)
	}

	// Extract status
	for _, pattern := range p.patterns.statusPatterns {
		if match := pattern.FindStringSubmatch(text); match != nil {
			version.Status = strings.ToLower(strings.TrimSpace(match[1]))
			break
		}
	}

	return version, nil
}

// ExtractSections parses document sections from text.
func (p *WorkingPaperParser) ExtractSections(text string) ([]Section, error) {
	var sections []Section
	sectionNum := 0

	// Find all section headers
	type sectionMatch struct {
		number string
		title  string
		start  int
		level  int
	}
	var matches []sectionMatch

	for _, pattern := range p.patterns.sectionPatterns {
		for _, match := range pattern.FindAllStringSubmatchIndex(text, -1) {
			if len(match) >= 6 {
				numStart, numEnd := match[2], match[3]
				titleStart, titleEnd := match[4], match[5]
				sm := sectionMatch{
					number: strings.TrimSpace(text[numStart:numEnd]),
					title:  strings.TrimSpace(text[titleStart:titleEnd]),
					start:  match[0],
					level:  p.determineSectionLevel(text[numStart:numEnd]),
				}
				matches = append(matches, sm)
			}
		}
	}

	// Sort by position
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].start < matches[i].start {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	// Remove duplicates (overlapping matches)
	var deduped []sectionMatch
	for i, m := range matches {
		isDupe := false
		for j := 0; j < i; j++ {
			if absInt(matches[j].start-m.start) < 10 {
				isDupe = true
				break
			}
		}
		if !isDupe {
			deduped = append(deduped, m)
		}
	}
	matches = deduped

	// Extract section content
	for i, match := range matches {
		sectionNum++

		var textEnd int
		if i+1 < len(matches) {
			textEnd = matches[i+1].start
		} else {
			textEnd = len(text)
		}

		sectionText := ""
		if match.start < textEnd {
			// Skip the header line
			headerEnd := strings.Index(text[match.start:], "\n")
			if headerEnd > 0 {
				sectionText = strings.TrimSpace(text[match.start+headerEnd : textEnd])
			}
		}

		section := Section{
			URI:    p.generateSectionURI(match.number),
			Number: match.number,
			Title:  match.title,
			Level:  match.level,
			Text:   sectionText,
		}

		// Extract paragraphs within section
		section.Paragraphs = p.extractParagraphs(sectionText, section.URI)

		sections = append(sections, section)
	}

	return sections, nil
}

// CompareVersions computes the differences between two document versions.
func (p *WorkingPaperParser) CompareVersions(v1, v2 *WorkingPaper) (*VersionDiff, error) {
	if v1 == nil || v2 == nil {
		return nil, fmt.Errorf("both versions must be non-nil")
	}

	diff := &VersionDiff{
		OldVersion: v1.Version,
		NewVersion: v2.Version,
	}

	// Build section maps by number
	v1Sections := make(map[string]Section)
	for _, s := range v1.Sections {
		v1Sections[s.Number] = s
	}

	v2Sections := make(map[string]Section)
	for _, s := range v2.Sections {
		v2Sections[s.Number] = s
	}

	// Find added sections (in v2 but not v1)
	for num, s := range v2Sections {
		if _, exists := v1Sections[num]; !exists {
			diff.Added = append(diff.Added, s)
		}
	}

	// Find removed sections (in v1 but not v2)
	for num, s := range v1Sections {
		if _, exists := v2Sections[num]; !exists {
			diff.Removed = append(diff.Removed, s)
		}
	}

	// Find modified sections (in both but different)
	for num, s1 := range v1Sections {
		if s2, exists := v2Sections[num]; exists {
			if s1.Text != s2.Text || s1.Title != s2.Title {
				change := SectionChange{
					SectionNumber: num,
					OldText:       s1.Text,
					NewText:       s2.Text,
					ChangeType:    p.classifyChange(s1, s2),
				}
				diff.Modified = append(diff.Modified, change)
			}
		}
	}

	// Generate summary
	diff.Summary = p.generateChangeSummary(diff)

	return diff, nil
}

// Helper methods

func (p *WorkingPaperParser) extractDate(text string) time.Time {
	for _, pattern := range p.patterns.datePatterns {
		if match := pattern.FindStringSubmatch(text); match != nil {
			if date, err := p.parseDate(match); err == nil {
				return date
			}
		}
	}
	return time.Time{}
}

func (p *WorkingPaperParser) parseDate(match []string) (time.Time, error) {
	// Try standard formats first
	dateStr := strings.TrimSpace(match[0])
	formats := []string{
		"2 January 2006",
		"January 2, 2006",
		"2006-01-02",
		"02/01/2006",
		"01/02/2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	// Try to extract components
	if len(match) >= 4 {
		day, _ := strconv.Atoi(match[1])
		month := p.parseMonth(match[2])
		year, _ := strconv.Atoi(match[3])

		if day == 0 && month == 0 {
			// Try YYMMDD format
			if len(match[1]) == 2 && len(match[2]) == 2 && len(match[3]) == 2 {
				year, _ = strconv.Atoi(match[1])
				month, _ = strconv.Atoi(match[2])
				day, _ = strconv.Atoi(match[3])
				if year < 100 {
					year += 2000
				}
			}
		}

		if day > 0 && month > 0 && year > 0 {
			return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse date")
}

func (p *WorkingPaperParser) parseMonth(s string) int {
	months := map[string]int{
		"january": 1, "jan": 1, "01": 1, "1": 1,
		"february": 2, "feb": 2, "02": 2, "2": 2,
		"march": 3, "mar": 3, "03": 3, "3": 3,
		"april": 4, "apr": 4, "04": 4, "4": 4,
		"may": 5, "05": 5, "5": 5,
		"june": 6, "jun": 6, "06": 6, "6": 6,
		"july": 7, "jul": 7, "07": 7, "7": 7,
		"august": 8, "aug": 8, "08": 8, "8": 8,
		"september": 9, "sep": 9, "sept": 9, "09": 9, "9": 9,
		"october": 10, "oct": 10, "10": 10,
		"november": 11, "nov": 11, "11": 11,
		"december": 12, "dec": 12, "12": 12,
	}
	return months[strings.ToLower(s)]
}

func (p *WorkingPaperParser) detectDocumentType(text string) DocumentType {
	textLower := strings.ToLower(text)

	switch {
	case strings.Contains(textLower, "draft regulation") || strings.Contains(textLower, "draft directive"):
		return TypeDraftRegulation
	case strings.Contains(textLower, "position paper") || strings.Contains(textLower, "position of"):
		return TypePositionPaper
	case strings.Contains(textLower, "discussion paper") || strings.Contains(textLower, "discussion document"):
		return TypeDiscussionPaper
	case strings.Contains(textLower, "annotated agenda"):
		return TypeAnnotatedAgenda
	case strings.Contains(textLower, "work plan") || strings.Contains(textLower, "work programme") ||
		strings.Contains(textLower, "resourcing plan"):
		return TypeWorkPlan
	case strings.Contains(textLower, "report") || strings.Contains(textLower, "assessment"):
		return TypeReport
	default:
		return TypeWorkingPaper
	}
}

func (p *WorkingPaperParser) determineSectionLevel(number string) int {
	// Count dots to determine level
	dots := strings.Count(number, ".")
	if dots > 0 {
		return dots + 1
	}
	return 1
}

func (p *WorkingPaperParser) extractParagraphs(text string, sectionURI string) []Paragraph {
	var paragraphs []Paragraph

	matches := p.patterns.paragraphPattern.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			paragraphs = append(paragraphs, Paragraph{
				Number:     match[1],
				Text:       strings.TrimSpace(match[2]),
				SectionURI: sectionURI,
			})
		}
	}

	return paragraphs
}

func (p *WorkingPaperParser) extractActionPoints(text string) []ActionPoint {
	var actions []ActionPoint
	actionNum := 0

	for _, pattern := range p.patterns.actionPointPatterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			actionNum++
			ap := ActionPoint{
				Number: fmt.Sprintf("%d", actionNum),
				Status: "pending",
			}
			if len(match) >= 3 {
				ap.Number = match[1]
				ap.Description = strings.TrimSpace(match[2])
			} else if len(match) >= 2 {
				ap.Description = strings.TrimSpace(match[1])
			}
			if ap.Description != "" {
				actions = append(actions, ap)
			}
		}
	}

	return actions
}

func (p *WorkingPaperParser) extractAnnotations(text string) []Annotation {
	var annotations []Annotation

	// Extract bracketed annotations
	for _, pattern := range p.patterns.annotationPatterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				ann := Annotation{
					Type: "note",
					Text: strings.TrimSpace(match[1]),
				}
				annotations = append(annotations, ann)
			}
		}
	}

	// Extract delegation reservations
	matches := p.patterns.reservationPattern.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			ann := Annotation{
				Type:          "reservation",
				Text:          match[0],
				IsReservation: true,
				Delegation:    strings.TrimSpace(match[1]),
			}
			annotations = append(annotations, ann)
		}
	}

	return annotations
}

func (p *WorkingPaperParser) extractReferences(text string) []DocumentReference {
	var refs []DocumentReference
	seen := make(map[string]bool)

	for _, pattern := range p.patterns.referencePatterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				id := strings.TrimSpace(match[1])
				if !seen[id] {
					seen[id] = true
					refs = append(refs, DocumentReference{
						Type:       "document",
						Identifier: id,
					})
				}
			}
		}
	}

	return refs
}

func (p *WorkingPaperParser) extractAnnexes(text string) []Annex {
	var annexes []Annex

	matches := p.patterns.annexPattern.FindAllStringSubmatchIndex(text, -1)
	for i, match := range matches {
		if len(match) >= 4 {
			numStart, numEnd := match[2], match[3]
			number := strings.TrimSpace(text[numStart:numEnd])

			title := ""
			if len(match) >= 6 && match[4] >= 0 {
				title = strings.TrimSpace(text[match[4]:match[5]])
			}

			// Extract annex content
			contentStart := match[1]
			var contentEnd int
			if i+1 < len(matches) {
				contentEnd = matches[i+1][0]
			} else {
				contentEnd = len(text)
				if contentEnd-contentStart > 2000 {
					contentEnd = contentStart + 2000
				}
			}

			annexText := ""
			if contentStart < contentEnd {
				annexText = strings.TrimSpace(text[contentStart:contentEnd])
			}

			annexes = append(annexes, Annex{
				Number: number,
				Title:  title,
				Text:   annexText,
			})
		}
	}

	return annexes
}

func (p *WorkingPaperParser) classifyChange(old, new Section) string {
	oldLen := len(old.Text)
	newLen := len(new.Text)

	if old.Title != new.Title {
		return "renamed"
	}
	if newLen > oldLen*2 {
		return "expanded"
	}
	if newLen < oldLen/2 {
		return "reduced"
	}
	return "modified"
}

func (p *WorkingPaperParser) generateChangeSummary(diff *VersionDiff) string {
	parts := []string{}

	if len(diff.Added) > 0 {
		parts = append(parts, fmt.Sprintf("%d section(s) added", len(diff.Added)))
	}
	if len(diff.Removed) > 0 {
		parts = append(parts, fmt.Sprintf("%d section(s) removed", len(diff.Removed)))
	}
	if len(diff.Modified) > 0 {
		parts = append(parts, fmt.Sprintf("%d section(s) modified", len(diff.Modified)))
	}

	if len(parts) == 0 {
		return "No changes detected"
	}
	return strings.Join(parts, ", ")
}

// URI generation helpers

func (p *WorkingPaperParser) generateDocumentURI(doc *WorkingPaper) string {
	if doc.Identifier != "" {
		return fmt.Sprintf("%s/documents/%s", p.BaseURI, sanitizeForURI(doc.Identifier))
	}
	if !doc.Date.IsZero() {
		return fmt.Sprintf("%s/documents/%s", p.BaseURI, doc.Date.Format("2006-01-02"))
	}
	return fmt.Sprintf("%s/documents/unknown", p.BaseURI)
}

func (p *WorkingPaperParser) generateSectionURI(number string) string {
	return fmt.Sprintf("%s/sections/%s", p.BaseURI, sanitizeForURI(number))
}

// Note: sanitizeForURI function is defined in parser.go

// absInt returns the absolute value of an integer.
func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// String returns a human-readable representation of the working paper.
func (wp *WorkingPaper) String() string {
	if wp.Title != "" {
		return fmt.Sprintf("%s: %s (v%s, %s)", wp.Identifier, wp.Title, wp.Version.Number, wp.Status)
	}
	return fmt.Sprintf("%s (v%s, %s)", wp.Identifier, wp.Version.Number, wp.Status)
}
