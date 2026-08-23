package cppinstability

import (
	"fmt"
	"os"
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/cpp"

	"boy-scout/internal/instability"
	"boy-scout/internal/srcfiles"
)

type Options struct {
	ExcludeFiles []string
	Debug        bool
}

// BuildGraph constructs the complete C++ file-import graph.
// Package unit = one source file (.cpp, .h, .hpp).
// Only quoted, in-project #include directives count as edges.
func BuildGraph(paths []string, opts Options) (instability.Graph, error) {
	graph := instability.Graph{
		Packages: make(map[string]instability.PackageStats),
		Edges:    []instability.Edge{},
		Skipped:  []srcfiles.SkippedFile{},
	}

	if len(paths) == 0 {
		paths = []string{"."}
	}

	// Normalize root path
	root, err := normalizeRootPath(paths[0])
	if err != nil {
		return graph, err
	}

	graph.Root = root
	graph.ModuleName = filepath.Base(root) // Use directory name as module name for C++

	// Collect .cpp, .h, .hpp files
	filesToCheck, _, skipped := srcfiles.Collect(paths, []string{".cpp", ".h", ".hpp"}, opts.ExcludeFiles)
	graph.Skipped = append(graph.Skipped, skipped...)

	// Parse files and collect includes
	packageImports, packageFiles := collectCppIncludes(filesToCheck, root)

	// Compute edges and coupling metrics using exported instability functions
	edges, caferent, cefferent := instability.ComputeEdgesAndCoupling(packageImports)

	for _, e := range edges {
		instability.AddPackageStatsForEdge(&graph, e, caferent, cefferent, packageFiles)
		graph.Edges = append(graph.Edges, e)
	}

	return graph, nil
}

// normalizeRootPath converts a path to an absolute directory path.
func normalizeRootPath(path string) (string, error) {
	root, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve root path: %w", err)
	}
	stat, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("failed to stat root: %w", err)
	}
	if !stat.IsDir() {
		root = filepath.Dir(root)
	}
	return root, nil
}

// Check runs the C++ instability check and returns a report.
func Check(paths []string, minGap float64, opts Options) (instability.Report, error) {
	if minGap < 0 {
		return instability.Report{}, fmt.Errorf("minGap must be non-negative, got %f", minGap)
	}

	graph, err := BuildGraph(paths, opts)
	if err != nil {
		return instability.Report{}, err
	}

	violations, totalGap := instability.ComputeViolations(graph, minGap)
	violationRate, weightedViolationRate := instability.ComputeViolationRates(graph, violations, totalGap)

	return instability.Report{
		Violations:            violations,
		Skipped:               graph.Skipped,
		TotalEdges:            len(graph.Edges),
		ViolationRate:         violationRate,
		WeightedViolationRate: weightedViolationRate,
	}, nil
}

// processFileForIncludes parses a single file and extracts its #include dependencies
func processFileForIncludes(filePath, root string, parser *sitter.Parser, packageImports map[string]map[string]bool, packageFiles map[string][]string) {
	// Normalize to absolute then relative path (per plan spec)
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return
	}
	relPath, err := filepath.Rel(root, absPath)
	if err != nil {
		return
	}

	// Read and parse file
	content, err := os.ReadFile(absPath)
	if err != nil {
		return
	}

	tree := parser.Parse(nil, content)
	treeRoot := tree.RootNode()

	// Initialize the package's import set
	if packageImports[relPath] == nil {
		packageImports[relPath] = make(map[string]bool)
	}
	packageFiles[relPath] = []string{absPath}

	// Extract #include directives
	extractIncludes(treeRoot, content, relPath, absPath, packageImports, root)
}

// collectCppIncludes parses C++ files and collects #include dependencies.
// Returns packageImports (file -> set of files it includes) and packageFiles (file -> [file]).
func collectCppIncludes(filesToCheck []string, root string) (map[string]map[string]bool, map[string][]string) {
	packageImports := make(map[string]map[string]bool)
	packageFiles := make(map[string][]string)

	parser := sitter.NewParser()
	parser.SetLanguage(cpp.GetLanguage())

	for _, filePath := range filesToCheck {
		processFileForIncludes(filePath, root, parser, packageImports, packageFiles)
	}

	return packageImports, packageFiles
}

// extractIncludes recursively walks the tree-sitter AST to find preproc_include nodes.
func extractIncludes(node *sitter.Node, sourceCode []byte, fileKey, filePath string, packageImports map[string]map[string]bool, projectRoot string) {
	if node.Type() == "preproc_include" {
		processInclude(node, sourceCode, fileKey, filePath, packageImports, projectRoot)
	}

	for i := uint32(0); i < node.ChildCount(); i++ {
		child := node.Child(int(i))
		extractIncludes(child, sourceCode, fileKey, filePath, packageImports, projectRoot)
	}
}

// processInclude extracts the include path from a preproc_include node.
// Only processes quoted includes (#include "..."), ignores angle-bracket includes (#include <...>).
func processInclude(node *sitter.Node, sourceCode []byte, fileKey, filePath string, packageImports map[string]map[string]bool, projectRoot string) {
	// preproc_include has 2 children: #include keyword and the path (string_literal or system_lib_string)
	if node.ChildCount() < 2 {
		return
	}

	pathNode := node.Child(1) // Second child is the path

	// Only process quoted includes (string_literal); ignore system_lib_string (<...>)
	if pathNode.Type() != "string_literal" {
		return
	}

	// Extract the include path from string_literal
	includePath := extractStringContent(pathNode, sourceCode)
	if includePath == "" {
		return
	}

	// Resolve and validate the include path
	relIncludePath, ok := resolveAndValidateInclude(filePath, includePath, projectRoot)
	if !ok {
		return
	}

	packageImports[fileKey][relIncludePath] = true
}

// extractStringContent extracts the string value from a string_literal node.
func extractStringContent(pathNode *sitter.Node, sourceCode []byte) string {
	// string_literal has children: ", string_content, "
	for i := uint32(0); i < pathNode.ChildCount(); i++ {
		child := pathNode.Child(int(i))
		if child.Type() == "string_content" {
			return string(child.Content(sourceCode))
		}
	}
	return ""
}

// resolveAndValidateInclude resolves an include path and checks if the file exists.
// Returns the relative path to project root and whether it was successful.
func resolveAndValidateInclude(filePath, includePath, projectRoot string) (string, bool) {
	// Resolve the include path relative to the including file's directory
	fileDir := filepath.Dir(filePath)
	resolvedPath := filepath.Join(fileDir, includePath)

	// Normalize to absolute then relative path (per plan spec)
	absIncludePath, err := filepath.Abs(resolvedPath)
	if err != nil {
		return "", false
	}

	// Make relative to project root
	relIncludePath, err := filepath.Rel(projectRoot, absIncludePath)
	if err != nil {
		return "", false
	}

	// Only count if the resolved path exists as a file
	if _, err := os.Stat(absIncludePath); err != nil {
		return "", false
	}

	return relIncludePath, true
}
