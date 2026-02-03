# Regula Testing Guide

This document describes testing strategies for the Regula regulation mapper.

## Test Categories

### Unit Tests

Run all unit tests:

```bash
go test ./... -v
```

Run tests for a specific package:

```bash
go test ./pkg/extract/... -v
go test ./pkg/store/... -v
go test ./pkg/simulate/... -v
go test ./pkg/analysis/... -v
go test ./pkg/draft/... -v
```

### Integration Tests

Integration tests use the GDPR test data at `testdata/gdpr.txt`:

```bash
go test ./pkg/simulate/... -v -run TestGDPR
go test ./pkg/extract/... -v -run TestIntegration
```

### ELI Vocabulary Tests

Test ELI (European Legislation Identifier) enrichment:

```bash
# Run all ELI mapper tests
go test ./pkg/store/... -run "TestIsEU|TestEnrich" -v

# Run GDPR ELI integration test
go test ./pkg/store/... -run TestEnrichWithELI_GDPRIntegration -v
```

### JSON-LD Serialization Tests

Test JSON-LD (Linked Data) export format:

```bash
# Run all JSON-LD tests
go test ./pkg/store/... -v -run TestJSONLD

# Run specific JSON-LD test categories
go test ./pkg/store/... -run TestJSONLD_Serialize -v       # Serialization tests
go test ./pkg/store/... -run TestJSONLD_CompactURI -v      # URI compaction
go test ./pkg/store/... -run TestJSONLD_ExpandURI -v       # URI expansion
go test ./pkg/store/... -run TestJSONLD_PredicateToKey -v  # Predicate mapping
go test ./pkg/store/... -run TestJSONLD_IsRelationship -v  # Relationship detection

# Run GDPR JSON-LD integration test
go test ./pkg/store/... -run TestJSONLD_Serialize_GDPRIntegration -v

# CLI: Export as compact JSON-LD (with @context)
regula export --source testdata/gdpr.txt --format jsonld --output graph.jsonld

# CLI: Export as expanded JSON-LD (full URIs, no @context)
regula export --source testdata/gdpr.txt --format jsonld --expanded --output graph-expanded.jsonld

# CLI: Export with ELI enrichment
regula export --source testdata/gdpr.txt --format jsonld --eli --output graph-eli.jsonld
```

### RDF/XML Serialization Tests

Test RDF/XML export format for legacy system compatibility:

```bash
# Run all RDF/XML tests
go test ./pkg/store/... -v -run "TestRDFXML|TestEscapeXML|TestSplitPrefixedName|TestExpandToFullURI|TestIsURIObject" -count=1

# Run serialization tests
go test ./pkg/store/... -run TestRDFXMLSerialize -v

# Run individual serialization tests
go test ./pkg/store/... -run TestRDFXMLSerialize_EmptyStore -v          # Minimal valid XML
go test ./pkg/store/... -run TestRDFXMLSerialize_SingleTriple -v        # Single Description block
go test ./pkg/store/... -run TestRDFXMLSerialize_SubjectGrouping -v     # Multiple predicates grouped
go test ./pkg/store/... -run TestRDFXMLSerialize_URIObject -v           # rdf:resource attribute
go test ./pkg/store/... -run TestRDFXMLSerialize_RDFType -v             # rdf:type with resource
go test ./pkg/store/... -run TestRDFXMLSerialize_XMLEscaping -v         # & < > escaping
go test ./pkg/store/... -run TestRDFXMLSerialize_NamespaceDeclarations -v  # xmlns attributes
go test ./pkg/store/... -run TestRDFXMLSerialize_MultipleObjects -v     # Separate elements per object
go test ./pkg/store/... -run TestRDFXMLSerialize_TypeFirst -v           # rdf:type ordered first

# Run XML escaping tests
go test ./pkg/store/... -run TestEscapeXMLText -v
go test ./pkg/store/... -run TestEscapeXMLAttribute -v

# Run URI splitting tests
go test ./pkg/store/... -run TestSplitPrefixedName -v

# Run GDPR integration test
go test ./pkg/store/... -run TestRDFXMLSerialize_GDPRIntegration -v

# Run concurrent access test
go test ./pkg/store/... -run TestRDFXMLSerialize_ConcurrentAccess -v

# CLI: Export as RDF/XML
regula export --source testdata/gdpr.txt --format rdfxml
regula export --source testdata/gdpr.txt --format rdfxml --output graph.rdf
regula export --source testdata/gdpr.txt --format xml --output graph.xml
regula export --source testdata/gdpr.txt --format rdfxml --eli --output graph-eli.rdf
```

### Citation Parser Tests

Test the extensible citation parser interface, EU citation parser, Bluebook (US) parser, and OSCOLA (UK) parser:

```bash
# Run all citation parser tests
go test ./pkg/citation/... -v

# Run EU citation parser tests
go test ./pkg/citation/... -run TestEUCitationParser -v

# Run Bluebook (US) citation parser tests
go test ./pkg/citation/... -run TestBluebook -v

# Run Bluebook USC/CFR/Public Law tests
go test ./pkg/citation/... -run "TestBluebookParserParseUSC|TestBluebookParserParseCFR|TestBluebookParserParsePublicLaw" -v

# Run OSCOLA (UK/Commonwealth) citation parser tests
go test ./pkg/citation/... -run TestOSCOLA -v

# Run OSCOLA-specific citation type tests
go test ./pkg/citation/... -run "TestOSCOLAParserParseActs|TestOSCOLAParserParseSI|TestOSCOLAParserParseCases" -v

# Run OSCOLA section/schedule/part tests
go test ./pkg/citation/... -run "TestOSCOLAParserParseSections|TestOSCOLAParserParseSchedules|TestOSCOLAParserParseParts" -v

# Run OSCOLA ECHR article tests
go test ./pkg/citation/... -run TestOSCOLAParserParseECHR -v

# Run UK integration tests (DPA 2018, SI example)
go test ./pkg/citation/... -run "TestOSCOLAParserDPA2018Integration|TestOSCOLAParserSIIntegration" -v

# Run CCPA/VCDPA integration tests (US citations)
go test ./pkg/citation/... -run "TestBluebookParserCCPAIntegration|TestBluebookParserVCDPAIntegration" -v

# Run registry operation tests
go test ./pkg/citation/... -run TestCitationRegistry -v

# Run bridge conversion tests
go test ./pkg/citation/... -run "TestCitationFromReference|TestReferenceFromCitation|TestRoundtrip|TestBatch" -v

# Run GDPR integration test (parses full GDPR text)
go test ./pkg/citation/... -run TestEUCitationParserGDPRIntegration -v
```

### Temporal Reference Extraction Tests

Test temporal qualifier detection ("as amended by", "as in force on", "repealed by", etc.):

```bash
# Run all temporal extraction tests
go test ./pkg/extract/... -v -run TestExtractTemporal

# Run individual temporal pattern tests
go test ./pkg/extract/... -run TestExtractTemporalAsAmendedBy -v      # "as amended by {doc}"
go test ./pkg/extract/... -run TestExtractTemporalAsAmended -v         # "as amended" standalone
go test ./pkg/extract/... -run TestExtractTemporalAsInForceOn -v       # "as in force on {date}"
go test ./pkg/extract/... -run TestExtractTemporalEnterIntoForce -v    # "enter(s/ed) into force"
go test ./pkg/extract/... -run TestExtractTemporalAsOriginallyEnacted -v  # "as originally enacted"
go test ./pkg/extract/... -run TestExtractTemporalAsItStoodOn -v       # "as it stood on {date}"
go test ./pkg/extract/... -run TestExtractTemporalConsolidated -v      # "consolidated version"
go test ./pkg/extract/... -run TestExtractTemporalRepealedBy -v        # "repealed by {doc}"
go test ./pkg/extract/... -run TestExtractTemporalRepealedWithEffect -v  # "repealed with effect from {date}"

# Run date parsing tests
go test ./pkg/extract/... -run TestExtractTemporalParseEuropeanDate -v

# Run GDPR integration test (verifies temporal refs from Article 92, 94, 99)
go test ./pkg/extract/... -run TestExtractTemporalGDPRIntegration -v

# Run UK SI integration test
go test ./pkg/extract/... -run TestExtractTemporalUKSIIntegration -v

# Run temporal builder tests (verifies RDF triple emission)
go test ./pkg/store/... -run TestBuildReferenceWithTemporal -v
go test ./pkg/store/... -run TestBuildGDPRGraph_TemporalReferences -v
```

### EUR-Lex Connector Tests

Test the EUR-Lex connector for CELEX number generation, ELI URI generation, validation caching, and HTTP client integration:

```bash
# Run all EUR-Lex connector tests
go test ./pkg/eurlex/... -v

# Run CELEX generation tests
go test ./pkg/eurlex/... -run TestGenerateCELEX -v

# Run ELI URI generation tests
go test ./pkg/eurlex/... -run TestGenerateELI -v

# Run year normalization and number padding tests
go test ./pkg/eurlex/... -run "TestNormalizeYear|TestPadCELEXNumber" -v

# Run validation cache tests
go test ./pkg/eurlex/... -run TestCache -v

# Run HTTP client and validation tests (uses mock HTTP client)
go test ./pkg/eurlex/... -run "TestValidate|TestFetch|TestUserAgent|TestHTTPMethod" -v

# Verify ELI does not pad numbers (unlike CELEX)
go test ./pkg/eurlex/... -run TestELIDoesNotPadNumbers -v
```

