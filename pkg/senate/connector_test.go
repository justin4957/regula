package senate

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// Sample vote menu XML fixture
const sampleVoteMenuXML = `<?xml version="1.0" encoding="UTF-8"?>
<vote_summary>
  <congress>119</congress>
  <session>1</session>
  <votes>
    <vote>
      <vote_number>00001</vote_number>
      <vote_date>January 3, 2025</vote_date>
      <issue>PN1</issue>
      <question>On the Motion</question>
      <result>Agreed to</result>
      <title>Motion to Table Schumer Amdt. No. 1</title>
    </vote>
    <vote>
      <vote_number>00002</vote_number>
      <vote_date>January 6, 2025</vote_date>
      <issue>S.1</issue>
      <question>On Passage</question>
      <result>Passed</result>
      <title>Laken Riley Act</title>
    </vote>
  </votes>
</vote_summary>`

// Sample roll call vote XML fixture
const sampleRollCallVoteXML = `<?xml version="1.0" encoding="UTF-8"?>
<roll_call_vote>
  <congress>119</congress>
  <session>1</session>
  <congress_year>2025</congress_year>
  <vote_number>00001</vote_number>
  <vote_date>January 3, 2025, 12:05 PM</vote_date>
  <modify_date>January 3, 2025, 12:19 PM</modify_date>
  <vote_question_text>On the Motion (Motion to Table Schumer Amdt. No. 1)</vote_question_text>
  <vote_document_text>Motion to Table Schumer Amendment No. 1</vote_document_text>
  <vote_result_text>Motion to Table Agreed to</vote_result_text>
  <question>On the Motion</question>
  <vote_title>Motion to Table Schumer Amdt. No. 1</vote_title>
  <majority_requirement>1/2</majority_requirement>
  <vote_result>Motion to Table Agreed to</vote_result>
  <document>
    <document_type>PN</document_type>
    <document_number>1</document_number>
    <document_title>Marco Rubio, of Florida, to be Secretary of State</document_title>
  </document>
  <count>
    <yeas>52</yeas>
    <nays>47</nays>
    <present>0</present>
    <absent>1</absent>
  </count>
  <members>
    <member>
      <member_full>Baldwin (D-WI)</member_full>
      <last_name>Baldwin</last_name>
      <first_name>Tammy</first_name>
      <party>D</party>
      <state>WI</state>
      <vote_cast>Nay</vote_cast>
      <lis_member_id>S354</lis_member_id>
    </member>
    <member>
      <member_full>Barrasso (R-WY)</member_full>
      <last_name>Barrasso</last_name>
      <first_name>John</first_name>
      <party>R</party>
      <state>WY</state>
      <vote_cast>Yea</vote_cast>
      <lis_member_id>S317</lis_member_id>
    </member>
    <member>
      <member_full>Fetterman (D-PA)</member_full>
      <last_name>Fetterman</last_name>
      <first_name>John</first_name>
      <party>D</party>
      <state>PA</state>
      <vote_cast>Not Voting</vote_cast>
      <lis_member_id>S412</lis_member_id>
    </member>
  </members>
</roll_call_vote>`

