// Package app holds top-level application state and the Bubble Tea model.
package app

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"wordterm/internal/editor"
	"wordterm/internal/navigator"
	"wordterm/internal/project"
	"wordterm/internal/settings"
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
	focus         Focus

	sess *storage.Session
	proj *project.Project
	cur  project.SceneRef
	ed   *editor.Editor
	// word counts of the working copy on disk: scene, chapter, project
	ws, wc, wp int
	msg        string
	quitArmed  bool
	discard    bool // user chose to discard unsaved changes on quit

	nav    *navigator.Navigator
	prompt *prompt

	cfg      settings.Settings
	lastAuto time.Time
	menu     *menuState
	overlay  overlayKind
	adjust   adjustKind
	rebind   *rebindState
}

type overlayKind int

const (
	ovNone overlayKind = iota
	ovHelp
	ovAbout
)

type adjustKind int

const (
	adjNone adjustKind = iota
	adjEditor
	adjWrite
)

// prompt is a one-line question shown in the status bar: either free text
// (text=true, done gets the entered text) or a single key from keys.
type prompt struct {
	label    string
	text     bool
	optional bool // text prompt that accepts an empty answer
	keys     string
	input []rune
	done  func(m *Model, answer string) tea.Cmd
}

// Discarded reports whether the user chose to discard unsaved changes when quitting.
func (m Model) Discarded() bool { return m.discard }

// Session returns the current working-copy session (it changes on New Project).
func (m Model) Session() *storage.Session { return m.sess }

func New(sess *storage.Session, proj *project.Project, cfg settings.Settings, note string) Model {
	m := Model{sess: sess, proj: proj, ed: editor.New(""), nav: navigator.New(), cfg: cfg, msg: note, lastAuto: time.Now()}
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
	if m.flush() != nil { // never drop edits of the scene we are leaving
		return
	}
	txt, err := m.proj.ReadScene(ref)
	if err != nil {
		m.msg = "Error: " + err.Error()
		return
	}
	m.cur, m.ed = ref, editor.New(txt)
	r := ref
	m.proj.Meta.LastOpened = &r
	m.nav.Expand(ref.Chapter)
	m.nav.Select(navigator.Item{Kind: navigator.Scene, Chapter: ref.Chapter, Scene: ref.Scene})
	m.syncSize()
	m.refreshCounts()
}

