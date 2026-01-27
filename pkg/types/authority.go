package types

// InstitutionID identifies an institution.
type InstitutionID struct {
	ID string
}

// InstitutionType represents the type of institution.
type InstitutionType int

const (
	InstitutionTypeLegislature InstitutionType = iota
	InstitutionTypeCourt
	InstitutionTypeExecutiveAgency
	InstitutionTypeRegulatoryBody
	InstitutionTypeProfessionalBody
	InstitutionTypeInternationalOrg
	InstitutionTypeConstitutionalBody
)

// Institution represents an institution that can grant authority.
type Institution struct {
	ID              InstitutionID
	Name            string
	InstitutionType InstitutionType
	Jurisdiction    Jurisdiction
	Parent          *InstitutionID
	Established     Date
}

// Currency represents a currency.
type Currency string

const (
	CurrencyUSD Currency = "USD"
	CurrencyEUR Currency = "EUR"
	CurrencyGBP Currency = "GBP"
	CurrencyJPY Currency = "JPY"
)

// MonetaryLimit represents a monetary limit on authority.
type MonetaryLimit struct {
	Amount   int
	Currency Currency
}

// CaseType represents the type of case.
type CaseType int

const (
	CaseTypeCivil CaseType = iota
	CaseTypeCriminal
	CaseTypeAdministrative
	CaseTypeConstitutional
	CaseTypeAppellate
	CaseTypeOriginalJurisdiction
)

// AuthorityScope represents the scope within which authority applies.
type AuthorityScope struct {
	Geographic    GeographicScope
	SubjectMatter []SubjectMatter
	Jurisdictions []Jurisdiction
	MonetaryLimit *MonetaryLimit
	CaseTypes     []CaseType
}

