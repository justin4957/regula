# Regula RDF Ontology

This document describes the RDF ontology used by Regula to represent regulatory documents as queryable knowledge graphs.

## Namespaces

| Prefix | Namespace URI | Description |
|--------|---------------|-------------|
| `reg:` | `https://regula.dev/ontology#` | Regula regulation ontology |
| `rdf:` | `http://www.w3.org/1999/02/22-rdf-syntax-ns#` | RDF core vocabulary |
| `rdfs:` | `http://www.w3.org/2000/01/rdf-schema#` | RDF Schema |
| `xsd:` | `http://www.w3.org/2001/XMLSchema#` | XML Schema datatypes |
| `dc:` | `http://purl.org/dc/terms/` | Dublin Core metadata |
| `eli:` | `http://data.europa.eu/eli/ontology#` | European Legislation Identifier |
| `frbr:` | `http://purl.org/vocab/frbr/core#` | Functional Requirements for Bibliographic Records |

## Classes

### Document Types

| Class | Description | Example |
|-------|-------------|---------|
| `reg:Regulation` | Top-level EU regulation | GDPR |
| `reg:Directive` | EU directive | Directive 95/46/EC |
| `reg:Decision` | EU decision | Decision 2010/87/EU |

### Structural Elements

| Class | Description | Example |
|-------|-------------|---------|
| `reg:Chapter` | Chapter within a regulation | Chapter III (Rights) |
| `reg:Section` | Section within a chapter | Section 2 (Information) |
| `reg:Article` | Article (main provision unit) | Article 17 |
| `reg:Paragraph` | Numbered paragraph within article | Article 17(1) |
| `reg:Point` | Lettered point within paragraph | Article 6(1)(a) |
| `reg:SubPoint` | Sub-point within a point | - |
| `reg:Recital` | Preamble recital | Recital 39 |
| `reg:Preamble` | Preamble section | - |

### Semantic Elements

| Class | Description | Example |
|-------|-------------|---------|
| `reg:DefinedTerm` | Defined term from Article 4 | "personal data" |
| `reg:Reference` | Cross-reference | Art 17 → Art 6 |
| `reg:Obligation` | Obligation imposed by provision | Notification obligation |
| `reg:Right` | Right granted by provision | Right to erasure |

## Properties

### Metadata Properties

| Property | Domain | Range | Description |
|----------|--------|-------|-------------|
| `reg:title` | Any | `xsd:string` | Title text |
| `reg:text` | Any | `xsd:string` | Full text content |
| `reg:number` | Any | `xsd:string` | Number/identifier |
| `reg:identifier` | `reg:Regulation` | `xsd:string` | Formal ID (e.g., "(EU) 2016/679") |
| `reg:date` | Any | `xsd:date` | Relevant date |

### Structural Properties

| Property | Domain | Range | Description |
|----------|--------|-------|-------------|
| `reg:partOf` | Any | Any | Child → Parent containment |
| `reg:contains` | Any | Any | Parent → Child containment |
| `reg:belongsTo` | Any | `reg:Regulation` | Membership in regulation |
| `reg:hasChapter` | `reg:Regulation` | `reg:Chapter` | Regulation → Chapter |
| `reg:hasSection` | `reg:Chapter` | `reg:Section` | Chapter → Section |
| `reg:hasArticle` | `reg:Chapter`/`reg:Section` | `reg:Article` | Container → Article |
| `reg:hasParagraph` | `reg:Article` | `reg:Paragraph` | Article → Paragraph |
| `reg:hasPoint` | `reg:Paragraph` | `reg:Point` | Paragraph → Point |

### Reference Properties

| Property | Domain | Range | Description |
|----------|--------|-------|-------------|
| `reg:references` | Any | Any | Cross-reference to provision |
| `reg:referencedBy` | Any | Any | Inverse of references |
| `reg:externalRef` | Any | Any | Reference to external document |
| `reg:refersToArticle` | Any | `reg:Article` | Specific article reference |
| `reg:refersToChapter` | Any | `reg:Chapter` | Specific chapter reference |

### Definition Properties

| Property | Domain | Range | Description |
|----------|--------|-------|-------------|
| `reg:definedIn` | `reg:DefinedTerm` | `reg:Article` | Where term is defined |
| `reg:defines` | `reg:Article` | `reg:DefinedTerm` | What terms article defines |
| `reg:definition` | `reg:DefinedTerm` | `xsd:string` | Definition text |
| `reg:term` | `reg:DefinedTerm` | `xsd:string` | The defined term |
| `reg:normalizedTerm` | `reg:DefinedTerm` | `xsd:string` | Lowercase normalized form |
| `reg:usesTerm` | Any | `reg:DefinedTerm` | Provision uses defined term |

### Amendment Properties

