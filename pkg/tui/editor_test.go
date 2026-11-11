package tui

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
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

	// :e! forces a discard; :e keeps the unsaved-changes guard.
	cmd := e.executeCommand("e! foo.txt")
	if cmd == nil {
		t.Fatal(":e! should return an open command")
	}
	msg, ok := cmd().(OpenFileMsg)
	if !ok || msg.Path != "foo.txt" || !msg.Force {
		t.Errorf(":e!: got %#v, want an OpenFileMsg{Path: %q, Force: true}", msg, "foo.txt")
	}
	if cmd := e.executeCommand("e foo.txt"); cmd == nil {
		t.Fatal(":e should return an open command")
	} else if msg := cmd().(OpenFileMsg); msg.Force {
		t.Error(":e should not force-discard unsaved changes")
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
	m := InitialModel(dir, "", config.DefaultConfig())
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
	if got.agent.currentFile != missing {
		t.Errorf("agent.currentFile = %q, want %q", got.agent.currentFile, missing)
	}
}

func TestOpenRelativePath(t *testing.T) {
	dir := t.TempDir()
	abs := filepath.Join(dir, "rel.txt")
	if err := os.WriteFile(abs, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := InitialModel(dir, "", config.DefaultConfig())
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
	m := InitialModel(dir, "", config.DefaultConfig())
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

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("old"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(path, []byte("new")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "new" {
		t.Fatalf("content = %q, %v; want %q", got, err, "new")
	}
	if info, err := os.Stat(path); err == nil && info.Mode().Perm() != 0o640 {
		t.Errorf("mode = %v, want 0640 preserved", info.Mode().Perm())
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("left %d files behind, want only f.txt", len(entries))
	}
}

func TestWriteFileAtomicKeepsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need extra privileges on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	link := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(link, []byte("new")); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("save replaced the symlink with a regular file")
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "new" {
		t.Fatalf("target content = %q, %v; want %q", got, err, "new")
	}
}

func TestUnsavedChangesGuard(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	for _, f := range []string{a, b} {
		if err := os.WriteFile(f, []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	m := InitialModel(dir, "", config.DefaultConfig())
	m2, _ := m.Update(OpenFileMsg{Path: a})
	m = m2.(Model)
	m.editor.textarea.SetValue("edited")
	m.editor.modified = true

	// Opening another file must not replace the unsaved buffer.
	m2, _ = m.Update(OpenFileMsg{Path: b})
	m = m2.(Model)
	if m.editor.filename != a {
		t.Errorf("unsaved buffer was replaced: filename = %q, want %q", m.editor.filename, a)
	}
	if !strings.Contains(m.editor.msg, "Unsaved") {
		t.Errorf("msg = %q, want an unsaved-changes warning", m.editor.msg)
	}

	// :e! discards and opens the new file.
	m2, _ = m.Update(OpenFileMsg{Path: b, Force: true})
	m = m2.(Model)
	if m.editor.filename != b || m.editor.textarea.Value() != "content" {
		t.Errorf("force open: got (%q, %q), want (%q, %q)", m.editor.filename, m.editor.textarea.Value(), b, "content")
	}
}

func TestInitOpensFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "x.txt")
	m := InitialModel(filepath.Dir(file), file, config.DefaultConfig())
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init returned no command")
	}
	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("Init returned %T, want tea.BatchMsg", cmd())
	}
	for _, c := range batch {
		if c == nil {
			continue
		}
		if msg, ok := c().(OpenFileMsg); ok {
			if msg.Path != file {
				t.Errorf("OpenFileMsg path = %q, want %q", msg.Path, file)
			}
			return
		}
	}
	t.Error("Init did not schedule an OpenFileMsg for the command-line file")
}

func TestWelcomeKeyStartsInsert(t *testing.T) {
	m := InitialModel(t.TempDir(), "", config.DefaultConfig())
	if m.focus != FocusEditor {
		t.Errorf("initial focus = %v, want FocusEditor", m.focus)
	}
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	got := m2.(Model)
	if got.showWelcome {
		t.Error("welcome screen still showing after a key")
	}
	if got.editor.mode != ModeInsert {
		t.Errorf("mode = %v, want ModeInsert after pressing i on the welcome screen", got.editor.mode)
	}
}

func TestToggleTreeMovesFocus(t *testing.T) {
	m := InitialModel(t.TempDir(), "", config.DefaultConfig())
	m.focus = FocusFileTree
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	got := m2.(Model)
	if got.showTree {
		t.Fatal("tree still shown after Ctrl+T")
	}
	if got.focus != FocusEditor {
		t.Errorf("focus = %v, want FocusEditor after hiding the focused tree", got.focus)
	}
}

func TestQuitWarningVisibleFromAgent(t *testing.T) {
	m := InitialModel(t.TempDir(), "", config.DefaultConfig())
	m.focus = FocusAgent
	m.editor.modified = true
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd != nil {
		t.Error("Ctrl+C with unsaved changes should not quit")
	}
	got := m2.(Model)
	if got.focus != FocusEditor {
		t.Errorf("focus = %v, want FocusEditor so the warning is visible", got.focus)
	}
	if !strings.Contains(got.editor.msg, "No write") {
		t.Errorf("msg = %q, want the no-write warning", got.editor.msg)
	}
}

