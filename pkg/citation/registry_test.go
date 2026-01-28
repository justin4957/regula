package citation

import (
	"sync"
	"testing"
)

// mockCitationParser is a test double implementing CitationParser.
type mockCitationParser struct {
	name           string
	jurisdictions  []string
	parseResults   []*Citation
	parseErr       error
	normalizeValue string
	toURIValue     string
	toURIErr       error
}

func (m *mockCitationParser) Name() string            { return m.name }
func (m *mockCitationParser) Jurisdictions() []string  { return m.jurisdictions }
func (m *mockCitationParser) Parse(text string) ([]*Citation, error) {
	return m.parseResults, m.parseErr
}
func (m *mockCitationParser) Normalize(citation *Citation) string { return m.normalizeValue }
func (m *mockCitationParser) ToURI(citation *Citation) (string, error) {
	return m.toURIValue, m.toURIErr
}

func TestNewCitationRegistry(t *testing.T) {
	registry := NewCitationRegistry()
	if registry == nil {
		t.Fatal("NewCitationRegistry returned nil")
	}
	if registry.Count() != 0 {
		t.Errorf("Expected 0 parsers, got %d", registry.Count())
	}
	if names := registry.List(); len(names) != 0 {
		t.Errorf("Expected empty list, got %v", names)
	}
}

func TestCitationRegistryRegister(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		registry := NewCitationRegistry()
		parser := &mockCitationParser{name: "TestParser", jurisdictions: []string{"EU"}}
		if err := registry.Register(parser); err != nil {
			t.Fatalf("Register failed: %v", err)
		}
		if registry.Count() != 1 {
			t.Errorf("Expected 1 parser, got %d", registry.Count())
		}
	})

	t.Run("duplicate_rejected", func(t *testing.T) {
		registry := NewCitationRegistry()
		parser := &mockCitationParser{name: "TestParser", jurisdictions: []string{"EU"}}
		_ = registry.Register(parser)
		err := registry.Register(parser)
		if err == nil {
			t.Error("Expected error for duplicate registration")
		}
	})

	t.Run("nil_rejected", func(t *testing.T) {
		registry := NewCitationRegistry()
		err := registry.Register(nil)
		if err == nil {
			t.Error("Expected error for nil parser")
		}
	})

	t.Run("empty_name_rejected", func(t *testing.T) {
		registry := NewCitationRegistry()
		parser := &mockCitationParser{name: "", jurisdictions: []string{"EU"}}
		err := registry.Register(parser)
		if err == nil {
			t.Error("Expected error for empty parser name")
		}
	})
}

func TestCitationRegistryUnregister(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		registry := NewCitationRegistry()
		parser := &mockCitationParser{name: "TestParser", jurisdictions: []string{"EU"}}
		_ = registry.Register(parser)
		if err := registry.Unregister("TestParser"); err != nil {
			t.Fatalf("Unregister failed: %v", err)
		}
		if registry.Count() != 0 {
			t.Errorf("Expected 0 parsers after unregister, got %d", registry.Count())
		}
	})

	t.Run("not_found", func(t *testing.T) {
		registry := NewCitationRegistry()
		err := registry.Unregister("NonExistent")
		if err == nil {
			t.Error("Expected error for unregistering nonexistent parser")
		}
	})

	t.Run("jurisdiction_index_cleaned", func(t *testing.T) {
		registry := NewCitationRegistry()
		parser := &mockCitationParser{name: "TestParser", jurisdictions: []string{"EU", "UK"}}
		_ = registry.Register(parser)
		_ = registry.Unregister("TestParser")

		euParsers := registry.ListByJurisdiction("EU")
		if len(euParsers) != 0 {
			t.Errorf("Expected no EU parsers after unregister, got %v", euParsers)
		}
		ukParsers := registry.ListByJurisdiction("UK")
		if len(ukParsers) != 0 {
			t.Errorf("Expected no UK parsers after unregister, got %v", ukParsers)
		}
	})
}

func TestCitationRegistryGet(t *testing.T) {
	registry := NewCitationRegistry()
	parser := &mockCitationParser{name: "TestParser", jurisdictions: []string{"EU"}}
	_ = registry.Register(parser)

	t.Run("found", func(t *testing.T) {
		retrieved, ok := registry.Get("TestParser")
		if !ok {
			t.Fatal("Expected parser to be found")
		}
		if retrieved.Name() != "TestParser" {
			t.Errorf("Expected name TestParser, got %s", retrieved.Name())
		}
	})

	t.Run("not_found", func(t *testing.T) {
		_, ok := registry.Get("NonExistent")
		if ok {
			t.Error("Expected parser not to be found")
		}
	})
}

