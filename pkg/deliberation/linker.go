package deliberation

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/coolbeans/regula/pkg/store"
)

// LinkResult represents the outcome of a linking operation.
type LinkResult struct {
	// ProvisionURI is the resolved URI of the provision.
	ProvisionURI string `json:"provision_uri"`

	// RawText is the original reference text.
	RawText string `json:"raw_text"`

	// Confidence indicates how confident we are in the link (0.0-1.0).
	Confidence float64 `json:"confidence"`

	// LinkType classifies the type of link.
	LinkType string `json:"link_type"` // "explicit", "implicit", "inferred"

	// Source indicates where the reference was found.
	Source string `json:"source"` // "agenda_title", "intervention", "decision", etc.
}

// LinkingReport summarizes the linking results for a meeting.
type LinkingReport struct {
	// MeetingURI is the meeting that was processed.
	MeetingURI string `json:"meeting_uri"`

	// TotalReferences is the count of detected references.
	TotalReferences int `json:"total_references"`

	// ResolvedCount is the number successfully resolved.
	ResolvedCount int `json:"resolved_count"`

	// UnresolvedCount is the number that couldn't be resolved.
	UnresolvedCount int `json:"unresolved_count"`

	// Links contains all successful links.
	Links []LinkResult `json:"links"`

	// UnresolvedReferences lists references that couldn't be resolved.
	UnresolvedReferences []string `json:"unresolved_references,omitempty"`

	// Errors lists any errors encountered during linking.
	Errors []string `json:"errors,omitempty"`
}

// DeliberationLinker links meeting discussions to provisions in regulation graphs.
type DeliberationLinker struct {
	// regulationStore contains the regulation triples.
	regulationStore *store.TripleStore

	// baseURI is the base URI for generated links.
	baseURI string

	// Reference patterns
	articlePattern        *regexp.Regexp
	articleParenPattern   *regexp.Regexp
	sectionPattern        *regexp.Regexp
	sectionSubdivPattern  *regexp.Regexp
	chapterPattern        *regexp.Regexp
	regulationPattern     *regexp.Regexp
	directivePattern      *regexp.Regexp
	documentNumPattern    *regexp.Regexp
	meetingRefPattern     *regexp.Regexp
	unResolutionPattern   *regexp.Regexp
	euDocPattern          *regexp.Regexp

	// provisionIndex caches known provisions for resolution.
	provisionIndex map[string]bool

	// articleToURI maps article numbers to URIs.
	articleToURI map[string]string
}

// NewDeliberationLinker creates a new linker with access to a regulation store.
func NewDeliberationLinker(regulationStore *store.TripleStore, baseURI string) *DeliberationLinker {
	linker := &DeliberationLinker{
		regulationStore: regulationStore,
		baseURI:         baseURI,
		provisionIndex:  make(map[string]bool),
		articleToURI:    make(map[string]string),
	}
	linker.compilePatterns()
	linker.buildProvisionIndex()
	return linker
}

// compilePatterns initializes all regex patterns for reference detection.
func (l *DeliberationLinker) compilePatterns() {
	// Article references (EU-style)
	l.articlePattern = regexp.MustCompile(`(?i)Article\s+(\d+)`)
	l.articleParenPattern = regexp.MustCompile(`(?i)Article\s+(\d+)\((\d+)\)(?:\(([a-z])\))?`)

	// Section references (generic and US-style)
	l.sectionPattern = regexp.MustCompile(`(?i)Section\s+(\d+(?:\.\d+)*)`)
	l.sectionSubdivPattern = regexp.MustCompile(`(?i)Section\s+(\d+(?:\.\d+)*)\(([a-z])\)`)

	// Chapter references
	l.chapterPattern = regexp.MustCompile(`(?i)Chapter\s+([IVXLCDM]+|\d+)`)

	// EU regulation/directive references
	l.regulationPattern = regexp.MustCompile(`(?i)Regulation\s+\(?(?:EU|EC|EEC)?\)?\s*(?:No\.?\s*)?(\d{4}/\d+|\d+/\d{4})(?:/(?:EU|EC|EEC))?`)
	l.directivePattern = regexp.MustCompile(`(?i)Directive\s+(\d{2,4}/\d+(?:/(?:EU|EC|EEC))?)`)  // Matches "Directive 95/46/EC"

	// Document number references (COM, SEC, SWD, etc.)
	l.documentNumPattern = regexp.MustCompile(`(?i)(COM|SEC|SWD)\s*\(\d{4}\)\s*\d+`)

	// Meeting references
	l.meetingRefPattern = regexp.MustCompile(`(?i)(?:at\s+the\s+)?(\d+)(?:st|nd|rd|th)\s+meeting`)

	// UN resolution references (require "Resolution" prefix to avoid matching doc numbers)
	l.unResolutionPattern = regexp.MustCompile(`(?i)Resolution\s+([A-Z]/RES/\d+/\d+|\d+/\d+)`)

	// EU document references
	l.euDocPattern = regexp.MustCompile(`(?i)(?:document\s+)?(\d+/\d+(?:/\d+)?(?:\s+REV\s*\d*)?)`)
}

