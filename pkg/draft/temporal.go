package draft

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/coolbeans/regula/pkg/library"
	"github.com/coolbeans/regula/pkg/store"
)

// TemporalIssueType classifies the kind of temporal consistency issue detected
// in proposed legislation.
type TemporalIssueType int

const (
	// TemporalGap indicates a repeal takes effect before its replacement,
	// leaving a period where no rule is in force.
	TemporalGap TemporalIssueType = iota
	// TemporalContradiction indicates two contradictory rules are
	// simultaneously in force during the same period.
	TemporalContradiction
	// TemporalRetroactive indicates the draft applies retroactively to
	// events that occurred before the date of enactment.
	TemporalRetroactive
	// TemporalSunset is an informational finding indicating a provision
	// has an expiration date or sunset clause.
	TemporalSunset
)

// temporalIssueTypeLabels maps temporal issue types to human-readable strings.
var temporalIssueTypeLabels = [...]string{
	TemporalGap:           "temporal_gap",
	TemporalContradiction: "temporal_contradiction",
	TemporalRetroactive:   "temporal_retroactive",
	TemporalSunset:        "temporal_sunset",
}

// String returns a human-readable label for the temporal issue type.
func (t TemporalIssueType) String() string {
	if int(t) < len(temporalIssueTypeLabels) {
		return temporalIssueTypeLabels[t]
	}
	return "unknown"
}

// TemporalFinding represents a detected temporal consistency issue or
// informational finding about temporal aspects of proposed legislation.
type TemporalFinding struct {
	Type           TemporalIssueType `json:"type"`
	Severity       ConflictSeverity  `json:"severity"`
	Description    string            `json:"description"`
	EffectiveDate  *time.Time        `json:"effective_date,omitempty"`
	AffectedPeriod string            `json:"affected_period,omitempty"`
	Provisions     []string          `json:"provisions,omitempty"`
}

// EffectiveDateInfo contains parsed effective date information from bill text.
type EffectiveDateInfo struct {
	Date           *time.Time `json:"date,omitempty"`
	IsDateOfEnactment bool    `json:"is_date_of_enactment"`
	DaysAfterEnactment int    `json:"days_after_enactment,omitempty"`
	RawText        string     `json:"raw_text"`
}

// AnalyzeTemporalConsistency examines a computed diff for temporal consistency
// issues including gaps between repeal and replacement, contradictory provisions
// in force simultaneously, and retroactive application.
func AnalyzeTemporalConsistency(diff *DraftDiff, libraryPath string) ([]TemporalFinding, error) {
	if diff == nil {
		return nil, fmt.Errorf("diff is nil")
	}

	var findings []TemporalFinding

	// Extract effective date from bill text
	effectiveDateInfo := ParseEffectiveDate(diff.Bill.RawText)

	// Detect temporal gaps between repeals and additions
	gapFindings := DetectTemporalGaps(diff.Removed, diff.Added, effectiveDateInfo)
	findings = append(findings, gapFindings...)

	// Detect retroactive application language
	retroactiveFindings := DetectRetroactiveApplication(diff.Bill.RawText)
	findings = append(findings, retroactiveFindings...)

	// Detect temporal contradictions via triple store temporal metadata
	if libraryPath != "" {
		lib, err := library.Open(libraryPath)
		if err == nil {
			contradictionFindings := DetectTemporalContradictions(diff.Modified, lib, effectiveDateInfo)
			findings = append(findings, contradictionFindings...)
		}
	}

	// Detect sunset clauses (informational)
	sunsetFindings := DetectSunsetClauses(diff.Bill.RawText)
	findings = append(findings, sunsetFindings...)

	return findings, nil
}

