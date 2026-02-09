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

// ConstructResult represents the result of a CONSTRUCT query execution.
type ConstructResult struct {
	Triples []ConstructedTriple // Constructed triples
	Count   int                 // Number of triples
	Metrics QueryMetrics        // Execution metrics
}

// ConstructedTriple represents a triple produced by a CONSTRUCT query.
type ConstructedTriple struct {
	Subject   string
	Predicate string
	Object    string
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

// Execute executes a parsed SELECT query.
func (e *Executor) Execute(query *Query) (*QueryResult, error) {
	return e.ExecuteWithContext(context.Background(), query)
}

// ExecuteWithContext executes a parsed SELECT query with context for cancellation.
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

	return nil, fmt.Errorf("unsupported query type for Execute: %s (use ExecuteConstruct for CONSTRUCT queries)", query.Type)
}

// ExecuteConstruct executes a parsed CONSTRUCT query.
func (e *Executor) ExecuteConstruct(query *Query) (*ConstructResult, error) {
	return e.ExecuteConstructWithContext(context.Background(), query)
}

// ExecuteConstructWithContext executes a parsed CONSTRUCT query with context for cancellation.
func (e *Executor) ExecuteConstructWithContext(ctx context.Context, query *Query) (*ConstructResult, error) {
	startTime := time.Now()
	metrics := QueryMetrics{}

	// Apply timeout if set
	if e.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.timeout)
		defer cancel()
	}

	if query.Type != ConstructQueryType {
		return nil, fmt.Errorf("expected CONSTRUCT query, got: %s", query.Type)
	}

	result, err := e.executeConstruct(ctx, query.Construct, &metrics)
	if err != nil {
		return nil, err
	}
	metrics.TotalTime = time.Since(startTime)
	result.Metrics = metrics
	return result, nil
}

// ExecuteString parses and executes a SPARQL SELECT query string.
func (e *Executor) ExecuteString(queryStr string) (*QueryResult, error) {
	return e.ExecuteStringWithContext(context.Background(), queryStr)
}

// ExecuteStringWithContext parses and executes a SPARQL SELECT query string with context.
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

// ExecuteDescribe executes a parsed DESCRIBE query.
func (e *Executor) ExecuteDescribe(query *Query) (*ConstructResult, error) {
	return e.ExecuteDescribeWithContext(context.Background(), query)
}

// ExecuteDescribeWithContext executes a parsed DESCRIBE query with context for cancellation.
func (e *Executor) ExecuteDescribeWithContext(ctx context.Context, query *Query) (*ConstructResult, error) {
	startTime := time.Now()
	metrics := QueryMetrics{}

	// Apply timeout if set
	if e.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.timeout)
		defer cancel()
	}

	if query.Type != DescribeQueryType {
		return nil, fmt.Errorf("expected DESCRIBE query, got: %s", query.Type)
	}

	result, err := e.executeDescribe(ctx, query.Describe, &metrics)
	if err != nil {
		return nil, err
	}
	metrics.TotalTime = time.Since(startTime)
	result.Metrics = metrics
	return result, nil
}

// ExecuteDescribeString parses and executes a SPARQL DESCRIBE query string.
func (e *Executor) ExecuteDescribeString(queryStr string) (*ConstructResult, error) {
	return e.ExecuteDescribeStringWithContext(context.Background(), queryStr)
}

// ExecuteDescribeStringWithContext parses and executes a SPARQL DESCRIBE query string with context.
func (e *Executor) ExecuteDescribeStringWithContext(ctx context.Context, queryStr string) (*ConstructResult, error) {
	startTime := time.Now()

	query, err := ParseQuery(queryStr)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	result, err := e.ExecuteDescribeWithContext(ctx, query)
	if err != nil {
		return nil, err
	}

	result.Metrics.ParseTime = time.Since(startTime) - result.Metrics.PlanTime - result.Metrics.ExecuteTime
	return result, nil
}

