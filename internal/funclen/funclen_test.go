package funclen_test

import (
	"os"
	"strings"
	"testing"

	"go-gardener/internal/funclen"
)

func TestCheck_NoViolationUnderLimit(t *testing.T) {
	// Write a small 5-line function to a temp file
	src := `package main

func Foo() {
	x := 1
}
`
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/test.go"
	if err := os.WriteFile(tmpFile, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	report, err := funclen.Check([]string{tmpFile}, 100, funclen.Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(report.Violations))
	}
}

func TestCheck_ReportsViolationOverLimit(t *testing.T) {
	// Create a function with exactly 105 lines from { to }
	// func LongFunc() { (line 3 counting from 1)
	// x := 1  x 103 (lines 4-106, but we'll have fewer lines of filler)
	// }  (closing brace)
	// We want: endLine - startLine + 1 = 105, so if startLine is 3, endLine should be 107
	// That means: 107 - 3 + 1 = 105
	// So we need: line 3 is {, lines 4-106 are filler (103 lines), line 107 is }
	lines := []string{"package main", "", "func LongFunc() {"}

	// Add 103 lines of filler so that function spans from line 3 ({ ) to line 107 (})
	// which gives 107 - 3 + 1 = 105 lines
	for i := 0; i < 103; i++ {
		lines = append(lines, "\tx := 1")
	}
	lines = append(lines, "}")

	src := strings.Join(lines, "\n")

	// Verify the line count: we should have 3 (boilerplate) + 103 (filler) + 1 (closing) = 107 lines total
	numLines := strings.Count(src, "\n") + 1
	if numLines != 107 {
		t.Fatalf("fixture has %d lines, expected 107; src:\n%s", numLines, src)
	}

	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/long.go"
	if err := os.WriteFile(tmpFile, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	report, err := funclen.Check([]string{tmpFile}, 100, funclen.Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(report.Violations))
	}

	v := report.Violations[0]
	if v.Length != 105 {
		t.Errorf("expected Length=105, got %d", v.Length)
	}
	if v.Limit != 100 {
		t.Errorf("expected Limit=100, got %d", v.Limit)
	}
	if v.Func != "LongFunc" {
		t.Errorf("expected Func=LongFunc, got %q", v.Func)
	}
	if !strings.HasSuffix(v.File, "long.go") {
		t.Errorf("expected File to end with 'long.go', got %q", v.File)
	}
}

