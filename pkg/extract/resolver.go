package extract

import (
	"fmt"
	"sort"
	"strings"
)

// ResolutionStatus indicates the outcome of resolving a reference.
type ResolutionStatus string

const (
	ResolutionResolved   ResolutionStatus = "resolved"
	ResolutionAmbiguous  ResolutionStatus = "ambiguous"
	ResolutionNotFound   ResolutionStatus = "not_found"
	ResolutionExternal   ResolutionStatus = "external"
	ResolutionPartial    ResolutionStatus = "partial" // Article found but paragraph/point not verified
	ResolutionSelfRef    ResolutionStatus = "self_ref"
	ResolutionRangeRef   ResolutionStatus = "range_ref"
)

// ResolutionConfidence indicates how confident we are in the resolution.
type ResolutionConfidence float64

const (
	ConfidenceHigh   ResolutionConfidence = 1.0
	ConfidenceMedium ResolutionConfidence = 0.75
	ConfidenceLow    ResolutionConfidence = 0.5
	ConfidenceNone   ResolutionConfidence = 0.0
)

// ResolvedReference represents a reference after resolution.
type ResolvedReference struct {
	// Original reference
	Original *Reference `json:"original"`

	// Resolution result
	Status     ResolutionStatus     `json:"status"`
	Confidence ResolutionConfidence `json:"confidence"`

	// Resolved target(s)
	TargetURI  string   `json:"target_uri,omitempty"`
	TargetURIs []string `json:"target_uris,omitempty"` // For ranges or ambiguous refs

	// Resolution metadata
	Reason          string `json:"reason,omitempty"`
	AlternativeURIs []string `json:"alternative_uris,omitempty"`

	// Context used for resolution
	ContextArticle  int    `json:"context_article,omitempty"`
	ContextChapter  string `json:"context_chapter,omitempty"`
}

// ReferenceResolver resolves detected references to provision URIs.
type ReferenceResolver struct {
	baseURI string
	regID   string

	// Available provisions for resolution
	articles   map[int]bool         // Article numbers that exist
	chapters   map[string]bool      // Chapter numbers (Roman numerals) that exist
	sections   map[string]bool      // Section identifiers that exist
	paragraphs map[string]bool      // Paragraph identifiers (Art:Para) that exist
	points     map[string]bool      // Point identifiers (Art:Para:Point) that exist

	// Article to chapter mapping for context resolution
	articleChapter map[int]string
}

// NewReferenceResolver creates a new resolver.
func NewReferenceResolver(baseURI, regID string) *ReferenceResolver {
	if !strings.HasSuffix(baseURI, "#") && !strings.HasSuffix(baseURI, "/") {
		baseURI += "#"
	}
	return &ReferenceResolver{
		baseURI:        baseURI,
		regID:          regID,
		articles:       make(map[int]bool),
		chapters:       make(map[string]bool),
		sections:       make(map[string]bool),
		paragraphs:     make(map[string]bool),
		points:         make(map[string]bool),
		articleChapter: make(map[int]string),
	}
}

// IndexDocument indexes all provisions in a document for resolution.
func (r *ReferenceResolver) IndexDocument(doc *Document) {
	if doc == nil {
		return
	}

	for _, chapter := range doc.Chapters {
		r.chapters[chapter.Number] = true

		// Index sections
		for _, section := range chapter.Sections {
			sectionKey := chapter.Number + ":" + itoa(section.Number)
			r.sections[sectionKey] = true

			// Index articles in sections
			for _, article := range section.Articles {
				r.indexArticle(article, chapter.Number)
			}
		}

		// Index articles directly in chapter
		for _, article := range chapter.Articles {
			r.indexArticle(article, chapter.Number)
		}
	}
}

// indexArticle indexes an article and its sub-elements.
func (r *ReferenceResolver) indexArticle(article *Article, chapterNum string) {
	r.articles[article.Number] = true
	r.articleChapter[article.Number] = chapterNum

	// Index paragraphs
	for _, para := range article.Paragraphs {
		paraKey := fmt.Sprintf("%d:%d", article.Number, para.Number)
		r.paragraphs[paraKey] = true

		// Index points
		for _, point := range para.Points {
			pointKey := fmt.Sprintf("%d:%d:%s", article.Number, para.Number, point.Letter)
			r.points[pointKey] = true
		}
	}
}

