package store

import (
	"fmt"
	"sync"
)

// IndexStats contains statistics about the triple store for query optimization.
type IndexStats struct {
	TotalTriples     int            `json:"total_triples"`
	UniqueSubjects   int            `json:"unique_subjects"`
	UniquePredicates int            `json:"unique_predicates"`
	UniqueObjects    int            `json:"unique_objects"`
	PredicateCounts  map[string]int `json:"predicate_counts"`
	SubjectCounts    map[string]int `json:"subject_counts"`
	ObjectCounts     map[string]int `json:"object_counts"`
}

// TripleStore is an in-memory RDF triple store with multiple indexes.
// It provides efficient lookups via three indexes:
//   - SPO: Subject -> Predicate -> Object (find facts about a subject)
//   - POS: Predicate -> Object -> Subject (find subjects with property=value)
//   - OSP: Object -> Subject -> Predicate (find subjects pointing to object)
type TripleStore struct {
	mu sync.RWMutex

	// SPO index: Subject -> Predicate -> Object -> exists
	spo map[string]map[string]map[string]bool

	// POS index: Predicate -> Object -> Subject -> exists
	pos map[string]map[string]map[string]bool

	// OSP index: Object -> Subject -> Predicate -> exists
	osp map[string]map[string]map[string]bool

	// Triple count
	count int

	// Statistics for query optimization
	predicateCounts map[string]int
	subjectCounts   map[string]int
	objectCounts    map[string]int
}

// NewTripleStore creates a new in-memory triple store with all indexes initialized.
func NewTripleStore() *TripleStore {
	return &TripleStore{
		spo:             make(map[string]map[string]map[string]bool),
		pos:             make(map[string]map[string]map[string]bool),
		osp:             make(map[string]map[string]map[string]bool),
		count:           0,
		predicateCounts: make(map[string]int),
		subjectCounts:   make(map[string]int),
		objectCounts:    make(map[string]int),
	}
}

// Add inserts a triple into the store. Returns nil if successful or if the
// triple already exists (idempotent operation).
func (ts *TripleStore) Add(subject, predicate, object string) error {
	if subject == "" || predicate == "" || object == "" {
		return fmt.Errorf("triple components cannot be empty")
	}

	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Check if triple already exists
	if ts.existsUnsafe(subject, predicate, object) {
		return nil // Already exists, idempotent
	}

	// Add to SPO index
	if ts.spo[subject] == nil {
		ts.spo[subject] = make(map[string]map[string]bool)
	}
	if ts.spo[subject][predicate] == nil {
		ts.spo[subject][predicate] = make(map[string]bool)
	}
	ts.spo[subject][predicate][object] = true

	// Add to POS index
	if ts.pos[predicate] == nil {
		ts.pos[predicate] = make(map[string]map[string]bool)
	}
	if ts.pos[predicate][object] == nil {
		ts.pos[predicate][object] = make(map[string]bool)
	}
	ts.pos[predicate][object][subject] = true

	// Add to OSP index
	if ts.osp[object] == nil {
		ts.osp[object] = make(map[string]map[string]bool)
	}
	if ts.osp[object][subject] == nil {
		ts.osp[object][subject] = make(map[string]bool)
	}
	ts.osp[object][subject][predicate] = true

	// Update statistics
	ts.predicateCounts[predicate]++
	ts.subjectCounts[subject]++
	ts.objectCounts[object]++
	ts.count++

	return nil
}

// AddTriple inserts a Triple struct into the store.
func (ts *TripleStore) AddTriple(triple Triple) error {
	return ts.Add(triple.Subject, triple.Predicate, triple.Object)
}

// BulkAdd inserts multiple triples efficiently. Holds the write lock for the
// entire operation to minimize lock contention.
func (ts *TripleStore) BulkAdd(triples []Triple) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	for _, triple := range triples {
		if triple.Subject == "" || triple.Predicate == "" || triple.Object == "" {
			continue // Skip invalid triples
		}

		// Check if triple already exists
		if ts.existsUnsafe(triple.Subject, triple.Predicate, triple.Object) {
			continue
		}

		subject := triple.Subject
		predicate := triple.Predicate
		object := triple.Object

		// Add to SPO index
		if ts.spo[subject] == nil {
			ts.spo[subject] = make(map[string]map[string]bool)
		}
		if ts.spo[subject][predicate] == nil {
			ts.spo[subject][predicate] = make(map[string]bool)
		}
		ts.spo[subject][predicate][object] = true

		// Add to POS index
		if ts.pos[predicate] == nil {
			ts.pos[predicate] = make(map[string]map[string]bool)
		}
		if ts.pos[predicate][object] == nil {
			ts.pos[predicate][object] = make(map[string]bool)
		}
		ts.pos[predicate][object][subject] = true

		// Add to OSP index
		if ts.osp[object] == nil {
			ts.osp[object] = make(map[string]map[string]bool)
		}
		if ts.osp[object][subject] == nil {
			ts.osp[object][subject] = make(map[string]bool)
		}
		ts.osp[object][subject][predicate] = true

		// Update statistics
		ts.predicateCounts[predicate]++
		ts.subjectCounts[subject]++
		ts.objectCounts[object]++
		ts.count++
	}

	return nil
}

