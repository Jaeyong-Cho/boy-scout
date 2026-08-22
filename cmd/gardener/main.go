package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gardener-go/internal/cppfunclen"
	"gardener-go/internal/crap"
	"gardener-go/internal/funclen"
	"gardener-go/internal/setup"
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

// resolveArgs parses fs, defaults paths to "." when none are given, and parses/validates
// the --exclude-file/--exclude-func flag values, collapsing the usual four-step
// dance (parse, default paths, parse excludes, validate excludes) into one error check.
func resolveArgs(fs *flag.FlagSet, args []string, excludeFile, excludeFunc *string) (paths, excludeFiles, excludeFuncs []string, err error) {
	if err := fs.Parse(args); err != nil {
		return nil, nil, nil, fmt.Errorf("parse error: %w", err)
	}

	paths = fs.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	excludeFiles, excludeFuncs, err = excludePatternsFrom(*excludeFile, *excludeFunc)
	if err != nil {
		return nil, nil, nil, err
	}
	return paths, excludeFiles, excludeFuncs, nil
}

// excludePatternsFrom parses and validates the --exclude-file/--exclude-func flag values.
func excludePatternsFrom(excludeFile, excludeFunc string) (excludeFiles, excludeFuncs []string, err error) {
	excludeFiles = parsePatterns(excludeFile)
	excludeFuncs = parsePatterns(excludeFunc)
	if err := validatePatterns(excludeFiles); err != nil {
		return nil, nil, err
	}
	if err := validatePatterns(excludeFuncs); err != nil {
		return nil, nil, err
	}
	return excludeFiles, excludeFuncs, nil
}

// langSubcommands maps each language to its subcommands.
var langSubcommands = map[string]map[string]func(args []string, stdout, stderr io.Writer) int{
	"go": {
		"funclen": runGoFunclen,
		"crap":    runGoCrap,
		"all":     runGoAll,
	},
	"cpp": {
		"funclen": runCppFunclen,
	},
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: gardener <lang> <command> [options] [paths...]")
		fmt.Fprintln(stderr, "languages: go")
		fmt.Fprintln(stderr, "gardener setup [--global]")
		return 2
	}

	// Handle setup separately (lang-less)
	if args[0] == "setup" {
		return runSetup(args[1:], stdout, stderr)
	}

	// Dispatch by language
	lang := args[0]
	subcommands, ok := langSubcommands[lang]
	if !ok {
		fmt.Fprintf(stderr, "unknown language: %s\n", lang)
		return 2
	}

	if len(args) < 2 {
		fmt.Fprintf(stderr, "usage: gardener %s <command> [options] [paths...]\n", lang)
		return 2
	}

	subcommand := args[1]
	fn, ok := subcommands[subcommand]
	if !ok {
		fmt.Fprintf(stderr, "unknown subcommand for %s: %s\n", lang, subcommand)
		return 2
	}

	assertf(fn != nil, "registered subcommand handler for %s/%s is nil", lang, subcommand)
	return fn(args[2:], stdout, stderr)
}

func runGoFunclen(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("funclen", flag.ContinueOnError)
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

// writeCrapLines writes a crap report's violations and excluded entries to w,
// each line prefixed with prefix (e.g. "[crap] " when combined with other checks).
func writeCrapLines(w io.Writer, prefix string, report crap.Report) {
	for _, v := range report.Violations {
		fmt.Fprintf(w, "%s%s:%d: function %s has CRAP score %.2f (complexity=%d, coverage=%.1f%%, threshold=%.2f)\n",
			prefix, v.File, v.Line, v.Func, v.Score, v.Complexity, v.Coverage*100, v.Threshold)
	}
	for _, f := range report.ExcludedFiles {
		fmt.Fprintf(w, "%sexcluded file: %s\n", prefix, f)
	}
	for _, exc := range report.ExcludedFuncs {
		fmt.Fprintf(w, "%s%s:%d: function %s excluded (%s)\n",
			prefix, exc.File, exc.Line, exc.Func, exc.Reason)
	}
}

func renderCrapText(report crap.Report, stdout, stderr io.Writer) int {
	writeCrapLines(stdout, "", report)
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

	funclenReport, crapReport, err := checkAll(paths, excludeFiles, excludeFuncs, *debug)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	combined := combinedReport{
		Funclen: funclenReport,
		Crap:    crapReport,
	}

	// Render output
	if *format == "json" {
		return renderAllJSON(combined, stdout, stderr)
	}
	return renderAllText(combined, stdout, stderr)
}

// checkAll runs both the funclen and crap checks with shared options.
func checkAll(paths []string, excludeFiles, excludeFuncs []string, debug bool) (funclen.Report, crap.Report, error) {
	opts := funclen.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        debug,
	}
	crapOpts := crap.Options{
		ExcludeFiles: excludeFiles,
		ExcludeFuncs: excludeFuncs,
		Debug:        debug,
	}

	funclenReport, err := funclen.Check(paths, 50, opts)
	if err != nil {
		return funclen.Report{}, crap.Report{}, err
	}

	crapReport, err := crap.Check(paths, 6.0, crapOpts)
	if err != nil {
		return funclen.Report{}, crap.Report{}, err
	}

	return funclenReport, crapReport, nil
}

