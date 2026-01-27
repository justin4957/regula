# Regula Development Guidelines

## Project Overview

Regula is an automated regulation mapper that transforms dense legal regulations into auditable, queryable, and simulatable programs.

## Architecture Origins

This project synthesizes ideas from:
- **lex-sim** (Crisp): Legal domain modeling with type-safe verification
- **GraphFS** (Go): RDF triple store with SPARQL queries and impact analysis
- **Crisp type system** (Haskell): Sophisticated type checking patterns

## Development Phases

See `docs/ROADMAP.md` for the complete development roadmap.

Current focus: **Phase 1 - Core Type System**

## Code Conventions

### Go Style
- Follow standard Go conventions
- Use descriptive variable names
- Prefer composition over inheritance
- Document exported types and functions

### Type System (pkg/types/)
- Ported from lex-sim Crisp code
- Use `Kind` suffix for discriminated union tags
- Constructors follow the pattern: `func TypeName(params) Type`
- Proof types implement the `Proof` interface

### Testing
- Unit tests for all types
- Integration tests for workflows
- Use table-driven tests where appropriate

## Key Design Decisions

### Proof Types in Go
Unlike Crisp's compile-time proofs, Go proofs are runtime-verified:
```go
proof := ProveBindsOn(higherCourt, lowerCourt)
if proof == nil {
    // Cannot prove relationship
}
if err := proof.Verify(); err != nil {
    // Proof is invalid
}
```

### RDF Predicates
Use the `reg:` namespace for regulation-specific predicates:
- `reg:amends` - Amendment relationship
- `reg:supersedes` - Supersession relationship
- `reg:delegatesTo` - Authority delegation
- `reg:grantsRight` - Rights granted
- `reg:imposesObligation` - Obligations imposed

## Building and Testing

```bash
# Build
make build

# Test
make test

# Run examples
make example-init
make example-query
```

## Contribution Guidelines

1. Create feature branches from `main`
2. Write tests for new functionality
3. Update documentation as needed
4. Run `make lint` before committing
5. Use conventional commit messages

## Related Documentation

- `docs/ROADMAP.md` - Development phases and timeline
- `docs/ARCHITECTURE.md` - System architecture details
- `examples/` - Example scenarios and regulations
