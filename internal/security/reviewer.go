package security

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"time"
)

// SecurityReviewer performs automated security code review
type SecurityReviewer struct {
	rules []SecurityReviewRule
}

// SecurityReviewRule defines a security rule for code review
type SecurityReviewRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Severity    string   `json:"severity"` // critical, high, medium, low
	Category    string   `json:"category"` // injection, auth, crypto, etc.
	Patterns    []string `json:"patterns"`
	Fix         string   `json:"fix"`
	Examples    []string `json:"examples"`
}

// SecurityViolation represents a security violation found in code
type SecurityViolation struct {
	Rule       SecurityReviewRule `json:"rule"`
	File       string             `json:"file"`
	Line       int                `json:"line"`
	Column     int                `json:"column"`
	Code       string             `json:"code"`
	Message    string             `json:"message"`
	Severity   string             `json:"severity"`
	Confidence float64            `json:"confidence"` // 0.0 to 1.0
	Fix        string             `json:"fix"`
}

// SecurityReviewResult contains the results of a security review
type SecurityReviewResult struct {
	File       string              `json:"file"`
	Violations []SecurityViolation `json:"violations"`
	Summary    SecuritySummary     `json:"summary"`
	Duration   int64               `json:"duration_ms"`
}

// SecuritySummary provides a summary of security review results
type SecuritySummary struct {
	TotalViolations int  `json:"total_violations"`
	Critical        int  `json:"critical"`
	High            int  `json:"high"`
	Medium          int  `json:"medium"`
	Low             int  `json:"low"`
	Passed          bool `json:"passed"`
}

// NewSecurityReviewer creates a new security reviewer with default rules
func NewSecurityReviewer() *SecurityReviewer {
	return &SecurityReviewer{
		rules: getDefaultSecurityReviewRules(),
	}
}

// ReviewFile performs security review on a single file
func (sr *SecurityReviewer) ReviewFile(filePath string) (*SecurityReviewResult, error) {
	startTime := time.Now()

	// Parse the Go file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filePath, err)
	}

	var violations []SecurityViolation

	// Walk the AST and check for security violations
	ast.Walk(&securityVisitor{
		reviewer:   sr,
		fset:       fset,
		filePath:   filePath,
		violations: &violations,
	}, node)

	// Calculate summary
	summary := calculateSummary(violations)

	duration := time.Since(startTime).Milliseconds()

	return &SecurityReviewResult{
		File:       filePath,
		Violations: violations,
		Summary:    summary,
		Duration:   duration,
	}, nil
}

