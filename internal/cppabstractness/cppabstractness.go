package cppabstractness

import (
	"fmt"
	"os"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/cpp"

	"boy-scout/internal/abstractness"
	"boy-scout/internal/cppinstability"
	"boy-scout/internal/instability"
)

type Options struct {
	ExcludeFiles []string
	Debug        bool
}

// Check analyzes the abstractness and instability of C++ files.
// Returns abstractness.Report with C++ classes counted by pure virtual presence.
func Check(paths []string, minDistance float64, opts Options) (abstractness.Report, error) {
	if minDistance < 0 {
		return abstractness.Report{}, fmt.Errorf("minDistance must be non-negative, got %f", minDistance)
	}

	// Build the dependency graph using Story 1 (cppinstability)
	graph, err := cppinstability.BuildGraph(paths, cppinstability.Options{
		ExcludeFiles: opts.ExcludeFiles,
		Debug:        opts.Debug,
	})
	if err != nil {
		return abstractness.Report{}, err
	}

	violations := []abstractness.PackageDiagnosis{}
	skipped := append([]abstractness.SkippedFile{}, graph.Skipped...)

	// For each file (package in C++ terms), count abstract/concrete classes
	for filePath, pkgStats := range graph.Packages {
		if diag := diagnosisForFile(filePath, pkgStats, minDistance); diag != nil {
			violations = append(violations, *diag)
		}
	}

	return abstractness.Report{
		Violations:    violations,
		Skipped:       skipped,
		TotalPackages: len(graph.Packages),
	}, nil
}

// diagnosisForFile analyzes one file's abstractness and returns a violation if it's in a zone (Pain/Uselessness).
func diagnosisForFile(filePath string, pkgStats instability.PackageStats, minDistance float64) *abstractness.PackageDiagnosis {
	abstractCount, concreteCount := countClassesInFiles(pkgStats.Files)

	// Skip files with no classes (divide-by-zero guard)
	if abstractCount+concreteCount == 0 {
		return nil
	}

	// Compute abstractness: ratio of abstract classes to total classes
	abstractnessVal := float64(abstractCount) / float64(abstractCount+concreteCount)

	// Apply the distance formula: signedD = A + I - 1
	signedD := abstractnessVal + pkgStats.Instability - 1.0
	distance := absFloat64(signedD)

	// Determine zone: Pain (signedD < -minDistance) or Uselessness (signedD > minDistance)
	if signedD < -minDistance {
		return &abstractness.PackageDiagnosis{
			ImportPath:   filePath,
			Abstractness: abstractnessVal,
			Instability:  pkgStats.Instability,
			Distance:     distance,
			Zone:         "Pain",
			SurfaceRatio: 0,
		}
	} else if signedD > minDistance {
		return &abstractness.PackageDiagnosis{
			ImportPath:   filePath,
			Abstractness: abstractnessVal,
			Instability:  pkgStats.Instability,
			Distance:     distance,
			Zone:         "Uselessness",
			SurfaceRatio: 0,
		}
	}
	return nil
}

// countClassesInFiles counts abstract and concrete classes in a list of files.
// A class is abstract if it has ≥1 pure virtual method (contains pure_virtual_clause).
func countClassesInFiles(files []string) (int, int) {
	abstractCount := 0
	concreteCount := 0

	parser := sitter.NewParser()
	parser.SetLanguage(cpp.GetLanguage())

	for _, filePath := range files {
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		tree := parser.Parse(nil, content)
		treeRoot := tree.RootNode()

		// Walk the tree to find class_specifier nodes
		walkClassSpecifiers(treeRoot, &abstractCount, &concreteCount)
	}

	return abstractCount, concreteCount
}

// walkClassSpecifiers recursively finds class_specifier nodes and classifies them.
func walkClassSpecifiers(node *sitter.Node, abstractCount, concreteCount *int) {
	if node.Type() == "class_specifier" || node.Type() == "struct_specifier" {
		if hasAnyPureVirtual(node) {
			*abstractCount++
		} else {
			*concreteCount++
		}
	}

	for i := uint32(0); i < node.ChildCount(); i++ {
		child := node.Child(int(i))
		walkClassSpecifiers(child, abstractCount, concreteCount)
	}
}

// hasAnyPureVirtual checks if a class_specifier contains any pure_virtual_clause.
func hasAnyPureVirtual(node *sitter.Node) bool {
	return hasPureVirtualChild(node)
}

// hasPureVirtualChild recursively searches for pure_virtual_clause nodes.
func hasPureVirtualChild(node *sitter.Node) bool {
	if node.Type() == "pure_virtual_clause" {
		return true
	}

	for i := uint32(0); i < node.ChildCount(); i++ {
		child := node.Child(int(i))
		if hasPureVirtualChild(child) {
			return true
		}
	}

	return false
}

// absFloat64 returns the absolute value of x.
func absFloat64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
