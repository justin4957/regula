package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coolbeans/regula/pkg/analysis"
	"github.com/coolbeans/regula/pkg/extract"
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

	return cmd
}

func queryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query [sparql-query]",
		Short: "Query the regulation graph",
		Long: `Execute a SPARQL query against the regulation knowledge graph.

You must first ingest a regulation document using 'regula ingest'.

Supports both SELECT and CONSTRUCT queries.

Examples:
  # Basic SELECT query
  regula query "SELECT ?article ?title WHERE { ?article rdf:type reg:Article . ?article reg:title ?title } LIMIT 5"

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
	cmd.Flags().StringP("format", "f", "table", "Output format (table, json, csv for SELECT; turtle, ntriples, json for CONSTRUCT)")
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

Example:
  regula validate --source gdpr.txt
  regula validate --source gdpr.txt --threshold 0.85
  regula validate --source gdpr.txt --format json
  regula validate --source gdpr.txt --check references
  regula validate --source ccpa.txt --profile CCPA
  regula validate --source gdpr.txt --check gates
  regula validate --source gdpr.txt --check links
  regula validate --source gdpr.txt --check links --report links.json`,
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

				if formatStr == "json" {
					jsonData, err := gateReport.ToJSON()
					if err != nil {
						return fmt.Errorf("failed to serialize gate report: %w", err)
					}
					fmt.Println(string(jsonData))
				} else {
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

			// Set profile if specified, otherwise auto-detect
			if profileName != "" {
				regType := validate.RegulationType(profileName)
				if profile, ok := validate.ValidationProfiles[regType]; ok {
					validator.SetRegulationType(regType)
					validator.SetProfile(profile)
				} else {
					return fmt.Errorf("unknown validation profile: %s\nAvailable profiles: GDPR, CCPA, Generic", profileName)
				}
			}

			result := validator.Validate(doc, resolved, definitions, usages, annotations, ts)

			// Output result
			if formatStr == "json" {
				data, err := result.ToJSON()
				if err != nil {
					return fmt.Errorf("failed to serialize result: %w", err)
				}
				fmt.Println(string(data))
			} else {
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
	cmd.Flags().StringP("format", "f", "text", "Output format (text, json)")
	cmd.Flags().String("base-uri", "https://regula.dev/regulations/", "Base URI for the graph")
	cmd.Flags().Float64("threshold", 0.80, "Pass/fail threshold (0.0-1.0)")
	cmd.Flags().String("profile", "", "Validation profile (GDPR, CCPA, Generic) - auto-detected if not specified")
	cmd.Flags().StringSlice("skip-gates", []string{}, "Gates to skip (V0,V1,V2,V3)")
	cmd.Flags().Bool("strict", false, "Halt pipeline on gate failure")
	cmd.Flags().Bool("fail-on-warn", false, "Halt pipeline on gate warnings")
	cmd.Flags().String("report", "", "Save link validation report to file (JSON or Markdown based on extension)")

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

			default:
				return fmt.Errorf("unknown format: %s (use json, dot, turtle, jsonld, or summary)", formatStr)
			}

			return nil
		},
	}

	cmd.Flags().StringP("source", "s", "", "Source document path")
	cmd.Flags().StringP("format", "f", "summary", "Output format (json, dot, turtle, jsonld, summary)")
	cmd.Flags().StringP("output", "o", "", "Output file path")
	cmd.Flags().Bool("relations-only", true, "Export only relationship edges (default: true)")
	cmd.Flags().Bool("eli", false, "Enrich with ELI (European Legislation Identifier) vocabulary for EU documents")
	cmd.Flags().Bool("expanded", false, "Output expanded JSON-LD (full URIs, no @context) instead of compact form")

	return cmd
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
