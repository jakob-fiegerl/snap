package main

import (
	"os/exec"
	"strings"
)

// GetGitDiff returns the git diff of staged or unstaged changes
func GetGitDiff() (string, error) {
	// First try to get staged changes
	cmd := exec.Command("git", "diff", "--cached")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	diff := string(output)
	if strings.TrimSpace(diff) == "" {
		// If no staged changes, get unstaged changes
		cmd = exec.Command("git", "diff")
		output, err = cmd.Output()
		if err != nil {
			return "", err
		}
		diff = string(output)
	}

	return diff, nil
}

// StageAllChanges stages all changes in the repository
func StageAllChanges() error {
	cmd := exec.Command("git", "add", "-A")
	return cmd.Run()
}

// CommitChanges commits staged changes with the given message
func CommitChanges(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	return cmd.Run()
}

// GetStatus returns the git status showing modified, added, and untracked files
func GetStatus() (string, error) {
	cmd := exec.Command("git", "status", "--short")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
