package linkcheck

import (
	"sync"
	"time"
)

// cacheEntry holds a cached link result and its expiration time.
type cacheEntry struct {
	result    *LinkResult
	expiresAt time.Time
}

// LinkCache is a thread-safe, in-memory TTL cache for link validation results.
// Entries are lazily expired on access (checked during Get).
type LinkCache struct {
	mu         sync.RWMutex
	entries    map[string]cacheEntry
	defaultTTL time.Duration
}

// NewLinkCache creates a new cache with the given default TTL.
func NewLinkCache(defaultTTL time.Duration) *LinkCache {
	return &LinkCache{
		entries:    make(map[string]cacheEntry),
		defaultTTL: defaultTTL,
	}
}

// Get retrieves a cached link result by URI.
// Returns the result and true if found and not expired, or nil and false otherwise.
// Expired entries are lazily removed on access.
func (linkCache *LinkCache) Get(uri string) (*LinkResult, bool) {
	linkCache.mu.RLock()
	entry, exists := linkCache.entries[uri]
	linkCache.mu.RUnlock()

	if !exists {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		// Lazily remove expired entry.
		linkCache.mu.Lock()
		// Re-check in case another goroutine already removed or replaced it.
		if current, stillExists := linkCache.entries[uri]; stillExists && time.Now().After(current.expiresAt) {
			delete(linkCache.entries, uri)
		}
		linkCache.mu.Unlock()
		return nil, false
	}

	return entry.result, true
}

// Set stores a link result in the cache with the default TTL.
func (linkCache *LinkCache) Set(uri string, result *LinkResult) {
	linkCache.mu.Lock()
	linkCache.entries[uri] = cacheEntry{
		result:    result,
		expiresAt: time.Now().Add(linkCache.defaultTTL),
	}
	linkCache.mu.Unlock()
}

// Invalidate removes a specific entry from the cache.
func (linkCache *LinkCache) Invalidate(uri string) {
	linkCache.mu.Lock()
	delete(linkCache.entries, uri)
	linkCache.mu.Unlock()
}

// Clear removes all entries from the cache.
func (linkCache *LinkCache) Clear() {
	linkCache.mu.Lock()
	linkCache.entries = make(map[string]cacheEntry)
	linkCache.mu.Unlock()
}

// Len returns the number of entries currently in the cache (including potentially expired ones).
func (linkCache *LinkCache) Len() int {
	linkCache.mu.RLock()
	count := len(linkCache.entries)
	linkCache.mu.RUnlock()
	return count
}

// Cleanup removes all expired entries from the cache.
// This can be called periodically to prevent memory growth.
func (linkCache *LinkCache) Cleanup() int {
	linkCache.mu.Lock()
	defer linkCache.mu.Unlock()

	expiredCount := 0
	now := time.Now()

	for uri, entry := range linkCache.entries {
		if now.After(entry.expiresAt) {
			delete(linkCache.entries, uri)
			expiredCount++
		}
	}

	return expiredCount
}
