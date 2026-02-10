// Package deliberation provides types and functions for modeling deliberation
// documents including meetings, agendas, decisions, and their evolution over time.
package deliberation

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// VotingPatternAnalyzer analyzes voting records across multiple meetings
// to identify patterns, coalitions, and voting behavior trends.
type VotingPatternAnalyzer struct {
	// patterns for extracting vote information from text
	patterns *votePatterns

	// store for persisting vote records and analysis results
	store *store.TripleStore

	// baseURI for generating vote URIs
	baseURI string

	// voteCounter for generating unique IDs
	voteCounter int
}

// votePatterns contains compiled regex patterns for vote extraction.
type votePatterns struct {
	// Roll call vote patterns
	rollCallPatterns []*regexp.Regexp

	// Voice vote patterns
	voiceVotePatterns []*regexp.Regexp

	// Vote tally patterns (for/against/abstain)
	tallyPatterns []*regexp.Regexp

	// Individual vote position patterns
	positionPatterns []*regexp.Regexp

	// Stakeholder/voter extraction patterns
	voterPatterns []*regexp.Regexp

	// Outcome patterns (adopted, rejected, etc.)
	outcomePatterns []*regexp.Regexp

	// Explanation of vote patterns
	explanationPatterns []*regexp.Regexp
}

// NewVotingPatternAnalyzer creates a new voting pattern analyzer.
func NewVotingPatternAnalyzer(tripleStore *store.TripleStore, baseURI string) *VotingPatternAnalyzer {
	return &VotingPatternAnalyzer{
		patterns:    compileVotePatterns(),
		store:       tripleStore,
		baseURI:     baseURI,
		voteCounter: 0,
	}
}

// compileVotePatterns creates the pattern set for vote extraction.
func compileVotePatterns() *votePatterns {
	return &votePatterns{
		// Roll call vote patterns
		rollCallPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:roll\s*call|recorded|nominal)\s*vote`),
			regexp.MustCompile(`(?i)vote\s*(?:by|on)\s*(?:roll\s*call|name)`),
			regexp.MustCompile(`(?i)(?:the\s+)?vote\s+was\s+taken\s+by\s+(?:roll\s*call|name)`),
		},

		// Voice vote patterns
		voiceVotePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:voice|viva\s*voce|show\s*of\s*hands)\s*vote`),
			regexp.MustCompile(`(?i)vote(?:d)?\s+by\s+(?:voice|show\s*of\s*hands|acclamation)`),
			regexp.MustCompile(`(?i)(?:adopted|rejected)\s+(?:by\s+)?(?:acclamation|consensus|unanimously)`),
			regexp.MustCompile(`(?i)without\s+(?:a\s+)?vote`),
		},

		// Vote tally patterns
		tallyPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(\d+)\s*(?:for|in\s+favour|yes)[,\s]+(\d+)\s*(?:against|no)[,\s]+(\d+)\s*(?:abstain|abstentions?)`),
			regexp.MustCompile(`(?i)(?:for|yes)[:\s]+(\d+)[,\s]+(?:against|no)[:\s]+(\d+)[,\s]+(?:abstain(?:ing|tions?)?)[:\s]+(\d+)`),
			regexp.MustCompile(`(?i)(\d+)\s*(?:to|–|-)\s*(\d+)(?:\s*(?:with|,)\s*(\d+)\s*abstentions?)?`),
			regexp.MustCompile(`(?i)(?:vote|result)[:\s]+(\d+)\s*/\s*(\d+)\s*/\s*(\d+)`),
		},

		// Individual vote position patterns
		positionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)([A-Z][a-zA-Z\s.'-]+?):\s*(for|against|yes|no|abstain|not\s*voting|absent)`),
			regexp.MustCompile(`(?i)(for|against|yes|no|abstain)[:\s]+([A-Z][a-zA-Z\s,'-]+)`),
			regexp.MustCompile(`(?i)([A-Z][a-zA-Z\s.'-]+?)\s+(?:voted|cast\s+(?:a\s+)?vote)\s+(for|against|in\s+favour|yes|no|abstain)`),
		},

		// Voter extraction patterns
		voterPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^([A-Z][a-zA-Z\s.'-]+?)(?:\s*\([^)]+\))?$`),
			regexp.MustCompile(`(?i)(?:Mr|Ms|Mrs|Dr|Prof)\.?\s+([A-Z][a-zA-Z\s.'-]+)`),
			regexp.MustCompile(`(?i)(?:The\s+)?(?:representative|delegate|delegation)\s+(?:of|from)\s+([A-Z][a-zA-Z\s.'-]+)`),
			regexp.MustCompile(`(?i)([A-Z]{2,}(?:\s+[A-Z]+)*)`), // EU-style country codes
		},

		// Outcome patterns
		outcomePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:was\s+)?(?:duly\s+)?adopted(?:\s+(?:by|with))?`),
			regexp.MustCompile(`(?i)(?:was\s+)?(?:duly\s+)?rejected`),
			regexp.MustCompile(`(?i)(?:motion|proposal|amendment)\s+(?:was\s+)?(?:carried|lost|passed|defeated)`),
			regexp.MustCompile(`(?i)(?:passed|carried|adopted)\s+(?:with|by)\s+(\d+)\s+(?:votes?|majority)`),
		},

		// Explanation of vote patterns
		explanationPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)explanation\s+of\s+vote[:\s]+(.+?)(?:\n\n|$)`),
			regexp.MustCompile(`(?i)([A-Z][a-zA-Z\s.'-]+?)\s+(?:explained|stated)\s+(?:that\s+)?(?:(?:they|he|she)\s+)?voted\s+(for|against)\s+because\s+(.+?)(?:\.|$)`),
			regexp.MustCompile(`(?i)(?:voting|voted)\s+(for|against)[,\s]+([A-Z][a-zA-Z\s.'-]+?)\s+(?:noted|explained|stated)\s+(.+?)(?:\.|$)`),
		},
	}
}

