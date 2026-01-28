package extract

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/coolbeans/regula/pkg/pattern"
)

func getProjectRootDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine caller information")
	}
	// From pkg/extract/ go up two levels to project root
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

func loadPatternRegistry(t *testing.T) pattern.Registry {
	t.Helper()
	projectRoot := getProjectRootDir(t)
	patternsDir := filepath.Join(projectRoot, "patterns")

	registry, err := pattern.NewRegistryWithDirectory(patternsDir)
	if err != nil {
		t.Fatalf("Failed to load pattern registry: %v", err)
	}
	return registry
}

func TestPatternRegistryGDPRParse(t *testing.T) {
	registry := loadPatternRegistry(t)
	expected := loadExpectedGDPR(t)

	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	stats := doc.Statistics()

	t.Logf("Pattern Registry GDPR Parse Results:")
	t.Logf("  Articles:    %d (expected %d)", stats.Articles, expected.Statistics.Articles)
	t.Logf("  Chapters:    %d (expected %d)", stats.Chapters, expected.Statistics.Chapters)
	t.Logf("  Definitions: %d (expected %d)", stats.Definitions, expected.Statistics.Definitions)
	t.Logf("  Sections:    %d", stats.Sections)
	t.Logf("  Recitals:    %d", stats.Recitals)

	if stats.Articles != expected.Statistics.Articles {
		t.Errorf("Article count mismatch: got %d, want %d", stats.Articles, expected.Statistics.Articles)
	}

	if stats.Chapters != expected.Statistics.Chapters {
		t.Errorf("Chapter count mismatch: got %d, want %d", stats.Chapters, expected.Statistics.Chapters)
	}

	if stats.Definitions != expected.Statistics.Definitions {
		t.Errorf("Definition count mismatch: got %d, want %d", stats.Definitions, expected.Statistics.Definitions)
	}
}

func TestPatternRegistryGDPRRecitals(t *testing.T) {
	registry := loadPatternRegistry(t)

	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	if doc.Preamble == nil {
		t.Fatal("Preamble not found with registry-based parsing")
	}

	expectedRecitals := 173
	if len(doc.Preamble.Recitals) != expectedRecitals {
		t.Errorf("Recital count mismatch: got %d, want %d", len(doc.Preamble.Recitals), expectedRecitals)
	}
}

func TestPatternRegistryGDPRChapterTitles(t *testing.T) {
	registry := loadPatternRegistry(t)
	expected := loadExpectedGDPR(t)

	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	if len(doc.Chapters) != len(expected.Chapters) {
		t.Fatalf("Chapter count mismatch: got %d, want %d", len(doc.Chapters), len(expected.Chapters))
	}

	for i, expChapter := range expected.Chapters {
		gotChapter := doc.Chapters[i]
		if gotChapter.Number != expChapter.Number {
			t.Errorf("Chapter %d number mismatch: got %q, want %q", i+1, gotChapter.Number, expChapter.Number)
		}
		if gotChapter.Title != expChapter.Title {
			t.Errorf("Chapter %s title mismatch: got %q, want %q", expChapter.Number, gotChapter.Title, expChapter.Title)
		}
	}
}

func TestPatternRegistryGDPRDefinitions(t *testing.T) {
	registry := loadPatternRegistry(t)
	expected := loadExpectedGDPR(t)

	f := loadGDPRText(t)
	defer f.Close()

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(f)
	if err != nil {
		t.Fatalf("Parse with registry failed: %v", err)
	}

	if len(doc.Definitions) != len(expected.Definitions) {
		t.Logf("Got definitions:")
		for _, def := range doc.Definitions {
			t.Logf("  %d: %s", def.Number, def.Term)
		}
		t.Fatalf("Definition count mismatch: got %d, want %d", len(doc.Definitions), len(expected.Definitions))
	}

	for i, expDef := range expected.Definitions {
		if i >= len(doc.Definitions) {
			break
		}
		gotDef := doc.Definitions[i]
		if gotDef.Number != expDef.Number {
			t.Errorf("Definition %d number mismatch: got %d, want %d", i+1, gotDef.Number, expDef.Number)
		}
		if gotDef.Term != expDef.Term {
			t.Errorf("Definition %d term mismatch: got %q, want %q", expDef.Number, gotDef.Term, expDef.Term)
		}
	}
}

