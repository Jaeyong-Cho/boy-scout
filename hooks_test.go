package hooks_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCommitMsgHook_AcceptsConventionalFormat verifies that a valid
// Conventional Commits message passes the commit-msg hook.
func TestCommitMsgHook_AcceptsConventionalFormat(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "commit-msg-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("fix: exclude test files from dependency graph\n"); err != nil {
		t.Fatalf("failed to write message: %v", err)
	}
	tmpFile.Close()

	cmd := exec.Command("bash", ".githooks/commit-msg", tmpFile.Name())
	cmd.Dir, _ = os.Getwd()
	if err := cmd.Run(); err != nil {
		t.Errorf("commit-msg hook rejected valid format: %v", err)
	}
}

// TestCommitMsgHook_RejectsNonConventionalFormat verifies that a non-Conventional
// Commits message fails the hook with a helpful error.
func TestCommitMsgHook_RejectsNonConventionalFormat(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "commit-msg-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("fixed the bug\n"); err != nil {
		t.Fatalf("failed to write message: %v", err)
	}
	tmpFile.Close()

	cmd := exec.Command("bash", ".githooks/commit-msg", tmpFile.Name())
	cmd.Dir, _ = os.Getwd()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err == nil {
		t.Error("commit-msg hook accepted invalid format")
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Conventional Commits")) {
		t.Errorf("stderr missing 'Conventional Commits': %s", stderr.String())
	}
}

// TestCommitMsgHook_ExemptsMergeCommits verifies that Merge and Revert
// commits are exempt from the format check.
func TestCommitMsgHook_ExemptsMergeCommits(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "commit-msg-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("Merge branch 'feat/x' into main\n"); err != nil {
		t.Fatalf("failed to write message: %v", err)
	}
	tmpFile.Close()

	cmd := exec.Command("bash", ".githooks/commit-msg", tmpFile.Name())
	cmd.Dir, _ = os.Getwd()
	if err := cmd.Run(); err != nil {
		t.Errorf("commit-msg hook rejected exempt merge commit: %v", err)
	}
}

// TestPreCommitHook_PassesWhenCheckSucceeds verifies that the pre-commit
// hook exits 0 when `make check` succeeds.
func TestPreCommitHook_PassesWhenCheckSucceeds(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "pre-commit-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a stub Makefile that always succeeds
	makefile := filepath.Join(tmpDir, "Makefile")
	if err := ioutil.WriteFile(makefile, []byte("check:\n\texit 0\n"), 0644); err != nil {
		t.Fatalf("failed to write stub Makefile: %v", err)
	}

	repoRoot, _ := os.Getwd()
	hookPath := filepath.Join(repoRoot, ".githooks/pre-commit")
	cmd := exec.Command("bash", hookPath)
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Errorf("pre-commit hook failed when check succeeded: %v", err)
	}
}

// TestPrePushHook_FailsWhenCheckFails verifies that the pre-push hook
// exits non-zero when `make check` fails.
func TestPrePushHook_FailsWhenCheckFails(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "pre-push-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a stub Makefile that always fails
	makefile := filepath.Join(tmpDir, "Makefile")
	if err := ioutil.WriteFile(makefile, []byte("check:\n\texit 1\n"), 0644); err != nil {
		t.Fatalf("failed to write stub Makefile: %v", err)
	}

	repoRoot, _ := os.Getwd()
	hookPath := filepath.Join(repoRoot, ".githooks/pre-push")
	cmd := exec.Command("bash", hookPath)
	cmd.Dir = tmpDir
	if err := cmd.Run(); err == nil {
		t.Error("pre-push hook succeeded when check failed")
	}
}

// TestInstallHooks_SetsHooksPath verifies that `make install-hooks`
// sets `git config core.hooksPath` to `.githooks`.
func TestInstallHooks_SetsHooksPath(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "install-hooks-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize a git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git init: %v", err)
	}

	// Copy .githooks to temp dir
	srcHooks := ".githooks"
	dstHooks := filepath.Join(tmpDir, ".githooks")
	if err := os.MkdirAll(dstHooks, 0755); err != nil {
		t.Fatalf("failed to create .githooks in temp dir: %v", err)
	}

	// Copy hook files from repo root
	for _, hookName := range []string{"commit-msg", "pre-commit", "pre-push"} {
		src := filepath.Join(srcHooks, hookName)
		dst := filepath.Join(dstHooks, hookName)
		data, err := ioutil.ReadFile(src)
		if err != nil {
			t.Logf("hook file %s not yet created (expected in RED phase): %v", hookName, err)
			continue
		}
		if err := ioutil.WriteFile(dst, data, 0755); err != nil {
			t.Fatalf("failed to copy hook %s: %v", hookName, err)
		}
	}

	// Copy Makefile to temp dir
	makefileSrc, _ := os.Getwd()
	makefilePath := filepath.Join(makefileSrc, "Makefile")
	makefile, err := ioutil.ReadFile(makefilePath)
	if err != nil {
		t.Fatalf("failed to read Makefile: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, "Makefile"), makefile, 0644); err != nil {
		t.Fatalf("failed to copy Makefile: %v", err)
	}

	// Run `make install-hooks` in the temp dir
	cmd = exec.Command("make", "install-hooks")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("make install-hooks failed: %v", err)
	}

	// Check that `git config core.hooksPath` is `.githooks`
	cmd = exec.Command("git", "config", "core.hooksPath")
	cmd.Dir = tmpDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Errorf("git config core.hooksPath failed: %v", err)
	}

	output := bytes.TrimSpace(stdout.Bytes())
	if string(output) != ".githooks" {
		t.Errorf("expected core.hooksPath to be '.githooks', got '%s'", string(output))
	}
}
