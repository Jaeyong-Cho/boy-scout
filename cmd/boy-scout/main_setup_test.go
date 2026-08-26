package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

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
	exitCode := run([]string{"setup", "agents"}, &stdoutBuf, &stderrBuf)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d (stderr: %s)", exitCode, stderrBuf.String())
	}

	stdout := stdoutBuf.String()
	if !strings.Contains(stdout, "boy-scout skill installed:") {
		t.Errorf("expected 'boy-scout skill installed:' in stdout, got: %s", stdout)
	}

	// Verify the skill file exists
	skillPath := filepath.Join(tmpDir, ".agents", "skills", "boy-scout", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatalf("skill file not found at %q: %v", skillPath, err)
	}

	// Verify the binary was copied
	binPath := filepath.Join(tmpDir, ".agents", "bin", "boy-scout")
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("binary not found at %q: %v", binPath, err)
	}
}

func TestRun_SetupGlobalWritesToHomeDir(t *testing.T) {
	// Create a temporary "home" directory
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"setup", "agents", "--global"}, &stdoutBuf, &stderrBuf)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d (stderr: %s)", exitCode, stderrBuf.String())
	}

	// Verify the skill file exists in the home directory
	skillPath := filepath.Join(homeDir, ".agents", "skills", "boy-scout", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatalf("skill file not found at %q: %v", skillPath, err)
	}

	// Verify the binary was copied
	binPath := filepath.Join(homeDir, ".agents", "bin", "boy-scout")
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
	exitCode := run([]string{"setup", "agents", "extra-arg"}, &stdoutBuf, &stderrBuf)

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
	exitCode := run([]string{"setup", "agents"}, &stdoutBuf, &stderrBuf)

	if exitCode != 2 {
		t.Errorf("expected exit code 2 for write failure, got %d", exitCode)
	}

	stderr := stderrBuf.String()
	if !strings.Contains(stderr, "error:") {
		t.Errorf("expected error message in stderr, got: %s", stderr)
	}
}

func TestRun_AllRespectsPerCheckerIgnoreComment(t *testing.T) {
	// Create a fixture with a function long enough for gofunclen (105 lines)
	// and complex enough for crap (5 nested ifs = complexity 5, with no coverage = score = 5^2*(1-0)^3 + 5 = 30, way above 6.0 threshold)
	src := "package main\n\n// boy-scout:ignore:crap\nfunc ViolatingFunc() {\n"
	for i := 0; i < 100; i++ {
		src += fmt.Sprintf("\t_ = %d\n", i)
	}
	src += "\tif true {\n\t\tif true {\n\t\t\tif true {\n\t\t\t\tif true {\n\t\t\t\t\tif true {\n\t\t\t\t\t\t_ = \"x\"\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t}\n}\n"

	tmpDir := t.TempDir()
	writeGoFile(t, tmpDir, "main.go", src)
	initModule(t, tmpDir)

	report, exitCode := runAllJSON(t)

	// funclen should report ViolatingFunc (directive doesn't name gofunclen)
	if !hasViolation(report.Gofunclen.Violations, "ViolatingFunc") {
		t.Errorf("expected ViolatingFunc in funclen violations, got %v", report.Gofunclen.Violations)
	}

	// crap should NOT report ViolatingFunc (directive names crap, so it's excluded)
	if hasViolation(report.Crap.Violations, "ViolatingFunc") {
		t.Errorf("expected ViolatingFunc NOT in crap violations, but found it")
	}

	// Exit code should be 1 (violation from gofunclen)
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
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

	// Run "all" with JSON format (uses default thresholds: gofunclen=50, crap=6.0)
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
	code := "void longFunction() {\n" + cppFuncLenFillerLines(51) + "}"

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

func TestPromptForTarget_NumericSelection(t *testing.T) {
	stdin := strings.NewReader("2\n")
	var stdout bytes.Buffer

	name, err := promptForTarget(stdin, &stdout)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if name != "copilot" {
		t.Errorf("expected 'copilot', got %q", name)
	}

	output := stdout.String()
	if !strings.Contains(output, "1) claude") {
		t.Errorf("expected '1) claude' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "2) copilot") {
		t.Errorf("expected '2) copilot' in output, got:\n%s", output)
	}
}

func TestPromptForTarget_NameSelection(t *testing.T) {
	stdin := strings.NewReader("pi\n")
	var stdout bytes.Buffer

	name, err := promptForTarget(stdin, &stdout)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if name != "pi" {
		t.Errorf("expected 'pi', got %q", name)
	}
}

func TestPromptForTarget_InvalidSelectionReturnsError(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		wantErr  bool
	}{
		{"invalid index", "99\n", true},
		{"invalid name", "nope\n", true},
		{"empty input", "\n", true},
		{"EOF", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stdin := strings.NewReader(tc.input)
			var stdout bytes.Buffer

			name, err := promptForTarget(stdin, &stdout)
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got none (returned name=%q)", name)
			}
		})
	}
}

