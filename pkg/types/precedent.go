package types

// CitationKind represents the type of citation format.
type CitationKind int

const (
	CitationNeutral CitationKind = iota
	CitationReporter
	CitationParallel
)

// Citation uniquely identifies a judicial decision.
type Citation struct {
	Kind CitationKind

	// For Neutral
	Jurisdiction *Jurisdiction
	CourtCode    *string
	Year         int
	Number       int

	// For Reporter
	Volume   int
	Reporter string
	Page     int

	// For Parallel
	Parallel []Citation
}

// NeutralCitation creates a neutral citation.
func NeutralCitation(jurisdiction Jurisdiction, court string, year, number int) Citation {
	return Citation{
		Kind:         CitationNeutral,
		Jurisdiction: &jurisdiction,
		CourtCode:    &court,
		Year:         year,
		Number:       number,
	}
}

// ReporterCitation creates a reporter citation.
func ReporterCitation(volume int, reporter string, page, year int) Citation {
	return Citation{
		Kind:     CitationReporter,
		Volume:   volume,
		Reporter: reporter,
		Page:     page,
		Year:     year,
	}
}

// JudicialTitle represents a judge's title.
type JudicialTitle int

const (
	JudicialTitleChiefJustice JudicialTitle = iota
	JudicialTitleAssociateJustice
	JudicialTitleJustice
	JudicialTitleJudge
	JudicialTitleMagistrate
	JudicialTitleLordChancellor
	JudicialTitleLordChiefJustice
	JudicialTitleMasterOfTheRolls
	JudicialTitleLordJusticeOfAppeal
)

// Judge represents a judge.
type Judge struct {
	Name      string
	Title     JudicialTitle
	Appointed Date
	Retired   *Date
}

// KeyFact represents a key fact in a case.
type KeyFact struct {
	Description string
	Relevance   string // Why this fact matters
}

// DisputedFact represents a disputed fact.
type DisputedFact struct {
	Description      string
	PlaintiffVersion string
	DefendantVersion string
	Finding          *string // Court's finding
}

// FactPattern represents the facts of a case.
type FactPattern struct {
	Summary       string
	KeyFacts      []KeyFact
	DisputedFacts []DisputedFact
}

// HowReached represents how a case reached the court.
type HowReached int

const (
	HowReachedOriginalJurisdiction HowReached = iota
	HowReachedAppealFromTrial
	HowReachedAppealFromIntermediate
	HowReachedCertiorari
	HowReachedLeaveToAppeal
	HowReachedCaseStated
	HowReachedReference
)

// PriorProceeding represents a prior proceeding in the case.
type PriorProceeding struct {
	Court    string
	Decision string
	Date     Date
}

// ProceduralHistory represents the procedural history of a case.
type ProceduralHistory struct {
	PriorProceedings []PriorProceeding
	HowReached       HowReached
}

// OpinionType represents the type of judicial opinion.
type OpinionType int

const (
	OpinionTypeMajority OpinionType = iota
	OpinionTypePlurality
	OpinionTypeConcurrence
	OpinionTypeConcurrenceInJudgment
	OpinionTypeDissent
	OpinionTypeDissentInPart
)

// RatioScope represents how broadly/narrowly the ratio should be read.
type RatioScope int

const (
	RatioScopeBroad RatioScope = iota
	RatioScopeNarrow
	RatioScopeDisputed
)

// RatioScopeInfo contains scope information.
type RatioScopeInfo struct {
	Kind        RatioScope
	Description *string // for Broad/Narrow
}

// Condition represents a condition in a legal proposition.
type Condition struct {
	Description string
}

// Exception represents an exception in a legal proposition.
type Exception struct {
	Description string
}

// LegalProposition represents an abstract statement of law.
type LegalProposition struct {
	Statement  string
	Conditions []Condition
	Exceptions []Exception
}

// RatioDecidendi represents the binding principle of a decision.
type RatioDecidendi struct {
	Proposition   LegalProposition
	ParagraphRefs []int // Where stated in opinion
	Scope         RatioScopeInfo
}

// ObiterSignificance represents the significance of obiter dictum.
type ObiterSignificance int

const (
	ObiterSignificanceMereObservation ObiterSignificance = iota
	ObiterSignificanceSignallingFuture
	ObiterSignificanceClarifyingRatio
	ObiterSignificanceHypothetical
)

