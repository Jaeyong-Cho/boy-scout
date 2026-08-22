package funcignore

import (
	"fmt"
	"go/ast"
	"path"
	"strings"
)

func assertf(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}

// Reason checks whether a function should be excluded for the given checker,
// based on name-pattern flags or a `// gardener:ignore[:checker[,checker...]]`
// doc-comment directive. Returns (excluded, reason) where reason is one of:
// "flag", "comment", or "" (not excluded). checker is the calling checker's
// own CLI subcommand name (e.g. "funclen", "crap").
func Reason(fn *ast.FuncDecl, patterns []string, checker string) (bool, string) {
	assertf(checker != "", "funcignore.Reason called with empty checker name")

	for _, p := range patterns {
		if match, _ := path.Match(p, fn.Name.Name); match {
			return true, "flag"
		}
	}

	if fn.Doc != nil {
		for _, comment := range fn.Doc.List {
			text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			if text == "gardener:ignore" {
				return true, "comment"
			}
			if rest, ok := strings.CutPrefix(text, "gardener:ignore:"); ok {
				for _, name := range strings.Split(rest, ",") {
					if strings.TrimSpace(name) == checker {
						return true, "comment"
					}
				}
			}
		}
	}

	return false, ""
}
