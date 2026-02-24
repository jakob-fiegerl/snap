package main

import (
	"fmt"
	"os/exec"
	"strconv"
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

// FileStatus represents a single file's status
type FileStatus struct {
	Status   string
	Filename string
	Staged   bool
}

// GetCommitsAheadOfRemote returns the number of commits ahead of the remote branch
func GetCommitsAheadOfRemote() (int, error) {
	// Check if upstream exists first
	hasUpstream, err := HasUpstreamBranch()
	if err != nil || !hasUpstream {
		return 0, nil
	}

	cmd := exec.Command("git", "rev-list", "--count", "@{upstream}..HEAD")
	output, err := cmd.Output()
	if err != nil {
		return 0, nil
	}

	var count int
	fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &count)
	return count, nil
}

// GetUnpushedCommits returns commits that are ahead of the remote branch
func GetUnpushedCommits() ([]CommitInfo, error) {
	// Check if upstream exists first
	hasUpstream, err := HasUpstreamBranch()
	if err != nil || !hasUpstream {
		return []CommitInfo{}, nil
	}

	// Get commits between upstream and HEAD
	args := []string{"log", "--pretty=format:%H|%h|%s|%an|%ai|%ar", "@{upstream}..HEAD"}
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return []CommitInfo{}, nil
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

// GetBaseCommit returns the last commit that exists on the remote (upstream)
func GetBaseCommit() (CommitInfo, error) {
	// Check if upstream exists first
	hasUpstream, err := HasUpstreamBranch()
	if err != nil || !hasUpstream {
		// If no upstream, return the last commit
		commits, err := GetCommitHistory(1, false, "", "")
		if err != nil || len(commits) == 0 {
			return CommitInfo{}, fmt.Errorf("no commits found")
		}
		return commits[0], nil
	}

	// Get the commit that upstream points to
	args := []string{"log", "--pretty=format:%H|%h|%s|%an|%ai|%ar", "-1", "@{upstream}"}
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return CommitInfo{}, err
	}

	line := strings.TrimSpace(string(output))
	parts := strings.SplitN(line, "|", 6)
	if len(parts) != 6 {
		return CommitInfo{}, fmt.Errorf("invalid commit format")
	}

	return CommitInfo{
		Hash:         parts[0],
		ShortHash:    parts[1],
		Message:      parts[2],
		Author:       parts[3],
		Date:         parts[4],
		RelativeTime: parts[5],
	}, nil
}

