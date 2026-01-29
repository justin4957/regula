package store

import (
	"encoding/json"
	"sort"
	"strings"
)

// JSONLDContext represents a JSON-LD @context document.
type JSONLDContext map[string]interface{}

// JSONLDSerializer converts a TripleStore into JSON-LD format.
type JSONLDSerializer struct {
	prefixMappings []PrefixMapping
	prefixIndex    map[string]string // prefix -> namespace
	namespaceIndex map[string]string // namespace -> prefix
	compactForm    bool              // If true, produce compact JSON-LD; otherwise expanded
}

// JSONLDOption is a functional option for configuring the JSONLDSerializer.
type JSONLDOption func(*JSONLDSerializer)

// NewJSONLDSerializer creates a JSONLDSerializer with standard prefix declarations.
func NewJSONLDSerializer(options ...JSONLDOption) *JSONLDSerializer {
	serializer := &JSONLDSerializer{
		prefixMappings: defaultPrefixMappings(),
		compactForm:    true, // Default to compact form
	}

	for _, option := range options {
		option(serializer)
	}

	serializer.rebuildIndexes()

	return serializer
}

// WithJSONLDPrefix adds or overrides a prefix mapping.
func WithJSONLDPrefix(prefix, namespace string) JSONLDOption {
	return func(serializer *JSONLDSerializer) {
		serializer.prefixMappings = append(serializer.prefixMappings, PrefixMapping{
			Prefix:    prefix,
			Namespace: namespace,
		})
	}
}

// WithExpandedForm configures the serializer to output expanded JSON-LD (no context compaction).
func WithExpandedForm() JSONLDOption {
	return func(serializer *JSONLDSerializer) {
		serializer.compactForm = false
	}
}

// WithCompactForm configures the serializer to output compact JSON-LD (with @context).
func WithCompactForm() JSONLDOption {
	return func(serializer *JSONLDSerializer) {
		serializer.compactForm = true
	}
}

func (serializer *JSONLDSerializer) rebuildIndexes() {
	serializer.prefixIndex = make(map[string]string, len(serializer.prefixMappings))
	serializer.namespaceIndex = make(map[string]string, len(serializer.prefixMappings))

	for _, mapping := range serializer.prefixMappings {
		serializer.prefixIndex[mapping.Prefix] = mapping.Namespace
		serializer.namespaceIndex[mapping.Namespace] = mapping.Prefix
	}
}

// BuildContext creates the JSON-LD @context document from prefix mappings.
func (serializer *JSONLDSerializer) BuildContext() JSONLDContext {
	context := make(JSONLDContext)

	// Add namespace prefixes
	for _, mapping := range serializer.prefixMappings {
		context[mapping.Prefix] = mapping.Namespace
	}

	// Add well-known term mappings for common predicates
	// These map compact property names to their full IRIs
	context["type"] = "@type"
	context["id"] = "@id"

	// Map reg: predicates to JSON-LD property terms
	context["title"] = map[string]string{"@id": "reg:title"}
	context["text"] = map[string]string{"@id": "reg:text"}
	context["number"] = map[string]string{"@id": "reg:number"}
	context["identifier"] = map[string]string{"@id": "reg:identifier"}
	context["partOf"] = map[string]string{"@id": "reg:partOf", "@type": "@id"}
	context["contains"] = map[string]string{"@id": "reg:contains", "@type": "@id"}
	context["belongsTo"] = map[string]string{"@id": "reg:belongsTo", "@type": "@id"}
	context["references"] = map[string]string{"@id": "reg:references", "@type": "@id"}
	context["referencedBy"] = map[string]string{"@id": "reg:referencedBy", "@type": "@id"}
	context["defines"] = map[string]string{"@id": "reg:defines", "@type": "@id"}
	context["definedIn"] = map[string]string{"@id": "reg:definedIn", "@type": "@id"}
	context["term"] = map[string]string{"@id": "reg:term"}
	context["definition"] = map[string]string{"@id": "reg:definition"}
	context["grantsRight"] = map[string]string{"@id": "reg:grantsRight", "@type": "@id"}
	context["imposesObligation"] = map[string]string{"@id": "reg:imposesObligation", "@type": "@id"}

	// ELI vocabulary mappings
	context["eli_title"] = map[string]string{"@id": "eli:title"}
	context["eli_id_local"] = map[string]string{"@id": "eli:id_local"}
	context["eli_is_part_of"] = map[string]string{"@id": "eli:is_part_of", "@type": "@id"}
	context["eli_has_part"] = map[string]string{"@id": "eli:has_part", "@type": "@id"}
	context["eli_cites"] = map[string]string{"@id": "eli:cites", "@type": "@id"}
	context["eli_cited_by"] = map[string]string{"@id": "eli:cited_by", "@type": "@id"}

	// Dublin Core mappings
	context["dc_title"] = map[string]string{"@id": "dc:title"}
	context["dc_description"] = map[string]string{"@id": "dc:description"}
	context["dc_date"] = map[string]string{"@id": "dc:date"}

	return context
}

