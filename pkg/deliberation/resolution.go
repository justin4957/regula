package deliberation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Resolution represents a formal resolution or decision document adopted by
// a deliberative body. Resolutions have a structured format with preamble
// recitals and operative clauses.
type Resolution struct {
	// URI is the unique identifier for this resolution in the knowledge graph.
	URI string `json:"uri"`

	// Identifier is the document number (e.g., "A/RES/79/100", "2024/123/EU").
	Identifier string `json:"identifier"`

	// Title is the resolution title or subject matter.
	Title string `json:"title"`

	// AdoptingBody is the body that adopted the resolution.
	AdoptingBody string `json:"adopting_body"`

	// AdoptionDate is when the resolution was adopted.
	AdoptionDate time.Time `json:"adoption_date"`

	// Session identifies the session during which it was adopted.
	Session string `json:"session,omitempty"`

	// Status indicates the resolution status (adopted, draft, superseded).
	Status string `json:"status"`

	// Preamble contains the recitals (Recalling, Noting, Considering, etc.).
	Preamble []Recital `json:"preamble,omitempty"`

	// OperativeClauses contains the numbered decisions/directives.
	OperativeClauses []OperativeClause `json:"operative_clauses,omitempty"`

	// Vote contains the vote record if available.
	Vote *VoteRecord `json:"vote,omitempty"`

	// References lists cross-references to other documents.
	References []ResolutionReference `json:"references,omitempty"`

	// SubjectAreas lists topics/themes covered.
	SubjectAreas []string `json:"subject_areas,omitempty"`

	// SourceDocument is the URI of the source document.
	SourceDocument string `json:"source_document,omitempty"`

	// Language is the document language (e.g., "en", "fr").
	Language string `json:"language,omitempty"`

	// MeetingURI links to the meeting where this was adopted.
	MeetingURI string `json:"meeting_uri,omitempty"`

	// SupersedesURI links to a resolution this one supersedes.
	SupersedesURI string `json:"supersedes_uri,omitempty"`

	// SupersededByURI links to a later resolution that supersedes this.
	SupersededByURI string `json:"superseded_by_uri,omitempty"`
}

// Recital represents a preamble paragraph that provides context, legal basis,
// or references to prior decisions. Recitals typically begin with phrases like
// "Recalling", "Noting", "Considering", "Guided by", etc.
type Recital struct {
	// URI is the unique identifier for this recital.
	URI string `json:"uri"`

	// Number is the recital number (if numbered).
	Number int `json:"number,omitempty"`

	// IntroPhrase is the opening phrase (e.g., "Recalling", "Noting with concern").
	IntroPhrase string `json:"intro_phrase"`

	// Text is the full recital text.
	Text string `json:"text"`

	// References lists documents/resolutions referenced in this recital.
	References []ResolutionReference `json:"references,omitempty"`

	// ResolutionURI links back to the parent resolution.
	ResolutionURI string `json:"resolution_uri,omitempty"`
}

// OperativeClause represents a numbered paragraph in the operative part of
// a resolution. These contain the actual decisions, directives, or requests.
type OperativeClause struct {
	// URI is the unique identifier for this clause.
	URI string `json:"uri"`

	// Number is the clause number.
	Number int `json:"number"`

	// ActionVerb is the main verb (e.g., "Decides", "Requests", "Calls upon").
	ActionVerb string `json:"action_verb"`

	// Text is the full clause text.
	Text string `json:"text"`

	// SubClauses contains lettered or numbered sub-paragraphs.
	SubClauses []SubClause `json:"sub_clauses,omitempty"`

	// AddressedTo lists entities the clause is directed at.
	AddressedTo []string `json:"addressed_to,omitempty"`

	// Deadline is a deadline specified in the clause.
	Deadline *time.Time `json:"deadline,omitempty"`

	// References lists documents/provisions referenced in this clause.
	References []ResolutionReference `json:"references,omitempty"`

	// ResolutionURI links back to the parent resolution.
	ResolutionURI string `json:"resolution_uri,omitempty"`
}

// SubClause represents a sub-paragraph within an operative clause.
type SubClause struct {
	// Identifier is the sub-clause identifier (e.g., "a", "i", "1").
	Identifier string `json:"identifier"`

	// Text is the sub-clause text.
	Text string `json:"text"`

	// ParentClauseURI links to the parent clause.
	ParentClauseURI string `json:"parent_clause_uri,omitempty"`
}

