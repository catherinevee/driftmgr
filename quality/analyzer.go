package quality

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

// Analyzer analyzes code quality metrics
type Analyzer struct {
	projectRoot string
	thresholds  Thresholds
}

// Thresholds defines quality thresholds
type Thresholds struct {
	CyclomaticComplexity  int
	CognitiveComplexity   int
	MaxFileLines          int
	MaxFunctionLines      int
	MaxFunctionArguments  int
	MaxNestingDepth       int
	MinTestCoverage       float64
	MaxDuplication        float64
}

// DefaultThresholds returns recommended thresholds
func DefaultThresholds() Thresholds {
	return Thresholds{
		CyclomaticComplexity:  10,
		CognitiveComplexity:   15,
		MaxFileLines:          500,
		MaxFunctionLines:      50,
		MaxFunctionArguments:  5,
		MaxNestingDepth:       4,
		MinTestCoverage:       80.0,
		MaxDuplication:        5.0,
	}
}

// NewAnalyzer creates a new quality analyzer
func NewAnalyzer(projectRoot string) *Analyzer {
	return &Analyzer{
		projectRoot: projectRoot,
		thresholds:  DefaultThresholds(),
	}
}

// FileMetrics contains metrics for a single file
type FileMetrics struct {
	Path                 string
	Lines                int
	Functions            []FunctionMetrics
	CyclomaticComplexity int
	CognitiveComplexity  int
	Imports              int
	Comments             int
	TestCoverage         float64
	Violations           []Violation
}

// FunctionMetrics contains metrics for a function
type FunctionMetrics struct {
	Name                 string
	Lines                int
	Arguments            int
	CyclomaticComplexity int
	CognitiveComplexity  int
	NestingDepth         int
	Returns              int
}

// Violation represents a quality violation
type Violation struct {
	Type        string
	Severity    string
	File        string
	Line        int
	Column      int
	Message     string
	Suggestion  string
}

// AnalyzeFile analyzes a single Go file
func (a *Analyzer) AnalyzeFile(path string) (*FileMetrics, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	metrics := &FileMetrics{
		Path:       path,
		Lines:      countLines(string(src)),
		Functions:  []FunctionMetrics{},
		Violations: []Violation{},
	}

	// Analyze the AST
	visitor := &astVisitor{
		fset:       fset,
		metrics:    metrics,
		thresholds: a.thresholds,
	}
	ast.Walk(visitor, node)

	// Check file-level violations
	a.checkFileViolations(metrics)

	return metrics, nil
}

// astVisitor walks the AST and collects metrics
type astVisitor struct {
	fset         *token.FileSet
	metrics      *FileMetrics
	thresholds   Thresholds
	currentFunc  *FunctionMetrics
	nestingDepth int
}

func (v *astVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.FuncDecl:
		return v.visitFunction(n)
	case *ast.IfStmt:
		v.visitIf(n)
	case *ast.ForStmt, *ast.RangeStmt:
		v.visitLoop(n)
	case *ast.SwitchStmt, *ast.TypeSwitchStmt:
		v.visitSwitch(n)
	case *ast.ImportSpec:
		v.metrics.Imports++
	}
	return v
}

func (v *astVisitor) visitFunction(fn *ast.FuncDecl) ast.Visitor {
	funcMetrics := FunctionMetrics{
		Name:      fn.Name.Name,
		Lines:     countFunctionLines(v.fset, fn),
		Arguments: countArguments(fn),
	}

	// Calculate complexity
	funcMetrics.CyclomaticComplexity = calculateCyclomaticComplexity(fn)
	funcMetrics.CognitiveComplexity = calculateCognitiveComplexity(fn)
	funcMetrics.NestingDepth = calculateMaxNesting(fn)

	// Check for violations
	v.checkFunctionViolations(&funcMetrics, fn)

	v.metrics.Functions = append(v.metrics.Functions, funcMetrics)
	v.currentFunc = &funcMetrics

	// Continue visiting
	return v
}

func (v *astVisitor) visitIf(stmt *ast.IfStmt) {
	v.metrics.CyclomaticComplexity++
	v.nestingDepth++
	v.metrics.CognitiveComplexity += 1 + v.nestingDepth
}

func (v *astVisitor) visitLoop(stmt ast.Node) {
	v.metrics.CyclomaticComplexity++
	v.nestingDepth++
	v.metrics.CognitiveComplexity += 1 + v.nestingDepth
}

func (v *astVisitor) visitSwitch(stmt ast.Node) {
	v.metrics.CyclomaticComplexity++
	v.metrics.CognitiveComplexity++
}

