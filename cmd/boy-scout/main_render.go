package main

import (
	"encoding/json"
	"fmt"
	"io"

	"boy-scout/internal/assertutil"
	"boy-scout/internal/collen"
	"boy-scout/internal/cppcomplexity"
	"boy-scout/internal/cppfunclen"
	"boy-scout/internal/duplication"
	"boy-scout/internal/filelen"
	"boy-scout/internal/gocomplexity"
	"boy-scout/internal/gofunclen"
	"boy-scout/internal/tscomplexity"
	"boy-scout/internal/tsfunclen"
)

// reportError writes an error message to stderr; the runner owns the exit code.
func reportError(err error, stderr io.Writer) {
	fmt.Fprintf(stderr, "error: %v\n", err)
}

// selectAndRender chooses between JSON and text renderer based on format flag.
func selectAndRender(format *string, jsonRender, textRender func(io.Writer, io.Writer) int, stdout, stderr io.Writer) int {
	if *format == "json" {
		return jsonRender(stdout, stderr)
	}
	return textRender(stdout, stderr)
}

// renderReportAsJSON writes a report and returns the exit code for its results.
func renderReportAsJSON[R any](report R, numViolations, numSkipped int, stdout, stderr io.Writer) int {
	data, err := json.Marshal(report)
	assertutil.Assertf(err == nil, "json.Marshal failed: %v", err)
	fmt.Fprintf(stdout, "%s\n", string(data))
	return exitCodeFor(numViolations, numSkipped)
}

// writeLines is a generic helper that writes violations and excluded entries to w,
// each line prefixed with prefix. It accepts two formatter functions to customize the output.
// Nil slices are handled gracefully (no output for nil).
func writeLines[V, E any](w io.Writer, prefix string, violations []V, excluded []E, formatViolation func(V) string, formatExcluded func(E) string) {
	if violations == nil {
		violations = []V{}
	}
	if excluded == nil {
		excluded = []E{}
	}
	for _, v := range violations {
		fmt.Fprintf(w, "%s%s\n", prefix, formatViolation(v))
	}
	for _, e := range excluded {
		fmt.Fprintf(w, "%s%s\n", prefix, formatExcluded(e))
	}
}

// writeFilelenLines writes a filelen report's violations and excluded files to w,
// each line prefixed with prefix (e.g. "[filelen] " when combined with other checks).
func writeFilelenLines(w io.Writer, prefix string, report filelen.Report) {
	writeLines(w, prefix, report.Violations, report.ExcludedFiles,
		func(v filelen.Violation) string {
			return fmt.Sprintf("%s: %d lines (limit %d)",
				v.File, v.Lines, v.Limit)
		},
		func(f string) string {
			return fmt.Sprintf("excluded file: %s", f)
		},
	)
}

