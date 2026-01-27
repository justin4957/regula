# Regula

**Automated Regulation Mapper** - Transform dense legal regulations into auditable, queryable, and simulatable programs.

## Vision

Regula ingests complex regulatory documents and produces:
- **Queryable knowledge graphs** via SPARQL/GraphQL
- **Type-safe domain models** with compile-time verification
- **Impact analysis** for regulatory changes
- **Simulation engine** for compliance scenarios
- **Audit trails** with provenance tracking

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    REGULATION INPUT LAYER                        │
│         PDF, XML, Plain Text, Legislative APIs, RSS feeds        │
└───────────────────────────────┬─────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────┐
│                   EXTRACTION PIPELINE                            │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────────┐  │
│  │ Structure   │  │ Semantic     │  │ Cross-Reference        │  │
│  │ Parser      │──│ Extractor    │──│ Resolver               │  │
│  │ (sections)  │  │ (LLM-assist) │  │ (link provisions)      │  │
│  └─────────────┘  └──────────────┘  └────────────────────────┘  │
└───────────────────────────────┬─────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────┐
│                 TYPED DOMAIN MODEL (from lex-sim)                │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌──────────────┐  │
│  │Jurisdiction│ │ Provision  │ │ Authority  │ │  Temporal    │  │
│  │  Types     │ │   Types    │ │   Chain    │ │  Validity    │  │
│  └────────────┘ └────────────┘ └────────────┘ └──────────────┘  │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌──────────────┐  │
│  │ Precedent  │ │ Amendment  │ │Supersession│ │    Proof     │  │
│  │  Binding   │ │  Records   │ │  Tracking  │ │    Types     │  │
│  └────────────┘ └────────────┘ └────────────┘ └──────────────┘  │
└───────────────────────────────┬─────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────┐
│              RDF TRIPLE STORE (from GraphFS)                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Subject          Predicate              Object          │    │
│  │  ─────────────────────────────────────────────────────  │    │
│  │  <GDPR:Art17>     rdf:type               Provision      │    │
│  │  <GDPR:Art17>     reg:grantsRight        Erasure        │    │
│  │  <GDPR:Art17>     reg:supersedes         <Dir95/46:12>  │    │
│  │  <GDPR:Art17>     reg:delegatesTo        <DPA>          │    │
│  │  <GDPR:Art17>     reg:validFrom          2018-05-25     │    │
│  └─────────────────────────────────────────────────────────┘    │
│  Indexes: SPO, POS, OSP for O(1) lookups                         │
└───────────────────────────────┬─────────────────────────────────┘
                                │
        ┌───────────┬───────────┴───────────┬───────────┐
        ▼           ▼                       ▼           ▼
   ┌─────────┐ ┌──────────┐          ┌──────────┐ ┌──────────┐
   │ SPARQL  │ │  IMPACT  │          │SIMULATION│ │  AUDIT   │
   │ QUERIES │ │ ANALYSIS │          │  ENGINE  │ │  TRAIL   │
   │         │ │          │          │          │ │          │
   │"What    │ │"If Art.17│          │"Given    │ │"Prove    │
   │requires │ │changes,  │          │data X,   │ │authority │
   │consent?"│ │what else │          │compliant?│ │chain for │
   │         │ │breaks?"  │          │          │ │decision" │
   └─────────┘ └──────────┘          └──────────┘ └──────────┘
```

## Project Structure

```
regula/
├── cmd/regula/           # CLI application
├── pkg/
│   ├── types/            # Ported lex-sim type system (Go)
│   │   ├── jurisdiction.go
│   │   ├── provision.go
│   │   ├── authority.go
│   │   ├── temporal.go
│   │   ├── precedent.go
│   │   └── proof.go
│   ├── extract/          # Extraction pipeline
│   │   ├── parser.go     # Structure parsing
│   │   ├── semantic.go   # LLM-assisted extraction
│   │   └── resolver.go   # Cross-reference resolution
│   ├── store/            # RDF triple store (from GraphFS)
│   │   ├── triple.go
│   │   ├── store.go
│   │   └── index.go
│   ├── query/            # Query engines
│   │   ├── sparql.go
│   │   └── graphql.go
│   ├── analysis/         # Analysis tools
│   │   ├── impact.go
│   │   ├── compliance.go
│   │   └── conflict.go
│   ├── simulate/         # Simulation engine
│   │   ├── scenario.go
│   │   └── evaluate.go
│   └── audit/            # Audit trail
│       ├── provenance.go
│       └── proof.go
├── internal/
│   └── llm/              # LLM integration
├── examples/
│   ├── gdpr/             # GDPR regulation example
│   └── tax/              # Tax code example
└── docs/
    ├── ROADMAP.md
    └── ARCHITECTURE.md
```

## Key Features

### From lex-sim (Legal Domain Model)
- **Jurisdiction types**: Countries, courts, legal systems, supranational bodies
- **Provision types**: Acts, sections, subsections, schedules, amendments
- **Authority chains**: Delegation, revocation, scope limitations
- **Temporal validity**: When laws are in force, supersession tracking
- **Proof types**: Compile-time verification of legal relationships

### From GraphFS (Graph Infrastructure)
- **RDF triple store**: Efficient storage with SPO/POS/OSP indexes
- **SPARQL queries**: Find provisions, dependencies, conflicts
- **Impact analysis**: What changes when a provision is amended
- **Visualization**: Dependency graphs, Mermaid diagrams

### Novel Components
- **LLM extraction**: Parse dense legal text into structured data
- **Compliance simulation**: "What-if" scenario evaluation
- **Conflict detection**: Find contradictory provisions
- **Audit provenance**: Track reasoning chains for decisions

## Quick Start

```bash
# Initialize a regulation project
regula init gdpr-analysis

# Ingest a regulation document
regula ingest --source gdpr.pdf --format pdf

# Query the regulation graph
regula query "SELECT ?provision WHERE { ?provision reg:requires reg:Consent }"

# Analyze impact of a change
regula impact --provision "GDPR:Art17" --change "remove"

# Simulate a compliance scenario
regula simulate --scenario consent-withdrawal.yaml

# Generate audit trail
regula audit --decision "data-deletion-request-123"
```

## Development Status

See [ROADMAP.md](docs/ROADMAP.md) for detailed development phases.

**Current Phase**: Phase 0 - Foundation Setup

## License

MIT

## Acknowledgments

- **lex-sim**: Legal domain modeling concepts
- **GraphFS**: RDF triple store and query infrastructure
- **Crisp**: Type system design patterns
