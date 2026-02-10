// Package deliberation provides speaker and stakeholder extraction and linking
// for deliberation documents, enabling participation analysis across meetings.
package deliberation

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// StakeholderType classifies the type of stakeholder.
type StakeholderType int

const (
	// StakeholderMemberState represents a member state/country.
	StakeholderMemberState StakeholderType = iota
	// StakeholderDelegation represents a delegation.
	StakeholderDelegation
	// StakeholderOrganization represents an organization.
	StakeholderOrganization
	// StakeholderPoliticalGroup represents a political group.
	StakeholderPoliticalGroup
	// StakeholderCommittee represents a committee.
	StakeholderCommittee
	// StakeholderSecretariat represents a secretariat.
	StakeholderSecretariat
	// StakeholderObserver represents an observer.
	StakeholderObserver
	// StakeholderIndividual represents an individual person.
	StakeholderIndividual
)

// String returns a human-readable label for the stakeholder type.
func (t StakeholderType) String() string {
	switch t {
	case StakeholderMemberState:
		return "member_state"
	case StakeholderDelegation:
		return "delegation"
	case StakeholderOrganization:
		return "organization"
	case StakeholderPoliticalGroup:
		return "political_group"
	case StakeholderCommittee:
		return "committee"
	case StakeholderSecretariat:
		return "secretariat"
	case StakeholderObserver:
		return "observer"
	case StakeholderIndividual:
		return "individual"
	default:
		return "unknown"
	}
}

// ParseStakeholderType converts a string to a StakeholderType.
func ParseStakeholderType(s string) StakeholderType {
	switch strings.ToLower(s) {
	case "member_state", "memberstate", "member state":
		return StakeholderMemberState
	case "delegation":
		return StakeholderDelegation
	case "organization", "org":
		return StakeholderOrganization
	case "political_group", "politicalgroup", "political group":
		return StakeholderPoliticalGroup
	case "committee":
		return StakeholderCommittee
	case "secretariat":
		return StakeholderSecretariat
	case "observer":
		return StakeholderObserver
	case "individual", "person":
		return StakeholderIndividual
	default:
		return StakeholderOrganization
	}
}

// RoleAssignment represents a role held by a speaker or stakeholder.
type RoleAssignment struct {
	// Role is the role title (e.g., "Chair", "Rapporteur").
	Role string `json:"role"`

	// StartDate is when the role began.
	StartDate *time.Time `json:"start_date,omitempty"`

	// EndDate is when the role ended.
	EndDate *time.Time `json:"end_date,omitempty"`

	// Scope indicates where the role applies (committee, working group, etc.).
	Scope string `json:"scope,omitempty"`

	// ProcessURI links to the deliberation process.
	ProcessURI string `json:"process_uri,omitempty"`
}

// Speaker represents a named individual who speaks in meetings.
type Speaker struct {
	// URI is the unique identifier for this speaker.
	URI string `json:"uri"`

	// Name is the speaker's name.
	Name string `json:"name"`

	// Aliases are alternative names/references (e.g., "Mr. Smith").
	Aliases []string `json:"aliases,omitempty"`

	// Affiliation is the stakeholder URI this speaker belongs to.
	Affiliation string `json:"affiliation,omitempty"`

	// AffiliationName is the human-readable affiliation name.
	AffiliationName string `json:"affiliation_name,omitempty"`

	// Roles are role assignments for this speaker.
	Roles []RoleAssignment `json:"roles,omitempty"`

	// MeetingsAttended are URIs of meetings this speaker attended.
	MeetingsAttended []string `json:"meetings_attended,omitempty"`

	// InterventionCount is the total number of interventions.
	InterventionCount int `json:"intervention_count,omitempty"`
}

// ExtractedStakeholder represents a stakeholder entity.
type ExtractedStakeholder struct {
	// URI is the unique identifier for this stakeholder.
	URI string `json:"uri"`

	// Name is the stakeholder's name.
	Name string `json:"name"`

	// Type classifies the stakeholder.
	Type StakeholderType `json:"type"`

	// Aliases are alternative names/references.
	Aliases []string `json:"aliases,omitempty"`

	// Members are URIs of member entities (for groups/coalitions).
	Members []string `json:"members,omitempty"`

	// ParentOrg is the URI of the parent organization.
	ParentOrg string `json:"parent_org,omitempty"`

	// Speakers are URIs of speakers affiliated with this stakeholder.
	Speakers []string `json:"speakers,omitempty"`

	// MeetingsParticipated are URIs of meetings this stakeholder participated in.
	MeetingsParticipated []string `json:"meetings_participated,omitempty"`

	// Roles are role assignments for this stakeholder.
	Roles []RoleAssignment `json:"roles,omitempty"`
}

// EntityMention represents an unresolved entity mention in text.
type EntityMention struct {
	// Text is the original mention text.
	Text string `json:"text"`

	// NormalizedText is the normalized form.
	NormalizedText string `json:"normalized_text"`

	// MeetingURI is the meeting where this mention occurred.
	MeetingURI string `json:"meeting_uri,omitempty"`

	// SourceOffset is the character offset in the source text.
	SourceOffset int `json:"source_offset,omitempty"`

	// Context is surrounding text for disambiguation.
	Context string `json:"context,omitempty"`

	// ProbableType is the inferred entity type.
	ProbableType string `json:"probable_type,omitempty"`

	// ResolvedURI is the URI if resolved, empty if unresolved.
	ResolvedURI string `json:"resolved_uri,omitempty"`

	// Confidence is the resolution confidence (0.0-1.0).
	Confidence float64 `json:"confidence,omitempty"`
}

