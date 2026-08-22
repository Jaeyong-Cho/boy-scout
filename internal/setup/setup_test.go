package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_CreatesSkillFileAtBaseDir(t *testing.T) {
	baseDir := t.TempDir()

	path, err := Run(baseDir, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	expectedPath := filepath.Join(baseDir, ".agents", "skills", "gardener-go", "SKILL.md")
	if path != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, path)
	}

	// Verify the file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("skill file not found at %q: %v", path, err)
	}

	// Verify the file is non-empty
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}
	if len(content) == 0 {
		t.Errorf("skill file is empty")
	}
}

func TestRun_OverwritesExistingSkillFile(t *testing.T) {
	baseDir := t.TempDir()

	// First run
	path1, err := Run(baseDir, "")
	if err != nil {
		t.Fatalf("first Run failed: %v", err)
	}

	content1, err := os.ReadFile(path1)
	if err != nil {
		t.Fatalf("failed to read skill file after first run: %v", err)
	}

	// Second run (should overwrite without error)
	path2, err := Run(baseDir, "")
	if err != nil {
		t.Fatalf("second Run failed: %v", err)
	}

	if path1 != path2 {
		t.Errorf("expected same path on second run, got %q vs %q", path1, path2)
	}

	content2, err := os.ReadFile(path2)
	if err != nil {
		t.Fatalf("failed to read skill file after second run: %v", err)
	}

	if string(content1) != string(content2) {
		t.Errorf("skill file content changed on overwrite (may be expected if template changed)")
	}
}

func TestRun_TemplateDeclaresUserInvokedFixLoop(t *testing.T) {
	baseDir := t.TempDir()

	path, err := Run(baseDir, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}

	contentStr := string(content)

	// Verify frontmatter
	if !strings.Contains(contentStr, "name: gardener-go") {
		t.Errorf("expected 'name: gardener-go' in frontmatter, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "description:") {
		t.Errorf("expected 'description:' in frontmatter, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "disable-model-invocation: true") {
		t.Errorf("expected 'disable-model-invocation: true' in frontmatter, got:\n%s", contentStr)
	}

	// Verify instructions
	if !strings.Contains(contentStr, "gardener-go all") {
		t.Errorf("expected 'gardener-go all' in skill body, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "go test") {
		t.Errorf("expected 'go test' in skill body, got:\n%s", contentStr)
	}
}

func TestRun_CopiesBinaryWhenPathGiven(t *testing.T) {
	baseDir := t.TempDir()

	binarySrc := filepath.Join(t.TempDir(), "gardener-go")
	if err := os.WriteFile(binarySrc, []byte("fake binary content"), 0755); err != nil {
		t.Fatalf("failed to write fake binary: %v", err)
	}

	_, err := Run(baseDir, binarySrc)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	binPath := filepath.Join(baseDir, ".agents", "bin", "gardener-go")
	content, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("expected binary copied to %q: %v", binPath, err)
	}
	if string(content) != "fake binary content" {
		t.Errorf("expected copied binary content to match source, got %q", content)
	}
}

func TestRun_ReturnsErrorWhenDirUnwritable(t *testing.T) {
	tmpDir := t.TempDir()
	agentFile := filepath.Join(tmpDir, ".agents")

	// Create a regular file at the .agent path to block directory creation
	if err := os.WriteFile(agentFile, []byte("blocking file"), 0644); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}

	path, err := Run(tmpDir, "")
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

	path, err := Run(baseDir, missingBinary)
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
	skillPath := filepath.Join(baseDir, ".agents", "skills", "gardener-go", "SKILL.md")
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		t.Fatalf("failed to pre-create skill path as directory: %v", err)
	}

	path, err := Run(baseDir, "")
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

func TestRun_TemplateDeclaresTDDRefactorAndViolationGuidance(t *testing.T) {
	baseDir := t.TempDir()

	path, err := Run(baseDir, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}
	contentStr := string(content)

	cases := []struct {
		name   string
		marker string
	}{
		{"BaselineGreenGate", "tests were already failing before any gardener-go edit"},
		{"FunclenExplainsCleanCodeRule", "one level of abstraction"},
		{"CrapExplainsCleanCodeRule", "high complexity plus low test coverage"},
		{"CharacterizationTestForZeroCoverage", "characterization test"},
		{"AttemptCapIncludesCharacterizationTest", "including its characterization test"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(contentStr, c.marker) {
				t.Errorf("expected skill template to contain %q, got:\n%s", c.marker, contentStr)
			}
		})
	}
}

func TestRun_TemplateHasNoMachineLocalPath(t *testing.T) {
	baseDir := t.TempDir()

	path, err := Run(baseDir, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}
	contentStr := string(content)

	if strings.Contains(contentStr, "/Users/") {
		t.Errorf("expected skill template to not contain /Users/, got:\n%s", contentStr)
	}
	if strings.Contains(contentStr, "~/workspace") {
		t.Errorf("expected skill template to not contain ~/workspace, got:\n%s", contentStr)
	}
}
