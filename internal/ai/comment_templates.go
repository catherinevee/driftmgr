package ai

import (
	"fmt"
	"strings"
)

// CommentTemplate provides AI-optimized comment templates for different code patterns
type CommentTemplate struct {
	Type        string            `json:"type"`
	Pattern     string            `json:"pattern"`
	Template    string            `json:"template"`
	Examples    []string          `json:"examples"`
	Constraints []string          `json:"constraints"`
	Security    []SecurityNote    `json:"security"`
	Performance []PerformanceNote `json:"performance"`
}

// SecurityNote provides security-related guidance for code comments
type SecurityNote struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Example     string `json:"example"`
}

// PerformanceNote provides performance-related guidance for code comments
type PerformanceNote struct {
	Metric      string `json:"metric"`
	Description string `json:"description"`
	Example     string `json:"example"`
}

// GetCommentTemplate returns the appropriate comment template for a given code pattern
func GetCommentTemplate(pattern string) *CommentTemplate {
	templates := getCommentTemplates()

	for _, template := range templates {
		if strings.Contains(pattern, template.Pattern) {
			return &template
		}
	}

	// Return default template if no specific pattern matches
	return &CommentTemplate{
		Type:     "default",
		Template: getDefaultTemplate(),
	}
}

// getCommentTemplates returns all available comment templates
func getCommentTemplates() []CommentTemplate {
	return []CommentTemplate{
		{
			Type:     "http_handler",
			Pattern:  "func.*http.ResponseWriter.*http.Request",
			Template: getHTTPHandlerTemplate(),
			Examples: []string{
				"// handleUserLogin authenticates user credentials and returns JWT token\n//\n// Security: Validates input, rate limits requests, logs authentication attempts\n// Performance: Completes within 200ms, supports 1000 req/sec\n// Error Handling: Returns generic errors to users, detailed logs for debugging\nfunc handleUserLogin(w http.ResponseWriter, r *http.Request) {",
			},
			Constraints: []string{
				"Maximum 50 lines per handler",
				"Input validation required",
				"Rate limiting mandatory",
				"Structured logging only",
			},
			Security: []SecurityNote{
				{
					Type:        "input_validation",
					Description: "Validate all input parameters",
					Example:     "// SEC-2: Input validation prevents injection attacks",
				},
				{
					Type:        "authentication",
					Description: "Implement proper authentication",
					Example:     "// SEC-4: JWT token validation with expiration check",
				},
			},
			Performance: []PerformanceNote{
				{
					Metric:      "latency",
					Description: "Response time under 200ms",
					Example:     "// Performance: Completes within 200ms for 95th percentile",
				},
			},
		},
		{
			Type:     "database_operation",
			Pattern:  "func.*db.*Query|func.*db.*Exec",
			Template: getDatabaseTemplate(),
			Examples: []string{
				"// getUserByID retrieves user information from database\n//\n// Security: Uses parameterized queries to prevent SQL injection\n// Performance: Indexed query, completes within 50ms\n// Error Handling: Returns structured errors, logs database issues\nfunc getUserByID(db *sql.DB, userID string) (*User, error) {",
			},
			Constraints: []string{
				"Parameterized queries only",
				"Connection pooling required",
				"Transaction management for multi-statement operations",
				"Query timeout configuration",
			},
			Security: []SecurityNote{
				{
					Type:        "sql_injection",
					Description: "Use parameterized queries",
					Example:     "// SEC-1: Parameterized query prevents SQL injection",
				},
			},
			Performance: []PerformanceNote{
				{
					Metric:      "query_time",
					Description: "Database queries under 50ms",
					Example:     "// Performance: Indexed query completes within 50ms",
				},
			},
		},
		{
			Type:     "api_client",
			Pattern:  "func.*http.Client|func.*http.Get|func.*http.Post",
			Template: getAPIClientTemplate(),
			Examples: []string{
				"// callExternalAPI makes authenticated request to external service\n//\n// Security: Validates SSL certificates, uses secure headers\n// Performance: 5 second timeout, retry with exponential backoff\n// Error Handling: Distinguishes between retryable and permanent errors\nfunc callExternalAPI(client *http.Client, url string) (*Response, error) {",
			},
			Constraints: []string{
				"Timeout configuration required",
				"Retry logic with exponential backoff",
				"SSL certificate validation",
				"Request/response logging",
			},
			Security: []SecurityNote{
				{
					Type:        "ssl_validation",
					Description: "Validate SSL certificates",
					Example:     "// SEC-5: SSL certificate validation prevents MITM attacks",
				},
			},
			Performance: []PerformanceNote{
				{
					Metric:      "timeout",
					Description: "5 second timeout with retries",
					Example:     "// Performance: 5s timeout, 3 retries with exponential backoff",
				},
			},
		},
		{
			Type:     "file_operation",
			Pattern:  "func.*os.Open|func.*ioutil.ReadFile|func.*os.WriteFile",
			Template: getFileOperationTemplate(),
			Examples: []string{
				"// readConfigFile loads configuration from secure file path\n//\n// Security: Validates file path, prevents directory traversal\n// Performance: Caches file content, 10ms read time\n// Error Handling: Returns structured errors, logs file access\nfunc readConfigFile(filePath string) (*Config, error) {",
			},
			Constraints: []string{
				"Path validation required",
				"File size limits",
				"Permission checks",
				"Atomic operations for writes",
			},
			Security: []SecurityNote{
				{
					Type:        "path_traversal",
					Description: "Validate file paths",
					Example:     "// SEC-6: Path validation prevents directory traversal",
				},
			},
			Performance: []PerformanceNote{
				{
					Metric:      "file_size",
					Description: "File size limits",
					Example:     "// Performance: Max 10MB file size, cached for 1 hour",
				},
			},
		},
		{
			Type:     "cryptographic_function",
			Pattern:  "func.*crypto|func.*hash|func.*encrypt",
			Template: getCryptoTemplate(),
			Examples: []string{
				"// hashPassword creates secure password hash using Argon2\n//\n// Security: Uses Argon2id with recommended parameters\n// Performance: 100ms hash time, 64MB memory usage\n// Error Handling: Returns error on hash failure, never logs passwords\nfunc hashPassword(password string) (string, error) {",
			},
			Constraints: []string{
				"Use approved algorithms only",
				"Proper key management",
				"Secure random number generation",
				"Never log sensitive data",
			},
			Security: []SecurityNote{
				{
					Type:        "cryptography",
					Description: "Use secure algorithms",
					Example:     "// SEC-5: Argon2id with recommended parameters",
				},
			},
			Performance: []PerformanceNote{
				{
					Metric:      "hash_time",
					Description: "Hash time under 100ms",
					Example:     "// Performance: 100ms hash time, 64MB memory",
				},
			},
		},
		{
			Type:     "logging_function",
			Pattern:  "func.*log|func.*Log|func.*Print",
			Template: getLoggingTemplate(),
			Examples: []string{
				"// logSecurityEvent records security-related events\n//\n// Security: Sanitizes sensitive data, uses structured logging\n// Performance: Async logging, 1ms overhead\n// Error Handling: Never fails, graceful degradation\nfunc logSecurityEvent(event string, details map[string]interface{}) {",
			},
			Constraints: []string{
				"Structured logging only",
				"Sanitize sensitive data",
				"Async logging for performance",
				"Never log passwords or tokens",
			},
			Security: []SecurityNote{
				{
					Type:        "data_sanitization",
					Description: "Sanitize sensitive data",
					Example:     "// SEC-8: Sensitive data sanitized before logging",
				},
			},
			Performance: []PerformanceNote{
				{
					Metric:      "logging_overhead",
					Description: "Minimal logging overhead",
					Example:     "// Performance: Async logging, 1ms overhead",
				},
			},
		},
	}
}

