// Package senate provides a connector for fetching US Senate roll call votes
// from the publicly available XML feeds at senate.gov.
package senate

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// DefaultBaseURL is the base URL for Senate XML feeds.
const DefaultBaseURL = "https://www.senate.gov/legislative"

// DefaultUserAgent is the User-Agent header sent with requests.
const DefaultUserAgent = "regula-senate-connector/1.0"

// DefaultRateLimit is the minimum interval between requests.
const DefaultRateLimit = 500 * time.Millisecond

// VotePosition represents how a senator voted.
type VotePosition string

const (
	VoteYea       VotePosition = "Yea"
	VoteNay       VotePosition = "Nay"
	VotePresent   VotePosition = "Present"
	VoteNotVoting VotePosition = "NotVoting"
	VoteAbsent    VotePosition = "Absent"
)

// ParseVotePosition converts a string to a VotePosition.
func ParseVotePosition(s string) VotePosition {
	normalized := strings.TrimSpace(strings.ToLower(s))
	switch normalized {
	case "yea", "aye", "yes":
		return VoteYea
	case "nay", "no":
		return VoteNay
	case "present":
		return VotePresent
	case "not voting", "not_voting", "notvoting":
		return VoteNotVoting
	case "absent":
		return VoteAbsent
	default:
		return VoteNotVoting
	}
}

// String returns the string representation of a VotePosition.
func (v VotePosition) String() string {
	return string(v)
}

// MemberVote represents an individual senator's vote.
type MemberVote struct {
	FullName    string       `json:"full_name"`
	LastName    string       `json:"last_name"`
	FirstName   string       `json:"first_name"`
	Party       string       `json:"party"`
	State       string       `json:"state"`
	Vote        VotePosition `json:"vote"`
	LISMemberID string       `json:"lis_member_id"`
}

// VoteCount holds the vote tally.
type VoteCount struct {
	Yeas    int `json:"yeas"`
	Nays    int `json:"nays"`
	Present int `json:"present"`
	Absent  int `json:"absent"`
}

// VoteDocument represents the document being voted on.
type VoteDocument struct {
	Type   string `json:"type"`
	Number string `json:"number"`
	Title  string `json:"title"`
}

// RollCallVote represents a complete roll call vote record.
type RollCallVote struct {
	Congress     int           `json:"congress"`
	Session      int           `json:"session"`
	VoteNumber   int           `json:"vote_number"`
	VoteDate     time.Time     `json:"vote_date"`
	ModifyDate   time.Time     `json:"modify_date,omitempty"`
	Question     string        `json:"question"`
	QuestionText string        `json:"question_text"`
	DocumentText string        `json:"document_text"`
	Result       string        `json:"result"`
	ResultText   string        `json:"result_text"`
	MajorityReq  string        `json:"majority_req"`
	VoteTitle    string        `json:"vote_title"`
	Document     *VoteDocument `json:"document,omitempty"`
	Count        VoteCount     `json:"count"`
	Members      []MemberVote  `json:"members"`
}

// VoteMenuEntry represents an entry in the vote menu listing.
type VoteMenuEntry struct {
	VoteNumber int       `json:"vote_number"`
	VoteDate   time.Time `json:"vote_date"`
	Issue      string    `json:"issue"`
	Question   string    `json:"question"`
	Result     string    `json:"result"`
	VoteTitle  string    `json:"vote_title"`
}

// VoteMenu represents the list of votes for a congress/session.
type VoteMenu struct {
	Congress int             `json:"congress"`
	Session  int             `json:"session"`
	Year     int             `json:"year"`
	Votes    []VoteMenuEntry `json:"votes"`
}

// ConnectorConfig holds configuration for the SenateVoteConnector.
type ConnectorConfig struct {
	// BaseURL is the base URL for Senate XML feeds.
	BaseURL string

	// HTTPClient is the underlying HTTP client.
	HTTPClient *http.Client

	// RateLimit is the minimum interval between requests.
	RateLimit time.Duration

	// UserAgent is the User-Agent header.
	UserAgent string
}

// DefaultConfig returns a ConnectorConfig with sensible defaults.
func DefaultConfig() ConnectorConfig {
	return ConnectorConfig{
		BaseURL:    DefaultBaseURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		RateLimit:  DefaultRateLimit,
		UserAgent:  DefaultUserAgent,
	}
}