// ExecuteConstructString parses and executes a SPARQL CONSTRUCT query string.
func (e *Executor) ExecuteConstructString(queryStr string) (*ConstructResult, error) {
	return e.ExecuteConstructStringWithContext(context.Background(), queryStr)
}

// ExecuteConstructStringWithContext parses and executes a SPARQL CONSTRUCT query string with context.
func (e *Executor) ExecuteConstructStringWithContext(ctx context.Context, queryStr string) (*ConstructResult, error) {
	startTime := time.Now()

	query, err := ParseQuery(queryStr)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	result, err := e.ExecuteConstructWithContext(ctx, query)
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

	// Branch: aggregate queries take a separate execution path
	if query.HasAggregates() {
		return e.executeAggregateSelect(ctx, query, bindings, metrics, executeStart)
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

// executeAggregateSelect handles SELECT queries with aggregate functions (COUNT, SUM, etc.).
func (e *Executor) executeAggregateSelect(ctx context.Context, query *SelectQuery, bindings []map[string]string, metrics *QueryMetrics, executeStart time.Time) (*QueryResult, error) {
	// Step 1: Group bindings by GROUP BY variables
	groups := e.groupBindings(query.GroupBy, bindings)

	// Step 2: Compute aggregates for each group
	aggregatedBindings := e.computeAggregates(query, groups)

	// Step 3: Apply HAVING filters
	for _, havingFilter := range query.Having {
		aggregatedBindings = e.applyHavingFilter(query, havingFilter, aggregatedBindings)
	}

	// Step 4: Apply ORDER BY with numeric awareness for aggregate aliases
	if len(query.OrderBy) > 0 {
		aggregatedBindings = e.applyAggregateOrderBy(query, aggregatedBindings)
	}

	// Step 5: Apply OFFSET
	if query.Offset > 0 {
		if query.Offset < len(aggregatedBindings) {
			aggregatedBindings = aggregatedBindings[query.Offset:]
		} else {
			aggregatedBindings = []map[string]string{}
		}
	}

	// Step 6: Apply LIMIT
	if query.Limit > 0 && query.Limit < len(aggregatedBindings) {
		aggregatedBindings = aggregatedBindings[:query.Limit]
	}

	metrics.ExecuteTime = time.Since(executeStart)
	metrics.ResultCount = len(aggregatedBindings)

	// Build result with output variables (GROUP BY vars + aggregate aliases)
	result := &QueryResult{
		Bindings: aggregatedBindings,
		Count:    len(aggregatedBindings),
	}

	// Output variables: GROUP BY variables first, then aggregate aliases
	for _, v := range query.Variables {
		result.Variables = append(result.Variables, StripVariable(v))
	}
	for _, agg := range query.Aggregates {
		result.Variables = append(result.Variables, StripVariable(agg.Alias))
	}

	return result, nil
}

// groupBindings partitions bindings into groups based on GROUP BY variables.
// Returns a map from group key to the bindings in that group.
// If no GROUP BY variables, all bindings form a single group.
func (e *Executor) groupBindings(groupByVars []string, bindings []map[string]string) map[string][]map[string]string {
	groups := make(map[string][]map[string]string)

	if len(groupByVars) == 0 {
		// No GROUP BY — all bindings form a single group
		groups[""] = bindings
		return groups
	}

	for _, binding := range bindings {
		// Build group key from GROUP BY variable values
		var keyParts []string
		for _, groupVar := range groupByVars {
			varName := StripVariable(groupVar)
			keyParts = append(keyParts, binding[varName])
		}
		groupKey := strings.Join(keyParts, "\x00")
		groups[groupKey] = append(groups[groupKey], binding)
	}

	return groups
}

// computeAggregates computes aggregate values for each group and produces output bindings.
func (e *Executor) computeAggregates(query *SelectQuery, groups map[string][]map[string]string) []map[string]string {
	var resultBindings []map[string]string

	for _, groupBindings := range groups {
		if len(groupBindings) == 0 {
			continue
		}

		outputBinding := make(map[string]string)

		// Copy GROUP BY variable values from the first binding in the group
		for _, groupVar := range query.GroupBy {
			varName := StripVariable(groupVar)
			outputBinding[varName] = groupBindings[0][varName]
		}

		// Compute each aggregate function
		for _, agg := range query.Aggregates {
			aliasName := StripVariable(agg.Alias)
			sourceVarName := StripVariable(agg.Variable)

			switch agg.Function {
			case AggregateCOUNT:
				outputBinding[aliasName] = computeCount(groupBindings, sourceVarName, agg.Distinct)
			case AggregateSUM:
				outputBinding[aliasName] = computeSum(groupBindings, sourceVarName)
			case AggregateAVG:
				outputBinding[aliasName] = computeAvg(groupBindings, sourceVarName)
			case AggregateMIN:
				outputBinding[aliasName] = computeMin(groupBindings, sourceVarName)
			case AggregateMAX:
				outputBinding[aliasName] = computeMax(groupBindings, sourceVarName)
			}
		}

		resultBindings = append(resultBindings, outputBinding)
	}

	return resultBindings
}

// computeCount counts non-empty values; with distinct, counts unique values.
func computeCount(bindings []map[string]string, varName string, distinct bool) string {
	if distinct {
		uniqueValues := make(map[string]bool)
		for _, binding := range bindings {
			if val, ok := binding[varName]; ok && val != "" {
				uniqueValues[val] = true
			}
		}
		return strconv.Itoa(len(uniqueValues))
	}

	count := 0
	for _, binding := range bindings {
		if val, ok := binding[varName]; ok && val != "" {
			count++
		}
	}
	return strconv.Itoa(count)
}

// computeSum sums numeric values, returning an integer string if no fraction.
func computeSum(bindings []map[string]string, varName string) string {
	sum := 0.0
	hasDecimal := false
	for _, binding := range bindings {
		if val, ok := binding[varName]; ok && val != "" {
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				sum += f
				if strings.Contains(val, ".") {
					hasDecimal = true
				}
			}
		}
	}
	if !hasDecimal && sum == float64(int64(sum)) {
		return strconv.FormatInt(int64(sum), 10)
	}
	return strconv.FormatFloat(sum, 'f', -1, 64)
}

// computeAvg computes the average of numeric values.
func computeAvg(bindings []map[string]string, varName string) string {
	sum := 0.0
	count := 0
	for _, binding := range bindings {
		if val, ok := binding[varName]; ok && val != "" {
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				sum += f
				count++
			}
		}
	}
	if count == 0 {
		return "0"
	}
	avg := sum / float64(count)
	if avg == float64(int64(avg)) {
		return strconv.FormatInt(int64(avg), 10)
	}
	return strconv.FormatFloat(avg, 'f', -1, 64)
}

