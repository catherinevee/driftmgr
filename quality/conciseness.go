package quality

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

// ConcisenessAnalyzer analyzes code for conciseness improvements
type ConcisenessAnalyzer struct {
	issues []ConcisenessIssue
}

// ConcisenessIssue represents a conciseness improvement opportunity
type ConcisenessIssue struct {
	Type       string
	File       string
	Line       int
	Message    string
	Original   string
	Improved   string
	Suggestion string
}

// NewConcisenessAnalyzer creates a new analyzer
func NewConcisenessAnalyzer() *ConcisenessAnalyzer {
	return &ConcisenessAnalyzer{
		issues: []ConcisenessIssue{},
	}
}

// AnalyzeFile analyzes a file for conciseness
func (c *ConcisenessAnalyzer) AnalyzeFile(filepath string) ([]ConcisenessIssue, error) {
	src, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filepath, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	c.issues = []ConcisenessIssue{}

	// Run various checks
	c.checkVerboseConditionals(fset, node, filepath)
	c.checkUnnecessaryElse(fset, node, filepath)
	c.checkRedundantVariables(fset, node, filepath)
	c.checkVerboseLoops(fset, node, filepath)
	c.checkErrorHandling(fset, node, filepath)

	return c.issues, nil
}

// checkVerboseConditionals finds verbose conditional patterns
func (c *ConcisenessAnalyzer) checkVerboseConditionals(fset *token.FileSet, node *ast.File, filepath string) {
	ast.Inspect(node, func(n ast.Node) bool {
		if ifStmt, ok := n.(*ast.IfStmt); ok {
			// Check for if x == true
			if binExpr, ok := ifStmt.Cond.(*ast.BinaryExpr); ok {
				if binExpr.Op == token.EQL {
					if ident, ok := binExpr.Y.(*ast.Ident); ok && ident.Name == "true" {
						pos := fset.Position(ifStmt.Pos())
						c.issues = append(c.issues, ConcisenessIssue{
							Type:       "verbose_conditional",
							File:       filepath,
							Line:       pos.Line,
							Message:    "Comparing with true is unnecessary",
							Original:   "if x == true",
							Improved:   "if x",
							Suggestion: "Remove explicit comparison with true",
						})
					}

					if ident, ok := binExpr.Y.(*ast.Ident); ok && ident.Name == "false" {
						pos := fset.Position(ifStmt.Pos())
						c.issues = append(c.issues, ConcisenessIssue{
							Type:       "verbose_conditional",
							File:       filepath,
							Line:       pos.Line,
							Message:    "Comparing with false is verbose",
							Original:   "if x == false",
							Improved:   "if !x",
							Suggestion: "Use negation instead of comparing with false",
						})
					}
				}

				// Check for x != nil
				if binExpr.Op == token.NEQ {
					if ident, ok := binExpr.Y.(*ast.Ident); ok && ident.Name == "nil" {
						pos := fset.Position(ifStmt.Pos())
						leftStr := formatNode(fset, binExpr.X)
						c.issues = append(c.issues, ConcisenessIssue{
							Type:       "verbose_nil_check",
							File:       filepath,
							Line:       pos.Line,
							Message:    "Use 'is not' for nil comparisons",
							Original:   fmt.Sprintf("if %s != nil", leftStr),
							Improved:   fmt.Sprintf("if %s != nil", leftStr), // Go uses != for nil
							Suggestion: "This is correct for Go",
						})
					}
				}
			}
		}
		return true
	})
}

// checkUnnecessaryElse finds unnecessary else after return/continue/break
func (c *ConcisenessAnalyzer) checkUnnecessaryElse(fset *token.FileSet, node *ast.File, filepath string) {
	ast.Inspect(node, func(n ast.Node) bool {
		if ifStmt, ok := n.(*ast.IfStmt); ok {
			if ifStmt.Else != nil {
				// Check if the if-block ends with return/continue/break
				if len(ifStmt.Body.List) > 0 {
					lastStmt := ifStmt.Body.List[len(ifStmt.Body.List)-1]

					isTerminal := false
					switch lastStmt.(type) {
					case *ast.ReturnStmt, *ast.BranchStmt:
						isTerminal = true
					}

					if isTerminal {
						pos := fset.Position(ifStmt.Else.Pos())
						c.issues = append(c.issues, ConcisenessIssue{
							Type:       "unnecessary_else",
							File:       filepath,
							Line:       pos.Line,
							Message:    "Else is unnecessary after return/break/continue",
							Original:   "if condition {\n    return x\n} else {\n    // code\n}",
							Improved:   "if condition {\n    return x\n}\n// code",
							Suggestion: "Remove else and unindent the code",
						})
					}
				}
			}
		}
		return true
	})
}

