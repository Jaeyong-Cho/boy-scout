package gofiles

import (
	"os"
	"path/filepath"
	"testing"
)

// writeGoFile writes content to path, creating any parent directories needed.
func writeGoFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func TestCollect_WalksDirectoryRecursivelySkippingVendorAndDotDirs(t *testing.T) {
	// Create a temporary directory tree: a normal subdir, a vendor/ dir and a
	// dot-dir (both should be skipped), and a root-level file.
	tmpDir := t.TempDir()

	writeGoFile(t, filepath.Join(tmpDir, "normal/normal.go"), "package main\n")
	writeGoFile(t, filepath.Join(tmpDir, "vendor/vendor.go"), "package main\n")
	writeGoFile(t, filepath.Join(tmpDir, ".hidden/hidden.go"), "package main\n")
	writeGoFile(t, filepath.Join(tmpDir, "root.go"), "package main\n")

	// Call Collect
	files, excluded, skipped := Collect([]string{tmpDir}, nil)

	// Check results
	if len(excluded) > 0 {
		t.Errorf("expected no excluded files, got %d: %v", len(excluded), excluded)
	}
	if len(skipped) > 0 {
		t.Errorf("expected no skipped files, got %d: %v", len(skipped), skipped)
	}

	// Should have 2 files: normal/normal.go and root.go
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d: %v", len(files), files)
	}

	// Verify the files are correct (vendor and dot-dirs are excluded)
	fileMap := make(map[string]bool)
	for _, f := range files {
		rel, _ := filepath.Rel(tmpDir, f)
		fileMap[rel] = true
	}

	if !fileMap["normal/normal.go"] {
		t.Error("expected normal/normal.go to be included")
	}
	if !fileMap["root.go"] {
		t.Error("expected root.go to be included")
	}
	if fileMap["vendor/vendor.go"] {
		t.Error("expected vendor/vendor.go to be excluded")
	}
	if fileMap[".hidden/hidden.go"] {
		t.Error("expected .hidden/hidden.go to be excluded")
	}
}

func TestCollect_ExcludesFilesMatchingGlobPattern(t *testing.T) {
	// Create a temporary directory with foo.go and foo_test.go
	tmpDir := t.TempDir()

	fooFile := filepath.Join(tmpDir, "foo.go")
	if err := os.WriteFile(fooFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write foo.go: %v", err)
	}

	testFile := filepath.Join(tmpDir, "foo_test.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write foo_test.go: %v", err)
	}

	// Call Collect with exclude pattern
	files, excluded, _ := Collect([]string{tmpDir}, []string{"*_test.go"})

	if len(excluded) != 1 {
		t.Errorf("expected 1 excluded file, got %d: %v", len(excluded), excluded)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d: %v", len(files), files)
	}

	// Verify the excluded file is foo_test.go
	fileMap := make(map[string]bool)
	for _, f := range files {
		rel, _ := filepath.Rel(tmpDir, f)
		fileMap[rel] = true
	}

	if fileMap["foo_test.go"] {
		t.Error("expected foo_test.go to be excluded")
	}
	if !fileMap["foo.go"] {
		t.Error("expected foo.go to be included")
	}
}

func TestCollect_ExcludeGlobMatchesByBasenameOrFullPath(t *testing.T) {
	// Create a nested directory structure
	tmpDir := t.TempDir()

	pkgDir := filepath.Join(tmpDir, "pkg")
	if err := os.Mkdir(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create pkg dir: %v", err)
	}

	testFile := filepath.Join(pkgDir, "foo_test.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write foo_test.go: %v", err)
	}

	// Call Collect with pattern that has no slash (should match by basename)
	files, excluded, _ := Collect([]string{tmpDir}, []string{"*_test.go"})

	if len(excluded) != 1 {
		t.Errorf("expected 1 excluded file, got %d: %v", len(excluded), excluded)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d: %v", len(files), files)
	}
}

func TestCollect_ExcludePatternMatchingNothingIsNoOp(t *testing.T) {
	// Create a temporary directory with foo.go
	tmpDir := t.TempDir()

	fooFile := filepath.Join(tmpDir, "foo.go")
	if err := os.WriteFile(fooFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write foo.go: %v", err)
	}

	// Call Collect with pattern that matches nothing
	files, excluded, _ := Collect([]string{tmpDir}, []string{"*.nomatch"})

	if len(excluded) != 0 {
		t.Errorf("expected 0 excluded files, got %d: %v", len(excluded), excluded)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d: %v", len(files), files)
	}
}