| Property | Domain | Range | Description |
|----------|--------|-------|-------------|
| `reg:amends` | `reg:Regulation` | `reg:Regulation` | Amendment relationship |
| `reg:amendedBy` | `reg:Regulation` | `reg:Regulation` | Inverse of amends |
| `reg:supersedes` | `reg:Regulation` | `reg:Regulation` | Replacement relationship |
| `reg:repeals` | Any | Any | Repeal relationship |
| `reg:delegatesTo` | Any | Any | Delegation of power |

### Semantic Properties

| Property | Domain | Range | Description |
|----------|--------|-------|-------------|
| `reg:grantsRight` | Any | `reg:Right` | Provision grants a right |
| `reg:imposesObligation` | Any | `reg:Obligation` | Provision creates obligation |
| `reg:requires` | Any | Any | Requirement (e.g., consent) |
| `reg:prohibits` | Any | Any | Prohibition |
| `reg:permits` | Any | Any | Permission |
| `reg:exempts` | Any | Any | Exemption |
| `reg:appliesTo` | Any | Any | Applicability |
| `reg:subjectTo` | Any | Any | Subject to conditions |

### Entity Properties

| Property | Domain | Range | Description |
|----------|--------|-------|-------------|
| `reg:actor` | Any | Any | Actor in obligation/right |
| `reg:beneficiary` | `reg:Right` | Any | Who benefits from right |
| `reg:dutyBearer` | `reg:Obligation` | Any | Who bears obligation |
| `reg:dataSubject` | Any | Any | Relation to data subject |
| `reg:controller` | Any | Any | Relation to controller |
| `reg:processor` | Any | Any | Relation to processor |

### Temporal Properties

| Property | Domain | Range | Description |
|----------|--------|-------|-------------|
| `reg:effectiveDate` | Any | `xsd:date` | When provision takes effect |
| `reg:expiryDate` | Any | `xsd:date` | When provision expires |
| `reg:deadline` | Any | `xsd:string` | Compliance deadline |
| `reg:timeLimit` | Any | `xsd:string` | Time limit text |

### Provenance Properties

| Property | Domain | Range | Description |
|----------|--------|-------|-------------|
| `reg:sourceDocument` | Any | `xsd:anyURI` | Source document URI |
| `reg:sourceOffset` | Any | `xsd:integer` | Character offset in source |
| `reg:sourceLength` | Any | `xsd:integer` | Length in source |
| `reg:extractedFrom` | Any | Any | Extraction source |
| `reg:extractedAt` | Any | `xsd:dateTime` | Extraction timestamp |

## Named Instances

### Rights (GDPR)

| Instance | Description | Source |
|----------|-------------|--------|
| `reg:RightOfAccess` | Right of access | Art 15 |
| `reg:RightToRectification` | Right to rectification | Art 16 |
| `reg:RightToErasure` | Right to erasure | Art 17 |
| `reg:RightToRestriction` | Right to restriction | Art 18 |
| `reg:RightToDataPortability` | Right to data portability | Art 20 |
| `reg:RightToObject` | Right to object | Art 21 |
| `reg:RightAgainstAutomatedDecision` | Right against automated decision | Art 22 |
| `reg:RightToWithdrawConsent` | Right to withdraw consent | Art 7(3) |
| `reg:RightToLodgeComplaint` | Right to lodge complaint | Art 77 |
| `reg:RightToEffectiveRemedy` | Right to effective remedy | Art 78, 79 |
| `reg:RightToCompensation` | Right to compensation | Art 82 |
| `reg:RightToInformation` | Right to information | Art 13, 14 |

### Obligations (GDPR)

| Instance | Description | Source |
|----------|-------------|--------|
| `reg:TransparencyObligation` | Transparency obligation | Art 12 |
| `reg:NotificationObligation` | Notification obligation | Art 19, 33, 34 |
| `reg:SecurityObligation` | Security obligation | Art 32 |
| `reg:RecordKeepingObligation` | Record keeping obligation | Art 30 |
| `reg:ImpactAssessmentObligation` | DPIA obligation | Art 35 |
| `reg:CooperationObligation` | Cooperation obligation | Art 31 |
| `reg:AppointmentObligation` | DPO appointment obligation | Art 37 |

### Legal Bases (GDPR Article 6)

| Instance | Description |
|----------|-------------|
| `reg:Consent` | Consent of data subject |
| `reg:ContractPerformance` | Performance of contract |
| `reg:LegalObligation` | Compliance with legal obligation |
| `reg:VitalInterest` | Vital interests |
| `reg:PublicTask` | Public interest or official authority |
| `reg:LegitimateInterest` | Legitimate interests |

## URI Patterns

### Regulation URIs

