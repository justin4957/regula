package types

import (
	"errors"
	"fmt"
)

// Proof represents a verified assertion about legal relationships.
// Unlike Crisp's compile-time proofs, Go proofs are runtime-verified.
// Functions requiring proofs should call Verify() and handle errors.
type Proof interface {
	// Verify checks if the proof is valid.
	Verify() error

	// ProofType returns the type of proof.
	ProofType() string

	// Evidence returns the chain of reasoning.
	Evidence() []Evidence
}

// Evidence represents a piece of evidence supporting a proof.
type Evidence struct {
	Description string
	Source      string
}

// BindsOnProof proves that one court's decisions bind another court.
type BindsOnProof struct {
	HigherCourt Court
	LowerCourt  Court
}

func (p *BindsOnProof) ProofType() string {
	return "BindsOn"
}

func (p *BindsOnProof) Verify() error {
	// Check court level
	if p.HigherCourt.Level <= p.LowerCourt.Level {
		return fmt.Errorf("court %s (level %v) does not outrank court %s (level %v)",
			p.HigherCourt.Name, p.HigherCourt.Level,
			p.LowerCourt.Name, p.LowerCourt.Level)
	}

	// Check geographic coverage
	if !p.HigherCourt.Geographic.Contains(p.LowerCourt.Geographic) {
		return fmt.Errorf("court %s geographic scope does not cover court %s",
			p.HigherCourt.Name, p.LowerCourt.Name)
	}

	return nil
}

func (p *BindsOnProof) Evidence() []Evidence {
	return []Evidence{
		{
			Description: fmt.Sprintf("%s is level %v, %s is level %v",
				p.HigherCourt.Name, p.HigherCourt.Level,
				p.LowerCourt.Name, p.LowerCourt.Level),
			Source: "Court hierarchy",
		},
		{
			Description: fmt.Sprintf("%s geographic scope contains %s",
				p.HigherCourt.Name, p.LowerCourt.Name),
			Source: "Geographic analysis",
		},
	}
}

// ProveBindsOn attempts to construct a BindsOnProof.
// Returns nil if the relationship cannot be proven.
func ProveBindsOn(higher, lower Court) *BindsOnProof {
	proof := &BindsOnProof{
		HigherCourt: higher,
		LowerCourt:  lower,
	}
	if proof.Verify() != nil {
		return nil
	}
	return proof
}

// HasJurisdictionProof proves that a court has jurisdiction over a matter.
type HasJurisdictionProof struct {
	Court  Court
	Matter LegalMatter
}

// LegalMatter represents a legal matter that can be before a court.
type LegalMatter struct {
	MatterType      SubjectMatter
	Location        GeographicScope
	Parties         []Party
	AmountInDispute *int
	FederalQuestion bool
}

// Party represents a party to a legal matter.
type Party struct {
	Name      string
	PartyType PartyType
	Domicile  Jurisdiction
}

// PartyType represents the type of party.
type PartyType int

const (
	PartyTypeIndividual PartyType = iota
	PartyTypeCorporation
	PartyTypeGovernment
	PartyTypeOrganization
)

func (p *HasJurisdictionProof) ProofType() string {
	return "HasJurisdiction"
}

func (p *HasJurisdictionProof) Verify() error {
	// Check subject matter jurisdiction
	if p.Court.SubjectMatter != SubjectMatterGeneral && p.Court.SubjectMatter != p.Matter.MatterType {
		return fmt.Errorf("court %s has subject matter %v, matter requires %v",
			p.Court.Name, p.Court.SubjectMatter, p.Matter.MatterType)
	}

	// Check geographic jurisdiction
	if !p.Court.Geographic.Contains(p.Matter.Location) {
		return fmt.Errorf("court %s geographic scope does not cover matter location",
			p.Court.Name)
	}

	return nil
}

