package store

import (
	"fmt"
	"sort"
	"strings"
)

// PrefixMapping associates a short prefix label with its full namespace URI.
type PrefixMapping struct {
	Prefix    string
	Namespace string
}

// TurtleSerializer converts a TripleStore into W3C-compliant Turtle (TTL) format.
type TurtleSerializer struct {
	prefixMappings []PrefixMapping
	prefixIndex    map[string]string // prefix -> namespace
	namespaceIndex map[string]string // namespace -> prefix
}

// TurtleOption is a functional option for configuring the TurtleSerializer.
type TurtleOption func(*TurtleSerializer)

// NewTurtleSerializer creates a TurtleSerializer with standard prefix declarations.
func NewTurtleSerializer(options ...TurtleOption) *TurtleSerializer {
	serializer := &TurtleSerializer{
		prefixMappings: defaultPrefixMappings(),
	}

	for _, option := range options {
		option(serializer)
	}

	serializer.rebuildIndexes()

	return serializer
}

// WithPrefix adds or overrides a prefix mapping.
func WithPrefix(prefix, namespace string) TurtleOption {
	return func(serializer *TurtleSerializer) {
		serializer.prefixMappings = append(serializer.prefixMappings, PrefixMapping{
			Prefix:    prefix,
			Namespace: namespace,
		})
	}
}

// WithoutDefaultPrefixes clears default prefixes so only custom ones are used.
func WithoutDefaultPrefixes() TurtleOption {
	return func(serializer *TurtleSerializer) {
		serializer.prefixMappings = nil
	}
}

func defaultPrefixMappings() []PrefixMapping {
	return []PrefixMapping{
		{Prefix: "rdf", Namespace: NamespaceRDF},
		{Prefix: "rdfs", Namespace: NamespaceRDFS},
		{Prefix: "xsd", Namespace: NamespaceXSD},
		{Prefix: "dc", Namespace: NamespaceDC},
		{Prefix: "reg", Namespace: NamespaceReg},
		{Prefix: "eli", Namespace: NamespaceELI},
		{Prefix: "frbr", Namespace: NamespaceFRBR},
	}
}

func (serializer *TurtleSerializer) rebuildIndexes() {
	serializer.prefixIndex = make(map[string]string, len(serializer.prefixMappings))
	serializer.namespaceIndex = make(map[string]string, len(serializer.prefixMappings))

	for _, mapping := range serializer.prefixMappings {
		serializer.prefixIndex[mapping.Prefix] = mapping.Namespace
		serializer.namespaceIndex[mapping.Namespace] = mapping.Prefix
	}
}

// Serialize converts all triples in the store to Turtle format.
func (serializer *TurtleSerializer) Serialize(store *TripleStore) string {
	var builder strings.Builder

	serializer.writePrefixDeclarations(&builder)

	subjectGroups := serializer.groupTriplesBySubject(store)
	sortedSubjects := sortedKeys(subjectGroups)

	for subjectIndex, subject := range sortedSubjects {
		if subjectIndex > 0 {
			builder.WriteString("\n")
		}
		serializer.writeSubjectGroup(&builder, subject, subjectGroups[subject])
	}

	return builder.String()
}

func (serializer *TurtleSerializer) writePrefixDeclarations(builder *strings.Builder) {
	sortedPrefixes := make([]PrefixMapping, len(serializer.prefixMappings))
	copy(sortedPrefixes, serializer.prefixMappings)
	sort.Slice(sortedPrefixes, func(i, j int) bool {
		return sortedPrefixes[i].Prefix < sortedPrefixes[j].Prefix
	})

	for _, mapping := range sortedPrefixes {
		fmt.Fprintf(builder, "@prefix %s: <%s> .\n", mapping.Prefix, mapping.Namespace)
	}

	if len(serializer.prefixMappings) > 0 {
		builder.WriteString("\n")
	}
}

// groupTriplesBySubject organizes triples into subject -> predicate -> []object.
func (serializer *TurtleSerializer) groupTriplesBySubject(store *TripleStore) map[string]map[string][]string {
	allTriples := store.All()

	subjectGroups := make(map[string]map[string][]string)

	for _, triple := range allTriples {
		if _, exists := subjectGroups[triple.Subject]; !exists {
			subjectGroups[triple.Subject] = make(map[string][]string)
		}
		subjectGroups[triple.Subject][triple.Predicate] = append(
			subjectGroups[triple.Subject][triple.Predicate],
			triple.Object,
		)
	}

	return subjectGroups
}

func (serializer *TurtleSerializer) writeSubjectGroup(
	builder *strings.Builder,
	subject string,
	predicateObjectMap map[string][]string,
) {
	builder.WriteString(serializer.formatResource(subject))

	sortedPredicates := serializer.sortPredicatesTypeFirst(predicateObjectMap)

	for predicateIndex, predicate := range sortedPredicates {
		objects := predicateObjectMap[predicate]
		sort.Strings(objects)

		if predicateIndex == 0 {
			builder.WriteString(" ")
		} else {
			builder.WriteString(" ;\n    ")
		}

		formattedPredicate := serializer.formatPredicate(predicate)
		builder.WriteString(formattedPredicate)

		for objectIndex, object := range objects {
			if objectIndex > 0 {
				builder.WriteString(" ,\n        ")
			} else {
				builder.WriteString(" ")
			}
			builder.WriteString(serializer.formatObject(object))
		}
	}

	builder.WriteString(" .\n")
}

