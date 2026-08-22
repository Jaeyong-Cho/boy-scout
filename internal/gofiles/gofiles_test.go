package gofiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollect_WalksDirectoryRecursivelySkippingVendorAndDotDirs(t *testing.T) {
	// Create a temporary directory tree
	tmpDir := t.TempDir()

	// Create a normal subdirectory with a .go file
	normalDir := filepath.Join(tmpDir, "normal")
	if err := os.Mkdir(normalDir, 0755); err != nil {
		t.Fatalf("failed to create normal dir: %v", err)
	}
	normalFile := filepath.Join(normalDir, "normal.go")
	if err := os.WriteFile(normalFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write normal.go: %v", err)
	}

	// Create a vendor directory with a .go file (should be skipped)
	vendorDir := filepath.Join(tmpDir, "vendor")
	if err := os.Mkdir(vendorDir, 0755); err != nil {
		t.Fatalf("failed to create vendor dir: %v", err)
	}
	vendorFile := filepath.Join(vendorDir, "vendor.go")
	if err := os.WriteFile(vendorFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write vendor.go: %v", err)
	}

	// Create a dot directory with a .go file (should be skipped)
	dotDir := filepath.Join(tmpDir, ".hidden")
	if err := os.Mkdir(dotDir, 0755); err != nil {
		t.Fatalf("failed to create dot dir: %v", err)
	}
	dotFile := filepath.Join(dotDir, "hidden.go")
	if err := os.WriteFile(dotFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write hidden.go: %v", err)
	}

	// Create a root-level .go file
	rootFile := filepath.Join(tmpDir, "root.go")
	if err := os.WriteFile(rootFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write root.go: %v", err)
	}

	// Call Collect
	files, skipped := Collect([]string{tmpDir})

	// Check results
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
