package setup

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed skill.md references/*.md
var skillContent embed.FS

func assertf(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}

func assertNoMachineLocalPath(name string, content []byte) {
	contentStr := string(content)
	assertf(!strings.Contains(contentStr, "/Users/"), "embedded %s references a machine-local path; violation explanations must be self-contained", name)
	assertf(!strings.Contains(contentStr, "~/workspace"), "embedded %s references a machine-local path; violation explanations must be self-contained", name)
}

// Run creates a skill file at baseDir/{prefix}/skills/boy-scout/SKILL.md and copies
// the boy-scout binary to baseDir/{prefix}/bin/boy-scout. It overwrites if they already exist.
// It also writes reference files to baseDir/{prefix}/skills/boy-scout/references/.
// It returns the path to the written skill file.
func Run(baseDir string, binaryPath string, prefix string) (string, error) {
	// Read the embedded skill.md template
	content, err := skillContent.ReadFile("skill.md")
	if err != nil {
		return "", fmt.Errorf("failed to read embedded skill template: %w", err)
	}

	assertf(len(content) > 0, "embedded skill.md is empty")
	assertNoMachineLocalPath("skill.md", content)
	assertf(!strings.Contains(string(content), "gardener"), "embedded skill.md still references old tool name 'gardener'")
	assertf(prefix != "", "prefix must not be empty")

	// Build the target paths
	skillPath := filepath.Join(baseDir, prefix, "skills", "boy-scout", "SKILL.md")
	binPath := filepath.Join(baseDir, prefix, "bin", "boy-scout")

	if err := ensureDirs(filepath.Dir(skillPath), filepath.Dir(binPath)); err != nil {
		return "", err
	}

	// Write the skill file
	if err := os.WriteFile(skillPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write skill file: %w", err)
	}

	// Write reference files
	if err := writeReferenceFiles(filepath.Dir(skillPath)); err != nil {
		return "", err
	}

	if err := copyBinary(binaryPath, binPath); err != nil {
		return "", err
	}

	assertf(strings.HasSuffix(skillPath, filepath.Join(prefix, "skills", "boy-scout", "SKILL.md")), "unexpected skill path: %s", skillPath)

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

// writeReferenceFiles writes all embedded reference files to skillDir/references/.
func writeReferenceFiles(skillDir string) error {
	refDir := filepath.Join(skillDir, "references")
	if err := os.MkdirAll(refDir, 0755); err != nil {
		return fmt.Errorf("failed to create references directory: %w", err)
	}

	entries, err := skillContent.ReadDir("references")
	if err != nil {
		return fmt.Errorf("failed to read references directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		content, err := skillContent.ReadFile(filepath.Join("references", entry.Name()))
		if err != nil {
			return fmt.Errorf("failed to read reference file %s: %w", entry.Name(), err)
		}

		assertNoMachineLocalPath(entry.Name(), content)

		path := filepath.Join(refDir, entry.Name())
		if err := os.WriteFile(path, content, 0644); err != nil {
			return fmt.Errorf("failed to write reference file %s: %w", entry.Name(), err)
		}
	}

	return nil
}
