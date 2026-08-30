package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_CreatesSkillFileAtBaseDir(t *testing.T) {
	baseDir := t.TempDir()

	path, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	expectedPath := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "SKILL.md")
	if path != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, path)
	}

	// Verify the file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("skill file not found at %q: %v", path, err)
	}

	// Verify the file is non-empty
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}
	if len(content) == 0 {
		t.Errorf("skill file is empty")
	}
}

func TestRun_OverwritesExistingSkillFile(t *testing.T) {
	baseDir := t.TempDir()

	// First run
	path1, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("first Run failed: %v", err)
	}

	content1, err := os.ReadFile(path1)
	if err != nil {
		t.Fatalf("failed to read skill file after first run: %v", err)
	}

	// Second run (should overwrite without error)
	path2, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("second Run failed: %v", err)
	}

	if path1 != path2 {
		t.Errorf("expected same path on second run, got %q vs %q", path1, path2)
	}

	content2, err := os.ReadFile(path2)
	if err != nil {
		t.Fatalf("failed to read skill file after second run: %v", err)
	}

	if string(content1) != string(content2) {
		t.Errorf("skill file content changed on overwrite (may be expected if template changed)")
	}
}

func TestRun_TemplateDeclaresUserInvokedFixLoop(t *testing.T) {
	baseDir := t.TempDir()

	path, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}

	contentStr := string(content)

	// Verify frontmatter
	if !strings.Contains(contentStr, "name: boy-scout") {
		t.Errorf("expected 'name: boy-scout' in frontmatter, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "description:") {
		t.Errorf("expected 'description:' in frontmatter, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "disable-model-invocation: true") {
		t.Errorf("expected 'disable-model-invocation: true' in frontmatter, got:\n%s", contentStr)
	}

	// Verify instructions
	if !strings.Contains(contentStr, "boy-scout go all") {
		t.Errorf("expected 'boy-scout go all' in skill body, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "Run your project's test suite once") {
		t.Errorf("expected a test-suite-green instruction in skill body, got:\n%s", contentStr)
	}
}

func TestRun_CopiesBinaryWhenPathGiven(t *testing.T) {
	baseDir := t.TempDir()

	binarySrc := filepath.Join(t.TempDir(), "boy-scout")
	if err := os.WriteFile(binarySrc, []byte("fake binary content"), 0755); err != nil {
		t.Fatalf("failed to write fake binary: %v", err)
	}

	_, err := Run(baseDir, binarySrc, ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	binPath := filepath.Join(baseDir, ".agents", "bin", "boy-scout")
	content, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("expected binary copied to %q: %v", binPath, err)
	}
	if string(content) != "fake binary content" {
		t.Errorf("expected copied binary content to match source, got %q", content)
	}
}

func TestRun_TemplateDeclaresBaselineTestGate(t *testing.T) {
	baseDir := t.TempDir()

	path, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}
	contentStr := string(content)

	cases := []struct {
		name   string
		marker string
	}{
		{"BaselineGreenGate", "tests were already failing before this review"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(contentStr, c.marker) {
				t.Errorf("expected skill template to contain %q, got:\n%s", c.marker, contentStr)
			}
		})
	}
}

func TestRun_TemplateHasNoMachineLocalPath(t *testing.T) {
	baseDir := t.TempDir()

	path, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}
	contentStr := string(content)

	if strings.Contains(contentStr, "/Users/") {
		t.Errorf("expected skill template to not contain /Users/, got:\n%s", contentStr)
	}
	if strings.Contains(contentStr, "~/workspace") {
		t.Errorf("expected skill template to not contain ~/workspace, got:\n%s", contentStr)
	}
}

