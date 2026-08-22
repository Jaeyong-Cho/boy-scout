package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"go-gardener/internal/crap"
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

func runCrap(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("crap", flag.ContinueOnError)
	threshold := fs.Float64("threshold", 6.0, "CRAP score threshold")
	format := fs.String("format", "text", "output format: text or json")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "parse error: %v\n", err)
		return 2
	}

	paths := fs.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	report, err := crap.Check(paths, *threshold)
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

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "parse error: %v\n", err)
		return 2
	}

	paths := fs.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// Run funclen check
	funclenReport, err := funclen.Check(paths, 50)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	// Run crap check
	crapReport, err := crap.Check(paths, 6.0)
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
	// Render funclen violations
	for _, v := range report.Funclen.Violations {
		fmt.Fprintf(stdout, "[funclen] %s:%d: function %s is %d lines (limit %d)\n",
			v.File, v.Line, v.Func, v.Length, v.Limit)
	}

	// Render crap violations
	for _, v := range report.Crap.Violations {
		fmt.Fprintf(stdout, "[crap] %s:%d: function %s has CRAP score %.2f (complexity=%d, coverage=%.1f%%, threshold=%.2f)\n",
			v.File, v.Line, v.Func, v.Score, v.Complexity, v.Coverage*100, v.Threshold)
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
