package cppduplication

import (
	"fmt"
	"path/filepath"
	"strings"

	"boy-scout/internal/assertutil"
	"boy-scout/internal/duplication"
	"boy-scout/internal/srcfiles"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/cpp"
)

type Options struct {
	ExcludeFiles []string
	ExcludeFuncs []string
	Debug        bool
}

// Report is a type alias for duplication.Report (shapes are identical)
type Report = duplication.Report

type ExcludedFunc = duplication.ExcludedFunc
type SkippedFile = srcfiles.SkippedFile

// funcInfo holds parsed function metadata for C++ duplication comparison (cpp-specific)
type funcInfo struct {
	file         string
	line         int
	endLine      int
	name         string
	rawSequence  []string
	blindSequence []string
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

// funcDef temporarily holds parsed function definition info
type funcDef struct {
	name      string
	startLine int
	endLine   int
	bodyNode  *sitter.Node // retained for tokenization
}

// findFunctionDefinitions recursively finds all top-level function_definition nodes
// ponytail: 3rd copy of this walk (cppfunclen, cppcomplexity, cppduplication) — extract to internal/cppfuncwalk if a 4th consumer needs it.
func findFunctionDefinitions(node *sitter.Node, source []byte, results *[]funcDef) {
	if node == nil {
		return
	}
	if node.Type() == "function_definition" {
		def := extractFunctionDef(node, source)
		if def != nil {
			*results = append(*results, *def)
		}
		return
	}
	for i := uint32(0); i < node.ChildCount(); i++ {
		findFunctionDefinitions(node.Child(int(i)), source, results)
	}
}

// extractFunctionDef extracts the function name, body span, and bodyNode from a function_definition node
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
	endLine := int(bodyNode.EndPoint().Row) + 1

	return &funcDef{
		name:      name,
		startLine: startLine,
		endLine:   endLine,
		bodyNode:  bodyNode,
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

// declaratorFallbackName extracts the declarator text before the parameter list
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

// cppTokenSequence recursively walks a C++ function body node and extracts token sequences.
// Returns raw (unchanged tokens) and blind (identifiers → positional aliases, literals → type placeholders).
// Skips comments entirely. Asserts len(raw) == len(blind) and len(raw) > 0.
func cppTokenSequence(bodyNode *sitter.Node, source []byte) (raw, blind []string) {
	identMap := make(map[string]string)
	nextID := 1

	walkTokens(bodyNode, source, &raw, &blind, identMap, &nextID)

	assertutil.Assertf(len(raw) == len(blind), "cppTokenSequence postcondition violated: len(raw) != len(blind)")
	assertutil.Assertf(len(raw) > 0, "cppTokenSequence postcondition violated: empty sequence for non-nil bodyNode")

	return raw, blind
}

// walkTokens recursively extracts tokens from a node, handling identifiers and literals
func walkTokens(node *sitter.Node, source []byte, raw, blind *[]string, identMap map[string]string, nextID *int) {
	if node == nil {
		return
	}

	// Skip comments entirely
	if node.Type() == "comment" {
		return
	}

	// Leaf node: emit token
	if node.ChildCount() == 0 {
		text := string(source[node.StartByte():node.EndByte()])
		nodeType := node.Type()

		*raw = append(*raw, text)

		// Map identifiers to positional aliases, literals to type placeholders
		if nodeType == "identifier" {
			if alias, exists := identMap[text]; exists {
				*blind = append(*blind, alias)
			} else {
				alias := fmt.Sprintf("ID%d", *nextID)
				identMap[text] = alias
				*nextID++
				*blind = append(*blind, alias)
			}
		} else if isLiteral(nodeType) {
			*blind = append(*blind, "LIT_"+nodeType)
		} else {
			// Keywords, operators, punctuation
			*blind = append(*blind, text)
		}
		return
	}

	// Recurse into children
	for i := uint32(0); i < node.ChildCount(); i++ {
		walkTokens(node.Child(int(i)), source, raw, blind, identMap, nextID)
	}
}

// isLiteral checks if a node type represents a literal value
func isLiteral(nodeType string) bool {
	switch nodeType {
	case "number_literal", "string_literal", "char_literal", "true", "false", "null", "nullptr":
		return true
	default:
		return false
	}
}

// scanFileForCppDuplication parses a C++ file and extracts eligible functions
func scanFileForCppDuplication(filePath string, minLines int, opts Options) ([]funcInfo, []ExcludedFunc, *SkippedFile) {
	source, err := srcfiles.ReadFile(filePath)
	if err != nil {
		return nil, nil, &SkippedFile{File: filePath, Error: err.Error()}
	}

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(cpp.GetLanguage())

	tree := parser.Parse(nil, source)
	defer tree.Close()

	root := tree.RootNode()

	if hasErrorNode(root) {
		return nil, nil, &SkippedFile{File: filePath, Error: "parse error: ERROR node found in tree"}
	}

	var funcDefs []funcDef
	findFunctionDefinitions(root, source, &funcDefs)

	var allFuncs []funcInfo
	var excludedFuncs []ExcludedFunc

	for _, fn := range funcDefs {
		if matchesExcludeFunc(fn.name, opts.ExcludeFuncs) {
			if opts.Debug {
				excludedFuncs = append(excludedFuncs, ExcludedFunc{
					File:   filePath,
					Line:   fn.startLine,
					Func:   fn.name,
					Reason: "flag",
				})
			}
			continue
		}

		length := fn.endLine - fn.startLine + 1
		if length < minLines {
			continue
		}

		raw, blind := cppTokenSequence(fn.bodyNode, source)
		allFuncs = append(allFuncs, funcInfo{
			file:          filePath,
			line:          fn.startLine,
			endLine:       fn.endLine,
			name:          fn.name,
			rawSequence:   raw,
			blindSequence: blind,
		})
	}

	return allFuncs, excludedFuncs, nil
}

// reportDuplicates compares all function pairs and builds violation list
func reportDuplicates(allFuncs []funcInfo, minSimilarity float64) []duplication.Violation {
	var violations []duplication.Violation
	seen := make(map[string]bool)

	for i := 0; i < len(allFuncs); i++ {
		for j := i + 1; j < len(allFuncs); j++ {
			a := &allFuncs[i]
			b := &allFuncs[j]

			cloneType, similarity := duplication.ClassifyPair(a.rawSequence, a.blindSequence, b.rawSequence, b.blindSequence, minSimilarity)
			if cloneType == "" {
				continue
			}

			var first, second *funcInfo
			if a.file < b.file || (a.file == b.file && a.line < b.line) {
				first, second = a, b
			} else {
				first, second = b, a
			}

			key := fmt.Sprintf("%s:%d:%s|%s:%d:%s", first.file, first.line, first.name, second.file, second.line, second.name)
			if seen[key] {
				continue
			}
			seen[key] = true

			dupLines := first.endLine - first.line + 1

			violation := duplication.Violation{
				FileA:      first.file,
				LineA:      first.line,
				FuncA:      first.name,
				FileB:      second.file,
				LineB:      second.line,
				FuncB:      second.name,
				Type:       cloneType,
				DupLines:   dupLines,
				Similarity: similarity,
			}
			violations = append(violations, violation)
		}
	}
	return violations
}

// CheckWithSimilarity scans C++ files and reports function duplicates with an LCS-based similarity threshold
func CheckWithSimilarity(paths []string, minLines int, minSimilarity float64, opts Options) (Report, error) {
	assertutil.Assertf(minLines > 0, "minLines must be positive, got %d", minLines)

	report := Report{
		Violations:    []duplication.Violation{},
		Skipped:       []srcfiles.SkippedFile{},
		ExcludedFiles: []string{},
		ExcludedFuncs: []ExcludedFunc{},
	}

	filesToCheck, excludedFiles, skipped := srcfiles.Collect(paths, []string{".cpp", ".h", ".hpp"}, opts.ExcludeFiles)
	report.Skipped = append(report.Skipped, skipped...)
	if opts.Debug {
		report.ExcludedFiles = append(report.ExcludedFiles, excludedFiles...)
	}

	var allFuncs []funcInfo
	for _, filePath := range filesToCheck {
		funcs, excludedFuncs, skippedFile := scanFileForCppDuplication(filePath, minLines, opts)
		if skippedFile != nil {
			report.Skipped = append(report.Skipped, *skippedFile)
			continue
		}
		allFuncs = append(allFuncs, funcs...)
		if opts.Debug {
			report.ExcludedFuncs = append(report.ExcludedFuncs, excludedFuncs...)
		}
	}

	report.Violations = reportDuplicates(allFuncs, minSimilarity)
	report.Clusters = duplication.BuildClusters(report.Violations)

	return report, nil
}

// Check scans C++ files and reports function duplicates with default 0.70 similarity threshold
func Check(paths []string, minLines int, opts Options) (Report, error) {
	return CheckWithSimilarity(paths, minLines, 0.70, opts)
}