// computeMin finds the minimum value, using numeric comparison with string fallback.
func computeMin(bindings []map[string]string, varName string) string {
	var minVal string
	first := true
	for _, binding := range bindings {
		if val, ok := binding[varName]; ok && val != "" {
			if first {
				minVal = val
				first = false
				continue
			}
			if compareValues(val, minVal) < 0 {
				minVal = val
			}
		}
	}
	return minVal
}

// computeMax finds the maximum value, using numeric comparison with string fallback.
func computeMax(bindings []map[string]string, varName string) string {
	var maxVal string
	first := true
	for _, binding := range bindings {
		if val, ok := binding[varName]; ok && val != "" {
			if first {
				maxVal = val
				first = false
				continue
			}
			if compareValues(val, maxVal) > 0 {
				maxVal = val
			}
		}
	}
	return maxVal
}

// compareValues compares two values numerically if possible, falling back to string comparison.
func compareValues(a, b string) int {
	aFloat, aErr := strconv.ParseFloat(a, 64)
	bFloat, bErr := strconv.ParseFloat(b, 64)
	if aErr == nil && bErr == nil {
		if aFloat < bFloat {
			return -1
		}
		if aFloat > bFloat {
			return 1
		}
		return 0
	}
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// applyHavingFilter evaluates a HAVING clause by substituting aggregate function calls
// with computed values and evaluating the resulting numeric comparison.
func (e *Executor) applyHavingFilter(query *SelectQuery, havingFilter Filter, bindings []map[string]string) []map[string]string {
	var filtered []map[string]string

	// Build a map from aggregate function call to alias for substitution
	// e.g., "COUNT(?article)" -> alias name from the aggregate
	aggAliasMap := make(map[string]string)
	for _, agg := range query.Aggregates {
		callStr := strings.ToUpper(string(agg.Function)) + "(" + agg.Variable + ")"
		aggAliasMap[callStr] = StripVariable(agg.Alias)
	}

	for _, binding := range bindings {
		expr := havingFilter.Expression

		// Substitute aggregate function calls with their computed values
		for callStr, aliasName := range aggAliasMap {
			if val, ok := binding[aliasName]; ok {
				expr = strings.ReplaceAll(expr, callStr, val)
				// Also try case-insensitive match
				expr = regexp.MustCompile(`(?i)`+regexp.QuoteMeta(callStr)).ReplaceAllString(expr, val)
			}
		}

		// Evaluate as numeric comparison
		if evaluateHavingExpression(expr) {
			filtered = append(filtered, binding)
		}
	}

	return filtered
}

// evaluateHavingExpression evaluates a simple numeric comparison expression
// like "3 > 1" or "10 >= 5".
func evaluateHavingExpression(expr string) bool {
	expr = strings.TrimSpace(expr)

	comparisonRegex := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*(>|<|>=|<=|=|!=)\s*(\d+(?:\.\d+)?)$`)
	match := comparisonRegex.FindStringSubmatch(expr)
	if match == nil {
		return true // Can't parse — default to true
	}

	leftVal, _ := strconv.ParseFloat(match[1], 64)
	operator := match[2]
	rightVal, _ := strconv.ParseFloat(match[3], 64)

	switch operator {
	case ">":
		return leftVal > rightVal
	case "<":
		return leftVal < rightVal
	case ">=":
		return leftVal >= rightVal
	case "<=":
		return leftVal <= rightVal
	case "=":
		return leftVal == rightVal
	case "!=":
		return leftVal != rightVal
	}

	return true
}

// applyAggregateOrderBy sorts aggregate result bindings with numeric awareness
// for aggregate alias columns.
func (e *Executor) applyAggregateOrderBy(query *SelectQuery, bindings []map[string]string) []map[string]string {
	if len(query.OrderBy) == 0 {
		return bindings
	}

	// Build set of aggregate alias names for numeric sorting
	aggregateAliases := make(map[string]bool)
	for _, agg := range query.Aggregates {
		aggregateAliases[StripVariable(agg.Alias)] = true
	}

	sort.SliceStable(bindings, func(i, j int) bool {
		for _, ob := range query.OrderBy {
			varName := StripVariable(ob.Variable)
			valI := bindings[i][varName]
			valJ := bindings[j][varName]

			if valI == valJ {
				continue
			}

			// Use numeric comparison for aggregate aliases
			if aggregateAliases[varName] {
				numI, errI := strconv.ParseFloat(valI, 64)
				numJ, errJ := strconv.ParseFloat(valJ, 64)
				if errI == nil && errJ == nil {
					if ob.Descending {
						return numI > numJ
					}
					return numI < numJ
				}
			}

			// Fallback to lexicographic comparison
			if ob.Descending {
				return valI > valJ
			}
			return valI < valJ
		}
		return false
	})

	return bindings
}

// executeConstruct executes a CONSTRUCT query.
func (e *Executor) executeConstruct(ctx context.Context, query *ConstructQuery, metrics *QueryMetrics) (*ConstructResult, error) {
	planStart := time.Now()
	metrics.PlanTime = time.Since(planStart)
	metrics.PatternsCount = len(query.Where)

	executeStart := time.Now()

	// Start with a single empty binding
	bindings := []map[string]string{{}}

	// Process each triple pattern in WHERE clause
	for _, pattern := range query.Where {
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

	// Construct triples from template using bindings
	seen := make(map[string]bool)
	var triples []ConstructedTriple

	for _, binding := range bindings {
		for _, templatePattern := range query.Template {
			// Substitute variables in template with bound values
			subject := e.substituteVariable(templatePattern.Subject, binding)
			predicate := e.substituteVariable(templatePattern.Predicate, binding)
			object := e.substituteVariable(templatePattern.Object, binding)

			// Skip triples with unbound variables (they would have empty values)
			if subject == "" || predicate == "" || object == "" {
				continue
			}

			// De-duplicate triples
			key := subject + "|" + predicate + "|" + object
			if !seen[key] {
				seen[key] = true
				triples = append(triples, ConstructedTriple{
					Subject:   subject,
					Predicate: predicate,
					Object:    object,
				})
			}
		}
	}

	metrics.ExecuteTime = time.Since(executeStart)
	metrics.ResultCount = len(triples)

	return &ConstructResult{
		Triples: triples,
		Count:   len(triples),
	}, nil
}

// executeDescribe executes a DESCRIBE query by collecting all triples where
// target resources appear as subject or object (bidirectional).
func (e *Executor) executeDescribe(ctx context.Context, query *DescribeQuery, metrics *QueryMetrics) (*ConstructResult, error) {
	planStart := time.Now()
	metrics.PlanTime = time.Since(planStart)
	metrics.PatternsCount = len(query.Where)

	executeStart := time.Now()

	var targetURIs []string

	if len(query.Where) == 0 {
		// Direct URI form: DESCRIBE <uri> or DESCRIBE prefix:name
		for _, resource := range query.Resources {
			resolvedURI := resolveResourceURI(resource)
			if resolvedURI != "" {
				targetURIs = append(targetURIs, resolvedURI)
			}
		}
	} else {
		// Variable form: DESCRIBE ?var WHERE { ... }
		bindings := []map[string]string{{}}

		for _, pattern := range query.Where {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			bindings = e.matchPattern(pattern, bindings)
			if len(bindings) == 0 {
				break
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

		// Extract unique URIs from variable bindings
		seenURIs := make(map[string]bool)
		for _, binding := range bindings {
			for _, resource := range query.Resources {
				if IsVariable(resource) {
					varName := StripVariable(resource)
					if uri, ok := binding[varName]; ok && uri != "" && !seenURIs[uri] {
						seenURIs[uri] = true
						targetURIs = append(targetURIs, uri)
					}
				}
			}
		}
	}

	// Collect all triples bidirectionally for each target URI
	seen := make(map[string]bool)
	var triples []ConstructedTriple

	for _, uri := range targetURIs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Triples where URI is the subject
		subjectTriples := e.store.Find(uri, "", "")
		for _, triple := range subjectTriples {
			tripleKey := triple.Subject + "|" + triple.Predicate + "|" + triple.Object
			if !seen[tripleKey] {
				seen[tripleKey] = true
				triples = append(triples, ConstructedTriple{
					Subject:   triple.Subject,
					Predicate: triple.Predicate,
					Object:    triple.Object,
				})
			}
		}

		// Triples where URI is the object (bidirectional)
		objectTriples := e.store.Find("", "", uri)
		for _, triple := range objectTriples {
			tripleKey := triple.Subject + "|" + triple.Predicate + "|" + triple.Object
			if !seen[tripleKey] {
				seen[tripleKey] = true
				triples = append(triples, ConstructedTriple{
					Subject:   triple.Subject,
					Predicate: triple.Predicate,
					Object:    triple.Object,
				})
			}
		}
	}

	metrics.ExecuteTime = time.Since(executeStart)
	metrics.ResultCount = len(triples)

	return &ConstructResult{
		Triples: triples,
		Count:   len(triples),
	}, nil
}

// resolveResourceURI resolves a resource identifier to a plain URI string.
func resolveResourceURI(resource string) string {
	if IsURI(resource) {
		return StripURI(resource)
	}
	return resource
}

// substituteVariable replaces a variable with its bound value from the binding.
func (e *Executor) substituteVariable(term string, binding map[string]string) string {
	if IsVariable(term) {
		varName := StripVariable(term)
		if value, ok := binding[varName]; ok {
			return value
		}
		return "" // Unbound variable
	}

	// Strip literal quotes if present
	if IsLiteral(term) {
		return StripLiteral(term)
	}

	// Strip URI brackets if present
	if IsURI(term) {
		return StripURI(term)
	}

	return term
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
	FormatTable    OutputFormat = "table"
	FormatJSON     OutputFormat = "json"
	FormatCSV      OutputFormat = "csv"
	FormatTurtle   OutputFormat = "turtle"
	FormatNTriples OutputFormat = "ntriples"
)

// Common URI prefixes that can be compacted in output.
var commonURIPrefixes = []struct {
	prefix string
	short  string
}{
	{"https://regula.dev/regulations/", ""},
	{"https://regula.dev/ontology#", "reg:"},
	{"http://www.w3.org/1999/02/22-rdf-syntax-ns#", "rdf:"},
	{"http://www.w3.org/2000/01/rdf-schema#", "rdfs:"},
	{"http://purl.org/dc/terms/", "dc:"},
	{"http://data.europa.eu/eli/ontology#", "eli:"},
}

// CompactURI shortens a full URI to a more readable compact form.
// For example: "https://regula.dev/regulations/GDPR:Art17" -> "GDPR:Art17"
func CompactURI(uri string) string {
	for _, p := range commonURIPrefixes {
		if strings.HasPrefix(uri, p.prefix) {
			return p.short + strings.TrimPrefix(uri, p.prefix)
		}
	}
	return uri
}

// CompactBindings applies CompactURI to all values in the bindings.
func CompactBindings(bindings []map[string]string) []map[string]string {
	result := make([]map[string]string, len(bindings))
	for i, binding := range bindings {
		newBinding := make(map[string]string)
		for k, v := range binding {
			newBinding[k] = CompactURI(v)
		}
		result[i] = newBinding
	}
	return result
}

// WithCompactURIs returns a copy of the QueryResult with compacted URIs.
func (r *QueryResult) WithCompactURIs() *QueryResult {
	return &QueryResult{
		Variables: r.Variables,
		Bindings:  CompactBindings(r.Bindings),
		Count:     r.Count,
		Metrics:   r.Metrics,
	}
}

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

// Format formats the CONSTRUCT result in the specified format.
func (r *ConstructResult) Format(format OutputFormat) (string, error) {
	switch format {
	case FormatTurtle:
		return r.FormatTurtle(), nil
	case FormatNTriples:
		return r.FormatNTriples(), nil
	case FormatJSON:
		return r.FormatJSON()
	default:
		return "", fmt.Errorf("unsupported format for CONSTRUCT results: %s (use turtle, ntriples, or json)", format)
	}
}

// FormatTurtle formats the constructed triples in Turtle format.
func (r *ConstructResult) FormatTurtle() string {
	if len(r.Triples) == 0 {
		return "# No triples constructed\n"
	}

	var sb strings.Builder
	sb.WriteString("# CONSTRUCT query result\n")
	sb.WriteString(fmt.Sprintf("# %d triple(s)\n\n", r.Count))

	// Group triples by subject for compact Turtle output
	subjectTriples := make(map[string][]ConstructedTriple)
	var subjects []string
	for _, triple := range r.Triples {
		if _, exists := subjectTriples[triple.Subject]; !exists {
			subjects = append(subjects, triple.Subject)
		}
		subjectTriples[triple.Subject] = append(subjectTriples[triple.Subject], triple)
	}

	for i, subject := range subjects {
		triples := subjectTriples[subject]

		// Write subject with first predicate-object
		sb.WriteString(formatTurtleTerm(subject))
		sb.WriteString(" ")
		sb.WriteString(formatTurtleTerm(triples[0].Predicate))
		sb.WriteString(" ")
		sb.WriteString(formatTurtleTerm(triples[0].Object))

		// Write remaining predicate-objects for same subject
		for j := 1; j < len(triples); j++ {
			sb.WriteString(" ;\n    ")
			sb.WriteString(formatTurtleTerm(triples[j].Predicate))
			sb.WriteString(" ")
			sb.WriteString(formatTurtleTerm(triples[j].Object))
		}

		sb.WriteString(" .\n")
		if i < len(subjects)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// FormatNTriples formats the constructed triples in N-Triples format.
func (r *ConstructResult) FormatNTriples() string {
	if len(r.Triples) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, triple := range r.Triples {
		sb.WriteString(formatNTriplesTerm(triple.Subject))
		sb.WriteString(" ")
		sb.WriteString(formatNTriplesTerm(triple.Predicate))
		sb.WriteString(" ")
		sb.WriteString(formatNTriplesTerm(triple.Object))
		sb.WriteString(" .\n")
	}

	return sb.String()
}

// FormatJSON formats the constructed triples as JSON.
func (r *ConstructResult) FormatJSON() (string, error) {
	type jsonTriple struct {
		Subject   string `json:"subject"`
		Predicate string `json:"predicate"`
		Object    string `json:"object"`
	}
	type jsonResult struct {
		Triples []jsonTriple `json:"triples"`
		Count   int          `json:"count"`
	}

	result := jsonResult{
		Triples: make([]jsonTriple, len(r.Triples)),
		Count:   r.Count,
	}

	for i, t := range r.Triples {
		result.Triples[i] = jsonTriple{
			Subject:   t.Subject,
			Predicate: t.Predicate,
			Object:    t.Object,
		}
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// formatTurtleTerm formats a term for Turtle output.
func formatTurtleTerm(term string) string {
	// If it looks like a URI (contains :// or starts with known scheme)
	if strings.Contains(term, "://") || strings.HasPrefix(term, "urn:") {
		return "<" + term + ">"
	}
	// If it looks like a prefixed name, use as-is
	if strings.Contains(term, ":") && !strings.Contains(term, " ") {
		return term
	}
	// Otherwise, treat as a literal
	return `"` + escapeLiteral(term) + `"`
}

// formatNTriplesTerm formats a term for N-Triples output.
func formatNTriplesTerm(term string) string {
	// If it looks like a URI
	if strings.Contains(term, "://") || strings.HasPrefix(term, "urn:") {
		return "<" + term + ">"
	}
	// If it looks like a prefixed name that needs expansion, wrap as URI
	if strings.Contains(term, ":") && !strings.Contains(term, " ") && !strings.HasPrefix(term, "_:") {
		// For N-Triples, prefixed names should ideally be expanded
		// For now, treat as URI-like reference
		return "<" + term + ">"
	}
	// Blank nodes
	if strings.HasPrefix(term, "_:") {
		return term
	}
	// Otherwise, treat as a literal
	return `"` + escapeLiteral(term) + `"`
}

// escapeLiteral escapes special characters in a literal string.
func escapeLiteral(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
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
		Variables:  query.Variables,
		Aggregates: query.Aggregates,
		GroupBy:    query.GroupBy,
		Having:     query.Having,
		Distinct:   query.Distinct,
		Where:      make([]TriplePattern, len(query.Where)),
		Optional:   query.Optional,
		Filters:    query.Filters,
		OrderBy:    query.OrderBy,
		Limit:      query.Limit,
		Offset:     query.Offset,
		Prefixes:   query.Prefixes,
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
