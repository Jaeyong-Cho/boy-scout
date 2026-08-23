/*
---
type: Source Code
title: duplication
description: Detect Type-1 (exact) and Type-2 (renamed identifier) function clones using token-based normalization and cross-file comparison.
tags: [boy-scout, clean-code-checks, duplication]
timestamp: 2026-08-24T00:00:00+09:00
---
*/

package duplication

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"strings"

	"boy-scout/internal/funcignore"
	"boy-scout/internal/srcfiles"
)

type Violation struct {
	FileA    string `json:"fileA"`
	LineA    int    `json:"lineA"`
	FuncA    string `json:"funcA"`
	FileB    string `json:"fileB"`
	LineB    int    `json:"lineB"`
	FuncB    string `json:"funcB"`
	Type     string `json:"type"`      // "Type-1" or "Type-2"
	DupLines int    `json:"dupLines"`  // duplicated line count
}

// SkippedFile is a type alias for srcfiles.SkippedFile
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
	Violations    []Violation
	Skipped       []SkippedFile
	ExcludedFiles []string
	ExcludedFuncs []ExcludedFunc
}

func assertf(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}

// FuncInfo holds parsed function metadata for comparison
type FuncInfo struct {
	File         string
	Line         int
	Func         *ast.FuncDecl
	Fset         *token.FileSet // FileSet needed for accurate line calculations
	RawSequence  []string
	BlindSequence []string
}

// tokenSequence extracts tokens from a function's source, returning two sequences:
// raw (unchanged) and blind (identifiers → aliases, literals → placeholders)
func tokenSequence(fn *ast.FuncDecl, fset *token.FileSet, src []byte) (raw, blind []string, err error) {
	if fn.Body == nil {
		return []string{}, []string{}, nil
	}

	startPos := fn.Body.Pos()
	endPos := fn.Body.End()

	// Extract source text for this function body
	// Position values are 1-indexed, but we need 0-indexed access
	start := int(startPos) - 1
	end := int(endPos)

	if start < 0 || end > len(src) || start >= end {
		return nil, nil, fmt.Errorf("invalid position range for function %s", fn.Name.Name)
	}

	funcSrc := src[start:end]

	// Tokenize using scanner
	s := scanner.Scanner{}
	s.Init(fset.AddFile("", -1, len(funcSrc)), funcSrc, nil, 0) // Don't scan comments

	// Map identifiers to positional aliases within this function
	identMap := make(map[string]string)
	nextID := 1

	raw = []string{}
	blind = []string{}

	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}

		rawStr := lit
		if lit == "" {
			rawStr = tok.String()
		}
		raw = append(raw, rawStr)

		// Compute blind version
		var blindStr string
		switch tok {
		case token.IDENT:
			if alias, exists := identMap[lit]; exists {
				blindStr = alias
			} else {
				alias = fmt.Sprintf("ID%d", nextID)
				identMap[lit] = alias
				nextID++
				blindStr = alias
			}
		case token.INT, token.FLOAT, token.STRING, token.CHAR, token.IMAG:
			blindStr = fmt.Sprintf("LIT_%s", tok.String())
		default:
			blindStr = rawStr
		}
		blind = append(blind, blindStr)
	}

	return raw, blind, nil
}

// funcLength computes the physical line count of a function's body
func funcLength(fn *ast.FuncDecl, fset *token.FileSet) int {
	if fn.Body == nil {
		return 0
	}
	startLine := fset.Position(fn.Body.Pos()).Line
	endLine := fset.Position(fn.Body.End()).Line
	return endLine - startLine + 1
}

// scanFileForDuplication parses filePath and extracts eligible functions
func scanFileForDuplication(filePath string, minLines int, opts Options) ([]FuncInfo, []ExcludedFunc, *SkippedFile) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, &SkippedFile{File: filePath, Error: err.Error()}
	}

	// Read source file for token extraction
	src, err := srcfiles.ReadFile(filePath)
	if err != nil {
		return nil, nil, &SkippedFile{File: filePath, Error: err.Error()}
	}

	var funcs []FuncInfo
	var excluded []ExcludedFunc

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		line := fset.Position(fn.Pos()).Line

		// Check if excluded by comment or flag
		if isExcluded, reason := funcignore.Reason(fn, opts.ExcludeFuncs, "duplication"); isExcluded {
			if opts.Debug {
				excluded = append(excluded, ExcludedFunc{File: filePath, Line: line, Func: fn.Name.Name, Reason: reason})
			}
			continue
		}

		// Check minimum length
		length := funcLength(fn, fset)
		if length < minLines {
			continue
		}

		// Extract token sequences
		raw, blind, err := tokenSequence(fn, fset, src)
		if err != nil {
			continue
		}

		funcs = append(funcs, FuncInfo{
			File:          filePath,
			Line:          line,
			Func:          fn,
			Fset:          fset,
			RawSequence:   raw,
			BlindSequence: blind,
		})
	}

	return funcs, excluded, nil
}

