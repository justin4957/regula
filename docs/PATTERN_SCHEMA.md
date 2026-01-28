# Pattern Definition Schema

This document describes the YAML/JSON schema for defining legislative document format patterns in Regula.

## Overview

Pattern definitions tell Regula how to:
1. **Detect** document formats using weighted indicators
2. **Parse** document structure (chapters, sections, articles, etc.)
3. **Extract** legal definitions
4. **Identify** internal and external references

## Schema Location

The JSON Schema is available at:
- `schemas/pattern-v1.0.json`

## Quick Reference

```yaml
# Pattern Definition v1.0
name: string                    # Human-readable name (required)
version: string                 # Semantic version (required)
jurisdiction: string            # ISO country code or subdivision
format_id: string               # Unique identifier (required)

detection:                      # Format detection config (required)
  required_indicators: []       # Must match for detection
  optional_indicators: []       # Increase confidence
  negative_indicators: []       # Decrease confidence

structure:                      # Document structure parsing
  hierarchy: []                 # Structural elements
  preamble: {}                  # Preamble configuration

definitions:                    # Legal definition extraction
  location: []                  # Where to find definitions
  pattern: string               # How definitions are formatted

references:                     # Reference extraction
  internal: []                  # Same-document references
  external: []                  # Other document references
```

## Required Fields

Every pattern must include:
- `name` - Human-readable name
- `format_id` - Unique identifier (lowercase, hyphen-separated)
- `version` - Semantic version (e.g., "1.0.0")
- `detection.required_indicators` - At least one indicator

## Detection Configuration

Detection uses weighted indicators to calculate a confidence score.

### Required Indicators

At least one required indicator must match for the format to be detected:

```yaml
detection:
  required_indicators:
    - pattern: '\bU\.?S\.?C\.?\s*§?\s*\d+'
      weight: 25
    - pattern: '\bUnited\s+States\s+Code\b'
      weight: 20
```

| Field | Type | Description |
|-------|------|-------------|
| `pattern` | string | Regular expression (Go/RE2 syntax) |
| `weight` | integer | Positive weight (1-100) |

### Optional Indicators

Add to confidence but are not required:

```yaml
  optional_indicators:
    - pattern: '\bTitle\s+\d+'
      weight: 10
    - pattern: '\bChapter\s+\d+'
      weight: 5
```

### Negative Indicators

Decrease confidence (use negative weights):

```yaml
  negative_indicators:
    - pattern: '\bEuropean\s+Union\b'
      weight: -20
    - pattern: '\bDirective\s+\d{4}/\d+'
      weight: -25
```

| Field | Type | Description |
|-------|------|-------------|
| `pattern` | string | Regular expression (Go/RE2 syntax) |
| `weight` | integer | Negative weight (-100 to -1) |

### Confidence Calculation

```
confidence = (sum of matched weights) / (sum of required indicator weights)
confidence = clamp(confidence, 0, 1)
```

## Structure Configuration

Define the document's hierarchical structure.

### Hierarchy

List structural elements from highest to lowest level:

```yaml
structure:
  hierarchy:
    - type: "title"
      pattern: '^TITLE\s+(\d+)[—\-–]\s*(.+)$'
      number_group: 1
      title_inline: true

    - type: "chapter"
      pattern: '^CHAPTER\s+(\d+)'
      number_group: 1
      title_follows: true

    - type: "section"
      pattern: '^§\s*(\d+)\.'
      number_group: 1

    - type: "subsection"
      pattern: '^\(([a-z])\)'
      number_group: 1
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | enum | Element type (see below) |
| `pattern` | string | Regex pattern to match |
| `title_follows` | boolean | Title on next line |
| `title_inline` | boolean | Title in same match |
| `number_group` | integer | Capture group for number |

**Supported hierarchy types:**
- `title`, `part`, `chapter`, `subchapter`, `division`
- `section`, `subsection`, `article`
- `paragraph`, `subparagraph`, `point`, `subpoint`
- `subdivision`, `schedule`, `annex`

### Preamble Configuration

For documents with preambles (EU legislation, etc.):

```yaml
structure:
  preamble:
    start_pattern: '^THE\s+EUROPEAN\s+PARLIAMENT'
    end_pattern: '^HAVE\s+ADOPTED\s+THIS'
    recital_pattern: '^\((\d+)\)\s+'
```

## Definition Extraction

Configure how to find and extract legal definitions.

```yaml
definitions:
  location:
    - section_title: '(?i)definitions?'
    - section_number: 1798.140

  pattern: '(?:^|\n)\s*["""]([^"""]+)["""]\\s+means?'
