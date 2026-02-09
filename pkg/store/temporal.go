package store

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// TemporalStore extends TripleStore with temporal versioning capabilities.
// It uses named graphs to track different versions of entities over time,
// allowing queries at specific points in time and version history tracking.
type TemporalStore struct {
	// Base triple store for the "current" state
	*TripleStore

	// Named graphs for versioned data
	// Key: graph name (often a version identifier or timestamp)
	graphs map[string]*TripleStore

	// Version metadata
	// Key: subject URI, Value: list of versions
	versionIndex map[string][]VersionInfo

	// Mutex for graph operations
	graphMu sync.RWMutex
}

// VersionInfo contains metadata about a specific version of an entity.
type VersionInfo struct {
	// URI is the version-specific URI.
	URI string `json:"uri"`

	// Version is the version identifier (e.g., "1.0", "2023-01-15").
	Version string `json:"version"`

	// ValidFrom is when this version became valid.
	ValidFrom time.Time `json:"valid_from"`

	// ValidUntil is when this version ceased to be valid (zero if still valid).
	ValidUntil time.Time `json:"valid_until,omitempty"`

	// GraphName is the named graph containing this version's triples.
	GraphName string `json:"graph_name"`

	// SupersedesURI is the URI of the version this one replaced.
	SupersedesURI string `json:"supersedes_uri,omitempty"`

	// SupersededByURI is the URI of the version that replaced this one.
	SupersededByURI string `json:"superseded_by_uri,omitempty"`

	// Status indicates the version status (draft, active, superseded, withdrawn).
	Status string `json:"status"`

	// MeetingURI links to the meeting where this version was adopted.
	MeetingURI string `json:"meeting_uri,omitempty"`
}

// VersionHistory contains the complete version history for an entity.
type VersionHistory struct {
	// CanonicalURI is the abstract/canonical URI for the entity.
	CanonicalURI string `json:"canonical_uri"`

	// CurrentVersion is the currently active version.
	CurrentVersion *VersionInfo `json:"current_version,omitempty"`

	// Versions lists all versions in chronological order.
	Versions []VersionInfo `json:"versions"`
}

// TemporalQueryResult contains results from a temporal query.
type TemporalQueryResult struct {
	// AsOf is the point in time the query was evaluated at.
	AsOf time.Time `json:"as_of"`

	// Version is the version that was active at that time.
	Version *VersionInfo `json:"version,omitempty"`

	// Triples contains the matching triples.
	Triples []Triple `json:"triples"`
}

// NewTemporalStore creates a new temporal store.
func NewTemporalStore() *TemporalStore {
	return &TemporalStore{
		TripleStore:  NewTripleStore(),
		graphs:       make(map[string]*TripleStore),
		versionIndex: make(map[string][]VersionInfo),
	}
}

// NewTemporalStoreFromTripleStore wraps an existing TripleStore with temporal capabilities.
func NewTemporalStoreFromTripleStore(ts *TripleStore) *TemporalStore {
	return &TemporalStore{
		TripleStore:  ts,
		graphs:       make(map[string]*TripleStore),
		versionIndex: make(map[string][]VersionInfo),
	}
}

// CreateGraph creates a new named graph for versioned data.
func (ts *TemporalStore) CreateGraph(name string) *TripleStore {
	ts.graphMu.Lock()
	defer ts.graphMu.Unlock()

	if existing, ok := ts.graphs[name]; ok {
		return existing
	}

	graph := NewTripleStore()
	ts.graphs[name] = graph
	return graph
}

// GetGraph retrieves a named graph by name.
func (ts *TemporalStore) GetGraph(name string) (*TripleStore, bool) {
	ts.graphMu.RLock()
	defer ts.graphMu.RUnlock()

	graph, ok := ts.graphs[name]
	return graph, ok
}

