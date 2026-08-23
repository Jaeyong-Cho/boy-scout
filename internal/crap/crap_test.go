package crap

import (
	"math"
	"os"
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

func TestCrapDefaultThreshold_BorderlineCaseNotFlaggedAtNewDefault(t *testing.T) {
	// Complexity 8, coverage 70% => CRAP ≈ 9.7
	// This test documents the new default behavior (threshold 30.0)
	// Before the fix: should fail (9.7 > 6.0, violates at old default)
	// After the fix: should pass (9.7 < 30.0, doesn't violate at new default)
	writeFixtureModule(t)

	// Write a function with complexity 8 (8 independent if branches)
	writeFile(t, "main.go", `package main
func BorderlineFunc(x int) string {
	if x == 1 { return "a" }
	if x == 2 { return "b" }
	if x == 3 { return "c" }
	if x == 4 { return "d" }
	if x == 5 { return "e" }
	if x == 6 { return "f" }
	if x == 7 { return "g" }
	if x == 8 { return "h" }
	return "default"
}
`)

	// Write a test file with ~70% coverage (covers x==1 through x==7, misses some branches)
	writeFile(t, "main_test.go", `package main
import "testing"

func TestBorderlineFunc(t *testing.T) {
	cases := []struct {
		x int
		want string
	}{
		{1, "a"},
		{2, "b"},
		{3, "c"},
		{4, "d"},
		{5, "e"},
		{6, "f"},
		{7, "g"},
	}
	for _, tt := range cases {
		got := BorderlineFunc(tt.x)
		if got != tt.want {
			t.Errorf("BorderlineFunc(%d) = %q, want %q", tt.x, got, tt.want)
		}
	}
}
`)

	// Check with the new default threshold of 30.0
	// This test will FAIL before the fix (old default 6.0 is used by CLI)
	// and PASS after the fix (new default 30.0 is used by CLI)
	report, err := Check([]string{"."}, 30.0, Options{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// With threshold 30.0, BorderlineFunc (CRAP ≈ 9.7) should NOT violate
	for _, v := range report.Violations {
		if v.Func == "BorderlineFunc" {
			t.Errorf("BorderlineFunc should not violate at threshold 30.0 (score ≈ 9.7), but got violation: %v", v)
		}
	}
}

func TestCrapDefaultThreshold_OldDefaultStillOverridable(t *testing.T) {
	// Same borderline function with complexity 8, coverage 70% => CRAP ≈ 9.7
	// When explicitly overridden to 6.0, it should still flag as violation (9.7 > 6.0)
	writeFixtureModule(t)

	// Same function as above
	writeFile(t, "main.go", `package main
func BorderlineFunc(x int) string {
	if x == 1 { return "a" }
	if x == 2 { return "b" }
	if x == 3 { return "c" }
	if x == 4 { return "d" }
	if x == 5 { return "e" }
	if x == 6 { return "f" }
	if x == 7 { return "g" }
	if x == 8 { return "h" }
	return "default"
}
`)

	writeFile(t, "main_test.go", `package main
import "testing"

func TestBorderlineFunc(t *testing.T) {
	cases := []struct {
		x int
		want string
	}{
		{1, "a"},
		{2, "b"},
		{3, "c"},
		{4, "d"},
		{5, "e"},
		{6, "f"},
		{7, "g"},
	}
	for _, tt := range cases {
		got := BorderlineFunc(tt.x)
		if got != tt.want {
			t.Errorf("BorderlineFunc(%d) = %q, want %q", tt.x, got, tt.want)
		}
	}
}
`)

	// Check with explicit old default 6.0
	report, err := Check([]string{"."}, 6.0, Options{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// With old default 6.0, BorderlineFunc (CRAP ≈ 9.7) SHOULD violate
	v := mustFindViolation(t, report.Violations, "BorderlineFunc")
	if v.Score <= 6.0 {
		t.Errorf("BorderlineFunc should violate at threshold 6.0 (score ≈ 9.7), got score %f", v.Score)
	}
}