// SenateVoteConnector fetches US Senate roll call votes from senate.gov.
type SenateVoteConnector struct {
	config       ConnectorConfig
	lastRequest  time.Time
	lastReqMutex sync.Mutex
}

// NewSenateVoteConnector creates a new connector with the given configuration.
func NewSenateVoteConnector(config ConnectorConfig) *SenateVoteConnector {
	if config.BaseURL == "" {
		config.BaseURL = DefaultBaseURL
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	if config.RateLimit == 0 {
		config.RateLimit = DefaultRateLimit
	}
	if config.UserAgent == "" {
		config.UserAgent = DefaultUserAgent
	}
	return &SenateVoteConnector{config: config}
}

// NewDefaultConnector creates a connector with default configuration.
func NewDefaultConnector() *SenateVoteConnector {
	return NewSenateVoteConnector(DefaultConfig())
}

// rateLimit ensures we don't exceed the rate limit.
func (c *SenateVoteConnector) rateLimit() {
	c.lastReqMutex.Lock()
	defer c.lastReqMutex.Unlock()

	elapsed := time.Since(c.lastRequest)
	if elapsed < c.config.RateLimit {
		time.Sleep(c.config.RateLimit - elapsed)
	}
	c.lastRequest = time.Now()
}

// fetch performs an HTTP GET request with rate limiting.
func (c *SenateVoteConnector) fetch(url string) ([]byte, error) {
	c.rateLimit()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", c.config.UserAgent)

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}

// FetchVoteMenu fetches the vote menu listing all votes for a congress/session.
func (c *SenateVoteConnector) FetchVoteMenu(congress, session int) (*VoteMenu, error) {
	url := fmt.Sprintf("%s/LIS/roll_call_lists/vote_menu_%d_%d.xml",
		c.config.BaseURL, congress, session)

	body, err := c.fetch(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch vote menu: %w", err)
	}

	return parseVoteMenu(body, congress, session)
}

// FetchVote fetches an individual roll call vote.
func (c *SenateVoteConnector) FetchVote(congress, session, voteNumber int) (*RollCallVote, error) {
	url := fmt.Sprintf("%s/LIS/roll_call_votes/vote%d%d/vote_%d_%d_%05d.xml",
		c.config.BaseURL, congress, session, congress, session, voteNumber)

	body, err := c.fetch(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch vote %d-%d-%d: %w",
			congress, session, voteNumber, err)
	}

	return parseRollCallVote(body)
}

// FetchAllVotes fetches all votes for a congress/session.
func (c *SenateVoteConnector) FetchAllVotes(congress, session int) ([]*RollCallVote, error) {
	menu, err := c.FetchVoteMenu(congress, session)
	if err != nil {
		return nil, err
	}

	votes := make([]*RollCallVote, 0, len(menu.Votes))
	for _, entry := range menu.Votes {
		vote, err := c.FetchVote(congress, session, entry.VoteNumber)
		if err != nil {
			// Log warning but continue with other votes
			continue
		}
		votes = append(votes, vote)
	}

	return votes, nil
}

// FetchVotesSince fetches votes after the given vote number (for incremental sync).
func (c *SenateVoteConnector) FetchVotesSince(congress, session, lastVoteNumber int) ([]*RollCallVote, error) {
	menu, err := c.FetchVoteMenu(congress, session)
	if err != nil {
		return nil, err
	}

	votes := make([]*RollCallVote, 0)
	for _, entry := range menu.Votes {
		if entry.VoteNumber > lastVoteNumber {
			vote, err := c.FetchVote(congress, session, entry.VoteNumber)
			if err != nil {
				continue
			}
			votes = append(votes, vote)
		}
	}

	return votes, nil
}

// FetchVotesByDateRange fetches votes within a date range.
func (c *SenateVoteConnector) FetchVotesByDateRange(congress, session int, start, end time.Time) ([]*RollCallVote, error) {
	menu, err := c.FetchVoteMenu(congress, session)
	if err != nil {
		return nil, err
	}

	votes := make([]*RollCallVote, 0)
	for _, entry := range menu.Votes {
		if !entry.VoteDate.Before(start) && !entry.VoteDate.After(end) {
			vote, err := c.FetchVote(congress, session, entry.VoteNumber)
			if err != nil {
				continue
			}
			votes = append(votes, vote)
		}
	}

	return votes, nil
}

