package tui

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/CiaranMccarthy1/boba-text/pkg/config"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type EditorMode int

const (
	ModeNormal EditorMode = iota
	ModeInsert
	ModeCommand
	ModeSearch
)

type EditorModel struct {
	textarea    textarea.Model
	textinput   textinput.Model
	mode        EditorMode
	width       int
	height      int
	filename    string
	msg         string
	modified    bool
	commands    config.Commands
	keys        config.Keys
	yankBuffer  string
	searchQuery string
	searchInput textinput.Model
	undoStack   []string
	redoStack   []string
}

// maxUndoEntries/maxUndoBytes bound undo history so a 10 MB buffer can't
// balloon into a GB of snapshots.
const (
	maxUndoEntries = 100
	maxUndoBytes   = 64 << 20
)

// NewEditor creates a new editor model with the given command configuration.
func NewEditor(cmdConfig config.Commands, keyConfig config.Keys) EditorModel {
	ta := textarea.New()
	ta.Placeholder = "Empty file..."
	ta.Focus()
	ta.CharLimit = 0
	ta.ShowLineNumbers = true

	ti := textinput.New()
	ti.Prompt = ":"
	ti.Placeholder = ""
	ti.CharLimit = 156
	ti.Width = 50

	si := textinput.New()
	si.Prompt = "/"
	si.Placeholder = "search..."
	si.CharLimit = 156
	si.Width = 50

	return EditorModel{
		textarea:    ta,
		textinput:   ti,
		searchInput: si,
		mode:        ModeNormal,
		commands:    cmdConfig,
		keys:        keyConfig,
	}
}

func (m EditorModel) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, textinput.Blink)
}

