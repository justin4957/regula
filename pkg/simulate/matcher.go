package simulate

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/coolbeans/regula/pkg/extract"
	"github.com/coolbeans/regula/pkg/store"
)

// RelevanceScore indicates how directly a provision applies to a scenario.
type RelevanceScore string

const (
	// RelevanceDirect indicates the provision directly applies to the scenario.
	RelevanceDirect RelevanceScore = "DIRECT"
	// RelevanceTriggered indicates the provision is triggered by scenario actions.
	RelevanceTriggered RelevanceScore = "TRIGGERED"
	// RelevanceRelated indicates the provision is related but not directly applicable.
	RelevanceRelated RelevanceScore = "RELATED"
)

// MatchedProvision represents a provision matched to a scenario.
type MatchedProvision struct {
	URI           string                       `json:"uri"`
	ArticleNum    int                          `json:"article_num"`
	Title         string                       `json:"title"`
	Relevance     RelevanceScore               `json:"relevance"`
	Score         float64                      `json:"score"`
	MatchReasons  []string                     `json:"match_reasons"`
	Rights        []*extract.SemanticAnnotation `json:"rights,omitempty"`
	Obligations   []*extract.SemanticAnnotation `json:"obligations,omitempty"`
	Keywords      []string                     `json:"matched_keywords,omitempty"`
	ReferencedBy  []int                        `json:"referenced_by,omitempty"`
}

// MatchResult contains the results of provision matching.
type MatchResult struct {
	Scenario       *Scenario            `json:"scenario"`
	DirectMatches  []*MatchedProvision  `json:"direct_matches"`
	TriggeredMatches []*MatchedProvision `json:"triggered_matches"`
	RelatedMatches []*MatchedProvision  `json:"related_matches"`
	AllMatches     []*MatchedProvision  `json:"all_matches"`
	Summary        *MatchSummary        `json:"summary"`
}

// MatchSummary provides summary statistics for the match.
type MatchSummary struct {
	TotalMatches    int                    `json:"total_matches"`
	DirectCount     int                    `json:"direct_count"`
	TriggeredCount  int                    `json:"triggered_count"`
	RelatedCount    int                    `json:"related_count"`
	RightsInvolved  []extract.RightType    `json:"rights_involved"`
	ObligationsInvolved []extract.ObligationType `json:"obligations_involved"`
	KeyArticles     []int                  `json:"key_articles"`
}

// ProvisionMatcher matches scenarios to applicable provisions.
type ProvisionMatcher struct {
	store          *store.TripleStore
	baseURI        string
	semanticLookup *extract.SemanticLookup
	doc            *extract.Document

	// Action to right type mapping
	actionRightMap map[ActionType][]extract.RightType

	// Action to obligation type mapping
	actionObligMap map[ActionType][]extract.ObligationType

	// Keyword to article mapping (built from graph)
	keywordArticles map[string][]int
}

// NewProvisionMatcher creates a new provision matcher.
func NewProvisionMatcher(ts *store.TripleStore, baseURI string, annotations []*extract.SemanticAnnotation, doc *extract.Document) *ProvisionMatcher {
	matcher := &ProvisionMatcher{
		store:          ts,
		baseURI:        baseURI,
		semanticLookup: extract.NewSemanticLookup(annotations),
		doc:            doc,
		keywordArticles: make(map[string][]int),
	}

	matcher.initActionMappings()
	matcher.buildKeywordIndex()

	return matcher
}