// EntityContext provides context for entity resolution.
type EntityContext struct {
	// MeetingURI is the current meeting.
	MeetingURI string

	// ProcessURI is the deliberation process.
	ProcessURI string

	// PreviousMentions are recently resolved entities.
	PreviousMentions []string

	// TopicURI is the current topic being discussed.
	TopicURI string
}

// EntityExtractor extracts speakers and stakeholders from deliberation text.
type EntityExtractor struct {
	// store is the triple store containing deliberation data.
	store *store.TripleStore

	// baseURI is the base URI for generating entity URIs.
	baseURI string

	// patterns are compiled regex patterns for extraction.
	patterns *entityPatterns

	// resolver is the entity resolver for linking.
	resolver *EntityResolver
}

// entityPatterns contains compiled regex patterns for entity extraction.
type entityPatterns struct {
	// Speaker patterns
	speakerWithAffiliation []*regexp.Regexp
	speakerRole            []*regexp.Regexp
	speakerName            []*regexp.Regexp

	// Stakeholder patterns
	memberState     []*regexp.Regexp
	delegation      []*regexp.Regexp
	organization    []*regexp.Regexp
	roleReference   []*regexp.Regexp
	votingRecord    []*regexp.Regexp
	documentAuthor  []*regexp.Regexp
}

// NewEntityExtractor creates a new entity extractor.
func NewEntityExtractor(tripleStore *store.TripleStore, baseURI string) *EntityExtractor {
	extractor := &EntityExtractor{
		store:    tripleStore,
		baseURI:  baseURI,
		patterns: compileEntityPatterns(),
	}
	extractor.resolver = NewEntityResolver(tripleStore, baseURI)
	return extractor
}

// compileEntityPatterns creates the pattern set for entity extraction.
func compileEntityPatterns() *entityPatterns {
	return &entityPatterns{
		// Speaker with affiliation: "Mr./Ms. [Name] ([Delegation])..."
		speakerWithAffiliation: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:Mr|Ms|Mrs|Dr|Prof)\.?\s+([A-Z][a-zA-Z\s.'-]+?)\s*\(([^)]+)\)`),
			regexp.MustCompile(`(?i)([A-Z][a-zA-Z\s.'-]+?),?\s+(?:representative|delegate|ambassador)\s+(?:of|from)\s+([A-Z][a-zA-Z\s]+)`),
		},

		// Speaker role: "The Chair noted...", "The Rapporteur presented..."
		speakerRole: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:The\s+)?(Chair(?:man|woman|person)?|President|Vice[- ]?President|Rapporteur|Secretary|Secretary[- ]?General)\s+(?:noted|stated|said|presented|proposed|observed|announced|reported|explained|suggested|remarked|concluded|summarized)`),
			regexp.MustCompile(`(?i)(?:The\s+)?(Chair(?:man|woman|person)?|President|Rapporteur)\s+([A-Z][a-zA-Z\s.'-]+?)\s+(?:noted|stated|said|presented)`),
		},

		// Speaker name: plain name mentions
		speakerName: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:Mr|Ms|Mrs|Dr|Prof)\.?\s+([A-Z][a-zA-Z\s.'-]{2,30}?)(?:\s+(?:said|stated|noted|proposed|remarked|observed|suggested|asked|responded|replied|emphasized|stressed|pointed out|indicated|explained))`),
		},

		// Member state patterns
		memberState: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:The\s+)?(?:representative|delegate|delegation|ambassador)\s+(?:of|from)\s+([A-Z][a-zA-Z\s]+?)(?:\s+(?:stated|said|noted|proposed|emphasized|stressed|supported|opposed|requested|suggested|observed|remarked|indicated|expressed|highlighted|underlined|welcomed|regretted|agreed|disagreed))`),
			regexp.MustCompile(`(?i)([A-Z][a-zA-Z\s]+?)(?:'s\s+(?:representative|delegate|delegation))`),
			regexp.MustCompile(`(?i)(?:submitted|proposed|tabled)\s+by\s+([A-Z][a-zA-Z\s]+)`),
		},

		// Delegation patterns
		delegation: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:The\s+)?([A-Z][a-zA-Z\s]+?)\s+delegation(?:\s+(?:stated|said|noted|proposed|supported|opposed))?`),
			regexp.MustCompile(`(?i)delegation\s+(?:of|from)\s+([A-Z][a-zA-Z\s]+)`),
		},

		// Organization patterns
		organization: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:The\s+)?([A-Z][a-zA-Z\s]+?)\s+(?:Commission|Council|Parliament|Committee|Secretariat|Agency|Bureau|Office|Department|Ministry|Authority)`),
			regexp.MustCompile(`(?i)(?:representative|observer)\s+(?:of|from)\s+(?:the\s+)?([A-Z][a-zA-Z\s]+?(?:\s+(?:Commission|Council|Organization|Association|Federation|Union|Agency)))`),
		},

		// Role reference patterns
		roleReference: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:The\s+)?(Chair(?:man|woman|person)?|President|Vice[- ]?President|Rapporteur|Secretary(?:[- ]General)?|Co[- ]?Chair|Coordinator)`),
		},

		// Voting record patterns: "[Member State]: For/Against/Abstain"
		votingRecord: []*regexp.Regexp{
			regexp.MustCompile(`(?i)([A-Z][a-zA-Z\s]+?):\s*(?:For|Against|Abstain|In\s+favour|Not\s+voting|Absent)`),
			regexp.MustCompile(`(?i)(?:Voted?\s+)?(For|Against|Abstain):\s+([A-Z][a-zA-Z\s,]+)`),
		},

		// Document author patterns
		documentAuthor: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:Submitted|Proposed|Tabled|Drafted|Prepared)\s+by[:\s]+([A-Z][a-zA-Z\s,]+)`),
			regexp.MustCompile(`(?i)Author[s]?[:\s]+([A-Z][a-zA-Z\s,]+)`),
			regexp.MustCompile(`(?i)(?:Co[- ]?)?Sponsor(?:ed)?(?:\s+by)?[:\s]+([A-Z][a-zA-Z\s,]+)`),
		},
	}
}

