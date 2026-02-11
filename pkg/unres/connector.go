// Package unres provides a connector for parsing UN General Assembly and
// Security Council resolutions from the UNxml GitHub repositories using
// the Akoma Ntoso for UN (AKN4UN) XML schema.
package unres

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// UNBody represents a UN deliberative body.
type UNBody int

const (
	// BodyGeneralAssembly represents the UN General Assembly.
	BodyGeneralAssembly UNBody = iota
	// BodySecurityCouncil represents the UN Security Council.
	BodySecurityCouncil
	// BodyECOSOC represents the Economic and Social Council.
	BodyECOSOC
)

// String returns the string representation of a UNBody.
func (b UNBody) String() string {
	switch b {
	case BodyGeneralAssembly:
		return "general-assembly"
	case BodySecurityCouncil:
		return "security-council"
	case BodyECOSOC:
		return "ecosoc"
	default:
		return "unknown"
	}
}

// Abbreviation returns the short form of the body name.
func (b UNBody) Abbreviation() string {
	switch b {
	case BodyGeneralAssembly:
		return "GA"
	case BodySecurityCouncil:
		return "SC"
	case BodyECOSOC:
		return "ECOSOC"
	default:
		return "UN"
	}
}

// ParseUNBody converts a string to a UNBody.
func ParseUNBody(s string) UNBody {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "ga", "general-assembly", "generalassembly", "unga":
		return BodyGeneralAssembly
	case "sc", "security-council", "securitycouncil", "unsc":
		return BodySecurityCouncil
	case "ecosoc", "economic-and-social-council":
		return BodyECOSOC
	default:
		return BodyGeneralAssembly
	}
}

// DocumentReference represents a reference to another UN document.
type DocumentReference struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
	URI        string `json:"uri"`
	ShowAs     string `json:"show_as,omitempty"`
}

// ConceptReference represents a reference to a UN ontology concept.
type ConceptReference struct {
	ID     string `json:"id"`
	URI    string `json:"uri"`
	ShowAs string `json:"show_as"`
}

// OrganizationReference represents a reference to a UN organization.
type OrganizationReference struct {
	ID     string `json:"id"`
	URI    string `json:"uri"`
	ShowAs string `json:"show_as"`
}

// PreambleRecital represents a preamble paragraph with its intro phrase.
type PreambleRecital struct {
	IntroPhrase string              `json:"intro_phrase"`
	Text        string              `json:"text"`
	References  []DocumentReference `json:"references,omitempty"`
}

// OperativeParagraph represents an operative paragraph.
type OperativeParagraph struct {
	Number        int                  `json:"number"`
	EId           string               `json:"eid,omitempty"`
	Action        string               `json:"action"`
	Text          string               `json:"text"`
	SubParagraphs []OperativeParagraph `json:"sub_paragraphs,omitempty"`
}

// UNResolution represents a parsed UN resolution.
type UNResolution struct {
	// Identification
	DocumentID   string    `json:"document_id"`
	Body         UNBody    `json:"body"`
	Session      int       `json:"session"`
	Number       int       `json:"number"`
	AdoptionDate time.Time `json:"adoption_date"`
	Language     string    `json:"language"`

	// Content
	Title     string `json:"title"`
	Proponent string `json:"proponent"`

	// Structure
	Preamble       []PreambleRecital    `json:"preamble"`
	OperativeParts []OperativeParagraph `json:"operative_parts"`

	// References
	References    []DocumentReference     `json:"references"`
	Concepts      []ConceptReference      `json:"concepts"`
	Organizations []OrganizationReference `json:"organizations"`

	// Source
	SourcePath string `json:"source_path,omitempty"`
	AKNURI     string `json:"akn_uri,omitempty"`
}

// ResolutionRef is a lightweight reference to a resolution.
type ResolutionRef struct {
	DocumentID string `json:"document_id"`
	Path       string `json:"path"`
	Body       UNBody `json:"body"`
	Session    int    `json:"session"`
	Number     int    `json:"number"`
}

// UNResolutionConnector parses UN resolutions from local or remote sources.
type UNResolutionConnector struct {
	LocalPath string
}

// NewUNResolutionConnector creates a new connector with a local path.
func NewUNResolutionConnector(localPath string) *UNResolutionConnector {
	return &UNResolutionConnector{LocalPath: localPath}
}

