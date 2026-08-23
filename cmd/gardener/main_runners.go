package main

import (
	"flag"
	"fmt"
	"io"

	"gardener-go/internal/cppfunclen"
	"gardener-go/internal/crap"
	"gardener-go/internal/filelen"
	"gardener-go/internal/gofunclen"
)

func runGoFunclen(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gofunclen", flag.ContinueOnError)
	maxLines := fs.Int("max-lines", 50, "maximum function length in lines")
	format := fs.String("format", "text", "output format: text or json")
	excludeFile := fs.String("exclude-file", "", "comma-separated glob patterns for files to exclude")
	excludeFunc := fs.String("exclude-func", "", "comma-separated glob patterns for functions to exclude")
	debug := fs.Bool("debug", false, "include excluded files and functions in output")

	paths, excludeFiles, excludeFuncs, err := resolveArgs(fs, args, excludeFile, excludeFunc)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	opts := gofunclen.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        *debug,
	}

	report, err := gofunclen.Check(paths, *maxLines, opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	if *format == "json" {
		return renderJSON(report, stdout, stderr)
	}
	return renderText(report, stdout, stderr)
}

func runGoCrap(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("crap", flag.ContinueOnError)
	threshold := fs.Float64("threshold", 6.0, "CRAP score threshold")
	format := fs.String("format", "text", "output format: text or json")
	excludeFile := fs.String("exclude-file", "", "comma-separated glob patterns for files to exclude")
	excludeFunc := fs.String("exclude-func", "", "comma-separated glob patterns for functions to exclude")
	debug := fs.Bool("debug", false, "include excluded files and functions in output")

	paths, excludeFiles, excludeFuncs, err := resolveArgs(fs, args, excludeFile, excludeFunc)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	opts := crap.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        *debug,
	}

	report, err := crap.Check(paths, *threshold, opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	if *format == "json" {
		return renderCrapJSON(report, stdout, stderr)
	}
	return renderCrapText(report, stdout, stderr)
}

func runGoFilelen(args []string, stdout, stderr io.Writer) int {
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

	report, err := filelen.Check(paths, *maxLines, []string{".go"}, opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	if *format == "json" {
		return renderFilelenJSON(report, stdout, stderr)
	}
	return renderFilelenText(report, stdout, stderr)
}

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

func runGoAll(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("all", flag.ContinueOnError)
	format := fs.String("format", "text", "output format: text or json")
	excludeFile := fs.String("exclude-file", "", "comma-separated glob patterns for files to exclude")
	excludeFunc := fs.String("exclude-func", "", "comma-separated glob patterns for functions to exclude")
	debug := fs.Bool("debug", false, "include excluded files and functions in output")

	paths, excludeFiles, excludeFuncs, err := resolveArgs(fs, args, excludeFile, excludeFunc)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	gofunclenReport, crapReport, filelenReport, err := checkAll(paths, excludeFiles, excludeFuncs, *debug)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	combined := combinedReport{
		Gofunclen: gofunclenReport,
		Crap:      crapReport,
		Filelen:   filelenReport,
	}

	// Render output
	if *format == "json" {
		return renderAllJSON(combined, stdout, stderr)
	}
	return renderAllText(combined, stdout, stderr)
}

// checkAll runs the gofunclen, crap, and filelen checks with shared options.
func checkAll(paths []string, excludeFiles, excludeFuncs []string, debug bool) (gofunclen.Report, crap.Report, filelen.Report, error) {
	opts := gofunclen.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        debug,
	}
	crapOpts := crap.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        debug,
	}
	filelenOpts := filelen.Options{
		ExcludeFiles: excludeFiles,
		Debug:        debug,
	}

	gofunclenReport, err := gofunclen.Check(paths, 50, opts)
	if err != nil {
		return gofunclen.Report{}, crap.Report{}, filelen.Report{}, err
	}

	crapReport, err := crap.Check(paths, 6.0, crapOpts)
	if err != nil {
		return gofunclen.Report{}, crap.Report{}, filelen.Report{}, err
	}

	filelenReport, err := filelen.Check(paths, 300, []string{".go"}, filelenOpts)
	if err != nil {
		return gofunclen.Report{}, crap.Report{}, filelen.Report{}, err
	}

	return gofunclenReport, crapReport, filelenReport, nil
}
