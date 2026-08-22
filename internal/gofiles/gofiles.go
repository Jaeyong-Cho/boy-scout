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

// shouldSkipDir reports whether a subdirectory named name should be pruned
// from the walk (vendor/ and dot-directories, but never the root itself).
func shouldSkipDir(name string) bool {
	return (name == "vendor" || strings.HasPrefix(name, ".")) && name != "."
}

// dirWalker accumulates the .go files found while walking a directory tree,
// split into files/excluded by excludePatterns.
type dirWalker struct {
	excludePatterns []string
	files           []string
	excluded        []string
}

// visitFile classifies a regular file encountered during the walk.
func (w *dirWalker) visitFile(filePath string) {
	if !strings.HasSuffix(filePath, ".go") {
		return
	}
	if matchesExclude(filePath, w.excludePatterns) {
		w.excluded = append(w.excluded, filePath)
	} else {
		w.files = append(w.files, filePath)
	}
}

// visit is a fs.WalkDirFunc that prunes vendor/dot-directories and classifies
// every .go file it encounters.
func (w *dirWalker) visit(filePath string, d os.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if d.IsDir() {
		if shouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}
		return nil
	}
	w.visitFile(filePath)
	return nil
}

// walkDir walks root and returns the .go files found (split into files/excluded
// by excludePatterns), skipping vendor/ and dot-directories.
func walkDir(root string, excludePatterns []string) (files []string, excluded []string, err error) {
	w := &dirWalker{excludePatterns: excludePatterns}
	err = filepath.WalkDir(root, w.visit)
	return w.files, w.excluded, err
}

// Collect walks the provided paths and returns a list of all .go files found,
// skipping vendor/ and dot-directories. Returns three slices: files (the .go files),
// excluded (files matching excludePatterns), and skipped (any files/directories that could not be processed).
func Collect(paths []string, excludePatterns []string) (files []string, excluded []string, skipped []SkippedFile) {
	files = []string{}
	excluded = []string{}
	skipped = []SkippedFile{}

	for _, path := range paths {
		pathFiles, pathExcluded, err := collectPath(path, excludePatterns)
		files = append(files, pathFiles...)
		excluded = append(excluded, pathExcluded...)
		if err != nil {
			skipped = append(skipped, SkippedFile{File: path, Error: err.Error()})
		}
	}

	return files, excluded, skipped
}

// collectPath collects the .go files under a single path (file or directory).
// err is non-nil if path itself couldn't be stat'd or the directory walk failed.
func collectPath(path string, excludePatterns []string) (files, excluded []string, err error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, nil, err
	}

	if !stat.IsDir() {
		if matchesExclude(path, excludePatterns) {
			return nil, []string{path}, nil
		}
		return []string{path}, nil, nil
	}

	return walkDir(path, excludePatterns)
}
