// Package deliberation provides provenance chain building for tracing
// how decisions evolved from initial proposal through adoption.
package deliberation

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// ProvenanceEventType classifies the type of event in a provenance chain.
type ProvenanceEventType int

const (
	// ProvenanceProposal indicates a new proposal was submitted.
	ProvenanceProposal ProvenanceEventType = iota
	// ProvenanceDiscussion indicates general discussion occurred.
	ProvenanceDiscussion
	// ProvenanceAmendment indicates an amendment was proposed.
	ProvenanceAmendment
	// ProvenanceVote indicates a vote was taken.
	ProvenanceVote
	// ProvenanceDeferral indicates the matter was deferred.
	ProvenanceDeferral
	// ProvenanceAdoption indicates the provision was adopted.
	ProvenanceAdoption
	// ProvenanceRejection indicates the provision was rejected.
	ProvenanceRejection
	// ProvenanceWithdrawal indicates the provision was withdrawn.
	ProvenanceWithdrawal
)

// String returns a human-readable label for the event type.
func (t ProvenanceEventType) String() string {
	switch t {
	case ProvenanceProposal:
		return "PROPOSAL"
	case ProvenanceDiscussion:
		return "DISCUSSION"
	case ProvenanceAmendment:
		return "AMENDMENT"
	case ProvenanceVote:
		return "VOTE"
	case ProvenanceDeferral:
		return "DEFERRAL"
	case ProvenanceAdoption:
		return "ADOPTION"
	case ProvenanceRejection:
		return "REJECTION"
	case ProvenanceWithdrawal:
		return "WITHDRAWAL"
	default:
		return "UNKNOWN"
	}
}

// parseProvenanceEventType converts a string to a ProvenanceEventType.
func parseProvenanceEventType(s string) ProvenanceEventType {
	switch strings.ToLower(s) {
	case "proposal", "proposed":
		return ProvenanceProposal
	case "discussion", "discussed":
		return ProvenanceDiscussion
	case "amendment", "amended":
		return ProvenanceAmendment
	case "vote", "voted":
		return ProvenanceVote
	case "deferral", "deferred":
		return ProvenanceDeferral
	case "adoption", "adopted":
		return ProvenanceAdoption
	case "rejection", "rejected":
		return ProvenanceRejection
	case "withdrawal", "withdrawn":
		return ProvenanceWithdrawal
	default:
		return ProvenanceDiscussion
	}
}

// ProvenanceStep represents a single step in a provenance chain.
type ProvenanceStep struct {
	// Timestamp is when this step occurred.
	Timestamp time.Time `json:"timestamp"`

	// MeetingURI is the meeting where this step occurred.
	MeetingURI string `json:"meeting_uri"`

	// MeetingLabel is a human-readable meeting name.
	MeetingLabel string `json:"meeting_label"`

	// EventType classifies the type of event.
	EventType ProvenanceEventType `json:"event_type"`

	// Description describes what happened in this step.
	Description string `json:"description"`

	// Actor is who initiated this step (stakeholder URI).
	Actor string `json:"actor,omitempty"`

	// ActorName is the human-readable name of the actor.
	ActorName string `json:"actor_name,omitempty"`

	// FromState is the state before this step.
	FromState string `json:"from_state"`

	// ToState is the state after this step.
	ToState string `json:"to_state"`

	// RelatedURIs contains related entities (amendments, votes, etc.).
	RelatedURIs []string `json:"related_uris,omitempty"`

	// VoteRecord contains vote details if this is a vote step.
	VoteRecord *VoteRecord `json:"vote_record,omitempty"`
}

// ProvenanceChain represents the complete provenance of a decision.
type ProvenanceChain struct {
	// FinalDecision is the URI of the adopted decision.
	FinalDecision string `json:"final_decision"`

	// ProvisionURI is the provision this decision affects.
	ProvisionURI string `json:"provision_uri"`

	// ProvisionLabel is a human-readable provision name.
	ProvisionLabel string `json:"provision_label,omitempty"`

	// OriginMeeting is where the topic was first raised.
	OriginMeeting string `json:"origin_meeting"`

	// OriginMeetingLabel is the human-readable name.
	OriginMeetingLabel string `json:"origin_meeting_label,omitempty"`

	// AdoptionMeeting is where the final decision was made.
	AdoptionMeeting string `json:"adoption_meeting,omitempty"`

	// AdoptionMeetingLabel is the human-readable name.
	AdoptionMeetingLabel string `json:"adoption_meeting_label,omitempty"`

	// Steps contains all provenance steps in chronological order.
	Steps []ProvenanceStep `json:"steps"`

	// TotalDuration is the time from first proposal to adoption.
	TotalDuration time.Duration `json:"total_duration"`

	// MeetingCount is the number of unique meetings involved.
	MeetingCount int `json:"meeting_count"`

	// AmendmentCount is the number of amendments in the chain.
	AmendmentCount int `json:"amendment_count"`

	// VoteCount is the number of votes in the chain.
	VoteCount int `json:"vote_count"`

	// CurrentState is the final state of the provision.
	CurrentState string `json:"current_state"`
}

