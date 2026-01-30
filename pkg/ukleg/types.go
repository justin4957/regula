// Package ukleg provides a connector to UK legislation.gov.uk for generating
// legislation URIs, validating UK legislation references, and fetching metadata.
package ukleg

import (
	"time"
)

// LegislationType represents the type of UK legislation on legislation.gov.uk.
// See: https://www.legislation.gov.uk/developer
type LegislationType string

const (
	LegislationTypeUKPGA LegislationType = "ukpga" // UK Public General Acts
	LegislationTypeUKSI  LegislationType = "uksi"  // UK Statutory Instruments
)

// LegislationBaseURL is the base URL for human-readable legislation.gov.uk pages.
const LegislationBaseURL = "https://www.legislation.gov.uk/"

// LegislationIDBaseURL is the base URL for stable legislation.gov.uk identifier URIs
// that support content negotiation (XML, RDF, HTML).
const LegislationIDBaseURL = "https://www.legislation.gov.uk/id/"

// LegislationURI is a structured representation of a legislation.gov.uk URI.
// Format: https://www.legislation.gov.uk/{type}/{year}/{number}
// Example: https://www.legislation.gov.uk/ukpga/2018/12
type LegislationURI struct {
	LegislationType LegislationType `json:"legislation_type"`
	Year            string          `json:"year"`
	Number          string          `json:"number"`
	Section         string          `json:"section,omitempty"` // Optional section-level reference
}

// String returns the full legislation.gov.uk URI for human-readable access.
// If Section is set, returns the /id/ variant with section path appended.
func (legislationURI LegislationURI) String() string {
	if legislationURI.Section != "" {
		return LegislationIDBaseURL + string(legislationURI.LegislationType) + "/" +
			legislationURI.Year + "/" + legislationURI.Number + "/section/" + legislationURI.Section
	}
	return LegislationBaseURL + string(legislationURI.LegislationType) + "/" +
		legislationURI.Year + "/" + legislationURI.Number
}

// IDString returns the stable /id/ URI variant for API and content negotiation use.
func (legislationURI LegislationURI) IDString() string {
	baseURI := LegislationIDBaseURL + string(legislationURI.LegislationType) + "/" +
		legislationURI.Year + "/" + legislationURI.Number
	if legislationURI.Section != "" {
		return baseURI + "/section/" + legislationURI.Section
	}
	return baseURI
}

// ValidationResult captures the outcome of a URI validation via HEAD request.
type ValidationResult struct {
	URI        string    `json:"uri"`
	Valid      bool      `json:"valid"`
	StatusCode int       `json:"status_code"`
	CheckedAt  time.Time `json:"checked_at"`
	Error      string    `json:"error,omitempty"`
}

// DocumentMetadata holds metadata fetched from legislation.gov.uk about a document.
type DocumentMetadata struct {
	LegislationType string `json:"legislation_type"`
	Year            string `json:"year"`
	Number          string `json:"number"`
	Title           string `json:"title,omitempty"`
	DateEnacted     string `json:"date_enacted,omitempty"`
	URI             string `json:"uri,omitempty"`
}
