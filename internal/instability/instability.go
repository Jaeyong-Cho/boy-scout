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

func Check(paths []string, minGap float64, opts Options) (Report, error) {
	assertf(minGap >= 0, "minGap must be non-negative, got %f", minGap)

	report := Report{
		Violations:            []Violation{},
		Skipped:               []SkippedFile{},
		TotalEdges:            0,
		ViolationRate:         0,
		WeightedViolationRate: 0,
	}

	// Determine module root and module name
	if len(paths) == 0 {
		paths = []string{"."}
	}

	root, moduleName, err := findModuleRoot(paths[0])
	if err != nil {
		return report, err
	}

	// Collect all .go files
	filesToCheck, _, skipped := srcfiles.Collect(paths, []string{".go"}, opts.ExcludeFiles)
	report.Skipped = append(report.Skipped, skipped...)

	// Parse each file and collect imports, grouping by directory
	// Map from package import path to set of imports it makes (as strings)
	packageImports := make(map[string]map[string]bool)
	// Map from directory to package import path
	dirToPackage := make(map[string]string)

	for _, filePath := range filesToCheck {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
		if err != nil {
			report.Skipped = append(report.Skipped, SkippedFile{File: filePath, Error: err.Error()})
			continue
		}

		// Determine package directory and its import path
		dir := filepath.Dir(filePath)
		relDir, _ := filepath.Rel(root, dir)
		pkgImportPath := moduleName
		if relDir != "." {
			pkgImportPath = moduleName + "/" + filepath.ToSlash(relDir)
		}
		dirToPackage[dir] = pkgImportPath

		// Initialize package imports map if needed
		if packageImports[pkgImportPath] == nil {
			packageImports[pkgImportPath] = make(map[string]bool)
		}

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

	report.TotalEdges = len(edgeSet)

	// Compute instability and violations
	violations := []Violation{}
	totalGap := 0.0

	for e := range edgeSet {
		ca := caferent[e.source]
		ce := cefferent[e.source]
		assertf(ca+ce > 0, "Ca+Ce > 0 for package in edge")
		i_a := float64(ce) / float64(ca+ce)

		ca_b := caferent[e.target]
		ce_b := cefferent[e.target]
		assertf(ca_b+ce_b > 0, "Ca+Ce > 0 for package in edge")
		i_b := float64(ce_b) / float64(ca_b+ce_b)

		gap := i_b - i_a
		if gap > 0 {
			totalGap += gap
		}

		if gap > minGap {
			assertf(gap > minGap, "appended violation has Gap > minGap")
			violations = append(violations, Violation{
				Source: e.source,
				Target: e.target,
				I_A:    i_a,
				I_B:    i_b,
				Gap:    gap,
			})
		}
	}

	report.Violations = violations

	// Compute violation rates
	if report.TotalEdges > 0 {
		violationCount := 0
		for e := range edgeSet {
			ca := caferent[e.source]
			ce := cefferent[e.source]
			i_a := float64(ce) / float64(ca+ce)
			ca_b := caferent[e.target]
			ce_b := cefferent[e.target]
			i_b := float64(ce_b) / float64(ca_b+ce_b)
			if i_b > i_a {
				violationCount++
			}
		}
		report.ViolationRate = float64(violationCount) / float64(report.TotalEdges)
		report.WeightedViolationRate = totalGap / float64(report.TotalEdges)
	}

	return report, nil
}
