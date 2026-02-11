package unres

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// Sample Akoma Ntoso XML fixture
const sampleAKNResolutionXML = `<?xml version="1.0" encoding="UTF-8"?>
<akomaNtoso xmlns="http://docs.oasis-open.org/legaldocml/ns/akn/3.0">
  <doc name="resolution">
    <meta>
      <identification source="#un">
        <FRBRWork>
          <FRBRthis value="/akn/un/statement/deliberation/unga/2024-12-20/79-100/!main"/>
          <FRBRuri value="/akn/un/statement/deliberation/unga/2024-12-20/79-100"/>
          <FRBRdate date="2024-12-20" name="adoption"/>
          <FRBRauthor href="#unga" as="#author"/>
        </FRBRWork>
        <FRBRExpression>
          <FRBRlanguage language="eng"/>
        </FRBRExpression>
      </identification>
      <references source="#un">
        <TLCOrganization eId="unga" href="/akn/un/ontology/organization/un/generalAssembly"
                         showAs="General Assembly"/>
        <TLCOrganization eId="securityCouncil" href="/akn/un/ontology/organization/un/securityCouncil"
                         showAs="Security Council"/>
        <TLCConcept eId="humanRights" href="/akn/un/ontology/concept/humanRights"
                    showAs="human rights"/>
      </references>
    </meta>
    <preface>
      <p class="docNumber">Resolution 79/100</p>
      <p class="docTitle">Situation of human rights in Country X</p>
      <p class="docProponent">adopted by the General Assembly</p>
    </preface>
    <preamble>
      <container name="recitals">
        <p><i>Guided by</i> the Charter of the United Nations and the Universal Declaration of Human Rights,</p>
        <p><i>Recalling</i> its resolution 78/200 of 20 December 2023,</p>
        <p><i>Expressing grave concern</i> at the deteriorating situation of human rights,</p>
      </container>
    </preamble>
    <body>
      <paragraph eId="para_1">
        <num>1.</num>
        <content>
          <p><i>Condemns</i> the ongoing violations of human rights in Country X;</p>
        </content>
      </paragraph>
      <paragraph eId="para_2">
        <num>2.</num>
        <content>
          <p><i>Calls upon</i> all parties to respect international humanitarian law;</p>
        </content>
      </paragraph>
      <paragraph eId="para_3">
        <num>3.</num>
        <content>
          <p><i>Decides</i> to remain seized of the matter.</p>
        </content>
      </paragraph>
    </body>
    <conclusions>
      <p>79th plenary meeting</p>
      <p>20 December 2024</p>
    </conclusions>
  </doc>
</akomaNtoso>`

// Sample Security Council resolution XML
const sampleSCResolutionXML = `<?xml version="1.0" encoding="UTF-8"?>
<akomaNtoso xmlns="http://docs.oasis-open.org/legaldocml/ns/akn/3.0">
  <doc name="resolution">
    <meta>
      <identification source="#un">
        <FRBRWork>
          <FRBRthis value="/akn/un/statement/deliberation/unsc/2024-03-15/2798/!main"/>
          <FRBRuri value="/akn/un/statement/deliberation/unsc/2024-03-15/2798"/>
          <FRBRdate date="2024-03-15" name="adoption"/>
          <FRBRauthor href="#unsc" as="#author"/>
        </FRBRWork>
        <FRBRExpression>
          <FRBRlanguage language="eng"/>
        </FRBRExpression>
      </identification>
      <references source="#un">
        <TLCOrganization eId="unsc" href="/akn/un/ontology/organization/un/securityCouncil"
                         showAs="Security Council"/>
      </references>
    </meta>
    <preface>
      <p class="docNumber">Resolution 2798 (2024)</p>
      <p class="docTitle">The situation concerning Region Y</p>
      <p class="docProponent">adopted by the Security Council at its 9589th meeting</p>
    </preface>
    <preamble>
      <p><i>Recalling</i> its previous resolutions on the matter,</p>
      <p><i>Reaffirming</i> its commitment to the sovereignty and territorial integrity of all States,</p>
    </preamble>
    <body>
      <paragraph eId="para_1">
        <num>1.</num>
        <content>
          <p><i>Demands</i> an immediate ceasefire;</p>
        </content>
      </paragraph>
      <paragraph eId="para_2">
        <num>2.</num>
        <content>
          <p><i>Authorizes</i> the deployment of peacekeeping forces;</p>
        </content>
      </paragraph>
    </body>
  </doc>
</akomaNtoso>`

