package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateChecking state = iota
	stateStaging
	stateGettingDiff
	stateGenerating
	stateConfirming
	stateEditing
	stateCommitting
	stateDone
	stateError
)

type model struct {
	state         state
	spinner       spinner.Model
	textInput     textinput.Model
	err           error
	diff          string
	commitMessage string
	originalMsg   string
	cursor        int
	seed          int
	ollamaRunning bool
	stagedChanges bool
	generatedMsg  bool
	userConfirmed bool
	useCustomMsg  bool
}

type checkOllamaMsg struct {
	running bool
}

type stageChangesMsg struct {
	err error
}

type getDiffMsg struct {
	diff string
	err  error
}

type generateMsgMsg struct {
	message string
	err     error
}

type commitMsg struct {
	err error
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			MarginBottom(1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	highlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFF00")).
			Bold(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2)
)

func initialModel(seed int) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	ti := textinput.New()
	ti.Placeholder = "Enter commit message..."
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 60

	return model{
		state:     stateChecking,
		seed:      seed,
		spinner:   s,
		textInput: ti,
	}
}

func initialModelWithMessage(seed int, customMessage string) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	ti := textinput.New()
	ti.Placeholder = "Enter commit message..."
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 60

	if customMessage != "" {
		// Skip AI generation, go straight to staging
		return model{
			state:         stateStaging,
			seed:          seed,
			commitMessage: customMessage,
			useCustomMsg:  true,
			spinner:       s,
			textInput:     ti,
		}
	}

	// No custom message - use AI generation
	return initialModel(seed)
}

func (m model) Init() tea.Cmd {
	if m.useCustomMsg {
		return tea.Batch(m.spinner.Tick, stageChanges)
	}
	return tea.Batch(m.spinner.Tick, checkOllama)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle text input in edit mode
		if m.state == stateEditing {
			switch msg.String() {
			case "ctrl+c", "q":
				// Cancel editing, go back to confirming with original message
				m.commitMessage = m.originalMsg
				m.state = stateConfirming
				return m, nil
			case "enter":
				// Accept edited message
				m.commitMessage = m.textInput.Value()
				if strings.TrimSpace(m.commitMessage) == "" {
					m.state = stateError
					m.err = fmt.Errorf("commit message cannot be empty")
					return m, tea.Quit
				}
				m.state = stateCommitting
				return m, commitChanges(m.commitMessage)
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}

		// Handle confirmation state
		switch msg.String() {
		case "ctrl+c", "q":
			if m.state != stateConfirming {
				return m, tea.Quit
			}
			// In confirming state, treat as decline
			m.state = stateDone
			m.err = fmt.Errorf("commit cancelled")
			return m, tea.Quit

		case "y", "Y":
			if m.state == stateConfirming {
				m.state = stateCommitting
				return m, commitChanges(m.commitMessage)
			}

		case "n", "N":
			if m.state == stateConfirming {
				m.state = stateDone
				m.err = fmt.Errorf("commit cancelled")
				return m, tea.Quit
			}

		case "e", "E":
			if m.state == stateConfirming {
				// Enter edit mode
				m.originalMsg = m.commitMessage
				m.textInput.SetValue(m.commitMessage)
				m.textInput.Focus()
				m.state = stateEditing
				return m, textinput.Blink
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case checkOllamaMsg:
		if !msg.running {
			m.state = stateError
			m.err = fmt.Errorf("Ollama is not running. Please start Ollama first")
			return m, tea.Quit
		}
		m.ollamaRunning = true
		m.state = stateStaging
		return m, stageChanges

	case stageChangesMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, tea.Quit
		}
		m.stagedChanges = true
		m.state = stateGettingDiff
		return m, getDiff

	case getDiffMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, tea.Quit
		}
		if strings.TrimSpace(msg.diff) == "" {
			m.state = stateError
			m.err = fmt.Errorf("no changes to commit")
			return m, tea.Quit
		}
		m.diff = msg.diff

		// If using custom message, skip AI generation
		if m.useCustomMsg {
			m.state = stateConfirming
			return m, nil
		}

		m.state = stateGenerating
		return m, generateMessage(msg.diff, m.seed)

	case generateMsgMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, tea.Quit
		}

		// Validate message is not empty
		cleanMsg := strings.TrimSpace(msg.message)
		if cleanMsg == "" {
			m.state = stateError
			m.err = fmt.Errorf("AI generated an empty commit message. Try again or use custom message")
			return m, tea.Quit
		}

		// Validate message follows conventional commit format
		parts := strings.Split(cleanMsg, ":")
		if len(parts) < 2 || parts[0] == "" {
			m.state = stateError
			m.err = fmt.Errorf("Invalid commit message format: %q. Expected: type: description", cleanMsg)
			return m, tea.Quit
		}

		m.commitMessage = cleanMsg
		m.generatedMsg = true
		m.state = stateConfirming
		return m, nil

	case commitMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, tea.Quit
		}
		m.state = stateDone
		return m, tea.Quit
	}

	return m, nil
}

func (m model) View() string {
	switch m.state {
	case stateChecking:
		if m.useCustomMsg {
			return fmt.Sprintf("%s Staging changes...", m.spinner.View())
		}
		return fmt.Sprintf("%s Checking Ollama...", m.spinner.View())

	case stateStaging:
		return fmt.Sprintf("%s Staging changes...", m.spinner.View())

	case stateGettingDiff:
		return fmt.Sprintf("%s Getting changes...", m.spinner.View())

	case stateGenerating:
		return fmt.Sprintf("%s Generating commit message...", m.spinner.View())

	case stateConfirming:
		// Compact inline confirmation
		msgStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)
		debugStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)
		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

		// Show message type for debugging
		msgType := "Generated"
		if m.useCustomMsg {
			msgType = "Custom"
		}

		return fmt.Sprintf("\n%s %s\n\n%s %s",
			msgStyle.Render(m.commitMessage),
			debugStyle.Render(fmt.Sprintf("[%s message]", msgType)),
			highlightStyle.Render("(y)es, (n)o, (e)dit:"),
			helpStyle.Render(""),
		)

	case stateEditing:
		return fmt.Sprintf("\n%s\n%s",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("Edit commit message (Enter to save, Ctrl+C to cancel):"),
			m.textInput.View(),
		)

	case stateCommitting:
		return fmt.Sprintf("%s Committing...", m.spinner.View())

	case stateDone:
		if m.err != nil {
			return errorStyle.Render(fmt.Sprintf("✗ %s", m.err))
		}
		return successStyle.Render("✓ Changes committed successfully!")

	case stateError:
		return errorStyle.Render(fmt.Sprintf("✗ Error: %s", m.err))
	}

	return ""
}

