package query

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ParseQuery parses a SPARQL query string and returns a Query object.
func ParseQuery(queryStr string) (*Query, error) {
	queryStr = strings.TrimSpace(queryStr)

	if queryStr == "" {
		return nil, fmt.Errorf("empty query")
	}

	// Detect query type by keyword position
	upperQuery := strings.ToUpper(queryStr)

	describeIdx := strings.Index(upperQuery, "DESCRIBE")
	constructIdx := strings.Index(upperQuery, "CONSTRUCT")
	selectIdx := strings.Index(upperQuery, "SELECT")

	// DESCRIBE query takes priority if it appears first
	if describeIdx >= 0 &&
		(constructIdx < 0 || describeIdx < constructIdx) &&
		(selectIdx < 0 || describeIdx < selectIdx) {
		describeQuery, err := parseDescribeQuery(queryStr)
		if err != nil {
			return nil, err
		}
		return &Query{
			Type:     DescribeQueryType,
			Describe: describeQuery,
		}, nil
	}

	// CONSTRUCT query if CONSTRUCT appears and (SELECT doesn't appear or CONSTRUCT appears first)
	if constructIdx >= 0 && (selectIdx < 0 || constructIdx < selectIdx) {
		constructQuery, err := parseConstructQuery(queryStr)
		if err != nil {
			return nil, err
		}
		return &Query{
			Type:      ConstructQueryType,
			Construct: constructQuery,
		}, nil
	}

	if selectIdx >= 0 {
		selectQuery, err := parseSelectQuery(queryStr)
		if err != nil {
			return nil, err
		}
		return &Query{
			Type:   SelectQueryType,
			Select: selectQuery,
		}, nil
	}

	return nil, fmt.Errorf("unsupported query type: only SELECT, CONSTRUCT, and DESCRIBE queries are supported")
}

