# Milestone 3: Extraction Pipeline - Validation Report

**Date:** 2026-01-27
**Status:** ✅ COMPLETE (4/5 required issues closed, 1 optional issue open)

## Overview

Milestone 3 focused on building the extraction pipeline to transform parsed regulatory documents into a queryable knowledge graph with semantic annotations.

## Issue Status

| Issue | Title | Status | PR |
|-------|-------|--------|-----|
| #12 | M3.1: Implement reference resolver | ✅ Closed | #32 |
| #13 | M3.4: Implement obligation/right extraction | ✅ Closed | #33 |
| #29 | M3.3: Build provision relationship graph | ✅ Closed | #34 |
| #30 | M3.5: Add validation command | ✅ Closed | #35 |
| #31 | M3.2: Implement LLM-assisted extraction | ⏸️ Open (Optional) | - |

**Note:** Issue #31 is marked as optional and deferred. The milestone targets are met without LLM assistance.

## Validation Results

### Overall Metrics (GDPR)

```
Overall Score: 94.7%
Threshold: 80.0%
Status: PASS
```

### Reference Resolution (#12)

**Target:** ≥85% resolution rate
**Achieved:** 100% resolution rate

```
Reference Resolution:
  Total references: 255
  Resolved: 242 (100.0%)
  Unresolved: 0
    - External: 13
    - Ambiguous: 0
    - Not found: 0
```

**Resolution Features:**
- Internal references fully resolved to article/paragraph/point URIs
- External references identified and tagged (Directives, Treaties)
- Range references supported ("Articles 13 to 18")
- Confidence scoring (High/Medium/Low)

### Semantic Extraction (#13)

**Target:** Identify key GDPR rights and obligations
**Achieved:** 100% of known rights detected

```
Semantic Extraction:
  Rights found: 60 (in 24 articles)
  Obligations found: 70 (in 41 articles)
  Known GDPR rights: 6/6
```

**Detected Rights:**
- Right of access (Art 15)
- Right to rectification (Art 16)
- Right to erasure (Art 17)
- Right to restriction (Art 18)
- Right to data portability (Art 20)
- Right to object (Art 21)

**Detected Obligation Types:**
- Consent obligations
- Breach notification obligations
- Security obligations
- Record-keeping obligations
- Impact assessment obligations
- Appointment obligations

### Relationship Graph (#29)

**Target:** Build connected graph with all relationship types
**Achieved:** 3,134 relationship edges

```
Relationship Graph Summary
==========================

Total relationships: 3134

Relationship Types:
  reg:partOf                1016
  reg:belongsTo             512
  reg:usesTerm              349
  reg:resolvedTarget        297
  reg:references            179
  reg:referencedBy          179
  reg:hasRecital            173
  reg:contains              133
  reg:hasArticle            99
  reg:imposesObligation     62
  reg:grantsRight           44
  reg:defines               26
  reg:definedIn             26
  reg:hasSection            15
  reg:externalRef           13
  reg:hasChapter            11
```

**Graph Features:**
- Bidirectional reference tracking
- Term usage linking (349 term-to-article links)
- Hierarchical containment (chapters → sections → articles)
- Export to JSON and DOT (Graphviz) formats

### Validation Command (#30)

**Target:** CI-ready validation with JSON output
**Achieved:** Comprehensive validation with configurable thresholds

```
Graph Connectivity:
  Total provisions: 99
  Connected: 78 (78.8%)
  Orphans: 21

Definition Coverage:
  Defined terms: 26
  Terms with usage links: 26 (100.0%)
  Total term usages: 349
  Articles using terms: 88
```

## Functional Examples

### 1. Ingest Regulation