### Batch Link Validation Tests

Test the batch link validator for external reference URI validation with rate limiting:

```bash
# Run all linkcheck tests
go test ./pkg/linkcheck/... -v

# Run type and config tests
go test ./pkg/linkcheck/... -run "TestLinkResult|TestExtractDomain|TestBatchConfig|TestValidationProgress" -v

# Run report tests
go test ./pkg/linkcheck/... -run "TestValidationReport" -v

# Run cache tests
go test ./pkg/linkcheck/... -run "TestLinkCache" -v

# Run validator tests (uses mock HTTP server)
go test ./pkg/linkcheck/... -run "TestBatchValidator" -v

# Run rate limiting tests
go test ./pkg/linkcheck/... -run "TestRateLimited|TestDomainRateLimiter" -v

# CLI: Run link validation
regula validate --source testdata/gdpr.txt --check links

# CLI: Save link report to JSON
regula validate --source testdata/gdpr.txt --check links --report links.json

# CLI: Save link report to Markdown
regula validate --source testdata/gdpr.txt --check links --report links.md
```

### US Code Connector Tests

Test the US Code connector for USC/CFR URI generation, validation caching, and HTTP client integration:

```bash
# Run all US Code connector tests
go test ./pkg/uscode/... -v

# Run URI generation tests
go test ./pkg/uscode/... -run "TestGenerateUSCURI|TestGenerateCFRURI" -v

# Run citation parsing tests
go test ./pkg/uscode/... -run "TestParseUSCNumber|TestParseCFRNumber" -v

# Run validation tests (uses mock HTTP client)
go test ./pkg/uscode/... -run "TestValidateURI|TestValidateUSC|TestValidateCFR" -v

# Run caching tests
go test ./pkg/uscode/... -run TestCaching -v

# Run real connection integration tests (hits uscode.house.gov and ecfr.gov)
go test ./pkg/uscode/... -run "TestIntegration" -v

# Connection summary test (validates all major citations)
go test ./pkg/uscode/... -run TestIntegration_ConnectionSummary -v
```

**Note:** Integration tests hit real government servers (uscode.house.gov, ecfr.gov). They are skipped with `-short` flag and may be affected by network conditions.

### SPARQL CONSTRUCT Query Tests

Test the SPARQL CONSTRUCT query support for graph extraction and transformation:

```bash
# Run all CONSTRUCT query tests
go test ./pkg/query/... -v -run Construct

# Run CONSTRUCT parsing tests
go test ./pkg/query/... -run "TestParseQuery_SimpleConstruct|TestParseQuery_ConstructMultiple|TestParseQuery_ConstructWithOptional|TestParseQuery_ConstructWithFilter" -v

# Run CONSTRUCT validation tests
go test ./pkg/query/... -run TestConstructQuery_Validate -v

# Run CONSTRUCT execution tests
go test ./pkg/query/... -run "TestExecutor_SimpleConstruct|TestExecutor_ConstructWithFilter|TestExecutor_ConstructWithOptional|TestExecutor_ConstructDeduplication" -v

# Run CONSTRUCT result formatting tests
go test ./pkg/query/... -run "TestConstructResult_FormatTurtle|TestConstructResult_FormatNTriples|TestConstructResult_FormatJSON" -v

# CLI: Run CONSTRUCT query with Turtle output (default)
regula query --source testdata/gdpr.txt \
  "CONSTRUCT { ?a <http://example.org/title> ?t } WHERE { ?a rdf:type reg:Article . ?a reg:title ?t }"

# CLI: Run CONSTRUCT query with N-Triples output
regula query --source testdata/gdpr.txt --format ntriples \
  "CONSTRUCT { ?a <http://example.org/type> <http://example.org/Article> } WHERE { ?a rdf:type reg:Article }"

# CLI: Run CONSTRUCT query with JSON output
regula query --source testdata/gdpr.txt --format json \
  "CONSTRUCT { ?a <http://example.org/type> <http://example.org/Article> } WHERE { ?a rdf:type reg:Article }"

# CLI: Run CONSTRUCT query with timing
regula query --source testdata/gdpr.txt --timing \
  "CONSTRUCT { ?a <http://example.org/title> ?t } WHERE { ?a rdf:type reg:Article . ?a reg:title ?t }"
```

### SPARQL DESCRIBE Query Tests

Test the SPARQL DESCRIBE query support for entity-centric summaries:

```bash
# Run all DESCRIBE parser tests
go test ./pkg/query/... -v -run "TestParseQuery_Describe|TestDescribeQuery"

# Run DESCRIBE execution tests
go test ./pkg/query/... -v -run TestExecutor_Describe

# Run DESCRIBE bidirectional test (verifies subject + object lookup)
go test ./pkg/query/... -v -run TestExecutor_DescribeBidirectional

# Run DESCRIBE formatting tests
go test ./pkg/query/... -v -run "TestExecutor_DescribeFormatTurtle|TestExecutor_DescribeFormatJSON"

# Run DESCRIBE benchmarks
go test ./pkg/query/... -v -bench BenchmarkExecutor_Describe -benchmem

# CLI: DESCRIBE with direct URI
regula query --source testdata/gdpr.txt "DESCRIBE GDPR:Art17"

# CLI: DESCRIBE with variable
regula query --source testdata/gdpr.txt \
  "DESCRIBE ?article WHERE { ?article reg:title \"Right to erasure\" }"

# CLI: DESCRIBE with JSON output
regula query --source testdata/gdpr.txt --format json "DESCRIBE GDPR:Art17"

# CLI: DESCRIBE with timing
regula query --source testdata/gdpr.txt --timing "DESCRIBE GDPR:Art17"

# CLI: Use describe-article template
regula query --source testdata/gdpr.txt --template describe-article
```

### UK Legislation Connector Tests

Test the UK legislation.gov.uk connector for URI generation, validation caching, and HTTP client integration:

```bash
# Run all UK legislation connector tests
go test ./pkg/ukleg/... -v

# Run URI generation tests
go test ./pkg/ukleg/... -run "TestGenerateLegislationURI|TestGenerateSectionURI|TestLegislationURI" -v

# Run cache tests
go test ./pkg/ukleg/... -run TestCache -v

# Run validation tests (uses mock HTTP client)
go test ./pkg/ukleg/... -run "TestValidateURI|TestValidateCitation" -v

# Run metadata fetch tests
go test ./pkg/ukleg/... -run TestFetchMetadata -v

# Run HTTP method and header tests
go test ./pkg/ukleg/... -run "TestHTTPMethodUsed|TestUserAgentHeader|TestFetchMetadata_AcceptHeader" -v

# Run real connection integration tests (hits legislation.gov.uk)
go test ./pkg/ukleg/... -run TestIntegration -v

# Run end-to-end parse and validate test
go test ./pkg/ukleg/... -run TestIntegration_ParseAndValidate -v
```

**Note:** Integration tests hit real government servers (legislation.gov.uk). They are skipped with `-short` flag and may be affected by network conditions.

### Validation Gate Tests

Test the validation checkpoint/gate system with per-stage quality metrics:

```bash
# Run all gate tests
go test ./pkg/validate/... -v -run TestGate

# Run individual gate tests
go test ./pkg/validate/... -run TestSchemaGate -v    # V0: Schema validation
go test ./pkg/validate/... -run TestStructureGate -v  # V1: Structure validation
go test ./pkg/validate/... -run TestCoverageGate -v   # V2: Coverage validation
go test ./pkg/validate/... -run TestQualityGate -v    # V3: Quality validation

# Run pipeline behavior tests
go test ./pkg/validate/... -run TestGatePipeline -v

# Run gate report serialization tests
go test ./pkg/validate/... -run TestGateReport -v

# Run GDPR integration test with all 4 gates
go test ./pkg/validate/... -run TestGatePipeline_GDPRIntegration -v

# CLI: Run ingestion with gates enabled
regula ingest --source testdata/gdpr.txt --gates
regula ingest --source testdata/gdpr.txt --gates --strict
regula ingest --source testdata/gdpr.txt --gates --skip-gates V0

# CLI: Run gate-based validation
regula validate --source testdata/gdpr.txt --check gates
regula validate --source testdata/gdpr.txt --check gates --format json
```

### Validation Report Generation Tests

Test HTML and Markdown report generation for validation results and gate reports:

```bash
# Run all report generation tests
go test ./pkg/validate/... -v -run "TestValidationResult_To|TestGateReport_To" -count=1

# Run Markdown report tests
go test ./pkg/validate/... -v -run TestValidationResult_ToMarkdown -count=1
go test ./pkg/validate/... -v -run TestGateReport_ToMarkdown -count=1

# Run HTML report tests
go test ./pkg/validate/... -v -run TestValidationResult_ToHTML -count=1
go test ./pkg/validate/... -v -run TestGateReport_ToHTML -count=1

# Run helper function tests
go test ./pkg/validate/... -v -run "TestStatusToMarkdownBadge|TestEscapeMarkdownTableCell|TestStatusToHTMLColor|TestScoreToHTMLColor" -count=1

# CLI: Generate Markdown report to stdout
regula validate --source testdata/gdpr.txt --format markdown

# CLI: Generate HTML report to stdout
regula validate --source testdata/gdpr.txt --format html

# CLI: Generate gate report in Markdown
regula validate --source testdata/gdpr.txt --check gates --format markdown

# CLI: Save report to file (format based on extension)
regula validate --source testdata/gdpr.txt --report report.html
regula validate --source testdata/gdpr.txt --report report.md
regula validate --source testdata/gdpr.txt --check gates --report gates.html
```

### Test Coverage