// ListResolutions lists all resolutions for a body and optional session.
func (c *UNResolutionConnector) ListResolutions(body UNBody, session int) ([]ResolutionRef, error) {
	repoPath := c.getRepoPath(body)
	if repoPath == "" {
		return nil, fmt.Errorf("unknown body: %v", body)
	}

	var refs []ResolutionRef

	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".xml") {
			return nil
		}

		ref := parseResolutionPath(path, body)
		if ref != nil {
			// Filter by session if specified
			if session > 0 && ref.Session != session {
				return nil
			}
			refs = append(refs, *ref)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Sort by session and number
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Session != refs[j].Session {
			return refs[i].Session < refs[j].Session
		}
		return refs[i].Number < refs[j].Number
	})

	return refs, nil
}

// ParseResolution parses a resolution from a file path.
func (c *UNResolutionConnector) ParseResolution(path string) (*UNResolution, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	res, err := ParseAKNResolution(data)
	if err != nil {
		return nil, err
	}

	res.SourcePath = path
	return res, nil
}

// ParseResolutionByRef parses a resolution by reference.
func (c *UNResolutionConnector) ParseResolutionByRef(ref ResolutionRef) (*UNResolution, error) {
	return c.ParseResolution(ref.Path)
}

// getRepoPath returns the repository path for a body.
func (c *UNResolutionConnector) getRepoPath(body UNBody) string {
	switch body {
	case BodyGeneralAssembly:
		return filepath.Join(c.LocalPath, "GAresolutions")
	case BodySecurityCouncil:
		return filepath.Join(c.LocalPath, "SCresolutions")
	case BodyECOSOC:
		return filepath.Join(c.LocalPath, "ECOSOCresolutions")
	default:
		return ""
	}
}

// parseResolutionPath extracts resolution info from a file path.
func parseResolutionPath(path string, body UNBody) *ResolutionRef {
	filename := filepath.Base(path)
	filename = strings.TrimSuffix(filename, ".xml")

	// Try to extract session and number from filename
	// Common patterns: "A_RES_79_100", "S_RES_2798", "res_79_100"
	var session, number int

	// Pattern: A_RES_{session}_{number} or S_RES_{number}
	if strings.Contains(filename, "_RES_") {
		parts := strings.Split(filename, "_RES_")
		if len(parts) == 2 {
			subparts := strings.Split(parts[1], "_")
			if body == BodySecurityCouncil {
				// SC: S_RES_{number}
				if len(subparts) >= 1 {
					number, _ = strconv.Atoi(subparts[0])
				}
			} else {
				// GA/ECOSOC: A_RES_{session}_{number}
				if len(subparts) >= 2 {
					session, _ = strconv.Atoi(subparts[0])
					number, _ = strconv.Atoi(subparts[1])
				}
			}
		}
	}

	// Build document ID
	var docID string
	switch body {
	case BodyGeneralAssembly:
		docID = fmt.Sprintf("A/RES/%d/%d", session, number)
	case BodySecurityCouncil:
		docID = fmt.Sprintf("S/RES/%d", number)
	case BodyECOSOC:
		docID = fmt.Sprintf("E/RES/%d/%d", session, number)
	}

	return &ResolutionRef{
		DocumentID: docID,
		Path:       path,
		Body:       body,
		Session:    session,
		Number:     number,
	}
}

// XML structures for Akoma Ntoso parsing

type xmlAkomaNtoso struct {
	XMLName xml.Name `xml:"akomaNtoso"`
	Doc     xmlDoc   `xml:"doc"`
}

type xmlDoc struct {
	Name        string         `xml:"name,attr"`
	Meta        xmlMeta        `xml:"meta"`
	Preface     xmlPreface     `xml:"preface"`
	Preamble    xmlPreamble    `xml:"preamble"`
	Body        xmlBody        `xml:"body"`
	Conclusions xmlConclusions `xml:"conclusions"`
}

type xmlMeta struct {
	Identification xmlIdentification `xml:"identification"`
	References     xmlReferences     `xml:"references"`
}

type xmlIdentification struct {
	Source     string          `xml:"source,attr"`
	FRBRWork   xmlFRBRWork     `xml:"FRBRWork"`
	FRBRExpr   xmlFRBRExpr     `xml:"FRBRExpression"`
	FRBRManif  xmlFRBRManif    `xml:"FRBRManifestation"`
}