// initActionMappings initializes mappings between actions and semantic types.
func (m *ProvisionMatcher) initActionMappings() {
	// Map actions to relevant right types
	m.actionRightMap = map[ActionType][]extract.RightType{
		ActionWithdrawConsent:    {extract.RightWithdrawConsent},
		ActionRequestAccess:      {extract.RightAccess, extract.RightInformation},
		ActionRequestErasure:     {extract.RightErasure},
		ActionRequestRectify:     {extract.RightRectification},
		ActionRequestPortability: {extract.RightPortability},
		ActionObjectProcessing:   {extract.RightObject},
		ActionFileComplaint:      {extract.RightLodgeComplaint, extract.RightEffectiveRemedy},
	}

	// Map actions to relevant obligation types
	m.actionObligMap = map[ActionType][]extract.ObligationType{
		ActionWithdrawConsent:    {extract.ObligationConsent, extract.ObligationRespond},
		ActionRequestAccess:      {extract.ObligationProvideInformation, extract.ObligationRespond, extract.ObligationTransparency},
		ActionRequestErasure:     {extract.ObligationRespond, extract.ObligationNotifySubject},
		ActionRequestRectify:     {extract.ObligationRespond},
		ActionRequestPortability: {extract.ObligationRespond},
		ActionObjectProcessing:   {extract.ObligationRespond},
		ActionProcessData:        {extract.ObligationLawfulProcessing, extract.ObligationSecure, extract.ObligationRecord},
		ActionTransferData:       {extract.ObligationLawfulProcessing},
		ActionBreach:             {extract.ObligationNotifyBreach, extract.ObligationNotifySubject, extract.ObligationSecure},
		ActionCollectData:        {extract.ObligationConsent, extract.ObligationProvideInformation, extract.ObligationTransparency},
		ActionProvideConsent:     {extract.ObligationConsent},
	}
}

// buildKeywordIndex builds an index of keywords to article numbers.
func (m *ProvisionMatcher) buildKeywordIndex() {
	if m.doc == nil {
		return
	}

	for _, article := range m.doc.AllArticles() {
		// Extract keywords from title
		titleWords := extractWordsFromText(article.Title)
		for _, word := range titleWords {
			m.keywordArticles[word] = appendUnique(m.keywordArticles[word], article.Number)
		}

		// Extract keywords from text
		textWords := extractWordsFromText(article.Text)
		for _, word := range textWords {
			m.keywordArticles[word] = appendUnique(m.keywordArticles[word], article.Number)
		}
	}
}

// Match matches a scenario to applicable provisions.
func (m *ProvisionMatcher) Match(scenario *Scenario) *MatchResult {
	result := &MatchResult{
		Scenario:        scenario,
		DirectMatches:   make([]*MatchedProvision, 0),
		TriggeredMatches: make([]*MatchedProvision, 0),
		RelatedMatches:  make([]*MatchedProvision, 0),
		AllMatches:      make([]*MatchedProvision, 0),
		Summary: &MatchSummary{
			RightsInvolved:      make([]extract.RightType, 0),
			ObligationsInvolved: make([]extract.ObligationType, 0),
			KeyArticles:         make([]int, 0),
		},
	}

	// Track matched articles to avoid duplicates
	matchedArticles := make(map[int]*MatchedProvision)

	// Step 1: Find direct matches based on action types
	m.findDirectMatches(scenario, matchedArticles)

	// Step 2: Find triggered matches (provisions referenced by direct matches)
	m.findTriggeredMatches(matchedArticles)

	// Step 3: Find related matches based on keywords
	m.findRelatedMatches(scenario, matchedArticles)

	// Categorize and collect results
	for _, match := range matchedArticles {
		result.AllMatches = append(result.AllMatches, match)

		switch match.Relevance {
		case RelevanceDirect:
			result.DirectMatches = append(result.DirectMatches, match)
		case RelevanceTriggered:
			result.TriggeredMatches = append(result.TriggeredMatches, match)
		case RelevanceRelated:
			result.RelatedMatches = append(result.RelatedMatches, match)
		}
	}

	// Sort all by score descending
	sort.Slice(result.AllMatches, func(i, j int) bool {
		return result.AllMatches[i].Score > result.AllMatches[j].Score
	})
	sort.Slice(result.DirectMatches, func(i, j int) bool {
		return result.DirectMatches[i].Score > result.DirectMatches[j].Score
	})
	sort.Slice(result.TriggeredMatches, func(i, j int) bool {
		return result.TriggeredMatches[i].Score > result.TriggeredMatches[j].Score
	})
	sort.Slice(result.RelatedMatches, func(i, j int) bool {
		return result.RelatedMatches[i].Score > result.RelatedMatches[j].Score
	})

	// Calculate summary
	m.calculateSummary(result)

	return result
}

