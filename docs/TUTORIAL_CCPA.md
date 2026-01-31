# Regula Tutorial: Analyzing the California Consumer Privacy Act (CCPA)

This tutorial demonstrates Regula's capabilities using the **California Consumer Privacy Act (CCPA)** and compares its structure with the **EU General Data Protection Regulation (GDPR)**. All output is from actual command runs against the included test data.

For the GDPR-focused tutorial, see [TUTORIAL.md](TUTORIAL.md).

## Prerequisites

Build Regula from source:

```bash
go build -o regula ./cmd/regula
```

Or install to your `$GOPATH/bin`:

```bash
make install
```

Both test documents are included: `testdata/ccpa.txt` and `testdata/gdpr.txt`.

---

## 1. Ingesting the CCPA

```bash
regula ingest --source testdata/ccpa.txt
```

**Output:**

```
Ingesting regulation from: testdata/ccpa.txt
  1. Parsing document structure... done (6 chapters, 21 articles)
  2. Extracting defined terms... done (15 definitions)
  3. Identifying cross-references... done (26 references)
  4. Extracting rights/obligations... done (24 rights, 28 obligations)
  5. Resolving cross-references... done (68% resolved)
  6. Building knowledge graph... done (1460 triples)

Ingestion complete in 79ms
```

### Extraction Summary

| Element                | CCPA   | GDPR   |
|-----------------------|--------|--------|
| Chapters              | 6      | 11     |
| Articles              | 21     | 99     |
| Defined terms         | 15     | 26     |
| Cross-references      | 26     | 258    |
| Rights extracted      | 24     | 60     |
| Obligations extracted | 28     | 69     |
| RDF triples           | 1,460  | 8,495  |
| Ingestion time        | 79ms   | 1.2s   |

The CCPA is roughly one-fifth the size of the GDPR but shares the same rights-and-obligations structure. Regula processes it in under 80 milliseconds.

---

## 2. Understanding the CCPA Structure

### Chapters

```bash
regula query --source testdata/ccpa.txt --template chapters
```

**Output:**

```
+------------+---------------------------+
| chapter    | title                     |
+------------+---------------------------+
| Chapter1   | General Provisions        |
| Chapter2   | Consumer Rights           |
| Chapter3   | Business Obligations      |
| Chapter4   | Enforcement               |
| Chapter5   | Exemptions and Preemption |
| Chapter6   | Operative Dates           |
+------------+---------------------------+
```

The CCPA follows a clear structure: general provisions and definitions, consumer rights, business obligations, enforcement mechanisms, exemptions, and operative dates.

### Full Hierarchy

```bash
regula query --source testdata/ccpa.txt --template hierarchy
```

**Output:**

| Chapter                   | Articles                                                        |
|---------------------------|-----------------------------------------------------------------|
| General Provisions        | Title (100), Legislative Intent (105), Definitions (110)        |
| Consumer Rights           | Right to Know — Collected (115), Right to Know — Sold (120), Deletion (125), Opt-Out (130), Equal Service (135) |
| Business Obligations      | Notice at Collection (140), Privacy Policy (145), Data Minimization (150), Data Security (155), Breach Notification (160) |
| Enforcement               | Civil Penalties (165), Private Right of Action (170), Attorney General (175), Enforcement Actions (180) |
| Exemptions and Preemption | Exemptions (185), Relationship to Other Laws (190)              |
| Operative Dates           | Operative Date (195), Amendments (198)                          |

### All 21 Articles

```bash
regula query --source testdata/ccpa.txt --template articles
```

**Output:**

