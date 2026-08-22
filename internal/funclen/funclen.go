package funclen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

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

type Report struct {
	Violations []Violation
	Skipped    []SkippedFile
}

func assertf(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}

func Check(paths []string, maxLines int) (Report, error) {
	assertf(maxLines > 0, "maxLines must be positive, got %d", maxLines)

	report := Report{
		Violations: []Violation{},
		Skipped:    []SkippedFile{},
	}

	// Collect all .go files from the given paths
	filesToCheck, skipped := gofiles.Collect(paths)
	report.Skipped = append(report.Skipped, skipped...)

	// Check each collected file
	for _, filePath := range filesToCheck {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filePath, nil, 0)
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