// ResolveAll resolves all references in a document.
func (r *ReferenceResolver) ResolveAll(refs []*Reference) []*ResolvedReference {
	resolved := make([]*ResolvedReference, 0, len(refs))
	for _, ref := range refs {
		resolved = append(resolved, r.Resolve(ref))
	}
	return resolved
}

// Resolve resolves a single reference.
func (r *ReferenceResolver) Resolve(ref *Reference) *ResolvedReference {
	result := &ResolvedReference{
		Original:       ref,
		ContextArticle: ref.SourceArticle,
	}

	// Get context chapter if available
	if chap, ok := r.articleChapter[ref.SourceArticle]; ok {
		result.ContextChapter = chap
	}

	// Handle external references
	if ref.Type == ReferenceTypeExternal {
		result.Status = ResolutionExternal
		result.Confidence = ConfidenceHigh
		result.TargetURI = r.buildExternalURI(ref)
		result.Reason = "External reference to " + ref.ExternalDoc
		return result
	}

	// Handle self-references (reference within same article)
	if ref.Target == TargetParagraph || ref.Target == TargetPoint {
		// Check if this is a relative reference within the same article
		if ref.ArticleNum == 0 {
			return r.resolveRelativeReference(ref, result)
		}
	}

	// Handle by target type
	switch ref.Target {
	case TargetArticle:
		return r.resolveArticleReference(ref, result)
	case TargetParagraph:
		return r.resolveParagraphReference(ref, result)
	case TargetPoint:
		return r.resolvePointReference(ref, result)
	case TargetChapter:
		return r.resolveChapterReference(ref, result)
	case TargetSection:
		return r.resolveSectionReference(ref, result)
	default:
		result.Status = ResolutionNotFound
		result.Confidence = ConfidenceNone
		result.Reason = "Unknown target type: " + string(ref.Target)
		return result
	}
}

// resolveArticleReference resolves an article reference.
func (r *ReferenceResolver) resolveArticleReference(ref *Reference, result *ResolvedReference) *ResolvedReference {
	// Handle range references (e.g., "Articles 13 to 18")
	if ref.SubRef == "range" {
		return r.resolveArticleRange(ref, result)
	}

	// Check if article exists
	if !r.articles[ref.ArticleNum] {
		result.Status = ResolutionNotFound
		result.Confidence = ConfidenceNone
		result.Reason = fmt.Sprintf("Article %d does not exist", ref.ArticleNum)
		return result
	}

	// Build target URI
	targetURI := r.articleURI(ref.ArticleNum)

	// If paragraph is specified, try to resolve more specifically
	if ref.ParagraphNum > 0 {
		paraKey := fmt.Sprintf("%d:%d", ref.ArticleNum, ref.ParagraphNum)
		if r.paragraphs[paraKey] {
			targetURI = r.paragraphURI(ref.ArticleNum, ref.ParagraphNum)

			// If point is specified, try to resolve even more specifically
			if ref.PointLetter != "" {
				pointKey := fmt.Sprintf("%d:%d:%s", ref.ArticleNum, ref.ParagraphNum, ref.PointLetter)
				if r.points[pointKey] {
					targetURI = r.pointURI(ref.ArticleNum, ref.ParagraphNum, ref.PointLetter)
					result.Status = ResolutionResolved
					result.Confidence = ConfidenceHigh
					result.Reason = "Fully resolved to specific point"
				} else {
					result.Status = ResolutionPartial
					result.Confidence = ConfidenceMedium
					result.Reason = fmt.Sprintf("Point (%s) not found in Article %d(%d)", ref.PointLetter, ref.ArticleNum, ref.ParagraphNum)
				}
			} else {
				result.Status = ResolutionResolved
				result.Confidence = ConfidenceHigh
				result.Reason = "Resolved to specific paragraph"
			}
		} else {
			result.Status = ResolutionPartial
			result.Confidence = ConfidenceMedium
			result.Reason = fmt.Sprintf("Paragraph %d not found in Article %d", ref.ParagraphNum, ref.ArticleNum)
		}
	} else {
		result.Status = ResolutionResolved
		result.Confidence = ConfidenceHigh
		result.Reason = "Resolved to article"
	}

	result.TargetURI = targetURI
	return result
}

