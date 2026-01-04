package main

import (
	"fmt"
	"strings"

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
	stateCommitting
	stateDone
	stateError
)

type model struct {
	state         state
	err           error
	diff          string
	commitMessage string
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
	return model{
		state: stateChecking,
		seed:  seed,
	}
}

func initialModelWithMessage(seed int, customMessage string) model {
	if customMessage != "" {
		// Skip AI generation, go straight to staging
		return model{
			state:         stateStaging,
			seed:          seed,
			commitMessage: customMessage,
			useCustomMsg:  true,
		}
	}
	return initialModel(seed)
}

func (m model) Init() tea.Cmd {
	if m.useCustomMsg {
		return stageChanges
	}
	return checkOllama
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "y", "Y":
			if m.state == stateConfirming {
				m.state = stateCommitting
				return m, commitChanges(m.commitMessage)
			}

		case "n", "N":
			if m.state == stateConfirming {
				m.state = stateDone
				m.err = fmt.Errorf("commit cancelled by user")
				return m, tea.Quit
			}
		}

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
	var s strings.Builder

	s.WriteString(titleStyle.Render("📸 Snap - AI-Powered Git Snapshot Tool"))
	s.WriteString("\n\n")

	switch m.state {
	case stateChecking:
		s.WriteString("⏳ Checking Ollama connection...")
		s.WriteString("\n")

	case stateStaging:
		if !m.useCustomMsg {
			s.WriteString(successStyle.Render("✓ Ollama is running"))
			s.WriteString("\n")
		}
		s.WriteString("⏳ Staging changes...")
		s.WriteString("\n")

	case stateGettingDiff:
		if !m.useCustomMsg {
			s.WriteString(successStyle.Render("✓ Ollama is running"))
			s.WriteString("\n")
		}
		s.WriteString(successStyle.Render("✓ Staged all changes"))
		s.WriteString("\n")
		s.WriteString("⏳ Getting git diff...")
		s.WriteString("\n")

	case stateGenerating:
		s.WriteString(successStyle.Render("✓ Ollama is running"))
		s.WriteString("\n")
		s.WriteString(successStyle.Render("✓ Staged all changes"))
		s.WriteString("\n")
		s.WriteString(successStyle.Render("✓ Got git diff"))
		s.WriteString("\n")
		s.WriteString("⏳ Generating commit message with Phi-4...")
		s.WriteString("\n")

	case stateConfirming:
		if m.useCustomMsg {
			s.WriteString(successStyle.Render("✓ Staged all changes"))
			s.WriteString("\n")
			s.WriteString(successStyle.Render("✓ Got git diff"))
			s.WriteString("\n\n")
		} else {
			s.WriteString(successStyle.Render("✓ Ollama is running"))
			s.WriteString("\n")
			s.WriteString(successStyle.Render("✓ Staged all changes"))
			s.WriteString("\n")
			s.WriteString(successStyle.Render("✓ Got git diff"))
			s.WriteString("\n")
			s.WriteString(successStyle.Render("✓ Generated commit message"))
			s.WriteString("\n\n")
		}

		s.WriteString(boxStyle.Width(60).Render(m.commitMessage))
		s.WriteString("\n\n")
		s.WriteString(highlightStyle.Render("Commit with this message? (y/n): "))

	case stateCommitting:
		s.WriteString(successStyle.Render("✓ Committing changes..."))
		s.WriteString("\n")

	case stateDone:
		if m.err != nil {
			s.WriteString(errorStyle.Render(fmt.Sprintf("✗ %s", m.err)))
			s.WriteString("\n")
		} else {
			s.WriteString(successStyle.Render("✓ Changes committed successfully!"))
			s.WriteString("\n")
		}

	case stateError:
		s.WriteString(errorStyle.Render(fmt.Sprintf("✗ Error: %s", m.err)))
		s.WriteString("\n")
	}

	if m.state != stateDone && m.state != stateError {
		s.WriteString("\n")
		s.WriteString(infoStyle.Render("Press q or ctrl+c to quit"))
	}

	return s.String()
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