// ResolutionReference represents a reference to another document within
// a resolution's preamble or operative part.
type ResolutionReference struct {
	// Type classifies the reference (resolution, treaty, report, regulation).
	Type string `json:"type"`

	// Identifier is the document identifier (e.g., "A/RES/78/200").
	Identifier string `json:"identifier"`

	// Title is the document title if known.
	Title string `json:"title,omitempty"`

	// URI is the resolved URI for the referenced document.
	URI string `json:"uri,omitempty"`

	// Date is the document date if known.
	Date *time.Time `json:"date,omitempty"`
}

// ResolutionParser extracts structured data from formal resolutions and decisions.
type ResolutionParser struct {
	// BaseURI is the base URI for generated entity URIs.
	BaseURI string

	// Format hints at the expected document format (un, eu, generic).
	Format string

	// patterns holds compiled regex patterns for extraction.
	patterns *resolutionPatterns
}

// resolutionPatterns holds compiled regex patterns for resolution parsing.
type resolutionPatterns struct {
	// Identifier patterns
	identifierPatterns []*regexp.Regexp

	// Date patterns
	datePatterns []*regexp.Regexp

	// Body/session patterns
	bodyPattern    *regexp.Regexp
	sessionPattern *regexp.Regexp

	// Title pattern
	titlePattern *regexp.Regexp

	// Preamble patterns
	preambleStart    *regexp.Regexp
	preambleEnd      *regexp.Regexp
	recitalPatterns  []*regexp.Regexp
	introPhrasesRe   *regexp.Regexp

	// Operative patterns
	operativeStart    *regexp.Regexp
	clausePattern     *regexp.Regexp
	subClausePattern  *regexp.Regexp
	actionVerbPattern *regexp.Regexp

	// Vote patterns
	votePatterns      []*regexp.Regexp
	adoptionPattern   *regexp.Regexp
	consensusPattern  *regexp.Regexp

	// Reference patterns
	resolutionRefPattern *regexp.Regexp
	treatyRefPattern     *regexp.Regexp
	regulationRefPattern *regexp.Regexp
	reportRefPattern     *regexp.Regexp
}

// NewResolutionParser creates a new parser with default patterns.
func NewResolutionParser(baseURI string) *ResolutionParser {
	return &ResolutionParser{
		BaseURI:  baseURI,
		Format:   "generic",
		patterns: compileGenericResolutionPatterns(),
	}
}

// NewResolutionParserWithFormat creates a parser optimized for a specific format.
func NewResolutionParserWithFormat(baseURI, format string) *ResolutionParser {
	p := &ResolutionParser{
		BaseURI: baseURI,
		Format:  format,
	}

	switch format {
	case "un":
		p.patterns = compileUNResolutionPatterns()
	case "eu":
		p.patterns = compileEUResolutionPatterns()
	default:
		p.patterns = compileGenericResolutionPatterns()
	}

	return p
}

// Parse extracts a complete Resolution structure from resolution text.
func (p *ResolutionParser) Parse(source string) (*Resolution, error) {
	if source == "" {
		return nil, fmt.Errorf("empty source text")
	}

	resolution := &Resolution{
		Status: "adopted",
	}

	// Extract identifier
	for _, pattern := range p.patterns.identifierPatterns {
		if match := pattern.FindStringSubmatch(source); match != nil {
			resolution.Identifier = strings.TrimSpace(match[0])
			break
		}
	}

	// Extract date
	for _, pattern := range p.patterns.datePatterns {
		if match := pattern.FindStringSubmatch(source); match != nil {
			if date, err := p.parseDate(match); err == nil {
				resolution.AdoptionDate = date
				break
			}
		}
	}

	// Extract adopting body
	if match := p.patterns.bodyPattern.FindStringSubmatch(source); match != nil {
		resolution.AdoptingBody = strings.TrimSpace(match[1])
	}

	// Extract session
	if match := p.patterns.sessionPattern.FindStringSubmatch(source); match != nil {
		resolution.Session = strings.TrimSpace(match[0])
	}

	// Extract title
	if match := p.patterns.titlePattern.FindStringSubmatch(source); match != nil {
		resolution.Title = strings.TrimSpace(match[1])
	}

	// Generate URI
	if resolution.URI == "" {
		resolution.URI = p.generateResolutionURI(resolution)
	}

	// Extract preamble recitals
	recitals, _ := p.ExtractPreamble(source)
	resolution.Preamble = recitals

	// Extract operative clauses
	clauses, _ := p.ExtractOperativeClauses(source)
	resolution.OperativeClauses = clauses

	// Extract vote record
	if vote, _ := p.ExtractVoteRecord(source); vote != nil {
		resolution.Vote = vote
	}

	// Extract all references
	resolution.References = p.extractAllReferences(source)

	// Link recitals and clauses back to resolution
	for i := range resolution.Preamble {
		resolution.Preamble[i].ResolutionURI = resolution.URI
	}
	for i := range resolution.OperativeClauses {
		resolution.OperativeClauses[i].ResolutionURI = resolution.URI
	}

	return resolution, nil
}