// resolveArticleRange resolves a reference to multiple articles.
func (r *ReferenceResolver) resolveArticleRange(ref *Reference, result *ResolvedReference) *ResolvedReference {
	// Parse the range from identifier (e.g., "Articles 13-18")
	startArticle := ref.ArticleNum

	// Extract end article from raw text
	endArticle := startArticle
	parts := strings.FieldsFunc(ref.RawText, func(c rune) bool {
		return c == ' ' || c == '-'
	})
	for i, part := range parts {
		if part == "to" || part == "and" {
			if i+1 < len(parts) {
				if num := mustAtoiSafe(parts[i+1]); num > 0 {
					endArticle = num
				}
			}
		}
		if num := mustAtoiSafe(part); num > startArticle {
			endArticle = num
		}
	}

	// Collect all articles in range
	var targetURIs []string
	var missing []int
	for artNum := startArticle; artNum <= endArticle; artNum++ {
		if r.articles[artNum] {
			targetURIs = append(targetURIs, r.articleURI(artNum))
		} else {
			missing = append(missing, artNum)
		}
	}

	result.TargetURIs = targetURIs
	result.Status = ResolutionRangeRef

	if len(missing) == 0 {
		result.Confidence = ConfidenceHigh
		result.Reason = fmt.Sprintf("All %d articles in range resolved", len(targetURIs))
	} else if len(targetURIs) > 0 {
		result.Confidence = ConfidenceMedium
		result.Reason = fmt.Sprintf("%d articles resolved, %d not found", len(targetURIs), len(missing))
	} else {
		result.Status = ResolutionNotFound
		result.Confidence = ConfidenceNone
		result.Reason = "No articles in range found"
	}

	return result
}

// resolveParagraphReference resolves a paragraph reference.
func (r *ReferenceResolver) resolveParagraphReference(ref *Reference, result *ResolvedReference) *ResolvedReference {
	// If article is specified, use it
	articleNum := ref.ArticleNum
	if articleNum == 0 {
		// Use context article (paragraph within same article)
		articleNum = ref.SourceArticle
	}

	paraKey := fmt.Sprintf("%d:%d", articleNum, ref.ParagraphNum)
	if r.paragraphs[paraKey] {
		result.Status = ResolutionResolved
		result.Confidence = ConfidenceHigh
		result.TargetURI = r.paragraphURI(articleNum, ref.ParagraphNum)
		result.Reason = "Resolved to paragraph"
	} else if r.articles[articleNum] {
		result.Status = ResolutionPartial
		result.Confidence = ConfidenceMedium
		result.TargetURI = r.articleURI(articleNum)
		result.Reason = fmt.Sprintf("Paragraph %d not found; resolved to article", ref.ParagraphNum)
	} else {
		result.Status = ResolutionNotFound
		result.Confidence = ConfidenceNone
		result.Reason = fmt.Sprintf("Article %d does not exist", articleNum)
	}

	return result
}