// ExtractEntities extracts all entities from meeting minutes text.
func (e *EntityExtractor) ExtractEntities(text string, context EntityContext) (*ExtractionResult, error) {
	if e.store == nil {
		return nil, fmt.Errorf("triple store is nil")
	}

	result := &ExtractionResult{
		Speakers:     []Speaker{},
		Stakeholders: []ExtractedStakeholder{},
		Mentions:     []EntityMention{},
		Resolved:     0,
		Unresolved:   0,
	}

	// Extract speakers with affiliations
	e.extractSpeakersWithAffiliation(text, context, result)

	// Extract role-based speakers
	e.extractRoleSpeakers(text, context, result)

	// Extract named speakers
	e.extractNamedSpeakers(text, context, result)

	// Extract member states
	e.extractMemberStates(text, context, result)

	// Extract delegations
	e.extractDelegations(text, context, result)

	// Extract organizations
	e.extractOrganizations(text, context, result)

	// Extract from voting records
	e.extractFromVotingRecords(text, context, result)

	// Extract document authors
	e.extractDocumentAuthors(text, context, result)

	// Deduplicate results
	result.Speakers = e.deduplicateSpeakers(result.Speakers)
	result.Stakeholders = e.deduplicateStakeholders(result.Stakeholders)

	// Count resolved vs unresolved
	for _, m := range result.Mentions {
		if m.ResolvedURI != "" {
			result.Resolved++
		} else {
			result.Unresolved++
		}
	}

	return result, nil
}

// ExtractionResult contains the results of entity extraction.
type ExtractionResult struct {
	// Speakers are extracted speaker entities.
	Speakers []Speaker `json:"speakers"`

	// Stakeholders are extracted stakeholder entities.
	Stakeholders []ExtractedStakeholder `json:"stakeholders"`

	// Mentions are all entity mentions found.
	Mentions []EntityMention `json:"mentions"`

	// Resolved is the count of resolved mentions.
	Resolved int `json:"resolved"`

	// Unresolved is the count of unresolved mentions.
	Unresolved int `json:"unresolved"`
}

// extractSpeakersWithAffiliation extracts speakers with their affiliations.
func (e *EntityExtractor) extractSpeakersWithAffiliation(text string, context EntityContext, result *ExtractionResult) {
	for _, pattern := range e.patterns.speakerWithAffiliation {
		matches := pattern.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			if len(match) >= 6 {
				name := text[match[2]:match[3]]
				affiliation := text[match[4]:match[5]]

				mention := EntityMention{
					Text:           text[match[0]:match[1]],
					NormalizedText: normalizeName(name),
					MeetingURI:     context.MeetingURI,
					SourceOffset:   match[0],
					ProbableType:   "speaker",
				}

				// Try to resolve
				resolved, confidence := e.resolver.Resolve(name, context)
				if resolved != nil {
					mention.ResolvedURI = resolved.URI
					mention.Confidence = confidence
				}

				result.Mentions = append(result.Mentions, mention)

				// Create speaker
				speaker := Speaker{
					URI:             e.generateSpeakerURI(name),
					Name:            normalizeName(name),
					Affiliation:     e.generateStakeholderURI(affiliation),
					AffiliationName: normalizeName(affiliation),
				}
				if mention.ResolvedURI != "" {
					speaker.URI = mention.ResolvedURI
				}
				result.Speakers = append(result.Speakers, speaker)

				// Create stakeholder for affiliation
				stakeholder := ExtractedStakeholder{
					URI:  e.generateStakeholderURI(affiliation),
					Name: normalizeName(affiliation),
					Type: e.inferStakeholderType(affiliation),
				}
				result.Stakeholders = append(result.Stakeholders, stakeholder)
			}
		}
	}
}

