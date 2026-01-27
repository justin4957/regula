package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coolbeans/regula/pkg/extract"
	"github.com/coolbeans/regula/pkg/query"
	"github.com/coolbeans/regula/pkg/store"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

// Global state for the loaded graph
var (
	tripleStore *store.TripleStore
	executor    *query.Executor
	graphLoaded bool
	graphPath   string
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
	rootCmd.AddCommand(simulateCmd())
	rootCmd.AddCommand(auditCmd())

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

			if source == "" {
				return fmt.Errorf("--source flag is required")
			}

			// Check if file exists
			if _, err := os.Stat(source); os.IsNotExist(err) {
				return fmt.Errorf("source file not found: %s", source)
			}

			fmt.Printf("Ingesting regulation from: %s\n", source)
			startTime := time.Now()

			// Step 1: Parse document
			fmt.Print("  1. Parsing document structure... ")
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
			fmt.Printf("done (%d chapters, %d articles)\n", len(doc.Chapters), countArticles(doc))

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

			// Step 5: Build knowledge graph
			fmt.Print("  5. Building knowledge graph... ")
			tripleStore = store.NewTripleStore()
			builder := store.NewGraphBuilder(tripleStore, baseURI)
			stats, err := builder.BuildWithSemantics(doc, defExtractor, refExtractor, semExtractor)
			if err != nil {
				return fmt.Errorf("failed to build graph: %w", err)
			}
			fmt.Printf("done (%d triples)\n", stats.TotalTriples)

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

	return cmd
}

func queryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query [sparql-query]",
		Short: "Query the regulation graph",
		Long: `Execute a SPARQL query against the regulation knowledge graph.

You must first ingest a regulation document using 'regula ingest'.

Examples:
  # Basic query
  regula query "SELECT ?article ?title WHERE { ?article rdf:type reg:Article . ?article reg:title ?title } LIMIT 5"

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

			// Execute query
			startTime := time.Now()
			result, err := executor.ExecuteString(queryStr)
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
	cmd.Flags().StringP("format", "f", "table", "Output format (table, json, csv)")
	cmd.Flags().Bool("timing", false, "Show query execution timing")
	cmd.Flags().StringP("source", "s", "", "Source document to ingest before querying")
	cmd.Flags().Bool("list-templates", false, "List available query templates")

	return cmd
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

	tripleStore = store.NewTripleStore()
	builder := store.NewGraphBuilder(tripleStore, "https://regula.dev/regulations/")

	defExtractor := extract.NewDefinitionExtractor()
	refExtractor := extract.NewReferenceExtractor()
	semExtractor := extract.NewSemanticExtractor()

	_, err = builder.BuildWithSemantics(doc, defExtractor, refExtractor, semExtractor)
	if err != nil {
		return fmt.Errorf("failed to build graph: %w", err)
	}

	executor = query.NewExecutor(tripleStore)
	graphLoaded = true
	graphPath = source
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
		Short: "Validate reference resolution",
		Long: `Validate reference resolution and report statistics.

Checks cross-reference resolution accuracy and reports:
  - Resolution rate for internal references
  - Confidence distribution
  - Unresolved references with reasons

Example:
  regula validate --source gdpr.txt
  regula validate --source gdpr.txt --check references
  regula validate --source gdpr.txt --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			source, _ := cmd.Flags().GetString("source")
			checkType, _ := cmd.Flags().GetString("check")
			formatStr, _ := cmd.Flags().GetString("format")
			baseURI, _ := cmd.Flags().GetString("base-uri")

			if source == "" {
				return fmt.Errorf("--source flag is required")
			}

			// Check if file exists
			if _, err := os.Stat(source); os.IsNotExist(err) {
				return fmt.Errorf("source file not found: %s", source)
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

			// Extract references
			refExtractor := extract.NewReferenceExtractor()
			refs := refExtractor.ExtractFromDocument(doc)

			// Create resolver and index document
			resolver := extract.NewReferenceResolver(baseURI, "GDPR")
			resolver.IndexDocument(doc)

			// Resolve all references
			resolved := resolver.ResolveAll(refs)

			// Generate report
			report := extract.GenerateReport(resolved)

			// Handle different checks
			if checkType == "references" || checkType == "" {
				if formatStr == "json" {
					encoder := json.NewEncoder(os.Stdout)
					encoder.SetIndent("", "  ")
					return encoder.Encode(report)
				}

				// Print text report
				fmt.Println(report.String())

				// Pass/fail based on 85% target
				if report.ResolutionRate >= 0.85 {
					fmt.Printf("Status: PASS (resolution rate %.1f%% >= 85%%)\n", report.ResolutionRate*100)
				} else {
					fmt.Printf("Status: FAIL (resolution rate %.1f%% < 85%%)\n", report.ResolutionRate*100)
					return fmt.Errorf("resolution rate below 85%% target")
				}
			}

			return nil
		},
	}

	cmd.Flags().StringP("source", "s", "", "Source document path")
	cmd.Flags().String("check", "references", "What to check (references)")
	cmd.Flags().StringP("format", "f", "text", "Output format (text, json)")
	cmd.Flags().String("base-uri", "https://regula.dev/regulations/", "Base URI for the graph")

	return cmd
}

func impactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "impact",
		Short: "Analyze impact of regulatory changes",
		Long: `Analyze the impact of changes to a provision.

Example:
  regula impact --provision "GDPR:Art17" --change amend`,
		RunE: func(cmd *cobra.Command, args []string) error {
			provision, _ := cmd.Flags().GetString("provision")
			change, _ := cmd.Flags().GetString("change")
			source, _ := cmd.Flags().GetString("source")

			if provision == "" {
				return fmt.Errorf("--provision flag is required")
			}

			// Load graph if source specified
			if source != "" {
				if err := loadAndIngest(source); err != nil {
					return err
				}
			}

			if !graphLoaded {
				return fmt.Errorf("no graph loaded. Run 'regula ingest --source <file>' first")
			}

			fmt.Printf("Analyzing impact of %s to %s\n", change, provision)

			// Find what references this provision
			queryStr := fmt.Sprintf(`SELECT ?article ?title WHERE {
  ?article reg:references ?target .
  ?article reg:title ?title .
  FILTER(CONTAINS(?target, "%s"))
}`, strings.TrimPrefix(provision, "GDPR:"))

			result, err := executor.ExecuteString(queryStr)
			if err != nil {
				return fmt.Errorf("query error: %w", err)
			}

			fmt.Printf("\nArticles referencing %s: %d\n", provision, result.Count)
			if result.Count > 0 {
				fmt.Println(result.FormatTable())
			}

			fmt.Println("\nNote: Full impact analysis (transitive dependencies, risk assessment) coming in Phase 4")
			return nil
		},
	}

	cmd.Flags().StringP("provision", "p", "", "Provision ID to analyze")
	cmd.Flags().StringP("change", "c", "amend", "Type of change (amend, repeal, add)")
	cmd.Flags().IntP("depth", "d", 3, "Transitive dependency depth")
	cmd.Flags().StringP("source", "s", "", "Source document to analyze")

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