// resolvePointReference resolves a point reference.
func (r *ReferenceResolver) resolvePointReference(ref *Reference, result *ResolvedReference) *ResolvedReference {
	// For relative point references, use context
	articleNum := ref.ArticleNum
	if articleNum == 0 {
		articleNum = ref.SourceArticle
	}
	paraNum := ref.ParagraphNum
	if paraNum == 0 {
		paraNum = 1 // Default to paragraph 1
	}

	// Handle point range
	if ref.SubRef == "range" {
		return r.resolvePointRange(ref, result, articleNum, paraNum)
	}

	pointKey := fmt.Sprintf("%d:%d:%s", articleNum, paraNum, ref.PointLetter)
	if r.points[pointKey] {
		result.Status = ResolutionResolved
		result.Confidence = ConfidenceHigh
		result.TargetURI = r.pointURI(articleNum, paraNum, ref.PointLetter)
		result.Reason = "Resolved to point"
	} else {
		// Try to find in any paragraph
		for p := 1; p <= 10; p++ {
			testKey := fmt.Sprintf("%d:%d:%s", articleNum, p, ref.PointLetter)
			if r.points[testKey] {
				result.Status = ResolutionResolved
				result.Confidence = ConfidenceMedium
				result.TargetURI = r.pointURI(articleNum, p, ref.PointLetter)
				result.Reason = fmt.Sprintf("Point found in paragraph %d (context suggested %d)", p, paraNum)
				return result
			}
		}

		// Fall back to article
		if r.articles[articleNum] {
			result.Status = ResolutionPartial
			result.Confidence = ConfidenceLow
			result.TargetURI = r.articleURI(articleNum)
			result.Reason = fmt.Sprintf("Point (%s) not found; resolved to article", ref.PointLetter)
		} else {
			result.Status = ResolutionNotFound
			result.Confidence = ConfidenceNone
			result.Reason = "Cannot resolve point reference"
		}
	}

	return result
}

// resolvePointRange resolves a range of points.
func (r *ReferenceResolver) resolvePointRange(ref *Reference, result *ResolvedReference, articleNum, paraNum int) *ResolvedReference {
	startLetter := ref.PointLetter
	endLetter := startLetter

	// Extract end letter from raw text
	if strings.Contains(ref.RawText, "to") {
		parts := strings.Split(ref.RawText, "to")
		if len(parts) >= 2 {
			// Extract letter from "(x)"
			for _, c := range parts[1] {
				if c >= 'a' && c <= 'z' {
					endLetter = string(c)
					break
				}
			}
		}
	}

	// Collect all points in range
	var targetURIs []string
	for letter := startLetter[0]; letter <= endLetter[0]; letter++ {
		pointKey := fmt.Sprintf("%d:%d:%s", articleNum, paraNum, string(letter))
		if r.points[pointKey] {
			targetURIs = append(targetURIs, r.pointURI(articleNum, paraNum, string(letter)))
		}
	}

	result.TargetURIs = targetURIs
	result.Status = ResolutionRangeRef

	if len(targetURIs) > 0 {
		result.Confidence = ConfidenceHigh
		result.Reason = fmt.Sprintf("%d points in range resolved", len(targetURIs))
	} else {
		result.Status = ResolutionPartial
		result.Confidence = ConfidenceLow
		result.TargetURI = r.articleURI(articleNum)
		result.Reason = "No points in range found; resolved to article"
	}

	return result
}

// resolveChapterReference resolves a chapter reference.
func (r *ReferenceResolver) resolveChapterReference(ref *Reference, result *ResolvedReference) *ResolvedReference {
	if r.chapters[ref.ChapterNum] {
		result.Status = ResolutionResolved
		result.Confidence = ConfidenceHigh
		result.TargetURI = r.chapterURI(ref.ChapterNum)
		result.Reason = "Resolved to chapter"
	} else {
		result.Status = ResolutionNotFound
		result.Confidence = ConfidenceNone
		result.Reason = fmt.Sprintf("Chapter %s does not exist", ref.ChapterNum)
	}
	return result
}

