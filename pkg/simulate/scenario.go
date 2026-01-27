// Package simulate provides scenario simulation and provision matching.
package simulate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coolbeans/regula/pkg/extract"
)

// ActionType represents the type of action in a scenario.
type ActionType string

const (
	ActionWithdrawConsent   ActionType = "withdraw_consent"
	ActionRequestAccess     ActionType = "request_access"
	ActionRequestErasure    ActionType = "request_erasure"
	ActionRequestRectify    ActionType = "request_rectification"
	ActionRequestPortability ActionType = "request_portability"
	ActionObjectProcessing  ActionType = "object_processing"
	ActionProcessData       ActionType = "process_data"
	ActionTransferData      ActionType = "transfer_data"
	ActionBreach            ActionType = "data_breach"
	ActionCollectData       ActionType = "collect_data"
	ActionProvideConsent    ActionType = "provide_consent"
	ActionFileComplaint     ActionType = "file_complaint"
	ActionCustom            ActionType = "custom"
)

// Scenario represents a compliance scenario to evaluate.
type Scenario struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Entities    []ScenarioEntity       `json:"entities"`
	Actions     []ScenarioAction       `json:"actions"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Keywords    []string               `json:"keywords,omitempty"`
}

// ScenarioEntity represents an entity involved in the scenario.
type ScenarioEntity struct {
	ID         string           `json:"id"`
	Type       extract.EntityType `json:"type"`
	Name       string           `json:"name,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// ScenarioAction represents an action in the scenario.
type ScenarioAction struct {
	ID          string     `json:"id"`
	Type        ActionType `json:"type"`
	Actor       string     `json:"actor"`      // Entity ID
	Target      string     `json:"target,omitempty"` // Entity ID or data type
	Description string     `json:"description,omitempty"`
	Triggers    []string   `json:"triggers,omitempty"` // Action IDs triggered by this
	Keywords    []string   `json:"keywords,omitempty"`
}

// NewScenario creates a new scenario with the given name.
func NewScenario(name string) *Scenario {
	return &Scenario{
		ID:       generateID(name),
		Name:     name,
		Entities: make([]ScenarioEntity, 0),
		Actions:  make([]ScenarioAction, 0),
		Context:  make(map[string]interface{}),
		Keywords: make([]string, 0),
	}
}

// AddEntity adds an entity to the scenario.
func (s *Scenario) AddEntity(entityType extract.EntityType, name string) *Scenario {
	entity := ScenarioEntity{
		ID:         generateID(name),
		Type:       entityType,
		Name:       name,
		Attributes: make(map[string]string),
	}
	s.Entities = append(s.Entities, entity)
	return s
}

// AddAction adds an action to the scenario.
func (s *Scenario) AddAction(actionType ActionType, actorID string, description string) *Scenario {
	action := ScenarioAction{
		ID:          fmt.Sprintf("action_%d", len(s.Actions)+1),
		Type:        actionType,
		Actor:       actorID,
		Description: description,
		Keywords:    extractKeywords(actionType, description),
	}
	s.Actions = append(s.Actions, action)
	return s
}

// AddKeyword adds a keyword to the scenario for matching.
func (s *Scenario) AddKeyword(keyword string) *Scenario {
	s.Keywords = append(s.Keywords, strings.ToLower(keyword))
	return s
}

// GetAllKeywords returns all keywords from the scenario.
func (s *Scenario) GetAllKeywords() []string {
	keywords := make([]string, 0)
	keywords = append(keywords, s.Keywords...)

	for _, action := range s.Actions {
		keywords = append(keywords, action.Keywords...)
	}

	return uniqueStrings(keywords)
}

// GetEntityTypes returns all entity types in the scenario.
func (s *Scenario) GetEntityTypes() []extract.EntityType {
	types := make([]extract.EntityType, 0)
	seen := make(map[extract.EntityType]bool)

	for _, entity := range s.Entities {
		if !seen[entity.Type] {
			types = append(types, entity.Type)
			seen[entity.Type] = true
		}
	}

	return types
}

// GetActionTypes returns all action types in the scenario.
func (s *Scenario) GetActionTypes() []ActionType {
	types := make([]ActionType, 0)
	seen := make(map[ActionType]bool)

	for _, action := range s.Actions {
		if !seen[action.Type] {
			types = append(types, action.Type)
			seen[action.Type] = true
		}
	}

	return types
}

// ToJSON serializes the scenario to JSON.
func (s *Scenario) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// ScenarioFromJSON parses a scenario from JSON.
func ScenarioFromJSON(data []byte) (*Scenario, error) {
	var scenario Scenario
	if err := json.Unmarshal(data, &scenario); err != nil {
		return nil, err
	}
	return &scenario, nil
}

// generateID creates a simple ID from a name.
func generateID(name string) string {
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "_")
	// Keep only alphanumeric and underscore
	var result strings.Builder
	for _, c := range id {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			result.WriteRune(c)
		}
	}
	return result.String()
}

