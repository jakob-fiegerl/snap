package main

import (
	"fmt"
	"os"
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
    changes           Show uncommitted changes
    sync              Smart push/pull with remote
    stack             Show commit history as a visual timeline
    branch            Manage branches
    replay <branch>   Replay commits onto another branch (rebase)
    tags              Manage tags

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

func printChangesHelp() {
	fmt.Println(`Usage: snap changes

Show uncommitted changes (staged and unstaged files).

Example:
  snap changes`)
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

func printTagsHelp() {
	fmt.Println(`Usage: snap tags [SUBCOMMAND]

Manage tags - list, inspect, diff, or create.

Subcommands:
  inspect <tag>       Inspect a tag (commits, stats, metadata)
  diff                Show commits since last tag
  create <version>    Create and push a new annotated tag

Examples:
  snap tags                     List all tags interactively
  snap tags inspect v1.0.0      Inspect a specific tag
  snap tags diff                Show commits since last tag
  snap tags create v1.0.0       Create and push a new tag`)
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
			fmt.Println("Use 'snap changes' to see what's changed")
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
		fmt.Println("  2. Run 'snap changes' to see what's new")
		fmt.Println("  3. Run 'snap save \"Initial commit\"' to save your work")
		os.Exit(0)

	case "changes":
		if hasHelpFlag() {
			printChangesHelp()
			os.Exit(0)
		}
		status, err := GetColoredStatus()
		if err != nil {
			fmt.Printf("Error: failed to get status: %v\n", err)
			os.Exit(1)
		}

		if status == "" {
			fmt.Println("No changes - everything is clean!")
		} else {
			fmt.Println("Changes:")
			fmt.Print(status)
		}
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
		p := tea.NewProgram(initialSyncModel(pullOnly), tea.WithAltScreen())
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

	case "tags":
		if hasHelpFlag() {
			printTagsHelp()
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
					fmt.Println("Usage: snap tags inspect <tag>")
					fmt.Println("\nExample:")
					fmt.Println("  snap tags inspect v1.0.0")
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
					fmt.Println("Usage: snap tags create <version>")
					fmt.Println("\nExample:")
					fmt.Println("  snap tags create v1.0.0")
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
				fmt.Println("Or run 'snap tags' to list all tags")
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

		p := tea.NewProgram(initialModelWithMessage(seed, customMessage), tea.WithAltScreen())
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
