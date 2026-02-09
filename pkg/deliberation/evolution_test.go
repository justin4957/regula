package deliberation

import (
	"testing"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// buildEvolutionTestStore creates a test store with provision evolution data.
func buildEvolutionTestStore() *store.TripleStore {
	ts := store.NewTripleStore()

	baseURI := "https://regula.dev/regulations/GDPR#"

	// Create Article 17 with evolution history
	art17 := baseURI + "Art17"
	ts.Add(art17, store.RDFType, store.ClassArticle)
	ts.Add(art17, store.RDFSLabel, "Article 17")
	ts.Add(art17, store.PropTitle, "Right to erasure")
	ts.Add(art17, store.PropPartOf, baseURI+"ChapterIII")

	// Version 1 - Initial proposal
	v1 := art17 + ":v1"
	ts.Add(v1, store.RDFType, "reg:ProposedText")
	ts.Add(v1, store.PropVersionOf, art17)
	ts.Add(v1, store.PropVersionNumber, "v1")
	ts.Add(v1, "reg:eventType", "proposed")
	ts.Add(v1, "reg:proposedAt", "https://example.org/meetings/wg-38")
	ts.Add(v1, store.PropProposedBy, "https://example.org/delegations/DE")
	ts.Add(v1, store.PropText, "Original proposed text for right to erasure.")
	ts.Add(v1, "reg:eventDate", "2024-01-15T10:00:00Z")

	// Version 2 - Amendment
	v2 := art17 + ":v2"
	ts.Add(v2, store.RDFType, "reg:AmendedText")
	ts.Add(v2, store.PropVersionOf, art17)
	ts.Add(v2, store.PropVersionNumber, "v2")
	ts.Add(v2, "reg:eventType", "amended")
	ts.Add(v2, "reg:amendedAt", "https://example.org/meetings/wg-41")
	ts.Add(v2, store.PropProposedBy, "https://example.org/delegations/FR")
	ts.Add(v2, store.PropText, "Amended text with additional exceptions.")
	ts.Add(v2, store.PropPreviousVersion, v1)
	ts.Add(v2, store.PropAmends, v1)
	ts.Add(v2, "reg:eventDate", "2024-02-20T14:00:00Z")
	ts.Add(v1, store.PropNextVersion, v2)

	// Version 3 - Final adoption
	v3 := art17 + ":v3"
	ts.Add(v3, store.RDFType, "reg:AdoptedText")
	ts.Add(v3, store.PropVersionOf, art17)
	ts.Add(v3, store.PropVersionNumber, "v3")
	ts.Add(v3, "reg:eventType", "adopted")
	ts.Add(v3, "reg:adoptedAt", "https://example.org/meetings/wg-43")
	ts.Add(v3, store.PropText, "Final adopted text for right to erasure with exceptions.")
	ts.Add(v3, store.PropPreviousVersion, v2)
	ts.Add(v3, "reg:eventDate", "2024-03-10T09:00:00Z")
	ts.Add(v2, store.PropNextVersion, v3)

	// Vote record for adoption
	vote := v3 + ":vote"
	ts.Add(v3, "reg:hasVote", vote)
	ts.Add(vote, store.RDFType, store.ClassVoteRecord)
	ts.Add(vote, store.PropVoteFor, "24")
	ts.Add(vote, store.PropVoteAgainst, "3")
	ts.Add(vote, store.PropVoteAbstain, "1")
	ts.Add(vote, store.PropVoteResult, "adopted")

	// Set current version
	ts.Add(art17, store.PropCurrentVersion, v3)

	// Create Chapter III
	chapterIII := baseURI + "ChapterIII"
	ts.Add(chapterIII, store.RDFType, store.ClassChapter)
	ts.Add(chapterIII, store.RDFSLabel, "Chapter III")
	ts.Add(chapterIII, store.PropTitle, "Rights of the data subject")

	// Create Article 18 with different evolution
	art18 := baseURI + "Art18"
	ts.Add(art18, store.RDFType, store.ClassArticle)
	ts.Add(art18, store.RDFSLabel, "Article 18")
	ts.Add(art18, store.PropTitle, "Right to restriction of processing")
	ts.Add(art18, store.PropPartOf, chapterIII)

	// Article 18 - Amendment by ES
	art18v1 := art18 + ":v1"
	ts.Add(art18v1, store.RDFType, "reg:AmendedText")
	ts.Add(art18v1, store.PropVersionOf, art18)
	ts.Add(art18v1, store.PropVersionNumber, "v1")
	ts.Add(art18v1, "reg:eventType", "amended")
	ts.Add(art18v1, "reg:amendedAt", "https://example.org/meetings/wg-40")
	ts.Add(art18v1, store.PropProposedBy, "https://example.org/delegations/ES")
	ts.Add(art18v1, store.PropText, "Text for right to restriction.")
	ts.Add(art18v1, "reg:eventDate", "2024-02-01T11:00:00Z")

	// Create meetings
	meetings := []struct {
		uri   string
		label string
		date  string
	}{
		{"https://example.org/meetings/wg-38", "Working Group Meeting 38", "2024-01-15"},
		{"https://example.org/meetings/wg-40", "Working Group Meeting 40", "2024-02-01"},
		{"https://example.org/meetings/wg-41", "Working Group Meeting 41", "2024-02-20"},
		{"https://example.org/meetings/wg-43", "Working Group Meeting 43", "2024-03-10"},
	}
	for _, m := range meetings {
		ts.Add(m.uri, store.RDFType, store.ClassMeeting)
		ts.Add(m.uri, store.RDFSLabel, m.label)
		ts.Add(m.uri, store.PropMeetingDate, m.date)
	}

	// Create delegations/stakeholders
	delegations := []struct {
		uri  string
		name string
	}{
		{"https://example.org/delegations/DE", "Germany"},
		{"https://example.org/delegations/FR", "France"},
		{"https://example.org/delegations/ES", "Spain"},
	}
	for _, d := range delegations {
		ts.Add(d.uri, store.RDFType, store.ClassStakeholder)
		ts.Add(d.uri, store.RDFSLabel, d.name)
	}

	return ts
}

func TestNewEvolutionTracker(t *testing.T) {
	ts := buildEvolutionTestStore()
	tracker := NewEvolutionTracker(ts, "https://regula.dev/regulations/GDPR#")

	if tracker == nil {
		t.Fatal("Expected non-nil tracker")
	}
	if tracker.store != ts {
		t.Error("Expected store to be set")
	}
}

func TestEvolutionTracker_GetEvolution(t *testing.T) {
	ts := buildEvolutionTestStore()
	tracker := NewEvolutionTracker(ts, "https://regula.dev/regulations/GDPR#")

	evolution, err := tracker.GetEvolution("https://regula.dev/regulations/GDPR#Art17")
	if err != nil {
		t.Fatalf("GetEvolution failed: %v", err)
	}

	if evolution.ProvisionURI != "https://regula.dev/regulations/GDPR#Art17" {
		t.Errorf("Expected provision URI, got %s", evolution.ProvisionURI)
	}

	if evolution.ProvisionLabel != "Article 17" {
		t.Errorf("Expected label 'Article 17', got %s", evolution.ProvisionLabel)
	}

	if evolution.TotalVersions < 3 {
		t.Errorf("Expected at least 3 versions, got %d", evolution.TotalVersions)
	}

	t.Logf("Evolution: %d versions, %d amendments", evolution.TotalVersions, evolution.AmendmentCount)
	for _, v := range evolution.Versions {
		t.Logf("  %s: %s at %s by %s", v.Version, v.EventType, v.MeetingName, v.ProposerName)
	}
}

func TestEvolutionTracker_GetEvolution_CurrentVersion(t *testing.T) {
	ts := buildEvolutionTestStore()
	tracker := NewEvolutionTracker(ts, "https://regula.dev/regulations/GDPR#")

	evolution, err := tracker.GetEvolution("https://regula.dev/regulations/GDPR#Art17")
	if err != nil {
		t.Fatalf("GetEvolution failed: %v", err)
	}

	if evolution.CurrentVersionURI != "https://regula.dev/regulations/GDPR#Art17:v3" {
		t.Errorf("Expected current version v3, got %s", evolution.CurrentVersionURI)
	}
}

func TestEvolutionTracker_GetEvolution_Timeline(t *testing.T) {
	ts := buildEvolutionTestStore()
	tracker := NewEvolutionTracker(ts, "https://regula.dev/regulations/GDPR#")

	evolution, err := tracker.GetEvolution("https://regula.dev/regulations/GDPR#Art17")
	if err != nil {
		t.Fatalf("GetEvolution failed: %v", err)
	}

	if len(evolution.Timeline) < 3 {
		t.Errorf("Expected at least 3 timeline entries, got %d", len(evolution.Timeline))
	}

	// Check timeline is in chronological order
	for i := 1; i < len(evolution.Timeline); i++ {
		if evolution.Timeline[i].Date.Before(evolution.Timeline[i-1].Date) {
			t.Errorf("Timeline not in chronological order at index %d", i)
		}
	}

	t.Logf("Timeline entries:")
	for _, entry := range evolution.Timeline {
		t.Logf("  %s: %s - %s", entry.Date.Format("2006-01-02"), entry.Label, entry.Description)
	}
}

func TestEvolutionTracker_GetEvolution_VoteRecord(t *testing.T) {
	ts := buildEvolutionTestStore()
	tracker := NewEvolutionTracker(ts, "https://regula.dev/regulations/GDPR#")

	evolution, err := tracker.GetEvolution("https://regula.dev/regulations/GDPR#Art17")
	if err != nil {
		t.Fatalf("GetEvolution failed: %v", err)
	}

	// Find the adopted version
	var adoptedVersion *VersionEvent
	for i, v := range evolution.Versions {
		if v.EventType == EventAdopted {
			adoptedVersion = &evolution.Versions[i]
			break
		}
	}

	if adoptedVersion == nil {
		t.Fatal("Expected to find adopted version")
	}

	if adoptedVersion.Vote == nil {
		t.Fatal("Expected vote record on adopted version")
	}

	if adoptedVersion.Vote.ForCount != 24 {
		t.Errorf("Expected 24 votes for, got %d", adoptedVersion.Vote.ForCount)
	}
	if adoptedVersion.Vote.AgainstCount != 3 {
		t.Errorf("Expected 3 votes against, got %d", adoptedVersion.Vote.AgainstCount)
	}
	if adoptedVersion.Vote.AbstainCount != 1 {
		t.Errorf("Expected 1 abstention, got %d", adoptedVersion.Vote.AbstainCount)
	}
}

func TestEvolutionTracker_RecordProposal(t *testing.T) {
	ts := store.NewTripleStore()
	tracker := NewEvolutionTracker(ts, "https://example.org/")

	provisionURI := "https://example.org/Art1"
	meetingURI := "https://example.org/meetings/m1"
	proposerURI := "https://example.org/delegations/DE"
	text := "Proposed text for Article 1"

	err := tracker.RecordProposal(provisionURI, meetingURI, proposerURI, text)
	if err != nil {
		t.Fatalf("RecordProposal failed: %v", err)
	}

	// Verify version was created
	versionTriples := ts.Find("", store.PropVersionOf, provisionURI)
	if len(versionTriples) != 1 {
		t.Errorf("Expected 1 version, got %d", len(versionTriples))
	}

	// Verify current version is set
	currentTriples := ts.Find(provisionURI, store.PropCurrentVersion, "")
	if len(currentTriples) != 1 {
		t.Errorf("Expected current version to be set")
	}

	// Verify text is stored
	versionURI := versionTriples[0].Subject
	textTriples := ts.Find(versionURI, store.PropText, "")
	if len(textTriples) != 1 || textTriples[0].Object != text {
		t.Errorf("Expected text to be stored")
	}
}

func TestEvolutionTracker_RecordAmendment(t *testing.T) {
	ts := store.NewTripleStore()
	tracker := NewEvolutionTracker(ts, "https://example.org/")

	provisionURI := "https://example.org/Art1"
	meetingURI := "https://example.org/meetings/m1"
	proposerURI := "https://example.org/delegations/DE"

	// First record a proposal
	err := tracker.RecordProposal(provisionURI, meetingURI, proposerURI, "Original text")
	if err != nil {
		t.Fatalf("RecordProposal failed: %v", err)
	}

	// Then record an amendment
	amendMeetingURI := "https://example.org/meetings/m2"
	amendProposerURI := "https://example.org/delegations/FR"
	amendText := "Amended text"

	err = tracker.RecordAmendment(provisionURI, amendMeetingURI, amendProposerURI, amendText)
	if err != nil {
		t.Fatalf("RecordAmendment failed: %v", err)
	}

	// Verify two versions exist
	versionTriples := ts.Find("", store.PropVersionOf, provisionURI)
	if len(versionTriples) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(versionTriples))
	}

	// Verify current version is updated
	currentTriples := ts.Find(provisionURI, store.PropCurrentVersion, "")
	if len(currentTriples) != 1 {
		t.Fatalf("Expected current version to be set")
	}

	// Verify the amendment has previous version link
	v2 := currentTriples[0].Object
	prevTriples := ts.Find(v2, store.PropPreviousVersion, "")
	if len(prevTriples) != 1 {
		t.Errorf("Expected previous version link")
	}
}

