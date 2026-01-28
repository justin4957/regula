// Package validate provides validation and quality metrics for regulatory graphs.
package validate

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/coolbeans/regula/pkg/extract"
	"github.com/coolbeans/regula/pkg/store"
)

// ValidationResult represents the complete validation report.
type ValidationResult struct {
	// Overall status
	Status     ValidationStatus `json:"status"`
	Threshold  float64          `json:"threshold"`
	OverallScore float64        `json:"overall_score"`

	// Profile used
	ProfileName string            `json:"profile_name"`
	ProfileUsed *ValidationProfile `json:"profile_used,omitempty"`

	// Component results
	References   *ReferenceValidation   `json:"references"`
	Connectivity *ConnectivityValidation `json:"connectivity"`
	Definitions  *DefinitionValidation  `json:"definitions"`
	Semantics    *SemanticValidation    `json:"semantics"`
	Structure    *StructureValidation   `json:"structure"`

	// Component scores (for transparency)
	ComponentScores *ComponentScores `json:"component_scores,omitempty"`

	// Summary
	Issues   []ValidationIssue `json:"issues"`
	Warnings []ValidationIssue `json:"warnings"`
}

// ComponentScores shows individual scores and weights for transparency.
type ComponentScores struct {
	ReferenceScore   float64 `json:"reference_score"`
	ReferenceWeight  float64 `json:"reference_weight"`
	ConnectivityScore float64 `json:"connectivity_score"`
	ConnectivityWeight float64 `json:"connectivity_weight"`
	DefinitionScore  float64 `json:"definition_score"`
	DefinitionWeight float64 `json:"definition_weight"`
	SemanticScore    float64 `json:"semantic_score"`
	SemanticWeight   float64 `json:"semantic_weight"`
	StructureScore   float64 `json:"structure_score"`
	StructureWeight  float64 `json:"structure_weight"`
}

// StructureValidation contains document structure quality metrics.
type StructureValidation struct {
	// Counts
	TotalArticles    int `json:"total_articles"`
	TotalChapters    int `json:"total_chapters"`
	TotalSections    int `json:"total_sections"`
	TotalRecitals    int `json:"total_recitals"`

	// Expected vs actual (from profile)
	ExpectedArticles    int     `json:"expected_articles"`
	ExpectedDefinitions int     `json:"expected_definitions"`
	ExpectedChapters    int     `json:"expected_chapters"`

	// Quality metrics
	ArticleCompleteness   float64 `json:"article_completeness"`   // % of expected articles
	DefinitionCompleteness float64 `json:"definition_completeness"` // % of expected definitions
	ChapterCompleteness   float64 `json:"chapter_completeness"`   // % of expected chapters

	// Content quality
	ArticlesWithContent   int     `json:"articles_with_content"`
	ArticlesEmpty         int     `json:"articles_empty"`
	ContentRate           float64 `json:"content_rate"` // % of articles with meaningful content

	// Overall structure score
	StructureScore float64 `json:"structure_score"`
}

// ValidationStatus indicates pass/fail.
type ValidationStatus string

const (
	StatusPass ValidationStatus = "PASS"
	StatusFail ValidationStatus = "FAIL"
	StatusWarn ValidationStatus = "WARN"
)

// ValidationIssue represents a single issue or warning.
type ValidationIssue struct {
	Category    string `json:"category"`
	Severity    string `json:"severity"` // "error", "warning", "info"
	Message     string `json:"message"`
	Count       int    `json:"count,omitempty"`
	Examples    []string `json:"examples,omitempty"`
}

// ReferenceValidation contains reference resolution metrics.
type ReferenceValidation struct {
	TotalReferences int     `json:"total_references"`
	Resolved        int     `json:"resolved"`
	Partial         int     `json:"partial"`
	Ambiguous       int     `json:"ambiguous"`
	NotFound        int     `json:"not_found"`
	External        int     `json:"external"`
	RangeRefs       int     `json:"range_refs"`

	ResolutionRate   float64 `json:"resolution_rate"`
	HighConfidence   int     `json:"high_confidence"`
	MediumConfidence int     `json:"medium_confidence"`
	LowConfidence    int     `json:"low_confidence"`

	// Details for issues
	UnresolvedExamples []ReferenceExample `json:"unresolved_examples,omitempty"`
	AmbiguousExamples  []ReferenceExample `json:"ambiguous_examples,omitempty"`
}

// ReferenceExample provides details about a specific reference.
type ReferenceExample struct {
	SourceArticle int    `json:"source_article"`
	RawText       string `json:"raw_text"`
	Reason        string `json:"reason"`
}

// ConnectivityValidation contains graph connectivity metrics.
type ConnectivityValidation struct {
	TotalProvisions   int     `json:"total_provisions"`
	ConnectedCount    int     `json:"connected_count"`
	OrphanCount       int     `json:"orphan_count"`
	ConnectivityRate  float64 `json:"connectivity_rate"`

	// Orphan details
	OrphanArticles []int `json:"orphan_articles,omitempty"`

	// Graph metrics
	AvgIncomingRefs   float64 `json:"avg_incoming_refs"`
	AvgOutgoingRefs   float64 `json:"avg_outgoing_refs"`
	MostReferenced    []ArticleRefCount `json:"most_referenced,omitempty"`
	MostReferencing   []ArticleRefCount `json:"most_referencing,omitempty"`
}

// ArticleRefCount tracks reference counts per article.
type ArticleRefCount struct {
	ArticleNum int `json:"article_num"`
	Count      int `json:"count"`
}

// DefinitionValidation contains definition coverage metrics.
type DefinitionValidation struct {
	TotalDefinitions   int     `json:"total_definitions"`
	UsedDefinitions    int     `json:"used_definitions"`
	UnusedDefinitions  int     `json:"unused_definitions"`
	UsageRate          float64 `json:"usage_rate"`

	TotalUsages        int     `json:"total_usages"`
	ArticlesWithTerms  int     `json:"articles_with_terms"`

	// Unused terms
	UnusedTerms []string `json:"unused_terms,omitempty"`

	// Most used
	MostUsedTerms []TermUsageCount `json:"most_used_terms,omitempty"`
}

