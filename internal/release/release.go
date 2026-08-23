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
	// Parse the last tag to extract major.minor.patch.
	major, minor, patch := parseTag(lastTag)

	// Classify each commit and determine the highest-priority bump.
	bump := bumpNone
	for _, subject := range commitSubjects {
		b := classifyCommit(subject)
		if b > bump {
			bump = b
		}
	}

	// If no bump-worthy commit, return not-ok.
	if bump == bumpNone {
		return "", false
	}

	// If this is the first-ever release (no prior tag), start at v0.1.0.
	if lastTag == "" {
		return "v0.1.0", true
	}

	// Apply the pre-1.0 rule: breaking changes bump minor, not major, when major=0.
	if bump == bumpMajor && major == 0 {
		bump = bumpMinor
	}

	// Assert the pre-1.0 invariant: after applying the pre-1.0 rule above,
	// we should never have major==0 and bump==bumpMajor.
	assertf(major > 0 || bump != bumpMajor, "NextVersion: pre-1.0 tag %s must never bump major on a breaking-change commit", lastTag)

	// Compute the next version based on the bump type.
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

	return fmt.Sprintf("v%d.%d.%d", major, minor, patch), true
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
	// Ignore Merge and Revert subjects.
	if strings.HasPrefix(subject, "Merge ") || strings.HasPrefix(subject, "Revert ") {
		return bumpNone
	}

	// Match Conventional Commits format: type(scope)?!?: description
	// Types: feat, fix, docs, style, refactor, perf, test, chore
	// Keep in sync with .githooks/commit-msg if the Conventional Commits convention changes.
	re := regexp.MustCompile(`^(feat|fix|docs|style|refactor|perf|test|chore)(\([a-z0-9-]+\))?(!)?:`)
	matches := re.FindStringSubmatch(subject)
	if matches == nil {
		return bumpNone
	}

	commitType := matches[1]
	hasBreaking := matches[3] != "" // The ! marker

	// Determine bump based on type.
	switch commitType {
	case "feat":
		if hasBreaking {
			return bumpMajor
		}
		return bumpMinor
	case "fix":
		if hasBreaking {
			return bumpMajor
		}
		return bumpPatch
	default:
		// docs, chore, etc. don't trigger a bump.
		return bumpNone
	}
}

// ChangelogEntry returns a Markdown section for a given version and commit
// subjects, grouping feat/fix commits under "Features" and "Fixes" headers.
// Subjects of other types (docs, chore, etc.) are excluded from the output.
func ChangelogEntry(version string, commitSubjects []string) string {
	var features, fixes []string

	for _, subject := range commitSubjects {
		// Extract the commit type.
		re := regexp.MustCompile(`^(feat|fix|docs|style|refactor|perf|test|chore)(\([a-z0-9-]+\))?(!)?:\s*(.+)$`)
		matches := re.FindStringSubmatch(subject)
		if matches == nil {
			continue
		}

		commitType := matches[1]
		description := matches[4]

		// Group by type; ignore non-user-visible types.
		switch commitType {
		case "feat":
			features = append(features, description)
		case "fix":
			fixes = append(fixes, description)
		}
	}

	// Build the Markdown entry.
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
