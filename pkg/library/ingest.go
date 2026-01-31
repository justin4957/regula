package library

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coolbeans/regula/pkg/extract"
	"github.com/coolbeans/regula/pkg/store"
)

const defaultBaseURI = "https://regula.dev/regulations/"

// IngestFromText runs the full extraction pipeline on source text and returns a
// populated TripleStore with extraction statistics. An optional formatHint
// (e.g., "us", "eu", "uk") bypasses automatic format detection.
func IngestFromText(sourceText []byte, documentID string, baseURI string, formatHint ...string) (*IngestResult, error) {
	if len(sourceText) == 0 {
		return nil, fmt.Errorf("source text is empty")
	}
	if documentID == "" {
		return nil, fmt.Errorf("document ID is required")
	}
	if baseURI == "" {
		baseURI = defaultBaseURI
	}

	regID := strings.ToUpper(documentID)

	// Step 1: Parse document structure
	parser := extract.NewParser()
	// Apply format hint if provided, bypassing auto-detection
	if len(formatHint) > 0 && formatHint[0] != "" {
		parser.SetFormatHint(extract.DocumentFormat(formatHint[0]))
	}
	reader := strings.NewReader(string(sourceText))
	doc, err := parser.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse document: %w", err)
	}

	// Step 2: Extract definitions
	defExtractor := extract.NewDefinitionExtractor()

	// Step 3: Extract cross-references
	refExtractor := extract.NewReferenceExtractor()

	// Step 4: Extract rights and obligations
	semExtractor := extract.NewSemanticExtractor()

	// Step 5: Resolve references
	resolver := extract.NewReferenceResolver(baseURI, regID)
	resolver.IndexDocument(doc)

	// Step 6: Build complete knowledge graph
	tripleStore := store.NewTripleStore()
	builder := store.NewGraphBuilder(tripleStore, baseURI)
	buildStats, err := builder.BuildComplete(doc, defExtractor, refExtractor, resolver, semExtractor)
	if err != nil {
		return nil, fmt.Errorf("failed to build graph: %w", err)
	}

	documentStats := &DocumentStats{
		TotalTriples: buildStats.TotalTriples,
		Articles:     buildStats.Articles,
		Chapters:     buildStats.Chapters,
		Sections:     buildStats.Sections,
		Recitals:     buildStats.Recitals,
		Definitions:  buildStats.Definitions,
		References:   buildStats.References,
		Rights:       buildStats.Rights,
		Obligations:  buildStats.Obligations,
		TermUsages:   buildStats.TermUsages,
		SourceBytes:  len(sourceText),
	}

	return &IngestResult{
		TripleStore: tripleStore,
		Stats:       documentStats,
		DocumentID:  documentID,
		RegID:       regID,
	}, nil
}

// IngestFromFile reads a file from disk and runs the ingestion pipeline.
func IngestFromFile(filePath string, documentID string, baseURI string) (*IngestResult, error) {
	sourceText, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	if documentID == "" {
		documentID = DeriveDocumentID(filePath)
	}

	return IngestFromText(sourceText, documentID, baseURI)
}

// DeriveDocumentID creates a document ID from a file path by lowercasing the basename
// without its extension.
func DeriveDocumentID(filePath string) string {
	baseName := filepath.Base(filePath)
	if idx := strings.LastIndex(baseName, "."); idx != -1 {
		baseName = baseName[:idx]
	}
	return strings.ToLower(baseName)
}