| Test | What it verifies |
|------|------------------|
| `TestValidationResult_ToMarkdown` | Headers, summary table, scores, status badge |
| `TestValidationResult_ToMarkdown_AllComponents` | All 5 component sections rendered with data |
| `TestValidationResult_ToMarkdown_EmptyComponents` | Nil components omitted gracefully |
| `TestValidationResult_ToMarkdown_FailStatus` | FAIL badge rendered for failing results |
| `TestValidationResult_ToMarkdown_TableEscaping` | Pipe characters escaped in table cells |
| `TestValidationResult_ToHTML` | HTML structure, CSS, tables, details elements |
| `TestValidationResult_ToHTML_PassStatus` | Green color (#4caf50) for PASS |
| `TestValidationResult_ToHTML_FailStatus` | Red color (#f44336) for FAIL |
| `TestValidationResult_ToHTML_ComponentScoreBars` | Score bar widths match component scores |
| `TestValidationResult_ToHTML_HTMLEscaping` | XSS-safe HTML escaping |
| `TestValidationResult_ToHTML_EmptyComponents` | Valid HTML with no components |
| `TestGateReport_ToMarkdown` | Gate headers, summary, results |
| `TestGateReport_ToMarkdown_WithWarnings` | Warning content rendered |
| `TestGateReport_ToMarkdown_SkippedGates` | Skip badge and reason |
| `TestGateReport_ToMarkdown_FailedGates` | Fail badge and error content |
| `TestGateReport_ToHTML` | HTML structure with gate cards |
| `TestGateReport_ToHTML_PassStatus` | Green styling for overall pass |
| `TestGateReport_ToHTML_FailStatus` | Red styling for overall fail |
| `TestGateReport_ToHTML_HaltedPipeline` | Halted pipeline warning alert |
| `TestGateReport_ToHTML_SkippedGateCard` | Grey styling for skipped gates |

### Validation Profile Auto-Generation Tests

Test profile generation, document analysis, weight suggestion, YAML serialization, and file I/O:

```bash
# Run all profile generation tests
go test ./pkg/validate/... -v -run "TestSuggestProfile|TestProfileFromYAML|TestProfileSuggestion|TestAnalyzeDocument|TestSuggestWeights|TestComputeConfidence|TestLoadProfile|TestSaveProfile|TestRoundToDecimals" -count=1

# Run profile suggestion tests
go test ./pkg/validate/... -v -run TestSuggestProfile -count=1

# Run individual profile suggestion tests
go test ./pkg/validate/... -run TestSuggestProfile_LargeDocument -v      # 99 articles, 11 chapters
go test ./pkg/validate/... -run TestSuggestProfile_SmallDocument -v      # 5 articles, 2 chapters
go test ./pkg/validate/... -run TestSuggestProfile_NoDefinitions -v      # Definition weight reduced
go test ./pkg/validate/... -run TestSuggestProfile_HeavyReferences -v    # Reference weight boosted
go test ./pkg/validate/... -run TestSuggestProfile_NoSemantics -v        # Semantic weight reduced
go test ./pkg/validate/... -run TestSuggestProfile_Reasoning -v          # Reasoning entries present
go test ./pkg/validate/... -run TestSuggestProfile_KnownRightsAndObligations -v  # Rights/obligations classified

# Run YAML serialization tests
go test ./pkg/validate/... -v -run "TestProfileSuggestion_ToYAML|TestProfileFromYAML" -count=1

# Run file I/O tests
go test ./pkg/validate/... -v -run "TestLoadProfileFromFile|TestSaveProfileToFile" -count=1

# Run document analysis and weight normalization tests
go test ./pkg/validate/... -v -run "TestAnalyzeDocument|TestSuggestWeights_Normalization" -count=1

# Run confidence scoring tests
go test ./pkg/validate/... -v -run TestComputeConfidence -count=1

# CLI: Suggest a profile from document analysis
regula validate --source testdata/gdpr.txt --suggest-profile

# CLI: Suggest profile with JSON output
regula validate --source testdata/gdpr.txt --suggest-profile --format json

# CLI: Suggest profile with YAML output
regula validate --source testdata/gdpr.txt --suggest-profile --format yaml

# CLI: Generate and save profile to YAML file
regula validate --source testdata/gdpr.txt --generate-profile gdpr-custom.yaml

# CLI: Load a custom profile for validation
regula validate --source testdata/gdpr.txt --load-profile gdpr-custom.yaml
```

### Test Coverage

| Test | What it verifies |
|------|------------------|
| `TestSuggestProfile_LargeDocument` | Correct expected counts, weight normalization, high confidence |
| `TestSuggestProfile_SmallDocument` | Adjusted counts for small documents, lower confidence |
| `TestSuggestProfile_NoDefinitions` | Definition weight reduced below default (0.20) |
| `TestSuggestProfile_HeavyReferences` | Reference weight boosted above default (0.25) |
| `TestSuggestProfile_NoSemantics` | Semantic weight reduced below default (0.20) |
| `TestSuggestProfile_Reasoning` | Reasoning entries for expected.articles, .definitions, .chapters |
| `TestSuggestProfile_KnownRightsAndObligations` | Rights/obligation types classified from semantics |
| `TestProfileSuggestion_ToYAML` | YAML output contains all expected fields |
| `TestProfileSuggestion_ToYAML_Roundtrip` | Serialize → deserialize preserves values |
| `TestProfileFromYAML` | Parse YAML into ValidationProfile with correct values |
| `TestProfileFromYAML_NoRights` | Empty slices (not nil) when no rights/obligations |
| `TestProfileFromYAML_Invalid` | Malformed YAML returns error |
| `TestLoadProfileFromFile` | Read profile from temp YAML file |
| `TestLoadProfileFromFile_NotFound` | Nonexistent file returns error |
| `TestSaveProfileToFile` | Write to file and verify round-trip |
| `TestAnalyzeDocument` | Article/chapter/definition/reference/rights counts, densities |
| `TestAnalyzeDocument_Empty` | Zero-value document produces zero densities |
| `TestSuggestWeights_Normalization` | Weights sum to 1.0 across 4 document scenarios |
| `TestComputeConfidence` | Confidence varies: empty (0.0-0.1), articles-only (0.2-0.4), complete (0.8-1.0) |
| `TestProfileSuggestion_String` | Human-readable output contains key sections |
| `TestProfileSuggestion_ToJSON` | JSON output contains profile, reasoning, confidence |
| `TestRoundToDecimals` | Rounding utility correctness |

### Draft Legislation Parser Tests

Test the draft bill parser, amendment pattern recognizer, and legislation diff engine (`pkg/draft/`):

```bash
# Run all draft legislation tests
go test ./pkg/draft/... -v -count=1

# Run parser tests only
go test ./pkg/draft/... -run "TestNewParser|TestParse|TestSection|TestShortTitle|TestBill|TestAmendments" -v

# Run amendment pattern recognition tests
go test ./pkg/draft/... -run "TestNewRecognizer|TestClassify|TestParseTarget|TestExtractAmendments" -v

# Run legislation diff tests
go test ./pkg/draft/... -run "TestComputeDiff|TestResolveAmendment|TestCountAffected|TestFindCross" -v

# Run with coverage
go test ./pkg/draft/... -coverprofile=draft-coverage.out -count=1
go tool cover -func=draft-coverage.out
```

#### Bill Parser Tests (`pkg/draft/parser_test.go`)

| Test | What it verifies |
|------|------------------|
| `TestNewParser` | Parser constructor compiles all regex patterns |
| `TestParseFullBill` | Full bill with metadata, sections, and amendment text |
| `TestParseMinimalBill` | Minimal bill with single section |
| `TestParseBillFromFile` | File loading round-trip via `ParseBillFromFile()` |
| `TestParseBillString` | String parsing via `ParseBill()` convenience function |
| `TestParseEmptyInput` | Graceful handling of empty input |
| `TestParseNoSections` | Header-only bill with no SEC. markers |
| `TestSectionBoundaries` | Multiple sections correctly split at SEC. boundaries |
| `TestShortTitleExtraction` | Short title extracted from "may be cited as" pattern |
| `TestShortTitleMissing` | Bill without short title handled gracefully |
| `TestBillStatistics` | `Statistics()` method returns correct aggregate counts |
| `TestBillString` | `String()` representation includes bill number and title |
| `TestBillStringWithoutCongress` | String representation when Congress field is empty |
| `TestAmendmentsInitializedEmpty` | Each section's Amendments slice initialized to `[]` (not nil) |
| `TestSectionNumberParsing` | Section numbers extracted from various SEC. header formats |

#### Amendment Pattern Recognition Tests (`pkg/draft/patterns_test.go`)

| Test | What it verifies |
|------|------------------|
| `TestNewRecognizer` | Recognizer constructor compiles all 15 regex patterns |
| `TestClassifyAmendmentType` | All 6 amendment types classified (11 subtests: strike-insert, dollar amounts, repeal, hereby repealed, add new section, add at end, add at end new subsection, redesignate paragraph, redesignate subsection, table of contents, no match) |
| `TestParseTargetReference` | USC target extraction (5 subtests: USC citation, USC with subsection, title comma format, title-of-the format, no reference) |
| `TestExtractAmendments_StrikeInsert` | Strike-and-insert pattern with target title, section, strike/insert text |
| `TestExtractAmendments_Repeal` | Section repeal and "hereby repealed" with subsection (2 subtests) |
| `TestExtractAmendments_AddNewSection` | New section insertion after existing section |
| `TestExtractAmendments_AddAtEnd` | Append content to end of existing section |
| `TestExtractAmendments_Redesignate` | Paragraph and subsection redesignation |
| `TestExtractAmendments_MultipleInOneSection` | Compound amendments within a single bill section |
| `TestExtractAmendments_NonAmendmentSection` | Non-amendment sections return empty results (4 subtests: definitions, effective date, short title, empty text) |
| `TestExtractAmendments_Integration` | Full integration with real bill sections from `testdata/drafts/hr1234.txt` (5 subtests) |

#### Legislation Diff Tests (`pkg/draft/diff_test.go`)

Tests use a mock library with seeded triple stores (42 triples for USC Title 15 articles 6502, 6503, 6505 with cross-references).

| Test | What it verifies |
|------|------------------|
| `TestComputeDiff_SingleModification` | Strike-and-insert classified as Modified, existing text and affected triples populated |
| `TestComputeDiff_Repeal` | Repeal classified as Removed, cross-references identified |
| `TestComputeDiff_AddNewSection` | New section classified as Added, proposed text populated |
| `TestComputeDiff_MultipleAmendments` | Mixed amendment types in one bill correctly separated into Added/Modified/Removed |
| `TestComputeDiff_UnresolvedTarget` | Amendment targeting non-existent section collected in `UnresolvedTargets` |
| `TestResolveAmendmentTarget` | Target resolution to knowledge graph URIs (5 subtests: basic section, subsection, missing title, missing section, document not in library) |
| `TestCountAffectedTriples` | Counts triples where target URI appears as subject or object |
| `TestFindCrossReferences` | Bidirectional cross-reference lookup via `reg:references` and `reg:referencedBy` |

#### CLI Testing

```bash
# Build and test CLI commands
go build -o regula ./cmd/regula

# Parse and display bill structure (table format)
regula draft ingest --bill testdata/drafts/hr1234.txt

# Parse and display bill structure (JSON format)
regula draft ingest --bill testdata/drafts/hr1234.txt --format json

# Compute diff against knowledge graph (table format)
regula draft diff --bill testdata/drafts/hr1234.txt --path .regula

# Compute diff (JSON format)
regula draft diff --bill testdata/drafts/hr1234.txt --format json

# Compute diff (CSV format)
regula draft diff --bill testdata/drafts/hr1234.txt --format csv

# Error handling: missing --bill flag
regula draft ingest   # exits with error
regula draft diff     # exits with error
```

#### Test Coverage

Overall package coverage: **87.3%** (34 tests + 41 subtests)

| File | Function | Coverage |
|------|----------|----------|
| `parser.go` | `NewParser` | 100% |
| `parser.go` | `Parse` | 88.9% |
| `parser.go` | `ParseBill` | 100% |
| `parser.go` | `ParseBillFromFile` | 80.0% |
| `patterns.go` | `NewRecognizer` | 100% |
| `patterns.go` | `ClassifyAmendmentType` | 100% |
| `patterns.go` | `ParseTargetReference` | 89.5% |
| `patterns.go` | `ExtractAmendments` | 89.5% |
| `diff.go` | `ComputeDiff` | 82.8% |
| `diff.go` | `ResolveAmendmentTarget` | 92.3% |
| `diff.go` | `CountAffectedTriples` | 100% |
| `diff.go` | `FindCrossReferences` | 100% |
| `types.go` | `Statistics` | 100% |
| `types.go` | `String` | 100% |

#### Test Data

| File | Description |
|------|-------------|
| `testdata/drafts/hr1234.txt` | H.R. 1234 — Children's Online Privacy Protection Modernization Act (5 sections, 4 amendments targeting Title 15) |
| `testdata/drafts/s456_minimal.txt` | S. 456 — AI small business study bill (1 section, 0 amendments) |

### E2E Tests

The E2E test script validates the complete MVP functionality:

```bash
# Build the binary first
go build -o regula ./cmd/regula

# Run E2E tests
./scripts/e2e-test.sh
```

## E2E Test Coverage

The E2E test script (`scripts/e2e-test.sh`) validates 25 criteria across three categories:

### Core Pipeline Tests (8 tests)

| Test | Command | Validation |
|------|---------|------------|
| Repository initialization | `regula init` | Creates project directories |
| GDPR ingestion | `regula ingest` | Parses document successfully |
| Chapter listing | `regula query --template chapters` | Lists chapters |
| Definition extraction | `regula query --template definitions` | Finds definitions |
| Article content retrieval | `regula query` | Returns article content |
| Selective article export | `regula export` | Exports specified articles |
| Scenario listing | `regula match --list-scenarios` | Lists predefined scenarios |
| Scenario matching | `regula match --scenario` | Matches provisions |

### Threshold Validation Tests (8 tests)

| Metric | Threshold | Description |
|--------|-----------|-------------|
| Article count | ≥ 50 | Parsed articles from GDPR |
| Definition count | ≥ 20 | Extracted defined terms |
| Reference count | ≥ 100 | Identified cross-references |
| Reference resolution rate | ≥ 80% | References resolved to URIs |
| Definition resolution rate | ≥ 80% | Definitions linked to usages |
| Graph triple count | ≥ 500 | RDF triples in knowledge graph |
| Impact analysis | ≥ 10 | Affected provisions for Art 17 |
| Scenario matching | ≥ 5 | Matched provisions per scenario |

### Output Format Tests (9 tests)

| Test | Validation |
|------|------------|
| JSON parsing validity | Output parses as valid JSON |
| JSON output structure | Contains expected fields |
| JSON array format | Arrays properly formatted |
| Markdown table generation | Table format correct |
| Export plain text format | Text output readable |
| Export contains article text | Full article content included |
| Impact JSON format | Impact results valid JSON |
| Match JSON format | Match results valid JSON |
| Summary output format | Summary statistics present |

## Running Tests in CI

The GitHub Actions workflow (`.github/workflows/e2e-test.yml`) runs tests automatically:

```yaml
- name: Run unit tests
  run: go test ./... -v

- name: Run E2E tests
  run: ./scripts/e2e-test.sh
  env:
    REGULA_BIN: regula
    GDPR_FILE: testdata/gdpr.txt
    CI: true
```

## Test Data

### GDPR Test File

The primary test data is `testdata/gdpr.txt`, containing the full GDPR text with:

- 11 chapters
- 99 articles
- 173 recitals
- 26 defined terms
- 255+ cross-references

### CCPA Test File

US-style regulation test data at `testdata/ccpa.txt`:

- 6 chapters
- 21 articles (sections)
- 15 defined terms
- California Civil Code format (Section 1798.xxx)

### VCDPA Test File

Virginia Consumer Data Protection Act at `testdata/vcdpa.txt`:

- 7 chapters
- 12 sections (59.1-575 through 59.1-585)
- 22 defined terms
- Virginia Code format (Section 59.1-xxx)
- Heavy external law references (HIPAA, GLBA, FCRA, FERPA, etc.)

**Testing VCDPA:**

```bash
# Validate VCDPA document
regula validate --source testdata/vcdpa.txt

# Auto-detects VCDPA profile based on document content
# Expected output:
#   Profile: VCDPA
#   Rights found: 12 (in 3 articles)
#   Known VCDPA rights: 6/6
#   Definitions: 20 defined terms
#   Structure: 93.9%
```

**Note on VCDPA Reference Resolution:**

VCDPA contains many references to external federal laws (HIPAA, GLBA, COPPA, etc.)
that are written in short form without full U.S.C. citations (e.g., "Section 1320d"
instead of "42 U.S.C. § 1320d"). These may show as unresolved internal references.
The semantic extraction and definition coverage are more reliable metrics for
VCDPA validation.

### Corpus Golden File Testing

The test corpus at `testdata/corpus/` provides comprehensive golden file validation across 15 jurisdictions and 5 format families (EU Regulation, EU Directive, US State/Federal, UK Primary/Secondary, Generic/International).

**Run corpus tests:**

```bash
# Validate parser output against golden files
go test ./pkg/extract/... -run TestCorpusGoldenFiles -v

# Regenerate golden files after parser changes
go test ./pkg/extract/... -run TestCorpusGoldenFiles -update -v

# Validate manifest integrity (source files exist, no duplicate IDs, minimum 10 entries)
go test ./pkg/extract/... -run TestCorpusManifestIntegrity -v
```

**Corpus structure:**

```
testdata/corpus/
  manifest.json          # Corpus metadata and entry list
  SOURCES.md             # Provenance documentation
  eu-gdpr/expected.json  # Golden file per jurisdiction
  eu-eprivacy/source.txt # Source document (5 new documents)
  ...
```

**Adding a new jurisdiction:**

1. Add the source document to `testdata/corpus/<id>/source.txt` (or reference an existing `testdata/*.txt` file)
2. Add an entry to `testdata/corpus/manifest.json` with metadata
3. Generate the golden file: `go test ./pkg/extract/... -run TestCorpusGoldenFiles -update -v`
4. Verify: `go test ./pkg/extract/... -run TestCorpusGoldenFiles -v`

**Current corpus entries (15 jurisdictions):**

| ID | Jurisdiction | Format | Chapters | Articles | Definitions | Recitals |
|----|-------------|--------|----------|----------|-------------|----------|
| `eu-gdpr` | EU | EU Regulation | 11 | 99 | 26 | 173 |
| `eu-eprivacy` | EU | EU Directive | 5 | 21 | 8 | 8 |
| `us-ca-ccpa` | US-CA | US State | 6 | 21 | 15 | 0 |
| `us-va-vcdpa` | US-VA | US State | 7 | 11 | 20 | 0 |
| `us-co-cpa` | US-CO | US State | 1 | 10 | 18 | 0 |
| `us-ct-ctdpa` | US-CT | US State | 1 | 9 | 17 | 0 |
| `us-ut-ucpa` | US-UT | US State | 4 | 7 | 13 | 0 |
| `us-tx-tdpsa` | US-TX | US State | 1 | 7 | 18 | 0 |
| `us-ia-icdpa` | US-IA | US State | 1 | 7 | 15 | 0 |
| `us-hipaa` | US-Federal | US Federal | 2 | 0 | 0 | 0 |
| `us-hipaa-cfr` | US-Federal | US CFR | 2 | 8 | 0 | 0 |
| `gb-dpa2018` | GB | UK Primary | 9 | 21 | 4 | 0 |
| `gb-si-example` | GB | UK SI | 4 | 9 | 5 | 0 |
| `intl-uncitral` | INTL | Generic | 3 | 15 | 6 | 0 |
| `au-privacy` | AU | Generic | 1 | 16 | 0 | 0 |

### Expected Output Files

Expected outputs for comparison testing:

| File | Description |
|------|-------------|
| `testdata/art17-impact-expected.json` | Expected impact analysis for Art 17 |
| `testdata/corpus/*/expected.json` | Golden file corpus (15 jurisdictions) |

## Writing New Tests

### Unit Test Pattern

```go
func TestFeatureName(t *testing.T) {
    // Setup
    ts := store.NewTripleStore()

    // Execute
    result := functionUnderTest(input)

    // Assert
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

### Integration Test Pattern

```go
func TestGDPRIntegration(t *testing.T) {
    file, err := os.Open("../../testdata/gdpr.txt")
    if err != nil {
        t.Skipf("Skipping GDPR test: %v", err)
    }
    defer file.Close()

    parser := extract.NewParser()
    doc, err := parser.Parse(file)
    if err != nil {
        t.Fatalf("Failed to parse: %v", err)
    }

    // Test assertions...
}
```

### Table-Driven Tests

```go
func TestMultipleCases(t *testing.T) {
    cases := []struct {
        name     string
        input    string
        expected int
    }{
        {"empty", "", 0},
        {"single", "Art 1", 1},
        {"multiple", "Art 1 and Art 2", 2},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            result := countReferences(tc.input)
            if result != tc.expected {
                t.Errorf("Expected %d, got %d", tc.expected, result)
            }
        })
    }
}
```

## Test Coverage

Generate coverage report:

```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## Debugging Tests

