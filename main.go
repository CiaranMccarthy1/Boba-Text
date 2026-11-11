package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/CiaranMccarthy1/boba-text/pkg/config"
	"github.com/CiaranMccarthy1/boba-text/pkg/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [file]\n\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println(version())
		return
	}

	startDir, openFile, err := resolveStart(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "boba-text:", err)
		os.Exit(1)
	}

	cfg := config.Load()
	p := tea.NewProgram(tui.InitialModel(startDir, openFile, cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "boba-text:", err)
		os.Exit(1)
	}
}

// resolveStart maps an optional command-line path to a tree root and a file to open.
func resolveStart(arg string) (startDir, openFile string, err error) {
	startDir, err = os.Getwd()
	if err != nil {
		return "", "", err
	}
	if arg == "" {
		return startDir, "", nil
	}
	abs, err := filepath.Abs(arg)
	if err != nil {
		return "", "", err
	}
	if info, err := os.Stat(abs); err == nil && info.IsDir() {
		return abs, "", nil
	}
	return filepath.Dir(abs), abs, nil
}

func version() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return "boba-text " + info.Main.Version
	}
	return "boba-text dev"
}
