package app

import (
	"strings"

	"github.com/Vahtera/wordterm/internal/themes"
	"github.com/Vahtera/wordterm/internal/ui"
)

// themeState is the Settings > Change Theme... chooser. Moving the selection
// previews the theme at once; Esc puts the previous theme back.
type themeState struct {
	idx  int
	orig string
}

func (m *Model) openThemeChooser() {
	idx := 0
	for i, e := range themes.List() {
		if e.ID == m.cfg.Theme {
			idx = i
		}
	}
	m.themeSel = &themeState{idx: idx, orig: m.cfg.Theme}
}

func (m *Model) previewTheme(id string) {
	m.cfg.Theme = id
	th, note := themes.Resolve(m.cfg)
	m.theme = th
	if note != "" {
		m.msg = note
	}
}

func (m *Model) themeKey(k string) {
	t, list := m.themeSel, themes.List()
	switch k {
	case "up":
		t.idx = (t.idx + len(list) - 1) % len(list)
		m.previewTheme(list[t.idx].ID)
	case "down":
		t.idx = (t.idx + 1) % len(list)
		m.previewTheme(list[t.idx].ID)
	case "enter":
		m.themeSel = nil
		m.saveSettings()
		if m.msg == "" {
			m.msg = "Theme: " + list[t.idx].Label
		}
	case "esc":
		m.previewTheme(t.orig)
		m.themeSel = nil
	}
}

func (m Model) themeBox() string {
	t := m.themeSel
	list := themes.List()
	lines := []string{"Change Theme", ""}
	for i, e := range list {
		mark := "  "
		if e.ID == t.orig {
			mark = "* "
		}
		row := ui.Pad(" "+mark+e.Label, 30)
		if i == t.idx {
			row = invert.Render(row)
		}
		lines = append(lines, row)
	}
	lines = append(lines, "", "Up/Down: preview   Enter: apply   Esc: cancel", "* = current theme")
	if list[t.idx].ID == "user" {
		if path, err := themes.UserPath(); err == nil {
			lines = append(lines, "Edit colors in: "+path)
		}
	}
	return m.theme.Box(strings.Join(lines, "\n"))
}
