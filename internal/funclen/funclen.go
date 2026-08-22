package funclen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"strings"

	"go-gardener/internal/gofiles"
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

	// Check each collected file
	for _, filePath := range filesToCheck {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
		if err != nil {
			report.Skipped = append(report.Skipped, SkippedFile{
				File:  filePath,
				Error: err.Error(),
			})
			continue
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			// Calculate function length: from opening { to closing }, inclusive
			startLine := fset.Position(fn.Body.Pos()).Line
			endLine := fset.Position(fn.Body.End()).Line
			length := endLine - startLine + 1

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

			if length > maxLines {
				assertf(length > maxLines, "appended violation does not exceed limit %d", maxLines)
				report.Violations = append(report.Violations, Violation{
					File:   filePath,
					Line:   startLine,
					Func:   fn.Name.Name,
					Length: length,
					Limit:  maxLines,
				})
			}
		}
	}

	return report, nil
}
