package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

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
	exitCode := run([]string{"funclen", tmpFile}, &stdoutBuf, &stderrBuf)

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
	exitCode := run([]string{"funclen", "--max-lines=50", tmpFile}, &stdoutBuf, &stderrBuf)

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
	exitCode := run([]string{"funclen"}, &stdoutBuf, &stderrBuf)

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
	exitCode := run([]string{"funclen", "--format=json", tmpFile}, &stdoutBuf, &stderrBuf)

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
				lines := []string{"package main", "", "func Big() {"}
				for i := 0; i < 103; i++ {
					lines = append(lines, "\tx := 1")
				}
				lines = append(lines, "}")
				src := strings.Join(lines, "\n")
				if err := os.WriteFile(tmpDir+"/big.go", []byte(src), 0644); err != nil {
					t.Fatalf("WriteFile failed: %v", err)
				}
			}

			// Create a skipped file if needed
			if tc.hasSkipped {
				if err := os.WriteFile(tmpDir+"/bad.go", []byte("invalid syntax {"), 0644); err != nil {
					t.Fatalf("WriteFile failed: %v", err)
				}
			}

			// If no violations and no skipped, create a clean file
			if !tc.hasViolations && !tc.hasSkipped {
				src := `package main

func Small() {
	x := 1
}
`
				if err := os.WriteFile(tmpDir+"/small.go", []byte(src), 0644); err != nil {
					t.Fatalf("WriteFile failed: %v", err)
				}
			}

			var stdoutBuf, stderrBuf bytes.Buffer
			exitCode := run([]string{"funclen", tmpDir}, &stdoutBuf, &stderrBuf)

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
	if !strings.Contains(stderr, "funclen") || !strings.Contains(stderr, "all") {
		t.Errorf("expected usage message with 'funclen' and 'all', got:\n%s", stderr)
	}

	if exitCode == 0 {
		t.Errorf("expected non-zero exit code, got 0")
	}
}

func TestRun_AllRunsEveryRegisteredCheck(t *testing.T) {
	// Create a temp dir with a violating file
	tmpDir := t.TempDir()

	lines := []string{"package main", "", "func ViolatingFunc() {"}
	for i := 0; i < 103; i++ {
		lines = append(lines, "\tx := 1")
	}
	lines = append(lines, "}")
	src := strings.Join(lines, "\n")

	if err := os.WriteFile(tmpDir+"/violating.go", []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"all", tmpDir}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()
	if !strings.Contains(output, "ViolatingFunc is 105 lines (limit 50)") {
		t.Errorf("expected funclen violation in 'all' output, got:\n%s", output)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}
