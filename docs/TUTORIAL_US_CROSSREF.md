# Regula Tutorial: US Privacy Law Cross-Reference Analysis

This tutorial demonstrates Regula's cross-legislation analysis capabilities using **US state and federal privacy laws**. It focuses on the `compare` and `refs` commands to reveal how state privacy statutes share definitions, rights, and obligations — and how they reference federal legislation.

For single-document tutorials, see [TUTORIAL.md](TUTORIAL.md) (GDPR) and [TUTORIAL_CCPA.md](TUTORIAL_CCPA.md) (CCPA).

## Prerequisites

Build Regula from source:

```bash
go build -o regula ./cmd/regula
```

Or install to your `$GOPATH/bin`:

```bash
make install
```

This tutorial uses the following US privacy law test data files included in `testdata/`:

| File | Law | Year | Level |
|------|-----|------|-------|
| `ccpa.txt` | California Consumer Privacy Act (CCPA) | 2018 | State |
| `vcdpa.txt` | Virginia Consumer Data Protection Act (VCDPA) | 2021 | State |
| `tdpsa.txt` | Texas Data Privacy and Security Act (TDPSA) | 2023 | State |
| `us-coppa.txt` | Children's Online Privacy Protection Act (COPPA) | 1998 | Federal |

---

## 1. The US Privacy Law Landscape

The United States lacks a single comprehensive federal privacy law comparable to the EU's GDPR. Instead, privacy regulation has developed through a patchwork of sector-specific federal statutes and an expanding wave of state-level comprehensive privacy laws.

**Federal baseline:**
- **COPPA** (1998) — Children's online privacy, the earliest federal statute focused on consumer data collection
- **HIPAA** (1996) — Health information privacy (42 U.S.C. § 1320d)
- **GLBA** (1999) — Financial information privacy (15 U.S.C. § 6801)
- **FERPA** (1974) — Education records privacy (20 U.S.C. § 1232g)
- **DPPA** (1994) — Driver's privacy (18 U.S.C. § 2721)

**State wave:**
1. **California** (CCPA, 2018) — First comprehensive state consumer privacy law
2. **Virginia** (VCDPA, 2021) — Adopted controller/processor terminology closer to GDPR
3. **Colorado** (CPA, 2021), **Connecticut** (CTDPA, 2022), **Utah** (UCPA, 2022)
4. **Iowa** (ICDPA, 2023), **Texas** (TDPSA, 2023)

Regula can quantify how these laws relate to each other — shared terminology, overlapping rights, and common references to federal statutes.

---

## 2. Ingesting US Privacy Laws

Start by ingesting the core US privacy documents:

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

Ingestion complete in 81ms
```

```bash
regula ingest --source testdata/vcdpa.txt
```

**Output:**

```
Ingesting regulation from: testdata/vcdpa.txt
  1. Parsing document structure... done (7 chapters, 11 articles)
  2. Extracting defined terms... done (20 definitions)
  3. Identifying cross-references... done (52 references)
  4. Extracting rights/obligations... done (12 rights, 7 obligations)
  5. Resolving cross-references... done (0% resolved)
  6. Building knowledge graph... done (1506 triples)

Ingestion complete in 113ms
```

```bash
regula ingest --source testdata/tdpsa.txt
```

**Output:**

```
Ingesting regulation from: testdata/tdpsa.txt
  1. Parsing document structure... done (1 chapters, 7 articles)
  2. Extracting defined terms... done (18 definitions)
  3. Identifying cross-references... done (4 references)
  4. Extracting rights/obligations... done (5 rights, 5 obligations)
  5. Resolving cross-references... done (0% resolved)
  6. Building knowledge graph... done (619 triples)

