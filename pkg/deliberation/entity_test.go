package deliberation

import (
	"strings"
	"testing"

	"github.com/coolbeans/regula/pkg/store"
)

// buildEntityTriples creates test data for entity extraction testing.
func buildEntityTriples() *store.TripleStore {
	ts := store.NewTripleStore()

	// Known stakeholders
	germany := "reg:stakeholder:germany"
	ts.Add(germany, "rdf:type", store.ClassStakeholder)
	ts.Add(germany, store.RDFSLabel, "Germany")
	ts.Add(germany, store.PropStakeholderType, "member_state")
	ts.Add(germany, store.PropStakeholderAlias, "Federal Republic of Germany")
	ts.Add(germany, store.PropStakeholderAlias, "German delegation")

	france := "reg:stakeholder:france"
	ts.Add(france, "rdf:type", store.ClassStakeholder)
	ts.Add(france, store.RDFSLabel, "France")
	ts.Add(france, store.PropStakeholderType, "member_state")
	ts.Add(france, store.PropStakeholderAlias, "French Republic")

	commission := "reg:stakeholder:european-commission"
	ts.Add(commission, "rdf:type", store.ClassStakeholder)
	ts.Add(commission, store.RDFSLabel, "European Commission")
	ts.Add(commission, store.PropStakeholderType, "organization")
	ts.Add(commission, store.PropStakeholderAlias, "the Commission")

	// Known speakers
	smithURI := "reg:speaker:john-smith"
	ts.Add(smithURI, "rdf:type", store.ClassStakeholder)
	ts.Add(smithURI, store.RDFSLabel, "John Smith")
	ts.Add(smithURI, store.PropStakeholderType, "individual")
	ts.Add(smithURI, store.PropMemberOf, germany)

	// Meeting with participants
	meeting1 := "reg:meeting1"
	ts.Add(meeting1, "rdf:type", store.ClassMeeting)
	ts.Add(meeting1, store.RDFSLabel, "Meeting 1")
	ts.Add(meeting1, store.PropParticipant, germany)
	ts.Add(meeting1, store.PropParticipant, france)
	ts.Add(meeting1, store.PropParticipant, commission)

	// Speaker intervention
	ts.Add(meeting1, store.PropSpeaker, smithURI)

	return ts
}

func TestNewEntityExtractor(t *testing.T) {
	ts := store.NewTripleStore()
	extractor := NewEntityExtractor(ts, "https://example.org/")

	if extractor == nil {
		t.Fatal("expected non-nil EntityExtractor")
	}
	if extractor.store != ts {
		t.Error("expected store to be set")
	}
	if extractor.patterns == nil {
		t.Error("expected patterns to be compiled")
	}
	if extractor.resolver == nil {
		t.Error("expected resolver to be created")
	}
}

func TestExtractEntities_NilStore(t *testing.T) {
	extractor := &EntityExtractor{store: nil, patterns: compileEntityPatterns()}
	_, err := extractor.ExtractEntities("test text", EntityContext{})
	if err == nil {
		t.Error("expected error for nil store")
	}
}

