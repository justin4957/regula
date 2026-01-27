# Regula Development Roadmap

## Goal
Build an MVP of an automated regulation mapper that can ingest dense regulatory text and produce an auditable, queryable, simulatable program.

## Timeline Overview

| Phase | Focus | Duration | Status |
|-------|-------|----------|--------|
| 0 | Foundation & Setup | 1 week | In Progress |
| 1 | Core Type System | 2 weeks | Pending |
| 2 | RDF Store & Queries | 1 week | Pending |
| 3 | Extraction Pipeline | 2 weeks | Pending |
| 4 | Analysis & Impact | 1 week | Pending |
| 5 | Simulation Engine | 1 week | Pending |
| 6 | MVP Integration | 1 week | Pending |

**Total to MVP: ~9 weeks**

---

## Phase 0: Foundation & Setup (Week 1)

### Goals
- [x] Create repository structure
- [x] Define architecture
- [x] Document roadmap
- [ ] Set up Go module and dependencies
- [ ] Port core type definitions from lex-sim
- [ ] Establish testing infrastructure

### Tasks

#### 0.1 Repository Setup
```bash
regula/
├── go.mod
├── go.sum
├── README.md
├── Makefile
├── .gitignore
├── cmd/regula/main.go
├── pkg/
├── internal/
├── examples/
└── docs/
```

#### 0.2 Dependencies
- `github.com/spf13/cobra` - CLI framework
- `gopkg.in/yaml.v3` - Configuration
- `github.com/stretchr/testify` - Testing

#### 0.3 Core Type Stubs
Create interface definitions for all major types to establish contracts.

### Deliverables
- [ ] Compiling Go project with CLI skeleton
- [ ] Type interface definitions
- [ ] Initial test suite structure

---

## Phase 1: Core Type System (Weeks 2-3)

### Goals
Port the lex-sim type system from Crisp to Go, focusing on:
- Jurisdiction modeling
- Provision structure
- Authority chains
- Temporal validity

### Tasks

#### 1.1 Jurisdiction Types (`pkg/types/jurisdiction.go`)

**From Crisp:**
```crisp
type Country = Country {
  name : String,
  iso_code : String,
  legal_system : LegalSystem
}
```

**To Go:**
```go
type Country struct {
    Name       string
    ISOCode    string
    LegalSystem LegalSystem
}

type LegalSystem int
const (
    CommonLaw LegalSystem = iota
    CivilLaw
    ReligiousLaw
    CustomaryLaw
    MixedLaw
)
```

**Types to port:**
- [ ] `Country`, `State`, `Municipality`
- [ ] `SupranationalBody` (EU, UN, WTO, etc.)
- [ ] `Court`, `CourtLevel`, `CourtHierarchy`
- [ ] `LegalSystem`, `LegalTradition`

#### 1.2 Provision Types (`pkg/types/provision.go`)

**From Crisp:**
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

**To Go:**
```go
type Provision struct {
    ID            ProvisionID
    Number        ProvisionNumber
    Title         *string
    Text          LegalText
    EffectiveDate time.Time
    ExpiryDate    *time.Time
    Jurisdiction  Jurisdiction
    References    []CrossReference
    Definitions   []DefinedTerm
}
```

**Types to port:**
- [ ] `Provision`, `ProvisionID`, `ProvisionNumber`
- [ ] `Section`, `Subsection`, `Paragraph`, `Clause`
- [ ] `Act`, `Regulation`, `Directive`
- [ ] `Schedule`, `Annex`, `Appendix`
- [ ] `LegalText`, `CrossReference`, `DefinedTerm`
- [ ] `Amendment`, `AmendmentRecord`
- [ ] `Commencement`, `CommencementRule`
- [ ] `Repeal`, `RepealRecord`, `SavingsClause`

#### 1.3 Authority Types (`pkg/types/authority.go`)

**From Crisp:**
```crisp
type Authority = Authority {
  holder : AuthorityHolder,
  action : AuthorityAction,
  scope : AuthorityScope,
  source : AuthoritySource,
  delegation_chain : List DelegationRecord
}
```

**To Go:**
```go
type Authority struct {
    Holder          AuthorityHolder
    Action          AuthorityAction
    Scope           AuthorityScope
    Source          AuthoritySource
    DelegationChain []DelegationRecord
    Restrictions    []Restriction
}
```

**Types to port:**
- [ ] `Authority`, `AuthorityHolder`, `AuthorityAction`
- [ ] `AuthorityScope`, `AuthoritySource`
- [ ] `DelegationRecord`, `DelegationChain`
- [ ] `Restriction`, `ScopeLimitation`
- [ ] `Institution`, `Office`, `Committee`

#### 1.4 Temporal Types (`pkg/types/temporal.go`)

**Types to port:**
- [ ] `TemporalRange`, `ValidityPeriod`
- [ ] `AsOf`, `Between`, `Current`, `Historical`
- [ ] `SupersessionRecord`
- [ ] `AmendmentTimeline`

#### 1.5 Precedent Types (`pkg/types/precedent.go`)

