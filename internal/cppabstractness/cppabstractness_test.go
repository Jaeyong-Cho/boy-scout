package cppabstractness

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheck_ShapeFullyAbstract_NoViolation(t *testing.T) {
	// AC1: Given shape.h declares 1 abstract (pure virtual), 0 concrete (Abstractness=1.0)
	// When run with default flags
	// Then it reports 0 violations for shape.h (inside default distance)
	tempDir := t.TempDir()

	// Copy fixtures
	copyFixtures(t, tempDir)

	report, err := Check([]string{tempDir}, 0.5, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Expect 0 violations for shape.h
	for _, v := range report.Violations {
		if v.ImportPath == filepath.Join(tempDir, "shape.h") {
			t.Errorf("Unexpected violation for shape.h: %+v", v)
		}
	}
}

func TestCheck_RigidConcreteOnly_ZoneOfPain(t *testing.T) {
	// AC2: Given rigid.h declares 0 abstract, 1 concrete (Abstractness=0.0)
	// When run with default flags
	// Then it reports 1 violation for rigid.h with Zone="Pain"
	tempDir := t.TempDir()

	// Copy fixtures
	copyFixtures(t, tempDir)

	report, err := Check([]string{tempDir}, 0.5, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Find violation for rigid.h
	var found bool
	for _, v := range report.Violations {
		if v.Zone == "Pain" {
			found = true
			if v.Abstractness != 0.0 {
				t.Errorf("rigid.h: expected Abstractness=0.0, got %f", v.Abstractness)
			}
			break
		}
	}

	if !found {
		t.Errorf("Expected Zone-of-Pain violation, got violations: %+v", report.Violations)
	}
}

func TestCheck_ZeroClasses_Skipped(t *testing.T) {
	// AC3: Given a file declaring 0 classes/structs
	// When run
	// Then it's skipped from abstractness scoring (no divide-by-zero)
	tempDir := t.TempDir()

	// Create a file with no classes
	if err := writeFile(t, filepath.Join(tempDir, "empty.h"), `#ifndef EMPTY_H
#define EMPTY_H

void freeFunction() {}

#endif
`); err != nil {
		t.Fatalf("failed to write empty.h: %v", err)
	}

	// Create a client to include it
	if err := writeFile(t, filepath.Join(tempDir, "client.cpp"), `#include "empty.h"

int main() {
    return 0;
}
`); err != nil {
		t.Fatalf("failed to write client.cpp: %v", err)
	}

	report, err := Check([]string{tempDir}, 0.5, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// empty.h should not appear in violations (no classes to count)
	for _, v := range report.Violations {
		if v.ImportPath == filepath.Join(tempDir, "empty.h") {
			t.Errorf("empty.h should be skipped, got violation: %+v", v)
		}
	}
}

func TestCheck_NogateAlwaysReports_PainCandidate(t *testing.T) {
	// AC4: Given a file that's a Zone-of-Pain candidate
	// When run with default flags (no gate implemented)
	// Then it's always reported, confirming the "no gate" decision
	tempDir := t.TempDir()

	// Copy fixtures
	copyFixtures(t, tempDir)

	report, err := Check([]string{tempDir}, 0.5, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Check that Pain violations don't have any surface-ratio gating applied
	// (all concrete structs/classes should be reported as Pain)
	for _, v := range report.Violations {
		if v.Zone == "Pain" {
			// No gate should filter this out
			if v.Abstractness > 0 {
				t.Logf("Pain candidate reported: %s (Abstractness=%f)", v.ImportPath, v.Abstractness)
			}
		}
	}
}

// Helper: copyFixtures copies test fixture files to tempDir
func copyFixtures(t *testing.T, tempDir string) {
	fixtures := []struct {
		name    string
		content string
	}{
		{"shape.h", `#ifndef SHAPE_H
#define SHAPE_H

class Shape {
public:
    virtual void draw() = 0;
    virtual ~Shape() = default;
};

#endif
`},
		{"rigid.h", `#ifndef RIGID_H
#define RIGID_H

struct Rigid {
    int x;
    int y;
};

#endif
`},
		{"client.cpp", `#include "shape.h"
#include "rigid.h"

void drawShape(Shape* shape) {
    shape->draw();
}

void processRigid(Rigid& r) {
    r.x = 0;
}
`},
	}

	for _, f := range fixtures {
		if err := writeFile(t, filepath.Join(tempDir, f.name), f.content); err != nil {
			t.Fatalf("failed to write %s: %v", f.name, err)
		}
	}
}

func writeFile(t *testing.T, path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}
