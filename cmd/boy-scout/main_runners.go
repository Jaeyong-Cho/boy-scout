package main

import (
	"flag"
	"fmt"
	"io"

	"boy-scout/internal/abstractness"
	"boy-scout/internal/crap"
	"boy-scout/internal/duplication"
	"boy-scout/internal/filelen"
	"boy-scout/internal/gocomplexity"
	"boy-scout/internal/gofunclen"
	"boy-scout/internal/instability"
)

// CheckerConfig wraps the setup and execution of a checker command.
// Setup registers flags and returns a factory that accepts the debug flag
// (populated after flag parsing) and returns the actual checker function.
type CheckerConfig struct {
	Name         string
	Setup        func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error)
	JSONRenderer func(interface{}, io.Writer, io.Writer) int
	TextRenderer func(interface{}, io.Writer, io.Writer) int
}

// runCheck is the generic runner that all individual checkers delegate to.
// It handles flag parsing, error reporting, and output rendering.
func runCheck(cfg CheckerConfig, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet(cfg.Name, flag.ContinueOnError)
	format := fs.String("format", "text", "output format: text or json")
	excludeFile := fs.String("exclude-file", "", "comma-separated glob patterns for files to exclude")
	excludeFunc := fs.String("exclude-func", "", "comma-separated glob patterns for functions to exclude")
	debug := fs.Bool("debug", false, "enable debug output")

	// Let the config register its own flags and get back a factory.
	checkFnFactory := cfg.Setup(fs)

	// Parse flags (this populates *debug).
	paths, excludeFiles, excludeFuncs, err := resolveArgs(fs, args, excludeFile, excludeFunc)
	if err != nil {
		reportError(err, stderr)
		return 2
	}

	// Now create the actual checker function with the parsed debug flag.
	checkFn := checkFnFactory(*debug)

	report, err := checkFn(paths, excludeFiles, excludeFuncs)
	if err != nil {
		reportError(err, stderr)
		return 2
	}

	return selectAndRender(format,
		func(stdout, stderr io.Writer) int { return cfg.JSONRenderer(report, stdout, stderr) },
		func(stdout, stderr io.Writer) int { return cfg.TextRenderer(report, stdout, stderr) },
		stdout, stderr)
}

// ============ Go Checkers ============

var goFunclenCfg = CheckerConfig{
	Name: "gofunclen",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
		maxLines := fs.Int("max-lines", 50, "maximum function length in lines")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
				opts := gofunclen.Options{
					ExcludeFiles: excludeFiles,
					ExcludeFuncs: excludeFuncs,
					Debug:        debug,
				}
				return gofunclen.Check(paths, *maxLines, opts)
			}
		}
	},
	JSONRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderJSON(report.(gofunclen.Report), stdout, stderr)
	},
	TextRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderText(report.(gofunclen.Report), stdout, stderr)
	},
}

var goComplexityCfg = CheckerConfig{
	Name: "complexity",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
		maxComplexity := fs.Int("max-complexity", 10, "maximum cyclomatic complexity per function")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
				opts := gocomplexity.Options{
					ExcludeFiles: excludeFiles,
					ExcludeFuncs: excludeFuncs,
					Debug:        debug,
				}
				return gocomplexity.Check(paths, *maxComplexity, opts)
			}
		}
	},
	JSONRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderComplexityJSON(report.(gocomplexity.Report), stdout, stderr)
	},
	TextRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderComplexityText(report.(gocomplexity.Report), stdout, stderr)
	},
}

var goCrapCfg = CheckerConfig{
	Name: "crap",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
		threshold := fs.Float64("threshold", 30.0, "CRAP score threshold")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
				opts := crap.Options{
					ExcludeFiles: excludeFiles,
					ExcludeFuncs: excludeFuncs,
					Debug:        debug,
				}
				return crap.Check(paths, *threshold, opts)
			}
		}
	},
	JSONRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderCrapJSON(report.(crap.Report), stdout, stderr)
	},
	TextRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderCrapText(report.(crap.Report), stdout, stderr)
	},
}

var goFilelenCfg = CheckerConfig{
	Name: "filelen",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
		maxLines := fs.Int("max-lines", 300, "maximum file length in lines")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
				opts := filelen.Options{
					ExcludeFiles: excludeFiles,
					Debug:        debug,
				}
				return filelen.Check(paths, *maxLines, []string{".go"}, opts)
			}
		}
	},
	JSONRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderFilelenJSON(report.(filelen.Report), stdout, stderr)
	},
	TextRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderFilelenText(report.(filelen.Report), stdout, stderr)
	},
}

