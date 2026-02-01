// Package playground provides pre-built SPARQL analysis query templates
// for exploring USC and other ingested legislation data in the library.
package playground

import (
	"fmt"
	"sort"
	"strings"
)

// TemplateParameter describes a named parameter a template accepts.
type TemplateParameter struct {
	Name         string // parameter name (e.g., "title")
	Description  string // human-readable description
	DefaultValue string // default if not provided
	Required     bool   // whether the parameter must be supplied
}

// PlaygroundTemplate holds a pre-built analysis query for the playground.
type PlaygroundTemplate struct {
	Name        string              // unique slug (e.g., "top-chapters-by-sections")
	Description string              // one-line description
	Category    string              // grouping label (e.g., "structure", "semantics")
	Query       string              // SPARQL query string, may contain %s placeholders
	Parameters  []TemplateParameter // parameters for substitution
}

var templateRegistry = map[string]PlaygroundTemplate{
	"top-chapters-by-sections": {
		Name:        "top-chapters-by-sections",
		Description: "Top chapters by section count",
		Category:    "structure",
		Query: `SELECT ?chapter ?chapterTitle (COUNT(?section) AS ?sectionCount) WHERE {
  ?chapter rdf:type reg:Chapter .
  ?chapter reg:title ?chapterTitle .
  ?chapter reg:hasSection ?section .
  ?section rdf:type reg:Section .
} GROUP BY ?chapter ?chapterTitle ORDER BY DESC(?sectionCount) LIMIT 20`,
	},

	"sections-with-obligations": {
		Name:        "sections-with-obligations",
		Description: "Sections containing obligation predicates",
		Category:    "semantics",
		Query: `SELECT ?section ?sectionTitle ?obligation ?obligationType WHERE {
  ?section rdf:type reg:Section .
  ?section reg:title ?sectionTitle .
  ?section reg:contains ?article .
  ?article reg:imposesObligation ?obligation .
  ?obligation reg:obligationType ?obligationType .
} ORDER BY ?section`,
	},

	"definition-coverage": {
		Name:        "definition-coverage",
		Description: "Definition count per title/regulation",
		Category:    "definitions",
		Query: `SELECT ?regulation ?regulationTitle (COUNT(?definition) AS ?definitionCount) WHERE {
  ?regulation rdf:type reg:Regulation .
  ?regulation reg:title ?regulationTitle .
  ?definition rdf:type reg:DefinedTerm .
  ?definition reg:belongsTo ?regulation .
} GROUP BY ?regulation ?regulationTitle ORDER BY DESC(?definitionCount)`,
	},

	"cross-ref-density": {
		Name:        "cross-ref-density",
		Description: "Articles with the most cross-references",
		Category:    "cross-reference",
		Query: `SELECT ?article ?articleTitle (COUNT(?target) AS ?refCount) WHERE {
  ?article rdf:type reg:Article .
  ?article reg:title ?articleTitle .
  ?article reg:references ?target .
  %s
} GROUP BY ?article ?articleTitle ORDER BY DESC(?refCount) LIMIT 30`,
		Parameters: []TemplateParameter{
			{
				Name:        "title",
				Description: "Filter by title number (e.g., 42)",
			},
		},
	},

	"rights-enumeration": {
		Name:        "rights-enumeration",
		Description: "All identified rights across titles",
		Category:    "semantics",
		Query: `SELECT ?right ?rightType ?article ?articleTitle WHERE {
  ?right rdf:type reg:Right .
  ?right reg:rightType ?rightType .
  ?right reg:partOf ?article .
  ?article reg:title ?articleTitle .
} ORDER BY ?rightType ?article`,
	},

	"title-size-comparison": {
		Name:        "title-size-comparison",
		Description: "Rank titles by article count",
		Category:    "structure",
		Query: `SELECT ?regulation ?regulationTitle (COUNT(?article) AS ?articleCount) WHERE {
  ?regulation rdf:type reg:Regulation .
  ?regulation reg:title ?regulationTitle .
  ?article rdf:type reg:Article .
  ?article reg:belongsTo ?regulation .
} GROUP BY ?regulation ?regulationTitle ORDER BY DESC(?articleCount)`,
	},

	"orphan-sections": {
		Name:        "orphan-sections",
		Description: "Sections with no outgoing cross-references",
		Category:    "cross-reference",
		Query: `SELECT ?section ?sectionTitle WHERE {
  ?section rdf:type reg:Section .
  ?section reg:title ?sectionTitle .
  OPTIONAL { ?section reg:contains ?article . ?article reg:references ?target . }
  FILTER(!BOUND(?target))
} ORDER BY ?section`,
	},

	"definition-reuse": {
		Name:        "definition-reuse",
		Description: "Terms defined in multiple titles or sections",
		Category:    "definitions",
		Query: `SELECT ?termText (COUNT(DISTINCT ?regulation) AS ?titleCount) WHERE {
  ?term rdf:type reg:DefinedTerm .
  ?term reg:normalizedTerm ?termText .
  ?term reg:belongsTo ?regulation .
} GROUP BY ?termText HAVING(COUNT(DISTINCT ?regulation) > 1) ORDER BY DESC(?titleCount)`,
	},

	"chapter-structure": {
		Name:        "chapter-structure",
		Description: "Hierarchical breakdown of title > chapter > section",
		Category:    "structure",
		Query: `SELECT ?regulation ?regulationTitle ?chapter ?chapterTitle ?section ?sectionTitle WHERE {
  ?regulation rdf:type reg:Regulation .
  ?regulation reg:title ?regulationTitle .
  ?regulation reg:hasChapter ?chapter .
  ?chapter rdf:type reg:Chapter .
  ?chapter reg:title ?chapterTitle .
  OPTIONAL { ?chapter reg:hasSection ?section . ?section rdf:type reg:Section . ?section reg:title ?sectionTitle . }
  %s
} ORDER BY ?regulation ?chapter ?section`,
		Parameters: []TemplateParameter{
			{
				Name:        "title",
				Description: "Filter by title number (e.g., 42)",
			},
		},
	},

	"temporal-analysis": {
		Name:        "temporal-analysis",
		Description: "References with temporal qualifiers (amendment dates, effective dates)",
		Category:    "temporal",
		Query: `SELECT ?reference ?temporalKind ?temporalDescription ?article ?articleTitle WHERE {
  ?reference rdf:type reg:Reference .
  ?reference reg:temporalKind ?temporalKind .
  OPTIONAL { ?reference reg:temporalDescription ?temporalDescription . }
  ?reference reg:partOf ?article .
  ?article reg:title ?articleTitle .
} ORDER BY ?temporalKind ?article`,
	},
}

