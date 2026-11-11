package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// tempConfigDir points Load at an empty temp directory for one test.
func tempConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old := userConfigDir
	userConfigDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { userConfigDir = old })
	return dir
}

func writeConfig(t *testing.T, dir, body string) {
	t.Helper()
	path := filepath.Join(dir, "boba-text", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDefaultsWhenMissing(t *testing.T) {
	tempConfigDir(t)
	if got, want := Load(), DefaultConfig(); !reflect.DeepEqual(got, want) {
		t.Errorf("Load() = %+v, want defaults", got)
	}
}

func TestLoadPartialOverrides(t *testing.T) {
	dir := tempConfigDir(t)
	writeConfig(t, dir, `
[colors]
primary = "#123456"

[keys]
toggle_tree = "ctrl+g"

[ai]
provider = "openai"
model = "gpt-4o-mini"

[commands]
save = ["w"]
`)
	got := Load()
	want := DefaultConfig()

	if got.Colors.Primary != "#123456" {
		t.Errorf("primary = %q, want #123456", got.Colors.Primary)
	}
	if got.Colors.Text != want.Colors.Text {
		t.Errorf("text = %q, want the default %q", got.Colors.Text, want.Colors.Text)
	}
	if got.Keys.ToggleTree != "ctrl+g" {
		t.Errorf("toggle_tree = %q, want ctrl+g", got.Keys.ToggleTree)
	}
	if got.Keys.Quit != want.Keys.Quit {
		t.Errorf("quit = %q, want the default %q", got.Keys.Quit, want.Keys.Quit)
	}
	if got.AI.Provider != "openai" || got.AI.Model != "gpt-4o-mini" {
		t.Errorf("ai = %+v, want openai / gpt-4o-mini", got.AI)
	}
	if !reflect.DeepEqual(got.Commands.Save, []string{"w"}) {
		t.Errorf("commands.save = %v, want [w]", got.Commands.Save)
	}
}

func TestLoadInvalidFallsBack(t *testing.T) {
	dir := tempConfigDir(t)
	writeConfig(t, dir, "[colors\nprimary = 1")
	if got, want := Load(), DefaultConfig(); !reflect.DeepEqual(got, want) {
		t.Errorf("Load() = %+v, want defaults after invalid TOML", got)
	}
}

func TestLoadUnknownKeyIgnored(t *testing.T) {
	dir := tempConfigDir(t)
	// "dark" was removed; old configs must keep working.
	writeConfig(t, dir, "[colors]\ndark = \"#000000\"\n")
	if got, want := Load(), DefaultConfig(); !reflect.DeepEqual(got, want) {
		t.Errorf("Load() = %+v, want defaults with an unknown key", got)
	}
}