// parseSelectQuery parses a SELECT query.
func parseSelectQuery(queryStr string) (*SelectQuery, error) {
	query := &SelectQuery{
		Prefixes: make(map[string]string),
		Limit:    0,
		Offset:   0,
	}

	// Extract PREFIX declarations
	prefixRegex := regexp.MustCompile(`(?i)PREFIX\s+(\w+):\s*<([^>]+)>`)
	prefixMatches := prefixRegex.FindAllStringSubmatch(queryStr, -1)
	for _, match := range prefixMatches {
		if len(match) == 3 {
			query.Prefixes[match[1]] = match[2]
		}
	}

	// Remove PREFIX declarations for easier parsing
	queryStr = prefixRegex.ReplaceAllString(queryStr, "")

	// Check for DISTINCT
	distinctRegex := regexp.MustCompile(`(?i)\bSELECT\s+DISTINCT\b`)
	if distinctRegex.MatchString(queryStr) {
		query.Distinct = true
		queryStr = regexp.MustCompile(`(?i)\bDISTINCT\b`).ReplaceAllString(queryStr, "")
	}

	// Extract variables from SELECT clause
	selectRegex := regexp.MustCompile(`(?i)SELECT\s+([\s\S]*?)\s+WHERE`)
	selectMatch := selectRegex.FindStringSubmatch(queryStr)
	if selectMatch == nil {
		return nil, fmt.Errorf("invalid SELECT query: missing WHERE clause")
	}

	varsStr := strings.TrimSpace(selectMatch[1])
	if varsStr == "*" {
		query.Variables = []string{"*"}
	} else {
		// Extract aggregate expressions first: (COUNT(?x) AS ?count), (SUM(?y) AS ?total), etc.
		aggregateRegex := regexp.MustCompile(`(?i)\(\s*(COUNT|SUM|AVG|MIN|MAX)\s*\(\s*(DISTINCT\s+)?\?(\w+)\s*\)\s+AS\s+\?(\w+)\s*\)`)
		aggregateMatches := aggregateRegex.FindAllStringSubmatch(varsStr, -1)
		for _, match := range aggregateMatches {
			if len(match) == 5 {
				aggExpr := AggregateExpression{
					Function: AggregateFunction(strings.ToUpper(match[1])),
					Variable: "?" + match[3],
					Alias:    "?" + match[4],
					Distinct: strings.TrimSpace(match[2]) != "",
				}
				query.Aggregates = append(query.Aggregates, aggExpr)
			}
		}

		// Remove aggregate expressions from varsStr, then extract remaining plain variables
		remainingVars := aggregateRegex.ReplaceAllString(varsStr, "")
		varRegex := regexp.MustCompile(`\?(\w+)`)
		varMatches := varRegex.FindAllString(remainingVars, -1)
		query.Variables = varMatches

		// If no plain vars and no aggregates, it's an error
		if len(varMatches) == 0 && len(query.Aggregates) == 0 {
			return nil, fmt.Errorf("no variables found in SELECT clause")
		}
	}

	// Extract the main WHERE clause content — use the last closing brace that
	// matches the WHERE opening brace to handle nested braces in OPTIONAL etc.
	whereRegex := regexp.MustCompile(`(?i)WHERE\s*\{([\s\S]*)\}`)
	whereMatch := whereRegex.FindStringSubmatch(queryStr)
	if whereMatch == nil {
		return nil, fmt.Errorf("invalid WHERE clause: missing braces")
	}

	whereClause := whereMatch[1]

	// Extract OPTIONAL clauses before parsing main patterns
	optionalRegex := regexp.MustCompile(`(?i)OPTIONAL\s*\{([^}]+)\}`)
	optionalMatches := optionalRegex.FindAllStringSubmatch(whereClause, -1)
	for _, match := range optionalMatches {
		if len(match) == 2 {
			optionalPatterns, err := parseTriplePatterns(match[1], query.Prefixes)
			if err != nil {
				return nil, fmt.Errorf("error parsing OPTIONAL clause: %w", err)
			}
			query.Optional = append(query.Optional, optionalPatterns)
		}
	}

	// Remove OPTIONAL clauses from main WHERE clause
	mainWhereClause := optionalRegex.ReplaceAllString(whereClause, "")

	// Extract FILTER clauses
	query.Filters = extractFilters(mainWhereClause)

	// Remove FILTER clauses before parsing triple patterns
	filterRemoveRegex := regexp.MustCompile(`(?i)FILTER\s*\([^)]*\)`)
	mainWhereClause = filterRemoveRegex.ReplaceAllString(mainWhereClause, "")

	// Parse main triple patterns
	patterns, err := parseTriplePatterns(mainWhereClause, query.Prefixes)
	if err != nil {
		return nil, err
	}
	query.Where = patterns

	// Extract GROUP BY
	groupByRegex := regexp.MustCompile(`(?i)GROUP\s+BY\s+((?:\?\w+\s*)+)`)
	groupByMatch := groupByRegex.FindStringSubmatch(queryStr)
	if groupByMatch != nil {
		groupByVarRegex := regexp.MustCompile(`\?\w+`)
		groupByVars := groupByVarRegex.FindAllString(groupByMatch[1], -1)
		query.GroupBy = groupByVars
	}

	// Extract HAVING clauses (uses balanced parenthesis like FILTER)
	query.Having = extractHaving(queryStr)

	// Extract ORDER BY - handle both ASC/DESC(?var) and simple ?var forms
	orderByRegex := regexp.MustCompile(`(?i)ORDER\s+BY\s+((?:(?:ASC|DESC)\s*\(\s*\?\w+\s*\)|\?\w+)(?:\s+(?:ASC|DESC)\s*\(\s*\?\w+\s*\)|\s+\?\w+)*)`)
	orderByMatch := orderByRegex.FindStringSubmatch(queryStr)
	if orderByMatch != nil {
		orderByStr := orderByMatch[1]
		query.OrderBy = parseOrderBy(orderByStr)
	}

	// Extract LIMIT
	limitRegex := regexp.MustCompile(`(?i)LIMIT\s+(\d+)`)
	limitMatch := limitRegex.FindStringSubmatch(queryStr)
	if limitMatch != nil {
		limit, _ := strconv.Atoi(limitMatch[1])
		query.Limit = limit
	}

	// Extract OFFSET
	offsetRegex := regexp.MustCompile(`(?i)OFFSET\s+(\d+)`)
	offsetMatch := offsetRegex.FindStringSubmatch(queryStr)
	if offsetMatch != nil {
		offset, _ := strconv.Atoi(offsetMatch[1])
		query.Offset = offset
	}

	return query, nil
}