func renderAllText(report combinedReport, stdout, stderr io.Writer) int {
	writeFunclenLines(stdout, "[funclen] ", report.Funclen)
	writeCrapLines(stdout, "[crap] ", report.Crap)

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

// prepareSetupArgs resolves the install base directory (home dir if global, else
// cwd) and the current executable path needed by setup.Run.
func prepareSetupArgs(global bool) (baseDir, exePath string, err error) {
	if global {
		baseDir, err = os.UserHomeDir()
	} else {
		baseDir = "."
	}
	if err != nil {
		return "", "", fmt.Errorf("failed to get home directory: %w", err)
	}

	exePath, err = os.Executable()
	if err != nil {
		return "", "", fmt.Errorf("failed to get executable path: %w", err)
	}

	return baseDir, exePath, nil
}

func runSetup(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	global := fs.Bool("global", false, "install to ~/.agents/skills/gardener instead of ./.agents/skills/gardener")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "parse error: %v\n", err)
		return 2
	}

	// Reject extra positional arguments
	if len(fs.Args()) > 0 {
		fmt.Fprintln(stderr, "usage: gardener setup [--global]")
		return 2
	}

	baseDir, exePath, err := prepareSetupArgs(*global)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}
	assertf(baseDir != "", "baseDir must not be empty")

	path, err := setup.Run(baseDir, exePath)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	fmt.Fprintf(stdout, "gardener skill installed: %s\n", path)
	return 0
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

// writeFunclenLines writes a funclen report's violations and excluded entries to w,
// each line prefixed with prefix (e.g. "[funclen] " when combined with other checks).
func writeFunclenLines(w io.Writer, prefix string, report funclen.Report) {
	for _, v := range report.Violations {
		fmt.Fprintf(w, "%s%s:%d: function %s is %d lines (limit %d)\n",
			prefix, v.File, v.Line, v.Func, v.Length, v.Limit)
	}
	for _, f := range report.ExcludedFiles {
		fmt.Fprintf(w, "%sexcluded file: %s\n", prefix, f)
	}
	for _, exc := range report.ExcludedFuncs {
		fmt.Fprintf(w, "%s%s:%d: function %s excluded (%s)\n",
			prefix, exc.File, exc.Line, exc.Func, exc.Reason)
	}
}

func renderText(report funclen.Report, stdout, stderr io.Writer) int {
	writeFunclenLines(stdout, "", report)
	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func renderJSON(report funclen.Report, stdout, stderr io.Writer) int {
	data, err := json.Marshal(report)
	assertf(err == nil, "json.Marshal failed: %v", err)

	fmt.Fprintf(stdout, "%s\n", string(data))

	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

// writeCppFunclenLines writes a cpp funclen report's violations and excluded entries to w.
func writeCppFunclenLines(w io.Writer, prefix string, report cppfunclen.Report) {
	for _, v := range report.Violations {
		fmt.Fprintf(w, "%s%s:%d: function %s is %d lines (limit %d)\n",
			prefix, v.File, v.Line, v.Func, v.Length, v.Limit)
	}
	for _, exc := range report.ExcludedFuncs {
		fmt.Fprintf(w, "%s%s: function %s excluded (%s)\n",
			prefix, exc.File, exc.Func, exc.Reason)
	}
}

func renderCppFunclenText(report cppfunclen.Report, stdout, stderr io.Writer) int {
	writeCppFunclenLines(stdout, "", report)
	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func renderCppFunclenJSON(report cppfunclen.Report, stdout, stderr io.Writer) int {
	data, err := json.Marshal(report)
	assertf(err == nil, "json.Marshal failed: %v", err)

	fmt.Fprintf(stdout, "%s\n", string(data))

	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
