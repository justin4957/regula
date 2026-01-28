package validate

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/coolbeans/regula/pkg/extract"
	"github.com/coolbeans/regula/pkg/store"
)

// ValidationGate represents a validation checkpoint in the ingestion pipeline.
// Each gate evaluates stage-specific quality metrics against configurable thresholds.
type ValidationGate interface {
	// Name returns the unique identifier for this gate (e.g., "V0", "V1", "V2", "V3").
	Name() string

	// Run executes the gate's validation logic against the provided context.
	Run(ctx *ValidationContext) *GateResult

	// Thresholds returns the default thresholds for this gate's metrics.
	// Keys are metric names, values are minimum acceptable scores (0.0-1.0).
	Thresholds() map[string]float64
}

// ValidationContext provides all data available to a gate at a given pipeline stage.
// Fields are populated incrementally as the pipeline progresses — earlier gates
// will see nil/zero for fields that haven't been produced yet.
type ValidationContext struct {
	// Document is available after parsing (V1+).
	Document *extract.Document

	// References is available after reference extraction (V2+).
	References []*extract.Reference

	// Definitions is available after definition extraction (V2+).
	Definitions []*extract.DefinedTerm

	// Semantics is available after semantic extraction (V2+).
	Semantics []*extract.SemanticAnnotation

	// TermUsages is available after term usage extraction (V2+).
	TermUsages []*extract.TermUsage

	// ResolvedReferences is available after resolution (V3).
	ResolvedReferences []*extract.ResolvedReference

	// TripleStore is available after graph building (V3).
	TripleStore *store.TripleStore

	// Config holds user-provided thresholds and behavior flags.
	Config *ValidationConfig

	// SourcePath is the path to the source file (V0+).
	SourcePath string

	// SourceSize is the file size in bytes (V0+).
	SourceSize int64

	// ParseDuration is how long parsing took (V1+).
	ParseDuration time.Duration
}

// ValidationConfig holds user-configurable settings for gate execution.
type ValidationConfig struct {
	// Thresholds overrides per-gate metric thresholds.
	// Key format: "GateName.MetricName" (e.g., "V0.file_size", "V2.definition_coverage").
	Thresholds map[string]float64

	// SkipGates lists gate names to skip entirely (e.g., ["V0", "V2"]).
	SkipGates []string

	// StrictMode causes the pipeline to halt on gate failure.
	StrictMode bool

	// FailOnWarn causes the pipeline to halt on any warning.
	FailOnWarn bool
}

// DefaultValidationConfig returns a config with no overrides and default behavior.
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		Thresholds: make(map[string]float64),
		SkipGates:  make([]string, 0),
		StrictMode: false,
		FailOnWarn: false,
	}
}

// GateResult captures the outcome of a single gate execution.
type GateResult struct {
	Gate       string             `json:"gate"`
	Passed     bool               `json:"passed"`
	Score      float64            `json:"score"`
	Metrics    map[string]float64 `json:"metrics"`
	Warnings   []GateWarning      `json:"warnings,omitempty"`
	Errors     []GateError        `json:"errors,omitempty"`
	Duration   time.Duration      `json:"duration"`
	Skipped    bool               `json:"skipped,omitempty"`
	SkipReason string             `json:"skip_reason,omitempty"`
}

// GateWarning represents a non-fatal issue detected by a gate.
type GateWarning struct {
	Metric  string  `json:"metric"`
	Message string  `json:"message"`
	Value   float64 `json:"value,omitempty"`
}

// GateError represents a fatal issue detected by a gate.
type GateError struct {
	Metric  string  `json:"metric"`
	Message string  `json:"message"`
	Value   float64 `json:"value,omitempty"`
}

// GateReport aggregates results from all gates in a pipeline run.
type GateReport struct {
	Results      []*GateResult `json:"results"`
	OverallPass  bool          `json:"overall_pass"`
	TotalScore   float64       `json:"total_score"`
	GatesPassed  int           `json:"gates_passed"`
	GatesFailed  int           `json:"gates_failed"`
	GatesSkipped int           `json:"gates_skipped"`
	Duration     time.Duration `json:"duration"`
	HaltedAt     string        `json:"halted_at,omitempty"`
}

