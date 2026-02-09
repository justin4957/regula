# Deliberation Document Schema Design

This document explains the design decisions behind the deliberation document schema, including the temporal modeling approach and how it integrates with the existing Regula knowledge graph.

## Overview

The deliberation schema extends Regula from static regulation parsing to **living knowledge graphs** that track the evolution of provisions through meetings, discussions, and decisions over time.

## Core Design Principles

### 1. Temporal Anchoring via Meetings

Meetings serve as the primary temporal anchor in the deliberation graph. Every decision, motion, vote, and intervention is linked to the meeting where it occurred, enabling:

- **Point-in-time queries**: "What was the state of Article 5 after Meeting 3?"
- **Evolution tracking**: "How did Article 5 change across meetings?"
- **Provenance chains**: "Trace the path from proposal to adoption"

```
Meeting 1 ─────── Meeting 2 ─────── Meeting 3 ─────── Meeting 4
    │                 │                 │                 │
    │                 │                 │                 │
    ▼                 ▼                 ▼                 ▼
[proposed]       [amended]         [voted]          [adopted]
```

### 2. Hierarchical Document Structure

The schema follows a natural hierarchy:

```
DeliberationProcess
    └── Meeting
            └── AgendaItem
                    ├── Motion/Amendment
                    │       └── VoteRecord
                    │               └── IndividualVote
                    ├── Decision
                    ├── Intervention
                    └── ActionItem
```

Each level links to its parent via URI references, enabling both top-down traversal (process → meetings → items) and bottom-up provenance tracking (decision → meeting → process).

### 3. Status-Based State Machines

Key entities use enum-based status fields that represent state machines:

**MeetingStatus:**
```
scheduled ──► in_progress ──► completed
     │                            │
     └──► cancelled               │
     └──► postponed ──────────────┘
```

**MotionStatus:**
```
proposed ──► seconded ──► debated ──► voted ──► adopted
    │            │                        │         │
    │            └────────────────────────┘         │
    │                                               │
    └──► withdrawn                     rejected ◄──┘
                                           │
                        tabled ◄───────────┘
```

**ActionItemStatus:**
```
pending ──► in_progress ──► completed
    │            │
    └──► deferred
    └──► cancelled
```

### 4. Dual Reference Pattern

Entities maintain both URI references and human-readable names for key relationships:

```go
type Motion struct {
    ProposerURI  string `json:"proposer_uri"`   // For graph traversal
    ProposerName string `json:"proposer_name"`  // For display
}
```

This enables efficient graph queries while maintaining readable output without additional lookups.

## Temporal Modeling Approach

### Current Approach: Explicit Meeting Links

Rather than using RDF reification or named graphs for temporal data, we use explicit meeting links:

```turtle
<motion:amend-1> reg:proposedAt <meeting:2024-05> .
<motion:amend-1> reg:adoptedAt <meeting:2024-06> .
<decision:dec-1> reg:decidedAt <meeting:2024-06> .
```

**Advantages:**
- Simple to query with standard SPARQL
- No special temporal extensions required
- Natural mapping to deliberation domain

**Trade-offs:**
- Point-in-time queries require filtering by meeting dates
- Version chains must be explicitly constructed

### Future Consideration: RDF-Star

For more complex temporal requirements, RDF-star could annotate triples with validity periods:

```turtle
<< <provision:art5> reg:text "Original text" >> reg:validFrom "2024-01-01" .
<< <provision:art5> reg:text "Amended text" >> reg:validFrom "2024-06-01" .
```