func renderFilelenText(report filelen.Report, stdout, stderr io.Writer) int {
	writeFilelenLines(stdout, "", report)
	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func renderFilelenJSON(report filelen.Report, stdout, stderr io.Writer) int {
	return renderReportAsJSON(report, len(report.Violations), len(report.Skipped), stdout, stderr)
}

// writeCollenLines writes a collen report's violations and excluded files to w,
// each line prefixed with prefix (e.g. "[collen] " when combined with other checks).
func writeCollenLines(w io.Writer, prefix string, report collen.Report) {
	writeLines(w, prefix, report.Violations, report.ExcludedFiles,
		func(v collen.Violation) string {
			return fmt.Sprintf("%s:%d: %d chars (limit %d)",
				v.File, v.Line, v.Length, v.Limit)
		},
		func(f string) string {
			return fmt.Sprintf("excluded file: %s", f)
		},
	)
}

func renderCollenText(report collen.Report, stdout, stderr io.Writer) int {
	writeCollenLines(stdout, "", report)
	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func renderCollenJSON(report collen.Report, stdout, stderr io.Writer) int {
	return renderReportAsJSON(report, len(report.Violations), len(report.Skipped), stdout, stderr)
}

// writeDuplicationLines writes a duplication report's violations to w,
// each line prefixed with prefix (e.g. "[duplication] " when combined with other checks).
func writeDuplicationLines(w io.Writer, prefix string, report duplication.Report) {
	for _, v := range report.Violations {
		if v.Type == "Type-3" {
			fmt.Fprintf(w, "%s%s:%d: function %s is %s duplicate of %s:%d function %s (%.1f%% similar, %d duplicated lines)\n",
				prefix, v.FileA, v.LineA, v.FuncA, v.Type, v.FileB, v.LineB, v.FuncB, v.Similarity*100, v.DupLines)
		} else {
			fmt.Fprintf(w, "%s%s:%d: function %s is %s duplicate of %s:%d function %s (%d duplicated lines)\n",
				prefix, v.FileA, v.LineA, v.FuncA, v.Type, v.FileB, v.LineB, v.FuncB, v.DupLines)
		}
	}
	// Write cluster summaries
	for _, c := range report.Clusters {
		fmt.Fprintf(w, "%s%d functions clustered as one duplicate group (%d duplicated lines total, cross-package: %t)\n",
			prefix, len(c.Members), c.DupLines, c.CrossPackage)
	}
	for _, f := range report.ExcludedFiles {
		fmt.Fprintf(w, "%sexcluded file: %s\n", prefix, f)
	}
	for _, exc := range report.ExcludedFuncs {
		fmt.Fprintf(w, "%s%s:%d: function %s excluded (%s)\n",
			prefix, exc.File, exc.Line, exc.Func, exc.Reason)
	}
}

func renderDuplicationText(report duplication.Report, stdout, stderr io.Writer) int {
	writeDuplicationLines(stdout, "", report)
	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func renderDuplicationJSON(report duplication.Report, stdout, stderr io.Writer) int {
	return renderReportAsJSON(report, len(report.Violations), len(report.Skipped), stdout, stderr)
}

type combinedReport struct {
	Gofunclen   gofunclen.Report    `json:"gofunclen"`
	Complexity  gocomplexity.Report `json:"complexity"`
	Filelen     filelen.Report      `json:"filelen"`
	Collen      collen.Report       `json:"collen"`
	Duplication duplication.Report  `json:"duplication"`
}

// ponytail: one combinedReport struct per language (go/cpp/ts) — generalize to a shared keyed-report type if a 4th language shows up.
type cppCombinedReport struct {
	Funclen     cppfunclen.Report    `json:"funclen"`
	Complexity  cppcomplexity.Report `json:"complexity"`
	Filelen     filelen.Report       `json:"filelen"`
	Collen      collen.Report        `json:"collen"`
	Duplication duplication.Report   `json:"duplication"`
}

type tsCombinedReport struct {
	Funclen tsfunclen.Report `json:"funclen"`
	Filelen filelen.Report   `json:"filelen"`
	Collen  collen.Report    `json:"collen"`
}

func renderAllText(report combinedReport, stdout, stderr io.Writer) int {
	writeGofunclenLines(stdout, "[gofunclen] ", report.Gofunclen)
	writeComplexityLines(stdout, "[complexity] ", report.Complexity)
	writeFilelenLines(stdout, "[filelen] ", report.Filelen)
	writeCollenLines(stdout, "[collen] ", report.Collen)
	writeDuplicationLines(stdout, "[duplication] ", report.Duplication)

	totalViolations := len(report.Gofunclen.Violations) + len(report.Complexity.Violations) + len(report.Filelen.Violations) + len(report.Collen.Violations) + len(report.Duplication.Violations)
	totalSkipped := len(report.Gofunclen.Skipped) + len(report.Complexity.Skipped) + len(report.Filelen.Skipped) + len(report.Collen.Skipped) + len(report.Duplication.Skipped)

	return exitCodeFor(totalViolations, totalSkipped)
}

func renderAllJSON(report combinedReport, stdout, stderr io.Writer) int {
	totalViolations := len(report.Gofunclen.Violations) + len(report.Complexity.Violations) + len(report.Filelen.Violations) + len(report.Collen.Violations) + len(report.Duplication.Violations)
	totalSkipped := len(report.Gofunclen.Skipped) + len(report.Complexity.Skipped) + len(report.Filelen.Skipped) + len(report.Collen.Skipped) + len(report.Duplication.Skipped)

	return renderReportAsJSON(report, totalViolations, totalSkipped, stdout, stderr)
}

func renderCppAllText(report cppCombinedReport, stdout, stderr io.Writer) int {
	writeCppFunclenLines(stdout, "[funclen] ", report.Funclen)
	writeCppComplexityLines(stdout, "[complexity] ", report.Complexity)
	writeFilelenLines(stdout, "[filelen] ", report.Filelen)
	writeCollenLines(stdout, "[collen] ", report.Collen)
	writeDuplicationLines(stdout, "[duplication] ", report.Duplication)

	totalViolations := len(report.Funclen.Violations) + len(report.Complexity.Violations) + len(report.Filelen.Violations) + len(report.Collen.Violations) + len(report.Duplication.Violations)
	totalSkipped := len(report.Funclen.Skipped) + len(report.Complexity.Skipped) + len(report.Filelen.Skipped) + len(report.Collen.Skipped) + len(report.Duplication.Skipped)

	return exitCodeFor(totalViolations, totalSkipped)
}

func renderCppAllJSON(report cppCombinedReport, stdout, stderr io.Writer) int {
	totalViolations := len(report.Funclen.Violations) + len(report.Complexity.Violations) + len(report.Filelen.Violations) + len(report.Collen.Violations) + len(report.Duplication.Violations)
	totalSkipped := len(report.Funclen.Skipped) + len(report.Complexity.Skipped) + len(report.Filelen.Skipped) + len(report.Collen.Skipped) + len(report.Duplication.Skipped)

	return renderReportAsJSON(report, totalViolations, totalSkipped, stdout, stderr)
}

func renderTsAllText(report tsCombinedReport, stdout, stderr io.Writer) int {
	writeTsFunclenLines(stdout, "[funclen] ", report.Funclen)
	writeFilelenLines(stdout, "[filelen] ", report.Filelen)
	writeCollenLines(stdout, "[collen] ", report.Collen)

	totalViolations := len(report.Funclen.Violations) + len(report.Filelen.Violations) + len(report.Collen.Violations)
	totalSkipped := len(report.Funclen.Skipped) + len(report.Filelen.Skipped) + len(report.Collen.Skipped)

	return exitCodeFor(totalViolations, totalSkipped)
}

func renderTsAllJSON(report tsCombinedReport, stdout, stderr io.Writer) int {
	totalViolations := len(report.Funclen.Violations) + len(report.Filelen.Violations) + len(report.Collen.Violations)
	totalSkipped := len(report.Funclen.Skipped) + len(report.Filelen.Skipped) + len(report.Collen.Skipped)

	return renderReportAsJSON(report, totalViolations, totalSkipped, stdout, stderr)
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

	assertutil.Assertf(code == 0 || code == 1 || code == 2, "unexpected exit code %d", code)
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
	return renderReportAsJSON(report, len(report.Violations), len(report.Skipped), stdout, stderr)
}

// writeComplexityLines writes a complexity report's violations and excluded entries to w,
// each line prefixed with prefix (e.g. "[complexity] " when combined with other checks).
func writeComplexityLines(w io.Writer, prefix string, report gocomplexity.Report) {
	for _, v := range report.Violations {
		fmt.Fprintf(w, "%s%s:%d: function %s has complexity=%d, limit=%d\n",
			prefix, v.File, v.Line, v.Func, v.Complexity, v.Limit)
	}
	for _, f := range report.ExcludedFiles {
		fmt.Fprintf(w, "%sexcluded file: %s\n", prefix, f)
	}
	for _, exc := range report.ExcludedFuncs {
		fmt.Fprintf(w, "%s%s:%d: function %s excluded (%s)\n",
			prefix, exc.File, exc.Line, exc.Func, exc.Reason)
	}
}

func renderComplexityText(report gocomplexity.Report, stdout, stderr io.Writer) int {
	writeComplexityLines(stdout, "", report)
	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func renderComplexityJSON(report gocomplexity.Report, stdout, stderr io.Writer) int {
	return renderReportAsJSON(report, len(report.Violations), len(report.Skipped), stdout, stderr)
}

// writeCppFunclenLines writes a cpp funclen report's violations and excluded entries to w.
func writeCppFunclenLines(w io.Writer, prefix string, report cppfunclen.Report) {
	writeLines(w, prefix, report.Violations, report.ExcludedFuncs,
		func(v cppfunclen.Violation) string {
			return fmt.Sprintf("%s:%d: function %s is %d lines (limit %d)",
				v.File, v.Line, v.Func, v.Length, v.Limit)
		},
		func(exc cppfunclen.ExcludedFunc) string {
			return fmt.Sprintf("%s: function %s excluded (%s)",
				exc.File, exc.Func, exc.Reason)
		},
	)
}

func renderCppFunclenText(report cppfunclen.Report, stdout, stderr io.Writer) int {
	writeCppFunclenLines(stdout, "", report)
	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func renderCppFunclenJSON(report cppfunclen.Report, stdout, stderr io.Writer) int {
	return renderReportAsJSON(report, len(report.Violations), len(report.Skipped), stdout, stderr)
}

// writeCppComplexityLines writes a cpp complexity report's violations and excluded entries to w.
func writeCppComplexityLines(w io.Writer, prefix string, report cppcomplexity.Report) {
	writeLines(w, prefix, report.Violations, report.ExcludedFuncs,
		func(v cppcomplexity.Violation) string {
			return fmt.Sprintf("%s:%d: function %s has complexity=%d (limit %d)", v.File, v.Line, v.Func, v.Complexity, v.Limit)
		},
		func(exc cppcomplexity.ExcludedFunc) string {
			return fmt.Sprintf("%s:%d: function %s excluded (%s)", exc.File, exc.Line, exc.Func, exc.Reason)
		},
	)
}

func renderCppComplexityText(report cppcomplexity.Report, stdout, stderr io.Writer) int {
	writeCppComplexityLines(stdout, "", report)
	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func renderCppComplexityJSON(report cppcomplexity.Report, stdout, stderr io.Writer) int {
	return renderReportAsJSON(report, len(report.Violations), len(report.Skipped), stdout, stderr)
}

// writeTsFunclenLines writes a ts funclen report's violations and excluded entries to w.
func writeTsFunclenLines(w io.Writer, prefix string, report tsfunclen.Report) {
	writeLines(w, prefix, report.Violations, report.ExcludedFuncs,
		func(v tsfunclen.Violation) string {
			return fmt.Sprintf("%s:%d: function %s is %d lines (limit %d)",
				v.File, v.Line, v.Func, v.Length, v.Limit)
		},
		func(exc tsfunclen.ExcludedFunc) string {
			return fmt.Sprintf("%s: function %s excluded (%s)",
				exc.File, exc.Func, exc.Reason)
		},
	)
}

func renderTsFunclenText(report tsfunclen.Report, stdout, stderr io.Writer) int {
	writeTsFunclenLines(stdout, "", report)
	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func renderTsFunclenJSON(report tsfunclen.Report, stdout, stderr io.Writer) int {
	return renderReportAsJSON(report, len(report.Violations), len(report.Skipped), stdout, stderr)
}

// writeTsComplexityLines writes a ts complexity report's violations and excluded entries to w.
func writeTsComplexityLines(w io.Writer, prefix string, report tscomplexity.Report) {
	writeLines(w, prefix, report.Violations, report.ExcludedFuncs,
		func(v tscomplexity.Violation) string {
			return fmt.Sprintf("%s:%d: function %s has complexity=%d (limit %d)", v.File, v.Line, v.Func, v.Complexity, v.Limit)
		},
		func(exc tscomplexity.ExcludedFunc) string {
			return fmt.Sprintf("%s:%d: function %s excluded (%s)", exc.File, exc.Line, exc.Func, exc.Reason)
		},
	)
}

func renderTsComplexityText(report tscomplexity.Report, stdout, stderr io.Writer) int {
	writeTsComplexityLines(stdout, "", report)
	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func renderTsComplexityJSON(report tscomplexity.Report, stdout, stderr io.Writer) int {
	return renderReportAsJSON(report, len(report.Violations), len(report.Skipped), stdout, stderr)
}
