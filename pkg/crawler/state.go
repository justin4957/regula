package crawler

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// CrawlStatus represents the overall status of a crawl session.
type CrawlStatus string

const (
	// CrawlStatusRunning indicates the crawl is in progress.
	CrawlStatusRunning CrawlStatus = "running"

	// CrawlStatusPaused indicates the crawl was paused and can be resumed.
	CrawlStatusPaused CrawlStatus = "paused"

	// CrawlStatusCompleted indicates the crawl finished successfully.
	CrawlStatusCompleted CrawlStatus = "completed"

	// CrawlStatusFailed indicates the crawl terminated due to an error.
	CrawlStatusFailed CrawlStatus = "failed"
)

// CrawlState holds the serializable state of a crawl session for resumability.
type CrawlState struct {
	// Status is the overall crawl status.
	Status CrawlStatus `json:"status"`

	// Frontier is the queue of items yet to be processed.
	Frontier []*CrawlItem `json:"frontier"`

	// Visited is the set of document IDs already processed.
	Visited map[string]bool `json:"visited"`

	// Seeds is the original set of seeds that started the crawl.
	Seeds []CrawlSeed `json:"seeds"`

	// Config is the crawl configuration used.
	Config CrawlConfig `json:"config"`

	// Statistics tracks cumulative crawl statistics.
	Statistics CrawlStateStats `json:"statistics"`

	// StartedAt is when the crawl session started.
	StartedAt time.Time `json:"started_at"`

	// UpdatedAt is when the state was last saved.
	UpdatedAt time.Time `json:"updated_at"`

	// ProcessedItems holds completed/failed items for reporting.
	ProcessedItems []*CrawlItem `json:"processed_items"`
}

// CrawlStateStats tracks cumulative statistics during a crawl.
type CrawlStateStats struct {
	TotalDiscovered int `json:"total_discovered"`
	TotalIngested   int `json:"total_ingested"`
	TotalFailed     int `json:"total_failed"`
	TotalSkipped    int `json:"total_skipped"`
	CurrentDepth    int `json:"current_depth"`
	MaxDepthReached int `json:"max_depth_reached"`
}

// NewCrawlState creates a new empty crawl state.
func NewCrawlState(seeds []CrawlSeed, config CrawlConfig) *CrawlState {
	return &CrawlState{
		Status:         CrawlStatusRunning,
		Frontier:       make([]*CrawlItem, 0),
		Visited:        make(map[string]bool),
		Seeds:          seeds,
		Config:         config,
		StartedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		ProcessedItems: make([]*CrawlItem, 0),
	}
}

// Enqueue adds an item to the frontier if not already visited.
func (state *CrawlState) Enqueue(item *CrawlItem) bool {
	if state.Visited[item.DocumentID] {
		return false
	}
	state.Frontier = append(state.Frontier, item)
	state.Statistics.TotalDiscovered++
	return true
}

// Dequeue removes and returns the next item from the frontier.
// Returns nil if the frontier is empty.
func (state *CrawlState) Dequeue() *CrawlItem {
	if len(state.Frontier) == 0 {
		return nil
	}
	nextItem := state.Frontier[0]
	state.Frontier = state.Frontier[1:]
	return nextItem
}

// MarkVisited records a document ID as visited.
func (state *CrawlState) MarkVisited(documentID string) {
	state.Visited[documentID] = true
}

// IsVisited checks if a document ID has already been processed.
func (state *CrawlState) IsVisited(documentID string) bool {
	return state.Visited[documentID]
}

// RecordProcessed adds a completed/failed item to the processed list.
func (state *CrawlState) RecordProcessed(item *CrawlItem) {
	state.ProcessedItems = append(state.ProcessedItems, item)
	switch item.Status {
	case CrawlItemIngested:
		state.Statistics.TotalIngested++
	case CrawlItemFailed:
		state.Statistics.TotalFailed++
	case CrawlItemSkipped:
		state.Statistics.TotalSkipped++
	}
	if item.Depth > state.Statistics.MaxDepthReached {
		state.Statistics.MaxDepthReached = item.Depth
	}
}

// FrontierSize returns the number of items in the frontier.
func (state *CrawlState) FrontierSize() int {
	return len(state.Frontier)
}

// WithinLimits checks if the crawl is still within configured limits.
func (state *CrawlState) WithinLimits() bool {
	return state.Statistics.TotalIngested < state.Config.MaxDocuments
}

// SaveState writes the crawl state to disk as JSON.
func (state *CrawlState) SaveState(statePath string) error {
	state.UpdatedAt = time.Now()

	stateJSON, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal crawl state: %w", err)
	}

	if err := os.WriteFile(statePath, stateJSON, 0644); err != nil {
		return fmt.Errorf("failed to write crawl state to %s: %w", statePath, err)
	}

	return nil
}

// LoadState reads a crawl state from disk.
func LoadState(statePath string) (*CrawlState, error) {
	stateJSON, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read crawl state from %s: %w", statePath, err)
	}

	var crawlState CrawlState
	if err := json.Unmarshal(stateJSON, &crawlState); err != nil {
		return nil, fmt.Errorf("failed to parse crawl state: %w", err)
	}

	if crawlState.Visited == nil {
		crawlState.Visited = make(map[string]bool)
	}

	return &crawlState, nil
}