// ExtractedVote represents a vote record extracted from meeting text.
type ExtractedVote struct {
	// Subject is what was voted on
	Subject string

	// VoteType is roll_call, voice, show_of_hands, etc.
	VoteType string

	// Outcome is adopted, rejected, etc.
	Outcome string

	// ForCount is the number of votes in favor
	ForCount int

	// AgainstCount is the number of votes against
	AgainstCount int

	// AbstainCount is the number of abstentions
	AbstainCount int

	// AbsentCount is the number of absent/not voting
	AbsentCount int

	// IndividualVotes contains per-voter positions (for roll calls)
	IndividualVotes []ExtractedIndividualVote

	// SourceOffset is where in the text this vote was found
	SourceOffset int
}

// ExtractedIndividualVote represents a single voter's position.
type ExtractedIndividualVote struct {
	// VoterName is the name of the voter
	VoterName string

	// Position is for, against, abstain, or absent
	Position VotePosition

	// Explanation is optional explanation of vote
	Explanation string
}

// Coalition represents a group of stakeholders that frequently vote together.
type Coalition struct {
	// Members lists the stakeholder URIs in the coalition
	Members []string

	// MemberNames lists the human-readable names
	MemberNames []string

	// AgreementRate is the percentage of votes where members voted the same
	AgreementRate float64

	// SharedVotes is the number of votes where all members voted the same
	SharedVotes int

	// TotalVotes is the total number of votes considered
	TotalVotes int

	// Topics lists subject areas where the coalition votes together
	Topics []string
}

// VotingProfile represents a stakeholder's voting behavior.
type VotingProfile struct {
	// StakeholderURI is the unique identifier
	StakeholderURI string

	// StakeholderName is the human-readable name
	StakeholderName string

	// TotalVotes is the number of votes cast
	TotalVotes int

	// ForVotes is the number of votes in favor
	ForVotes int

	// AgainstVotes is the number of votes against
	AgainstVotes int

	// AbstainVotes is the number of abstentions
	AbstainVotes int

	// AbsentVotes is the number of times absent/not voting
	AbsentVotes int

	// Coalitions lists coalitions this stakeholder belongs to
	Coalitions []string

	// TopicsFor lists topics consistently voted for
	TopicsFor []string

	// TopicsAgainst lists topics consistently voted against
	TopicsAgainst []string

	// PositionChanges tracks topics where position changed over time
	PositionChanges []PositionChange
}

// PositionChange represents a change in voting position on a topic.
type PositionChange struct {
	// Topic is the subject area
	Topic string

	// PreviousPosition was the earlier stance
	PreviousPosition VotePosition

	// NewPosition is the current stance
	NewPosition VotePosition

	// MeetingURI is where the change occurred
	MeetingURI string

	// Date is when the change occurred
	Date time.Time
}

// SwingVoter represents a stakeholder whose positions vary.
type SwingVoter struct {
	// StakeholderURI is the unique identifier
	StakeholderURI string

	// StakeholderName is the human-readable name
	StakeholderName string

	// VariabilityScore measures how often positions change (0-1)
	VariabilityScore float64

	// TopicsVaried lists topics with varied positions
	TopicsVaried []string

	// VotingHistory shows chronological voting record
	VotingHistory []VoteHistoryEntry
}

// VoteHistoryEntry represents a single vote in a stakeholder's history.
type VoteHistoryEntry struct {
	// MeetingURI is where the vote occurred
	MeetingURI string

	// Date is when the vote occurred
	Date time.Time

	// Subject is what was voted on
	Subject string

	// Topic is the subject category
	Topic string

	// Position is how they voted
	Position VotePosition
}

// ConsistentOpponent represents a stakeholder that consistently opposes certain topics.
type ConsistentOpponent struct {
	// StakeholderURI is the unique identifier
	StakeholderURI string

	// StakeholderName is the human-readable name
	StakeholderName string

	// OpposedTopics lists topics consistently opposed
	OpposedTopics []TopicOpposition
}

