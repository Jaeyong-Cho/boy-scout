package gocomplexity

import (
	"go/ast"
	"go/parser"
	"go/token"

	"boy-scout/internal/assertutil"
	"boy-scout/internal/funcignore"
	"boy-scout/internal/srcfiles"
)

type Violation struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	Func       string `json:"func"`
	Complexity int    `json:"complexity"`
	Limit      int    `json:"limit"`
}

// SkippedFile is a type alias for srcfiles.SkippedFile, preserving the existing
// JSON field names and shape of the Violation output.
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

// evalFuncComplexity checks a single function's complexity, or reports why it was excluded.
// Exactly one of the two return values is non-nil.
func evalFuncComplexity(fn *ast.FuncDecl, fset *token.FileSet, filePath string, maxComplexity int, opts Options) (*Violation, *ExcludedFunc) {
	startLine := fset.Position(fn.Body.Pos()).Line

	if excluded, reason := funcignore.Reason(fn, opts.ExcludeFuncs, "complexity"); excluded {
		if !opts.Debug {
			return nil, nil
		}
		return nil, &ExcludedFunc{File: filePath, Line: startLine, Func: fn.Name.Name, Reason: reason}
	}

	complexity := CyclomaticComplexity(fn)
	if complexity <= maxComplexity {
		return nil, nil
	}

	assertutil.Assertf(complexity > maxComplexity, "appended violation does not exceed limit %d", maxComplexity)
	return &Violation{File: filePath, Line: startLine, Func: fn.Name.Name, Complexity: complexity, Limit: maxComplexity}, nil
}

// scanFileForComplexity parses filePath and evaluates the complexity of every non-excluded
// function in it. skipped is non-nil if the file itself couldn't be parsed.
func scanFileForComplexity(filePath string, maxComplexity int, opts Options) (violations []Violation, excludedFuncs []ExcludedFunc, skipped *SkippedFile) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, &SkippedFile{File: filePath, Error: err.Error()}
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		violation, excluded := evalFuncComplexity(fn, fset, filePath, maxComplexity, opts)
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

func Check(paths []string, maxComplexity int, opts Options) (Report, error) {
	assertutil.Assertf(maxComplexity > 0, "maxComplexity must be positive, got %d", maxComplexity)

	report := Report{
		Violations:    []Violation{},
		Skipped:       []SkippedFile{},
		ExcludedFiles: []string{},
		ExcludedFuncs: []ExcludedFunc{},
	}

	// Collect all .go files from the given paths
	filesToCheck, excludedFiles, skipped := srcfiles.Collect(paths, []string{".go"}, opts.ExcludeFiles)
	report.Skipped = append(report.Skipped, skipped...)
	if opts.Debug {
		report.ExcludedFiles = append(report.ExcludedFiles, excludedFiles...)
	}

	for _, filePath := range filesToCheck {
		violations, excludedFuncs, skippedFile := scanFileForComplexity(filePath, maxComplexity, opts)
		if skippedFile != nil {
			report.Skipped = append(report.Skipped, *skippedFile)
			continue
		}
		report.Violations = append(report.Violations, violations...)
		if opts.Debug {
			report.ExcludedFuncs = append(report.ExcludedFuncs, excludedFuncs...)
		}
	}

	return report, nil
}
