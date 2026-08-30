package main

import (
	"flag"
	"io"

	"boy-scout/internal/cppcomplexity"
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

// runCppAll runs all C++ checks sequentially.
func runCppAll(args []string, stdout, stderr io.Writer) int {
	checks := []struct {
		name string
		fn   func([]string, io.Writer, io.Writer) int
	}{
		{"funclen", runCppFunclen},
		{"filelen", runCppFilelen},
	}

	for _, check := range checks {
		if result := check.fn(args, stdout, stderr); result != 0 {
			return result
		}
	}
	return 0
}