// findDirectMatches finds provisions that directly apply to scenario actions.
func (m *ProvisionMatcher) findDirectMatches(scenario *Scenario, matches map[int]*MatchedProvision) {
	for _, action := range scenario.Actions {
		// Find rights that match this action
		if rightTypes, ok := m.actionRightMap[action.Type]; ok {
			for _, rightType := range rightTypes {
				annotations := m.semanticLookup.GetByRightType(rightType)
				for _, ann := range annotations {
					match := m.getOrCreateMatch(matches, ann.ArticleNum)
					match.Relevance = RelevanceDirect
					match.Score = max(match.Score, 1.0*ann.Confidence)
					match.MatchReasons = appendUnique(match.MatchReasons,
						fmt.Sprintf("Grants %s (action: %s)", rightType, action.Type))
					match.Rights = append(match.Rights, ann)
				}
			}
		}

		// Find obligations that match this action
		if obligTypes, ok := m.actionObligMap[action.Type]; ok {
			for _, obligType := range obligTypes {
				annotations := m.semanticLookup.GetByObligationType(obligType)
				for _, ann := range annotations {
					match := m.getOrCreateMatch(matches, ann.ArticleNum)
					if match.Relevance != RelevanceDirect {
						match.Relevance = RelevanceDirect
					}
					match.Score = max(match.Score, 0.9*ann.Confidence)
					match.MatchReasons = appendUnique(match.MatchReasons,
						fmt.Sprintf("Imposes %s (action: %s)", obligType, action.Type))
					match.Obligations = append(match.Obligations, ann)
				}
			}
		}
	}
}

// findTriggeredMatches finds provisions referenced by direct matches.
func (m *ProvisionMatcher) findTriggeredMatches(matches map[int]*MatchedProvision) {
	directArticles := make([]int, 0)
	for artNum, match := range matches {
		if match.Relevance == RelevanceDirect {
			directArticles = append(directArticles, artNum)
		}
	}

	for _, artNum := range directArticles {
		artURI := fmt.Sprintf("%sGDPR:Art%d", m.baseURI, artNum)

		// Find provisions that reference this article
		triples := m.store.Find("", store.PropReferences, artURI)
		for _, t := range triples {
			refArtNum := extractArticleNum(t.Subject)
			if refArtNum > 0 && matches[refArtNum] == nil {
				match := m.getOrCreateMatch(matches, refArtNum)
				match.Relevance = RelevanceTriggered
				match.Score = max(match.Score, 0.7)
				match.MatchReasons = appendUnique(match.MatchReasons,
					fmt.Sprintf("References Article %d", artNum))
				match.ReferencedBy = appendUnique(match.ReferencedBy, artNum)
			}
		}

		// Find provisions referenced by this article
		triples = m.store.Find(artURI, store.PropReferences, "")
		for _, t := range triples {
			refArtNum := extractArticleNum(t.Object)
			if refArtNum > 0 && matches[refArtNum] == nil {
				match := m.getOrCreateMatch(matches, refArtNum)
				if match.Relevance != RelevanceDirect && match.Relevance != RelevanceTriggered {
					match.Relevance = RelevanceTriggered
				}
				match.Score = max(match.Score, 0.6)
				match.MatchReasons = appendUnique(match.MatchReasons,
					fmt.Sprintf("Referenced by Article %d", artNum))
			}
		}
	}
}

