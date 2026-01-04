package main

import (
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
