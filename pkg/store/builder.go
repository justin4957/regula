package store

import (
	"fmt"
	"strings"

	"github.com/coolbeans/regula/pkg/extract"
)

// GraphBuilder converts extracted regulatory documents into RDF triples.
type GraphBuilder struct {
	store   *TripleStore
	baseURI string
	regID   string
}

// BuildStats contains statistics about the graph building process.
type BuildStats struct {
	TotalTriples      int `json:"total_triples"`
	ArticleTriples    int `json:"article_triples"`
	ChapterTriples    int `json:"chapter_triples"`
	SectionTriples    int `json:"section_triples"`
	RecitalTriples    int `json:"recital_triples"`
	DefinitionTriples int `json:"definition_triples"`
	ReferenceTriples  int `json:"reference_triples"`
	SemanticTriples   int `json:"semantic_triples"`
	Articles          int `json:"articles"`
	Chapters          int `json:"chapters"`
	Sections          int `json:"sections"`
	Recitals          int `json:"recitals"`
	Definitions       int `json:"definitions"`
	References        int `json:"references"`
	Rights            int `json:"rights"`
	Obligations       int `json:"obligations"`
}

// NewGraphBuilder creates a new GraphBuilder with the given store and base URI.
func NewGraphBuilder(store *TripleStore, baseURI string) *GraphBuilder {
	if !strings.HasSuffix(baseURI, "#") && !strings.HasSuffix(baseURI, "/") {
		baseURI += "#"
	}
	return &GraphBuilder{
		store:   store,
		baseURI: baseURI,
	}
}

// Build converts a parsed document into RDF triples and adds them to the store.
func (b *GraphBuilder) Build(doc *extract.Document) (*BuildStats, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}

	stats := &BuildStats{}

	// Determine regulation ID from identifier
	b.regID = b.extractRegID(doc.Identifier)

	// Build regulation node
	b.buildRegulation(doc, stats)

	// Build preamble and recitals
	if doc.Preamble != nil {
		b.buildPreamble(doc.Preamble, stats)
	}

	// Build chapters and their contents
	for _, chapter := range doc.Chapters {
		b.buildChapter(chapter, stats)
	}

	// Build definitions from extracted definitions
	for _, def := range doc.Definitions {
		b.buildDefinition(def, stats)
	}

	stats.TotalTriples = b.store.Count()
	return stats, nil
}

// BuildWithExtractors builds the graph using additional extractors for richer data.
func (b *GraphBuilder) BuildWithExtractors(
	doc *extract.Document,
	defExtractor *extract.DefinitionExtractor,
	refExtractor *extract.ReferenceExtractor,
) (*BuildStats, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}

	stats := &BuildStats{}

	// Determine regulation ID
	b.regID = b.extractRegID(doc.Identifier)

	// Build basic structure
	b.buildRegulation(doc, stats)

	if doc.Preamble != nil {
		b.buildPreamble(doc.Preamble, stats)
	}

	for _, chapter := range doc.Chapters {
		b.buildChapter(chapter, stats)
	}

	// Build rich definitions if extractor provided
	if defExtractor != nil {
		definitions := defExtractor.ExtractDefinitions(doc)
		for _, def := range definitions {
			b.buildDefinedTerm(def, stats)
		}
	} else {
		// Fall back to basic definitions
		for _, def := range doc.Definitions {
			b.buildDefinition(def, stats)
		}
	}

	// Build references if extractor provided
	if refExtractor != nil {
		refs := refExtractor.ExtractFromDocument(doc)
		for _, ref := range refs {
			b.buildReference(ref, stats)
		}
	}

	stats.TotalTriples = b.store.Count()
	return stats, nil
}

// extractRegID extracts a short regulation ID from the identifier.
func (b *GraphBuilder) extractRegID(identifier string) string {
	// Try to extract from identifiers like "(EU) 2016/679"
	if strings.Contains(identifier, "2016/679") {
		return "GDPR"
	}
	if strings.Contains(identifier, "/") {
		parts := strings.Split(identifier, "/")
		if len(parts) >= 2 {
			return "Reg" + strings.TrimSpace(parts[len(parts)-1])
		}
	}
	// Default
	return "Regulation"
}

// URI builders

func (b *GraphBuilder) regulationURI() string {
	return b.baseURI + b.regID
}

func (b *GraphBuilder) chapterURI(number string) string {
	return b.baseURI + b.regID + ":Chapter" + number
}

