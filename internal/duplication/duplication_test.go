package duplication

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testDir creates a temporary directory for a test
func testDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "duplication_test_")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// writeFile writes content to a file in dir
func writeFile(t *testing.T, dir, name, content string) string {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", name, err)
	}
	return path
}

// TestCheck_ReportsType1ExactDuplicate verifies exact token-for-token matches
func TestCheck_ReportsType1ExactDuplicate(t *testing.T) {
	dir := testDir(t)

	// Create two files with identical functions
	writeFile(t, dir, "a.go", `
package test

func Duplicate() error {
	x := 1
	y := 2
	z := x + y
	return nil
}
`)

	writeFile(t, dir, "b.go", `
package test

func Duplicate2() error {
	x := 1
	y := 2
	z := x + y
	return nil
}
`)

	opts := Options{ExcludeFuncs: []string{}, ExcludeFiles: []string{}}
	report, err := Check([]string{dir}, 5, opts)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(report.Violations))
	}

	v := report.Violations[0]
	if v.Type != "Type-1" {
		t.Errorf("expected Type-1, got %s", v.Type)
	}
	if v.DupLines != 6 {
		t.Errorf("expected 6 duplicate lines, got %d", v.DupLines)
	}
}

// TestCheck_ReportsType2RenamedDuplicate verifies renamed-identifier matches
func TestCheck_ReportsType2RenamedDuplicate(t *testing.T) {
	dir := testDir(t)

	// Create two functions with same structure but different identifiers
	writeFile(t, dir, "tax.go", `
package billing

func CalculateTax(amount float64) float64 {
	rate := 0.08
	total := amount * rate
	if total < 0 {
		total = 0
	}
	return total
}
`)

	writeFile(t, dir, "fee.go", `
package billing

func CalculateFee(price float64) float64 {
	pct := 0.08
	result := price * pct
	if result < 0 {
		result = 0
	}
	return result
}
`)

	opts := Options{ExcludeFuncs: []string{}, ExcludeFiles: []string{}}
	report, err := Check([]string{dir}, 5, opts)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(report.Violations))
	}

	v := report.Violations[0]
	if v.Type != "Type-2" {
		t.Errorf("expected Type-2, got %s", v.Type)
	}
	// Stable ordering: fee.go < tax.go, so FuncA should be CalculateFee
	if v.FuncA != "CalculateFee" || v.FuncB != "CalculateTax" {
		t.Errorf("unexpected function names: %s vs %s", v.FuncA, v.FuncB)
	}
}

// TestCheck_NoViolationForDissimilarFunctions verifies no false positives
func TestCheck_NoViolationForDissimilarFunctions(t *testing.T) {
	dir := testDir(t)

	writeFile(t, dir, "a.go", `
package test

func DifferentA() int {
	a := 1
	b := 2
	c := 3
	return a + b + c
}

func DifferentB() string {
	x := "hello"
	y := "world"
	z := x + y
	return z
}
`)

	opts := Options{ExcludeFuncs: []string{}, ExcludeFiles: []string{}}
	report, err := Check([]string{dir}, 5, opts)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Fatalf("expected no violations, got %d", len(report.Violations))
	}
}

// TestCheck_NoViolationBelowMinLines verifies minLines filtering
func TestCheck_NoViolationBelowMinLines(t *testing.T) {
	dir := testDir(t)

	// Two identical but short functions
	writeFile(t, dir, "a.go", `
package test

func Short() {
	x := 1
}
`)

	writeFile(t, dir, "b.go", `
package test

func Short2() {
	x := 1
}
`)

	opts := Options{ExcludeFuncs: []string{}, ExcludeFiles: []string{}}
	report, err := Check([]string{dir}, 5, opts) // minLines=5, but functions are only 2 lines
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Fatalf("expected no violations (below minLines), got %d", len(report.Violations))
	}
}

// TestCheck_SingleFunctionProducesEmptyReport verifies edge case of single function
func TestCheck_SingleFunctionProducesEmptyReport(t *testing.T) {
	dir := testDir(t)

	writeFile(t, dir, "a.go", `
package test

func OnlyOne() int {
	x := 1
	y := 2
	z := 3
	return x + y + z
}
`)

	opts := Options{ExcludeFuncs: []string{}, ExcludeFiles: []string{}}
	report, err := Check([]string{dir}, 5, opts)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Fatalf("expected no violations, got %d", len(report.Violations))
	}
}

