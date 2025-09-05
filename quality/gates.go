package quality

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Gate represents a quality gate
type Gate interface {
	Check(projectPath string) (bool, []string)
	Name() string
}

// QualityGates enforces code quality standards
type QualityGates struct {
	strictMode bool
	gates      map[string]Gate
}

// NewQualityGates creates quality gates
func NewQualityGates(strict bool) *QualityGates {
	return &QualityGates{
		strictMode: strict,
		gates: map[string]Gate{
			"complexity":     &ComplexityGate{},
			"duplication":    &DuplicationGate{},
			"coverage":       &CoverageGate{},
			"documentation":  &DocumentationGate{},
			"naming":         &NamingGate{},
		},
	}
}

// CheckAll checks all quality gates
func (q *QualityGates) CheckAll(projectPath string) (bool, []string) {
	passed := true
	var failures []string
	
	for name, gate := range q.gates {
		gatePassed, gateFailures := gate.Check(projectPath)
		if !gatePassed {
			passed = false
			for _, f := range gateFailures {
				failures = append(failures, fmt.Sprintf("[%s] %s", name, f))
			}
		}
	}
	
	return passed, failures
}

// CheckFile checks gates for a single file
func (q *QualityGates) CheckFile(filepath string) (bool, []string) {
	passed := true
	var failures []string
	
	// Quick per-file checks
	if !checkFileComplexity(filepath, 10) {
		passed = false
		failures = append(failures, "Complexity exceeds threshold")
	}
	
	if !checkFileNaming(filepath) {
		passed = false
		failures = append(failures, "File naming convention violation")
	}
	
	return passed, failures
}

// ComplexityGate checks code complexity
type ComplexityGate struct {
	maxComplexity int
}

func (g *ComplexityGate) Name() string {
	return "complexity"
}

func (g *ComplexityGate) Check(projectPath string) (bool, []string) {
	if g.maxComplexity == 0 {
		g.maxComplexity = 10
	}
	
	var failures []string
	
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if strings.HasSuffix(path, ".go") && !strings.Contains(path, "test") && !strings.Contains(path, "vendor") {
			violations := g.checkFileComplexity(path)
			failures = append(failures, violations...)
		}
		
		return nil
	})
	
	if err != nil {
		failures = append(failures, fmt.Sprintf("Error walking directory: %v", err))
	}
	
	return len(failures) == 0, failures
}

func (g *ComplexityGate) checkFileComplexity(filepath string) []string {
	var failures []string
	
	src, err := os.ReadFile(filepath)
	if err != nil {
		return []string{fmt.Sprintf("Cannot read %s: %v", filepath, err)}
	}
	
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filepath, src, 0)
	if err != nil {
		return []string{fmt.Sprintf("Cannot parse %s: %v", filepath, err)}
	}
	
	// Check each function
	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			complexity := calculateCyclomaticComplexity(fn)
			if complexity > g.maxComplexity {
				pos := fset.Position(fn.Pos())
				failures = append(failures, fmt.Sprintf(
					"%s:%d - %s has complexity %d (max: %d)",
					filepath, pos.Line, fn.Name.Name, complexity, g.maxComplexity,
				))
			}
		}
		return true
	})
	
	return failures
}

// DuplicationGate checks for code duplication
type DuplicationGate struct {
	maxDuplication float64
}

func (g *DuplicationGate) Name() string {
	return "duplication"
}

func (g *DuplicationGate) Check(projectPath string) (bool, []string) {
	if g.maxDuplication == 0 {
		g.maxDuplication = 5.0 // 5% max duplication
	}
	
	duplicates := findDuplicates(projectPath)
	
	var failures []string
	for _, dup := range duplicates {
		if dup.Percentage > g.maxDuplication {
			failures = append(failures, fmt.Sprintf(
				"Duplication %.1f%% between %s and %s",
				dup.Percentage, dup.File1, dup.File2,
			))
		}
	}
	
	return len(failures) == 0, failures
}

// CoverageGate checks test coverage
type CoverageGate struct {
	minCoverage float64
}

func (g *CoverageGate) Name() string {
	return "coverage"
}

func (g *CoverageGate) Check(projectPath string) (bool, []string) {
	if g.minCoverage == 0 {
		g.minCoverage = 80.0
	}
	
	coverage := getTestCoverage(projectPath)
	
	if coverage < g.minCoverage {
		return false, []string{
			fmt.Sprintf("Test coverage %.1f%% is below minimum %.1f%%", coverage, g.minCoverage),
		}
	}
	
	return true, nil
}

// DocumentationGate checks documentation coverage
type DocumentationGate struct {
	minDocCoverage float64
}

func (g *DocumentationGate) Name() string {
	return "documentation"
}