// ProvenanceBuilder builds provenance chains from deliberation data.
type ProvenanceBuilder struct {
	// store is the triple store containing deliberation data.
	store *store.TripleStore

	// baseURI is the base URI for the deliberation data.
	baseURI string
}

// NewProvenanceBuilder creates a new provenance builder.
func NewProvenanceBuilder(tripleStore *store.TripleStore, baseURI string) *ProvenanceBuilder {
	return &ProvenanceBuilder{
		store:   tripleStore,
		baseURI: baseURI,
	}
}

// BuildChainForDecision builds a provenance chain for a specific decision.
func (b *ProvenanceBuilder) BuildChainForDecision(decisionURI string) (*ProvenanceChain, error) {
	if b.store == nil {
		return nil, fmt.Errorf("triple store is nil")
	}
	if decisionURI == "" {
		return nil, fmt.Errorf("decision URI is required")
	}

	chain := &ProvenanceChain{
		FinalDecision: decisionURI,
		Steps:         []ProvenanceStep{},
	}

	// Get the provision this decision affects
	affectsTriples := b.store.Find(decisionURI, store.PropAffectsProvision, "")
	if len(affectsTriples) > 0 {
		chain.ProvisionURI = affectsTriples[0].Object
		chain.ProvisionLabel = b.getLabel(chain.ProvisionURI)
	}

	// Find all events related to this provision
	if chain.ProvisionURI != "" {
		events := b.findProvisionEvents(chain.ProvisionURI)
		chain.Steps = events
	}

	// Also add events directly related to the decision
	decisionEvents := b.findDecisionEvents(decisionURI)
	chain.Steps = append(chain.Steps, decisionEvents...)

	// Sort chronologically
	sort.Slice(chain.Steps, func(i, j int) bool {
		return chain.Steps[i].Timestamp.Before(chain.Steps[j].Timestamp)
	})

	// Remove duplicates and compute state transitions
	chain.Steps = b.deduplicateAndComputeStates(chain.Steps)

	// Calculate statistics
	b.calculateChainStatistics(chain)

	return chain, nil
}

// BuildChainForProvision builds a provenance chain for a specific provision.
func (b *ProvenanceBuilder) BuildChainForProvision(provisionURI string) (*ProvenanceChain, error) {
	if b.store == nil {
		return nil, fmt.Errorf("triple store is nil")
	}
	if provisionURI == "" {
		return nil, fmt.Errorf("provision URI is required")
	}

	chain := &ProvenanceChain{
		ProvisionURI:   provisionURI,
		ProvisionLabel: b.getLabel(provisionURI),
		Steps:          []ProvenanceStep{},
	}

	// Find all events related to this provision
	events := b.findProvisionEvents(provisionURI)
	chain.Steps = events

	// Sort chronologically
	sort.Slice(chain.Steps, func(i, j int) bool {
		return chain.Steps[i].Timestamp.Before(chain.Steps[j].Timestamp)
	})

	// Compute state transitions
	chain.Steps = b.deduplicateAndComputeStates(chain.Steps)

	// Find the final decision if adopted
	adoptedTriples := b.store.Find("", store.PropAffectsProvision, provisionURI)
	for _, at := range adoptedTriples {
		typeTriples := b.store.Find(at.Subject, store.PropDecisionType, "")
		for _, tt := range typeTriples {
			if tt.Object == "adoption" || tt.Object == "adopted" {
				chain.FinalDecision = at.Subject
				break
			}
		}
	}

	// Calculate statistics
	b.calculateChainStatistics(chain)

	return chain, nil
}

