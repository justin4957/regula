// Package store provides RDF triple storage and schema definitions for regulatory data.
package store

// Namespace URIs for the regulation ontology.
const (
	// NamespaceReg is the namespace for regulation-specific predicates.
	NamespaceReg = "https://regula.dev/ontology#"

	// NamespaceRDF is the standard RDF namespace.
	NamespaceRDF = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"

	// NamespaceRDFS is the RDF Schema namespace.
	NamespaceRDFS = "http://www.w3.org/2000/01/rdf-schema#"

	// NamespaceXSD is the XML Schema namespace for datatypes.
	NamespaceXSD = "http://www.w3.org/2001/XMLSchema#"

	// NamespaceDC is the Dublin Core namespace for metadata.
	NamespaceDC = "http://purl.org/dc/terms/"

	// NamespaceELI is the European Legislation Identifier ontology namespace.
	NamespaceELI = "http://data.europa.eu/eli/ontology#"

	// NamespaceFRBR is the Functional Requirements for Bibliographic Records namespace.
	NamespaceFRBR = "http://purl.org/vocab/frbr/core#"
)

// Namespace prefixes for compact URI representation.
const (
	PrefixReg  = "reg:"
	PrefixRDF  = "rdf:"
	PrefixRDFS = "rdfs:"
	PrefixXSD  = "xsd:"
	PrefixDC   = "dc:"
	PrefixELI  = "eli:"
	PrefixFRBR = "frbr:"
)

// ELI Classes - European Legislation Identifier types.
const (
	// ELIClassLegalResource represents a legislative resource at the Work level (FRBR).
	ELIClassLegalResource = "eli:LegalResource"

	// ELIClassLegalResourceSubdivision represents a subdivision of a legal resource.
	ELIClassLegalResourceSubdivision = "eli:LegalResourceSubdivision"

	// ELIClassLegalExpression represents a particular linguistic expression of a resource.
	ELIClassLegalExpression = "eli:LegalExpression"
)

// ELI Properties - European Legislation Identifier predicates.
const (
	// ELIPropTitle is the title of a legal resource.
	ELIPropTitle = "eli:title"

	// ELIPropIDLocal is the local identifier within the resource.
	ELIPropIDLocal = "eli:id_local"

	// ELIPropIsPartOf indicates hierarchical containment (child -> parent).
	ELIPropIsPartOf = "eli:is_part_of"

	// ELIPropHasPart indicates hierarchical containment (parent -> child).
	ELIPropHasPart = "eli:has_part"

	// ELIPropDateDocument is the date of the document.
	ELIPropDateDocument = "eli:date_document"

	// ELIPropVersion is the version identifier.
	ELIPropVersion = "eli:version"

	// ELIPropDescription is a description of the resource.
	ELIPropDescription = "eli:description"

	// ELIPropCites indicates a citation relationship.
	ELIPropCites = "eli:cites"

	// ELIPropCitedBy indicates an incoming citation (inverse of cites).
	ELIPropCitedBy = "eli:cited_by"

	// ELIPropTypeDocument is the document type classification.
	ELIPropTypeDocument = "eli:type_document"

	// ELIPropLanguage is the language of the resource.
	ELIPropLanguage = "eli:language"

	// ELIPropIsAbout indicates the subject matter of the resource.
	ELIPropIsAbout = "eli:is_about"

	// ELIPropPassedBy indicates the institution that passed the document.
	ELIPropPassedBy = "eli:passed_by"
)

// RDF Standard Predicates.
const (
	// RDFType indicates the class of a resource.
	RDFType = "rdf:type"

	// RDFSLabel provides a human-readable label.
	RDFSLabel = "rdfs:label"

	// RDFSComment provides a description.
	RDFSComment = "rdfs:comment"

	// RDFSSubClassOf indicates class hierarchy.
	RDFSSubClassOf = "rdfs:subClassOf"
)