**Types to port:**
- [ ] `Precedent`, `CaseDecision`
- [ ] `Citation`, `CitationFormat`
- [ ] `Holding`, `RatioDecidendi`, `ObiterDictum`
- [ ] `PrecedentRelationship` (Follows, Distinguishes, Overrules, etc.)
- [ ] `BindingForce` (Mandatory, Persuasive, Foreign)

#### 1.6 Proof Types (`pkg/types/proof.go`)

**From Crisp:**
```crisp
type Proof a where
  BindsOn : Court -> Court -> Proof (BindsOn higher lower)
  HasJurisdiction : Court -> Matter -> Proof (HasJurisdiction court matter)
  IsGoodLaw : Citation -> Date -> Proof (IsGoodLaw citation date)
```

**To Go (using interfaces):**
```go
type Proof interface {
    Verify() error
    ProofType() string
}

type BindsOnProof struct {
    HigherCourt Court
    LowerCourt  Court
}

func (p BindsOnProof) Verify() error {
    // Verify hierarchical relationship
}
```

### Deliverables
- [ ] Complete type system in `pkg/types/`
- [ ] Unit tests for all types
- [ ] Type validation functions
- [ ] JSON/YAML serialization

---

## Phase 2: RDF Store & Queries (Week 4)

### Goals
Port GraphFS triple store and adapt for regulation domain.

### Tasks

#### 2.1 Triple Store (`pkg/store/`)
- [ ] Port `Triple` type from GraphFS
- [ ] Port `TripleStore` with SPO/POS/OSP indexes
- [ ] Add regulation-specific predicates
- [ ] Thread-safe operations

#### 2.2 Regulation Predicates (`pkg/store/predicates.go`)
```go
const (
    // Core relationships
    RDFType        = "rdf:type"
    RegAmends      = "reg:amends"
    RegSupersedes  = "reg:supersedes"
    RegReferences  = "reg:references"
    RegDelegatesTo = "reg:delegatesTo"

    // Temporal
    RegValidFrom   = "reg:validFrom"
    RegValidUntil  = "reg:validUntil"

    // Authority
    RegGrantsRight    = "reg:grantsRight"
    RegImposesObligation = "reg:imposesObligation"
    RegRequires       = "reg:requires"

    // Hierarchy
    RegPartOf      = "reg:partOf"
    RegContains    = "reg:contains"
)
```

#### 2.3 SPARQL Engine (`pkg/query/sparql.go`)
- [ ] Port SPARQL parser from GraphFS
- [ ] SELECT, WHERE, FILTER support
- [ ] Query optimization
- [ ] Result streaming

#### 2.4 Query Templates (`pkg/query/templates.go`)
Pre-built queries for common regulation questions:
```go
var QueryTemplates = map[string]string{
    "provisions-requiring-consent": `
        SELECT ?provision ?text WHERE {
            ?provision rdf:type reg:Provision .
            ?provision reg:requires reg:Consent .
            ?provision reg:text ?text .
        }
    `,
    "amendments-since": `
        SELECT ?amendment ?target ?date WHERE {
            ?amendment rdf:type reg:Amendment .
            ?amendment reg:amends ?target .
            ?amendment reg:effectiveDate ?date .
            FILTER(?date > "%s")
        }
    `,
}
```

### Deliverables
- [ ] Working triple store
- [ ] SPARQL query engine
- [ ] CLI `query` command
- [ ] Query template library

---

## Phase 3: Extraction Pipeline (Weeks 5-6)

### Goals
Build the pipeline to extract structured data from regulatory text.

### Tasks

#### 3.1 Document Parser (`pkg/extract/parser.go`)
- [ ] PDF text extraction
- [ ] XML/HTML parsing (legislative markup)
- [ ] Plain text handling
- [ ] Structure detection (sections, subsections)

#### 3.2 Semantic Extractor (`pkg/extract/semantic.go`)
LLM-assisted extraction:
```go
type ExtractionPrompt struct {
    SystemPrompt string
    Document     string
    Schema       string  // JSON schema for output
}

type ExtractedProvision struct {
    Number      string
    Title       string
    Text        string
    References  []string
    Definitions []DefinedTermExtract
    Obligations []string
    Rights      []string
}
```

**Extraction tasks:**
- [ ] Provision boundaries
- [ ] Cross-references (internal and external)
- [ ] Defined terms
- [ ] Obligations and rights
- [ ] Authority delegations
- [ ] Temporal conditions

#### 3.3 Cross-Reference Resolver (`pkg/extract/resolver.go`)
- [ ] Parse reference patterns ("Article 17", "Section 5(2)(a)")
- [ ] Resolve internal references
- [ ] Resolve external references (other acts)
- [ ] Handle ambiguous references

#### 3.4 Validation Layer (`pkg/extract/validate.go`)
- [ ] Schema validation
- [ ] Consistency checks
- [ ] Human review queue for uncertain extractions

### Deliverables
- [ ] Working extraction pipeline
- [ ] CLI `ingest` command
- [ ] Extraction quality metrics
- [ ] Human review interface (CLI)

---

## Phase 4: Analysis & Impact (Week 7)

### Goals
Build analysis tools for understanding regulatory relationships.

