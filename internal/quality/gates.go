package quality

import (
	"context"
	"fmt"
	"time"
)

// QualityGate represents a quality check that must pass
type QualityGate struct {
	ID          string                                                                        `json:"id"`
	Name        string                                                                        `json:"name"`
	Description string                                                                        `json:"description"`
	Type        GateType                                                                      `json:"type"`
	Severity    Severity                                                                      `json:"severity"`
	Threshold   float64                                                                       `json:"threshold"`
	Enabled     bool                                                                          `json:"enabled"`
	Config      map[string]interface{}                                                        `json:"config"`
	Check       func(ctx context.Context, config map[string]interface{}) (*GateResult, error) `json:"-"`
}

// GateType represents the type of quality gate
type GateType string

const (
	GateTypeSecurity      GateType = "security"
	GateTypePerformance   GateType = "performance"
	GateTypeCoverage      GateType = "coverage"
	GateTypeComplexity    GateType = "complexity"
	GateTypeDependencies  GateType = "dependencies"
	GateTypeDocumentation GateType = "documentation"
	GateTypeStandards     GateType = "standards"
)

// Severity represents the severity level of a gate
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// GateResult contains the result of a quality gate check
type GateResult struct {
	GateID      string                 `json:"gate_id"`
	Passed      bool                   `json:"passed"`
	Score       float64                `json:"score"`
	Threshold   float64                `json:"threshold"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details"`
	Duration    time.Duration          `json:"duration"`
	Timestamp   time.Time              `json:"timestamp"`
	Violations  []Violation            `json:"violations"`
	Suggestions []string               `json:"suggestions"`
}

// Violation represents a specific quality violation
type Violation struct {
	Type     string                 `json:"type"`
	Severity Severity               `json:"severity"`
	File     string                 `json:"file"`
	Line     int                    `json:"line"`
	Column   int                    `json:"column"`
	Message  string                 `json:"message"`
	Rule     string                 `json:"rule"`
	Details  map[string]interface{} `json:"details"`
	Fix      string                 `json:"fix"`
}

// QualityGateManager manages and executes quality gates
type QualityGateManager struct {
	gates   map[string]*QualityGate
	results map[string][]*GateResult
	config  *ManagerConfig
}

// ManagerConfig contains configuration for the quality gate manager
type ManagerConfig struct {
	Timeout       time.Duration `json:"timeout"`
	Parallel      bool          `json:"parallel"`
	FailFast      bool          `json:"fail_fast"`
	ReportFormat  string        `json:"report_format"`
	OutputPath    string        `json:"output_path"`
	EnableCaching bool          `json:"enable_caching"`
	CacheTTL      time.Duration `json:"cache_ttl"`
}

// NewQualityGateManager creates a new quality gate manager
func NewQualityGateManager(config *ManagerConfig) *QualityGateManager {
	if config == nil {
		config = getDefaultManagerConfig()
	}

	manager := &QualityGateManager{
		gates:   make(map[string]*QualityGate),
		results: make(map[string][]*GateResult),
		config:  config,
	}

	// Register default gates
	manager.registerDefaultGates()

	return manager
}

// RegisterGate registers a new quality gate
func (qgm *QualityGateManager) RegisterGate(gate *QualityGate) error {
	if gate.ID == "" {
		return fmt.Errorf("gate ID cannot be empty")
	}

	if gate.Check == nil {
		return fmt.Errorf("gate check function cannot be nil")
	}

	qgm.gates[gate.ID] = gate
	return nil
}

// ExecuteGates executes all enabled quality gates
func (qgm *QualityGateManager) ExecuteGates(ctx context.Context) (*QualityReport, error) {
	start := time.Now()

	// Get enabled gates
	enabledGates := qgm.getEnabledGates()
	if len(enabledGates) == 0 {
		return &QualityReport{
			OverallPassed: true,
			Message:       "No quality gates enabled",
			Duration:      time.Since(start),
		}, nil
	}

	// Execute gates
	var results []*GateResult
	var criticalFailures []string

	for _, gate := range enabledGates {
		gateCtx, cancel := context.WithTimeout(ctx, qgm.config.Timeout)
		defer cancel()

		result, err := qgm.executeGate(gateCtx, gate)
		if err != nil {
			if qgm.config.FailFast {
				return nil, fmt.Errorf("gate %s failed: %w", gate.ID, err)
			}

			// Create failure result
			result = &GateResult{
				GateID:    gate.ID,
				Passed:    false,
				Message:   fmt.Sprintf("Gate execution failed: %v", err),
				Timestamp: time.Now(),
			}
		}

		results = append(results, result)

		// Track critical failures
		if !result.Passed && gate.Severity == SeverityCritical {
			criticalFailures = append(criticalFailures, gate.ID)
		}

		// Fail fast on critical failures
		if qgm.config.FailFast && !result.Passed && gate.Severity == SeverityCritical {
			break
		}
	}

	// Generate report
	report := qgm.generateReport(results, time.Since(start))

	// Store results
	qgm.results[time.Now().Format("2006-01-02T15:04:05")] = results

	return report, nil
}

