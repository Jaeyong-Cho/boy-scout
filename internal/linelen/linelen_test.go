package linelen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"runtime"
)

func TestCheck_ReportsLineOverLimit(t *testing.T) {
	tmpdir := t.TempDir()
	filename := filepath.Join(tmpdir, "test.go")

	// Create a file with one 105-char line (no quotes)
	// "x = " (4 chars) + "1"*101 = 105 chars total
	line105 := "x = " + strings.Repeat("1", 101)

	if err := os.WriteFile(filename, []byte(line105+"\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	opts := Options{
		ExcludeFiles: []string{},
		Debug:        false,
	}

	report, err := Check([]string{tmpdir}, 100, []string{".go"}, opts)
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
	if v.Line != 1 {
		t.Errorf("expected Line=1, got %d", v.Line)
	}
}

func TestCheck_ExemptsLineThatFitsAfterStrippingQuotes(t *testing.T) {
	tmpdir := t.TempDir()
	filename := filepath.Join(tmpdir, "test.go")

	// Create a file with a 130-char line where the quoted substring is 40 chars.
	// After stripping: 130 - 40 = 90 chars (fits under 100)
	// Line structure: "x = " (4) + "quoted_" (7) + "quoted string here " (40) + "_more" (5) + " = 1" (4) + misc (70)
	line := "x = " + strings.Repeat("a", 43) + "\"" + strings.Repeat("b", 40) + "\"" + strings.Repeat("c", 37)

	if err := os.WriteFile(filename, []byte(line+"\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	opts := Options{
		ExcludeFiles: []string{},
		Debug:        false,
	}

	report, err := Check([]string{tmpdir}, 100, []string{".go"}, opts)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Fatalf("expected 0 violations (line fits after quote stripping), got %d", len(report.Violations))
	}
}

func TestCheck_ReportsLineThatStillOverflowsAfterStripping(t *testing.T) {
	tmpdir := t.TempDir()
	filename := filepath.Join(tmpdir, "test.go")

	// Create a file with a 130-char line where the quoted substring is only 10 chars.
	// After stripping: 130 - 10 - 2 (quotes) = 118 chars (still over 100)
	// Need: 4 + a_count + 1 + 10 + 1 + c_count = 130
	// So: a_count + c_count = 114
	line := "x = " + strings.Repeat("a", 57) + "\"" + strings.Repeat("b", 10) + "\"" + strings.Repeat("c", 57)

	if err := os.WriteFile(filename, []byte(line+"\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	opts := Options{
		ExcludeFiles: []string{},
		Debug:        false,
	}

	report, err := Check([]string{tmpdir}, 100, []string{".go"}, opts)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Fatalf("expected 1 violation (line still over 100 after stripping), got %d", len(report.Violations))
	}

	v := report.Violations[0]
	// 4 + 57 + 1 + 10 + 1 + 57 = 130
	if v.Length != 130 {
		t.Errorf("expected Length=130, got %d", v.Length)
	}
}

func TestCheck_LineAtExactLimitIsCompliant(t *testing.T) {
	tmpdir := t.TempDir()
	filename := filepath.Join(tmpdir, "test.go")

	// Create a file with a line of exactly 100 characters
	line100 := strings.Repeat("x", 100)

	if err := os.WriteFile(filename, []byte(line100+"\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	opts := Options{
		ExcludeFiles: []string{},
		Debug:        false,
	}

	report, err := Check([]string{tmpdir}, 100, []string{".go"}, opts)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Fatalf("expected 0 violations (line at exact limit is compliant), got %d", len(report.Violations))
	}
}

func TestCheck_LineOneOverLimitIsViolation(t *testing.T) {
	tmpdir := t.TempDir()
	filename := filepath.Join(tmpdir, "test.go")

	// Create a file with a line of exactly 101 characters, no quotes
	line101 := strings.Repeat("x", 101)

	if err := os.WriteFile(filename, []byte(line101+"\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	opts := Options{
		ExcludeFiles: []string{},
		Debug:        false,
	}

	report, err := Check([]string{tmpdir}, 100, []string{".go"}, opts)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Fatalf("expected 1 violation (line one over limit), got %d", len(report.Violations))
	}

	v := report.Violations[0]
	if v.Length != 101 {
		t.Errorf("expected Length=101, got %d", v.Length)
	}
}

func TestCheck_EmptyFileHasNoViolations(t *testing.T) {
	tmpdir := t.TempDir()
	filename := filepath.Join(tmpdir, "test.go")

	// Create an empty file
	if err := os.WriteFile(filename, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	opts := Options{
		ExcludeFiles: []string{},
		Debug:        false,
	}

	report, err := Check([]string{tmpdir}, 100, []string{".go"}, opts)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Fatalf("expected 0 violations (empty file), got %d", len(report.Violations))
	}
}

func TestCheck_SkipsUnreadableFile(t *testing.T) {
	// Skip on Windows since permission denied works differently
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	tmpdir := t.TempDir()
	filename := filepath.Join(tmpdir, "test.go")

	// Create a readable file first
	if err := os.WriteFile(filename, []byte("x = 1\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Make it unreadable
	if err := os.Chmod(filename, 0000); err != nil {
		t.Fatalf("failed to chmod test file: %v", err)
	}
	t.Cleanup(func() {
		// Restore permissions for cleanup
		os.Chmod(filename, 0644)
	})

	opts := Options{
		ExcludeFiles: []string{},
		Debug:        false,
	}

	report, err := Check([]string{tmpdir}, 100, []string{".go"}, opts)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Fatalf("expected 0 violations (file is unreadable), got %d", len(report.Violations))
	}

	if len(report.Skipped) != 1 {
		t.Fatalf("expected 1 skipped file, got %d", len(report.Skipped))
	}

	skipped := report.Skipped[0]
	if skipped.File != filename {
		t.Errorf("expected skipped file %s, got %s", filename, skipped.File)
	}
	if skipped.Error == "" {
		t.Errorf("expected error message in skipped file, got empty string")
	}
}

func TestCheck_TemplateLiteralInterpolationBehaviorPinned(t *testing.T) {
	// This test pins the CURRENT (documented, not necessarily "correct") behavior:
	// The naive stripper treats the whole backtick span as one strippable unit,
	// so a line whose overflow is actually inside ${...} gets wrongly exempted.
	// This is a known limitation documented in the ponytail comment.
	// This test exists to catch if that behavior changes unintentionally.

	tmpdir := t.TempDir()
	filename := filepath.Join(tmpdir, "test.ts")

	// Create a line over 100 chars where the overflow is INSIDE the template literal interpolation.
	// Structure: `short ${` (7) + long_expression (90+) + `}` (1) = over 100
	// Since backticks are treated as quote chars for TS, the whole thing gets stripped
	// So: 0 chars after stripping, which is < 100, so NO VIOLATION reported.
	longExpr := strings.Repeat("x", 95)
	line := "`short ${" + longExpr + "}` = 1"
	// Length: 9 + 95 + 4 = 108 chars

	if err := os.WriteFile(filename, []byte(line+"\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	opts := Options{
		ExcludeFiles: []string{},
		Debug:        false,
	}

	report, err := Check([]string{tmpdir}, 100, []string{".ts"}, opts)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// With the naive stripper, the entire backtick span is removed, leaving " = 1" (4 chars)
	// which is < 100, so no violation is reported.
	// This is WRONG behavior (the line is actually over 100), but it's the known limitation.
	if len(report.Violations) != 0 {
		t.Errorf("PINNED: naive stripper currently removes the entire template literal, so no violation is reported. Got %d violations (behavior changed?)", len(report.Violations))
	}
}
