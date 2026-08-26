package funcignore

import (
	"go/ast"
	"path"
	"strings"

	"boy-scout/internal/assertutil"
)

// Reason checks whether a function should be excluded for the given checker,
// based on name-pattern flags or a `// boy-scout:ignore[:checker[,checker...]]`
// doc-comment directive. Returns (excluded, reason) where reason is one of:
// "flag", "comment", or "" (not excluded). checker is the calling checker's
// own CLI subcommand name (e.g. "funclen", "crap").
func Reason(fn *ast.FuncDecl, patterns []string, checker string) (bool, string) {
	assertutil.Assertf(checker != "", "funcignore.Reason called with empty checker name")

	for _, p := range patterns {
		if match, _ := path.Match(p, fn.Name.Name); match {
			return true, "flag"
		}
	}

	if hasIgnoreComment(fn, checker) {
		return true, "comment"
	}

	return false, ""
}

func hasIgnoreComment(fn *ast.FuncDecl, checker string) bool {
	if fn.Doc == nil {
		return false
	}
	for _, comment := range fn.Doc.List {
		if commentIgnores(comment.Text, checker) {
			return true
		}
	}
	return false
}

// commentIgnores reports whether a single doc-comment line is a
// `// boy-scout:ignore` or `// boy-scout:ignore:checker[,checker...]` directive
// applying to checker.
func commentIgnores(commentText, checker string) bool {
	text := strings.TrimSpace(strings.TrimPrefix(commentText, "//"))
	if text == "boy-scout:ignore" {
		return true
	}
	rest, ok := strings.CutPrefix(text, "boy-scout:ignore:")
	if !ok {
		return false
	}
	for _, name := range strings.Split(rest, ",") {
		if strings.TrimSpace(name) == checker {
			return true
		}
	}
	return false
}
