package store

import (
	"testing"
	"time"
)

func TestTemporalStore_NewTemporalStore(t *testing.T) {
	ts := NewTemporalStore()

	if ts == nil {
		t.Fatal("Expected non-nil temporal store")
	}
	if ts.TripleStore == nil {
		t.Error("Expected non-nil base triple store")
	}
	if ts.graphs == nil {
		t.Error("Expected non-nil graphs map")
	}
	if ts.versionIndex == nil {
		t.Error("Expected non-nil version index")
	}
}

func TestTemporalStore_CreateGraph(t *testing.T) {
	ts := NewTemporalStore()

	graph := ts.CreateGraph("v1")
	if graph == nil {
		t.Fatal("Expected non-nil graph")
	}

	// Creating same graph again should return existing
	graph2 := ts.CreateGraph("v1")
	if graph != graph2 {
		t.Error("Expected same graph instance for same name")
	}
}

func TestTemporalStore_GetGraph(t *testing.T) {
	ts := NewTemporalStore()

	// Non-existent graph
	_, ok := ts.GetGraph("nonexistent")
	if ok {
		t.Error("Expected false for non-existent graph")
	}

	// Create and retrieve
	ts.CreateGraph("v1")
	graph, ok := ts.GetGraph("v1")
	if !ok {
		t.Error("Expected true for existing graph")
	}
	if graph == nil {
		t.Error("Expected non-nil graph")
	}
}

func TestTemporalStore_ListGraphs(t *testing.T) {
	ts := NewTemporalStore()

	// Initially empty
	names := ts.ListGraphs()
	if len(names) != 0 {
		t.Errorf("Expected 0 graphs, got %d", len(names))
	}

	// Create some graphs
	ts.CreateGraph("v1")
	ts.CreateGraph("v2")
	ts.CreateGraph("draft")

	names = ts.ListGraphs()
	if len(names) != 3 {
		t.Errorf("Expected 3 graphs, got %d", len(names))
	}

	// Should be sorted
	expected := []string{"draft", "v1", "v2"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("Expected %s at position %d, got %s", expected[i], i, name)
		}
	}
}

func TestTemporalStore_DeleteGraph(t *testing.T) {
	ts := NewTemporalStore()

	// Delete non-existent
	if ts.DeleteGraph("nonexistent") {
		t.Error("Expected false for deleting non-existent graph")
	}

	// Create and delete
	ts.CreateGraph("v1")
	if !ts.DeleteGraph("v1") {
		t.Error("Expected true for deleting existing graph")
	}

	// Verify deleted
	_, ok := ts.GetGraph("v1")
	if ok {
		t.Error("Graph should be deleted")
	}
}

func TestTemporalStore_AddVersioned(t *testing.T) {
	ts := NewTemporalStore()

	err := ts.AddVersioned("subject1", "predicate1", "object1", "v1")
	if err != nil {
		t.Fatalf("AddVersioned failed: %v", err)
	}

	// Verify triple is in the graph
	graph, ok := ts.GetGraph("v1")
	if !ok {
		t.Fatal("Expected graph to be created")
	}

	triples := graph.Find("subject1", "", "")
	if len(triples) != 1 {
		t.Errorf("Expected 1 triple, got %d", len(triples))
	}
}

func TestTemporalStore_AddVersion(t *testing.T) {
	ts := NewTemporalStore()

	canonicalURI := "https://example.org/doc/article17"
	version := VersionInfo{
		URI:       "https://example.org/doc/article17:v1",
		Version:   "1.0",
		ValidFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:    "active",
	}

	err := ts.AddVersion(canonicalURI, version)
	if err != nil {
		t.Fatalf("AddVersion failed: %v", err)
	}

	// Verify version is indexed
	history, err := ts.GetVersionHistory(canonicalURI)
	if err != nil {
		t.Fatalf("GetVersionHistory failed: %v", err)
	}
	if len(history.Versions) != 1 {
		t.Errorf("Expected 1 version, got %d", len(history.Versions))
	}

	// Verify triples were added
	triples := ts.TripleStore.Find(version.URI, PropVersionOf, "")
	if len(triples) != 1 {
		t.Errorf("Expected 1 versionOf triple, got %d", len(triples))
	}
}