type xmlFRBRWork struct {
	FRBRthis   xmlFRBRValue `xml:"FRBRthis"`
	FRBRuri    xmlFRBRValue `xml:"FRBRuri"`
	FRBRdate   xmlFRBRDate  `xml:"FRBRdate"`
	FRBRauthor xmlFRBRRef   `xml:"FRBRauthor"`
}

type xmlFRBRExpr struct {
	FRBRlanguage xmlFRBRLang `xml:"FRBRlanguage"`
}

type xmlFRBRManif struct{}

type xmlFRBRValue struct {
	Value string `xml:"value,attr"`
}

type xmlFRBRDate struct {
	Date string `xml:"date,attr"`
	Name string `xml:"name,attr"`
}

type xmlFRBRRef struct {
	Href string `xml:"href,attr"`
	As   string `xml:"as,attr"`
}

type xmlFRBRLang struct {
	Language string `xml:"language,attr"`
}

type xmlReferences struct {
	Source        string               `xml:"source,attr"`
	Organizations []xmlTLCOrganization `xml:"TLCOrganization"`
	Concepts      []xmlTLCConcept      `xml:"TLCConcept"`
	References    []xmlTLCReference    `xml:"TLCReference"`
}

type xmlTLCOrganization struct {
	EId    string `xml:"eId,attr"`
	Href   string `xml:"href,attr"`
	ShowAs string `xml:"showAs,attr"`
}

type xmlTLCConcept struct {
	EId    string `xml:"eId,attr"`
	Href   string `xml:"href,attr"`
	ShowAs string `xml:"showAs,attr"`
}

type xmlTLCReference struct {
	EId    string `xml:"eId,attr"`
	Href   string `xml:"href,attr"`
	ShowAs string `xml:"showAs,attr"`
}

type xmlPreface struct {
	Ps []xmlP `xml:"p"`
}

type xmlPreamble struct {
	Container xmlContainer `xml:"container"`
	Ps        []xmlP       `xml:"p"`
}

type xmlContainer struct {
	Name string `xml:"name,attr"`
	Ps   []xmlP `xml:"p"`
}

type xmlBody struct {
	Paragraphs []xmlParagraph `xml:"paragraph"`
	Points     []xmlPoint     `xml:"point"`
	Ps         []xmlP         `xml:"p"`
}

type xmlParagraph struct {
	EId     string     `xml:"eId,attr"`
	Num     string     `xml:"num"`
	Content xmlContent `xml:"content"`
}

type xmlPoint struct {
	EId     string     `xml:"eId,attr"`
	Num     string     `xml:"num"`
	Content xmlContent `xml:"content"`
}

type xmlContent struct {
	Ps []xmlP `xml:"p"`
}

type xmlP struct {
	Class   string `xml:"class,attr"`
	Content string `xml:",innerxml"`
}

type xmlConclusions struct {
	Ps []xmlP `xml:"p"`
}

// ParseAKNResolution parses Akoma Ntoso XML into a UNResolution.
func ParseAKNResolution(data []byte) (*UNResolution, error) {
	var akn xmlAkomaNtoso
	if err := xml.Unmarshal(data, &akn); err != nil {
		return nil, fmt.Errorf("failed to parse AKN XML: %w", err)
	}

	res := &UNResolution{
		Language:      "eng",
		Preamble:      make([]PreambleRecital, 0),
		OperativeParts: make([]OperativeParagraph, 0),
		References:    make([]DocumentReference, 0),
		Concepts:      make([]ConceptReference, 0),
		Organizations: make([]OrganizationReference, 0),
	}

	// Parse identification
	parseIdentification(&akn.Doc.Meta.Identification, res)

	// Parse references
	parseReferences(&akn.Doc.Meta.References, res)

	// Parse preface
	parsePreface(&akn.Doc.Preface, res)

	// Parse preamble
	parsePreamble(&akn.Doc.Preamble, res)

	// Parse body
	parseBody(&akn.Doc.Body, res)

	return res, nil
}