// findProvisionEvents finds all events related to a provision.
func (b *ProvenanceBuilder) findProvisionEvents(provisionURI string) []ProvenanceStep {
	var events []ProvenanceStep

	// Find proposals
	proposedTriples := b.store.Find(provisionURI, "reg:proposedAt", "")
	for _, pt := range proposedTriples {
		event := b.buildProvenanceStep(pt.Object, ProvenanceProposal, "Initial proposal submitted")
		proposerTriples := b.store.Find(provisionURI, store.PropProposedBy, "")
		if len(proposerTriples) > 0 {
			event.Actor = proposerTriples[0].Object
			event.ActorName = b.getLabel(event.Actor)
		}
		events = append(events, event)
	}

	// Find discussions
	discussedTriples := b.store.Find(provisionURI, store.PropDiscussedAt, "")
	for _, dt := range discussedTriples {
		event := b.buildProvenanceStep(dt.Object, ProvenanceDiscussion, "General discussion")
		events = append(events, event)
	}

	// Find amendments through version history
	versionTriples := b.store.Find("", store.PropVersionOf, provisionURI)
	for _, vt := range versionTriples {
		versionURI := vt.Subject

		// Check if this version was amended
		amendedTriples := b.store.Find(versionURI, "reg:amendedAt", "")
		for _, at := range amendedTriples {
			event := b.buildProvenanceStep(at.Object, ProvenanceAmendment, "Amendment proposed")
			event.RelatedURIs = append(event.RelatedURIs, versionURI)

			// Get proposer
			proposerTriples := b.store.Find(versionURI, store.PropProposedBy, "")
			if len(proposerTriples) > 0 {
				event.Actor = proposerTriples[0].Object
				event.ActorName = b.getLabel(event.Actor)
			}

			// Get amendment text for description
			textTriples := b.store.Find(versionURI, store.PropText, "")
			if len(textTriples) > 0 {
				text := textTriples[0].Object
				if len(text) > 100 {
					text = text[:100] + "..."
				}
				event.Description = fmt.Sprintf("Amendment: %s", text)
			}

			events = append(events, event)
		}

		// Check for adoption
		adoptedTriples := b.store.Find(versionURI, "reg:adoptedAt", "")
		for _, at := range adoptedTriples {
			event := b.buildProvenanceStep(at.Object, ProvenanceAdoption, "Provision adopted")
			event.RelatedURIs = append(event.RelatedURIs, versionURI)

			// Get vote record
			voteTriples := b.store.Find(versionURI, "reg:hasVote", "")
			if len(voteTriples) > 0 {
				event.VoteRecord = b.getVoteRecord(voteTriples[0].Object)
				if event.VoteRecord != nil {
					event.Description = fmt.Sprintf("Adopted (%d-%d-%d)",
						event.VoteRecord.ForCount,
						event.VoteRecord.AgainstCount,
						event.VoteRecord.AbstainCount)
				}
			}

			events = append(events, event)
		}

		// Check for rejection
		rejectedTriples := b.store.Find(versionURI, "reg:rejectedAt", "")
		for _, rt := range rejectedTriples {
			event := b.buildProvenanceStep(rt.Object, ProvenanceRejection, "Amendment rejected")
			event.RelatedURIs = append(event.RelatedURIs, versionURI)

			// Get vote record
			voteTriples := b.store.Find(versionURI, "reg:hasVote", "")
			if len(voteTriples) > 0 {
				event.VoteRecord = b.getVoteRecord(voteTriples[0].Object)
				if event.VoteRecord != nil {
					event.Description = fmt.Sprintf("Rejected (%d-%d-%d)",
						event.VoteRecord.ForCount,
						event.VoteRecord.AgainstCount,
						event.VoteRecord.AbstainCount)
				}
			}

			events = append(events, event)
		}
	}

	// Find decisions affecting this provision
	decisionTriples := b.store.Find("", store.PropAffectsProvision, provisionURI)
	for _, dt := range decisionTriples {
		decisionURI := dt.Subject

		// Get decision meeting
		meetingTriples := b.store.Find(decisionURI, store.PropDecidedAt, "")
		if len(meetingTriples) == 0 {
			continue
		}

		// Get decision type
		typeTriples := b.store.Find(decisionURI, store.PropDecisionType, "")
		eventType := ProvenanceAdoption
		description := "Decision made"
		if len(typeTriples) > 0 {
			switch typeTriples[0].Object {
			case "adoption", "adopted":
				eventType = ProvenanceAdoption
				description = "Decision: adoption"
			case "rejection", "rejected":
				eventType = ProvenanceRejection
				description = "Decision: rejection"
			case "deferral", "deferred":
				eventType = ProvenanceDeferral
				description = "Decision: deferred"
			case "amendment", "amended":
				eventType = ProvenanceAmendment
				description = "Decision: amendment"
			}
		}

		event := b.buildProvenanceStep(meetingTriples[0].Object, eventType, description)
		event.RelatedURIs = append(event.RelatedURIs, decisionURI)

		// Get decision title
		titleTriples := b.store.Find(decisionURI, store.PropTitle, "")
		if len(titleTriples) > 0 {
			event.Description = titleTriples[0].Object
		}

		events = append(events, event)
	}

	// Find deferrals
	deferredTriples := b.store.Find(provisionURI, store.PropDeferredTo, "")
	for _, dt := range deferredTriples {
		event := b.buildProvenanceStep(dt.Object, ProvenanceDeferral, "Deferred to future meeting")
		events = append(events, event)
	}

	return events
}

