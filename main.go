package main

import (
	"fmt"

	"os"

	"github.com/CiaranMccarthy1/boba-text/pkg/config"

	"github.com/CiaranMccarthy1/boba-text/pkg/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {

	cwd, err := os.Getwd()

	if err != nil {

		fmt.Printf("Error getting cwd: %v\n", err)

		os.Exit(1)

	}

	cfg := config.Load()

	p := tea.NewProgram(tui.InitialModel(cwd, cfg), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {

		fmt.Printf("Alas, there's been an error: %v", err)

		os.Exit(1)

	}

}