// parseIdentification extracts identification metadata.
func parseIdentification(id *xmlIdentification, res *UNResolution) {
	res.AKNURI = id.FRBRWork.FRBRuri.Value

	// Parse date
	if id.FRBRWork.FRBRdate.Date != "" {
		if t, err := time.Parse("2006-01-02", id.FRBRWork.FRBRdate.Date); err == nil {
			res.AdoptionDate = t
		}
	}

	// Parse language
	if id.FRBRExpr.FRBRlanguage.Language != "" {
		res.Language = id.FRBRExpr.FRBRlanguage.Language
	}

	// Extract body, session, number from URI
	// Example: /akn/un/statement/deliberation/unga/2024-12-20/79-100/!main
	uri := id.FRBRWork.FRBRuri.Value
	if strings.Contains(uri, "/unga/") {
		res.Body = BodyGeneralAssembly
		extractSessionNumber(uri, res)
	} else if strings.Contains(uri, "/unsc/") || strings.Contains(uri, "/sc/") {
		res.Body = BodySecurityCouncil
		extractSessionNumber(uri, res)
	} else if strings.Contains(uri, "/ecosoc/") {
		res.Body = BodyECOSOC
		extractSessionNumber(uri, res)
	}

	// Build document ID
	if res.DocumentID == "" {
		switch res.Body {
		case BodyGeneralAssembly:
			res.DocumentID = fmt.Sprintf("A/RES/%d/%d", res.Session, res.Number)
		case BodySecurityCouncil:
			res.DocumentID = fmt.Sprintf("S/RES/%d", res.Number)
		case BodyECOSOC:
			res.DocumentID = fmt.Sprintf("E/RES/%d/%d", res.Session, res.Number)
		}
	}
}

// extractSessionNumber extracts session and number from AKN URI.
func extractSessionNumber(uri string, res *UNResolution) {
	// Pattern: .../79-100/... or .../2798/...
	parts := strings.Split(uri, "/")
	for _, part := range parts {
		if strings.Contains(part, "-") {
			// Session-Number format
			subparts := strings.Split(part, "-")
			if len(subparts) == 2 {
				res.Session, _ = strconv.Atoi(subparts[0])
				res.Number, _ = strconv.Atoi(subparts[1])
				return
			}
		} else if n, err := strconv.Atoi(part); err == nil && n > 0 {
			// Just number (for SC resolutions)
			if res.Body == BodySecurityCouncil && n > 100 {
				res.Number = n
			}
		}
	}
}

// parseReferences extracts TLC references.
func parseReferences(refs *xmlReferences, res *UNResolution) {
	for _, org := range refs.Organizations {
		res.Organizations = append(res.Organizations, OrganizationReference{
			ID:     org.EId,
			URI:    org.Href,
			ShowAs: org.ShowAs,
		})
	}

	for _, concept := range refs.Concepts {
		res.Concepts = append(res.Concepts, ConceptReference{
			ID:     concept.EId,
			URI:    concept.Href,
			ShowAs: concept.ShowAs,
		})
	}

	for _, ref := range refs.References {
		res.References = append(res.References, DocumentReference{
			Type:       "reference",
			Identifier: ref.EId,
			URI:        ref.Href,
			ShowAs:     ref.ShowAs,
		})
	}
}

// parsePreface extracts preface metadata.
func parsePreface(preface *xmlPreface, res *UNResolution) {
	for _, p := range preface.Ps {
		text := cleanXMLContent(p.Content)
		switch p.Class {
		case "docNumber":
			// Extract resolution number from "Resolution 79/100"
			if matches := regexp.MustCompile(`(\d+)/(\d+)`).FindStringSubmatch(text); len(matches) == 3 {
				res.Session, _ = strconv.Atoi(matches[1])
				res.Number, _ = strconv.Atoi(matches[2])
			} else if matches := regexp.MustCompile(`(\d+)`).FindStringSubmatch(text); len(matches) == 2 {
				res.Number, _ = strconv.Atoi(matches[1])
			}
			if res.DocumentID == "" {
				res.DocumentID = text
			}
		case "docTitle":
			res.Title = text
		case "docProponent":
			res.Proponent = text
		}
	}
}

