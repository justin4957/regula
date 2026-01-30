package fetch

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DiskCache provides persistent, file-based caching for fetch results.
// Each cached entry is stored as a JSON file keyed by a SHA-256 hash of the URL.
type DiskCache struct {
	cacheDir string
	cacheTTL time.Duration
}

// diskCacheEntry wraps a FetchResult with an expiration timestamp for TTL enforcement.
type diskCacheEntry struct {
	Result    FetchResult `json:"result"`
	ExpiresAt time.Time   `json:"expires_at"`
}

// NewDiskCache creates a new disk cache in the given directory with the specified TTL.
// Creates the directory if it does not exist.
func NewDiskCache(cacheDir string, cacheTTL time.Duration) (*DiskCache, error) {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory %s: %w", cacheDir, err)
	}

	return &DiskCache{
		cacheDir: cacheDir,
		cacheTTL: cacheTTL,
	}, nil
}

// Get retrieves a cached fetch result for the given URL.
// Returns the result and true if found and not expired, or a zero FetchResult and false otherwise.
func (cache *DiskCache) Get(url string) (FetchResult, bool) {
	cacheFilePath := cache.pathFor(url)

	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		return FetchResult{}, false
	}

	var entry diskCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return FetchResult{}, false
	}

	if time.Now().After(entry.ExpiresAt) {
		// Entry expired — remove stale file.
		_ = os.Remove(cacheFilePath)
		return FetchResult{}, false
	}

	return entry.Result, true
}

// Set stores a fetch result in the cache for the given URL.
func (cache *DiskCache) Set(url string, result FetchResult) error {
	entry := diskCacheEntry{
		Result:    result,
		ExpiresAt: time.Now().Add(cache.cacheTTL),
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	cacheFilePath := cache.pathFor(url)
	if err := os.WriteFile(cacheFilePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write cache file %s: %w", cacheFilePath, err)
	}

	return nil
}

// keyFor returns the SHA-256 hash of the URL, used as the cache filename.
func (cache *DiskCache) keyFor(url string) string {
	hash := sha256.Sum256([]byte(url))
	return hex.EncodeToString(hash[:])
}

// pathFor returns the full file path for a cached URL.
func (cache *DiskCache) pathFor(url string) string {
	return filepath.Join(cache.cacheDir, cache.keyFor(url)+".json")
}
