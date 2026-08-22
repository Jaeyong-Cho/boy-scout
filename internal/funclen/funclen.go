package funclen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"strings"

	"gardener-go/internal/gofiles"
)

type Violation struct {
	File   string
	Line   int
	Func   string
	Length int
	Limit  int
}

// SkippedFile is a type alias for gofiles.SkippedFile, preserving the existing
// JSON field names and shape of the Violation output.
type SkippedFile = gofiles.SkippedFile

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

// evalFuncLen checks a single function's length, or reports why it was excluded.
// Exactly one of the two return values is non-nil.
func evalFuncLen(fn *ast.FuncDecl, fset *token.FileSet, filePath string, maxLines int, opts Options) (*Violation, *ExcludedFunc) {
	// Calculate function length: from opening { to closing }, inclusive
	startLine := fset.Position(fn.Body.Pos()).Line
	endLine := fset.Position(fn.Body.End()).Line
	length := endLine - startLine + 1

	if excluded, reason := excludeFuncReason(fn, opts.ExcludeFuncs); excluded {
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
	filesToCheck, excludedFiles, skipped := gofiles.Collect(paths, opts.ExcludeFiles)
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