// extractRoleSpeakers extracts speakers referenced by role.
func (e *EntityExtractor) extractRoleSpeakers(text string, context EntityContext, result *ExtractionResult) {
	for _, pattern := range e.patterns.speakerRole {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				role := match[1]

				mention := EntityMention{
					Text:           match[0],
					NormalizedText: normalizeRole(role),
					MeetingURI:     context.MeetingURI,
					ProbableType:   "role",
				}

				// Try to resolve role to a specific person
				resolved, confidence := e.resolver.ResolveRole(role, context)
				if resolved != nil {
					mention.ResolvedURI = resolved.URI
					mention.Confidence = confidence
				}

				result.Mentions = append(result.Mentions, mention)

				// If we have a name in the match (pattern 2)
				if len(match) >= 3 && match[2] != "" {
					speaker := Speaker{
						URI:  e.generateSpeakerURI(match[2]),
						Name: normalizeName(match[2]),
						Roles: []RoleAssignment{{
							Role:       normalizeRole(role),
							ProcessURI: context.ProcessURI,
						}},
					}
					result.Speakers = append(result.Speakers, speaker)
				}
			}
		}
	}
}

// extractNamedSpeakers extracts speakers by name.
func (e *EntityExtractor) extractNamedSpeakers(text string, context EntityContext, result *ExtractionResult) {
	for _, pattern := range e.patterns.speakerName {
		matches := pattern.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			if len(match) >= 4 {
				name := text[match[2]:match[3]]

				mention := EntityMention{
					Text:           text[match[0]:match[1]],
					NormalizedText: normalizeName(name),
					MeetingURI:     context.MeetingURI,
					SourceOffset:   match[0],
					ProbableType:   "speaker",
				}

				// Get context for disambiguation
				contextStart := match[0] - 50
				if contextStart < 0 {
					contextStart = 0
				}
				contextEnd := match[1] + 50
				if contextEnd > len(text) {
					contextEnd = len(text)
				}
				mention.Context = text[contextStart:contextEnd]

				// Try to resolve
				resolved, confidence := e.resolver.Resolve(name, context)
				if resolved != nil {
					mention.ResolvedURI = resolved.URI
					mention.Confidence = confidence
				}

				result.Mentions = append(result.Mentions, mention)

				speaker := Speaker{
					URI:  e.generateSpeakerURI(name),
					Name: normalizeName(name),
				}
				if mention.ResolvedURI != "" {
					speaker.URI = mention.ResolvedURI
				}
				result.Speakers = append(result.Speakers, speaker)
			}
		}
	}
}

// extractMemberStates extracts member state mentions.
func (e *EntityExtractor) extractMemberStates(text string, context EntityContext, result *ExtractionResult) {
	for _, pattern := range e.patterns.memberState {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				stateName := strings.TrimSpace(match[1])
				if !isValidStateName(stateName) {
					continue
				}

				mention := EntityMention{
					Text:           match[0],
					NormalizedText: normalizeName(stateName),
					MeetingURI:     context.MeetingURI,
					ProbableType:   "member_state",
				}

				// Try to resolve
				resolved, confidence := e.resolver.Resolve(stateName, context)
				if resolved != nil {
					mention.ResolvedURI = resolved.URI
					mention.Confidence = confidence
				}

				result.Mentions = append(result.Mentions, mention)

				stakeholder := ExtractedStakeholder{
					URI:  e.generateStakeholderURI(stateName),
					Name: normalizeName(stateName),
					Type: StakeholderMemberState,
				}
				if mention.ResolvedURI != "" {
					stakeholder.URI = mention.ResolvedURI
				}
				result.Stakeholders = append(result.Stakeholders, stakeholder)
			}
		}
	}
}

// extractDelegations extracts delegation mentions.
func (e *EntityExtractor) extractDelegations(text string, context EntityContext, result *ExtractionResult) {
	for _, pattern := range e.patterns.delegation {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				delegationName := strings.TrimSpace(match[1])
				if !isValidDelegationName(delegationName) {
					continue
				}

				mention := EntityMention{
					Text:           match[0],
					NormalizedText: normalizeName(delegationName),
					MeetingURI:     context.MeetingURI,
					ProbableType:   "delegation",
				}

				resolved, confidence := e.resolver.Resolve(delegationName, context)
				if resolved != nil {
					mention.ResolvedURI = resolved.URI
					mention.Confidence = confidence
				}

				result.Mentions = append(result.Mentions, mention)

				stakeholder := ExtractedStakeholder{
					URI:  e.generateStakeholderURI(delegationName),
					Name: normalizeName(delegationName),
					Type: StakeholderDelegation,
				}
				if mention.ResolvedURI != "" {
					stakeholder.URI = mention.ResolvedURI
				}
				result.Stakeholders = append(result.Stakeholders, stakeholder)
			}
		}
	}
}

// extractOrganizations extracts organization mentions.
func (e *EntityExtractor) extractOrganizations(text string, context EntityContext, result *ExtractionResult) {
	for _, pattern := range e.patterns.organization {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				orgName := strings.TrimSpace(match[1])
				if len(orgName) < 3 {
					continue
				}

				mention := EntityMention{
					Text:           match[0],
					NormalizedText: normalizeName(orgName),
					MeetingURI:     context.MeetingURI,
					ProbableType:   "organization",
				}

				resolved, confidence := e.resolver.Resolve(orgName, context)
				if resolved != nil {
					mention.ResolvedURI = resolved.URI
					mention.Confidence = confidence
				}

				result.Mentions = append(result.Mentions, mention)

				stakeholder := ExtractedStakeholder{
					URI:  e.generateStakeholderURI(orgName),
					Name: normalizeName(orgName),
					Type: StakeholderOrganization,
				}
				if mention.ResolvedURI != "" {
					stakeholder.URI = mention.ResolvedURI
				}
				result.Stakeholders = append(result.Stakeholders, stakeholder)
			}
		}
	}
}