// ExtractPreamble parses preamble recitals from the resolution text.
func (p *ResolutionParser) ExtractPreamble(text string) ([]Recital, error) {
	var recitals []Recital

	// Find preamble section
	preambleText := text
	if match := p.patterns.preambleStart.FindStringIndex(text); match != nil {
		startIdx := match[0]
		endIdx := len(text)
		if endMatch := p.patterns.preambleEnd.FindStringIndex(text[startIdx:]); endMatch != nil {
			endIdx = startIdx + endMatch[0]
		}
		preambleText = text[startIdx:endIdx]
	}

	// Extract individual recitals
	recitalNum := 0
	for _, pattern := range p.patterns.recitalPatterns {
		matches := pattern.FindAllStringSubmatch(preambleText, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}

			recitalNum++
			introPhrase := ""
			recitalText := strings.TrimSpace(match[1])

			// Extract intro phrase
			if introMatch := p.patterns.introPhrasesRe.FindStringSubmatch(recitalText); introMatch != nil {
				introPhrase = strings.TrimSpace(introMatch[1])
				// Remove intro phrase from text for cleaner storage
				recitalText = strings.TrimPrefix(recitalText, introMatch[0])
				recitalText = strings.TrimSpace(recitalText)
			}

			recital := Recital{
				URI:         p.generateRecitalURI(recitalNum),
				Number:      recitalNum,
				IntroPhrase: introPhrase,
				Text:        recitalText,
				References:  p.extractReferencesFromText(recitalText),
			}

			recitals = append(recitals, recital)
		}
	}

	// Also try to find recitals by intro phrases directly if none found
	if len(recitals) == 0 {
		introMatches := p.patterns.introPhrasesRe.FindAllStringSubmatchIndex(text, -1)
		for i, match := range introMatches {
			if len(match) < 4 {
				continue
			}

			startIdx := match[0]
			var endIdx int
			if i+1 < len(introMatches) {
				endIdx = introMatches[i+1][0]
			} else {
				// Find end by looking for operative section or end of document
				if opMatch := p.patterns.operativeStart.FindStringIndex(text[startIdx:]); opMatch != nil {
					endIdx = startIdx + opMatch[0]
				} else {
					endIdx = startIdx + 500
					if endIdx > len(text) {
						endIdx = len(text)
					}
				}
			}

			recitalText := strings.TrimSpace(text[startIdx:endIdx])
			// Clean up: remove trailing punctuation patterns
			if idx := strings.Index(recitalText, "\n\n"); idx > 0 {
				recitalText = recitalText[:idx]
			}

			introPhrase := strings.TrimSpace(text[match[2]:match[3]])

			recitalNum++
			recital := Recital{
				URI:         p.generateRecitalURI(recitalNum),
				Number:      recitalNum,
				IntroPhrase: introPhrase,
				Text:        recitalText,
				References:  p.extractReferencesFromText(recitalText),
			}

			recitals = append(recitals, recital)
		}
	}

	return recitals, nil
}

