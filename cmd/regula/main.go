package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/coolbeans/regula/pkg/analysis"
	"github.com/coolbeans/regula/pkg/bulk"
	"github.com/coolbeans/regula/pkg/crawler"
	"github.com/coolbeans/regula/pkg/draft"
	"github.com/coolbeans/regula/pkg/eurlex"
	"github.com/coolbeans/regula/pkg/extract"
	"github.com/coolbeans/regula/pkg/fetch"
	"github.com/coolbeans/regula/pkg/library"
	"github.com/coolbeans/regula/pkg/pattern"
	"github.com/coolbeans/regula/pkg/linkcheck"
	"github.com/coolbeans/regula/pkg/playground"
	"github.com/coolbeans/regula/pkg/query"
	"github.com/coolbeans/regula/pkg/simulate"
	"github.com/coolbeans/regula/pkg/store"
	"github.com/coolbeans/regula/pkg/validate"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

// Global state for the loaded graph
var (
	tripleStore      *store.TripleStore
	executor         *query.Executor
	graphLoaded      bool
	graphPath        string
	loadedDocType    extract.DocumentType
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "regula",
		Short: "Automated Regulation Mapper",
		Long: `Regula transforms dense legal regulations into auditable,
queryable, and simulatable programs.

It ingests regulatory documents and produces:
  - Queryable knowledge graphs via SPARQL
  - Type-safe domain models with compile-time verification
  - Impact analysis for regulatory changes
  - Simulation engine for compliance scenarios
  - Audit trails with provenance tracking`,
		Version: version,
	}

	// Add subcommands
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(ingestCmd())
	rootCmd.AddCommand(queryCmd())
	rootCmd.AddCommand(validateCmd())
	rootCmd.AddCommand(impactCmd())
	rootCmd.AddCommand(matchCmd())
	rootCmd.AddCommand(simulateCmd())
	rootCmd.AddCommand(auditCmd())
	rootCmd.AddCommand(exportCmd())
	rootCmd.AddCommand(compareCmd())
	rootCmd.AddCommand(refsCmd())
	rootCmd.AddCommand(libraryCmd())
	rootCmd.AddCommand(crawlCmd())
	rootCmd.AddCommand(playgroundCmd())
	rootCmd.AddCommand(bulkCmd())
	rootCmd.AddCommand(draftCmd())
	rootCmd.AddCommand(searchCmd())
	rootCmd.AddCommand(navigateCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [project-name]",
		Short: "Initialize a new regulation project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := "regula-project"
			if len(args) > 0 {
				projectName = args[0]
			}

			// Create directories
			dirs := []string{
				filepath.Join(projectName, "regulations"),
				filepath.Join(projectName, "graphs"),
				filepath.Join(projectName, "scenarios"),
				filepath.Join(projectName, "reports"),
			}

			for _, dir := range dirs {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return fmt.Errorf("failed to create directory %s: %w", dir, err)
				}
			}

			fmt.Printf("Initialized regulation project: %s\n", projectName)
			fmt.Println("Created directories:")
			for _, dir := range dirs {
				fmt.Printf("  - %s/\n", dir)
			}
			fmt.Printf("\nNext steps:\n")
			fmt.Printf("  1. Add regulation documents to %s/regulations/\n", projectName)
			fmt.Printf("  2. Run: regula ingest --source %s/regulations/your-doc.txt\n", projectName)
			fmt.Printf("  3. Run: regula query \"SELECT ?article WHERE { ?article rdf:type reg:Article }\"\n")
			return nil
		},
	}
}

func ingestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Ingest a regulation document",
		Long: `Ingest a regulation document and build a queryable knowledge graph.

Supported formats: TXT, MD (Markdown-formatted regulations)

Example:
  regula ingest --source gdpr.txt
  regula ingest --source gdpr.txt --output gdpr-graph.json --stats`,
		RunE: func(cmd *cobra.Command, args []string) error {
			source, _ := cmd.Flags().GetString("source")
			output, _ := cmd.Flags().GetString("output")
			showStats, _ := cmd.Flags().GetBool("stats")
			baseURI, _ := cmd.Flags().GetString("base-uri")
			enableGates, _ := cmd.Flags().GetBool("gates")
			skipGates, _ := cmd.Flags().GetStringSlice("skip-gates")
			strictMode, _ := cmd.Flags().GetBool("strict")
			failOnWarn, _ := cmd.Flags().GetBool("fail-on-warn")
			fetchRefs, _ := cmd.Flags().GetBool("fetch-refs")
			maxDepth, _ := cmd.Flags().GetInt("max-depth")
			maxDocuments, _ := cmd.Flags().GetInt("max-documents")
			allowedDomains, _ := cmd.Flags().GetStringSlice("allowed-domains")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			cacheDir, _ := cmd.Flags().GetString("cache-dir")

			if source == "" {
				return fmt.Errorf("--source flag is required")
			}

			// Check if file exists
			fileInfo, err := os.Stat(source)
			if os.IsNotExist(err) {
				return fmt.Errorf("source file not found: %s", source)
			}
			if err != nil {
				return fmt.Errorf("failed to stat source: %w", err)
			}

			fmt.Printf("Ingesting regulation from: %s\n", source)
			startTime := time.Now()

			// Set up validation gates if enabled.
			var gatePipeline *validate.GatePipeline
			var gateContext *validate.ValidationContext
			if enableGates {
				gateConfig := &validate.ValidationConfig{
					Thresholds: make(map[string]float64),
					SkipGates:  skipGates,
					StrictMode: strictMode,
					FailOnWarn: failOnWarn,
				}
				gatePipeline = validate.NewGatePipeline(gateConfig)
				gatePipeline.RegisterDefaultGates()
				gateContext = &validate.ValidationContext{
					SourcePath: source,
					SourceSize: fileInfo.Size(),
					Config:     gateConfig,
				}
			}

			// Gate V0: Schema validation (after file load, before parsing).
			if gatePipeline != nil {
				v0Result := gatePipeline.RunGate("V0", gateContext)
				if v0Result != nil && !v0Result.Skipped {
					printGateResult(v0Result)
					if !v0Result.Passed && strictMode {
						return fmt.Errorf("pipeline halted: gate V0 (schema) failed")
					}
				}
			}

			// Step 1: Parse document
			fmt.Print("  1. Parsing document structure... ")
			file, err := os.Open(source)
			if err != nil {
				return fmt.Errorf("failed to open source: %w", err)
			}
			defer file.Close()

			parseStart := time.Now()
			parser := newParserWithPatterns()
			doc, err := parser.Parse(file)
			if err != nil {
				return fmt.Errorf("failed to parse document: %w", err)
			}
			parseDuration := time.Since(parseStart)
			fmt.Printf("done (%d chapters, %d articles)\n", len(doc.Chapters), countArticles(doc))

			// Gate V1: Structure validation (after parsing).
			if gatePipeline != nil {
				gateContext.Document = doc
				gateContext.ParseDuration = parseDuration
				v1Result := gatePipeline.RunGate("V1", gateContext)
				if v1Result != nil && !v1Result.Skipped {
					printGateResult(v1Result)
					if !v1Result.Passed && strictMode {
						return fmt.Errorf("pipeline halted: gate V1 (structure) failed")
					}
				}
			}

			// Step 2: Extract definitions
			fmt.Print("  2. Extracting defined terms... ")
			defExtractor := extract.NewDefinitionExtractor()
			definitions := defExtractor.ExtractDefinitions(doc)
			fmt.Printf("done (%d definitions)\n", len(definitions))

			// Step 3: Extract cross-references
			fmt.Print("  3. Identifying cross-references... ")
			refExtractor := extract.NewReferenceExtractor()
			references := refExtractor.ExtractFromDocument(doc)
			fmt.Printf("done (%d references)\n", len(references))

			// Step 4: Extract rights and obligations
			fmt.Print("  4. Extracting rights/obligations... ")
			semExtractor := extract.NewSemanticExtractor()
			semantics := semExtractor.ExtractFromDocument(doc)
			semStats := extract.CalculateSemanticStats(semantics)
			fmt.Printf("done (%d rights, %d obligations)\n", semStats.Rights, semStats.Obligations)

			// Gate V2: Coverage validation (after extraction).
			if gatePipeline != nil {
				gateContext.Definitions = definitions
				gateContext.References = references
				gateContext.Semantics = semantics
				v2Result := gatePipeline.RunGate("V2", gateContext)
				if v2Result != nil && !v2Result.Skipped {
					printGateResult(v2Result)
					if !v2Result.Passed && strictMode {
						return fmt.Errorf("pipeline halted: gate V2 (coverage) failed")
					}
				}
			}

			// Step 5: Resolve references
			fmt.Print("  5. Resolving cross-references... ")
			resolver := extract.NewReferenceResolver(baseURI, extractDocID(source))
			resolver.IndexDocument(doc)
			resolved := resolver.ResolveAll(references)
			report := extract.GenerateReport(resolved)
			fmt.Printf("done (%.0f%% resolved)\n", report.ResolutionRate*100)

			// Step 6: Build complete knowledge graph
			fmt.Print("  6. Building knowledge graph... ")
			tripleStore = store.NewTripleStore()
			builder := store.NewGraphBuilder(tripleStore, baseURI)
			stats, err := builder.BuildComplete(doc, defExtractor, refExtractor, resolver, semExtractor)
			if err != nil {
				return fmt.Errorf("failed to build graph: %w", err)
			}
			fmt.Printf("done (%d triples)\n", stats.TotalTriples)

			// Gate V3: Quality validation (after resolution + graph).
			if gatePipeline != nil {
				gateContext.ResolvedReferences = resolved
				gateContext.TripleStore = tripleStore
				v3Result := gatePipeline.RunGate("V3", gateContext)
				if v3Result != nil && !v3Result.Skipped {
					printGateResult(v3Result)
				}
			}

			// Step 7: Fetch external references (optional)
			if fetchRefs {
				fmt.Print("  7. Fetching external references... ")

				fetchConfig := fetch.FetchConfig{
					MaxDepth:       maxDepth,
					MaxDocuments:   maxDocuments,
					AllowedDomains: allowedDomains,
					RateLimit:      fetch.DefaultFetchRateLimit,
					Timeout:        fetch.DefaultFetchTimeout,
					CacheDir:       cacheDir,
					DryRun:         dryRun,
				}

				eurlexValidator := eurlex.NewEURLexClient(eurlex.DefaultConfig())
				recursiveFetcher, fetcherErr := fetch.NewRecursiveFetcher(fetchConfig, eurlexValidator)
				if fetcherErr != nil {
					return fmt.Errorf("failed to initialize recursive fetcher: %w", fetcherErr)
				}

				sourceDocURI := baseURI + "GDPR"
				var fetchReport *fetch.FetchReport

				if dryRun {
					fetchReport, fetcherErr = recursiveFetcher.Plan(tripleStore, sourceDocURI)
				} else {
					fetchReport, fetcherErr = recursiveFetcher.Fetch(tripleStore, sourceDocURI)
				}

				if fetcherErr != nil {
					fmt.Printf("warning: %v\n", fetcherErr)
				} else {
					if dryRun {
						fmt.Println("done (dry-run)")
					} else {
						fmt.Printf("done (%d fetched, %d cached, %d failed, %d triples added)\n",
							fetchReport.FetchedCount, fetchReport.CachedCount,
							fetchReport.FailedCount, fetchReport.TriplesAdded)
					}
					fmt.Println()
					fmt.Print(fetchReport.String())
				}
			}

			// Initialize executor
			executor = query.NewExecutor(tripleStore)
			graphLoaded = true
			graphPath = source

			elapsed := time.Since(startTime)
			fmt.Printf("\nIngestion complete in %v\n", elapsed)

			// Show stats if requested
			if showStats {
				fmt.Println("\nGraph Statistics:")
				fmt.Printf("  Total triples:    %d\n", stats.TotalTriples)
				fmt.Printf("  Articles:         %d\n", stats.Articles)
				fmt.Printf("  Chapters:         %d\n", stats.Chapters)
				fmt.Printf("  Sections:         %d\n", stats.Sections)
				fmt.Printf("  Recitals:         %d\n", stats.Recitals)
				fmt.Printf("  Definitions:      %d\n", stats.Definitions)
				fmt.Printf("  References:       %d\n", stats.References)
				fmt.Printf("  Rights:           %d\n", stats.Rights)
				fmt.Printf("  Obligations:      %d\n", stats.Obligations)
				fmt.Printf("  Term usages:      %d\n", stats.TermUsages)
			}

			// Save graph if output specified
			if output != "" {
				fmt.Printf("\nSaving graph to: %s\n", output)
				if err := saveGraph(tripleStore, output); err != nil {
					return fmt.Errorf("failed to save graph: %w", err)
				}
				fmt.Println("Graph saved successfully.")
			}

			fmt.Println("\nReady for queries. Run: regula query \"SELECT ?article WHERE { ?article rdf:type reg:Article } LIMIT 5\"")
			return nil
		},
	}

	cmd.Flags().StringP("source", "s", "", "Source document path")
	cmd.Flags().StringP("output", "o", "", "Output graph file (JSON)")
	cmd.Flags().Bool("stats", false, "Show detailed statistics")
	cmd.Flags().String("base-uri", "https://regula.dev/regulations/", "Base URI for the graph")
	cmd.Flags().Bool("gates", false, "Enable validation gates during ingestion")
	cmd.Flags().StringSlice("skip-gates", []string{}, "Gates to skip (V0,V1,V2,V3)")
	cmd.Flags().Bool("strict", false, "Halt pipeline on gate failure")
	cmd.Flags().Bool("fail-on-warn", false, "Halt pipeline on gate warnings")

	// Recursive fetch flags
	cmd.Flags().Bool("fetch-refs", false, "Fetch external referenced documents to build a federated graph")
	cmd.Flags().Int("max-depth", fetch.DefaultMaxDepth, "Maximum recursion depth for fetching external references")
	cmd.Flags().Int("max-documents", fetch.DefaultMaxDocuments, "Maximum number of external documents to fetch")
	cmd.Flags().StringSlice("allowed-domains", []string{}, "Restrict fetching to these domains (empty allows all)")
	cmd.Flags().Bool("dry-run", false, "Plan what would be fetched without making network calls")
	cmd.Flags().String("cache-dir", "", "Directory for caching fetched document metadata")

	return cmd
}

func queryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query [sparql-query]",
		Short: "Query the regulation graph",
		Long: `Execute a SPARQL query against the regulation knowledge graph.

You must first ingest a regulation document using 'regula ingest'.

Supports SELECT, CONSTRUCT, and DESCRIBE queries.

Examples:
  # Basic SELECT query
  regula query "SELECT ?article ?title WHERE { ?article rdf:type reg:Article . ?article reg:title ?title } LIMIT 5"

  # DESCRIBE query with direct URI
  regula query "DESCRIBE GDPR:Art17"

  # DESCRIBE query with variable
  regula query "DESCRIBE ?article WHERE { ?article reg:title \"Right to erasure\" }"

  # CONSTRUCT query to extract subgraph
  regula query "CONSTRUCT { ?a reg:hasTitle ?t } WHERE { ?a rdf:type reg:Article . ?a reg:title ?t }"

  # CONSTRUCT with Turtle output
  regula query --format turtle "CONSTRUCT { ?a rdf:type reg:Article } WHERE { ?a rdf:type reg:Article }"

  # CONSTRUCT with N-Triples output
  regula query --format ntriples "CONSTRUCT { ?a rdf:type reg:Article } WHERE { ?a rdf:type reg:Article }"

  # Use a template
  regula query --template definitions

  # JSON output
  regula query --format json "SELECT ?term WHERE { ?term rdf:type reg:DefinedTerm }"

  # With timing
  regula query --timing "SELECT ?a WHERE { ?a rdf:type reg:Article }"

Available templates:
  articles     - List all articles with titles
  definitions  - List all defined terms
  chapters     - List all chapters
  references   - List cross-references
  rights       - Find provisions granting rights`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateName, _ := cmd.Flags().GetString("template")
			formatStr, _ := cmd.Flags().GetString("format")
			showTiming, _ := cmd.Flags().GetBool("timing")
			source, _ := cmd.Flags().GetString("source")
			listTemplates, _ := cmd.Flags().GetBool("list-templates")

			// List templates
			if listTemplates {
				printTemplates()
				return nil
			}

			// Get the query
			var queryStr string
			if templateName != "" {
				tmpl, ok := queryTemplates[templateName]
				if !ok {
					return fmt.Errorf("unknown template: %s\nUse --list-templates to see available templates", templateName)
				}
				queryStr = tmpl.Query
				if !showTiming {
					fmt.Printf("Template: %s\n", templateName)
					fmt.Printf("Description: %s\n\n", tmpl.Description)
				}
			} else if len(args) > 0 {
				queryStr = args[0]
			} else {
				return fmt.Errorf("provide a query or use --template\nUse --list-templates to see available templates")
			}

			// Load graph if source specified
			if source != "" {
				if err := loadAndIngest(source); err != nil {
					return err
				}
			}

			// Check if graph is loaded
			if !graphLoaded {
				return fmt.Errorf("no graph loaded. Run 'regula ingest --source <file>' first, or use --source flag")
			}

			// Parse query to determine type
			parsedQuery, err := query.ParseQuery(queryStr)
			if err != nil {
				return fmt.Errorf("query parse error: %w", err)
			}

			startTime := time.Now()

			// Handle CONSTRUCT queries
			if parsedQuery.Type == query.ConstructQueryType {
				return executeConstructQuery(cmd, parsedQuery, formatStr, showTiming, startTime)
			}

			// Handle DESCRIBE queries
			if parsedQuery.Type == query.DescribeQueryType {
				return executeDescribeQuery(cmd, parsedQuery, formatStr, showTiming, startTime)
			}

			// Execute SELECT query
			result, err := executor.Execute(parsedQuery)
			queryTime := time.Since(startTime)

			if err != nil {
				return fmt.Errorf("query error: %w", err)
			}

			// Format output
			format := query.OutputFormat(formatStr)
			output, err := result.Format(format)
			if err != nil {
				return fmt.Errorf("format error: %w", err)
			}

			fmt.Print(output)

			// Show timing if requested
			if showTiming {
				fmt.Printf("\nQuery executed in %v\n", queryTime)
				fmt.Printf("  Parse:   %v\n", result.Metrics.ParseTime)
				fmt.Printf("  Plan:    %v\n", result.Metrics.PlanTime)
				fmt.Printf("  Execute: %v\n", result.Metrics.ExecuteTime)
			}

			return nil
		},
	}

	cmd.Flags().StringP("template", "t", "", "Use a pre-built query template")
	cmd.Flags().StringP("format", "f", "table", "Output format (table, json, csv for SELECT; turtle, ntriples, json for CONSTRUCT/DESCRIBE)")
	cmd.Flags().Bool("timing", false, "Show query execution timing")
	cmd.Flags().StringP("source", "s", "", "Source document to ingest before querying")
	cmd.Flags().Bool("list-templates", false, "List available query templates")

	return cmd
}

// executeConstructQuery handles execution and output of CONSTRUCT queries.
func executeConstructQuery(cmd *cobra.Command, parsedQuery *query.Query, formatStr string, showTiming bool, startTime time.Time) error {
	result, err := executor.ExecuteConstruct(parsedQuery)
	queryTime := time.Since(startTime)

	if err != nil {
		return fmt.Errorf("CONSTRUCT query error: %w", err)
	}

	// Default format for CONSTRUCT is turtle
	if formatStr == "table" || formatStr == "csv" {
		formatStr = "turtle"
	}

	format := query.OutputFormat(formatStr)
	output, err := result.Format(format)
	if err != nil {
		return fmt.Errorf("format error: %w", err)
	}

	fmt.Print(output)

	// Show timing if requested
	if showTiming {
		fmt.Printf("\nCONSTRUCT query executed in %v\n", queryTime)
		fmt.Printf("  Parse:   %v\n", result.Metrics.ParseTime)
		fmt.Printf("  Execute: %v\n", result.Metrics.ExecuteTime)
		fmt.Printf("  Triples: %d\n", result.Count)
	}

	return nil
}

// executeDescribeQuery handles execution and output of DESCRIBE queries.
func executeDescribeQuery(cmd *cobra.Command, parsedQuery *query.Query, formatStr string, showTiming bool, startTime time.Time) error {
	result, err := executor.ExecuteDescribe(parsedQuery)
	queryTime := time.Since(startTime)

	if err != nil {
		return fmt.Errorf("DESCRIBE query error: %w", err)
	}

	// Default format for DESCRIBE is turtle
	if formatStr == "table" || formatStr == "csv" {
		formatStr = "turtle"
	}

	format := query.OutputFormat(formatStr)
	output, err := result.Format(format)
	if err != nil {
		return fmt.Errorf("format error: %w", err)
	}

	fmt.Print(output)

	// Show timing if requested
	if showTiming {
		fmt.Printf("\nDESCRIBE query executed in %v\n", queryTime)
		fmt.Printf("  Parse:   %v\n", result.Metrics.ParseTime)
		fmt.Printf("  Execute: %v\n", result.Metrics.ExecuteTime)
		fmt.Printf("  Triples: %d\n", result.Count)
	}

	return nil
}

// QueryTemplate represents a pre-built query template.
type QueryTemplate struct {
	Name        string
	Description string
	Query       string
}

var queryTemplates = map[string]QueryTemplate{
	"articles": {
		Name:        "articles",
		Description: "List all articles with titles",
		Query: `SELECT ?article ?title WHERE {
  ?article rdf:type reg:Article .
  ?article reg:title ?title .
} ORDER BY ?article`,
	},
	"definitions": {
		Name:        "definitions",
		Description: "List all defined terms with their definitions",
		Query: `SELECT ?term ?termText ?definition WHERE {
  ?term rdf:type reg:DefinedTerm .
  ?term reg:term ?termText .
  OPTIONAL { ?term reg:definition ?definition . }
} ORDER BY ?termText`,
	},
	"chapters": {
		Name:        "chapters",
		Description: "List all chapters with titles",
		Query: `SELECT ?chapter ?title WHERE {
  ?chapter rdf:type reg:Chapter .
  ?chapter reg:title ?title .
} ORDER BY ?chapter`,
	},
	"references": {
		Name:        "references",
		Description: "List all cross-references between articles",
		Query: `SELECT ?from ?to WHERE {
  ?from reg:references ?to .
} ORDER BY ?from LIMIT 50`,
	},
	"rights": {
		Name:        "rights",
		Description: "Find articles that grant rights",
		Query: `SELECT ?article ?title ?right ?rightType WHERE {
  ?article rdf:type reg:Article .
  ?article reg:title ?title .
  ?article reg:grantsRight ?right .
  ?right reg:rightType ?rightType .
} ORDER BY ?article`,
	},
	"obligations": {
		Name:        "obligations",
		Description: "Find articles that impose obligations",
		Query: `SELECT ?article ?title ?oblig ?obligType WHERE {
  ?article rdf:type reg:Article .
  ?article reg:title ?title .
  ?article reg:imposesObligation ?oblig .
  ?oblig reg:obligationType ?obligType .
} ORDER BY ?article`,
	},
	"right-types": {
		Name:        "right-types",
		Description: "List distinct right types found",
		Query: `SELECT DISTINCT ?rightType WHERE {
  ?right rdf:type reg:Right .
  ?right reg:rightType ?rightType .
}`,
	},
	"obligation-types": {
		Name:        "obligation-types",
		Description: "List distinct obligation types found",
		Query: `SELECT DISTINCT ?obligType WHERE {
  ?oblig rdf:type reg:Obligation .
  ?oblig reg:obligationType ?obligType .
}`,
	},
	"recitals": {
		Name:        "recitals",
		Description: "List all recitals",
		Query: `SELECT ?recital ?num WHERE {
  ?recital rdf:type reg:Recital .
  ?recital reg:number ?num .
} ORDER BY ?num LIMIT 20`,
	},
	"article-refs": {
		Name:        "article-refs",
		Description: "Find what articles reference a specific article (replace Art17 with article number)",
		Query: `SELECT ?article ?title WHERE {
  ?article reg:references ?target .
  ?article reg:title ?title .
  FILTER(CONTAINS(?target, "Art17"))
}`,
	},
	"search": {
		Name:        "search",
		Description: "Search for articles containing 'erasure' in title",
		Query: `SELECT ?article ?title WHERE {
  ?article rdf:type reg:Article .
  ?article reg:title ?title .
  FILTER(CONTAINS(?title, "erasure"))
}`,
	},
	"term-usage": {
		Name:        "term-usage",
		Description: "Find which articles use defined terms",
		Query: `SELECT ?article ?term WHERE {
  ?article reg:usesTerm ?termUri .
  ?termUri reg:term ?term .
} ORDER BY ?term LIMIT 50`,
	},
	"term-articles": {
		Name:        "term-articles",
		Description: "Find articles using a specific term (default: personal data)",
		Query: `SELECT ?article ?title WHERE {
  ?article reg:usesTerm ?termUri .
  ?termUri reg:normalizedTerm "personal data" .
  ?article reg:title ?title .
} ORDER BY ?article`,
	},
	"article-terms": {
		Name:        "article-terms",
		Description: "Find all terms used in Article 17",
		Query: `SELECT ?term WHERE {
  ?article reg:usesTerm ?termUri .
  ?termUri reg:term ?term .
  FILTER(CONTAINS(?article, "Art17"))
}`,
	},
	"hierarchy": {
		Name:        "hierarchy",
		Description: "Show document hierarchy (chapters contain articles)",
		Query: `SELECT ?chapter ?chapterTitle ?article ?articleTitle WHERE {
  ?chapter rdf:type reg:Chapter .
  ?chapter reg:title ?chapterTitle .
  ?chapter reg:contains ?article .
  ?article rdf:type reg:Article .
  ?article reg:title ?articleTitle .
} ORDER BY ?chapter ?article LIMIT 30`,
	},
	"most-referenced": {
		Name:        "most-referenced",
		Description: "Find the most referenced articles",
		Query: `SELECT ?target WHERE {
  ?source reg:references ?target .
} ORDER BY ?target`,
	},
	"definition-links": {
		Name:        "definition-links",
		Description: "Show terms and their defining articles",
		Query: `SELECT ?term ?article WHERE {
  ?termUri rdf:type reg:DefinedTerm .
  ?termUri reg:term ?term .
  ?termUri reg:definedIn ?article .
} ORDER BY ?term`,
	},
	"bidirectional": {
		Name:        "bidirectional",
		Description: "Show bidirectional reference relationships",
		Query: `SELECT ?source ?target WHERE {
  ?source reg:references ?target .
  ?target reg:referencedBy ?source .
} LIMIT 20`,
	},
	"describe-article": {
		Name:        "describe-article",
		Description: "Describe Article 17 (all triples where it appears as subject or object)",
		Query:       `DESCRIBE GDPR:Art17`,
	},
}

