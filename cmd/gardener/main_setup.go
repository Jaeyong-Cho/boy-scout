package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"gardener-go/internal/setup"
)

// setupTarget maps a target name (claude, copilot, pi, agents) to its directory prefix.
type setupTarget struct {
	name   string
	prefix string
}

// setupTargets is the ordered list of valid setup targets.
var setupTargets = []setupTarget{
	{"claude", ".claude"},
	{"copilot", ".copilot"},
	{"pi", ".pi/agent"},
	{"agents", ".agents"},
}

// setupTargetNames returns a slice of valid target names.
func setupTargetNames() []string {
	names := make([]string, len(setupTargets))
	for i, t := range setupTargets {
		names[i] = t.name
	}
	return names
}

// lookupSetupTarget looks up a target by name and returns its prefix and whether it was found.
func lookupSetupTarget(name string) (prefix string, ok bool) {
	for _, t := range setupTargets {
		if t.name == name {
			return t.prefix, true
		}
	}
	return "", false
}

// printTargetPrompt writes the numbered target list and input prompt to stdout.
func printTargetPrompt(stdout io.Writer) {
	fmt.Fprintf(stdout, "Select a target:\n")
	for i, t := range setupTargets {
		fmt.Fprintf(stdout, "%d) %s\n", i+1, t.name)
	}
	fmt.Fprintf(stdout, "> ")
}

// parseTargetSelection resolves one line of raw input to a target name,
// matching either by 1-based index or by name.
func parseTargetSelection(input string) (name string, err error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("empty input; valid targets: %v", setupTargetNames())
	}

	// Try to parse as 1-based index
	if idx, err := strconv.Atoi(input); err == nil {
		if idx >= 1 && idx <= len(setupTargets) {
			return setupTargets[idx-1].name, nil
		}
		return "", fmt.Errorf("invalid index %d; valid targets: %v", idx, setupTargetNames())
	}

	// Try to match by name
	if _, ok := lookupSetupTarget(input); ok {
		return input, nil
	}

	return "", fmt.Errorf("unknown target %q; valid targets: %v", input, setupTargetNames())
}

// promptForTarget prints a numbered list of targets to stdout, reads one line from stdin,
// and returns the target name (either by 1-based index or name match).
// Returns an error if the input is invalid, empty, or EOF.
func promptForTarget(stdin io.Reader, stdout io.Writer) (name string, err error) {
	printTargetPrompt(stdout)

	scanner := bufio.NewScanner(stdin)
	if !scanner.Scan() {
		// EOF or scan error (empty input, no newline, etc.)
		return "", fmt.Errorf("no input provided; valid targets: %v", setupTargetNames())
	}

	return parseTargetSelection(scanner.Text())
}

// isInteractive checks if the given reader is a terminal (real TTY).
func isInteractive(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// resolveTarget determines the target name from positional args or by prompting.
// When interactive, it prompts the user. When not interactive, it returns an error.
func resolveTarget(positional []string, stdin io.Reader, interactive bool, stdout io.Writer) (name string, prefix string, err error) {
	if len(positional) > 0 {
		// Explicit target provided
		targetName := positional[0]
		prefix, ok := lookupSetupTarget(targetName)
		if !ok {
			return "", "", fmt.Errorf("unknown target %q; valid targets: %v", targetName, setupTargetNames())
		}
		return targetName, prefix, nil
	}

	// No target provided; need to ask
	if interactive {
		targetName, err := promptForTarget(stdin, stdout)
		if err != nil {
			return "", "", err
		}
		prefix, ok := lookupSetupTarget(targetName)
		assertf(ok, "promptForTarget returned unknown target %q", targetName)
		return targetName, prefix, nil
	}

	// Non-interactive with no target
	return "", "", fmt.Errorf("no target provided; valid targets: %v", setupTargetNames())
}

// prepareSetupArgs resolves the install base directory (home dir if global, else
// cwd) and the current executable path needed by setup.Run.
func prepareSetupArgs(global bool) (baseDir, exePath string, err error) {
	if global {
		baseDir, err = os.UserHomeDir()
	} else {
		baseDir = "."
	}
	if err != nil {
		return "", "", fmt.Errorf("failed to get home directory: %w", err)
	}

	exePath, err = os.Executable()
	if err != nil {
		return "", "", fmt.Errorf("failed to get executable path: %w", err)
	}

	return baseDir, exePath, nil
}

// parseSetupArgs manually scans args for a --global flag and collects positional
// arguments, since --global may appear in any position and flag.Parse() alone
// can't express that. Returns an error (with any usage message already written
// to stderr) if an unknown flag or more than one positional argument is found.
func parseSetupArgs(args []string, stderr io.Writer) (global bool, positional []string, err error) {
	for _, arg := range args {
		if arg == "--global" || arg == "-global" {
			global = true
		} else if strings.HasPrefix(arg, "-") {
			fmt.Fprintf(stderr, "unknown flag: %s\n", arg)
			fmt.Fprintln(stderr, "usage: gardener setup [claude|copilot|pi|agents] [--global]")
			return false, nil, fmt.Errorf("unknown flag: %s", arg)
		} else {
			positional = append(positional, arg)
		}
	}

	if len(positional) > 1 {
		fmt.Fprintln(stderr, "usage: gardener setup [claude|copilot|pi|agents] [--global]")
		return false, nil, fmt.Errorf("too many positional arguments")
	}

	return global, positional, nil
}

func runSetup(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	global, positional, err := parseSetupArgs(args, stderr)
	if err != nil {
		return 2
	}

	baseDir, exePath, err := prepareSetupArgs(global)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}
	assertf(baseDir != "", "baseDir must not be empty")

	// Resolve the target (either explicit or by prompting)
	_, prefix, err := resolveTarget(positional, stdin, isInteractive(stdin), stdout)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	path, err := setup.Run(baseDir, exePath, prefix)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	fmt.Fprintf(stdout, "gardener skill installed: %s\n", path)
	return 0
}