// parseConstructQuery parses a CONSTRUCT query.
func parseConstructQuery(queryStr string) (*ConstructQuery, error) {
	query := &ConstructQuery{
		Prefixes: make(map[string]string),
	}

	// Extract PREFIX declarations
	prefixRegex := regexp.MustCompile(`(?i)PREFIX\s+(\w+):\s*<([^>]+)>`)
	prefixMatches := prefixRegex.FindAllStringSubmatch(queryStr, -1)
	for _, match := range prefixMatches {
		if len(match) == 3 {
			query.Prefixes[match[1]] = match[2]
		}
	}

	// Remove PREFIX declarations for easier parsing
	queryStr = prefixRegex.ReplaceAllString(queryStr, "")

	// Extract CONSTRUCT template (between CONSTRUCT { and })
	constructRegex := regexp.MustCompile(`(?i)CONSTRUCT\s*\{([\s\S]*?)\}\s*WHERE`)
	constructMatch := constructRegex.FindStringSubmatch(queryStr)
	if constructMatch == nil {
		return nil, fmt.Errorf("invalid CONSTRUCT query: missing CONSTRUCT template or WHERE clause")
	}

	// Parse template patterns
	templatePatterns, err := parseTriplePatterns(constructMatch[1], query.Prefixes)
	if err != nil {
		return nil, fmt.Errorf("error parsing CONSTRUCT template: %w", err)
	}
	query.Template = templatePatterns

	// Extract the main WHERE clause content
	whereRegex := regexp.MustCompile(`(?i)WHERE\s*\{([\s\S]*)\}`)
	whereMatch := whereRegex.FindStringSubmatch(queryStr)
	if whereMatch == nil {
		return nil, fmt.Errorf("invalid WHERE clause: missing braces")
	}

	whereClause := whereMatch[1]

	// Extract OPTIONAL clauses before parsing main patterns
	optionalRegex := regexp.MustCompile(`(?i)OPTIONAL\s*\{([^}]+)\}`)
	optionalMatches := optionalRegex.FindAllStringSubmatch(whereClause, -1)
	for _, match := range optionalMatches {
		if len(match) == 2 {
			optionalPatterns, err := parseTriplePatterns(match[1], query.Prefixes)
			if err != nil {
				return nil, fmt.Errorf("error parsing OPTIONAL clause: %w", err)
			}
			query.Optional = append(query.Optional, optionalPatterns)
		}
	}

	// Remove OPTIONAL clauses from main WHERE clause
	mainWhereClause := optionalRegex.ReplaceAllString(whereClause, "")

	// Extract FILTER clauses
	query.Filters = extractFilters(mainWhereClause)

	// Remove FILTER clauses before parsing triple patterns
	filterRemoveRegex := regexp.MustCompile(`(?i)FILTER\s*\([^)]*\)`)
	mainWhereClause = filterRemoveRegex.ReplaceAllString(mainWhereClause, "")

	// Parse main triple patterns
	patterns, err := parseTriplePatterns(mainWhereClause, query.Prefixes)
	if err != nil {
		return nil, err
	}
	query.Where = patterns

	return query, nil
}

// parseDescribeQuery parses a DESCRIBE query.
func parseDescribeQuery(queryStr string) (*DescribeQuery, error) {
	describeQuery := &DescribeQuery{
		Prefixes: make(map[string]string),
	}

	// Extract PREFIX declarations
	prefixRegex := regexp.MustCompile(`(?i)PREFIX\s+(\w+):\s*<([^>]+)>`)
	prefixMatches := prefixRegex.FindAllStringSubmatch(queryStr, -1)
	for _, match := range prefixMatches {
		if len(match) == 3 {
			describeQuery.Prefixes[match[1]] = match[2]
		}
	}

	// Remove PREFIX declarations for easier parsing
	queryStr = prefixRegex.ReplaceAllString(queryStr, "")

	// Detect query form by checking for WHERE clause
	upperQuery := strings.ToUpper(queryStr)
	whereIdx := strings.Index(upperQuery, "WHERE")
	hasWhere := whereIdx > 0

	if hasWhere {
		// Variable form: DESCRIBE ?var WHERE { ... }
		describeRegex := regexp.MustCompile(`(?i)DESCRIBE\s+([\s\S]*?)\s+WHERE`)
		describeMatch := describeRegex.FindStringSubmatch(queryStr)
		if describeMatch == nil {
			return nil, fmt.Errorf("invalid DESCRIBE query: could not parse resources before WHERE")
		}

		resourcesStr := strings.TrimSpace(describeMatch[1])
		describeQuery.Resources = parseResourceList(resourcesStr)

		if len(describeQuery.Resources) == 0 {
			return nil, fmt.Errorf("DESCRIBE query has no resources to describe")
		}

		// Extract the main WHERE clause content
		whereRegex := regexp.MustCompile(`(?i)WHERE\s*\{([\s\S]*)\}`)
		whereMatch := whereRegex.FindStringSubmatch(queryStr)
		if whereMatch == nil {
			return nil, fmt.Errorf("invalid WHERE clause: missing braces")
		}

		whereClause := whereMatch[1]

		// Extract OPTIONAL clauses before parsing main patterns
		optionalRegex := regexp.MustCompile(`(?i)OPTIONAL\s*\{([^}]+)\}`)
		optionalMatches := optionalRegex.FindAllStringSubmatch(whereClause, -1)
		for _, match := range optionalMatches {
			if len(match) == 2 {
				optionalPatterns, err := parseTriplePatterns(match[1], describeQuery.Prefixes)
				if err != nil {
					return nil, fmt.Errorf("error parsing OPTIONAL clause: %w", err)
				}
				describeQuery.Optional = append(describeQuery.Optional, optionalPatterns)
			}
		}

		// Remove OPTIONAL clauses from main WHERE clause
		mainWhereClause := optionalRegex.ReplaceAllString(whereClause, "")

		// Extract FILTER clauses
		describeQuery.Filters = extractFilters(mainWhereClause)

		// Remove FILTER clauses before parsing triple patterns
		filterRemoveRegex := regexp.MustCompile(`(?i)FILTER\s*\([^)]*\)`)
		mainWhereClause = filterRemoveRegex.ReplaceAllString(mainWhereClause, "")

		// Parse main triple patterns
		patterns, err := parseTriplePatterns(mainWhereClause, describeQuery.Prefixes)
		if err != nil {
			return nil, err
		}
		describeQuery.Where = patterns

	} else {
		// Direct URI form: DESCRIBE <uri> or DESCRIBE prefix:name
		describeRegex := regexp.MustCompile(`(?i)DESCRIBE\s+([\s\S]+)`)
		describeMatch := describeRegex.FindStringSubmatch(queryStr)
		if describeMatch == nil {
			return nil, fmt.Errorf("invalid DESCRIBE query: no resources specified")
		}

		resourcesStr := strings.TrimSpace(describeMatch[1])
		describeQuery.Resources = parseResourceList(resourcesStr)

		if len(describeQuery.Resources) == 0 {
			return nil, fmt.Errorf("DESCRIBE query has no resources to describe")
		}
	}

	return describeQuery, nil
}

