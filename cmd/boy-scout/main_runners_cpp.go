package main

import (
	"flag"
	"fmt"
	"io"

	"boy-scout/internal/assertutil"
	"boy-scout/internal/cppcohesion"
	"boy-scout/internal/cppcomplexity"
	"boy-scout/internal/cppduplication"
	"boy-scout/internal/cppfunclen"
	"boy-scout/internal/filelen"
	"boy-scout/internal/linelen"
)

// ============ C++ Checkers ============

var cppFilelenCfg = CheckerConfig{
	Name: "filelen",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
		maxLines := fs.Int("max-lines", 300, "maximum file length in lines")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
				opts := filelen.Options{
					ExcludeFiles: excludeFiles,
					Debug:        debug,
				}
				return filelen.Check(paths, *maxLines, []string{".cpp", ".h", ".hpp"}, opts)
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

var cppLinelenCfg = CheckerConfig{
	Name: "linelen",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
		maxChars := fs.Int("max-chars", 100, "maximum line length in characters")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
				opts := linelen.Options{
					ExcludeFiles: excludeFiles,
					Debug:        debug,
				}
				return linelen.Check(paths, *maxChars, []string{".cpp", ".h", ".hpp"}, opts)
			}
		}
	},
	JSONRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderLinelenJSON(report.(linelen.Report), stdout, stderr)
	},
	TextRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderLinelenText(report.(linelen.Report), stdout, stderr)
	},
}

var cppFunclenCfg = CheckerConfig{
	Name: "funclen",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
		maxLines := fs.Int("max-lines", 50, "maximum function length in lines")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
				opts := cppfunclen.Options{
					ExcludeFiles: excludeFiles,
					ExcludeFuncs: excludeFuncs,
					Debug:        debug,
				}
				return cppfunclen.Check(paths, *maxLines, opts)
			}
		}
	},
	JSONRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderCppFunclenJSON(report.(cppfunclen.Report), stdout, stderr)
	},
	TextRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderCppFunclenText(report.(cppfunclen.Report), stdout, stderr)
	},
}

var cppComplexityCfg = CheckerConfig{
	Name: "complexity",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
		maxComplexity := fs.Int("max-complexity", 6, "maximum cyclomatic complexity per function")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
				opts := cppcomplexity.Options{
					ExcludeFiles: excludeFiles,
					ExcludeFuncs: excludeFuncs,
					Debug:        debug,
				}
				return cppcomplexity.Check(paths, *maxComplexity, opts)
			}
		}
	},
	JSONRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderCppComplexityJSON(report.(cppcomplexity.Report), stdout, stderr)
	},
	TextRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderCppComplexityText(report.(cppcomplexity.Report), stdout, stderr)
	},
}

var cppCohesionCfg = CheckerConfig{
	Name: "cohesion",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
				opts := cppcohesion.Options{
					ExcludeFiles: excludeFiles,
					Debug:        debug,
				}
				return cppcohesion.Check(paths, opts)
			}
		}
	},
	JSONRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderCppCohesionJSON(report.(cppcohesion.Report), stdout, stderr)
	},
	TextRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderCppCohesionText(report.(cppcohesion.Report), stdout, stderr)
	},
}

var cppDuplicationCfg = CheckerConfig{
	Name: "duplication",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
		minLines := fs.Int("min-lines", 5, "minimum function length in lines to compare")
		minSimilarity := fs.Float64("min-similarity", 0.70, "minimum LCS-based similarity ratio for Type-3 detection (0.0-1.0)")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
				if *minSimilarity < 0.0 || *minSimilarity > 1.0 {
					return nil, fmt.Errorf("--min-similarity must be in range [0.0, 1.0], got %v", *minSimilarity)
				}
				opts := cppduplication.Options{
					ExcludeFiles: excludeFiles,
					ExcludeFuncs: excludeFuncs,
					Debug:        debug,
				}
				return cppduplication.CheckWithSimilarity(paths, *minLines, *minSimilarity, opts)
			}
		}
	},
	JSONRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderDuplicationJSON(report.(cppduplication.Report), stdout, stderr)
	},
	TextRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderDuplicationText(report.(cppduplication.Report), stdout, stderr)
	},
}

// Thin wrappers that dispatch to configs.
func runCppFilelen(args []string, stdout, stderr io.Writer) int {
	return runCheck(cppFilelenCfg, args, stdout, stderr)
}

func runCppLinelen(args []string, stdout, stderr io.Writer) int {
	return runCheck(cppLinelenCfg, args, stdout, stderr)
}

func runCppFunclen(args []string, stdout, stderr io.Writer) int {
	return runCheck(cppFunclenCfg, args, stdout, stderr)
}

