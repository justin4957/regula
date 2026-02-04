# Regula Architecture

## Overview

Regula combines three architectural influences:

1. **lex-sim** (Crisp): Legal domain modeling with type-safe verification
2. **GraphFS** (Go): RDF triple store with SPARQL queries and impact analysis
3. **Crisp type system** (Haskell): Sophisticated type checking patterns

This document details how these are synthesized into a unified architecture.

---

## Layer Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         CLI / API Layer                              │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐       │
│  │ ingest  │ │  query  │ │ impact  │ │simulate │ │  audit  │       │
│  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘       │
└───────┼──────────┼─────────┼─────────┼─────────┼───────────────────┘
        │          │         │         │         │
┌───────▼──────────▼─────────▼─────────▼─────────▼───────────────────┐
│                      Service Layer                                   │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐                 │
│  │  Extraction  │ │   Analysis   │ │  Simulation  │                 │
│  │   Service    │ │   Service    │ │   Service    │                 │
│  └──────┬───────┘ └──────┬───────┘ └──────┬───────┘                 │
└─────────┼────────────────┼────────────────┼─────────────────────────┘
          │                │                │
┌─────────▼────────────────▼────────────────▼─────────────────────────┐
│                      Domain Layer                                    │
│  ┌────────────────────────────────────────────────────────────┐     │
│  │                    Type System (from lex-sim)               │     │
│  │  Jurisdiction │ Provision │ Authority │ Temporal │ Proof   │     │
│  └────────────────────────────────────────────────────────────┘     │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
┌─────────────────────────────────▼───────────────────────────────────┐
│                     Storage Layer                                    │
│  ┌────────────────────────────────────────────────────────────┐     │
│  │              RDF Triple Store (from GraphFS)                │     │
│  │     SPO Index │ POS Index │ OSP Index │ Query Engine        │     │
│  └────────────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Domain Type System

### Type Mapping: Crisp → Go

The lex-sim type system is written in Crisp. Here's how we translate to Go:

#### Sum Types (Enums)

**Crisp:**
```crisp
type LegalSystem =
  | CommonLaw
  | CivilLaw
  | ReligiousLaw { tradition : String }
  | CustomaryLaw
  | MixedLaw { components : List LegalSystem }
```

**Go:**
```go
type LegalSystemKind int

const (
    LegalSystemCommonLaw LegalSystemKind = iota
    LegalSystemCivilLaw
    LegalSystemReligiousLaw
    LegalSystemCustomaryLaw
    LegalSystemMixedLaw
)

type LegalSystem struct {
    Kind       LegalSystemKind
    Tradition  string         // for ReligiousLaw
    Components []LegalSystem  // for MixedLaw
}

func CommonLaw() LegalSystem {
    return LegalSystem{Kind: LegalSystemCommonLaw}
}

func ReligiousLaw(tradition string) LegalSystem {
    return LegalSystem{Kind: LegalSystemReligiousLaw, Tradition: tradition}
}
```

#### Product Types (Records)

**Crisp:**
```crisp
type Provision = Provision {
  id : ProvisionId,
  number : ProvisionNumber,
  title : Option String,
  text : LegalText,
  effective_date : Date,
  jurisdiction : Jurisdiction
}
```

**Go:**
```go
type Provision struct {
    ID            ProvisionID
    Number        ProvisionNumber
    Title         *string  // nil = None
    Text          LegalText
    EffectiveDate time.Time
    Jurisdiction  Jurisdiction
}
```

#### Proof Types

**Crisp (from lex-sim):**
```crisp
type Proof a where
  BindsOn : Court -> Court -> Proof (BindsOn higher lower)
  HasJurisdiction : Court -> Matter -> Proof (HasJurisdiction court matter)
  IsGoodLaw : Citation -> Date -> Proof (IsGoodLaw citation date)
  HasAuthority : Holder -> Action -> Scope -> Proof (HasAuthority holder action scope)
```

**Go (interface-based):**
```go
// Proof is a verified assertion about legal relationships
type Proof interface {
    // Verify checks if the proof is valid
    Verify(ctx *VerificationContext) error

    // ProofType returns the type of proof
    ProofType() string

    // Evidence returns the chain of reasoning
    Evidence() []Evidence
}

// BindsOnProof proves that one court binds another
type BindsOnProof struct {
    HigherCourt Court
    LowerCourt  Court
    Hierarchy   CourtHierarchy
}

func (p *BindsOnProof) Verify(ctx *VerificationContext) error {
    // Check that HigherCourt is actually above LowerCourt in the hierarchy
    if !p.Hierarchy.IsAbove(p.HigherCourt, p.LowerCourt) {
        return fmt.Errorf("%s does not bind %s in hierarchy %s",
            p.HigherCourt.Name, p.LowerCourt.Name, p.Hierarchy.Name)
    }
    return nil
}

// ProveBindsOn attempts to construct a BindsOnProof
// Returns nil if the relationship cannot be proven
func ProveBindsOn(higher, lower Court, h CourtHierarchy) *BindsOnProof {
    proof := &BindsOnProof{
        HigherCourt: higher,
        LowerCourt:  lower,
        Hierarchy:   h,
    }
    if proof.Verify(nil) != nil {
        return nil
    }
    return proof
}
```