// ListGraphs returns the names of all named graphs.
func (ts *TemporalStore) ListGraphs() []string {
	ts.graphMu.RLock()
	defer ts.graphMu.RUnlock()

	names := make([]string, 0, len(ts.graphs))
	for name := range ts.graphs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// DeleteGraph removes a named graph.
func (ts *TemporalStore) DeleteGraph(name string) bool {
	ts.graphMu.Lock()
	defer ts.graphMu.Unlock()

	if _, ok := ts.graphs[name]; ok {
		delete(ts.graphs, name)
		return true
	}
	return false
}

// AddVersioned adds a triple to a specific version/graph.
func (ts *TemporalStore) AddVersioned(subject, predicate, object, graphName string) error {
	graph := ts.CreateGraph(graphName)
	return graph.Add(subject, predicate, object)
}

// AddVersion registers a new version for an entity.
func (ts *TemporalStore) AddVersion(canonicalURI string, version VersionInfo) error {
	if canonicalURI == "" {
		return fmt.Errorf("canonical URI cannot be empty")
	}
	if version.URI == "" {
		return fmt.Errorf("version URI cannot be empty")
	}

	ts.graphMu.Lock()
	defer ts.graphMu.Unlock()

	// Add to version index
	ts.versionIndex[canonicalURI] = append(ts.versionIndex[canonicalURI], version)

	// Sort versions by ValidFrom date
	sort.Slice(ts.versionIndex[canonicalURI], func(i, j int) bool {
		return ts.versionIndex[canonicalURI][i].ValidFrom.Before(ts.versionIndex[canonicalURI][j].ValidFrom)
	})

	// Add version triples to main store
	ts.TripleStore.Add(version.URI, PropVersionOf, canonicalURI)
	ts.TripleStore.Add(version.URI, PropVersionNumber, version.Version)
	ts.TripleStore.Add(version.URI, PropVersionStatus, version.Status)

	if !version.ValidFrom.IsZero() {
		ts.TripleStore.Add(version.URI, PropValidFrom, version.ValidFrom.Format(time.RFC3339))
	}
	if !version.ValidUntil.IsZero() {
		ts.TripleStore.Add(version.URI, PropValidUntil, version.ValidUntil.Format(time.RFC3339))
	}
	if version.SupersedesURI != "" {
		ts.TripleStore.Add(version.URI, PropSupersedes, version.SupersedesURI)
		ts.TripleStore.Add(version.SupersedesURI, PropSupersededBy, version.URI)
	}
	if version.MeetingURI != "" {
		ts.TripleStore.Add(version.URI, PropDecidedAt, version.MeetingURI)
	}

	return nil
}

// SetCurrentVersion marks a version as the current/active version.
func (ts *TemporalStore) SetCurrentVersion(canonicalURI, versionURI string) error {
	ts.graphMu.Lock()
	defer ts.graphMu.Unlock()

	versions, ok := ts.versionIndex[canonicalURI]
	if !ok {
		return fmt.Errorf("no versions found for %s", canonicalURI)
	}

	// Find and update the version
	found := false
	for i := range versions {
		if versions[i].URI == versionURI {
			versions[i].Status = "active"
			found = true
		} else if versions[i].Status == "active" {
			// Mark previous current as superseded
			versions[i].Status = "superseded"
			if versions[i].ValidUntil.IsZero() {
				versions[i].ValidUntil = time.Now()
			}
		}
	}

	if !found {
		return fmt.Errorf("version %s not found for %s", versionURI, canonicalURI)
	}

	ts.versionIndex[canonicalURI] = versions

	// Update triples
	ts.TripleStore.Add(canonicalURI, PropCurrentVersion, versionURI)
	ts.TripleStore.Add(versionURI, PropVersionStatus, "active")

	return nil
}

// GetVersionHistory returns the complete version history for an entity.
func (ts *TemporalStore) GetVersionHistory(canonicalURI string) (*VersionHistory, error) {
	ts.graphMu.RLock()
	defer ts.graphMu.RUnlock()

	versions, ok := ts.versionIndex[canonicalURI]
	if !ok {
		return nil, fmt.Errorf("no versions found for %s", canonicalURI)
	}

	history := &VersionHistory{
		CanonicalURI: canonicalURI,
		Versions:     make([]VersionInfo, len(versions)),
	}

	copy(history.Versions, versions)

	// Find current version
	for i := range history.Versions {
		if history.Versions[i].Status == "active" {
			v := history.Versions[i]
			history.CurrentVersion = &v
			break
		}
	}

	return history, nil
}

// QueryAtTime finds triples valid at a specific point in time.
func (ts *TemporalStore) QueryAtTime(subject, predicate, object string, asOf time.Time) (*TemporalQueryResult, error) {
	ts.graphMu.RLock()
	defer ts.graphMu.RUnlock()

	result := &TemporalQueryResult{
		AsOf:    asOf,
		Triples: []Triple{},
	}

	// If subject is specified, find the version valid at that time
	if subject != "" {
		version := ts.findVersionAtTime(subject, asOf)
		if version != nil {
			result.Version = version

			// Query from the version's graph if it exists
			if graph, ok := ts.graphs[version.GraphName]; ok {
				triples := graph.Find(subject, predicate, object)
				result.Triples = triples
				return result, nil
			}

			// Fall back to querying the version URI directly
			if version.URI != subject {
				triples := ts.TripleStore.Find(version.URI, predicate, object)
				result.Triples = triples
				return result, nil
			}
		}
	}

	// Query all graphs and filter by validity
	// Start with main store
	mainTriples := ts.TripleStore.Find(subject, predicate, object)
	for _, t := range mainTriples {
		if ts.isValidAtTime(t.Subject, asOf) {
			result.Triples = append(result.Triples, t)
		}
	}

	// Query named graphs
	for _, graph := range ts.graphs {
		graphTriples := graph.Find(subject, predicate, object)
		for _, t := range graphTriples {
			if ts.isValidAtTime(t.Subject, asOf) {
				result.Triples = append(result.Triples, t)
			}
		}
	}

	return result, nil
}

// GetVersionAtTime returns the version of an entity that was valid at a specific time.
func (ts *TemporalStore) GetVersionAtTime(canonicalURI string, asOf time.Time) (*VersionInfo, error) {
	ts.graphMu.RLock()
	defer ts.graphMu.RUnlock()

	version := ts.findVersionAtTime(canonicalURI, asOf)
	if version == nil {
		return nil, fmt.Errorf("no version found for %s at %s", canonicalURI, asOf.Format(time.RFC3339))
	}

	return version, nil
}

// findVersionAtTime is an internal method to find the version valid at a given time.
// Caller must hold at least a read lock.
func (ts *TemporalStore) findVersionAtTime(canonicalURI string, asOf time.Time) *VersionInfo {
	versions, ok := ts.versionIndex[canonicalURI]
	if !ok {
		return nil
	}

	// Find the latest version valid at the given time
	// Versions are sorted by ValidFrom, so iterate backwards
	for i := len(versions) - 1; i >= 0; i-- {
		v := versions[i]
		if v.ValidFrom.Before(asOf) || v.ValidFrom.Equal(asOf) {
			if v.ValidUntil.IsZero() || v.ValidUntil.After(asOf) {
				return &v
			}
		}
	}

	return nil
}

// isValidAtTime checks if an entity (by subject URI) is valid at the given time.
// Caller must hold at least a read lock.
func (ts *TemporalStore) isValidAtTime(subjectURI string, asOf time.Time) bool {
	// Check if subject has version info
	versions, ok := ts.versionIndex[subjectURI]
	if !ok {
		// No version info - assume always valid
		return true
	}

	for _, v := range versions {
		if v.URI == subjectURI {
			if v.ValidFrom.Before(asOf) || v.ValidFrom.Equal(asOf) {
				if v.ValidUntil.IsZero() || v.ValidUntil.After(asOf) {
					return true
				}
			}
		}
	}

	return false
}

// GetLatestVersion returns the most recent version of an entity.
func (ts *TemporalStore) GetLatestVersion(canonicalURI string) (*VersionInfo, error) {
	ts.graphMu.RLock()
	defer ts.graphMu.RUnlock()

	versions, ok := ts.versionIndex[canonicalURI]
	if !ok || len(versions) == 0 {
		return nil, fmt.Errorf("no versions found for %s", canonicalURI)
	}

	// Return the last version (sorted by ValidFrom)
	latest := versions[len(versions)-1]
	return &latest, nil
}

// GetActiveVersions returns all currently active versions (no ValidUntil set).
func (ts *TemporalStore) GetActiveVersions() []VersionInfo {
	ts.graphMu.RLock()
	defer ts.graphMu.RUnlock()

	var active []VersionInfo
	for _, versions := range ts.versionIndex {
		for _, v := range versions {
			if v.Status == "active" || (v.ValidUntil.IsZero() && v.Status != "superseded" && v.Status != "withdrawn") {
				active = append(active, v)
			}
		}
	}

	return active
}

// LinkVersionToMeeting associates a version with the meeting where it was adopted.
func (ts *TemporalStore) LinkVersionToMeeting(versionURI, meetingURI string) error {
	ts.graphMu.Lock()
	defer ts.graphMu.Unlock()

	// Update version index
	for canonicalURI, versions := range ts.versionIndex {
		for i, v := range versions {
			if v.URI == versionURI {
				ts.versionIndex[canonicalURI][i].MeetingURI = meetingURI
				break
			}
		}
	}

	// Add triple
	return ts.TripleStore.Add(versionURI, PropDecidedAt, meetingURI)
}

// SupersedeVersion marks an old version as superseded by a new version.
func (ts *TemporalStore) SupersedeVersion(oldVersionURI, newVersionURI string, supersededAt time.Time) error {
	ts.graphMu.Lock()
	defer ts.graphMu.Unlock()

	// Update version index
	for canonicalURI, versions := range ts.versionIndex {
		for i, v := range versions {
			if v.URI == oldVersionURI {
				ts.versionIndex[canonicalURI][i].SupersededByURI = newVersionURI
				ts.versionIndex[canonicalURI][i].Status = "superseded"
				if supersededAt.IsZero() {
					ts.versionIndex[canonicalURI][i].ValidUntil = time.Now()
				} else {
					ts.versionIndex[canonicalURI][i].ValidUntil = supersededAt
				}
			}
			if v.URI == newVersionURI {
				ts.versionIndex[canonicalURI][i].SupersedesURI = oldVersionURI
			}
		}
	}

	// Add triples
	ts.TripleStore.Add(newVersionURI, PropSupersedes, oldVersionURI)
	ts.TripleStore.Add(oldVersionURI, PropSupersededBy, newVersionURI)
	ts.TripleStore.Add(newVersionURI, PropPreviousVersion, oldVersionURI)
	ts.TripleStore.Add(oldVersionURI, PropNextVersion, newVersionURI)
	if !supersededAt.IsZero() {
		ts.TripleStore.Add(oldVersionURI, PropSupersededDate, supersededAt.Format(time.RFC3339))
	}

	return nil
}

// CompareVersions returns the triples that differ between two versions.
func (ts *TemporalStore) CompareVersions(version1URI, version2URI string) (*VersionDiff, error) {
	ts.graphMu.RLock()
	defer ts.graphMu.RUnlock()

	diff := &VersionDiff{
		Version1URI: version1URI,
		Version2URI: version2URI,
		Added:       []Triple{},
		Removed:     []Triple{},
		Modified:    []TripleChange{},
	}

	// Get triples for both versions
	triples1 := ts.TripleStore.Find(version1URI, "", "")
	triples2 := ts.TripleStore.Find(version2URI, "", "")

	// Build maps for comparison (predicate -> object)
	map1 := make(map[string]string)
	for _, t := range triples1 {
		map1[t.Predicate] = t.Object
	}

	map2 := make(map[string]string)
	for _, t := range triples2 {
		map2[t.Predicate] = t.Object
	}

	// Find added (in v2 but not v1)
	for _, t := range triples2 {
		if _, exists := map1[t.Predicate]; !exists {
			diff.Added = append(diff.Added, t)
		}
	}

	// Find removed (in v1 but not v2)
	for _, t := range triples1 {
		if _, exists := map2[t.Predicate]; !exists {
			diff.Removed = append(diff.Removed, t)
		}
	}

	// Find modified (same predicate, different object)
	for _, t := range triples1 {
		if newObj, exists := map2[t.Predicate]; exists {
			if newObj != t.Object {
				diff.Modified = append(diff.Modified, TripleChange{
					Predicate: t.Predicate,
					OldObject: t.Object,
					NewObject: newObj,
				})
			}
		}
	}

	return diff, nil
}

// VersionDiff represents differences between two versions.
type VersionDiff struct {
	// Version1URI is the first (older) version.
	Version1URI string `json:"version1_uri"`

	// Version2URI is the second (newer) version.
	Version2URI string `json:"version2_uri"`

	// Added contains triples present in v2 but not v1.
	Added []Triple `json:"added,omitempty"`

	// Removed contains triples present in v1 but not v2.
	Removed []Triple `json:"removed,omitempty"`

	// Modified contains triples with the same predicate but different values.
	Modified []TripleChange `json:"modified,omitempty"`
}

// TripleChange represents a change in a triple's object value.
type TripleChange struct {
	// Predicate is the predicate that changed.
	Predicate string `json:"predicate"`

	// OldObject is the previous object value.
	OldObject string `json:"old_object"`

	// NewObject is the new object value.
	NewObject string `json:"new_object"`
}

// Summary returns a text summary of the version diff.
func (d *VersionDiff) Summary() string {
	return fmt.Sprintf("%d added, %d removed, %d modified",
		len(d.Added), len(d.Removed), len(d.Modified))
}

// IsEmpty returns true if there are no differences.
func (d *VersionDiff) IsEmpty() bool {
	return len(d.Added) == 0 && len(d.Removed) == 0 && len(d.Modified) == 0
}

// Stats returns statistics about the temporal store.
type TemporalStats struct {
	// BaseStats contains stats from the main triple store.
	BaseStats IndexStats `json:"base_stats"`

	// GraphCount is the number of named graphs.
	GraphCount int `json:"graph_count"`

	// VersionedEntities is the count of entities with version history.
	VersionedEntities int `json:"versioned_entities"`

	// TotalVersions is the total number of versions across all entities.
	TotalVersions int `json:"total_versions"`

	// ActiveVersions is the count of currently active versions.
	ActiveVersions int `json:"active_versions"`
}

// Stats returns statistics about the temporal store.
func (ts *TemporalStore) Stats() TemporalStats {
	ts.graphMu.RLock()
	defer ts.graphMu.RUnlock()

	stats := TemporalStats{
		BaseStats:         ts.TripleStore.Stats(),
		GraphCount:        len(ts.graphs),
		VersionedEntities: len(ts.versionIndex),
	}

	activeCount := 0
	for _, versions := range ts.versionIndex {
		stats.TotalVersions += len(versions)
		for _, v := range versions {
			if v.Status == "active" {
				activeCount++
			}
		}
	}
	stats.ActiveVersions = activeCount

	return stats
}
