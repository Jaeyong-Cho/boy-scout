package release

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func assertf(cond bool, format string, args ...any) {
	if !cond {
		panic(fmt.Sprintf(format, args...))
	}
}

// bumpType represents a semantic version bump priority: patch < minor < major.
type bumpType int

const (
	bumpNone bumpType = iota
	bumpPatch
	bumpMinor
	bumpMajor
)

// NextVersion computes the next SemVer tag from a prior tag and a list of
// commit subjects (using Conventional Commits types). It returns the next
// version string and ok=true if any bump-worthy commit exists; otherwise
// it returns ("", false).
//
// Rules:
//   - fix → patch bump
//   - feat → minor bump
//   - breaking change (! suffix or BREAKING CHANGE: footer) → major bump,
//     unless major version is 0 (pre-1.0 initial development), in which
//     case → minor bump per SemVer spec
//   - Merge/Revert subjects are ignored
//   - If lastTag is empty and at least one bump-worthy commit exists,
//     start at v0.1.0 (first-ever release)
//   - If no bump-worthy commits, return ("", false)
//
// Precondition: if lastTag is not empty, it must parse to non-negative
// major/minor/patch integers or be treated as "no tag".
func NextVersion(lastTag string, commitSubjects []string) (next string, ok bool) {
	bump := highestBump(commitSubjects)
	if bump == bumpNone {
		return "", false
	}

	if lastTag == "" {
		return "v0.1.0", true
	}

	major, minor, patch := parseTag(lastTag)
	return applyBump(major, minor, patch, bump), true
}

// highestBump finds the highest-priority bump across all commits.
func highestBump(subjects []string) bumpType {
	bump := bumpNone
	for _, subject := range subjects {
		b := classifyCommit(subject)
		if b > bump {
			bump = b
		}
	}
	return bump
}

// applyBump increments version components based on bump type,
// accounting for the pre-1.0 rule: breaking changes bump minor, not major, when major=0.
func applyBump(major, minor, patch int, bump bumpType) string {
	if bump == bumpMajor && major == 0 {
		bump = bumpMinor
	}

	assertf(major > 0 || bump != bumpMajor, "applyBump: major=0 must never have bump=bumpMajor")

	switch bump {
	case bumpPatch:
		patch++
	case bumpMinor:
		minor++
		patch = 0
	case bumpMajor:
		major++
		minor = 0
		patch = 0
	}

	return fmt.Sprintf("v%d.%d.%d", major, minor, patch)
}

// parseTag parses a version string in the form vMAJOR.MINOR.PATCH.
// If the tag is empty or doesn't match the pattern, it returns (0, 0, 0).
// Precondition: if the tag matches the pattern, it must parse to non-negative integers.
func parseTag(tag string) (major, minor, patch int) {
	if tag == "" {
		return 0, 0, 0
	}

	// Match vMAJOR.MINOR.PATCH format.
	re := regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)
	matches := re.FindStringSubmatch(tag)
	if matches == nil {
		// Malformed tag; treat as "no tag".
		return 0, 0, 0
	}

	// Parse the components. The regex ensures they are non-negative integers.
	major, _ = strconv.Atoi(matches[1])
	minor, _ = strconv.Atoi(matches[2])
	patch, _ = strconv.Atoi(matches[3])

	// Assert that parsed values are non-negative (they should be from regex).
	assertf(major >= 0 && minor >= 0 && patch >= 0,
		"parseTag: parsed non-negative integers from %s", tag)

	return major, minor, patch
}

// classifyCommit returns the bump type for a given commit subject.
// It recognizes Conventional Commits format and breaking-change markers.
//   - fix → patch
//   - feat → minor
//   - feat! or fix! → breaking (major or minor, depending on current major version)
//   - Merge/Revert subjects → none (ignored)
//   - Other types (docs, chore, etc.) → none
//
// The pattern matches: <type>(<scope>)?!?: <description>
// Keep in sync with .githooks/commit-msg if the Conventional Commits convention changes.
func classifyCommit(subject string) bumpType {
	if strings.HasPrefix(subject, "Merge ") || strings.HasPrefix(subject, "Revert ") {
		return bumpNone
	}

	commitType, hasBreaking := parseCommitType(subject)
	if hasBreaking {
		return bumpMajor
	}

	switch commitType {
	case "feat":
		return bumpMinor
	case "fix":
		return bumpPatch
	default:
		return bumpNone
	}
}

// parseCommitType extracts the commit type and breaking-change marker from a subject.
// Returns ("", false) if the subject doesn't match Conventional Commits format.
func parseCommitType(subject string) (typ string, hasBreaking bool) {
	re := regexp.MustCompile(`^(feat|fix|docs|style|refactor|perf|test|chore)(\([a-z0-9-]+\))?(!)?:`)
	matches := re.FindStringSubmatch(subject)
	if matches == nil {
		return "", false
	}
	return matches[1], matches[3] != ""
}

// ChangelogEntry returns a Markdown section for a given version and commit
// subjects, grouping feat/fix commits under "Features" and "Fixes" headers.
// Subjects of other types (docs, chore, etc.) are excluded from the output.
func ChangelogEntry(version string, commitSubjects []string) string {
	features, fixes := groupCommits(commitSubjects)

	var sb strings.Builder
	fmt.Fprintf(&sb, "## %s\n\n", version)

	if len(features) > 0 {
		sb.WriteString("### Features\n\n")
		for _, f := range features {
			fmt.Fprintf(&sb, "- %s\n", f)
		}
		sb.WriteString("\n")
	}

	if len(fixes) > 0 {
		sb.WriteString("### Fixes\n\n")
		for _, f := range fixes {
			fmt.Fprintf(&sb, "- %s\n", f)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// groupCommits groups commit subjects into features and fixes by type.
func groupCommits(subjects []string) (features, fixes []string) {
	re := regexp.MustCompile(`^(feat|fix|docs|style|refactor|perf|test|chore)(\([a-z0-9-]+\))?(!)?:\s*(.+)$`)

	for _, subject := range subjects {
		matches := re.FindStringSubmatch(subject)
		if matches == nil {
			continue
		}

		commitType := matches[1]
		description := matches[4]

		switch commitType {
		case "feat":
			features = append(features, description)
		case "fix":
			fixes = append(fixes, description)
		}
	}

	return features, fixes
}