### Tasks

#### 4.1 Dependency Graph (`pkg/analysis/dependency.go`)
- [ ] Port graph algorithms from GraphFS
- [ ] Topological sort of provisions
- [ ] Strongly connected components (circular dependencies)
- [ ] Shortest path between provisions

#### 4.2 Impact Analysis (`pkg/analysis/impact.go`)
```go
type ImpactAnalysis struct {
    TargetProvision ProvisionID
    ChangeType      ChangeType  // Amend, Repeal, Add
    DirectImpact    []ProvisionID
    TransitiveImpact []ProvisionID
    RiskLevel       RiskLevel
    Recommendations []string
}
```

- [ ] Direct impact (provisions referencing target)
- [ ] Transitive impact (provisions affected transitively)
- [ ] Risk assessment
- [ ] Conflict detection

#### 4.3 Conflict Detection (`pkg/analysis/conflict.go`)
- [ ] Contradictory provisions
- [ ] Overlapping jurisdiction
- [ ] Temporal conflicts
- [ ] Authority conflicts

### Deliverables
- [ ] CLI `impact` command
- [ ] CLI `conflicts` command
- [ ] Impact visualization

---

## Phase 5: Simulation Engine (Week 8)

### Goals
Enable "what-if" scenario evaluation against the regulation graph.

### Tasks

#### 5.1 Scenario Definition (`pkg/simulate/scenario.go`)
```yaml
# consent-withdrawal.yaml
scenario: "User withdraws consent for data processing"
given:
  - entity: "DataSubject"
    action: "withdraws"
    object: "Consent"
    context:
      data_type: "personal_data"
      processing_purpose: "marketing"
when:
  - trigger: "consent_withdrawal_received"
then:
  - check: "obligations_triggered"
  - check: "rights_applicable"
  - check: "timeline_requirements"
```

#### 5.2 Scenario Evaluator (`pkg/simulate/evaluate.go`)
- [ ] Parse scenario YAML
- [ ] Query regulation graph for applicable provisions
- [ ] Evaluate conditions
- [ ] Generate compliance report

#### 5.3 Compliance Report (`pkg/simulate/report.go`)
```go
type ComplianceReport struct {
    Scenario        string
    ApplicableProvisions []ProvisionID
    Obligations     []Obligation
    Rights          []Right
    Deadlines       []Deadline
    RiskAreas       []RiskArea
    Recommendations []string
}
```

### Deliverables
- [ ] CLI `simulate` command
- [ ] Scenario YAML schema
- [ ] Compliance report generation

---

## Phase 6: MVP Integration (Week 9)

### Goals
Integrate all components into a working end-to-end system.

### Tasks

#### 6.1 CLI Polish
- [ ] Consistent command structure
- [ ] Help documentation
- [ ] Shell completions
- [ ] Output formatting (table, JSON, YAML)

#### 6.2 End-to-End Example
Create complete GDPR example:
- [ ] Ingest GDPR text
- [ ] Extract provisions
- [ ] Build regulation graph
- [ ] Run example queries
- [ ] Perform impact analysis
- [ ] Simulate consent withdrawal scenario

#### 6.3 Documentation
- [ ] User guide
- [ ] API documentation
- [ ] Example walkthroughs

#### 6.4 Testing
- [ ] Integration tests
- [ ] Example validation
- [ ] Performance benchmarks

### Deliverables
- [ ] Working MVP with GDPR example
- [ ] Complete documentation
- [ ] Demo video/walkthrough

---

## Post-MVP Phases

### Phase 7: Web Interface
- GraphQL API
- Web dashboard
- Interactive graph explorer

### Phase 8: Advanced Extraction
- Multi-document correlation
- Historical version tracking
- Amendment auto-detection

### Phase 9: Multi-Jurisdiction
- Cross-jurisdictional mapping
- Equivalence detection
- Conflict resolution

### Phase 10: Production Hardening
- Database persistence (PostgreSQL)
- Authentication/authorization
- Audit logging
- Performance optimization

---

## Success Metrics

### MVP Success Criteria
1. **Ingest**: Can process a 100+ page regulation PDF
2. **Extract**: >80% accuracy on provision boundary detection
3. **Query**: <100ms response for typical SPARQL queries
4. **Impact**: Correctly identifies direct and transitive dependencies
5. **Simulate**: Evaluates basic compliance scenarios

### Quality Gates
- All tests passing
- Documentation complete
- Example workflows functional
- Code review completed

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| LLM extraction accuracy | Human review queue, iterative prompts |
| Complex reference patterns | Extensible parser, pattern library |
| Performance at scale | Efficient indexes, query optimization |
| Legal domain complexity | Start with single regulation (GDPR) |

---

## Getting Started

```bash
# Clone repository
git clone <repo-url>
cd regula

# Install dependencies
go mod download

# Run tests
go test ./...

# Build CLI
go build -o regula ./cmd/regula

# Try example
./regula init gdpr-example
./regula ingest --source examples/gdpr/gdpr.txt
./regula query "SELECT ?p WHERE { ?p rdf:type reg:Provision }"
```
