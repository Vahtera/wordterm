package app

import (
	"fmt"
	"strings"

	"github.com/Vahtera/wordterm/internal/settings"
	"github.com/Vahtera/wordterm/internal/ui"
)

// Version is set at build time: -ldflags "-X github.com/Vahtera/wordterm/internal/app.Version=1.2.3"
var Version = "1.0.0-dev"

// helpLines is the key list shown by F1 and by the built-in help project.
func (m Model) helpLines() []string {
	lines := []string{"Wordterm keys", "", fmt.Sprintf("%-10s %s", "Esc", "Main menu (fixed)")}
	for _, a := range settings.Actions {
		lines = append(lines, fmt.Sprintf("%-10s %s", m.cfg.Keys[a.ID], a.Label))
	}
	return append(lines, "",
		fmt.Sprintf("%-10s %s", "Arrows", "Move cursor / navigate tree"),
		fmt.Sprintf("%-10s %s", "Enter", "Open scene / toggle chapter"),
		fmt.Sprintf("%-10s %s", "Ctrl+Z/Y", "Undo / redo"),
		fmt.Sprintf("%-10s %s", "Ctrl+C", "Quit (fixed)"),
		"", "Start writing: Esc > File > New Project...")
}

func (m Model) helpText() string { return strings.Join(m.helpLines(), "\n") }

func (m Model) helpBox() string {
	return boxStyle.Render(strings.Join(append(m.helpLines(), "", "Esc or Q closes"), "\n"))
}

func (m Model) aboutBox() string {
	return boxStyle.Render(strings.Join([]string{
		"Wordterm", "Version " + Version, "",
		"Copyright © 2026 Anna Vahtera.", "License: GNU GPL v3", "", "Esc or Q closes"}, "\n"))
}

type rebindState struct {
	idx       int
	capturing bool
	msg       string
}

func (m Model) rebindBox() string {
	r := m.rebind
	lines := []string{"Rebind Keys", ""}
	for i, a := range settings.Actions {
		key := m.cfg.Keys[a.ID]
		if r.capturing && i == r.idx {
			key = "press a key..."
		}
		row := ui.Pad(fmt.Sprintf(" %-16s %s", a.Label, key), 38)
		if i == r.idx {
			row = invert.Render(row)
		}
		lines = append(lines, row)
	}
	lines = append(lines, "", "Esc (main menu) cannot be rebound.", r.msg, "Enter: change  D: default  Esc: back")
	return boxStyle.Render(strings.Join(lines, "\n"))
}

func (m *Model) rebindKey(k string) {
	r := m.rebind
	a := settings.Actions[r.idx]
	if r.capturing {
		r.capturing = false
		if err := m.cfg.SetKey(a.ID, k); err != nil {
			r.msg = err.Error()
			return
		}
		r.msg = a.Label + " is now " + k
		m.saveSettings()
		m.refreshWelcome()
		return
	}
	n := len(settings.Actions)
	switch k {
	case "up":
		r.idx = (r.idx + n - 1) % n
	case "down":
		r.idx = (r.idx + 1) % n
	case "enter":
		r.capturing, r.msg = true, "Press the new key combination"
	case "d":
		if err := m.cfg.SetKey(a.ID, a.Default); err != nil {
			r.msg = err.Error()
		} else {
			r.msg = a.Label + " reset to " + a.Default
			m.saveSettings()
			m.refreshWelcome()
		}
	case "esc", "q":
		m.rebind = nil
	}
}

// writeW is the writing-column width: Write Size, never wider than the editor.
func (m Model) writeW(l ui.Layout) int {
	avail := max(l.EditW-2, 1) // one column margin on each side
	if w := m.cfg.WriteSize; w > 0 && w < avail {
		return max(w, min(10, avail))
	}
	return avail
}

func (m Model) adjustHint(l ui.Layout) string {
	name, val := "Editor Size", l.EditW
	if m.adjust == adjWrite {
		name, val = "Write Size", m.writeW(l)
	}
	return fmt.Sprintf("%s: %d columns | Left/Right ±1, Up/Down ±5 (Up = wider), Enter to finish", name, val)
}

func (m *Model) adjustKey(k string) {
	d := 0
	switch k {
	case "left":
		d = -1
	case "right":
		d = 1
	case "up":
		d = 5
	case "down":
		d = -5
	case "enter", "esc":
		m.adjust = adjNone
		m.saveSettings()
		return
	}
	l := m.layout()
	if m.adjust == adjEditor {
		m.cfg.EditorWidth = ui.Compute(m.width, m.height, l.EditW+d).EditW
	} else {
		avail := max(l.EditW-2, 1)
		n := min(max(m.writeW(l)+d, min(10, avail)), avail)
		if n == avail {
			n = 0 // default: same as the editor width
		}
		m.cfg.WriteSize = n
	}
	m.syncSize()
}