func TestAgentHistoryAndFileContext(t *testing.T) {
	a := NewAgent(config.AI{}, config.Keys{})
	a.SetCurrentFile("f.go", "old")
	if file, content := a.fileContext(); file != "f.go" || content != "old" {
		t.Errorf("first context = (%q, %q), want (f.go, old)", file, content)
	}

	// Once the model has seen it, don't resend an unchanged file.
	a.sentFile, a.sentContent = "f.go", "old"
	if file, _ := a.fileContext(); file != "" {
		t.Errorf("unchanged file resent: %q", file)
	}

	// An edit makes the file context fresh again.
	a.SetCurrentFile("f.go", "new")
	if file, content := a.fileContext(); file != "f.go" || content != "new" {
		t.Errorf("after edit context = (%q, %q), want (f.go, new)", file, content)
	}

	// A different file starts a new conversation.
	a.history = []string{"User: hi", "Assistant: hello"}
	a.SetCurrentFile("g.go", "")
	if len(a.history) != 0 {
		t.Errorf("history = %v, want reset on file switch", a.history)
	}

	// The transcript stays bounded.
	for i := 0; i < maxHistoryEntries+10; i++ {
		a.history = append(a.history, "User: x")
	}
	a.trimHistory()
	if len(a.history) > maxHistoryEntries {
		t.Errorf("history has %d entries, want <= %d", len(a.history), maxHistoryEntries)
	}
}

func TestAgentRemembersTurns(t *testing.T) {
	a := NewAgent(config.AI{}, config.Keys{})
	a.currentFile, a.fileContent = "f.go", "code"
	a.pendingUser, a.pendingFile, a.pendingContent = "explain", "f.go", "code"

	a, _ = a.Update(AIResponseMsg{Response: "it is code"})
	if len(a.history) != 2 || a.history[0] != "User: explain" || a.history[1] != "Assistant: it is code" {
		t.Errorf("history = %v, want one remembered turn", a.history)
	}
	if a.sentFile != "f.go" || a.sentContent != "code" {
		t.Errorf("sent = (%q, %q), want the file marked as seen", a.sentFile, a.sentContent)
	}

	// A failed request is not remembered as a turn.
	a.history = nil
	a.pendingUser = "again"
	a, _ = a.Update(AIResponseMsg{Err: errors.New("boom")})
	if len(a.history) != 0 {
		t.Errorf("history after an error = %v, want empty", a.history)
	}
}

func TestFileType(t *testing.T) {
	cases := map[string]string{
		"":         "text",
		"Makefile": "text",
		"a.go":     "go",
		"a.js":     "javascript",
		"a.ts":     "typescript",
		"a.py":     "python",
		"a.rs":     "rust",
		"a.md":     "markdown",
		"a.yml":    "yaml",
		"a.toml":   "toml",
		"a.xyz":    "xyz",
		"a.GO":     "go",
	}
	for name, want := range cases {
		if got := fileType(name); got != want {
			t.Errorf("fileType(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestExpandHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	if got, want := expandHome("~/notes.md"), filepath.Join(home, "notes.md"); got != want {
		t.Errorf("expandHome(~/notes.md) = %q, want %q", got, want)
	}
	if got := expandHome("~"); got != home {
		t.Errorf("expandHome(~) = %q, want %q", got, home)
	}
	for _, path := range []string{"~other/x", "rel/x", "/abs/x"} {
		if got := expandHome(path); got != path {
			t.Errorf("expandHome(%q) = %q, want unchanged", path, got)
		}
	}
}

func TestAgentTypingDoesNotScroll(t *testing.T) {
	a := NewAgent(config.AI{}, config.Keys{})
	a.SetSize(40, 20)
	a.messages = []string{strings.Repeat("line\n", 100)}
	a.renderMessages()
	a.viewport.SetYOffset(0)

	a, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if a.viewport.YOffset != 0 {
		t.Errorf("typing j scrolled the chat to %d", a.viewport.YOffset)
	}

	a, _ = a.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	if a.viewport.YOffset == 0 {
		t.Error("pgdown did not scroll the chat")
	}
}

func TestUndoCursorStaysAtEdit(t *testing.T) {
	cfg := config.DefaultConfig()
	e := NewEditor(cfg.Commands, cfg.Keys)
	e.SetContent("aaa\nbbb\nccc", "f.txt")

	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyEsc})
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if got := e.textarea.Value(); got != "aaa\nbbb\nccc" {
		t.Fatalf("undo: got %q", got)
	}

	// Typing now must land at the edit (top), not at the end of the buffer.
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
	if got := e.textarea.Value(); !strings.HasPrefix(got, "Yaaa") {
		t.Errorf("after undo the cursor is not at the edit: %q", got)
	}
}