func printTemplates() {
	fmt.Println("Available query templates:")
	fmt.Println()
	for name, tmpl := range queryTemplates {
		fmt.Printf("  %-15s %s\n", name, tmpl.Description)
	}
	fmt.Println()
	fmt.Println("Usage: regula query --template <name>")
}

func loadAndIngest(source string) error {
	file, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer file.Close()

	parser := newParserWithPatterns()
	doc, err := parser.Parse(file)
	if err != nil {
		return fmt.Errorf("failed to parse document: %w", err)
	}

	baseURI := "https://regula.dev/regulations/"
	tripleStore = store.NewTripleStore()
	builder := store.NewGraphBuilder(tripleStore, baseURI)

	defExtractor := extract.NewDefinitionExtractor()
	refExtractor := extract.NewReferenceExtractor()
	semExtractor := extract.NewSemanticExtractor()
	resolver := extract.NewReferenceResolver(baseURI, extractDocID(source))
	resolver.IndexDocument(doc)

	_, err = builder.BuildComplete(doc, defExtractor, refExtractor, resolver, semExtractor)
	if err != nil {
		return fmt.Errorf("failed to build graph: %w", err)
	}

	executor = query.NewExecutor(tripleStore)
	graphLoaded = true
	graphPath = source
	loadedDocType = doc.Type
	return nil
}

func countArticles(doc *extract.Document) int {
	count := 0
	for _, ch := range doc.Chapters {
		for _, sec := range ch.Sections {
			count += len(sec.Articles)
		}
		count += len(ch.Articles)
	}
	return count
}

func printGateResult(gateResult *validate.GateResult) {
	statusLabel := "PASS"
	if !gateResult.Passed {
		statusLabel = "FAIL"
	}
	if gateResult.Skipped {
		statusLabel = "SKIP"
	}
	fmt.Printf("  [%s] Gate %s (score: %.0f%%)\n", statusLabel, gateResult.Gate, gateResult.Score*100)
	for _, gateError := range gateResult.Errors {
		fmt.Printf("    ERROR: %s\n", gateError.Message)
	}
	for _, gateWarning := range gateResult.Warnings {
		fmt.Printf("    WARN: %s\n", gateWarning.Message)
	}
}

func saveGraph(ts *store.TripleStore, path string) error {
	triples := ts.All()
	data := make([]map[string]string, len(triples))
	for i, t := range triples {
		data[i] = map[string]string{
			"subject":   t.Subject,
			"predicate": t.Predicate,
			"object":    t.Object,
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func validateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate graph consistency and extraction quality",
		Long: `Validate graph consistency and report extraction quality metrics.

Checks:
  - Reference resolution accuracy
  - Graph connectivity (orphan provisions)
  - Definition coverage (term usage)
  - Semantic extraction (rights/obligations)
  - Structure quality (completeness)

Validation Profiles:
  GDPR     - European General Data Protection Regulation
  CCPA     - California Consumer Privacy Act
  Generic  - Minimal criteria for unknown regulations

Link Validation (--check links):
  Validates external reference URIs with per-domain rate limiting.
  Use --report to save results to a file (JSON or Markdown).

Profile Auto-Generation:
  --suggest-profile    Analyze document and print suggested profile
  --generate-profile   Generate profile and save to YAML file
  --load-profile       Load custom validation profile from YAML file

Example:
  regula validate --source gdpr.txt
  regula validate --source gdpr.txt --threshold 0.85
  regula validate --source gdpr.txt --format json
  regula validate --source gdpr.txt --check references
  regula validate --source ccpa.txt --profile CCPA
  regula validate --source gdpr.txt --check gates
  regula validate --source gdpr.txt --check links
  regula validate --source gdpr.txt --check links --report links.json
  regula validate --source gdpr.txt --suggest-profile
  regula validate --source gdpr.txt --suggest-profile --format json
  regula validate --source gdpr.txt --generate-profile gdpr-custom.yaml
  regula validate --source gdpr.txt --load-profile gdpr-custom.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			source, _ := cmd.Flags().GetString("source")
			checkType, _ := cmd.Flags().GetString("check")
			formatStr, _ := cmd.Flags().GetString("format")
			baseURI, _ := cmd.Flags().GetString("base-uri")
			threshold, _ := cmd.Flags().GetFloat64("threshold")
			profileName, _ := cmd.Flags().GetString("profile")
			skipGates, _ := cmd.Flags().GetStringSlice("skip-gates")
			strictMode, _ := cmd.Flags().GetBool("strict")
			failOnWarn, _ := cmd.Flags().GetBool("fail-on-warn")
			reportPath, _ := cmd.Flags().GetString("report")
			suggestProfile, _ := cmd.Flags().GetBool("suggest-profile")
			generateProfilePath, _ := cmd.Flags().GetString("generate-profile")
			loadProfilePath, _ := cmd.Flags().GetString("load-profile")

			if source == "" {
				return fmt.Errorf("--source flag is required")
			}

			// Check if file exists
			fileInfo, err := os.Stat(source)
			if os.IsNotExist(err) {
				return fmt.Errorf("source file not found: %s", source)
			}
			if err != nil {
				return fmt.Errorf("failed to stat source: %w", err)
			}

			// Parse document
			file, err := os.Open(source)
			if err != nil {
				return fmt.Errorf("failed to open source: %w", err)
			}
			defer file.Close()

			parser := newParserWithPatterns()
			doc, err := parser.Parse(file)
			if err != nil {
				return fmt.Errorf("failed to parse document: %w", err)
			}

			// Extract definitions
			defExtractor := extract.NewDefinitionExtractor()
			definitions := defExtractor.ExtractDefinitions(doc)

			// Extract references
			refExtractor := extract.NewReferenceExtractor()
			refs := refExtractor.ExtractFromDocument(doc)

			// Create resolver and index document
			resolver := extract.NewReferenceResolver(baseURI, extractDocID(source))
			resolver.IndexDocument(doc)

			// Resolve all references
			resolved := resolver.ResolveAll(refs)

			// Extract term usages
			usageExtractor := extract.NewTermUsageExtractor(definitions)
			usages := usageExtractor.ExtractFromDocument(doc)

			// Extract semantics
			semExtractor := extract.NewSemanticExtractor()
			annotations := semExtractor.ExtractFromDocument(doc)

			// Build graph for connectivity check
			ts := store.NewTripleStore()
			builder := store.NewGraphBuilder(ts, baseURI)
			_, err = builder.BuildComplete(doc, defExtractor, refExtractor, resolver, semExtractor)
			if err != nil {
				return fmt.Errorf("failed to build graph: %w", err)
			}

			// Handle --suggest-profile: analyze and output suggested profile
			if suggestProfile {
				profileGenerator := validate.NewProfileGenerator()
				profileSuggestion := profileGenerator.SuggestProfile(doc, definitions, resolved, annotations, usages)

				switch formatStr {
				case "json":
					jsonData, jsonErr := profileSuggestion.ToJSON()
					if jsonErr != nil {
						return fmt.Errorf("failed to serialize profile suggestion: %w", jsonErr)
					}
					fmt.Println(string(jsonData))
				case "yaml":
					yamlData, yamlErr := profileSuggestion.ToYAML()
					if yamlErr != nil {
						return fmt.Errorf("failed to serialize profile suggestion: %w", yamlErr)
					}
					fmt.Print(string(yamlData))
				default:
					fmt.Print(profileSuggestion.String())
				}
				return nil
			}

			// Handle --generate-profile: generate and save profile to YAML file
			if generateProfilePath != "" {
				profileGenerator := validate.NewProfileGenerator()
				profileSuggestion := profileGenerator.SuggestProfile(doc, definitions, resolved, annotations, usages)

				if err := validate.SaveProfileToFile(profileSuggestion, generateProfilePath); err != nil {
					return fmt.Errorf("failed to save profile: %w", err)
				}
				fmt.Printf("Profile saved to: %s\n", generateProfilePath)
				fmt.Printf("Confidence: %.0f%%\n", profileSuggestion.Confidence*100)
				return nil
			}

			// Handle legacy check type for backwards compatibility
			if checkType == "references" {
				report := extract.GenerateReport(resolved)
				if formatStr == "json" {
					encoder := json.NewEncoder(os.Stdout)
					encoder.SetIndent("", "  ")
					return encoder.Encode(report)
				}
				fmt.Println(report.String())
				if report.ResolutionRate >= 0.85 {
					fmt.Printf("Status: PASS (resolution rate %.1f%% >= 85%%)\n", report.ResolutionRate*100)
				} else {
					fmt.Printf("Status: FAIL (resolution rate %.1f%% < 85%%)\n", report.ResolutionRate*100)
					return fmt.Errorf("resolution rate below 85%% target")
				}
				return nil
			}

			// Gate-based validation
			if checkType == "gates" {
				gateConfig := &validate.ValidationConfig{
					Thresholds: make(map[string]float64),
					SkipGates:  skipGates,
					StrictMode: strictMode,
					FailOnWarn: failOnWarn,
				}
				gatePipeline := validate.NewGatePipeline(gateConfig)
				gatePipeline.RegisterDefaultGates()

				gateContext := &validate.ValidationContext{
					SourcePath:         source,
					SourceSize:         fileInfo.Size(),
					Document:           doc,
					Definitions:        definitions,
					References:         refs,
					Semantics:          annotations,
					TermUsages:         usages,
					ResolvedReferences: resolved,
					TripleStore:        ts,
					Config:             gateConfig,
				}

				gateReport := gatePipeline.Run(gateContext)

				// Save report to file if --report flag is set
				if reportPath != "" {
					var reportData []byte
					if strings.HasSuffix(reportPath, ".html") {
						reportData = []byte(gateReport.ToHTML())
					} else if strings.HasSuffix(reportPath, ".md") {
						reportData = []byte(gateReport.ToMarkdown())
					} else {
						var jsonErr error
						reportData, jsonErr = gateReport.ToJSON()
						if jsonErr != nil {
							return fmt.Errorf("failed to serialize gate report: %w", jsonErr)
						}
					}
					if err := os.WriteFile(reportPath, reportData, 0644); err != nil {
						return fmt.Errorf("failed to write report: %w", err)
					}
					fmt.Printf("Report saved to: %s\n\n", reportPath)
				}

				switch formatStr {
				case "json":
					jsonData, err := gateReport.ToJSON()
					if err != nil {
						return fmt.Errorf("failed to serialize gate report: %w", err)
					}
					fmt.Println(string(jsonData))
				case "html":
					fmt.Print(gateReport.ToHTML())
				case "markdown":
					fmt.Print(gateReport.ToMarkdown())
				default:
					fmt.Print(gateReport.String())
				}

				if !gateReport.OverallPass {
					return fmt.Errorf("gate validation failed: overall score %.1f%%", gateReport.TotalScore*100)
				}
				return nil
			}

			// Link validation - validates external reference URIs
			if checkType == "links" {
				// Collect external URIs from resolved references
				externalURIs := collectExternalURIs(resolved)

				if len(externalURIs) == 0 {
					fmt.Println("No external URIs found to validate.")
					return nil
				}

				fmt.Printf("Validating %d external link(s)...\n\n", len(externalURIs))

				// Configure batch validator
				config := linkcheck.DefaultBatchConfig()
				config.DefaultRateLimit = 1 * time.Second
				config.DefaultTimeout = 30 * time.Second
				config.Concurrency = 3

				// Add domain-specific rate limits for known legal sources
				config.WithDomainConfig(&linkcheck.DomainConfig{
					Domain:    "eur-lex.europa.eu",
					RateLimit: 2 * time.Second,
					Timeout:   60 * time.Second,
				})
				config.WithDomainConfig(&linkcheck.DomainConfig{
					Domain:    "data.europa.eu",
					RateLimit: 2 * time.Second,
					Timeout:   60 * time.Second,
				})
				config.WithDomainConfig(&linkcheck.DomainConfig{
					Domain:    "uscode.house.gov",
					RateLimit: 2 * time.Second,
					Timeout:   60 * time.Second,
				})
				config.WithDomainConfig(&linkcheck.DomainConfig{
					Domain:    "ecfr.gov",
					RateLimit: 2 * time.Second,
					Timeout:   60 * time.Second,
				})
				config.WithDomainConfig(&linkcheck.DomainConfig{
					Domain:    "www.legislation.gov.uk",
					RateLimit: 2 * time.Second,
					Timeout:   60 * time.Second,
				})

				validator := linkcheck.NewBatchValidator(config)

				// Set progress callback for CLI feedback
				validator.SetProgressCallback(func(progress *linkcheck.ValidationProgress) {
					fmt.Printf("\r  Progress: %d/%d (%.1f%%) - %s",
						progress.CompletedLinks, progress.TotalLinks,
						progress.PercentComplete(), progress.CurrentDomain)
				})

				linkReport := validator.ValidateLinks(externalURIs)
				fmt.Printf("\r%s\n", strings.Repeat(" ", 80)) // Clear progress line

				// Output report
				if reportPath != "" {
					var reportData []byte
					var err error

					if strings.HasSuffix(reportPath, ".md") {
						reportData = []byte(linkReport.ToMarkdown())
					} else {
						reportData, err = linkReport.ToJSON()
						if err != nil {
							return fmt.Errorf("failed to serialize link report: %w", err)
						}
					}

					if err := os.WriteFile(reportPath, reportData, 0644); err != nil {
						return fmt.Errorf("failed to write report: %w", err)
					}
					fmt.Printf("Report saved to: %s\n\n", reportPath)
				}

				// Print summary
				if formatStr == "json" {
					jsonData, err := linkReport.ToJSON()
					if err != nil {
						return fmt.Errorf("failed to serialize link report: %w", err)
					}
					fmt.Println(string(jsonData))
				} else {
					fmt.Print(linkReport.String())
				}

				// Return error if too many broken links
				if linkReport.SuccessRate() < threshold*100 {
					return fmt.Errorf("link validation failed: success rate %.1f%% below threshold %.1f%%",
						linkReport.SuccessRate(), threshold*100)
				}

				return nil
			}

			// Full validation
			validator := validate.NewValidator(threshold)

			// Set profile: --load-profile takes priority, then --profile, then auto-detect
			if loadProfilePath != "" {
				customProfile, loadErr := validate.LoadProfileFromFile(loadProfilePath)
				if loadErr != nil {
					return fmt.Errorf("failed to load profile: %w", loadErr)
				}
				validator.SetProfile(customProfile)
				fmt.Printf("Loaded custom profile: %s\n\n", customProfile.Name)
			} else if profileName != "" {
				regType := validate.RegulationType(profileName)
				if profile, ok := validate.ValidationProfiles[regType]; ok {
					validator.SetRegulationType(regType)
					validator.SetProfile(profile)
				} else {
					return fmt.Errorf("unknown validation profile: %s\nAvailable profiles: GDPR, CCPA, Generic", profileName)
				}
			}

			result := validator.Validate(doc, resolved, definitions, usages, annotations, ts)

			// Save report to file if --report flag is set
			if reportPath != "" {
				var reportData []byte
				if strings.HasSuffix(reportPath, ".html") {
					reportData = []byte(result.ToHTML())
				} else if strings.HasSuffix(reportPath, ".md") {
					reportData = []byte(result.ToMarkdown())
				} else {
					var jsonErr error
					reportData, jsonErr = result.ToJSON()
					if jsonErr != nil {
						return fmt.Errorf("failed to serialize result: %w", jsonErr)
					}
				}
				if err := os.WriteFile(reportPath, reportData, 0644); err != nil {
					return fmt.Errorf("failed to write report: %w", err)
				}
				fmt.Printf("Report saved to: %s\n\n", reportPath)
			}

			// Output result
			switch formatStr {
			case "json":
				data, err := result.ToJSON()
				if err != nil {
					return fmt.Errorf("failed to serialize result: %w", err)
				}
				fmt.Println(string(data))
			case "html":
				fmt.Print(result.ToHTML())
			case "markdown":
				fmt.Print(result.ToMarkdown())
			default:
				fmt.Println(result.String())
			}

			// Return error if validation failed
			if result.Status == validate.StatusFail {
				return fmt.Errorf("validation failed: overall score %.1f%% below threshold %.1f%%",
					result.OverallScore*100, result.Threshold*100)
			}

			return nil
		},
	}

	cmd.Flags().StringP("source", "s", "", "Source document path")
	cmd.Flags().String("check", "all", "What to check (all, references, gates, links)")
	cmd.Flags().StringP("format", "f", "text", "Output format (text, json, html, markdown)")
	cmd.Flags().String("base-uri", "https://regula.dev/regulations/", "Base URI for the graph")
	cmd.Flags().Float64("threshold", 0.80, "Pass/fail threshold (0.0-1.0)")
	cmd.Flags().String("profile", "", "Validation profile (GDPR, CCPA, Generic) - auto-detected if not specified")
	cmd.Flags().StringSlice("skip-gates", []string{}, "Gates to skip (V0,V1,V2,V3)")
	cmd.Flags().Bool("strict", false, "Halt pipeline on gate failure")
	cmd.Flags().Bool("fail-on-warn", false, "Halt pipeline on gate warnings")
	cmd.Flags().String("report", "", "Save validation report to file (format based on extension: .html, .md, .json)")
	cmd.Flags().Bool("suggest-profile", false, "Analyze document and print suggested validation profile")
	cmd.Flags().String("generate-profile", "", "Generate validation profile and save to YAML file")
	cmd.Flags().String("load-profile", "", "Load custom validation profile from YAML file")

	return cmd
}

func impactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "impact",
		Short: "Analyze impact of regulatory changes",
		Long: `Analyze the impact of changes to a provision.

Performs comprehensive impact analysis including:
  - Direct impact: provisions that reference the target
  - Reverse impact: provisions the target references
  - Transitive impact: configurable depth traversal

Examples:
  regula impact --provision "Art17" --source gdpr.txt
  regula impact --provision "GDPR:Art17" --depth 2 --source gdpr.txt
  regula impact --provision "Art17" --direction incoming --source gdpr.txt
  regula impact --provision "Art17" --format json --source gdpr.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			provision, _ := cmd.Flags().GetString("provision")
			source, _ := cmd.Flags().GetString("source")
			depth, _ := cmd.Flags().GetInt("depth")
			directionStr, _ := cmd.Flags().GetString("direction")
			formatStr, _ := cmd.Flags().GetString("format")
			baseURI, _ := cmd.Flags().GetString("base-uri")

			if provision == "" {
				return fmt.Errorf("--provision flag is required")
			}

			if source == "" {
				return fmt.Errorf("--source flag is required")
			}

			// Load graph if source specified
			if !graphLoaded || graphPath != source {
				if err := loadAndIngest(source); err != nil {
					return err
				}
			}

			// Parse direction
			var direction analysis.ImpactDirection
			switch directionStr {
			case "incoming":
				direction = analysis.DirectionIncoming
			case "outgoing":
				direction = analysis.DirectionOutgoing
			case "both":
				direction = analysis.DirectionBoth
			default:
				return fmt.Errorf("invalid direction: %s (use incoming, outgoing, or both)", directionStr)
			}

			// Create analyzer and run analysis
			analyzer := analysis.NewImpactAnalyzer(tripleStore, baseURI)
			result := analyzer.AnalyzeByID(provision, depth, direction)

			// Output result
			switch formatStr {
			case "json":
				data, err := result.ToJSON()
				if err != nil {
					return fmt.Errorf("failed to serialize result: %w", err)
				}
				fmt.Println(string(data))
			case "table":
				fmt.Println(result.FormatTable())
			default:
				fmt.Println(result.String())
			}

			return nil
		},
	}

	cmd.Flags().StringP("provision", "p", "", "Provision ID to analyze (e.g., Art17, GDPR:Art17)")
	cmd.Flags().IntP("depth", "d", 2, "Transitive dependency depth (1=direct only)")
	cmd.Flags().StringP("direction", "D", "both", "Direction of analysis (incoming, outgoing, both)")
	cmd.Flags().StringP("source", "s", "", "Source document to analyze")
	cmd.Flags().StringP("format", "f", "text", "Output format (text, json, table)")
	cmd.Flags().String("base-uri", "https://regula.dev/regulations/", "Base URI for the graph")

	return cmd
}

func matchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "match",
		Short: "Match a scenario to applicable provisions",
		Long: `Match a compliance scenario to applicable provisions in the regulation.

Finds provisions that are:
  - DIRECT: Directly applicable (grants relevant rights or imposes relevant obligations)
  - TRIGGERED: Triggered by direct provisions (referenced by or references direct matches)
  - RELATED: Related by keywords

Built-in scenarios:
  consent_withdrawal  - Data subject withdraws consent
  access_request     - Data subject requests access to data
  erasure_request    - Data subject requests erasure of data
  data_breach        - Personal data breach handling

Examples:
  regula match --scenario consent_withdrawal --source gdpr.txt
  regula match --scenario access_request --source gdpr.txt --format json
  regula match --scenario data_breach --source gdpr.txt --format table`,
		RunE: func(cmd *cobra.Command, args []string) error {
			scenarioName, _ := cmd.Flags().GetString("scenario")
			source, _ := cmd.Flags().GetString("source")
			formatStr, _ := cmd.Flags().GetString("format")
			baseURI, _ := cmd.Flags().GetString("base-uri")
			listScenarios, _ := cmd.Flags().GetBool("list-scenarios")

			// List available scenarios
			if listScenarios {
				fmt.Println("Available scenarios:")
				for name, s := range simulate.PredefinedScenarios {
					fmt.Printf("  %-20s %s\n", name, s.Description)
				}
				return nil
			}

			if scenarioName == "" {
				return fmt.Errorf("--scenario flag is required\nUse --list-scenarios to see available scenarios")
			}

			if source == "" {
				return fmt.Errorf("--source flag is required")
			}

			// Get scenario
			scenario, ok := simulate.PredefinedScenarios[scenarioName]
			if !ok {
				return fmt.Errorf("unknown scenario: %s\nUse --list-scenarios to see available scenarios", scenarioName)
			}

			// Parse document
			file, err := os.Open(source)
			if err != nil {
				return fmt.Errorf("failed to open source: %w", err)
			}
			defer file.Close()

			parser := newParserWithPatterns()
			doc, err := parser.Parse(file)
			if err != nil {
				return fmt.Errorf("failed to parse document: %w", err)
			}

			// Build graph
			ts := store.NewTripleStore()
			builder := store.NewGraphBuilder(ts, baseURI)

			defExtractor := extract.NewDefinitionExtractor()
			refExtractor := extract.NewReferenceExtractor()
			semExtractor := extract.NewSemanticExtractor()
			resolver := extract.NewReferenceResolver(baseURI, extractDocID(source))
			resolver.IndexDocument(doc)

			_, err = builder.BuildComplete(doc, defExtractor, refExtractor, resolver, semExtractor)
			if err != nil {
				return fmt.Errorf("failed to build graph: %w", err)
			}

			// Extract semantic annotations
			annotations := semExtractor.ExtractFromDocument(doc)

			// Create matcher and match
			matcher := simulate.NewProvisionMatcher(ts, baseURI, annotations, doc)
			result := matcher.Match(scenario)

			// Output result
			switch formatStr {
			case "json":
				data, err := result.ToJSON()
				if err != nil {
					return fmt.Errorf("failed to serialize result: %w", err)
				}
				fmt.Println(string(data))
			case "table":
				fmt.Println(result.FormatTable())
			default:
				fmt.Println(result.String())
			}

			return nil
		},
	}

	cmd.Flags().StringP("scenario", "S", "", "Scenario name (consent_withdrawal, access_request, etc.)")
	cmd.Flags().StringP("source", "s", "", "Source document to analyze")
	cmd.Flags().StringP("format", "f", "text", "Output format (text, json, table)")
	cmd.Flags().String("base-uri", "https://regula.dev/regulations/", "Base URI for the graph")
	cmd.Flags().Bool("list-scenarios", false, "List available scenarios")

	return cmd
}

func simulateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "simulate",
		Short: "Simulate a compliance scenario",
		Long: `Evaluate a compliance scenario against the regulation graph.

Example:
  regula simulate --scenario consent-withdrawal.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			scenario, _ := cmd.Flags().GetString("scenario")

			if scenario == "" {
				return fmt.Errorf("--scenario flag is required")
			}

			fmt.Printf("Simulating scenario: %s\n", scenario)
			fmt.Println("\nScenario Evaluation:")
			fmt.Println("  - Loading scenario definition...")
			fmt.Println("  - Finding applicable provisions...")
			fmt.Println("  - Evaluating obligations...")
			fmt.Println("  - Checking timelines...")
			fmt.Println("  - Generating compliance report...")
			fmt.Println("\n[Not implemented - Phase 5]")
			return nil
		},
	}

	cmd.Flags().StringP("scenario", "s", "", "Scenario file path (YAML)")
	cmd.Flags().StringP("output", "o", "report", "Output format (report, json)")

	return cmd
}

func auditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Generate audit trail for a decision",
		Long: `Generate an audit trail showing the reasoning chain for a decision.

