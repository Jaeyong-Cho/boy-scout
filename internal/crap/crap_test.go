package crap

import (
	"math"
	"os"
	"slices"
	"testing"
)

func TestCrapScore_MatchesFormula(t *testing.T) {
	tests := []struct {
		comp     int
		cov      float64
		expected float64
	}{
		{2, 1.0, 2.0},     // 2² × (1-1)³ + 2 = 4 × 0 + 2 = 2
		{2, 0.0, 6.0},     // 2² × (1-0)³ + 2 = 4 × 1 + 2 = 6
		{4, 0.5, 6.0},     // 4² × (1-0.5)³ + 4 = 16 × 0.125 + 4 = 2 + 4 = 6
		{1, 1.0, 1.0},     // 1² × (1-1)³ + 1 = 1 × 0 + 1 = 1
	}

	for _, tt := range tests {
		score := crapScore(tt.comp, tt.cov)
		if math.Abs(score-tt.expected) > 0.0001 {
			t.Errorf("crapScore(%d, %f) = %f, want %f", tt.comp, tt.cov, score, tt.expected)
		}
	}
}

func TestEvaluate_ScoreExactlyAtThresholdIsCompliant(t *testing.T) {
	score, violated := evaluate(2, 1.0, 2.0) // Score is exactly 2.0
	if violated {
		t.Errorf("score exactly at threshold should not violate")
	}
	if math.Abs(score-2.0) > 0.0001 {
		t.Errorf("expected score 2.0, got %f", score)
	}
}

func TestEvaluate_ScoreOverThresholdIsViolation(t *testing.T) {
	score, violated := evaluate(4, 0.0, 6.0) // Score is 16 + 4 = 20
	if !violated {
		t.Errorf("score over threshold should violate")
	}
	if score <= 6.0 {
		t.Errorf("expected score > 6.0, got %f", score)
	}
}

// chdirTemp creates a temp dir, chdirs into it, and restores the original cwd on cleanup.
func chdirTemp(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldCwd) })
}

// writeFile writes content to path in the current directory.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

// writeFixtureModule chdirs into a fresh temp dir and writes a go.mod for it.
func writeFixtureModule(t *testing.T) {
	t.Helper()
	chdirTemp(t)
	writeFile(t, "go.mod", "module fixture\n\ngo 1.24\n")
}

func TestCheck_MissingGoModReturnsError(t *testing.T) {
	// Create a temp directory with no go.mod anywhere above
	chdirTemp(t)

	_, err := Check([]string{"."}, 6.0, Options{})
	if err == nil {
		t.Errorf("expected error when go.mod is missing")
	}
}

func TestCheck_BuildFailureReturnsError(t *testing.T) {
	writeFixtureModule(t)

	// Write a .go file with invalid syntax - missing closing brace
	writeFile(t, "main.go", `package main
func F() int {
	return 1
`)

	// Write a test file (so go test actually tries to compile)
	writeFile(t, "main_test.go", `package main
import "testing"
func TestF(t *testing.T) {
	_ = F()
}
`)

	_, err := Check([]string{"."}, 6.0, Options{})
	if err == nil {
		t.Errorf("expected error when go test fails to build")
	}
}

func TestCheck_TestFailureStillScoresWithPartialCoverage(t *testing.T) {
	writeFixtureModule(t)

	// Write a simple package
	writeFile(t, "main.go", `package main
func Add(a, b int) int {
	return a + b
}
`)

	// Write a test file that exercises the function but fails
	writeFile(t, "main_test.go", `package main
import "testing"

func TestAdd(t *testing.T) {
	result := Add(1, 2)
	_ = result
	t.Fatal("intentional failure")
}
`)

	// Check should succeed (with the partial coverage)
	report, err := Check([]string{"."}, 6.0, Options{})
	if err != nil {
		t.Errorf("expected no error even with test failure: %v", err)
	}

	// Should have at least one function (the Add function)
	// It's covered since the test exercises it
	if len(report.Violations) != 0 && report.Violations[0].Func != "Add" {
		t.Errorf("expected to find Add function in report")
	}
}