// parsePreamble extracts preamble recitals.
func parsePreamble(preamble *xmlPreamble, res *UNResolution) {
	// Handle container if present
	if preamble.Container.Name == "recitals" {
		for _, p := range preamble.Container.Ps {
			recital := extractRecital(p.Content)
			if recital.Text != "" {
				res.Preamble = append(res.Preamble, recital)
			}
		}
	}

	// Handle direct paragraphs
	for _, p := range preamble.Ps {
		recital := extractRecital(p.Content)
		if recital.Text != "" {
			res.Preamble = append(res.Preamble, recital)
		}
	}
}

// extractRecital extracts intro phrase and text from preamble paragraph.
func extractRecital(content string) PreambleRecital {
	text := cleanXMLContent(content)
	recital := PreambleRecital{Text: text}

	// Common intro phrases in UN resolutions
	// Ordered from longest to shortest to match most specific first
	introPhrases := []string{
		"Expressing grave concern", "Expressing deep concern",
		"Noting with deep concern", "Noting with concern",
		"Deeply concerned", "Gravely concerned",
		"Taking into account", "Bearing in mind",
		"Having considered", "Having examined", "Having received",
		"Taking note",
		"Guided by", "Recalling", "Reaffirming", "Noting",
		"Expressing", "Mindful of",
		"Welcoming", "Acknowledging", "Recognizing", "Concerned",
		"Alarmed", "Deploring", "Condemning",
		"Emphasizing", "Stressing", "Underlining",
		"Convinced", "Determined",
	}

	for _, phrase := range introPhrases {
		if strings.HasPrefix(text, phrase) {
			recital.IntroPhrase = phrase
			recital.Text = strings.TrimPrefix(text, phrase)
			recital.Text = strings.TrimSpace(recital.Text)
			break
		}
	}

	// Extract references from text
	recital.References = extractDocumentReferences(text)

	return recital
}

// parseBody extracts operative paragraphs.
func parseBody(body *xmlBody, res *UNResolution) {
	for _, para := range body.Paragraphs {
		op := parseOperativeParagraph(para)
		if op.Text != "" {
			res.OperativeParts = append(res.OperativeParts, op)
		}
	}

	// Handle points as paragraphs
	for i, point := range body.Points {
		var text strings.Builder
		for _, p := range point.Content.Ps {
			text.WriteString(cleanXMLContent(p.Content))
			text.WriteString(" ")
		}

		op := OperativeParagraph{
			Number: i + 1,
			EId:    point.EId,
			Text:   strings.TrimSpace(text.String()),
			Action: extractActionVerb(text.String()),
		}
		if op.Text != "" {
			res.OperativeParts = append(res.OperativeParts, op)
		}
	}

	// Handle direct paragraphs
	for i, p := range body.Ps {
		text := cleanXMLContent(p.Content)
		op := OperativeParagraph{
			Number: i + 1,
			Text:   text,
			Action: extractActionVerb(text),
		}
		if op.Text != "" {
			res.OperativeParts = append(res.OperativeParts, op)
		}
	}
}

// parseOperativeParagraph parses a single operative paragraph.
func parseOperativeParagraph(para xmlParagraph) OperativeParagraph {
	var text strings.Builder
	for _, p := range para.Content.Ps {
		text.WriteString(cleanXMLContent(p.Content))
		text.WriteString(" ")
	}

	num, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(para.Num), "."))

	return OperativeParagraph{
		Number: num,
		EId:    para.EId,
		Text:   strings.TrimSpace(text.String()),
		Action: extractActionVerb(text.String()),
	}
}

// extractActionVerb extracts the action verb from operative text.
func extractActionVerb(text string) string {
	// Common action verbs in UN resolutions
	actionVerbs := []string{
		"Condemns", "Strongly condemns", "Calls upon", "Urges", "Demands",
		"Decides", "Requests", "Encourages", "Welcomes", "Reaffirms",
		"Expresses", "Deplores", "Regrets", "Notes", "Takes note",
		"Authorizes", "Extends", "Renews", "Establishes", "Recommends",
		"Invites", "Appeals", "Stresses", "Emphasizes", "Underlines",
		"Affirms", "Declares", "Proclaims", "Resolves", "Confirms",
	}

	for _, verb := range actionVerbs {
		if strings.HasPrefix(text, verb) {
			return verb
		}
	}

	// Try to extract first word if italicized
	if strings.HasPrefix(text, "<i>") {
		end := strings.Index(text, "</i>")
		if end > 3 {
			return text[3:end]
		}
	}

	return ""
}