// resolveSectionReference resolves a section reference.
func (r *ReferenceResolver) resolveSectionReference(ref *Reference, result *ResolvedReference) *ResolvedReference {
	// US-style section references (e.g., Section 1798.100) map to articles
	// The ArticleNum field contains the normalized article number (e.g., 100 for Section 1798.100)
	if ref.ArticleNum > 0 && r.isUSStyleSectionRef(ref) {
		return r.resolveUSStyleSectionReference(ref, result)
	}

	// EU-style section reference resolution
	// Try to find section using context chapter
	contextChapter := result.ContextChapter

	// Check explicit chapter if available, otherwise use context
	sectionKey := contextChapter + ":" + itoa(ref.SectionNum)
	if r.sections[sectionKey] {
		result.Status = ResolutionResolved
		result.Confidence = ConfidenceHigh
		result.TargetURI = r.sectionURI(contextChapter, ref.SectionNum)
		result.Reason = "Resolved to section in context chapter"
		return result
	}

	// Try to find in any chapter
	for chapNum := range r.chapters {
		testKey := chapNum + ":" + itoa(ref.SectionNum)
		if r.sections[testKey] {
			result.Status = ResolutionResolved
			result.Confidence = ConfidenceMedium
			result.TargetURI = r.sectionURI(chapNum, ref.SectionNum)
			result.Reason = fmt.Sprintf("Section found in Chapter %s", chapNum)

			// Record alternatives
			for otherChap := range r.chapters {
				if otherChap != chapNum {
					otherKey := otherChap + ":" + itoa(ref.SectionNum)
					if r.sections[otherKey] {
						result.AlternativeURIs = append(result.AlternativeURIs, r.sectionURI(otherChap, ref.SectionNum))
					}
				}
			}
			if len(result.AlternativeURIs) > 0 {
				result.Status = ResolutionAmbiguous
				result.Confidence = ConfidenceLow
				result.Reason = "Multiple sections found with same number"
			}
			return result
		}
	}

	result.Status = ResolutionNotFound
	result.Confidence = ConfidenceNone
	result.Reason = fmt.Sprintf("Section %d not found", ref.SectionNum)
	return result
}

// isUSStyleSectionRef checks if this is a US-style section reference (Section 1798.xxx).
func (r *ReferenceResolver) isUSStyleSectionRef(ref *Reference) bool {
	// US-style section refs have SectionNum > 1000000 (codePrefix*1000 + sectionNum pattern)
	// or contain subdivision references
	return ref.SectionNum >= 1000000 || ref.SubRef == "subdivision" || ref.SubRef == "paragraph" || ref.SubRef == "range"
}

// resolveUSStyleSectionReference resolves US-style California Code section references.
// Section 1798.100 maps to Article 100, Section 1798.185(a) maps to Article 185 paragraph a.
func (r *ReferenceResolver) resolveUSStyleSectionReference(ref *Reference, result *ResolvedReference) *ResolvedReference {
	// For US-style sections, ArticleNum contains the normalized article number
	articleNum := ref.ArticleNum

	// Handle range references (e.g., "Sections 1798.100 to 1798.199")
	if ref.SubRef == "range" {
		return r.resolveUSStyleSectionRange(ref, result)
	}

	// Check if article exists
	if !r.articles[articleNum] {
		result.Status = ResolutionNotFound
		result.Confidence = ConfidenceNone
		result.Reason = fmt.Sprintf("Section %s not found (Article %d does not exist)", ref.Identifier, articleNum)
		return result
	}

	// Build target URI
	targetURI := r.articleURI(articleNum)

	// Handle subdivision references (e.g., Section 1798.100(a))
	if ref.PointLetter != "" {
		// In US regulations, subdivisions often map to paragraphs
		// Try to find a matching paragraph
		for paraNum := 1; paraNum <= 20; paraNum++ {
			pointKey := fmt.Sprintf("%d:%d:%s", articleNum, paraNum, ref.PointLetter)
			if r.points[pointKey] {
				targetURI = r.pointURI(articleNum, paraNum, ref.PointLetter)
				result.Status = ResolutionResolved
				result.Confidence = ConfidenceHigh
				result.TargetURI = targetURI
				result.Reason = fmt.Sprintf("Resolved to Article %d subdivision (%s)", articleNum, ref.PointLetter)
				return result
			}
		}

		// Subdivision not found as point, resolve to article with partial confidence
		result.Status = ResolutionPartial
		result.Confidence = ConfidenceMedium
		result.TargetURI = targetURI
		result.Reason = fmt.Sprintf("Resolved to Article %d (subdivision %s not indexed)", articleNum, ref.PointLetter)
		return result
	}

	// Handle paragraph references within subdivisions (e.g., Section 1798.185(a)(1))
	if ref.ParagraphNum > 0 && ref.PointLetter != "" {
		// This is handled above in the subdivision case with ParagraphNum
		result.Status = ResolutionPartial
		result.Confidence = ConfidenceMedium
		result.TargetURI = targetURI
		result.Reason = fmt.Sprintf("Resolved to Article %d (paragraph %d of subdivision %s not fully indexed)",
			articleNum, ref.ParagraphNum, ref.PointLetter)
		return result
	}

	// Simple section reference resolved to article
	result.Status = ResolutionResolved
	result.Confidence = ConfidenceHigh
	result.TargetURI = targetURI
	result.Reason = fmt.Sprintf("Section 1798.%d resolved to Article %d", articleNum, articleNum)
	return result
}