func checkOllama() tea.Msg {
	running := CheckOllamaRunning()
	return checkOllamaMsg{running: running}
}

func stageChanges() tea.Msg {
	err := StageAllChanges()
	return stageChangesMsg{err: err}
}

func getDiff() tea.Msg {
	diff, err := GetGitDiff()
	return getDiffMsg{diff: diff, err: err}
}

func generateMessage(diff string, seed int) tea.Cmd {
	return func() tea.Msg {
		message, err := GenerateCommitMessage(diff, seed)
		return generateMsgMsg{message: message, err: err}
	}
}

func commitChanges(message string) tea.Cmd {
	return func() tea.Msg {
		err := CommitChanges(message)
		return commitMsg{err: err}
	}
}

// Branch TUI model
type branchState int

const (
	branchStateList branchState = iota
	branchStateCreating
	branchStateSwitching
	branchStateDeleting
	branchStateDone
	branchStateError
)

type branchModel struct {
	state      branchState
	branches   []BranchInfo
	cursor     int
	textInput  textinput.Model
	spinner    spinner.Model
	err        error
	mode       string // "list", "new", "switch", "delete"
	branchName string
	showHelp   bool
}

type getBranchesMsg struct {
	branches []BranchInfo
	err      error
}

type createBranchMsg struct {
	err error
}

type switchBranchMsg struct {
	err error
}

type deleteBranchMsg struct {
	err error
}

func initialBranchModel(mode string, branchName string) branchModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	ti := textinput.New()
	ti.Placeholder = "Enter branch name..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 40

	return branchModel{
		state:      branchStateList,
		spinner:    s,
		textInput:  ti,
		mode:       mode,
		branchName: branchName,
		showHelp:   mode == "list",
	}
}

func (m branchModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, getBranches)
}

func (m branchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle text input in creating mode
		if m.state == branchStateCreating && m.mode == "new" {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "enter":
				branchName := strings.TrimSpace(m.textInput.Value())
				if branchName == "" {
					m.state = branchStateError
					m.err = fmt.Errorf("branch name cannot be empty")
					return m, tea.Quit
				}
				m.branchName = branchName
				return m, createAndSwitchBranch(branchName)
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}

		// Handle list navigation and actions
		if m.state == branchStateList {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.branches)-1 {
					m.cursor++
				}
			case "enter":
				if len(m.branches) > 0 && m.cursor < len(m.branches) {
					selectedBranch := m.branches[m.cursor]
					if !selectedBranch.Current {
						m.branchName = selectedBranch.Name
						m.state = branchStateSwitching
						return m, switchToBranch(selectedBranch.Name)
					}
				}
			case "n":
				// Create new branch
				m.mode = "new"
				m.state = branchStateCreating
				m.textInput.Focus()
				return m, textinput.Blink
			case "d":
				// Delete selected branch
				if len(m.branches) > 0 && m.cursor < len(m.branches) {
					selectedBranch := m.branches[m.cursor]
					if selectedBranch.Current {
						m.state = branchStateError
						m.err = fmt.Errorf("cannot delete current branch")
						return m, tea.Quit
					}
					m.branchName = selectedBranch.Name
					m.state = branchStateDeleting
					return m, deleteBranchCmd(selectedBranch.Name)
				}
			case "?":
				m.showHelp = !m.showHelp
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case getBranchesMsg:
		if msg.err != nil {
			m.state = branchStateError
			m.err = msg.err
			return m, tea.Quit
		}
		m.branches = msg.branches

		// Handle different modes
		switch m.mode {
		case "new":
			if m.branchName == "" {
				// Interactive mode - show input
				m.state = branchStateCreating
				return m, textinput.Blink
			} else {
				// Direct mode - create immediately
				return m, createAndSwitchBranch(m.branchName)
			}
		case "switch":
			// Switch to specified branch
			m.state = branchStateSwitching
			return m, switchToBranch(m.branchName)
		case "delete":
			// Delete specified branch
			m.state = branchStateDeleting
			return m, deleteBranchCmd(m.branchName)
		default:
			// List mode - just display
			m.state = branchStateList
		}
		return m, nil

	case createBranchMsg:
		if msg.err != nil {
			m.state = branchStateError
			m.err = msg.err
			return m, tea.Quit
		}
		m.state = branchStateDone
		return m, tea.Quit

	case switchBranchMsg:
		if msg.err != nil {
			m.state = branchStateError
			m.err = msg.err
			return m, tea.Quit
		}
		m.state = branchStateDone
		return m, tea.Quit

	case deleteBranchMsg:
		if msg.err != nil {
			m.state = branchStateError
			m.err = msg.err
			return m, tea.Quit
		}
		m.state = branchStateDone
		return m, tea.Quit
	}

	return m, nil
}