Example:
  regula audit --decision "data-deletion-request-123"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			decision, _ := cmd.Flags().GetString("decision")

			if decision == "" {
				return fmt.Errorf("--decision flag is required")
			}

			fmt.Printf("Generating audit trail for: %s\n", decision)
			fmt.Println("\nAudit Trail:")
			fmt.Println("  - Decision ID: " + decision)
			fmt.Println("  - Timestamp: [calculating...]")
			fmt.Println("  - Applicable provisions: [calculating...]")
			fmt.Println("  - Reasoning chain: [calculating...]")
			fmt.Println("  - Proofs verified: [calculating...]")
			fmt.Println("\n[Not implemented - Phase 6]")
			return nil
		},
	}

	cmd.Flags().StringP("decision", "d", "", "Decision ID to audit")
	cmd.Flags().StringP("output", "o", "text", "Output format (text, json, prov)")

	return cmd
}

func exportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export the relationship graph for visualization",
		Long: `Export the regulation relationship graph in various formats.

Supported formats:
  - json:    JSON graph format with nodes and edges
  - dot:     DOT format for Graphviz visualization
  - turtle:  W3C Turtle (TTL) RDF serialization
  - jsonld:  JSON-LD (Linked Data) format with @context
  - rdfxml:  RDF/XML format for legacy system compatibility
  - summary: Relationship statistics and summary

Use --eli to add ELI (European Legislation Identifier) vocabulary triples
alongside reg: triples for EU documents (regulation, directive, decision).

JSON-LD Options:
  --expanded  Output expanded JSON-LD (full URIs, no @context) instead of compact form

Example:
  regula export --source gdpr.txt --format json --output graph.json
  regula export --source gdpr.txt --format dot --output graph.dot
  regula export --source gdpr.txt --format turtle --output graph.ttl
  regula export --source gdpr.txt --format turtle --eli --output graph-eli.ttl
  regula export --source gdpr.txt --format jsonld --output graph.jsonld
  regula export --source gdpr.txt --format jsonld --expanded --output graph-expanded.jsonld
  regula export --source gdpr.txt --format rdfxml --output graph.rdf
  regula export --source gdpr.txt --format summary`,
		RunE: func(cmd *cobra.Command, args []string) error {
			source, _ := cmd.Flags().GetString("source")
			formatStr, _ := cmd.Flags().GetString("format")
			output, _ := cmd.Flags().GetString("output")
			relationsOnly, _ := cmd.Flags().GetBool("relations-only")
			enableELI, _ := cmd.Flags().GetBool("eli")
			expandedJSONLD, _ := cmd.Flags().GetBool("expanded")

			if source == "" {
				return fmt.Errorf("--source flag is required")
			}

			// Load and ingest if needed
			if !graphLoaded || graphPath != source {
				if err := loadAndIngest(source); err != nil {
					return err
				}
			}

			// Optionally enrich with ELI vocabulary
			if enableELI {
				eliStats := store.EnrichWithELI(tripleStore, loadedDocType)
				if eliStats.TotalTriples > 0 {
					fmt.Printf("ELI enrichment: %d triples added (%d class, %d property)\n",
						eliStats.TotalTriples, eliStats.ClassTriples, eliStats.PropertyTriples)
				} else if !store.IsEUDocumentType(loadedDocType) {
					fmt.Println("ELI enrichment skipped: document is not an EU legislative type")
				}
			}

			switch formatStr {
			case "json":
				var export *store.GraphExport
				if relationsOnly {
					export = store.ExportRelationshipSubgraph(tripleStore)
				} else {
					export = store.ExportGraph(tripleStore)
				}

				data, err := export.ToJSON()
				if err != nil {
					return fmt.Errorf("failed to serialize graph: %w", err)
				}

				if output != "" {
					if err := os.WriteFile(output, data, 0644); err != nil {
						return fmt.Errorf("failed to write file: %w", err)
					}
					fmt.Printf("Graph exported to: %s\n", output)
					fmt.Printf("  Nodes: %d\n", export.Stats.TotalNodes)
					fmt.Printf("  Edges: %d\n", export.Stats.TotalEdges)
				} else {
					fmt.Println(string(data))
				}

			case "dot":
				export := store.ExportRelationshipSubgraph(tripleStore)
				dotContent := export.ToDOT()

				if output != "" {
					if err := os.WriteFile(output, []byte(dotContent), 0644); err != nil {
						return fmt.Errorf("failed to write file: %w", err)
					}
					fmt.Printf("DOT graph exported to: %s\n", output)
					fmt.Println("\nTo visualize with Graphviz:")
					fmt.Printf("  dot -Tpng %s -o graph.png\n", output)
					fmt.Printf("  dot -Tsvg %s -o graph.svg\n", output)
				} else {
					fmt.Println(dotContent)
				}

			case "turtle":
				serializer := store.NewTurtleSerializer()
				turtleOutput := serializer.Serialize(tripleStore)

				if output != "" {
					if err := os.WriteFile(output, []byte(turtleOutput), 0644); err != nil {
						return fmt.Errorf("failed to write file: %w", err)
					}
					fmt.Printf("Turtle graph exported to: %s\n", output)
					fmt.Printf("  Triples: %d\n", tripleStore.Count())
				} else {
					fmt.Print(turtleOutput)
				}

			case "jsonld":
				var serializer *store.JSONLDSerializer
				if expandedJSONLD {
					serializer = store.NewJSONLDSerializer(store.WithExpandedForm())
				} else {
					serializer = store.NewJSONLDSerializer(store.WithCompactForm())
				}

				jsonldOutput, err := serializer.Serialize(tripleStore)
				if err != nil {
					return fmt.Errorf("failed to serialize JSON-LD: %w", err)
				}

				if output != "" {
					if err := os.WriteFile(output, jsonldOutput, 0644); err != nil {
						return fmt.Errorf("failed to write file: %w", err)
					}
					fmt.Printf("JSON-LD graph exported to: %s\n", output)
					fmt.Printf("  Triples: %d\n", tripleStore.Count())
					if expandedJSONLD {
						fmt.Println("  Format: expanded (full URIs)")
					} else {
						fmt.Println("  Format: compact (with @context)")
					}
				} else {
					fmt.Print(string(jsonldOutput))
				}

			case "rdfxml", "xml":
				rdfxmlSerializer := store.NewRDFXMLSerializer()
				rdfxmlOutput := rdfxmlSerializer.Serialize(tripleStore)

				if output != "" {
					if err := os.WriteFile(output, []byte(rdfxmlOutput), 0644); err != nil {
						return fmt.Errorf("failed to write file: %w", err)
					}
					fmt.Printf("RDF/XML graph exported to: %s\n", output)
					fmt.Printf("  Triples: %d\n", tripleStore.Count())
				} else {
					fmt.Print(rdfxmlOutput)
				}

			case "summary":
				summary := store.CalculateRelationshipSummary(tripleStore)

				fmt.Println("Relationship Graph Summary")
				fmt.Println("==========================")
				fmt.Printf("\nTotal relationships: %d\n\n", summary.TotalRelationships)

				fmt.Println("Relationship Types:")
				for relType, count := range summary.RelationshipCounts {
					fmt.Printf("  %-25s %d\n", relType, count)
				}

				if len(summary.MostReferencedArticles) > 0 {
					fmt.Println("\nMost Referenced Articles:")
					for _, arc := range summary.MostReferencedArticles {
						fmt.Printf("  Article %d: %d incoming references\n", arc.ArticleNum, arc.Count)
					}
				}

				if len(summary.MostReferencingArticles) > 0 {
					fmt.Println("\nArticles With Most Outgoing References:")
					for _, arc := range summary.MostReferencingArticles {
						fmt.Printf("  Article %d: %d outgoing references\n", arc.ArticleNum, arc.Count)
					}
				}

				if summary.ExternalRefCount > 0 {
					fmt.Printf("\nExternal References: %d total, %d unique targets\n",
						summary.ExternalRefCount, len(summary.ExternalRefTargets))
					if len(summary.TopExternalTargets) > 0 {
						fmt.Println("\nTop External Reference Targets:")
						for _, ext := range summary.TopExternalTargets {
							fmt.Printf("  %-40s %d\n", ext.Target, ext.Count)
						}
					}
				}

			default:
				return fmt.Errorf("unknown format: %s (use json, dot, turtle, jsonld, rdfxml, or summary)", formatStr)
			}

			return nil
		},
	}

	cmd.Flags().StringP("source", "s", "", "Source document path")
	cmd.Flags().StringP("format", "f", "summary", "Output format (json, dot, turtle, jsonld, rdfxml, summary)")
	cmd.Flags().StringP("output", "o", "", "Output file path")
	cmd.Flags().Bool("relations-only", true, "Export only relationship edges (default: true)")
	cmd.Flags().Bool("eli", false, "Enrich with ELI (European Legislation Identifier) vocabulary for EU documents")
	cmd.Flags().Bool("expanded", false, "Output expanded JSON-LD (full URIs, no @context) instead of compact form")

	return cmd
}

func compareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare",
		Short: "Compare regulation documents or House Rules versions",
		Long: `Compare two or more regulation documents to find shared definitions,
rights, obligations, and external reference targets.

Outputs structural comparison, concept overlaps, and external reference analysis.

Commands:
  rules     Compare two versions of House Rules (e.g., 118th vs 119th Congress)

Example:
  regula compare --sources testdata/gdpr.txt,testdata/ccpa.txt
  regula compare --sources testdata/gdpr.txt,testdata/ccpa.txt --format json
  regula compare --sources testdata/gdpr.txt,testdata/ccpa.txt,testdata/eu-ai-act.txt --format dot --output comparison.dot
  regula compare rules --base house-rules-118th.txt --target house-rules-119th.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			sourcesStr, _ := cmd.Flags().GetString("sources")
			formatStr, _ := cmd.Flags().GetString("format")
			output, _ := cmd.Flags().GetString("output")

			if sourcesStr == "" {
				return fmt.Errorf("--sources flag is required (comma-separated list of document paths)")
			}

			sources := strings.Split(sourcesStr, ",")
			if len(sources) < 2 {
				return fmt.Errorf("at least 2 source documents are required for comparison")
			}

			// Trim whitespace from paths
			for i := range sources {
				sources[i] = strings.TrimSpace(sources[i])
			}

			fmt.Printf("Comparing %d documents...\n\n", len(sources))
			startTime := time.Now()

			crossRefAnalyzer := analysis.NewCrossRefAnalyzer()

			// Ingest each document into its own store
			for _, sourcePath := range sources {
				if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
					return fmt.Errorf("source file not found: %s", sourcePath)
				}

				file, err := os.Open(sourcePath)
				if err != nil {
					return fmt.Errorf("failed to open %s: %w", sourcePath, err)
				}

				parser := newParserWithPatterns()
				doc, err := parser.Parse(file)
				file.Close()
				if err != nil {
					return fmt.Errorf("failed to parse %s: %w", sourcePath, err)
				}

				docStore := store.NewTripleStore()
				baseURI := "https://regula.dev/regulations/"
				builder := store.NewGraphBuilder(docStore, baseURI)
				defExtractor := extract.NewDefinitionExtractor()
				refExtractor := extract.NewReferenceExtractor()
				semExtractor := extract.NewSemanticExtractor()

				docID := extractDocID(sourcePath)
				resolver := extract.NewReferenceResolver(baseURI, docID)
				resolver.IndexDocument(doc)

				_, err = builder.BuildComplete(doc, defExtractor, refExtractor, resolver, semExtractor)
				if err != nil {
					return fmt.Errorf("failed to build graph for %s: %w", sourcePath, err)
				}

				label := doc.Title
				if label == "" {
					label = docID
				}
				crossRefAnalyzer.AddDocument(docID, label, docStore)
				fmt.Printf("  Loaded %s: %d triples\n", docID, docStore.Count())
			}

			fmt.Printf("\nAnalysis completed in %s\n\n", time.Since(startTime))

			// Run analysis based on number of documents
			if len(sources) == 2 {
				docIDs := make([]string, 2)
				for i, src := range sources {
					docIDs[i] = extractDocID(src)
				}
				comparison := crossRefAnalyzer.CompareDocuments(docIDs[0], docIDs[1])

				switch formatStr {
				case "table":
					fmt.Print(comparison.String())
				case "json":
					jsonData, err := comparison.ToJSON()
					if err != nil {
						return fmt.Errorf("failed to serialize JSON: %w", err)
					}
					if output != "" {
						if err := os.WriteFile(output, jsonData, 0644); err != nil {
							return fmt.Errorf("failed to write file: %w", err)
						}
						fmt.Printf("Comparison exported to: %s\n", output)
					} else {
						fmt.Println(string(jsonData))
					}
				case "dot":
					dotContent := comparison.ToDOT()
					if output != "" {
						if err := os.WriteFile(output, []byte(dotContent), 0644); err != nil {
							return fmt.Errorf("failed to write file: %w", err)
						}
						fmt.Printf("DOT graph exported to: %s\n", output)
						fmt.Println("\nTo visualize with Graphviz:")
						fmt.Printf("  dot -Tpng %s -o comparison.png\n", output)
					} else {
						fmt.Println(dotContent)
					}
				default:
					return fmt.Errorf("unknown format: %s (use table, json, or dot)", formatStr)
				}
			} else {
				result := crossRefAnalyzer.Analyze()

				switch formatStr {
				case "table":
					fmt.Print(result.FormatTable())
					fmt.Println()
					fmt.Print(result.String())
				case "json":
					jsonData, err := result.ToJSON()
					if err != nil {
						return fmt.Errorf("failed to serialize JSON: %w", err)
					}
					if output != "" {
						if err := os.WriteFile(output, jsonData, 0644); err != nil {
							return fmt.Errorf("failed to write file: %w", err)
						}
						fmt.Printf("Analysis exported to: %s\n", output)
					} else {
						fmt.Println(string(jsonData))
					}
				case "dot":
					dotContent := result.ToDOT()
					if output != "" {
						if err := os.WriteFile(output, []byte(dotContent), 0644); err != nil {
							return fmt.Errorf("failed to write file: %w", err)
						}
						fmt.Printf("DOT graph exported to: %s\n", output)
						fmt.Println("\nTo visualize with Graphviz:")
						fmt.Printf("  dot -Tpng %s -o comparison.png\n", output)
					} else {
						fmt.Println(dotContent)
					}
				default:
					return fmt.Errorf("unknown format: %s (use table, json, or dot)", formatStr)
				}
			}

			return nil
		},
	}

	cmd.Flags().String("sources", "", "Comma-separated list of source document paths")
	cmd.Flags().StringP("format", "f", "table", "Output format (table, json, dot)")
	cmd.Flags().StringP("output", "o", "", "Output file path")

	cmd.AddCommand(compareRulesCmd())

	return cmd
}

func compareRulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rules",
		Short: "Compare two versions of House Rules",
		Long: `Compare two versions of House Rules (e.g., 118th vs 119th Congress) to track
rule changes, added/removed/modified clauses, and structural differences.

The output shows:
  - Summary of rules modified, clauses added/removed/modified
  - Detailed changes organized by rule
  - Similarity scores for modified clauses
  - Change summaries (minor, moderate, substantial, major)

Example:
  regula compare rules --base house-rules-118th.txt --target house-rules-119th.txt
  regula compare rules --base house-rules-118th.txt --target house-rules-119th.txt --format json
  regula compare rules --base 118th.txt --target 119th.txt --threshold 80`,
		RunE: func(cmd *cobra.Command, args []string) error {
			basePath, _ := cmd.Flags().GetString("base")
			targetPath, _ := cmd.Flags().GetString("target")
			formatStr, _ := cmd.Flags().GetString("format")
			output, _ := cmd.Flags().GetString("output")
			threshold, _ := cmd.Flags().GetInt("threshold")

			if basePath == "" || targetPath == "" {
				return fmt.Errorf("both --base and --target flags are required")
			}

			// Read base file
			baseContent, err := os.ReadFile(basePath)
			if err != nil {
				return fmt.Errorf("failed to read base file: %w", err)
			}

			// Read target file
			targetContent, err := os.ReadFile(targetPath)
			if err != nil {
				return fmt.Errorf("failed to read target file: %w", err)
			}

			// Extract version labels from filenames
			baseVersion := extractCongressLabel(basePath)
			targetVersion := extractCongressLabel(targetPath)

			// Create differ and compare
			differ := extract.NewRulesDiffer(string(baseContent), string(targetContent))
			report := differ.Compare(baseVersion, targetVersion)

			var outputContent []byte

			switch formatStr {
			case "table", "text":
				if threshold > 0 {
					// Filter to significant changes only
					significant := report.GetSignificantChanges(threshold)
					fmt.Printf("Showing changes with similarity <= %d%%\n\n", threshold)
					for _, change := range significant {
						fmt.Printf("Rule %s, Clause %s: %s", change.Rule, change.Clause, change.Type)
						if change.Summary != "" {
							fmt.Printf(" - %s", change.Summary)
						}
						if change.SimilarityScore > 0 {
							fmt.Printf(" (%d%% similar)", change.SimilarityScore)
						}
						fmt.Println()
					}
				} else {
					fmt.Print(report.String())
				}
			case "json":
				jsonData, err := report.ToJSON()
				if err != nil {
					return fmt.Errorf("failed to serialize JSON: %w", err)
				}
				outputContent = jsonData
				if output == "" {
					fmt.Println(string(jsonData))
				}
			default:
				return fmt.Errorf("unknown format: %s (use table, text, or json)", formatStr)
			}

			if output != "" && len(outputContent) > 0 {
				if err := os.WriteFile(output, outputContent, 0644); err != nil {
					return fmt.Errorf("failed to write output file: %w", err)
				}
				fmt.Printf("Report written to: %s\n", output)
			}

			return nil
		},
	}

	cmd.Flags().String("base", "", "Path to the base (older) House Rules file")
	cmd.Flags().String("target", "", "Path to the target (newer) House Rules file")
	cmd.Flags().StringP("format", "f", "table", "Output format (table, text, json)")
	cmd.Flags().StringP("output", "o", "", "Output file path")
	cmd.Flags().Int("threshold", 0, "Show only changes with similarity <= threshold (0 = show all)")

	return cmd
}

// extractCongressLabel extracts a Congress label from a filename.
func extractCongressLabel(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))

	// Try to find congress number patterns like "118th", "119th"
	for _, suffix := range []string{"th", "st", "nd", "rd"} {
		if idx := strings.Index(base, suffix); idx > 0 {
			// Find the start of the number
			start := idx
			for start > 0 && base[start-1] >= '0' && base[start-1] <= '9' {
				start--
			}
			if start < idx {
				return base[start:idx+len(suffix)] + " Congress"
			}
		}
	}

	// Fall back to filename
	return base
}

func refsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refs",
		Short: "Analyze references in a regulation document",
		Long: `Analyze internal and external references in a regulation document.

Shows clustered external references, reference frequency, and per-provision details.

For House Rules, use --format matrix to generate a rule-to-rule cross-reference
adjacency matrix showing inter-rule dependencies.

Example:
  regula refs --source testdata/gdpr.txt
  regula refs --source testdata/gdpr.txt --format json
  regula refs --source testdata/eu-ai-act.txt --external-only
  regula refs --source house-rules-119th.txt --format matrix
  regula refs --source house-rules-119th.txt --format matrix-csv
  regula refs --source house-rules-119th.txt --format matrix-svg --output matrix.svg`,
		RunE: func(cmd *cobra.Command, args []string) error {
			source, _ := cmd.Flags().GetString("source")
			formatStr, _ := cmd.Flags().GetString("format")
			externalOnly, _ := cmd.Flags().GetBool("external-only")
			output, _ := cmd.Flags().GetString("output")

			if source == "" {
				return fmt.Errorf("--source flag is required")
			}

			if _, err := os.Stat(source); os.IsNotExist(err) {
				return fmt.Errorf("source file not found: %s", source)
			}

			file, err := os.Open(source)
			if err != nil {
				return fmt.Errorf("failed to open source: %w", err)
			}

			parser := newParserWithPatterns()
			doc, err := parser.Parse(file)
			file.Close()
			if err != nil {
				return fmt.Errorf("failed to parse document: %w", err)
			}

			docStore := store.NewTripleStore()
			baseURI := "https://regula.dev/regulations/"
			builder := store.NewGraphBuilder(docStore, baseURI)
			defExtractor := extract.NewDefinitionExtractor()
			refExtractor := extract.NewReferenceExtractor()
			semExtractor := extract.NewSemanticExtractor()

			docID := extractDocID(source)
			resolver := extract.NewReferenceResolver(baseURI, docID)
			resolver.IndexDocument(doc)

			_, err = builder.BuildComplete(doc, defExtractor, refExtractor, resolver, semExtractor)
			if err != nil {
				return fmt.Errorf("failed to build graph: %w", err)
			}

			label := doc.Title
			if label == "" {
				label = docID
			}

			crossRefAnalyzer := analysis.NewCrossRefAnalyzer()
			crossRefAnalyzer.AddDocument(docID, label, docStore)

			// Handle matrix formats
			if strings.HasPrefix(formatStr, "matrix") {
				report := analysis.GenerateMatrixReport(docStore)

				if report.Matrix.TotalRefs == 0 {
					fmt.Println("No cross-references found for matrix visualization.")
					fmt.Println("Matrix format works best with House Rules or similar documents with rule-to-rule references.")
					return nil
				}

				var outputContent string
				switch formatStr {
				case "matrix":
					// ASCII table format
					outputContent = report.String()
				case "matrix-csv":
					outputContent = report.Matrix.ToCSV()
				case "matrix-svg":
					outputContent = report.Matrix.ToSVGHeatmap()
				case "matrix-json":
					jsonData, err := report.ToJSON()
					if err != nil {
						return fmt.Errorf("failed to serialize JSON: %w", err)
					}
					outputContent = string(jsonData)
				default:
					return fmt.Errorf("unknown matrix format: %s (use matrix, matrix-csv, matrix-svg, or matrix-json)", formatStr)
				}

				if output != "" {
					if err := os.WriteFile(output, []byte(outputContent), 0644); err != nil {
						return fmt.Errorf("failed to write file: %w", err)
					}
					fmt.Printf("Matrix report exported to: %s\n", output)
				} else {
					fmt.Print(outputContent)
				}
				return nil
			}

			if externalOnly {
				report := crossRefAnalyzer.AnalyzeExternalRefs(docID)

				switch formatStr {
				case "table":
					fmt.Print(report.String())
				case "json":
					jsonData, err := report.ToJSON()
					if err != nil {
						return fmt.Errorf("failed to serialize JSON: %w", err)
					}
					if output != "" {
						if err := os.WriteFile(output, jsonData, 0644); err != nil {
							return fmt.Errorf("failed to write file: %w", err)
						}
						fmt.Printf("External reference report exported to: %s\n", output)
					} else {
						fmt.Println(string(jsonData))
					}
				default:
					return fmt.Errorf("unknown format: %s (use table or json)", formatStr)
				}
			} else {
				// Full reference summary (internal + external)
				summary := store.CalculateRelationshipSummary(docStore)

				fmt.Printf("Reference Analysis: %s\n", label)
				fmt.Println("=" + strings.Repeat("=", 50))
				fmt.Printf("\nTotal relationships: %d\n", summary.TotalRelationships)
				fmt.Printf("Internal references: %d\n", summary.RelationshipCounts["reg:references"])
				fmt.Printf("External references: %d\n\n", summary.ExternalRefCount)

				if summary.ExternalRefCount > 0 {
					fmt.Printf("External Reference Targets (%d unique):\n", len(summary.ExternalRefTargets))
					for _, ext := range summary.TopExternalTargets {
						fmt.Printf("  %-45s %d\n", ext.Target, ext.Count)
					}
					fmt.Println()
				}

				if len(summary.MostReferencedArticles) > 0 {
					fmt.Println("Most Referenced Articles (internal):")
					for _, arc := range summary.MostReferencedArticles {
						fmt.Printf("  Article %d: %d incoming references\n", arc.ArticleNum, arc.Count)
					}
					fmt.Println()
				}

				if len(summary.MostReferencingArticles) > 0 {
					fmt.Println("Articles With Most Outgoing References:")
					for _, arc := range summary.MostReferencingArticles {
						fmt.Printf("  Article %d: %d outgoing references\n", arc.ArticleNum, arc.Count)
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringP("source", "s", "", "Source document path")
	cmd.Flags().StringP("format", "f", "table", "Output format (table, json, matrix, matrix-csv, matrix-svg, matrix-json)")
	cmd.Flags().StringP("output", "o", "", "Output file path")
	cmd.Flags().Bool("external-only", false, "Show only external references")

	return cmd
}

// extractDocID extracts a document identifier from a file path.
// newParserWithPatterns creates a parser with the pattern registry loaded from
// the patterns directory. Falls back to a plain parser if patterns cannot be loaded.
func newParserWithPatterns() *extract.Parser {
	registry := pattern.NewRegistry()
	// Try common pattern directory locations relative to the binary
	for _, dir := range []string{"patterns", "../../patterns", "../patterns"} {
		if _, err := os.Stat(dir); err == nil {
			if err := registry.LoadDirectory(dir); err == nil {
				return extract.NewParserWithRegistry(registry)
			}
		}
	}
	return extract.NewParser()
}

func extractDocID(sourcePath string) string {
	baseName := filepath.Base(sourcePath)
	// Remove extension
	if idx := strings.LastIndex(baseName, "."); idx != -1 {
		baseName = baseName[:idx]
	}
	return strings.ToUpper(baseName)
}

// collectExternalURIs extracts external reference URIs from resolved references.
func collectExternalURIs(resolved []*extract.ResolvedReference) []linkcheck.LinkInput {
	seen := make(map[string]bool)
	var links []linkcheck.LinkInput

	for _, ref := range resolved {
		// Check target URI
		if ref.TargetURI != "" && isExternalURI(ref.TargetURI) && !seen[ref.TargetURI] {
			seen[ref.TargetURI] = true
			links = append(links, linkcheck.LinkInput{
				URI:           ref.TargetURI,
				SourceContext: formatSourceContext(ref),
			})
		}

		// Check multiple target URIs
		for _, uri := range ref.TargetURIs {
			if isExternalURI(uri) && !seen[uri] {
				seen[uri] = true
				links = append(links, linkcheck.LinkInput{
					URI:           uri,
					SourceContext: formatSourceContext(ref),
				})
			}
		}

		// Check alternative URIs
		for _, uri := range ref.AlternativeURIs {
			if isExternalURI(uri) && !seen[uri] {
				seen[uri] = true
				links = append(links, linkcheck.LinkInput{
					URI:           uri,
					SourceContext: formatSourceContext(ref),
				})
			}
		}
	}

	return links
}

// isExternalURI checks if a URI is an external HTTP(S) URL.
func isExternalURI(uri string) bool {
	return strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://")
}

// formatSourceContext creates a human-readable source context for a reference.
func formatSourceContext(ref *extract.ResolvedReference) string {
	if ref.Original == nil {
		return ""
	}

	if ref.ContextArticle > 0 {
		return fmt.Sprintf("Article %d", ref.ContextArticle)
	}

	if ref.Original.SourceArticle > 0 {
		return fmt.Sprintf("Article %d", ref.Original.SourceArticle)
	}

	return ref.Original.RawText
}

// defaultLibraryPath returns the default library location.
func defaultLibraryPath() string {
	return ".regula"
}

func libraryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "library",
		Short: "Manage the legislation library",
		Long: `Manage a persistent library of ingested legislation documents.

