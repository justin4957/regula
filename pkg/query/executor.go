// Package query provides SPARQL query parsing and execution.
package query

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/coolbeans/regula/pkg/store"
)

// Executor executes SPARQL queries against a triple store.
type Executor struct {
	store          *store.TripleStore
	planner        *QueryPlanner
	enablePlanning bool
	timeout        time.Duration
}

// ExecutorOption configures an executor.
type ExecutorOption func(*Executor)

// WithPlanning enables or disables query planning/optimization.
func WithPlanning(enabled bool) ExecutorOption {
	return func(e *Executor) {
		e.enablePlanning = enabled
	}
}

// WithTimeout sets the query execution timeout.
func WithTimeout(d time.Duration) ExecutorOption {
	return func(e *Executor) {
		e.timeout = d
	}
}

// NewExecutor creates a new query executor.
func NewExecutor(tripleStore *store.TripleStore, opts ...ExecutorOption) *Executor {
	e := &Executor{
		store:          tripleStore,
		planner:        NewQueryPlanner(tripleStore.Stats()),
		enablePlanning: true,
		timeout:        30 * time.Second, // Default 30s timeout
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// RefreshStats updates the query planner with current store statistics.
func (e *Executor) RefreshStats() {
	e.planner = NewQueryPlanner(e.store.Stats())
}

// QueryResult represents the result of a query execution.
type QueryResult struct {
	Variables []string            // Variable names (without ?)
	Bindings  []map[string]string // Variable bindings for each result row
	Count     int                 // Number of result rows
	Metrics   QueryMetrics        // Execution metrics
}

// QueryMetrics contains performance metrics for query execution.
type QueryMetrics struct {
	ParseTime     time.Duration `json:"parse_time"`
	PlanTime      time.Duration `json:"plan_time"`
	ExecuteTime   time.Duration `json:"execute_time"`
	TotalTime     time.Duration `json:"total_time"`
	PatternsCount int           `json:"patterns_count"`
	ResultCount   int           `json:"result_count"`
}

// Execute executes a parsed query.
func (e *Executor) Execute(query *Query) (*QueryResult, error) {
	return e.ExecuteWithContext(context.Background(), query)
}

// ExecuteWithContext executes a parsed query with context for cancellation.
func (e *Executor) ExecuteWithContext(ctx context.Context, query *Query) (*QueryResult, error) {
	startTime := time.Now()
	metrics := QueryMetrics{}

	// Apply timeout if set
	if e.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.timeout)
		defer cancel()
	}

	if query.Type == SelectQueryType {
		result, err := e.executeSelect(ctx, query.Select, &metrics)
		if err != nil {
			return nil, err
		}
		metrics.TotalTime = time.Since(startTime)
		result.Metrics = metrics
		return result, nil
	}

	return nil, fmt.Errorf("unsupported query type: %s", query.Type)
}

// ExecuteString parses and executes a SPARQL query string.
func (e *Executor) ExecuteString(queryStr string) (*QueryResult, error) {
	return e.ExecuteStringWithContext(context.Background(), queryStr)
}

// ExecuteStringWithContext parses and executes a SPARQL query string with context.
func (e *Executor) ExecuteStringWithContext(ctx context.Context, queryStr string) (*QueryResult, error) {
	startTime := time.Now()

	query, err := ParseQuery(queryStr)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	result, err := e.ExecuteWithContext(ctx, query)
	if err != nil {
		return nil, err
	}

	result.Metrics.ParseTime = time.Since(startTime) - result.Metrics.PlanTime - result.Metrics.ExecuteTime
	return result, nil
}