func (b *GraphBuilder) sectionURI(chapterNum string, sectionNum int) string {
	return b.baseURI + b.regID + ":Chapter" + chapterNum + ":Section" + itoa(sectionNum)
}

func (b *GraphBuilder) articleURI(number int) string {
	return b.baseURI + b.regID + ":Art" + itoa(number)
}

func (b *GraphBuilder) paragraphURI(articleNum, paraNum int) string {
	return b.baseURI + b.regID + ":Art" + itoa(articleNum) + "(" + itoa(paraNum) + ")"
}

func (b *GraphBuilder) pointURI(articleNum, paraNum int, letter string) string {
	return b.baseURI + b.regID + ":Art" + itoa(articleNum) + "(" + itoa(paraNum) + ")(" + letter + ")"
}

func (b *GraphBuilder) recitalURI(number int) string {
	return b.baseURI + b.regID + ":Recital" + itoa(number)
}

func (b *GraphBuilder) preambleURI() string {
	return b.baseURI + b.regID + ":Preamble"
}

func (b *GraphBuilder) definitionURI(normalizedTerm string) string {
	safeTerm := b.normalizeTerm(normalizedTerm)
	return b.baseURI + b.regID + ":Term:" + safeTerm
}

func (b *GraphBuilder) referenceURI(sourceArticle int, offset int) string {
	return b.baseURI + b.regID + ":Ref:Art" + itoa(sourceArticle) + ":" + itoa(offset)
}

// normalizeTerm creates a URI-safe version of a term.
func (b *GraphBuilder) normalizeTerm(term string) string {
	var result strings.Builder
	for _, c := range strings.ToLower(term) {
		if c == ' ' {
			result.WriteRune('_')
		} else if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			result.WriteRune(c)
		}
	}
	return result.String()
}

// Build methods

func (b *GraphBuilder) buildRegulation(doc *extract.Document, stats *BuildStats) {
	uri := b.regulationURI()

	// Type
	var docClass string
	switch doc.Type {
	case extract.DocumentTypeRegulation:
		docClass = ClassRegulation
	case extract.DocumentTypeDirective:
		docClass = ClassDirective
	case extract.DocumentTypeDecision:
		docClass = ClassDecision
	default:
		docClass = ClassRegulation
	}
	b.store.Add(uri, RDFType, docClass)

	// Metadata
	if doc.Title != "" {
		b.store.Add(uri, PropTitle, doc.Title)
	}
	if doc.Identifier != "" {
		b.store.Add(uri, PropIdentifier, doc.Identifier)
	}

	// Label for easy querying
	b.store.Add(uri, RDFSLabel, b.regID)
}

func (b *GraphBuilder) buildPreamble(preamble *extract.Preamble, stats *BuildStats) {
	preambleURI := b.preambleURI()
	regURI := b.regulationURI()

	b.store.Add(preambleURI, RDFType, ClassPreamble)
	b.store.Add(preambleURI, PropPartOf, regURI)
	b.store.Add(regURI, PropContains, preambleURI)

	// Build recitals
	for _, recital := range preamble.Recitals {
		b.buildRecital(recital, preambleURI, stats)
	}
}

func (b *GraphBuilder) buildRecital(recital *extract.Recital, preambleURI string, stats *BuildStats) {
	uri := b.recitalURI(recital.Number)

	b.store.Add(uri, RDFType, ClassRecital)
	b.store.Add(uri, PropNumber, itoa(recital.Number))
	b.store.Add(uri, PropPartOf, preambleURI)
	b.store.Add(preambleURI, PropHasRecital, uri)

	if recital.Text != "" {
		b.store.Add(uri, PropText, recital.Text)
	}

	stats.Recitals++
	stats.RecitalTriples += 4 // type, number, partOf, hasRecital
	if recital.Text != "" {
		stats.RecitalTriples++
	}
}

func (b *GraphBuilder) buildChapter(chapter *extract.Chapter, stats *BuildStats) {
	uri := b.chapterURI(chapter.Number)
	regURI := b.regulationURI()

	b.store.Add(uri, RDFType, ClassChapter)
	b.store.Add(uri, PropNumber, chapter.Number)
	if chapter.Title != "" {
		b.store.Add(uri, PropTitle, chapter.Title)
	}

	// Hierarchy
	b.store.Add(uri, PropPartOf, regURI)
	b.store.Add(uri, PropBelongsTo, regURI)
	b.store.Add(regURI, PropHasChapter, uri)
	b.store.Add(regURI, PropContains, uri)

	stats.Chapters++
	stats.ChapterTriples += 6 // type, number, partOf, belongsTo, hasChapter, contains
	if chapter.Title != "" {
		stats.ChapterTriples++
	}

	// Build sections
	for _, section := range chapter.Sections {
		b.buildSection(section, chapter.Number, uri, stats)
	}

	// Build articles directly in chapter (not in sections)
	for _, article := range chapter.Articles {
		b.buildArticle(article, uri, stats)
	}
}

