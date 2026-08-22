package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	Funclen struct {
		Violations []funcViolation `json:"violations"`
	} `json:"funclen"`
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
	exitCode := run([]string{"go", "funclen", tmpFile}, &stdoutBuf, &stderrBuf)

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

func TestRun_FunclenRespectsMaxLinesFlag(t *testing.T) {
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
	exitCode := run([]string{"go", "funclen", "--max-lines=50", tmpFile}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	// Should report violation with limit=50, not 100
	if !strings.Contains(output, "Moderate is 60 lines (limit 50)") {
		t.Errorf("expected output to contain 'Moderate is 60 lines (limit 50)', got:\n%s", output)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

func TestRun_FunclenDefaultsToCurrentDir(t *testing.T) {
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
	exitCode := run([]string{"go", "funclen"}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()
	stderr := stderrBuf.String()
	if !strings.Contains(output, "Big is 105 lines (limit 50)") {
		t.Errorf("expected output to contain 'Big is 105 lines (limit 50)', got stdout:\n%s\nstderr:\n%s", output, stderr)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

func TestRun_JSONFormatOutputsValidSchema(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := writeGoFile(t, tmpDir, "viol.go", longFuncSrc("main", "Violating", 105))

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "funclen", "--format=json", tmpFile}, &stdoutBuf, &stderrBuf)

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
			exitCode := run([]string{"go", "funclen", tmpDir}, &stdoutBuf, &stderrBuf)

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
	if !strings.Contains(stderr, "gardener <lang> <command>") || !strings.Contains(stderr, "languages: go") || !strings.Contains(stderr, "gardener setup") {
		t.Errorf("expected usage message with 'gardener <lang> <command>', 'languages: go' and 'gardener setup', got:\n%s", stderr)
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
		t.Errorf("expected funclen violation in 'all' output, got:\nstdout: %s\nstderr: %s", output, stderr)
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

func TestRun_AllCombinesFunclenAndCrap(t *testing.T) {
	// Create a file with both funclen and crap violations
	src := `package main
func ViolatingFunc() {
`
	// Add 105 lines to violate funclen
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
	// Should have both funclen and crap violations
	if !strings.Contains(output, "[funclen]") {
		t.Errorf("expected '[funclen]' in output, got:\n%s", output)
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
// combination of funclen/crap violations for the "all" subcommand.
func writeAllCheckFixtures(t *testing.T, dir string, funclenViolating, crapViolating bool) {
	if funclenViolating {
		writeGoFile(t, dir, "long.go", longFuncSrc("main", "LongFunc", 105))
	}
	if crapViolating {
		writeGoFile(t, dir, "complex.go", complexFuncSrc)
	}
	if !funclenViolating && !crapViolating {
		writeGoFile(t, dir, "clean.go", cleanFuncSrc)
	}
}

func TestRun_AllExitCodePriorityAcrossBothChecks(t *testing.T) {
	testCases := []struct {
		name             string
		funclenViolating bool
		crapViolating    bool
		expectedExitCode int
	}{
		{"both clean", false, false, 0},
		{"only funclen violated", true, false, 1},
		{"only crap violated", false, true, 1},
		{"both violated", true, true, 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			writeAllCheckFixtures(t, tmpDir, tc.funclenViolating, tc.crapViolating)
			initModule(t, tmpDir)

			var stdoutBuf, stderrBuf bytes.Buffer
			exitCode := run([]string{"go", "all", "."}, &stdoutBuf, &stderrBuf)

			if exitCode != tc.expectedExitCode {
				t.Errorf("expected exit code %d, got %d (output: %s)", tc.expectedExitCode, exitCode, stdoutBuf.String())
			}
		})
	}
}

func TestRun_ExcludeFlagsFilterFunclenOutput(t *testing.T) {
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
	exitCode := run([]string{"go", "funclen", "--exclude-file=*_test.go", "--exclude-func=Foo", tmpDir}, &stdoutBuf, &stderrBuf)

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
	exitCode := run([]string{"go", "funclen", "--exclude-file=[", tmpDir}, &stdoutBuf, &stderrBuf)

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
	exitCode := run([]string{"go", "funclen", "--exclude-file=*_test.go", "--debug", tmpDir}, &stdoutBuf, &stderrBuf)

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
	// Should have no violations from funclen (Foo excluded)
	if strings.Contains(output, "[funclen] foo.go") && strings.Contains(output, "Foo is 105 lines") {
		t.Errorf("expected funclen violation to be excluded, got:\n%s", output)
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
	exitCode := run([]string{"go", "funclen", "--exclude-func=Long", tmpDir}, &stdoutBuf, &stderrBuf)

	// Exit code should be 0 (clean) even though a real violation was excluded
	if exitCode != 0 {
		t.Errorf("expected exit code 0 (excluded violation doesn't count), got %d", exitCode)
	}
}

func TestRun_SetupLocalWritesRelativeSkillFile(t *testing.T) {
	tmpDir := t.TempDir()

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
	exitCode := run([]string{"setup"}, &stdoutBuf, &stderrBuf)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d (stderr: %s)", exitCode, stderrBuf.String())
	}

	stdout := stdoutBuf.String()
	if !strings.Contains(stdout, "gardener skill installed:") {
		t.Errorf("expected 'gardener skill installed:' in stdout, got: %s", stdout)
	}

	// Verify the skill file exists
	skillPath := filepath.Join(tmpDir, ".agents", "skills", "gardener", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatalf("skill file not found at %q: %v", skillPath, err)
	}

	// Verify the binary was copied
	binPath := filepath.Join(tmpDir, ".agents", "bin", "gardener")
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("binary not found at %q: %v", binPath, err)
	}
}

func TestRun_SetupGlobalWritesToHomeDir(t *testing.T) {
	// Create a temporary "home" directory
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"setup", "--global"}, &stdoutBuf, &stderrBuf)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d (stderr: %s)", exitCode, stderrBuf.String())
	}

	// Verify the skill file exists in the home directory
	skillPath := filepath.Join(homeDir, ".agents", "skills", "gardener", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatalf("skill file not found at %q: %v", skillPath, err)
	}

	// Verify the binary was copied
	binPath := filepath.Join(homeDir, ".agents", "bin", "gardener")
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("binary not found at %q: %v", binPath, err)
	}

	stdout := stdoutBuf.String()
	if !strings.Contains(stdout, homeDir) {
		t.Errorf("expected home dir %q in stdout, got: %s", homeDir, stdout)
	}
}

func TestRun_SetupRejectsExtraArgs(t *testing.T) {
	tmpDir := t.TempDir()

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
	exitCode := run([]string{"setup", "extra-arg"}, &stdoutBuf, &stderrBuf)

	if exitCode != 2 {
		t.Errorf("expected exit code 2 for usage error, got %d", exitCode)
	}

	stderr := stderrBuf.String()
	if !strings.Contains(stderr, "usage") {
		t.Errorf("expected usage message in stderr, got: %s", stderr)
	}

	// Verify no .agents directory was created
	agentDir := filepath.Join(tmpDir, ".agents")
	if _, err := os.Stat(agentDir); err == nil {
		t.Errorf("expected .agent directory not to be created for invalid args")
	}
}

func TestRun_SetupWriteFailureExitsTwo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file at the .agents path to block directory creation
	blockingFile := filepath.Join(tmpDir, ".agents")
	if err := os.WriteFile(blockingFile, []byte("blocking"), 0644); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}

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
	exitCode := run([]string{"setup"}, &stdoutBuf, &stderrBuf)

	if exitCode != 2 {
		t.Errorf("expected exit code 2 for write failure, got %d", exitCode)
	}

	stderr := stderrBuf.String()
	if !strings.Contains(stderr, "error:") {
		t.Errorf("expected error message in stderr, got: %s", stderr)
	}
}

