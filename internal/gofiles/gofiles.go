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

// Collect walks the provided paths and returns a list of all .go files found,
// skipping vendor/ and dot-directories. Returns two slices: files (the .go files)
// and skipped (any files/directories that could not be processed).
func Collect(paths []string) (files []string, skipped []SkippedFile) {
	files = []string{}
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
			// It's a file, add it directly
			files = append(files, path)
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
					files = append(files, filePath)
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

	return files, skipped
}
