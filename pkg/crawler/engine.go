package crawler

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/coolbeans/regula/pkg/library"
	"github.com/coolbeans/regula/pkg/store"
)

// Crawler is a BFS tree-walking legislation crawler that discovers and ingests
// US legislation by following cross-references between documents.
type Crawler struct {
	config     CrawlConfig
	lib        *library.Library
	resolver   *SourceResolver
	fetcher    *ContentFetcher
	provenance *ProvenanceTracker
}

// NewCrawler creates a Crawler with the given configuration. Opens or initializes
// the library at config.LibraryPath.
func NewCrawler(config CrawlConfig) (*Crawler, error) {
	// Open or initialize the library
	lib, err := library.Open(config.LibraryPath)
	if err != nil {
		lib, err = library.Init(config.LibraryPath, config.BaseURI)
		if err != nil {
			return nil, fmt.Errorf("failed to open or initialize library at %s: %w", config.LibraryPath, err)
		}
	}

	// Set up domain configs if not provided
	if config.DomainConfigs == nil {
		config.DomainConfigs = DefaultDomainConfigs()
	}

	return &Crawler{
		config:     config,
		lib:        lib,
		resolver:   NewSourceResolver(),
		fetcher:    NewContentFetcher(config),
		provenance: NewProvenanceTracker(config.BaseURI),
	}, nil
}

// NewCrawlerWithLibrary creates a Crawler using an existing library instance.
// Useful for testing and when the library is already open.
func NewCrawlerWithLibrary(config CrawlConfig, lib *library.Library) *Crawler {
	if config.DomainConfigs == nil {
		config.DomainConfigs = DefaultDomainConfigs()
	}

	return &Crawler{
		config:     config,
		lib:        lib,
		resolver:   NewSourceResolver(),
		fetcher:    NewContentFetcher(config),
		provenance: NewProvenanceTracker(config.BaseURI),
	}
}

// SetFetcher replaces the content fetcher (useful for testing with mock servers).
func (crawler *Crawler) SetFetcher(fetcher *ContentFetcher) {
	crawler.fetcher = fetcher
}

