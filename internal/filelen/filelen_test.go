/*
---
type: Source Code
title: filelen tests
description: Unit tests for the file line-count checker, verifying correct line counting, exclusion handling, and skip behavior.
tags: [boy-scout, clean-code-checks, test]
timestamp: 2026-08-22T00:00:00+09:00
---
*/

package filelen_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"boy-scout/internal/filelen"
)

// writeFile writes content to path, creating parent directories as needed.
// If path already exists, it overwrites it.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("MkdirAll %s failed: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile %s failed: %v", path, err)
	}
}

// lineSrc generates source code with exactly n lines, each being "// line N".
// The result contains n newlines (one after each line) except there's no trailing newline,
// so we have n-1 newlines + trailing content = n lines.
func lineSrc(n int) string {
	if n == 0 {
		return ""
	}
	lines := make([]string, n)
	for i := 0; i < n; i++ {
		lines[i] = "// line " + fmt.Sprintf("%d", i+1)
	}
	// Don't add trailing newline - this ensures the file has n newlines + trailing content = n lines
	return strings.Join(lines, "\n")
}

// Test: Given a file with 300 lines or fewer - When filelen.Check runs - Then no violations
func TestCheck_NoViolationUnderLimit(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "ok.go")
	writeFile(t, filePath, lineSrc(300))

	report, err := filelen.Check([]string{filePath}, 300, []string{".go"}, filelen.Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(report.Violations))
	}
}

// Test: Given a file with 301 lines - When filelen.Check runs - Then one violation with correct details
func TestCheck_ReportsViolationOverLimit(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "toolong.go")
	writeFile(t, filePath, lineSrc(301))

	report, err := filelen.Check([]string{filePath}, 300, []string{".go"}, filelen.Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Errorf("expected 1 violation, got %d", len(report.Violations))
		return
	}

	v := report.Violations[0]
	if v.File != filePath {
		t.Errorf("expected file %s, got %s", filePath, v.File)
	}
	if v.Lines != 301 {
		t.Errorf("expected 301 lines, got %d", v.Lines)
	}
	if v.Limit != 300 {
		t.Errorf("expected limit 300, got %d", v.Limit)
	}
}

// Test: Given a file of exactly 300 lines - When checked against limit 300 - Then no violation
func TestCheck_ExactlyAtLimitIsCompliant(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "exact.go")
	writeFile(t, filePath, lineSrc(300))

	report, err := filelen.Check([]string{filePath}, 300, []string{".go"}, filelen.Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations for file at exactly the limit, got %d", len(report.Violations))
	}
}

// Test: Given a completely empty file (0 bytes) - When checked - Then it reports 0 lines and no violation
func TestCheck_EmptyFileIsCompliant(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "empty.go")
	writeFile(t, filePath, "")

	report, err := filelen.Check([]string{filePath}, 300, []string{".go"}, filelen.Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations for empty file, got %d", len(report.Violations))
	}
	if len(report.Skipped) != 0 {
		t.Errorf("expected 0 skipped for empty file, got %d", len(report.Skipped))
	}
}

// Test: Given a directory with nested subdirs, some files over limit, one in vendor/, one in .hidden/
// When filelen.Check runs - Then violations come from every non-skipped file and vendor/.hidden/ are not scanned
func TestCheck_WalksDirectoryRecursivelySkippingVendorAndDotDirs(t *testing.T) {
	dir := t.TempDir()

	// File that should be checked and violate
	writeFile(t, filepath.Join(dir, "normal.go"), lineSrc(350))

	// File in vendor/ that should be skipped
	writeFile(t, filepath.Join(dir, "vendor", "dep.go"), lineSrc(350))

	// File in .hidden/ that should be skipped
	writeFile(t, filepath.Join(dir, ".hidden", "secret.go"), lineSrc(350))

	// File in a normal subdir that should be checked
	writeFile(t, filepath.Join(dir, "subdir", "nested.go"), lineSrc(320))

	report, err := filelen.Check([]string{dir}, 300, []string{".go"}, filelen.Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Should have violations from normal.go and nested.go, not vendor or .hidden
	if len(report.Violations) != 2 {
		t.Errorf("expected 2 violations (normal.go, nested.go), got %d", len(report.Violations))
		for i, v := range report.Violations {
			t.Logf("  violation %d: %s", i, v.File)
		}
	}
}

// Test: Given an empty directory (no matching files) - When checked - Then both violations and skipped are empty
func TestCheck_EmptyDirectoryProducesEmptyReport(t *testing.T) {
	dir := t.TempDir()

	report, err := filelen.Check([]string{dir}, 300, []string{".go"}, filelen.Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations for empty directory, got %d", len(report.Violations))
	}
	if len(report.Skipped) != 0 {
		t.Errorf("expected 0 skipped for empty directory, got %d", len(report.Skipped))
	}
}

// Test: Given a file with no read permission - When checked - Then it appears in Skipped
func TestCheck_SkipsUnreadableFileAndContinues(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod semantics differ on Windows")
	}

	dir := t.TempDir()

	// Create one readable file that should be checked
	readablePath := filepath.Join(dir, "readable.go")
	writeFile(t, readablePath, lineSrc(350))

	// Create one unreadable file
	unreadablePath := filepath.Join(dir, "unreadable.go")
	writeFile(t, unreadablePath, lineSrc(350))
	if err := os.Chmod(unreadablePath, 0000); err != nil {
		t.Fatalf("Chmod failed: %v", err)
	}
	t.Cleanup(func() {
		os.Chmod(unreadablePath, 0644) // restore for cleanup
	})

	report, err := filelen.Check([]string{dir}, 300, []string{".go"}, filelen.Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Should have one violation (readable.go) and one skipped (unreadable.go)
	if len(report.Violations) != 1 {
		t.Errorf("expected 1 violation, got %d", len(report.Violations))
	}
	if len(report.Skipped) != 1 {
		t.Errorf("expected 1 skipped, got %d", len(report.Skipped))
	}
}

// Test: Given --exclude-file matching a file that would violate - When checked - Then it's not reported
func TestCheck_RespectsExcludeFilePattern(t *testing.T) {
	dir := t.TempDir()

	// File that would violate
	violatingPath := filepath.Join(dir, "toobig.go")
	writeFile(t, violatingPath, lineSrc(350))

	// File within limit
	okPath := filepath.Join(dir, "ok.go")
	writeFile(t, okPath, lineSrc(250))

	report, err := filelen.Check([]string{dir}, 300, []string{".go"}, filelen.Options{
		ExcludeFiles: []string{"toobig.go"},
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations (excluded pattern should filter toobig.go), got %d", len(report.Violations))
	}
}