// TermUsageCount tracks usage for a term.
type TermUsageCount struct {
	Term         string `json:"term"`
	UsageCount   int    `json:"usage_count"`
	ArticleCount int    `json:"article_count"`
}

// SemanticValidation contains semantic extraction metrics.
type SemanticValidation struct {
	RightsCount       int     `json:"rights_count"`
	ObligationsCount  int     `json:"obligations_count"`
	ArticlesWithRights int    `json:"articles_with_rights"`
	ArticlesWithOblig  int    `json:"articles_with_obligations"`

	// Known rights validation (regulation-aware)
	RegulationType    string   `json:"regulation_type"` // "GDPR", "CCPA", etc.
	KnownRightsFound  int      `json:"known_rights_found"`
	KnownRightsTotal  int      `json:"known_rights_total"`
	MissingRights     []string `json:"missing_rights,omitempty"`

	// Types found
	RightTypes      []string `json:"right_types,omitempty"`
	ObligationTypes []string `json:"obligation_types,omitempty"`
}

// RegulationType indicates the type of regulation being validated.
type RegulationType string

const (
	RegulationGDPR    RegulationType = "GDPR"
	RegulationCCPA    RegulationType = "CCPA"
	RegulationVCDPA   RegulationType = "VCDPA"
	RegulationGeneric RegulationType = "Generic"
)

