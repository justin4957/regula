package extract

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var updateGoldenFiles = flag.Bool("update", false, "update golden files in testdata/corpus/")

// CorpusManifest represents the top-level corpus configuration.
type CorpusManifest struct {
	Version     string        `json:"version"`
	Description string        `json:"description"`
	Entries     []CorpusEntry `json:"entries"`
}

// CorpusEntry represents a single document in the test corpus.
type CorpusEntry struct {
	ID           string `json:"id"`
	Jurisdiction string `json:"jurisdiction"`
	ShortName    string `json:"short_name"`
	FullName     string `json:"full_name"`
	Format       string `json:"format"`
	SourcePath   string `json:"source_path"`
	ExpectedPath string `json:"expected_path"`
	SourceInfo   string `json:"source_info"`
	UseRegistry  bool   `json:"use_registry"`
}

// CorpusExpected is the golden file format for all jurisdictions.
type CorpusExpected struct {
	Metadata   CorpusMetadata `json:"metadata"`
	Statistics Statistics     `json:"statistics"`
	Document   *Document      `json:"document"`
}

// CorpusMetadata provides traceability for golden files.
type CorpusMetadata struct {
	CorpusID     string `json:"corpus_id"`
	Jurisdiction string `json:"jurisdiction"`
	ShortName    string `json:"short_name"`
	GeneratedAt  string `json:"generated_at"`
}

func loadCorpusManifest(t *testing.T) (*CorpusManifest, string) {
	t.Helper()
	corpusDir := filepath.Join(getProjectRootDir(t), "testdata", "corpus")
	manifestPath := filepath.Join(corpusDir, "manifest.json")

	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("Failed to read corpus manifest: %v", err)
	}

	var manifest CorpusManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("Failed to parse corpus manifest: %v", err)
	}

	return &manifest, corpusDir
}

func TestCorpusGoldenFiles(t *testing.T) {
	manifest, corpusDir := loadCorpusManifest(t)
	registry := loadPatternRegistry(t)

	if len(manifest.Entries) < 10 {
		t.Errorf("Corpus must have at least 10 entries, got %d", len(manifest.Entries))
	}

	for _, entry := range manifest.Entries {
		t.Run(entry.ID, func(t *testing.T) {
			sourcePath := filepath.Join(corpusDir, entry.SourcePath)
			expectedPath := filepath.Join(corpusDir, entry.ExpectedPath)

			// Parse source document
			sourceFile, err := os.Open(sourcePath)
			if err != nil {
				t.Fatalf("Failed to open source %s: %v", sourcePath, err)
			}
			defer sourceFile.Close()

			var parser *Parser
			if entry.UseRegistry {
				parser = NewParserWithRegistry(registry)
			} else {
				parser = NewParser()
			}

			document, err := parser.Parse(sourceFile)
			if err != nil {
				t.Fatalf("Parse failed for %s: %v", entry.ID, err)
			}

			actualStatistics := document.Statistics()

			// Update mode: write golden file
			if *updateGoldenFiles {
				goldenOutput := CorpusExpected{
					Metadata: CorpusMetadata{
						CorpusID:     entry.ID,
						Jurisdiction: entry.Jurisdiction,
						ShortName:    entry.ShortName,
						GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
					},
					Statistics: actualStatistics,
					Document:   document,
				}
				goldenJSON, err := json.MarshalIndent(goldenOutput, "", "  ")
				if err != nil {
					t.Fatalf("Failed to marshal golden file: %v", err)
				}
				if err := os.WriteFile(expectedPath, goldenJSON, 0644); err != nil {
					t.Fatalf("Failed to write golden file %s: %v", expectedPath, err)
				}
				t.Logf("Updated golden file: %s", expectedPath)
				return
			}

			// Compare mode: load and validate against golden file
			expectedData, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("Failed to read golden file (run with -update to generate): %v", err)
			}

			var expected CorpusExpected
			if err := json.Unmarshal(expectedData, &expected); err != nil {
				t.Fatalf("Failed to parse golden file: %v", err)
			}

			// Validate statistics
			if actualStatistics.Chapters != expected.Statistics.Chapters {
				t.Errorf("Chapters: got %d, want %d", actualStatistics.Chapters, expected.Statistics.Chapters)
			}
			if actualStatistics.Sections != expected.Statistics.Sections {
				t.Errorf("Sections: got %d, want %d", actualStatistics.Sections, expected.Statistics.Sections)
			}
			if actualStatistics.Articles != expected.Statistics.Articles {
				t.Errorf("Articles: got %d, want %d", actualStatistics.Articles, expected.Statistics.Articles)
			}
			if actualStatistics.Definitions != expected.Statistics.Definitions {
				t.Errorf("Definitions: got %d, want %d", actualStatistics.Definitions, expected.Statistics.Definitions)
			}
			if actualStatistics.Recitals != expected.Statistics.Recitals {
				t.Errorf("Recitals: got %d, want %d", actualStatistics.Recitals, expected.Statistics.Recitals)
			}

			// Validate document type
			if document.Type != expected.Document.Type {
				t.Errorf("Document type: got %q, want %q", document.Type, expected.Document.Type)
			}

			// Validate chapter structure
			if len(document.Chapters) != len(expected.Document.Chapters) {
				t.Errorf("Chapter count: got %d, want %d", len(document.Chapters), len(expected.Document.Chapters))
			} else {
				for i, expectedChapter := range expected.Document.Chapters {
					actualChapter := document.Chapters[i]
					if actualChapter.Number != expectedChapter.Number {
						t.Errorf("Chapter %d number: got %q, want %q", i, actualChapter.Number, expectedChapter.Number)
					}
					if actualChapter.Title != expectedChapter.Title {
						t.Errorf("Chapter %s title: got %q, want %q", expectedChapter.Number, actualChapter.Title, expectedChapter.Title)
					}
				}
			}

			// Validate definitions
			if len(document.Definitions) != len(expected.Document.Definitions) {
				t.Errorf("Definition count in document: got %d, want %d", len(document.Definitions), len(expected.Document.Definitions))
			} else {
				for i, expectedDefinition := range expected.Document.Definitions {
					actualDefinition := document.Definitions[i]
					if actualDefinition.Term != expectedDefinition.Term {
						t.Errorf("Definition %d term: got %q, want %q", i, actualDefinition.Term, expectedDefinition.Term)
					}
				}
			}

			t.Logf("[%s] %d chapters, %d sections, %d articles, %d definitions, %d recitals",
				entry.ID,
				actualStatistics.Chapters,
				actualStatistics.Sections,
				actualStatistics.Articles,
				actualStatistics.Definitions,
				actualStatistics.Recitals)
		})
	}
}

func TestCorpusManifestIntegrity(t *testing.T) {
	manifest, corpusDir := loadCorpusManifest(t)

	// Verify all source files exist
	for _, entry := range manifest.Entries {
		sourcePath := filepath.Join(corpusDir, entry.SourcePath)
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			t.Errorf("Source file missing for %s: %s", entry.ID, sourcePath)
		}
	}

	// Verify no duplicate IDs
	seenIDs := make(map[string]bool)
	for _, entry := range manifest.Entries {
		if seenIDs[entry.ID] {
			t.Errorf("Duplicate corpus ID: %s", entry.ID)
		}
		seenIDs[entry.ID] = true
	}

	// Verify minimum corpus size
	if len(manifest.Entries) < 10 {
		t.Errorf("Corpus must have at least 10 entries, got %d", len(manifest.Entries))
	}

	t.Logf("Corpus manifest: %d entries across jurisdictions", len(manifest.Entries))
}
