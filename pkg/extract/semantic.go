package extract

import (
	"regexp"
	"strings"
)

// SemanticType indicates the type of semantic annotation.
type SemanticType string

const (
	SemanticRight      SemanticType = "right"
	SemanticObligation SemanticType = "obligation"
	SemanticProhibition SemanticType = "prohibition"
	SemanticPermission SemanticType = "permission"
	SemanticExemption  SemanticType = "exemption"
)

// EntityType represents the type of entity involved in a right/obligation.
type EntityType string

const (
	EntityDataSubject       EntityType = "DataSubject"
	EntityController        EntityType = "Controller"
	EntityProcessor         EntityType = "Processor"
	EntitySupervisoryAuth   EntityType = "SupervisoryAuthority"
	EntityMemberState       EntityType = "MemberState"
	EntityThirdParty        EntityType = "ThirdParty"
	EntityRecipient         EntityType = "Recipient"
	EntityRepresentative    EntityType = "Representative"
	EntityDataProtectionOff EntityType = "DataProtectionOfficer"
	EntityUnspecified       EntityType = "Unspecified"
)

// RightType represents specific types of rights.
type RightType string

const (
	RightAccess          RightType = "RightOfAccess"
	RightRectification   RightType = "RightToRectification"
	RightErasure         RightType = "RightToErasure"
	RightRestriction     RightType = "RightToRestriction"
	RightPortability     RightType = "RightToDataPortability"
	RightObject          RightType = "RightToObject"
	RightNotAutomated    RightType = "RightAgainstAutomatedDecision"
	RightWithdrawConsent RightType = "RightToWithdrawConsent"
	RightLodgeComplaint  RightType = "RightToLodgeComplaint"
	RightEffectiveRemedy RightType = "RightToEffectiveRemedy"
	RightCompensation    RightType = "RightToCompensation"
	RightInformation     RightType = "RightToInformation"
	RightNotification    RightType = "RightToNotification"
	RightGeneric         RightType = "Right"
)

// ObligationType represents specific types of obligations.
type ObligationType string

const (
	ObligationLawfulProcessing   ObligationType = "LawfulProcessingObligation"
	ObligationConsent            ObligationType = "ConsentObligation"
	ObligationTransparency       ObligationType = "TransparencyObligation"
	ObligationNotifyBreach       ObligationType = "BreachNotificationObligation"
	ObligationNotifySubject      ObligationType = "SubjectNotificationObligation"
	ObligationSecure             ObligationType = "SecurityObligation"
	ObligationRecord             ObligationType = "RecordKeepingObligation"
	ObligationImpactAssessment   ObligationType = "ImpactAssessmentObligation"
	ObligationCooperate          ObligationType = "CooperationObligation"
	ObligationAppoint            ObligationType = "AppointmentObligation"
	ObligationProvideInformation ObligationType = "InformationProvisionObligation"
	ObligationRespond            ObligationType = "ResponseObligation"
	ObligationEnsure             ObligationType = "EnsureObligation"
	ObligationImplement          ObligationType = "ImplementationObligation"
	ObligationVerify             ObligationType = "VerificationObligation"
	ObligationGeneric            ObligationType = "Obligation"
)

// SemanticAnnotation represents an extracted right or obligation.
type SemanticAnnotation struct {
	Type           SemanticType   `json:"type"`
	ArticleNum     int            `json:"article_num"`
	ParagraphNum   int            `json:"paragraph_num,omitempty"`
	PointLetter    string         `json:"point_letter,omitempty"`

	// For rights
	RightType      RightType      `json:"right_type,omitempty"`
	Beneficiary    EntityType     `json:"beneficiary,omitempty"`

	// For obligations
	ObligationType ObligationType `json:"obligation_type,omitempty"`
	DutyBearer     EntityType     `json:"duty_bearer,omitempty"`

	// Common fields
	MatchedText    string         `json:"matched_text"`
	MatchedPattern string         `json:"matched_pattern"`
	Confidence     float64        `json:"confidence"`
	Context        string         `json:"context,omitempty"` // Surrounding text
}

