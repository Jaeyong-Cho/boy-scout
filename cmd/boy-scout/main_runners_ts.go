package main

import (
	"flag"
	"fmt"
	"io"

	"boy-scout/internal/assertutil"
	"boy-scout/internal/collen"
	"boy-scout/internal/filelen"
	"boy-scout/internal/tscomplexity"
	"boy-scout/internal/tsfunclen"
)

// ============ TypeScript Checkers ============

var tsFunclenCfg = CheckerConfig[tsfunclen.Report]{
	Name: "funclen",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (tsfunclen.Report, error) {
		maxLines := fs.Int("max-lines", 50, "maximum function length in lines")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (tsfunclen.Report, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (tsfunclen.Report, error) {
				opts := tsfunclen.Options{
					ExcludeFiles: excludeFiles,
					ExcludeFuncs: excludeFuncs,
					Debug:        debug,
				}
				return tsfunclen.Check(paths, *maxLines, opts)
			}
		}
	},
	JSONRenderer: renderTsFunclenJSON,
	TextRenderer: renderTsFunclenText,
}

var tsFilelenCfg = CheckerConfig[filelen.Report]{
	Name: "filelen",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (filelen.Report, error) {
		maxLines := fs.Int("max-lines", 300, "maximum file length in lines")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (filelen.Report, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (filelen.Report, error) {
				opts := filelen.Options{
					ExcludeFiles: excludeFiles,
					Debug:        debug,
				}
				return filelen.Check(paths, *maxLines, []string{".ts", ".tsx", ".html", ".css"}, opts)
			}
		}
	},
	JSONRenderer: renderFilelenJSON,
	TextRenderer: renderFilelenText,
}

var tsCollenCfg = CheckerConfig[collen.Report]{
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
				return collen.Check(paths, *maxChars, []string{".ts", ".tsx", ".html", ".css"}, opts)
			}
		}
	},
	JSONRenderer: renderCollenJSON,
	TextRenderer: renderCollenText,
}

var tsComplexityCfg = CheckerConfig[tscomplexity.Report]{
	Name: "complexity",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (tscomplexity.Report, error) {
		maxComplexity := fs.Int("max-complexity", 6, "maximum cyclomatic complexity per function")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (tscomplexity.Report, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (tscomplexity.Report, error) {
				opts := tscomplexity.Options{ExcludeFiles: excludeFiles, ExcludeFuncs: excludeFuncs, Debug: debug}
				return tscomplexity.Check(paths, *maxComplexity, opts)
			}
		}
	},
	JSONRenderer: renderTsComplexityJSON,
	TextRenderer: renderTsComplexityText,
}

// Thin wrappers that dispatch to configs.
func runTsFunclen(args []string, stdout, stderr io.Writer) int {
	return runCheck(tsFunclenCfg, args, stdout, stderr)
}

func runTsFilelen(args []string, stdout, stderr io.Writer) int {
	return runCheck(tsFilelenCfg, args, stdout, stderr)
}

func runTsCollen(args []string, stdout, stderr io.Writer) int {
	return runCheck(tsCollenCfg, args, stdout, stderr)
}

func runTsComplexity(args []string, stdout, stderr io.Writer) int {
	return runCheck(tsComplexityCfg, args, stdout, stderr)
}

// runTsAll runs all TypeScript checks and combines their reports.
func runTsAll(args []string, stdout, stderr io.Writer) int {
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

	combined, err := checkAllTs(paths, excludeFiles, excludeFuncs, *debug)
	if err != nil {
		reportError(err, stderr)
		return 2
	}

	// Render output
	if *format == "json" {
		return renderTsAllJSON(combined, stdout, stderr)
	}
	return renderTsAllText(combined, stdout, stderr)
}

// checkAllTs runs all TypeScript checks with shared options.
func checkAllTs(paths []string, excludeFiles, excludeFuncs []string, debug bool) (tsCombinedReport, error) {
	assertutil.Assertf(len(paths) > 0, "checkAllTs: paths must not be empty")

	var report tsCombinedReport

	checks := []func() error{
		func() error {
			var err error
			report.Funclen, err = checkAllTsFunclen(paths, excludeFiles, excludeFuncs, debug)
			return err
		},
		func() error {
			var err error
			report.Filelen, err = checkAllTsFilelen(paths, excludeFiles, debug)
			return err
		},
		func() error {
			var err error
			report.Collen, err = checkAllTsCollen(paths, excludeFiles, debug)
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

func checkAllTsFunclen(paths []string, excludeFiles, excludeFuncs []string, debug bool) (tsfunclen.Report, error) {
	opts := tsfunclen.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        debug,
	}
	return tsfunclen.Check(paths, 50, opts)
}

func checkAllTsFilelen(paths []string, excludeFiles []string, debug bool) (filelen.Report, error) {
	opts := filelen.Options{
		ExcludeFiles: excludeFiles,
		Debug:        debug,
	}
	return filelen.Check(paths, 300, []string{".ts", ".tsx", ".html", ".css"}, opts)
}

func checkAllTsCollen(paths []string, excludeFiles []string, debug bool) (collen.Report, error) {
	opts := collen.Options{
		ExcludeFiles: excludeFiles,
		Debug:        debug,
	}
	return collen.Check(paths, 100, []string{".ts", ".tsx", ".html", ".css"}, opts)
}
