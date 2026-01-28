package eurlex

import (
	"fmt"
	"strconv"

	"github.com/coolbeans/regula/pkg/citation"
)

// GenerateCELEX creates a CELEX number from a parsed EU citation.
// Returns an error if the citation lacks required components (year, number)
// or has an unsupported citation type.
//
// CELEX format: {Sector}{Year}{TypeCode}{PaddedNumber}
// Example: Regulation (EU) 2016/679 -> "32016R0679"
func GenerateCELEX(citationRef *citation.Citation) (CELEXNumber, error) {
	if citationRef == nil {
		return CELEXNumber{}, fmt.Errorf("citation cannot be nil")
	}

	docYear := citationRef.Components.DocYear
	docNumber := citationRef.Components.DocNumber

	if docYear == "" {
		return CELEXNumber{}, fmt.Errorf("citation missing required year component")
	}
	if docNumber == "" {
		return CELEXNumber{}, fmt.Errorf("citation missing required number component")
	}

	typeCode, err := citationTypeToDocumentTypeCode(citationRef.Type)
	if err != nil {
		return CELEXNumber{}, err
	}

	normalizedYear := normalizeYear(docYear)
	paddedNumber := padCELEXNumber(docNumber)

	return CELEXNumber{
		Sector:   SectorLegislation,
		Year:     normalizedYear,
		TypeCode: typeCode,
		Number:   paddedNumber,
	}, nil
}

// citationTypeToDocumentTypeCode maps citation.CitationType to the CELEX DocumentTypeCode.
func citationTypeToDocumentTypeCode(citationType citation.CitationType) (DocumentTypeCode, error) {
	switch citationType {
	case citation.CitationTypeRegulation:
		return TypeRegulation, nil
	case citation.CitationTypeDirective:
		return TypeDirective, nil
	case citation.CitationTypeDecision:
		return TypeDecision, nil
	default:
		return "", fmt.Errorf("unsupported citation type for CELEX generation: %s", citationType)
	}
}

// normalizeYear converts a 2-digit year to 4-digit.
// Uses 1958 as the cutoff (year the EU/EEC was founded):
// - Years >= 58 are interpreted as 19xx (e.g., "95" -> "1995")
// - Years < 58 are interpreted as 20xx (e.g., "16" -> "2016")
// 4-digit years pass through unchanged.
func normalizeYear(yearString string) string {
	if len(yearString) == 2 {
		yearValue, err := strconv.Atoi(yearString)
		if err != nil {
			return yearString
		}
		if yearValue >= 58 {
			return "19" + yearString
		}
		return "20" + yearString
	}
	return yearString
}

// padCELEXNumber pads a document number to 4 digits with leading zeros.
// Example: "679" -> "0679", "46" -> "0046", "1" -> "0001"
func padCELEXNumber(numberString string) string {
	for len(numberString) < 4 {
		numberString = "0" + numberString
	}
	return numberString
}
