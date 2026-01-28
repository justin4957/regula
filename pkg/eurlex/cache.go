package eurlex

import (
	"sync"
	"time"
)

// DefaultCacheTTL is the default time-to-live for cached validation results.
const DefaultCacheTTL = 1 * time.Hour

// cacheEntry holds a cached validation result and its expiration time.
type cacheEntry struct {
	result    ValidationResult
	expiresAt time.Time
}

// ValidationCache is a thread-safe, in-memory TTL cache for URI validation results.
// Entries are lazily expired on access (checked during Get).
type ValidationCache struct {
	mu         sync.RWMutex
	entries    map[string]cacheEntry
	defaultTTL time.Duration
}

// NewValidationCache creates a new cache with the given default TTL.
func NewValidationCache(defaultTTL time.Duration) *ValidationCache {
	return &ValidationCache{
		entries:    make(map[string]cacheEntry),
		defaultTTL: defaultTTL,
	}
}

// Get retrieves a cached validation result by key.
// Returns the result and true if found and not expired, or a zero value and false otherwise.
// Expired entries are lazily removed on access.
func (validationCache *ValidationCache) Get(key string) (ValidationResult, bool) {
	validationCache.mu.RLock()
	entry, exists := validationCache.entries[key]
	validationCache.mu.RUnlock()

	if !exists {
		return ValidationResult{}, false
	}

	if time.Now().After(entry.expiresAt) {
		// Lazily remove expired entry.
		validationCache.mu.Lock()
		// Re-check in case another goroutine already removed or replaced it.
		if current, stillExists := validationCache.entries[key]; stillExists && time.Now().After(current.expiresAt) {
			delete(validationCache.entries, key)
		}
		validationCache.mu.Unlock()
		return ValidationResult{}, false
	}

	return entry.result, true
}

// Set stores a validation result in the cache with the default TTL.
func (validationCache *ValidationCache) Set(key string, result ValidationResult) {
	validationCache.mu.Lock()
	validationCache.entries[key] = cacheEntry{
		result:    result,
		expiresAt: time.Now().Add(validationCache.defaultTTL),
	}
	validationCache.mu.Unlock()
}

// Invalidate removes a specific entry from the cache.
func (validationCache *ValidationCache) Invalidate(key string) {
	validationCache.mu.Lock()
	delete(validationCache.entries, key)
	validationCache.mu.Unlock()
}

// Len returns the number of entries currently in the cache (including potentially expired ones).
func (validationCache *ValidationCache) Len() int {
	validationCache.mu.RLock()
	count := len(validationCache.entries)
	validationCache.mu.RUnlock()
	return count
}
