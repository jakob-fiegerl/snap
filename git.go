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

// CheckRemoteExists checks if a remote repository is configured
func CheckRemoteExists() (bool, error) {
	cmd := exec.Command("git", "remote")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(output))) > 0, nil
}

// GetCurrentBranch returns the name of the current branch
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// HasUpstreamBranch checks if the current branch has an upstream branch
func HasUpstreamBranch() (bool, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "@{upstream}")
	err := cmd.Run()
	return err == nil, nil
}

// PullChanges pulls changes from the remote repository
func PullChanges() (string, error) {
	cmd := exec.Command("git", "pull")
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// PushChanges pushes changes to the remote repository
func PushChanges() (string, error) {
	cmd := exec.Command("git", "push")
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// PushWithUpstream pushes changes and sets upstream tracking
func PushWithUpstream(branch string) (string, error) {
	cmd := exec.Command("git", "push", "-u", "origin", branch)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// CheckForUncommittedChanges checks if there are uncommitted changes
func CheckForUncommittedChanges() (bool, error) {
	status, err := GetStatus()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(status)) > 0, nil
}

// InitRepository initializes a new git repository
func InitRepository() (string, error) {
	cmd := exec.Command("git", "init")
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// IsGitRepository checks if the current directory is a git repository
func IsGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	err := cmd.Run()
	return err == nil
}

// CommitInfo represents a single commit in the history
type CommitInfo struct {
	Hash         string
	ShortHash    string
	Message      string
	Author       string
	Date         string
	RelativeTime string
}

// GetCommitHistory returns a list of commits with formatting
func GetCommitHistory(limit int, allBranches bool, author string, filePath string) ([]CommitInfo, error) {
	args := []string{"log", "--pretty=format:%H|%h|%s|%an|%ai|%ar"}

	if limit > 0 {
		args = append(args, fmt.Sprintf("-%d", limit))
	}

	if allBranches {
		args = append(args, "--all")
	}

	if author != "" {
		args = append(args, fmt.Sprintf("--author=%s", author))
	}

	if filePath != "" {
		args = append(args, "--", filePath)
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	if len(output) == 0 {
		return []CommitInfo{}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	commits := make([]CommitInfo, 0, len(lines))

	for _, line := range lines {
		parts := strings.SplitN(line, "|", 6)
		if len(parts) != 6 {
			continue
		}

		commits = append(commits, CommitInfo{
			Hash:         parts[0],
			ShortHash:    parts[1],
			Message:      parts[2],
			Author:       parts[3],
			Date:         parts[4],
			RelativeTime: parts[5],
		})
	}

	return commits, nil
}
