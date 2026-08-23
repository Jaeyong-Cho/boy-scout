package crap

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// coverageData holds the parsed coverage profile grouped by import path,
// plus the module info needed to resolve a file path to its import path.
type coverageData struct {
	moduleRoot     string
	modulePath     string
	blocksByFile   map[string][]profileBlock
	filesInProfile map[string]bool
}

// loadCoverage finds the module root, runs go test to produce a coverage
// profile for paths, and parses it into a coverageData. The returned cleanup
// removes the temporary profile file and must be called once done.
func loadCoverage(paths []string) (data coverageData, cleanup func(), err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return coverageData{}, nil, err
	}

	moduleRoot, modulePath, err := findModule(cwd)
	if err != nil {
		return coverageData{}, nil, err
	}

	profilePath, cleanup, err := runGoTest(moduleRoot, paths)
	if err != nil {
		return coverageData{}, nil, err
	}

	blocks, err := readProfileBlocks(profilePath)
	if err != nil {
		cleanup()
		return coverageData{}, nil, err
	}

	blocksByFile, filesInProfile := groupBlocksByFile(blocks)

	return coverageData{
		moduleRoot:     moduleRoot,
		modulePath:     modulePath,
		blocksByFile:   blocksByFile,
		filesInProfile: filesInProfile,
	}, cleanup, nil
}

// readProfileBlocks opens and parses the coverage profile at profilePath.
func readProfileBlocks(profilePath string) ([]profileBlock, error) {
	profileFile, err := os.Open(profilePath)
	if err != nil {
		return nil, err
	}
	defer profileFile.Close()
	return parseProfile(profileFile)
}

// groupBlocksByFile indexes blocks by their source file, for coverage lookups.
func groupBlocksByFile(blocks []profileBlock) (blocksByFile map[string][]profileBlock, filesInProfile map[string]bool) {
	blocksByFile = make(map[string][]profileBlock)
	filesInProfile = make(map[string]bool)
	for _, block := range blocks {
		blocksByFile[block.file] = append(blocksByFile[block.file], block)
		filesInProfile[block.file] = true
	}
	return blocksByFile, filesInProfile
}

// importPathFor computes the import path go test coverage profiles use for filePath.
// filePath may be relative (as collected from CLI args) while moduleRoot is always
// absolute, so filepath.Rel would silently fail (err ignored) and yield an empty
// relPath, making every file miss its coverage entry. Resolve to absolute first so
// Rel always succeeds.
func importPathFor(filePath, moduleRoot, modulePath string) string {
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		absFilePath = filePath
	}
	relPath, _ := filepath.Rel(moduleRoot, absFilePath)
	return modulePath + "/" + strings.ReplaceAll(relPath, string(filepath.Separator), "/")
}

// findModule walks upward from startDir looking for a go.mod file.
// Returns the directory containing go.mod, the module path, and nil on success.
// Returns error if no go.mod is found up to the filesystem root.
var reModuleLine = regexp.MustCompile(`^module\s+(\S+)`)

// parseModulePath extracts the module path from the content of a go.mod file.
func parseModulePath(data []byte) (modulePath string, ok bool) {
	for _, line := range strings.Split(string(data), "\n") {
		if match := reModuleLine.FindStringSubmatch(line); match != nil {
			return match[1], true
		}
	}
	return "", false
}

func findModule(startDir string) (root, modulePath string, err error) {
	dir := startDir

	for {
		data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil {
			if mp, ok := parseModulePath(data); ok {
				return dir, mp, nil
			}
			return "", "", fmt.Errorf("go.mod found but no module line")
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "", "", fmt.Errorf("no go.mod found")
		}
		dir = parent
	}
}