func TestParseVotePosition(t *testing.T) {
	tests := []struct {
		input string
		want  VotePosition
	}{
		{"Yea", VoteYea},
		{"yea", VoteYea},
		{"YEA", VoteYea},
		{"Aye", VoteYea},
		{"Yes", VoteYea},
		{"Nay", VoteNay},
		{"nay", VoteNay},
		{"No", VoteNay},
		{"Present", VotePresent},
		{"Not Voting", VoteNotVoting},
		{"not voting", VoteNotVoting},
		{"NotVoting", VoteNotVoting},
		{"Absent", VoteAbsent},
		{"unknown", VoteNotVoting},
	}

	for _, tt := range tests {
		got := ParseVotePosition(tt.input)
		if got != tt.want {
			t.Errorf("ParseVotePosition(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseVoteMenu(t *testing.T) {
	menu, err := parseVoteMenu([]byte(sampleVoteMenuXML), 119, 1)
	if err != nil {
		t.Fatalf("parseVoteMenu() error = %v", err)
	}

	if menu.Congress != 119 {
		t.Errorf("Congress = %d, want 119", menu.Congress)
	}
	if menu.Session != 1 {
		t.Errorf("Session = %d, want 1", menu.Session)
	}
	if len(menu.Votes) != 2 {
		t.Fatalf("len(Votes) = %d, want 2", len(menu.Votes))
	}

	// Check first vote
	vote1 := menu.Votes[0]
	if vote1.VoteNumber != 1 {
		t.Errorf("Vote[0].VoteNumber = %d, want 1", vote1.VoteNumber)
	}
	if vote1.Question != "On the Motion" {
		t.Errorf("Vote[0].Question = %q, want %q", vote1.Question, "On the Motion")
	}
	if vote1.Result != "Agreed to" {
		t.Errorf("Vote[0].Result = %q, want %q", vote1.Result, "Agreed to")
	}

	// Check second vote
	vote2 := menu.Votes[1]
	if vote2.VoteNumber != 2 {
		t.Errorf("Vote[1].VoteNumber = %d, want 2", vote2.VoteNumber)
	}
	if vote2.VoteTitle != "Laken Riley Act" {
		t.Errorf("Vote[1].VoteTitle = %q, want %q", vote2.VoteTitle, "Laken Riley Act")
	}
}

func TestParseRollCallVote(t *testing.T) {
	vote, err := parseRollCallVote([]byte(sampleRollCallVoteXML))
	if err != nil {
		t.Fatalf("parseRollCallVote() error = %v", err)
	}

	// Verify basic vote info
	if vote.Congress != 119 {
		t.Errorf("Congress = %d, want 119", vote.Congress)
	}
	if vote.Session != 1 {
		t.Errorf("Session = %d, want 1", vote.Session)
	}
	if vote.VoteNumber != 1 {
		t.Errorf("VoteNumber = %d, want 1", vote.VoteNumber)
	}
	if vote.Question != "On the Motion" {
		t.Errorf("Question = %q, want %q", vote.Question, "On the Motion")
	}
	if vote.MajorityReq != "1/2" {
		t.Errorf("MajorityReq = %q, want %q", vote.MajorityReq, "1/2")
	}

	// Verify vote counts
	if vote.Count.Yeas != 52 {
		t.Errorf("Count.Yeas = %d, want 52", vote.Count.Yeas)
	}
	if vote.Count.Nays != 47 {
		t.Errorf("Count.Nays = %d, want 47", vote.Count.Nays)
	}
	if vote.Count.Present != 0 {
		t.Errorf("Count.Present = %d, want 0", vote.Count.Present)
	}
	if vote.Count.Absent != 1 {
		t.Errorf("Count.Absent = %d, want 1", vote.Count.Absent)
	}

	// Verify document
	if vote.Document == nil {
		t.Fatal("Document is nil")
	}
	if vote.Document.Type != "PN" {
		t.Errorf("Document.Type = %q, want %q", vote.Document.Type, "PN")
	}
	if vote.Document.Number != "1" {
		t.Errorf("Document.Number = %q, want %q", vote.Document.Number, "1")
	}

	// Verify members
	if len(vote.Members) != 3 {
		t.Fatalf("len(Members) = %d, want 3", len(vote.Members))
	}

	// Check Baldwin
	baldwin := vote.Members[0]
	if baldwin.LastName != "Baldwin" {
		t.Errorf("Member[0].LastName = %q, want %q", baldwin.LastName, "Baldwin")
	}
	if baldwin.Party != "D" {
		t.Errorf("Member[0].Party = %q, want %q", baldwin.Party, "D")
	}
	if baldwin.State != "WI" {
		t.Errorf("Member[0].State = %q, want %q", baldwin.State, "WI")
	}
	if baldwin.Vote != VoteNay {
		t.Errorf("Member[0].Vote = %v, want %v", baldwin.Vote, VoteNay)
	}
	if baldwin.LISMemberID != "S354" {
		t.Errorf("Member[0].LISMemberID = %q, want %q", baldwin.LISMemberID, "S354")
	}

	// Check Barrasso
	barrasso := vote.Members[1]
	if barrasso.Vote != VoteYea {
		t.Errorf("Member[1].Vote = %v, want %v", barrasso.Vote, VoteYea)
	}

	// Check Fetterman (Not Voting)
	fetterman := vote.Members[2]
	if fetterman.Vote != VoteNotVoting {
		t.Errorf("Member[2].Vote = %v, want %v", fetterman.Vote, VoteNotVoting)
	}
}

func TestParseVoteDate(t *testing.T) {
	tests := []struct {
		input string
		want  time.Time
	}{
		{
			"January 3, 2025, 12:05 PM",
			time.Date(2025, 1, 3, 12, 5, 0, 0, time.UTC),
		},
		{
			"January 3, 2025",
			time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC),
		},
		{
			"",
			time.Time{},
		},
	}

	for _, tt := range tests {
		got := parseVoteDate(tt.input)
		// Compare year, month, day for simplicity
		if !tt.want.IsZero() {
			if got.Year() != tt.want.Year() || got.Month() != tt.want.Month() || got.Day() != tt.want.Day() {
				t.Errorf("parseVoteDate(%q) = %v, want %v", tt.input, got, tt.want)
			}
		} else if !got.IsZero() {
			t.Errorf("parseVoteDate(%q) = %v, want zero time", tt.input, got)
		}
	}
}

func TestVoteToTriples(t *testing.T) {
	vote, _ := parseRollCallVote([]byte(sampleRollCallVoteXML))
	triples := VoteToTriples(vote, "senate:")

	if len(triples) == 0 {
		t.Fatal("VoteToTriples() returned no triples")
	}

	// Check for expected triples
	tripleMap := make(map[string]map[string]string)
	for _, tr := range triples {
		if tripleMap[tr.Subject] == nil {
			tripleMap[tr.Subject] = make(map[string]string)
		}
		tripleMap[tr.Subject][tr.Predicate] = tr.Object
	}

	// Verify vote record
	voteURI := "senate:vote/119/1/1"
	if tripleMap[voteURI] == nil {
		t.Fatal("Vote URI not found in triples")
	}
	if tripleMap[voteURI]["rdf:type"] != store.ClassVoteRecord {
		t.Errorf("Vote type = %q, want %q", tripleMap[voteURI]["rdf:type"], store.ClassVoteRecord)
	}
	if tripleMap[voteURI]["reg:congress"] != "119" {
		t.Errorf("Congress = %q, want %q", tripleMap[voteURI]["reg:congress"], "119")
	}
	if tripleMap[voteURI][store.PropVoteFor] != "52" {
		t.Errorf("VoteFor = %q, want %q", tripleMap[voteURI][store.PropVoteFor], "52")
	}
	if tripleMap[voteURI][store.PropVoteAgainst] != "47" {
		t.Errorf("VoteAgainst = %q, want %q", tripleMap[voteURI][store.PropVoteAgainst], "47")
	}

	// Verify senator entity
	memberURI := "senate:member/S354"
	if tripleMap[memberURI] == nil {
		t.Fatal("Member URI S354 not found in triples")
	}
	if tripleMap[memberURI]["reg:party"] != "D" {
		t.Errorf("Member party = %q, want %q", tripleMap[memberURI]["reg:party"], "D")
	}
	if tripleMap[memberURI]["reg:state"] != "WI" {
		t.Errorf("Member state = %q, want %q", tripleMap[memberURI]["reg:state"], "WI")
	}

	// Verify individual vote
	indVoteURI := "senate:vote/119/1/1/baldwin"
	if tripleMap[indVoteURI] == nil {
		t.Fatal("Individual vote URI not found in triples")
	}
	if tripleMap[indVoteURI][store.PropVotePosition] != "Nay" {
		t.Errorf("VotePosition = %q, want %q", tripleMap[indVoteURI][store.PropVotePosition], "Nay")
	}
	if tripleMap[indVoteURI][store.PropVoter] != memberURI {
		t.Errorf("Voter = %q, want %q", tripleMap[indVoteURI][store.PropVoter], memberURI)
	}

	// Verify document
	docURI := "senate:document/PN/1"
	if tripleMap[docURI] == nil {
		t.Fatal("Document URI not found in triples")
	}
	if tripleMap[docURI]["reg:documentType"] != "PN" {
		t.Errorf("Document type = %q, want %q", tripleMap[docURI]["reg:documentType"], "PN")
	}
}

func TestIngestVotes(t *testing.T) {
	vote, _ := parseRollCallVote([]byte(sampleRollCallVoteXML))
	ts := store.NewTripleStore()

	err := IngestVotes([]*RollCallVote{vote}, ts, "senate:")
	if err != nil {
		t.Fatalf("IngestVotes() error = %v", err)
	}

	// Verify vote was ingested
	voteTriples := ts.Find("senate:vote/119/1/1", "rdf:type", "")
	if len(voteTriples) == 0 {
		t.Error("Vote was not ingested")
	}

	// Verify member was ingested
	memberTriples := ts.Find("senate:member/S354", "reg:party", "")
	if len(memberTriples) == 0 {
		t.Error("Member was not ingested")
	}
	if memberTriples[0].Object != "D" {
		t.Errorf("Member party = %q, want %q", memberTriples[0].Object, "D")
	}
}

func TestSenateVoteConnector_FetchVoteMenu(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(sampleVoteMenuXML))
	}))
	defer server.Close()

	connector := NewSenateVoteConnector(ConnectorConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		RateLimit:  time.Millisecond,
	})

	menu, err := connector.FetchVoteMenu(119, 1)
	if err != nil {
		t.Fatalf("FetchVoteMenu() error = %v", err)
	}

	if len(menu.Votes) != 2 {
		t.Errorf("len(Votes) = %d, want 2", len(menu.Votes))
	}
}

