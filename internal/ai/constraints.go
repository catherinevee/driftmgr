package ai

import (
	"fmt"
	"strings"
)

// CodeGenerationConstraints defines constraints for AI code generation
type CodeGenerationConstraints struct {
	MaxFunctionLength     int     `json:"max_function_length"`
	MaxNestingDepth       int     `json:"max_nesting_depth"`
	MaxParameters         int     `json:"max_parameters"`
	MaxFileLength         int     `json:"max_file_length"`
	MaxStructFields       int     `json:"max_struct_fields"`
	MaxInterfaceMethods   int     `json:"max_interface_methods"`
	Temperature           float64 `json:"temperature"`
	VerbosityPenalty      float64 `json:"verbosity_penalty"`
	ComplexityPenalty     float64 `json:"complexity_penalty"`
	DuplicationPenalty    float64 `json:"duplication_penalty"`
	SecurityWeight        float64 `json:"security_weight"`
	PerformanceWeight     float64 `json:"performance_weight"`
	MaintainabilityWeight float64 `json:"maintainability_weight"`
}

// ConstraintViolation represents a violation of code generation constraints
type ConstraintViolation struct {
	Type        string   `json:"type"`
	Severity    string   `json:"severity"`
	Message     string   `json:"message"`
	File        string   `json:"file"`
	Line        int      `json:"line"`
	Column      int      `json:"column"`
	Current     int      `json:"current"`
	Limit       int      `json:"limit"`
	Suggestions []string `json:"suggestions"`
}

// ConstraintEnforcer enforces code generation constraints
type ConstraintEnforcer struct {
	constraints *CodeGenerationConstraints
	rules       []ConstraintRule
}

// ConstraintRule defines a specific constraint rule
type ConstraintRule struct {
	ID          string                                                                 `json:"id"`
	Name        string                                                                 `json:"name"`
	Description string                                                                 `json:"description"`
	Type        ConstraintType                                                         `json:"type"`
	Severity    string                                                                 `json:"severity"`
	Check       func(code string, config map[string]interface{}) []ConstraintViolation `json:"-"`
	Config      map[string]interface{}                                                 `json:"config"`
}

// ConstraintType represents the type of constraint
type ConstraintType string

const (
	ConstraintTypeLength      ConstraintType = "length"
	ConstraintTypeComplexity  ConstraintType = "complexity"
	ConstraintTypeNesting     ConstraintType = "nesting"
	ConstraintTypeParameters  ConstraintType = "parameters"
	ConstraintTypeDuplication ConstraintType = "duplication"
	ConstraintTypeSecurity    ConstraintType = "security"
	ConstraintTypePerformance ConstraintType = "performance"
	ConstraintTypeStandards   ConstraintType = "standards"
)

// NewConstraintEnforcer creates a new constraint enforcer
func NewConstraintEnforcer(constraints *CodeGenerationConstraints) *ConstraintEnforcer {
	if constraints == nil {
		constraints = getDefaultConstraints()
	}

	enforcer := &ConstraintEnforcer{
		constraints: constraints,
		rules:       make([]ConstraintRule, 0),
	}

	// Register default rules
	enforcer.registerDefaultRules()

	return enforcer
}

// ApplyConstraints applies all constraints to the given code
func (ce *ConstraintEnforcer) ApplyConstraints(code string, filePath string) ([]ConstraintViolation, error) {
	var violations []ConstraintViolation

	for _, rule := range ce.rules {
		ruleViolations := rule.Check(code, rule.Config)

		// Add file path to violations
		for i := range ruleViolations {
			ruleViolations[i].File = filePath
		}

		violations = append(violations, ruleViolations...)
	}

	return violations, nil
}