// Classes - Types of regulatory entities.
const (
	// ClassRegulation represents a top-level regulation document.
	ClassRegulation = "reg:Regulation"

	// ClassDirective represents an EU directive.
	ClassDirective = "reg:Directive"

	// ClassDecision represents an EU decision.
	ClassDecision = "reg:Decision"

	// ClassChapter represents a chapter within a regulation.
	ClassChapter = "reg:Chapter"

	// ClassSection represents a section within a chapter.
	ClassSection = "reg:Section"

	// ClassArticle represents an article (main provision unit).
	ClassArticle = "reg:Article"

	// ClassParagraph represents a numbered paragraph within an article.
	ClassParagraph = "reg:Paragraph"

	// ClassPoint represents a lettered point within a paragraph.
	ClassPoint = "reg:Point"

	// ClassSubPoint represents a sub-point within a point.
	ClassSubPoint = "reg:SubPoint"

	// ClassRecital represents a preamble recital.
	ClassRecital = "reg:Recital"

	// ClassPreamble represents the preamble section.
	ClassPreamble = "reg:Preamble"

	// ClassDefinedTerm represents a defined term from Article 4 or similar.
	ClassDefinedTerm = "reg:DefinedTerm"

	// ClassReference represents a cross-reference.
	ClassReference = "reg:Reference"

	// ClassObligation represents an obligation imposed by a provision.
	ClassObligation = "reg:Obligation"

	// ClassRight represents a right granted by a provision.
	ClassRight = "reg:Right"
)

// Metadata Properties - Basic descriptive predicates.
const (
	// PropTitle is the title of a provision or document.
	PropTitle = "reg:title"

	// PropText is the full text content of a provision.
	PropText = "reg:text"

	// PropNumber is the number/identifier of a provision (e.g., article number).
	PropNumber = "reg:number"

	// PropIdentifier is the formal identifier (e.g., "(EU) 2016/679").
	PropIdentifier = "reg:identifier"

	// PropLabel is a human-readable label (alias for rdfs:label).
	PropLabel = "reg:label"

	// PropDate is the date of adoption or entry into force.
	PropDate = "reg:date"

	// PropVersion is the version identifier.
	PropVersion = "reg:version"
)

// Structural Relationships - Hierarchical containment.
const (
	// PropPartOf indicates hierarchical containment (child -> parent).
	// Example: <GDPR:Art17> reg:partOf <GDPR:ChapterIII>
	PropPartOf = "reg:partOf"

	// PropContains indicates hierarchical containment (parent -> child).
	// Example: <GDPR:ChapterIII> reg:contains <GDPR:Art17>
	PropContains = "reg:contains"

	// PropBelongsTo indicates membership in a regulation.
	// Example: <GDPR:Art17> reg:belongsTo <GDPR>
	PropBelongsTo = "reg:belongsTo"

	// PropHasChapter links regulation to its chapters.
	PropHasChapter = "reg:hasChapter"

	// PropHasSection links chapter to its sections.
	PropHasSection = "reg:hasSection"

	// PropHasArticle links chapter/section to its articles.
	PropHasArticle = "reg:hasArticle"

	// PropHasParagraph links article to its paragraphs.
	PropHasParagraph = "reg:hasParagraph"

	// PropHasPoint links paragraph to its points.
	PropHasPoint = "reg:hasPoint"

	// PropHasRecital links preamble to its recitals.
	PropHasRecital = "reg:hasRecital"
)

// Cross-Reference Properties - Links between provisions.
const (
	// PropReferences indicates a cross-reference to another provision.
	// Example: <GDPR:Art17> reg:references <GDPR:Art6>
	PropReferences = "reg:references"

	// PropReferencedBy indicates incoming references (inverse of references).
	PropReferencedBy = "reg:referencedBy"

	// PropExternalRef indicates a reference to an external document.
	// Example: <GDPR:Art1> reg:externalRef <Directive:95/46/EC>
	PropExternalRef = "reg:externalRef"

	// PropRefersToArticle specifically references an article.
	PropRefersToArticle = "reg:refersToArticle"

	// PropRefersToChapter specifically references a chapter.
	PropRefersToChapter = "reg:refersToChapter"

	// PropRefersToParagraph specifically references a paragraph.
	PropRefersToParagraph = "reg:refersToParagraph"

	// PropRefersToPoint specifically references a point.
	PropRefersToPoint = "reg:refersToPoint"
)

