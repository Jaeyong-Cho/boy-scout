package main

import (
	"encoding/json"
	"fmt"
	"io"

	"boy-scout/internal/abstractness"
	"boy-scout/internal/cppfunclen"
	"boy-scout/internal/crap"
	"boy-scout/internal/filelen"
	"boy-scout/internal/gofunclen"
	"boy-scout/internal/instability"
	"boy-scout/internal/tsfunclen"
)

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

// writeFilelenLines writes a filelen report's violations and excluded files to w,
// each line prefixed with prefix (e.g. "[filelen] " when combined with other checks).
func writeFilelenLines(w io.Writer, prefix string, report filelen.Report) {
	for _, v := range report.Violations {
		fmt.Fprintf(w, "%s%s: %d lines (limit %d)\n",
			prefix, v.File, v.Lines, v.Limit)
	}
	for _, f := range report.ExcludedFiles {
		fmt.Fprintf(w, "%sexcluded file: %s\n", prefix, f)
	}
}

func renderFilelenText(report filelen.Report, stdout, stderr io.Writer) int {
	writeFilelenLines(stdout, "", report)
	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func renderFilelenJSON(report filelen.Report, stdout, stderr io.Writer) int {
	data, err := json.Marshal(report)
	assertf(err == nil, "json.Marshal failed: %v", err)

	fmt.Fprintf(stdout, "%s\n", string(data))

	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

// writeInstabilityLines writes an instability report's violations to w,
// each line prefixed with prefix (e.g. "[instability] " when combined with other checks).
func writeInstabilityLines(w io.Writer, prefix string, report instability.Report) {
	for _, v := range report.Violations {
		fmt.Fprintf(w, "%s%s -> %s: Gap=%.3f (I_source=%.3f, I_target=%.3f)\n",
			prefix, v.Source, v.Target, v.Gap, v.I_A, v.I_B)
	}
	for _, f := range report.Skipped {
		fmt.Fprintf(w, "%sskipped file: %s (%s)\n", prefix, f.File, f.Error)
	}
	fmt.Fprintf(w, "%stotal edges: %d, violation rate: %.3f, weighted violation rate: %.3f\n",
		prefix, report.TotalEdges, report.ViolationRate, report.WeightedViolationRate)
}

func renderInstabilityText(report instability.Report, stdout, stderr io.Writer) int {
	writeInstabilityLines(stdout, "", report)
	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func renderInstabilityJSON(report instability.Report, stdout, stderr io.Writer) int {
	data, err := json.Marshal(report)
	assertf(err == nil, "json.Marshal failed: %v", err)

	fmt.Fprintf(stdout, "%s\n", string(data))

	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

// writeAbstractnessLines writes an abstractness report's violations to w,
// each line prefixed with prefix (e.g. "[abstractness] " when combined with other checks).
func writeAbstractnessLines(w io.Writer, prefix string, report abstractness.Report) {
	for _, v := range report.Violations {
		if v.Zone == "Pain" {
			fmt.Fprintf(w, "%s%s: Zone=%s, Distance=%.3f, SurfaceRatio=%.3f (A=%.3f, I=%.3f)\n",
				prefix, v.ImportPath, v.Zone, v.Distance, v.SurfaceRatio, v.Abstractness, v.Instability)
		} else {
			fmt.Fprintf(w, "%s%s: Zone=%s, Distance=%.3f (A=%.3f, I=%.3f)\n",
				prefix, v.ImportPath, v.Zone, v.Distance, v.Abstractness, v.Instability)
		}
	}
	for _, f := range report.Skipped {
		fmt.Fprintf(w, "%sskipped file: %s (%s)\n", prefix, f.File, f.Error)
	}
	fmt.Fprintf(w, "%stotal packages: %d\n",
		prefix, report.TotalPackages)
}

func renderAbstractnessText(report abstractness.Report, stdout, stderr io.Writer) int {
	writeAbstractnessLines(stdout, "", report)
	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func renderAbstractnessJSON(report abstractness.Report, stdout, stderr io.Writer) int {
	data, err := json.Marshal(report)
	assertf(err == nil, "json.Marshal failed: %v", err)

	fmt.Fprintf(stdout, "%s\n", string(data))

	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

type combinedReport struct {
	Gofunclen   gofunclen.Report   `json:"gofunclen"`
	Crap        crap.Report        `json:"crap"`
	Filelen     filelen.Report     `json:"filelen"`
	Instability instability.Report `json:"instability"`
	Abstractness abstractness.Report `json:"abstractness"`
}

func renderAllText(report combinedReport, stdout, stderr io.Writer) int {
	writeGofunclenLines(stdout, "[gofunclen] ", report.Gofunclen)
	writeCrapLines(stdout, "[crap] ", report.Crap)
	writeFilelenLines(stdout, "[filelen] ", report.Filelen)
	writeInstabilityLines(stdout, "[instability] ", report.Instability)
	writeAbstractnessLines(stdout, "[abstractness] ", report.Abstractness)

	totalViolations := len(report.Gofunclen.Violations) + len(report.Crap.Violations) + len(report.Filelen.Violations) + len(report.Instability.Violations) + len(report.Abstractness.Violations)
	totalSkipped := len(report.Gofunclen.Skipped) + len(report.Crap.Skipped) + len(report.Filelen.Skipped) + len(report.Instability.Skipped) + len(report.Abstractness.Skipped)

	return exitCodeFor(totalViolations, totalSkipped)
}

func renderAllJSON(report combinedReport, stdout, stderr io.Writer) int {
	data, err := json.Marshal(report)
	assertf(err == nil, "json.Marshal failed: %v", err)

	fmt.Fprintf(stdout, "%s\n", string(data))

	totalViolations := len(report.Gofunclen.Violations) + len(report.Crap.Violations) + len(report.Filelen.Violations) + len(report.Instability.Violations) + len(report.Abstractness.Violations)
	totalSkipped := len(report.Gofunclen.Skipped) + len(report.Crap.Skipped) + len(report.Filelen.Skipped) + len(report.Instability.Skipped) + len(report.Abstractness.Skipped)

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

// writeTsFunclenLines writes a ts funclen report's violations and excluded entries to w.
func writeTsFunclenLines(w io.Writer, prefix string, report tsfunclen.Report) {
	for _, v := range report.Violations {
		fmt.Fprintf(w, "%s%s:%d: function %s is %d lines (limit %d)\n",
			prefix, v.File, v.Line, v.Func, v.Length, v.Limit)
	}
	for _, exc := range report.ExcludedFuncs {
		fmt.Fprintf(w, "%s%s: function %s excluded (%s)\n",
			prefix, exc.File, exc.Func, exc.Reason)
	}
}

func renderTsFunclenText(report tsfunclen.Report, stdout, stderr io.Writer) int {
	writeTsFunclenLines(stdout, "", report)
	return exitCodeFor(len(report.Violations), len(report.Skipped))
}

func renderTsFunclenJSON(report tsfunclen.Report, stdout, stderr io.Writer) int {
	data, err := json.Marshal(report)
	assertf(err == nil, "json.Marshal failed: %v", err)

	fmt.Fprintf(stdout, "%s\n", string(data))

	return exitCodeFor(len(report.Violations), len(report.Skipped))
}