// XML structures for parsing

type xmlVoteMenu struct {
	XMLName  xml.Name         `xml:"vote_summary"`
	Congress xmlCongress      `xml:"congress"`
	Session  xmlSession       `xml:"session"`
	Votes    []xmlVoteMenuRow `xml:"votes>vote"`
}

type xmlCongress struct {
	Number int `xml:",chardata"`
}

type xmlSession struct {
	Number int `xml:",chardata"`
}

type xmlVoteMenuRow struct {
	VoteNumber string `xml:"vote_number"`
	VoteDate   string `xml:"vote_date"`
	Issue      string `xml:"issue"`
	Question   string `xml:"question"`
	Result     string `xml:"result"`
	VoteTitle  string `xml:"title"`
}

type xmlRollCallVote struct {
	XMLName          xml.Name       `xml:"roll_call_vote"`
	Congress         int            `xml:"congress"`
	Session          int            `xml:"session"`
	CongressYear     int            `xml:"congress_year"`
	VoteNumber       string         `xml:"vote_number"`
	VoteDate         string         `xml:"vote_date"`
	ModifyDate       string         `xml:"modify_date"`
	VoteQuestionText string         `xml:"vote_question_text"`
	VoteDocumentText string         `xml:"vote_document_text"`
	VoteResultText   string         `xml:"vote_result_text"`
	Question         string         `xml:"question"`
	VoteTitle        string         `xml:"vote_title"`
	MajorityReq      string         `xml:"majority_requirement"`
	VoteResult       string         `xml:"vote_result"`
	Document         xmlDocument    `xml:"document"`
	Count            xmlCount       `xml:"count"`
	Members          xmlMembersList `xml:"members"`
}

type xmlDocument struct {
	DocType   string `xml:"document_type"`
	DocNumber string `xml:"document_number"`
	DocTitle  string `xml:"document_title"`
}

type xmlCount struct {
	Yeas    int `xml:"yeas"`
	Nays    int `xml:"nays"`
	Present int `xml:"present"`
	Absent  int `xml:"absent"`
}

type xmlMembersList struct {
	Members []xmlMember `xml:"member"`
}

type xmlMember struct {
	MemberFull  string `xml:"member_full"`
	LastName    string `xml:"last_name"`
	FirstName   string `xml:"first_name"`
	Party       string `xml:"party"`
	State       string `xml:"state"`
	VoteCast    string `xml:"vote_cast"`
	LISMemberID string `xml:"lis_member_id"`
}

// parseVoteMenu parses vote menu XML.
func parseVoteMenu(data []byte, congress, session int) (*VoteMenu, error) {
	var xmlMenu xmlVoteMenu
	if err := xml.Unmarshal(data, &xmlMenu); err != nil {
		return nil, fmt.Errorf("failed to parse vote menu XML: %w", err)
	}

	menu := &VoteMenu{
		Congress: congress,
		Session:  session,
		Votes:    make([]VoteMenuEntry, 0, len(xmlMenu.Votes)),
	}

	for _, v := range xmlMenu.Votes {
		voteNum, _ := strconv.Atoi(strings.TrimSpace(v.VoteNumber))
		voteDate := parseVoteDate(v.VoteDate)

		entry := VoteMenuEntry{
			VoteNumber: voteNum,
			VoteDate:   voteDate,
			Issue:      strings.TrimSpace(v.Issue),
			Question:   strings.TrimSpace(v.Question),
			Result:     strings.TrimSpace(v.Result),
			VoteTitle:  strings.TrimSpace(v.VoteTitle),
		}
		menu.Votes = append(menu.Votes, entry)
	}

	return menu, nil
}

