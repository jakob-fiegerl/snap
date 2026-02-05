package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestRepo creates a temporary git repository for testing
func setupTestRepo(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "snap-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	if err := cmd.Run(); err != nil {
		os.Chdir(originalDir)
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user for tests
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		os.Chdir(originalDir)
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create test file: %v", err)
	}

	exec.Command("git", "add", ".").Run()
	exec.Command("git", "commit", "-m", "Initial commit").Run()

	cleanup := func() {
		os.Chdir(originalDir)
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestGetBranches(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	branches, err := GetBranches()
	if err != nil {
		t.Fatalf("GetBranches failed: %v", err)
	}

	if len(branches) == 0 {
		t.Fatal("Expected at least one branch (main/master)")
	}

	// Check that one branch is marked as current
	foundCurrent := false
	for _, branch := range branches {
		if branch.Current {
			foundCurrent = true
			break
		}
	}

	if !foundCurrent {
		t.Error("No branch marked as current")
	}
}

func TestCreateBranch(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	branchName := "test-branch"
	err := CreateBranch(branchName)
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// Verify branch was created
	branches, err := GetBranches()
	if err != nil {
		t.Fatalf("GetBranches failed: %v", err)
	}

	found := false
	for _, branch := range branches {
		if branch.Name == branchName {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Branch '%s' was not created", branchName)
	}
}

func TestCreateAndSwitchBranch(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	branchName := "feature-branch"
	err := CreateAndSwitchBranch(branchName)
	if err != nil {
		t.Fatalf("CreateAndSwitchBranch failed: %v", err)
	}

	// Verify we're on the new branch
	currentBranch, err := GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	if currentBranch != branchName {
		t.Errorf("Expected current branch to be '%s', got '%s'", branchName, currentBranch)
	}
}

func TestSwitchBranch(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a new branch
	branchName := "switch-test"
	err := CreateBranch(branchName)
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// Switch to it
	err = SwitchBranch(branchName)
	if err != nil {
		t.Fatalf("SwitchBranch failed: %v", err)
	}

	// Verify we're on the new branch
	currentBranch, err := GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	if currentBranch != branchName {
		t.Errorf("Expected current branch to be '%s', got '%s'", branchName, currentBranch)
	}
}

func TestDeleteBranch(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a new branch
	branchName := "delete-test"
	err := CreateBranch(branchName)
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// Delete it
	err = DeleteBranch(branchName)
	if err != nil {
		t.Fatalf("DeleteBranch failed: %v", err)
	}

	// Verify it's gone
	branches, err := GetBranches()
	if err != nil {
		t.Fatalf("GetBranches failed: %v", err)
	}

	for _, branch := range branches {
		if branch.Name == branchName {
			t.Errorf("Branch '%s' should have been deleted", branchName)
		}
	}
}

func TestDeleteBranchWhileOnBranch(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create and switch to a new branch
	branchName := "current-branch"
	err := CreateAndSwitchBranch(branchName)
	if err != nil {
		t.Fatalf("CreateAndSwitchBranch failed: %v", err)
	}

	// Try to delete current branch - should fail
	err = DeleteBranch(branchName)
	if err == nil {
		t.Error("Expected error when deleting current branch, got nil")
	}
}

func TestGetCurrentBranch(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	branch, err := GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	if branch == "" {
		t.Error("Expected non-empty branch name")
	}

	// Common default branches
	validBranches := map[string]bool{
		"main":   true,
		"master": true,
	}

	if !validBranches[branch] {
		t.Logf("Warning: unexpected default branch '%s' (expected 'main' or 'master')", branch)
	}
}

func TestGetRebaseCommits(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a feature branch
	err := CreateAndSwitchBranch("feature")
	if err != nil {
		t.Fatalf("CreateAndSwitchBranch failed: %v", err)
	}

	// Add some commits on feature branch
	for i := 1; i <= 3; i++ {
		testFile := filepath.Join(".", fmt.Sprintf("feature%d.txt", i))
		if err := os.WriteFile(testFile, []byte(fmt.Sprintf("feature %d", i)), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		exec.Command("git", "add", ".").Run()
		exec.Command("git", "commit", "-m", fmt.Sprintf("Feature commit %d", i)).Run()
	}

	// Get the commits that would be replayed onto main/master
	commits, err := GetRebaseCommits("main")
	if err != nil {
		// Try master if main doesn't exist
		commits, err = GetRebaseCommits("master")
		if err != nil {
			t.Fatalf("GetRebaseCommits failed: %v", err)
		}
	}

	if len(commits) != 3 {
		t.Errorf("Expected 3 commits to replay, got %d", len(commits))
	}

	// Verify commits are in correct order (newest first)
	for i, commit := range commits {
		expectedMsg := fmt.Sprintf("Feature commit %d", 3-i)
		if commit.Message != expectedMsg {
			t.Errorf("Commit %d: expected message '%s', got '%s'", i, expectedMsg, commit.Message)
		}
	}
}

func TestReplayCommits(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	// Get the main branch name
	mainBranch, err := GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	// Add a commit to main
	testFile := filepath.Join(".", "main.txt")
	if err := os.WriteFile(testFile, []byte("main content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	exec.Command("git", "add", ".").Run()
	exec.Command("git", "commit", "-m", "Main commit").Run()

	// Create and switch to feature branch from the initial commit
	exec.Command("git", "checkout", "HEAD~1").Run()
	err = CreateAndSwitchBranch("feature")
	if err != nil {
		t.Fatalf("CreateAndSwitchBranch failed: %v", err)
	}

	// Add commits on feature branch
	for i := 1; i <= 2; i++ {
		testFile := filepath.Join(".", fmt.Sprintf("feature%d.txt", i))
		if err := os.WriteFile(testFile, []byte(fmt.Sprintf("feature %d", i)), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		exec.Command("git", "add", ".").Run()
		exec.Command("git", "commit", "-m", fmt.Sprintf("Feature commit %d", i)).Run()
	}

	// Replay feature commits onto main
	output, err := ReplayCommits(mainBranch)
	if err != nil {
		t.Fatalf("ReplayCommits failed: %v\nOutput: %s", err, output)
	}

	// Verify we're still on feature branch
	currentBranch, err := GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	if currentBranch != "feature" {
		t.Errorf("Expected to be on 'feature' branch, got '%s'", currentBranch)
	}

	// Verify the commits are there
	commits, err := GetCommitHistory(5, false, "", "")
	if err != nil {
		t.Fatalf("GetCommitHistory failed: %v", err)
	}

	// Should have at least the feature commits plus main commit
	if len(commits) < 3 {
		t.Errorf("Expected at least 3 commits after replay, got %d", len(commits))
	}
}

func TestCheckRebaseInProgress(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	// Initially, no rebase should be in progress
	inProgress, err := CheckRebaseInProgress()
	if err != nil {
		t.Fatalf("CheckRebaseInProgress failed: %v", err)
	}

	if inProgress {
		t.Error("Expected no rebase in progress")
	}
}

func TestGetBranchesWithShortNames(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create branches with various name lengths to test parsing
	testBranches := []string{"a", "ab", "abc", "feature", "very-long-branch-name"}

	for _, branchName := range testBranches {
		err := CreateBranch(branchName)
		if err != nil {
			t.Fatalf("Failed to create branch '%s': %v", branchName, err)
		}
	}

	// Get all branches
	branches, err := GetBranches()
	if err != nil {
		t.Fatalf("GetBranches failed: %v", err)
	}

	// Verify all test branches are present with correct names
	foundBranches := make(map[string]bool)
	for _, branch := range branches {
		foundBranches[branch.Name] = true
	}

	for _, expected := range testBranches {
		if !foundBranches[expected] {
			t.Errorf("Branch '%s' not found in results. Found branches: %v", expected, foundBranches)
		}
	}
}

func TestGetMergeBase(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	// Get the main branch
	mainBranch, err := GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	// Create a feature branch
	err = CreateAndSwitchBranch("feature")
	if err != nil {
		t.Fatalf("CreateAndSwitchBranch failed: %v", err)
	}

	// Add a commit
	testFile := filepath.Join(".", "feature.txt")
	if err := os.WriteFile(testFile, []byte("feature"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	exec.Command("git", "add", ".").Run()
	exec.Command("git", "commit", "-m", "Feature commit").Run()

	// Get merge base
	mergeBase, err := GetMergeBase(mainBranch, "feature")
	if err != nil {
		t.Fatalf("GetMergeBase failed: %v", err)
	}

	if mergeBase == "" {
		t.Error("Expected non-empty merge base")
	}

	// Verify it's a valid commit hash (40 characters)
	if len(mergeBase) != 40 {
		t.Errorf("Expected merge base to be 40 characters, got %d", len(mergeBase))
	}
}

func TestCheckoutCommit(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	// Get the initial commit hash
	commits, err := GetCommitHistory(1, false, "", "")
	if err != nil {
		t.Fatalf("GetCommitHistory failed: %v", err)
	}

	if len(commits) == 0 {
		t.Fatal("Expected at least one commit")
	}

	initialCommit := commits[0].Hash

	// Create a new commit
	testFile := filepath.Join(".", "new.txt")
	if err := os.WriteFile(testFile, []byte("new content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	exec.Command("git", "add", ".").Run()
	exec.Command("git", "commit", "-m", "New commit").Run()

	// Checkout the initial commit
	err = CheckoutCommit(initialCommit)
	if err != nil {
		t.Fatalf("CheckoutCommit failed: %v", err)
	}

	// Verify we're in detached HEAD state
	isDetached, err := IsDetachedHead()
	if err != nil {
		t.Fatalf("IsDetachedHead failed: %v", err)
	}

	if !isDetached {
		t.Error("Expected to be in detached HEAD state after checking out a commit")
	}
}

func TestIsDetachedHead(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	// Initially, should not be detached
	isDetached, err := IsDetachedHead()
	if err != nil {
		t.Fatalf("IsDetachedHead failed: %v", err)
	}

	if isDetached {
		t.Error("Expected not to be in detached HEAD state initially")
	}

	// Get a commit to checkout
	commits, err := GetCommitHistory(1, false, "", "")
	if err != nil {
		t.Fatalf("GetCommitHistory failed: %v", err)
	}

	if len(commits) > 0 {
		// Checkout the commit
		err = CheckoutCommit(commits[0].Hash)
		if err != nil {
			t.Fatalf("CheckoutCommit failed: %v", err)
		}

		// Now should be detached
		isDetached, err = IsDetachedHead()
		if err != nil {
			t.Fatalf("IsDetachedHead failed: %v", err)
		}

		if !isDetached {
			t.Error("Expected to be in detached HEAD state after checking out a commit")
		}
	}
}

func TestGetCommitDetails(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	// Get a commit
	commits, err := GetCommitHistory(1, false, "", "")
	if err != nil {
		t.Fatalf("GetCommitHistory failed: %v", err)
	}

	if len(commits) == 0 {
		t.Fatal("Expected at least one commit")
	}

	// Get details
	details, err := GetCommitDetails(commits[0].Hash)
	if err != nil {
		t.Fatalf("GetCommitDetails failed: %v", err)
	}

	if details == "" {
		t.Error("Expected non-empty commit details")
	}

	// Should contain the commit message
	if !strings.Contains(details, commits[0].Message) {
		t.Errorf("Expected details to contain commit message '%s', got: %s", commits[0].Message, details)
	}
}

func TestRemoteToHTTPS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "GitHub SSH",
			input:    "git@github.com:user/repo.git",
			expected: "https://github.com/user/repo",
		},
		{
			name:     "GitHub HTTPS",
			input:    "https://github.com/user/repo.git",
			expected: "https://github.com/user/repo",
		},
		{
			name:     "GitLab SSH",
			input:    "git@gitlab.com:user/repo.git",
			expected: "https://gitlab.com/user/repo",
		},
		{
			name:     "Bitbucket SSH",
			input:    "git@bitbucket.org:user/repo.git",
			expected: "https://bitbucket.org/user/repo",
		},
		{
			name:     "SSH with ssh:// prefix",
			input:    "ssh://git@github.com/user/repo.git",
			expected: "https://github.com/user/repo",
		},
		{
			name:     "HTTPS without .git suffix",
			input:    "https://github.com/user/repo",
			expected: "https://github.com/user/repo",
		},
		{
			name:     "HTTP URL",
			input:    "http://github.com/user/repo.git",
			expected: "http://github.com/user/repo",
		},
		{
			name:     "Unknown format",
			input:    "ftp://example.com/repo",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := remoteToHTTPS(tt.input)
			if result != tt.expected {
				t.Errorf("remoteToHTTPS(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetTagDetail(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create an annotated tag
	err := CreateAnnotatedTag("v1.0.0", "Release v1.0.0")
	if err != nil {
		t.Fatalf("CreateAnnotatedTag failed: %v", err)
	}

	detail, err := GetTagDetail("v1.0.0")
	if err != nil {
		t.Fatalf("GetTagDetail failed: %v", err)
	}

	if detail.Name != "v1.0.0" {
		t.Errorf("Expected tag name 'v1.0.0', got '%s'", detail.Name)
	}

	if detail.Subject != "Release v1.0.0" {
		t.Errorf("Expected subject 'Release v1.0.0', got '%s'", detail.Subject)
	}

	if detail.ShortHash == "" {
		t.Error("Expected non-empty short hash")
	}

	if detail.TaggerName == "" {
		t.Error("Expected non-empty tagger name")
	}

	if detail.RelativeTime == "" {
		t.Error("Expected non-empty relative time")
	}
}

func TestGetTagDetailNotFound(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	_, err := GetTagDetail("nonexistent-tag")
	if err == nil {
		t.Error("Expected error for nonexistent tag")
	}
}

func TestGetPreviousTag(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create first tag
	err := CreateAnnotatedTag("v1.0.0", "Release v1.0.0")
	if err != nil {
		t.Fatalf("CreateAnnotatedTag failed: %v", err)
	}

	// Add a commit and create second tag
	testFile := filepath.Join(".", "new.txt")
	if err := os.WriteFile(testFile, []byte("new content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	exec.Command("git", "add", ".").Run()
	exec.Command("git", "commit", "-m", "New commit").Run()

	err = CreateAnnotatedTag("v2.0.0", "Release v2.0.0")
	if err != nil {
		t.Fatalf("CreateAnnotatedTag failed: %v", err)
	}

	// Previous tag of v2.0.0 should be v1.0.0
	prevTag, err := GetPreviousTag("v2.0.0")
	if err != nil {
		t.Fatalf("GetPreviousTag failed: %v", err)
	}

	if prevTag != "v1.0.0" {
		t.Errorf("Expected previous tag 'v1.0.0', got '%s'", prevTag)
	}

	// First tag should have no previous tag
	prevTag, err = GetPreviousTag("v1.0.0")
	if err != nil {
		t.Fatalf("GetPreviousTag for first tag failed: %v", err)
	}

	if prevTag != "" {
		t.Errorf("Expected empty previous tag for first tag, got '%s'", prevTag)
	}
}

func TestGetCommitsBetweenTags(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create first tag
	err := CreateAnnotatedTag("v1.0.0", "Release v1.0.0")
	if err != nil {
		t.Fatalf("CreateAnnotatedTag failed: %v", err)
	}

	// Add commits
	for i := 1; i <= 3; i++ {
		testFile := filepath.Join(".", fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(testFile, []byte(fmt.Sprintf("content %d", i)), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		exec.Command("git", "add", ".").Run()
		exec.Command("git", "commit", "-m", fmt.Sprintf("Commit %d", i)).Run()
	}

	// Create second tag
	err = CreateAnnotatedTag("v2.0.0", "Release v2.0.0")
	if err != nil {
		t.Fatalf("CreateAnnotatedTag failed: %v", err)
	}

	// Get commits between tags
	commits, err := GetCommitsBetweenTags("v1.0.0", "v2.0.0")
	if err != nil {
		t.Fatalf("GetCommitsBetweenTags failed: %v", err)
	}

	if len(commits) != 3 {
		t.Errorf("Expected 3 commits between tags, got %d", len(commits))
	}

	// Test with empty fromTag (first tag)
	allCommits, err := GetCommitsBetweenTags("", "v1.0.0")
	if err != nil {
		t.Fatalf("GetCommitsBetweenTags with empty fromTag failed: %v", err)
	}

	if len(allCommits) == 0 {
		t.Error("Expected at least one commit for first tag range")
	}
}

func TestGetTagRangeDiffStats(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create first tag
	err := CreateAnnotatedTag("v1.0.0", "Release v1.0.0")
	if err != nil {
		t.Fatalf("CreateAnnotatedTag failed: %v", err)
	}

	// Add a file
	testFile := filepath.Join(".", "stats.txt")
	if err := os.WriteFile(testFile, []byte("line1\nline2\nline3\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	exec.Command("git", "add", ".").Run()
	exec.Command("git", "commit", "-m", "Add stats file").Run()

	// Create second tag
	err = CreateAnnotatedTag("v2.0.0", "Release v2.0.0")
	if err != nil {
		t.Fatalf("CreateAnnotatedTag failed: %v", err)
	}

	additions, deletions, filesChanged, err := GetTagRangeDiffStats("v1.0.0", "v2.0.0")
	if err != nil {
		t.Fatalf("GetTagRangeDiffStats failed: %v", err)
	}

	if additions == 0 {
		t.Error("Expected non-zero additions")
	}

	if filesChanged == 0 {
		t.Error("Expected non-zero files changed")
	}

	// Deletions should be 0 since we only added a file
	if deletions != 0 {
		t.Errorf("Expected 0 deletions, got %d", deletions)
	}
}

func TestGetTagURL(t *testing.T) {
	_, cleanup := setupTestRepo(t)
	defer cleanup()

	tests := []struct {
		name        string
		remoteURL   string
		tagName     string
		expectedURL string
	}{
		{
			name:        "GitHub",
			remoteURL:   "git@github.com:user/repo.git",
			tagName:     "v1.0.0",
			expectedURL: "https://github.com/user/repo/releases/tag/v1.0.0",
		},
		{
			name:        "GitLab",
			remoteURL:   "git@gitlab.com:user/repo.git",
			tagName:     "v2.0.0",
			expectedURL: "https://gitlab.com/user/repo/-/tags/v2.0.0",
		},
		{
			name:        "Bitbucket",
			remoteURL:   "git@bitbucket.org:user/repo.git",
			tagName:     "v3.0.0",
			expectedURL: "https://bitbucket.org/user/repo/src/v3.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the remote URL
			exec.Command("git", "remote", "remove", "origin").Run()
			err := exec.Command("git", "remote", "add", "origin", tt.remoteURL).Run()
			if err != nil {
				t.Fatalf("Failed to set remote: %v", err)
			}

			url, err := GetTagURL(tt.tagName)
			if err != nil {
				t.Fatalf("GetTagURL failed: %v", err)
			}

			if url != tt.expectedURL {
				t.Errorf("GetTagURL(%q) = %q, want %q", tt.tagName, url, tt.expectedURL)
			}
		})
	}
}
