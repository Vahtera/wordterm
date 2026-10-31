package app

import (
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/Vahtera/wordterm/internal/editor"
	"github.com/Vahtera/wordterm/internal/project"
	"github.com/Vahtera/wordterm/internal/settings"
	"github.com/Vahtera/wordterm/internal/storage"
)

// NewWelcome builds the read-only built-in project shown when Wordterm starts
// without a project: chapter "Help", scene "Keys", containing the same text as
// F1. It lives in a temporary folder (returned so the caller can delete it) and
// is never offered for crash recovery.
func NewWelcome(cfg settings.Settings, note string) (Model, string, error) {
	dir, err := os.MkdirTemp("", "wordterm-help-*")
	if err != nil {
		return Model{}, "", err
	}
	fail := func(err error) (Model, string, error) {
		os.RemoveAll(dir)
		return Model{}, "", err
	}
	ref := project.SceneRef{Chapter: "Help", Scene: "Keys"}
	p := &project.Project{Root: dir, MetaPath: filepath.Join(dir, "Wordterm.json"), Meta: project.Meta{
		Version: project.FormatVersion, Name: "Wordterm", LastOpened: &ref,
		Chapters: []project.Chapter{{Name: "Help", Scenes: []project.Scene{{Name: "Keys", File: "Keys"}}}},
	}}
	if err := os.Mkdir(filepath.Join(dir, "Help"), 0o755); err != nil {
		return fail(err)
	}
	if err := storage.WriteFileAtomic(project.SceneFile(dir, ref), []byte(Model{cfg: cfg}.helpText()), 0o644); err != nil {
		return fail(err)
	}
	if err := p.WriteMeta(); err != nil {
		return fail(err)
	}
	sess, err := storage.Open(dir)
	if err != nil {
		return fail(err)
	}
	sess.Ephemeral()
	proj, err := project.Load(sess.Root)
	if err != nil {
		sess.Close()
		return fail(err)
	}
	if note == "" {
		note = "Built-in help (read-only). Esc > File > New Project to start writing"
	}
	m := New(sess, proj, cfg, note)
	m.welcome = true
	m.refreshWelcome()
	return m, dir, nil
}

// refreshWelcome re-renders the help text (key bindings may have changed).
func (m *Model) refreshWelcome() {
	if m.welcome && m.hasScene() {
		m.ed = editor.New(m.helpText())
		m.syncSize()
	}
}

// readOnly blocks changes to the built-in project and offers a new project instead.
func (m *Model) readOnly() bool {
	if m.welcome {
		m.offerNewProject()
	}
	return m.welcome
}

func (m *Model) offerNewProject() {
	m.ask("This help project is read-only. Create a new project instead? (y/n)", "yn", func(m *Model, a string) tea.Cmd {
		if a == "y" {
			return m.newProject()
		}
		return nil
	})
}

// isEditKey reports whether a key press would change the text (as opposed to moving the cursor).
func isEditKey(msg tea.KeyPressMsg, k string) bool {
	switch k {
	case "left", "right", "up", "down", "home", "end", "pgup", "pgdown":
		return false
	case "enter", "backspace", "delete", "ctrl+z", "ctrl+y":
		return true
	}
	t := msg.Key().Text
	return t != "" && (k == t || t == " ")
}
