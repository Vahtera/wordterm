# Wordterm

Distraction-free terminal writing app for chapter-based projects (Go, Bubble Tea v2, Lip Gloss).

## Build and run

    go get charm.land/bubbletea/v2@latest charm.land/lipgloss/v2@latest
    go get github.com/charmbracelet/x/ansi@latest
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
- Phase 2 (project management): project JSON + validation, create/load, working copy, save, reload,
  crash recovery, last-opened scene, word counts. Navigator is a read-only tree and the editor a
  read-only preview until Phases 3-4. Keys: F3 pane, F9 reload, F10 save (no confirmation dialogs
  until Phase 5), Ctrl+Q quit.
- Phases 3-7: not started.