// parseResourceList parses a space-separated list of URIs, variables, prefixed names,
// or bare identifiers used as resource references.
func parseResourceList(resourcesStr string) []string {
	var resources []string
	tokens := tokenize(resourcesStr)

	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		// Accept URIs (<...>), variables (?...), prefixed names (prefix:local),
		// or bare identifiers used as resource references in the store
		if IsURI(token) || IsVariable(token) || IsPrefixed(token) || isIdentifier(token) {
			resources = append(resources, token)
		}
	}

	return resources
}

// isIdentifier checks if a string is a bare identifier (alphanumeric, not a keyword).
func isIdentifier(s string) bool {
	if len(s) == 0 || s[0] == '"' || s[0] == '<' || s[0] == '?' {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.') {
			return false
		}
	}
	return true
}

// parseOrderBy parses ORDER BY clause variables.
func parseOrderBy(orderByStr string) []OrderBy {
	var orderBys []OrderBy

	// Match ASC(?var) or DESC(?var) patterns
	funcRegex := regexp.MustCompile(`(?i)(ASC|DESC)\s*\(\s*\?(\w+)\s*\)`)
	funcMatches := funcRegex.FindAllStringSubmatch(orderByStr, -1)
	for _, match := range funcMatches {
		if len(match) == 3 {
			orderBys = append(orderBys, OrderBy{
				Variable:   "?" + match[2],
				Descending: strings.ToUpper(match[1]) == "DESC",
			})
		}
	}

	// If no function matches, try simple variable format
	if len(orderBys) == 0 {
		varRegex := regexp.MustCompile(`\?(\w+)`)
		varMatches := varRegex.FindAllStringSubmatch(orderByStr, -1)
		for _, match := range varMatches {
			if len(match) == 2 {
				orderBys = append(orderBys, OrderBy{
					Variable:   "?" + match[1],
					Descending: false,
				})
			}
		}
	}

	return orderBys
}

// extractFilters extracts FILTER clauses with balanced parentheses.
func extractFilters(whereClause string) []Filter {
	var filters []Filter

	// Find all FILTER keywords
	filterKeyword := regexp.MustCompile(`(?i)\bFILTER\s*\(`)
	matches := filterKeyword.FindAllStringIndex(whereClause, -1)

	for _, match := range matches {
		startIdx := match[1] // Position after "FILTER("

		// Find matching closing parenthesis
		depth := 1
		endIdx := startIdx
		for endIdx < len(whereClause) && depth > 0 {
			if whereClause[endIdx] == '(' {
				depth++
			} else if whereClause[endIdx] == ')' {
				depth--
			}
			endIdx++
		}

		if depth == 0 {
			// Extract expression between balanced parentheses
			expression := strings.TrimSpace(whereClause[startIdx : endIdx-1])
			filters = append(filters, Filter{Expression: expression})
		}
	}

	return filters
}

// extractHaving extracts HAVING clauses with balanced parentheses from the full query string.
func extractHaving(queryStr string) []Filter {
	var havingFilters []Filter

	havingKeyword := regexp.MustCompile(`(?i)\bHAVING\s*\(`)
	matches := havingKeyword.FindAllStringIndex(queryStr, -1)

	for _, match := range matches {
		startIdx := match[1] // Position after "HAVING("

		// Find matching closing parenthesis
		depth := 1
		endIdx := startIdx
		for endIdx < len(queryStr) && depth > 0 {
			if queryStr[endIdx] == '(' {
				depth++
			} else if queryStr[endIdx] == ')' {
				depth--
			}
			endIdx++
		}

		if depth == 0 {
			expression := strings.TrimSpace(queryStr[startIdx : endIdx-1])
			havingFilters = append(havingFilters, Filter{Expression: expression})
		}
	}

	return havingFilters
}

