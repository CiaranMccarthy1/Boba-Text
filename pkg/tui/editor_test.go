package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CiaranMccarthy1/boba-text/pkg/config"
	tea "github.com/charmbracelet/bubbletea"
)

func TestFindMatchForward(t *testing.T) {
	cases := []struct {
		name             string
		lines            []string
		query            string
		row, col         int
		wantRow, wantCol int
		wantOK           bool
	}{
		{"find at start", []string{"hello world", "abc hello", "hello"}, "hello", 0, 0, 0, 0, true},
		{"skip match under cursor", []string{"hello world", "abc hello", "hello"}, "hello", 0, 5, 1, 4, true},
		// wrap-around lives in findNextMatch, not here
		{"no match past end", []string{"hello world", "abc hello"}, "hello", 2, 0, 0, 0, false},
		{"no match", []string{"hello world"}, "zebra", 0, 0, 0, 0, false},
		{"multibyte runes", []string{"héllo héllo"}, "héllo", 0, 4, 0, 6, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			row, col, ok := findMatchForward(c.lines, c.query, c.row, c.col)
			if ok != c.wantOK || row != c.wantRow || col != c.wantCol {
				t.Errorf("findMatchForward(%q, start %d:%d) = (%d,%d,%v), want (%d,%d,%v)",
					c.query, c.row, c.col, row, col, ok, c.wantRow, c.wantCol, c.wantOK)
			}
		})
	}
}

func TestFindMatchBackward(t *testing.T) {
	lines := []string{"one two one two"}
	cases := []struct {
		name             string
		query            string
		row, col         int
		wantRow, wantCol int
		wantOK           bool
	}{
		{"earlier on same line", "one", 0, 14, 0, 8, true},
		{"first on line", "one", 0, 3, 0, 0, true},
		{"no match", "zebra", 0, 14, 0, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			row, col, ok := findMatchBackward(lines, c.query, c.row, c.col)
			if ok != c.wantOK || row != c.wantRow || col != c.wantCol {
				t.Errorf("findMatchBackward(%q, start %d:%d) = (%d,%d,%v), want (%d,%d,%v)",
					c.query, c.row, c.col, row, col, ok, c.wantRow, c.wantCol, c.wantOK)
			}
		})
	}
}

func TestRuneByteConversions(t *testing.T) {
	// "héllo": 5 runes, 6 bytes (é is 2 bytes)
	cases := []struct {
		s    string
		in   int
		want int
	}{
		{"héllo", 0, 0},
		{"héllo", 2, 3},  // after h(1) + é(2)
		{"héllo", 99, 6}, // past end clamps to byte length
	}
	for _, c := range cases {
		if got := runeIndexToByteIndex(c.s, c.in); got != c.want {
			t.Errorf("runeIndexToByteIndex(%q, %d) = %d, want %d", c.s, c.in, got, c.want)
		}
	}
	cases = []struct {
		s    string
		in   int
		want int
	}{
		{"héllo", 3, 2}, // byte 3 is after "hé"
		{"héllo", 0, 0},
		{"héllo", 99, 5}, // past end clamps to rune count
	}
	for _, c := range cases {
		if got := byteIndexToRuneIndex(c.s, c.in); got != c.want {
			t.Errorf("byteIndexToRuneIndex(%q, %d) = %d, want %d", c.s, c.in, got, c.want)
		}
	}
}

// assertQuit checks that cmd is non-nil and produces a tea.QuitMsg.
func assertQuit(t *testing.T, cmd tea.Cmd, label string) {
	t.Helper()
	if cmd == nil {
		t.Fatalf("%s: got nil cmd, want quit", label)
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("%s: cmd did not produce QuitMsg", label)
	}
}

func TestExecuteCommand(t *testing.T) {
	e := NewEditor(
		config.Commands{Quit: []string{"q", "quit"}, Save: []string{"w"}},
		config.Keys{},
	)

	assertQuit(t, e.executeCommand("q"), ":q when unmodified")

	e.modified = true
	if cmd := e.executeCommand("q"); cmd != nil {
		t.Error(":q with unsaved changes should not quit")
	}
	if e.msg == "" {
		t.Error(":q with unsaved changes should show a message")
	}

	assertQuit(t, e.executeCommand("q!"), ":q! forces quit even when modified")
	e.modified = false
	assertQuit(t, e.executeCommand("x"), ":x saves and quits")

	if cmd := e.executeCommand("zzz"); cmd != nil || e.msg != "Unknown command: zzz" {
		t.Errorf("unknown command: got cmd=%v msg=%q", cmd, e.msg)
	}

	// Line-number jump on an empty buffer must not panic or quit.
	if cmd := e.executeCommand("42"); cmd != nil {
		t.Error(":42 should not quit")
	}
}