Ingestion complete in 21ms
```

```bash
regula ingest --source testdata/us-coppa.txt
```

**Output:**

```
Ingesting regulation from: testdata/us-coppa.txt
  1. Parsing document structure... done (3 chapters, 10 articles)
  2. Extracting defined terms... done (7 definitions)
  3. Identifying cross-references... done (4 references)
  4. Extracting rights/obligations... done (0 rights, 1 obligations)
  5. Resolving cross-references... done (0% resolved)
  6. Building knowledge graph... done (409 triples)

Ingestion complete in 30ms
```

### Structural Comparison

| Element           | CCPA | VCDPA | TDPSA | COPPA |
|-------------------|------|-------|-------|-------|
| Chapters          | 6    | 7     | 1     | 3     |
| Articles          | 21   | 11    | 7     | 10    |
| Defined terms     | 15   | 20    | 18    | 7     |
| Cross-references  | 26   | 52    | 4     | 4     |
| Rights extracted  | 24   | 12    | 5     | 0     |
| Obligations       | 28   | 7     | 5     | 1     |
| RDF triples       | 1,460| 1,506 | 619   | 409   |
| Ingestion time    | 81ms | 113ms | 21ms  | 30ms  |

Virginia's VCDPA produces more triples than the CCPA despite having fewer articles, reflecting its denser cross-reference structure (52 references vs CCPA's 26). Texas follows a more concise structure with 7 articles but 18 defined terms — the most of any document here.

---

## 3. Pairwise Comparison: CCPA vs Virginia

The `compare` command reveals structural and semantic overlap between two laws:

```bash
regula compare --sources testdata/ccpa.txt,testdata/vcdpa.txt --format table
```

**Output:**

```
Comparison: CALIFORNIA CONSUMER PRIVACY ACT OF 2018 vs VIRGINIA CONSUMER DATA PROTECTION ACT
===================================================

+------------------+----------+----------+
| Metric           | CCPA     | VCDPA    |
+------------------+----------+----------+
| Articles         |       21 |       11 |
| Definitions      |       15 |       20 |
| References       |       10 |        0 |
| Rights           |       11 |       10 |
| Obligations      |       24 |        7 |
| External Refs    |        7 |       18 |
| Total Triples    |     1460 |     1506 |
+------------------+----------+----------+

Shared definitions:     2
Shared rights:          5
Shared obligations:     2
Shared external refs:   3

Shared Definitions:
  - consumer
  - third party

Shared Rights:
  - right
  - righttodelete
  - righttoknow
  - righttonondiscrimination
  - righttooptout

Shared Obligations:
  - dataminimizationobligation
  - obligation

Shared External Reference Targets:
  - 15 u.s.c. § 1681
  - 18 u.s.c. § 2721
  - 20 u.s.c. § 1232g
```

### Analysis

**Shared definitions** — Both laws define "consumer" and "third party," though with jurisdiction-specific scope (California residents vs Virginia residents). Virginia introduces GDPR-influenced terms not in the CCPA: "controller," "processor," "consent," "biometric data," and "sensitive data."

**Shared rights** — Five rights appear in both laws. The right to delete, right to know, right to opt out, and right to non-discrimination form a common core across US state privacy legislation.

**Shared external references** — Both laws reference the same three federal statutes:
- **15 U.S.C. § 1681** — Fair Credit Reporting Act (FCRA)
- **18 U.S.C. § 2721** — Driver's Privacy Protection Act (DPPA)
- **20 U.S.C. § 1232g** — Family Educational Rights and Privacy Act (FERPA)

These shared federal references reflect the common exemption pattern: state privacy laws typically exempt data already regulated by sector-specific federal statutes.

---

## 4. Three-Way Analysis: California, Virginia, and Texas

Adding Texas to the comparison reveals how a later-generation state law relates to both pioneers:

```bash
regula compare --sources testdata/ccpa.txt,testdata/vcdpa.txt,testdata/tdpsa.txt --format table
```

**Output (structural metrics):**

```
+------------------+----------+----------+----------+
| Metric           |     CCPA |    TDPSA |    VCDPA |
+------------------+----------+----------+----------+
| Articles         |       21 |        7 |       11 |
| Definitions      |       15 |       18 |       20 |
| References       |       10 |        0 |        0 |
| Rights           |       11 |        5 |       10 |
| Obligations      |       24 |        5 |        7 |
| External Refs    |        7 |        1 |       18 |
| Total Triples    |     1460 |      619 |     1506 |
+------------------+----------+----------+----------+
```

**Shared concepts (25 total):**

```
Shared Definitions: 16
---
  affiliate — TDPSA, VCDPA
  biometric data — TDPSA, VCDPA
  child — TDPSA, VCDPA
  consent — TDPSA, VCDPA
  consumer — CCPA, TDPSA, VCDPA
  controller — TDPSA, VCDPA
  dark pattern — TDPSA, VCDPA
  de-identified data — TDPSA, VCDPA
  personal data — TDPSA, VCDPA
  processing — CCPA, TDPSA
  processor — TDPSA, VCDPA
  profiling — TDPSA, VCDPA
  pseudonymous data — TDPSA, VCDPA
  sensitive data — TDPSA, VCDPA
  targeted advertising — TDPSA, VCDPA
  third party — CCPA, TDPSA, VCDPA

