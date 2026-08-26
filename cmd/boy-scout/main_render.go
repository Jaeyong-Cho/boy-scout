package main

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"boy-scout/internal/abstractness"
	"boy-scout/internal/assertutil"
	"boy-scout/internal/cppfunclen"
	"boy-scout/internal/crap"
	"boy-scout/internal/duplication"
	"boy-scout/internal/filelen"
	"boy-scout/internal/gocomplexity"
	"boy-scout/internal/gofunclen"
	"boy-scout/internal/instability"
	"boy-scout/internal/linelen"
	"boy-scout/internal/tsfunclen"
)

// reportError writes an error message to stderr and returns exit code 2.
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

// renderReportAsJSON marshals any report to JSON and renders it.
// Uses reflection to extract violation and skipped counts from the report.
func renderReportAsJSON(report any, stdout, stderr io.Writer) int {
	data, err := json.Marshal(report)
	assertutil.Assertf(err == nil, "json.Marshal failed: %v", err)

	fmt.Fprintf(stdout, "%s\n", string(data))

	// Extract violation and skipped counts using reflection
	numViolations, numSkipped := 0, 0
	if rv := reflect.ValueOf(report); rv.Kind() == reflect.Struct {
		if violations := rv.FieldByName("Violations"); violations.IsValid() {
			numViolations = violations.Len()
		}
		if skipped := rv.FieldByName("Skipped"); skipped.IsValid() {
			numSkipped = skipped.Len()
		}
	}

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
	return renderReportAsJSON(report, stdout, stderr)
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
	return renderReportAsJSON(report, stdout, stderr)
}

// writeLinelenLines writes a linelen report's violations and excluded files to w,
// each line prefixed with prefix (e.g. "[linelen] " when combined with other checks).
func writeLinelenLines(w io.Writer, prefix string, report linelen.Report) {
	writeLines(w, prefix, report.Violations, report.ExcludedFiles,
		func(v linelen.Violation) string {
			return fmt.Sprintf("%s:%d: %d chars (limit %d)",
				v.File, v.Line, v.Length, v.Limit)
		},
		func(f string) string {
			return fmt.Sprintf("excluded file: %s", f)
		},
	)
}

