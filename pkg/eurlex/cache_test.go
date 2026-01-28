package eurlex

import (
	"sync"
	"testing"
	"time"
)

func TestCacheSetAndGet(t *testing.T) {
	validationCache := NewValidationCache(1 * time.Hour)

	result := ValidationResult{
		URI:        "http://data.europa.eu/eli/reg/2016/679/oj",
		Valid:      true,
		StatusCode: 200,
		CheckedAt:  time.Now(),
	}

	validationCache.Set("test-key", result)

	retrieved, found := validationCache.Get("test-key")
	if !found {
		t.Fatal("Expected to find cached entry")
	}
	if retrieved.URI != result.URI {
		t.Errorf("URI: got %q, want %q", retrieved.URI, result.URI)
	}
	if retrieved.Valid != result.Valid {
		t.Errorf("Valid: got %v, want %v", retrieved.Valid, result.Valid)
	}
	if retrieved.StatusCode != result.StatusCode {
		t.Errorf("StatusCode: got %d, want %d", retrieved.StatusCode, result.StatusCode)
	}
}

func TestCacheMiss(t *testing.T) {
	validationCache := NewValidationCache(1 * time.Hour)

	_, found := validationCache.Get("nonexistent-key")
	if found {
		t.Error("Expected cache miss for nonexistent key")
	}
}

func TestCacheTTLExpiration(t *testing.T) {
	// Use a very short TTL so it expires almost immediately.
	validationCache := NewValidationCache(1 * time.Millisecond)

	result := ValidationResult{
		URI:   "http://data.europa.eu/eli/reg/2016/679/oj",
		Valid: true,
	}

	validationCache.Set("expiring-key", result)

	// Wait for the entry to expire.
	time.Sleep(5 * time.Millisecond)

	_, found := validationCache.Get("expiring-key")
	if found {
		t.Error("Expected cache entry to be expired")
	}
}

func TestCacheInvalidate(t *testing.T) {
	validationCache := NewValidationCache(1 * time.Hour)

	result := ValidationResult{URI: "test-uri", Valid: true}
	validationCache.Set("to-remove", result)

	// Confirm it's there.
	_, found := validationCache.Get("to-remove")
	if !found {
		t.Fatal("Expected entry to exist before invalidation")
	}

	validationCache.Invalidate("to-remove")

	_, found = validationCache.Get("to-remove")
	if found {
		t.Error("Expected entry to be removed after invalidation")
	}
}

func TestCacheInvalidateNonexistent(t *testing.T) {
	validationCache := NewValidationCache(1 * time.Hour)

	// Should not panic when invalidating a key that doesn't exist.
	validationCache.Invalidate("does-not-exist")
}

func TestCacheLen(t *testing.T) {
	validationCache := NewValidationCache(1 * time.Hour)

	if validationCache.Len() != 0 {
		t.Errorf("Expected empty cache, got %d entries", validationCache.Len())
	}

	validationCache.Set("key-1", ValidationResult{URI: "uri-1"})
	validationCache.Set("key-2", ValidationResult{URI: "uri-2"})
	validationCache.Set("key-3", ValidationResult{URI: "uri-3"})

	if validationCache.Len() != 3 {
		t.Errorf("Expected 3 entries, got %d", validationCache.Len())
	}
}

func TestCacheOverwrite(t *testing.T) {
	validationCache := NewValidationCache(1 * time.Hour)

	validationCache.Set("key", ValidationResult{URI: "first", Valid: false})
	validationCache.Set("key", ValidationResult{URI: "second", Valid: true})

	retrieved, found := validationCache.Get("key")
	if !found {
		t.Fatal("Expected to find cached entry after overwrite")
	}
	if retrieved.URI != "second" {
		t.Errorf("URI: got %q, want %q (should be overwritten)", retrieved.URI, "second")
	}
	if !retrieved.Valid {
		t.Error("Valid: expected true after overwrite")
	}
	if validationCache.Len() != 1 {
		t.Errorf("Expected 1 entry after overwrite, got %d", validationCache.Len())
	}
}

func TestCacheConcurrentAccess(t *testing.T) {
	validationCache := NewValidationCache(1 * time.Hour)
	concurrentWorkers := 50

	var waitGroup sync.WaitGroup
	waitGroup.Add(concurrentWorkers * 2) // readers + writers

	// Concurrent writers.
	for workerIndex := 0; workerIndex < concurrentWorkers; workerIndex++ {
		go func(index int) {
			defer waitGroup.Done()
			key := "concurrent-key"
			result := ValidationResult{
				URI:        key,
				StatusCode: index,
			}
			validationCache.Set(key, result)
		}(workerIndex)
	}

	// Concurrent readers.
	for workerIndex := 0; workerIndex < concurrentWorkers; workerIndex++ {
		go func() {
			defer waitGroup.Done()
			validationCache.Get("concurrent-key")
		}()
	}

	waitGroup.Wait()

	// Should have exactly 1 entry (all writers used the same key).
	if validationCache.Len() != 1 {
		t.Errorf("Expected 1 entry after concurrent access, got %d", validationCache.Len())
	}
}
