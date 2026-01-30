package ukleg

import (
	"sync"
	"testing"
	"time"
)

func TestCacheSetAndGet(t *testing.T) {
	cache := NewValidationCache(1 * time.Hour)

	expectedResult := ValidationResult{
		URI:        "https://www.legislation.gov.uk/ukpga/2018/12",
		Valid:      true,
		StatusCode: 200,
		CheckedAt:  time.Now(),
	}

	cache.Set("ukpga/2018/12", expectedResult)

	retrievedResult, found := cache.Get("ukpga/2018/12")
	if !found {
		t.Fatal("Expected cache hit, got miss")
	}
	if retrievedResult.URI != expectedResult.URI {
		t.Errorf("URI: got %q, want %q", retrievedResult.URI, expectedResult.URI)
	}
	if retrievedResult.Valid != expectedResult.Valid {
		t.Errorf("Valid: got %v, want %v", retrievedResult.Valid, expectedResult.Valid)
	}
	if retrievedResult.StatusCode != expectedResult.StatusCode {
		t.Errorf("StatusCode: got %d, want %d", retrievedResult.StatusCode, expectedResult.StatusCode)
	}
}

func TestCacheMiss(t *testing.T) {
	cache := NewValidationCache(1 * time.Hour)

	_, found := cache.Get("nonexistent-key")
	if found {
		t.Error("Expected cache miss for nonexistent key")
	}
}

func TestCacheTTLExpiration(t *testing.T) {
	cache := NewValidationCache(5 * time.Millisecond)

	cache.Set("short-lived", ValidationResult{
		URI:   "https://www.legislation.gov.uk/ukpga/2018/12",
		Valid: true,
	})

	// Should be available immediately.
	_, found := cache.Get("short-lived")
	if !found {
		t.Fatal("Expected cache hit before expiration")
	}

	// Wait for expiration.
	time.Sleep(10 * time.Millisecond)

	_, found = cache.Get("short-lived")
	if found {
		t.Error("Expected cache miss after TTL expiration")
	}
}

func TestCacheInvalidate(t *testing.T) {
	cache := NewValidationCache(1 * time.Hour)

	cache.Set("to-remove", ValidationResult{URI: "test", Valid: true})

	// Verify entry exists.
	_, found := cache.Get("to-remove")
	if !found {
		t.Fatal("Expected cache hit before invalidation")
	}

	cache.Invalidate("to-remove")

	_, found = cache.Get("to-remove")
	if found {
		t.Error("Expected cache miss after invalidation")
	}
}

func TestCacheInvalidateNonexistent(t *testing.T) {
	cache := NewValidationCache(1 * time.Hour)

	// Should not panic.
	cache.Invalidate("nonexistent-key")
}

func TestCacheLen(t *testing.T) {
	cache := NewValidationCache(1 * time.Hour)

	if cache.Len() != 0 {
		t.Errorf("Empty cache Len: got %d, want 0", cache.Len())
	}

	cache.Set("key1", ValidationResult{URI: "uri1"})
	cache.Set("key2", ValidationResult{URI: "uri2"})
	cache.Set("key3", ValidationResult{URI: "uri3"})

	if cache.Len() != 3 {
		t.Errorf("Cache Len after 3 sets: got %d, want 3", cache.Len())
	}
}

func TestCacheOverwrite(t *testing.T) {
	cache := NewValidationCache(1 * time.Hour)

	cache.Set("key", ValidationResult{URI: "original", Valid: false})
	cache.Set("key", ValidationResult{URI: "updated", Valid: true})

	result, found := cache.Get("key")
	if !found {
		t.Fatal("Expected cache hit after overwrite")
	}
	if result.URI != "updated" {
		t.Errorf("URI: got %q, want %q", result.URI, "updated")
	}
	if !result.Valid {
		t.Error("Valid: expected true after overwrite")
	}

	if cache.Len() != 1 {
		t.Errorf("Cache Len after overwrite: got %d, want 1", cache.Len())
	}
}

func TestCacheConcurrentAccess(t *testing.T) {
	cache := NewValidationCache(1 * time.Hour)
	concurrencyLevel := 50

	var waitGroup sync.WaitGroup
	waitGroup.Add(concurrencyLevel)

	for i := 0; i < concurrencyLevel; i++ {
		go func(index int) {
			defer waitGroup.Done()

			key := "concurrent-key"
			cache.Set(key, ValidationResult{
				URI:        "https://www.legislation.gov.uk/ukpga/2018/12",
				Valid:      true,
				StatusCode: 200,
			})
			cache.Get(key)
			cache.Len()
		}(i)
	}

	waitGroup.Wait()

	// If we reach here without a race condition panic, the test passes.
	result, found := cache.Get("concurrent-key")
	if !found {
		t.Fatal("Expected cache hit after concurrent access")
	}
	if !result.Valid {
		t.Error("Expected Valid after concurrent writes")
	}
}
