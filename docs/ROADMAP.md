# Regula Development Roadmap

## MVP Definition

**Goal**: A working system that can ingest a single regulation (GDPR), extract its structure, store it as a queryable graph, and answer basic compliance questions.

### MVP Success Criteria (All Must Pass)

| Criteria | Validation Test |
|----------|-----------------|
| **Ingest** | `regula ingest gdpr.txt` completes without error, extracts ≥50 provisions |
| **Query** | `regula query "SELECT ?p WHERE { ?p rdf:type reg:Provision }"` returns results in <100ms |
| **Cross-refs** | ≥80% of internal cross-references correctly resolved (manual audit of 20 samples) |
| **Impact** | `regula impact --provision "GDPR:Art17"` identifies Art 7, 12, 13, 14 as related |
| **Simulate** | `regula simulate consent-withdrawal.yaml` produces compliance report with correct obligations |
| **Audit** | Every query/simulation result includes traceable provenance to source provisions |

### MVP Demo Script
```bash
# 1. Initialize project
regula init gdpr-demo

# 2. Ingest GDPR text
regula ingest --source regulations/gdpr.txt

# 3. Verify extraction
regula query "SELECT (COUNT(?p) as ?count) WHERE { ?p rdf:type reg:Provision }"
# Expected: ?count = 99 (GDPR has 99 articles)

# 4. Query specific provisions
regula query "SELECT ?p ?title WHERE { ?p reg:requires reg:Consent . ?p reg:title ?title }"
# Expected: Articles 6, 7, 8, 9 (consent-related)

# 5. Impact analysis
regula impact --provision "GDPR:Art17" --depth 2
# Expected: Shows Art 7(3), Art 19, Art 12 as directly affected

# 6. Compliance simulation
regula simulate --scenario examples/gdpr/consent-withdrawal.yaml
# Expected: Compliance report with obligations, timelines, applicable provisions

# 7. Audit trail
regula audit --decision "simulation-001"
# Expected: Full provenance chain from conclusion to source articles
```

---

## Milestone Overview

| Milestone | Focus | Duration | Validation Gate |
|-----------|-------|----------|-----------------|
| M1 | Real Data Foundation | 2 weeks | Parse GDPR, extract 50+ provisions |
| M2 | Queryable Graph | 2 weeks | SPARQL queries return correct results |
| M3 | Extraction Pipeline | 2 weeks | 80% cross-reference accuracy |
| M4 | Analysis Engine | 1 week | Impact analysis matches manual review |
| M5 | Simulation MVP | 1 week | Scenario evaluation produces valid report |
| M6 | Integration & Polish | 1 week | Full demo script passes |

**Total: 9 weeks to MVP**

---

## Milestone 1: Real Data Foundation (Weeks 1-2)

### Goal
Get real GDPR text into the system and prove we can parse regulatory structure.

### Validation Test
```bash
# Parse GDPR and output structure
regula ingest --source testdata/gdpr.txt --output-structure

# Expected output:
# Parsed: 99 articles
# Chapters: 11
# Sections: 22
# Definitions found: 26 (Article 4)
# Cross-references detected: 150+
```

### Issues

#### M1.1: Obtain and prepare GDPR test data
- Download official GDPR text (EUR-Lex)
- Clean and normalize formatting
- Create `testdata/gdpr.txt` and `testdata/gdpr-expected.json`
- **Acceptance**: File exists, UTF-8 encoded, sections clearly delimited

#### M1.2: Implement basic document parser
- Parse article/section boundaries
- Extract article numbers and titles
- Handle nested structure (Chapter > Section > Article > Paragraph)
- **Acceptance**: `go test ./pkg/extract/... -run TestParseGDPR` passes

#### M1.3: Implement provision extraction
- Extract provision text content
- Identify paragraph and point numbering
- Preserve original formatting/numbering
- **Acceptance**: Round-trip test (parse → serialize → parse) matches