func (p *HasJurisdictionProof) Evidence() []Evidence {
	return []Evidence{
		{
			Description: fmt.Sprintf("Court %s has subject matter jurisdiction for %v matters",
				p.Court.Name, p.Matter.MatterType),
			Source: "Subject matter analysis",
		},
		{
			Description: fmt.Sprintf("Court %s geographic scope covers matter location",
				p.Court.Name),
			Source: "Geographic analysis",
		},
	}
}

// ProveHasJurisdiction attempts to construct a HasJurisdictionProof.
func ProveHasJurisdiction(court Court, matter LegalMatter) *HasJurisdictionProof {
	proof := &HasJurisdictionProof{
		Court:  court,
		Matter: matter,
	}
	if proof.Verify() != nil {
		return nil
	}
	return proof
}

// IsGoodLawProof proves that a case is still good law as of a date.
type IsGoodLawProof struct {
	Citation     Citation
	AsOf         Date
	NotOverruled bool
	NotSuperseded bool
	VerifiedBy   string // Source of verification
}

func (p *IsGoodLawProof) ProofType() string {
	return "IsGoodLaw"
}

func (p *IsGoodLawProof) Verify() error {
	if !p.NotOverruled {
		return errors.New("case has been overruled")
	}
	if !p.NotSuperseded {
		return errors.New("case has been superseded by statute")
	}
	return nil
}

func (p *IsGoodLawProof) Evidence() []Evidence {
	return []Evidence{
		{
			Description: "No overruling decision found",
			Source:      p.VerifiedBy,
		},
		{
			Description: "No superseding statute found",
			Source:      p.VerifiedBy,
		},
	}
}

// HasAuthorityProof proves that an entity has valid authority for an action.
type HasAuthorityProof struct {
	Holder    AuthorityHolder
	Action    AuthorityAction
	Scope     AuthorityScope
	Authority *Authority
}

func (p *HasAuthorityProof) ProofType() string {
	return "HasAuthority"
}

func (p *HasAuthorityProof) Verify() error {
	if p.Authority == nil {
		return errors.New("no authority provided")
	}

	// Check temporal validity
	if !p.Authority.IsValid(Today()) {
		return errors.New("authority is not temporally valid")
	}

	// Check action matches
	if !p.Authority.Action.Matches(p.Action) {
		return errors.New("authority action does not match requested action")
	}

	// Check scope coverage
	if !p.Authority.Scope.Covers(p.Scope) {
		return errors.New("authority scope does not cover requested scope")
	}

	// Check delegation chain if present
	if p.Authority.DelegationChain != nil && !p.Authority.DelegationChain.IsValid() {
		return errors.New("delegation chain is invalid")
	}

	return nil
}

func (p *HasAuthorityProof) Evidence() []Evidence {
	evidence := []Evidence{
		{
			Description: fmt.Sprintf("Authority %s is temporally valid", p.Authority.ID.ID),
			Source:      "Authority record",
		},
		{
			Description: "Action matches authority grant",
			Source:      "Action comparison",
		},
		{
			Description: "Scope is covered by authority",
			Source:      "Scope analysis",
		},
	}

	if p.Authority.DelegationChain != nil {
		evidence = append(evidence, Evidence{
			Description: fmt.Sprintf("Delegation chain depth: %d", p.Authority.DelegationChain.Depth()),
			Source:      "Delegation analysis",
		})
	}

	return evidence
}

// ProveHasAuthority attempts to construct a HasAuthorityProof.
func ProveHasAuthority(holder AuthorityHolder, action AuthorityAction, scope AuthorityScope, auth *Authority) *HasAuthorityProof {
	proof := &HasAuthorityProof{
		Holder:    holder,
		Action:    action,
		Scope:     scope,
		Authority: auth,
	}
	if proof.Verify() != nil {
		return nil
	}
	return proof
}

// ValidDelegationProof proves that a delegation chain is valid.
type ValidDelegationProof struct {
	Chain DelegationChain
}

func (p *ValidDelegationProof) ProofType() string {
	return "ValidDelegation"
}

func (p *ValidDelegationProof) Verify() error {
	if !p.Chain.IsValid() {
		return errors.New("delegation chain is invalid")
	}
	return nil
}

