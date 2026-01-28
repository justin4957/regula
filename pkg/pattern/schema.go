package pattern

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed schema.json
var schemaFS embed.FS

// SchemaVersion is the current pattern schema version
const SchemaVersion = "1.0.0"

// ValidationError represents a schema validation error with context
type ValidationError struct {
	Field   string
	Message string
	Value   interface{}
}

func (e ValidationError) Error() string {
	if e.Value != nil {
		return fmt.Sprintf("%s: %s (got: %v)", e.Field, e.Message, e.Value)
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

func (errs ValidationErrors) Error() string {
	if len(errs) == 0 {
		return "no errors"
	}
	if len(errs) == 1 {
		return errs[0].Error()
	}
	messages := make([]string, len(errs))
	for i, err := range errs {
		messages[i] = err.Error()
	}
	return fmt.Sprintf("%d validation errors:\n  - %s", len(errs), strings.Join(messages, "\n  - "))
}

// ValidateSchema performs comprehensive validation of a FormatPattern.
// It returns descriptive errors for all validation failures.
func ValidateSchema(fp *FormatPattern) ValidationErrors {
	var errs ValidationErrors

	// Required field validation
	if fp.Name == "" {
		errs = append(errs, ValidationError{
			Field:   "name",
			Message: "required field is missing",
		})
	}

	if fp.FormatID == "" {
		errs = append(errs, ValidationError{
			Field:   "format_id",
			Message: "required field is missing",
		})
	} else if !isValidFormatID(fp.FormatID) {
		errs = append(errs, ValidationError{
			Field:   "format_id",
			Message: "must be lowercase alphanumeric with hyphens, starting with a letter",
			Value:   fp.FormatID,
		})
	}

	if fp.Version == "" {
		errs = append(errs, ValidationError{
			Field:   "version",
			Message: "required field is missing",
		})
	} else if !isValidVersion(fp.Version) {
		errs = append(errs, ValidationError{
			Field:   "version",
			Message: "must be semantic version (e.g., 1.0.0)",
			Value:   fp.Version,
		})
	}

	// Detection validation
	errs = append(errs, validateDetection(&fp.Detection)...)

	// Structure validation
	errs = append(errs, validateStructure(&fp.Structure)...)

	// Definitions validation
	errs = append(errs, validateDefinitions(&fp.Definitions)...)

	// References validation
	errs = append(errs, validateReferences(&fp.References)...)

	return errs
}

func validateDetection(d *DetectionConfig) ValidationErrors {
	var errs ValidationErrors

	if len(d.RequiredIndicators) == 0 {
		errs = append(errs, ValidationError{
			Field:   "detection.required_indicators",
			Message: "at least one required indicator is needed",
		})
	}

	for i, ind := range d.RequiredIndicators {
		field := fmt.Sprintf("detection.required_indicators[%d]", i)
		errs = append(errs, validateIndicator(field, &ind, true)...)
	}

	for i, ind := range d.OptionalIndicators {
		field := fmt.Sprintf("detection.optional_indicators[%d]", i)
		errs = append(errs, validateIndicator(field, &ind, true)...)
	}

	for i, ind := range d.NegativeIndicators {
		field := fmt.Sprintf("detection.negative_indicators[%d]", i)
		errs = append(errs, validateIndicator(field, &ind, false)...)
	}

	return errs
}

func validateIndicator(field string, ind *Indicator, positiveWeight bool) ValidationErrors {
	var errs ValidationErrors

	if ind.Pattern == "" {
		errs = append(errs, ValidationError{
			Field:   field + ".pattern",
			Message: "pattern is required",
		})
	}

	if positiveWeight {
		if ind.Weight < 1 {
			errs = append(errs, ValidationError{
				Field:   field + ".weight",
				Message: "weight must be positive (>= 1)",
				Value:   ind.Weight,
			})
		}
		if ind.Weight > 100 {
			errs = append(errs, ValidationError{
				Field:   field + ".weight",
				Message: "weight must be <= 100",
				Value:   ind.Weight,
			})
		}
	} else {
		if ind.Weight > -1 {
			errs = append(errs, ValidationError{
				Field:   field + ".weight",
				Message: "negative indicator weight must be negative (<= -1)",
				Value:   ind.Weight,
			})
		}
		if ind.Weight < -100 {
			errs = append(errs, ValidationError{
				Field:   field + ".weight",
				Message: "weight must be >= -100",
				Value:   ind.Weight,
			})
		}
	}

	return errs
}

func validateStructure(s *StructureConfig) ValidationErrors {
	var errs ValidationErrors

	validTypes := map[string]bool{
		"title": true, "part": true, "chapter": true, "subchapter": true,
		"division": true, "section": true, "subsection": true, "article": true,
		"paragraph": true, "subparagraph": true, "point": true, "subpoint": true,
		"subdivision": true, "schedule": true, "annex": true,
	}

	for i, level := range s.Hierarchy {
		field := fmt.Sprintf("structure.hierarchy[%d]", i)

		if level.Type == "" {
			errs = append(errs, ValidationError{
				Field:   field + ".type",
				Message: "type is required",
			})
		} else if !validTypes[level.Type] {
			errs = append(errs, ValidationError{
				Field:   field + ".type",
				Message: "invalid hierarchy type",
				Value:   level.Type,
			})
		}

		if level.Pattern == "" {
			errs = append(errs, ValidationError{
				Field:   field + ".pattern",
				Message: "pattern is required",
			})
		}

		if level.NumberGroup < 0 {
			errs = append(errs, ValidationError{
				Field:   field + ".number_group",
				Message: "must be non-negative",
				Value:   level.NumberGroup,
			})
		}
	}

	return errs
}

func validateDefinitions(d *DefinitionConfig) ValidationErrors {
	var errs ValidationErrors

	for i, loc := range d.Location {
		field := fmt.Sprintf("definitions.location[%d]", i)

		if loc.SectionNumber == 0 && loc.SectionTitle == "" {
			errs = append(errs, ValidationError{
				Field:   field,
				Message: "must specify either section_number or section_title",
			})
		}
	}

	return errs
}

func validateReferences(r *ReferenceConfig) ValidationErrors {
	var errs ValidationErrors

	validTargets := map[string]bool{
		"title": true, "part": true, "chapter": true, "section": true,
		"subsection": true, "article": true, "paragraph": true, "point": true,
		"subdivision": true, "recital": true, "schedule": true, "annex": true,
	}

	for i, ref := range r.Internal {
		field := fmt.Sprintf("references.internal[%d]", i)

		if ref.Pattern == "" {
			errs = append(errs, ValidationError{
				Field:   field + ".pattern",
				Message: "pattern is required",
			})
		}

		if ref.Target == "" {
			errs = append(errs, ValidationError{
				Field:   field + ".target",
				Message: "target is required",
			})
		} else if !validTargets[ref.Target] {
			errs = append(errs, ValidationError{
				Field:   field + ".target",
				Message: "invalid target type",
				Value:   ref.Target,
			})
		}
	}

	for i, ref := range r.External {
		field := fmt.Sprintf("references.external[%d]", i)

		if ref.Pattern == "" {
			errs = append(errs, ValidationError{
				Field:   field + ".pattern",
				Message: "pattern is required",
			})
		}

		if ref.Type == "" {
			errs = append(errs, ValidationError{
				Field:   field + ".type",
				Message: "type is required",
			})
		}
	}

	return errs
}

func isValidFormatID(id string) bool {
	if len(id) == 0 {
		return false
	}
	// Must start with lowercase letter
	if id[0] < 'a' || id[0] > 'z' {
		return false
	}
	// Rest must be lowercase alphanumeric or hyphen
	for _, c := range id[1:] {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return true
}

func isValidVersion(v string) bool {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if len(part) == 0 {
			return false
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}

// GetEmbeddedSchema returns the embedded JSON Schema as a map
func GetEmbeddedSchema() (map[string]interface{}, error) {
	data, err := schemaFS.ReadFile("schema.json")
	if err != nil {
		return nil, fmt.Errorf("reading embedded schema: %w", err)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("parsing embedded schema: %w", err)
	}

	return schema, nil
}
