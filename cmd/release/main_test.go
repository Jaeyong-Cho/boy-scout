package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestMain_PrintsVersionOrNone verifies that cmd/release prints a version
// matching ^v[0-9]+\.[0-9]+\.[0-9]+$ or the literal "none".
func TestMain_PrintsVersionOrNone(t *testing.T) {
	// Build the release binary to a temp path.
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "release")

	buildCmd := exec.Command("go", "build", "-o", binPath, "boy-scout/cmd/release")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build release binary: %v", err)
	}

	// Create a temp git fixture repo.
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.Mkdir(repoDir, 0755); err != nil {
		t.Fatalf("failed to create temp repo dir: %v", err)
	}

	// Initialize the repo.
	initCmd := exec.Command("git", "init")
	initCmd.Dir = repoDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("failed to git init: %v", err)
	}

	// Configure git user for commits.
	configCmd := exec.Command("git", "config", "user.email", "test@example.com")
	configCmd.Dir = repoDir
	if err := configCmd.Run(); err != nil {
		t.Fatalf("failed to git config user.email: %v", err)
	}

	configCmd = exec.Command("git", "config", "user.name", "Test User")
	configCmd.Dir = repoDir
	if err := configCmd.Run(); err != nil {
		t.Fatalf("failed to git config user.name: %v", err)
	}

	// Create an initial commit with a bump-worthy subject.
	if err := os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = repoDir
	if err := addCmd.Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}

	commitCmd := exec.Command("git", "commit", "-m", "feat: initial feature")
	commitCmd.Dir = repoDir
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}

	// Run the built release binary in the fixture repo.
	releaseCmd := exec.Command(binPath)
	releaseCmd.Dir = repoDir
	var stdout, stderr bytes.Buffer
	releaseCmd.Stdout = &stdout
	releaseCmd.Stderr = &stderr
	releaseCmd.Run() // Don't check error; cmd/release may legitimately exit non-zero.

	output := stdout.String()
	trimmedOutput := strings.TrimSpace(output)

	// Should match ^v[0-9]+\.[0-9]+\.[0-9]+$ or be exactly "none".
	if trimmedOutput != "none" {
		versionRegex := regexp.MustCompile(`^v\d+\.\d+\.\d+$`)
		if !versionRegex.MatchString(trimmedOutput) {
			t.Errorf("expected version matching ^v[0-9]+\\.[0-9]+\\.[0-9]+$ or 'none', got %q", trimmedOutput)
		}
	}
}