var goDuplicationCfg = CheckerConfig{
	Name: "duplication",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
		minLines := fs.Int("min-lines", 5, "minimum function length in lines to compare")
		minSimilarity := fs.Float64("min-similarity", 0.70, "minimum LCS-based similarity ratio for Type-3 detection (0.0-1.0)")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
				if *minSimilarity < 0.0 || *minSimilarity > 1.0 {
					return nil, fmt.Errorf("--min-similarity must be in range [0.0, 1.0], got %v", *minSimilarity)
				}
				opts := duplication.Options{
					ExcludeFiles: excludeFiles,
					ExcludeFuncs: excludeFuncs,
					Debug:        debug,
				}
				return duplication.CheckWithSimilarity(paths, *minLines, *minSimilarity, opts)
			}
		}
	},
	JSONRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderDuplicationJSON(report.(duplication.Report), stdout, stderr)
	},
	TextRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderDuplicationText(report.(duplication.Report), stdout, stderr)
	},
}

var goInstabilityCfg = CheckerConfig{
	Name: "instability",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
		minGap := fs.Float64("min-gap", 0, "minimum gap to report (violations with Gap > min-gap)")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
				opts := instability.Options{
					ExcludeFiles: excludeFiles,
					Debug:        debug,
				}
				return instability.Check(paths, *minGap, opts)
			}
		}
	},
	JSONRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderInstabilityJSON(report.(instability.Report), stdout, stderr)
	},
	TextRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderInstabilityText(report.(instability.Report), stdout, stderr)
	},
}

var goAbstractnessCfg = CheckerConfig{
	Name: "abstractness",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
		minDistance := fs.Float64("min-distance", 0.5, "minimum distance from main sequence to report (packages with |signedD| > min-distance)")
		minSurfaceRatio := fs.Float64("min-surface-ratio", 0.5, "minimum surface ratio for Pain zone candidates (ratio of exported to total declarations; only applies to Pain, not Uselessness)")
		ignoreDeepModuleGate := fs.Bool("ignore-deep-module-gate", false, "if true, flag all Pain candidates regardless of surface ratio (Story 2 behavior)")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
				opts := abstractness.Options{
					ExcludeFiles:         excludeFiles,
					Debug:                debug,
					IgnoreDeepModuleGate: *ignoreDeepModuleGate,
				}
				return abstractness.CheckWithSurfaceRatio(paths, *minDistance, *minSurfaceRatio, opts)
			}
		}
	},
	JSONRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderAbstractnessJSON(report.(abstractness.Report), stdout, stderr)
	},
	TextRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderAbstractnessText(report.(abstractness.Report), stdout, stderr)
	},
}

// Thin wrappers that dispatch to configs (preserves the dispatch map interface).
func runGoFunclen(args []string, stdout, stderr io.Writer) int {
	return runCheck(goFunclenCfg, args, stdout, stderr)
}

func runGoComplexity(args []string, stdout, stderr io.Writer) int {
	return runCheck(goComplexityCfg, args, stdout, stderr)
}

func runGoCrap(args []string, stdout, stderr io.Writer) int {
	return runCheck(goCrapCfg, args, stdout, stderr)
}

func runGoFilelen(args []string, stdout, stderr io.Writer) int {
	return runCheck(goFilelenCfg, args, stdout, stderr)
}

func runGoDuplication(args []string, stdout, stderr io.Writer) int {
	return runCheck(goDuplicationCfg, args, stdout, stderr)
}

func runGoInstability(args []string, stdout, stderr io.Writer) int {
	return runCheck(goInstabilityCfg, args, stdout, stderr)
}

func runGoAbstractness(args []string, stdout, stderr io.Writer) int {
	return runCheck(goAbstractnessCfg, args, stdout, stderr)
}