func (m *Model) syncSize() {
	if l := m.layout(); l.OK {
		m.ed.SetWrap(m.cfg.WordWrap)
		m.ed.SetSize(m.writeW(l), l.PaneH-1) // 1 header row
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

func (m *Model) requestReload() {
	if m.flush() != nil {
		return
	}
	if has, err := m.sess.HasChanges(); err != nil || has {
		m.ask("Discard unsaved changes and reload? (y/n)", "yn", func(m *Model, a string) tea.Cmd {
			if a == "y" {
				m.reloadNow()
			}
			return nil
		})
		return
	}
	m.reloadNow()
}

func (m *Model) reloadNow() {
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
	m.cur = project.SceneRef{}
	m.ed = editor.New("") // discard in-memory edits
	m.openLast()
	m.msg = "Reloaded"
}

func (m *Model) ask(label, keys string, done func(*Model, string) tea.Cmd) {
	m.prompt = &prompt{label: label, keys: keys, done: done}
}

func (m *Model) askText(label, initial string, done func(*Model, string) tea.Cmd) {
	m.prompt = &prompt{label: label, text: true, input: []rune(initial), done: done}
}

// requestQuit quits at once when nothing is unsaved, otherwise asks.
func (m *Model) requestQuit() tea.Cmd {
	m.prompt, m.menu, m.overlay, m.adjust, m.rebind = nil, nil, ovNone, adjNone, nil
	if m.flush() != nil && !m.quitArmed {
		m.quitArmed = true
		m.msg = "Cannot write working copy; quit again to exit anyway"
		return nil
	}
	if has, err := m.sess.HasChanges(); err != nil || !has {
		return tea.Quit
	}
	m.ask("Unsaved changes: [S]ave, [D]iscard, [C]ancel", "sdc", func(m *Model, a string) tea.Cmd {
		switch a {
		case "s":
			if m.save(); m.msg == "Saved" {
				return tea.Quit
			}
		case "d":
			m.discard = true
			return tea.Quit
		}
		return nil
	})
	return nil
}

func (m *Model) promptKey(msg tea.KeyPressMsg) tea.Cmd {
	p, k := m.prompt, msg.String()
	if k == "esc" {
		m.prompt = nil
		return nil
	}
	if !p.text {
		if a := strings.ToLower(k); len(a) == 1 && strings.Contains(p.keys, a) {
			m.prompt = nil
			return p.done(m, a)
		}
		return nil
	}
	switch k {
	case "enter":
		text := strings.TrimSpace(string(p.input))
		m.prompt = nil
		if text == "" && !p.optional {
			return nil
		}
		return p.done(m, text)
	case "backspace":
		if len(p.input) > 0 {
			p.input = p.input[:len(p.input)-1]
		}
	default:
		if t := msg.Key().Text; t != "" && (k == t || t == " ") {
			p.input = append(p.input, []rune(t)...)
		}
	}
	return nil
}

func (m Model) Init() tea.Cmd { return tick() }

func (m Model) layout() ui.Layout { return ui.Compute(m.width, m.height, m.cfg.EditorWidth) }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.syncSize()
	case flushMsg:
		m.flush()
		m.autosave()
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
		k := keyName(msg)
		if k == "ctrl+c" {
			return m, m.requestQuit()
		}
		switch {
		case m.rebind != nil:
			m.rebindKey(k)
		case m.overlay != ovNone:
			if k == "esc" || k == "q" {
				m.overlay = ovNone
			}
		case m.adjust != adjNone:
			m.adjustKey(k)
		case m.menu != nil:
			return m, m.menuKey(k)
		case m.prompt != nil:
			return m, m.promptKey(msg)
		case k == "esc": // fixed: always opens the main menu
			m.menu = &menuState{}
		default:
			return m, m.dispatch(msg, k)
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

func (m *Model) navKey(msg tea.KeyPressMsg) {
	k, meta := msg.String(), &m.proj.Meta
	switch {
	case k == "up":
		m.nav.Up(meta)
	case k == "down":
		m.nav.Down(meta)
	case k == "left":
		m.nav.Left(meta)
	case k == "right":
		m.nav.Right(meta)
	case k == "enter":
		it := m.nav.Selected(meta)
		switch it.Kind {
		case navigator.Scene:
			if ref := (project.SceneRef{Chapter: it.Chapter, Scene: it.Scene}); ref != m.cur {
				m.openScene(ref)
			}
		case navigator.Chapter:
			m.nav.Toggle(it.Chapter)
		}
	}
}

func (m *Model) fail(err error) tea.Cmd {
	m.msg = "Error: " + err.Error()
	return nil
}

func (m *Model) addItem() {
	meta := &m.proj.Meta
	it := m.nav.Selected(meta)
	if it.Kind == navigator.Root {
		m.askText("New chapter name:", "", func(m *Model, name string) tea.Cmd {
			if err := m.proj.AddChapter(name); err != nil {
				return m.fail(err)
			}
			m.nav.Select(navigator.Item{Kind: navigator.Chapter, Chapter: name})
			return nil
		})
		return
	}
	ch, at := it.Chapter, -1
	if it.Kind == navigator.Scene {
		at = meta.SceneIndex(ch, it.Scene) + 1
	}
	m.askText("New scene name:", "", func(m *Model, name string) tea.Cmd {
		ref, err := m.proj.AddScene(ch, at, name)
		if err != nil {
			return m.fail(err)
		}
		m.nav.Expand(ch)
		m.nav.Select(navigator.Item{Kind: navigator.Scene, Chapter: ch, Scene: ref.Scene})
		if !m.hasScene() {
			m.openScene(ref)
		}
		m.refreshCounts()
		return nil
	})
}

func (m *Model) deleteItem() {
	meta := &m.proj.Meta
	it := m.nav.Selected(meta)
	switch it.Kind {
	case navigator.Root:
		m.msg = "The project cannot be deleted"
	case navigator.Chapter:
		if n := len(meta.Chapters[meta.ChapterIndex(it.Chapter)].Scenes); n > 0 {
			m.msg = "Chapter is not empty: delete its scenes first"
			return
		}
		m.ask(fmt.Sprintf("Delete chapter '%s'? (y/n)", it.Chapter), "yn", func(m *Model, a string) tea.Cmd {
			if a == "y" {
				if err := m.proj.DeleteChapter(it.Chapter); err != nil {
					return m.fail(err)
				}
			}
			return nil
		})
	case navigator.Scene:
		ref := project.SceneRef{Chapter: it.Chapter, Scene: it.Scene}
		m.ask(fmt.Sprintf("Delete scene '%s'? (y/n)", meta.SceneName(it.Chapter, it.Scene)), "yn", func(m *Model, a string) tea.Cmd {
			if a != "y" {
				return nil
			}
			if err := m.proj.DeleteScene(ref.Chapter, ref.Scene); err != nil {
				return m.fail(err)
			}
			if m.cur == ref {
				m.cur = project.SceneRef{}
				m.ed = editor.New("")
				m.openLast()
			}
			m.refreshCounts()
			return nil
		})
	}
}

func (m *Model) renameItem() {
	meta := &m.proj.Meta
	it := m.nav.Selected(meta)
	switch it.Kind {
	case navigator.Root:
		m.askText("Project name:", meta.Name, func(m *Model, name string) tea.Cmd {
			if err := m.proj.RenameProject(name); err != nil {
				return m.fail(err)
			}
			return nil
		})
	case navigator.Chapter:
		m.askText("Rename chapter:", it.Chapter, func(m *Model, name string) tea.Cmd {
			if m.flush() != nil {
				return nil
			}
			if err := m.proj.RenameChapter(it.Chapter, name); err != nil {
				return m.fail(err)
			}
			m.nav.Rename(it.Chapter, name)
			if m.cur.Chapter == it.Chapter {
				m.cur.Chapter = name
			}
			return nil
		})
	case navigator.Scene:
		m.askText("Rename scene:", meta.SceneName(it.Chapter, it.Scene), func(m *Model, name string) tea.Cmd {
			if m.flush() != nil {
				return nil
			}
			ref, err := m.proj.RenameScene(it.Chapter, it.Scene, name)
			if err != nil {
				return m.fail(err)
			}
			if m.cur.Chapter == it.Chapter && m.cur.Scene == it.Scene {
				m.cur = ref
			}
			m.nav.Select(navigator.Item{Kind: navigator.Scene, Chapter: ref.Chapter, Scene: ref.Scene})
			return nil
		})
	}
}

// moveItem implements F5 (d=-1) and F6 (d=+1).
func (m *Model) moveItem(d int) {
	meta := &m.proj.Meta
	it := m.nav.Selected(meta)
	if it.Kind == navigator.Root || m.flush() != nil {
		return
	}
	ci := meta.ChapterIndex(it.Chapter)
	if it.Kind == navigator.Chapter {
		if ni, err := m.proj.MoveChapter(ci, d); err != nil {
			m.fail(err)
		} else if ni == ci {
			m.msg = "Cannot move further"
		}
		return
	}
	si := meta.SceneIndex(it.Chapter, it.Scene)
	nci, nsi, err := m.proj.MoveScene(ci, si, d)
	if err != nil {
		m.fail(err)
		return
	}
	if nci == ci && nsi == si {
		m.msg = "Cannot move further"
		return
	}
	ref := project.SceneRef{Chapter: meta.Chapters[nci].Name, Scene: meta.Chapters[nci].Scenes[nsi].File}
	if m.cur.Chapter == it.Chapter && m.cur.Scene == it.Scene {
		m.cur = ref
	}
	m.nav.Expand(ref.Chapter)
	m.nav.Select(navigator.Item{Kind: navigator.Scene, Chapter: ref.Chapter, Scene: ref.Scene})
	m.refreshCounts()
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
		nav := ui.PaneContent{Header: "Navigator", Focused: m.focus == FocusNavigator,
			Lines: m.nav.Lines(&m.proj.Meta, m.cur, l.NavW, l.PaneH-1)}
		edit := ui.PaneContent{Header: "Editor", Lines: rows, Focused: m.focus == FocusEditor}
		status := fmt.Sprintf("Words: Scene: %d | Chapter: %d | Project: %d | Autosave: %s",
			live, m.wc-m.ws+live, m.wp-m.ws+live, onOff(m.cfg.Autosave))
		switch {
		case m.adjust != adjNone:
			status = m.adjustHint(l)
		case m.prompt != nil && m.prompt.text:
			status = m.prompt.label + " " + string(m.prompt.input) + "█  (Enter accept, Esc cancel)"
		case m.prompt != nil:
			status = m.prompt.label
		case m.msg != "":
			status += " | " + m.msg
		}
		s = ui.Frame(l, "Wordterm", nav, edit, status)
		center := func(box string) {
			x, y := max((m.width-lipgloss.Width(box))/2, 0), max((m.height-lipgloss.Height(box))/2, 0)
			s = ui.Overlay(s, box, x, y)
		}
		switch {
		case m.rebind != nil:
			center(m.rebindBox())
		case m.overlay == ovHelp:
			center(m.helpBox())
		case m.overlay == ovAbout:
			center(m.aboutBox())
		case m.menu != nil:
			s = ui.Overlay(s, m.menuBox(), 0, 0)
		}
	}
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}
