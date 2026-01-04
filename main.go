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

USAGE:
    snap [COMMAND] [OPTIONS]
    snap save [MESSAGE] [OPTIONS]

COMMANDS:
    save [MESSAGE]      Save changes with AI-generated or custom commit message
    changes             Show uncommitted changes
    sync [--from]       Smart push/pull - sync with remote repository
    help, --help, -h    Show this help message
    version, --version  Show version information

OPTIONS:
    --seed <number>     Set the seed for reproducible commit messages (default: 42)
    --message, -m       Custom commit message (alternative to positional argument)

EXAMPLES:
    snap                       Show this help message
    snap changes               Show what files have changed
    snap save                  Save changes with AI-generated message
    snap save "fix: bug"       Save changes with custom message
    snap save -m "fix: bug"    Save changes with custom message (flag style)
    snap save --seed 123       Use a custom seed for AI generation
    snap sync                  Push and pull changes automatically
    snap sync --from           Only pull changes from remote
    snap help                  Show this help message
    snap version               Show version information

DESCRIPTION:
    Snap uses Ollama's Phi-4 model to automatically generate meaningful
    conventional commit messages based on your git diff. You can also provide
    your own commit message. It provides an interactive TUI for reviewing
    and confirming commits.

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

		// Check if remote exists
		hasRemote, err := CheckRemoteExists()
		if err != nil {
			fmt.Printf("Error: failed to check remote: %v\n", err)
			os.Exit(1)
		}
		if !hasRemote {
			fmt.Println("Error: no remote repository configured")
			fmt.Println("Add a remote with: git remote add origin <url>")
			os.Exit(1)
		}

		// Check for uncommitted changes
		hasChanges, err := CheckForUncommittedChanges()
		if err != nil {
			fmt.Printf("Error: failed to check for changes: %v\n", err)
			os.Exit(1)
		}
		if hasChanges {
			fmt.Println("Warning: You have uncommitted changes")
			fmt.Println("Run 'snap save' first to commit your changes")
			os.Exit(1)
		}

		// Get current branch
		branch, err := GetCurrentBranch()
		if err != nil {
			fmt.Printf("Error: failed to get current branch: %v\n", err)
			os.Exit(1)
		}

		// Pull changes
		fmt.Println("⏳ Pulling changes from remote...")
		pullOutput, pullErr := PullChanges()
		if pullErr != nil {
			// Check if it's a conflict
			if strings.Contains(pullOutput, "CONFLICT") {
				fmt.Println("✗ Merge conflict detected!")
				fmt.Println("\nConflicting files:")
				fmt.Println(pullOutput)
				fmt.Println("\nResolve conflicts manually, then run 'snap save' to commit")
				os.Exit(1)
			}
			fmt.Printf("Error pulling changes: %v\n", pullErr)
			fmt.Println(pullOutput)
			os.Exit(1)
		}

		if strings.Contains(pullOutput, "Already up to date") {
			fmt.Println("✓ Already up to date with remote")
		} else {
			fmt.Println("✓ Pulled changes successfully")
		}

		// If --from flag, only pull
		if pullOnly {
			os.Exit(0)
		}

		// Push changes
		fmt.Println("⏳ Pushing changes to remote...")

		// Check if upstream is set
		hasUpstream, err := HasUpstreamBranch()
		if err != nil {
			fmt.Printf("Error: failed to check upstream: %v\n", err)
			os.Exit(1)
		}

		var pushOutput string
		var pushErr error

		if !hasUpstream {
			// First push, set upstream
			fmt.Printf("Setting upstream for branch '%s'...\n", branch)
			pushOutput, pushErr = PushWithUpstream(branch)
		} else {
			pushOutput, pushErr = PushChanges()
		}

		if pushErr != nil {
			fmt.Printf("Error pushing changes: %v\n", pushErr)
			fmt.Println(pushOutput)
			os.Exit(1)
		}

		if strings.Contains(pushOutput, "Everything up-to-date") {
			fmt.Println("✓ Everything up-to-date with remote")
		} else {
			fmt.Println("✓ Pushed changes successfully")
		}

		fmt.Println("\n✓ Sync complete!")
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

		// Run the TUI
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
