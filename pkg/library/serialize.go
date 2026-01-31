package library

import (
	"encoding/json"
	"fmt"

	"github.com/coolbeans/regula/pkg/store"
)

// SerializeTripleStore converts all triples in a TripleStore to a JSON byte slice.
func SerializeTripleStore(tripleStore *store.TripleStore) ([]byte, error) {
	if tripleStore == nil {
		return nil, fmt.Errorf("triple store is nil")
	}

	allTriples := tripleStore.All()
	serialized := make([]SerializedTriple, len(allTriples))
	for i, triple := range allTriples {
		serialized[i] = FromStoreTriple(triple)
	}

	return json.Marshal(serialized)
}

// DeserializeTripleStore creates a new TripleStore and populates it from a JSON byte slice.
func DeserializeTripleStore(data []byte) (*store.TripleStore, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	var serialized []SerializedTriple
	if err := json.Unmarshal(data, &serialized); err != nil {
		return nil, fmt.Errorf("failed to unmarshal triples: %w", err)
	}

	tripleStore := store.NewTripleStore()
	storeTriples := make([]store.Triple, len(serialized))
	for i, serializedTriple := range serialized {
		storeTriples[i] = serializedTriple.ToStoreTriple()
	}

	if err := tripleStore.BulkAdd(storeTriples); err != nil {
		return nil, fmt.Errorf("failed to bulk add triples: %w", err)
	}

	return tripleStore, nil
}
