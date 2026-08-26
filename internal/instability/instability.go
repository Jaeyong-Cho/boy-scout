package instability

import (
	"boy-scout/internal/assertutil"
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
	Violations            []Violation
	Skipped               []SkippedFile
	TotalEdges            int
	ViolationRate         float64 // (# edges with Gap > 0) / total edges
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

func Check(paths []string, minGap float64, opts Options) (Report, error) {
	assertutil.Assertf(minGap >= 0, "minGap must be non-negative, got %f", minGap)

	graph, err := BuildGraph(paths, opts)
	if err != nil {
		return Report{}, err
	}

	violations, totalGap := ComputeViolations(graph, minGap)
	violationRate, weightedViolationRate := ComputeViolationRates(graph, violations, totalGap)

	return Report{
		Violations:            violations,
		Skipped:               graph.Skipped,
		TotalEdges:            len(graph.Edges),
		ViolationRate:         violationRate,
		WeightedViolationRate: weightedViolationRate,
	}, nil
}

// ComputeViolations finds all violations where an edge's gap exceeds minGap.
func ComputeViolations(graph Graph, minGap float64) ([]Violation, float64) {
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
			assertutil.Assertf(gap > minGap, "appended violation has Gap > minGap")
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

// ComputeViolationRates calculates violation rate metrics.
func ComputeViolationRates(graph Graph, violations []Violation, totalGap float64) (float64, float64) {
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