func TestEvolutionTracker_RecordAdoption(t *testing.T) {
	ts := store.NewTripleStore()
	tracker := NewEvolutionTracker(ts, "https://example.org/")

	provisionURI := "https://example.org/Art1"
	meetingURI := "https://example.org/meetings/m1"

	// First record a proposal
	err := tracker.RecordProposal(provisionURI, meetingURI, "", "Original text")
	if err != nil {
		t.Fatalf("RecordProposal failed: %v", err)
	}

	// Then record adoption with vote
	vote := &VoteRecord{
		ForCount:     20,
		AgainstCount: 5,
		AbstainCount: 2,
		Result:       "adopted",
	}

	adoptionMeetingURI := "https://example.org/meetings/m2"
	err = tracker.RecordAdoption(provisionURI, adoptionMeetingURI, vote)
	if err != nil {
		t.Fatalf("RecordAdoption failed: %v", err)
	}

	// Verify adopted type
	currentTriples := ts.Find(provisionURI, store.PropCurrentVersion, "")
	if len(currentTriples) != 1 {
		t.Fatalf("Expected current version")
	}

	currentVersion := currentTriples[0].Object
	typeTriples := ts.Find(currentVersion, store.RDFType, "reg:AdoptedText")
	if len(typeTriples) == 0 {
		t.Error("Expected AdoptedText type")
	}

	// Verify vote record exists
	voteTriples := ts.Find(currentVersion, "reg:hasVote", "")
	if len(voteTriples) != 1 {
		t.Error("Expected vote record")
	}
}

