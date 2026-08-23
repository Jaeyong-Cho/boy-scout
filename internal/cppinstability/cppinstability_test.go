package cppinstability

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildGraphNoViolations(t *testing.T) {
	// Test AC 1: normal case with no direction violations
	// When running go test, CWD is the project root
	fixturePath := "testdata/cpp-instability"

	// Get current working directory and navigate to project root
	cwd, _ := os.Getwd()
	// From internal/cppinstability, go up to boy-scout root
	projectRoot := filepath.Join(cwd, "../..")
	fixturePath = filepath.Join(projectRoot, fixturePath)

	t.Logf("CWD: %s", cwd)
	t.Logf("ProjectRoot: %s", projectRoot)
	t.Logf("Fixture path: %s", fixturePath)

	graph, err := BuildGraph([]string{fixturePath}, Options{})
	if err != nil {
		t.Fatalf("BuildGraph failed: %v", err)
	}

	// Should have 5 edges (lexer.h→token.h, lexer.cpp→lexer.h, parser.h→token.h, parser.cpp→parser.h, parser.cpp→token.h)
	if len(graph.Edges) != 5 {
		t.Errorf("Expected 5 edges, got %d", len(graph.Edges))
	}

	// Debug: print actual packages
	t.Logf("Graph packages:")
	for k, v := range graph.Packages {
		t.Logf("  %s: Ca=%d, Ce=%d, I=%f", k, v.Ca, v.Ce, v.Instability)
	}

	// token.h should have Ca=3, Ce=0, I=0
	// Package key will be just "token.h" since the root is the fixture directory
	tokenKey := "token.h"
	tokenStats, ok := graph.Packages[tokenKey]
	if !ok {
		// Try to find it with different path separators or normalization
		for k := range graph.Packages {
			if strings.Contains(k, "token.h") {
				t.Logf("Found token.h at key: %s", k)
				tokenKey = k
				tokenStats = graph.Packages[k]
				ok = true
				break
			}
		}
		if !ok {
			t.Errorf("token.h not in graph.Packages")
		}
	}
	if ok {
		if tokenStats.Ca != 3 {
			t.Errorf("token.h: expected Ca=3, got %d", tokenStats.Ca)
		}
		if tokenStats.Ce != 0 {
			t.Errorf("token.h: expected Ce=0, got %d", tokenStats.Ce)
		}
		if tokenStats.Instability != 0 {
			t.Errorf("token.h: expected I=0, got %f", tokenStats.Instability)
		}
	}

	// parser.cpp should have Ca=0, Ce=2, I=1.0
	parserCppKey := "parser.cpp"
	parserCppStats, ok := graph.Packages[parserCppKey]
	if !ok {
		// Try to find it with different path separators or normalization
		for k := range graph.Packages {
			if strings.Contains(k, "parser.cpp") {
				t.Logf("Found parser.cpp at key: %s", k)
				parserCppKey = k
				parserCppStats = graph.Packages[k]
				ok = true
				break
			}
		}
		if !ok {
			t.Errorf("parser.cpp not in graph.Packages")
		}
	}
	if ok {
		if parserCppStats.Ca != 0 {
			t.Errorf("parser.cpp: expected Ca=0, got %d", parserCppStats.Ca)
		}
		if parserCppStats.Ce != 2 {
			t.Errorf("parser.cpp: expected Ce=2, got %d", parserCppStats.Ce)
		}
		if parserCppStats.Instability != 1.0 {
			t.Errorf("parser.cpp: expected I=1.0, got %f", parserCppStats.Instability)
		}
	}
}

