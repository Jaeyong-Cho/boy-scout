package cppfunclen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFile writes content to path, creating any parent directories needed.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

// fillerLines returns n valid, uniquely-named C++ declaration lines starting
// at index start, used to pad a function body past the funclen limit.
func fillerLines(start, n int) string {
	var b strings.Builder
	for i := start; i < start+n; i++ {
		fmt.Fprintf(&b, "  int v%d = %d;\n", i, i)
	}
	return b.String()
}

func TestCheck_ReportsViolationForOverLimitFunction(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a C++ file with a function over 50 lines
	code := "void longFunction() {\n" + fillerLines(0, 51) + "}"

	cppFile := filepath.Join(tmpDir, "test.cpp")
	writeFile(t, cppFile, code)

	// Call Check with maxLines=50
	report, err := Check([]string{tmpDir}, 50, Options{})

	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(report.Violations), report.Violations)
	}

	if len(report.Violations) == 1 {
		v := report.Violations[0]
		if v.Func != "longFunction" {
			t.Errorf("expected func name 'longFunction', got %q", v.Func)
		}
		if v.Limit != 50 {
			t.Errorf("expected limit 50, got %d", v.Limit)
		}
		if v.Length <= 50 {
			t.Errorf("expected length > 50, got %d", v.Length)
		}
	}
}

func TestCheck_SkipsBodylessDeclarations(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a header file with only declarations (no body)
	code := `void foo();
int bar();
class Widget;
`

	hppFile := filepath.Join(tmpDir, "declarations.hpp")
	writeFile(t, hppFile, code)

	report, err := Check([]string{tmpDir}, 50, Options{})

	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations for declarations, got %d: %v", len(report.Violations), report.Violations)
	}
}

func TestCheck_ZeroMatchingFilesReturnsEmptyReport(t *testing.T) {
	tmpDir := t.TempDir()

	report, err := Check([]string{tmpDir}, 50, Options{})

	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(report.Violations))
	}
	if len(report.Skipped) != 0 {
		t.Errorf("expected 0 skipped, got %d", len(report.Skipped))
	}
	if len(report.ExcludedFuncs) != 0 {
		t.Errorf("expected 0 excluded funcs, got %d", len(report.ExcludedFuncs))
	}
}

func TestCheck_QualifiesOutOfLineMethodName(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a C++ file with an out-of-line method definition
	code := `class Widget {
public:
  void resize();
};

void Widget::resize() {
` + fillerLines(0, 51) + `}`

	cppFile := filepath.Join(tmpDir, "widget.cpp")
	writeFile(t, cppFile, code)

	report, err := Check([]string{tmpDir}, 50, Options{})

	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Errorf("expected 1 violation, got %d", len(report.Violations))
	}

	if len(report.Violations) == 1 {
		v := report.Violations[0]
		if v.Func != "Widget::resize" {
			t.Errorf("expected func name 'Widget::resize', got %q", v.Func)
		}
	}
}

func TestCheck_AttributesLambdaLinesToEnclosingFunction(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a C++ file with a lambda nested in a function
	code := `void outerFunction() {
` + fillerLines(0, 31) + `
  auto lambda = [](int val) {
    int result = val * 2;
    return result;
  };

` + fillerLines(31, 20) + `}`

	cppFile := filepath.Join(tmpDir, "lambda.cpp")
	writeFile(t, cppFile, code)

	report, err := Check([]string{tmpDir}, 50, Options{})

	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Should have exactly one violation for the outer function
	if len(report.Violations) != 1 {
		t.Errorf("expected 1 violation (for outer function), got %d: %v", len(report.Violations), report.Violations)
	}

	if len(report.Violations) == 1 {
		v := report.Violations[0]
		if v.Func != "outerFunction" {
			t.Errorf("expected violation for 'outerFunction', got %q", v.Func)
		}
	}
}

func TestCheck_ExcludeFuncFlagFiltersByGlobName(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a C++ file with a function matching the exclude pattern
	code := "void fooTest() {\n" + fillerLines(0, 51) + "}"

	cppFile := filepath.Join(tmpDir, "test.cpp")
	writeFile(t, cppFile, code)

	// Call Check with exclude pattern
	report, err := Check([]string{tmpDir}, 50, Options{ExcludeFuncs: []string{"*Test"}})

	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations (fooTest should be excluded), got %d: %v", len(report.Violations), report.Violations)
	}

	// With Debug, should appear in ExcludedFuncs
	report, err = Check([]string{tmpDir}, 50, Options{ExcludeFuncs: []string{"*Test"}, Debug: true})

	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.ExcludedFuncs) != 1 {
		t.Errorf("expected 1 excluded func in debug mode, got %d: %v", len(report.ExcludedFuncs), report.ExcludedFuncs)
	}
}

func TestCheck_SyntaxErrorFileIsSkippedEntirely(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a C++ file with a syntax error (unclosed comment)
	code := `void brokenFunction() {
  int x = 1;
  int y = 2;
  /* unclosed comment`

	cppFile := filepath.Join(tmpDir, "broken.cpp")
	writeFile(t, cppFile, code)

	report, err := Check([]string{tmpDir}, 50, Options{})

	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Skipped) != 1 {
		t.Errorf("expected 1 skipped file, got %d: %v", len(report.Skipped), report.Skipped)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations (file has syntax error), got %d: %v", len(report.Violations), report.Violations)
	}
}
