package pattern

import (
	"testing"
)

func TestFormatPatternValidate(t *testing.T) {
	tests := []struct {
		name      string
		pattern   FormatPattern
		wantError bool
	}{
		{
			name: "valid pattern",
			pattern: FormatPattern{
				Name:     "Test Pattern",
				FormatID: "test-format",
				Version:  "1.0.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: "test.*pattern", Weight: 10},
					},
				},
			},
			wantError: false,
		},
		{
			name: "missing name",
			pattern: FormatPattern{
				FormatID: "test-format",
				Version:  "1.0.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: "test", Weight: 10},
					},
				},
			},
			wantError: true,
		},
		{
			name: "missing format_id",
			pattern: FormatPattern{
				Name:    "Test Pattern",
				Version: "1.0.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: "test", Weight: 10},
					},
				},
			},
			wantError: true,
		},
		{
			name: "missing version",
			pattern: FormatPattern{
				Name:     "Test Pattern",
				FormatID: "test-format",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: "test", Weight: 10},
					},
				},
			},
			wantError: true,
		},
		{
			name: "no required indicators",
			pattern: FormatPattern{
				Name:     "Test Pattern",
				FormatID: "test-format",
				Version:  "1.0.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pattern.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestFormatPatternCompile(t *testing.T) {
	tests := []struct {
		name      string
		pattern   FormatPattern
		wantError bool
	}{
		{
			name: "valid regex patterns",
			pattern: FormatPattern{
				Name:     "Test Pattern",
				FormatID: "test-format",
				Version:  "1.0.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: `\bSection\s+\d+`, Weight: 10},
					},
					OptionalIndicators: []Indicator{
						{Pattern: `Article\s+[IVXLCDM]+`, Weight: 5},
					},
					NegativeIndicators: []Indicator{
						{Pattern: `DRAFT|UNOFFICIAL`, Weight: -5},
					},
				},
				Structure: StructureConfig{
					Hierarchy: []HierarchyLevel{
						{Type: "section", Pattern: `^Section\s+(\d+)\.`, NumberGroup: 1},
					},
				},
			},
			wantError: false,
		},
		{
			name: "invalid required indicator regex",
			pattern: FormatPattern{
				Name:     "Test Pattern",
				FormatID: "test-format",
				Version:  "1.0.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: `[invalid`, Weight: 10},
					},
				},
			},
			wantError: true,
		},
		{
			name: "invalid optional indicator regex",
			pattern: FormatPattern{
				Name:     "Test Pattern",
				FormatID: "test-format",
				Version:  "1.0.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: `valid`, Weight: 10},
					},
					OptionalIndicators: []Indicator{
						{Pattern: `(unclosed`, Weight: 5},
					},
				},
			},
			wantError: true,
		},
		{
			name: "invalid hierarchy pattern",
			pattern: FormatPattern{
				Name:     "Test Pattern",
				FormatID: "test-format",
				Version:  "1.0.0",
				Detection: DetectionConfig{
					RequiredIndicators: []Indicator{
						{Pattern: `valid`, Weight: 10},
					},
				},
				Structure: StructureConfig{
					Hierarchy: []HierarchyLevel{
						{Type: "section", Pattern: `[bad`},
					},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pattern.Compile()
			if (err != nil) != tt.wantError {
				t.Errorf("Compile() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestFormatPatternIsCompiled(t *testing.T) {
	pattern := FormatPattern{
		Name:     "Test Pattern",
		FormatID: "test-format",
		Version:  "1.0.0",
		Detection: DetectionConfig{
			RequiredIndicators: []Indicator{
				{Pattern: `test`, Weight: 10},
			},
		},
	}

	if pattern.IsCompiled() {
		t.Error("IsCompiled() should return false before compilation")
	}

	if err := pattern.Compile(); err != nil {
		t.Fatalf("Compile() unexpected error: %v", err)
	}

	if !pattern.IsCompiled() {
		t.Error("IsCompiled() should return true after compilation")
	}
}

func TestFormatPatternGetHierarchyLevel(t *testing.T) {
	pattern := FormatPattern{
		Name:     "Test Pattern",
		FormatID: "test-format",
		Version:  "1.0.0",
		Detection: DetectionConfig{
			RequiredIndicators: []Indicator{
				{Pattern: `test`, Weight: 10},
			},
		},
		Structure: StructureConfig{
			Hierarchy: []HierarchyLevel{
				{Type: "chapter", Pattern: `^Chapter\s+(\d+)`},
				{Type: "section", Pattern: `^Section\s+(\d+)`},
				{Type: "paragraph", Pattern: `^\(([a-z])\)`},
			},
		},
	}

	tests := []struct {
		levelType string
		want      bool
	}{
		{"chapter", true},
		{"section", true},
		{"paragraph", true},
		{"article", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.levelType, func(t *testing.T) {
			level := pattern.GetHierarchyLevel(tt.levelType)
			if (level != nil) != tt.want {
				t.Errorf("GetHierarchyLevel(%q) found = %v, want %v", tt.levelType, level != nil, tt.want)
			}
		})
	}
}
