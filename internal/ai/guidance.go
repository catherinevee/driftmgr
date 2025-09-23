package ai

import (
	"context"
	"fmt"
	"strings"
)

// AIGuidance provides intelligent guidance and constraint enforcement for AI coding
type AIGuidance struct {
	constraints    *ConstraintEnforcer
	antiPatterns   *AntiPatternDetector
	contextManager *ContextManager
	config         *GuidanceConfig
}

// GuidanceConfig contains configuration for AI guidance
type GuidanceConfig struct {
	EnableGuidance       bool    `json:"enable_guidance"`
	StrictMode           bool    `json:"strict_mode"`
	ContextWindow        int     `json:"context_window"`
	MaxSuggestions       int     `json:"max_suggestions"`
	LearningEnabled      bool    `json:"learning_enabled"`
	FeedbackEnabled      bool    `json:"feedback_enabled"`
	AutoCorrection       bool    `json:"auto_correction"`
	QualityThreshold     float64 `json:"quality_threshold"`
	SecurityThreshold    float64 `json:"security_threshold"`
	PerformanceThreshold float64 `json:"performance_threshold"`
}

// ContextManager manages context for AI guidance
type ContextManager struct {
	contexts map[string]*Context
	config   *ContextConfig
}

// Context represents a context for AI guidance
type Context struct {
	ID         string                 `json:"id"`
	Type       ContextType            `json:"type"`
	Content    string                 `json:"content"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  int64                  `json:"created_at"`
	LastUsed   int64                  `json:"last_used"`
	UsageCount int                    `json:"usage_count"`
	Relevance  float64                `json:"relevance"`
}

// ContextType represents the type of context
type ContextType string

const (
	ContextTypeCode          ContextType = "code"
	ContextTypeDocumentation ContextType = "documentation"
	ContextTypeError         ContextType = "error"
	ContextTypePattern       ContextType = "pattern"
	ContextTypeExample       ContextType = "example"
	ContextTypeConstraint    ContextType = "constraint"
)

// ContextConfig contains configuration for context management
type ContextConfig struct {
	MaxContexts        int     `json:"max_contexts"`
	RelevanceThreshold float64 `json:"relevance_threshold"`
	TTL                int64   `json:"ttl"`
	AutoCleanup        bool    `json:"auto_cleanup"`
}

// GuidanceRequest represents a request for AI guidance
type GuidanceRequest struct {
	Code        string                 `json:"code"`
	File        string                 `json:"file"`
	Task        string                 `json:"task"`
	Context     map[string]interface{} `json:"context"`
	Constraints []string               `json:"constraints"`
	Preferences GuidancePreferences    `json:"preferences"`
}

// GuidancePreferences contains user preferences for guidance
type GuidancePreferences struct {
	VerbosityLevel string   `json:"verbosity_level"` // minimal, normal, detailed
	FocusAreas     []string `json:"focus_areas"`     // security, performance, maintainability
	IgnorePatterns []string `json:"ignore_patterns"`
	CustomRules    []string `json:"custom_rules"`
	LearningMode   bool     `json:"learning_mode"`
	AutoApply      bool     `json:"auto_apply"`
}

// GuidanceResponse contains the response from AI guidance
type GuidanceResponse struct {
	Suggestions      []Suggestion           `json:"suggestions"`
	Violations       []ConstraintViolation  `json:"violations"`
	AntiPatterns     []AntiPatternMatch     `json:"anti_patterns"`
	QualityScore     float64                `json:"quality_score"`
	SecurityScore    float64                `json:"security_score"`
	PerformanceScore float64                `json:"performance_score"`
	Recommendations  []string               `json:"recommendations"`
	Context          []Context              `json:"context"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// Suggestion represents a code improvement suggestion
type Suggestion struct {
	ID          string                 `json:"id"`
	Type        SuggestionType         `json:"type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Code        string                 `json:"code"`
	Replacement string                 `json:"replacement"`
	Confidence  float64                `json:"confidence"`
	Impact      SuggestionImpact       `json:"impact"`
	Effort      SuggestionEffort       `json:"effort"`
	Priority    SuggestionPriority     `json:"priority"`
	Category    string                 `json:"category"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SuggestionType represents the type of suggestion
type SuggestionType string

const (
	SuggestionTypeRefactor      SuggestionType = "refactor"
	SuggestionTypeOptimize      SuggestionType = "optimize"
	SuggestionTypeSecurity      SuggestionType = "security"
	SuggestionTypePerformance   SuggestionType = "performance"
	SuggestionTypeStyle         SuggestionType = "style"
	SuggestionTypeDocumentation SuggestionType = "documentation"
	SuggestionTypeTest          SuggestionType = "test"
)

// SuggestionImpact represents the impact of a suggestion
type SuggestionImpact string

const (
	ImpactHigh   SuggestionImpact = "high"
	ImpactMedium SuggestionImpact = "medium"
	ImpactLow    SuggestionImpact = "low"
)

// SuggestionEffort represents the effort required for a suggestion
type SuggestionEffort string

const (
	EffortHigh   SuggestionEffort = "high"
	EffortMedium SuggestionEffort = "medium"
	EffortLow    SuggestionEffort = "low"
)

// SuggestionPriority represents the priority of a suggestion
type SuggestionPriority string

const (
	PriorityCritical SuggestionPriority = "critical"
	PriorityHigh     SuggestionPriority = "high"
	PriorityMedium   SuggestionPriority = "medium"
	PriorityLow      SuggestionPriority = "low"
)

// NewAIGuidance creates a new AI guidance system
func NewAIGuidance(config *GuidanceConfig) *AIGuidance {
	if config == nil {
		config = getDefaultGuidanceConfig()
	}

	return &AIGuidance{
		constraints:    NewConstraintEnforcer(nil),
		antiPatterns:   NewAntiPatternDetector(nil),
		contextManager: NewContextManager(nil),
		config:         config,
	}
}

// ProvideGuidance provides intelligent guidance for the given code
func (ag *AIGuidance) ProvideGuidance(ctx context.Context, request *GuidanceRequest) (*GuidanceResponse, error) {
	if !ag.config.EnableGuidance {
		return &GuidanceResponse{
			QualityScore:     100.0,
			SecurityScore:    100.0,
			PerformanceScore: 100.0,
		}, nil
	}

	// Get relevant context
	contexts := ag.contextManager.GetRelevantContext(request.Code, request.Task)

	// Check constraints
	constraintViolations, err := ag.constraints.ApplyConstraints(request.Code, request.File)
	if err != nil {
		return nil, fmt.Errorf("constraint check failed: %w", err)
	}

	// Detect anti-patterns
	antiPatternReport, err := ag.antiPatterns.DetectAntiPatterns(request.Code, request.File)
	if err != nil {
		return nil, fmt.Errorf("anti-pattern detection failed: %w", err)
	}

	// Generate suggestions
	suggestions := ag.generateSuggestions(request, constraintViolations, antiPatternReport.Matches)

	// Calculate scores
	qualityScore := ag.calculateQualityScore(constraintViolations, antiPatternReport.Matches)
	securityScore := ag.calculateSecurityScore(constraintViolations, antiPatternReport.Matches)
	performanceScore := ag.calculatePerformanceScore(constraintViolations, antiPatternReport.Matches)

	// Generate recommendations
	recommendations := ag.generateRecommendations(suggestions, constraintViolations, antiPatternReport.Matches)

	// Filter suggestions based on preferences
	suggestions = ag.filterSuggestions(suggestions, request.Preferences)

	// Limit suggestions
	if len(suggestions) > ag.config.MaxSuggestions {
		suggestions = suggestions[:ag.config.MaxSuggestions]
	}

	response := &GuidanceResponse{
		Suggestions:      suggestions,
		Violations:       constraintViolations,
		AntiPatterns:     antiPatternReport.Matches,
		QualityScore:     qualityScore,
		SecurityScore:    securityScore,
		PerformanceScore: performanceScore,
		Recommendations:  recommendations,
		Context:          contexts,
		Metadata: map[string]interface{}{
			"total_violations":    len(constraintViolations),
			"total_anti_patterns": len(antiPatternReport.Matches),
			"total_suggestions":   len(suggestions),
			"guidance_version":    "1.0",
		},
	}

	return response, nil
}

// GeneratePrompt generates an optimized prompt for AI coding
func (ag *AIGuidance) GeneratePrompt(request *GuidanceRequest) string {
	var prompt strings.Builder

	// Add task description
	prompt.WriteString(fmt.Sprintf("Task: %s\n\n", request.Task))

	// Add constraints
	constraintPrompt := ag.constraints.GenerateConstraintPrompt("")
	prompt.WriteString(constraintPrompt)
	prompt.WriteString("\n\n")

	// Add context
	if len(request.Context) > 0 {
		prompt.WriteString("CONTEXT:\n")
		for key, value := range request.Context {
			prompt.WriteString(fmt.Sprintf("- %s: %v\n", key, value))
		}
		prompt.WriteString("\n")
	}

	// Add preferences
	if len(request.Preferences.FocusAreas) > 0 {
		prompt.WriteString("FOCUS AREAS:\n")
		for _, area := range request.Preferences.FocusAreas {
			prompt.WriteString(fmt.Sprintf("- %s\n", area))
		}
		prompt.WriteString("\n")
	}

	// Add code to work with
	prompt.WriteString("CODE TO IMPROVE:\n")
	prompt.WriteString("```go\n")
	prompt.WriteString(request.Code)
	prompt.WriteString("\n```\n\n")

	// Add specific instructions
	prompt.WriteString("INSTRUCTIONS:\n")
	prompt.WriteString("1. Analyze the code for improvements\n")
	prompt.WriteString("2. Focus on the specified areas\n")
	prompt.WriteString("3. Provide specific, actionable suggestions\n")
	prompt.WriteString("4. Include code examples for improvements\n")
	prompt.WriteString("5. Explain the reasoning behind each suggestion\n")

	return prompt.String()
}

// Helper methods

func (ag *AIGuidance) generateSuggestions(request *GuidanceRequest, violations []ConstraintViolation, antiPatterns []AntiPatternMatch) []Suggestion {
	var suggestions []Suggestion

	// Generate suggestions from constraint violations
	for _, violation := range violations {
		suggestion := Suggestion{
			ID:          fmt.Sprintf("constraint_%s_%d", violation.Type, violation.Line),
			Type:        ag.mapViolationToSuggestionType(violation.Type),
			Title:       fmt.Sprintf("Fix %s violation", violation.Type),
			Description: violation.Message,
			Code:        "", // Would extract from context
			Replacement: "", // Would generate replacement
			Confidence:  0.9,
			Impact:      ag.mapSeverityToImpact(violation.Severity),
			Effort:      ag.mapViolationToEffort(violation.Type),
			Priority:    ag.mapSeverityToPriority(violation.Severity),
			Category:    violation.Type,
			Metadata: map[string]interface{}{
				"line":     violation.Line,
				"column":   violation.Column,
				"severity": violation.Severity,
			},
		}
		suggestions = append(suggestions, suggestion)
	}

	// Generate suggestions from anti-patterns
	for _, antiPattern := range antiPatterns {
		suggestion := Suggestion{
			ID:          fmt.Sprintf("antipattern_%s_%d", antiPattern.Pattern.ID, antiPattern.Line),
			Type:        ag.mapAntiPatternToSuggestionType(antiPattern.Pattern.Category),
			Title:       fmt.Sprintf("Fix %s anti-pattern", antiPattern.Pattern.Name),
			Description: antiPattern.Pattern.Description,
			Code:        antiPattern.Code,
			Replacement: "", // Would generate replacement
			Confidence:  antiPattern.Confidence,
			Impact:      ag.mapSeverityToImpact(string(antiPattern.Pattern.Severity)),
			Effort:      ag.mapAntiPatternToEffort(antiPattern.Pattern.Category),
			Priority:    ag.mapSeverityToPriority(string(antiPattern.Pattern.Severity)),
			Category:    string(antiPattern.Pattern.Category),
			Metadata: map[string]interface{}{
				"line":       antiPattern.Line,
				"column":     antiPattern.Column,
				"severity":   antiPattern.Pattern.Severity,
				"confidence": antiPattern.Confidence,
			},
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}

func (ag *AIGuidance) calculateQualityScore(violations []ConstraintViolation, antiPatterns []AntiPatternMatch) float64 {
	score := 100.0

	// Deduct points for violations
	for _, violation := range violations {
		switch violation.Severity {
		case "critical":
			score -= 20.0
		case "high":
			score -= 10.0
		case "medium":
			score -= 5.0
		case "low":
			score -= 2.0
		}
	}

	// Deduct points for anti-patterns
	for _, antiPattern := range antiPatterns {
		switch antiPattern.Pattern.Severity {
		case SeverityCritical:
			score -= 15.0 * antiPattern.Confidence
		case SeverityHigh:
			score -= 8.0 * antiPattern.Confidence
		case SeverityMedium:
			score -= 4.0 * antiPattern.Confidence
		case SeverityLow:
			score -= 1.0 * antiPattern.Confidence
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (ag *AIGuidance) calculateSecurityScore(violations []ConstraintViolation, antiPatterns []AntiPatternMatch) float64 {
	score := 100.0

	// Focus on security-related violations and anti-patterns
	for _, violation := range violations {
		if violation.Type == "security" {
			switch violation.Severity {
			case "critical":
				score -= 30.0
			case "high":
				score -= 20.0
			case "medium":
				score -= 10.0
			case "low":
				score -= 5.0
			}
		}
	}

	for _, antiPattern := range antiPatterns {
		if antiPattern.Pattern.Category == CategorySecurity {
			switch antiPattern.Pattern.Severity {
			case SeverityCritical:
				score -= 25.0 * antiPattern.Confidence
			case SeverityHigh:
				score -= 15.0 * antiPattern.Confidence
			case SeverityMedium:
				score -= 8.0 * antiPattern.Confidence
			case SeverityLow:
				score -= 3.0 * antiPattern.Confidence
			}
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (ag *AIGuidance) calculatePerformanceScore(violations []ConstraintViolation, antiPatterns []AntiPatternMatch) float64 {
	score := 100.0

	// Focus on performance-related violations and anti-patterns
	for _, violation := range violations {
		if violation.Type == "performance" {
			switch violation.Severity {
			case "critical":
				score -= 25.0
			case "high":
				score -= 15.0
			case "medium":
				score -= 8.0
			case "low":
				score -= 3.0
			}
		}
	}

	for _, antiPattern := range antiPatterns {
		if antiPattern.Pattern.Category == CategoryPerformance {
			switch antiPattern.Pattern.Severity {
			case SeverityCritical:
				score -= 20.0 * antiPattern.Confidence
			case SeverityHigh:
				score -= 12.0 * antiPattern.Confidence
			case SeverityMedium:
				score -= 6.0 * antiPattern.Confidence
			case SeverityLow:
				score -= 2.0 * antiPattern.Confidence
			}
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (ag *AIGuidance) generateRecommendations(suggestions []Suggestion, violations []ConstraintViolation, antiPatterns []AntiPatternMatch) []string {
	var recommendations []string

	// Generate recommendations based on suggestions
	priorityCounts := make(map[SuggestionPriority]int)
	for _, suggestion := range suggestions {
		priorityCounts[suggestion.Priority]++
	}

	if priorityCounts[PriorityCritical] > 0 {
		recommendations = append(recommendations, "Address critical issues immediately")
	}

	if priorityCounts[PriorityHigh] > 2 {
		recommendations = append(recommendations, "Focus on high-priority improvements")
	}

	// Generate recommendations based on categories
	categoryCounts := make(map[string]int)
	for _, suggestion := range suggestions {
		categoryCounts[suggestion.Category]++
	}

	if categoryCounts["security"] > 0 {
		recommendations = append(recommendations, "Review and fix security vulnerabilities")
	}

	if categoryCounts["performance"] > 0 {
		recommendations = append(recommendations, "Optimize performance-critical code")
	}

	if categoryCounts["complexity"] > 0 {
		recommendations = append(recommendations, "Simplify complex code structures")
	}

	return recommendations
}

func (ag *AIGuidance) filterSuggestions(suggestions []Suggestion, preferences GuidancePreferences) []Suggestion {
	var filtered []Suggestion

	for _, suggestion := range suggestions {
		// Check if suggestion should be ignored
		shouldIgnore := false
		for _, ignorePattern := range preferences.IgnorePatterns {
			if strings.Contains(strings.ToLower(suggestion.Title), strings.ToLower(ignorePattern)) {
				shouldIgnore = true
				break
			}
		}

		if shouldIgnore {
			continue
		}

		// Check focus areas
		if len(preferences.FocusAreas) > 0 {
			matchesFocus := false
			for _, focusArea := range preferences.FocusAreas {
				if strings.Contains(strings.ToLower(suggestion.Category), strings.ToLower(focusArea)) {
					matchesFocus = true
					break
				}
			}

			if !matchesFocus {
				continue
			}
		}

		filtered = append(filtered, suggestion)
	}

	return filtered
}

// Mapping helper methods

func (ag *AIGuidance) mapViolationToSuggestionType(violationType string) SuggestionType {
	switch violationType {
	case "security":
		return SuggestionTypeSecurity
	case "performance":
		return SuggestionTypePerformance
	case "length", "complexity", "nesting":
		return SuggestionTypeRefactor
	case "duplication":
		return SuggestionTypeRefactor
	default:
		return SuggestionTypeStyle
	}
}

func (ag *AIGuidance) mapAntiPatternToSuggestionType(category AntiPatternCategory) SuggestionType {
	switch category {
	case CategorySecurity:
		return SuggestionTypeSecurity
	case CategoryPerformance:
		return SuggestionTypePerformance
	case CategoryVerbosity, CategoryComplexity:
		return SuggestionTypeRefactor
	case CategoryDuplication:
		return SuggestionTypeRefactor
	case CategoryStandards:
		return SuggestionTypeStyle
	default:
		return SuggestionTypeRefactor
	}
}

func (ag *AIGuidance) mapSeverityToImpact(severity string) SuggestionImpact {
	switch severity {
	case "critical":
		return ImpactHigh
	case "high":
		return ImpactHigh
	case "medium":
		return ImpactMedium
	case "low":
		return ImpactLow
	default:
		return ImpactMedium
	}
}

func (ag *AIGuidance) mapSeverityToPriority(severity string) SuggestionPriority {
	switch severity {
	case "critical":
		return PriorityCritical
	case "high":
		return PriorityHigh
	case "medium":
		return PriorityMedium
	case "low":
		return PriorityLow
	default:
		return PriorityMedium
	}
}

func (ag *AIGuidance) mapViolationToEffort(violationType string) SuggestionEffort {
	switch violationType {
	case "length", "nesting":
		return EffortMedium
	case "complexity":
		return EffortHigh
	case "parameters":
		return EffortLow
	case "security":
		return EffortHigh
	case "performance":
		return EffortMedium
	default:
		return EffortMedium
	}
}

func (ag *AIGuidance) mapAntiPatternToEffort(category AntiPatternCategory) SuggestionEffort {
	switch category {
	case CategorySecurity:
		return EffortHigh
	case CategoryPerformance:
		return EffortMedium
	case CategoryVerbosity:
		return EffortLow
	case CategoryComplexity:
		return EffortHigh
	case CategoryDuplication:
		return EffortMedium
	case CategoryStandards:
		return EffortLow
	default:
		return EffortMedium
	}
}

// ContextManager implementation

func NewContextManager(config *ContextConfig) *ContextManager {
	if config == nil {
		config = getDefaultContextConfig()
	}

	return &ContextManager{
		contexts: make(map[string]*Context),
		config:   config,
	}
}

func (cm *ContextManager) GetRelevantContext(code string, task string) []Context {
	// Simplified implementation - would use more sophisticated matching
	var relevant []Context

	for _, context := range cm.contexts {
		if cm.isRelevant(context, code, task) {
			relevant = append(relevant, *context)
		}
	}

	return relevant
}

func (cm *ContextManager) isRelevant(context *Context, code string, task string) bool {
	// Simplified relevance calculation
	relevance := 0.0

	// Check if context content matches code patterns
	if strings.Contains(strings.ToLower(context.Content), strings.ToLower(task)) {
		relevance += 0.5
	}

	// Check if context type matches
	if context.Type == ContextTypeCode || context.Type == ContextTypeExample {
		relevance += 0.3
	}

	return relevance >= cm.config.RelevanceThreshold
}

// Configuration helpers

func getDefaultGuidanceConfig() *GuidanceConfig {
	return &GuidanceConfig{
		EnableGuidance:       true,
		StrictMode:           false,
		ContextWindow:        4000,
		MaxSuggestions:       10,
		LearningEnabled:      true,
		FeedbackEnabled:      true,
		AutoCorrection:       false,
		QualityThreshold:     80.0,
		SecurityThreshold:    90.0,
		PerformanceThreshold: 85.0,
	}
}

func getDefaultContextConfig() *ContextConfig {
	return &ContextConfig{
		MaxContexts:        1000,
		RelevanceThreshold: 0.3,
		TTL:                86400, // 24 hours
		AutoCleanup:        true,
	}
}
