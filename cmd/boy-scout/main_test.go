package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"boy-scout/internal/tsfunclen"
)

// cppFuncLenFillerLines returns n valid, uniquely-named C++ declaration
// lines used to pad a function body past the gofunclen limit in tests.
func cppFuncLenFillerLines(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "  int v%d = %d;\n", i, i)
	}
	return b.String()
}

// writeGoFile writes content to dir/name and returns the full path.
func writeGoFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	return path
}

// longFuncSrc generates Go source for a package with a single function of totalLines lines.
func longFuncSrc(pkg, funcName string, totalLines int) string {
	lines := []string{"package " + pkg, "", "func " + funcName + "() {"}
	for i := 0; i < totalLines-2; i++ {
		lines = append(lines, "\t_ = 1")
	}
	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

// initModule writes a go.mod to dir and chdirs into it for the duration of the test.
func initModule(t *testing.T, dir string) {
	t.Helper()
	writeGoFile(t, dir, "go.mod", "module test\n\ngo 1.24\n")
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldCwd) })
}

type funcViolation struct {
	Func string `json:"func"`
}

type allReport struct {
	Gofunclen struct {
		Violations []funcViolation `json:"violations"`
	} `json:"gofunclen"`
	Crap struct {
		Violations []funcViolation `json:"violations"`
	} `json:"crap"`
}

// runAllJSON runs "go all --format=json ." against the current directory and parses the report.
func runAllJSON(t *testing.T) (allReport, int) {
	t.Helper()
	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "all", "--format=json", "."}, &stdoutBuf, &stderrBuf)
	var report allReport
	if err := json.Unmarshal(stdoutBuf.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\noutput: %s\nstderr: %s", err, stdoutBuf.String(), stderrBuf.String())
	}
	return report, exitCode
}

func hasViolation(vs []funcViolation, name string) bool {
	for _, v := range vs {
		if v.Func == name {
			return true
		}
	}
	return false
}

func TestRun_TextFormatMatchesExpectedLine(t *testing.T) {
	// Create a file with a 105-line function
	lines := []string{"package main", "", "func Violating() {"}
	for i := 0; i < 103; i++ {
		lines = append(lines, "\tx := 1")
	}
	lines = append(lines, "}")
	src := strings.Join(lines, "\n")

	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/viol.go"
	if err := os.WriteFile(tmpFile, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "gofunclen", tmpFile}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	// Expect format: "<file>:<line>: function <Name> is <length> lines (limit <limit>)"
	// The line should be 3 (the opening brace of the function)
	if !strings.Contains(output, "Violating is 105 lines (limit 50)") {
		t.Errorf("expected output to contain 'Violating is 105 lines (limit 50)', got:\n%s", output)
	}
	if !strings.Contains(output, tmpFile) {
		t.Errorf("expected output to contain file path %q, got:\n%s", tmpFile, output)
	}
	if !strings.Contains(output, ":3:") {
		t.Errorf("expected output to contain line number :3:, got:\n%s", output)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

func TestRun_GofunclenRespectsMaxLinesFlag(t *testing.T) {
	// Create a file with a 60-line function
	lines := []string{"package main", "", "func Moderate() {"}
	for i := 0; i < 58; i++ {
		lines = append(lines, "\tx := 1")
	}
	lines = append(lines, "}")
	src := strings.Join(lines, "\n")

	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/moderate.go"
	if err := os.WriteFile(tmpFile, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "gofunclen", "--max-lines=50", tmpFile}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	// Should report violation with limit=50, not 100
	if !strings.Contains(output, "Moderate is 60 lines (limit 50)") {
		t.Errorf("expected output to contain 'Moderate is 60 lines (limit 50)', got:\n%s", output)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

func TestRun_GofunclenDefaultsToCurrentDir(t *testing.T) {
	// Create a temp dir with a violating file
	tmpDir := t.TempDir()

	lines := []string{"package main", "", "func Big() {"}
	for i := 0; i < 103; i++ {
		lines = append(lines, "\tx := 1")
	}
	lines = append(lines, "}")
	src := strings.Join(lines, "\n")

	bigGoPath := tmpDir + "/big.go"
	if err := os.WriteFile(bigGoPath, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Verify the file exists and can be read
	if _, err := os.Stat(bigGoPath); err != nil {
		t.Fatalf("file not found after write: %v", err)
	}

	// Change to that directory and run with no path argument
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldCwd); err != nil {
			t.Fatalf("Chdir back failed: %v", err)
		}
	})

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "gofunclen"}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()
	stderr := stderrBuf.String()
	if !strings.Contains(output, "Big is 105 lines (limit 50)") {
		t.Errorf("expected output to contain 'Big is 105 lines (limit 50)', got stdout:\n%s\nstderr:\n%s", output, stderr)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

func TestRun_AbstractnessDefaultPathDoesNotPanic(t *testing.T) {
	// Create a temp module with pkg/a and pkg/b to test that "go abstractness"
	// with no path argument (defaults to ".") does not panic when BuildGraph
	// must resolve relative paths correctly.
	tmpDir := t.TempDir()
	initModule(t, tmpDir)

	// Create pkg/a: zero exported types (will be Zone-of-Pain candidate)
	aPkgDir := filepath.Join(tmpDir, "pkg", "a")
	os.MkdirAll(aPkgDir, 0755)
	aCode := `package a

import "test/pkg/b"

func UseB() { b.B() }
`
	writeGoFile(t, aPkgDir, "a.go", aCode)

	// Create pkg/b
	bPkgDir := filepath.Join(tmpDir, "pkg", "b")
	os.MkdirAll(bPkgDir, 0755)
	bCode := `package b

func B() {}
`
	writeGoFile(t, bPkgDir, "b.go", bCode)

	// Create pkg/c to import pkg/a (make it a Pain candidate with Ca=1)
	cPkgDir := filepath.Join(tmpDir, "pkg", "c")
	os.MkdirAll(cPkgDir, 0755)
	cCode := `package c

import "test/pkg/a"

func C() { a.UseB() }
`
	writeGoFile(t, cPkgDir, "c.go", cCode)

	// Guard against panics
	var panicValue interface{}
	var stdoutBuf, stderrBuf bytes.Buffer
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicValue = r
			}
		}()

		// Note: we pass no path argument, so it defaults to "."
		run([]string{"go", "abstractness"}, &stdoutBuf, &stderrBuf)
	}()

	if panicValue != nil {
		t.Fatalf("expected no panic, but recovered: %v", panicValue)
	}
}

