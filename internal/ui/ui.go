// Package ui provides a terminal user interface for Reflex using Bubbletea.
package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Message types for external communication via p.Send()

// StatusUpdateMsg updates the header status text.
type StatusUpdateMsg struct {
	Status string
}

// ProcessOutputLineMsg appends a line to the log viewport.
type ProcessOutputLineMsg struct {
	Line string
}

// ClearLogsMsg clears all logs from the viewport.
type ClearLogsMsg struct{}

// Styles
var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			MarginBottom(1)

	statusRunning = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)

	statusRestarting = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFCC00")).
				Bold(true)

	statusStopped = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Bold(true)

	viewportStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			MarginTop(1)
)

// Model represents the TUI state.
type Model struct {
	viewport    viewport.Model
	status      string
	logs        []string
	ready       bool
	width       int
	height      int
}

// New creates a new UI model with default values.
func New() Model {
	return Model{
		status: "Initializing",
		logs:   []string{},
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 3 // header + margin
		helpHeight := 2   // help text + margin
		viewportHeight := m.height - headerHeight - helpHeight - 2 // border padding

		if !m.ready {
			m.viewport = viewport.New(m.width-4, viewportHeight)
			m.viewport.SetContent(strings.Join(m.logs, "\n"))
			m.ready = true
		} else {
			m.viewport.Width = m.width - 4
			m.viewport.Height = viewportHeight
		}

	case StatusUpdateMsg:
		m.status = msg.Status

	case ProcessOutputLineMsg:
		m.logs = append(m.logs, msg.Line)
		if m.ready {
			m.viewport.SetContent(strings.Join(m.logs, "\n"))
			m.viewport.GotoBottom()
		}

	case ClearLogsMsg:
		m.logs = []string{}
		if m.ready {
			m.viewport.SetContent("")
		}
	}

	if m.ready {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model.
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Render header with styled status
	styledStatus := m.styledStatus()
	header := headerStyle.Render("⚡ Reflex") + " " + styledStatus

	// Render viewport with border
	viewportContent := viewportStyle.Render(m.viewport.View())

	// Help text
	help := helpStyle.Render("↑/↓: scroll • q: quit")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		viewportContent,
		help,
	)
}

// styledStatus returns the status text with appropriate styling.
func (m Model) styledStatus() string {
	status := strings.ToLower(m.status)

	switch {
	case strings.Contains(status, "running"):
		return statusRunning.Render("● " + m.status)
	case strings.Contains(status, "restart"):
		return statusRestarting.Render("◐ " + m.status)
	case strings.Contains(status, "stop"):
		return statusStopped.Render("○ " + m.status)
	default:
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Render("◌ " + m.status)
	}
}
