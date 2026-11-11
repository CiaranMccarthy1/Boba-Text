# Boba Text

A modern, **Neovim-inspired** terminal code editor with a built-in **AI agent** (Gemini, OpenAI, Claude, or OpenRouter). Built with Go, Bubble Tea, and Lipgloss.

## Features

- **Neovim-Style Modal Editing** - Normal, Insert, Command, and Search modes with proper Vim keybindings
- **Vim Motions** - `h/j/k/l`, `w/b`, `0/$`, `G`, `o/O`, `A/I`, and more
- **Command Mode** - `:w`, `:q`, `:wq`, `:q!`, `:e <file>`, `:<line>` jump
- **Live Search** - `/` to search, `n`/`N` to navigate matches
- **File Tree with Icons** - Collapsible sidebar with filetype icons and sorted entries
- **AI Agent** - Chat with Google Gemini, OpenAI, Claude, or OpenRouter about your code and request refactors
- **Neovim-Style Status Line** - lualine-inspired bar showing mode, filename, modified state, and filetype
- **Welcome Screen** - ASCII art splash on startup (like alpha.nvim)
- **Configurable** - Full TOML config for colors, keys, AI, and commands

## Quick Start

**Install with Go:**

```sh
go install github.com/CiaranMccarthy1/boba-text@latest
boba-text
```

**Or build from source:**

```sh
go build -o boba-text .
./boba-text
```

Open a file directly with `./boba-text path/to/file`, or print the version with `./boba-text --version`. On Windows use `boba-text.exe`, or the batch shortcut:

```powershell
.\boba
```

## AI Agent Setup

The agent remembers the conversation and sends the current file only on the first message or after you edit it. Your prompt and that file content are sent to the selected provider.

Set the API key env var for your chosen provider:

```powershell
# gemini
$env:GEMINI_API_KEY = "your-api-key-here"
# openai
$env:OPENAI_API_KEY = "your-api-key-here"
# claude
$env:ANTHROPIC_API_KEY = "your-api-key-here"
# openrouter
$env:OPENROUTER_API_KEY = "your-api-key-here"
```

Then set `provider` and `model` in the `[ai]` section of your config (see Configuration below).

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
| `u` | Undo |
| `Ctrl+R` | Redo |

### Editor - Insert Mode

| Key | Action |
| :--- | :--- |
| `Esc` | Return to Normal Mode |
| *(any)* | Type into the buffer |

### Editor - Command Mode

| Command | Action |
| :--- | :--- |
| `:w` / `:s` | Save file |
| `:w <file>` | Save to a new file (also creates it) |
| `:q` | Quit |
| `:wq` / `:x` | Save and quit |
| `:q!` | Force quit (discard changes) |
| `:e <file>` | Open a file (creates it if missing) |
| `:e! <file>` | Open a file, discarding unsaved changes |
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
- **AI**: [Google Gemini](https://ai.google.dev/), [OpenAI](https://openai.com/), [Anthropic Claude](https://www.anthropic.com/), [OpenRouter](https://openrouter.ai/)

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
provider = "gemini"
name = "Gemini"
model = "gemini-2.0-flash"

[commands]
save = ["w", "s", "save"]
quit = ["q", "quit", "exit"]
```

## License

MIT