func TestRun_AllRespectsPerCheckerIgnoreComment(t *testing.T) {
	// Create a fixture with a function long enough for funclen (105 lines)
	// and complex enough for crap (5 nested ifs = complexity 5, with no coverage = score = 5^2*(1-0)^3 + 5 = 30, way above 6.0 threshold)
	src := "package main\n\n// gardener:ignore:crap\nfunc ViolatingFunc() {\n"
	for i := 0; i < 100; i++ {
		src += fmt.Sprintf("\t_ = %d\n", i)
	}
	src += "\tif true {\n\t\tif true {\n\t\t\tif true {\n\t\t\t\tif true {\n\t\t\t\t\tif true {\n\t\t\t\t\t\t_ = \"x\"\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t}\n}\n"

	tmpDir := t.TempDir()
	writeGoFile(t, tmpDir, "main.go", src)
	initModule(t, tmpDir)

	report, exitCode := runAllJSON(t)

	// funclen should report ViolatingFunc (directive doesn't name funclen)
	if !hasViolation(report.Funclen.Violations, "ViolatingFunc") {
		t.Errorf("expected ViolatingFunc in funclen violations, got %v", report.Funclen.Violations)
	}

	// crap should NOT report ViolatingFunc (directive names crap, so it's excluded)
	if hasViolation(report.Crap.Violations, "ViolatingFunc") {
		t.Errorf("expected ViolatingFunc NOT in crap violations, but found it")
	}

	// Exit code should be 1 (violation from funclen)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

func TestRun_CrapIgnoresTestFilesByDefaultOnCLI(t *testing.T) {
	tmpDir := t.TempDir()

	// Write main.go with a trivial function
	writeGoFile(t, tmpDir, "main.go", `package main
func Add(a, b int) int {
	return a + b
}
`)

	// Write main_test.go with an untested, deeply-nested helper function
	writeGoFile(t, tmpDir, "main_test.go", `package main
import "testing"
func TestAdd(t *testing.T) {
	_ = Add(1, 2)
}
func chdirTemp() {
	if true { if true { if true { if true { if true { } } } } }
}
`)

	// Write go.mod
	writeGoFile(t, tmpDir, "go.mod", "module test\n\ngo 1.24\n")

	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Run crap with low threshold, no --exclude-file flag
	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "crap", "--threshold=1", "."}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	// Assert that chdirTemp from main_test.go is NOT in the output
	if strings.Contains(output, "chdirTemp") {
		t.Errorf("expected chdirTemp not to be reported, but found it in output:\n%s", output)
	}

	// Exit code should be 0 (no violations, since the only complex function is in excluded test file)
	if exitCode != 0 {
		t.Errorf("expected exit code 0 (clean), got %d", exitCode)
	}
}

