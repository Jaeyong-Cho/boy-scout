package main

import (
	"flag"
	"fmt"
	"io"

	"boy-scout/internal/assertutil"
	"boy-scout/internal/collen"
	"boy-scout/internal/cppcomplexity"
	"boy-scout/internal/cppduplication"
	"boy-scout/internal/cppfunclen"
	"boy-scout/internal/filelen"
)

// ============ C++ Checkers ============

var cppFilelenCfg = CheckerConfig[filelen.Report]{
	Name: "filelen",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (filelen.Report, error) {
		maxLines := fs.Int("max-lines", 300, "maximum file length in lines")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (filelen.Report, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (filelen.Report, error) {
				opts := filelen.Options{
					ExcludeFiles: excludeFiles,
					Debug:        debug,
				}
				return filelen.Check(paths, *maxLines, []string{".cpp", ".h", ".hpp"}, opts)
			}
		}
	},
	JSONRenderer: renderFilelenJSON,
	TextRenderer: renderFilelenText,
}

var cppCollenCfg = CheckerConfig[collen.Report]{
	Name: "collen",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (collen.Report, error) {
		maxChars := fs.Int("max-chars", 100, "maximum line length in characters")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (collen.Report, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (collen.Report, error) {
				if *maxChars <= 0 {
					return collen.Report{}, fmt.Errorf("--max-chars must be positive, got %d", *maxChars)
				}
				opts := collen.Options{
					ExcludeFiles: excludeFiles,
					Debug:        debug,
				}
				return collen.Check(paths, *maxChars, []string{".cpp", ".h", ".hpp"}, opts)
			}
		}
	},
	JSONRenderer: renderCollenJSON,
	TextRenderer: renderCollenText,
}

var cppFunclenCfg = CheckerConfig[cppfunclen.Report]{
	Name: "funclen",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (cppfunclen.Report, error) {
		maxLines := fs.Int("max-lines", 50, "maximum function length in lines")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (cppfunclen.Report, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (cppfunclen.Report, error) {
				opts := cppfunclen.Options{
					ExcludeFiles: excludeFiles,
					ExcludeFuncs: excludeFuncs,
					Debug:        debug,
				}
				return cppfunclen.Check(paths, *maxLines, opts)
			}
		}
	},
	JSONRenderer: renderCppFunclenJSON,
	TextRenderer: renderCppFunclenText,
}

var cppComplexityCfg = CheckerConfig[cppcomplexity.Report]{
	Name: "complexity",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (cppcomplexity.Report, error) {
		maxComplexity := fs.Int("max-complexity", 6, "maximum cyclomatic complexity per function")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (cppcomplexity.Report, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (cppcomplexity.Report, error) {
				opts := cppcomplexity.Options{
					ExcludeFiles: excludeFiles,
					ExcludeFuncs: excludeFuncs,
					Debug:        debug,
				}
				return cppcomplexity.Check(paths, *maxComplexity, opts)
			}
		}
	},
	JSONRenderer: renderCppComplexityJSON,
	TextRenderer: renderCppComplexityText,
}

var cppDuplicationCfg = CheckerConfig[cppduplication.Report]{
	Name: "duplication",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (cppduplication.Report, error) {
		minLines := fs.Int("min-lines", 5, "minimum function length in lines to compare")
		minSimilarity := fs.Float64("min-similarity", 0.70, "minimum LCS-based similarity ratio for Type-3 detection (0.0-1.0)")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (cppduplication.Report, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (cppduplication.Report, error) {
				if *minSimilarity < 0.0 || *minSimilarity > 1.0 {
					return cppduplication.Report{}, fmt.Errorf("--min-similarity must be in range [0.0, 1.0], got %v", *minSimilarity)
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
	JSONRenderer: renderDuplicationJSON,
	TextRenderer: renderDuplicationText,
}

// Thin wrappers that dispatch to configs.
func runCppFilelen(args []string, stdout, stderr io.Writer) int {
	return runCheck(cppFilelenCfg, args, stdout, stderr)
}

func runCppCollen(args []string, stdout, stderr io.Writer) int {
	return runCheck(cppCollenCfg, args, stdout, stderr)
}

func runCppFunclen(args []string, stdout, stderr io.Writer) int {
	return runCheck(cppFunclenCfg, args, stdout, stderr)
}

func runCppComplexity(args []string, stdout, stderr io.Writer) int {
	return runCheck(cppComplexityCfg, args, stdout, stderr)
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

	combined, err := checkAllCpp(paths, excludeFiles, excludeFuncs, *debug)
	if err != nil {
		reportError(err, stderr)
		return 2
	}

	// Render output
	if *format == "json" {
		return renderCppAllJSON(combined, stdout, stderr)
	}
	return renderCppAllText(combined, stdout, stderr)
}

// checkAllCpp runs all C++ checks with shared options.
func checkAllCpp(paths []string, excludeFiles, excludeFuncs []string, debug bool) (cppCombinedReport, error) {
	assertutil.Assertf(len(paths) > 0, "checkAllCpp: paths must not be empty")

	var report cppCombinedReport

	checks := []func() error{
		func() error {
			var err error
			report.Funclen, err = checkAllCppFunclen(paths, excludeFiles, excludeFuncs, debug)
			return err
		},
		func() error {
			var err error
			report.Complexity, err = checkAllCppComplexity(paths, excludeFiles, excludeFuncs, debug)
			return err
		},
		func() error {
			var err error
			report.Filelen, err = checkAllCppFilelen(paths, excludeFiles, debug)
			return err
		},
		func() error {
			var err error
			report.Collen, err = checkAllCppCollen(paths, excludeFiles, debug)
			return err
		},
		func() error {
			var err error
			report.Duplication, err = checkAllCppDuplication(paths, excludeFiles, excludeFuncs, debug)
			return err
		},
	}

	for _, check := range checks {
		if err := check(); err != nil {
			return report, err
		}
	}

	return report, nil
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

func checkAllCppCollen(paths []string, excludeFiles []string, debug bool) (collen.Report, error) {
	opts := collen.Options{
		ExcludeFiles: excludeFiles,
		Debug:        debug,
	}
	return collen.Check(paths, 100, []string{".cpp", ".h", ".hpp"}, opts)
}

func checkAllCppComplexity(paths []string, excludeFiles, excludeFuncs []string, debug bool) (cppcomplexity.Report, error) {
	opts := cppcomplexity.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        debug,
	}
	return cppcomplexity.Check(paths, 6, opts)
}

func checkAllCppDuplication(paths []string, excludeFiles, excludeFuncs []string, debug bool) (cppduplication.Report, error) {
	opts := cppduplication.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        debug,
	}
	return cppduplication.Check(paths, 5, opts)
}
