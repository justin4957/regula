package library

import (
	"testing"

	"github.com/coolbeans/regula/pkg/store"
)

func TestSerializeDeserializeRoundTrip(t *testing.T) {
	original := store.NewTripleStore()
	original.Add("http://example.org/art1", "rdf:type", "reg:Article")
	original.Add("http://example.org/art1", "reg:title", "Right to erasure")
	original.Add("http://example.org/art2", "rdf:type", "reg:Article")
	original.Add("http://example.org/art2", "reg:references", "http://example.org/art1")

	data, err := SerializeTripleStore(original)
	if err != nil {
		t.Fatalf("SerializeTripleStore failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("serialized data is empty")
	}

	restored, err := DeserializeTripleStore(data)
	if err != nil {
		t.Fatalf("DeserializeTripleStore failed: %v", err)
	}

	if restored.Count() != original.Count() {
		t.Errorf("triple count mismatch: got %d, want %d", restored.Count(), original.Count())
	}

	// Verify specific triples survived the round trip
	if !restored.Exists("http://example.org/art1", "rdf:type", "reg:Article") {
		t.Error("missing triple: art1 rdf:type reg:Article")
	}
	if !restored.Exists("http://example.org/art1", "reg:title", "Right to erasure") {
		t.Error("missing triple: art1 reg:title 'Right to erasure'")
	}
	if !restored.Exists("http://example.org/art2", "reg:references", "http://example.org/art1") {
		t.Error("missing triple: art2 reg:references art1")
	}
}

func TestSerializeEmptyStore(t *testing.T) {
	emptyStore := store.NewTripleStore()

	data, err := SerializeTripleStore(emptyStore)
	if err != nil {
		t.Fatalf("SerializeTripleStore failed for empty store: %v", err)
	}

	restored, err := DeserializeTripleStore(data)
	if err != nil {
		t.Fatalf("DeserializeTripleStore failed for empty store: %v", err)
	}

	if restored.Count() != 0 {
		t.Errorf("expected empty store, got %d triples", restored.Count())
	}
}

func TestSerializeNilStore(t *testing.T) {
	_, err := SerializeTripleStore(nil)
	if err == nil {
		t.Error("expected error for nil store")
	}
}

func TestDeserializeEmptyData(t *testing.T) {
	_, err := DeserializeTripleStore([]byte{})
	if err == nil {
		t.Error("expected error for empty data")
	}
}

func TestDeserializeMalformedJSON(t *testing.T) {
	_, err := DeserializeTripleStore([]byte("not json"))
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestFromStoreTriple(t *testing.T) {
	original := store.NewTriple("s", "p", "o")
	serialized := FromStoreTriple(original)

	if serialized.Subject != "s" || serialized.Predicate != "p" || serialized.Object != "o" {
		t.Errorf("FromStoreTriple mismatch: got %+v", serialized)
	}
}

func TestSerializedTripleToStoreTriple(t *testing.T) {
	serialized := SerializedTriple{Subject: "s", Predicate: "p", Object: "o"}
	restored := serialized.ToStoreTriple()

	if restored.Subject != "s" || restored.Predicate != "p" || restored.Object != "o" {
		t.Errorf("ToStoreTriple mismatch: got %+v", restored)
	}
}