// extractDocumentReferences extracts document references from text.
func extractDocumentReferences(text string) []DocumentReference {
	var refs []DocumentReference

	// Pattern for GA resolutions: resolution 78/200, A/RES/78/200
	gaPattern := regexp.MustCompile(`(?:resolution\s+|A/RES/)(\d+)/(\d+)`)
	for _, match := range gaPattern.FindAllStringSubmatch(text, -1) {
		if len(match) == 3 {
			refs = append(refs, DocumentReference{
				Type:       "resolution",
				Identifier: fmt.Sprintf("A/RES/%s/%s", match[1], match[2]),
			})
		}
	}

	// Pattern for SC resolutions: S/RES/2798
	scPattern := regexp.MustCompile(`S/RES/(\d+)`)
	for _, match := range scPattern.FindAllStringSubmatch(text, -1) {
		if len(match) == 2 {
			refs = append(refs, DocumentReference{
				Type:       "resolution",
				Identifier: fmt.Sprintf("S/RES/%s", match[1]),
			})
		}
	}

	return refs
}

// cleanXMLContent removes XML tags and cleans whitespace.
func cleanXMLContent(content string) string {
	// Remove XML tags
	tagPattern := regexp.MustCompile(`<[^>]+>`)
	text := tagPattern.ReplaceAllString(content, "")

	// Decode common entities
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	// Normalize whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

// ResolutionToTriples converts a UNResolution to RDF triples.
func ResolutionToTriples(res *UNResolution, baseURI string) []store.Triple {
	triples := make([]store.Triple, 0)

	// Resolution URI
	resURI := fmt.Sprintf("%sresolution/%s", baseURI, sanitizeURI(res.DocumentID))

	// Basic metadata
	triples = append(triples, store.Triple{Subject: resURI, Predicate: "rdf:type", Object: "reg:Resolution"})
	triples = append(triples, store.Triple{Subject: resURI, Predicate: store.RDFSLabel, Object: res.Title})
	triples = append(triples, store.Triple{Subject: resURI, Predicate: "reg:documentID", Object: res.DocumentID})
	triples = append(triples, store.Triple{Subject: resURI, Predicate: "reg:body", Object: fmt.Sprintf("%sbody/%s", baseURI, res.Body.String())})
	triples = append(triples, store.Triple{Subject: resURI, Predicate: "reg:session", Object: fmt.Sprintf("%d", res.Session)})
	triples = append(triples, store.Triple{Subject: resURI, Predicate: store.PropNumber, Object: fmt.Sprintf("%d", res.Number)})

	if !res.AdoptionDate.IsZero() {
		triples = append(triples, store.Triple{Subject: resURI, Predicate: "reg:adoptionDate", Object: res.AdoptionDate.Format("2006-01-02")})
	}
	if res.Language != "" {
		triples = append(triples, store.Triple{Subject: resURI, Predicate: "reg:language", Object: res.Language})
	}
	if res.Proponent != "" {
		triples = append(triples, store.Triple{Subject: resURI, Predicate: "reg:proponent", Object: res.Proponent})
	}
	if res.AKNURI != "" {
		triples = append(triples, store.Triple{Subject: resURI, Predicate: "reg:aknURI", Object: res.AKNURI})
	}

	// Preamble recitals
	for i, recital := range res.Preamble {
		recitalURI := fmt.Sprintf("%s/preamble/%d", resURI, i+1)
		triples = append(triples, store.Triple{Subject: recitalURI, Predicate: "rdf:type", Object: "reg:PreambleRecital"})
		triples = append(triples, store.Triple{Subject: recitalURI, Predicate: store.PropPartOf, Object: resURI})
		triples = append(triples, store.Triple{Subject: recitalURI, Predicate: store.PropNumber, Object: fmt.Sprintf("%d", i+1)})
		if recital.IntroPhrase != "" {
			triples = append(triples, store.Triple{Subject: recitalURI, Predicate: "reg:introPhrase", Object: recital.IntroPhrase})
		}
		triples = append(triples, store.Triple{Subject: recitalURI, Predicate: store.PropText, Object: recital.Text})

		// Add references from recital
		for _, ref := range recital.References {
			refURI := fmt.Sprintf("%sresolution/%s", baseURI, sanitizeURI(ref.Identifier))
			triples = append(triples, store.Triple{Subject: recitalURI, Predicate: store.PropReferences, Object: refURI})
		}
	}

	// Operative paragraphs
	for _, para := range res.OperativeParts {
		paraURI := fmt.Sprintf("%s/para/%d", resURI, para.Number)
		triples = append(triples, store.Triple{Subject: paraURI, Predicate: "rdf:type", Object: "reg:OperativeParagraph"})
		triples = append(triples, store.Triple{Subject: paraURI, Predicate: store.PropPartOf, Object: resURI})
		triples = append(triples, store.Triple{Subject: paraURI, Predicate: store.PropNumber, Object: fmt.Sprintf("%d", para.Number)})
		if para.Action != "" {
			triples = append(triples, store.Triple{Subject: paraURI, Predicate: "reg:action", Object: para.Action})

			// Map actions to semantic predicates
			actionURI := mapActionToObligation(para.Action, baseURI)
			if actionURI != "" {
				triples = append(triples, store.Triple{Subject: paraURI, Predicate: store.PropImposesObligation, Object: actionURI})
			}
		}
		triples = append(triples, store.Triple{Subject: paraURI, Predicate: store.PropText, Object: para.Text})
	}

	// Document references
	for _, ref := range res.References {
		refURI := fmt.Sprintf("%sresolution/%s", baseURI, sanitizeURI(ref.Identifier))
		triples = append(triples, store.Triple{Subject: resURI, Predicate: store.PropReferences, Object: refURI})
	}

	// Concepts
	for _, concept := range res.Concepts {
		conceptURI := fmt.Sprintf("%sconcept/%s", baseURI, sanitizeURI(concept.ID))
		triples = append(triples, store.Triple{Subject: resURI, Predicate: "reg:concerns", Object: conceptURI})
		triples = append(triples, store.Triple{Subject: conceptURI, Predicate: "rdf:type", Object: "reg:Concept"})
		triples = append(triples, store.Triple{Subject: conceptURI, Predicate: store.RDFSLabel, Object: concept.ShowAs})
		if concept.URI != "" {
			triples = append(triples, store.Triple{Subject: conceptURI, Predicate: "reg:ontologyURI", Object: concept.URI})
		}
	}

	// Organizations
	for _, org := range res.Organizations {
		orgURI := fmt.Sprintf("%sorganization/%s", baseURI, sanitizeURI(org.ID))
		triples = append(triples, store.Triple{Subject: resURI, Predicate: "reg:involves", Object: orgURI})
		triples = append(triples, store.Triple{Subject: orgURI, Predicate: "rdf:type", Object: "reg:Organization"})
		triples = append(triples, store.Triple{Subject: orgURI, Predicate: store.RDFSLabel, Object: org.ShowAs})
		if org.URI != "" {
			triples = append(triples, store.Triple{Subject: orgURI, Predicate: "reg:ontologyURI", Object: org.URI})
		}
	}

	return triples
}

// mapActionToObligation maps action verbs to obligation URIs.
func mapActionToObligation(action string, baseURI string) string {
	switch action {
	case "Condemns", "Strongly condemns":
		return baseURI + "obligation/condemnation"
	case "Calls upon", "Urges":
		return baseURI + "obligation/call-to-action"
	case "Demands":
		return baseURI + "obligation/demand"
	case "Decides":
		return baseURI + "obligation/decision"
	case "Requests":
		return baseURI + "obligation/request"
	case "Authorizes":
		return baseURI + "obligation/authorization"
	default:
		return ""
	}
}

// sanitizeURI removes invalid characters from URI components.
func sanitizeURI(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ToLower(s)
	s = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(s, "")
	return s
}

// IngestResolutions ingests resolutions into a triple store.
func IngestResolutions(resolutions []*UNResolution, tripleStore *store.TripleStore, baseURI string) error {
	if tripleStore == nil {
		return fmt.Errorf("triple store is nil")
	}

	for _, res := range resolutions {
		triples := ResolutionToTriples(res, baseURI)
		if err := tripleStore.BulkAdd(triples); err != nil {
			return fmt.Errorf("failed to ingest resolution %s: %w", res.DocumentID, err)
		}
	}

	return nil
}