// Definition Properties - Term definitions.
const (
	// PropDefinedIn indicates where a term is defined.
	// Example: <reg:PersonalData> reg:definedIn <GDPR:Art4>
	PropDefinedIn = "reg:definedIn"

	// PropDefines indicates what terms an article defines.
	// Example: <GDPR:Art4> reg:defines <reg:PersonalData>
	PropDefines = "reg:defines"

	// PropDefinition contains the definition text.
	PropDefinition = "reg:definition"

	// PropTerm is the defined term itself.
	PropTerm = "reg:term"

	// PropNormalizedTerm is the lowercase normalized form.
	PropNormalizedTerm = "reg:normalizedTerm"

	// PropScope indicates the scope where a definition applies.
	PropScope = "reg:scope"

	// PropUsesTerm indicates a provision uses a defined term.
	PropUsesTerm = "reg:usesTerm"
)

// Amendment Properties - Document evolution.
const (
	// PropAmends indicates an amendment relationship.
	// Example: <Regulation:2024/XXX> reg:amends <GDPR>
	PropAmends = "reg:amends"

	// PropAmendedBy indicates incoming amendments (inverse).
	PropAmendedBy = "reg:amendedBy"

	// PropSupersedes indicates replacement of previous regulation.
	PropSupersedes = "reg:supersedes"

	// PropSupersededBy indicates being replaced (inverse).
	PropSupersededBy = "reg:supersededBy"

	// PropRepeals indicates repealing another provision.
	PropRepeals = "reg:repeals"

	// PropRepealedBy indicates being repealed (inverse).
	PropRepealedBy = "reg:repealedBy"

	// PropDelegatesTo indicates delegation of power.
	PropDelegatesTo = "reg:delegatesTo"
)

// Semantic Properties - Legal meaning and effects.
const (
	// PropGrantsRight indicates a provision grants a right.
	// Example: <GDPR:Art17> reg:grantsRight <reg:RightToErasure>
	PropGrantsRight = "reg:grantsRight"

	// PropImposesObligation indicates a provision creates an obligation.
	// Example: <GDPR:Art12> reg:imposesObligation <reg:TransparencyObligation>
	PropImposesObligation = "reg:imposesObligation"

	// PropRequires indicates a requirement (e.g., consent).
	// Example: <GDPR:Art6> reg:requires <reg:Consent>
	PropRequires = "reg:requires"

	// PropProhibits indicates something is prohibited.
	PropProhibits = "reg:prohibits"

	// PropPermits indicates something is permitted.
	PropPermits = "reg:permits"

	// PropExempts indicates an exemption.
	PropExempts = "reg:exempts"

	// PropAppliesTo indicates what entities/situations apply.
	PropAppliesTo = "reg:appliesTo"

	// PropSubjectTo indicates being subject to conditions.
	PropSubjectTo = "reg:subjectTo"
)

// Entity Properties - Data subjects, controllers, etc.
const (
	// PropActor indicates the actor in an obligation or right.
	PropActor = "reg:actor"

	// PropBeneficiary indicates who benefits from a right.
	PropBeneficiary = "reg:beneficiary"

	// PropDutyBearer indicates who bears an obligation.
	PropDutyBearer = "reg:dutyBearer"

	// PropDataSubject indicates relation to data subject.
	PropDataSubject = "reg:dataSubject"

	// PropController indicates relation to data controller.
	PropController = "reg:controller"

	// PropProcessor indicates relation to data processor.
	PropProcessor = "reg:processor"
)

// Temporal Properties - Time-related aspects.
const (
	// PropEffectiveDate is when a provision comes into effect.
	PropEffectiveDate = "reg:effectiveDate"

	// PropExpiryDate is when a provision expires.
	PropExpiryDate = "reg:expiryDate"

	// PropDeadline indicates a deadline for compliance.
	PropDeadline = "reg:deadline"

	// PropTimeLimit indicates a time limit (e.g., "within 1 month").
	PropTimeLimit = "reg:timeLimit"

	// PropTemporalKind classifies the temporal qualifier (e.g., "as_amended", "in_force_on", "repealed").
	PropTemporalKind = "reg:temporalKind"

	// PropTemporalDescription is the full matched text of the temporal qualifier.
	PropTemporalDescription = "reg:temporalDescription"
)