Shared Rights: 7
---
  right — CCPA, TDPSA, VCDPA
  righttocorrect — TDPSA, VCDPA
  righttodataportability — TDPSA, VCDPA
  righttodelete — CCPA, TDPSA, VCDPA
  righttoknow — CCPA, TDPSA, VCDPA
  righttonondiscrimination — CCPA, VCDPA
  righttooptout — CCPA, VCDPA
```

### The Virginia Model

The three-way comparison makes a structural pattern visible: **Texas adopted Virginia's terminology almost wholesale.** Of 16 shared definitions, 14 are shared between TDPSA and VCDPA (affiliate, biometric data, child, consent, controller, dark pattern, de-identified data, personal data, processor, profiling, pseudonymous data, sensitive data, targeted advertising). Only "consumer," "processing," and "third party" bridge all three laws.

This reflects the well-documented "Virginia model" phenomenon: after CCPA established the concept of comprehensive state privacy legislation, Virginia created a more GDPR-aligned framework that became the template for subsequent state laws.

**Universal definitions** (all 3 states): consumer, third party

**Virginia-model definitions** (TDPSA + VCDPA, not CCPA): controller, processor, consent, biometric data, sensitive data, profiling, dark pattern, de-identified data, pseudonymous data, targeted advertising, affiliate, child

**California-specific terms** (not shared): personal information (CCPA uses "personal information" rather than "personal data"), business, service provider

---

## 5. Full US Privacy Landscape: Four-Document Analysis

Adding federal COPPA to the analysis shows the state-federal connection:

```bash
regula compare --sources testdata/ccpa.txt,testdata/vcdpa.txt,testdata/tdpsa.txt,testdata/us-coppa.txt --format table
```

**Output (document summary):**

```
+----------------------+----------+------+-------+------+------+----------+
| Document             | Articles | Defs | Refs  | Rts  | Obls | ExtRefs  |
+----------------------+----------+------+-------+------+------+----------+
| CALIFORNIA CONSUM... |       21 |   15 |    10 |   11 |   24 |        7 |
| TEXAS DATA PRIVAC... |        7 |   18 |     0 |    5 |    5 |        1 |
| CHILDREN'S ONLINE... |       10 |    7 |     0 |    0 |    2 |        1 |
| VIRGINIA CONSUMER... |       11 |   20 |     0 |   10 |    7 |       18 |
+----------------------+----------+------+-------+------+------+----------+
```

**Cross-law shared concepts (28 total):**

```
Shared Definitions: 18
---
  affiliate — TDPSA, VCDPA
  biometric data — TDPSA, VCDPA
  child — TDPSA, US-COPPA, VCDPA
  consent — TDPSA, VCDPA
  consumer — CCPA, TDPSA, VCDPA
  controller — TDPSA, VCDPA
  dark pattern — TDPSA, VCDPA
  de-identified data — TDPSA, VCDPA
  person — CCPA, US-COPPA
  personal data — TDPSA, VCDPA
  personal information — CCPA, US-COPPA
  processing — CCPA, TDPSA
  processor — TDPSA, VCDPA
  profiling — TDPSA, VCDPA
  pseudonymous data — TDPSA, VCDPA
  sensitive data — TDPSA, VCDPA
  targeted advertising — TDPSA, VCDPA
  third party — CCPA, TDPSA, VCDPA