### Verbose Output

```bash
go test ./pkg/simulate/... -v -run TestProvisionMatcher
```

### Test with Logging

```go
func TestWithLogging(t *testing.T) {
    t.Logf("Debug info: %v", someValue)
    // ...
}
```

### Skip Long-Running Tests

```bash
go test ./... -short
```

In test code:
```go
func TestLongRunning(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping long-running test")
    }
    // ...
}
```

## Fuzz Testing

Fuzz testing uses Go 1.18+ native fuzzing to test parsers and extractors with randomly generated inputs.

### Running Fuzz Tests

```bash
# Run all fuzz tests briefly (5 seconds each)
go test ./pkg/extract/... -fuzz=FuzzParser -fuzztime=5s
go test ./pkg/extract/... -fuzz=FuzzReferenceExtractor -fuzztime=5s
go test ./pkg/extract/... -fuzz=FuzzDefinitionExtractor -fuzztime=5s
go test ./pkg/extract/... -fuzz=FuzzSemanticExtractor -fuzztime=5s
go test ./pkg/citation/... -fuzz=FuzzEUCitationParser -fuzztime=5s
go test ./pkg/citation/... -fuzz=FuzzBluebookParser -fuzztime=5s
go test ./pkg/citation/... -fuzz=FuzzCitationRegistry -fuzztime=5s
go test ./pkg/query/... -fuzz=FuzzParseQuery -fuzztime=5s
go test ./pkg/query/... -fuzz=FuzzExecuteQuery -fuzztime=5s
go test ./pkg/query/... -fuzz=FuzzFilterEvaluation -fuzztime=5s
```

### Extended Fuzz Testing

For longer fuzz runs to find deeper edge cases:

```bash
# 30 seconds per target (recommended for PR validation)
go test ./pkg/extract/... -fuzz=FuzzParser -fuzztime=30s

# 5 minutes per target (recommended for release testing)
go test ./pkg/extract/... -fuzz=FuzzParser -fuzztime=5m

# Unlimited fuzzing (run until stopped with Ctrl+C)
go test ./pkg/extract/... -fuzz=FuzzParser
```

### Fuzz Test Corpus

Fuzz tests automatically save interesting inputs to `testdata/fuzz/<FuzzTestName>/`:

```bash
# View discovered corpus entries
ls pkg/extract/testdata/fuzz/FuzzParser/

# Clean corpus (reset fuzzing state)
rm -rf pkg/extract/testdata/fuzz/
```

### Available Fuzz Targets

| Package | Target | Tests |
|---------|--------|-------|
| `pkg/extract` | `FuzzParser` | Document parser with arbitrary text |
| `pkg/extract` | `FuzzReferenceExtractor` | Reference extractor with arbitrary text |
| `pkg/extract` | `FuzzDefinitionExtractor` | Definition extractor with arbitrary text |
| `pkg/extract` | `FuzzSemanticExtractor` | Semantic annotation extractor |
| `pkg/citation` | `FuzzEUCitationParser` | EU citation parser (Regulation, Directive, etc.) |
| `pkg/citation` | `FuzzBluebookParser` | US citation parser (U.S.C., C.F.R., etc.) |
| `pkg/citation` | `FuzzOSCOLAParser` | UK/Commonwealth citation parser (Acts, SIs, cases, ECHR) |
| `pkg/citation` | `FuzzCitationRegistry` | Combined citation registry (EU + US + UK) |
| `pkg/query` | `FuzzParseQuery` | SPARQL query parser |
| `pkg/query` | `FuzzExecuteQuery` | SPARQL query executor |
| `pkg/query` | `FuzzFilterEvaluation` | SPARQL FILTER expression evaluator |

### Handling Crashes

If a fuzz test finds a crash:

1. The crash input is saved to `testdata/fuzz/<FuzzTestName>/`
2. Fix the issue in the code
3. Run the fuzz test again to verify the fix
4. Add a regression test if appropriate

```bash
# Reproduce a specific crash
go test ./pkg/extract/... -run=FuzzParser/crash_input_filename
```

### CI Integration

Brief fuzz tests (10s each) run automatically on all PRs as part of the E2E test workflow.

## Performance Testing

### Benchmark Tests

```go
func BenchmarkIngest(b *testing.B) {
    for i := 0; i < b.N; i++ {
        // Ingestion code
    }
}
```

Run benchmarks:
```bash
go test ./pkg/extract/... -bench=. -benchmem
```

### Query Timing

Use `--timing` flag to measure query performance:

```bash
regula query --source testdata/gdpr.txt --template articles --timing
```

## Recursive Document Fetching (`pkg/fetch/`)

The recursive fetch package provides federated graph building by fetching metadata for external references.

### Unit Tests

