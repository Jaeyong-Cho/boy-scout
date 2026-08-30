package tscomplexity

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTmpTsFile(t *testing.T, dir, filename, content string) string {
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", filename, err)
	}
	return path
}

func TestComplexity7FlaggedAtDefault6(t *testing.T) {
	tmpDir := t.TempDir()
	// Complexity 7: base 1 + 6 if statements
	src := `function func() {
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
	writeTmpTsFile(t, tmpDir, "test.ts", src)

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
	src := `function func() {
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
	writeTmpTsFile(t, tmpDir, "test.ts", src)

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
	src := `function func() {
  if (a && b || c) {}
}`
	writeTmpTsFile(t, tmpDir, "test.ts", src)

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
	src := `function func(x) {
  switch (x) {
    case 1: break;
    case 2: break;
    case 3: break;
    default: break;
  }
}`
	writeTmpTsFile(t, tmpDir, "test.ts", src)

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

func TestForOfAndForIn(t *testing.T) {
	tmpDir := t.TempDir()
	// Two functions, each with complexity 2 (base 1 + for_in_statement 1)
	src := `function forOf() {
  for (const x of arr) {}
}
function forIn() {
  for (const k in obj) {}
}`
	writeTmpTsFile(t, tmpDir, "test.ts", src)

	report, err := Check([]string{tmpDir}, 1, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 2 {
		t.Errorf("expected 2 violations at limit 1, got %d", len(report.Violations))
	}
}

func TestForLoop(t *testing.T) {
	tmpDir := t.TempDir()
	// Complexity 3: base 1 + for (1) + if (1)
	src := `function func() {
  for (let i = 0; i < 10; i++) {
    if (i > 5) {}
  }
}`
	writeTmpTsFile(t, tmpDir, "test.ts", src)

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

func TestNullishAndOptionalChainingNotCounted(t *testing.T) {
	tmpDir := t.TempDir()
	// Complexity 1: base 1 — nullish coalescing, optional chaining, and logical assignment NOT counted
	src := `function func() {
  const a = x ?? y;
  const b = obj?.prop;
  let c = c ?? 5;
  c ??= 10;
  c ||= 20;
  c &&= 30;
}`
	writeTmpTsFile(t, tmpDir, "test.ts", src)

	report, err := Check([]string{tmpDir}, 1, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations at limit 1, got %d", len(report.Violations))
	}
}

func TestAnonymousCallbackCountsTowardEnclosing(t *testing.T) {
	tmpDir := t.TempDir()
	// The arrow function inside map is not extracted as a separate function,
	// its if statement counts toward the enclosing named function
	// Complexity 2: base 1 + if (1)
	src := `function process() {
  arr.map(x => { if (x) return x; });
}`
	writeTmpTsFile(t, tmpDir, "test.ts", src)

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

func TestSyntaxErrorSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	src := `function broken( {`
	writeTmpTsFile(t, tmpDir, "broken.ts", src)

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

func TestExcludeFuncs(t *testing.T) {
	tmpDir := t.TempDir()
	src := `function complexFunc() {
  if (1) if (2) if (3) if (4) if (5) if (6) if (7) {}
}
function simpleFunc() {
  return;
}`
	writeTmpTsFile(t, tmpDir, "test.ts", src)

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

func TestTsxFile(t *testing.T) {
	tmpDir := t.TempDir()
	// Complexity 7: base 1 + 6 if statements
	src := `function Component() {
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
  return <div></div>;
}`
	writeTmpTsFile(t, tmpDir, "test.tsx", src)

	report, err := Check([]string{tmpDir}, 6, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Errorf("expected 1 violation in tsx file, got %d", len(report.Violations))
	}
	if len(report.Violations) > 0 {
		v := report.Violations[0]
		if v.Complexity != 7 {
			t.Errorf("expected complexity 7, got %d", v.Complexity)
		}
	}
}
