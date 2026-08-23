package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_ReturnsErrorWhenDirUnwritable(t *testing.T) {
	tmpDir := t.TempDir()
	agentFile := filepath.Join(tmpDir, ".agents")

	// Create a regular file at the .agent path to block directory creation
	if err := os.WriteFile(agentFile, []byte("blocking file"), 0644); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}

	path, err := Run(tmpDir, "", ".agents")
	if err == nil {
		t.Fatalf("expected error when directory is unwritable, but got none")
	}

	if path != "" {
		t.Errorf("expected empty path on error, got %q", path)
	}

	// Error should mention the failure
	if !strings.Contains(err.Error(), "failed to create") {
		t.Errorf("expected error about directory creation, got: %v", err)
	}
}

func TestRun_ReturnsErrorWhenBinaryUnreadable(t *testing.T) {
	baseDir := t.TempDir()
	missingBinary := filepath.Join(t.TempDir(), "does-not-exist")

	path, err := Run(baseDir, missingBinary, ".agents")
	if err == nil {
		t.Fatalf("expected error when binary path is unreadable, but got none")
	}
	if path != "" {
		t.Errorf("expected empty path on error, got %q", path)
	}
	if !strings.Contains(err.Error(), "failed to read binary") {
		t.Errorf("expected error about reading binary, got: %v", err)
	}
}

func TestRun_ReturnsErrorWhenSkillFileIsDirectory(t *testing.T) {
	baseDir := t.TempDir()
	skillPath := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "SKILL.md")
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		t.Fatalf("failed to pre-create skill path as directory: %v", err)
	}

	path, err := Run(baseDir, "", ".agents")
	if err == nil {
		t.Fatalf("expected error when skill file path is a directory, but got none")
	}
	if path != "" {
		t.Errorf("expected empty path on error, got %q", path)
	}
	if !strings.Contains(err.Error(), "failed to write skill file") {
		t.Errorf("expected error about writing skill file, got: %v", err)
	}
}