// Crawl performs a BFS crawl starting from the given seeds. Each seed can be
// a document ID (from the library), a citation string, or a direct URL.
func (crawler *Crawler) Crawl(seeds []CrawlSeed) (*CrawlReport, error) {
	crawlState := NewCrawlState(seeds, crawler.config)
	report := NewCrawlReport(crawler.config.DryRun, seeds)

	// Enqueue initial seeds
	for _, seed := range seeds {
		seedItems, err := crawler.seedToItems(seed)
		if err != nil {
			report.RecordItem(&CrawlItem{
				Citation:   seed.Value,
				DocumentID: seed.Value,
				Depth:      0,
				Status:     CrawlItemFailed,
				Error:      err.Error(),
			})
			continue
		}
		for _, seedItem := range seedItems {
			crawlState.Enqueue(seedItem)
		}
	}

	// BFS loop
	for crawlState.FrontierSize() > 0 && crawlState.WithinLimits() {
		currentItem := crawlState.Dequeue()
		if currentItem == nil {
			break
		}

		// Check depth limit
		if currentItem.Depth > crawler.config.MaxDepth {
			currentItem.Status = CrawlItemSkipped
			currentItem.Error = "exceeded max depth"
			crawlState.RecordProcessed(currentItem)
			report.RecordItem(currentItem)
			continue
		}

		// Skip if already visited
		if crawlState.IsVisited(currentItem.DocumentID) {
			currentItem.Status = CrawlItemSkipped
			currentItem.Error = "already visited"
			crawlState.RecordProcessed(currentItem)
			report.RecordItem(currentItem)
			continue
		}

		// Skip if already in library
		if existingDoc := crawler.lib.GetDocument(currentItem.DocumentID); existingDoc != nil {
			crawlState.MarkVisited(currentItem.DocumentID)
			currentItem.Status = CrawlItemSkipped
			currentItem.Error = "already in library"
			crawlState.RecordProcessed(currentItem)
			report.RecordItem(currentItem)

			// Still extract references from existing documents for BFS expansion
			crawler.extractAndEnqueueRefs(currentItem, crawlState)
			continue
		}

		crawlState.MarkVisited(currentItem.DocumentID)

		// Dry run: just record what would be fetched
		if crawler.config.DryRun {
			currentItem.Status = CrawlItemPending
			crawlState.RecordProcessed(currentItem)
			report.RecordItem(currentItem)
			continue
		}

		// Resolve URL if not set
		if currentItem.URL == "" {
			resolved, err := crawler.resolver.Resolve(currentItem.Citation)
			if err != nil {
				resolved, err = crawler.resolver.ResolveURN(currentItem.Citation)
			}
			if err != nil {
				currentItem.Status = CrawlItemFailed
				currentItem.Error = fmt.Sprintf("failed to resolve: %v", err)
				crawlState.RecordProcessed(currentItem)
				report.RecordItem(currentItem)
				crawler.provenance.RecordFailure(currentItem.Citation, "", currentItem.Error)
				continue
			}
			currentItem.URL = resolved.URL
			currentItem.Domain = resolved.Domain
			if currentItem.DocumentID == "" {
				currentItem.DocumentID = resolved.DocumentID
			}
		}

		// Fetch content
		currentItem.Status = CrawlItemFetching
		fetchedContent, err := crawler.fetcher.Fetch(currentItem.URL)
		if err != nil {
			currentItem.Status = CrawlItemFailed
			currentItem.Error = fmt.Sprintf("fetch failed: %v", err)
			crawlState.RecordProcessed(currentItem)
			report.RecordItem(currentItem)
			crawler.provenance.RecordFailure(currentItem.Citation, currentItem.URL, currentItem.Error)
			continue
		}

		// Ingest into library
		if len(fetchedContent.PlainText) == 0 {
			currentItem.Status = CrawlItemFailed
			currentItem.Error = "empty content after extraction"
			crawlState.RecordProcessed(currentItem)
			report.RecordItem(currentItem)
			crawler.provenance.RecordFailure(currentItem.Citation, currentItem.URL, currentItem.Error)
			continue
		}

		addOptions := library.AddOptions{
			Name:      currentItem.DocumentID,
			ShortName: currentItem.DocumentID,
			FullName:  currentItem.Citation,
			Force:     false,
		}

		_, err = crawler.lib.AddDocument(currentItem.DocumentID, fetchedContent.PlainText, addOptions)
		if err != nil {
			currentItem.Status = CrawlItemFailed
			currentItem.Error = fmt.Sprintf("ingestion failed: %v", err)
			crawlState.RecordProcessed(currentItem)
			report.RecordItem(currentItem)
			crawler.provenance.RecordFailure(currentItem.Citation, currentItem.URL, currentItem.Error)
			continue
		}

		currentItem.Status = CrawlItemIngested
		currentItem.FetchedAt = fetchedContent.FetchedAt
		crawlState.RecordProcessed(currentItem)
		report.RecordItem(currentItem)

		// Record provenance
		crawler.provenance.RecordFetch(currentItem.DocumentID, currentItem.URL, fetchedContent.FetchedAt)
		if currentItem.DiscoveredBy != "" {
			crawler.provenance.RecordDiscovery(currentItem.DiscoveredBy, currentItem.DocumentID, currentItem.Citation, currentItem.Depth)
		}

		// Extract cross-references and enqueue new items
		crawler.extractAndEnqueueRefs(currentItem, crawlState)

		// Save state periodically
		if crawler.config.StatePath != "" {
			_ = crawlState.SaveState(crawler.config.StatePath)
		}
	}

	// Final state save
	crawlState.Status = CrawlStatusCompleted
	if crawler.config.StatePath != "" {
		_ = crawlState.SaveState(crawler.config.StatePath)
	}

	return report, nil
}

// CrawlFromDocument starts a crawl seeded from an existing library document's
// external references.
func (crawler *Crawler) CrawlFromDocument(documentID string) (*CrawlReport, error) {
	seeds := []CrawlSeed{
		{Type: SeedTypeDocumentID, Value: documentID},
	}
	return crawler.Crawl(seeds)
}

// CrawlFromCitation starts a crawl from a raw citation string.
func (crawler *Crawler) CrawlFromCitation(citation string) (*CrawlReport, error) {
	seeds := []CrawlSeed{
		{Type: SeedTypeCitation, Value: citation},
	}
	return crawler.Crawl(seeds)
}

