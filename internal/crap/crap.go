package crap

import (
	"go/ast"
	"go/parser"
	"go/token"
	"math"

	"boy-scout/internal/assertutil"
	"boy-scout/internal/funcignore"
	"boy-scout/internal/gocomplexity"
	"boy-scout/internal/srcfiles"
)

// defaultExcludeFiles are excluded from CRAP scoring even when the caller
// passes no ExcludeFiles of its own. go test -coverprofile never
// instruments _test.go files, so functionCoverage always returns 0.0 for
// any function inside one — the CRAP formula is meaningless for test code
// by construction, not just noisy, so this default has no opt-out flag.
var defaultExcludeFiles = []string{"*_test.go"}

type Violation struct {
	File       string  `json:"file"`
	Line       int     `json:"line"`
	Func       string  `json:"func"`
	Complexity int     `json:"complexity"`
	Coverage   float64 `json:"coverage"`
	Score      float64 `json:"score"`
	Threshold  float64 `json:"threshold"`
}

type SkippedFile = srcfiles.SkippedFile

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
	assertutil.Assertf(comp >= 1, "crapScore requires comp>=1, got %d", comp)
	assertutil.Assertf(cov >= 0 && cov <= 1, "crapScore requires cov in [0,1], got %f", cov)

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

	comp := gocomplexity.CyclomaticComplexity(fn)
	cov := functionCoverage(fileBlocks, fileInProfile, startLine, endLine)
	score, violated := evaluate(comp, cov, threshold)
	if !violated {
		return nil, nil
	}

	assertutil.Assertf(score > threshold, "appended violation score %f does not exceed threshold %f", score, threshold)
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
	assertutil.Assertf(threshold > 0, "threshold must be positive, got %v", threshold)

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

	excludeFiles := append(append([]string{}, defaultExcludeFiles...), opts.ExcludeFiles...)
	assertutil.Assertf(len(excludeFiles) >= len(defaultExcludeFiles), "crap.Check: merged exclude patterns lost the default test-file exclude")
	filesToCheck, excludedFiles, skipped := srcfiles.Collect(paths, []string{".go"}, excludeFiles)
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