// TestCheck_SkipsTestFiles verifies _test.go exclusion
func TestCheck_SkipsTestFiles(t *testing.T) {
	dir := testDir(t)

	// One function in a regular file
	writeFile(t, dir, "impl.go", `
package test

func CalculateTax(amount float64) float64 {
	rate := 0.08
	total := amount * rate
	if total < 0 {
		total = 0
	}
	return total
}
`)

	// Identical function in a _test.go file (should be skipped)
	writeFile(t, dir, "impl_test.go", `
package test

func CalculateTax(amount float64) float64 {
	rate := 0.08
	total := amount * rate
	if total < 0 {
		total = 0
	}
	return total
}
`)

	opts := Options{ExcludeFuncs: []string{}, ExcludeFiles: []string{}}
	report, err := Check([]string{dir}, 5, opts)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Fatalf("expected no violations (test files excluded), got %d", len(report.Violations))
	}
}

// TestCheck_ExcludeFuncByCommentDirective verifies boy-scout:ignore:duplication
func TestCheck_ExcludeFuncByCommentDirective(t *testing.T) {
	dir := testDir(t)

	writeFile(t, dir, "tax.go", `
package billing

func CalculateTax(amount float64) float64 {
	rate := 0.08
	total := amount * rate
	if total < 0 {
		total = 0
	}
	return total
}
`)

	// This one has the ignore directive
	writeFile(t, dir, "fee.go", `
package billing

// boy-scout:ignore:duplication
func CalculateFee(price float64) float64 {
	pct := 0.08
	result := price * pct
	if result < 0 {
		result = 0
	}
	return result
}
`)

	opts := Options{ExcludeFuncs: []string{}, ExcludeFiles: []string{}, Debug: true}
	report, err := Check([]string{dir}, 5, opts)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Fatalf("expected no violations (function excluded by comment), got %d", len(report.Violations))
	}

	// Check that Debug mode captured the excluded function
	if len(report.ExcludedFuncs) != 1 {
		t.Fatalf("expected 1 excluded function in Debug mode, got %d", len(report.ExcludedFuncs))
	}
	if report.ExcludedFuncs[0].Func != "CalculateFee" {
		t.Errorf("expected CalculateFee to be excluded, got %s", report.ExcludedFuncs[0].Func)
	}
}

// TestCheck_SkipsUnparseableFileAndContinues verifies error handling
func TestCheck_SkipsUnparseableFileAndContinues(t *testing.T) {
	dir := testDir(t)

	// Valid Go file
	writeFile(t, dir, "valid.go", `
package test

func Duplicate() error {
	x := 1
	y := 2
	z := x + y
	return nil
}
`)

	// Duplicate of the first function to have a pair
	writeFile(t, dir, "valid2.go", `
package test

func Duplicate2() error {
	x := 1
	y := 2
	z := x + y
	return nil
}
`)

	// Invalid Go file (syntax error)
	writeFile(t, dir, "invalid.go", `
package test

func Broken(
`)

	opts := Options{ExcludeFuncs: []string{}, ExcludeFiles: []string{}}
	report, err := Check([]string{dir}, 5, opts)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Should have one violation from the valid functions
	if len(report.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(report.Violations))
	}

	// Should have skipped the invalid file
	if len(report.Skipped) != 1 {
		t.Fatalf("expected 1 skipped file, got %d", len(report.Skipped))
	}
	if !strings.Contains(report.Skipped[0].File, "invalid.go") {
		t.Errorf("expected invalid.go in skipped, got %s", report.Skipped[0].File)
	}
}

// TestCheck_MinLinesAssertion tests the precondition that minLines > 0
func TestCheck_MinLinesAssertion(t *testing.T) {
	dir := testDir(t)
	writeFile(t, dir, "a.go", "package test\nfunc A() {}")

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for minLines <= 0, but no panic occurred")
		}
	}()

	opts := Options{}
	Check([]string{dir}, 0, opts)
}

// TestCheck_PairComparisonAssertion ensures i != j in pair loop
func TestCheck_PairComparisonAssertion(t *testing.T) {
	// This test verifies the assertion is there, but can't directly trigger it
	// since the loop structure guarantees i < j. The assertion is a defensive
	// check for future refactors.
	dir := testDir(t)

	writeFile(t, dir, "a.go", `
package test

func Duplicate() error {
	x := 1
	y := 2
	z := x + y
	return nil
}
`)

	writeFile(t, dir, "b.go", `
package test

func Duplicate2() error {
	x := 1
	y := 2
	z := x + y
	return nil
}
`)

	opts := Options{ExcludeFuncs: []string{}, ExcludeFiles: []string{}}
	report, err := Check([]string{dir}, 5, opts)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Just verify the check completes without panic
	if len(report.Violations) != 1 {
		t.Errorf("expected 1 violation, got %d", len(report.Violations))
	}
}