#### M1.4: Implement definition extraction
- Parse "Article 4 - Definitions" structure
- Extract term → definition mappings
- Handle nested definitions
- **Acceptance**: All 26 GDPR definitions extracted correctly

#### M1.5: Implement cross-reference detection
- Regex patterns for internal refs ("Article 17", "paragraph 1", etc.)
- Detect external refs ("Directive 95/46/EC")
- Store as unresolved references
- **Acceptance**: ≥90% of references detected (manual audit of 50 samples)

### Deliverables
- [ ] `testdata/gdpr.txt` - Source text
- [ ] `testdata/gdpr-expected.json` - Expected parse output
- [ ] `pkg/extract/parser.go` - Document parser
- [ ] `pkg/extract/parser_test.go` - Parser tests with GDPR
- [ ] Parsing accuracy report

---

## Milestone 2: Queryable Graph (Weeks 3-4)

### Goal
Store extracted provisions in RDF triple store, query with SPARQL.

### Validation Test
```bash
# Load GDPR into graph
regula ingest --source testdata/gdpr.txt

# Query provisions
regula query "SELECT ?article ?title WHERE {
  ?article rdf:type reg:Article .
  ?article reg:title ?title
} ORDER BY ?article LIMIT 10"

# Expected: First 10 GDPR articles with titles

# Query definitions
regula query "SELECT ?term ?definition WHERE {
  ?term rdf:type reg:DefinedTerm .
  ?term reg:definition ?definition
}"

# Expected: 26 definitions from Article 4
```

### Issues

#### M2.1: Port triple store from GraphFS
- Adapt `internal/store/store.go` from GraphFS
- Implement SPO, POS, OSP indexes
- Add thread-safe operations
- **Acceptance**: Benchmark inserts 10,000 triples/sec

#### M2.2: Define regulation RDF schema
- Create `pkg/store/schema.go` with predicates
- Document ontology in `docs/ONTOLOGY.md`
- Include: reg:Article, reg:Section, reg:DefinedTerm, reg:references, etc.
- **Acceptance**: Schema covers all provision types

#### M2.3: Implement graph builder
- Convert extracted provisions to RDF triples
- Generate URIs for provisions (e.g., `<GDPR:Art17>`)
- Link definitions to their defining provisions
- **Acceptance**: GDPR produces ~5000 triples

#### M2.4: Port SPARQL parser from GraphFS
- Basic SELECT queries
- WHERE clause with triple patterns
- FILTER expressions
- **Acceptance**: All example queries in tests pass

#### M2.5: Implement query executor
- Query planning
- Index selection
- Result formatting (table, JSON, CSV)
- **Acceptance**: Queries complete in <100ms for GDPR-sized graph

#### M2.6: Add query CLI command
- `regula query <sparql>` execution
- `regula query --template <name>` for common queries
- Output format flags
- **Acceptance**: Demo queries work end-to-end

### Deliverables
- [ ] `pkg/store/triple.go` - Triple type
- [ ] `pkg/store/store.go` - Triple store with indexes
- [ ] `pkg/store/schema.go` - RDF schema
- [ ] `pkg/query/parser.go` - SPARQL parser
- [ ] `pkg/query/executor.go` - Query execution
- [ ] `docs/ONTOLOGY.md` - Schema documentation
- [ ] Query performance benchmarks

---

## Milestone 3: Extraction Pipeline (Weeks 5-6)

### Goal
Resolve cross-references and build complete provision graph with relationships.

### Validation Test
```bash
# Check cross-reference resolution
regula validate --check references

# Expected output:
# Total references: 156
# Resolved: 142 (91%)
# Unresolved: 14 (9%)
#   - External: 8 (Directive 95/46/EC, etc.)
#   - Ambiguous: 6 (require manual review)

# Query resolved references
regula query "SELECT ?from ?to WHERE {
  ?from reg:references ?to .
  ?to reg:title ?title
} LIMIT 10"

# Expected: Shows which articles reference which
```

### Issues

