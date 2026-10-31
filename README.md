# Wordterm

A distraction-free terminal writing app for chapter-based projects such as novels.
Plain-text files, no lock-in, one standalone executable per platform.

Written in Go with Bubble Tea v2, Lip Gloss and Bubbles-era Charm libraries.

![wordterm default window](https://github.com/Vahtera/wordterm/default.png?raw=true)

## Install / build

You need Go (the version in `go.mod`).

    go install github.com/Vahtera/wordterm@latest     # installs the `wordterm` command

(Installed this way the About dialog shows the development version; use `build.sh` with `VERSION=...` for release builds.)

From a checkout:

    go mod tidy                 # first time: fetches dependencies and writes go.sum
    go run .                    # try it
    ./build.sh                  # vet + test + build all targets into dist/
    ./build.sh --native         # only this machine
    .\build.ps1                 # same as build.sh, for PowerShell

`dist/` then holds `wordterm-{linux,windows,darwin}-{amd64,arm64}[.exe]` and `SHA256SUMS`.
Copy the file for your system anywhere on your PATH. No other files are needed.
`VERSION=1.2.3 ./build.sh` stamps the version shown in Help > About.

Dependencies: bubbletea v2, lipgloss v2, charmbracelet/x/ansi, rivo/uniseg, BurntSushi/toml.
If you start from a fresh checkout without `go.mod`: `go mod init github.com/Vahtera/wordterm`, then
`go get charm.land/bubbletea/v2 charm.land/lipgloss/v2 github.com/charmbracelet/x/ansi github.com/rivo/uniseg github.com/BurntSushi/toml` and `go mod tidy`.

## Usage

    wordterm                                    built-in read-only help (chapter Help, scene Keys)
    wordterm <project-dir>                      open a project
    wordterm <name>                             open a project by name: Default Save Path first, then ./<name>
    wordterm new <name> [-path DIR] [-author NAME]
                                                create a project and open it (DIR and author default to Settings)

Without arguments you get a read-only copy of the key list. Press Esc > File > New Project to start writing.

### Keys (defaults; all except Esc can be rebound in Settings > Rebind Keys)

| Key | Action |
|---|---|
| Esc | Main menu (fixed) |
| F1 | Help (current bindings) |
| F3 | Switch pane (Navigator / Editor) |
| F11 | Screenwriter Mode on/off |
| F10 / F9 | Save / Reload project (both ask first) |
| Ctrl+Q, Ctrl+C | Quit (asks Save / Discard / Cancel if there are unsaved changes) |
| Ctrl+Z / Ctrl+Y | Undo / redo (per open scene) |
| Navigator: arrows, Enter | Move, collapse/expand, open scene |
| Navigator: `+` `-` | Add / delete item (a chapter must be empty before it can be deleted) |
| Navigator: F2 | Rename |
| Navigator: F5 / F6 | Move up / down (scenes cross chapter borders; the file moves too) |

The default pane switch is F3 rather than Ctrl+Tab because most terminals cannot report Ctrl+Tab.

### Menus

File: New Project..., Open Project..., Save, Save As..., Quit.
Settings: Editor Size, Write Size, Screenwriter Mode, Rebind Keys, Word Wrap, Autosave, Default Save Path..., Author Name...
Help: About..., Keys.

## Projects

    Example/
    ├── Example.json            metadata: names, order, last opened scene, format version
    ├── Chapter1/
    │   ├── Scene1.txt          UTF-8 plain text
    │   └── Scene2.txt
    └── Chapter2/
        └── Scene1_2.txt        stored under a different file name if a move would clash

![wordterm Example Project](https://github.com/Vahtera/wordterm/example_project.png?raw=true)

The project folder must contain exactly one `.json` file. Chapter and scene order is stored in the JSON.
A scene is a plain string in the JSON, or `{"name": "Scene1", "file": "Scene1_2"}` when its file name had
to differ from its name. Line endings are saved as LF.

### Working copy and saving

Opening a project copies it to a temporary working copy (`<temp dir>/wordterm/session-*`).
Everything you do happens in that copy. **The original is only written by Save, Save As and Autosave.**
Switching scenes or chapters never discards edits. If the original was changed outside Wordterm since you
opened it, Save asks before overwriting it, and autosave pauses instead of overwriting.

If Wordterm is killed or crashes, the working copy stays. The next start offers to **R**ecover,
**D**iscard or **S**kip each leftover copy. Edits are written to the working copy every 2 seconds, so at most
the last couple of seconds of typing can be lost. After recovering, the first Save asks once before overwriting
the original.

### Screenwriter Mode

Shows only the writing column (width = Write Size), centred, with the cursor line kept in the vertical middle.
Prompts and messages appear on the bottom row only when needed.

![wordterm with Screenwriter Mode enabled.](https://github.com/Vahtera/wordterm/screenwriter_mode.png?raw=true)

## Settings file

Created on first change; plain TOML. Location: `~/.config/wordterm/config.toml` (Linux),
`~/Library/Application Support/wordterm/config.toml` (macOS), `%AppData%\wordterm\config.toml` (Windows).

    editor_width = 0            # 0 = default (70% of the window)
    write_size = 0              # 0 = same as the editor width
    word_wrap = true
    autosave = false            # off by default
    autosave_interval = 60      # seconds, minimum 5
    default_save_path = ""
    author_name = ""

    [keys]
    save = "f10"                # one entry per action; see Settings > Rebind Keys

A config file that cannot be parsed is renamed to `config.toml.broken` and defaults are used.

## Known limitations

- Undo history is kept per open scene and is reset when you switch scenes.
- No selection, cut or copy inside the editor (paste from the terminal works).
- Text widths follow Unicode rules; a few terminals disagree on emoji or ambiguous-width characters, which
  can shift the cursor on those lines.
- Open Project cannot switch between drive letters on Windows.
- Autosave interval is set in the config file, not in a menu.
- Developed and tested on Fedora (KDE) with Kitty. Other systems and terminals follow the checklist in TESTING.md.

## Development

    go vet ./... && go test ./...

Layout: `internal/app` (model, menus, dialogs), `editor` (text buffer), `navigator` (tree), `project`
(format and operations), `storage` (working copy, atomic writes, recovery), `settings` (TOML, key bindings),
`ui` (layout and drawing helpers). Rule: the app only ever touches the working copy; only `storage.Session`
writes to the original.

## License

GPLv3, see LICENSE.
