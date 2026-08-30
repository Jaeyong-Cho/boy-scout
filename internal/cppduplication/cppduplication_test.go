package cppduplication

import (
	"os"
	"testing"
)

// TestCheck_ReportsType1ExactDuplicate verifies that byte-identical function bodies
// in different files are detected as Type-1 clones
func TestCheck_ReportsType1ExactDuplicate(t *testing.T) {
	tmpDir := t.TempDir()

	// Write first C++ file with functions (5 lines to meet minLines threshold)
	funcBodyA := `int addInts(int a, int b) {
    int x = a;
    int y = b;
    int sum = x + y;
    return sum;
}
`
	funcBodyB := `int addNums(int a, int b) {
    int x = a;
    int y = b;
    int sum = x + y;
    return sum;
}
`

	fileA := tmpDir + "/a.cpp"
	fileB := tmpDir + "/b.cpp"

	if err := os.WriteFile(fileA, []byte(funcBodyA), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(fileB, []byte(funcBodyB), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	report, err := Check([]string{tmpDir}, 5, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) == 0 {
		t.Fatal("expected at least one violation for Type-1 duplicate, got 0")
	}

	violation := report.Violations[0]
	if violation.Type != "Type-1" {
		t.Errorf("expected Type-1, got %s", violation.Type)
	}
	if violation.FileA != fileA || violation.FileB != fileB {
		t.Errorf("expected files %s and %s, got %s and %s", fileA, fileB, violation.FileA, violation.FileB)
	}
}

// TestCheck_ReportsType2RenamedDuplicate verifies Type-2 detection with renamed identifiers
func TestCheck_ReportsType2RenamedDuplicate(t *testing.T) {
	tmpDir := t.TempDir()

	funcBodyA := `int addInts(int a, int b) {
    int x = a;
    int y = b;
    int sum = x + y;
    return sum;
}
`
	// Same structure but renamed identifiers
	funcBodyB := `int addNumbers(int param1, int param2) {
    int var1 = param1;
    int var2 = param2;
    int total = var1 + var2;
    return total;
}
`

	fileA := tmpDir + "/a.cpp"
	fileB := tmpDir + "/b.cpp"

	if err := os.WriteFile(fileA, []byte(funcBodyA), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(fileB, []byte(funcBodyB), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	report, err := Check([]string{tmpDir}, 5, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) == 0 {
		t.Fatal("expected at least one violation for Type-2 duplicate, got 0")
	}

	violation := report.Violations[0]
	if violation.Type != "Type-2" {
		t.Errorf("expected Type-2, got %s", violation.Type)
	}
}

// TestCheck_ReportsType3NearMiss verifies Type-3 detection with LCS similarity
func TestCheck_ReportsType3NearMiss(t *testing.T) {
	tmpDir := t.TempDir()

	funcBodyA := `int addInts(int a, int b) {
    int x = a;
    int y = b;
    int sum = x + y;
    return sum;
}
`
	// Same structure with one extra harmless line
	funcBodyB := `int addNumbers(int a, int b) {
    int x = a;
    int y = b;
    (void)x;
    int sum = x + y;
    return sum;
}
`

	fileA := tmpDir + "/a.cpp"
	fileB := tmpDir + "/b.cpp"

	if err := os.WriteFile(fileA, []byte(funcBodyA), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(fileB, []byte(funcBodyB), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	report, err := Check([]string{tmpDir}, 5, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) == 0 {
		t.Fatal("expected at least one violation for Type-3 duplicate, got 0")
	}

	violation := report.Violations[0]
	if violation.Type != "Type-3" {
		t.Errorf("expected Type-3, got %s", violation.Type)
	}
	if violation.Similarity < 0.70 || violation.Similarity > 1.0 {
		t.Errorf("expected similarity in [0.70, 1.0], got %f", violation.Similarity)
	}
}

// TestCheck_NoViolationBelowMinLines verifies that short functions are skipped
func TestCheck_NoViolationBelowMinLines(t *testing.T) {
	tmpDir := t.TempDir()

	// 2-line functions
	funcBody := `int id(int a) {
    return a;
}
`

	fileA := tmpDir + "/a.cpp"
	fileB := tmpDir + "/b.cpp"

	if err := os.WriteFile(fileA, []byte(funcBody), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(fileB, []byte(funcBody), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	report, err := Check([]string{tmpDir}, 5, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) > 0 {
		t.Errorf("expected no violations below minLines, got %d", len(report.Violations))
	}
}

// TestCheck_ScansHeaderFiles verifies that .hpp files are scanned
func TestCheck_ScansHeaderFiles(t *testing.T) {
	tmpDir := t.TempDir()

	fileA := tmpDir + "/getter.hpp"
	fileB := tmpDir + "/other.hpp"

	// Make the getter body longer to meet minLines
	longGetter := `int getValue() {
    int x = 42;
    int y = x;
    int z = y;
    return z;
}
`

	if err := os.WriteFile(fileA, []byte(longGetter), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(fileB, []byte(longGetter), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	report, err := Check([]string{tmpDir}, 5, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if len(report.Violations) == 0 {
		t.Fatal("expected violations in .hpp files")
	}

	violation := report.Violations[0]
	if violation.FileA != fileA || violation.FileB != fileB {
		t.Errorf("expected .hpp files, got %s and %s", violation.FileA, violation.FileB)
	}
}

// TestCheck_SkipsUnparseableFileAndContinues verifies error handling
func TestCheck_SkipsUnparseableFileAndContinues(t *testing.T) {
	tmpDir := t.TempDir()

	brokenCode := `int broken() {
    {{{  // unmatched braces
`

	validCode := `int func1() {
    int x = 1;
    int y = 2;
    int z = 3;
    return x + y + z;
}
int func2() {
    int x = 1;
    int y = 2;
    int z = 3;
    return x + y + z;
}
`

	if err := os.WriteFile(tmpDir+"/broken.cpp", []byte(brokenCode), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(tmpDir+"/valid.cpp", []byte(validCode), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	report, err := Check([]string{tmpDir}, 5, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Should have skipped the broken file
	if len(report.Skipped) == 0 {
		t.Fatal("expected skipped file for parse error")
	}

	// Should have found the duplicate in the valid file
	if len(report.Violations) == 0 {
		t.Fatal("expected violations from valid file despite parse error in broken file")
	}
}
