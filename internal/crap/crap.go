package crap

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"path"
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

type ExcludedFunc struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Func   string `json:"func"`
	Reason string `json:"reason"`
}

type Options struct {
	ExcludeFiles []string
	ExcludeFuncs []string
	Debug        bool
}

type Report struct {
	Violations    []Violation    `json:"violations"`
	Skipped       []SkippedFile  `json:"skipped"`
	ExcludedFiles []string       `json:"excludedFiles"`
	ExcludedFuncs []ExcludedFunc `json:"excludedFuncs"`
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

// excludeFuncReason checks if a function should be excluded based on name patterns or comment directives.
// Returns (excluded, reason) where reason is one of: "flag", "comment", or "" (if not excluded).
func excludeFuncReason(fn *ast.FuncDecl, patterns []string) (bool, string) {
	// Check if function name matches any exclude pattern
	for _, p := range patterns {
		if match, _ := path.Match(p, fn.Name.Name); match {
			return true, "flag"
		}
	}

	// Check for // gardener:ignore comment directive
	if fn.Doc != nil {
		for _, comment := range fn.Doc.List {
			text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			if text == "gardener:ignore" {
				return true, "comment"
			}
		}
	}

	return false, ""
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
func Check(paths []string, threshold float64, opts Options) (Report, error) {
	assertf(threshold > 0, "threshold must be positive, got %v", threshold)

	report := Report{
		Violations:    []Violation{},
		Skipped:       []SkippedFile{},
		ExcludedFiles: []string{},
		ExcludedFuncs: []ExcludedFunc{},
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
	filesToCheck, excludedFiles, skipped := gofiles.Collect(paths, opts.ExcludeFiles)
	report.Skipped = append(report.Skipped, skipped...)
	if opts.Debug {
		report.ExcludedFiles = append(report.ExcludedFiles, excludedFiles...)
	}

	// Scan each source file
	for _, filePath := range filesToCheck {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
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

			// Check if function is excluded
			excluded, reason := excludeFuncReason(fn, opts.ExcludeFuncs)
			if excluded {
				if opts.Debug {
					report.ExcludedFuncs = append(report.ExcludedFuncs, ExcludedFunc{
						File:   filePath,
						Line:   startLine,
						Func:   fn.Name.Name,
						Reason: reason,
					})
				}
				continue
			}

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
