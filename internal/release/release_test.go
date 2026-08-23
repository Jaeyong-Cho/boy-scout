package release_test

import (
	"strings"
	"testing"

	"boy-scout/internal/release"
)

// TestNextVersion_FeatBeatsFix verifies that when both fix and feat commits exist,
// the minor bump (feat) takes precedence over patch (fix).
func TestNextVersion_FeatBeatsFix(t *testing.T) {
	next, ok := release.NextVersion("v0.4.0", []string{"fix: x", "feat: add abstractness flag"})
	if !ok {
		t.Fatal("NextVersion should return ok=true for bump-worthy commits")
	}
	if next != "v0.5.0" {
		t.Errorf("NextVersion: got %q, want v0.5.0", next)
	}
}

// TestNextVersion_FixOnlyIsPatch verifies that a fix-only commit produces a patch bump.
func TestNextVersion_FixOnlyIsPatch(t *testing.T) {
	next, ok := release.NextVersion("v0.4.0", []string{"fix: x"})
	if !ok {
		t.Fatal("NextVersion should return ok=true for bump-worthy commits")
	}
	if next != "v0.4.1" {
		t.Errorf("NextVersion: got %q, want v0.4.1", next)
	}
}

// TestNextVersion_NoPriorTagStartsAtZeroOneZero verifies that the first-ever
// release starts at v0.1.0 when at least one bump-worthy commit exists.
func TestNextVersion_NoPriorTagStartsAtZeroOneZero(t *testing.T) {
	next, ok := release.NextVersion("", []string{"feat: initial CLI"})
	if !ok {
		t.Fatal("NextVersion should return ok=true for bump-worthy commits")
	}
	if next != "v0.1.0" {
		t.Errorf("NextVersion: got %q, want v0.1.0", next)
	}
}

// TestNextVersion_BreakingPre1_0BumpsMinorNotMajor verifies that while the
// major version is 0, a breaking-change commit bumps minor (not major).
func TestNextVersion_BreakingPre1_0BumpsMinorNotMajor(t *testing.T) {
	next, ok := release.NextVersion("v0.4.1", []string{"feat!: rename flag"})
	if !ok {
		t.Fatal("NextVersion should return ok=true for bump-worthy commits")
	}
	if next != "v0.5.0" {
		t.Errorf("NextVersion: got %q, want v0.5.0 (minor bump, not major, since major=0)", next)
	}
}

// TestNextVersion_NoBumpWorthyCommitsReturnsNotOK verifies that when no
// bump-worthy commits exist (only docs, chore, etc.), ok is false.
func TestNextVersion_NoBumpWorthyCommitsReturnsNotOK(t *testing.T) {
	next, ok := release.NextVersion("v0.4.1", []string{"docs: fix typo", "chore: cleanup"})
	if ok {
		t.Fatal("NextVersion should return ok=false when no bump-worthy commits exist")
	}
	if next != "" {
		t.Errorf("NextVersion should return empty string when ok=false; got %q", next)
	}
}

// TestNextVersion_IgnoresMergeAndRevertSubjects verifies that Merge and Revert
// commit subjects are ignored and don't prevent a bump.
func TestNextVersion_IgnoresMergeAndRevertSubjects(t *testing.T) {
	next, ok := release.NextVersion("v0.4.1", []string{"Merge branch 'x' into main", "fix: y"})
	if !ok {
		t.Fatal("NextVersion should return ok=true; the merge is ignored, the fix is bump-worthy")
	}
	if next != "v0.4.2" {
		t.Errorf("NextVersion: got %q, want v0.4.2", next)
	}
}

// TestChangelogEntry_GroupsByType verifies that ChangelogEntry groups feat/fix
// by type in a Markdown section, excludes non-user-visible types (chore, docs, etc.),
// and includes the version heading.
func TestChangelogEntry_GroupsByType(t *testing.T) {
	entry := release.ChangelogEntry("v0.5.0", []string{"feat: add x", "fix: y", "chore: z"})

	// Should contain the version heading.
	if !strings.Contains(entry, "## v0.5.0") {
		t.Errorf("ChangelogEntry should contain '## v0.5.0'; got:\n%s", entry)
	}

	// Should contain a Features section with the feat subject.
	if !strings.Contains(entry, "Features") || !strings.Contains(entry, "add x") {
		t.Errorf("ChangelogEntry should contain Features section with 'add x'; got:\n%s", entry)
	}

	// Should contain a Fixes section with the fix subject.
	if !strings.Contains(entry, "Fixes") || !strings.Contains(entry, "y") {
		t.Errorf("ChangelogEntry should contain Fixes section with 'y'; got:\n%s", entry)
	}

	// Should NOT mention the chore subject.
	if strings.Contains(entry, "chore: z") || strings.Contains(entry, "cleanup") {
		t.Errorf("ChangelogEntry should not include chore commits; got:\n%s", entry)
	}
}