---

## RDF Schema

### Namespaces

```turtle
@prefix rdf:  <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix xsd:  <http://www.w3.org/2001/XMLSchema#> .
@prefix reg:  <https://regula.dev/ontology#> .
@prefix prov: <http://www.w3.org/ns/prov#> .
```

### Core Classes

```turtle
# Structural
reg:Provision      rdfs:subClassOf rdfs:Resource .
reg:Act            rdfs:subClassOf reg:Provision .
reg:Section        rdfs:subClassOf reg:Provision .
reg:Subsection     rdfs:subClassOf reg:Provision .
reg:Schedule       rdfs:subClassOf reg:Provision .

# Jurisdictional
reg:Jurisdiction   rdfs:subClassOf rdfs:Resource .
reg:Country        rdfs:subClassOf reg:Jurisdiction .
reg:Court          rdfs:subClassOf rdfs:Resource .

# Authority
reg:Authority      rdfs:subClassOf rdfs:Resource .
reg:Delegation     rdfs:subClassOf rdfs:Resource .

# Temporal
reg:TemporalEntity rdfs:subClassOf rdfs:Resource .
reg:Amendment      rdfs:subClassOf reg:TemporalEntity .
```

### Core Predicates

```turtle
# Structural relationships
reg:partOf         rdfs:domain reg:Provision ; rdfs:range reg:Provision .
reg:contains       rdfs:domain reg:Provision ; rdfs:range reg:Provision .
reg:references     rdfs:domain reg:Provision ; rdfs:range reg:Provision .

# Amendment relationships
reg:amends         rdfs:domain reg:Amendment ; rdfs:range reg:Provision .
reg:supersedes     rdfs:domain reg:Provision ; rdfs:range reg:Provision .
reg:repeals        rdfs:domain reg:Provision ; rdfs:range reg:Provision .

# Authority relationships
reg:delegatesTo    rdfs:domain reg:Provision ; rdfs:range reg:Authority .
reg:grantsRight    rdfs:domain reg:Provision ; rdfs:range reg:Right .
reg:imposesObligation rdfs:domain reg:Provision ; rdfs:range reg:Obligation .

# Temporal properties
reg:validFrom      rdfs:domain reg:Provision ; rdfs:range xsd:date .
reg:validUntil     rdfs:domain reg:Provision ; rdfs:range xsd:date .

# Jurisdictional
reg:jurisdiction   rdfs:domain reg:Provision ; rdfs:range reg:Jurisdiction .

# Content
reg:text           rdfs:domain reg:Provision ; rdfs:range xsd:string .
reg:title          rdfs:domain reg:Provision ; rdfs:range xsd:string .
reg:number         rdfs:domain reg:Provision ; rdfs:range xsd:string .
```

---

## Extraction Pipeline

### Stage 1: Document Parsing

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│    PDF      │     │    XML      │     │   Plain     │
│   Parser    │     │   Parser    │     │   Text      │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       └───────────────────┼───────────────────┘
                           │
                           ▼
                  ┌─────────────────┐
                  │ Unified Document │
                  │    Structure     │
                  └────────┬────────┘
                           │
```

### Stage 2: Semantic Extraction (LLM-Assisted)

```go
type ExtractionRequest struct {
    Document    string
    Schema      ExtractionSchema
    Context     ExtractionContext
}

type ExtractionSchema struct {
    // What to extract
    ExtractProvisions   bool
    ExtractDefinitions  bool
    ExtractReferences   bool
    ExtractObligations  bool
    ExtractRights       bool
    ExtractDelegations  bool
}