// MergeFrom copies all triples from the source store into this store.
// Returns the number of new triples added (duplicates are skipped via idempotent Add).
func (ts *TripleStore) MergeFrom(source *TripleStore) int {
	sourceTriples := source.All()
	previousCount := ts.Count()
	_ = ts.BulkAdd(sourceTriples)
	return ts.Count() - previousCount
}

// Find queries triples matching the pattern. Use empty string "" for wildcards.
// Returns all matching triples.
func (ts *TripleStore) Find(subject, predicate, object string) []Triple {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	return ts.findUnsafe(subject, predicate, object)
}

// FindPattern queries using a TriplePattern.
func (ts *TripleStore) FindPattern(pattern TriplePattern) []Triple {
	return ts.Find(pattern.Subject, pattern.Predicate, pattern.Object)
}

// Exists checks if a specific triple exists in the store.
func (ts *TripleStore) Exists(subject, predicate, object string) bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	return ts.existsUnsafe(subject, predicate, object)
}

// Get retrieves all properties for a subject as a map of predicate -> []objects.
func (ts *TripleStore) Get(subject string) map[string][]string {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	result := make(map[string][]string)

	if pMap, ok := ts.spo[subject]; ok {
		for p, oMap := range pMap {
			objects := make([]string, 0, len(oMap))
			for o := range oMap {
				objects = append(objects, o)
			}
			result[p] = objects
		}
	}

	return result
}

// GetOne retrieves a single object value for a subject-predicate pair.
// Returns empty string if not found.
func (ts *TripleStore) GetOne(subject, predicate string) string {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if pMap, ok := ts.spo[subject]; ok {
		if oMap, ok := pMap[predicate]; ok {
			for o := range oMap {
				return o // Return first match
			}
		}
	}

	return ""
}

// Delete removes matching triples. Use "" for wildcards to delete multiple.
func (ts *TripleStore) Delete(subject, predicate, object string) int {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Find matching triples first
	matches := ts.findUnsafe(subject, predicate, object)

	// Delete each match
	for _, triple := range matches {
		ts.deleteTripleUnsafe(triple.Subject, triple.Predicate, triple.Object)
	}

	return len(matches)
}

// DeleteTriple removes a specific triple.
func (ts *TripleStore) DeleteTriple(triple Triple) bool {
	return ts.Delete(triple.Subject, triple.Predicate, triple.Object) > 0
}

// Clear removes all triples from the store.
func (ts *TripleStore) Clear() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.spo = make(map[string]map[string]map[string]bool)
	ts.pos = make(map[string]map[string]map[string]bool)
	ts.osp = make(map[string]map[string]map[string]bool)
	ts.count = 0
	ts.predicateCounts = make(map[string]int)
	ts.subjectCounts = make(map[string]int)
	ts.objectCounts = make(map[string]int)
}

// Count returns the total number of triples in the store.
func (ts *TripleStore) Count() int {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.count
}

// Subjects returns all unique subjects in the store.
func (ts *TripleStore) Subjects() []string {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	subjects := make([]string, 0, len(ts.spo))
	for s := range ts.spo {
		subjects = append(subjects, s)
	}
	return subjects
}

// Predicates returns all unique predicates in the store.
func (ts *TripleStore) Predicates() []string {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	predicates := make([]string, 0, len(ts.pos))
	for p := range ts.pos {
		predicates = append(predicates, p)
	}
	return predicates
}

// Objects returns all unique objects in the store.
func (ts *TripleStore) Objects() []string {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	objects := make([]string, 0, len(ts.osp))
	for o := range ts.osp {
		objects = append(objects, o)
	}
	return objects
}

// Stats returns statistics about the store for query optimization.
func (ts *TripleStore) Stats() IndexStats {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	// Create copies to prevent external modification
	predicateCounts := make(map[string]int, len(ts.predicateCounts))
	for k, v := range ts.predicateCounts {
		predicateCounts[k] = v
	}

	subjectCounts := make(map[string]int, len(ts.subjectCounts))
	for k, v := range ts.subjectCounts {
		subjectCounts[k] = v
	}

	objectCounts := make(map[string]int, len(ts.objectCounts))
	for k, v := range ts.objectCounts {
		objectCounts[k] = v
	}

	return IndexStats{
		TotalTriples:     ts.count,
		UniqueSubjects:   len(ts.spo),
		UniquePredicates: len(ts.pos),
		UniqueObjects:    len(ts.osp),
		PredicateCounts:  predicateCounts,
		SubjectCounts:    subjectCounts,
		ObjectCounts:     objectCounts,
	}
}

