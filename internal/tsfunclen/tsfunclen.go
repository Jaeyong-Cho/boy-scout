package tsfunclen

import (
	"fmt"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"boy-scout/internal/srcfiles"
)

type Violation struct {
	File   string
	Line   int
	Func   string
	Length int
	Limit  int
}

type SkippedFile = srcfiles.SkippedFile

type ExcludedFunc struct {
	File   string
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
	ExcludedFuncs []ExcludedFunc
}

func assertf(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}

// hasErrorNode recursively checks if any node in the tree is an ERROR node
func hasErrorNode(node *sitter.Node) bool {
	if node == nil {
		return false
	}
	if node.Type() == "ERROR" {
		return true
	}
	for i := uint32(0); i < node.ChildCount(); i++ {
		if hasErrorNode(node.Child(int(i))) {
			return true
		}
	}
	return false
}

// findFunctionDefinitions recursively finds all top-level function-like nodes
func findFunctionDefinitions(node *sitter.Node, source []byte, results *[]funcDef) {
	if node == nil {
		return
	}

	if isFunctionLikeDeclaration(node) {
		if def := extractFunctionDef(node, source); def != nil {
			*results = append(*results, *def)
		}
		return
	}

	if isNamedArrowFunction(node) {
		if def := extractArrowFunctionDef(node, source); def != nil {
			*results = append(*results, *def)
		}
		return
	}

	// Arrow functions that aren't named and other nodes: recurse into children
	for i := uint32(0); i < node.ChildCount(); i++ {
		findFunctionDefinitions(node.Child(int(i)), source, results)
	}
}

// matchesExcludeFunc checks if a function name matches any exclude pattern
func matchesExcludeFunc(funcName string, patterns []string) bool {
	for _, pattern := range patterns {
		if match, _ := filepath.Match(pattern, funcName); match {
			return true
		}
	}
	return false
}

// getLanguageForFile returns the appropriate tree-sitter language for a .ts or .tsx file
func getLanguageForFile(filePath string) *sitter.Language {
	if strings.HasSuffix(filePath, ".tsx") {
		return tsx.GetLanguage()
	}
	return typescript.GetLanguage()
}

// scanFileForTsLength parses a TypeScript file and evaluates functions
func scanFileForTsLength(filePath string, source []byte, maxLines int, opts Options) (violations []Violation, excludedFuncs []ExcludedFunc, skipped *SkippedFile) {
	parser := sitter.NewParser()
	defer parser.Close()

	lang := getLanguageForFile(filePath)
	parser.SetLanguage(lang)

	tree := parser.Parse(nil, source)
	defer tree.Close()

	root := tree.RootNode()

	// Check for parse errors
	if hasErrorNode(root) {
		return nil, nil, &SkippedFile{File: filePath, Error: "parse error: ERROR node found in tree"}
	}

	// Find all top-level function definitions
	var funcDefs []funcDef
	findFunctionDefinitions(root, source, &funcDefs)

	// Evaluate each function
	for _, fn := range funcDefs {
		length := fn.endLine - fn.startLine + 1

		// Check if function is excluded by glob pattern
		if matchesExcludeFunc(fn.name, opts.ExcludeFuncs) {
			excludedFuncs = append(excludedFuncs, ExcludedFunc{
				File:   filePath,
				Func:   fn.name,
				Reason: "matched exclude-func pattern",
			})
			continue
		}

		// Check if function exceeds limit
		if length > maxLines {
			assertf(length > maxLines, "appended violation does not exceed limit %d", maxLines)
			violations = append(violations, Violation{
				File:   filePath,
				Line:   fn.startLine,
				Func:   fn.name,
				Length: length,
				Limit:  maxLines,
			})
		}
	}

	return violations, excludedFuncs, skipped
}

// Check analyzes TypeScript files in the given paths for function length violations
func Check(paths []string, maxLines int, opts Options) (Report, error) {
	assertf(maxLines > 0, "maxLines must be positive, got %d", maxLines)

	files, excluded, skipped := srcfiles.Collect(paths, []string{".ts", ".tsx"}, opts.ExcludeFiles)

	var allViolations []Violation
	var allSkipped []SkippedFile
	var allExcludedFuncs []ExcludedFunc

	// Add files that were excluded by patterns
	for _, filePath := range excluded {
		allExcludedFuncs = append(allExcludedFuncs, ExcludedFunc{
			File:   filePath,
			Func:   "",
			Reason: "matched exclude-file pattern",
		})
	}

	// Add files that couldn't be accessed
	allSkipped = append(allSkipped, skipped...)

	for _, filePath := range files {
		source, err := srcfiles.ReadFile(filePath)
		if err != nil {
			allSkipped = append(allSkipped, SkippedFile{File: filePath, Error: err.Error()})
			continue
		}

		violations, excludedFuncs, skipped := scanFileForTsLength(filePath, source, maxLines, opts)
		if skipped != nil {
			// Invariant: syntax error file must produce zero violations and excluded funcs from this file
			assertf(len(violations) == 0, "syntax error file %s produced violations", filePath)
			assertf(len(excludedFuncs) == 0, "syntax error file %s produced excluded funcs", filePath)
			allSkipped = append(allSkipped, *skipped)
		} else {
			allViolations = append(allViolations, violations...)
			allExcludedFuncs = append(allExcludedFuncs, excludedFuncs...)
		}
	}

	return Report{
		Violations:    allViolations,
		Skipped:       allSkipped,
		ExcludedFuncs: allExcludedFuncs,
	}, nil
}
