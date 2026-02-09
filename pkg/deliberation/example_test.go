package deliberation_test

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/coolbeans/regula/pkg/deliberation"
)

// ExampleMeeting_fullDeliberation demonstrates a complete meeting with agenda items,
// motions, decisions, interventions, and action items - showing all types in context.
func ExampleMeeting_fullDeliberation() {
	// Meeting date and times
	meetingDate := time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC)
	startTime := time.Date(2024, 5, 15, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, 5, 15, 17, 0, 0, 0, time.UTC)
	decidedAt := time.Date(2024, 5, 15, 14, 30, 0, 0, time.UTC)
	dueDate := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	// Create the meeting
	meeting := deliberation.NewMeeting(
		"https://example.org/meetings/wg-2024-05",
		"WG-2024-05",
		"Working Group A - 5th Session",
		meetingDate,
	)
	meeting.Series = "Working Group A"
	meeting.Sequence = 5
	meeting.StartTime = &startTime
	meeting.EndTime = &endTime
	meeting.Location = "Conference Room B / Virtual"
	meeting.Status = deliberation.MeetingCompleted
	meeting.Chair = "https://example.org/stakeholders/member-state-x"
	meeting.ChairName = "Member State X"
	meeting.ProcessURI = "https://example.org/processes/gdpr-review-2024"
	meeting.PreviousMeeting = "https://example.org/meetings/wg-2024-04"

	// Create agenda item with full deliberation
	agendaItem := deliberation.NewAgendaItem(
		"https://example.org/agenda/wg-2024-05/item-3",
		"3",
		"Review of Article 5 - Data Retention",
		meeting.URI,
	)
	agendaItem.Description = "Consideration of proposed amendments to Article 5 data retention provisions"
	agendaItem.Outcome = deliberation.OutcomeDecided
	agendaItem.ProvisionsDiscussed = []string{
		"https://regula.dev/regulations/GDPR:Art5",
		"https://regula.dev/regulations/GDPR:Art5:1",
	}

	// Create motion/amendment
	motion := deliberation.NewMotion(
		"https://example.org/motions/wg-2024-05/amend-1",
		"Amendment 1",
		"30-day data retention limit",
		"Personal data shall be retained for no longer than 30 days unless required for ongoing legal proceedings.",
		"https://example.org/stakeholders/member-state-y",
		meeting.URI,
	)
	motion.Type = "amendment"
	motion.ExistingText = "Personal data shall be retained for no longer than necessary."
	motion.ProposedText = "Personal data shall be retained for no longer than 30 days unless required for ongoing legal proceedings."
	motion.TargetProvisionURI = "https://regula.dev/regulations/GDPR:Art5:1:e"
	motion.SeconderURI = "https://example.org/stakeholders/member-state-z"
	motion.SeconderName = "Member State Z"
	motion.Status = deliberation.MotionAdopted
	motion.SupportersURIs = []string{
		"https://example.org/stakeholders/member-state-a",
		"https://example.org/stakeholders/member-state-b",
	}
	motion.OpponentsURIs = []string{
		"https://example.org/stakeholders/member-state-c",
	}

	// Create vote record
	voteRecord := &deliberation.VoteRecord{
		URI:              "https://example.org/votes/wg-2024-05/vote-1",
		VoteDate:         decidedAt,
		VoteType:         "roll_call",
		Question:         "Shall Amendment 1 (30-day retention limit) be adopted?",
		Result:           "adopted",
		MajorityRequired: "simple",
		ForCount:         18,
		AgainstCount:     4,
		AbstainCount:     1,
		AbsentCount:      2,
		MeetingURI:       meeting.URI,
		MotionURI:        motion.URI,
		IndividualVotes: []deliberation.IndividualVote{
			{
				VoterURI:  "https://example.org/stakeholders/member-state-x",
				VoterName: "Member State X",
				Position:  deliberation.VoteFor,
			},
			{
				VoterURI:    "https://example.org/stakeholders/member-state-c",
				VoterName:   "Member State C",
				Position:    deliberation.VoteAgainst,
				Explanation: "Concerns about implementation burden on small businesses",
			},
		},
	}
	motion.Vote = voteRecord

	// Create interventions
	interventions := []deliberation.Intervention{
		{
			URI:             "https://example.org/interventions/wg-2024-05/int-1",
			SpeakerURI:      "https://example.org/speakers/delegate-y",
			SpeakerName:     "Delegate from Member State Y",
			AffiliationURI:  "https://example.org/stakeholders/member-state-y",
			AffiliationName: "Member State Y",
			MeetingURI:      meeting.URI,
			AgendaItemURI:   agendaItem.URI,
			MotionURI:       motion.URI,
			Position:        deliberation.PositionSupport,
			Summary:         "Introduced Amendment 1, citing need for concrete retention limits to protect data subjects",
			Sequence:        1,
		},
		{
			URI:             "https://example.org/interventions/wg-2024-05/int-2",
			SpeakerURI:      "https://example.org/speakers/delegate-c",
			SpeakerName:     "Delegate from Member State C",
			AffiliationURI:  "https://example.org/stakeholders/member-state-c",
			AffiliationName: "Member State C",
			MeetingURI:      meeting.URI,
			AgendaItemURI:   agendaItem.URI,
			MotionURI:       motion.URI,
			Position:        deliberation.PositionOppose,
			Summary:         "Expressed concerns about implementation burden, requested flexibility for SMEs",
			Sequence:        2,
		},
		{
			URI:             "https://example.org/interventions/wg-2024-05/int-3",
			SpeakerURI:      "https://example.org/speakers/delegate-a",
			SpeakerName:     "Delegate from Member State A",
			AffiliationURI:  "https://example.org/stakeholders/member-state-a",
			AffiliationName: "Member State A",
			MeetingURI:      meeting.URI,
			AgendaItemURI:   agendaItem.URI,
			MotionURI:       motion.URI,
			Position:        deliberation.PositionQualified,
			Summary:         "Supported the amendment with suggestion to add exception for legal holds",
			Sequence:        3,
		},
	}
	agendaItem.Interventions = interventions

	// Create decision
	decision := deliberation.NewDecision(
		"https://example.org/decisions/wg-2024-05/dec-1",
		"Decision WG-2024-05-01",
		"Adoption of 30-day retention limit",
		"The Working Group adopted Amendment 1, establishing a 30-day retention limit for personal data with exceptions for legal proceedings.",
		meeting.URI,
		decidedAt,
	)
	decision.Type = "adoption"
	decision.AgendaItemURI = agendaItem.URI
	decision.MotionURI = motion.URI
	decision.VoteURI = voteRecord.URI
	decision.AffectedProvisionURIs = []string{
		"https://regula.dev/regulations/GDPR:Art5:1:e",
	}

	agendaItem.Motions = []deliberation.Motion{*motion}
	agendaItem.Decisions = []deliberation.Decision{*decision}

	// Create action item
	action := deliberation.NewActionItem(
		"https://example.org/actions/wg-2024-05/action-1",
		"Action WG-2024-05-01",
		"Draft consolidated text incorporating the adopted amendment for Article 5",
		[]string{"https://example.org/stakeholders/secretariat"},
		meeting.URI,
	)
	action.AssignedToNames = []string{"Secretariat"}
	action.DueDate = &dueDate
	action.AgendaItemURI = agendaItem.URI
	action.RelatedProvisionURIs = []string{
		"https://regula.dev/regulations/GDPR:Art5",
	}
	action.Priority = "high"

	agendaItem.ActionItems = []deliberation.ActionItem{*action}
	meeting.AgendaItems = []deliberation.AgendaItem{*agendaItem}

	// Serialize to JSON to demonstrate structure
	jsonBytes, err := json.MarshalIndent(meeting, "", "  ")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Print a summary instead of full JSON (which would be very long)
	fmt.Printf("Meeting: %s\n", meeting.String())
	fmt.Printf("Status: %s\n", meeting.Status)
	fmt.Printf("Chair: %s\n", meeting.ChairName)
	fmt.Printf("Agenda Items: %d\n", len(meeting.AgendaItems))
	fmt.Printf("  Item %s: %s\n", agendaItem.Number, agendaItem.Title)
	fmt.Printf("    Outcome: %s\n", agendaItem.Outcome)
	fmt.Printf("    Motions: %d\n", len(agendaItem.Motions))
	fmt.Printf("    Decisions: %d\n", len(agendaItem.Decisions))
	fmt.Printf("    Interventions: %d\n", len(agendaItem.Interventions))
	fmt.Printf("    Action Items: %d\n", len(agendaItem.ActionItems))
	fmt.Printf("Vote Result: %s (%d-%d-%d)\n",
		voteRecord.Result,
		voteRecord.ForCount,
		voteRecord.AgainstCount,
		voteRecord.AbstainCount)
	// JSON output is available but length varies based on implementation
	_ = jsonBytes

	// Output:
	// Meeting: Working Group A - 5th Session (WG-2024-05) - 2024-05-15
	// Status: completed
	// Chair: Member State X
	// Agenda Items: 1
	//   Item 3: Review of Article 5 - Data Retention
	//     Outcome: decided
	//     Motions: 1
	//     Decisions: 1
	//     Interventions: 3
	//     Action Items: 1
	// Vote Result: adopted (18-4-1)
}