// findRelatedMatches finds provisions related by keywords.
func (m *ProvisionMatcher) findRelatedMatches(scenario *Scenario, matches map[int]*MatchedProvision) {
	keywords := scenario.GetAllKeywords()

	for _, keyword := range keywords {
		if articles, ok := m.keywordArticles[keyword]; ok {
			for _, artNum := range articles {
				if matches[artNum] == nil {
					match := m.getOrCreateMatch(matches, artNum)
					match.Relevance = RelevanceRelated
					match.Score = max(match.Score, 0.3)
					match.Keywords = appendUnique(match.Keywords, keyword)
					match.MatchReasons = appendUnique(match.MatchReasons,
						fmt.Sprintf("Contains keyword: %s", keyword))
				} else {
					// Already matched - add keyword info
					match := matches[artNum]
					match.Keywords = appendUnique(match.Keywords, keyword)
					// Boost score slightly for keyword matches
					match.Score = min(match.Score+0.05, 1.0)
				}
			}
		}
	}
}

// getOrCreateMatch gets or creates a match for an article.
func (m *ProvisionMatcher) getOrCreateMatch(matches map[int]*MatchedProvision, artNum int) *MatchedProvision {
	if match, ok := matches[artNum]; ok {
		return match
	}

	match := &MatchedProvision{
		URI:          fmt.Sprintf("%sGDPR:Art%d", m.baseURI, artNum),
		ArticleNum:   artNum,
		Title:        m.getArticleTitle(artNum),
		MatchReasons: make([]string, 0),
		Rights:       make([]*extract.SemanticAnnotation, 0),
		Obligations:  make([]*extract.SemanticAnnotation, 0),
		Keywords:     make([]string, 0),
		ReferencedBy: make([]int, 0),
	}
	matches[artNum] = match
	return match
}

// getArticleTitle retrieves the title for an article.
func (m *ProvisionMatcher) getArticleTitle(artNum int) string {
	uri := fmt.Sprintf("%sGDPR:Art%d", m.baseURI, artNum)
	triples := m.store.Find(uri, store.PropTitle, "")
	if len(triples) > 0 {
		return triples[0].Object
	}
	return fmt.Sprintf("Article %d", artNum)
}

// calculateSummary calculates summary statistics for the match result.
func (m *ProvisionMatcher) calculateSummary(result *MatchResult) {
	result.Summary.TotalMatches = len(result.AllMatches)
	result.Summary.DirectCount = len(result.DirectMatches)
	result.Summary.TriggeredCount = len(result.TriggeredMatches)
	result.Summary.RelatedCount = len(result.RelatedMatches)

	rightsSet := make(map[extract.RightType]bool)
	obligsSet := make(map[extract.ObligationType]bool)

	for _, match := range result.AllMatches {
		for _, right := range match.Rights {
			rightsSet[right.RightType] = true
		}
		for _, oblig := range match.Obligations {
			obligsSet[oblig.ObligationType] = true
		}

		if match.Relevance == RelevanceDirect {
			result.Summary.KeyArticles = appendUnique(result.Summary.KeyArticles, match.ArticleNum)
		}
	}

	for rt := range rightsSet {
		result.Summary.RightsInvolved = append(result.Summary.RightsInvolved, rt)
	}
	for ot := range obligsSet {
		result.Summary.ObligationsInvolved = append(result.Summary.ObligationsInvolved, ot)
	}
}

// extractArticleNum extracts article number from a URI.
func extractArticleNum(uri string) int {
	if idx := strings.Index(uri, ":Art"); idx != -1 {
		rest := uri[idx+4:]
		var num int
		for _, c := range rest {
			if c >= '0' && c <= '9' {
				num = num*10 + int(c-'0')
			} else {
				break
			}
		}
		return num
	}
	return 0
}

