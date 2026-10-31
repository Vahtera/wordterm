// Package app holds top-level application state and the Bubble Tea model.
package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"wordterm/internal/project"
	"wordterm/internal/storage"
	"wordterm/internal/ui"
)

type Focus int

const (
	FocusNavigator Focus = iota
	FocusEditor
)

// Model reads project data only from the working copy (sess.Root).
type Model struct {
	width, height int
	editorWidth   int // 0 = default; becomes a setting in Phase 5
	focus         Focus

	sess *storage.Session
	proj *project.Project
	cur  project.SceneRef
	text string
	// word counts: scene, chapter, project
	ws, wc, wp int
	msg        string
}

func New(sess *storage.Session, proj *project.Project) Model {
	m := Model{sess: sess, proj: proj}
	m.openLast()
	return m
}

// openLast opens the last opened scene, falling back to the first scene.
func (m *Model) openLast() {
	ref, ok := m.proj.Meta.Resolve()
	if !ok {
		m.cur, m.text = project.SceneRef{}, ""
		m.refreshCounts()
		return
	}
	m.openScene(ref)
}

// openScene records the scene as last opened (in memory only; it reaches
// disk on Save, so merely browsing never makes the project "changed").
func (m *Model) openScene(ref project.SceneRef) {
	txt, err := m.proj.ReadScene(ref)
	if err != nil {
		m.msg = "Error: " + err.Error()
		return
	}
	m.cur, m.text = ref, txt
	r := ref
	m.proj.Meta.LastOpened = &r
	m.refreshCounts()
}

func (m *Model) refreshCounts() {
	var err error
	m.ws, m.wc, m.wp, err = m.proj.Counts(m.cur)
	if err != nil {
		m.msg = "Error: " + err.Error()
	}
}

func (m *Model) save() {
	if err := m.proj.WriteMeta(); err != nil { // working copy
		m.msg = "Save failed: " + err.Error()
		return
	}
	if err := m.sess.Save(); err != nil { // working copy -> original
		m.msg = "Save failed (working copy kept): " + err.Error()
		return
	}
	m.msg = "Saved"
}

func (m *Model) reload() {
	if has, err := m.sess.HasChanges(); err != nil || has {
		m.msg = "Reload refused: unsaved changes (confirmation dialog arrives in Phase 5)"
		return
	}
	if err := m.sess.Reload(); err != nil {
		m.msg = "Reload failed: " + err.Error()
		return
	}
	p, err := project.Load(m.sess.Root)
	if err != nil {
		m.msg = "Reload failed: " + err.Error()
		return
	}
	last := m.proj.Meta.LastOpened
	m.proj = p
	if last != nil {
		m.proj.Meta.LastOpened = last
	}
	m.openLast()
	m.msg = "Reloaded"
}

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
		m.msg = ""
		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			return m, tea.Quit
		case "f3": // default pane switch; becomes rebindable in Phase 5
			if m.focus == FocusNavigator {
				m.focus = FocusEditor
			} else {
				m.focus = FocusNavigator
			}
		case "f10":
			m.save()
		case "f9":
			m.reload()
		}
	}
	return m, nil
}

func treeLines(meta project.Meta, cur project.SceneRef) []string {
	lines := []string{meta.Name}
	for ci, c := range meta.Chapters {
		branch, pad := "├─ ", "│  "
		if ci == len(meta.Chapters)-1 {
			branch, pad = "└─ ", "   "
		}
		lines = append(lines, branch+c.Name)
		for si, s := range c.Scenes {
			sb, mark := "├─ ", ""
			if si == len(c.Scenes)-1 {
				sb = "└─ "
			}
			if cur == (project.SceneRef{Chapter: c.Name, Scene: s}) {
				mark = " ◂"
			}
			lines = append(lines, pad+sb+s+mark)
		}
	}
	return lines
}

func (m Model) View() tea.View {
	var s string
	switch l := m.layout(); {
	case m.width == 0:
		s = ""
	case !l.OK:
		s = ui.TooSmall(m.width, m.height)
	default:
		nav := ui.PaneContent{Header: "Navigator", Lines: treeLines(m.proj.Meta, m.cur), Focused: m.focus == FocusNavigator}
		edit := ui.PaneContent{Header: "Editor (read-only preview)", Lines: strings.Split(m.text, "\n"), Focused: m.focus == FocusEditor}
		status := fmt.Sprintf("Words: Scene: %d | Chapter: %d | Project: %d", m.ws, m.wc, m.wp)
		if m.msg != "" {
			status += " | " + m.msg
		}
		s = ui.Frame(l, "Wordterm", nav, edit, status)
	}
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}
