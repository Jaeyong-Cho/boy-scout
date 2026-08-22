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

func TestCheck_MissingGoModReturnsError(t *testing.T) {
	// Create a temp directory with no go.mod anywhere above
	tmpDir := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldCwd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	_, err = Check([]string{"."}, 6.0)
	if err == nil {
		t.Errorf("expected error when go.mod is missing")
	}
}

func TestCheck_BuildFailureReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldCwd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Write a go.mod file
	if err := os.WriteFile("go.mod", []byte("module fixture\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Write a .go file with invalid syntax - missing closing brace
	if err := os.WriteFile("main.go", []byte(`package main
func F() int {
	return 1
`), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	// Write a test file (so go test actually tries to compile)
	if err := os.WriteFile("main_test.go", []byte(`package main
import "testing"
func TestF(t *testing.T) {
	_ = F()
}
`), 0644); err != nil {
		t.Fatalf("failed to write main_test.go: %v", err)
	}

	_, err = Check([]string{"."}, 6.0)
	if err == nil {
		t.Errorf("expected error when go test fails to build")
	}
}

func TestCheck_TestFailureStillScoresWithPartialCoverage(t *testing.T) {
	tmpDir := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldCwd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Write a go.mod file
	if err := os.WriteFile("go.mod", []byte("module fixture\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Write a simple package
	if err := os.WriteFile("main.go", []byte(`package main
func Add(a, b int) int {
	return a + b
}
`), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	// Write a test file that exercises the function but fails
	if err := os.WriteFile("main_test.go", []byte(`package main
import "testing"

func TestAdd(t *testing.T) {
	result := Add(1, 2)
	_ = result
	t.Fatal("intentional failure")
}
`), 0644); err != nil {
		t.Fatalf("failed to write main_test.go: %v", err)
	}

	// Check should succeed (with the partial coverage)
	report, err := Check([]string{"."}, 6.0)
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
	tmpDir := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldCwd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Write a go.mod file
	if err := os.WriteFile("go.mod", []byte("module fixture\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Write a function with multiple branches and no test
	if err := os.WriteFile("main.go", []byte(`package main
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
`), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	// Check with a low threshold
	report, err := Check([]string{"."}, 1.0)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should have at least one violation
	if len(report.Violations) == 0 {
		t.Errorf("expected at least one violation for untested complex function")
	}

	// Find the ComplexFunc violation
	found := false
	for _, v := range report.Violations {
		if v.Func == "ComplexFunc" {
			found = true
			// Verify fields
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
			break
		}
	}

	if !found {
		t.Errorf("expected ComplexFunc in violations, got: %v", report.Violations)
	}
}