// ExtractOperativeClauses parses numbered operative clauses from the resolution.
func (p *ResolutionParser) ExtractOperativeClauses(text string) ([]OperativeClause, error) {
	var clauses []OperativeClause

	// Find operative section
	operativeText := text
	if match := p.patterns.operativeStart.FindStringIndex(text); match != nil {
		operativeText = text[match[0]:]
	}

	// Extract numbered clauses
	matches := p.patterns.clausePattern.FindAllStringSubmatchIndex(operativeText, -1)
	for i, match := range matches {
		if len(match) < 4 {
			continue
		}

		numStr := operativeText[match[2]:match[3]]
		num, _ := strconv.Atoi(numStr)

		// Determine clause text extent
		textStart := match[1]
		var textEnd int
		if i+1 < len(matches) {
			textEnd = matches[i+1][0]
		} else {
			textEnd = len(operativeText)
		}

		clauseText := strings.TrimSpace(operativeText[textStart:textEnd])

		// Extract action verb
		actionVerb := ""
		if verbMatch := p.patterns.actionVerbPattern.FindStringSubmatch(clauseText); verbMatch != nil {
			actionVerb = strings.TrimSpace(verbMatch[1])
		}

		clause := OperativeClause{
			URI:        p.generateClauseURI(num),
			Number:     num,
			ActionVerb: actionVerb,
			Text:       clauseText,
			References: p.extractReferencesFromText(clauseText),
		}

		// Extract sub-clauses
		subMatches := p.patterns.subClausePattern.FindAllStringSubmatch(clauseText, -1)
		for _, subMatch := range subMatches {
			if len(subMatch) >= 3 {
				subClause := SubClause{
					Identifier:      strings.TrimSpace(subMatch[1]),
					Text:            strings.TrimSpace(subMatch[2]),
					ParentClauseURI: clause.URI,
				}
				clause.SubClauses = append(clause.SubClauses, subClause)
			}
		}

		// Extract addressees
		clause.AddressedTo = p.extractAddressees(clauseText)

		// Extract deadline if present
		clause.Deadline = p.extractDeadline(clauseText)

		clauses = append(clauses, clause)
	}

	return clauses, nil
}

// ExtractVoteRecord extracts vote information from the resolution text.
func (p *ResolutionParser) ExtractVoteRecord(text string) (*VoteRecord, error) {
	// Check for consensus adoption
	if match := p.patterns.consensusPattern.FindString(text); match != "" {
		return &VoteRecord{
			URI:      p.generateVoteURI(1),
			VoteType: "consensus",
			Result:   "adopted by consensus",
		}, nil
	}

	// Look for recorded vote
	for _, pattern := range p.patterns.votePatterns {
		if match := pattern.FindStringSubmatch(text); match != nil {
			vote := &VoteRecord{
				URI:      p.generateVoteURI(1),
				VoteType: "recorded",
			}

			// Parse vote counts
			if len(match) >= 4 {
				vote.ForCount, _ = strconv.Atoi(match[1])
				vote.AgainstCount, _ = strconv.Atoi(match[2])
				vote.AbstainCount, _ = strconv.Atoi(match[3])
			}

			if vote.ForCount > vote.AgainstCount {
				vote.Result = "adopted"
			} else {
				vote.Result = "rejected"
			}

			return vote, nil
		}
	}

	// Check for simple adoption statement
	if match := p.patterns.adoptionPattern.FindString(text); match != "" {
		return &VoteRecord{
			URI:      p.generateVoteURI(1),
			VoteType: "unrecorded",
			Result:   "adopted",
		}, nil
	}

	return nil, nil
}

// Helper methods