func runCppComplexity(args []string, stdout, stderr io.Writer) int {
	return runCheck(cppComplexityCfg, args, stdout, stderr)
}

func runCppCohesion(args []string, stdout, stderr io.Writer) int {
	return runCheck(cppCohesionCfg, args, stdout, stderr)
}

func runCppDuplication(args []string, stdout, stderr io.Writer) int {
	return runCheck(cppDuplicationCfg, args, stdout, stderr)
}

// runCppAll runs all C++ checks and combines their reports.
func runCppAll(args []string, stdout, stderr io.Writer) int {
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

	funclenReport, complexityReport, cohesionReport, filelenReport, linelenReport, duplicationReport, err := checkAllCpp(paths, excludeFiles, excludeFuncs, *debug)
	if err != nil {
		reportError(err, stderr)
		return 2
	}

	combined := cppCombinedReport{
		Funclen:     funclenReport,
		Complexity:  complexityReport,
		Cohesion:    cohesionReport,
		Filelen:     filelenReport,
		Linelen:     linelenReport,
		Duplication: duplicationReport,
	}

	// Render output
	if *format == "json" {
		return renderCppAllJSON(combined, stdout, stderr)
	}
	return renderCppAllText(combined, stdout, stderr)
}

// checkAllCpp runs all C++ checks with shared options.
func checkAllCpp(paths []string, excludeFiles, excludeFuncs []string, debug bool) (cppfunclen.Report, cppcomplexity.Report, cppcohesion.Report, filelen.Report, linelen.Report, cppduplication.Report, error) {
	assertutil.Assertf(len(paths) > 0, "checkAllCpp: paths must not be empty")

	var (
		funclenReport     cppfunclen.Report
		complexityReport  cppcomplexity.Report
		cohesionReport    cppcohesion.Report
		filelenReport     filelen.Report
		linelenReport     linelen.Report
		duplicationReport cppduplication.Report
	)

	checks := []func() error{
		func() error {
			var err error
			funclenReport, err = checkAllCppFunclen(paths, excludeFiles, excludeFuncs, debug)
			return err
		},
		func() error {
			var err error
			complexityReport, err = checkAllCppComplexity(paths, excludeFiles, excludeFuncs, debug)
			return err
		},
		func() error {
			var err error
			cohesionReport, err = checkAllCppCohesion(paths, excludeFiles, debug)
			return err
		},
		func() error {
			var err error
			filelenReport, err = checkAllCppFilelen(paths, excludeFiles, debug)
			return err
		},
		func() error {
			var err error
			linelenReport, err = checkAllCppLinelen(paths, excludeFiles, debug)
			return err
		},
		func() error {
			var err error
			duplicationReport, err = checkAllCppDuplication(paths, excludeFiles, excludeFuncs, debug)
			return err
		},
	}

	for _, check := range checks {
		if err := check(); err != nil {
			return funclenReport, complexityReport, cohesionReport, filelenReport, linelenReport, duplicationReport, err
		}
	}

	return funclenReport, complexityReport, cohesionReport, filelenReport, linelenReport, duplicationReport, nil
}

func checkAllCppFunclen(paths []string, excludeFiles, excludeFuncs []string, debug bool) (cppfunclen.Report, error) {
	opts := cppfunclen.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        debug,
	}
	return cppfunclen.Check(paths, 50, opts)
}

func checkAllCppFilelen(paths []string, excludeFiles []string, debug bool) (filelen.Report, error) {
	opts := filelen.Options{
		ExcludeFiles: excludeFiles,
		Debug:        debug,
	}
	return filelen.Check(paths, 300, []string{".cpp", ".h", ".hpp"}, opts)
}

func checkAllCppLinelen(paths []string, excludeFiles []string, debug bool) (linelen.Report, error) {
	opts := linelen.Options{
		ExcludeFiles: excludeFiles,
		Debug:        debug,
	}
	return linelen.Check(paths, 100, []string{".cpp", ".h", ".hpp"}, opts)
}

func checkAllCppComplexity(paths []string, excludeFiles, excludeFuncs []string, debug bool) (cppcomplexity.Report, error) {
	opts := cppcomplexity.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        debug,
	}
	return cppcomplexity.Check(paths, 6, opts)
}

func checkAllCppCohesion(paths []string, excludeFiles []string, debug bool) (cppcohesion.Report, error) {
	opts := cppcohesion.Options{
		ExcludeFiles: excludeFiles,
		Debug:        debug,
	}
	return cppcohesion.Check(paths, opts)
}

func checkAllCppDuplication(paths []string, excludeFiles, excludeFuncs []string, debug bool) (cppduplication.Report, error) {
	opts := cppduplication.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        debug,
	}
	return cppduplication.Check(paths, 5, opts)
}