Shared Obligations: 3
---
  dataminimizationobligation — CCPA, VCDPA
  noticeatcollectionobligation — CCPA, US-COPPA
  obligation — CCPA, TDPSA, US-COPPA, VCDPA
```

### Federal-State Connections

Two definitions bridge state and federal law:
- **"child"** — defined in COPPA (under 13), TDPSA, and VCDPA. State laws adopted COPPA's age threshold.
- **"personal information"** — shared between CCPA and COPPA, reflecting their common consumer-focused terminology.

The obligation overlap is significant: **"noticeatcollectionobligation"** appears in both CCPA and federal COPPA, showing that COPPA's notice requirement (operators must notify parents before collecting children's data) directly influenced California's broader notice-at-collection mandate.

**External Reference Targets (17 unique, 27 total):**

```
  42 u.s.c. § 1320d (5 refs from VCDPA)       — HIPAA
  15 u.s.c. § 6501 (3 refs from VCDPA)         — COPPA
  15 u.s.c. § 1681 (2 refs from CCPA, VCDPA)   — FCRA
  18 u.s.c. § 2721 (2 refs from CCPA, VCDPA)   — DPPA
  20 u.s.c. § 1232g (2 refs from CCPA, VCDPA)  — FERPA
  42 u.s.c. § 290d (2 refs from VCDPA)          — Substance Abuse Records
```

HIPAA (42 U.S.C. § 1320d) is the most-referenced federal statute with 5 references from Virginia alone, reflecting the extensive health data exemptions built into state privacy laws.

---

## 6. External Reference Analysis

The `refs` command provides detailed external reference analysis for individual documents.

### Virginia's Federal References

```bash
regula refs --source testdata/vcdpa.txt --external-only
```

**Output:**

```
External Reference Report: VIRGINIA CONSUMER DATA PROTECTION ACT
===================================================

Total external references: 18
Unique external targets:   11

External Documents Referenced:
+------+---------------------------------------------------+
| Refs | Target Document                                   |
+------+---------------------------------------------------+
|    5 | 42 u.s.c. § 1320d                                 |
|    3 | 15 u.s.c. § 6501                                  |
|    2 | 42 u.s.c. § 290d                                  |
|    1 | 12 u.s.c. § 2001                                  |
|    1 | 15 u.s.c. § 1681                                  |
|    1 | 15 u.s.c. § 6801                                  |
|    1 | 18 u.s.c. § 2721                                  |
|    1 | 20 u.s.c. § 1232g                                 |
|    1 | 42 u.s.c. § 11101                                 |
|    1 | 42 u.s.c. § 299b                                  |
|    1 | 45 c.f.r. part 46                                 |
+------+---------------------------------------------------+
```

Virginia references 11 distinct federal statutes — the densest external reference network of any state law in this analysis. The references cluster into three categories:

**Health data** (8 references): HIPAA (42 U.S.C. § 1320d, 5 refs), substance abuse records (42 U.S.C. § 290d, 2 refs), healthcare quality (42 U.S.C. § 299b, 1 ref)

**Children and education** (4 references): COPPA (15 U.S.C. § 6501, 3 refs), FERPA (20 U.S.C. § 1232g, 1 ref)

**Financial and consumer** (3 references): FCRA (15 U.S.C. § 1681, 1 ref), GLBA (15 U.S.C. § 6801, 1 ref), DPPA (18 U.S.C. § 2721, 1 ref)

### California's Federal References

```bash
regula refs --source testdata/ccpa.txt --external-only
```

**Output:**

```
External Reference Report: CALIFORNIA CONSUMER PRIVACY ACT OF 2018
===================================================