// JSONLDDocument represents a complete JSON-LD document.
type JSONLDDocument struct {
	Context interface{}              `json:"@context,omitempty"`
	Graph   []map[string]interface{} `json:"@graph"`
}

// Serialize converts all triples in the store to JSON-LD format.
func (serializer *JSONLDSerializer) Serialize(store *TripleStore) ([]byte, error) {
	if serializer.compactForm {
		return serializer.serializeCompact(store)
	}
	return serializer.serializeExpanded(store)
}

// serializeCompact produces compact JSON-LD with @context.
func (serializer *JSONLDSerializer) serializeCompact(store *TripleStore) ([]byte, error) {
	subjectGroups := serializer.groupTriplesBySubject(store)
	sortedSubjects := sortedKeys(subjectGroups)

	graph := make([]map[string]interface{}, 0, len(sortedSubjects))

	for _, subject := range sortedSubjects {
		node := serializer.buildNode(subject, subjectGroups[subject])
		graph = append(graph, node)
	}

	doc := JSONLDDocument{
		Context: serializer.BuildContext(),
		Graph:   graph,
	}

	return json.MarshalIndent(doc, "", "  ")
}

// serializeExpanded produces expanded JSON-LD (no context, full IRIs).
func (serializer *JSONLDSerializer) serializeExpanded(store *TripleStore) ([]byte, error) {
	subjectGroups := serializer.groupTriplesBySubject(store)
	sortedSubjects := sortedKeys(subjectGroups)

	graph := make([]map[string]interface{}, 0, len(sortedSubjects))

	for _, subject := range sortedSubjects {
		node := serializer.buildExpandedNode(subject, subjectGroups[subject])
		graph = append(graph, node)
	}

	return json.MarshalIndent(graph, "", "  ")
}

