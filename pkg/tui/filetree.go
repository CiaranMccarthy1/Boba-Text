package tui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/CiaranMccarthy1/boba-text/pkg/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// parentDirEntry implements os.DirEntry for the ".." parent directory entry.
type parentDirEntry struct{}

// Name returns ".." to represent the parent directory.
func (e parentDirEntry) Name() string { return ".." }

// IsDir returns true since parent directory entries are always directories.
func (e parentDirEntry) IsDir() bool { return true }

// Type returns fs.ModeDir indicating a directory type.
func (e parentDirEntry) Type() fs.FileMode { return fs.ModeDir }

// Info returns nil as parent directory entries don't need file info.
func (e parentDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

type FileTreeModel struct {
	path     string
	files    []os.DirEntry
	cursor   int
	selected string
	width    int
	height   int
	msg      string
	keys     config.Keys
}

// NewFileTree creates a new file tree model starting at the given path.
func NewFileTree(startPath string, keyConfig config.Keys) FileTreeModel {
	m := FileTreeModel{
		path: startPath,
		keys: keyConfig,
	}
	m.loadFiles()
	return m
}

// loadFiles loads directory entries from the current path and adds parent entry if not at root.
// Sorts directories first, then files alphabetically.
func (m *FileTreeModel) loadFiles() {
	entries, err := os.ReadDir(m.path)
	if err != nil {
		m.files = nil
		m.msg = err.Error()
		return
	}
	m.msg = ""

	// Sort: directories first, then files, alphabetical within each group
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	parent := filepath.Dir(m.path)
	if parent != m.path {
		newFiles := make([]os.DirEntry, 0, len(entries)+1)
		newFiles = append(newFiles, parentDirEntry{})
		newFiles = append(newFiles, entries...)
		m.files = newFiles
	} else {
		m.files = entries
	}
}

func (m FileTreeModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the file tree state.
func (m FileTreeModel) Update(msg tea.Msg) (FileTreeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case m.keys.TreeUp, m.keys.TreeUpAlt:
			if m.cursor > 0 {
				m.cursor--
			}
		case m.keys.TreeDown, m.keys.TreeDownAlt:
			if m.cursor < len(m.files)-1 {
				m.cursor++
			}
		case m.keys.TreeOpen:
			if len(m.files) == 0 {
				break
			}
			selected := m.files[m.cursor]

			if selected.Name() == ".." {
				m.path = filepath.Dir(m.path)
				m.loadFiles()
				m.cursor = 0
				return m, nil
			}

			newPath := filepath.Join(m.path, selected.Name())
			if selected.IsDir() {
				m.path = newPath
				m.loadFiles()
				m.cursor = 0
			} else {
				m.selected = newPath
				return m, func() tea.Msg {
					return OpenFileMsg{Path: newPath}
				}
			}
		case m.keys.TreeBack, m.keys.TreeBackAlt, m.keys.TreeBackAlt2:
			parent := filepath.Dir(m.path)
			if parent != m.path {
				m.path = parent
				m.loadFiles()
				m.cursor = 0
			}
		}
	}
	return m, nil
}

// fileIcon returns an icon for the file based on its extension.
func fileIcon(name string, isDir bool) string {
	if isDir {
		if name == ".." {
			return "⮤"
		}
		return "📁"
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".go":
		return "🐹"
	case ".js", ".jsx":
		return "📜"
	case ".ts", ".tsx":
		return "📘"
	case ".py":
		return "🐍"
	case ".rs":
		return "🦀"
	case ".md":
		return "📝"
	case ".json":
		return "📋"
	case ".toml", ".yaml", ".yml":
		return "⚙️"
	case ".html":
		return "🌐"
	case ".css":
		return "🎨"
	case ".sh", ".bat", ".ps1":
		return "🐚"
	case ".git", ".gitignore":
		return "🙈"
	case ".jpg", ".jpeg", ".png", ".gif", ".svg":
		return "🖼️"
	case ".mod":
		return "📦"
	default:
		return "📄"
	}
}

// View renders the file tree as a string.
func (m FileTreeModel) View() string {
	var s strings.Builder

	// Shortened path display
	dirName := filepath.Base(m.path)
	s.WriteString(StyleBold.Render("  "+dirName) + "\n")
	sepWidth := m.width - 4
	if sepWidth < 1 {
		sepWidth = 1
	}
	s.WriteString(StyleDim.Render(" "+strings.Repeat("─", sepWidth)) + "\n")

	if m.msg != "" {
		s.WriteString(StyleDim.Render(" "+m.msg) + "\n")
	}

	visible := m.height - 4
	if visible < 1 {
		visible = 1
	}

	// Scrolling window
	start := 0
	if m.cursor >= visible {
		start = m.cursor - visible + 1
	}
	end := start + visible
	if end > len(m.files) {
		end = len(m.files)
	}

	for i := start; i < end; i++ {
		f := m.files[i]
		icon := fileIcon(f.Name(), f.IsDir())
		name := f.Name()
		if f.IsDir() && name != ".." {
			name += "/"
		}

		if m.cursor == i {
			line := fmt.Sprintf(" ➜ %s %s", icon, name)
			s.WriteString(StyleSelected.Render(line) + "\n")
		} else {
			var iconStyle lipgloss.Style
			if f.IsDir() {
				iconStyle = StyleDirIcon
			} else {
				iconStyle = StyleFileIcon
			}
			line := fmt.Sprintf("   %s %s", iconStyle.Render(icon), StyleDim.Render(name))
			s.WriteString(line + "\n")
		}
	}

	return StyleFileTree.
		Width(m.width).
		Height(m.height).
		Render(s.String())
}

type OpenFileMsg struct {
	Path string
}