This is noted as a future enhancement in the epic (#215).

## RDF Predicate Organization

Predicates are organized by function in `pkg/store/schema.go`:

| Category | Example Predicates |
|----------|-------------------|
| Meeting Structure | `reg:meetingDate`, `reg:meetingSeries`, `reg:hasAgendaItem` |
| Agenda Items | `reg:agendaItemNumber`, `reg:agendaItemOutcome`, `reg:provisionDiscussed` |
| Motions | `reg:proposedBy`, `reg:secondedBy`, `reg:motionStatus` |
| Voting | `reg:voteFor`, `reg:voteAgainst`, `reg:votePosition` |
| Decisions | `reg:decidedAt`, `reg:affectsProvision`, `reg:decisionType` |
| Interventions | `reg:speaker`, `reg:interventionPosition` |
| Actions | `reg:actionAssignedTo`, `reg:actionDueDate`, `reg:actionStatus` |
| Process | `reg:partOfProcess`, `reg:processStatus` |

## Integration with Existing Schema

The deliberation schema extends the existing regulation schema:

### Cross-References to Provisions

```turtle
# Agenda item discusses existing provision
<agenda:item-3> reg:provisionDiscussed <GDPR:Art5> .

# Motion targets existing provision
<motion:amend-1> reg:targetProvision <GDPR:Art5:1:e> .

# Decision affects existing provision
<decision:dec-1> reg:affectsProvision <GDPR:Art5:1:e> .
```

### Bidirectional Links

```turtle
# From regulation graph
<GDPR:Art5> reg:discussedAt <meeting:2024-05> .
<GDPR:Art5> reg:amendedAt <meeting:2024-06> .

# From deliberation graph
<meeting:2024-05> reg:discusses <GDPR:Art5> .
<decision:dec-1> reg:affectsProvision <GDPR:Art5> .
```

### Stakeholder as Actor

The `Stakeholder` type extends the existing `Entity` concept:

```turtle
<stakeholder:member-state-x> a reg:Stakeholder ;
    reg:stakeholderType "member_state" ;
    reg:hasRole [ reg:role "Chair" ; reg:roleScope "Working Group A" ] .

# Used as actor in various contexts
<motion:amend-1> reg:proposedBy <stakeholder:member-state-y> .
<vote:v1> reg:voter <stakeholder:member-state-x> .
<action:a1> reg:actionAssignedTo <stakeholder:secretariat> .
```

## Query Patterns

### 1. Evolution Query

"Show how Article 5 evolved across all meetings"

```sparql
SELECT ?meeting ?date ?eventType ?description WHERE {
    ?event reg:affectsProvision <GDPR:Art5> .
    ?event reg:meetingURI ?meeting .
    ?meeting reg:meetingDate ?date .
    ?event rdf:type ?eventType .
    OPTIONAL { ?event reg:description ?description }
} ORDER BY ?date
```

### 2. Stakeholder Position Query

"What positions did Member State X take on data retention?"

```sparql
SELECT ?meeting ?topic ?position WHERE {
    ?intervention reg:speaker <stakeholder:member-state-x> .
    ?intervention reg:meetingURI ?meeting .
    ?intervention reg:interventionPosition ?position .
    ?intervention reg:agendaItemURI ?item .
    ?item reg:title ?topic .
    FILTER(CONTAINS(LCASE(?topic), "retention"))
}
```

### 3. Pending Actions Query

"List all overdue action items"

```sparql
SELECT ?action ?description ?dueDate ?assignee WHERE {
    ?action a reg:ActionItem .
    ?action reg:actionStatus "pending" .
    ?action reg:actionDueDate ?dueDate .
    ?action reg:description ?description .
    ?action reg:actionAssignedTo ?assignee .
    FILTER(?dueDate < NOW())
}
```

### 4. Vote Coalition Query

"Find stakeholders who voted with Member State X more than 80% of the time"

```sparql
SELECT ?other (COUNT(?shared) AS ?matches) WHERE {
    ?vote1 reg:voter <stakeholder:member-state-x> .
    ?vote1 reg:votePosition ?pos .
    ?vote1 reg:onVote ?voteRecord .

    ?vote2 reg:onVote ?voteRecord .
    ?vote2 reg:voter ?other .
    ?vote2 reg:votePosition ?pos .

    FILTER(?other != <stakeholder:member-state-x>)
} GROUP BY ?other
HAVING(COUNT(?shared) > 5)
ORDER BY DESC(?matches)
```

## JSON Schema Compatibility

Types are designed for easy JSON serialization with struct tags:

```go
type Meeting struct {
    URI        string    `json:"uri"`
    Date       time.Time `json:"date"`
    Status     MeetingStatus `json:"status"`
    // ...
}
```

This enables:
- REST API responses
- Document import/export
- Configuration files
- Test fixtures

## Future Extensions

1. **Temporal SPARQL**: Add `AS_OF`, `BETWEEN` query modifiers
2. **Document Versioning**: Track provision text at each meeting
3. **Automated Parsing**: Extract entities from meeting minutes text
4. **Visualization**: Timeline and graph views
5. **Notifications**: Alert on overdue actions, stalled topics

## Related Issues

- #215 - Living Deliberation Graphs (parent epic)
- #217 - Meeting minutes parser
- #218 - Resolution and decision parser
- #220 - Temporal predicates and versioning
- #235 - Speaker and stakeholder extraction