func (b *GraphBuilder) buildSection(section *extract.Section, chapterNum string, chapterURI string, stats *BuildStats) {
	uri := b.sectionURI(chapterNum, section.Number)
	regURI := b.regulationURI()

	b.store.Add(uri, RDFType, ClassSection)
	b.store.Add(uri, PropNumber, itoa(section.Number))
	if section.Title != "" {
		b.store.Add(uri, PropTitle, section.Title)
	}

	// Hierarchy
	b.store.Add(uri, PropPartOf, chapterURI)
	b.store.Add(uri, PropBelongsTo, regURI)
	b.store.Add(chapterURI, PropHasSection, uri)
	b.store.Add(chapterURI, PropContains, uri)

	stats.Sections++
	stats.SectionTriples += 6 // type, number, partOf, belongsTo, hasSection, contains
	if section.Title != "" {
		stats.SectionTriples++
	}

	// Build articles in section
	for _, article := range section.Articles {
		b.buildArticle(article, uri, stats)
	}
}

func (b *GraphBuilder) buildArticle(article *extract.Article, parentURI string, stats *BuildStats) {
	uri := b.articleURI(article.Number)
	regURI := b.regulationURI()

	b.store.Add(uri, RDFType, ClassArticle)
	b.store.Add(uri, PropNumber, itoa(article.Number))
	if article.Title != "" {
		b.store.Add(uri, PropTitle, article.Title)
	}
	if article.Text != "" {
		b.store.Add(uri, PropText, article.Text)
	}

	// Hierarchy
	b.store.Add(uri, PropPartOf, parentURI)
	b.store.Add(uri, PropBelongsTo, regURI)
	b.store.Add(parentURI, PropHasArticle, uri)
	b.store.Add(parentURI, PropContains, uri)

	stats.Articles++
	stats.ArticleTriples += 6 // type, number, partOf, belongsTo, hasArticle, contains
	if article.Title != "" {
		stats.ArticleTriples++
	}
	if article.Text != "" {
		stats.ArticleTriples++
	}

	// Build paragraphs
	for _, para := range article.Paragraphs {
		b.buildParagraph(para, article.Number, uri, stats)
	}
}

func (b *GraphBuilder) buildParagraph(para *extract.Paragraph, articleNum int, articleURI string, stats *BuildStats) {
	uri := b.paragraphURI(articleNum, para.Number)
	regURI := b.regulationURI()

	b.store.Add(uri, RDFType, ClassParagraph)
	b.store.Add(uri, PropNumber, itoa(para.Number))
	if para.Text != "" {
		b.store.Add(uri, PropText, para.Text)
	}

	// Hierarchy
	b.store.Add(uri, PropPartOf, articleURI)
	b.store.Add(uri, PropBelongsTo, regURI)
	b.store.Add(articleURI, PropHasParagraph, uri)
	b.store.Add(articleURI, PropContains, uri)

	// Build points
	for _, point := range para.Points {
		b.buildPoint(point, articleNum, para.Number, uri, stats)
	}
}

func (b *GraphBuilder) buildPoint(point *extract.Point, articleNum, paraNum int, paraURI string, stats *BuildStats) {
	uri := b.pointURI(articleNum, paraNum, point.Letter)
	regURI := b.regulationURI()

	b.store.Add(uri, RDFType, ClassPoint)
	b.store.Add(uri, PropNumber, point.Letter)
	if point.Text != "" {
		b.store.Add(uri, PropText, point.Text)
	}

	// Hierarchy
	b.store.Add(uri, PropPartOf, paraURI)
	b.store.Add(uri, PropBelongsTo, regURI)
	b.store.Add(paraURI, PropHasPoint, uri)
	b.store.Add(paraURI, PropContains, uri)
}

