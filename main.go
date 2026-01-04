package main

import (
	"fmt"
	"os"
	"strconv"

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
    help, --help, -h    Show this help message
    version, --version  Show version information

OPTIONS:
    --seed <number>     Set the seed for reproducible commit messages (default: 42)
    --message, -m       Custom commit message (alternative to positional argument)

EXAMPLES:
    snap                       Show this help message
    snap save                  Save changes with AI-generated message
    snap save "fix: bug"       Save changes with custom message
    snap save -m "fix: bug"    Save changes with custom message (flag style)
    snap save --seed 123       Use a custom seed for AI generation
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
