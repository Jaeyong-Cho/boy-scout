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
		switch stmt := n.(type) {
		case *ast.IfStmt:
			complexity++
		case *ast.ForStmt:
			complexity++
		case *ast.RangeStmt:
			complexity++
		case *ast.CaseClause:
			if stmt.List != nil {
				complexity++
			}
		case *ast.CommClause:
			if stmt.Comm != nil {
				complexity++
			}
		case *ast.BinaryExpr:
			if stmt.Op == token.LAND || stmt.Op == token.LOR {
				complexity++
			}
		}
		return true
	})

	assertf(complexity >= 1, "complexity must be >=1, got %d for func %s", complexity, fn.Name.Name)
	return complexity
}