#### M3.1: Implement reference resolver
- Parse reference patterns ("Article 17", "Article 17(1)(a)")
- Resolve to provision URIs
- Handle ambiguous references
- **Acceptance**: ≥85% resolution rate on GDPR

#### M3.2: Implement LLM-assisted extraction (optional)
- Structured prompts for complex extractions
- Validate LLM outputs against schema
- Human review queue for uncertain results
- **Acceptance**: LLM improves resolution by ≥5%

#### M3.3: Build provision relationship graph
- Add `reg:references` edges
- Add `reg:definedIn` edges (terms to definitions)
- Add `reg:partOf` edges (article to chapter)
- **Acceptance**: Graph visualization shows connected structure

#### M3.4: Implement obligation/right extraction
- Identify obligation language ("shall", "must")
- Identify right language ("right to", "entitled to")
- Tag provisions with `reg:imposesObligation`, `reg:grantsRight`
- **Acceptance**: Manual audit confirms ≥80% accuracy on 20 samples

#### M3.5: Add validation command
- `regula validate` checks graph consistency
- Reports unresolved references
- Reports orphan provisions
- **Acceptance**: GDPR passes validation with documented exceptions

### Deliverables
- [ ] `pkg/extract/resolver.go` - Reference resolution
- [ ] `pkg/extract/semantic.go` - Obligation/right extraction
- [ ] `pkg/extract/validate.go` - Validation checks
- [ ] Reference resolution accuracy report
- [ ] Graph visualization of GDPR structure

---

## Milestone 4: Analysis Engine (Week 7)

### Goal
Analyze impact of regulatory changes and detect conflicts.

### Validation Test
```bash
# Impact analysis for Article 17 (Right to Erasure)
regula impact --provision "GDPR:Art17" --depth 2

# Expected output:
# Direct Impact (references Art 17):
#   - GDPR:Art19 (Notification obligation)
#   - GDPR:Art12 (Transparent communication)
#
# Provisions Art 17 references:
#   - GDPR:Art6 (Lawfulness of processing)
#   - GDPR:Art9 (Special categories)
#   - GDPR:Art17(3) (Exceptions)
#
# Transitive Impact (depth 2):
#   - GDPR:Art7 (Conditions for consent) via Art6
#   - ...
#
# Risk Assessment: MEDIUM
# Reason: Art 17 is referenced by 2 provisions, references 5 provisions

# Verify against manual analysis
diff <(regula impact --provision "GDPR:Art17" --format json) testdata/art17-impact-expected.json
```

### Issues

#### M4.1: Port graph algorithms from GraphFS
- Topological sort
- Transitive closure
- Shortest path
- Strongly connected components (circular refs)
- **Acceptance**: All algorithms have tests with known graphs

#### M4.2: Implement impact analysis
- Direct impact (provisions referencing target)
- Reverse impact (provisions target references)
- Transitive impact (configurable depth)
- **Acceptance**: Art 17 analysis matches manual review

#### M4.3: Implement risk assessment
- Score based on reference count, centrality
- Categorize: LOW, MEDIUM, HIGH, CRITICAL
- Generate recommendations
- **Acceptance**: Risk levels are defensible (documented rationale)

#### M4.4: Add conflict detection
- Detect contradictory obligations
- Detect overlapping definitions
- Report potential conflicts
- **Acceptance**: No false positives on GDPR (internally consistent)

#### M4.5: Add impact CLI command
- `regula impact --provision <id>`
- `regula impact --change amend|repeal|add`
- Output formats: text, JSON, graph
- **Acceptance**: Demo commands work correctly

### Deliverables
- [ ] `pkg/analysis/algorithms.go` - Graph algorithms
- [ ] `pkg/analysis/impact.go` - Impact analysis
- [ ] `pkg/analysis/conflict.go` - Conflict detection
- [ ] `testdata/art17-impact-expected.json` - Expected results
- [ ] Impact analysis validation report

---

