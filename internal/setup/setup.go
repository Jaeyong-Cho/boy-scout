package setup

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed skill.md
var skillContent embed.FS

func assertf(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}

// Run creates a skill file at baseDir/{prefix}/skills/gardener/SKILL.md and copies
// the gardener binary to baseDir/{prefix}/bin/gardener. It overwrites if they already exist.
// It returns the path to the written skill file.
func Run(baseDir string, binaryPath string, prefix string) (string, error) {
	// Read the embedded skill.md template
	content, err := skillContent.ReadFile("skill.md")
	if err != nil {
		return "", fmt.Errorf("failed to read embedded skill template: %w", err)
	}

	assertf(len(content) > 0, "embedded skill.md is empty")
	assertf(!strings.Contains(string(content), "/Users/"), "embedded skill.md references a machine-local path; violation explanations must be self-contained")
	assertf(!strings.Contains(string(content), "~/workspace"), "embedded skill.md references a machine-local path; violation explanations must be self-contained")
	assertf(prefix != "", "prefix must not be empty")

	// Build the target paths
	skillPath := filepath.Join(baseDir, prefix, "skills", "gardener", "SKILL.md")
	binPath := filepath.Join(baseDir, prefix, "bin", "gardener")

	if err := ensureDirs(filepath.Dir(skillPath), filepath.Dir(binPath)); err != nil {
		return "", err
	}

	// Write the skill file
	if err := os.WriteFile(skillPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write skill file: %w", err)
	}

	if err := copyBinary(binaryPath, binPath); err != nil {
		return "", err
	}

	assertf(strings.HasSuffix(skillPath, filepath.Join(prefix, "skills", "gardener", "SKILL.md")), "unexpected skill path: %s", skillPath)

	return skillPath, nil
}

// ensureDirs creates the parent directories for the skill file and binary.
func ensureDirs(skillDir, binDir string) error {
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}
	return nil
}

// copyBinary copies the binary at binaryPath to binPath. A no-op if binaryPath is empty.
func copyBinary(binaryPath, binPath string) error {
	if binaryPath == "" {
		return nil
	}
	binaryContent, err := os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to read binary: %w", err)
	}
	if err := os.WriteFile(binPath, binaryContent, 0755); err != nil {
		return fmt.Errorf("failed to write binary: %w", err)
	}
	return nil
}