The library stores both plain text sources and serialized RDF graphs
on disk, enabling cross-legislation analysis without re-ingesting.

Examples:
  regula library init
  regula library seed --testdata-dir testdata
  regula library list
  regula library status
  regula library add --source testdata/gdpr.txt --id eu-gdpr --jurisdiction EU
  regula library query --template rights --documents eu-gdpr,us-ca-ccpa
  regula library source eu-gdpr
  regula library export --document eu-gdpr --format json
  regula library remove test-doc`,
	}

	cmd.AddCommand(libraryInitCmd())
	cmd.AddCommand(libraryAddCmd())
	cmd.AddCommand(librarySeedCmd())
	cmd.AddCommand(libraryListCmd())
	cmd.AddCommand(libraryStatusCmd())
	cmd.AddCommand(libraryQueryCmd())
	cmd.AddCommand(libraryRemoveCmd())
	cmd.AddCommand(libraryExportCmd())
	cmd.AddCommand(librarySourceCmd())

	return cmd
}

func libraryInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new legislation library",
		RunE: func(cmd *cobra.Command, args []string) error {
			libraryPath, _ := cmd.Flags().GetString("path")
			baseURI, _ := cmd.Flags().GetString("base-uri")

			lib, err := library.Init(libraryPath, baseURI)
			if err != nil {
				return fmt.Errorf("failed to initialize library: %w", err)
			}

			fmt.Printf("Library initialized at: %s\n", lib.Path())
			fmt.Printf("Base URI: %s\n", lib.BaseURI())
			fmt.Println("\nNext steps:")
			fmt.Println("  regula library seed --testdata-dir testdata")
			fmt.Println("  regula library add --source path/to/legislation.txt --id my-doc")
			return nil
		},
	}

	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")
	cmd.Flags().String("base-uri", "", "Base URI for the knowledge graph (default: https://regula.dev/regulations/)")

	return cmd
}

func libraryAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a legislation document to the library",
		Long: `Ingest a legislation document and store it in the library.

The document is parsed, extracted, and its RDF graph is serialized to disk.

Examples:
  regula library add --source testdata/gdpr.txt --id eu-gdpr --jurisdiction EU
  regula library add --source testdata/ccpa.txt --id us-ca-ccpa --name CCPA --jurisdiction US-CA
  regula library add --source my-law.txt --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			sourcePath, _ := cmd.Flags().GetString("source")
			documentID, _ := cmd.Flags().GetString("id")
			documentName, _ := cmd.Flags().GetString("name")
			jurisdiction, _ := cmd.Flags().GetString("jurisdiction")
			format, _ := cmd.Flags().GetString("format")
			tags, _ := cmd.Flags().GetStringSlice("tags")
			force, _ := cmd.Flags().GetBool("force")
			libraryPath, _ := cmd.Flags().GetString("path")

			if sourcePath == "" {
				return fmt.Errorf("--source flag is required")
			}

			sourceText, err := os.ReadFile(sourcePath)
			if err != nil {
				return fmt.Errorf("failed to read source: %w", err)
			}

			if documentID == "" {
				documentID = library.DeriveDocumentID(sourcePath)
			}

			lib, err := library.Open(libraryPath)
			if err != nil {
				return fmt.Errorf("library not found at %s (run 'regula library init' first): %w", libraryPath, err)
			}

			if documentName == "" {
				documentName = documentID
			}

			fmt.Printf("Adding document: %s\n", documentID)
			fmt.Printf("  Source: %s (%d bytes)\n", sourcePath, len(sourceText))

			entry, err := lib.AddDocument(documentID, sourceText, library.AddOptions{
				Name:         documentName,
				ShortName:    documentName,
				Jurisdiction: jurisdiction,
				Format:       format,
				Tags:         tags,
				Force:        force,
			})
			if err != nil {
				return fmt.Errorf("failed to add document: %w", err)
			}

			if entry.Status == library.StatusReady {
				fmt.Printf("  Status: ready\n")
				if entry.Stats != nil {
					fmt.Printf("  Triples: %d\n", entry.Stats.TotalTriples)
					fmt.Printf("  Articles: %d\n", entry.Stats.Articles)
					fmt.Printf("  Definitions: %d\n", entry.Stats.Definitions)
					fmt.Printf("  References: %d\n", entry.Stats.References)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringP("source", "s", "", "Source document path")
	cmd.Flags().String("id", "", "Document identifier (derived from filename if omitted)")
	cmd.Flags().String("name", "", "Human-readable name")
	cmd.Flags().String("jurisdiction", "", "Jurisdiction code (e.g., EU, US-CA, GB)")
	cmd.Flags().String("format", "", "Parser format hint (eu, us, uk, generic)")
	cmd.Flags().StringSlice("tags", []string{}, "Tags for categorization")
	cmd.Flags().Bool("force", false, "Overwrite existing document")
	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")

	return cmd
}

func librarySeedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the library with testdata legislation",
		Long: `Ingest all known testdata documents into the library.

Processes 18 legislation documents spanning EU, US (state and federal),
UK, Australian, and international jurisdictions.

Example:
  regula library seed --testdata-dir testdata`,
		RunE: func(cmd *cobra.Command, args []string) error {
			testdataDir, _ := cmd.Flags().GetString("testdata-dir")
			libraryPath, _ := cmd.Flags().GetString("path")

			lib, err := library.Open(libraryPath)
			if err != nil {
				// Auto-init if not exists
				lib, err = library.Init(libraryPath, "")
				if err != nil {
					return fmt.Errorf("failed to initialize library: %w", err)
				}
				fmt.Printf("Library initialized at: %s\n\n", lib.Path())
			}

			entries := library.DefaultCorpusEntries()
			fmt.Printf("Seeding library with %d documents from %s\n\n", len(entries), testdataDir)

			seedReport, err := library.SeedFromCorpus(lib, testdataDir, entries)
			if err != nil {
				return fmt.Errorf("seeding failed: %w", err)
			}

			for _, entryState := range seedReport.Entries {
				switch entryState.Status {
				case "ingested":
					entry := lib.GetDocument(entryState.ID)
					tripleCount := 0
					if entry != nil && entry.Stats != nil {
						tripleCount = entry.Stats.TotalTriples
					}
					fmt.Printf("  [OK] %-20s %d triples\n", entryState.ID, tripleCount)
				case "skipped":
					fmt.Printf("  [SKIP] %-18s already in library\n", entryState.ID)
				case "failed":
					fmt.Printf("  [FAIL] %-18s %s\n", entryState.ID, entryState.Error)
				}
			}

			fmt.Printf("\nSeed complete: %d ingested, %d skipped, %d failed\n",
				seedReport.Succeeded, seedReport.Skipped, seedReport.Failed)

			libraryStats := lib.Stats()
			fmt.Printf("\nLibrary totals: %d documents, %d triples\n",
				libraryStats.TotalDocuments, libraryStats.TotalTriples)

			return nil
		},
	}

	cmd.Flags().String("testdata-dir", "testdata", "Path to testdata directory")
	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")

	return cmd
}

func libraryListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all documents in the library",
		RunE: func(cmd *cobra.Command, args []string) error {
			libraryPath, _ := cmd.Flags().GetString("path")
			formatStr, _ := cmd.Flags().GetString("format")
			jurisdiction, _ := cmd.Flags().GetString("jurisdiction")

			lib, err := library.Open(libraryPath)
			if err != nil {
				return fmt.Errorf("library not found at %s: %w", libraryPath, err)
			}

			docs := lib.ListDocuments()

			// Filter by jurisdiction
			if jurisdiction != "" {
				filtered := make([]*library.DocumentEntry, 0)
				for _, entry := range docs {
					if entry.Jurisdiction == jurisdiction {
						filtered = append(filtered, entry)
					}
				}
				docs = filtered
			}

			if formatStr == "json" {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				return encoder.Encode(docs)
			}

			if len(docs) == 0 {
				fmt.Println("Library is empty. Run 'regula library seed' to add testdata documents.")
				return nil
			}

			fmt.Printf("%-22s %-22s %-12s %-8s %8s %8s %8s\n",
				"ID", "NAME", "JURISDICTION", "STATUS", "TRIPLES", "ARTICLES", "DEFS")
			fmt.Println(strings.Repeat("-", 100))

			for _, entry := range docs {
				tripleCount := 0
				articleCount := 0
				definitionCount := 0
				if entry.Stats != nil {
					tripleCount = entry.Stats.TotalTriples
					articleCount = entry.Stats.Articles
					definitionCount = entry.Stats.Definitions
				}
				name := entry.ShortName
				if name == "" {
					name = entry.Name
				}
				if name == "" {
					name = entry.ID
				}
				fmt.Printf("%-22s %-22s %-12s %-8s %8d %8d %8d\n",
					truncateString(entry.ID, 22),
					truncateString(name, 22),
					entry.Jurisdiction,
					entry.Status,
					tripleCount,
					articleCount,
					definitionCount,
				)
			}

			fmt.Printf("\n%d document(s)\n", len(docs))
			return nil
		},
	}

	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")
	cmd.Flags().StringP("format", "f", "table", "Output format (table, json)")
	cmd.Flags().String("jurisdiction", "", "Filter by jurisdiction")

	return cmd
}

func libraryStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show library statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			libraryPath, _ := cmd.Flags().GetString("path")

			lib, err := library.Open(libraryPath)
			if err != nil {
				return fmt.Errorf("library not found at %s: %w", libraryPath, err)
			}

			libraryStats := lib.Stats()

			fmt.Printf("Library: %s\n", lib.Path())
			fmt.Printf("Base URI: %s\n\n", lib.BaseURI())
			fmt.Printf("Documents:    %d\n", libraryStats.TotalDocuments)
			fmt.Printf("Total triples: %d\n", libraryStats.TotalTriples)
			fmt.Printf("Total articles: %d\n", libraryStats.TotalArticles)
			fmt.Printf("Total definitions: %d\n", libraryStats.TotalDefinitions)
			fmt.Printf("Total references: %d\n", libraryStats.TotalReferences)
			fmt.Printf("Total rights: %d\n", libraryStats.TotalRights)
			fmt.Printf("Total obligations: %d\n", libraryStats.TotalObligations)

			if len(libraryStats.ByJurisdiction) > 0 {
				fmt.Println("\nBy Jurisdiction:")
				for jurisdictionKey, jurisdictionCount := range libraryStats.ByJurisdiction {
					fmt.Printf("  %-15s %d\n", jurisdictionKey, jurisdictionCount)
				}
			}

			if len(libraryStats.ByStatus) > 0 {
				fmt.Println("\nBy Status:")
				for statusKey, statusCount := range libraryStats.ByStatus {
					fmt.Printf("  %-15s %d\n", statusKey, statusCount)
				}
			}

			return nil
		},
	}

	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")

	return cmd
}

func libraryQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query [sparql-query]",
		Short: "Query across library documents",
		Long: `Execute a SPARQL query against one or more library documents.

By default queries all documents. Use --documents to specify a subset.

Examples:
  regula library query --template definitions
  regula library query --template rights --documents eu-gdpr,us-ca-ccpa
  regula library query "SELECT ?article ?title WHERE { ?article rdf:type reg:Article . ?article reg:title ?title } LIMIT 10"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			libraryPath, _ := cmd.Flags().GetString("path")
			templateName, _ := cmd.Flags().GetString("template")
			formatStr, _ := cmd.Flags().GetString("format")
			documentIDs, _ := cmd.Flags().GetStringSlice("documents")
			showTiming, _ := cmd.Flags().GetBool("timing")
			limit, _ := cmd.Flags().GetInt("limit")

			lib, err := library.Open(libraryPath)
			if err != nil {
				return fmt.Errorf("library not found at %s: %w", libraryPath, err)
			}

			// Determine query string
			var queryStr string
			if templateName != "" {
				tmpl, ok := queryTemplates[templateName]
				if !ok {
					return fmt.Errorf("unknown template: %s\nUse 'regula query --list-templates' to see available templates", templateName)
				}
				queryStr = tmpl.Query
				if !showTiming {
					fmt.Printf("Template: %s\n", templateName)
					fmt.Printf("Description: %s\n\n", tmpl.Description)
				}
			} else if len(args) > 0 {
				queryStr = args[0]
			} else {
				return fmt.Errorf("provide a query or use --template")
			}

			// Add LIMIT if specified and not already in query
			if limit > 0 && !strings.Contains(strings.ToUpper(queryStr), "LIMIT") {
				queryStr += fmt.Sprintf(" LIMIT %d", limit)
			}

			// Load triple stores
			var mergedStore *store.TripleStore
			if len(documentIDs) > 0 {
				mergedStore, err = lib.LoadMergedTripleStore(documentIDs...)
			} else {
				mergedStore, err = lib.LoadAllTripleStores()
			}
			if err != nil {
				return fmt.Errorf("failed to load triple stores: %w", err)
			}

			// Parse the SPARQL query
			parsedQuery, parseErr := query.ParseQuery(queryStr)
			if parseErr != nil {
				return fmt.Errorf("query parse error: %w", parseErr)
			}

			queryExecutor := query.NewExecutor(mergedStore)

			startTime := time.Now()
			result, queryErr := queryExecutor.Execute(parsedQuery)
			elapsed := time.Since(startTime)

			if queryErr != nil {
				return fmt.Errorf("query failed: %w", queryErr)
			}

			if showTiming {
				fmt.Printf("Query executed in %v (%d results, %d triples searched)\n",
					elapsed, result.Count, mergedStore.Count())
			}

			// Format output
			outputFormat := query.OutputFormat(formatStr)
			output, fmtErr := result.Format(outputFormat)
			if fmtErr != nil {
				return fmt.Errorf("format error: %w", fmtErr)
			}
			fmt.Print(output)

			return nil
		},
	}

	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")
	cmd.Flags().String("template", "", "Use a built-in query template")
	cmd.Flags().StringP("format", "f", "table", "Output format (table, json, csv)")
	cmd.Flags().StringSlice("documents", []string{}, "Document IDs to query (comma-separated, default: all)")
	cmd.Flags().Bool("timing", false, "Show query execution time")
	cmd.Flags().Int("limit", 0, "Limit number of results")

	return cmd
}

func libraryRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <document-id>",
		Short: "Remove a document from the library",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			libraryPath, _ := cmd.Flags().GetString("path")
			documentID := args[0]

			lib, err := library.Open(libraryPath)
			if err != nil {
				return fmt.Errorf("library not found at %s: %w", libraryPath, err)
			}

			if err := lib.RemoveDocument(documentID); err != nil {
				return fmt.Errorf("failed to remove document: %w", err)
			}

			fmt.Printf("Removed document: %s\n", documentID)
			return nil
		},
	}

	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")

	return cmd
}

func libraryExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export a document's RDF graph",
		Long: `Export a document's serialized RDF graph in various formats.

Examples:
  regula library export --document eu-gdpr --format json
  regula library export --document eu-gdpr --format summary`,
		RunE: func(cmd *cobra.Command, args []string) error {
			libraryPath, _ := cmd.Flags().GetString("path")
			documentID, _ := cmd.Flags().GetString("document")
			formatStr, _ := cmd.Flags().GetString("format")
			outputPath, _ := cmd.Flags().GetString("output")

			if documentID == "" {
				return fmt.Errorf("--document flag is required")
			}

			lib, err := library.Open(libraryPath)
			if err != nil {
				return fmt.Errorf("library not found at %s: %w", libraryPath, err)
			}

			tripleStore, err := lib.LoadTripleStore(documentID)
			if err != nil {
				return fmt.Errorf("failed to load document: %w", err)
			}

			var output string

			switch formatStr {
			case "json":
				data, marshalErr := library.SerializeTripleStore(tripleStore)
				if marshalErr != nil {
					return fmt.Errorf("failed to serialize: %w", marshalErr)
				}
				output = string(data)
			case "summary":
				exportStats := tripleStore.Stats()
				output = fmt.Sprintf("Document: %s\n", documentID)
				output += fmt.Sprintf("Total triples: %d\n", exportStats.TotalTriples)
				output += fmt.Sprintf("Unique subjects: %d\n", exportStats.UniqueSubjects)
				output += fmt.Sprintf("Unique predicates: %d\n", exportStats.UniquePredicates)
				output += fmt.Sprintf("Unique objects: %d\n", exportStats.UniqueObjects)
				if len(exportStats.PredicateCounts) > 0 {
					output += "\nPredicate Counts:\n"
					for predicateKey, predicateCount := range exportStats.PredicateCounts {
						output += fmt.Sprintf("  %-40s %d\n", predicateKey, predicateCount)
					}
				}
			default:
				// N-Triples format
				allTriples := tripleStore.All()
				var tripleLines []string
				for _, triple := range allTriples {
					tripleLines = append(tripleLines, triple.NTriples())
				}
				output = strings.Join(tripleLines, "\n")
			}

			if outputPath != "" {
				if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
					return fmt.Errorf("failed to write output: %w", err)
				}
				fmt.Printf("Exported %s to %s\n", documentID, outputPath)
			} else {
				fmt.Print(output)
			}

			return nil
		},
	}

	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")
	cmd.Flags().String("document", "", "Document ID to export")
	cmd.Flags().StringP("format", "f", "ntriples", "Output format (json, summary, ntriples)")
	cmd.Flags().StringP("output", "o", "", "Output file path")

	return cmd
}

func librarySourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source <document-id>",
		Short: "Display the original source text of a document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			libraryPath, _ := cmd.Flags().GetString("path")
			documentID := args[0]

			lib, err := library.Open(libraryPath)
			if err != nil {
				return fmt.Errorf("library not found at %s: %w", libraryPath, err)
			}

			sourceText, err := lib.LoadSourceText(documentID)
			if err != nil {
				return fmt.Errorf("failed to load source: %w", err)
			}

			fmt.Print(string(sourceText))
			return nil
		},
	}

	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")

	return cmd
}

func truncateString(inputStr string, maxLength int) string {
	if len(inputStr) <= maxLength {
		return inputStr
	}
	return inputStr[:maxLength-3] + "..."
}

// crawlCmd creates the crawl command for legislation discovery.
func crawlCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "crawl",
		Short: "Crawl and discover legislation by following cross-references",
		Long: `Performs a BFS tree-walking crawl starting from a seed document, citation,
or URL. The crawler follows cross-references in ingested legislation to discover
and ingest related documents from US law sources (USC, CFR, state codes).

Each discovered document is ingested into the library, its cross-references are
extracted, and newly discovered citations are enqueued for further crawling.

The crawl stops when it reaches the configured depth limit, document limit,
or exhausts all discoverable references.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			seedDocID, _ := cmd.Flags().GetString("seed")
			citationStr, _ := cmd.Flags().GetString("citation")
			seedURL, _ := cmd.Flags().GetString("url")
			maxDepth, _ := cmd.Flags().GetInt("max-depth")
			maxDocuments, _ := cmd.Flags().GetInt("max-documents")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			resumeCrawl, _ := cmd.Flags().GetBool("resume")
			allowedDomainsStr, _ := cmd.Flags().GetString("allowed-domains")
			rateLimitStr, _ := cmd.Flags().GetString("rate-limit")
			outputFormat, _ := cmd.Flags().GetString("format")
			libraryPath, _ := cmd.Flags().GetString("path")

			if seedDocID == "" && citationStr == "" && seedURL == "" && !resumeCrawl {
				return fmt.Errorf("specify at least one of --seed, --citation, --url, or --resume")
			}

			// Parse rate limit
			rateLimit := crawler.DefaultCrawlRateLimit
			if rateLimitStr != "" {
				parsedDuration, err := time.ParseDuration(rateLimitStr)
				if err != nil {
					return fmt.Errorf("invalid rate limit %q: %w", rateLimitStr, err)
				}
				rateLimit = parsedDuration
			}

			// Parse allowed domains
			var allowedDomains []string
			if allowedDomainsStr != "" {
				allowedDomains = strings.Split(allowedDomainsStr, ",")
				for i := range allowedDomains {
					allowedDomains[i] = strings.TrimSpace(allowedDomains[i])
				}
			}

			crawlConfig := crawler.CrawlConfig{
				MaxDepth:       maxDepth,
				MaxDocuments:   maxDocuments,
				AllowedDomains: allowedDomains,
				RateLimit:      rateLimit,
				Timeout:        crawler.DefaultCrawlTimeout,
				LibraryPath:    libraryPath,
				BaseURI:        "https://regula.dev/regulations/",
				DryRun:         dryRun,
				Resume:         resumeCrawl,
				UserAgent:      crawler.DefaultCrawlUserAgent,
				DomainConfigs:  crawler.DefaultDomainConfigs(),
				OutputFormat:   outputFormat,
			}

			crawlerInstance, err := crawler.NewCrawler(crawlConfig)
			if err != nil {
				return fmt.Errorf("failed to initialize crawler: %w", err)
			}

			// Handle resume
			if resumeCrawl {
				statePath := filepath.Join(libraryPath, "crawl-state.json")
				fmt.Fprintf(os.Stderr, "Resuming crawl from %s...\n", statePath)
				crawlReport, err := crawlerInstance.Resume(statePath)
				if err != nil {
					return fmt.Errorf("failed to resume crawl: %w", err)
				}
				fmt.Print(crawlReport.Format(outputFormat))
				return nil
			}

			// Build seeds
			var seeds []crawler.CrawlSeed
			if seedDocID != "" {
				seeds = append(seeds, crawler.CrawlSeed{
					Type:  crawler.SeedTypeDocumentID,
					Value: seedDocID,
				})
			}
			if citationStr != "" {
				seeds = append(seeds, crawler.CrawlSeed{
					Type:  crawler.SeedTypeCitation,
					Value: citationStr,
				})
			}
			if seedURL != "" {
				seeds = append(seeds, crawler.CrawlSeed{
					Type:  crawler.SeedTypeURL,
					Value: seedURL,
				})
			}

			if dryRun {
				fmt.Fprintf(os.Stderr, "Planning crawl (dry run) with %d seed(s), max depth %d, max documents %d...\n",
					len(seeds), maxDepth, maxDocuments)
			} else {
				fmt.Fprintf(os.Stderr, "Starting crawl with %d seed(s), max depth %d, max documents %d...\n",
					len(seeds), maxDepth, maxDocuments)
			}

			crawlReport, err := crawlerInstance.Crawl(seeds)
			if err != nil {
				return fmt.Errorf("crawl failed: %w", err)
			}

			fmt.Print(crawlReport.Format(outputFormat))
			return nil
		},
	}

	cmd.Flags().String("seed", "", "Seed from an existing library document ID")
	cmd.Flags().String("citation", "", "Seed from a US law citation (e.g., '42 U.S.C. § 1320d')")
	cmd.Flags().String("url", "", "Seed from a direct URL")
	cmd.Flags().Int("max-depth", crawler.DefaultCrawlMaxDepth, "Maximum BFS depth for following references")
	cmd.Flags().Int("max-documents", crawler.DefaultCrawlMaxDocuments, "Maximum number of documents to ingest")
	cmd.Flags().Bool("dry-run", false, "Plan the crawl without making network requests")
	cmd.Flags().Bool("resume", false, "Resume a previously interrupted crawl")
	cmd.Flags().String("allowed-domains", "", "Comma-separated list of allowed domains")
	cmd.Flags().String("rate-limit", "3s", "Minimum interval between requests per domain")
	cmd.Flags().String("format", "table", "Output format (table, json)")
	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")

	return cmd
}

func playgroundCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "playground",
		Short: "USC triple store analysis playground",
		Long: `Analysis playground with pre-built query templates for exploring
ingested legislation data in the library.

Provides 10+ analysis query templates runnable via CLI, plus custom
SPARQL query support with CSV/JSON export and timing/pagination.

Commands:
  list     List available analysis query templates
  run      Run a template by name
  query    Run a custom SPARQL query

Examples:
  regula playground list
  regula playground run top-chapters-by-sections
  regula playground run cross-ref-density --title 42
  regula playground run definition-coverage --export json
  regula playground run rights-enumeration --limit 50 --offset 10
  regula playground query "SELECT ?s ?p ?o WHERE { ?s ?p ?o } LIMIT 10"`,
	}

	cmd.AddCommand(playgroundListCmd())
	cmd.AddCommand(playgroundRunCmd())
	cmd.AddCommand(playgroundQueryCmd())

	return cmd
}

func playgroundListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available analysis query templates",
		Long: `List all pre-built analysis query templates with their categories
and supported parameters.

Examples:
  regula playground list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			templateNames := playground.TemplateNames()

			fmt.Println("Available playground analysis templates:")
			fmt.Println()

			for _, templateName := range templateNames {
				template, _ := playground.Get(templateName)
				fmt.Printf("  %-28s [%-15s] %s\n", templateName, template.Category, template.Description)
				for _, parameter := range template.Parameters {
					requiredLabel := "optional"
					if parameter.Required {
						requiredLabel = "required"
					}
					fmt.Printf("    --%s (%s): %s\n", parameter.Name, requiredLabel, parameter.Description)
				}
			}

			fmt.Println()
			fmt.Println("Usage:")
			fmt.Println("  regula playground run <template-name>")
			fmt.Println("  regula playground run <template-name> --title 42")
			fmt.Println("  regula playground run <template-name> --export csv")
			return nil
		},
	}
}

func playgroundRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <template-name>",
		Short: "Run a playground analysis template",
		Long: `Run a pre-built analysis query template against the library.

Templates are parameterizable (e.g., --title 42) and support export
to table, JSON, or CSV formats.

Examples:
  regula playground run top-chapters-by-sections
  regula playground run cross-ref-density --title 42
  regula playground run definition-coverage --export json
  regula playground run rights-enumeration --limit 50 --offset 10
  regula playground run chapter-structure --title 42 --export csv > structure.csv`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			templateName := args[0]
			titleFilter, _ := cmd.Flags().GetString("title")
			exportFormat, _ := cmd.Flags().GetString("export")
			limitValue, _ := cmd.Flags().GetInt("limit")
			offsetValue, _ := cmd.Flags().GetInt("offset")
			showTiming, _ := cmd.Flags().GetBool("timing")
			libraryPath, _ := cmd.Flags().GetString("path")
			documentIDs, _ := cmd.Flags().GetStringSlice("documents")

			// Look up template
			template, exists := playground.Get(templateName)
			if !exists {
				return fmt.Errorf("unknown template: %s\nUse 'regula playground list' to see available templates", templateName)
			}

			// Build parameter map
			parameterValues := make(map[string]string)
			if titleFilter != "" {
				parameterValues["title"] = titleFilter
			}

			// Render query with parameters
			renderedQuery, renderErr := playground.RenderQuery(template, parameterValues)
			if renderErr != nil {
				return fmt.Errorf("template parameter error: %w", renderErr)
			}

			// Append LIMIT/OFFSET if not already in query
			if limitValue > 0 && !strings.Contains(strings.ToUpper(renderedQuery), "LIMIT") {
				renderedQuery += fmt.Sprintf(" LIMIT %d", limitValue)
			}
			if offsetValue > 0 && !strings.Contains(strings.ToUpper(renderedQuery), "OFFSET") {
				renderedQuery += fmt.Sprintf(" OFFSET %d", offsetValue)
			}

			// Open library
			lib, err := library.Open(libraryPath)
			if err != nil {
				return fmt.Errorf("library not found at %s: %w", libraryPath, err)
			}

			// Load triple stores
			var mergedStore *store.TripleStore
			if len(documentIDs) > 0 {
				mergedStore, err = lib.LoadMergedTripleStore(documentIDs...)
			} else {
				mergedStore, err = lib.LoadAllTripleStores()
			}
			if err != nil {
				return fmt.Errorf("failed to load triple stores: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Template: %s\n", template.Name)
			fmt.Fprintf(os.Stderr, "Description: %s\n", template.Description)
			if titleFilter != "" {
				fmt.Fprintf(os.Stderr, "Title filter: %s\n", titleFilter)
			}
			fmt.Fprintln(os.Stderr)

			// Execute query
			return executePlaygroundQuery(mergedStore, renderedQuery, exportFormat, showTiming)
		},
	}

	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")
	cmd.Flags().StringSlice("documents", []string{}, "Document IDs to query (comma-separated, default: all)")
	cmd.Flags().String("title", "", "Title number filter for templates that support it")
	cmd.Flags().String("export", "table", "Output format (table, json, csv)")
	cmd.Flags().Int("limit", 0, "Limit number of results")
	cmd.Flags().Int("offset", 0, "Skip first N results")
	cmd.Flags().Bool("timing", false, "Show query execution time")

	return cmd
}

func playgroundQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query [sparql-query]",
		Short: "Run a custom SPARQL query against the library",
		Long: `Run an arbitrary SPARQL query against all ingested library documents.

Supports SELECT, CONSTRUCT, and DESCRIBE queries.

Examples:
  regula playground query "SELECT ?article ?title WHERE { ?article rdf:type reg:Article . ?article reg:title ?title } LIMIT 10"
  regula playground query --export json "SELECT ?s ?p ?o WHERE { ?s ?p ?o } LIMIT 5"
  regula playground query --export csv "SELECT ?term WHERE { ?term rdf:type reg:DefinedTerm }" > terms.csv`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			queryStr := args[0]
			exportFormat, _ := cmd.Flags().GetString("export")
			limitValue, _ := cmd.Flags().GetInt("limit")
			offsetValue, _ := cmd.Flags().GetInt("offset")
			showTiming, _ := cmd.Flags().GetBool("timing")
			libraryPath, _ := cmd.Flags().GetString("path")
			documentIDs, _ := cmd.Flags().GetStringSlice("documents")

			// Append LIMIT/OFFSET if not already in query
			if limitValue > 0 && !strings.Contains(strings.ToUpper(queryStr), "LIMIT") {
				queryStr += fmt.Sprintf(" LIMIT %d", limitValue)
			}
			if offsetValue > 0 && !strings.Contains(strings.ToUpper(queryStr), "OFFSET") {
				queryStr += fmt.Sprintf(" OFFSET %d", offsetValue)
			}

			// Open library
			lib, err := library.Open(libraryPath)
			if err != nil {
				return fmt.Errorf("library not found at %s: %w", libraryPath, err)
			}

			// Load triple stores
			var mergedStore *store.TripleStore
			if len(documentIDs) > 0 {
				mergedStore, err = lib.LoadMergedTripleStore(documentIDs...)
			} else {
				mergedStore, err = lib.LoadAllTripleStores()
			}
			if err != nil {
				return fmt.Errorf("failed to load triple stores: %w", err)
			}

			return executePlaygroundQuery(mergedStore, queryStr, exportFormat, showTiming)
		},
	}

	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")
	cmd.Flags().StringSlice("documents", []string{}, "Document IDs to query (comma-separated, default: all)")
	cmd.Flags().String("export", "table", "Output format (table, json, csv)")
	cmd.Flags().Int("limit", 0, "Limit number of results")
	cmd.Flags().Int("offset", 0, "Skip first N results")
	cmd.Flags().Bool("timing", false, "Show query execution time")

	return cmd
}

// executePlaygroundQuery parses, executes, and formats a SPARQL query against the given store.
func executePlaygroundQuery(tripleStore *store.TripleStore, queryStr string, exportFormat string, showTiming bool) error {
	parsedQuery, parseErr := query.ParseQuery(queryStr)
	if parseErr != nil {
		return fmt.Errorf("query parse error: %w", parseErr)
	}

	queryExecutor := query.NewExecutor(tripleStore)
	startTime := time.Now()

	// Route by query type
	switch parsedQuery.Type {
	case query.ConstructQueryType:
		result, err := queryExecutor.ExecuteConstruct(parsedQuery)
		elapsed := time.Since(startTime)
		if err != nil {
			return fmt.Errorf("CONSTRUCT query error: %w", err)
		}

		outputFormat := query.OutputFormat(exportFormat)
		if outputFormat == query.FormatTable || outputFormat == query.FormatCSV {
			outputFormat = query.FormatTurtle
		}
		output, fmtErr := result.Format(outputFormat)
		if fmtErr != nil {
			return fmt.Errorf("format error: %w", fmtErr)
		}
		fmt.Print(output)
		fmt.Fprintf(os.Stderr, "\n%d triples returned", result.Count)
		if showTiming {
			fmt.Fprintf(os.Stderr, " (%v)", elapsed)
		}
		fmt.Fprintln(os.Stderr)

	case query.DescribeQueryType:
		result, err := queryExecutor.ExecuteDescribe(parsedQuery)
		elapsed := time.Since(startTime)
		if err != nil {
			return fmt.Errorf("DESCRIBE query error: %w", err)
		}

		outputFormat := query.OutputFormat(exportFormat)
		if outputFormat == query.FormatTable || outputFormat == query.FormatCSV {
			outputFormat = query.FormatTurtle
		}
		output, fmtErr := result.Format(outputFormat)
		if fmtErr != nil {
			return fmt.Errorf("format error: %w", fmtErr)
		}
		fmt.Print(output)
		fmt.Fprintf(os.Stderr, "\n%d triples returned", result.Count)
		if showTiming {
			fmt.Fprintf(os.Stderr, " (%v)", elapsed)
		}
		fmt.Fprintln(os.Stderr)

	default:
		// SELECT query (default)
		result, err := queryExecutor.Execute(parsedQuery)
		elapsed := time.Since(startTime)
		if err != nil {
			return fmt.Errorf("query error: %w", err)
		}

		outputFormat := query.OutputFormat(exportFormat)
		output, fmtErr := result.Format(outputFormat)
		if fmtErr != nil {
			return fmt.Errorf("format error: %w", fmtErr)
		}
		fmt.Print(output)
		fmt.Fprintf(os.Stderr, "\n%d rows returned", result.Count)
		if showTiming {
			fmt.Fprintf(os.Stderr, " (%v)", elapsed)
			fmt.Fprintf(os.Stderr, "\n  Parse:   %v", result.Metrics.ParseTime)
			fmt.Fprintf(os.Stderr, "\n  Plan:    %v", result.Metrics.PlanTime)
			fmt.Fprintf(os.Stderr, "\n  Execute: %v", result.Metrics.ExecuteTime)
			fmt.Fprintf(os.Stderr, "\n  Triples: %d searched", tripleStore.Count())
		}
		fmt.Fprintln(os.Stderr)
	}

	return nil
}

func bulkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bulk",
		Short: "Bulk download and ingest legislation from official sources",
		Long: `Download and ingest legislation data in bulk from 5 official sources:

  uscode        US Code XML from uscode.house.gov (54 titles)
  cfr           Code of Federal Regulations from govinfo.gov (50 titles)
  california    California codes from leginfo.legislature.ca.gov (30 codes)
  archive       State code archives from Internet Archive govlaw collection
  parliamentary Congressional rules: House Rules, Senate Rules, Joint Rules

Workflow:
  1. regula bulk list <source>          List available datasets
  2. regula bulk download <source>      Download archives to .regula/downloads/
  3. regula bulk ingest --source <src>  Parse downloaded files and add to library
  4. regula bulk status                 Check download/ingest progress
  5. regula bulk stats                  Show comprehensive ingestion statistics`,
	}

	cmd.AddCommand(bulkListCmd())
	cmd.AddCommand(bulkDownloadCmd())
	cmd.AddCommand(bulkIngestCmd())
	cmd.AddCommand(bulkStatusCmd())
	cmd.AddCommand(bulkStatsCmd())

	return cmd
}

func bulkListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <source>",
		Short: "List available datasets from a bulk source",
		Long: `List all available datasets from a bulk legislation source.

Sources: uscode, cfr, california, archive, parliamentary

Examples:
  regula bulk list uscode         List all 54 US Code titles
  regula bulk list cfr            List all 50 CFR titles
  regula bulk list california     List all 30 California codes
  regula bulk list archive        List Internet Archive govlaw items
  regula bulk list parliamentary  List House Rules, Senate Rules, Joint Rules`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sourceName := args[0]
			yearFlag, _ := cmd.Flags().GetString("year")

			downloadConfig := bulk.DefaultDownloadConfig()
			if yearFlag != "" {
				downloadConfig.CFRYear = yearFlag
			}

			source, err := bulk.ResolveSource(sourceName, downloadConfig)
			if err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "Listing datasets from %s: %s\n\n", source.Name(), source.Description())

			datasets, err := source.ListDatasets()
			if err != nil {
				return fmt.Errorf("failed to list datasets: %w", err)
			}

			fmt.Print(bulk.FormatDatasetTable(datasets))
			return nil
		},
	}

	cmd.Flags().String("year", "", "CFR edition year (default: 2024)")

	return cmd
}

func bulkDownloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download <source>",
		Short: "Download legislation archives from a bulk source",
		Long: `Download legislation data from a bulk source to .regula/downloads/.

Files are downloaded with resume support: existing files are skipped.
A manifest.json tracks all completed downloads.

Sources: uscode, cfr, california, archive, parliamentary

Examples:
  regula bulk download uscode                     Download all 54 USC title ZIPs
  regula bulk download uscode --titles 42,26      Download specific titles
  regula bulk download cfr --year 2024            Download all CFR for 2024
  regula bulk download california --titles CIV,PEN Download specific CA codes
  regula bulk download parliamentary              Download all congressional rules
  regula bulk download uscode --dry-run           Show what would be downloaded`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sourceName := args[0]
			titlesFlag, _ := cmd.Flags().GetString("titles")
			yearFlag, _ := cmd.Flags().GetString("year")
			rateLimitFlag, _ := cmd.Flags().GetString("rate-limit")
			dryRunFlag, _ := cmd.Flags().GetBool("dry-run")
			libraryPath, _ := cmd.Flags().GetString("path")

			downloadConfig := bulk.DefaultDownloadConfig()
			downloadConfig.DownloadDirectory = filepath.Join(libraryPath, "downloads")
			downloadConfig.DryRun = dryRunFlag

			if yearFlag != "" {
				downloadConfig.CFRYear = yearFlag
			}
			if rateLimitFlag != "" {
				parsedDuration, err := time.ParseDuration(rateLimitFlag)
				if err != nil {
					return fmt.Errorf("invalid rate limit %q: %w", rateLimitFlag, err)
				}
				downloadConfig.RateLimit = parsedDuration
			}
			if titlesFlag != "" {
				downloadConfig.TitleFilter = strings.Split(titlesFlag, ",")
			}

			source, err := bulk.ResolveSource(sourceName, downloadConfig)
			if err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "Source: %s — %s\n", source.Name(), source.Description())

			datasets, err := source.ListDatasets()
			if err != nil {
				return fmt.Errorf("failed to list datasets: %w", err)
			}

			// Apply title filter
			if len(downloadConfig.TitleFilter) > 0 {
				var filteredDatasets []bulk.Dataset
				for _, dataset := range datasets {
					for _, filterTitle := range downloadConfig.TitleFilter {
						if strings.Contains(
							strings.ToLower(dataset.Identifier),
							strings.ToLower(strings.TrimSpace(filterTitle))) {
							filteredDatasets = append(filteredDatasets, dataset)
							break
						}
					}
				}
				datasets = filteredDatasets
			}

			if len(datasets) == 0 {
				fmt.Fprintln(os.Stderr, "No datasets match the filter.")
				return nil
			}

			if dryRunFlag {
				fmt.Fprintf(os.Stderr, "\nDry run: would download %d datasets\n\n", len(datasets))
				fmt.Print(bulk.FormatDatasetTable(datasets))
				return nil
			}

			downloader, err := bulk.NewDownloader(downloadConfig)
			if err != nil {
				return fmt.Errorf("failed to initialize downloader: %w", err)
			}

			fmt.Fprintf(os.Stderr, "\nDownloading %d datasets to %s\n\n", len(datasets), downloadConfig.DownloadDirectory)

			var downloadedCount, skippedCount, failedCount int
			for datasetIndex, dataset := range datasets {
				fmt.Fprintf(os.Stderr, "[%d/%d] %s\n", datasetIndex+1, len(datasets), dataset.DisplayName)

				result, err := source.DownloadDataset(dataset, downloader)
				if err != nil {
					fmt.Fprintf(os.Stderr, "  ERROR: %v\n", err)
					failedCount++
					continue
				}
				if result.Skipped {
					fmt.Fprintf(os.Stderr, "  Skipped (already downloaded: %s)\n", bulk.FormatBytes(result.BytesWritten))
					skippedCount++
				} else {
					fmt.Fprintf(os.Stderr, "  Downloaded: %s\n", bulk.FormatBytes(result.BytesWritten))
					downloadedCount++
				}
			}

			fmt.Fprintf(os.Stderr, "\nDone: %d downloaded, %d skipped, %d failed (of %d total)\n",
				downloadedCount, skippedCount, failedCount, len(datasets))
			return nil
		},
	}

	cmd.Flags().String("titles", "", "Comma-separated title/code filter (e.g., '42,26' or 'CIV,PEN')")
	cmd.Flags().String("year", "", "CFR edition year (default: 2024)")
	cmd.Flags().String("rate-limit", "", "Minimum interval between requests per domain (default: 3s)")
	cmd.Flags().Bool("dry-run", false, "Show what would be downloaded without fetching")
	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")

	return cmd
}

func bulkIngestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Ingest downloaded bulk data into the library",
		Long: `Parse downloaded bulk legislation files and add them to the library.

Downloads must be completed first via 'regula bulk download'.
Each downloaded file is parsed (XML, text, or archive) and ingested
as a library document with extracted RDF triples.

Examples:
  regula bulk ingest --source uscode              Ingest downloaded USC files
  regula bulk ingest --source cfr                 Ingest downloaded CFR files
  regula bulk ingest --source california          Ingest California codes
  regula bulk ingest --all                        Ingest all downloaded sources
  regula bulk ingest --source uscode --titles 42  Ingest specific title
  regula bulk ingest --dry-run --all              Show what would be ingested
  regula bulk ingest --force --source uscode      Re-ingest even if already in library`,
		RunE: func(cmd *cobra.Command, args []string) error {
			sourceFilter, _ := cmd.Flags().GetString("source")
			allSources, _ := cmd.Flags().GetBool("all")
			titlesFlag, _ := cmd.Flags().GetString("titles")
			forceFlag, _ := cmd.Flags().GetBool("force")
			dryRunFlag, _ := cmd.Flags().GetBool("dry-run")
			formatFlag, _ := cmd.Flags().GetString("format")
			libraryPath, _ := cmd.Flags().GetString("path")

			if sourceFilter == "" && !allSources {
				return fmt.Errorf("specify --source <name> or --all")
			}

			downloadDirectory := filepath.Join(libraryPath, "downloads")

			ingestConfig := bulk.IngestConfig{
				LibraryPath:       libraryPath,
				DownloadDirectory: downloadDirectory,
				SourceFilter:      sourceFilter,
				Force:             forceFlag,
				DryRun:            dryRunFlag,
				BaseURI:           "https://regula.dev/regulations/",
			}
			if titlesFlag != "" {
				ingestConfig.TitleFilter = strings.Split(titlesFlag, ",")
			}

			// Open or initialize library
			lib, err := library.Open(libraryPath)
			if err != nil {
				lib, err = library.Init(libraryPath, ingestConfig.BaseURI)
				if err != nil {
					return fmt.Errorf("failed to open library at %s: %w", libraryPath, err)
				}
			}

			ingester := bulk.NewBulkIngester(ingestConfig, lib)

			var report *bulk.IngestReport

			if allSources {
				fmt.Fprintf(os.Stderr, "Ingesting all downloaded sources from %s\n", downloadDirectory)
				report, err = ingester.IngestAll(downloadDirectory)
			} else {
				fmt.Fprintf(os.Stderr, "Ingesting source %q from %s\n", sourceFilter, downloadDirectory)
				report, err = ingester.IngestSource(sourceFilter, downloadDirectory)
			}

			if err != nil {
				return fmt.Errorf("ingest failed: %w", err)
			}

			switch formatFlag {
			case "json":
				fmt.Print(bulk.FormatIngestReportJSON(report))
			default:
				fmt.Print(bulk.FormatIngestReport(report))
			}

			return nil
		},
	}

	cmd.Flags().String("source", "", "Source to ingest (uscode, cfr, california, archive)")
	cmd.Flags().Bool("all", false, "Ingest all downloaded sources")
	cmd.Flags().String("titles", "", "Comma-separated title filter (e.g., '42,26')")
	cmd.Flags().Bool("force", false, "Re-ingest documents even if already in library")
	cmd.Flags().Bool("dry-run", false, "Show what would be ingested without adding to library")
	cmd.Flags().String("format", "table", "Output format (table, json)")
	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")

	return cmd
}

func bulkStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show bulk download and ingest status",
		Long: `Display the current state of bulk downloads and ingestion.

Shows per-source download counts, file sizes, ingest status, and statistics.

Examples:
  regula bulk status                  Show all download/ingest status
  regula bulk status --source uscode  Show status for USC only`,
		RunE: func(cmd *cobra.Command, args []string) error {
			sourceFilter, _ := cmd.Flags().GetString("source")
			libraryPath, _ := cmd.Flags().GetString("path")

			downloadDirectory := filepath.Join(libraryPath, "downloads")
			manifestPath := filepath.Join(downloadDirectory, "manifest.json")

			manifest, err := bulk.LoadManifest(manifestPath)
			if err != nil {
				return fmt.Errorf("failed to load download manifest: %w", err)
			}

			// Load library document stats if available
			var documentStats map[string]*bulk.DocumentStatsSummary
			lib, libErr := library.Open(libraryPath)
			if libErr == nil {
				documentStats = make(map[string]*bulk.DocumentStatsSummary)
				for _, doc := range lib.ListDocuments() {
					summary := &bulk.DocumentStatsSummary{
						Status: string(doc.Status),
					}
					if doc.Stats != nil {
						summary.Triples = doc.Stats.TotalTriples
						summary.Articles = doc.Stats.Articles
						summary.Chapters = doc.Stats.Chapters
					}
					documentStats[doc.ID] = summary
				}
			}

			fmt.Print(bulk.FormatStatusReport(manifest, sourceFilter, documentStats))
			return nil
		},
	}

	cmd.Flags().String("source", "", "Filter status to a specific source")
	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")

	return cmd
}

func bulkStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show comprehensive statistics for ingested bulk data",
		Long: `Display a detailed breakdown of ingested bulk legislation data.

Shows per-title metrics (triples, articles, chapters, definitions,
cross-references, rights, obligations), aggregate totals, and
titles ingested vs. total.

Supports table, JSON, and CSV output formats.

Examples:
  regula bulk stats                          Show stats as ASCII table
  regula bulk stats --format json            Output as JSON
  regula bulk stats --format csv             Output as CSV
  regula bulk stats --source uscode          Filter to USC only`,
		RunE: func(cmd *cobra.Command, args []string) error {
			sourceFilter, _ := cmd.Flags().GetString("source")
			formatFlag, _ := cmd.Flags().GetString("format")
			libraryPath, _ := cmd.Flags().GetString("path")

			downloadDirectory := filepath.Join(libraryPath, "downloads")
			manifestPath := filepath.Join(downloadDirectory, "manifest.json")

			manifest, err := bulk.LoadManifest(manifestPath)
			if err != nil {
				return fmt.Errorf("failed to load download manifest: %w", err)
			}

			// Apply source filter to manifest if specified
			if sourceFilter != "" {
				filteredManifest := bulk.NewDownloadManifest()
				for _, record := range manifest.Downloads {
					if record.SourceName == sourceFilter {
						filteredManifest.RecordDownload(record)
					}
				}
				manifest = filteredManifest
			}

			// Load library document stats
			var documentStats map[string]*bulk.DocumentStatsSummary
			lib, libErr := library.Open(libraryPath)
			if libErr == nil {
				documentStats = make(map[string]*bulk.DocumentStatsSummary)
				for _, doc := range lib.ListDocuments() {
					summary := &bulk.DocumentStatsSummary{
						Status:      string(doc.Status),
						DisplayName: doc.Name,
						IngestedAt:  doc.IngestedAt,
						Source:      doc.SourceInfo,
					}
					if doc.Stats != nil {
						summary.Triples = doc.Stats.TotalTriples
						summary.Articles = doc.Stats.Articles
						summary.Chapters = doc.Stats.Chapters
						summary.Definitions = doc.Stats.Definitions
						summary.References = doc.Stats.References
						summary.Rights = doc.Stats.Rights
						summary.Obligations = doc.Stats.Obligations
					}
					documentStats[doc.ID] = summary
				}
			}

			report := bulk.CollectStats(manifest, documentStats)

			switch formatFlag {
			case "json":
				fmt.Println(bulk.FormatStatsJSON(report))
			case "csv":
				fmt.Print(bulk.FormatStatsCSV(report))
			default:
				fmt.Print(bulk.FormatStatsTable(report))
			}

			return nil
		},
	}

	cmd.Flags().String("format", "table", "Output format (table, json, csv)")
	cmd.Flags().String("source", "", "Filter statistics to a specific source")
	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")

	return cmd
}

// --- Draft legislation analysis commands ---

func draftCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "draft",
		Short: "Draft legislation analysis pipeline",
		Long: `Parse draft Congressional bills and analyze their impact on existing law.

