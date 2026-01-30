# Regula Tutorial: Automated Regulation Mapping

This tutorial walks through Regula's core capabilities using the **EU General Data Protection Regulation (GDPR)** as a real-world example. Each section includes commands you can run and real output from the tool.

## Prerequisites

Build Regula from source:

```bash
go build -o regula cmd/regula/main.go
```

Or run directly with `go run`:

```bash
go run cmd/regula/main.go --help
```

The GDPR test document is included at `testdata/gdpr.txt`.

---

## 1. Ingesting a Regulation

The `ingest` command parses a regulation document, extracts its structure, and builds an RDF knowledge graph.

```bash
regula ingest --source testdata/gdpr.txt
```

**Output:**

```
Ingesting regulation from: testdata/gdpr.txt
  1. Parsing document structure... done (11 chapters, 99 articles)
  2. Extracting defined terms... done (26 definitions)
  3. Identifying cross-references... done (258 references)
  4. Extracting rights/obligations... done (60 rights, 69 obligations)
  5. Resolving cross-references... done (99% resolved)
  6. Building knowledge graph... done (8495 triples)

Ingestion complete in 1.22s
```

From a single text file, Regula automatically extracts:

| Extracted Element       | Count |
|------------------------|-------|
| Chapters               | 11    |
| Articles               | 99    |
| Defined terms          | 26    |
| Cross-references       | 258   |
| Rights granted         | 60    |
| Obligations imposed    | 69    |
| RDF triples produced   | 8,495 |

The entire GDPR is parsed, structured, and converted into a queryable knowledge graph in ~1.2 seconds.

### Automatic Mapping Statistics

The ingestion pipeline performs six stages of extraction. Each stage operates on the raw text and produces structured RDF triples:

1. **Document Parsing**: Identifies 11 chapters, 15 sections, 99 articles, and 173 recitals from structural markers
2. **Definition Extraction**: Finds 26 formally defined terms using pattern matching (`'term' means...`)
3. **Reference Identification**: Detects 258 internal cross-references using citation patterns (`Article 6`, `pursuant to Article 17(1)`)
4. **Semantic Extraction**: Identifies 60 rights and 69 obligations from regulatory language patterns
5. **Reference Resolution**: Resolves 99% of cross-references to specific article URIs (3 unresolved are ambiguous external references)
6. **Graph Building**: Constructs an RDF triple store with 8,495 triples across 17 relationship types

The 3,141 relationships break down as:

| Relationship Type         | Count | Description                          |
|--------------------------|------:|--------------------------------------|
| `reg:partOf`             | 1,019 | Hierarchical containment             |
| `reg:belongsTo`          |   515 | Reverse containment                  |
| `reg:usesTerm`           |   349 | Article uses a defined term          |
| `reg:resolvedTarget`     |   297 | Resolved reference targets           |
| `reg:referencedBy`       |   179 | Incoming cross-references            |
| `reg:references`         |   179 | Outgoing cross-references            |
| `reg:hasRecital`         |   173 | Preamble recitals                    |
| `reg:contains`           |   133 | Chapter contains articles            |
| `reg:hasArticle`         |    99 | Document has articles                |
| `reg:imposesObligation`  |    62 | Obligation imposed                   |
| `reg:grantsRight`        |    44 | Right granted                        |
| `reg:defines`            |    26 | Term defined in article              |
| `reg:definedIn`          |    26 | Reverse of defines                   |
| `reg:hasSection`         |    15 | Chapter has sections                 |
| `reg:externalRef`        |    13 | External legislation references      |
| `reg:hasChapter`         |    11 | Document has chapters                |
| `reg:repealedBy`         |     1 | Repealed provision                   |

---

## 2. Querying the Knowledge Graph

Regula supports SPARQL-style queries against the knowledge graph. Built-in query templates cover common regulatory analysis patterns.

### List Available Templates

