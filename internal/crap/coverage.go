package crap

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type profileBlock struct {
	file      string
	startLine int
	endLine   int
	numStmt   int
	count     int
}

// parseProfile parses a coverage profile in the standard go test -coverprofile format.
// It skips the "mode: ..." line and parses each block line.
// Each line has format: file:startLine.startCol,endLine.endCol numStmt count
func parseProfile(r io.Reader) ([]profileBlock, error) {
	var blocks []profileBlock
	scanner := bufio.NewScanner(r)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip the mode line (first line)
		if lineNum == 1 && strings.HasPrefix(line, "mode:") {
			continue
		}

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Parse the block line: file:startLine.startCol,endLine.endCol numStmt count
		// Example: example.com/pkg/file.go:3.14,5.2 2 1
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("malformed coverage line at line %d: %s", lineNum, line)
		}

		file := parts[0]
		rest := parts[1]

		// Parse the range and counts: "3.14,5.2 2 1"
		var startLine, startCol, endLine, endCol, numStmt, count int
		_, err := fmt.Sscanf(rest, "%d.%d,%d.%d %d %d",
			&startLine, &startCol, &endLine, &endCol, &numStmt, &count)
		if err != nil {
			return nil, fmt.Errorf("malformed coverage block at line %d: %s", lineNum, line)
		}

		blocks = append(blocks, profileBlock{
			file:      file,
			startLine: startLine,
			endLine:   endLine,
			numStmt:   numStmt,
			count:     count,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return blocks, nil
}

// functionCoverage calculates the fraction of statements executed in a function.
// - If fileInProfile is false, the file is not in the coverage profile at all,
//   so coverage is 0%.
// - If fileInProfile is true but there are no matching blocks in the range,
//   and the sum of numStmt is 0, the function is vacuously 100% covered
//   (no statements to miss).
// - Otherwise, coverage is (statements executed) / (total statements).
func functionCoverage(blocks []profileBlock, fileInProfile bool, startLine, endLine int) float64 {
	if !fileInProfile {
		return 0.0
	}

	var totalStmt int
	var coveredStmt int

	for _, block := range blocks {
		// Check if block overlaps with function range
		if block.startLine >= startLine && block.endLine <= endLine {
			totalStmt += block.numStmt
			if block.count > 0 {
				coveredStmt += block.numStmt
			}
		}
	}

	// If no statements found, vacuously 100% covered
	if totalStmt == 0 {
		return 1.0
	}

	cov := float64(coveredStmt) / float64(totalStmt)
	assertf(cov >= 0 && cov <= 1, "coverage fraction out of range: %f", cov)
	return cov
}

// testTargets converts a list of paths to go test package patterns.
// A file path uses its parent directory; a directory path is used as-is.
// Paths are converted to "./..." patterns, and duplicates are removed.
func testTargets(paths []string) []string {
	targets := make(map[string]struct{})

	for _, path := range paths {
		stat, err := os.Stat(path)
		if err != nil {
			// Skip unreachable paths
			continue
		}

		var dir string
		if stat.IsDir() {
			dir = path
		} else {
			dir = filepath.Dir(path)
		}

		// Convert to go test pattern
		var target string
		if dir == "." {
			target = "./..."
		} else {
			target = dir + "/..."
		}

		targets[target] = struct{}{}
	}

	// Convert map to slice
	var result []string
	for t := range targets {
		result = append(result, t)
	}
	return result
}

// runGoTest runs go test -covermode=set -coverprofile for the given paths.
// It returns the path to the generated coverage profile, a cleanup function,
// and any error that prevented the coverage file from being produced.
// If go test compiles but fails assertions, it still returns the profile
// (partial coverage is used).
func runGoTest(dir string, paths []string) (profilePath string, cleanup func(), err error) {
	tmpFile, err := os.CreateTemp("", "gardener-crap-*.out")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close() // Close the handle, we just need the path

	targets := testTargets(paths)
	args := []string{"test", "-covermode=set", "-coverprofile=" + tmpPath}
	args = append(args, targets...)

	cmd := exec.Command("go", args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()

	// Check if the command failed (this includes build failures)
	// But we only return the error if the profile file wasn't created or is empty.
	// If tests failed but the coverage file was produced, we can still use it.
	if err != nil {
		// Command failed - check if profile file exists and has content
		if _, statErr := os.Stat(tmpPath); statErr != nil {
			// Profile file not created at all, this is a build failure
			return "", nil, fmt.Errorf("go test failed: %s", string(output))
		}
		// Profile file exists, might be partial coverage from a build error
		// We consider it a fatal error only if there's no coverage data
		// (just the mode line). Check file size.
		info, _ := os.Stat(tmpPath)
		if info.Size() <= 11 { // "mode: set\n" is ~10 bytes
			return "", nil, fmt.Errorf("go test failed to build: %s", string(output))
		}
	}

	// Profile file exists with content, return success
	return tmpPath, func() {
		os.Remove(tmpPath)
	}, nil
}
