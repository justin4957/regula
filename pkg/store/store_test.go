package store

import (
	"fmt"
	"sync"
	"testing"
)

func TestNewTripleStore(t *testing.T) {
	store := NewTripleStore()

	if store == nil {
		t.Fatal("NewTripleStore returned nil")
	}

	if store.Count() != 0 {
		t.Errorf("New store should have 0 triples, got %d", store.Count())
	}
}

func TestTripleStore_Add(t *testing.T) {
	store := NewTripleStore()

	// Add a triple
	err := store.Add("GDPR:Art1", "rdf:type", "reg:Article")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if store.Count() != 1 {
		t.Errorf("Expected 1 triple, got %d", store.Count())
	}

	// Add same triple again (idempotent)
	err = store.Add("GDPR:Art1", "rdf:type", "reg:Article")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if store.Count() != 1 {
		t.Errorf("Expected 1 triple after duplicate add, got %d", store.Count())
	}

	// Add different triple
	err = store.Add("GDPR:Art1", "reg:title", "Subject-matter and objectives")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if store.Count() != 2 {
		t.Errorf("Expected 2 triples, got %d", store.Count())
	}
}

func TestTripleStore_AddTriple(t *testing.T) {
	store := NewTripleStore()

	triple := NewTriple("GDPR:Art17", "rdf:type", "reg:Article")
	err := store.AddTriple(triple)
	if err != nil {
		t.Fatalf("AddTriple failed: %v", err)
	}

	if store.Count() != 1 {
		t.Errorf("Expected 1 triple, got %d", store.Count())
	}
}

func TestTripleStore_Add_InvalidTriple(t *testing.T) {
	store := NewTripleStore()

	// Empty subject
	err := store.Add("", "rdf:type", "reg:Article")
	if err == nil {
		t.Error("Expected error for empty subject")
	}

	// Empty predicate
	err = store.Add("GDPR:Art1", "", "reg:Article")
	if err == nil {
		t.Error("Expected error for empty predicate")
	}

	// Empty object
	err = store.Add("GDPR:Art1", "rdf:type", "")
	if err == nil {
		t.Error("Expected error for empty object")
	}

	if store.Count() != 0 {
		t.Errorf("Store should be empty after invalid adds, got %d", store.Count())
	}
}

func TestTripleStore_BulkAdd(t *testing.T) {
	store := NewTripleStore()

	triples := []Triple{
		NewTriple("GDPR:Art1", "rdf:type", "reg:Article"),
		NewTriple("GDPR:Art1", "reg:number", "1"),
		NewTriple("GDPR:Art1", "reg:title", "Subject-matter and objectives"),
		NewTriple("GDPR:Art2", "rdf:type", "reg:Article"),
		NewTriple("GDPR:Art2", "reg:number", "2"),
	}

	err := store.BulkAdd(triples)
	if err != nil {
		t.Fatalf("BulkAdd failed: %v", err)
	}

	if store.Count() != 5 {
		t.Errorf("Expected 5 triples, got %d", store.Count())
	}
}

func TestTripleStore_BulkAdd_WithDuplicates(t *testing.T) {
	store := NewTripleStore()

	triples := []Triple{
		NewTriple("GDPR:Art1", "rdf:type", "reg:Article"),
		NewTriple("GDPR:Art1", "rdf:type", "reg:Article"), // Duplicate
		NewTriple("GDPR:Art2", "rdf:type", "reg:Article"),
	}

	err := store.BulkAdd(triples)
	if err != nil {
		t.Fatalf("BulkAdd failed: %v", err)
	}

	if store.Count() != 2 {
		t.Errorf("Expected 2 triples after dedup, got %d", store.Count())
	}
}

func TestTripleStore_Find_AllWildcard(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	// Find all
	results := store.Find("", "", "")

	if len(results) != store.Count() {
		t.Errorf("Expected %d results, got %d", store.Count(), len(results))
	}
}