func TestResolveTarget(t *testing.T) {
	testCases := []struct {
		name        string
		positional  []string
		stdin       string
		interactive bool
		wantName    string
		wantPrefix  string
		wantErr     bool
	}{
		{"explicit target found", []string{"copilot"}, "", false, "copilot", ".copilot", false},
		{"explicit target unknown", []string{"bogus"}, "", false, "", "", true},
		{"interactive prompt success", nil, "pi\n", true, "pi", ".pi/agent", false},
		{"interactive prompt error", nil, "\n", true, "", "", true},
		{"non-interactive no target", nil, "", false, "", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			name, prefix, err := resolveTarget(tc.positional, strings.NewReader(tc.stdin), tc.interactive, &stdout)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got none (name=%q prefix=%q)", name, prefix)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if name != tc.wantName {
				t.Errorf("expected name %q, got %q", tc.wantName, name)
			}
			if prefix != tc.wantPrefix {
				t.Errorf("expected prefix %q, got %q", tc.wantPrefix, prefix)
			}
		})
	}
}

func TestRun_SetupClaudeWritesClaudeSkillPath(t *testing.T) {
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
	exitCode := run([]string{"setup", "claude"}, &stdoutBuf, &stderrBuf)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d (stderr: %s)", exitCode, stderrBuf.String())
	}

	skillPath := filepath.Join(tmpDir, ".claude", "skills", "boy-scout", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatalf("skill file not found at %q: %v", skillPath, err)
	}

	binPath := filepath.Join(tmpDir, ".claude", "bin", "boy-scout")
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("binary not found at %q: %v", binPath, err)
	}
}

func TestRun_SetupCopilotWritesCopilotSkillPath(t *testing.T) {
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
	exitCode := run([]string{"setup", "copilot"}, &stdoutBuf, &stderrBuf)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d (stderr: %s)", exitCode, stderrBuf.String())
	}

	skillPath := filepath.Join(tmpDir, ".copilot", "skills", "boy-scout", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatalf("skill file not found at %q: %v", skillPath, err)
	}

	binPath := filepath.Join(tmpDir, ".copilot", "bin", "boy-scout")
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("binary not found at %q: %v", binPath, err)
	}
}

func TestRun_SetupPiWritesPiAgentSkillPath(t *testing.T) {
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
	exitCode := run([]string{"setup", "pi"}, &stdoutBuf, &stderrBuf)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d (stderr: %s)", exitCode, stderrBuf.String())
	}

	skillPath := filepath.Join(tmpDir, ".pi", "agent", "skills", "boy-scout", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatalf("skill file not found at %q: %v", skillPath, err)
	}

	binPath := filepath.Join(tmpDir, ".pi", "agent", "bin", "boy-scout")
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("binary not found at %q: %v", binPath, err)
	}
}

