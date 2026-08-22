package crap

import (
	"fmt"
	"go/ast"
	"go/token"
)

func assertf(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}

// cyclomaticComplexity calculates the McCabe cyclomatic complexity of a function.
// It counts:
// - Base complexity: 1
// - *ast.IfStmt: +1 each
// - *ast.ForStmt: +1 each
// - *ast.RangeStmt: +1 each
// - *ast.CaseClause with non-nil list: +1 each (default cases do NOT count)
// - *ast.CommClause with non-nil comm: +1 each (for select statements)
// - *ast.BinaryExpr with token.LAND or token.LOR: +1 each
//
// Complexity is computed only at the top level of the function body,
// so closures' branches count toward the enclosing function.
func cyclomaticComplexity(fn *ast.FuncDecl) int {
	complexity := 1

	if fn.Body == nil {
		return complexity
	}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		complexity += nodeComplexity(n)
		return true
	})

	assertf(complexity >= 1, "complexity must be >=1, got %d for func %s", complexity, fn.Name.Name)
	return complexity
}

// logicalOps are the binary operators that add to cyclomatic complexity.
var logicalOps = map[token.Token]bool{token.LAND: true, token.LOR: true}

// nodeComplexity returns the cyclomatic complexity contribution of a single AST node,
// per the rules documented on cyclomaticComplexity.
func nodeComplexity(n ast.Node) int {
	switch stmt := n.(type) {
	case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
		return 1
	case *ast.CaseClause:
		return boolToInt(stmt.List != nil)
	case *ast.CommClause:
		return boolToInt(stmt.Comm != nil)
	case *ast.BinaryExpr:
		return boolToInt(logicalOps[stmt.Op])
	}
	return 0
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