func (m EditorModel) Update(msg tea.Msg) (EditorModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.mode {
		case ModeNormal:
			cmd = m.handleNormalMode(msg)
			if cmd != nil {
				return m, cmd
			}
		case ModeInsert:
			switch msg.String() {
			case m.keys.EditorNormalMode:
				m.mode = ModeNormal
				m.msg = ""
			default:
				m.textarea, cmd = m.textarea.Update(msg)
				cmds = append(cmds, cmd)
				m.modified = true
			}
		case ModeCommand:
			switch msg.String() {
			case m.keys.EditorNormalMode:
				m.mode = ModeNormal
				m.textinput.Blur()
				m.msg = ""
			case m.keys.EditorCommandRun:
				cmd = m.executeCommand(m.textinput.Value())
				if cmd != nil {
					return m, cmd
				}
			default:
				m.textinput, cmd = m.textinput.Update(msg)
				cmds = append(cmds, cmd)
			}
		case ModeSearch:
			switch msg.String() {
			case m.keys.EditorNormalMode:
				m.mode = ModeNormal
				m.searchInput.Blur()
				m.msg = ""
			case "enter":
				m.searchQuery = m.searchInput.Value()
				m.mode = ModeNormal
				m.searchInput.Blur()
				if m.searchQuery != "" {
					if m.findNextMatch(true) {
						m.msg = "/" + m.searchQuery
					} else {
						m.msg = "No match: /" + m.searchQuery
					}
				}
			default:
				m.searchInput, cmd = m.searchInput.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// handleNormalMode processes key presses in Normal mode (Vim motions).
func (m *EditorModel) handleNormalMode(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()
	switch key {
	// Enter insert mode
	case m.keys.EditorInsertMode:
		m.pushUndo()
		m.mode = ModeInsert
		m.textarea.Focus()
		m.msg = ""
	// Enter command mode
	case m.keys.EditorCommandMode:
		m.mode = ModeCommand
		m.textinput.Focus()
		m.textinput.SetValue("")
		m.msg = ""
	// Search mode
	case "/":
		m.mode = ModeSearch
		m.searchInput.Focus()
		m.searchInput.SetValue("")
		m.msg = ""
	// Cursor movement
	case "j", "down":
		m.textarea.CursorDown()
	case "k", "up":
		m.textarea.CursorUp()
	case "h", "left":
		li := m.textarea.LineInfo()
		col := li.StartColumn + li.ColumnOffset
		if col > 0 {
			m.textarea.SetCursor(col - 1)
		}
	case "l", "right":
		li := m.textarea.LineInfo()
		m.textarea.SetCursor(li.StartColumn + li.ColumnOffset + 1)
	// Word motions
	case "w":
		return m.wordForward()
	case "b":
		return m.wordBackward()
	// Line start/end
	case "0", "home":
		m.textarea.CursorStart()
	case "$", "end":
		m.textarea.CursorEnd()
	// Top/bottom of file
	case "G":
		m.moveToLine(m.textarea.LineCount() - 1)
		m.textarea.CursorStart()
	// Insert above/below
	case "o":
		m.pushUndo()
		m.textarea.CursorEnd()
		cmd := m.applyTextareaKey(tea.KeyMsg{Type: tea.KeyEnter})
		m.textarea.Focus()
		m.mode = ModeInsert
		m.modified = true
		return cmd
	case "O":
		m.pushUndo()
		m.textarea.CursorStart()
		cmd := m.applyTextareaKey(tea.KeyMsg{Type: tea.KeyEnter})
		m.moveToLine(m.textarea.Line() - 1)
		m.textarea.CursorStart()
		m.textarea.Focus()
		m.mode = ModeInsert
		m.modified = true
		return cmd
	// Insert at start/end of line
	case "A":
		m.pushUndo()
		m.textarea.CursorEnd()
		m.textarea.Focus()
		m.mode = ModeInsert
	case "I":
		m.pushUndo()
		m.textarea.CursorStart()
		m.textarea.Focus()
		m.mode = ModeInsert
	// Delete char under cursor
	case "x":
		m.pushUndo()
		cmd := m.applyTextareaKey(tea.KeyMsg{Type: tea.KeyDelete})
		m.modified = true
		return cmd
	// Search next/prev
	case "n":
		if m.searchQuery != "" {
			if m.findNextMatch(true) {
				m.msg = "/" + m.searchQuery
			} else {
				m.msg = "No match: /" + m.searchQuery
			}
		}
	case "N":
		if m.searchQuery != "" {
			if m.findNextMatch(false) {
				m.msg = "/" + m.searchQuery
			} else {
				m.msg = "No match: /" + m.searchQuery
			}
		}
	// Yank current line
	case "y":
		m.yankCurrentLine()
	// Paste
	case "p":
		if m.yankBuffer != "" {
			m.pushUndo()
			m.textarea.Focus()
			m.textarea.InsertString(m.yankBuffer)
			m.modified = true
			m.msg = "Pasted"
		}
	// Undo / redo
	case "u":
		m.undo()
	case "ctrl+r":
		m.redo()
	}
	return nil
}

// executeCommand processes a command-mode command string.
func (m *EditorModel) executeCommand(val string) tea.Cmd {
	m.mode = ModeNormal
	m.textinput.Blur()

	// Handle line number jump :<number>
	if len(val) > 0 && val[0] >= '0' && val[0] <= '9' {
		lineNum, err := strconv.Atoi(val)
		if err != nil {
			m.msg = "Invalid line: " + val
			return nil
		}
		m.jumpToLine(lineNum)
		m.msg = fmt.Sprintf("Jump to line %d", lineNum)
		return nil
	}

	switch {
	case val == "wq" || val == "x":
		m.saveFile()
		if m.modified {
			return nil // save failed; stay in the editor so work isn't lost
		}
		return tea.Quit
	case strings.HasPrefix(val, "w "):
		if path := strings.TrimSpace(strings.TrimPrefix(val, "w ")); path != "" {
			m.filename = path
			return m.saveFile()
		}
	case slices.Contains(m.commands.Save, val):
		return m.saveFile()
	case slices.Contains(m.commands.Quit, val):
		if m.modified {
			m.msg = "No write since last change (use :q!)"
			return nil
		}
		return tea.Quit
	case val == "q!":
		return tea.Quit
	case strings.HasPrefix(val, "e "):
		path := strings.TrimPrefix(val, "e ")
		path = strings.TrimSpace(path)
		if path != "" {
			return func() tea.Msg { return OpenFileMsg{Path: path} }
		}
		m.msg = "Usage: :e <filename>"
	default:
		m.msg = "Unknown command: " + val
	}
	return nil
}

// saveFile writes the editor content to disk.
func (m *EditorModel) saveFile() tea.Cmd {
	if m.filename != "" {
		err := os.WriteFile(m.filename, []byte(m.textarea.Value()), 0644)
		if err != nil {
			m.msg = "Error saving: " + err.Error()
		} else {
			m.msg = "Saved: " + m.filename
			m.modified = false
		}
	} else {
		m.msg = "No filename set!"
	}
	return nil
}

// applyTextareaKey sends a key message through the textarea to reuse its motion logic.
func (m *EditorModel) applyTextareaKey(msg tea.KeyMsg) tea.Cmd {
	ta, cmd := m.textarea.Update(msg)
	m.textarea = ta
	return cmd
}

// wordForward moves the cursor forward by one word.
func (m *EditorModel) wordForward() tea.Cmd {
	return m.applyTextareaKey(tea.KeyMsg{Type: tea.KeyRight, Alt: true})
}

// wordBackward moves the cursor backward by one word.
func (m *EditorModel) wordBackward() tea.Cmd {
	return m.applyTextareaKey(tea.KeyMsg{Type: tea.KeyLeft, Alt: true})
}

func (m *EditorModel) currentCursor() (int, int) {
	row := m.textarea.Line()
	li := m.textarea.LineInfo()
	col := li.StartColumn + li.ColumnOffset
	if col < 0 {
		col = 0
	}
	return row, col
}

func (m *EditorModel) moveToLine(target int) {
	maxLine := m.textarea.LineCount() - 1
	if maxLine < 0 {
		return
	}
	if target < 0 {
		target = 0
	} else if target > maxLine {
		target = maxLine
	}
	for m.textarea.Line() < target {
		m.textarea.CursorDown()
	}
	for m.textarea.Line() > target {
		m.textarea.CursorUp()
	}
}

func (m *EditorModel) moveCursorTo(row, col int) {
	m.moveToLine(row)
	m.textarea.SetCursor(col)
}

func (m *EditorModel) jumpToLine(lineNum int) {
	if lineNum < 1 {
		lineNum = 1
	}
	m.moveToLine(lineNum - 1)
	m.textarea.CursorStart()
}

func (m *EditorModel) yankCurrentLine() {
	lines := strings.Split(m.textarea.Value(), "\n")
	row := m.textarea.Line()
	if row < 0 || row >= len(lines) {
		return
	}
	m.yankBuffer = lines[row]
	m.msg = "Yanked line"
}

func (m *EditorModel) findNextMatch(forward bool) bool {
	if m.searchQuery == "" {
		return false
	}
	lines := strings.Split(m.textarea.Value(), "\n")
	if len(lines) == 0 {
		return false
	}
	row, col := m.currentCursor()
	if forward {
		if matchRow, matchCol, ok := findMatchForward(lines, m.searchQuery, row, col+1); ok {
			m.moveCursorTo(matchRow, matchCol)
			return true
		}
		if matchRow, matchCol, ok := findMatchForward(lines, m.searchQuery, 0, 0); ok {
			m.moveCursorTo(matchRow, matchCol)
			return true
		}
		return false
	}

	startCol := col - 1
	if matchRow, matchCol, ok := findMatchBackward(lines, m.searchQuery, row, startCol); ok {
		m.moveCursorTo(matchRow, matchCol)
		return true
	}
	lastRow := len(lines) - 1
	lastCol := utf8.RuneCountInString(lines[lastRow]) - 1
	if matchRow, matchCol, ok := findMatchBackward(lines, m.searchQuery, lastRow, lastCol); ok {
		m.moveCursorTo(matchRow, matchCol)
		return true
	}
	return false
}

func findMatchForward(lines []string, query string, startRow int, startCol int) (int, int, bool) {
	if startRow < 0 || startRow >= len(lines) {
		return 0, 0, false
	}
	for row := startRow; row < len(lines); row++ {
		line := lines[row]
		startByte := 0
		if row == startRow {
			startByte = runeIndexToByteIndex(line, startCol)
		}
		idx := strings.Index(line[startByte:], query)
		if idx >= 0 {
			matchByte := startByte + idx
			matchCol := byteIndexToRuneIndex(line, matchByte)
			return row, matchCol, true
		}
	}
	return 0, 0, false
}

func findMatchBackward(lines []string, query string, startRow int, startCol int) (int, int, bool) {
	if startRow < 0 || startRow >= len(lines) {
		return 0, 0, false
	}
	for row := startRow; row >= 0; row-- {
		line := lines[row]
		endByte := len(line)
		if row == startRow {
			endByte = runeIndexToByteIndex(line, startCol+1)
		}
		if endByte < 0 {
			endByte = 0
		}
		idx := strings.LastIndex(line[:endByte], query)
		if idx >= 0 {
			matchCol := byteIndexToRuneIndex(line, idx)
			return row, matchCol, true
		}
	}
	return 0, 0, false
}

func runeIndexToByteIndex(s string, runeIndex int) int {
	if runeIndex <= 0 {
		return 0
	}
	if runeIndex >= utf8.RuneCountInString(s) {
		return len(s)
	}
	idx := 0
	for i := 0; i < runeIndex && idx < len(s); i++ {
		_, size := utf8.DecodeRuneInString(s[idx:])
		idx += size
	}
	return idx
}

func byteIndexToRuneIndex(s string, byteIndex int) int {
	if byteIndex <= 0 {
		return 0
	}
	if byteIndex >= len(s) {
		return utf8.RuneCountInString(s)
	}
	return utf8.RuneCountInString(s[:byteIndex])
}

// View renders the editor as a string.
func (m EditorModel) View() string {
	// Build the status line (Neovim lualine-style)
	var barContent string
	var modeStyle lipgloss.Style

	if m.mode == ModeCommand {
		barContent = m.textinput.View()
		modeStyle = StyleModeCommand
	} else if m.mode == ModeSearch {
		barContent = m.searchInput.View()
		modeStyle = StyleModeCommand
	} else {
		modeTxt := modeName(m.mode)
		switch m.mode {
		case ModeInsert:
			modeStyle = StyleModeInsert
		default:
			modeStyle = StyleModeNormal
		}

		// File info section
		fname := m.filename
		if fname == "" {
			fname = "[No Name]"
		}
		modifiedMark := ""
		if m.modified {
			modifiedMark = " [+]"
		}

		msgInfo := ""
		if m.msg != "" {
			msgInfo = "  " + m.msg
		}

		// Lualine-style: MODE | filename [+] | message
		statusRight := StyleDim.Render(fmt.Sprintf(" %s ", fileType(m.filename)))
		barContent = fmt.Sprintf("%s %s%s%s%s",
			modeStyle.Render(" "+modeTxt+" "),
			fname, modifiedMark, msgInfo, statusRight)
		// Return early with full status bar
		statusBar := lipgloss.NewStyle().
			Foreground(ColorText).
			Background(lipgloss.Color("#2A2A2A")).
			Width(m.width).
			Padding(0, 0).
			Render(barContent)

		return StyleEditor.
			Width(m.width).
			Height(m.height).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					m.textarea.View(),
					statusBar,
				),
			)
	}

	// Command/Search mode status bar
	statusBar := lipgloss.NewStyle().
		Foreground(ColorText).
		Background(lipgloss.Color("#2A2A2A")).
		Width(m.width).
		Padding(0, 0).
		Render(barContent)

	return StyleEditor.
		Width(m.width).
		Height(m.height).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				m.textarea.View(),
				statusBar,
			),
		)
}