func TestRun_SetupUnknownTargetIsUsageError(t *testing.T) {
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
	exitCode := run([]string{"setup", "bogus"}, &stdoutBuf, &stderrBuf)

	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}

	stderr := stderrBuf.String()
	if !strings.Contains(stderr, "error:") || !strings.Contains(stderr, "unknown target") {
		t.Errorf("expected 'unknown target' error in stderr, got: %s", stderr)
	}

	// Verify no directory was created
	for _, prefix := range []string{".agents", ".claude", ".copilot", ".pi", ".bogus"} {
		dirPath := filepath.Join(tmpDir, prefix)
		if _, err := os.Stat(dirPath); err == nil {
			t.Errorf("expected %s directory not to be created for unknown target", prefix)
		}
	}
}

func TestRun_SetupNoTargetNonInteractiveIsUsageError(t *testing.T) {
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

	// Call runSetup directly with a non-interactive stdin (strings.Reader, not os.Stdin)
	stdin := strings.NewReader("")
	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := runSetup([]string{}, stdin, &stdoutBuf, &stderrBuf)

	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}

	stderr := stderrBuf.String()
	// Should list valid targets
	if !strings.Contains(stderr, "claude") || !strings.Contains(stderr, "copilot") {
		t.Errorf("expected valid targets listed in stderr, got: %s", stderr)
	}

	// Verify no files were written
	for _, prefix := range []string{".agents", ".claude", ".copilot", ".pi"} {
		dirPath := filepath.Join(tmpDir, prefix)
		if _, err := os.Stat(dirPath); err == nil {
			t.Errorf("expected %s directory not to be created for non-interactive no-target", prefix)
		}
	}
}

func TestParseSetupArgs(t *testing.T) {
	testCases := []struct {
		name           string
		args           []string
		wantGlobal     bool
		wantPositional []string
		wantErr        bool
	}{
		{"no args", nil, false, nil, false},
		{"global flag", []string{"--global"}, true, nil, false},
		{"single-dash global flag", []string{"-global"}, true, nil, false},
		{"target and global", []string{"claude", "--global"}, true, []string{"claude"}, false},
		{"unknown flag", []string{"--bogus"}, false, nil, true},
		{"too many positional args", []string{"claude", "copilot"}, false, nil, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var stderrBuf bytes.Buffer
			global, positional, err := parseSetupArgs(tc.args, &stderrBuf)

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if global != tc.wantGlobal {
				t.Errorf("expected global=%v, got %v", tc.wantGlobal, global)
			}
			if !reflect.DeepEqual(positional, tc.wantPositional) {
				t.Errorf("expected positional=%v, got %v", tc.wantPositional, positional)
			}
		})
	}
}

func TestRun_SetupGlobalFlagComposesInEitherOrder(t *testing.T) {
	testCases := []struct {
		name string
		args []string
	}{
		{"target then flag", []string{"setup", "claude", "--global"}},
		{"flag then target", []string{"setup", "--global", "claude"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			homeDir := t.TempDir()
			t.Setenv("HOME", homeDir)

			var stdoutBuf, stderrBuf bytes.Buffer
			exitCode := run(tc.args, &stdoutBuf, &stderrBuf)

			if exitCode != 0 {
				t.Errorf("expected exit code 0, got %d (stderr: %s)", exitCode, stderrBuf.String())
			}

			skillPath := filepath.Join(homeDir, ".claude", "skills", "boy-scout", "SKILL.md")
			if _, err := os.Stat(skillPath); err != nil {
				t.Fatalf("skill file not found at %q: %v", skillPath, err)
			}
		})
	}
}

