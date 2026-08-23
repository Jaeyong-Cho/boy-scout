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
		{"CapAtFive", "Cap each run at 5 violations"},
		{"SeverityWorstFirst", "highest number first"},
		{"TestFilesDeferredLast", "deferred to the end of the list"},
		{"CapSkippedDistinctFromUnresolved", "skipped by the cap"},
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