type ExtractionContext struct {
    // Regulation metadata
    RegulationName string
    Jurisdiction   string
    EffectiveDate  time.Time

    // Already extracted items (for reference resolution)
    KnownProvisions  []ProvisionID
    KnownDefinitions map[string]string
}
```

### Stage 3: Reference Resolution

```
┌─────────────────────────────────────────────────────────────┐
│                   Reference Patterns                         │
├─────────────────────────────────────────────────────────────┤
│ Internal: "Article 17", "Section 5(2)(a)", "paragraph 3"    │
│ External: "Directive 95/46/EC", "GDPR Article 4"            │
│ Defined:  "personal data" (as defined in Article 4)         │
└─────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                   Resolution Strategy                        │
├─────────────────────────────────────────────────────────────┤
│ 1. Normalize reference format                               │
│ 2. Search current document                                  │
│ 3. Search linked regulations                                │
│ 4. Flag unresolved for human review                         │
└─────────────────────────────────────────────────────────────┘
```

---

## Draft Legislation Pipeline

The draft legislation pipeline (`pkg/draft/`) parses proposed Congressional bills, recognizes amendment instructions, and computes a structured diff against the existing USC knowledge graph. The pipeline follows conventions from the [Office of Legislative Counsel Drafting Manual](https://legcounsel.house.gov/HOLC/Drafting_Legislation/drafting-guide.html).

### Pipeline Overview

```
                    Draft Bill Text (HR/S)
                            │
                            ▼
                 ┌─────────────────────┐
                 │    Bill Parser       │  pkg/draft/parser.go
                 │  ParseBillFromFile() │  Extracts metadata, splits sections
                 └──────────┬──────────┘
                            │
                   DraftBill with []DraftSection
                            │
                            ▼
                 ┌─────────────────────┐
                 │  Amendment          │  pkg/draft/patterns.go
                 │  Recognizer         │  Classifies and extracts amendments
                 │  ExtractAmendments()│  per section
                 └──────────┬──────────┘
                            │
                   DraftBill with []Amendment per section
                            │
                            ▼
                 ┌─────────────────────┐
                 │    Diff Engine       │  pkg/draft/diff.go
                 │    ComputeDiff()     │  Resolves targets, loads triple
                 │                      │  stores, classifies changes
                 └──────────┬──────────┘
                            │
                            ▼
                   DraftDiff (Added/Removed/Modified/
                   Redesignated/Unresolved + triple counts)
```

### Amendment Pattern Grammar

Congressional bills amend existing law using standardized phrasing. The recognizer matches six amendment types, checked in priority order (most specific first):

| Priority | Type | Pattern | Example |
|----------|------|---------|---------|
| 1 | Strike-and-insert | `striking "X" and inserting "Y"` | `is amended by striking "13" and inserting "16"` |
| 2 | Redesignation | `redesignating {unit} (X) as {unit} (Y)` | `is amended by redesignating paragraph (13A) as paragraph (13B)` |
| 3 | Table of contents | `table of contents ... is amended` | `The table of contents for chapter 5 of title 42 is amended` |
| 4 | Add new section | `inserting after section X the following new section` | `is amended by inserting after section 523 the following new section:` |
| 5 | Add at end | `adding at the end the following` | `is amended by adding at the end the following:` |
| 6 | Repeal | `is repealed` / `is hereby repealed` | `Section 230(c)(1) of title 47 is repealed` |

Priority ordering prevents ambiguity: a strike-insert is checked before repeal because both may contain "is amended" preambles, and redesignation is checked before add-at-end because both may reference subsection targets.

### Target Reference Resolution

Amendment targets identify provisions using two formats. The recognizer tries the parenthetical format first, then the prose format:

**Parenthetical (U.S.C. citation):**
```
(15 U.S.C. 6502)           → Title 15, Section 6502
(15 U.S.C. 6505(d))        → Title 15, Section 6505, Subsection (d)
(11 U.S.C. 101 et seq.)    → Title 11, Section 101
```

**Prose ("Section X of title Y"):**
```
Section 6502(b)(1) of title 15, United States Code
Section 101 of title 11 of the United States Code
```

Resolved targets map to the knowledge graph as:
```
Title number  →  Document ID:  us-usc-title-{N}
Section ref   →  Target URI:   {baseURI}US-USC-TITLE-{N}:Art{section}
Subsection    →  Target URI:   {baseURI}US-USC-TITLE-{N}:Art{section}({subsection})
```

For example, "Section 6502(b)(1) of title 15" resolves to:
- Document ID: `us-usc-title-15`
- Target URI: `https://regula.dev/regulations/US-USC-TITLE-15:Art6502((b)(1))`

### Linguistic Variations

The recognizer handles common drafting variations:

