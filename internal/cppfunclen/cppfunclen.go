package cppfunclen

import (
	"path/filepath"
	"strings"

	"boy-scout/internal/assertutil"
	"boy-scout/internal/srcfiles"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/cpp"
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

// findFunctionDefinitions recursively finds all top-level function_definition nodes
func findFunctionDefinitions(node *sitter.Node, source []byte, results *[]funcDef) {
	if node == nil {
		return
	}
	if node.Type() == "function_definition" {
		// Extract the function definition
		def := extractFunctionDef(node, source)
		if def != nil {
			*results = append(*results, *def)
		}
		// Don't recurse into nested functions for separate reporting
		// (but their lines will be counted as part of the enclosing function)
		return
	}
	// Recurse into children to find nested function_definitions
	for i := uint32(0); i < node.ChildCount(); i++ {
		findFunctionDefinitions(node.Child(int(i)), source, results)
	}
}

type funcDef struct {
	name      string
	startLine int
	endLine   int
}

// extractFunctionDef extracts the function name and body span from a function_definition node
func extractFunctionDef(node *sitter.Node, source []byte) *funcDef {
	assertutil.Assertf(node.Type() == "function_definition", "extractFunctionDef called on non-function_definition node: %s", node.Type())

	// Find the body (compound_statement child)
	var bodyNode *sitter.Node
	var declaratorNode *sitter.Node

	for i := uint32(0); i < node.ChildCount(); i++ {
		child := node.Child(int(i))
		if child.Type() == "compound_statement" {
			bodyNode = child
		}
		if child.Type() == "function_declarator" {
			declaratorNode = child
		}
	}

	// If no body, skip (it's a declaration, not a definition)
	if bodyNode == nil {
		return nil
	}

	assertutil.Assertf(bodyNode != nil, "function_definition node without compound_statement (body) field: %s", node.Type())

	// Extract function name
	name := extractFunctionName(declaratorNode, source)

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

// extractFunctionName extracts the qualified or simple name of a function.
// For out-of-line methods like Widget::resize(), it extracts the qualified name.
func extractFunctionName(declaratorNode *sitter.Node, source []byte) string {
	if declaratorNode == nil {
		return "?"
	}
	if name := declaratorIdentifierName(declaratorNode, source); name != "" {
		return name
	}
	return declaratorFallbackName(declaratorNode, source)
}

// declaratorIdentifierName walks the declarator's direct children looking for
// an identifier/qualified_identifier/field_identifier node, returning "" if none found.
// The structure is: function_declarator -> qualified_identifier or identifier.
func declaratorIdentifierName(declaratorNode *sitter.Node, source []byte) string {
	for i := uint32(0); i < declaratorNode.ChildCount(); i++ {
		child := declaratorNode.Child(int(i))
		if child.Type() == "qualified_identifier" || child.Type() == "identifier" || child.Type() == "field_identifier" {
			return strings.TrimSpace(string(source[child.StartByte():child.EndByte()]))
		}
	}
	return ""
}

// declaratorFallbackName extracts the declarator text before its parameter
// list, used when no identifier child node was found.
func declaratorFallbackName(declaratorNode *sitter.Node, source []byte) string {
	text := string(source[declaratorNode.StartByte():declaratorNode.EndByte()])
	if idx := strings.Index(text, "("); idx != -1 {
		return strings.TrimSpace(text[:idx])
	}
	return "?"
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

// scanFileForCppLength parses a C++ file and evaluates functions
func scanFileForCppLength(filePath string, source []byte, maxLines int, opts Options) (violations []Violation, excludedFuncs []ExcludedFunc, skipped *SkippedFile) {
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(cpp.GetLanguage())

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
			if opts.Debug {
				excludedFuncs = append(excludedFuncs, ExcludedFunc{
					File:   filePath,
					Func:   fn.name,
					Reason: "flag",
				})
			}
			continue
		}

		// Check if length exceeds limit
		if length > maxLines {
			assertutil.Assertf(length > maxLines, "appended violation does not exceed limit %d", maxLines)
			violations = append(violations, Violation{
				File:   filePath,
				Line:   fn.startLine,
				Func:   fn.name,
				Length: length,
				Limit:  maxLines,
			})
		}
	}

	return violations, excludedFuncs, nil
}

// Check walks the provided paths and returns a report of all C++ functions exceeding maxLines.
//
// NOTE: Comment-based boy-scout:ignore support is not implemented for C++ — this was
// deferred to a future plan. Tree-sitter's C++ grammar comment-handling was never
// verified, so we don't build on it here.
func Check(paths []string, maxLines int, opts Options) (Report, error) {
	assertutil.Assertf(maxLines > 0, "maxLines must be positive, got %d", maxLines)

	report := Report{
		Violations:    []Violation{},
		Skipped:       []SkippedFile{},
		ExcludedFuncs: []ExcludedFunc{},
	}

	// Collect all .cpp/.h/.hpp files from the given paths
	filesToCheck, _, skipped := srcfiles.Collect(paths, []string{".cpp", ".h", ".hpp"}, opts.ExcludeFiles)
	report.Skipped = append(report.Skipped, skipped...)

	for _, filePath := range filesToCheck {
		// Read file contents
		source, err := srcfiles.ReadFile(filePath)
		if err != nil {
			report.Skipped = append(report.Skipped, SkippedFile{File: filePath, Error: err.Error()})
			continue
		}

		violations, excludedFuncs, skippedFile := scanFileForCppLength(filePath, source, maxLines, opts)
		if skippedFile != nil {
			// Invariant: syntax error file must produce zero violations and excluded funcs from this file
			assertutil.Assertf(len(violations) == 0, "syntax error file %s produced violations", filePath)
			assertutil.Assertf(len(excludedFuncs) == 0, "syntax error file %s produced excluded funcs", filePath)
			report.Skipped = append(report.Skipped, *skippedFile)
			continue
		}
		report.Violations = append(report.Violations, violations...)
		report.ExcludedFuncs = append(report.ExcludedFuncs, excludedFuncs...)
	}

	return report, nil
}