```
https://regula.dev/regulations/GDPR#
https://regula.dev/regulations/GDPR#Art17
https://regula.dev/regulations/GDPR#ChapterIII
https://regula.dev/regulations/GDPR#ChapterIII:Section2
https://regula.dev/regulations/GDPR#Art6:1:a
https://regula.dev/regulations/GDPR#Recital39
https://regula.dev/regulations/GDPR#Term:personal_data
```

### Pattern Structure

| Element | Pattern | Example |
|---------|---------|---------|
| Regulation | `{base}` | `GDPR#` |
| Chapter | `{base}Chapter{num}` | `GDPR#ChapterIII` |
| Section | `{base}Chapter{num}:Section{num}` | `GDPR#ChapterIII:Section2` |
| Article | `{base}Art{num}` | `GDPR#Art17` |
| Paragraph | `{base}Art{num}:{para}` | `GDPR#Art17:1` |
| Point | `{base}Art{num}:{para}:{letter}` | `GDPR#Art6:1:a` |
| Recital | `{base}Recital{num}` | `GDPR#Recital39` |
| Defined Term | `{base}Term:{normalized}` | `GDPR#Term:personal_data` |

## Example Triples

### GDPR Article 17 (Right to Erasure)

```turtle
@prefix reg: <https://regula.dev/ontology#> .
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix gdpr: <https://regula.dev/regulations/GDPR#> .

# Article metadata
gdpr:Art17 rdf:type reg:Article .
gdpr:Art17 reg:number "17" .
gdpr:Art17 reg:title "Right to erasure ('right to be forgotten')" .
gdpr:Art17 reg:text "1. The data subject shall have the right to obtain..." .

# Structural relationships
gdpr:Art17 reg:partOf gdpr:ChapterIII .
gdpr:Art17 reg:belongsTo gdpr: .
gdpr:ChapterIII reg:contains gdpr:Art17 .

# Cross-references
gdpr:Art17 reg:references gdpr:Art6 .
gdpr:Art17 reg:references gdpr:Art9 .
gdpr:Art17 reg:references gdpr:Art17:3 .

# Semantic annotations
gdpr:Art17 reg:grantsRight reg:RightToErasure .
gdpr:Art17 reg:beneficiary reg:DataSubject .
gdpr:Art17 reg:dutyBearer reg:Controller .

# Provenance
gdpr:Art17 reg:sourceOffset 45678 .
gdpr:Art17 reg:sourceLength 2340 .
```

### GDPR Article 4 Definition

```turtle
@prefix reg: <https://regula.dev/ontology#> .
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix gdpr: <https://regula.dev/regulations/GDPR#> .

# Defined term
gdpr:Term:personal_data rdf:type reg:DefinedTerm .
gdpr:Term:personal_data reg:term "personal data" .
gdpr:Term:personal_data reg:normalizedTerm "personal data" .
gdpr:Term:personal_data reg:number "1" .
gdpr:Term:personal_data reg:definedIn gdpr:Art4 .
gdpr:Term:personal_data reg:definition "any information relating to an identified or identifiable natural person ('data subject')..." .

# Article 4 defines this term
gdpr:Art4 reg:defines gdpr:Term:personal_data .
```

### Cross-Reference Example

```turtle
@prefix reg: <https://regula.dev/ontology#> .
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix gdpr: <https://regula.dev/regulations/GDPR#> .

# Article 17 references Article 6(1)
gdpr:Art17 reg:references gdpr:Art6 .
gdpr:Art17 reg:refersToArticle gdpr:Art6 .
gdpr:Art6 reg:referencedBy gdpr:Art17 .

# Reference to external directive
gdpr:Art1 reg:externalRef <https://eur-lex.europa.eu/Directive/95/46/EC> .
```

## Query Examples

### Find All Articles

```sparql
SELECT ?article ?title
WHERE {
  ?article rdf:type reg:Article .
  ?article reg:title ?title .
}
ORDER BY ?article
```

### Find Articles Granting Rights

```sparql
SELECT ?article ?right
WHERE {
  ?article rdf:type reg:Article .
  ?article reg:grantsRight ?right .
}
```

### Find All References to Article 6

```sparql
SELECT ?source ?sourceTitle
WHERE {
  ?source reg:references <GDPR:Art6> .
  ?source reg:title ?sourceTitle .
}
```

### Find Definitions

```sparql
SELECT ?term ?definition
WHERE {
  ?def rdf:type reg:DefinedTerm .
  ?def reg:term ?term .
  ?def reg:definition ?definition .
}
```

### Find Chapter Structure

```sparql
SELECT ?chapter ?chapterTitle ?article ?articleTitle
WHERE {
  ?chapter rdf:type reg:Chapter .
  ?chapter reg:title ?chapterTitle .
  ?article reg:partOf ?chapter .
  ?article reg:title ?articleTitle .
}
ORDER BY ?chapter ?article
```

## ELI Vocabulary Mapping

