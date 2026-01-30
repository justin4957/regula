package store

import (
	"fmt"
	"sort"
	"strings"
)

// RDFXMLSerializer converts a TripleStore into W3C-compliant RDF/XML format.
type RDFXMLSerializer struct {
	prefixMappings []PrefixMapping
	prefixIndex    map[string]string // prefix -> namespace
	namespaceIndex map[string]string // namespace -> prefix
}

// RDFXMLOption is a functional option for configuring the RDFXMLSerializer.
type RDFXMLOption func(*RDFXMLSerializer)

// NewRDFXMLSerializer creates an RDFXMLSerializer with standard namespace declarations.
func NewRDFXMLSerializer(options ...RDFXMLOption) *RDFXMLSerializer {
	serializer := &RDFXMLSerializer{
		prefixMappings: defaultPrefixMappings(),
	}

	for _, option := range options {
		option(serializer)
	}

	serializer.rebuildIndexes()

	return serializer
}

// WithRDFXMLPrefix adds or overrides a namespace prefix mapping.
func WithRDFXMLPrefix(prefix, namespace string) RDFXMLOption {
	return func(serializer *RDFXMLSerializer) {
		serializer.prefixMappings = append(serializer.prefixMappings, PrefixMapping{
			Prefix:    prefix,
			Namespace: namespace,
		})
	}
}

// WithoutRDFXMLDefaultPrefixes clears default prefixes so only custom ones are used.
func WithoutRDFXMLDefaultPrefixes() RDFXMLOption {
	return func(serializer *RDFXMLSerializer) {
		serializer.prefixMappings = nil
	}
}

func (serializer *RDFXMLSerializer) rebuildIndexes() {
	serializer.prefixIndex = make(map[string]string, len(serializer.prefixMappings))
	serializer.namespaceIndex = make(map[string]string, len(serializer.prefixMappings))

	for _, mapping := range serializer.prefixMappings {
		serializer.prefixIndex[mapping.Prefix] = mapping.Namespace
		serializer.namespaceIndex[mapping.Namespace] = mapping.Prefix
	}
}

// Serialize converts all triples in the store to RDF/XML format.
func (serializer *RDFXMLSerializer) Serialize(store *TripleStore) string {
	var builder strings.Builder

	subjectGroups := serializer.groupTriplesBySubject(store)
	sortedSubjects := sortedKeys(subjectGroups)

	serializer.writeXMLHeader(&builder)

	for _, subject := range sortedSubjects {
		serializer.writeDescription(&builder, subject, subjectGroups[subject])
	}

	serializer.writeXMLFooter(&builder)

	return builder.String()
}

