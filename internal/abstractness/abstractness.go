package abstractness

import (
	"fmt"

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