func TestExtractSpeakersWithAffiliation(t *testing.T) {
	ts := buildEntityTriples()
	extractor := NewEntityExtractor(ts, "reg:")

	text := `The meeting began with opening remarks.
Mr. John Smith (Germany) stated that the proposal was acceptable.
Ms. Marie Dupont (France) expressed support for the amendment.
Dr. Hans Mueller (German delegation) raised concerns about timing.`

	result, err := extractor.ExtractEntities(text, EntityContext{MeetingURI: "reg:meeting1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should extract speakers
	if len(result.Speakers) < 2 {
		t.Errorf("expected at least 2 speakers, got %d", len(result.Speakers))
	}

	// Check for John Smith
	foundSmith := false
	for _, s := range result.Speakers {
		if strings.Contains(s.Name, "John Smith") {
			foundSmith = true
			if s.AffiliationName != "Germany" {
				t.Errorf("expected affiliation 'Germany', got %q", s.AffiliationName)
			}
			break
		}
	}
	if !foundSmith {
		t.Error("expected to find John Smith in speakers")
	}
}

func TestExtractRoleSpeakers(t *testing.T) {
	ts := buildEntityTriples()
	extractor := NewEntityExtractor(ts, "reg:")

	text := `The Chair noted that a quorum was present.
The Rapporteur presented the draft report.
The Secretary-General announced the next item.
The President proposed to adjourn.`

	result, err := extractor.ExtractEntities(text, EntityContext{MeetingURI: "reg:meeting1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should extract role-based mentions
	foundChair := false
	foundRapporteur := false
	for _, m := range result.Mentions {
		if m.ProbableType == "role" {
			if strings.Contains(m.NormalizedText, "chair") {
				foundChair = true
			}
			if strings.Contains(m.NormalizedText, "rapporteur") {
				foundRapporteur = true
			}
		}
	}

	if !foundChair {
		t.Error("expected to find Chair role")
	}
	if !foundRapporteur {
		t.Error("expected to find Rapporteur role")
	}
}

func TestExtractMemberStates(t *testing.T) {
	ts := buildEntityTriples()
	extractor := NewEntityExtractor(ts, "reg:")

	text := `The representative of Germany stated that further discussion was needed.
The delegate from France proposed an amendment.
Italy's representative expressed support.
Submitted by: Spain and Portugal`

	result, err := extractor.ExtractEntities(text, EntityContext{MeetingURI: "reg:meeting1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should extract member state stakeholders
	foundGermany := false
	foundFrance := false
	for _, s := range result.Stakeholders {
		if s.Type == StakeholderMemberState {
			if strings.Contains(s.Name, "Germany") {
				foundGermany = true
			}
			if strings.Contains(s.Name, "France") {
				foundFrance = true
			}
		}
	}

	if !foundGermany {
		t.Error("expected to find Germany as member state")
	}
	if !foundFrance {
		t.Error("expected to find France as member state")
	}
}

func TestExtractDelegations(t *testing.T) {
	ts := buildEntityTriples()
	extractor := NewEntityExtractor(ts, "reg:")

	text := `The German delegation stated its position.
The French delegation proposed to defer the vote.
Delegation of Italy requested the floor.`

	result, err := extractor.ExtractEntities(text, EntityContext{MeetingURI: "reg:meeting1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should extract delegations
	foundDelegation := false
	for _, s := range result.Stakeholders {
		if s.Type == StakeholderDelegation || strings.Contains(s.Name, "German") {
			foundDelegation = true
			break
		}
	}

	if !foundDelegation {
		t.Error("expected to find at least one delegation")
	}
}

func TestExtractOrganizations(t *testing.T) {
	ts := buildEntityTriples()
	extractor := NewEntityExtractor(ts, "reg:")

	text := `The European Commission presented its proposal.
The Council Secretariat circulated the document.
Representative of the World Trade Organization attended as observer.`

	result, err := extractor.ExtractEntities(text, EntityContext{MeetingURI: "reg:meeting1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should extract organizations
	foundOrg := false
	for _, s := range result.Stakeholders {
		if s.Type == StakeholderOrganization || s.Type == StakeholderSecretariat {
			foundOrg = true
			break
		}
	}

	if !foundOrg {
		t.Error("expected to find at least one organization")
	}
}

func TestExtractFromVotingRecords(t *testing.T) {
	ts := buildEntityTriples()
	extractor := NewEntityExtractor(ts, "reg:")

	text := `The vote was taken with the following result:
Germany: For
France: For
Italy: Against
Spain: Abstain
For: Poland, Netherlands, Belgium
Against: Portugal, Greece`

	result, err := extractor.ExtractEntities(text, EntityContext{MeetingURI: "reg:meeting1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should extract voting stakeholders
	memberStateCount := 0
	for _, s := range result.Stakeholders {
		if s.Type == StakeholderMemberState {
			memberStateCount++
		}
	}

	if memberStateCount < 3 {
		t.Errorf("expected at least 3 member states from voting records, got %d", memberStateCount)
	}
}

func TestExtractDocumentAuthors(t *testing.T) {
	ts := buildEntityTriples()
	extractor := NewEntityExtractor(ts, "reg:")

	text := `DOCUMENT A/123/Rev.1
Submitted by: Germany, France
Co-sponsored by: Italy, Spain
Author: Dr. Hans Mueller`

	result, err := extractor.ExtractEntities(text, EntityContext{MeetingURI: "reg:meeting1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should extract authors
	if len(result.Mentions) == 0 {
		t.Error("expected to extract author mentions")
	}

	// Should have found Germany as author
	foundGermanyAuthor := false
	for _, m := range result.Mentions {
		if strings.Contains(m.NormalizedText, "Germany") {
			foundGermanyAuthor = true
			break
		}
	}
	if !foundGermanyAuthor {
		t.Error("expected to find Germany as document author")
	}
}

func TestEntityResolver(t *testing.T) {
	ts := buildEntityTriples()
	resolver := NewEntityResolver(ts, "reg:")

	// Test exact match
	entity, confidence := resolver.Resolve("Germany", EntityContext{})
	if entity == nil {
		t.Fatal("expected to resolve 'Germany'")
	}
	if confidence < 1.0 {
		t.Errorf("expected confidence 1.0 for exact match, got %f", confidence)
	}

	// Test alias match
	entity, confidence = resolver.Resolve("Federal Republic of Germany", EntityContext{})
	if entity == nil {
		t.Fatal("expected to resolve 'Federal Republic of Germany'")
	}
	if confidence < 1.0 {
		t.Errorf("expected confidence 1.0 for alias match, got %f", confidence)
	}

	// Test fuzzy match
	entity, confidence = resolver.Resolve("German", EntityContext{})
	if entity != nil && confidence < 0.5 {
		// Fuzzy match may or may not work depending on threshold
		t.Logf("fuzzy match found with confidence %f", confidence)
	}
}

func TestEntityResolverAddAlias(t *testing.T) {
	ts := buildEntityTriples()
	resolver := NewEntityResolver(ts, "reg:")

	// Add a new alias
	resolver.AddAlias("reg:stakeholder:germany", "DE")

	// Should now resolve via new alias
	entity, confidence := resolver.Resolve("DE", EntityContext{})
	if entity == nil {
		t.Fatal("expected to resolve 'DE' after adding alias")
	}
	if confidence < 1.0 {
		t.Errorf("expected confidence 1.0 for alias match, got %f", confidence)
	}
}

func TestEntityResolverAddEntity(t *testing.T) {
	ts := buildEntityTriples()
	resolver := NewEntityResolver(ts, "reg:")

	// Add a new entity
	newEntity := &ResolvedEntity{
		URI:     "reg:stakeholder:poland",
		Name:    "Poland",
		Type:    "member_state",
		Aliases: []string{"Republic of Poland", "Polish delegation"},
	}
	resolver.AddEntity(newEntity)

	// Should resolve by name
	entity, _ := resolver.Resolve("Poland", EntityContext{})
	if entity == nil {
		t.Fatal("expected to resolve 'Poland' after adding entity")
	}

	// Should resolve by alias
	entity, _ = resolver.Resolve("Republic of Poland", EntityContext{})
	if entity == nil {
		t.Fatal("expected to resolve 'Republic of Poland' after adding entity")
	}
}

func TestStakeholderTypeString(t *testing.T) {
	tests := []struct {
		sType StakeholderType
		want  string
	}{
		{StakeholderMemberState, "member_state"},
		{StakeholderDelegation, "delegation"},
		{StakeholderOrganization, "organization"},
		{StakeholderPoliticalGroup, "political_group"},
		{StakeholderCommittee, "committee"},
		{StakeholderSecretariat, "secretariat"},
		{StakeholderObserver, "observer"},
		{StakeholderIndividual, "individual"},
		{StakeholderType(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.sType.String()
		if got != tt.want {
			t.Errorf("StakeholderType(%d).String() = %q, want %q", tt.sType, got, tt.want)
		}
	}
}

func TestParseStakeholderType(t *testing.T) {
	tests := []struct {
		input string
		want  StakeholderType
	}{
		{"member_state", StakeholderMemberState},
		{"memberstate", StakeholderMemberState},
		{"member state", StakeholderMemberState},
		{"delegation", StakeholderDelegation},
		{"organization", StakeholderOrganization},
		{"org", StakeholderOrganization},
		{"political_group", StakeholderPoliticalGroup},
		{"committee", StakeholderCommittee},
		{"secretariat", StakeholderSecretariat},
		{"observer", StakeholderObserver},
		{"individual", StakeholderIndividual},
		{"person", StakeholderIndividual},
		{"unknown", StakeholderOrganization}, // default
	}

	for _, tt := range tests {
		got := ParseStakeholderType(tt.input)
		if got != tt.want {
			t.Errorf("ParseStakeholderType(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Mr. John Smith", "John Smith"},
		{"Ms. Marie Dupont", "Marie Dupont"},
		{"Dr. Hans Mueller", "Hans Mueller"},
		{"Prof. Anna Schmidt", "Anna Schmidt"},
		{"The Chair", "Chair"},
		{"  Germany  ", "Germany"},
		{"Federal   Republic   of   Germany", "Federal Republic of Germany"},
	}

	for _, tt := range tests {
		got := normalizeName(tt.input)
		if got != tt.want {
			t.Errorf("normalizeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeRole(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Chair", "chair"},
		{"Vice-President", "vicepresident"},
		{"Secretary General", "secretarygeneral"},
		{"  Rapporteur  ", "rapporteur"},
	}

	for _, tt := range tests {
		got := normalizeRole(tt.input)
		if got != tt.want {
			t.Errorf("normalizeRole(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSimilarityScore(t *testing.T) {
	tests := []struct {
		a       string
		b       string
		minWant float64
	}{
		{"Germany", "Germany", 1.0},
		{"Federal Republic of Germany", "Germany", 0.2},
		{"France", "Italy", 0.0},
		{"European Commission", "the Commission", 0.3},
	}

	for _, tt := range tests {
		got := similarityScore(tt.a, tt.b)
		if got < tt.minWant {
			t.Errorf("similarityScore(%q, %q) = %f, want >= %f", tt.a, tt.b, got, tt.minWant)
		}
	}
}

func TestIsValidStateName(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Germany", true},
		{"France", true},
		{"United Kingdom", true},
		{"the", false},
		{"for", false},
		{"against", false},
		{"A", false},
		{"", false},
	}

	for _, tt := range tests {
		got := isValidStateName(tt.input)
		if got != tt.want {
			t.Errorf("isValidStateName(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsLikelyPersonName(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Mr. John Smith", true},
		{"John Smith", true},
		{"European Commission", false},
		{"German delegation", false},
		{"Council Secretariat", false},
		{"", false},
		{"A", false},
	}

	for _, tt := range tests {
		got := isLikelyPersonName(tt.input)
		if got != tt.want {
			t.Errorf("isLikelyPersonName(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsLikelyCountryName(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Germany", true},
		{"France", true},
		{"Poland", true},
		{"Finland", true},
		{"Romania", true},
		{"European Commission", false},
		{"Chair", false},
	}

	for _, tt := range tests {
		got := isLikelyCountryName(tt.input)
		if got != tt.want {
			t.Errorf("isLikelyCountryName(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsVotePosition(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"For", true},
		{"for", true},
		{"Against", true},
		{"Abstain", true},
		{"In favour", true},
		{"Germany", false},
		{"Chair", false},
	}

	for _, tt := range tests {
		got := isVotePosition(tt.input)
		if got != tt.want {
			t.Errorf("isVotePosition(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestInferStakeholderType(t *testing.T) {
	ts := store.NewTripleStore()
	extractor := NewEntityExtractor(ts, "reg:")

	tests := []struct {
		input string
		want  StakeholderType
	}{
		{"European Commission", StakeholderOrganization},
		{"Council of the EU", StakeholderOrganization},
		{"European Parliament", StakeholderOrganization},
		{"Budget Committee", StakeholderCommittee},
		{"Council Secretariat", StakeholderSecretariat},
		{"German delegation", StakeholderDelegation},
		{"Socialist Group", StakeholderPoliticalGroup},
		{"UNICEF Observer", StakeholderObserver},
		{"Germany", StakeholderMemberState},
		{"France", StakeholderMemberState},
	}

	for _, tt := range tests {
		got := extractor.inferStakeholderType(tt.input)
		if got != tt.want {
			t.Errorf("inferStakeholderType(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestPersistEntities(t *testing.T) {
	ts := store.NewTripleStore()
	extractor := NewEntityExtractor(ts, "reg:")

	result := &ExtractionResult{
		Speakers: []Speaker{
			{
				URI:             "reg:speaker:john-doe",
				Name:            "John Doe",
				Affiliation:     "reg:stakeholder:germany",
				AffiliationName: "Germany",
				Aliases:         []string{"Mr. Doe"},
				Roles: []RoleAssignment{
					{Role: "Rapporteur", Scope: "Working Group A"},
				},
			},
		},
		Stakeholders: []ExtractedStakeholder{
			{
				URI:     "reg:stakeholder:spain",
				Name:    "Spain",
				Type:    StakeholderMemberState,
				Aliases: []string{"Kingdom of Spain"},
			},
		},
	}

	err := extractor.PersistEntities(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify speaker was persisted
	typeTriples := ts.Find("reg:speaker:john-doe", "rdf:type", "")
	if len(typeTriples) == 0 {
		t.Error("expected speaker to be persisted with type")
	}

	labelTriples := ts.Find("reg:speaker:john-doe", store.RDFSLabel, "")
	if len(labelTriples) == 0 || labelTriples[0].Object != "John Doe" {
		t.Error("expected speaker label to be persisted")
	}

	// Verify stakeholder was persisted
	typeTriples = ts.Find("reg:stakeholder:spain", "rdf:type", "")
	if len(typeTriples) == 0 {
		t.Error("expected stakeholder to be persisted with type")
	}
}

func TestGetUnresolvedMentions(t *testing.T) {
	ts := store.NewTripleStore()
	extractor := NewEntityExtractor(ts, "reg:")

	result := &ExtractionResult{
		Mentions: []EntityMention{
			{Text: "Germany", ResolvedURI: "reg:stakeholder:germany", Confidence: 1.0},
			{Text: "Unknown Country", ResolvedURI: "", Confidence: 0.0},
			{Text: "France", ResolvedURI: "reg:stakeholder:france", Confidence: 1.0},
			{Text: "Another Unknown", ResolvedURI: "", Confidence: 0.0},
		},
	}

	unresolved := extractor.GetUnresolvedMentions(result)
	if len(unresolved) != 2 {
		t.Errorf("expected 2 unresolved mentions, got %d", len(unresolved))
	}

	for _, m := range unresolved {
		if m.ResolvedURI != "" {
			t.Error("unresolved mention should have empty ResolvedURI")
		}
	}
}

func TestRenderEntityGraph(t *testing.T) {
	ts := store.NewTripleStore()
	extractor := NewEntityExtractor(ts, "reg:")

	result := &ExtractionResult{
		Speakers: []Speaker{
			{URI: "reg:speaker:john-smith", Name: "John Smith", Affiliation: "reg:stakeholder:germany"},
			{URI: "reg:speaker:marie-dupont", Name: "Marie Dupont", Affiliation: "reg:stakeholder:france"},
		},
		Stakeholders: []ExtractedStakeholder{
			{URI: "reg:stakeholder:germany", Name: "Germany", Type: StakeholderMemberState},
			{URI: "reg:stakeholder:france", Name: "France", Type: StakeholderMemberState},
		},
	}

	dot := extractor.RenderEntityGraph(result)

	// Check DOT structure
	if !strings.Contains(dot, "digraph Entities") {
		t.Error("expected digraph declaration")
	}
	if !strings.Contains(dot, "cluster_speakers") {
		t.Error("expected speakers cluster")
	}
	if !strings.Contains(dot, "cluster_stakeholders") {
		t.Error("expected stakeholders cluster")
	}
	if !strings.Contains(dot, "John Smith") {
		t.Error("expected John Smith node")
	}
	if !strings.Contains(dot, "affiliation") {
		t.Error("expected affiliation edges")
	}
}

func TestExtractionResultRenderJSON(t *testing.T) {
	result := &ExtractionResult{
		Speakers: []Speaker{
			{URI: "reg:speaker:john-smith", Name: "John Smith"},
		},
		Stakeholders: []ExtractedStakeholder{
			{URI: "reg:stakeholder:germany", Name: "Germany", Type: StakeholderMemberState},
		},
		Mentions:   []EntityMention{},
		Resolved:   1,
		Unresolved: 0,
	}

	jsonOutput, err := result.RenderJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(jsonOutput, "\"speakers\"") {
		t.Error("expected speakers field in JSON")
	}
	if !strings.Contains(jsonOutput, "\"stakeholders\"") {
		t.Error("expected stakeholders field in JSON")
	}
	if !strings.Contains(jsonOutput, "John Smith") {
		t.Error("expected John Smith in JSON output")
	}
}

func TestDeduplication(t *testing.T) {
	ts := store.NewTripleStore()
	extractor := NewEntityExtractor(ts, "reg:")

	// Speakers with duplicates
	speakers := []Speaker{
		{URI: "reg:speaker:john-smith", Name: "John Smith"},
		{URI: "reg:speaker:john-smith", Name: "John Smith"}, // duplicate
		{URI: "reg:speaker:marie-dupont", Name: "Marie Dupont"},
	}

	deduplicated := extractor.deduplicateSpeakers(speakers)
	if len(deduplicated) != 2 {
		t.Errorf("expected 2 unique speakers, got %d", len(deduplicated))
	}

	// Stakeholders with duplicates
	stakeholders := []ExtractedStakeholder{
		{URI: "reg:stakeholder:germany", Name: "Germany"},
		{URI: "reg:stakeholder:germany", Name: "Germany"}, // duplicate
		{URI: "reg:stakeholder:france", Name: "France"},
	}

	deduplicatedStakeholders := extractor.deduplicateStakeholders(stakeholders)
	if len(deduplicatedStakeholders) != 2 {
		t.Errorf("expected 2 unique stakeholders, got %d", len(deduplicatedStakeholders))
	}
}

func TestGenerateURIs(t *testing.T) {
	ts := store.NewTripleStore()
	extractor := NewEntityExtractor(ts, "https://example.org/")

	speakerURI := extractor.generateSpeakerURI("John Smith")
	if !strings.HasPrefix(speakerURI, "https://example.org/speaker:") {
		t.Errorf("speaker URI should have correct prefix, got %s", speakerURI)
	}
	if !strings.Contains(speakerURI, "john-smith") {
		t.Errorf("speaker URI should contain normalized name, got %s", speakerURI)
	}

	stakeholderURI := extractor.generateStakeholderURI("European Commission")
	if !strings.HasPrefix(stakeholderURI, "https://example.org/stakeholder:") {
		t.Errorf("stakeholder URI should have correct prefix, got %s", stakeholderURI)
	}
	if !strings.Contains(stakeholderURI, "european-commission") {
		t.Errorf("stakeholder URI should contain normalized name, got %s", stakeholderURI)
	}
}
