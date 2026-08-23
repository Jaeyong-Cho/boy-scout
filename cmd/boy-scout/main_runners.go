package main

import (
	"flag"
	"fmt"
	"io"

	"boy-scout/internal/abstractness"
	"boy-scout/internal/crap"
	"boy-scout/internal/duplication"
	"boy-scout/internal/filelen"
	"boy-scout/internal/gofunclen"
	"boy-scout/internal/instability"
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
	threshold := fs.Float64("threshold", 30.0, "CRAP score threshold")
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

func runGoDuplication(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("duplication", flag.ContinueOnError)
	minLines := fs.Int("min-lines", 5, "minimum function length in lines to compare")
	minSimilarity := fs.Float64("min-similarity", 0.70, "minimum LCS-based similarity ratio for Type-3 detection (0.0-1.0)")
	format := fs.String("format", "text", "output format: text or json")
	excludeFile := fs.String("exclude-file", "", "comma-separated glob patterns for files to exclude")
	excludeFunc := fs.String("exclude-func", "", "comma-separated glob patterns for functions to exclude")
	debug := fs.Bool("debug", false, "include excluded files and functions in output")

	paths, excludeFiles, excludeFuncs, err := resolveArgs(fs, args, excludeFile, excludeFunc)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	// Validate minSimilarity range
	if *minSimilarity < 0.0 || *minSimilarity > 1.0 {
		fmt.Fprintf(stderr, "error: --min-similarity must be in range [0.0, 1.0], got %v\n", *minSimilarity)
		return 2
	}

	opts := duplication.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        *debug,
	}

	report, err := duplication.CheckWithSimilarity(paths, *minLines, *minSimilarity, opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	if *format == "json" {
		return renderDuplicationJSON(report, stdout, stderr)
	}
	return renderDuplicationText(report, stdout, stderr)
}

func runGoInstability(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("instability", flag.ContinueOnError)
	minGap := fs.Float64("min-gap", 0, "minimum gap to report (violations with Gap > min-gap)")
	format := fs.String("format", "text", "output format: text or json")
	excludeFile := fs.String("exclude-file", "", "comma-separated glob patterns for files to exclude")
	excludeFunc := fs.String("exclude-func", "", "unused (instability has no function-level concept)")
	debug := fs.Bool("debug", false, "unused")

	paths, excludeFiles, _, err := resolveArgs(fs, args, excludeFile, excludeFunc)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	opts := instability.Options{
		ExcludeFiles: excludeFiles,
		Debug:        *debug,
	}

	report, err := instability.Check(paths, *minGap, opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	if *format == "json" {
		return renderInstabilityJSON(report, stdout, stderr)
	}
	return renderInstabilityText(report, stdout, stderr)
}

func runGoAbstractness(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("abstractness", flag.ContinueOnError)
	minDistance := fs.Float64("min-distance", 0.5, "minimum distance from main sequence to report (packages with |signedD| > min-distance)")
	minSurfaceRatio := fs.Float64("min-surface-ratio", 0.5, "minimum surface ratio for Pain zone candidates (ratio of exported to total declarations; only applies to Pain, not Uselessness)")
	ignoreDeepModuleGate := fs.Bool("ignore-deep-module-gate", false, "if true, flag all Pain candidates regardless of surface ratio (Story 2 behavior)")
	format := fs.String("format", "text", "output format: text or json")
	excludeFile := fs.String("exclude-file", "", "comma-separated glob patterns for files to exclude")
	excludeFunc := fs.String("exclude-func", "", "unused (abstractness has no function-level concept)")
	debug := fs.Bool("debug", false, "unused")

	paths, excludeFiles, _, err := resolveArgs(fs, args, excludeFile, excludeFunc)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	opts := abstractness.Options{
		ExcludeFiles:         excludeFiles,
		Debug:                *debug,
		IgnoreDeepModuleGate: *ignoreDeepModuleGate,
	}

	report, err := abstractness.CheckWithSurfaceRatio(paths, *minDistance, *minSurfaceRatio, opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	if *format == "json" {
		return renderAbstractnessJSON(report, stdout, stderr)
	}
	return renderAbstractnessText(report, stdout, stderr)
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

	gofunclenReport, crapReport, filelenReport, duplicationReport, instabilityReport, abstractnessReport, err := checkAll(paths, excludeFiles, excludeFuncs, *debug)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	combined := combinedReport{
		Gofunclen:   gofunclenReport,
		Crap:        crapReport,
		Filelen:     filelenReport,
		Duplication: duplicationReport,
		Instability: instabilityReport,
		Abstractness: abstractnessReport,
	}

	// Render output
	if *format == "json" {
		return renderAllJSON(combined, stdout, stderr)
	}
	return renderAllText(combined, stdout, stderr)
}

// buildCheckClosures constructs check functions that populate the reports
func buildCheckClosures(paths []string, excludeFiles, excludeFuncs []string, debug bool, reports *struct {
	gofunclen   *gofunclen.Report
	crap        *crap.Report
	filelen     *filelen.Report
	duplication *duplication.Report
	instability *instability.Report
	abstractness *abstractness.Report
}) []func() error {
	return []func() error{
		func() error {
			var err error
			*reports.gofunclen, err = checkAllGofunclen(paths, excludeFiles, excludeFuncs, debug)
			return err
		},
		func() error {
			var err error
			*reports.crap, err = checkAllCrap(paths, excludeFiles, excludeFuncs, debug)
			return err
		},
		func() error {
			var err error
			*reports.filelen, err = checkAllFilelen(paths, excludeFiles, debug)
			return err
		},
		func() error {
			var err error
			*reports.duplication, err = checkAllDuplication(paths, excludeFiles, excludeFuncs, debug)
			return err
		},
		func() error {
			var err error
			*reports.instability, err = checkAllInstability(paths, excludeFiles, debug)
			return err
		},
		func() error {
			var err error
			*reports.abstractness, err = checkAllAbstractness(paths, excludeFiles, debug)
			return err
		},
	}
}

// checkAll runs all checks with shared options.
func checkAll(paths []string, excludeFiles, excludeFuncs []string, debug bool) (gofunclen.Report, crap.Report, filelen.Report, duplication.Report, instability.Report, abstractness.Report, error) {
	var (
		gofunclenReport   gofunclen.Report
		crapReport        crap.Report
		filelenReport     filelen.Report
		duplicationReport duplication.Report
		instabilityReport instability.Report
		abstractnessReport abstractness.Report
	)

	reports := &struct {
		gofunclen   *gofunclen.Report
		crap        *crap.Report
		filelen     *filelen.Report
		duplication *duplication.Report
		instability *instability.Report
		abstractness *abstractness.Report
	}{
		gofunclen:    &gofunclenReport,
		crap:         &crapReport,
		filelen:      &filelenReport,
		duplication:  &duplicationReport,
		instability:  &instabilityReport,
		abstractness: &abstractnessReport,
	}

	checks := buildCheckClosures(paths, excludeFiles, excludeFuncs, debug, reports)
	for _, check := range checks {
		if err := check(); err != nil {
			return gofunclenReport, crapReport, filelenReport, duplicationReport, instabilityReport, abstractnessReport, err
		}
	}

	return gofunclenReport, crapReport, filelenReport, duplicationReport, instabilityReport, abstractnessReport, nil
}

func checkAllGofunclen(paths []string, excludeFiles, excludeFuncs []string, debug bool) (gofunclen.Report, error) {
	opts := gofunclen.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        debug,
	}
	return gofunclen.Check(paths, 50, opts)
}

func checkAllCrap(paths []string, excludeFiles, excludeFuncs []string, debug bool) (crap.Report, error) {
	opts := crap.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        debug,
	}
	return crap.Check(paths, 6.0, opts)
}

func checkAllFilelen(paths []string, excludeFiles []string, debug bool) (filelen.Report, error) {
	opts := filelen.Options{
		ExcludeFiles: excludeFiles,
		Debug:        debug,
	}
	return filelen.Check(paths, 300, []string{".go"}, opts)
}

func checkAllDuplication(paths []string, excludeFiles, excludeFuncs []string, debug bool) (duplication.Report, error) {
	opts := duplication.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        debug,
	}
	return duplication.Check(paths, 5, opts)
}

func checkAllInstability(paths []string, excludeFiles []string, debug bool) (instability.Report, error) {
	opts := instability.Options{
		ExcludeFiles: excludeFiles,
		Debug:        debug,
	}
	return instability.Check(paths, 0, opts)
}

func checkAllAbstractness(paths []string, excludeFiles []string, debug bool) (abstractness.Report, error) {
	opts := abstractness.Options{
		ExcludeFiles: excludeFiles,
		Debug:        debug,
	}
	return abstractness.Check(paths, 0.5, opts)
}