Total external references: 7
Unique external targets:   7

External Documents Referenced:
+------+---------------------------------------------------+
| Refs | Target Document                                   |
+------+---------------------------------------------------+
|    1 | 15 u.s.c. § 1681                                  |
|    1 | 18 u.s.c. § 2721                                  |
|    1 | 20 u.s.c. § 1232g                                 |
|    1 | 34 c.f.r. part 99                                 |
|    1 | cal. title 18 § 17014                             |
|    1 | pub. l. 104-191                                   |
|    1 | pub. l. 106-102                                   |
+------+---------------------------------------------------+
```

California references 7 federal statutes with one reference each — a flatter distribution than Virginia's. The CCPA references HIPAA and GLBA using Public Law numbers (Pub. L. 104-191 and Pub. L. 106-102) rather than U.S. Code sections, reflecting the different citation conventions used by the California Legislature.

### Full Reference Analysis

The `refs` command without `--external-only` shows both internal and external references:

```bash
regula refs --source testdata/ccpa.txt
```

**Output:**

```
Reference Analysis: CALIFORNIA CONSUMER PRIVACY ACT OF 2018
===================================================

Total relationships: 503
Internal references: 10
External references: 7

External Reference Targets (7 unique):
  cal. title 18 § 17014                         1
  20 u.s.c. § 1232g                             1
  18 u.s.c. § 2721                              1
  15 u.s.c. § 1681                              1
  pub. l. 106-102                               1
  pub. l. 104-191                               1
  34 c.f.r. part 99                             1

Most Referenced Articles (internal):
  Article 130: 4 incoming references
  Article 115: 2 incoming references
  Article 155: 1 incoming references
  Article 150: 1 incoming references

Articles With Most Outgoing References:
  Article 175: 3 outgoing references
  Article 125: 1 outgoing references
  Article 120: 1 outgoing references
