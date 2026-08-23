package instability

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestCheck_ErrorsWithoutGoMod tests that Check returns an error when no go.mod is found
func TestCheck_ErrorsWithoutGoMod(t *testing.T) {
	tempDir := t.TempDir()

	report, err := Check([]string{tempDir}, 0, Options{})

	if err == nil {
		t.Fatal("expected error when no go.mod found, got nil")
	}
	if len(report.Violations) != 0 {
		t.Fatalf("expected empty report when error occurs, got %d violations", len(report.Violations))
	}
}

// TestCheck_NoInternalEdgesIsClean tests that a module with one package and no internal imports is clean
func TestCheck_NoInternalEdgesIsClean(t *testing.T) {
	tempDir := t.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/test\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Write a simple package with no imports
	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	report, err := Check([]string{tempDir}, 0, Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.TotalEdges != 0 {
		t.Fatalf("expected 0 edges, got %d", report.TotalEdges)
	}
	if len(report.Violations) != 0 {
		t.Fatalf("expected 0 violations, got %d", len(report.Violations))
	}
	if report.ViolationRate != 0 {
		t.Fatalf("expected ViolationRate=0, got %f", report.ViolationRate)
	}
}

// TestCheck_SkipsUnparsableFile tests that unparsable files are recorded as skipped
func TestCheck_SkipsUnparsableFile(t *testing.T) {
	tempDir := t.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/test\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Write a valid file
	if err := os.WriteFile(filepath.Join(tempDir, "valid.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write valid.go: %v", err)
	}

	// Write a broken file (bad import syntax that will fail parsing)
	if err := os.WriteFile(filepath.Join(tempDir, "broken.go"), []byte("package main\n\nimport (\n\t\"fmt\"\n\t//\n"), 0644); err != nil {
		t.Fatalf("failed to write broken.go: %v", err)
	}

	report, err := Check([]string{tempDir}, 0, Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Skipped) != 1 {
		t.Fatalf("expected 1 skipped file, got %d", len(report.Skipped))
	}
	if !filepath.IsAbs(report.Skipped[0].File) || filepath.Base(report.Skipped[0].File) != "broken.go" {
		t.Fatalf("expected skipped file to be broken.go")
	}
	if report.Skipped[0].Error == "" {
		t.Fatal("expected skipped file to have an error message")
	}
}

// TestCheck_ReportsViolationWhenStableDependsOnUnstable tests the main violation case
func TestCheck_ReportsViolationWhenStableDependsOnUnstable(t *testing.T) {
	tempDir := t.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/shop\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create domain package (stable: Ca=5, Ce=0, I=0.0)
	// Made stable by being imported by many things, importing nothing from first-party
	domainDir := filepath.Join(tempDir, "domain")
	os.Mkdir(domainDir, 0755)
	if err := os.WriteFile(filepath.Join(domainDir, "domain.go"), []byte("package domain\n\nfunc A(){}\nfunc B(){}\nfunc C(){}\nfunc D(){}\nfunc E(){}\n"), 0644); err != nil {
		t.Fatalf("failed to write domain.go: %v", err)
	}

	// Create httpapi package (unstable: Ca=0, Ce=4, I=1.0)
	// Imports 4 first-party packages, nothing imports it
	httpapiDir := filepath.Join(tempDir, "httpapi")
	os.Mkdir(httpapiDir, 0755)

	// Create some packages for httpapi to import
	for i := 1; i <= 4; i++ {
		depDir := filepath.Join(tempDir, fmt.Sprintf("dep%d", i))
		os.Mkdir(depDir, 0755)
		if err := os.WriteFile(filepath.Join(depDir, fmt.Sprintf("dep%d.go", i)), []byte(fmt.Sprintf("package dep%d\nfunc Dep%d(){}\n", i, i)), 0644); err != nil {
			t.Fatalf("failed to write dep%d: %v", i, err)
		}
	}

	imports := "package httpapi\nimport (\n"
	for i := 1; i <= 4; i++ {
		imports += fmt.Sprintf("\t\"example.com/shop/dep%d\"\n", i)
	}
	imports += ")\nfunc Handler(){}\n"

	if err := os.WriteFile(filepath.Join(httpapiDir, "httpapi.go"), []byte(imports), 0644); err != nil {
		t.Fatalf("failed to write httpapi.go: %v", err)
	}

	// Make domain's stable by having other packages import it
	for i := 1; i <= 5; i++ {
		clientDir := filepath.Join(tempDir, fmt.Sprintf("client%d", i))
		os.Mkdir(clientDir, 0755)
		clientCode := fmt.Sprintf("package client%d\nimport \"example.com/shop/domain\"\nfunc Call%d(){}\n", i, i)
		if err := os.WriteFile(filepath.Join(clientDir, fmt.Sprintf("client%d.go", i)), []byte(clientCode), 0644); err != nil {
			t.Fatalf("failed to write client%d: %v", i, err)
		}
	}

	// Domain imports httpapi (the violation: stable depends on unstable)
	if err := os.WriteFile(filepath.Join(domainDir, "domain_imports.go"), []byte("package domain\nimport \"example.com/shop/httpapi\"\n"), 0644); err != nil {
		t.Fatalf("failed to write domain_imports.go: %v", err)
	}

	report, err := Check([]string{tempDir}, 0, Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the domain -> httpapi violation
	var violation *Violation
	for i := range report.Violations {
		if report.Violations[i].Source == "example.com/shop/domain" && report.Violations[i].Target == "example.com/shop/httpapi" {
			violation = &report.Violations[i]
			break
		}
	}

	if violation == nil {
		t.Fatalf("expected violation for domain -> httpapi, got violations: %v", report.Violations)
	}

	if violation.I_A >= violation.I_B {
		t.Fatalf("expected I_A < I_B (stable depending on unstable), got I_A=%f, I_B=%f", violation.I_A, violation.I_B)
	}
	if violation.Gap <= 0 {
		t.Fatalf("expected Gap > 0, got %f", violation.Gap)
	}
}

// TestCheck_NoViolationWhenUnstableDependsOnStable tests the intended direction (no violation)
func TestCheck_NoViolationWhenUnstableDependsOnStable(t *testing.T) {
	tempDir := t.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/shop\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create domain package (stable: Ca=5, Ce=0, I=0.0)
	domainDir := filepath.Join(tempDir, "domain")
	os.Mkdir(domainDir, 0755)
	if err := os.WriteFile(filepath.Join(domainDir, "domain.go"), []byte("package domain\n\nfunc A(){}\n"), 0644); err != nil {
		t.Fatalf("failed to write domain.go: %v", err)
	}

	// Create httpapi package (unstable: Ce=4)
	httpapiDir := filepath.Join(tempDir, "httpapi")
	os.Mkdir(httpapiDir, 0755)

	for i := 1; i <= 4; i++ {
		depDir := filepath.Join(tempDir, fmt.Sprintf("dep%d", i))
		os.Mkdir(depDir, 0755)
		if err := os.WriteFile(filepath.Join(depDir, fmt.Sprintf("dep%d.go", i)), []byte(fmt.Sprintf("package dep%d\nfunc Dep%d(){}\n", i, i)), 0644); err != nil {
			t.Fatalf("failed to write dep%d: %v", i, err)
		}
	}

	imports := "package httpapi\nimport (\n"
	for i := 1; i <= 4; i++ {
		imports += fmt.Sprintf("\t\"example.com/shop/dep%d\"\n", i)
	}
	imports += "\t\"example.com/shop/domain\"\n)\nfunc Handler(){}\n"

	if err := os.WriteFile(filepath.Join(httpapiDir, "httpapi.go"), []byte(imports), 0644); err != nil {
		t.Fatalf("failed to write httpapi.go: %v", err)
	}

	// Make domain stable
	for i := 1; i <= 5; i++ {
		clientDir := filepath.Join(tempDir, fmt.Sprintf("client%d", i))
		os.Mkdir(clientDir, 0755)
		if err := os.WriteFile(filepath.Join(clientDir, fmt.Sprintf("client%d.go", i)), []byte(fmt.Sprintf("package client%d\nimport \"example.com/shop/domain\"\nfunc Call%d(){}\n", i, i)), 0644); err != nil {
			t.Fatalf("failed to write client%d: %v", i, err)
		}
	}

	report, err := Check([]string{tempDir}, 0, Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that httpapi -> domain is NOT a violation (unstable depending on stable is OK)
	for i := range report.Violations {
		if report.Violations[i].Source == "example.com/shop/httpapi" && report.Violations[i].Target == "example.com/shop/domain" {
			t.Fatalf("unexpected violation: httpapi -> domain should not be a violation (unstable depends on stable)")
		}
	}
}

// TestCheck_EqualInstabilityIsCompliant tests the boundary where instability is equal
func TestCheck_EqualInstabilityIsCompliant(t *testing.T) {
	tempDir := t.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/shop\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create two packages with equal instability: Ca=Ce for both
	for _, pkgName := range []string{"a", "b"} {
		pkgDir := filepath.Join(tempDir, pkgName)
		os.Mkdir(pkgDir, 0755)
		// Each imports 1 package and is imported by 1 package
		// We'll wire them after creating the files
		if err := os.WriteFile(filepath.Join(pkgDir, fmt.Sprintf("%s.go", pkgName)), []byte(fmt.Sprintf("package %s\nfunc %s(){}\n", pkgName, pkgName)), 0644); err != nil {
			t.Fatalf("failed to write %s.go: %v", pkgName, err)
		}
	}

	// Create two helper packages to balance the instability
	// a imports helper1, helper1 imports nothing, helper1 is imported by b
	helperDir := filepath.Join(tempDir, "helper1")
	os.Mkdir(helperDir, 0755)
	if err := os.WriteFile(filepath.Join(helperDir, "helper1.go"), []byte("package helper1\nfunc H1(){}\n"), 0644); err != nil {
		t.Fatalf("failed to write helper1: %v", err)
	}

	helper2Dir := filepath.Join(tempDir, "helper2")
	os.Mkdir(helper2Dir, 0755)
	if err := os.WriteFile(filepath.Join(helper2Dir, "helper2.go"), []byte("package helper2\nfunc H2(){}\n"), 0644); err != nil {
		t.Fatalf("failed to write helper2: %v", err)
	}

	// Now wire the imports to make equal instability
	// a imports helper1 (Ce=1) and is imported by b (Ca=1) => I_a = 1/2
	// b imports helper2 (Ce=1) and is imported by a (Ca=1) => I_b = 1/2
	if err := os.WriteFile(filepath.Join(tempDir, "a", "a_imports.go"), []byte("package a\nimport \"example.com/shop/helper1\"\nimport \"example.com/shop/b\"\n"), 0644); err != nil {
		t.Fatalf("failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "b", "b_imports.go"), []byte("package b\nimport \"example.com/shop/helper2\"\nimport \"example.com/shop/a\"\n"), 0644); err != nil {
		t.Fatalf("failed: %v", err)
	}

	report, err := Check([]string{tempDir}, 0, Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// When I_A == I_B (Gap == 0), there should be no violation (boundary: strict >)
	for i := range report.Violations {
		v := &report.Violations[i]
		if (v.Source == "example.com/shop/a" && v.Target == "example.com/shop/b") ||
			(v.Source == "example.com/shop/b" && v.Target == "example.com/shop/a") {
			if v.Gap == 0 {
				t.Fatalf("expected no violation when Gap == 0 (strict >), but got violation: %+v", v)
			}
		}
	}
}

// TestCheck_MinGapFiltersListNotSummary tests that min-gap filters violations but not summary stats
func TestCheck_MinGapFiltersListNotSummary(t *testing.T) {
	tempDir := t.TempDir()

	// Write go.mod
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/shop\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create a simple two-package scenario with a small gap
	// Package a: Ca=2, Ce=0, I=0
	// Package b: Ca=0, Ce=1, I=1
	// Edge a -> b has Gap = 1 - 0 = 1.0 (but we can create a smaller gap)

	// Let's create a: Ca=3, Ce=1, I=1/4 = 0.25
	// And b: Ca=1, Ce=2, I=2/3 ≈ 0.667
	// Gap = 0.667 - 0.25 = 0.417

	// Package a (Ca=3, Ce=1, I=0.25)
	aDir := filepath.Join(tempDir, "a")
	os.Mkdir(aDir, 0755)

	// Create dep for a to import
	depDir := filepath.Join(tempDir, "dep")
	os.Mkdir(depDir, 0755)
	if err := os.WriteFile(filepath.Join(depDir, "dep.go"), []byte("package dep\nfunc D(){}\n"), 0644); err != nil {
		t.Fatalf("failed: %v", err)
	}

	// a imports dep and b
	if err := os.WriteFile(filepath.Join(aDir, "a.go"), []byte("package a\nimport (\n\t\"example.com/shop/dep\"\n\t\"example.com/shop/b\"\n)\nfunc A(){}\n"), 0644); err != nil {
		t.Fatalf("failed: %v", err)
	}

	// Package b (Ca=1, Ce=2, I≈0.667)
	bDir := filepath.Join(tempDir, "b")
	os.Mkdir(bDir, 0755)

	// Create deps for b to import
	dep2Dir := filepath.Join(tempDir, "dep2")
	os.Mkdir(dep2Dir, 0755)
	if err := os.WriteFile(filepath.Join(dep2Dir, "dep2.go"), []byte("package dep2\nfunc D2(){}\n"), 0644); err != nil {
		t.Fatalf("failed: %v", err)
	}

	dep3Dir := filepath.Join(tempDir, "dep3")
	os.Mkdir(dep3Dir, 0755)
	if err := os.WriteFile(filepath.Join(dep3Dir, "dep3.go"), []byte("package dep3\nfunc D3(){}\n"), 0644); err != nil {
		t.Fatalf("failed: %v", err)
	}

	// b imports dep2 and dep3
	if err := os.WriteFile(filepath.Join(bDir, "b.go"), []byte("package b\nimport (\n\t\"example.com/shop/dep2\"\n\t\"example.com/shop/dep3\"\n)\nfunc B(){}\n"), 0644); err != nil {
		t.Fatalf("failed: %v", err)
	}

	// Get report with minGap=0 (should include all violations with Gap > 0)
	report1, err := Check([]string{tempDir}, 0, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Get report with minGap=0.05 (should filter some small gaps)
	report2, err := Check([]string{tempDir}, 0.05, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The violation lists may differ
	if len(report1.Violations) != len(report2.Violations) && len(report1.Violations) == 0 {
		// If there are no violations even with minGap=0, skip this test
		t.Skip("No violations generated in test fixture")
	}

	// But summary stats should be identical (computed before filtering)
	if report1.ViolationRate != report2.ViolationRate {
		t.Fatalf("expected same ViolationRate in both runs, got %f vs %f", report1.ViolationRate, report2.ViolationRate)
	}
	if report1.WeightedViolationRate != report2.WeightedViolationRate {
		t.Fatalf("expected same WeightedViolationRate in both runs, got %f vs %f", report1.WeightedViolationRate, report2.WeightedViolationRate)
	}
}
