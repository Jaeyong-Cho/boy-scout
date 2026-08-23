package main

import (
	"flag"
	"fmt"
	"io"

	"boy-scout/internal/cppfunclen"
	"boy-scout/internal/cppinstability"
	"boy-scout/internal/filelen"
)

func runCppFilelen(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("filelen", flag.ContinueOnError)
	maxLines := fs.Int("max-lines", 300, "maximum file length in lines")
	format := fs.String("format", "text", "output format: text or json")
	excludeFile := fs.String("exclude-file", "", "comma-separated glob patterns for files to exclude")
	excludeFunc := fs.String("exclude-func", "", "unused (filelen has no function-level concept)")
	debug := fs.Bool("debug", false, "include excluded files in output")

	paths, excludeFiles, _, err := resolveArgs(fs, args, excludeFile, excludeFunc)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	opts := filelen.Options{
		ExcludeFiles: excludeFiles,
		Debug:        *debug,
	}

	report, err := filelen.Check(paths, *maxLines, []string{".cpp", ".h", ".hpp"}, opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	if *format == "json" {
		return renderFilelenJSON(report, stdout, stderr)
	}
	return renderFilelenText(report, stdout, stderr)
}

func runCppFunclen(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("funclen", flag.ContinueOnError)
	maxLines := fs.Int("max-lines", 50, "maximum function length in lines")
	format := fs.String("format", "text", "output format: text or json")
	excludeFile := fs.String("exclude-file", "", "comma-separated glob patterns for files to exclude")
	excludeFunc := fs.String("exclude-func", "", "comma-separated glob patterns for functions to exclude")
	debug := fs.Bool("debug", false, "include excluded functions in output")

	paths, excludeFiles, excludeFuncs, err := resolveArgs(fs, args, excludeFile, excludeFunc)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	opts := cppfunclen.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        *debug,
	}

	report, err := cppfunclen.Check(paths, *maxLines, opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	if *format == "json" {
		return renderCppFunclenJSON(report, stdout, stderr)
	}
	return renderCppFunclenText(report, stdout, stderr)
}

func runCppInstability(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("instability", flag.ContinueOnError)
	minGap := fs.Float64("min-gap", 0, "minimum gap threshold for violations")
	format := fs.String("format", "text", "output format: text or json")
	excludeFile := fs.String("exclude-file", "", "comma-separated glob patterns for files to exclude")
	debug := fs.Bool("debug", false, "enable debug output")
	excludeFunc := fs.String("exclude-func", "", "unused (instability has no function-level concept)")

	paths, excludeFiles, _, err := resolveArgs(fs, args, excludeFile, excludeFunc)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	opts := cppinstability.Options{
		ExcludeFiles: excludeFiles,
		Debug:        *debug,
	}

	report, err := cppinstability.Check(paths, *minGap, opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	if *format == "json" {
		return renderInstabilityJSON(report, stdout, stderr)
	}
	return renderInstabilityText(report, stdout, stderr)
}
