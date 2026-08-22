package crap

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func parseFunc(t *testing.T, src string) *ast.FuncDecl {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(file.Decls) == 0 {
		t.Fatalf("no declarations in parsed file")
	}
	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("first declaration is not a FuncDecl")
	}
	return fn
}

func TestCyclomaticComplexity_StraightLineIsOne(t *testing.T) {
	src := `package main
func F() {
	x := 1
	_ = x
}`
	fn := parseFunc(t, src)
	c := cyclomaticComplexity(fn)
	if c != 1 {
		t.Errorf("expected 1, got %d", c)
	}
}

func TestCyclomaticComplexity_IfAddsOne(t *testing.T) {
	src := `package main
func F(x int) {
	if x > 0 {
		_ = x
	}
}`
	fn := parseFunc(t, src)
	c := cyclomaticComplexity(fn)
	if c != 2 {
		t.Errorf("expected 2, got %d", c)
	}
}

func TestCyclomaticComplexity_ForAndRangeEachAddOne(t *testing.T) {
	src := `package main
func F() {
	for i := 0; i < 10; i++ {
		_ = i
	}
	for range []int{} {
	}
}`
	fn := parseFunc(t, src)
	c := cyclomaticComplexity(fn)
	if c != 3 {
		t.Errorf("expected 3, got %d", c)
	}
}

func TestCyclomaticComplexity_LogicalAndOrAddsOne(t *testing.T) {
	src := `package main
func F(a, b bool) {
	if a || b {
		_ = a
	}
}`
	fn := parseFunc(t, src)
	c := cyclomaticComplexity(fn)
	if c != 3 {
		t.Errorf("expected 3, got %d", c)
	}
}

func TestCyclomaticComplexity_SwitchCasesCountExcludingDefault(t *testing.T) {
	src := `package main
func F(x int) {
	switch x {
	case 1:
		_ = x
	case 2:
		_ = x
	case 3:
		_ = x
	default:
		_ = x
	}
}`
	fn := parseFunc(t, src)
	c := cyclomaticComplexity(fn)
	if c != 4 {
		t.Errorf("expected 4, got %d", c)
	}
}