// extractFromVotingRecords extracts stakeholders from voting records.
func (e *EntityExtractor) extractFromVotingRecords(text string, context EntityContext, result *ExtractionResult) {
	for _, pattern := range e.patterns.votingRecord {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			// Handle different capture group orders
			var names string
			if len(match) >= 3 {
				// Pattern with position first, then names
				if isVotePosition(match[1]) {
					names = match[2]
				} else {
					names = match[1]
				}
			} else if len(match) >= 2 {
				names = match[1]
			}

			// Split comma-separated names
			for _, name := range strings.Split(names, ",") {
				name = strings.TrimSpace(name)
				if name == "" || !isValidStateName(name) {
					continue
				}

				mention := EntityMention{
					Text:           name,
					NormalizedText: normalizeName(name),
					MeetingURI:     context.MeetingURI,
					ProbableType:   "member_state",
				}

				resolved, confidence := e.resolver.Resolve(name, context)
				if resolved != nil {
					mention.ResolvedURI = resolved.URI
					mention.Confidence = confidence
				}

				result.Mentions = append(result.Mentions, mention)

				stakeholder := ExtractedStakeholder{
					URI:  e.generateStakeholderURI(name),
					Name: normalizeName(name),
					Type: StakeholderMemberState,
				}
				if mention.ResolvedURI != "" {
					stakeholder.URI = mention.ResolvedURI
				}
				result.Stakeholders = append(result.Stakeholders, stakeholder)
			}
		}
	}
}

// extractDocumentAuthors extracts authors from document metadata.
func (e *EntityExtractor) extractDocumentAuthors(text string, context EntityContext, result *ExtractionResult) {
	for _, pattern := range e.patterns.documentAuthor {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				authors := match[1]

				// Split comma-separated authors
				for _, author := range strings.Split(authors, ",") {
					author = strings.TrimSpace(author)
					if author == "" {
						continue
					}

					mention := EntityMention{
						Text:           author,
						NormalizedText: normalizeName(author),
						MeetingURI:     context.MeetingURI,
						ProbableType:   e.inferMentionType(author),
					}

					resolved, confidence := e.resolver.Resolve(author, context)
					if resolved != nil {
						mention.ResolvedURI = resolved.URI
						mention.Confidence = confidence
					}

					result.Mentions = append(result.Mentions, mention)

					// Could be a speaker or stakeholder
					if isLikelyPersonName(author) {
						speaker := Speaker{
							URI:  e.generateSpeakerURI(author),
							Name: normalizeName(author),
						}
						result.Speakers = append(result.Speakers, speaker)
					} else {
						stakeholder := ExtractedStakeholder{
							URI:  e.generateStakeholderURI(author),
							Name: normalizeName(author),
							Type: e.inferStakeholderType(author),
						}
						result.Stakeholders = append(result.Stakeholders, stakeholder)
					}
				}
			}
		}
	}
}

// generateSpeakerURI generates a URI for a speaker.
func (e *EntityExtractor) generateSpeakerURI(name string) string {
	normalized := strings.ToLower(normalizeName(name))
	normalized = strings.ReplaceAll(normalized, " ", "-")
	normalized = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(normalized, "")
	return fmt.Sprintf("%sspeaker:%s", e.baseURI, normalized)
}

// generateStakeholderURI generates a URI for a stakeholder.
func (e *EntityExtractor) generateStakeholderURI(name string) string {
	normalized := strings.ToLower(normalizeName(name))
	normalized = strings.ReplaceAll(normalized, " ", "-")
	normalized = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(normalized, "")
	return fmt.Sprintf("%sstakeholder:%s", e.baseURI, normalized)
}

// inferStakeholderType infers the type of stakeholder from the name.
func (e *EntityExtractor) inferStakeholderType(name string) StakeholderType {
	lower := strings.ToLower(name)

	// Check more specific types before general ones
	if strings.Contains(lower, "secretariat") {
		return StakeholderSecretariat
	}
	if strings.Contains(lower, "committee") {
		return StakeholderCommittee
	}
	if strings.Contains(lower, "delegation") {
		return StakeholderDelegation
	}
	if strings.Contains(lower, "group") {
		return StakeholderPoliticalGroup
	}
	if strings.Contains(lower, "observer") {
		return StakeholderObserver
	}
	if strings.Contains(lower, "commission") {
		return StakeholderOrganization
	}
	if strings.Contains(lower, "council") {
		return StakeholderOrganization
	}
	if strings.Contains(lower, "parliament") {
		return StakeholderOrganization
	}

	// Check if it's likely a country name
	if isLikelyCountryName(name) {
		return StakeholderMemberState
	}

	return StakeholderOrganization
}

// inferMentionType infers whether a mention is a speaker or stakeholder.
func (e *EntityExtractor) inferMentionType(text string) string {
	if isLikelyPersonName(text) {
		return "speaker"
	}
	return "stakeholder"
}