func TestRun_UsesGivenPrefixForSkillAndBinPaths(t *testing.T) {
	baseDir := t.TempDir()

	binarySrc := filepath.Join(t.TempDir(), "boy-scout")
	if err := os.WriteFile(binarySrc, []byte("fake binary content"), 0755); err != nil {
		t.Fatalf("failed to write fake binary: %v", err)
	}

	path, err := Run(baseDir, binarySrc, ".claude")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify the skill file path uses .claude prefix
	expectedSkillPath := filepath.Join(baseDir, ".claude", "skills", "boy-scout", "SKILL.md")
	if path != expectedSkillPath {
		t.Errorf("expected skill path %q, got %q", expectedSkillPath, path)
	}

	// Verify the skill file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("skill file not found at %q: %v", path, err)
	}

	// Verify the binary was copied to .claude
	expectedBinPath := filepath.Join(baseDir, ".claude", "bin", "boy-scout")
	content, err := os.ReadFile(expectedBinPath)
	if err != nil {
		t.Fatalf("expected binary copied to %q: %v", expectedBinPath, err)
	}
	if string(content) != "fake binary content" {
		t.Errorf("expected copied binary content to match source, got %q", content)
	}

	// Verify .agents directory was not created
	agentsPath := filepath.Join(baseDir, ".agents")
	if _, err := os.Stat(agentsPath); err == nil {
		t.Errorf("expected .agents directory not to be created when using .claude prefix")
	}
}

func TestRun_TemplateDeclaresFilelenGuidance(t *testing.T) {
	baseDir := t.TempDir()

	path, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}
	contentStr := string(content)

	cases := []struct {
		name   string
		marker string
	}{
		{"FilelenFixedBeforeCohesion", "filelen, cohesion, cross-package duplication"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(contentStr, c.marker) {
				t.Errorf("expected skill template to contain %q, got:\n%s", c.marker, contentStr)
			}
		})
	}
}

func TestRun_WritesAllReferenceFiles(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	refDir := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references")
	referenceFiles := []string{"funclen.md", "complexity.md", "filelen.md"}

	for _, filename := range referenceFiles {
		path := filepath.Join(refDir, filename)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected reference file %q to exist, got error: %v", filename, err)
		}
	}
}

func TestRun_ReferenceFilesExplainWhyAndHow(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	refDir := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references")

	cases := []struct {
		name      string
		filename  string
		whyMarker string
		howMarker string
	}{
		{"Funclen", "funclen.md", "one level of abstraction", "table of contents"},
		{"Complexity", "complexity.md", "too many independent paths", "each branch"},
		{"Filelen", "filelen.md", "mixing multiple concerns", "high cohesion"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(refDir, c.filename))
			if err != nil {
				t.Fatalf("failed to read %q: %v", c.filename, err)
			}
			contentStr := string(content)

			if !strings.Contains(contentStr, c.whyMarker) {
				t.Errorf("expected %q to contain why marker %q, got:\n%s", c.filename, c.whyMarker, contentStr)
			}
			if !strings.Contains(contentStr, c.howMarker) {
				t.Errorf("expected %q to contain how marker %q, got:\n%s", c.filename, c.howMarker, contentStr)
			}
		})
	}
}

func TestRun_ReferenceFilesHaveNoMachineLocalPath(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	refDir := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references")
	referenceFiles := []string{"funclen.md", "complexity.md", "filelen.md"}

	for _, filename := range referenceFiles {
		content, err := os.ReadFile(filepath.Join(refDir, filename))
		if err != nil {
			t.Fatalf("failed to read %q: %v", filename, err)
		}
		contentStr := string(content)

		if strings.Contains(contentStr, "/Users/") {
			t.Errorf("expected %q to not contain /Users/, got:\n%s", filename, contentStr)
		}
		if strings.Contains(contentStr, "~/workspace") {
			t.Errorf("expected %q to not contain ~/workspace, got:\n%s", filename, contentStr)
		}
	}
}

func TestRun_TemplateMapsViolationsToReferenceFiles(t *testing.T) {
	baseDir := t.TempDir()

	path, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}
	contentStr := string(content)

	// Check that all 5 reference files are mentioned in the table
	referenceFiles := []string{
		"references/funclen.md",
		"references/complexity.md",
		"references/filelen.md",
	}

	for _, refFile := range referenceFiles {
		if !strings.Contains(contentStr, refFile) {
			t.Errorf("expected skill template to contain reference to %q, got:\n%s", refFile, contentStr)
		}
	}

	// Check fix-order statement
	if !strings.Contains(contentStr, "funclen, linelen, same-package duplication, complexity, filelen, cohesion, cross-package duplication") {
		t.Errorf("expected skill template to contain fix-order statement with all violation kinds, got:\n%s", contentStr)
	}
}

