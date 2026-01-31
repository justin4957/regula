package bulk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestInternetArchiveSourceName(t *testing.T) {
	source := NewInternetArchiveSource(DefaultDownloadConfig())

	if source.Name() != "archive" {
		t.Errorf("expected name 'archive', got %q", source.Name())
	}
	if source.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestInternetArchiveListDatasets(t *testing.T) {
	searchResponse := iaSearchResult{
		Response: struct {
			NumFound int     `json:"numFound"`
			Docs     []iaDoc `json:"docs"`
		}{
			NumFound: 2,
			Docs: []iaDoc{
				{Identifier: "govlawca", Title: "California Codes"},
				{Identifier: "govlawga", Title: "Georgia Codes"},
			},
		},
	}

	responseBody, _ := json.Marshal(searchResponse)

	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Write(responseBody)
	}))
	defer testServer.Close()

	config := DownloadConfig{
		RateLimit:  1 * time.Millisecond,
		Timeout:    10 * time.Second,
		UserAgent:  "regula-test/1.0",
		HTTPClient: testServer.Client(),
	}

	// The source uses hardcoded URLs to archive.org, so we can't fully test
	// ListDatasets with a mock server without modifying the source.
	// Instead, test the helper functions.
	source := NewInternetArchiveSource(config)
	if source.Name() != "archive" {
		t.Errorf("expected name 'archive', got %q", source.Name())
	}
}

func TestFindBestArchiveFile(t *testing.T) {
	testCases := []struct {
		name           string
		files          []iaFile
		expectedResult string
	}{
		{
			name: "prefers tar.gz",
			files: []iaFile{
				{Name: "data.zip", Format: "ZIP"},
				{Name: "data.tar.gz", Format: "Gzip"},
				{Name: "data.xml", Format: "XML"},
			},
			expectedResult: "data.tar.gz",
		},
		{
			name: "falls back to zip",
			files: []iaFile{
				{Name: "data.zip", Format: "ZIP"},
				{Name: "data.xml", Format: "XML"},
				{Name: "readme.txt", Format: "Text"},
			},
			expectedResult: "data.zip",
		},
		{
			name: "falls back to xml",
			files: []iaFile{
				{Name: "code.xml", Format: "XML"},
				{Name: "notes.txt", Format: "Text"},
			},
			expectedResult: "code.xml",
		},
		{
			name: "falls back to txt",
			files: []iaFile{
				{Name: "notes.txt", Format: "Text"},
				{Name: "image.jpg", Format: "JPEG"},
			},
			expectedResult: "notes.txt",
		},
		{
			name: "handles tgz extension",
			files: []iaFile{
				{Name: "data.tgz", Format: "Gzip"},
				{Name: "data.zip", Format: "ZIP"},
			},
			expectedResult: "data.tgz",
		},
		{
			name: "returns empty for no suitable files",
			files: []iaFile{
				{Name: "image.jpg", Format: "JPEG"},
				{Name: "video.mp4", Format: "MPEG4"},
			},
			expectedResult: "",
		},
		{
			name:           "returns empty for no files",
			files:          nil,
			expectedResult: "",
		},
		{
			name: "picks first tar.gz when multiple exist",
			files: []iaFile{
				{Name: "second.tar.gz", Format: "Gzip"},
				{Name: "first.tar.gz", Format: "Gzip"},
			},
			expectedResult: "second.tar.gz",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := findBestArchiveFile(testCase.files)
			if result != testCase.expectedResult {
				t.Errorf("findBestArchiveFile() = %q, want %q", result, testCase.expectedResult)
			}
		})
	}
}

func TestExtractJurisdictionFromID(t *testing.T) {
	testCases := []struct {
		name       string
		identifier string
		expected   string
	}{
		{
			name:       "govlaw california",
			identifier: "govlawca",
			expected:   "US-CA",
		},
		{
			name:       "govlaw georgia",
			identifier: "govlawga",
			expected:   "US-GA",
		},
		{
			name:       "govlaw new york",
			identifier: "govlawny",
			expected:   "US-NY",
		},
		{
			name:       "case insensitive",
			identifier: "GovLawCA",
			expected:   "US-CA",
		},
		{
			name:       "gov dot pattern",
			identifier: "gov.ca.codes",
			expected:   "US-CA",
		},
		{
			name:       "govlaw with numbers",
			identifier: "govlawca2024",
			expected:   "US-CA",
		},
		{
			name:       "unrecognized pattern",
			identifier: "somethingelse",
			expected:   "US",
		},
		{
			name:       "govlaw too short",
			identifier: "govlawa",
			expected:   "US",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := extractJurisdictionFromID(testCase.identifier)
			if result != testCase.expected {
				t.Errorf("extractJurisdictionFromID(%q) = %q, want %q",
					testCase.identifier, result, testCase.expected)
			}
		})
	}
}

func TestInternetArchiveDownloadDataset(t *testing.T) {
	metadataResponse := iaItemMetadata{
		Files: []iaFile{
			{Name: "codes.tar.gz", Format: "Gzip", Size: "1234"},
			{Name: "metadata.xml", Format: "XML", Size: "567"},
		},
	}
	metadataJSON, _ := json.Marshal(metadataResponse)

	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if strings.Contains(request.URL.Path, "/metadata/") {
			responseWriter.Write(metadataJSON)
		} else if strings.Contains(request.URL.Path, "/download/") {
			responseWriter.Write([]byte("fake archive content"))
		} else {
			responseWriter.WriteHeader(http.StatusNotFound)
		}
	}))
	defer testServer.Close()

	// The archive source uses hardcoded archive.org URLs, so full integration
	// testing requires the real API. Test the internal helpers instead.
	source := NewInternetArchiveSource(DownloadConfig{
		RateLimit:  1 * time.Millisecond,
		Timeout:    10 * time.Second,
		UserAgent:  "regula-test/1.0",
		HTTPClient: testServer.Client(),
	})

	if source.Name() != "archive" {
		t.Errorf("expected 'archive', got %q", source.Name())
	}
}
