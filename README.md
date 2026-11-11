# Boba Text

A modern, **Neovim-inspired** terminal code editor with a built-in **Gemini AI agent**. Built with Go, Bubble Tea, and Lipgloss.

## Features

- **Neovim-Style Modal Editing** - Normal, Insert, Command, and Search modes with proper Vim keybindings
- **Vim Motions** - `h/j/k/l`, `w/b`, `0/$`, `G`, `o/O`, `A/I`, and more
- **Command Mode** - `:w`, `:q`, `:wq`, `:q!`, `:e <file>`, `:<line>` jump
- **Live Search** - `/` to search, `n`/`N` to navigate matches
- **File Tree with Icons** - Collapsible sidebar with filetype icons and sorted entries
- **Gemini AI Agent** - Chat with Google Gemini about your code and request refactors
- **Neovim-Style Status Line** - lualine-inspired bar showing mode, filename, modified state, and filetype
- **Welcome Screen** - ASCII art splash on startup (like alpha.nvim)
- **Configurable** - Full TOML config for colors, keys, AI, and commands
- **Instant Startup** - Pre-compiled binary for zero-latency launch

## Quick Start

**Build & Run:**

```powershell
go build -o boba-text.exe .
./boba-text.exe
```

**Or use the batch shortcut (Windows):**

```powershell
.\boba
```

## AI Agent Setup - In progress

Set your Gemini API key to enable the built-in AI assistant:

```powershell
$env:GEMINI_API_KEY = "your-api-key-here"
```

## Keybindings

### Global

| Key | Action |
| :--- | :--- |
| `Ctrl+T` | Toggle File Tree sidebar |
| `Ctrl+E` | Focus / Toggle File Tree |
| `Ctrl+A` | Focus AI Agent |
| `Tab` | Cycle focus (Tree → Editor → Agent) |
| `Ctrl+C` | Quit |

### Editor - Normal Mode

| Key | Action |
| :--- | :--- |
| `h/j/k/l` | Move cursor Left / Down / Up / Right |
| `w` / `b` | Word forward / backward |
| `0` / `$` | Line start / end |
| `G` | Go to end of file |
| `i` | Enter **Insert Mode** |
| `I` / `A` | Insert at line start / end |
| `o` / `O` | Open line below / above |
| `/` | Enter **Search Mode** |
| `:` | Enter **Command Mode** |
| `p` | Paste yanked text |
| `x` | Delete character |

### Editor - Insert Mode

| Key | Action |
| :--- | :--- |
| `Esc` | Return to Normal Mode |
| *(any)* | Type into the buffer |

### Editor - Command Mode

| Command | Action |
| :--- | :--- |
| `:w` / `:s` | Save file |
| `:q` | Quit |
| `:wq` / `:x` | Save and quit |
| `:q!` | Force quit (discard changes) |
| `:e <file>` | Open a file |
| `:<number>` | Jump to line number |

### Editor - Search Mode

| Key | Action |
| :--- | :--- |
| `Enter` | Execute search |
| `n` / `N` | Next / Previous match (Normal Mode) |
| `Esc` | Cancel search |

### File Tree

| Key | Action |
| :--- | :--- |
| `j` / `k` | Navigate Down / Up |
| `Enter` | Open file / Enter directory |
| `h` / `Backspace` | Go to parent directory |

### AI Agent

| Key | Action |
| :--- | :--- |
| `Enter` | Send message to Gemini |

## Stack

- **Language**: [Go](https://go.dev/)
- **TUI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **Styling**: [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **Components**: [Bubbles](https://github.com/charmbracelet/bubbles)
- **AI**: [Google Gemini API](https://ai.google.dev/)

## Configuration

Boba Text comes with sensible defaults built-in. If you want to override them globally, you can create a `~/.config/boba-text/config.toml` (or `%APPDATA%\boba-text\config.toml` on Windows).

Here is an example overriding colors, keys, and AI configuration:

```toml
[colors]
primary = "#FF00FF"
text = "#FFFFFF"

[keys]
toggle_tree = "ctrl+t"
focus_tree = "ctrl+e"
focus_agent = "ctrl+a"
cycle_focus = "tab"
quit = "ctrl+c"
save = "ctrl+s"

tree_up = "up"
tree_up_alt = "k"
tree_down = "down"
tree_down_alt = "j"
tree_open = "enter"
tree_back = "backspace"
tree_back_alt = "h"
tree_back_alt_2 = "left"

editor_insert_mode = "i"
editor_command_mode = ":"
editor_normal_mode = "esc"
editor_command_run = "enter"

agent_send = "enter"

[ai]
name = "Gemini"
model = "gemini-2.0-flash"

[commands]
save = ["w", "s", "save"]
quit = ["q", "quit", "exit"]
```

## License

MIT
