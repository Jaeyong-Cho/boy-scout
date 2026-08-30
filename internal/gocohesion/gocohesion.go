package gocohesion

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"

	"boy-scout/internal/cohesion"
	"boy-scout/internal/srcfiles"
)

type Violation struct {
	File      string  `json:"file"`
	Line      int     `json:"line"`
	Class     string  `json:"class"`
	LCOM4     int     `json:"lcom4"`
	LCOM4Level string `json:"lcom4Level"`
	TCC       float64 `json:"tcc"`
	TCCLevel  string  `json:"tccLevel"`
	LCC       float64 `json:"lcc"`
	LCCLevel  string  `json:"lccLevel"`
}

type SkippedFile = srcfiles.SkippedFile

type Options struct {
	ExcludeFiles []string
	Debug        bool
}

type Report struct {
	Violations []Violation   `json:"violations"`
	Skipped    []SkippedFile `json:"skipped"`
}

// structInfo holds parsed info about a struct: its fields and methods
type structInfo struct {
	Name  string
	File  string
	Line  int
	Fields map[string]bool // field names
	Methods map[string]*methodInfo
}

type methodInfo struct {
	Name  string
	Fields map[string]bool // fields this method touches
	Calls map[string]bool  // other methods this method calls
}

// WorstLevel returns the worst of the three levels in a Violation.
func WorstLevel(v Violation) string {
	levels := []string{v.LCOM4Level, v.TCCLevel, v.LCCLevel}
	worst := "good"
	for _, level := range levels {
		if level == "danger" {
			worst = "danger"
		} else if level == "warning" && worst != "danger" {
			worst = "warning"
		}
	}
	return worst
}

// Check analyzes Go files for struct cohesion violations.
func Check(paths []string, opts Options) (Report, error) {
	report := Report{
		Violations: []Violation{},
		Skipped:    []SkippedFile{},
	}

	filesToCheck, _, skipped := srcfiles.Collect(paths, []string{".go"}, opts.ExcludeFiles)
	report.Skipped = append(report.Skipped, skipped...)

	// Group files by directory (same package)
	filesByDir := make(map[string][]string)
	for _, f := range filesToCheck {
		dir := filepath.Dir(f)
		filesByDir[dir] = append(filesByDir[dir], f)
	}

	// Process each directory
	for _, files := range filesByDir {
		structsByName := make(map[string]*structInfo)

		// Parse all files in this directory and collect structs
		for _, filePath := range files {
			if err := parseStructsInFile(filePath, structsByName); err != nil {
				report.Skipped = append(report.Skipped, srcfiles.SkippedFile{File: filePath, Error: err.Error()})
				continue
			}
		}

		// Parse all files again and collect methods
		for _, filePath := range files {
			if err := parseMethodsInFile(filePath, structsByName); err != nil {
				// Already reported as skipped in the first pass, don't double-report
				continue
			}
		}

		// Score each struct
		for _, si := range structsByName {
			if len(si.Methods) < 2 {
				continue // Skip structs with < 2 methods
			}

			// Build cohesion.Method slice
			methods := make([]cohesion.Method, 0, len(si.Methods))
			methodNameToIdx := make(map[string]int)
			for mName, mi := range si.Methods {
				methodNameToIdx[mName] = len(methods)
				methods = append(methods, cohesion.Method{
					Name:   mName,
					Fields: mi.Fields,
					Calls:  mi.Calls,
				})
			}

			score := cohesion.Compute(methods)
			if cohesion.Worst(score) != "good" {
				report.Violations = append(report.Violations, Violation{
					File:       si.File,
					Line:       si.Line,
					Class:      si.Name,
					LCOM4:      score.LCOM4,
					LCOM4Level: score.LCOM4Level,
					TCC:        score.TCC,
					TCCLevel:   score.TCCLevel,
					LCC:        score.LCC,
					LCCLevel:   score.LCCLevel,
				})
			}
		}
	}

	return report, nil
}

// parseStructsInFile parses filePath and extracts all top-level struct types,
// storing them in structsByName.
func parseStructsInFile(filePath string, structsByName map[string]*structInfo) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}

		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Extract field names
			fields := make(map[string]bool)
			for _, f := range st.Fields.List {
				for _, name := range f.Names {
					fields[name.Name] = true
				}
			}

			structsByName[ts.Name.Name] = &structInfo{
				Name:    ts.Name.Name,
				File:    filePath,
				Line:    fset.Position(ts.Pos()).Line,
				Fields:  fields,
				Methods: make(map[string]*methodInfo),
			}
		}
	}

	return nil
}

// parseMethodsInFile parses filePath and extracts methods for known structs,
// recording field touches and method calls.
func parseMethodsInFile(filePath string, structsByName map[string]*structInfo) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Recv == nil {
			continue
		}

		// Get receiver type name (strip leading *)
		recvTypeName := getReceiverType(fd.Recv)
		if recvTypeName == "" {
			continue
		}

		si, ok := structsByName[recvTypeName]
		if !ok {
			continue
		}

		// Analyze method body for field touches and method calls
		fields := make(map[string]bool)
		calls := make(map[string]bool)

		if fd.Body != nil {
			analyzeMethodBody(fd.Body, recvTypeName, si.Fields, fields, calls)
		}

		si.Methods[fd.Name.Name] = &methodInfo{
			Name:   fd.Name.Name,
			Fields: fields,
			Calls:  calls,
		}
	}

	return nil
}

// getReceiverType extracts the struct type name from a receiver list.
// Returns "" if not a valid receiver.
func getReceiverType(recv *ast.FieldList) string {
	if recv.NumFields() != 1 {
		return ""
	}
	f := recv.List[0]
	if len(f.Names) == 0 {
		return ""
	}

	// Handle *Type or Type
	var expr ast.Expr = f.Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}

	id, ok := expr.(*ast.Ident)
	if !ok {
		return ""
	}
	return id.Name
}

// analyzeMethodBody walks the method body's AST, recording field accesses and method calls.
func analyzeMethodBody(body *ast.BlockStmt, recvTypeName string, knownFields map[string]bool, fields, calls map[string]bool) {
	// Find receiver identifier (usually first parameter)
	var recvIdent string
	// We'll detect it from SelectorExpr usage

	ast.Inspect(body, func(n ast.Node) bool {
		if n == nil {
			return true
		}

		// Look for SelectorExpr where the base is the receiver
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		id, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}

		// We assume the selector is on the receiver (usually named 'f', 's', 'c', etc.)
		// Set recvIdent on first usage
		if recvIdent == "" {
			recvIdent = id.Name
		}

		// If this selector is on the receiver
		if id.Name == recvIdent {
			fieldOrMethodName := sel.Sel.Name

			// Check if it's a known field
			if knownFields[fieldOrMethodName] {
				fields[fieldOrMethodName] = true
			} else {
				// Assume it's a method call (optimistic: could be a field that's also a method)
				calls[fieldOrMethodName] = true
			}
		}

		return true
	})
}
