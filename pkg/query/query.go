// Package query provides SPARQL query parsing and data structures.
package query

// Query represents a parsed SPARQL query.
type Query struct {
	Type      QueryType
	Select    *SelectQuery
	Construct *ConstructQuery
	Describe  *DescribeQuery
}

// QueryType represents the type of SPARQL query.
type QueryType string

const (
	// SelectQueryType represents a SELECT query.
	SelectQueryType QueryType = "SELECT"
	// ConstructQueryType represents a CONSTRUCT query.
	ConstructQueryType QueryType = "CONSTRUCT"
	// DescribeQueryType represents a DESCRIBE query.
	DescribeQueryType QueryType = "DESCRIBE"
)

// AggregateFunction represents a SPARQL aggregate function.
type AggregateFunction string

const (
	AggregateCOUNT AggregateFunction = "COUNT"
	AggregateSUM   AggregateFunction = "SUM"
	AggregateAVG   AggregateFunction = "AVG"
	AggregateMIN   AggregateFunction = "MIN"
	AggregateMAX   AggregateFunction = "MAX"
)

// AggregateExpression represents a parsed aggregate expression like (COUNT(?x) AS ?count).
type AggregateExpression struct {
	Function AggregateFunction // COUNT, SUM, AVG, MIN, MAX
	Variable string            // Source variable (e.g., "?x")
	Alias    string            // Result alias (e.g., "?count")
	Distinct bool              // COUNT(DISTINCT ?x)
}

// SelectQuery represents a parsed SELECT query.
type SelectQuery struct {
	Variables  []string              // Variables to select (e.g., ["?subject", "?predicate"])
	Aggregates []AggregateExpression // Aggregate expressions (e.g., COUNT(?x) AS ?count)
	GroupBy    []string              // GROUP BY variables (e.g., ["?chapter"])
	Having     []Filter              // HAVING clauses (post-aggregation filters)
	Distinct   bool                  // DISTINCT modifier
	Where      []TriplePattern       // WHERE clause triple patterns
	Optional   [][]TriplePattern     // OPTIONAL clause patterns
	Filters    []Filter              // FILTER clauses
	OrderBy    []OrderBy             // ORDER BY clauses
	Limit      int                   // LIMIT (0 = no limit)
	Offset     int                   // OFFSET (0 = no offset)
	Prefixes   map[string]string     // Prefix declarations
}

// HasAggregates returns true if the query uses aggregate functions.
func (q *SelectQuery) HasAggregates() bool {
	return len(q.Aggregates) > 0
}

// AllOutputVariables returns all variables that appear in the query output,
// including both plain SELECT variables and aggregate aliases.
func (q *SelectQuery) AllOutputVariables() []string {
	var outputVars []string
	outputVars = append(outputVars, q.Variables...)
	for _, agg := range q.Aggregates {
		outputVars = append(outputVars, agg.Alias)
	}
	return outputVars
}

// IsAggregateAlias checks if a variable is an alias for an aggregate expression.
func (q *SelectQuery) IsAggregateAlias(variable string) bool {
	for _, agg := range q.Aggregates {
		if agg.Alias == variable {
			return true
		}
	}
	return false
}

// ConstructQuery represents a parsed CONSTRUCT query.
type ConstructQuery struct {
	Template []TriplePattern   // CONSTRUCT template patterns
	Where    []TriplePattern   // WHERE clause triple patterns
	Optional [][]TriplePattern // OPTIONAL clause patterns
	Filters  []Filter          // FILTER clauses
	Prefixes map[string]string // Prefix declarations
}

// DescribeQuery represents a parsed DESCRIBE query.
type DescribeQuery struct {
	Resources []string          // URIs, variables, or prefixed names to describe
	Where     []TriplePattern   // WHERE clause triple patterns (optional, for variable form)
	Optional  [][]TriplePattern // OPTIONAL clause patterns
	Filters   []Filter          // FILTER clauses
	Prefixes  map[string]string // Prefix declarations
}

// TriplePattern represents a triple pattern in a WHERE clause.
type TriplePattern struct {
	Subject   string // Can be variable (?var), URI (<uri>), or prefixed (reg:Article)
	Predicate string
	Object    string
}

// Filter represents a FILTER clause.
type Filter struct {
	Expression string // Filter expression (e.g., "CONTAINS(?title, \"erasure\")")
}

// OrderBy represents an ORDER BY clause.
type OrderBy struct {
	Variable   string
	Descending bool
}

// IsVariable checks if a string is a SPARQL variable.
func IsVariable(s string) bool {
	return len(s) > 0 && s[0] == '?'
}

// IsURI checks if a string is a URI reference (enclosed in angle brackets).
// Empty URIs (<>) are not considered valid.
func IsURI(s string) bool {
	return len(s) > 2 && s[0] == '<' && s[len(s)-1] == '>'
}

// IsLiteral checks if a string is a quoted literal.
// Empty literals ("") are not considered valid.
func IsLiteral(s string) bool {
	return len(s) > 2 && s[0] == '"' && s[len(s)-1] == '"'
}

// IsPrefixed checks if a string is a prefixed name (e.g., reg:Article).
func IsPrefixed(s string) bool {
	if len(s) == 0 || s[0] == '?' || s[0] == '<' || s[0] == '"' {
		return false
	}
	for i, c := range s {
		if c == ':' && i > 0 && i < len(s)-1 {
			return true
		}
	}
	return false
}

// StripVariable removes the ? prefix from a variable.
func StripVariable(s string) string {
	if IsVariable(s) {
		return s[1:]
	}
	return s
}

// StripURI removes the < > brackets from a URI.
func StripURI(s string) string {
	if IsURI(s) {
		return s[1 : len(s)-1]
	}
	return s
}

// StripLiteral removes the quotes from a literal string.
func StripLiteral(s string) string {
	if IsLiteral(s) {
		return s[1 : len(s)-1]
	}
	return s
}

// VariableName returns the variable name without the ? prefix, or empty if not a variable.
func VariableName(s string) string {
	if IsVariable(s) {
		return s[1:]
	}
	return ""
}