func (m branchModel) View() string {
	switch m.state {
	case branchStateList:
		var s strings.Builder

		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4"))

		s.WriteString(titleStyle.Render("Branches"))
		s.WriteString("\n\n")

		currentStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)
		normalStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))
		cursorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)
		dimStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

		for i, branch := range m.branches {
			cursor := "  "
			if i == m.cursor {
				cursor = cursorStyle.Render("→ ")
			}

			branchMark := " "
			if branch.Current {
				branchMark = "*"
			}

			branchStyle := normalStyle
			if branch.Current {
				branchStyle = currentStyle
			}

			s.WriteString(fmt.Sprintf("%s%s %s",
				cursor,
				branchMark,
				branchStyle.Render(branch.Name),
			))

			if branch.Upstream != "" {
				s.WriteString(fmt.Sprintf(" %s", dimStyle.Render(fmt.Sprintf("[%s]", branch.Upstream))))
			}

			if branch.LastCommit != "" {
				s.WriteString(fmt.Sprintf(" %s", dimStyle.Render(branch.LastCommit)))
			}

			s.WriteString("\n")
		}

		if m.showHelp {
			s.WriteString("\n")
			helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			s.WriteString(helpStyle.Render("↑/k: up  ↓/j: down  Enter: switch  n: new branch  d: delete  ?: help  q: quit"))
		} else {
			s.WriteString("\n")
			helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			s.WriteString(helpStyle.Render("Press ? for help"))
		}

		return s.String()

	case branchStateCreating:
		if m.mode == "new" && m.branchName == "" {
			return fmt.Sprintf("\n%s\n%s\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).Render("Create new branch"),
				m.textInput.View(),
			)
		}
		return fmt.Sprintf("%s Creating and switching to branch '%s'...", m.spinner.View(), m.branchName)

	case branchStateSwitching:
		return fmt.Sprintf("%s Switching to branch '%s'...", m.spinner.View(), m.branchName)

	case branchStateDeleting:
		return fmt.Sprintf("%s Deleting branch '%s'...", m.spinner.View(), m.branchName)

	case branchStateDone:
		switch m.mode {
		case "new":
			return successStyle.Render(fmt.Sprintf("✓ Created and switched to branch '%s'", m.branchName))
		case "switch":
			return successStyle.Render(fmt.Sprintf("✓ Switched to branch '%s'", m.branchName))
		case "delete":
			return successStyle.Render(fmt.Sprintf("✓ Deleted branch '%s'", m.branchName))
		default:
			return successStyle.Render("✓ Done")
		}

	case branchStateError:
		return errorStyle.Render(fmt.Sprintf("✗ Error: %s", m.err))
	}

	return ""
}

func getBranches() tea.Msg {
	branches, err := GetBranches()
	return getBranchesMsg{branches: branches, err: err}
}

func createAndSwitchBranch(branchName string) tea.Cmd {
	return func() tea.Msg {
		err := CreateAndSwitchBranch(branchName)
		return createBranchMsg{err: err}
	}
}

func switchToBranch(branchName string) tea.Cmd {
	return func() tea.Msg {
		err := SwitchBranch(branchName)
		return switchBranchMsg{err: err}
	}
}

func deleteBranchCmd(branchName string) tea.Cmd {
	return func() tea.Msg {
		err := DeleteBranch(branchName)
		return deleteBranchMsg{err: err}
	}
}

// Replay (rebase) TUI model
type replayState int

const (
	replayStateChecking replayState = iota
	replayStateShowingCommits
	replayStateConfirming
	replayStateReplaying
	replayStateConflict
	replayStateDone
	replayStateError
)

type replayModel struct {
	state         replayState
	spinner       spinner.Model
	err           error
	ontoBranch    string
	currentBranch string
	commits       []CommitInfo
	interactive   bool
	output        string
	cursor        int
}

type getReplayCommitsMsg struct {
	commits []CommitInfo
	err     error
}

type replayCommitsMsg struct {
	output string
	err    error
}

type checkRebaseMsg struct {
	inProgress bool
	err        error
}

func initialReplayModel(ontoBranch string, interactive bool) replayModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	return replayModel{
		state:       replayStateChecking,
		spinner:     s,
		ontoBranch:  ontoBranch,
		interactive: interactive,
	}
}

func (m replayModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, checkForRebase)
}