```bash
$ regula ingest --source testdata/gdpr.txt --stats
Ingesting regulation from: testdata/gdpr.txt
  1. Parsing document structure... done (11 chapters, 99 articles)
  2. Extracting defined terms... done (26 definitions)
  3. Identifying cross-references... done (255 references)
  4. Extracting rights/obligations... done (60 rights, 69 obligations)
  5. Resolving cross-references... done (100% resolved)
  6. Building knowledge graph... done (8457 triples)

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

### 2. Query Rights by Article

```bash
$ regula query --source testdata/gdpr.txt --template rights | head -12
Template: rights
Description: Find articles that grant rights

+-------------------------------------------+--------------------------------------------------+---------------------------------------------+--------------------+
| article                                   | title                                            | right                                       | rightType          |
+-------------------------------------------+--------------------------------------------------+---------------------------------------------+--------------------+
| GDPR:Art15                                | Right of access by the data subject              | GDPR:Right:15:RightOfAccess                 | RightOfAccess      |
| GDPR:Art16                                | Right to rectification                           | GDPR:Right:16:RightToRectification          | RightToRectification|
| GDPR:Art17                                | Right to erasure ('right to be forgotten')       | GDPR:Right:17:RightToErasure                | RightToErasure     |
| GDPR:Art18                                | Right to restriction of processing               | GDPR:Right:18:RightToRestriction            | RightToRestriction |
| GDPR:Art20                                | Right to data portability                        | GDPR:Right:20:RightToDataPortability        | RightToDataPortability|
| GDPR:Art21                                | Right to object                                  | GDPR:Right:21:RightToObject                 | RightToObject      |
```

### 3. Query Obligations

```bash
$ regula query --source testdata/gdpr.txt --template obligations | head -12
Template: obligations
Description: Find articles that impose obligations

+-------------------------------------------+----------------------------------------------+-----------------------------------------------+---------------------------+
| article                                   | title                                        | oblig                                         | obligType                 |
+-------------------------------------------+----------------------------------------------+-----------------------------------------------+---------------------------+
| GDPR:Art30                                | Records of processing activities             | GDPR:Obligation:30:RecordKeepingObligation    | RecordKeepingObligation   |
| GDPR:Art32                                | Security of processing                       | GDPR:Obligation:32:SecurityObligation         | SecurityObligation        |
| GDPR:Art33                                | Notification of a personal data breach...   | GDPR:Obligation:33:BreachNotificationOblig... | BreachNotificationOblig...|
| GDPR:Art35                                | Data protection impact assessment            | GDPR:Obligation:35:ImpactAssessmentOblig...   | ImpactAssessmentOblig...  |
| GDPR:Art37                                | Designation of the data protection officer   | GDPR:Obligation:37:AppointmentObligation      | AppointmentObligation     |
```

### 4. Query Term Usage

```bash
$ regula query --source testdata/gdpr.txt --template term-usage | head -15
Template: term-usage
Description: Find which articles use defined terms

+-------------------------------------------+-------------------------+
| article                                   | term                    |
+-------------------------------------------+-------------------------+
| GDPR:Art46                                | binding corporate rules |
| GDPR:Art47                                | binding corporate rules |
| GDPR:Art49                                | binding corporate rules |
| GDPR:Art6                                 | consent                 |
| GDPR:Art7                                 | consent                 |
| GDPR:Art8                                 | consent                 |
| GDPR:Art9                                 | consent                 |
```

### 5. Query Cross-References

```bash
$ regula query --source testdata/gdpr.txt --template references | head -15
Template: references
Description: List all cross-references between articles

+-------------------------------------------+-------------------------------------------+
| from                                      | to                                        |
+-------------------------------------------+-------------------------------------------+
| GDPR:Art8                                 | GDPR:Art6                                 |
| GDPR:Art13                                | GDPR:Art6                                 |
| GDPR:Art14                                | GDPR:Art6                                 |
| GDPR:Art17                                | GDPR:Art6                                 |
| GDPR:Art20                                | GDPR:Art6                                 |
| GDPR:Art21                                | GDPR:Art6                                 |
```

### 6. Export Graph Summary

```bash
$ regula export --source testdata/gdpr.txt --format summary
Relationship Graph Summary
==========================