// splitTriples splits a WHERE clause by periods, but not periods inside URIs or literals.
func splitTriples(whereClause string) []string {
	var triples []string
	var current strings.Builder
	inURI := false
	inLiteral := false

	for i := 0; i < len(whereClause); i++ {
		ch := whereClause[i]
		if ch == '<' && !inLiteral {
			inURI = true
			current.WriteByte(ch)
		} else if ch == '>' && inURI {
			inURI = false
			current.WriteByte(ch)
		} else if ch == '"' && !inURI {
			inLiteral = !inLiteral
			current.WriteByte(ch)
		} else if ch == '.' && !inURI && !inLiteral {
			// Triple terminator
			if current.Len() > 0 {
				triples = append(triples, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(ch)
		}
	}

	// Add final triple if any
	if current.Len() > 0 {
		triples = append(triples, current.String())
	}

	return triples
}

// parseTriplePatterns parses triple patterns from a WHERE clause.
func parseTriplePatterns(whereClause string, prefixes map[string]string) ([]TriplePattern, error) {
	var patterns []TriplePattern

	// Split by period (end of triple) but not periods inside URIs or literals
	lines := splitTriples(whereClause)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip FILTER lines (already extracted)
		if strings.HasPrefix(strings.ToUpper(line), "FILTER") {
			continue
		}

		// Handle semicolon (same subject continuation)
		parts := strings.Split(line, ";")

		var currentSubject string
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			// Tokenize the triple pattern
			tokens := tokenize(part)
			if len(tokens) < 3 {
				// Check if this is a continuation (has only 2 tokens)
				if len(tokens) == 2 && currentSubject != "" {
					tokens = append([]string{currentSubject}, tokens...)
				} else {
					continue
				}
			}

			subject := tokens[0]
			predicate := tokens[1]

			// Handle "a" as rdf:type
			if predicate == "a" {
				predicate = "rdf:type"
			}

			// Object is everything after predicate (handle multi-word literals)
			object := strings.Join(tokens[2:], " ")

			patterns = append(patterns, TriplePattern{
				Subject:   subject,
				Predicate: predicate,
				Object:    object,
			})

			currentSubject = subject
		}
	}

	return patterns, nil
}