// GenerateConstraintPrompt generates a prompt with constraints for AI
func (ce *ConstraintEnforcer) GenerateConstraintPrompt(basePrompt string) string {
	constraints := ce.buildConstraintText()

	return fmt.Sprintf(`%s

CONSTRAINTS (MUST FOLLOW):
%s

CRITICAL RULES:
- Maximum function length: %d lines
- Maximum nesting depth: %d levels
- Maximum parameters: %d per function
- Maximum file length: %d lines
- Keep it simple and focused
- Solve current problem, don't anticipate future needs
- Use existing patterns from codebase
- NO EMOJI in code or comments
- Security-first approach required
- Performance considerations mandatory

VERBOSITY CONTROL:
- Write minimal, focused code
- Avoid over-engineering
- Don't add unnecessary abstractions
- Use clear, concise variable names
- Prefer composition over inheritance
- Single responsibility principle

SECURITY REQUIREMENTS:
- Input validation on all external data
- Parameterized queries for database operations
- Secure error handling (no sensitive data exposure)
- Authentication and authorization checks
- Rate limiting on API endpoints
- Secure logging (no secrets in logs)

PERFORMANCE REQUIREMENTS:
- Response time < 200ms for API endpoints
- Memory usage optimization
- Efficient algorithms and data structures
- Connection pooling for database operations
- Caching where appropriate

If any constraint cannot be met, explain why and suggest alternatives.`,
		basePrompt, constraints,
		ce.constraints.MaxFunctionLength,
		ce.constraints.MaxNestingDepth,
		ce.constraints.MaxParameters,
		ce.constraints.MaxFileLength)
}

// ValidateCode validates code against all constraints
func (ce *ConstraintEnforcer) ValidateCode(code string, filePath string) (*ValidationResult, error) {
	violations, err := ce.ApplyConstraints(code, filePath)
	if err != nil {
		return nil, err
	}

	result := &ValidationResult{
		Valid:      len(violations) == 0,
		Violations: violations,
		Score:      ce.calculateScore(violations),
		Summary:    ce.generateSummary(violations),
	}

	return result, nil
}

// ValidationResult contains the result of code validation
type ValidationResult struct {
	Valid      bool                  `json:"valid"`
	Violations []ConstraintViolation `json:"violations"`
	Score      float64               `json:"score"`
	Summary    ValidationSummary     `json:"summary"`
}

// ValidationSummary provides a summary of validation results
type ValidationSummary struct {
	TotalViolations int            `json:"total_violations"`
	ByType          map[string]int `json:"by_type"`
	BySeverity      map[string]int `json:"by_severity"`
	Recommendations []string       `json:"recommendations"`
	Score           float64        `json:"score"`
}

// Helper methods

func (ce *ConstraintEnforcer) buildConstraintText() string {
	var constraints []string

	constraints = append(constraints, "FUNCTION CONSTRAINTS:")
	constraints = append(constraints, fmt.Sprintf("- Maximum length: %d lines", ce.constraints.MaxFunctionLength))
	constraints = append(constraints, fmt.Sprintf("- Maximum nesting: %d levels", ce.constraints.MaxNestingDepth))
	constraints = append(constraints, fmt.Sprintf("- Maximum parameters: %d", ce.constraints.MaxParameters))

	constraints = append(constraints, "\nFILE CONSTRAINTS:")
	constraints = append(constraints, fmt.Sprintf("- Maximum length: %d lines", ce.constraints.MaxFileLength))
	constraints = append(constraints, fmt.Sprintf("- Maximum struct fields: %d", ce.constraints.MaxStructFields))
	constraints = append(constraints, fmt.Sprintf("- Maximum interface methods: %d", ce.constraints.MaxInterfaceMethods))

	constraints = append(constraints, "\nQUALITY CONSTRAINTS:")
	constraints = append(constraints, "- Single responsibility principle")
	constraints = append(constraints, "- DRY (Don't Repeat Yourself)")
	constraints = append(constraints, "- Clear, descriptive naming")
	constraints = append(constraints, "- Minimal complexity")

	constraints = append(constraints, "\nSECURITY CONSTRAINTS:")
	constraints = append(constraints, "- Input validation required")
	constraints = append(constraints, "- Secure error handling")
	constraints = append(constraints, "- No hardcoded secrets")
	constraints = append(constraints, "- Authentication checks")

	return strings.Join(constraints, "\n")
}