// modeName returns the display name for the given editor mode.
func modeName(m EditorMode) string {
	switch m {
	case ModeInsert:
		return "INSERT"
	case ModeCommand:
		return "COMMAND"
	case ModeSearch:
		return "SEARCH"
	default:
		return "NORMAL"
	}
}

// fileType returns a short filetype label based on extension.
func fileType(filename string) string {
	if filename == "" {
		return "text"
	}
	parts := strings.Split(filename, ".")
	if len(parts) < 2 {
		return "text"
	}
	ext := parts[len(parts)-1]
	switch ext {
	case "go":
		return "go"
	case "js":
		return "javascript"
	case "ts":
		return "typescript"
	case "py":
		return "python"
	case "rs":
		return "rust"
	case "md":
		return "markdown"
	case "json":
		return "json"
	case "toml":
		return "toml"
	case "yaml", "yml":
		return "yaml"
	case "html":
		return "html"
	case "css":
		return "css"
	default:
		return ext
	}
}

// SetContent sets the editor content and filename.
func (m *EditorModel) SetContent(content string, filename string) {
	m.textarea.SetValue(content)
	// SetValue leaves the cursor at the end of the inserted text; open at the top.
	m.textarea.CursorStart()
	m.filename = filename
	m.modified = false
	m.msg = ""
	// New file, fresh history.
	m.undoStack = nil
	m.redoStack = nil
}

