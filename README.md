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
- **Draft legislation analysis**: Parse Congressional bills, compute diffs against existing law, detect conflicts, and generate impact reports

## Quick Start

```bash
# Build the CLI
go build -o regula ./cmd/regula

# Or install to $GOPATH/bin
make install

# Ingest a regulation document and display statistics
regula ingest --source testdata/gdpr.txt --stats

# Output:
# Ingesting regulation from: testdata/gdpr.txt
#   1. Parsing document structure... done (11 chapters, 99 articles)
#   2. Extracting defined terms... done (26 definitions)
#   3. Identifying cross-references... done (255 references)
#   4. Building knowledge graph... done (4226 triples)

# Query the regulation graph with SPARQL
regula query --source testdata/gdpr.txt \
  "SELECT ?article ?title WHERE { ?article rdf:type reg:Article . ?article reg:title ?title } LIMIT 5"

# Use built-in query templates
regula query --source testdata/gdpr.txt --template definitions
regula query --source testdata/gdpr.txt --template chapters
regula query --source testdata/gdpr.txt --template references --limit 10

# Output as JSON or CSV
regula query --source testdata/gdpr.txt --template articles --format json --limit 3
regula query --source testdata/gdpr.txt --template articles --format csv --limit 3

# Show query execution time
regula query --source testdata/gdpr.txt --template articles --timing
```

### Query Templates

| Template | Description |
|----------|-------------|
| `articles` | List all articles with titles |
| `definitions` | List all defined terms with their definitions |
| `chapters` | List chapters with article counts |
| `references` | Show cross-references between articles |
| `rights` | Find articles that grant rights |
| `recitals` | List all recitals |
| `article-refs` | Find references from a specific article |
| `search` | Search articles by keyword in title |

### Example Queries

```bash
# Find all articles mentioning "right" in the title
regula query --source testdata/gdpr.txt \
  "SELECT ?article ?title WHERE { ?article rdf:type reg:Article . ?article reg:title ?title . FILTER(CONTAINS(?title, \"Right\")) }"

# Find articles that reference other articles
regula query --source testdata/gdpr.txt \
  "SELECT ?from ?to WHERE { ?from reg:references ?to . ?to rdf:type reg:Article } LIMIT 20"

# List all defined terms
regula query --source testdata/gdpr.txt \
  "SELECT ?term ?text WHERE { ?term rdf:type reg:DefinedTerm . ?term reg:term ?text }"
```

## Draft Legislation Analysis

Analyze Congressional bills against the existing US Code knowledge graph:

```bash
# Parse a draft bill and display structure
regula draft ingest --bill testdata/drafts/hr1234.txt

# Compute diff against existing law
regula draft diff --bill testdata/drafts/hr1234.txt --path .regula

# Run impact analysis (transitive dependencies)
regula draft impact --bill testdata/drafts/hr1234.txt --depth 2

# Detect obligation and rights conflicts
regula draft conflicts --bill testdata/drafts/hr1234.txt

# Run scenario simulation (baseline vs proposed)
regula draft simulate --bill testdata/drafts/hr1234.txt --scenario consent_withdrawal

# Generate full legislative impact report
regula draft report --bill testdata/drafts/hr1234.txt --format markdown
regula draft report --bill testdata/drafts/hr1234.txt --format html --output report.html
```

## Future Commands (Planned)

```bash
# Initialize a regulation project
regula init gdpr-analysis

# Analyze impact of a change
regula impact --provision "GDPR:Art17" --change "remove"

# Simulate a compliance scenario
regula simulate --scenario consent-withdrawal.yaml

# Generate audit trail
regula audit --decision "data-deletion-request-123"
```

## Development Status

See [ROADMAP.md](docs/ROADMAP.md) for detailed development phases.

**Current Phase**: Phase 2 - Knowledge Graph (Complete)

### Completed Milestones
- **M1.1-1.5**: Core type system (jurisdiction, provision, authority, temporal, proof types)
- **M2.1-2.3**: Document parser, definition extractor, reference extractor
- **M2.4**: RDF triple store with SPO/POS/OSP indexes
- **M2.5**: SPARQL query parser and executor with query planning
- **M2.6**: CLI with ingest and query commands

## License

MIT

## Acknowledgments

- **lex-sim**: Legal domain modeling concepts
- **GraphFS**: RDF triple store and query infrastructure
- **Crisp**: Type system design patterns