| Variation | Examples |
|-----------|----------|
| Amendment preamble | `"is amended—"` (em dash), `"is amended--"` (double hyphen), `"is amended by"` |
| Title reference | `"of title 42, United States Code"`, `"of title 42 of the United States Code"` |
| Repeal emphasis | `"is repealed"`, `"is hereby repealed"` |
| Quoted text delimiters | Straight quotes `"..."` and curly quotes `\u201c...\u201d` |
| Redesignation units | `paragraph`, `subsection`, `section`, `subparagraph`, `clause` |
| Numbered clauses | `"(1) by striking..."`, `"(2) by inserting..."` within compound amendments |
| Lettered sub-items | `"(A) in paragraph (1)..."`, `"(B) in paragraph (2)..."` |

### Compound Amendments

A single bill section may contain multiple amendments targeting different provisions. The recognizer splits these by detecting numbered clause boundaries:

```
Section 6502 of title 15, United States Code, is amended—
    (1) in <paragraph (1), by striking "13" and inserting "16";>
    (2) <by adding at the end the following:>
        "(C) PROHIBITION ON TARGETED ADVERTISING.—..."
```

Each numbered clause `(1)`, `(2)`, etc. is processed independently. The target title and section from the preamble carry forward into each clause.

### Diff Classification

The diff engine (`ComputeDiff`) resolves each amendment target against the library, loads the document's triple store, and classifies the change:

| Amendment Type | Diff Category | ProposedText | ExistingText |
|----------------|---------------|-------------|--------------|
| `strike_insert` | Modified | Insert text | Current provision text |
| `repeal` | Removed | — | Current provision text |
| `add_new_section` | Added | New section text | — |
| `add_at_end` | Added | Appended text | — |
| `redesignate` | Redesignated | New designation | — |
| `table_of_contents` | Modified | Updated TOC entry | Current TOC text |

For each resolved target, the engine also counts:
- **Affected triples**: triples where the target URI appears as subject or object
- **Cross-references**: provisions referencing or referenced by the target (via `reg:references` and `reg:referencedBy`)
- **Unresolved targets**: amendments targeting provisions not found in the knowledge graph

---

## Query Engine

### SPARQL Support

Based on GraphFS query engine, supporting:

```sparql
# Basic queries
SELECT ?provision ?title WHERE {
    ?provision rdf:type reg:Provision .
    ?provision reg:title ?title .
}

# Filtering
SELECT ?provision WHERE {
    ?provision rdf:type reg:Provision .
    ?provision reg:validFrom ?date .
    FILTER(?date > "2018-05-25"^^xsd:date)
}

# Path queries
SELECT ?affected WHERE {
    <GDPR:Art17> reg:references+ ?affected .
}

# OPTIONAL
SELECT ?provision ?amendment WHERE {
    ?provision rdf:type reg:Provision .
    OPTIONAL { ?amendment reg:amends ?provision }
}
```

### Query Templates

Pre-built queries for common regulation questions:

| Template | Description |
|----------|-------------|
| `provisions-by-topic` | Find provisions related to a topic |
| `amendments-timeline` | Get amendment history for a provision |
| `authority-chain` | Trace delegation chain |
| `cross-references` | Find all references to/from a provision |
| `obligations-by-actor` | Find obligations for a specific actor type |

---

## Analysis Engine

### Impact Analysis Algorithm

```go
func AnalyzeImpact(store *TripleStore, target ProvisionID, change ChangeType) *ImpactAnalysis {
    result := &ImpactAnalysis{
        TargetProvision: target,
        ChangeType:      change,
    }

    // 1. Find direct references
    directRefs := store.Query(`
        SELECT ?referrer WHERE {
            ?referrer reg:references <%s> .
        }
    `, target)
    result.DirectImpact = directRefs

    // 2. Find transitive references (configurable depth)
    transitiveRefs := store.Query(`
        SELECT ?referrer WHERE {
            ?referrer reg:references+ <%s> .
        }
    `, target)
    result.TransitiveImpact = transitiveRefs

    // 3. Find authority delegations
    delegations := store.Query(`
        SELECT ?delegation WHERE {
            <%s> reg:delegatesTo ?delegation .
        }
    `, target)
    result.AffectedDelegations = delegations

    // 4. Assess risk
    result.RiskLevel = assessRisk(result)

    return result
}
```

### Conflict Detection

```go
type ConflictType int

const (
    ConflictContradiction   ConflictType = iota  // A says X, B says not X
    ConflictOverlap                               // A and B both regulate same thing differently
    ConflictTemporal                              // A valid when B also valid, conflict
    ConflictAuthority                             // A delegates to X, B revokes from X
)

type Conflict struct {
    Type        ConflictType
    Provisions  []ProvisionID
    Description string
    Severity    Severity
    Resolution  *string  // Suggested resolution if determinable
}
```

---

## Simulation Engine

