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
}

type Options struct {
	ExcludeFiles []string
	ExcludeFuncs []string // unused
	Debug        bool
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

// countExportedTypes scans the given .go files and returns the count of exported
// interface and struct type declarations. Returns (interfaces, structs).
func countExportedTypes(files []string) (interfaces, structs int, skipped []SkippedFile) {
	for _, filePath := range files {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filePath, nil, 0) // full mode to get type bodies
		if err != nil {
			skipped = append(skipped, SkippedFile{File: filePath, Error: err.Error()})
			continue
		}

		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				// Only count exported (capitalized) names
				if !typeSpec.Name.IsExported() {
					continue
				}

				// Check if it's an interface or struct type
				switch typeSpec.Type.(type) {
				case *ast.InterfaceType:
					interfaces++
				case *ast.StructType:
					structs++
					// Ignore other type kinds (aliases, func types, etc.)
				}
			}
		}
	}

	return interfaces, structs, skipped
}

// Check analyzes the abstractness and instability of packages in the given paths.
func Check(paths []string, minDistance float64, opts Options) (Report, error) {
	assertf(minDistance >= 0, "minDistance must be non-negative, got %f", minDistance)

	report := Report{
		Violations:    []PackageDiagnosis{},
		Skipped:       []SkippedFile{},
		TotalPackages: 0,
	}

	// Build the graph using instability package
	graph, err := instability.BuildGraph(paths, instability.Options{
		ExcludeFiles: opts.ExcludeFiles,
		ExcludeFuncs: opts.ExcludeFuncs,
		Debug:        opts.Debug,
	})
	if err != nil {
		return report, err
	}

	report.Skipped = append(report.Skipped, graph.Skipped...)

	// For each package in the graph, compute abstractness and diagnosis
	for importPath, pkgStats := range graph.Packages {
		// Count exported types in this package
		interfaces, structs, skipped := countExportedTypes(pkgStats.Files)
		report.Skipped = append(report.Skipped, skipped...)

		// Skip this package's diagnosis if we couldn't parse its files
		if len(skipped) > 0 {
			continue
		}

		// Compute abstractness
		total := interfaces + structs
		var abstractness float64
		if total == 0 {
			// No division happens at 0/0; A = 0 for packages with no exported types
			assertf(total == 0, "zero exported types implies A = 0, no division")
			abstractness = 0.0
		} else {
			abstractness = float64(interfaces) / float64(total)
		}

		// Compute signed distance
		signedD := abstractness + pkgStats.Instability - 1.0
		distance := signedD
		if distance < 0 {
			distance = -distance
		}

		// Determine zone
		var zone string
		isPain := signedD < -minDistance
		isUselessness := signedD > minDistance

		assertf(!(isPain && isUselessness), "package cannot be both Pain and Uselessness")

		if isPain {
			zone = "Pain"
		} else if isUselessness {
			zone = "Uselessness"
		} else {
			// No violation
			continue
		}

		report.Violations = append(report.Violations, PackageDiagnosis{
			ImportPath:   importPath,
			Abstractness: abstractness,
			Instability:  pkgStats.Instability,
			Distance:     distance,
			Zone:         zone,
		})
	}

	// Count total packages analyzed (those with computable instability = those in graph)
	report.TotalPackages = len(graph.Packages)

	return report, nil
}
