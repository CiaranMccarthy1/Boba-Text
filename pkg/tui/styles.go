package tui

import (
	"github.com/CiaranMccarthy1/boba-text/pkg/config"
	"github.com/charmbracelet/lipgloss"
)

var (
	ColorText      lipgloss.Color
	ColorSubText   lipgloss.Color
	ColorPrimary   lipgloss.Color
	ColorSecondary lipgloss.Color
	ColorAccent    lipgloss.Color
	ColorSuccess   lipgloss.Color
	ColorWarning   lipgloss.Color
	ColorError     lipgloss.Color

	StyleDim        lipgloss.Style
	StyleBold       lipgloss.Style
	StyleSelected   lipgloss.Style
	StyleFileTree   lipgloss.Style
	StyleEditor     lipgloss.Style
	StyleAgent      lipgloss.Style
	StyleModeNormal lipgloss.Style
	StyleModeInsert lipgloss.Style
	StyleWelcome    lipgloss.Style
	StyleWelcomeDim lipgloss.Style
	StyleFileIcon   lipgloss.Style
	StyleDirIcon    lipgloss.Style
)

// InitStyles initializes all global styles with the given color configuration.
func InitStyles(c config.Colors) {
	ColorText = lipgloss.Color(c.Text)
	ColorSubText = lipgloss.Color(c.SubText)
	ColorPrimary = lipgloss.Color(c.Primary)
	ColorSecondary = lipgloss.Color(c.Secondary)
	ColorAccent = lipgloss.Color(c.Accent)
	ColorSuccess = lipgloss.Color(c.Success)
	ColorWarning = lipgloss.Color(c.Warning)
	ColorError = lipgloss.Color(c.Error)

	StyleDim = lipgloss.NewStyle().Foreground(ColorSubText)
	StyleBold = lipgloss.NewStyle().Bold(true).Foreground(ColorText)

	StyleSelected = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	StyleFileTree = lipgloss.NewStyle().
		Padding(1, 2)

	StyleEditor = lipgloss.NewStyle().
		Padding(0, 1)

	StyleAgent = lipgloss.NewStyle().
		Padding(0, 1)

	// Mode indicators (Neovim-style lualine colors)
	StyleModeNormal = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1E1E1E")).
		Background(ColorAccent).
		Bold(true).
		Padding(0, 1)

	StyleModeInsert = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1E1E1E")).
		Background(ColorSuccess).
		Bold(true).
		Padding(0, 1)

	// Welcome screen
	StyleWelcome = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	StyleWelcomeDim = lipgloss.NewStyle().
		Foreground(ColorSubText).
		Italic(true)

	// File tree icons
	StyleFileIcon = lipgloss.NewStyle().
		Foreground(ColorAccent)

	StyleDirIcon = lipgloss.NewStyle().
		Foreground(ColorWarning)
}
