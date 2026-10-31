# Cross-platform test checklist

Automated: `go vet ./... && go test ./...` (run by CI on Linux x86-64/ARM64, macOS ARM64, Windows x86-64;
`./build.sh` cross-compiles all six targets). The items below need a person at a real terminal.
Record the result per system/terminal; "tested by the author" so far: Fedora 44 KDE, Kitty.

## Per system (Linux, Windows, macOS; x86-64 and ARM64 where you have the hardware)
- [ ] `wordterm` starts, shows the read-only help project; typing offers "create a new project".
- [ ] New Project creates the folder and opens it; the original folder is untouched until Save.
- [ ] Type in two scenes, switch back and forth: nothing is lost; Save writes both; reopen shows them.
- [ ] Quit with unsaved changes: Save / Discard / Cancel each behave correctly.
- [ ] Kill the process (`kill -9`, or close the terminal window / Task Manager): restart offers recovery; Recover restores the text.
- [ ] Resize the window to the minimum: "Terminal Size too Small" appears, any key quits, terminal is restored.
- [ ] Rename, move (including across chapters), add, delete chapters and scenes; check the files on disk match.
- [ ] Settings persist after restart (editor width, write size, word wrap, autosave, key rebinding).
- [ ] Open Project browses folders and opens a project; Save As creates a complete copy.
- [ ] F11 Screenwriter Mode: cursor line stays centred; leaving restores the layout.
- [ ] The exit leaves the terminal normal (cursor visible, no stray colours or alt-screen).

## Per terminal emulator
Suggested: Kitty, GNOME Terminal / Konsole, Alacritty, WezTerm, Windows Terminal (PowerShell and cmd), iTerm2, Terminal.app, tmux inside any of them.
- [ ] Box-drawing characters and `▾ ▸ ◂ █` render; selected row is clearly inverted.
- [ ] F1-F11 reach the program (some terminals or desktops use F11 for fullscreen; if so rebind it or use the menu).
- [ ] Paste of several lines works (bracketed paste).
- [ ] Alt/Ctrl combinations used for bindings are not intercepted.

## Unicode and layouts
- [ ] Type and edit: å ä ö, é with combining accent, CJK (日本語), an emoji, right-to-left text (display order is not reordered; note any problems).
- [ ] Cursor movement and Backspace treat each of these as one character; wrapping does not split them.
- [ ] Non-US keyboard layout: letters, AltGr characters, and the `+` / `-` navigator keys work (rebind if your layout makes them awkward).
- [ ] Word counts match a reference count for text in several languages.
