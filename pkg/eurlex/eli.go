package eurlex

import (
	"fmt"

	"github.com/coolbeans/regula/pkg/citation"
)

// ELI type slugs used in the URI path.
const (
	eliSlugRegulation = "reg"
	eliSlugDirective  = "dir"
	eliSlugDecision   = "dec"
)

// GenerateELI creates an ELI URI from a parsed EU citation.
// Returns an error if the citation lacks required components or
// has an unsupported citation type.
//
// ELI format: http://data.europa.eu/eli/{type}/{year}/{number}/oj
// Example: Regulation (EU) 2016/679 -> http://data.europa.eu/eli/reg/2016/679/oj
func GenerateELI(citationRef *citation.Citation) (ELIURI, error) {
	if citationRef == nil {
		return ELIURI{}, fmt.Errorf("citation cannot be nil")
	}

	docYear := citationRef.Components.DocYear
	docNumber := citationRef.Components.DocNumber

	if docYear == "" {
		return ELIURI{}, fmt.Errorf("citation missing required year component")
	}
	if docNumber == "" {
		return ELIURI{}, fmt.Errorf("citation missing required number component")
	}

	typeSlug, err := citationTypeToELISlug(citationRef.Type)
	if err != nil {
		return ELIURI{}, err
	}

	normalizedYear := normalizeYear(docYear)

	return ELIURI{
		TypeSlug: typeSlug,
		Year:     normalizedYear,
		Number:   docNumber, // ELI uses unpadded numbers.
	}, nil
}

// citationTypeToELISlug maps citation.CitationType to the ELI type path segment.
func citationTypeToELISlug(citationType citation.CitationType) (string, error) {
	switch citationType {
	case citation.CitationTypeRegulation:
		return eliSlugRegulation, nil
	case citation.CitationTypeDirective:
		return eliSlugDirective, nil
	case citation.CitationTypeDecision:
		return eliSlugDecision, nil
	default:
		return "", fmt.Errorf("unsupported citation type for ELI generation: %s", citationType)
	}
}
