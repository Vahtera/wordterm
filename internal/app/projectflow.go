package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"wordterm/internal/editor"
	"wordterm/internal/navigator"
	"wordterm/internal/project"
	"wordterm/internal/storage"
)

func (m *Model) askSave() {
	m.ask("Save project? (y/n)", "yn", func(m *Model, a string) tea.Cmd {
		if a == "y" {
			m.save()
		}
		return nil
	})
}

func (m *Model) askTextOptional(label string, done func(*Model, string) tea.Cmd) {
	m.prompt = &prompt{label: label, text: true, optional: true, done: done}
}

func (m *Model) autosave() {
	if !m.cfg.Autosave || m.prompt != nil {
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
	if m.save(); m.msg == "Saved" {
		m.msg = "Autosaved"
	}
}

func expandPath(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[1:])
		}
	}
	return p
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

func (m *Model) newProject() tea.Cmd {
	return m.confirmLeave(func(m *Model) tea.Cmd {
		m.askText("New project name:", "", func(m *Model, name string) tea.Cmd {
			if err := project.ValidateName(name); err != nil {
				return m.fail(err)
			}
			m.askText("Author name:", "", func(m *Model, author string) tea.Cmd {
				m.askTextOptional("Author info (optional, Enter to skip):", func(m *Model, info string) tea.Cmd {
					m.askText("Destination directory:", filepath.Dir(m.sess.Original), func(m *Model, dest string) tea.Cmd {
						dir, err := project.Create(expandPath(dest), name, author, info)
						if err != nil {
							return m.fail(err)
						}
						return m.switchProject(dir)
					})
					return nil
				})
				return nil
			})
			return nil
		})
		return nil
	})
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
	if m.flush() != nil {
		return nil
	}
	def := filepath.Join(filepath.Dir(m.sess.Original), m.proj.Meta.Name+"_copy")
	m.askText("Save as (new directory):", def, func(m *Model, dest string) tea.Cmd {
		abs, err := filepath.Abs(expandPath(dest))
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