func TestCitationRegistryList(t *testing.T) {
	registry := NewCitationRegistry()
	_ = registry.Register(&mockCitationParser{name: "Bravo", jurisdictions: []string{"US"}})
	_ = registry.Register(&mockCitationParser{name: "Alpha", jurisdictions: []string{"EU"}})
	_ = registry.Register(&mockCitationParser{name: "Charlie", jurisdictions: []string{"UK"}})

	names := registry.List()
	if len(names) != 3 {
		t.Fatalf("Expected 3 names, got %d", len(names))
	}
	// Verify sorted order.
	if names[0] != "Alpha" || names[1] != "Bravo" || names[2] != "Charlie" {
		t.Errorf("Expected sorted [Alpha, Bravo, Charlie], got %v", names)
	}
}

func TestCitationRegistryListByJurisdiction(t *testing.T) {
	registry := NewCitationRegistry()
	_ = registry.Register(&mockCitationParser{name: "EUParser", jurisdictions: []string{"EU"}})
	_ = registry.Register(&mockCitationParser{name: "USParser", jurisdictions: []string{"US"}})
	_ = registry.Register(&mockCitationParser{name: "MultiParser", jurisdictions: []string{"EU", "US"}})

	t.Run("EU_parsers", func(t *testing.T) {
		euParsers := registry.ListByJurisdiction("EU")
		if len(euParsers) != 2 {
			t.Errorf("Expected 2 EU parsers, got %d: %v", len(euParsers), euParsers)
		}
	})

	t.Run("US_parsers", func(t *testing.T) {
		usParsers := registry.ListByJurisdiction("US")
		if len(usParsers) != 2 {
			t.Errorf("Expected 2 US parsers, got %d: %v", len(usParsers), usParsers)
		}
	})

	t.Run("case_insensitive", func(t *testing.T) {
		euParsers := registry.ListByJurisdiction("eu")
		if len(euParsers) != 2 {
			t.Errorf("Expected case-insensitive match, got %d parsers", len(euParsers))
		}
	})

	t.Run("no_match", func(t *testing.T) {
		parsers := registry.ListByJurisdiction("JP")
		if len(parsers) != 0 {
			t.Errorf("Expected 0 parsers for JP, got %d", len(parsers))
		}
	})
}

func TestCitationRegistryParseAll(t *testing.T) {
	t.Run("merges_results_from_multiple_parsers", func(t *testing.T) {
		registry := NewCitationRegistry()
		parser1 := &mockCitationParser{
			name:          "Parser1",
			jurisdictions: []string{"EU"},
			parseResults: []*Citation{
				{RawText: "Article 5", TextOffset: 10, TextLength: 9, Confidence: 0.9},
			},
		}
		parser2 := &mockCitationParser{
			name:          "Parser2",
			jurisdictions: []string{"EU"},
			parseResults: []*Citation{
				{RawText: "Article 17", TextOffset: 50, TextLength: 10, Confidence: 0.8},
			},
		}
		_ = registry.Register(parser1)
		_ = registry.Register(parser2)

		results := registry.ParseAll("some text with Article 5 and Article 17", "")
		if len(results) != 2 {
			t.Errorf("Expected 2 citations, got %d", len(results))
		}
	})

	t.Run("jurisdiction_filter", func(t *testing.T) {
		registry := NewCitationRegistry()
		euParser := &mockCitationParser{
			name:          "EUParser",
			jurisdictions: []string{"EU"},
			parseResults: []*Citation{
				{RawText: "Directive 95/46/EC", TextOffset: 0, TextLength: 18, Confidence: 1.0},
			},
		}
		usParser := &mockCitationParser{
			name:          "USParser",
			jurisdictions: []string{"US"},
			parseResults: []*Citation{
				{RawText: "15 U.S.C. Section 1681", TextOffset: 30, TextLength: 22, Confidence: 1.0},
			},
		}
		_ = registry.Register(euParser)
		_ = registry.Register(usParser)

		euResults := registry.ParseAll("text", "EU")
		if len(euResults) != 1 {
			t.Errorf("Expected 1 EU citation, got %d", len(euResults))
		}
		if len(euResults) > 0 && euResults[0].RawText != "Directive 95/46/EC" {
			t.Errorf("Expected EU citation, got %q", euResults[0].RawText)
		}
	})

	t.Run("deduplication_by_overlap", func(t *testing.T) {
		registry := NewCitationRegistry()
		parser1 := &mockCitationParser{
			name:          "HighConf",
			jurisdictions: []string{"EU"},
			parseResults: []*Citation{
				{RawText: "Article 6(1)(a)", TextOffset: 10, TextLength: 15, Confidence: 1.0},
			},
		}
		parser2 := &mockCitationParser{
			name:          "LowConf",
			jurisdictions: []string{"EU"},
			parseResults: []*Citation{
				{RawText: "Article 6", TextOffset: 10, TextLength: 9, Confidence: 0.5},
			},
		}
		_ = registry.Register(parser1)
		_ = registry.Register(parser2)

		results := registry.ParseAll("text with Article 6(1)(a)", "")
		if len(results) != 1 {
			t.Errorf("Expected 1 deduplicated citation, got %d", len(results))
		}
		if len(results) > 0 && results[0].Confidence != 1.0 {
			t.Errorf("Expected highest confidence (1.0), got %f", results[0].Confidence)
		}
	})

	t.Run("sorted_by_offset", func(t *testing.T) {
		registry := NewCitationRegistry()
		parser := &mockCitationParser{
			name:          "TestParser",
			jurisdictions: []string{"EU"},
			parseResults: []*Citation{
				{RawText: "Article 17", TextOffset: 50, TextLength: 10, Confidence: 0.9},
				{RawText: "Article 5", TextOffset: 10, TextLength: 9, Confidence: 0.9},
			},
		}
		_ = registry.Register(parser)

		results := registry.ParseAll("text", "")
		if len(results) < 2 {
			t.Fatalf("Expected 2 results, got %d", len(results))
		}
		if results[0].TextOffset > results[1].TextOffset {
			t.Error("Results should be sorted by TextOffset ascending")
		}
	})

	t.Run("empty_text", func(t *testing.T) {
		registry := NewCitationRegistry()
		parser := &mockCitationParser{
			name:          "TestParser",
			jurisdictions: []string{"EU"},
			parseResults:  []*Citation{},
		}
		_ = registry.Register(parser)

		results := registry.ParseAll("", "")
		if results != nil && len(results) != 0 {
			t.Errorf("Expected empty results for empty text, got %d", len(results))
		}
	})
}