func (b *GraphBuilder) buildDefinition(def *extract.Definition, stats *BuildStats) {
	uri := b.definitionURI(def.Term)
	regURI := b.regulationURI()
	article4URI := b.articleURI(4)

	b.store.Add(uri, RDFType, ClassDefinedTerm)
	b.store.Add(uri, PropNumber, itoa(def.Number))
	b.store.Add(uri, PropTerm, def.Term)
	b.store.Add(uri, PropNormalizedTerm, strings.ToLower(def.Term))
	if def.Text != "" {
		b.store.Add(uri, PropDefinition, def.Text)
	}

	// Links
	b.store.Add(uri, PropDefinedIn, article4URI)
	b.store.Add(uri, PropBelongsTo, regURI)
	b.store.Add(article4URI, PropDefines, uri)

	stats.Definitions++
	stats.DefinitionTriples += 7 // type, number, term, normalized, definedIn, belongsTo, defines
	if def.Text != "" {
		stats.DefinitionTriples++
	}
}

func (b *GraphBuilder) buildDefinedTerm(def *extract.DefinedTerm, stats *BuildStats) {
	uri := b.definitionURI(def.NormalizedTerm)
	regURI := b.regulationURI()
	articleURI := b.articleURI(def.ArticleRef)

	b.store.Add(uri, RDFType, ClassDefinedTerm)
	b.store.Add(uri, PropNumber, itoa(def.Number))
	b.store.Add(uri, PropTerm, def.Term)
	b.store.Add(uri, PropNormalizedTerm, def.NormalizedTerm)
	if def.Definition != "" {
		b.store.Add(uri, PropDefinition, def.Definition)
	}
	if def.Scope != "" {
		b.store.Add(uri, PropScope, def.Scope)
	}

	// Links
	b.store.Add(uri, PropDefinedIn, articleURI)
	b.store.Add(uri, PropBelongsTo, regURI)
	b.store.Add(articleURI, PropDefines, uri)

	// Sub-points
	for _, sp := range def.SubPoints {
		subURI := uri + ":" + sp.Letter
		b.store.Add(subURI, RDFType, ClassSubPoint)
		b.store.Add(subURI, PropNumber, sp.Letter)
		b.store.Add(subURI, PropText, sp.Text)
		b.store.Add(subURI, PropPartOf, uri)
		b.store.Add(uri, PropContains, subURI)
	}

	stats.Definitions++
	stats.DefinitionTriples += 7
	if def.Definition != "" {
		stats.DefinitionTriples++
	}
	if def.Scope != "" {
		stats.DefinitionTriples++
	}
	stats.DefinitionTriples += len(def.SubPoints) * 5
}

func (b *GraphBuilder) buildReference(ref *extract.Reference, stats *BuildStats) {
	uri := b.referenceURI(ref.SourceArticle, ref.TextOffset)
	sourceURI := b.articleURI(ref.SourceArticle)
	regURI := b.regulationURI()

	b.store.Add(uri, RDFType, ClassReference)
	b.store.Add(uri, PropText, ref.RawText)
	b.store.Add(uri, PropIdentifier, ref.Identifier)

	// Source location
	b.store.Add(uri, PropSourceOffset, itoa(ref.TextOffset))
	b.store.Add(uri, PropSourceLength, itoa(ref.TextLength))

	// Link to source article
	b.store.Add(uri, PropPartOf, sourceURI)
	b.store.Add(uri, PropBelongsTo, regURI)

	// Reference type
	if ref.Type == extract.ReferenceTypeInternal {
		// Try to resolve internal reference
		if ref.ArticleNum > 0 {
			targetURI := b.articleURI(ref.ArticleNum)
			b.store.Add(sourceURI, PropReferences, targetURI)
			b.store.Add(targetURI, PropReferencedBy, sourceURI)

			// More specific reference
			if ref.ParagraphNum > 0 {
				targetURI = b.paragraphURI(ref.ArticleNum, ref.ParagraphNum)
				if ref.PointLetter != "" {
					targetURI = b.pointURI(ref.ArticleNum, ref.ParagraphNum, ref.PointLetter)
				}
				b.store.Add(uri, PropRefersToArticle, targetURI)
			}
		} else if ref.ChapterNum != "" {
			targetURI := b.chapterURI(ref.ChapterNum)
			b.store.Add(sourceURI, PropRefersToChapter, targetURI)
		}
	} else {
		// External reference - store as literal
		b.store.Add(uri, PropExternalRef, ref.Identifier)
		if ref.ExternalDoc != "" {
			b.store.Add(uri, "reg:externalDocType", ref.ExternalDoc)
		}
	}

	stats.References++
	stats.ReferenceTriples += 7 // type, text, identifier, offset, length, partOf, belongsTo
}

