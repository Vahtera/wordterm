package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"

	"wordterm/internal/editor"
	"wordterm/internal/navigator"
	"wordterm/internal/project"
	"wordterm/internal/settings"
	"wordterm/internal/storage"
)

func (m *Model) askSave() {
	if m.readOnly() {
		return
	}
	m.ask("Save project? (y/n)", "yn", func(m *Model, a string) tea.Cmd {
		if a == "y" {
			m.save()
		}
		return nil
	})
}

func (m *Model) askTextOptional(label, initial string, done func(*Model, string) tea.Cmd) {
	m.prompt = &prompt{label: label, text: true, optional: true, input: []rune(initial), done: done}
}

func (m *Model) autosave() {
	if !m.cfg.Autosave || m.prompt != nil || m.welcome || m.autoPaused {
		return
	}
	iv := time.Duration(max(m.cfg.AutosaveInterval, 5)) * time.Second
	if time.Since(m.lastAuto) < iv {
		return
	}
	m.lastAuto = time.Now()
	if m.flush() != nil {
		return
	}
	if has, err := m.sess.HasChanges(); err != nil || !has {
		return
	}
	if m.saveWith(false); m.msg == "Saved" {
		m.msg = "Autosaved"
	}
}

// confirmLeave runs next once it is safe to leave the current project:
// immediately if nothing is unsaved, otherwise after Save or Discard.
func (m *Model) confirmLeave(next func(m *Model) tea.Cmd) tea.Cmd {
	if m.flush() != nil {
		return nil
	}
	if has, err := m.sess.HasChanges(); err == nil && !has {
		return next(m)
	}
	m.ask("Unsaved changes: [S]ave, [D]iscard, [C]ancel", "sdc", func(m *Model, a string) tea.Cmd {
		switch a {
		case "s":
			m.save()
			if m.msg != "Saved" {
				return nil
			}
			return next(m)
		case "d":
			return next(m)
		}
		return nil
	})
	return nil
}

// newProject asks only for a name; the folder and author come from Settings
// (and are asked for here only if they are not set).
func (m *Model) newProject() tea.Cmd {
	return m.confirmLeave(func(m *Model) tea.Cmd {
		m.askText("New project name:", "", func(m *Model, name string) tea.Cmd {
			if err := project.ValidateName(name); err != nil {
				return m.fail(err)
			}
			return m.createWithDefaults(name)
		})
		return nil
	})
}

func (m *Model) createWithDefaults(name string) tea.Cmd {
	author, parent := m.cfg.AuthorName, settings.ExpandPath(m.cfg.DefaultSavePath)
	create := func(m *Model) tea.Cmd {
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return m.fail(err)
		}
		dir, err := project.Create(parent, name, author, "")
		if err != nil {
			return m.fail(err)
		}
		return m.switchProject(dir)
	}
	askParent := func(m *Model) tea.Cmd {
		if parent != "" {
			return create(m)
		}
		home, _ := os.UserHomeDir()
		m.askText("Destination directory:", home, func(m *Model, d string) tea.Cmd {
			parent = settings.ExpandPath(d)
			return create(m)
		})
		return nil
	}
	if author == "" {
		m.askTextOptional("Author name (Enter to skip):", "", func(m *Model, a string) tea.Cmd {
			author = a
			return askParent(m)
		})
		return nil
	}
	return askParent(m)
}

func (m *Model) askSavePath() tea.Cmd {
	m.askTextOptional("Default save path (empty clears):", m.cfg.DefaultSavePath, func(m *Model, p string) tea.Cmd {
		if p != "" {
			abs, err := filepath.Abs(settings.ExpandPath(p))
			if err != nil {
				return m.fail(err)
			}
			if st, err := os.Stat(abs); err == nil && !st.IsDir() {
				return m.fail(fmt.Errorf("%s is not a directory", abs))
			}
			p = abs
		}
		m.cfg.DefaultSavePath = p
		m.saveSettings()
		if m.msg == "" {
			m.msg = "Default save path: " + p
		}
		return nil
	})
	return nil
}

func (m *Model) askAuthor() tea.Cmd {
	m.askTextOptional("Author name (empty clears):", m.cfg.AuthorName, func(m *Model, a string) tea.Cmd {
		m.cfg.AuthorName = a
		m.saveSettings()
		if m.msg == "" {
			m.msg = "Author name: " + a
		}
		return nil
	})
	return nil
}

// switchProject opens the project in dir in a fresh working copy and closes the old one.
func (m *Model) switchProject(dir string) tea.Cmd {
	if _, err := project.Load(dir); err != nil {
		return m.fail(err)
	}
	ns, err := storage.Open(dir)
	if err != nil {
		return m.fail(err)
	}
	np, err := project.Load(ns.Root)
	if err != nil {
		ns.Close()
		return m.fail(err)
	}
	old := m.sess
	m.welcome = false
	m.sess, m.proj = ns, np
	m.cur, m.ed, m.nav = project.SceneRef{}, editor.New(""), navigator.New()
	m.openLast()
	old.Close()
	m.msg = "Opened " + np.Meta.Name
	return nil
}

// saveAs writes a complete copy of the project to a new directory and
// continues the session there. The original project is not modified.
func (m *Model) saveAs() tea.Cmd {
	if m.readOnly() {
		return nil
	}
	if m.flush() != nil {
		return nil
	}
	def := filepath.Join(filepath.Dir(m.sess.Original), m.proj.Meta.Name+"_copy")
	m.askText("Save as (new directory):", def, func(m *Model, dest string) tea.Cmd {
		abs, err := filepath.Abs(settings.ExpandPath(dest))
		if err != nil {
			return m.fail(err)
		}
		entries, err := os.ReadDir(abs)
		switch {
		case err == nil && len(entries) > 0:
			return m.fail(fmt.Errorf("%s already exists and is not empty", abs))
		case err != nil && !os.IsNotExist(err):
			return m.fail(err)
		}
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return m.fail(err)
		}
		if err := m.proj.WriteMeta(); err != nil {
			return m.fail(err)
		}
		if err := storage.SyncTree(m.sess.Root, abs); err != nil {
			return m.fail(err)
		}
		if err := m.sess.SetOriginal(abs); err != nil {
			return m.fail(err)
		}
		m.msg = "Saved as " + abs
		return nil
	})
	return nil
}
