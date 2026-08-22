package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go-gardener/internal/crap"
	"go-gardener/internal/funclen"
)

func assertf(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}

// parsePatterns returns nil if s is empty, else splits on comma and drops empty segments.
func parsePatterns(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		if p := strings.TrimSpace(p); p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// validatePatterns checks that each pattern is a valid glob.
func validatePatterns(patterns []string) error {
	for _, p := range patterns {
		if _, err := filepath.Match(p, ""); err != nil {
			return fmt.Errorf("invalid exclude pattern %q: %w", p, err)
		}
	}
	return nil
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: gardener <subcommand> [options] [paths...]")
		fmt.Fprintln(stderr, "subcommands: funclen, crap, all")
		return 2
	}

	subcommand := args[0]

	switch subcommand {
	case "funclen":
		return runFunclen(args[1:], stdout, stderr)
	case "crap":
		return runCrap(args[1:], stdout, stderr)
	case "all":
		return runAll(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown subcommand: %s\n", subcommand)
		return 2
	}
}

func runFunclen(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("funclen", flag.ContinueOnError)
	maxLines := fs.Int("max-lines", 50, "maximum function length in lines")
	format := fs.String("format", "text", "output format: text or json")
	excludeFile := fs.String("exclude-file", "", "comma-separated glob patterns for files to exclude")
	excludeFunc := fs.String("exclude-func", "", "comma-separated glob patterns for functions to exclude")
	debug := fs.Bool("debug", false, "include excluded files and functions in output")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "parse error: %v\n", err)
		return 2
	}

	paths := fs.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// Parse and validate patterns
	excludeFiles := parsePatterns(*excludeFile)
	excludeFuncs := parsePatterns(*excludeFunc)
	if err := validatePatterns(excludeFiles); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}
	if err := validatePatterns(excludeFuncs); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	opts := funclen.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        *debug,
	}

	report, err := funclen.Check(paths, *maxLines, opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	// Render output
	if *format == "json" {
		return renderJSON(report, stdout, stderr)
	} else {
		return renderText(report, stdout, stderr)
	}
}

func runCrap(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("crap", flag.ContinueOnError)
	threshold := fs.Float64("threshold", 6.0, "CRAP score threshold")
	format := fs.String("format", "text", "output format: text or json")
	excludeFile := fs.String("exclude-file", "", "comma-separated glob patterns for files to exclude")
	excludeFunc := fs.String("exclude-func", "", "comma-separated glob patterns for functions to exclude")
	debug := fs.Bool("debug", false, "include excluded files and functions in output")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "parse error: %v\n", err)
		return 2
	}

	paths := fs.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// Parse and validate patterns
	excludeFiles := parsePatterns(*excludeFile)
	excludeFuncs := parsePatterns(*excludeFunc)
	if err := validatePatterns(excludeFiles); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}
	if err := validatePatterns(excludeFuncs); err != nil {
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

	// Render output
	if *format == "json" {
		return renderCrapJSON(report, stdout, stderr)
	} else {
		return renderCrapText(report, stdout, stderr)
	}
}