// buildProvisionIndex indexes all provisions from the regulation store.
func (l *DeliberationLinker) buildProvisionIndex() {
	if l.regulationStore == nil {
		return
	}

	// Find all articles
	articleTriples := l.regulationStore.Find("", store.RDFType, store.ClassArticle)
	for _, t := range articleTriples {
		l.provisionIndex[t.Subject] = true

		// Extract article number from label or number property
		numberTriples := l.regulationStore.Find(t.Subject, store.PropNumber, "")
		if len(numberTriples) > 0 {
			l.articleToURI[numberTriples[0].Object] = t.Subject
		}
	}

	// Find all sections
	sectionTriples := l.regulationStore.Find("", store.RDFType, store.ClassSection)
	for _, t := range sectionTriples {
		l.provisionIndex[t.Subject] = true
	}

	// Find all chapters
	chapterTriples := l.regulationStore.Find("", store.RDFType, store.ClassChapter)
	for _, t := range chapterTriples {
		l.provisionIndex[t.Subject] = true
	}

	// Find all paragraphs
	paraTriples := l.regulationStore.Find("", store.RDFType, store.ClassParagraph)
	for _, t := range paraTriples {
		l.provisionIndex[t.Subject] = true
	}

	// Find all points
	pointTriples := l.regulationStore.Find("", store.RDFType, store.ClassPoint)
	for _, t := range pointTriples {
		l.provisionIndex[t.Subject] = true
	}
}

// LinkMeetingToRegulations finds and links all provision references in a meeting.
func (l *DeliberationLinker) LinkMeetingToRegulations(meeting *Meeting, targetStore *store.TripleStore) (*LinkingReport, error) {
	if meeting == nil {
		return nil, fmt.Errorf("meeting cannot be nil")
	}

	report := &LinkingReport{
		MeetingURI: meeting.URI,
		Links:      []LinkResult{},
	}

	// Process agenda items
	for i := range meeting.AgendaItems {
		item := &meeting.AgendaItems[i]
		itemLinks, unresolved := l.findLinksInAgendaItem(item)

		for _, link := range itemLinks {
			report.Links = append(report.Links, link)

			// Add triples to target store
			if targetStore != nil {
				// Provision was discussed at meeting
				targetStore.Add(link.ProvisionURI, store.PropDiscussedAt, meeting.URI)

				// Agenda item discusses provision
				targetStore.Add(item.URI, "reg:discusses", link.ProvisionURI)
			}

			// Update agenda item's ProvisionsDiscussed
			if !contains(item.ProvisionsDiscussed, link.ProvisionURI) {
				item.ProvisionsDiscussed = append(item.ProvisionsDiscussed, link.ProvisionURI)
			}
		}

		report.UnresolvedReferences = append(report.UnresolvedReferences, unresolved...)
	}

	// Calculate statistics
	report.TotalReferences = len(report.Links) + len(report.UnresolvedReferences)
	report.ResolvedCount = len(report.Links)
	report.UnresolvedCount = len(report.UnresolvedReferences)

	return report, nil
}