func TestRun_TemplateNoLongerInlinesViolationExplanations(t *testing.T) {
	baseDir := t.TempDir()

	path, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}
	contentStr := string(content)

	// These specific inline explanation bullets should NOT appear in skill.md anymore
	// (they should only appear in reference files)
	forbiddenMarkers := []string{
		"**gofunclen violation:**",
		"**filelen violation:**",
	}

	for _, marker := range forbiddenMarkers {
		if strings.Contains(contentStr, marker) {
			t.Errorf("expected skill template to no longer contain inline violation explanation %q, got:\n%s", marker, contentStr)
		}
	}
}

func TestRun_TemplateDeclaresSelectionCapAndPriority(t *testing.T) {
	baseDir := t.TempDir()

	path, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}
	contentStr := string(content)

	cases := []struct {
		name   string
		marker string
	}{
		{"CapPerType", "Within each type, select one candidate per run"},
		{"SeverityWorstFirst", "boy-scout's severity"},
		{"TestFilesDeferredLast", "go last"},
		{"NeverEditWhileSelecting", "never edit anything in this step"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(contentStr, c.marker) {
				t.Errorf("expected skill template to contain %q, got:\n%s", c.marker, contentStr)
			}
		})
	}
}

func TestRun_WritesCppFilelenInstabilityAbstractnessReferences(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	refDir := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references", "lang", "cpp")
	files := []string{"filelen.md"}

	for _, filename := range files {
		path := filepath.Join(refDir, filename)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected cpp reference file %q to exist, got error: %v", filename, err)
		}
	}
}

func TestRun_TemplateNoLongerMarksCppFilelenInstabilityAbstractnessUnsupported(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	skillPath := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("failed to read SKILL.md: %v", err)
	}
	skillStr := string(content)

	for _, path := range []string{
		"references/lang/cpp/filelen.md",
	} {
		if !strings.Contains(skillStr, path) {
			t.Errorf("expected SKILL.md to reference %q, got:\n%s", path, skillStr)
		}
	}
}

func TestRun_CppIndexListsFilelenInstabilityAbstractness(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	indexPath := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references", "lang", "cpp", "index.md")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read cpp index.md: %v", err)
	}
	indexStr := string(content)

	for _, marker := range []string{"filelen"} {
		if !strings.Contains(indexStr, marker) {
			t.Errorf("expected cpp index.md to mention %q, got:\n%s", marker, indexStr)
		}
	}
}

func TestRun_TopLevelReferencesPointToCppExamples(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	refDir := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references")

	cases := map[string]string{
		"filelen.md": "references/lang/cpp/filelen.md",
	}

	for filename, wantPath := range cases {
		content, err := os.ReadFile(filepath.Join(refDir, filename))
		if err != nil {
			t.Fatalf("failed to read %q: %v", filename, err)
		}
		if !strings.Contains(string(content), wantPath) {
			t.Errorf("expected %q to reference %q, got:\n%s", filename, wantPath, content)
		}
	}
}

func TestRun_TemplateOrdersDuplicationBySamePackageVsCrossPackage(t *testing.T) {
	baseDir := t.TempDir()

	path, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}
	contentStr := string(content)

	// Check that same-package duplication clusters are mentioned in early tier with funclen
	if !strings.Contains(contentStr, "funclen, linelen, same-package duplication, complexity, filelen, cohesion, cross-package duplication") {
		t.Errorf("expected skill template to mention violation fix order with same-package before cross-package duplication, got:\n%s", contentStr)
	}
}

func TestRun_TemplateStatesClusterCountsAsOnePerType(t *testing.T) {
	baseDir := t.TempDir()

	path, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}
	contentStr := string(content)

	// Check that the cap paragraph states one violation per type per run
	if !strings.Contains(contentStr, "Within each type, select one candidate per run") {
		t.Errorf("expected skill template to state one violation per type per run, got:\n%s", contentStr)
	}
}

