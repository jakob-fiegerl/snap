package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WorkspaceMetadata represents metadata for a workspace
type WorkspaceMetadata struct {
	Name       string `json:"name"`
	Branch     string `json:"branch"`
	BaseBranch string `json:"base_branch"`
	BaseCommit string `json:"base_commit"`
}

// GetWorkspaceDir returns the path to the workspaces directory
func GetWorkspaceDir() (string, error) {
	// Get git root directory
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}
	gitRoot := strings.TrimSpace(string(output))
	return filepath.Join(gitRoot, "workspaces"), nil
}

// GetWorkspaceMetadataDir returns the path to the workspace metadata directory
func GetWorkspaceMetadataDir() (string, error) {
	// Get git root directory
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}
	gitRoot := strings.TrimSpace(string(output))
	metadataDir := filepath.Join(gitRoot, ".snap", "workspaces")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create metadata directory: %w", err)
	}

	return metadataDir, nil
}

// GetWorkspaceMetadataPath returns the path to a workspace's metadata file
func GetWorkspaceMetadataPath(name string) (string, error) {
	metadataDir, err := GetWorkspaceMetadataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(metadataDir, name+".json"), nil
}

// SaveWorkspaceMetadata saves workspace metadata to disk
func SaveWorkspaceMetadata(metadata WorkspaceMetadata) error {
	metadataPath, err := GetWorkspaceMetadataPath(metadata.Name)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// LoadWorkspaceMetadata loads workspace metadata from disk
func LoadWorkspaceMetadata(name string) (WorkspaceMetadata, error) {
	metadataPath, err := GetWorkspaceMetadataPath(name)
	if err != nil {
		return WorkspaceMetadata{}, err
	}

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return WorkspaceMetadata{}, fmt.Errorf("workspace '%s' not found", name)
	}

	var metadata WorkspaceMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return WorkspaceMetadata{}, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return metadata, nil
}

// CreateWorkspace creates a new workspace using git worktree
func CreateWorkspace(name, baseBranch string) error {
	// Validate workspace name
	if name == "" {
		return fmt.Errorf("workspace name cannot be empty")
	}

	// Check if workspace already exists
	metadataPath, err := GetWorkspaceMetadataPath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(metadataPath); err == nil {
		return fmt.Errorf("workspace '%s' already exists", name)
	}

	// Get current branch if baseBranch is empty
	if baseBranch == "" {
		baseBranch, err = GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}
	}

	// Get base commit hash
	cmd := exec.Command("git", "rev-parse", baseBranch)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("branch '%s' not found", baseBranch)
	}
	baseCommit := strings.TrimSpace(string(output))

	// Create workspace branch name
	branchName := "workspace/" + name

	// Check if branch already exists
	cmd = exec.Command("git", "rev-parse", "--verify", branchName)
	if err := cmd.Run(); err == nil {
		return fmt.Errorf("branch '%s' already exists", branchName)
	}

	// Get workspace directory
	workspaceDir, err := GetWorkspaceDir()
	if err != nil {
		return err
	}

	// Create workspaces directory if it doesn't exist
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return fmt.Errorf("failed to create workspaces directory: %w", err)
	}

	workspacePath := filepath.Join(workspaceDir, name)

	// Check if directory already exists
	if _, err := os.Stat(workspacePath); err == nil {
		return fmt.Errorf("directory '%s' already exists", workspacePath)
	}

	// Create git worktree
	cmd = exec.Command("git", "worktree", "add", "-b", branchName, workspacePath, baseBranch)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w\n%s", err, string(output))
	}

	// Save metadata
	metadata := WorkspaceMetadata{
		Name:       name,
		Branch:     branchName,
		BaseBranch: baseBranch,
		BaseCommit: baseCommit,
	}

	if err := SaveWorkspaceMetadata(metadata); err != nil {
		// Try to clean up the worktree if metadata save fails
		_ = exec.Command("git", "worktree", "remove", workspacePath).Run()
		_ = exec.Command("git", "branch", "-D", branchName).Run()
		return err
	}

	return nil
}

