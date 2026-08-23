package abstractness

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"boy-scout/internal/instability"
	"boy-scout/internal/srcfiles"
)

// SkippedFile is a type alias for srcfiles.SkippedFile.
type SkippedFile = srcfiles.SkippedFile

type PackageDiagnosis struct {
	ImportPath   string  // the package being diagnosed
	Abstractness float64 // A: ratio of exported interfaces to (interfaces + structs)
	Instability  float64 // I: from the graph
	Distance     float64 // |A + I - 1|
	Zone         string  // "Pain" or "Uselessness"
	SurfaceRatio float64 // ratio of exported declarations to total declarations (for Pain candidates; 0 for Uselessness)
}

type Options struct {
	ExcludeFiles         []string
	ExcludeFuncs         []string // unused
	Debug                bool
	IgnoreDeepModuleGate bool // when true, flag all Pain candidates regardless of SurfaceRatio
}

type Report struct {
	Violations    []PackageDiagnosis
	Skipped       []SkippedFile
	TotalPackages int
}

func assertf(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}

// surfaceRatio computes (# exported top-level declarations) / (# total top-level declarations).
// Top-level declarations counted: FuncDecl (including methods) and each spec in GenDecl.
// Returns the ratio and any parse errors encountered.
// Pre: files must be non-empty and contain at least one computable declaration (asserted).
func surfaceRatio(files []string) (ratio float64, err error) {
	var exportedCount, totalCount int

	for _, filePath := range files {
		exp, tot := countDeclarationsInFile(filePath)
		exportedCount += exp
		totalCount += tot
	}

	assertf(totalCount > 0, "surfaceRatio called on package with 0 declarations; this should not happen for a Zone-of-Pain candidate")
	return float64(exportedCount) / float64(totalCount), nil
}

// countDeclarationsInFile counts exported and total top-level declarations in a file.
func countDeclarationsInFile(filePath string) (exported, total int) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return 0, 0 // skip unparsable files
	}

	for _, decl := range file.Decls {
		e, t := processDeclForCount(decl)
		exported += e
		total += t
	}

	return exported, total
}

// processDeclForCount extracts counts from a single declaration.
func processDeclForCount(decl ast.Decl) (exported, total int) {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		total = 1
		if ast.IsExported(d.Name.Name) {
			exported = 1
		}
	case *ast.GenDecl:
		if isCountableGenDecl(d) {
			exported, total = addSpecCounts(d.Specs, 0, 0)
		}
	}
	return exported, total
}

// addSpecCounts accumulates counts from GenDecl specs.
func addSpecCounts(specs []ast.Spec, exported, total int) (int, int) {
	for _, spec := range specs {
		total++
		if isExportedSpec(spec) {
			exported++
		}
	}
	return exported, total
}

// isCountableGenDecl returns true if the GenDecl should have its specs counted.
func isCountableGenDecl(d *ast.GenDecl) bool {
	return d.Tok == token.TYPE || d.Tok == token.VAR || d.Tok == token.CONST
}

// isExportedSpec checks if a spec is exported by extracting its name.
func isExportedSpec(spec ast.Spec) bool {
	var name string
	switch s := spec.(type) {
	case *ast.TypeSpec:
		name = s.Name.Name
	case *ast.ValueSpec:
		if len(s.Names) > 0 {
			name = s.Names[0].Name
		}
	}
	return name != "" && ast.IsExported(name)
}

// countExportedTypes scans the given .go files and returns the count of exported
// interface and struct type declarations. Returns (interfaces, structs).
func countExportedTypes(files []string) (interfaces, structs int, skipped []SkippedFile) {
	for _, filePath := range files {
		iCnt, sCnt, fileSkipped, err := countTypesInFile(filePath)
		if err != nil {
			skipped = append(skipped, SkippedFile{File: filePath, Error: err.Error()})
			continue
		}
		interfaces += iCnt
		structs += sCnt
		skipped = append(skipped, fileSkipped...)
	}

	return interfaces, structs, skipped
}

// countTypesInFile counts exported interface and struct types in a file.
func countTypesInFile(filePath string) (int, int, []SkippedFile, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return 0, 0, nil, err
	}

	var interfaces, structs int
	for _, decl := range file.Decls {
		// Only process type declarations
		if g, ok := decl.(*ast.GenDecl); ok && g.Tok == token.TYPE {
			for _, spec := range g.Specs {
				// Only process type specs (not import specs)
				if ts, ok := spec.(*ast.TypeSpec); ok {
					iCnt, sCnt := countTypeKinds(ts)
					interfaces += iCnt
					structs += sCnt
				}
			}
		}
	}

	return interfaces, structs, nil, nil
}