// ReviewDirectory performs security review on all Go files in a directory
func (sr *SecurityReviewer) ReviewDirectory(dirPath string) ([]SecurityReviewResult, error) {
	var results []SecurityReviewResult

	// Find all Go files
	goFiles, err := filepath.Glob(filepath.Join(dirPath, "**/*.go"))
	if err != nil {
		return nil, fmt.Errorf("failed to find Go files: %w", err)
	}

	for _, file := range goFiles {
		result, err := sr.ReviewFile(file)
		if err != nil {
			// Log error but continue with other files
			fmt.Printf("Warning: failed to review file %s: %v\n", file, err)
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

// securityVisitor implements ast.Visitor for security analysis
type securityVisitor struct {
	reviewer   *SecurityReviewer
	fset       *token.FileSet
	filePath   string
	violations *[]SecurityViolation
}

// Visit implements ast.Visitor
func (v *securityVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}

	// Check for security violations based on node type
	switch n := node.(type) {
	case *ast.CallExpr:
		v.checkCallExpression(n)
	case *ast.BinaryExpr:
		v.checkBinaryExpression(n)
	case *ast.AssignStmt:
		v.checkAssignment(n)
	case *ast.GenDecl:
		v.checkDeclaration(n)
	}

	return v
}

// checkCallExpression checks function calls for security issues
func (v *securityVisitor) checkCallExpression(call *ast.CallExpr) {
	// Get the function name
	funcName := getFunctionName(call.Fun)
	if funcName == "" {
		return
	}

	// Check against security rules
	for _, rule := range v.reviewer.rules {
		for _, pattern := range rule.Patterns {
			if matched, _ := regexp.MatchString(pattern, funcName); matched {
				violation := SecurityViolation{
					Rule:       rule,
					File:       v.filePath,
					Line:       v.fset.Position(call.Pos()).Line,
					Column:     v.fset.Position(call.Pos()).Column,
					Code:       getCodeSnippet(v.fset, call),
					Message:    rule.Description,
					Severity:   rule.Severity,
					Confidence: 0.9, // High confidence for pattern matches
					Fix:        rule.Fix,
				}
				*v.violations = append(*v.violations, violation)
			}
		}
	}
}

// checkBinaryExpression checks binary expressions for security issues
func (v *securityVisitor) checkBinaryExpression(expr *ast.BinaryExpr) {
	// Check for string concatenation in SQL contexts
	if expr.Op == token.ADD {
		// This is a simplified check - in practice, you'd need more context
		// to determine if this is a SQL injection risk
	}
}

// checkAssignment checks variable assignments for security issues
func (v *securityVisitor) checkAssignment(stmt *ast.AssignStmt) {
	// Check for assignments that might introduce security vulnerabilities
	for _, expr := range stmt.Rhs {
		if call, ok := expr.(*ast.CallExpr); ok {
			v.checkCallExpression(call)
		}
	}
}

// checkDeclaration checks variable declarations for security issues
func (v *securityVisitor) checkDeclaration(decl *ast.GenDecl) {
	// Check for security-related variable declarations
}

// getDefaultSecurityRules returns the default set of security rules
func getDefaultSecurityReviewRules() []SecurityReviewRule {
	return []SecurityReviewRule{
		// SEC-1: Database query security
		{
			ID:          "SEC-1",
			Name:        "SQL Injection Prevention",
			Description: "Database queries must use parameterized statements",
			Severity:    "critical",
			Category:    "injection",
			Patterns: []string{
				`fmt\.Sprintf.*SELECT.*%s`,
				`fmt\.Sprintf.*INSERT.*%s`,
				`fmt\.Sprintf.*UPDATE.*%s`,
				`fmt\.Sprintf.*DELETE.*%s`,
				`strings\.Replace.*SELECT`,
				`strings\.Replace.*INSERT`,
				`strings\.Replace.*UPDATE`,
				`strings\.Replace.*DELETE`,
				`exec.*SELECT.*\+`,
				`exec.*INSERT.*\+`,
				`exec.*UPDATE.*\+`,
				`exec.*DELETE.*\+`,
			},
			Fix: "Use database/sql prepared statements or ORM parameterized queries",
			Examples: []string{
				"❌ db.Exec(fmt.Sprintf(\"SELECT * FROM users WHERE id = %s\", userID))",
				"✅ db.Exec(\"SELECT * FROM users WHERE id = ?\", userID)",
			},
		},

		// SEC-2: Input validation
		{
			ID:          "SEC-2",
			Name:        "Input Validation",
			Description: "All external input must be validated",
			Severity:    "high",
			Category:    "validation",
			Patterns: []string{
				`r\.FormValue\(.*\)`,
				`r\.URL\.Query\(\)\.Get\(.*\)`,
				`r\.Header\.Get\(.*\)`,
				`json\.Unmarshal\(.*r\.Body`,
			},
			Fix: "Validate all input using whitelist patterns, length limits, and type checking",
			Examples: []string{
				"❌ userID := r.FormValue(\"id\")",
				"✅ userID := validateAndSanitizeInput(r.FormValue(\"id\"))",
			},
		},

		// SEC-3: Error handling
		{
			ID:          "SEC-3",
			Name:        "Secure Error Handling",
			Description: "Error messages must not leak sensitive information",
			Severity:    "medium",
			Category:    "information_disclosure",
			Patterns: []string{
				`fmt\.Errorf.*password`,
				`fmt\.Errorf.*secret`,
				`fmt\.Errorf.*token`,
				`fmt\.Errorf.*key`,
				`errors\.New.*password`,
				`errors\.New.*secret`,
			},
			Fix: "Use generic error messages for users, detailed logs for debugging",
			Examples: []string{
				"❌ return fmt.Errorf(\"invalid password: %s\", password)",
				"✅ return errors.New(\"authentication failed\")",
			},
		},

		// SEC-4: Authentication
		{
			ID:          "SEC-4",
			Name:        "Authentication Security",
			Description: "Authentication must be properly implemented",
			Severity:    "critical",
			Category:    "authentication",
			Patterns: []string{
				`password.*==.*password`,
				`strings\.Equal.*password`,
				`crypto/md5`,
				`crypto/sha1`,
			},
			Fix: "Use secure password hashing (bcrypt, argon2) and proper authentication flows",
			Examples: []string{
				"❌ if password == storedPassword { ... }",
				"✅ if bcrypt.CompareHashAndPassword(hashedPassword, []byte(password)) == nil { ... }",
			},
		},

		// SEC-5: Cryptography
		{
			ID:          "SEC-5",
			Name:        "Cryptographic Security",
			Description: "Use secure cryptographic functions",
			Severity:    "high",
			Category:    "cryptography",
			Patterns: []string{
				`crypto/md5`,
				`crypto/sha1`,
				`DES\.NewCipher`,
				`RC4\.NewCipher`,
			},
			Fix: "Use SHA-256 or stronger for hashing, AES for encryption",
			Examples: []string{
				"❌ md5.New()",
				"✅ sha256.New()",
			},
		},

		// SEC-6: File operations
		{
			ID:          "SEC-6",
			Name:        "Path Traversal Prevention",
			Description: "File operations must prevent path traversal",
			Severity:    "high",
			Category:    "path_traversal",
			Patterns: []string{
				`os\.Open\(.*\+`,
				`ioutil\.ReadFile\(.*\+`,
				`filepath\.Join\(.*r\.FormValue`,
			},
			Fix: "Validate and sanitize file paths, use filepath.Join with validated inputs",
			Examples: []string{
				"❌ os.Open(\"/uploads/\" + filename)",
				"✅ os.Open(filepath.Join(\"/uploads/\", sanitizeFilename(filename)))",
			},
		},

		// SEC-7: HTTP security
		{
			ID:          "SEC-7",
			Name:        "HTTP Security Headers",
			Description: "HTTP responses must include security headers",
			Severity:    "medium",
			Category:    "http_security",
			Patterns: []string{
				`w\.Header\(\)\.Set\(.*Content-Security-Policy`,
				`w\.Header\(\)\.Set\(.*X-Frame-Options`,
				`w\.Header\(\)\.Set\(.*X-Content-Type-Options`,
			},
			Fix: "Include security headers: CSP, X-Frame-Options, X-Content-Type-Options, etc.",
			Examples: []string{
				"❌ w.Header().Set(\"Content-Type\", \"text/html\")",
				"✅ w.Header().Set(\"Content-Security-Policy\", \"default-src 'self'\")",
			},
		},

		// SEC-8: Logging security
		{
			ID:          "SEC-8",
			Name:        "Secure Logging",
			Description: "Logs must not contain sensitive information",
			Severity:    "medium",
			Category:    "information_disclosure",
			Patterns: []string{
				`log\.Printf.*password`,
				`log\.Printf.*secret`,
				`log\.Printf.*token`,
				`fmt\.Printf.*password`,
			},
			Fix: "Sanitize logs to remove sensitive information",
			Examples: []string{
				"❌ log.Printf(\"User %s password: %s\", username, password)",
				"✅ log.Printf(\"User %s authentication attempt\", username)",
			},
		},

		// SEC-9: Rate limiting
		{
			ID:          "SEC-9",
			Name:        "Rate Limiting",
			Description: "API endpoints must implement rate limiting",
			Severity:    "medium",
			Category:    "rate_limiting",
			Patterns: []string{
				`http\.HandleFunc\(.*func\(w http\.ResponseWriter, r \*http\.Request\)`,
			},
			Fix: "Implement rate limiting middleware for all API endpoints",
			Examples: []string{
				"❌ http.HandleFunc(\"/api/\", handler)",
				"✅ http.HandleFunc(\"/api/\", rateLimitMiddleware(handler))",
			},
		},

		// SEC-10: CORS security
		{
			ID:          "SEC-10",
			Name:        "CORS Configuration",
			Description: "CORS must be properly configured",
			Severity:    "medium",
			Category:    "cors",
			Patterns: []string{
				`Access-Control-Allow-Origin.*\*`,
			},
			Fix: "Use specific origins instead of wildcard for CORS",
			Examples: []string{
				"❌ w.Header().Set(\"Access-Control-Allow-Origin\", \"*\")",
				"✅ w.Header().Set(\"Access-Control-Allow-Origin\", \"https://example.com\")",
			},
		},
	}
}

// Helper functions

func getFunctionName(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		if ident, ok := f.X.(*ast.Ident); ok {
			return ident.Name + "." + f.Sel.Name
		}
	}
	return ""
}

func getCodeSnippet(fset *token.FileSet, node ast.Node) string {
	// This is a simplified implementation
	// In practice, you'd want to extract the actual code snippet
	return fmt.Sprintf("Code at line %d", fset.Position(node.Pos()).Line)
}

func calculateSummary(violations []SecurityViolation) SecuritySummary {
	summary := SecuritySummary{
		TotalViolations: len(violations),
	}

	for _, v := range violations {
		switch v.Severity {
		case "critical":
			summary.Critical++
		case "high":
			summary.High++
		case "medium":
			summary.Medium++
		case "low":
			summary.Low++
		}
	}

	// Pass if no critical or high severity violations
	summary.Passed = summary.Critical == 0 && summary.High == 0

	return summary
}

// ReviewPackage performs security review on a Go package
func (sr *SecurityReviewer) ReviewPackage(packagePath string) ([]SecurityReviewResult, error) {
	return sr.ReviewDirectory(packagePath)
}

// GetSecurityRules returns all configured security rules
func (sr *SecurityReviewer) GetSecurityRules() []SecurityReviewRule {
	return sr.rules
}

// AddSecurityRule adds a new security rule
func (sr *SecurityReviewer) AddSecurityRule(rule SecurityReviewRule) {
	sr.rules = append(sr.rules, rule)
}

// RemoveSecurityRule removes a security rule by ID
func (sr *SecurityReviewer) RemoveSecurityRule(ruleID string) {
	for i, rule := range sr.rules {
		if rule.ID == ruleID {
			sr.rules = append(sr.rules[:i], sr.rules[i+1:]...)
			break
		}
	}
}