// ToJSON serializes the gate report as indented JSON.
func (gateReport *GateReport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(gateReport, "", "  ")
}

// String returns a human-readable gate report.
func (gateReport *GateReport) String() string {
	var reportBuilder strings.Builder

	reportBuilder.WriteString("Validation Gate Report\n")
	reportBuilder.WriteString("======================\n\n")

	for _, gateResult := range gateReport.Results {
		statusLabel := "PASS"
		if gateResult.Skipped {
			statusLabel = "SKIP"
		} else if !gateResult.Passed {
			statusLabel = "FAIL"
		}

		reportBuilder.WriteString(fmt.Sprintf("[%s] Gate %s (score: %.1f%%, %v)\n",
			statusLabel, gateResult.Gate, gateResult.Score*100, gateResult.Duration))

		if gateResult.Skipped {
			reportBuilder.WriteString(fmt.Sprintf("  Reason: %s\n", gateResult.SkipReason))
		}

		for metricName, metricValue := range gateResult.Metrics {
			reportBuilder.WriteString(fmt.Sprintf("  %s: %.1f%%\n", metricName, metricValue*100))
		}

		for _, gateWarning := range gateResult.Warnings {
			reportBuilder.WriteString(fmt.Sprintf("  WARNING [%s]: %s\n", gateWarning.Metric, gateWarning.Message))
		}

		for _, gateError := range gateResult.Errors {
			reportBuilder.WriteString(fmt.Sprintf("  ERROR [%s]: %s\n", gateError.Metric, gateError.Message))
		}

		reportBuilder.WriteString("\n")
	}

	reportBuilder.WriteString(fmt.Sprintf("Summary: %d passed, %d failed, %d skipped\n",
		gateReport.GatesPassed, gateReport.GatesFailed, gateReport.GatesSkipped))
	reportBuilder.WriteString(fmt.Sprintf("Overall Score: %.1f%%\n", gateReport.TotalScore*100))

	overallStatus := "PASS"
	if !gateReport.OverallPass {
		overallStatus = "FAIL"
	}
	reportBuilder.WriteString(fmt.Sprintf("Status: %s\n", overallStatus))

	if gateReport.HaltedAt != "" {
		reportBuilder.WriteString(fmt.Sprintf("Pipeline halted at: %s\n", gateReport.HaltedAt))
	}

	reportBuilder.WriteString(fmt.Sprintf("Total Duration: %v\n", gateReport.Duration))

	return reportBuilder.String()
}

// GatePipeline executes validation gates in sequence and collects results.
type GatePipeline struct {
	gates  []ValidationGate
	config *ValidationConfig
}

// NewGatePipeline creates a pipeline with the given configuration.
func NewGatePipeline(config *ValidationConfig) *GatePipeline {
	if config == nil {
		config = DefaultValidationConfig()
	}
	return &GatePipeline{
		gates:  make([]ValidationGate, 0),
		config: config,
	}
}

// RegisterGate adds a gate to the pipeline. Gates execute in registration order.
func (gatePipeline *GatePipeline) RegisterGate(gate ValidationGate) {
	gatePipeline.gates = append(gatePipeline.gates, gate)
}

// RegisterDefaultGates registers the four standard gates (V0-V3).
func (gatePipeline *GatePipeline) RegisterDefaultGates() {
	gatePipeline.RegisterGate(NewSchemaGate())
	gatePipeline.RegisterGate(NewStructureGate())
	gatePipeline.RegisterGate(NewCoverageGate())
	gatePipeline.RegisterGate(NewQualityGate())
}

