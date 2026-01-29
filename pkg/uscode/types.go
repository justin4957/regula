// Package uscode provides a connector to the United States Code (uscode.house.gov)
// for generating USC URIs and validating US statutory references.
package uscode

import (
	"time"
)

// USCNumber represents a structured US Code citation.
// Format: {Title} U.S.C. § {Section}
// Example: "42 U.S.C. § 1983" = Title 42, Section 1983
type USCNumber struct {
	Title      string `json:"title"`
	Section    string `json:"section"`
	Subsection string `json:"subsection,omitempty"`
}

// String returns the canonical USC string representation.
func (uscNumber USCNumber) String() string {
	result := uscNumber.Title + " U.S.C. § " + uscNumber.Section
	if uscNumber.Subsection != "" {
		result += uscNumber.Subsection
	}
	return result
}

// CFRNumber represents a structured Code of Federal Regulations citation.
// Format: {Title} C.F.R. § {Part}.{Section}
// Example: "45 C.F.R. § 164.502" = Title 45, Part 164, Section 502
type CFRNumber struct {
	Title   string `json:"title"`
	Part    string `json:"part"`
	Section string `json:"section,omitempty"`
}

// String returns the canonical CFR string representation.
func (cfrNumber CFRNumber) String() string {
	result := cfrNumber.Title + " C.F.R. § " + cfrNumber.Part
	if cfrNumber.Section != "" {
		result += "." + cfrNumber.Section
	}
	return result
}

// USCURI represents a US Code URI for the House of Representatives website.
// Base URL: https://uscode.house.gov/view.xhtml
type USCURI struct {
	Title   string `json:"title"`
	Section string `json:"section"`
}

// USCBaseURL is the base URL for US Code URIs.
const USCBaseURL = "https://uscode.house.gov/view.xhtml"

// String returns the full USC URI.
func (uscURI USCURI) String() string {
	return USCBaseURL + "?req=granuleid:USC-prelim-title" + uscURI.Title + "-section" + uscURI.Section + "&edition=prelim"
}

// CFRURI represents a Code of Federal Regulations URI for eCFR.
// Base URL: https://www.ecfr.gov/current/title-{title}/part-{part}
type CFRURI struct {
	Title   string `json:"title"`
	Part    string `json:"part"`
	Section string `json:"section,omitempty"`
}

// CFRBaseURL is the base URL for eCFR URIs.
const CFRBaseURL = "https://www.ecfr.gov/current/"

// String returns the full eCFR URI.
func (cfrURI CFRURI) String() string {
	result := CFRBaseURL + "title-" + cfrURI.Title + "/part-" + cfrURI.Part
	if cfrURI.Section != "" {
		result += "/section-" + cfrURI.Part + "." + cfrURI.Section
	}
	return result
}

// ValidationResult captures the outcome of a URI validation via HEAD request.
type ValidationResult struct {
	URI        string    `json:"uri"`
	Valid      bool      `json:"valid"`
	StatusCode int       `json:"status_code"`
	CheckedAt  time.Time `json:"checked_at"`
	Error      string    `json:"error,omitempty"`
}

// DocumentMetadata holds metadata about a US statutory document.
type DocumentMetadata struct {
	Title       string `json:"title"`
	Section     string `json:"section"`
	DisplayName string `json:"display_name,omitempty"`
	Heading     string `json:"heading,omitempty"`
}
