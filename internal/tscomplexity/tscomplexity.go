package tscomplexity

import (
	"strings"

	"boy-scout/internal/assertutil"
	"boy-scout/internal/srcfiles"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
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

// getLanguageForFile returns the appropriate tree-sitter language for a .ts or .tsx file
func getLanguageForFile(filePath string) *sitter.Language {
	if strings.HasSuffix(filePath, ".tsx") {
		return tsx.GetLanguage()
	}
	return typescript.GetLanguage()
}

// cyclomaticComplexity walks a function's body and counts decision points
func cyclomaticComplexity(funcNode *sitter.Node, source []byte) int {
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
		// +1 for classic for loops
		if nodeType == "for_statement" {
			complexity++
		}
		// +1 for for...of and for...in loops (same node type)
		if nodeType == "for_in_statement" {
			complexity++
		}
		// +1 for while loops
		if nodeType == "while_statement" {
			complexity++
		}
		// +1 for do-while loops
		if nodeType == "do_statement" {
			complexity++
		}
		// +1 for case statements (but not default — switch_default has its own type)
		if nodeType == "switch_case" {
			complexity++
		}
		// +1 for catch clauses
		if nodeType == "catch_clause" {
			complexity++
		}
		// +1 for ternary operator
		if nodeType == "ternary_expression" {
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

		// Recurse into children (do NOT specially skip nested functions—their code counts toward enclosing function)
		for i := uint32(0); i < node.ChildCount(); i++ {
			walkBody(node.Child(int(i)))
		}
	}

	// Find the function body and walk it
	for i := uint32(0); i < funcNode.ChildCount(); i++ {
		child := funcNode.Child(int(i))
		if child.Type() == "statement_block" {
			walkBody(child)
			break
		}
	}

	assertutil.Assertf(complexity >= 1, "cyclomatic complexity must be at least 1, got %d", complexity)
	return complexity
}

// matchFunctionGlob checks if a function name matches a glob pattern
func matchFunctionGlob(funcName, pattern string) bool {
	if strings.Contains(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(funcName, prefix)
	}
	return funcName == pattern
}

// processFunctionsForComplexity checks each function in the list for excluded patterns
// and collects violations for those exceeding the complexity limit.
func processFunctionsForComplexity(filePath string, functions []funcDef, src []byte, maxComplexity int, opts Options) ([]Violation, []ExcludedFunc) {
	var violations []Violation
	var excludedFuncs []ExcludedFunc

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
			assertutil.Assertf(complexity > maxComplexity, "appended violation does not exceed limit %d", maxComplexity)
			violations = append(violations, Violation{
				File:       filePath,
				Line:       fn.startLine,
				Func:       fnName,
				Complexity: complexity,
				Limit:      maxComplexity,
			})
		}
	}

	return violations, excludedFuncs
}

// scanFileForComplexity parses a TypeScript file and evaluates complexity of every function
func scanFileForComplexity(filePath string, source []byte, maxComplexity int, opts Options) (violations []Violation, excludedFuncs []ExcludedFunc, skipped *SkippedFile) {
	parser := sitter.NewParser()
	defer parser.Close()

	lang := getLanguageForFile(filePath)
	parser.SetLanguage(lang)

	tree := parser.Parse(nil, source)
	defer tree.Close()

	rootNode := tree.RootNode()
	if hasErrorNode(rootNode) {
		return nil, nil, &SkippedFile{File: filePath, Error: "parse error"}
	}

	var functions []funcDef
	findFunctionDefinitions(rootNode, source, &functions)

	violations, excludedFuncs = processFunctionsForComplexity(filePath, functions, source, maxComplexity, opts)
	return violations, excludedFuncs, nil
}

func Check(paths []string, maxComplexity int, opts Options) (Report, error) {
	assertutil.Assertf(maxComplexity > 0, "maxComplexity must be positive, got %d", maxComplexity)

	report := Report{
		Violations:    []Violation{},
		Skipped:       []SkippedFile{},
		ExcludedFuncs: []ExcludedFunc{},
	}

	filesToCheck, _, skipped := srcfiles.Collect(paths, []string{".ts", ".tsx"}, opts.ExcludeFiles)
	report.Skipped = append(report.Skipped, skipped...)

	for _, filePath := range filesToCheck {
		source, err := srcfiles.ReadFile(filePath)
		if err != nil {
			report.Skipped = append(report.Skipped, SkippedFile{File: filePath, Error: err.Error()})
			continue
		}

		violations, excludedFuncs, skippedFile := scanFileForComplexity(filePath, source, maxComplexity, opts)
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