// ParseEffectiveDate extracts effective or enactment date information from
// bill text using common Congressional formatting patterns.
func ParseEffectiveDate(billText string) *EffectiveDateInfo {
	normalizedText := strings.ToLower(strings.Join(strings.Fields(billText), " "))

	// Pattern: "shall take effect on the date of enactment"
	if dateOfEnactmentPattern.MatchString(normalizedText) {
		return &EffectiveDateInfo{
			IsDateOfEnactment: true,
			RawText:           extractMatchedText(normalizedText, dateOfEnactmentPattern),
		}
	}

	// Pattern: "effective N days after the date of enactment"
	if matches := daysAfterEnactmentPattern.FindStringSubmatch(normalizedText); len(matches) >= 2 {
		days, err := strconv.Atoi(matches[1])
		if err == nil {
			return &EffectiveDateInfo{
				DaysAfterEnactment: days,
				RawText:            matches[0],
			}
		}
	}

	// Pattern: "shall take effect on [specific date]"
	if matches := specificDatePattern.FindStringSubmatch(normalizedText); len(matches) >= 4 {
		date := parseSpecificDate(matches[1], matches[2], matches[3])
		if date != nil {
			return &EffectiveDateInfo{
				Date:    date,
				RawText: matches[0],
			}
		}
	}

	// Pattern: "shall apply to fiscal years beginning after [date]"
	if matches := fiscalYearPattern.FindStringSubmatch(normalizedText); len(matches) >= 4 {
		date := parseSpecificDate(matches[1], matches[2], matches[3])
		if date != nil {
			return &EffectiveDateInfo{
				Date:    date,
				RawText: matches[0],
			}
		}
	}

	// Pattern: "shall not take effect until [date]"
	if matches := notUntilPattern.FindStringSubmatch(normalizedText); len(matches) >= 4 {
		date := parseSpecificDate(matches[1], matches[2], matches[3])
		if date != nil {
			return &EffectiveDateInfo{
				Date:    date,
				RawText: matches[0],
			}
		}
	}

	return nil
}

// DetectTemporalGaps checks for periods where a provision is repealed but its
// replacement has not yet taken effect, creating a regulatory gap.
func DetectTemporalGaps(repeals []DiffEntry, additions []DiffEntry, effectiveDateInfo *EffectiveDateInfo) []TemporalFinding {
	var findings []TemporalFinding

	if len(repeals) == 0 {
		return findings
	}

	// Build a map of sections being added (potential replacements)
	addedSections := make(map[string]DiffEntry)
	for _, addition := range additions {
		sectionKey := extractSectionFromURI(addition.TargetURI)
		if sectionKey != "" {
			addedSections[sectionKey] = addition
		}
	}

	for _, repeal := range repeals {
		repealSection := extractSectionFromURI(repeal.TargetURI)
		if repealSection == "" {
			continue
		}

		// Check if there's a corresponding addition that could be a replacement
		// Look for additions in nearby sections (e.g., 6502 -> 6502A, 6502 -> 6510)
		hasReplacement := false
		var replacementEntry DiffEntry
		for addedSection, entry := range addedSections {
			if isRelatedSection(repealSection, addedSection) {
				hasReplacement = true
				replacementEntry = entry
				break
			}
		}

		if hasReplacement {
			// Check effective dates for gap
			// If we can't determine dates, flag as potential gap
			if effectiveDateInfo == nil {
				findings = append(findings, TemporalFinding{
					Type:        TemporalGap,
					Severity:    ConflictWarning,
					Description: fmt.Sprintf("potential temporal gap: section %s is repealed and section %s is added, but effective dates could not be determined", repealSection, extractSectionFromURI(replacementEntry.TargetURI)),
					Provisions:  []string{repeal.TargetURI, replacementEntry.TargetURI},
				})
			}
		} else {
			// Repeal without any apparent replacement in same bill
			findings = append(findings, TemporalFinding{
				Type:        TemporalGap,
				Severity:    ConflictWarning,
				Description: fmt.Sprintf("section %s is repealed with no apparent replacement in this bill", repealSection),
				Provisions:  []string{repeal.TargetURI},
			})
		}
	}

	return findings
}