func TestCitationRegistryConcurrentAccess(t *testing.T) {
	registry := NewCitationRegistry()

	// Pre-register a parser for reads.
	baseParser := &mockCitationParser{
		name:          "BaseParser",
		jurisdictions: []string{"EU"},
		parseResults: []*Citation{
			{RawText: "Article 1", TextOffset: 0, TextLength: 9, Confidence: 0.9},
		},
	}
	_ = registry.Register(baseParser)

	var wg sync.WaitGroup
	goroutineCount := 20

	// Concurrent reads.
	for i := 0; i < goroutineCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = registry.List()
			_ = registry.ListByJurisdiction("EU")
			_ = registry.ParseAll("Article 1", "")
			_ = registry.Count()
		}()
	}

	// Concurrent writes.
	for i := 0; i < goroutineCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			parserName := "ConcurrentParser" + string(rune('A'+idx%26))
			parser := &mockCitationParser{
				name:          parserName,
				jurisdictions: []string{"EU"},
			}
			// Ignore errors (duplicates are expected).
			_ = registry.Register(parser)
		}(i)
	}

	wg.Wait()

	// Verify registry is in a consistent state.
	count := registry.Count()
	if count < 1 {
		t.Errorf("Expected at least 1 parser after concurrent access, got %d", count)
	}
}

func TestCitationsOverlap(t *testing.T) {
	cases := []struct {
		name     string
		a, b     *Citation
		expected bool
	}{
		{
			name:     "no_overlap",
			a:        &Citation{TextOffset: 0, TextLength: 10},
			b:        &Citation{TextOffset: 20, TextLength: 10},
			expected: false,
		},
		{
			name:     "full_overlap",
			a:        &Citation{TextOffset: 5, TextLength: 10},
			b:        &Citation{TextOffset: 5, TextLength: 10},
			expected: true,
		},
		{
			name:     "partial_overlap",
			a:        &Citation{TextOffset: 5, TextLength: 10},
			b:        &Citation{TextOffset: 10, TextLength: 10},
			expected: true,
		},
		{
			name:     "adjacent_no_overlap",
			a:        &Citation{TextOffset: 0, TextLength: 10},
			b:        &Citation{TextOffset: 10, TextLength: 10},
			expected: false,
		},
		{
			name:     "no_position_same_text",
			a:        &Citation{RawText: "Article 5"},
			b:        &Citation{RawText: "Article 5"},
			expected: true,
		},
		{
			name:     "no_position_different_text",
			a:        &Citation{RawText: "Article 5"},
			b:        &Citation{RawText: "Article 6"},
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := citationsOverlap(tc.a, tc.b)
			if result != tc.expected {
				t.Errorf("Expected overlap=%v, got %v", tc.expected, result)
			}
		})
	}
}