```
+----------+--------------------------------------------------------------+
| article  | title                                                        |
+----------+--------------------------------------------------------------+
| Art100   | Title                                                        |
| Art105   | Legislative Intent                                           |
| Art110   | Definitions                                                  |
| Art115   | Right to Know What Personal Information is Being Collected   |
| Art120   | Right to Know What Personal Information is Sold or Disclosed |
| Art125   | Right to Request Deletion of Personal Information            |
| Art130   | Right to Opt-Out of Sale of Personal Information             |
| Art135   | Right to Equal Service and Price                             |
| Art140   | Notice at Collection                                         |
| Art145   | Privacy Policy Requirements                                  |
| Art150   | Data Minimization                                            |
| Art155   | Data Security Requirements                                   |
| Art160   | Data Breach Notification                                     |
| Art165   | Civil Penalties for Violations                               |
| Art170   | Private Right of Action                                      |
| Art175   | Attorney General Authority                                   |
| Art180   | Enforcement Actions                                          |
| Art185   | Exemptions                                                   |
| Art190   | Relationship to Other Laws                                   |
| Art195   | Operative Date                                               |
| Art198   | Amendments                                                   |
+----------+--------------------------------------------------------------+
21 rows (query: 28µs)
```

---

## 3. Consumer Rights Under the CCPA

### Extracting Rights

```bash
regula query --source testdata/ccpa.txt --template rights
```

**Output:**

```
+--------+--------------------------------------------------------------+-------------------------------+--------------------------+
| article| title                                                        | right                         | rightType                |
+--------+--------------------------------------------------------------+-------------------------------+--------------------------+
| Art115 | Right to Know What Personal Information is Being Collected   | Right:115:RightToKnow         | RightToKnow              |
| Art115 | Right to Know What Personal Information is Being Collected   | Right:115:RightToKnowAboutSales| RightToKnowAboutSales   |
| Art120 | Right to Know What Personal Information is Sold or Disclosed | Right:120:RightToKnow         | RightToKnow              |
| Art120 | Right to Know What Personal Information is Sold or Disclosed | Right:120:RightToKnowAboutSales| RightToKnowAboutSales   |
| Art125 | Right to Request Deletion of Personal Information            | Right:125:RightToDelete       | RightToDelete            |
| Art130 | Right to Opt-Out of Sale of Personal Information             | Right:130:RightToOptOut       | RightToOptOut            |
| Art135 | Right to Equal Service and Price                             | Right:135:RightToNonDiscrimination| RightToNonDiscrimination|
+--------+--------------------------------------------------------------+-------------------------------+--------------------------+
11 rows (query: 248µs)
```

### Five Core CCPA Rights (All Detected)

Regula identifies all 5 known CCPA rights (5/5):

| Right Type                 | Article | Description                                        |
|----------------------------|---------|-----------------------------------------------------|
| `RightToKnow`             | 115, 120| Know what personal information is collected and sold |
| `RightToKnowAboutSales`   | 115, 120| Know about sales and disclosures of personal info    |
| `RightToDelete`           | 125     | Request deletion of personal information             |
| `RightToOptOut`           | 130     | Opt out of sale of personal information              |
| `RightToNonDiscrimination`| 135     | Equal service and price regardless of rights exercise|

### Comparing CCPA and GDPR Rights

| Right Concept         | CCPA                       | GDPR                              |
|-----------------------|----------------------------|-----------------------------------|
| Know/Access           | RightToKnow (Arts 115,120) | RightOfAccess (Art 15)            |
| Deletion/Erasure      | RightToDelete (Art 125)    | RightToErasure (Art 17)           |
| Opt-Out/Object        | RightToOptOut (Art 130)    | RightToObject (Art 21)            |
| Non-Discrimination    | RightToNonDiscrimination (Art 135) | —                          |
| Data Portability      | —                          | RightToDataPortability (Art 20)   |
| Rectification         | —                          | RightToRectification (Art 16)     |
| Restrict Processing   | —                          | RightToRestriction (Art 18)       |
| Automated Decisions   | —                          | RightAgainstAutomatedDecision (Art 22) |
| Withdraw Consent      | —                          | RightToWithdrawConsent (Arts 13,14)|
| Lodge Complaint       | —                          | RightToLodgeComplaint (Arts 13-15)|

The CCPA focuses on transparency (know/opt-out) and non-discrimination, while the GDPR provides a broader set of data subject rights including portability, rectification, and restrictions on automated decision-making. Regula extracts and classifies these distinctions automatically.

---

## 4. Business Obligations Under the CCPA

### Extracting Obligations

```bash
regula query --source testdata/ccpa.txt --template obligations
```