// TopicOpposition represents consistent opposition to a topic.
type TopicOpposition struct {
	// Topic is the subject area
	Topic string

	// OppositionRate is the percentage of votes against (0-1)
	OppositionRate float64

	// TotalVotes is the number of votes on this topic
	TotalVotes int

	// AgainstVotes is the number of votes against
	AgainstVotes int
}

// VotingPatternAnalysis contains the complete analysis results.
type VotingPatternAnalysis struct {
	// Coalitions lists detected voting coalitions
	Coalitions []Coalition

	// SwingVoters lists stakeholders with variable positions
	SwingVoters []SwingVoter

	// ConsistentOpponents lists stakeholders that consistently oppose certain topics
	ConsistentOpponents []ConsistentOpponent

	// TopicClusters groups votes by subject matter
	TopicClusters []TopicCluster

	// VoterProfiles contains detailed profiles for each stakeholder
	VoterProfiles map[string]*VotingProfile

	// Summary provides high-level statistics
	Summary VotingPatternSummary
}

// TopicCluster groups votes by subject matter.
type TopicCluster struct {
	// Topic is the subject area name
	Topic string

	// Votes lists vote URIs on this topic
	Votes []string

	// TotalVotes is the count of votes
	TotalVotes int

	// AdoptedCount is how many were adopted
	AdoptedCount int

	// RejectedCount is how many were rejected
	RejectedCount int

	// ControversialVotes lists close/contested votes
	ControversialVotes []string
}

// VotingPatternSummary provides high-level voting statistics.
type VotingPatternSummary struct {
	// TotalVotes is the number of votes analyzed
	TotalVotes int

	// TotalVoters is the number of unique voters
	TotalVoters int

	// CoalitionCount is the number of detected coalitions
	CoalitionCount int

	// SwingVoterCount is the number of swing voters identified
	SwingVoterCount int

	// MostActiveVoter is the stakeholder with most votes cast
	MostActiveVoter string

	// HighestAgreementPair lists the two voters with highest agreement rate
	HighestAgreementPair []string

	// MostControversialTopic is the topic with most split votes
	MostControversialTopic string
}

// ExtractVotes extracts vote records from meeting text.
func (v *VotingPatternAnalyzer) ExtractVotes(text string, meetingURI string, meetingDate time.Time) ([]ExtractedVote, error) {
	var votes []ExtractedVote

	// Detect vote type
	voteType := v.detectVoteType(text)

	// Extract tallies
	for _, pattern := range v.patterns.tallyPatterns {
		matches := pattern.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			if len(match) >= 8 {
				vote := ExtractedVote{
					VoteType:     voteType,
					SourceOffset: match[0],
				}

				// Parse counts
				forStr := text[match[2]:match[3]]
				againstStr := text[match[4]:match[5]]
				abstainStr := text[match[6]:match[7]]

				vote.ForCount = parseVoteCount(forStr)
				vote.AgainstCount = parseVoteCount(againstStr)
				vote.AbstainCount = parseVoteCount(abstainStr)

				// Extract subject from context
				vote.Subject = v.extractVoteSubject(text, match[0])

				// Determine outcome
				vote.Outcome = v.determineOutcome(vote.ForCount, vote.AgainstCount, text[voteMaxInt(0, match[0]-200):voteMinInt(len(text), match[1]+200)])

				votes = append(votes, vote)
			}
		}
	}

	// Extract individual votes for roll calls
	if voteType == "roll_call" {
		individualVotes := v.extractIndividualVotes(text)
		if len(votes) > 0 && len(individualVotes) > 0 {
			votes[0].IndividualVotes = individualVotes
		} else if len(individualVotes) > 0 {
			// Create a vote record from individual votes
			vote := ExtractedVote{
				VoteType:        "roll_call",
				IndividualVotes: individualVotes,
			}
			vote.ForCount, vote.AgainstCount, vote.AbstainCount, vote.AbsentCount = countIndividualVotes(individualVotes)
			votes = append(votes, vote)
		}
	}

	return votes, nil
}

// detectVoteType determines the type of vote from text.
func (v *VotingPatternAnalyzer) detectVoteType(text string) string {
	for _, pattern := range v.patterns.rollCallPatterns {
		if pattern.MatchString(text) {
			return "roll_call"
		}
	}
	for _, pattern := range v.patterns.voiceVotePatterns {
		if pattern.MatchString(text) {
			return "voice"
		}
	}
	return "unknown"
}