// tokenize splits a triple pattern into tokens, respecting URIs and literals.
func tokenize(s string) []string {
	var tokens []string
	var current strings.Builder
	inURI := false
	inLiteral := false

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if ch == '<' && !inLiteral {
			inURI = true
			current.WriteByte(ch)
		} else if ch == '>' && inURI {
			inURI = false
			current.WriteByte(ch)
			// End of URI token
			tokens = append(tokens, current.String())
			current.Reset()
		} else if ch == '"' {
			inLiteral = !inLiteral
			current.WriteByte(ch)
			if !inLiteral {
				// End of literal token
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else if (ch == ' ' || ch == '\t' || ch == '\n') && !inURI && !inLiteral {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// ExpandPrefixes expands all prefixed URIs in a query using the declared prefixes.
func (q *SelectQuery) ExpandPrefixes() {
	// Expand in WHERE patterns
	for i := range q.Where {
		q.Where[i].Subject = expandPrefix(q.Where[i].Subject, q.Prefixes)
		q.Where[i].Predicate = expandPrefix(q.Where[i].Predicate, q.Prefixes)
		q.Where[i].Object = expandPrefix(q.Where[i].Object, q.Prefixes)
	}

	// Expand in OPTIONAL patterns
	for i := range q.Optional {
		for j := range q.Optional[i] {
			q.Optional[i][j].Subject = expandPrefix(q.Optional[i][j].Subject, q.Prefixes)
			q.Optional[i][j].Predicate = expandPrefix(q.Optional[i][j].Predicate, q.Prefixes)
			q.Optional[i][j].Object = expandPrefix(q.Optional[i][j].Object, q.Prefixes)
		}
	}
}

// ExpandPrefixes expands all prefixed URIs in a CONSTRUCT query using the declared prefixes.
func (q *ConstructQuery) ExpandPrefixes() {
	// Expand in CONSTRUCT template patterns
	for i := range q.Template {
		q.Template[i].Subject = expandPrefix(q.Template[i].Subject, q.Prefixes)
		q.Template[i].Predicate = expandPrefix(q.Template[i].Predicate, q.Prefixes)
		q.Template[i].Object = expandPrefix(q.Template[i].Object, q.Prefixes)
	}

	// Expand in WHERE patterns
	for i := range q.Where {
		q.Where[i].Subject = expandPrefix(q.Where[i].Subject, q.Prefixes)
		q.Where[i].Predicate = expandPrefix(q.Where[i].Predicate, q.Prefixes)
		q.Where[i].Object = expandPrefix(q.Where[i].Object, q.Prefixes)
	}

	// Expand in OPTIONAL patterns
	for i := range q.Optional {
		for j := range q.Optional[i] {
			q.Optional[i][j].Subject = expandPrefix(q.Optional[i][j].Subject, q.Prefixes)
			q.Optional[i][j].Predicate = expandPrefix(q.Optional[i][j].Predicate, q.Prefixes)
			q.Optional[i][j].Object = expandPrefix(q.Optional[i][j].Object, q.Prefixes)
		}
	}
}

// expandPrefix expands a prefixed URI using the provided prefix map.
func expandPrefix(term string, prefixes map[string]string) string {
	term = strings.TrimSpace(term)

	// Skip if already a variable, full URI, or literal
	if term == "" || term[0] == '?' || term[0] == '<' || term[0] == '"' {
		return term
	}

	// Check if it's a prefixed URI
	colonIdx := strings.Index(term, ":")
	if colonIdx > 0 && colonIdx < len(term)-1 {
		prefix := term[:colonIdx]
		localName := term[colonIdx+1:]

		if baseURI, ok := prefixes[prefix]; ok {
			return "<" + baseURI + localName + ">"
		}
	}

	return term
}

// Validate checks if the query is well-formed and returns validation errors.
func (q *Query) Validate() []error {
	var errors []error

	if q.Type == "" {
		errors = append(errors, fmt.Errorf("query type is not set"))
	}

	if q.Select == nil && q.Type == SelectQueryType {
		errors = append(errors, fmt.Errorf("SELECT query missing select clause"))
		return errors
	}

	if q.Construct == nil && q.Type == ConstructQueryType {
		errors = append(errors, fmt.Errorf("CONSTRUCT query missing construct clause"))
		return errors
	}

	if q.Describe == nil && q.Type == DescribeQueryType {
		errors = append(errors, fmt.Errorf("DESCRIBE query missing describe clause"))
		return errors
	}

	if q.Select != nil {
		errors = append(errors, q.Select.Validate()...)
	}

	if q.Construct != nil {
		errors = append(errors, q.Construct.Validate()...)
	}

	if q.Describe != nil {
		errors = append(errors, q.Describe.Validate()...)
	}

	return errors
}

// Validate checks if the SELECT query is well-formed.
func (q *SelectQuery) Validate() []error {
	var errors []error

	if len(q.Variables) == 0 && len(q.Aggregates) == 0 {
		errors = append(errors, fmt.Errorf("SELECT clause has no variables"))
	}

	if len(q.Where) == 0 {
		errors = append(errors, fmt.Errorf("WHERE clause has no triple patterns"))
	}

	// Collect all variables bound in WHERE and OPTIONAL patterns
	boundVars := make(map[string]bool)
	for _, p := range q.Where {
		if IsVariable(p.Subject) {
			boundVars[p.Subject] = true
		}
		if IsVariable(p.Predicate) {
			boundVars[p.Predicate] = true
		}
		if IsVariable(p.Object) {
			boundVars[p.Object] = true
		}
	}
	for _, opt := range q.Optional {
		for _, p := range opt {
			if IsVariable(p.Subject) {
				boundVars[p.Subject] = true
			}
			if IsVariable(p.Predicate) {
				boundVars[p.Predicate] = true
			}
			if IsVariable(p.Object) {
				boundVars[p.Object] = true
			}
		}
	}

	if q.HasAggregates() {
		// Aggregate-specific validation
		for _, agg := range q.Aggregates {
			if !boundVars[agg.Variable] {
				errors = append(errors, fmt.Errorf("aggregate source variable %s is not bound in WHERE clause", agg.Variable))
			}
		}

		// GROUP BY variables must be bound in WHERE
		for _, groupVar := range q.GroupBy {
			if !boundVars[groupVar] {
				errors = append(errors, fmt.Errorf("GROUP BY variable %s is not bound in WHERE clause", groupVar))
			}
		}

		// Plain SELECT variables must appear in GROUP BY when aggregates are present
		groupBySet := make(map[string]bool)
		for _, groupVar := range q.GroupBy {
			groupBySet[groupVar] = true
		}
		for _, v := range q.Variables {
			if v != "*" && !groupBySet[v] {
				errors = append(errors, fmt.Errorf("variable %s in SELECT must appear in GROUP BY when aggregates are used", v))
			}
		}

		// ORDER BY can reference aggregate aliases or GROUP BY variables
		aliasSet := make(map[string]bool)
		for _, agg := range q.Aggregates {
			aliasSet[agg.Alias] = true
		}
		for _, ob := range q.OrderBy {
			if !groupBySet[ob.Variable] && !aliasSet[ob.Variable] {
				errors = append(errors, fmt.Errorf("ORDER BY variable %s must be a GROUP BY variable or aggregate alias", ob.Variable))
			}
		}
	} else {
		// Non-aggregate validation (existing behavior)
		if len(q.Variables) > 0 && q.Variables[0] != "*" {
			for _, v := range q.Variables {
				if !boundVars[v] {
					errors = append(errors, fmt.Errorf("variable %s in SELECT is not bound in WHERE clause", v))
				}
			}
		}

		// Check ORDER BY variables are bound
		for _, ob := range q.OrderBy {
			found := false
			for _, v := range q.Variables {
				if v == "*" || v == ob.Variable {
					found = true
					break
				}
			}
			if !found {
				errors = append(errors, fmt.Errorf("ORDER BY variable %s is not in SELECT clause", ob.Variable))
			}
		}
	}

	if q.Limit < 0 {
		errors = append(errors, fmt.Errorf("LIMIT cannot be negative"))
	}

	if q.Offset < 0 {
		errors = append(errors, fmt.Errorf("OFFSET cannot be negative"))
	}

	return errors
}

// String returns a string representation of the query (for debugging).
func (q *Query) String() string {
	if q.Select != nil {
		return q.Select.String()
	}
	if q.Construct != nil {
		return q.Construct.String()
	}
	if q.Describe != nil {
		return q.Describe.String()
	}
	return "<unknown query type>"
}

// String returns a string representation of the SELECT query.
func (q *SelectQuery) String() string {
	var sb strings.Builder

	// Prefixes
	for prefix, uri := range q.Prefixes {
		sb.WriteString(fmt.Sprintf("PREFIX %s: <%s>\n", prefix, uri))
	}

	// SELECT clause
	sb.WriteString("SELECT ")
	if q.Distinct {
		sb.WriteString("DISTINCT ")
	}

	var selectParts []string
	if len(q.Variables) == 1 && q.Variables[0] == "*" {
		selectParts = append(selectParts, "*")
	} else {
		selectParts = append(selectParts, q.Variables...)
	}
	for _, agg := range q.Aggregates {
		aggStr := "("
		aggStr += string(agg.Function) + "("
		if agg.Distinct {
			aggStr += "DISTINCT "
		}
		aggStr += agg.Variable + ") AS " + agg.Alias + ")"
		selectParts = append(selectParts, aggStr)
	}
	sb.WriteString(strings.Join(selectParts, " "))

	// WHERE clause
	sb.WriteString(" WHERE {\n")
	for _, p := range q.Where {
		sb.WriteString(fmt.Sprintf("  %s %s %s .\n", p.Subject, p.Predicate, p.Object))
	}
	for _, f := range q.Filters {
		sb.WriteString(fmt.Sprintf("  FILTER(%s)\n", f.Expression))
	}
	for _, opt := range q.Optional {
		sb.WriteString("  OPTIONAL {\n")
		for _, p := range opt {
			sb.WriteString(fmt.Sprintf("    %s %s %s .\n", p.Subject, p.Predicate, p.Object))
		}
		sb.WriteString("  }\n")
	}
	sb.WriteString("}")

	// GROUP BY
	if len(q.GroupBy) > 0 {
		sb.WriteString(" GROUP BY ")
		sb.WriteString(strings.Join(q.GroupBy, " "))
	}

	// HAVING
	for _, h := range q.Having {
		sb.WriteString(fmt.Sprintf(" HAVING(%s)", h.Expression))
	}

	// ORDER BY
	if len(q.OrderBy) > 0 {
		sb.WriteString(" ORDER BY")
		for _, ob := range q.OrderBy {
			if ob.Descending {
				sb.WriteString(fmt.Sprintf(" DESC(%s)", ob.Variable))
			} else {
				sb.WriteString(fmt.Sprintf(" %s", ob.Variable))
			}
		}
	}

	// LIMIT
	if q.Limit > 0 {
		sb.WriteString(fmt.Sprintf(" LIMIT %d", q.Limit))
	}

	// OFFSET
	if q.Offset > 0 {
		sb.WriteString(fmt.Sprintf(" OFFSET %d", q.Offset))
	}

	return sb.String()
}

// Validate checks if the CONSTRUCT query is well-formed.
func (q *ConstructQuery) Validate() []error {
	var errors []error

	if len(q.Template) == 0 {
		errors = append(errors, fmt.Errorf("CONSTRUCT template has no triple patterns"))
	}

	if len(q.Where) == 0 {
		errors = append(errors, fmt.Errorf("WHERE clause has no triple patterns"))
	}

	// Collect all variables bound in WHERE clause and OPTIONAL
	boundVars := make(map[string]bool)
	for _, p := range q.Where {
		if IsVariable(p.Subject) {
			boundVars[p.Subject] = true
		}
		if IsVariable(p.Predicate) {
			boundVars[p.Predicate] = true
		}
		if IsVariable(p.Object) {
			boundVars[p.Object] = true
		}
	}
	for _, opt := range q.Optional {
		for _, p := range opt {
			if IsVariable(p.Subject) {
				boundVars[p.Subject] = true
			}
			if IsVariable(p.Predicate) {
				boundVars[p.Predicate] = true
			}
			if IsVariable(p.Object) {
				boundVars[p.Object] = true
			}
		}
	}

	// Check that all variables in template are bound in WHERE clause
	for _, p := range q.Template {
		if IsVariable(p.Subject) && !boundVars[p.Subject] {
			errors = append(errors, fmt.Errorf("variable %s in CONSTRUCT template is not bound in WHERE clause", p.Subject))
		}
		if IsVariable(p.Predicate) && !boundVars[p.Predicate] {
			errors = append(errors, fmt.Errorf("variable %s in CONSTRUCT template is not bound in WHERE clause", p.Predicate))
		}
		if IsVariable(p.Object) && !boundVars[p.Object] {
			errors = append(errors, fmt.Errorf("variable %s in CONSTRUCT template is not bound in WHERE clause", p.Object))
		}
	}

	return errors
}

// String returns a string representation of the CONSTRUCT query.
func (q *ConstructQuery) String() string {
	var sb strings.Builder

	// Prefixes
	for prefix, uri := range q.Prefixes {
		sb.WriteString(fmt.Sprintf("PREFIX %s: <%s>\n", prefix, uri))
	}

	// CONSTRUCT template
	sb.WriteString("CONSTRUCT {\n")
	for _, p := range q.Template {
		sb.WriteString(fmt.Sprintf("  %s %s %s .\n", p.Subject, p.Predicate, p.Object))
	}
	sb.WriteString("}")

	// WHERE clause
	sb.WriteString(" WHERE {\n")
	for _, p := range q.Where {
		sb.WriteString(fmt.Sprintf("  %s %s %s .\n", p.Subject, p.Predicate, p.Object))
	}
	for _, f := range q.Filters {
		sb.WriteString(fmt.Sprintf("  FILTER(%s)\n", f.Expression))
	}
	for _, opt := range q.Optional {
		sb.WriteString("  OPTIONAL {\n")
		for _, p := range opt {
			sb.WriteString(fmt.Sprintf("    %s %s %s .\n", p.Subject, p.Predicate, p.Object))
		}
		sb.WriteString("  }\n")
	}
	sb.WriteString("}")

	return sb.String()
}

// Validate checks if the DESCRIBE query is well-formed.
func (q *DescribeQuery) Validate() []error {
	var errors []error

	if len(q.Resources) == 0 {
		errors = append(errors, fmt.Errorf("DESCRIBE query has no resources"))
	}

	// If WHERE clause exists, verify resource variables are bound
	if len(q.Where) > 0 {
		boundVars := make(map[string]bool)
		for _, p := range q.Where {
			if IsVariable(p.Subject) {
				boundVars[p.Subject] = true
			}
			if IsVariable(p.Predicate) {
				boundVars[p.Predicate] = true
			}
			if IsVariable(p.Object) {
				boundVars[p.Object] = true
			}
		}
		for _, opt := range q.Optional {
			for _, p := range opt {
				if IsVariable(p.Subject) {
					boundVars[p.Subject] = true
				}
				if IsVariable(p.Predicate) {
					boundVars[p.Predicate] = true
				}
				if IsVariable(p.Object) {
					boundVars[p.Object] = true
				}
			}
		}

		for _, resource := range q.Resources {
			if IsVariable(resource) && !boundVars[resource] {
				errors = append(errors, fmt.Errorf("variable %s in DESCRIBE is not bound in WHERE clause", resource))
			}
		}
	}

	return errors
}

// ExpandPrefixes expands all prefixed URIs in a DESCRIBE query using the declared prefixes.
func (q *DescribeQuery) ExpandPrefixes() {
	// Expand resources
	for i := range q.Resources {
		q.Resources[i] = expandPrefix(q.Resources[i], q.Prefixes)
	}

	// Expand in WHERE patterns
	for i := range q.Where {
		q.Where[i].Subject = expandPrefix(q.Where[i].Subject, q.Prefixes)
		q.Where[i].Predicate = expandPrefix(q.Where[i].Predicate, q.Prefixes)
		q.Where[i].Object = expandPrefix(q.Where[i].Object, q.Prefixes)
	}

	// Expand in OPTIONAL patterns
	for i := range q.Optional {
		for j := range q.Optional[i] {
			q.Optional[i][j].Subject = expandPrefix(q.Optional[i][j].Subject, q.Prefixes)
			q.Optional[i][j].Predicate = expandPrefix(q.Optional[i][j].Predicate, q.Prefixes)
			q.Optional[i][j].Object = expandPrefix(q.Optional[i][j].Object, q.Prefixes)
		}
	}
}

// String returns a string representation of the DESCRIBE query.
func (q *DescribeQuery) String() string {
	var sb strings.Builder

	// Prefixes
	for prefix, uri := range q.Prefixes {
		sb.WriteString(fmt.Sprintf("PREFIX %s: <%s>\n", prefix, uri))
	}

	// DESCRIBE clause
	sb.WriteString("DESCRIBE ")
	sb.WriteString(strings.Join(q.Resources, " "))

	// WHERE clause (if present)
	if len(q.Where) > 0 {
		sb.WriteString(" WHERE {\n")
		for _, p := range q.Where {
			sb.WriteString(fmt.Sprintf("  %s %s %s .\n", p.Subject, p.Predicate, p.Object))
		}
		for _, f := range q.Filters {
			sb.WriteString(fmt.Sprintf("  FILTER(%s)\n", f.Expression))
		}
		for _, opt := range q.Optional {
			sb.WriteString("  OPTIONAL {\n")
			for _, p := range opt {
				sb.WriteString(fmt.Sprintf("    %s %s %s .\n", p.Subject, p.Predicate, p.Object))
			}
			sb.WriteString("  }\n")
		}
		sb.WriteString("}")
	}

	return sb.String()
}