Commands:
  ingest    Parse a draft bill and display its structure and amendments
  diff      Compute structural diff against the USC knowledge graph
  impact    Run impact analysis against the USC knowledge graph
  conflicts Run conflict and consistency analysis
  simulate  Run compliance scenario simulation
  report    Generate comprehensive legislative impact report

Examples:
  regula draft ingest --bill draft-hr-1234.txt
  regula draft ingest --bill draft-hr-1234.txt --format json
  regula draft diff --bill draft-hr-1234.txt --path .regula
  regula draft diff --bill draft-hr-1234.txt --format csv
  regula draft impact --bill draft-hr-1234.txt --depth 2
  regula draft impact --bill draft-hr-1234.txt --format dot --output impact.dot
  regula draft conflicts --bill draft-hr-1234.txt
  regula draft conflicts --bill draft-hr-1234.txt --severity error
  regula draft simulate --bill draft-hr-1234.txt --scenario consent_withdrawal
  regula draft simulate --list-scenarios
  regula draft report --bill draft-hr-1234.txt --format markdown
  regula draft report --bill draft-hr-1234.txt --format html --output report.html`,
	}

	cmd.AddCommand(draftIngestCmd())
	cmd.AddCommand(draftDiffCmd())
	cmd.AddCommand(draftImpactCmd())
	cmd.AddCommand(draftConflictsCmd())
	cmd.AddCommand(draftSimulateCmd())
	cmd.AddCommand(draftReportCmd())

	return cmd
}

func draftIngestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Parse a draft bill and display its structure",
		Long: `Parse a draft Congressional bill file and display its structural
metadata, sections, and extracted amendments.

The parser extracts bill number, title, Congress, session, and
individual sections. The amendment recognizer then analyzes each
section's text to identify amendment instructions (strike-and-insert,
repeal, add new section, etc.) and their USC targets.

Examples:
  regula draft ingest --bill testdata/drafts/hr1234.txt
  regula draft ingest --bill draft-hr-1234.txt --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			billPath, _ := cmd.Flags().GetString("bill")
			formatFlag, _ := cmd.Flags().GetString("format")

			if billPath == "" {
				return fmt.Errorf("--bill flag is required: specify the path to a draft bill file")
			}

			bill, err := parseBillWithAmendments(billPath)
			if err != nil {
				return err
			}

			switch formatFlag {
			case "json":
				data, marshalErr := json.MarshalIndent(bill, "", "  ")
				if marshalErr != nil {
					return fmt.Errorf("failed to marshal JSON: %w", marshalErr)
				}
				fmt.Println(string(data))
			default:
				fmt.Print(formatIngestTable(bill))
			}

			return nil
		},
	}

	cmd.Flags().String("bill", "", "Path to draft bill file (required)")
	cmd.Flags().String("format", "table", "Output format (table, json)")

	return cmd
}

func draftDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Compute structural diff against the USC knowledge graph",
		Long: `Compute and display a structured diff between a draft bill's
amendments and existing provisions in the USC knowledge graph.

For each amendment, resolves the target provision in the library,
counts affected triples, and identifies cross-references. Results
are classified as added, removed, modified, or redesignated.

Requires a populated library (use 'regula bulk ingest' first).

Examples:
  regula draft diff --bill testdata/drafts/hr1234.txt
  regula draft diff --bill draft-hr-1234.txt --path .regula
  regula draft diff --bill draft-hr-1234.txt --format json
  regula draft diff --bill draft-hr-1234.txt --format csv`,
		RunE: func(cmd *cobra.Command, args []string) error {
			billPath, _ := cmd.Flags().GetString("bill")
			libraryPath, _ := cmd.Flags().GetString("path")
			formatFlag, _ := cmd.Flags().GetString("format")

			if billPath == "" {
				return fmt.Errorf("--bill flag is required: specify the path to a draft bill file")
			}

			bill, err := parseBillWithAmendments(billPath)
			if err != nil {
				return err
			}

			diffResult, err := draft.ComputeDiff(bill, libraryPath)
			if err != nil {
				return fmt.Errorf("diff computation failed: %w", err)
			}

			switch formatFlag {
			case "json":
				data, marshalErr := json.MarshalIndent(diffResult, "", "  ")
				if marshalErr != nil {
					return fmt.Errorf("failed to marshal JSON: %w", marshalErr)
				}
				fmt.Println(string(data))
			case "csv":
				fmt.Print(formatDiffCSV(diffResult))
			default:
				fmt.Print(formatDiffTable(diffResult))
			}

			return nil
		},
	}

	cmd.Flags().String("bill", "", "Path to draft bill file (required)")
	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")
	cmd.Flags().String("format", "table", "Output format (table, json, csv)")

	return cmd
}

// parseBillWithAmendments parses a draft bill file and runs amendment
// recognition on each section. The parser alone does not extract amendments;
// the Recognizer must be applied separately.
func parseBillWithAmendments(billPath string) (*draft.DraftBill, error) {
	bill, err := draft.ParseBillFromFile(billPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bill: %w", err)
	}

	recognizer := draft.NewRecognizer()
	for _, section := range bill.Sections {
		amendments, extractErr := recognizer.ExtractAmendments(section.RawText)
		if extractErr != nil {
			return nil, fmt.Errorf("failed to extract amendments from section %s: %w", section.Number, extractErr)
		}
		section.Amendments = amendments
	}

	return bill, nil
}

// collectTargetTitles returns a sorted list of unique USC title numbers
// referenced across all amendments in the bill.
func collectTargetTitles(bill *draft.DraftBill) []string {
	titleSet := make(map[string]bool)
	for _, section := range bill.Sections {
		for _, amendment := range section.Amendments {
			if amendment.TargetTitle != "" {
				titleSet[amendment.TargetTitle] = true
			}
		}
	}

	titles := make([]string, 0, len(titleSet))
	for title := range titleSet {
		titles = append(titles, title)
	}
	sort.Strings(titles)
	return titles
}

// formatIngestTable formats a parsed bill as a human-readable table showing
// bill metadata, per-section structure, and amendment counts.
func formatIngestTable(bill *draft.DraftBill) string {
	var builder strings.Builder
	stats := bill.Statistics()
	targetTitles := collectTargetTitles(bill)

	builder.WriteString(fmt.Sprintf("\nDraft Bill: %s\n", bill.String()))
	builder.WriteString(strings.Repeat("═", 70) + "\n")

	if bill.Congress != "" {
		builder.WriteString(fmt.Sprintf("  Congress:    %s\n", bill.Congress))
	}
	if bill.Session != "" {
		builder.WriteString(fmt.Sprintf("  Session:     %s\n", bill.Session))
	}
	builder.WriteString(fmt.Sprintf("  Sections:    %d\n", stats.SectionCount))
	builder.WriteString(fmt.Sprintf("  Amendments:  %d\n", stats.AmendmentCount))
	if len(targetTitles) > 0 {
		builder.WriteString(fmt.Sprintf("  Targets:     %d title(s) (%s)\n", len(targetTitles), strings.Join(targetTitles, ", ")))
	}
	builder.WriteString(strings.Repeat("─", 70) + "\n")

	// Per-section table
	builder.WriteString(fmt.Sprintf("  %-6s %-38s %s\n", "Sec.", "Title", "Amendments"))
	builder.WriteString(fmt.Sprintf("  %-6s %-38s %s\n", "────", "──────────────────────────────────────", "──────────"))
	for _, section := range bill.Sections {
		sectionTitle := section.Title
		if len(sectionTitle) > 38 {
			sectionTitle = sectionTitle[:35] + "..."
		}

		amendmentSummary := fmt.Sprintf("%d", len(section.Amendments))
		if len(section.Amendments) > 0 {
			sectionTargets := make(map[string]bool)
			for _, amendment := range section.Amendments {
				if amendment.TargetTitle != "" {
					sectionTargets[amendment.TargetTitle] = true
				}
			}
			targetList := make([]string, 0, len(sectionTargets))
			for title := range sectionTargets {
				targetList = append(targetList, "Title "+title)
			}
			sort.Strings(targetList)
			if len(targetList) > 0 {
				amendmentSummary += " → " + strings.Join(targetList, ", ")
			}
		}

		builder.WriteString(fmt.Sprintf("  %-6s %-38s %s\n", section.Number, sectionTitle, amendmentSummary))
	}

	builder.WriteString(strings.Repeat("─", 70) + "\n")
	return builder.String()
}

// formatDiffTable formats a DraftDiff as a styled table matching the output
// example from the issue specification.
func formatDiffTable(diffResult *draft.DraftDiff) string {
	var builder strings.Builder
	bill := diffResult.Bill
	stats := bill.Statistics()
	targetTitles := collectTargetTitles(bill)

	builder.WriteString(fmt.Sprintf("\nDraft Legislation Diff: %s\n", bill.BillNumber))
	builder.WriteString(strings.Repeat("═", 70) + "\n")
	builder.WriteString(fmt.Sprintf("  Bill sections: %d\n", stats.SectionCount))
	builder.WriteString(fmt.Sprintf("  Amendments:    %d\n", stats.AmendmentCount))
	if len(targetTitles) > 0 {
		builder.WriteString(fmt.Sprintf("  Targets:       %d title(s) (%s)\n", len(targetTitles), strings.Join(targetTitles, ", ")))
	}
	builder.WriteString("\n")

	// Modified entries
	if len(diffResult.Modified) > 0 {
		builder.WriteString(fmt.Sprintf("  MODIFIED (%d):\n", len(diffResult.Modified)))
		for _, entry := range diffResult.Modified {
			targetLabel := formatTargetLabel(entry)
			incomingCount := len(entry.CrossRefsTo)
			builder.WriteString(fmt.Sprintf("    %-30s %4d triples affected  %3d incoming refs\n",
				targetLabel, entry.AffectedTriples, incomingCount))
		}
		builder.WriteString("\n")
	}

	// Removed entries
	if len(diffResult.Removed) > 0 {
		builder.WriteString(fmt.Sprintf("  REPEALED (%d):\n", len(diffResult.Removed)))
		for _, entry := range diffResult.Removed {
			targetLabel := formatTargetLabel(entry)
			incomingCount := len(entry.CrossRefsTo)
			highImpact := ""
			if incomingCount >= 10 {
				highImpact = "  !! HIGH IMPACT"
			}
			builder.WriteString(fmt.Sprintf("    %-30s %4d triples affected  %3d incoming refs%s\n",
				targetLabel, entry.AffectedTriples, incomingCount, highImpact))
		}
		builder.WriteString("\n")
	}

	// Added entries
	if len(diffResult.Added) > 0 {
		builder.WriteString(fmt.Sprintf("  ADDED (%d):\n", len(diffResult.Added)))
		for _, entry := range diffResult.Added {
			targetLabel := formatTargetLabel(entry) + " (new)"
			incomingCount := len(entry.CrossRefsTo)
			builder.WriteString(fmt.Sprintf("    %-30s %4d triples affected  %3d incoming refs\n",
				targetLabel, entry.AffectedTriples, incomingCount))
		}
		builder.WriteString("\n")
	}

	// Redesignated entries
	if len(diffResult.Redesignated) > 0 {
		builder.WriteString(fmt.Sprintf("  REDESIGNATED (%d):\n", len(diffResult.Redesignated)))
		for _, entry := range diffResult.Redesignated {
			targetLabel := formatTargetLabel(entry)
			builder.WriteString(fmt.Sprintf("    %-30s %4d triples affected\n",
				targetLabel, entry.AffectedTriples))
		}
		builder.WriteString("\n")
	}

	// Unresolved targets
	builder.WriteString(fmt.Sprintf("  UNRESOLVED (%d)", len(diffResult.UnresolvedTargets)))
	if len(diffResult.UnresolvedTargets) > 0 {
		builder.WriteString(":\n")
		for _, target := range diffResult.UnresolvedTargets {
			builder.WriteString(fmt.Sprintf("    %s\n", target))
		}
	}
	builder.WriteString("\n\n")

	builder.WriteString(fmt.Sprintf("  Total triples affected: %d\n", diffResult.TriplesInvalidated))
	builder.WriteString(strings.Repeat("─", 70) + "\n")

	return builder.String()
}

// formatTargetLabel creates a human-readable label like "Title 15 §6502(d)"
// from a DiffEntry's amendment target fields.
func formatTargetLabel(entry draft.DiffEntry) string {
	label := "Title " + entry.Amendment.TargetTitle + " §" + entry.Amendment.TargetSection
	if entry.Amendment.TargetSubsection != "" {
		label += "(" + entry.Amendment.TargetSubsection + ")"
	}
	return label
}

// formatDiffCSV formats a DraftDiff as CSV with a header row and one row
// per diff entry across all categories.
func formatDiffCSV(diffResult *draft.DraftDiff) string {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	header := []string{"category", "title", "section", "subsection", "type", "triples_affected", "incoming_refs", "outgoing_refs", "target_uri", "document_id"}
	_ = writer.Write(header)

	writeDiffEntries := func(category string, entries []draft.DiffEntry) {
		for _, entry := range entries {
			row := []string{
				category,
				entry.Amendment.TargetTitle,
				entry.Amendment.TargetSection,
				entry.Amendment.TargetSubsection,
				string(entry.Amendment.Type),
				fmt.Sprintf("%d", entry.AffectedTriples),
				fmt.Sprintf("%d", len(entry.CrossRefsTo)),
				fmt.Sprintf("%d", len(entry.CrossRefsFrom)),
				entry.TargetURI,
				entry.TargetDocumentID,
			}
			_ = writer.Write(row)
		}
	}

	writeDiffEntries("modified", diffResult.Modified)
	writeDiffEntries("repealed", diffResult.Removed)
	writeDiffEntries("added", diffResult.Added)
	writeDiffEntries("redesignated", diffResult.Redesignated)

	writer.Flush()
	return buffer.String()
}

func draftImpactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "impact",
		Short: "Run impact analysis for a draft bill",
		Long: `Run transitive impact analysis for a draft bill against the USC
knowledge graph. Identifies directly and transitively affected provisions,
broken cross-references, and obligation/rights changes.

Requires a populated library (use 'regula bulk ingest' first).

Output formats:
  table   Styled summary with per-amendment breakdown (default)
  json    Full DraftImpactResult as indented JSON
  dot     Graphviz DOT graph (pipe to 'dot -Tpng' for rendering)

Examples:
  regula draft impact --bill draft-hr-1234.txt
  regula draft impact --bill draft-hr-1234.txt --depth 3
  regula draft impact --bill draft-hr-1234.txt --format json
  regula draft impact --bill draft-hr-1234.txt --format dot --output impact.dot
  regula draft impact --bill draft-hr-1234.txt --title-filter 15,42`,
		RunE: func(cmd *cobra.Command, args []string) error {
			billPath, _ := cmd.Flags().GetString("bill")
			libraryPath, _ := cmd.Flags().GetString("path")
			depthFlag, _ := cmd.Flags().GetInt("depth")
			formatFlag, _ := cmd.Flags().GetString("format")
			outputPath, _ := cmd.Flags().GetString("output")
			titleFilter, _ := cmd.Flags().GetString("title-filter")

			if billPath == "" {
				return fmt.Errorf("--bill flag is required: specify the path to a draft bill file")
			}

			bill, err := parseBillWithAmendments(billPath)
			if err != nil {
				return err
			}

			diffResult, err := draft.ComputeDiff(bill, libraryPath)
			if err != nil {
				return fmt.Errorf("diff computation failed: %w", err)
			}

			impactResult, err := draft.AnalyzeDraftImpact(diffResult, libraryPath, depthFlag)
			if err != nil {
				return fmt.Errorf("impact analysis failed: %w", err)
			}

			impactResult.SortByDepth()

			// Apply title filter if specified
			if titleFilter != "" {
				filterTitles := parseTitleFilter(titleFilter)
				impactResult = filterImpactByTitles(impactResult, filterTitles)
			}

			switch formatFlag {
			case "json":
				data, marshalErr := json.MarshalIndent(impactResult, "", "  ")
				if marshalErr != nil {
					return fmt.Errorf("failed to marshal JSON: %w", marshalErr)
				}
				fmt.Println(string(data))
			case "dot":
				dotContent, dotErr := draft.RenderImpactGraph(impactResult)
				if dotErr != nil {
					return fmt.Errorf("failed to render DOT graph: %w", dotErr)
				}
				if outputPath != "" {
					if writeErr := os.WriteFile(outputPath, []byte(dotContent), 0644); writeErr != nil {
						return fmt.Errorf("failed to write DOT file: %w", writeErr)
					}
					fmt.Fprintf(os.Stderr, "DOT graph written to %s\n", outputPath)
					fmt.Fprintf(os.Stderr, "Render with: dot -Tpng %s -o impact.png\n", outputPath)
				} else {
					fmt.Print(dotContent)
				}
			default:
				fmt.Print(formatImpactTable(impactResult))
			}

			return nil
		},
	}

	cmd.Flags().String("bill", "", "Path to draft bill file (required)")
	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")
	cmd.Flags().Int("depth", 2, "Transitive dependency depth (1=direct only)")
	cmd.Flags().String("format", "table", "Output format (table, json, dot)")
	cmd.Flags().String("output", "", "Output file path (useful for DOT format)")
	cmd.Flags().String("title-filter", "", "Limit analysis to specific USC titles (comma-separated, e.g. 15,42)")

	return cmd
}

// parseTitleFilter splits a comma-separated title filter string into a set
// of title numbers for lookup.
func parseTitleFilter(titleFilter string) map[string]bool {
	filterSet := make(map[string]bool)
	for _, segment := range strings.Split(titleFilter, ",") {
		trimmed := strings.TrimSpace(segment)
		if trimmed != "" {
			filterSet[trimmed] = true
		}
	}
	return filterSet
}

// filterImpactByTitles returns a copy of the impact result containing only
// provisions and broken refs whose document IDs match the specified title
// numbers. The diff is preserved unfiltered.
func filterImpactByTitles(result *draft.DraftImpactResult, titleNumbers map[string]bool) *draft.DraftImpactResult {
	matchesFilter := func(documentID string) bool {
		for titleNum := range titleNumbers {
			if strings.HasSuffix(documentID, "-"+titleNum) {
				return true
			}
		}
		return false
	}

	filtered := &draft.DraftImpactResult{
		Bill:              result.Bill,
		Diff:              result.Diff,
		ObligationChanges: result.ObligationChanges,
		RightsChanges:     result.RightsChanges,
		MaxDepthReached:   result.MaxDepthReached,
	}

	for _, provision := range result.DirectlyAffected {
		if matchesFilter(provision.DocumentID) {
			filtered.DirectlyAffected = append(filtered.DirectlyAffected, provision)
		}
	}
	for _, provision := range result.TransitivelyAffected {
		if matchesFilter(provision.DocumentID) {
			filtered.TransitivelyAffected = append(filtered.TransitivelyAffected, provision)
		}
	}
	for _, brokenRef := range result.BrokenCrossRefs {
		if matchesFilter(brokenRef.SourceDocumentID) {
			filtered.BrokenCrossRefs = append(filtered.BrokenCrossRefs, brokenRef)
		}
	}

	filtered.TotalProvisionsAffected = len(filtered.DirectlyAffected) + len(filtered.TransitivelyAffected)
	return filtered
}

// formatImpactTable formats a DraftImpactResult as a styled table matching
// the output example from issue #156.
func formatImpactTable(result *draft.DraftImpactResult) string {
	var builder strings.Builder
	bill := result.Bill
	targetTitles := collectTargetTitles(bill)

	billLabel := bill.BillNumber
	if bill.ShortTitle != "" {
		billLabel += " — " + bill.ShortTitle
	} else if bill.Title != "" {
		billLabel += " — " + bill.Title
	}

	builder.WriteString(fmt.Sprintf("\nDraft Impact Analysis: %s\n", billLabel))
	builder.WriteString(strings.Repeat("═", 70) + "\n")

	stats := bill.Statistics()
	titleSummary := ""
	if len(targetTitles) > 0 {
		titleSummary = fmt.Sprintf(" across %d title(s)", len(targetTitles))
	}
	builder.WriteString(fmt.Sprintf("  Amendments: %d%s\n", stats.AmendmentCount, titleSummary))
	builder.WriteString(fmt.Sprintf("  Directly affected: %d provisions\n", len(result.DirectlyAffected)))
	builder.WriteString(fmt.Sprintf("  Transitively affected: %d provisions (depth %d)\n", len(result.TransitivelyAffected), result.MaxDepthReached))
	builder.WriteString(fmt.Sprintf("  Broken cross-references: %d\n", len(result.BrokenCrossRefs)))
	builder.WriteString("\n")

	// Per-amendment impact breakdown
	if result.Diff != nil {
		amendmentEntries := collectAmendmentSummaryEntries(result)
		if len(amendmentEntries) > 0 {
			builder.WriteString("  IMPACT BY AMENDMENT:\n")
			builder.WriteString("  " + strings.Repeat("─", 58) + "\n")
			for _, entry := range amendmentEntries {
				builder.WriteString(entry)
			}
			builder.WriteString("\n")
		}
	}

	// Broken cross-references section
	if len(result.BrokenCrossRefs) > 0 {
		builder.WriteString("  BROKEN CROSS-REFERENCES:\n")
		builder.WriteString("  " + strings.Repeat("─", 58) + "\n")
		for _, brokenRef := range result.BrokenCrossRefs {
			severityTag := strings.ToUpper(brokenRef.Severity.String())
			builder.WriteString(fmt.Sprintf("    [%s] %s → %s (%s)\n",
				severityTag,
				truncateImpactLabel(brokenRef.SourceLabel, 30),
				truncateImpactLabel(brokenRef.TargetLabel, 20),
				brokenRef.Reason))
		}
		builder.WriteString("\n")
	}

	// Obligation/rights changes section
	obligationTotal := len(result.ObligationChanges.Added) +
		len(result.ObligationChanges.Removed) +
		len(result.ObligationChanges.Modified)
	rightsTotal := len(result.RightsChanges.Added) +
		len(result.RightsChanges.Removed)

	if obligationTotal > 0 || rightsTotal > 0 {
		builder.WriteString("  OBLIGATION/RIGHTS CHANGES:\n")
		builder.WriteString("  " + strings.Repeat("─", 58) + "\n")
		builder.WriteString(fmt.Sprintf("    Obligations added:    %d\n", len(result.ObligationChanges.Added)))
		builder.WriteString(fmt.Sprintf("    Obligations removed:  %d\n", len(result.ObligationChanges.Removed)))
		builder.WriteString(fmt.Sprintf("    Obligations modified: %d\n", len(result.ObligationChanges.Modified)))
		builder.WriteString(fmt.Sprintf("    Rights added:         %d\n", len(result.RightsChanges.Added)))
		builder.WriteString(fmt.Sprintf("    Rights removed:       %d\n", len(result.RightsChanges.Removed)))
		builder.WriteString("\n")
	}

	builder.WriteString(strings.Repeat("═", 70) + "\n")
	return builder.String()
}

// collectAmendmentSummaryEntries generates per-amendment summary lines for
// the impact table, showing direct/transitive counts and broken refs per
// modified or repealed target.
func collectAmendmentSummaryEntries(result *draft.DraftImpactResult) []string {
	var entries []string
	if result.Diff == nil {
		return entries
	}

	writeAmendmentGroup := func(diffEntries []draft.DiffEntry, changeLabel string) {
		for _, entry := range diffEntries {
			targetLabel := formatTargetLabel(entry)
			directCount := countAffectedByTarget(entry, result.DirectlyAffected)
			transitiveCount := countAffectedByTarget(entry, result.TransitivelyAffected)
			brokenRefCount := countBrokenRefsByTarget(entry, result.BrokenCrossRefs)

			var entryBuilder strings.Builder
			highImpact := ""
			if directCount >= 10 {
				highImpact = "  !! HIGH IMPACT"
			}
			entryBuilder.WriteString(fmt.Sprintf("  %s [%s]%s\n", targetLabel, changeLabel, highImpact))
			entryBuilder.WriteString(fmt.Sprintf("    Direct:      %d provisions reference this section\n", directCount))
			entryBuilder.WriteString(fmt.Sprintf("    Transitive:  %d provisions at depth 2+\n", transitiveCount))

			obligationCount := countObligationsByTarget(entry, result)
			if obligationCount > 0 {
				entryBuilder.WriteString(fmt.Sprintf("    Obligations: %d modified\n", obligationCount))
			}
			if brokenRefCount > 0 {
				entryBuilder.WriteString(fmt.Sprintf("    Broken refs: %d (severity: error)\n", brokenRefCount))
			}
			entryBuilder.WriteString("\n")
			entries = append(entries, entryBuilder.String())
		}
	}

	writeAmendmentGroup(result.Diff.Modified, "MODIFIED")
	writeAmendmentGroup(result.Diff.Removed, "REPEALED")
	writeAmendmentGroup(result.Diff.Added, "ADDED")
	writeAmendmentGroup(result.Diff.Redesignated, "REDESIGNATED")

	return entries
}

// countAffectedByTarget counts how many affected provisions reference the
// given diff entry's target, by checking if the provision's Reason field
// mentions the target URI label.
func countAffectedByTarget(entry draft.DiffEntry, provisions []draft.AffectedProvision) int {
	targetLabel := extractDiffTargetLabel(entry)
	count := 0
	for _, provision := range provisions {
		if strings.Contains(provision.Reason, targetLabel) {
			count++
		}
	}
	return count
}