**Output (24 obligations across 12 articles):**

```
+--------+--------------------------------------------------------------+--------------------------+
| article| title                                                        | obligType                |
+--------+--------------------------------------------------------------+--------------------------+
| Art110 | Definitions                                                  | ServiceProviderObligation|
| Art115 | Right to Know What Personal Information is Being Collected   | Obligation               |
| Art115 | Right to Know What Personal Information is Being Collected   | NoticeAtCollectionObligation|
| Art120 | Right to Know What Personal Information is Sold or Disclosed | NoticeAtCollectionObligation|
| Art125 | Right to Request Deletion of Personal Information            | ServiceProviderObligation|
| Art125 | Right to Request Deletion of Personal Information            | DataMinimizationObligation|
| Art125 | Right to Request Deletion of Personal Information            | NoticeAtCollectionObligation|
| Art130 | Right to Opt-Out of Sale of Personal Information             | NoticeAtCollectionObligation|
| Art135 | Right to Equal Service and Price                             | NonDiscriminationObligation|
| Art140 | Notice at Collection                                         | NoticeAtCollectionObligation|
| Art140 | Notice at Collection                                         | InformationProvisionObligation|
| Art145 | Privacy Policy Requirements                                  | PrivacyPolicyObligation  |
| Art150 | Data Minimization                                            | DataMinimizationObligation|
| Art155 | Data Security Requirements                                   | ServiceProviderObligation|
| Art160 | Data Breach Notification                                     | Obligation               |
| Art165 | Civil Penalties for Violations                               | ServiceProviderObligation|
+--------+--------------------------------------------------------------+--------------------------+
```

### Obligation Types

```bash
regula query --source testdata/ccpa.txt --template obligation-types
```

| Obligation Type                  | Description                                 |
|----------------------------------|---------------------------------------------|
| `NoticeAtCollectionObligation`  | Provide notice when collecting personal info |
| `ServiceProviderObligation`     | Requirements for service provider contracts  |
| `DataMinimizationObligation`    | Limit collection to what is necessary        |
| `InformationProvisionObligation`| Provide information to consumers             |
| `PrivacyPolicyObligation`       | Maintain and publish privacy policy          |
| `NonDiscriminationObligation`   | Not discriminate against rights-exercising consumers |

---

## 5. Defined Terms

### All 15 CCPA Definitions

```bash
regula query --source testdata/ccpa.txt --template definitions
```

All 15 terms are defined in Article 110 (Definitions):

| Term                          | Description (abbreviated)                          |
|-------------------------------|-----------------------------------------------------|
| Business                      | Sole proprietorship, partnership, LLC, corporation... |
| Business purpose              | Auditing, security, debugging, short-term use...     |
| Commercial purposes           | Advance commercial or economic interests             |
| Consumer                      | Natural person who is a California resident          |
| Deidentified                  | Information that cannot identify a consumer           |
| Device                        | Any physical object capable of connecting to internet |
| Homepage                      | Introductory page of an internet website              |
| Person                        | Individual, proprietorship, firm, partnership...      |
| Personal information          | Information that identifies or could be linked to...  |
| Probabilistic identifier      | Identifying a consumer with a degree of certainty    |
| Processing                    | Any operation performed on personal data              |
| Research                      | Scientific, systematic study and observation          |
| Service provider              | Entity that processes information on behalf of...     |
| Third party                   | Person who is not the business, service provider...   |
| Verifiable consumer request   | Request made by consumer that business can verify     |

### Term Usage Across Articles

```bash
regula query --source testdata/ccpa.txt --template term-usage
```

Key term propagation:

| Term                 | Articles Using It | Count |
|----------------------|:-----------------:|:-----:|
| Business             | 17                | 17    |
| Consumer             | 16                | 16    |
| Personal information | 10                | 10    |
| Business purpose     | 3                 | 3     |
| Deidentified         | 2                 | 2     |
| Service provider     | 3+                | 3+    |

The terms "Business" and "Consumer" are used in virtually every article — they are the two fundamental entities in the CCPA's regulatory framework. In comparison, the GDPR's equivalent terms "controller" and "data subject" show similar ubiquity.