func TestUNBodyString(t *testing.T) {
	tests := []struct {
		body UNBody
		want string
	}{
		{BodyGeneralAssembly, "general-assembly"},
		{BodySecurityCouncil, "security-council"},
		{BodyECOSOC, "ecosoc"},
	}

	for _, tt := range tests {
		got := tt.body.String()
		if got != tt.want {
			t.Errorf("(%v).String() = %q, want %q", tt.body, got, tt.want)
		}
	}
}

func TestUNBodyAbbreviation(t *testing.T) {
	tests := []struct {
		body UNBody
		want string
	}{
		{BodyGeneralAssembly, "GA"},
		{BodySecurityCouncil, "SC"},
		{BodyECOSOC, "ECOSOC"},
	}

	for _, tt := range tests {
		got := tt.body.Abbreviation()
		if got != tt.want {
			t.Errorf("(%v).Abbreviation() = %q, want %q", tt.body, got, tt.want)
		}
	}
}

func TestParseUNBody(t *testing.T) {
	tests := []struct {
		input string
		want  UNBody
	}{
		{"ga", BodyGeneralAssembly},
		{"GA", BodyGeneralAssembly},
		{"general-assembly", BodyGeneralAssembly},
		{"unga", BodyGeneralAssembly},
		{"sc", BodySecurityCouncil},
		{"SC", BodySecurityCouncil},
		{"security-council", BodySecurityCouncil},
		{"unsc", BodySecurityCouncil},
		{"ecosoc", BodyECOSOC},
		{"ECOSOC", BodyECOSOC},
		{"unknown", BodyGeneralAssembly}, // default
	}

	for _, tt := range tests {
		got := ParseUNBody(tt.input)
		if got != tt.want {
			t.Errorf("ParseUNBody(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseAKNResolution_GA(t *testing.T) {
	res, err := ParseAKNResolution([]byte(sampleAKNResolutionXML))
	if err != nil {
		t.Fatalf("ParseAKNResolution() error = %v", err)
	}

	// Verify identification
	if res.Body != BodyGeneralAssembly {
		t.Errorf("Body = %v, want %v", res.Body, BodyGeneralAssembly)
	}
	if res.Session != 79 {
		t.Errorf("Session = %d, want 79", res.Session)
	}
	if res.Number != 100 {
		t.Errorf("Number = %d, want 100", res.Number)
	}
	if res.Language != "eng" {
		t.Errorf("Language = %q, want %q", res.Language, "eng")
	}

	// Verify date
	expectedDate := time.Date(2024, 12, 20, 0, 0, 0, 0, time.UTC)
	if !res.AdoptionDate.Equal(expectedDate) {
		t.Errorf("AdoptionDate = %v, want %v", res.AdoptionDate, expectedDate)
	}

	// Verify preface
	if res.Title != "Situation of human rights in Country X" {
		t.Errorf("Title = %q, want %q", res.Title, "Situation of human rights in Country X")
	}
	if res.Proponent != "adopted by the General Assembly" {
		t.Errorf("Proponent = %q, want %q", res.Proponent, "adopted by the General Assembly")
	}

	// Verify preamble
	if len(res.Preamble) != 3 {
		t.Fatalf("len(Preamble) = %d, want 3", len(res.Preamble))
	}
	if res.Preamble[0].IntroPhrase != "Guided by" {
		t.Errorf("Preamble[0].IntroPhrase = %q, want %q", res.Preamble[0].IntroPhrase, "Guided by")
	}
	if res.Preamble[1].IntroPhrase != "Recalling" {
		t.Errorf("Preamble[1].IntroPhrase = %q, want %q", res.Preamble[1].IntroPhrase, "Recalling")
	}
	if res.Preamble[2].IntroPhrase != "Expressing grave concern" {
		t.Errorf("Preamble[2].IntroPhrase = %q, want %q", res.Preamble[2].IntroPhrase, "Expressing grave concern")
	}

	// Verify operative paragraphs
	if len(res.OperativeParts) != 3 {
		t.Fatalf("len(OperativeParts) = %d, want 3", len(res.OperativeParts))
	}
	if res.OperativeParts[0].Number != 1 {
		t.Errorf("OperativeParts[0].Number = %d, want 1", res.OperativeParts[0].Number)
	}
	if res.OperativeParts[0].Action != "Condemns" {
		t.Errorf("OperativeParts[0].Action = %q, want %q", res.OperativeParts[0].Action, "Condemns")
	}
	if res.OperativeParts[1].Action != "Calls upon" {
		t.Errorf("OperativeParts[1].Action = %q, want %q", res.OperativeParts[1].Action, "Calls upon")
	}
	if res.OperativeParts[2].Action != "Decides" {
		t.Errorf("OperativeParts[2].Action = %q, want %q", res.OperativeParts[2].Action, "Decides")
	}

	// Verify references
	if len(res.Organizations) != 2 {
		t.Errorf("len(Organizations) = %d, want 2", len(res.Organizations))
	}
	if len(res.Concepts) != 1 {
		t.Errorf("len(Concepts) = %d, want 1", len(res.Concepts))
	}
	if res.Concepts[0].ShowAs != "human rights" {
		t.Errorf("Concepts[0].ShowAs = %q, want %q", res.Concepts[0].ShowAs, "human rights")
	}
}

func TestParseAKNResolution_SC(t *testing.T) {
	res, err := ParseAKNResolution([]byte(sampleSCResolutionXML))
	if err != nil {
		t.Fatalf("ParseAKNResolution() error = %v", err)
	}

	// Verify SC-specific fields
	if res.Body != BodySecurityCouncil {
		t.Errorf("Body = %v, want %v", res.Body, BodySecurityCouncil)
	}
	if res.Number != 2798 {
		t.Errorf("Number = %d, want 2798", res.Number)
	}

	// Verify operative paragraphs
	if len(res.OperativeParts) != 2 {
		t.Fatalf("len(OperativeParts) = %d, want 2", len(res.OperativeParts))
	}
	if res.OperativeParts[0].Action != "Demands" {
		t.Errorf("OperativeParts[0].Action = %q, want %q", res.OperativeParts[0].Action, "Demands")
	}
	if res.OperativeParts[1].Action != "Authorizes" {
		t.Errorf("OperativeParts[1].Action = %q, want %q", res.OperativeParts[1].Action, "Authorizes")
	}
}

func TestExtractActionVerb(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Condemns the ongoing violations", "Condemns"},
		{"Strongly condemns the attacks", "Strongly condemns"},
		{"Calls upon all parties", "Calls upon"},
		{"Urges Member States", "Urges"},
		{"Demands an immediate ceasefire", "Demands"},
		{"Decides to remain seized", "Decides"},
		{"Requests the Secretary-General", "Requests"},
		{"Encourages all parties", "Encourages"},
		{"Authorizes the deployment", "Authorizes"},
		{"Some other text", ""},
	}

	for _, tt := range tests {
		got := extractActionVerb(tt.input)
		if got != tt.want {
			t.Errorf("extractActionVerb(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractDocumentReferences(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"Recalling its resolution 78/200 of 20 December 2023", 1},
		{"Recalling A/RES/78/200 and A/RES/77/150", 2},
		{"Recalling S/RES/2798", 1},
		{"No references here", 0},
	}

	for _, tt := range tests {
		refs := extractDocumentReferences(tt.input)
		if len(refs) != tt.want {
			t.Errorf("extractDocumentReferences(%q) = %d refs, want %d", tt.input, len(refs), tt.want)
		}
	}
}

func TestExtractRecital(t *testing.T) {
	tests := []struct {
		content     string
		wantPhrase  string
		wantHasText bool
	}{
		{"Guided by the Charter of the United Nations", "Guided by", true},
		{"Recalling its resolution 78/200", "Recalling", true},
		{"Expressing grave concern at the situation", "Expressing grave concern", true},
		{"Some text without intro phrase", "", true},
	}

	for _, tt := range tests {
		recital := extractRecital(tt.content)
		if recital.IntroPhrase != tt.wantPhrase {
			t.Errorf("extractRecital(%q).IntroPhrase = %q, want %q", tt.content, recital.IntroPhrase, tt.wantPhrase)
		}
		hasText := recital.Text != ""
		if hasText != tt.wantHasText {
			t.Errorf("extractRecital(%q) hasText = %v, want %v", tt.content, hasText, tt.wantHasText)
		}
	}
}

func TestCleanXMLContent(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<i>Condemns</i> the violations", "Condemns the violations"},
		{"<p>Simple text</p>", "Simple text"},
		{"Text with &amp; entity", "Text with & entity"},
		{"Multiple   spaces", "Multiple spaces"},
		{"  Trimmed  ", "Trimmed"},
	}

	for _, tt := range tests {
		got := cleanXMLContent(tt.input)
		if got != tt.want {
			t.Errorf("cleanXMLContent(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolutionToTriples(t *testing.T) {
	res, _ := ParseAKNResolution([]byte(sampleAKNResolutionXML))
	triples := ResolutionToTriples(res, "un:")

	if len(triples) == 0 {
		t.Fatal("ResolutionToTriples() returned no triples")
	}

	// Build triple map
	tripleMap := make(map[string]map[string]string)
	for _, tr := range triples {
		if tripleMap[tr.Subject] == nil {
			tripleMap[tr.Subject] = make(map[string]string)
		}
		tripleMap[tr.Subject][tr.Predicate] = tr.Object
	}

	// Verify resolution
	resURI := "un:resolution/a-res-79-100"
	if tripleMap[resURI] == nil {
		t.Fatal("Resolution URI not found in triples")
	}
	if tripleMap[resURI]["rdf:type"] != "reg:Resolution" {
		t.Errorf("Resolution type = %q, want %q", tripleMap[resURI]["rdf:type"], "reg:Resolution")
	}
	if tripleMap[resURI]["reg:session"] != "79" {
		t.Errorf("Session = %q, want %q", tripleMap[resURI]["reg:session"], "79")
	}

	// Verify preamble
	preambleURI := resURI + "/preamble/1"
	if tripleMap[preambleURI] == nil {
		t.Fatal("Preamble URI not found in triples")
	}
	if tripleMap[preambleURI]["reg:introPhrase"] != "Guided by" {
		t.Errorf("Preamble introPhrase = %q, want %q", tripleMap[preambleURI]["reg:introPhrase"], "Guided by")
	}

	// Verify operative paragraph
	paraURI := resURI + "/para/1"
	if tripleMap[paraURI] == nil {
		t.Fatal("Paragraph URI not found in triples")
	}
	if tripleMap[paraURI]["reg:action"] != "Condemns" {
		t.Errorf("Paragraph action = %q, want %q", tripleMap[paraURI]["reg:action"], "Condemns")
	}

	// Verify concept
	conceptURI := "un:concept/humanrights"
	if tripleMap[conceptURI] == nil {
		t.Fatal("Concept URI not found in triples")
	}
	if tripleMap[conceptURI][store.RDFSLabel] != "human rights" {
		t.Errorf("Concept label = %q, want %q", tripleMap[conceptURI][store.RDFSLabel], "human rights")
	}
}

func TestIngestResolutions(t *testing.T) {
	res, _ := ParseAKNResolution([]byte(sampleAKNResolutionXML))
	ts := store.NewTripleStore()

	err := IngestResolutions([]*UNResolution{res}, ts, "un:")
	if err != nil {
		t.Fatalf("IngestResolutions() error = %v", err)
	}

	// Verify resolution was ingested
	resTriples := ts.Find("un:resolution/a-res-79-100", "rdf:type", "")
	if len(resTriples) == 0 {
		t.Error("Resolution was not ingested")
	}

	// Verify preamble was ingested
	preambleTriples := ts.Find("un:resolution/a-res-79-100/preamble/1", "reg:introPhrase", "")
	if len(preambleTriples) == 0 {
		t.Error("Preamble was not ingested")
	}
}

func TestIngestResolutionsNilStore(t *testing.T) {
	res, _ := ParseAKNResolution([]byte(sampleAKNResolutionXML))
	err := IngestResolutions([]*UNResolution{res}, nil, "un:")
	if err == nil {
		t.Error("IngestResolutions() should return error for nil store")
	}
}

func TestSanitizeURI(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"A/RES/79/100", "a-res-79-100"},
		{"S/RES/2798", "s-res-2798"},
		{"Some Text", "some-text"},
		{"With (Special) Chars!", "with-special-chars"},
	}

	for _, tt := range tests {
		got := sanitizeURI(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeURI(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMapActionToObligation(t *testing.T) {
	tests := []struct {
		action   string
		wantNil  bool
		contains string
	}{
		{"Condemns", false, "condemnation"},
		{"Strongly condemns", false, "condemnation"},
		{"Calls upon", false, "call-to-action"},
		{"Demands", false, "demand"},
		{"Decides", false, "decision"},
		{"Requests", false, "request"},
		{"Authorizes", false, "authorization"},
		{"Unknown action", true, ""},
	}

	for _, tt := range tests {
		got := mapActionToObligation(tt.action, "un:")
		if tt.wantNil && got != "" {
			t.Errorf("mapActionToObligation(%q) = %q, want empty", tt.action, got)
		}
		if !tt.wantNil && got == "" {
			t.Errorf("mapActionToObligation(%q) = empty, want non-empty", tt.action)
		}
		if tt.contains != "" && got != "" {
			if got != "un:obligation/"+tt.contains {
				t.Errorf("mapActionToObligation(%q) = %q, want to contain %q", tt.action, got, tt.contains)
			}
		}
	}
}

func TestNewUNResolutionConnector(t *testing.T) {
	connector := NewUNResolutionConnector("/path/to/repos")
	if connector == nil {
		t.Fatal("NewUNResolutionConnector() returned nil")
	}
	if connector.LocalPath != "/path/to/repos" {
		t.Errorf("LocalPath = %q, want %q", connector.LocalPath, "/path/to/repos")
	}
}

func TestListResolutions(t *testing.T) {
	// Create temp directory with sample files
	tempDir, err := os.MkdirTemp("", "unres-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create GA resolutions directory
	gaDir := filepath.Join(tempDir, "GAresolutions")
	if err := os.MkdirAll(gaDir, 0755); err != nil {
		t.Fatalf("Failed to create GA dir: %v", err)
	}

	// Create sample files
	files := []string{
		"A_RES_79_100.xml",
		"A_RES_79_101.xml",
		"A_RES_78_50.xml",
	}
	for _, f := range files {
		path := filepath.Join(gaDir, f)
		if err := os.WriteFile(path, []byte(sampleAKNResolutionXML), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	connector := NewUNResolutionConnector(tempDir)
	refs, err := connector.ListResolutions(BodyGeneralAssembly, 0)
	if err != nil {
		t.Fatalf("ListResolutions() error = %v", err)
	}

	if len(refs) != 3 {
		t.Errorf("len(refs) = %d, want 3", len(refs))
	}

	// Test session filtering
	refs79, err := connector.ListResolutions(BodyGeneralAssembly, 79)
	if err != nil {
		t.Fatalf("ListResolutions(session=79) error = %v", err)
	}
	if len(refs79) != 2 {
		t.Errorf("len(refs79) = %d, want 2", len(refs79))
	}
}

func TestParseResolution(t *testing.T) {
	// Create temp file
	tempFile, err := os.CreateTemp("", "unres-test-*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.WriteString(sampleAKNResolutionXML); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tempFile.Close()

	connector := NewUNResolutionConnector("")
	res, err := connector.ParseResolution(tempFile.Name())
	if err != nil {
		t.Fatalf("ParseResolution() error = %v", err)
	}

	if res.Session != 79 {
		t.Errorf("Session = %d, want 79", res.Session)
	}
	if res.SourcePath != tempFile.Name() {
		t.Errorf("SourcePath = %q, want %q", res.SourcePath, tempFile.Name())
	}
}

func TestParseResolutionPath(t *testing.T) {
	tests := []struct {
		path        string
		body        UNBody
		wantSession int
		wantNumber  int
	}{
		{"/path/to/A_RES_79_100.xml", BodyGeneralAssembly, 79, 100},
		{"/path/to/A_RES_78_50.xml", BodyGeneralAssembly, 78, 50},
		{"/path/to/S_RES_2798.xml", BodySecurityCouncil, 0, 2798},
	}

	for _, tt := range tests {
		ref := parseResolutionPath(tt.path, tt.body)
		if ref == nil {
			t.Errorf("parseResolutionPath(%q) = nil", tt.path)
			continue
		}
		if ref.Session != tt.wantSession {
			t.Errorf("parseResolutionPath(%q).Session = %d, want %d", tt.path, ref.Session, tt.wantSession)
		}
		if ref.Number != tt.wantNumber {
			t.Errorf("parseResolutionPath(%q).Number = %d, want %d", tt.path, ref.Number, tt.wantNumber)
		}
	}
}

func TestExtractSessionNumber(t *testing.T) {
	tests := []struct {
		uri         string
		body        UNBody
		wantSession int
		wantNumber  int
	}{
		{"/akn/un/statement/deliberation/unga/2024-12-20/79-100/!main", BodyGeneralAssembly, 79, 100},
		{"/akn/un/statement/deliberation/unsc/2024-03-15/2798/!main", BodySecurityCouncil, 0, 2798},
	}

	for _, tt := range tests {
		res := &UNResolution{Body: tt.body}
		extractSessionNumber(tt.uri, res)
		if res.Session != tt.wantSession {
			t.Errorf("extractSessionNumber(%q).Session = %d, want %d", tt.uri, res.Session, tt.wantSession)
		}
		if res.Number != tt.wantNumber {
			t.Errorf("extractSessionNumber(%q).Number = %d, want %d", tt.uri, res.Number, tt.wantNumber)
		}
	}
}

func TestPreambleWithReferences(t *testing.T) {
	res, _ := ParseAKNResolution([]byte(sampleAKNResolutionXML))

	// The second preamble recital references resolution 78/200
	if len(res.Preamble) < 2 {
		t.Fatal("Expected at least 2 preamble recitals")
	}

	refs := res.Preamble[1].References
	if len(refs) != 1 {
		t.Errorf("Preamble[1].References = %d, want 1", len(refs))
	} else if refs[0].Identifier != "A/RES/78/200" {
		t.Errorf("Preamble[1].References[0].Identifier = %q, want %q", refs[0].Identifier, "A/RES/78/200")
	}
}