// ListWorkspaces returns a list of all workspaces
func ListWorkspaces() ([]WorkspaceMetadata, error) {
	metadataDir, err := GetWorkspaceMetadataDir()
	if err != nil {
		return nil, err
	}

	// Read all metadata files
	entries, err := os.ReadDir(metadataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []WorkspaceMetadata{}, nil
		}
		return nil, fmt.Errorf("failed to read metadata directory: %w", err)
	}

	var workspaces []WorkspaceMetadata
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".json")
		metadata, err := LoadWorkspaceMetadata(name)
		if err != nil {
			// Skip invalid metadata files
			continue
		}

		workspaces = append(workspaces, metadata)
	}

	return workspaces, nil
}

// SwitchToWorkspace switches to a workspace by changing directory
func SwitchToWorkspace(name string) (string, error) {
	// Load metadata to verify workspace exists
	metadata, err := LoadWorkspaceMetadata(name)
	if err != nil {
		return "", err
	}

	// Get workspace directory
	workspaceDir, err := GetWorkspaceDir()
	if err != nil {
		return "", err
	}

	workspacePath := filepath.Join(workspaceDir, metadata.Name)

	// Verify the directory exists
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		return "", fmt.Errorf("workspace directory '%s' does not exist", workspacePath)
	}

	return workspacePath, nil
}

// DeleteWorkspace removes a workspace and its worktree
func DeleteWorkspace(name string) error {
	// Load metadata
	metadata, err := LoadWorkspaceMetadata(name)
	if err != nil {
		return err
	}

	// Get workspace directory
	workspaceDir, err := GetWorkspaceDir()
	if err != nil {
		return err
	}

	workspacePath := filepath.Join(workspaceDir, name)

	// Remove worktree
	cmd := exec.Command("git", "worktree", "remove", workspacePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %w\n%s", err, string(output))
	}

	// Delete branch
	cmd = exec.Command("git", "branch", "-D", metadata.Branch)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete branch: %w\n%s", err, string(output))
	}

	// Delete metadata file
	metadataPath, err := GetWorkspaceMetadataPath(name)
	if err != nil {
		return err
	}

	if err := os.Remove(metadataPath); err != nil {
		return fmt.Errorf("failed to remove metadata: %w", err)
	}

	return nil
}

// MergeWorkspace merges a workspace into a target branch
func MergeWorkspace(name, targetBranch string) error {
	// Load metadata
	metadata, err := LoadWorkspaceMetadata(name)
	if err != nil {
		return err
	}

	// Get current branch
	currentBranch, err := GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// If target branch is empty, use the base branch
	if targetBranch == "" {
		targetBranch = metadata.BaseBranch
	}

	// Checkout target branch
	if currentBranch != targetBranch {
		if err := SwitchBranch(targetBranch); err != nil {
			return fmt.Errorf("failed to switch to branch '%s': %w", targetBranch, err)
		}
	}

	// Merge workspace branch
	cmd := exec.Command("git", "merge", metadata.Branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("merge failed: %w\n%s", err, string(output))
	}

	// Delete the workspace after successful merge
	if err := DeleteWorkspace(name); err != nil {
		return fmt.Errorf("merge succeeded but cleanup failed: %w", err)
	}

	return nil
}

// GetCurrentWorkspace returns the name of the current workspace, or empty if not in a workspace
func GetCurrentWorkspace() (string, error) {
	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Get workspace directory
	workspaceDir, err := GetWorkspaceDir()
	if err != nil {
		return "", err
	}

	// Check if current directory is within workspaces directory
	if !strings.HasPrefix(cwd, workspaceDir) {
		return "", nil // Not in a workspace
	}

	// Extract workspace name from path
	relPath, err := filepath.Rel(workspaceDir, cwd)
	if err != nil {
		return "", nil
	}

	// Get the first directory component
	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) == 0 {
		return "", nil
	}

	return parts[0], nil
}