func TestEvolutionTracker_CompareVersions(t *testing.T) {
	ts := buildEvolutionTestStore()
	tracker := NewEvolutionTracker(ts, "https://regula.dev/regulations/GDPR#")

	v1 := "https://regula.dev/regulations/GDPR#Art17:v1"
	v2 := "https://regula.dev/regulations/GDPR#Art17:v2"

	diff, err := tracker.CompareVersions(v1, v2)
	if err != nil {
		t.Fatalf("CompareVersions failed: %v", err)
	}

	if diff.Text1 == "" {
		t.Error("Expected text1 to be populated")
	}
	if diff.Text2 == "" {
		t.Error("Expected text2 to be populated")
	}
	if diff.Text1 == diff.Text2 {
		t.Error("Expected texts to be different")
	}

	if len(diff.Changes) == 0 {
		t.Error("Expected changes to be detected")
	}

	t.Logf("Diff: %s -> %s", diff.Text1[:20]+"...", diff.Text2[:20]+"...")
	for _, change := range diff.Changes {
		t.Logf("  Change type: %s", change.Type)
	}
}

func TestEvolutionTracker_QueryAmendments(t *testing.T) {
	ts := buildEvolutionTestStore()
	tracker := NewEvolutionTracker(ts, "https://regula.dev/regulations/GDPR#")

	// Query all amendments
	results, err := tracker.QueryAmendments(nil)
	if err != nil {
		t.Fatalf("QueryAmendments failed: %v", err)
	}

	if len(results) < 2 {
		t.Errorf("Expected at least 2 amendments, got %d", len(results))
	}

	t.Logf("Found %d amendments:", len(results))
	for _, r := range results {
		t.Logf("  %s by %s at %s", r.ProvisionLabel, r.ProposerName, r.MeetingName)
	}
}

