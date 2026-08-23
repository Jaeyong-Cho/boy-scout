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
	if !strings.Contains(contentStr, "go test") {
		t.Errorf("expected 'go test' in skill body, got:\n%s", contentStr)
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

func TestRun_TemplateDeclaresTDDRefactorAndViolationGuidance(t *testing.T) {
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
		{"BaselineGreenGate", "tests were already failing before any boy-scout edit"},
		{"CharacterizationTestForZeroCoverage", "characterization test"},
		{"AttemptCapIncludesCharacterizationTest", "including its characterization test"},
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
		{"FilelenFixedLast", "then filelen, then instability, then abstractness"},
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
	referenceFiles := []string{"funclen.md", "crap.md", "filelen.md", "instability.md", "abstractness.md"}

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
		{"Crap", "crap.md", "high complexity plus low test coverage", "characterization test"},
		{"Filelen", "filelen.md", "mixing multiple concerns", "high cohesion"},
		{"Instability", "instability.md", "least-stable thing it leans on", "Point the dependency the other way"},
		{"Abstractness", "abstractness.md", "Zone of Pain", "deep module"},
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
	referenceFiles := []string{"funclen.md", "crap.md", "filelen.md", "instability.md", "abstractness.md"}

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
		"references/crap.md",
		"references/filelen.md",
		"references/instability.md",
		"references/abstractness.md",
	}

	for _, refFile := range referenceFiles {
		if !strings.Contains(contentStr, refFile) {
			t.Errorf("expected skill template to contain reference to %q, got:\n%s", refFile, contentStr)
		}
	}

	// Check fix-order statement
	if !strings.Contains(contentStr, "then filelen, then instability, then abstractness") {
		t.Errorf("expected skill template to contain fix-order statement with all 5 kinds, got:\n%s", contentStr)
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
		"**crap violation:**",
		"**filelen violation:**",
	}

	for _, marker := range forbiddenMarkers {
		if strings.Contains(contentStr, marker) {
			t.Errorf("expected skill template to no longer contain inline violation explanation %q, got:\n%s", marker, contentStr)
		}
	}
}

func TestRun_TemplateDeclaresFixCapAndPriority(t *testing.T) {
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
		{"CapPerType", "Fix one violation per violation-type per run"},
		{"SeverityWorstFirst", "highest number first"},
		{"TestFilesDeferredLast", "deferred to the end of the list"},
		{"ExampleMultipleViolationTypes", "one per kind"},
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
	files := []string{"filelen.md", "instability.md", "abstractness.md"}

	for _, filename := range files {
		path := filepath.Join(refDir, filename)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected cpp reference file %q to exist, got error: %v", filename, err)
		}
	}
}

func TestRun_CppLangReferenceFilesContainCppExamples(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	refDir := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references", "lang", "cpp")

	cases := []struct {
		name     string
		filename string
		marker   string
	}{
		{"Filelen", "filelen.md", "#include"},
		{"Instability", "instability.md", "domain.hpp"},
		{"Abstractness", "abstractness.md", "pure virtual"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(refDir, c.filename))
			if err != nil {
				t.Fatalf("failed to read %q: %v", c.filename, err)
			}
			if !strings.Contains(string(content), c.marker) {
				t.Errorf("expected %q to contain %q, got:\n%s", c.filename, c.marker, content)
			}
		})
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
		"references/lang/cpp/instability.md",
		"references/lang/cpp/abstractness.md",
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

	for _, marker := range []string{"filelen", "instability", "abstractness"} {
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
		"filelen.md":      "references/lang/cpp/filelen.md",
		"instability.md":  "references/lang/cpp/instability.md",
		"abstractness.md": "references/lang/cpp/abstractness.md",
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
	if !strings.Contains(contentStr, "funclen violations first, then same-package duplication clusters") {
		t.Errorf("expected skill template to mention same-package duplication clusters after funclen, got:\n%s", contentStr)
	}

	// Check that cross-package duplication clusters are mentioned in late tier after abstractness
	if !strings.Contains(contentStr, "then cross-package duplication clusters last") {
		t.Errorf("expected skill template to mention cross-package duplication clusters last, got:\n%s", contentStr)
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
	if !strings.Contains(contentStr, "Fix one violation per violation-type per run") {
		t.Errorf("expected skill template to state one violation per type per run, got:\n%s", contentStr)
	}
}

func TestRun_TemplateStatesDuplicationClusterFixRule(t *testing.T) {
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

	// Check for atomic edit instruction
	if !strings.Contains(contentStr, "one atomic multi-file edit") {
		t.Errorf("expected skill template to mention atomic edit requirement, got:\n%s", contentStr)
	}

	// Check for extract-a-helper default
	if !strings.Contains(contentStr, "extract-a-helper") && !strings.Contains(contentStr, "extract one shared helper") {
		t.Errorf("expected skill template to mention extract-a-helper default, got:\n%s", contentStr)
	}

	// Check for never delete-one-copy rule
	if !strings.Contains(contentStr, "never delete-one-copy") {
		t.Errorf("expected skill template to mention never delete-one-copy rule, got:\n%s", contentStr)
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

	// Check that it notes C++ isn't supported yet
	if !strings.Contains(contentStr, "not yet supported") || !strings.Contains(contentStr, "C++") {
		t.Errorf("expected duplication.md to note C++ not yet supported, got:\n%s", contentStr)
	}
}

func TestRun_WritesGoDuplicationReferenceExample(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	refPath := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references", "lang", "go", "duplication.md")
	content, err := os.ReadFile(refPath)
	if err != nil {
		t.Fatalf("failed to read Go duplication.md: %v", err)
	}
	contentStr := string(content)

	// Check for Go code fences (```go ... ```)
	if !strings.Contains(contentStr, "```go") {
		t.Errorf("expected Go duplication.md to contain Go code example, got:\n%s", contentStr)
	}

	// Check for before/after pattern
	if !strings.Contains(contentStr, "Problem example") && !strings.Contains(contentStr, "before") {
		t.Errorf("expected Go duplication.md to have a problem/before example, got:\n%s", contentStr)
	}

	if !strings.Contains(contentStr, "Good resolve example") && !strings.Contains(contentStr, "after") {
		t.Errorf("expected Go duplication.md to have a good/after example, got:\n%s", contentStr)
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

	// Check that C++ column notes duplication isn't supported for C++
	if !strings.Contains(contentStr, "(not yet supported for C++)") {
		t.Errorf("expected SKILL.md to note duplication not yet supported for C++, got:\n%s", contentStr)
	}
}

func TestRun_CppReferencesHaveNoDuplicationFile(t *testing.T) {
	baseDir := t.TempDir()

	_, err := Run(baseDir, "", ".agents")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	duplicationPath := filepath.Join(baseDir, ".agents", "skills", "boy-scout", "references", "lang", "cpp", "duplication.md")
	if _, err := os.Stat(duplicationPath); err == nil {
		t.Errorf("expected C++ duplication.md to NOT exist, but file was found at %q", duplicationPath)
	} else if !os.IsNotExist(err) {
		t.Errorf("unexpected error checking for C++ duplication.md: %v", err)
	}
}
