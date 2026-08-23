package gofunclen_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gardener-go/internal/gofunclen"
)

// writeFile writes content to path, failing the test on error.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile %s failed: %v", path, err)
	}
}

// funcSrc generates Go source for "package main" with a single function named
// funcName containing fillerLines statement lines (plus any preamble lines,
// e.g. a comment directive, placed before the func declaration).
func funcSrc(funcName string, fillerLines int, preamble ...string) string {
	lines := append([]string{"package main", ""}, preamble...)
	lines = append(lines, "func "+funcName+"() {")
	for i := 0; i < fillerLines; i++ {
		lines = append(lines, "\tx := 1")
	}
	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

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

	report, err := gofunclen.Check([]string{tmpFile}, 100, gofunclen.Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(report.Violations))
	}
}

func TestCheck_ReportsViolationOverLimit(t *testing.T) {
	// funcSrc("LongFunc", 103) spans from the { line to the } line for
	// endLine - startLine + 1 = 105 lines, i.e. a violation of the 100-line limit.
	src := funcSrc("LongFunc", 103)

	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/long.go"
	writeFile(t, tmpFile, src)

	report, err := gofunclen.Check([]string{tmpFile}, 100, gofunclen.Options{})
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
	// funcSrc("Exactly100", 98) spans exactly 100 lines from { to }, the limit itself.
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/exact.go"
	writeFile(t, tmpFile, funcSrc("Exactly100", 98))

	report, err := gofunclen.Check([]string{tmpFile}, 100, gofunclen.Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations at limit, got %d", len(report.Violations))
	}
}

func TestCheck_WalksDirectoryRecursivelySkippingVendorAndDotDirs(t *testing.T) {
	// Build a temp dir tree with violations under normal dirs, and would-be
	// violations under vendor/ and a dot-dir that should be skipped.
	tmpDir := t.TempDir()

	fixtures := []struct {
		dir     string
		file    string
		fn      string
		skipped bool
	}{
		{"pkg", "a.go", "ViolatingA", false},
		{"pkg/sub", "b.go", "ViolatingB", false},
		{"vendor/dep", "c.go", "ViolatingC", true},
		{".git", "d.go", "ViolatingD", true},
	}

	for _, fx := range fixtures {
		dir := filepath.Join(tmpDir, fx.dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
		writeFile(t, filepath.Join(dir, fx.file), funcSrc(fx.fn, 103))
	}

	report, err := gofunclen.Check([]string{tmpDir}, 100, gofunclen.Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 2 {
		t.Fatalf("expected 2 violations, got %d", len(report.Violations))
	}

	funcs := make(map[string]bool)
	for _, v := range report.Violations {
		funcs[v.Func] = true
	}

	for _, fx := range fixtures {
		if funcs[fx.fn] == fx.skipped {
			t.Errorf("violation presence for %s: got %v, want %v", fx.fn, funcs[fx.fn], !fx.skipped)
		}
	}
}

func TestCheck_EmptyDirectoryProducesEmptyReport(t *testing.T) {
	tmpDir := t.TempDir()

	report, err := gofunclen.Check([]string{tmpDir}, 100, gofunclen.Options{})
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

	report, err := gofunclen.Check([]string{tmpDir}, 100, gofunclen.Options{})
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

