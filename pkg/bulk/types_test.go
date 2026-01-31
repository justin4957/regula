package bulk

import (
	"testing"
)

func TestDefaultDownloadConfig(t *testing.T) {
	config := DefaultDownloadConfig()

	if config.DownloadDirectory != ".regula/downloads" {
		t.Errorf("expected download dir '.regula/downloads', got %q", config.DownloadDirectory)
	}
	if config.RateLimit == 0 {
		t.Error("expected non-zero rate limit")
	}
	if config.Timeout == 0 {
		t.Error("expected non-zero timeout")
	}
	if config.UserAgent == "" {
		t.Error("expected non-empty user agent")
	}
	if config.CFRYear != "2024" {
		t.Errorf("expected CFR year '2024', got %q", config.CFRYear)
	}
}

func TestResolveSource(t *testing.T) {
	config := DefaultDownloadConfig()

	testCases := []struct {
		name         string
		sourceName   string
		expectedName string
		expectError  bool
	}{
		{
			name:         "uscode source",
			sourceName:   "uscode",
			expectedName: "uscode",
		},
		{
			name:         "cfr source",
			sourceName:   "cfr",
			expectedName: "cfr",
		},
		{
			name:         "california source",
			sourceName:   "california",
			expectedName: "california",
		},
		{
			name:         "archive source",
			sourceName:   "archive",
			expectedName: "archive",
		},
		{
			name:        "unknown source",
			sourceName:  "invalid",
			expectError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			source, err := ResolveSource(testCase.sourceName, config)

			if testCase.expectError {
				if err == nil {
					t.Error("expected error for unknown source")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if source.Name() != testCase.expectedName {
				t.Errorf("expected name %q, got %q", testCase.expectedName, source.Name())
			}
			if source.Description() == "" {
				t.Error("expected non-empty description")
			}
		})
	}
}

func TestAllSourceNames(t *testing.T) {
	sourceNames := AllSourceNames()

	if len(sourceNames) != 4 {
		t.Fatalf("expected 4 source names, got %d", len(sourceNames))
	}

	expectedNames := map[string]bool{
		"uscode":     false,
		"cfr":        false,
		"california": false,
		"archive":    false,
	}

	for _, name := range sourceNames {
		if _, exists := expectedNames[name]; !exists {
			t.Errorf("unexpected source name: %q", name)
		}
		expectedNames[name] = true
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("missing source name: %q", name)
		}
	}
}
