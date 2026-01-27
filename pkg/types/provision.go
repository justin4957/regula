package types

// ProvisionID uniquely identifies a provision.
type ProvisionID struct {
	Jurisdiction Jurisdiction
	ActID        ActID
	Section      SectionNumber
	Subsection   *SubsectionNumber
	Paragraph    *ParagraphNumber
}

// ActID uniquely identifies an act.
type ActID struct {
	Jurisdiction Jurisdiction
	ShortTitle   string
	Year         int
	Chapter      *int // Chapter number in session laws
}

// SectionNumberKind represents the type of section numbering.
type SectionNumberKind int

const (
	SectionNumberNumeric SectionNumberKind = iota
	SectionNumberAlphanumeric
)

// SectionNumber represents section numbering (can be numeric or alphanumeric).
type SectionNumber struct {
	Kind    SectionNumberKind
	Numeric int    // for Numeric
	Alpha   string // for Alphanumeric (e.g., "12A", "3bis")
}

// NumericSection creates a numeric section number.
func NumericSection(n int) SectionNumber {
	return SectionNumber{Kind: SectionNumberNumeric, Numeric: n}
}

// AlphanumericSection creates an alphanumeric section number.
func AlphanumericSection(s string) SectionNumber {
	return SectionNumber{Kind: SectionNumberAlphanumeric, Alpha: s}
}

// SubsectionNumberKind represents the type of subsection numbering.
type SubsectionNumberKind int

const (
	SubsectionNumberNumeric SubsectionNumberKind = iota
	SubsectionNumberLetter
	SubsectionNumberRoman
)

// SubsectionNumber represents subsection numbering.
type SubsectionNumber struct {
	Kind    SubsectionNumberKind
	Numeric int    // for Numeric
	Letter  rune   // for Letter (a), (b), etc.
	Roman   string // for Roman (i), (ii), etc.
}

// ParagraphNumber represents paragraph numbering within subsections.
type ParagraphNumber struct {
	Kind    SubsectionNumberKind // Reuse the same kinds
	Numeric int
	Letter  rune
}

// TextSpan represents a span of text within a provision.
type TextSpan struct {
	Start int
	End   int
}

// ReferenceType represents the type of cross-reference.
type ReferenceType int

const (
	ReferenceTypeCitation ReferenceType = iota
	ReferenceTypeIncorporation
	ReferenceTypeException
	ReferenceTypeCondition
	ReferenceTypeDefinition
)

// CrossReference represents a cross-reference to another provision.
type CrossReference struct {
	Target   ProvisionID
	TextSpan TextSpan
	RefType  ReferenceType
}

// DefinitionScopeKind represents the scope of a defined term.
type DefinitionScopeKind int

const (
	DefinitionScopeWholeAct DefinitionScopeKind = iota
	DefinitionScopePart
	DefinitionScopeSection
	DefinitionScopeThisProvision
)

// DefinitionScope represents where a defined term applies.
type DefinitionScope struct {
	Kind    DefinitionScopeKind
	Part    *int           // for Part scope
	Section *SectionNumber // for Section scope
}

// DefinedTerm represents a term defined within the act.
type DefinedTerm struct {
	Term       string
	Definition string
	Scope      DefinitionScope
}

// AnnotationAuthor represents who authored an annotation.
type AnnotationAuthor int

const (
	AnnotationAuthorLegislative AnnotationAuthor = iota
	AnnotationAuthorJudicial
	AnnotationAuthorEditorial
	AnnotationAuthorScholarly
)

// Annotation represents editorial annotations on legal text.
type Annotation struct {
	Text   string
	Author AnnotationAuthor
	Date   Date
}

// LegalText represents structured legal text with cross-references.
type LegalText struct {
	Raw         string
	Language    Language
	References  []CrossReference
	Definitions []DefinedTerm
	Annotations []Annotation
}

// LegislativeNoteKind represents the type of legislative note.
type LegislativeNoteKind int

const (
	LegislativeNoteMarginal LegislativeNoteKind = iota
	LegislativeNoteFootnote
	LegislativeNoteHistorical
	LegislativeNoteComparativeLaw
)

// LegislativeNote represents notes attached to provisions.
type LegislativeNote struct {
	Kind LegislativeNoteKind
	Text string
}

// Provision represents a single provision within an act.
type Provision struct {
	ID         ProvisionID
	ParentAct  ActID
	Section    SectionNumber
	Subsection *SubsectionNumber
	Paragraph  *ParagraphNumber
	Heading    *string
	Text       LegalText
	Validity   TemporalValidity
	Amendments []AmendmentRecord
	Notes      []LegislativeNote
}

// IsValidAt checks if the provision is valid at a given date.
func (p Provision) IsValidAt(date Date) bool {
	return p.Validity.IsValidAt(date)
}