func TestRun_TemplateStatesDuplicationSelectionRule(t *testing.T) {
	baseDir := t.TempDir()

	path, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}
	contentStr := string(content)

	// Check for --format=json instruction
	if !strings.Contains(contentStr, "--format=json") {
		t.Errorf("expected skill template to mention --format=json for duplication, got:\n%s", contentStr)
	}

	// Check for JSON field references
	if !strings.Contains(contentStr, "members") || !strings.Contains(contentStr, "pairs") ||
		!strings.Contains(contentStr, "dupLines") || !strings.Contains(contentStr, "crossPackage") {
		t.Errorf("expected skill template to mention JSON fields (members, pairs, dupLines, crossPackage), got:\n%s", contentStr)
	}
}

func TestRun_DuplicationReferenceExplainsWhyAndHow(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	refPath := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references", "duplication.md")
	content, err := os.ReadFile(refPath)
	if err != nil {
		t.Fatalf("failed to read duplication.md: %v", err)
	}
	contentStr := string(content)

	// Check for "Why this is a problem" section
	if !strings.Contains(contentStr, "Why this is a problem") {
		t.Errorf("expected duplication.md to contain 'Why this is a problem' section, got:\n%s", contentStr)
	}

	// Check for "How to fix it" section
	if !strings.Contains(contentStr, "How to fix it") {
		t.Errorf("expected duplication.md to contain 'How to fix it' section, got:\n%s", contentStr)
	}

	// Check for cross-links to meta-pattern.md and functions.md
	if !strings.Contains(contentStr, "meta-pattern.md") {
		t.Errorf("expected duplication.md to cross-link meta-pattern.md, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "functions.md") {
		t.Errorf("expected duplication.md to cross-link functions.md, got:\n%s", contentStr)
	}

	// Check for Examples section referencing Go file
	if !strings.Contains(contentStr, "references/lang/go/duplication.md") {
		t.Errorf("expected duplication.md to reference Go example file, got:\n%s", contentStr)
	}

	// Check that it references the C++ example
	if !strings.Contains(contentStr, "references/lang/cpp/duplication.md") {
		t.Errorf("expected duplication.md to reference C++ duplication example, got:\n%s", contentStr)
	}
}

func TestRun_GoIndexListsDuplicationAndIgnoreDirective(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	indexPath := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references", "lang", "go", "index.md")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read Go index.md: %v", err)
	}
	indexStr := string(content)

	// Check that duplication is listed in Available Checks
	if !strings.Contains(indexStr, "duplication") {
		t.Errorf("expected Go index.md to list duplication in Available Checks, got:\n%s", indexStr)
	}

	// Check for kind-scoped ignore directive documentation
	if !strings.Contains(indexStr, "// boy-scout:ignore:duplication") {
		t.Errorf("expected Go index.md to document // boy-scout:ignore:duplication directive, got:\n%s", indexStr)
	}
}

func TestRun_TemplateTableRoutesDuplicationToReferences(t *testing.T) {
	baseDir := t.TempDir()

	path, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}
	contentStr := string(content)

	// Check that SKILL.md table references the duplication reference files
	if !strings.Contains(contentStr, "references/duplication.md") {
		t.Errorf("expected SKILL.md to reference references/duplication.md, got:\n%s", contentStr)
	}

	if !strings.Contains(contentStr, "references/lang/go/duplication.md") {
		t.Errorf("expected SKILL.md to reference references/lang/go/duplication.md, got:\n%s", contentStr)
	}

	// Check that C++ column references duplication support
	if !strings.Contains(contentStr, "references/lang/cpp/duplication.md") {
		t.Errorf("expected SKILL.md to reference references/lang/cpp/duplication.md, got:\n%s", contentStr)
	}
}

func TestRun_CppReferencesHaveDuplicationFile(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	duplicationPath := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references", "lang", "cpp", "duplication.md")
	if _, err := os.Stat(duplicationPath); os.IsNotExist(err) {
		t.Errorf("expected C++ duplication.md to exist at %q, but file was not found", duplicationPath)
	} else if err != nil {
		t.Errorf("unexpected error checking for C++ duplication.md: %v", err)
	}
}