// Run executes all registered gates in order against the provided context.
// If StrictMode is set in config, the pipeline halts on the first gate failure.
// If FailOnWarn is set, the pipeline halts on any warning.
// Gates listed in SkipGates are skipped with a recorded skip result.
func (gatePipeline *GatePipeline) Run(ctx *ValidationContext) *GateReport {
	pipelineStartTime := time.Now()

	gateReport := &GateReport{
		Results:     make([]*GateResult, 0, len(gatePipeline.gates)),
		OverallPass: true,
	}

	for _, gate := range gatePipeline.gates {
		if gatePipeline.isGateSkipped(gate.Name()) {
			skipResult := &GateResult{
				Gate:       gate.Name(),
				Skipped:    true,
				SkipReason: "skipped by configuration",
				Metrics:    make(map[string]float64),
			}
			gateReport.Results = append(gateReport.Results, skipResult)
			gateReport.GatesSkipped++
			continue
		}

		gateResult := gate.Run(ctx)
		gateReport.Results = append(gateReport.Results, gateResult)

		if gateResult.Passed {
			gateReport.GatesPassed++
		} else {
			gateReport.GatesFailed++
			gateReport.OverallPass = false

			if gatePipeline.config.StrictMode {
				gateReport.HaltedAt = gate.Name()
				break
			}
		}

		if gatePipeline.config.FailOnWarn && len(gateResult.Warnings) > 0 {
			gateReport.OverallPass = false
			gateReport.HaltedAt = gate.Name()
			break
		}
	}

	// Calculate total score from non-skipped gates.
	scoredGateCount := 0
	totalScore := 0.0
	for _, gateResult := range gateReport.Results {
		if !gateResult.Skipped {
			totalScore += gateResult.Score
			scoredGateCount++
		}
	}
	if scoredGateCount > 0 {
		gateReport.TotalScore = totalScore / float64(scoredGateCount)
	}

	gateReport.Duration = time.Since(pipelineStartTime)
	return gateReport
}

// RunGate executes a single named gate. Useful for running individual checkpoints
// at specific pipeline stages rather than all gates at once.
// Returns nil if the gate is not found or is configured to be skipped.
func (gatePipeline *GatePipeline) RunGate(gateName string, ctx *ValidationContext) *GateResult {
	if gatePipeline.isGateSkipped(gateName) {
		return &GateResult{
			Gate:       gateName,
			Skipped:    true,
			SkipReason: "skipped by configuration",
			Metrics:    make(map[string]float64),
		}
	}

	for _, gate := range gatePipeline.gates {
		if gate.Name() == gateName {
			return gate.Run(ctx)
		}
	}

	return nil
}

// isGateSkipped checks if a gate should be skipped per configuration.
func (gatePipeline *GatePipeline) isGateSkipped(gateName string) bool {
	for _, skipName := range gatePipeline.config.SkipGates {
		if strings.EqualFold(skipName, gateName) {
			return true
		}
	}
	return false
}

// effectiveThreshold returns the threshold for a metric, checking config overrides
// first, then falling back to the gate's default thresholds.
func effectiveThreshold(config *ValidationConfig, gate ValidationGate, metricName string) float64 {
	if config != nil && config.Thresholds != nil {
		configKey := gate.Name() + "." + metricName
		if threshold, exists := config.Thresholds[configKey]; exists {
			return threshold
		}
	}
	defaults := gate.Thresholds()
	if threshold, exists := defaults[metricName]; exists {
		return threshold
	}
	return 0.80
}

// evaluateMetrics is a helper that computes the gate score and populates
// warnings/errors based on metrics vs thresholds.
func evaluateMetrics(gateResult *GateResult, config *ValidationConfig, gate ValidationGate) {
	metricCount := len(gateResult.Metrics)
	if metricCount == 0 {
		gateResult.Score = 1.0
		gateResult.Passed = true
		return
	}

	totalScore := 0.0
	allPassed := true

	for metricName, metricValue := range gateResult.Metrics {
		threshold := effectiveThreshold(config, gate, metricName)
		totalScore += metricValue

		if metricValue < threshold {
			allPassed = false
			gateResult.Errors = append(gateResult.Errors, GateError{
				Metric:  metricName,
				Message: fmt.Sprintf("%s (%.1f%%) below threshold (%.1f%%)", metricName, metricValue*100, threshold*100),
				Value:   metricValue,
			})
		} else if metricValue < threshold*1.1 {
			// Within 10% of threshold — emit warning.
			gateResult.Warnings = append(gateResult.Warnings, GateWarning{
				Metric:  metricName,
				Message: fmt.Sprintf("%s (%.1f%%) close to threshold (%.1f%%)", metricName, metricValue*100, threshold*100),
				Value:   metricValue,
			})
		}
	}

	gateResult.Score = totalScore / float64(metricCount)
	gateResult.Passed = allPassed
}