// getDefaultTemplate returns the default comment template
func getDefaultTemplate() string {
	return `// [FunctionName] [brief description of what the function does]
//
// Usage:
//   result, err := [FunctionName](params)
//   if err != nil {
//       return err
//   }
//
// Security: [security considerations and validations]
// Performance: [performance characteristics and limits]
// Error Handling: [error handling strategy and recovery]
func [FunctionName](params) (returnType, error) {`
}

// getHTTPHandlerTemplate returns the HTTP handler comment template
func getHTTPHandlerTemplate() string {
	return `// [HandlerName] handles [specific HTTP operation] requests
//
// Usage:
//   http.HandleFunc("/api/[endpoint]", [HandlerName])
//
// Security: Input validation, authentication, rate limiting
// Performance: [response time]ms response time, [throughput] req/sec
// Error Handling: Generic user messages, detailed logs for debugging
func [HandlerName](w http.ResponseWriter, r *http.Request) {`
}

// getDatabaseTemplate returns the database operation comment template
func getDatabaseTemplate() string {
	return `// [OperationName] [description of database operation]
//
// Usage:
//   result, err := [OperationName](db, params)
//   if err != nil {
//       return nil, fmt.Errorf("database operation failed: %w", err)
//   }
//
// Security: Parameterized queries prevent SQL injection
// Performance: [query time]ms query time, uses [indexes/optimizations]
// Error Handling: Returns structured errors, logs database issues
func [OperationName](db *sql.DB, params) (returnType, error) {`
}

