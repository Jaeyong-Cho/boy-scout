package main

import (
	"flag"
	"io"

	"boy-scout/internal/assertutil"
	"boy-scout/internal/filelen"
	"boy-scout/internal/linelen"
	"boy-scout/internal/tscomplexity"
	"boy-scout/internal/tsfunclen"
)

// ============ TypeScript Checkers ============

var tsFunclenCfg = CheckerConfig{
	Name: "funclen",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
		maxLines := fs.Int("max-lines", 50, "maximum function length in lines")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
				opts := tsfunclen.Options{
					ExcludeFiles: excludeFiles,
					ExcludeFuncs: excludeFuncs,
					Debug:        debug,
				}
				return tsfunclen.Check(paths, *maxLines, opts)
			}
		}
	},
	JSONRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderTsFunclenJSON(report.(tsfunclen.Report), stdout, stderr)
	},
	TextRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderTsFunclenText(report.(tsfunclen.Report), stdout, stderr)
	},
}

var tsFilelenCfg = CheckerConfig{
	Name: "filelen",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
		maxLines := fs.Int("max-lines", 300, "maximum file length in lines")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
				opts := filelen.Options{
					ExcludeFiles: excludeFiles,
					Debug:        debug,
				}
				return filelen.Check(paths, *maxLines, []string{".ts", ".tsx", ".html", ".css"}, opts)
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

var tsLinelenCfg = CheckerConfig{
	Name: "linelen",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
		maxChars := fs.Int("max-chars", 100, "maximum line length in characters")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
				opts := linelen.Options{
					ExcludeFiles: excludeFiles,
					Debug:        debug,
				}
				return linelen.Check(paths, *maxChars, []string{".ts", ".tsx", ".html", ".css"}, opts)
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

var tsComplexityCfg = CheckerConfig{
	Name: "complexity",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
		maxComplexity := fs.Int("max-complexity", 6, "maximum cyclomatic complexity per function")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
				opts := tscomplexity.Options{ExcludeFiles: excludeFiles, ExcludeFuncs: excludeFuncs, Debug: debug}
				return tscomplexity.Check(paths, *maxComplexity, opts)
			}
		}
	},
	JSONRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderTsComplexityJSON(report.(tscomplexity.Report), stdout, stderr)
	},
	TextRenderer: func(report interface{}, stdout, stderr io.Writer) int {
		return renderTsComplexityText(report.(tscomplexity.Report), stdout, stderr)
	},
}

// Thin wrappers that dispatch to configs.
func runTsFunclen(args []string, stdout, stderr io.Writer) int {
	return runCheck(tsFunclenCfg, args, stdout, stderr)
}

func runTsFilelen(args []string, stdout, stderr io.Writer) int {
	return runCheck(tsFilelenCfg, args, stdout, stderr)
}

func runTsLinelen(args []string, stdout, stderr io.Writer) int {
	return runCheck(tsLinelenCfg, args, stdout, stderr)
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

	funclenReport, filelenReport, linelenReport, err := checkAllTs(paths, excludeFiles, excludeFuncs, *debug)
	if err != nil {
		reportError(err, stderr)
		return 2
	}

	combined := tsCombinedReport{
		Funclen: funclenReport,
		Filelen: filelenReport,
		Linelen: linelenReport,
	}

	// Render output
	if *format == "json" {
		return renderTsAllJSON(combined, stdout, stderr)
	}
	return renderTsAllText(combined, stdout, stderr)
}

// checkAllTs runs all TypeScript checks with shared options.
func checkAllTs(paths []string, excludeFiles, excludeFuncs []string, debug bool) (tsfunclen.Report, filelen.Report, linelen.Report, error) {
	assertutil.Assertf(len(paths) > 0, "checkAllTs: paths must not be empty")

	var (
		funclenReport tsfunclen.Report
		filelenReport filelen.Report
		linelenReport linelen.Report
	)

	checks := []func() error{
		func() error {
			var err error
			funclenReport, err = checkAllTsFunclen(paths, excludeFiles, excludeFuncs, debug)
			return err
		},
		func() error {
			var err error
			filelenReport, err = checkAllTsFilelen(paths, excludeFiles, debug)
			return err
		},
		func() error {
			var err error
			linelenReport, err = checkAllTsLinelen(paths, excludeFiles, debug)
			return err
		},
	}

	for _, check := range checks {
		if err := check(); err != nil {
			return funclenReport, filelenReport, linelenReport, err
		}
	}

	return funclenReport, filelenReport, linelenReport, nil
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

func checkAllTsLinelen(paths []string, excludeFiles []string, debug bool) (linelen.Report, error) {
	opts := linelen.Options{
		ExcludeFiles: excludeFiles,
		Debug:        debug,
	}
	return linelen.Check(paths, 100, []string{".ts", ".tsx", ".html", ".css"}, opts)
}
