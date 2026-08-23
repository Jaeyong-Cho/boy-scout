package instability

import (
	"bufio"
	"fmt"
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

	// If path is a file, start from its directory
	stat, err := os.Stat(absPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to stat path: %w", err)
	}
	if !stat.IsDir() {
		absPath = filepath.Dir(absPath)
	}

	// Walk upward looking for go.mod
	for {
		gomodPath := filepath.Join(absPath, "go.mod")
		if _, err := os.Stat(gomodPath); err == nil {
			// Found go.mod, now read the module name
			file, err := os.Open(gomodPath)
			if err != nil {
				return "", "", fmt.Errorf("failed to open go.mod: %w", err)
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
						return absPath, moduleName, nil
					}
				}
			}

			return "", "", fmt.Errorf("go.mod found but module line not parsed")
		}

		// Move to parent directory
		parent := filepath.Dir(absPath)
		if parent == absPath {
			// Reached filesystem root
			return "", "", fmt.Errorf("no go.mod found in path or any parent directory")
		}
		absPath = parent
	}
}

// BuildGraph constructs the complete package-import graph for a module.
// Only packages that appear in at least one edge are included in the result.
func BuildGraph(paths []string, opts Options) (Graph, error) {
	graph := Graph{
		Packages: make(map[string]PackageStats),
		Edges:    []Edge{},
		Skipped:  []SkippedFile{},
	}

	// Determine module root and module name
	if len(paths) == 0 {
		paths = []string{"."}
	}

	root, moduleName, err := findModuleRoot(paths[0])
	if err != nil {
		return graph, err
	}

	graph.ModuleName = moduleName
	graph.Root = root

	// Collect all .go files
	filesToCheck, _, skipped := srcfiles.Collect(paths, []string{".go"}, opts.ExcludeFiles)
	graph.Skipped = append(graph.Skipped, skipped...)

	// Parse each file and collect imports, grouping by directory
	// Map from package import path to set of imports it makes (as strings)
	packageImports := make(map[string]map[string]bool)
	// Map from package import path to list of files in that package
	packageFiles := make(map[string][]string)
	// Map from directory to package import path
	dirToPackage := make(map[string]string)

	for _, filePath := range filesToCheck {
		if strings.HasSuffix(filePath, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filePath, nil, 0)
		if err != nil {
			graph.Skipped = append(graph.Skipped, SkippedFile{File: filePath, Error: err.Error()})
			continue
		}

		// Determine package directory and its import path
		dir := filepath.Dir(filePath)
		absDir, err := filepath.Abs(dir)
		assertf(err == nil, "filepath.Abs failed for dir %q: %v", dir, err)
		relDir, err := filepath.Rel(root, absDir)
		assertf(err == nil, "filepath.Rel failed for root %q dir %q: %v", root, absDir, err)
		pkgImportPath := moduleName
		if relDir != "." {
			pkgImportPath = moduleName + "/" + filepath.ToSlash(relDir)
		}
		dirToPackage[dir] = pkgImportPath

		// Initialize package imports map if needed
		if packageImports[pkgImportPath] == nil {
			packageImports[pkgImportPath] = make(map[string]bool)
		}

		packageFiles[pkgImportPath] = append(packageFiles[pkgImportPath], filePath)

		// Extract imports
		for _, spec := range file.Imports {
			importPath := strings.Trim(spec.Path.Value, `"`)
			// Only count first-party imports (those under the module path)
			if strings.HasPrefix(importPath, moduleName+"/") || importPath == moduleName {
				packageImports[pkgImportPath][importPath] = true
			}
		}
	}

	// Build edges and compute Ca/Ce
	type edge struct {
		source string
		target string
	}
	edgeSet := make(map[edge]bool)
	caferent := make(map[string]int) // Ca - afferent coupling
	cefferent := make(map[string]int) // Ce - efferent coupling

	for pkg, imports := range packageImports {
		cefferent[pkg] = len(imports)
		for importedPkg := range imports {
			if importedPkg != pkg {
				edgeSet[edge{pkg, importedPkg}] = true
				caferent[importedPkg]++
			}
		}
	}

	// Build the result graph: only include packages that appear in edges
	for e := range edgeSet {
		ca := caferent[e.source]
		ce := cefferent[e.source]
		assertf(ca+ce > 0, "Ca+Ce > 0 for package in edge")
		instability := float64(ce) / float64(ca+ce)

		if _, exists := graph.Packages[e.source]; !exists {
			graph.Packages[e.source] = PackageStats{
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

		if _, exists := graph.Packages[e.target]; !exists {
			graph.Packages[e.target] = PackageStats{
				Ca:          ca_b,
				Ce:          ce_b,
				Instability: instability_b,
				Files:       packageFiles[e.target],
			}
		}

		graph.Edges = append(graph.Edges, Edge{Source: e.source, Target: e.target})
	}

	return graph, nil
}

func Check(paths []string, minGap float64, opts Options) (Report, error) {
	assertf(minGap >= 0, "minGap must be non-negative, got %f", minGap)

	report := Report{
		Violations:            []Violation{},
		Skipped:               []SkippedFile{},
		TotalEdges:            0,
		ViolationRate:         0,
		WeightedViolationRate: 0,
	}

	// Build the graph
	graph, err := BuildGraph(paths, opts)
	if err != nil {
		return report, err
	}

	report.Skipped = graph.Skipped
	report.TotalEdges = len(graph.Edges)

	// Compute violations
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

	report.Violations = violations

	// Compute violation rates
	if report.TotalEdges > 0 {
		violationCount := 0
		for _, e := range graph.Edges {
			source := graph.Packages[e.Source]
			target := graph.Packages[e.Target]
			if target.Instability > source.Instability {
				violationCount++
			}
		}
		report.ViolationRate = float64(violationCount) / float64(report.TotalEdges)
		report.WeightedViolationRate = totalGap / float64(report.TotalEdges)
	}

	return report, nil
}
