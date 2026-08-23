package instability

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"boy-scout/internal/srcfiles"
)

func assertf_graph(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}

// findModuleRoot walks upward from path looking for go.mod, returning (root, moduleName, error).
// Returns error if no go.mod is found by the filesystem root.
func findModuleRoot(path string) (root, moduleName string, err error) {
	absPath, err := normalizeAndValidatePath(path)
	if err != nil {
		return "", "", err
	}

	return searchForGoMod(absPath)
}

// normalizeAndValidatePath resolves path to an absolute directory path.
func normalizeAndValidatePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	stat, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat path: %w", err)
	}
	if !stat.IsDir() {
		absPath = filepath.Dir(absPath)
	}

	return absPath, nil
}

// searchForGoMod walks upward from absPath looking for go.mod.
func searchForGoMod(absPath string) (string, string, error) {
	for {
		gomodPath := filepath.Join(absPath, "go.mod")
		if _, err := os.Stat(gomodPath); err == nil {
			moduleName, err := readModuleName(gomodPath)
			if err != nil {
				return "", "", err
			}
			return absPath, moduleName, nil
		}

		parent := filepath.Dir(absPath)
		if parent == absPath {
			return "", "", fmt.Errorf("no go.mod found in path or any parent directory")
		}
		absPath = parent
	}
}

// readModuleName extracts the module name from go.mod.
func readModuleName(gomodPath string) (string, error) {
	file, err := os.Open(gomodPath)
	if err != nil {
		return "", fmt.Errorf("failed to open go.mod: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				moduleName := parts[1]
				assertf_graph(moduleName != "", "parsed module line is non-empty")
				return moduleName, nil
			}
		}
	}

	return "", fmt.Errorf("go.mod found but module line not parsed")
}

// BuildGraph constructs the complete package-import graph for a module.
// Only packages that appear in at least one edge are included in the result.
func BuildGraph(paths []string, opts Options) (Graph, error) {
	graph := Graph{
		Packages: make(map[string]PackageStats),
		Edges:    []Edge{},
		Skipped:  []SkippedFile{},
	}

	if len(paths) == 0 {
		paths = []string{"."}
	}

	root, moduleName, err := findModuleRoot(paths[0])
	if err != nil {
		return graph, err
	}

	graph.ModuleName = moduleName
	graph.Root = root

	filesToCheck, _, skipped := srcfiles.Collect(paths, []string{".go"}, opts.ExcludeFiles)
	graph.Skipped = append(graph.Skipped, skipped...)

	packageImports, packageFiles, var_skipped := collectPackageImports(filesToCheck, root, moduleName)
	graph.Skipped = append(graph.Skipped, var_skipped...)

	buildGraphEdges(&graph, packageImports, packageFiles)

	return graph, nil
}

// collectPackageImports parses files and groups imports by package.
func collectPackageImports(filesToCheck []string, root, moduleName string) (map[string]map[string]bool, map[string][]string, []SkippedFile) {
	packageImports := make(map[string]map[string]bool)
	packageFiles := make(map[string][]string)
	var skipped []SkippedFile

	for _, filePath := range filesToCheck {
		if strings.HasSuffix(filePath, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filePath, nil, 0)
		if err != nil {
			skipped = append(skipped, SkippedFile{File: filePath, Error: err.Error()})
			continue
		}

		pkgImportPath := resolvePackageImportPath(filePath, root, moduleName)
		ensurePackageImportsMap(packageImports, pkgImportPath)
		packageFiles[pkgImportPath] = append(packageFiles[pkgImportPath], filePath)
		recordImports(packageImports, pkgImportPath, file.Imports, moduleName)
	}

	return packageImports, packageFiles, skipped
}

// resolvePackageImportPath computes the import path for a file's package.
func resolvePackageImportPath(filePath, root, moduleName string) string {
	dir := filepath.Dir(filePath)
	absDir, err := filepath.Abs(dir)
	assertf_graph(err == nil, "filepath.Abs failed for dir %q: %v", dir, err)
	relDir, err := filepath.Rel(root, absDir)
	assertf_graph(err == nil, "filepath.Rel failed for root %q dir %q: %v", root, absDir, err)
	if relDir == "." {
		return moduleName
	}
	return moduleName + "/" + filepath.ToSlash(relDir)
}

// ensurePackageImportsMap initializes the imports map if needed.
func ensurePackageImportsMap(packageImports map[string]map[string]bool, pkgImportPath string) {
	if packageImports[pkgImportPath] == nil {
		packageImports[pkgImportPath] = make(map[string]bool)
	}
}

// recordImports extracts and records first-party imports from file specs.
func recordImports(packageImports map[string]map[string]bool, pkgImportPath string, specs []*ast.ImportSpec, moduleName string) {
	for _, spec := range specs {
		importPath := strings.Trim(spec.Path.Value, `"`)
		if strings.HasPrefix(importPath, moduleName+"/") || importPath == moduleName {
			packageImports[pkgImportPath][importPath] = true
		}
	}
}

// buildGraphEdges computes edges and coupling metrics from package imports.
func buildGraphEdges(g *Graph, packageImports map[string]map[string]bool, packageFiles map[string][]string) {
	edgeSet, caferent, cefferent := computeEdgesAndCoupling(packageImports)

	for e := range edgeSet {
		addPackageStatsForEdge(g, e, caferent, cefferent, packageFiles)
		g.Edges = append(g.Edges, Edge{Source: e.source, Target: e.target})
	}
}

// computeEdgesAndCoupling builds the edge set and computes coupling metrics.
func computeEdgesAndCoupling(packageImports map[string]map[string]bool) (map[internalEdge]bool, map[string]int, map[string]int) {
	edgeSet := make(map[internalEdge]bool)
	caferent := make(map[string]int)
	cefferent := make(map[string]int)

	for pkg, imports := range packageImports {
		cefferent[pkg] = len(imports)
		for importedPkg := range imports {
			if importedPkg != pkg {
				edgeSet[internalEdge{pkg, importedPkg}] = true
				caferent[importedPkg]++
			}
		}
	}

	return edgeSet, caferent, cefferent
}

// addPackageStatsForEdge adds package stats for both packages involved in an edge.
func addPackageStatsForEdge(g *Graph, e internalEdge, caferent, cefferent map[string]int, packageFiles map[string][]string) {
	ca := caferent[e.source]
	ce := cefferent[e.source]
	assertf_graph(ca+ce > 0, "Ca+Ce > 0 for package in edge")
	instability := float64(ce) / float64(ca+ce)

	if _, exists := g.Packages[e.source]; !exists {
		g.Packages[e.source] = PackageStats{
			Ca:          ca,
			Ce:          ce,
			Instability: instability,
			Files:       packageFiles[e.source],
		}
	}

	ca_b := caferent[e.target]
	ce_b := cefferent[e.target]
	assertf_graph(ca_b+ce_b > 0, "Ca+Ce > 0 for package in edge")
	instability_b := float64(ce_b) / float64(ca_b+ce_b)

	if _, exists := g.Packages[e.target]; !exists {
		g.Packages[e.target] = PackageStats{
			Ca:          ca_b,
			Ce:          ce_b,
			Instability: instability_b,
			Files:       packageFiles[e.target],
		}
	}
}
