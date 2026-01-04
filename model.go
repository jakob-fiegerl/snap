package main

import (
	"fmt"
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
		m.commitMessage = msg.message
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
		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))
		return fmt.Sprintf("\n%s\n%s %s",
			msgStyle.Render(m.commitMessage),
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
