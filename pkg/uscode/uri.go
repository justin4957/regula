package uscode

import (
	"fmt"
	"strings"

	"github.com/coolbeans/regula/pkg/citation"
)

// GenerateUSCURI creates a USC URI from a citation.
// The citation must have Type=CitationTypeCode and Components.CodeName="USC".
func GenerateUSCURI(citationRef *citation.Citation) (*USCURI, error) {
	if citationRef == nil {
		return nil, fmt.Errorf("citation is nil")
	}

	if citationRef.Components.CodeName != "USC" {
		return nil, fmt.Errorf("citation is not a USC reference (got %q)", citationRef.Components.CodeName)
	}

	title := citationRef.Components.Title
	section := citationRef.Components.Section

	if title == "" {
		return nil, fmt.Errorf("USC citation missing title")
	}
	if section == "" {
		return nil, fmt.Errorf("USC citation missing section")
	}

	return &USCURI{
		Title:   title,
		Section: section,
	}, nil
}

// GenerateCFRURI creates a CFR URI from a citation.
// The citation must have Type=CitationTypeCode and Components.CodeName="CFR".
func GenerateCFRURI(citationRef *citation.Citation) (*CFRURI, error) {
	if citationRef == nil {
		return nil, fmt.Errorf("citation is nil")
	}

	if citationRef.Components.CodeName != "CFR" {
		return nil, fmt.Errorf("citation is not a CFR reference (got %q)", citationRef.Components.CodeName)
	}

	title := citationRef.Components.Title
	section := citationRef.Components.Section

	if title == "" {
		return nil, fmt.Errorf("CFR citation missing title")
	}
	if section == "" {
		return nil, fmt.Errorf("CFR citation missing section/part")
	}

	// Parse section which may be in format "164" or "164.502"
	var part, subSection string
	if idx := strings.Index(section, "."); idx != -1 {
		part = section[:idx]
		subSection = section[idx+1:]
	} else {
		part = section
	}

	return &CFRURI{
		Title:   title,
		Part:    part,
		Section: subSection,
	}, nil
}

// ParseUSCNumber parses a USC citation string into a structured USCNumber.
// Supported formats:
//   - "42 U.S.C. § 1983"
//   - "15 U.S.C. Section 1681"
//   - "42 USC 1983"
func ParseUSCNumber(citation string) (*USCNumber, error) {
	citation = strings.TrimSpace(citation)
	if citation == "" {
		return nil, fmt.Errorf("citation is empty")
	}

	// Try to extract title and section using common patterns
	// Pattern: {title} U.S.C. {separator} {section}
	var title, section string

	// Normalize the citation
	normalized := strings.ReplaceAll(citation, "U.S.C.", "USC")
	normalized = strings.ReplaceAll(normalized, "§", "")
	normalized = strings.ReplaceAll(normalized, "Section", "")
	normalized = strings.ReplaceAll(normalized, "Sec.", "")

	parts := strings.Fields(normalized)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid USC citation format: %s", citation)
	}

	// First part should be the title number
	title = parts[0]

	// Find "USC" and take the next part as section
	for i, part := range parts {
		if part == "USC" && i+1 < len(parts) {
			section = parts[i+1]
			break
		}
	}

	if section == "" {
		return nil, fmt.Errorf("could not parse section from USC citation: %s", citation)
	}

	return &USCNumber{
		Title:   title,
		Section: section,
	}, nil
}

// ParseCFRNumber parses a CFR citation string into a structured CFRNumber.
// Supported formats:
//   - "45 C.F.R. Part 164"
//   - "45 C.F.R. § 164.502"
//   - "45 CFR 164"
func ParseCFRNumber(citationStr string) (*CFRNumber, error) {
	citationStr = strings.TrimSpace(citationStr)
	if citationStr == "" {
		return nil, fmt.Errorf("citation is empty")
	}

	// Normalize the citation
	normalized := strings.ReplaceAll(citationStr, "C.F.R.", "CFR")
	normalized = strings.ReplaceAll(normalized, "§", "")
	normalized = strings.ReplaceAll(normalized, "Part", "")

	parts := strings.Fields(normalized)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid CFR citation format: %s", citationStr)
	}

	// First part should be the title number
	title := parts[0]

	// Find "CFR" and take the next part as part/section
	var partSection string
	for i, part := range parts {
		if part == "CFR" && i+1 < len(parts) {
			partSection = parts[i+1]
			break
		}
	}

	if partSection == "" {
		return nil, fmt.Errorf("could not parse part from CFR citation: %s", citationStr)
	}

	// Parse part.section format
	var part, section string
	if idx := strings.Index(partSection, "."); idx != -1 {
		part = partSection[:idx]
		section = partSection[idx+1:]
	} else {
		part = partSection
	}

	return &CFRNumber{
		Title:   title,
		Part:    part,
		Section: section,
	}, nil
}