// getAPIClientTemplate returns the API client comment template
func getAPIClientTemplate() string {
	return `// [ClientName] makes authenticated request to [external service]
//
// Usage:
//   client := &http.Client{Timeout: 5 * time.Second}
//   response, err := [ClientName](client, url, params)
//
// Security: SSL validation, secure headers, no credential logging
// Performance: [timeout]s timeout, [retries] retries with exponential backoff
// Error Handling: Distinguishes retryable vs permanent errors
func [ClientName](client *http.Client, url string, params) (*Response, error) {`
}

// getFileOperationTemplate returns the file operation comment template
func getFileOperationTemplate() string {
	return `// [OperationName] [description of file operation]
//
// Usage:
//   data, err := [OperationName](filePath)
//   if err != nil {
//       return nil, fmt.Errorf("file operation failed: %w", err)
//   }
//
// Security: Path validation prevents directory traversal
// Performance: [operation time]ms operation time, [caching strategy]
// Error Handling: Returns structured errors, logs file access
func [OperationName](filePath string) (returnType, error) {`
}

// getCryptoTemplate returns the cryptographic function comment template
func getCryptoTemplate() string {
	return `// [FunctionName] performs [cryptographic operation] using [algorithm]
//
// Usage:
//   result, err := [FunctionName](input)
//   if err != nil {
//       return "", fmt.Errorf("crypto operation failed: %w", err)
//   }
//
// Security: Uses [algorithm] with [parameters], never logs sensitive data
// Performance: [operation time]ms operation time, [memory usage]MB memory
// Error Handling: Returns error on failure, never exposes sensitive data
func [FunctionName](input string) (string, error) {`
}

// getLoggingTemplate returns the logging function comment template
func getLoggingTemplate() string {
	return `// [LogFunction] records [type of event] with structured logging
//
// Usage:
//   [LogFunction](level, message, fields)
//
// Security: Sanitizes sensitive data, uses structured format
// Performance: Async logging, [overhead]ms overhead
// Error Handling: Never fails, graceful degradation
func [LogFunction](level string, message string, fields map[string]interface{}) {`
}

// GenerateComment generates an AI-optimized comment for a given function
func GenerateComment(functionSignature string, functionType string) string {
	template := GetCommentTemplate(functionSignature)

	// Replace placeholders with actual function details
	comment := template.Template

	// Add security notes if applicable
	if len(template.Security) > 0 {
		comment += "\n//\n// Security Notes:"
		for _, note := range template.Security {
			comment += fmt.Sprintf("\n// - %s: %s", note.Type, note.Description)
		}
	}

	// Add performance notes if applicable
	if len(template.Performance) > 0 {
		comment += "\n//\n// Performance Notes:"
		for _, note := range template.Performance {
			comment += fmt.Sprintf("\n// - %s: %s", note.Metric, note.Description)
		}
	}

	// Add constraints if applicable
	if len(template.Constraints) > 0 {
		comment += "\n//\n// Constraints:"
		for _, constraint := range template.Constraints {
			comment += fmt.Sprintf("\n// - %s", constraint)
		}
	}

	return comment
}

// ValidateComment checks if a comment follows AI-optimized patterns
func ValidateComment(comment string) []string {
	var issues []string

	// Check for required sections
	requiredSections := []string{"Usage:", "Security:", "Performance:", "Error Handling:"}
	for _, section := range requiredSections {
		if !strings.Contains(comment, section) {
			issues = append(issues, fmt.Sprintf("Missing required section: %s", section))
		}
	}

	// Check for security considerations
	if !strings.Contains(comment, "Security:") {
		issues = append(issues, "Missing security considerations")
	}

	// Check for performance characteristics
	if !strings.Contains(comment, "Performance:") {
		issues = append(issues, "Missing performance characteristics")
	}

	// Check for error handling strategy
	if !strings.Contains(comment, "Error Handling:") {
		issues = append(issues, "Missing error handling strategy")
	}

	// Check for usage examples
	if !strings.Contains(comment, "Usage:") {
		issues = append(issues, "Missing usage examples")
	}

	return issues
}