// DetectTemporalContradictions checks for provisions that would be simultaneously
// in force with contradictory requirements by examining temporal metadata in the
// knowledge graph.
func DetectTemporalContradictions(modifications []DiffEntry, lib *library.Library, effectiveDateInfo *EffectiveDateInfo) []TemporalFinding {
	var findings []TemporalFinding

	if len(modifications) == 0 || lib == nil {
		return findings
	}

	tripleStoreCache := make(map[string]*store.TripleStore)

	for _, mod := range modifications {
		tripleStore, err := loadOrCacheTripleStore(lib, mod.TargetDocumentID, tripleStoreCache)
		if err != nil {
			continue
		}

		// Check for temporal metadata on the target provision
		temporalKindTriples := tripleStore.Find(mod.TargetURI, store.PropTemporalKind, "")
		for _, triple := range temporalKindTriples {
			temporalKind := triple.Object

			// If existing provision has "in_force_on" temporal kind and we're modifying it,
			// check for potential contradiction during transition period
			if strings.Contains(strings.ToLower(temporalKind), "in_force") ||
				strings.Contains(strings.ToLower(temporalKind), "as_amended") {

				// Look for related provisions with conflicting temporal status
				relatedProvisions := findRelatedProvisionsWithTemporalConflict(mod.TargetURI, tripleStore)
				if len(relatedProvisions) > 0 {
					findings = append(findings, TemporalFinding{
						Type:     TemporalContradiction,
						Severity: ConflictError,
						Description: fmt.Sprintf(
							"potential temporal contradiction: modifying %s which has temporal status '%s' while related provisions may be in conflicting temporal states",
							extractURILabel(mod.TargetURI),
							temporalKind,
						),
						Provisions: append([]string{mod.TargetURI}, relatedProvisions...),
					})
				}
			}
		}
	}

	return findings
}

// DetectRetroactiveApplication scans bill text for language indicating the
// legislation applies retroactively to events before the date of enactment.
func DetectRetroactiveApplication(billText string) []TemporalFinding {
	var findings []TemporalFinding
	normalizedText := strings.ToLower(strings.Join(strings.Fields(billText), " "))

	for _, pattern := range retroactivePatterns {
		if matches := pattern.FindStringSubmatch(normalizedText); len(matches) > 0 {
			findings = append(findings, TemporalFinding{
				Type:        TemporalRetroactive,
				Severity:    ConflictWarning,
				Description: fmt.Sprintf("retroactive application detected: '%s'", truncateText(matches[0], 100)),
			})
		}
	}

	return findings
}

// DetectSunsetClauses scans bill text for sunset or expiration clauses,
// flagging them as informational findings.
func DetectSunsetClauses(billText string) []TemporalFinding {
	var findings []TemporalFinding
	normalizedText := strings.ToLower(strings.Join(strings.Fields(billText), " "))

	for _, pattern := range sunsetPatterns {
		if matches := pattern.FindStringSubmatch(normalizedText); len(matches) > 0 {
			findings = append(findings, TemporalFinding{
				Type:        TemporalSunset,
				Severity:    ConflictInfo,
				Description: fmt.Sprintf("sunset clause detected: '%s'", truncateText(matches[0], 100)),
			})
		}
	}

	return findings
}

// Effective date parsing patterns
var (
	dateOfEnactmentPattern = regexp.MustCompile(`(?i)(?:shall|will)\s+(?:take\s+effect|become\s+effective)\s+(?:on\s+)?(?:the\s+)?date\s+of\s+(?:the\s+)?enactment`)

	daysAfterEnactmentPattern = regexp.MustCompile(`(?i)(?:effective|take\s+effect)\s+(\d+)\s+days?\s+after\s+(?:the\s+)?date\s+of\s+(?:the\s+)?enactment`)

	specificDatePattern = regexp.MustCompile(`(?i)(?:shall|will)\s+(?:take\s+effect|become\s+effective)\s+(?:on\s+)?(january|february|march|april|may|june|july|august|september|october|november|december)\s+(\d{1,2}),?\s+(\d{4})`)

	fiscalYearPattern = regexp.MustCompile(`(?i)shall\s+apply\s+to\s+fiscal\s+years?\s+beginning\s+after\s+(january|february|march|april|may|june|july|august|september|october|november|december)\s+(\d{1,2}),?\s+(\d{4})`)

	notUntilPattern = regexp.MustCompile(`(?i)shall\s+not\s+take\s+effect\s+until\s+(january|february|march|april|may|june|july|august|september|october|november|december)\s+(\d{1,2}),?\s+(\d{4})`)
)