// countBrokenRefsByTarget counts broken references whose target matches
// the given diff entry.
func countBrokenRefsByTarget(entry draft.DiffEntry, brokenRefs []draft.BrokenReference) int {
	count := 0
	for _, brokenRef := range brokenRefs {
		if brokenRef.TargetURI == entry.TargetURI {
			count++
		}
	}
	return count
}

// countObligationsByTarget counts obligation changes associated with a
// diff entry's target URI label.
func countObligationsByTarget(entry draft.DiffEntry, result *draft.DraftImpactResult) int {
	targetLabel := extractDiffTargetLabel(entry)
	count := 0
	for _, obligation := range result.ObligationChanges.Modified {
		if strings.Contains(obligation, targetLabel) {
			count++
		}
	}
	for _, obligation := range result.ObligationChanges.Removed {
		if strings.Contains(obligation, targetLabel) {
			count++
		}
	}
	return count
}

// extractDiffTargetLabel extracts the last segment of a diff entry's
// target URI for matching against reason strings.
func extractDiffTargetLabel(entry draft.DiffEntry) string {
	uri := entry.TargetURI
	for i := len(uri) - 1; i >= 0; i-- {
		if uri[i] == ':' || uri[i] == '/' || uri[i] == '#' {
			return uri[i+1:]
		}
	}
	return uri
}

// truncateImpactLabel shortens a label for table display.
func truncateImpactLabel(label string, maxLen int) string {
	if len(label) <= maxLen {
		return label
	}
	return label[:maxLen-3] + "..."
}

func draftConflictsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "conflicts",
		Short: "Run conflict and consistency analysis for a draft bill",
		Long: `Run conflict and consistency analysis for a draft bill against the USC
knowledge graph. Detects obligation conflicts, rights conflicts, and
temporal consistency issues.

Analysis types:
  - Obligation conflicts: contradictions, duplicates, orphaned obligations
  - Rights conflicts: narrowing, contradictions with obligations, expansions
  - Temporal issues: gaps, contradictions, retroactive application, sunsets

Output formats:
  table   Styled summary grouped by severity (default)
  json    Full analysis results as indented JSON

Severity levels:
  error   Direct contradictions that must be resolved
  warning Potential conflicts that should be reviewed
  info    Informational findings (duplicates, expansions)

Examples:
  regula draft conflicts --bill draft-hr-1234.txt
  regula draft conflicts --bill draft-hr-1234.txt --format json
  regula draft conflicts --bill draft-hr-1234.txt --severity error
  regula draft conflicts --bill draft-hr-1234.txt --skip-temporal`,
		RunE: func(cmd *cobra.Command, args []string) error {
			billPath, _ := cmd.Flags().GetString("bill")
			libraryPath, _ := cmd.Flags().GetString("path")
			formatFlag, _ := cmd.Flags().GetString("format")
			severityFilter, _ := cmd.Flags().GetString("severity")
			skipTemporal, _ := cmd.Flags().GetBool("skip-temporal")

			if billPath == "" {
				return fmt.Errorf("--bill flag is required: specify the path to a draft bill file")
			}

			bill, err := parseBillWithAmendments(billPath)
			if err != nil {
				return err
			}

			diffResult, err := draft.ComputeDiff(bill, libraryPath)
			if err != nil {
				return fmt.Errorf("diff computation failed: %w", err)
			}

			impactResult, err := draft.AnalyzeDraftImpact(diffResult, libraryPath, 2)
			if err != nil {
				return fmt.Errorf("impact analysis failed: %w", err)
			}

			// Run conflict detection
			conflictReport, err := draft.DetectObligationConflicts(diffResult, impactResult, libraryPath)
			if err != nil {
				return fmt.Errorf("obligation conflict detection failed: %w", err)
			}

			// Run rights conflict detection
			rightsConflicts, err := draft.DetectRightsConflicts(diffResult, impactResult, libraryPath)
			if err != nil {
				return fmt.Errorf("rights conflict detection failed: %w", err)
			}

			// Run temporal consistency analysis (unless skipped)
			var temporalFindings []draft.TemporalFinding
			if !skipTemporal {
				temporalFindings, err = draft.AnalyzeTemporalConsistency(diffResult, libraryPath)
				if err != nil {
					return fmt.Errorf("temporal analysis failed: %w", err)
				}
			}

			// Build combined analysis result
			analysisResult := buildConflictAnalysisResult(bill, conflictReport, rightsConflicts, temporalFindings)

			// Apply severity filter
			if severityFilter != "all" && severityFilter != "" {
				analysisResult = filterAnalysisBySeverity(analysisResult, severityFilter)
			}

			switch formatFlag {
			case "json":
				data, marshalErr := json.MarshalIndent(analysisResult, "", "  ")
				if marshalErr != nil {
					return fmt.Errorf("failed to marshal JSON: %w", marshalErr)
				}
				fmt.Println(string(data))
			default:
				fmt.Print(formatConflictsTable(analysisResult))
			}

			// Exit code 1 if any errors found (useful for CI/pipeline integration)
			if analysisResult.Summary.Errors > 0 {
				os.Exit(1)
			}

			return nil
		},
	}

	cmd.Flags().String("bill", "", "Path to draft bill file (required)")
	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")
	cmd.Flags().String("format", "table", "Output format (table, json)")
	cmd.Flags().String("severity", "all", "Filter by severity (error, warning, info, all)")
	cmd.Flags().Bool("skip-temporal", false, "Skip temporal consistency analysis")

	return cmd
}

// ConflictAnalysisResult aggregates all conflict and consistency analysis
// results for a draft bill.
type ConflictAnalysisResult struct {
	Bill             *draft.DraftBill       `json:"bill"`
	Conflicts        []ConflictEntry        `json:"conflicts"`
	TemporalFindings []TemporalEntry        `json:"temporal_findings"`
	Summary          ConflictAnalysisSummary `json:"summary"`
}

// ConflictEntry represents a unified conflict entry for CLI output.
type ConflictEntry struct {
	Category    string               `json:"category"`
	Type        string               `json:"type"`
	Severity    draft.ConflictSeverity `json:"severity"`
	Description string               `json:"description"`
	DraftText   string               `json:"draft_text,omitempty"`
	ExistingText string              `json:"existing_text,omitempty"`
	Provision   string               `json:"provision,omitempty"`
}

// TemporalEntry represents a temporal finding for CLI output.
type TemporalEntry struct {
	Type        string               `json:"type"`
	Severity    draft.ConflictSeverity `json:"severity"`
	Description string               `json:"description"`
	Provisions  []string             `json:"provisions,omitempty"`
}

// ConflictAnalysisSummary provides aggregate counts for the analysis.
type ConflictAnalysisSummary struct {
	TotalFindings    int `json:"total_findings"`
	Errors           int `json:"errors"`
	Warnings         int `json:"warnings"`
	Infos            int `json:"infos"`
	ObligationIssues int `json:"obligation_issues"`
	RightsIssues     int `json:"rights_issues"`
	TemporalIssues   int `json:"temporal_issues"`
}

// buildConflictAnalysisResult combines obligation conflicts, rights conflicts,
// and temporal findings into a unified analysis result.
func buildConflictAnalysisResult(
	bill *draft.DraftBill,
	conflictReport *draft.ConflictReport,
	rightsConflicts []draft.Conflict,
	temporalFindings []draft.TemporalFinding,
) *ConflictAnalysisResult {
	result := &ConflictAnalysisResult{
		Bill:             bill,
		Conflicts:        []ConflictEntry{},
		TemporalFindings: []TemporalEntry{},
	}

	// Add obligation conflicts
	if conflictReport != nil {
		for _, conflict := range conflictReport.Conflicts {
			result.Conflicts = append(result.Conflicts, ConflictEntry{
				Category:     "obligation",
				Type:         conflict.Type.String(),
				Severity:     conflict.Severity,
				Description:  conflict.Description,
				DraftText:    conflict.ProposedText,
				ExistingText: conflict.ExistingText,
				Provision:    conflict.ExistingProvision,
			})
		}
	}

	// Add rights conflicts
	for _, conflict := range rightsConflicts {
		result.Conflicts = append(result.Conflicts, ConflictEntry{
			Category:     "rights",
			Type:         conflict.Type.String(),
			Severity:     conflict.Severity,
			Description:  conflict.Description,
			DraftText:    conflict.ProposedText,
			ExistingText: conflict.ExistingText,
			Provision:    conflict.ExistingProvision,
		})
	}

	// Add temporal findings
	for _, finding := range temporalFindings {
		result.TemporalFindings = append(result.TemporalFindings, TemporalEntry{
			Type:        finding.Type.String(),
			Severity:    finding.Severity,
			Description: finding.Description,
			Provisions:  finding.Provisions,
		})
	}

	// Sort conflicts by severity (errors first)
	sortConflictEntries(result.Conflicts)

	// Compute summary
	result.Summary = computeConflictAnalysisSummary(result)

	return result
}

// sortConflictEntries sorts conflicts by severity (errors first, then warnings,
// then info).
func sortConflictEntries(conflicts []ConflictEntry) {
	// Simple bubble sort to maintain stability; conflicts are typically small
	for i := 0; i < len(conflicts); i++ {
		for j := i + 1; j < len(conflicts); j++ {
			if conflicts[j].Severity < conflicts[i].Severity {
				conflicts[i], conflicts[j] = conflicts[j], conflicts[i]
			}
		}
	}
}

// computeConflictAnalysisSummary aggregates counts from the analysis result.
func computeConflictAnalysisSummary(result *ConflictAnalysisResult) ConflictAnalysisSummary {
	summary := ConflictAnalysisSummary{}

	for _, conflict := range result.Conflicts {
		switch conflict.Severity {
		case draft.ConflictError:
			summary.Errors++
		case draft.ConflictWarning:
			summary.Warnings++
		case draft.ConflictInfo:
			summary.Infos++
		}

		if conflict.Category == "obligation" {
			summary.ObligationIssues++
		} else if conflict.Category == "rights" {
			summary.RightsIssues++
		}
	}

	for _, finding := range result.TemporalFindings {
		switch finding.Severity {
		case draft.ConflictError:
			summary.Errors++
		case draft.ConflictWarning:
			summary.Warnings++
		case draft.ConflictInfo:
			summary.Infos++
		}
		summary.TemporalIssues++
	}

	summary.TotalFindings = len(result.Conflicts) + len(result.TemporalFindings)
	return summary
}

// filterAnalysisBySeverity returns a copy of the analysis result containing
// only findings matching the specified severity level.
func filterAnalysisBySeverity(result *ConflictAnalysisResult, severityFilter string) *ConflictAnalysisResult {
	targetSeverity := parseSeverityFilter(severityFilter)
	if targetSeverity < 0 {
		return result // Invalid filter, return all
	}

	filtered := &ConflictAnalysisResult{
		Bill:             result.Bill,
		Conflicts:        []ConflictEntry{},
		TemporalFindings: []TemporalEntry{},
	}

	for _, conflict := range result.Conflicts {
		if conflict.Severity == draft.ConflictSeverity(targetSeverity) {
			filtered.Conflicts = append(filtered.Conflicts, conflict)
		}
	}

	for _, finding := range result.TemporalFindings {
		if finding.Severity == draft.ConflictSeverity(targetSeverity) {
			filtered.TemporalFindings = append(filtered.TemporalFindings, finding)
		}
	}

	filtered.Summary = computeConflictAnalysisSummary(filtered)
	return filtered
}

// parseSeverityFilter converts a severity string to its integer value.
func parseSeverityFilter(severity string) int {
	switch strings.ToLower(severity) {
	case "error":
		return int(draft.ConflictError)
	case "warning":
		return int(draft.ConflictWarning)
	case "info":
		return int(draft.ConflictInfo)
	default:
		return -1
	}
}

// formatConflictsTable formats the conflict analysis result as a styled table.
func formatConflictsTable(result *ConflictAnalysisResult) string {
	var builder strings.Builder
	bill := result.Bill

	billLabel := bill.BillNumber
	if bill.ShortTitle != "" {
		billLabel += " — " + bill.ShortTitle
	} else if bill.Title != "" {
		billLabel += " — " + bill.Title
	}

	builder.WriteString(fmt.Sprintf("\nDraft Conflict Analysis: %s\n", billLabel))
	builder.WriteString(strings.Repeat("═", 70) + "\n")

	summary := result.Summary
	summaryParts := []string{}
	if summary.Errors > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("%d error(s)", summary.Errors))
	}
	if summary.Warnings > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("%d warning(s)", summary.Warnings))
	}
	if summary.Infos > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("%d info", summary.Infos))
	}

	if summary.TotalFindings == 0 {
		builder.WriteString("  No conflicts or issues found.\n")
		builder.WriteString(strings.Repeat("═", 70) + "\n")
		return builder.String()
	}

	builder.WriteString(fmt.Sprintf("  Findings: %d (%s)\n\n", summary.TotalFindings, strings.Join(summaryParts, ", ")))

	// Group and display by severity
	if summary.Errors > 0 {
		builder.WriteString("  ERRORS:\n")
		builder.WriteString("  " + strings.Repeat("─", 58) + "\n")
		for _, conflict := range result.Conflicts {
			if conflict.Severity == draft.ConflictError {
				builder.WriteString(formatConflictEntry(conflict))
			}
		}
		for _, finding := range result.TemporalFindings {
			if finding.Severity == draft.ConflictError {
				builder.WriteString(formatTemporalEntry(finding))
			}
		}
		builder.WriteString("\n")
	}

	if summary.Warnings > 0 {
		builder.WriteString("  WARNINGS:\n")
		builder.WriteString("  " + strings.Repeat("─", 58) + "\n")
		for _, conflict := range result.Conflicts {
			if conflict.Severity == draft.ConflictWarning {
				builder.WriteString(formatConflictEntry(conflict))
			}
		}
		for _, finding := range result.TemporalFindings {
			if finding.Severity == draft.ConflictWarning {
				builder.WriteString(formatTemporalEntry(finding))
			}
		}
		builder.WriteString("\n")
	}

	if summary.Infos > 0 {
		builder.WriteString("  INFO:\n")
		builder.WriteString("  " + strings.Repeat("─", 58) + "\n")
		for _, conflict := range result.Conflicts {
			if conflict.Severity == draft.ConflictInfo {
				builder.WriteString(formatConflictEntry(conflict))
			}
		}
		for _, finding := range result.TemporalFindings {
			if finding.Severity == draft.ConflictInfo {
				builder.WriteString(formatTemporalEntry(finding))
			}
		}
		builder.WriteString("\n")
	}

	builder.WriteString(strings.Repeat("═", 70) + "\n")
	return builder.String()
}

// formatConflictEntry formats a single conflict entry for table display.
func formatConflictEntry(conflict ConflictEntry) string {
	var builder strings.Builder
	severityTag := strings.ToUpper(conflict.Severity.String())
	typeLabel := formatConflictTypeLabel(conflict.Type)

	builder.WriteString(fmt.Sprintf("    [%s] %s\n", severityTag, typeLabel))
	builder.WriteString(fmt.Sprintf("      %s\n", truncateConflictText(conflict.Description, 60)))

	if conflict.ExistingText != "" && conflict.DraftText != "" {
		builder.WriteString(fmt.Sprintf("      Existing: %s\n", truncateConflictText(conflict.ExistingText, 50)))
		builder.WriteString(fmt.Sprintf("      Proposed: %s\n", truncateConflictText(conflict.DraftText, 50)))
	}

	return builder.String()
}

// formatTemporalEntry formats a single temporal finding for table display.
func formatTemporalEntry(finding TemporalEntry) string {
	var builder strings.Builder
	severityTag := strings.ToUpper(finding.Severity.String())
	typeLabel := formatTemporalTypeLabel(finding.Type)

	builder.WriteString(fmt.Sprintf("    [%s] %s\n", severityTag, typeLabel))
	builder.WriteString(fmt.Sprintf("      %s\n", truncateConflictText(finding.Description, 60)))

	return builder.String()
}

// formatConflictTypeLabel converts a conflict type string to a human-readable label.
func formatConflictTypeLabel(conflictType string) string {
	labels := map[string]string{
		"obligation_contradiction": "Obligation contradiction",
		"obligation_duplicate":     "Duplicate obligation",
		"obligation_orphaned":      "Orphaned obligation",
		"rights_narrowing":         "Rights narrowing",
		"rights_contradiction":     "Rights-obligation conflict",
		"rights_expansion":         "Rights expansion",
	}
	if label, ok := labels[conflictType]; ok {
		return label
	}
	return conflictType
}

// formatTemporalTypeLabel converts a temporal type string to a human-readable label.
func formatTemporalTypeLabel(temporalType string) string {
	labels := map[string]string{
		"temporal_gap":           "Temporal gap",
		"temporal_contradiction": "Temporal contradiction",
		"temporal_retroactive":   "Retroactive application",
		"temporal_sunset":        "Sunset clause",
	}
	if label, ok := labels[temporalType]; ok {
		return label
	}
	return temporalType
}

// truncateConflictText truncates text for conflict table display.
func truncateConflictText(text string, maxLen int) string {
	// Normalize whitespace
	normalized := strings.Join(strings.Fields(text), " ")
	if len(normalized) <= maxLen {
		return normalized
	}
	return normalized[:maxLen-3] + "..."
}

// draftSimulateCmd creates the 'regula draft simulate' command.
func draftSimulateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "simulate",
		Short: "Run compliance scenario simulation",
		Long: `Run compliance scenarios against current law and proposed law (with draft overlay)
and display side-by-side comparison.

This command parses a draft bill, applies it as an overlay to the existing USC
knowledge graph, and compares scenario match results between baseline (current law)
and proposed (current law + draft amendments).

Available scenarios:
  consent_withdrawal  - Data subject withdraws consent for processing
  access_request      - Data subject requests access to their personal data
  erasure_request     - Data subject requests deletion of their personal data
  data_breach         - Personal data breach handling scenario

Use --list-scenarios to see all available scenarios with descriptions.

Examples:
  regula draft simulate --list-scenarios
  regula draft simulate --bill draft-hr-1234.txt --scenario consent_withdrawal
  regula draft simulate --bill draft-hr-1234.txt --scenario access_request --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			listScenarios, _ := cmd.Flags().GetBool("list-scenarios")
			billPath, _ := cmd.Flags().GetString("bill")
			scenarioName, _ := cmd.Flags().GetString("scenario")
			libraryPath, _ := cmd.Flags().GetString("path")
			formatFlag, _ := cmd.Flags().GetString("format")

			// Handle --list-scenarios
			if listScenarios {
				fmt.Print(formatScenarioList())
				return nil
			}

			// Validate required flags
			if billPath == "" {
				return fmt.Errorf("--bill flag is required: specify the path to a draft bill file")
			}
			if scenarioName == "" {
				return fmt.Errorf("--scenario flag is required: specify a scenario name (use --list-scenarios to see available)")
			}

			// Get the scenario
			scenario, ok := simulate.PredefinedScenarios[scenarioName]
			if !ok {
				return fmt.Errorf("unknown scenario '%s' (use --list-scenarios to see available)", scenarioName)
			}

			// Parse the bill with amendments
			bill, err := parseBillWithAmendments(billPath)
			if err != nil {
				return err
			}

			// Compute diff against the library
			diffResult, err := draft.ComputeDiff(bill, libraryPath)
			if err != nil {
				return fmt.Errorf("diff computation failed: %w", err)
			}

			// Open the library to get base store and URI
			lib, err := library.Open(libraryPath)
			if err != nil {
				return fmt.Errorf("failed to open library: %w", err)
			}

			// Apply draft overlay
			overlay, err := draft.ApplyDraftOverlay(diffResult, libraryPath)
			if err != nil {
				return fmt.Errorf("failed to apply draft overlay: %w", err)
			}

			// Load a base store for comparison (merge all document stores)
			baseStore, err := loadMergedTripleStore(lib, diffResult)
			if err != nil {
				return fmt.Errorf("failed to load base store: %w", err)
			}

			// Compare scenarios
			comparison, err := draft.CompareScenarios(
				scenarioName,
				scenario,
				baseStore,
				overlay.OverlayStore,
				lib.BaseURI(),
				bill,
			)
			if err != nil {
				return fmt.Errorf("scenario comparison failed: %w", err)
			}

			// Output based on format
			switch formatFlag {
			case "json":
				output := draft.FormatScenarioComparison(comparison, "json")
				fmt.Println(output)
			default:
				output := formatSimulateTable(comparison, bill)
				fmt.Print(output)
			}

			return nil
		},
	}

	cmd.Flags().Bool("list-scenarios", false, "List available scenarios")
	cmd.Flags().String("bill", "", "Path to draft bill file")
	cmd.Flags().String("scenario", "", "Scenario name to simulate")
	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")
	cmd.Flags().String("format", "table", "Output format (table, json)")

	return cmd
}

// formatScenarioList formats the list of available scenarios.
func formatScenarioList() string {
	var sb strings.Builder

	sb.WriteString("Available Scenarios\n")
	sb.WriteString(strings.Repeat("=", 60) + "\n\n")

	scenarios := []struct {
		id   string
		name string
		desc string
	}{
		{"consent_withdrawal", "Consent Withdrawal", "Data subject withdraws previously given consent for data processing"},
		{"access_request", "Data Access Request", "Data subject requests access to their personal data"},
		{"erasure_request", "Data Erasure Request", "Data subject requests erasure of their personal data"},
		{"data_breach", "Data Breach", "Personal data breach occurs and must be handled"},
	}

	for _, s := range scenarios {
		sb.WriteString(fmt.Sprintf("  %-20s  %s\n", s.id, s.name))
		sb.WriteString(fmt.Sprintf("  %-20s  %s\n\n", "", s.desc))
	}

	sb.WriteString("Usage:\n")
	sb.WriteString("  regula draft simulate --bill <path> --scenario <id>\n")

	return sb.String()
}

// loadMergedTripleStore loads and merges triple stores for all documents affected by the diff.
func loadMergedTripleStore(lib *library.Library, diffResult *draft.DraftDiff) (*store.TripleStore, error) {
	merged := store.NewTripleStore()

	// Collect unique document IDs from the diff
	docIDs := make(map[string]bool)
	for _, entry := range diffResult.Added {
		docIDs[entry.TargetDocumentID] = true
	}
	for _, entry := range diffResult.Removed {
		docIDs[entry.TargetDocumentID] = true
	}
	for _, entry := range diffResult.Modified {
		docIDs[entry.TargetDocumentID] = true
	}
	for _, entry := range diffResult.Redesignated {
		docIDs[entry.TargetDocumentID] = true
	}

	// If no documents affected, try to load a default document
	if len(docIDs) == 0 {
		// List all documents in the library
		docs := lib.ListDocuments()
		for _, doc := range docs {
			docIDs[doc.ID] = true
		}
	}

	// Load and merge each document's triple store
	for docID := range docIDs {
		docStore, err := lib.LoadTripleStore(docID)
		if err != nil {
			continue // Skip documents that can't be loaded
		}
		merged.MergeFrom(docStore)
	}

	return merged, nil
}

// formatSimulateTable formats the scenario comparison as a styled table.
func formatSimulateTable(comparison *draft.ScenarioComparison, bill *draft.DraftBill) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("Scenario Comparison: %s\n", comparison.Scenario))

	billLabel := bill.BillNumber
	if bill.ShortTitle != "" {
		billLabel += " — " + bill.ShortTitle
	} else if bill.Title != "" {
		billLabel += " — " + bill.Title
	}
	sb.WriteString(fmt.Sprintf("Bill: %s\n", billLabel))
	sb.WriteString(strings.Repeat("═", 66) + "\n\n")

	// Summary table
	baselineCount := 0
	proposedCount := 0
	baselineObligations := 0
	proposedObligations := 0
	baselineRights := 0
	proposedRights := 0

	if comparison.Baseline != nil && comparison.Baseline.Summary != nil {
		baselineCount = comparison.Baseline.Summary.TotalMatches
		baselineObligations = len(comparison.Baseline.Summary.ObligationsInvolved)
		baselineRights = len(comparison.Baseline.Summary.RightsInvolved)
	}
	if comparison.Proposed != nil && comparison.Proposed.Summary != nil {
		proposedCount = comparison.Proposed.Summary.TotalMatches
		proposedObligations = len(comparison.Proposed.Summary.ObligationsInvolved)
		proposedRights = len(comparison.Proposed.Summary.RightsInvolved)
	}

	sb.WriteString(fmt.Sprintf("                        %-15s %-15s\n", "CURRENT LAW", "PROPOSED LAW"))
	sb.WriteString("  " + strings.Repeat("─", 62) + "\n")

	// Applicable provisions
	provDiff := ""
	if proposedCount > baselineCount {
		provDiff = fmt.Sprintf(" (+%d)", proposedCount-baselineCount)
	} else if proposedCount < baselineCount {
		provDiff = fmt.Sprintf(" (-%d)", baselineCount-proposedCount)
	}
	sb.WriteString(fmt.Sprintf("  Applicable provisions:  %-15d %d%s\n", baselineCount, proposedCount, provDiff))

	// Obligations triggered
	obligDiff := ""
	if proposedObligations > baselineObligations {
		obligDiff = fmt.Sprintf(" (+%d)", proposedObligations-baselineObligations)
	} else if proposedObligations < baselineObligations {
		obligDiff = fmt.Sprintf(" (-%d)", baselineObligations-proposedObligations)
	}
	sb.WriteString(fmt.Sprintf("  Obligations triggered:  %-15d %d%s\n", baselineObligations, proposedObligations, obligDiff))

	// Rights involved
	rightsDiff := ""
	if proposedRights > baselineRights {
		rightsDiff = fmt.Sprintf(" (+%d)", proposedRights-baselineRights)
	} else if proposedRights < baselineRights {
		rightsDiff = fmt.Sprintf(" (-%d)", baselineRights-proposedRights)
	}
	sb.WriteString(fmt.Sprintf("  Rights involved:        %-15d %d%s\n", baselineRights, proposedRights, rightsDiff))
	sb.WriteString("\n")

	// Newly applicable provisions
	sb.WriteString(fmt.Sprintf("  NEWLY APPLICABLE (+%d):\n", len(comparison.NewlyApplicable)))
	if len(comparison.NewlyApplicable) == 0 {
		sb.WriteString("    (none)\n")
	} else {
		for _, diff := range comparison.NewlyApplicable {
			label := diff.Label
			if label == "" {
				label = extractSimulateURILabel(diff.URI)
			}
			sb.WriteString(fmt.Sprintf("    [%-7s] %s\n", diff.ProposedRelevance, label))
		}
	}
	sb.WriteString("\n")

	// No longer applicable provisions
	sb.WriteString(fmt.Sprintf("  NO LONGER APPLICABLE (-%d):\n", len(comparison.NoLongerApplicable)))
	if len(comparison.NoLongerApplicable) == 0 {
		sb.WriteString("    (none)\n")
	} else {
		for _, diff := range comparison.NoLongerApplicable {
			label := diff.Label
			if label == "" {
				label = extractSimulateURILabel(diff.URI)
			}
			sb.WriteString(fmt.Sprintf("    [%-7s] %s\n", diff.BaselineRelevance, label))
		}
	}
	sb.WriteString("\n")

	// Changed relevance
	sb.WriteString(fmt.Sprintf("  CHANGED RELEVANCE (%d):\n", len(comparison.ChangedRelevance)))
	if len(comparison.ChangedRelevance) == 0 {
		sb.WriteString("    (none)\n")
	} else {
		for _, diff := range comparison.ChangedRelevance {
			label := diff.Label
			if label == "" {
				label = extractSimulateURILabel(diff.URI)
			}
			sb.WriteString(fmt.Sprintf("    %s: %s → %s\n", label, diff.BaselineRelevance, diff.ProposedRelevance))
		}
	}
	sb.WriteString("\n")

	// Obligation diff
	if comparison.ObligationsDiff.Added > 0 || comparison.ObligationsDiff.Removed > 0 {
		sb.WriteString("  OBLIGATION DIFF:\n")
		if comparison.ObligationsDiff.Added > 0 {
			sb.WriteString(fmt.Sprintf("    + %d new obligation type(s)\n", comparison.ObligationsDiff.Added))
		}
		if comparison.ObligationsDiff.Removed > 0 {
			sb.WriteString(fmt.Sprintf("    - %d obligation type(s) removed\n", comparison.ObligationsDiff.Removed))
		}
		sb.WriteString("\n")
	}

	// Rights diff
	if comparison.RightsDiff.Added > 0 || comparison.RightsDiff.Removed > 0 {
		sb.WriteString("  RIGHTS DIFF:\n")
		if comparison.RightsDiff.Added > 0 {
			sb.WriteString(fmt.Sprintf("    + %d new right type(s)\n", comparison.RightsDiff.Added))
		}
		if comparison.RightsDiff.Removed > 0 {
			sb.WriteString(fmt.Sprintf("    - %d right type(s) removed\n", comparison.RightsDiff.Removed))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(strings.Repeat("═", 66) + "\n")

	return sb.String()
}

// extractSimulateURILabel extracts a human-readable label from a URI.
func extractSimulateURILabel(uri string) string {
	if uri == "" {
		return ""
	}

	// Check for fragment first (#)
	if idx := strings.LastIndex(uri, "#"); idx != -1 {
		return uri[idx+1:]
	}
	// Then check for colon (e.g., "GDPR:Art6" -> "Art6")
	// But skip colons that are part of URL scheme (://)
	if idx := strings.LastIndex(uri, ":"); idx != -1 {
		if idx+2 < len(uri) && uri[idx+1:idx+3] != "//" {
			return uri[idx+1:]
		}
	}
	// Finally check for path separator
	if idx := strings.LastIndex(uri, "/"); idx != -1 {
		return uri[idx+1:]
	}
	return uri
}

func draftReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate comprehensive legislative impact report",
		Long: `Generate a full legislative impact report for a draft bill.

