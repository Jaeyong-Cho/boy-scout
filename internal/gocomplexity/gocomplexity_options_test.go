package gocomplexity_test

import (
	"os"
	"strings"
	"testing"

	"boy-scout/internal/gocomplexity"
)

// complexFuncSrc generates Go source for "package main" with a single function
// of specified complexity, with optional preamble lines (e.g. comment directives)
// placed before the func declaration.
func complexFuncSrc(funcName string, complexity int, preamble ...string) string {
	lines := append([]string{"package main", ""}, preamble...)
	lines = append(lines, "func "+funcName+"() {")
	// Each 'if' adds 1 to complexity (base is 1)
	for i := 0; i < complexity-1; i++ {
		lines = append(lines, "\tif true {")
	}
	lines = append(lines, "\t\t_ = 1")
	for i := 0; i < complexity-1; i++ {
		lines = append(lines, "\t}")
	}
	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

func TestCheck_ExcludeFileSkipsFileAndReportsWhenDebug(t *testing.T) {
	// Create foo.go with a violation and foo_test.go with a violation
	tmpDir := t.TempDir()

	writeFile(t, tmpDir+"/foo.go", simpleFuncSrc("Foo", 12))
	writeFile(t, tmpDir+"/foo_test.go", simpleFuncSrc("TestFoo", 12))

	// With debug on, excluded files should appear
	report, err := gocomplexity.Check([]string{tmpDir}, 10, gocomplexity.Options{
		ExcludeFiles: []string{"*_test.go"},
		Debug:        true,
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(report.Violations))
	}
	if report.Violations[0].Func != "Foo" {
		t.Errorf("expected violation for Foo, got %q", report.Violations[0].Func)
	}

	if len(report.ExcludedFiles) != 1 {
		t.Errorf("expected 1 excluded file when debug=true, got %d", len(report.ExcludedFiles))
	}
}

func TestCheck_ExcludedItemsHiddenUnlessDebug(t *testing.T) {
	// Create foo.go with a violation
	tmpDir := t.TempDir()

	writeFile(t, tmpDir+"/foo.go", simpleFuncSrc("Foo", 12))
	writeFile(t, tmpDir+"/foo_test.go", simpleFuncSrc("TestFoo", 12))

	// With debug off, excluded files should NOT appear
	report, err := gocomplexity.Check([]string{tmpDir}, 10, gocomplexity.Options{
		ExcludeFiles: []string{"*_test.go"},
		Debug:        false,
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.ExcludedFiles) != 0 {
		t.Errorf("expected 0 excluded files when debug=false, got %d", len(report.ExcludedFiles))
	}
	if len(report.ExcludedFuncs) != 0 {
		t.Errorf("expected 0 excluded funcs when debug=false, got %d", len(report.ExcludedFuncs))
	}
}

func TestCheck_ExcludeFuncByNamePattern(t *testing.T) {
	// Create TestHelper function (simple) and TestRealFunc (complex)
	tmpDir := t.TempDir()

	// TestHelper is simple (no violation)
	src := `package main

func TestHelper() {
	x := 1
}

func TestRealFunc() {
`
	// Make TestRealFunc complex (12)
	for i := 0; i < 11; i++ {
		src += "\tif true {\n"
	}
	src += "\t\t_ = 1\n"
	for i := 0; i < 11; i++ {
		src += "\t}\n"
	}
	src += "}\n"

	if err := os.WriteFile(tmpDir+"/test.go", []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Exclude Test* pattern - both TestHelper and TestRealFunc should match
	report, err := gocomplexity.Check([]string{tmpDir}, 10, gocomplexity.Options{
		ExcludeFuncs: []string{"Test*"},
		Debug:        true,
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations with exclude pattern, got %d", len(report.Violations))
	}

	// Should have 2 excluded funcs (both Test*)
	if len(report.ExcludedFuncs) != 2 {
		t.Errorf("expected 2 excluded funcs, got %d", len(report.ExcludedFuncs))
	}

	// Check that both have reason="flag"
	for _, exc := range report.ExcludedFuncs {
		if exc.Reason != "flag" {
			t.Errorf("expected reason=flag, got %q", exc.Reason)
		}
	}
}

func TestCheck_ExcludeFuncByCommentDirective(t *testing.T) {
	tmpDir := t.TempDir()

	writeFile(t, tmpDir+"/test.go", complexFuncSrc("Foo", 12, "// boy-scout:ignore"))

	report, err := gocomplexity.Check([]string{tmpDir}, 10, gocomplexity.Options{
		Debug: true,
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations with comment directive, got %d", len(report.Violations))
	}

	if len(report.ExcludedFuncs) != 1 {
		t.Errorf("expected 1 excluded func, got %d", len(report.ExcludedFuncs))
	}

	if len(report.ExcludedFuncs) > 0 {
		if report.ExcludedFuncs[0].Reason != "comment" {
			t.Errorf("expected reason=comment, got %q", report.ExcludedFuncs[0].Reason)
		}
		if report.ExcludedFuncs[0].Func != "Foo" {
			t.Errorf("expected Func=Foo, got %q", report.ExcludedFuncs[0].Func)
		}
	}
}

func TestCheck_CommentDirectiveTypoIsNotExcluded(t *testing.T) {
	tmpDir := t.TempDir()

	// Typo: missing colon
	writeFile(t, tmpDir+"/test.go", complexFuncSrc("Foo", 12, "// boy-scoutignore"))

	report, err := gocomplexity.Check([]string{tmpDir}, 10, gocomplexity.Options{
		Debug: true,
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Should have 1 violation because typo is not recognized
	if len(report.Violations) != 1 {
		t.Errorf("expected 1 violation (typo not recognized), got %d", len(report.Violations))
	}

	if len(report.ExcludedFuncs) != 0 {
		t.Errorf("expected 0 excluded funcs with typo, got %d", len(report.ExcludedFuncs))
	}
}

func TestCheck_ExcludeFuncByCommentDirective_NamesThisChecker(t *testing.T) {
	tmpDir := t.TempDir()

	writeFile(t, tmpDir+"/test.go", complexFuncSrc("Foo", 12, "// boy-scout:ignore:complexity"))

	report, err := gocomplexity.Check([]string{tmpDir}, 10, gocomplexity.Options{
		Debug: true,
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations with complexity-specific comment directive, got %d", len(report.Violations))
	}

	if len(report.ExcludedFuncs) != 1 {
		t.Errorf("expected 1 excluded func, got %d", len(report.ExcludedFuncs))
	}

	if len(report.ExcludedFuncs) > 0 {
		if report.ExcludedFuncs[0].Reason != "comment" {
			t.Errorf("expected reason=comment, got %q", report.ExcludedFuncs[0].Reason)
		}
		if report.ExcludedFuncs[0].Func != "Foo" {
			t.Errorf("expected Func=Foo, got %q", report.ExcludedFuncs[0].Func)
		}
	}
}

func TestCheck_ExcludeFuncByCommentDirective_NamesOtherCheckerOnly(t *testing.T) {
	tmpDir := t.TempDir()

	writeFile(t, tmpDir+"/test.go", complexFuncSrc("Foo", 12, "// boy-scout:ignore:crap"))

	report, err := gocomplexity.Check([]string{tmpDir}, 10, gocomplexity.Options{
		Debug: true,
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Should have 1 violation because directive doesn't name complexity
	if len(report.Violations) != 1 {
		t.Errorf("expected 1 violation (directive names other checker), got %d", len(report.Violations))
	}

	if len(report.ExcludedFuncs) != 0 {
		t.Errorf("expected 0 excluded funcs (directive names other checker), got %d", len(report.ExcludedFuncs))
	}
}
