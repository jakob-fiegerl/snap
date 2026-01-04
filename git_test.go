package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
