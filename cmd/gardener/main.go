package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"go-gardener/internal/funclen"
)

func assertf(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: gardener <subcommand> [options] [paths...]")
		fmt.Fprintln(stderr, "subcommands: funclen, all")
		return 2
	}

	subcommand := args[0]

	switch subcommand {
	case "funclen":
		return runFunclen(args[1:], stdout, stderr)
	case "all":
		return runAll(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown subcommand: %s\n", subcommand)
		return 2
	}
}

func runFunclen(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("funclen", flag.ContinueOnError)
	maxLines := fs.Int("max-lines", 100, "maximum function length in lines")
	format := fs.String("format", "text", "output format: text or json")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "parse error: %v\n", err)
		return 2
	}

	paths := fs.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	report, err := funclen.Check(paths, *maxLines)
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

func runAll(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("all", flag.ContinueOnError)
	format := fs.String("format", "text", "output format: text or json")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "parse error: %v\n", err)
		return 2
	}

	paths := fs.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// Run all checks and combine results
	var allViolations []funclen.Violation
	var allSkipped []funclen.SkippedFile

	// Run funclen check
	report, err := funclen.Check(paths, 100)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}
	allViolations = append(allViolations, report.Violations...)
	allSkipped = append(allSkipped, report.Skipped...)

	// Combine into a single report
	combinedReport := funclen.Report{
		Violations: allViolations,
		Skipped:    allSkipped,
	}

	// Render output
	if *format == "json" {
		return renderJSON(combinedReport, stdout, stderr)
	} else {
		return renderText(combinedReport, stdout, stderr)
	}
}

func renderText(report funclen.Report, stdout, stderr io.Writer) int {
	for _, v := range report.Violations {
		fmt.Fprintf(stdout, "%s:%d: function %s is %d lines (limit %d)\n",
			v.File, v.Line, v.Func, v.Length, v.Limit)
	}

	code := 0
	if len(report.Skipped) > 0 {
		code = 2
	} else if len(report.Violations) > 0 {
		code = 1
	}

	assertf(code == 0 || code == 1 || code == 2, "unexpected exit code %d", code)
	return code
}

func renderJSON(report funclen.Report, stdout, stderr io.Writer) int {
	data, err := json.Marshal(report)
	assertf(err == nil, "json.Marshal failed: %v", err)

	fmt.Fprintf(stdout, "%s\n", string(data))

	code := 0
	if len(report.Skipped) > 0 {
		code = 2
	} else if len(report.Violations) > 0 {
		code = 1
	}

	assertf(code == 0 || code == 1 || code == 2, "unexpected exit code %d", code)
	return code
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
