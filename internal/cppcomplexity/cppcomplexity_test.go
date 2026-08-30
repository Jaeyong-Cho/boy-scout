package cppcomplexity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTmpCppFile(t *testing.T, dir, filename, content string) string {
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", filename, err)
	}
	return path
}

func TestComplexity7FlaggedAtDefault6(t *testing.T) {
	tmpDir := t.TempDir()
	// Complexity 7: base 1 + 6 if statements
	src := `void func() {
  if (true) {
    if (true) {
      if (true) {
        if (true) {
          if (true) {
            if (true) {}
          }
        }
      }
    }
  }
}`
	writeTmpCppFile(t, tmpDir, "test.cpp", src)

	report, err := Check([]string{tmpDir}, 6, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Errorf("expected 1 violation, got %d", len(report.Violations))
	}
	if len(report.Violations) > 0 {
		v := report.Violations[0]
		if v.Complexity != 7 {
			t.Errorf("expected complexity 7, got %d", v.Complexity)
		}
		if v.Limit != 6 {
			t.Errorf("expected limit 6, got %d", v.Limit)
		}
	}
}

func TestBoundaryComplexity6AtLimit6(t *testing.T) {
	tmpDir := t.TempDir()
	// Complexity 6: base 1 + 5 if statements
	src := `void func() {
  if (true) {
    if (true) {
      if (true) {
        if (true) {
          if (true) {}
        }
      }
    }
  }
}`
	writeTmpCppFile(t, tmpDir, "test.cpp", src)

	report, err := Check([]string{tmpDir}, 6, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations at limit, got %d: %v", len(report.Violations), report.Violations)
	}
}

func TestLogicalOperators(t *testing.T) {
	tmpDir := t.TempDir()
	// Complexity 4: base 1 + if (1) + && (1) + || (1)
	src := `void func() {
  if (a && b || c) {}
}`
	writeTmpCppFile(t, tmpDir, "test.cpp", src)

	report, err := Check([]string{tmpDir}, 10, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations with limit 10, got %d", len(report.Violations))
	}

	// Verify we counted the operators correctly by lowering limit
	report2, _ := Check([]string{tmpDir}, 3, Options{})
	if len(report2.Violations) != 1 {
		t.Errorf("expected 1 violation at limit 3, got %d", len(report2.Violations))
	}
	if len(report2.Violations) > 0 && report2.Violations[0].Complexity != 4 {
		t.Errorf("expected complexity 4, got %d", report2.Violations[0].Complexity)
	}
}

func TestSwitchCaseVsDefault(t *testing.T) {
	tmpDir := t.TempDir()
	// Complexity 4: base 1 + 3 cases (not counting default)
	src := `void func(int x) {
  switch (x) {
    case 1: break;
    case 2: break;
    case 3: break;
    default: break;
  }
}`
	writeTmpCppFile(t, tmpDir, "test.cpp", src)

	report, err := Check([]string{tmpDir}, 10, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations with limit 10, got %d", len(report.Violations))
	}

	// Verify we counted 4 with limit 3
	report2, _ := Check([]string{tmpDir}, 3, Options{})
	if len(report2.Violations) != 1 {
		t.Errorf("expected 1 violation at limit 3, got %d", len(report2.Violations))
	}
	if len(report2.Violations) > 0 && report2.Violations[0].Complexity != 4 {
		t.Errorf("expected complexity 4, got %d", report2.Violations[0].Complexity)
	}
}

func TestSyntaxErrorSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	src := `void broken( {`
	writeTmpCppFile(t, tmpDir, "broken.cpp", src)

	report, err := Check([]string{tmpDir}, 6, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Skipped) != 1 {
		t.Errorf("expected 1 skipped file, got %d", len(report.Skipped))
	}
	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations from skipped file, got %d", len(report.Violations))
	}
}

func TestTernaryOperator(t *testing.T) {
	tmpDir := t.TempDir()
	// Complexity 2: base 1 + ternary (1)
	src := `void func() {
  int x = (true) ? 1 : 2;
}`
	writeTmpCppFile(t, tmpDir, "test.cpp", src)

	report, err := Check([]string{tmpDir}, 1, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Errorf("expected 1 violation at limit 1, got %d", len(report.Violations))
	}
	if len(report.Violations) > 0 && report.Violations[0].Complexity != 2 {
		t.Errorf("expected complexity 2, got %d", report.Violations[0].Complexity)
	}
}

func TestCatchClause(t *testing.T) {
	tmpDir := t.TempDir()
	// Complexity 3: base 1 + try (0) + 2 catch clauses
	src := `void func() {
  try {
    foo();
  } catch (std::exception& e) {
    bar();
  } catch (...) {
    baz();
  }
}`
	writeTmpCppFile(t, tmpDir, "test.cpp", src)

	report, err := Check([]string{tmpDir}, 10, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations with limit 10, got %d", len(report.Violations))
	}

	report2, _ := Check([]string{tmpDir}, 2, Options{})
	if len(report2.Violations) != 1 {
		t.Errorf("expected 1 violation at limit 2, got %d", len(report2.Violations))
	}
	if len(report2.Violations) > 0 && report2.Violations[0].Complexity != 3 {
		t.Errorf("expected complexity 3, got %d", report2.Violations[0].Complexity)
	}
}

