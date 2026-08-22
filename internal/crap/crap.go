package crap

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go-gardener/internal/gofiles"
)

type Violation struct {
	File       string  `json:"file"`
	Line       int     `json:"line"`
	Func       string  `json:"func"`
	Complexity int     `json:"complexity"`
	Coverage   float64 `json:"coverage"`
	Score      float64 `json:"score"`
	Threshold  float64 `json:"threshold"`
}

type SkippedFile = gofiles.SkippedFile

type Report struct {
	Violations []Violation  `json:"violations"`
	Skipped    []SkippedFile `json:"skipped"`
}

// crapScore calculates the CRAP score using the formula:
// CRAP(m) = comp(m)² × (1 − cov(m))³ + comp(m)
// where comp is cyclomatic complexity (int >= 1) and cov is coverage (float64 in [0.0, 1.0]).
func crapScore(comp int, cov float64) float64 {
	assertf(comp >= 1, "crapScore requires comp>=1, got %d", comp)
	assertf(cov >= 0 && cov <= 1, "crapScore requires cov in [0,1], got %f", cov)

	score := math.Pow(float64(comp), 2)*math.Pow(1-cov, 3) + float64(comp)
	return score
}

// evaluate computes the CRAP score and determines if it violates the threshold.
// A violation occurs when score > threshold (exactly equal to threshold is compliant).
func evaluate(comp int, cov float64, threshold float64) (score float64, violated bool) {
	score = crapScore(comp, cov)
	violated = score > threshold
	return
}

// findModule walks upward from startDir looking for a go.mod file.
// Returns the directory containing go.mod, the module path, and nil on success.
// Returns error if no go.mod is found up to the filesystem root.
func findModule(startDir string) (root, modulePath string, err error) {
	dir := startDir
	reModuleLine := regexp.MustCompile(`^module\s+(\S+)`)

	for {
		goModPath := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(goModPath)
		if err == nil {
			// Found it, parse the module path
			for _, line := range strings.Split(string(data), "\n") {
				if match := reModuleLine.FindStringSubmatch(line); match != nil {
					return dir, match[1], nil
				}
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

// Check scans the provided paths for Go source files and calculates their CRAP scores.
// It returns a report of all functions exceeding the threshold, or an error if
// the go.mod file is missing or go test fails to build.
func Check(paths []string, threshold float64) (Report, error) {
	assertf(threshold > 0, "threshold must be positive, got %v", threshold)

	report := Report{
		Violations: []Violation{},
		Skipped:    []SkippedFile{},
	}

	// Find the module root and module path
	cwd, err := os.Getwd()
	if err != nil {
		return report, err
	}

	moduleRoot, modulePath, err := findModule(cwd)
	if err != nil {
		return report, err
	}

	// Run go test to get coverage
	profilePath, cleanup, err := runGoTest(moduleRoot, paths)
	if err != nil {
		return report, err
	}
	defer cleanup()

	// Parse the coverage profile
	profileFile, err := os.Open(profilePath)
	if err != nil {
		return report, err
	}
	defer profileFile.Close()

	blocks, err := parseProfile(profileFile)
	if err != nil {
		return report, err
	}

	// Group blocks by file
	blocksByFile := make(map[string][]profileBlock)
	filesInProfile := make(map[string]bool)
	for _, block := range blocks {
		blocksByFile[block.file] = append(blocksByFile[block.file], block)
		filesInProfile[block.file] = true
	}

	// Collect source files
	filesToCheck, skipped := gofiles.Collect(paths)
	report.Skipped = append(report.Skipped, skipped...)

	// Scan each source file
	for _, filePath := range filesToCheck {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filePath, nil, 0)
		if err != nil {
			report.Skipped = append(report.Skipped, gofiles.SkippedFile{
				File:  filePath,
				Error: err.Error(),
			})
			continue
		}

		// Compute the expected import path. filePath may be relative (as collected
		// from CLI args) while moduleRoot is always absolute, so filepath.Rel would
		// silently fail (err ignored) and yield an empty relPath, making every file
		// miss its coverage entry. Resolve to absolute first so Rel always succeeds.
		absFilePath, err := filepath.Abs(filePath)
		if err != nil {
			absFilePath = filePath
		}
		relPath, _ := filepath.Rel(moduleRoot, absFilePath)
		importPath := modulePath + "/" + strings.ReplaceAll(relPath, string(filepath.Separator), "/")

		// Check if this file has coverage entries
		fileBlocks := blocksByFile[importPath]
		fileInProfile := filesInProfile[importPath]

		// Check each function in the file
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			// Get function position
			startLine := fset.Position(fn.Body.Pos()).Line
			endLine := fset.Position(fn.Body.End()).Line

			// Calculate complexity and coverage
			comp := cyclomaticComplexity(fn)
			cov := functionCoverage(fileBlocks, fileInProfile, startLine, endLine)

			// Evaluate CRAP score
			score, violated := evaluate(comp, cov, threshold)

			if violated {
				assertf(score > threshold, "appended violation score %f does not exceed threshold %f", score, threshold)
				report.Violations = append(report.Violations, Violation{
					File:       filePath,
					Line:       startLine,
					Func:       fn.Name.Name,
					Complexity: comp,
					Coverage:   cov,
					Score:      score,
					Threshold:  threshold,
				})
			}
		}
	}

	return report, nil
}