func TestRun_FilelenRespectsMaxLinesFlag(t *testing.T) {
	// Create a file with 60 lines
	lines := []string{"package main"}
	for i := 0; i < 59; i++ {
		lines = append(lines, fmt.Sprintf("var x%d = %d", i, i))
	}
	src := strings.Join(lines, "\n")

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "toolong.go")
	if err := os.WriteFile(tmpFile, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "filelen", "--max-lines=50", tmpFile}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	// Should report violation with limit=50, not 300
	if !strings.Contains(output, "60 lines (limit 50)") {
		t.Errorf("expected output to contain '60 lines (limit 50)', got:\n%s", output)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

func TestRun_FilelenCountsLoCNotPhysicalLines(t *testing.T) {
	// Create a file with 320 physical lines but only 41 LoC
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "bigcomments.go")

	content := "package main\n"
	for i := 0; i < 40; i++ {
		content += fmt.Sprintf("var x%d = %d\n", i, i)
	}
	// Add 279 blank lines to reach 320 total physical lines
	for i := 0; i < 279; i++ {
		content += "\n"
	}

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "filelen", tmpFile}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	// Should not report a violation since LoC (41) is well under limit (300)
	if strings.Contains(output, filepath.Base(tmpFile)) {
		t.Errorf("expected no output for file with 320 physical / 41 LoC, got:\n%s", output)
	}

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestRun_FilelenTextFormatMatchesExpectedLine(t *testing.T) {
	// Create a file with 350 lines (over default limit of 300)
	lines := []string{"package main"}
	for i := 0; i < 349; i++ {
		lines = append(lines, fmt.Sprintf("var x%d = %d", i, i))
	}
	src := strings.Join(lines, "\n")

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "big.go")
	if err := os.WriteFile(tmpFile, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "filelen", tmpFile}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	// Should contain the file path followed by line count and limit
	if !strings.Contains(output, tmpFile+": 350 lines (limit 300)") {
		t.Errorf("expected output to contain '%s: 350 lines (limit 300)', got:\n%s", tmpFile, output)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1 (has violations), got %d", exitCode)
	}
}

