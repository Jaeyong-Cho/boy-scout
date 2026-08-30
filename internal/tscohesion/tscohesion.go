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

// analyzeFile extracts classes from the source and checks their cohesion
func analyzeFile(filePath string, source []byte, report *Report) error {
	lines := strings.Split(string(source), "\n")

	// Simple regex-based class extraction for TypeScript
	classPattern := regexp.MustCompile(`^\s*(?:export\s+)?class\s+(\w+)\s*(?:\{|\s)`)

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		matches := classPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		className := matches[1]
		classStartLine := i + 1
		ci := &classInfo{
			Name:    className,
			File:    filePath,
			Line:    classStartLine,
			Fields:  make(map[string]bool),
			Methods: make(map[string]*methodInfo),
		}

		// Find the end of the class (closing brace)
		braceCount := 0
		classEndLine := len(lines)

		for j := i; j < len(lines); j++ {
			for _, c := range lines[j] {
				if c == '{' {
					braceCount++
				} else if c == '}' {
					braceCount--
				}
			}
			if braceCount == 0 && j > i {
				classEndLine = j
				break
			}
		}

		// Extract fields and methods from the class body (lines i+1 to classEndLine-1)
		for j := i + 1; j < classEndLine; j++ {
			classLineContent := strings.TrimSpace(lines[j])

			// Extract fields: look for "name: type;"
			if strings.Contains(classLineContent, ":") && !strings.Contains(classLineContent, "(") {
				parts := strings.Split(classLineContent, ":")
				if len(parts) > 0 {
					fieldName := strings.TrimSpace(parts[0])
					if isValidIdentifier(fieldName) && !isKeyword(fieldName) {
						ci.Fields[fieldName] = true
					}
				}
			}

			// Extract method signatures: look for "name(...)"
			if strings.Contains(classLineContent, "(") && !strings.HasPrefix(classLineContent, "//") {
				methodPattern := regexp.MustCompile(`(?:async\s+)?(?:public|private|protected|static)?\s*(?:async\s+)?(\w+)\s*\(`)
				methodMatches := methodPattern.FindStringSubmatch(classLineContent)

				if methodMatches != nil && len(methodMatches) > 1 {
					methodName := methodMatches[1]
					if !isKeyword(methodName) {
						// Extract method body
						methodBody := extractMethodBody(lines, j)
						if methodBody != "" {
							mi := &methodInfo{
								Name:   methodName,
								Fields: make(map[string]bool),
								Calls:  make(map[string]bool),
							}
							analyzeMethodBody(methodBody, ci.Fields, ci.Methods, mi)
							ci.Methods[methodName] = mi
						}
					}
				}
			}
		}

		// Score if >= 2 methods
		if len(ci.Methods) >= 2 {
			methods := make([]cohesion.Method, 0, len(ci.Methods))
			for methodName, mi := range ci.Methods {
				methods = append(methods, cohesion.Method{
					Name:   methodName,
					Fields: mi.Fields,
					Calls:  mi.Calls,
				})
			}

			score := cohesion.Compute(methods)
			if cohesion.Worst(score) != "good" {
				report.Violations = append(report.Violations, Violation{
					File:       filePath,
					Line:       ci.Line,
					Class:      ci.Name,
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

	return nil
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
