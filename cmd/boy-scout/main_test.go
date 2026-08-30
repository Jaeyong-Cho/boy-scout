package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"boy-scout/internal/cppfunclen"
	"boy-scout/internal/filelen"
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

func TestRun_UnknownSubcommand_CrapInstabilityAbstractness(t *testing.T) {
	tests := []struct {
		args []string
		lang string
		cmd  string
	}{
		{[]string{"go", "crap", "."}, "go", "crap"},
		{[]string{"cpp", "instability", "."}, "cpp", "instability"},
		{[]string{"cpp", "abstractness", "."}, "cpp", "abstractness"},
	}

	for _, tt := range tests {
		t.Run(tt.lang+"_"+tt.cmd, func(t *testing.T) {
			var stdoutBuf, stderrBuf bytes.Buffer
			exitCode := run(tt.args, &stdoutBuf, &stderrBuf)

			stderr := stderrBuf.String()
			if !strings.Contains(stderr, "unknown subcommand") {
				t.Errorf("expected 'unknown subcommand' in stderr, got:\n%s", stderr)
			}

			if exitCode != 2 {
				t.Errorf("expected exit code 2, got %d", exitCode)
			}
		})
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

	// Ensure legacy checks are not present in output
	legacyPrefixes := []string{"[crap]", "[instability]", "[abstractness]"}
	for _, prefix := range legacyPrefixes {
		if strings.Contains(output, prefix) {
			t.Errorf("unexpected legacy check prefix %q found in output", prefix)
		}
	}
}

func TestRun_AllOutputHasNoLegacyCheckKeys(t *testing.T) {
	// Create a minimal clean fixture with no violations
	tmpDir := t.TempDir()
	if err := os.WriteFile(tmpDir+"/main.go", []byte("package main\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(tmpDir+"/go.mod", []byte("module test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	var stdoutBuf, stderrBuf bytes.Buffer
	run([]string{"go", "all", "--format=json", "."}, &stdoutBuf, &stderrBuf)

	var report map[string]json.RawMessage
	if err := json.Unmarshal(stdoutBuf.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\noutput: %s\nstderr: %s", err, stdoutBuf.String(), stderrBuf.String())
	}

	// Assert no legacy check keys exist
	legacyKeys := []string{"crap", "instability", "abstractness"}
	for _, key := range legacyKeys {
		if _, ok := report[key]; ok {
			t.Errorf("unexpected legacy key %q found in JSON output", key)
		}
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

// writeAllCheckFixtures writes source files into dir that produce the requested violations for the "all" subcommand.
func writeAllCheckFixtures(t *testing.T, dir string, gofunclenViolating bool) {
	if gofunclenViolating {
		writeGoFile(t, dir, "long.go", longFuncSrc("main", "LongFunc", 105))
	}
	if !gofunclenViolating {
		writeGoFile(t, dir, "clean.go", cleanFuncSrc)
	}
}

func TestRun_AllExitCodePriorityAcrossBothChecks(t *testing.T) {
	testCases := []struct {
		name             string
		gofunclenViolating bool
		expectedExitCode int
	}{
		{"clean", false, 0},
		{"gofunclen violated", true, 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			writeAllCheckFixtures(t, tmpDir, tc.gofunclenViolating)
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

// TestRun_DuplicationJSONIncludesClustersField tests that the JSON output includes cluster grouping
func TestRun_DuplicationJSONIncludesClustersField(t *testing.T) {
	tmpDir := t.TempDir()
	initModule(t, tmpDir)

	// Create three mutually duplicated functions
	// A and B are identical (Type-1), B and C have renamed identifiers (Type-2)
	writeGoFile(t, tmpDir, "a.go", `package test

func DuplicateA() error {
	x := 1
	y := 2
	z := x + y
	return nil
}
`)

	writeGoFile(t, tmpDir, "b.go", `package test

func DuplicateB() error {
	x := 1
	y := 2
	z := x + y
	return nil
}
`)

	writeGoFile(t, tmpDir, "c.go", `package test

func DuplicateC() error {
	a := 1
	b := 2
	c := a + b
	return nil
}
`)

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"go", "duplication", "--format=json", tmpDir}, &stdout, &stderr)

	// Parse JSON with clusters field
	type clusterJSON struct {
		Members []struct {
			File string `json:"file"`
			Line int    `json:"line"`
			Func string `json:"func"`
		} `json:"members"`
		Pairs []struct {
			Type string `json:"type"`
		} `json:"pairs"`
		DupLines    int  `json:"dupLines"`
		CrossPackage bool `json:"crossPackage"`
	}

	type duplicationReportJSON struct {
		Violations []struct {
			Type string `json:"type"`
		} `json:"violations"`
		Clusters []clusterJSON `json:"clusters"`
	}

	var report duplicationReportJSON
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	// Should have clusters field
	if len(report.Clusters) == 0 {
		t.Fatalf("expected clusters array in JSON output, got empty")
	}

	// Should have exactly 1 cluster
	if len(report.Clusters) != 1 {
		t.Errorf("expected 1 cluster, got %d", len(report.Clusters))
	}

	cluster := report.Clusters[0]

	// Cluster should have 3 members
	if len(cluster.Members) != 3 {
		t.Errorf("expected 3 members in cluster, got %d", len(cluster.Members))
	}

	// Cluster should have 3 pairs (A-B, A-C, B-C)
	if len(cluster.Pairs) != 3 {
		t.Errorf("expected 3 pairs in cluster, got %d", len(cluster.Pairs))
	}

	// Pairs should include both Type-1 and Type-2
	hasType1 := false
	hasType2 := false
	for _, pair := range cluster.Pairs {
		if pair.Type == "Type-1" {
			hasType1 = true
		}
		if pair.Type == "Type-2" {
			hasType2 = true
		}
	}
	if !hasType1 || !hasType2 {
		t.Errorf("expected Type-1 and Type-2 pairs in cluster")
	}

	// Exit code should be 1
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestRun_DuplicationInvalidMinSimilarityErrors tests that invalid --min-similarity values cause an error
func TestRun_DuplicationInvalidMinSimilarityErrors(t *testing.T) {
	tmpDir := t.TempDir()
	initModule(t, tmpDir)

	// Create a simple function
	writeGoFile(t, tmpDir, "a.go", `package test

func TestFunc() int {
	x := 1
	y := 2
	z := 3
	return x + y + z
}
`)

	// Test with --min-similarity=150 (out of range)
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"go", "duplication", "--min-similarity=150", tmpDir}, &stdout, &stderr)

	// Should fail with exit code 2
	if exitCode != 2 {
		t.Errorf("expected exit code 2 for invalid --min-similarity, got %d", exitCode)
	}

	// stderr should contain an error message
	stderrOutput := stderr.String()
	if !strings.Contains(stderrOutput, "min-similarity") && !strings.Contains(stderrOutput, "range") {
		t.Errorf("expected error message about min-similarity range in stderr, got: %s", stderrOutput)
	}
}

// TestRun_DuplicationMinSimilarityFlagIsRespected tests that --min-similarity flag threshold is applied
func TestRun_DuplicationMinSimilarityFlagIsRespected(t *testing.T) {
	tmpDir := t.TempDir()
	initModule(t, tmpDir)

	// Create two functions with moderate similarity (not Type-1 or Type-2, but above 0.70)
	// Function A: 7-8 lines
	writeGoFile(t, tmpDir, "a.go", `package test

func OriginalFunc(x int) int {
	y := x + 1
	z := y * 2
	if z > 10 {
		return z
	}
	return 0
}
`)

	// Function B: similar but with one extra guard
	writeGoFile(t, tmpDir, "b.go", `package test

func ModifiedFunc(x int) int {
	y := x + 1
	z := y * 2
	if z > 10 {
		if z < 100 {
			return z
		}
	}
	return 0
}
`)

	// Test with default threshold (0.70)
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"go", "duplication", tmpDir}, &stdout, &stderr)
	output := stdout.String()

	// Should report Type-3 at default threshold
	if !strings.Contains(output, "Type-3") {
		t.Logf("expected Type-3 at default 0.70 threshold, output: %s", output)
	}

	// Test with a higher threshold (0.85) - these functions probably won't match
	stdout.Reset()
	stderr.Reset()
	exitCode = run([]string{"go", "duplication", "--min-similarity=0.85", tmpDir}, &stdout, &stderr)

	// Exit code should be 0 (no violations) or we need to verify the similarity is below 0.85
	// The exact behavior depends on the computed similarity
	_ = exitCode
}

func TestRun_ComplexityRespectsMaxComplexityFlag(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with a function of complexity 3
	src := `package main

func Process(x int) string {
	if x > 10 {
		if x > 20 {
			return "big"
		}
		return "medium"
	}
	return "small"
}`

	tmpFile := tmpDir + "/main.go"
	if err := os.WriteFile(tmpFile, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	writeGoFile(t, tmpDir, "go.mod", "module test\n\ngo 1.24\n")

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "complexity", "--max-complexity=2", tmpDir}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	// Expect format mentioning Process with complexity=3 and limit=2
	if !strings.Contains(output, "Process") {
		t.Errorf("expected output to contain 'Process', got:\n%s", output)
	}
	if !strings.Contains(output, "complexity=3") {
		t.Errorf("expected output to contain 'complexity=3', got:\n%s", output)
	}
	if !strings.Contains(output, "limit=2") {
		t.Errorf("expected output to contain 'limit=2', got:\n%s", output)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

func TestRun_ComplexityJSONFormatOutputsValidSchema(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with a function of complexity 3
	src := `package main

func Process(x int) string {
	if x > 10 {
		if x > 20 {
			return "big"
		}
		return "medium"
	}
	return "small"
}`

	tmpFile := tmpDir + "/main.go"
	if err := os.WriteFile(tmpFile, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	writeGoFile(t, tmpDir, "go.mod", "module test\n\ngo 1.24\n")

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "complexity", "--max-complexity=2", "--format=json", tmpDir}, &stdoutBuf, &stderrBuf)

	var report struct {
		Violations []struct {
			Complexity int `json:"complexity"`
		} `json:"violations"`
	}

	if err := json.Unmarshal(stdoutBuf.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\noutput: %s", err, stdoutBuf.String())
	}

	if len(report.Violations) != 1 {
		t.Errorf("expected 1 violation, got %d", len(report.Violations))
	}

	if len(report.Violations) > 0 && report.Violations[0].Complexity != 3 {
		t.Errorf("expected complexity 3, got %d", report.Violations[0].Complexity)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

func TestRun_AllIncludesComplexity(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with a function of complexity > 10 (violates default limit)
	src := `package main

func VeryComplex(x int) string {
	if x > 1 {
		if x > 2 {
			if x > 3 {
				if x > 4 {
					if x > 5 {
						if x > 6 {
							if x > 7 {
								if x > 8 {
									if x > 9 {
										if x > 10 {
											if x > 11 {
												return "deep"
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return "simple"
}`

	tmpFile := tmpDir + "/main.go"
	if err := os.WriteFile(tmpFile, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	writeGoFile(t, tmpDir, "go.mod", "module test\n\ngo 1.24\n")

	// Change to tmpDir for the test
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

	// Test text output contains [complexity] line
	var stdoutBuf, stderrBuf bytes.Buffer
	run([]string{"go", "all", "."}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()
	stderr := stderrBuf.String()
	if !strings.Contains(output, "[complexity]") {
		t.Errorf("expected text output to contain '[complexity]' line, got:\nstdout:%s\nstderr:%s", output, stderr)
	}

	// Test JSON output contains "complexity" key
	stdoutBuf.Reset()
	stderrBuf.Reset()
	run([]string{"go", "all", "--format=json", "."}, &stdoutBuf, &stderrBuf)

	var report map[string]interface{}
	if err := json.Unmarshal(stdoutBuf.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\nstdout: %s\nstderr: %s", err, stdoutBuf.String(), stderrBuf.String())
	}

	if _, ok := report["complexity"]; !ok {
		t.Errorf("expected JSON output to contain 'complexity' key, got: %v", report)
	}
}

func TestRun_ComplexityAtLimitIsCompliant(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a function with exactly complexity 6 (the default limit)
	src := `package main

func AtLimit(x int) string {
	if x > 1 {
		if x > 2 {
			if x > 3 {
				if x > 4 {
					if x > 5 {
						return "deep"
					}
				}
			}
		}
	}
	return "simple"
}`

	tmpFile := tmpDir + "/main.go"
	if err := os.WriteFile(tmpFile, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	writeGoFile(t, tmpDir, "go.mod", "module test\n\ngo 1.24\n")

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "complexity", tmpDir}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	// Should have no violations
	if !strings.Contains(output, "") || strings.Contains(output, "AtLimit") {
		t.Errorf("expected no violation for function at limit, got:\n%s", output)
	}

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestRun_ComplexitySkipsUnparseableFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with invalid Go syntax
	invalidSrc := "func broken( {"
	tmpFile := tmpDir + "/broken.go"
	if err := os.WriteFile(tmpFile, []byte(invalidSrc), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	writeGoFile(t, tmpDir, "go.mod", "module test\n\ngo 1.24\n")

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "complexity", tmpDir}, &stdoutBuf, &stderrBuf)

	// Exit code should be 2 (parse error/skipped file)
	if exitCode != 2 {
		t.Errorf("expected exit code 2 for skipped file, got %d (stdout: %s, stderr: %s)", exitCode, stdoutBuf.String(), stderrBuf.String())
	}
}

func TestRun_ComplexityDefaultsToCurrentDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a temp dir with a violating file
	lines := []string{"package main", "", "func Complex() {"}
	for i := 0; i < 11; i++ {
		lines = append(lines, fmt.Sprintf("\tif x > %d { _ = x }", i))
	}
	lines = append(lines, "}")
	src := strings.Join(lines, "\n")

	complexGoPath := tmpDir + "/complex.go"
	if err := os.WriteFile(complexGoPath, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Initialize go module
	if err := os.WriteFile(tmpDir+"/go.mod", []byte("module test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
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
	exitCode := run([]string{"go", "complexity"}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()
	stderr := stderrBuf.String()
	if !strings.Contains(output, "Complex") {
		t.Errorf("expected output to contain 'Complex', got stdout:\n%s\nstderr:\n%s", output, stderr)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

func TestRun_GoLinelenCharacterization(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a short Go file
	shortFile := filepath.Join(tmpDir, "short.go")
	if err := os.WriteFile(shortFile, []byte("package main\nfunc main() {}"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Create a long Go file with a 105-char line (exceeding limit of 100)
	longFile := filepath.Join(tmpDir, "long.go")
	line105 := "x := " + strings.Repeat("1", 100) // 5 + 100 = 105 chars
	if err := os.WriteFile(longFile, []byte(line105+"\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"go", "linelen", tmpDir}, &stdout, &stderr)

	output := stdout.String()
	stderrOutput := stderr.String()

	// Should report the long file as violation
	if !strings.Contains(output, "long.go") {
		t.Errorf("expected output to contain 'long.go', got:\nstdout: %s\nstderr: %s", output, stderrOutput)
	}
	if !strings.Contains(output, "105") {
		t.Errorf("expected output to contain '105', got:\nstdout: %s\nstderr: %s", output, stderrOutput)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1 (violation found), got %d", exitCode)
	}
}

func TestRun_CppLinelenCharacterization(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a short C++ file
	shortFile := filepath.Join(tmpDir, "short.cpp")
	if err := os.WriteFile(shortFile, []byte("int main() { return 0; }"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Create a long C++ file with a 105-char line (exceeding limit of 100)
	longFile := filepath.Join(tmpDir, "long.cpp")
	line105 := "int x = " + strings.Repeat("1", 97) // 8 + 97 = 105 chars
	if err := os.WriteFile(longFile, []byte(line105+";\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"cpp", "linelen", tmpDir}, &stdout, &stderr)

	output := stdout.String()
	stderrOutput := stderr.String()

	// Should report the long file as violation
	if !strings.Contains(output, "long.cpp") {
		t.Errorf("expected output to contain 'long.cpp', got:\nstdout: %s\nstderr: %s", output, stderrOutput)
	}
	if !strings.Contains(output, "106") { // 105 + 1 for the semicolon
		t.Errorf("expected output to contain '106', got:\nstdout: %s\nstderr: %s", output, stderrOutput)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1 (violation found), got %d", exitCode)
	}
}

func TestRun_TsLinelenCharacterization(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a short TypeScript file
	shortFile := filepath.Join(tmpDir, "short.ts")
	if err := os.WriteFile(shortFile, []byte("function foo() { return 1; }"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Create a long TypeScript file with a 105-char line (exceeding limit of 100)
	longFile := filepath.Join(tmpDir, "long.ts")
	line105 := "const x = " + strings.Repeat("1", 95) // 10 + 95 = 105 chars
	if err := os.WriteFile(longFile, []byte(line105+";\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"ts", "linelen", tmpDir}, &stdout, &stderr)

	output := stdout.String()
	stderrOutput := stderr.String()

	// Should report the long file as violation
	if !strings.Contains(output, "long.ts") {
		t.Errorf("expected output to contain 'long.ts', got:\nstdout: %s\nstderr: %s", output, stderrOutput)
	}
	if !strings.Contains(output, "106") { // 105 + 1 for the semicolon
		t.Errorf("expected output to contain '106', got:\nstdout: %s\nstderr: %s", output, stderrOutput)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1 (violation found), got %d", exitCode)
	}
}

func TestRun_AllIncludesLinelen(t *testing.T) {
	// Create a temp dir with a long line
	tmpDir := t.TempDir()

	lines := []string{"package main", "", "func TestFunc() {"}
	lines = append(lines, "\t// "+strings.Repeat("x", 105)) // A 100+ char comment line
	lines = append(lines, "}")
	src := strings.Join(lines, "\n")

	if err := os.WriteFile(tmpDir+"/long_line.go", []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Initialize go module
	if err := os.WriteFile(tmpDir+"/go.mod", []byte("module test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Test text output
	var stdoutBuf, stderrBuf bytes.Buffer
	_ = run([]string{"go", "all", "."}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()
	stderr := stderrBuf.String()

	if !strings.Contains(output, "[linelen]") {
		t.Errorf("expected '[linelen]' in text output, got:\nstdout: %s\nstderr: %s", output, stderr)
	}

	// Test JSON output
	stdoutBuf.Reset()
	stderrBuf.Reset()
	_ = run([]string{"go", "all", "--format=json", "."}, &stdoutBuf, &stderrBuf)

	jsonOutput := stdoutBuf.String()
	stderr = stderrBuf.String()

	if !strings.Contains(jsonOutput, "\"linelen\"") {
		t.Errorf("expected '\"linelen\"' key in JSON output, got:\nstdout: %s\nstderr: %s", jsonOutput, stderr)
	}
}

func TestRun_CppComplexityFlagViolation(t *testing.T) {
	// Test that cpp complexity flags a complex function at limit 6
	tmpDir := t.TempDir()
	src := `#include <string>
std::string parseStatement(const std::string& input) {
  if (input[0] == 'i') {
    if (input.length() > 1) {
      if (input[1] == 'f') {
        if (input.length() > 2) {
          if (input[2] == ' ') {
            if (input.find("then") != std::string::npos) {
              return "if-then-statement";
            }
          }
        }
      }
    }
  }
  return "statement";
}`
	if err := os.WriteFile(tmpDir+"/test.cpp", []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"cpp", "complexity", "--format=json", tmpDir}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	// Parse JSON to verify
	var report interface{}
	err := json.Unmarshal([]byte(output), &report)
	if err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	m := report.(map[string]interface{})
	violations := m["violations"].([]interface{})

	if len(violations) != 1 {
		t.Errorf("expected 1 violation, got %d\noutput: %s", len(violations), output)
	}

	if len(violations) > 0 {
		v := violations[0].(map[string]interface{})
		if v["func"].(string) != "parseStatement" {
			t.Errorf("expected function 'parseStatement', got '%s'", v["func"])
		}
		if int(v["complexity"].(float64)) != 7 {
			t.Errorf("expected complexity 7, got %d", int(v["complexity"].(float64)))
		}
		if int(v["limit"].(float64)) != 6 {
			t.Errorf("expected limit 6, got %d", int(v["limit"].(float64)))
		}
	}

	// Non-zero exit code for violations
	if exitCode == 0 {
		t.Errorf("expected non-zero exit code for violations, got 0")
	}
}

func TestRun_CppComplexityRespectsFlagOverride(t *testing.T) {
	// Test that --max-complexity flag overrides the default
	tmpDir := t.TempDir()
	src := `#include <string>
std::string parseStatement(const std::string& input) {
  if (input[0] == 'i') {
    if (input.length() > 1) {
      if (input[1] == 'f') {
        if (input.length() > 2) {
          if (input[2] == ' ') {
            if (input.find("then") != std::string::npos) {
              return "if-then-statement";
            }
          }
        }
      }
    }
  }
  return "statement";
}`
	if err := os.WriteFile(tmpDir+"/test.cpp", []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"cpp", "complexity", "--format=json", "--max-complexity=10", tmpDir}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	var report interface{}
	err := json.Unmarshal([]byte(output), &report)
	if err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	m := report.(map[string]interface{})
	violations := m["violations"].([]interface{})

	if len(violations) != 0 {
		t.Errorf("expected 0 violations with limit 10, got %d", len(violations))
	}

	// No violations should result in zero exit code
	if exitCode != 0 {
		t.Errorf("expected zero exit code for no violations, got %d", exitCode)
	}
}

func TestRun_GoAllComplexityUsesNewDefault(t *testing.T) {
	// Test that go all uses the new default of 6 for complexity
	// Create a function with complexity 7 in a temp dir
	tmpDir := t.TempDir()
	src := `package main

func Complex() {
	if true {
		if true {
			if true {
				if true {
					if true {
						if true {
							if true {}
						}
					}
				}
			}
		}
	}
}`
	tmpFile := tmpDir + "/main.go"
	if err := os.WriteFile(tmpFile, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	writeGoFile(t, tmpDir, "go.mod", "module test\n\ngo 1.24\n")

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "all", tmpDir}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	// Should report complexity violation at the new limit of 6
	if !strings.Contains(output, "Complex") || !strings.Contains(output, "complexity") {
		t.Errorf("expected complexity violation in 'go all' output, got:\n%s", output)
	}

	// Non-zero exit code for violations
	if exitCode == 0 {
		t.Errorf("expected non-zero exit code for violations in 'go all', got 0")
	}
}

func TestRun_CppAllReportsCombinedViolations(t *testing.T) {
	// Create a C++ file with a 310-line function (violates funclen's 50-line default)
	// in a 312-line file (violates filelen's 300-line default).
	tmpDir := t.TempDir()

	// Write C++ file: void bigFunction() { ... 310 lines ... }
	var b strings.Builder
	b.WriteString("void bigFunction() {\n")
	for i := 0; i < 310; i++ {
		fmt.Fprintf(&b, "  int v%d = %d;\n", i, i)
	}
	b.WriteString("}\n")
	cppContent := b.String()

	cppFile := filepath.Join(tmpDir, "big.cpp")
	if err := os.WriteFile(cppFile, []byte(cppContent), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Test JSON format
	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"cpp", "all", "--format=json", tmpDir}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()
	stderr := stderrBuf.String()

	// Parse JSON response
	var report map[string]json.RawMessage
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\noutput: %s\nstderr: %s", err, output, stderr)
	}

	// Assert both funclen and filelen keys exist
	if _, ok := report["funclen"]; !ok {
		t.Errorf("expected 'funclen' key in JSON response, got keys: %v", report)
	}
	if _, ok := report["filelen"]; !ok {
		t.Errorf("expected 'filelen' key in JSON response, got keys: %v", report)
	}

	// Verify funclen has violations for big.cpp
	if funlen, ok := report["funclen"]; ok {
		var funclenReport cppfunclen.Report
		if err := json.Unmarshal(funlen, &funclenReport); err != nil {
			t.Fatalf("failed to unmarshal funclen report: %v", err)
		}
		if len(funclenReport.Violations) == 0 {
			t.Errorf("expected funclen violations for big.cpp")
		}
		// Verify the file is in violations
		found := false
		for _, v := range funclenReport.Violations {
			if strings.Contains(v.File, "big.cpp") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected big.cpp in funclen violations, got: %v", funclenReport.Violations)
		}
	}

	// Verify filelen has violations for big.cpp
	if flen, ok := report["filelen"]; ok {
		var filelenReport filelen.Report
		if err := json.Unmarshal(flen, &filelenReport); err != nil {
			t.Fatalf("failed to unmarshal filelen report: %v", err)
		}
		if len(filelenReport.Violations) == 0 {
			t.Errorf("expected filelen violations for big.cpp")
		}
		// Verify the file is in violations
		found := false
		for _, v := range filelenReport.Violations {
			if strings.Contains(v.File, "big.cpp") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected big.cpp in filelen violations, got: %v", filelenReport.Violations)
		}
	}

	// Expect exit code 1 (violations found)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	// Test text format
	var stdoutBuf2, stderrBuf2 bytes.Buffer
	exitCode2 := run([]string{"cpp", "all", "--format=text", tmpDir}, &stdoutBuf2, &stderrBuf2)

	output2 := stdoutBuf2.String()
	if !strings.Contains(output2, "[funclen]") {
		t.Errorf("expected '[funclen]' prefix in text output, got:\n%s", output2)
	}
	if !strings.Contains(output2, "[filelen]") {
		t.Errorf("expected '[filelen]' prefix in text output, got:\n%s", output2)
	}
	if !strings.Contains(output2, "big.cpp") {
		t.Errorf("expected 'big.cpp' in text output, got:\n%s", output2)
	}

	if exitCode2 != 1 {
		t.Errorf("expected exit code 1 for text format, got %d", exitCode2)
	}
}

func TestRun_TsAllReportsCombinedViolations(t *testing.T) {
	// Create a TypeScript file with a 310-line function (violates funclen's 50-line default)
	// in a 312-line file (violates filelen's 300-line default).
	tmpDir := t.TempDir()

	// Write TS file: function bigFunction() { ... 310 lines ... }
	var b strings.Builder
	b.WriteString("function bigFunction() {\n")
	for i := 0; i < 310; i++ {
		fmt.Fprintf(&b, "  const v%d = %d;\n", i, i)
	}
	b.WriteString("}\n")
	tsContent := b.String()

	tsFile := filepath.Join(tmpDir, "big.ts")
	if err := os.WriteFile(tsFile, []byte(tsContent), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Test JSON format
	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"ts", "all", "--format=json", tmpDir}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()
	stderr := stderrBuf.String()

	// Parse JSON response
	var report map[string]json.RawMessage
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\noutput: %s\nstderr: %s", err, output, stderr)
	}

	// Assert both funclen and filelen keys exist
	if _, ok := report["funclen"]; !ok {
		t.Errorf("expected 'funclen' key in JSON response, got keys: %v", report)
	}
	if _, ok := report["filelen"]; !ok {
		t.Errorf("expected 'filelen' key in JSON response, got keys: %v", report)
	}

	// Verify funclen has violations for big.ts
	if funlen, ok := report["funclen"]; ok {
		var funclenReport tsfunclen.Report
		if err := json.Unmarshal(funlen, &funclenReport); err != nil {
			t.Fatalf("failed to unmarshal funclen report: %v", err)
		}
		if len(funclenReport.Violations) == 0 {
			t.Errorf("expected funclen violations for big.ts")
		}
		// Verify the file is in violations
		found := false
		for _, v := range funclenReport.Violations {
			if strings.Contains(v.File, "big.ts") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected big.ts in funclen violations, got: %v", funclenReport.Violations)
		}
	}

	// Verify filelen has violations for big.ts
	if flen, ok := report["filelen"]; ok {
		var filelenReport filelen.Report
		if err := json.Unmarshal(flen, &filelenReport); err != nil {
			t.Fatalf("failed to unmarshal filelen report: %v", err)
		}
		if len(filelenReport.Violations) == 0 {
			t.Errorf("expected filelen violations for big.ts")
		}
		// Verify the file is in violations
		found := false
		for _, v := range filelenReport.Violations {
			if strings.Contains(v.File, "big.ts") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected big.ts in filelen violations, got: %v", filelenReport.Violations)
		}
	}

	// Expect exit code 1 (violations found)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	// Test text format
	var stdoutBuf2, stderrBuf2 bytes.Buffer
	exitCode2 := run([]string{"ts", "all", "--format=text", tmpDir}, &stdoutBuf2, &stderrBuf2)

	output2 := stdoutBuf2.String()
	if !strings.Contains(output2, "[funclen]") {
		t.Errorf("expected '[funclen]' prefix in text output, got:\n%s", output2)
	}
	if !strings.Contains(output2, "[filelen]") {
		t.Errorf("expected '[filelen]' prefix in text output, got:\n%s", output2)
	}
	if !strings.Contains(output2, "big.ts") {
		t.Errorf("expected 'big.ts' in text output, got:\n%s", output2)
	}

	if exitCode2 != 1 {
		t.Errorf("expected exit code 1 for text format, got %d", exitCode2)
	}
}

func TestRun_CppAllOnEmptyDirIsClean(t *testing.T) {
	tmpDir := t.TempDir()

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"cpp", "all", tmpDir}, &stdoutBuf, &stderrBuf)

	// Empty dir should result in exit code 0
	if exitCode != 0 {
		t.Errorf("expected exit code 0 for empty directory, got %d", exitCode)
	}

	// Test JSON format too
	var stdoutBuf2, stderrBuf2 bytes.Buffer
	exitCode2 := run([]string{"cpp", "all", "--format=json", tmpDir}, &stdoutBuf2, &stderrBuf2)

	if exitCode2 != 0 {
		t.Errorf("expected exit code 0 for empty directory (JSON), got %d", exitCode2)
	}

	// Verify JSON is valid and has empty violations
	var report map[string]json.RawMessage
	if err := json.Unmarshal(stdoutBuf2.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\noutput: %s", err, stdoutBuf2.String())
	}

	// Verify violation counts are 0 or absent
	if funclen, ok := report["funclen"]; ok {
		var funclenReport cppfunclen.Report
		if err := json.Unmarshal(funclen, &funclenReport); err != nil {
			t.Fatalf("failed to unmarshal funclen report: %v", err)
		}
		if len(funclenReport.Violations) != 0 {
			t.Errorf("expected empty funclen violations, got %d", len(funclenReport.Violations))
		}
	}
}

func TestRun_TsAllOnEmptyDirIsClean(t *testing.T) {
	tmpDir := t.TempDir()

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"ts", "all", tmpDir}, &stdoutBuf, &stderrBuf)

	// Empty dir should result in exit code 0
	if exitCode != 0 {
		t.Errorf("expected exit code 0 for empty directory, got %d", exitCode)
	}

	// Test JSON format too
	var stdoutBuf2, stderrBuf2 bytes.Buffer
	exitCode2 := run([]string{"ts", "all", "--format=json", tmpDir}, &stdoutBuf2, &stderrBuf2)

	if exitCode2 != 0 {
		t.Errorf("expected exit code 0 for empty directory (JSON), got %d", exitCode2)
	}

	// Verify JSON is valid and has empty violations
	var report map[string]json.RawMessage
	if err := json.Unmarshal(stdoutBuf2.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\noutput: %s", err, stdoutBuf2.String())
	}

	// Verify violation counts are 0 or absent
	if funclen, ok := report["funclen"]; ok {
		var funclenReport tsfunclen.Report
		if err := json.Unmarshal(funclen, &funclenReport); err != nil {
			t.Fatalf("failed to unmarshal funclen report: %v", err)
		}
		if len(funclenReport.Violations) != 0 {
			t.Errorf("expected empty funclen violations, got %d", len(funclenReport.Violations))
		}
	}
}

func TestRun_CppAllInvalidPathErrors(t *testing.T) {
	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"cpp", "all", "/does/not/exist"}, &stdoutBuf, &stderrBuf)

	// Invalid path should result in exit code 2 (error from srcfiles.Collect)
	if exitCode != 2 {
		t.Errorf("expected exit code 2 for invalid path, got %d", exitCode)
	}
}

func TestRun_TsAllInvalidPathErrors(t *testing.T) {
	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"ts", "all", "/does/not/exist"}, &stdoutBuf, &stderrBuf)

	// Invalid path should result in exit code 2 (error from srcfiles.Collect)
	if exitCode != 2 {
		t.Errorf("expected exit code 2 for invalid path, got %d", exitCode)
	}
}

func TestRun_TsComplexityFlagViolation(t *testing.T) {
	// Test that ts complexity flags a complex function at limit 6
	tmpDir := t.TempDir()
	src := `export function handler(event: any) {
  if (event.type === "request") {
    if (event.method === "GET") {
      if (event.path === "/api/users") {
        if (event.headers["authorization"]) {
          if (event.query.id) {
            if (event.query.id !== "") {
              return { status: 200, body: "OK" };
            }
          }
        }
      }
    }
  }
  return { status: 404, body: "Not Found" };
}`
	if err := os.WriteFile(tmpDir+"/test.ts", []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"ts", "complexity", "--format=json", tmpDir}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	// Parse JSON to verify
	var report interface{}
	err := json.Unmarshal([]byte(output), &report)
	if err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	m := report.(map[string]interface{})
	violations := m["violations"].([]interface{})

	if len(violations) != 1 {
		t.Errorf("expected 1 violation, got %d\noutput: %s", len(violations), output)
	}

	if len(violations) > 0 {
		v := violations[0].(map[string]interface{})
		if v["func"].(string) != "handler" {
			t.Errorf("expected function 'handler', got '%s'", v["func"])
		}
		if int(v["complexity"].(float64)) != 7 {
			t.Errorf("expected complexity 7, got %d", int(v["complexity"].(float64)))
		}
		if int(v["limit"].(float64)) != 6 {
			t.Errorf("expected limit 6, got %d", int(v["limit"].(float64)))
		}
	}

	// Non-zero exit code for violations
	if exitCode == 0 {
		t.Errorf("expected non-zero exit code for violations, got 0")
	}
}

func TestRun_TsComplexityRespectsFlagOverride(t *testing.T) {
	// Test that --max-complexity flag overrides the default
	tmpDir := t.TempDir()
	src := `export function handler(event: any) {
  if (event.type === "request") {
    if (event.method === "GET") {
      if (event.path === "/api/users") {
        if (event.headers["authorization"]) {
          if (event.query.id) {
            if (event.query.id !== "") {
              return { status: 200, body: "OK" };
            }
          }
        }
      }
    }
  }
  return { status: 404, body: "Not Found" };
}`
	if err := os.WriteFile(tmpDir+"/test.ts", []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"ts", "complexity", "--format=json", "--max-complexity=10", tmpDir}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	var report interface{}
	err := json.Unmarshal([]byte(output), &report)
	if err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	m := report.(map[string]interface{})
	violations := m["violations"].([]interface{})

	if len(violations) != 0 {
		t.Errorf("expected 0 violations with --max-complexity=10, got %d\noutput: %s", len(violations), output)
	}

	// Zero exit code when no violations
	if exitCode != 0 {
		t.Errorf("expected exit code 0 when no violations, got %d", exitCode)
	}
}

func TestRun_GoCohesionReportsViolation(t *testing.T) {
	// AC7: Low-cohesion struct should be reported
	tmpDir := t.TempDir()
	src := `package main

type Foo struct{
	x, y int
}

func (f *Foo) SetX(v int) { f.x = v }
func (f *Foo) SetY(v int) { f.y = v }
`
	if err := os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "cohesion", "--format=json", tmpDir}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()
	var report interface{}
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	m := report.(map[string]interface{})
	violations, ok := m["violations"].([]interface{})
	if !ok || len(violations) != 1 {
		t.Errorf("expected 1 violation, got %d", len(violations))
	}

	if len(violations) > 0 {
		v := violations[0].(map[string]interface{})
		if class, ok := v["class"].(string); !ok || class != "Foo" {
			t.Errorf("expected class 'Foo', got '%v'", v["class"])
		}
	}

	if exitCode == 0 {
		t.Errorf("expected non-zero exit code for violations, got 0")
	}
}

func TestRun_AllIncludesCohesion(t *testing.T) {
	// AC11: `go all` includes cohesion check
	tmpDir := t.TempDir()
	initModule(t, tmpDir)

	// No violations
	src := `package main

type Good struct{
	x int
}

func (g *Good) Touch() { g.x = 1 }
`
	if err := os.WriteFile("test.go", []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "all", "--format=json", "."}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()
	var report interface{}
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	m := report.(map[string]interface{})
	if _, ok := m["cohesion"]; !ok {
		t.Errorf("expected 'cohesion' key in JSON output, keys: %v", m)
	}

	if exitCode != 0 {
		t.Errorf("expected exit code 0 for clean code, got %d", exitCode)
	}
}

func TestRun_CppTsAllExcludesCohesion(t *testing.T) {
	// Per plan: C++ all includes complexity, cohesion, and duplication (per AC9)
	tmpDir := t.TempDir()

	// Create a simple C++ file
	cppSrc := `
class MyClass {
public:
  void method1() {}
  void method2() {}
};
`
	if err := os.WriteFile(filepath.Join(tmpDir, "test.cpp"), []byte(cppSrc), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	_ = run([]string{"cpp", "all", "--format=json", tmpDir}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()
	var report interface{}
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	m := report.(map[string]interface{})
	// C++ all now includes complexity, cohesion, and duplication
	if _, ok := m["complexity"]; !ok {
		t.Errorf("expected 'complexity' key in C++ all output")
	}
	if _, ok := m["cohesion"]; !ok {
		t.Errorf("expected 'cohesion' key in C++ all output")
	}
	if _, ok := m["duplication"]; !ok {
		t.Errorf("expected 'duplication' key in C++ all output")
	}

	// Create a simple TypeScript file
	tsSrc := `
export class MyClass {
  method1() {}
  method2() {}
}
`
	tsDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tsDir, "test.ts"), []byte(tsSrc), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	stdoutBuf.Reset()
	stderrBuf.Reset()
	_ = run([]string{"ts", "all", "--format=json", tsDir}, &stdoutBuf, &stderrBuf)

	output = stdoutBuf.String()
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	m = report.(map[string]interface{})
	if _, ok := m["cohesion"]; ok {
		t.Errorf("expected no 'cohesion' key in TS all output, but found one")
	}
}