This command runs the complete analysis pipeline:
  1. Parse draft bill and extract amendments
  2. Compute structural diff against USC knowledge graph
  3. Analyze transitive impact across provisions
  4. Detect obligation and rights conflicts
  5. Analyze temporal consistency
  6. Run scenario comparisons (optional)
  7. Generate formatted report

The report aggregates all findings into a single document with:
  - Executive summary with key metrics
  - Risk level assessment (low/medium/high)
  - Structural changes (modified/repealed/added provisions)
  - Impact analysis (direct and transitive effects)
  - Conflict findings (errors/warnings/info)
  - Temporal analysis (gaps, contradictions, retroactive application)
  - Scenario comparisons (baseline vs proposed)
  - Visualization (DOT graph for Markdown/HTML)

Exit codes reflect risk level:
  0 = low risk
  1 = medium risk
  2 = high risk

Examples:
  # Full report to markdown (default)
  regula draft report --bill draft-hr-1234.txt --path .regula

  # HTML report to file
  regula draft report --bill draft-hr-1234.txt --format html --output report.html

  # Quick analysis without scenarios
  regula draft report --bill draft-hr-1234.txt --skip-scenarios

  # JSON for programmatic consumption
  regula draft report --bill draft-hr-1234.txt --format json > report.json

  # Deep impact analysis
  regula draft report --bill draft-hr-1234.txt --depth 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			billPath, _ := cmd.Flags().GetString("bill")
			libraryPath, _ := cmd.Flags().GetString("path")
			formatFlag, _ := cmd.Flags().GetString("format")
			outputPath, _ := cmd.Flags().GetString("output")
			depthFlag, _ := cmd.Flags().GetInt("depth")
			scenariosFlag, _ := cmd.Flags().GetString("scenarios")
			skipTemporal, _ := cmd.Flags().GetBool("skip-temporal")
			skipScenarios, _ := cmd.Flags().GetBool("skip-scenarios")

			// Validate required flags
			if billPath == "" {
				return fmt.Errorf("--bill flag is required: specify the path to a draft bill file")
			}

			// Parse the bill with amendments
			bill, err := parseBillWithAmendments(billPath)
			if err != nil {
				return err
			}

			// Build report options
			options := draft.ReportOptions{
				IncludeDiff:          true,
				IncludeImpact:        true,
				ImpactDepth:          depthFlag,
				IncludeConflicts:     true,
				IncludeTemporal:      !skipTemporal,
				IncludeVisualization: formatFlag != "json", // Skip DOT for JSON
				Scenarios:            []string{},
			}

			// Parse scenarios flag
			if !skipScenarios && scenariosFlag != "none" {
				if scenariosFlag == "all" {
					for scenarioID := range simulate.PredefinedScenarios {
						options.Scenarios = append(options.Scenarios, scenarioID)
					}
				} else if scenariosFlag != "" {
					options.Scenarios = strings.Split(scenariosFlag, ",")
					for i, s := range options.Scenarios {
						options.Scenarios[i] = strings.TrimSpace(s)
					}
				}
			}

			// Generate the report
			report, err := draft.GenerateReport(bill, libraryPath, options)
			if err != nil {
				// GenerateReport may return partial report with error
				if report == nil {
					return fmt.Errorf("report generation failed: %w", err)
				}
				// Log warning but continue with partial report
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}

			// Run scenario comparisons if requested
			if len(options.Scenarios) > 0 && report.Diff != nil {
				scenarioResults, scenarioErr := runReportScenarios(report, libraryPath, options.Scenarios)
				if scenarioErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: scenario comparison failed: %v\n", scenarioErr)
				} else {
					report.ScenarioResults = scenarioResults
				}
			}

			// Render the report in the requested format
			var output string
			var renderErr error
			switch strings.ToLower(formatFlag) {
			case "json":
				output, renderErr = draft.RenderReportJSON(report)
			case "html":
				output, renderErr = draft.RenderReportHTML(report)
			case "markdown", "md":
				fallthrough
			default:
				output, renderErr = draft.RenderReportMarkdown(report)
			}

			if renderErr != nil {
				return fmt.Errorf("failed to render report: %w", renderErr)
			}

			// Write output
			if outputPath != "" {
				if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
					return fmt.Errorf("failed to write output file: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Report written to %s\n", outputPath)
			} else {
				fmt.Print(output)
			}

			// Exit with code based on risk level
			switch report.RiskLevel {
			case draft.RiskMedium:
				os.Exit(1)
			case draft.RiskHigh:
				os.Exit(2)
			}

			return nil
		},
	}

	cmd.Flags().String("bill", "", "Path to draft bill file (required)")
	cmd.Flags().String("path", defaultLibraryPath(), "Library directory path")
	cmd.Flags().String("format", "markdown", "Output format: markdown, json, html")
	cmd.Flags().String("output", "", "Output file path (default: stdout)")
	cmd.Flags().Int("depth", 2, "Transitive impact analysis depth")
	cmd.Flags().String("scenarios", "none", "Scenarios to test (comma-separated, 'all', or 'none')")
	cmd.Flags().Bool("skip-temporal", false, "Skip temporal consistency analysis")
	cmd.Flags().Bool("skip-scenarios", false, "Skip scenario comparison (faster)")

	return cmd
}

// runReportScenarios runs scenario comparisons for the report.
func runReportScenarios(report *draft.LegislativeImpactReport, libraryPath string, scenarioIDs []string) ([]*draft.ScenarioComparison, error) {
	if report.Diff == nil {
		return nil, fmt.Errorf("diff is nil")
	}

	lib, err := library.Open(libraryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open library: %w", err)
	}

	// Apply draft overlay
	overlay, err := draft.ApplyDraftOverlay(report.Diff, libraryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to apply draft overlay: %w", err)
	}

	// Load base store
	baseStore, err := loadMergedTripleStore(lib, report.Diff)
	if err != nil {
		return nil, fmt.Errorf("failed to load base store: %w", err)
	}

	var results []*draft.ScenarioComparison

	for _, scenarioID := range scenarioIDs {
		scenario, ok := simulate.PredefinedScenarios[scenarioID]
		if !ok {
			continue // Skip unknown scenarios
		}

		comparison, compareErr := draft.CompareScenarios(
			scenarioID,
			scenario,
			baseStore,
			overlay.OverlayStore,
			lib.BaseURI(),
			report.Bill,
		)
		if compareErr != nil {
			continue // Skip failed comparisons
		}

		results = append(results, comparison)
	}

	return results, nil
}

// searchCmd returns the search command for finding committee jurisdictions.
func searchCmd() *cobra.Command {
	var sourcePath string
	var committeeQuery string
	var keywordQuery string
	var templateName string
	var listCommittees bool
	var listTemplates bool
	var formatOutput string
	var limitResults int

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search for committee jurisdictions or procedural keywords in House Rules",
		Long: `Search House Rules by committee jurisdiction or procedural keyword.

COMMITTEE SEARCH (Rule X):
Uses Rule X, clause 1 of the House Rules to find which committee
has jurisdiction over a given subject matter.

KEYWORD SEARCH (all rules):
Searches across all House Rules for procedural concepts. Use --keyword
for free-text search or --template for pre-built procedural queries.

Examples:
  # Find committees with cybersecurity jurisdiction
  regula search --source house-rules-119th.txt --committee cybersecurity

  # Search for "quorum" across all rules
  regula search --source house-rules-119th.txt --keyword quorum

  # Use a pre-built template for voting procedures
  regula search --source house-rules-119th.txt --template voting

  # List available templates
  regula search --list-templates

  # List all committees and their jurisdictions
  regula search --source house-rules-119th.txt --list-committees

  # Output as JSON with limit
  regula search --source house-rules-119th.txt --keyword amendment --format json --limit 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle --list-templates (no source required)
			if listTemplates {
				return outputTemplateList(formatOutput)
			}

			if sourcePath == "" {
				return fmt.Errorf("--source flag is required")
			}

			// Read the source file
			data, err := os.ReadFile(sourcePath)
			if err != nil {
				return fmt.Errorf("failed to read source file: %w", err)
			}
			text := string(data)

			// Handle --keyword or --template search (procedural keyword navigator)
			if keywordQuery != "" || templateName != "" {
				searcher := extract.NewKeywordSearcher()
				searcher.ParseHouseRules(text)

				var matches []extract.KeywordMatch
				var queryLabel string

				if templateName != "" {
					matches = searcher.SearchWithTemplate(templateName)
					queryLabel = templateName + " (template)"
				} else {
					matches = searcher.Search(keywordQuery)
					queryLabel = keywordQuery
				}

				if len(matches) == 0 {
					fmt.Printf("No matches found for %q\n", queryLabel)
					return nil
				}

				if limitResults > 0 && len(matches) > limitResults {
					matches = matches[:limitResults]
				}

				return outputKeywordResults(matches, queryLabel, formatOutput)
			}

			// Handle committee-based search (original functionality)

			// Find Rule X section
			ruleXStart := strings.Index(text, "RULE X")
			if ruleXStart == -1 {
				return fmt.Errorf("could not find Rule X in document")
			}

			// Find Rule XI to delimit
			ruleXEnd := strings.Index(text[ruleXStart+6:], "RULE XI")
			if ruleXEnd == -1 {
				ruleXEnd = len(text) - ruleXStart
			} else {
				ruleXEnd += ruleXStart + 6
			}

			ruleXText := text[ruleXStart:ruleXEnd]

			// Extract committees
			extractor := extract.NewCommitteeJurisdictionExtractor()
			committees := extractor.ExtractFromRuleX(ruleXText)

			if len(committees) == 0 {
				return fmt.Errorf("no committees found in Rule X")
			}

			// Handle --list-committees
			if listCommittees {
				return outputCommitteeList(committees, formatOutput)
			}

			// Handle --committee search
			if committeeQuery == "" {
				return fmt.Errorf("use --committee, --keyword, --template, --list-committees, or --list-templates")
			}

			matches := extract.SearchCommitteeByTopic(committees, committeeQuery)
			if len(matches) == 0 {
				fmt.Printf("No committees found with jurisdiction over %q\n", committeeQuery)
				return nil
			}

			return outputSearchResults(matches, committeeQuery, formatOutput)
		},
	}

	cmd.Flags().StringVar(&sourcePath, "source", "", "Path to House Rules source file")
	cmd.Flags().StringVar(&committeeQuery, "committee", "", "Topic to search for in committee jurisdictions")
	cmd.Flags().StringVar(&keywordQuery, "keyword", "", "Keyword to search across all House Rules")
	cmd.Flags().StringVar(&templateName, "template", "", "Pre-built template (voting, quorum, amendments, debate, etc.)")
	cmd.Flags().BoolVar(&listCommittees, "list-committees", false, "List all committees and their jurisdictions")
	cmd.Flags().BoolVar(&listTemplates, "list-templates", false, "List available procedural keyword templates")
	cmd.Flags().StringVar(&formatOutput, "format", "table", "Output format (table, json)")
	cmd.Flags().IntVar(&limitResults, "limit", 0, "Limit number of results (0 for unlimited)")

	return cmd
}

// outputCommitteeList prints the list of all committees.
func outputCommitteeList(committees []extract.CommitteeJurisdiction, format string) error {
	if format == "json" {
		data, err := json.MarshalIndent(committees, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	// Table format
	fmt.Println("Congressional Committees (from Rule X)")
	fmt.Println(strings.Repeat("═", 70))

	for _, committee := range committees {
		fmt.Printf("\n[%s] %s\n", committee.Letter, committee.Name)
		fmt.Printf("    Source: %s\n", committee.SourceClause)
		fmt.Printf("    Jurisdictions: %d topics\n", len(committee.Topics))

		// Show first 5 topics
		shown := 0
		for _, topic := range committee.Topics {
			if shown >= 5 {
				remaining := len(committee.Topics) - 5
				fmt.Printf("      ... and %d more\n", remaining)
				break
			}
			topicText := topic.Text
			if len(topicText) > 60 {
				topicText = topicText[:57] + "..."
			}
			fmt.Printf("      (%s) %s\n", topic.Number, topicText)
			shown++
		}
	}

	fmt.Printf("\nTotal: %d committees\n", len(committees))
	return nil
}

// outputSearchResults prints the search results.
func outputSearchResults(matches []extract.CommitteeJurisdictionMatch, query, format string) error {
	if format == "json" {
		type jsonMatch struct {
			Committee    string `json:"committee"`
			Jurisdiction string `json:"jurisdiction"`
			Source       string `json:"source"`
		}
		var jsonMatches []jsonMatch
		for _, m := range matches {
			jsonMatches = append(jsonMatches, jsonMatch{
				Committee:    m.Committee.Name,
				Jurisdiction: m.MatchedTopic.Text,
				Source:       m.SourceRef,
			})
		}
		data, err := json.MarshalIndent(jsonMatches, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	// Table format
	fmt.Printf("Committees with jurisdiction over %q\n", query)
	fmt.Println(strings.Repeat("═", 70))

	// Group by committee
	byCommittee := make(map[string][]extract.CommitteeJurisdictionMatch)
	for _, m := range matches {
		byCommittee[m.Committee.Name] = append(byCommittee[m.Committee.Name], m)
	}

	for committeeName, committeeMatches := range byCommittee {
		fmt.Printf("\n%s\n", committeeName)
		for _, m := range committeeMatches {
			jurisdictionText := m.MatchedTopic.Text
			if len(jurisdictionText) > 60 {
				jurisdictionText = jurisdictionText[:57] + "..."
			}
			fmt.Printf("  Jurisdiction: %q\n", jurisdictionText)
			fmt.Printf("  Source: %s\n", m.SourceRef)
		}
	}

	fmt.Printf("\nTotal: %d matches across %d committees\n", len(matches), len(byCommittee))
	return nil
}

// outputKeywordResults prints keyword search results.
func outputKeywordResults(matches []extract.KeywordMatch, query, format string) error {
	if format == "json" {
		type jsonMatch struct {
			Rule        string `json:"rule"`
			RuleTitle   string `json:"rule_title"`
			Clause      string `json:"clause"`
			ClauseTitle string `json:"clause_title,omitempty"`
			Context     string `json:"context"`
			Score       int    `json:"score"`
			MatchCount  int    `json:"match_count"`
		}
		var jsonMatches []jsonMatch
		for _, m := range matches {
			jsonMatches = append(jsonMatches, jsonMatch{
				Rule:        m.Rule,
				RuleTitle:   m.RuleTitle,
				Clause:      m.Clause,
				ClauseTitle: m.ClauseTitle,
				Context:     m.Context,
				Score:       m.Score,
				MatchCount:  m.MatchCount,
			})
		}
		data, err := json.MarshalIndent(jsonMatches, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	// Table format
	fmt.Printf("Search results for %q\n", query)
	fmt.Println(strings.Repeat("═", 70))

	for i, m := range matches {
		ruleRef := fmt.Sprintf("Rule %s, clause %s", m.Rule, m.Clause)
		if m.ClauseTitle != "" {
			fmt.Printf("\n[%d] %s: %s\n", i+1, ruleRef, m.ClauseTitle)
		} else if m.RuleTitle != "" {
			fmt.Printf("\n[%d] %s (%s)\n", i+1, ruleRef, m.RuleTitle)
		} else {
			fmt.Printf("\n[%d] %s\n", i+1, ruleRef)
		}
		fmt.Printf("    Context: %q\n", m.Context)
		fmt.Printf("    Matches: %d (score: %d)\n", m.MatchCount, m.Score)
	}

	fmt.Printf("\nTotal: %d matches\n", len(matches))
	return nil
}

// outputTemplateList prints the available procedural keyword templates.
func outputTemplateList(format string) error {
	templates := extract.GetTemplates()

	if format == "json" {
		type templateInfo struct {
			Name     string   `json:"name"`
			Keywords []string `json:"keywords"`
		}
		var templateList []templateInfo
		for _, name := range templates {
			templateList = append(templateList, templateInfo{
				Name:     name,
				Keywords: extract.ProceduralKeywords[name],
			})
		}
		data, err := json.MarshalIndent(templateList, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	// Table format
	fmt.Println("Available Procedural Keyword Templates")
	fmt.Println(strings.Repeat("═", 70))

	for _, name := range templates {
		keywords := extract.ProceduralKeywords[name]
		keywordSummary := strings.Join(keywords[:min(len(keywords), 5)], ", ")
		if len(keywords) > 5 {
			keywordSummary += fmt.Sprintf(", ... (%d total)", len(keywords))
		}
		fmt.Printf("\n%s:\n", name)
		fmt.Printf("  Keywords: %s\n", keywordSummary)
	}

	fmt.Printf("\nTotal: %d templates\n", len(templates))
	fmt.Println("\nUsage: regula search --source <file> --template <template-name>")
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func navigateCmd() *cobra.Command {
	var sourcePath string
	var action string
	var listActions bool
	var discover bool
	var formatOutput string

	cmd := &cobra.Command{
		Use:   "navigate",
		Short: "Navigate procedural paths through House Rules",
		Long: `Generate step-by-step procedural guides for legislative actions.

Given a legislative action (e.g., "introduce a bill", "propose an amendment"),
traces the relevant rules to show the procedural steps required.

Examples:
  # Show steps for introducing a bill
  regula navigate --source house-rules-119th.txt --action "introduce a bill"

  # Use short alias for action
  regula navigate --source house-rules-119th.txt --action vote

  # Discover additional related clauses
  regula navigate --source house-rules-119th.txt --action amend --discover

  # List all available actions
  regula navigate --list-actions

  # Output as JSON
  regula navigate --source house-rules-119th.txt --action debate --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle --list-actions (no source required)
			if listActions {
				return outputActionList(formatOutput)
			}

			if sourcePath == "" {
				return fmt.Errorf("--source flag is required")
			}

			if action == "" {
				return fmt.Errorf("--action flag is required (or use --list-actions)")
			}

			// Read the source file
			data, err := os.ReadFile(sourcePath)
			if err != nil {
				return fmt.Errorf("failed to read source file: %w", err)
			}

			// Parse House Rules
			searcher := extract.NewKeywordSearcher()
			searcher.ParseHouseRules(string(data))

			// Create pathfinder
			pathfinder := extract.NewPathfinder(searcher)

			// Navigate
			var path *extract.ProceduralPath
			if discover {
				path, err = pathfinder.NavigateWithDiscovery(action)
			} else {
				path, err = pathfinder.Navigate(action)
			}
			if err != nil {
				return err
			}

			// Output
			return outputProceduralPath(path, formatOutput)
		},
	}

	cmd.Flags().StringVar(&sourcePath, "source", "", "Path to House Rules source file")
	cmd.Flags().StringVar(&action, "action", "", "Legislative action to navigate (e.g., 'introduce-bill', 'vote', 'amend')")
	cmd.Flags().BoolVar(&listActions, "list-actions", false, "List all available procedural actions")
	cmd.Flags().BoolVar(&discover, "discover", false, "Discover additional related clauses via keyword search")
	cmd.Flags().StringVar(&formatOutput, "format", "text", "Output format (text, json)")

	return cmd
}

// outputActionList prints the list of available procedural actions.
func outputActionList(format string) error {
	pathfinder := extract.NewPathfinder(nil)
	scenarios := pathfinder.ListScenarios()

	if format == "json" {
		type actionInfo struct {
			Action      string   `json:"action"`
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Related     []string `json:"related_actions"`
		}
		var actionList []actionInfo
		for _, s := range scenarios {
			actionList = append(actionList, actionInfo{
				Action:      s.Action,
				Title:       s.Title,
				Description: s.Description,
				Related:     s.RelatedActions,
			})
		}
		data, err := json.MarshalIndent(actionList, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	// Text format
	fmt.Println("Available Procedural Actions")
	fmt.Println(strings.Repeat("═", 60))

	for _, s := range scenarios {
		fmt.Printf("\n%s (%s)\n", s.Title, s.Action)
		fmt.Printf("  %s\n", s.Description)
		if len(s.RelatedActions) > 0 {
			fmt.Printf("  Related: %s\n", strings.Join(s.RelatedActions, ", "))
		}
	}

	fmt.Printf("\nTotal: %d actions\n", len(scenarios))
	fmt.Println("\nUsage: regula navigate --source <file> --action <action>")
	return nil
}

// outputProceduralPath prints the procedural path.
func outputProceduralPath(path *extract.ProceduralPath, format string) error {
	if format == "json" {
		type stepJSON struct {
			StepNumber  int      `json:"step_number"`
			Title       string   `json:"title"`
			Rule        string   `json:"rule"`
			Clause      string   `json:"clause"`
			ClauseTitle string   `json:"clause_title,omitempty"`
			Description string   `json:"description"`
			Excerpt     string   `json:"excerpt,omitempty"`
			References  []string `json:"references,omitempty"`
		}
		type pathJSON struct {
			Action         string     `json:"action"`
			Title          string     `json:"title"`
			Description    string     `json:"description"`
			Steps          []stepJSON `json:"steps"`
			RelatedActions []string   `json:"related_actions"`
		}

		output := pathJSON{
			Action:         path.Action,
			Title:          path.Title,
			Description:    path.Description,
			RelatedActions: path.RelatedActions,
		}
		for _, step := range path.Steps {
			output.Steps = append(output.Steps, stepJSON{
				StepNumber:  step.StepNumber,
				Title:       step.Title,
				Rule:        step.Rule,
				Clause:      step.Clause,
				ClauseTitle: step.ClauseTitle,
				Description: step.Description,
				Excerpt:     step.Excerpt,
				References:  step.References,
			})
		}

		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	// Text format
	fmt.Print(path.String())
	return nil
}
