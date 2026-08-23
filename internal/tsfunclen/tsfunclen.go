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

	nodeType := node.Type()

	// Check if this is a function-like node we should report
	if nodeType == "function_declaration" || nodeType == "generator_function_declaration" || nodeType == "method_definition" {
		def := extractFunctionDef(node, source)
		if def != nil {
			*results = append(*results, *def)
		}
		// Don't recurse into nested functions for separate reporting
		return
	}

	// For named arrow functions: check if this is an arrow_function whose parent is variable_declarator
	if nodeType == "arrow_function" {
		parent := node.Parent()
		if parent != nil && parent.Type() == "variable_declarator" {
			def := extractArrowFunctionDef(node, source)
			if def != nil {
				*results = append(*results, *def)
			}
			// Don't recurse into the arrow function's body
			return
		}
		// For anonymous arrow functions, don't report them separately, just skip recursion
		return
	}

	// Recurse into children to find nested functions
	for i := uint32(0); i < node.ChildCount(); i++ {
		findFunctionDefinitions(node.Child(int(i)), source, results)
	}
}

type funcDef struct {
	name      string
	startLine int
	endLine   int
}

// extractFunctionDef extracts function name and body span from function_declaration, generator_function_declaration, or method_definition
func extractFunctionDef(node *sitter.Node, source []byte) *funcDef {
	nodeType := node.Type()
	assertf(nodeType == "function_declaration" || nodeType == "generator_function_declaration" || nodeType == "method_definition",
		"extractFunctionDef called on non-function node: %s", nodeType)

	// Find the body (statement_block child) and name (identifier child)
	var bodyNode *sitter.Node
	var nameNode *sitter.Node

	for i := uint32(0); i < node.ChildCount(); i++ {
		child := node.Child(int(i))
		childType := child.Type()
		if childType == "statement_block" {
			bodyNode = child
		}
		if childType == "identifier" || childType == "property_identifier" {
			if nameNode == nil {
				nameNode = child
			}
		}
	}

	// Body is required
	assertf(bodyNode != nil, "function-like node without statement_block (body): %s", nodeType)

	name := "?"
	if nameNode != nil {
		name = strings.TrimSpace(string(source[nameNode.StartByte():nameNode.EndByte()]))
	}

	// Get start and end line from body's range
	// Note: tree-sitter uses 0-indexed lines, but we want 1-indexed
	startLine := int(bodyNode.StartPoint().Row) + 1
	endLine := int(bodyNode.EndPoint().Row) + 1

	return &funcDef{
		name:      name,
		startLine: startLine,
		endLine:   endLine,
	}
}

// extractArrowFunctionDef extracts name and body from a named arrow function
func extractArrowFunctionDef(node *sitter.Node, source []byte) *funcDef {
	assertf(node.Type() == "arrow_function", "extractArrowFunctionDef called on non-arrow_function: %s", node.Type())

	parent := node.Parent()
	assertf(parent != nil && parent.Type() == "variable_declarator", "arrow_function not directly parented by variable_declarator")

	// Get name from the variable_declarator's first child (identifier)
	var nameNode *sitter.Node
	for i := uint32(0); i < parent.ChildCount(); i++ {
		child := parent.Child(int(i))
		if child.Type() == "identifier" {
			nameNode = child
			break
		}
	}

	// Get body from arrow_function's statement_block child
	var bodyNode *sitter.Node
	for i := uint32(0); i < node.ChildCount(); i++ {
		child := node.Child(int(i))
		if child.Type() == "statement_block" {
			bodyNode = child
			break
		}
	}

	assertf(bodyNode != nil, "arrow_function without statement_block (body)")

	name := "?"
	if nameNode != nil {
		name = strings.TrimSpace(string(source[nameNode.StartByte():nameNode.EndByte()]))
	}

	// Get start and end line from body's range
	startLine := int(bodyNode.StartPoint().Row) + 1
	endLine := int(bodyNode.EndPoint().Row) + 1

	return &funcDef{
		name:      name,
		startLine: startLine,
		endLine:   endLine,
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
