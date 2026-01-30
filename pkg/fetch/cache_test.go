package fetch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiskCache_SetAndGet(t *testing.T) {
	cacheDir := t.TempDir()
	cache, err := NewDiskCache(cacheDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewDiskCache failed: %v", err)
	}

	testResult := FetchResult{
		Reference: FetchableReference{
			URN: "urn:eu:regulation:2016/679",
			URL: "http://data.europa.eu/eli/reg/2016/679/oj",
		},
		Success:    true,
		StatusCode: 200,
		Metadata:   map[string]string{"type": "reg", "year": "2016", "number": "679"},
		FetchedAt:  time.Now(),
	}

	testURL := "http://data.europa.eu/eli/reg/2016/679/oj"

	if err := cache.Set(testURL, testResult); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	retrievedResult, found := cache.Get(testURL)
	if !found {
		t.Fatal("Get returned not found for cached URL")
	}

	if retrievedResult.Reference.URN != testResult.Reference.URN {
		t.Errorf("URN: got %q, want %q", retrievedResult.Reference.URN, testResult.Reference.URN)
	}
	if retrievedResult.StatusCode != testResult.StatusCode {
		t.Errorf("StatusCode: got %d, want %d", retrievedResult.StatusCode, testResult.StatusCode)
	}
	if !retrievedResult.Success {
		t.Error("Success: got false, want true")
	}
}

func TestDiskCache_Miss(t *testing.T) {
	cacheDir := t.TempDir()
	cache, err := NewDiskCache(cacheDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewDiskCache failed: %v", err)
	}

	_, found := cache.Get("http://nonexistent.example.com/doc")
	if found {
		t.Error("Get returned found for uncached URL")
	}
}

func TestDiskCache_TTLExpiration(t *testing.T) {
	cacheDir := t.TempDir()
	// Create cache with 1 millisecond TTL for immediate expiration.
	cache, err := NewDiskCache(cacheDir, 1*time.Millisecond)
	if err != nil {
		t.Fatalf("NewDiskCache failed: %v", err)
	}

	testURL := "http://data.europa.eu/eli/reg/2016/679/oj"
	testResult := FetchResult{
		Success:    true,
		StatusCode: 200,
	}

	if err := cache.Set(testURL, testResult); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Wait for expiration.
	time.Sleep(5 * time.Millisecond)

	_, found := cache.Get(testURL)
	if found {
		t.Error("Get returned found for expired entry")
	}

	// Verify the stale file was cleaned up.
	cacheFilePath := cache.pathFor(testURL)
	if _, err := os.Stat(cacheFilePath); !os.IsNotExist(err) {
		t.Error("Expired cache file was not removed")
	}
}

func TestDiskCache_InvalidDir(t *testing.T) {
	// Try to create a cache in a path that can't be created.
	invalidPath := filepath.Join(t.TempDir(), "nonexistent", "\x00invalid")
	_, err := NewDiskCache(invalidPath, 1*time.Hour)
	if err == nil {
		t.Error("Expected error for invalid cache directory, got nil")
	}
}

func TestDiskCache_Overwrite(t *testing.T) {
	cacheDir := t.TempDir()
	cache, err := NewDiskCache(cacheDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewDiskCache failed: %v", err)
	}

	testURL := "http://data.europa.eu/eli/reg/2016/679/oj"

	firstResult := FetchResult{
		Success:    false,
		StatusCode: 404,
		Error:      "not found",
	}

	secondResult := FetchResult{
		Success:    true,
		StatusCode: 200,
	}

	if err := cache.Set(testURL, firstResult); err != nil {
		t.Fatalf("First Set failed: %v", err)
	}

	if err := cache.Set(testURL, secondResult); err != nil {
		t.Fatalf("Second Set failed: %v", err)
	}

	retrievedResult, found := cache.Get(testURL)
	if !found {
		t.Fatal("Get returned not found after overwrite")
	}
	if !retrievedResult.Success {
		t.Error("Retrieved result should reflect second (successful) set")
	}
	if retrievedResult.StatusCode != 200 {
		t.Errorf("StatusCode: got %d, want 200", retrievedResult.StatusCode)
	}
}

func TestDiskCache_KeyFor(t *testing.T) {
	cacheDir := t.TempDir()
	cache, err := NewDiskCache(cacheDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewDiskCache failed: %v", err)
	}

	// Same URL should produce the same key.
	key1 := cache.keyFor("http://example.com/doc1")
	key2 := cache.keyFor("http://example.com/doc1")
	if key1 != key2 {
		t.Errorf("Same URL produced different keys: %q vs %q", key1, key2)
	}

	// Different URLs should produce different keys.
	key3 := cache.keyFor("http://example.com/doc2")
	if key1 == key3 {
		t.Error("Different URLs produced the same key")
	}

	// Key should be a 64-character hex string (SHA-256).
	if len(key1) != 64 {
		t.Errorf("Key length: got %d, want 64", len(key1))
	}
}

func TestDiskCache_CorruptedFile(t *testing.T) {
	cacheDir := t.TempDir()
	cache, err := NewDiskCache(cacheDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewDiskCache failed: %v", err)
	}

	testURL := "http://data.europa.eu/eli/reg/2016/679/oj"
	cacheFilePath := cache.pathFor(testURL)

	// Write corrupted JSON.
	if err := os.WriteFile(cacheFilePath, []byte("not valid json"), 0o644); err != nil {
		t.Fatalf("Failed to write corrupted file: %v", err)
	}

	_, found := cache.Get(testURL)
	if found {
		t.Error("Get returned found for corrupted cache file")
	}
}