// executeSelect executes a SELECT query.
func (e *Executor) executeSelect(ctx context.Context, query *SelectQuery, metrics *QueryMetrics) (*QueryResult, error) {
	planStart := time.Now()

	// Optimize query if planning is enabled
	optimizedQuery := query
	if e.enablePlanning && len(query.Where) > 1 {
		optimizedQuery = e.planner.OptimizeQuery(query)
	}
	metrics.PlanTime = time.Since(planStart)
	metrics.PatternsCount = len(optimizedQuery.Where)

	executeStart := time.Now()

	// Start with a single empty binding
	bindings := []map[string]string{{}}

	// Process each triple pattern (in optimized order)
	for _, pattern := range optimizedQuery.Where {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		bindings = e.matchPattern(pattern, bindings)
		if len(bindings) == 0 {
			break // No matches, short-circuit
		}
	}

	// Process OPTIONAL patterns
	for _, optPatterns := range query.Optional {
		bindings = e.processOptional(ctx, optPatterns, bindings)
	}

	// Apply filters
	for _, filter := range query.Filters {
		bindings = e.applyFilter(filter, bindings)
	}

	// Apply ORDER BY before DISTINCT (to get consistent ordering)
	if len(query.OrderBy) > 0 {
		bindings = e.applyOrderBy(query.OrderBy, bindings)
	}

	// Apply DISTINCT
	if query.Distinct {
		bindings = e.applyDistinct(bindings, query.Variables)
	}

	// Apply OFFSET
	if query.Offset > 0 {
		if query.Offset < len(bindings) {
			bindings = bindings[query.Offset:]
		} else {
			bindings = []map[string]string{}
		}
	}

	// Apply LIMIT
	if query.Limit > 0 && query.Limit < len(bindings) {
		bindings = bindings[:query.Limit]
	}

	metrics.ExecuteTime = time.Since(executeStart)
	metrics.ResultCount = len(bindings)

	// Build result with projected variables
	result := &QueryResult{
		Bindings: bindings,
		Count:    len(bindings),
	}

	// Determine variables to return
	if len(query.Variables) == 1 && query.Variables[0] == "*" {
		// Return all variables
		varSet := make(map[string]bool)
		for _, binding := range bindings {
			for v := range binding {
				varSet[v] = true
			}
		}
		for v := range varSet {
			result.Variables = append(result.Variables, v)
		}
		sort.Strings(result.Variables)
	} else {
		// Return specified variables (strip ? prefix)
		for _, v := range query.Variables {
			result.Variables = append(result.Variables, StripVariable(v))
		}
	}

	return result, nil
}

// matchPattern matches a triple pattern against the store.
func (e *Executor) matchPattern(pattern TriplePattern, currentBindings []map[string]string) []map[string]string {
	var newBindings []map[string]string

	for _, binding := range currentBindings {
		// Resolve pattern with current bindings
		subject := e.resolveValue(pattern.Subject, binding)
		predicate := e.resolveValue(pattern.Predicate, binding)
		object := e.resolveValue(pattern.Object, binding)

		// Query triple store
		triples := e.store.Find(subject, predicate, object)

		// Create new bindings for each matching triple
		for _, triple := range triples {
			newBinding := make(map[string]string)
			// Copy existing bindings
			for k, v := range binding {
				newBinding[k] = v
			}

			// Add new variable bindings
			if IsVariable(pattern.Subject) {
				varName := StripVariable(pattern.Subject)
				if existing, ok := newBinding[varName]; ok {
					// Variable already bound - check consistency
					if existing != triple.Subject {
						continue // Skip inconsistent binding
					}
				} else {
					newBinding[varName] = triple.Subject
				}
			}
			if IsVariable(pattern.Predicate) {
				varName := StripVariable(pattern.Predicate)
				if existing, ok := newBinding[varName]; ok {
					if existing != triple.Predicate {
						continue
					}
				} else {
					newBinding[varName] = triple.Predicate
				}
			}
			if IsVariable(pattern.Object) {
				varName := StripVariable(pattern.Object)
				if existing, ok := newBinding[varName]; ok {
					if existing != triple.Object {
						continue
					}
				} else {
					newBinding[varName] = triple.Object
				}
			}

			newBindings = append(newBindings, newBinding)
		}
	}

	return newBindings
}

// processOptional processes OPTIONAL patterns (left outer join).
func (e *Executor) processOptional(ctx context.Context, patterns []TriplePattern, currentBindings []map[string]string) []map[string]string {
	var result []map[string]string

	for _, binding := range currentBindings {
		// Try to match optional patterns
		optBindings := []map[string]string{binding}
		for _, pattern := range patterns {
			select {
			case <-ctx.Done():
				return currentBindings // Return original on cancellation
			default:
			}
			optBindings = e.matchPattern(pattern, optBindings)
		}

		if len(optBindings) > 0 {
			// Optional matched - use extended bindings
			result = append(result, optBindings...)
		} else {
			// Optional didn't match - keep original binding
			result = append(result, binding)
		}
	}

	return result
}

