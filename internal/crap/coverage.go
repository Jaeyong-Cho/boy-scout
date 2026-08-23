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

		if shouldSkipProfileLine(line, lineNum) {
			continue
		}

		block, err := parseProfileLine(line, lineNum)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return blocks, nil
}

// shouldSkipProfileLine reports whether line should be skipped: either the
// leading "mode: ..." line, or a blank line.
func shouldSkipProfileLine(line string, lineNum int) bool {
	if lineNum == 1 && strings.HasPrefix(line, "mode:") {
		return true
	}
	return strings.TrimSpace(line) == ""
}

// parseProfileLine parses a single coverage block line: file:startLine.startCol,endLine.endCol numStmt count
// Example: example.com/pkg/file.go:3.14,5.2 2 1
func parseProfileLine(line string, lineNum int) (profileBlock, error) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return profileBlock{}, fmt.Errorf("malformed coverage line at line %d: %s", lineNum, line)
	}

	file := parts[0]
	rest := parts[1]

	var startLine, startCol, endLine, endCol, numStmt, count int
	_, err := fmt.Sscanf(rest, "%d.%d,%d.%d %d %d",
		&startLine, &startCol, &endLine, &endCol, &numStmt, &count)
	if err != nil {
		return profileBlock{}, fmt.Errorf("malformed coverage block at line %d: %s", lineNum, line)
	}

	return profileBlock{
		file:      file,
		startLine: startLine,
		endLine:   endLine,
		numStmt:   numStmt,
		count:     count,
	}, nil
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

	totalStmt, coveredStmt := sumOverlapping(blocks, startLine, endLine)

	// If no statements found, vacuously 100% covered
	if totalStmt == 0 {
		return 1.0
	}

	cov := float64(coveredStmt) / float64(totalStmt)
	assertCoverageInRange(cov)
	return cov
}

// sumOverlapping sums the statement counts of blocks that fall within [startLine, endLine].
func sumOverlapping(blocks []profileBlock, startLine, endLine int) (totalStmt, coveredStmt int) {
	for _, block := range blocks {
		if block.startLine >= startLine && block.endLine <= endLine {
			totalStmt += block.numStmt
			if block.count > 0 {
				coveredStmt += block.numStmt
			}
		}
	}
	return totalStmt, coveredStmt
}

func assertCoverageInRange(cov float64) {
	assertf(cov >= 0 && cov <= 1, "coverage fraction out of range: %f", cov)
}

// testTargets converts a list of paths to go test package patterns.
// A file path uses its parent directory; a directory path is used as-is.
// Paths are converted to "./..." patterns, and duplicates are removed.
func testTargets(paths []string) []string {
	targets := make(map[string]struct{})

	for _, path := range paths {
		target, ok := testTarget(path)
		if ok {
			targets[target] = struct{}{}
		}
	}

	// Convert map to slice
	var result []string
	for t := range targets {
		result = append(result, t)
	}
	return result
}

// testTarget converts a single path to its go test package pattern (e.g. "./..." or
// "dir/..."). ok is false if the path is unreachable and should be skipped.
func testTarget(path string) (target string, ok bool) {
	stat, err := os.Stat(path)
	if err != nil {
		return "", false
	}

	dir := path
	if !stat.IsDir() {
		dir = filepath.Dir(path)
	}

	if dir == "." {
		return "./...", true
	}
	return dir + "/...", true
}

// runGoTest runs go test -covermode=set -coverprofile for the given paths.
// It returns the path to the generated coverage profile, a cleanup function,
// and any error that prevented the coverage file from being produced.
// If go test compiles but fails assertions, it still returns the profile
// (partial coverage is used).
func runGoTest(dir string, paths []string) (profilePath string, cleanup func(), err error) {
	tmpFile, err := os.CreateTemp("", "boy-scout-crap-*.out")
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