```bash
regula query --source testdata/gdpr.txt --list-templates
```

**19 available templates:**

| Template             | Description                                        |
|----------------------|----------------------------------------------------|
| `articles`           | List all articles with titles                      |
| `chapters`           | List all chapters with titles                      |
| `definitions`        | List all defined terms with definitions            |
| `rights`             | Find articles that grant rights                    |
| `obligations`        | Find articles that impose obligations              |
| `references`         | List all cross-references between articles         |
| `most-referenced`    | Find the most referenced articles                  |
| `article-refs`       | Find what articles reference a specific article    |
| `article-terms`      | Find all terms used in a specific article          |
| `term-usage`         | Find which articles use defined terms              |
| `term-articles`      | Find articles using a specific term                |
| `hierarchy`          | Show document hierarchy                            |
| `bidirectional`      | Show bidirectional reference relationships          |
| `right-types`        | List distinct right types found                    |
| `obligation-types`   | List distinct obligation types found               |
| `describe-article`   | Describe all triples for a specific article        |
| `definition-links`   | Show terms and their defining articles             |
| `recitals`           | List all recitals                                  |
| `search`             | Search for articles containing a keyword           |

### Example: Find All Defined Terms

```bash
regula query --source testdata/gdpr.txt --template definitions --timing
```

**Output (abbreviated):**

```
+------------------------------------+---------------------------------+---------------------------+
| term                               | termText                        | definition                |
+------------------------------------+---------------------------------+---------------------------+
| GDPR:Term:personal_data            | personal data                   | any information relating  |
|                                    |                                 | to an identified or       |
|                                    |                                 | identifiable natural      |
|                                    |                                 | person...                 |
| GDPR:Term:processing               | processing                      | any operation or set of   |
|                                    |                                 | operations which is       |
|                                    |                                 | performed on personal     |
|                                    |                                 | data...                   |
| GDPR:Term:consent                  | consent                         | any freely given, specific|
|                                    |                                 | informed and unambiguous  |
|                                    |                                 | indication...             |
| GDPR:Term:controller               | controller                      | the natural or legal      |
|                                    |                                 | person, public authority  |
|                                    |                                 | which determines the      |
|                                    |                                 | purposes and means...     |
| ...                                | ...                             | ...                       |
+------------------------------------+---------------------------------+---------------------------+
26 rows

Query executed in 85.79µs
  Parse:   0s
  Plan:    5.33µs
  Execute: 65.04µs
```

All 26 GDPR-defined terms are extracted and queryable in **86 microseconds**.

### Example: Find Rights-Granting Articles

```bash
regula query --source testdata/gdpr.txt --template rights
```

Returns 60 rights across 24 articles, with typed classifications:

| Right Type                      | Example Article | Description                        |
|---------------------------------|----------------|------------------------------------|
| `RightToErasure`               | Article 17     | Right to be forgotten              |
| `RightOfAccess`                | Article 15     | Access to personal data            |
| `RightToRectification`         | Article 16     | Correct inaccurate data            |
| `RightToDataPortability`       | Article 20     | Receive data in portable format    |
| `RightToObject`                | Article 21     | Object to processing               |
| `RightAgainstAutomatedDecision`| Article 22     | Not be subject to automated decisions |
| `RightToWithdrawConsent`       | Articles 13, 14| Withdraw previously given consent  |
| `RightToRestriction`           | Article 18     | Restrict processing                |
| `RightToLodgeComplaint`        | Articles 13-15, 47 | Lodge complaint with authority |
| `RightToNotification`          | Article 15     | Be notified about data processing  |

All 6 core GDPR rights are detected: 6/6 known rights found.

### Example: Find Obligation-Imposing Articles

```bash
regula query --source testdata/gdpr.txt --template obligations
```

Returns 69 obligations across 41 articles:

| Obligation Type                    | Example Article | Description                    |
|------------------------------------|----------------|--------------------------------|
| `BreachNotificationObligation`    | Articles 19, 33| Notify authority of breaches   |
| `SecurityObligation`             | Articles 24, 25, 32| Implement security measures |
| `RecordKeepingObligation`        | Article 30     | Maintain processing records    |
| `ConsentObligation`              | Articles 7, 8  | Obtain valid consent           |
| `InformationProvisionObligation` | Articles 12, 14| Provide information to subjects|
| `ImplementationObligation`       | Articles 12, 32| Implement required measures    |
| `SubjectNotificationObligation`  | Article 34     | Notify subjects of breaches    |
| `EnsureObligation`               | Article 25     | Ensure data protection by design|
| `ResponseObligation`             | Article 62     | Respond to requests            |

### Example: Find the Most Referenced Articles

```bash
regula query --source testdata/gdpr.txt --template most-referenced
```

**Output:**

```
Article 6:  9 incoming references   (Lawfulness of processing)
Article 9:  7 incoming references   (Special categories of personal data)
Article 43: 6 incoming references   (Certification)
Article 65: 6 incoming references   (Dispute resolution by the Board)
Article 40: 6 incoming references   (Codes of conduct)
```

These are the load-bearing provisions of the GDPR. When drafting amendments, these articles have the widest downstream impact.

Articles with the most outgoing references (regulatory "hubs"):

| Article | Outgoing Refs | Title                    |
|---------|:-------------:|--------------------------|
| Art 70  | 18            | Tasks of the Board       |
| Art 12  | 11            | Transparent information  |
| Art 58  | 9             | Powers                   |
| Art 40  | 8             | Codes of conduct         |
| Art 11  | 7             | Processing without ID    |
| Art 83  | 6             | Administrative fines     |
| Art 28  | 6             | Processor                |

### Example: Term Usage Across Articles

```bash
regula query --source testdata/gdpr.txt --template term-usage
```

Tracks where each defined term appears across the regulation. Selected examples:

| Term                    | Articles Using It | Count |
|-------------------------|:-----------------:|:-----:|
| consent                 | 13                | 13    |
| controller              | 16+               | 16+   |
| personal data           | widespread        | many  |
| binding corporate rules | 7                 | 7     |
| biometric data          | 1                 | 1     |

The term "consent" appears in Articles 6, 7, 8, 9, 13, 14, 17, 18, 20, 22, 40, 49, and 83. Changing the definition of "consent" in Article 4 would affect the interpretation of all 13 articles.

### Custom SPARQL Queries

Beyond templates, you can write custom SPARQL:

```bash
regula query --source testdata/gdpr.txt \
  "SELECT ?article ?title WHERE {
     ?article rdf:type reg:Article .
     ?article reg:title ?title .
     ?article reg:grantsRight ?right .
     ?right rdf:type reg:RightToErasure
   }"
```

### Output Formats

Query results can be formatted as tables, JSON, or CSV:

```bash
# Table format (default)
regula query --source testdata/gdpr.txt --template articles --format table

# JSON format for programmatic consumption
regula query --source testdata/gdpr.txt --template articles --format json

# CSV format for spreadsheets
regula query --source testdata/gdpr.txt --template articles --format csv
```

For CONSTRUCT and DESCRIBE queries:

```bash
# Turtle output
regula query --source testdata/gdpr.txt --template describe-article --format turtle

# N-Triples output
regula query --source testdata/gdpr.txt --template describe-article --format ntriples

# JSON output
regula query --source testdata/gdpr.txt --template describe-article --format json
```

### Query Performance

All queries execute in microseconds against the in-memory triple store:

| Query                | Execution Time | Results |
|----------------------|---------------|---------|
| 26 definitions       | 86 µs         | 26 rows |
| Bidirectional refs   | 224 µs        | 20 rows |
| DESCRIBE article     | 22 µs         | 59 triples |

Total wall time including document ingestion is under 1 second per query command.

---

## 3. Impact Analysis

