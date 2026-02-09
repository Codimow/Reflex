package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type (
	FileChangedMsg       struct{}
	ProcessOutputLineMsg struct{ Line string }
	ClearLogsMsg         struct{}
	StatusUpdateMsg      struct{ Status string }
)

type Model struct {
	viewport    viewport.Model
	status      string
	logContent  strings.Builder
	width       int
	height      int
	headerStyle lipgloss.Style
	ready       bool
}

func NewModel() Model {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	return Model{
		status:      "Initializing...",
		headerStyle: headerStyle,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.renderHeader())
		if !m.ready {
			m.width = msg.Width
			m.height = msg.Height
			m.viewport = viewport.New(m.width, m.height-headerHeight)
			m.viewport.SetContent("Initializing Reflex...\n")
			m.ready = true
		} else {
			m.width = msg.Width
			m.height = msg.Height
			m.viewport.Width = m.width
			m.viewport.Height = m.height - headerHeight
		}

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

	case StatusUpdateMsg:
		m.status = msg.Status

	case ClearLogsMsg:
		m.logContent.Reset()
		m.logContent.WriteString(fmt.Sprintf("🔄 Process restarting due to file change...\n\n"))
		m.viewport.SetContent(m.logContent.String())


	case ProcessOutputLineMsg:
		m.logContent.WriteString(msg.Line + "\n")
		m.viewport.SetContent(m.logContent.String())
		m.viewport.GotoBottom()
	}

	var cmd tea.Cmd
	if m.ready {
		m.viewport, cmd = m.viewport.Update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}
	return fmt.Sprintf("%s\n%s", m.renderHeader(), m.viewport.View())
}

func (m Model) renderHeader() string {
	return m.headerStyle.Width(m.width).Render("Reflex | Status: " + m.status)
}
