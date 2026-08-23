package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"boy-scout/internal/release"
)

func main() {
	changelogFlag := flag.Bool("changelog", false, "print changelog entry instead of version")
	flag.Parse()

	// Get the last git tag.
	lastTagCmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	tagOutput, err := lastTagCmd.Output()
	lastTag := strings.TrimSpace(string(tagOutput))

	// If there's no tag, that's OK; we treat it as "no prior tag".
	if err != nil {
		lastTag = ""
	}

	// Get commit subjects since the last tag (or all of HEAD if no tag).
	var logCmd *exec.Cmd
	if lastTag == "" {
		// No prior tag: get all commits.
		logCmd = exec.Command("git", "log", "--format=%s")
	} else {
		// Get commits since the last tag.
		logCmd = exec.Command("git", "log", lastTag+"..HEAD", "--format=%s")
	}

	logOutput, err := logCmd.Output()
	if err != nil {
		// If the git command fails, print nothing and exit.
		os.Exit(1)
	}

	var subjects []string
	for _, line := range strings.Split(strings.TrimSpace(string(logOutput)), "\n") {
		if line != "" {
			subjects = append(subjects, line)
		}
	}

	// Compute the next version.
	next, ok := release.NextVersion(lastTag, subjects)
	if !ok {
		fmt.Println("none")
		os.Exit(0)
	}

	// If -changelog flag is set, print the changelog entry instead of the version.
	if *changelogFlag {
		fmt.Print(release.ChangelogEntry(next, subjects))
		os.Exit(0)
	}

	fmt.Println(next)
	os.Exit(0)
}