The `impact` command traces how a change to one provision ripples through the regulation. This is the core of inter-legislation modelling: understanding which articles depend on, reference, or are triggered by a given provision.

### Example: Impact of Changing Article 17 (Right to Erasure)

```bash
regula impact --source testdata/gdpr.txt --provision Art17
```

**Output:**

```
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
  - Transparent information, communication and modalities (Article 12)
  - Notification obligation regarding erasure (Article 19)
  - Tasks of the Board (Article 70)
  - Processing which does not require identification (Article 11)

Direct Outgoing (provisions this references):
  - Lawfulness of processing (Article 6)
  - Processing of special categories of personal data (Article 9)
  - Right to object (Article 21)
  - Safeguards for archiving/research purposes (Article 89)

Transitive Impact (depth 2): 34 additional provisions
  Including: Right of access (Art 15), Right to restriction (Art 18),
  Binding corporate rules (Art 47), Certification (Art 42),
  Administrative fines (Art 83), Urgency procedure (Art 66),
  Data protection by design (Art 25), and 27 more...

Affected by Type:
  Article: 41
  Chapter: 1
```

From a single provision, Regula identifies **42 affected provisions** across two levels of transitive dependency in **0.84 seconds**.

### Configuring Analysis Depth

```bash
# Direct dependencies only
regula impact --source testdata/gdpr.txt --provision Art17 --depth 1

# Three levels of transitive impact
regula impact --source testdata/gdpr.txt --provision Art17 --depth 3

# Only incoming references (what depends on this article)
regula impact --source testdata/gdpr.txt --provision Art17 --direction incoming

# Only outgoing references (what this article depends on)
regula impact --source testdata/gdpr.txt --provision Art17 --direction outgoing

# JSON output for programmatic use
regula impact --source testdata/gdpr.txt --provision Art17 --format json

# Table format
regula impact --source testdata/gdpr.txt --provision Art17 --format table
```

### Impact Analysis as a Regulatory Tool

Without Regula, a legal team assessing the impact of amending Article 17 would need to:
1. Manually search the entire 88-page GDPR for references to Article 17
2. For each reference found, search again for references to *those* articles
3. Repeat for each level of transitive dependency
4. Classify each affected provision by type and direction

This typically requires hours of work by experienced regulatory analysts. With Regula, the complete transitive impact analysis identifies all 42 affected provisions in under 1 second.

---

## 4. Scenario Matching

The `match` command evaluates a compliance scenario against the regulation, finding directly applicable provisions, triggered provisions, and related articles.

### Available Scenarios

```bash
regula match --list-scenarios
```

| Scenario               | Description                                            |
|-----------------------|--------------------------------------------------------|
| `consent_withdrawal`  | Data subject withdraws previously given consent        |
| `access_request`      | Data subject requests access to their personal data    |
| `erasure_request`     | Data subject requests erasure of their personal data   |
| `data_breach`         | Personal data breach occurs and must be handled        |

### Example: Data Erasure Request

```bash
regula match --source testdata/gdpr.txt --scenario erasure_request
```

**Output:**

```
Provision Matching Results for: Data Erasure Request
===================================================

Summary:
  Total matches: 88
  Direct: 3
  Triggered: 11
  Related: 74

Direct Matches:
  Art 34: Communication of a personal data breach to the data subject (score: 1.00)
    - Imposes SubjectNotificationObligation (action: request_erasure)
  Art 17: Right to erasure ('right to be forgotten') (score: 1.00)
    - Grants RightToErasure (action: request_erasure)
  Art 62: Joint operations of supervisory authorities (score: 0.82)
    - Imposes ResponseObligation (action: request_erasure)

Triggered Matches:
  Art 19: Notification obligation (score: 0.95) - References Article 17
  Art 12: Transparent information (score: 0.85) - References Article 17
  Art 11: Processing without identification (score: 0.85) - References Article 17
  Art 70: Tasks of the Board (score: 0.85) - References Article 17
  Art 60: Cooperation between supervisory authorities (score: 0.80)
  Art 6:  Lawfulness of processing (score: 0.75) - Referenced by Article 17
  Art 9:  Special categories of personal data (score: 0.75)
  Art 89: Safeguards for archiving/research (score: 0.75)
  Art 21: Right to object (score: 0.75) - Referenced by Article 17
  Art 66: Urgency procedure (score: 0.65)
  Art 55: Competence (score: 0.60)

Related Matches: (74 articles)
  Art 18: Right to restriction of processing (score: 0.50)
  Art 58: Powers (score: 0.50)
  ... and 72 more
```

