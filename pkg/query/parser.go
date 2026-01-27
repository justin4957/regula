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

	// Detect query type
	upperQuery := strings.ToUpper(queryStr)
	if strings.Contains(upperQuery, "SELECT") {
		selectQuery, err := parseSelectQuery(queryStr)
		if err != nil {
			return nil, err
		}
		return &Query{
			Type:   SelectQueryType,
			Select: selectQuery,
		}, nil
	}

	return nil, fmt.Errorf("unsupported query type: only SELECT queries are supported")
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
		// Extract variables (including those with ?)
		varRegex := regexp.MustCompile(`\?(\w+)`)
		varMatches := varRegex.FindAllString(varsStr, -1)
		if len(varMatches) == 0 {
			return nil, fmt.Errorf("no variables found in SELECT clause")
		}
		query.Variables = varMatches
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

	if q.Select != nil {
		errors = append(errors, q.Select.Validate()...)
	}

	return errors
}

// Validate checks if the SELECT query is well-formed.
func (q *SelectQuery) Validate() []error {
	var errors []error

	if len(q.Variables) == 0 {
		errors = append(errors, fmt.Errorf("SELECT clause has no variables"))
	}

	if len(q.Where) == 0 {
		errors = append(errors, fmt.Errorf("WHERE clause has no triple patterns"))
	}

	// Check that all selected variables (except *) appear in WHERE patterns
	if len(q.Variables) > 0 && q.Variables[0] != "*" {
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
		// Also check OPTIONAL patterns
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
	if len(q.Variables) == 1 && q.Variables[0] == "*" {
		sb.WriteString("*")
	} else {
		sb.WriteString(strings.Join(q.Variables, " "))
	}

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