func (m replayModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state == replayStateShowingCommits {
			switch msg.String() {
			case "ctrl+c", "q", "n", "N":
				m.state = replayStateDone
				m.err = fmt.Errorf("replay cancelled")
				return m, tea.Quit
			case "y", "Y", "enter":
				m.state = replayStateReplaying
				return m, replayCommits(m.ontoBranch)
			}
		} else if m.state == replayStateConfirming {
			switch msg.String() {
			case "ctrl+c", "q", "n", "N":
				m.state = replayStateDone
				m.err = fmt.Errorf("replay cancelled")
				return m, tea.Quit
			case "y", "Y":
				m.state = replayStateReplaying
				return m, replayCommits(m.ontoBranch)
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case checkRebaseMsg:
		if msg.err != nil {
			m.state = replayStateError
			m.err = msg.err
			return m, tea.Quit
		}
		if msg.inProgress {
			m.state = replayStateError
			m.err = fmt.Errorf("rebase already in progress. Use 'git rebase --continue', '--skip', or '--abort'")
			return m, tea.Quit
		}
		// Get current branch
		currentBranch, err := GetCurrentBranch()
		if err != nil {
			m.state = replayStateError
			m.err = err
			return m, tea.Quit
		}
		m.currentBranch = currentBranch

		// Check if branch is same as onto branch
		if currentBranch == m.ontoBranch {
			m.state = replayStateError
			m.err = fmt.Errorf("already on branch '%s', nothing to replay", m.ontoBranch)
			return m, tea.Quit
		}

		return m, getReplayCommitsCmd(m.ontoBranch)

	case getReplayCommitsMsg:
		if msg.err != nil {
			m.state = replayStateError
			m.err = msg.err
			return m, tea.Quit
		}
		m.commits = msg.commits

		if len(msg.commits) == 0 {
			m.state = replayStateError
			m.err = fmt.Errorf("no commits to replay (already up to date with '%s')", m.ontoBranch)
			return m, tea.Quit
		}

		m.state = replayStateShowingCommits
		return m, nil

	case replayCommitsMsg:
		if msg.err != nil {
			// Check if it's a conflict
			if strings.Contains(msg.output, "CONFLICT") || strings.Contains(msg.output, "conflict") {
				m.state = replayStateConflict
				m.output = msg.output
				return m, tea.Quit
			}
			m.state = replayStateError
			m.err = msg.err
			m.output = msg.output
			return m, tea.Quit
		}
		m.output = msg.output
		m.state = replayStateDone
		return m, tea.Quit
	}

	return m, nil
}

func (m replayModel) View() string {
	switch m.state {
	case replayStateChecking:
		return fmt.Sprintf("%s Checking repository status...", m.spinner.View())

	case replayStateShowingCommits:
		var s strings.Builder

		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4"))

		s.WriteString(titleStyle.Render(fmt.Sprintf("Replay commits from '%s' onto '%s'", m.currentBranch, m.ontoBranch)))
		s.WriteString("\n\n")

		infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		s.WriteString(infoStyle.Render(fmt.Sprintf("The following %d commit(s) will be replayed:", len(m.commits))))
		s.WriteString("\n\n")

		commitStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
		hashStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

		// Show commits in reverse order (oldest first, as they'll be applied)
		for i := len(m.commits) - 1; i >= 0; i-- {
			commit := m.commits[i]
			s.WriteString(fmt.Sprintf("%s %s %s\n",
				commitStyle.Render("●"),
				commit.Message,
				timeStyle.Render(fmt.Sprintf("(%s)", commit.RelativeTime)),
			))
			s.WriteString(fmt.Sprintf("  %s\n", hashStyle.Render(commit.ShortHash)))
			if i > 0 {
				s.WriteString(commitStyle.Render("│") + "\n")
			}
		}

		s.WriteString("\n")
		s.WriteString(highlightStyle.Render("Proceed with replay? (y/n): "))

		return s.String()

	case replayStateConfirming:
		return fmt.Sprintf("\n%s\n%s",
			fmt.Sprintf("Replay %d commits from '%s' onto '%s'?", len(m.commits), m.currentBranch, m.ontoBranch),
			highlightStyle.Render("(y)es or (n)o: "),
		)

	case replayStateReplaying:
		return fmt.Sprintf("%s Replaying commits onto '%s'...", m.spinner.View(), m.ontoBranch)

	case replayStateConflict:
		var s strings.Builder
		s.WriteString(errorStyle.Render("✗ Conflicts detected during replay") + "\n\n")

		infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		s.WriteString(infoStyle.Render("Please resolve conflicts and then:") + "\n")
		s.WriteString("  • Fix conflicts in your files\n")
		s.WriteString("  • Stage the resolved files: " + highlightStyle.Render("git add <files>") + "\n")
		s.WriteString("  • Continue: " + highlightStyle.Render("git rebase --continue") + "\n")
		s.WriteString("  • Or abort: " + highlightStyle.Render("git rebase --abort") + "\n\n")

		if m.output != "" {
			s.WriteString(infoStyle.Render("Git output:") + "\n")
			s.WriteString(m.output + "\n")
		}

		return s.String()

	case replayStateDone:
		if m.err != nil {
			return errorStyle.Render(fmt.Sprintf("✗ %s", m.err))
		}
		return successStyle.Render(fmt.Sprintf("✓ Successfully replayed %d commit(s) onto '%s'", len(m.commits), m.ontoBranch))

	case replayStateError:
		errMsg := errorStyle.Render(fmt.Sprintf("✗ Error: %s", m.err))
		if m.output != "" {
			infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			errMsg += "\n\n" + infoStyle.Render("Git output:") + "\n" + m.output
		}
		return errMsg
	}

	return ""
}

func checkForRebase() tea.Msg {
	inProgress, err := CheckRebaseInProgress()
	return checkRebaseMsg{inProgress: inProgress, err: err}
}

func getReplayCommitsCmd(ontoBranch string) tea.Cmd {
	return func() tea.Msg {
		commits, err := GetRebaseCommits(ontoBranch)
		return getReplayCommitsMsg{commits: commits, err: err}
	}
}

func replayCommits(ontoBranch string) tea.Cmd {
	return func() tea.Msg {
		output, err := ReplayCommits(ontoBranch)
		return replayCommitsMsg{output: output, err: err}
	}
}

// Stack (commit history) TUI model
type stackState int

const (
	stackStateLoading stackState = iota
	stackStateList
	stackStateFiltering
	stackStateCheckingOut
	stackStateShowingDetails
	stackStateDone
	stackStateError
)

type stackModel struct {
	state           stackState
	spinner         spinner.Model
	textInput       textinput.Model
	commits         []CommitInfo
	filteredCommits []CommitInfo
	cursor          int
	err             error
	allBranches     bool
	mineOnly        bool
	filePath        string
	author          string
	limit           int
	filterMode      bool
	filterQuery     string
	showHelp        bool
	selectedCommit  *CommitInfo
}

type getCommitsMsg struct {
	commits []CommitInfo
	err     error
}

type checkoutCommitMsg struct {
	err error
}

func initialStackModel(limit int, allBranches bool, mineOnly bool, filePath string) stackModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	ti := textinput.New()
	ti.Placeholder = "Type to filter commits..."
	ti.CharLimit = 100
	ti.Width = 50
	ti.Prompt = ""
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	// Get author if mineOnly is true
	author := ""
	if mineOnly {
		cmd := exec.Command("git", "config", "user.name")
		output, err := cmd.Output()
		if err == nil {
			author = strings.TrimSpace(string(output))
		}
	}

	return stackModel{
		state:           stackStateLoading,
		spinner:         s,
		textInput:       ti,
		allBranches:     allBranches,
		mineOnly:        mineOnly,
		filePath:        filePath,
		author:          author,
		limit:           limit,
		showHelp:        true,
		commits:         []CommitInfo{},
		filteredCommits: []CommitInfo{},
		filterQuery:     "",
		filterMode:      false,
	}
}

func (m stackModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, getCommits(m.limit, m.allBranches, m.author, m.filePath))
}

func (m stackModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle list navigation and filter mode toggle
		if m.state == stackStateList {
			// Check for filter mode entry FIRST, before handling filter input
			if !m.filterMode && msg.String() == "/" {
				// Enter filter mode
				m.filterMode = true
				m.filterQuery = ""
				m.textInput.SetValue("")
				m.textInput.Focus()
				// Return with blink to start cursor animation
				return m, textinput.Blink
			}

			// Handle filter mode input
			if m.filterMode {
				switch msg.String() {
				case "esc", "ctrl+c", "q":
					m.filterMode = false
					m.filterQuery = ""
					m.textInput.SetValue("")
					m.textInput.Blur()
					m.filteredCommits = m.commits
					m.cursor = 0
					return m, nil
				case "enter":
					m.filterMode = false
					m.textInput.Blur()
					return m, nil
				default:
					var cmd tea.Cmd
					m.textInput, cmd = m.textInput.Update(msg)
					m.filterQuery = m.textInput.Value()
					m.applyFilter()
					m.cursor = 0
					return m, cmd
				}
			}

			// Handle normal list navigation
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.getDisplayCommits())-1 {
					m.cursor++
				}
			case "g":
				// Go to top
				m.cursor = 0
			case "G":
				// Go to bottom
				m.cursor = len(m.getDisplayCommits()) - 1
			case "c":
				// Clear filter
				m.filterQuery = ""
				m.textInput.SetValue("")
				m.filteredCommits = m.commits
				m.cursor = 0
			case "enter":
				// Checkout selected commit
				commits := m.getDisplayCommits()
				if len(commits) > 0 && m.cursor < len(commits) {
					m.selectedCommit = &commits[m.cursor]
					m.state = stackStateCheckingOut
					return m, checkoutCommitCmd(m.selectedCommit.Hash)
				}
			// Disabled for now - coming soon
			// case "d":
			// 	// Show commit details
			// 	commits := m.getDisplayCommits()
			// 	if len(commits) > 0 && m.cursor < len(commits) {
			// 		m.selectedCommit = &commits[m.cursor]
			// 		m.state = stackStateShowingDetails
			// 		return m, getCommitDetailsCmd(m.selectedCommit.Hash)
			// 	}
			case "?":
				m.showHelp = !m.showHelp
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case getCommitsMsg:
		if msg.err != nil {
			m.state = stackStateError
			m.err = msg.err
			return m, tea.Quit
		}
		m.commits = msg.commits
		m.filteredCommits = msg.commits
		if len(msg.commits) == 0 {
			m.state = stackStateError
			m.err = fmt.Errorf("no commits found")
			return m, tea.Quit
		}
		m.state = stackStateList
		return m, nil

	case checkoutCommitMsg:
		if msg.err != nil {
			m.state = stackStateError
			m.err = msg.err
			return m, tea.Quit
		}
		m.state = stackStateDone
		return m, tea.Quit
	}

	return m, nil
}