// countTypeKinds counts interface and struct types in a TypeSpec if exported.
func countTypeKinds(typeSpec *ast.TypeSpec) (int, int) {
	// Only count exported (capitalized) names
	if !typeSpec.Name.IsExported() {
		return 0, 0
	}

	// Check if it's an interface or struct type
	switch typeSpec.Type.(type) {
	case *ast.InterfaceType:
		return 1, 0
	case *ast.StructType:
		return 0, 1
	}

	// Ignore other type kinds (aliases, func types, etc.)
	return 0, 0
}

// Check analyzes the abstractness and instability of packages in the given paths.
func Check(paths []string, minDistance float64, opts Options) (Report, error) {
	return CheckWithSurfaceRatio(paths, minDistance, 0.5, opts)
}

// CheckWithSurfaceRatio analyzes abstractness/instability and gates Pain candidates by SurfaceRatio.
func CheckWithSurfaceRatio(paths []string, minDistance float64, minSurfaceRatio float64, opts Options) (Report, error) {
	assertf(minDistance >= 0, "minDistance must be non-negative, got %f", minDistance)
	assertf(minSurfaceRatio >= 0, "minSurfaceRatio must be non-negative, got %f", minSurfaceRatio)

	graph, err := instability.BuildGraph(paths, instability.Options{
		ExcludeFiles: opts.ExcludeFiles,
		ExcludeFuncs: opts.ExcludeFuncs,
		Debug:        opts.Debug,
	})
	if err != nil {
		return Report{}, err
	}

	violations := []PackageDiagnosis{}
	skipped := append([]SkippedFile{}, graph.Skipped...)

	for importPath, pkgStats := range graph.Packages {
		diag, skipErr := diagnosePackage(importPath, pkgStats, minDistance, minSurfaceRatio, opts.IgnoreDeepModuleGate)
		if skipErr != nil {
			skipped = append(skipped, skipErr...)
		}
		if diag != nil {
			violations = append(violations, *diag)
		}
	}

	return Report{
		Violations:    violations,
		Skipped:       skipped,
		TotalPackages: len(graph.Packages),
	}, nil
}

// diagnosePackage analyzes a single package for abstractness/instability violations.
// Returns nil diagnosis if the package is compliant or has parse errors.
func diagnosePackage(importPath string, pkgStats instability.PackageStats, minDistance, minSurfaceRatio float64, ignoreDeepModuleGate bool) (*PackageDiagnosis, []SkippedFile) {
	interfaces, structs, skipped := countExportedTypes(pkgStats.Files)
	if len(skipped) > 0 {
		return nil, skipped
	}

	abstractness := computeAbstractness(interfaces, structs)
	signedD := abstractness + pkgStats.Instability - 1.0
	distance := absFloat64(signedD)

	zone, surfaceRatioVal, shouldReport := determineZone(signedD, minDistance, minSurfaceRatio, ignoreDeepModuleGate, pkgStats.Files)
	if !shouldReport {
		return nil, nil
	}

	return &PackageDiagnosis{
		ImportPath:   importPath,
		Abstractness: abstractness,
		Instability:  pkgStats.Instability,
		Distance:     distance,
		Zone:         zone,
		SurfaceRatio: surfaceRatioVal,
	}, nil
}

// computeAbstractness returns the ratio of exported interfaces to (interfaces + structs).
func computeAbstractness(interfaces, structs int) float64 {
	total := interfaces + structs
	if total == 0 {
		assertf(total == 0, "zero exported types implies A = 0, no division")
		return 0.0
	}
	return float64(interfaces) / float64(total)
}

// absFloat64 returns the absolute value of x.
func absFloat64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// determineZone classifies a package as Pain/Uselessness and applies surface-ratio gating.
// Returns (zone, surfaceRatio, shouldReport).
func determineZone(signedD, minDistance, minSurfaceRatio float64, ignoreDeepModuleGate bool, files []string) (string, float64, bool) {
	isPain := signedD < -minDistance
	isUselessness := signedD > minDistance
	assertf(!(isPain && isUselessness), "package cannot be both Pain and Uselessness")

	if isPain {
		ratio, _ := surfaceRatio(files)
		if !ignoreDeepModuleGate && ratio < minSurfaceRatio {
			return "", 0, false
		}
		return "Pain", ratio, true
	}

	if isUselessness {
		// Uselessness rows always have SurfaceRatio=0
		return "Uselessness", 0, true
	}

	return "", 0, false
}
