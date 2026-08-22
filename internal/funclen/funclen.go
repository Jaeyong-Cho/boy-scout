package funclen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type Violation struct {
	File   string
	Line   int
	Func   string
	Length int
	Limit  int
}

type SkippedFile struct {
	File  string
	Error string
}

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
	var filesToCheck []string

	for _, path := range paths {
		stat, err := os.Stat(path)
		if err != nil {
			report.Skipped = append(report.Skipped, SkippedFile{
				File:  path,
				Error: err.Error(),
			})
			continue
		}

		if !stat.IsDir() {
			// It's a file, add it directly
			filesToCheck = append(filesToCheck, path)
		} else {
			// It's a directory, walk it
			err := filepath.WalkDir(path, func(filePath string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}

				// Skip vendor/ and dot-directories (but not the root)
				if d.IsDir() {
					name := d.Name()
					// Don't skip the root directory itself, only subdirectories
					if (name == "vendor" || strings.HasPrefix(name, ".")) && name != "." {
						return filepath.SkipDir
					}
					return nil
				}

				// Check if it's a .go file
				if strings.HasSuffix(filePath, ".go") {
					filesToCheck = append(filesToCheck, filePath)
				}

				return nil
			})

			if err != nil {
				report.Skipped = append(report.Skipped, SkippedFile{
					File:  path,
					Error: err.Error(),
				})
			}
		}
	}

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