func TestWQQuitGuard(t *testing.T) {
	// A directory as filename always fails to save.
	dir := t.TempDir()
	e := NewEditor(config.Commands{}, config.Keys{})
	e.SetContent("hi", dir)
	e.modified = true
	if cmd := e.executeCommand("wq"); cmd != nil {
		t.Error(":wq should NOT quit when the save fails")
	}
	if !e.modified {
		t.Error(":wq after failed save should leave the buffer marked modified")
	}

	// A real file: save succeeds, so :wq quits.
	f := t.TempDir() + "/ok.txt"
	e.SetContent("hi", f)
	e.modified = true
	assertQuit(t, e.executeCommand("wq"), ":wq after successful save")
}

func TestUndoRedo(t *testing.T) {
	e := NewEditor(config.Commands{}, config.Keys{})
	e.SetContent("hello", "f.txt")

	// `x` deletes one char; each delete is its own undo step.
	e.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if got := e.textarea.Value(); got != "ello" {
		t.Fatalf("after x: got %q, want %q", got, "ello")
	}
	e.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if got := e.textarea.Value(); got != "hello" {
		t.Errorf("after u: got %q, want %q", got, "hello")
	}
	// Redo brings the deletion back.
	e.handleNormalMode(tea.KeyMsg{Type: tea.KeyCtrlR})
	if got := e.textarea.Value(); got != "ello" {
		t.Errorf("after ctrl+r: got %q, want %q", got, "ello")
	}

	// Insert-mode typing coalesces into one undo step.
	e = NewEditor(config.DefaultConfig().Commands, config.DefaultConfig().Keys)
	e.SetContent("hello", "f.txt")
	e.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	e.applyTextareaKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	e.applyTextareaKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
	e.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if got := e.textarea.Value(); got != "hello" {
		t.Errorf("after coalesced undo: got %q, want %q", got, "hello")
	}

	// Undo on a fresh history says so.
	e2 := NewEditor(config.Commands{}, config.Keys{})
	e2.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if e2.msg != "Nothing to undo" {
		t.Errorf("undo with empty stack: got msg %q", e2.msg)
	}
}

func TestOpenNewFile(t *testing.T) {
	// ":e <missing>" must create an empty, saveable buffer.
	dir := t.TempDir()
	missing := dir + "/new.txt"
	m := InitialModel(dir, config.DefaultConfig())
	m2, _ := m.Update(OpenFileMsg{Path: missing})
	got := m2.(Model)
	if got.editor.filename != missing {
		t.Errorf("filename = %q, want %q", got.editor.filename, missing)
	}
	if got.editor.textarea.Value() != "" {
		t.Errorf("new buffer should be empty, got %q", got.editor.textarea.Value())
	}
	if got.editor.msg != "" {
		t.Errorf("creating a new file should not show an error, got %q", got.editor.msg)
	}
}

func TestOpenRelativePath(t *testing.T) {
	dir := t.TempDir()
	abs := filepath.Join(dir, "rel.txt")
	if err := os.WriteFile(abs, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := InitialModel(dir, config.DefaultConfig())
	m2, _ := m.Update(OpenFileMsg{Path: "rel.txt"})
	if got := m2.(Model).editor.filename; got != abs {
		t.Errorf("filename = %q, want %q (resolved against the tree)", got, abs)
	}
}

func TestBinaryFileRefused(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin.dat")
	if err := os.WriteFile(bin, []byte("abc\x00def"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := InitialModel(dir, config.DefaultConfig())
	m2, _ := m.Update(OpenFileMsg{Path: bin})
	got := m2.(Model)
	if got.editor.filename != "" {
		t.Error("binary file was loaded into the editor")
	}
	if !strings.Contains(got.editor.msg, "Binary") {
		t.Errorf("msg = %q, want a Binary warning", got.editor.msg)
	}
}

func TestInsertArrowNotModified(t *testing.T) {
	cfg := config.DefaultConfig()
	e := NewEditor(cfg.Commands, cfg.Keys)
	e.SetContent("hello", "f.txt")
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyUp})
	if e.modified {
		t.Error("arrow key in insert mode marked the buffer modified")
	}
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if !e.modified {
		t.Error("typing did not mark the buffer modified")
	}
}

func TestFileIcons(t *testing.T) {
	// Regression: every known extension must render a non-empty icon.
	for _, name := range []string{"a.go", "a.js", "a.ts", "a.py", "a.rs", "a.md", "a.json", "a.toml", "a.html", "a.css", "a.sh", "a.png", "a.mod", "a.txt", "a.xyz"} {
		if fileIcon(name, false) == "" {
			t.Errorf("fileIcon(%q) is empty", name)
		}
	}
	if fileIcon("sub", true) == "" {
		t.Error("dir icon is empty")
	}
	if fileIcon("..", true) != "⮤" {
		t.Error("parent entry icon changed")
	}
}