// Registry returns all registered playground templates keyed by name.
func Registry() map[string]PlaygroundTemplate {
	return templateRegistry
}

// TemplateNames returns template names in sorted order for consistent listing.
func TemplateNames() []string {
	names := make([]string, 0, len(templateRegistry))
	for name := range templateRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Get returns a template by name, or false if not found.
func Get(name string) (PlaygroundTemplate, bool) {
	template, exists := templateRegistry[name]
	return template, exists
}

// RenderQuery substitutes parameters into the template query string.
// parameterValues maps parameter name to value. Missing optional parameters
// produce empty substitutions; missing required parameters return an error.
func RenderQuery(template PlaygroundTemplate, parameterValues map[string]string) (string, error) {
	// Check for required parameters
	for _, parameter := range template.Parameters {
		if parameter.Required {
			if value, exists := parameterValues[parameter.Name]; !exists || value == "" {
				return "", fmt.Errorf("required parameter --%s not provided: %s", parameter.Name, parameter.Description)
			}
		}
	}

	renderedQuery := template.Query

	// Build filter clause for "title" parameter
	filterClause := ""
	if titleValue, exists := parameterValues["title"]; exists && titleValue != "" {
		filterClause = fmt.Sprintf(`FILTER(CONTAINS(STR(?article), "%s") || CONTAINS(STR(?regulation), "%s"))`, titleValue, titleValue)
	}

	// Substitute the %s placeholder if present
	if strings.Contains(renderedQuery, "%s") {
		renderedQuery = fmt.Sprintf(renderedQuery, filterClause)
	}

	return renderedQuery, nil
}
