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

### Expected Output Files

Expected outputs for comparison testing:

| File | Description |
|------|-------------|
| `testdata/art17-impact-expected.json` | Expected impact analysis for Art 17 |

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
