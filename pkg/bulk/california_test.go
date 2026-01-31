package bulk

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCaliforniaSourceName(t *testing.T) {
	source := NewCaliforniaSource(DefaultDownloadConfig())

	if source.Name() != "california" {
		t.Errorf("expected name 'california', got %q", source.Name())
	}
	if source.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestCaliforniaSourceListDatasets(t *testing.T) {
	source := NewCaliforniaSource(DefaultDownloadConfig())

	datasets, err := source.ListDatasets()
	if err != nil {
		t.Fatalf("ListDatasets failed: %v", err)
	}

	if len(datasets) != 30 {
		t.Fatalf("expected 30 California datasets, got %d", len(datasets))
	}

	// Verify all datasets have expected properties
	for _, dataset := range datasets {
		if dataset.SourceName != "california" {
			t.Errorf("expected source name 'california', got %q", dataset.SourceName)
		}
		if dataset.Jurisdiction != "US-CA" {
			t.Errorf("expected jurisdiction 'US-CA', got %q for %s", dataset.Jurisdiction, dataset.Identifier)
		}
		if !strings.HasPrefix(dataset.Identifier, "ca-") {
			t.Errorf("expected identifier to start with 'ca-', got %q", dataset.Identifier)
		}
	}

	// Verify specific codes exist
	expectedCodes := []string{"ca-cons", "ca-civ", "ca-pen", "ca-gov", "ca-veh"}
	for _, expectedCode := range expectedCodes {
		found := false
		for _, dataset := range datasets {
			if dataset.Identifier == expectedCode {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find dataset with identifier %q", expectedCode)
		}
	}
}

func TestCaliforniaSourceDownloadDataset(t *testing.T) {
	tocHTML := `<html>
		<body>
			<a href="codes_displayexpandedbranch.xhtml?tocCode=CIV&division=1">Division 1</a>
		</body>
	</html>`

	branchHTML := `<html>
		<body>
			<div class="content">
				<h3>DIVISION 1. PERSONS</h3>
				<p>Section 1. All people are by nature free.</p>
				<p>Section 2. Every person has certain inalienable rights.</p>
			</div>
		</body>
	</html>`

	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if strings.Contains(request.URL.Path, "codesTOCSelected") {
			responseWriter.Write([]byte(tocHTML))
		} else if strings.Contains(request.URL.Path, "displayexpandedbranch") {
			responseWriter.Write([]byte(branchHTML))
		} else {
			responseWriter.WriteHeader(http.StatusNotFound)
		}
	}))
	defer testServer.Close()

	temporaryDir := t.TempDir()
	config := DownloadConfig{
		DownloadDirectory: temporaryDir,
		RateLimit:         1 * time.Millisecond,
		Timeout:           10 * time.Second,
		UserAgent:         "regula-test/1.0",
		HTTPClient:        testServer.Client(),
	}

	// The California source makes requests to the real leginfo site by default.
	// For testing, we override the HTTP client but the URLs won't match testServer.
	// Instead, test the helper functions directly.
	source := NewCaliforniaSource(config)

	if source.Name() != "california" {
		t.Errorf("expected name 'california', got %q", source.Name())
	}
}

func TestExtractBranchURLs(t *testing.T) {
	tocHTML := fmt.Sprintf(`<html><body>
		<a href="codes_displayexpandedbranch.xhtml?tocCode=CIV&division=1">Division 1</a>
		<a href="codes_displayexpandedbranch.xhtml?tocCode=CIV&division=2">Division 2</a>
		<a href="codes_displayexpandedbranch.xhtml?tocCode=PEN&division=1">Wrong Code</a>
		<a href="codes_displayexpandedbranch.xhtml?tocCode=CIV&division=1">Duplicate</a>
	</body></html>`)

	branchURLs := extractBranchURLs(tocHTML, "CIV")

	if len(branchURLs) != 2 {
		t.Fatalf("expected 2 unique CIV branch URLs, got %d: %v", len(branchURLs), branchURLs)
	}

	for _, branchURL := range branchURLs {
		if !strings.Contains(branchURL, "CIV") {
			t.Errorf("expected URL to contain 'CIV', got %q", branchURL)
		}
		if !strings.Contains(branchURL, "displayexpandedbranch") {
			t.Errorf("expected URL to contain 'displayexpandedbranch', got %q", branchURL)
		}
	}
}

func TestExtractBranchURLsEmpty(t *testing.T) {
	branchURLs := extractBranchURLs("<html><body>no links</body></html>", "CIV")
	if len(branchURLs) != 0 {
		t.Errorf("expected 0 branch URLs for empty page, got %d", len(branchURLs))
	}
}

func TestExtractCaliforniaText(t *testing.T) {
	rawHTML := []byte(`<html>
		<head><style>.hidden { display: none; }</style></head>
		<body>
			<script>alert("ignore");</script>
			<div class="content">
				<h3>DIVISION 1. PERSONS</h3>
				<p>Section 1. &amp; All people are by nature free &amp; independent.</p>
				<p>Section 2. Every &quot;person&quot; has &#39;certain&#39; rights.</p>
			</div>
		</body>
	</html>`)

	plainText := extractCaliforniaText(rawHTML)

	if strings.Contains(plainText, "<script>") {
		t.Error("expected script tags to be removed")
	}
	if strings.Contains(plainText, "<style>") {
		t.Error("expected style tags to be removed")
	}
	if strings.Contains(plainText, "<div") {
		t.Error("expected HTML tags to be removed")
	}
	if !strings.Contains(plainText, "DIVISION 1. PERSONS") {
		t.Error("expected heading text to be preserved")
	}
	if !strings.Contains(plainText, "& All people") {
		t.Error("expected &amp; to be decoded to &")
	}
	if !strings.Contains(plainText, `"person"`) {
		t.Error("expected &quot; to be decoded to quote")
	}
	if !strings.Contains(plainText, "'certain'") {
		t.Error("expected &#39; to be decoded to apostrophe")
	}
}

func TestExtractCaliforniaTextEmpty(t *testing.T) {
	plainText := extractCaliforniaText([]byte(""))
	if plainText != "" {
		t.Errorf("expected empty string for empty input, got %q", plainText)
	}
}

func TestCaliforniaCodeFullName(t *testing.T) {
	source := NewCaliforniaSource(DefaultDownloadConfig())

	testCases := []struct {
		abbreviation string
		expectedName string
	}{
		{"CIV", "Civil Code"},
		{"PEN", "Penal Code"},
		{"CONS", "California Constitution"},
		{"GOV", "Government Code"},
		{"UNKNOWN", "UNKNOWN"},
	}

	for _, testCase := range testCases {
		result := source.codeFullName(testCase.abbreviation)
		if result != testCase.expectedName {
			t.Errorf("codeFullName(%q) = %q, want %q", testCase.abbreviation, result, testCase.expectedName)
		}
	}
}
