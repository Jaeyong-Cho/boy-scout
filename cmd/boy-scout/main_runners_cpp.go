package main

import (
	"flag"
	"io"

	"boy-scout/internal/abstractness"
	"boy-scout/internal/cppabstractness"
	"boy-scout/internal/cppfunclen"
	"boy-scout/internal/cppinstability"
	"boy-scout/internal/filelen"
	"boy-scout/internal/instability"
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

var cppInstabilityCfg = CheckerConfig{
	Name: "instability",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
		minGap := fs.Float64("min-gap", 0, "minimum gap threshold for violations")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
				opts := cppinstability.Options{
					ExcludeFiles: excludeFiles,
					Debug:        debug,
				}
				return cppinstability.Check(paths, *minGap, opts)
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

var cppAbstractnessCfg = CheckerConfig{
	Name: "abstractness",
	Setup: func(fs *flag.FlagSet) func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
		minDistance := fs.Float64("min-distance", 0.5, "minimum distance from main sequence to report (files with |signedD| > min-distance)")
		return func(debug bool) func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
			return func(paths, excludeFiles, excludeFuncs []string) (interface{}, error) {
				opts := cppabstractness.Options{
					ExcludeFiles: excludeFiles,
					Debug:        debug,
				}
				return cppabstractness.Check(paths, *minDistance, opts)
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

// Thin wrappers that dispatch to configs.
func runCppFilelen(args []string, stdout, stderr io.Writer) int {
	return runCheck(cppFilelenCfg, args, stdout, stderr)
}

func runCppFunclen(args []string, stdout, stderr io.Writer) int {
	return runCheck(cppFunclenCfg, args, stdout, stderr)
}

func runCppInstability(args []string, stdout, stderr io.Writer) int {
	return runCheck(cppInstabilityCfg, args, stdout, stderr)
}

func runCppAbstractness(args []string, stdout, stderr io.Writer) int {
	return runCheck(cppAbstractnessCfg, args, stdout, stderr)
}

// runCppAll runs all C++ checks sequentially.
func runCppAll(args []string, stdout, stderr io.Writer) int {
	checks := []struct {
		name string
		fn   func([]string, io.Writer, io.Writer) int
	}{
		{"funclen", runCppFunclen},
		{"filelen", runCppFilelen},
		{"instability", runCppInstability},
		{"abstractness", runCppAbstractness},
	}

	for _, check := range checks {
		if result := check.fn(args, stdout, stderr); result != 0 {
			return result
		}
	}
	return 0
}
