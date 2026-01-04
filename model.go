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