func TestBuildGraphViolation(t *testing.T) {
	// Test AC 2: with token.h → parser.h edge (direction violation)
	cwd, _ := os.Getwd()
	projectRoot := filepath.Join(cwd, "../..")
	fixturePath := filepath.Join(projectRoot, "testdata/cpp-instability")
	tokenHPath := filepath.Join(fixturePath, "token.h")

	// Temporarily modify token.h to add #include "parser.h"
	originalContent, err := os.ReadFile(tokenHPath)
	if err != nil {
		t.Fatalf("Failed to read token.h: %v", err)
	}
	defer os.WriteFile(tokenHPath, originalContent, 0644) // Restore after test

	modifiedContent := "#ifndef TOKEN_H\n#define TOKEN_H\n\n#include \"parser.h\"\n\n// Token class - now depends on parser.h\n\n#endif\n"
	if err := os.WriteFile(tokenHPath, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify token.h: %v", err)
	}

	report, err := Check([]string{fixturePath}, 0, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Should report exactly 1 violation: token.h → parser.h
	if len(report.Violations) != 1 {
		t.Errorf("Expected 1 violation, got %d", len(report.Violations))
		for _, v := range report.Violations {
			t.Logf("  %s → %s: Gap=%f", v.Source, v.Target, v.Gap)
		}
	} else {
		v := report.Violations[0]
		if v.Source != "token.h" || v.Target != "parser.h" {
			t.Errorf("Expected violation token.h → parser.h, got %s → %s", v.Source, v.Target)
		}
		if v.Gap <= 0 {
			t.Errorf("Expected Gap > 0 for direction violation, got %f", v.Gap)
		}
	}
}

func TestAngleBracketIncludesIgnored(t *testing.T) {
	// Test AC 3: #include <...> should be ignored, only #include "..." counted
	cwd, _ := os.Getwd()
	projectRoot := filepath.Join(cwd, "../..")
	fixturePath := filepath.Join(projectRoot, "testdata/cpp-instability")
	lexerHPath := filepath.Join(fixturePath, "lexer.h")

	// Temporarily add #include <vector> to lexer.h
	originalContent, err := os.ReadFile(lexerHPath)
	if err != nil {
		t.Fatalf("Failed to read lexer.h: %v", err)
	}
	defer os.WriteFile(lexerHPath, originalContent, 0644)

	modifiedContent := "#ifndef LEXER_H\n#define LEXER_H\n\n#include <vector>\n#include \"token.h\"\n\n// Lexer class\n\n#endif\n"
	if err := os.WriteFile(lexerHPath, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify lexer.h: %v", err)
	}

	graph, err := BuildGraph([]string{fixturePath}, Options{})
	if err != nil {
		t.Fatalf("BuildGraph failed: %v", err)
	}

	// <vector> should NOT appear as a package
	for k := range graph.Packages {
		if strings.Contains(k, "vector") {
			t.Errorf("angle-bracket include <vector> should not appear in graph, but found key: %s", k)
		}
	}

	// lexer.h should still only have 1 outgoing edge (to token.h)
	lexerHStats, ok := graph.Packages["lexer.h"]
	if !ok {
		t.Fatal("lexer.h not in graph.Packages")
	}
	if lexerHStats.Ce != 1 {
		t.Errorf("lexer.h: expected Ce=1 (only token.h), got %d", lexerHStats.Ce)
	}
}

func TestIsolatedFilesNotInGraph(t *testing.T) {
	// Test AC 4: a file with 0 includes and 0 dependents should not appear in graph.Packages
	cwd, _ := os.Getwd()
	projectRoot := filepath.Join(cwd, "../..")
	fixturePath := filepath.Join(projectRoot, "testdata/cpp-instability")

	// Create a temporary isolated file
	isolatedPath := filepath.Join(fixturePath, "isolated.h")
	isolatedContent := "#ifndef ISOLATED_H\n#define ISOLATED_H\n\n// Isolated file, no includes, no dependents\n\n#endif\n"
	if err := os.WriteFile(isolatedPath, []byte(isolatedContent), 0644); err != nil {
		t.Fatalf("Failed to create isolated.h: %v", err)
	}
	defer os.Remove(isolatedPath)

	graph, err := BuildGraph([]string{fixturePath}, Options{})
	if err != nil {
		t.Fatalf("BuildGraph failed: %v", err)
	}

	// isolated.h should not appear in graph.Packages
	if _, ok := graph.Packages["isolated.h"]; ok {
		t.Error("isolated.h should not appear in graph.Packages")
	}

	// ViolationRate calculation should not crash (no division by zero)
	report, err := Check([]string{fixturePath}, 0, Options{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if report.TotalEdges == 0 {
		t.Errorf("Expected some edges from the fixture, got 0")
	}
}
