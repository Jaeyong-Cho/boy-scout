package crap

import (
	"strings"
	"testing"
)

func TestParseProfile_ParsesValidEntries(t *testing.T) {
	profile := `mode: set
example.com/pkg/file.go:3.14,5.2 2 1
example.com/pkg/file.go:6.2,6.10 1 0
`
	blocks, err := parseProfile(strings.NewReader(profile))
	if err != nil {
		t.Fatalf("parseProfile failed: %v", err)
	}

	if len(blocks) != 2 {
		t.Errorf("expected 2 blocks, got %d", len(blocks))
	}

	// Check first block
	if blocks[0].file != "example.com/pkg/file.go" {
		t.Errorf("expected file 'example.com/pkg/file.go', got '%s'", blocks[0].file)
	}
	if blocks[0].startLine != 3 {
		t.Errorf("expected startLine 3, got %d", blocks[0].startLine)
	}
	if blocks[0].endLine != 5 {
		t.Errorf("expected endLine 5, got %d", blocks[0].endLine)
	}
	if blocks[0].numStmt != 2 {
		t.Errorf("expected numStmt 2, got %d", blocks[0].numStmt)
	}
	if blocks[0].count != 1 {
		t.Errorf("expected count 1, got %d", blocks[0].count)
	}

	// Check second block
	if blocks[1].file != "example.com/pkg/file.go" {
		t.Errorf("expected file 'example.com/pkg/file.go', got '%s'", blocks[1].file)
	}
	if blocks[1].startLine != 6 {
		t.Errorf("expected startLine 6, got %d", blocks[1].startLine)
	}
	if blocks[1].endLine != 6 {
		t.Errorf("expected endLine 6, got %d", blocks[1].endLine)
	}
	if blocks[1].numStmt != 1 {
		t.Errorf("expected numStmt 1, got %d", blocks[1].numStmt)
	}
	if blocks[1].count != 0 {
		t.Errorf("expected count 0, got %d", blocks[1].count)
	}
}

func TestFunctionCoverage_FileAbsentFromProfileIsZero(t *testing.T) {
	blocks := []profileBlock{} // Empty
	cov := functionCoverage(blocks, false, 1, 10)
	if cov != 0.0 {
		t.Errorf("expected 0.0, got %f", cov)
	}
}

func TestFunctionCoverage_EmptyFunctionBodyIsFullyCovered(t *testing.T) {
	blocks := []profileBlock{
		{file: "test.go", startLine: 1, endLine: 3, numStmt: 5, count: 1},
		{file: "test.go", startLine: 5, endLine: 7, numStmt: 3, count: 0},
	}
	// Function is at lines 10-12 with no matching blocks
	cov := functionCoverage(blocks, true, 10, 12)
	if cov != 1.0 {
		t.Errorf("expected 1.0, got %f", cov)
	}
}

func TestFunctionCoverage_PartialCoverageComputesFraction(t *testing.T) {
	blocks := []profileBlock{
		{file: "test.go", startLine: 1, endLine: 5, numStmt: 3, count: 1},
		{file: "test.go", startLine: 5, endLine: 10, numStmt: 3, count: 0},
		{file: "test.go", startLine: 10, endLine: 15, numStmt: 4, count: 1},
	}
	// Function covers lines 1-15: total 10 stmts, 7 covered (3+4), 6 uncovered (3)
	// Wait, let me recalculate: blocks 1 (3 stmts, covered), 2 (3 stmts, uncovered), 3 (4 stmts, covered)
	// Total = 10, Covered = 7 - wait, the test says 6 covered out of 10
	// Let me re-read the test comment: "blocks summing to 10 total statements within range, 6 of them (by summed numStmt where count>0) covered"
	// So blocks should be:
	// - 1 block with 6 stmts, all covered (count > 0)
	// - 1 block with 4 stmts, not covered (count == 0)
	// Total = 10, covered = 6, coverage = 0.6

	blocks = []profileBlock{
		{file: "test.go", startLine: 1, endLine: 5, numStmt: 6, count: 1},
		{file: "test.go", startLine: 5, endLine: 10, numStmt: 4, count: 0},
	}
	cov := functionCoverage(blocks, true, 1, 10)
	if cov != 0.6 {
		t.Errorf("expected 0.6, got %f", cov)
	}
}
