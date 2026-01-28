# CCPA Analysis Example

This example demonstrates using regula to analyze the California Consumer Privacy Act (CCPA).

## Quick Start

```bash
# From the regula root directory
cd examples/ccpa-analysis

# Initialize a project
../../regula init ccpa-project

# Ingest the CCPA
../../regula ingest --source ../../testdata/ccpa.txt --stats
```

## Example Outputs

### 1. Ingestion Statistics

```bash
$ regula ingest --source testdata/ccpa.txt --stats
```

Output:
```
Ingesting regulation from: testdata/ccpa.txt
  1. Parsing document structure... done (6 chapters, 21 articles)
  2. Extracting defined terms... done (15 definitions)
  3. Identifying cross-references... done (19 references)
  4. Extracting rights/obligations... done (3 rights, 2 obligations)
  5. Resolving cross-references... done (0% resolved)
  6. Building knowledge graph... done (1026 triples)

Graph Statistics:
  Total triples:    1026
  Articles:         21
  Chapters:         6
  Sections:         0
  Recitals:         0
  Definitions:      15
  References:       19
  Rights:           3
  Obligations:      2
  Term usages:      74
```

### 2. Document Structure

The CCPA is organized into 6 chapters:

| Chapter | Title | Sections |
|---------|-------|----------|
| 1 | General Provisions | 1798.100, 1798.105, 1798.110 |
| 2 | Consumer Rights | 1798.115, 1798.120, 1798.125, 1798.130, 1798.135 |
| 3 | Business Obligations | 1798.140, 1798.145, 1798.150, 1798.155, 1798.160 |
| 4 | Enforcement | 1798.165, 1798.170, 1798.175, 1798.180 |
| 5 | Exemptions and Preemption | 1798.185, 1798.190 |
| 6 | Operative Dates | 1798.195, 1798.198 |

### 3. Querying Articles

```bash
$ regula query --source testdata/ccpa.txt --template articles
```

Output:
```
+--------------------------------------------------+--------------------------------------------------------------+
| article                                          | title                                                        |
+--------------------------------------------------+--------------------------------------------------------------+
| https://regula.dev/regulations/Regulation:Art100 | Title                                                        |
| https://regula.dev/regulations/Regulation:Art105 | Legislative Intent                                           |
| https://regula.dev/regulations/Regulation:Art110 | Definitions                                                  |
| https://regula.dev/regulations/Regulation:Art115 | Right to Know What Personal Information is Being Collected   |
| https://regula.dev/regulations/Regulation:Art120 | Right to Know What Personal Information is Sold or Disclosed |
| https://regula.dev/regulations/Regulation:Art125 | Right to Request Deletion of Personal Information            |
| https://regula.dev/regulations/Regulation:Art130 | Right to Opt-Out of Sale of Personal Information             |
| https://regula.dev/regulations/Regulation:Art135 | Right to Equal Service and Price                             |
| https://regula.dev/regulations/Regulation:Art140 | Notice at Collection                                         |
| https://regula.dev/regulations/Regulation:Art145 | Privacy Policy Requirements                                  |
| https://regula.dev/regulations/Regulation:Art150 | Data Minimization                                            |
| https://regula.dev/regulations/Regulation:Art155 | Data Security Requirements                                   |
| https://regula.dev/regulations/Regulation:Art160 | Data Breach Notification                                     |
| https://regula.dev/regulations/Regulation:Art165 | Civil Penalties for Violations                               |
| https://regula.dev/regulations/Regulation:Art170 | Private Right of Action                                      |
| https://regula.dev/regulations/Regulation:Art175 | Attorney General Authority                                   |
| https://regula.dev/regulations/Regulation:Art180 | Enforcement Actions                                          |
| https://regula.dev/regulations/Regulation:Art185 | Exemptions                                                   |
| https://regula.dev/regulations/Regulation:Art190 | Relationship to Other Laws                                   |
| https://regula.dev/regulations/Regulation:Art195 | Operative Date                                               |
| https://regula.dev/regulations/Regulation:Art198 | Amendments                                                   |
+--------------------------------------------------+--------------------------------------------------------------+
21 rows
```

### 4. Querying Definitions

```bash
$ regula query --source testdata/ccpa.txt --template definitions
```

The CCPA defines 15 key terms in Section 1798.110:

| # | Term | Description |
|---|------|-------------|
| 1 | Business | Entity collecting consumer personal information |
| 2 | Business purpose | Operational use of personal information |
| 3 | Commercial purposes | Advancing commercial/economic interests |
| 4 | Consumer | California resident |
| 5 | Deidentified | Information not linkable to consumer |
| 6 | Device | Internet-connected physical object |
| 7 | Homepage | Introductory page of website |
| 8 | Person | Individual or legal entity |
| 9 | Personal information | Information linked to consumer/household |
| 10 | Probabilistic identifier | Consumer identification by probability |
| 11 | Processing | Operations on personal information |
| 12 | Research | Scientific/systematic study |
| 13 | Service provider | Entity processing info for business |
| 14 | Third party | Non-business, non-service-provider |
| 15 | Verifiable consumer request | Authenticated consumer request |

### 5. Consumer Rights Under CCPA

The CCPA grants California consumers five key rights:

1. **Right to Know** (Section 1798.115)
   - Categories of personal information collected
   - Sources of information
   - Business purpose for collection
   - Third parties with whom info is shared

2. **Right to Know About Sales** (Section 1798.120)
   - Categories sold and to whom
   - Categories disclosed for business purpose

3. **Right to Delete** (Section 1798.125)
   - Request deletion of personal information
   - Business must delete and direct service providers to delete

4. **Right to Opt-Out** (Section 1798.130)
   - Direct business not to sell personal information
   - Special protections for minors under 16

5. **Right to Non-Discrimination** (Section 1798.135)
   - Cannot deny goods/services
   - Cannot charge different prices
   - Cannot provide different quality

### 6. Exporting the Graph

```bash
$ regula export --source testdata/ccpa.txt --format summary
```

Output:
```
Relationship Graph Summary
==========================

Total relationships: 1026

Relationship Types:
  reg:partOf              126
  reg:usesTerm            74
  reg:belongsTo           63
  reg:definedIn           15
  reg:defines             15
  reg:references          19
  ...

Most Referenced Articles:
  Section 1798.110: 5 incoming references
  Section 1798.130: 3 incoming references
  Section 1798.115: 2 incoming references
```

## Comparing CCPA to GDPR

| Aspect | CCPA | GDPR |
|--------|------|------|
| Chapters | 6 | 11 |
| Articles/Sections | 21 | 99 |
| Definitions | 15 | 26 |
| Graph Triples | 1,026 | 8,457 |
| Cross-references | 19 | 255 |
| Term usages | 74 | 349 |

## Key Differences in Structure

1. **Numbering**: CCPA uses decimal section numbers (1798.100), GDPR uses sequential articles (Article 1)

2. **Definitions Location**:
   - CCPA: Section 1798.110
   - GDPR: Article 4

3. **Rights Organization**:
   - CCPA: Chapter 2 (Consumer Rights)
   - GDPR: Chapter III (Rights of the data subject)

## Next Steps

- Run scenario matching: `regula match --source testdata/ccpa.txt --scenario data_breach`
- Export for visualization: `regula export --source testdata/ccpa.txt --format dot -o ccpa.dot`
- Compare with GDPR: Run both analyses and compare the graphs