// Provenance Properties - Source and origin tracking.
const (
	// PropSourceDocument is the source document URI.
	PropSourceDocument = "reg:sourceDocument"

	// PropSourceOffset is the character offset in source.
	PropSourceOffset = "reg:sourceOffset"

	// PropSourceLength is the length of text in source.
	PropSourceLength = "reg:sourceLength"

	// PropExtractedFrom indicates extraction source.
	PropExtractedFrom = "reg:extractedFrom"

	// PropExtractedAt is the extraction timestamp.
	PropExtractedAt = "reg:extractedAt"
)

// Resolution Properties - Reference resolution tracking.
const (
	// PropResolutionStatus indicates the resolution outcome.
	// Values: "resolved", "partial", "ambiguous", "not_found", "external"
	PropResolutionStatus = "reg:resolutionStatus"

	// PropResolutionConfidence indicates confidence in resolution (0.0-1.0).
	PropResolutionConfidence = "reg:resolutionConfidence"

	// PropResolutionReason explains the resolution decision.
	PropResolutionReason = "reg:resolutionReason"

	// PropResolvedTarget is the resolved target URI.
	PropResolvedTarget = "reg:resolvedTarget"

	// PropAlternativeTarget lists alternative resolution targets.
	PropAlternativeTarget = "reg:alternativeTarget"
)

// Common Right and Obligation types.
const (
	// Right types
	RightAccess          = "reg:RightOfAccess"
	RightRectification   = "reg:RightToRectification"
	RightErasure         = "reg:RightToErasure"
	RightRestriction     = "reg:RightToRestriction"
	RightPortability     = "reg:RightToDataPortability"
	RightObject          = "reg:RightToObject"
	RightNotAutomated    = "reg:RightAgainstAutomatedDecision"
	RightWithdrawConsent = "reg:RightToWithdrawConsent"
	RightLodgeComplaint  = "reg:RightToLodgeComplaint"
	RightEffectiveRemedy = "reg:RightToEffectiveRemedy"
	RightCompensation    = "reg:RightToCompensation"
	RightInformation     = "reg:RightToInformation"

	// Obligation types
	ObligationTransparency     = "reg:TransparencyObligation"
	ObligationNotify           = "reg:NotificationObligation"
	ObligationSecure           = "reg:SecurityObligation"
	ObligationRecord           = "reg:RecordKeepingObligation"
	ObligationImpactAssessment = "reg:ImpactAssessmentObligation"
	ObligationCooperate        = "reg:CooperationObligation"
	ObligationAppoint          = "reg:AppointmentObligation"
)

// Federation Properties - Cross-document graph linking for recursive fetching.
const (
	// ClassExternalDocument represents an external document fetched during federation.
	ClassExternalDocument = "reg:ExternalDocument"

	// PropFederatedFrom links a source document to a fetched external document.
	PropFederatedFrom = "reg:federatedFrom"

	// PropFetchedAt is the timestamp when the external document was fetched.
	PropFetchedAt = "reg:fetchedAt"

	// PropFetchDepth is the BFS depth at which the document was discovered.
	PropFetchDepth = "reg:fetchDepth"

	// PropExternalDocURI is the resolved HTTP URL of the external document.
	PropExternalDocURI = "reg:externalDocURI"
)

// Crawl Provenance Properties - Tracking legislation discovery via crawling.
const (
	// ClassCrawledDocument represents a document discovered and ingested by the crawler.
	ClassCrawledDocument = "reg:CrawledDocument"

	// PropCrawlDiscoveredBy records which document led to the discovery of this one.
	PropCrawlDiscoveredBy = "reg:crawlDiscoveredBy"

	// PropCrawlCitation records the citation text that triggered discovery.
	PropCrawlCitation = "reg:crawlCitation"

	// PropCrawlDepth records the BFS depth at which the document was discovered.
	PropCrawlDepth = "reg:crawlDepth"

	// PropCrawlSource records the source domain or URL from which the document was fetched.
	PropCrawlSource = "reg:crawlSource"

	// PropCrawlStatus records the crawl processing status of this document.
	PropCrawlStatus = "reg:crawlStatus"

	// PropCrawlFetchedAt records when the crawler fetched this document.
	PropCrawlFetchedAt = "reg:crawlFetchedAt"
)

