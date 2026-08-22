package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gardener-go/internal/cppfunclen"
	"gardener-go/internal/crap"
	"gardener-go/internal/gofunclen"
	"gardener-go/internal/setup"
)

func assertf(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}

func init() {
	// Invariant: every entry in setupTargets has a non-empty name and prefix
	assertf(len(setupTargets) == 4, "setupTargets must have exactly 4 entries")
	for i, t := range setupTargets {
		assertf(t.name != "", "setupTargets[%d].name must not be empty", i)
		assertf(t.prefix != "", "setupTargets[%d].prefix must not be empty", i)
	}
}

// parsePatterns returns nil if s is empty, else splits on comma and drops empty segments.
// setupTarget maps a target name (claude, copilot, pi, agents) to its directory prefix.
type setupTarget struct {
	name   string
	prefix string
}

// setupTargets is the ordered list of valid setup targets.
var setupTargets = []setupTarget{
	{"claude", ".claude"},
	{"copilot", ".copilot"},
	{"pi", ".pi/agent"},
	{"agents", ".agents"},
}

// setupTargetNames returns a slice of valid target names.
func setupTargetNames() []string {
	names := make([]string, len(setupTargets))
	for i, t := range setupTargets {
		names[i] = t.name
	}
	return names
}

// lookupSetupTarget looks up a target by name and returns its prefix and whether it was found.
func lookupSetupTarget(name string) (prefix string, ok bool) {
	for _, t := range setupTargets {
		if t.name == name {
			return t.prefix, true
		}
	}
	return "", false
}

// printTargetPrompt writes the numbered target list and input prompt to stdout.
func printTargetPrompt(stdout io.Writer) {
	fmt.Fprintf(stdout, "Select a target:\n")
	for i, t := range setupTargets {
		fmt.Fprintf(stdout, "%d) %s\n", i+1, t.name)
	}
	fmt.Fprintf(stdout, "> ")
}

// parseTargetSelection resolves one line of raw input to a target name,
// matching either by 1-based index or by name.
func parseTargetSelection(input string) (name string, err error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("empty input; valid targets: %v", setupTargetNames())
	}

	// Try to parse as 1-based index
	if idx, err := strconv.Atoi(input); err == nil {
		if idx >= 1 && idx <= len(setupTargets) {
			return setupTargets[idx-1].name, nil
		}
		return "", fmt.Errorf("invalid index %d; valid targets: %v", idx, setupTargetNames())
	}

	// Try to match by name
	if _, ok := lookupSetupTarget(input); ok {
		return input, nil
	}

	return "", fmt.Errorf("unknown target %q; valid targets: %v", input, setupTargetNames())
}

// promptForTarget prints a numbered list of targets to stdout, reads one line from stdin,
// and returns the target name (either by 1-based index or name match).
// Returns an error if the input is invalid, empty, or EOF.
func promptForTarget(stdin io.Reader, stdout io.Writer) (name string, err error) {
	printTargetPrompt(stdout)

	scanner := bufio.NewScanner(stdin)
	if !scanner.Scan() {
		// EOF or scan error (empty input, no newline, etc.)
		return "", fmt.Errorf("no input provided; valid targets: %v", setupTargetNames())
	}

	return parseTargetSelection(scanner.Text())
}

// isInteractive checks if the given reader is a terminal (real TTY).
func isInteractive(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// resolveTarget determines the target name from positional args or by prompting.
// When interactive, it prompts the user. When not interactive, it returns an error.
func resolveTarget(positional []string, stdin io.Reader, interactive bool, stdout io.Writer) (name string, prefix string, err error) {
	if len(positional) > 0 {
		// Explicit target provided
		targetName := positional[0]
		prefix, ok := lookupSetupTarget(targetName)
		if !ok {
			return "", "", fmt.Errorf("unknown target %q; valid targets: %v", targetName, setupTargetNames())
		}
		return targetName, prefix, nil
	}

	// No target provided; need to ask
	if interactive {
		targetName, err := promptForTarget(stdin, stdout)
		if err != nil {
			return "", "", err
		}
		prefix, ok := lookupSetupTarget(targetName)
		assertf(ok, "promptForTarget returned unknown target %q", targetName)
		return targetName, prefix, nil
	}

	// Non-interactive with no target
	return "", "", fmt.Errorf("no target provided; valid targets: %v", setupTargetNames())
}

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
		"gofunclen": runGoFunclen,
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
		fmt.Fprintln(stderr, "gardener setup [claude|copilot|pi|agents] [--global]")
		return 2
	}

	// Handle setup separately (lang-less)
	if args[0] == "setup" {
		return runSetup(args[1:], os.Stdin, stdout, stderr)
	}

	return dispatchLang(args[0], args[1:], stdout, stderr)
}

