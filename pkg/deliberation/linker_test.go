package deliberation

import (
	"testing"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// buildTestRegulationStore creates a test regulation store with sample provisions.
func buildTestRegulationStore() *store.TripleStore {
	ts := store.NewTripleStore()

	baseURI := "https://regula.dev/regulations/GDPR#"

	// Add articles
	for i := 1; i <= 99; i++ {
		uri := baseURI + "Art" + itoa(i)
		ts.Add(uri, store.RDFType, store.ClassArticle)
		ts.Add(uri, store.PropNumber, itoa(i))
		ts.Add(uri, store.PropTitle, "Article "+itoa(i))
	}

	// Add paragraphs for Article 6
	for para := 1; para <= 4; para++ {
		uri := baseURI + "Art6:" + itoa(para)
		ts.Add(uri, store.RDFType, store.ClassParagraph)
		ts.Add(uri, store.PropPartOf, baseURI+"Art6")
	}

	// Add points for Article 6(1)
	for _, letter := range []string{"a", "b", "c", "d", "e", "f"} {
		uri := baseURI + "Art6:1:" + letter
		ts.Add(uri, store.RDFType, store.ClassPoint)
		ts.Add(uri, store.PropPartOf, baseURI+"Art6:1")
	}

	// Add chapters
	for _, chapter := range []string{"I", "II", "III", "IV", "V"} {
		uri := baseURI + "Chapter" + chapter
		ts.Add(uri, store.RDFType, store.ClassChapter)
		ts.Add(uri, store.PropNumber, chapter)
	}

	return ts
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	if i < 0 {
		return "-" + itoa(-i)
	}
	digits := ""
	for i > 0 {
		digits = string(rune('0'+i%10)) + digits
		i /= 10
	}
	return digits
}

func TestNewDeliberationLinker(t *testing.T) {
	regStore := buildTestRegulationStore()
	linker := NewDeliberationLinker(regStore, "https://regula.dev/regulations/GDPR#")

	if linker == nil {
		t.Fatal("Expected non-nil linker")
	}
	if linker.regulationStore != regStore {
		t.Error("Expected regulation store to be set")
	}
	if len(linker.provisionIndex) == 0 {
		t.Error("Expected provision index to be built")
	}
}

func TestDeliberationLinker_ExtractReferences(t *testing.T) {
	linker := NewDeliberationLinker(nil, "https://example.org/")

	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "article reference",
			text:     "As required by Article 6, processing must have a lawful basis.",
			expected: []string{"Article 6"},
		},
		{
			name:     "article with paragraph",
			text:     "According to Article 17(1), data subjects have the right to erasure.",
			expected: []string{"Article 17(1)"},
		},
		{
			name:     "article with paragraph and point",
			text:     "Based on Article 6(1)(a), consent is required.",
			expected: []string{"Article 6(1)(a)"},
		},
		{
			name:     "multiple articles separate",
			text:     "Article 12 and Article 17 cover transparency and erasure.",
			expected: []string{"Article 12", "Article 17"},
		},
		{
			name:     "chapter reference",
			text:     "Chapter III sets out data subject rights.",
			expected: []string{"Chapter III"},
		},
		{
			name:     "regulation reference",
			text:     "In accordance with Regulation (EU) 2016/679 (the GDPR).",
			expected: []string{"Regulation (EU) 2016/679"},
		},
		{
			name:     "directive reference",
			text:     "Directive 95/46/EC was repealed by the GDPR.",
			expected: []string{"Directive 95/46/EC"},
		},
		{
			name:     "section reference",
			text:     "Section 1798.100 defines consumer rights.",
			expected: []string{"Section 1798.100"},
		},
		{
			name:     "mixed references",
			text:     "Article 5 and Chapter II implement Regulation (EU) 2016/679.",
			expected: []string{"Regulation (EU) 2016/679", "Article 5", "Chapter II"},
		},
		{
			name:     "empty text",
			text:     "",
			expected: nil,
		},
		{
			name:     "no references",
			text:     "The committee discussed general matters.",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := linker.extractReferences(tt.text)

			if len(refs) != len(tt.expected) {
				t.Errorf("Expected %d references, got %d: %v", len(tt.expected), len(refs), refs)
				return
			}

			for _, exp := range tt.expected {
				found := false
				for _, ref := range refs {
					if ref == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find reference '%s' in %v", exp, refs)
				}
			}
		})
	}
}