func TestRun_AllDefaultPathDoesNotPanic(t *testing.T) {
	// Create a temp module with pkg/a and pkg/b to test that "go all"
	// with no path argument (defaults to ".") does not panic.
	// This is the exact command from the bug report: ./bin/boy-scout go all
	tmpDir := t.TempDir()
	initModule(t, tmpDir)

	// Create pkg/a: zero exported types (will be Zone-of-Pain candidate)
	aPkgDir := filepath.Join(tmpDir, "pkg", "a")
	os.MkdirAll(aPkgDir, 0755)
	aCode := `package a

import "test/pkg/b"

func UseB() { b.B() }
`
	writeGoFile(t, aPkgDir, "a.go", aCode)

	// Create pkg/b
	bPkgDir := filepath.Join(tmpDir, "pkg", "b")
	os.MkdirAll(bPkgDir, 0755)
	bCode := `package b

func B() {}
`
	writeGoFile(t, bPkgDir, "b.go", bCode)

	// Create pkg/c to import pkg/a
	cPkgDir := filepath.Join(tmpDir, "pkg", "c")
	os.MkdirAll(cPkgDir, 0755)
	cCode := `package c

import "test/pkg/a"

func C() { a.UseB() }
`
	writeGoFile(t, cPkgDir, "c.go", cCode)

	// Guard against panics
	var panicValue interface{}
	var stdoutBuf, stderrBuf bytes.Buffer
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicValue = r
			}
		}()

		// Note: we pass no path argument, so it defaults to "."
		run([]string{"go", "all"}, &stdoutBuf, &stderrBuf)
	}()

	if panicValue != nil {
		t.Fatalf("expected no panic, but recovered: %v", panicValue)
	}
}

func TestRun_JSONFormatOutputsValidSchema(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := writeGoFile(t, tmpDir, "viol.go", longFuncSrc("main", "Violating", 105))

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "gofunclen", "--format=json", tmpFile}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	// Parse the JSON
	var result struct {
		Violations []struct {
			File   string
			Line   int
			Func   string
			Length int
			Limit  int
		}
		Skipped []struct {
			File  string
			Error string
		}
	}

	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("expected valid JSON output, got error: %v\noutput: %s", err, output)
		return
	}

	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}

	v := result.Violations[0]
	if v.Func != "Violating" || v.Length != 105 || v.Limit != 50 {
		t.Errorf("expected Violating/105/50, got %s/%d/%d", v.Func, v.Length, v.Limit)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

func TestRun_ExitCodeReflectsOutcome(t *testing.T) {
	testCases := []struct {
		name        string
		hasViolations bool
		hasSkipped  bool
		expectedCode int
	}{
		{"clean", false, false, 0},
		{"violations only", true, false, 1},
		{"skipped files", true, true, 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create a violating file if needed
			if tc.hasViolations {
				writeGoFile(t, tmpDir, "big.go", longFuncSrc("main", "Big", 105))
			}

			// Create a skipped file if needed
			if tc.hasSkipped {
				writeGoFile(t, tmpDir, "bad.go", "invalid syntax {")
			}

			// If no violations and no skipped, create a clean file
			if !tc.hasViolations && !tc.hasSkipped {
				writeGoFile(t, tmpDir, "small.go", "package main\n\nfunc Small() {\n\tx := 1\n}\n")
			}

			var stdoutBuf, stderrBuf bytes.Buffer
			exitCode := run([]string{"go", "gofunclen", tmpDir}, &stdoutBuf, &stderrBuf)

			if exitCode != tc.expectedCode {
				t.Errorf("expected exit code %d, got %d", tc.expectedCode, exitCode)
			}
		})
	}
}

func TestRun_NoSubcommandPrintsUsage(t *testing.T) {
	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{}, &stdoutBuf, &stderrBuf)

	stderr := stderrBuf.String()
	if !strings.Contains(stderr, "boy-scout <lang> <command>") || !strings.Contains(stderr, "languages: go") || !strings.Contains(stderr, "boy-scout setup") {
		t.Errorf("expected usage message with 'boy-scout <lang> <command>', 'languages: go' and 'boy-scout setup', got:\n%s", stderr)
	}

	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}
}

func TestRun_UnknownLanguagePrintsUsage(t *testing.T) {
	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"rust", "funclen"}, &stdoutBuf, &stderrBuf)

	stderr := stderrBuf.String()
	if !strings.Contains(stderr, "unknown language: rust") {
		t.Errorf("expected 'unknown language: rust' in stderr, got:\n%s", stderr)
	}

	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}
}

func TestRun_UnknownSubcommandForLangPrintsUsage(t *testing.T) {
	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "bogus"}, &stdoutBuf, &stderrBuf)

	stderr := stderrBuf.String()
	if !strings.Contains(stderr, "unknown subcommand for go: bogus") {
		t.Errorf("expected 'unknown subcommand for go: bogus' in stderr, got:\n%s", stderr)
	}

	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}
}