// GetEnhancedStatus returns a beautifully formatted git status with metadata
func GetEnhancedStatus() (string, error) {
	var result strings.Builder

	// Styles
	stagedHeaderStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorSuccess)

	unstagedHeaderStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorSecondary)

	localCommitsHeaderStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorSecondary)

	originBranchStyle := lipgloss.NewStyle().
		Foreground(colorSuccess)

	infoStyle := lipgloss.NewStyle().
		Foreground(colorMuted)

	whiteStyle := lipgloss.NewStyle().Foreground(colorText)
	filenameStyle := lipgloss.NewStyle().Foreground(colorMuted)
	boxStyle := lipgloss.NewStyle().Foreground(colorMuted)

	// Get current branch
	branch, err := GetCurrentBranch()
	if err != nil {
		return "", err
	}

	// Get base commit (last pushed)
	baseCommit, err := GetBaseCommit()
	var hasBase bool
	if err == nil {
		hasBase = true
	}

	// Get unpushed commits
	unpushedCommits, _ := GetUnpushedCommits()

	// Get status
	cmd := exec.Command("git", "status", "--short")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse files into staged and unstaged
	lines := strings.Split(string(output), "\n")
	var stagedFiles []FileStatus
	var unstagedFiles []FileStatus

	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		statusCode := line[0:2]
		filename := line[3:]

		// Check staged status (first character)
		if statusCode[0] != ' ' && statusCode[0] != '?' {
			var status string
			switch statusCode[0] {
			case 'A':
				status = "A"
			case 'M':
				status = "M"
			case 'D':
				status = "D"
			case 'R':
				status = "R"
			case 'C':
				status = "C"
			default:
				status = "?"
			}
			stagedFiles = append(stagedFiles, FileStatus{
				Status:   status,
				Filename: filename,
				Staged:   true,
			})
		}

		// Check unstaged status (second character)
		if statusCode[1] != ' ' {
			var status string
			switch statusCode[1] {
			case 'M':
				status = "M"
			case 'D':
				status = "D"
			case '?':
				status = "?"
			default:
				status = "?"
			}
			unstagedFiles = append(unstagedFiles, FileStatus{
				Status:   status,
				Filename: filename,
				Staged:   false,
			})
		}

		// Handle untracked files
		if statusCode == "??" {
			unstagedFiles = append(unstagedFiles, FileStatus{
				Status:   "?",
				Filename: filename,
				Staged:   false,
			})
		}
	}

	// Check if we have anything to display
	hasChanges := len(stagedFiles) > 0 || len(unstagedFiles) > 0
	hasUnpushed := len(unpushedCommits) > 0

	if !hasChanges && !hasUnpushed {
		result.WriteString(stagedHeaderStyle.Render("✓ Working tree clean - no changes"))
		result.WriteString("\n")
		return result.String(), nil
	}

	// Start the box only if we have staged changes
	if len(stagedFiles) > 0 {
		result.WriteString(boxStyle.Render("┌"))
		result.WriteString("\n")
	}

	// Display staged changes
	if len(stagedFiles) > 0 {
		result.WriteString(boxStyle.Render("│  "))
		result.WriteString(stagedHeaderStyle.Render("[staged changes]"))
		result.WriteString("\n")
		for _, file := range stagedFiles {
			result.WriteString(boxStyle.Render("│  "))
			result.WriteString(whiteStyle.Render(fmt.Sprintf("%s", file.Status)))
			result.WriteString("  ")
			result.WriteString(filenameStyle.Render(file.Filename))
			result.WriteString("\n")
		}
		result.WriteString(boxStyle.Render("│"))
		result.WriteString("\n")
	}

	// Display unstaged changes
	if len(unstagedFiles) > 0 {
		result.WriteString(boxStyle.Render("├─ "))
		result.WriteString(unstagedHeaderStyle.Render("[unstaged changes]"))
		result.WriteString("\n")
		for _, file := range unstagedFiles {
			result.WriteString(boxStyle.Render("│  "))
			result.WriteString(whiteStyle.Render(fmt.Sprintf("%s", file.Status)))
			result.WriteString("  ")
			result.WriteString(filenameStyle.Render(file.Filename))
			result.WriteString("\n")
		}
		result.WriteString(boxStyle.Render("│"))
		result.WriteString("\n")
	}

	// Display unpushed commits
	if len(unpushedCommits) > 0 {
		result.WriteString(boxStyle.Render("├─ "))
		result.WriteString(localCommitsHeaderStyle.Render("[local commits]"))
		result.WriteString("\n")
		for _, commit := range unpushedCommits {
			result.WriteString(boxStyle.Render("│  "))
			result.WriteString(infoStyle.Render(commit.ShortHash))
			result.WriteString(infoStyle.Render(" • "))
			// Truncate commit message to prevent line wrapping (max 80 chars)
			msg := commit.Message
			if len(msg) > 80 {
				msg = msg[:77] + "..."
			}
			result.WriteString(whiteStyle.Render(msg))
			result.WriteString(infoStyle.Render(fmt.Sprintf(" • %s", commit.RelativeTime)))
			result.WriteString("\n")
		}
		result.WriteString(boxStyle.Render("│"))
		result.WriteString("\n")
	}

	// Display base commit at bottom
	if hasBase {
		result.WriteString(boxStyle.Render("┴  "))
		result.WriteString(originBranchStyle.Render(fmt.Sprintf("[origin/%s]", branch)))
		result.WriteString(infoStyle.Render(" "))
		result.WriteString(infoStyle.Render(baseCommit.ShortHash))
		result.WriteString(infoStyle.Render(" • "))
		// Truncate commit message to prevent line wrapping (max 80 chars)
		msg := baseCommit.Message
		if len(msg) > 80 {
			msg = msg[:77] + "..."
		}
		result.WriteString(whiteStyle.Render(msg))
		result.WriteString(infoStyle.Render(fmt.Sprintf(" • %s", baseCommit.RelativeTime)))
		result.WriteString("\n")
	}

	return result.String(), nil
}