func (p *ResolutionParser) parseDate(match []string) (time.Time, error) {
	formats := []string{
		"2 January 2006",
		"January 2, 2006",
		"02/01/2006",
		"2006-01-02",
		"2 Jan 2006",
	}

	dateStr := strings.TrimSpace(match[0])
	dateStr = regexp.MustCompile(`\s+`).ReplaceAllString(dateStr, " ")

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
		if day > 0 && month > 0 && year > 0 {
			return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse date: %s", dateStr)
}

func (p *ResolutionParser) parseMonth(s string) int {
	months := map[string]int{
		"january": 1, "jan": 1,
		"february": 2, "feb": 2,
		"march": 3, "mar": 3,
		"april": 4, "apr": 4,
		"may": 5,
		"june": 6, "jun": 6,
		"july": 7, "jul": 7,
		"august": 8, "aug": 8,
		"september": 9, "sep": 9, "sept": 9,
		"october": 10, "oct": 10,
		"november": 11, "nov": 11,
		"december": 12, "dec": 12,
	}
	return months[strings.ToLower(s)]
}

func (p *ResolutionParser) extractAllReferences(text string) []ResolutionReference {
	var refs []ResolutionReference
	seen := make(map[string]bool)

	// Resolution references
	if matches := p.patterns.resolutionRefPattern.FindAllStringSubmatch(text, -1); matches != nil {
		for _, match := range matches {
			if len(match) >= 2 && !seen[match[1]] {
				seen[match[1]] = true
				refs = append(refs, ResolutionReference{
					Type:       "resolution",
					Identifier: match[1],
				})
			}
		}
	}

	// Treaty references
	if matches := p.patterns.treatyRefPattern.FindAllStringSubmatch(text, -1); matches != nil {
		for _, match := range matches {
			id := match[0]
			if !seen[id] {
				seen[id] = true
				refs = append(refs, ResolutionReference{
					Type:       "treaty",
					Identifier: id,
				})
			}
		}
	}

	// Regulation references
	if matches := p.patterns.regulationRefPattern.FindAllStringSubmatch(text, -1); matches != nil {
		for _, match := range matches {
			if len(match) >= 1 && !seen[match[0]] {
				seen[match[0]] = true
				refs = append(refs, ResolutionReference{
					Type:       "regulation",
					Identifier: match[0],
				})
			}
		}
	}

	// Report references
	if matches := p.patterns.reportRefPattern.FindAllStringSubmatch(text, -1); matches != nil {
		for _, match := range matches {
			if len(match) >= 2 && !seen[match[1]] {
				seen[match[1]] = true
				refs = append(refs, ResolutionReference{
					Type:       "report",
					Identifier: match[1],
				})
			}
		}
	}

	return refs
}

func (p *ResolutionParser) extractReferencesFromText(text string) []ResolutionReference {
	return p.extractAllReferences(text)
}

func (p *ResolutionParser) extractAddressees(text string) []string {
	var addressees []string
	seen := make(map[string]bool)

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:calls upon|requests|invites|urges)\s+(?:the\s+)?([A-Z][a-zA-Z\s]+?)(?:\s+to\b|,)`),
		regexp.MustCompile(`(?i)(?:Member\s+States|all\s+States|the\s+Secretary-General|the\s+Commission)`),
	}

	for _, pattern := range patterns {
		if matches := pattern.FindAllStringSubmatch(text, -1); matches != nil {
			for _, match := range matches {
				var addressee string
				if len(match) > 1 {
					addressee = strings.TrimSpace(match[1])
				} else {
					addressee = strings.TrimSpace(match[0])
				}
				if addressee != "" && !seen[addressee] {
					seen[addressee] = true
					addressees = append(addressees, addressee)
				}
			}
		}
	}

	return addressees
}

func (p *ResolutionParser) extractDeadline(text string) *time.Time {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)by\s+(\d{1,2})\s+(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{4})`),
		regexp.MustCompile(`(?i)(?:before|no\s+later\s+than)\s+(\d{1,2})\s+(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{4})`),
		regexp.MustCompile(`(?i)within\s+(\d+)\s+(?:days?|months?|years?)`),
	}

	for _, pattern := range patterns {
		if match := pattern.FindStringSubmatch(text); match != nil {
			if date, err := p.parseDate(match); err == nil {
				return &date
			}
		}
	}

	return nil
}

// URI generation helpers

func (p *ResolutionParser) generateResolutionURI(r *Resolution) string {
	if r.Identifier != "" {
		return fmt.Sprintf("%s/resolutions/%s", p.BaseURI, sanitizeForURI(r.Identifier))
	}
	if !r.AdoptionDate.IsZero() {
		return fmt.Sprintf("%s/resolutions/%s", p.BaseURI, r.AdoptionDate.Format("2006-01-02"))
	}
	return fmt.Sprintf("%s/resolutions/unknown", p.BaseURI)
}

func (p *ResolutionParser) generateRecitalURI(num int) string {
	return fmt.Sprintf("%s/recitals/%d", p.BaseURI, num)
}

func (p *ResolutionParser) generateClauseURI(num int) string {
	return fmt.Sprintf("%s/clauses/%d", p.BaseURI, num)
}

func (p *ResolutionParser) generateVoteURI(num int) string {
	return fmt.Sprintf("%s/votes/%d", p.BaseURI, num)
}

// String returns a human-readable representation of the resolution.
func (r *Resolution) String() string {
	if r.Title != "" {
		return fmt.Sprintf("%s: %s (%s)", r.Identifier, r.Title, r.AdoptionDate.Format("2006-01-02"))
	}
	return fmt.Sprintf("%s (%s)", r.Identifier, r.AdoptionDate.Format("2006-01-02"))
}
