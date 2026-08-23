package tsfunclen

import (
	"os"
	"testing"
)

func genBody(n int) string {
	code := ""
	for i := 0; i < n; i++ {
		code += "  const x = 1;\n"
	}
	return code
}

func TestCheck_ReportsViolationForOverLimitFunction(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a function exactly 62 lines (from opening brace to closing brace)
	code := "function renderWidget() {\n" + genBody(60) + "}\n"

	filePath := tmpDir + "/widget.ts"
	os.WriteFile(filePath, []byte(code), 0644)

	report, err := Check([]string{tmpDir}, 50, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Errorf("Expected 1 violation, got %d", len(report.Violations))
	}

	if len(report.Violations) > 0 {
		v := report.Violations[0]
		if v.Func != "renderWidget" {
			t.Errorf("Expected function name renderWidget, got %s", v.Func)
		}
		if v.Length != 62 {
			t.Errorf("Expected length 62, got %d", v.Length)
		}
		if v.Limit != 50 {
			t.Errorf("Expected limit 50, got %d", v.Limit)
		}
	}
}

func TestCheck_CountsNamedArrowFunctionAsFunction(t *testing.T) {
	tmpDir := t.TempDir()

	// Named arrow function over limit
	code := "const renderWidget = () => {\n" + genBody(60) + "};\n"
	filePath := tmpDir + "/widget.ts"
	os.WriteFile(filePath, []byte(code), 0644)

	report, err := Check([]string{tmpDir}, 50, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Errorf("Expected 1 violation, got %d", len(report.Violations))
	}

	if len(report.Violations) > 0 {
		v := report.Violations[0]
		if v.Func != "renderWidget" {
			t.Errorf("Expected function name renderWidget, got %s", v.Func)
		}
		if v.Length != 62 {
			t.Errorf("Expected length 62, got %d", v.Length)
		}
	}
}

func TestCheck_ParsesTsxFunctionComponent(t *testing.T) {
	tmpDir := t.TempDir()

	// TSX file with JSX (55 lines total from opening brace to closing brace)
	code := "function Button() {\n" + genBody(52) + "  return <div>Click me</div>;\n}\n"
	filePath := tmpDir + "/Button.tsx"
	os.WriteFile(filePath, []byte(code), 0644)

	report, err := Check([]string{tmpDir}, 50, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Errorf("Expected 1 violation for TSX, got %d", len(report.Violations))
	}

	if len(report.Skipped) > 0 {
		t.Errorf("Expected 0 skipped files for valid TSX, got %d: %v", len(report.Skipped), report.Skipped)
	}

	if len(report.Violations) > 0 {
		v := report.Violations[0]
		if v.Func != "Button" {
			t.Errorf("Expected function name Button, got %s", v.Func)
		}
		if v.Length != 55 {
			t.Errorf("Expected length 55, got %d", v.Length)
		}
	}
}

func TestCheck_AttributesAnonymousCallbackLinesToEnclosingFunction(t *testing.T) {
	tmpDir := t.TempDir()

	// Outer function containing anonymous arrow callback in map
	// Total: 1 (function decl) + 45 (outer body) + 1 (closing brace) = 47 lines initially
	// Add 10 more via genBody to make it 57 lines
	code := "function outerFunction() {\n" +
		"  const items = [1, 2, 3];\n" +
		"  const mapped = items.map(x => {\n" +
		"    const y = x * 2;\n" +
		"    return y;\n" +
		"  });\n" +
		genBody(44) +
		"}\n"

	filePath := tmpDir + "/list.ts"
	os.WriteFile(filePath, []byte(code), 0644)

	report, err := Check([]string{tmpDir}, 50, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Only outerFunction should be reported
	if len(report.Violations) != 1 {
		t.Errorf("Expected 1 violation (only outer), got %d\nViolations: %v\nSkipped: %v", len(report.Violations), report.Violations, report.Skipped)
	}

	if len(report.Violations) > 0 {
		v := report.Violations[0]
		if v.Func != "outerFunction" {
			t.Errorf("Expected function name outerFunction, got %s", v.Func)
		}
	}
}

func TestCheck_ExactlyAtLimitIsCompliant(t *testing.T) {
	tmpDir := t.TempDir()

	// Function exactly 50 lines (should NOT be reported)
	code := "function compliant() {\n" + genBody(48) + "}\n"
	filePath := tmpDir + "/compliant.ts"
	os.WriteFile(filePath, []byte(code), 0644)

	report, err := Check([]string{tmpDir}, 50, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("Expected 0 violations for function at limit, got %d", len(report.Violations))
	}
}

func TestCheck_SyntaxErrorFileIsSkippedEntirely(t *testing.T) {
	tmpDir := t.TempDir()

	// Broken syntax with error node (unclosed brace with content after)
	code := "function foo() {\n  const x = 1;\n}\nconst y = <\n"
	filePath := tmpDir + "/broken.ts"
	os.WriteFile(filePath, []byte(code), 0644)

	report, err := Check([]string{tmpDir}, 50, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("Expected 0 violations for broken file, got %d", len(report.Violations))
	}

	if len(report.Skipped) != 1 {
		t.Errorf("Expected 1 skipped file, got %d: %v", len(report.Skipped), report.Skipped)
	}

	if len(report.ExcludedFuncs) != 0 {
		t.Errorf("Expected 0 excluded funcs for broken file, got %d", len(report.ExcludedFuncs))
	}
}

func TestCheck_ZeroMatchingFilesReturnsEmptyReport(t *testing.T) {
	tmpDir := t.TempDir()

	// Empty directory
	report, err := Check([]string{tmpDir}, 50, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("Expected 0 violations for empty dir, got %d", len(report.Violations))
	}

	if len(report.Skipped) != 0 {
		t.Errorf("Expected 0 skipped for empty dir, got %d", len(report.Skipped))
	}

	if len(report.ExcludedFuncs) != 0 {
		t.Errorf("Expected 0 excluded funcs for empty dir, got %d", len(report.ExcludedFuncs))
	}
}