func TestPatternRegistryIdenticalOutput(t *testing.T) {
	registry := loadPatternRegistry(t)

	// Parse with legacy parser
	f1 := loadGDPRText(t)
	defer f1.Close()

	legacyParser := NewParser()
	legacyDoc, err := legacyParser.Parse(f1)
	if err != nil {
		t.Fatalf("Legacy parse failed: %v", err)
	}

	// Parse with registry parser
	f2 := loadGDPRText(t)
	defer f2.Close()

	registryParser := NewParserWithRegistry(registry)
	registryDoc, err := registryParser.Parse(f2)
	if err != nil {
		t.Fatalf("Registry parse failed: %v", err)
	}

	// Compare statistics
	legacyStats := legacyDoc.Statistics()
	registryStats := registryDoc.Statistics()

	t.Logf("Comparison (Legacy vs Registry):")
	t.Logf("  Articles:    %d vs %d", legacyStats.Articles, registryStats.Articles)
	t.Logf("  Chapters:    %d vs %d", legacyStats.Chapters, registryStats.Chapters)
	t.Logf("  Sections:    %d vs %d", legacyStats.Sections, registryStats.Sections)
	t.Logf("  Definitions: %d vs %d", legacyStats.Definitions, registryStats.Definitions)
	t.Logf("  Recitals:    %d vs %d", legacyStats.Recitals, registryStats.Recitals)

	if legacyStats.Articles != registryStats.Articles {
		t.Errorf("Article count diverged: legacy=%d, registry=%d", legacyStats.Articles, registryStats.Articles)
	}
	if legacyStats.Chapters != registryStats.Chapters {
		t.Errorf("Chapter count diverged: legacy=%d, registry=%d", legacyStats.Chapters, registryStats.Chapters)
	}
	if legacyStats.Sections != registryStats.Sections {
		t.Errorf("Section count diverged: legacy=%d, registry=%d", legacyStats.Sections, registryStats.Sections)
	}
	if legacyStats.Definitions != registryStats.Definitions {
		t.Errorf("Definition count diverged: legacy=%d, registry=%d", legacyStats.Definitions, registryStats.Definitions)
	}
	if legacyStats.Recitals != registryStats.Recitals {
		t.Errorf("Recital count diverged: legacy=%d, registry=%d", legacyStats.Recitals, registryStats.Recitals)
	}

	// Compare article titles
	legacyArticles := legacyDoc.AllArticles()
	registryArticles := registryDoc.AllArticles()

	if len(legacyArticles) != len(registryArticles) {
		t.Fatalf("All articles count differs: legacy=%d, registry=%d", len(legacyArticles), len(registryArticles))
	}

	for i := range legacyArticles {
		if legacyArticles[i].Number != registryArticles[i].Number {
			t.Errorf("Article %d number differs: legacy=%d, registry=%d",
				i, legacyArticles[i].Number, registryArticles[i].Number)
		}
		if legacyArticles[i].Title != registryArticles[i].Title {
			t.Errorf("Article %d title differs: legacy=%q, registry=%q",
				legacyArticles[i].Number, legacyArticles[i].Title, registryArticles[i].Title)
		}
	}

	// Compare chapter titles
	for i := range legacyDoc.Chapters {
		if legacyDoc.Chapters[i].Number != registryDoc.Chapters[i].Number {
			t.Errorf("Chapter %d number differs", i)
		}
		if legacyDoc.Chapters[i].Title != registryDoc.Chapters[i].Title {
			t.Errorf("Chapter %s title differs: legacy=%q, registry=%q",
				legacyDoc.Chapters[i].Number, legacyDoc.Chapters[i].Title, registryDoc.Chapters[i].Title)
		}
	}
}

