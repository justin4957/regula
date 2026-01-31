package crawler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCrawlStateEnqueueDequeue(t *testing.T) {
	crawlState := NewCrawlState(nil, DefaultCrawlConfig())

	item1 := &CrawlItem{DocumentID: "doc-1", Citation: "42 USC 1320d", Depth: 0}
	item2 := &CrawlItem{DocumentID: "doc-2", Citation: "15 USC 6501", Depth: 1}

	if !crawlState.Enqueue(item1) {
		t.Error("expected first enqueue to succeed")
	}
	if !crawlState.Enqueue(item2) {
		t.Error("expected second enqueue to succeed")
	}

	if crawlState.FrontierSize() != 2 {
		t.Errorf("frontier size = %d, want 2", crawlState.FrontierSize())
	}

	dequeued := crawlState.Dequeue()
	if dequeued == nil || dequeued.DocumentID != "doc-1" {
		t.Error("expected first dequeued item to be doc-1")
	}

	if crawlState.FrontierSize() != 1 {
		t.Errorf("frontier size = %d, want 1", crawlState.FrontierSize())
	}
}

func TestCrawlStateVisited(t *testing.T) {
	crawlState := NewCrawlState(nil, DefaultCrawlConfig())

	if crawlState.IsVisited("doc-1") {
		t.Error("doc-1 should not be visited initially")
	}

	crawlState.MarkVisited("doc-1")

	if !crawlState.IsVisited("doc-1") {
		t.Error("doc-1 should be visited after marking")
	}

	// Enqueue should fail for visited documents
	item := &CrawlItem{DocumentID: "doc-1", Depth: 0}
	crawlState.MarkVisited("doc-1")
	if crawlState.Enqueue(item) {
		t.Error("enqueue should fail for visited document")
	}
}

func TestCrawlStateDequeueEmpty(t *testing.T) {
	crawlState := NewCrawlState(nil, DefaultCrawlConfig())

	if dequeued := crawlState.Dequeue(); dequeued != nil {
		t.Error("dequeue from empty frontier should return nil")
	}
}

func TestCrawlStateRecordProcessed(t *testing.T) {
	crawlState := NewCrawlState(nil, DefaultCrawlConfig())

	crawlState.RecordProcessed(&CrawlItem{Status: CrawlItemIngested, Depth: 1})
	crawlState.RecordProcessed(&CrawlItem{Status: CrawlItemFailed, Depth: 2})
	crawlState.RecordProcessed(&CrawlItem{Status: CrawlItemSkipped, Depth: 1})

	if crawlState.Statistics.TotalIngested != 1 {
		t.Errorf("total ingested = %d, want 1", crawlState.Statistics.TotalIngested)
	}
	if crawlState.Statistics.TotalFailed != 1 {
		t.Errorf("total failed = %d, want 1", crawlState.Statistics.TotalFailed)
	}
	if crawlState.Statistics.TotalSkipped != 1 {
		t.Errorf("total skipped = %d, want 1", crawlState.Statistics.TotalSkipped)
	}
	if crawlState.Statistics.MaxDepthReached != 2 {
		t.Errorf("max depth = %d, want 2", crawlState.Statistics.MaxDepthReached)
	}
}

func TestCrawlStateWithinLimits(t *testing.T) {
	config := DefaultCrawlConfig()
	config.MaxDocuments = 2
	crawlState := NewCrawlState(nil, config)

	if !crawlState.WithinLimits() {
		t.Error("should be within limits initially")
	}

	crawlState.RecordProcessed(&CrawlItem{Status: CrawlItemIngested})
	crawlState.RecordProcessed(&CrawlItem{Status: CrawlItemIngested})

	if crawlState.WithinLimits() {
		t.Error("should exceed limits after 2 ingested documents with max=2")
	}
}

func TestCrawlStateSaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "crawl-state.json")

	originalState := NewCrawlState(
		[]CrawlSeed{{Type: SeedTypeCitation, Value: "42 USC 1320d"}},
		DefaultCrawlConfig(),
	)
	originalState.Enqueue(&CrawlItem{DocumentID: "doc-1", Citation: "42 USC 1320d", Depth: 0})
	originalState.Enqueue(&CrawlItem{DocumentID: "doc-2", Citation: "15 USC 6501", Depth: 1})
	originalState.MarkVisited("doc-0")
	originalState.RecordProcessed(&CrawlItem{Status: CrawlItemIngested, Depth: 0})

	if err := originalState.SaveState(statePath); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Fatal("state file was not created")
	}

	// Load and verify
	loadedState, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	if loadedState.FrontierSize() != 2 {
		t.Errorf("loaded frontier size = %d, want 2", loadedState.FrontierSize())
	}
	if !loadedState.IsVisited("doc-0") {
		t.Error("loaded state should have doc-0 as visited")
	}
	if len(loadedState.Seeds) != 1 {
		t.Errorf("loaded seeds count = %d, want 1", len(loadedState.Seeds))
	}
	if loadedState.Statistics.TotalIngested != 1 {
		t.Errorf("loaded total ingested = %d, want 1", loadedState.Statistics.TotalIngested)
	}
}

func TestLoadStateFileNotFound(t *testing.T) {
	_, err := LoadState("/nonexistent/path/state.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoadStateMalformedJSON(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "bad-state.json")

	os.WriteFile(statePath, []byte("{invalid json}"), 0644)

	_, err := LoadState(statePath)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}