```

The internal reference analysis reveals Article 130 (dealing with consumer rights) as the CCPA's structural hub with 4 incoming references, while Article 175 (enforcement provisions) has the most outgoing references at 3.

### External Reference Comparison Across States

| Federal Statute | CCPA | VCDPA | TDPSA | Purpose |
|----------------|------|-------|-------|---------|
| HIPAA (42 U.S.C. § 1320d / Pub. L. 104-191) | 1 | 5 | — | Health data exemption |
| COPPA (15 U.S.C. § 6501) | — | 3 | — | Children's data exemption |
| GLBA (15 U.S.C. § 6801 / Pub. L. 106-102) | 1 | 1 | — | Financial data exemption |
| FCRA (15 U.S.C. § 1681) | 1 | 1 | — | Credit reporting exemption |
| DPPA (18 U.S.C. § 2721) | 1 | 1 | — | Driver's records exemption |
| FERPA (20 U.S.C. § 1232g) | 1 | 1 | — | Education records exemption |
| SBA (13 C.F.R. Part 121) | — | — | 1 | Small business threshold |

Virginia references the most federal statutes (11 unique targets, 18 total references). California references 7 with a flatter distribution. Texas has minimal external references (1), using the SBA definition to establish its small business exemption threshold.

---

## 7. Rights Across State Privacy Laws

The `query` command with the `right-types` template reveals which rights each law grants:

```bash
regula query --source testdata/ccpa.txt --template right-types
```

```
+--------------------------+
| rightType                |
+--------------------------+
| Right                    |
| RightToNonDiscrimination |
| RightToKnow              |
| RightToKnowAboutSales    |
| RightToDelete            |
| RightToOptOut            |
+--------------------------+
6 rows
```

```bash
regula query --source testdata/vcdpa.txt --template right-types
```

```
+--------------------------+
| rightType                |
+--------------------------+
| RightToOptOut            |
| RightToCorrect           |
| Right                    |
| RightToNonDiscrimination |
| RightOfAccess            |
| RightToDataPortability   |
| RightToKnow              |
| RightToDelete            |
+--------------------------+
8 rows
```

```bash
regula query --source testdata/tdpsa.txt --template right-types
```

```
+------------------------+
| rightType              |
+------------------------+
| RightToCorrect         |
| Right                  |
| RightToDataPortability |
| RightToKnow            |
| RightToDelete          |
+------------------------+
5 rows
```

### Rights Comparison Table

| Right | CCPA | VCDPA | TDPSA |
|-------|------|-------|-------|
| RightToKnow | Yes | Yes | Yes |
| RightToDelete | Yes | Yes | Yes |
| RightToOptOut | Yes | Yes | — |
| RightToNonDiscrimination | Yes | Yes | — |
| RightToCorrect | — | Yes | Yes |
| RightToDataPortability | — | Yes | Yes |
| RightOfAccess | — | Yes | — |
| RightToKnowAboutSales | Yes | — | — |

**Universal rights** (all 3 states): RightToKnow, RightToDelete

**California-specific**: RightToKnowAboutSales (reflecting CCPA's focus on the sale of personal information)

**Virginia-model rights** (VCDPA + TDPSA): RightToCorrect, RightToDataPortability — these GDPR-influenced rights were adopted by Virginia and carried forward to Texas

The data portability and correction rights are absent from the CCPA (as originally enacted), while the CCPA's sale-transparency right is absent from Virginia-model laws that use "opt-out of targeted advertising" instead.

---

## 8. Obligations Across States

```bash
regula query --source testdata/ccpa.txt --template obligation-types
```

```
+--------------------------------+
| obligType                      |
+--------------------------------+
| Obligation                     |
| ServiceProviderObligation      |
| NonDiscriminationObligation    |
| InformationProvisionObligation |
| NoticeAtCollectionObligation   |
| PrivacyPolicyObligation        |
| DataMinimizationObligation     |
+--------------------------------+
7 rows
```

```bash
regula query --source testdata/vcdpa.txt --template obligation-types
```

```
+----------------------------+
| obligType                  |
+----------------------------+
| DataMinimizationObligation |
| Obligation                 |
| ImplementationObligation   |
| ResponseObligation         |
+----------------------------+
4 rows
```

### Obligations Comparison

| Obligation | CCPA | VCDPA | TDPSA |
|-----------|------|-------|-------|
| DataMinimizationObligation | Yes | Yes | — |
| NoticeAtCollectionObligation | Yes | — | — |
| ServiceProviderObligation | Yes | — | — |
| NonDiscriminationObligation | Yes | — | — |
| InformationProvisionObligation | Yes | — | — |
| PrivacyPolicyObligation | Yes | — | — |
| ImplementationObligation | — | Yes | — |
| ResponseObligation | — | Yes | — |

California imposes the most granular obligation types (7 distinct categories). Virginia uses broader obligation categories with fewer, more general types. This reflects a philosophical difference: the CCPA prescribes specific obligations (service provider requirements, privacy policy mandates), while the VCDPA establishes broader duties (implementation obligations, response obligations) that are more adaptable.

---

## 9. Visualizing the Cross-Reference Network

The `compare` command can export a Graphviz DOT graph for visual analysis:

```bash
regula compare --sources testdata/ccpa.txt,testdata/vcdpa.txt,testdata/tdpsa.txt,testdata/us-coppa.txt --format dot --output us-privacy-network.dot
```

**Output:**

```
DOT graph exported to: us-privacy-network.dot