// Plan performs a dry-run crawl: resolves citations and plans the crawl tree
// without making any network requests or modifying the library.
func (crawler *Crawler) Plan(seeds []CrawlSeed) (*CrawlReport, error) {
	originalDryRun := crawler.config.DryRun
	crawler.config.DryRun = true
	defer func() { crawler.config.DryRun = originalDryRun }()
	return crawler.Crawl(seeds)
}

// Resume continues a previously interrupted crawl from saved state.
func (crawler *Crawler) Resume(statePath string) (*CrawlReport, error) {
	crawlState, err := LoadState(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load crawl state: %w", err)
	}

	crawlState.Status = CrawlStatusRunning
	report := NewCrawlReport(crawlState.Config.DryRun, crawlState.Seeds)

	// Replay processed items into report
	for _, processedItem := range crawlState.ProcessedItems {
		report.RecordItem(processedItem)
	}

	// Continue BFS from frontier
	for crawlState.FrontierSize() > 0 && crawlState.WithinLimits() {
		currentItem := crawlState.Dequeue()
		if currentItem == nil {
			break
		}

		if currentItem.Depth > crawlState.Config.MaxDepth {
			currentItem.Status = CrawlItemSkipped
			currentItem.Error = "exceeded max depth"
			crawlState.RecordProcessed(currentItem)
			report.RecordItem(currentItem)
			continue
		}

		if crawlState.IsVisited(currentItem.DocumentID) {
			continue
		}

		crawlState.MarkVisited(currentItem.DocumentID)

		// Resolve URL
		if currentItem.URL == "" {
			resolved, err := crawler.resolver.Resolve(currentItem.Citation)
			if err != nil {
				resolved, err = crawler.resolver.ResolveURN(currentItem.Citation)
			}
			if err != nil {
				currentItem.Status = CrawlItemFailed
				currentItem.Error = fmt.Sprintf("failed to resolve: %v", err)
				crawlState.RecordProcessed(currentItem)
				report.RecordItem(currentItem)
				continue
			}
			currentItem.URL = resolved.URL
			currentItem.Domain = resolved.Domain
		}

		// Fetch and ingest
		fetchedContent, err := crawler.fetcher.Fetch(currentItem.URL)
		if err != nil {
			currentItem.Status = CrawlItemFailed
			currentItem.Error = fmt.Sprintf("fetch failed: %v", err)
			crawlState.RecordProcessed(currentItem)
			report.RecordItem(currentItem)
			continue
		}

		if len(fetchedContent.PlainText) == 0 {
			currentItem.Status = CrawlItemFailed
			currentItem.Error = "empty content"
			crawlState.RecordProcessed(currentItem)
			report.RecordItem(currentItem)
			continue
		}

		addOptions := library.AddOptions{
			Name:      currentItem.DocumentID,
			ShortName: currentItem.DocumentID,
			FullName:  currentItem.Citation,
			Force:     false,
		}

		_, err = crawler.lib.AddDocument(currentItem.DocumentID, fetchedContent.PlainText, addOptions)
		if err != nil {
			currentItem.Status = CrawlItemFailed
			currentItem.Error = fmt.Sprintf("ingestion failed: %v", err)
			crawlState.RecordProcessed(currentItem)
			report.RecordItem(currentItem)
			continue
		}

		currentItem.Status = CrawlItemIngested
		currentItem.FetchedAt = fetchedContent.FetchedAt
		crawlState.RecordProcessed(currentItem)
		report.RecordItem(currentItem)

		crawler.extractAndEnqueueRefs(currentItem, crawlState)

		if statePath != "" {
			_ = crawlState.SaveState(statePath)
		}
	}

	crawlState.Status = CrawlStatusCompleted
	if statePath != "" {
		_ = crawlState.SaveState(statePath)
	}

	return report, nil
}

// Provenance returns the provenance tracker for inspecting discovery chains.
func (crawler *Crawler) Provenance() *ProvenanceTracker {
	return crawler.provenance
}

// seedToItems converts a CrawlSeed to one or more CrawlItems for the frontier.
func (crawler *Crawler) seedToItems(seed CrawlSeed) ([]*CrawlItem, error) {
	switch seed.Type {
	case SeedTypeDocumentID:
		return crawler.seedFromDocumentID(seed.Value)
	case SeedTypeCitation:
		return crawler.seedFromCitation(seed.Value)
	case SeedTypeURL:
		return crawler.seedFromURL(seed.Value)
	default:
		return nil, fmt.Errorf("unknown seed type: %s", seed.Type)
	}
}