func TestCheck_ExactlyAtLimitIsCompliant(t *testing.T) {
	// Create a function with exactly 100 lines (the limit) — should not be a violation
	// func Exactly100() { (line 3)
	// x := 1 x 98 (lines 4-101)
	// } (line 102)
	// Span: 102 - 3 + 1 = 100 lines
	lines := []string{"package main", "", "func Exactly100() {"}

	for i := 0; i < 98; i++ {
		lines = append(lines, "\tx := 1")
	}
	lines = append(lines, "}")

	src := strings.Join(lines, "\n")

	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/exact.go"
	if err := os.WriteFile(tmpFile, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	report, err := funclen.Check([]string{tmpFile}, 100, funclen.Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations at limit, got %d", len(report.Violations))
	}
}

func TestCheck_WalksDirectoryRecursivelySkippingVendorAndDotDirs(t *testing.T) {
	// Build a temp dir tree with violations, vendor/, and .hidden/
	tmpDir := t.TempDir()

	// pkg/a.go - violation (105 lines)
	aLines := []string{"package main", "", "func ViolatingA() {"}
	for i := 0; i < 103; i++ {
		aLines = append(aLines, "\tx := 1")
	}
	aLines = append(aLines, "}")
	aSrc := strings.Join(aLines, "\n")

	if err := os.MkdirAll(tmpDir+"/pkg", 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(tmpDir+"/pkg/a.go", []byte(aSrc), 0644); err != nil {
		t.Fatalf("WriteFile a.go failed: %v", err)
	}

	// pkg/sub/b.go - violation (105 lines)
	bLines := []string{"package main", "", "func ViolatingB() {"}
	for i := 0; i < 103; i++ {
		bLines = append(bLines, "\tx := 1")
	}
	bLines = append(bLines, "}")
	bSrc := strings.Join(bLines, "\n")

	if err := os.MkdirAll(tmpDir+"/pkg/sub", 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(tmpDir+"/pkg/sub/b.go", []byte(bSrc), 0644); err != nil {
		t.Fatalf("WriteFile b.go failed: %v", err)
	}

	// vendor/dep/c.go - would be violation but in vendor/ so skipped
	cLines := []string{"package main", "", "func ViolatingC() {"}
	for i := 0; i < 103; i++ {
		cLines = append(cLines, "\tx := 1")
	}
	cLines = append(cLines, "}")
	cSrc := strings.Join(cLines, "\n")

	if err := os.MkdirAll(tmpDir+"/vendor/dep", 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(tmpDir+"/vendor/dep/c.go", []byte(cSrc), 0644); err != nil {
		t.Fatalf("WriteFile c.go failed: %v", err)
	}

	// .git/d.go - would be violation but in dot-dir so skipped
	dLines := []string{"package main", "", "func ViolatingD() {"}
	for i := 0; i < 103; i++ {
		dLines = append(dLines, "\tx := 1")
	}
	dLines = append(dLines, "}")
	dSrc := strings.Join(dLines, "\n")

	if err := os.MkdirAll(tmpDir+"/.git", 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(tmpDir+"/.git/d.go", []byte(dSrc), 0644); err != nil {
		t.Fatalf("WriteFile d.go failed: %v", err)
	}

	report, err := funclen.Check([]string{tmpDir}, 100, funclen.Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 2 {
		t.Fatalf("expected 2 violations, got %d", len(report.Violations))
	}

	// Check that we got violations from a.go and b.go, not from vendor or .git
	funcs := make(map[string]bool)
	for _, v := range report.Violations {
		funcs[v.Func] = true
	}

	if !funcs["ViolatingA"] {
		t.Errorf("expected violation from ViolatingA")
	}
	if !funcs["ViolatingB"] {
		t.Errorf("expected violation from ViolatingB")
	}
	if funcs["ViolatingC"] {
		t.Errorf("unexpected violation from ViolatingC (should be in vendor/)")
	}
	if funcs["ViolatingD"] {
		t.Errorf("unexpected violation from ViolatingD (should be in .git/)")
	}
}

func TestCheck_EmptyDirectoryProducesEmptyReport(t *testing.T) {
	tmpDir := t.TempDir()

	report, err := funclen.Check([]string{tmpDir}, 100, funclen.Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(report.Violations))
	}
	if len(report.Skipped) != 0 {
		t.Errorf("expected 0 skipped files, got %d", len(report.Skipped))
	}
}

func TestCheck_SkipsUnparseableFileAndContinues(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid file with no violations
	goodSrc := `package main

func Good() {
	x := 1
}
`
	if err := os.WriteFile(tmpDir+"/good.go", []byte(goodSrc), 0644); err != nil {
		t.Fatalf("WriteFile good.go failed: %v", err)
	}

	// Create an invalid file with syntax error
	badSrc := `package x
func broken( {
`
	if err := os.WriteFile(tmpDir+"/bad.go", []byte(badSrc), 0644); err != nil {
		t.Fatalf("WriteFile bad.go failed: %v", err)
	}

	report, err := funclen.Check([]string{tmpDir}, 100, funclen.Options{})
	if err != nil {
		t.Fatalf("Check should not return error, got: %v", err)
	}

	// Should have no violations (good.go is small, bad.go is skipped)
	if len(report.Violations) != 0 {
		t.Fatalf("expected 0 violations, got %d", len(report.Violations))
	}

	// Should have exactly 1 skipped file (bad.go)
	if len(report.Skipped) != 1 {
		t.Fatalf("expected 1 skipped file, got %d", len(report.Skipped))
	}

	if !strings.Contains(report.Skipped[0].File, "bad.go") {
		t.Errorf("expected skipped file to be bad.go, got %q", report.Skipped[0].File)
	}
	if report.Skipped[0].Error == "" {
		t.Errorf("expected skipped file to have error message")
	}
}

func TestCheck_ExcludeFileSkipsFileAndReportsWhenDebug(t *testing.T) {
	// Create foo.go with a violation and foo_test.go with a violation
	tmpDir := t.TempDir()

	fooLines := []string{"package main", "", "func Foo() {"}
	for i := 0; i < 103; i++ {
		fooLines = append(fooLines, "\tx := 1")
	}
	fooLines = append(fooLines, "}")
	fooSrc := strings.Join(fooLines, "\n")

	if err := os.WriteFile(tmpDir+"/foo.go", []byte(fooSrc), 0644); err != nil {
		t.Fatalf("WriteFile foo.go failed: %v", err)
	}

	testLines := []string{"package main", "", "func TestFoo() {"}
	for i := 0; i < 103; i++ {
		testLines = append(testLines, "\tx := 1")
	}
	testLines = append(testLines, "}")
	testSrc := strings.Join(testLines, "\n")

	if err := os.WriteFile(tmpDir+"/foo_test.go", []byte(testSrc), 0644); err != nil {
		t.Fatalf("WriteFile foo_test.go failed: %v", err)
	}

	// With debug on, excluded files should appear
	report, err := funclen.Check([]string{tmpDir}, 100, funclen.Options{
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

	fooLines := []string{"package main", "", "func Foo() {"}
	for i := 0; i < 103; i++ {
		fooLines = append(fooLines, "\tx := 1")
	}
	fooLines = append(fooLines, "}")
	fooSrc := strings.Join(fooLines, "\n")

	if err := os.WriteFile(tmpDir+"/foo.go", []byte(fooSrc), 0644); err != nil {
		t.Fatalf("WriteFile foo.go failed: %v", err)
	}

	testLines := []string{"package main", "", "func TestFoo() {"}
	for i := 0; i < 103; i++ {
		testLines = append(testLines, "\tx := 1")
	}
	testLines = append(testLines, "}")
	testSrc := strings.Join(testLines, "\n")

	if err := os.WriteFile(tmpDir+"/foo_test.go", []byte(testSrc), 0644); err != nil {
		t.Fatalf("WriteFile foo_test.go failed: %v", err)
	}

	// With debug off, excluded files should NOT appear
	report, err := funclen.Check([]string{tmpDir}, 100, funclen.Options{
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
	report, err := funclen.Check([]string{tmpDir}, 100, funclen.Options{
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

	lines := []string{"package main", "", "// gardener:ignore", "func Foo() {"}
	// Make it 105 lines
	for i := 0; i < 103; i++ {
		lines = append(lines, "\tx := 1")
	}
	lines = append(lines, "}")
	src := strings.Join(lines, "\n")

	if err := os.WriteFile(tmpDir+"/test.go", []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	report, err := funclen.Check([]string{tmpDir}, 100, funclen.Options{
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
	lines := []string{"package main", "", "// gardenerignore", "func Foo() {"}
	// Make it 105 lines
	for i := 0; i < 103; i++ {
		lines = append(lines, "\tx := 1")
	}
	lines = append(lines, "}")
	src := strings.Join(lines, "\n")

	if err := os.WriteFile(tmpDir+"/test.go", []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	report, err := funclen.Check([]string{tmpDir}, 100, funclen.Options{
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
