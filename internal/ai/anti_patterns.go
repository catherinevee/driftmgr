package ai

import (
	"fmt"
	"regexp"
	"strings"
)

// AntiPatternDetector detects and prevents common AI coding anti-patterns
type AntiPatternDetector struct {
	patterns []AntiPattern
	config   *AntiPatternConfig
}

// AntiPatternConfig contains configuration for anti-pattern detection
type AntiPatternConfig struct {
	EnableDetection  bool          `json:"enable_detection"`
	StrictMode       bool          `json:"strict_mode"`
	WarningThreshold float64       `json:"warning_threshold"`
	ErrorThreshold   float64       `json:"error_threshold"`
	CustomPatterns   []AntiPattern `json:"custom_patterns"`
	IgnorePatterns   []string      `json:"ignore_patterns"`
}

// AntiPattern represents a specific anti-pattern to detect
type AntiPattern struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    AntiPatternCategory    `json:"category"`
	Severity    AntiPatternSeverity    `json:"severity"`
	Pattern     string                 `json:"pattern"`
	Regex       *regexp.Regexp         `json:"-"`
	Examples    []string               `json:"examples"`
	Fix         string                 `json:"fix"`
	Prevention  string                 `json:"prevention"`
	Config      map[string]interface{} `json:"config"`
}

// AntiPatternCategory represents the category of an anti-pattern
type AntiPatternCategory string

const (
	CategoryVerbosity       AntiPatternCategory = "verbosity"
	CategorySecurity        AntiPatternCategory = "security"
	CategoryPerformance     AntiPatternCategory = "performance"
	CategoryMaintainability AntiPatternCategory = "maintainability"
	CategoryComplexity      AntiPatternCategory = "complexity"
	CategoryDuplication     AntiPatternCategory = "duplication"
	CategoryStandards       AntiPatternCategory = "standards"
)

// AntiPatternSeverity represents the severity of an anti-pattern
type AntiPatternSeverity string

const (
	SeverityCritical AntiPatternSeverity = "critical"
	SeverityHigh     AntiPatternSeverity = "high"
	SeverityMedium   AntiPatternSeverity = "medium"
	SeverityLow      AntiPatternSeverity = "low"
)

// AntiPatternMatch represents a detected anti-pattern
type AntiPatternMatch struct {
	Pattern     AntiPattern `json:"pattern"`
	File        string      `json:"file"`
	Line        int         `json:"line"`
	Column      int         `json:"column"`
	Code        string      `json:"code"`
	Context     string      `json:"context"`
	Confidence  float64     `json:"confidence"`
	Suggestions []string    `json:"suggestions"`
}

// AntiPatternReport contains the results of anti-pattern detection
type AntiPatternReport struct {
	TotalMatches    int                `json:"total_matches"`
	ByCategory      map[string]int     `json:"by_category"`
	BySeverity      map[string]int     `json:"by_severity"`
	Matches         []AntiPatternMatch `json:"matches"`
	Score           float64            `json:"score"`
	Recommendations []string           `json:"recommendations"`
	Summary         AntiPatternSummary `json:"summary"`
}

// AntiPatternSummary provides a summary of anti-pattern detection
type AntiPatternSummary struct {
	CriticalIssues int      `json:"critical_issues"`
	HighIssues     int      `json:"high_issues"`
	MediumIssues   int      `json:"medium_issues"`
	LowIssues      int      `json:"low_issues"`
	TopCategories  []string `json:"top_categories"`
	OverallHealth  string   `json:"overall_health"`
}

// NewAntiPatternDetector creates a new anti-pattern detector
func NewAntiPatternDetector(config *AntiPatternConfig) *AntiPatternDetector {
	if config == nil {
		config = getDefaultAntiPatternConfig()
	}

	detector := &AntiPatternDetector{
		patterns: make([]AntiPattern, 0),
		config:   config,
	}

	// Register default anti-patterns
	detector.registerDefaultPatterns()

	// Add custom patterns
	detector.patterns = append(detector.patterns, config.CustomPatterns...)

	return detector
}

// DetectAntiPatterns detects anti-patterns in the given code
func (apd *AntiPatternDetector) DetectAntiPatterns(code string, filePath string) (*AntiPatternReport, error) {
	if !apd.config.EnableDetection {
		return &AntiPatternReport{
			Score: 100.0,
			Summary: AntiPatternSummary{
				OverallHealth: "disabled",
			},
		}, nil
	}

	var matches []AntiPatternMatch

	for _, pattern := range apd.patterns {
		// Skip ignored patterns
		if apd.isPatternIgnored(pattern.ID) {
			continue
		}

		patternMatches := apd.detectPattern(code, pattern, filePath)
		matches = append(matches, patternMatches...)
	}

	// Generate report
	report := apd.generateReport(matches)

	return report, nil
}

