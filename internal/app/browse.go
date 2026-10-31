package app

import (
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"wordterm/internal/project"
	"wordterm/internal/settings"
	"wordterm/internal/ui"
)

// browseState is the folder browser behind File > Open Project...
// Enter on a folder that contains a project .json opens that project;
// Enter on any other folder goes into it.
type browseState struct {
	dir     string
	entries []browseEntry
	idx     int
	top     int
	msg     string
}

type browseEntry struct {
	name, path string
	project    bool
}

func isDir(path string, e os.DirEntry) bool {
	if e.IsDir() {
		return true
	}
	if e.Type()&os.ModeSymlink != 0 {
		st, err := os.Stat(path)
		return err == nil && st.IsDir()
	}
	return false
}

// cd lists dir (without hidden folders) and selects the entry called selectName, if any.
func (b *browseState) cd(dir, selectName string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	des, err := os.ReadDir(abs)
	if err != nil {
		return err
	}
	var entries []browseEntry
	if parent := filepath.Dir(abs); parent != abs {
		entries = append(entries, browseEntry{name: "..", path: parent})
	}
	for _, e := range des {
		p := filepath.Join(abs, e.Name())
		if strings.HasPrefix(e.Name(), ".") || !isDir(p, e) {
			continue
		}
		entries = append(entries, browseEntry{name: e.Name(), path: p, project: project.IsProjectDir(p)})
	}
	b.dir, b.entries, b.idx, b.top, b.msg = abs, entries, 0, 0, ""
	for i, e := range entries {
		if e.name == selectName {
			b.idx = i
		}
	}
	return nil
}

// browseStart is the Default Save Path if it exists, otherwise the home folder.
func (m Model) browseStart() string {
	if p := settings.ExpandPath(m.cfg.DefaultSavePath); p != "" {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return "."
}

func (m *Model) openProject() tea.Cmd {
	return m.confirmLeave(func(m *Model) tea.Cmd {
		b := &browseState{}
		if err := b.cd(m.browseStart(), ""); err != nil {
			return m.fail(err)
		}
		m.browse = b
		return nil
	})
}

func (m *Model) browseKey(k string) tea.Cmd {
	b := m.browse
	n := len(b.entries)
	switch k {
	case "esc":
		m.browse = nil
	case "up":
		b.idx = max(b.idx-1, 0)
	case "down":
		b.idx = max(min(b.idx+1, n-1), 0)
	case "pgup":
		b.idx = max(b.idx-10, 0)
	case "pgdown":
		b.idx = max(min(b.idx+10, n-1), 0)
	case "home":
		b.idx = 0
	case "end":
		b.idx = max(n-1, 0)
	case "left", "backspace":
		if parent := filepath.Dir(b.dir); parent != b.dir {
			if err := b.cd(parent, filepath.Base(b.dir)); err != nil {
				b.msg = err.Error()
			}
		}
	case "enter", "right":
		if n == 0 {
			return nil
		}
		e := b.entries[b.idx]
		if e.project && k == "enter" {
			m.browse = nil
			return m.switchProject(e.path)
		}
		sel := ""
		if e.name == ".." {
			sel = filepath.Base(b.dir)
		}
		if err := b.cd(e.path, sel); err != nil {
			b.msg = err.Error()
		}
	}
	return nil
}

func (m Model) browseBox() string {
	b := m.browse
	rows := max(min(m.height-8, 15), 3)
	if b.idx < b.top {
		b.top = b.idx
	}
	if b.idx >= b.top+rows {
		b.top = b.idx - rows + 1
	}
	w := min(max(m.width-8, 24), 64)
	path := []rune(b.dir)
	if len(path) > w {
		path = append([]rune("…"), path[len(path)-w+1:]...)
	}
	lines := []string{"Open Project", string(path), ""}
	for i := b.top; i < len(b.entries) && i < b.top+rows; i++ {
		e := b.entries[i]
		label := e.name + "/"
		if e.project {
			label += "  [project]"
		}
		row := ui.Pad(" "+label, w)
		if i == b.idx {
			row = invert.Render(row)
		}
		lines = append(lines, row)
	}
	if len(b.entries) == 0 {
		lines = append(lines, " (no folders here)")
	}
	lines = append(lines, "", b.msg, "Enter: open/enter  Left: up  Esc: cancel")
	return boxStyle.Render(strings.Join(lines, "\n"))
}
