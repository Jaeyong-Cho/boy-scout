/*
---
type: Source Code
title: linelen
description: Language-agnostic line-length checker, flagging physical lines over a configurable character limit (default 100), with quote-stripping exemption.
tags: [boy-scout, clean-code-checks]
timestamp: 2026-08-26T00:00:00+09:00
---
*/

package linelen

import (
	"strings"
	"unicode/utf8"

	"boy-scout/internal/assertutil"
	"boy-scout/internal/srcfiles"
)

// quoteCharsByExt maps file extensions to their quote characters
var quoteCharsByExt = map[string][]rune{
	".go":   {'"', '\'', '`'},
	".cpp":  {'"', '\''},
	".h":    {'"', '\''},
	".hpp":  {'"', '\''},
	".ts":   {'"', '\'', '`'},
	".tsx":  {'"', '\'', '`'},
	".html": {'"', '\''},
	".css":  {'"', '\''},
}

type Violation struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Length int    `json:"length"`
	Limit  int    `json:"limit"`
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

// stripQuoted removes quoted substrings from a line, returning the result.
// It performs a single-pass rune scan, backslash-escape-aware for closing quote search.
// Unterminated quotes are left untouched.
// ponytail: naive per-char quote stripping, ${} interpolation not distinguished from literal text — upgrade to a real tokenizer if TS false-exemptions show up on real code.
func stripQuoted(line string, quoteChars []rune) string {
	runes := []rune(line)
	result := []rune{}
	i := 0

	for i < len(runes) {
		currentRune := runes[i]

		// Check if current rune is a quote character
		isQuote := false
		var quoteChar rune
		for _, q := range quoteChars {
			if currentRune == q {
				isQuote = true
				quoteChar = q
				break
			}
		}

		if isQuote {
			// Found an opening quote, scan for closing quote
			quoteStart := i
			i++ // skip opening quote
			found := false
			for i < len(runes) {
				if runes[i] == '\\' && i+1 < len(runes) {
					// Escape sequence, skip both the backslash and next char
					i += 2
					continue
				}
				if runes[i] == quoteChar {
					// Found closing quote
					i++ // skip closing quote
					found = true
					break
				}
				i++
			}
			// If not found, quote is unterminated - include it in result
			if !found {
				result = append(result, runes[quoteStart:]...)
				break // reached end of line
			}
			// If found, we skip the entire quoted span (it's already not in result)
		} else {
			// Regular character
			result = append(result, currentRune)
			i++
		}
	}

	stripped := string(result)
	assertutil.Assertf(utf8.RuneCountInString(stripped) <= utf8.RuneCountInString(line), "stripping quotes must not increase length: %d > %d for %q", utf8.RuneCountInString(stripped), utf8.RuneCountInString(line), line)
	return stripped
}

// Check scans files matching the given extensions and reports lines exceeding maxChars.
// It walks directories, skipping vendor/ and dot-directories, and respects exclude patterns.
func Check(paths []string, maxChars int, extensions []string, opts Options) (Report, error) {
	assertutil.Assertf(maxChars > 0, "maxChars must be positive, got %d", maxChars)

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

		if len(content) == 0 {
			continue
		}

		// Split content into lines
		lines := strings.Split(string(content), "\n")
		// Remove trailing empty element if content ends with \n
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}

		ext := ""
		if len(filePath) > 0 {
			// Extract extension from filename
			parts := strings.Split(filePath, ".")
			if len(parts) > 1 {
				ext = "." + parts[len(parts)-1]
			}
		}
		quoteChars := quoteCharsByExt[ext]

		for lineNum, line := range lines {
			charCount := utf8.RuneCountInString(line)
			if charCount > maxChars {
				// Check if the line would fit after stripping quotes
				stripped := stripQuoted(line, quoteChars)
				strippedCount := utf8.RuneCountInString(stripped)
				if strippedCount > maxChars {
					v := Violation{
						File:   filePath,
						Line:   lineNum + 1, // 1-indexed
						Length: charCount,
						Limit:  maxChars,
					}
					assertutil.Assertf(v.Length > maxChars, "appended violation does not exceed limit %d", maxChars)
					report.Violations = append(report.Violations, v)
				}
			}
		}
	}

	return report, nil
}