// GetColoredStatus returns a colored, human-readable git status (legacy)
func GetColoredStatus() (string, error) {
	return GetEnhancedStatus()
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

// BranchInfo represents a git branch with metadata
type BranchInfo struct {
	Name       string
	Current    bool
	LastCommit string
	Upstream   string
	Author     string
	Date       string
	Ahead      int
	Behind     int
}

// GetBranches returns a list of all branches with metadata
func GetBranches() ([]BranchInfo, error) {
	format := "%(HEAD)||%(refname:short)||%(authorname)||%(committerdate:relative)||%(upstream:short)||%(subject)"
	cmd := exec.Command("git", "for-each-ref", "--sort=-creatordate", "refs/heads/", "--format="+format)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	if len(output) == 0 {
		return []BranchInfo{}, nil
	}

	baseBranch := getBaseBranch()
	lines := strings.Split(strings.TrimRight(string(output), "\n"), "\n")
	branches := make([]BranchInfo, 0, len(lines))

	for _, line := range lines {
		parts := strings.SplitN(line, "||", 6)
		if len(parts) < 6 {
			continue
		}

		name := parts[1]
		branch := BranchInfo{
			Current:    parts[0] == "*",
			Name:       name,
			Author:     parts[2],
			Date:       parts[3],
			Upstream:   parts[4],
			LastCommit: parts[5],
		}

		if name != baseBranch {
			branch.Ahead, branch.Behind = branchAheadBehind(name, baseBranch)
		}

		branches = append(branches, branch)
	}

	// Pin current branch to top
	for i, b := range branches {
		if b.Current && i > 0 {
			branches = append([]BranchInfo{b}, append(branches[:i:i], branches[i+1:]...)...)
			break
		}
	}

	return branches, nil
}

// getBaseBranch returns "main" if it exists, otherwise "master"
func getBaseBranch() string {
	cmd := exec.Command("git", "rev-parse", "--verify", "main")
	if cmd.Run() == nil {
		return "main"
	}
	return "master"
}

// branchAheadBehind returns how many commits branch is ahead/behind base
func branchAheadBehind(branch, base string) (ahead, behind int) {
	cmd := exec.Command("git", "rev-list", "--left-right", "--count", base+"..."+branch)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0
	}
	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) != 2 {
		return 0, 0
	}
	behind, _ = strconv.Atoi(parts[0])
	ahead, _ = strconv.Atoi(parts[1])
	return ahead, behind
}

// CreateBranch creates a new branch
func CreateBranch(branchName string) error {
	cmd := exec.Command("git", "branch", branchName)
	return cmd.Run()
}

