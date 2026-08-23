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

type Violation struct {
	Source string  // import path of the package doing the importing
	Target string  // import path of the package being imported
	I_A    float64 // instability of Source
	I_B    float64 // instability of Target
	Gap    float64 // I_B - I_A
}

// SkippedFile is a type alias for srcfiles.SkippedFile, preserving the existing
// JSON field names and shape of the Violation output.
type SkippedFile = srcfiles.SkippedFile

type Options struct {
	ExcludeFiles []string
	ExcludeFuncs []string // unused (instability has no function-level concept)
	Debug        bool
}

type Report struct {
	Violations           []Violation
	Skipped              []SkippedFile
	TotalEdges           int
	ViolationRate        float64 // (# edges with Gap > 0) / total edges
	WeightedViolationRate float64 // (sum of max(0, Gap) over all edges) / total edges
}

// PackageStats holds the coupling metrics for a single package.
type PackageStats struct {
	Ca          int      // afferent coupling: number of packages importing this one
	Ce          int      // efferent coupling: number of packages this one imports
	Instability float64  // Ce / (Ca + Ce); only meaningful when Ca+Ce > 0
	Files       []string // absolute paths of .go files in this package's directory
}

// Edge represents a single import edge between two packages.
type Edge struct {
	Source string // import path of the package doing the importing
	Target string // import path of the package being imported
}

// internalEdge is used for intermediate edge computation.
type internalEdge struct {
	source string
	target string
}

// Graph holds the complete package-import graph for a module.
type Graph struct {
	ModuleName string
	Root       string
	Packages   map[string]PackageStats // import path -> stats; only packages that appear in an edge
	Edges      []Edge
	Skipped    []SkippedFile
}

func assertf(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}

// findModuleRoot walks upward from path looking for go.mod, returning (root, moduleName, error).
// Returns error if no go.mod is found by the filesystem root.
func findModuleRoot(path string) (root, moduleName string, err error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve path: %w", err)
	}

	stat, err := os.Stat(absPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to stat path: %w", err)
	}
	if !stat.IsDir() {
		absPath = filepath.Dir(absPath)
	}

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
				assertf(moduleName != "", "parsed module line is non-empty")
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
	assertf(err == nil, "filepath.Abs failed for dir %q: %v", dir, err)
	relDir, err := filepath.Rel(root, absDir)
	assertf(err == nil, "filepath.Rel failed for root %q dir %q: %v", root, absDir, err)
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
	assertf(ca+ce > 0, "Ca+Ce > 0 for package in edge")
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
	assertf(ca_b+ce_b > 0, "Ca+Ce > 0 for package in edge")
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

func Check(paths []string, minGap float64, opts Options) (Report, error) {
	assertf(minGap >= 0, "minGap must be non-negative, got %f", minGap)

	graph, err := BuildGraph(paths, opts)
	if err != nil {
		return Report{}, err
	}

	violations, totalGap := computeViolations(graph, minGap)
	violationRate, weightedViolationRate := computeViolationRates(graph, violations, totalGap)

	return Report{
		Violations:            violations,
		Skipped:               graph.Skipped,
		TotalEdges:            len(graph.Edges),
		ViolationRate:         violationRate,
		WeightedViolationRate: weightedViolationRate,
	}, nil
}

// computeViolations finds all violations where an edge's gap exceeds minGap.
func computeViolations(graph Graph, minGap float64) ([]Violation, float64) {
	violations := []Violation{}
	totalGap := 0.0

	for _, e := range graph.Edges {
		source := graph.Packages[e.Source]
		target := graph.Packages[e.Target]
		gap := target.Instability - source.Instability

		if gap > 0 {
			totalGap += gap
		}

		if gap > minGap {
			assertf(gap > minGap, "appended violation has Gap > minGap")
			violations = append(violations, Violation{
				Source: e.Source,
				Target: e.Target,
				I_A:    source.Instability,
				I_B:    target.Instability,
				Gap:    gap,
			})
		}
	}

	return violations, totalGap
}

// computeViolationRates calculates violation rate metrics.
func computeViolationRates(graph Graph, violations []Violation, totalGap float64) (float64, float64) {
	if len(graph.Edges) == 0 {
		return 0, 0
	}

	violationCount := 0
	for _, e := range graph.Edges {
		source := graph.Packages[e.Source]
		target := graph.Packages[e.Target]
		if target.Instability > source.Instability {
			violationCount++
		}
	}

	violationRate := float64(violationCount) / float64(len(graph.Edges))
	weightedViolationRate := totalGap / float64(len(graph.Edges))
	return violationRate, weightedViolationRate
}
