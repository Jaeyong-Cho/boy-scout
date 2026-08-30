package tscomplexity

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"boy-scout/internal/assertutil"
)

// isFunctionLikeDeclaration checks if node is a function/generator/method declaration
func isFunctionLikeDeclaration(node *sitter.Node) bool {
	t := node.Type()
	return t == "function_declaration" || t == "generator_function_declaration" || t == "method_definition"
}

// isNamedArrowFunction checks if node is a named arrow function (arrow_function with variable_declarator parent)
func isNamedArrowFunction(node *sitter.Node) bool {
	if node.Type() != "arrow_function" {
		return false
	}
	parent := node.Parent()
	return parent != nil && parent.Type() == "variable_declarator"
}

// findChildByTypes returns the first child of node matching any of the given types.
func findChildByTypes(node *sitter.Node, types ...string) *sitter.Node {
	typeMap := make(map[string]bool)
	for _, t := range types {
		typeMap[t] = true
	}
	for i := uint32(0); i < node.ChildCount(); i++ {
		child := node.Child(int(i))
		if typeMap[child.Type()] {
			return child
		}
	}
	return nil
}

// extractFunctionDef extracts function name and node from function_declaration, generator_function_declaration, or method_definition
func extractFunctionDef(node *sitter.Node, source []byte) *funcDef {
	nodeType := node.Type()
	assertutil.Assertf(nodeType == "function_declaration" || nodeType == "generator_function_declaration" || nodeType == "method_definition",
		"extractFunctionDef called on non-function node: %s", nodeType)

	bodyNode := findChildByTypes(node, "statement_block")
	nameNode := findChildByTypes(node, "identifier", "property_identifier")

	assertutil.Assertf(bodyNode != nil, "function-like node without statement_block (body): %s", nodeType)

	name := "?"
	if nameNode != nil {
		name = strings.TrimSpace(string(source[nameNode.StartByte():nameNode.EndByte()]))
	}

	startLine := int(bodyNode.StartPoint().Row) + 1

	return &funcDef{
		name:      name,
		startLine: startLine,
		node:      node,
	}
}

// extractArrowFunctionDef extracts name and node from a named arrow function
func extractArrowFunctionDef(node *sitter.Node, source []byte) *funcDef {
	assertutil.Assertf(node.Type() == "arrow_function", "extractArrowFunctionDef called on non-arrow_function: %s", node.Type())

	parent := node.Parent()
	assertutil.Assertf(parent != nil && parent.Type() == "variable_declarator", "arrow_function not directly parented by variable_declarator")

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

	assertutil.Assertf(bodyNode != nil, "arrow_function without statement_block (body)")

	name := "?"
	if nameNode != nil {
		name = strings.TrimSpace(string(source[nameNode.StartByte():nameNode.EndByte()]))
	}

	startLine := int(bodyNode.StartPoint().Row) + 1

	return &funcDef{
		name:      name,
		startLine: startLine,
		node:      node,
	}
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