func TestTemporalStore_AddVersion_Multiple(t *testing.T) {
	ts := NewTemporalStore()

	canonicalURI := "https://example.org/doc/article17"

	// Add multiple versions
	v1 := VersionInfo{
		URI:       canonicalURI + ":v1",
		Version:   "1.0",
		ValidFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:    "superseded",
	}
	v2 := VersionInfo{
		URI:       canonicalURI + ":v2",
		Version:   "2.0",
		ValidFrom: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		Status:    "active",
	}

	ts.AddVersion(canonicalURI, v1)
	ts.AddVersion(canonicalURI, v2)

	history, _ := ts.GetVersionHistory(canonicalURI)
	if len(history.Versions) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(history.Versions))
	}

	// Versions should be sorted by ValidFrom
	if history.Versions[0].Version != "1.0" {
		t.Error("Expected v1 to be first (sorted by ValidFrom)")
	}
	if history.Versions[1].Version != "2.0" {
		t.Error("Expected v2 to be second (sorted by ValidFrom)")
	}
}

func TestTemporalStore_SetCurrentVersion(t *testing.T) {
	ts := NewTemporalStore()

	canonicalURI := "https://example.org/doc/article17"
	versionURI := canonicalURI + ":v1"

	// Add version first
	ts.AddVersion(canonicalURI, VersionInfo{
		URI:       versionURI,
		Version:   "1.0",
		ValidFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:    "draft",
	})

	// Set as current
	err := ts.SetCurrentVersion(canonicalURI, versionURI)
	if err != nil {
		t.Fatalf("SetCurrentVersion failed: %v", err)
	}

	// Verify triple
	triples := ts.TripleStore.Find(canonicalURI, PropCurrentVersion, "")
	if len(triples) != 1 {
		t.Errorf("Expected 1 currentVersion triple, got %d", len(triples))
	}
	if triples[0].Object != versionURI {
		t.Errorf("Expected object %s, got %s", versionURI, triples[0].Object)
	}

	// Verify history shows current
	history, _ := ts.GetVersionHistory(canonicalURI)
	if history.CurrentVersion == nil {
		t.Error("Expected current version to be set")
	} else if history.CurrentVersion.URI != versionURI {
		t.Errorf("Expected current version URI %s, got %s", versionURI, history.CurrentVersion.URI)
	}
}

func TestTemporalStore_GetVersionAtTime(t *testing.T) {
	ts := NewTemporalStore()

	canonicalURI := "https://example.org/doc/article17"

	// Add versions at different times
	v1 := VersionInfo{
		URI:        canonicalURI + ":v1",
		Version:    "1.0",
		ValidFrom:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		ValidUntil: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		Status:     "superseded",
	}
	v2 := VersionInfo{
		URI:       canonicalURI + ":v2",
		Version:   "2.0",
		ValidFrom: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		Status:    "active",
	}

	ts.AddVersion(canonicalURI, v1)
	ts.AddVersion(canonicalURI, v2)

	tests := []struct {
		name           string
		asOf           time.Time
		expectedVersion string
		expectError    bool
	}{
		{
			name:           "before v1",
			asOf:           time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
			expectedVersion: "",
			expectError:    true,
		},
		{
			name:           "during v1 validity",
			asOf:           time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			expectedVersion: "1.0",
			expectError:    false,
		},
		{
			name:           "during v2 validity",
			asOf:           time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC),
			expectedVersion: "2.0",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := ts.GetVersionAtTime(canonicalURI, tt.asOf)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if version.Version != tt.expectedVersion {
					t.Errorf("Expected version %s, got %s", tt.expectedVersion, version.Version)
				}
			}
		})
	}
}

func TestTemporalStore_QueryAtTime(t *testing.T) {
	ts := NewTemporalStore()

	canonicalURI := "https://example.org/doc/article17"
	v1URI := canonicalURI + ":v1"

	// Add version with triples
	ts.AddVersion(canonicalURI, VersionInfo{
		URI:       v1URI,
		Version:   "1.0",
		ValidFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		GraphName: "v1",
		Status:    "active",
	})

	// Add triples to the version's graph
	ts.AddVersioned(v1URI, PropText, "Original text", "v1")

	// Query at valid time
	result, err := ts.QueryAtTime(v1URI, PropText, "", time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("QueryAtTime failed: %v", err)
	}

	if len(result.Triples) != 1 {
		t.Errorf("Expected 1 triple, got %d", len(result.Triples))
	}
}