// findLinksInAgendaItem extracts and resolves references from an agenda item.
func (l *DeliberationLinker) findLinksInAgendaItem(item *AgendaItem) ([]LinkResult, []string) {
	var links []LinkResult
	var unresolved []string

	// Check agenda item title
	titleRefs := l.extractReferences(item.Title)
	for _, ref := range titleRefs {
		uri, confidence := l.resolveReference(ref)
		if uri != "" {
			links = append(links, LinkResult{
				ProvisionURI: uri,
				RawText:      ref,
				Confidence:   confidence,
				LinkType:     "explicit",
				Source:       "agenda_title",
			})
		} else {
			unresolved = append(unresolved, ref)
		}
	}

	// Check agenda item description
	descRefs := l.extractReferences(item.Description)
	for _, ref := range descRefs {
		uri, confidence := l.resolveReference(ref)
		if uri != "" {
			links = append(links, LinkResult{
				ProvisionURI: uri,
				RawText:      ref,
				Confidence:   confidence,
				LinkType:     "explicit",
				Source:       "agenda_description",
			})
		} else {
			unresolved = append(unresolved, ref)
		}
	}

	// Check interventions
	for _, intervention := range item.Interventions {
		intRefs := l.extractReferences(intervention.Summary)
		for _, ref := range intRefs {
			uri, confidence := l.resolveReference(ref)
			if uri != "" {
				links = append(links, LinkResult{
					ProvisionURI: uri,
					RawText:      ref,
					Confidence:   confidence,
					LinkType:     "explicit",
					Source:       "intervention",
				})
			} else {
				unresolved = append(unresolved, ref)
			}
		}
	}

	// Check decisions
	for _, decision := range item.Decisions {
		decRefs := l.extractReferences(decision.Description)
		for _, ref := range decRefs {
			uri, confidence := l.resolveReference(ref)
			if uri != "" {
				links = append(links, LinkResult{
					ProvisionURI: uri,
					RawText:      ref,
					Confidence:   confidence,
					LinkType:     "explicit",
					Source:       "decision",
				})
			} else {
				unresolved = append(unresolved, ref)
			}
		}
	}

	// Check motions
	for _, motion := range item.Motions {
		motionRefs := l.extractReferences(motion.Text)
		for _, ref := range motionRefs {
			uri, confidence := l.resolveReference(ref)
			if uri != "" {
				links = append(links, LinkResult{
					ProvisionURI: uri,
					RawText:      ref,
					Confidence:   confidence,
					LinkType:     "explicit",
					Source:       "motion",
				})
			} else {
				unresolved = append(unresolved, ref)
			}
		}
	}

	// Check notes
	notesRefs := l.extractReferences(item.Notes)
	for _, ref := range notesRefs {
		uri, confidence := l.resolveReference(ref)
		if uri != "" {
			links = append(links, LinkResult{
				ProvisionURI: uri,
				RawText:      ref,
				Confidence:   confidence,
				LinkType:     "explicit",
				Source:       "notes",
			})
		} else {
			unresolved = append(unresolved, ref)
		}
	}

	// Deduplicate links
	links = deduplicateLinks(links)
	unresolved = deduplicateStrings(unresolved)

	return links, unresolved
}