func TestDeliberationLinker_ResolveReference(t *testing.T) {
	regStore := buildTestRegulationStore()
	linker := NewDeliberationLinker(regStore, "https://regula.dev/regulations/GDPR#")

	tests := []struct {
		name           string
		ref            string
		expectedURI    string
		minConfidence  float64
		expectError    bool
	}{
		{
			name:          "article found",
			ref:           "Article 6",
			expectedURI:   "https://regula.dev/regulations/GDPR#Art6",
			minConfidence: 1.0,
			expectError:   false,
		},
		{
			name:          "article with paragraph",
			ref:           "Article 6(1)",
			expectedURI:   "https://regula.dev/regulations/GDPR#Art6:1",
			minConfidence: 1.0,
			expectError:   false,
		},
		{
			name:          "article with paragraph and point",
			ref:           "Article 6(1)(a)",
			expectedURI:   "https://regula.dev/regulations/GDPR#Art6:1:a",
			minConfidence: 1.0,
			expectError:   false,
		},
		{
			name:          "chapter found",
			ref:           "Chapter III",
			expectedURI:   "https://regula.dev/regulations/GDPR#ChapterIII",
			minConfidence: 1.0,
			expectError:   false,
		},
		{
			name:          "article not found low confidence",
			ref:           "Article 999",
			expectedURI:   "https://regula.dev/regulations/GDPR#Art999",
			minConfidence: 0.0,
			expectError:   true,
		},
		{
			name:          "external regulation",
			ref:           "Regulation (EU) 2016/679",
			expectedURI:   "https://regula.dev/regulations/EU/2016/679",
			minConfidence: 0.5,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri, err := linker.ResolveReference(tt.ref)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error")
				}
				return
			}

			if err != nil && !tt.expectError {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if uri != tt.expectedURI {
				t.Errorf("Expected URI %s, got %s", tt.expectedURI, uri)
			}
		})
	}
}

func TestDeliberationLinker_LinkMeetingToRegulations(t *testing.T) {
	regStore := buildTestRegulationStore()
	linker := NewDeliberationLinker(regStore, "https://regula.dev/regulations/GDPR#")
	targetStore := store.NewTripleStore()

	meeting := &Meeting{
		URI:        "https://example.org/meetings/wg-43",
		Identifier: "WG-43",
		Title:      "Working Group Meeting 43",
		Date:       time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
		AgendaItems: []AgendaItem{
			{
				URI:         "https://example.org/meetings/wg-43/item1",
				Number:      "1",
				Title:       "Discussion on Article 17 implementation",
				Description: "The committee will discuss Article 17(1) erasure requests.",
				MeetingURI:  "https://example.org/meetings/wg-43",
			},
			{
				URI:         "https://example.org/meetings/wg-43/item2",
				Number:      "2",
				Title:       "Review of Chapter III provisions",
				Description: "Review of data subject rights in Chapter III, particularly Article 6(1)(a) consent.",
				MeetingURI:  "https://example.org/meetings/wg-43",
			},
		},
	}

	report, err := linker.LinkMeetingToRegulations(meeting, targetStore)
	if err != nil {
		t.Fatalf("LinkMeetingToRegulations failed: %v", err)
	}

	t.Logf("Report: %d total, %d resolved, %d unresolved",
		report.TotalReferences, report.ResolvedCount, report.UnresolvedCount)

	if report.ResolvedCount == 0 {
		t.Error("Expected some resolved references")
	}

	// Check that links were created
	for _, link := range report.Links {
		t.Logf("Link: %s -> %s (confidence: %.2f, source: %s)",
			link.RawText, link.ProvisionURI, link.Confidence, link.Source)
	}

	// Verify triples were added to target store
	discussedTriples := targetStore.Find("", store.PropDiscussedAt, meeting.URI)
	if len(discussedTriples) == 0 {
		t.Error("Expected discussedAt triples to be created")
	}

	// Verify agenda item was updated
	if len(meeting.AgendaItems[0].ProvisionsDiscussed) == 0 {
		t.Error("Expected ProvisionsDiscussed to be updated")
	}
}

