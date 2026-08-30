package cppcohesion

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
	Name   string
	File   string
	Line   int
	Fields map[string]bool
	Methods map[string]*methodInfo
}

type methodInfo struct {
	Name   string
	Fields map[string]bool
	Calls  map[string]bool
}

// Check analyzes C++ files for class cohesion violations
// Using a simple pattern-based approach for inline class definitions
func Check(paths []string, opts Options) (Report, error) {
	report := Report{
		Violations: []Violation{},
		Skipped:    []SkippedFile{},
	}

	filesToCheck, _, skipped := srcfiles.Collect(paths, []string{".cpp", ".cc", ".cxx", ".h", ".hpp"}, opts.ExcludeFiles)
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

	// Simple regex-based class extraction for inline class definitions
	classPattern := regexp.MustCompile(`^\s*(class|struct)\s+(\w+)\s*\{`)

	for i, line := range lines {
		matches := classPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		className := matches[2]
		classStartLine := i + 1
		ci := &classInfo{
			Name:    className,
			File:    filePath,
			Line:    classStartLine,
			Fields:  make(map[string]bool),
			Methods: make(map[string]*methodInfo),
		}

		// Find the end of the class (closing brace)
		braceCount := 1
		classEndLine := i + 1

		for j := i + 1; j < len(lines); j++ {
			for _, c := range lines[j] {
				if c == '{' {
					braceCount++
				} else if c == '}' {
					braceCount--
				}
			}
			if braceCount == 0 {
				classEndLine = j
				break
			}
		}

		// Extract fields and methods from the class body
		classBody := strings.Join(lines[i+1:classEndLine], "\n")
		extractFieldsFromBody(classBody, ci)
		extractMethodsFromBody(classBody, ci)

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

// extractFieldsFromBody finds field declarations in class body
func extractFieldsFromBody(body string, ci *classInfo) {
	// Simple pattern: type name;
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		// Skip method declarations and public/private keywords
		if strings.Contains(line, "(") || strings.Contains(line, ")") {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "public:") ||
		   strings.HasPrefix(strings.TrimSpace(line), "private:") ||
		   strings.HasPrefix(strings.TrimSpace(line), "protected:") {
			continue
		}

		// Look for field declarations: type name;
		parts := strings.FieldsFunc(line, func(r rune) bool { return r == ';' || r == '=' || r == '{' })
		for i := 0; i < len(parts)-1; i++ {
			// Last word before ; is likely the field name
			trimmed := strings.TrimSpace(parts[i])
			words := strings.Fields(trimmed)
			if len(words) > 0 {
				fieldName := words[len(words)-1]
				// Simple heuristic: if it looks like a valid identifier and not a keyword
				if isValidIdentifier(fieldName) && !isKeyword(fieldName) {
					ci.Fields[fieldName] = true
				}
			}
		}
	}
}

// extractMethodsFromBody finds method definitions in class body
func extractMethodsFromBody(body string, ci *classInfo) {
	lines := strings.Split(body, "\n")
	i := 0
	for i < len(lines) {
		line := lines[i]

		// Skip access modifiers
		trimmed := strings.TrimSpace(line)
		if strings.HasSuffix(trimmed, ":") && (strings.HasPrefix(trimmed, "public") ||
			strings.HasPrefix(trimmed, "private") || strings.HasPrefix(trimmed, "protected")) {
			i++
			continue
		}

		// Look for method signature: type name() {
		methodPattern := regexp.MustCompile(`\b(?:void|int|bool|float|double|auto|\w+\*?)\s+(\w+)\s*\([^)]*\)\s*\{`)
		matches := methodPattern.FindStringSubmatch(line)
		if matches == nil {
			i++
			continue
		}

		methodName := matches[1]
		// Skip if it's actually a function (contains ::) or a keyword
		if strings.Contains(methodName, "::") || isKeyword(methodName) {
			i++
			continue
		}

		mi := &methodInfo{
			Name:   methodName,
			Fields: make(map[string]bool),
			Calls:  make(map[string]bool),
		}

		// Find the method body
		braceCount := 0
		methodBodyStart := i
		methodBodyEnd := i

		for j := i; j < len(lines); j++ {
			for _, c := range lines[j] {
				if c == '{' {
					braceCount++
				} else if c == '}' {
					braceCount--
				}
			}
			if braceCount > 0 {
				methodBodyStart = j + 1
				break
			}
			if braceCount == 0 && j > i {
				methodBodyEnd = j - 1
				break
			}
		}

		// Analyze method body for field/method references
		if methodBodyStart < len(lines) && methodBodyEnd <= len(lines) {
			methodBody := strings.Join(lines[methodBodyStart:methodBodyEnd], "\n")
			analyzeMethodBody(methodBody, ci.Fields, ci.Methods, mi)
		}

		ci.Methods[methodName] = mi
		i++
	}
}

// analyzeMethodBody extracts field touches and method calls
func analyzeMethodBody(body string, fields map[string]bool, methods map[string]*methodInfo, mi *methodInfo) {
	// Simple heuristic: look for identifiers that match known fields or methods
	words := regexp.MustCompile(`\b\w+\b`).FindAllString(body, -1)
	for _, word := range words {
		if fields[word] {
			mi.Fields[word] = true
		}
		if _, exists := methods[word]; exists {
			mi.Calls[word] = true
		}
	}
}

// isValidIdentifier checks if a string is a valid C++ identifier
func isValidIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	if !(s[0] == '_' || (s[0] >= 'a' && s[0] <= 'z') || (s[0] >= 'A' && s[0] <= 'Z')) {
		return false
	}
	for _, c := range s {
		if !(c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

// isKeyword checks if a string is a C++ keyword
func isKeyword(s string) bool {
	keywords := map[string]bool{
		"void": true, "int": true, "bool": true, "float": true, "double": true,
		"char": true, "long": true, "short": true, "auto": true, "const": true,
		"static": true, "virtual": true, "return": true, "if": true, "else": true,
		"for": true, "while": true, "do": true, "break": true, "continue": true,
		"class": true, "struct": true, "union": true, "enum": true, "public": true,
		"private": true, "protected": true, "namespace": true, "typedef": true,
		"template": true, "new": true, "delete": true, "nullptr": true, "true": true,
		"false": true, "this": true,
	}
	return keywords[s]
}
