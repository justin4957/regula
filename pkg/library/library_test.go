package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitAndOpen(t *testing.T) {
	tempDir := t.TempDir()
	libraryPath := filepath.Join(tempDir, "test-library")

	lib, err := Init(libraryPath, "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if lib == nil {
		t.Fatal("library is nil")
	}

	// Verify manifest file exists
	if _, err := os.Stat(filepath.Join(libraryPath, manifestFileName)); os.IsNotExist(err) {
		t.Error("manifest file was not created")
	}

	// Open should succeed
	reopened, err := Open(libraryPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if reopened.BaseURI() != defaultBaseURI {
		t.Errorf("unexpected base URI: %s", reopened.BaseURI())
	}
}

func TestOpenNonExistent(t *testing.T) {
	_, err := Open("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent library")
	}
}

func TestAddDocument(t *testing.T) {
	tempDir := t.TempDir()
	lib, err := Init(filepath.Join(tempDir, "lib"), "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	sourceText, err := os.ReadFile(filepath.Join("..", "..", "testdata", "vcdpa.txt"))
	if err != nil {
		t.Skipf("VCDPA test data not available: %v", err)
	}

	entry, err := lib.AddDocument("us-va-vcdpa", sourceText, AddOptions{
		ShortName:    "VCDPA",
		Jurisdiction: "US-VA",
		Format:       "us",
	})
	if err != nil {
		t.Fatalf("AddDocument failed: %v", err)
	}

	if entry.ID != "us-va-vcdpa" {
		t.Errorf("unexpected ID: %s", entry.ID)
	}
	if entry.Status != StatusReady {
		t.Errorf("unexpected status: %s", entry.Status)
	}
	if entry.Stats == nil || entry.Stats.TotalTriples == 0 {
		t.Error("expected stats with triples > 0")
	}
}

func TestAddDocumentIdempotent(t *testing.T) {
	tempDir := t.TempDir()
	lib, err := Init(filepath.Join(tempDir, "lib"), "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	sourceText, err := os.ReadFile(filepath.Join("..", "..", "testdata", "vcdpa.txt"))
	if err != nil {
		t.Skipf("VCDPA test data not available: %v", err)
	}

	opts := AddOptions{ShortName: "VCDPA", Jurisdiction: "US-VA"}

	entry1, err := lib.AddDocument("us-va-vcdpa", sourceText, opts)
	if err != nil {
		t.Fatalf("first AddDocument failed: %v", err)
	}

	// Second add should be idempotent (returns existing entry)
	entry2, err := lib.AddDocument("us-va-vcdpa", sourceText, opts)
	if err != nil {
		t.Fatalf("second AddDocument failed: %v", err)
	}

	if entry1.ID != entry2.ID {
		t.Error("idempotent add returned different entry")
	}

	docs := lib.ListDocuments()
	if len(docs) != 1 {
		t.Errorf("expected 1 document, got %d", len(docs))
	}
}

func TestRemoveDocument(t *testing.T) {
	tempDir := t.TempDir()
	lib, err := Init(filepath.Join(tempDir, "lib"), "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	sourceText, err := os.ReadFile(filepath.Join("..", "..", "testdata", "vcdpa.txt"))
	if err != nil {
		t.Skipf("VCDPA test data not available: %v", err)
	}

	_, err = lib.AddDocument("us-va-vcdpa", sourceText, AddOptions{})
	if err != nil {
		t.Fatalf("AddDocument failed: %v", err)
	}

	err = lib.RemoveDocument("us-va-vcdpa")
	if err != nil {
		t.Fatalf("RemoveDocument failed: %v", err)
	}

	if lib.GetDocument("us-va-vcdpa") != nil {
		t.Error("document still exists after removal")
	}

	docs := lib.ListDocuments()
	if len(docs) != 0 {
		t.Errorf("expected 0 documents, got %d", len(docs))
	}
}

func TestRemoveDocumentNotFound(t *testing.T) {
	tempDir := t.TempDir()
	lib, err := Init(filepath.Join(tempDir, "lib"), "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	err = lib.RemoveDocument("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent document")
	}
}

func TestLoadTripleStore(t *testing.T) {
	tempDir := t.TempDir()
	lib, err := Init(filepath.Join(tempDir, "lib"), "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	sourceText, err := os.ReadFile(filepath.Join("..", "..", "testdata", "vcdpa.txt"))
	if err != nil {
		t.Skipf("VCDPA test data not available: %v", err)
	}

	entry, err := lib.AddDocument("us-va-vcdpa", sourceText, AddOptions{})
	if err != nil {
		t.Fatalf("AddDocument failed: %v", err)
	}

	tripleStore, err := lib.LoadTripleStore("us-va-vcdpa")
	if err != nil {
		t.Fatalf("LoadTripleStore failed: %v", err)
	}

	if tripleStore.Count() != entry.Stats.TotalTriples {
		t.Errorf("triple count mismatch: got %d, want %d", tripleStore.Count(), entry.Stats.TotalTriples)
	}
}

func TestLoadTripleStoreNotFound(t *testing.T) {
	tempDir := t.TempDir()
	lib, err := Init(filepath.Join(tempDir, "lib"), "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	_, err = lib.LoadTripleStore("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent document")
	}
}

func TestLoadMergedTripleStore(t *testing.T) {
	tempDir := t.TempDir()
	lib, err := Init(filepath.Join(tempDir, "lib"), "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	vcdpaText, err := os.ReadFile(filepath.Join("..", "..", "testdata", "vcdpa.txt"))
	if err != nil {
		t.Skipf("VCDPA test data not available: %v", err)
	}
	tdpsaText, err := os.ReadFile(filepath.Join("..", "..", "testdata", "tdpsa.txt"))
	if err != nil {
		t.Skipf("TDPSA test data not available: %v", err)
	}

	vcdpaEntry, err := lib.AddDocument("us-va-vcdpa", vcdpaText, AddOptions{})
	if err != nil {
		t.Fatalf("AddDocument (VCDPA) failed: %v", err)
	}
	tdpsaEntry, err := lib.AddDocument("us-tx-tdpsa", tdpsaText, AddOptions{})
	if err != nil {
		t.Fatalf("AddDocument (TDPSA) failed: %v", err)
	}

	merged, err := lib.LoadMergedTripleStore("us-va-vcdpa", "us-tx-tdpsa")
	if err != nil {
		t.Fatalf("LoadMergedTripleStore failed: %v", err)
	}

	// Merged count should be at least as large as the larger of the two
	minExpected := vcdpaEntry.Stats.TotalTriples
	if tdpsaEntry.Stats.TotalTriples > minExpected {
		minExpected = tdpsaEntry.Stats.TotalTriples
	}
	if merged.Count() < minExpected {
		t.Errorf("merged count %d is less than min expected %d", merged.Count(), minExpected)
	}
}

func TestLoadSourceText(t *testing.T) {
	tempDir := t.TempDir()
	lib, err := Init(filepath.Join(tempDir, "lib"), "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	originalText, err := os.ReadFile(filepath.Join("..", "..", "testdata", "vcdpa.txt"))
	if err != nil {
		t.Skipf("VCDPA test data not available: %v", err)
	}

	_, err = lib.AddDocument("us-va-vcdpa", originalText, AddOptions{})
	if err != nil {
		t.Fatalf("AddDocument failed: %v", err)
	}

	recovered, err := lib.LoadSourceText("us-va-vcdpa")
	if err != nil {
		t.Fatalf("LoadSourceText failed: %v", err)
	}

	if string(recovered) != string(originalText) {
		t.Error("recovered source text does not match original")
	}
}

func TestStats(t *testing.T) {
	tempDir := t.TempDir()
	lib, err := Init(filepath.Join(tempDir, "lib"), "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	sourceText, err := os.ReadFile(filepath.Join("..", "..", "testdata", "vcdpa.txt"))
	if err != nil {
		t.Skipf("VCDPA test data not available: %v", err)
	}

	_, err = lib.AddDocument("us-va-vcdpa", sourceText, AddOptions{Jurisdiction: "US-VA"})
	if err != nil {
		t.Fatalf("AddDocument failed: %v", err)
	}

	libraryStats := lib.Stats()
	if libraryStats.TotalDocuments != 1 {
		t.Errorf("expected 1 document, got %d", libraryStats.TotalDocuments)
	}
	if libraryStats.TotalTriples == 0 {
		t.Error("expected triples > 0")
	}
	if libraryStats.ByJurisdiction["US-VA"] != 1 {
		t.Errorf("expected 1 US-VA document, got %d", libraryStats.ByJurisdiction["US-VA"])
	}
	if libraryStats.ByStatus["ready"] != 1 {
		t.Errorf("expected 1 ready document, got %d", libraryStats.ByStatus["ready"])
	}
}

func TestListDocuments(t *testing.T) {
	tempDir := t.TempDir()
	lib, err := Init(filepath.Join(tempDir, "lib"), "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Empty library
	docs := lib.ListDocuments()
	if len(docs) != 0 {
		t.Errorf("expected 0 documents, got %d", len(docs))
	}

	sourceText, err := os.ReadFile(filepath.Join("..", "..", "testdata", "vcdpa.txt"))
	if err != nil {
		t.Skipf("VCDPA test data not available: %v", err)
	}

	_, err = lib.AddDocument("b-doc", sourceText, AddOptions{})
	if err != nil {
		t.Fatalf("AddDocument (b-doc) failed: %v", err)
	}
	_, err = lib.AddDocument("a-doc", sourceText, AddOptions{})
	if err != nil {
		t.Fatalf("AddDocument (a-doc) failed: %v", err)
	}

	docs = lib.ListDocuments()
	if len(docs) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(docs))
	}
	// Should be sorted by ID
	if docs[0].ID != "a-doc" {
		t.Errorf("expected first document to be 'a-doc', got '%s'", docs[0].ID)
	}
}

func TestPersistenceAcrossOpenClose(t *testing.T) {
	tempDir := t.TempDir()
	libraryPath := filepath.Join(tempDir, "lib")

	lib, err := Init(libraryPath, "")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	sourceText, err := os.ReadFile(filepath.Join("..", "..", "testdata", "vcdpa.txt"))
	if err != nil {
		t.Skipf("VCDPA test data not available: %v", err)
	}

	_, err = lib.AddDocument("test-doc", sourceText, AddOptions{Jurisdiction: "US-VA"})
	if err != nil {
		t.Fatalf("AddDocument failed: %v", err)
	}

	lib.Close()

	// Re-open and verify persistence
	reopened, err := Open(libraryPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	entry := reopened.GetDocument("test-doc")
	if entry == nil {
		t.Fatal("document not found after re-open")
	}
	if entry.Jurisdiction != "US-VA" {
		t.Errorf("unexpected jurisdiction: %s", entry.Jurisdiction)
	}

	tripleStore, err := reopened.LoadTripleStore("test-doc")
	if err != nil {
		t.Fatalf("LoadTripleStore after re-open failed: %v", err)
	}
	if tripleStore.Count() == 0 {
		t.Error("triple store is empty after re-open")
	}
}
