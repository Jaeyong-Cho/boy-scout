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

	deepcacheDir := filepath.Join(tmpDir, "deepcache")
	if err := os.MkdirAll(deepcacheDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

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

	var testFuncs strings.Builder
	testFuncs.WriteString("package deepcache\n\nimport \"testing\"\n\n")
	for i := 1; i <= 25; i++ {
		fmt.Fprintf(&testFuncs, "func TestBehavior%d(t *testing.T) {}\n", i)
	}
	writeGoFile(t, deepcacheDir, "deepcache_test.go", testFuncs.String())

	for i := 1; i <= 8; i++ {
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

	var stdoutBuf, stderrBuf bytes.Buffer
	run([]string{"go", "abstractness", "--format=json", tmpDir}, &stdoutBuf, &stderrBuf)

	var report struct {
		Violations []struct {
			ImportPath   string
			SurfaceRatio float64
		}
	}
	if err := json.Unmarshal(stdoutBuf.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\noutput: %s", err, stdoutBuf.String())
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