// formatResource formats a subject or predicate (always a URI or prefixed name).
func (serializer *TurtleSerializer) formatResource(value string) string {
	if isFullURI(value) {
		if compacted, ok := serializer.compactURI(value); ok {
			return compacted
		}
		return "<" + escapeIRI(value) + ">"
	}
	return value
}

// formatPredicate formats a predicate, using "a" shorthand for rdf:type.
func (serializer *TurtleSerializer) formatPredicate(predicate string) string {
	if predicate == RDFType || predicate == NamespaceRDF+"type" {
		return "a"
	}
	return serializer.formatResource(predicate)
}

// formatObject formats an object which may be a URI, prefixed name, or literal.
func (serializer *TurtleSerializer) formatObject(value string) string {
	if isFullURI(value) {
		if compacted, ok := serializer.compactURI(value); ok {
			return compacted
		}
		return "<" + escapeIRI(value) + ">"
	}

	if isPrefixedName(value) {
		return value
	}

	return formatLiteral(value)
}

// compactURI replaces a full namespace URI with its prefix form.
func (serializer *TurtleSerializer) compactURI(fullURI string) (string, bool) {
	// Try longest namespace match first for correctness
	bestPrefix := ""
	bestNamespace := ""
	for namespace, prefix := range serializer.namespaceIndex {
		if strings.HasPrefix(fullURI, namespace) && len(namespace) > len(bestNamespace) {
			localName := fullURI[len(namespace):]
			if isValidLocalName(localName) {
				bestPrefix = prefix
				bestNamespace = namespace
			}
		}
	}

	if bestNamespace != "" {
		return bestPrefix + ":" + fullURI[len(bestNamespace):], true
	}
	return "", false
}

// sortPredicatesTypeFirst sorts predicates with rdf:type first, then alphabetically.
func (serializer *TurtleSerializer) sortPredicatesTypeFirst(predicateObjectMap map[string][]string) []string {
	predicates := make([]string, 0, len(predicateObjectMap))
	hasRDFType := false
	rdfTypeKey := ""

	for predicate := range predicateObjectMap {
		if predicate == RDFType || predicate == NamespaceRDF+"type" {
			hasRDFType = true
			rdfTypeKey = predicate
		} else {
			predicates = append(predicates, predicate)
		}
	}

	sort.Strings(predicates)

	if hasRDFType {
		predicates = append([]string{rdfTypeKey}, predicates...)
	}

	return predicates
}

// isFullURI checks if a value is a full URI (starts with a scheme).
func isFullURI(value string) bool {
	return strings.HasPrefix(value, "http://") ||
		strings.HasPrefix(value, "https://") ||
		strings.HasPrefix(value, "urn:")
}

// isPrefixedName checks if a value looks like a valid Turtle prefixed name.
func isPrefixedName(value string) bool {
	colonIndex := strings.Index(value, ":")
	if colonIndex <= 0 {
		return false
	}

	prefix := value[:colonIndex]
	localName := value[colonIndex+1:]

	for _, char := range prefix {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			return false
		}
	}

	if localName == "" || strings.ContainsAny(localName, " \t\n\r") {
		return false
	}

	return true
}

// isValidLocalName checks if a string is a valid Turtle local name.
func isValidLocalName(localName string) bool {
	if localName == "" {
		return false
	}
	return !strings.ContainsAny(localName, " \t\n\r<>\"{}|^`\\")
}

// formatLiteral wraps a string value in Turtle-compliant double quotes.
func formatLiteral(value string) string {
	escaped := escapeLiteralString(value)

	if strings.Contains(value, "\n") {
		return `"""` + escaped + `"""`
	}

	return `"` + escaped + `"`
}

// escapeLiteralString escapes special characters per W3C Turtle spec.
func escapeLiteralString(value string) string {
	var builder strings.Builder
	builder.Grow(len(value) + len(value)/8)

	for _, char := range value {
		switch char {
		case '\\':
			builder.WriteString(`\\`)
		case '"':
			builder.WriteString(`\"`)
		case '\n':
			builder.WriteString(`\n`)
		case '\r':
			builder.WriteString(`\r`)
		case '\t':
			builder.WriteString(`\t`)
		default:
			builder.WriteRune(char)
		}
	}

	return builder.String()
}

// escapeIRI escapes characters not allowed in IRIs within angle brackets.
func escapeIRI(iri string) string {
	var builder strings.Builder
	builder.Grow(len(iri))

	for _, char := range iri {
		switch char {
		case '<':
			builder.WriteString(`\u003C`)
		case '>':
			builder.WriteString(`\u003E`)
		case '"':
			builder.WriteString(`\u0022`)
		case ' ':
			builder.WriteString(`\u0020`)
		case '{':
			builder.WriteString(`\u007B`)
		case '}':
			builder.WriteString(`\u007D`)
		default:
			builder.WriteRune(char)
		}
	}

	return builder.String()
}

// sortedKeys returns the keys of a map sorted alphabetically.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
