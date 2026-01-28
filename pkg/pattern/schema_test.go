package pattern

import (
	"strings"
	"testing"
)

func TestValidateSchema(t *testing.T) {
	tests := []struct {
		name       string
		pattern    FormatPattern
		wantErrors int
		wantFields []string
	}{
		{
			name: "valid minimal pattern",
			pattern: FormatPattern{
				Name:     "Test Pattern",
				FormatID: "test-pattern",
				Version:  "1.0.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: `test`, Weight: 10},
					},
				},
			},
			wantErrors: 0,
		},
		{
			name: "valid full pattern",
			pattern: FormatPattern{
				Name:         "Full Test Pattern",
				FormatID:     "full-test",
				Version:      "2.1.0",
				Jurisdiction: "US-CA",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: `required`, Weight: 25},
					},
					OptionalIndicators: []Indicator{
						{Pattern: `optional`, Weight: 10},
					},
					NegativeIndicators: []Indicator{
						{Pattern: `negative`, Weight: -15},
					},
				},
				Structure: StructureConfig{
					Hierarchy: []HierarchyLevel{
						{Type: "section", Pattern: `^Section\s+(\d+)`, NumberGroup: 1},
					},
				},
				Definitions: DefinitionConfig{
					Location: []DefinitionLocation{
						{SectionTitle: "definitions"},
					},
					Pattern: `"([^"]+)" means`,
				},
				References: ReferenceConfig{
					Internal: []ReferencePattern{
						{Pattern: `Section\s+(\d+)`, Target: "section", Groups: map[string]int{"number": 1}},
					},
					External: []ExternalReferencePattern{
						{Pattern: `(\d+) U.S.C.`, Type: "usc", URITemplate: "https://example.com/{title}"},
					},
				},
			},
			wantErrors: 0,
		},
		{
			name:       "missing all required fields",
			pattern:    FormatPattern{},
			wantErrors: 4,
			wantFields: []string{"name", "format_id", "version", "detection.required_indicators"},
		},
		{
			name: "invalid format_id",
			pattern: FormatPattern{
				Name:     "Test",
				FormatID: "Invalid_ID",
				Version:  "1.0.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: `test`, Weight: 10},
					},
				},
			},
			wantErrors: 1,
			wantFields: []string{"format_id"},
		},
		{
			name: "format_id starting with number",
			pattern: FormatPattern{
				Name:     "Test",
				FormatID: "123-test",
				Version:  "1.0.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: `test`, Weight: 10},
					},
				},
			},
			wantErrors: 1,
			wantFields: []string{"format_id"},
		},
		{
			name: "invalid version format",
			pattern: FormatPattern{
				Name:     "Test",
				FormatID: "test",
				Version:  "1.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: `test`, Weight: 10},
					},
				},
			},
			wantErrors: 1,
			wantFields: []string{"version"},
		},
		{
			name: "indicator weight too low",
			pattern: FormatPattern{
				Name:     "Test",
				FormatID: "test",
				Version:  "1.0.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: `test`, Weight: 0},
					},
				},
			},
			wantErrors: 1,
			wantFields: []string{"detection.required_indicators[0].weight"},
		},
		{
			name: "indicator weight too high",
			pattern: FormatPattern{
				Name:     "Test",
				FormatID: "test",
				Version:  "1.0.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: `test`, Weight: 150},
					},
				},
			},
			wantErrors: 1,
			wantFields: []string{"detection.required_indicators[0].weight"},
		},
		{
			name: "negative indicator with positive weight",
			pattern: FormatPattern{
				Name:     "Test",
				FormatID: "test",
				Version:  "1.0.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: `test`, Weight: 10},
					},
					NegativeIndicators: []Indicator{
						{Pattern: `bad`, Weight: 5},
					},
				},
			},
			wantErrors: 1,
			wantFields: []string{"detection.negative_indicators[0].weight"},
		},
		{
			name: "invalid hierarchy type",
			pattern: FormatPattern{
				Name:     "Test",
				FormatID: "test",
				Version:  "1.0.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: `test`, Weight: 10},
					},
				},
				Structure: StructureConfig{
					Hierarchy: []HierarchyLevel{
						{Type: "invalid_type", Pattern: `test`},
					},
				},
			},
			wantErrors: 1,
			wantFields: []string{"structure.hierarchy[0].type"},
		},
		{
			name: "missing hierarchy pattern",
			pattern: FormatPattern{
				Name:     "Test",
				FormatID: "test",
				Version:  "1.0.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: `test`, Weight: 10},
					},
				},
				Structure: StructureConfig{
					Hierarchy: []HierarchyLevel{
						{Type: "section"},
					},
				},
			},
			wantErrors: 1,
			wantFields: []string{"structure.hierarchy[0].pattern"},
		},
		{
			name: "definition location missing both fields",
			pattern: FormatPattern{
				Name:     "Test",
				FormatID: "test",
				Version:  "1.0.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: `test`, Weight: 10},
					},
				},
				Definitions: DefinitionConfig{
					Location: []DefinitionLocation{
						{},
					},
				},
			},
			wantErrors: 1,
			wantFields: []string{"definitions.location[0]"},
		},
		{
			name: "internal reference missing target",
			pattern: FormatPattern{
				Name:     "Test",
				FormatID: "test",
				Version:  "1.0.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: `test`, Weight: 10},
					},
				},
				References: ReferenceConfig{
					Internal: []ReferencePattern{
						{Pattern: `Section\s+(\d+)`},
					},
				},
			},
			wantErrors: 1,
			wantFields: []string{"references.internal[0].target"},
		},
		{
			name: "invalid internal reference target",
			pattern: FormatPattern{
				Name:     "Test",
				FormatID: "test",
				Version:  "1.0.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: `test`, Weight: 10},
					},
				},
				References: ReferenceConfig{
					Internal: []ReferencePattern{
						{Pattern: `test`, Target: "invalid_target"},
					},
				},
			},
			wantErrors: 1,
			wantFields: []string{"references.internal[0].target"},
		},
		{
			name: "external reference missing type",
			pattern: FormatPattern{
				Name:     "Test",
				FormatID: "test",
				Version:  "1.0.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: `test`, Weight: 10},
					},
				},
				References: ReferenceConfig{
					External: []ExternalReferencePattern{
						{Pattern: `test`},
					},
				},
			},
			wantErrors: 1,
			wantFields: []string{"references.external[0].type"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateSchema(&tt.pattern)

			if len(errs) != tt.wantErrors {
				t.Errorf("ValidateSchema() got %d errors, want %d", len(errs), tt.wantErrors)
				for _, err := range errs {
					t.Logf("  Error: %s", err.Error())
				}
			}

			// Check that expected fields are mentioned in errors
			for _, wantField := range tt.wantFields {
				found := false
				for _, err := range errs {
					if strings.Contains(err.Field, wantField) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error for field %q not found", wantField)
				}
			}
		})
	}
}