// resolveValue resolves a pattern value using variable bindings.
func (e *Executor) resolveValue(value string, binding map[string]string) string {
	if IsVariable(value) {
		// Look up variable in bindings
		if boundValue, ok := binding[StripVariable(value)]; ok {
			return boundValue
		}
		// Unbound variable - use empty string as wildcard
		return ""
	}

	// Strip literal quotes if present
	if IsLiteral(value) {
		return StripLiteral(value)
	}

	// Strip URI brackets if present
	if IsURI(value) {
		return StripURI(value)
	}

	return value
}

// applyFilter applies a FILTER clause to bindings.
func (e *Executor) applyFilter(filter Filter, bindings []map[string]string) []map[string]string {
	var filtered []map[string]string

	for _, binding := range bindings {
		if e.evaluateFilter(filter.Expression, binding) {
			filtered = append(filtered, binding)
		}
	}

	return filtered
}

// evaluateFilter evaluates a filter expression.
func (e *Executor) evaluateFilter(expression string, binding map[string]string) bool {
	// Replace variables with their values
	expr := expression
	for varName, value := range binding {
		expr = strings.ReplaceAll(expr, "?"+varName, `"`+value+`"`)
	}

	// Handle STR() function - just extracts the string value
	strPattern := regexp.MustCompile(`STR\s*\(\s*"([^"]+)"\s*\)`)
	expr = strPattern.ReplaceAllString(expr, `"$1"`)

	// REGEX filter: REGEX("value", "pattern") or REGEX(?var, "pattern")
	regexPattern := regexp.MustCompile(`(?i)REGEX\s*\(\s*"([^"]+)"\s*,\s*"([^"]+)"\s*\)`)
	if match := regexPattern.FindStringSubmatch(expr); match != nil {
		value := match[1]
		pattern := match[2]
		matched, _ := regexp.MatchString(pattern, value)
		return matched
	}

	// CONTAINS: CONTAINS("value", "substring")
	containsPattern := regexp.MustCompile(`(?i)CONTAINS\s*\(\s*"([^"]+)"\s*,\s*"([^"]+)"\s*\)`)
	if match := containsPattern.FindStringSubmatch(expr); match != nil {
		value := match[1]
		substring := match[2]
		return strings.Contains(value, substring)
	}

	// STRSTARTS: STRSTARTS("value", "prefix")
	strstartsPattern := regexp.MustCompile(`(?i)STRSTARTS\s*\(\s*"([^"]+)"\s*,\s*"([^"]+)"\s*\)`)
	if match := strstartsPattern.FindStringSubmatch(expr); match != nil {
		value := match[1]
		prefix := match[2]
		return strings.HasPrefix(value, prefix)
	}

	// STRENDS: STRENDS("value", "suffix")
	strendsPattern := regexp.MustCompile(`(?i)STRENDS\s*\(\s*"([^"]+)"\s*,\s*"([^"]+)"\s*\)`)
	if match := strendsPattern.FindStringSubmatch(expr); match != nil {
		value := match[1]
		suffix := match[2]
		return strings.HasSuffix(value, suffix)
	}

	// Numeric comparison: "value" > number, "value" < number, etc.
	numComparePattern := regexp.MustCompile(`"([^"]+)"\s*(>|<|>=|<=|=|!=)\s*(\d+)`)
	if match := numComparePattern.FindStringSubmatch(expr); match != nil {
		value, err := strconv.Atoi(match[1])
		if err != nil {
			return false // Non-numeric comparison fails
		}
		threshold, _ := strconv.Atoi(match[3])
		switch match[2] {
		case ">":
			return value > threshold
		case "<":
			return value < threshold
		case ">=":
			return value >= threshold
		case "<=":
			return value <= threshold
		case "=":
			return value == threshold
		case "!=":
			return value != threshold
		}
	}

	// String equality: "value1" = "value2"
	eqPattern := regexp.MustCompile(`"([^"]+)"\s*=\s*"([^"]+)"`)
	if match := eqPattern.FindStringSubmatch(expr); match != nil {
		return match[1] == match[2]
	}

	// String inequality: "value1" != "value2"
	neqPattern := regexp.MustCompile(`"([^"]+)"\s*!=\s*"([^"]+)"`)
	if match := neqPattern.FindStringSubmatch(expr); match != nil {
		return match[1] != match[2]
	}

	// BOUND: BOUND(?var) - check if variable is bound
	boundPattern := regexp.MustCompile(`(?i)BOUND\s*\(\s*\?(\w+)\s*\)`)
	if match := boundPattern.FindStringSubmatch(expression); match != nil {
		_, ok := binding[match[1]]
		return ok
	}

	// !BOUND: !BOUND(?var) - check if variable is NOT bound
	notBoundPattern := regexp.MustCompile(`(?i)!\s*BOUND\s*\(\s*\?(\w+)\s*\)`)
	if match := notBoundPattern.FindStringSubmatch(expression); match != nil {
		_, ok := binding[match[1]]
		return !ok
	}

	// Default: assume true if we can't parse
	return true
}

