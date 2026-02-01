package playground

import (
	"sort"
	"strings"
	"testing"

	"github.com/coolbeans/regula/pkg/query"
)

func TestRegistryContainsAllTemplates(t *testing.T) {
	requiredTemplates := []string{
		"top-chapters-by-sections",
		"sections-with-obligations",
		"definition-coverage",
		"cross-ref-density",
		"rights-enumeration",
		"title-size-comparison",
		"orphan-sections",
		"definition-reuse",
		"chapter-structure",
		"temporal-analysis",
	}

	registry := Registry()
	for _, templateName := range requiredTemplates {
		if _, exists := registry[templateName]; !exists {
			t.Errorf("missing required template: %s", templateName)
		}
	}

	if len(registry) < 10 {
		t.Errorf("expected at least 10 templates, got %d", len(registry))
	}
}

func TestTemplateNamesAreSorted(t *testing.T) {
	names := TemplateNames()

	if len(names) < 10 {
		t.Fatalf("expected at least 10 template names, got %d", len(names))
	}

	sorted := make([]string, len(names))
	copy(sorted, names)
	sort.Strings(sorted)

	for nameIndex, name := range names {
		if name != sorted[nameIndex] {
			t.Errorf("names[%d] = %q, want %q (not sorted)", nameIndex, name, sorted[nameIndex])
		}
	}
}

func TestGetExistingTemplate(t *testing.T) {
	template, exists := Get("cross-ref-density")
	if !exists {
		t.Fatal("expected cross-ref-density template to exist")
	}
	if template.Name != "cross-ref-density" {
		t.Errorf("template name = %q, want %q", template.Name, "cross-ref-density")
	}
	if template.Category != "cross-reference" {
		t.Errorf("template category = %q, want %q", template.Category, "cross-reference")
	}
	if len(template.Parameters) != 1 {
		t.Errorf("expected 1 parameter, got %d", len(template.Parameters))
	}
}

func TestGetMissingTemplate(t *testing.T) {
	_, exists := Get("nonexistent-template")
	if exists {
		t.Error("expected nonexistent template to return false")
	}
}

func TestRenderQueryNoParameters(t *testing.T) {
	template, _ := Get("top-chapters-by-sections")
	rendered, err := RenderQuery(template, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Template has no %s placeholder, should return unchanged
	if rendered != template.Query {
		t.Errorf("rendered query should equal original for template without parameters")
	}
}

func TestRenderQueryWithTitleFilter(t *testing.T) {
	template, _ := Get("cross-ref-density")
	params := map[string]string{"title": "42"}

	rendered, err := RenderQuery(template, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(rendered, "%%s") || (strings.Contains(rendered, "%") && strings.Contains(rendered, "s}")) {
		t.Error("rendered query still contains unsubstituted placeholder")
	}
	placeholder := "%s"
	if strings.Contains(rendered, placeholder) {
		t.Error("rendered query still contains unsubstituted placeholder")
	}
	if !strings.Contains(rendered, `CONTAINS(STR(?article), "42")`) {
		t.Error("expected FILTER with title value '42'")
	}
}

func TestRenderQueryWithEmptyTitleFilter(t *testing.T) {
	template, _ := Get("cross-ref-density")
	params := map[string]string{"title": ""}

	rendered, err := RenderQuery(template, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	placeholder := "%s"
	if strings.Contains(rendered, placeholder) {
		t.Error("rendered query still contains unsubstituted placeholder")
	}
	// Empty title should not inject a FILTER
	if strings.Contains(rendered, "FILTER") {
		t.Error("empty title should not inject a FILTER clause")
	}
}

func TestRenderQueryNoParamsForParameterized(t *testing.T) {
	template, _ := Get("chapter-structure")
	rendered, err := RenderQuery(template, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	placeholder := "%s"
	if strings.Contains(rendered, placeholder) {
		t.Error("rendered query still contains unsubstituted placeholder")
	}
}

func TestRenderQueryMissingRequired(t *testing.T) {
	// Create a template with a required parameter for testing
	template := PlaygroundTemplate{
		Name:  "test-required",
		Query: `SELECT ?s WHERE { ?s ?p ?o . %s }`,
		Parameters: []TemplateParameter{
			{Name: "required-param", Description: "A required param", Required: true},
		},
	}

	_, err := RenderQuery(template, nil)
	if err == nil {
		t.Error("expected error for missing required parameter")
	}
	if err != nil && !strings.Contains(err.Error(), "required-param") {
		t.Errorf("error should mention parameter name, got: %v", err)
	}
}

func TestAllTemplatesHaveRequiredFields(t *testing.T) {
	for templateName, template := range Registry() {
		if template.Name == "" {
			t.Errorf("template %q has empty Name", templateName)
		}
		if template.Name != templateName {
			t.Errorf("template key %q does not match Name %q", templateName, template.Name)
		}
		if template.Description == "" {
			t.Errorf("template %q has empty Description", templateName)
		}
		if template.Category == "" {
			t.Errorf("template %q has empty Category", templateName)
		}
		if template.Query == "" {
			t.Errorf("template %q has empty Query", templateName)
		}
	}
}

func TestAllTemplatesParseSuccessfully(t *testing.T) {
	for templateName, template := range Registry() {
		t.Run(templateName, func(t *testing.T) {
			// Render with empty params to clear %s placeholders
			rendered, err := RenderQuery(template, nil)
			if err != nil {
				t.Fatalf("RenderQuery failed: %v", err)
			}

			// Parse through the SPARQL parser
			parsedQuery, parseErr := query.ParseQuery(rendered)
			if parseErr != nil {
				t.Fatalf("ParseQuery failed for template %q: %v\nQuery: %s", templateName, parseErr, rendered)
			}

			if parsedQuery == nil {
				t.Fatalf("ParseQuery returned nil without error for template %q", templateName)
			}
		})
	}
}

func TestAllTemplatesWithTitleParam(t *testing.T) {
	// Templates with title parameter should also parse when title is provided
	parameterizedTemplates := []string{"cross-ref-density", "chapter-structure"}

	for _, templateName := range parameterizedTemplates {
		t.Run(templateName+"-with-title", func(t *testing.T) {
			template, exists := Get(templateName)
			if !exists {
				t.Fatalf("template %q not found", templateName)
			}

			rendered, err := RenderQuery(template, map[string]string{"title": "42"})
			if err != nil {
				t.Fatalf("RenderQuery failed: %v", err)
			}

			parsedQuery, parseErr := query.ParseQuery(rendered)
			if parseErr != nil {
				t.Fatalf("ParseQuery failed for template %q with title param: %v\nQuery: %s", templateName, parseErr, rendered)
			}

			if parsedQuery == nil {
				t.Fatalf("ParseQuery returned nil without error for template %q", templateName)
			}
		})
	}
}

func TestTemplateCategoriesAreValid(t *testing.T) {
	validCategories := map[string]bool{
		"structure":       true,
		"semantics":       true,
		"cross-reference": true,
		"definitions":     true,
		"temporal":        true,
	}

	for templateName, template := range Registry() {
		if !validCategories[template.Category] {
			t.Errorf("template %q has invalid category %q", templateName, template.Category)
		}
	}
}
