package citation

import (
	"github.com/coolbeans/regula/pkg/extract"
)

// CitationFromReference converts an existing extract.Reference to a Citation.
// This enables the new citation system to consume legacy reference extraction results.
func CitationFromReference(ref *extract.Reference) *Citation {
	citationType := mapReferenceTargetToCitationType(ref.Target)
	referenceType := mapReferenceTypeToInternal(ref.Type)

	citation := &Citation{
		RawText:      ref.RawText,
		Type:         citationType,
		Jurisdiction: inferJurisdictionFromReference(ref),
		Document:     ref.ExternalDoc,
		Subdivision:  ref.Identifier,
		Confidence:   1.0,
		Parser:       "legacy-reference-extractor",
		TextOffset:   ref.TextOffset,
		TextLength:   ref.TextLength,
		Components: CitationComponents{
			DocYear:         ref.DocYear,
			DocNumber:       ref.DocNumber,
			ArticleNumber:   ref.ArticleNum,
			ParagraphNumber: ref.ParagraphNum,
			PointLetter:     ref.PointLetter,
			ChapterNumber:   ref.ChapterNum,
		},
	}

	// For internal references, populate Document from the target type.
	if referenceType == "internal" && citation.Document == "" {
		citation.Document = ""
	}

	return citation
}

// ReferenceFromCitation converts a Citation back to an extract.Reference.
// This enables backward-compatible integration with the existing graph builder
// and resolver systems.
func ReferenceFromCitation(citation *Citation, sourceArticle int) *extract.Reference {
	return &extract.Reference{
		Type:          mapCitationToReferenceType(citation),
		Target:        mapCitationTypeToReferenceTarget(citation.Type),
		RawText:       citation.RawText,
		Identifier:    citation.Subdivision,
		SourceArticle: sourceArticle,
		TextOffset:    citation.TextOffset,
		TextLength:    citation.TextLength,
		ArticleNum:    citation.Components.ArticleNumber,
		ParagraphNum:  citation.Components.ParagraphNumber,
		PointLetter:   citation.Components.PointLetter,
		ChapterNum:    citation.Components.ChapterNumber,
		ExternalDoc:   citation.Document,
		DocYear:       citation.Components.DocYear,
		DocNumber:     citation.Components.DocNumber,
	}
}

// BatchCitationsFromReferences converts a slice of References to Citations.
func BatchCitationsFromReferences(refs []*extract.Reference) []*Citation {
	citations := make([]*Citation, 0, len(refs))
	for _, ref := range refs {
		citations = append(citations, CitationFromReference(ref))
	}
	return citations
}

// BatchReferencesFromCitations converts a slice of Citations to References.
func BatchReferencesFromCitations(citations []*Citation, sourceArticle int) []*extract.Reference {
	refs := make([]*extract.Reference, 0, len(citations))
	for _, cit := range citations {
		refs = append(refs, ReferenceFromCitation(cit, sourceArticle))
	}
	return refs
}

// mapReferenceTargetToCitationType maps extract.ReferenceTarget to CitationType.
func mapReferenceTargetToCitationType(target extract.ReferenceTarget) CitationType {
	switch target {
	case extract.TargetDirective:
		return CitationTypeDirective
	case extract.TargetRegulation:
		return CitationTypeRegulation
	case extract.TargetDecision:
		return CitationTypeDecision
	case extract.TargetTreaty:
		return CitationTypeTreaty
	case extract.TargetArticle, extract.TargetParagraph, extract.TargetPoint,
		extract.TargetChapter, extract.TargetSection:
		return CitationTypeStatute
	default:
		return CitationTypeUnknown
	}
}

// mapCitationTypeToReferenceTarget maps CitationType to extract.ReferenceTarget.
func mapCitationTypeToReferenceTarget(citationType CitationType) extract.ReferenceTarget {
	switch citationType {
	case CitationTypeDirective:
		return extract.TargetDirective
	case CitationTypeRegulation:
		return extract.TargetRegulation
	case CitationTypeDecision:
		return extract.TargetDecision
	case CitationTypeTreaty:
		return extract.TargetTreaty
	default:
		return extract.TargetArticle
	}
}

// mapCitationToReferenceType determines internal vs external reference type.
func mapCitationToReferenceType(citation *Citation) extract.ReferenceType {
	switch citation.Type {
	case CitationTypeRegulation, CitationTypeDirective,
		CitationTypeDecision, CitationTypeTreaty:
		return extract.ReferenceTypeExternal
	default:
		return extract.ReferenceTypeInternal
	}
}

// mapReferenceTypeToInternal returns "internal" or "external" as a string.
func mapReferenceTypeToInternal(refType extract.ReferenceType) string {
	if refType == extract.ReferenceTypeExternal {
		return "external"
	}
	return "internal"
}

// inferJurisdictionFromReference determines jurisdiction from reference metadata.
func inferJurisdictionFromReference(ref *extract.Reference) string {
	switch ref.Target {
	case extract.TargetDirective, extract.TargetRegulation,
		extract.TargetDecision, extract.TargetTreaty:
		return "EU"
	default:
		// Infer from external doc field if available.
		switch ref.ExternalDoc {
		case "USC", "CFR", "PublicLaw", "CalTitle":
			return "US"
		default:
			return ""
		}
	}
}