### Example: Data Breach Response

```bash
regula match --source testdata/gdpr.txt --scenario data_breach
```

Identifies 6 directly applicable provisions:

| Article | Provision                              | Obligation Type               | Score |
|---------|----------------------------------------|-------------------------------|-------|
| Art 33  | Notification to supervisory authority  | BreachNotificationObligation  | 1.00  |
| Art 32  | Security of processing                 | SecurityObligation            | 1.00  |
| Art 34  | Communication to data subject          | SubjectNotificationObligation | 1.00  |
| Art 25  | Data protection by design              | SecurityObligation            | 1.00  |
| Art 19  | Notification re: erasure/rectification | BreachNotificationObligation  | 1.00  |
| Art 24  | Responsibility of the controller       | SecurityObligation            | 0.95  |

Plus 10 triggered provisions and 64 related provisions — a comprehensive compliance checklist generated in **1.6 seconds**.

### Example: Consent Withdrawal

```bash
regula match --source testdata/gdpr.txt --scenario consent_withdrawal
```

Returns 5 direct matches covering the consent lifecycle:

| Article | Provision                                         | Score |
|---------|---------------------------------------------------|-------|
| Art 7   | Conditions for consent                            | 1.00  |
| Art 8   | Child's consent for information society services  | 1.00  |
| Art 13  | Information provided at collection                | 1.00  |
| Art 14  | Information provided without direct collection    | 1.00  |
| Art 62  | Joint operations of supervisory authorities       | 0.82  |

Plus 10 triggered and 73 related provisions.

### Match Relevance Categories

| Category      | Score Range | Description                                          |
|---------------|-------------|------------------------------------------------------|
| **Direct**    | 0.80 - 1.00 | Directly grants rights or imposes obligations matching the scenario |
| **Triggered** | 0.60 - 0.95 | Referenced by or references direct matches           |
| **Related**   | < 0.60       | Connected by keyword similarity                      |

---

## 5. Validation and Quality Assessment

Regula validates the extracted knowledge graph for completeness and correctness using weighted component scoring and a multi-stage gate pipeline.

### Full Validation

```bash
regula validate --source testdata/gdpr.txt
```

**Output (abbreviated):**

```
Definition Coverage:
  Defined terms: 26
  Terms with usage links: 26 (100.0%)
  Total term usages: 349
  Articles using terms: 88

Semantic Extraction:
  Rights found: 60 (in 24 articles)
  Obligations found: 70 (in 41 articles)
  Known GDPR rights: 6/6

Structure Quality:
  Articles: 99 (expected: 99, 100.0%)
  Chapters: 11 (expected: 11, 100.0%)
  Content quality: 100.0% articles with content
  Structure score: 100.0%

Component Scores:
  References:    98.8% (weight: 25%)
  Connectivity:  78.8% (weight: 20%)
  Definitions:   100.0% (weight: 20%)
  Semantics:     100.0% (weight: 20%)
  Structure:     100.0% (weight: 15%)

Overall Score: 95.5%
Threshold: 80.0%
Status: PASS
```

### Gate-Based Validation Pipeline

```bash
regula validate --source testdata/gdpr.txt --check gates
```

Four sequential gates run in under 84 microseconds total:

| Gate | Name      | Score   | Duration | What It Checks                              |
|------|-----------|---------|----------|---------------------------------------------|
| V0   | Schema    | 100.0%  | 15 µs   | File readable, not empty, valid size         |
| V1   | Structure | 100.0%  | 10 µs   | Has articles, structure complete, density    |
| V2   | Coverage  | 84.2%   | 14 µs   | Definitions, references, semantic coverage   |
| V3   | Quality   | 83.8%   | 43 µs   | Resolution rate, confidence, graph connectivity |

**Overall: 92.0%** — All four gates pass.

### Validation Profiles

Regula supports regulation-specific validation profiles:

```bash
# Use a built-in profile
regula validate --source testdata/gdpr.txt --profile GDPR

# Auto-suggest a profile based on document analysis
regula validate --source testdata/gdpr.txt --suggest-profile

# Generate a profile YAML file from document analysis
regula validate --source testdata/gdpr.txt --generate-profile my-profile.yaml

# Load a custom profile
regula validate --source testdata/gdpr.txt --load-profile my-profile.yaml
```

Built-in profiles:

| Profile  | Expected Articles | Expected Chapters | Threshold |
|----------|------------------:|-------------------:|----------:|
| GDPR     | 99                | 11                 | 80%       |
| CCPA     | 31                | 5                  | 75%       |
| Generic  | flexible          | flexible           | 70%       |

### Report Formats

```bash
# HTML report
regula validate --source testdata/gdpr.txt --format html --report report.html

# Markdown report
regula validate --source testdata/gdpr.txt --format markdown --report report.md

# JSON report (for programmatic use)
regula validate --source testdata/gdpr.txt --format json --report report.json
```

---

## 6. Inter-Legislation Modelling

Regula's knowledge graph represents legislation as a connected web of provisions, enabling structural and semantic analysis of regulatory frameworks.

### Cross-Reference Network

The GDPR contains **258 internal cross-references** connecting its 99 articles. These references form a directed graph that reveals the regulatory structure.

```bash
regula query --source testdata/gdpr.txt --template bidirectional
```

Shows pairs of articles that reference each other, revealing mutual dependencies in the regulatory framework.

### Term Propagation as a Modelling Tool

Defined terms create semantic links across articles. When a term is defined in Article 4 and used across multiple provisions, changing that definition has cascading effects.

```bash
regula query --source testdata/gdpr.txt --template term-usage
```

Example: The term "consent" appears in 13 articles. The term "controller" appears in 16+ articles. These terms are semantic integration points — changing their definitions affects the interpretation of every article that uses them.

### Combining Impact Analysis with Scenario Matching

For comprehensive regulatory modelling, combine impact analysis with scenario matching:

```bash
# First: What articles are affected by changing Art 17?
regula impact --source testdata/gdpr.txt --provision Art17

# Then: How does an erasure request interact with those affected provisions?
regula match --source testdata/gdpr.txt --scenario erasure_request
```

The impact analysis identifies **42 provisions** affected by changes to Article 17. The scenario matching identifies **88 provisions** relevant to an erasure request. The overlap between these sets reveals which provisions are both structurally connected and functionally relevant to erasure operations.

### Relationship Graph Summary

```bash
regula export --source testdata/gdpr.txt --format summary
```

**Output:**

```
Relationship Graph Summary
==========================

Total relationships: 3141

Relationship Types:
  reg:partOf              1019
  reg:belongsTo            515
  reg:usesTerm             349
  reg:resolvedTarget       297
  reg:referencedBy         179
  reg:references           179
  reg:hasRecital           173
  reg:contains             133
  reg:hasArticle            99
  reg:imposesObligation     62
  reg:grantsRight           44
  reg:defines               26
  reg:definedIn             26
  reg:hasSection            15
  reg:externalRef           13
  reg:hasChapter            11
  reg:repealedBy             1

Most Referenced Articles:
  Article 6:  9 incoming references
  Article 9:  7 incoming references
  Article 43: 6 incoming references
  Article 65: 6 incoming references
  Article 40: 6 incoming references

Articles With Most Outgoing References:
  Article 70: 18 outgoing references
  Article 12: 11 outgoing references
  Article 58: 9 outgoing references
  Article 40: 8 outgoing references
  Article 11: 7 outgoing references
```