func TestEvolutionTracker_QueryAmendments_ByProposer(t *testing.T) {
	ts := buildEvolutionTestStore()
	tracker := NewEvolutionTracker(ts, "https://regula.dev/regulations/GDPR#")

	// Query amendments by France
	query := &AmendmentQuery{
		ProposerURI: "https://example.org/delegations/FR",
	}
	results, err := tracker.QueryAmendments(query)
	if err != nil {
		t.Fatalf("QueryAmendments failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 amendment by France, got %d", len(results))
	}

	if len(results) > 0 && results[0].ProposerName != "France" {
		t.Errorf("Expected France, got %s", results[0].ProposerName)
	}
}

func TestEvolutionTracker_QueryAmendments_ByChapter(t *testing.T) {
	ts := buildEvolutionTestStore()
	tracker := NewEvolutionTracker(ts, "https://regula.dev/regulations/GDPR#")

	// Query amendments in Chapter III
	query := &AmendmentQuery{
		ChapterURI: "https://regula.dev/regulations/GDPR#ChapterIII",
	}
	results, err := tracker.QueryAmendments(query)
	if err != nil {
		t.Fatalf("QueryAmendments failed: %v", err)
	}

	// Both Art17 and Art18 are in Chapter III
	if len(results) < 2 {
		t.Errorf("Expected at least 2 amendments in Chapter III, got %d", len(results))
	}

	t.Logf("Chapter III amendments: %d", len(results))
	for _, r := range results {
		t.Logf("  %s", r.ProvisionLabel)
	}
}