Total relationships: 3134

Most Referenced Articles:
  Article 6: 9 incoming references
  Article 9: 7 incoming references
  Article 43: 6 incoming references
  Article 40: 6 incoming references
  Article 65: 6 incoming references

Articles With Most Outgoing References:
  Article 70: 18 outgoing references
  Article 12: 11 outgoing references
  Article 58: 9 outgoing references
```

### 7. Validate for CI

```bash
$ regula validate --source testdata/gdpr.txt --format json | jq '{status, score: .overall_score, threshold}'
{
  "status": "PASS",
  "score": 0.946969696969697,
  "threshold": 0.8
}
```

### 8. Custom SPARQL Query

```bash
$ regula query --source testdata/gdpr.txt "SELECT ?article ?title WHERE {
  ?article rdf:type reg:Article .
  ?article reg:title ?title .
  FILTER(CONTAINS(?title, 'erasure'))
}"
+-------------------------------------------+----------------------------------------+
| article                                   | title                                  |
+-------------------------------------------+----------------------------------------+
| https://regula.dev/regulations/GDPR:Art17 | Right to erasure ('right to be forgotten') |
| https://regula.dev/regulations/GDPR:Art19 | Notification obligation regarding rectification or erasure of personal data or restriction of processing |
```

## Available Query Templates

| Template | Description |
|----------|-------------|
| `articles` | List all articles with titles |
| `definitions` | List all defined terms with definitions |
| `chapters` | List all chapters with titles |
| `references` | List cross-references between articles |
| `rights` | Find articles that grant rights |
| `obligations` | Find articles that impose obligations |
| `right-types` | List distinct right types found |
| `obligation-types` | List distinct obligation types found |
| `recitals` | List all recitals |
| `article-refs` | Find what articles reference a specific article |
| `search` | Search for articles by title content |
| `term-usage` | Find which articles use defined terms |
| `term-articles` | Find articles using a specific term |
| `article-terms` | Find all terms used in an article |
| `hierarchy` | Show document hierarchy |
| `most-referenced` | Find most referenced articles |
| `definition-links` | Show terms and defining articles |
| `bidirectional` | Show bidirectional reference relationships |

## Deferred: LLM-Assisted Extraction (#31)

Issue #31 is marked optional and remains open. The current rule-based extraction achieves:
- 100% reference resolution rate (exceeds target)
- 100% known rights detection
- 100% definition coverage

LLM assistance may be considered for:
- Extracting complex semantic relationships
- Handling non-standard provision structures
- Improving ambiguous reference resolution

## Acceptance Criteria Summary

| Criterion | Target | Achieved | Status |
|-----------|--------|----------|--------|
| Reference resolution rate | ≥85% | 100% | ✅ |
| Known GDPR rights detected | 6/6 | 6/6 | ✅ |
| Graph connectivity | Connected structure | 3134 edges | ✅ |
| Orphan provision identification | All identified | 21 identified | ✅ |
| Validation command | CI-ready JSON | Implemented | ✅ |
| Definition coverage | Track usage | 100% (349 usages) | ✅ |

## Next Steps

1. **M4: Query Interface** - Implement advanced query capabilities
2. **M5: Impact Analysis** - Add change impact prediction
3. **Consider #31** - Evaluate LLM-assisted extraction if edge cases emerge

## Files Added/Modified in M3

### New Packages
- `pkg/extract/resolver.go` - Reference resolution with confidence scoring
- `pkg/extract/semantic.go` - Rights/obligations extraction
- `pkg/extract/term_usage.go` - Term usage tracking
- `pkg/store/export.go` - Graph export (JSON, DOT)
- `pkg/validate/validate.go` - Comprehensive validation

### Modified Files
- `pkg/store/builder.go` - BuildComplete with all extractors
- `cmd/regula/main.go` - CLI commands and query templates
- `pkg/store/schema.go` - New relationship predicates