// extractVoteSubject extracts what was being voted on from surrounding context.
func (v *VotingPatternAnalyzer) extractVoteSubject(text string, offset int) string {
	// Look back for subject indicators
	start := voteMaxInt(0, offset-500)
	context := text[start:offset]

	// Common patterns for vote subjects
	subjectPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:vote|voting)\s+on\s+(.+?)(?:\.|:|\n)`),
		regexp.MustCompile(`(?i)(?:motion|proposal|amendment)\s+(?:to\s+)?(.+?)(?:\.|:|\n)`),
		regexp.MustCompile(`(?i)(?:put|submitted)\s+to\s+(?:a\s+)?vote[:\s]+(.+?)(?:\.|$)`),
		regexp.MustCompile(`(?i)(?:Article|Section|Paragraph)\s+\d+[^.\n]*`),
	}

	for _, pattern := range subjectPatterns {
		if match := pattern.FindStringSubmatch(context); len(match) > 1 {
			return strings.TrimSpace(match[1])
		}
	}

	return ""
}

// determineOutcome determines the vote outcome from counts and context.
func (v *VotingPatternAnalyzer) determineOutcome(forCount, againstCount int, context string) string {
	// Check for explicit outcome in text
	for _, pattern := range v.patterns.outcomePatterns {
		if match := pattern.FindString(context); match != "" {
			lower := strings.ToLower(match)
			if strings.Contains(lower, "adopted") || strings.Contains(lower, "passed") || strings.Contains(lower, "carried") {
				return "adopted"
			}
			if strings.Contains(lower, "rejected") || strings.Contains(lower, "lost") || strings.Contains(lower, "defeated") {
				return "rejected"
			}
		}
	}

	// Infer from counts
	if forCount > againstCount {
		return "adopted"
	} else if againstCount > forCount {
		return "rejected"
	}
	return "unknown"
}

// extractIndividualVotes extracts per-voter positions from roll call text.
func (v *VotingPatternAnalyzer) extractIndividualVotes(text string) []ExtractedIndividualVote {
	var votes []ExtractedIndividualVote

	for _, pattern := range v.patterns.positionPatterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				var voter, posStr string
				// Pattern may capture voter first or position first
				if isPositionString(match[1]) {
					posStr = match[1]
					voter = match[2]
				} else {
					voter = match[1]
					posStr = match[2]
				}

				// Parse multiple voters separated by commas
				voters := splitVoters(voter)
				position := parsePosition(posStr)

				for _, voterName := range voters {
					votes = append(votes, ExtractedIndividualVote{
						VoterName: strings.TrimSpace(voterName),
						Position:  position,
					})
				}
			}
		}
	}

	// Extract explanations
	votes = v.extractExplanations(text, votes)

	return votes
}

// extractExplanations adds explanations of vote to individual votes.
func (v *VotingPatternAnalyzer) extractExplanations(text string, votes []ExtractedIndividualVote) []ExtractedIndividualVote {
	for _, pattern := range v.patterns.explanationPatterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 4 {
				voter := strings.TrimSpace(match[1])
				explanation := strings.TrimSpace(match[3])

				// Find matching vote and add explanation
				for i := range votes {
					if strings.EqualFold(votes[i].VoterName, voter) {
						votes[i].Explanation = explanation
						break
					}
				}
			}
		}
	}
	return votes
}

// CreateVoteRecord creates a VoteRecord from extracted data.
func (v *VotingPatternAnalyzer) CreateVoteRecord(extracted ExtractedVote, meetingURI string, voteDate time.Time) *VoteRecord {
	v.voteCounter++
	uri := fmt.Sprintf("%svote:%d", v.baseURI, v.voteCounter)

	record := &VoteRecord{
		URI:          uri,
		VoteDate:     voteDate,
		VoteType:     extracted.VoteType,
		Question:     extracted.Subject,
		Result:       extracted.Outcome,
		ForCount:     extracted.ForCount,
		AgainstCount: extracted.AgainstCount,
		AbstainCount: extracted.AbstainCount,
		AbsentCount:  extracted.AbsentCount,
		MeetingURI:   meetingURI,
	}

	// Convert individual votes
	for _, ev := range extracted.IndividualVotes {
		record.IndividualVotes = append(record.IndividualVotes, IndividualVote{
			VoterName:   ev.VoterName,
			Position:    ev.Position,
			Explanation: ev.Explanation,
		})
	}

	return record
}

// PersistVoteRecord saves a vote record to the triple store.
func (v *VotingPatternAnalyzer) PersistVoteRecord(record *VoteRecord) error {
	if v.store == nil {
		return fmt.Errorf("no triple store configured")
	}

	// Add vote record triples
	v.store.Add(record.URI, store.RDFType, store.ClassVoteRecord)
	v.store.Add(record.URI, store.PropVoteDate, record.VoteDate.Format(time.RFC3339))
	v.store.Add(record.URI, store.PropVoteType, record.VoteType)
	v.store.Add(record.URI, store.PropVoteQuestion, record.Question)
	v.store.Add(record.URI, store.PropVoteResult, record.Result)
	v.store.Add(record.URI, store.PropVoteFor, fmt.Sprintf("%d", record.ForCount))
	v.store.Add(record.URI, store.PropVoteAgainst, fmt.Sprintf("%d", record.AgainstCount))
	v.store.Add(record.URI, store.PropVoteAbstain, fmt.Sprintf("%d", record.AbstainCount))
	v.store.Add(record.URI, store.PropVoteAbsent, fmt.Sprintf("%d", record.AbsentCount))
	v.store.Add(record.URI, store.PropPartOf, record.MeetingURI)

	// Add individual vote triples
	for i, iv := range record.IndividualVotes {
		ivURI := fmt.Sprintf("%s:individual:%d", record.URI, i)
		v.store.Add(ivURI, store.RDFType, store.ClassIndividualVote)
		v.store.Add(ivURI, store.PropOnVote, record.URI)
		v.store.Add(ivURI, store.RDFSLabel, iv.VoterName)
		v.store.Add(ivURI, store.PropVotePosition, iv.Position.String())
		if iv.Explanation != "" {
			v.store.Add(ivURI, store.PropVoteExplanation, iv.Explanation)
		}
		if iv.VoterURI != "" {
			v.store.Add(ivURI, store.PropVoter, iv.VoterURI)
		}
	}

	return nil
}

// AnalyzePatterns performs comprehensive voting pattern analysis.
func (v *VotingPatternAnalyzer) AnalyzePatterns(minAgreementThreshold float64) (*VotingPatternAnalysis, error) {
	if v.store == nil {
		return nil, fmt.Errorf("no triple store configured")
	}

	analysis := &VotingPatternAnalysis{
		VoterProfiles: make(map[string]*VotingProfile),
	}

	// Load all vote records
	voteRecords, err := v.loadVoteRecords()
	if err != nil {
		return nil, err
	}

	// Build voter profiles
	analysis.VoterProfiles = v.buildVoterProfiles(voteRecords)

	// Detect coalitions
	analysis.Coalitions = v.detectCoalitions(voteRecords, minAgreementThreshold)

	// Identify swing voters
	analysis.SwingVoters = v.identifySwingVoters(voteRecords)

	// Find consistent opponents
	analysis.ConsistentOpponents = v.findConsistentOpponents(voteRecords)

	// Cluster by topic
	analysis.TopicClusters = v.clusterByTopic(voteRecords)

	// Generate summary
	analysis.Summary = v.generateSummary(analysis)

	return analysis, nil
}

// loadVoteRecords retrieves all vote records from the triple store.
func (v *VotingPatternAnalyzer) loadVoteRecords() ([]*VoteRecord, error) {
	var records []*VoteRecord

	// Find all vote records
	voteTriples := v.store.Find("", store.RDFType, store.ClassVoteRecord)
	for _, triple := range voteTriples {
		voteURI := triple.Subject
		record := &VoteRecord{URI: voteURI}

		// Load vote properties
		if props := v.store.Find(voteURI, store.PropVoteDate, ""); len(props) > 0 {
			if t, err := time.Parse(time.RFC3339, props[0].Object); err == nil {
				record.VoteDate = t
			}
		}
		if props := v.store.Find(voteURI, store.PropVoteType, ""); len(props) > 0 {
			record.VoteType = props[0].Object
		}
		if props := v.store.Find(voteURI, store.PropVoteQuestion, ""); len(props) > 0 {
			record.Question = props[0].Object
		}
		if props := v.store.Find(voteURI, store.PropVoteResult, ""); len(props) > 0 {
			record.Result = props[0].Object
		}
		if props := v.store.Find(voteURI, store.PropVoteFor, ""); len(props) > 0 {
			record.ForCount = parseVoteCount(props[0].Object)
		}
		if props := v.store.Find(voteURI, store.PropVoteAgainst, ""); len(props) > 0 {
			record.AgainstCount = parseVoteCount(props[0].Object)
		}
		if props := v.store.Find(voteURI, store.PropVoteAbstain, ""); len(props) > 0 {
			record.AbstainCount = parseVoteCount(props[0].Object)
		}
		if props := v.store.Find(voteURI, store.PropPartOf, ""); len(props) > 0 {
			record.MeetingURI = props[0].Object
		}

		// Load individual votes
		individualTriples := v.store.Find("", store.PropOnVote, voteURI)
		for _, ivTriple := range individualTriples {
			iv := IndividualVote{}
			ivURI := ivTriple.Subject

			if props := v.store.Find(ivURI, store.RDFSLabel, ""); len(props) > 0 {
				iv.VoterName = props[0].Object
			}
			if props := v.store.Find(ivURI, store.PropVoter, ""); len(props) > 0 {
				iv.VoterURI = props[0].Object
			}
			if props := v.store.Find(ivURI, store.PropVotePosition, ""); len(props) > 0 {
				iv.Position = parsePosition(props[0].Object)
			}
			if props := v.store.Find(ivURI, store.PropVoteExplanation, ""); len(props) > 0 {
				iv.Explanation = props[0].Object
			}

			record.IndividualVotes = append(record.IndividualVotes, iv)
		}

		records = append(records, record)
	}

	// Sort by date
	sort.Slice(records, func(i, j int) bool {
		return records[i].VoteDate.Before(records[j].VoteDate)
	})

	return records, nil
}

// buildVoterProfiles creates profiles for each voter.
func (v *VotingPatternAnalyzer) buildVoterProfiles(records []*VoteRecord) map[string]*VotingProfile {
	profiles := make(map[string]*VotingProfile)

	for _, record := range records {
		for _, iv := range record.IndividualVotes {
			// Use voter name as key if no URI
			key := iv.VoterURI
			if key == "" {
				key = normalizeVoterName(iv.VoterName)
			}

			profile, exists := profiles[key]
			if !exists {
				profile = &VotingProfile{
					StakeholderURI:  iv.VoterURI,
					StakeholderName: iv.VoterName,
				}
				profiles[key] = profile
			}

			profile.TotalVotes++
			switch iv.Position {
			case VoteFor:
				profile.ForVotes++
			case VoteAgainst:
				profile.AgainstVotes++
			case VoteAbstain:
				profile.AbstainVotes++
			case VoteAbsent:
				profile.AbsentVotes++
			}
		}
	}

	return profiles
}

// detectCoalitions identifies groups of voters that frequently vote together.
func (v *VotingPatternAnalyzer) detectCoalitions(records []*VoteRecord, threshold float64) []Coalition {
	// Build voter agreement matrix
	voterAgreement := make(map[string]map[string]int)
	voterVotes := make(map[string]int)

	for _, record := range records {
		// Build a map of how each voter voted in this record
		voteMap := make(map[string]VotePosition)
		for _, iv := range record.IndividualVotes {
			key := normalizeVoterName(iv.VoterName)
			voteMap[key] = iv.Position
			voterVotes[key]++
		}

		// Count agreements
		for voter1, pos1 := range voteMap {
			if voterAgreement[voter1] == nil {
				voterAgreement[voter1] = make(map[string]int)
			}
			for voter2, pos2 := range voteMap {
				if voter1 != voter2 && pos1 == pos2 {
					voterAgreement[voter1][voter2]++
				}
			}
		}
	}

	// Find coalitions based on agreement threshold
	var coalitions []Coalition
	processedPairs := make(map[string]bool)

	for voter1, agreements := range voterAgreement {
		for voter2, sharedCount := range agreements {
			pairKey := makePairKey(voter1, voter2)
			if processedPairs[pairKey] {
				continue
			}
			processedPairs[pairKey] = true

			// Calculate agreement rate
			minVotes := voteMinInt(voterVotes[voter1], voterVotes[voter2])
			if minVotes < 5 { // Require minimum 5 shared votes
				continue
			}

			agreementRate := float64(sharedCount) / float64(minVotes)
			if agreementRate >= threshold {
				coalition := Coalition{
					Members:       []string{voter1, voter2},
					MemberNames:   []string{voter1, voter2},
					AgreementRate: agreementRate,
					SharedVotes:   sharedCount,
					TotalVotes:    minVotes,
				}
				coalitions = append(coalitions, coalition)
			}
		}
	}

	// Sort by agreement rate
	sort.Slice(coalitions, func(i, j int) bool {
		return coalitions[i].AgreementRate > coalitions[j].AgreementRate
	})

	return coalitions
}

// identifySwingVoters finds voters with variable positions.
func (v *VotingPatternAnalyzer) identifySwingVoters(records []*VoteRecord) []SwingVoter {
	// Track position changes by topic
	voterTopicPositions := make(map[string]map[string][]VotePosition)

	for _, record := range records {
		topic := extractTopic(record.Question)
		for _, iv := range record.IndividualVotes {
			voter := normalizeVoterName(iv.VoterName)
			if voterTopicPositions[voter] == nil {
				voterTopicPositions[voter] = make(map[string][]VotePosition)
			}
			voterTopicPositions[voter][topic] = append(voterTopicPositions[voter][topic], iv.Position)
		}
	}

	var swingVoters []SwingVoter

	for voter, topicPositions := range voterTopicPositions {
		variableTopics := 0
		totalTopics := 0
		var topicsVaried []string

		for topic, positions := range topicPositions {
			if len(positions) < 2 {
				continue
			}
			totalTopics++

			// Check if position varied
			firstPos := positions[0]
			varied := false
			for _, pos := range positions[1:] {
				if pos != firstPos {
					varied = true
					break
				}
			}
			if varied {
				variableTopics++
				topicsVaried = append(topicsVaried, topic)
			}
		}

		if totalTopics > 0 && variableTopics > 0 {
			score := float64(variableTopics) / float64(totalTopics)
			if score > 0.2 { // At least 20% topic variability
				swingVoters = append(swingVoters, SwingVoter{
					StakeholderName:  voter,
					VariabilityScore: score,
					TopicsVaried:     topicsVaried,
				})
			}
		}
	}

	// Sort by variability score
	sort.Slice(swingVoters, func(i, j int) bool {
		return swingVoters[i].VariabilityScore > swingVoters[j].VariabilityScore
	})

	return swingVoters
}

// findConsistentOpponents identifies voters that consistently oppose certain topics.
func (v *VotingPatternAnalyzer) findConsistentOpponents(records []*VoteRecord) []ConsistentOpponent {
	// Track opposition by topic
	voterTopicOpposition := make(map[string]map[string]*TopicOpposition)

	for _, record := range records {
		topic := extractTopic(record.Question)
		for _, iv := range record.IndividualVotes {
			voter := normalizeVoterName(iv.VoterName)
			if voterTopicOpposition[voter] == nil {
				voterTopicOpposition[voter] = make(map[string]*TopicOpposition)
			}
			if voterTopicOpposition[voter][topic] == nil {
				voterTopicOpposition[voter][topic] = &TopicOpposition{Topic: topic}
			}

			voterTopicOpposition[voter][topic].TotalVotes++
			if iv.Position == VoteAgainst {
				voterTopicOpposition[voter][topic].AgainstVotes++
			}
		}
	}

	var opponents []ConsistentOpponent

	for voter, topicOppositions := range voterTopicOpposition {
		var opposedTopics []TopicOpposition

		for _, opp := range topicOppositions {
			if opp.TotalVotes >= 3 { // Minimum 3 votes on topic
				opp.OppositionRate = float64(opp.AgainstVotes) / float64(opp.TotalVotes)
				if opp.OppositionRate >= 0.8 { // 80%+ opposition rate
					opposedTopics = append(opposedTopics, *opp)
				}
			}
		}

		if len(opposedTopics) > 0 {
			opponents = append(opponents, ConsistentOpponent{
				StakeholderName: voter,
				OpposedTopics:   opposedTopics,
			})
		}
	}

	return opponents
}

// clusterByTopic groups votes by subject matter.
func (v *VotingPatternAnalyzer) clusterByTopic(records []*VoteRecord) []TopicCluster {
	topicVotes := make(map[string]*TopicCluster)

	for _, record := range records {
		topic := extractTopic(record.Question)
		if topicVotes[topic] == nil {
			topicVotes[topic] = &TopicCluster{Topic: topic}
		}

		topicVotes[topic].Votes = append(topicVotes[topic].Votes, record.URI)
		topicVotes[topic].TotalVotes++

		if record.Result == "adopted" {
			topicVotes[topic].AdoptedCount++
		} else if record.Result == "rejected" {
			topicVotes[topic].RejectedCount++
		}

		// Check if controversial (close vote)
		if isControversial(record) {
			topicVotes[topic].ControversialVotes = append(topicVotes[topic].ControversialVotes, record.URI)
		}
	}

	var clusters []TopicCluster
	for _, cluster := range topicVotes {
		clusters = append(clusters, *cluster)
	}

	// Sort by total votes
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].TotalVotes > clusters[j].TotalVotes
	})

	return clusters
}

// generateSummary creates a summary of the analysis.
func (v *VotingPatternAnalyzer) generateSummary(analysis *VotingPatternAnalysis) VotingPatternSummary {
	summary := VotingPatternSummary{
		TotalVoters:    len(analysis.VoterProfiles),
		CoalitionCount: len(analysis.Coalitions),
		SwingVoterCount: len(analysis.SwingVoters),
	}

	// Count total votes
	for _, profile := range analysis.VoterProfiles {
		summary.TotalVotes += profile.TotalVotes
	}
	summary.TotalVotes /= 2 // Avoid double counting from individual votes

	// Find most active voter
	maxVotes := 0
	for name, profile := range analysis.VoterProfiles {
		if profile.TotalVotes > maxVotes {
			maxVotes = profile.TotalVotes
			summary.MostActiveVoter = name
		}
	}

	// Find highest agreement pair
	if len(analysis.Coalitions) > 0 {
		summary.HighestAgreementPair = analysis.Coalitions[0].MemberNames
	}

	// Find most controversial topic
	maxControversial := 0
	for _, cluster := range analysis.TopicClusters {
		if len(cluster.ControversialVotes) > maxControversial {
			maxControversial = len(cluster.ControversialVotes)
			summary.MostControversialTopic = cluster.Topic
		}
	}

	return summary
}

// GetVotingHistoryByStakeholder returns the voting history for a stakeholder.
func (v *VotingPatternAnalyzer) GetVotingHistoryByStakeholder(stakeholderID string) ([]VoteHistoryEntry, error) {
	records, err := v.loadVoteRecords()
	if err != nil {
		return nil, err
	}

	var history []VoteHistoryEntry

	for _, record := range records {
		for _, iv := range record.IndividualVotes {
			if normalizeVoterName(iv.VoterName) == stakeholderID ||
				iv.VoterURI == stakeholderID {
				history = append(history, VoteHistoryEntry{
					MeetingURI: record.MeetingURI,
					Date:       record.VoteDate,
					Subject:    record.Question,
					Topic:      extractTopic(record.Question),
					Position:   iv.Position,
				})
			}
		}
	}

	// Sort by date
	sort.Slice(history, func(i, j int) bool {
		return history[i].Date.Before(history[j].Date)
	})

	return history, nil
}

// GetVotingHistoryByTopic returns votes on a specific topic.
func (v *VotingPatternAnalyzer) GetVotingHistoryByTopic(topic string) ([]*VoteRecord, error) {
	records, err := v.loadVoteRecords()
	if err != nil {
		return nil, err
	}

	var topicRecords []*VoteRecord
	normalizedTopic := strings.ToLower(topic)

	for _, record := range records {
		if strings.Contains(strings.ToLower(record.Question), normalizedTopic) ||
			strings.Contains(strings.ToLower(extractTopic(record.Question)), normalizedTopic) {
			topicRecords = append(topicRecords, record)
		}
	}

	return topicRecords, nil
}

// FindCoalition identifies the voting coalition for a stakeholder.
func (v *VotingPatternAnalyzer) FindCoalition(stakeholderID string, threshold float64) (*Coalition, error) {
	analysis, err := v.AnalyzePatterns(threshold)
	if err != nil {
		return nil, err
	}

	for _, coalition := range analysis.Coalitions {
		for _, member := range coalition.Members {
			if member == stakeholderID {
				return &coalition, nil
			}
		}
	}

	return nil, nil
}

// ExportNetworkData exports coalition data for network visualization.
func (v *VotingPatternAnalyzer) ExportNetworkData(minAgreement float64) (*NetworkVisualizationData, error) {
	analysis, err := v.AnalyzePatterns(minAgreement)
	if err != nil {
		return nil, err
	}

	data := &NetworkVisualizationData{
		Nodes: make([]NetworkNode, 0),
		Edges: make([]NetworkEdge, 0),
	}

	// Add nodes for all voters
	nodeIndex := make(map[string]int)
	i := 0
	for name, profile := range analysis.VoterProfiles {
		data.Nodes = append(data.Nodes, NetworkNode{
			ID:    name,
			Label: profile.StakeholderName,
			Size:  float64(profile.TotalVotes),
		})
		nodeIndex[name] = i
		i++
	}

	// Add edges for coalitions
	for _, coalition := range analysis.Coalitions {
		if len(coalition.Members) >= 2 {
			data.Edges = append(data.Edges, NetworkEdge{
				Source: coalition.Members[0],
				Target: coalition.Members[1],
				Weight: coalition.AgreementRate,
			})
		}
	}

	return data, nil
}

// NetworkVisualizationData represents data for coalition network graphs.
type NetworkVisualizationData struct {
	// Nodes are voters
	Nodes []NetworkNode

	// Edges are agreement relationships
	Edges []NetworkEdge
}

// NetworkNode represents a voter in the network.
type NetworkNode struct {
	ID    string
	Label string
	Size  float64
}

// NetworkEdge represents an agreement relationship.
type NetworkEdge struct {
	Source string
	Target string
	Weight float64
}

// Helper functions

func parseVoteCount(s string) int {
	var count int
	fmt.Sscanf(strings.TrimSpace(s), "%d", &count)
	return count
}

func isPositionString(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	return lower == "for" || lower == "against" || lower == "yes" || lower == "no" ||
		lower == "abstain" || lower == "not voting" || lower == "absent" ||
		lower == "in favour"
}

func parsePosition(s string) VotePosition {
	lower := strings.ToLower(strings.TrimSpace(s))
	switch {
	case lower == "for" || lower == "yes" || lower == "in favour":
		return VoteFor
	case lower == "against" || lower == "no":
		return VoteAgainst
	case lower == "abstain":
		return VoteAbstain
	case lower == "absent" || lower == "not voting":
		return VoteAbsent
	default:
		return VoteAbstain
	}
}

func splitVoters(s string) []string {
	// Split by comma, semicolon, or "and"
	parts := regexp.MustCompile(`[,;]|\s+and\s+`).Split(s, -1)
	var voters []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			voters = append(voters, part)
		}
	}
	return voters
}

func countIndividualVotes(votes []ExtractedIndividualVote) (forCount, againstCount, abstainCount, absentCount int) {
	for _, v := range votes {
		switch v.Position {
		case VoteFor:
			forCount++
		case VoteAgainst:
			againstCount++
		case VoteAbstain:
			abstainCount++
		case VoteAbsent:
			absentCount++
		}
	}
	return
}

func normalizeVoterName(name string) string {
	// Remove titles, extra whitespace, normalize case
	name = strings.TrimSpace(name)
	name = regexp.MustCompile(`(?i)^(Mr|Ms|Mrs|Dr|Prof)\.?\s+`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")
	return strings.ToLower(name)
}

func makePairKey(a, b string) string {
	if a < b {
		return a + "|" + b
	}
	return b + "|" + a
}

func extractTopic(question string) string {
	// Extract topic keywords from vote question
	if question == "" {
		return "general"
	}

	// Look for article/section references
	if match := regexp.MustCompile(`(?i)(Article|Section)\s+\d+`).FindString(question); match != "" {
		return match
	}

	// Extract first few words
	words := strings.Fields(question)
	if len(words) > 5 {
		words = words[:5]
	}
	return strings.Join(words, " ")
}

func isControversial(record *VoteRecord) bool {
	total := record.ForCount + record.AgainstCount
	if total == 0 {
		return false
	}

	// Close vote: difference less than 20% of total
	diff := voteAbsInt(record.ForCount - record.AgainstCount)
	return float64(diff)/float64(total) < 0.2
}

func voteAbsInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func voteMaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func voteMinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
