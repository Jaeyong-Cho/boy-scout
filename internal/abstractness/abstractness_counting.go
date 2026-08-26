package abstractness

import (
	"go/ast"
	"go/parser"
	"go/token"

	"boy-scout/internal/assertutil"
)

// surfaceRatio computes (# exported top-level declarations) / (# total top-level declarations).
// Top-level declarations counted: FuncDecl (including methods) and each spec in GenDecl.
// Returns the ratio and any parse errors encountered.
// Pre: files must be non-empty and contain at least one computable declaration (asserted).
func surfaceRatio(files []string) (ratio float64, err error) {
	var exportedCount, totalCount int

	for _, filePath := range files {
		exp, tot := countDeclarationsInFile(filePath)
		exportedCount += exp
		totalCount += tot
	}

	assertutil.Assertf(totalCount > 0, "surfaceRatio called on package with 0 declarations; this should not happen for a Zone-of-Pain candidate")
	return float64(exportedCount) / float64(totalCount), nil
}

// countDeclarationsInFile counts exported and total top-level declarations in a file.
func countDeclarationsInFile(filePath string) (exported, total int) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return 0, 0 // skip unparsable files
	}

	for _, decl := range file.Decls {
		e, t := processDeclForCount(decl)
		exported += e
		total += t
	}

	return exported, total
}

// processDeclForCount extracts counts from a single declaration.
func processDeclForCount(decl ast.Decl) (exported, total int) {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		total = 1
		if ast.IsExported(d.Name.Name) {
			exported = 1
		}
	case *ast.GenDecl:
		if isCountableGenDecl(d) {
			exported, total = addSpecCounts(d.Specs, 0, 0)
		}
	}
	return exported, total
}

// addSpecCounts accumulates counts from GenDecl specs.
func addSpecCounts(specs []ast.Spec, exported, total int) (int, int) {
	for _, spec := range specs {
		total++
		if isExportedSpec(spec) {
			exported++
		}
	}
	return exported, total
}

// isCountableGenDecl returns true if the GenDecl should have its specs counted.
func isCountableGenDecl(d *ast.GenDecl) bool {
	return d.Tok == token.TYPE || d.Tok == token.VAR || d.Tok == token.CONST
}

// isExportedSpec checks if a spec is exported by extracting its name.
func isExportedSpec(spec ast.Spec) bool {
	var name string
	switch s := spec.(type) {
	case *ast.TypeSpec:
		name = s.Name.Name
	case *ast.ValueSpec:
		if len(s.Names) > 0 {
			name = s.Names[0].Name
		}
	}
	return name != "" && ast.IsExported(name)
}

// countExportedTypes scans the given .go files and returns the count of exported
// interface and struct type declarations. Returns (interfaces, structs).
func countExportedTypes(files []string) (interfaces, structs int, skipped []SkippedFile) {
	for _, filePath := range files {
		iCnt, sCnt, fileSkipped, err := countTypesInFile(filePath)
		if err != nil {
			skipped = append(skipped, SkippedFile{File: filePath, Error: err.Error()})
			continue
		}
		interfaces += iCnt
		structs += sCnt
		skipped = append(skipped, fileSkipped...)
	}

	return interfaces, structs, skipped
}

// countTypesInFile counts exported interface and struct types in a file.
func countTypesInFile(filePath string) (int, int, []SkippedFile, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return 0, 0, nil, err
	}

	interfaces, structs := countTypesInDecls(file.Decls)
	return interfaces, structs, nil, nil
}

// countTypesInDecls counts exported interface and struct types in a list of declarations.
func countTypesInDecls(decls []ast.Decl) (int, int) {
	var interfaces, structs int
	for _, decl := range decls {
		// Only process type declarations
		if g, ok := decl.(*ast.GenDecl); ok && g.Tok == token.TYPE {
			iCnt, sCnt := countTypesInGenDecl(g)
			interfaces += iCnt
			structs += sCnt
		}
	}
	return interfaces, structs
}

// countTypesInGenDecl counts exported interface and struct types in a generic declaration.
func countTypesInGenDecl(g *ast.GenDecl) (int, int) {
	var interfaces, structs int
	for _, spec := range g.Specs {
		// Only process type specs (not import specs)
		if ts, ok := spec.(*ast.TypeSpec); ok {
			iCnt, sCnt := countTypeKinds(ts)
			interfaces += iCnt
			structs += sCnt
		}
	}
	return interfaces, structs
}

// countTypeKinds counts interface and struct types in a TypeSpec if exported.
func countTypeKinds(typeSpec *ast.TypeSpec) (int, int) {
	// Only count exported (capitalized) names
	if !typeSpec.Name.IsExported() {
		return 0, 0
	}

	// Check if it's an interface or struct type
	switch typeSpec.Type.(type) {
	case *ast.InterfaceType:
		return 1, 0
	case *ast.StructType:
		return 0, 1
	}

	// Ignore other type kinds (aliases, func types, etc.)
	return 0, 0
}