// applyOrderBy sorts bindings by variables.
func (e *Executor) applyOrderBy(orderBys []OrderBy, bindings []map[string]string) []map[string]string {
	if len(orderBys) == 0 {
		return bindings
	}

	sort.SliceStable(bindings, func(i, j int) bool {
		for _, ob := range orderBys {
			varName := StripVariable(ob.Variable)
			valI := bindings[i][varName]
			valJ := bindings[j][varName]

			if valI == valJ {
				continue // Try next sort key
			}

			if ob.Descending {
				return valI > valJ
			}
			return valI < valJ
		}
		return false
	})

	return bindings
}

// applyDistinct removes duplicate bindings based on selected variables.
func (e *Executor) applyDistinct(bindings []map[string]string, variables []string) []map[string]string {
	seen := make(map[string]bool)
	var unique []map[string]string

	for _, binding := range bindings {
		// Create key from relevant variable values
		var key string
		if len(variables) == 1 && variables[0] == "*" {
			// All variables
			var keys []string
			for k, v := range binding {
				keys = append(keys, k+"="+v)
			}
			sort.Strings(keys)
			key = strings.Join(keys, "|")
		} else {
			// Specified variables
			var values []string
			for _, v := range variables {
				varName := StripVariable(v)
				values = append(values, binding[varName])
			}
			key = strings.Join(values, "|")
		}

		if !seen[key] {
			seen[key] = true
			unique = append(unique, binding)
		}
	}

	return unique
}

// Output format types.
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatCSV   OutputFormat = "csv"
)