// dispatchLang looks up the subcommand handler for lang and runs it with the
// remaining args, printing usage/error messages to stderr as needed.
func dispatchLang(lang string, args []string, stdout, stderr io.Writer) int {
	subcommands, ok := langSubcommands[lang]
	if !ok {
		fmt.Fprintf(stderr, "unknown language: %s\n", lang)
		return 2
	}

	if len(args) < 1 {
		fmt.Fprintf(stderr, "usage: gardener %s <command> [options] [paths...]\n", lang)
		return 2
	}

	subcommand := args[0]
	fn, ok := subcommands[subcommand]
	if !ok {
		fmt.Fprintf(stderr, "unknown subcommand for %s: %s\n", lang, subcommand)
		return 2
	}

	assertf(fn != nil, "registered subcommand handler for %s/%s is nil", lang, subcommand)
	return fn(args[1:], stdout, stderr)
}

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
	Gofunclen gofunclen.Report `json:"gofunclen"`
	Crap      crap.Report      `json:"crap"`
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

	gofunclenReport, crapReport, err := checkAll(paths, excludeFiles, excludeFuncs, *debug)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	combined := combinedReport{
		Gofunclen: gofunclenReport,
		Crap:      crapReport,
	}

	// Render output
	if *format == "json" {
		return renderAllJSON(combined, stdout, stderr)
	}
	return renderAllText(combined, stdout, stderr)
}

// checkAll runs both the gofunclen and crap checks with shared options.
func checkAll(paths []string, excludeFiles, excludeFuncs []string, debug bool) (gofunclen.Report, crap.Report, error) {
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

	gofunclenReport, err := gofunclen.Check(paths, 50, opts)
	if err != nil {
		return gofunclen.Report{}, crap.Report{}, err
	}

	crapReport, err := crap.Check(paths, 6.0, crapOpts)
	if err != nil {
		return gofunclen.Report{}, crap.Report{}, err
	}

	return gofunclenReport, crapReport, nil
}

func renderAllText(report combinedReport, stdout, stderr io.Writer) int {
	writeGofunclenLines(stdout, "[gofunclen] ", report.Gofunclen)
	writeCrapLines(stdout, "[crap] ", report.Crap)

	totalViolations := len(report.Gofunclen.Violations) + len(report.Crap.Violations)
	totalSkipped := len(report.Gofunclen.Skipped) + len(report.Crap.Skipped)

	return exitCodeFor(totalViolations, totalSkipped)
}

func renderAllJSON(report combinedReport, stdout, stderr io.Writer) int {
	data, err := json.Marshal(report)
	assertf(err == nil, "json.Marshal failed: %v", err)

	fmt.Fprintf(stdout, "%s\n", string(data))

	totalViolations := len(report.Gofunclen.Violations) + len(report.Crap.Violations)
	totalSkipped := len(report.Gofunclen.Skipped) + len(report.Crap.Skipped)

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

// parseSetupArgs manually scans args for a --global flag and collects positional
// arguments, since --global may appear in any position and flag.Parse() alone
// can't express that. Returns an error (with any usage message already written
// to stderr) if an unknown flag or more than one positional argument is found.
func parseSetupArgs(args []string, stderr io.Writer) (global bool, positional []string, err error) {
	for _, arg := range args {
		if arg == "--global" || arg == "-global" {
			global = true
		} else if strings.HasPrefix(arg, "-") {
			fmt.Fprintf(stderr, "unknown flag: %s\n", arg)
			fmt.Fprintln(stderr, "usage: gardener setup [claude|copilot|pi|agents] [--global]")
			return false, nil, fmt.Errorf("unknown flag: %s", arg)
		} else {
			positional = append(positional, arg)
		}
	}

	if len(positional) > 1 {
		fmt.Fprintln(stderr, "usage: gardener setup [claude|copilot|pi|agents] [--global]")
		return false, nil, fmt.Errorf("too many positional arguments")
	}

	return global, positional, nil
}

func runSetup(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	global, positional, err := parseSetupArgs(args, stderr)
	if err != nil {
		return 2
	}

	baseDir, exePath, err := prepareSetupArgs(global)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}
	assertf(baseDir != "", "baseDir must not be empty")

	// Resolve the target (either explicit or by prompting)
	_, prefix, err := resolveTarget(positional, stdin, isInteractive(stdin), stdout)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	path, err := setup.Run(baseDir, exePath, prefix)
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

// writeGofunclenLines writes a gofunclen report's violations and excluded entries to w,
// each line prefixed with prefix (e.g. "[gofunclen] " when combined with other checks).
func writeGofunclenLines(w io.Writer, prefix string, report gofunclen.Report) {
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

func renderText(report gofunclen.Report, stdout, stderr io.Writer) int {
	writeGofunclenLines(stdout, "", report)
	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func renderJSON(report gofunclen.Report, stdout, stderr io.Writer) int {
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