// ExecuteGate executes a specific quality gate
func (qgm *QualityGateManager) ExecuteGate(ctx context.Context, gateID string) (*GateResult, error) {
	gate, exists := qgm.gates[gateID]
	if !exists {
		return nil, fmt.Errorf("gate %s not found", gateID)
	}

	if !gate.Enabled {
		return &GateResult{
			GateID:    gate.ID,
			Passed:    true,
			Message:   "Gate is disabled",
			Timestamp: time.Now(),
		}, nil
	}

	return qgm.executeGate(ctx, gate)
}

// QualityReport contains the overall quality assessment
type QualityReport struct {
	OverallPassed   bool           `json:"overall_passed"`
	OverallScore    float64        `json:"overall_score"`
	Message         string         `json:"message"`
	Duration        time.Duration  `json:"duration"`
	GateResults     []*GateResult  `json:"gate_results"`
	Summary         QualitySummary `json:"summary"`
	Recommendations []string       `json:"recommendations"`
	Timestamp       time.Time      `json:"timestamp"`
}

// QualitySummary provides a summary of quality metrics
type QualitySummary struct {
	TotalGates       int `json:"total_gates"`
	PassedGates      int `json:"passed_gates"`
	FailedGates      int `json:"failed_gates"`
	CriticalFailures int `json:"critical_failures"`
	HighFailures     int `json:"high_failures"`
	MediumFailures   int `json:"medium_failures"`
	LowFailures      int `json:"low_failures"`
	TotalViolations  int `json:"total_violations"`
}

// Helper methods

func (qgm *QualityGateManager) getEnabledGates() []*QualityGate {
	var enabled []*QualityGate
	for _, gate := range qgm.gates {
		if gate.Enabled {
			enabled = append(enabled, gate)
		}
	}
	return enabled
}

func (qgm *QualityGateManager) executeGate(ctx context.Context, gate *QualityGate) (*GateResult, error) {
	start := time.Now()

	result, err := gate.Check(ctx, gate.Config)
	if err != nil {
		return nil, err
	}

	result.GateID = gate.ID
	result.Threshold = gate.Threshold
	result.Duration = time.Since(start)
	result.Timestamp = time.Now()

	// Determine if gate passed
	result.Passed = result.Score >= gate.Threshold

	return result, nil
}

func (qgm *QualityGateManager) generateReport(results []*GateResult, duration time.Duration) *QualityReport {
	report := &QualityReport{
		GateResults: results,
		Duration:    duration,
		Timestamp:   time.Now(),
	}

	// Calculate summary
	summary := QualitySummary{
		TotalGates: len(results),
	}

	var totalScore float64
	var recommendations []string

	for _, result := range results {
		if result.Passed {
			summary.PassedGates++
		} else {
			summary.FailedGates++

			// Count violations by severity
			for _, violation := range result.Violations {
				summary.TotalViolations++
				switch violation.Severity {
				case SeverityCritical:
					summary.CriticalFailures++
				case SeverityHigh:
					summary.HighFailures++
				case SeverityMedium:
					summary.MediumFailures++
				case SeverityLow:
					summary.LowFailures++
				}
			}

			// Add recommendations
			recommendations = append(recommendations, result.Suggestions...)
		}

		totalScore += result.Score
	}

	// Calculate overall score
	if len(results) > 0 {
		report.OverallScore = totalScore / float64(len(results))
	}

	// Determine overall pass/fail
	report.OverallPassed = summary.CriticalFailures == 0 && summary.HighFailures == 0

	// Generate message
	if report.OverallPassed {
		report.Message = fmt.Sprintf("All quality gates passed (Score: %.2f)", report.OverallScore)
	} else {
		report.Message = fmt.Sprintf("Quality gates failed: %d critical, %d high violations",
			summary.CriticalFailures, summary.HighFailures)
	}

	report.Summary = summary
	report.Recommendations = recommendations

	return report
}