// SwitchBranch switches to an existing branch
func SwitchBranch(branchName string) error {
	cmd := exec.Command("git", "checkout", branchName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}

// CreateAndSwitchBranch creates and switches to a new branch
func CreateAndSwitchBranch(branchName string) error {
	cmd := exec.Command("git", "checkout", "-b", branchName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}

// DeleteBranch deletes a branch (safe, won't delete if unmerged)
func DeleteBranch(branchName string) error {
	cmd := exec.Command("git", "branch", "-d", branchName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}

// ForceDeleteBranch deletes a branch even if unmerged
func ForceDeleteBranch(branchName string) error {
	cmd := exec.Command("git", "branch", "-D", branchName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}

// CheckoutCommit checks out a specific commit (detached HEAD state)
func CheckoutCommit(commitHash string) error {
	cmd := exec.Command("git", "checkout", commitHash)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}

// IsDetachedHead checks if HEAD is in detached state
func IsDetachedHead() (bool, error) {
	cmd := exec.Command("git", "symbolic-ref", "-q", "HEAD")
	err := cmd.Run()
	// If error, we're in detached HEAD state
	return err != nil, nil
}

// GetCommitDetails returns detailed information about a commit
func GetCommitDetails(commitHash string) (string, error) {
	cmd := exec.Command("git", "show", "--stat", "--pretty=fuller", commitHash)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// ReplayCommits rebases current branch onto the specified branch
func ReplayCommits(ontoBranch string) (string, error) {
	cmd := exec.Command("git", "rebase", ontoBranch)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// ReplayCommitsInteractive starts an interactive rebase
func ReplayCommitsInteractive(ontoBranch string) error {
	// Interactive rebase requires a TTY, so we can't use CombinedOutput
	// We need to use the GIT_SEQUENCE_EDITOR to handle the interactive part
	return fmt.Errorf("interactive rebase must be handled through TUI")
}

// AbortRebase aborts an ongoing rebase operation
func AbortRebase() error {
	cmd := exec.Command("git", "rebase", "--abort")
	return cmd.Run()
}

// ContinueRebase continues a rebase after resolving conflicts
func ContinueRebase() (string, error) {
	cmd := exec.Command("git", "rebase", "--continue")
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// SkipRebaseCommit skips the current commit during rebase
func SkipRebaseCommit() (string, error) {
	cmd := exec.Command("git", "rebase", "--skip")
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// CheckRebaseInProgress checks if a rebase is currently in progress
func CheckRebaseInProgress() (bool, error) {
	// Check for rebase-merge or rebase-apply directory
	cmd := exec.Command("git", "rev-parse", "--git-path", "rebase-merge")
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}

	rebasePath := strings.TrimSpace(string(output))
	if rebasePath == "" {
		return false, nil
	}

	// Check if the directory exists
	cmd = exec.Command("test", "-d", rebasePath)
	err = cmd.Run()
	return err == nil, nil
}

// GetRebaseCommits gets the list of commits that would be replayed
func GetRebaseCommits(ontoBranch string) ([]CommitInfo, error) {
	// Get commits in current branch that are not in ontoBranch
	currentBranch, err := GetCurrentBranch()
	if err != nil {
		return nil, err
	}

	// Use git log to show commits that will be replayed
	args := []string{"log", "--pretty=format:%H|%h|%s|%an|%ai|%ar", fmt.Sprintf("%s..%s", ontoBranch, currentBranch)}
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

// GetMergeBase finds the common ancestor between two branches
func GetMergeBase(branch1, branch2 string) (string, error) {
	cmd := exec.Command("git", "merge-base", branch1, branch2)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// TagInfo represents a git tag with metadata
type TagInfo struct {
	Name         string
	ShortHash    string
	Message      string
	RelativeTime string
	Author       string
}

// CommitWithStats represents a commit with change statistics
type CommitWithStats struct {
	Hash         string
	ShortHash    string
	Message      string
	Author       string
	RelativeTime string
	Additions    int
	Deletions    int
	FilesChanged int
}

// GetTags returns a list of all tags sorted by date (newest first)
func GetTags() ([]TagInfo, error) {
	// Use for-each-ref to get tag info sorted by creatordate descending
	cmd := exec.Command("git", "for-each-ref",
		"--sort=-creatordate",
		"--format=%(refname:short)|%(objectname:short)|%(subject)|%(creatordate:relative)|%(taggername)%(*authorname)",
		"refs/tags")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	if len(output) == 0 {
		return []TagInfo{}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	tags := make([]TagInfo, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 5 {
			continue
		}

		tags = append(tags, TagInfo{
			Name:         parts[0],
			ShortHash:    parts[1],
			Message:      parts[2],
			RelativeTime: parts[3],
			Author:       parts[4],
		})
	}

	return tags, nil
}

// GetMostRecentTag returns the most recent tag on the current branch
func GetMostRecentTag() (string, error) {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetCommitsSinceTag returns commits between a tag and HEAD with stats
func GetCommitsSinceTag(tagName string) ([]CommitWithStats, error) {
	var ref string
	if tagName == "" {
		// If no tag, get all commits
		ref = ""
	} else {
		ref = tagName + "..HEAD"
	}

	// Get commit info
	args := []string{"log", "--no-merges", "--pretty=format:%H|%h|%s|%an|%ar"}
	if ref != "" {
		args = append(args, ref)
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	if len(output) == 0 {
		return []CommitWithStats{}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	commits := make([]CommitWithStats, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 5 {
			continue
		}

		commit := CommitWithStats{
			Hash:         parts[0],
			ShortHash:    parts[1],
			Message:      parts[2],
			Author:       parts[3],
			RelativeTime: parts[4],
		}

		// Get stats for this commit
		statsCmd := exec.Command("git", "diff", "--shortstat", commit.Hash+"^.."+commit.Hash)
		statsOutput, err := statsCmd.Output()
		if err == nil {
			parseCommitStats(string(statsOutput), &commit)
		}

		commits = append(commits, commit)
	}

	return commits, nil
}

// parseCommitStats parses git diff --shortstat output
func parseCommitStats(stats string, commit *CommitWithStats) {
	// Format: " 3 files changed, 10 insertions(+), 5 deletions(-)"
	stats = strings.TrimSpace(stats)
	if stats == "" {
		return
	}

	// Parse files changed
	if idx := strings.Index(stats, " file"); idx > 0 {
		var files int
		fmt.Sscanf(stats[:idx], "%d", &files)
		commit.FilesChanged = files
	}

	// Parse insertions
	if idx := strings.Index(stats, " insertion"); idx > 0 {
		// Find the number before "insertion"
		part := stats[:idx]
		if lastComma := strings.LastIndex(part, ","); lastComma >= 0 {
			part = part[lastComma+1:]
		}
		var ins int
		fmt.Sscanf(strings.TrimSpace(part), "%d", &ins)
		commit.Additions = ins
	}

	// Parse deletions
	if idx := strings.Index(stats, " deletion"); idx > 0 {
		part := stats[:idx]
		if lastComma := strings.LastIndex(part, ","); lastComma >= 0 {
			part = part[lastComma+1:]
		}
		var del int
		fmt.Sscanf(strings.TrimSpace(part), "%d", &del)
		commit.Deletions = del
	}
}

// GetTagDiffStats returns total stats between a tag and HEAD
func GetTagDiffStats(tagName string) (additions, deletions, filesChanged int, err error) {
	var ref string
	if tagName == "" {
		// Compare with empty tree
		ref = "4b825dc642cb6eb9a060e54bf8d69288fbee4904..HEAD"
	} else {
		ref = tagName + "..HEAD"
	}

	cmd := exec.Command("git", "diff", "--shortstat", ref)
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, 0, err
	}

	stats := strings.TrimSpace(string(output))
	if stats == "" {
		return 0, 0, 0, nil
	}

	// Parse files changed
	if idx := strings.Index(stats, " file"); idx > 0 {
		fmt.Sscanf(stats[:idx], "%d", &filesChanged)
	}

	// Parse insertions
	if idx := strings.Index(stats, " insertion"); idx > 0 {
		part := stats[:idx]
		if lastComma := strings.LastIndex(part, ","); lastComma >= 0 {
			part = part[lastComma+1:]
		}
		fmt.Sscanf(strings.TrimSpace(part), "%d", &additions)
	}

	// Parse deletions
	if idx := strings.Index(stats, " deletion"); idx > 0 {
		part := stats[:idx]
		if lastComma := strings.LastIndex(part, ","); lastComma >= 0 {
			part = part[lastComma+1:]
		}
		fmt.Sscanf(strings.TrimSpace(part), "%d", &deletions)
	}

	return additions, deletions, filesChanged, nil
}

// CreateAnnotatedTag creates an annotated tag with a message
func CreateAnnotatedTag(tagName, message string) error {
	cmd := exec.Command("git", "tag", "-a", tagName, "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}

// PushTag pushes a tag to the remote repository
func PushTag(tagName string) (string, error) {
	cmd := exec.Command("git", "push", "origin", tagName)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// DeleteTag deletes a local tag
func DeleteTag(tagName string) error {
	cmd := exec.Command("git", "tag", "-d", tagName)
	return cmd.Run()
}

// GetRemoteURL returns the URL of the origin remote
func GetRemoteURL() (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetTagURL constructs a web URL for a tag based on the remote provider
func GetTagURL(tagName string) (string, error) {
	remoteURL, err := GetRemoteURL()
	if err != nil {
		return "", err
	}

	baseURL := remoteToHTTPS(remoteURL)
	if baseURL == "" {
		return "", fmt.Errorf("could not parse remote URL: %s", remoteURL)
	}

	// Determine provider from the host
	switch {
	case strings.Contains(baseURL, "gitlab.com") || strings.Contains(baseURL, "gitlab."):
		return baseURL + "/-/tags/" + tagName, nil
	case strings.Contains(baseURL, "bitbucket.org") || strings.Contains(baseURL, "bitbucket."):
		return baseURL + "/src/" + tagName, nil
	default:
		// GitHub and other GitHub-compatible hosts (Gitea, Forgejo, etc.)
		return baseURL + "/releases/tag/" + tagName, nil
	}
}

// remoteToHTTPS converts a git remote URL (SSH or HTTPS) to a base HTTPS URL
func remoteToHTTPS(remoteURL string) string {
	// Remove trailing .git
	remoteURL = strings.TrimSuffix(remoteURL, ".git")

	// SSH format: git@host:user/repo
	if strings.HasPrefix(remoteURL, "git@") {
		remoteURL = strings.TrimPrefix(remoteURL, "git@")
		// Replace first : with /
		remoteURL = strings.Replace(remoteURL, ":", "/", 1)
		return "https://" + remoteURL
	}

	// SSH format: ssh://git@host/user/repo
	if strings.HasPrefix(remoteURL, "ssh://") {
		remoteURL = strings.TrimPrefix(remoteURL, "ssh://")
		// Remove user@ prefix
		if atIdx := strings.Index(remoteURL, "@"); atIdx >= 0 {
			remoteURL = remoteURL[atIdx+1:]
		}
		return "https://" + remoteURL
	}

	// Already HTTPS
	if strings.HasPrefix(remoteURL, "https://") || strings.HasPrefix(remoteURL, "http://") {
		return remoteURL
	}

	return ""
}

// TagDetailInfo represents detailed metadata for a single tag
type TagDetailInfo struct {
	Name         string
	FullHash     string
	ShortHash    string
	TaggerName   string
	TaggerEmail  string
	Subject      string
	Body         string
	RelativeTime string
	Date         string
}

// GetTagDetail returns detailed metadata for a specific tag
func GetTagDetail(tagName string) (TagDetailInfo, error) {
	cmd := exec.Command("git", "for-each-ref",
		"--format=%(refname:short)|%(objectname)|%(objectname:short)|%(taggername)|%(taggeremail)|%(subject)|%(body)|%(creatordate:relative)|%(creatordate:iso)",
		fmt.Sprintf("refs/tags/%s", tagName))
	output, err := cmd.Output()
	if err != nil {
		return TagDetailInfo{}, fmt.Errorf("failed to get tag detail: %w", err)
	}

	line := strings.TrimSpace(string(output))
	if line == "" {
		return TagDetailInfo{}, fmt.Errorf("tag '%s' not found", tagName)
	}

	parts := strings.SplitN(line, "|", 9)
	if len(parts) < 9 {
		return TagDetailInfo{}, fmt.Errorf("unexpected tag format for '%s'", tagName)
	}

	return TagDetailInfo{
		Name:         parts[0],
		FullHash:     parts[1],
		ShortHash:    parts[2],
		TaggerName:   parts[3],
		TaggerEmail:  strings.Trim(parts[4], "<>"),
		Subject:      parts[5],
		Body:         strings.TrimSpace(parts[6]),
		RelativeTime: parts[7],
		Date:         parts[8],
	}, nil
}

// GetPreviousTag returns the tag before the given tag, or empty string if none
func GetPreviousTag(tagName string) (string, error) {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0", tagName+"^")
	output, err := cmd.Output()
	if err != nil {
		// No previous tag found (this is the first tag)
		return "", nil
	}
	return strings.TrimSpace(string(output)), nil
}

// GetCommitsBetweenTags returns commits between two tags with stats
func GetCommitsBetweenTags(fromTag, toTag string) ([]CommitWithStats, error) {
	var ref string
	if fromTag == "" {
		// First tag - get all commits up to toTag
		ref = toTag
	} else {
		ref = fromTag + ".." + toTag
	}

	args := []string{"log", "--no-merges", "--pretty=format:%H|%h|%s|%an|%ar", ref}

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	if len(output) == 0 {
		return []CommitWithStats{}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	commits := make([]CommitWithStats, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 5 {
			continue
		}

		commit := CommitWithStats{
			Hash:         parts[0],
			ShortHash:    parts[1],
			Message:      parts[2],
			Author:       parts[3],
			RelativeTime: parts[4],
		}

		// Get stats for this commit
		statsCmd := exec.Command("git", "diff", "--shortstat", commit.Hash+"^.."+commit.Hash)
		statsOutput, err := statsCmd.Output()
		if err == nil {
			parseCommitStats(string(statsOutput), &commit)
		}

		commits = append(commits, commit)
	}

	return commits, nil
}

// GetTagRangeDiffStats returns total stats between two tags
func GetTagRangeDiffStats(fromTag, toTag string) (additions, deletions, filesChanged int, err error) {
	var ref string
	if fromTag == "" {
		// Compare with empty tree
		ref = "4b825dc642cb6eb9a060e54bf8d69288fbee4904.." + toTag
	} else {
		ref = fromTag + ".." + toTag
	}

	cmd := exec.Command("git", "diff", "--shortstat", ref)
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, 0, err
	}

	stats := strings.TrimSpace(string(output))
	if stats == "" {
		return 0, 0, 0, nil
	}

	// Parse files changed
	if idx := strings.Index(stats, " file"); idx > 0 {
		fmt.Sscanf(stats[:idx], "%d", &filesChanged)
	}

	// Parse insertions
	if idx := strings.Index(stats, " insertion"); idx > 0 {
		part := stats[:idx]
		if lastComma := strings.LastIndex(part, ","); lastComma >= 0 {
			part = part[lastComma+1:]
		}
		fmt.Sscanf(strings.TrimSpace(part), "%d", &additions)
	}

	// Parse deletions
	if idx := strings.Index(stats, " deletion"); idx > 0 {
		part := stats[:idx]
		if lastComma := strings.LastIndex(part, ","); lastComma >= 0 {
			part = part[lastComma+1:]
		}
		fmt.Sscanf(strings.TrimSpace(part), "%d", &deletions)
	}

	return additions, deletions, filesChanged, nil
}

// GetLastCommitMessage returns the message of the most recent commit
func GetLastCommitMessage() (string, error) {
	cmd := exec.Command("git", "log", "-1", "--pretty=format:%s")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetCommitMessage returns the full message of a specific commit
func GetCommitMessage(commitHash string) (string, error) {
	cmd := exec.Command("git", "log", "-1", "--pretty=format:%s", commitHash)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// AmendCommitMessage rewrites the last commit message
func AmendCommitMessage(newMessage string) error {
	cmd := exec.Command("git", "commit", "--amend", "-m", newMessage)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}

// RewordCommit rewrites a specific commit message using interactive rebase
// This is more complex and requires setting up an interactive rebase
func RewordCommit(commitHash string, newMessage string) error {
	// This will be handled by the TUI model using a sequence of commands
	return fmt.Errorf("reword via interactive rebase must be handled through TUI")
}