---

## 7. Exporting the Knowledge Graph

The `export` command serializes the knowledge graph into standard formats for use with external tools.

### Available Formats

| Format   | Flag       | Use Case                                    |
|----------|------------|---------------------------------------------|
| Summary  | `summary`  | Relationship statistics and top articles     |
| Turtle   | `turtle`   | Standard RDF serialization for SPARQL tools  |
| RDF/XML  | `rdfxml`   | W3C standard for XML-based RDF toolchains    |
| JSON-LD  | `jsonld`   | JSON-based RDF for web applications          |
| DOT      | `dot`      | GraphViz visualization                       |
| JSON     | `json`     | General-purpose structured data              |

### Turtle Export

```bash
regula export --source testdata/gdpr.txt --format turtle --output gdpr.ttl
```

Produces ~9,548 lines of Turtle RDF covering all 8,495 triples. Compatible with Apache Jena, RDFLib, Blazegraph, and any SPARQL endpoint.

### RDF/XML Export

```bash
regula export --source testdata/gdpr.txt --format rdfxml --output gdpr.rdf
```

Produces W3C-compliant RDF/XML:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<rdf:RDF
    xmlns:dc="http://purl.org/dc/elements/1.1/"
    xmlns:eli="http://data.europa.eu/eli/ontology#"
    xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
    xmlns:reg="https://regula.dev/ontology#">

  <rdf:Description rdf:about="https://regula.dev/regulations/GDPR:Art17">
    <rdf:type rdf:resource="https://regula.dev/ontology#Article"/>
    <reg:title>Right to erasure ('right to be forgotten')</reg:title>
    <reg:references rdf:resource="https://regula.dev/regulations/GDPR:Art6"/>
    <reg:grantsRight rdf:resource="https://regula.dev/regulations/GDPR:Right:17:RightToErasure"/>
  </rdf:Description>

</rdf:RDF>
```

Both `rdfxml` and `xml` are accepted as format names.

### GraphViz Visualization

```bash
regula export --source testdata/gdpr.txt --format dot --output gdpr.dot
dot -Tpng gdpr.dot -o gdpr.png
```

The DOT output uses color coding: light blue for articles, light green for chapters, light yellow for sections, light pink for definitions, light coral for rights.

### JSON-LD Export

```bash
regula export --source testdata/gdpr.txt --format jsonld --output gdpr.jsonld
```

JSON-LD format for web application integration.

---

## 8. Performance Summary

All benchmarks measured on GDPR (88 pages, 99 articles, 8,495 triples).

### End-to-End Timing

| Operation                        | Wall Time  |
|----------------------------------|------------|
| Full document ingest             | ~1.2s      |
| Full validation                  | ~1.5s      |
| Gate validation (4 gates)        | 84 µs      |
| Impact analysis (depth 2)        | ~0.84s     |
| Scenario matching                | ~1.3-1.6s  |
| Profile suggestion               | ~1.3s      |
| Export (turtle/rdfxml/summary)   | ~0.9-1.0s  |

### Query Execution Time (Excluding Ingestion)

| Query                      | Execution Time |
|----------------------------|---------------|
| Definitions (26 terms)     | 86 µs         |
| Bidirectional references   | 224 µs        |
| DESCRIBE article           | 22 µs         |

### Manual Analysis vs. Regula

| Task                                             | Manual Estimate | Regula  |
|--------------------------------------------------|-----------------|---------|
| Read and catalog 99 GDPR articles                | 4-8 hours       | 1.2s    |
| Extract all 26 defined terms and definitions     | 1-2 hours       | 1.2s    |
| Map 258 cross-references between articles        | 8-16 hours      | 1.2s    |
| Identify 60 rights across 24 articles            | 4-8 hours       | 1.2s    |
| Identify 69 obligations across 41 articles       | 4-8 hours       | 1.2s    |
| Trace impact of amending one article (depth 2)   | 2-4 hours       | 0.84s   |
| Find all provisions for a compliance scenario    | 1-3 hours       | 1.3s    |
| Validate extraction completeness                 | 2-4 hours       | 1.5s    |
| Generate relationship summary                    | 4-8 hours       | 0.9s    |

**Total for full GDPR analysis: 30-61 hours manual work reduced to under 10 seconds.**

---

## 9. Complete Workflow Example

A full regulatory analysis workflow:

```bash
# 1. Ingest the regulation and build the knowledge graph
regula ingest --source testdata/gdpr.txt

