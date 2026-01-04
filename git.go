package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

// GetColoredStatus returns a colored, human-readable git status
func GetColoredStatus() (string, error) {
	cmd := exec.Command("git", "status", "--short")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	if len(output) == 0 {
		return "", nil
	}

	lines := strings.Split(string(output), "\n")
	var result strings.Builder

	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
	orangeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8800"))

	for _, line := range lines {
		// Skip empty lines
		if len(line) < 4 {
			continue
		}

		// Git status format: XY filename (with leading space)
		// Position 0: space, Position 1: staged status, Position 2: unstaged status
		statusCode := line[0:2]
		filename := line[3:]

		var color lipgloss.Style
		var status string

		switch {
		// New file (untracked)
		case statusCode == "??":
			color = redStyle
			status = "  untracked"
		// Added (staged)
		case statusCode[0] == 'A':
			color = greenStyle
			status = "  added"
		// Modified (unstaged)
		case statusCode[1] == 'M':
			color = orangeStyle
			status = "  modified"
		// Modified (staged)
		case statusCode[0] == 'M':
			color = greenStyle
			status = "  modified"
		// Deleted (unstaged)
		case statusCode[1] == 'D':
			color = redStyle
			status = "  deleted"
		// Deleted (staged)
		case statusCode[0] == 'D':
			color = redStyle
			status = "  deleted"
		// Renamed
		case statusCode[0] == 'R':
			color = greenStyle
			status = "  renamed"
		// Copied
		case statusCode[0] == 'C':
			color = greenStyle
			status = "  copied"
		default:
			color = orangeStyle
			status = "  changed"
		}

		result.WriteString(color.Render(fmt.Sprintf("%-12s", status)))
		result.WriteString(" ")
		result.WriteString(color.Render(filename))
		result.WriteString("\n")
	}

	return result.String(), nil
}
