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

	output, success := releaseOutput(*changelogFlag)
	fmt.Print(output)
	if !success {
		os.Exit(1)
	}
}

// releaseOutput computes the release output (version or changelog).
func releaseOutput(printChangelog bool) (string, bool) {
	lastTag := getLastTag()
	subjects, err := getCommitSubjects(lastTag)
	if err != nil {
		return "", false
	}

	next, ok := release.NextVersion(lastTag, subjects)
	if !ok {
		return "none\n", true
	}

	if printChangelog {
		return release.ChangelogEntry(next, subjects), true
	}

	return next + "\n", true
}

func getLastTag() string {
	lastTagCmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	tagOutput, err := lastTagCmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(tagOutput))
}

func getCommitSubjects(lastTag string) ([]string, error) {
	var logCmd *exec.Cmd
	if lastTag == "" {
		logCmd = exec.Command("git", "log", "--format=%s")
	} else {
		logCmd = exec.Command("git", "log", lastTag+"..HEAD", "--format=%s")
	}

	logOutput, err := logCmd.Output()
	if err != nil {
		return nil, err
	}

	var subjects []string
	for _, line := range strings.Split(strings.TrimSpace(string(logOutput)), "\n") {
		if line != "" {
			subjects = append(subjects, line)
		}
	}
	return subjects, nil
}