// runGoAll runs all Go checks and combines their reports.
func runGoAll(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("all", flag.ContinueOnError)
	format := fs.String("format", "text", "output format: text or json")
	excludeFile := fs.String("exclude-file", "", "comma-separated glob patterns for files to exclude")
	excludeFunc := fs.String("exclude-func", "", "comma-separated glob patterns for functions to exclude")
	debug := fs.Bool("debug", false, "include excluded files and functions in output")

	paths, excludeFiles, excludeFuncs, err := resolveArgs(fs, args, excludeFile, excludeFunc)
	if err != nil {
		reportError(err, stderr)
		return 2
	}

	gofunclenReport, complexityReport, crapReport, filelenReport, duplicationReport, instabilityReport, abstractnessReport, err := checkAll(paths, excludeFiles, excludeFuncs, *debug)
	if err != nil {
		reportError(err, stderr)
		return 2
	}

	combined := combinedReport{
		Gofunclen:    gofunclenReport,
		Complexity:   complexityReport,
		Crap:         crapReport,
		Filelen:      filelenReport,
		Duplication:  duplicationReport,
		Instability:  instabilityReport,
		Abstractness: abstractnessReport,
	}

	// Render output
	if *format == "json" {
		return renderAllJSON(combined, stdout, stderr)
	}
	return renderAllText(combined, stdout, stderr)
}

// checkAll runs all checks with shared options.
func checkAll(paths []string, excludeFiles, excludeFuncs []string, debug bool) (gofunclen.Report, gocomplexity.Report, crap.Report, filelen.Report, duplication.Report, instability.Report, abstractness.Report, error) {
	var (
		gofunclenReport   gofunclen.Report
		complexityReport  gocomplexity.Report
		crapReport        crap.Report
		filelenReport     filelen.Report
		duplicationReport duplication.Report
		instabilityReport instability.Report
		abstractnessReport abstractness.Report
	)

	checks := []func() error{
		func() error {
			var err error
			gofunclenReport, err = checkAllGofunclen(paths, excludeFiles, excludeFuncs, debug)
			return err
		},
		func() error {
			var err error
			complexityReport, err = checkAllComplexity(paths, excludeFiles, excludeFuncs, debug)
			return err
		},
		func() error {
			var err error
			crapReport, err = checkAllCrap(paths, excludeFiles, excludeFuncs, debug)
			return err
		},
		func() error {
			var err error
			filelenReport, err = checkAllFilelen(paths, excludeFiles, debug)
			return err
		},
		func() error {
			var err error
			duplicationReport, err = checkAllDuplication(paths, excludeFiles, excludeFuncs, debug)
			return err
		},
		func() error {
			var err error
			instabilityReport, err = checkAllInstability(paths, excludeFiles, debug)
			return err
		},
		func() error {
			var err error
			abstractnessReport, err = checkAllAbstractness(paths, excludeFiles, debug)
			return err
		},
	}

	for _, check := range checks {
		if err := check(); err != nil {
			return gofunclenReport, complexityReport, crapReport, filelenReport, duplicationReport, instabilityReport, abstractnessReport, err
		}
	}

	return gofunclenReport, complexityReport, crapReport, filelenReport, duplicationReport, instabilityReport, abstractnessReport, nil
}

func checkAllGofunclen(paths []string, excludeFiles, excludeFuncs []string, debug bool) (gofunclen.Report, error) {
	opts := gofunclen.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        debug,
	}
	return gofunclen.Check(paths, 50, opts)
}

func checkAllComplexity(paths []string, excludeFiles, excludeFuncs []string, debug bool) (gocomplexity.Report, error) {
	opts := gocomplexity.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        debug,
	}
	// ponytail: same default (10) in standalone and `all` mode, unlike crap's 30-vs-6 split — split it if a stricter `all` default turns out to matter.
	return gocomplexity.Check(paths, 10, opts)
}

func checkAllCrap(paths []string, excludeFiles, excludeFuncs []string, debug bool) (crap.Report, error) {
	opts := crap.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        debug,
	}
	return crap.Check(paths, 6.0, opts)
}

func checkAllFilelen(paths []string, excludeFiles []string, debug bool) (filelen.Report, error) {
	opts := filelen.Options{
		ExcludeFiles: excludeFiles,
		Debug:        debug,
	}
	return filelen.Check(paths, 300, []string{".go"}, opts)
}

func checkAllDuplication(paths []string, excludeFiles, excludeFuncs []string, debug bool) (duplication.Report, error) {
	opts := duplication.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        debug,
	}
	return duplication.Check(paths, 5, opts)
}

func checkAllInstability(paths []string, excludeFiles []string, debug bool) (instability.Report, error) {
	opts := instability.Options{
		ExcludeFiles: excludeFiles,
		Debug:        debug,
	}
	return instability.Check(paths, 0, opts)
}

func checkAllAbstractness(paths []string, excludeFiles []string, debug bool) (abstractness.Report, error) {
	opts := abstractness.Options{
		ExcludeFiles: excludeFiles,
		Debug:        debug,
	}
	return abstractness.Check(paths, 0.5, opts)
}