func renderCrapText(report crap.Report, stdout, stderr io.Writer) int {
	for _, v := range report.Violations {
		fmt.Fprintf(stdout, "%s:%d: function %s has CRAP score %.2f (complexity=%d, coverage=%.1f%%, threshold=%.2f)\n",
			v.File, v.Line, v.Func, v.Score, v.Complexity, v.Coverage*100, v.Threshold)
	}

	// Render excluded files
	for _, f := range report.ExcludedFiles {
		fmt.Fprintf(stdout, "excluded file: %s\n", f)
	}

	// Render excluded functions
	for _, exc := range report.ExcludedFuncs {
		fmt.Fprintf(stdout, "%s:%d: function %s excluded (%s)\n",
			exc.File, exc.Line, exc.Func, exc.Reason)
	}

	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func renderCrapJSON(report crap.Report, stdout, stderr io.Writer) int {
	data, err := json.Marshal(report)
	assertf(err == nil, "json.Marshal failed: %v", err)

	fmt.Fprintf(stdout, "%s\n", string(data))

	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

type combinedReport struct {
	Funclen funclen.Report `json:"funclen"`
	Crap    crap.Report    `json:"crap"`
}

func runAll(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("all", flag.ContinueOnError)
	format := fs.String("format", "text", "output format: text or json")
	excludeFile := fs.String("exclude-file", "", "comma-separated glob patterns for files to exclude")
	excludeFunc := fs.String("exclude-func", "", "comma-separated glob patterns for functions to exclude")
	debug := fs.Bool("debug", false, "include excluded files and functions in output")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "parse error: %v\n", err)
		return 2
	}

	paths := fs.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// Parse and validate patterns
	excludeFiles := parsePatterns(*excludeFile)
	excludeFuncs := parsePatterns(*excludeFunc)
	if err := validatePatterns(excludeFiles); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}
	if err := validatePatterns(excludeFuncs); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	opts := funclen.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        *debug,
	}
	crapOpts := crap.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        *debug,
	}

	// Run funclen check
	funclenReport, err := funclen.Check(paths, 50, opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	// Run crap check
	crapReport, err := crap.Check(paths, 6.0, crapOpts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	// Combine into a single report
	combined := combinedReport{
		Funclen: funclenReport,
		Crap:    crapReport,
	}

	// Render output
	if *format == "json" {
		return renderAllJSON(combined, stdout, stderr)
	} else {
		return renderAllText(combined, stdout, stderr)
	}
}

func renderAllText(report combinedReport, stdout, stderr io.Writer) int {
	// Render funclen violations and excluded
	for _, v := range report.Funclen.Violations {
		fmt.Fprintf(stdout, "[funclen] %s:%d: function %s is %d lines (limit %d)\n",
			v.File, v.Line, v.Func, v.Length, v.Limit)
	}
	for _, f := range report.Funclen.ExcludedFiles {
		fmt.Fprintf(stdout, "[funclen] excluded file: %s\n", f)
	}
	for _, exc := range report.Funclen.ExcludedFuncs {
		fmt.Fprintf(stdout, "[funclen] %s:%d: function %s excluded (%s)\n",
			exc.File, exc.Line, exc.Func, exc.Reason)
	}

	// Render crap violations and excluded
	for _, v := range report.Crap.Violations {
		fmt.Fprintf(stdout, "[crap] %s:%d: function %s has CRAP score %.2f (complexity=%d, coverage=%.1f%%, threshold=%.2f)\n",
			v.File, v.Line, v.Func, v.Score, v.Complexity, v.Coverage*100, v.Threshold)
	}
	for _, f := range report.Crap.ExcludedFiles {
		fmt.Fprintf(stdout, "[crap] excluded file: %s\n", f)
	}
	for _, exc := range report.Crap.ExcludedFuncs {
		fmt.Fprintf(stdout, "[crap] %s:%d: function %s excluded (%s)\n",
			exc.File, exc.Line, exc.Func, exc.Reason)
	}

	totalViolations := len(report.Funclen.Violations) + len(report.Crap.Violations)
	totalSkipped := len(report.Funclen.Skipped) + len(report.Crap.Skipped)

	return exitCodeFor(totalViolations, totalSkipped)
}

func renderAllJSON(report combinedReport, stdout, stderr io.Writer) int {
	data, err := json.Marshal(report)
	assertf(err == nil, "json.Marshal failed: %v", err)

	fmt.Fprintf(stdout, "%s\n", string(data))

	totalViolations := len(report.Funclen.Violations) + len(report.Crap.Violations)
	totalSkipped := len(report.Funclen.Skipped) + len(report.Crap.Skipped)

	return exitCodeFor(totalViolations, totalSkipped)
}

// exitCodeFor computes the exit code based on violation and skipped file counts.
// Skipped/fatal errors take priority (code 2), then violations (code 1), then clean (code 0).
func exitCodeFor(numViolations, numSkipped int) int {
	code := 0
	if numSkipped > 0 {
		code = 2
	} else if numViolations > 0 {
		code = 1
	}

	assertf(code == 0 || code == 1 || code == 2, "unexpected exit code %d", code)
	return code
}

func renderText(report funclen.Report, stdout, stderr io.Writer) int {
	for _, v := range report.Violations {
		fmt.Fprintf(stdout, "%s:%d: function %s is %d lines (limit %d)\n",
			v.File, v.Line, v.Func, v.Length, v.Limit)
	}

	// Render excluded files
	for _, f := range report.ExcludedFiles {
		fmt.Fprintf(stdout, "excluded file: %s\n", f)
	}

	// Render excluded functions
	for _, exc := range report.ExcludedFuncs {
		fmt.Fprintf(stdout, "%s:%d: function %s excluded (%s)\n",
			exc.File, exc.Line, exc.Func, exc.Reason)
	}

	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func renderJSON(report funclen.Report, stdout, stderr io.Writer) int {
	data, err := json.Marshal(report)
	assertf(err == nil, "json.Marshal failed: %v", err)

	fmt.Fprintf(stdout, "%s\n", string(data))

	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
