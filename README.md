# Wordterm

Distraction-free terminal writing app for chapter-based projects (Go, Bubble Tea v2, Lip Gloss).

## Build and run

    go get charm.land/bubbletea/v2@latest charm.land/lipgloss/v2@latest
    go get github.com/charmbracelet/x/ansi@latest github.com/rivo/uniseg@latest
    go mod tidy
    go test ./...
    go run . new ~/Documents Example "Your Name"   # create a project and open it
    go run . ~/Documents/Example                   # open an existing project

Cross-compile: `GOOS=windows GOARCH=arm64 go build -o wordterm.exe .` (same pattern for linux/darwin, amd64/arm64).

## Working-copy rule

The project is copied to a temp working copy (`<os temp>/wordterm/session-*/project`). All app
reads and writes use that copy. The original is touched only by Save (F10).
After a crash the copy is kept; on next launch you are asked to recover or discard it.

## Status

- Phase 1 (foundation): done.
- Phase 2 (project management): done.
- Phase 3 (editor): grapheme-aware text buffer, word wrap, scrolling, undo/redo, paste. Focus the
  editor with F3 and type. Edits live in memory and are written to the working copy every 2 s, on
  F10/F9 and on quit; the original changes only on F10. Navigator is still a read-only tree
  (Phase 4). Keys: arrows, Home/End, PgUp/PgDn, Backspace/Delete, Ctrl+Z undo, Ctrl+Y redo,
  F3 pane, F9 reload, F10 save, Ctrl+Q quit. Quitting with unsaved edits keeps the working copy
  and offers recovery on next start (quit prompt arrives in Phase 5).
- Phase 4 (navigator): Up/Down/Left/Right navigate the tree (Navigator focused), Enter opens a scene or
  toggles a chapter, `+` add, `-` delete (chapter must be empty), F2 rename, F5/F6 move (scenes cross
  chapter edges and the file moves too). Selection is drawn in inverted colours. Questions and name
  prompts appear in the status bar until the dialog system arrives in Phase 5. F9/F10 and quit now ask
  for confirmation. Undo history is per open scene (reset when you switch scenes).
- Phases 5-7: not started.
