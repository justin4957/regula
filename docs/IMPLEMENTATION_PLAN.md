# Implementation Plan: Arbitrary Legislation Support

## Executive Summary

This plan outlines the upgrade of the Regula CLI to handle arbitrary legislation ingestion with automatic parsing, reference extraction, RDF/Turtle export, and effective external reference detection. The implementation is structured into 5 epics across 4 milestones, with persistent validation gates to ensure quality at each phase.

---

## Table of Contents

1. [Vision and Goals](#vision-and-goals)
2. [Current State Assessment](#current-state-assessment)
3. [Architecture Overview](#architecture-overview)
4. [Epics and User Stories](#epics-and-user-stories)
5. [Milestones and Timeline](#milestones-and-timeline)
6. [Persistent Validation Framework](#persistent-validation-framework)
7. [Technical Specifications](#technical-specifications)
8. [Risk Assessment](#risk-assessment)
9. [Success Metrics](#success-metrics)

---

## Vision and Goals

### Vision
Transform Regula into a universal legislative knowledge graph engine capable of ingesting any regulatory document format, automatically detecting its structure, extracting cross-references (internal and external), and producing queryable RDF/Turtle representations.

### Primary Goals

1. **Universal Format Detection**: Automatically identify and parse legislative documents from any jurisdiction without manual configuration
2. **Comprehensive Reference Extraction**: Detect and classify both internal cross-references and external citations to other laws, treaties, and regulations
3. **Standard RDF/Turtle Output**: Export knowledge graphs in standard RDF formats (Turtle, N-Triples, RDF/XML) with proper URI schemes
4. **External Reference Resolution**: Link external references to canonical identifiers and optionally fetch referenced documents
5. **Self-Validating Pipeline**: Continuous quality metrics throughout ingestion with configurable thresholds

---

## Current State Assessment

### Supported Formats
| Format | Status | Coverage |
|--------|--------|----------|
| EU Regulations (GDPR-style) | Full | Chapters, Articles, Recitals, Definitions |
| US California Civil Code (CCPA) | Full | Sections with California numbering |
| US Virginia Code (VCDPA) | Partial | Section 59.1-xxx format |
| UK Legislation | Not Supported | - |
| International Treaties | Not Supported | - |
| Other Jurisdictions | Not Supported | - |

### Current Limitations
1. Hard-coded format detection patterns
2. Reference extraction limited to EU/US patterns
3. External references not linked to canonical sources
4. No RDF/Turtle export (only JSON, DOT, summary)
5. Validation profiles require manual curation
6. No schema validation for input documents

---

## Architecture Overview

### Target Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           REGULA CLI                                     │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │   ingest    │  │   query     │  │   export    │  │  validate   │    │
│  │  --auto     │  │  --sparql   │  │  --turtle   │  │  --strict   │    │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘    │
├─────────┴────────────────┴────────────────┴────────────────┴────────────┤
│                        FORMAT DETECTION ENGINE                           │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  Pattern Library (Pluggable)                                     │    │
│  │  ├── EU Patterns (GDPR, Directives, Regulations)                │    │
│  │  ├── US Federal Patterns (USC, CFR, Public Laws)                │    │
│  │  ├── US State Patterns (CA, VA, NY, TX, FL, CO, CT, UT, IA)    │    │
│  │  ├── UK Patterns (Acts, Statutory Instruments)                  │    │
│  │  ├── International Patterns (Treaties, Conventions)             │    │
│  │  └── Custom Patterns (User-defined)                             │    │
│  └─────────────────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────────────────┤
│                        EXTRACTION PIPELINE                               │
│  ┌────────────┐   ┌────────────┐   ┌────────────┐   ┌────────────┐     │
│  │  Structure │ → │ Definition │ → │ Reference  │ → │  Semantic  │     │
│  │  Extractor │   │ Extractor  │   │ Extractor  │   │ Extractor  │     │
│  └────────────┘   └────────────┘   └────────────┘   └────────────┘     │
│         ↓                ↓                ↓                ↓            │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                    VALIDATION GATE                               │    │
│  │  • Structure Score ≥ threshold                                  │    │
│  │  • Definition Coverage ≥ threshold                              │    │
│  │  • Reference Resolution ≥ threshold                             │    │
│  │  • Semantic Extraction ≥ threshold                              │    │
│  └─────────────────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────────────────┤
│                     EXTERNAL REFERENCE ENGINE                            │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  Citation Resolver                                               │    │
│  │  ├── Legal Citation Parser (Bluebook, OSCOLA, etc.)             │    │
│  │  ├── Canonical URI Generator                                     │    │
│  │  ├── External Source Registry                                    │    │
│  │  │   ├── EUR-Lex (EU legislation)                               │    │
│  │  │   ├── US Code (uscode.house.gov)                             │    │
│  │  │   ├── CFR (ecfr.gov)                                         │    │
│  │  │   ├── UK Legislation (legislation.gov.uk)                    │    │
│  │  │   └── Custom Registries                                       │    │
│  │  └── Link Validation (optional HTTP HEAD checks)                │    │
│  └─────────────────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────────────────┤
│                        KNOWLEDGE GRAPH STORE                             │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  Triple Store with Multi-Format Export                          │    │
│  │  ├── Internal: SPO/POS/OSP indexes                              │    │
│  │  ├── Export: Turtle, N-Triples, RDF/XML, JSON-LD               │    │
│  │  ├── Query: SPARQL subset support                               │    │
│  │  └── Schema: ELI, FRBR, Dublin Core compatible                  │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Epics and User Stories

### Epic 1: Universal Format Detection Engine
**Goal**: Automatically detect and parse any legislative document format without manual configuration.

#### E1.1: Pattern Library Architecture
- **E1.1.1**: Design pluggable pattern registry with hot-loading support
- **E1.1.2**: Create pattern definition schema (YAML/JSON configuration)
- **E1.1.3**: Implement pattern matching scoring algorithm with confidence levels
- **E1.1.4**: Add fallback to generic parsing when no pattern matches

#### E1.2: Jurisdiction Pattern Libraries
- **E1.2.1**: Refactor existing EU patterns into pattern library format
- **E1.2.2**: Refactor existing US patterns into pattern library format
- **E1.2.3**: Add UK legislation patterns (Acts, SIs, SSIs)
- **E1.2.4**: Add international treaty patterns (UN, WTO, bilateral)
- **E1.2.5**: Add additional US state patterns (NY, TX, FL, CO, CT, UT, IA)

#### E1.3: Adaptive Structure Detection
- **E1.3.1**: Implement hierarchical structure inference from whitespace/numbering
- **E1.3.2**: Add definition section auto-detection (not hardcoded article numbers)
- **E1.3.3**: Implement preamble/enacting clause detection across formats
- **E1.3.4**: Add amendment/modification detection patterns

**Acceptance Criteria**:
- [ ] Parser correctly identifies format for 10+ different jurisdiction samples
- [ ] Pattern library loads from external configuration files
- [ ] Confidence scores provided for format detection
- [ ] Unknown formats parsed with generic extractor (graceful degradation)

---

### Epic 2: Comprehensive Reference Extraction System
**Goal**: Extract and classify all cross-references with support for complex citation formats.

#### E2.1: Citation Parser Framework
- **E2.1.1**: Design extensible citation parser interface
- **E2.1.2**: Implement Bluebook citation parser (US legal citations)
- **E2.1.3**: Implement OSCOLA citation parser (UK/Commonwealth)
- **E2.1.4**: Implement EU citation format parser (OJ references, CELEX)
- **E2.1.5**: Add citation normalization to canonical form

#### E2.2: Internal Reference Detection
- **E2.2.1**: Enhance relative reference resolution ("this section", "paragraph above")
- **E2.2.2**: Add range reference support ("Articles 15-22", "Sections 1798.100-199")
- **E2.2.3**: Implement conditional reference detection ("subject to Article X")
- **E2.2.4**: Add amendment reference tracking ("as amended by")

#### E2.3: External Reference Classification
- **E2.3.1**: Design external reference taxonomy (treaties, statutes, regulations, cases)
- **E2.3.2**: Implement jurisdiction detection from citation format
- **E2.3.3**: Add temporal reference handling (specific versions, "as in force on")
- **E2.3.4**: Create reference confidence scoring based on pattern match quality

#### E2.4: Reference Resolution Pipeline
- **E2.4.1**: Build internal reference resolver with disambiguation
- **E2.4.2**: Create external reference URI generator
- **E2.4.3**: Implement reference graph builder (bidirectional links)
- **E2.4.4**: Add unresolved reference reporting with suggestions

**Acceptance Criteria**:
- [ ] 95%+ resolution rate for internal references in test corpus
- [ ] External references classified by jurisdiction and document type
- [ ] Citation normalization produces consistent canonical forms
- [ ] Reference confidence scores correlate with actual accuracy

---

### Epic 3: RDF/Turtle Export and Schema Compliance
**Goal**: Export knowledge graphs in standard RDF formats with legal ontology alignment.

#### E3.1: RDF Serialization Formats
- **E3.1.1**: Implement Turtle (TTL) export with proper prefix declarations
- **E3.1.2**: Implement N-Triples export for streaming/bulk loading
- **E3.1.3**: Implement RDF/XML export for legacy system compatibility
- **E3.1.4**: Implement JSON-LD export with context documents
- **E3.1.5**: Add format auto-detection from output file extension

#### E3.2: Legal Ontology Alignment
- **E3.2.1**: Integrate ELI (European Legislation Identifier) vocabulary
- **E3.2.2**: Add FRBR (Functional Requirements for Bibliographic Records) support
- **E3.2.3**: Include Dublin Core metadata properties
- **E3.2.4**: Design custom regula: namespace for extended properties
- **E3.2.5**: Create ontology mapping configuration for custom vocabularies

#### E3.3: URI Scheme Design
- **E3.3.1**: Design jurisdiction-aware URI scheme
- **E3.3.2**: Implement CELEX-compatible URIs for EU legislation
- **E3.3.3**: Implement US Code citation URIs
- **E3.3.4**: Add configurable URI templates per jurisdiction
- **E3.3.5**: Support relative vs absolute URI modes

#### E3.4: SPARQL Query Enhancement
- **E3.4.1**: Extend query parser with additional SPARQL features
- **E3.4.2**: Add CONSTRUCT query support for graph extraction
- **E3.4.3**: Implement DESCRIBE queries for entity summaries
- **E3.4.4**: Add federated query hints for external endpoints

**Acceptance Criteria**:
- [ ] Turtle output validates with standard RDF validators
- [ ] Exported graphs loadable in Jena, rdflib, and other RDF tools
- [ ] URIs dereferenceable (where applicable) or follow standard patterns
- [ ] Ontology alignment documented and configurable

---

### Epic 4: External Reference Detection and Linking
**Goal**: Detect external references, generate canonical URIs, and optionally validate/fetch linked documents.

#### E4.1: External Source Registry
- **E4.1.1**: Design external source registry schema
- **E4.1.2**: Implement EUR-Lex connector (EU legislation database)
- **E4.1.3**: Implement US Code connector (uscode.house.gov)
- **E4.1.4**: Implement eCFR connector (Code of Federal Regulations)
- **E4.1.5**: Implement UK legislation.gov.uk connector
- **E4.1.6**: Add custom registry support for proprietary sources

#### E4.2: Citation-to-URI Mapping
- **E4.2.1**: Build citation parser to structured representation
- **E4.2.2**: Create URI template engine for each jurisdiction
- **E4.2.3**: Implement fuzzy matching for variant citation formats
- **E4.2.4**: Add citation disambiguation using context

#### E4.3: Link Validation
- **E4.3.1**: Implement HEAD request validation for HTTP URIs
- **E4.3.2**: Add batch validation with rate limiting
- **E4.3.3**: Create validation caching to avoid repeated checks
- **E4.3.4**: Generate link validation report with broken link detection

#### E4.4: Document Fetching (Optional)
- **E4.4.1**: Design recursive ingestion pipeline
- **E4.4.2**: Implement depth-limited document fetching
- **E4.4.3**: Add document format conversion for fetched content
- **E4.4.4**: Create federated graph with cross-document references

**Acceptance Criteria**:
- [ ] External references link to valid URIs for known registries
- [ ] Link validation reports broken/outdated references
- [ ] Optional document fetching respects rate limits and robots.txt
- [ ] Federated queries span ingested documents

---

### Epic 5: Persistent Validation and Quality Framework
**Goal**: Continuous quality validation throughout the ingestion pipeline with configurable thresholds.

#### E5.1: Validation Gate System
- **E5.1.1**: Design validation checkpoint interface
- **E5.1.2**: Implement post-parsing validation gate
- **E5.1.3**: Implement post-extraction validation gate
- **E5.1.4**: Implement post-resolution validation gate
- **E5.1.5**: Add configurable threshold overrides per gate

#### E5.2: Quality Metrics Dashboard
- **E5.2.1**: Create real-time metrics collection during ingestion
- **E5.2.2**: Implement quality trend tracking across ingestions
- **E5.2.3**: Add regression detection for repeated ingestions
- **E5.2.4**: Build quality report generator (HTML, JSON, Markdown)

#### E5.3: Adaptive Validation Profiles
- **E5.3.1**: Implement profile auto-generation from document analysis
- **E5.3.2**: Add machine learning for threshold tuning
- **E5.3.3**: Create profile inheritance for regulation families
- **E5.3.4**: Support profile versioning and comparison

#### E5.4: Validation Test Suite
- **E5.4.1**: Create golden file test corpus (10+ jurisdictions)
- **E5.4.2**: Implement differential testing for parser changes
- **E5.4.3**: Add fuzzing for edge case detection
- **E5.4.4**: Create continuous integration validation workflow

**Acceptance Criteria**:
- [ ] All validation gates configurable via CLI flags and config files
- [ ] Quality metrics available in real-time during ingestion
- [ ] Regression detection catches parsing/extraction degradation
- [ ] Test corpus covers 10+ distinct jurisdiction formats

---

## Milestones and Timeline

### Milestone 1: Foundation (Weeks 1-4)
**Theme**: Pattern Library and Core Refactoring

| Week | Epic | Deliverables |
|------|------|--------------|
| 1 | E1.1 | Pattern library architecture, schema design |
| 2 | E1.2 | EU/US pattern refactoring, UK patterns |
| 3 | E1.3 | Adaptive structure detection |
| 4 | E5.4 | Golden file test corpus, CI validation |

**Validation Gate M1**:
- [ ] Pattern library loads all existing patterns from config
- [ ] UK legislation sample parses with >80% structure accuracy
- [ ] Test corpus established with 5+ jurisdictions
- [ ] No regression in GDPR/CCPA/VCDPA parsing

---

### Milestone 2: Reference Engine (Weeks 5-8)
**Theme**: Comprehensive Reference Extraction

| Week | Epic | Deliverables |
|------|------|--------------|
| 5 | E2.1 | Citation parser framework, Bluebook parser |
| 6 | E2.2 | Enhanced internal reference detection |
| 7 | E2.3 | External reference classification |
| 8 | E2.4 | Reference resolution pipeline |

**Validation Gate M2**:
- [ ] Internal reference resolution ≥95% on test corpus
- [ ] External references classified with jurisdiction
- [ ] Citation normalization produces consistent output
- [ ] Reference graph includes bidirectional links

---

### Milestone 3: RDF Export (Weeks 9-12)
**Theme**: Standard RDF Output and Ontology Alignment

| Week | Epic | Deliverables |
|------|------|--------------|
| 9 | E3.1 | Turtle and N-Triples export |
| 10 | E3.2 | ELI/FRBR/Dublin Core alignment |
| 11 | E3.3 | URI scheme implementation |
| 12 | E3.4 | SPARQL query enhancements |

**Validation Gate M3**:
- [ ] Turtle output validates with W3C RDF validator
- [ ] Exported graphs loadable in Apache Jena
- [ ] URIs follow jurisdiction-standard patterns
- [ ] CONSTRUCT queries produce valid subgraphs

---

### Milestone 4: External Linking (Weeks 13-16)
**Theme**: External Reference Resolution and Validation

| Week | Epic | Deliverables |
|------|------|--------------|
| 13 | E4.1 | External source registry, EUR-Lex connector |
| 14 | E4.2 | Citation-to-URI mapping engine |
| 15 | E4.3 | Link validation system |
| 16 | E4.4, E5.1-3 | Document fetching, validation dashboard |

**Validation Gate M4**:
- [ ] EUR-Lex and US Code references resolve to valid URIs
- [ ] Link validation detects >90% of broken links
- [ ] Quality dashboard displays real-time metrics
- [ ] Full pipeline processes arbitrary legislation end-to-end

---

## Persistent Validation Framework

### Validation Checkpoints

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        INGESTION PIPELINE                                │
│                                                                          │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐          │
│  │  INPUT   │ →  │  PARSE   │ →  │ EXTRACT  │ →  │ RESOLVE  │ → OUTPUT │
│  │ Document │    │ Structure│    │ Entities │    │References│          │
│  └────┬─────┘    └────┬─────┘    └────┬─────┘    └────┬─────┘          │
│       │               │               │               │                 │
│       ▼               ▼               ▼               ▼                 │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐              │
│  │  GATE   │    │  GATE   │    │  GATE   │    │  GATE   │              │
│  │   V0    │    │   V1    │    │   V2    │    │   V3    │              │
│  │ Schema  │    │Structure│    │Coverage │    │Resolution│              │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘              │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Gate Definitions

#### Gate V0: Input Schema Validation
| Metric | Threshold | Action on Failure |
|--------|-----------|-------------------|
| File readable | Required | Abort |
| Text encoding valid (UTF-8) | Required | Abort |
| Minimum content length | 1000 chars | Warning |
| Maximum file size | 50 MB | Abort |

#### Gate V1: Structure Validation
| Metric | Threshold | Action on Failure |
|--------|-----------|-------------------|
| Format detected | Confidence ≥ 0.5 | Warning (use generic) |
| Articles/Sections found | ≥ 1 | Warning |
| Hierarchical structure | Valid nesting | Warning |
| Title extracted | Non-empty | Warning |

#### Gate V2: Extraction Coverage
| Metric | Threshold | Action on Failure |
|--------|-----------|-------------------|
| Definition coverage | ≥ 80% terms used | Warning |
| Reference detection | ≥ 10 refs OR ≥ 1 per 5 articles | Info |
| Semantic extraction | ≥ 1 right OR obligation | Info |
| Content completeness | ≥ 90% articles with text | Warning |

#### Gate V3: Resolution Quality
| Metric | Threshold | Action on Failure |
|--------|-----------|-------------------|
| Internal resolution | ≥ 90% | Warning |
| External classification | 100% external refs classified | Info |
| URI generation | 100% refs have URI | Warning |
| Confidence average | ≥ 0.7 | Info |

### Persistent Validation Configuration

```yaml
# .regula/validation.yaml
validation:
  gates:
    v0_schema:
      enabled: true
      fail_on_warning: false
      thresholds:
        min_content_length: 1000
        max_file_size_mb: 50

    v1_structure:
      enabled: true
      fail_on_warning: false
      thresholds:
        format_confidence: 0.5
        min_articles: 1

    v2_extraction:
      enabled: true
      fail_on_warning: false
      thresholds:
        definition_coverage: 0.8
        content_completeness: 0.9

    v3_resolution:
      enabled: true
      fail_on_warning: true
      thresholds:
        internal_resolution: 0.9
        confidence_average: 0.7

  reporting:
    format: json  # json, text, html
    output: validation-report.json
    include_details: true
    track_history: true
```

### CLI Validation Flags

```bash
# Strict mode: fail on any warning
regula ingest --source doc.txt --strict

# Skip specific gates
regula ingest --source doc.txt --skip-gate v2

# Custom thresholds
regula ingest --source doc.txt --threshold internal_resolution=0.8

# Validation-only mode (no output)
regula validate --source doc.txt --gates all

# Generate validation report
regula validate --source doc.txt --report validation.html
```

---

## Technical Specifications

### Pattern Library Schema

```yaml
# patterns/uk-legislation.yaml
name: UK Primary Legislation
version: "1.0"
jurisdiction: UK
format_id: uk_primary

detection:
  required_indicators:
    - pattern: "^An Act"
      weight: 3
    - pattern: "BE IT ENACTED"
      weight: 3
    - pattern: "\\[\\d{4}\\s+c\\.\\s*\\d+\\]"  # [2018 c. 12]
      weight: 2

  negative_indicators:
    - pattern: "REGULATION \\(EU\\)"
      weight: -5

structure:
  hierarchy:
    - type: part
      pattern: "^PART\\s+(\\d+|[IVXLC]+)"
      title_follows: true
    - type: chapter
      pattern: "^CHAPTER\\s+(\\d+|[IVXLC]+)"
      title_follows: true
    - type: section
      pattern: "^(\\d+)\\s+"
      title_inline: true
    - type: subsection
      pattern: "^\\((\\d+)\\)"
    - type: paragraph
      pattern: "^\\(([a-z])\\)"

definitions:
  location:
    - section_title: "Interpretation"
    - section_title: "Definitions"
  pattern: '"([^"]+)"\\s+(?:means|has the meaning)'

references:
  internal:
    - pattern: "section\\s+(\\d+)"
      target: section
    - pattern: "sections\\s+(\\d+)\\s+(?:to|and)\\s+(\\d+)"
      target: section_range
    - pattern: "subsection\\s+\\((\\d+)\\)"
      target: subsection
  external:
    - pattern: "(\\d{4})\\s+c\\.\\s*(\\d+)"
      type: uk_act
      uri_template: "http://www.legislation.gov.uk/ukpga/{year}/{chapter}"
    - pattern: "S\\.I\\.\\s*(\\d{4})/(\\d+)"
      type: uk_si
      uri_template: "http://www.legislation.gov.uk/uksi/{year}/{number}"
```

### External Reference Registry Schema

```yaml
# registries/eur-lex.yaml
name: EUR-Lex
id: eur_lex
base_uri: "http://data.europa.eu/eli"

document_types:
  regulation:
    celex_prefix: "3"
    uri_pattern: "{base_uri}/reg/{year}/{number}/oj"
  directive:
    celex_prefix: "3"
    uri_pattern: "{base_uri}/dir/{year}/{number}/oj"
  decision:
    celex_prefix: "3"
    uri_pattern: "{base_uri}/dec/{year}/{number}/oj"

citation_patterns:
  - regex: "Regulation\\s+\\(E[CU]\\)\\s+(No\\s+)?(\\d+)/(\\d+)"
    groups:
      number: 2
      year: 3
    type: regulation
  - regex: "Directive\\s+(\\d+)/(\\d+)/E[CU]"
    groups:
      year: 1
      number: 2
    type: directive

validation:
  endpoint: "https://eur-lex.europa.eu/legal-content/EN/TXT/?uri=CELEX:{celex}"
  method: HEAD
  success_codes: [200, 301, 302]
```

### RDF Export Schema

```turtle
# Output prefix declarations
@prefix eli: <http://data.europa.eu/eli/ontology#> .
@prefix frbr: <http://purl.org/vocab/frbr/core#> .
@prefix dct: <http://purl.org/dc/terms/> .
@prefix reg: <https://regula.dev/ontology#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .

# Example article export
<https://regula.dev/regulations/GDPR/art/17> a eli:LegalResourceSubdivision ;
    eli:id_local "Article 17" ;
    eli:title "Right to erasure ('right to be forgotten')"@en ;
    eli:is_part_of <https://regula.dev/regulations/GDPR> ;
    dct:isReferencedBy <https://regula.dev/regulations/GDPR/art/19> ;
    dct:references <https://regula.dev/regulations/GDPR/art/6> ;
    reg:grantsRight reg:RightToErasure ;
    reg:text "1. The data subject shall have the right..."@en .

# External reference
<https://regula.dev/regulations/GDPR/ref/directive-95-46-ec> a reg:ExternalReference ;
    reg:citationText "Directive 95/46/EC" ;
    reg:resolvedUri <http://data.europa.eu/eli/dir/1995/46/oj> ;
    reg:jurisdiction "EU" ;
    reg:documentType "directive" ;
    reg:validationStatus "verified" ;
    reg:validatedAt "2024-01-15T10:30:00Z"^^xsd:dateTime .
```

---

## Risk Assessment

### Technical Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Pattern library performance with many patterns | Medium | Medium | Implement pattern indexing and early-exit optimization |
| Citation format ambiguity | High | Medium | Use context-aware disambiguation and confidence scoring |
| External service rate limiting | High | Low | Implement caching and batch processing with backoff |
| RDF serialization edge cases | Medium | Low | Use established libraries (go-rdf) and comprehensive testing |
| Format detection false positives | Medium | Medium | Multi-indicator voting with confidence thresholds |

### Schedule Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| UK pattern complexity underestimated | Medium | Medium | Start with primary legislation, defer SIs |
| External registry API changes | Low | High | Abstract registry interface, monitor for changes |
| Validation framework scope creep | Medium | Medium | Fixed gate definitions, defer ML features |

---

## Success Metrics

### Quantitative Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Format detection accuracy | ≥ 95% | Correct format on test corpus |
| Internal reference resolution | ≥ 95% | Resolved/Total internal refs |
| External reference classification | 100% | All external refs have jurisdiction |
| RDF validation pass rate | 100% | W3C validator on all exports |
| Ingestion throughput | ≥ 100 articles/sec | Benchmark on standard hardware |
| Test corpus coverage | ≥ 10 jurisdictions | Distinct format samples |

### Qualitative Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| CLI usability | Intuitive for new users | User testing feedback |
| Documentation completeness | All features documented | Coverage audit |
| Error message quality | Actionable suggestions | User feedback |
| Extension ease | New format in < 1 hour | Developer testing |

---

## Appendix A: Test Corpus Requirements

### Required Samples (Minimum 10)

| # | Jurisdiction | Document | Format |
|---|--------------|----------|--------|
| 1 | EU | GDPR (2016/679) | EU Regulation |
| 2 | EU | ePrivacy Directive (2002/58/EC) | EU Directive |
| 3 | US-CA | CCPA (Cal. Civ. Code 1798) | US State Code |
| 4 | US-VA | VCDPA (Va. Code 59.1-575) | US State Code |
| 5 | US-Fed | HIPAA (42 USC 1320d) | US Federal Code |
| 6 | US-Fed | 45 CFR Part 164 | US CFR |
| 7 | UK | Data Protection Act 2018 | UK Primary Legislation |
| 8 | UK | GDPR SI 2019/419 | UK Statutory Instrument |
| 9 | INT | UNCITRAL Model Law | International Model Law |
| 10 | AU | Privacy Act 1988 | Australian Legislation |

### Extended Samples (Optional)

| # | Jurisdiction | Document | Format |
|---|--------------|----------|--------|
| 11 | US-CO | CPA (C.R.S. 6-1-1301) | US State Code |
| 12 | US-CT | CTDPA (CGS 42-515) | US State Code |
| 13 | CA | PIPEDA | Canadian Federal |
| 14 | DE | BDSG | German Federal |
| 15 | FR | Loi Informatique | French National |

---

## Appendix B: CLI Command Reference (Target State)

```bash
# Ingest with auto-detection
regula ingest --source document.txt --output graph.ttl --format turtle

# Ingest with explicit format
regula ingest --source document.txt --format-hint uk_primary

# Ingest with strict validation
regula ingest --source document.txt --strict --fail-on-warning

# Validate only (no output)
regula validate --source document.txt --report validation.json

# Export to multiple formats
regula export --source graph.db --format turtle --output graph.ttl
regula export --source graph.db --format ntriples --output graph.nt
regula export --source graph.db --format rdfxml --output graph.rdf
regula export --source graph.db --format jsonld --output graph.jsonld

# Query with SPARQL
regula query --source graph.db --sparql "SELECT ?s ?p ?o WHERE { ?s ?p ?o } LIMIT 10"

# Validate external references
regula validate-links --source graph.ttl --registry eur-lex,uscode

# Fetch referenced documents
regula fetch-refs --source graph.ttl --depth 1 --output refs/
```

---

## Appendix C: Migration Path

### Phase 1: Non-Breaking Changes
- Add pattern library alongside existing parser
- New CLI flags optional with sensible defaults
- Existing validation profiles remain functional

### Phase 2: Deprecation Notices
- Warn on hardcoded pattern usage
- Suggest migration to pattern library
- Document breaking changes for next major version

### Phase 3: Breaking Changes (Major Version)
- Remove hardcoded patterns
- Require pattern library configuration
- Update all documentation and examples

---

*Document Version: 1.0*
*Last Updated: 2026-01-27*
*Authors: Claude Code*