func TestEvolutionTracker_QueryAmendments_ByDateRange(t *testing.T) {
	ts := buildEvolutionTestStore()
	tracker := NewEvolutionTracker(ts, "https://regula.dev/regulations/GDPR#")

	// Query amendments in February 2024
	fromDate := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2024, 2, 28, 23, 59, 59, 0, time.UTC)

	query := &AmendmentQuery{
		FromDate: &fromDate,
		ToDate:   &toDate,
	}
	results, err := tracker.QueryAmendments(query)
	if err != nil {
		t.Fatalf("QueryAmendments failed: %v", err)
	}

	// Art17:v2 (Feb 20) and Art18:v1 (Feb 1) should match
	if len(results) != 2 {
		t.Errorf("Expected 2 amendments in February, got %d", len(results))
	}
}

func TestEvolutionTracker_GetChapterAmendments(t *testing.T) {
	ts := buildEvolutionTestStore()
	tracker := NewEvolutionTracker(ts, "https://regula.dev/regulations/GDPR#")

	results, err := tracker.GetChapterAmendments("https://regula.dev/regulations/GDPR#ChapterIII")
	if err != nil {
		t.Fatalf("GetChapterAmendments failed: %v", err)
	}

	if len(results) < 2 {
		t.Errorf("Expected at least 2 amendments, got %d", len(results))
	}
}

func TestEvolutionTracker_GetProposerAmendments(t *testing.T) {
	ts := buildEvolutionTestStore()
	tracker := NewEvolutionTracker(ts, "https://regula.dev/regulations/GDPR#")

	results, err := tracker.GetProposerAmendments("https://example.org/delegations/FR")
	if err != nil {
		t.Fatalf("GetProposerAmendments failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 amendment, got %d", len(results))
	}
}

func TestEvolutionTracker_GenerateTimelineData(t *testing.T) {
	ts := buildEvolutionTestStore()
	tracker := NewEvolutionTracker(ts, "https://regula.dev/regulations/GDPR#")

	timeline, err := tracker.GenerateTimelineData("https://regula.dev/regulations/GDPR#Art17")
	if err != nil {
		t.Fatalf("GenerateTimelineData failed: %v", err)
	}

	if len(timeline) < 3 {
		t.Errorf("Expected at least 3 timeline entries, got %d", len(timeline))
	}

	// Verify timeline can be used for visualization
	for _, entry := range timeline {
		if entry.Date.IsZero() {
			t.Error("Timeline entry has zero date")
		}
		if entry.Description == "" {
			t.Error("Timeline entry has empty description")
		}
	}
}

func TestEvolutionTracker_GetCurrentText(t *testing.T) {
	ts := buildEvolutionTestStore()
	tracker := NewEvolutionTracker(ts, "https://regula.dev/regulations/GDPR#")

	text, err := tracker.GetCurrentText("https://regula.dev/regulations/GDPR#Art17")
	if err != nil {
		t.Fatalf("GetCurrentText failed: %v", err)
	}

	if text == "" {
		t.Error("Expected non-empty text")
	}

	expectedPrefix := "Final adopted text"
	if len(text) < len(expectedPrefix) || text[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Expected text to start with '%s', got '%s'", expectedPrefix, text)
	}
}

func TestEvolutionTracker_GetVersionText(t *testing.T) {
	ts := buildEvolutionTestStore()
	tracker := NewEvolutionTracker(ts, "https://regula.dev/regulations/GDPR#")

	// Get v1 text
	text, err := tracker.GetVersionText("https://regula.dev/regulations/GDPR#Art17:v1")
	if err != nil {
		t.Fatalf("GetVersionText failed: %v", err)
	}

	expectedPrefix := "Original proposed"
	if len(text) < len(expectedPrefix) || text[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Expected text to start with '%s', got '%s'", expectedPrefix, text)
	}
}