func (ce *ConstraintEnforcer) calculateScore(violations []ConstraintViolation) float64 {
	if len(violations) == 0 {
		return 100.0
	}

	// Calculate penalty based on violation severity
	penalty := 0.0
	for _, violation := range violations {
		switch violation.Severity {
		case "critical":
			penalty += 20.0
		case "high":
			penalty += 10.0
		case "medium":
			penalty += 5.0
		case "low":
			penalty += 2.0
		}
	}

	score := 100.0 - penalty
	if score < 0 {
		score = 0
	}

	return score
}

func (ce *ConstraintEnforcer) generateSummary(violations []ConstraintViolation) ValidationSummary {
	summary := ValidationSummary{
		ByType:     make(map[string]int),
		BySeverity: make(map[string]int),
	}

	summary.TotalViolations = len(violations)

	for _, violation := range violations {
		summary.ByType[violation.Type]++
		summary.BySeverity[violation.Severity]++
	}

	summary.Score = ce.calculateScore(violations)
	summary.Recommendations = ce.generateRecommendations(violations)

	return summary
}

func (ce *ConstraintEnforcer) generateRecommendations(violations []ConstraintViolation) []string {
	var recommendations []string

	typeCounts := make(map[string]int)
	for _, violation := range violations {
		typeCounts[violation.Type]++
	}

	// Generate recommendations based on violation types
	if typeCounts["length"] > 0 {
		recommendations = append(recommendations, "Break down long functions into smaller, focused functions")
	}

	if typeCounts["complexity"] > 0 {
		recommendations = append(recommendations, "Reduce cyclomatic complexity by simplifying logic")
	}

	if typeCounts["nesting"] > 0 {
		recommendations = append(recommendations, "Reduce nesting depth using early returns and guard clauses")
	}

	if typeCounts["parameters"] > 0 {
		recommendations = append(recommendations, "Reduce function parameters by using structs or configuration objects")
	}

	if typeCounts["duplication"] > 0 {
		recommendations = append(recommendations, "Extract common code into reusable functions")
	}

	if typeCounts["security"] > 0 {
		recommendations = append(recommendations, "Review and implement security best practices")
	}

	if typeCounts["performance"] > 0 {
		recommendations = append(recommendations, "Optimize performance-critical code paths")
	}

	return recommendations
}

// Register default constraint rules

func (ce *ConstraintEnforcer) registerDefaultRules() {
	// Function length rule
	ce.rules = append(ce.rules, ConstraintRule{
		ID:          "function_length",
		Name:        "Function Length",
		Description: "Ensures functions don't exceed maximum length",
		Type:        ConstraintTypeLength,
		Severity:    "high",
		Config: map[string]interface{}{
			"max_length": ce.constraints.MaxFunctionLength,
		},
		Check: ce.checkFunctionLength,
	})

	// Nesting depth rule
	ce.rules = append(ce.rules, ConstraintRule{
		ID:          "nesting_depth",
		Name:        "Nesting Depth",
		Description: "Ensures code doesn't exceed maximum nesting depth",
		Type:        ConstraintTypeNesting,
		Severity:    "medium",
		Config: map[string]interface{}{
			"max_depth": ce.constraints.MaxNestingDepth,
		},
		Check: ce.checkNestingDepth,
	})

	// Parameter count rule
	ce.rules = append(ce.rules, ConstraintRule{
		ID:          "parameter_count",
		Name:        "Parameter Count",
		Description: "Ensures functions don't exceed maximum parameter count",
		Type:        ConstraintTypeParameters,
		Severity:    "medium",
		Config: map[string]interface{}{
			"max_parameters": ce.constraints.MaxParameters,
		},
		Check: ce.checkParameterCount,
	})

	// File length rule
	ce.rules = append(ce.rules, ConstraintRule{
		ID:          "file_length",
		Name:        "File Length",
		Description: "Ensures files don't exceed maximum length",
		Type:        ConstraintTypeLength,
		Severity:    "low",
		Config: map[string]interface{}{
			"max_length": ce.constraints.MaxFileLength,
		},
		Check: ce.checkFileLength,
	})

	// Code duplication rule
	ce.rules = append(ce.rules, ConstraintRule{
		ID:          "code_duplication",
		Name:        "Code Duplication",
		Description: "Detects and prevents code duplication",
		Type:        ConstraintTypeDuplication,
		Severity:    "medium",
		Config: map[string]interface{}{
			"similarity_threshold": 0.8,
		},
		Check: ce.checkCodeDuplication,
	})

	// Security rule
	ce.rules = append(ce.rules, ConstraintRule{
		ID:          "security_patterns",
		Name:        "Security Patterns",
		Description: "Ensures security best practices are followed",
		Type:        ConstraintTypeSecurity,
		Severity:    "critical",
		Config: map[string]interface{}{
			"check_input_validation":  true,
			"check_sql_injection":     true,
			"check_hardcoded_secrets": true,
		},
		Check: ce.checkSecurityPatterns,
	})
}