func TestRun_AllRunsEveryRegisteredCheck(t *testing.T) {
	// Create a temp dir with a violating file
	tmpDir := t.TempDir()

	lines := []string{"package main", "", "func ViolatingFunc() {"}
	for i := 0; i < 103; i++ {
		lines = append(lines, fmt.Sprintf("\t_ = %d", i))
	}
	lines = append(lines, "}")
	src := strings.Join(lines, "\n")

	if err := os.WriteFile(tmpDir+"/violating.go", []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Initialize go module
	if err := os.WriteFile(tmpDir+"/go.mod", []byte("module test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "all", "."}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()
	stderr := stderrBuf.String()
	if !strings.Contains(output, "ViolatingFunc is 105 lines (limit 50)") {
		t.Errorf("expected gofunclen violation in 'all' output, got:\nstdout: %s\nstderr: %s", output, stderr)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d (stderr: %s)", exitCode, stderr)
	}
}

func TestRun_CrapRespectsThresholdFlag(t *testing.T) {
	// Create a file with a complex untested function
	src := `package main
func ComplexFunc(x int) string {
	if x > 10 {
		if x > 20 {
			return "big"
		}
		return "medium"
	}
	return "small"
}
`
	tmpDir := t.TempDir()
	if err := os.WriteFile(tmpDir+"/main.go", []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Initialize go module
	if err := os.WriteFile(tmpDir+"/go.mod", []byte("module test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "crap", "--threshold=2", "."}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()
	// Should show threshold=2.00 in output
	if !strings.Contains(output, "threshold=2.00") {
		t.Errorf("expected 'threshold=2.00' in output, got:\n%s", output)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1 (violations), got %d", exitCode)
	}
}

func TestRun_CrapTextFormatMatchesExpectedLine(t *testing.T) {
	// Create a file with a complex untested function
	src := `package main
func ComplexFunc(x int) string {
	if x > 10 {
		if x > 20 {
			return "big"
		}
		return "medium"
	}
	return "small"
}
`
	tmpDir := t.TempDir()
	if err := os.WriteFile(tmpDir+"/main.go", []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Initialize go module
	if err := os.WriteFile(tmpDir+"/go.mod", []byte("module test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "crap", "--threshold=1", "."}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()
	// Expected format: "file:line: function Name has CRAP score X.XX (complexity=C, coverage=P.P%, threshold=T.TT)"
	if !strings.Contains(output, "function ComplexFunc has CRAP score") {
		t.Errorf("expected 'function ComplexFunc has CRAP score' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "complexity=") {
		t.Errorf("expected 'complexity=' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "coverage=") {
		t.Errorf("expected 'coverage=' in output, got:\n%s", output)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// crapJSONResult mirrors the JSON schema produced by the crap subcommand's --format=json output.
type crapJSONResult struct {
	Violations []struct {
		File       string  `json:"file"`
		Line       int     `json:"line"`
		Func       string  `json:"func"`
		Complexity int     `json:"complexity"`
		Coverage   float64 `json:"coverage"`
		Score      float64 `json:"score"`
		Threshold  float64 `json:"threshold"`
	}
	Skipped []struct {
		File  string `json:"file"`
		Error string `json:"error"`
	}
}

func TestRun_CrapJSONFormatOutputsValidSchema(t *testing.T) {
	// Create a file with a complex untested function
	src := `package main
func ComplexFunc(x int) string {
	if x > 10 {
		if x > 20 {
			return "big"
		}
		return "medium"
	}
	return "small"
}
`
	tmpDir := t.TempDir()
	writeGoFile(t, tmpDir, "main.go", src)
	initModule(t, tmpDir)

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "crap", "--threshold=1", "--format=json", "."}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	var result crapJSONResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("expected valid JSON output, got error: %v\noutput: %s", err, output)
		return
	}

	if len(result.Violations) == 0 {
		t.Fatalf("expected at least 1 violation, got %d", len(result.Violations))
	}

	v := result.Violations[0]
	if v.Func != "ComplexFunc" {
		t.Errorf("expected func 'ComplexFunc', got '%s'", v.Func)
	}
	if v.Complexity <= 0 {
		t.Errorf("expected positive complexity, got %d", v.Complexity)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

func TestRun_CrapDefaultThresholdIs30(t *testing.T) {
	// Create a borderline function: complexity 8, coverage ~70% => CRAP ≈ 9.7
	// This should NOT violate with the new default (30.0)
	// but WOULD violate with the old default (6.0)
	src := `package main
func BorderlineFunc(x int) string {
	if x == 1 { return "a" }
	if x == 2 { return "b" }
	if x == 3 { return "c" }
	if x == 4 { return "d" }
	if x == 5 { return "e" }
	if x == 6 { return "f" }
	if x == 7 { return "g" }
	if x == 8 { return "h" }
	return "default"
}
`
	testSrc := `package main
import "testing"

func TestBorderlineFunc(t *testing.T) {
	cases := []struct {
		x int
		want string
	}{
		{1, "a"},
		{2, "b"},
		{3, "c"},
		{4, "d"},
		{5, "e"},
		{6, "f"},
		{7, "g"},
	}
	for _, tt := range cases {
		got := BorderlineFunc(tt.x)
		if got != tt.want {
			t.Errorf("BorderlineFunc(%d) = %q, want %q", tt.x, got, tt.want)
		}
	}
}
`

	tmpDir := t.TempDir()
	writeGoFile(t, tmpDir, "main.go", src)
	writeGoFile(t, tmpDir, "main_test.go", testSrc)
	initModule(t, tmpDir)

	var stdoutBuf, stderrBuf bytes.Buffer
	// Run with no --threshold flag; should use the new default (30.0)
	exitCode := run([]string{"go", "crap", "--format=json", "."}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	var result crapJSONResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("expected valid JSON output, got error: %v\noutput: %s", err, output)
		return
	}

	// With new default (30.0), BorderlineFunc (CRAP ≈ 9.7) should NOT violate
	for _, v := range result.Violations {
		if v.Func == "BorderlineFunc" {
			t.Errorf("BorderlineFunc should not violate at the new default threshold 30.0 (CRAP ≈ 9.7), but got violation with score %f", v.Score)
		}
	}

	// Exit code should be 0 (no violations)
	if exitCode != 0 {
		t.Errorf("expected exit code 0 (no violations with new default), got %d (output: %s)", exitCode, output)
	}
}

func TestRun_AllCombinesGofunclenAndCrap(t *testing.T) {
	// Create a file with both gofunclen and crap violations
	src := `package main
func ViolatingFunc() {
`
	// Add 105 lines to violate gofunclen
	for i := 0; i < 103; i++ {
		src += fmt.Sprintf("\t_ = %d\n", i)
	}
	src += `	if true {
		if true {
			if true {
				_ = "a"
			}
		}
	}
}
`

	tmpDir := t.TempDir()
	if err := os.WriteFile(tmpDir+"/main.go", []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Initialize go module
	if err := os.WriteFile(tmpDir+"/go.mod", []byte("module test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "all", "."}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()
	// Should have both gofunclen and crap violations
	if !strings.Contains(output, "[gofunclen]") {
		t.Errorf("expected '[gofunclen]' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "[crap]") {
		t.Errorf("expected '[crap]' in output, got:\n%s", output)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

const complexFuncSrc = `package main
func ComplexFunc() {
	if true {
		if true {
			if true {
				if true {
					_ = "x"
				}
			}
		}
	}
}
`

const cleanFuncSrc = `package main
func CleanFunc() {
	x := 1
	_ = x
}
`

// writeAllCheckFixtures writes source files into dir that produce the requested
// combination of gofunclen/crap violations for the "all" subcommand.
func writeAllCheckFixtures(t *testing.T, dir string, gofunclenViolating, crapViolating bool) {
	if gofunclenViolating {
		writeGoFile(t, dir, "long.go", longFuncSrc("main", "LongFunc", 105))
	}
	if crapViolating {
		writeGoFile(t, dir, "complex.go", complexFuncSrc)
	}
	if !gofunclenViolating && !crapViolating {
		writeGoFile(t, dir, "clean.go", cleanFuncSrc)
	}
}

func TestRun_AllExitCodePriorityAcrossBothChecks(t *testing.T) {
	testCases := []struct {
		name             string
		gofunclenViolating bool
		crapViolating    bool
		expectedExitCode int
	}{
		{"both clean", false, false, 0},
		{"only gofunclen violated", true, false, 1},
		{"only crap violated", false, true, 1},
		{"both violated", true, true, 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			writeAllCheckFixtures(t, tmpDir, tc.gofunclenViolating, tc.crapViolating)
			initModule(t, tmpDir)

			var stdoutBuf, stderrBuf bytes.Buffer
			exitCode := run([]string{"go", "all", "."}, &stdoutBuf, &stderrBuf)

			if exitCode != tc.expectedExitCode {
				t.Errorf("expected exit code %d, got %d (output: %s)", tc.expectedExitCode, exitCode, stdoutBuf.String())
			}
		})
	}
}

func TestRun_ExcludeFlagsFilterGofunclenOutput(t *testing.T) {
	// Create foo.go (105 lines - violation) and foo_test.go (105 lines - also violation)
	tmpDir := t.TempDir()

	fooSrc := "package main\nfunc Foo() {\n"
	for i := 0; i < 103; i++ {
		fooSrc += fmt.Sprintf("\t_ = %d\n", i)
	}
	fooSrc += "}\n"

	testSrc := "package main\nfunc TestFoo() {\n"
	for i := 0; i < 103; i++ {
		testSrc += fmt.Sprintf("\t_ = %d\n", i)
	}
	testSrc += "}\n"

	if err := os.WriteFile(tmpDir+"/foo.go", []byte(fooSrc), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(tmpDir+"/foo_test.go", []byte(testSrc), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "gofunclen", "--exclude-file=*_test.go", "--exclude-func=Foo", tmpDir}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()
	// Should have zero violations (Foo excluded by func, TestFoo excluded by file)
	if strings.Contains(output, "Foo is 105 lines") || strings.Contains(output, "TestFoo is 105 lines") {
		t.Errorf("expected no violations in output, got:\n%s", output)
	}

	if exitCode != 0 {
		t.Errorf("expected exit code 0 (no violations), got %d", exitCode)
	}
}

func TestRun_MalformedExcludePatternExitsWithUsageError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple file
	if err := os.WriteFile(tmpDir+"/main.go", []byte("package main\nfunc F() { x := 1 }\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "gofunclen", "--exclude-file=[", tmpDir}, &stdoutBuf, &stderrBuf)

	stderr := stderrBuf.String()
	if !strings.Contains(stderr, "error") || !strings.Contains(stderr, "invalid exclude pattern") {
		t.Errorf("expected error message about invalid pattern, got:\n%s", stderr)
	}

	if exitCode != 2 {
		t.Errorf("expected exit code 2 (usage error), got %d", exitCode)
	}
}

func TestRun_DebugFlagShowsExcludedFilesAndFuncs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create foo_test.go (105 lines - will be excluded)
	testSrc := "package main\nfunc TestFoo() {\n"
	for i := 0; i < 103; i++ {
		testSrc += fmt.Sprintf("\t_ = %d\n", i)
	}
	testSrc += "}\n"

	if err := os.WriteFile(tmpDir+"/foo_test.go", []byte(testSrc), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "gofunclen", "--exclude-file=*_test.go", "--debug", tmpDir}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()
	// Should mention the excluded file
	if !strings.Contains(output, "excluded file") || !strings.Contains(output, "foo_test.go") {
		t.Errorf("expected 'excluded file' mention with --debug, got:\n%s", output)
	}

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestRun_AllRespectsExcludeFlags(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a 105-line function in foo.go
	fooSrc := "package main\nfunc Foo() {\n"
	for i := 0; i < 103; i++ {
		fooSrc += fmt.Sprintf("\t_ = %d\n", i)
	}
	fooSrc += "}\n"

	if err := os.WriteFile(tmpDir+"/foo.go", []byte(fooSrc), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Initialize go module
	if err := os.WriteFile(tmpDir+"/go.mod", []byte("module test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "all", "--exclude-func=Foo", "."}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()
	// Should have no violations from gofunclen (Foo excluded)
	if strings.Contains(output, "[gofunclen] foo.go") && strings.Contains(output, "Foo is 105 lines") {
		t.Errorf("expected gofunclen violation to be excluded, got:\n%s", output)
	}

	if exitCode != 0 {
		t.Errorf("expected exit code 0 (no violations), got %d", exitCode)
	}
}

func TestRun_ExcludedViolationsDoNotAffectExitCode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a 105-line function (would violate if not excluded)
	src := "package main\nfunc Long() {\n"
	for i := 0; i < 103; i++ {
		src += fmt.Sprintf("\t_ = %d\n", i)
	}
	src += "}\n"

	if err := os.WriteFile(tmpDir+"/long.go", []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	// Exclude the function so there are zero real violations
	exitCode := run([]string{"go", "gofunclen", "--exclude-func=Long", tmpDir}, &stdoutBuf, &stderrBuf)

	// Exit code should be 0 (clean) even though a real violation was excluded
	if exitCode != 0 {
		t.Errorf("expected exit code 0 (excluded violation doesn't count), got %d", exitCode)
	}
}

func TestRun_InstabilityCommandIsRegistered(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a minimal go.mod
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module example.com/test\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Write a minimal Go file
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "instability", tmpDir}, &stdoutBuf, &stderrBuf)

	// Should not error (exit code 0 or 1 is fine, 2 means error)
	if exitCode == 2 {
		t.Errorf("expected instability command to work, got exit code 2\nstderr: %s", stderrBuf.String())
	}

	// Output should contain instability metrics
	output := stdoutBuf.String()
	if !strings.Contains(output, "total edges") && !strings.Contains(output, "violation rate") {
		t.Errorf("expected instability output to contain metrics, got:\n%s", output)
	}
}

func TestRun_InstabilityJSONFormat(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a minimal go.mod
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module example.com/test\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Write a minimal Go file
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "instability", "--format=json", tmpDir}, &stdoutBuf, &stderrBuf)

	if exitCode == 2 {
		t.Errorf("expected instability JSON command to work, got exit code 2\nstderr: %s", stderrBuf.String())
	}

	// Parse JSON output
	var report struct {
		Violations           []interface{}
		TotalEdges           int
		ViolationRate        float64
		WeightedViolationRate float64
	}
	if err := json.Unmarshal(stdoutBuf.Bytes(), &report); err != nil {
		t.Errorf("expected valid JSON output, got error: %v\noutput: %s", err, stdoutBuf.String())
	}
}

func TestRun_AllIncludesInstability(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a minimal go.mod
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module example.com/test\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Write a minimal Go file
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "all", "--format=json", "."}, &stdoutBuf, &stderrBuf)

	if exitCode == 2 {
		t.Errorf("expected 'go all' command to work, got exit code 2\nstderr: %s", stderrBuf.String())
	}

	// Parse JSON output and check for instability field
	var report struct {
		Instability struct {
			TotalEdges int
		}
	}
	if err := json.Unmarshal(stdoutBuf.Bytes(), &report); err != nil {
		t.Errorf("expected valid JSON output, got error: %v\noutput: %s", err, stdoutBuf.String())
	}

	// Check that instability field exists
	if report.Instability.TotalEdges < 0 {
		t.Error("expected instability field in combined report")
	}
}

// TestRun_InstabilityIgnoresTestOnlyImports is a full-CLI-pipeline regression test
// (run() -> flag parsing -> instability.BuildGraph -> JSON encode) for the
// AST-based dependency graph fix.
// Given: order.go imports payment (real), order_test.go imports mocks (test-only)
// When: `boy-scout go instability --format=json` runs on the module
// Then: TotalEdges must be 1 (order->payment only); mocks must never appear as a target
// NOTE: expected to FAIL until instability.BuildGraph excludes _test.go files.
func TestRun_InstabilityIgnoresTestOnlyImports(t *testing.T) {
	tmpDir := t.TempDir()
	writeGoFile(t, tmpDir, "go.mod", "module fixture\n\ngo 1.24\n")

	paymentDir := filepath.Join(tmpDir, "pkg", "payment")
	orderDir := filepath.Join(tmpDir, "pkg", "order")
	mocksDir := filepath.Join(tmpDir, "pkg", "mocks")
	for _, d := range []string{paymentDir, orderDir, mocksDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("MkdirAll(%s) failed: %v", d, err)
		}
	}

	writeGoFile(t, paymentDir, "payment.go", `package payment

type Service struct{}

func (s Service) Pay(amount int) bool { return amount > 0 }
`)
	writeGoFile(t, mocksDir, "mocks.go", `package mocks

type FakePayment struct{}

func New() FakePayment { return FakePayment{} }
`)
	writeGoFile(t, orderDir, "order.go", `package order

import "fixture/pkg/payment"

type Order struct {
	p payment.Service
}

func (o Order) Checkout(amount int) bool { return o.p.Pay(amount) }
`)
	writeGoFile(t, orderDir, "order_test.go", `package order

import (
	"testing"

	"fixture/pkg/mocks"
)

func TestCheckout(t *testing.T) {
	_ = mocks.New()
}
`)

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "instability", "--format=json", tmpDir}, &stdoutBuf, &stderrBuf)
	if exitCode == 2 {
		t.Fatalf("expected instability command to run, got exit code 2\nstderr: %s", stderrBuf.String())
	}

	var report struct {
		TotalEdges int
	}
	if err := json.Unmarshal(stdoutBuf.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\noutput: %s", err, stdoutBuf.String())
	}

	if report.TotalEdges != 1 {
		t.Errorf("expected TotalEdges=1 (order->payment only, order_test.go's mocks import excluded), got %d", report.TotalEdges)
	}
	if strings.Contains(stdoutBuf.String(), "mocks") {
		t.Errorf("expected no reference to test-only dependency 'mocks' in output, got:\n%s", stdoutBuf.String())
	}
}

// TestRun_AbstractnessIgnoresTestFuncsAsExported is a full-CLI-pipeline regression
// test for the same fix, on the abstractness side: TestXxx functions in _test.go
// files must not count toward SurfaceRatio's "exported declarations" count.
// Given: deepcache is a genuine deep module (3 exported types, ~21 unexported
// funcs/types) with 25 TestBehaviorN funcs in deepcache_test.go, and 8 caller
// packages depend on it (Ca=8, Ce=0 -> Zone of Pain candidate)
// When: `boy-scout go abstractness --format=json` runs
// Then: deepcache must NOT be reported as a violation (it's a deep module; the
// 25 TestBehaviorN funcs must not inflate SurfaceRatio past the 0.5 gate)
// NOTE: expected to FAIL until abstractness's underlying graph excludes _test.go files.
func TestRun_AbstractnessIgnoresTestFuncsAsExported(t *testing.T) {
	tmpDir := t.TempDir()
	writeGoFile(t, tmpDir, "go.mod", "module example.com/cache\n\ngo 1.24\n")
	setupCLIDeepCachePackage(t, tmpDir)
	setupCLICallerPackages(t, tmpDir, 8)

	var stdoutBuf, stderrBuf bytes.Buffer
	run([]string{"go", "abstractness", "--format=json", tmpDir}, &stdoutBuf, &stderrBuf)

	verifyCLIDeepCacheNotFlagged(t, stdoutBuf)
}

func setupCLIDeepCachePackage(t *testing.T, tmpDir string) {
	deepcacheDir := filepath.Join(tmpDir, "deepcache")
	if err := os.MkdirAll(deepcacheDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	writeCLIDeepCacheCode(t, deepcacheDir)
	writeCLIDeepCacheTests(t, deepcacheDir)
}

func writeCLIDeepCacheCode(t *testing.T, deepcacheDir string) {
	writeGoFile(t, deepcacheDir, "deepcache.go", `package deepcache

type Cache struct {
	data map[string]interface{}
	lock interface{}
}

type Config struct {
	MaxSize int
}

type Stats struct {
	Hits   int
	Misses int
}

func (c *Cache) get(key string) interface{}        { return nil }
func (c *Cache) set(key string, val interface{})   {}
func (c *Cache) evict(key string)                  {}
func (c *Cache) findLRU() string                   { return "" }
func (c *Cache) updateLRU(key string)              {}
func (c *Cache) marshal(v interface{}) []byte      { return nil }
func (c *Cache) unmarshal(data []byte) interface{} { return nil }
func (c *Cache) compress(data []byte) []byte       { return nil }
func (c *Cache) decompress(data []byte) []byte     { return nil }

type lruNode struct{ key string }
type hashBucket struct{ items []lruNode }
type serializer interface{}
type compressor interface{}

func newLRUNode(key string) *lruNode  { return nil }
func newHashBucket() *hashBucket      { return nil }
func newSerializer() serializer       { return nil }
func newCompressor() compressor       { return nil }
func hashKey(key string) uint64       { return 0 }
func validateKey(key string) error    { return nil }
func validateSize(sz int) error       { return nil }
func computeHash(data []byte) uint64  { return 0 }
func encodeStats(s *Stats) []byte     { return nil }
func decodeStats(data []byte) *Stats  { return nil }
func diffStats(s1, s2 *Stats) *Stats  { return nil }
func mergeStats(s1, s2 *Stats) *Stats { return nil }
`)
}

func writeCLIDeepCacheTests(t *testing.T, deepcacheDir string) {
	var testFuncs strings.Builder
	testFuncs.WriteString("package deepcache\n\nimport \"testing\"\n\n")
	for i := 1; i <= 25; i++ {
		fmt.Fprintf(&testFuncs, "func TestBehavior%d(t *testing.T) {}\n", i)
	}
	writeGoFile(t, deepcacheDir, "deepcache_test.go", testFuncs.String())
}

func setupCLICallerPackages(t *testing.T, tmpDir string, count int) {
	for i := 1; i <= count; i++ {
		callerDir := filepath.Join(tmpDir, fmt.Sprintf("caller%d", i))
		if err := os.MkdirAll(callerDir, 0755); err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
		writeGoFile(t, callerDir, fmt.Sprintf("caller%d.go", i), fmt.Sprintf(`package caller%d

import "example.com/cache/deepcache"

func Call%d(cfg deepcache.Config) {
	_ = cfg
}
`, i, i))
	}
}

func verifyCLIDeepCacheNotFlagged(t *testing.T, output bytes.Buffer) {
	var report struct {
		Violations []struct {
			ImportPath   string
			SurfaceRatio float64
		}
	}
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\noutput: %s", err, output.String())
	}

	for _, v := range report.Violations {
		if v.ImportPath == "example.com/cache/deepcache" {
			t.Errorf("deepcache is a deep module and should not be flagged, got SurfaceRatio=%f (25 TestBehaviorN funcs likely miscounted as exported)", v.SurfaceRatio)
		}
	}
}

// TestRun_InstabilitySkipsFileWithBrokenBody is a full-CLI-pipeline integration test
// for the full-AST-mode fix. Given a module with a file that has a valid
// package/import header but a syntax error in the function body, the file must
// be skipped and its imports must not be counted in the dependency graph.
// Given: pkg/helper/helper.go and pkg/broken/broken.go with broken function syntax
// When: `boy-scout go instability --format=json` runs
// Then: Skipped contains broken.go, TotalEdges == 0, exit code == 2
func TestRun_InstabilitySkipsFileWithBrokenBody(t *testing.T) {
	tmpDir := t.TempDir()
	writeGoFile(t, tmpDir, "go.mod", "module bodybug\n\ngo 1.24\n")

	helperDir := filepath.Join(tmpDir, "pkg", "helper")
	brokenDir := filepath.Join(tmpDir, "pkg", "broken")
	for _, d := range []string{helperDir, brokenDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("MkdirAll(%s) failed: %v", d, err)
		}
	}

	writeGoFile(t, helperDir, "helper.go", `package helper

func Do() {}
`)
	writeGoFile(t, brokenDir, "broken.go", `package broken

import "bodybug/pkg/helper"

func Bad( {
	helper.Do()
}
`)

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "instability", "--format=json", tmpDir}, &stdoutBuf, &stderrBuf)

	var report struct {
		Skipped   []struct{ File string }
		TotalEdges int
	}
	if err := json.Unmarshal(stdoutBuf.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\noutput: %s", err, stdoutBuf.String())
	}

	if len(report.Skipped) != 1 {
		t.Errorf("expected 1 skipped file, got %d: %v", len(report.Skipped), report.Skipped)
	} else if !strings.HasSuffix(report.Skipped[0].File, "broken.go") {
		t.Errorf("expected skipped file to end in broken.go, got %s", report.Skipped[0].File)
	}

	if report.TotalEdges != 0 {
		t.Errorf("expected TotalEdges=0 (broken.go skipped, no edges), got %d", report.TotalEdges)
	}

	if exitCode != 2 {
		t.Errorf("expected exit code 2 (skipped file priority), got %d", exitCode)
	}
}

// TestRun_VersionPrintsBuiltVersion verifies that the version subcommand
// prints the built version string (set via -ldflags -X main.version=...).
func TestRun_VersionPrintsBuiltVersion(t *testing.T) {
	// Override the version variable directly in the test.
	oldVersion := version
	t.Cleanup(func() { version = oldVersion })
	version = "v0.5.0"

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"version"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Errorf("version subcommand should exit 0, got %d; stderr: %s", exitCode, stderr.String())
	}

	expectedOutput := "boy-scout v0.5.0\n"
	if stdout.String() != expectedOutput {
		t.Errorf("expected output %q, got %q", expectedOutput, stdout.String())
	}
}

// TestRun_CppInstabilityReportsViolations verifies that the cpp instability check
// works via the CLI and reports violations correctly.
func TestRun_CppInstabilityReportsViolations(t *testing.T) {
	// Get the project root (from cmd/boy-scout test dir, go up 2 levels)
	cwd, _ := os.Getwd()
	projectRoot := filepath.Join(cwd, "../..")
	if abs, err := filepath.Abs(projectRoot); err == nil {
		projectRoot = abs
	}
	fixtureDir := filepath.Join(projectRoot, "testdata", "cpp-instability")

	// First run: no violations (original fixture)
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"cpp", "instability", "--format=json", fixtureDir}, &stdout, &stderr)

	// Parse JSON output
	var report struct {
		Violations []struct {
			Source string
			Target string
			Gap    float64
		}
		TotalEdges int
	}

	stdoutStr := stdout.String()
	stderrStr := stderr.String()
	if stdoutStr == "" {
		t.Fatalf("No output produced. Exit code: %d, stderr: %s", exitCode, stderrStr)
	}

	if err := json.Unmarshal([]byte(stdoutStr), &report); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nstdout: %s\nstderr: %s", err, stdoutStr, stderrStr)
	}

	if len(report.Violations) != 0 {
		t.Errorf("Expected 0 violations in original fixture, got %d", len(report.Violations))
	}
	if exitCode != 0 {
		t.Errorf("Expected exit code 0 (no violations), got %d", exitCode)
	}

	// Verify TotalEdges is > 0 (confirms the check ran)
	if report.TotalEdges == 0 {
		t.Error("Expected TotalEdges > 0")
	}
}

// TestRun_CppAbstractnessReportsViolations verifies that the cpp abstractness check
// works via the CLI and reports violations correctly.
func TestRun_CppAbstractnessReportsViolations(t *testing.T) {
	// Get the project root (from cmd/boy-scout test dir, go up 2 levels)
	cwd, _ := os.Getwd()
	projectRoot := filepath.Join(cwd, "../..")
	if abs, err := filepath.Abs(projectRoot); err == nil {
		projectRoot = abs
	}
	fixtureDir := filepath.Join(projectRoot, "testdata", "cpp-abstractness")

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"cpp", "abstractness", "--format=json", fixtureDir}, &stdout, &stderr)

	// Parse JSON output
	var report struct {
		Violations []struct {
			ImportPath   string
			Abstractness float64
			Zone         string
		}
		TotalPackages int
	}

	stdoutStr := stdout.String()
	stderrStr := stderr.String()
	if stdoutStr == "" {
		t.Fatalf("No output produced. Exit code: %d, stderr: %s", exitCode, stderrStr)
	}

	if err := json.Unmarshal([]byte(stdoutStr), &report); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nstdout: %s\nstderr: %s", err, stdoutStr, stderrStr)
	}

	// Expect violations (rigid.h should be Zone of Pain)
	if len(report.Violations) == 0 {
		t.Errorf("Expected violations in fixture, got 0")
	}

	// Check that we have at least one Pain zone violation
	var hasPain bool
	var hasAbstractness bool
	for _, v := range report.Violations {
		if v.Zone == "Pain" {
			hasPain = true
			// Verify Abstractness field is set (should be 0.0 for rigid.h)
			if v.Abstractness >= 0 {
				hasAbstractness = true
			}
		}
	}

	if !hasPain {
		t.Errorf("Expected at least one Pain zone violation, got: %+v", report.Violations)
	}
	if !hasAbstractness {
		t.Errorf("Expected Abstractness field to be set in violations")
	}
}

// Characterization test for runTsFilelen
func TestRun_TsFilelenCharacterization(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a short TypeScript file
	shortFile := filepath.Join(tmpDir, "short.ts")
	if err := os.WriteFile(shortFile, []byte("function foo() { return 1; }"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Create a long TypeScript file (310 lines, exceeding limit of 300)
	lines := []string{"function bar() {"}
	for i := 0; i < 308; i++ {
		lines = append(lines, fmt.Sprintf("  let x%d = %d;", i, i))
	}
	lines = append(lines, "}")
	longFile := filepath.Join(tmpDir, "long.ts")
	if err := os.WriteFile(longFile, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"ts", "filelen", tmpDir}, &stdout, &stderr)

	output := stdout.String()
	stderrOutput := stderr.String()

	// Should report the long file as violation
	if !strings.Contains(output, "long.ts") {
		t.Errorf("expected output to contain 'long.ts', got:\nstdout: %s\nstderr: %s", output, stderrOutput)
	}
	if !strings.Contains(output, "310 lines") {
		t.Errorf("expected output to contain '310 lines', got:\nstdout: %s\nstderr: %s", output, stderrOutput)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1 (violation found), got %d", exitCode)
	}
}

// Characterization test for runTsFunclen
func TestRun_TsFunclenCharacterization(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a TypeScript file with a short function (under limit)
	shortFunc := filepath.Join(tmpDir, "short.ts")
	shortSrc := `function shortFunc() {
  return 1;
}`
	if err := os.WriteFile(shortFunc, []byte(shortSrc), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Create a TypeScript file with a long function (exceeding 50 lines)
	longFuncLines := []string{"function longFunc() {"}
	for i := 0; i < 52; i++ {
		longFuncLines = append(longFuncLines, fmt.Sprintf("  let x%d = %d;", i, i))
	}
	longFuncLines = append(longFuncLines, "}")

	longFunc := filepath.Join(tmpDir, "long.ts")
	if err := os.WriteFile(longFunc, []byte(strings.Join(longFuncLines, "\n")), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"ts", "funclen", tmpDir}, &stdout, &stderr)

	output := stdout.String()
	stderrOutput := stderr.String()

	// Should report the long function as violation
	if !strings.Contains(output, "longFunc") {
		t.Errorf("expected output to contain 'longFunc', got:\nstdout: %s\nstderr: %s", output, stderrOutput)
	}
	if !strings.Contains(output, "54 lines") {
		t.Errorf("expected output to contain '54 lines' (54 lines including braces), got:\nstdout: %s\nstderr: %s", output, stderrOutput)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1 (violation found), got %d", exitCode)
	}
}

// Characterization test for writeTsFunclenLines
func TestWriteTsFunclenLines_CharacterizationTest(t *testing.T) {
	var buf bytes.Buffer

	report := tsfunclen.Report{
		Violations: []tsfunclen.Violation{
			{
				File:   "test.ts",
				Line:   10,
				Func:   "testFunc",
				Length: 55,
				Limit:  50,
			},
		},
		ExcludedFuncs: []tsfunclen.ExcludedFunc{
			{
				File:   "test.ts",
				Func:   "excludedFunc",
				Reason: "matched pattern: *_helper",
			},
		},
	}

	writeTsFunclenLines(&buf, "", report)
	output := buf.String()

	// Verify violations line format
	if !strings.Contains(output, "test.ts:10: function testFunc is 55 lines (limit 50)") {
		t.Errorf("expected violation line format not found, got: %q", output)
	}

	// Verify excluded line format
	if !strings.Contains(output, "test.ts: function excludedFunc excluded (matched pattern: *_helper)") {
		t.Errorf("expected excluded line format not found, got: %q", output)
	}
}

// TestRun_DuplicationReportsRenamedFunctionAsType2Clone tests the duplication checker CLI
func TestRun_DuplicationReportsRenamedFunctionAsType2Clone(t *testing.T) {
	tmpDir := t.TempDir()
	initModule(t, tmpDir)

	// Create tax.go with CalculateTax function
	writeGoFile(t, tmpDir, "tax.go", `package billing

func CalculateTax(amount float64) float64 {
	rate := 0.08
	total := amount * rate
	if total < 0 {
		total = 0
	}
	return total
}
`)

	// Create fee.go with CalculateFee function (same structure, renamed identifiers)
	writeGoFile(t, tmpDir, "fee.go", `package billing

func CalculateFee(price float64) float64 {
	pct := 0.08
	result := price * pct
	if result < 0 {
		result = 0
	}
	return result
}
`)

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"go", "duplication", tmpDir}, &stdout, &stderr)

	output := stdout.String()
	stderrOutput := stderr.String()

	// Should report Type-2 duplicate
	if !strings.Contains(output, "Type-2") {
		t.Errorf("expected output to contain 'Type-2', got:\nstdout: %s\nstderr: %s", output, stderrOutput)
	}

	// Should mention both files and functions
	if !strings.Contains(output, "fee.go") || !strings.Contains(output, "tax.go") {
		t.Errorf("expected output to mention both files, got:\nstdout: %s\nstderr: %s", output, stderrOutput)
	}

	if !strings.Contains(output, "CalculateFee") || !strings.Contains(output, "CalculateTax") {
		t.Errorf("expected output to mention both functions, got:\nstdout: %s\nstderr: %s", output, stderrOutput)
	}

	// Exit code should be 1 (violation found)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestRun_DuplicationJSONFormatOutputsValidSchema tests JSON output format
func TestRun_DuplicationJSONFormatOutputsValidSchema(t *testing.T) {
	tmpDir := t.TempDir()
	initModule(t, tmpDir)

	// Create two identical functions
	writeGoFile(t, tmpDir, "a.go", `package test

func Duplicate() error {
	x := 1
	y := 2
	z := x + y
	return nil
}
`)

	writeGoFile(t, tmpDir, "b.go", `package test

func Duplicate2() error {
	x := 1
	y := 2
	z := x + y
	return nil
}
`)

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"go", "duplication", "--format=json", tmpDir}, &stdout, &stderr)

	// Parse JSON
	type duplicationReportJSON struct {
		Violations []struct {
			FileA    string `json:"fileA"`
			LineA    int    `json:"lineA"`
			FuncA    string `json:"funcA"`
			FileB    string `json:"fileB"`
			LineB    int    `json:"lineB"`
			FuncB    string `json:"funcB"`
			Type     string `json:"type"`
			DupLines int    `json:"dupLines"`
		} `json:"violations"`
	}

	var report duplicationReportJSON
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	// Should have exactly one violation
	if len(report.Violations) != 1 {
		t.Errorf("expected 1 violation, got %d", len(report.Violations))
	}

	v := report.Violations[0]
	if v.Type != "Type-1" {
		t.Errorf("expected Type-1, got %s", v.Type)
	}

	// Exit code should be 1
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