## Milestone 5: Simulation MVP (Week 8)

### Goal
Evaluate compliance scenarios against the regulation graph.

### Validation Test
```bash
# Run consent withdrawal scenario
regula simulate --scenario examples/gdpr/consent-withdrawal.yaml

# Expected output:
# Scenario: Consent Withdrawal
#
# Applicable Provisions:
#   - GDPR:Art7(3) - Right to withdraw consent [DIRECT]
#   - GDPR:Art17(1) - Right to erasure [TRIGGERED]
#   - GDPR:Art12(3) - Response timeline [PROCEDURAL]
#
# Obligations for DataController:
#   - Cease processing: without undue delay
#   - Acknowledge withdrawal: within 1 month
#   - Complete erasure: within 1 month
#
# Compliance Status: REQUIRES_ACTION
#
# Timeline:
#   Day 0: Withdrawal received
#   Day 1-3: Cease processing (recommended)
#   Day 30: Acknowledge withdrawal (required)
#   Day 30: Complete erasure (required)
#
# Exceptions Available:
#   - GDPR:Art17(3)(b) - Legal obligation to retain
#   - GDPR:Art17(3)(e) - Legal claims

# Verify key obligations are identified
regula simulate --scenario examples/gdpr/consent-withdrawal.yaml --format json | \
  jq '.obligations[] | select(.type == "acknowledge_withdrawal")'
# Expected: Shows 1 month deadline
```

### Issues

#### M5.1: Define scenario schema
- YAML schema for scenarios
- Entities, triggers, expected outcomes
- JSON Schema for validation
- **Acceptance**: Schema documented, examples validate

#### M5.2: Implement scenario parser
- Parse scenario YAML
- Validate against schema
- Create internal scenario model
- **Acceptance**: All example scenarios parse correctly

#### M5.3: Implement provision matcher
- Match scenario conditions to provisions
- Score relevance (DIRECT, TRIGGERED, RELATED)
- Handle entity type matching
- **Acceptance**: Consent scenario finds Art 7, 17, 12

#### M5.4: Implement obligation extractor
- Extract obligations from matched provisions
- Parse deadline language ("without undue delay", "within 1 month")
- Identify responsible parties
- **Acceptance**: Correct obligations for consent scenario

#### M5.5: Implement compliance evaluator
- Evaluate scenario against obligations
- Generate compliance status
- Identify gaps and risks
- **Acceptance**: Produces actionable compliance report

#### M5.6: Add simulate CLI command
- `regula simulate --scenario <file>`
- Output formats: report, JSON
- Verbose mode for debugging
- **Acceptance**: Demo scenario works end-to-end

### Deliverables
- [ ] `pkg/simulate/schema.go` - Scenario schema
- [ ] `pkg/simulate/parser.go` - Scenario parser
- [ ] `pkg/simulate/matcher.go` - Provision matching
- [ ] `pkg/simulate/evaluator.go` - Compliance evaluation
- [ ] `examples/gdpr/consent-withdrawal.yaml` - Test scenario
- [ ] `examples/gdpr/data-breach.yaml` - Additional scenario
- [ ] Simulation accuracy report

---

## Milestone 6: Integration & Polish (Week 9)

### Goal
Integrate all components, add audit trails, polish for demo.

### Validation Test
```bash
# Full end-to-end test script
./scripts/e2e-test.sh

# Expected: All steps pass
# Step 1: Init project ✓
# Step 2: Ingest GDPR ✓
# Step 3: Validate extraction ✓
# Step 4: Run queries ✓
# Step 5: Impact analysis ✓
# Step 6: Simulate scenario ✓
# Step 7: Generate audit trail ✓
#
# MVP Validation: PASSED
# - Provisions extracted: 99/99 ✓
# - Query response time: 45ms ✓
# - Cross-ref accuracy: 91% ✓
# - Impact analysis: matches expected ✓
# - Simulation: correct obligations ✓
# - Audit trail: complete ✓
```

### Issues