func (v *astVisitor) checkFunctionViolations(fm *FunctionMetrics, fn *ast.FuncDecl) {
	pos := v.fset.Position(fn.Pos())

	// Check function length
	if fm.Lines > v.thresholds.MaxFunctionLines {
		v.metrics.Violations = append(v.metrics.Violations, Violation{
			Type:     "function_too_long",
			Severity: "warning",
			File:     v.metrics.Path,
			Line:     pos.Line,
			Message:  fmt.Sprintf("Function %s has %d lines, max is %d", fm.Name, fm.Lines, v.thresholds.MaxFunctionLines),
			Suggestion: "Consider breaking this function into smaller functions",
		})
	}

	// Check arguments
	if fm.Arguments > v.thresholds.MaxFunctionArguments {
		v.metrics.Violations = append(v.metrics.Violations, Violation{
			Type:     "too_many_arguments",
			Severity: "warning",
			File:     v.metrics.Path,
			Line:     pos.Line,
			Message:  fmt.Sprintf("Function %s has %d arguments, max is %d", fm.Name, fm.Arguments, v.thresholds.MaxFunctionArguments),
			Suggestion: "Consider using a struct for parameters",
		})
	}

	// Check complexity
	if fm.CyclomaticComplexity > v.thresholds.CyclomaticComplexity {
		v.metrics.Violations = append(v.metrics.Violations, Violation{
			Type:     "high_complexity",
			Severity: "error",
			File:     v.metrics.Path,
			Line:     pos.Line,
			Message:  fmt.Sprintf("Function %s has cyclomatic complexity %d, max is %d", fm.Name, fm.CyclomaticComplexity, v.thresholds.CyclomaticComplexity),
			Suggestion: "Simplify the logic or extract helper functions",
		})
	}
}

func (a *Analyzer) checkFileViolations(metrics *FileMetrics) {
	// Check file length
	if metrics.Lines > a.thresholds.MaxFileLines {
		metrics.Violations = append(metrics.Violations, Violation{
			Type:     "file_too_long",
			Severity: "warning",
			File:     metrics.Path,
			Line:     1,
			Message:  fmt.Sprintf("File has %d lines, max is %d", metrics.Lines, a.thresholds.MaxFileLines),
			Suggestion: "Consider splitting into multiple files",
		})
	}

	// Calculate overall complexity
	for _, fn := range metrics.Functions {
		metrics.CyclomaticComplexity += fn.CyclomaticComplexity
		metrics.CognitiveComplexity += fn.CognitiveComplexity
	}
}

// Helper functions

func countLines(src string) int {
	return strings.Count(src, "\n") + 1
}

func countFunctionLines(fset *token.FileSet, fn *ast.FuncDecl) int {
	start := fset.Position(fn.Pos()).Line
	end := fset.Position(fn.End()).Line
	return end - start + 1
}

func countArguments(fn *ast.FuncDecl) int {
	if fn.Type.Params == nil {
		return 0
	}
	
	count := 0
	for _, field := range fn.Type.Params.List {
		if field.Names == nil {
			count++ // Anonymous parameter
		} else {
			count += len(field.Names)
		}
	}
	return count
}

func calculateCyclomaticComplexity(fn *ast.FuncDecl) int {
	complexity := 1 // Base complexity
	
	ast.Inspect(fn, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
			complexity++
		case *ast.CaseClause:
			complexity++
		case *ast.FuncLit:
			return false // Don't count nested functions
		}
		return true
	})
	
	return complexity
}

func calculateCognitiveComplexity(fn *ast.FuncDecl) int {
	visitor := &cognitiveVisitor{complexity: 0, nesting: 0}
	ast.Walk(visitor, fn)
	return visitor.complexity
}

type cognitiveVisitor struct {
	complexity int
	nesting    int
}

func (v *cognitiveVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.IfStmt:
		v.complexity += 1 + v.nesting
		v.nesting++
		defer func() { v.nesting-- }()
		
	case *ast.ForStmt, *ast.RangeStmt:
		v.complexity += 1 + v.nesting
		v.nesting++
		defer func() { v.nesting-- }()
		
	case *ast.SwitchStmt, *ast.TypeSwitchStmt:
		v.complexity++
		
	case *ast.BinaryExpr:
		if n.Op == token.LAND || n.Op == token.LOR {
			v.complexity++
		}
	}
	
	return v
}

func calculateMaxNesting(fn *ast.FuncDecl) int {
	visitor := &nestingVisitor{current: 0, max: 0}
	ast.Walk(visitor, fn)
	return visitor.max
}

type nestingVisitor struct {
	current int
	max     int
}

func (v *nestingVisitor) Visit(node ast.Node) ast.Visitor {
	switch node.(type) {
	case *ast.BlockStmt:
		v.current++
		if v.current > v.max {
			v.max = v.current
		}
		defer func() { v.current-- }()
	}
	return v
}