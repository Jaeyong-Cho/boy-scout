package gocohesion_test

import (
	"os"
	"path/filepath"
	"testing"

	"boy-scout/internal/gocohesion"
)

func TestCheck_ReportsLowCohesionStruct(t *testing.T) {
	// AC7: Given a struct with 2+ methods, no shared fields, no calls
	src := `package main

type Foo struct{
	x, y int
}

func (f *Foo) SetX(v int) { f.x = v }
func (f *Foo) SetY(v int) { f.y = v }
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(tmpFile, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	report, err := gocohesion.Check([]string{tmpDir}, gocohesion.Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(report.Violations))
	}

	v := report.Violations[0]
	if v.Class != "Foo" {
		t.Errorf("class: got %q, want %q", v.Class, "Foo")
	}
	if v.LCOM4 != 2 {
		t.Errorf("LCOM4: got %d, want 2", v.LCOM4)
	}
	if v.TCC != 0.0 {
		t.Errorf("TCC: got %f, want 0.0", v.TCC)
	}
	if v.LCC != 0.0 {
		t.Errorf("LCC: got %f, want 0.0", v.LCC)
	}
}

func TestCheck_SkipsStructWithOneMethod(t *testing.T) {
	// AC6: struct with 1 method should not appear in report
	src := `package main

type Single struct{
	x int
}

func (s *Single) GetX() int { return s.x }
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(tmpFile, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	report, err := gocohesion.Check([]string{tmpDir}, gocohesion.Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations for struct with 1 method, got %d", len(report.Violations))
	}
}

func TestCheck_MethodsAcrossFilesSameStruct(t *testing.T) {
	// AC8: methods in different files of same package belong to same struct
	src1 := `package main

type Cohesive struct{
	x int
}

func (c *Cohesive) SetX(v int) { c.x = v }
`
	src2 := `package main

func (c *Cohesive) GetX() int { return c.x }
`
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "file1.go"), []byte(src1), 0644); err != nil {
		t.Fatalf("WriteFile file1 failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte(src2), 0644); err != nil {
		t.Fatalf("WriteFile file2 failed: %v", err)
	}

	report, err := gocohesion.Check([]string{tmpDir}, gocohesion.Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Both methods touch x, so high cohesion (TCC = 1, LCC = 1) → no violation
	if len(report.Violations) != 0 {
		t.Errorf("expected 0 violations for cohesive struct across files, got %d", len(report.Violations))
	}
}

func TestCheck_SkipsUnparseableFile(t *testing.T) {
	// AC13: unparseable file should be skipped, not crash
	src := "package main\n\nthis is not valid go syntax @@@@"
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.go")
	if err := os.WriteFile(tmpFile, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	report, err := gocohesion.Check([]string{tmpDir}, gocohesion.Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Skipped) == 0 {
		t.Errorf("expected file to be skipped, got none")
	}
}
