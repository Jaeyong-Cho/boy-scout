package cppcomplexity

import (
	"strings"

	"boy-scout/internal/assertutil"
	"boy-scout/internal/srcfiles"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/cpp"
)

type Violation struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	Func       string `json:"func"`
	Complexity int    `json:"complexity"`
	Limit      int    `json:"limit"`
}

type SkippedFile = srcfiles.SkippedFile

type ExcludedFunc struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Func   string `json:"func"`
	Reason string `json:"reason"`
}

type Options struct {
	ExcludeFiles []string
	ExcludeFuncs []string
	Debug        bool
}

type Report struct {
	Violations    []Violation    `json:"violations"`
	Skipped       []SkippedFile  `json:"skipped"`
	ExcludedFuncs []ExcludedFunc `json:"excludedFuncs"`
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

type funcDef struct {
	name      string
	startLine int
	node      *sitter.Node
}

// findFunctionDefinitions recursively finds all top-level function_definition nodes
func findFunctionDefinitions(node *sitter.Node, source []byte, results *[]funcDef) {
	if node == nil {
		return
	}
	if node.Type() == "function_definition" {
		def := extractFunctionDef(node, source)
		if def != nil {
			*results = append(*results, *def)
		}
		// Don't recurse into nested functions
		return
	}
	for i := uint32(0); i < node.ChildCount(); i++ {
		findFunctionDefinitions(node.Child(int(i)), source, results)
	}
}

// extractFunctionDef extracts the function name and AST node from a function_definition node
func extractFunctionDef(node *sitter.Node, source []byte) *funcDef {
	assertutil.Assertf(node.Type() == "function_definition", "extractFunctionDef called on non-function_definition node: %s", node.Type())

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

	if bodyNode == nil {
		return nil
	}

	assertutil.Assertf(bodyNode != nil, "function_definition node without compound_statement (body) field: %s", node.Type())

	name := extractFunctionName(declaratorNode, source)
	startLine := int(bodyNode.StartPoint().Row) + 1

	return &funcDef{
		name:      name,
		startLine: startLine,
		node:      node,
	}
}

// extractFunctionName extracts the qualified or simple name of a function
func extractFunctionName(declaratorNode *sitter.Node, source []byte) string {
	if declaratorNode == nil {
		return "?"
	}
	if name := declaratorIdentifierName(declaratorNode, source); name != "" {
		return name
	}
	return declaratorFallbackName(declaratorNode, source)
}

// declaratorIdentifierName walks the declarator's direct children looking for an identifier node
func declaratorIdentifierName(declaratorNode *sitter.Node, source []byte) string {
	for i := uint32(0); i < declaratorNode.ChildCount(); i++ {
		child := declaratorNode.Child(int(i))
		if child.Type() == "qualified_identifier" || child.Type() == "identifier" || child.Type() == "field_identifier" {
			return strings.TrimSpace(string(source[child.StartByte():child.EndByte()]))
		}
	}
	return ""
}

// declaratorFallbackName extracts the declarator text before its parameter list
func declaratorFallbackName(declaratorNode *sitter.Node, source []byte) string {
	text := string(source[declaratorNode.StartByte() : declaratorNode.EndByte()])
	if idx := strings.Index(text, "("); idx > 0 {
		text = text[:idx]
	}
	return strings.TrimSpace(text)
}

// cyclomaticComplexity walks a function's body and counts decision points
func cyclomaticComplexity(funcNode *sitter.Node, source []byte) int {
	assertutil.Assertf(funcNode.Type() == "function_definition", "cyclomaticComplexity called on non-function_definition node: %s", funcNode.Type())
	complexity := 1 // Base complexity for every function

	// Walk the function body's subtree and count decision points
	var walkBody func(*sitter.Node)
	walkBody = func(node *sitter.Node) {
		if node == nil {
			return
		}

		nodeType := node.Type()

		// +1 for if statements
		if nodeType == "if_statement" {
			complexity++
		}
		// +1 for for and range-for loops
		if nodeType == "for_statement" || nodeType == "for_range_loop" {
			complexity++
		}
		// +1 for while and do-while loops
		if nodeType == "while_statement" || nodeType == "do_statement" {
			complexity++
		}
		// +1 for case statements (but not default)
		if nodeType == "case_statement" {
			if node.ChildByFieldName("value") != nil {
				complexity++
			}
		}
		// +1 for catch clauses
		if nodeType == "catch_clause" {
			complexity++
		}
		// +1 for ternary operator
		if nodeType == "conditional_expression" {
			complexity++
		}
		// +1 for logical operators && and ||
		if nodeType == "binary_expression" {
			if operatorNode := node.ChildByFieldName("operator"); operatorNode != nil {
				opText := string(source[operatorNode.StartByte() : operatorNode.EndByte()])
				if opText == "&&" || opText == "||" {
					complexity++
				}
			}
		}

		// Recurse into children (do NOT specially skip nested function_definitions—their code counts toward enclosing function)
		for i := uint32(0); i < node.ChildCount(); i++ {
			walkBody(node.Child(int(i)))
		}
	}

	// Find the function body
	for i := uint32(0); i < funcNode.ChildCount(); i++ {
		child := funcNode.Child(int(i))
		if child.Type() == "compound_statement" {
			walkBody(child)
			break
		}
	}

	return complexity
}

// scanFileForCppComplexity parses a C++ file and evaluates complexity of every function
func scanFileForCppComplexity(filePath string, maxComplexity int, opts Options) (violations []Violation, excludedFuncs []ExcludedFunc, skipped *SkippedFile) {
	src, err := srcfiles.ReadFile(filePath)
	if err != nil {
		return nil, nil, &SkippedFile{File: filePath, Error: err.Error()}
	}

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(cpp.GetLanguage())

	tree := parser.Parse(nil, src)
	defer tree.Close()

	rootNode := tree.RootNode()
	if hasErrorNode(rootNode) {
		return nil, nil, &SkippedFile{File: filePath, Error: "parse error"}
	}

	var functions []funcDef
	findFunctionDefinitions(rootNode, src, &functions)

	for _, fn := range functions {
		complexity := cyclomaticComplexity(fn.node, src)

		// Check if excluded
		fnName := fn.name
		excluded := false
		var reason string
		for _, pattern := range opts.ExcludeFuncs {
			if matchFunctionGlob(fnName, pattern) {
				excluded = true
				reason = "matched exclude pattern: " + pattern
				break
			}
		}

		if excluded {
			if opts.Debug {
				excludedFuncs = append(excludedFuncs, ExcludedFunc{
					File:   filePath,
					Line:   fn.startLine,
					Func:   fnName,
					Reason: reason,
				})
			}
			continue
		}

		if complexity > maxComplexity {
			violations = append(violations, Violation{
				File:       filePath,
				Line:       fn.startLine,
				Func:       fnName,
				Complexity: complexity,
				Limit:      maxComplexity,
			})
		}
	}

	return violations, excludedFuncs, nil
}

// matchFunctionGlob checks if a function name matches a glob pattern
// (simplified glob matching — mirrors cppfunclen's approach)
func matchFunctionGlob(funcName, pattern string) bool {
	if strings.Contains(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(funcName, prefix)
	}
	return funcName == pattern
}

func Check(paths []string, maxComplexity int, opts Options) (Report, error) {
	assertutil.Assertf(maxComplexity > 0, "maxComplexity must be positive, got %d", maxComplexity)

	report := Report{
		Violations:    []Violation{},
		Skipped:       []SkippedFile{},
		ExcludedFuncs: []ExcludedFunc{},
	}

	filesToCheck, _, skipped := srcfiles.Collect(paths, []string{".cpp", ".cc", ".cxx", ".c++", ".hpp", ".h"}, opts.ExcludeFiles)
	report.Skipped = append(report.Skipped, skipped...)

	for _, filePath := range filesToCheck {
		violations, excludedFuncs, skippedFile := scanFileForCppComplexity(filePath, maxComplexity, opts)
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
