package ukleg

import (
	"fmt"

	"github.com/coolbeans/regula/pkg/citation"
)

// GenerateLegislationURI creates a legislation.gov.uk URI from a parsed UK citation.
// Returns an error if the citation lacks required components or has an unsupported type.
//
// URI formats:
//   - Acts: https://www.legislation.gov.uk/ukpga/{year}/{chapter}
//   - SIs:  https://www.legislation.gov.uk/uksi/{year}/{number}
func GenerateLegislationURI(citationRef *citation.Citation) (LegislationURI, error) {
	if citationRef == nil {
		return LegislationURI{}, fmt.Errorf("citation is nil")
	}

	legislationType, err := citationTypeToLegislationSlug(citationRef)
	if err != nil {
		return LegislationURI{}, err
	}

	docYear := citationRef.Components.DocYear
	docNumber := citationRef.Components.DocNumber

	if docYear == "" {
		return LegislationURI{}, fmt.Errorf("citation missing year component")
	}

	if docNumber == "" {
		return LegislationURI{}, fmt.Errorf("citation missing number component (chapter number for Acts, SI number for Statutory Instruments)")
	}

	legislationURI := LegislationURI{
		LegislationType: legislationType,
		Year:            docYear,
		Number:          docNumber,
	}

	// If the citation includes a section reference, populate the Section field.
	if citationRef.Components.Section != "" {
		legislationURI.Section = citationRef.Components.Section
	}

	return legislationURI, nil
}

// GenerateSectionURI creates a legislation.gov.uk URI for a section reference
// within a known Act. This requires the Act's year, chapter number, and section number.
//
// Example: GenerateSectionURI("2018", "12", "6")
//
//	→ https://www.legislation.gov.uk/id/ukpga/2018/12/section/6
func GenerateSectionURI(actYear string, actChapter string, sectionNumber string) LegislationURI {
	return LegislationURI{
		LegislationType: LegislationTypeUKPGA,
		Year:            actYear,
		Number:          actChapter,
		Section:         sectionNumber,
	}
}

// citationTypeToLegislationSlug maps a citation type to its legislation.gov.uk
// URI path slug.
func citationTypeToLegislationSlug(citationRef *citation.Citation) (LegislationType, error) {
	switch citationRef.Type {
	case citation.CitationTypeStatute:
		if citationRef.Components.CodeName == "ukact" {
			return LegislationTypeUKPGA, nil
		}
		return "", fmt.Errorf("unsupported statute code name for UK legislation: %q", citationRef.Components.CodeName)

	case citation.CitationTypeRegulation:
		return LegislationTypeUKSI, nil

	default:
		return "", fmt.Errorf("unsupported citation type for legislation.gov.uk URI: %s", citationRef.Type)
	}
}
