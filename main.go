package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const version = "1.0.0"

func printHelp() {
	brandStyle := lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	taglineStyle := lipgloss.NewStyle().Foreground(colorMuted)
	sectionStyle := lipgloss.NewStyle().Foreground(colorSecondary).Bold(true)
	cmdStyle := lipgloss.NewStyle().Foreground(colorPrimary)
	argStyle := lipgloss.NewStyle().Foreground(colorMuted)
	descStyle := lipgloss.NewStyle().Foreground(colorText)
	tipStyle := lipgloss.NewStyle().Foreground(colorMuted).Italic(true)

	type row struct {
		name string
		args string
		desc string
	}

	cmds := []row{
		{"init", "", "Initialize a new repository"},
		{"status", "", "Show repository status and changes"},
		{"save", "[message]", "Save changes with AI-generated or custom message"},
		{"log", "", "Show commit history as a visual timeline"},
		{"sync", "", "Smart push/pull with remote"},
		{"branch", "", "Manage branches"},
		{"replay", "<branch>", "Replay commits onto another branch"},
		{"reword", "[commit]", "Reword a commit message"},
		{"tag", "", "Manage tags"},
		{"mr", "[create]", "Open pull/merge request in browser"},
	}

	meta := []row{
		{"version", "", "Show version information"},
		{"<command>", "--help", "Show help for a specific command"},
	}

	// Compute column width from the longest name+args pair
	colWidth := 0
	for _, r := range append(cmds, meta...) {
		w := len(r.name)
		if r.args != "" {
			w += 1 + len(r.args)
		}
		if w > colWidth {
			colWidth = w
		}
	}
	colWidth += 4 // right padding before description

	renderRow := func(r row) string {
		// Compute padding based on visible text length (before styling)
		plain := r.name
		if r.args != "" {
			plain += " " + r.args
		}
		pad := strings.Repeat(" ", colWidth-len(plain))

		styled := cmdStyle.Render(r.name)
		if r.args != "" {
			styled += " " + argStyle.Render(r.args)
		}
		return "    " + styled + pad + descStyle.Render(r.desc)
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s  %s\n", brandStyle.Render("◆ snap"), taglineStyle.Render("v"+version)))
	sb.WriteString(fmt.Sprintf("  %s\n", taglineStyle.Render("AI-powered Git, simplified.")))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s\n", sectionStyle.Render("COMMANDS")))
	sb.WriteString("\n")
	for _, r := range cmds {
		sb.WriteString(renderRow(r) + "\n")
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s\n", sectionStyle.Render("MORE")))
	sb.WriteString("\n")
	for _, r := range meta {
		sb.WriteString(renderRow(r) + "\n")
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s\n", tipStyle.Render("Tip: run 'snap save' to snapshot your work with an AI commit message.")))
	sb.WriteString("\n")

	fmt.Print(sb.String())
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

// helpRow is a single row in a help section: a name (colored primary) and optional description (muted).
// If name is empty, only desc is rendered (useful for bullet-style info lines).
type helpRow struct {
	name string
	desc string
}

// helpSection is a titled group of rows in a command help screen.
type helpSection struct {
	title string
	rows  []helpRow
}

// renderCommandHelp builds a styled help string for a sub-command.
func renderCommandHelp(name, args, desc string, sections []helpSection) string {
	snapStyle := lipgloss.NewStyle().Foreground(colorMuted)
	cmdStyle := lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	argsStyle := lipgloss.NewStyle().Foreground(colorMuted)
	descStyle := lipgloss.NewStyle().Foreground(colorText)
	sectionStyle := lipgloss.NewStyle().Foreground(colorMuted).Bold(true)
	nameStyle := lipgloss.NewStyle().Foreground(colorText)
	metaStyle := lipgloss.NewStyle().Foreground(colorMuted)

	var sb strings.Builder
	sb.WriteString("\n")

	usageLine := snapStyle.Render("snap") + " " + cmdStyle.Render(name)
	if args != "" {
		usageLine += " " + argsStyle.Render(args)
	}
	sb.WriteString("  " + usageLine + "\n")
	sb.WriteString("\n")
	sb.WriteString("  " + descStyle.Render(desc) + "\n")

	for _, section := range sections {
		sb.WriteString("\n")
		sb.WriteString("  " + sectionStyle.Render(section.title) + "\n")
		sb.WriteString("\n")

		// Compute column width only from rows that have both name and desc
		colWidth := 0
		for _, row := range section.rows {
			if row.name != "" && row.desc != "" && len(row.name) > colWidth {
				colWidth = len(row.name)
			}
		}
		colWidth += 4

		for _, row := range section.rows {
			switch {
			case row.name != "" && row.desc != "":
				pad := strings.Repeat(" ", colWidth-len(row.name))
				sb.WriteString("    " + nameStyle.Render(row.name) + pad + metaStyle.Render(row.desc) + "\n")
			case row.name != "":
				sb.WriteString("    " + nameStyle.Render(row.name) + "\n")
			default:
				sb.WriteString("    " + metaStyle.Render(row.desc) + "\n")
			}
		}
	}

	sb.WriteString("\n")
	return sb.String()
}

func printInitHelp() {
	fmt.Print(renderCommandHelp("init", "", "Initialize a new git repository in the current directory.",
		[]helpSection{
			{title: "EXAMPLE", rows: []helpRow{
				{name: "snap init"},
			}},
		},
	))
}

func printStatusHelp() {
	fmt.Print(renderCommandHelp("status", "", "Show repository status including current branch and all changes.",
		[]helpSection{
			{title: "SHOWS", rows: []helpRow{
				{desc: "Current branch name"},
				{desc: "Last commit information (hash, message, date)"},
				{desc: "Staged changes (ready to commit)"},
				{desc: "Unstaged changes (modified but not staged)"},
				{desc: "Untracked files (new files not in git)"},
			}},
			{title: "EXAMPLE", rows: []helpRow{
				{name: "snap status"},
			}},
		},
	))
}

func printSaveHelp() {
	fmt.Print(renderCommandHelp("save", "[message]", "Save changes with an AI-generated or custom commit message.",
		[]helpSection{
			{title: "OPTIONS", rows: []helpRow{
				{name: "--seed <number>", desc: "Set the seed for reproducible AI messages (default: 42)"},
				{name: "--message, -m", desc: "Custom commit message (alternative to positional argument)"},
			}},
			{title: "EXAMPLES", rows: []helpRow{
				{name: "snap save", desc: "Save with AI-generated message"},
				{name: `snap save "fix: bug"`, desc: "Save with custom message"},
				{name: `snap save -m "fix: bug"`, desc: "Save with custom message (flag style)"},
				{name: "snap save --seed 123", desc: "Use a custom seed for AI generation"},
			}},
		},
	))
}

func printSyncHelp() {
	fmt.Print(renderCommandHelp("sync", "[options]", "Smart push/pull — sync with remote repository.",
		[]helpSection{
			{title: "OPTIONS", rows: []helpRow{
				{name: "--from", desc: "Only pull changes from remote (skip push)"},
			}},
			{title: "EXAMPLES", rows: []helpRow{
				{name: "snap sync", desc: "Push and pull changes automatically"},
				{name: "snap sync --from", desc: "Only pull changes from remote"},
			}},
		},
	))
}

func printStackHelp() {
	fmt.Print(renderCommandHelp("log", "[file] [options]", "Show commit history as a visual timeline.",
		[]helpSection{
			{title: "OPTIONS", rows: []helpRow{
				{name: "--all", desc: "Include all branches"},
				{name: "--mine", desc: "Show only your commits"},
				{name: "--plain", desc: "Non-interactive mode (for piping/scripts)"},
			}},
			{title: "EXAMPLES", rows: []helpRow{
				{name: "snap log", desc: "Interactive commit history viewer"},
				{name: "snap log --all", desc: "Include all branches"},
				{name: "snap log --mine", desc: "Show only your commits"},
				{name: "snap log --plain", desc: "Non-interactive mode"},
				{name: "snap log README.md", desc: "Show history for a specific file"},
			}},
		},
	))
}

func printBranchHelp() {
	fmt.Print(renderCommandHelp("branch", "[subcommand]", "Manage branches — list, create, switch, or delete.",
		[]helpSection{
			{title: "SUBCOMMANDS", rows: []helpRow{
				{name: "new, create", desc: "Create and switch to a new branch"},
				{name: "switch, checkout", desc: "Switch to an existing branch"},
				{name: "delete, remove", desc: "Delete a branch"},
			}},
			{title: "EXAMPLES", rows: []helpRow{
				{name: "snap branch", desc: "List all branches (interactive)"},
				{name: "snap branch new feature", desc: "Create and switch to 'feature' branch"},
				{name: "snap branch switch main", desc: "Switch to 'main' branch"},
				{name: "snap branch delete feature", desc: "Delete 'feature' branch"},
			}},
		},
	))
}

func printReplayHelp() {
	fmt.Print(renderCommandHelp("replay", "<branch> [options]", "Replay commits onto another branch (rebase).",
		[]helpSection{
			{title: "OPTIONS", rows: []helpRow{
				{name: "--interactive, -i", desc: "Interactive replay (not yet implemented)"},
			}},
			{title: "EXAMPLES", rows: []helpRow{
				{name: "snap replay main", desc: "Replay current branch commits onto main"},
				{name: "snap replay main -i", desc: "Interactive replay"},
			}},
		},
	))
}

func printTagHelp() {
	fmt.Print(renderCommandHelp("tag", "[subcommand]", "Manage tags — list, inspect, diff, or create.",
		[]helpSection{
			{title: "SUBCOMMANDS", rows: []helpRow{
				{name: "inspect <tag>", desc: "Inspect a tag (commits, stats, metadata)"},
				{name: "diff", desc: "Show commits since last tag"},
				{name: "create <version>", desc: "Create and push a new annotated tag"},
			}},
			{title: "EXAMPLES", rows: []helpRow{
				{name: "snap tag", desc: "List all tags interactively"},
				{name: "snap tag inspect v1.0.0", desc: "Inspect a specific tag"},
				{name: "snap tag diff", desc: "Show commits since last tag"},
				{name: "snap tag create v1.0.0", desc: "Create and push a new tag"},
			}},
		},
	))
}

func printRewordHelp() {
	fmt.Print(renderCommandHelp("reword", "[commit]", "Reword a commit message — edit the most recent commit or a specific one.",
		[]helpSection{
			{title: "EXAMPLES", rows: []helpRow{
				{name: "snap reword", desc: "Reword the most recent commit"},
				{name: "snap reword abc123", desc: "Reword a specific commit (hash)"},
			}},
			{title: "CONTROLS", rows: []helpRow{
				{name: "Enter", desc: "Confirm and apply the new message"},
				{name: "Esc", desc: "Cancel and exit"},
			}},
		},
	))
}

func printMRHelp() {
	fmt.Print(renderCommandHelp("mr", "[subcommand]", "Open a pull request or merge request in your browser.",
		[]helpSection{
			{title: "SUBCOMMANDS", rows: []helpRow{
				{name: "(none)", desc: "Open existing PR/MR for the current branch"},
				{name: "create", desc: "Open PR/MR creation page for the current branch"},
			}},
			{title: "NOTES", rows: []helpRow{
				{desc: "Supports GitHub, GitLab Cloud, and GitLab self-hosted."},
			}},
			{title: "EXAMPLES", rows: []helpRow{
				{name: "snap mr", desc: "Open current branch's PR/MR"},
				{name: "snap mr create", desc: "Open PR/MR creation page"},
			}},
		},
	))
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

	case "log":
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
				fmt.Fprintf(os.Stderr, "Tip: Use 'snap log --plain' for non-interactive mode\n\n")
				os.Exit(1)
			}
		}()

		p := tea.NewProgram(initialStackModel(limit, allBranches, mineOnly, filePath), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			// If TUI fails, fall back to plain mode
			fmt.Fprintf(os.Stderr, "Interactive mode failed: %v\n", err)
			fmt.Fprintf(os.Stderr, "Tip: Use 'snap log --plain' for non-interactive mode\n\n")
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

		// List mode runs inline (no fullscreen), other modes use alt screen
		var p *tea.Program
		if mode == "list" {
			p = tea.NewProgram(initialBranchModel(mode, branchName))
		} else {
			p = tea.NewProgram(initialBranchModel(mode, branchName), tea.WithAltScreen())
		}
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

	case "mr":
		if hasHelpFlag() {
			printMRHelp()
			os.Exit(0)
		}

		branch, err := GetCurrentBranch()
		if err != nil {
			fmt.Printf("Error: could not get current branch: %v\n", err)
			os.Exit(1)
		}

		subcommand := ""
		if len(os.Args) > 2 {
			subcommand = os.Args[2]
		}

		var mrURL string
		switch subcommand {
		case "create":
			mrURL, err = GetMRCreateURL(branch)
		case "":
			mrURL, err = GetMRViewURL(branch)
		default:
			fmt.Printf("Unknown subcommand: %s\n", subcommand)
			printMRHelp()
			os.Exit(1)
		}

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Opening: %s\n", mrURL)
		if err := OpenURL(mrURL); err != nil {
			fmt.Printf("Could not open browser: %v\n", err)
			fmt.Println("Copy the URL above to open manually.")
			os.Exit(1)
		}
		os.Exit(0)

	default:
		fmt.Printf("Error: unknown command '%s'\n", command)
		fmt.Println("\nRun 'snap help' for usage information")
		os.Exit(1)
	}
}