// Legal basis types (for GDPR Article 6).
const (
	LegalBasisConsent            = "reg:Consent"
	LegalBasisContract           = "reg:ContractPerformance"
	LegalBasisLegalObligation    = "reg:LegalObligation"
	LegalBasisVitalInterest      = "reg:VitalInterest"
	LegalBasisPublicTask         = "reg:PublicTask"
	LegalBasisLegitimateInterest = "reg:LegitimateInterest"
)

// Congressional Committee Classes and Properties.
const (
	// ClassCommittee represents a congressional committee.
	ClassCommittee = "reg:Committee"

	// ClassJurisdictionTopic represents a committee jurisdiction topic.
	ClassJurisdictionTopic = "reg:JurisdictionTopic"

	// PropHasJurisdiction links a committee to its jurisdiction topics.
	PropHasJurisdiction = "reg:hasJurisdiction"

	// PropJurisdictionText contains the text of a jurisdiction topic.
	PropJurisdictionText = "reg:jurisdictionText"

	// PropCommitteeLetter is the rule letter (e.g., "a", "j").
	PropCommitteeLetter = "reg:committeeLetter"

	// PropSourceClause is the source clause reference (e.g., "Rule X, clause 1(j)(4)").
	PropSourceClause = "reg:sourceClause"
)

// URIBuilder helps construct URIs for regulatory entities.
type URIBuilder struct {
	BaseURI string
}

// NewURIBuilder creates a new URI builder with the given base URI.
func NewURIBuilder(baseURI string) *URIBuilder {
	return &URIBuilder{BaseURI: baseURI}
}

// Regulation creates a URI for a regulation.
func (b *URIBuilder) Regulation(id string) string {
	return b.BaseURI + id
}

// Chapter creates a URI for a chapter.
func (b *URIBuilder) Chapter(regID, chapterNum string) string {
	return b.BaseURI + regID + ":Chapter" + chapterNum
}

// Section creates a URI for a section.
func (b *URIBuilder) Section(regID, chapterNum string, sectionNum int) string {
	return b.BaseURI + regID + ":Chapter" + chapterNum + ":Section" + itoa(sectionNum)
}

// Article creates a URI for an article.
func (b *URIBuilder) Article(regID string, articleNum int) string {
	return b.BaseURI + regID + ":Art" + itoa(articleNum)
}

// Paragraph creates a URI for a paragraph.
func (b *URIBuilder) Paragraph(regID string, articleNum, paraNum int) string {
	return b.BaseURI + regID + ":Art" + itoa(articleNum) + ":" + itoa(paraNum)
}

// Point creates a URI for a point.
func (b *URIBuilder) Point(regID string, articleNum, paraNum int, letter string) string {
	return b.BaseURI + regID + ":Art" + itoa(articleNum) + ":" + itoa(paraNum) + ":" + letter
}

// Recital creates a URI for a recital.
func (b *URIBuilder) Recital(regID string, recitalNum int) string {
	return b.BaseURI + regID + ":Recital" + itoa(recitalNum)
}

// DefinedTerm creates a URI for a defined term.
func (b *URIBuilder) DefinedTerm(regID, normalizedTerm string) string {
	// Replace spaces with underscores for URI-safe term
	safeTerm := ""
	for _, c := range normalizedTerm {
		if c == ' ' {
			safeTerm += "_"
		} else if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			safeTerm += string(c)
		}
	}
	return b.BaseURI + regID + ":Term:" + safeTerm
}

// itoa converts int to string (simple helper to avoid importing strconv).
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	if i < 0 {
		return "-" + itoa(-i)
	}

	digits := ""
	for i > 0 {
		digits = string(rune('0'+i%10)) + digits
		i /= 10
	}
	return digits
}

// DefaultURIBuilder creates a builder with the default regula namespace.
func DefaultURIBuilder() *URIBuilder {
	return NewURIBuilder(NamespaceReg)
}

// GDPRURIBuilder creates a builder specifically for GDPR URIs.
func GDPRURIBuilder() *URIBuilder {
	return NewURIBuilder("https://regula.dev/regulations/GDPR#")
}
