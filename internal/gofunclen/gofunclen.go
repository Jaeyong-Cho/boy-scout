package gofunclen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"gardener-go/internal/funcignore"
	"gardener-go/internal/srcfiles"
)

type Violation struct {
	File   string
	Line   int
	Func   string
	Length int
	Limit  int
}

// SkippedFile is a type alias for srcfiles.SkippedFile, preserving the existing
// JSON field names and shape of the Violation output.
type SkippedFile = srcfiles.SkippedFile

type ExcludedFunc struct {
	File   string
	Line   int
	Func   string
	Reason string
}

type Options struct {
	ExcludeFiles []string
	ExcludeFuncs []string
	Debug        bool
}

type Report struct {
	Violations   []Violation
	Skipped      []SkippedFile
	ExcludedFiles []string
	ExcludedFuncs []ExcludedFunc
}

func assertf(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}

// evalFuncLen checks a single function's length, or reports why it was excluded.
// Exactly one of the two return values is non-nil.
func evalFuncLen(fn *ast.FuncDecl, fset *token.FileSet, filePath string, maxLines int, opts Options) (*Violation, *ExcludedFunc) {
	// Calculate function length: from opening { to closing }, inclusive
	startLine := fset.Position(fn.Body.Pos()).Line
	endLine := fset.Position(fn.Body.End()).Line
	length := endLine - startLine + 1

	if excluded, reason := funcignore.Reason(fn, opts.ExcludeFuncs, "gofunclen"); excluded {
		if !opts.Debug {
			return nil, nil
		}
		return nil, &ExcludedFunc{File: filePath, Line: startLine, Func: fn.Name.Name, Reason: reason}
	}

	if length <= maxLines {
		return nil, nil
	}

	assertf(length > maxLines, "appended violation does not exceed limit %d", maxLines)
	return &Violation{File: filePath, Line: startLine, Func: fn.Name.Name, Length: length, Limit: maxLines}, nil
}

// scanFileForLength parses filePath and evaluates the length of every non-excluded
// function in it. skipped is non-nil if the file itself couldn't be parsed.
func scanFileForLength(filePath string, maxLines int, opts Options) (violations []Violation, excludedFuncs []ExcludedFunc, skipped *SkippedFile) {
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

		violation, excluded := evalFuncLen(fn, fset, filePath, maxLines, opts)
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

func Check(paths []string, maxLines int, opts Options) (Report, error) {
	assertf(maxLines > 0, "maxLines must be positive, got %d", maxLines)

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
		violations, excludedFuncs, skippedFile := scanFileForLength(filePath, maxLines, opts)
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
