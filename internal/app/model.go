// Package app holds top-level application state and the Bubble Tea model.
package app

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"wordterm/internal/editor"
	"wordterm/internal/project"
	"wordterm/internal/storage"
	"wordterm/internal/ui"
)

type Focus int

const (
	FocusNavigator Focus = iota
	FocusEditor
)

type flushMsg struct{}

func tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return flushMsg{} })
}

// Model reads and writes project data only in the working copy (sess.Root).
type Model struct {
	width, height int
	editorWidth   int // 0 = default; becomes a setting in Phase 5
	focus         Focus

	sess *storage.Session
	proj *project.Project
	cur  project.SceneRef
	ed   *editor.Editor
	// word counts of the working copy on disk: scene, chapter, project
	ws, wc, wp int
	msg        string
	quitArmed  bool
}

func New(sess *storage.Session, proj *project.Project) Model {
	m := Model{sess: sess, proj: proj, ed: editor.New("")}
	m.openLast()
	return m
}

func (m *Model) hasScene() bool { return m.cur != (project.SceneRef{}) }

// openLast opens the last opened scene, falling back to the first scene.
func (m *Model) openLast() {
	ref, ok := m.proj.Meta.Resolve()
	if !ok {
		m.cur, m.ed = project.SceneRef{}, editor.New("")
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
	m.cur, m.ed = ref, editor.New(txt)
	r := ref
	m.proj.Meta.LastOpened = &r
	m.syncSize()
	m.refreshCounts()
}

func (m *Model) syncSize() {
	if l := m.layout(); l.OK {
		m.ed.SetSize(l.EditW-2, l.PaneH-1) // 1 column margin each side, 1 header row
	}
}

func (m *Model) refreshCounts() {
	var err error
	m.ws, m.wc, m.wp, err = m.proj.Counts(m.cur)
	if err != nil {
		m.msg = "Error: " + err.Error()
	}
}

// flush writes the open scene from the editor into the working copy.
func (m *Model) flush() error {
	if !m.hasScene() || !m.ed.Dirty() {
		return nil
	}
	err := storage.WriteFileAtomic(project.SceneFile(m.proj.Root, m.cur), []byte(m.ed.Text()), 0o644)
	if err != nil {
		m.msg = "Cannot write working copy: " + err.Error()
		return err
	}
	m.ed.ClearDirty()
	m.refreshCounts()
	return nil
}

func (m *Model) save() {
	if m.flush() != nil {
		return
	}
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
	if m.flush() != nil {
		return
	}
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

func (m Model) Init() tea.Cmd { return tick() }

func (m Model) layout() ui.Layout { return ui.Compute(m.width, m.height, m.editorWidth) }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.syncSize()
	case flushMsg:
		m.flush()
		return m, tick()
	case tea.PasteMsg:
		if m.focus == FocusEditor && m.hasScene() {
			m.ed.Insert(fmt.Sprint(msg))
		}
	case tea.KeyPressMsg:
		if !m.layout().OK {
			m.flush()
			return m, tea.Quit // any key quits from the too-small screen
		}
		m.msg = ""
		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			if m.flush() != nil && !m.quitArmed {
				m.quitArmed = true
				m.msg = "Cannot write working copy; quit again to exit anyway"
				return m, nil
			}
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
		default:
			if m.focus == FocusEditor && m.hasScene() {
				m.editorKey(msg)
			}
		}
	}
	return m, nil
}

func (m *Model) editorKey(msg tea.KeyPressMsg) {
	ed, k := m.ed, msg.String()
	switch k {
	case "left":
		ed.Left()
	case "right":
		ed.Right()
	case "up":
		ed.Up()
	case "down":
		ed.Down()
	case "home":
		ed.Home()
	case "end":
		ed.End()
	case "pgup":
		ed.PageUp()
	case "pgdown":
		ed.PageDown()
	case "enter":
		ed.Insert("\n")
	case "backspace":
		ed.Backspace()
	case "delete":
		ed.Delete()
	case "ctrl+z":
		ed.Undo()
	case "ctrl+y":
		ed.Redo()
	default:
		// Printable input: Text is set and matches the key string (alt/ctrl combos don't).
		if t := msg.Key().Text; t != "" && (k == t || t == " ") {
			ed.Insert(t)
		}
	}
}

func (m Model) treeLines() []string {
	meta := m.proj.Meta
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
			if m.cur == (project.SceneRef{Chapter: c.Name, Scene: s}) {
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
		var rows []string
		live := 0
		if m.hasScene() {
			for _, r := range m.ed.Lines(m.focus == FocusEditor) {
				rows = append(rows, " "+r)
			}
			live = m.ed.WordCount()
		}
		nav := ui.PaneContent{Header: "Navigator", Lines: m.treeLines(), Focused: m.focus == FocusNavigator}
		edit := ui.PaneContent{Header: "Editor", Lines: rows, Focused: m.focus == FocusEditor}
		status := fmt.Sprintf("Words: Scene: %d | Chapter: %d | Project: %d",
			live, m.wc-m.ws+live, m.wp-m.ws+live)
		if m.msg != "" {
			status += " | " + m.msg
		}
		s = ui.Frame(l, "Wordterm", nav, edit, status)
	}
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}