// extractWordsFromText extracts meaningful words from text.
func extractWordsFromText(text string) []string {
	words := make([]string, 0)
	text = strings.ToLower(text)

	// Split by non-alphanumeric
	var word strings.Builder
	for _, c := range text {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			word.WriteRune(c)
		} else if word.Len() > 0 {
			w := word.String()
			if len(w) > 3 && isRelevantWord(w) {
				words = append(words, w)
			}
			word.Reset()
		}
	}
	if word.Len() > 3 {
		w := word.String()
		if isRelevantWord(w) {
			words = append(words, w)
		}
	}

	return words
}

// appendUnique appends value to slice if not already present.
func appendUnique[T comparable](slice []T, value T) []T {
	for _, v := range slice {
		if v == value {
			return slice
		}
	}
	return append(slice, value)
}

// max returns the maximum of two float64 values.
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// min returns the minimum of two float64 values.
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// ToJSON serializes the match result to JSON.
func (r *MatchResult) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// String returns a human-readable string representation.
func (r *MatchResult) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Provision Matching Results for: %s\n", r.Scenario.Name))
	sb.WriteString("=" + strings.Repeat("=", 50) + "\n\n")

	sb.WriteString("Summary:\n")
	sb.WriteString(fmt.Sprintf("  Total matches: %d\n", r.Summary.TotalMatches))
	sb.WriteString(fmt.Sprintf("  Direct: %d\n", r.Summary.DirectCount))
	sb.WriteString(fmt.Sprintf("  Triggered: %d\n", r.Summary.TriggeredCount))
	sb.WriteString(fmt.Sprintf("  Related: %d\n\n", r.Summary.RelatedCount))

	if len(r.DirectMatches) > 0 {
		sb.WriteString("Direct Matches:\n")
		for _, match := range r.DirectMatches {
			sb.WriteString(fmt.Sprintf("  Art %d: %s (score: %.2f)\n",
				match.ArticleNum, match.Title, match.Score))
			for _, reason := range match.MatchReasons {
				sb.WriteString(fmt.Sprintf("    - %s\n", reason))
			}
		}
		sb.WriteString("\n")
	}

	if len(r.TriggeredMatches) > 0 {
		sb.WriteString("Triggered Matches:\n")
		for _, match := range r.TriggeredMatches {
			sb.WriteString(fmt.Sprintf("  Art %d: %s (score: %.2f)\n",
				match.ArticleNum, match.Title, match.Score))
			for _, reason := range match.MatchReasons {
				sb.WriteString(fmt.Sprintf("    - %s\n", reason))
			}
		}
		sb.WriteString("\n")
	}

	if len(r.RelatedMatches) > 0 {
		sb.WriteString(fmt.Sprintf("Related Matches: (%d articles)\n", len(r.RelatedMatches)))
		// Show top 5 related
		count := 0
		for _, match := range r.RelatedMatches {
			if count >= 5 {
				sb.WriteString(fmt.Sprintf("  ... and %d more\n", len(r.RelatedMatches)-5))
				break
			}
			sb.WriteString(fmt.Sprintf("  Art %d: %s (score: %.2f)\n",
				match.ArticleNum, match.Title, match.Score))
			count++
		}
	}

	return sb.String()
}

// FormatTable formats the result as a table.
func (r *MatchResult) FormatTable() string {
	var sb strings.Builder

	sb.WriteString("+----------+---------+------+--------------------------------------------------+\n")
	sb.WriteString("| Article  | Relev.  | Score| Title                                            |\n")
	sb.WriteString("+----------+---------+------+--------------------------------------------------+\n")

	for _, match := range r.AllMatches {
		title := match.Title
		if len(title) > 48 {
			title = title[:45] + "..."
		}
		sb.WriteString(fmt.Sprintf("| Art %-4d | %-7s | %.2f | %-48s |\n",
			match.ArticleNum, match.Relevance, match.Score, title))
	}

	sb.WriteString("+----------+---------+------+--------------------------------------------------+\n")

	return sb.String()
}
