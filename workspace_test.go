package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetWorkspaceDir(t *testing.T) {
	// This test assumes we're in a git repository
	workspaceDir, err := GetWorkspaceDir()
	if err != nil {
		t.Fatalf("GetWorkspaceDir failed: %v", err)
	}

	if workspaceDir == "" {
		t.Fatal("GetWorkspaceDir returned empty string")
	}

	// Check that it ends with "workspaces"
	if filepath.Base(workspaceDir) != "workspaces" {
		t.Errorf("Expected workspace directory to end with 'workspaces', got: %s", workspaceDir)
	}
}

func TestGetWorkspaceMetadataDir(t *testing.T) {
	// This test assumes we're in a git repository
	metadataDir, err := GetWorkspaceMetadataDir()
	if err != nil {
		t.Fatalf("GetWorkspaceMetadataDir failed: %v", err)
	}

	if metadataDir == "" {
		t.Fatal("GetWorkspaceMetadataDir returned empty string")
	}

	// Check that the directory was created
	info, err := os.Stat(metadataDir)
	if err != nil {
		t.Fatalf("Metadata directory was not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("Metadata path is not a directory")
	}
}

func TestSaveAndLoadWorkspaceMetadata(t *testing.T) {
	// Create test metadata
	testMetadata := WorkspaceMetadata{
		Name:       "test-workspace",
		Branch:     "workspace/test-workspace",
		BaseBranch: "main",
		BaseCommit: "abc123",
	}

	// Save metadata
	err := SaveWorkspaceMetadata(testMetadata)
	if err != nil {
		t.Fatalf("SaveWorkspaceMetadata failed: %v", err)
	}

	// Load metadata
	loadedMetadata, err := LoadWorkspaceMetadata("test-workspace")
	if err != nil {
		t.Fatalf("LoadWorkspaceMetadata failed: %v", err)
	}

	// Verify metadata
	if loadedMetadata.Name != testMetadata.Name {
		t.Errorf("Expected name %s, got %s", testMetadata.Name, loadedMetadata.Name)
	}
	if loadedMetadata.Branch != testMetadata.Branch {
		t.Errorf("Expected branch %s, got %s", testMetadata.Branch, loadedMetadata.Branch)
	}
	if loadedMetadata.BaseBranch != testMetadata.BaseBranch {
		t.Errorf("Expected base branch %s, got %s", testMetadata.BaseBranch, loadedMetadata.BaseBranch)
	}
	if loadedMetadata.BaseCommit != testMetadata.BaseCommit {
		t.Errorf("Expected base commit %s, got %s", testMetadata.BaseCommit, loadedMetadata.BaseCommit)
	}

	// Clean up
	metadataPath, _ := GetWorkspaceMetadataPath("test-workspace")
	_ = os.Remove(metadataPath)
}

func TestLoadWorkspaceMetadataNotFound(t *testing.T) {
	// Try to load a workspace that doesn't exist
	_, err := LoadWorkspaceMetadata("nonexistent-workspace")
	if err == nil {
		t.Error("Expected error when loading nonexistent workspace, got nil")
	}
}

func TestListWorkspaces(t *testing.T) {
	// Create test metadata
	testMetadata1 := WorkspaceMetadata{
		Name:       "test-workspace-1",
		Branch:     "workspace/test-workspace-1",
		BaseBranch: "main",
		BaseCommit: "abc123",
	}
	testMetadata2 := WorkspaceMetadata{
		Name:       "test-workspace-2",
		Branch:     "workspace/test-workspace-2",
		BaseBranch: "main",
		BaseCommit: "def456",
	}

	// Save metadata
	_ = SaveWorkspaceMetadata(testMetadata1)
	_ = SaveWorkspaceMetadata(testMetadata2)

	// List workspaces
	workspaces, err := ListWorkspaces()
	if err != nil {
		t.Fatalf("ListWorkspaces failed: %v", err)
	}

	// Check that both workspaces are in the list
	found1 := false
	found2 := false
	for _, ws := range workspaces {
		if ws.Name == "test-workspace-1" {
			found1 = true
		}
		if ws.Name == "test-workspace-2" {
			found2 = true
		}
	}

	if !found1 {
		t.Error("test-workspace-1 not found in workspace list")
	}
	if !found2 {
		t.Error("test-workspace-2 not found in workspace list")
	}

	// Clean up
	metadataPath1, _ := GetWorkspaceMetadataPath("test-workspace-1")
	metadataPath2, _ := GetWorkspaceMetadataPath("test-workspace-2")
	_ = os.Remove(metadataPath1)
	_ = os.Remove(metadataPath2)
}

func TestGetCurrentWorkspace(t *testing.T) {
	// Get current workspace (should be empty if not in a workspace)
	currentWorkspace, err := GetCurrentWorkspace()
	if err != nil {
		t.Fatalf("GetCurrentWorkspace failed: %v", err)
	}

	// We're likely not in a workspace, so it should be empty
	// This test just verifies the function doesn't crash
	_ = currentWorkspace
}