```bash
# URN-to-URL mapping tests
go test ./pkg/fetch/... -v -run TestMapURN -count=1
go test ./pkg/fetch/... -v -run TestParseEUDocURN -count=1

# Disk cache tests
go test ./pkg/fetch/... -v -run TestDiskCache -count=1

# Recursive fetcher tests (uses mock HTTP client)
go test ./pkg/fetch/... -v -run TestRecursiveFetcher -count=1

# Report formatting tests
go test ./pkg/fetch/... -v -run TestFetchReport -count=1

# All fetch tests
go test ./pkg/fetch/... -v -short -count=1
```

### Integration Tests

Integration tests make real HTTP requests to EUR-Lex and are skipped in short mode:

```bash
# Run with real network calls (not in short mode)
go test ./pkg/fetch/... -v -run TestIntegration -count=1
```

### CLI Usage

```bash
# Dry-run: see what would be fetched without network calls
regula ingest --source gdpr.txt --fetch-refs --dry-run

# Fetch with defaults (max-depth=2, max-documents=10)
regula ingest --source gdpr.txt --fetch-refs

# Limit fetch scope
regula ingest --source gdpr.txt --fetch-refs --max-depth 1 --max-documents 5

# Restrict to specific domains
regula ingest --source gdpr.txt --fetch-refs --allowed-domains data.europa.eu

# Enable disk caching for cross-session persistence
regula ingest --source gdpr.txt --fetch-refs --cache-dir ~/.regula/cache
```

### Test Coverage

| Test | What it verifies |
|---|---|
| `TestMapURN_EURegulation` | Regulation URN → ELI URL |
| `TestMapURN_EUDirective` | Directive URN → ELI URL |
| `TestMapURN_EUDecision` | Decision URN → ELI URL |
| `TestMapURN_Treaty` | Treaty URNs return error |
| `TestMapURN_USSource` | US USC/CFR URNs resolve; other US subtypes return error |
| `TestDiskCache_*` | Set/Get, TTL expiration, overwrite, corruption |
| `TestRecursiveFetcher_Fetch_*` | Success, depth/doc limits, domain filtering |
| `TestRecursiveFetcher_Fetch_CacheHit` | Disk cache prevents redundant HTTP calls |
| `TestRecursiveFetcher_Fetch_FailureGraceful` | Pipeline continues on individual failures |
| `TestRecursiveFetcher_Plan_DryRun` | No HTTP calls in dry-run mode |
| `TestRecursiveFetcher_FederatedTriples` | Cross-document RDF triples generated |

## Cross-Legislation Analysis Tests

Test the cross-reference analysis engine:

```bash
# Run all cross-reference tests
go test ./pkg/analysis/... -v -run "TestCrossRef|TestCompare|TestAnalyzeExternal" -count=1

# Run specific test categories
go test ./pkg/analysis/... -v -run "TestCompareDocuments" -count=1     # pair-wise comparison
go test ./pkg/analysis/... -v -run "TestAnalyzeExternalRefs" -count=1   # external ref analysis
go test ./pkg/analysis/... -v -run "TestAnalyze_Multi" -count=1         # multi-document analysis
go test ./pkg/analysis/... -v -run "TestNormalize" -count=1             # normalization helpers
go test ./pkg/analysis/... -v -run "ToDOT" -count=1                    # DOT graph output
```

### CLI Integration Tests

Test the compare and refs commands end-to-end:

```bash
# Compare two documents
regula compare --sources testdata/gdpr.txt,testdata/ccpa.txt --format table

# Compare three documents
regula compare --sources testdata/gdpr.txt,testdata/ccpa.txt,testdata/eu-ai-act.txt --format table

# Export comparison as JSON
regula compare --sources testdata/gdpr.txt,testdata/ccpa.txt --format json

# Export comparison as DOT graph
regula compare --sources testdata/gdpr.txt,testdata/eu-ai-act.txt --format dot --output comparison.dot

# Analyze all references in a document
regula refs --source testdata/gdpr.txt

# Analyze external references only
regula refs --source testdata/eu-ai-act.txt --external-only

# Export external ref analysis as JSON
regula refs --source testdata/gdpr.txt --external-only --format json

# Verify enhanced summary export includes external refs
regula export --source testdata/gdpr.txt --format summary
```

### Cross-Reference Test Matrix

| Test | What it verifies |
|---|---|
| `TestNewCrossRefAnalyzer` | Constructor, empty state |
| `TestCrossRefAnalyzer_AddDocument` | Document registration |
| `TestAnalyzeExternalRefs_SingleDoc` | External ref clustering and frequency |
| `TestAnalyzeExternalRefs_Empty` | No external refs |
| `TestAnalyzeExternalRefs_NonexistentDoc` | Nonexistent document handling |
| `TestCompareDocuments_DefinitionOverlap` | Shared defined terms via normalized matching |
| `TestCompareDocuments_RightsOverlap` | Shared rights across documents |
| `TestCompareDocuments_ObligationOverlap` | Shared obligations across documents |
| `TestCompareDocuments_ExternalRefOverlap` | Common external reference targets |
| `TestAnalyze_MultiDocument` | Full 3-document analysis |
| `TestCrossRefResult_String` | Human-readable output formatting |
| `TestCrossRefResult_FormatTable` | Table comparison formatting |
| `TestCrossRefResult_ToJSON` | JSON serialization |
| `TestCrossRefResult_ToDOT` | Graphviz DOT generation |
| `TestComparisonResult_String` | Pair-wise comparison output |
| `TestComparisonResult_ToDOT` | Pair-wise DOT generation |
| `TestExternalRefReport_String` | External ref report formatting |
| `TestNormalizeExternalRef` | External ref normalization |
| `TestNormalizeConceptName` | Concept name normalization |
| `TestSanitizeDOTID` | DOT ID sanitization |
| `TestBuildDocumentSummary` | Document summary construction |
| `TestBuildDocumentSummary_Nonexistent` | Nonexistent document summary |

### Test Data Files

| File | Type | Articles | Definitions | Cross-refs |
|---|---|---|---|---|
| `testdata/gdpr.txt` | EU Regulation | 99 | 26 | 179 |
| `testdata/ccpa.txt` | US State Law | 21 | 15 | 10 |
| `testdata/eu-ai-act.txt` | EU Regulation | 15 | 57 | 51 |
| `testdata/eu-dsa.txt` | EU Regulation | 11 | 15 | 46 |
| `testdata/us-coppa.txt` | US Federal Law | 10 | 7 | 4 |

## Legislation Library Tests

The `pkg/library` package has tests covering the persistent legislation library engine:

```bash
# Run all library tests
go test ./pkg/library/... -v -count=1

# Run specific test suites
go test ./pkg/library/... -run TestSerialize -v      # Serialization round-trip
go test ./pkg/library/... -run TestIngest -v          # Ingestion pipeline
go test ./pkg/library/... -run TestAdd -v             # Document add/idempotency
go test ./pkg/library/... -run TestLoad -v            # Triple store loading
go test ./pkg/library/... -run TestSeed -v            # Corpus seeding
go test ./pkg/library/... -run TestPersistence -v     # Open/close persistence
```

### Library CLI End-to-End Testing

```bash
# Build the binary
go build -o regula ./cmd/regula

# Initialize a library
regula library init --path /tmp/test-lib

# Seed with all testdata (18 documents)
regula library seed --testdata-dir testdata --path /tmp/test-lib

# List documents
regula library list --path /tmp/test-lib

# Filter by jurisdiction
regula library list --jurisdiction EU --path /tmp/test-lib

# Show library statistics
regula library status --path /tmp/test-lib

# Query across documents
regula library query --template rights --documents eu-gdpr,us-ca-ccpa --path /tmp/test-lib

# Add a single document
regula library add --source testdata/gdpr.txt --id eu-gdpr --jurisdiction EU --path /tmp/test-lib

# View source text
regula library source eu-gdpr --path /tmp/test-lib | head -5

# Export graph summary
regula library export --document eu-gdpr --format summary --path /tmp/test-lib

# Remove a document
regula library remove eu-gdpr --path /tmp/test-lib

# Clean up
rm -rf /tmp/test-lib
```

## Legislation Crawler Tests

The `pkg/crawler` package provides a BFS tree-walking crawler that discovers and ingests US legislation by following cross-references.

### Unit Tests

```bash
# Run all crawler tests
go test ./pkg/crawler/... -v -count=1

# Run specific test suites
go test ./pkg/crawler/... -run TestResolve -v         # Citation-to-URL resolution
go test ./pkg/crawler/... -run TestContentFetcher -v   # HTTP fetching + rate limiting
go test ./pkg/crawler/... -run TestCrawlState -v       # Frontier queue + visited set
go test ./pkg/crawler/... -run TestCrawler -v           # BFS engine integration
go test ./pkg/crawler/... -run TestExtractText -v       # HTML-to-text conversion
go test ./pkg/crawler/... -run TestProvenance -v        # RDF provenance tracking
```

### Citation Resolution Tests