// BuildWithResolver builds the graph using a reference resolver for enhanced resolution tracking.
func (b *GraphBuilder) BuildWithResolver(
	doc *extract.Document,
	defExtractor *extract.DefinitionExtractor,
	refExtractor *extract.ReferenceExtractor,
	resolver *extract.ReferenceResolver,
) (*BuildStats, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}

	stats := &BuildStats{}

	// Determine regulation ID
	b.regID = b.extractRegID(doc.Identifier)

	// Build basic structure
	b.buildRegulation(doc, stats)

	if doc.Preamble != nil {
		b.buildPreamble(doc.Preamble, stats)
	}

	for _, chapter := range doc.Chapters {
		b.buildChapter(chapter, stats)
	}

	// Build rich definitions if extractor provided
	if defExtractor != nil {
		definitions := defExtractor.ExtractDefinitions(doc)
		for _, def := range definitions {
			b.buildDefinedTerm(def, stats)
		}
	} else {
		for _, def := range doc.Definitions {
			b.buildDefinition(def, stats)
		}
	}

	// Build references with resolution if resolver provided
	if refExtractor != nil {
		refs := refExtractor.ExtractFromDocument(doc)

		if resolver != nil {
			// Index the document for resolution
			resolver.IndexDocument(doc)

			// Resolve and build each reference
			resolved := resolver.ResolveAll(refs)
			for _, res := range resolved {
				b.buildResolvedReference(res, stats)
			}
		} else {
			// Fall back to basic reference building
			for _, ref := range refs {
				b.buildReference(ref, stats)
			}
		}
	}

	stats.TotalTriples = b.store.Count()
	return stats, nil
}

// buildResolvedReference builds a reference with resolution metadata.
func (b *GraphBuilder) buildResolvedReference(res *extract.ResolvedReference, stats *BuildStats) {
	ref := res.Original
	uri := b.referenceURI(ref.SourceArticle, ref.TextOffset)
	sourceURI := b.articleURI(ref.SourceArticle)
	regURI := b.regulationURI()

	// Basic reference properties
	b.store.Add(uri, RDFType, ClassReference)
	b.store.Add(uri, PropText, ref.RawText)
	b.store.Add(uri, PropIdentifier, ref.Identifier)
	b.store.Add(uri, PropSourceOffset, itoa(ref.TextOffset))
	b.store.Add(uri, PropSourceLength, itoa(ref.TextLength))
	b.store.Add(uri, PropPartOf, sourceURI)
	b.store.Add(uri, PropBelongsTo, regURI)

	// Resolution metadata
	b.store.Add(uri, PropResolutionStatus, string(res.Status))
	b.store.Add(uri, PropResolutionConfidence, fmt.Sprintf("%.2f", res.Confidence))
	if res.Reason != "" {
		b.store.Add(uri, PropResolutionReason, res.Reason)
	}

	// Link to resolved target(s)
	if res.TargetURI != "" {
		b.store.Add(uri, PropResolvedTarget, res.TargetURI)

		// Create direct reference link for resolved internal references
		if res.Status == extract.ResolutionResolved || res.Status == extract.ResolutionPartial {
			b.store.Add(sourceURI, PropReferences, res.TargetURI)
			b.store.Add(res.TargetURI, PropReferencedBy, sourceURI)
		}
	}

	// For range references, link to all targets
	for _, targetURI := range res.TargetURIs {
		b.store.Add(uri, PropResolvedTarget, targetURI)
		b.store.Add(sourceURI, PropReferences, targetURI)
		b.store.Add(targetURI, PropReferencedBy, sourceURI)
	}

	// Record alternative targets for ambiguous refs
	for _, altURI := range res.AlternativeURIs {
		b.store.Add(uri, PropAlternativeTarget, altURI)
	}

	// External references
	if res.Status == extract.ResolutionExternal {
		b.store.Add(uri, PropExternalRef, ref.Identifier)
		if ref.ExternalDoc != "" {
			b.store.Add(uri, "reg:externalDocType", ref.ExternalDoc)
		}
	}

	stats.References++
	stats.ReferenceTriples += 10 // base triples plus resolution metadata
}

