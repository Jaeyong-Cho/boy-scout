package crap

import (
	"testing"
)

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

