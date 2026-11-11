package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CiaranMccarthy1/boba-text/pkg/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// maxOpenFileBytes caps files opened in the editor to keep the TUI responsive.
const maxOpenFileBytes = 10 * 1024 * 1024

type Focus int

const (
	FocusFileTree Focus = iota
	FocusEditor
	FocusAgent
)

type Model struct {
	fileTree    FileTreeModel
	editor      EditorModel
	agent       AgentModel
	focus       Focus
	showTree    bool
	showWelcome bool
	width       int
	height      int
	keys        config.Keys
	startPath   string
}

// InitialModel creates the initial application model with the given configuration.
func InitialModel(startPath string, cfg config.Config) Model {
	InitStyles(cfg.Colors)
	return Model{
		fileTree:    NewFileTree(startPath, cfg.Keys),
		editor:      NewEditor(cfg.Commands, cfg.Keys),
		agent:       NewAgent(cfg.AI, cfg.Keys),
		focus:       FocusFileTree,
		showTree:    true,
		showWelcome: true,
		keys:        cfg.Keys,
		startPath:   startPath,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.fileTree.Init(), m.editor.Init(), m.agent.Init())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Dismiss welcome screen on any key
		if m.showWelcome {
			m.showWelcome = false
			return m, nil
		}

		switch msg.String() {
		case m.keys.Quit:
			if m.editor.modified {
				m.editor.msg = "No write since last change (use :q!)"
				return m, nil
			}
			return m, tea.Quit
		case m.keys.CycleFocus:
			if m.focus == FocusEditor && m.editor.mode == ModeInsert {
				break
			}
			m.focus = (m.focus + 1) % 3
			return m, nil

		case m.keys.Save:
			m.editor.saveFile()
			return m, nil

		case m.keys.ToggleTree:
			m.showTree = !m.showTree
			m.resizePanes()
			return m, nil

		case m.keys.FocusTree:
			if m.focus == FocusFileTree {
				m.focus = FocusEditor
			} else {
				m.focus = FocusFileTree
				if !m.showTree {
					m.showTree = true
					m.resizePanes()
				}
			}
			return m, nil

		case m.keys.FocusAgent:
			if m.focus == FocusAgent {
				m.focus = FocusEditor
			} else {
				m.focus = FocusAgent
			}
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizePanes()

	case AIResponseMsg:
		// Always deliver to the agent, even if the user moved focus away mid-request.
		m.agent, cmd = m.agent.Update(msg)
		return m, cmd

	case OpenFileMsg:
		m.showWelcome = false
		m.focus = FocusEditor
		info, err := os.Stat(msg.Path)
		if err == nil && info.Size() > maxOpenFileBytes {
			m.editor.msg = fmt.Sprintf("File too large (%d MB): %s", info.Size()/(1024*1024), msg.Path)
			break
		}
		content, err := os.ReadFile(msg.Path)
		if err != nil {
			m.editor.msg = "Cannot open: " + err.Error()
			break
		}
		m.editor.SetContent(string(content), msg.Path)
		m.agent.SetCurrentFile(msg.Path)
	}

	switch m.focus {
	case FocusFileTree:
		if m.showTree {
			m.fileTree, cmd = m.fileTree.Update(msg)
			cmds = append(cmds, cmd)
		}
	case FocusEditor:
		m.editor, cmd = m.editor.Update(msg)
		cmds = append(cmds, cmd)
	case FocusAgent:
		m.agent, cmd = m.agent.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) resizePanes() {
	// Content pane width: terminal minus the rounded border (2), minus the
	// tree column (30) and its StyleFileTree padding (1,2) when shown.
	contentWidth := m.width - 2
	if m.showTree {
		contentWidth -= 34
		m.fileTree.width = 30
		m.fileTree.height = m.height - 2
	}
	if contentWidth < 10 {
		contentWidth = 10
	}
	contentHeight := m.height - 2

	m.editor.width = contentWidth
	m.editor.height = contentHeight
	m.editor.textarea.SetWidth(contentWidth - 2)
	m.editor.textarea.SetHeight(contentHeight - 4)

	m.agent.SetSize(contentWidth, contentHeight)
}

// renderWelcome renders the alpha.nvim-style welcome screen.
func (m Model) renderWelcome() string {
	logo := []string{
		"",
		"  ██████╗  ██████╗ ██████╗  █████╗ ",
		"  ██╔══██╗██╔═══██╗██╔══██╗██╔══██╗",
		"  ██████╔╝██║   ██║██████╔╝███████║",
		"  ██╔══██╗██║   ██║██╔══██╗██╔══██╗",
		"  ██████╔╝╚██████╔╝██████╔╝██║  ██║",
		"  ╚═════╝  ╚═════╝ ╚═════╝ ╚═╝  ╚═╝",
		"",
		"         T  E  X  T    E D I T O R",
		"",
	}

	actions := []string{
		"",
		"  Press  i  to start editing",
		"  Press  Ctrl+E  to browse files",
		"  Press  Ctrl+A  for AI agent",
		"  Press  Ctrl+C  to quit",
		"",
		"  ──────────────────────────────",
		"  Project: " + filepath.Base(m.startPath),
		"",
	}

	var b strings.Builder
	for _, line := range logo {
		b.WriteString(StyleWelcome.Render(line) + "\n")
	}
	for _, line := range actions {
		b.WriteString(StyleWelcomeDim.Render(line) + "\n")
	}

	return b.String()
}

func (m Model) View() string {
	var content string

	if m.showWelcome {
		content = m.renderWelcome()
	} else if m.focus == FocusAgent {
		content = m.agent.View()
	} else {
		content = m.editor.View()
	}

	var borderColor lipgloss.Color
	switch m.focus {
	case FocusAgent:
		borderColor = ColorSuccess
	case FocusEditor:
		borderColor = ColorPrimary
	default:
		borderColor = ColorSubText
	}

	contentStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(m.editor.width).
		Height(m.editor.height)

	if m.showTree {
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.fileTree.View(),
			contentStyle.Render(content),
		)
	}

	return contentStyle.Render(content)
}