// findDecisionEvents finds events directly related to a decision.
func (b *ProvenanceBuilder) findDecisionEvents(decisionURI string) []ProvenanceStep {
	var events []ProvenanceStep

	// Get the meeting where the decision was made
	meetingTriples := b.store.Find(decisionURI, store.PropDecidedAt, "")
	if len(meetingTriples) > 0 {
		typeTriples := b.store.Find(decisionURI, store.PropDecisionType, "")
		eventType := ProvenanceAdoption
		if len(typeTriples) > 0 {
			eventType = parseProvenanceEventType(typeTriples[0].Object)
		}

		titleTriples := b.store.Find(decisionURI, store.PropTitle, "")
		description := "Decision made"
		if len(titleTriples) > 0 {
			description = titleTriples[0].Object
		}

		event := b.buildProvenanceStep(meetingTriples[0].Object, eventType, description)
		event.RelatedURIs = append(event.RelatedURIs, decisionURI)
		events = append(events, event)
	}

	// Get any motions leading to this decision
	motionTriples := b.store.Find(decisionURI, "reg:basedOnMotion", "")
	for _, mt := range motionTriples {
		motionMeetingTriples := b.store.Find(mt.Object, "reg:proposedAt", "")
		if len(motionMeetingTriples) > 0 {
			event := b.buildProvenanceStep(motionMeetingTriples[0].Object, ProvenanceProposal, "Motion proposed")
			event.RelatedURIs = append(event.RelatedURIs, mt.Object)

			proposerTriples := b.store.Find(mt.Object, store.PropProposedBy, "")
			if len(proposerTriples) > 0 {
				event.Actor = proposerTriples[0].Object
				event.ActorName = b.getLabel(event.Actor)
			}

			events = append(events, event)
		}
	}

	return events
}

// buildProvenanceStep creates a ProvenanceStep from a meeting URI.
func (b *ProvenanceBuilder) buildProvenanceStep(meetingURI string, eventType ProvenanceEventType, description string) ProvenanceStep {
	step := ProvenanceStep{
		MeetingURI:   meetingURI,
		MeetingLabel: b.getLabel(meetingURI),
		EventType:    eventType,
		Description:  description,
	}

	// Get meeting date
	dateTriples := b.store.Find(meetingURI, store.PropMeetingDate, "")
	if len(dateTriples) > 0 {
		if d, err := time.Parse(time.RFC3339, dateTriples[0].Object); err == nil {
			step.Timestamp = d
		} else if d, err := time.Parse("2006-01-02", dateTriples[0].Object); err == nil {
			step.Timestamp = d
		}
	}

	return step
}

// deduplicateAndComputeStates removes duplicate events and computes state transitions.
func (b *ProvenanceBuilder) deduplicateAndComputeStates(steps []ProvenanceStep) []ProvenanceStep {
	if len(steps) == 0 {
		return steps
	}

	// Deduplicate by meeting + event type
	seen := make(map[string]bool)
	var unique []ProvenanceStep
	for _, step := range steps {
		key := fmt.Sprintf("%s:%d:%s", step.MeetingURI, step.EventType, step.Timestamp.Format("2006-01-02"))
		if !seen[key] {
			seen[key] = true
			unique = append(unique, step)
		}
	}

	// Compute state transitions
	currentState := "none"
	for i := range unique {
		unique[i].FromState = currentState
		unique[i].ToState = computeNextState(currentState, unique[i].EventType)
		currentState = unique[i].ToState
	}

	return unique
}