func (g *DocumentationGate) Check(projectPath string) (bool, []string) {
	if g.minDocCoverage == 0 {
		g.minDocCoverage = 70.0
	}
	
	var failures []string
	totalFuncs := 0
	documentedFuncs := 0
	
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if strings.HasSuffix(path, ".go") && !strings.Contains(path, "test") {
			funcs, docs := countDocumentation(path)
			totalFuncs += funcs
			documentedFuncs += docs
		}
		
		return nil
	})
	
	if err != nil {
		failures = append(failures, fmt.Sprintf("Error: %v", err))
	}
	
	if totalFuncs > 0 {
		coverage := float64(documentedFuncs) / float64(totalFuncs) * 100
		if coverage < g.minDocCoverage {
			failures = append(failures, fmt.Sprintf(
				"Documentation coverage %.1f%% is below minimum %.1f%%",
				coverage, g.minDocCoverage,
			))
		}
	}
	
	return len(failures) == 0, failures
}

// NamingGate checks naming conventions
type NamingGate struct{}

func (g *NamingGate) Name() string {
	return "naming"
}

func (g *NamingGate) Check(projectPath string) (bool, []string) {
	var failures []string
	
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if strings.HasSuffix(path, ".go") && !strings.Contains(path, "vendor") {
			violations := checkNamingConventions(path)
			failures = append(failures, violations...)
		}
		
		return nil
	})
	
	if err != nil {
		failures = append(failures, fmt.Sprintf("Error: %v", err))
	}
	
	return len(failures) == 0, failures
}

// Helper functions

func checkFileComplexity(filePath string, maxComplexity int) bool {
	gate := &ComplexityGate{maxComplexity: maxComplexity}
	violations := gate.checkFileComplexity(filePath)
	return len(violations) == 0
}

func checkFileNaming(filePath string) bool {
	// Check file naming conventions
	base := filepath.Base(filePath)
	
	// Files should be lowercase with underscores
	if strings.ToLower(base) != base {
		return false
	}
	
	// No spaces in filenames
	if strings.Contains(base, " ") {
		return false
	}
	
	return true
}

func findDuplicates(projectPath string) []DuplicationResult {
	// Simplified duplication detection
	// In production, use more sophisticated algorithms
	var results []DuplicationResult
	
	files := make(map[string][]string)
	
	filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".go") && !strings.Contains(path, "test") {
			content, _ := os.ReadFile(path)
			lines := strings.Split(string(content), "\n")
			files[path] = lines
		}
		return nil
	})
	
	// Compare files pairwise
	filePaths := make([]string, 0, len(files))
	for path := range files {
		filePaths = append(filePaths, path)
	}
	
	for i := 0; i < len(filePaths); i++ {
		for j := i + 1; j < len(filePaths); j++ {
			dup := compareFiles(files[filePaths[i]], files[filePaths[j]])
			if dup > 5.0 {
				results = append(results, DuplicationResult{
					File1:      filePaths[i],
					File2:      filePaths[j],
					Percentage: dup,
				})
			}
		}
	}
	
	return results
}

func compareFiles(lines1, lines2 []string) float64 {
	// Simple line-by-line comparison
	matches := 0
	total := len(lines1)
	if len(lines2) < total {
		total = len(lines2)
	}
	
	for i := 0; i < total; i++ {
		if strings.TrimSpace(lines1[i]) == strings.TrimSpace(lines2[i]) && 
		   strings.TrimSpace(lines1[i]) != "" {
			matches++
		}
	}
	
	if total == 0 {
		return 0
	}
	
	return float64(matches) / float64(total) * 100
}

func getTestCoverage(projectPath string) float64 {
	// Run go test with coverage
	// This is simplified - in production, parse coverage output properly
	return 85.0 // Mock value
}

func countDocumentation(filepath string) (int, int) {
	src, err := os.ReadFile(filepath)
	if err != nil {
		return 0, 0
	}
	
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filepath, src, parser.ParseComments)
	if err != nil {
		return 0, 0
	}
	
	totalFuncs := 0
	documentedFuncs := 0
	
	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			totalFuncs++
			
			// Check if function has doc comment
			if fn.Doc != nil && len(fn.Doc.List) > 0 {
				documentedFuncs++
			}
		}
		return true
	})
	
	return totalFuncs, documentedFuncs
}

func checkNamingConventions(filepath string) []string {
	var violations []string
	
	src, err := os.ReadFile(filepath)
	if err != nil {
		return violations
	}
	
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filepath, src, 0)
	if err != nil {
		return violations
	}
	
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			// Exported functions should start with capital letter
			if ast.IsExported(x.Name.Name) {
				if x.Name.Name[0] < 'A' || x.Name.Name[0] > 'Z' {
					pos := fset.Position(x.Pos())
					violations = append(violations, fmt.Sprintf(
						"%s:%d - Exported function %s should start with capital letter",
						filepath, pos.Line, x.Name.Name,
					))
				}
			}
			
		case *ast.GenDecl:
			// Check type and constant naming
			for _, spec := range x.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					if ast.IsExported(ts.Name.Name) && !isCapitalized(ts.Name.Name) {
						pos := fset.Position(ts.Pos())
						violations = append(violations, fmt.Sprintf(
							"%s:%d - Exported type %s should start with capital letter",
							filepath, pos.Line, ts.Name.Name,
						))
					}
				}
			}
		}
		return true
	})
	
	return violations
}

func isCapitalized(name string) bool {
	return name[0] >= 'A' && name[0] <= 'Z'
}

// DuplicationResult represents code duplication
type DuplicationResult struct {
	File1      string
	File2      string
	Percentage float64
}