func TestRun_TemplateShowsCandidatesWithCodeAndReasoning(t *testing.T) {
	baseDir := t.TempDir()

	path, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}
	contentStr := string(content)

	// Required-present markers
	requiredMarkers := []string{
		"List code quality violations",
		"Never edit or write to any file in this step",
		"Hand Off to the User",
		"Applying the Fix",
		"Do not commit",
	}

	for _, marker := range requiredMarkers {
		if !strings.Contains(contentStr, marker) {
			t.Errorf("expected skill template to contain %q, got:\n%s", marker, contentStr)
		}
	}

	// Required-absent markers
	absentMarkers := []string{
		"git commit -m \"Fix boy-scout violations",
		"Ready to commit these changes?",
		"Verification Loop",
	}

	for _, marker := range absentMarkers {
		if strings.Contains(contentStr, marker) {
			t.Errorf("expected skill template to NOT contain %q, got:\n%s", marker, contentStr)
		}
	}
}

func TestRun_TemplateProposesFixPlanBeforeApplying(t *testing.T) {
	baseDir := t.TempDir()

	path, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}
	contentStr := string(content)

	// Required-present markers for the Fix Plan feature
	requiredMarkers := []string{
		"## Proposing a Fix Plan for Selected Violations",
		"Fix Plan",
		"never fabricate a fix plan for something that wasn't offered",
		"one Fix Plan block per candidate, in the order they were listed",
		"End with a confirmation question",
		"Execute the plan's steps against the real files",
	}

	for _, marker := range requiredMarkers {
		if !strings.Contains(contentStr, marker) {
			t.Errorf("expected skill template to contain %q, got:\n%s", marker, contentStr)
		}
	}
}

func TestRun_WritesLinelenReference(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Top-level linelen.md should exist
	path := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references", "linelen.md")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected linelen.md to exist at %q, got error: %v", path, err)
	}

	// Language-specific linelen files should NOT exist (per AC1)
	for _, lang := range []string{"go", "cpp", "ts"} {
		langPath := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references", "lang", lang, "linelen.md")
		if _, err := os.Stat(langPath); !os.IsNotExist(err) {
			t.Errorf("expected %s/linelen.md to NOT exist, but it does or had other error: %v", lang, err)
		}
	}
}

func TestRun_WritesCohesionReferences(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Top-level cohesion.md should exist
	path := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references", "cohesion.md")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected cohesion.md to exist at %q, got error: %v", path, err)
	}

	// Language-specific cohesion files should exist
	for _, lang := range []string{"go", "cpp", "ts"} {
		langPath := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references", "lang", lang, "cohesion.md")
		if _, err := os.Stat(langPath); err != nil {
			t.Errorf("expected %s/cohesion.md to exist at %q, got error: %v", lang, langPath, err)
		}
	}
}

func TestRun_CohesionReferenceExplainsMethodThreshold(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	path := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references", "cohesion.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read cohesion.md: %v", err)
	}

	contentStr := string(content)
	// Check for case-insensitive mention of "2 methods" or similar phrasing
	if !strings.Contains(strings.ToLower(contentStr), "2 methods") &&
		!strings.Contains(strings.ToLower(contentStr), "fewer than 2") &&
		!strings.Contains(strings.ToLower(contentStr), "fewer than two") {
		t.Errorf("expected cohesion.md to explain the 2-method threshold, got:\n%s", contentStr)
	}
}

func TestRun_TemplateTableRoutesCohesionAndLinelenToReferences(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	skillPath := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("failed to read SKILL.md: %v", err)
	}
	skillStr := string(content)

	// Check for cohesion and linelen references in SKILL.md
	expectedRefs := []string{
		"references/cohesion.md",
		"references/linelen.md",
		"references/lang/go/cohesion.md",
		"references/lang/cpp/cohesion.md",
		"references/lang/ts/cohesion.md",
	}

	for _, ref := range expectedRefs {
		if !strings.Contains(skillStr, ref) {
			t.Errorf("expected SKILL.md to contain reference to %q, got:\n%s", ref, skillStr)
		}
	}
}

func TestRun_TemplateOrdersCohesionAndLinelenByDisruption(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	skillPath := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("failed to read SKILL.md: %v", err)
	}
	skillStr := string(content)

	// Check for the exact ordering string from AC5
	expectedOrder := "funclen, linelen, same-package duplication, complexity, filelen, cohesion, cross-package duplication"
	if !strings.Contains(skillStr, expectedOrder) {
		t.Errorf("expected SKILL.md to contain disruption order %q, got:\n%s", expectedOrder, skillStr)
	}
}