```

| Field | Type | Description |
|-------|------|-------------|
| `location.section_title` | string | Regex for section titles |
| `location.section_number` | integer | Specific section number |
| `pattern` | string | Pattern to extract definitions |

## Reference Configuration

### Internal References

References within the same document:

```yaml
references:
  internal:
    - pattern: '(?:Section|§)\s*(\d+(?:\.\d+)?)'
      target: "section"
      groups:
        number: 1

    - pattern: 'Article\s+(\d+)(?:\((\d+)\))?'
      target: "article"
      groups:
        article: 1
        paragraph: 2
```

| Field | Type | Description |
|-------|------|-------------|
| `pattern` | string | Regex to match references |
| `target` | enum | Target element type |
| `groups` | object | Named capture group mappings |

### External References

References to other documents:

```yaml
references:
  external:
    - pattern: '(\d+)\s+U\.?S\.?C\.?\s*§?\s*(\d+)'
      type: "usc"
      uri_template: "https://www.law.cornell.edu/uscode/text/{title}/{section}"
      groups:
        title: 1
        section: 2

    - pattern: 'Directive\s+(\d{4})/(\d+)/(EC|EU)'
      type: "directive"
      uri_template: "http://data.europa.eu/eli/dir/{year}/{number}/oj"
      groups:
        year: 1
        number: 2
```

| Field | Type | Description |
|-------|------|-------------|
| `pattern` | string | Regex to match references |
| `type` | string | Document type identifier |
| `uri_template` | string | Template for URI generation |
| `groups` | object | Named capture group mappings |

**URI Template Variables:** Use `{name}` syntax to insert captured group values.

## Complete Example

```yaml
# US Code Pattern Definition
name: "United States Code"
format_id: "usc"
version: "1.0.0"
jurisdiction: "US-Federal"

detection:
  required_indicators:
    - pattern: '\bU\.?S\.?C\.?\s*§?\s*\d+'
      weight: 25
  optional_indicators:
    - pattern: '\bTitle\s+\d+'
      weight: 10
  negative_indicators:
    - pattern: '\bEuropean\s+Union\b'
      weight: -20

structure:
  hierarchy:
    - type: "title"
      pattern: '^TITLE\s+(\d+)'
      number_group: 1
    - type: "section"
      pattern: '^§\s*(\d+)\.'
      number_group: 1

definitions:
  location:
    - section_title: '(?i)definitions?'
  pattern: '["""]([^"""]+)["""]\\s+means?'

references:
  internal:
    - pattern: '(?:Section|§)\s*(\d+)'
      target: "section"
      groups:
        number: 1
  external:
    - pattern: '(\d+)\s+U\.?S\.?C\.?\s*§?\s*(\d+)'
      type: "usc"
      uri_template: "https://www.law.cornell.edu/uscode/text/{title}/{section}"
      groups:
        title: 1
        section: 2
```

## Validation

### Using JSON Schema

Validate patterns against the JSON Schema:

```bash
# Using ajv-cli
npx ajv validate -s schemas/pattern-v1.0.json -d patterns/us-code.yaml

# Using Python jsonschema
python -c "
import yaml
import jsonschema
schema = json.load(open('schemas/pattern-v1.0.json'))
pattern = yaml.safe_load(open('patterns/us-code.yaml'))
jsonschema.validate(pattern, schema)
"
```

### Using Go API

```go
import "github.com/justin4957/regula/pkg/pattern"

registry := pattern.NewRegistry()
err := registry.LoadFile("patterns/us-code.yaml")
if err != nil {
    // Validation or compilation error
    log.Fatal(err)
}
```

The registry validates:
1. Required fields are present
2. All regex patterns compile successfully
3. No duplicate format IDs (same version)

## Regular Expression Syntax

Patterns use Go's RE2 syntax. Key features:
- Case insensitive: `(?i)pattern`
- Non-capturing groups: `(?:...)`
- Named groups: `(?P<name>...)`
- Character classes: `\d`, `\w`, `\s`, `\b`

**Note:** Backreferences are NOT supported in RE2.

## Jurisdiction Codes

Use ISO 3166-1 alpha-2 codes with optional subdivisions:

| Code | Description |
|------|-------------|
| `US` | United States (federal) |
| `US-CA` | California |
| `US-VA` | Virginia |
| `EU` | European Union |
| `GB` | United Kingdom |
| `INT` | International |

## Best Practices

1. **Start specific**: Use highly specific required indicators to avoid false positives
2. **Balance weights**: Required indicators should sum to a reasonable baseline
3. **Test thoroughly**: Test patterns against real documents from the jurisdiction
4. **Version patterns**: Use semantic versioning when updating patterns
5. **Document edge cases**: Note any known limitations in comments

## Example Patterns

See the `patterns/` directory for complete examples:
- `us-code.yaml` - United States Code
- `eu-directive.yaml` - EU Directives
- `eu-regulation.yaml` - EU Regulations (GDPR, etc.)
- `california-code.yaml` - California state code (CCPA)
- `uk-statutory-instrument.yaml` - UK Statutory Instruments