# 2. Validate extraction quality
regula validate --source testdata/gdpr.txt

# 3. Run gate-based validation pipeline
regula validate --source testdata/gdpr.txt --check gates

# 4. Explore structure
regula query --source testdata/gdpr.txt --template articles
regula query --source testdata/gdpr.txt --template chapters
regula query --source testdata/gdpr.txt --template definitions
regula query --source testdata/gdpr.txt --template hierarchy

# 5. Analyze rights and obligations
regula query --source testdata/gdpr.txt --template rights
regula query --source testdata/gdpr.txt --template obligations
regula query --source testdata/gdpr.txt --template right-types
regula query --source testdata/gdpr.txt --template obligation-types

# 6. Explore cross-references
regula query --source testdata/gdpr.txt --template references
regula query --source testdata/gdpr.txt --template most-referenced
regula query --source testdata/gdpr.txt --template bidirectional

# 7. Analyze impact of proposed amendments
regula impact --source testdata/gdpr.txt --provision Art17
regula impact --source testdata/gdpr.txt --provision Art6 --depth 3

# 8. Match compliance scenarios
regula match --source testdata/gdpr.txt --scenario data_breach
regula match --source testdata/gdpr.txt --scenario consent_withdrawal
regula match --source testdata/gdpr.txt --scenario erasure_request
regula match --source testdata/gdpr.txt --scenario access_request

# 9. Export for external tools
regula export --source testdata/gdpr.txt --format summary
regula export --source testdata/gdpr.txt --format rdfxml --output gdpr.rdf
regula export --source testdata/gdpr.txt --format turtle --output gdpr.ttl
regula export --source testdata/gdpr.txt --format jsonld --output gdpr.jsonld
regula export --source testdata/gdpr.txt --format dot --output gdpr.dot
```

Each command runs independently. The source document is re-ingested on each invocation, keeping the workflow stateless and reproducible.

---

## 10. All Output Formats

| Command    | Available Formats                                                      |
|------------|------------------------------------------------------------------------|
| `query`    | table, json, csv (SELECT); turtle, ntriples, json (CONSTRUCT/DESCRIBE) |
| `validate` | text, json, html, markdown                                             |
| `impact`   | text, json, table                                                      |
| `match`    | text, json, table                                                      |
| `export`   | json, dot, turtle, jsonld, rdfxml, summary                             |

Example with JSON output for programmatic consumption:

```bash
regula match --source testdata/gdpr.txt --scenario data_breach --format json | jq '.direct[]'
regula validate --source testdata/gdpr.txt --format json | jq '.overallScore'
regula impact --source testdata/gdpr.txt --provision Art17 --format json | jq '.summary'
```

---

## Next Steps

- Review [TESTING.md](TESTING.md) for development and testing strategies
- See [ARCHITECTURE.md](ARCHITECTURE.md) for system design details
- Check [ROADMAP.md](ROADMAP.md) for upcoming features