// deduplicateSpeakers removes duplicate speakers by URI.
func (e *EntityExtractor) deduplicateSpeakers(speakers []Speaker) []Speaker {
	seen := make(map[string]bool)
	var unique []Speaker
	for _, s := range speakers {
		if !seen[s.URI] {
			seen[s.URI] = true
			unique = append(unique, s)
		}
	}
	return unique
}

// deduplicateStakeholders removes duplicate stakeholders by URI.
func (e *EntityExtractor) deduplicateStakeholders(stakeholders []ExtractedStakeholder) []ExtractedStakeholder {
	seen := make(map[string]bool)
	var unique []ExtractedStakeholder
	for _, s := range stakeholders {
		if !seen[s.URI] {
			seen[s.URI] = true
			unique = append(unique, s)
		}
	}
	return unique
}

// EntityResolver resolves entity mentions to canonical URIs.
type EntityResolver struct {
	// store is the triple store containing known entities.
	store *store.TripleStore

	// baseURI is the base URI for the deliberation data.
	baseURI string

	// knownEntities maps URI to entity data.
	knownEntities map[string]*ResolvedEntity

	// aliasIndex maps normalized aliases to URIs.
	aliasIndex map[string]string
}

// ResolvedEntity represents a resolved entity.
type ResolvedEntity struct {
	URI     string
	Name    string
	Type    string
	Aliases []string
}

// NewEntityResolver creates a new entity resolver.
func NewEntityResolver(tripleStore *store.TripleStore, baseURI string) *EntityResolver {
	resolver := &EntityResolver{
		store:         tripleStore,
		baseURI:       baseURI,
		knownEntities: make(map[string]*ResolvedEntity),
		aliasIndex:    make(map[string]string),
	}
	resolver.loadKnownEntities()
	return resolver
}

// loadKnownEntities loads known entities from the triple store.
func (r *EntityResolver) loadKnownEntities() {
	if r.store == nil {
		return
	}

	// Load stakeholders
	stakeholderTriples := r.store.Find("", "rdf:type", store.ClassStakeholder)
	for _, st := range stakeholderTriples {
		entity := &ResolvedEntity{URI: st.Subject, Type: "stakeholder"}

		// Get name
		nameTriples := r.store.Find(st.Subject, store.RDFSLabel, "")
		if len(nameTriples) > 0 {
			entity.Name = nameTriples[0].Object
			r.aliasIndex[normalizeName(entity.Name)] = st.Subject
		}

		// Get aliases
		aliasTriples := r.store.Find(st.Subject, store.PropStakeholderAlias, "")
		for _, at := range aliasTriples {
			entity.Aliases = append(entity.Aliases, at.Object)
			r.aliasIndex[normalizeName(at.Object)] = st.Subject
		}

		r.knownEntities[st.Subject] = entity
	}

	// Load speakers (individuals who have spoken)
	speakerTriples := r.store.Find("", store.PropSpeaker, "")
	for _, st := range speakerTriples {
		speakerURI := st.Object
		if _, exists := r.knownEntities[speakerURI]; !exists {
			entity := &ResolvedEntity{URI: speakerURI, Type: "speaker"}

			nameTriples := r.store.Find(speakerURI, store.RDFSLabel, "")
			if len(nameTriples) > 0 {
				entity.Name = nameTriples[0].Object
				r.aliasIndex[normalizeName(entity.Name)] = speakerURI
			}

			r.knownEntities[speakerURI] = entity
		}
	}
}

// Resolve attempts to resolve an entity mention.
func (r *EntityResolver) Resolve(mention string, context EntityContext) (*ResolvedEntity, float64) {
	normalized := normalizeName(mention)

	// 1. Exact alias match
	if uri, ok := r.aliasIndex[normalized]; ok {
		return r.knownEntities[uri], 1.0
	}

	// 2. Fuzzy match
	candidates := r.fuzzyMatch(normalized, 0.8)
	if len(candidates) == 1 {
		return candidates[0].Entity, candidates[0].Score
	}

	// 3. Disambiguate using context
	if len(candidates) > 1 {
		best := r.disambiguate(candidates, context)
		if best != nil {
			return best.Entity, best.Score
		}
	}

	return nil, 0.0
}

// ResolveRole attempts to resolve a role reference to a specific person.
func (r *EntityResolver) ResolveRole(role string, context EntityContext) (*ResolvedEntity, float64) {
	if r.store == nil {
		return nil, 0.0
	}

	normalizedRole := normalizeRole(role)

	// Find entities with this role in the current meeting or process
	roleTriples := r.store.Find("", store.PropHasRole, "")
	for _, rt := range roleTriples {
		// Check if this role matches
		roleValueTriples := r.store.Find(rt.Object, "reg:roleName", "")
		for _, rv := range roleValueTriples {
			if normalizeRole(rv.Object) == normalizedRole {
				// Check scope matches context
				scopeTriples := r.store.Find(rt.Object, store.PropRoleScope, "")
				for _, st := range scopeTriples {
					if st.Object == context.ProcessURI || st.Object == context.MeetingURI {
						if entity, exists := r.knownEntities[rt.Subject]; exists {
							return entity, 0.9
						}
					}
				}
			}
		}
	}

	return nil, 0.0
}