func (m *stackModel) applyFilter() {
	if m.filterQuery == "" {
		m.filteredCommits = m.commits
		return
	}

	query := strings.ToLower(m.filterQuery)
	filtered := make([]CommitInfo, 0)

	for _, commit := range m.commits {
		// Search in message, hash, and author
		if strings.Contains(strings.ToLower(commit.Message), query) ||
			strings.Contains(strings.ToLower(commit.Hash), query) ||
			strings.Contains(strings.ToLower(commit.ShortHash), query) ||
			strings.Contains(strings.ToLower(commit.Author), query) {
			filtered = append(filtered, commit)
		}
	}

	m.filteredCommits = filtered
}

func (m stackModel) getDisplayCommits() []CommitInfo {
	if m.filterQuery != "" {
		if m.filteredCommits == nil {
			return []CommitInfo{}
		}
		return m.filteredCommits
	}
	if m.commits == nil {
		return []CommitInfo{}
	}
	return m.commits
}

func (m stackModel) View() string {
	switch m.state {
	case stackStateLoading:
		return fmt.Sprintf("%s Loading commits...", m.spinner.View())

	case stackStateList:
		var s strings.Builder

		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4"))

		title := "Commit History"
		if m.allBranches {
			title += " (all branches)"
		}
		if m.mineOnly {
			title += " (mine)"
		}
		if m.filePath != "" {
			title += fmt.Sprintf(" - %s", m.filePath)
		}

		s.WriteString(titleStyle.Render(title))
		s.WriteString("\n\n")

		// Show filter bar
		if m.filterMode {
			filterLabelStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7D56F4")).
				Bold(true)
			s.WriteString(filterLabelStyle.Render("🔍 Filter: "))
			s.WriteString(m.textInput.View())
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(" (Esc/Ctrl+C/q to cancel)"))
			s.WriteString("\n\n")
		} else if m.filterQuery != "" {
			filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			s.WriteString(filterStyle.Render(fmt.Sprintf("🔍 Active filter: %s (press 'c' to clear)", m.filterQuery)))
			s.WriteString("\n\n")
		}

		commits := m.getDisplayCommits()

		if len(commits) == 0 {
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("No commits match filter"))
			s.WriteString("\n")
		} else {
			commitStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
			timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			hashStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			authorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
			pipeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

			for i, commit := range commits {
				cursor := "  "
				if i == m.cursor {
					cursor = cursorStyle.Render("→ ")
				}

				// Show bullet, time and message
				s.WriteString(fmt.Sprintf("%s%s %s %s\n",
					cursor,
					commitStyle.Render("●"),
					timeStyle.Render(commit.RelativeTime),
					commit.Message,
				))

				// Show hash and author
				s.WriteString(fmt.Sprintf("   %s",
					hashStyle.Render(commit.ShortHash),
				))

				if !m.mineOnly {
					s.WriteString(fmt.Sprintf(" %s",
						authorStyle.Render(fmt.Sprintf("by %s", commit.Author)),
					))
				}
				s.WriteString("\n")

				// Show pipe between commits (except for last one)
				if i < len(commits)-1 {
					s.WriteString("  " + pipeStyle.Render("│") + "\n")
				}
			}
		}

		s.WriteString("\n")

		// Show help
		if m.showHelp {
			helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			if m.filterMode {
				s.WriteString(helpStyle.Render("Type to filter • Enter: apply • Esc/Ctrl+C/q: cancel"))
			} else {
				s.WriteString(helpStyle.Render("↑/k: up  ↓/j: down  g: top  G: bottom  /: filter  c: clear filter"))
				s.WriteString("\n")
				s.WriteString(helpStyle.Render("Enter: checkout  ?: toggle help  q: quit"))
			}
		} else {
			helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			s.WriteString(helpStyle.Render("Press ? for help"))
		}

		return s.String()

	case stackStateCheckingOut:
		if m.selectedCommit != nil {
			return fmt.Sprintf("%s Checking out commit %s...", m.spinner.View(), m.selectedCommit.ShortHash)
		}
		return fmt.Sprintf("%s Checking out commit...", m.spinner.View())

	case stackStateDone:
		if m.selectedCommit != nil {
			warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAA00")).Bold(true)
			infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

			var s strings.Builder
			s.WriteString(successStyle.Render(fmt.Sprintf("✓ Checked out commit %s", m.selectedCommit.ShortHash)) + "\n")
			s.WriteString(warningStyle.Render("⚠ You are now in 'detached HEAD' state") + "\n\n")
			s.WriteString(infoStyle.Render("You can look around, make experimental changes and commit them.\n"))
			s.WriteString(infoStyle.Render("To return to a branch, run: ") + highlightStyle.Render("snap branch switch <branch-name>"))

			return s.String()
		}
		return successStyle.Render("✓ Done")

	case stackStateError:
		return errorStyle.Render(fmt.Sprintf("✗ Error: %s", m.err))
	}

	return ""
}