// resolveUSStyleSectionRange resolves a range of US-style sections.
func (r *ReferenceResolver) resolveUSStyleSectionRange(ref *Reference, result *ResolvedReference) *ResolvedReference {
	// Parse range from identifier (e.g., "Sections 1798.100-1798.199")
	startArticle := ref.ArticleNum

	// Extract end article from identifier
	endArticle := startArticle
	parts := strings.FieldsFunc(ref.Identifier, func(c rune) bool {
		return c == '-' || c == '.'
	})
	for i, part := range parts {
		if num := mustAtoiSafe(part); num > 0 {
			if i > 0 && num > startArticle && num < 1000 {
				endArticle = num
			}
		}
	}

	// If we couldn't parse the end, try from raw text
	if endArticle == startArticle {
		rawParts := strings.FieldsFunc(ref.RawText, func(c rune) bool {
			return c == ' ' || c == '-' || c == '.'
		})
		for _, part := range rawParts {
			if num := mustAtoiSafe(part); num > startArticle && num < 1000 {
				endArticle = num
			}
		}
	}

	// Collect all articles in range
	var targetURIs []string
	var found, missing int
	for artNum := startArticle; artNum <= endArticle; artNum++ {
		if r.articles[artNum] {
			targetURIs = append(targetURIs, r.articleURI(artNum))
			found++
		} else {
			missing++
		}
	}

	result.TargetURIs = targetURIs
	result.Status = ResolutionRangeRef

	if found > 0 && missing == 0 {
		result.Confidence = ConfidenceHigh
		result.Reason = fmt.Sprintf("All %d sections in range resolved to articles", found)
	} else if found > 0 {
		result.Confidence = ConfidenceMedium
		result.Reason = fmt.Sprintf("%d sections resolved, %d not found", found, missing)
	} else {
		result.Status = ResolutionNotFound
		result.Confidence = ConfidenceNone
		result.Reason = "No sections in range found"
	}

	return result
}

// resolveRelativeReference resolves references like "paragraph 1" without article context.
func (r *ReferenceResolver) resolveRelativeReference(ref *Reference, result *ResolvedReference) *ResolvedReference {
	// Use source article as context
	articleNum := ref.SourceArticle

	switch ref.Target {
	case TargetParagraph:
		ref.ArticleNum = articleNum
		return r.resolveParagraphReference(ref, result)
	case TargetPoint:
		ref.ArticleNum = articleNum
		return r.resolvePointReference(ref, result)
	default:
		result.Status = ResolutionNotFound
		result.Confidence = ConfidenceNone
		result.Reason = "Cannot resolve relative reference"
		return result
	}
}

// URI builders (must match GraphBuilder)

func (r *ReferenceResolver) articleURI(number int) string {
	return r.baseURI + r.regID + ":Art" + itoa(number)
}

func (r *ReferenceResolver) paragraphURI(articleNum, paraNum int) string {
	return r.baseURI + r.regID + ":Art" + itoa(articleNum) + "(" + itoa(paraNum) + ")"
}

