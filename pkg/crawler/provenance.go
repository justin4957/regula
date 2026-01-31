package crawler

import (
	"fmt"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// ProvenanceTracker records crawl provenance as RDF triples, tracking
// how documents were discovered, from which sources, and at what depth.
type ProvenanceTracker struct {
	tripleStore *store.TripleStore
	baseURI     string
}

// NewProvenanceTracker creates a ProvenanceTracker with a dedicated triple store.
func NewProvenanceTracker(baseURI string) *ProvenanceTracker {
	return &ProvenanceTracker{
		tripleStore: store.NewTripleStore(),
		baseURI:     baseURI,
	}
}

// RecordDiscovery records that a document was discovered through a cross-reference
// from another document during crawling.
func (tracker *ProvenanceTracker) RecordDiscovery(sourceDocumentID, discoveredDocumentID, citation string, depth int) {
	discoveredURI := tracker.documentURI(discoveredDocumentID)
	sourceURI := tracker.documentURI(sourceDocumentID)

	triples := []store.Triple{
		{Subject: discoveredURI, Predicate: "rdf:type", Object: store.ClassCrawledDocument},
		{Subject: discoveredURI, Predicate: store.PropCrawlDiscoveredBy, Object: sourceURI},
		{Subject: discoveredURI, Predicate: store.PropCrawlCitation, Object: citation},
		{Subject: discoveredURI, Predicate: store.PropCrawlDepth, Object: fmt.Sprintf("%d", depth)},
	}

	_ = tracker.tripleStore.BulkAdd(triples)
}

// RecordFetch records that a document was successfully fetched from a URL.
func (tracker *ProvenanceTracker) RecordFetch(documentID, fetchURL string, fetchedAt time.Time) {
	documentURI := tracker.documentURI(documentID)

	triples := []store.Triple{
		{Subject: documentURI, Predicate: store.PropCrawlSource, Object: fetchURL},
		{Subject: documentURI, Predicate: store.PropCrawlFetchedAt, Object: fetchedAt.Format(time.RFC3339)},
		{Subject: documentURI, Predicate: store.PropCrawlStatus, Object: "ingested"},
	}

	_ = tracker.tripleStore.BulkAdd(triples)
}

// RecordFailure records that a crawl attempt for a citation/URL failed.
func (tracker *ProvenanceTracker) RecordFailure(citation, fetchURL, reason string) {
	failureURI := fmt.Sprintf("%scrawl:failure:%s", tracker.baseURI, sanitizeForURI(citation))

	triples := []store.Triple{
		{Subject: failureURI, Predicate: "rdf:type", Object: store.ClassCrawledDocument},
		{Subject: failureURI, Predicate: store.PropCrawlCitation, Object: citation},
		{Subject: failureURI, Predicate: store.PropCrawlStatus, Object: "failed"},
	}

	if fetchURL != "" {
		triples = append(triples, store.Triple{
			Subject: failureURI, Predicate: store.PropCrawlSource, Object: fetchURL,
		})
	}

	_ = tracker.tripleStore.BulkAdd(triples)
}

// TripleStore returns the underlying triple store containing all provenance triples.
func (tracker *ProvenanceTracker) TripleStore() *store.TripleStore {
	return tracker.tripleStore
}

// DiscoveryCount returns the number of unique discovered documents.
func (tracker *ProvenanceTracker) DiscoveryCount() int {
	discoveredTriples := tracker.tripleStore.Find("", "rdf:type", store.ClassCrawledDocument)
	return len(discoveredTriples)
}

// GetDiscoveryChain returns the chain of document IDs that led to the discovery
// of the given document, ordered from seed to target.
func (tracker *ProvenanceTracker) GetDiscoveryChain(documentID string) []string {
	documentURI := tracker.documentURI(documentID)
	var chain []string

	currentURI := documentURI
	visited := make(map[string]bool)

	for {
		if visited[currentURI] {
			break // prevent cycles
		}
		visited[currentURI] = true

		discoveredByTriples := tracker.tripleStore.Find(currentURI, store.PropCrawlDiscoveredBy, "")
		if len(discoveredByTriples) == 0 {
			break
		}

		parentURI := discoveredByTriples[0].Object
		chain = append([]string{tracker.uriToDocID(parentURI)}, chain...)
		currentURI = parentURI
	}

	return chain
}

// documentURI builds a full URI for a document ID.
func (tracker *ProvenanceTracker) documentURI(documentID string) string {
	return tracker.baseURI + documentID
}

// uriToDocID extracts a document ID from a full URI.
func (tracker *ProvenanceTracker) uriToDocID(uri string) string {
	if len(uri) > len(tracker.baseURI) {
		return uri[len(tracker.baseURI):]
	}
	return uri
}

// sanitizeForURI replaces non-URI-safe characters with underscores.
func sanitizeForURI(input string) string {
	var builder []byte
	for _, character := range []byte(input) {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '-' || character == '_' || character == '.' {
			builder = append(builder, character)
		} else {
			builder = append(builder, '_')
		}
	}
	return string(builder)
}
