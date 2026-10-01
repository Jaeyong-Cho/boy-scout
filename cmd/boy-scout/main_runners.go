package main

import (
	"flag"
	"fmt"
	"io"

	"boy-scout/internal/collen"
	"boy-scout/internal/duplication"
	"boy-scout/internal/filelen"
	"boy-scout/internal/gocomplexity"
	"boy-scout/internal/gofunclen"
)

// CheckerConfig wraps the setup and execution of a checker command.
// Setup registers flags and returns a factory that accepts the debug flag
// (populated after flag parsing) and returns the actual checker function.
type CheckerConfig[R any] struct {
	Name         string
	Setup        func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (R, error)
	JSONRenderer func(R, io.Writer, io.Writer) int
	TextRenderer func(R, io.Writer, io.Writer) int
}

// runCheck is the generic runner that all individual checkers delegate to.
// It handles flag parsing, error reporting, and output rendering.
func runCheck[R any](cfg CheckerConfig[R], args []string, stdout, stderr io.Writer) int {
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

var goFunclenCfg = CheckerConfig[gofunclen.Report]{
	Name: "gofunclen",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (gofunclen.Report, error) {
		maxLines := fs.Int("max-lines", 50, "maximum function length in lines")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (gofunclen.Report, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (gofunclen.Report, error) {
				opts := gofunclen.Options{
					ExcludeFiles: excludeFiles,
					ExcludeFuncs: excludeFuncs,
					Debug:        debug,
				}
				return gofunclen.Check(paths, *maxLines, opts)
			}
		}
	},
	JSONRenderer: renderJSON,
	TextRenderer: renderText,
}

var goComplexityCfg = CheckerConfig[gocomplexity.Report]{
	Name: "complexity",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (gocomplexity.Report, error) {
		maxComplexity := fs.Int("max-complexity", 6, "maximum cyclomatic complexity per function")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (gocomplexity.Report, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (gocomplexity.Report, error) {
				opts := gocomplexity.Options{
					ExcludeFiles: excludeFiles,
					ExcludeFuncs: excludeFuncs,
					Debug:        debug,
				}
				return gocomplexity.Check(paths, *maxComplexity, opts)
			}
		}
	},
	JSONRenderer: renderComplexityJSON,
	TextRenderer: renderComplexityText,
}

var goFilelenCfg = CheckerConfig[filelen.Report]{
	Name: "filelen",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (filelen.Report, error) {
		maxLines := fs.Int("max-lines", 300, "maximum file length in lines")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (filelen.Report, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (filelen.Report, error) {
				opts := filelen.Options{
					ExcludeFiles: excludeFiles,
					Debug:        debug,
				}
				return filelen.Check(paths, *maxLines, []string{".go"}, opts)
			}
		}
	},
	JSONRenderer: renderFilelenJSON,
	TextRenderer: renderFilelenText,
}

var goCollenCfg = CheckerConfig[collen.Report]{
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
				return collen.Check(paths, *maxChars, []string{".go"}, opts)
			}
		}
	},
	JSONRenderer: renderCollenJSON,
	TextRenderer: renderCollenText,
}

var goDuplicationCfg = CheckerConfig[duplication.Report]{
	Name: "duplication",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (duplication.Report, error) {
		minLines := fs.Int("min-lines", 5, "minimum function length in lines to compare")
		minSimilarity := fs.Float64("min-similarity", 0.70, "minimum LCS-based similarity ratio for Type-3 detection (0.0-1.0)")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (duplication.Report, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (duplication.Report, error) {
				if *minSimilarity < 0.0 || *minSimilarity > 1.0 {
					return duplication.Report{}, fmt.Errorf("--min-similarity must be in range [0.0, 1.0], got %v", *minSimilarity)
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
	JSONRenderer: renderDuplicationJSON,
	TextRenderer: renderDuplicationText,
}

// Thin wrappers that dispatch to configs (preserves the dispatch map interface).
func runGoFunclen(args []string, stdout, stderr io.Writer) int {
	return runCheck(goFunclenCfg, args, stdout, stderr)
}

func runGoComplexity(args []string, stdout, stderr io.Writer) int {
	return runCheck(goComplexityCfg, args, stdout, stderr)
}

func runGoFilelen(args []string, stdout, stderr io.Writer) int {
	return runCheck(goFilelenCfg, args, stdout, stderr)
}

func runGoCollen(args []string, stdout, stderr io.Writer) int {
	return runCheck(goCollenCfg, args, stdout, stderr)
}

func runGoDuplication(args []string, stdout, stderr io.Writer) int {
	return runCheck(goDuplicationCfg, args, stdout, stderr)
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

	combined, err := checkAll(paths, excludeFiles, excludeFuncs, *debug)
	if err != nil {
		reportError(err, stderr)
		return 2
	}

	// Render output
	if *format == "json" {
		return renderAllJSON(combined, stdout, stderr)
	}
	return renderAllText(combined, stdout, stderr)
}

// checkAll runs all checks with shared options.
func checkAll(paths []string, excludeFiles, excludeFuncs []string, debug bool) (combinedReport, error) {
	var report combinedReport

	checks := []func() error{
		func() error {
			var err error
			report.Gofunclen, err = checkAllGofunclen(paths, excludeFiles, excludeFuncs, debug)
			return err
		},
		func() error {
			var err error
			report.Complexity, err = checkAllComplexity(paths, excludeFiles, excludeFuncs, debug)
			return err
		},
		func() error {
			var err error
			report.Filelen, err = checkAllFilelen(paths, excludeFiles, debug)
			return err
		},
		func() error {
			var err error
			report.Collen, err = checkAllCollen(paths, excludeFiles, debug)
			return err
		},
		func() error {
			var err error
			report.Duplication, err = checkAllDuplication(paths, excludeFiles, excludeFuncs, debug)
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
	return gocomplexity.Check(paths, 6, opts)
}

func checkAllFilelen(paths []string, excludeFiles []string, debug bool) (filelen.Report, error) {
	opts := filelen.Options{
		ExcludeFiles: excludeFiles,
		Debug:        debug,
	}
	return filelen.Check(paths, 300, []string{".go"}, opts)
}

func checkAllCollen(paths []string, excludeFiles []string, debug bool) (collen.Report, error) {
	opts := collen.Options{
		ExcludeFiles: excludeFiles,
		Debug:        debug,
	}
	return collen.Check(paths, 100, []string{".go"}, opts)
}

func checkAllDuplication(paths []string, excludeFiles, excludeFuncs []string, debug bool) (duplication.Report, error) {
	opts := duplication.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        debug,
	}
	return duplication.Check(paths, 5, opts)
}
