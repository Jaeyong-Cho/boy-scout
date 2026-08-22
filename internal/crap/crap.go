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

	"gardener-go/internal/funcignore"
	"gardener-go/internal/gofiles"
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

// evalFuncCrap evaluates a single function's CRAP score, or reports why it was excluded.
// Exactly one of the two return values is non-nil.
func evalFuncCrap(fn *ast.FuncDecl, fset *token.FileSet, filePath string, fileBlocks []profileBlock, fileInProfile bool, threshold float64, opts Options) (*Violation, *ExcludedFunc) {
	startLine := fset.Position(fn.Body.Pos()).Line
	endLine := fset.Position(fn.Body.End()).Line

	if excluded, reason := funcignore.Reason(fn, opts.ExcludeFuncs, "crap"); excluded {
		if !opts.Debug {
			return nil, nil
		}
		return nil, &ExcludedFunc{File: filePath, Line: startLine, Func: fn.Name.Name, Reason: reason}
	}

	comp := cyclomaticComplexity(fn)
	cov := functionCoverage(fileBlocks, fileInProfile, startLine, endLine)
	score, violated := evaluate(comp, cov, threshold)
	if !violated {
		return nil, nil
	}

	assertf(score > threshold, "appended violation score %f does not exceed threshold %f", score, threshold)
	return &Violation{
		File:       filePath,
		Line:       startLine,
		Func:       fn.Name.Name,
		Complexity: comp,
		Coverage:   cov,
		Score:      score,
		Threshold:  threshold,
	}, nil
}

// scanFileForCrap parses filePath and evaluates the CRAP score of every
// non-excluded function in it. skipped is non-nil if the file itself
// couldn't be parsed.
func scanFileForCrap(filePath string, data coverageData, threshold float64, opts Options) (violations []Violation, excludedFuncs []ExcludedFunc, skipped *SkippedFile) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, &SkippedFile{File: filePath, Error: err.Error()}
	}

	importPath := importPathFor(filePath, data.moduleRoot, data.modulePath)
	fileBlocks := data.blocksByFile[importPath]
	fileInProfile := data.filesInProfile[importPath]

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		violation, excluded := evalFuncCrap(fn, fset, filePath, fileBlocks, fileInProfile, threshold, opts)
		violations, excludedFuncs = appendEvalResult(violations, excludedFuncs, violation, excluded)
	}

	return violations, excludedFuncs, nil
}

// appendEvalResult appends violation/excluded to the respective slice if non-nil.
func appendEvalResult(violations []Violation, excludedFuncs []ExcludedFunc, violation *Violation, excluded *ExcludedFunc) ([]Violation, []ExcludedFunc) {
	if violation != nil {
		violations = append(violations, *violation)
	}
	if excluded != nil {
		excludedFuncs = append(excludedFuncs, *excluded)
	}
	return violations, excludedFuncs
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

	data, cleanup, err := loadCoverage(paths)
	if err != nil {
		return report, err
	}
	defer cleanup()

	filesToCheck, excludedFiles, skipped := gofiles.Collect(paths, opts.ExcludeFiles)
	report.Skipped = append(report.Skipped, skipped...)
	if opts.Debug {
		report.ExcludedFiles = append(report.ExcludedFiles, excludedFiles...)
	}

	for _, filePath := range filesToCheck {
		accumulateFileResults(&report, filePath, data, threshold, opts)
	}

	return report, nil
}

// accumulateFileResults scans filePath and merges its violations/excluded/skipped
// results into report.
func accumulateFileResults(report *Report, filePath string, data coverageData, threshold float64, opts Options) {
	violations, excludedFuncs, skippedFile := scanFileForCrap(filePath, data, threshold, opts)
	if skippedFile != nil {
		report.Skipped = append(report.Skipped, *skippedFile)
		return
	}
	report.Violations = append(report.Violations, violations...)
	if opts.Debug {
		report.ExcludedFuncs = append(report.ExcludedFuncs, excludedFuncs...)
	}
}
