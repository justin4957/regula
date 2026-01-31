package crawler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestExtractTextFromHTMLBasic(t *testing.T) {
	html := []byte(`<html><head><title>Test</title></head><body>
<h1>Section 1</h1>
<p>This is paragraph one.</p>
<p>This is paragraph two.</p>
</body></html>`)

	result := string(ExtractTextFromHTML(html))

	if !strings.Contains(result, "Section 1") {
		t.Error("expected heading text 'Section 1' in output")
	}
	if !strings.Contains(result, "This is paragraph one.") {
		t.Error("expected paragraph text in output")
	}
}

func TestExtractTextFromHTMLRemovesScripts(t *testing.T) {
	html := []byte(`<body>
<script>var x = 1; alert("bad");</script>
<p>Visible content</p>
<style>.hidden { display: none; }</style>
</body>`)

	result := string(ExtractTextFromHTML(html))

	if strings.Contains(result, "alert") {
		t.Error("script content should be removed")
	}
	if strings.Contains(result, "display: none") {
		t.Error("style content should be removed")
	}
	if !strings.Contains(result, "Visible content") {
		t.Error("visible content should be preserved")
	}
}

func TestExtractTextFromHTMLRemovesNavHeaderFooter(t *testing.T) {
	html := []byte(`<body>
<header><a href="/">Home</a></header>
<nav><ul><li>Menu item</li></ul></nav>
<main><p>Main content here</p></main>
<footer>Copyright 2024</footer>
</body>`)

	result := string(ExtractTextFromHTML(html))

	if strings.Contains(result, "Menu item") {
		t.Error("nav content should be removed")
	}
	if !strings.Contains(result, "Main content here") {
		t.Error("main content should be preserved")
	}
}

func TestExtractTextFromHTMLDecodesEntities(t *testing.T) {
	html := []byte(`<body><p>Tom &amp; Jerry &lt;3 &gt; &quot;love&quot;</p></body>`)

	result := string(ExtractTextFromHTML(html))

	if !strings.Contains(result, "Tom & Jerry") {
		t.Errorf("expected decoded &amp; in: %s", result)
	}
	if !strings.Contains(result, `"love"`) {
		t.Errorf("expected decoded &quot; in: %s", result)
	}
}

func TestExtractTextFromHTMLListItems(t *testing.T) {
	html := []byte(`<body>
<ul>
<li>First item</li>
<li>Second item</li>
<li>Third item</li>
</ul>
</body>`)

	result := string(ExtractTextFromHTML(html))

	if !strings.Contains(result, "- First item") {
		t.Errorf("expected list marker in: %s", result)
	}
}

func TestExtractTextFromHTMLPlainText(t *testing.T) {
	// When there are no HTML tags, the input is returned cleaned
	text := []byte("This is just plain text with no HTML.")
	result := string(ExtractTextFromHTML(text))

	if !strings.Contains(result, "plain text") {
		t.Errorf("expected plain text preserved: %s", result)
	}
}

func TestFetchWithMockServer(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Content-Type", "text/html")
		responseWriter.WriteHeader(http.StatusOK)
		responseWriter.Write([]byte(`<html><body><h1>Test Law</h1><p>Section 1. This is a test provision.</p></body></html>`))
	}))
	defer testServer.Close()

	config := CrawlConfig{
		RateLimit: 10 * time.Millisecond,
		Timeout:   5 * time.Second,
		UserAgent: "test-crawler/1.0",
	}
	fetcher := NewContentFetcher(config)

	fetchedContent, err := fetcher.Fetch(testServer.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fetchedContent.StatusCode != 200 {
		t.Errorf("status code = %d, want 200", fetchedContent.StatusCode)
	}

	plainText := string(fetchedContent.PlainText)
	if !strings.Contains(plainText, "Test Law") {
		t.Errorf("extracted text missing 'Test Law': %s", plainText)
	}
	if !strings.Contains(plainText, "test provision") {
		t.Errorf("extracted text missing 'test provision': %s", plainText)
	}
}

func TestFetchHTTPError(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.WriteHeader(http.StatusNotFound)
	}))
	defer testServer.Close()

	config := CrawlConfig{
		RateLimit: 10 * time.Millisecond,
		Timeout:   5 * time.Second,
	}
	fetcher := NewContentFetcher(config)

	_, err := fetcher.Fetch(testServer.URL)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error = %q, want '404' in message", err.Error())
	}
}

func TestFetchEmptyURL(t *testing.T) {
	config := CrawlConfig{
		RateLimit: 10 * time.Millisecond,
		Timeout:   5 * time.Second,
	}
	fetcher := NewContentFetcher(config)

	_, err := fetcher.Fetch("")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestFetchPlainText(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Content-Type", "text/plain")
		responseWriter.WriteHeader(http.StatusOK)
		responseWriter.Write([]byte("Section 1798.100. Consumer rights."))
	}))
	defer testServer.Close()

	config := CrawlConfig{
		RateLimit: 10 * time.Millisecond,
		Timeout:   5 * time.Second,
	}
	fetcher := NewContentFetcher(config)

	fetchedContent, err := fetcher.Fetch(testServer.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	plainText := string(fetchedContent.PlainText)
	if !strings.Contains(plainText, "Consumer rights") {
		t.Errorf("plain text content missing: %s", plainText)
	}
}