// parseRollCallVote parses roll call vote XML.
func parseRollCallVote(data []byte) (*RollCallVote, error) {
	var xmlVote xmlRollCallVote
	if err := xml.Unmarshal(data, &xmlVote); err != nil {
		return nil, fmt.Errorf("failed to parse roll call vote XML: %w", err)
	}

	voteNum, _ := strconv.Atoi(strings.TrimSpace(xmlVote.VoteNumber))

	vote := &RollCallVote{
		Congress:     xmlVote.Congress,
		Session:      xmlVote.Session,
		VoteNumber:   voteNum,
		VoteDate:     parseVoteDate(xmlVote.VoteDate),
		ModifyDate:   parseVoteDate(xmlVote.ModifyDate),
		Question:     strings.TrimSpace(xmlVote.Question),
		QuestionText: strings.TrimSpace(xmlVote.VoteQuestionText),
		DocumentText: strings.TrimSpace(xmlVote.VoteDocumentText),
		Result:       strings.TrimSpace(xmlVote.VoteResult),
		ResultText:   strings.TrimSpace(xmlVote.VoteResultText),
		MajorityReq:  strings.TrimSpace(xmlVote.MajorityReq),
		VoteTitle:    strings.TrimSpace(xmlVote.VoteTitle),
		Count: VoteCount{
			Yeas:    xmlVote.Count.Yeas,
			Nays:    xmlVote.Count.Nays,
			Present: xmlVote.Count.Present,
			Absent:  xmlVote.Count.Absent,
		},
		Members: make([]MemberVote, 0, len(xmlVote.Members.Members)),
	}

	// Parse document if present
	if xmlVote.Document.DocType != "" || xmlVote.Document.DocNumber != "" {
		vote.Document = &VoteDocument{
			Type:   strings.TrimSpace(xmlVote.Document.DocType),
			Number: strings.TrimSpace(xmlVote.Document.DocNumber),
			Title:  strings.TrimSpace(xmlVote.Document.DocTitle),
		}
	}

	// Parse members
	for _, m := range xmlVote.Members.Members {
		member := MemberVote{
			FullName:    strings.TrimSpace(m.MemberFull),
			LastName:    strings.TrimSpace(m.LastName),
			FirstName:   strings.TrimSpace(m.FirstName),
			Party:       strings.TrimSpace(m.Party),
			State:       strings.TrimSpace(m.State),
			Vote:        ParseVotePosition(m.VoteCast),
			LISMemberID: strings.TrimSpace(m.LISMemberID),
		}
		vote.Members = append(vote.Members, member)
	}

	return vote, nil
}

// parseVoteDate parses date strings from Senate XML.
// Formats: "January 3, 2025, 12:05 PM" or "January 3, 2025"
func parseVoteDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}

	// Try full format with time
	formats := []string{
		"January 2, 2006, 3:04 PM",
		"January 2, 2006, 03:04 PM",
		"January 2, 2006",
		"Jan 2, 2006",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}

	return time.Time{}
}