// fuzzyMatchResult holds a fuzzy match result.
type fuzzyMatchResult struct {
	Entity *ResolvedEntity
	Score  float64
}

// fuzzyMatch performs fuzzy matching against known entities.
func (r *EntityResolver) fuzzyMatch(normalized string, threshold float64) []fuzzyMatchResult {
	var results []fuzzyMatchResult

	for alias, uri := range r.aliasIndex {
		score := similarityScore(normalized, alias)
		if score >= threshold {
			if entity, exists := r.knownEntities[uri]; exists {
				results = append(results, fuzzyMatchResult{
					Entity: entity,
					Score:  score,
				})
			}
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// disambiguate selects the best match using context.
func (r *EntityResolver) disambiguate(candidates []fuzzyMatchResult, context EntityContext) *fuzzyMatchResult {
	if len(candidates) == 0 {
		return nil
	}

	// Prefer entities that have participated in the same meeting
	for i, c := range candidates {
		partTriples := r.store.Find(context.MeetingURI, store.PropParticipant, c.Entity.URI)
		if len(partTriples) > 0 {
			candidates[i].Score += 0.1
		}
	}

	// Re-sort and return best
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	return &candidates[0]
}

// AddAlias adds an alias for an entity.
func (r *EntityResolver) AddAlias(uri, alias string) {
	normalized := normalizeName(alias)
	r.aliasIndex[normalized] = uri

	if entity, exists := r.knownEntities[uri]; exists {
		entity.Aliases = append(entity.Aliases, alias)
	}
}

// AddEntity adds a new entity to the resolver.
func (r *EntityResolver) AddEntity(entity *ResolvedEntity) {
	r.knownEntities[entity.URI] = entity
	r.aliasIndex[normalizeName(entity.Name)] = entity.URI
	for _, alias := range entity.Aliases {
		r.aliasIndex[normalizeName(alias)] = entity.URI
	}
}

// Helper functions

// normalizeName normalizes a name for comparison.
func normalizeName(name string) string {
	// Remove common prefixes
	name = regexp.MustCompile(`(?i)^(Mr|Ms|Mrs|Dr|Prof)\.?\s+`).ReplaceAllString(name, "")
	name = regexp.MustCompile(`(?i)^The\s+`).ReplaceAllString(name, "")

	// Trim and normalize whitespace
	name = strings.TrimSpace(name)
	name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")

	return name
}

// normalizeRole normalizes a role name.
func normalizeRole(role string) string {
	role = strings.TrimSpace(role)
	role = strings.ToLower(role)
	role = strings.ReplaceAll(role, "-", "")
	role = strings.ReplaceAll(role, " ", "")
	return role
}

// similarityScore computes a simple similarity score between two strings.
func similarityScore(a, b string) float64 {
	a = strings.ToLower(a)
	b = strings.ToLower(b)

	if a == b {
		return 1.0
	}

	// Simple Jaccard-like similarity on words
	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)

	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0.0
	}

	setA := make(map[string]bool)
	for _, w := range wordsA {
		setA[w] = true
	}

	intersection := 0
	for _, w := range wordsB {
		if setA[w] {
			intersection++
		}
	}

	union := len(wordsA) + len(wordsB) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// isValidStateName checks if a name is likely a valid state/country name.
func isValidStateName(name string) bool {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 50 {
		return false
	}

	// Should start with uppercase
	if len(name) > 0 && (name[0] < 'A' || name[0] > 'Z') {
		return false
	}

	// Exclude common false positives
	lower := strings.ToLower(name)
	excluded := []string{"the", "this", "that", "which", "where", "when", "what", "who", "how",
		"for", "against", "abstain", "voted", "voting", "vote"}
	for _, ex := range excluded {
		if lower == ex {
			return false
		}
	}

	return true
}

// isValidDelegationName checks if a name is likely a valid delegation name.
func isValidDelegationName(name string) bool {
	return isValidStateName(name)
}

// isLikelyPersonName checks if text looks like a person's name.
func isLikelyPersonName(text string) bool {
	// Check for common patterns
	if regexp.MustCompile(`(?i)^(?:Mr|Ms|Mrs|Dr|Prof)\.?\s+`).MatchString(text) {
		return true
	}

	words := strings.Fields(text)
	if len(words) < 2 || len(words) > 5 {
		return false
	}

	// All words should start with uppercase
	for _, w := range words {
		if len(w) > 0 && (w[0] < 'A' || w[0] > 'Z') {
			return false
		}
	}

	// Shouldn't contain organization keywords
	lower := strings.ToLower(text)
	orgKeywords := []string{"commission", "council", "parliament", "committee",
		"organization", "association", "federation", "union", "agency",
		"delegation", "secretariat", "ministry", "department"}
	for _, kw := range orgKeywords {
		if strings.Contains(lower, kw) {
			return false
		}
	}

	return true
}

// isLikelyCountryName checks if text looks like a country name.
func isLikelyCountryName(name string) bool {
	// Common country name patterns
	lower := strings.ToLower(name)

	// Check for common country suffixes
	countrySuffixes := []string{"land", "stan", "ia", "ica"}
	for _, suffix := range countrySuffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}

	// Common EU member state names
	euStates := []string{"germany", "france", "italy", "spain", "poland", "romania",
		"netherlands", "belgium", "greece", "czech", "portugal", "sweden",
		"hungary", "austria", "bulgaria", "denmark", "finland", "slovakia",
		"ireland", "croatia", "lithuania", "slovenia", "latvia", "estonia",
		"cyprus", "luxembourg", "malta"}
	for _, state := range euStates {
		if strings.Contains(lower, state) {
			return true
		}
	}

	return false
}

// isVotePosition checks if text is a vote position.
func isVotePosition(text string) bool {
	lower := strings.ToLower(text)
	positions := []string{"for", "against", "abstain", "in favour", "not voting", "absent"}
	for _, pos := range positions {
		if lower == pos {
			return true
		}
	}
	return false
}

// PersistEntities saves extracted entities to the triple store.
func (e *EntityExtractor) PersistEntities(result *ExtractionResult) error {
	if e.store == nil {
		return fmt.Errorf("triple store is nil")
	}

	// Persist speakers
	for _, speaker := range result.Speakers {
		e.store.Add(speaker.URI, "rdf:type", store.ClassStakeholder)
		e.store.Add(speaker.URI, store.PropStakeholderType, "individual")
		e.store.Add(speaker.URI, store.RDFSLabel, speaker.Name)

		if speaker.Affiliation != "" {
			e.store.Add(speaker.URI, store.PropMemberOf, speaker.Affiliation)
		}

		for _, alias := range speaker.Aliases {
			e.store.Add(speaker.URI, store.PropStakeholderAlias, alias)
		}

		for _, role := range speaker.Roles {
			roleURI := fmt.Sprintf("%s:role", speaker.URI)
			e.store.Add(speaker.URI, store.PropHasRole, roleURI)
			e.store.Add(roleURI, "reg:roleName", role.Role)
			if role.Scope != "" {
				e.store.Add(roleURI, store.PropRoleScope, role.Scope)
			}
		}
	}

	// Persist stakeholders
	for _, stakeholder := range result.Stakeholders {
		e.store.Add(stakeholder.URI, "rdf:type", store.ClassStakeholder)
		e.store.Add(stakeholder.URI, store.PropStakeholderType, stakeholder.Type.String())
		e.store.Add(stakeholder.URI, store.RDFSLabel, stakeholder.Name)

		if stakeholder.ParentOrg != "" {
			e.store.Add(stakeholder.URI, store.PropMemberOf, stakeholder.ParentOrg)
		}

		for _, alias := range stakeholder.Aliases {
			e.store.Add(stakeholder.URI, store.PropStakeholderAlias, alias)
		}

		for _, member := range stakeholder.Members {
			e.store.Add(member, store.PropMemberOf, stakeholder.URI)
		}
	}

	return nil
}

// GetUnresolvedMentions returns all unresolved entity mentions.
func (e *EntityExtractor) GetUnresolvedMentions(result *ExtractionResult) []EntityMention {
	var unresolved []EntityMention
	for _, m := range result.Mentions {
		if m.ResolvedURI == "" {
			unresolved = append(unresolved, m)
		}
	}
	return unresolved
}

// RenderEntityGraph renders entities as a DOT graph.
func (e *EntityExtractor) RenderEntityGraph(result *ExtractionResult) string {
	var sb strings.Builder

	sb.WriteString("digraph Entities {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [shape=box];\n\n")

	// Speakers
	sb.WriteString("  subgraph cluster_speakers {\n")
	sb.WriteString("    label=\"Speakers\";\n")
	sb.WriteString("    style=filled;\n")
	sb.WriteString("    color=lightblue;\n")
	for _, s := range result.Speakers {
		nodeID := strings.ReplaceAll(s.URI, ":", "_")
		nodeID = strings.ReplaceAll(nodeID, "/", "_")
		sb.WriteString(fmt.Sprintf("    %s [label=\"%s\"];\n", nodeID, s.Name))
	}
	sb.WriteString("  }\n\n")

	// Stakeholders
	sb.WriteString("  subgraph cluster_stakeholders {\n")
	sb.WriteString("    label=\"Stakeholders\";\n")
	sb.WriteString("    style=filled;\n")
	sb.WriteString("    color=lightgreen;\n")
	for _, s := range result.Stakeholders {
		nodeID := strings.ReplaceAll(s.URI, ":", "_")
		nodeID = strings.ReplaceAll(nodeID, "/", "_")
		sb.WriteString(fmt.Sprintf("    %s [label=\"%s\\n(%s)\"];\n", nodeID, s.Name, s.Type.String()))
	}
	sb.WriteString("  }\n\n")

	// Affiliations
	for _, s := range result.Speakers {
		if s.Affiliation != "" {
			speakerID := strings.ReplaceAll(s.URI, ":", "_")
			speakerID = strings.ReplaceAll(speakerID, "/", "_")
			affiliationID := strings.ReplaceAll(s.Affiliation, ":", "_")
			affiliationID = strings.ReplaceAll(affiliationID, "/", "_")
			sb.WriteString(fmt.Sprintf("  %s -> %s [label=\"affiliation\"];\n", speakerID, affiliationID))
		}
	}

	sb.WriteString("}\n")
	return sb.String()
}

// RenderJSON renders extraction results as JSON.
func (result *ExtractionResult) RenderJSON() (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
