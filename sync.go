package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type syncState int

const (
	syncStateChecking syncState = iota
	syncStatePulling
	syncStatePushing
	syncStateDone
	syncStateError
)

type syncModel struct {
	state      syncState
	spinner    spinner.Model
	err        error
	pullOnly   bool
	branch     string
	pullOutput string
	pushOutput string
}

type syncCheckMsg struct {
	hasRemote   bool
	hasChanges  bool
	branch      string
	hasUpstream bool
	err         error
}

type syncPullMsg struct {
	output string
	err    error
}

type syncPushMsg struct {
	output string
	err    error
}

func initialSyncModel(pullOnly bool) syncModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	return syncModel{
		state:    syncStateChecking,
		spinner:  s,
		pullOnly: pullOnly,
	}
}

func (m syncModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, checkSync)
}

func (m syncModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case syncCheckMsg:
		if msg.err != nil {
			m.state = syncStateError
			m.err = msg.err
			return m, tea.Quit
		}
		if !msg.hasRemote {
			m.state = syncStateError
			m.err = fmt.Errorf("no remote repository configured")
			return m, tea.Quit
		}
		if msg.hasChanges {
			m.state = syncStateError
			m.err = fmt.Errorf("you have uncommitted changes - run 'snap save' first")
			return m, tea.Quit
		}

		m.branch = msg.branch
		m.state = syncStatePulling
		return m, pullChanges

	case syncPullMsg:
		m.pullOutput = msg.output
		if msg.err != nil {
			if strings.Contains(msg.output, "CONFLICT") {
				m.state = syncStateError
				m.err = fmt.Errorf("merge conflict detected - resolve manually and run 'snap save'")
				return m, tea.Quit
			}
			m.state = syncStateError
			m.err = msg.err
			return m, tea.Quit
		}

		if m.pullOnly {
			m.state = syncStateDone
			return m, tea.Quit
		}

		m.state = syncStatePushing
		return m, pushChanges(m.branch)

	case syncPushMsg:
		m.pushOutput = msg.output
		if msg.err != nil {
			m.state = syncStateError
			m.err = msg.err
			return m, tea.Quit
		}

		m.state = syncStateDone
		return m, tea.Quit
	}

	return m, nil
}

func (m syncModel) View() string {
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))

	switch m.state {
	case syncStateChecking:
		return fmt.Sprintf("%s Checking repository...", m.spinner.View())

	case syncStatePulling:
		return fmt.Sprintf("%s Pulling changes...", m.spinner.View())

	case syncStatePushing:
		return fmt.Sprintf("%s Pushing changes...", m.spinner.View())

	case syncStateDone:
		if m.pullOnly {
			if strings.Contains(m.pullOutput, "Already up to date") {
				return successStyle.Render("✓ Already up to date")
			}
			return successStyle.Render("✓ Pulled changes successfully")
		}

		// Full sync
		pullMsg := "pulled"
		if strings.Contains(m.pullOutput, "Already up to date") {
			pullMsg = "up to date"
		}

		pushMsg := "pushed"
		if strings.Contains(m.pushOutput, "Everything up-to-date") {
			pushMsg = "up to date"
		}

		return successStyle.Render(fmt.Sprintf("✓ Sync complete (%s, %s)", pullMsg, pushMsg))

	case syncStateError:
		return errorStyle.Render(fmt.Sprintf("✗ Error: %s", m.err))
	}

	return ""
}

func checkSync() tea.Msg {
	hasRemote, err := CheckRemoteExists()
	if err != nil {
		return syncCheckMsg{err: err}
	}

	hasChanges, err := CheckForUncommittedChanges()
	if err != nil {
		return syncCheckMsg{err: err}
	}

	branch, err := GetCurrentBranch()
	if err != nil {
		return syncCheckMsg{err: err}
	}

	hasUpstream, err := HasUpstreamBranch()
	if err != nil {
		return syncCheckMsg{err: err}
	}

	return syncCheckMsg{
		hasRemote:   hasRemote,
		hasChanges:  hasChanges,
		branch:      branch,
		hasUpstream: hasUpstream,
	}
}

func pullChanges() tea.Msg {
	output, err := PullChanges()
	return syncPullMsg{output: output, err: err}
}

func pushChanges(branch string) tea.Cmd {
	return func() tea.Msg {
		hasUpstream, _ := HasUpstreamBranch()

		var output string
		var err error

		if !hasUpstream {
			output, err = PushWithUpstream(branch)
		} else {
			output, err = PushChanges()
		}

		return syncPushMsg{output: output, err: err}
	}
}