// checkRedundantVariables finds redundant variable assignments
func (c *ConcisenessAnalyzer) checkRedundantVariables(fset *token.FileSet, node *ast.File, filepath string) {
	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			if fn.Body != nil && len(fn.Body.List) >= 2 {
				// Check last two statements
				stmts := fn.Body.List
				if len(stmts) >= 2 {
					secondLast := stmts[len(stmts)-2]
					last := stmts[len(stmts)-1]

					// Pattern: x := expr; return x
					if assign, ok := secondLast.(*ast.AssignStmt); ok {
						if ret, ok := last.(*ast.ReturnStmt); ok {
							if len(assign.Lhs) == 1 && len(ret.Results) == 1 {
								if lhsIdent, ok := assign.Lhs[0].(*ast.Ident); ok {
									if retIdent, ok := ret.Results[0].(*ast.Ident); ok {
										if lhsIdent.Name == retIdent.Name {
											pos := fset.Position(assign.Pos())
											c.issues = append(c.issues, ConcisenessIssue{
												Type:       "redundant_variable",
												File:       filepath,
												Line:       pos.Line,
												Message:    "Redundant variable assignment before return",
												Original:   fmt.Sprintf("%s := ...\nreturn %s", lhsIdent.Name, lhsIdent.Name),
												Improved:   "return ...",
												Suggestion: "Return the expression directly",
											})
										}
									}
								}
							}
						}
					}
				}
			}
		}
		return true
	})
}

// checkVerboseLoops finds loops that could be simplified
func (c *ConcisenessAnalyzer) checkVerboseLoops(fset *token.FileSet, node *ast.File, filepath string) {
	ast.Inspect(node, func(n ast.Node) bool {
		if forStmt, ok := n.(*ast.RangeStmt); ok {
			// Check for append pattern
			if len(forStmt.Body.List) == 1 {
				if exprStmt, ok := forStmt.Body.List[0].(*ast.ExprStmt); ok {
					if call, ok := exprStmt.X.(*ast.CallExpr); ok {
						if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
							if sel.Sel.Name == "append" {
								pos := fset.Position(forStmt.Pos())
								c.issues = append(c.issues, ConcisenessIssue{
									Type:       "verbose_loop",
									File:       filepath,
									Line:       pos.Line,
									Message:    "Loop with append can be a slice operation",
									Original:   "for _, x := range items {\n    result = append(result, x)\n}",
									Improved:   "result = append(result, items...)",
									Suggestion: "Use variadic append for simple copies",
								})
							}
						}
					}
				}
			}
		}
		return true
	})
}

// checkErrorHandling finds verbose error handling patterns
func (c *ConcisenessAnalyzer) checkErrorHandling(fset *token.FileSet, node *ast.File, filepath string) {
	ast.Inspect(node, func(n ast.Node) bool {
		if ifStmt, ok := n.(*ast.IfStmt); ok {
			// Pattern: if err != nil { return err }
			if binExpr, ok := ifStmt.Cond.(*ast.BinaryExpr); ok {
				if binExpr.Op == token.NEQ {
					if errIdent, ok := binExpr.X.(*ast.Ident); ok {
						if nilIdent, ok := binExpr.Y.(*ast.Ident); ok {
							if errIdent.Name == "err" && nilIdent.Name == "nil" {
								if len(ifStmt.Body.List) == 1 {
									if ret, ok := ifStmt.Body.List[0].(*ast.ReturnStmt); ok {
										if len(ret.Results) == 1 {
											if retErr, ok := ret.Results[0].(*ast.Ident); ok {
												if retErr.Name == "err" && ifStmt.Else == nil {
													// This is actually the idiomatic Go pattern
													// Don't flag it as an issue
													return true
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
		return true
	})
}

// RefactorFile automatically refactors a file for conciseness
func RefactorFile(filepath string) error {
	src, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filepath, src, parser.ParseComments)
	if err != nil {
		return err
	}

	// Apply transformations
	refactorer := &conciseRefactorer{}
	ast.Walk(refactorer, node)

	// Format and write back
	var buf strings.Builder
	if err := format.Node(&buf, fset, node); err != nil {
		return err
	}

	return os.WriteFile(filepath, []byte(buf.String()), 0644)
}

// conciseRefactorer applies conciseness transformations
type conciseRefactorer struct{}

func (r *conciseRefactorer) Visit(node ast.Node) ast.Visitor {
	// Apply various refactoring patterns
	switch n := node.(type) {
	case *ast.IfStmt:
		r.simplifyIf(n)
	case *ast.RangeStmt:
		r.simplifyRange(n)
	}
	return r
}

func (r *conciseRefactorer) simplifyIf(stmt *ast.IfStmt) {
	// Simplify boolean comparisons
	if binExpr, ok := stmt.Cond.(*ast.BinaryExpr); ok {
		if binExpr.Op == token.EQL {
			if ident, ok := binExpr.Y.(*ast.Ident); ok {
				if ident.Name == "true" {
					// Replace x == true with x
					stmt.Cond = binExpr.X
				}
			}
		}
	}
}

func (r *conciseRefactorer) simplifyRange(stmt *ast.RangeStmt) {
	// Simplify range statements where index is not used
	if ident, ok := stmt.Key.(*ast.Ident); ok {
		if ident.Name == "_" && stmt.Value == nil {
			// for _ := range x can be simplified in some cases
			// But this depends on the body
		}
	}
}

// Helper function to format AST node as string
func formatNode(fset *token.FileSet, node ast.Node) string {
	var buf strings.Builder
	format.Node(&buf, fset, node)
	return buf.String()
}