// classifyPair compares two functions and returns their clone type if they match
func classifyPair(a, b *FuncInfo) (cloneType string) {
	// Check if raw sequences match (Type-1)
	if sequenceEqual(a.RawSequence, b.RawSequence) {
		cloneType = "Type-1"
	} else if sequenceEqual(a.BlindSequence, b.BlindSequence) {
		cloneType = "Type-2"
	}

	// Assertion: if we classified as a clone, one of the sequences must match
	assertf(cloneType == "" || sequenceEqual(a.RawSequence, b.RawSequence) || sequenceEqual(a.BlindSequence, b.BlindSequence),
		"classified as %s but neither raw nor blind sequences match", cloneType)

	return cloneType
}

// sequenceEqual compares two token sequences for exact match
func sequenceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Check scans .go files (excluding _test.go) and reports function duplicates
func Check(paths []string, minLines int, opts Options) (Report, error) {
	assertf(minLines > 0, "minLines must be positive, got %d", minLines)

	report := Report{
		Violations:    []Violation{},
		Skipped:       []SkippedFile{},
		ExcludedFiles: []string{},
		ExcludedFuncs: []ExcludedFunc{},
	}

	// Collect all .go files
	filesToCheck, excludedFiles, skipped := srcfiles.Collect(paths, []string{".go"}, opts.ExcludeFiles)
	report.Skipped = append(report.Skipped, skipped...)
	if opts.Debug {
		report.ExcludedFiles = append(report.ExcludedFiles, excludedFiles...)
	}

	// Filter out _test.go files
	var nonTestFiles []string
	for _, f := range filesToCheck {
		if !strings.HasSuffix(f, "_test.go") {
			nonTestFiles = append(nonTestFiles, f)
		}
	}

	// Scan all eligible files and collect functions
	var allFuncs []FuncInfo
	for _, filePath := range nonTestFiles {
		funcs, excludedFuncs, skippedFile := scanFileForDuplication(filePath, minLines, opts)
		if skippedFile != nil {
			report.Skipped = append(report.Skipped, *skippedFile)
			continue
		}
		allFuncs = append(allFuncs, funcs...)
		if opts.Debug {
			report.ExcludedFuncs = append(report.ExcludedFuncs, excludedFuncs...)
		}
	}

	// Compare all unordered pairs
	seen := make(map[string]bool)
	for i := 0; i < len(allFuncs); i++ {
		for j := i + 1; j < len(allFuncs); j++ {
			a := &allFuncs[i]
			b := &allFuncs[j]

			// Assertion: never comparing a function against itself in unordered pairs
			assertf(i != j, "comparing function against itself")

			cloneType := classifyPair(a, b)
			if cloneType == "" {
				continue
			}

			// Stable ordering: earlier by (file, line) is always A
			var first, second *FuncInfo
			if a.File < b.File || (a.File == b.File && a.Line < b.Line) {
				first, second = a, b
			} else {
				first, second = b, a
			}

			key := fmt.Sprintf("%s:%d:%s|%s:%d:%s", first.File, first.Line, first.Func.Name.Name, second.File, second.Line, second.Func.Name.Name)
			if seen[key] {
				continue
			}
			seen[key] = true

			// Calculate duplicate line count from first function
			dupLines := funcLength(first.Func, first.Fset)

			violation := Violation{
				FileA:    first.File,
				LineA:    first.Line,
				FuncA:    first.Func.Name.Name,
				FileB:    second.File,
				LineB:    second.Line,
				FuncB:    second.Func.Name.Name,
				Type:     cloneType,
				DupLines: dupLines,
			}
			report.Violations = append(report.Violations, violation)
		}
	}

	return report, nil
}