// extractKeywords extracts keywords from an action type and description.
func extractKeywords(actionType ActionType, description string) []string {
	keywords := make([]string, 0)

	// Add keywords based on action type
	switch actionType {
	case ActionWithdrawConsent:
		keywords = append(keywords, "consent", "withdraw", "withdrawal")
	case ActionRequestAccess:
		keywords = append(keywords, "access", "request", "obtain", "copy")
	case ActionRequestErasure:
		keywords = append(keywords, "erasure", "delete", "forgotten", "erase")
	case ActionRequestRectify:
		keywords = append(keywords, "rectification", "correct", "rectify", "amend")
	case ActionRequestPortability:
		keywords = append(keywords, "portability", "transfer", "receive", "transmit")
	case ActionObjectProcessing:
		keywords = append(keywords, "object", "objection", "stop")
	case ActionProcessData:
		keywords = append(keywords, "process", "processing", "use")
	case ActionTransferData:
		keywords = append(keywords, "transfer", "third country", "international")
	case ActionBreach:
		keywords = append(keywords, "breach", "incident", "security", "notification")
	case ActionCollectData:
		keywords = append(keywords, "collect", "obtain", "gather")
	case ActionProvideConsent:
		keywords = append(keywords, "consent", "agree", "permission")
	case ActionFileComplaint:
		keywords = append(keywords, "complaint", "lodge", "supervisory")
	}

	// Extract keywords from description
	descWords := strings.Fields(strings.ToLower(description))
	for _, word := range descWords {
		if len(word) > 3 && isRelevantWord(word) {
			keywords = append(keywords, word)
		}
	}

	return uniqueStrings(keywords)
}

// isRelevantWord checks if a word is relevant for keyword extraction.
func isRelevantWord(word string) bool {
	// Skip common words
	stopWords := map[string]bool{
		"the": true, "and": true, "for": true, "with": true,
		"that": true, "this": true, "from": true, "have": true,
		"been": true, "will": true, "shall": true, "must": true,
		"their": true, "they": true, "which": true, "when": true,
	}
	return !stopWords[word]
}

// uniqueStrings returns unique strings from a slice.
func uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// PredefinedScenarios contains common compliance scenarios.
var PredefinedScenarios = map[string]*Scenario{
	"consent_withdrawal": ConsentWithdrawalScenario(),
	"access_request":     AccessRequestScenario(),
	"erasure_request":    ErasureRequestScenario(),
	"data_breach":        DataBreachScenario(),
}

// ConsentWithdrawalScenario creates a scenario for consent withdrawal.
func ConsentWithdrawalScenario() *Scenario {
	s := NewScenario("Consent Withdrawal")
	s.Description = "Data subject withdraws previously given consent for data processing"
	s.AddEntity(extract.EntityDataSubject, "Data Subject")
	s.AddEntity(extract.EntityController, "Data Controller")
	s.AddAction(ActionWithdrawConsent, "data_subject", "Data subject withdraws consent")
	s.AddKeyword("consent")
	s.AddKeyword("withdraw")
	s.AddKeyword("withdrawal")
	s.AddKeyword("revoke")
	return s
}

// AccessRequestScenario creates a scenario for data access request.
func AccessRequestScenario() *Scenario {
	s := NewScenario("Data Access Request")
	s.Description = "Data subject requests access to their personal data"
	s.AddEntity(extract.EntityDataSubject, "Data Subject")
	s.AddEntity(extract.EntityController, "Data Controller")
	s.AddAction(ActionRequestAccess, "data_subject", "Data subject requests access to personal data")
	s.AddKeyword("access")
	s.AddKeyword("copy")
	s.AddKeyword("obtain")
	return s
}

// ErasureRequestScenario creates a scenario for data erasure request.
func ErasureRequestScenario() *Scenario {
	s := NewScenario("Data Erasure Request")
	s.Description = "Data subject requests erasure of their personal data"
	s.AddEntity(extract.EntityDataSubject, "Data Subject")
	s.AddEntity(extract.EntityController, "Data Controller")
	s.AddAction(ActionRequestErasure, "data_subject", "Data subject requests erasure of personal data")
	s.AddKeyword("erasure")
	s.AddKeyword("delete")
	s.AddKeyword("forgotten")
	return s
}

// DataBreachScenario creates a scenario for data breach handling.
func DataBreachScenario() *Scenario {
	s := NewScenario("Data Breach")
	s.Description = "Personal data breach occurs and must be handled"
	s.AddEntity(extract.EntityController, "Data Controller")
	s.AddEntity(extract.EntitySupervisoryAuth, "Supervisory Authority")
	s.AddEntity(extract.EntityDataSubject, "Affected Data Subjects")
	s.AddAction(ActionBreach, "data_controller", "Personal data breach detected")
	s.AddKeyword("breach")
	s.AddKeyword("notification")
	s.AddKeyword("security")
	s.AddKeyword("incident")
	return s
}