func renderLinelenText(report linelen.Report, stdout, stderr io.Writer) int {
	writeLinelenLines(stdout, "", report)
	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func renderLinelenJSON(report linelen.Report, stdout, stderr io.Writer) int {
	return renderReportAsJSON(report, stdout, stderr)
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
	return renderReportAsJSON(report, stdout, stderr)
}

// writeInstabilityLines writes an instability report's violations to w,
// each line prefixed with prefix (e.g. "[instability] " when combined with other checks).
func writeInstabilityLines(w io.Writer, prefix string, report instability.Report) {
	writeLines(w, prefix, report.Violations, report.Skipped,
		func(v instability.Violation) string {
			return fmt.Sprintf("%s -> %s: Gap=%.3f (I_source=%.3f, I_target=%.3f)",
				v.Source, v.Target, v.Gap, v.I_A, v.I_B)
		},
		func(f instability.SkippedFile) string {
			return fmt.Sprintf("skipped file: %s (%s)", f.File, f.Error)
		},
	)
	fmt.Fprintf(w, "%stotal edges: %d, violation rate: %.3f, weighted violation rate: %.3f\n",
		prefix, report.TotalEdges, report.ViolationRate, report.WeightedViolationRate)
}

func renderInstabilityText(report instability.Report, stdout, stderr io.Writer) int {
	writeInstabilityLines(stdout, "", report)
	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func renderInstabilityJSON(report instability.Report, stdout, stderr io.Writer) int {
	return renderReportAsJSON(report, stdout, stderr)
}

// writeAbstractnessLines writes an abstractness report's violations to w,
// each line prefixed with prefix (e.g. "[abstractness] " when combined with other checks).
func writeAbstractnessLines(w io.Writer, prefix string, report abstractness.Report) {
	writeLines(w, prefix, report.Violations, report.Skipped,
		func(v abstractness.PackageDiagnosis) string {
			if v.Zone == "Pain" {
				return fmt.Sprintf("%s: Zone=%s, Distance=%.3f, SurfaceRatio=%.3f (A=%.3f, I=%.3f)",
					v.ImportPath, v.Zone, v.Distance, v.SurfaceRatio, v.Abstractness, v.Instability)
			}
			return fmt.Sprintf("%s: Zone=%s, Distance=%.3f (A=%.3f, I=%.3f)",
				v.ImportPath, v.Zone, v.Distance, v.Abstractness, v.Instability)
		},
		func(f abstractness.SkippedFile) string {
			return fmt.Sprintf("skipped file: %s (%s)", f.File, f.Error)
		},
	)
	fmt.Fprintf(w, "%stotal packages: %d\n",
		prefix, report.TotalPackages)
}

func renderAbstractnessText(report abstractness.Report, stdout, stderr io.Writer) int {
	writeAbstractnessLines(stdout, "", report)
	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func renderAbstractnessJSON(report abstractness.Report, stdout, stderr io.Writer) int {
	return renderReportAsJSON(report, stdout, stderr)
}

type combinedReport struct {
	Gofunclen    gofunclen.Report    `json:"gofunclen"`
	Complexity   gocomplexity.Report `json:"complexity"`
	Crap         crap.Report         `json:"crap"`
	Filelen      filelen.Report      `json:"filelen"`
	Linelen      linelen.Report      `json:"linelen"`
	Duplication  duplication.Report  `json:"duplication"`
	Instability  instability.Report  `json:"instability"`
	Abstractness abstractness.Report `json:"abstractness"`
}

func renderAllText(report combinedReport, stdout, stderr io.Writer) int {
	writeGofunclenLines(stdout, "[gofunclen] ", report.Gofunclen)
	writeComplexityLines(stdout, "[complexity] ", report.Complexity)
	writeCrapLines(stdout, "[crap] ", report.Crap)
	writeFilelenLines(stdout, "[filelen] ", report.Filelen)
	writeLinelenLines(stdout, "[linelen] ", report.Linelen)
	writeDuplicationLines(stdout, "[duplication] ", report.Duplication)
	writeInstabilityLines(stdout, "[instability] ", report.Instability)
	writeAbstractnessLines(stdout, "[abstractness] ", report.Abstractness)

	totalViolations := len(report.Gofunclen.Violations) + len(report.Complexity.Violations) + len(report.Crap.Violations) + len(report.Filelen.Violations) + len(report.Linelen.Violations) + len(report.Duplication.Violations) + len(report.Instability.Violations) + len(report.Abstractness.Violations)
	totalSkipped := len(report.Gofunclen.Skipped) + len(report.Complexity.Skipped) + len(report.Crap.Skipped) + len(report.Filelen.Skipped) + len(report.Linelen.Skipped) + len(report.Duplication.Skipped) + len(report.Instability.Skipped) + len(report.Abstractness.Skipped)

	return exitCodeFor(totalViolations, totalSkipped)
}

func renderAllJSON(report combinedReport, stdout, stderr io.Writer) int {
	data, err := json.Marshal(report)
	assertutil.Assertf(err == nil, "json.Marshal failed: %v", err)

	fmt.Fprintf(stdout, "%s\n", string(data))

	totalViolations := len(report.Gofunclen.Violations) + len(report.Complexity.Violations) + len(report.Crap.Violations) + len(report.Filelen.Violations) + len(report.Linelen.Violations) + len(report.Duplication.Violations) + len(report.Instability.Violations) + len(report.Abstractness.Violations)
	totalSkipped := len(report.Gofunclen.Skipped) + len(report.Complexity.Skipped) + len(report.Crap.Skipped) + len(report.Filelen.Skipped) + len(report.Linelen.Skipped) + len(report.Duplication.Skipped) + len(report.Instability.Skipped) + len(report.Abstractness.Skipped)

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
	return renderReportAsJSON(report, stdout, stderr)
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
	return renderReportAsJSON(report, stdout, stderr)
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
	return renderReportAsJSON(report, stdout, stderr)
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
	return renderReportAsJSON(report, stdout, stderr)
}