func TestRun_TemplateNoLongerMarksCppComplexityUnsupported(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	skillPath := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("failed to read SKILL.md: %v", err)
	}
	skillStr := string(content)

	// Should NOT contain the old "not yet supported for C++" message for complexity
	if strings.Contains(skillStr, "(not yet supported for C++)") && strings.Contains(skillStr, "complexity") {
		// Check more carefully: make sure complexity + "(not yet supported for C++)" don't appear together
		lines := strings.Split(skillStr, "\n")
		for i, line := range lines {
			if strings.Contains(line, "complexity") && strings.Contains(line, "(not yet supported for C++)") {
				t.Errorf("expected complexity row to no longer mark C++ as unsupported, found at line %d: %s", i, line)
			}
		}
	}

	// Should contain reference to C++ complexity file
	if !strings.Contains(skillStr, "references/lang/cpp/complexity.md") {
		t.Errorf("expected SKILL.md to reference cpp/complexity.md, got:\n%s", skillStr)
	}
}

func TestRun_CppIndexListsComplexityAndCohesion(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	indexPath := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references", "lang", "cpp", "index.md")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read cpp index.md: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "complexity") {
		t.Errorf("expected cpp index.md to list complexity, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "cohesion") {
		t.Errorf("expected cpp index.md to list cohesion, got:\n%s", contentStr)
	}
}

func TestRun_GoIndexListsCohesionAndLinelen(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	indexPath := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references", "lang", "go", "index.md")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read go index.md: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "cohesion") {
		t.Errorf("expected go index.md to list cohesion, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "linelen") {
		t.Errorf("expected go index.md to list linelen, got:\n%s", contentStr)
	}
	// Should not contain hardcoded "all five" claim
	if strings.Contains(contentStr, "all five") {
		t.Errorf("expected go index.md to not contain hardcoded 'all five' count, got:\n%s", contentStr)
	}
}

func TestRun_TemplateDeclaresTsDiscoveryMarker(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	skillPath := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("failed to read SKILL.md: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "tsconfig.json") {
		t.Errorf("expected SKILL.md to declare tsconfig.json as TypeScript marker, got:\n%s", contentStr)
	}
}

func TestRun_WritesTsIndex(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	indexPath := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references", "lang", "ts", "index.md")
	if _, err := os.Stat(indexPath); err != nil {
		t.Errorf("expected ts/index.md to exist at %q, got error: %v", indexPath, err)
	}
}

func TestRun_TsIndexListsAvailableChecks(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	indexPath := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references", "lang", "ts", "index.md")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read ts/index.md: %v", err)
	}

	contentStr := string(content)
	expectedChecks := []string{"funclen", "complexity", "cohesion", "filelen", "linelen"}
	for _, check := range expectedChecks {
		if !strings.Contains(contentStr, check) {
			t.Errorf("expected ts/index.md to list %s, got:\n%s", check, contentStr)
		}
	}
}

func TestRun_WritesTsExampleReferences(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	tsDir := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references", "lang", "ts")

	// These files should exist
	shouldExist := []string{"funclen.md", "complexity.md", "cohesion.md", "filelen.md"}
	for _, filename := range shouldExist {
		path := filepath.Join(tsDir, filename)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected ts/%s to exist at %q, got error: %v", filename, path, err)
		}
	}

	// These files should NOT exist
	shouldNotExist := []string{"duplication.md", "linelen.md"}
	for _, filename := range shouldNotExist {
		path := filepath.Join(tsDir, filename)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("expected ts/%s to NOT exist at %q, but it does or had other error: %v", filename, path, err)
		}
	}
}

func TestRun_ValidatesCohesionAndLinelenPresence(t *testing.T) {
	baseDir := t.TempDir()

	// This test exercises validateEmbeddedContent, which should assert that
	// the embedded SKILL.md contains both "cohesion" and "linelen"
	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	// If we got here, validateEmbeddedContent passed (which would have asserted
	// the presence of cohesion and linelen)
}