// groupTriplesBySubject organizes triples into subject -> predicate -> []object.
func (serializer *RDFXMLSerializer) groupTriplesBySubject(store *TripleStore) map[string]map[string][]string {
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

// writeXMLHeader writes the XML declaration and opening rdf:RDF element with namespace attributes.
func (serializer *RDFXMLSerializer) writeXMLHeader(builder *strings.Builder) {
	builder.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	builder.WriteString("<rdf:RDF")

	sortedPrefixes := make([]PrefixMapping, len(serializer.prefixMappings))
	copy(sortedPrefixes, serializer.prefixMappings)
	sort.Slice(sortedPrefixes, func(i, j int) bool {
		return sortedPrefixes[i].Prefix < sortedPrefixes[j].Prefix
	})

	for _, mapping := range sortedPrefixes {
		fmt.Fprintf(builder, "\n    xmlns:%s=\"%s\"", mapping.Prefix, escapeXMLAttribute(mapping.Namespace))
	}

	builder.WriteString(">\n")
}

// writeXMLFooter writes the closing rdf:RDF element.
func (serializer *RDFXMLSerializer) writeXMLFooter(builder *strings.Builder) {
	builder.WriteString("</rdf:RDF>\n")
}

// writeDescription writes an rdf:Description block for a single subject.
func (serializer *RDFXMLSerializer) writeDescription(
	builder *strings.Builder,
	subject string,
	predicateObjectMap map[string][]string,
) {
	subjectURI := serializer.expandToFullURI(subject)

	builder.WriteString("\n")
	fmt.Fprintf(builder, "  <rdf:Description rdf:about=\"%s\">\n", escapeXMLAttribute(subjectURI))

	sortedPredicates := serializer.sortPredicatesTypeFirst(predicateObjectMap)

	for _, predicate := range sortedPredicates {
		objects := predicateObjectMap[predicate]
		sort.Strings(objects)

		for _, object := range objects {
			serializer.writeProperty(builder, predicate, object)
		}
	}

	builder.WriteString("  </rdf:Description>\n")
}

// writeProperty writes a single predicate-object pair as an XML element.
func (serializer *RDFXMLSerializer) writeProperty(builder *strings.Builder, predicate string, object string) {
	elementName := serializer.predicateToElementName(predicate)

	if isURIObject(object) {
		objectURI := serializer.expandToFullURI(object)
		fmt.Fprintf(builder, "    <%s rdf:resource=\"%s\"/>\n", elementName, escapeXMLAttribute(objectURI))
	} else {
		fmt.Fprintf(builder, "    <%s>%s</%s>\n", elementName, escapeXMLText(object), elementName)
	}
}

// predicateToElementName converts a predicate URI or prefixed name to an XML element name.
// Prefixed names like "reg:title" become the element name directly.
// Full URIs are split into namespace + local name to produce a prefixed element name.
func (serializer *RDFXMLSerializer) predicateToElementName(predicate string) string {
	if isFullURI(predicate) {
		if prefix, localName, ok := serializer.splitPrefixedName(predicate); ok {
			return prefix + ":" + localName
		}
		// Fall back to full URI — not valid XML element name, but preserves data
		return predicate
	}

	// Already a prefixed name like "rdf:type" or "reg:title"
	return predicate
}

// expandToFullURI converts a prefixed name to its full URI form.
// If the value is already a full URI, it is returned unchanged.
func (serializer *RDFXMLSerializer) expandToFullURI(value string) string {
	if isFullURI(value) {
		return value
	}

	colonIndex := strings.Index(value, ":")
	if colonIndex <= 0 {
		return value
	}

	prefix := value[:colonIndex]
	localName := value[colonIndex+1:]

	if namespace, exists := serializer.prefixIndex[prefix]; exists {
		return namespace + localName
	}

	return value
}

// splitPrefixedName splits a full URI into a registered prefix and local name.
// Returns the prefix, local name, and whether a matching namespace was found.
func (serializer *RDFXMLSerializer) splitPrefixedName(fullURI string) (string, string, bool) {
	bestPrefix := ""
	bestNamespace := ""

	for namespace, prefix := range serializer.namespaceIndex {
		if strings.HasPrefix(fullURI, namespace) && len(namespace) > len(bestNamespace) {
			localName := fullURI[len(namespace):]
			if localName != "" {
				bestPrefix = prefix
				bestNamespace = namespace
			}
		}
	}

	if bestNamespace != "" {
		return bestPrefix, fullURI[len(bestNamespace):], true
	}

	return "", "", false
}

// sortPredicatesTypeFirst sorts predicates with rdf:type first, then alphabetically.
func (serializer *RDFXMLSerializer) sortPredicatesTypeFirst(predicateObjectMap map[string][]string) []string {
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

// isURIObject determines whether an object value is a URI reference (as opposed to a literal).
func isURIObject(value string) bool {
	if isFullURI(value) {
		return true
	}
	if isPrefixedName(value) {
		return true
	}
	return false
}

// escapeXMLText escapes characters that are special in XML text content.
func escapeXMLText(text string) string {
	var builder strings.Builder
	builder.Grow(len(text) + len(text)/8)

	for _, char := range text {
		switch char {
		case '&':
			builder.WriteString("&amp;")
		case '<':
			builder.WriteString("&lt;")
		case '>':
			builder.WriteString("&gt;")
		default:
			builder.WriteRune(char)
		}
	}

	return builder.String()
}

// escapeXMLAttribute escapes characters that are special in XML attribute values.
func escapeXMLAttribute(text string) string {
	var builder strings.Builder
	builder.Grow(len(text) + len(text)/8)

	for _, char := range text {
		switch char {
		case '&':
			builder.WriteString("&amp;")
		case '<':
			builder.WriteString("&lt;")
		case '>':
			builder.WriteString("&gt;")
		case '"':
			builder.WriteString("&quot;")
		default:
			builder.WriteRune(char)
		}
	}

	return builder.String()
}
