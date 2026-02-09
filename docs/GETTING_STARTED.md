# Getting Started with Regula

This comprehensive guide walks you through Regula's capabilities for exploring and analyzing legal regulations. Includes real CLI output examples.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Ingesting Documents](#ingesting-documents)
3. [Querying the Knowledge Graph](#querying-the-knowledge-graph)
4. [Impact Analysis](#impact-analysis)
5. [Compliance Scenarios](#compliance-scenarios)
6. [Validation](#validation)
7. [Export Formats](#export-formats)
8. [US Code Analysis](#us-code-analysis)
9. [Parliamentary Rules](#parliamentary-rules)
10. [Draft Legislation](#draft-legislation)

---

## Quick Start

### Build from Source

```bash
git clone https://github.com/justin4957/regula.git
cd regula
go build -o regula ./cmd/regula
```

### First Command

```
$ ./regula --help

Regula transforms dense legal regulations into auditable,
queryable, and simulatable programs.

It ingests regulatory documents and produces:
  - Queryable knowledge graphs via SPARQL
  - Type-safe domain models with compile-time verification
  - Impact analysis for regulatory changes
  - Simulation engine for compliance scenarios
  - Audit trails with provenance tracking

Available Commands:
  bulk        Bulk download and ingest legislation from official sources
  compare     Compare regulation documents or House Rules versions
  draft       Draft legislation analysis pipeline
  export      Export the relationship graph for visualization
  impact      Analyze impact of regulatory changes
  ingest      Ingest a regulation document
  library     Manage the legislation library
  match       Match a scenario to applicable provisions
  playground  USC triple store analysis playground
  query       Query the regulation graph
  validate    Validate graph consistency and extraction quality
```

---

## Ingesting Documents

### Basic Ingestion

```
$ ./regula ingest --source testdata/gdpr.txt --stats

Ingesting regulation from: testdata/gdpr.txt
  1. Parsing document structure... done (11 chapters, 99 articles)
  2. Extracting defined terms... done (26 definitions)
  3. Identifying cross-references... done (258 references)
  4. Extracting rights/obligations... done (60 rights, 69 obligations)
  5. Resolving cross-references... done (99% resolved)
  6. Building knowledge graph... done (8495 triples)

Ingestion complete in 1.507859334s

Graph Statistics:
  Total triples:    8495
  Articles:         99
  Chapters:         11
  Sections:         15
  Recitals:         173
  Definitions:      26
  References:       258
  Rights:           60
  Obligations:      70
  Term usages:      349
```

### Ingestion Options

```
$ ./regula ingest --help

Flags:
  --base-uri string    Base URI for the graph (default "https://regula.dev/regulations/")
  --fetch-refs         Fetch external referenced documents to build a federated graph
  --gates              Enable validation gates during ingestion
  --max-depth int      Maximum recursion depth for fetching external references (default 2)
  -o, --output string  Output graph file (JSON)
  -s, --source string  Source document path
  --stats              Show detailed statistics
```

---

## Querying the Knowledge Graph

### SPARQL Queries

List articles with titles:

```
$ ./regula query --source testdata/gdpr.txt \
  "SELECT ?article ?title WHERE { ?article rdf:type reg:Article . ?article reg:title ?title } LIMIT 10"

+-------------------------------------------+-----------------------------------------------------------+
| article                                   | title                                                     |
+-------------------------------------------+-----------------------------------------------------------+
| https://regula.dev/regulations/GDPR:Art72 | Procedure                                                 |
| https://regula.dev/regulations/GDPR:Art75 | Secretariat                                               |
| https://regula.dev/regulations/GDPR:Art22 | Automated individual decision-making, including profiling |
| https://regula.dev/regulations/GDPR:Art38 | Position of the data protection officer                   |
| https://regula.dev/regulations/GDPR:Art62 | Joint operations of supervisory authorities               |
| https://regula.dev/regulations/GDPR:Art80 | Representation of data subjects                           |
| https://regula.dev/regulations/GDPR:Art84 | Penalties                                                 |
| https://regula.dev/regulations/GDPR:Art86 | Processing and public access to official documents        |
| https://regula.dev/regulations/GDPR:Art44 | General principle for transfers                           |
| https://regula.dev/regulations/GDPR:Art49 | Derogations for specific situations                       |
+-------------------------------------------+-----------------------------------------------------------+
10 rows
```

### Query Templates

```
$ ./regula query --source testdata/gdpr.txt --template definitions

Template: definitions
Description: List all defined terms with their definitions

+-----------------------------------------------+---------------------------------+
| term                                          | termText                        |
+-----------------------------------------------+---------------------------------+
| GDPR:Term:binding_corporate_rules             | binding corporate rules         |
| GDPR:Term:biometric_data                      | biometric data                  |
| GDPR:Term:consent                             | consent                         |
| GDPR:Term:controller                          | controller                      |
| GDPR:Term:personal_data                       | personal data                   |
+-----------------------------------------------+---------------------------------+
```

Available templates: `articles`, `definitions`, `chapters`, `references`, `rights`

### Output Formats

```bash
# JSON output
./regula query --format json "SELECT ?term WHERE { ?term rdf:type reg:DefinedTerm }"

# CSV output
./regula query --format csv "SELECT ?article ?title WHERE { ... }"

# With timing
./regula query --timing "SELECT ?a WHERE { ?a rdf:type reg:Article }"
```

---

## Impact Analysis

Analyze how changes to one provision affect others:

```
$ ./regula impact --provision "Art17" --source testdata/gdpr.txt --depth 2

Impact Analysis for: Right to erasure ('right to be forgotten')
URI: https://regula.dev/regulations/GDPR:Art17
Analysis Depth: 2
===================================================

Summary:
  Total affected provisions: 42
  Direct incoming (references this): 4
  Direct outgoing (this references): 4
  Transitive: 34
  Max depth reached: 2

Direct Incoming (provisions referencing this):
  - Tasks of the Board (Article)
  - Processing which does not require identification (Article)
  - Transparent information, communication and modalities... (Article)
  - Notification obligation regarding rectification or erasure... (Article)

Direct Outgoing (provisions this references):
  - Processing of special categories of personal data (Article)
  - Right to object (Article)
  - Safeguards and derogations relating to processing... (Article)
  - Lawfulness of processing (Article)

Transitive Impact:
  Depth 2:
    - Independence (Article, incoming)
    - Reports (Article, incoming)
    - Certification (Article, outgoing)
    - Codes of conduct (Article, outgoing)
    ... and 29 more

Affected by Type:
  Article: 41
  Chapter: 1
```

### Impact Options

```bash
# Incoming references only
./regula impact --provision "Art6" --direction incoming --source testdata/gdpr.txt

# Deeper analysis
./regula impact --provision "Art6" --depth 3 --source testdata/gdpr.txt

# JSON output
./regula impact --provision "Art6" --format json --source testdata/gdpr.txt
```

---

## Compliance Scenarios

Match real-world scenarios to applicable provisions:

```
$ ./regula match --scenario consent_withdrawal --source testdata/gdpr.txt

Provision Matching Results for: Consent Withdrawal
===================================================

Summary:
  Total matches: 88
  Direct: 5
  Triggered: 10
  Related: 73

Direct Matches:
  Art 14: Information to be provided... (score: 1.00)
    - Grants RightToWithdrawConsent (action: withdraw_consent)
  Art 8: Conditions applicable to child's consent... (score: 1.00)
    - Imposes ConsentObligation (action: withdraw_consent)
  Art 7: Conditions for consent (score: 1.00)
    - Grants RightToWithdrawConsent (action: withdraw_consent)
    - Imposes ConsentObligation (action: withdraw_consent)

Triggered Matches:
  Art 60: Cooperation between the lead supervisory authority... (score: 0.80)
  Art 12: Transparent information, communication... (score: 0.80)
  Art 6: Lawfulness of processing (score: 0.75)
```

### Built-in Scenarios

```bash
./regula match --list-scenarios

# Available:
#   consent_withdrawal  - Data subject withdraws consent
#   access_request      - Data subject requests access to data
#   erasure_request     - Data subject requests erasure of data
#   data_breach         - Personal data breach handling
```

---

## Validation

Check extraction quality and graph consistency:

```
$ ./regula validate --source testdata/gdpr.txt

Validation Report
=================
Profile: GDPR

Reference Resolution:
  Total references: 258
  Resolved: 242 (98.8%)
  Unresolved: 3

Graph Connectivity:
  Total provisions: 99
  Connected: 78 (78.8%)
  Most referenced:
    - Article 6: 9 references
    - Article 9: 7 references

Definition Coverage:
  Defined terms: 26
  Terms with usage links: 26 (100.0%)
  Total term usages: 349

Semantic Extraction:
  Rights found: 60 (in 24 articles)
  Obligations found: 70 (in 41 articles)

Component Scores:
  References:    98.8% (weight: 25%)
  Connectivity:  78.8% (weight: 20%)
  Definitions:   100.0% (weight: 20%)
  Semantics:     100.0% (weight: 20%)
  Structure:     100.0% (weight: 15%)

Overall Score: 95.5%
Status: PASS
```

### Validation Options

```bash
# Specific profile
./regula validate --source testdata/ccpa.txt --profile CCPA

# Check only references
./regula validate --source testdata/gdpr.txt --check references

# Generate validation profile
./regula validate --source testdata/gdpr.txt --suggest-profile

# Export report
./regula validate --source testdata/gdpr.txt --report validation.html
```

---

## Export Formats

### RDF/Turtle

```
$ ./regula export --source testdata/gdpr.txt --format turtle | head -30

@prefix dc: <http://purl.org/dc/terms/> .
@prefix eli: <http://data.europa.eu/eli/ontology#> .
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix reg: <https://regula.dev/ontology#> .

<https://regula.dev/regulations/GDPR> a reg:Regulation ;
    rdfs:label "GDPR" ;
    reg:identifier "(EU) 2016/679" ;
    reg:title "REGULATION (EU) 2016/679..." .

<https://regula.dev/regulations/GDPR:Art1> a reg:Article ;
    reg:number "1" ;
    reg:partOf <https://regula.dev/regulations/GDPR:ChapterI> ;
    reg:title "Subject-matter and objectives" .
```

### All Export Formats

```bash
./regula export --source testdata/gdpr.txt --format json --output graph.json
./regula export --source testdata/gdpr.txt --format dot --output graph.dot
./regula export --source testdata/gdpr.txt --format jsonld --output graph.jsonld
./regula export --source testdata/gdpr.txt --format rdfxml --output graph.rdf
./regula export --source testdata/gdpr.txt --format turtle --eli --output graph-eli.ttl
```

---

## US Code Analysis

### Bulk Sources

```
$ ./regula bulk --help

Download and ingest legislation data in bulk from 5 official sources:

  uscode        US Code XML from uscode.house.gov (54 titles)
  cfr           Code of Federal Regulations from govinfo.gov (50 titles)
  california    California codes from leginfo.legislature.ca.gov (30 codes)
  archive       State code archives from Internet Archive govlaw collection
  parliamentary Congressional rules: House Rules, Senate Rules, Joint Rules
```

### Analysis Playground

```
$ ./regula playground list

Available playground analysis templates:

  chapter-structure            [structure      ] Hierarchical breakdown
  cross-ref-density            [cross-reference] Articles with most cross-references
  definition-coverage          [definitions    ] Definition count per title
  rights-enumeration           [semantics      ] All identified rights
  sections-with-obligations    [semantics      ] Sections with obligations
  temporal-analysis            [temporal       ] References with temporal qualifiers
  title-size-comparison        [structure      ] Rank titles by article count
  top-chapters-by-sections     [structure      ] Top chapters by section count
```

### Running Templates

```bash
./regula playground run top-chapters-by-sections --path .regula
./regula playground run cross-ref-density --title 42 --path .regula
./regula playground run definition-coverage --export json --path .regula
./regula playground query "SELECT ?s ?p ?o WHERE { ?s ?p ?o } LIMIT 10" --path .regula
```

---

## Parliamentary Rules

### Available Documents

```
$ ./regula bulk list parliamentary

IDENTIFIER                       JURISDICTION FORMAT   DISPLAY NAME
────────────────────────────────────────────────────────────────────
house-rules-119th                US-Federal   pdf      House Rules (119th Congress)
senate-rules                     US-Federal   htm      Senate Standing Rules
joint-rules                      US-Federal   pdf      Joint Rules of Congress
house-rules-committee-procedures US-Federal   htm      House Rules Committee Procedures
senate-precedents-riddick        US-Federal   pdf      Riddick's Senate Procedure

Total: 5 datasets
```

### Working with Rules

```bash
# Download and ingest
./regula bulk download parliamentary --dataset house-rules-119th
./regula bulk ingest --source parliamentary --path .regula

# Search rules
./regula search committee --query "agriculture" --source house-rules.txt
./regula search keywords --query "unanimous consent" --source house-rules.txt

# Compare versions
./regula compare rules --base house-rules-118th.txt --target house-rules-119th.txt
```

---

## Draft Legislation

### Analyzing Draft Bills

```
$ ./regula draft --help

Commands:
  ingest    Parse a draft bill and display its structure and amendments
  diff      Compute structural diff against the USC knowledge graph
  impact    Run impact analysis against the USC knowledge graph
  conflicts Run conflict and consistency analysis
  simulate  Run compliance scenario simulation
  report    Generate comprehensive legislative impact report
```

### Draft Analysis Pipeline

```bash
# Parse bill structure
./regula draft ingest --bill draft-hr-1234.txt

# Compute diff against existing law
./regula draft diff --bill draft-hr-1234.txt --path .regula

# Run impact analysis
./regula draft impact --bill draft-hr-1234.txt --depth 2 --path .regula

# Check for conflicts
./regula draft conflicts --bill draft-hr-1234.txt --path .regula

# Simulate scenarios
./regula draft simulate --bill draft-hr-1234.txt --scenario consent_withdrawal

# Generate report
./regula draft report --bill draft-hr-1234.txt --format html --output report.html
```

---

## Getting Help

```bash
./regula --help              # General help
./regula <command> --help    # Command-specific help
```

## Related Documentation

- [TUTORIAL.md](./TUTORIAL.md) - GDPR-focused tutorial
- [ARCHITECTURE.md](./ARCHITECTURE.md) - System design
- [TESTING.md](./TESTING.md) - Testing strategies
- [PATTERN_SCHEMA.md](./PATTERN_SCHEMA.md) - Pattern file format