// pushUndo snapshots the buffer before a change so `u` can restore it.
func (m *EditorModel) pushUndo() {
	m.undoStack = append(m.undoStack, m.textarea.Value())
	m.redoStack = nil
	// Drop oldest snapshots past the entry and memory caps.
	total := 0
	for _, s := range m.undoStack {
		total += len(s)
	}
	for (len(m.undoStack) > maxUndoEntries || total > maxUndoBytes) && len(m.undoStack) > 1 {
		total -= len(m.undoStack[0])
		m.undoStack = m.undoStack[1:]
	}
}

// undo restores the last snapshot; redo re-applies it.
func (m *EditorModel) undo() {
	if len(m.undoStack) == 0 {
		m.msg = "Nothing to undo"
		return
	}
	m.redoStack = append(m.redoStack, m.textarea.Value())
	m.textarea.SetValue(m.undoStack[len(m.undoStack)-1])
	m.undoStack = m.undoStack[:len(m.undoStack)-1]
	// shortcut: undo always marks the buffer modified, even if you undo back
	// to the saved state; upgrade by diffing against the file on disk.
	m.modified = true
	m.msg = "Undone"
}

func (m *EditorModel) redo() {
	if len(m.redoStack) == 0 {
		m.msg = "Nothing to redo"
		return
	}
	m.undoStack = append(m.undoStack, m.textarea.Value())
	m.textarea.SetValue(m.redoStack[len(m.redoStack)-1])
	m.redoStack = m.redoStack[:len(m.redoStack)-1]
	m.modified = true
	m.msg = "Redone"
}