func (r *ReferenceResolver) pointURI(articleNum, paraNum int, letter string) string {
	return r.baseURI + r.regID + ":Art" + itoa(articleNum) + "(" + itoa(paraNum) + ")(" + letter + ")"
}

func (r *ReferenceResolver) chapterURI(number string) string {
	return r.baseURI + r.regID + ":Chapter" + number
}

func (r *ReferenceResolver) sectionURI(chapterNum string, sectionNum int) string {
	return r.baseURI + r.regID + ":Chapter" + chapterNum + ":Section" + itoa(sectionNum)
}

func (r *ReferenceResolver) buildExternalURI(ref *Reference) string {
	switch ref.Target {
	case TargetDirective:
		return fmt.Sprintf("urn:eu:directive:%s/%s", ref.DocYear, ref.DocNumber)
	case TargetRegulation:
		// Handle both EU and US regulations
		if ref.ExternalDoc == "USC" {
			return fmt.Sprintf("urn:us:usc:%s/%d", ref.DocNumber, ref.SectionNum)
		} else if ref.ExternalDoc == "CFR" {
			return fmt.Sprintf("urn:us:cfr:%s/%d", ref.DocNumber, ref.SectionNum)
		} else if ref.ExternalDoc == "PublicLaw" {
			return fmt.Sprintf("urn:us:pl:%s-%s", ref.DocYear, ref.DocNumber)
		}
		return fmt.Sprintf("urn:eu:regulation:%s/%s", ref.DocYear, ref.DocNumber)
	case TargetTreaty:
		return fmt.Sprintf("urn:eu:treaty:%s", ref.Identifier)
	case TargetDecision:
		return fmt.Sprintf("urn:eu:decision:%s/%s", ref.DocYear, ref.DocNumber)
	case TargetSection:
		// Handle external US-style section references
		if ref.ExternalDoc == "CalTitle" {
			return fmt.Sprintf("urn:us:ca:title%s/sec%d", ref.DocNumber, ref.SectionNum)
		}
		if ref.ExternalDoc == "USC" {
			return fmt.Sprintf("urn:us:usc:%s/%d", ref.DocNumber, ref.SectionNum)
		}
		if ref.ExternalDoc == "USAct" {
			actSlug := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(ref.DocNumber), " ", "-"))
			return fmt.Sprintf("urn:us:act:%s/sec%d", actSlug, ref.SectionNum)
		}
		return "urn:external:" + ref.Identifier
	default:
		return "urn:external:" + ref.Identifier
	}
}

// mustAtoiSafe converts string to int, returning 0 on failure.
func mustAtoiSafe(s string) int {
	// Remove any non-digit characters
	var digits strings.Builder
	for _, c := range s {
		if c >= '0' && c <= '9' {
			digits.WriteRune(c)
		}
	}
	if digits.Len() == 0 {
		return 0
	}
	return mustAtoi(digits.String())
}

// ResolutionReport provides a summary of reference resolution results.
type ResolutionReport struct {
	TotalReferences int `json:"total_references"`

	// Resolution status counts
	Resolved   int `json:"resolved"`
	Partial    int `json:"partial"`
	Ambiguous  int `json:"ambiguous"`
	NotFound   int `json:"not_found"`
	External   int `json:"external"`
	SelfRef    int `json:"self_ref"`
	RangeRef   int `json:"range_ref"`

	// Confidence distribution
	HighConfidence   int `json:"high_confidence"`
	MediumConfidence int `json:"medium_confidence"`
	LowConfidence    int `json:"low_confidence"`

	// Calculated metrics
	ResolutionRate   float64 `json:"resolution_rate"`   // (Resolved + Partial + RangeRef) / (Total - External)
	ConfidenceRate   float64 `json:"confidence_rate"`   // High / Total

	// Details for reporting
	UnresolvedRefs []*ResolvedReference `json:"unresolved_refs,omitempty"`
	AmbiguousRefs  []*ResolvedReference `json:"ambiguous_refs,omitempty"`
}