func getCommits(limit int, allBranches bool, author string, filePath string) tea.Cmd {
	return func() tea.Msg {
		commits, err := GetCommitHistory(limit, allBranches, author, filePath)
		return getCommitsMsg{commits: commits, err: err}
	}
}

func checkoutCommitCmd(commitHash string) tea.Cmd {
	return func() tea.Msg {
		err := CheckoutCommit(commitHash)
		return checkoutCommitMsg{err: err}
	}
}

func getCommitDetailsCmd(commitHash string) tea.Cmd {
	return func() tea.Msg {
		// For now, just return nil - we'll implement details view later
		return nil
	}
}

// Tags TUI model
type tagsState int

const (
	tagsStateLoading tagsState = iota
	tagsStateList
	tagsStateDone
	tagsStateError
)

type tagsModel struct {
	state        tagsState
	spinner      spinner.Model
	textInput    textinput.Model
	tags         []TagInfo
	filteredTags []TagInfo
	cursor       int
	err          error
	filterMode   bool
	filterQuery  string
	showHelp     bool
	width        int
	height       int
}

type getTagsMsg struct {
	tags []TagInfo
	err  error
}

func initialTagsModel() tagsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	ti := textinput.New()
	ti.Placeholder = "Type to filter tags..."
	ti.CharLimit = 100
	ti.Width = 50
	ti.Prompt = ""
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	return tagsModel{
		state:        tagsStateLoading,
		spinner:      s,
		textInput:    ti,
		showHelp:     true,
		tags:         []TagInfo{},
		filteredTags: []TagInfo{},
		filterQuery:  "",
		filterMode:   false,
		width:        80, // default, will be updated by WindowSizeMsg
		height:       24,
	}
}

func (m tagsModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, getTagsCmd)
}

func (m tagsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = msg.Width - 20
		return m, nil

	case tea.KeyMsg:
		if m.state == tagsStateList {
			// Check for filter mode entry FIRST
			if !m.filterMode && msg.String() == "/" {
				m.filterMode = true
				m.filterQuery = ""
				m.textInput.SetValue("")
				m.textInput.Focus()
				return m, textinput.Blink
			}

			// Handle filter mode input
			if m.filterMode {
				switch msg.String() {
				case "esc", "ctrl+c", "q":
					m.filterMode = false
					m.filterQuery = ""
					m.textInput.SetValue("")
					m.textInput.Blur()
					m.filteredTags = m.tags
					m.cursor = 0
					return m, nil
				case "enter":
					m.filterMode = false
					m.textInput.Blur()
					return m, nil
				default:
					var cmd tea.Cmd
					m.textInput, cmd = m.textInput.Update(msg)
					m.filterQuery = m.textInput.Value()
					m.applyFilter()
					m.cursor = 0
					return m, cmd
				}
			}

			// Handle normal list navigation
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.getDisplayTags())-1 {
					m.cursor++
				}
			case "g":
				m.cursor = 0
			case "G":
				m.cursor = len(m.getDisplayTags()) - 1
			case "c":
				m.filterQuery = ""
				m.textInput.SetValue("")
				m.filteredTags = m.tags
				m.cursor = 0
			case "?":
				m.showHelp = !m.showHelp
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case getTagsMsg:
		if msg.err != nil {
			m.state = tagsStateError
			m.err = msg.err
			return m, tea.Quit
		}
		m.tags = msg.tags
		m.filteredTags = msg.tags
		if len(msg.tags) == 0 {
			m.state = tagsStateError
			m.err = fmt.Errorf("no tags found")
			return m, tea.Quit
		}
		m.state = tagsStateList
		return m, nil
	}

	return m, nil
}

func (m *tagsModel) applyFilter() {
	if m.filterQuery == "" {
		m.filteredTags = m.tags
		return
	}

	query := strings.ToLower(m.filterQuery)
	filtered := make([]TagInfo, 0)

	for _, tag := range m.tags {
		if strings.Contains(strings.ToLower(tag.Name), query) ||
			strings.Contains(strings.ToLower(tag.Message), query) ||
			strings.Contains(strings.ToLower(tag.ShortHash), query) {
			filtered = append(filtered, tag)
		}
	}

	m.filteredTags = filtered
}

func (m tagsModel) getDisplayTags() []TagInfo {
	if m.filterQuery != "" {
		if m.filteredTags == nil {
			return []TagInfo{}
		}
		return m.filteredTags
	}
	if m.tags == nil {
		return []TagInfo{}
	}
	return m.tags
}

func (m tagsModel) View() string {
	switch m.state {
	case tagsStateLoading:
		return fmt.Sprintf("%s Loading tags...", m.spinner.View())

	case tagsStateList:
		var s strings.Builder

		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4"))

		s.WriteString(titleStyle.Render("Tags"))
		s.WriteString("\n\n")

		// Show filter bar
		if m.filterMode {
			filterLabelStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7D56F4")).
				Bold(true)
			s.WriteString(filterLabelStyle.Render("Filter: "))
			s.WriteString(m.textInput.View())
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(" (Esc to cancel)"))
			s.WriteString("\n\n")
		} else if m.filterQuery != "" {
			filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			s.WriteString(filterStyle.Render(fmt.Sprintf("Filter: %s (press 'c' to clear)", m.filterQuery)))
			s.WriteString("\n\n")
		}

		tags := m.getDisplayTags()

		if len(tags) == 0 {
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("No tags match filter"))
			s.WriteString("\n")
		} else {
			tagStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
			hashStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
			cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)

			// Calculate max tag name width for alignment
			maxTagWidth := 0
			for _, tag := range tags {
				if len(tag.Name) > maxTagWidth {
					maxTagWidth = len(tag.Name)
				}
			}

			for i, tag := range tags {
				cursor := "  "
				if i == m.cursor {
					cursor = cursorStyle.Render("→ ")
				}

				// Pad tag name for alignment
				paddedName := fmt.Sprintf("%-*s", maxTagWidth, tag.Name)

				// Calculate available width for message
				// cursor(2) + name + spacing(2) + hash(7) + spacing(2) + time(~15)
				metaWidth := 2 + maxTagWidth + 2 + 7 + 2 + len(tag.RelativeTime) + 2
				msgWidth := m.width - metaWidth
				if msgWidth < 20 {
					msgWidth = 20
				}

				// Truncate message if needed
				msg := tag.Message
				if len(msg) > msgWidth {
					msg = msg[:msgWidth-3] + "..."
				}

				// Build the line with proper spacing
				line := fmt.Sprintf("%s%s  %s  %s",
					cursor,
					tagStyle.Render(paddedName),
					hashStyle.Render(tag.ShortHash),
					timeStyle.Render(tag.RelativeTime),
				)

				if msg != "" {
					line += "  " + msgStyle.Render(msg)
				}

				s.WriteString(line)
				s.WriteString("\n")

				if i < len(tags)-1 {
					s.WriteString("\n")
				}
			}
		}

		s.WriteString("\n")

		// Show help
		if m.showHelp {
			helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			if m.filterMode {
				s.WriteString(helpStyle.Render("Type to filter • Enter: apply • Esc: cancel"))
			} else {
				s.WriteString(helpStyle.Render("↑/k: up  ↓/j: down  g: top  G: bottom  /: filter  c: clear  ?: help  q: quit"))
			}
		} else {
			helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			s.WriteString(helpStyle.Render("Press ? for help"))
		}

		return s.String()

	case tagsStateError:
		return errorStyle.Render(fmt.Sprintf("✗ Error: %s", m.err))
	}

	return ""
}