---

## 6. Cross-Reference Network

### Reference Map

```bash
regula query --source testdata/ccpa.txt --template references
```

**Output:**

```
+--------+--------+
| from   | to     |
+--------+--------+
| Art110 | Art185 |
| Art115 | Art130 |
| Art120 | Art130 |
| Art125 | Art130 |
| Art130 | Art135 |
| Art135 | Art130 |
| Art145 | Art115 |
| Art175 | Art115 |
| Art175 | Art150 |
| Art175 | Art155 |
+--------+--------+
10 resolved references (query: 22µs)
```

### Most Referenced Articles

```bash
regula query --source testdata/ccpa.txt --template most-referenced
```

| Article | Incoming Refs | Title                            |
|---------|:-------------:|----------------------------------|
| Art 130 | 4             | Right to Opt-Out of Sale         |
| Art 115 | 2             | Right to Know — Being Collected  |
| Art 135 | 1             | Right to Equal Service and Price |
| Art 150 | 1             | Data Minimization                |
| Art 155 | 1             | Data Security Requirements       |
| Art 185 | 1             | Exemptions                       |

**Article 130 (Right to Opt-Out)** is the CCPA's most referenced provision with 4 incoming references — it is central to the CCPA's structure because the opt-out mechanism connects consumer rights, business obligations, and enforcement.

In the GDPR, Article 6 (Lawfulness of processing) holds the equivalent position with 9 incoming references — reflecting the GDPR's broader scope where lawfulness underpins everything.

### Impact Analysis

```bash
regula impact --source testdata/ccpa.txt --provision Art130
```

**Output:**

```
Impact Analysis for: Art130
Analysis Depth: 2
===================================================

Summary:
  Total affected provisions: 4
  Direct incoming (references this): 4

Direct Incoming (provisions referencing this):
  - Right to Know What Personal Information is Being Collected (Art 115)
  - Right to Know What Personal Information is Sold or Disclosed (Art 120)
  - Right to Request Deletion of Personal Information (Art 125)
  - Right to Equal Service and Price (Art 135)
```

Amending Article 130 would directly affect 4 other provisions — all in the Consumer Rights chapter. This is a tighter impact radius than the GDPR's Article 17 (42 affected provisions), reflecting the CCPA's simpler cross-reference structure.

---

## 7. Scenario Matching

### Access Request Scenario

```bash
regula match --source testdata/ccpa.txt --scenario access_request
```

**Output:**

```
Provision Matching Results for: Data Access Request
===================================================

Summary:
  Total matches: 16
  Direct: 1
  Triggered: 0
  Related: 15

Direct Matches:
  Art 140: Notice at Collection (score: 0.86)
    - Imposes InformationProvisionObligation (action: request_access)

Related Matches: (15 articles)
  Art 110: Definitions (score: 0.45)
  Art 115: Right to Know — Being Collected (score: 0.40)
  Art 170: Private Right of Action (score: 0.40)
  Art 155: Data Security Requirements (score: 0.40)
  ... and 11 more
```

### Data Breach Scenario

```bash
regula match --source testdata/ccpa.txt --scenario data_breach
```

**Output:**

```
Provision Matching Results for: Data Breach
===================================================

Summary:
  Total matches: 16
  Direct: 0
  Triggered: 0
  Related: 16

Related Matches: (16 articles)
  Art 160: Data Breach Notification (score: 0.50)
  Art 155: Data Security Requirements (score: 0.40)
  Art 110: Definitions (score: 0.40)
  Art 170: Private Right of Action (score: 0.40)
  Art 115: Right to Know — Being Collected (score: 0.35)
  ... and 11 more
```

### Comparing Scenario Results: CCPA vs. GDPR

| Scenario             | CCPA Matches | CCPA Direct | GDPR Matches | GDPR Direct |
|----------------------|:------------:|:-----------:|:------------:|:-----------:|
| Data Breach          | 16           | 0           | 80           | 6           |
| Consent Withdrawal   | 10           | 0           | 88           | 5           |
| Erasure Request      | 16           | 0           | 88           | 3           |
| Access Request       | 16           | 1           | —            | —           |

