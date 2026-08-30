// Package cohesion computes LCOM4, TCC, and LCC metrics for classes/structs.
package cohesion

import (
	"boy-scout/internal/assertutil"
)

// Method represents a class/struct method and the fields/methods it interacts with.
type Method struct {
	Name   string
	Fields map[string]bool // field names this method touches
	Calls  map[string]bool  // other method names this method calls
}

// Score holds the three cohesion metrics and their severity levels.
type Score struct {
	LCOM4     int
	LCOM4Level string
	TCC        float64
	TCCLevel   string
	LCC        float64
	LCCLevel   string
}

// Compute calculates LCOM4, TCC, and LCC for a set of methods.
// Precondition: len(methods) >= 2 (caller must filter).
func Compute(methods []Method) Score {
	assertutil.Assertf(len(methods) >= 2, "cohesion.Compute: need at least 2 methods, got %d", len(methods))

	n := len(methods)

	// Build fieldGraph: true if methods i and j share a field
	fieldGraph := make([][]bool, n)
	for i := range fieldGraph {
		fieldGraph[i] = make([]bool, n)
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			for field := range methods[i].Fields {
				if methods[j].Fields[field] {
					fieldGraph[i][j] = true
					fieldGraph[j][i] = true
					break
				}
			}
		}
	}

	// Build fullGraph: fieldGraph edges OR method-calls edges
	fullGraph := make([][]bool, n)
	for i := range fullGraph {
		fullGraph[i] = make([]bool, n)
		for j := range fullGraph[i] {
			fullGraph[i][j] = fieldGraph[i][j]
		}
	}
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i != j && (methods[i].Calls[methods[j].Name] || methods[j].Calls[methods[i].Name]) {
				fullGraph[i][j] = true
			}
		}
	}

	// Count connected components in fullGraph using union-find
	parent := make([]int, n)
	for i := 0; i < n; i++ {
		parent[i] = i
	}

	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if fullGraph[i][j] {
				pi, pj := find(i), find(j)
				if pi != pj {
					parent[pi] = pj
				}
			}
		}
	}

	components := make(map[int]bool)
	for i := 0; i < n; i++ {
		components[find(i)] = true
	}
	lcom4 := len(components)

	// Calculate TCC: fraction of method pairs sharing a field (upper triangle only)
	fieldPairs := 0
	totalPairs := n * (n - 1) / 2
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if fieldGraph[i][j] {
				fieldPairs++
			}
		}
	}
	tcc := 0.0
	if totalPairs > 0 {
		tcc = float64(fieldPairs) / float64(totalPairs)
	}

	// Calculate LCC: fraction of method pairs connected (directly or indirectly via any path)
	// Use fullGraph to count connected component sizes, then sum pairs within each component
	lccParent := make([]int, n)
	for i := 0; i < n; i++ {
		lccParent[i] = i
	}

	var findLCC func(int) int
	findLCC = func(x int) int {
		if lccParent[x] != x {
			lccParent[x] = findLCC(lccParent[x])
		}
		return lccParent[x]
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if fullGraph[i][j] {
				pi, pj := findLCC(i), findLCC(j)
				if pi != pj {
					lccParent[pi] = pj
				}
			}
		}
	}

	componentSizes := make(map[int]int)
	for i := 0; i < n; i++ {
		root := findLCC(i)
		componentSizes[root]++
	}

	lccPairs := 0
	for _, size := range componentSizes {
		lccPairs += size * (size - 1) / 2
	}
	lcc := 0.0
	if totalPairs > 0 {
		lcc = float64(lccPairs) / float64(totalPairs)
	}

	return Score{
		LCOM4:      lcom4,
		LCOM4Level: lcom4Level(lcom4),
		TCC:        tcc,
		TCCLevel:   ratioLevel(tcc),
		LCC:        lcc,
		LCCLevel:   ratioLevel(lcc),
	}
}

func lcom4Level(v int) string {
	if v == 1 {
		return "good"
	}
	if v == 2 {
		return "warning"
	}
	return "danger"
}

func ratioLevel(v float64) string {
	if v > 0.8 {
		return "good"
	}
	if v >= 0.5 {
		return "warning"
	}
	return "danger"
}

// Worst returns the most severe of the three levels: danger > warning > good.
func Worst(s Score) string {
	levels := []string{s.LCOM4Level, s.TCCLevel, s.LCCLevel}
	worst := "good"
	for _, level := range levels {
		if level == "danger" {
			worst = "danger"
		} else if level == "warning" && worst != "danger" {
			worst = "warning"
		}
	}
	assertutil.Assertf(worst != "", "cohesion: unclassified score %v", s)
	return worst
}
