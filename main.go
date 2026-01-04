package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const version = "1.0.0"

func printHelp() {
	help := `Snap - AI-Powered Git Snapshot Tool

USAGE:
    snap [COMMAND] [OPTIONS]
    snap save [MESSAGE] [OPTIONS]

COMMANDS:
    init                Initialize a new repository
    save [MESSAGE]      Save changes with AI-generated or custom commit message
    changes             Show uncommitted changes
    sync [--from]       Smart push/pull - sync with remote repository
    stack [OPTIONS]     Show commit history as a visual timeline
    branch [OPTIONS]    Manage branches - list, create, switch, or delete
    replay <branch>     Replay commits onto another branch (rebase)
    help, --help, -h    Show this help message
    version, --version  Show version information

OPTIONS:
    --seed <number>     Set the seed for reproducible commit messages (default: 42)
    --message, -m       Custom commit message (alternative to positional argument)

EXAMPLES:
    snap                       Show this help message
    snap init                  Start a new repository
    snap changes               Show what files have changed
    snap save                  Save changes with AI-generated message
    snap save "fix: bug"       Save changes with custom message
    snap save -m "fix: bug"    Save changes with custom message (flag style)
    snap save --seed 123       Use a custom seed for AI generation
    snap sync                  Push and pull changes automatically
    snap sync --from           Only pull changes from remote
    snap stack                 Show commit history
    snap stack --all           Show all branches
    snap stack --mine          Show only your commits
    snap stack README.md       Show history for a specific file
    snap branch                List all branches (interactive)
    snap branch new feature    Create and switch to 'feature' branch
    snap branch switch main    Switch to 'main' branch
    snap branch delete feature Delete 'feature' branch
    snap replay main           Replay current branch commits onto main
    snap replay main -i        Interactive replay (rebase -i)
    snap help                  Show this help message
    snap version               Show version information

DESCRIPTION:
    Snap uses Ollama's Phi-4 model to automatically generate meaningful
    conventional commit messages based on your git diff. You can also provide
    your own commit message. 
    
    When AI generates a message, you can:
        y - Accept the message
        n - Decline and cancel
        e - Edit the message before committing

    Before using AI mode, make sure Ollama is running:
        ollama serve

REQUIREMENTS (AI mode only):
    - Git repository
    - Ollama running locally (http://localhost:11434)
    - Phi-4 model installed (ollama pull phi4)

KEYBOARD SHORTCUTS:
    y/Y         Confirm and commit
    n/N         Cancel commit
    q/Ctrl+C    Quit application
`
	fmt.Println(help)
}

func printVersion() {
	fmt.Printf("Snap version %s\n", version)
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
		// Parse flags
		allBranches := false
		mineOnly := false
		filePath := ""
		limit := 20 // Default to 20 commits

		for i := 2; i < len(os.Args); i++ {
			arg := os.Args[i]
			if arg == "--all" {
				allBranches = true
			} else if arg == "--mine" {
				mineOnly = true
			} else if !strings.HasPrefix(arg, "-") {
				// Assume it's a file path
				filePath = arg
			}
		}

		// Get git user name for --mine filter
		author := ""
		if mineOnly {
			cmd := exec.Command("git", "config", "user.name")
			output, err := cmd.Output()
			if err != nil {
				fmt.Printf("Error: failed to get git user name: %v\n", err)
				os.Exit(1)
			}
			author = strings.TrimSpace(string(output))
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

		// Render the stack
		commitStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
		timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		hashStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		pipeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

		for i, commit := range commits {
			// Show bullet and message
			fmt.Printf("%s %s %s\n",
				commitStyle.Render("●"),
				timeStyle.Render(commit.RelativeTime),
				commit.Message,
			)

			// Show hash on same line
			fmt.Printf("  %s\n", hashStyle.Render(commit.ShortHash))

			// Show pipe between commits (except for last one)
			if i < len(commits)-1 {
				fmt.Println(pipeStyle.Render("│"))
			}
		}

		os.Exit(0)

	case "branch":
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
		p := tea.NewProgram(initialBranchModel(mode, branchName))
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)

	case "replay":
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
		p := tea.NewProgram(initialReplayModel(ontoBranch, interactive))
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)

	case "save":
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

		// Run the TUI (inline, no alt screen)
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
