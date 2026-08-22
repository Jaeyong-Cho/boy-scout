package gofiles

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SkippedFile struct {
	File  string
	Error string
}

func assertf(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}

// matchesExclude returns true if path matches any exclude pattern by full path or basename.
func matchesExclude(path string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	basename := filepath.Base(path)
	for _, p := range patterns {
		if match, _ := filepath.Match(p, path); match {
			return true
		}
		if match, _ := filepath.Match(p, basename); match {
			return true
		}
	}
	return false
}

// Collect walks the provided paths and returns a list of all .go files found,
// skipping vendor/ and dot-directories. Returns three slices: files (the .go files),
// excluded (files matching excludePatterns), and skipped (any files/directories that could not be processed).
func Collect(paths []string, excludePatterns []string) (files []string, excluded []string, skipped []SkippedFile) {
	files = []string{}
	excluded = []string{}
	skipped = []SkippedFile{}

	for _, path := range paths {
		stat, err := os.Stat(path)
		if err != nil {
			skipped = append(skipped, SkippedFile{
				File:  path,
				Error: err.Error(),
			})
			continue
		}

		if !stat.IsDir() {
			// It's a file, add it directly (or to excluded if it matches)
			if matchesExclude(path, excludePatterns) {
				excluded = append(excluded, path)
			} else {
				files = append(files, path)
			}
		} else {
			// It's a directory, walk it
			err := filepath.WalkDir(path, func(filePath string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}

				// Skip vendor/ and dot-directories (but not the root)
				if d.IsDir() {
					name := d.Name()
					// Don't skip the root directory itself, only subdirectories
					if (name == "vendor" || strings.HasPrefix(name, ".")) && name != "." {
						return filepath.SkipDir
					}
					return nil
				}

				// Check if it's a .go file
				if strings.HasSuffix(filePath, ".go") {
					if matchesExclude(filePath, excludePatterns) {
						excluded = append(excluded, filePath)
					} else {
						files = append(files, filePath)
					}
				}

				return nil
			})

			if err != nil {
				skipped = append(skipped, SkippedFile{
					File:  path,
					Error: err.Error(),
				})
			}
		}
	}

	return files, excluded, skipped
}
