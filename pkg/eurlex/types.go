// Package eurlex provides a connector to EUR-Lex for generating CELEX numbers,
// ELI URIs, and validating EU legislation references.
package eurlex

import (
	"time"
)

// DocumentSector represents the CELEX sector code.
// See: https://eur-lex.europa.eu/content/tools/TableOfSectors/types_of_documents_in_eurlex.html
type DocumentSector string

const (
	SectorTreaties                DocumentSector = "1"
	SectorInternationalAgreements DocumentSector = "2"
	SectorLegislation             DocumentSector = "3"
	SectorComplementaryLegislation DocumentSector = "4"
	SectorPreparatoryActs         DocumentSector = "5"
	SectorCaseLaw                 DocumentSector = "6"
)

// DocumentTypeCode represents the CELEX document type indicator within a sector.
type DocumentTypeCode string

const (
	TypeRegulation DocumentTypeCode = "R"
	TypeDirective  DocumentTypeCode = "L"
	TypeDecision   DocumentTypeCode = "D"
)

// CELEXNumber is a structured representation of a CELEX identifier.
// Format: {Sector}{Year}{TypeCode}{PaddedNumber}
// Example: "32016R0679" = Sector 3, Year 2016, Regulation, Number 0679
type CELEXNumber struct {
	Sector   DocumentSector   `json:"sector"`
	Year     string           `json:"year"`
	TypeCode DocumentTypeCode `json:"type_code"`
	Number   string           `json:"number"`
}

// String returns the canonical CELEX string representation.
func (celexNumber CELEXNumber) String() string {
	return string(celexNumber.Sector) + celexNumber.Year + string(celexNumber.TypeCode) + celexNumber.Number
}

// ELIURI represents a European Legislation Identifier URI.
// Format: http://data.europa.eu/eli/{type}/{year}/{number}/oj
type ELIURI struct {
	TypeSlug string `json:"type_slug"`
	Year     string `json:"year"`
	Number   string `json:"number"`
}

// ELIBaseURL is the base URL for ELI URIs.
const ELIBaseURL = "http://data.europa.eu/eli/"

// String returns the full ELI URI.
func (eliURI ELIURI) String() string {
	return ELIBaseURL + eliURI.TypeSlug + "/" + eliURI.Year + "/" + eliURI.Number + "/oj"
}

// ValidationResult captures the outcome of a URI validation via HEAD request.
type ValidationResult struct {
	URI        string    `json:"uri"`
	Valid      bool      `json:"valid"`
	StatusCode int       `json:"status_code"`
	CheckedAt  time.Time `json:"checked_at"`
	Error      string    `json:"error,omitempty"`
}

// DocumentMetadata holds metadata fetched from EUR-Lex about a document.
type DocumentMetadata struct {
	CELEX          string   `json:"celex"`
	Title          string   `json:"title"`
	DateOfDocument string   `json:"date_of_document,omitempty"`
	DateInForce    string   `json:"date_in_force,omitempty"`
	ELI            string   `json:"eli,omitempty"`
	Languages      []string `json:"languages,omitempty"`
}