// Retroactive application patterns
var retroactivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)shall\s+apply\s+to\s+(?:any\s+)?(?:action|conduct|violation|proceeding)s?\s+(?:taken|occurring|commenced)\s+(?:before|prior\s+to)\s+(?:the\s+)?date\s+of\s+(?:the\s+)?enactment`),
	regexp.MustCompile(`(?i)retroactive(?:ly)?\s+(?:to|effective)`),
	regexp.MustCompile(`(?i)applies?\s+(?:retroactively|to\s+past\s+(?:conduct|actions|events))`),
	regexp.MustCompile(`(?i)(?:effective|apply)\s+(?:as\s+of|beginning)\s+(?:a\s+date\s+)?(?:before|prior\s+to)\s+(?:the\s+)?(?:date\s+of\s+)?enactment`),
}

// Sunset clause patterns
var sunsetPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(?:this\s+)?(?:section|act|provision)\s+(?:shall\s+)?(?:expire|terminate)`),
	regexp.MustCompile(`(?i)cease\s+to\s+(?:be\s+)?(?:effective|in\s+effect)`),
	regexp.MustCompile(`(?i)sunset\s+(?:date|provision|clause)`),
	regexp.MustCompile(`(?i)shall\s+(?:not\s+)?(?:remain\s+)?in\s+effect\s+(?:only\s+)?(?:until|through|for\s+a\s+period\s+of)`),
	regexp.MustCompile(`(?i)(?:is\s+)?repealed\s+(?:effective|on)\s+(january|february|march|april|may|june|july|august|september|october|november|december)\s+\d{1,2},?\s+\d{4}`),
}

// monthNameToNumber maps month names to month numbers.
var monthNameToNumber = map[string]time.Month{
	"january":   time.January,
	"february":  time.February,
	"march":     time.March,
	"april":     time.April,
	"may":       time.May,
	"june":      time.June,
	"july":      time.July,
	"august":    time.August,
	"september": time.September,
	"october":   time.October,
	"november":  time.November,
	"december":  time.December,
}

// parseSpecificDate converts month name, day, and year strings to a time.Time.
func parseSpecificDate(monthName, day, year string) *time.Time {
	month, ok := monthNameToNumber[strings.ToLower(monthName)]
	if !ok {
		return nil
	}

	dayNum, err := strconv.Atoi(day)
	if err != nil || dayNum < 1 || dayNum > 31 {
		return nil
	}

	yearNum, err := strconv.Atoi(year)
	if err != nil || yearNum < 1900 || yearNum > 2100 {
		return nil
	}

	date := time.Date(yearNum, month, dayNum, 0, 0, 0, 0, time.UTC)
	return &date
}

// extractMatchedText returns the first match of a pattern in text.
func extractMatchedText(text string, pattern *regexp.Regexp) string {
	match := pattern.FindString(text)
	if match != "" {
		return match
	}
	return ""
}

// extractSectionFromURI extracts the section number from a knowledge graph URI.
// For example, ".../US-USC-TITLE-15:Art6502" returns "6502".
func extractSectionFromURI(uri string) string {
	// Look for :Art followed by numbers
	artPattern := regexp.MustCompile(`:Art(\d+)`)
	matches := artPattern.FindStringSubmatch(uri)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// isRelatedSection checks if two section numbers are related (potential
// replacement relationship). For example, 6502 and 6502A, or 6502 and 6510.
func isRelatedSection(original, candidate string) bool {
	if original == "" || candidate == "" {
		return false
	}

	// Exact match
	if original == candidate {
		return true
	}

	// Candidate is original with suffix (e.g., 6502 -> 6502A)
	if strings.HasPrefix(candidate, original) {
		return true
	}

	// Within same range (first 3 digits match for 4-digit sections)
	if len(original) >= 3 && len(candidate) >= 3 {
		return original[:len(original)-1] == candidate[:len(candidate)-1]
	}

	return false
}

// findRelatedProvisionsWithTemporalConflict finds provisions that reference the
// target and have conflicting temporal metadata.
func findRelatedProvisionsWithTemporalConflict(targetURI string, tripleStore *store.TripleStore) []string {
	var conflicting []string

	// Find provisions that reference this target
	incomingRefs := tripleStore.Find("", store.PropReferences, targetURI)
	for _, triple := range incomingRefs {
		relatedURI := triple.Subject

		// Check if related provision has temporal metadata indicating it's in force
		temporalTriples := tripleStore.Find(relatedURI, store.PropTemporalKind, "")
		for _, tempTriple := range temporalTriples {
			temporalKind := strings.ToLower(tempTriple.Object)
			if strings.Contains(temporalKind, "in_force") || strings.Contains(temporalKind, "current") {
				conflicting = append(conflicting, relatedURI)
				break
			}
		}
	}

	return conflicting
}

// truncateText truncates text to maxLen characters, adding "..." if truncated.
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}
