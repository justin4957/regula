package types

import "strings"

// Country represents an ISO 3166-1 country code.
type Country string

// Common country codes.
const (
	CountryUS Country = "US"
	CountryUK Country = "UK"
	CountryFR Country = "FR"
	CountryDE Country = "DE"
	CountryAU Country = "AU"
	CountryCA Country = "CA"
	CountryJP Country = "JP"
	CountryBR Country = "BR"
	CountryIN Country = "IN"
	CountryZA Country = "ZA"
)

// StateCode represents a state/province within a country.
type StateCode struct {
	Code    string
	Country Country
}

// CityCode represents a city/municipality.
type CityCode struct {
	Name  string
	State StateCode
}

// JurisdictionKind represents the type of jurisdiction.
type JurisdictionKind int

const (
	JurisdictionFederal JurisdictionKind = iota
	JurisdictionState
	JurisdictionMunicipal
	JurisdictionInternational
	JurisdictionSupranational
	JurisdictionTribal
	JurisdictionMaritime
	JurisdictionMilitary
)

// Jurisdiction represents a domain where law applies.
type Jurisdiction struct {
	Kind JurisdictionKind

	// For Federal, State, Municipal, Military
	Country Country

	// For State, Municipal
	State *StateCode

	// For Municipal
	City *CityCode

	// For International
	Treaty *TreatyID

	// For Supranational
	Body *SupranationalBody

	// For Tribal
	TribalNation *TribalNation

	// For Maritime
	MaritimeZone *MaritimeZone

	// For Military
	Branch *MilitaryBranch
}

// Federal creates a federal jurisdiction.
func Federal(country Country) Jurisdiction {
	return Jurisdiction{Kind: JurisdictionFederal, Country: country}
}

// State creates a state jurisdiction.
func State(country Country, state StateCode) Jurisdiction {
	return Jurisdiction{Kind: JurisdictionState, Country: country, State: &state}
}

// Municipal creates a municipal jurisdiction.
func Municipal(country Country, state StateCode, city CityCode) Jurisdiction {
	return Jurisdiction{
		Kind:    JurisdictionMunicipal,
		Country: country,
		State:   &state,
		City:    &city,
	}
}

// International creates an international treaty jurisdiction.
func International(treaty TreatyID) Jurisdiction {
	return Jurisdiction{Kind: JurisdictionInternational, Treaty: &treaty}
}

// Supranational creates a supranational body jurisdiction.
func Supranational(body SupranationalBody) Jurisdiction {
	return Jurisdiction{Kind: JurisdictionSupranational, Body: &body}
}

// TreatyID identifies an international treaty.
type TreatyID struct {
	Name    string
	Signed  Date
	Parties []Country
}

// SupranationalBody represents a supranational organization.
type SupranationalBody int

const (
	SupranationalEU SupranationalBody = iota
	SupranationalAU // African Union
	SupranationalASEAN
	SupranationalMercosur
	SupranationalUN
	SupranationalWTO
	SupranationalICC
)

func (s SupranationalBody) String() string {
	names := []string{"EU", "AU", "ASEAN", "Mercosur", "UN", "WTO", "ICC"}
	if int(s) < len(names) {
		return names[s]
	}
	return "Unknown"
}

// TribalNation represents a tribal nation.
type TribalNation struct {
	Name         string
	RecognizedBy Country
}

// MaritimeZoneKind represents the type of maritime zone.
type MaritimeZoneKind int

const (
	MaritimeZoneTerritorialSea MaritimeZoneKind = iota
	MaritimeZoneContiguousZone
	MaritimeZoneExclusiveEconomicZone
	MaritimeZoneHighSeas
	MaritimeZoneInternationalSeabed
)

// MaritimeZone represents a maritime zone.
type MaritimeZone struct {
	Kind    MaritimeZoneKind
	Country *Country // nil for HighSeas and InternationalSeabed
}

// MilitaryBranch represents a military branch.
type MilitaryBranch int

const (
	MilitaryArmy MilitaryBranch = iota
	MilitaryNavy
	MilitaryAirForce
	MilitaryMarines
	MilitaryCoastGuard
	MilitarySpaceForce
)

// LegalSystemKind represents the fundamental legal tradition.
type LegalSystemKind int

const (
	LegalSystemCommonLaw LegalSystemKind = iota
	LegalSystemCivilLaw
	LegalSystemMixed
	LegalSystemReligious
	LegalSystemCustomary
	LegalSystemSocialist
)

// ReligiousTradition represents a religious legal tradition.
type ReligiousTradition int

const (
	ReligiousIslamicSharia ReligiousTradition = iota
	ReligiousJewishHalakha
	ReligiousHinduLaw
	ReligiousCanonLaw
	ReligiousBuddhistVinaya
)

// LegalSystem represents the legal system of a jurisdiction.
type LegalSystem struct {
	Kind       LegalSystemKind
	Primary    *LegalSystem       // for Mixed
	Secondary  *LegalSystem       // for Mixed
	Tradition  *ReligiousTradition // for Religious
	Community  *string            // for Customary
}

// CommonLaw returns a common law legal system.
func CommonLaw() LegalSystem {
	return LegalSystem{Kind: LegalSystemCommonLaw}
}

// CivilLaw returns a civil law legal system.
func CivilLaw() LegalSystem {
	return LegalSystem{Kind: LegalSystemCivilLaw}
}

// MixedSystem returns a mixed legal system.
func MixedSystem(primary, secondary LegalSystem) LegalSystem {
	return LegalSystem{Kind: LegalSystemMixed, Primary: &primary, Secondary: &secondary}
}