// CommentAnalyzer analyzes code comments for AI optimization
type CommentAnalyzer struct {
	templates []CommentTemplate
}

// NewCommentAnalyzer creates a new comment analyzer
func NewCommentAnalyzer() *CommentAnalyzer {
	return &CommentAnalyzer{
		templates: getCommentTemplates(),
	}
}

// AnalyzeFunction analyzes a function and suggests comment improvements
func (ca *CommentAnalyzer) AnalyzeFunction(functionCode string) *CommentAnalysis {
	analysis := &CommentAnalysis{
		FunctionCode: functionCode,
		Issues:       []string{},
		Suggestions:  []string{},
		Score:        0,
	}

	// Extract function signature
	signature := extractFunctionSignature(functionCode)
	if signature == "" {
		analysis.Issues = append(analysis.Issues, "Could not extract function signature")
		return analysis
	}

	// Find appropriate template
	template := GetCommentTemplate(signature)

	// Check if comment exists
	if !hasComment(functionCode) {
		analysis.Issues = append(analysis.Issues, "Function lacks documentation comment")
		analysis.Suggestions = append(analysis.Suggestions, "Add comprehensive comment using template")
		analysis.Suggestions = append(analysis.Suggestions, template.Template)
		return analysis
	}

	// Extract existing comment
	comment := extractComment(functionCode)

	// Validate comment
	issues := ValidateComment(comment)
	analysis.Issues = append(analysis.Issues, issues...)

	// Calculate score
	analysis.Score = calculateCommentScore(comment, template)

	// Generate suggestions
	analysis.Suggestions = generateSuggestions(comment, template)

	return analysis
}

// CommentAnalysis contains the results of comment analysis
type CommentAnalysis struct {
	FunctionCode string   `json:"function_code"`
	Issues       []string `json:"issues"`
	Suggestions  []string `json:"suggestions"`
	Score        int      `json:"score"` // 0-100
}

// Helper functions

func extractFunctionSignature(code string) string {
	// Simplified implementation - in practice, you'd use AST parsing
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "func ") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func hasComment(code string) bool {
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			return true
		}
		if strings.HasPrefix(trimmed, "func ") {
			break
		}
	}
	return false
}

func extractComment(code string) string {
	lines := strings.Split(code, "\n")
	var commentLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			commentLines = append(commentLines, line)
		} else if strings.HasPrefix(trimmed, "func ") {
			break
		}
	}

	return strings.Join(commentLines, "\n")
}

func calculateCommentScore(comment string, template *CommentTemplate) int {
	score := 0

	// Check for required sections (20 points each)
	requiredSections := []string{"Usage:", "Security:", "Performance:", "Error Handling:"}
	for _, section := range requiredSections {
		if strings.Contains(comment, section) {
			score += 20
		}
	}

	// Check for security considerations (10 points)
	if strings.Contains(comment, "Security:") {
		score += 10
	}

	// Check for performance characteristics (10 points)
	if strings.Contains(comment, "Performance:") {
		score += 10
	}

	return score
}

func generateSuggestions(comment string, template *CommentTemplate) []string {
	var suggestions []string

	// Check for missing sections
	if !strings.Contains(comment, "Usage:") {
		suggestions = append(suggestions, "Add usage examples with code snippets")
	}

	if !strings.Contains(comment, "Security:") {
		suggestions = append(suggestions, "Add security considerations and validations")
	}

	if !strings.Contains(comment, "Performance:") {
		suggestions = append(suggestions, "Add performance characteristics and limits")
	}

	if !strings.Contains(comment, "Error Handling:") {
		suggestions = append(suggestions, "Add error handling strategy and recovery")
	}

	// Add template-specific suggestions
	if len(template.Security) > 0 {
		suggestions = append(suggestions, "Include security notes: "+strings.Join(getSecurityTypes(template.Security), ", "))
	}

	if len(template.Performance) > 0 {
		suggestions = append(suggestions, "Include performance notes: "+strings.Join(getPerformanceMetrics(template.Performance), ", "))
	}

	return suggestions
}

func getSecurityTypes(security []SecurityNote) []string {
	var types []string
	for _, note := range security {
		types = append(types, note.Type)
	}
	return types
}

func getPerformanceMetrics(performance []PerformanceNote) []string {
	var metrics []string
	for _, note := range performance {
		metrics = append(metrics, note.Metric)
	}
	return metrics
}