// RegisterAntiPattern registers a new anti-pattern
func (apd *AntiPatternDetector) RegisterAntiPattern(pattern AntiPattern) error {
	// Compile regex if pattern is provided
	if pattern.Pattern != "" {
		regex, err := regexp.Compile(pattern.Pattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
		pattern.Regex = regex
	}

	apd.patterns = append(apd.patterns, pattern)
	return nil
}

// Helper methods

func (apd *AntiPatternDetector) detectPattern(code string, pattern AntiPattern, filePath string) []AntiPatternMatch {
	var matches []AntiPatternMatch

	if pattern.Regex == nil {
		return matches
	}

	lines := strings.Split(code, "\n")

	for lineNum, line := range lines {
		lineMatches := pattern.Regex.FindAllStringIndex(line, -1)

		for _, match := range lineMatches {
			confidence := apd.calculateConfidence(line, pattern)

			// Apply thresholds
			if confidence < apd.config.WarningThreshold {
				continue
			}

			match := AntiPatternMatch{
				Pattern:    pattern,
				File:       filePath,
				Line:       lineNum + 1,
				Column:     match[0] + 1,
				Code:       line[match[0]:match[1]],
				Context:    apd.getContext(lines, lineNum, 2),
				Confidence: confidence,
				Suggestions: []string{
					pattern.Fix,
					pattern.Prevention,
				},
			}

			matches = append(matches, match)
		}
	}

	return matches
}

func (apd *AntiPatternDetector) calculateConfidence(line string, pattern AntiPattern) float64 {
	// Simplified confidence calculation
	confidence := 0.5 // Base confidence

	// Increase confidence based on pattern characteristics
	if strings.Contains(line, "TODO") || strings.Contains(line, "FIXME") {
		confidence += 0.2
	}

	if strings.Contains(line, "hack") || strings.Contains(line, "temporary") {
		confidence += 0.3
	}

	// Decrease confidence for common false positives
	if strings.Contains(line, "test") || strings.Contains(line, "example") {
		confidence -= 0.2
	}

	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

func (apd *AntiPatternDetector) getContext(lines []string, lineNum int, contextLines int) string {
	start := lineNum - contextLines
	if start < 0 {
		start = 0
	}

	end := lineNum + contextLines + 1
	if end > len(lines) {
		end = len(lines)
	}

	return strings.Join(lines[start:end], "\n")
}

func (apd *AntiPatternDetector) isPatternIgnored(patternID string) bool {
	for _, ignored := range apd.config.IgnorePatterns {
		if ignored == patternID {
			return true
		}
	}
	return false
}

func (apd *AntiPatternDetector) generateReport(matches []AntiPatternMatch) *AntiPatternReport {
	report := &AntiPatternReport{
		Matches:    matches,
		ByCategory: make(map[string]int),
		BySeverity: make(map[string]int),
	}

	report.TotalMatches = len(matches)

	// Count by category and severity
	for _, match := range matches {
		report.ByCategory[string(match.Pattern.Category)]++
		report.BySeverity[string(match.Pattern.Severity)]++
	}

	// Calculate score
	report.Score = apd.calculateScore(matches)

	// Generate recommendations
	report.Recommendations = apd.generateRecommendations(matches)

	// Generate summary
	report.Summary = apd.generateSummary(matches)

	return report
}

func (apd *AntiPatternDetector) calculateScore(matches []AntiPatternMatch) float64 {
	if len(matches) == 0 {
		return 100.0
	}

	penalty := 0.0
	for _, match := range matches {
		switch match.Pattern.Severity {
		case SeverityCritical:
			penalty += 20.0 * match.Confidence
		case SeverityHigh:
			penalty += 10.0 * match.Confidence
		case SeverityMedium:
			penalty += 5.0 * match.Confidence
		case SeverityLow:
			penalty += 2.0 * match.Confidence
		}
	}

	score := 100.0 - penalty
	if score < 0 {
		score = 0
	}

	return score
}

func (apd *AntiPatternDetector) generateRecommendations(matches []AntiPatternMatch) []string {
	var recommendations []string

	categoryCounts := make(map[AntiPatternCategory]int)
	for _, match := range matches {
		categoryCounts[match.Pattern.Category]++
	}

	// Generate recommendations based on most common categories
	if categoryCounts[CategoryVerbosity] > 0 {
		recommendations = append(recommendations, "Reduce code verbosity and over-engineering")
	}

	if categoryCounts[CategorySecurity] > 0 {
		recommendations = append(recommendations, "Review and fix security vulnerabilities")
	}

	if categoryCounts[CategoryPerformance] > 0 {
		recommendations = append(recommendations, "Optimize performance-critical code")
	}

	if categoryCounts[CategoryComplexity] > 0 {
		recommendations = append(recommendations, "Simplify complex code structures")
	}

	if categoryCounts[CategoryDuplication] > 0 {
		recommendations = append(recommendations, "Eliminate code duplication")
	}

	return recommendations
}

func (apd *AntiPatternDetector) generateSummary(matches []AntiPatternMatch) AntiPatternSummary {
	summary := AntiPatternSummary{}

	for _, match := range matches {
		switch match.Pattern.Severity {
		case SeverityCritical:
			summary.CriticalIssues++
		case SeverityHigh:
			summary.HighIssues++
		case SeverityMedium:
			summary.MediumIssues++
		case SeverityLow:
			summary.LowIssues++
		}
	}

	// Determine overall health
	if summary.CriticalIssues > 0 {
		summary.OverallHealth = "critical"
	} else if summary.HighIssues > 2 {
		summary.OverallHealth = "poor"
	} else if summary.HighIssues > 0 || summary.MediumIssues > 5 {
		summary.OverallHealth = "fair"
	} else if summary.MediumIssues > 0 || summary.LowIssues > 3 {
		summary.OverallHealth = "good"
	} else {
		summary.OverallHealth = "excellent"
	}

	// Find top categories
	categoryCounts := make(map[AntiPatternCategory]int)
	for _, match := range matches {
		categoryCounts[match.Pattern.Category]++
	}

	// Sort categories by count (simplified)
	for category, count := range categoryCounts {
		if count > 0 {
			summary.TopCategories = append(summary.TopCategories, string(category))
		}
	}

	return summary
}

// Register default anti-patterns

func (apd *AntiPatternDetector) registerDefaultPatterns() {
	// Verbosity anti-patterns
	apd.registerVerbosityPatterns()

	// Security anti-patterns
	apd.registerSecurityPatterns()

	// Performance anti-patterns
	apd.registerPerformancePatterns()

	// Complexity anti-patterns
	apd.registerComplexityPatterns()

	// Duplication anti-patterns
	apd.registerDuplicationPatterns()

	// Standards anti-patterns
	apd.registerStandardsPatterns()
}

func (apd *AntiPatternDetector) registerVerbosityPatterns() {
	// Over-engineering pattern
	apd.RegisterAntiPattern(AntiPattern{
		ID:          "over_engineering",
		Name:        "Over-Engineering",
		Description: "Unnecessary complexity and abstraction",
		Category:    CategoryVerbosity,
		Severity:    SeverityMedium,
		Pattern:     `(?i)(abstract|factory|builder|strategy|observer).*interface`,
		Examples: []string{
			"type AbstractFactory interface",
			"func NewBuilder() *Builder",
		},
		Fix:        "Simplify the design and use direct implementation",
		Prevention: "Start with simple solutions and add complexity only when needed",
	})

	// Excessive comments pattern
	apd.RegisterAntiPattern(AntiPattern{
		ID:          "excessive_comments",
		Name:        "Excessive Comments",
		Description: "Too many obvious or redundant comments",
		Category:    CategoryVerbosity,
		Severity:    SeverityLow,
		Pattern:     `//\s*(This|Here|Now|Then|So|Also|Additionally)`,
		Examples: []string{
			"// This function does something",
			"// Here we initialize the variable",
		},
		Fix:        "Remove obvious comments and keep only essential ones",
		Prevention: "Write self-documenting code with clear variable names",
	})
}

func (apd *AntiPatternDetector) registerSecurityPatterns() {
	// Hardcoded secrets pattern
	apd.RegisterAntiPattern(AntiPattern{
		ID:          "hardcoded_secrets",
		Name:        "Hardcoded Secrets",
		Description: "Hardcoded passwords, API keys, or other secrets",
		Category:    CategorySecurity,
		Severity:    SeverityCritical,
		Pattern:     `(?i)(password|secret|key|token)\s*=\s*["\'][^"\']+["\']`,
		Examples: []string{
			`password = "mypassword123"`,
			`apiKey = "sk-1234567890"`,
		},
		Fix:        "Use environment variables or secure configuration management",
		Prevention: "Never hardcode secrets in source code",
	})

	// SQL injection pattern
	apd.RegisterAntiPattern(AntiPattern{
		ID:          "sql_injection",
		Name:        "SQL Injection",
		Description: "Potential SQL injection vulnerability",
		Category:    CategorySecurity,
		Severity:    SeverityCritical,
		Pattern:     `fmt\.Sprintf.*SELECT.*%[sdv]`,
		Examples: []string{
			`fmt.Sprintf("SELECT * FROM users WHERE id = %d", userID)`,
		},
		Fix:        "Use parameterized queries or prepared statements",
		Prevention: "Always use parameterized queries for database operations",
	})
}

func (apd *AntiPatternDetector) registerPerformancePatterns() {
	// Inefficient string concatenation pattern
	apd.RegisterAntiPattern(AntiPattern{
		ID:          "inefficient_string_concat",
		Name:        "Inefficient String Concatenation",
		Description: "String concatenation in loops",
		Category:    CategoryPerformance,
		Severity:    SeverityMedium,
		Pattern:     `for.*\{.*\+.*=`,
		Examples: []string{
			"for i := range items { result += items[i] }",
		},
		Fix:        "Use strings.Builder or bytes.Buffer for efficient concatenation",
		Prevention: "Use appropriate data structures for string building",
	})

	// Unnecessary allocations pattern
	apd.RegisterAntiPattern(AntiPattern{
		ID:          "unnecessary_allocations",
		Name:        "Unnecessary Allocations",
		Description: "Creating unnecessary slices or maps",
		Category:    CategoryPerformance,
		Severity:    SeverityLow,
		Pattern:     `make\(\[\]string, 0\)`,
		Examples: []string{
			"items := make([]string, 0)",
		},
		Fix:        "Use nil slice or pre-allocate with known capacity",
		Prevention: "Pre-allocate slices and maps when size is known",
	})
}

func (apd *AntiPatternDetector) registerComplexityPatterns() {
	// Deep nesting pattern
	apd.RegisterAntiPattern(AntiPattern{
		ID:          "deep_nesting",
		Name:        "Deep Nesting",
		Description: "Excessive nesting levels",
		Category:    CategoryComplexity,
		Severity:    SeverityMedium,
		Pattern:     `if.*\{.*if.*\{.*if.*\{`,
		Examples: []string{
			"if condition1 { if condition2 { if condition3 { ... } } }",
		},
		Fix:        "Use early returns and guard clauses to reduce nesting",
		Prevention: "Keep nesting levels to a minimum (max 3 levels)",
	})

	// Long parameter lists pattern
	apd.RegisterAntiPattern(AntiPattern{
		ID:          "long_parameter_list",
		Name:        "Long Parameter List",
		Description: "Functions with too many parameters",
		Category:    CategoryComplexity,
		Severity:    SeverityMedium,
		Pattern:     `func\s+\w+\([^)]*,[^)]*,[^)]*,[^)]*,[^)]*,[^)]*\)`,
		Examples: []string{
			"func process(a, b, c, d, e, f string) error",
		},
		Fix:        "Use a configuration struct or builder pattern",
		Prevention: "Keep function parameters to 5 or fewer",
	})
}

func (apd *AntiPatternDetector) registerDuplicationPatterns() {
	// Code duplication pattern
	apd.RegisterAntiPattern(AntiPattern{
		ID:          "code_duplication",
		Name:        "Code Duplication",
		Description: "Repeated code blocks",
		Category:    CategoryDuplication,
		Severity:    SeverityMedium,
		Pattern:     `(\w+\([^)]*\)\s*\{[^}]*\})\s*\1`,
		Examples: []string{
			"validateUser(user) { ... } validateUser(user) { ... }",
		},
		Fix:        "Extract common code into reusable functions",
		Prevention: "Follow DRY principle and extract common functionality",
	})
}

func (apd *AntiPatternDetector) registerStandardsPatterns() {
	// Magic numbers pattern
	apd.RegisterAntiPattern(AntiPattern{
		ID:          "magic_numbers",
		Name:        "Magic Numbers",
		Description: "Hardcoded numeric literals without explanation",
		Category:    CategoryStandards,
		Severity:    SeverityLow,
		Pattern:     `\b(3|5|7|10|100|1000|3600|86400)\b`,
		Examples: []string{
			"timeout := 30 * time.Second",
			"maxRetries := 3",
		},
		Fix:        "Define named constants for magic numbers",
		Prevention: "Use named constants for all numeric literals",
	})

	// TODO/FIXME pattern
	apd.RegisterAntiPattern(AntiPattern{
		ID:          "todo_fixme",
		Name:        "TODO/FIXME Comments",
		Description: "TODO or FIXME comments in production code",
		Category:    CategoryStandards,
		Severity:    SeverityLow,
		Pattern:     `(?i)(TODO|FIXME|HACK|XXX)`,
		Examples: []string{
			"// TODO: implement error handling",
			"// FIXME: this is a temporary solution",
		},
		Fix:        "Address the TODO/FIXME or create a proper issue",
		Prevention: "Don't commit code with TODO/FIXME comments",
	})
}

// Configuration helpers

func getDefaultAntiPatternConfig() *AntiPatternConfig {
	return &AntiPatternConfig{
		EnableDetection:  true,
		StrictMode:       false,
		WarningThreshold: 0.3,
		ErrorThreshold:   0.7,
		CustomPatterns:   []AntiPattern{},
		IgnorePatterns:   []string{},
	}
}