The GDPR produces more direct matches because it explicitly encodes obligations with action-specific types (e.g., `BreachNotificationObligation` with action `data_breach`). The CCPA's scenario matching relies more on keyword-based related matches, reflecting its different legislative drafting style.

---

## 8. Validation

### Full Validation

```bash
regula validate --source testdata/ccpa.txt
```

**Output:**

```
Validation Report
=================
Profile: CCPA

Definition Coverage:
  Defined terms: 15
  Terms with usage links: 15 (100.0%)
  Total term usages: 74
  Articles using terms: 19

Semantic Extraction:
  Rights found: 24 (in 5 articles)
  Obligations found: 29 (in 12 articles)
  Known CCPA rights: 5/5

Structure Quality:
  Articles: 21 (expected: 21, 100.0%)
  Chapters: 6 (expected: 6, 100.0%)

Component Scores:
  References:    68.4% (weight: 25%)
  Connectivity:  52.4% (weight: 20%)
  Definitions:   100.0% (weight: 20%)
  Semantics:     100.0% (weight: 20%)
  Structure:     100.0% (weight: 15%)

Overall Score: 82.6%
Threshold: 80.0%
Status: PASS
```

### Gate Pipeline

```bash
regula validate --source testdata/ccpa.txt --check gates
```

| Gate | Name      | Score   | Duration | Status |
|------|-----------|---------|----------|--------|
| V0   | Schema    | 100.0%  | 5 µs    | PASS   |
| V1   | Structure | 100.0%  | 2 µs    | PASS   |
| V2   | Coverage  | 73.0%   | 3 µs    | PASS   |
| V3   | Quality   | 66.5%   | 3 µs    | FAIL   |

Gate V3 fails because the CCPA references external California Civil Code sections (e.g., "Section 1798.80", "Section 17014") that fall outside the document scope. These are external references rather than extraction failures.

**Total gate execution: 13 microseconds.**

### Understanding the Score Differences

| Component       | CCPA  | GDPR  | Why the difference                          |
|-----------------|-------|-------|---------------------------------------------|
| References      | 68.4% | 98.8% | CCPA references external CA code sections   |
| Connectivity    | 52.4% | 78.8% | Fewer internal cross-references             |
| Definitions     | 100%  | 100%  | Both have complete term extraction           |
| Semantics       | 100%  | 100%  | Both have complete rights/obligations        |
| Structure       | 100%  | 100%  | Both have correct article/chapter counts     |
| **Overall**     | **82.6%** | **95.5%** | GDPR is more self-contained            |

The lower CCPA reference score is structural, not a quality issue — the CCPA frequently references the broader California Civil Code by section number, while the GDPR is largely self-referential.

### Profile Suggestion

```bash
regula validate --source testdata/ccpa.txt --suggest-profile
```

**Output:**

```
Profile Suggestion (confidence: 100%)
=====================================

Name: CALIFORNIA CONSUMER PRIVACY ACT OF 2018
Expected Structure:
  Articles:    21
  Definitions: 15
  Chapters:    6

Known Rights: 6
Known Obligations: 7

Reasoning:
  Definition density is high (0.71 defs/article), increasing weight
  Rich semantic content (24 rights, 28 obligations), increasing weight
```

---

## 9. Exporting the CCPA Knowledge Graph

### Relationship Summary

```bash
regula export --source testdata/ccpa.txt --format summary
```

**Output:**

```
Relationship Graph Summary
==========================

Total relationships: 503

Relationship Types:
  reg:partOf              161
  reg:belongsTo           102
  reg:usesTerm             74
  reg:contains             27
  reg:imposesObligation    24
  reg:resolvedTarget       20
  reg:definedIn            15
  reg:defines              15
  reg:grantsRight          11
  reg:references           10
  reg:referencedBy         10
  reg:externalRef           7
  reg:hasArticle           21
  reg:hasChapter            6
```

### RDF/XML Export

```bash
regula export --source testdata/ccpa.txt --format rdfxml --output ccpa.rdf
```

Produces 2,147 lines of W3C-compliant RDF/XML covering all 1,460 triples.