func TestExcludeFuncs(t *testing.T) {
	tmpDir := t.TempDir()
	src := `void complexFunc() {
  if (1) if (2) if (3) if (4) if (5) if (6) if (7) {}
}
void simpleFunc() {
  return;
}`
	writeTmpCppFile(t, tmpDir, "test.cpp", src)

	report, err := Check([]string{tmpDir}, 6, Options{
		ExcludeFuncs: []string{"complexFunc"},
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations with excluded func, got %d", len(report.Violations))
	}
}

func TestMultipleFunctions(t *testing.T) {
	tmpDir := t.TempDir()
	src := `void func1() {
  if (1) if (2) if (3) if (4) if (5) if (6) if (7) {}
}
void func2() {
  if (1) if (2) {}
}
void func3() {
  if (1) if (2) if (3) if (4) if (5) if (6) if (7) {}
}`
	writeTmpCppFile(t, tmpDir, "test.cpp", src)

	report, err := Check([]string{tmpDir}, 6, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 2 {
		t.Errorf("expected 2 violations, got %d", len(report.Violations))
	}
}

func TestForLoop(t *testing.T) {
	tmpDir := t.TempDir()
	// Complexity 3: base 1 + for (1) + if (1)
	src := `void func() {
  for (int i = 0; i < 10; i++) {
    if (i > 5) {}
  }
}`
	writeTmpCppFile(t, tmpDir, "test.cpp", src)

	report, err := Check([]string{tmpDir}, 2, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Errorf("expected 1 violation at limit 2, got %d", len(report.Violations))
	}
	if len(report.Violations) > 0 && report.Violations[0].Complexity != 3 {
		t.Errorf("expected complexity 3, got %d", report.Violations[0].Complexity)
	}
}

func TestRangeFor(t *testing.T) {
	tmpDir := t.TempDir()
	// Complexity 2: base 1 + for-range (1)
	src := `void func(const std::vector<int>& v) {
  for (int x : v) {
    process(x);
  }
}`
	writeTmpCppFile(t, tmpDir, "test.cpp", src)

	report, err := Check([]string{tmpDir}, 1, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Errorf("expected 1 violation at limit 1, got %d", len(report.Violations))
	}
	if len(report.Violations) > 0 && report.Violations[0].Complexity != 2 {
		t.Errorf("expected complexity 2, got %d", report.Violations[0].Complexity)
	}
}

func TestWhileLoop(t *testing.T) {
	tmpDir := t.TempDir()
	// Complexity 2: base 1 + while (1)
	src := `void func() {
  while (x > 0) {
    x--;
  }
}`
	writeTmpCppFile(t, tmpDir, "test.cpp", src)

	report, err := Check([]string{tmpDir}, 1, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Errorf("expected 1 violation at limit 1, got %d", len(report.Violations))
	}
	if len(report.Violations) > 0 && report.Violations[0].Complexity != 2 {
		t.Errorf("expected complexity 2, got %d", report.Violations[0].Complexity)
	}
}

func TestDoWhileLoop(t *testing.T) {
	tmpDir := t.TempDir()
	// Complexity 2: base 1 + do-while (1)
	src := `void func() {
  do {
    x++;
  } while (x < 10);
}`
	writeTmpCppFile(t, tmpDir, "test.cpp", src)

	report, err := Check([]string{tmpDir}, 1, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Errorf("expected 1 violation at limit 1, got %d", len(report.Violations))
	}
	if len(report.Violations) > 0 && report.Violations[0].Complexity != 2 {
		t.Errorf("expected complexity 2, got %d", report.Violations[0].Complexity)
	}
}

func TestEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	writeTmpCppFile(t, tmpDir, "empty.cpp", "")

	report, err := Check([]string{tmpDir}, 6, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations in empty file, got %d", len(report.Violations))
	}
}

func TestNoFunctions(t *testing.T) {
	tmpDir := t.TempDir()
	src := `// Just a comment
int x = 5;
void globalVar;`
	writeTmpCppFile(t, tmpDir, "test.cpp", src)

	report, err := Check([]string{tmpDir}, 6, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations with no functions, got %d", len(report.Violations))
	}
}

func TestDebugExcludedFuncs(t *testing.T) {
	tmpDir := t.TempDir()
	src := `void complexFunc() {
  if (1) if (2) if (3) if (4) if (5) if (6) if (7) {}
}`
	writeTmpCppFile(t, tmpDir, "test.cpp", src)

	report, err := Check([]string{tmpDir}, 6, Options{
		ExcludeFuncs: []string{"complexFunc"},
		Debug:        true,
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.ExcludedFuncs) != 1 {
		t.Errorf("expected 1 excluded func in debug mode, got %d", len(report.ExcludedFuncs))
	}
	if len(report.ExcludedFuncs) > 0 {
		ex := report.ExcludedFuncs[0]
		if ex.Func != "complexFunc" {
			t.Errorf("expected excluded func 'complexFunc', got '%s'", ex.Func)
		}
		if !strings.Contains(ex.Reason, "exclude pattern") {
			t.Errorf("expected reason to mention exclude pattern, got '%s'", ex.Reason)
		}
	}
}