func TestRun_FilelenJSONFormatOutputsValidSchema(t *testing.T) {
	// Create a file with 350 lines
	lines := []string{"package main"}
	for i := 0; i < 349; i++ {
		lines = append(lines, fmt.Sprintf("var x%d = %d", i, i))
	}
	src := strings.Join(lines, "\n")

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "big.go")
	if err := os.WriteFile(tmpFile, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "filelen", "--format=json", tmpFile}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	var report map[string]interface{}
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, output)
	}

	// Check that the JSON has the expected structure
	if _, ok := report["Violations"]; !ok {
		t.Errorf("expected 'Violations' key in JSON output, got keys: %v", reflect.ValueOf(report).MapKeys())
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

func TestRun_CppFilelenOnlyScansCppExtensions(t *testing.T) {
	// Create both a .go and a .cpp file
	tmpDir := t.TempDir()

	// Create a .go file with 350 lines (should be ignored for cpp filelen)
	goLines := []string{"package main"}
	for i := 0; i < 349; i++ {
		goLines = append(goLines, fmt.Sprintf("var x%d = %d", i, i))
	}
	goFile := filepath.Join(tmpDir, "big.go")
	if err := os.WriteFile(goFile, []byte(strings.Join(goLines, "\n")), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Create a .cpp file with 350 lines (should be reported for cpp filelen)
	cppLines := []string{"#include <iostream>"}
	for i := 0; i < 349; i++ {
		cppLines = append(cppLines, fmt.Sprintf("int x%d = %d;", i, i))
	}
	cppFile := filepath.Join(tmpDir, "big.cpp")
	if err := os.WriteFile(cppFile, []byte(strings.Join(cppLines, "\n")), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"cpp", "filelen", tmpDir}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	// Should report the .cpp file, not the .go file
	if !strings.Contains(output, "big.cpp: 350 lines (limit 300)") {
		t.Errorf("expected output to contain 'big.cpp: 350 lines (limit 300)', got:\n%s", output)
	}
	if strings.Contains(output, "big.go") {
		t.Errorf("expected output to NOT contain 'big.go' (cpp filelen should ignore .go files), got:\n%s", output)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

func TestRun_AllIncludesFilelen(t *testing.T) {
	// Create a file that violates both gofunclen and filelen
	tmpDir := t.TempDir()

	// Create a file with a 60-line function and 350 total lines
	lines := []string{"package main", "", "func Big() {"}
	for i := 0; i < 347; i++ {
		lines = append(lines, "\t_ = "+fmt.Sprintf("%d", i))
	}
	lines = append(lines, "\t_ = 0") // use a blank identifier to avoid unused variable error
	lines = append(lines, "}")
	src := strings.Join(lines, "\n")

	srcFile := filepath.Join(tmpDir, "big.go")
	if err := os.WriteFile(srcFile, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Create a go.mod file
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "all", "."}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	// Should include [filelen] output alongside [gofunclen] and [crap]
	if !strings.Contains(output, "[filelen]") {
		t.Errorf("expected output to contain '[filelen]', got:\n%s", output)
	}
	if !strings.Contains(output, "[gofunclen]") {
		t.Errorf("expected output to contain '[gofunclen]', got:\n%s", output)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1 (has violations), got %d (stderr: %s)", exitCode, stderrBuf.String())
	}
}

func TestRun_AllIncludesFilelenJSON(t *testing.T) {
	// Create a file with 350 lines
	tmpDir := t.TempDir()

	lines := []string{"package main"}
	for i := 0; i < 349; i++ {
		lines = append(lines, fmt.Sprintf("var x%d = %d", i, i))
	}
	src := strings.Join(lines, "\n")

	srcFile := filepath.Join(tmpDir, "big.go")
	if err := os.WriteFile(srcFile, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Create a go.mod file
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	var stdoutBuf, stderrBuf bytes.Buffer
	exitCode := run([]string{"go", "all", "--format=json", "."}, &stdoutBuf, &stderrBuf)

	output := stdoutBuf.String()

	var report map[string]interface{}
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, output)
	}

	// Check that the JSON has filelen key
	if _, ok := report["filelen"]; !ok {
		t.Errorf("expected 'filelen' key in JSON output, got keys: %v", reflect.ValueOf(report).MapKeys())
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1 (has violations), got %d (stderr: %s)", exitCode, stderrBuf.String())
	}
}

func TestRun_FilelenExitCodeReflectsOutcome(t *testing.T) {
	testCases := []struct {
		name         string
		content      string
		maxLines     int
		expectedCode int
	}{
		{
			name:         "clean file",
			content:      "package main\n// small",
			maxLines:     300,
			expectedCode: 0,
		},
		{
			name:         "file with violations",
			content:      "package main\n" + strings.Repeat("var x = 1\n", 301),
			maxLines:     300,
			expectedCode: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			srcFile := filepath.Join(tmpDir, "test.go")
			if err := os.WriteFile(srcFile, []byte(tc.content), 0644); err != nil {
				t.Fatalf("WriteFile failed: %v", err)
			}

			var stdoutBuf, stderrBuf bytes.Buffer
			exitCode := run([]string{"go", "filelen", "--max-lines=" + fmt.Sprintf("%d", tc.maxLines), srcFile}, &stdoutBuf, &stderrBuf)

			if exitCode != tc.expectedCode {
				t.Errorf("expected exit code %d, got %d (output: %s)", tc.expectedCode, exitCode, stdoutBuf.String())
			}
		})
	}
}

func TestRun_SetupWritesReferenceFiles(t *testing.T) {
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
	exitCode := run([]string{"setup", "agents"}, &stdoutBuf, &stderrBuf)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d (stderr: %s)", exitCode, stderrBuf.String())
	}

	// Verify all 5 reference files exist
	refDir := filepath.Join(tmpDir, ".agents", "skills", "boy-scout", "references")
	referenceFiles := []string{"funclen.md", "complexity.md", "filelen.md", "instability.md", "abstractness.md"}

	for _, filename := range referenceFiles {
		path := filepath.Join(refDir, filename)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected reference file %q to exist, got error: %v", filename, err)
		}
	}

	// Verify funclen.md contains expected marker
	funclenPath := filepath.Join(refDir, "funclen.md")
	funclenContent, err := os.ReadFile(funclenPath)
	if err != nil {
		t.Fatalf("failed to read funclen.md: %v", err)
	}
	if !strings.Contains(string(funclenContent), "one level of abstraction") {
		t.Errorf("expected funclen.md to contain 'one level of abstraction', got:\n%s", funclenContent)
	}

	// Verify abstractness.md contains expected marker
	abstractnessPath := filepath.Join(refDir, "abstractness.md")
	abstractnessContent, err := os.ReadFile(abstractnessPath)
	if err != nil {
		t.Fatalf("failed to read abstractness.md: %v", err)
	}
	if !strings.Contains(string(abstractnessContent), "Zone of Pain") {
		t.Errorf("expected abstractness.md to contain 'Zone of Pain', got:\n%s", abstractnessContent)
	}
}

