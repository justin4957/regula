// Package citation provides an extensible citation parser interface for
// parsing legal citations across different jurisdictions and citation formats.
package citation

import (
	"github.com/coolbeans/regula/pkg/types"
)

// CitationType classifies the kind of legal citation.
type CitationType string

const (
	CitationTypeStatute    CitationType = "statute"
	CitationTypeRegulation CitationType = "regulation"
	CitationTypeDirective  CitationType = "directive"
	CitationTypeDecision   CitationType = "decision"
	CitationTypeTreaty     CitationType = "treaty"
	CitationTypeCase       CitationType = "case"
	CitationTypeCode       CitationType = "code"
	CitationTypeUnknown    CitationType = "unknown"
)

// TemporalRefKind classifies temporal qualifiers in a citation.
type TemporalRefKind string

const (
	TemporalAsAmended    TemporalRefKind = "as_amended"
	TemporalInForceOn    TemporalRefKind = "in_force_on"
	TemporalRepealed     TemporalRefKind = "repealed"
	TemporalOriginal     TemporalRefKind = "original"
	TemporalConsolidated TemporalRefKind = "consolidated"
)

// TemporalRef captures temporal qualifiers in a citation.
// Examples: "as amended by Regulation (EU) 2018/1725", "in force on 2025-01-01".
type TemporalRef struct {
	Kind        TemporalRefKind `json:"kind"`
	Description string          `json:"description"`
	Date        *types.Date     `json:"date,omitempty"`
}

// Citation represents a parsed legal citation with full metadata.
type Citation struct {
	// Raw text as found in the source document.
	RawText string `json:"raw_text"`

	// Parsed classification.
	Type         CitationType `json:"type"`
	Jurisdiction string       `json:"jurisdiction"`

	// Document identification.
	Document    string `json:"document"`
	Subdivision string `json:"subdivision,omitempty"`

	// Temporal qualifier (optional).
	Temporal *TemporalRef `json:"temporal,omitempty"`

	// Confidence score (0.0 to 1.0) from the parser.
	Confidence float64 `json:"confidence"`

	// Which parser produced this citation.
	Parser string `json:"parser"`

	// Position in source text (for alignment with existing Reference type).
	TextOffset int `json:"text_offset,omitempty"`
	TextLength int `json:"text_length,omitempty"`

	// Parsed components for structured access.
	Components CitationComponents `json:"components,omitempty"`
}

// CitationComponents holds granular parsed fields that vary by citation type.
type CitationComponents struct {
	// EU-style components.
	DocYear   string `json:"doc_year,omitempty"`
	DocNumber string `json:"doc_number,omitempty"`

	// Article-level components.
	ArticleNumber   int    `json:"article_number,omitempty"`
	ParagraphNumber int    `json:"paragraph_number,omitempty"`
	PointLetter     string `json:"point_letter,omitempty"`
	ChapterNumber   string `json:"chapter_number,omitempty"`

	// US-style components.
	Title     string `json:"title,omitempty"`
	Section   string `json:"section,omitempty"`
	CodeName  string `json:"code_name,omitempty"`
	PublicLaw string `json:"public_law,omitempty"`
}
