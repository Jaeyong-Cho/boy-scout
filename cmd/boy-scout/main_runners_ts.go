package main

import (
	"flag"
	"io"

	"boy-scout/internal/filelen"
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

// Thin wrappers that dispatch to configs.
func runTsFunclen(args []string, stdout, stderr io.Writer) int {
	return runCheck(tsFunclenCfg, args, stdout, stderr)
}

func runTsFilelen(args []string, stdout, stderr io.Writer) int {
	return runCheck(tsFilelenCfg, args, stdout, stderr)
}

// runTsAll runs all TypeScript checks sequentially.
func runTsAll(args []string, stdout, stderr io.Writer) int {
	checks := []struct {
		name string
		fn   func([]string, io.Writer, io.Writer) int
	}{
		{"funclen", runTsFunclen},
		{"filelen", runTsFilelen},
	}

	for _, check := range checks {
		if result := check.fn(args, stdout, stderr); result != 0 {
			return result
		}
	}
	return 0
}