// computeNextState determines the next state based on current state and event type.
func computeNextState(currentState string, eventType ProvenanceEventType) string {
	switch eventType {
	case ProvenanceProposal:
		return "proposed"
	case ProvenanceDiscussion:
		if currentState == "proposed" || currentState == "none" {
			return "under_discussion"
		}
		return currentState
	case ProvenanceAmendment:
		return "amended"
	case ProvenanceVote:
		return "voted"
	case ProvenanceDeferral:
		return "deferred"
	case ProvenanceAdoption:
		return "adopted"
	case ProvenanceRejection:
		return "rejected"
	case ProvenanceWithdrawal:
		return "withdrawn"
	default:
		return currentState
	}
}

// calculateChainStatistics computes statistics for the chain.
func (b *ProvenanceBuilder) calculateChainStatistics(chain *ProvenanceChain) {
	if len(chain.Steps) == 0 {
		return
	}

	// Set origin and adoption meetings
	chain.OriginMeeting = chain.Steps[0].MeetingURI
	chain.OriginMeetingLabel = chain.Steps[0].MeetingLabel

	lastStep := chain.Steps[len(chain.Steps)-1]
	if lastStep.EventType == ProvenanceAdoption {
		chain.AdoptionMeeting = lastStep.MeetingURI
		chain.AdoptionMeetingLabel = lastStep.MeetingLabel
	}

	// Calculate duration
	firstTime := chain.Steps[0].Timestamp
	lastTime := lastStep.Timestamp
	if !firstTime.IsZero() && !lastTime.IsZero() {
		chain.TotalDuration = lastTime.Sub(firstTime)
	}

	// Count unique meetings
	meetings := make(map[string]bool)
	for _, step := range chain.Steps {
		if step.MeetingURI != "" {
			meetings[step.MeetingURI] = true
		}
	}
	chain.MeetingCount = len(meetings)

	// Count amendments and votes
	for _, step := range chain.Steps {
		switch step.EventType {
		case ProvenanceAmendment:
			chain.AmendmentCount++
		case ProvenanceVote:
			chain.VoteCount++
		}
	}

	// Set current state
	chain.CurrentState = lastStep.ToState
}

// getLabel retrieves a human-readable label for a URI.
func (b *ProvenanceBuilder) getLabel(uri string) string {
	labelTriples := b.store.Find(uri, store.RDFSLabel, "")
	if len(labelTriples) > 0 {
		return labelTriples[0].Object
	}
	titleTriples := b.store.Find(uri, store.PropTitle, "")
	if len(titleTriples) > 0 {
		return titleTriples[0].Object
	}
	return extractURILabel(uri)
}

// getVoteRecord retrieves a vote record from the store.
func (b *ProvenanceBuilder) getVoteRecord(voteURI string) *VoteRecord {
	vote := &VoteRecord{URI: voteURI}

	// Get vote date
	dateTriples := b.store.Find(voteURI, store.PropVoteDate, "")
	if len(dateTriples) > 0 {
		if d, err := time.Parse(time.RFC3339, dateTriples[0].Object); err == nil {
			vote.VoteDate = d
		} else if d, err := time.Parse("2006-01-02", dateTriples[0].Object); err == nil {
			vote.VoteDate = d
		}
	}

	// Get vote counts
	forTriples := b.store.Find(voteURI, store.PropVoteFor, "")
	if len(forTriples) > 0 {
		vote.ForCount = parseInt(forTriples[0].Object)
	}

	againstTriples := b.store.Find(voteURI, store.PropVoteAgainst, "")
	if len(againstTriples) > 0 {
		vote.AgainstCount = parseInt(againstTriples[0].Object)
	}

	abstainTriples := b.store.Find(voteURI, store.PropVoteAbstain, "")
	if len(abstainTriples) > 0 {
		vote.AbstainCount = parseInt(abstainTriples[0].Object)
	}

	// Get result
	resultTriples := b.store.Find(voteURI, store.PropVoteResult, "")
	if len(resultTriples) > 0 {
		vote.Result = resultTriples[0].Object
	}

	return vote
}

