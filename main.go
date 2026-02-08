package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const version = "1.0.0"

func printHelp() {
	help := `Snap - AI-Powered Git Snapshot Tool

Usage: snap <command> [options]

Commands:
    init              Initialize a new repository
    save [message]    Save changes with AI-generated or custom message
    status            Show repository status and changes
    sync              Smart push/pull with remote
    stack             Show commit history as a visual timeline
    branch            Manage branches
    replay <branch>   Replay commits onto another branch (rebase)
    reword [commit]   Reword a commit message
    tag               Manage tags
    space             Manage workspaces

    help, --help      Show this help message
    version           Show version information

Run 'snap <command> --help' for more information on a command.
`
	fmt.Println(help)
}

func printVersion() {
	fmt.Printf("Snap version %s\n", version)
}

func hasHelpFlag() bool {
	for _, arg := range os.Args[2:] {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func printInitHelp() {
	fmt.Println(`Usage: snap init

Initialize a new git repository in the current directory.

Example:
  snap init`)
}

func printStatusHelp() {
	fmt.Println(`Usage: snap status

Show repository status including current branch, last commit, and all changes.

Displays:
  - Current branch name
  - Last commit information (hash, message, date)
  - Staged changes (ready to commit)
  - Unstaged changes (modified but not staged)
  - Untracked files (new files not in git)

Example:
  snap status`)
}

func printSaveHelp() {
	fmt.Println(`Usage: snap save [MESSAGE] [OPTIONS]

Save changes with an AI-generated or custom commit message.

Options:
  --seed <number>     Set the seed for reproducible AI messages (default: 42)
  --message, -m       Custom commit message (alternative to positional argument)

Examples:
  snap save                    Save with AI-generated message
  snap save "fix: bug"         Save with custom message
  snap save -m "fix: bug"      Save with custom message (flag style)
  snap save --seed 123         Use a custom seed for AI generation`)
}

func printSyncHelp() {
	fmt.Println(`Usage: snap sync [OPTIONS]

Smart push/pull - sync with remote repository.

Options:
  --from    Only pull changes from remote (skip push)

Examples:
  snap sync          Push and pull changes automatically
  snap sync --from   Only pull changes from remote`)
}

func printStackHelp() {
	fmt.Println(`Usage: snap stack [FILE] [OPTIONS]

Show commit history as a visual timeline.

Options:
  --all       Include all branches
  --mine      Show only your commits
  --plain     Non-interactive mode (for piping/scripts)

Examples:
  snap stack               Interactive commit history viewer
  snap stack --all         Include all branches
  snap stack --mine        Show only your commits
  snap stack --plain       Non-interactive mode
  snap stack README.md     Show history for a specific file`)
}

func printBranchHelp() {
	fmt.Println(`Usage: snap branch [SUBCOMMAND] [OPTIONS]

Manage branches - list, create, switch, or delete.

Subcommands:
  new, create       Create and switch to a new branch
  switch, checkout   Switch to an existing branch
  delete, remove     Delete a branch

Examples:
  snap branch                  List all branches (interactive)
  snap branch new feature      Create and switch to 'feature' branch
  snap branch switch main      Switch to 'main' branch
  snap branch delete feature   Delete 'feature' branch`)
}

func printReplayHelp() {
	fmt.Println(`Usage: snap replay <branch> [OPTIONS]

Replay commits onto another branch (rebase).

Options:
  --interactive, -i   Interactive replay (not yet implemented)

Examples:
  snap replay main       Replay current branch commits onto main
  snap replay main -i    Interactive replay`)
}

func printTagHelp() {
	fmt.Println(`Usage: snap tag [SUBCOMMAND]

Manage tags - list, inspect, diff, or create.

Subcommands:
  inspect <tag>       Inspect a tag (commits, stats, metadata)
  diff                Show commits since last tag
  create <version>    Create and push a new annotated tag

Examples:
  snap tag                     List all tags interactively
  snap tag inspect v1.0.0      Inspect a specific tag
  snap tag diff                Show commits since last tag
  snap tag create v1.0.0       Create and push a new tag`)
}

func printRewordHelp() {
	fmt.Println(`Usage: snap reword [COMMIT]

Reword a commit message - edit the most recent commit or a specific one.

Examples:
  snap reword                  Reword the most recent commit
  snap reword abc123           Reword a specific commit (hash)

Interactive controls:
  Enter               Confirm and apply the new message
  Esc                 Cancel and exit`)
}

func printSpaceHelp() {
	fmt.Println(`Usage: snap space [SUBCOMMAND] [OPTIONS]

Manage workspaces - create isolated development environments using git worktrees.

Subcommands:
  new <name>              Create a new workspace
  switch <name>           Switch to an existing workspace
  list                    List all workspaces
  merge <name> [--into]   Merge a workspace

Examples:
  snap space new my-feature              Create workspace from current branch
  snap space new my-feature --from main  Create workspace from main
  snap space switch my-feature           Switch to workspace
  snap space list                        List all workspaces
  snap space merge my-feature            Merge workspace into base branch
  snap space merge my-feature --into main  Merge workspace into main`)
}

func main() {
	seed := 42

	// Parse arguments
	if len(os.Args) == 1 {
		printHelp()
		os.Exit(0)
	}

	command := os.Args[1]

	// Handle commands
	switch command {
	case "help", "--help", "-h":
		printHelp()
		os.Exit(0)

	case "version", "--version", "-v":
		printVersion()
		os.Exit(0)

	case "init":
		if hasHelpFlag() {
			printInitHelp()
			os.Exit(0)
		}
		// Check if already a git repository
		if IsGitRepository() {
			fmt.Println("Error: already a git repository")
			fmt.Println("Use 'snap change' to see what's changed")
			os.Exit(1)
		}

		// Initialize repository
		fmt.Println("📸 Initializing new repository...")
		output, err := InitRepository()
		if err != nil {
			fmt.Printf("Error: failed to initialize repository: %v\n", err)
			fmt.Println(output)
			os.Exit(1)
		}

		fmt.Println("✓ Repository initialized!")
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Create some files")
		fmt.Println("  2. Run 'snap status' to see what's new")
		fmt.Println("  3. Run 'snap save \"Initial commit\"' to save your work")
		os.Exit(0)

	case "status":
		if hasHelpFlag() {
			printStatusHelp()
			os.Exit(0)
		}
		status, err := GetEnhancedStatus()
		if err != nil {
			fmt.Printf("Error: failed to get status: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(status)
		os.Exit(0)

	case "sync":
		if hasHelpFlag() {
			printSyncHelp()
			os.Exit(0)
		}
		// Check for --from flag (pull only)
		pullOnly := false
		for i := 2; i < len(os.Args); i++ {
			if os.Args[i] == "--from" {
				pullOnly = true
				break
			}
		}

		// Run the TUI
		p := tea.NewProgram(initialSyncModel(pullOnly))
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)

	case "stack":
		if hasHelpFlag() {
			printStackHelp()
			os.Exit(0)
		}
		// Parse flags
		allBranches := false
		mineOnly := false
		filePath := ""
		limit := 50 // Default to 50 commits for interactive mode
		plainMode := false

		for i := 2; i < len(os.Args); i++ {
			arg := os.Args[i]
			if arg == "--all" {
				allBranches = true
			} else if arg == "--mine" {
				mineOnly = true
			} else if arg == "--plain" {
				plainMode = true
				limit = 20 // Smaller limit for plain mode
			} else if !strings.HasPrefix(arg, "-") {
				// Assume it's a file path
				filePath = arg
			}
		}

		// Check if we should use plain mode (non-interactive)
		if plainMode {
			// Get git user name for --mine filter
			author := ""
			if mineOnly {
				// We need to get the author name, but that requires exec which we're avoiding
				// For now, just skip author filtering in plain mode
				author = ""
			}

			// Get commit history
			commits, err := GetCommitHistory(limit, allBranches, author, filePath)
			if err != nil {
				fmt.Printf("Error: failed to get commit history: %v\n", err)
				os.Exit(1)
			}

			if len(commits) == 0 {
				fmt.Println("No commits yet")
				os.Exit(0)
			}

			// Render the stack (non-interactive)
			for i, commit := range commits {
				fmt.Printf("● %s %s\n", commit.RelativeTime, commit.Message)
				fmt.Printf("  %s by %s\n", commit.ShortHash, commit.Author)
				if i < len(commits)-1 {
					fmt.Println("│")
				}
			}
			os.Exit(0)
		}

		// Run the interactive TUI with panic recovery
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "Fatal error in interactive mode: %v\n", r)
				fmt.Fprintf(os.Stderr, "Tip: Use 'snap stack --plain' for non-interactive mode\n\n")
				os.Exit(1)
			}
		}()

		p := tea.NewProgram(initialStackModel(limit, allBranches, mineOnly, filePath), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			// If TUI fails, fall back to plain mode
			fmt.Fprintf(os.Stderr, "Interactive mode failed: %v\n", err)
			fmt.Fprintf(os.Stderr, "Tip: Use 'snap stack --plain' for non-interactive mode\n\n")
			os.Exit(1)
		}
		os.Exit(0)

	case "branch":
		if hasHelpFlag() {
			printBranchHelp()
			os.Exit(0)
		}
		// Parse subcommand and arguments
		mode := "list" // Default to list mode
		branchName := ""

		if len(os.Args) > 2 {
			subcommand := os.Args[2]
			switch subcommand {
			case "new", "create":
				mode = "new"
				if len(os.Args) > 3 {
					branchName = os.Args[3]
				}
			case "switch", "checkout":
				mode = "switch"
				if len(os.Args) > 3 {
					branchName = os.Args[3]
				} else {
					fmt.Println("Error: branch name required for switch")
					fmt.Println("Usage: snap branch switch <branch-name>")
					os.Exit(1)
				}
			case "delete", "remove":
				mode = "delete"
				if len(os.Args) > 3 {
					branchName = os.Args[3]
				} else {
					fmt.Println("Error: branch name required for delete")
					fmt.Println("Usage: snap branch delete <branch-name>")
					os.Exit(1)
				}
			default:
				fmt.Printf("Error: unknown subcommand '%s'\n", subcommand)
				fmt.Println("\nValid subcommands: new, switch, delete")
				fmt.Println("Or run 'snap branch' to list branches interactively")
				os.Exit(1)
			}
		}

		// Run the TUI
		p := tea.NewProgram(initialBranchModel(mode, branchName), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)

	case "replay":
		if hasHelpFlag() {
			printReplayHelp()
			os.Exit(0)
		}
		// Parse arguments
		if len(os.Args) < 3 {
			fmt.Println("Error: target branch required")
			fmt.Println("Usage: snap replay <branch> [--interactive|-i]")
			fmt.Println("\nExample:")
			fmt.Println("  snap replay main       # Replay current branch onto main")
			fmt.Println("  snap replay main -i    # Interactive replay")
			os.Exit(1)
		}

		ontoBranch := os.Args[2]
		interactive := false

		// Check for interactive flag
		for i := 3; i < len(os.Args); i++ {
			if os.Args[i] == "--interactive" || os.Args[i] == "-i" {
				interactive = true
				break
			}
		}

		if interactive {
			fmt.Println("Error: interactive replay not yet implemented")
			fmt.Println("Use 'snap replay <branch>' for non-interactive replay")
			os.Exit(1)
		}

		// Run the TUI
		p := tea.NewProgram(initialReplayModel(ontoBranch, interactive), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)

	case "reword":
		if hasHelpFlag() {
			printRewordHelp()
			os.Exit(0)
		}
		// Parse arguments
		commitHash := ""

		if len(os.Args) > 2 && !strings.HasPrefix(os.Args[2], "-") {
			commitHash = os.Args[2]
		}

		// Run the TUI
		p := tea.NewProgram(initialRewordModel(commitHash))
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)

	case "tag":
		if hasHelpFlag() {
			printTagHelp()
			os.Exit(0)
		}
		// Parse subcommand
		if len(os.Args) > 2 {
			subcommand := os.Args[2]
			switch subcommand {
			case "inspect":
				// Inspect a specific tag
				if len(os.Args) < 4 {
					fmt.Println("Error: tag name required")
					fmt.Println("Usage: snap tag inspect <tag>")
					fmt.Println("\nExample:")
					fmt.Println("  snap tag inspect v1.0.0")
					os.Exit(1)
				}
				tagName := os.Args[3]
				p := tea.NewProgram(initialTagsInspectModel(tagName), tea.WithAltScreen())
				if _, err := p.Run(); err != nil {
					fmt.Printf("Error: %v\n", err)
					os.Exit(1)
				}
				os.Exit(0)

			case "diff":
				// Show diff since last tag
				p := tea.NewProgram(initialTagsDiffModel(), tea.WithAltScreen())
				if _, err := p.Run(); err != nil {
					fmt.Printf("Error: %v\n", err)
					os.Exit(1)
				}
				os.Exit(0)

			case "create":
				// Create a new tag
				if len(os.Args) < 4 {
					fmt.Println("Error: tag name required")
					fmt.Println("Usage: snap tag create <version>")
					fmt.Println("\nExample:")
					fmt.Println("  snap tag create v1.0.0")
					os.Exit(1)
				}
				tagName := os.Args[3]
				p := tea.NewProgram(initialTagsCreateModel(tagName), tea.WithAltScreen())
				if _, err := p.Run(); err != nil {
					fmt.Printf("Error: %v\n", err)
					os.Exit(1)
				}
				os.Exit(0)

			default:
				fmt.Printf("Error: unknown subcommand '%s'\n", subcommand)
				fmt.Println("\nValid subcommands: inspect, diff, create")
				fmt.Println("Or run 'snap tag' to list all tags")
				os.Exit(1)
			}
		}

		// No subcommand - run the tags list TUI
		p := tea.NewProgram(initialTagsModel(), tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		// If a tag was selected via Enter, launch the inspect view
		if tm, ok := finalModel.(tagsModel); ok && tm.selectedTag != "" {
			ip := tea.NewProgram(initialTagsInspectModel(tm.selectedTag), tea.WithAltScreen())
			if _, err := ip.Run(); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		}
		os.Exit(0)

	case "space":
		if hasHelpFlag() {
			printSpaceHelp()
			os.Exit(0)
		}

		// Parse subcommand
		if len(os.Args) < 3 {
			fmt.Println("Error: subcommand required")
			fmt.Println("\nValid subcommands: new, switch, list, merge")
			fmt.Println("Run 'snap space --help' for more information")
			os.Exit(1)
		}

		subcommand := os.Args[2]

		switch subcommand {
		case "new":
			// Parse workspace name and options
			if len(os.Args) < 4 {
				fmt.Println("Error: workspace name required")
				fmt.Println("Usage: snap space new <name> [--from branch]")
				fmt.Println("\nExample:")
				fmt.Println("  snap space new my-feature")
				fmt.Println("  snap space new my-feature --from main")
				os.Exit(1)
			}

			workspaceName := os.Args[3]
			baseBranch := ""

			// Parse --from flag
			for i := 4; i < len(os.Args); i++ {
				if os.Args[i] == "--from" && i+1 < len(os.Args) {
					baseBranch = os.Args[i+1]
					break
				}
			}

			fmt.Printf("📂 Creating workspace '%s'...\n", workspaceName)

			if err := CreateWorkspace(workspaceName, baseBranch); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			workspaceDir, _ := GetWorkspaceDir()
			workspacePath := filepath.Join(workspaceDir, workspaceName)

			fmt.Println("✓ Workspace created!")
			fmt.Printf("\nNext steps:\n")
			fmt.Printf("  cd %s\n", workspacePath)
			fmt.Printf("  # Make your changes\n")
			fmt.Printf("  snap save \"your changes\"\n")
			fmt.Printf("  snap space merge %s\n", workspaceName)
			os.Exit(0)

		case "switch":
			// Parse workspace name
			if len(os.Args) < 4 {
				fmt.Println("Error: workspace name required")
				fmt.Println("Usage: snap space switch <name>")
				fmt.Println("\nExample:")
				fmt.Println("  snap space switch my-feature")
				os.Exit(1)
			}

			workspaceName := os.Args[3]
			workspacePath, err := SwitchToWorkspace(workspaceName)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("✓ Workspace found at: %s\n", workspacePath)
			fmt.Printf("\nTo switch to this workspace, run:\n")
			fmt.Printf("  cd %s\n", workspacePath)
			os.Exit(0)

		case "list":
			workspaces, err := ListWorkspaces()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			if len(workspaces) == 0 {
				fmt.Println("No workspaces found")
				fmt.Println("\nCreate a new workspace with:")
				fmt.Println("  snap space new <name>")
				os.Exit(0)
			}

			// Get current workspace
			currentWorkspace, _ := GetCurrentWorkspace()

			fmt.Println("📂 Workspaces:")
			fmt.Println()

			for _, ws := range workspaces {
				prefix := "  "
				if ws.Name == currentWorkspace {
					prefix = "* "
				}
				fmt.Printf("%s%s (from %s)\n", prefix, ws.Name, ws.BaseBranch)
			}

			fmt.Println()
			fmt.Printf("Total: %d workspace(s)\n", len(workspaces))
			os.Exit(0)

		case "merge":
			// Parse workspace name and options
			if len(os.Args) < 4 {
				fmt.Println("Error: workspace name required")
				fmt.Println("Usage: snap space merge <name> [--into branch]")
				fmt.Println("\nExample:")
				fmt.Println("  snap space merge my-feature")
				fmt.Println("  snap space merge my-feature --into main")
				os.Exit(1)
			}

			workspaceName := os.Args[3]
			targetBranch := ""

			// Parse --into flag
			for i := 4; i < len(os.Args); i++ {
				if os.Args[i] == "--into" && i+1 < len(os.Args) {
					targetBranch = os.Args[i+1]
					break
				}
			}

			fmt.Printf("🔀 Merging workspace '%s'...\n", workspaceName)

			if err := MergeWorkspace(workspaceName, targetBranch); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("✓ Workspace merged and deleted!")
			os.Exit(0)

		default:
			fmt.Printf("Error: unknown subcommand '%s'\n", subcommand)
			fmt.Println("\nValid subcommands: new, switch, list, merge")
			fmt.Println("Run 'snap space --help' for more information")
			os.Exit(1)
		}

	case "save":
		if hasHelpFlag() {
			printSaveHelp()
			os.Exit(0)
		}
		var customMessage string

		// Parse save options
		for i := 2; i < len(os.Args); i++ {
			if os.Args[i] == "--seed" && i+1 < len(os.Args) {
				var err error
				seed, err = strconv.Atoi(os.Args[i+1])
				if err != nil {
					fmt.Printf("Error: invalid seed value '%s'\n", os.Args[i+1])
					os.Exit(1)
				}
				i++ // Skip the seed value
			} else if os.Args[i] == "--message" || os.Args[i] == "-m" {
				if i+1 < len(os.Args) {
					customMessage = os.Args[i+1]
					i++ // Skip the message value
				} else {
					fmt.Printf("Error: --message requires a value\n")
					os.Exit(1)
				}
			} else if len(os.Args[i]) > 0 && os.Args[i][0] != '-' && customMessage == "" {
				// Positional argument (custom message)
				customMessage = os.Args[i]
			} else {
				fmt.Printf("Error: unknown option '%s'\n", os.Args[i])
				fmt.Println("\nRun 'snap help' for usage information")
				os.Exit(1)
			}
		}

		p := tea.NewProgram(initialModelWithMessage(seed, customMessage))
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Printf("Error: unknown command '%s'\n", command)
		fmt.Println("\nRun 'snap help' for usage information")
		os.Exit(1)
	}
}
