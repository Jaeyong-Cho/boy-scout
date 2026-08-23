package gofunclen_test

import (
	"os"
	"testing"

	"boy-scout/internal/gofunclen"
)

func TestCheck_ExcludeFileSkipsFileAndReportsWhenDebug(t *testing.T) {
	// Create foo.go with a violation and foo_test.go with a violation
	tmpDir := t.TempDir()

	writeFile(t, tmpDir+"/foo.go", funcSrc("Foo", 103))
	writeFile(t, tmpDir+"/foo_test.go", funcSrc("TestFoo", 103))

	// With debug on, excluded files should appear
	report, err := gofunclen.Check([]string{tmpDir}, 100, gofunclen.Options{
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

	writeFile(t, tmpDir+"/foo.go", funcSrc("Foo", 103))
	writeFile(t, tmpDir+"/foo_test.go", funcSrc("TestFoo", 103))

	// With debug off, excluded files should NOT appear
	report, err := gofunclen.Check([]string{tmpDir}, 100, gofunclen.Options{
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
	// Create TestHelper function (short) and TestRealFunc (long)
	tmpDir := t.TempDir()

	// TestHelper is short (no violation)
	src := `package main

func TestHelper() {
	x := 1
}

func TestRealFunc() {
`
	// Make TestRealFunc 105 lines
	for i := 0; i < 103; i++ {
		src += "\tx := 1\n"
	}
	src += "}\n"

	if err := os.WriteFile(tmpDir+"/test.go", []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Exclude Test* pattern - both TestHelper and TestRealFunc should match
	report, err := gofunclen.Check([]string{tmpDir}, 100, gofunclen.Options{
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

	writeFile(t, tmpDir+"/test.go", funcSrc("Foo", 103, "// boy-scout:ignore"))

	report, err := gofunclen.Check([]string{tmpDir}, 100, gofunclen.Options{
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

	if report.ExcludedFuncs[0].Reason != "comment" {
		t.Errorf("expected reason=comment, got %q", report.ExcludedFuncs[0].Reason)
	}
	if report.ExcludedFuncs[0].Func != "Foo" {
		t.Errorf("expected Func=Foo, got %q", report.ExcludedFuncs[0].Func)
	}
}

func TestCheck_CommentDirectiveTypoIsNotExcluded(t *testing.T) {
	tmpDir := t.TempDir()

	// Typo: missing colon
	writeFile(t, tmpDir+"/test.go", funcSrc("Foo", 103, "// boy-scoutignore"))

	report, err := gofunclen.Check([]string{tmpDir}, 100, gofunclen.Options{
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

	writeFile(t, tmpDir+"/test.go", funcSrc("Foo", 103, "// boy-scout:ignore:gofunclen"))

	report, err := gofunclen.Check([]string{tmpDir}, 100, gofunclen.Options{
		Debug: true,
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations with gofunclen-specific comment directive, got %d", len(report.Violations))
	}

	if len(report.ExcludedFuncs) != 1 {
		t.Errorf("expected 1 excluded func, got %d", len(report.ExcludedFuncs))
	}

	if report.ExcludedFuncs[0].Reason != "comment" {
		t.Errorf("expected reason=comment, got %q", report.ExcludedFuncs[0].Reason)
	}
	if report.ExcludedFuncs[0].Func != "Foo" {
		t.Errorf("expected Func=Foo, got %q", report.ExcludedFuncs[0].Func)
	}
}

func TestCheck_ExcludeFuncByCommentDirective_NamesOtherCheckerOnly(t *testing.T) {
	tmpDir := t.TempDir()

	// Comment directive names only "crap", not "gofunclen"
	writeFile(t, tmpDir+"/test.go", funcSrc("Foo", 103, "// boy-scout:ignore:crap"))

	report, err := gofunclen.Check([]string{tmpDir}, 100, gofunclen.Options{
		Debug: true,
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Should have 1 violation because directive doesn't name gofunclen
	if len(report.Violations) != 1 {
		t.Errorf("expected 1 violation (directive names other checker), got %d", len(report.Violations))
	}

	if len(report.ExcludedFuncs) != 0 {
		t.Errorf("expected 0 excluded funcs (directive names other checker), got %d", len(report.ExcludedFuncs))
	}
}

func TestCheck_StillScoresTestFilesUnaffectedByCrapDefault(t *testing.T) {
	// Regression test: gofunclen should still score functions in _test.go files
	// even though crap.Check now excludes them by default
	tmpDir := t.TempDir()

	// Create a _test.go file with a function exceeding the limit (55 lines > 50 default)
	writeFile(t, tmpDir+"/helper_test.go", funcSrc("TestHelper", 53))

	report, err := gofunclen.Check([]string{tmpDir}, 50, gofunclen.Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// gofunclen should still report the violation in the test file
	if len(report.Violations) != 1 {
		t.Errorf("expected 1 violation for long function in _test.go, got %d", len(report.Violations))
	}

	if len(report.Violations) > 0 && report.Violations[0].Func != "TestHelper" {
		t.Errorf("expected violation for TestHelper, got %q", report.Violations[0].Func)
	}
}