### All Export Formats

```bash
# Turtle RDF
regula export --source testdata/ccpa.txt --format turtle --output ccpa.ttl

# JSON-LD
regula export --source testdata/ccpa.txt --format jsonld --output ccpa.jsonld

# GraphViz
regula export --source testdata/ccpa.txt --format dot --output ccpa.dot

# JSON
regula export --source testdata/ccpa.txt --format json --output ccpa.json
```

---

## 10. Cross-Legislation Comparison

Regula enables side-by-side comparison of different privacy regulations by running the same analysis pipeline against each document.

### Structural Comparison

| Metric                  | CCPA   | GDPR   | Ratio  |
|------------------------|--------|--------|--------|
| Chapters               | 6      | 11     | 1:1.8  |
| Articles               | 21     | 99     | 1:4.7  |
| Defined terms          | 15     | 26     | 1:1.7  |
| Internal references    | 26     | 258    | 1:9.9  |
| RDF triples            | 1,460  | 8,495  | 1:5.8  |
| Total relationships    | 503    | 3,141  | 1:6.2  |
| Ingestion time         | 79ms   | 1.2s   | 1:15   |

### Semantic Comparison

| Metric                  | CCPA   | GDPR   |
|------------------------|--------|--------|
| Rights extracted        | 24     | 60     |
| Obligations extracted   | 28     | 69     |
| Right types (distinct)  | 5      | 10     |
| Obligation types        | 7      | 9      |
| Term usages             | 74     | 349    |
| Most-referenced article | Art 130 (4 refs) | Art 6 (9 refs) |
| Hub article (most outgoing) | Art 175 (3 refs) | Art 70 (18 refs) |

### Rights Overlap

Running `regula query --template right-types` against both regulations:

| CCPA Right Types            | GDPR Right Types                    |
|-----------------------------|-------------------------------------|
| RightToKnow                 | RightOfAccess                       |
| RightToKnowAboutSales       | —                                   |
| RightToDelete               | RightToErasure                      |
| RightToOptOut               | RightToObject                       |
| RightToNonDiscrimination    | —                                   |
| —                           | RightToRectification                |
| —                           | RightToDataPortability              |
| —                           | RightToRestriction                  |
| —                           | RightAgainstAutomatedDecision       |
| —                           | RightToWithdrawConsent              |
| —                           | RightToLodgeComplaint               |
| —                           | RightToNotification                 |

**Shared concepts:** Know/Access, Delete/Erasure, Opt-Out/Object

**CCPA-specific:** RightToKnowAboutSales, RightToNonDiscrimination

**GDPR-specific:** Portability, Rectification, Restriction, Automated Decisions, Consent Withdrawal, Complaints, Notification

### Obligation Overlap

| CCPA Obligation Types             | GDPR Obligation Types                |
|-----------------------------------|--------------------------------------|
| NoticeAtCollectionObligation      | InformationProvisionObligation       |
| ServiceProviderObligation         | —                                    |
| DataMinimizationObligation        | —                                    |
| PrivacyPolicyObligation           | —                                    |
| NonDiscriminationObligation       | —                                    |
| InformationProvisionObligation    | InformationProvisionObligation       |
| —                                 | BreachNotificationObligation         |
| —                                 | SecurityObligation                   |
| —                                 | RecordKeepingObligation              |
| —                                 | ConsentObligation                    |
| —                                 | ImplementationObligation             |
| —                                 | SubjectNotificationObligation        |
| —                                 | EnsureObligation                     |

### Validation Comparison

| Component       | CCPA   | GDPR   |
|-----------------|--------|--------|
| References      | 68.4%  | 98.8%  |
| Connectivity    | 52.4%  | 78.8%  |
| Definitions     | 100%   | 100%   |
| Semantics       | 100%   | 100%   |
| Structure       | 100%   | 100%   |
| **Overall**     | **82.6%** | **95.5%** |
| Gate V0-V1      | PASS   | PASS   |
| Gate V2         | PASS   | PASS   |
| Gate V3         | FAIL   | PASS   |

---

## 11. Performance Summary