func TestRun_SetupWritesCapAndPriorityGuidance(t *testing.T) {
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
	exitCode := run([]string{"setup", "agents"}, &stdoutBuf, &stderrBuf)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d (stderr: %s)", exitCode, stderrBuf.String())
	}

	// Read the skill file
	skillPath := filepath.Join(tmpDir, ".agents", "skills", "boy-scout", "SKILL.md")
	skillContent, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}
	skillStr := string(skillContent)

	// Verify all 4 markers from the cap-and-priority guidance are present
	markers := []string{
		"Within each type, select one candidate per run",
		"boy-scout's severity",
		"go last",
		"never edit anything in this step",
	}

	for _, marker := range markers {
		if !strings.Contains(skillStr, marker) {
			t.Errorf("expected skill file to contain %q, got:\n%s", marker, skillStr)
		}
	}
}

func TestRun_SetupWritesCppFilelenInstabilityAbstractnessReferences(t *testing.T) {
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
	exitCode := run([]string{"setup", "agents"}, &stdoutBuf, &stderrBuf)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d (stderr: %s)", exitCode, stderrBuf.String())
	}

	refDir := filepath.Join(tmpDir, ".agents", "skills", "boy-scout", "references", "lang", "cpp")
	for _, filename := range []string{"filelen.md", "instability.md", "abstractness.md"} {
		path := filepath.Join(refDir, filename)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected %q to exist, got error: %v", filename, err)
		}
	}
}

func TestRun_SetupWritesDuplicationReferenceFiles(t *testing.T) {
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
	exitCode := run([]string{"setup", "agents"}, &stdoutBuf, &stderrBuf)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d (stderr: %s)", exitCode, stderrBuf.String())
	}

	// Verify top-level duplication reference file exists and is non-empty
	duplicationPath := filepath.Join(tmpDir, ".agents", "skills", "boy-scout", "references", "duplication.md")
	if _, err := os.Stat(duplicationPath); err != nil {
		t.Errorf("expected duplication.md to exist at %q: %v", duplicationPath, err)
	}
	content, err := os.ReadFile(duplicationPath)
	if err != nil {
		t.Fatalf("failed to read duplication.md: %v", err)
	}
	if len(content) == 0 {
		t.Errorf("expected duplication.md to be non-empty")
	}

	// Verify Go duplication reference file exists and is non-empty
	goPath := filepath.Join(tmpDir, ".agents", "skills", "boy-scout", "references", "lang", "go", "duplication.md")
	if _, err := os.Stat(goPath); err != nil {
		t.Errorf("expected Go duplication.md to exist at %q: %v", goPath, err)
	}
	content, err = os.ReadFile(goPath)
	if err != nil {
		t.Fatalf("failed to read Go duplication.md: %v", err)
	}
	if len(content) == 0 {
		t.Errorf("expected Go duplication.md to be non-empty")
	}
}