// QueryLongestChains finds provisions with the longest provenance chains.
func (b *ProvenanceBuilder) QueryLongestChains(limit int) ([]ProvenanceChainSummary, error) {
	if b.store == nil {
		return nil, fmt.Errorf("triple store is nil")
	}

	// Find all provisions
	provisionTypes := []string{store.ClassArticle, store.ClassParagraph, store.ClassSection}
	provisionMap := make(map[string]int)

	for _, pType := range provisionTypes {
		provTriples := b.store.Find("", "rdf:type", pType)
		for _, pt := range provTriples {
			// Count events for this provision
			events := b.findProvisionEvents(pt.Subject)
			if len(events) > 0 {
				provisionMap[pt.Subject] = len(events)
			}
		}
	}

	// Also check provisions with version history
	versionTriples := b.store.Find("", store.PropVersionOf, "")
	for _, vt := range versionTriples {
		if _, exists := provisionMap[vt.Object]; !exists {
			events := b.findProvisionEvents(vt.Object)
			if len(events) > 0 {
				provisionMap[vt.Object] = len(events)
			}
		}
	}

	// Convert to summaries and sort
	var summaries []ProvenanceChainSummary
	for uri, count := range provisionMap {
		summaries = append(summaries, ProvenanceChainSummary{
			ProvisionURI:   uri,
			ProvisionLabel: b.getLabel(uri),
			StepCount:      count,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].StepCount > summaries[j].StepCount
	})

	if limit > 0 && len(summaries) > limit {
		summaries = summaries[:limit]
	}

	return summaries, nil
}

// ProvenanceChainSummary provides a brief summary of a chain.
type ProvenanceChainSummary struct {
	ProvisionURI   string `json:"provision_uri"`
	ProvisionLabel string `json:"provision_label"`
	StepCount      int    `json:"step_count"`
}