func getTagsCmd() tea.Msg {
	tags, err := GetTags()
	return getTagsMsg{tags: tags, err: err}
}

// Tags Diff TUI model
type tagsDiffState int

const (
	tagsDiffStateLoading tagsDiffState = iota
	tagsDiffStateList
	tagsDiffStateError
)

type tagsDiffModel struct {
	state       tagsDiffState
	spinner     spinner.Model
	commits     []CommitWithStats
	previousTag string
	err         error
	width       int
	height      int
	cursor      int
	showHelp    bool
}

type getTagsDiffMsg struct {
	commits     []CommitWithStats
	previousTag string
	err         error
}

func initialTagsDiffModel() tagsDiffModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	return tagsDiffModel{
		state:    tagsDiffStateLoading,
		spinner:  s,
		showHelp: true,
		width:    80,
		height:   24,
	}
}

func (m tagsDiffModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, getTagsDiffCmd)
}

func (m tagsDiffModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.state == tagsDiffStateList {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.commits)-1 {
					m.cursor++
				}
			case "g":
				m.cursor = 0
			case "G":
				m.cursor = len(m.commits) - 1
			case "?":
				m.showHelp = !m.showHelp
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case getTagsDiffMsg:
		if msg.err != nil {
			m.state = tagsDiffStateError
			m.err = msg.err
			return m, tea.Quit
		}
		m.commits = msg.commits
		m.previousTag = msg.previousTag
		if len(msg.commits) == 0 {
			m.state = tagsDiffStateError
			m.err = fmt.Errorf("no commits since %s", msg.previousTag)
			return m, tea.Quit
		}
		m.state = tagsDiffStateList
		return m, nil
	}

	return m, nil
}

func (m tagsDiffModel) View() string {
	switch m.state {
	case tagsDiffStateLoading:
		return fmt.Sprintf("%s Loading diff...", m.spinner.View())

	case tagsDiffStateList:
		var s strings.Builder

		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4"))

		s.WriteString(titleStyle.Render("Changes since " + m.previousTag))
		s.WriteString("\n\n")

		// Summary
		totalAdditions := 0
		totalDeletions := 0
		for _, c := range m.commits {
			totalAdditions += c.Additions
			totalDeletions += c.Deletions
		}

		summaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
		delStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))

		s.WriteString(summaryStyle.Render(fmt.Sprintf("%d commits  ", len(m.commits))))
		s.WriteString(addStyle.Render(fmt.Sprintf("+%d", totalAdditions)))
		s.WriteString(summaryStyle.Render("  "))
		s.WriteString(delStyle.Render(fmt.Sprintf("-%d", totalDeletions)))
		s.WriteString("\n\n")

		// Commits
		hashStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAA00"))
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
		timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)

		for i, commit := range m.commits {
			cursor := "  "
			if i == m.cursor {
				cursor = cursorStyle.Render("→ ")
			}

			// Format: cursor hash +add -del message (time)
			statsStr := ""
			if commit.Additions > 0 || commit.Deletions > 0 {
				statsStr = fmt.Sprintf(" %s %s",
					addStyle.Render(fmt.Sprintf("+%d", commit.Additions)),
					delStyle.Render(fmt.Sprintf("-%d", commit.Deletions)),
				)
			}

			// Truncate message if needed
			msg := commit.Message
			maxMsgLen := m.width - 40
			if maxMsgLen < 20 {
				maxMsgLen = 20
			}
			if len(msg) > maxMsgLen {
				msg = msg[:maxMsgLen-3] + "..."
			}

			s.WriteString(fmt.Sprintf("%s%s%s  %s  %s\n",
				cursor,
				hashStyle.Render(commit.ShortHash),
				statsStr,
				msgStyle.Render(msg),
				timeStyle.Render(commit.RelativeTime),
			))
		}

		s.WriteString("\n")

		if m.showHelp {
			helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			s.WriteString(helpStyle.Render("↑/k: up  ↓/j: down  g: top  G: bottom  ?: help  q: quit"))
		} else {
			helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			s.WriteString(helpStyle.Render("Press ? for help"))
		}

		return s.String()

	case tagsDiffStateError:
		return errorStyle.Render(fmt.Sprintf("✗ Error: %s", m.err))
	}

	return ""
}

func getTagsDiffCmd() tea.Msg {
	// Get the most recent tag
	prevTag, err := GetMostRecentTag()
	if err != nil {
		prevTag = "(no previous tag)"
	}

	// Get commits since that tag
	commits, err := GetCommitsSinceTag(prevTag)
	if err != nil {
		return getTagsDiffMsg{err: err}
	}

	return getTagsDiffMsg{commits: commits, previousTag: prevTag}
}

// Tags Create TUI model
type tagsCreateState int

const (
	tagsCreateStateLoading tagsCreateState = iota
	tagsCreateStatePreview
	tagsCreateStateConfirm
	tagsCreateStateCreating
	tagsCreateStatePushing
	tagsCreateStateDone
	tagsCreateStateError
)