func TestTripleStore_Find_SPO_Index(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	// Find by subject
	results := store.Find("GDPR:Art1", "", "")
	if len(results) != 3 {
		t.Errorf("Expected 3 results for subject GDPR:Art1, got %d", len(results))
	}

	// Find by subject and predicate
	results = store.Find("GDPR:Art1", "rdf:type", "")
	if len(results) != 1 {
		t.Errorf("Expected 1 result for S+P, got %d", len(results))
	}

	// Find exact triple
	results = store.Find("GDPR:Art1", "rdf:type", "reg:Article")
	if len(results) != 1 {
		t.Errorf("Expected 1 result for exact match, got %d", len(results))
	}

	// Find by subject and object (P wildcard)
	results = store.Find("GDPR:Art1", "", "reg:Article")
	if len(results) != 1 {
		t.Errorf("Expected 1 result for S+O, got %d", len(results))
	}
}

func TestTripleStore_Find_POS_Index(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	// Find by predicate
	results := store.Find("", "rdf:type", "")
	if len(results) != 3 {
		t.Errorf("Expected 3 results for predicate rdf:type, got %d", len(results))
	}

	// Find by predicate and object
	results = store.Find("", "rdf:type", "reg:Article")
	if len(results) != 3 {
		t.Errorf("Expected 3 results for P+O, got %d", len(results))
	}
}

func TestTripleStore_Find_OSP_Index(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	// Find by object
	results := store.Find("", "", "reg:Article")
	if len(results) != 3 {
		t.Errorf("Expected 3 results for object reg:Article, got %d", len(results))
	}
}

func TestTripleStore_Find_NoMatch(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	results := store.Find("NonExistent", "", "")
	if len(results) != 0 {
		t.Errorf("Expected 0 results for non-existent subject, got %d", len(results))
	}
}

