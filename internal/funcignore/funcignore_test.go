package funcignore

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// Helper to parse a function with doc comment from source.
func parseFuncWithDoc(t *testing.T, src string) *ast.FuncDecl {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			return fn
		}
	}
	t.Fatal("no function found in source")
	return nil
}

func TestReason_CommentDirectiveNamesOnlyThisChecker(t *testing.T) {
	src := `
package test

// boy-scout:ignore:funclen
func LongFunc() {
	return
}
`
	fn := parseFuncWithDoc(t, src)

	// Checker "funclen" should be excluded.
	excluded, reason := Reason(fn, nil, "funclen")
	if !excluded || reason != "comment" {
		t.Errorf("funclen: got excluded=%v reason=%q, want excluded=true reason=comment", excluded, reason)
	}

	// Checker "crap" should NOT be excluded.
	excluded, reason = Reason(fn, nil, "crap")
	if excluded {
		t.Errorf("crap: got excluded=%v, want excluded=false", excluded)
	}
}

func TestReason_CommaListMatchesEitherChecker(t *testing.T) {
	src := `
package test

// boy-scout:ignore:funclen,crap
func ComplexFunc() {
	return
}
`
	fn := parseFuncWithDoc(t, src)

	excluded, reason := Reason(fn, nil, "funclen")
	if !excluded || reason != "comment" {
		t.Errorf("funclen: got excluded=%v reason=%q, want excluded=true reason=comment", excluded, reason)
	}

	excluded, reason = Reason(fn, nil, "crap")
	if !excluded || reason != "comment" {
		t.Errorf("crap: got excluded=%v reason=%q, want excluded=true reason=comment", excluded, reason)
	}
}

func TestReason_WhitespaceAroundNamesIsTrimmed(t *testing.T) {
	src := `
package test

// boy-scout:ignore: funclen , crap
func Func() {
	return
}
`
	fn := parseFuncWithDoc(t, src)

	excluded, reason := Reason(fn, nil, "funclen")
	if !excluded || reason != "comment" {
		t.Errorf("funclen: got excluded=%v reason=%q, want excluded=true reason=comment", excluded, reason)
	}

	excluded, reason = Reason(fn, nil, "crap")
	if !excluded || reason != "comment" {
		t.Errorf("crap: got excluded=%v reason=%q, want excluded=true reason=comment", excluded, reason)
	}
}

func TestReason_DuplicateCheckerNameInListIsNoOp(t *testing.T) {
	src := `
package test

// boy-scout:ignore:funclen,funclen
func Func() {
	return
}
`
	fn := parseFuncWithDoc(t, src)

	excluded, reason := Reason(fn, nil, "funclen")
	if !excluded || reason != "comment" {
		t.Errorf("got excluded=%v reason=%q, want excluded=true reason=comment", excluded, reason)
	}
}

func TestReason_EmptyListAfterColonExcludesNothing(t *testing.T) {
	src := `
package test

// boy-scout:ignore:
func Func() {
	return
}
`
	fn := parseFuncWithDoc(t, src)

	excluded, _ := Reason(fn, nil, "funclen")
	if excluded {
		t.Errorf("funclen: got excluded=%v, want excluded=false", excluded)
	}

	excluded, _ = Reason(fn, nil, "crap")
	if excluded {
		t.Errorf("crap: got excluded=%v, want excluded=false", excluded)
	}
}

func TestReason_UnknownCheckerNameExcludesNothing(t *testing.T) {
	src := `
package test

// boy-scout:ignore:fooey
func Func() {
	return
}
`
	fn := parseFuncWithDoc(t, src)

	excluded, _ := Reason(fn, nil, "funclen")
	if excluded {
		t.Errorf("funclen: got excluded=%v, want excluded=false", excluded)
	}

	excluded, _ = Reason(fn, nil, "crap")
	if excluded {
		t.Errorf("crap: got excluded=%v, want excluded=false", excluded)
	}
}

func TestReason_FlagPatternStillReturnsFlagReason(t *testing.T) {
	src := `
package test

func longFunctionName() {
	return
}
`
	fn := parseFuncWithDoc(t, src)

	excluded, reason := Reason(fn, []string{"long*"}, "funclen")
	if !excluded || reason != "flag" {
		t.Errorf("got excluded=%v reason=%q, want excluded=true reason=flag", excluded, reason)
	}
}

func TestReason_BareDirectiveExcludesRegardlessOfChecker(t *testing.T) {
	src := `
package test

// boy-scout:ignore
func Func() {
	return
}
`
	fn := parseFuncWithDoc(t, src)

	excluded, reason := Reason(fn, nil, "funclen")
	if !excluded || reason != "comment" {
		t.Errorf("funclen: got excluded=%v reason=%q, want excluded=true reason=comment", excluded, reason)
	}

	excluded, reason = Reason(fn, nil, "crap")
	if !excluded || reason != "comment" {
		t.Errorf("crap: got excluded=%v reason=%q, want excluded=true reason=comment", excluded, reason)
	}
}

func TestReason_OldGardenerMarkerNoLongerWorks(t *testing.T) {
	src := `
package test

// gardener:ignore
func OldMarkerFunc() {
	return
}
`
	fn := parseFuncWithDoc(t, src)

	excluded, _ := Reason(fn, nil, "funclen")
	if excluded {
		t.Errorf("funclen: got excluded=%v, want excluded=false (old marker should not work)", excluded)
	}

	excluded, _ = Reason(fn, nil, "crap")
	if excluded {
		t.Errorf("crap: got excluded=%v, want excluded=false (old marker should not work)", excluded)
	}
}