func TestSenateVoteConnector_FetchVote(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(sampleRollCallVoteXML))
	}))
	defer server.Close()

	connector := NewSenateVoteConnector(ConnectorConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		RateLimit:  time.Millisecond,
	})

	vote, err := connector.FetchVote(119, 1, 1)
	if err != nil {
		t.Fatalf("FetchVote() error = %v", err)
	}

	if vote.VoteNumber != 1 {
		t.Errorf("VoteNumber = %d, want 1", vote.VoteNumber)
	}
	if vote.Count.Yeas != 52 {
		t.Errorf("Count.Yeas = %d, want 52", vote.Count.Yeas)
	}
}

func TestNewDefaultConnector(t *testing.T) {
	connector := NewDefaultConnector()
	if connector == nil {
		t.Fatal("NewDefaultConnector() returned nil")
	}
	if connector.config.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %q, want %q", connector.config.BaseURL, DefaultBaseURL)
	}
	if connector.config.UserAgent != DefaultUserAgent {
		t.Errorf("UserAgent = %q, want %q", connector.config.UserAgent, DefaultUserAgent)
	}
}

func TestSanitizeURI(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Baldwin", "baldwin"},
		{"O'Connor", "oconnor"},
		{"Van Hollen", "van-hollen"},
		{"McCain III", "mccain-iii"},
		{"Duckworth", "duckworth"},
	}

	for _, tt := range tests {
		got := sanitizeURI(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeURI(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestVotePositionString(t *testing.T) {
	tests := []struct {
		input VotePosition
		want  string
	}{
		{VoteYea, "Yea"},
		{VoteNay, "Nay"},
		{VotePresent, "Present"},
		{VoteNotVoting, "NotVoting"},
		{VoteAbsent, "Absent"},
	}

	for _, tt := range tests {
		got := tt.input.String()
		if got != tt.want {
			t.Errorf("(%v).String() = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFetchVotesSince(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		if callCount == 0 {
			// First call is vote menu
			w.Write([]byte(sampleVoteMenuXML))
		} else {
			// Subsequent calls are individual votes
			w.Write([]byte(sampleRollCallVoteXML))
		}
		callCount++
	}))
	defer server.Close()

	connector := NewSenateVoteConnector(ConnectorConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		RateLimit:  time.Millisecond,
	})

	// Fetch votes since vote 1 (should only get vote 2)
	votes, err := connector.FetchVotesSince(119, 1, 1)
	if err != nil {
		t.Fatalf("FetchVotesSince() error = %v", err)
	}

	// Should have fetched vote 2 (vote 1 is <= lastVoteNumber)
	if len(votes) != 1 {
		t.Errorf("len(votes) = %d, want 1", len(votes))
	}
}

func TestEmptyVoteMenu(t *testing.T) {
	emptyMenuXML := `<?xml version="1.0" encoding="UTF-8"?>
<vote_summary>
  <congress>119</congress>
  <session>1</session>
  <votes></votes>
</vote_summary>`

	menu, err := parseVoteMenu([]byte(emptyMenuXML), 119, 1)
	if err != nil {
		t.Fatalf("parseVoteMenu() error = %v", err)
	}

	if len(menu.Votes) != 0 {
		t.Errorf("len(Votes) = %d, want 0", len(menu.Votes))
	}
}

func TestVoteWithNoDocument(t *testing.T) {
	noDocXML := `<?xml version="1.0" encoding="UTF-8"?>
<roll_call_vote>
  <congress>119</congress>
  <session>1</session>
  <vote_number>00001</vote_number>
  <vote_date>January 3, 2025</vote_date>
  <question>On the Motion</question>
  <vote_result>Agreed to</vote_result>
  <count>
    <yeas>52</yeas>
    <nays>47</nays>
    <present>0</present>
    <absent>1</absent>
  </count>
  <members></members>
</roll_call_vote>`

	vote, err := parseRollCallVote([]byte(noDocXML))
	if err != nil {
		t.Fatalf("parseRollCallVote() error = %v", err)
	}

	if vote.Document != nil {
		t.Error("Document should be nil when not present")
	}
}

func TestIngestVotesNilStore(t *testing.T) {
	vote, _ := parseRollCallVote([]byte(sampleRollCallVoteXML))
	err := IngestVotes([]*RollCallVote{vote}, nil, "senate:")
	if err == nil {
		t.Error("IngestVotes() should return error for nil store")
	}
}