### Scenario DSL

```yaml
scenario:
  name: "Consent Withdrawal"
  description: "User exercises right to withdraw consent"

  # Initial state
  given:
    - entity: DataSubject
      id: "user-123"
      attributes:
        consent_given: true
        consent_date: "2023-01-15"
        data_categories: ["email", "name", "preferences"]

    - entity: DataController
      id: "company-456"
      attributes:
        processing_purposes: ["marketing", "analytics"]

  # Trigger event
  when:
    action: withdraw_consent
    actor: DataSubject
    target: DataController
    parameters:
      scope: "all"

  # Expected outcomes
  then:
    - check: applicable_provisions
      expect:
        - "GDPR:Art7(3)"   # Right to withdraw
        - "GDPR:Art17"     # Right to erasure

    - check: obligations_triggered
      for: DataController
      expect:
        - type: cease_processing
          deadline: "without undue delay"
        - type: notify_processors
          deadline: "without undue delay"

    - check: timeline
      expect:
        - action: acknowledge_withdrawal
          deadline: "1 month"
        - action: complete_erasure
          deadline: "1 month"
```

### Evaluation Engine

```go
type EvaluationResult struct {
    Scenario            string
    ApplicableProvisions []ProvisionMatch
    TriggeredObligations []Obligation
    TriggeredRights      []Right
    Timelines           []TimelineRequirement
    Compliance          ComplianceStatus
    Gaps                []ComplianceGap
}

type ProvisionMatch struct {
    Provision   ProvisionID
    MatchReason string
    Confidence  float64
}

type ComplianceGap struct {
    Requirement string
    Missing     string
    Risk        Severity
    Suggestion  string
}
```

---

## Audit Trail

### Provenance Tracking

Every operation is tracked with W3C PROV ontology:

```turtle
<decision:123> a prov:Activity ;
    prov:wasAssociatedWith <user:admin> ;
    prov:startedAtTime "2024-01-15T10:30:00Z" ;
    prov:used <provision:GDPR-Art17> ;
    prov:used <scenario:consent-withdrawal> ;
    prov:generated <report:compliance-456> .

<report:compliance-456> a prov:Entity ;
    prov:wasGeneratedBy <decision:123> ;
    prov:wasDerivedFrom <provision:GDPR-Art17> ;
    prov:generatedAtTime "2024-01-15T10:30:05Z" .
```

### Proof Chain

```go
type AuditEntry struct {
    ID          string
    Timestamp   time.Time
    Action      string
    Actor       string
    Inputs      []ResourceRef
    Outputs     []ResourceRef
    Proofs      []Proof
    Reasoning   []ReasoningStep
}

type ReasoningStep struct {
    Step        int
    Description string
    Provisions  []ProvisionID
    Conclusion  string
}
```

---

## Performance Considerations

### Triple Store Indexes

From GraphFS, three indexes for O(1) lookups:

| Index | Pattern | Use Case |
|-------|---------|----------|
| SPO | Subject → Predicate → Object | "What does provision X reference?" |
| POS | Predicate → Object → Subject | "What provisions are of type Section?" |
| OSP | Object → Subject → Predicate | "What references provision Y?" |

### Query Optimization

- **Pattern ordering**: Most selective patterns first
- **Index selection**: Choose best index for each pattern
- **Join strategy**: Hash join for large result sets
- **Caching**: LRU cache for frequent queries

### Expected Performance

| Operation | Target | Notes |
|-----------|--------|-------|
| Triple insertion | 10,000/sec | Batch mode |
| Simple query | <10ms | Single pattern |
| Complex query | <100ms | 5+ patterns |
| Impact analysis | <1s | Full transitive closure |
| Simulation | <5s | Complete scenario |

---

## Extension Points

### Custom Extractors

```go
type Extractor interface {
    // Name returns the extractor identifier
    Name() string

    // CanHandle returns true if this extractor handles the format
    CanHandle(format DocumentFormat) bool

    // Extract processes a document and returns structured data
    Extract(doc Document, schema ExtractionSchema) (*ExtractionResult, error)
}
```

### Custom Analyzers

```go
type Analyzer interface {
    // Name returns the analyzer identifier
    Name() string

    // Analyze performs analysis on the regulation graph
    Analyze(store *TripleStore, config AnalysisConfig) (*AnalysisResult, error)
}
```

### Custom Predicates

New predicates can be registered:

```go
store.RegisterPredicate(Predicate{
    URI:         "https://example.com/custom#requires",
    Domain:      "reg:Provision",
    Range:       "reg:Requirement",
    Description: "Custom requirement relationship",
})
```