type tagsCreateModel struct {
	state       tagsCreateState
	spinner     spinner.Model
	commits     []CommitWithStats
	previousTag string
	newTag      string
	err         error
	width       int
	height      int
	cursor      int
	showHelp    bool
}

type createTagMsg struct {
	err error
}

type pushTagMsg struct {
	output string
	err    error
}

func initialTagsCreateModel(tagName string) tagsCreateModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	return tagsCreateModel{
		state:    tagsCreateStateLoading,
		spinner:  s,
		newTag:   tagName,
		showHelp: true,
		width:    80,
		height:   24,
	}
}

func (m tagsCreateModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, getTagsDiffCmd)
}

func (m tagsCreateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.state == tagsCreateStatePreview {
			switch msg.String() {
			case "ctrl+c", "q", "n", "N":
				return m, tea.Quit
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.commits)-1 {
					m.cursor++
				}
			case "y", "Y", "enter":
				m.state = tagsCreateStateCreating
				return m, createTagCmd(m.newTag, m.generateTagMessage())
			case "?":
				m.showHelp = !m.showHelp
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case getTagsDiffMsg:
		if msg.err != nil {
			m.state = tagsCreateStateError
			m.err = msg.err
			return m, tea.Quit
		}
		m.commits = msg.commits
		m.previousTag = msg.previousTag
		m.state = tagsCreateStatePreview
		return m, nil

	case createTagMsg:
		if msg.err != nil {
			m.state = tagsCreateStateError
			m.err = msg.err
			return m, tea.Quit
		}
		m.state = tagsCreateStatePushing
		return m, pushTagCmd(m.newTag)

	case pushTagMsg:
		if msg.err != nil {
			// Tag was created but push failed - delete local tag and report error
			DeleteTag(m.newTag)
			m.state = tagsCreateStateError
			m.err = fmt.Errorf("failed to push tag: %w", msg.err)
			return m, tea.Quit
		}
		m.state = tagsCreateStateDone
		return m, tea.Quit
	}

	return m, nil
}

func (m tagsCreateModel) generateTagMessage() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Release %s\n\n", m.newTag))

	if m.previousTag != "" && m.previousTag != "(no previous tag)" {
		sb.WriteString(fmt.Sprintf("Changes since %s:\n\n", m.previousTag))
	} else {
		sb.WriteString("Changes:\n\n")
	}

	for _, commit := range m.commits {
		sb.WriteString(fmt.Sprintf("- %s\n", commit.Message))
	}

	return sb.String()
}

func (m tagsCreateModel) View() string {
	switch m.state {
	case tagsCreateStateLoading:
		return fmt.Sprintf("%s Loading...", m.spinner.View())

	case tagsCreateStatePreview:
		var s strings.Builder

		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4"))

		s.WriteString(titleStyle.Render(fmt.Sprintf("Create tag %s", m.newTag)))
		s.WriteString("\n\n")

		// Previous tag info
		prevStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		if m.previousTag != "" && m.previousTag != "(no previous tag)" {
			s.WriteString(prevStyle.Render(fmt.Sprintf("Previous tag: %s", m.previousTag)))
		} else {
			s.WriteString(prevStyle.Render("No previous tag"))
		}
		s.WriteString("\n\n")

		// Summary
		totalAdditions := 0
		totalDeletions := 0
		for _, c := range m.commits {
			totalAdditions += c.Additions
			totalDeletions += c.Deletions
		}

		summaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
		delStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))

		if len(m.commits) > 0 {
			s.WriteString(summaryStyle.Render(fmt.Sprintf("%d commits  ", len(m.commits))))
			s.WriteString(addStyle.Render(fmt.Sprintf("+%d", totalAdditions)))
			s.WriteString(summaryStyle.Render("  "))
			s.WriteString(delStyle.Render(fmt.Sprintf("-%d", totalDeletions)))
			s.WriteString("\n\n")

			// Commits
			hashStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAA00"))
			msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
			timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)

			for i, commit := range m.commits {
				cursor := "  "
				if i == m.cursor {
					cursor = cursorStyle.Render("→ ")
				}

				statsStr := ""
				if commit.Additions > 0 || commit.Deletions > 0 {
					statsStr = fmt.Sprintf(" %s %s",
						addStyle.Render(fmt.Sprintf("+%d", commit.Additions)),
						delStyle.Render(fmt.Sprintf("-%d", commit.Deletions)),
					)
				}

				msg := commit.Message
				maxMsgLen := m.width - 40
				if maxMsgLen < 20 {
					maxMsgLen = 20
				}
				if len(msg) > maxMsgLen {
					msg = msg[:maxMsgLen-3] + "..."
				}

				s.WriteString(fmt.Sprintf("%s%s%s  %s  %s\n",
					cursor,
					hashStyle.Render(commit.ShortHash),
					statsStr,
					msgStyle.Render(msg),
					timeStyle.Render(commit.RelativeTime),
				))
			}
		} else {
			s.WriteString(summaryStyle.Render("No new commits (tag will be created at current HEAD)"))
			s.WriteString("\n")
		}

		s.WriteString("\n")
		s.WriteString(highlightStyle.Render("Create and push tag? (y/n): "))

		return s.String()

	case tagsCreateStateCreating:
		return fmt.Sprintf("%s Creating tag %s...", m.spinner.View(), m.newTag)

	case tagsCreateStatePushing:
		return fmt.Sprintf("%s Pushing tag %s...", m.spinner.View(), m.newTag)

	case tagsCreateStateDone:
		var s strings.Builder
		s.WriteString(successStyle.Render(fmt.Sprintf("✓ Created and pushed tag %s", m.newTag)))
		if m.previousTag != "" && m.previousTag != "(no previous tag)" {
			s.WriteString("\n")
			infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			s.WriteString(infoStyle.Render(fmt.Sprintf("  %d commits since %s", len(m.commits), m.previousTag)))
		}
		return s.String()

	case tagsCreateStateError:
		return errorStyle.Render(fmt.Sprintf("✗ Error: %s", m.err))
	}

	return ""
}

func createTagCmd(tagName, message string) tea.Cmd {
	return func() tea.Msg {
		err := CreateAnnotatedTag(tagName, message)
		return createTagMsg{err: err}
	}
}

func pushTagCmd(tagName string) tea.Cmd {
	return func() tea.Msg {
		output, err := PushTag(tagName)
		return pushTagMsg{output: output, err: err}
	}
}