// SemanticExtractor extracts rights and obligations from regulatory text.
type SemanticExtractor struct {
	// Right patterns
	rightPatterns []*semanticPattern

	// Obligation patterns
	obligationPatterns []*semanticPattern

	// Entity detection patterns
	entityPatterns map[EntityType]*regexp.Regexp
}

// semanticPattern represents a pattern for detecting semantic content.
type semanticPattern struct {
	Pattern     *regexp.Regexp
	Type        SemanticType
	RightType   RightType
	ObligType   ObligationType
	Beneficiary EntityType
	DutyBearer  EntityType
	Confidence  float64
	Description string
}

// NewSemanticExtractor creates a new extractor with default patterns.
func NewSemanticExtractor() *SemanticExtractor {
	extractor := &SemanticExtractor{
		entityPatterns: make(map[EntityType]*regexp.Regexp),
	}

	extractor.initRightPatterns()
	extractor.initObligationPatterns()
	extractor.initEntityPatterns()

	return extractor
}

// initRightPatterns initializes patterns for detecting rights.
func (e *SemanticExtractor) initRightPatterns() {
	e.rightPatterns = []*semanticPattern{
		// Specific GDPR rights
		{
			Pattern:     regexp.MustCompile(`(?i)right\s+(?:of\s+)?access`),
			Type:        SemanticRight,
			RightType:   RightAccess,
			Beneficiary: EntityDataSubject,
			Confidence:  1.0,
			Description: "Right of access",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)right\s+to\s+(?:obtain\s+)?rectification`),
			Type:        SemanticRight,
			RightType:   RightRectification,
			Beneficiary: EntityDataSubject,
			Confidence:  1.0,
			Description: "Right to rectification",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)right\s+to\s+(?:obtain\s+)?erasure|right\s+to\s+be\s+forgotten`),
			Type:        SemanticRight,
			RightType:   RightErasure,
			Beneficiary: EntityDataSubject,
			Confidence:  1.0,
			Description: "Right to erasure",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)right\s+to\s+(?:obtain\s+)?restriction\s+of\s+processing`),
			Type:        SemanticRight,
			RightType:   RightRestriction,
			Beneficiary: EntityDataSubject,
			Confidence:  1.0,
			Description: "Right to restriction of processing",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)right\s+to\s+data\s+portability`),
			Type:        SemanticRight,
			RightType:   RightPortability,
			Beneficiary: EntityDataSubject,
			Confidence:  1.0,
			Description: "Right to data portability",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)right\s+to\s+object`),
			Type:        SemanticRight,
			RightType:   RightObject,
			Beneficiary: EntityDataSubject,
			Confidence:  1.0,
			Description: "Right to object",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)not\s+(?:to\s+)?be\s+subject\s+to\s+(?:a\s+)?decision\s+based\s+solely\s+on\s+automated\s+processing`),
			Type:        SemanticRight,
			RightType:   RightNotAutomated,
			Beneficiary: EntityDataSubject,
			Confidence:  1.0,
			Description: "Right against automated decision-making",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)right\s+to\s+withdraw\s+(?:his\s+or\s+her\s+)?consent`),
			Type:        SemanticRight,
			RightType:   RightWithdrawConsent,
			Beneficiary: EntityDataSubject,
			Confidence:  1.0,
			Description: "Right to withdraw consent",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)right\s+to\s+lodge\s+a\s+complaint`),
			Type:        SemanticRight,
			RightType:   RightLodgeComplaint,
			Beneficiary: EntityDataSubject,
			Confidence:  1.0,
			Description: "Right to lodge complaint",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)right\s+to\s+an?\s+effective\s+(?:judicial\s+)?remedy`),
			Type:        SemanticRight,
			RightType:   RightEffectiveRemedy,
			Beneficiary: EntityDataSubject,
			Confidence:  1.0,
			Description: "Right to effective remedy",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)right\s+to\s+(?:receive\s+)?compensation`),
			Type:        SemanticRight,
			RightType:   RightCompensation,
			Beneficiary: EntityDataSubject,
			Confidence:  1.0,
			Description: "Right to compensation",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)right\s+to\s+(?:obtain\s+|receive\s+)?(?:the\s+)?(?:following\s+)?information`),
			Type:        SemanticRight,
			RightType:   RightInformation,
			Beneficiary: EntityDataSubject,
			Confidence:  0.9,
			Description: "Right to information",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)right\s+to\s+be\s+(?:informed|notified)`),
			Type:        SemanticRight,
			RightType:   RightNotification,
			Beneficiary: EntityDataSubject,
			Confidence:  0.9,
			Description: "Right to be notified",
		},
		// Generic right patterns
		{
			Pattern:     regexp.MustCompile(`(?i)(?:the\s+)?data\s+subject\s+(?:shall\s+)?ha(?:s|ve)\s+the\s+right`),
			Type:        SemanticRight,
			RightType:   RightGeneric,
			Beneficiary: EntityDataSubject,
			Confidence:  0.9,
			Description: "Data subject right (generic)",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)shall\s+have\s+the\s+right\s+to`),
			Type:        SemanticRight,
			RightType:   RightGeneric,
			Beneficiary: EntityUnspecified,
			Confidence:  0.8,
			Description: "Generic right grant",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(?:is|are)\s+entitled\s+to`),
			Type:        SemanticRight,
			RightType:   RightGeneric,
			Beneficiary: EntityUnspecified,
			Confidence:  0.7,
			Description: "Entitlement",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)may\s+(?:request|obtain|exercise)`),
			Type:        SemanticRight,
			RightType:   RightGeneric,
			Beneficiary: EntityUnspecified,
			Confidence:  0.6,
			Description: "May request/obtain",
		},
	}
}

// initObligationPatterns initializes patterns for detecting obligations.
func (e *SemanticExtractor) initObligationPatterns() {
	e.obligationPatterns = []*semanticPattern{
		// Specific GDPR obligations
		{
			Pattern:     regexp.MustCompile(`(?i)processing\s+shall\s+be\s+lawful`),
			Type:        SemanticObligation,
			ObligType:   ObligationLawfulProcessing,
			DutyBearer:  EntityController,
			Confidence:  1.0,
			Description: "Lawful processing obligation",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)consent\s+(?:shall\s+be|must\s+be|is)\s+(?:freely\s+)?given`),
			Type:        SemanticObligation,
			ObligType:   ObligationConsent,
			DutyBearer:  EntityController,
			Confidence:  1.0,
			Description: "Consent requirement",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(?:shall|must)\s+(?:be\s+able\s+to\s+)?demonstrate\s+(?:that\s+)?consent`),
			Type:        SemanticObligation,
			ObligType:   ObligationConsent,
			DutyBearer:  EntityController,
			Confidence:  1.0,
			Description: "Consent demonstration",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)notify\s+(?:the\s+)?(?:personal\s+data\s+)?breach\s+to\s+(?:the\s+)?supervisory\s+authority`),
			Type:        SemanticObligation,
			ObligType:   ObligationNotifyBreach,
			DutyBearer:  EntityController,
			Confidence:  1.0,
			Description: "Breach notification to authority",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)communicate\s+(?:the\s+)?(?:personal\s+data\s+)?breach\s+to\s+(?:the\s+)?data\s+subject`),
			Type:        SemanticObligation,
			ObligType:   ObligationNotifySubject,
			DutyBearer:  EntityController,
			Confidence:  1.0,
			Description: "Breach notification to subject",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(?:shall|must)\s+implement\s+(?:appropriate\s+)?(?:technical\s+and\s+organisational\s+)?(?:security\s+)?measures`),
			Type:        SemanticObligation,
			ObligType:   ObligationSecure,
			DutyBearer:  EntityController,
			Confidence:  1.0,
			Description: "Security measures obligation",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(?:shall|must)\s+maintain\s+a\s+record`),
			Type:        SemanticObligation,
			ObligType:   ObligationRecord,
			DutyBearer:  EntityController,
			Confidence:  1.0,
			Description: "Record-keeping obligation",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(?:shall|must)\s+(?:carry\s+out|conduct)\s+(?:a\s+|an\s+)?(?:data\s+protection\s+)?impact\s+assessment`),
			Type:        SemanticObligation,
			ObligType:   ObligationImpactAssessment,
			DutyBearer:  EntityController,
			Confidence:  1.0,
			Description: "Impact assessment obligation",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(?:shall|must)\s+cooperate\s+with\s+(?:the\s+)?supervisory\s+authority`),
			Type:        SemanticObligation,
			ObligType:   ObligationCooperate,
			DutyBearer:  EntityController,
			Confidence:  1.0,
			Description: "Cooperation obligation",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(?:shall|must)\s+designate\s+a\s+data\s+protection\s+officer`),
			Type:        SemanticObligation,
			ObligType:   ObligationAppoint,
			DutyBearer:  EntityController,
			Confidence:  1.0,
			Description: "DPO appointment obligation",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(?:shall|must)\s+provide\s+(?:the\s+)?(?:following\s+)?information`),
			Type:        SemanticObligation,
			ObligType:   ObligationProvideInformation,
			DutyBearer:  EntityController,
			Confidence:  0.9,
			Description: "Information provision obligation",
		},
		// Generic obligation patterns
		{
			Pattern:     regexp.MustCompile(`(?i)(?:the\s+)?controller\s+shall(?:\s+be\s+responsible)?`),
			Type:        SemanticObligation,
			ObligType:   ObligationGeneric,
			DutyBearer:  EntityController,
			Confidence:  0.9,
			Description: "Controller obligation",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(?:the\s+)?processor\s+shall`),
			Type:        SemanticObligation,
			ObligType:   ObligationGeneric,
			DutyBearer:  EntityProcessor,
			Confidence:  0.9,
			Description: "Processor obligation",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(?:shall|must)\s+ensure\s+(?:that)?`),
			Type:        SemanticObligation,
			ObligType:   ObligationEnsure,
			DutyBearer:  EntityUnspecified,
			Confidence:  0.8,
			Description: "Ensure obligation",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(?:shall|must)\s+take\s+(?:appropriate\s+)?(?:measures|steps|action)`),
			Type:        SemanticObligation,
			ObligType:   ObligationImplement,
			DutyBearer:  EntityUnspecified,
			Confidence:  0.8,
			Description: "Implementation obligation",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(?:shall|must)\s+verify`),
			Type:        SemanticObligation,
			ObligType:   ObligationVerify,
			DutyBearer:  EntityUnspecified,
			Confidence:  0.8,
			Description: "Verification obligation",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(?:shall|must)\s+(?:respond|reply)\s+(?:to\s+(?:the\s+)?(?:data\s+subject|request))?`),
			Type:        SemanticObligation,
			ObligType:   ObligationRespond,
			DutyBearer:  EntityController,
			Confidence:  0.8,
			Description: "Response obligation",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(?:is|are)\s+required\s+to`),
			Type:        SemanticObligation,
			ObligType:   ObligationGeneric,
			DutyBearer:  EntityUnspecified,
			Confidence:  0.7,
			Description: "Requirement",
		},
		{
			Pattern:     regexp.MustCompile(`(?i)(?:shall|must)\s+not\s+(?:be\s+)?(?:process|transfer|disclose)`),
			Type:        SemanticProhibition,
			ObligType:   ObligationGeneric,
			DutyBearer:  EntityUnspecified,
			Confidence:  0.8,
			Description: "Prohibition",
		},
	}
}

// initEntityPatterns initializes patterns for detecting entities.
func (e *SemanticExtractor) initEntityPatterns() {
	e.entityPatterns = map[EntityType]*regexp.Regexp{
		EntityDataSubject:       regexp.MustCompile(`(?i)data\s+subject`),
		EntityController:        regexp.MustCompile(`(?i)(?:the\s+)?controller`),
		EntityProcessor:         regexp.MustCompile(`(?i)(?:the\s+)?processor`),
		EntitySupervisoryAuth:   regexp.MustCompile(`(?i)supervisory\s+authorit(?:y|ies)`),
		EntityMemberState:       regexp.MustCompile(`(?i)member\s+state`),
		EntityThirdParty:        regexp.MustCompile(`(?i)third\s+part(?:y|ies)`),
		EntityRecipient:         regexp.MustCompile(`(?i)recipient`),
		EntityRepresentative:    regexp.MustCompile(`(?i)representative`),
		EntityDataProtectionOff: regexp.MustCompile(`(?i)data\s+protection\s+officer`),
	}
}

// ExtractFromDocument extracts all semantic annotations from a document.
func (e *SemanticExtractor) ExtractFromDocument(doc *Document) []*SemanticAnnotation {
	var annotations []*SemanticAnnotation

	for _, article := range doc.AllArticles() {
		articleAnnotations := e.ExtractFromArticle(article)
		annotations = append(annotations, articleAnnotations...)
	}

	return annotations
}

// ExtractFromArticle extracts semantic annotations from a single article.
func (e *SemanticExtractor) ExtractFromArticle(article *Article) []*SemanticAnnotation {
	if article == nil {
		return nil
	}

	var annotations []*SemanticAnnotation

	// Extract from article title (often contains the right/obligation name)
	if article.Title != "" {
		titleAnnotations := e.extractFromTitle(article.Title, article.Number)
		annotations = append(annotations, titleAnnotations...)
	}

	// Extract from article text
	if article.Text != "" {
		articleAnnotations := e.extractFromText(article.Text, article.Number, 0, "")
		annotations = append(annotations, articleAnnotations...)
	}

	// Extract from paragraphs
	for _, para := range article.Paragraphs {
		if para.Text != "" {
			paraAnnotations := e.extractFromText(para.Text, article.Number, para.Number, "")
			annotations = append(annotations, paraAnnotations...)
		}

		// Extract from points
		for _, point := range para.Points {
			if point.Text != "" {
				pointAnnotations := e.extractFromText(point.Text, article.Number, para.Number, point.Letter)
				annotations = append(annotations, pointAnnotations...)
			}
		}
	}

	return annotations
}

// extractFromTitle extracts semantic annotations from article titles.
func (e *SemanticExtractor) extractFromTitle(title string, articleNum int) []*SemanticAnnotation {
	var annotations []*SemanticAnnotation
	lowerTitle := strings.ToLower(title)

	// Title-specific right patterns
	titleRightPatterns := []struct {
		pattern   string
		rightType RightType
	}{
		{"right of access", RightAccess},
		{"right to access", RightAccess},
		{"right to rectification", RightRectification},
		{"right to erasure", RightErasure},
		{"right to be forgotten", RightErasure},
		{"right to restriction", RightRestriction},
		{"right to data portability", RightPortability},
		{"right to object", RightObject},
		{"automated individual decision", RightNotAutomated},
		{"right not to be subject", RightNotAutomated},
	}

	for _, p := range titleRightPatterns {
		if strings.Contains(lowerTitle, p.pattern) {
			annotations = append(annotations, &SemanticAnnotation{
				Type:           SemanticRight,
				ArticleNum:     articleNum,
				RightType:      p.rightType,
				Beneficiary:    EntityDataSubject,
				MatchedText:    title,
				MatchedPattern: "Title: " + p.pattern,
				Confidence:     1.0,
				Context:        title,
			})
		}
	}

	// Title-specific obligation patterns
	titleObligPatterns := []struct {
		pattern   string
		obligType ObligationType
	}{
		{"notification obligation", ObligationNotifyBreach},
		{"record of processing", ObligationRecord},
		{"impact assessment", ObligationImpactAssessment},
		{"data protection officer", ObligationAppoint},
		{"security of processing", ObligationSecure},
		{"lawfulness of processing", ObligationLawfulProcessing},
		{"conditions for consent", ObligationConsent},
	}

	for _, p := range titleObligPatterns {
		if strings.Contains(lowerTitle, p.pattern) {
			annotations = append(annotations, &SemanticAnnotation{
				Type:           SemanticObligation,
				ArticleNum:     articleNum,
				ObligationType: p.obligType,
				DutyBearer:     EntityController,
				MatchedText:    title,
				MatchedPattern: "Title: " + p.pattern,
				Confidence:     1.0,
				Context:        title,
			})
		}
	}

	return annotations
}

// extractFromText extracts semantic annotations from a text segment.
func (e *SemanticExtractor) extractFromText(text string, articleNum, paraNum int, pointLetter string) []*SemanticAnnotation {
	var annotations []*SemanticAnnotation

	// Check right patterns
	for _, pattern := range e.rightPatterns {
		if loc := pattern.Pattern.FindStringIndex(text); loc != nil {
			matchedText := text[loc[0]:loc[1]]
			annotation := &SemanticAnnotation{
				Type:           pattern.Type,
				ArticleNum:     articleNum,
				ParagraphNum:   paraNum,
				PointLetter:    pointLetter,
				RightType:      pattern.RightType,
				Beneficiary:    pattern.Beneficiary,
				MatchedText:    matchedText,
				MatchedPattern: pattern.Description,
				Confidence:     pattern.Confidence,
				Context:        extractContext(text, loc[0], loc[1], 50),
			}

			// Try to identify beneficiary from context if unspecified
			if annotation.Beneficiary == EntityUnspecified {
				annotation.Beneficiary = e.identifyEntity(text)
			}

			annotations = append(annotations, annotation)
		}
	}

	// Check obligation patterns
	for _, pattern := range e.obligationPatterns {
		if loc := pattern.Pattern.FindStringIndex(text); loc != nil {
			matchedText := text[loc[0]:loc[1]]
			annotation := &SemanticAnnotation{
				Type:           pattern.Type,
				ArticleNum:     articleNum,
				ParagraphNum:   paraNum,
				PointLetter:    pointLetter,
				ObligationType: pattern.ObligType,
				DutyBearer:     pattern.DutyBearer,
				MatchedText:    matchedText,
				MatchedPattern: pattern.Description,
				Confidence:     pattern.Confidence,
				Context:        extractContext(text, loc[0], loc[1], 50),
			}

			// Try to identify duty bearer from context if unspecified
			if annotation.DutyBearer == EntityUnspecified {
				annotation.DutyBearer = e.identifyEntity(text)
			}

			annotations = append(annotations, annotation)
		}
	}

	return annotations
}

// identifyEntity tries to identify the primary entity mentioned in text.
func (e *SemanticExtractor) identifyEntity(text string) EntityType {
	// Check for each entity type in order of specificity
	entityOrder := []EntityType{
		EntityDataSubject,
		EntityController,
		EntityProcessor,
		EntitySupervisoryAuth,
		EntityDataProtectionOff,
		EntityMemberState,
		EntityThirdParty,
		EntityRecipient,
		EntityRepresentative,
	}

	for _, entityType := range entityOrder {
		if pattern, ok := e.entityPatterns[entityType]; ok {
			if pattern.MatchString(text) {
				return entityType
			}
		}
	}

	return EntityUnspecified
}

// extractContext extracts surrounding context for a match.
func extractContext(text string, start, end, contextLen int) string {
	contextStart := start - contextLen
	if contextStart < 0 {
		contextStart = 0
	}
	contextEnd := end + contextLen
	if contextEnd > len(text) {
		contextEnd = len(text)
	}

	context := text[contextStart:contextEnd]

	// Clean up context
	context = strings.ReplaceAll(context, "\n", " ")
	context = strings.Join(strings.Fields(context), " ")

	// Add ellipsis if truncated
	if contextStart > 0 {
		context = "..." + context
	}
	if contextEnd < len(text) {
		context = context + "..."
	}

	return context
}

// SemanticStats holds statistics about extracted semantic annotations.
type SemanticStats struct {
	TotalAnnotations int                    `json:"total_annotations"`
	Rights           int                    `json:"rights"`
	Obligations      int                    `json:"obligations"`
	Prohibitions     int                    `json:"prohibitions"`
	Permissions      int                    `json:"permissions"`
	ByRightType      map[RightType]int      `json:"by_right_type"`
	ByObligationType map[ObligationType]int `json:"by_obligation_type"`
	ByBeneficiary    map[EntityType]int     `json:"by_beneficiary"`
	ByDutyBearer     map[EntityType]int     `json:"by_duty_bearer"`
	ArticlesWithRights      int             `json:"articles_with_rights"`
	ArticlesWithObligations int             `json:"articles_with_obligations"`
	HighConfidence   int                    `json:"high_confidence"`
	MediumConfidence int                    `json:"medium_confidence"`
	LowConfidence    int                    `json:"low_confidence"`
}

// CalculateSemanticStats calculates statistics for semantic annotations.
func CalculateSemanticStats(annotations []*SemanticAnnotation) *SemanticStats {
	stats := &SemanticStats{
		TotalAnnotations: len(annotations),
		ByRightType:      make(map[RightType]int),
		ByObligationType: make(map[ObligationType]int),
		ByBeneficiary:    make(map[EntityType]int),
		ByDutyBearer:     make(map[EntityType]int),
	}

	articlesWithRights := make(map[int]bool)
	articlesWithObligations := make(map[int]bool)

	for _, ann := range annotations {
		switch ann.Type {
		case SemanticRight:
			stats.Rights++
			stats.ByRightType[ann.RightType]++
			stats.ByBeneficiary[ann.Beneficiary]++
			articlesWithRights[ann.ArticleNum] = true
		case SemanticObligation:
			stats.Obligations++
			stats.ByObligationType[ann.ObligationType]++
			stats.ByDutyBearer[ann.DutyBearer]++
			articlesWithObligations[ann.ArticleNum] = true
		case SemanticProhibition:
			stats.Prohibitions++
			stats.ByObligationType[ann.ObligationType]++
			stats.ByDutyBearer[ann.DutyBearer]++
			articlesWithObligations[ann.ArticleNum] = true
		case SemanticPermission:
			stats.Permissions++
		}

		// Confidence distribution
		switch {
		case ann.Confidence >= 0.9:
			stats.HighConfidence++
		case ann.Confidence >= 0.7:
			stats.MediumConfidence++
		default:
			stats.LowConfidence++
		}
	}

	stats.ArticlesWithRights = len(articlesWithRights)
	stats.ArticlesWithObligations = len(articlesWithObligations)

	return stats
}

// SemanticLookup provides indexed access to semantic annotations.
type SemanticLookup struct {
	all              []*SemanticAnnotation
	byArticle        map[int][]*SemanticAnnotation
	rights           []*SemanticAnnotation
	obligations      []*SemanticAnnotation
	byRightType      map[RightType][]*SemanticAnnotation
	byObligationType map[ObligationType][]*SemanticAnnotation
}

// NewSemanticLookup creates a lookup from annotations.
func NewSemanticLookup(annotations []*SemanticAnnotation) *SemanticLookup {
	lookup := &SemanticLookup{
		all:              annotations,
		byArticle:        make(map[int][]*SemanticAnnotation),
		byRightType:      make(map[RightType][]*SemanticAnnotation),
		byObligationType: make(map[ObligationType][]*SemanticAnnotation),
	}

	for _, ann := range annotations {
		lookup.byArticle[ann.ArticleNum] = append(lookup.byArticle[ann.ArticleNum], ann)

		switch ann.Type {
		case SemanticRight:
			lookup.rights = append(lookup.rights, ann)
			lookup.byRightType[ann.RightType] = append(lookup.byRightType[ann.RightType], ann)
		case SemanticObligation, SemanticProhibition:
			lookup.obligations = append(lookup.obligations, ann)
			lookup.byObligationType[ann.ObligationType] = append(lookup.byObligationType[ann.ObligationType], ann)
		}
	}

	return lookup
}

// GetByArticle returns annotations for a specific article.
func (l *SemanticLookup) GetByArticle(articleNum int) []*SemanticAnnotation {
	return l.byArticle[articleNum]
}

// GetRights returns all right annotations.
func (l *SemanticLookup) GetRights() []*SemanticAnnotation {
	return l.rights
}

// GetObligations returns all obligation annotations.
func (l *SemanticLookup) GetObligations() []*SemanticAnnotation {
	return l.obligations
}

// GetByRightType returns annotations for a specific right type.
func (l *SemanticLookup) GetByRightType(rightType RightType) []*SemanticAnnotation {
	return l.byRightType[rightType]
}

// GetByObligationType returns annotations for a specific obligation type.
func (l *SemanticLookup) GetByObligationType(obligationType ObligationType) []*SemanticAnnotation {
	return l.byObligationType[obligationType]
}

// All returns all annotations.
func (l *SemanticLookup) All() []*SemanticAnnotation {
	return l.all
}

// RightsCount returns the number of rights.
func (l *SemanticLookup) RightsCount() int {
	return len(l.rights)
}

// ObligationsCount returns the number of obligations.
func (l *SemanticLookup) ObligationsCount() int {
	return len(l.obligations)
}
