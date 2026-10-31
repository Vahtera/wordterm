# Wordterm

Distraction-free terminal writing app for chapter-based projects (Go, Bubble Tea v2, Lip Gloss).

## Build

    go get charm.land/bubbletea/v2@latest charm.land/lipgloss/v2@latest
    go get github.com/charmbracelet/x/ansi@latest
    go mod tidy
    go test ./...
    go run .

Cross-compile: `GOOS=windows GOARCH=arm64 go build -o wordterm.exe .` (same pattern for linux/darwin, amd64/arm64).

## Status

- Phase 1 (foundation): two-pane frame, resize handling, minimum size screen. Written but not yet compiled by the author; see notes in the chat.
- Phases 2-7: not started.

Keys now: F3 switch pane, Ctrl+Q / Ctrl+C quit. Any key quits from the too-small screen.