// seedFromDocumentID extracts external references from an existing library document
// and returns them as crawl items.
func (crawler *Crawler) seedFromDocumentID(documentID string) ([]*CrawlItem, error) {
	existingDoc := crawler.lib.GetDocument(documentID)
	if existingDoc == nil {
		return nil, fmt.Errorf("document %q not found in library", documentID)
	}

	tripleStore, err := crawler.lib.LoadTripleStore(documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to load triple store for %s: %w", documentID, err)
	}

	return crawler.extractExternalRefs(tripleStore, documentID, 1), nil
}

// seedFromCitation resolves a citation and returns it as a crawl item.
func (crawler *Crawler) seedFromCitation(citation string) ([]*CrawlItem, error) {
	resolved, err := crawler.resolver.Resolve(citation)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve citation %q: %w", citation, err)
	}

	return []*CrawlItem{{
		Citation:   citation,
		URL:        resolved.URL,
		DocumentID: resolved.DocumentID,
		Depth:      0,
		Status:     CrawlItemPending,
		Domain:     resolved.Domain,
	}}, nil
}

// seedFromURL creates a crawl item from a direct URL.
func (crawler *Crawler) seedFromURL(targetURL string) ([]*CrawlItem, error) {
	domain := ExtractDomainFromURL(targetURL)
	documentID := deriveDocIDFromURL(targetURL)

	return []*CrawlItem{{
		Citation:   targetURL,
		URL:        targetURL,
		DocumentID: documentID,
		Depth:      0,
		Status:     CrawlItemPending,
		Domain:     domain,
	}}, nil
}

// extractAndEnqueueRefs loads a document's triple store and extracts external
// references, enqueueing them as new crawl items.
func (crawler *Crawler) extractAndEnqueueRefs(parentItem *CrawlItem, crawlState *CrawlState) {
	tripleStore, err := crawler.lib.LoadTripleStore(parentItem.DocumentID)
	if err != nil {
		return
	}

	newItems := crawler.extractExternalRefs(tripleStore, parentItem.DocumentID, parentItem.Depth+1)
	for _, newItem := range newItems {
		crawlState.Enqueue(newItem)
	}
}

// extractExternalRefs queries a triple store for external references and converts
// them to crawl items.
func (crawler *Crawler) extractExternalRefs(tripleStore *store.TripleStore, sourceDocumentID string, depth int) []*CrawlItem {
	externalRefTriples := tripleStore.Find("", store.PropExternalRef, "")

	seen := make(map[string]bool)
	var crawlItems []*CrawlItem

	for _, triple := range externalRefTriples {
		urn := triple.Object
		if seen[urn] {
			continue
		}
		seen[urn] = true

		// Try to resolve the URN
		resolved, err := crawler.resolver.ResolveURN(urn)
		if err != nil {
			// Try as a citation
			resolved, err = crawler.resolver.Resolve(urn)
			if err != nil {
				continue
			}
		}

		crawlItems = append(crawlItems, &CrawlItem{
			Citation:     urn,
			URL:          resolved.URL,
			DocumentID:   resolved.DocumentID,
			Depth:        depth,
			DiscoveredBy: sourceDocumentID,
			Status:       CrawlItemPending,
			Domain:       resolved.Domain,
		})
	}

	return crawlItems
}

// deriveDocIDFromURL creates a document ID from a URL by extracting domain and path.
func deriveDocIDFromURL(targetURL string) string {
	domain := ExtractDomainFromURL(targetURL)
	domain = strings.ReplaceAll(domain, "www.", "")
	domain = strings.ReplaceAll(domain, ".", "-")

	// Use last path segment as identifier
	pathSegment := filepath.Base(targetURL)
	if pathSegment == "." || pathSegment == "/" {
		pathSegment = "index"
	}

	documentID := fmt.Sprintf("crawled-%s-%s", domain, pathSegment)
	return strings.ToLower(documentID)
}

// defaultStatePath returns the default path for crawl state files.
func defaultStatePath(libraryPath string) string {
	return filepath.Join(libraryPath, "crawl-state.json")
}