```bash
# Test USC citation resolution (uscode.house.gov)
go test ./pkg/crawler/... -run "TestResolve_USCCitation|TestResolve_USCSection" -v

# Test CFR citation resolution (ecfr.gov)
go test ./pkg/crawler/... -run "TestResolve_CFRCitation|TestResolve_CFRPart" -v

# Test state code resolution (CA, VA, CO, CT, TX)
go test ./pkg/crawler/... -run "TestResolve_CaliforniaCode|TestResolve_VirginiaCode" -v

# Test Public Law and LII fallback
go test ./pkg/crawler/... -run "TestResolve_PublicLaw|TestResolve_LIIFallback" -v

# Test URN resolution (extends pkg/fetch URNMapper)
go test ./pkg/crawler/... -run TestResolveURN -v
go test ./pkg/fetch/... -run TestMapURN_USSource -v
```

### Fetcher Tests

```bash
# Test HTTP fetching with mock server
go test ./pkg/crawler/... -run TestContentFetcher -v

# Test HTML-to-text extraction
go test ./pkg/crawler/... -run "TestExtractTextFromHTML" -v

# Test per-domain rate limiting
go test ./pkg/crawler/... -run TestContentFetcher_RateLimit -v

# Test error handling (404, empty URL, etc.)
go test ./pkg/crawler/... -run "TestContentFetcher_HTTPError|TestContentFetcher_EmptyURL" -v
```

### State Persistence Tests

```bash
# Test frontier enqueue/dequeue
go test ./pkg/crawler/... -run TestCrawlStateEnqueueDequeue -v

# Test visited set and deduplication
go test ./pkg/crawler/... -run TestCrawlStateVisited -v

# Test save/load round-trip
go test ./pkg/crawler/... -run TestCrawlStateSaveAndLoad -v

# Test limit enforcement
go test ./pkg/crawler/... -run TestCrawlStateWithinLimits -v
```

### Engine Integration Tests

Tests use `httptest.Server` mock HTTP servers to verify BFS expansion without real network calls:

```bash
# Test crawl from URL seed
go test ./pkg/crawler/... -run TestCrawlerFromURL -v

# Test depth and document limits
go test ./pkg/crawler/... -run "TestCrawlerDepthLimit|TestCrawlerDocumentLimit" -v

# Test URL deduplication
go test ./pkg/crawler/... -run TestCrawlerDeduplication -v

# Test dry-run planning mode
go test ./pkg/crawler/... -run TestCrawlerDryRun -v

# Test report formatting (table + JSON)
go test ./pkg/crawler/... -run TestCrawlerReportFormat -v

# Test provenance tracking
go test ./pkg/crawler/... -run "TestCrawlerProvenance|TestCrawlerProvenanceFailure" -v

# Test error paths
go test ./pkg/crawler/... -run "TestCrawlerFromCitationNotFound|TestCrawlerFromDocumentNotFound" -v
```

### CLI End-to-End Testing

```bash
# Build the binary
go build -o regula ./cmd/regula

# Initialize a library and seed with test data
regula library init --path /tmp/test-crawl
regula library seed --testdata-dir testdata --path /tmp/test-crawl

# Dry-run: plan crawl from an existing document (no network calls)
regula crawl --seed us-ca-ccpa --dry-run --max-depth 2 --path /tmp/test-crawl

# Dry-run from a citation
regula crawl --citation "42 U.S.C. § 1320d" --dry-run --max-depth 1 --path /tmp/test-crawl

# Dry-run with JSON output
regula crawl --seed us-ca-ccpa --dry-run --max-depth 1 --max-documents 3 --format json --path /tmp/test-crawl

# Live crawl from seed (hits real servers, rate-limited)
regula crawl --seed us-ca-ccpa --max-depth 1 --max-documents 5 --path /tmp/test-crawl

# Crawl from a URL
regula crawl --url https://uscode.house.gov/view.xhtml?req=granuleid:USC-prelim-title42-section1320d --max-depth 1 --path /tmp/test-crawl

# Resume an interrupted crawl
regula crawl --resume --path /tmp/test-crawl

# Restrict to specific domains
regula crawl --seed us-ca-ccpa --allowed-domains uscode.house.gov,ecfr.gov --max-depth 2 --path /tmp/test-crawl

# Check what was discovered
regula library list --path /tmp/test-crawl
regula library status --path /tmp/test-crawl

# Clean up
rm -rf /tmp/test-crawl
```

### Crawler Test Coverage

| Test | What it verifies |
|---|---|
| `TestResolve_USCCitation` | USC citation → uscode.house.gov URL |
| `TestResolve_CFRCitation` | CFR citation → ecfr.gov URL |
| `TestResolve_PublicLaw` | Public Law → congress.gov URL |
| `TestResolve_CaliforniaCode` | California code → leginfo.legislature.ca.gov URL |
| `TestResolve_VirginiaCode` | Virginia code → law.lis.virginia.gov URL |
| `TestResolve_LIIFallback` | LII URL passthrough resolution |
| `TestResolve_UnrecognizedCitation` | Unknown citations return error |
| `TestResolveURN_USC` | URN-based USC resolution |
| `TestResolveURN_CFR` | URN-based CFR resolution |
| `TestContentFetcher_Success` | HTML fetch + text extraction |
| `TestContentFetcher_PlainText` | Plain text fetch passthrough |
| `TestContentFetcher_HTTPError` | 4xx/5xx error handling |
| `TestContentFetcher_RateLimit` | Per-domain throttling |
| `TestExtractTextFromHTML_*` | Body extraction, tag stripping, entity decoding |
| `TestCrawlStateEnqueueDequeue` | FIFO frontier queue ordering |
| `TestCrawlStateVisited` | Visited set prevents re-enqueue |
| `TestCrawlStateSaveAndLoad` | JSON state round-trip persistence |
| `TestCrawlStateWithinLimits` | Document limit enforcement |
| `TestCrawlerFromURL` | BFS expansion from URL seed |
| `TestCrawlerDepthLimit` | MaxDepth enforcement |
| `TestCrawlerDocumentLimit` | MaxDocuments enforcement |
| `TestCrawlerDeduplication` | Same URL ingested once |
| `TestCrawlerDryRun` | Plan mode (no fetching) |
| `TestCrawlerReportFormat` | Table + JSON output formatting |
| `TestCrawlerProvenance` | Discovery chain RDF triples |
| `TestCrawlerProvenanceFailure` | Failure recording in triples |
| `TestMapURN_USSource` | USC/CFR URN mapping in fetch package |

## Bulk Download & Ingest Tests

### Unit Tests

Run all bulk package tests:

```bash
go test ./pkg/bulk/... -v -count=1
```

Test files cover:
- `types_test.go` — Source resolution, config defaults
- `manifest_test.go` — JSON manifest save/load, record tracking
- `downloader_test.go` — HTTP download with httptest, ZIP/tar.gz extraction, resume, path traversal protection
- `uscode_test.go` — 54-title dataset listing, download to local path
- `cfr_test.go` — 50-title listing, configurable year, download
- `california_test.go` — 30-code listing, HTML text extraction, branch URL parsing
- `archive_test.go` — Best file selection, jurisdiction extraction from IA identifiers
- `xmlparse_test.go` — USLM and CFR XML parsing, plaintext conversion
- `ingest_test.go` — Document ID derivation, AddOptions generation, title filtering
- `report_test.go` — Byte formatting, dataset tables, ingest/status reports

| Test | Purpose |
|------|---------|
| `TestManifestSaveAndLoad` | JSON manifest round-trip persistence |
| `TestDownloadFile` | HTTP GET streaming with progress callbacks |
| `TestDownloadFileSkipsExisting` | Resume support (skip existing files) |
| `TestExtractZIP` | ZIP extraction with nested directories |
| `TestExtractZIPPathTraversal` | Path traversal attack prevention |
| `TestExtractTarGZ` | tar.gz extraction |
| `TestParseUSLMXML` | USLM XML struct deserialization |
| `TestParseCFRXML` | CFR XML struct deserialization |
| `TestUSLMToPlaintext` | USLM → plaintext with chapters/sections |
| `TestCFRToPlaintext` | CFR → plaintext with parts/subparts |
| `TestExtractCaliforniaText` | HTML tag stripping and entity decoding |
| `TestExtractBranchURLs` | TOC link extraction with deduplication |
| `TestFindBestArchiveFile` | IA file selection priority (tar.gz > zip > xml > txt) |
| `TestExtractJurisdictionFromID` | State code extraction from IA identifiers |
| `TestDeriveDocumentID` | Source-specific document ID mapping |
| `TestMatchesTitleFilter` | Case-insensitive title filtering |

### CLI Manual Testing

```bash
# List datasets
regula bulk list uscode
regula bulk list cfr
regula bulk list california

# Dry-run download (no network requests)
regula bulk download uscode --titles 04 --dry-run
regula bulk download cfr --titles 1 --dry-run

# Download a small title for local testing
regula bulk download uscode --titles 04

# Check download status
regula bulk status

# Ingest downloaded files (dry run)
regula bulk ingest --source uscode --titles 04 --dry-run

# Ingest downloaded files
regula bulk ingest --source uscode --titles 04
```

## SPARQL Aggregate Query Tests

### Unit Tests