// ObiterDictum represents a statement said in passing (not binding).
type ObiterDictum struct {
	Statement    string
	ParagraphRef int
	Significance ObiterSignificance
}

// InterpretationMethod represents how a provision was interpreted.
type InterpretationMethod int

const (
	InterpretationMethodTextual InterpretationMethod = iota
	InterpretationMethodPurposive
	InterpretationMethodContextual
	InterpretationMethodHistorical
	InterpretationMethodSystematic
	InterpretationMethodComparative
	InterpretationMethodConstitutional
)

// ProvisionInterpretation represents an interpretation of a provision.
type ProvisionInterpretation struct {
	Provision      ProvisionID
	Interpretation LegalProposition
	Method         InterpretationMethod
	ParagraphRef   int
}

// PrecedentRelation represents the relationship between precedents.
type PrecedentRelation int

const (
	PrecedentRelationFollows PrecedentRelation = iota
	PrecedentRelationApplies
	PrecedentRelationExtends
	PrecedentRelationDistinguishes
	PrecedentRelationLimits
	PrecedentRelationCriticizes
	PrecedentRelationDoubts
	PrecedentRelationDisapproves
	PrecedentRelationOverrules
	PrecedentRelationOverrulesInPart
	PrecedentRelationHarmonizes
	PrecedentRelationConsidersObiter
	PrecedentRelationNotFollowed
	PrecedentRelationAffirms
	PrecedentRelationReverses
)

// TreatmentType represents how a precedent was treated.
type TreatmentType int

const (
	TreatmentTypeDiscussed TreatmentType = iota
	TreatmentTypeMentioned
	TreatmentTypeQuoted
	TreatmentTypeCriticized
	TreatmentTypeApproved
	TreatmentTypeDistinguished
	TreatmentTypeApplied
)

// PrecedentCitation represents a citation to precedent within an opinion.
type PrecedentCitation struct {
	CitedCase    Citation
	Relationship PrecedentRelation
	Paragraph    *int
	Proposition  LegalProposition
	Treatment    TreatmentType
}

// ReasoningParagraph represents a paragraph of reasoning in an opinion.
type ReasoningParagraph struct {
	Number    int
	Text      string
	Citations []PrecedentCitation
	IsRatio   bool // Part of ratio decidendi?
}

// Opinion represents a judicial opinion.
type Opinion struct {
	Author                 Judge
	JoinedBy               []Judge
	OpinionType            OpinionType
	Reasoning              []ReasoningParagraph
	Ratio                  []RatioDecidendi
	Obiter                 []ObiterDictum
	ProvisionsInterpreted  []ProvisionInterpretation
	PrecedentsCited        []PrecedentCitation
}

// LegalIssue represents a legal issue presented in a case.
type LegalIssue struct {
	Question string
	Answer   *string
	Resolved bool
}

// Disposition represents the disposition of a case.
type Disposition int

const (
	DispositionAffirmed Disposition = iota
	DispositionReversed
	DispositionReversedInPart
	DispositionRemanded
	DispositionDismissed
	DispositionAllowed
	DispositionDismissedWithCosts
)

// Holding represents the court's decision on an issue.
type Holding struct {
	Issue         LegalIssue
	Proposition   LegalProposition
	Disposition   Disposition
	BindingOn     []Court
	PersuasiveFor []Court
}

// RemedyType represents the type of remedy granted.
type RemedyType int

const (
	RemedyTypeDamages RemedyType = iota
	RemedyTypeInjunction
	RemedyTypeDeclaratoryRelief
	RemedyTypeSpecificPerformance
	RemedyTypeRescission
	RemedyTypeRestitution
	RemedyTypeMandamus
	RemedyTypeCertiorari
	RemedyTypeHabeas
	RemedyTypeOther
)

// Remedy represents the remedy granted in a case.
type Remedy struct {
	Type        RemedyType
	Details     string
	Amount      *int   // for Damages
	Description *string // for Other
}

// CostsOrderType represents the type of costs order.
type CostsOrderType int

const (
	CostsOrderNoOrder CostsOrderType = iota
	CostsOrderToPlaintiff
	CostsOrderToDefendant
	CostsOrderInCause
	CostsOrderEachPartyOwn
	CostsOrderIndemnityBasis
)

// CostsOrder represents a costs order.
type CostsOrder struct {
	Order  CostsOrderType
	Amount *int
}

