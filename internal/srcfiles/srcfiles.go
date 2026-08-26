package srcfiles

import (
	"os"
	"path/filepath"
	"strings"
)

// ReadFile reads and returns the contents of a file.
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

type SkippedFile struct {
	File  string
	Error string
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

// matchesExtension returns true if filePath ends with any of the provided extensions.
func matchesExtension(filePath string, extensions []string) bool {
	for _, ext := range extensions {
		if strings.HasSuffix(filePath, ext) {
			return true
		}
	}
	return false
}

// dirWalker accumulates the source files found while walking a directory tree,
// split into files/excluded by excludePatterns.
type dirWalker struct {
	extensions      []string
	excludePatterns []string
	files           []string
	excluded        []string
}

// visitFile classifies a regular file encountered during the walk.
func (w *dirWalker) visitFile(filePath string) {
	if !matchesExtension(filePath, w.extensions) {
		return
	}
	if matchesExclude(filePath, w.excludePatterns) {
		w.excluded = append(w.excluded, filePath)
	} else {
		w.files = append(w.files, filePath)
	}
}

// visit is a fs.WalkDirFunc that prunes vendor/dot-directories and classifies
// every file matching the extensions it encounters.
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

// walkDir walks root and returns the source files matching extensions (split into files/excluded
// by excludePatterns), skipping vendor/ and dot-directories.
func walkDir(root string, extensions []string, excludePatterns []string) (files []string, excluded []string, err error) {
	w := &dirWalker{extensions: extensions, excludePatterns: excludePatterns}
	err = filepath.WalkDir(root, w.visit)
	return w.files, w.excluded, err
}

// Collect walks the provided paths and returns a list of all source files matching the given
// extensions, skipping vendor/ and dot-directories. Returns three slices: files (the matching files),
// excluded (files matching excludePatterns), and skipped (any files/directories that could not be processed).
func Collect(paths []string, extensions []string, excludePatterns []string) (files []string, excluded []string, skipped []SkippedFile) {
	files = []string{}
	excluded = []string{}
	skipped = []SkippedFile{}

	for _, path := range paths {
		pathFiles, pathExcluded, err := collectPath(path, extensions, excludePatterns)
		files = append(files, pathFiles...)
		excluded = append(excluded, pathExcluded...)
		if err != nil {
			skipped = append(skipped, SkippedFile{File: path, Error: err.Error()})
		}
	}

	return files, excluded, skipped
}

// collectPath collects the source files under a single path (file or directory).
// err is non-nil if path itself couldn't be stat'd or the directory walk failed.
func collectPath(path string, extensions []string, excludePatterns []string) (files, excluded []string, err error) {
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

	return walkDir(path, extensions, excludePatterns)
}