// Constraint check implementations

func (ce *ConstraintEnforcer) checkFunctionLength(code string, config map[string]interface{}) []ConstraintViolation {
	var violations []ConstraintViolation

	// Simplified implementation - would use AST parsing in practice
	lines := strings.Split(code, "\n")
	maxLength := config["max_length"].(int)

	currentFunction := ""
	currentLength := 0
	inFunction := false

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Detect function start
		if strings.HasPrefix(line, "func ") {
			if inFunction && currentLength > maxLength {
				violations = append(violations, ConstraintViolation{
					Type:     "length",
					Severity: "high",
					Message:  fmt.Sprintf("Function '%s' exceeds maximum length of %d lines", currentFunction, maxLength),
					Line:     i - currentLength + 1,
					Current:  currentLength,
					Limit:    maxLength,
					Suggestions: []string{
						"Break down the function into smaller functions",
						"Extract helper functions for complex logic",
						"Use composition to reduce function size",
					},
				})
			}

			currentFunction = line
			currentLength = 1
			inFunction = true
		} else if inFunction {
			currentLength++

			// Detect function end (simplified)
			if line == "}" && currentLength > 1 {
				if currentLength > maxLength {
					violations = append(violations, ConstraintViolation{
						Type:     "length",
						Severity: "high",
						Message:  fmt.Sprintf("Function '%s' exceeds maximum length of %d lines", currentFunction, maxLength),
						Line:     i - currentLength + 1,
						Current:  currentLength,
						Limit:    maxLength,
						Suggestions: []string{
							"Break down the function into smaller functions",
							"Extract helper functions for complex logic",
							"Use composition to reduce function size",
						},
					})
				}
				inFunction = false
				currentLength = 0
			}
		}
	}

	return violations
}

func (ce *ConstraintEnforcer) checkNestingDepth(code string, config map[string]interface{}) []ConstraintViolation {
	var violations []ConstraintViolation

	maxDepth := config["max_depth"].(int)
	lines := strings.Split(code, "\n")

	currentDepth := 0
	maxCurrentDepth := 0

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Count opening braces
		openBraces := strings.Count(line, "{")
		closeBraces := strings.Count(line, "}")

		currentDepth += openBraces - closeBraces

		if currentDepth > maxCurrentDepth {
			maxCurrentDepth = currentDepth
		}

		// Check for excessive nesting
		if currentDepth > maxDepth {
			violations = append(violations, ConstraintViolation{
				Type:     "nesting",
				Severity: "medium",
				Message:  fmt.Sprintf("Nesting depth %d exceeds maximum of %d", currentDepth, maxDepth),
				Line:     i + 1,
				Current:  currentDepth,
				Limit:    maxDepth,
				Suggestions: []string{
					"Use early returns to reduce nesting",
					"Extract nested logic into separate functions",
					"Use guard clauses for validation",
				},
			})
		}
	}

	return violations
}