#### M6.1: Implement audit trail
- Track all operations with timestamps
- Link results to source provisions
- Store reasoning chain
- **Acceptance**: Every result traceable to source

#### M6.2: Implement provenance queries
- `regula audit --decision <id>`
- Show full reasoning chain
- Export as JSON, W3C PROV format
- **Acceptance**: Audit command produces complete trail

#### M6.3: Create E2E test script
- Automated test of full demo flow
- Validate all MVP criteria
- Report pass/fail for each criterion
- **Acceptance**: Script runs in CI

#### M6.4: Performance optimization
- Profile slow operations
- Add caching where needed
- Target: full GDPR in <5s, queries <100ms
- **Acceptance**: Performance targets met

#### M6.5: Documentation and examples
- User guide with examples
- API documentation
- Troubleshooting guide
- **Acceptance**: New user can run demo in 10 minutes

#### M6.6: MVP demo recording
- Record demo video
- Write blog post / README update
- Prepare for feedback collection
- **Acceptance**: Demo materials ready

### Deliverables
- [ ] `pkg/audit/trail.go` - Audit trail
- [ ] `pkg/audit/provenance.go` - Provenance queries
- [ ] `scripts/e2e-test.sh` - E2E test script
- [ ] `docs/USER_GUIDE.md` - User documentation
- [ ] Performance benchmark report
- [ ] Demo video / materials

---

## Testing Strategy

### Test Levels

| Level | Purpose | Location | Run Frequency |
|-------|---------|----------|---------------|
| Unit | Test individual functions | `*_test.go` | Every commit |
| Integration | Test component interactions | `test/integration/` | Every PR |
| E2E | Test full workflows | `scripts/e2e-test.sh` | Daily / Release |
| Validation | Test against real data | `test/validation/` | Weekly |

### Test Data

| Data | Source | Purpose |
|------|--------|---------|
| `testdata/gdpr.txt` | EUR-Lex | Primary test regulation |
| `testdata/gdpr-expected.json` | Manual | Expected parse output |
| `testdata/art17-impact-expected.json` | Manual | Expected impact analysis |
| `examples/gdpr/*.yaml` | Manual | Simulation scenarios |

### Validation Audits

Each milestone includes manual validation:

1. **Extraction Audit** (M1, M3)
   - Sample 20 random provisions
   - Verify text matches source
   - Check cross-references resolved correctly

2. **Query Audit** (M2)
   - Run 10 standard queries
   - Verify results manually
   - Check performance metrics

3. **Impact Audit** (M4)
   - Select 5 key provisions
   - Manual impact analysis
   - Compare to tool output

4. **Simulation Audit** (M5)
   - Run 3 scenarios
   - Expert review of obligations
   - Verify completeness

---

## Issue Labels

| Label | Description |
|-------|-------------|
| `milestone:M1` - `milestone:M6` | Milestone association |
| `type:feature` | New functionality |
| `type:bug` | Bug fix |
| `type:test` | Test coverage |
| `type:docs` | Documentation |
| `priority:critical` | Blocks milestone |
| `priority:high` | Important for milestone |
| `priority:medium` | Should have |
| `priority:low` | Nice to have |
| `validation:required` | Needs manual validation |

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| LLM extraction unreliable | Rule-based fallback, human review queue |
| GDPR text formatting varies | Multiple parser strategies, normalization |
| Cross-reference patterns complex | Iterative pattern library, manual override |
| Performance issues at scale | Early profiling, index optimization |
| Scope creep | Strict MVP criteria, defer enhancements |

---

## Post-MVP Roadmap

After MVP validation, potential enhancements:

1. **Multi-regulation support** - Ingest additional regulations
2. **Web interface** - GraphQL API, dashboard
3. **Historical analysis** - Track regulation changes over time
4. **Team features** - Multi-user, review workflows
5. **Integration APIs** - Connect to compliance systems
6. **Advanced simulation** - Multi-party scenarios, conflict resolution