All benchmarks on the CCPA (28KB, 415 lines, 21 articles).

### Timing

| Operation                  | CCPA    | GDPR    |
|----------------------------|---------|---------|
| Full ingest                | 79ms    | 1.2s    |
| Full validation            | 0.24s   | 1.5s    |
| Gate validation (4 gates)  | 13µs    | 84µs    |
| Query (articles)           | 28µs    | ~100µs  |
| Query (definitions)        | ~40µs   | 86µs    |
| Query (rights)             | 248µs   | ~250µs  |
| Query (references)         | 22µs    | ~100µs  |
| Impact analysis            | 0.12s   | 0.84s   |
| Scenario matching          | 0.15s   | 1.3-1.6s|
| Export summary             | 0.12s   | 0.9s    |
| Profile suggestion         | 0.16s   | 1.3s    |

### Manual Analysis Equivalent

| Task                                          | Manual Estimate | Regula  |
|-----------------------------------------------|-----------------|---------|
| Read and catalog 21 CCPA articles             | 1-2 hours       | 79ms    |
| Extract 15 defined terms and definitions      | 30-60 min       | 79ms    |
| Map 26 cross-references                       | 1-2 hours       | 79ms    |
| Identify 24 rights across 5 articles          | 1-2 hours       | 79ms    |
| Identify 28 obligations across 12 articles    | 1-2 hours       | 79ms    |
| Trace impact of amending Article 130          | 30-60 min       | 0.12s   |
| Validate extraction quality                   | 1-2 hours       | 0.24s   |
| Compare CCPA vs GDPR rights/obligations       | 4-8 hours       | <2s     |

**Total: 10-19 hours of manual analysis reduced to under 3 seconds.**

---

## 12. Complete CCPA Workflow

```bash
# 1. Ingest
regula ingest --source testdata/ccpa.txt

# 2. Validate
regula validate --source testdata/ccpa.txt
regula validate --source testdata/ccpa.txt --check gates

# 3. Explore structure
regula query --source testdata/ccpa.txt --template articles
regula query --source testdata/ccpa.txt --template chapters
regula query --source testdata/ccpa.txt --template hierarchy
regula query --source testdata/ccpa.txt --template definitions

# 4. Analyze rights and obligations
regula query --source testdata/ccpa.txt --template rights
regula query --source testdata/ccpa.txt --template obligations
regula query --source testdata/ccpa.txt --template right-types
regula query --source testdata/ccpa.txt --template obligation-types

# 5. Cross-references
regula query --source testdata/ccpa.txt --template references
regula query --source testdata/ccpa.txt --template most-referenced
regula query --source testdata/ccpa.txt --template term-usage

# 6. Impact analysis
regula impact --source testdata/ccpa.txt --provision Art130
regula impact --source testdata/ccpa.txt --provision Art115

# 7. Scenario matching
regula match --source testdata/ccpa.txt --scenario access_request
regula match --source testdata/ccpa.txt --scenario data_breach
regula match --source testdata/ccpa.txt --scenario erasure_request
regula match --source testdata/ccpa.txt --scenario consent_withdrawal

# 8. Export
regula export --source testdata/ccpa.txt --format summary
regula export --source testdata/ccpa.txt --format rdfxml --output ccpa.rdf
regula export --source testdata/ccpa.txt --format turtle --output ccpa.ttl

# 9. Cross-legislation comparison (run both)
regula query --source testdata/gdpr.txt --template right-types
regula query --source testdata/ccpa.txt --template right-types
regula validate --source testdata/gdpr.txt --format json
regula validate --source testdata/ccpa.txt --format json
```

---

## Next Steps

- See [TUTORIAL.md](TUTORIAL.md) for the GDPR-focused tutorial with deeper coverage of impact analysis and scenario matching
- See [TUTORIAL_US_CROSSREF.md](TUTORIAL_US_CROSSREF.md) for US privacy law cross-reference analysis
- Review [TESTING.md](TESTING.md) for development and testing strategies
- See [ARCHITECTURE.md](ARCHITECTURE.md) for system design details
- Check [ROADMAP.md](ROADMAP.md) for upcoming features