// RenderASCII renders the provenance chain as an ASCII tree.
func (c *ProvenanceChain) RenderASCII() string {
	var sb strings.Builder

	// Header
	title := "Provenance Chain"
	if c.ProvisionLabel != "" {
		title = fmt.Sprintf("Provenance Chain: %s", c.ProvisionLabel)
	}
	sb.WriteString(title + "\n")
	sb.WriteString(strings.Repeat("═", len(title)) + "\n\n")

	if len(c.Steps) == 0 {
		sb.WriteString("No provenance steps found.\n")
		return sb.String()
	}

	// Group steps by meeting
	meetingSteps := make(map[string][]ProvenanceStep)
	meetingOrder := []string{}
	for _, step := range c.Steps {
		if _, exists := meetingSteps[step.MeetingURI]; !exists {
			meetingOrder = append(meetingOrder, step.MeetingURI)
		}
		meetingSteps[step.MeetingURI] = append(meetingSteps[step.MeetingURI], step)
	}

	// Render each meeting
	for i, meetingURI := range meetingOrder {
		steps := meetingSteps[meetingURI]
		if len(steps) == 0 {
			continue
		}

		// Meeting header
		meetingLabel := steps[0].MeetingLabel
		if meetingLabel == "" {
			meetingLabel = extractURILabel(meetingURI)
		}
		dateStr := ""
		if !steps[0].Timestamp.IsZero() {
			dateStr = fmt.Sprintf(" (%s)", steps[0].Timestamp.Format("2006-01-02"))
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", meetingLabel, dateStr))

		// Render steps
		for j, step := range steps {
			isLast := i == len(meetingOrder)-1 && j == len(steps)-1
			connector := "├─"
			if j == len(steps)-1 && i < len(meetingOrder)-1 {
				connector = "│"
			}
			if isLast {
				connector = "└─"
			}

			// Event line
			sb.WriteString(fmt.Sprintf("  %s [%s] %s\n", connector, step.EventType.String(), step.Description))

			// State transition
			if step.FromState != "" && step.ToState != "" {
				stateConnector := "│"
				if isLast {
					stateConnector = " "
				}
				sb.WriteString(fmt.Sprintf("  %s  State: %s → %s\n", stateConnector, step.FromState, step.ToState))
			}

			// Actor
			if step.ActorName != "" {
				stateConnector := "│"
				if isLast {
					stateConnector = " "
				}
				sb.WriteString(fmt.Sprintf("  %s  By: %s\n", stateConnector, step.ActorName))
			}

			// Vote record
			if step.VoteRecord != nil {
				stateConnector := "│"
				if isLast {
					stateConnector = " "
				}
				sb.WriteString(fmt.Sprintf("  %s  Vote: %d for, %d against, %d abstain\n",
					stateConnector, step.VoteRecord.ForCount, step.VoteRecord.AgainstCount, step.VoteRecord.AbstainCount))
			}
		}

		if i < len(meetingOrder)-1 {
			sb.WriteString("  │\n")
		}
	}

	// Summary
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("Duration: %s | Meetings: %d | Amendments: %d | Votes: %d\n",
		formatDuration(c.TotalDuration), c.MeetingCount, c.AmendmentCount, c.VoteCount))

	return sb.String()
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days == 0 {
		return "< 1 day"
	}
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

// RenderBrief renders an abbreviated version of the chain.
func (c *ProvenanceChain) RenderBrief() string {
	var sb strings.Builder

	label := c.ProvisionLabel
	if label == "" {
		label = extractURILabel(c.ProvisionURI)
	}

	sb.WriteString(fmt.Sprintf("%s: ", label))

	// List event types
	eventTypes := []string{}
	for _, step := range c.Steps {
		eventTypes = append(eventTypes, step.EventType.String())
	}
	sb.WriteString(strings.Join(eventTypes, " → "))

	sb.WriteString(fmt.Sprintf(" [%s, %d meetings]", c.CurrentState, c.MeetingCount))

	return sb.String()
}

// RenderJSON renders the provenance chain as JSON.
func (c *ProvenanceChain) RenderJSON() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RenderHTML renders the provenance chain as HTML.
func (c *ProvenanceChain) RenderHTML() string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>Provenance Chain</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 20px; }
h1 { color: #333; }
.chain { margin: 20px 0; }
.meeting { margin-bottom: 20px; }
.meeting-header { font-weight: bold; color: #0066cc; font-size: 1.1em; }
.meeting-date { color: #666; font-size: 0.9em; }
.step { margin: 10px 0 10px 30px; padding: 10px; border-left: 3px solid #ddd; }
.step.proposal { border-color: #28a745; }
.step.discussion { border-color: #6c757d; }
.step.amendment { border-color: #fd7e14; }
.step.vote { border-color: #007bff; }
.step.adoption { border-color: #28a745; background: #e6ffec; }
.step.rejection { border-color: #dc3545; background: #ffebe9; }
.step.deferral { border-color: #ffc107; }
.step.withdrawal { border-color: #6c757d; }
.event-type { font-weight: bold; color: #333; }
.description { margin-top: 5px; }
.state-change { color: #666; font-size: 0.9em; margin-top: 5px; }
.actor { color: #0066cc; font-size: 0.9em; }
.vote-record { background: #f5f5f5; padding: 5px 10px; margin-top: 5px; border-radius: 3px; }
.summary { background: #f5f5f5; padding: 15px; border-radius: 5px; margin-top: 20px; }
.summary-item { display: inline-block; margin-right: 20px; }
.summary-value { font-weight: bold; font-size: 1.2em; }
</style>
</head>
<body>
`)

	// Header
	title := "Provenance Chain"
	if c.ProvisionLabel != "" {
		title = fmt.Sprintf("Provenance Chain: %s", c.ProvisionLabel)
	}
	sb.WriteString(fmt.Sprintf("<h1>%s</h1>\n", title))

	if len(c.Steps) == 0 {
		sb.WriteString("<p>No provenance steps found.</p>\n")
		sb.WriteString("</body>\n</html>")
		return sb.String()
	}

	sb.WriteString("<div class=\"chain\">\n")

	// Group steps by meeting
	meetingSteps := make(map[string][]ProvenanceStep)
	meetingOrder := []string{}
	for _, step := range c.Steps {
		if _, exists := meetingSteps[step.MeetingURI]; !exists {
			meetingOrder = append(meetingOrder, step.MeetingURI)
		}
		meetingSteps[step.MeetingURI] = append(meetingSteps[step.MeetingURI], step)
	}

	// Render each meeting
	for _, meetingURI := range meetingOrder {
		steps := meetingSteps[meetingURI]
		if len(steps) == 0 {
			continue
		}

		sb.WriteString("<div class=\"meeting\">\n")

		// Meeting header
		meetingLabel := steps[0].MeetingLabel
		if meetingLabel == "" {
			meetingLabel = extractURILabel(meetingURI)
		}
		sb.WriteString(fmt.Sprintf("<div class=\"meeting-header\">%s", meetingLabel))
		if !steps[0].Timestamp.IsZero() {
			sb.WriteString(fmt.Sprintf(" <span class=\"meeting-date\">(%s)</span>", steps[0].Timestamp.Format("2006-01-02")))
		}
		sb.WriteString("</div>\n")

		// Render steps
		for _, step := range steps {
			cssClass := strings.ToLower(step.EventType.String())
			sb.WriteString(fmt.Sprintf("<div class=\"step %s\">\n", cssClass))
			sb.WriteString(fmt.Sprintf("<span class=\"event-type\">[%s]</span>\n", step.EventType.String()))
			sb.WriteString(fmt.Sprintf("<div class=\"description\">%s</div>\n", step.Description))

			if step.FromState != "" && step.ToState != "" {
				sb.WriteString(fmt.Sprintf("<div class=\"state-change\">State: %s → %s</div>\n", step.FromState, step.ToState))
			}

			if step.ActorName != "" {
				sb.WriteString(fmt.Sprintf("<div class=\"actor\">By: %s</div>\n", step.ActorName))
			}

			if step.VoteRecord != nil {
				sb.WriteString(fmt.Sprintf("<div class=\"vote-record\">Vote: %d for, %d against, %d abstain</div>\n",
					step.VoteRecord.ForCount, step.VoteRecord.AgainstCount, step.VoteRecord.AbstainCount))
			}

			sb.WriteString("</div>\n")
		}

		sb.WriteString("</div>\n")
	}

	sb.WriteString("</div>\n")

	// Summary
	sb.WriteString("<div class=\"summary\">\n")
	sb.WriteString(fmt.Sprintf("<div class=\"summary-item\"><span class=\"summary-value\">%s</span> Duration</div>\n", formatDuration(c.TotalDuration)))
	sb.WriteString(fmt.Sprintf("<div class=\"summary-item\"><span class=\"summary-value\">%d</span> Meetings</div>\n", c.MeetingCount))
	sb.WriteString(fmt.Sprintf("<div class=\"summary-item\"><span class=\"summary-value\">%d</span> Amendments</div>\n", c.AmendmentCount))
	sb.WriteString(fmt.Sprintf("<div class=\"summary-item\"><span class=\"summary-value\">%d</span> Votes</div>\n", c.VoteCount))
	sb.WriteString(fmt.Sprintf("<div class=\"summary-item\">Final State: <span class=\"summary-value\">%s</span></div>\n", c.CurrentState))
	sb.WriteString("</div>\n")

	sb.WriteString("</body>\n</html>")
	return sb.String()
}

// RenderMermaid renders the provenance chain as a Mermaid flowchart.
func (c *ProvenanceChain) RenderMermaid() string {
	var sb strings.Builder

	sb.WriteString("graph TD\n")

	if len(c.Steps) == 0 {
		sb.WriteString("    empty[No provenance steps]\n")
		return sb.String()
	}

	// Generate node IDs
	for i, step := range c.Steps {
		nodeID := fmt.Sprintf("step%d", i)
		label := fmt.Sprintf("%s<br/>%s", step.EventType.String(), step.MeetingLabel)
		if !step.Timestamp.IsZero() {
			label += fmt.Sprintf("<br/>%s", step.Timestamp.Format("2006-01-02"))
		}

		// Style based on event type
		style := ""
		switch step.EventType {
		case ProvenanceProposal:
			style = ":::proposal"
		case ProvenanceAdoption:
			style = ":::adoption"
		case ProvenanceRejection:
			style = ":::rejection"
		case ProvenanceAmendment:
			style = ":::amendment"
		}

		sb.WriteString(fmt.Sprintf("    %s[\"%s\"]%s\n", nodeID, label, style))

		// Connect to previous step
		if i > 0 {
			prevID := fmt.Sprintf("step%d", i-1)
			edgeLabel := step.ToState
			sb.WriteString(fmt.Sprintf("    %s -->|%s| %s\n", prevID, edgeLabel, nodeID))
		}
	}

	// Add style definitions
	sb.WriteString("\n    classDef proposal fill:#28a745,color:white\n")
	sb.WriteString("    classDef adoption fill:#28a745,color:white\n")
	sb.WriteString("    classDef rejection fill:#dc3545,color:white\n")
	sb.WriteString("    classDef amendment fill:#fd7e14,color:white\n")

	return sb.String()
}