func TestEvolutionTracker_NilStore(t *testing.T) {
	tracker := NewEvolutionTracker(nil, "https://example.org/")

	_, err := tracker.GetEvolution("https://example.org/Art1")
	if err == nil {
		t.Error("Expected error for nil store")
	}

	err = tracker.RecordProposal("uri", "meeting", "", "")
	if err == nil {
		t.Error("Expected error for nil store")
	}

	err = tracker.RecordAmendment("uri", "meeting", "", "")
	if err == nil {
		t.Error("Expected error for nil store")
	}

	err = tracker.RecordAdoption("uri", "meeting", nil)
	if err == nil {
		t.Error("Expected error for nil store")
	}

	_, err = tracker.CompareVersions("v1", "v2")
	if err == nil {
		t.Error("Expected error for nil store")
	}

	_, err = tracker.QueryAmendments(nil)
	if err == nil {
		t.Error("Expected error for nil store")
	}
}

func TestEvolutionTracker_EmptyProvisionURI(t *testing.T) {
	ts := store.NewTripleStore()
	tracker := NewEvolutionTracker(ts, "https://example.org/")

	_, err := tracker.GetEvolution("")
	if err == nil {
		t.Error("Expected error for empty provision URI")
	}

	err = tracker.RecordProposal("", "meeting", "", "")
	if err == nil {
		t.Error("Expected error for empty provision URI")
	}

	err = tracker.RecordAmendment("", "meeting", "", "")
	if err == nil {
		t.Error("Expected error for empty provision URI")
	}

	err = tracker.RecordAdoption("", "meeting", nil)
	if err == nil {
		t.Error("Expected error for empty provision URI")
	}
}

func TestEvolutionTracker_EmptyMeetingURI(t *testing.T) {
	ts := store.NewTripleStore()
	tracker := NewEvolutionTracker(ts, "https://example.org/")

	err := tracker.RecordProposal("provision", "", "", "")
	if err == nil {
		t.Error("Expected error for empty meeting URI")
	}

	err = tracker.RecordAmendment("provision", "", "", "")
	if err == nil {
		t.Error("Expected error for empty meeting URI")
	}

	err = tracker.RecordAdoption("provision", "", nil)
	if err == nil {
		t.Error("Expected error for empty meeting URI")
	}
}

func TestEventType_String(t *testing.T) {
	tests := []struct {
		eventType EventType
		expected  string
	}{
		{EventProposed, "proposed"},
		{EventAmended, "amended"},
		{EventAdopted, "adopted"},
		{EventRejected, "rejected"},
		{EventWithdrawn, "withdrawn"},
		{EventSuperseded, "superseded"},
		{EventRepealed, "repealed"},
		{EventType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.eventType.String(); got != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestParseEventType(t *testing.T) {
	tests := []struct {
		input    string
		expected EventType
	}{
		{"proposed", EventProposed},
		{"PROPOSED", EventProposed},
		{"amended", EventAmended},
		{"adopted", EventAdopted},
		{"rejected", EventRejected},
		{"withdrawn", EventWithdrawn},
		{"superseded", EventSuperseded},
		{"repealed", EventRepealed},
		{"unknown", EventProposed}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ParseEventType(tt.input); got != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestEvolutionTracker_RecordRejection(t *testing.T) {
	ts := store.NewTripleStore()
	tracker := NewEvolutionTracker(ts, "https://example.org/")

	provisionURI := "https://example.org/Art1"
	meetingURI := "https://example.org/meetings/m1"

	// First record a proposal
	err := tracker.RecordProposal(provisionURI, meetingURI, "", "Proposed text")
	if err != nil {
		t.Fatalf("RecordProposal failed: %v", err)
	}

	// Then record rejection
	vote := &VoteRecord{
		ForCount:     5,
		AgainstCount: 20,
		AbstainCount: 3,
		Result:       "rejected",
	}

	rejectionMeetingURI := "https://example.org/meetings/m2"
	err = tracker.RecordRejection(provisionURI, rejectionMeetingURI, vote)
	if err != nil {
		t.Fatalf("RecordRejection failed: %v", err)
	}

	// Verify rejection was recorded
	currentTriples := ts.Find(provisionURI, store.PropCurrentVersion, "")
	if len(currentTriples) != 1 {
		t.Fatalf("Expected current version")
	}

	currentVersion := currentTriples[0].Object
	eventTypeTriples := ts.Find(currentVersion, "reg:eventType", "rejected")
	if len(eventTypeTriples) == 0 {
		t.Error("Expected rejected event type")
	}
}
