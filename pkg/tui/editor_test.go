package tui

import (
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