// extractReferences finds all reference patterns in text.
func (l *DeliberationLinker) extractReferences(text string) []string {
	if text == "" {
		return nil
	}

	var refs []string
	seen := make(map[string]bool)
	// Track matched positions to avoid overlapping matches
	matchedPositions := make(map[int]int) // start position -> end position

	addRef := func(ref string) {
		ref = strings.TrimSpace(ref)
		if ref != "" && !seen[ref] {
			seen[ref] = true
			refs = append(refs, ref)
		}
	}

	isOverlapping := func(start, end int) bool {
		for s, e := range matchedPositions {
			if start >= s && start < e {
				return true
			}
			if end > s && end <= e {
				return true
			}
			if start <= s && end >= e {
				return true
			}
		}
		return false
	}

	recordMatch := func(start, end int) {
		matchedPositions[start] = end
	}

	// Extract regulation references first (longer, more specific)
	for _, indices := range l.regulationPattern.FindAllStringIndex(text, -1) {
		if !isOverlapping(indices[0], indices[1]) {
			addRef(text[indices[0]:indices[1]])
			recordMatch(indices[0], indices[1])
		}
	}

	// Extract directive references (longer, more specific)
	for _, indices := range l.directivePattern.FindAllStringIndex(text, -1) {
		if !isOverlapping(indices[0], indices[1]) {
			addRef(text[indices[0]:indices[1]])
			recordMatch(indices[0], indices[1])
		}
	}

	// Extract article references with paragraphs first (more specific)
	for _, indices := range l.articleParenPattern.FindAllStringIndex(text, -1) {
		if !isOverlapping(indices[0], indices[1]) {
			addRef(text[indices[0]:indices[1]])
			recordMatch(indices[0], indices[1])
		}
	}

	// Then extract simple article references
	for _, indices := range l.articlePattern.FindAllStringIndex(text, -1) {
		if !isOverlapping(indices[0], indices[1]) {
			addRef(text[indices[0]:indices[1]])
			recordMatch(indices[0], indices[1])
		}
	}

	// Extract section references
	for _, indices := range l.sectionSubdivPattern.FindAllStringIndex(text, -1) {
		if !isOverlapping(indices[0], indices[1]) {
			addRef(text[indices[0]:indices[1]])
			recordMatch(indices[0], indices[1])
		}
	}
	for _, indices := range l.sectionPattern.FindAllStringIndex(text, -1) {
		if !isOverlapping(indices[0], indices[1]) {
			addRef(text[indices[0]:indices[1]])
			recordMatch(indices[0], indices[1])
		}
	}

	// Extract chapter references
	for _, indices := range l.chapterPattern.FindAllStringIndex(text, -1) {
		if !isOverlapping(indices[0], indices[1]) {
			addRef(text[indices[0]:indices[1]])
			recordMatch(indices[0], indices[1])
		}
	}

	// Extract document number references
	for _, indices := range l.documentNumPattern.FindAllStringIndex(text, -1) {
		if !isOverlapping(indices[0], indices[1]) {
			addRef(text[indices[0]:indices[1]])
			recordMatch(indices[0], indices[1])
		}
	}

	// Extract UN resolution references
	for _, indices := range l.unResolutionPattern.FindAllStringIndex(text, -1) {
		if !isOverlapping(indices[0], indices[1]) {
			addRef(text[indices[0]:indices[1]])
			recordMatch(indices[0], indices[1])
		}
	}

	return refs
}

// resolveReference attempts to resolve a reference string to a provision URI.
func (l *DeliberationLinker) resolveReference(ref string) (string, float64) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", 0
	}

	// Try article pattern
	if match := l.articleParenPattern.FindStringSubmatch(ref); match != nil {
		articleNum := match[1]
		paraNum := match[2]
		var pointLetter string
		if len(match) > 3 && match[3] != "" {
			pointLetter = match[3]
		}
		return l.resolveArticleRef(articleNum, paraNum, pointLetter)
	}

	if match := l.articlePattern.FindStringSubmatch(ref); match != nil {
		articleNum := match[1]
		return l.resolveArticleRef(articleNum, "", "")
	}

	// Try section pattern
	if match := l.sectionSubdivPattern.FindStringSubmatch(ref); match != nil {
		sectionNum := match[1]
		subdiv := match[2]
		return l.resolveSectionRef(sectionNum, subdiv)
	}

	if match := l.sectionPattern.FindStringSubmatch(ref); match != nil {
		sectionNum := match[1]
		return l.resolveSectionRef(sectionNum, "")
	}

	// Try chapter pattern
	if match := l.chapterPattern.FindStringSubmatch(ref); match != nil {
		chapterNum := match[1]
		return l.resolveChapterRef(chapterNum)
	}

	// Try regulation pattern (external reference)
	if match := l.regulationPattern.FindStringSubmatch(ref); match != nil {
		regNum := match[1]
		return l.resolveRegulationRef(regNum)
	}

	// Try directive pattern (external reference)
	if match := l.directivePattern.FindStringSubmatch(ref); match != nil {
		dirNum := match[1]
		return l.resolveDirectiveRef(dirNum)
	}

	return "", 0
}