// registerDefaultGates registers the default set of quality gates
func (qgm *QualityGateManager) registerDefaultGates() {
	// Security gates
	qgm.RegisterGate(&QualityGate{
		ID:          "security-scan",
		Name:        "Security Scan",
		Description: "Comprehensive security vulnerability scan",
		Type:        GateTypeSecurity,
		Severity:    SeverityCritical,
		Threshold:   0.0, // No vulnerabilities allowed
		Enabled:     true,
		Config: map[string]interface{}{
			"tools": []string{"gosec", "semgrep", "govulncheck"},
		},
		Check: qgm.checkSecurityScan,
	})

	// Test coverage gate
	qgm.RegisterGate(&QualityGate{
		ID:          "test-coverage",
		Name:        "Test Coverage",
		Description: "Minimum test coverage requirement",
		Type:        GateTypeCoverage,
		Severity:    SeverityHigh,
		Threshold:   80.0, // 80% coverage required
		Enabled:     true,
		Config: map[string]interface{}{
			"minimum_coverage": 80.0,
		},
		Check: qgm.checkTestCoverage,
	})

	// Performance gate
	qgm.RegisterGate(&QualityGate{
		ID:          "performance-benchmarks",
		Name:        "Performance Benchmarks",
		Description: "Performance benchmark validation",
		Type:        GateTypePerformance,
		Severity:    SeverityMedium,
		Threshold:   90.0, // 90% of benchmarks must pass
		Enabled:     true,
		Config: map[string]interface{}{
			"benchmark_timeout": "30s",
		},
		Check: qgm.checkPerformanceBenchmarks,
	})

	// Code complexity gate
	qgm.RegisterGate(&QualityGate{
		ID:          "code-complexity",
		Name:        "Code Complexity",
		Description: "Cyclomatic complexity validation",
		Type:        GateTypeComplexity,
		Severity:    SeverityMedium,
		Threshold:   10.0, // Max complexity of 10
		Enabled:     true,
		Config: map[string]interface{}{
			"max_complexity": 10,
		},
		Check: qgm.checkCodeComplexity,
	})

	// Documentation gate
	qgm.RegisterGate(&QualityGate{
		ID:          "documentation-coverage",
		Name:        "Documentation Coverage",
		Description: "Public API documentation coverage",
		Type:        GateTypeDocumentation,
		Severity:    SeverityLow,
		Threshold:   90.0, // 90% of public APIs documented
		Enabled:     true,
		Config: map[string]interface{}{
			"minimum_documentation": 90.0,
		},
		Check: qgm.checkDocumentationCoverage,
	})

	// Dependency gate
	qgm.RegisterGate(&QualityGate{
		ID:          "dependency-security",
		Name:        "Dependency Security",
		Description: "Dependency vulnerability scan",
		Type:        GateTypeDependencies,
		Severity:    SeverityHigh,
		Threshold:   0.0, // No vulnerable dependencies
		Enabled:     true,
		Config: map[string]interface{}{
			"allow_medium": false,
			"allow_low":    false,
		},
		Check: qgm.checkDependencySecurity,
	})
}

// Gate check implementations

func (qgm *QualityGateManager) checkSecurityScan(ctx context.Context, config map[string]interface{}) (*GateResult, error) {
	// Implementation would run actual security tools
	// This is a mock implementation

	result := &GateResult{
		Score:   95.0, // Mock score
		Message: "Security scan completed",
		Details: map[string]interface{}{
			"vulnerabilities_found": 0,
			"scan_duration":         "2.5s",
		},
	}

	// Mock some violations for demonstration
	if result.Score < 100.0 {
		result.Violations = []Violation{
			{
				Type:     "security",
				Severity: SeverityMedium,
				File:     "internal/auth/handler.go",
				Line:     45,
				Message:  "Potential hardcoded secret detected",
				Rule:     "SEC-5",
				Fix:      "Use environment variables for secrets",
			},
		}
		result.Suggestions = []string{
			"Review hardcoded secrets in authentication handlers",
			"Implement proper secret management",
		}
	}

	return result, nil
}

func (qgm *QualityGateManager) checkTestCoverage(ctx context.Context, config map[string]interface{}) (*GateResult, error) {
	// Mock implementation
	coverage := 85.0 // Mock coverage percentage
	threshold := config["minimum_coverage"].(float64)

	result := &GateResult{
		Score:   coverage,
		Message: fmt.Sprintf("Test coverage: %.1f%%", coverage),
		Details: map[string]interface{}{
			"coverage_percentage": coverage,
			"threshold":           threshold,
		},
	}

	if coverage < threshold {
		result.Violations = []Violation{
			{
				Type:     "coverage",
				Severity: SeverityHigh,
				Message:  fmt.Sprintf("Test coverage %.1f%% is below threshold %.1f%%", coverage, threshold),
				Rule:     "COV-1",
				Fix:      "Add more unit tests to increase coverage",
			},
		}
		result.Suggestions = []string{
			"Add unit tests for uncovered functions",
			"Review test quality and effectiveness",
		}
	}

	return result, nil
}