func (p *ValidDelegationProof) Evidence() []Evidence {
	evidence := []Evidence{
		{
			Description: fmt.Sprintf("Chain has %d delegations", len(p.Chain.Delegations)),
			Source:      "Delegation chain",
		},
	}

	for i, d := range p.Chain.Delegations {
		evidence = append(evidence, Evidence{
			Description: fmt.Sprintf("Delegation %d: delegable=%v, not expired", i+1, d.Original != nil && d.Original.Delegable),
			Source:      fmt.Sprintf("Delegation %s", d.ID.ID),
		})
	}

	return evidence
}

// PrecedentApplication represents an application of precedent to a court's decision.
type PrecedentApplication struct {
	Precedent        Citation
	AppliedTo        string
	RatioApplied     []RatioDecidendi
	BindingStrength  BindingStrength
	ApplicationDate  Date
}

// ApplyPrecedent applies a precedent to a court's decision.
// Requires valid BindsOn and IsGoodLaw proofs.
func ApplyPrecedent(
	precedent Decision,
	toCourt Court,
	bindingProof *BindsOnProof,
	goodLawProof *IsGoodLawProof,
) (*PrecedentApplication, error) {
	// Verify proofs
	if err := bindingProof.Verify(); err != nil {
		return nil, fmt.Errorf("binding proof invalid: %w", err)
	}
	if err := goodLawProof.Verify(); err != nil {
		return nil, fmt.Errorf("good law proof invalid: %w", err)
	}

	return &PrecedentApplication{
		Precedent:       precedent.Citation,
		AppliedTo:       toCourt.Name,
		RatioApplied:    precedent.Majority.Ratio,
		BindingStrength: BindingStrengthStrong,
		ApplicationDate: Today(),
	}, nil
}

// AuthorityExercise represents an exercise of authority.
type AuthorityExercise struct {
	Holder        AuthorityHolder
	Action        AuthorityAction
	Scope         AuthorityScope
	ExercisedAt   Date
	ProofVerified bool
}

// ExerciseAuthority exercises authority for an action.
// Requires a valid HasAuthority proof.
func ExerciseAuthority(
	holder AuthorityHolder,
	action AuthorityAction,
	scope AuthorityScope,
	authorityProof *HasAuthorityProof,
) (*AuthorityExercise, error) {
	if err := authorityProof.Verify(); err != nil {
		return nil, fmt.Errorf("authority proof invalid: %w", err)
	}

	return &AuthorityExercise{
		Holder:        holder,
		Action:        action,
		Scope:         scope,
		ExercisedAt:   Today(),
		ProofVerified: true,
	}, nil
}

// BindingForceKind represents the type of binding force.
type BindingForceKind int

const (
	BindingForceMandatory BindingForceKind = iota
	BindingForcePersuasive
	BindingForceForeign
	BindingForceNone
)

// BindingForce represents the binding force of a citation.
type BindingForce struct {
	Kind     BindingForceKind
	Strength *BindingStrength
}

// BindingCitation represents a citation as binding authority.
type BindingCitation struct {
	Citation    Citation
	BindingType BindingForce
	CitedIn     string
	Verified    bool
}

// CiteAsBinding creates a binding citation.
// Requires valid BindsOn and IsGoodLaw proofs.
func CiteAsBinding(
	citation Citation,
	inCourt Court,
	fromCourt Court,
	bindingProof *BindsOnProof,
	goodLawProof *IsGoodLawProof,
) (*BindingCitation, error) {
	if err := bindingProof.Verify(); err != nil {
		return nil, fmt.Errorf("binding proof invalid: %w", err)
	}
	if err := goodLawProof.Verify(); err != nil {
		return nil, fmt.Errorf("good law proof invalid: %w", err)
	}

	strength := BindingStrengthStrong
	return &BindingCitation{
		Citation: citation,
		BindingType: BindingForce{
			Kind:     BindingForceMandatory,
			Strength: &strength,
		},
		CitedIn:  inCourt.Name,
		Verified: true,
	}, nil
}
