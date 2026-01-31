package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIngestFromText_GDPR(t *testing.T) {
	sourceText, err := os.ReadFile(filepath.Join("..", "..", "testdata", "gdpr.txt"))
	if err != nil {
		t.Skipf("GDPR test data not available: %v", err)
	}

	result, err := IngestFromText(sourceText, "eu-gdpr", "")
	if err != nil {
		t.Fatalf("IngestFromText failed: %v", err)
	}

	if result.TripleStore == nil {
		t.Fatal("TripleStore is nil")
	}
	if result.TripleStore.Count() == 0 {
		t.Error("TripleStore is empty")
	}
	if result.Stats == nil {
		t.Fatal("Stats is nil")
	}
	if result.Stats.Articles == 0 {
		t.Error("expected articles > 0")
	}
	if result.Stats.TotalTriples == 0 {
		t.Error("expected triples > 0")
	}
	if result.DocumentID != "eu-gdpr" {
		t.Errorf("unexpected document ID: %s", result.DocumentID)
	}
}

func TestIngestFromText_CCPA(t *testing.T) {
	sourceText, err := os.ReadFile(filepath.Join("..", "..", "testdata", "ccpa.txt"))
	if err != nil {
		t.Skipf("CCPA test data not available: %v", err)
	}

	result, err := IngestFromText(sourceText, "us-ca-ccpa", "")
	if err != nil {
		t.Fatalf("IngestFromText failed: %v", err)
	}

	if result.Stats.Articles == 0 {
		t.Error("expected articles > 0 for CCPA")
	}
}

func TestIngestFromText_EmptyInput(t *testing.T) {
	_, err := IngestFromText([]byte{}, "test", "")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestIngestFromText_EmptyDocumentID(t *testing.T) {
	_, err := IngestFromText([]byte("some text"), "", "")
	if err == nil {
		t.Error("expected error for empty document ID")
	}
}

func TestIngestFromFile(t *testing.T) {
	gdprPath := filepath.Join("..", "..", "testdata", "gdpr.txt")
	if _, err := os.Stat(gdprPath); os.IsNotExist(err) {
		t.Skipf("GDPR test data not available")
	}

	result, err := IngestFromFile(gdprPath, "", "")
	if err != nil {
		t.Fatalf("IngestFromFile failed: %v", err)
	}

	if result.DocumentID != "gdpr" {
		t.Errorf("unexpected derived document ID: %s", result.DocumentID)
	}
	if result.Stats.TotalTriples == 0 {
		t.Error("expected triples > 0")
	}
}

func TestIngestFromFile_NotFound(t *testing.T) {
	_, err := IngestFromFile("/nonexistent/file.txt", "test", "")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestDeriveDocumentID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"testdata/gdpr.txt", "gdpr"},
		{"/path/to/ccpa.txt", "ccpa"},
		{"file.txt", "file"},
		{"no-extension", "no-extension"},
		{"path/to/EU-AI-ACT.txt", "eu-ai-act"},
	}

	for _, testCase := range tests {
		result := DeriveDocumentID(testCase.input)
		if result != testCase.expected {
			t.Errorf("DeriveDocumentID(%q) = %q, want %q", testCase.input, result, testCase.expected)
		}
	}
}