func TestDeliberationLinker_FindDiscussedProvisions(t *testing.T) {
	regStore := buildTestRegulationStore()
	linker := NewDeliberationLinker(regStore, "https://regula.dev/regulations/GDPR#")

	item := &AgendaItem{
		URI:         "https://example.org/meetings/wg-43/item1",
		Number:      "1",
		Title:       "Discussion on Article 6 and Article 17",
		Description: "Review lawful basis under Article 6(1)(a) and erasure under Article 17.",
		MeetingURI:  "https://example.org/meetings/wg-43",
	}

	provisions, err := linker.FindDiscussedProvisions(item)
	if err != nil {
		t.Fatalf("FindDiscussedProvisions failed: %v", err)
	}

	if len(provisions) < 2 {
		t.Errorf("Expected at least 2 provisions, got %d: %v", len(provisions), provisions)
	}

	t.Logf("Found provisions: %v", provisions)
}

func TestDeliberationLinker_GetProvisionMeetings(t *testing.T) {
	linker := NewDeliberationLinker(nil, "https://example.org/")
	meetingStore := store.NewTripleStore()

	provisionURI := "https://regula.dev/regulations/GDPR#Art17"
	meeting1 := "https://example.org/meetings/wg-42"
	meeting2 := "https://example.org/meetings/wg-43"

	// Add discussion links
	meetingStore.Add(provisionURI, store.PropDiscussedAt, meeting1)
	meetingStore.Add(provisionURI, store.PropDiscussedAt, meeting2)

	meetings, err := linker.GetProvisionMeetings(provisionURI, meetingStore)
	if err != nil {
		t.Fatalf("GetProvisionMeetings failed: %v", err)
	}

	if len(meetings) != 2 {
		t.Errorf("Expected 2 meetings, got %d", len(meetings))
	}
}

func TestDeliberationLinker_GetMeetingProvisions(t *testing.T) {
	linker := NewDeliberationLinker(nil, "https://example.org/")
	meetingStore := store.NewTripleStore()

	meetingURI := "https://example.org/meetings/wg-43"
	provision1 := "https://regula.dev/regulations/GDPR#Art6"
	provision2 := "https://regula.dev/regulations/GDPR#Art17"

	// Add discussion links (inverse of discussedAt)
	meetingStore.Add(provision1, store.PropDiscussedAt, meetingURI)
	meetingStore.Add(provision2, store.PropDiscussedAt, meetingURI)

	provisions, err := linker.GetMeetingProvisions(meetingURI, meetingStore)
	if err != nil {
		t.Fatalf("GetMeetingProvisions failed: %v", err)
	}

	if len(provisions) != 2 {
		t.Errorf("Expected 2 provisions, got %d: %v", len(provisions), provisions)
	}
}

func TestDeliberationLinker_LinkResolutionToRegulations(t *testing.T) {
	regStore := buildTestRegulationStore()
	linker := NewDeliberationLinker(regStore, "https://regula.dev/regulations/GDPR#")
	targetStore := store.NewTripleStore()

	resolution := &Resolution{
		URI:          "https://example.org/resolutions/res-2024-1",
		Identifier:   "2024/1",
		Title:        "Resolution on Data Protection",
		AdoptingBody: "Data Protection Board",
		Preamble: []Recital{
			{
				URI:         "https://example.org/resolutions/res-2024-1/recital1",
				Number:      1,
				IntroPhrase: "Recalling",
				Text:        "Recalling Article 5 of Regulation (EU) 2016/679,",
			},
			{
				URI:         "https://example.org/resolutions/res-2024-1/recital2",
				Number:      2,
				IntroPhrase: "Noting",
				Text:        "Noting the provisions of Article 6(1)(a) and Article 17,",
			},
		},
		OperativeClauses: []OperativeClause{
			{
				URI:        "https://example.org/resolutions/res-2024-1/clause1",
				Number:     1,
				ActionVerb: "Decides",
				Text:       "Decides to implement guidelines under Article 12;",
			},
		},
	}

	report, err := linker.LinkResolutionToRegulations(resolution, targetStore)
	if err != nil {
		t.Fatalf("LinkResolutionToRegulations failed: %v", err)
	}

	t.Logf("Resolution linking: %d resolved, %d unresolved",
		report.ResolvedCount, report.UnresolvedCount)

	if report.ResolvedCount == 0 {
		t.Error("Expected some resolved references")
	}

	// Verify triples were added
	refTriples := targetStore.Find(resolution.URI, store.PropReferences, "")
	if len(refTriples) == 0 {
		t.Error("Expected reference triples to be created")
	}

	for _, link := range report.Links {
		t.Logf("Resolution link: %s -> %s (source: %s)",
			link.RawText, link.ProvisionURI, link.Source)
	}
}