// GenerateReport generates a resolution report from resolved references.
func GenerateReport(resolved []*ResolvedReference) *ResolutionReport {
	report := &ResolutionReport{
		TotalReferences: len(resolved),
	}

	var internalCount int

	for _, ref := range resolved {
		switch ref.Status {
		case ResolutionResolved:
			report.Resolved++
		case ResolutionPartial:
			report.Partial++
		case ResolutionAmbiguous:
			report.Ambiguous++
			report.AmbiguousRefs = append(report.AmbiguousRefs, ref)
		case ResolutionNotFound:
			report.NotFound++
			report.UnresolvedRefs = append(report.UnresolvedRefs, ref)
		case ResolutionExternal:
			report.External++
		case ResolutionSelfRef:
			report.SelfRef++
		case ResolutionRangeRef:
			report.RangeRef++
		}

		// Count confidence
		switch {
		case ref.Confidence >= ConfidenceHigh:
			report.HighConfidence++
		case ref.Confidence >= ConfidenceMedium:
			report.MediumConfidence++
		default:
			report.LowConfidence++
		}

		// Count internal references for rate calculation
		if ref.Original.Type == ReferenceTypeInternal {
			internalCount++
		}
	}

	// Calculate rates
	if internalCount > 0 {
		successfulResolutions := report.Resolved + report.Partial + report.RangeRef
		report.ResolutionRate = float64(successfulResolutions) / float64(internalCount)
	}
	if report.TotalReferences > 0 {
		report.ConfidenceRate = float64(report.HighConfidence) / float64(report.TotalReferences)
	}

	// Sort unresolved by source article for easier review
	sort.Slice(report.UnresolvedRefs, func(i, j int) bool {
		return report.UnresolvedRefs[i].Original.SourceArticle < report.UnresolvedRefs[j].Original.SourceArticle
	})

	return report
}

// String returns a human-readable summary of the resolution report.
func (r *ResolutionReport) String() string {
	var sb strings.Builder

	sb.WriteString("Reference Resolution Report\n")
	sb.WriteString("===========================\n\n")

	sb.WriteString(fmt.Sprintf("Total references: %d\n\n", r.TotalReferences))

	sb.WriteString("Resolution Status:\n")
	sb.WriteString(fmt.Sprintf("  Resolved:   %d\n", r.Resolved))
	sb.WriteString(fmt.Sprintf("  Partial:    %d\n", r.Partial))
	sb.WriteString(fmt.Sprintf("  Range refs: %d\n", r.RangeRef))
	sb.WriteString(fmt.Sprintf("  Ambiguous:  %d\n", r.Ambiguous))
	sb.WriteString(fmt.Sprintf("  Not found:  %d\n", r.NotFound))
	sb.WriteString(fmt.Sprintf("  External:   %d\n\n", r.External))

	sb.WriteString("Confidence Distribution:\n")
	sb.WriteString(fmt.Sprintf("  High:   %d\n", r.HighConfidence))
	sb.WriteString(fmt.Sprintf("  Medium: %d\n", r.MediumConfidence))
	sb.WriteString(fmt.Sprintf("  Low:    %d\n\n", r.LowConfidence))

	sb.WriteString(fmt.Sprintf("Resolution rate: %.1f%% (internal refs)\n", r.ResolutionRate*100))
	sb.WriteString(fmt.Sprintf("High confidence: %.1f%%\n\n", r.ConfidenceRate*100))

	if len(r.UnresolvedRefs) > 0 {
		sb.WriteString("Unresolved References:\n")
		for _, ref := range r.UnresolvedRefs {
			sb.WriteString(fmt.Sprintf("  - Article %d: %q - %s\n",
				ref.Original.SourceArticle, ref.Original.RawText, ref.Reason))
		}
		sb.WriteString("\n")
	}

	if len(r.AmbiguousRefs) > 0 {
		sb.WriteString("Ambiguous References:\n")
		for _, ref := range r.AmbiguousRefs {
			sb.WriteString(fmt.Sprintf("  - Article %d: %q - %s\n",
				ref.Original.SourceArticle, ref.Original.RawText, ref.Reason))
		}
	}

	return sb.String()
}