// Decision represents a judicial decision.
type Decision struct {
	Citation          Citation
	CaseName          string
	Court             Court
	Decided           Date
	Argued            *Date
	Judges            []Judge
	Presiding         Judge
	Majority          Opinion
	Concurrences      []Opinion
	Dissents          []Opinion
	Facts             FactPattern
	ProceduralHistory ProceduralHistory
	Issues            []LegalIssue
	Holdings          []Holding
	Remedy            *Remedy
	Costs             *CostsOrder
}

// BindingStrength represents the strength of binding precedent.
type BindingStrength int

const (
	BindingStrengthAbsolute BindingStrength = iota // Highest court, directly on point
	BindingStrengthStrong                           // High court, clear statement
	BindingStrengthQualified                        // May be distinguishable
)

// PersuasiveStrength represents the strength of persuasive precedent.
type PersuasiveStrength int

const (
	PersuasiveStrengthHighly PersuasiveStrength = iota // Same level court, well-reasoned
	PersuasiveStrengthModerate                         // Worth considering
	PersuasiveStrengthWeak                             // Noted but not compelling
)

// ForeignStrength represents the strength of foreign precedent.
type ForeignStrength int

const (
	ForeignStrengthInfluential ForeignStrength = iota // From respected foreign court
	ForeignStrengthConsidered                         // Worth noting
	ForeignStrengthMereReference                      // Mentioned only
)

// PrecedentWeightKind represents the type of precedent weight.
type PrecedentWeightKind int

const (
	PrecedentWeightBinding PrecedentWeightKind = iota
	PrecedentWeightPersuasive
	PrecedentWeightForeignPersuasive
	PrecedentWeightNotApplicable
)

// PrecedentWeight represents the weight of precedent for a court.
type PrecedentWeight struct {
	Kind              PrecedentWeightKind
	BindingStrength   *BindingStrength
	PersuasiveStrength *PersuasiveStrength
	ForeignStrength   *ForeignStrength
}

// BindingWeight creates a binding precedent weight.
func BindingWeight(strength BindingStrength) PrecedentWeight {
	return PrecedentWeight{Kind: PrecedentWeightBinding, BindingStrength: &strength}
}

// PersuasiveWeight creates a persuasive precedent weight.
func PersuasiveWeight(strength PersuasiveStrength) PrecedentWeight {
	return PrecedentWeight{Kind: PrecedentWeightPersuasive, PersuasiveStrength: &strength}
}

// ForeignWeight creates a foreign persuasive precedent weight.
func ForeignWeight(strength ForeignStrength) PrecedentWeight {
	return PrecedentWeight{Kind: PrecedentWeightForeignPersuasive, ForeignStrength: &strength}
}

// NotApplicableWeight creates a not applicable precedent weight.
func NotApplicableWeight() PrecedentWeight {
	return PrecedentWeight{Kind: PrecedentWeightNotApplicable}
}

// IsBinding checks if this decision is binding on a court.
func (d Decision) IsBinding(onCourt Court) bool {
	// Check court level
	if d.Court.Level <= onCourt.Level {
		return false
	}
	// Check geographic scope
	if !d.Court.Geographic.Contains(onCourt.Geographic) {
		return false
	}
	return true
}

// PrecedentWeightFor calculates the precedential weight for a court.
func (d Decision) PrecedentWeightFor(court Court) PrecedentWeight {
	if d.IsBinding(court) {
		if d.Court.Level == CourtLevelHighestAppellate {
			return BindingWeight(BindingStrengthAbsolute)
		}
		return BindingWeight(BindingStrengthStrong)
	}

	// Same jurisdiction but not binding
	if d.Court.Jurisdiction == court.Jurisdiction {
		if d.Court.Level == court.Level {
			return PersuasiveWeight(PersuasiveStrengthHighly)
		}
		return PersuasiveWeight(PersuasiveStrengthModerate)
	}

	// Foreign jurisdiction
	if d.Court.Level >= CourtLevelIntermediateAppellate {
		return ForeignWeight(ForeignStrengthInfluential)
	}
	return ForeignWeight(ForeignStrengthConsidered)
}

// CitingCase represents a case citing another decision.
type CitingCase struct {
	CitingDecision Citation
	Relationship   PrecedentRelation
	Paragraph      int
	Treatment      TreatmentType
}