func TestRun_AllCrapSectionIgnoresTestFilesByDefault(t *testing.T) {
	tmpDir := t.TempDir()

	writeGoFile(t, tmpDir, "main.go", `package main
func Add(a, b int) int {
	return a + b
}
`)

	// main_test.go has an untested, deeply-nested helper function
	writeGoFile(t, tmpDir, "main_test.go", `package main
import "testing"
func TestAdd(t *testing.T) {
	_ = Add(1, 2)
}
func chdirTemp() {
	if true { if true { if true { if true { if true { } } } } }
}
`)

	initModule(t, tmpDir)

	// Run "all" with JSON format (uses default thresholds: funclen=50, crap=6.0)
	report, exitCode := runAllJSON(t)

	if hasViolation(report.Crap.Violations, "chdirTemp") {
		t.Errorf("expected chdirTemp not to be reported in crap section, but found it")
	}

	if exitCode != 0 {
		t.Errorf("expected exit code 0 (clean), got %d", exitCode)
	}
}

func TestRun_CppFunclenReportsViolation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a C++ file with a function exceeding the default 50-line limit
	code := `void longFunction() {
  int x = 1;
  int y = 2;
  int z = 3;
  int a = 4;
  int b = 5;
  int c = 6;
  int d = 7;
  int e = 8;
  int f = 9;
  int g = 10;
  int h = 11;
  int i = 12;
  int j = 13;
  int k = 14;
  int l = 15;
  int m = 16;
  int n = 17;
  int o = 18;
  int p = 19;
  int q = 20;
  int r = 21;
  int s = 22;
  int t = 23;
  int u = 24;
  int v = 25;
  int w = 26;
  int x1 = 27;
  int x2 = 28;
  int x3 = 29;
  int x4 = 30;
  int x5 = 31;
  int x6 = 32;
  int x7 = 33;
  int x8 = 34;
  int x9 = 35;
  int x10 = 36;
  int x11 = 37;
  int x12 = 38;
  int x13 = 39;
  int x14 = 40;
  int x15 = 41;
  int x16 = 42;
  int x17 = 43;
  int x18 = 44;
  int x19 = 45;
  int x20 = 46;
  int x21 = 47;
  int x22 = 48;
  int x23 = 49;
  int x24 = 50;
  int x25 = 51;
}`

	cppFile := filepath.Join(tmpDir, "test.cpp")
	if err := os.WriteFile(cppFile, []byte(code), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"cpp", "funclen", tmpDir}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	// Should report the violation
	if !strings.Contains(output, "longFunction") {
		t.Errorf("expected output to contain 'longFunction', got:\n%s", output)
	}
	if !strings.Contains(output, "limit 50") {
		t.Errorf("expected output to contain 'limit 50', got:\n%s", output)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d (stderr: %s)", exitCode, stderrBuf.String())
	}
}

func TestRun_CppSyntaxErrorFileSkippedExitsTwo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a C++ file with syntax error (unclosed comment)
	code := `void brokenFunction() {
  int x = 1;
  /* unclosed comment`

	cppFile := filepath.Join(tmpDir, "broken.cpp")
	if err := os.WriteFile(cppFile, []byte(code), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"cpp", "funclen", tmpDir}, &stdoutBuf, &stderrBuf)

	if exitCode != 2 {
		t.Errorf("expected exit code 2 (skipped file takes priority), got %d", exitCode)
	}
}

func TestRun_CppUnregisteredSubcommandExitsUsageError(t *testing.T) {
	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"cpp", "crap"}, &stdoutBuf, &stderrBuf)

	stderr := stderrBuf.String()
	if !strings.Contains(stderr, "unknown subcommand for cpp: crap") {
		t.Errorf("expected 'unknown subcommand for cpp: crap' in stderr, got:\n%s", stderr)
	}

	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}
}