// SerializeToString returns the JSON-LD as a string.
func (serializer *JSONLDSerializer) SerializeToString(store *TripleStore) (string, error) {
	data, err := serializer.Serialize(store)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// groupTriplesBySubject organizes triples into subject -> predicate -> []object.
func (serializer *JSONLDSerializer) groupTriplesBySubject(store *TripleStore) map[string]map[string][]string {
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

// buildNode creates a compact JSON-LD node from a subject and its predicates.
func (serializer *JSONLDSerializer) buildNode(subject string, predicateObjectMap map[string][]string) map[string]interface{} {
	node := make(map[string]interface{})

	// Set @id
	node["@id"] = serializer.compactURI(subject)

	// Sort predicates for deterministic output
	sortedPredicates := serializer.sortPredicatesTypeFirst(predicateObjectMap)

	for _, predicate := range sortedPredicates {
		objects := predicateObjectMap[predicate]
		sort.Strings(objects)

		key := serializer.predicateToKey(predicate)
		value := serializer.formatObjects(predicate, objects)

		node[key] = value
	}

	return node
}

// buildExpandedNode creates an expanded JSON-LD node with full IRIs.
func (serializer *JSONLDSerializer) buildExpandedNode(subject string, predicateObjectMap map[string][]string) map[string]interface{} {
	node := make(map[string]interface{})

	// Set @id with full IRI
	node["@id"] = serializer.expandURI(subject)

	sortedPredicates := serializer.sortPredicatesTypeFirst(predicateObjectMap)

	for _, predicate := range sortedPredicates {
		objects := predicateObjectMap[predicate]
		sort.Strings(objects)

		key := serializer.expandURI(predicate)
		value := serializer.formatExpandedObjects(predicate, objects)

		node[key] = value
	}

	return node
}

// predicateToKey converts a predicate URI to a JSON-LD key.
func (serializer *JSONLDSerializer) predicateToKey(predicate string) string {
	// Handle rdf:type specially
	if predicate == RDFType || predicate == NamespaceRDF+"type" {
		return "@type"
	}

	// Try to compact to prefixed form
	compacted := serializer.compactURI(predicate)

	// If it's a known reg: predicate, use the short form without prefix
	if strings.HasPrefix(compacted, "reg:") {
		return compacted[4:] // Remove "reg:" prefix for cleaner JSON
	}

	return compacted
}

// formatObjects formats object values for JSON-LD.
func (serializer *JSONLDSerializer) formatObjects(predicate string, objects []string) interface{} {
	// Handle rdf:type specially - always use @type with compacted class names
	if predicate == RDFType || predicate == NamespaceRDF+"type" {
		if len(objects) == 1 {
			return serializer.compactURI(objects[0])
		}
		result := make([]string, len(objects))
		for i, obj := range objects {
			result[i] = serializer.compactURI(obj)
		}
		return result
	}

	// Check if predicate is a relationship (object is a URI reference)
	isRelationship := serializer.isRelationshipPredicate(predicate)

	if len(objects) == 1 {
		if isRelationship {
			return map[string]string{"@id": serializer.compactURI(objects[0])}
		}
		return objects[0]
	}

	// Multiple objects
	result := make([]interface{}, len(objects))
	for i, obj := range objects {
		if isRelationship {
			result[i] = map[string]string{"@id": serializer.compactURI(obj)}
		} else {
			result[i] = obj
		}
	}
	return result
}

// formatExpandedObjects formats object values for expanded JSON-LD.
func (serializer *JSONLDSerializer) formatExpandedObjects(predicate string, objects []string) interface{} {
	// Handle rdf:type - use @type array
	if predicate == RDFType || predicate == NamespaceRDF+"type" {
		result := make([]map[string]string, len(objects))
		for i, obj := range objects {
			result[i] = map[string]string{"@id": serializer.expandURI(obj)}
		}
		return result
	}

	isRelationship := serializer.isRelationshipPredicate(predicate)

	result := make([]map[string]string, len(objects))
	for i, obj := range objects {
		if isRelationship {
			result[i] = map[string]string{"@id": serializer.expandURI(obj)}
		} else {
			result[i] = map[string]string{"@value": obj}
		}
	}

	return result
}

// compactURI replaces full namespace URIs with prefixed forms.
func (serializer *JSONLDSerializer) compactURI(fullURI string) string {
	// If already a prefixed name (but not a full URI), return as-is
	if isPrefixedName(fullURI) && !isFullURI(fullURI) {
		return fullURI
	}

	// Try longest namespace match first
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
		return bestPrefix + ":" + fullURI[len(bestNamespace):]
	}

	return fullURI
}

// expandURI converts prefixed names to full URIs.
func (serializer *JSONLDSerializer) expandURI(uri string) string {
	// If it's a full URI, return as-is
	if isFullURI(uri) {
		return uri
	}

	// Try to expand prefixed name
	colonIndex := strings.Index(uri, ":")
	if colonIndex > 0 {
		prefix := uri[:colonIndex]
		localName := uri[colonIndex+1:]

		if namespace, exists := serializer.prefixIndex[prefix]; exists {
			return namespace + localName
		}
	}

	return uri
}

// isRelationshipPredicate checks if a predicate indicates a relationship (URI object).
func (serializer *JSONLDSerializer) isRelationshipPredicate(predicate string) bool {
	// Expand the predicate to check against known relationships
	expandedPredicate := serializer.expandURI(predicate)

	relationshipPredicates := []string{
		PropPartOf,
		PropContains,
		PropBelongsTo,
		PropHasChapter,
		PropHasSection,
		PropHasArticle,
		PropHasParagraph,
		PropHasPoint,
		PropHasRecital,
		PropReferences,
		PropReferencedBy,
		PropRefersToArticle,
		PropRefersToChapter,
		PropRefersToPoint,
		PropDefines,
		PropDefinedIn,
		PropUsesTerm,
		PropGrantsRight,
		PropImposesObligation,
		PropAmends,
		PropAmendedBy,
		PropSupersedes,
		PropSupersededBy,
		PropRepeals,
		PropRepealedBy,
		PropDelegatesTo,
		PropResolvedTarget,
		PropAlternativeTarget,
		PropExternalRef,
		ELIPropIsPartOf,
		ELIPropHasPart,
		ELIPropCites,
		ELIPropCitedBy,
	}

	// Check both compact and expanded forms
	for _, rp := range relationshipPredicates {
		expandedRP := serializer.expandURI(rp)
		if predicate == rp || expandedPredicate == expandedRP || predicate == expandedRP {
			return true
		}
	}
	return false
}

// sortPredicatesTypeFirst sorts predicates with rdf:type first, then alphabetically.
func (serializer *JSONLDSerializer) sortPredicatesTypeFirst(predicateObjectMap map[string][]string) []string {
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

// GetContextOnly returns just the @context portion as JSON.
func (serializer *JSONLDSerializer) GetContextOnly() ([]byte, error) {
	context := serializer.BuildContext()
	return json.MarshalIndent(context, "", "  ")
}

// DefaultJSONLDContext returns the standard context for regulation documents.
func DefaultJSONLDContext() JSONLDContext {
	serializer := NewJSONLDSerializer()
	return serializer.BuildContext()
}
