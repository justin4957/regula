package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coolbeans/regula/pkg/analysis"
	"github.com/coolbeans/regula/pkg/eurlex"
	"github.com/coolbeans/regula/pkg/extract"
	"github.com/coolbeans/regula/pkg/fetch"
	"github.com/coolbeans/regula/pkg/library"
	"github.com/coolbeans/regula/pkg/linkcheck"
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
			parser := extract.NewParser()
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
			resolver := extract.NewReferenceResolver(baseURI, "GDPR")
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

	parser := extract.NewParser()
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
	resolver := extract.NewReferenceResolver(baseURI, "GDPR")
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

			parser := extract.NewParser()
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
			resolver := extract.NewReferenceResolver(baseURI, "GDPR")
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

			parser := extract.NewParser()
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
			resolver := extract.NewReferenceResolver(baseURI, "GDPR")
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
		Short: "Compare multiple regulation documents",
		Long: `Compare two or more regulation documents to find shared definitions,
rights, obligations, and external reference targets.

Outputs structural comparison, concept overlaps, and external reference analysis.

Example:
  regula compare --sources testdata/gdpr.txt,testdata/ccpa.txt
  regula compare --sources testdata/gdpr.txt,testdata/ccpa.txt --format json
  regula compare --sources testdata/gdpr.txt,testdata/ccpa.txt,testdata/eu-ai-act.txt --format dot --output comparison.dot`,
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

				parser := extract.NewParser()
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

	return cmd
}

func refsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refs",
		Short: "Analyze references in a regulation document",
		Long: `Analyze internal and external references in a regulation document.

Shows clustered external references, reference frequency, and per-provision details.

Example:
  regula refs --source testdata/gdpr.txt
  regula refs --source testdata/gdpr.txt --format json
  regula refs --source testdata/eu-ai-act.txt --external-only`,
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

			parser := extract.NewParser()
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
	cmd.Flags().StringP("format", "f", "table", "Output format (table, json)")
	cmd.Flags().StringP("output", "o", "", "Output file path")
	cmd.Flags().Bool("external-only", false, "Show only external references")

	return cmd
}

// extractDocID extracts a document identifier from a file path.
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