func TestTripleStore_FindPattern(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	pattern := NewTriplePattern("GDPR:Art1", "rdf:type", "")
	results := store.FindPattern(pattern)

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestTripleStore_Exists(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	// Should exist
	if !store.Exists("GDPR:Art1", "rdf:type", "reg:Article") {
		t.Error("Expected triple to exist")
	}

	// Should not exist
	if store.Exists("GDPR:Art1", "rdf:type", "reg:Chapter") {
		t.Error("Expected triple to not exist")
	}
}

func TestTripleStore_Get(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	props := store.Get("GDPR:Art1")

	if len(props) != 3 {
		t.Errorf("Expected 3 properties, got %d", len(props))
	}

	// Check specific property
	types := props["rdf:type"]
	if len(types) != 1 || types[0] != "reg:Article" {
		t.Errorf("Unexpected rdf:type value: %v", types)
	}
}

func TestTripleStore_GetOne(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	title := store.GetOne("GDPR:Art1", "reg:title")
	if title != "Subject-matter and objectives" {
		t.Errorf("Unexpected title: %s", title)
	}

	// Non-existent
	nothing := store.GetOne("GDPR:Art1", "nonexistent")
	if nothing != "" {
		t.Errorf("Expected empty string for non-existent, got %s", nothing)
	}
}

func TestTripleStore_Delete(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	initialCount := store.Count()

	// Delete specific triple
	deleted := store.Delete("GDPR:Art1", "rdf:type", "reg:Article")
	if deleted != 1 {
		t.Errorf("Expected 1 deletion, got %d", deleted)
	}

	if store.Count() != initialCount-1 {
		t.Errorf("Expected %d triples after delete, got %d", initialCount-1, store.Count())
	}

	// Verify it's gone
	if store.Exists("GDPR:Art1", "rdf:type", "reg:Article") {
		t.Error("Triple should not exist after deletion")
	}
}

func TestTripleStore_Delete_Wildcard(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	// Delete all triples for subject
	deleted := store.Delete("GDPR:Art1", "", "")
	if deleted != 3 {
		t.Errorf("Expected 3 deletions, got %d", deleted)
	}

	// Verify subject is gone
	results := store.Find("GDPR:Art1", "", "")
	if len(results) != 0 {
		t.Errorf("Subject should have no triples, got %d", len(results))
	}
}

func TestTripleStore_DeleteTriple(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	triple := NewTriple("GDPR:Art1", "rdf:type", "reg:Article")
	deleted := store.DeleteTriple(triple)

	if !deleted {
		t.Error("Expected deletion to succeed")
	}

	if store.Exists("GDPR:Art1", "rdf:type", "reg:Article") {
		t.Error("Triple should not exist after deletion")
	}
}

func TestTripleStore_Clear(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	if store.Count() == 0 {
		t.Fatal("Store should not be empty before clear")
	}

	store.Clear()

	if store.Count() != 0 {
		t.Errorf("Store should be empty after clear, got %d", store.Count())
	}

	if len(store.Subjects()) != 0 {
		t.Error("Subjects should be empty after clear")
	}
}

func TestTripleStore_Subjects(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	subjects := store.Subjects()

	if len(subjects) != 3 {
		t.Errorf("Expected 3 unique subjects, got %d", len(subjects))
	}

	// Check all expected subjects are present
	subjectMap := make(map[string]bool)
	for _, s := range subjects {
		subjectMap[s] = true
	}

	for _, expected := range []string{"GDPR:Art1", "GDPR:Art2", "GDPR:Art17"} {
		if !subjectMap[expected] {
			t.Errorf("Expected subject %s not found", expected)
		}
	}
}

func TestTripleStore_Predicates(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	predicates := store.Predicates()

	if len(predicates) != 3 {
		t.Errorf("Expected 3 unique predicates, got %d", len(predicates))
	}
}

func TestTripleStore_Objects(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	objects := store.Objects()

	// Should have: reg:Article, 1, Subject-matter..., 2, 17
	if len(objects) < 3 {
		t.Errorf("Expected at least 3 unique objects, got %d", len(objects))
	}
}

func TestTripleStore_Stats(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	stats := store.Stats()

	if stats.TotalTriples != store.Count() {
		t.Errorf("Stats total mismatch: %d vs %d", stats.TotalTriples, store.Count())
	}

	if stats.UniqueSubjects != 3 {
		t.Errorf("Expected 3 unique subjects, got %d", stats.UniqueSubjects)
	}

	// rdf:type should have count of 3
	if stats.PredicateCounts["rdf:type"] != 3 {
		t.Errorf("Expected 3 for rdf:type, got %d", stats.PredicateCounts["rdf:type"])
	}
}

func TestTripleStore_String(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	s := store.String()
	expected := fmt.Sprintf("TripleStore{triples: %d, subjects: 3, predicates: 3, objects:", store.Count())

	if len(s) < len(expected) {
		t.Errorf("String representation too short: %s", s)
	}
}

func TestTripleStore_All(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	all := store.All()

	if len(all) != store.Count() {
		t.Errorf("All() returned %d triples, expected %d", len(all), store.Count())
	}
}

func TestTripleStore_ConcurrentReads(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	var wg sync.WaitGroup
	errChan := make(chan error, 100)

	// Multiple concurrent readers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results := store.Find("GDPR:Art1", "", "")
			if len(results) != 3 {
				errChan <- fmt.Errorf("expected 3 results, got %d", len(results))
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Error(err)
	}
}

func TestTripleStore_ConcurrentWrites(t *testing.T) {
	store := NewTripleStore()

	var wg sync.WaitGroup

	// Multiple concurrent writers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			subject := fmt.Sprintf("test:subject%d", n)
			store.Add(subject, "rdf:type", "test:Thing")
		}(i)
	}

	wg.Wait()

	if store.Count() != 100 {
		t.Errorf("Expected 100 triples, got %d", store.Count())
	}
}

func TestTripleStore_ConcurrentReadWrite(t *testing.T) {
	store := NewTripleStore()
	populateTestStore(store)

	var wg sync.WaitGroup

	// Writers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			subject := fmt.Sprintf("concurrent:subject%d", n)
			store.Add(subject, "rdf:type", "test:Thing")
		}(i)
	}

	// Readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.Find("", "rdf:type", "")
		}()
	}

	wg.Wait()

	// Should have original + 50 new triples
	if store.Count() < 50 {
		t.Errorf("Expected at least 50 triples, got %d", store.Count())
	}
}