func TestDeliberationLinker_NilInputs(t *testing.T) {
	linker := NewDeliberationLinker(nil, "https://example.org/")

	// Test nil meeting
	_, err := linker.LinkMeetingToRegulations(nil, nil)
	if err == nil {
		t.Error("Expected error for nil meeting")
	}

	// Test nil agenda item
	_, err = linker.FindDiscussedProvisions(nil)
	if err == nil {
		t.Error("Expected error for nil agenda item")
	}

	// Test nil resolution
	_, err = linker.LinkResolutionToRegulations(nil, nil)
	if err == nil {
		t.Error("Expected error for nil resolution")
	}

	// Test nil meeting store
	_, err = linker.GetProvisionMeetings("uri", nil)
	if err == nil {
		t.Error("Expected error for nil meeting store")
	}

	_, err = linker.GetMeetingProvisions("uri", nil)
	if err == nil {
		t.Error("Expected error for nil meeting store")
	}
}

func TestDeliberationLinker_InterventionReferences(t *testing.T) {
	regStore := buildTestRegulationStore()
	linker := NewDeliberationLinker(regStore, "https://regula.dev/regulations/GDPR#")
	targetStore := store.NewTripleStore()

	meeting := &Meeting{
		URI:        "https://example.org/meetings/wg-43",
		Identifier: "WG-43",
		AgendaItems: []AgendaItem{
			{
				URI:        "https://example.org/meetings/wg-43/item1",
				Number:     "1",
				Title:      "General discussion",
				MeetingURI: "https://example.org/meetings/wg-43",
				Interventions: []Intervention{
					{
						URI:        "https://example.org/meetings/wg-43/item1/int1",
						SpeakerURI: "https://example.org/stakeholders/member1",
						Summary:    "The member raised concerns about Article 6(1)(a) consent requirements.",
						Position:   PositionOppose,
					},
					{
						URI:        "https://example.org/meetings/wg-43/item1/int2",
						SpeakerURI: "https://example.org/stakeholders/member2",
						Summary:    "Another member supported the interpretation of Article 17.",
						Position:   PositionSupport,
					},
				},
			},
		},
	}

	report, err := linker.LinkMeetingToRegulations(meeting, targetStore)
	if err != nil {
		t.Fatalf("LinkMeetingToRegulations failed: %v", err)
	}

	// Check for intervention sources
	interventionLinks := 0
	for _, link := range report.Links {
		if link.Source == "intervention" {
			interventionLinks++
		}
	}

	if interventionLinks < 2 {
		t.Errorf("Expected at least 2 intervention links, got %d", interventionLinks)
	}
}

func TestLinkingReport(t *testing.T) {
	report := &LinkingReport{
		MeetingURI:      "https://example.org/meetings/wg-43",
		TotalReferences: 5,
		ResolvedCount:   4,
		UnresolvedCount: 1,
		Links: []LinkResult{
			{ProvisionURI: "uri1", Confidence: 1.0},
			{ProvisionURI: "uri2", Confidence: 0.75},
		},
		UnresolvedReferences: []string{"unknown ref"},
	}

	if report.TotalReferences != 5 {
		t.Errorf("Expected 5 total references, got %d", report.TotalReferences)
	}
	if len(report.Links) != 2 {
		t.Errorf("Expected 2 links, got %d", len(report.Links))
	}
	if len(report.UnresolvedReferences) != 1 {
		t.Errorf("Expected 1 unresolved reference, got %d", len(report.UnresolvedReferences))
	}
}

func TestDeduplicateLinks(t *testing.T) {
	links := []LinkResult{
		{ProvisionURI: "uri1", Source: "title", Confidence: 0.5},
		{ProvisionURI: "uri1", Source: "title", Confidence: 1.0}, // Duplicate with higher confidence
		{ProvisionURI: "uri2", Source: "description", Confidence: 0.75},
		{ProvisionURI: "uri1", Source: "description", Confidence: 0.5}, // Same URI, different source
	}

	result := deduplicateLinks(links)

	// Should have 3 unique combinations (uri1|title, uri2|description, uri1|description)
	if len(result) != 3 {
		t.Errorf("Expected 3 deduplicated links, got %d", len(result))
	}

	// The uri1|title link should have confidence 1.0 (the higher one)
	for _, link := range result {
		if link.ProvisionURI == "uri1" && link.Source == "title" {
			if link.Confidence != 1.0 {
				t.Errorf("Expected confidence 1.0 for duplicate, got %.2f", link.Confidence)
			}
		}
	}
}
