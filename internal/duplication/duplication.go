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
	FileA      string  `json:"fileA"`
	LineA      int     `json:"lineA"`
	FuncA      string  `json:"funcA"`
	FileB      string  `json:"fileB"`
	LineB      int     `json:"lineB"`
	FuncB      string  `json:"funcB"`
	Type       string  `json:"type"`       // "Type-1", "Type-2", or "Type-3"
	DupLines   int     `json:"dupLines"`   // duplicated line count
	Similarity float64 `json:"similarity"` // LCS-based similarity (Type-3 only)
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

// tokenToBlind maps a token to its blind representation (identifier→ID, literal→LIT_type)
func tokenToBlind(tok token.Token, lit string, rawStr string, identMap map[string]string, nextID *int) string {
	switch tok {
	case token.IDENT:
		if alias, exists := identMap[lit]; exists {
			return alias
		}
		alias := fmt.Sprintf("ID%d", *nextID)
		identMap[lit] = alias
		*nextID++
		return alias
	case token.INT, token.FLOAT, token.STRING, token.CHAR, token.IMAG:
		return fmt.Sprintf("LIT_%s", tok.String())
	default:
		return rawStr
	}
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
		blind = append(blind, tokenToBlind(tok, lit, rawStr, identMap, &nextID))
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

// processDeclsForFunctions extracts functions from AST declarations
func processDeclsForFunctions(filePath string, file *ast.File, fset *token.FileSet, src []byte, minLines int, opts Options) ([]FuncInfo, []ExcludedFunc) {
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

	return funcs, excluded
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

	funcs, excluded := processDeclsForFunctions(filePath, file, fset, src, minLines, opts)
	return funcs, excluded, nil
}

// classifyPair compares two functions and returns their clone type and similarity
// Type-1: exact raw match, Type-2: exact blind match, Type-3: above-threshold similarity on blind
// Returns ("", 0.0) if no match at or above minSimilarity
func classifyPair(a, b *FuncInfo, minSimilarity float64) (cloneType string, similarity float64) {
	// Check if raw sequences match (Type-1)
	if sequenceEqual(a.RawSequence, b.RawSequence) {
		return "Type-1", 1.0
	}

	// Check if blind sequences match exactly (Type-2)
	if sequenceEqual(a.BlindSequence, b.BlindSequence) {
		return "Type-2", 1.0
	}

	// Check Type-3: compute LCS similarity on blind sequences
	similarity = lcsSimilarity(a.BlindSequence, b.BlindSequence)
	if similarity >= minSimilarity {
		return "Type-3", similarity
	}

	return "", 0.0
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

// lcsSimilarity computes LCS-based similarity ratio: 2*LCS(a,b)/(len(a)+len(b))
// Returns a value in [0.0, 1.0] where 1.0 means identical sequences.
// ponytail: O(N²) time/space per pair; fine at function-sized sequences.
func lcsSimilarity(a, b []string) float64 {
	assertf(len(a) > 0 || len(b) > 0, "lcsSimilarity called with two empty sequences")

	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	// Compute LCS length using dynamic programming
	m, n := len(a), len(b)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				if dp[i-1][j] > dp[i][j-1] {
					dp[i][j] = dp[i-1][j]
				} else {
					dp[i][j] = dp[i][j-1]
				}
			}
		}
	}

	lcsLength := dp[m][n]
	ratio := float64(2*lcsLength) / float64(m+n)

	assertf(ratio >= 0.0 && ratio <= 1.0, "similarity ratio %f out of [0,1] range", ratio)

	return ratio
}

// collectAndFilterFiles gathers .go files and filters out test files
func collectAndFilterFiles(paths []string, opts Options) (nonTestFiles []string, report *Report, err error) {
	report = &Report{
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
	for _, f := range filesToCheck {
		if !strings.HasSuffix(f, "_test.go") {
			nonTestFiles = append(nonTestFiles, f)
		}
	}

	return nonTestFiles, report, nil
}

// scanFilesForFunctions scans eligible files and extracts all functions
func scanFilesForFunctions(files []string, minLines int, opts Options, report *Report) []FuncInfo {
	var allFuncs []FuncInfo
	for _, filePath := range files {
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
	return allFuncs
}

// reportDuplicates compares all function pairs and builds violation list with similarity threshold
func reportDuplicates(allFuncs []FuncInfo, minSimilarity float64) []Violation {
	var violations []Violation
	seen := make(map[string]bool)
	for i := 0; i < len(allFuncs); i++ {
		for j := i + 1; j < len(allFuncs); j++ {
			a := &allFuncs[i]
			b := &allFuncs[j]

			// Assertion: never comparing a function against itself in unordered pairs
			assertf(i != j, "comparing function against itself")

			cloneType, similarity := classifyPair(a, b, minSimilarity)
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
				FileA:      first.File,
				LineA:      first.Line,
				FuncA:      first.Func.Name.Name,
				FileB:      second.File,
				LineB:      second.Line,
				FuncB:      second.Func.Name.Name,
				Type:       cloneType,
				DupLines:   dupLines,
				Similarity: similarity,
			}
			violations = append(violations, violation)
		}
	}
	return violations
}

// CheckWithSimilarity scans .go files and reports function duplicates with LCS-based similarity threshold
func CheckWithSimilarity(paths []string, minLines int, minSimilarity float64, opts Options) (Report, error) {
	assertf(minLines > 0, "minLines must be positive, got %d", minLines)

	nonTestFiles, report, err := collectAndFilterFiles(paths, opts)
	if err != nil {
		return *report, err
	}

	allFuncs := scanFilesForFunctions(nonTestFiles, minLines, opts, report)
	report.Violations = reportDuplicates(allFuncs, minSimilarity)

	return *report, nil
}

// Check scans .go files (excluding _test.go) and reports function duplicates with default 0.70 similarity threshold
func Check(paths []string, minLines int, opts Options) (Report, error) {
	return CheckWithSimilarity(paths, minLines, 0.70, opts)
}
