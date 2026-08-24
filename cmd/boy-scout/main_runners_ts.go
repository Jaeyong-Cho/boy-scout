package main

import (
	"flag"
	"io"

	"boy-scout/internal/filelen"
	"boy-scout/internal/tsfunclen"
)

func runTsFunclen(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("funclen", flag.ContinueOnError)
	maxLines := fs.Int("max-lines", 50, "maximum function length in lines")
	format := fs.String("format", "text", "output format: text or json")
	excludeFile := fs.String("exclude-file", "", "comma-separated glob patterns for files to exclude")
	excludeFunc := fs.String("exclude-func", "", "comma-separated glob patterns for functions to exclude")
	debug := fs.Bool("debug", false, "include excluded functions in output")

	paths, excludeFiles, excludeFuncs, err := resolveArgs(fs, args, excludeFile, excludeFunc)
	if err != nil {
		reportError(err, stderr)
		return 2
	}

	opts := tsfunclen.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        *debug,
	}

	report, err := tsfunclen.Check(paths, *maxLines, opts)
	if err != nil {
		reportError(err, stderr)
		return 2
	}

	return selectAndRender(format,
		func(stdout, stderr io.Writer) int { return renderTsFunclenJSON(report, stdout, stderr) },
		func(stdout, stderr io.Writer) int { return renderTsFunclenText(report, stdout, stderr) },
		stdout, stderr)
}

func runTsFilelen(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("filelen", flag.ContinueOnError)
	maxLines := fs.Int("max-lines", 300, "maximum file length in lines")
	format := fs.String("format", "text", "output format: text or json")
	excludeFile := fs.String("exclude-file", "", "comma-separated glob patterns for files to exclude")
	excludeFunc := fs.String("exclude-func", "", "unused (filelen has no function-level concept)")
	debug := fs.Bool("debug", false, "include excluded files in output")

	paths, excludeFiles, _, err := resolveArgs(fs, args, excludeFile, excludeFunc)
	if err != nil {
		reportError(err, stderr)
		return 2
	}

	opts := filelen.Options{
		ExcludeFiles: excludeFiles,
		Debug:        *debug,
	}

	report, err := filelen.Check(paths, *maxLines, []string{".ts", ".tsx", ".html", ".css"}, opts)
	if err != nil {
		reportError(err, stderr)
		return 2
	}

	return selectAndRender(format,
		func(stdout, stderr io.Writer) int { return renderFilelenJSON(report, stdout, stderr) },
		func(stdout, stderr io.Writer) int { return renderFilelenText(report, stdout, stderr) },
		stdout, stderr)
}