// resolveArticleRef resolves an article reference to a URI.
func (l *DeliberationLinker) resolveArticleRef(articleNum, paraNum, pointLetter string) (string, float64) {
	// Check direct mapping
	if uri, ok := l.articleToURI[articleNum]; ok {
		if paraNum != "" {
			// Try to find the specific paragraph
			paraURI := uri + ":" + paraNum
			if pointLetter != "" {
				paraURI += ":" + pointLetter
			}
			if l.provisionIndex[paraURI] {
				return paraURI, 1.0
			}
			// Return article with partial confidence
			return uri, 0.75
		}
		return uri, 1.0
	}

	// Try constructing URI
	uri := fmt.Sprintf("%sArt%s", l.baseURI, articleNum)
	if l.provisionIndex[uri] {
		if paraNum != "" {
			paraURI := uri + ":" + paraNum
			if pointLetter != "" {
				paraURI += ":" + pointLetter
			}
			if l.provisionIndex[paraURI] {
				return paraURI, 1.0
			}
			return uri, 0.75
		}
		return uri, 1.0
	}

	// Not found, but return a constructed URI with low confidence
	if paraNum != "" {
		uri = fmt.Sprintf("%sArt%s:%s", l.baseURI, articleNum, paraNum)
		if pointLetter != "" {
			uri += ":" + pointLetter
		}
	}
	return uri, 0.25
}

// resolveSectionRef resolves a section reference to a URI.
func (l *DeliberationLinker) resolveSectionRef(sectionNum, subdiv string) (string, float64) {
	uri := fmt.Sprintf("%sSection%s", l.baseURI, sectionNum)
	if subdiv != "" {
		uri += ":" + subdiv
	}

	if l.provisionIndex[uri] {
		return uri, 1.0
	}

	// Return with low confidence
	return uri, 0.25
}

// resolveChapterRef resolves a chapter reference to a URI.
func (l *DeliberationLinker) resolveChapterRef(chapterNum string) (string, float64) {
	uri := fmt.Sprintf("%sChapter%s", l.baseURI, chapterNum)

	if l.provisionIndex[uri] {
		return uri, 1.0
	}

	// Return with low confidence
	return uri, 0.25
}

// resolveRegulationRef resolves a regulation reference to a URI.
func (l *DeliberationLinker) resolveRegulationRef(regNum string) (string, float64) {
	// Normalize the number format
	regNum = strings.ReplaceAll(regNum, " ", "")

	uri := fmt.Sprintf("https://regula.dev/regulations/EU/%s", regNum)

	// External references always have medium confidence unless verified
	return uri, 0.5
}

// resolveDirectiveRef resolves a directive reference to a URI.
func (l *DeliberationLinker) resolveDirectiveRef(dirNum string) (string, float64) {
	dirNum = strings.ReplaceAll(dirNum, " ", "")
	uri := fmt.Sprintf("https://regula.dev/directives/EU/%s", dirNum)
	return uri, 0.5
}

// FindDiscussedProvisions returns all provisions discussed in an agenda item.
func (l *DeliberationLinker) FindDiscussedProvisions(item *AgendaItem) ([]string, error) {
	if item == nil {
		return nil, fmt.Errorf("agenda item cannot be nil")
	}

	links, _ := l.findLinksInAgendaItem(item)

	uris := make([]string, 0, len(links))
	seen := make(map[string]bool)

	for _, link := range links {
		if !seen[link.ProvisionURI] {
			seen[link.ProvisionURI] = true
			uris = append(uris, link.ProvisionURI)
		}
	}

	return uris, nil
}

// ResolveReference attempts to resolve a single reference string.
func (l *DeliberationLinker) ResolveReference(ref string) (string, error) {
	uri, confidence := l.resolveReference(ref)
	if uri == "" {
		return "", fmt.Errorf("could not resolve reference: %s", ref)
	}
	if confidence < 0.5 {
		return uri, fmt.Errorf("low confidence resolution for: %s (confidence: %.2f)", ref, confidence)
	}
	return uri, nil
}

// GetProvisionMeetings returns all meetings where a provision was discussed.
func (l *DeliberationLinker) GetProvisionMeetings(provisionURI string, meetingStore *store.TripleStore) ([]string, error) {
	if meetingStore == nil {
		return nil, fmt.Errorf("meeting store cannot be nil")
	}

	triples := meetingStore.Find(provisionURI, store.PropDiscussedAt, "")
	meetings := make([]string, 0, len(triples))
	for _, t := range triples {
		meetings = append(meetings, t.Object)
	}

	return meetings, nil
}

