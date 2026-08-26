/*
---
type: Source Code
title: filelen
description: Language-agnostic file line-count checker, dispatched per language (go/cpp) like funclen, reusing the existing srcfiles walker.
tags: [boy-scout, clean-code-checks]
timestamp: 2026-08-22T00:00:00+09:00
---
*/

package filelen

import (
	"bytes"
	"path/filepath"
	"strings"

	"boy-scout/internal/assertutil"
	"boy-scout/internal/srcfiles"
)

// commentSyntax defines line and block comment tokens for a language
type commentSyntax struct {
	Line       string
	BlockStart string
	BlockEnd   string
}

// commentSyntaxByExt maps file extensions to their comment syntax
var commentSyntaxByExt = map[string]commentSyntax{
	".go":   {Line: "//", BlockStart: "/*", BlockEnd: "*/"},
	".cpp":  {Line: "//", BlockStart: "/*", BlockEnd: "*/"},
	".h":    {Line: "//", BlockStart: "/*", BlockEnd: "*/"},
	".hpp":  {Line: "//", BlockStart: "/*", BlockEnd: "*/"},
	".ts":   {Line: "//", BlockStart: "/*", BlockEnd: "*/"},
	".tsx":  {Line: "//", BlockStart: "/*", BlockEnd: "*/"},
	".css":  {BlockStart: "/*", BlockEnd: "*/"},
	".html": {BlockStart: "<!--", BlockEnd: "-->"},
}

type Violation struct {
	File  string `json:"file"`
	Lines int    `json:"lines"`
	Limit int    `json:"limit"`
}

// SkippedFile is a type alias for srcfiles.SkippedFile
type SkippedFile = srcfiles.SkippedFile

type Options struct {
	ExcludeFiles []string
	Debug        bool
}

type Report struct {
	Violations    []Violation
	Skipped       []SkippedFile
	ExcludedFiles []string
}

// countPhysicalLines counts the number of physical lines in a byte slice.
// An empty file is 0 lines. A file is counted as N lines if it contains N newlines,
// plus 1 if there is content after the last newline without a trailing newline.
func countPhysicalLines(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	count := bytes.Count(content, []byte("\n"))
	// If the file doesn't end with a newline, add 1
	if content[len(content)-1] != '\n' {
		count++
	}
	return count
}

// countLoC counts the number of lines of code (non-blank, non-comment lines) in content.
// It uses a naive per-line classifier: blank lines and line-comment-prefixed lines don't count.
// Lines fully inside a block comment don't count. A line with trailing code + comment counts.
// ponytail: naive per-line comment/string detection, not a real lexer — a comment token inside
// a string literal can misclassify; upgrade to a real tokenizer if that shows up on real code
func countLoC(content []byte, ext string) int {
	if len(content) == 0 {
		return 0
	}

	syntax := commentSyntaxByExt[ext]

	// Split content by newlines, removing the trailing empty element if content ends in \n
	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	locCount := 0
	inBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// If we're inside a block comment, look for the end
		if inBlock {
			if syntax.BlockEnd != "" && strings.Contains(trimmed, syntax.BlockEnd) {
				inBlock = false
			}
			// Line is part of block comment, don't count
			continue
		}

		// Skip blank lines
		if trimmed == "" {
			continue
		}

		// Skip line comments
		if syntax.Line != "" && strings.HasPrefix(trimmed, syntax.Line) {
			continue
		}

		// Check for block comment start
		if syntax.BlockStart != "" && strings.Contains(trimmed, syntax.BlockStart) {
			// Check if block also ends on the same line
			if syntax.BlockEnd != "" && strings.Contains(trimmed, syntax.BlockEnd) {
				// Single-line block comment
				// Check if there's non-whitespace content after the block end
				blockEndIdx := strings.LastIndex(trimmed, syntax.BlockEnd)
				afterBlockEnd := strings.TrimSpace(trimmed[blockEndIdx+len(syntax.BlockEnd):])
				if afterBlockEnd == "" {
					// Just a block comment, no code
					continue
				}
				// There's code after the block comment, count it
				locCount++
			} else {
				// Block comment starts but doesn't end on this line
				inBlock = true
				// Line doesn't count as LoC
				continue
			}
		} else {
			// Normal line of code
			locCount++
		}
	}

	return locCount
}

// Check scans files matching the given extensions and reports those exceeding maxLines.
// It walks directories, skipping vendor/ and dot-directories, and respects exclude patterns.
func Check(paths []string, maxLines int, extensions []string, opts Options) (Report, error) {
	assertutil.Assertf(maxLines > 0, "maxLines must be positive, got %d", maxLines)
	assertutil.Assertf(len(extensions) > 0, "extensions must not be empty")

	report := Report{
		Violations:    []Violation{},
		Skipped:       []SkippedFile{},
		ExcludedFiles: []string{},
	}

	// Collect all files matching the extensions
	filesToCheck, excludedFiles, skipped := srcfiles.Collect(paths, extensions, opts.ExcludeFiles)
	report.Skipped = append(report.Skipped, skipped...)
	if opts.Debug {
		report.ExcludedFiles = append(report.ExcludedFiles, excludedFiles...)
	}

	for _, filePath := range filesToCheck {
		content, err := srcfiles.ReadFile(filePath)
		if err != nil {
			report.Skipped = append(report.Skipped, SkippedFile{File: filePath, Error: err.Error()})
			continue
		}

		physicalLines := countPhysicalLines(content)
		loc := countLoC(content, filepath.Ext(filePath))

		// Postcondition: LoC must not exceed physical lines
		assertutil.Assertf(loc <= physicalLines, "LoC (%d) must not exceed physical line count (%d) for %s", loc, physicalLines, filePath)

		if loc > maxLines {
			report.Violations = append(report.Violations, Violation{
				File:  filePath,
				Lines: loc,
				Limit: maxLines,
			})
		}
	}

	return report, nil
}
