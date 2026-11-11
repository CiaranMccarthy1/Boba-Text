package tui

import (
	"fmt"
	"strings"

	"github.com/CiaranMccarthy1/boba-text/pkg/ai"
	"github.com/CiaranMccarthy1/boba-text/pkg/config"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AIResponseMsg carries the AI response back to the TUI from the async call.
type AIResponseMsg struct {
	Response string
	Err      error
}

type AgentModel struct {
	viewport    viewport.Model
	textarea    textarea.Model
	messages    []string
	senderStyle lipgloss.Style
	aiStyle     lipgloss.Style
	errorStyle  lipgloss.Style
	width       int
	height      int
	config      config.AI
	keys        config.Keys
	waiting     bool
	currentFile string
}

// NewAgent creates a new AI agent model with the given configuration.
func NewAgent(aiConfig config.AI, keyConfig config.Keys) AgentModel {
	ta := textarea.New()
	ta.Placeholder = "Ask your AI assistant..."
	ta.Focus()
	ta.CharLimit = 0
	ta.SetHeight(3)
	ta.ShowLineNumbers = false

	vp := viewport.New(0, 0)

	welcome := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("Boba AI Agent") + "\n\n"
	welcome += lipgloss.NewStyle().Foreground(ColorSubText).Render("Ask questions about your code and request refactors.\n\n")
	welcome += lipgloss.NewStyle().Foreground(ColorAccent).Render("Tips:\n")
	welcome += lipgloss.NewStyle().Foreground(ColorSubText).Render("  • \"explain this code\" — analyzes the current file\n")
	welcome += lipgloss.NewStyle().Foreground(ColorSubText).Render("  • \"refactor for readability\" — suggests improvements\n\n")
	welcome += lipgloss.NewStyle().Foreground(ColorWarning).Render("Set one API key env var to enable AI:\n")
	welcome += lipgloss.NewStyle().Foreground(ColorSubText).Render("  gemini: GEMINI_API_KEY, openai: OPENAI_API_KEY,\n")
	welcome += lipgloss.NewStyle().Foreground(ColorSubText).Render("  claude: ANTHROPIC_API_KEY, openrouter: OPENROUTER_API_KEY\n")

	vp.SetContent(welcome)

	return AgentModel{
		textarea:    ta,
		viewport:    vp,
		messages:    []string{},
		senderStyle: lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true),
		aiStyle:     lipgloss.NewStyle().Foreground(ColorSuccess),
		errorStyle:  lipgloss.NewStyle().Foreground(ColorError),
		config:      aiConfig,
		keys:        keyConfig,
	}
}

func (m AgentModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m AgentModel) Update(msg tea.Msg) (AgentModel, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case AIResponseMsg:
		m.waiting = false
		if msg.Err != nil {
			errMsg := m.errorStyle.Render("Error: ") + msg.Err.Error()
			m.messages = append(m.messages, errMsg)
		} else {
			aiName := m.config.Name
			if aiName == "" {
				aiName = "AI"
			}
			aiMsg := m.aiStyle.Render(aiName+": ") + msg.Response
			m.messages = append(m.messages, aiMsg)
		}
		m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
		m.viewport.GotoBottom()

	case tea.KeyMsg:
		switch msg.String() {
		case m.keys.AgentSend:
			if m.textarea.Value() == "" || m.waiting {
				break
			}
			userInput := m.textarea.Value()
			userMsg := m.senderStyle.Render("You: ") + userInput
			m.messages = append(m.messages, userMsg)

			// Send to the configured AI provider asynchronously
			m.waiting = true
			m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
			m.textarea.Reset()
			m.viewport.GotoBottom()

			return m, tea.Batch(tiCmd, vpCmd, func() tea.Msg {
				resp, err := ai.Send(m.config.Provider, m.config.Model, m.currentFile, userInput)
				return AIResponseMsg{Response: resp, Err: err}
			})
		}
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m AgentModel) View() string {
	var statusLine string
	if m.waiting {
		statusLine = StyleDim.Render("Generating...")
	}

	return StyleAgent.Render(
		fmt.Sprintf(
			"%s\n%s\n%s",
			m.viewport.View(),
			statusLine,
			m.textarea.View(),
		),
	)
}

func (m *AgentModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.textarea.SetWidth(w)
	m.viewport.Width = w
	m.viewport.Height = h - m.textarea.Height() - 4
}

// SetCurrentFile tells the agent what file is currently open in the editor.
func (m *AgentModel) SetCurrentFile(path string) {
	m.currentFile = path
}
