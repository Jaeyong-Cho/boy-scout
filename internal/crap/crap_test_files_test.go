package crap

import (
	"slices"
	"testing"
)

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