func TestTripleStore_IndexConsistency(t *testing.T) {
	store := NewTripleStore()

	// Add some triples
	store.Add("s1", "p1", "o1")
	store.Add("s1", "p2", "o2")
	store.Add("s2", "p1", "o1")

	// Verify all indexes are consistent
	// SPO query
	spo := store.Find("s1", "", "")
	if len(spo) != 2 {
		t.Errorf("SPO index: expected 2, got %d", len(spo))
	}

	// POS query
	pos := store.Find("", "p1", "")
	if len(pos) != 2 {
		t.Errorf("POS index: expected 2, got %d", len(pos))
	}

	// OSP query
	osp := store.Find("", "", "o1")
	if len(osp) != 2 {
		t.Errorf("OSP index: expected 2, got %d", len(osp))
	}

	// Delete and verify consistency
	store.Delete("s1", "p1", "o1")

	spo = store.Find("s1", "", "")
	if len(spo) != 1 {
		t.Errorf("After delete SPO: expected 1, got %d", len(spo))
	}

	pos = store.Find("", "p1", "")
	if len(pos) != 1 {
		t.Errorf("After delete POS: expected 1, got %d", len(pos))
	}

	osp = store.Find("", "", "o1")
	if len(osp) != 1 {
		t.Errorf("After delete OSP: expected 1, got %d", len(osp))
	}
}

// Benchmarks

func BenchmarkTripleStore_Add(b *testing.B) {
	store := NewTripleStore()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		subject := fmt.Sprintf("subject%d", i)
		store.Add(subject, "rdf:type", "test:Thing")
	}
}

func BenchmarkTripleStore_BulkAdd(b *testing.B) {
	// Prepare triples
	triples := make([]Triple, 1000)
	for i := 0; i < 1000; i++ {
		triples[i] = NewTriple(
			fmt.Sprintf("subject%d", i),
			"rdf:type",
			"test:Thing",
		)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store := NewTripleStore()
		store.BulkAdd(triples)
	}
}

func BenchmarkTripleStore_Find_SPO(b *testing.B) {
	store := NewTripleStore()
	populateLargeStore(store, 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Find("subject5000", "", "")
	}
}

func BenchmarkTripleStore_Find_POS(b *testing.B) {
	store := NewTripleStore()
	populateLargeStore(store, 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Find("", "rdf:type", "test:Thing")
	}
}

func BenchmarkTripleStore_Find_OSP(b *testing.B) {
	store := NewTripleStore()
	populateLargeStore(store, 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Find("", "", "test:Thing")
	}
}

func BenchmarkTripleStore_Exists(b *testing.B) {
	store := NewTripleStore()
	populateLargeStore(store, 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Exists("subject5000", "rdf:type", "test:Thing")
	}
}

// Helper functions

func populateTestStore(store *TripleStore) {
	triples := []Triple{
		// Article 1
		NewTriple("GDPR:Art1", "rdf:type", "reg:Article"),
		NewTriple("GDPR:Art1", "reg:number", "1"),
		NewTriple("GDPR:Art1", "reg:title", "Subject-matter and objectives"),
		// Article 2
		NewTriple("GDPR:Art2", "rdf:type", "reg:Article"),
		NewTriple("GDPR:Art2", "reg:number", "2"),
		// Article 17
		NewTriple("GDPR:Art17", "rdf:type", "reg:Article"),
		NewTriple("GDPR:Art17", "reg:number", "17"),
	}
	store.BulkAdd(triples)
}

func populateLargeStore(store *TripleStore, n int) {
	triples := make([]Triple, n)
	for i := 0; i < n; i++ {
		triples[i] = NewTriple(
			fmt.Sprintf("subject%d", i),
			"rdf:type",
			"test:Thing",
		)
	}
	store.BulkAdd(triples)
}
