package cppcohesion

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheck_ReportsLowCohesionClass(t *testing.T) {
	tmpdir := t.TempDir()

	// Low-cohesion class: two methods that don't share fields or call each other
	source := `class Foo {
public:
    int x;
    int y;
    void setX(int v) { x = v; }
    void setY(int v) { y = v; }
};`

	file := filepath.Join(tmpdir, "foo.cpp")
	if err := os.WriteFile(file, []byte(source), 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	report, err := Check([]string{tmpdir}, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Fatalf("Expected 1 violation, got %d", len(report.Violations))
	}

	v := report.Violations[0]
	if v.Class != "Foo" {
		t.Errorf("Expected class name 'Foo', got %q", v.Class)
	}
	if v.LCOM4 != 2 {
		t.Errorf("Expected LCOM4=2, got %d", v.LCOM4)
	}
	if v.LCOM4Level != "warning" {
		t.Errorf("Expected LCOM4Level='warning', got %q", v.LCOM4Level)
	}
	if v.TCC != 0.0 {
		t.Errorf("Expected TCC=0, got %f", v.TCC)
	}
	if v.TCCLevel != "danger" {
		t.Errorf("Expected TCCLevel='danger', got %q", v.TCCLevel)
	}
}

func TestCheck_SkipsUnparseableFile(t *testing.T) {
	tmpdir := t.TempDir()

	// Invalid C++ syntax - won't parse as a class, so no violations expected
	source := `this is not valid c++`

	file := filepath.Join(tmpdir, "broken.cpp")
	if err := os.WriteFile(file, []byte(source), 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	report, err := Check([]string{tmpdir}, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Simple parser doesn't fail on invalid syntax, just doesn't find classes
	// So we should have no violations
	if len(report.Violations) != 0 {
		t.Errorf("Expected 0 violations, got %d", len(report.Violations))
	}
}