func TestValidationErrorsString(t *testing.T) {
	tests := []struct {
		name   string
		errs   ValidationErrors
		expect string
	}{
		{
			name:   "no errors",
			errs:   ValidationErrors{},
			expect: "no errors",
		},
		{
			name: "single error",
			errs: ValidationErrors{
				{Field: "name", Message: "is required"},
			},
			expect: "name: is required",
		},
		{
			name: "multiple errors",
			errs: ValidationErrors{
				{Field: "name", Message: "is required"},
				{Field: "version", Message: "invalid format", Value: "1.0"},
			},
			expect: "2 validation errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.errs.Error()
			if !strings.Contains(result, tt.expect) {
				t.Errorf("Error() = %q, want to contain %q", result, tt.expect)
			}
		})
	}
}

func TestIsValidFormatID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"test", true},
		{"test-pattern", true},
		{"us-code", true},
		{"eu-directive-2016", true},
		{"a", true},
		{"a1", true},
		{"", false},
		{"Test", false},
		{"test_pattern", false},
		{"123test", false},
		{"-test", false},
		{"test-", true}, // trailing hyphen is technically valid
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			if got := isValidFormatID(tt.id); got != tt.want {
				t.Errorf("isValidFormatID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestIsValidVersion(t *testing.T) {
	tests := []struct {
		v    string
		want bool
	}{
		{"1.0.0", true},
		{"0.0.1", true},
		{"10.20.30", true},
		{"1.0", false},
		{"1", false},
		{"1.0.0.0", false},
		{"v1.0.0", false},
		{"1.0.0-beta", false},
		{"", false},
		{"...", false},
		{"1.0.", false},
	}

	for _, tt := range tests {
		t.Run(tt.v, func(t *testing.T) {
			if got := isValidVersion(tt.v); got != tt.want {
				t.Errorf("isValidVersion(%q) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}

func TestGetEmbeddedSchema(t *testing.T) {
	schema, err := GetEmbeddedSchema()
	if err != nil {
		t.Fatalf("GetEmbeddedSchema() error = %v", err)
	}

	// Check that it's a valid JSON Schema
	if schema["$schema"] == nil {
		t.Error("Schema missing $schema field")
	}

	if schema["title"] == nil {
		t.Error("Schema missing title field")
	}

	if schema["properties"] == nil {
		t.Error("Schema missing properties field")
	}

	// Check required fields are defined
	required, ok := schema["required"].([]interface{})
	if !ok {
		t.Fatal("Schema missing required array")
	}

	expectedRequired := map[string]bool{
		"name": false, "format_id": false, "version": false, "detection": false,
	}
	for _, r := range required {
		if s, ok := r.(string); ok {
			expectedRequired[s] = true
		}
	}

	for field, found := range expectedRequired {
		if !found {
			t.Errorf("Expected %q in required fields", field)
		}
	}
}