// Format formats the query result in the specified format.
func (r *QueryResult) Format(format OutputFormat) (string, error) {
	switch format {
	case FormatJSON:
		return r.FormatJSON()
	case FormatCSV:
		return r.FormatCSV()
	case FormatTable:
		return r.FormatTable(), nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// FormatTable formats the result as an ASCII table.
func (r *QueryResult) FormatTable() string {
	if len(r.Variables) == 0 || len(r.Bindings) == 0 {
		return fmt.Sprintf("No results (%d rows)\n", r.Count)
	}

	var sb strings.Builder

	// Calculate column widths
	widths := make([]int, len(r.Variables))
	for i, v := range r.Variables {
		widths[i] = len(v)
	}
	for _, binding := range r.Bindings {
		for i, v := range r.Variables {
			if len(binding[v]) > widths[i] {
				widths[i] = len(binding[v])
			}
		}
	}

	// Header separator
	var sep strings.Builder
	sep.WriteString("+")
	for _, w := range widths {
		sep.WriteString(strings.Repeat("-", w+2))
		sep.WriteString("+")
	}
	sep.WriteString("\n")

	sb.WriteString(sep.String())

	// Header row
	sb.WriteString("|")
	for i, v := range r.Variables {
		sb.WriteString(fmt.Sprintf(" %-*s |", widths[i], v))
	}
	sb.WriteString("\n")
	sb.WriteString(sep.String())

	// Data rows
	for _, binding := range r.Bindings {
		sb.WriteString("|")
		for i, v := range r.Variables {
			sb.WriteString(fmt.Sprintf(" %-*s |", widths[i], binding[v]))
		}
		sb.WriteString("\n")
	}
	sb.WriteString(sep.String())

	sb.WriteString(fmt.Sprintf("%d rows\n", r.Count))
	return sb.String()
}

// FormatJSON formats the result as JSON.
func (r *QueryResult) FormatJSON() (string, error) {
	type jsonResult struct {
		Variables []string            `json:"variables"`
		Bindings  []map[string]string `json:"bindings"`
		Count     int                 `json:"count"`
	}

	result := jsonResult{
		Variables: r.Variables,
		Bindings:  r.Bindings,
		Count:     r.Count,
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FormatCSV formats the result as CSV.
func (r *QueryResult) FormatCSV() (string, error) {
	var sb strings.Builder
	writer := csv.NewWriter(&sb)

	// Header row
	if err := writer.Write(r.Variables); err != nil {
		return "", err
	}

	// Data rows
	for _, binding := range r.Bindings {
		row := make([]string, len(r.Variables))
		for i, v := range r.Variables {
			row[i] = binding[v]
		}
		if err := writer.Write(row); err != nil {
			return "", err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}

	return sb.String(), nil
}

// QueryPlanner optimizes SPARQL query execution using index statistics.
type QueryPlanner struct {
	stats store.IndexStats
}

// NewQueryPlanner creates a new query planner with index statistics.
func NewQueryPlanner(stats store.IndexStats) *QueryPlanner {
	return &QueryPlanner{
		stats: stats,
	}
}

// OptimizeQuery reorders triple patterns for optimal execution.
func (qp *QueryPlanner) OptimizeQuery(query *SelectQuery) *SelectQuery {
	if len(query.Where) <= 1 {
		return query
	}

	// Create a copy to avoid modifying original
	optimized := &SelectQuery{
		Variables: query.Variables,
		Distinct:  query.Distinct,
		Where:     make([]TriplePattern, len(query.Where)),
		Optional:  query.Optional,
		Filters:   query.Filters,
		OrderBy:   query.OrderBy,
		Limit:     query.Limit,
		Offset:    query.Offset,
		Prefixes:  query.Prefixes,
	}
	copy(optimized.Where, query.Where)

	// Calculate selectivity for each pattern
	type patternWithSelectivity struct {
		pattern     TriplePattern
		selectivity float64
	}

	selectivities := make([]patternWithSelectivity, len(optimized.Where))
	for i, pattern := range optimized.Where {
		selectivities[i] = patternWithSelectivity{
			pattern:     pattern,
			selectivity: qp.estimateSelectivity(pattern),
		}
	}

	// Sort patterns by selectivity (most selective first = lowest value)
	sort.SliceStable(selectivities, func(i, j int) bool {
		return selectivities[i].selectivity < selectivities[j].selectivity
	})

	// Apply optimized order
	for i, sel := range selectivities {
		optimized.Where[i] = sel.pattern
	}

	return optimized
}

// estimateSelectivity estimates the selectivity of a triple pattern.
// Lower values = more selective (fewer results expected).
func (qp *QueryPlanner) estimateSelectivity(pattern TriplePattern) float64 {
	if qp.stats.TotalTriples == 0 {
		return 1.0
	}

	selectivity := float64(qp.stats.TotalTriples)
	boundCount := 0

	// Adjust based on what's bound (not a variable)
	if !IsVariable(pattern.Subject) {
		boundCount++
		if count, ok := qp.stats.SubjectCounts[pattern.Subject]; ok {
			selectivity = float64(count)
		} else {
			selectivity = 0.1 // Unknown subject is very selective
		}
	}

	if !IsVariable(pattern.Predicate) {
		boundCount++
		if count, ok := qp.stats.PredicateCounts[pattern.Predicate]; ok {
			if boundCount == 1 {
				selectivity = float64(count)
			} else {
				selectivity *= float64(count) / float64(qp.stats.TotalTriples)
			}
		} else {
			selectivity *= 0.1
		}
	}

	if !IsVariable(pattern.Object) {
		boundCount++
		if count, ok := qp.stats.ObjectCounts[pattern.Object]; ok {
			if boundCount == 1 {
				selectivity = float64(count)
			} else {
				selectivity *= float64(count) / float64(qp.stats.TotalTriples)
			}
		} else {
			selectivity *= 0.1
		}
	}

	// Ensure minimum selectivity to avoid division issues
	if selectivity < 0.1 {
		selectivity = 0.1
	}

	return selectivity
}
