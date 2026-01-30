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
go run cmd/regula/main.go export --source testdata/gdpr.txt --format jsonld --output graph.jsonld

# CLI: Export as expanded JSON-LD (full URIs, no @context)
go run cmd/regula/main.go export --source testdata/gdpr.txt --format jsonld --expanded --output graph-expanded.jsonld

# CLI: Export with ELI enrichment
go run cmd/regula/main.go export --source testdata/gdpr.txt --format jsonld --eli --output graph-eli.jsonld
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
go run cmd/regula/main.go validate --source testdata/gdpr.txt --check links

# CLI: Save link report to JSON
go run cmd/regula/main.go validate --source testdata/gdpr.txt --check links --report links.json

# CLI: Save link report to Markdown
go run cmd/regula/main.go validate --source testdata/gdpr.txt --check links --report links.md
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
go run cmd/regula/main.go query --source testdata/gdpr.txt \
  "CONSTRUCT { ?a <http://example.org/title> ?t } WHERE { ?a rdf:type reg:Article . ?a reg:title ?t }"

# CLI: Run CONSTRUCT query with N-Triples output
go run cmd/regula/main.go query --source testdata/gdpr.txt --format ntriples \
  "CONSTRUCT { ?a <http://example.org/type> <http://example.org/Article> } WHERE { ?a rdf:type reg:Article }"

# CLI: Run CONSTRUCT query with JSON output
go run cmd/regula/main.go query --source testdata/gdpr.txt --format json \
  "CONSTRUCT { ?a <http://example.org/type> <http://example.org/Article> } WHERE { ?a rdf:type reg:Article }"

# CLI: Run CONSTRUCT query with timing
go run cmd/regula/main.go query --source testdata/gdpr.txt --timing \
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
go run cmd/regula/main.go query --source testdata/gdpr.txt "DESCRIBE GDPR:Art17"

# CLI: DESCRIBE with variable
go run cmd/regula/main.go query --source testdata/gdpr.txt \
  "DESCRIBE ?article WHERE { ?article reg:title \"Right to erasure\" }"

# CLI: DESCRIBE with JSON output
go run cmd/regula/main.go query --source testdata/gdpr.txt --format json "DESCRIBE GDPR:Art17"

# CLI: DESCRIBE with timing
go run cmd/regula/main.go query --source testdata/gdpr.txt --timing "DESCRIBE GDPR:Art17"

# CLI: Use describe-article template
go run cmd/regula/main.go query --source testdata/gdpr.txt --template describe-article
```

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
go run cmd/regula/main.go ingest --source testdata/gdpr.txt --gates
go run cmd/regula/main.go ingest --source testdata/gdpr.txt --gates --strict
go run cmd/regula/main.go ingest --source testdata/gdpr.txt --gates --skip-gates V0

# CLI: Run gate-based validation
go run cmd/regula/main.go validate --source testdata/gdpr.txt --check gates
go run cmd/regula/main.go validate --source testdata/gdpr.txt --check gates --format json
```

### E2E Tests

The E2E test script validates the complete MVP functionality:

```bash
# Build the binary first
go build -o ./regula ./cmd/regula

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
    REGULA_BIN: ./regula
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
go run cmd/regula/main.go validate --source testdata/vcdpa.txt

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
./regula query --source testdata/gdpr.txt --template articles --timing
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
