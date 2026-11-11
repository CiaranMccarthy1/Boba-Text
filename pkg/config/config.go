package config

import (
	"fmt"
	"os"

	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Colors struct {
	Text string `toml:"text"`

	SubText string `toml:"subtext"`

	Primary string `toml:"primary"`

	Secondary string `toml:"secondary"`

	Accent string `toml:"accent"`

	Success string `toml:"success"`

	Warning string `toml:"warning"`

	Error string `toml:"error"`

	Dark string `toml:"dark"`
}

type Keys struct {
	ToggleTree string `toml:"toggle_tree"`

	FocusTree string `toml:"focus_tree"`

	FocusAgent string `toml:"focus_agent"`

	CycleFocus string `toml:"cycle_focus"`

	Quit string `toml:"quit"`

	Save string `toml:"save"`

	TreeUp       string `toml:"tree_up"`
	TreeUpAlt    string `toml:"tree_up_alt"`
	TreeDown     string `toml:"tree_down"`
	TreeDownAlt  string `toml:"tree_down_alt"`
	TreeOpen     string `toml:"tree_open"`
	TreeBack     string `toml:"tree_back"`
	TreeBackAlt  string `toml:"tree_back_alt"`
	TreeBackAlt2 string `toml:"tree_back_alt_2"`

	EditorInsertMode  string `toml:"editor_insert_mode"`
	EditorCommandMode string `toml:"editor_command_mode"`
	EditorNormalMode  string `toml:"editor_normal_mode"`
	EditorCommandRun  string `toml:"editor_command_run"`

	AgentSend string `toml:"agent_send"`
}

type AI struct {
	Provider string `toml:"provider"`

	Name string `toml:"name"`

	Model string `toml:"model"`
}

type Commands struct {
	Save []string `toml:"save"`

	Quit []string `toml:"quit"`
}

type Config struct {
	Colors Colors `toml:"colors"`

	Keys Keys `toml:"keys"`

	AI AI `toml:"ai"`

	Commands Commands `toml:"commands"`
}

// DefaultConfig returns the default configuration with sensible preset values.

func DefaultConfig() Config {
	return Config{
		Colors: Colors{
			Text:      "#FAFAFA",
			SubText:   "#7D7D7D",
			Primary:   "#F25D94", // Neon Pink
			Secondary: "#A550DF", // Purple
			Accent:    "#61AFEF", // Blue
			Success:   "#98C379", // Green
			Warning:   "#E5C07B", // Yellow
			Error:     "#E06C75", // Red
			Dark:      "#1E1E1E",
		},
		Keys: Keys{
			ToggleTree:        "ctrl+t",
			FocusTree:         "ctrl+e",
			FocusAgent:        "ctrl+a",
			CycleFocus:        "tab",
			Quit:              "ctrl+c",
			Save:              "ctrl+s",
			TreeUp:            "up",
			TreeUpAlt:         "k",
			TreeDown:          "down",
			TreeDownAlt:       "j",
			TreeOpen:          "enter",
			TreeBack:          "backspace",
			TreeBackAlt:       "h",
			TreeBackAlt2:      "left",
			EditorInsertMode:  "i",
			EditorCommandMode: ":",
			EditorNormalMode:  "esc",
			EditorCommandRun:  "enter",
			AgentSend:         "enter",
		},
		AI: AI{
			Provider: "gemini",
			Name:     "Agent",
			Model:    "gemini-2.0-flash",
		},
		Commands: Commands{
			Save: []string{"w", "s", "save", "write"},
			Quit: []string{"q", "quit", "exit"},
		},
	}
}

// Load attempts to load configuration from local file or home directory, falling back to defaults.

func Load() Config {

	configDir, err := os.UserConfigDir()

	if err != nil {

		return DefaultConfig()

	}

	path := filepath.Join(configDir, "boba-text", "config.toml")

	if _, err := os.Stat(path); err != nil {

		return DefaultConfig()

	}

	cfg := DefaultConfig()

	// Unmarshal over defaults (this is a simple way, though slightly imperfect for deep structs, works here)

	if _, err := toml.DecodeFile(path, &cfg); err != nil {

		fmt.Fprintf(os.Stderr, "boba-text: ignoring invalid config %s: %v\n", path, err)

		return DefaultConfig()

	}

	return cfg

}

//Test comment
