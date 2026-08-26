package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"boy-scout/internal/assertutil"
)

// version is the built version of boy-scout, overridable via -ldflags -X main.version=...
var version = "dev"

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
		"gofunclen":   runGoFunclen,
		"complexity":  runGoComplexity,
		"filelen":     runGoFilelen,
		"linelen":     runGoLinelen,
		"duplication": runGoDuplication,
		"all":         runGoAll,
	},
	"cpp": {
		"funclen": runCppFunclen,
		"filelen": runCppFilelen,
		"linelen": runCppLinelen,
	},
	"ts": {
		"funclen": runTsFunclen,
		"filelen": runTsFilelen,
		"linelen": runTsLinelen,
	},
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: boy-scout <lang> <command> [options] [paths...]")
		fmt.Fprintln(stderr, "languages: go, cpp, ts")
		fmt.Fprintln(stderr, "boy-scout setup [claude|copilot|pi|agents] [--global]")
		return 2
	}

	// Handle setup separately (lang-less)
	if args[0] == "setup" {
		return runSetup(args[1:], os.Stdin, stdout, stderr)
	}

	// Handle version separately (lang-less)
	if args[0] == "version" {
		fmt.Fprintf(stdout, "boy-scout %s\n", version)
		return 0
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
		fmt.Fprintf(stderr, "usage: boy-scout %s <command> [options] [paths...]\n", lang)
		return 2
	}

	subcommand := args[0]
	fn, ok := subcommands[subcommand]
	if !ok {
		fmt.Fprintf(stderr, "unknown subcommand for %s: %s\n", lang, subcommand)
		return 2
	}

	assertutil.Assertf(fn != nil, "registered subcommand handler for %s/%s is nil", lang, subcommand)
	return fn(args[1:], stdout, stderr)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