// Covers checks if this scope covers another scope.
func (s AuthorityScope) Covers(requested AuthorityScope) bool {
	// Check geographic coverage
	if !s.Geographic.Contains(requested.Geographic) {
		return false
	}

	// Check subject matter coverage
	hasGeneral := false
	for _, sm := range s.SubjectMatter {
		if sm == SubjectMatterGeneral {
			hasGeneral = true
			break
		}
	}
	if !hasGeneral {
		for _, req := range requested.SubjectMatter {
			found := false
			for _, granted := range s.SubjectMatter {
				if req == granted {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// Check jurisdictions coverage
	for _, req := range requested.Jurisdictions {
		found := false
		for _, granted := range s.Jurisdictions {
			if req == granted {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// RestrictionID identifies a restriction.
type RestrictionID struct {
	ID string
}

// RestrictionTypeKind represents the kind of restriction.
type RestrictionTypeKind int

const (
	RestrictionTypeTemporal RestrictionTypeKind = iota
	RestrictionTypeProcedural
	RestrictionTypeApprovalRequired
	RestrictionTypeRecusalRequired
	RestrictionTypeQuorumRequired
	RestrictionTypeSupermajorityRequired
	RestrictionTypeNoticeRequired
	RestrictionTypeAppealableOnly
	RestrictionTypeNonDelegable
)

// TimeRange represents a time range within a day.
type TimeRange struct {
	StartHour int
	EndHour   int
}

// RecusalCondition represents a condition requiring recusal.
type RecusalCondition int

const (
	RecusalConditionFinancialInterest RecusalCondition = iota
	RecusalConditionFamilialRelation
	RecusalConditionPriorInvolvement
	RecusalConditionBiasAppearance
	RecusalConditionOther
)

// RestrictionType represents the type of restriction on authority.
type RestrictionType struct {
	Kind              RestrictionTypeKind
	ValidHours        []TimeRange        // for Temporal
	Procedure         *string            // for Procedural
	ApprovalFrom      *InstitutionID     // for ApprovalRequired
	RecusalConditions []RecusalCondition // for RecusalRequired
	QuorumMinimum     *int               // for QuorumRequired
	Threshold         *float64           // for SupermajorityRequired
	NoticeDays        *int               // for NoticeRequired
}

// RestrictionSourceKind represents the source of a restriction.
type RestrictionSourceKind int

const (
	RestrictionSourceConstitutional RestrictionSourceKind = iota
	RestrictionSourceStatutory
	RestrictionSourceRegulatory
	RestrictionSourceProcedural
	RestrictionSourceEthical
)

// RestrictionSource represents where a restriction comes from.
type RestrictionSource struct {
	Kind       RestrictionSourceKind
	Provision  *ProvisionID // for Statutory
	Regulation *string      // for Regulatory
	Rule       *string      // for Procedural
	Code       *string      // for Ethical
}

// Restriction represents a restriction on how authority can be exercised.
type Restriction struct {
	ID              RestrictionID
	Description     string
	RestrictionType RestrictionType
	Source          RestrictionSource
}

// AuthorityID identifies an authority.
type AuthorityID struct {
	ID          string
	GrantedDate Date
}

// AuthorityHolderKind represents the type of authority holder.
type AuthorityHolderKind int

const (
	AuthorityHolderInstitution AuthorityHolderKind = iota
	AuthorityHolderCourt
	AuthorityHolderIndividual
	AuthorityHolderOffice
	AuthorityHolderCommittee
)

// IndividualID identifies an individual.
type IndividualID struct {
	ID string
}

// CredentialType represents the type of credential.
type CredentialType int

const (
	CredentialTypeJudicialAppointment CredentialType = iota
	CredentialTypeBarAdmission
	CredentialTypeLegislativeElection
	CredentialTypeExecutiveAppointment
	CredentialTypeProfessionalLicense
)

// Credential represents a credential held by an individual.
type Credential struct {
	CredentialType CredentialType
	IssuedBy       InstitutionID
	IssuedDate     Date
	Expires        *Date
	Jurisdiction   *Jurisdiction // for BarAdmission
	LicenseType    *string       // for ProfessionalLicense
}

// Individual represents an individual who can hold authority.
type Individual struct {
	ID          IndividualID
	Name        string
	Credentials []Credential
}

// OfficeID identifies an office.
type OfficeID struct {
	ID string
}

// Office represents an office that carries authority.
type Office struct {
	ID          OfficeID
	Title       string
	Institution InstitutionID
	TermLength  *int // In years, nil = lifetime/indefinite
}

// CommitteeID identifies a committee.
type CommitteeID struct {
	ID string
}

// Committee represents a committee that can hold collective authority.
type Committee struct {
	ID      CommitteeID
	Name    string
	Parent  InstitutionID
	Members []IndividualID
	Quorum  int
}

// AuthorityHolder represents who can hold authority.
type AuthorityHolder struct {
	Kind        AuthorityHolderKind
	Institution *Institution
	Court       *Court
	Individual  *Individual
	Office      *Office
	Committee   *Committee
}

// InstitutionHolder creates an authority holder for an institution.
func InstitutionHolder(inst Institution) AuthorityHolder {
	return AuthorityHolder{Kind: AuthorityHolderInstitution, Institution: &inst}
}

// CourtHolder creates an authority holder for a court.
func CourtHolder(court Court) AuthorityHolder {
	return AuthorityHolder{Kind: AuthorityHolderCourt, Court: &court}
}

// IndividualHolder creates an authority holder for an individual.
func IndividualHolder(ind Individual) AuthorityHolder {
	return AuthorityHolder{Kind: AuthorityHolderIndividual, Individual: &ind}
}

// OfficeAuthHolder creates an authority holder for an office.
func OfficeAuthHolder(office Office) AuthorityHolder {
	return AuthorityHolder{Kind: AuthorityHolderOffice, Office: &office}
}

// CommitteeHolder creates an authority holder for a committee.
func CommitteeHolder(committee Committee) AuthorityHolder {
	return AuthorityHolder{Kind: AuthorityHolderCommittee, Committee: &committee}
}

// AuthorityActionKind represents the category of authority action.
type AuthorityActionKind int

const (
	AuthorityActionJudicial AuthorityActionKind = iota
	AuthorityActionLegislative
	AuthorityActionInterpretive
	AuthorityActionEnforcement
	AuthorityActionAdministrative
	AuthorityActionDelegating
)

// JudicialAction represents a judicial authority action.
type JudicialAction int

const (
	JudicialActionHearCase JudicialAction = iota
	JudicialActionIssueInjunction
	JudicialActionHoldInContempt
	JudicialActionIssueWarrant
	JudicialActionSentenceDefendant
	JudicialActionApprovePleaDeal
	JudicialActionCertifyClass
	JudicialActionAppointCounsel
	JudicialActionSealRecord
	JudicialActionTransferVenue
)

// LegislativeAction represents a legislative authority action.
type LegislativeAction int

const (
	LegislativeActionIntroduceBill LegislativeAction = iota
	LegislativeActionVoteOnBill
	LegislativeActionAmendBill
	LegislativeActionFilibusterBill
	LegislativeActionOverrideVeto
	LegislativeActionSubpoena
	LegislativeActionHoldHearing
	LegislativeActionConfirmNominee
	LegislativeActionImpeach
	LegislativeActionDeclareWar
)

// InterpretiveAction represents an interpretive authority action.
type InterpretiveAction int

const (
	InterpretiveActionInterpretStatute InterpretiveAction = iota
	InterpretiveActionInterpretConstitution
	InterpretiveActionInterpretRegulation
	InterpretiveActionInterpretContract
	InterpretiveActionInterpretTreaty
	InterpretiveActionIssueAdvisoryOpinion
)

// EnforcementAction represents an enforcement authority action.
type EnforcementAction int

const (
	EnforcementActionArrest EnforcementAction = iota
	EnforcementActionSearch
	EnforcementActionSeize
	EnforcementActionProsecute
	EnforcementActionIssueSubpoena
	EnforcementActionGrantImmunity
	EnforcementActionExtradite
	EnforcementActionDeport
	EnforcementActionExecuteJudgment
)

// AdministrativeAction represents an administrative authority action.
type AdministrativeAction int

const (
	AdministrativeActionIssueRegulation AdministrativeAction = iota
	AdministrativeActionGrantLicense
	AdministrativeActionRevokeLicense
	AdministrativeActionIssuePermit
	AdministrativeActionConductInvestigation
	AdministrativeActionImposeFine
	AdministrativeActionIssueOrder
)

// DelegatingAction represents a delegating authority action.
type DelegatingAction int

const (
	DelegatingActionDelegate DelegatingAction = iota
	DelegatingActionSubdelegate
	DelegatingActionRevokeDelegate
)

// AuthorityAction represents what an authority permits the holder to do.
type AuthorityAction struct {
	Kind           AuthorityActionKind
	Judicial       *JudicialAction
	Legislative    *LegislativeAction
	Interpretive   *InterpretiveAction
	Enforcement    *EnforcementAction
	Administrative *AdministrativeAction
	Delegating     *DelegatingAction
}

// JudicialAuthorityAction creates a judicial authority action.
func JudicialAuthorityAction(action JudicialAction) AuthorityAction {
	return AuthorityAction{Kind: AuthorityActionJudicial, Judicial: &action}
}

// LegislativeAuthorityAction creates a legislative authority action.
func LegislativeAuthorityAction(action LegislativeAction) AuthorityAction {
	return AuthorityAction{Kind: AuthorityActionLegislative, Legislative: &action}
}

// Matches checks if this action matches another (exact or subtype).
func (a AuthorityAction) Matches(requested AuthorityAction) bool {
	if a.Kind != requested.Kind {
		return false
	}
	switch a.Kind {
	case AuthorityActionJudicial:
		return a.Judicial != nil && requested.Judicial != nil && *a.Judicial == *requested.Judicial
	case AuthorityActionLegislative:
		return a.Legislative != nil && requested.Legislative != nil && *a.Legislative == *requested.Legislative
	case AuthorityActionInterpretive:
		return a.Interpretive != nil && requested.Interpretive != nil && *a.Interpretive == *requested.Interpretive
	case AuthorityActionEnforcement:
		return a.Enforcement != nil && requested.Enforcement != nil && *a.Enforcement == *requested.Enforcement
	case AuthorityActionAdministrative:
		return a.Administrative != nil && requested.Administrative != nil && *a.Administrative == *requested.Administrative
	case AuthorityActionDelegating:
		// DelegateAuthority implies SubdelegateAuthority
		if a.Delegating != nil && requested.Delegating != nil {
			if *a.Delegating == DelegatingActionDelegate && *requested.Delegating == DelegatingActionSubdelegate {
				return true
			}
			return *a.Delegating == *requested.Delegating
		}
		return false
	}
	return false
}

// DelegationID identifies a delegation.
type DelegationID struct {
	ID string
}

// Delegation represents a delegation of authority from one holder to another.
type Delegation struct {
	ID              DelegationID
	Original        *Authority
	DelegatedTo     AuthorityHolder
	DelegatedBy     AuthorityHolder
	DelegatedOn     Date
	ScopeLimitation *AuthorityScope // Can narrow scope
	Expires         *Date
	Revocable       bool
	Subdelegable    bool
}

// DelegationChain represents a chain of delegations from original grant to current holder.
type DelegationChain struct {
	OriginalGrant *Authority
	Delegations   []Delegation
	CurrentHolder AuthorityHolder
}

// Depth returns the depth of the delegation chain.
func (c DelegationChain) Depth() int {
	return len(c.Delegations)
}

// IsValid checks if the delegation chain is valid.
func (c DelegationChain) IsValid() bool {
	for _, d := range c.Delegations {
		if d.Original == nil || !d.Original.Delegable {
			return false
		}
		if d.Expires != nil && d.Expires.Before(Today()) {
			return false
		}
	}
	return true
}

// RevocationID identifies a revocation.
type RevocationID struct {
	ID string
}

// RevocationReason represents the reason for revocation.
type RevocationReason int

const (
	RevocationReasonExpiration RevocationReason = iota
	RevocationReasonResignation
	RevocationReasonRemoval
	RevocationReasonDeath
	RevocationReasonIncapacity
	RevocationReasonMisconduct
	RevocationReasonSuperseded
	RevocationReasonVoluntary
	RevocationReasonStructural
)

// Revocation represents a revocation of previously granted authority.
type Revocation struct {
	ID        RevocationID
	Authority AuthorityID
	RevokedBy AuthorityHolder
	RevokedOn Date
	Reason    RevocationReason
	Effective Date // May differ from RevokedOn
}

// Authority represents authority granted to a holder.
type Authority struct {
	ID              AuthorityID
	Action          AuthorityAction
	GrantedBy       Institution
	GrantedTo       AuthorityHolder
	Scope           AuthorityScope
	Validity        TemporalRange
	Delegable       bool
	Restrictions    []Restriction
	DelegationChain *DelegationChain
}

// IsValid checks if the authority is currently valid.
func (a Authority) IsValid(asOf Date) bool {
	return a.Validity.IsValidAt(asOf)
}

// Covers checks if this authority covers a specific action and scope.
func (a Authority) Covers(action AuthorityAction, scope AuthorityScope) bool {
	if !a.Action.Matches(action) {
		return false
	}
	if !a.Scope.Covers(scope) {
		return false
	}
	return true
}

// CanDelegate checks if this authority can be delegated.
func (a Authority) CanDelegate() bool {
	if !a.Delegable {
		return false
	}
	for _, r := range a.Restrictions {
		if r.RestrictionType.Kind == RestrictionTypeNonDelegable {
			return false
		}
	}
	return true
}

// GrantingInstitution returns the original granting institution.
func (a Authority) GrantingInstitution() Institution {
	if a.DelegationChain != nil && a.DelegationChain.OriginalGrant != nil {
		return a.DelegationChain.OriginalGrant.GrantedBy
	}
	return a.GrantedBy
}