func TestCheck_ReportsViolationForComplexUntestedFunction(t *testing.T) {
	writeFixtureModule(t)

	// Write a function with multiple branches and no test
	writeFile(t, "main.go", `package main
func ComplexFunc(x int) string {
	if x > 10 {
		if x > 20 {
			return "big"
		}
		return "medium"
	}
	if x < 0 {
		return "negative"
	}
	return "small"
}
`)

	// Check with a low threshold
	report, err := Check([]string{"."}, 1.0, Options{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should have at least one violation
	if len(report.Violations) == 0 {
		t.Errorf("expected at least one violation for untested complex function")
	}

	v := mustFindViolation(t, report.Violations, "ComplexFunc")
	if v.File != "main.go" {
		t.Errorf("expected file main.go, got %s", v.File)
	}
	if v.Complexity <= 0 {
		t.Errorf("expected positive complexity, got %d", v.Complexity)
	}
	if v.Coverage != 0.0 {
		t.Errorf("expected 0 coverage for untested function, got %f", v.Coverage)
	}
	if v.Score <= 0.0 {
		t.Errorf("expected positive score, got %f", v.Score)
	}
}

// mustFindViolation returns the violation for funcName, failing the test if absent.
func mustFindViolation(t *testing.T, violations []Violation, funcName string) Violation {
	t.Helper()
	for _, v := range violations {
		if v.Func == funcName {
			return v
		}
	}
	t.Fatalf("expected %s in violations, got: %v", funcName, violations)
	return Violation{}
}

func TestCheck_ExcludeFileSkipsFileAndReportsWhenDebug(t *testing.T) {
	writeFixtureModule(t)

	// Write main.go with a complex untested function
	writeFile(t, "main.go", `package main
func ComplexFunc() {
	if true { if true { if true { if true { if true { } } } } }
}
`)

	// Write main_test.go to satisfy go test
	writeFile(t, "main_test.go", `package main
import "testing"
func TestDummy(t *testing.T) {}
`)

	// Run with exclude-file debug on
	report, err := Check([]string{"."}, 6.0, Options{
		ExcludeFiles: []string{"main.go"},
		Debug:        true,
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Should have 2 excluded files: main.go (user-provided) + main_test.go (default)
	if len(report.ExcludedFiles) != 2 {
		t.Errorf("expected 2 excluded files when debug=true (main.go + main_test.go default), got %d: %v", len(report.ExcludedFiles), report.ExcludedFiles)
	}
}

func TestCheck_ExcludedItemsHiddenUnlessDebug(t *testing.T) {
	writeFixtureModule(t)

	// Write main.go with a complex untested function
	writeFile(t, "main.go", `package main
func ComplexFunc() {
	if true { if true { if true { if true { if true { } } } } }
}
`)

	// Write main_test.go
	writeFile(t, "main_test.go", `package main
import "testing"
func TestDummy(t *testing.T) {}
`)

	// Run with exclude-file debug off
	report, err := Check([]string{"."}, 6.0, Options{
		ExcludeFiles: []string{"main.go"},
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
	writeFixtureModule(t)

	writeFile(t, "main.go", `package main
func ComplexFunc() {
	if true { if true { if true { if true { if true { } } } } }
}
`)
	writeFile(t, "main_test.go", `package main
import "testing"
func TestDummy(t *testing.T) {}
`)

	report, err := Check([]string{"."}, 6.0, Options{
		ExcludeFuncs: []string{"Complex*"},
		Debug:        true,
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations with exclude pattern, got %d", len(report.Violations))
	}
	if len(report.ExcludedFuncs) != 1 {
		t.Fatalf("expected 1 excluded func, got %d", len(report.ExcludedFuncs))
	}
	if report.ExcludedFuncs[0].Reason != "flag" {
		t.Errorf("expected reason=flag, got %q", report.ExcludedFuncs[0].Reason)
	}
}

func TestCheck_ExcludeFuncByCommentDirective(t *testing.T) {
	writeFixtureModule(t)

	writeFile(t, "main.go", `package main

// gardener:ignore
func ComplexFunc() {
	if true { if true { if true { if true { if true { } } } } }
}
`)
	writeFile(t, "main_test.go", `package main
import "testing"
func TestDummy(t *testing.T) {}
`)

	report, err := Check([]string{"."}, 6.0, Options{Debug: true})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations with comment directive, got %d", len(report.Violations))
	}
	if len(report.ExcludedFuncs) != 1 {
		t.Fatalf("expected 1 excluded func, got %d", len(report.ExcludedFuncs))
	}
	if report.ExcludedFuncs[0].Reason != "comment" {
		t.Errorf("expected reason=comment, got %q", report.ExcludedFuncs[0].Reason)
	}
}

func TestCheck_ExcludeFuncByCommentDirective_NamesThisChecker(t *testing.T) {
	writeFixtureModule(t)

	writeFile(t, "main.go", `package main

// gardener:ignore:crap
func ComplexFunc() {
	if true { if true { if true { if true { if true { } } } } }
}
`)
	writeFile(t, "main_test.go", `package main
import "testing"
func TestDummy(t *testing.T) {}
`)

	report, err := Check([]string{"."}, 6.0, Options{Debug: true})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations with crap-specific comment directive, got %d", len(report.Violations))
	}
	if len(report.ExcludedFuncs) != 1 {
		t.Fatalf("expected 1 excluded func, got %d", len(report.ExcludedFuncs))
	}
	if report.ExcludedFuncs[0].Reason != "comment" {
		t.Errorf("expected reason=comment, got %q", report.ExcludedFuncs[0].Reason)
	}
}

func TestCheck_ExcludeFuncByCommentDirective_NamesOtherCheckerOnly(t *testing.T) {
	writeFixtureModule(t)

	writeFile(t, "main.go", `package main

// gardener:ignore:funclen
func ComplexFunc() {
	if true { if true { if true { if true { if true { } } } } }
}
`)
	writeFile(t, "main_test.go", `package main
import "testing"
func TestDummy(t *testing.T) {}
`)

	report, err := Check([]string{"."}, 6.0, Options{Debug: true})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Should have 1 violation because directive doesn't name crap
	if len(report.Violations) != 1 {
		t.Errorf("expected 1 violation (directive names other checker), got %d", len(report.Violations))
	}

	if len(report.ExcludedFuncs) != 0 {
		t.Errorf("expected 0 excluded funcs (directive names other checker), got %d", len(report.ExcludedFuncs))
	}
}

func TestCheck_TestFilesExcludedFromCrapByDefault(t *testing.T) {
	writeFixtureModule(t)

	// Write main.go with a trivial function
	writeFile(t, "main.go", `package main
func Add(a, b int) int {
	return a + b
}
`)

	// Write main_test.go with an untested, deeply-nested helper function
	writeFile(t, "main_test.go", `package main
import "testing"
func TestAdd(t *testing.T) {
	_ = Add(1, 2)
}
func chdirTemp() {
	if true { if true { if true { if true { if true { } } } } }
}
`)

	// Call Check with low threshold, no ExcludeFiles set
	report, err := Check([]string{"."}, 1.0, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Assert that no violation is reported for the test file helper
	for _, v := range report.Violations {
		if v.Func == "chdirTemp" {
			t.Errorf("expected chdirTemp in main_test.go to be excluded by default, but found violation: %+v", v)
		}
	}
}

func TestCheck_DefaultTestFileExcludeVisibleInDebugOutput(t *testing.T) {
	writeFixtureModule(t)

	writeFile(t, "main.go", `package main
func Add(a, b int) int {
	return a + b
}
`)

	writeFile(t, "main_test.go", `package main
import "testing"
func TestAdd(t *testing.T) {
	_ = Add(1, 2)
}
func chdirTemp() {
	if true { if true { if true { if true { if true { } } } } }
}
`)

	// Call Check with Debug=true to see excluded files
	report, err := Check([]string{"."}, 1.0, Options{Debug: true})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Assert that main_test.go appears in ExcludedFiles
	if len(report.ExcludedFiles) == 0 {
		t.Errorf("expected at least 1 excluded file in debug output, got 0")
	}

	if !slices.Contains(report.ExcludedFiles, "main_test.go") {
		t.Errorf("expected main_test.go in ExcludedFiles, got: %v", report.ExcludedFiles)
	}
}

func TestCheck_DefaultTestFileExcludeCombinesWithUserExcludeFile(t *testing.T) {
	writeFixtureModule(t)

	writeFile(t, "main.go", `package main
func Add(a, b int) int {
	return a + b
}
`)

	writeFile(t, "main_test.go", `package main
import "testing"
func TestAdd(t *testing.T) {
	_ = Add(1, 2)
}
func chdirTemp() {
	if true { if true { if true { if true { if true { } } } } }
}
`)

	writeFile(t, "mocks_test.go", `package main
func mockSetup() {
	if true { if true { if true { if true { if true { } } } } }
}
`)

	// Call Check with both user-provided exclude and the default
	report, err := Check([]string{"."}, 1.0, Options{
		ExcludeFiles: []string{"mocks_test.go"},
		Debug:        true,
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if !slices.Contains(report.ExcludedFiles, "main_test.go") {
		t.Errorf("expected main_test.go in excluded files")
	}
	if !slices.Contains(report.ExcludedFiles, "mocks_test.go") {
		t.Errorf("expected mocks_test.go in excluded files")
	}

	// Assert no violations for functions in either excluded test file
	for _, v := range report.Violations {
		if v.Func == "chdirTemp" || v.Func == "mockSetup" {
			t.Errorf("unexpected violation in test file: %+v", v)
		}
	}
}

func TestCheck_FileNotMatchingTestSuffixStillScored(t *testing.T) {
	writeFixtureModule(t)

	// Write contest.go (contains "test" but wrong suffix) with untested function
	writeFile(t, "contest.go", `package main
func ComplexFunc() {
	if true { if true { if true { if true { if true { } } } } }
}
`)

	writeFile(t, "main_test.go", `package main
import "testing"
func TestDummy(t *testing.T) {}
`)

	// Call Check with low threshold
	report, err := Check([]string{"."}, 1.0, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Assert that ComplexFunc in contest.go IS reported as a violation
	v := mustFindViolation(t, report.Violations, "ComplexFunc")
	if v.File != "contest.go" {
		t.Errorf("expected violation in contest.go, got %s", v.File)
	}
}