func (ce *ConstraintEnforcer) checkParameterCount(code string, config map[string]interface{}) []ConstraintViolation {
	var violations []ConstraintViolation

	maxParams := config["max_parameters"].(int)
	lines := strings.Split(code, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Detect function declarations
		if strings.HasPrefix(line, "func ") {
			// Count parameters (simplified implementation)
			paramCount := strings.Count(line, ",") + 1

			// Adjust for functions with no parameters
			if !strings.Contains(line, "(") || strings.Contains(line, "()") {
				paramCount = 0
			}

			if paramCount > maxParams {
				violations = append(violations, ConstraintViolation{
					Type:     "parameters",
					Severity: "medium",
					Message:  fmt.Sprintf("Function has %d parameters, exceeds maximum of %d", paramCount, maxParams),
					Line:     i + 1,
					Current:  paramCount,
					Limit:    maxParams,
					Suggestions: []string{
						"Use a configuration struct for multiple parameters",
						"Group related parameters into a single struct",
						"Consider using builder pattern for complex initialization",
					},
				})
			}
		}
	}

	return violations
}

func (ce *ConstraintEnforcer) checkFileLength(code string, config map[string]interface{}) []ConstraintViolation {
	var violations []ConstraintViolation

	maxLength := config["max_length"].(int)
	lines := strings.Split(code, "\n")

	if len(lines) > maxLength {
		violations = append(violations, ConstraintViolation{
			Type:     "length",
			Severity: "low",
			Message:  fmt.Sprintf("File length %d exceeds maximum of %d lines", len(lines), maxLength),
			Line:     1,
			Current:  len(lines),
			Limit:    maxLength,
			Suggestions: []string{
				"Split the file into smaller, focused files",
				"Extract related functionality into separate packages",
				"Consider using interfaces to reduce coupling",
			},
		})
	}

	return violations
}

func (ce *ConstraintEnforcer) checkCodeDuplication(code string, config map[string]interface{}) []ConstraintViolation {
	var violations []ConstraintViolation

	// Simplified implementation - would use more sophisticated duplication detection
	lines := strings.Split(code, "\n")
	lineCounts := make(map[string]int)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 10 { // Only check substantial lines
			lineCounts[line]++
		}
	}

	// Check for duplicate lines
	for line, count := range lineCounts {
		if count > 2 { // Allow some duplication
			violations = append(violations, ConstraintViolation{
				Type:     "duplication",
				Severity: "medium",
				Message:  fmt.Sprintf("Line appears %d times: %s", count, line),
				Line:     0, // Would need to track line numbers
				Current:  count,
				Limit:    2,
				Suggestions: []string{
					"Extract duplicate code into a reusable function",
					"Use constants for repeated values",
					"Consider using a helper function",
				},
			})
		}
	}

	return violations
}

func (ce *ConstraintEnforcer) checkSecurityPatterns(code string, config map[string]interface{}) []ConstraintViolation {
	var violations []ConstraintViolation

	lines := strings.Split(code, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Check for hardcoded secrets
		if strings.Contains(strings.ToLower(line), "password") && strings.Contains(line, "=") {
			if !strings.Contains(line, "os.Getenv") && !strings.Contains(line, "config") {
				violations = append(violations, ConstraintViolation{
					Type:     "security",
					Severity: "critical",
					Message:  "Potential hardcoded secret detected",
					Line:     i + 1,
					Suggestions: []string{
						"Use environment variables for secrets",
						"Use a secure configuration management system",
						"Never hardcode passwords or API keys",
					},
				})
			}
		}

		// Check for SQL injection patterns
		if strings.Contains(line, "fmt.Sprintf") && strings.Contains(line, "SELECT") {
			violations = append(violations, ConstraintViolation{
				Type:     "security",
				Severity: "critical",
				Message:  "Potential SQL injection vulnerability",
				Line:     i + 1,
				Suggestions: []string{
					"Use parameterized queries",
					"Use prepared statements",
					"Validate and sanitize all input",
				},
			})
		}
	}

	return violations
}

// Configuration helpers

func getDefaultConstraints() *CodeGenerationConstraints {
	return &CodeGenerationConstraints{
		MaxFunctionLength:     50,
		MaxNestingDepth:       3,
		MaxParameters:         5,
		MaxFileLength:         500,
		MaxStructFields:       10,
		MaxInterfaceMethods:   8,
		Temperature:           0.7,
		VerbosityPenalty:      0.3,
		ComplexityPenalty:     0.4,
		DuplicationPenalty:    0.2,
		SecurityWeight:        0.4,
		PerformanceWeight:     0.3,
		MaintainabilityWeight: 0.3,
	}
}
