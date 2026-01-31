package fetch

import (
	"fmt"
	"strings"

	"github.com/coolbeans/regula/pkg/citation"
	"github.com/coolbeans/regula/pkg/eurlex"
	"github.com/coolbeans/regula/pkg/uscode"
)

// URNMapper translates URN-style identifiers (produced by the reference resolver)
// into fetchable HTTP URLs using existing connectors (EUR-Lex, UK legislation, etc.).
type URNMapper struct{}

// NewURNMapper creates a new URN mapper.
func NewURNMapper() *URNMapper {
	return &URNMapper{}
}

// MapURN converts a URN identifier to a fetchable HTTP URL.
// Returns an error if the URN format is unrecognized or the document type
// is not supported for fetching.
//
// Supported URN patterns:
//   - urn:eu:regulation:{year}/{number} → EUR-Lex ELI URL
//   - urn:eu:directive:{year}/{number}  → EUR-Lex ELI URL
//   - urn:eu:decision:{year}/{number}   → EUR-Lex ELI URL
func (mapper *URNMapper) MapURN(urn string) (string, error) {
	if urn == "" {
		return "", fmt.Errorf("URN is empty")
	}

	switch {
	case strings.HasPrefix(urn, "urn:eu:regulation:"):
		return mapper.mapEUDocument(urn, "urn:eu:regulation:", citation.CitationTypeRegulation)

	case strings.HasPrefix(urn, "urn:eu:directive:"):
		return mapper.mapEUDocument(urn, "urn:eu:directive:", citation.CitationTypeDirective)

	case strings.HasPrefix(urn, "urn:eu:decision:"):
		return mapper.mapEUDocument(urn, "urn:eu:decision:", citation.CitationTypeDecision)

	case strings.HasPrefix(urn, "urn:eu:treaty:"):
		return "", fmt.Errorf("treaties are not fetchable from EUR-Lex: %s", urn)

	case strings.HasPrefix(urn, "urn:us:usc:"):
		return mapper.mapUSCDocument(urn)

	case strings.HasPrefix(urn, "urn:us:cfr:"):
		return mapper.mapCFRDocument(urn)

	case strings.HasPrefix(urn, "urn:us:"):
		return "", fmt.Errorf("unsupported US legislation URN subtype: %s", urn)

	case strings.HasPrefix(urn, "urn:external:"):
		return "", fmt.Errorf("generic external references are not fetchable: %s", urn)

	default:
		return "", fmt.Errorf("unrecognized URN format: %s", urn)
	}
}

// mapEUDocument maps an EU-style URN to a fetchable ELI URL via eurlex.GenerateELI.
func (mapper *URNMapper) mapEUDocument(urn string, prefix string, citationType citation.CitationType) (string, error) {
	docYear, docNumber, err := parseEUDocURN(urn, prefix)
	if err != nil {
		return "", err
	}

	citationRef := &citation.Citation{
		Type: citationType,
		Components: citation.CitationComponents{
			DocYear:   docYear,
			DocNumber: docNumber,
		},
	}

	eliURI, err := eurlex.GenerateELI(citationRef)
	if err != nil {
		return "", fmt.Errorf("failed to generate ELI for %s: %w", urn, err)
	}

	return eliURI.String(), nil
}

// parseEUDocURN extracts year and number from an EU-style URN suffix.
// Expected format after prefix: "{year}/{number}"
func parseEUDocURN(urn string, prefix string) (year string, number string, err error) {
	suffix := strings.TrimPrefix(urn, prefix)
	if suffix == "" {
		return "", "", fmt.Errorf("URN has no document identifier: %s", urn)
	}

	parts := strings.SplitN(suffix, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("URN has invalid year/number format: %s (expected {year}/{number})", urn)
	}

	return parts[0], parts[1], nil
}

// mapUSCDocument maps a USC URN to a fetchable uscode.house.gov URL.
// Expected format: urn:us:usc:{title}/{section}
func (mapper *URNMapper) mapUSCDocument(urn string) (string, error) {
	suffix := strings.TrimPrefix(urn, "urn:us:usc:")
	if suffix == "" {
		return "", fmt.Errorf("USC URN has no title/section: %s", urn)
	}

	parts := strings.SplitN(suffix, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("USC URN has invalid title/section format: %s (expected urn:us:usc:{title}/{section})", urn)
	}

	uscURI := uscode.USCURI{
		Title:   parts[0],
		Section: parts[1],
	}
	return uscURI.String(), nil
}

// mapCFRDocument maps a CFR URN to a fetchable ecfr.gov URL.
// Expected formats:
//
//	urn:us:cfr:{title}/{part}
//	urn:us:cfr:{title}/{part}/{section}
func (mapper *URNMapper) mapCFRDocument(urn string) (string, error) {
	suffix := strings.TrimPrefix(urn, "urn:us:cfr:")
	if suffix == "" {
		return "", fmt.Errorf("CFR URN has no title/part: %s", urn)
	}

	parts := strings.SplitN(suffix, "/", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("CFR URN has invalid format: %s (expected urn:us:cfr:{title}/{part}[/{section}])", urn)
	}

	cfrURI := uscode.CFRURI{
		Title: parts[0],
		Part:  parts[1],
	}
	if len(parts) == 3 && parts[2] != "" {
		cfrURI.Section = parts[2]
	}
	return cfrURI.String(), nil
}
