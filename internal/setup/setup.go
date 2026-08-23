package setup

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed SKILL.md references/*
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
	// Read the embedded SKILL.md template
	content, err := skillContent.ReadFile("SKILL.md")
	if err != nil {
		return "", fmt.Errorf("failed to read embedded skill template: %w", err)
	}

	assertf(len(content) > 0, "embedded SKILL.md is empty")
	assertNoMachineLocalPath("SKILL.md", content)
	assertf(!strings.Contains(string(content), "gardener"), "embedded SKILL.md still references old tool name 'gardener'")
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

// writeReferenceFiles writes all embedded reference files to skillDir/references/, recursively including subdirectories.
func writeReferenceFiles(skillDir string) error {
	refDir := filepath.Join(skillDir, "references")
	if err := os.MkdirAll(refDir, 0755); err != nil {
		return fmt.Errorf("failed to create references directory: %w", err)
	}

	return writeReferenceFilesRecursive("references", refDir)
}

func writeReferenceFilesRecursive(srcDir, dstDir string) error {
	entries, err := skillContent.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", srcDir, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())

		if entry.IsDir() {
			if err := processReferenceDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := writeReferenceFile(srcPath, dstPath, entry.Name()); err != nil {
				return err
			}
		}
	}

	return nil
}

func processReferenceDir(srcPath, dstPath string) error {
	if err := os.MkdirAll(dstPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dstPath, err)
	}
	return writeReferenceFilesRecursive(srcPath, dstPath)
}

func writeReferenceFile(srcPath, dstPath, name string) error {
	content, err := skillContent.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", srcPath, err)
	}

	assertNoMachineLocalPath(name, content)

	if err := os.WriteFile(dstPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", dstPath, err)
	}
	return nil
}