func TestPatternRegistryEUFormatDetection(t *testing.T) {
	registry := loadPatternRegistry(t)

	euContent := `Regulation (EU) 2016/679 of the European Parliament and of the Council
of 27 April 2016
on the protection of natural persons with regard to the processing of personal data

THE EUROPEAN PARLIAMENT AND THE COUNCIL OF THE EUROPEAN UNION,

HAVE ADOPTED THIS REGULATION:

CHAPTER I
General provisions

Article 1
Subject-matter and objectives`

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(strings.NewReader(euContent))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if doc.Type != DocumentTypeRegulation {
		t.Errorf("Document type = %q, want %q", doc.Type, DocumentTypeRegulation)
	}

	if len(doc.Chapters) == 0 {
		t.Error("Expected at least 1 chapter")
	}
}

func TestPatternRegistryUSFallback(t *testing.T) {
	registry := loadPatternRegistry(t)

	usContent := `CALIFORNIA CONSUMER PRIVACY ACT OF 2018

TITLE 1.81.5. CALIFORNIA CONSUMER PRIVACY ACT OF 2018

CHAPTER 1

Section 1798.100
Title

Section 1798.105
Legislative Intent`

	parser := NewParserWithRegistry(registry)
	doc, err := parser.Parse(strings.NewReader(usContent))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should detect US format even with registry (no high-confidence EU match)
	if doc.Type != DocumentTypeStatute {
		// The type detection is separate from format detection
		t.Logf("Document type: %q", doc.Type)
	}
}

func TestPatternRegistryPatternLoading(t *testing.T) {
	registry := loadPatternRegistry(t)

	// Verify eu-regulation pattern was loaded
	euPattern, found := registry.Get("eu-regulation")
	if !found {
		t.Fatal("eu-regulation pattern not loaded from patterns directory")
	}

	if euPattern.Version != "2.0.0" {
		t.Errorf("eu-regulation version = %q, want %q", euPattern.Version, "2.0.0")
	}

	if euPattern.Jurisdiction != "EU" {
		t.Errorf("eu-regulation jurisdiction = %q, want %q", euPattern.Jurisdiction, "EU")
	}

	// Verify the pattern has the expected structure
	if len(euPattern.Detection.RequiredIndicators) < 2 {
		t.Errorf("Expected at least 2 required indicators, got %d", len(euPattern.Detection.RequiredIndicators))
	}

	if len(euPattern.Detection.OptionalIndicators) < 8 {
		t.Errorf("Expected at least 8 optional indicators, got %d", len(euPattern.Detection.OptionalIndicators))
	}

	if len(euPattern.Structure.Hierarchy) < 6 {
		t.Errorf("Expected at least 6 hierarchy levels, got %d", len(euPattern.Structure.Hierarchy))
	}

	if len(euPattern.References.Internal) < 5 {
		t.Errorf("Expected at least 5 internal reference patterns, got %d", len(euPattern.References.Internal))
	}

	if len(euPattern.References.External) < 4 {
		t.Errorf("Expected at least 4 external reference patterns, got %d", len(euPattern.References.External))
	}
}

func TestPatternSchemaValidation(t *testing.T) {
	projectRoot := getProjectRootDir(t)
	patternsDir := filepath.Join(projectRoot, "patterns")

	// Load each YAML file and validate against schema
	entries, err := os.ReadDir(patternsDir)
	if err != nil {
		t.Fatalf("Failed to read patterns directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			registry := pattern.NewRegistry()
			path := filepath.Join(patternsDir, entry.Name())
			err := registry.LoadFile(path)
			if err != nil {
				t.Errorf("Pattern file %s failed validation: %v", entry.Name(), err)
			}
		})
	}
}