// String returns a string representation of the store statistics.
func (ts *TripleStore) String() string {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	return fmt.Sprintf("TripleStore{triples: %d, subjects: %d, predicates: %d, objects: %d}",
		ts.count, len(ts.spo), len(ts.pos), len(ts.osp))
}

// All returns all triples in the store.
func (ts *TripleStore) All() []Triple {
	return ts.Find("", "", "")
}

// existsUnsafe checks if a triple exists without locking.
func (ts *TripleStore) existsUnsafe(subject, predicate, object string) bool {
	if pMap, ok := ts.spo[subject]; ok {
		if oMap, ok := pMap[predicate]; ok {
			return oMap[object]
		}
	}
	return false
}

// findUnsafe finds triples without locking.
func (ts *TripleStore) findUnsafe(subject, predicate, object string) []Triple {
	var results []Triple

	// All wildcards - return all triples
	if subject == "" && predicate == "" && object == "" {
		for s, pMap := range ts.spo {
			for p, oMap := range pMap {
				for o := range oMap {
					results = append(results, Triple{Subject: s, Predicate: p, Object: o})
				}
			}
		}
		return results
	}

	// Use most specific index based on what's specified
	if subject != "" {
		// Use SPO index
		if pMap, ok := ts.spo[subject]; ok {
			if predicate != "" {
				// S and P specified
				if oMap, ok := pMap[predicate]; ok {
					if object != "" {
						// All specified - check existence
						if oMap[object] {
							results = append(results, Triple{Subject: subject, Predicate: predicate, Object: object})
						}
					} else {
						// S and P specified, O wildcard
						for o := range oMap {
							results = append(results, Triple{Subject: subject, Predicate: predicate, Object: o})
						}
					}
				}
			} else {
				// S specified, P wildcard
				for p, oMap := range pMap {
					if object != "" {
						// S and O specified, P wildcard
						if oMap[object] {
							results = append(results, Triple{Subject: subject, Predicate: p, Object: object})
						}
					} else {
						// S specified, P and O wildcards
						for o := range oMap {
							results = append(results, Triple{Subject: subject, Predicate: p, Object: o})
						}
					}
				}
			}
		}
	} else if predicate != "" {
		// Use POS index (no subject specified)
		if oMap, ok := ts.pos[predicate]; ok {
			if object != "" {
				// P and O specified, S wildcard
				if sMap, ok := oMap[object]; ok {
					for s := range sMap {
						results = append(results, Triple{Subject: s, Predicate: predicate, Object: object})
					}
				}
			} else {
				// P specified, S and O wildcards
				for o, sMap := range oMap {
					for s := range sMap {
						results = append(results, Triple{Subject: s, Predicate: predicate, Object: o})
					}
				}
			}
		}
	} else if object != "" {
		// Use OSP index (only O specified)
		if sMap, ok := ts.osp[object]; ok {
			for s, pMap := range sMap {
				for p := range pMap {
					results = append(results, Triple{Subject: s, Predicate: p, Object: object})
				}
			}
		}
	}

	return results
}

// deleteTripleUnsafe deletes a specific triple without locking.
func (ts *TripleStore) deleteTripleUnsafe(subject, predicate, object string) {
	// Check if exists first
	if !ts.existsUnsafe(subject, predicate, object) {
		return
	}

	// Remove from SPO index
	if pMap, ok := ts.spo[subject]; ok {
		if oMap, ok := pMap[predicate]; ok {
			delete(oMap, object)
			if len(oMap) == 0 {
				delete(pMap, predicate)
			}
		}
		if len(pMap) == 0 {
			delete(ts.spo, subject)
		}
	}

	// Remove from POS index
	if oMap, ok := ts.pos[predicate]; ok {
		if sMap, ok := oMap[object]; ok {
			delete(sMap, subject)
			if len(sMap) == 0 {
				delete(oMap, object)
			}
		}
		if len(oMap) == 0 {
			delete(ts.pos, predicate)
		}
	}

	// Remove from OSP index
	if sMap, ok := ts.osp[object]; ok {
		if pMap, ok := sMap[subject]; ok {
			delete(pMap, predicate)
			if len(pMap) == 0 {
				delete(sMap, subject)
			}
		}
		if len(sMap) == 0 {
			delete(ts.osp, object)
		}
	}

	// Update statistics
	ts.predicateCounts[predicate]--
	if ts.predicateCounts[predicate] <= 0 {
		delete(ts.predicateCounts, predicate)
	}
	ts.subjectCounts[subject]--
	if ts.subjectCounts[subject] <= 0 {
		delete(ts.subjectCounts, subject)
	}
	ts.objectCounts[object]--
	if ts.objectCounts[object] <= 0 {
		delete(ts.objectCounts, object)
	}

	ts.count--
}
