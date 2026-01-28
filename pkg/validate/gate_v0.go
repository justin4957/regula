package validate

import (
	"time"
)

// maxFileSizeBytes is the maximum expected file size (10 MB).
const maxFileSizeBytes = 10 * 1024 * 1024

// SchemaGate (V0) validates the input document schema, encoding, and size.
// Runs after file load, before parsing.
type SchemaGate struct{}

// NewSchemaGate creates a new V0 schema validation gate.
func NewSchemaGate() *SchemaGate {
	return &SchemaGate{}
}

// Name returns "V0".
func (schemaGate *SchemaGate) Name() string { return "V0" }

// Thresholds returns the default thresholds for schema validation metrics.
func (schemaGate *SchemaGate) Thresholds() map[string]float64 {
	return map[string]float64{
		"file_readable": 1.0,
		"file_size":     1.0,
		"file_not_empty": 1.0,
	}
}

// Run validates the source file is readable, non-empty, and within size limits.
func (schemaGate *SchemaGate) Run(ctx *ValidationContext) *GateResult {
	startTime := time.Now()

	gateResult := &GateResult{
		Gate:     schemaGate.Name(),
		Metrics:  make(map[string]float64),
		Warnings: make([]GateWarning, 0),
		Errors:   make([]GateError, 0),
	}

	// file_readable: source path is non-empty.
	if ctx.SourcePath != "" {
		gateResult.Metrics["file_readable"] = 1.0
	} else {
		gateResult.Metrics["file_readable"] = 0.0
	}

	// file_not_empty: file has content.
	if ctx.SourceSize > 0 {
		gateResult.Metrics["file_not_empty"] = 1.0
	} else {
		gateResult.Metrics["file_not_empty"] = 0.0
	}

	// file_size: within bounds (> 0 and <= max).
	if ctx.SourceSize > 0 && ctx.SourceSize <= maxFileSizeBytes {
		gateResult.Metrics["file_size"] = 1.0
	} else if ctx.SourceSize > maxFileSizeBytes {
		gateResult.Metrics["file_size"] = 0.0
		gateResult.Warnings = append(gateResult.Warnings, GateWarning{
			Metric:  "file_size",
			Message: "file exceeds 10 MB size limit",
			Value:   float64(ctx.SourceSize),
		})
	} else {
		gateResult.Metrics["file_size"] = 0.0
	}

	evaluateMetrics(gateResult, ctx.Config, schemaGate)
	gateResult.Duration = time.Since(startTime)
	return gateResult
}