Run aggregate query tests:

```bash
go test ./pkg/query/... -v -count=1 -run Aggregate
```

The aggregate query system supports COUNT, SUM, AVG, MIN, MAX with GROUP BY, HAVING, and COUNT(DISTINCT). Tests are in `pkg/query/aggregate_test.go`.

#### Parser Tests

| Test | Purpose |
|------|---------|
| `TestParseQuery_AggregateCountGroupBy` | COUNT aggregate with GROUP BY clause |
| `TestParseQuery_MultipleAggregates` | Multiple aggregates (COUNT + SUM) in one query |
| `TestParseQuery_CountDistinct` | COUNT(DISTINCT) syntax parsing |
| `TestParseQuery_AggregateWithHaving` | GROUP BY with HAVING filter clause |
| `TestParseQuery_AggregateNoGroupBy` | Scalar aggregates without GROUP BY |
| `TestParseQuery_NonAggregateUnchanged` | Non-aggregate queries unaffected |
| `TestParseQuery_AggregateString` | String representation of aggregate queries |
| `TestParseQuery_AggregateValidation` | Validation errors for invalid aggregates |

#### Executor Tests

| Test | Purpose |
|------|---------|
| `TestExecutor_CountGroupBy` | COUNT with GROUP BY, verifies DESC ordering |
| `TestExecutor_CountNoGroupBy` | Scalar COUNT returning single result |
| `TestExecutor_AggregateOrderByDesc` | Descending order for aggregate results |
| `TestExecutor_SumAggregate` | SUM aggregate total calculations |
| `TestExecutor_AvgAggregate` | AVG aggregate average calculations |
| `TestExecutor_MinMaxAggregate` | MIN and MAX in one query |
| `TestExecutor_CountDistinct` | COUNT(DISTINCT) unique counting |
| `TestExecutor_Having` | GROUP BY with HAVING filter |
| `TestExecutor_AggregateBackwardCompat` | Non-aggregate queries still work |
| `TestExecutor_AggregateWithLimit` | LIMIT applied to aggregate results |
| `TestExecutor_AggregateWithOffset` | OFFSET applied to aggregate results |

#### Integration and Benchmark Tests

| Test | Purpose |
|------|---------|
| `TestGDPRQuery_ArticlesPerChapter` | Articles per chapter with real GDPR data |
| `TestGDPRQuery_TotalArticleCount` | Total article count across GDPR dataset |
| `TestCompareValues` | Numeric and string comparison for ordering |
| `TestEvaluateHavingExpression` | HAVING clause expression evaluation |
| `BenchmarkExecutor_AggregateCountGroupBy` | Performance: COUNT with GROUP BY |
| `BenchmarkExecutor_ScalarCount` | Performance: scalar COUNT |

### CLI Manual Testing

```bash
# Count articles per chapter (GROUP BY + ORDER BY DESC)
regula query "SELECT ?chapter (COUNT(?article) AS ?articleCount) WHERE { ?article reg:partOf ?chapter } GROUP BY ?chapter ORDER BY DESC(?articleCount)"

# Total provision count (scalar aggregate)
regula query "SELECT (COUNT(?p) AS ?count) WHERE { ?p rdf:type reg:Provision }"

# Definitions per title with minimum threshold (HAVING)
regula query "SELECT ?reg (COUNT(?def) AS ?defCount) WHERE { ?def reg:belongsTo ?reg } GROUP BY ?reg HAVING(COUNT(?def) > 5) ORDER BY DESC(?defCount)"

# Count distinct referenced targets
regula query "SELECT (COUNT(DISTINCT ?target) AS ?uniqueRefs) WHERE { ?article reg:references ?target }"
```

## Bulk Statistics Dashboard Tests

### Unit Tests

Run stats dashboard tests:

```bash
go test ./pkg/bulk/... -v -count=1 -run "Stats|Format"
```

Tests are in `pkg/bulk/report_test.go`. The dashboard supports table, JSON, and CSV output formats.

| Test | Purpose |
|------|---------|
| `TestFormatBytes` | Human-readable byte formatting (B, KB, MB, GB) |
| `TestFormatDatasetTable` | Dataset table output formatting |
| `TestFormatDatasetTableLongName` | Long display names truncated with ellipsis |
| `TestFormatDatasetTableEmpty` | Empty dataset list formatting |
| `TestFormatIngestReport` | Ingest report text with status markers |
| `TestFormatIngestReportJSON` | Ingest report JSON serialization |
| `TestFormatIngestReportWithAggregates` | Ingest report with aggregate totals |
| `TestFormatStatusReport` | Download status report formatting |
| `TestFormatStatusReportFiltered` | Status report filtered by source |
| `TestFormatStatusReportWithIngestStats` | Status report with ingestion statistics |
| `TestFormatDuration` | Duration formatting (ms, s, m) |
| `TestCollectStats` | Stats collection from manifest + document stats |
| `TestCollectStatsEmpty` | Stats collection with empty data |
| `TestFormatStatsTable` | Bulk ingestion statistics table output |
| `TestFormatStatsJSON` | Stats report JSON serialization |
| `TestFormatStatsCSV` | Stats report CSV with headers |
| `TestFormatNumber` | Comma-formatted number output (e.g., 25,100) |

### CLI Manual Testing

```bash
# View statistics table (default format)
regula bulk stats

# JSON output for programmatic consumption
regula bulk stats --export json

# CSV export for spreadsheet analysis
regula bulk stats --export csv > bulk-stats.csv

# Filter by source
regula bulk stats --source uscode
```

## Analysis Playground Tests

### Unit Tests

Run playground template tests:

```bash
go test ./pkg/playground/... -v -count=1
```

Tests are in `pkg/playground/templates_test.go`. The playground provides 10 pre-built SPARQL analysis templates spanning 5 categories: structure, semantics, cross-reference, definitions, and temporal.

| Test | Purpose |
|------|---------|
| `TestRegistryContainsAllTemplates` | All 10 required templates present in registry |
| `TestTemplateNamesAreSorted` | Template names returned in alphabetical order |
| `TestGetExistingTemplate` | Retrieve template by name with correct fields |
| `TestGetMissingTemplate` | Non-existent template returns false |
| `TestRenderQueryNoParameters` | Template without parameters renders unchanged |
| `TestRenderQueryWithTitleFilter` | Title parameter injects FILTER clause |
| `TestRenderQueryWithEmptyTitleFilter` | Empty title produces no FILTER |
| `TestRenderQueryNoParamsForParameterized` | Placeholder removed when no params supplied |
| `TestRenderQueryMissingRequired` | Error for missing required parameters |
| `TestAllTemplatesHaveRequiredFields` | All templates have Name, Description, Category, Query |
| `TestAllTemplatesParseSuccessfully` | All 10 templates parse as valid SPARQL |
| `TestAllTemplatesWithTitleParam` | Parameterized templates parse with title value |
| `TestTemplateCategoriesAreValid` | Categories match allowed set |

### Template Coverage

| Template | Category | Parameters | SPARQL Features |
|----------|----------|------------|-----------------|
| `top-chapters-by-sections` | structure | none | COUNT, GROUP BY, ORDER BY DESC, LIMIT |
| `sections-with-obligations` | semantics | none | Multi-hop join (section → article → obligation) |
| `definition-coverage` | definitions | none | COUNT, GROUP BY, ORDER BY DESC |
| `cross-ref-density` | cross-reference | `--title` | COUNT, GROUP BY, FILTER with parameter |
| `rights-enumeration` | semantics | none | Multi-join with type classification |
| `title-size-comparison` | structure | none | COUNT, GROUP BY, ORDER BY DESC |
| `orphan-sections` | cross-reference | none | OPTIONAL, FILTER(!BOUND()) |
| `definition-reuse` | definitions | none | COUNT(DISTINCT), HAVING > 1 |
| `chapter-structure` | structure | `--title` | OPTIONAL join, FILTER with parameter |
| `temporal-analysis` | temporal | none | OPTIONAL, temporal predicate joins |

### CLI Manual Testing

```bash
# List all available analysis templates
regula playground list

# Run a specific template
regula playground run top-chapters-by-sections --path .regula

# Run with title filter parameter
regula playground run cross-ref-density --path .regula --title 42

# Export results as JSON
regula playground run definition-coverage --path .regula --export json

# Export results as CSV
regula playground run title-size-comparison --path .regula --export csv

# Run a custom SPARQL query through the playground
regula playground query "SELECT ?s ?p ?o WHERE { ?s ?p ?o } LIMIT 10" --path .regula

# Paginate results
regula playground run top-chapters-by-sections --path .regula --limit 10 --offset 20

# Show execution timing
regula playground run definition-reuse --path .regula --timing
```

## Troubleshooting Tests

### Common Issues

1. **Test file not found**: Ensure working directory is project root
2. **Timeout**: Increase timeout with `-timeout 5m`
3. **Race conditions**: Run with `-race` flag

### CI-Specific Issues

- Tests may behave differently on Ubuntu vs macOS
- Ensure scripts use portable commands (no `grep -P`)
- Check file encoding (UTF-8)