// ValidationProfile defines regulation-specific validation criteria and weights.
type ValidationProfile struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`

	// Expected structure
	ExpectedArticles    int `json:"expected_articles"`
	ExpectedDefinitions int `json:"expected_definitions"`
	ExpectedChapters    int `json:"expected_chapters"`

	// Known rights for this regulation
	KnownRights []string `json:"known_rights"`

	// Known obligations for this regulation
	KnownObligations []string `json:"known_obligations"`

	// Scoring weights (must sum to 1.0)
	Weights ValidationWeights `json:"weights"`
}

// ValidationWeights defines the weight for each scoring component.
type ValidationWeights struct {
	ReferenceResolution float64 `json:"reference_resolution"`
	GraphConnectivity   float64 `json:"graph_connectivity"`
	DefinitionCoverage  float64 `json:"definition_coverage"`
	SemanticExtraction  float64 `json:"semantic_extraction"`
	StructureQuality    float64 `json:"structure_quality"`
}

// DefaultWeights returns the default scoring weights.
func DefaultWeights() ValidationWeights {
	return ValidationWeights{
		ReferenceResolution: 0.25,
		GraphConnectivity:   0.20,
		DefinitionCoverage:  0.20,
		SemanticExtraction:  0.20,
		StructureQuality:    0.15,
	}
}

// ValidationProfiles contains the built-in validation profiles.
var ValidationProfiles = map[RegulationType]*ValidationProfile{
	RegulationGDPR: {
		Name:                "GDPR",
		Description:         "General Data Protection Regulation (EU) 2016/679",
		ExpectedArticles:    99,
		ExpectedDefinitions: 26,
		ExpectedChapters:    11,
		KnownRights: []string{
			string(extract.RightAccess),
			string(extract.RightRectification),
			string(extract.RightErasure),
			string(extract.RightRestriction),
			string(extract.RightPortability),
			string(extract.RightObject),
		},
		KnownObligations: []string{
			string(extract.ObligationLawfulProcessing),
			string(extract.ObligationConsent),
			string(extract.ObligationSecure),
			string(extract.ObligationNotifyBreach),
			string(extract.ObligationRecord),
			string(extract.ObligationImpactAssessment),
		},
		Weights: DefaultWeights(),
	},
	RegulationCCPA: {
		Name:                "CCPA",
		Description:         "California Consumer Privacy Act of 2018",
		ExpectedArticles:    21,
		ExpectedDefinitions: 15,
		ExpectedChapters:    6,
		KnownRights: []string{
			string(extract.RightToKnow),
			string(extract.RightToDelete),
			string(extract.RightToOptOut),
			string(extract.RightToNonDiscrimination),
			string(extract.RightToKnowAboutSales),
		},
		KnownObligations: []string{
			string(extract.ObligationNoticeAtCollection),
			string(extract.ObligationPrivacyPolicy),
			string(extract.ObligationVerifyRequest),
			string(extract.ObligationNonDiscrimination),
			string(extract.ObligationServiceProvider),
		},
		Weights: DefaultWeights(),
	},
	RegulationVCDPA: {
		Name:                "VCDPA",
		Description:         "Virginia Consumer Data Protection Act (Code of Virginia Title 59.1 Chapter 53)",
		ExpectedArticles:    12,
		ExpectedDefinitions: 22,
		ExpectedChapters:    7,
		KnownRights: []string{
			string(extract.RightAccess),
			string(extract.RightToKnow),
			string(extract.RightToDelete),
			string(extract.RightToCorrect),
			string(extract.RightPortability),
			string(extract.RightToOptOut),
		},
		KnownObligations: []string{
			string(extract.ObligationPrivacyPolicy),
			string(extract.ObligationImpactAssessment),
			string(extract.ObligationSecure),
			string(extract.ObligationVerifyRequest),
			string(extract.ObligationNonDiscrimination),
		},
		Weights: DefaultWeights(),
	},
	RegulationGeneric: {
		Name:                "Generic",
		Description:         "Generic regulation (minimal criteria)",
		ExpectedArticles:    0, // No expectation
		ExpectedDefinitions: 0,
		ExpectedChapters:    0,
		KnownRights:         []string{},
		KnownObligations:    []string{},
		Weights:             DefaultWeights(),
	},
}

// Validator performs comprehensive validation on regulatory data.
type Validator struct {
	threshold      float64
	regulationType RegulationType
	profile        *ValidationProfile
}

// NewValidator creates a new Validator with the specified threshold.
func NewValidator(threshold float64) *Validator {
	if threshold <= 0 || threshold > 1.0 {
		threshold = 0.80 // Default 80% threshold
	}
	return &Validator{
		threshold:      threshold,
		regulationType: RegulationGeneric,
		profile:        nil, // Will be set on first validation
	}
}

// SetRegulationType sets the regulation type for regulation-aware validation.
func (v *Validator) SetRegulationType(regType RegulationType) {
	v.regulationType = regType
	v.profile = ValidationProfiles[regType]
}

// SetProfile sets a custom validation profile.
func (v *Validator) SetProfile(profile *ValidationProfile) {
	v.profile = profile
}

// GetProfile returns the current validation profile.
func (v *Validator) GetProfile() *ValidationProfile {
	return v.profile
}

// GetAvailableProfiles returns a list of available profile names.
func GetAvailableProfiles() []string {
	profiles := make([]string, 0, len(ValidationProfiles))
	for regType := range ValidationProfiles {
		profiles = append(profiles, string(regType))
	}
	return profiles
}

// DetectRegulationType auto-detects the regulation type from the document.
func (v *Validator) DetectRegulationType(doc *extract.Document) RegulationType {
	if doc == nil {
		return RegulationGeneric
	}

	identifier := strings.ToLower(doc.Identifier)
	title := strings.ToLower(doc.Title)

	// Check for GDPR
	if strings.Contains(identifier, "2016/679") ||
		strings.Contains(title, "gdpr") ||
		strings.Contains(title, "general data protection regulation") {
		return RegulationGDPR
	}

	// Check for CCPA
	if strings.Contains(identifier, "1798") ||
		strings.Contains(identifier, "cal.") ||
		strings.Contains(title, "ccpa") ||
		strings.Contains(title, "california consumer privacy act") ||
		strings.Contains(title, "california privacy") {
		return RegulationCCPA
	}

	// Check for VCDPA
	if strings.Contains(identifier, "59.1") ||
		strings.Contains(title, "vcdpa") ||
		strings.Contains(title, "virginia consumer data protection act") ||
		strings.Contains(title, "virginia privacy") ||
		(strings.Contains(title, "virginia") && strings.Contains(title, "consumer data")) {
		return RegulationVCDPA
	}

	return RegulationGeneric
}

// Validate performs full validation and returns a comprehensive report.
func (v *Validator) Validate(
	doc *extract.Document,
	resolvedRefs []*extract.ResolvedReference,
	definitions []*extract.DefinedTerm,
	termUsages []*extract.TermUsage,
	semantics []*extract.SemanticAnnotation,
	tripleStore *store.TripleStore,
) *ValidationResult {
	result := &ValidationResult{
		Threshold: v.threshold,
		Issues:    make([]ValidationIssue, 0),
		Warnings:  make([]ValidationIssue, 0),
	}

	// Auto-detect regulation type if not set
	if v.regulationType == RegulationGeneric {
		v.regulationType = v.DetectRegulationType(doc)
	}

	// Set profile based on regulation type if not already set
	if v.profile == nil {
		v.profile = ValidationProfiles[v.regulationType]
	}

	// Store profile info in result
	if v.profile != nil {
		result.ProfileName = v.profile.Name
		result.ProfileUsed = v.profile
	}

	// Validate references
	result.References = v.validateReferences(resolvedRefs)

	// Validate connectivity
	result.Connectivity = v.validateConnectivity(doc, tripleStore)

	// Validate definitions
	result.Definitions = v.validateDefinitions(definitions, termUsages)

	// Validate semantics (regulation-aware)
	result.Semantics = v.validateSemantics(semantics)

	// Validate structure (new)
	result.Structure = v.validateStructure(doc, definitions)

	// Calculate overall score and status using weighted scoring
	v.calculateWeightedScore(result)

	return result
}

// validateReferences validates reference resolution.
func (v *Validator) validateReferences(resolved []*extract.ResolvedReference) *ReferenceValidation {
	val := &ReferenceValidation{
		TotalReferences:    len(resolved),
		UnresolvedExamples: make([]ReferenceExample, 0),
		AmbiguousExamples:  make([]ReferenceExample, 0),
	}

	for _, ref := range resolved {
		switch ref.Status {
		case extract.ResolutionResolved:
			val.Resolved++
		case extract.ResolutionPartial:
			val.Partial++
		case extract.ResolutionAmbiguous:
			val.Ambiguous++
			if len(val.AmbiguousExamples) < 5 {
				val.AmbiguousExamples = append(val.AmbiguousExamples, ReferenceExample{
					SourceArticle: ref.Original.SourceArticle,
					RawText:       ref.Original.RawText,
					Reason:        ref.Reason,
				})
			}
		case extract.ResolutionNotFound:
			val.NotFound++
			if len(val.UnresolvedExamples) < 5 {
				val.UnresolvedExamples = append(val.UnresolvedExamples, ReferenceExample{
					SourceArticle: ref.Original.SourceArticle,
					RawText:       ref.Original.RawText,
					Reason:        ref.Reason,
				})
			}
		case extract.ResolutionExternal:
			val.External++
		case extract.ResolutionRangeRef:
			val.RangeRefs++
		}

		// Count confidence levels
		switch {
		case ref.Confidence >= extract.ConfidenceHigh:
			val.HighConfidence++
		case ref.Confidence >= extract.ConfidenceMedium:
			val.MediumConfidence++
		default:
			val.LowConfidence++
		}
	}

	// Calculate resolution rate (excluding external refs)
	internalRefs := val.TotalReferences - val.External
	if internalRefs > 0 {
		successfulResolutions := val.Resolved + val.Partial + val.RangeRefs
		val.ResolutionRate = float64(successfulResolutions) / float64(internalRefs)
	}

	return val
}

// validateConnectivity validates graph connectivity.
func (v *Validator) validateConnectivity(doc *extract.Document, tripleStore *store.TripleStore) *ConnectivityValidation {
	val := &ConnectivityValidation{
		OrphanArticles:  make([]int, 0),
		MostReferenced:  make([]ArticleRefCount, 0),
		MostReferencing: make([]ArticleRefCount, 0),
	}

	// Count provisions
	articles := collectArticles(doc)
	val.TotalProvisions = len(articles)

	// Check connectivity for each article
	incomingCounts := make(map[int]int)
	outgoingCounts := make(map[int]int)

	if tripleStore != nil {
		// Count references
		for _, t := range tripleStore.All() {
			if t.Predicate == store.PropReferences {
				sourceNum := extractArticleNum(t.Subject)
				targetNum := extractArticleNum(t.Object)
				if sourceNum > 0 {
					outgoingCounts[sourceNum]++
				}
				if targetNum > 0 {
					incomingCounts[targetNum]++
				}
			}
		}
	}

	// Check for orphans (no incoming or outgoing references, and not in key articles)
	keyArticles := map[int]bool{1: true, 2: true, 3: true, 4: true, 99: true} // Subject matter, scope, definitions, final provisions

	for _, art := range articles {
		incoming := incomingCounts[art.Number]
		outgoing := outgoingCounts[art.Number]

		if incoming > 0 || outgoing > 0 || keyArticles[art.Number] {
			val.ConnectedCount++
		} else {
			val.OrphanCount++
			val.OrphanArticles = append(val.OrphanArticles, art.Number)
		}
	}

	// Calculate connectivity rate
	if val.TotalProvisions > 0 {
		val.ConnectivityRate = float64(val.ConnectedCount) / float64(val.TotalProvisions)
	}

	// Calculate averages
	var totalIncoming, totalOutgoing int
	for _, count := range incomingCounts {
		totalIncoming += count
	}
	for _, count := range outgoingCounts {
		totalOutgoing += count
	}
	if val.TotalProvisions > 0 {
		val.AvgIncomingRefs = float64(totalIncoming) / float64(val.TotalProvisions)
		val.AvgOutgoingRefs = float64(totalOutgoing) / float64(val.TotalProvisions)
	}

	// Find most referenced
	for artNum, count := range incomingCounts {
		val.MostReferenced = append(val.MostReferenced, ArticleRefCount{
			ArticleNum: artNum,
			Count:      count,
		})
	}
	sort.Slice(val.MostReferenced, func(i, j int) bool {
		return val.MostReferenced[i].Count > val.MostReferenced[j].Count
	})
	if len(val.MostReferenced) > 5 {
		val.MostReferenced = val.MostReferenced[:5]
	}

	// Find most referencing
	for artNum, count := range outgoingCounts {
		val.MostReferencing = append(val.MostReferencing, ArticleRefCount{
			ArticleNum: artNum,
			Count:      count,
		})
	}
	sort.Slice(val.MostReferencing, func(i, j int) bool {
		return val.MostReferencing[i].Count > val.MostReferencing[j].Count
	})
	if len(val.MostReferencing) > 5 {
		val.MostReferencing = val.MostReferencing[:5]
	}

	return val
}

// validateDefinitions validates definition coverage.
func (v *Validator) validateDefinitions(definitions []*extract.DefinedTerm, usages []*extract.TermUsage) *DefinitionValidation {
	val := &DefinitionValidation{
		TotalDefinitions: len(definitions),
		TotalUsages:      len(usages),
		UnusedTerms:      make([]string, 0),
		MostUsedTerms:    make([]TermUsageCount, 0),
	}

	// Track which terms are used
	usedTerms := make(map[string]bool)
	termUsageCounts := make(map[string]int)
	termArticleCounts := make(map[string]map[int]bool)
	articlesWithTerms := make(map[int]bool)

	for _, usage := range usages {
		usedTerms[usage.NormalizedTerm] = true
		termUsageCounts[usage.NormalizedTerm] += usage.Count
		articlesWithTerms[usage.ArticleNum] = true

		if termArticleCounts[usage.NormalizedTerm] == nil {
			termArticleCounts[usage.NormalizedTerm] = make(map[int]bool)
		}
		termArticleCounts[usage.NormalizedTerm][usage.ArticleNum] = true
	}

	val.ArticlesWithTerms = len(articlesWithTerms)

	// Check which definitions are unused
	for _, def := range definitions {
		if usedTerms[def.NormalizedTerm] {
			val.UsedDefinitions++
		} else {
			val.UnusedDefinitions++
			val.UnusedTerms = append(val.UnusedTerms, def.Term)
		}
	}

	// Calculate usage rate
	if val.TotalDefinitions > 0 {
		val.UsageRate = float64(val.UsedDefinitions) / float64(val.TotalDefinitions)
	}

	// Build most used terms list
	for term, count := range termUsageCounts {
		val.MostUsedTerms = append(val.MostUsedTerms, TermUsageCount{
			Term:         term,
			UsageCount:   count,
			ArticleCount: len(termArticleCounts[term]),
		})
	}
	sort.Slice(val.MostUsedTerms, func(i, j int) bool {
		return val.MostUsedTerms[i].UsageCount > val.MostUsedTerms[j].UsageCount
	})
	if len(val.MostUsedTerms) > 10 {
		val.MostUsedTerms = val.MostUsedTerms[:10]
	}

	return val
}

// validateSemantics validates semantic extraction.
func (v *Validator) validateSemantics(annotations []*extract.SemanticAnnotation) *SemanticValidation {
	val := &SemanticValidation{
		MissingRights:   make([]string, 0),
		RightTypes:      make([]string, 0),
		ObligationTypes: make([]string, 0),
		RegulationType:  string(v.regulationType),
	}

	articlesWithRights := make(map[int]bool)
	articlesWithOblig := make(map[int]bool)
	rightTypesFound := make(map[string]bool)
	obligTypesFound := make(map[string]bool)

	for _, ann := range annotations {
		switch ann.Type {
		case extract.SemanticRight:
			val.RightsCount++
			articlesWithRights[ann.ArticleNum] = true
			rightTypesFound[string(ann.RightType)] = true
		case extract.SemanticObligation, extract.SemanticProhibition:
			val.ObligationsCount++
			articlesWithOblig[ann.ArticleNum] = true
			obligTypesFound[string(ann.ObligationType)] = true
		}
	}

	val.ArticlesWithRights = len(articlesWithRights)
	val.ArticlesWithOblig = len(articlesWithOblig)

	// Convert to slices
	for rt := range rightTypesFound {
		val.RightTypes = append(val.RightTypes, rt)
	}
	sort.Strings(val.RightTypes)

	for ot := range obligTypesFound {
		val.ObligationTypes = append(val.ObligationTypes, ot)
	}
	sort.Strings(val.ObligationTypes)

	// Get known rights based on regulation type
	knownRights := v.getKnownRights()
	val.KnownRightsTotal = len(knownRights)

	for rt := range rightTypesFound {
		if _, ok := knownRights[rt]; ok {
			knownRights[rt] = true
		}
	}

	for right, found := range knownRights {
		if found {
			val.KnownRightsFound++
		} else {
			val.MissingRights = append(val.MissingRights, right)
		}
	}
	sort.Strings(val.MissingRights)

	return val
}

// getKnownRights returns the known rights for the current regulation type.
func (v *Validator) getKnownRights() map[string]bool {
	// Use profile if available
	if v.profile != nil && len(v.profile.KnownRights) > 0 {
		rights := make(map[string]bool)
		for _, right := range v.profile.KnownRights {
			rights[right] = false
		}
		return rights
	}

	// Fallback to hardcoded values for backwards compatibility
	switch v.regulationType {
	case RegulationGDPR:
		return map[string]bool{
			string(extract.RightAccess):        false,
			string(extract.RightRectification): false,
			string(extract.RightErasure):       false,
			string(extract.RightRestriction):   false,
			string(extract.RightPortability):   false,
			string(extract.RightObject):        false,
		}
	case RegulationCCPA:
		return map[string]bool{
			string(extract.RightToKnow):              false,
			string(extract.RightToDelete):           false,
			string(extract.RightToOptOut):           false,
			string(extract.RightToNonDiscrimination): false,
			string(extract.RightToKnowAboutSales):   false,
		}
	default:
		// For generic regulations, don't require specific rights
		return make(map[string]bool)
	}
}

// validateStructure validates document structure completeness.
func (v *Validator) validateStructure(doc *extract.Document, definitions []*extract.DefinedTerm) *StructureValidation {
	val := &StructureValidation{}

	// Count structure elements
	articles := collectArticles(doc)
	val.TotalArticles = len(articles)
	val.TotalChapters = len(doc.Chapters)

	// Count sections
	for _, ch := range doc.Chapters {
		val.TotalSections += len(ch.Sections)
	}

	if doc.Preamble != nil {
		val.TotalRecitals = len(doc.Preamble.Recitals)
	}

	// Set expected values from profile
	if v.profile != nil {
		val.ExpectedArticles = v.profile.ExpectedArticles
		val.ExpectedDefinitions = v.profile.ExpectedDefinitions
		val.ExpectedChapters = v.profile.ExpectedChapters
	}

	// Calculate completeness scores
	if val.ExpectedArticles > 0 {
		val.ArticleCompleteness = min(1.0, float64(val.TotalArticles)/float64(val.ExpectedArticles))
	} else {
		val.ArticleCompleteness = 1.0 // No expectation means 100% complete
	}

	if val.ExpectedDefinitions > 0 {
		val.DefinitionCompleteness = min(1.0, float64(len(definitions))/float64(val.ExpectedDefinitions))
	} else {
		val.DefinitionCompleteness = 1.0
	}

	if val.ExpectedChapters > 0 {
		val.ChapterCompleteness = min(1.0, float64(val.TotalChapters)/float64(val.ExpectedChapters))
	} else {
		val.ChapterCompleteness = 1.0
	}

	// Check content quality (articles with meaningful content)
	for _, art := range articles {
		if len(art.Text) > 50 || len(art.Paragraphs) > 0 {
			val.ArticlesWithContent++
		} else {
			val.ArticlesEmpty++
		}
	}

	if val.TotalArticles > 0 {
		val.ContentRate = float64(val.ArticlesWithContent) / float64(val.TotalArticles)
	} else {
		val.ContentRate = 1.0
	}

	// Calculate overall structure score (weighted average of completeness metrics)
	val.StructureScore = (val.ArticleCompleteness*0.4 +
		val.DefinitionCompleteness*0.3 +
		val.ChapterCompleteness*0.2 +
		val.ContentRate*0.1)

	return val
}

// min returns the minimum of two float64 values.
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// calculateWeightedScore computes the overall validation score using weighted scoring.
func (v *Validator) calculateWeightedScore(result *ValidationResult) {
	weights := DefaultWeights()
	if v.profile != nil {
		weights = v.profile.Weights
	}

	// Initialize component scores
	result.ComponentScores = &ComponentScores{
		ReferenceWeight:    weights.ReferenceResolution,
		ConnectivityWeight: weights.GraphConnectivity,
		DefinitionWeight:   weights.DefinitionCoverage,
		SemanticWeight:     weights.SemanticExtraction,
		StructureWeight:    weights.StructureQuality,
	}

	var totalWeight float64
	var weightedSum float64

	// Reference resolution score
	if result.References != nil {
		result.ComponentScores.ReferenceScore = result.References.ResolutionRate
		weightedSum += result.References.ResolutionRate * weights.ReferenceResolution
		totalWeight += weights.ReferenceResolution

		if result.References.NotFound > 0 {
			result.Warnings = append(result.Warnings, ValidationIssue{
				Category: "references",
				Severity: "warning",
				Message:  fmt.Sprintf("%d references could not be resolved", result.References.NotFound),
				Count:    result.References.NotFound,
			})
		}
		if result.References.Ambiguous > 0 {
			result.Warnings = append(result.Warnings, ValidationIssue{
				Category: "references",
				Severity: "info",
				Message:  fmt.Sprintf("%d references are ambiguous and may need manual review", result.References.Ambiguous),
				Count:    result.References.Ambiguous,
			})
		}
	}

	// Connectivity score
	if result.Connectivity != nil {
		result.ComponentScores.ConnectivityScore = result.Connectivity.ConnectivityRate
		weightedSum += result.Connectivity.ConnectivityRate * weights.GraphConnectivity
		totalWeight += weights.GraphConnectivity

		if result.Connectivity.OrphanCount > 0 {
			examples := make([]string, 0)
			for i, artNum := range result.Connectivity.OrphanArticles {
				if i >= 5 {
					break
				}
				examples = append(examples, fmt.Sprintf("Article %d", artNum))
			}
			result.Warnings = append(result.Warnings, ValidationIssue{
				Category: "connectivity",
				Severity: "info",
				Message:  fmt.Sprintf("%d provisions have no cross-references", result.Connectivity.OrphanCount),
				Count:    result.Connectivity.OrphanCount,
				Examples: examples,
			})
		}
	}

	// Definition usage score
	if result.Definitions != nil {
		result.ComponentScores.DefinitionScore = result.Definitions.UsageRate
		weightedSum += result.Definitions.UsageRate * weights.DefinitionCoverage
		totalWeight += weights.DefinitionCoverage

		if result.Definitions.UnusedDefinitions > 0 {
			result.Warnings = append(result.Warnings, ValidationIssue{
				Category: "definitions",
				Severity: "info",
				Message:  fmt.Sprintf("%d defined terms are not referenced in other articles", result.Definitions.UnusedDefinitions),
				Count:    result.Definitions.UnusedDefinitions,
				Examples: result.Definitions.UnusedTerms,
			})
		}
	}

	// Semantics score (based on known rights coverage)
	if result.Semantics != nil && result.Semantics.KnownRightsTotal > 0 {
		semanticScore := float64(result.Semantics.KnownRightsFound) / float64(result.Semantics.KnownRightsTotal)
		result.ComponentScores.SemanticScore = semanticScore
		weightedSum += semanticScore * weights.SemanticExtraction
		totalWeight += weights.SemanticExtraction

		if len(result.Semantics.MissingRights) > 0 {
			regulationName := result.Semantics.RegulationType
			if regulationName == "" {
				regulationName = "known"
			}
			result.Warnings = append(result.Warnings, ValidationIssue{
				Category: "semantics",
				Severity: "warning",
				Message:  fmt.Sprintf("%d %s rights were not detected", len(result.Semantics.MissingRights), regulationName),
				Count:    len(result.Semantics.MissingRights),
				Examples: result.Semantics.MissingRights,
			})
		}
	} else if result.Semantics != nil {
		// For generic regulations without expected rights, use a quality score
		// based on whether any rights/obligations were found
		hasRights := result.Semantics.RightsCount > 0
		hasObligations := result.Semantics.ObligationsCount > 0
		qualityScore := 0.0
		if hasRights && hasObligations {
			qualityScore = 1.0
		} else if hasRights || hasObligations {
			qualityScore = 0.5
		}
		result.ComponentScores.SemanticScore = qualityScore
		weightedSum += qualityScore * weights.SemanticExtraction
		totalWeight += weights.SemanticExtraction
	}

	// Structure score
	if result.Structure != nil {
		result.ComponentScores.StructureScore = result.Structure.StructureScore
		weightedSum += result.Structure.StructureScore * weights.StructureQuality
		totalWeight += weights.StructureQuality

		if result.Structure.ArticlesEmpty > 0 {
			result.Warnings = append(result.Warnings, ValidationIssue{
				Category: "structure",
				Severity: "info",
				Message:  fmt.Sprintf("%d articles have minimal or no content", result.Structure.ArticlesEmpty),
				Count:    result.Structure.ArticlesEmpty,
			})
		}
	}

	// Calculate overall score
	if totalWeight > 0 {
		result.OverallScore = weightedSum / totalWeight
	}

	// Determine status
	if result.OverallScore >= v.threshold {
		result.Status = StatusPass
	} else if result.OverallScore >= v.threshold*0.9 {
		result.Status = StatusWarn
	} else {
		result.Status = StatusFail
		result.Issues = append(result.Issues, ValidationIssue{
			Category: "overall",
			Severity: "error",
			Message:  fmt.Sprintf("Overall score %.1f%% is below threshold %.1f%%", result.OverallScore*100, v.threshold*100),
		})
	}
}

// calculateOverallScore computes the overall validation score and status.
func (v *Validator) calculateOverallScore(result *ValidationResult) {
	scores := make([]float64, 0)

	// Reference resolution score
	if result.References != nil {
		scores = append(scores, result.References.ResolutionRate)

		if result.References.NotFound > 0 {
			result.Warnings = append(result.Warnings, ValidationIssue{
				Category: "references",
				Severity: "warning",
				Message:  fmt.Sprintf("%d references could not be resolved", result.References.NotFound),
				Count:    result.References.NotFound,
			})
		}
		if result.References.Ambiguous > 0 {
			result.Warnings = append(result.Warnings, ValidationIssue{
				Category: "references",
				Severity: "info",
				Message:  fmt.Sprintf("%d references are ambiguous and may need manual review", result.References.Ambiguous),
				Count:    result.References.Ambiguous,
			})
		}
	}

	// Connectivity score
	if result.Connectivity != nil {
		scores = append(scores, result.Connectivity.ConnectivityRate)

		if result.Connectivity.OrphanCount > 0 {
			examples := make([]string, 0)
			for i, artNum := range result.Connectivity.OrphanArticles {
				if i >= 5 {
					break
				}
				examples = append(examples, fmt.Sprintf("Article %d", artNum))
			}
			result.Warnings = append(result.Warnings, ValidationIssue{
				Category: "connectivity",
				Severity: "info",
				Message:  fmt.Sprintf("%d provisions have no cross-references", result.Connectivity.OrphanCount),
				Count:    result.Connectivity.OrphanCount,
				Examples: examples,
			})
		}
	}

	// Definition usage score
	if result.Definitions != nil {
		scores = append(scores, result.Definitions.UsageRate)

		if result.Definitions.UnusedDefinitions > 0 {
			result.Warnings = append(result.Warnings, ValidationIssue{
				Category: "definitions",
				Severity: "info",
				Message:  fmt.Sprintf("%d defined terms are not referenced in other articles", result.Definitions.UnusedDefinitions),
				Count:    result.Definitions.UnusedDefinitions,
				Examples: result.Definitions.UnusedTerms,
			})
		}
	}

	// Semantics score (based on known rights coverage)
	if result.Semantics != nil && result.Semantics.KnownRightsTotal > 0 {
		semanticScore := float64(result.Semantics.KnownRightsFound) / float64(result.Semantics.KnownRightsTotal)
		scores = append(scores, semanticScore)

		if len(result.Semantics.MissingRights) > 0 {
			regulationName := result.Semantics.RegulationType
			if regulationName == "" {
				regulationName = "known"
			}
			result.Warnings = append(result.Warnings, ValidationIssue{
				Category: "semantics",
				Severity: "warning",
				Message:  fmt.Sprintf("%d %s rights were not detected", len(result.Semantics.MissingRights), regulationName),
				Count:    len(result.Semantics.MissingRights),
				Examples: result.Semantics.MissingRights,
			})
		}
	}

	// Calculate average score
	if len(scores) > 0 {
		var total float64
		for _, s := range scores {
			total += s
		}
		result.OverallScore = total / float64(len(scores))
	}

	// Determine status
	if result.OverallScore >= v.threshold {
		result.Status = StatusPass
	} else if result.OverallScore >= v.threshold*0.9 {
		result.Status = StatusWarn
	} else {
		result.Status = StatusFail
		result.Issues = append(result.Issues, ValidationIssue{
			Category: "overall",
			Severity: "error",
			Message:  fmt.Sprintf("Overall score %.1f%% is below threshold %.1f%%", result.OverallScore*100, v.threshold*100),
		})
	}
}

// ToJSON serializes the validation result to JSON.
func (r *ValidationResult) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// String returns a human-readable validation report.
func (r *ValidationResult) String() string {
	var sb strings.Builder

	sb.WriteString("Validation Report\n")
	sb.WriteString("=================\n")

	// Profile used
	if r.ProfileName != "" {
		sb.WriteString(fmt.Sprintf("Profile: %s\n", r.ProfileName))
	}
	sb.WriteString("\n")

	// Reference Resolution
	if r.References != nil {
		sb.WriteString("Reference Resolution:\n")
		sb.WriteString(fmt.Sprintf("  Total references: %d\n", r.References.TotalReferences))
		sb.WriteString(fmt.Sprintf("  Resolved: %d (%.1f%%)\n",
			r.References.Resolved+r.References.Partial+r.References.RangeRefs,
			r.References.ResolutionRate*100))
		sb.WriteString(fmt.Sprintf("  Unresolved: %d\n", r.References.NotFound))
		sb.WriteString(fmt.Sprintf("    - External: %d\n", r.References.External))
		sb.WriteString(fmt.Sprintf("    - Ambiguous: %d\n", r.References.Ambiguous))
		sb.WriteString(fmt.Sprintf("    - Not found: %d\n", r.References.NotFound))

		if len(r.References.UnresolvedExamples) > 0 {
			sb.WriteString("  Examples:\n")
			for _, ex := range r.References.UnresolvedExamples {
				sb.WriteString(fmt.Sprintf("    - Art %d: %q (%s)\n", ex.SourceArticle, ex.RawText, ex.Reason))
			}
		}
		sb.WriteString("\n")
	}

	// Graph Connectivity
	if r.Connectivity != nil {
		sb.WriteString("Graph Connectivity:\n")
		sb.WriteString(fmt.Sprintf("  Total provisions: %d\n", r.Connectivity.TotalProvisions))
		sb.WriteString(fmt.Sprintf("  Connected: %d (%.1f%%)\n",
			r.Connectivity.ConnectedCount, r.Connectivity.ConnectivityRate*100))
		sb.WriteString(fmt.Sprintf("  Orphans: %d\n", r.Connectivity.OrphanCount))

		if len(r.Connectivity.OrphanArticles) > 0 && len(r.Connectivity.OrphanArticles) <= 10 {
			articles := make([]string, len(r.Connectivity.OrphanArticles))
			for i, a := range r.Connectivity.OrphanArticles {
				articles[i] = fmt.Sprintf("%d", a)
			}
			sb.WriteString(fmt.Sprintf("    Articles: %s\n", strings.Join(articles, ", ")))
		}

		if len(r.Connectivity.MostReferenced) > 0 {
			sb.WriteString("  Most referenced:\n")
			for _, arc := range r.Connectivity.MostReferenced {
				sb.WriteString(fmt.Sprintf("    - Article %d: %d references\n", arc.ArticleNum, arc.Count))
			}
		}
		sb.WriteString("\n")
	}

	// Definition Coverage
	if r.Definitions != nil {
		sb.WriteString("Definition Coverage:\n")
		sb.WriteString(fmt.Sprintf("  Defined terms: %d\n", r.Definitions.TotalDefinitions))
		sb.WriteString(fmt.Sprintf("  Terms with usage links: %d (%.1f%%)\n",
			r.Definitions.UsedDefinitions, r.Definitions.UsageRate*100))
		sb.WriteString(fmt.Sprintf("  Total term usages: %d\n", r.Definitions.TotalUsages))
		sb.WriteString(fmt.Sprintf("  Articles using terms: %d\n", r.Definitions.ArticlesWithTerms))

		if len(r.Definitions.UnusedTerms) > 0 {
			sb.WriteString("  Unused terms:\n")
			for _, term := range r.Definitions.UnusedTerms {
				sb.WriteString(fmt.Sprintf("    - %s\n", term))
			}
		}
		sb.WriteString("\n")
	}

	// Semantic Extraction
	if r.Semantics != nil {
		sb.WriteString("Semantic Extraction:\n")
		sb.WriteString(fmt.Sprintf("  Rights found: %d (in %d articles)\n",
			r.Semantics.RightsCount, r.Semantics.ArticlesWithRights))
		sb.WriteString(fmt.Sprintf("  Obligations found: %d (in %d articles)\n",
			r.Semantics.ObligationsCount, r.Semantics.ArticlesWithOblig))
		regulationLabel := r.Semantics.RegulationType
		if regulationLabel == "" {
			regulationLabel = "known"
		}
		sb.WriteString(fmt.Sprintf("  Known %s rights: %d/%d\n",
			regulationLabel, r.Semantics.KnownRightsFound, r.Semantics.KnownRightsTotal))

		if len(r.Semantics.MissingRights) > 0 {
			sb.WriteString("  Missing rights:\n")
			for _, right := range r.Semantics.MissingRights {
				sb.WriteString(fmt.Sprintf("    - %s\n", right))
			}
		}
		sb.WriteString("\n")
	}

	// Structure Quality
	if r.Structure != nil {
		sb.WriteString("Structure Quality:\n")
		sb.WriteString(fmt.Sprintf("  Articles: %d", r.Structure.TotalArticles))
		if r.Structure.ExpectedArticles > 0 {
			sb.WriteString(fmt.Sprintf(" (expected: %d, %.1f%%)",
				r.Structure.ExpectedArticles, r.Structure.ArticleCompleteness*100))
		}
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  Chapters: %d", r.Structure.TotalChapters))
		if r.Structure.ExpectedChapters > 0 {
			sb.WriteString(fmt.Sprintf(" (expected: %d, %.1f%%)",
				r.Structure.ExpectedChapters, r.Structure.ChapterCompleteness*100))
		}
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  Content quality: %.1f%% articles with content\n", r.Structure.ContentRate*100))
		sb.WriteString(fmt.Sprintf("  Structure score: %.1f%%\n", r.Structure.StructureScore*100))
		sb.WriteString("\n")
	}

	// Component Scores (when available)
	if r.ComponentScores != nil {
		sb.WriteString("Component Scores:\n")
		sb.WriteString(fmt.Sprintf("  References:    %.1f%% (weight: %.0f%%)\n",
			r.ComponentScores.ReferenceScore*100, r.ComponentScores.ReferenceWeight*100))
		sb.WriteString(fmt.Sprintf("  Connectivity:  %.1f%% (weight: %.0f%%)\n",
			r.ComponentScores.ConnectivityScore*100, r.ComponentScores.ConnectivityWeight*100))
		sb.WriteString(fmt.Sprintf("  Definitions:   %.1f%% (weight: %.0f%%)\n",
			r.ComponentScores.DefinitionScore*100, r.ComponentScores.DefinitionWeight*100))
		sb.WriteString(fmt.Sprintf("  Semantics:     %.1f%% (weight: %.0f%%)\n",
			r.ComponentScores.SemanticScore*100, r.ComponentScores.SemanticWeight*100))
		sb.WriteString(fmt.Sprintf("  Structure:     %.1f%% (weight: %.0f%%)\n",
			r.ComponentScores.StructureScore*100, r.ComponentScores.StructureWeight*100))
		sb.WriteString("\n")
	}

	// Warnings
	if len(r.Warnings) > 0 {
		sb.WriteString("Warnings:\n")
		for _, w := range r.Warnings {
			sb.WriteString(fmt.Sprintf("  [%s] %s\n", w.Category, w.Message))
		}
		sb.WriteString("\n")
	}

	// Overall Status
	sb.WriteString(fmt.Sprintf("Overall Score: %.1f%%\n", r.OverallScore*100))
	sb.WriteString(fmt.Sprintf("Threshold: %.1f%%\n", r.Threshold*100))
	sb.WriteString(fmt.Sprintf("Status: %s\n", r.Status))

	return sb.String()
}

// collectArticles collects all articles from a document.
func collectArticles(doc *extract.Document) []*extract.Article {
	articles := make([]*extract.Article, 0)
	for _, ch := range doc.Chapters {
		for _, sec := range ch.Sections {
			articles = append(articles, sec.Articles...)
		}
		articles = append(articles, ch.Articles...)
	}
	return articles
}

// extractArticleNum extracts article number from a URI.
func extractArticleNum(uri string) int {
	if idx := strings.Index(uri, ":Art"); idx != -1 {
		rest := uri[idx+4:]
		var numStr strings.Builder
		for _, c := range rest {
			if c >= '0' && c <= '9' {
				numStr.WriteRune(c)
			} else {
				break
			}
		}
		if numStr.Len() > 0 {
			var n int
			for _, c := range numStr.String() {
				n = n*10 + int(c-'0')
			}
			return n
		}
	}
	return 0
}