func TestTemporalStore_GetLatestVersion(t *testing.T) {
	ts := NewTemporalStore()

	canonicalURI := "https://example.org/doc/article17"

	// Add versions
	ts.AddVersion(canonicalURI, VersionInfo{
		URI:       canonicalURI + ":v1",
		Version:   "1.0",
		ValidFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	ts.AddVersion(canonicalURI, VersionInfo{
		URI:       canonicalURI + ":v2",
		Version:   "2.0",
		ValidFrom: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
	})

	latest, err := ts.GetLatestVersion(canonicalURI)
	if err != nil {
		t.Fatalf("GetLatestVersion failed: %v", err)
	}
	if latest.Version != "2.0" {
		t.Errorf("Expected version 2.0, got %s", latest.Version)
	}
}

func TestTemporalStore_GetActiveVersions(t *testing.T) {
	ts := NewTemporalStore()

	// Add active version
	ts.AddVersion("https://example.org/doc1", VersionInfo{
		URI:       "https://example.org/doc1:v1",
		Version:   "1.0",
		ValidFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:    "active",
	})

	// Add superseded version
	ts.AddVersion("https://example.org/doc2", VersionInfo{
		URI:       "https://example.org/doc2:v1",
		Version:   "1.0",
		ValidFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:    "superseded",
	})

	active := ts.GetActiveVersions()
	if len(active) != 1 {
		t.Errorf("Expected 1 active version, got %d", len(active))
	}
}

func TestTemporalStore_LinkVersionToMeeting(t *testing.T) {
	ts := NewTemporalStore()

	canonicalURI := "https://example.org/doc/article17"
	versionURI := canonicalURI + ":v1"
	meetingURI := "https://example.org/meetings/wg-42"

	ts.AddVersion(canonicalURI, VersionInfo{
		URI:       versionURI,
		Version:   "1.0",
		ValidFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:    "active",
	})

	err := ts.LinkVersionToMeeting(versionURI, meetingURI)
	if err != nil {
		t.Fatalf("LinkVersionToMeeting failed: %v", err)
	}

	// Verify triple
	triples := ts.TripleStore.Find(versionURI, PropDecidedAt, "")
	if len(triples) != 1 {
		t.Errorf("Expected 1 decidedAt triple, got %d", len(triples))
	}
}

func TestTemporalStore_SupersedeVersion(t *testing.T) {
	ts := NewTemporalStore()

	canonicalURI := "https://example.org/doc/article17"
	v1URI := canonicalURI + ":v1"
	v2URI := canonicalURI + ":v2"

	ts.AddVersion(canonicalURI, VersionInfo{
		URI:       v1URI,
		Version:   "1.0",
		ValidFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:    "active",
	})
	ts.AddVersion(canonicalURI, VersionInfo{
		URI:       v2URI,
		Version:   "2.0",
		ValidFrom: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		Status:    "draft",
	})

	supersededAt := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	err := ts.SupersedeVersion(v1URI, v2URI, supersededAt)
	if err != nil {
		t.Fatalf("SupersedeVersion failed: %v", err)
	}

	// Verify supersedes triple
	triples := ts.TripleStore.Find(v2URI, PropSupersedes, "")
	if len(triples) != 1 {
		t.Errorf("Expected 1 supersedes triple, got %d", len(triples))
	}

	// Verify supersededBy triple
	triples = ts.TripleStore.Find(v1URI, PropSupersededBy, "")
	if len(triples) != 1 {
		t.Errorf("Expected 1 supersededBy triple, got %d", len(triples))
	}

	// Verify previous/next version triples
	triples = ts.TripleStore.Find(v2URI, PropPreviousVersion, "")
	if len(triples) != 1 {
		t.Errorf("Expected 1 previousVersion triple, got %d", len(triples))
	}
	triples = ts.TripleStore.Find(v1URI, PropNextVersion, "")
	if len(triples) != 1 {
		t.Errorf("Expected 1 nextVersion triple, got %d", len(triples))
	}
}

func TestTemporalStore_CompareVersions(t *testing.T) {
	ts := NewTemporalStore()

	v1URI := "https://example.org/doc:v1"
	v2URI := "https://example.org/doc:v2"

	// Add triples for v1
	ts.Add(v1URI, PropText, "Original text")
	ts.Add(v1URI, PropTitle, "Version 1 Title")
	ts.Add(v1URI, "reg:removedProp", "value")

	// Add triples for v2
	ts.Add(v2URI, PropText, "Updated text")
	ts.Add(v2URI, PropTitle, "Version 1 Title") // Same
	ts.Add(v2URI, "reg:newProp", "new value")

	diff, err := ts.CompareVersions(v1URI, v2URI)
	if err != nil {
		t.Fatalf("CompareVersions failed: %v", err)
	}

	if len(diff.Added) != 1 {
		t.Errorf("Expected 1 added, got %d", len(diff.Added))
	}
	if len(diff.Removed) != 1 {
		t.Errorf("Expected 1 removed, got %d", len(diff.Removed))
	}
	if len(diff.Modified) != 1 {
		t.Errorf("Expected 1 modified, got %d", len(diff.Modified))
	}

	// Check modified
	if len(diff.Modified) > 0 {
		if diff.Modified[0].Predicate != PropText {
			t.Errorf("Expected modified predicate %s, got %s", PropText, diff.Modified[0].Predicate)
		}
		if diff.Modified[0].OldObject != "Original text" {
			t.Errorf("Expected old object 'Original text', got '%s'", diff.Modified[0].OldObject)
		}
		if diff.Modified[0].NewObject != "Updated text" {
			t.Errorf("Expected new object 'Updated text', got '%s'", diff.Modified[0].NewObject)
		}
	}

	// Test summary
	summary := diff.Summary()
	if summary != "1 added, 1 removed, 1 modified" {
		t.Errorf("Unexpected summary: %s", summary)
	}
}

func TestTemporalStore_Stats(t *testing.T) {
	ts := NewTemporalStore()

	// Add some data
	ts.AddVersion("https://example.org/doc1", VersionInfo{
		URI:       "https://example.org/doc1:v1",
		Version:   "1.0",
		ValidFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:    "active",
	})
	ts.AddVersion("https://example.org/doc2", VersionInfo{
		URI:       "https://example.org/doc2:v1",
		Version:   "1.0",
		ValidFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:    "active",
	})
	ts.CreateGraph("test-graph")

	stats := ts.Stats()

	if stats.VersionedEntities != 2 {
		t.Errorf("Expected 2 versioned entities, got %d", stats.VersionedEntities)
	}
	if stats.TotalVersions != 2 {
		t.Errorf("Expected 2 total versions, got %d", stats.TotalVersions)
	}
	if stats.ActiveVersions != 2 {
		t.Errorf("Expected 2 active versions, got %d", stats.ActiveVersions)
	}
	if stats.GraphCount != 1 {
		t.Errorf("Expected 1 graph, got %d", stats.GraphCount)
	}
}

func TestVersionDiff_IsEmpty(t *testing.T) {
	emptyDiff := &VersionDiff{}
	if !emptyDiff.IsEmpty() {
		t.Error("Expected empty diff to be empty")
	}

	nonEmptyDiff := &VersionDiff{
		Added: []Triple{{Subject: "s", Predicate: "p", Object: "o"}},
	}
	if nonEmptyDiff.IsEmpty() {
		t.Error("Expected non-empty diff to not be empty")
	}
}

func TestTemporalStore_AddVersion_Errors(t *testing.T) {
	ts := NewTemporalStore()

	// Empty canonical URI
	err := ts.AddVersion("", VersionInfo{URI: "test"})
	if err == nil {
		t.Error("Expected error for empty canonical URI")
	}

	// Empty version URI
	err = ts.AddVersion("test", VersionInfo{})
	if err == nil {
		t.Error("Expected error for empty version URI")
	}
}

func TestTemporalStore_SetCurrentVersion_NotFound(t *testing.T) {
	ts := NewTemporalStore()

	// No versions exist
	err := ts.SetCurrentVersion("https://example.org/doc", "https://example.org/doc:v1")
	if err == nil {
		t.Error("Expected error when no versions exist")
	}

	// Add a version but try to set a different one as current
	ts.AddVersion("https://example.org/doc", VersionInfo{
		URI:     "https://example.org/doc:v1",
		Version: "1.0",
	})
	err = ts.SetCurrentVersion("https://example.org/doc", "https://example.org/doc:v99")
	if err == nil {
		t.Error("Expected error for non-existent version")
	}
}

func TestTemporalStore_GetVersionHistory_NotFound(t *testing.T) {
	ts := NewTemporalStore()

	_, err := ts.GetVersionHistory("https://example.org/nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent entity")
	}
}

func TestTemporalStore_GetLatestVersion_NotFound(t *testing.T) {
	ts := NewTemporalStore()

	_, err := ts.GetLatestVersion("https://example.org/nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent entity")
	}
}

func TestNewTemporalStoreFromTripleStore(t *testing.T) {
	baseStore := NewTripleStore()
	baseStore.Add("subject", "predicate", "object")

	ts := NewTemporalStoreFromTripleStore(baseStore)

	if ts.TripleStore != baseStore {
		t.Error("Expected same triple store instance")
	}

	// Verify data is accessible
	triples := ts.Find("subject", "", "")
	if len(triples) != 1 {
		t.Errorf("Expected 1 triple from base store, got %d", len(triples))
	}
}
