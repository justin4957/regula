# Regula Tutorial

A hands-on guide to transforming legal regulations into queryable, analyzable knowledge graphs.

## Table of Contents

1. [Introduction](#introduction)
2. [Installation](#installation)
3. [Quick Start](#quick-start)
4. [Core Concepts](#core-concepts)
5. [Working with Regulations](#working-with-regulations)
6. [Querying the Knowledge Graph](#querying-the-knowledge-graph)
7. [Impact Analysis](#impact-analysis)
8. [Compliance Scenario Matching](#compliance-scenario-matching)
9. [Validation and Quality Assurance](#validation-and-quality-assurance)
10. [Exporting and Visualization](#exporting-and-visualization)
11. [Advanced Usage](#advanced-usage)

---

## Introduction

Regula transforms dense legal regulations into auditable, queryable, and simulatable programs. It ingests regulatory documents and produces:

- **Queryable knowledge graphs** via SPARQL-like queries
- **Type-safe domain models** with compile-time verification
- **Impact analysis** for regulatory changes
- **Simulation engine** for compliance scenarios
- **Audit trails** with provenance tracking

### What Problems Does Regula Solve?

1. **Regulatory Complexity**: Regulations like GDPR contain hundreds of articles with complex cross-references. Regula maps these relationships automatically.

2. **Impact Assessment**: When a regulation changes, Regula identifies all affected provisions through transitive dependency analysis.

3. **Compliance Verification**: Match real-world scenarios (data breach, consent withdrawal) to applicable legal provisions.

4. **Knowledge Extraction**: Extract definitions, rights, obligations, and cross-references from legal text.

---

## Installation

### Prerequisites

- Go 1.21 or later
- Git

### Building from Source

```bash
git clone https://github.com/justin4957/regula.git
cd regula
go build -o regula ./cmd/regula
```

### Verify Installation

```bash
./regula --version
# regula version 0.1.0

./regula --help
```

---

## Quick Start

Let's analyze the GDPR (General Data Protection Regulation) in under 5 minutes.

### Step 1: Initialize a Project

```bash
./regula init gdpr-analysis
```

Output:
```
Initialized regulation project: gdpr-analysis
Created directories:
  - gdpr-analysis/regulations/
  - gdpr-analysis/graphs/
  - gdpr-analysis/scenarios/
  - gdpr-analysis/reports/

Next steps:
  1. Add regulation documents to gdpr-analysis/regulations/
  2. Run: regula ingest --source gdpr-analysis/regulations/your-doc.txt
  3. Run: regula query "SELECT ?article WHERE { ?article rdf:type reg:Article }"
```

### Step 2: Ingest a Regulation

```bash
./regula ingest --source testdata/gdpr.txt --stats
```

Output:
```
Ingesting regulation from: testdata/gdpr.txt
  1. Parsing document structure... done (11 chapters, 99 articles)
  2. Extracting defined terms... done (26 definitions)
  3. Identifying cross-references... done (255 references)
  4. Extracting rights/obligations... done (60 rights, 69 obligations)
  5. Resolving cross-references... done (100% resolved)
  6. Building knowledge graph... done (8457 triples)

Ingestion complete in 284.008875ms

Graph Statistics:
  Total triples:    8457
  Articles:         99
  Chapters:         11
  Sections:         15
  Recitals:         173
  Definitions:      26
  References:       255
  Rights:           60
  Obligations:      70
  Term usages:      349
```

### Step 3: Query the Graph

```bash
./regula query --source testdata/gdpr.txt --template articles
```

Output (truncated):
```
+-------------------------------------------+-----------------------------------------------------------------------+
| article                                   | title                                                                 |
+-------------------------------------------+-----------------------------------------------------------------------+
| https://regula.dev/regulations/GDPR:Art1  | Subject-matter and objectives                                         |
| https://regula.dev/regulations/GDPR:Art15 | Right of access by the data subject                                   |
| https://regula.dev/regulations/GDPR:Art17 | Right to erasure ('right to be forgotten')                            |
| https://regula.dev/regulations/GDPR:Art33 | Notification of a personal data breach to the supervisory authority   |
...
```

---

## Core Concepts

### Knowledge Graph Structure

Regula builds an RDF triple store with the following node types:

| Node Type | Description | Color (DOT) |
|-----------|-------------|-------------|
| Article | Legal articles | Light blue |
| Chapter | Document chapters | Light green |
| Section | Chapter sections | Light yellow |
| Recital | Preamble recitals | Lavender |
| Definition | Defined terms | Light pink |
| Right | Granted rights | Light coral |
| Obligation | Imposed obligations | Light gray |
| Reference | Cross-references | White |

### Relationship Types

The graph captures these relationship types:

| Predicate | Description | Example |
|-----------|-------------|---------|
| `reg:references` | Article references another | Art 17 → Art 6 |
| `reg:referencedBy` | Inverse of references | Art 6 ← Art 17 |
| `reg:defines` | Term defined in article | Art 4 defines "consent" |
| `reg:usesTerm` | Article uses defined term | Art 7 uses "consent" |
| `reg:grantsRight` | Article grants a right | Art 15 grants RightOfAccess |
| `reg:imposesObligation` | Article imposes obligation | Art 33 imposes BreachNotificationObligation |
| `reg:partOf` | Hierarchical containment | Art 15 partOf ChapterIII |

---

## Working with Regulations

### Supported Document Formats

Regula supports plain text (`.txt`) and Markdown (`.md`) formatted regulations with structure markers:

```
CHAPTER I
General provisions

Article 1
Subject-matter and objectives

1. This Regulation lays down rules relating to the protection...

Article 2
Material scope

1. This Regulation applies to the processing of personal data...
```

### Ingestion Process

The ingestion pipeline performs six steps:

1. **Document Parsing**: Identifies chapters, sections, articles, recitals
2. **Definition Extraction**: Finds formally defined terms (e.g., "'personal data' means...")
3. **Reference Identification**: Detects cross-references (e.g., "pursuant to Article 6")
4. **Semantic Extraction**: Identifies rights and obligations from text patterns
5. **Reference Resolution**: Resolves references to specific article URIs
6. **Graph Building**: Constructs the RDF triple store

### Viewing Statistics

Use `--stats` to see extraction results:

```bash
./regula ingest --source testdata/gdpr.txt --stats
```

---

## Querying the Knowledge Graph

### Query Templates

Regula provides built-in query templates for common operations:

```bash
./regula query --list-templates
```

| Template | Description |
|----------|-------------|
| `articles` | List all articles with titles |
| `definitions` | List all defined terms with definitions |
| `chapters` | List all chapters |
| `references` | List cross-references between articles |
| `rights` | Find provisions granting rights |

### Using Templates

**List all articles:**
```bash
./regula query --source testdata/gdpr.txt --template articles
```

**List all definitions:**
```bash
./regula query --source testdata/gdpr.txt --template definitions
```

Output (truncated):
```
+--------------------------------------+---------------------------------+----------------------------------------+
| term                                 | termText                        | definition                             |
+--------------------------------------+---------------------------------+----------------------------------------+
| GDPR:Term:personal_data              | personal data                   | any information relating to an         |
|                                      |                                 | identified or identifiable natural     |
|                                      |                                 | person ('data subject')...             |
| GDPR:Term:consent                    | consent                         | any freely given, specific, informed   |
|                                      |                                 | and unambiguous indication of the      |
|                                      |                                 | data subject's wishes...               |
| GDPR:Term:controller                 | controller                      | the natural or legal person, public    |
|                                      |                                 | authority, agency or other body which  |
|                                      |                                 | determines the purposes and means...   |
+--------------------------------------+---------------------------------+----------------------------------------+
26 rows
```

**Find provisions granting rights:**
```bash
./regula query --source testdata/gdpr.txt --template rights
```

Output (truncated):
```
+------------+------------------------------------------+-----------------------------------------+-------------------------------+
| article    | title                                    | right                                   | rightType                     |
+------------+------------------------------------------+-----------------------------------------+-------------------------------+
| GDPR:Art15 | Right of access by the data subject      | GDPR:Right:15:RightOfAccess             | RightOfAccess                 |
| GDPR:Art16 | Right to rectification                   | GDPR:Right:16:RightToRectification      | RightToRectification          |
| GDPR:Art17 | Right to erasure ('right to be forgotten')| GDPR:Right:17:RightToErasure            | RightToErasure                |
| GDPR:Art18 | Right to restriction of processing       | GDPR:Right:18:RightToRestriction        | RightToRestriction            |
| GDPR:Art20 | Right to data portability                | GDPR:Right:20:RightToDataPortability    | RightToDataPortability        |
| GDPR:Art21 | Right to object                          | GDPR:Right:21:RightToObject             | RightToObject                 |
+------------+------------------------------------------+-----------------------------------------+-------------------------------+
```

### Output Formats

Query results can be formatted as tables, JSON, or CSV:

```bash
# Table format (default)
./regula query --source testdata/gdpr.txt --template articles --format table

# JSON format
./regula query --source testdata/gdpr.txt --template articles --format json

# CSV format
./regula query --source testdata/gdpr.txt --template articles --format csv
```

### Query Timing

Use `--timing` to measure query performance:

```bash
./regula query --source testdata/gdpr.txt --template articles --timing
```

---

## Impact Analysis

Impact analysis identifies provisions affected by changes to a specific article.

### Basic Impact Analysis

```bash
./regula impact --source testdata/gdpr.txt --provision "GDPR:Art17" --depth 2
```

Output:
```
Analyzing impact of amend to GDPR:Art17

Articles referencing GDPR:Art17: 4
+-------------------------------------------+----------------------------------------------------------------------------------------------------------+
| article                                   | title                                                                                                    |
+-------------------------------------------+----------------------------------------------------------------------------------------------------------+
| https://regula.dev/regulations/GDPR:Art19 | Notification obligation regarding rectification or erasure of personal data or restriction of processing |
| https://regula.dev/regulations/GDPR:Art70 | Tasks of the Board                                                                                       |
| https://regula.dev/regulations/GDPR:Art11 | Processing which does not require identification                                                         |
| https://regula.dev/regulations/GDPR:Art12 | Transparent information, communication and modalities for the exercise of the rights of the data subject |
+-------------------------------------------+----------------------------------------------------------------------------------------------------------+
```

### Impact Analysis Options

| Flag | Description | Default |
|------|-------------|---------|
| `--provision` | Provision ID to analyze | Required |
| `--change` | Type of change (amend, repeal, add) | amend |
| `--depth` | Transitive dependency depth | 3 |
| `--source` | Source document | Required |

### Understanding Impact Results

- **Direct Impact**: Articles that directly reference the changed provision
- **Transitive Impact**: Articles that reference those direct references (with depth > 1)

---

## Compliance Scenario Matching

Match real-world compliance scenarios to applicable legal provisions.

### Available Scenarios

```bash
./regula match --list-scenarios
```

Output:
```
Available scenarios:
  consent_withdrawal   Data subject withdraws previously given consent for data processing
  access_request       Data subject requests access to their personal data
  erasure_request      Data subject requests erasure of their personal data
  data_breach          Personal data breach occurs and must be handled
```

### Running a Scenario Match

**Consent Withdrawal:**
```bash
./regula match --source testdata/gdpr.txt --scenario consent_withdrawal
```

Output:
```
Provision Matching Results for: Consent Withdrawal
===================================================

Summary:
  Total matches: 88
  Direct: 5
  Triggered: 10
  Related: 73

Direct Matches:
  Art 7: Conditions for consent (score: 1.00)
    - Grants RightToWithdrawConsent (action: withdraw_consent)
    - Imposes ConsentObligation (action: withdraw_consent)
  Art 8: Conditions applicable to child's consent in relation to information society services (score: 1.00)
    - Imposes ConsentObligation (action: withdraw_consent)
  Art 13: Information to be provided where personal data are collected from the data subject (score: 1.00)
    - Grants RightToWithdrawConsent (action: withdraw_consent)
  Art 14: Information to be provided where personal data have not been obtained from the data subject (score: 1.00)
    - Grants RightToWithdrawConsent (action: withdraw_consent)

Triggered Matches:
  Art 12: Transparent information, communication and modalities... (score: 0.80)
    - References Article 14
  Art 9: Processing of special categories of personal data (score: 0.75)
    - Referenced by Article 14
  ...

Related Matches: (73 articles)
  Art 43: Certification bodies (score: 0.45)
  Art 17: Right to erasure ('right to be forgotten') (score: 0.45)
  ...
```

**Data Breach:**
```bash
./regula match --source testdata/gdpr.txt --scenario data_breach
```

Output:
```
Provision Matching Results for: Data Breach
===================================================

Summary:
  Total matches: 80
  Direct: 6
  Triggered: 10
  Related: 64

Direct Matches:
  Art 33: Notification of a personal data breach to the supervisory authority (score: 1.00)
    - Imposes BreachNotificationObligation (action: data_breach)
  Art 32: Security of processing (score: 1.00)
    - Imposes SecurityObligation (action: data_breach)
  Art 34: Communication of a personal data breach to the data subject (score: 1.00)
    - Imposes SubjectNotificationObligation (action: data_breach)
  Art 24: Responsibility of the controller (score: 0.95)
    - Imposes SecurityObligation (action: data_breach)
  ...
```

### Match Relevance Categories

| Category | Description | Score Range |
|----------|-------------|-------------|
| **DIRECT** | Directly grants rights or imposes obligations matching the scenario action | 0.80 - 1.00 |
| **TRIGGERED** | Referenced by or references direct matches | 0.60 - 0.90 |
| **RELATED** | Contains keywords related to the scenario | 0.30 - 0.50 |

### JSON Output

```bash
./regula match --source testdata/gdpr.txt --scenario access_request --format json
```

---

## Validation and Quality Assurance

Validate extraction quality and graph consistency.

### Running Validation

```bash
./regula validate --source testdata/gdpr.txt
```

Output:
```
Validation Report
=================

Reference Resolution:
  Total references: 255
  Resolved: 242 (100.0%)
  Unresolved: 0
    - External: 13
    - Ambiguous: 0
    - Not found: 0

Graph Connectivity:
  Total provisions: 99
  Connected: 78 (78.8%)
  Orphans: 21
  Most referenced:
    - Article 6: 9 references
    - Article 9: 7 references
    - Article 65: 6 references
    - Article 43: 6 references
    - Article 40: 6 references

Definition Coverage:
  Defined terms: 26
  Terms with usage links: 26 (100.0%)
  Total term usages: 349
  Articles using terms: 88

Semantic Extraction:
  Rights found: 60 (in 24 articles)
  Obligations found: 70 (in 41 articles)
  Known GDPR rights: 6/6

Warnings:
  [connectivity] 21 provisions have no cross-references

Overall Score: 94.7%
Threshold: 80.0%
Status: PASS
```

### Validation Options

| Flag | Description | Default |
|------|-------------|---------|
| `--threshold` | Pass/fail threshold (0.0-1.0) | 0.8 |
| `--check` | What to check (all, references) | all |
| `--format` | Output format (text, json) | text |

### Validation Metrics

| Metric | Description | Good Threshold |
|--------|-------------|----------------|
| Reference Resolution | % of references resolved to URIs | ≥ 80% |
| Graph Connectivity | % of provisions with cross-references | ≥ 70% |
| Definition Coverage | % of defined terms with usage links | ≥ 90% |
| Rights Extraction | Number of rights identified | ≥ 50 |
| Obligations Extraction | Number of obligations identified | ≥ 50 |

---

## Exporting and Visualization

Export the knowledge graph for visualization or further processing.

### Export Formats

```bash
# Summary statistics
./regula export --source testdata/gdpr.txt --format summary

# DOT format for Graphviz
./regula export --source testdata/gdpr.txt --format dot --output graph.dot

# JSON format
./regula export --source testdata/gdpr.txt --format json --output graph.json
```

### Summary Export

```bash
./regula export --source testdata/gdpr.txt --format summary
```

Output:
```
Relationship Graph Summary
==========================

Total relationships: 3134

Relationship Types:
  reg:contains              133
  reg:definedIn             26
  reg:grantsRight           44
  reg:externalRef           13
  reg:partOf                1016
  reg:resolvedTarget        297
  reg:usesTerm              349
  reg:hasRecital            173
  reg:defines               26
  reg:imposesObligation     62
  reg:referencedBy          179
  reg:references            179
  reg:hasSection            15
  reg:belongsTo             512
  reg:hasArticle            99
  reg:hasChapter            11

Most Referenced Articles:
  Article 6: 9 incoming references
  Article 9: 7 incoming references
  Article 40: 6 incoming references
  Article 43: 6 incoming references
  Article 65: 6 incoming references

Articles With Most Outgoing References:
  Article 70: 18 outgoing references
  Article 12: 11 outgoing references
  Article 58: 9 outgoing references
  Article 40: 8 outgoing references
```

### Graphviz Visualization

Export to DOT format and render with Graphviz:

```bash
./regula export --source testdata/gdpr.txt --format dot --output gdpr.dot
dot -Tpng gdpr.dot -o gdpr.png
```

The DOT output uses color coding:
- Light blue: Articles
- Light green: Chapters
- Light yellow: Sections
- Light pink: Definitions
- Light coral: Rights

---

## Advanced Usage

### Custom Base URI

Specify a custom base URI for the knowledge graph:

```bash
./regula ingest --source regulation.txt --base-uri "https://mycompany.com/legal/"
```

### Batch Processing

Process multiple regulations:

```bash
for file in regulations/*.txt; do
    ./regula ingest --source "$file" --output "graphs/$(basename "$file" .txt).json"
done
```

### Integration with CI/CD

Use the E2E test script to validate extraction quality in CI:

```bash
# In GitHub Actions or similar
./scripts/e2e-test.sh
```

The script validates:
- Article count (≥ 50)
- Definition count (≥ 20)
- Reference count (≥ 100)
- Reference resolution rate (≥ 80%)
- Graph triple count (≥ 500)
- Impact analysis coverage
- Scenario matching coverage

### Programmatic Access

Import regula packages directly in Go:

```go
package main

import (
    "github.com/coolbeans/regula/pkg/extract"
    "github.com/coolbeans/regula/pkg/store"
    "github.com/coolbeans/regula/pkg/simulate"
)

func main() {
    // Parse document
    parser := extract.NewParser()
    doc, _ := parser.Parse(file)

    // Build graph
    ts := store.NewTripleStore()
    builder := store.NewGraphBuilder(ts, "https://example.com/")
    builder.BuildComplete(doc, defExtractor, refExtractor, resolver, semExtractor)

    // Match scenario
    matcher := simulate.NewProvisionMatcher(ts, baseURI, annotations, doc)
    result := matcher.Match(simulate.ConsentWithdrawalScenario())
}
```

---

## Troubleshooting

### Common Issues

**"No graph loaded" error:**
```bash
# Use --source flag with query command
./regula query --source testdata/gdpr.txt --template articles
```

**Low reference resolution rate:**
- Check for non-standard reference formats in the source document
- External references (Directive 95/46/EC) are counted separately

**Missing definitions:**
- Ensure definitions follow the pattern: `'term' means ...` or `(1) 'term' ...`

### Getting Help

```bash
./regula --help
./regula [command] --help
```

---

## Next Steps

1. **Explore the GDPR**: Use the included `testdata/gdpr.txt` to explore all features
2. **Try Different Scenarios**: Match your compliance scenarios to applicable provisions
3. **Visualize the Graph**: Export to DOT format and render with Graphviz
4. **Integrate with Workflows**: Use validation in CI/CD pipelines

For more information:
- Architecture: `docs/ARCHITECTURE.md`
- Roadmap: `docs/ROADMAP.md`
- Ontology: `docs/ONTOLOGY.md`