// GetMeetingProvisions returns all provisions discussed in a meeting.
func (l *DeliberationLinker) GetMeetingProvisions(meetingURI string, meetingStore *store.TripleStore) ([]string, error) {
	if meetingStore == nil {
		return nil, fmt.Errorf("meeting store cannot be nil")
	}

	// Find all agenda items for this meeting
	agendaTriples := meetingStore.Find("", store.PropHasAgendaItem, "")
	var provisions []string
	seen := make(map[string]bool)

	for _, t := range agendaTriples {
		// Check if this agenda item belongs to the meeting
		if strings.Contains(t.Subject, meetingURI) || t.Subject == meetingURI {
			// Find provisions discussed in this agenda item
			discussTriples := meetingStore.Find(t.Object, "reg:discusses", "")
			for _, dt := range discussTriples {
				if !seen[dt.Object] {
					seen[dt.Object] = true
					provisions = append(provisions, dt.Object)
				}
			}
		}
	}

	// Also check direct discussedAt links (inverse query)
	discussedTriples := meetingStore.Find("", store.PropDiscussedAt, meetingURI)
	for _, t := range discussedTriples {
		if !seen[t.Subject] {
			seen[t.Subject] = true
			provisions = append(provisions, t.Subject)
		}
	}

	return provisions, nil
}

// LinkResolutionToRegulations links a resolution's references to regulations.
func (l *DeliberationLinker) LinkResolutionToRegulations(resolution *Resolution, targetStore *store.TripleStore) (*LinkingReport, error) {
	if resolution == nil {
		return nil, fmt.Errorf("resolution cannot be nil")
	}

	report := &LinkingReport{
		MeetingURI: resolution.URI,
		Links:      []LinkResult{},
	}

	// Process preamble recitals
	for _, recital := range resolution.Preamble {
		refs := l.extractReferences(recital.Text)
		for _, ref := range refs {
			uri, confidence := l.resolveReference(ref)
			if uri != "" {
				link := LinkResult{
					ProvisionURI: uri,
					RawText:      ref,
					Confidence:   confidence,
					LinkType:     "explicit",
					Source:       "recital",
				}
				report.Links = append(report.Links, link)

				if targetStore != nil {
					targetStore.Add(resolution.URI, store.PropReferences, uri)
				}
			} else {
				report.UnresolvedReferences = append(report.UnresolvedReferences, ref)
			}
		}
	}

	// Process operative clauses
	for _, clause := range resolution.OperativeClauses {
		refs := l.extractReferences(clause.Text)
		for _, ref := range refs {
			uri, confidence := l.resolveReference(ref)
			if uri != "" {
				link := LinkResult{
					ProvisionURI: uri,
					RawText:      ref,
					Confidence:   confidence,
					LinkType:     "explicit",
					Source:       "operative_clause",
				}
				report.Links = append(report.Links, link)

				if targetStore != nil {
					targetStore.Add(resolution.URI, store.PropReferences, uri)
				}
			} else {
				report.UnresolvedReferences = append(report.UnresolvedReferences, ref)
			}
		}
	}

	// Calculate statistics
	report.TotalReferences = len(report.Links) + len(report.UnresolvedReferences)
	report.ResolvedCount = len(report.Links)
	report.UnresolvedCount = len(report.UnresolvedReferences)

	return report, nil
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func deduplicateLinks(links []LinkResult) []LinkResult {
	seen := make(map[string]LinkResult)
	for _, link := range links {
		key := link.ProvisionURI + "|" + link.Source
		if existing, ok := seen[key]; ok {
			// Keep the one with higher confidence
			if link.Confidence > existing.Confidence {
				seen[key] = link
			}
		} else {
			seen[key] = link
		}
	}

	result := make([]LinkResult, 0, len(seen))
	for _, link := range seen {
		result = append(result, link)
	}

	// Sort by provision URI for consistent output
	sort.Slice(result, func(i, j int) bool {
		return result[i].ProvisionURI < result[j].ProvisionURI
	})

	return result
}

func deduplicateStrings(strs []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(strs))
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
