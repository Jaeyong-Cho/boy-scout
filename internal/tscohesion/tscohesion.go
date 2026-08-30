package tscohesion

import (
	"regexp"
	"strings"

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

type classInfo struct {
	Name    string
	File    string
	Line    int
	Fields  map[string]bool
	Methods map[string]*methodInfo
}

type methodInfo struct {
	Name   string
	Fields map[string]bool
	Calls  map[string]bool
}

// Check analyzes TypeScript files for class cohesion violations
func Check(paths []string, opts Options) (Report, error) {
	report := Report{
		Violations: []Violation{},
		Skipped:    []SkippedFile{},
	}

	filesToCheck, _, skipped := srcfiles.Collect(paths, []string{".ts", ".tsx"}, opts.ExcludeFiles)
	report.Skipped = append(report.Skipped, skipped...)

	for _, filePath := range filesToCheck {
		data, err := srcfiles.ReadFile(filePath)
		if err != nil {
			report.Skipped = append(report.Skipped, srcfiles.SkippedFile{File: filePath, Error: err.Error()})
			continue
		}

		// Analyze the file
		if err := analyzeFile(filePath, data, &report); err != nil {
			report.Skipped = append(report.Skipped, srcfiles.SkippedFile{File: filePath, Error: err.Error()})
			continue
		}
	}

	return report, nil
}

// classPattern matches a TypeScript class declaration line.
var classPattern = regexp.MustCompile(`^\s*(?:export\s+)?class\s+(\w+)\s*(?:\{|\s)`)

// analyzeFile extracts classes from the source and checks their cohesion
func analyzeFile(filePath string, source []byte, report *Report) error {
	lines := strings.Split(string(source), "\n")

	for i := 0; i < len(lines); i++ {
		matches := classPattern.FindStringSubmatch(lines[i])
		if matches == nil {
			continue
		}

		ci := &classInfo{
			Name:    matches[1],
			File:    filePath,
			Line:    i + 1,
			Fields:  make(map[string]bool),
			Methods: make(map[string]*methodInfo),
		}

		classEndLine := findClassEndLine(lines, i)
		extractClassMembers(lines, i+1, classEndLine, ci)

		if v := scoreClass(ci); v != nil {
			report.Violations = append(report.Violations, *v)
		}
	}

	return nil
}

// braceDelta returns the net change in brace depth contributed by line
// (count of '{' minus count of '}').
func braceDelta(line string) int {
	delta := 0
	for _, c := range line {
		if c == '{' {
			delta++
		} else if c == '}' {
			delta--
		}
	}
	return delta
}

// findClassEndLine returns the line index of the closing brace that matches
// the class opening at startLine, by counting braces.
func findClassEndLine(lines []string, startLine int) int {
	braceCount := 0
	for j := startLine; j < len(lines); j++ {
		braceCount += braceDelta(lines[j])
		if braceCount == 0 && j > startLine {
			return j
		}
	}
	return len(lines)
}

// extractField records line as a class field on ci if it looks like a
// TypeScript field declaration ("name: type;").
func extractField(line string, ci *classInfo) {
	if !strings.Contains(line, ":") || strings.Contains(line, "(") {
		return
	}
	parts := strings.Split(line, ":")
	if len(parts) == 0 {
		return
	}
	fieldName := strings.TrimSpace(parts[0])
	if isValidIdentifier(fieldName) && !isKeyword(fieldName) {
		ci.Fields[fieldName] = true
	}
}

// matchMethodName returns the method name declared on line and true, or
// ("", false) if line doesn't look like a TypeScript method signature.
func matchMethodName(line string, methodPattern *regexp.Regexp) (string, bool) {
	if !strings.Contains(line, "(") || strings.HasPrefix(line, "//") {
		return "", false
	}
	methodMatches := methodPattern.FindStringSubmatch(line)
	if methodMatches == nil || len(methodMatches) <= 1 || isKeyword(methodMatches[1]) {
		return "", false
	}
	return methodMatches[1], true
}

// extractMethod records line (at lines[j]) as a class method on ci if it
// looks like a TypeScript method signature ("name(...)").
func extractMethod(lines []string, j int, line string, pattern *regexp.Regexp, ci *classInfo) {
	methodName, ok := matchMethodName(line, pattern)
	if !ok {
		return
	}

	methodBody := extractMethodBody(lines, j)
	if methodBody == "" {
		return
	}
	mi := &methodInfo{
		Name:   methodName,
		Fields: make(map[string]bool),
		Calls:  make(map[string]bool),
	}
	analyzeMethodBody(methodBody, ci.Fields, ci.Methods, mi)
	ci.Methods[methodName] = mi
}

// extractClassMembers scans lines[start:end) (a class body) and populates
// ci.Fields and ci.Methods with the fields and methods it finds.
func extractClassMembers(lines []string, start, end int, ci *classInfo) {
	methodPattern := regexp.MustCompile(`(?:async\s+)?(?:public|private|protected|static)?\s*(?:async\s+)?(\w+)\s*\(`)

	for j := start; j < end; j++ {
		line := strings.TrimSpace(lines[j])
		extractField(line, ci)
		extractMethod(lines, j, line, methodPattern, ci)
	}
}

// scoreClass computes the cohesion score for ci and returns the resulting
// Violation, or nil if ci has too few methods to score or isn't a violation.
func scoreClass(ci *classInfo) *Violation {
	if len(ci.Methods) < 2 {
		return nil
	}

	methods := make([]cohesion.Method, 0, len(ci.Methods))
	for methodName, mi := range ci.Methods {
		methods = append(methods, cohesion.Method{
			Name:   methodName,
			Fields: mi.Fields,
			Calls:  mi.Calls,
		})
	}

	score := cohesion.Compute(methods)
	if cohesion.Worst(score) == "good" {
		return nil
	}
	return &Violation{
		File:       ci.File,
		Line:       ci.Line,
		Class:      ci.Name,
		LCOM4:      score.LCOM4,
		LCOM4Level: score.LCOM4Level,
		TCC:        score.TCC,
		TCCLevel:   score.TCCLevel,
		LCC:        score.LCC,
		LCCLevel:   score.LCCLevel,
	}
}

// extractMethodBody finds the body of a method starting at line startLine
func extractMethodBody(lines []string, startLine int) string {
	if startLine >= len(lines) {
		return ""
	}

	line := lines[startLine]
	braceCount := 0
	methodBody := []string{}

	// Count opening braces on the first line
	for _, c := range line {
		if c == '{' {
			braceCount++
		}
	}

	if braceCount == 0 {
		return ""
	}

	// Collect lines until closing brace
	for j := startLine + 1; j < len(lines) && braceCount > 0; j++ {
		methodBody = append(methodBody, lines[j])
		for _, c := range lines[j] {
			if c == '{' {
				braceCount++
			} else if c == '}' {
				braceCount--
			}
		}
	}

	return strings.Join(methodBody, "\n")
}

// analyzeMethodBody extracts field touches and method calls via "this."
func analyzeMethodBody(body string, fields map[string]bool, methods map[string]*methodInfo, mi *methodInfo) {
	// Look for this.fieldName or this.methodName()
	thisPattern := regexp.MustCompile(`this\.(\w+)`)
	matches := thisPattern.FindAllStringSubmatch(body, -1)

	for _, match := range matches {
		if len(match) > 1 {
			name := match[1]
			if fields[name] {
				mi.Fields[name] = true
			}
			if _, exists := methods[name]; exists {
				mi.Calls[name] = true
			}
		}
	}
}

// isValidIdentifier checks if a string is a valid TypeScript identifier
func isValidIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	if !(s[0] == '_' || s[0] == '$' || (s[0] >= 'a' && s[0] <= 'z') || (s[0] >= 'A' && s[0] <= 'Z')) {
		return false
	}
	for _, c := range s {
		if !(c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

// isKeyword checks if a string is a TypeScript keyword
func isKeyword(s string) bool {
	keywords := map[string]bool{
		"abstract": true, "any": true, "as": true, "boolean": true, "break": true,
		"case": true, "catch": true, "class": true, "const": true, "continue": true,
		"debugger": true, "declare": true, "default": true, "delete": true, "do": true,
		"else": true, "enum": true, "export": true, "extends": true, "false": true,
		"finally": true, "for": true, "from": true, "function": true, "global": true,
		"if": true, "implements": true, "import": true, "in": true, "instanceof": true,
		"interface": true, "is": true, "keyof": true, "let": true, "module": true,
		"namespace": true, "never": true, "new": true, "null": true, "number": true,
		"of": true, "package": true, "private": true, "protected": true, "public": true,
		"readonly": true, "require": true, "return": true, "static": true, "string": true,
		"super": true, "switch": true, "symbol": true, "this": true, "throw": true,
		"true": true, "try": true, "type": true, "typeof": true, "undefined": true,
		"var": true, "void": true, "while": true, "with": true, "yield": true,
	}
	return keywords[s]
}
