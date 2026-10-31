// Package app holds top-level application state and the Bubble Tea model.
package app

import (
	tea "charm.land/bubbletea/v2"

	"wordterm/internal/ui"
)

type Focus int

const (
	FocusNavigator Focus = iota
	FocusEditor
)

type Model struct {
	width, height int
	editorWidth   int // 0 = default; becomes a setting in Phase 5
	focus         Focus
}

func New() Model { return Model{} }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) layout() ui.Layout { return ui.Compute(m.width, m.height, m.editorWidth) }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		if !m.layout().OK {
			return m, tea.Quit // any key quits from the too-small screen
		}
		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			return m, tea.Quit
		case "f3": // default pane switch; becomes rebindable in Phase 5
			if m.focus == FocusNavigator {
				m.focus = FocusEditor
			} else {
				m.focus = FocusNavigator
			}
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	var s string
	switch l := m.layout(); {
	case m.width == 0:
		s = ""
	case !l.OK:
		s = ui.TooSmall(m.width, m.height)
	default:
		nav := ui.PaneContent{Header: "Navigator", Lines: []string{" (no project open)"}, Focused: m.focus == FocusNavigator}
		edit := ui.PaneContent{Header: "Editor", Focused: m.focus == FocusEditor}
		status := "Words: Scene: 0 | Chapter: 0 | Project: 0"
		s = ui.Frame(l, "Wordterm", nav, edit, status)
	}
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}