// IsCurrent checks if the provision is currently in force.
func (p Provision) IsCurrent() bool {
	return p.Validity.IsCurrent()
}

// ScheduleContentKind represents the type of schedule content.
type ScheduleContentKind int

const (
	ScheduleContentTable ScheduleContentKind = iota
	ScheduleContentText
	ScheduleContentForm
	ScheduleContentList
	ScheduleContentMixed
)

// FormFieldType represents the type of form field.
type FormFieldType int

const (
	FormFieldText FormFieldType = iota
	FormFieldDate
	FormFieldNumber
	FormFieldCheckbox
	FormFieldSelect
	FormFieldSignature
)

// FormField represents a field in a form template.
type FormField struct {
	Name      string
	FieldType FormFieldType
	Required  bool
	Options   []string // for Select type
}

// FormTemplate represents a form in a schedule.
type FormTemplate struct {
	Title        string
	Fields       []FormField
	Instructions *string
}

// ScheduleContent represents the content of a schedule.
type ScheduleContent struct {
	Kind    ScheduleContentKind
	Headers []string           // for Table
	Rows    [][]string         // for Table
	Text    *LegalText         // for Text
	Form    *FormTemplate      // for Form
	Items   []string           // for List
	Parts   []ScheduleContent  // for Mixed
}

// Schedule represents a schedule attached to an act.
type Schedule struct {
	Number   int
	Title    string
	Content  ScheduleContent
	Validity TemporalValidity
}

// CommencementRuleKind represents how an act comes into force.
type CommencementRuleKind int

const (
	CommencementOnEnactment CommencementRuleKind = iota
	CommencementOnRoyalAssent
	CommencementOnDate
	CommencementOnProclamation
	CommencementOnRegulation
	CommencementStaged
)

// CommencementStage represents a stage in staged commencement.
type CommencementStage struct {
	Sections []SectionNumber
	Rule     CommencementRule
}

// CommencementRule represents when legislation comes into force.
type CommencementRule struct {
	Kind   CommencementRuleKind
	Date   *Date               // for OnDate
	Stages []CommencementStage // for Staged
}

// SectionCommencement represents commencement for a specific section.
type SectionCommencement struct {
	Section SectionNumber
	Rule    CommencementRule
}

// CommencementRules represents all commencement rules for an act.
type CommencementRules struct {
	Default    CommencementRule
	Exceptions []SectionCommencement
}

// SavingsScopeKind represents the scope of a savings clause.
type SavingsScopeKind int

const (
	SavingsScopeAllProceedings SavingsScopeKind = iota
	SavingsScopeExistingRights
	SavingsScopePendingActions
	SavingsScopeSpecific
)

// SavingsScope represents the scope of a savings clause.
type SavingsScope struct {
	Kind        SavingsScopeKind
	Description *string // for Specific
}

// SavingsClause preserves certain effects of repealed law.
type SavingsClause struct {
	Description string
	Scope       SavingsScope
}

// RepealRecord records a repeal of legislation.
type RepealRecord struct {
	RepealedProvision ProvisionID
	RepealingAct      ActID
	RepealingSection  SectionNumber
	Effective         Date
	Savings           *SavingsClause
}

// TransitionalScopeKind represents the scope of transitional provisions.
type TransitionalScopeKind int

const (
	TransitionalScopePreExisting TransitionalScopeKind = iota
	TransitionalScopePending
	TransitionalScopeSpecific
)

// TransitionalScope represents what a transitional provision applies to.
type TransitionalScope struct {
	Kind        TransitionalScopeKind
	Description *string // for Specific
}

// TransitionalProvision represents a transitional provision.
type TransitionalProvision struct {
	Provision Provision
	AppliesTo TransitionalScope
	Expires   *Date
}

// Act represents a complete legislative act.
type Act struct {
	ID           ActID
	ShortTitle   string
	LongTitle    string
	Preamble     *LegalText
	Enacted      Date
	RoyalAssent  *Date // For Commonwealth countries
	Provisions   []Provision
	Schedules    []Schedule
	Commencement CommencementRules
	Repeals      []RepealRecord
	Transitional []TransitionalProvision
}

// FindProvision finds a provision by section number.
func (a Act) FindProvision(section SectionNumber) *Provision {
	for i := range a.Provisions {
		if a.Provisions[i].Section == section {
			return &a.Provisions[i]
		}
	}
	return nil
}

// PartStructure represents the structure of a part within an act.
type PartStructure struct {
	PartNumber *int
	Title      *string
	Divisions  []DivisionStructure
}

// DivisionStructure represents the structure of a division.
type DivisionStructure struct {
	DivisionNumber *int
	Title          *string
	Sections       []SectionNumber
}

// ActStructure represents the hierarchical structure of an act.
type ActStructure struct {
	ActID         ActID
	Parts         []PartStructure
	ScheduleCount int
}