func (qgm *QualityGateManager) checkPerformanceBenchmarks(ctx context.Context, config map[string]interface{}) (*GateResult, error) {
	// Mock implementation
	benchmarkScore := 92.0

	result := &GateResult{
		Score:   benchmarkScore,
		Message: fmt.Sprintf("Performance benchmarks: %.1f%% passed", benchmarkScore),
		Details: map[string]interface{}{
			"benchmarks_passed": benchmarkScore,
			"total_benchmarks":  10,
		},
	}

	if benchmarkScore < 90.0 {
		result.Violations = []Violation{
			{
				Type:     "performance",
				Severity: SeverityMedium,
				Message:  "Some performance benchmarks failed",
				Rule:     "PERF-1",
				Fix:      "Optimize slow functions and algorithms",
			},
		}
		result.Suggestions = []string{
			"Profile slow functions",
			"Optimize database queries",
			"Review memory allocations",
		}
	}

	return result, nil
}

func (qgm *QualityGateManager) checkCodeComplexity(ctx context.Context, config map[string]interface{}) (*GateResult, error) {
	// Mock implementation
	maxComplexity := 8.0
	threshold := config["max_complexity"].(float64)

	result := &GateResult{
		Score:   (threshold - maxComplexity) / threshold * 100,
		Message: fmt.Sprintf("Max complexity: %.1f (threshold: %.1f)", maxComplexity, threshold),
		Details: map[string]interface{}{
			"max_complexity": maxComplexity,
			"threshold":      threshold,
		},
	}

	if maxComplexity > threshold {
		result.Violations = []Violation{
			{
				Type:     "complexity",
				Severity: SeverityMedium,
				Message:  fmt.Sprintf("Function complexity %.1f exceeds threshold %.1f", maxComplexity, threshold),
				Rule:     "COMP-1",
				Fix:      "Refactor complex functions into smaller ones",
			},
		}
		result.Suggestions = []string{
			"Break down complex functions",
			"Extract helper functions",
			"Reduce nesting levels",
		}
	}

	return result, nil
}

func (qgm *QualityGateManager) checkDocumentationCoverage(ctx context.Context, config map[string]interface{}) (*GateResult, error) {
	// Mock implementation
	docCoverage := 88.0
	threshold := config["minimum_documentation"].(float64)

	result := &GateResult{
		Score:   docCoverage,
		Message: fmt.Sprintf("Documentation coverage: %.1f%%", docCoverage),
		Details: map[string]interface{}{
			"documentation_percentage": docCoverage,
			"threshold":                threshold,
		},
	}

	if docCoverage < threshold {
		result.Violations = []Violation{
			{
				Type:     "documentation",
				Severity: SeverityLow,
				Message:  fmt.Sprintf("Documentation coverage %.1f%% is below threshold %.1f%%", docCoverage, threshold),
				Rule:     "DOC-1",
				Fix:      "Add documentation for public APIs",
			},
		}
		result.Suggestions = []string{
			"Document all public functions",
			"Add usage examples",
			"Include parameter descriptions",
		}
	}

	return result, nil
}

func (qgm *QualityGateManager) checkDependencySecurity(ctx context.Context, config map[string]interface{}) (*GateResult, error) {
	// Mock implementation
	vulnerableDeps := 0

	result := &GateResult{
		Score:   100.0 - float64(vulnerableDeps)*10, // Mock scoring
		Message: fmt.Sprintf("Dependency scan: %d vulnerabilities found", vulnerableDeps),
		Details: map[string]interface{}{
			"vulnerable_dependencies": vulnerableDeps,
			"total_dependencies":      45,
		},
	}

	if vulnerableDeps > 0 {
		result.Violations = []Violation{
			{
				Type:     "dependency",
				Severity: SeverityHigh,
				Message:  fmt.Sprintf("%d vulnerable dependencies found", vulnerableDeps),
				Rule:     "DEP-1",
				Fix:      "Update vulnerable dependencies to secure versions",
			},
		}
		result.Suggestions = []string{
			"Update vulnerable dependencies",
			"Review dependency usage",
			"Consider alternative libraries",
		}
	}

	return result, nil
}

// Configuration helpers

func getDefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		Timeout:       5 * time.Minute,
		Parallel:      true,
		FailFast:      false,
		ReportFormat:  "json",
		OutputPath:    "./quality-reports",
		EnableCaching: true,
		CacheTTL:      1 * time.Hour,
	}
}
