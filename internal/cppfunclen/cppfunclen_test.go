package cppfunclen

import (
	"os"
	"path/filepath"
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

func TestCheck_ReportsViolationForOverLimitFunction(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a C++ file with a function over 50 lines
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
  int u = 23;
  int v = 24;
  int w = 25;
  int x1 = 26;
  int x2 = 27;
  int x3 = 28;
  int x4 = 29;
  int x5 = 30;
  int x6 = 31;
  int x7 = 32;
  int x8 = 33;
  int x9 = 34;
  int x10 = 35;
  int x11 = 36;
  int x12 = 37;
  int x13 = 38;
  int x14 = 39;
  int x15 = 40;
  int x16 = 41;
  int x17 = 42;
  int x18 = 43;
  int x19 = 44;
  int x20 = 45;
  int x21 = 46;
  int x22 = 47;
  int x23 = 48;
  int x24 = 49;
  int x25 = 50;
  int x26 = 51;
}`

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

  auto lambda = [](int val) {
    int result = val * 2;
    return result;
  };

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
	code := `void fooTest() {
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