// buildSemanticAnnotation builds triples for a semantic annotation (right or obligation).
func (b *GraphBuilder) buildSemanticAnnotation(ann *extract.SemanticAnnotation, stats *BuildStats) {
	articleURI := b.articleURI(ann.ArticleNum)
	regURI := b.regulationURI()

	switch ann.Type {
	case extract.SemanticRight:
		// Create right URI
		rightURI := fmt.Sprintf("%s%s:Right:%d:%s", b.baseURI, b.regID, ann.ArticleNum, ann.RightType)

		b.store.Add(rightURI, RDFType, ClassRight)
		b.store.Add(rightURI, "reg:rightType", string(ann.RightType))
		b.store.Add(rightURI, PropText, ann.MatchedText)
		b.store.Add(rightURI, "reg:confidence", fmt.Sprintf("%.2f", ann.Confidence))
		b.store.Add(rightURI, PropPartOf, articleURI)
		b.store.Add(rightURI, PropBelongsTo, regURI)

		// Link article to right
		b.store.Add(articleURI, PropGrantsRight, rightURI)

		// Beneficiary
		if ann.Beneficiary != extract.EntityUnspecified {
			b.store.Add(rightURI, "reg:beneficiary", string(ann.Beneficiary))
		}

		// Context
		if ann.Context != "" {
			b.store.Add(rightURI, "reg:context", ann.Context)
		}

		stats.Rights++
		stats.SemanticTriples += 8

	case extract.SemanticObligation, extract.SemanticProhibition:
		// Create obligation URI
		obligURI := fmt.Sprintf("%s%s:Obligation:%d:%s", b.baseURI, b.regID, ann.ArticleNum, ann.ObligationType)

		b.store.Add(obligURI, RDFType, ClassObligation)
		b.store.Add(obligURI, "reg:obligationType", string(ann.ObligationType))
		b.store.Add(obligURI, PropText, ann.MatchedText)
		b.store.Add(obligURI, "reg:confidence", fmt.Sprintf("%.2f", ann.Confidence))
		b.store.Add(obligURI, PropPartOf, articleURI)
		b.store.Add(obligURI, PropBelongsTo, regURI)

		// Link article to obligation
		b.store.Add(articleURI, PropImposesObligation, obligURI)

		// Duty bearer
		if ann.DutyBearer != extract.EntityUnspecified {
			b.store.Add(obligURI, "reg:dutyBearer", string(ann.DutyBearer))
		}

		// Mark if prohibition
		if ann.Type == extract.SemanticProhibition {
			b.store.Add(obligURI, "reg:isProhibition", "true")
		}

		// Context
		if ann.Context != "" {
			b.store.Add(obligURI, "reg:context", ann.Context)
		}

		stats.Obligations++
		stats.SemanticTriples += 9
	}
}

// BuildWithSemantics builds the graph including semantic extraction.
func (b *GraphBuilder) BuildWithSemantics(
	doc *extract.Document,
	defExtractor *extract.DefinitionExtractor,
	refExtractor *extract.ReferenceExtractor,
	semExtractor *extract.SemanticExtractor,
) (*BuildStats, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}

	stats := &BuildStats{}

	// Determine regulation ID
	b.regID = b.extractRegID(doc.Identifier)

	// Build basic structure
	b.buildRegulation(doc, stats)

	if doc.Preamble != nil {
		b.buildPreamble(doc.Preamble, stats)
	}

	for _, chapter := range doc.Chapters {
		b.buildChapter(chapter, stats)
	}

	// Build definitions
	if defExtractor != nil {
		definitions := defExtractor.ExtractDefinitions(doc)
		for _, def := range definitions {
			b.buildDefinedTerm(def, stats)
		}
	} else {
		for _, def := range doc.Definitions {
			b.buildDefinition(def, stats)
		}
	}

	// Build references
	if refExtractor != nil {
		refs := refExtractor.ExtractFromDocument(doc)
		for _, ref := range refs {
			b.buildReference(ref, stats)
		}
	}

	// Build semantic annotations
	if semExtractor != nil {
		annotations := semExtractor.ExtractFromDocument(doc)
		for _, ann := range annotations {
			b.buildSemanticAnnotation(ann, stats)
		}
	}

	stats.TotalTriples = b.store.Count()
	return stats, nil
}

// GetStore returns the underlying triple store.
func (b *GraphBuilder) GetStore() *TripleStore {
	return b.store
}

// GetRegulationURI returns the URI of the regulation.
func (b *GraphBuilder) GetRegulationURI() string {
	return b.regulationURI()
}

// GetRegID returns the regulation ID.
func (b *GraphBuilder) GetRegID() string {
	return b.regID
}

// GetBaseURI returns the base URI.
func (b *GraphBuilder) GetBaseURI() string {
	return b.baseURI
}