To visualize with Graphviz:
  dot -Tpng us-privacy-network.dot -o us-privacy-network.png
```

The DOT output creates:
- **Cluster subgraphs** for each document, showing article count, definitions, rights, and obligations
- **Green bidirectional edges** for shared definitions (e.g., "consumer" connecting CCPA, VCDPA, and TDPSA)
- **Orange bidirectional edges** for shared rights (e.g., "righttodelete" connecting all three state laws)
- **Brown bidirectional edges** for shared obligations
- **Red dashed edges** to hexagonal nodes for external references to federal statutes

The first lines of the generated DOT file:

```dot
digraph CrossLegislationAnalysis {
  rankdir=LR;
  compound=true;
  fontname="Helvetica";
  node [fontname="Helvetica" fontsize=10];
  edge [fontname="Helvetica" fontsize=8];

  subgraph cluster_0 {
    label="CALIFORNIA CONSUMER PRIVACY ACT OF 2018";
    style=filled;
    color=lightgrey;
    node [style=filled];

    "CCPA" [label="CALIFORNIA CONSUMER PRIVACY ACT OF 2018\nArticles: 21\nDefs: 15
      \nRights: 11\nObls: 24" shape=box fillcolor=lightyellow];
    "CCPA_defs" [label="15 definitions" shape=ellipse fillcolor=lightblue];
    "CCPA" -> "CCPA_defs" [label="defines" color=blue];
    ...
  }
  ...
}
```

To render the graph:

```bash
# PNG output
dot -Tpng us-privacy-network.dot -o us-privacy-network.png

# SVG for web embedding
dot -Tsvg us-privacy-network.dot -o us-privacy-network.svg

# PDF for print
dot -Tpdf us-privacy-network.dot -o us-privacy-network.pdf
```

### JSON Export

For programmatic analysis, export the comparison data as JSON:

```bash
regula compare --sources testdata/ccpa.txt,testdata/vcdpa.txt --format json --output ccpa-vcdpa-comparison.json
```

The JSON output includes all comparison metrics, shared concepts, and external reference clusters in a machine-readable format suitable for integration with other analysis tools or dashboards.

---

## 10. Federal COPPA as a Reference Hub

Federal COPPA occupies a unique position as both a standalone regulation and a reference target for state laws.

```bash
regula refs --source testdata/us-coppa.txt --external-only
```

**Output:**

```
External Reference Report: CHILDREN'S ONLINE PRIVACY PROTECTION ACT OF 1998
===================================================

Total external references: 1
Unique external targets:   1

External Documents Referenced:
+------+---------------------------------------------------+
| Refs | Target Document                                   |
+------+---------------------------------------------------+
|    1 | regulation (eu) 2016/679                          |
+------+---------------------------------------------------+
```

COPPA's single external reference points to the EU GDPR (Regulation 2016/679), reflecting the international dimension of children's data protection — COPPA's security requirements reference GDPR standards.

Comparing COPPA directly with the CCPA shows the state-federal relationship:

```bash
regula compare --sources testdata/ccpa.txt,testdata/us-coppa.txt --format table
```

**Output:**

```
Shared definitions:     2
Shared rights:          0
Shared obligations:     2

Shared Definitions:
  - person
  - personal information

Shared Obligations:
  - noticeatcollectionobligation
  - obligation
```

**"personal information"** is defined in both COPPA and the CCPA — COPPA narrowly (children's data collected online) and CCPA broadly (any California consumer's data). The shared "noticeatcollectionobligation" shows a direct lineage: COPPA's requirement that operators notify parents before collecting children's data predates and influenced the CCPA's broader notice-at-collection requirement for all consumer data.

---

## 11. Complete US Privacy Analysis Workflow

Reproduce the entire analysis with these commands:

```bash
# Build
go build -o regula ./cmd/regula

# 1. Ingest all US privacy laws
regula ingest --source testdata/ccpa.txt
regula ingest --source testdata/vcdpa.txt
regula ingest --source testdata/tdpsa.txt
regula ingest --source testdata/us-coppa.txt

# 2. Pairwise comparison: California vs Virginia
regula compare --sources testdata/ccpa.txt,testdata/vcdpa.txt --format table

# 3. Three-way comparison: add Texas
regula compare --sources testdata/ccpa.txt,testdata/vcdpa.txt,testdata/tdpsa.txt --format table

# 4. Full four-document analysis with federal COPPA
regula compare --sources testdata/ccpa.txt,testdata/vcdpa.txt,testdata/tdpsa.txt,testdata/us-coppa.txt --format table

# 5. External reference analysis per state
regula refs --source testdata/ccpa.txt --external-only
regula refs --source testdata/vcdpa.txt --external-only
regula refs --source testdata/tdpsa.txt --external-only
regula refs --source testdata/us-coppa.txt --external-only

# 6. Full reference analysis (internal + external)
regula refs --source testdata/ccpa.txt
regula refs --source testdata/vcdpa.txt

# 7. Rights and obligations per state
regula query --source testdata/ccpa.txt --template right-types
regula query --source testdata/vcdpa.txt --template right-types
regula query --source testdata/tdpsa.txt --template right-types
regula query --source testdata/ccpa.txt --template obligation-types
regula query --source testdata/vcdpa.txt --template obligation-types

# 8. Visualize the cross-reference network
regula compare --sources testdata/ccpa.txt,testdata/vcdpa.txt,testdata/tdpsa.txt,testdata/us-coppa.txt --format dot --output us-privacy-network.dot

# 9. Export comparison as JSON for further analysis
regula compare --sources testdata/ccpa.txt,testdata/vcdpa.txt,testdata/tdpsa.txt,testdata/us-coppa.txt --format json --output us-privacy-analysis.json
```

---

## 12. Key Findings

This analysis reveals several structural patterns in US privacy law:

**1. The Virginia Model dominates.** Texas adopted 14 of Virginia's defined terms verbatim. Post-2021 state privacy laws converge on VCDPA's GDPR-influenced terminology (controller, processor, consent, biometric data, sensitive data) rather than CCPA's California-specific language (business, service provider, personal information).

**2. Three rights are universal.** RightToKnow, RightToDelete, and a general right framework appear in every state law analyzed. These represent the minimum baseline for US state privacy legislation.

**3. Federal statutes serve as exemption anchors.** State laws don't build upon federal statutes — they carve out exemptions for data already governed by them. HIPAA (health), COPPA (children), GLBA (financial), FCRA (credit), FERPA (education), and DPPA (driver's records) form the federal framework that state laws must navigate around.

**4. Virginia is the most externally connected.** With 18 external references to 11 distinct federal statutes, the VCDPA has the densest federal-reference network, reflecting Virginia's explicit approach to delineating the boundary between state and federal data protection.

**5. COPPA influenced state notice obligations.** The "noticeatcollectionobligation" shared between COPPA and CCPA demonstrates how a narrow federal requirement (parental notice for children's data) expanded into a general state-level consumer notice mandate.

**6. Terminology divergence persists.** California uses "personal information" while Virginia-model states use "personal data." California defines "business" while Virginia-model states define "controller." These terminological differences create compliance complexity for organizations operating across multiple states.

---

## Next Steps

- See [TUTORIAL.md](TUTORIAL.md) for a comprehensive GDPR-focused tutorial with impact analysis and scenario matching
- See [TUTORIAL_CCPA.md](TUTORIAL_CCPA.md) for a CCPA deep-dive with GDPR comparison
- Review [TESTING.md](TESTING.md) for development and testing strategies
- See [ARCHITECTURE.md](ARCHITECTURE.md) for system design details
- Check [ROADMAP.md](ROADMAP.md) for upcoming features
