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
	"fmt"

	"boy-scout/internal/srcfiles"
)

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
	Violations   []Violation
	Skipped      []SkippedFile
	ExcludedFiles []string
}

func assertf(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}

// countLines counts the number of lines in a byte slice.
// An empty file is 0 lines. A file is counted as N lines if it contains N newlines,
// plus 1 if there is content after the last newline without a trailing newline.
func countLines(content []byte) int {
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

// Check scans files matching the given extensions and reports those exceeding maxLines.
// It walks directories, skipping vendor/ and dot-directories, and respects exclude patterns.
func Check(paths []string, maxLines int, extensions []string, opts Options) (Report, error) {
	assertf(maxLines > 0, "maxLines must be positive, got %d", maxLines)
	assertf(len(extensions) > 0, "extensions must not be empty")

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

		lineCount := countLines(content)
		if lineCount > maxLines {
			report.Violations = append(report.Violations, Violation{
				File:  filePath,
				Lines: lineCount,
				Limit: maxLines,
			})
		}
	}

	return report, nil
}