The [European Legislation Identifier (ELI)](http://data.europa.eu/eli/ontology) vocabulary is a standard for identifying and describing legal resources. Regula supports optional ELI enrichment that adds ELI triples **alongside** existing `reg:` triples, enabling interoperability with EUR-Lex and national legal information systems.

### When ELI Applies

ELI enrichment is only applied to **EU document types**: Regulation, Directive, and Decision. Non-EU documents (statutes, acts, generic) are not enriched with ELI vocabulary.

Enable ELI enrichment with the `--eli` flag:

```bash
regula export --source gdpr.txt --format turtle --eli
```

### ELI Class Mapping

| Regula Class | ELI Class | Description |
|---|---|---|
| `reg:Regulation` | `eli:LegalResource` | Top-level EU regulation |
| `reg:Directive` | `eli:LegalResource` | EU directive |
| `reg:Decision` | `eli:LegalResource` | EU decision |
| `reg:Chapter` | `eli:LegalResourceSubdivision` | Chapter within a regulation |
| `reg:Section` | `eli:LegalResourceSubdivision` | Section within a chapter |
| `reg:Article` | `eli:LegalResourceSubdivision` | Article (main provision unit) |
| `reg:Paragraph` | `eli:LegalResourceSubdivision` | Numbered paragraph |
| `reg:Point` | `eli:LegalResourceSubdivision` | Lettered point |
| `reg:Preamble` | `eli:LegalResourceSubdivision` | Preamble section |
| `reg:Recital` | `eli:LegalResourceSubdivision` | Preamble recital |

### ELI Property Mapping

| Regula Property | ELI Property | Description |
|---|---|---|
| `reg:title` | `eli:title` | Title of the resource |
| `reg:number` | `eli:id_local` | Local identifier (article number) |
| `reg:identifier` | `eli:id_local` | Formal identifier |
| `reg:partOf` | `eli:is_part_of` | Hierarchical containment (child → parent) |
| `reg:contains` | `eli:has_part` | Hierarchical containment (parent → child) |
| `reg:date` | `eli:date_document` | Document date |
| `reg:version` | `eli:version` | Version identifier |
| `reg:references` | `eli:cites` | Citation relationship |
| `reg:referencedBy` | `eli:cited_by` | Incoming citation (inverse) |

**Note**: `reg:text` is intentionally **not** mapped to `eli:description`. ELI's `description` property represents a summary, while `reg:text` contains the full provision text.

### Example: Dual-Typed Turtle Output

With ELI enrichment enabled, resources carry both `reg:` and `eli:` types:

```turtle
@prefix reg: <https://regula.dev/ontology#> .
@prefix eli: <http://data.europa.eu/eli/ontology#> .
@prefix gdpr: <https://regula.dev/regulations/GDPR#> .

gdpr:Art17 a reg:Article ,
             eli:LegalResourceSubdivision ;
    reg:title "Right to erasure ('right to be forgotten')" ;
    eli:title "Right to erasure ('right to be forgotten')" ;
    reg:number "17" ;
    eli:id_local "17" ;
    reg:partOf gdpr:ChapterIII ;
    eli:is_part_of gdpr:ChapterIII .
```

### ELI Version Compatibility

The ELI properties used are stable across both ELI 1.x and 2.x schemas. The `eli:version` property is part of ELI 2.x.

### FRBR Scope

The ELI ontology is built on the FRBR (Functional Requirements for Bibliographic Records) model with three levels:

- **Work** level: the abstract legislative intent → mapped via `eli:LegalResource`
- **Expression** level: a particular version or language → `eli:LegalExpression` (available but not used yet)
- **Manifestation** level: a specific publication format → not applicable

Regula currently maps at the **Work level** only. Expression and Manifestation levels are available for future multi-version or multi-language support.

## Schema Validation

The schema is designed to satisfy these requirements:

1. **Completeness**: Covers all provision types from `pkg/extract`:
   - Document, Chapter, Section, Article, Paragraph, Point
   - Recital, Definition, Reference

2. **Relationships**: Supports all relationship types:
   - Hierarchical (partOf, contains)
   - Cross-references (references, referencedBy)
   - Amendments (amends, supersedes, repeals)
   - Semantic (grantsRight, imposesObligation)

3. **GDPR Specificity**: Includes GDPR-specific concepts:
   - All 6 legal bases from Article 6
   - Data subject rights from Chapter III
   - Controller/processor obligations

4. **Provenance**: Tracks source information:
   - Source document, offset, length
   - Extraction timestamps

5. **Extensibility**: Uses standard RDF patterns:
   - Clear namespace separation
   - Consistent URI patterns
   - Compatible with SPARQL queries

6. **Interoperability**: ELI vocabulary integration:
   - Optional ELI enrichment for EU documents
   - Compatible with EUR-Lex ELI usage
   - Additive mapping preserves `reg:` triples