// LegalSystemOf returns the legal system for a jurisdiction.
func LegalSystemOf(j Jurisdiction) LegalSystem {
	switch j.Kind {
	case JurisdictionFederal:
		switch j.Country {
		case CountryUS, CountryUK, CountryAU, CountryCA, CountryIN:
			return CommonLaw()
		case CountryFR, CountryDE, CountryJP, CountryBR:
			return CivilLaw()
		case CountryZA:
			return MixedSystem(CommonLaw(), CivilLaw())
		}
	case JurisdictionState:
		if j.State != nil && j.State.Code == "QC" && j.Country == CountryCA {
			return CivilLaw()
		}
		return LegalSystemOf(Federal(j.Country))
	case JurisdictionSupranational:
		if j.Body != nil && *j.Body == SupranationalEU {
			return CivilLaw()
		}
	}
	return CommonLaw() // Default fallback
}

// CourtLevel represents the level of a court in the judicial hierarchy.
type CourtLevel int

const (
	CourtLevelTrial CourtLevel = iota
	CourtLevelSpecialized
	CourtLevelIntermediateAppellate
	CourtLevelConstitutional
	CourtLevelHighestAppellate
)

func (c CourtLevel) String() string {
	names := []string{"Trial", "Specialized", "Intermediate Appellate", "Constitutional", "Highest Appellate"}
	if int(c) < len(names) {
		return names[c]
	}
	return "Unknown"
}

// GeographicScopeKind represents the type of geographic scope.
type GeographicScopeKind int

const (
	GeographicScopeNational GeographicScopeKind = iota
	GeographicScopeRegional
	GeographicScopeDistrict
	GeographicScopeLocal
)

// GeographicScope represents the geographic scope of a court's jurisdiction.
type GeographicScope struct {
	Kind     GeographicScopeKind
	Regions  []string // for Regional
	District string   // for District
	Locality string   // for Local
}

// NationalScope returns a national geographic scope.
func NationalScope() GeographicScope {
	return GeographicScope{Kind: GeographicScopeNational}
}

// RegionalScope returns a regional geographic scope.
func RegionalScope(regions ...string) GeographicScope {
	return GeographicScope{Kind: GeographicScopeRegional, Regions: regions}
}

// DistrictScope returns a district geographic scope.
func DistrictScope(district string) GeographicScope {
	return GeographicScope{Kind: GeographicScopeDistrict, District: district}
}

// LocalScope returns a local geographic scope.
func LocalScope(locality string) GeographicScope {
	return GeographicScope{Kind: GeographicScopeLocal, Locality: locality}
}

// Contains checks if this geographic scope contains another.
func (g GeographicScope) Contains(inner GeographicScope) bool {
	switch g.Kind {
	case GeographicScopeNational:
		return true
	case GeographicScopeRegional:
		switch inner.Kind {
		case GeographicScopeNational:
			return false
		case GeographicScopeRegional:
			// All inner regions must be in outer regions
			for _, r := range inner.Regions {
				found := false
				for _, outer := range g.Regions {
					if r == outer {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
			return true
		case GeographicScopeDistrict:
			for _, r := range g.Regions {
				if strings.HasPrefix(inner.District, r) {
					return true
				}
			}
			return false
		case GeographicScopeLocal:
			for _, r := range g.Regions {
				if strings.HasPrefix(inner.Locality, r) {
					return true
				}
			}
			return false
		}
	case GeographicScopeDistrict:
		switch inner.Kind {
		case GeographicScopeLocal:
			return strings.HasPrefix(inner.Locality, g.District)
		case GeographicScopeDistrict:
			return inner.District == g.District
		}
		return false
	case GeographicScopeLocal:
		return inner.Kind == GeographicScopeLocal && inner.Locality == g.Locality
	}
	return false
}

// SubjectMatter represents the subject matter jurisdiction.
type SubjectMatter int

const (
	SubjectMatterGeneral SubjectMatter = iota
	SubjectMatterCriminal
	SubjectMatterCivil
	SubjectMatterAdministrative
	SubjectMatterTax
	SubjectMatterPatent
	SubjectMatterBankruptcy
	SubjectMatterFamily
	SubjectMatterJuvenile
	SubjectMatterImmigration
	SubjectMatterMilitary
	SubjectMatterMaritime
)

// Court represents a court within a jurisdiction.
type Court struct {
	Name          string
	Jurisdiction  Jurisdiction
	Level         CourtLevel
	Geographic    GeographicScope
	SubjectMatter SubjectMatter
	Established   Date
	JudgesCount   int
}

// Binds checks if this court's decisions bind another court.
func (c Court) Binds(lower Court) bool {
	// Check court level
	if c.Level <= lower.Level {
		return false
	}
	// Check geographic scope
	if !c.Geographic.Contains(lower.Geographic) {
		return false
	}
	return true
}

// JurisdictionInfo contains complete information about a jurisdiction.
type JurisdictionInfo struct {
	Jurisdiction      Jurisdiction
	System            LegalSystem
	Constitutional    *ConstitutionID
	Hierarchy         []Jurisdiction // Appeals path upward
	Subsidiaries      []Jurisdiction // Lower jurisdictions
	TreatyParties     []TreatyID
	OfficialLanguages []Language
	FoundingDate      *Date
}

// ConstitutionID identifies a constitution.
type ConstitutionID struct {
	Jurisdiction Jurisdiction
	Name         string
	Enacted      Date
	LastAmended  *Date
}

// Language represents a language.
type Language int

const (
	LanguageEnglish Language = iota
	LanguageFrench
	LanguageGerman
	LanguageSpanish
	LanguagePortuguese
	LanguageJapanese
	LanguageChinese
	LanguageArabic
	LanguageHindi
	LanguageOther
)