// VoteToTriples converts a RollCallVote to RDF triples.
func VoteToTriples(vote *RollCallVote, baseURI string) []store.Triple {
	triples := make([]store.Triple, 0)

	// Vote URI: senate:vote/{congress}/{session}/{number}
	voteURI := fmt.Sprintf("%svote/%d/%d/%d", baseURI, vote.Congress, vote.Session, vote.VoteNumber)

	// Vote type and metadata
	triples = append(triples, store.Triple{Subject: voteURI, Predicate: "rdf:type", Object: store.ClassVoteRecord})
	triples = append(triples, store.Triple{Subject: voteURI, Predicate: store.RDFSLabel, Object: vote.VoteTitle})
	triples = append(triples, store.Triple{Subject: voteURI, Predicate: "reg:congress", Object: fmt.Sprintf("%d", vote.Congress)})
	triples = append(triples, store.Triple{Subject: voteURI, Predicate: "reg:session", Object: fmt.Sprintf("%d", vote.Session)})
	triples = append(triples, store.Triple{Subject: voteURI, Predicate: "reg:voteNumber", Object: fmt.Sprintf("%d", vote.VoteNumber)})

	if !vote.VoteDate.IsZero() {
		triples = append(triples, store.Triple{Subject: voteURI, Predicate: store.PropVoteDate, Object: vote.VoteDate.Format(time.RFC3339)})
	}

	triples = append(triples, store.Triple{Subject: voteURI, Predicate: store.PropVoteQuestion, Object: vote.Question})
	if vote.QuestionText != "" {
		triples = append(triples, store.Triple{Subject: voteURI, Predicate: store.PropText, Object: vote.QuestionText})
	}
	triples = append(triples, store.Triple{Subject: voteURI, Predicate: store.PropVoteResult, Object: vote.Result})
	if vote.ResultText != "" {
		triples = append(triples, store.Triple{Subject: voteURI, Predicate: "reg:resultText", Object: vote.ResultText})
	}
	if vote.MajorityReq != "" {
		triples = append(triples, store.Triple{Subject: voteURI, Predicate: store.PropMajorityRequired, Object: vote.MajorityReq})
	}

	// Vote counts
	triples = append(triples, store.Triple{Subject: voteURI, Predicate: store.PropVoteFor, Object: fmt.Sprintf("%d", vote.Count.Yeas)})
	triples = append(triples, store.Triple{Subject: voteURI, Predicate: store.PropVoteAgainst, Object: fmt.Sprintf("%d", vote.Count.Nays)})
	triples = append(triples, store.Triple{Subject: voteURI, Predicate: store.PropVoteAbstain, Object: fmt.Sprintf("%d", vote.Count.Present)})
	triples = append(triples, store.Triple{Subject: voteURI, Predicate: store.PropVoteAbsent, Object: fmt.Sprintf("%d", vote.Count.Absent)})

	// Related document
	if vote.Document != nil && (vote.Document.Type != "" || vote.Document.Number != "") {
		docURI := fmt.Sprintf("%sdocument/%s/%s", baseURI, vote.Document.Type, vote.Document.Number)
		triples = append(triples, store.Triple{Subject: voteURI, Predicate: "reg:relatedDocument", Object: docURI})
		triples = append(triples, store.Triple{Subject: docURI, Predicate: "rdf:type", Object: "reg:LegislativeDocument"})
		triples = append(triples, store.Triple{Subject: docURI, Predicate: "reg:documentType", Object: vote.Document.Type})
		triples = append(triples, store.Triple{Subject: docURI, Predicate: store.PropNumber, Object: vote.Document.Number})
		if vote.Document.Title != "" {
			triples = append(triples, store.Triple{Subject: docURI, Predicate: store.RDFSLabel, Object: vote.Document.Title})
		}
	}

	// Individual member votes
	for _, member := range vote.Members {
		memberURI := fmt.Sprintf("%smember/%s", baseURI, member.LISMemberID)
		indVoteURI := fmt.Sprintf("%s/%s", voteURI, sanitizeURI(member.LastName))

		// Individual vote record
		triples = append(triples, store.Triple{Subject: indVoteURI, Predicate: "rdf:type", Object: store.ClassIndividualVote})
		triples = append(triples, store.Triple{Subject: indVoteURI, Predicate: store.PropVoter, Object: memberURI})
		triples = append(triples, store.Triple{Subject: indVoteURI, Predicate: store.PropOnVote, Object: voteURI})
		triples = append(triples, store.Triple{Subject: indVoteURI, Predicate: store.PropVotePosition, Object: string(member.Vote)})

		// Senator entity
		triples = append(triples, store.Triple{Subject: memberURI, Predicate: "rdf:type", Object: "reg:Legislator"})
		triples = append(triples, store.Triple{Subject: memberURI, Predicate: store.RDFSLabel, Object: fmt.Sprintf("%s %s", member.FirstName, member.LastName)})
		triples = append(triples, store.Triple{Subject: memberURI, Predicate: "reg:firstName", Object: member.FirstName})
		triples = append(triples, store.Triple{Subject: memberURI, Predicate: "reg:lastName", Object: member.LastName})
		triples = append(triples, store.Triple{Subject: memberURI, Predicate: "reg:party", Object: member.Party})
		triples = append(triples, store.Triple{Subject: memberURI, Predicate: "reg:state", Object: member.State})
		triples = append(triples, store.Triple{Subject: memberURI, Predicate: "reg:lisMemberID", Object: member.LISMemberID})
	}

	return triples
}

// sanitizeURI removes invalid characters from URI components.
func sanitizeURI(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(s, "")
	return s
}

// IngestVotes ingests votes into a triple store.
func IngestVotes(votes []*RollCallVote, tripleStore *store.TripleStore, baseURI string) error {
	if tripleStore == nil {
		return fmt.Errorf("triple store is nil")
	}

	for _, vote := range votes {
		triples := VoteToTriples(vote, baseURI)
		if err := tripleStore.BulkAdd(triples); err != nil {
			return fmt.Errorf("failed to ingest vote %d-%d-%d: %w",
				vote.Congress, vote.Session, vote.VoteNumber, err)
		}
	}

	return nil
}
